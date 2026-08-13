// Package registry is the local file-backed SM-DP+ profile registry: it
// maps activation codes to the carrier profiles that may be downloaded.
// The file holds the full encrypted-payload material (including lab
// Milenage credentials) and is written with 0600 permissions - treat it
// like the SIM vault (docs/adr/0006-sim-vault.md). Writes are atomic
// (tmp + rename).
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
)

// Entry is one activation code in the registry.
type Entry struct {
	Token        string           `json:"token"`
	Profile      *profile.Profile `json:"profile"`
	CreatedAt    time.Time        `json:"created_at"`
	DownloadedAt *time.Time       `json:"downloaded_at,omitempty"`
	ExpiresAt    *time.Time       `json:"expires_at,omitempty"` // code-level expiry (may be nil)
}

// ErrActivationCodeUsed means a single-use code was already downloaded.
var ErrActivationCodeUsed = errors.New("registry: activation code already used")

// ErrActivationCodeExpired means the code's validity window has passed.
var ErrActivationCodeExpired = errors.New("registry: activation code expired")

// Registry is a file-backed map of activation codes to profiles.
type Registry struct {
	mu      sync.Mutex
	path    string
	entries map[string]*Entry
}

// Open loads the registry from path, creating it if absent. The file is
// always chmod 0600.
func Open(path string) (*Registry, error) {
	r := &Registry{path: path, entries: make(map[string]*Entry)}
	data, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err):
		if err := r.saveLocked(); err != nil {
			return nil, err
		}
		return r, nil
	case err != nil:
		return nil, err
	}
	if len(data) == 0 {
		return r, nil
	}
	var entries []*Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("registry: corrupt file %s: %w", path, err)
	}
	for _, e := range entries {
		r.entries[e.Token] = e
	}
	return r, nil
}

// Add registers an activation code with its profile (no code-level expiry).
func (r *Registry) Add(token string, p *profile.Profile) error {
	return r.AddWithExpiry(token, p, time.Time{})
}

// AddWithExpiry registers an activation code with an optional code-level
// expiry (zero time = never expires at the code level; the profile's own
// ExpiresAt still applies at download time).
func (r *Registry) AddWithExpiry(token string, p *profile.Profile, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[token]; exists {
		return fmt.Errorf("registry: activation code %s already registered", token)
	}
	e := &Entry{
		Token:     token,
		Profile:   p,
		CreatedAt: time.Now().UTC(),
	}
	if !expiresAt.IsZero() {
		t := expiresAt.UTC()
		e.ExpiresAt = &t
	}
	r.entries[token] = e
	return r.saveLocked()
}

// Entry returns the raw entry for a token (for policy layers: single-use
// checks, expiry enforcement, auditing).
func (r *Registry) Entry(token string) (*Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[token]
	if !ok {
		return nil, fmt.Errorf("registry: unknown activation code")
	}
	return e, nil
}

// Resolve implements the SM-DP+ ProfileSource: it returns the profile for
// an activation code.
func (r *Registry) Resolve(token string) (*profile.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[token]
	if !ok {
		return nil, fmt.Errorf("registry: unknown activation code")
	}
	return e.Profile, nil
}

// ResolvePolicy enforces the code-level policy: single-use codes refuse a
// second download, and expired codes refuse everything. now is injected so
// tests can control time.
func (r *Registry) ResolvePolicy(token string, singleUse bool, now time.Time) (*profile.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[token]
	if !ok {
		return nil, fmt.Errorf("registry: unknown activation code")
	}
	if e.ExpiresAt != nil && now.After(*e.ExpiresAt) {
		return nil, ErrActivationCodeExpired
	}
	if singleUse && e.DownloadedAt != nil {
		return nil, ErrActivationCodeUsed
	}
	return e.Profile, nil
}

// MarkDownloaded records a successful download for an activation code.
// A registry policy layer (e.g. single-use codes) can build on this.
func (r *Registry) MarkDownloaded(token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[token]
	if !ok {
		return fmt.Errorf("registry: unknown activation code")
	}
	now := time.Now().UTC()
	e.DownloadedAt = &now
	return r.saveLocked()
}

// List returns all registered entries in insertion order.
func (r *Registry) List() []*Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Entry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	return out
}

// Revoke removes an activation code from the registry.
func (r *Registry) Revoke(token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[token]; !ok {
		return fmt.Errorf("registry: unknown activation code")
	}
	delete(r.entries, token)
	return r.saveLocked()
}

func (r *Registry) saveLocked() error {
	entries := make([]*Entry, 0, len(r.entries))
	for _, e := range r.entries {
		entries = append(entries, e)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, r.path); err != nil {
		return err
	}
	return os.Chmod(r.path, 0o600)
}
