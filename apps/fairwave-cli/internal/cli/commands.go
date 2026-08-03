package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
		Use:   "join",
		Short: "Join a neighboring box's mesh (peer)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Full mesh enrollment lives in M2 (peering milestone). For v0.1
			// we record the intent and print the documented manual path.
			if len(args) < 1 {
				return fmt.Errorf("usage: fairwave node join <endpoint host:port>")
			}
			fmt.Printf("join %s: peering mesh is a v0.3 feature; see docs/peering/\n", args[0])
			return nil
		},
	}
}

func nodeLeaveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "leave",
		Short: "Decommission a node",
		RunE: func(cmd *cobra.Command, args []string) error {
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
}

// ---- sim ----

func simCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sim",
		Short: "SIM lifecycle: issue, revoke, list",
	}
	cmd.AddCommand(simIssueCmd(), simRevokeCmd(), simListCmd())
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
		Short: "Revoke a SIM by IMSI",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: fairwave sim revoke <imsi>")
			}
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

func simListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List SIMs",
		RunE: func(cmd *cobra.Command, args []string) error {
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("usage: fairwave peer add <name> <endpoint host:port>")
			}
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
		Short: "Spectrum gate checks (read-only)",
	}
	cmd.AddCommand(spectrumCheckCmd(), spectrumArmCmd())
	return cmd
}

func spectrumCheckCmd() *cobra.Command {
	var country, band, licenseRef string
	var indoor bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check whether a band may transmit in a country",
		RunE: func(cmd *cobra.Command, args []string) error {
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
			} else {
				fmt.Println("TX DENIED:")
				for _, r := range resp.Reasons {
					fmt.Printf("  - %s\n", r)
				}
			}
			return nil
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
	var localBreakout bool
	var maxUEs int
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set policy (only the supported knobs)",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			var updated api.Policy
			if err := c.do("PUT", "/v1/policy", p, &updated); err != nil {
				return err
			}
			fmt.Printf("policy updated: local_breakout=%v max_ues=%d\n", updated.LocalBreakout, updated.MaxUEs)
			return nil
		},
	}
	cmd.Flags().BoolVar(&localBreakout, "local-breakout", true, "break out at edge NAT")
	cmd.Flags().IntVar(&maxUEs, "max-ues", 128, "fair-use UE cap")
	return cmd
}
