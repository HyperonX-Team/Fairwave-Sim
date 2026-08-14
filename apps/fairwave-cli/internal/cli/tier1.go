package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/spf13/cobra"
)

// ---- sim import ----

func simImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import a bureau batch (CSV or JSON) into the control plane",
		Long: "Import a bureau batch (CSV or JSON) into the control plane.\n" +
			"CSV headers: imsi,msisdn,profile,apn,status,expires_at (RFC3339).\n" +
			"JSON: an array of {\"imsi\",\"msisdn\",\"profile\",\"apn\",\"status\",\"expires_at\"}.\n" +
			"Ki/OPc are never part of an import - they stay in the bureau's vault.",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := parseSimImport(args[0])
			if err != nil {
				return err
			}
			c := newClient(cmd)
			var resp api.SimImportResponse
			if err := c.post("/v1/sims/import", api.SimImportRequest{Sims: items}, &resp); err != nil {
				return err
			}
			fmt.Printf("import: %d new, %d updated, %d skipped\n", resp.Imported, resp.Updated, len(resp.Skipped))
			for _, s := range resp.Skipped {
				fmt.Printf("  skipped: %s\n", s)
			}
			return nil
		},
	}
}

func parseSimImport(path string) ([]api.SimImportItem, error) {
	if len(path) < 1 {
		return nil, fmt.Errorf("sim import: empty path")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json", ".csv":
	default:
		return nil, fmt.Errorf("sim import: %q: unsupported extension %q (use .csv or .json)", path, ext)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if ext == ".json" {
		var items []api.SimImportItem
		if err := json.NewDecoder(f).Decode(&items); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		return items, nil
	}
	// CSV
	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[h] = i
	}
	var items []api.SimImportItem
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		item := api.SimImportItem{IMSI: cell(row, col, "imsi"), MSISDN: cell(row, col, "msisdn"),
			Profile: cell(row, col, "profile"), APN: cell(row, col, "apn"), Status: cell(row, col, "status")}
		if exp := cell(row, col, "expires_at"); exp != "" {
			if t, err := time.Parse(time.RFC3339, exp); err == nil {
				item.ExpiresAt = t
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func cell(row []string, col map[string]int, name string) string {
	i, ok := col[name]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}

// ---- quota / usage ----

func simQuotaCmd() *cobra.Command {
	var bytes uint64
	cmd := &cobra.Command{
		Use:   "quota",
		Short: "Set a SIM's fair-use data allowance (0 = unlimited)",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var sim api.SIM
			if err := c.post("/v1/sims/"+args[0]+"/quota", api.SimQuotaRequest{QuotaBytes: bytes}, &sim); err != nil {
				return err
			}
			fmt.Printf("quota for %s: %d bytes (%s)\n", sim.IMSI, sim.QuotaBytes, fmtBytesCLI(sim.QuotaBytes))
			return nil
		},
	}
	cmd.Flags().Uint64Var(&bytes, "bytes", 0, "quota in bytes (0 = unlimited)")
	_ = cmd.MarkFlagRequired("bytes")
	return cmd
}

func simUsageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usage",
		Short: "Show a SIM's accumulated data usage",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			var u api.SimUsage
			if err := c.get("/v1/sims/"+args[0]+"/usage", &u); err != nil {
				return err
			}
			fmt.Printf("imsi=%s up=%s dn=%s total=%s quota=%s updated=%s\n",
				u.IMSI, fmtBytesCLI(u.BytesUp), fmtBytesCLI(u.BytesDn), fmtBytesCLI(u.BytesUp+u.BytesDn),
				fmtBytesCLI(u.QuotaBytes), u.UpdatedAt.Format(time.RFC3339))
			return nil
		},
	}
}

func fmtBytesCLI(n uint64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// ---- compliance ----

func complianceCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "compliance",
		Short: "Export the regulator-ready compliance report (CSV)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			data, err := c.raw("GET", "/v1/compliance/export", nil, nil)
			if err != nil {
				return err
			}
			if out == "" {
				out = fmt.Sprintf("fairwave-compliance-%s.csv", time.Now().Format("20060102"))
			}
			if err := os.WriteFile(out, data, 0o600); err != nil {
				return err
			}
			fmt.Printf("compliance report -> %s (%d bytes)\n", out, len(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output path (default fairwave-compliance-<date>.csv)")
	return cmd
}

// ---- backup / restore ----

func backupCmd() *cobra.Command {
	var out, passphrase string
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Back up all control-plane state (tar.gz, optionally encrypted)",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			headers := map[string]string{}
			if passphrase != "" {
				headers["X-Fairwave-Passphrase"] = passphrase
			}
			data, err := c.raw("GET", "/v1/backup", nil, headers)
			if err != nil {
				return err
			}
			if out == "" {
				out = fmt.Sprintf("fairwave-backup-%s.tar.gz", time.Now().Format("20060102-150405"))
			}
			if err := os.WriteFile(out, data, 0o600); err != nil {
				return err
			}
			fmt.Printf("backup -> %s (%d bytes)\n", out, len(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output path (default fairwave-backup-<ts>.tar.gz)")
	cmd.Flags().StringVar(&passphrase, "passphrase", "", "encrypt the archive (AES-256-GCM, key=SHA-256 of passphrase)")
	return cmd
}

func restoreCmd() *cobra.Command {
	var passphrase string
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a backup archive (restart the control plane afterwards)",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			c := newClient(cmd)
			headers := map[string]string{}
			if passphrase != "" {
				headers["X-Fairwave-Passphrase"] = passphrase
			}
			var resp map[string]string
			body, err := c.raw("POST", "/v1/restore", data, headers)
			if err != nil {
				return err
			}
			_ = json.Unmarshal(body, &resp)
			fmt.Printf("restore: %s\n", resp["message"])
			return nil
		},
	}
	cmd.Flags().StringVar(&passphrase, "passphrase", "", "passphrase the archive was encrypted with")
	return cmd
}

// ---- tokens ----

func tokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Scoped API tokens (admin only)",
	}
	cmd.AddCommand(tokenCreateCmd(), tokenListCmd(), tokenRevokeCmd())
	return cmd
}

func tokenCreateCmd() *cobra.Command {
	var name, role string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Mint a scoped API token (secret shown once)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var resp api.TokenCreateResponse
			if err := c.post("/v1/tokens", api.TokenCreateRequest{Name: name, Role: api.TokenRole(role)}, &resp); err != nil {
				return err
			}
			fmt.Printf("token created (id %s, role %s)\n", resp.ID, resp.Role)
			fmt.Printf("  secret: %s\n", resp.Token)
			fmt.Println("store it safely; it cannot be shown again")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "human-readable name (required)")
	cmd.Flags().StringVar(&role, "role", "operator", "role: admin | operator | viewer")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func tokenListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List scoped tokens",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newClient(cmd)
			var tokens []api.Token
			if err := c.get("/v1/tokens", &tokens); err != nil {
				return err
			}
			for _, t := range tokens {
				state := "active"
				if t.Revoked {
					state = "revoked"
				}
				fmt.Printf("%-10s %-16s role=%-8s created=%s state=%s\n",
					t.ID, t.Name, t.Role, t.CreatedAt.Format(time.RFC3339), state)
			}
			return nil
		},
	}
}

func tokenRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke",
		Short: "Revoke a scoped token",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient(cmd)
			if err := c.do("DELETE", "/v1/tokens/"+args[0], nil, nil); err != nil {
				return err
			}
			fmt.Printf("token %s revoked\n", args[0])
			return nil
		},
	}
}

// ---- alerts ----

func alertsCmd() *cobra.Command {
	var watch bool
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "List operational alerts (active first)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			for {
				c := newClient(cmd)
				var alerts []api.Alert
				if err := c.get("/v1/alerts", &alerts); err != nil {
					return err
				}
				if len(alerts) == 0 {
					fmt.Println("no alerts")
				} else {
					for _, a := range alerts {
						state := "active"
						if a.Resolved {
							state = "resolved"
						}
						fmt.Printf("[%-8s] [%-8s] %-30s %s %s\n",
							state, a.Severity, a.Message, a.Target, a.TS.Format(time.RFC3339))
					}
				}
				if !watch {
					return nil
				}
				fmt.Println("--- watching (ctrl-c to stop) ---")
				time.Sleep(10 * time.Second)
			}
		},
	}
	cmd.Flags().BoolVar(&watch, "watch", false, "poll every 10s")
	return cmd
}
