// Package simops bridges the API layer to the SIM provisioner library.
package simops

import (
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

// BatchSpec is a thin alias for the provisioner spec.
type BatchSpec = simprov.BatchSpec

// GenerateBatch delegates to the provisioner with lab-safe defaults.
func GenerateBatch(spec BatchSpec) ([]simprov.Subscriber, error) {
	return simprov.GenerateBatch(spec, "")
}
