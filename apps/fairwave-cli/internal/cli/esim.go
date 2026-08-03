package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/activation"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/registry"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/smdp"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
	"github.com/spf13/cobra"
)

// esimCmd is the eSIM (SM-DP+) command group: mint activation codes and run
// the lab SM-DP+ server.
func esimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "esim",
		Short: "eSIM (SM-DP+) lifecycle: issue activation codes, run the lab server",
	}
	cmd.AddCommand(esimIssueCmd(), esimServeCmd())
	return cmd
}

// esimIssueCmd mints a lab eSIM profile + activation code for a lab vector
// IMSI and registers it in the local SM-DP+ registry.
func esimIssueCmd() *cobra.Command {
	var (
		imsi    string
		address string
		eid     string
		qrOut   string
	)
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Mint an eSIM profile + activation code (lab vectors only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			fmt.Printf("eSIM profile issued for IMSI %s\n", sub.IMSI)
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
			fmt.Printf("next: run `fairwave esim serve` and scan the QR with a phone's eSIM setup\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&imsi, "imsi", "", "lab vector IMSI (required)")
	cmd.Flags().StringVar(&address, "address", "fairwave.local:8443", "SM-DP+ address in the activation code")
	cmd.Flags().StringVar(&eid, "eid", "", "pin the profile to a target eUICC EID (optional)")
	cmd.Flags().StringVar(&qrOut, "qr", "", "write the activation QR PNG to this path (optional)")
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
