package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
	"github.com/spf13/cobra"
)

func nodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage Fairwave nodes",
	}
	cmd.AddCommand(
		nodeInitCmd(),
		nodeStatusCmd(),
		nodeHealthCmd(),
		nodeJoinCmd(),
		nodeLeaveCmd(),
	)
	return cmd
}

func nodeInitCmd() *cobra.Command {
	var (
		name    string
		country string
		role    string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a node on the control plane",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			if country == "" {
				country = "LAB"
			}
			if role == "" {
				role = "edge"
			}
			node := api.Node{Name: name, Country: country, Role: role}
			var created api.Node
			if err := c.post("/v1/nodes", node, &created); err != nil {
				return err
			}
			fmt.Printf("node initialized: %s (id %s, phase %s)\n", created.Name, created.ID, created.Phase)
			fmt.Printf("next: fairwave node status\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "fairwave-1", "node display name")
	cmd.Flags().StringVar(&country, "country", "LAB", "country code (LAB = virtual test PLMN)")
	cmd.Flags().StringVar(&role, "role", "edge", "node role: edge | hub | lab")
	return cmd
}

func nodeStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show control plane + node status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var st api.Status
			if err := c.get("/v1/status", &st); err != nil {
				return err
			}
			var nodes []api.Node
			if err := c.get("/v1/nodes", &nodes); err != nil {
				return err
			}
			fmt.Printf("fairwave-control %s\n", st.Version)
			fmt.Printf("  mode:      %s\n  phase:     %s\n  country:   %s\n",
				st.Mode, st.Phase, st.Country)
			fmt.Printf("  tx armed:  %v\n  nodes:     %d\n  ues:       %d\n  peers:     %d\n",
				st.TxArmed, st.Nodes, st.UEs, st.Peers)
			for _, n := range nodes {
				fmt.Printf("  node %-16s id=%-20s phase=%-10s tx=%v\n", n.Name, n.ID, n.Phase, n.TxArmed)
			}
			return nil
		},
	}
}

func nodeJoinCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "join",
		Short:        "Join a neighboring box's mesh (peer)",
		SilenceUsage: true, // not a usage problem - the feature is simply missing
		RunE: func(_ *cobra.Command, _ []string) error {
			// Full mesh enrollment lives in M2 (peering milestone). Until then
			// the command must fail loudly instead of reporting success.
			return fmt.Errorf("node join: peering is not yet implemented (M2); see docs/peering/ for the manual path")
		},
	}
}

func nodeLeaveCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "leave",
		Short: "Decommission a node",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !yes {
				ok, err := confirmDestructive(cmd, "Decommission ALL node(s)")
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted: no nodes were decommissioned")
				}
			}
			c := newClient(cmd)
			var nodes []api.Node
			if err := c.get("/v1/nodes", &nodes); err != nil {
				return err
			}
			for _, n := range nodes {
				if err := c.do("POST", "/v1/nodes/"+n.ID+"/leave", nil, nil); err != nil {
					return err
				}
				fmt.Printf("node %s left\n", n.ID)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt (required when stdout is not a TTY)")
	return cmd
}

// confirmDestructive prompts for confirmation of a destructive action. When
// stdout is not a TTY the operator cannot have answered interactively, so we
// refuse unless --yes was passed. Confirmation is y/yes (case-insensitive).
func confirmDestructive(cmd *cobra.Command, what string) (bool, error) {
	fi, err := os.Stdout.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false, fmt.Errorf("%s requires --yes (refusing: stdout is not a TTY)", what)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s? [y/N] ", what)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	}
	return false, nil
}

// ---- sim ----

func simCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sim",
		Short: "SIM lifecycle: issue, revoke, suspend, resume, list, export",
	}
	cmd.AddCommand(simIssueCmd(), simRevokeCmd(), simSuspendCmd(), simResumeCmd(), simGetCmd(), simListCmd(), simExportCmd(), simImportCmd(), simQuotaCmd(), simUsageCmd())
	return cmd
}

func simIssueCmd() *cobra.Command {
	var (
		profile string
		count   int
	)
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Issue SIMs (lab profile generates Ki/OPc locally)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var issued []api.SIM
			req := api.SimIssueRequest{Profile: profile, Count: count}
			if err := c.post("/v1/sims", req, &issued); err != nil {
				return err
			}
			for _, s := range issued {
				fmt.Printf("issued SIM imsi=%s profile=%s status=%s apn=%s\n", s.IMSI, s.Profile, s.Status, s.APN)
			}
			fmt.Printf("note: Ki/OPc were generated in the vault; export via sim export (bureau runbook)\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "lab", "SIM profile: lab | prod")
	cmd.Flags().IntVar(&count, "count", 1, "number of SIMs")
	return cmd
}

func simRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke",
		Short: "Revoke a SIM by IMSI (removed from the HSS)",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var sim api.SIM
			if err := c.post("/v1/sims/"+args[0]+"/revoke", nil, &sim); err != nil {
				return err
			}
			fmt.Printf("revoked imsi=%s status=%s\n", sim.IMSI, sim.Status)
			return nil
		},
	}
}

func simSuspendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suspend",
		Short: "Suspend a SIM (deactivate, credentials kept)",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var sim api.SIM
			if err := c.post("/v1/sims/"+args[0]+"/suspend", nil, &sim); err != nil {
				return err
			}
			fmt.Printf("suspended imsi=%s status=%s\n", sim.IMSI, sim.Status)
			return nil
		},
	}
}

func simResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume a suspended SIM",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var sim api.SIM
			if err := c.post("/v1/sims/"+args[0]+"/resume", nil, &sim); err != nil {
				return err
			}
			fmt.Printf("resumed imsi=%s status=%s\n", sim.IMSI, sim.Status)
			return nil
		},
	}
}

func simGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show one SIM by IMSI",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var sim api.SIM
			if err := c.get("/v1/sims/"+args[0], &sim); err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(sim)
		},
	}
}

// simExportCmd writes bureau-friendly batch files for a card vendor.
// Ki/OPc are only available from the provisioner/vault, never the control
// plane; --lab-creds fills them from the public lab test vectors for
// testing/demos (never for real cards).
func simExportCmd() *cobra.Command {
	var format, out string
	var labCreds bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export issued SIMs as a bureau batch (CSV/JSON)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != "csv" && format != "json" {
				return fmt.Errorf("format must be csv or json (got %q)", format)
			}
			c := newClient(cmd)
			var sims []api.SIM
			if err := c.get("/v1/sims", &sims); err != nil {
				return err
			}
			if len(sims) == 0 {
				return fmt.Errorf("no SIMs in the store to export")
			}
			subs := make([]simprov.Subscriber, 0, len(sims))
			for _, s := range sims {
				sub := simprov.Subscriber{
					IMSI:   s.IMSI,
					MSISDN: s.MSISDN,
					APN:    s.APN,
					Class:  s.Profile,
				}
				if labCreds {
					if vec, err := simprov.LoadTestVector(s.IMSI); err == nil {
						sub.Ki, sub.OPc, sub.AMF, sub.SQN = vec.Ki, vec.OPc, vec.AMF, vec.SQN
					} else {
						return fmt.Errorf("SIM %s has no lab vector; --lab-creds only works for lab test IMSIs", s.IMSI)
					}
				}
				subs = append(subs, sub)
			}
			if out == "" {
				out = simprov.DefaultOutputDir()
			}
			if err := os.MkdirAll(out, 0o750); err != nil {
				return err
			}
			stamp := time.Now().Format("20060102-150405")
			var path string
			if format == "csv" {
				path = filepath.Join(out, "fairwave-sims-"+stamp+".csv")
				if err := simprov.WriteCSV(path, subs); err != nil {
					return err
				}
			} else {
				path = filepath.Join(out, "fairwave-sims-"+stamp+".json")
				if err := simprov.WriteJSON(path, subs); err != nil {
					return err
				}
			}
			fmt.Printf("exported %d SIMs -> %s\n", len(subs), path)
			if !labCreds {
				fmt.Println("note: Ki/OPc are not in the control plane; merge them from the provisioner/vault (--lab-creds for lab vectors only)")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "csv", "output format: csv | json")
	cmd.Flags().StringVar(&out, "out", "", "output dir (default out/<date>)")
	cmd.Flags().BoolVar(&labCreds, "lab-creds", false, "fill Ki/OPc from public lab test vectors (lab only!)")
	return cmd
}

func simListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List SIMs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var sims []api.SIM
			if err := c.get("/v1/sims", &sims); err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(sims)
		},
	}
}

// ---- peers ----

func peerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Peering mesh management",
	}
	cmd.AddCommand(peerListCmd(), peerAddCmd())
	return cmd
}

func peerListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List known peers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var peers []api.Peer
			if err := c.get("/v1/peers", &peers); err != nil {
				return err
			}
			for _, p := range peers {
				fmt.Printf("peer %-16s status=%-8s endpoint=%s allowed=%v\n",
					p.Name, p.Status, p.Endpoint, p.AllowedIPs)
			}
			return nil
		},
	}
}

func peerAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a peer (manual; mDNS auto-discovery in M2)",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var p api.Peer
			if err := c.post("/v1/peers", api.Peer{Name: args[0], Endpoint: args[1]}, &p); err != nil {
				return err
			}
			fmt.Printf("peer added: %s (id %s, status %s)\n", p.Name, p.ID, p.Status)
			return nil
		},
	}
}

// ---- spectrum ----

func spectrumCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spectrum",
		Short: "Spectrum gate checks and TX arming",
	}
	cmd.AddCommand(spectrumCheckCmd(), spectrumArmCmd(), spectrumDisarmCmd())
	return cmd
}

func spectrumCheckCmd() *cobra.Command {
	var country, band, licenseRef string
	var indoor bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check whether a band may transmit in a country",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var resp api.SpectrumCheckResponse
			if err := c.post("/v1/spectrum/check", api.SpectrumCheckRequest{
				Country: country, Band: band, Indoor: indoor, LicenseRef: licenseRef,
			}, &resp); err != nil {
				return err
			}
			fmt.Printf("spectrum check %s/%s indoor=%v: ", country, band, indoor)
			if resp.Allowed {
				fmt.Println("ALLOWED")
			} else {
				fmt.Println("DENIED")
				for _, r := range resp.Reasons {
					fmt.Printf("  - %s\n", r)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "country code (required)")
	cmd.Flags().StringVar(&band, "band", "", "band, e.g. n48, b3 (required)")
	cmd.Flags().BoolVar(&indoor, "indoor", true, "indoor deployment")
	cmd.Flags().StringVar(&licenseRef, "license-ref", "", "license/SAS reference if required")
	_ = cmd.MarkFlagRequired("country")
	_ = cmd.MarkFlagRequired("band")
	return cmd
}

func spectrumArmCmd() *cobra.Command {
	var country, band, licenseRef string
	cmd := &cobra.Command{
		Use:   "arm",
		Short: "Arm TX (requires acknowledgment; lab mode refuses)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var resp api.TxArmResponse
			if err := c.post("/v1/tx/arm", api.TxArmRequest{
				Country:        country,
				Band:           band,
				Acknowledgment: "I hold authorization for this transmission",
				LicenseRef:     licenseRef,
			}, &resp); err != nil {
				return err
			}
			if resp.Armed {
				fmt.Println("TX ARMED (authorization recorded)")
				return nil
			}
			reason := strings.Join(resp.Reasons, "; ")
			if reason == "" {
				reason = "gate refused"
			}
			return fmt.Errorf("%w: TX arming denied (%s)", ErrPolicyBlocked, reason)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "country code (required)")
	cmd.Flags().StringVar(&band, "band", "", "band (required)")
	cmd.Flags().StringVar(&licenseRef, "license-ref", "", "license/SAS reference")
	_ = cmd.MarkFlagRequired("country")
	_ = cmd.MarkFlagRequired("band")
	return cmd
}

// ---- policy ----

func policyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Routing/QoS policy",
	}
	cmd.AddCommand(policyGetCmd(), policySetCmd())
	return cmd
}

func policyGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show current policy",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var p api.Policy
			if err := c.get("/v1/policy", &p); err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(p)
		},
	}
}

func policySetCmd() *cobra.Command {
	var (
		localBreakout bool
		maxUEs        int
		hubPeer       string
		qosDL         int
		qosUL         int
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set policy knobs (only the ones you pass change)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var p api.Policy
			if err := c.get("/v1/policy", &p); err != nil {
				return err
			}
			if cmd.Flags().Changed("local-breakout") {
				p.LocalBreakout = localBreakout
			}
			if cmd.Flags().Changed("max-ues") {
				p.MaxUEs = maxUEs
			}
			if cmd.Flags().Changed("hub-peer") {
				p.HubPeer = hubPeer
			}
			if cmd.Flags().Changed("qos-dl-mbps") {
				p.QoSDLMbps = qosDL
			}
			if cmd.Flags().Changed("qos-ul-mbps") {
				p.QoSULMbps = qosUL
			}
			var updated api.Policy
			if err := c.do("PUT", "/v1/policy", p, &updated); err != nil {
				return err
			}
			fmt.Printf("policy updated: local_breakout=%v max_ues=%d hub=%q qos=%d/%d Mbps\n",
				updated.LocalBreakout, updated.MaxUEs, updated.HubPeer, updated.QoSDLMbps, updated.QoSULMbps)
			return nil
		},
	}
	cmd.Flags().BoolVar(&localBreakout, "local-breakout", true, "break out at edge NAT")
	cmd.Flags().IntVar(&maxUEs, "max-ues", 128, "fair-use UE cap")
	cmd.Flags().StringVar(&hubPeer, "hub-peer", "", "optional hub peer id for off-site traffic")
	cmd.Flags().IntVar(&qosDL, "qos-dl-mbps", 0, "default DL cap per UE (Mbps)")
	cmd.Flags().IntVar(&qosUL, "qos-ul-mbps", 0, "default UL cap per UE (Mbps)")
	return cmd
}
