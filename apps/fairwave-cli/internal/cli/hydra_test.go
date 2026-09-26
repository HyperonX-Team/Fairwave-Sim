package cli

import (
	"errors"
	"strings"
	"testing"
)

// The hydra commands are exercised against 127.0.0.1:1 (nothing listens),
// so a backend failure is a deterministic connection refusal and no control
// plane is required.

func TestHydraStatusUnreachable(t *testing.T) {
	root := newTestRoot(t)
	root.SetArgs([]string{"hydra", "status", "--control", "http://127.0.0.1:1"})
	_, err := root.ExecuteC()
	if err == nil {
		t.Fatal("expected a backend error (control plane unreachable)")
	}
	if errors.Is(err, ErrUsage) {
		t.Fatalf("backend error %q must not be a usage error", err)
	}
}

func TestHydraWeaveCreateRequiresThreads(t *testing.T) {
	root := newTestRoot(t)
	root.SetArgs([]string{"hydra", "weave-create", "--id", "w", "--threads", " , , ", "--control", "http://127.0.0.1:1"})
	_, err := root.ExecuteC()
	if err == nil {
		t.Fatal("expected an error for an empty --threads list")
	}
	if !strings.Contains(err.Error(), "at least one thread") {
		t.Fatalf("error %q must explain the empty thread list", err)
	}
}

func TestHydraThreadAddRequiresFlags(t *testing.T) {
	root := newTestRoot(t)
	root.SetArgs([]string{"hydra", "thread-add", "--control", "http://127.0.0.1:1"})
	_, err := root.ExecuteC()
	if err == nil {
		t.Fatal("expected an error for missing required flags")
	}
	// cobra reports unset required flags directly (not via the usage-error
	// wrapper), so assert on the message rather than ErrUsage.
	if !strings.Contains(err.Error(), "required flag") {
		t.Fatalf("error %q must mention the missing required flags", err)
	}
}

func TestHydraThreadRemoveTooManyArgs(t *testing.T) {
	root := newTestRoot(t)
	root.SetArgs([]string{"hydra", "thread-remove", "a", "b"})
	_, err := root.ExecuteC()
	if err == nil {
		t.Fatal("expected an error for extra positional args")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("extra args must be a usage error, got %v", err)
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" a , b ,, c ")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("splitCSV = %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitCSV[%d] = %q want %q", i, got[i], want[i])
		}
	}
	if len(splitCSV("")) != 0 {
		t.Fatal("splitCSV(\"\") must be empty")
	}
}
