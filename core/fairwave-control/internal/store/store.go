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

// Store persists nodes, SIMs, peers, sessions, and policy as JSON files.
type Store struct {
	mu       sync.RWMutex
	dir      string
	nodes    map[string]*api.Node
	sims     map[string]*api.SIM
	peers    map[string]*api.Peer
	sessions []*api.Session
	policy   *api.Policy
}

// Open creates the store directory and loads any existing state.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	s := &Store{
		dir:    dir,
		nodes:  map[string]*api.Node{},
		sims:   map[string]*api.SIM{},
		peers:  map[string]*api.Peer{},
		policy: &api.Policy{LocalBreakout: true, MaxUEs: 128, APNs: []string{"internet", "ims"}},
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

func (s *Store) AddSession(sess *api.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = append(s.sessions, sess)
	return s.save("sessions", s.sessions)
}

func (s *Store) ListSessions() []*api.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*api.Session, len(s.sessions))
	copy(out, s.sessions)
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
