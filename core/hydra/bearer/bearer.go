// Package bearer manages the set of independent SIM/modem/path "threads"
// that a Hydra weave can stripe traffic across. A thread is the unit of
// capacity: one SIM in one modem on one box, reachable over one path.
package bearer

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/sched"
)

// ErrNotFound is returned when a thread id is unknown.
var ErrNotFound = errors.New("bearer: thread not found")

// Thread is one independent SIM/modem/path usable by a weave.
type Thread struct {
	ID         string    `json:"id"`
	Box        string    `json:"box,omitempty"`
	SIM        string    `json:"sim,omitempty"`
	Mbps       float64   `json:"mbps"`
	RTTms      float64   `json:"rtt_ms"`
	LossPct    float64   `json:"loss_pct"`
	QueueBytes uint64    `json:"queue_bytes"`
	Up         bool      `json:"up"`
	LastSeen   time.Time `json:"last_seen"`
}

// Candidate converts a thread into a scheduler candidate.
func (t Thread) Candidate() sched.Thread {
	return sched.Thread{
		ID:         t.ID,
		Mbps:       t.Mbps,
		RTTms:      t.RTTms,
		LossPct:    t.LossPct,
		QueueBytes: t.QueueBytes,
	}
}

// Registry is a concurrency-safe set of threads keyed by id.
type Registry struct {
	mu      sync.RWMutex
	threads map[string]*Thread
	now     func() time.Time
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{threads: map[string]*Thread{}, now: time.Now}
}

// Upsert validates and stores a thread. An empty LastSeen is stamped with
// the current time.
func (r *Registry) Upsert(t Thread) error {
	if t.ID == "" {
		return errors.New("bearer: thread id required")
	}
	if t.Mbps < 0 {
		return fmt.Errorf("bearer: thread %s: mbps must be >= 0", t.ID)
	}
	if t.RTTms < 0 {
		return fmt.Errorf("bearer: thread %s: rtt_ms must be >= 0", t.ID)
	}
	if t.LossPct < 0 || t.LossPct > 100 {
		return fmt.Errorf("bearer: thread %s: loss_pct must be in [0,100]", t.ID)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.LastSeen.IsZero() {
		t.LastSeen = r.now().UTC()
	}
	cp := t
	r.threads[t.ID] = &cp
	return nil
}

// Remove deletes a thread, reporting whether it existed.
func (r *Registry) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.threads[id]; !ok {
		return false
	}
	delete(r.threads, id)
	return true
}

// Get returns a copy of the thread with the given id.
func (r *Registry) Get(id string) (Thread, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.threads[id]
	if !ok {
		return Thread{}, false
	}
	return *t, true
}

// List returns all threads, sorted by id.
func (r *Registry) List() []Thread {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Thread, 0, len(r.threads))
	for _, t := range r.threads {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Up returns the threads currently marked up, sorted by id.
func (r *Registry) Up() []Thread {
	all := r.List()
	out := make([]Thread, 0, len(all))
	for _, t := range all {
		if t.Up {
			out = append(out, t)
		}
	}
	return out
}

// Candidates returns the scheduler candidates for the given thread ids,
// skipping ids that are unknown or down. Order follows ids.
func (r *Registry) Candidates(ids []string) []sched.Thread {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]sched.Thread, 0, len(ids))
	for _, id := range ids {
		t, ok := r.threads[id]
		if !ok || !t.Up {
			continue
		}
		out = append(out, t.Candidate())
	}
	return out
}

// AggregateMbps sums the capacity of every thread currently up. This is
// Hydra's headline number: the ceiling becomes the sum of the threads.
func (r *Registry) AggregateMbps() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var sum float64
	for _, t := range r.threads {
		if t.Up {
			sum += t.Mbps
		}
	}
	return sum
}

// AddQueue increases a thread's outstanding-byte counter.
func (r *Registry) AddQueue(id string, bytes uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.threads[id]; ok {
		t.QueueBytes += bytes
	}
}

// SubQueue decreases a thread's outstanding-byte counter, flooring at zero.
func (r *Registry) SubQueue(id string, bytes uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.threads[id]
	if !ok {
		return
	}
	if bytes >= t.QueueBytes {
		t.QueueBytes = 0
		return
	}
	t.QueueBytes -= bytes
}
