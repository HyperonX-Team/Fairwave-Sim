package lifecycle

import (
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

func TestHappyPath(t *testing.T) {
	seq := []api.LifecyclePhase{
		api.PhaseProvision,
		api.PhaseRegister,
		api.PhaseOnAir,
		api.PhasePeer,
		api.PhaseBreakout,
	}
	for i := 0; i < len(seq)-1; i++ {
		if err := Transition(seq[i], seq[i+1]); err != nil {
			t.Fatalf("%s -> %s: %v", seq[i], seq[i+1], err)
		}
	}
}

func TestSkippingDenied(t *testing.T) {
	if err := Transition(api.PhaseProvision, api.PhaseOnAir); err == nil {
		t.Fatal("must not skip phases")
	}
	if err := Transition(api.PhaseBreakout, api.PhaseProvision); err == nil {
		t.Fatal("must not go backwards")
	}
}

func TestNoOp(t *testing.T) {
	if err := Transition(api.PhaseOnAir, api.PhaseOnAir); err != nil {
		t.Fatalf("same-phase no-op: %v", err)
	}
}

func TestInvalidPhase(t *testing.T) {
	if err := Transition(api.PhaseProvision, api.LifecyclePhase("bogus")); err == nil {
		t.Fatal("bogus phase must error")
	}
}
