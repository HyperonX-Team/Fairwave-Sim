package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Process exit codes. The contract is documented in docs/software/fairwave-cli.md
// ("Rules for the CLI") and implemented by cmd/fairwave/main.go:
//
//	0  success
//	1  failure (general error)
//	2  usage error (bad arguments or flags)
//	3  policy/legal gate refused (spectrum block)
const (
	ExitOK          = 0
	ExitError       = 1
	ExitUsage       = 2
	ExitPolicyBlock = 3
)

// Sentinel errors returned by the command tree. main() classifies failures
// with errors.Is and maps them onto the exit codes above. Commands that hit a
// policy/legal gate (e.g. TX arming denied) should wrap ErrPolicyBlocked.
var (
	// ErrUsage marks argument/flag usage errors (exit 2).
	ErrUsage = errors.New("usage error")
	// ErrPolicyBlocked marks a policy/legal gate refusal (exit 3).
	ErrPolicyBlocked = errors.New("policy/legal gate refused")
)

// usageError wraps err with ErrUsage so errors.Is(err, ErrUsage) holds while
// keeping the original message intact.
func usageError(err error) error {
	return fmt.Errorf("%w: %w", ErrUsage, err)
}

// wrapArgs re-labels a cobra positional-args validator's failure as a usage
// error so main() can exit 2 instead of 1.
func wrapArgs(fn cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := fn(cmd, args); err != nil {
			return usageError(err)
		}
		return nil
	}
}

// ExactArgs requires exactly n positional arguments; violations are usage errors.
func ExactArgs(n int) cobra.PositionalArgs {
	return wrapArgs(cobra.ExactArgs(n))
}

// MinimumNArgs requires at least n positional arguments; violations are usage errors.
func MinimumNArgs(n int) cobra.PositionalArgs {
	return wrapArgs(cobra.MinimumNArgs(n))
}

// MaximumNArgs requires at most n positional arguments; violations are usage errors.
func MaximumNArgs(n int) cobra.PositionalArgs {
	return wrapArgs(cobra.MaximumNArgs(n))
}

// RangeArgs requires between minArgs and maxArgs positional arguments (inclusive);
// violations are usage errors.
func RangeArgs(minArgs, maxArgs int) cobra.PositionalArgs {
	return wrapArgs(cobra.RangeArgs(minArgs, maxArgs))
}
