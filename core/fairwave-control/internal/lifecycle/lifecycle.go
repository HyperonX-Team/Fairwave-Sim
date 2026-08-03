// Package lifecycle implements the node lifecycle state machine:
// provision → register → on-air → peer → breakout.
package lifecycle

import (
	"fmt"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// Transition validates a requested phase transition. Transitions may only
// move forward one step at a time (or to the same phase = no-op).
func Transition(current, next api.LifecyclePhase) error {
	if current == next {
		return nil
	}
	var curIdx, nextIdx int = -1, -1
	for i, p := range api.AllPhases {
		if p == current {
			curIdx = i
		}
		if p == next {
			nextIdx = i
		}
	}
	if curIdx == -1 || nextIdx == -1 {
		return fmt.Errorf("invalid phase %q or %q", current, next)
	}
	if nextIdx != curIdx+1 {
		next := "<terminal>"
		if curIdx+1 < len(api.AllPhases) {
			next = string(api.AllPhases[curIdx+1])
		}
		return fmt.Errorf("phase %q cannot follow %q (must step through %s)",
			next, current, next)
	}
	return nil
}
