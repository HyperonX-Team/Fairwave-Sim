// Package fabric is the Hydra weave engine. A weave binds a set of
// bearer threads (independent SIMs, modems, and boxes) to one anchor, then
// stripes a single logical flow across them. Outbound packets are handed
// to the delay-inflation scheduler; inbound packets are reassembled in
// order and egressed at the anchor.
//
// The engine is transport-agnostic. In the lab it is driven directly by
// the control plane and the ZMQ bench; on real hardware the same calls sit
// behind the GTP-U datapath.
package fabric

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/anchor"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/bearer"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/reorder"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/sched"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/shim"
)

// Engine errors.
var (
	// ErrNoWeave is returned for an unknown weave id.
	ErrNoWeave = errors.New("fabric: weave not found")
	// ErrWeaveExists is returned when creating a duplicate weave id.
	ErrWeaveExists = errors.New("fabric: weave already exists")
	// ErrNoThreads is returned when a weave has no usable threads.
	ErrNoThreads = errors.New("fabric: no usable threads")
	// ErrWeaveMismatch is returned when a frame's weave id does not match.
	ErrWeaveMismatch = errors.New("fabric: frame belongs to a different weave")
)

// DefaultMode is the only scheduling mode shipped today. The field exists
// so alternate policies can be recorded per weave without a schema change.
const DefaultMode = "delay-inflation"

// Weave is a set of threads bound to one anchor.
type Weave struct {
	ID        string    `json:"id"`
	Anchor    string    `json:"anchor"`
	Threads   []string  `json:"threads"`
	Mode      string    `json:"mode"`
	CreatedAt time.Time `json:"created_at"`
}

// StripResult is the assignment of one outbound packet to a thread.
type StripResult struct {
	WeaveID  string `json:"weave_id"`
	ThreadID string `json:"thread_id"`
	Seq      uint64 `json:"seq"`
	FrameLen int    `json:"frame_len"`
	// Frame is the shim-encoded packet. It is deliberately not serialized.
	Frame []byte `json:"-"`
}

// WeaveStats combines a weave's definition with its live counters.
type WeaveStats struct {
	Weave
	FramesSent     uint64        `json:"frames_sent"`
	BytesSent      uint64        `json:"bytes_sent"`
	AggregateMbps  float64       `json:"aggregate_mbps"`
	Reorder        reorder.Stats `json:"reorder"`
	BenchRuns      uint64        `json:"bench_runs"`
	BenchDelivered uint64        `json:"bench_delivered"`
}

// Status is the engine-wide summary surfaced at /v1/hydra/status.
type Status struct {
	Threads       int     `json:"threads"`
	ThreadsUp     int     `json:"threads_up"`
	Weaves        int     `json:"weaves"`
	AggregateMbps float64 `json:"aggregate_mbps"`
	FramesSent    uint64  `json:"frames_sent"`
	Delivered     uint64  `json:"delivered"`
	Anchor        string  `json:"anchor"`
}

// BenchResult is the output of a lab bench run: the proof that a weave's
// ceiling is the sum of its threads rather than any single one.
type BenchResult struct {
	WeaveID          string        `json:"weave_id"`
	Packets          int           `json:"packets"`
	Bytes            uint64        `json:"bytes"`
	Delivered        int           `json:"delivered"`
	AggregateMbps    float64       `json:"aggregate_mbps"`
	SingleThreadMbps float64       `json:"single_thread_mbps"`
	Speedup          float64       `json:"speedup"`
	Reorder          reorder.Stats `json:"reorder"`
	ElapsedMs        float64       `json:"elapsed_ms"`
}

type weaveState struct {
	w              Weave
	rb             *reorder.Buffer
	sendMu         sync.Mutex // serializes push+egress so payloads leave in order
	seq            uint64
	framesSent     uint64
	bytesSent      uint64
	benchRuns      uint64
	benchDelivered uint64
}

// Option configures the engine.
type Option func(*Engine)

// WithAnchorName records the anchor identity reported in status output.
func WithAnchorName(name string) Option {
	return func(e *Engine) { e.anchorName = name }
}

// WithClock injects a clock (used by tests for deterministic timestamps).
func WithClock(now func() time.Time) Option {
	return func(e *Engine) {
		if now != nil {
			e.now = now
		}
	}
}

// Engine owns every thread and weave in the node.
type Engine struct {
	mu         sync.Mutex
	reg        *bearer.Registry
	weaves     map[string]*weaveState
	egress     anchor.Egress
	anchorName string
	now        func() time.Time
}

// NewEngine creates an engine. A nil egress is replaced with a counting
// sink, so the engine is always usable.
func NewEngine(egress anchor.Egress, opts ...Option) *Engine {
	if egress == nil {
		egress = anchor.NewSink()
	}
	e := &Engine{
		reg:    bearer.NewRegistry(),
		weaves: map[string]*weaveState{},
		egress: egress,
		now:    time.Now,
	}
	for _, o := range opts {
		o(e)
	}
	if e.anchorName == "" {
		e.anchorName = egress.Name()
	}
	return e
}

// Registry exposes the underlying thread registry.
func (e *Engine) Registry() *bearer.Registry { return e.reg }

// Egress exposes the configured egress.
func (e *Engine) Egress() anchor.Egress { return e.egress }

// ---- threads ----

// AddThread validates and stores a thread.
func (e *Engine) AddThread(t bearer.Thread) error { return e.reg.Upsert(t) }

// RemoveThread deletes a thread, reporting whether it existed. Weaves that
// referenced it simply lose that thread from scheduling.
func (e *Engine) RemoveThread(id string) bool { return e.reg.Remove(id) }

// ListThreads returns every thread, sorted by id.
func (e *Engine) ListThreads() []bearer.Thread { return e.reg.List() }

// Thread returns one thread by id.
func (e *Engine) Thread(id string) (bearer.Thread, bool) { return e.reg.Get(id) }

// Complete marks bytes as drained from a thread's outstanding queue. The
// datapath calls it when a transmission finishes.
func (e *Engine) Complete(threadID string, bytes uint64) { e.reg.SubQueue(threadID, bytes) }

// ---- weaves ----

// CreateWeave binds the given threads to an anchor. An empty anchor name
// uses the engine's configured anchor.
func (e *Engine) CreateWeave(id, anchorName string, threadIDs []string) (Weave, error) {
	if id == "" {
		return Weave{}, errors.New("fabric: weave id required")
	}
	if anchorName == "" {
		anchorName = e.anchorName
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.weaves[id]; ok {
		return Weave{}, ErrWeaveExists
	}
	if err := e.validateThreads(threadIDs); err != nil {
		return Weave{}, err
	}
	w := Weave{
		ID:        id,
		Anchor:    anchorName,
		Threads:   append([]string(nil), threadIDs...),
		Mode:      DefaultMode,
		CreatedAt: e.now().UTC(),
	}
	e.weaves[id] = newWeaveState(w)
	return w, nil
}

// newWeaveState builds a weave's state with a reassembler that starts at
// sequence 1 and recovers from a lost packet after the default gap timeout.
func newWeaveState(w Weave) *weaveState {
	rb := reorder.NewAt(0, 1)
	rb.SetGapTimeout(reorder.DefaultGapTimeout)
	return &weaveState{w: w, rb: rb}
}

// validateThreads requires at least one known thread. Caller holds e.mu.
func (e *Engine) validateThreads(ids []string) error {
	if len(ids) == 0 {
		return ErrNoThreads
	}
	for _, id := range ids {
		if _, ok := e.reg.Get(id); !ok {
			return fmt.Errorf("fabric: unknown thread %q", id)
		}
	}
	return nil
}

// GetWeave returns a weave definition.
func (e *Engine) GetWeave(id string) (Weave, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	ws, ok := e.weaves[id]
	if !ok {
		return Weave{}, false
	}
	return ws.w, true
}

// ListWeaves returns every weave, sorted by id.
func (e *Engine) ListWeaves() []Weave {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Weave, 0, len(e.weaves))
	for _, ws := range e.weaves {
		out = append(out, ws.w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// DeleteWeave removes a weave, reporting whether it existed.
func (e *Engine) DeleteWeave(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.weaves[id]; !ok {
		return false
	}
	delete(e.weaves, id)
	return true
}

// ---- data path ----

// Strip assigns one outbound payload to a thread and returns the framed
// packet to transmit over that thread.
func (e *Engine) Strip(weaveID string, payload []byte) (StripResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	ws, ok := e.weaves[weaveID]
	if !ok {
		return StripResult{}, ErrNoWeave
	}
	cands := e.reg.Candidates(ws.w.Threads)
	if len(cands) == 0 {
		return StripResult{}, ErrNoThreads
	}
	ws.seq++
	seq := ws.seq
	t, err := sched.Pick(cands, uint64(len(payload)))
	if err != nil {
		return StripResult{}, err
	}
	frame := shim.Frame(shim.New(weaveHash(weaveID), seq, e.now().UnixNano()), payload)
	e.reg.AddQueue(t.ID, uint64(len(frame)))
	ws.framesSent++
	ws.bytesSent += uint64(len(frame))
	return StripResult{
		WeaveID:  weaveID,
		ThreadID: t.ID,
		Seq:      seq,
		FrameLen: len(frame),
		Frame:    frame,
	}, nil
}

// Reassemble decodes one inbound frame and returns any payloads that are
// now deliverable in order.
func (e *Engine) Reassemble(weaveID string, frame []byte) ([][]byte, error) {
	h, payload, err := shim.Decode(frame)
	if err != nil {
		return nil, err
	}
	if h.WeaveID != weaveHash(weaveID) {
		return nil, ErrWeaveMismatch
	}
	e.mu.Lock()
	ws, ok := e.weaves[weaveID]
	e.mu.Unlock()
	if !ok {
		return nil, ErrNoWeave
	}
	return ws.rb.Push(h.Seq, payload), nil
}

// Ingest reassembles an inbound frame and forwards every now-contiguous
// payload to the weave's anchor egress, in order. Thread readers run
// concurrently, so push and egress are serialized per weave: without that,
// two readers could deliver two contiguous payloads and race to send them,
// letting the egress observe them out of order. It returns the number
// delivered.
func (e *Engine) Ingest(weaveID string, frame []byte) (int, error) {
	h, payload, err := shim.Decode(frame)
	if err != nil {
		return 0, err
	}
	if h.WeaveID != weaveHash(weaveID) {
		return 0, ErrWeaveMismatch
	}
	e.mu.Lock()
	ws, ok := e.weaves[weaveID]
	e.mu.Unlock()
	if !ok {
		return 0, ErrNoWeave
	}
	ws.sendMu.Lock()
	defer ws.sendMu.Unlock()
	payloads := ws.rb.Push(h.Seq, payload)
	for _, p := range payloads {
		if err := e.egress.Send(p); err != nil {
			return 0, err
		}
	}
	return len(payloads), nil
}

// ---- stats ----

// WeaveStats returns a weave's definition plus its live counters.
func (e *Engine) WeaveStats(id string) (WeaveStats, bool) {
	e.mu.Lock()
	ws, ok := e.weaves[id]
	if !ok {
		e.mu.Unlock()
		return WeaveStats{}, false
	}
	stats := WeaveStats{
		Weave:          ws.w,
		FramesSent:     ws.framesSent,
		BytesSent:      ws.bytesSent,
		Reorder:        ws.rb.Stats(),
		BenchRuns:      ws.benchRuns,
		BenchDelivered: ws.benchDelivered,
	}
	e.mu.Unlock()
	stats.AggregateMbps = e.reg.AggregateMbps()
	return stats, true
}

// SetThreadHealth updates a thread's measured capability and liveness. It
// is how a running data-plane node feeds live link health into the
// scheduler: RTT and loss refine the projected completion time, and a
// down thread is skipped entirely. A non-positive mbps leaves the declared
// capacity untouched; a zero rttMs/lossPct on a healthy update is ignored
// when the thread is being marked down.
func (e *Engine) SetThreadHealth(id string, mbps, rttMs, lossPct float64, up bool) bool {
	return e.reg.Update(id, func(t *bearer.Thread) {
		if mbps > 0 {
			t.Mbps = mbps
		}
		t.RTTms = rttMs
		t.LossPct = lossPct
		t.Up = up
		t.LastSeen = e.now().UTC()
	})
}

// Restore loads persisted threads and weaves into the engine at startup.
// Threads are loaded first so weaves find them. Existing entries are kept
// (a duplicate weave id is ignored) so a restore never clobbers live state.
func (e *Engine) Restore(threads []bearer.Thread, weaves []Weave) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, t := range threads {
		if err := e.reg.Upsert(t); err != nil {
			return err
		}
	}
	for _, w := range weaves {
		if w.ID == "" {
			continue
		}
		if _, ok := e.weaves[w.ID]; ok {
			continue
		}
		if w.Mode == "" {
			w.Mode = DefaultMode
		}
		e.weaves[w.ID] = newWeaveState(w)
	}
	return nil
}

// WeaveHash is the 32-bit weave id carried in the shim header. The data
// plane needs it to build probe frames for a weave.
func WeaveHash(id string) uint32 { return weaveHash(id) }

// Status returns the engine-wide summary.
func (e *Engine) Status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	var frames uint64
	var delivered uint64
	for _, ws := range e.weaves {
		frames += ws.framesSent
		delivered += ws.rb.Stats().Delivered
	}
	up := e.reg.Up()
	return Status{
		Threads:       len(e.reg.List()),
		ThreadsUp:     len(up),
		Weaves:        len(e.weaves),
		AggregateMbps: e.reg.AggregateMbps(),
		FramesSent:    frames,
		Delivered:     delivered,
		Anchor:        e.anchorName,
	}
}

// Bench stripes packets across a weave and reassembles them from a
// deliberately reordered delivery sequence, proving the reorder path (and
// reporting the aggregate-versus-single-thread speedup). It is the lab's
// substitute for real radio throughput.
func (e *Engine) Bench(weaveID string, packets, pktBytes int) (BenchResult, error) {
	if packets <= 0 || pktBytes <= 0 {
		return BenchResult{}, errors.New("fabric: packets and pkt_bytes must be positive")
	}
	e.mu.Lock()
	ws, ok := e.weaves[weaveID]
	if !ok {
		e.mu.Unlock()
		return BenchResult{}, ErrNoWeave
	}
	threadIDs := append([]string(nil), ws.w.Threads...)
	e.mu.Unlock()

	cands := e.reg.Candidates(threadIDs)
	if len(cands) == 0 {
		return BenchResult{}, ErrNoThreads
	}

	results := make([]StripResult, 0, packets)
	var bytes uint64
	for i := 0; i < packets; i++ {
		payload := make([]byte, pktBytes)
		for j := range payload {
			payload[j] = byte(i)
		}
		r, err := e.Strip(weaveID, payload)
		if err != nil {
			return BenchResult{}, err
		}
		results = append(results, r)
		bytes += uint64(r.FrameLen)
	}

	// Feed the weave's own reassembler in an interleaved order (first, then
	// evens, then odds) to exercise reassembly hard. Using the weave's
	// buffer - not a throwaway one - keeps the sender's sequence and the
	// receiver's expectation in lockstep, so a bench never desynchronises
	// live traffic on the same weave. sendMu serialises against Ingest.
	ws.sendMu.Lock()
	defer ws.sendMu.Unlock()
	start := e.now()
	delivered := 0
	feed := func(r StripResult) error {
		_, payload, err := shim.Decode(r.Frame)
		if err != nil {
			return err
		}
		delivered += len(ws.rb.Push(r.Seq, payload))
		return nil
	}
	for i := 0; i < len(results); i += 2 {
		if err := feed(results[i]); err != nil {
			return BenchResult{}, err
		}
	}
	for i := 1; i < len(results); i += 2 {
		if err := feed(results[i]); err != nil {
			return BenchResult{}, err
		}
	}
	elapsed := e.now().Sub(start)
	ws.benchRuns++
	ws.benchDelivered += uint64(delivered)
	stats := ws.rb.Stats()

	aggregate := e.reg.AggregateMbps()
	single := 0.0
	for _, c := range cands {
		if c.Mbps > single {
			single = c.Mbps
		}
	}
	speedup := 1.0
	if single > 0 {
		speedup = aggregate / single
	}

	return BenchResult{
		WeaveID:          weaveID,
		Packets:          packets,
		Bytes:            bytes,
		Delivered:        delivered,
		AggregateMbps:    aggregate,
		SingleThreadMbps: single,
		Speedup:          speedup,
		Reorder:          stats,
		ElapsedMs:        float64(elapsed.Microseconds()) / 1000,
	}, nil
}

// weaveHash derives the 32-bit weave id carried in the shim header.
func weaveHash(id string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return h.Sum32()
}
