package cli

import (
	"errors"
	"strings"
	"testing"
)

// nodeLeaveCmd runs HTTP against the control plane once the --yes gate is
// passed. All tests point --control at 127.0.0.1:1 (nothing listens there) so
// the backend error is a deterministic connection refusal and no control
// plane is ever required.

func TestNodeLeaveYesSkipsConfirmation(t *testing.T) {
	root := newTestRoot(t)
	root.SetArgs([]string{"node", "leave", "--yes", "--control", "http://127.0.0.1:1"})
	_, err := root.ExecuteC()
	if err == nil {
		t.Fatal("expected a backend error (control plane unreachable)")
	}
	// The confirmation gate was skipped: the failure must come from the HTTP
	// call, never the "requires --yes" refusal.
	if strings.Contains(err.Error(), "requires --yes") {
		t.Fatalf("--yes must skip the confirmation gate, got refusal: %v", err)
	}
	if errors.Is(err, ErrUsage) {
		t.Fatalf("backend error %q must not be a usage error", err)
	}
}

func TestNodeLeaveRefusesWithoutYesOnNonTTY(t *testing.T) {
	root := newTestRoot(t)
	// go test runs with stdout as a pipe, so confirmDestructive sees a
	// non-TTY and must refuse unless --yes is present.
	root.SetArgs([]string{"node", "leave"})
	_, err := root.ExecuteC()
	if err == nil {
		t.Fatal("expected refusal when stdout is not a TTY")
	}
	if !strings.Contains(err.Error(), "requires --yes") {
		t.Fatalf("refusal error %q must mention --yes", err)
	}
	if errors.Is(err, ErrUsage) {
		t.Fatalf("refusal is a general failure (exit 1), not a usage error (exit 2): %v", err)
	}
}

func TestNodeLeaveTooManyArgsIsUsageError(t *testing.T) {
	root := newTestRoot(t)
	root.SetArgs([]string{"node", "leave", "extra"})
	_, err := root.ExecuteC()
	if err == nil {
		t.Fatal("expected an error for extra positional args")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("errors.Is(%q, ErrUsage) = false; extra args are a usage error", err)
	}
}
