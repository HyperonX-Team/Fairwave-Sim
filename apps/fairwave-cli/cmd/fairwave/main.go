// fairwave is the Fairwave operator CLI: nodes, SIMs, peers, spectrum gates.
package main

import (
	"errors"
	"os"
	"strings"

	"github.com/HyperonX-Team/Fairwave-Sim/apps/fairwave-cli/internal/cli"
)

func main() {
	err := cli.Root().Execute()
	if err == nil {
		return
	}
	// cobra already printed the error (and usage, for usage errors) to stderr
	// because root does not silence them; here we only map the documented
	// exit-code contract: 0 success, 1 general, 2 usage, 3 policy/legal block.
	switch {
	case errors.Is(err, cli.ErrPolicyBlocked):
		os.Exit(cli.ExitPolicyBlock)
	case errors.Is(err, cli.ErrUsage):
		os.Exit(cli.ExitUsage)
	case isCobraUsageError(err):
		os.Exit(cli.ExitUsage)
	default:
		os.Exit(cli.ExitError)
	}
}

// isCobraUsageError reports the failures cobra raises itself for missing
// required flags and unknown subcommands, which it does not wrap in a
// sentinel error.
func isCobraUsageError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "required flag(s)") ||
		strings.Contains(msg, "unknown command")
}
