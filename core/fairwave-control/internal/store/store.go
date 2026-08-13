// Package store is a small, file-backed, crash-safe key-value store for
// control-plane state. Production deployments swap this for a real DB
// behind the same interface (ADR 0006).
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// Store persists nodes, SIMs, peers, sessions, health, the audit log, TX
// state, tokens, usage, alerts, and policy as JSON files.
type Store struct {
	mu       sync.RWMutex
	dir      string
	nodes    map[string]*api.Node
	sims     map[string]*api.SIM
	peers    map[string]*api.Peer
	sessions []*api.Session
	health   map[string]*api.NodeHealth
	audit    []api.AuditEntry
	tokens   map[string]*api.Token
	usage    map[string]*api.SimUsage
	alerts   []*api.Alert
	txArmed  bool
	policy   *api.Policy
}

// Open creates the store directory and loads any existing state.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	s := &Store{
		dir:     dir,
		nodes:   map[string]*api.Node{},
		sims:    map[string]*api.SIM{},
		peers:   map[string]*api.Peer{},
		health:  map[string]*api.NodeHealth{},
		tokens:  map[string]*api.Token{},
		usage:   map[string]*api.SimUsage{},
		policy:  &api.Policy{LocalBreakout: true, MaxUEs: 128, APNs: []string{"internet", "ims"}},
		txArmed: false,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	loadJSON(filepath.Join(s.dir, "nodes.json"), &s.nodes)
	loadJSON(filepath.Join(s.dir, "sims.json"), &s.sims)
	loadJSON(filepath.Join(s.dir, "peers.json"), &s.peers)
	loadJSON(filepath.Join(s.dir, "sessions.json"), &s.sessions)
	loadJSON(filepath.Join(s.dir, "health.json"), &s.health)
	loadJSON(filepath.Join(s.dir, "audit.json"), &s.audit)
	loadJSON(filepath.Join(s.dir, "tokens.json"), &s.tokens)
	loadJSON(filepath.Join(s.dir, "usage.json"), &s.usage)
	loadJSON(filepath.Join(s.dir, "alerts.json"), &s.alerts)
	loadJSON(filepath.Join(s.dir, "tx.json"), &s.txArmed)
	loadJSON(filepath.Join(s.dir, "policy.json"), &s.policy)
	return nil
}

func loadJSON(path string, v interface{}) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return
	}
	_ = json.Unmarshal(data, v) // corrupt file -> start fresh, log by caller
}

func (s *Store) save(name string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, name+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ---- nodes ----

func (s *Store) UpsertNode(n *api.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[n.ID] = n
	return s.save("nodes", s.nodes)
}

func (s *Store) GetNode(id string) (*api.Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.nodes[id]
	return n, ok
}

func (s *Store) ListNodes() []*api.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*api.Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		out = append(out, n)
	}
	return out
}

func (s *Store) DeleteNode(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.nodes, id)
	return s.save("nodes", s.nodes)
}

// ---- sims ----

func (s *Store) UpsertSIM(sim *api.SIM) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sims[sim.IMSI] = sim
	return s.save("sims", s.sims)
}

func (s *Store) GetSIM(imsi string) (*api.SIM, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sim, ok := s.sims[imsi]
	return sim, ok
}

func (s *Store) ListSIMs() []*api.SIM {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*api.SIM, 0, len(s.sims))
	for _, sim := range s.sims {
		out = append(out, sim)
	}
	return out
}

// ---- peers ----

func (s *Store) UpsertPeer(p *api.Peer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peers[p.ID] = p
	return s.save("peers", s.peers)
}

func (s *Store) GetPeer(id string) (*api.Peer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.peers[id]
	return p, ok
}

func (s *Store) ListPeers() []*api.Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*api.Peer, 0, len(s.peers))
	for _, p := range s.peers {
		out = append(out, p)
	}
	return out
}

func (s *Store) DeletePeer(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, id)
	return s.save("peers", s.peers)
}

// ---- sessions ----

// AddSession appends a session record.
func (s *Store) AddSession(sess *api.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = append(s.sessions, sess)
	return s.save("sessions", s.sessions)
}

// ReplaceSessions swaps the whole session table for a fresh snapshot (the
// collector reports live network state; stale entries must not linger).
func (s *Store) ReplaceSessions(sessions []api.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ptr := make([]*api.Session, len(sessions))
	for i := range sessions {
		ptr[i] = &sessions[i]
	}
	s.sessions = ptr
	return s.save("sessions", s.sessions)
}

func (s *Store) ListSessions() []*api.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*api.Session, len(s.sessions))
	copy(out, s.sessions)
	return out
}

// ---- node health ----

// UpsertHealth records the latest agent heartbeat for a node.
func (s *Store) UpsertHealth(h *api.NodeHealth) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health[h.NodeID] = h
	return s.save("health", s.health)
}

func (s *Store) GetHealth(nodeID string) (*api.NodeHealth, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.health[nodeID]
	return h, ok
}

func (s *Store) ListHealth() []*api.NodeHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*api.NodeHealth, 0, len(s.health))
	for _, h := range s.health {
		out = append(out, h)
	}
	return out
}

// ---- audit log ----

// AppendAudit records one operator action. The log is append-only: no
// update or delete path exists, so history can't be silently rewritten.
func (s *Store) AppendAudit(e api.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audit = append(s.audit, e)
	return s.save("audit", s.audit)
}

// ListAudit returns the audit trail, most recent first.
func (s *Store) ListAudit() []api.AuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]api.AuditEntry, len(s.audit))
	copy(out, s.audit)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// ---- tx state ----

func (s *Store) GetTxArmed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.txArmed
}

// SetTxArmed persists the TX armed flag so a restart cannot silently
// re-arm the radio.
func (s *Store) SetTxArmed(armed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.txArmed = armed
	return s.save("tx", s.txArmed)
}

// DataDir returns the backing directory (used by backup/restore).
func (s *Store) DataDir() string { return s.dir }

// ---- scoped tokens ----

// UpsertToken stores a token record (hash only).
func (s *Store) UpsertToken(t *api.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[t.ID] = t
	return s.save("tokens", s.tokens)
}

func (s *Store) GetToken(id string) (*api.Token, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tokens[id]
	return t, ok
}

// TokenByHash finds a non-revoked token by its SHA-256 hash.
func (s *Store) TokenByHash(hash string) (*api.Token, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tokens {
		if t.TokenHash == hash && !t.Revoked {
			return t, true
		}
	}
	return nil, false
}

func (s *Store) ListTokens() []*api.Token {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*api.Token, 0, len(s.tokens))
	for _, t := range s.tokens {
		out = append(out, t)
	}
	return out
}

// RevokeToken disables a token; the record stays for audit.
func (s *Store) RevokeToken(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tokens[id]
	if !ok {
		return fmt.Errorf("no such token %s", id)
	}
	t.Revoked = true
	return s.save("tokens", s.tokens)
}

// ---- usage ----

// UpsertUsage stores the accumulated usage for one SIM.
func (s *Store) UpsertUsage(u *api.SimUsage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usage[u.IMSI] = u
	return s.save("usage", s.usage)
}

func (s *Store) GetUsage(imsi string) (*api.SimUsage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.usage[imsi]
	return u, ok
}

func (s *Store) ListUsage() []*api.SimUsage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*api.SimUsage, 0, len(s.usage))
	for _, u := range s.usage {
		out = append(out, u)
	}
	return out
}

// ---- alerts ----

// AppendAlert records a fired or resolved alert.
func (s *Store) AppendAlert(a *api.Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, a)
	return s.save("alerts", s.alerts)
}

// ResolveAlert marks an alert resolved by id.
func (s *Store) ResolveAlert(id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.alerts {
		if a.ID == id {
			a.Resolved = true
			a.ResolvedAt = &at
			break
		}
	}
	return s.save("alerts", s.alerts)
}

// ListAlerts returns all alerts, newest first.
func (s *Store) ListAlerts() []*api.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*api.Alert, len(s.alerts))
	copy(out, s.alerts)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// ActiveAlerts returns alerts that are not yet resolved.
func (s *Store) ActiveAlerts() []*api.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*api.Alert
	for _, a := range s.alerts {
		if !a.Resolved {
			out = append(out, a)
		}
	}
	return out
}

// ---- policy ----

func (s *Store) GetPolicy() *api.Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := *s.policy
	return &p
}

func (s *Store) SetPolicy(p *api.Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = p
	return s.save("policy", s.policy)
}

// Uptime helper for status endpoint.
var startedAt = time.Now()

// Uptime returns seconds since the store opened.
func Uptime() int64 { return int64(time.Since(startedAt).Seconds()) }
