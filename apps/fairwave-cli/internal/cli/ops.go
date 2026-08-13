package cli

import (
	"fmt"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/spf13/cobra"
)

// nodeHealthCmd shows the latest agent heartbeats for every node.
func nodeHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Show per-node agent health (heartbeats)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var health []api.NodeHealth
			if err := c.get("/v1/health", &health); err != nil {
				return err
			}
			if len(health) == 0 {
				fmt.Println("no telemetry yet; start fairwave-agent on the node (it posts heartbeats to /v1/telemetry)")
				return nil
			}
			for _, h := range health {
				state := "down"
				if h.Up {
					state = "up"
				}
				line := fmt.Sprintf("%-16s %-4s last=%-8s load=%-5s radio=%-4s watchdog=%s",
					h.NodeID, state, time.Since(h.TS).Round(time.Second), fmt.Sprintf("%.2f", h.Load1), h.Radio, h.Watchdog)
				if h.SDRTempC != nil {
					line += fmt.Sprintf(" sdr=%.1fC", *h.SDRTempC)
				}
				if h.GPSDO {
					line += " gpsdo=locked"
				}
				fmt.Println(line)
			}
			return nil
		},
	}
}

// spectrumDisarmCmd clears the TX armed flag.
func spectrumDisarmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disarm",
		Short: "Disarm TX (always allowed; recorded in the audit log)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var resp api.TxArmResponse
			if err := c.post("/v1/tx/disarm", nil, &resp); err != nil {
				return err
			}
			fmt.Println("TX DISARMED")
			return nil
		},
	}
}

// auditCmd prints the append-only operator audit trail.
func auditCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Show the operator audit trail (append-only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var entries []api.AuditEntry
			if err := c.get("/v1/audit", &entries); err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("audit log is empty")
				return nil
			}
			if limit > 0 && limit < len(entries) {
				entries = entries[:limit]
			}
			for _, e := range entries {
				fmt.Printf("%-24s %-18s %-18s %s\n",
					e.TS.Format(time.RFC3339), e.Action, e.Target, e.Detail)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "show at most N most recent entries (0 = all)")
	return cmd
}
