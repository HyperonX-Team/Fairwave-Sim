// Package registry is the local file-backed SM-DP+ profile registry: it
// maps activation codes to the carrier profiles that may be downloaded.
// The file holds the full encrypted-payload material (including lab
// Milenage credentials) and is written with 0600 permissions - treat it
// like the SIM vault (docs/adr/0006-sim-vault.md). Writes are atomic
// (tmp + rename).
package registry

import (
	"encoding/json"
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
}

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

// Add registers an activation code with its profile.
func (r *Registry) Add(token string, p *profile.Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[token]; exists {
		return fmt.Errorf("registry: activation code %s already registered", token)
	}
	r.entries[token] = &Entry{
		Token:     token,
		Profile:   p,
		CreatedAt: time.Now().UTC(),
	}
	return r.saveLocked()
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
