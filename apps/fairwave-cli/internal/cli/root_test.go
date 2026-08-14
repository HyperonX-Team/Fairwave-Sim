package cli

import (
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// groupOnlyCommands are the commands that legitimately have no RunE: they are
// pure group nodes whose children carry the actual work. Each entry states why
// the absence of RunE is intentional. Every other command in the tree must
// have a RunE.
var groupOnlyCommands = map[string]string{
	"fairwave":          "root group: container for the top-level command groups",
	"fairwave node":     "group: node lifecycle subcommands (init/status/health/join/leave)",
	"fairwave sim":      "group: SIM lifecycle subcommands (issue/revoke/.../import)",
	"fairwave esim":     "group: eSIM (SM-DP+) subcommands (issue/list/revoke/serve)",
	"fairwave peer":     "group: peering subcommands (list/add)",
	"fairwave policy":   "group: routing/QoS policy subcommands (get/set)",
	"fairwave spectrum": "group: spectrum gate subcommands (check/arm/disarm)",
	"fairwave token":    "group: scoped API token subcommands (create/list/revoke)",
	"fairwave config":   "group: configuration subcommands (validate)",
}

// newTestRoot returns a fresh command tree with cobra's error/usage printing
// silenced so test output stays readable. Root() builds a new tree on every
// call, so mutating the returned instance is safe.
func newTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	root := Root()
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root
}

// walkCommands visits cmd and every descendant, returning the full path of
// each command (e.g. "fairwave sim import") and indexing each path to its
// command in byPath.
func walkCommands(t *testing.T, cmd *cobra.Command, path string, byPath map[string]*cobra.Command) []string {
	t.Helper()
	byPath[path] = cmd
	names := []string{path}
	for _, sub := range cmd.Commands() {
		names = append(names, walkCommands(t, sub, path+" "+sub.Name(), byPath)...)
	}
	return names
}

func TestCommandTreeShape(t *testing.T) {
	root := newTestRoot(t)
	if got := root.Name(); got != "fairwave" {
		t.Fatalf("root.Name() = %q, want %q", got, "fairwave")
	}
	if root.Use == "" {
		t.Error("root: empty Use")
	}
	if root.Short == "" {
		t.Error("root: empty Short")
	}

	wantTop := []string{
		"alerts", "audit", "backup", "compliance", "config", "doctor", "esim",
		"node", "peer", "policy", "restore", "sim", "spectrum", "token", "version",
	}
	var gotTop []string
	for _, c := range root.Commands() {
		gotTop = append(gotTop, c.Name())
	}
	sort.Strings(gotTop)
	if strings.Join(gotTop, ",") != strings.Join(wantTop, ",") {
		t.Fatalf("top-level commands = %v, want %v", gotTop, wantTop)
	}
}

func TestCommandTreeEveryCommandWellFormed(t *testing.T) {
	root := newTestRoot(t)
	byPath := make(map[string]*cobra.Command)
	paths := walkCommands(t, root, "fairwave", byPath)
	if len(paths) < 10 {
		t.Fatalf("tree unexpectedly small: %d commands", len(paths))
	}

	for _, path := range paths {
		cmd := byPath[path]
		if cmd.Use == "" {
			t.Errorf("%s: empty Use", path)
		}
		if cmd.Short == "" {
			t.Errorf("%s: empty Short", path)
		}
		if cmd.RunE == nil {
			if _, ok := groupOnlyCommands[path]; !ok {
				t.Errorf("%s: nil RunE and not listed in groupOnlyCommands (add it there with a justification if it is a pure group node)", path)
			}
			continue
		}
		if _, ok := groupOnlyCommands[path]; ok {
			t.Errorf("%s: listed as group-only but has a RunE", path)
		}
	}
}

func TestRootUnknownFlagIsUsageError(t *testing.T) {
	root := newTestRoot(t)
	root.SetArgs([]string{"--definitely-not-a-real-flag"})
	_, err := root.ExecuteC()
	if err == nil {
		t.Fatal("expected an error for an unknown flag")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("errors.Is(%q, ErrUsage) = false; unknown flags must map to usage errors (exit 2)", err)
	}
}
