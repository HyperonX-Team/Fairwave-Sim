package cli

import (
	"encoding/json"
	"fmt"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/config"
	"github.com/spf13/cobra"
)

// ---- config ----

// configCmd is the group node for configuration subcommands.
func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Validate Fairwave configuration files",
	}
	cmd.AddCommand(configValidateCmd())
	return cmd
}

// validateResult is the JSON document emitted by `config validate` on stdout.
type validateResult struct {
	Valid bool   `json:"valid"`
	Path  string `json:"path"`
	Error string `json:"error,omitempty"`
}

func configValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <path>",
		Short: "Validate a fairwave-control.yaml configuration file",
		Long: "Validate <path> against the embedded fairwave-control JSON schema " +
			"and the semantic rules in core/fairwave-control/config.\n\n" +
			"config.Load applies FAIRWAVE_* environment overrides (e.g. " +
			"FAIRWAVE_SERVER_MODE, FAIRWAVE_TAC) before validating, so the same " +
			"file can validate differently depending on the environment.\n\n" +
			"Exits 0 with {\"valid\":true} on success, 1 with {\"valid\":false} " +
			"and the failure message on error.",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if _, err := config.Load(path); err != nil {
				printConfigValidate(cmd, validateResult{Valid: false, Path: path, Error: err.Error()})
				return err
			}
			printConfigValidate(cmd, validateResult{Valid: true, Path: path})
			return nil
		},
	}
}

// printConfigValidate emits a single-line JSON result to the command's stdout.
func printConfigValidate(cmd *cobra.Command, r validateResult) {
	b, err := json.Marshal(r)
	if err != nil {
		// validateResult is a static shape; marshal cannot fail.
		fmt.Fprintln(cmd.OutOrStdout(), `{"valid":false}`)
		return
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
}
