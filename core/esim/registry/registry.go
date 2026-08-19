// Package registry is the local file-backed SM-DP+ profile registry: it
// maps activation codes to the carrier profiles that may be downloaded.
// The file holds the full encrypted-payload material (including lab
// Milenage credentials) and is written with 0600 permissions - treat it
// like the SIM vault (docs/adr/0006-sim-vault.md). Writes are atomic
// (tmp + rename).
package registry

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
)

// Entry is one activation code in the registry. The sensitive carrier
// profile (which holds Milenage KI/OPc) is stored either directly in
// Profile (plaintext, lab default) or, when an encryption key is set, as
// ProfileCipher: base64(nonce || AES-GCM ciphertext || tag) of the
// marshaled profile. Metadata stays readable for audit/policy layers.
type Entry struct {
	Token          string           `json:"token"`
	Profile        *profile.Profile `json:"profile,omitempty"`
	ProfileCipher  string           `json:"profile_cipher,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	DownloadedAt   *time.Time       `json:"downloaded_at,omitempty"`
	ExpiresAt      *time.Time       `json:"expires_at,omitempty"` // code-level expiry (may be nil)
}

// ErrActivationCodeUsed means a single-use code was already downloaded.
var ErrActivationCodeUsed = errors.New("registry: activation code already used")

// ErrActivationCodeExpired means the code's validity window has passed.
var ErrActivationCodeExpired = errors.New("registry: activation code expired")

// Registry is a file-backed map of activation codes to profiles.
type Registry struct {
	mu      sync.Mutex
	path    string
	key     []byte // nil = plaintext entries (lab default)
	entries map[string]*Entry
}

// Open loads the registry from path, creating it if absent, in plaintext
// mode. The file is always chmod 0600.
func Open(path string) (*Registry, error) {
	return OpenWithKey(path, nil)
}

// OpenWithKey loads the registry with an optional encryption key. A nil key
// keeps plaintext entries (backward compatible). A key (16/24/32 bytes)
// encrypts each profile payload with AES-GCM at rest. An existing registry
// opened with the wrong key fails loudly rather than returning garbage.
func OpenWithKey(path string, key []byte) (*Registry, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	r := &Registry{path: path, key: key, entries: make(map[string]*Entry)}
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
		if err := r.decryptEntry(e); err != nil {
			return nil, fmt.Errorf("registry: %s: %w", path, err)
		}
		r.entries[e.Token] = e
	}
	return r, nil
}

// ValidateKey reports whether key is a usable AES key length (nil allowed).
func validateKey(key []byte) error {
	if key == nil {
		return nil
	}
	switch len(key) {
	case 16, 24, 32:
		return nil
	default:
		return fmt.Errorf("registry: key must be 16/24/32 bytes, got %d", len(key))
	}
}

// KeyFromPassphrase derives a 32-byte AES key from a passphrase (SHA-256).
// Use this to persist a human-held secret rather than a raw key.
func KeyFromPassphrase(passphrase string) []byte {
	h := sha256.Sum256([]byte(passphrase))
	return h[:]
}

func (r *Registry) encryptEntry(e *Entry) error {
	if r.key == nil || e.Profile == nil {
		return nil
	}
	plain, err := json.Marshal(e.Profile)
	if err != nil {
		return err
	}
	ct, err := aeadSeal(r.key, plain)
	if err != nil {
		return err
	}
	e.ProfileCipher = base64.StdEncoding.EncodeToString(ct)
	e.Profile = nil
	return nil
}

func (r *Registry) decryptEntry(e *Entry) error {
	if e.ProfileCipher == "" {
		return nil
	}
	ct, err := base64.StdEncoding.DecodeString(e.ProfileCipher)
	if err != nil {
		return fmt.Errorf("registry: corrupt profile_cipher (base64): %w", err)
	}
	plain, err := aeadOpen(r.key, ct)
	if err != nil {
		return fmt.Errorf("registry: cannot decrypt profile (wrong key?): %w", err)
	}
	p := &profile.Profile{}
	if err := json.Unmarshal(plain, p); err != nil {
		return fmt.Errorf("registry: decrypt ok but corrupt profile json: %w", err)
	}
	e.Profile = p
	return nil
}

// aeadSeal encrypts plain with AES-GCM under key, returning
// nonce || ciphertext || tag.
func aeadSeal(key, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

// aeadOpen decrypts nonce || ciphertext || tag under key.
func aeadOpen(key, sealed []byte) ([]byte, error) {
	if key == nil {
		return nil, errors.New("registry: encrypted entry but no key configured")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, errors.New("registry: ciphertext too short")
	}
	nonce := sealed[:gcm.NonceSize()]
	body := sealed[gcm.NonceSize():]
	return gcm.Open(nil, nonce, body, nil)
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
	if err := r.encryptEntry(e); err != nil {
		return err
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
	// Serialize a copy with profiles encrypted so the in-memory entries keep
	// their plaintext Profile (callers read r.entries directly).
	type encEntry struct {
		*Entry
		Profile *profile.Profile `json:"profile,omitempty"`
	}
	entries := make([]encEntry, 0, len(r.entries))
	for _, e := range r.entries {
		cp := *e
		if r.key != nil && e.Profile != nil {
			if err := r.encryptEntry(&cp); err != nil {
				return err
			}
		}
		entries = append(entries, encEntry{Entry: &cp})
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
