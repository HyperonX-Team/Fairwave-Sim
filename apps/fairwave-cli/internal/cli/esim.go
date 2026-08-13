package cli

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/activation"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/registry"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/smdp"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/hsswrite"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
	"github.com/spf13/cobra"
)

// esimCmd is the eSIM (SM-DP+) command group: mint activation codes, run
// the lab SM-DP+ server, and manage the registry.
func esimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "esim",
		Short: "eSIM (SM-DP+) lifecycle: issue activation codes, run the lab server",
	}
	cmd.AddCommand(esimIssueCmd(), esimListCmd(), esimRevokeCmd(), esimServeCmd())
	return cmd
}

// esimListCmd lists the activation codes registered with the control plane.
func esimListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List issued activation codes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var codes []api.EsimCode
			if err := c.get("/v1/esim/codes", &codes); err != nil {
				return err
			}
			if len(codes) == 0 {
				fmt.Println("no activation codes issued")
				return nil
			}
			for _, e := range codes {
				fmt.Printf("%-14s imsi=%-15s iccid=%-20s created=%s downloaded=%v\n",
					e.ActivationCode, e.IMSI, e.ICCID, e.CreatedAt.Format(time.RFC3339), e.DownloadedAt != nil)
			}
			return nil
		},
	}
}

// esimRevokeCmd removes an activation code from the registry.
func esimRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke",
		Short: "Revoke an activation code (no longer scannable)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: fairwave esim revoke <activation-code>")
			}
			c := newClient(cmd)
			var out map[string]string
			if err := c.post("/v1/esim/revoke", api.EsimRevokeRequest{ActivationCode: args[0]}, &out); err != nil {
				return err
			}
			fmt.Printf("activation code %s %s\n", args[0], out["status"])
			return nil
		},
	}
}

// esimIssueCmd mints an eSIM profile + activation code for an IMSI. When
// the control plane is reachable it issues through the embedded SM-DP+ (so
// the code can be scanned against the running server); otherwise it falls
// back to the standalone local registry + `fairwave esim serve`.
func esimIssueCmd() *cobra.Command {
	var (
		imsi         string
		address      string
		eid          string
		qrOut        string
		hssDriver    string
		hssContainer string
	)
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Mint an eSIM profile + activation code (lab vectors only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var hz map[string]bool
			if err := c.get("/v1/healthz", &hz); err == nil {
				// control-plane path: the server persists the registry and
				// serves /es9plus, so scanning the QR works immediately.
				var resp api.EsimIssueResponse
				if err := c.post("/v1/esim/issue", api.EsimIssueRequest{
					IMSI: imsi, Address: address, EID: eid,
				}, &resp); err != nil {
					return err
				}
				fmt.Printf("eSIM profile issued for IMSI %s (control plane)\n", resp.IMSI)
				fmt.Printf("  iccid:  %s\n", resp.ICCID)
				if eid != "" {
					fmt.Printf("  eid:    %s (pinned)\n", eid)
				}
				fmt.Printf("  code:   LPA:1$%s$%s\n", resp.SMDPAddress, resp.ActivationCode)
				fmt.Printf("  smdp:   %s (served by fairwave-control at /es9plus)\n", resp.SMDPAddress)
				if qrOut != "" {
					if resp.QRPNGBase64 == "" {
						return fmt.Errorf("control plane returned no QR payload; try without --qr")
					}
					png, err := base64.StdEncoding.DecodeString(resp.QRPNGBase64)
					if err != nil {
						return err
					}
					if err := os.WriteFile(qrOut, png, 0o600); err != nil {
						return err
					}
					fmt.Printf("  qr:     %s\n", qrOut)
				}
				return nil
			}

			// standalone fallback: local registry + `fairwave esim serve`.
			sub, err := simprov.LoadTestVector(imsi)
			if err != nil {
				return fmt.Errorf("lab vector lookup: %w (only the three dummy lab IMSIs are supported)", err)
			}
			token, err := activation.GenerateToken(12)
			if err != nil {
				return err
			}
			p, err := profile.NewLabProfile(sub, address, profile.WithEID(eid))
			if err != nil {
				return err
			}
			reg, err := registry.Open(registryPath(cmd))
			if err != nil {
				return err
			}
			if err := reg.Add(token, p); err != nil {
				return err
			}
			code := activation.New(address, token)
			fmt.Printf("eSIM profile issued for IMSI %s (standalone)\n", sub.IMSI)
			fmt.Printf("  iccid:  %s\n", p.ICCID)
			if eid != "" {
				fmt.Printf("  eid:    %s (pinned)\n", eid)
			}
			fmt.Printf("  code:   %s\n", code.String())
			fmt.Printf("  smdp:   %s\n", address)
			if qrOut != "" {
				png, err := code.QR()
				if err != nil {
					return err
				}
				if err := os.WriteFile(qrOut, png, 0o600); err != nil {
					return err
				}
				fmt.Printf("  qr:     %s\n", qrOut)
			}
			if hssDriver != "" && hssDriver != "none" {
				writer := hsswrite.New(hssDriver, hssContainer)
				if err := writer.Add(cmd.Context(), sub); err != nil {
					return fmt.Errorf("hss write-back: %w", err)
				}
				fmt.Printf("  hss:    subscriber %s written to Open5GS (%s/%s)\n", sub.IMSI, hssDriver, hssContainer)
			} else {
				fmt.Printf("note: HSS write-back skipped; pass --hss-driver mongosh to auto-seed Open5GS\n")
			}
			fmt.Printf("next: run `fairwave esim serve` and scan the QR with a phone's eSIM setup\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&imsi, "imsi", "", "lab vector IMSI (required)")
	cmd.Flags().StringVar(&address, "address", "fairwave.local:8443", "SM-DP+ address in the activation code")
	cmd.Flags().StringVar(&eid, "eid", "", "pin the profile to a target eUICC EID (optional)")
	cmd.Flags().StringVar(&qrOut, "qr", "", "write the activation QR PNG to this path (optional)")
	cmd.Flags().StringVar(&hssDriver, "hss-driver", envOr("FAIRWAVE_HSS_DRIVER", "none"), "HSS write-back driver: mongosh | dbctl | free5gc | none")
	cmd.Flags().StringVar(&hssContainer, "hss-container", envOr("FAIRWAVE_HSS_CONTAINER", "mongo"), "container to exec the HSS write-back into")
	_ = cmd.MarkFlagRequired("imsi")
	return cmd
}

// esimServeCmd runs the lab SM-DP+ server backed by the local registry.
// It serves plain HTTP; in production terminate TLS at a reverse proxy or
// front the control plane (docs/security/esim.md).
func esimServeCmd() *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the lab SM-DP+ server (ES9+ endpoints)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, err := registry.Open(registryPath(cmd))
			if err != nil {
				return err
			}
			srv := smdp.NewServer("fairwave-esim", smdp.NewMemStore(), reg.Resolve)
			fmt.Printf("fairwave esim SM-DP+ listening on %s (lab mode, TLS terminates elsewhere)\n", addr)
			return http.ListenAndServe(addr, srv.Handler())
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8443", "listen address")
	return cmd
}

func registryPath(cmd *cobra.Command) string {
	dd, _ := cmd.Flags().GetString("data-dir")
	return filepath.Join(dd, "esim-registry.json")
}
