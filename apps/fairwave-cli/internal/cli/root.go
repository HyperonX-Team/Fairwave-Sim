// Package cli implements the `fairwave` command-line tool.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Version of the CLI (overridden at build time).
var Version = "0.1.0"

// DefaultControlURL is the local control plane.
const DefaultControlURL = "http://localhost:8080"

// Root returns the fairwave command tree.
func Root() *cobra.Command {
	root := &cobra.Command{
		Use:           "fairwave",
		Short:         "Fairwave - community carrier in a pizza box",
		Long:          "Fairwave CLI: manage nodes, SIMs, peers, spectrum gates, and the local control plane.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("control", envOr("FAIRWAVE_CONTROL", DefaultControlURL), "control plane base URL")
	root.PersistentFlags().String("token", os.Getenv("FAIRWAVE_ADMIN_TOKEN"), "admin bearer token (or FAIRWAVE_ADMIN_TOKEN)")
	root.PersistentFlags().String("data-dir", filepath.Join(".", "data"), "local data dir")

	root.AddCommand(
		nodeCmd(),
		simCmd(),
		peerCmd(),
		spectrumCmd(),
		policyCmd(),
		doctorCmd(),
		versionCmd(),
	)
	return root
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// client resolves the control endpoint + token into an API client.
type client struct {
	base  string
	token string
}

func newClient(cmd *cobra.Command) *client {
	base, _ := cmd.Flags().GetString("control")
	tok, _ := cmd.Flags().GetString("token")
	if tok == "" {
		// fall back to token file written by the control plane
		if dd, _ := cmd.Flags().GetString("data-dir"); dd != "" {
			if b, err := os.ReadFile(filepath.Join(dd, "admin_token")); err == nil && len(b) > 0 {
				tok = string(b)
			}
		}
	}
	return &client{base: base, token: tok}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("fairwave %s\n", Version)
			return nil
		},
	}
}
