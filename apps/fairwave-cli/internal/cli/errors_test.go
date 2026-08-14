package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestExitCodeContract locks the documented exit-code contract
// (docs/software/fairwave-cli.md, "Rules for the CLI"):
//
//	0 success, 1 general failure, 2 usage error, 3 policy/legal block.
func TestExitCodeContract(t *testing.T) {
	cases := []struct {
		got  int
		want int
	}{
		{ExitOK, 0},
		{ExitError, 1},
		{ExitUsage, 2},
		{ExitPolicyBlock, 3},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("exit code = %d, want %d", tc.got, tc.want)
		}
	}
}

func TestUsageSentinelWrapping(t *testing.T) {
	base := errors.New("boom")
	if !errors.Is(usageError(base), ErrUsage) {
		t.Fatal("usageError must wrap ErrUsage")
	}
	if errors.Is(usageError(base), ErrPolicyBlocked) {
		t.Fatal("usageError must NOT wrap ErrPolicyBlocked")
	}
	// The original error must still be reachable through the chain.
	if !errors.Is(usageError(base), base) {
		t.Fatal("usageError must preserve the wrapped error in the chain")
	}
}

func TestSentinelsDistinct(t *testing.T) {
	if ErrUsage == ErrPolicyBlocked {
		t.Fatal("ErrUsage and ErrPolicyBlocked must be distinct values")
	}
	if errors.Is(ErrUsage, ErrPolicyBlocked) {
		t.Fatal("ErrUsage must not be classified as ErrPolicyBlocked")
	}
	if errors.Is(ErrPolicyBlocked, ErrUsage) {
		t.Fatal("ErrPolicyBlocked must not be classified as ErrUsage")
	}
}

func TestArgValidatorsClassifyViolationsAsUsage(t *testing.T) {
	cmd := &cobra.Command{Use: "probe"}
	cases := []struct {
		name string
		args cobra.PositionalArgs
		argv []string
	}{
		{"ExactArgs(1) too few", ExactArgs(1), nil},
		{"ExactArgs(0) too many", ExactArgs(0), []string{"a"}},
		{"MinimumNArgs(1) too few", MinimumNArgs(1), nil},
		{"MaximumNArgs(0) too many", MaximumNArgs(0), []string{"a"}},
		{"RangeArgs(1,2) too few", RangeArgs(1, 2), nil},
		{"RangeArgs(1,2) too many", RangeArgs(1, 2), []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.args(cmd, tc.argv)
			if err == nil {
				t.Fatal("expected a validation error")
			}
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("errors.Is(%q, ErrUsage) = false; want true", err)
			}
			if errors.Is(err, ErrPolicyBlocked) {
				t.Fatalf("validator error %q must not be ErrPolicyBlocked", err)
			}
		})
	}
}

func TestArgValidatorsAcceptInRangeArgs(t *testing.T) {
	cmd := &cobra.Command{Use: "probe"}
	cases := []struct {
		name string
		args cobra.PositionalArgs
		argv []string
	}{
		{"ExactArgs(1)", ExactArgs(1), []string{"a"}},
		{"MinimumNArgs(1)", MinimumNArgs(1), []string{"a"}},
		{"MaximumNArgs(1)", MaximumNArgs(1), []string{"a"}},
		{"RangeArgs(1,2)", RangeArgs(1, 2), []string{"a", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.args(cmd, tc.argv); err != nil {
				t.Fatalf("in-range args rejected: %v", err)
			}
		})
	}
}

func TestNodeJoinReturnsNotImplementedError(t *testing.T) {
	root := newTestRoot(t)
	root.SetArgs([]string{"node", "join"})
	_, err := root.ExecuteC()
	if err == nil {
		t.Fatal("node join must fail loudly (peering is not implemented)")
	}
	// It is a general failure (exit 1), NOT a usage error (exit 2) and NOT a
	// policy block (exit 3): the feature simply does not exist yet.
	if errors.Is(err, ErrUsage) {
		t.Fatalf("node join error %q must not be ErrUsage", err)
	}
	if errors.Is(err, ErrPolicyBlocked) {
		t.Fatalf("node join error %q must not be ErrPolicyBlocked", err)
	}
	if !strings.Contains(err.Error(), "M2") {
		t.Fatalf("node join error %q should reference the M2 milestone", err)
	}
}
