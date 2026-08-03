// Package policy implements the spectrum gate: country profiles, band
// allow-lists, and the acknowledgment logic that decides whether TX may be
// armed. This is the single source of truth for "what may transmit where".
package policy

import (
	"encoding/json"
	"fmt"
	"os"
)

// GateAckPhrase is the exact acknowledgment an operator must repeat to arm TX.
const GateAckPhrase = "I hold authorization for this transmission"

// Band is one permitted band within a country profile.
type Band struct {
	EarfcnMin   int    `json:"earfcn_min"`
	EarfcnMax   int    `json:"earfcn_max"`
	IndoorOnly  bool   `json:"indoor_only"`
	MaxEIRPDBM  int    `json:"max_eirp_dbm"`
	LicenseType string `json:"license_type"` // "shared", "local", "cbrs", "lab"
	Notes       string `json:"notes,omitempty"`
}

// Profile is one country's spectrum posture.
type Profile struct {
	Country string          `json:"country"`
	MCC     string          `json:"mcc"`
	Bands   map[string]Band `json:"bands"`
}

// Verdict is the outcome of a spectrum check.
type Verdict struct {
	Allowed bool
	Reasons []string
}

// Registry holds the curated country profiles.
type Registry struct {
	profiles map[string]Profile
}

// DefaultRegistry loads the built-in profiles plus any file at path
// (optional operator overlay, same JSON shape, merged).
func DefaultRegistry(overlayPath string) (*Registry, error) {
	r := &Registry{profiles: map[string]Profile{}}
	builtin := builtinProfiles()
	for c, p := range builtin {
		r.profiles[c] = p
	}
	if overlayPath != "" {
		data, err := os.ReadFile(overlayPath)
		if err != nil {
			return nil, fmt.Errorf("read overlay profile %s: %w", overlayPath, err)
		}
		var overlay map[string]Profile
		if err := json.Unmarshal(data, &overlay); err != nil {
			return nil, fmt.Errorf("parse overlay profile: %w", err)
		}
		for c, p := range overlay {
			r.profiles[c] = p
		}
	}
	return r, nil
}

// Check evaluates whether a band may transmit in a country with the given
// constraints. It returns a verdict with human-readable reasons.
func (r *Registry) Check(country, band string, indoor bool, licenseRef string) Verdict {
	var reasons []string
	allow := func(ok bool, why string) bool {
		if !ok {
			reasons = append(reasons, why)
		}
		return ok
	}

	p, ok := r.profiles[country]
	allow(ok, fmt.Sprintf("country %q has no curated spectrum profile; RF stays disabled", country))
	if !ok {
		return Verdict{Allowed: false, Reasons: reasons}
	}
	b, ok := p.Bands[band]
	allow(ok, fmt.Sprintf("band %q is not in the %s allow-list", band, country))
	if !ok {
		return Verdict{Allowed: false, Reasons: reasons}
	}
	allow(b.IndoorOnly == indoor || !b.IndoorOnly,
		fmt.Sprintf("band %q is indoor-only; deployment must be indoor", band))
	if b.LicenseType != "lab" {
		allow(licenseRef != "", fmt.Sprintf("band %q requires license reference (got none)", band))
	}
	return Verdict{Allowed: len(reasons) == 0, Reasons: reasons}
}

// MaxEIRP returns the cap for a country/band, or 0 (deny) if unlisted.
func (r *Registry) MaxEIRP(country, band string) int {
	p, ok := r.profiles[country]
	if !ok {
		return 0
	}
	b, ok := p.Bands[band]
	if !ok {
		return 0
	}
	return b.MaxEIRPDBM
}

// Countries lists known profiles (for CLI completion / docs).
func (r *Registry) Countries() []string {
	out := make([]string, 0, len(r.profiles))
	for c := range r.profiles {
		out = append(out, c)
	}
	return out
}

// builtinProfiles are curated from public regulatory summaries. They are a
// starting point, NOT legal advice — operators confirm with their regulator.
func builtinProfiles() map[string]Profile {
	return map[string]Profile{
		// LAB: country code "999" is the ITU-allocated test MCC range.
		// Only zmq/virtual devices may ever use it — no real RF.
		"LAB": {
			Country: "999",
			MCC:     "999",
			Bands: map[string]Band{
				"zmq": {
					EarfcnMin: 0, EarfcnMax: 65534, IndoorOnly: true,
					MaxEIRPDBM: -100, LicenseType: "lab",
					Notes: "Virtual radio only. RF power is impossible by construction.",
				},
			},
		},
		"US": {
			Country: "US",
			MCC:     "310",
			Bands: map[string]Band{
				"n48": {
					EarfcnMin: 55240, EarfcnMax: 56739, MaxEIRPDBM: 40,
					LicenseType: "cbrs", Notes: "CBRS GAA requires certified SAS; PAL requires grant.",
				},
				"n46": {
					EarfcnMin: 246, EarfcnMax: 636, IndoorOnly: true, MaxEIRPDBM: 36,
					LicenseType: "shared", Notes: "U-NII indoor small-cell, check FCC rules.",
				},
			},
		},
		"GB": {
			Country: "GB",
			MCC:     "234",
			Bands: map[string]Band{
				"b1":  {EarfcnMin: 300, EarfcnMax: 599, MaxEIRPDBM: 40, LicenseType: "local"},
				"b3":  {EarfcnMin: 1200, EarfcnMax: 1949, MaxEIRPDBM: 40, LicenseType: "local"},
				"b7":  {EarfcnMin: 2750, EarfcnMax: 3449, MaxEIRPDBM: 37, LicenseType: "local"},
				"b38": {EarfcnMin: 37750, EarfcnMax: 38249, MaxEIRPDBM: 37, LicenseType: "local"},
			},
		},
		"DE": {
			Country: "DE",
			MCC:     "262",
			Bands: map[string]Band{
				"b3":  {EarfcnMin: 1200, EarfcnMax: 1949, MaxEIRPDBM: 36, LicenseType: "local"},
				"b7":  {EarfcnMin: 2750, EarfcnMax: 3449, MaxEIRPDBM: 33, LicenseType: "local"},
				"b40": {EarfcnMin: 38650, EarfcnMax: 39649, MaxEIRPDBM: 33, LicenseType: "local"},
			},
		},
		"FR": {
			Country: "FR",
			MCC:     "208",
			Bands: map[string]Band{
				"b3":  {EarfcnMin: 1200, EarfcnMax: 1949, MaxEIRPDBM: 36, LicenseType: "local"},
				"b7":  {EarfcnMin: 2750, EarfcnMax: 3449, MaxEIRPDBM: 33, LicenseType: "local"},
				"b38": {EarfcnMin: 37750, EarfcnMax: 38249, MaxEIRPDBM: 33, LicenseType: "local"},
			},
		},
		"IN": {
			Country: "IN",
			MCC:     "405",
			Bands: map[string]Band{
				"b3": {EarfcnMin: 1200, EarfcnMax: 1949, MaxEIRPDBM: 33, LicenseType: "local",
					Notes: "Captive/enterprise use requires DoT approval."},
				"b40": {EarfcnMin: 38650, EarfcnMax: 39649, MaxEIRPDBM: 33, LicenseType: "local"},
			},
		},
		"AU": {
			Country: "AU",
			MCC:     "505",
			Bands: map[string]Band{
				"b28": {EarfcnMin: 9210, EarfcnMax: 9659, MaxEIRPDBM: 33, LicenseType: "local"},
				"b3":  {EarfcnMin: 1200, EarfcnMax: 1949, MaxEIRPDBM: 33, LicenseType: "local"},
			},
		},
		"CA": {
			Country: "CA",
			MCC:     "302",
			Bands: map[string]Band{
				"b7": {EarfcnMin: 2750, EarfcnMax: 3449, MaxEIRPDBM: 33, LicenseType: "local"},
				"n48": {
					EarfcnMin: 55240, EarfcnMax: 56739, MaxEIRPDBM: 40,
					LicenseType: "shared", Notes: "Shared access regime; confirm ISED policy.",
				},
			},
		},
	}
}
