package fabric

import (
	"errors"
	"testing"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/bearer"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/shim"
)

func newEngine(t *testing.T, threads ...bearer.Thread) *Engine {
	t.Helper()
	e := NewEngine(nil)
	for _, th := range threads {
		if err := e.AddThread(th); err != nil {
			t.Fatalf("add thread %s: %v", th.ID, err)
		}
	}
	return e
}

func TestCreateWeaveAndValidation(t *testing.T) {
	e := newEngine(t,
		bearer.Thread{ID: "t1", Mbps: 300, Up: true},
		bearer.Thread{ID: "t2", Mbps: 200, Up: true},
	)
	w, err := e.CreateWeave("main", "hub-a", []string{"t1", "t2"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if w.Mode != DefaultMode || w.Anchor != "hub-a" || len(w.Threads) != 2 {
		t.Fatalf("weave = %+v", w)
	}
	if _, err := e.CreateWeave("main", "", nil); !errors.Is(err, ErrWeaveExists) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := e.CreateWeave("bad", "", []string{"ghost"}); err == nil {
		t.Fatal("unknown thread accepted")
	}
	if _, err := e.CreateWeave("empty", "", nil); !errors.Is(err, ErrNoThreads) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := e.CreateWeave("", "", []string{"t1"}); err == nil {
		t.Fatal("empty id accepted")
	}
}

func TestDefaultsAnchorToEgress(t *testing.T) {
	e := NewEngine(nil)
	_ = e.AddThread(bearer.Thread{ID: "t1", Mbps: 100, Up: true})
	w, err := e.CreateWeave("w", "", []string{"t1"})
	if err != nil {
		t.Fatal(err)
	}
	if w.Anchor != "sink" {
		t.Fatalf("anchor = %q want sink", w.Anchor)
	}
	if e.Status().Anchor != "sink" {
		t.Fatalf("status anchor = %q", e.Status().Anchor)
	}
}

func TestStripNoWeaveOrThreads(t *testing.T) {
	e := newEngine(t)
	if _, err := e.Strip("nope", []byte("x")); !errors.Is(err, ErrNoWeave) {
		t.Fatalf("want ErrNoWeave, got %v", err)
	}
	_ = e.AddThread(bearer.Thread{ID: "down", Mbps: 100, Up: false})
	if _, err := e.CreateWeave("w", "", []string{"down"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Strip("w", []byte("x")); !errors.Is(err, ErrNoThreads) {
		t.Fatalf("want ErrNoThreads, got %v", err)
	}
}

func TestStripBalancesTowardFasterThread(t *testing.T) {
	e := newEngine(t,
		bearer.Thread{ID: "fast", Mbps: 1000, Up: true},
		bearer.Thread{ID: "slow", Mbps: 10, Up: true},
	)
	if _, err := e.CreateWeave("w", "", []string{"fast", "slow"}); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for i := 0; i < 200; i++ {
		r, err := e.Strip("w", make([]byte, 1000))
		if err != nil {
			t.Fatal(err)
		}
		counts[r.ThreadID]++
	}
	if counts["fast"] <= counts["slow"]*5 {
		t.Fatalf("fast link underused: %+v", counts)
	}
}

func TestStripSequenceIsMonotonic(t *testing.T) {
	e := newEngine(t, bearer.Thread{ID: "t1", Mbps: 100, Up: true})
	_, _ = e.CreateWeave("w", "", []string{"t1"})
	var last uint64
	for i := 0; i < 10; i++ {
		r, err := e.Strip("w", []byte("p"))
		if err != nil {
			t.Fatal(err)
		}
		if r.Seq != last+1 {
			t.Fatalf("seq %d after %d", r.Seq, last)
		}
		last = r.Seq
	}
}

func TestFullReassemblyLoop(t *testing.T) {
	e := newEngine(t,
		bearer.Thread{ID: "a", Mbps: 300, Up: true},
		bearer.Thread{ID: "b", Mbps: 150, Up: true},
		bearer.Thread{ID: "c", Mbps: 100, Up: true},
	)
	_, _ = e.CreateWeave("w", "", []string{"a", "b", "c"})

	const packets = 50
	results := make([]StripResult, 0, packets)
	for i := 0; i < packets; i++ {
		payload := make([]byte, 32)
		payload[0] = byte(i)
		r, err := e.Strip("w", payload)
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, r)
	}

	// Deliver out of order: first, then evens, then odds.
	var delivered [][]byte
	feed := func(r StripResult) {
		got, err := e.Reassemble("w", r.Frame)
		if err != nil {
			t.Fatalf("reassemble seq %d: %v", r.Seq, err)
		}
		delivered = append(delivered, got...)
	}
	for i := 0; i < len(results); i += 2 {
		feed(results[i])
	}
	for i := 1; i < len(results); i += 2 {
		feed(results[i])
	}

	if len(delivered) != packets {
		t.Fatalf("delivered %d want %d", len(delivered), packets)
	}
	for i, p := range delivered {
		if len(p) != 32 || p[0] != byte(i) {
			t.Fatalf("delivered[%d] = %v (want first byte %d)", i, p, i)
		}
	}
}

func TestReassembleWeaveMismatch(t *testing.T) {
	e := newEngine(t, bearer.Thread{ID: "t1", Mbps: 100, Up: true})
	_, _ = e.CreateWeave("w", "", []string{"t1"})
	frame := shim.Frame(shim.New(0xABCD1234, 1, 0), []byte("x"))
	if _, err := e.Reassemble("w", frame); !errors.Is(err, ErrWeaveMismatch) {
		t.Fatalf("want ErrWeaveMismatch, got %v", err)
	}
}

func TestIngestToSink(t *testing.T) {
	sink := &countingSink{}
	e := NewEngine(sink)
	_ = e.AddThread(bearer.Thread{ID: "t1", Mbps: 100, Up: true})
	_, _ = e.CreateWeave("w", "", []string{"t1"})

	var frames [][]byte
	for i := 0; i < 5; i++ {
		r, _ := e.Strip("w", []byte{byte(i)})
		frames = append(frames, r.Frame)
	}
	// Ingest in reverse is not valid (aligns high); ingest in order.
	total := 0
	for _, f := range frames {
		n, err := e.Ingest("w", f)
		if err != nil {
			t.Fatal(err)
		}
		total += n
	}
	if total != 5 || sink.frames != 5 {
		t.Fatalf("ingested=%d sink=%d", total, sink.frames)
	}
}

func TestDeleteWeaveAndStats(t *testing.T) {
	e := newEngine(t, bearer.Thread{ID: "t1", Mbps: 300, Up: true})
	_, _ = e.CreateWeave("w", "", []string{"t1"})
	_, _ = e.Strip("w", []byte("hello"))

	stats, ok := e.WeaveStats("w")
	if !ok {
		t.Fatal("stats missing")
	}
	if stats.FramesSent != 1 || stats.AggregateMbps != 300 {
		t.Fatalf("stats = %+v", stats)
	}
	if _, ok := e.WeaveStats("nope"); ok {
		t.Fatal("stats for unknown weave")
	}
	if !e.DeleteWeave("w") || e.DeleteWeave("w") {
		t.Fatal("delete semantics wrong")
	}
	if _, ok := e.GetWeave("w"); ok {
		t.Fatal("weave survived delete")
	}
}

func TestStatusAggregates(t *testing.T) {
	e := newEngine(t,
		bearer.Thread{ID: "t1", Mbps: 300, Up: true},
		bearer.Thread{ID: "t2", Mbps: 300, Up: true},
		bearer.Thread{ID: "t3", Mbps: 500, Up: false},
	)
	_, _ = e.CreateWeave("w", "", []string{"t1", "t2"})
	_, _ = e.Strip("w", []byte("x"))
	_, _ = e.Strip("w", []byte("y"))

	st := e.Status()
	if st.Threads != 3 || st.ThreadsUp != 2 || st.Weaves != 1 {
		t.Fatalf("status = %+v", st)
	}
	if st.AggregateMbps != 600 {
		t.Fatalf("aggregate = %v want 600", st.AggregateMbps)
	}
	if st.FramesSent != 2 {
		t.Fatalf("frames = %d want 2", st.FramesSent)
	}
}

func TestBenchSpeedup(t *testing.T) {
	// Three 300 Mbps threads: the weave should report ~3x a single thread.
	e := newEngine(t,
		bearer.Thread{ID: "a", Mbps: 300, Up: true},
		bearer.Thread{ID: "b", Mbps: 300, Up: true},
		bearer.Thread{ID: "c", Mbps: 300, Up: true},
	)
	_, _ = e.CreateWeave("w", "", []string{"a", "b", "c"})

	res, err := e.Bench("w", 300, 1200)
	if err != nil {
		t.Fatal(err)
	}
	if res.Delivered != 300 {
		t.Fatalf("delivered %d want 300", res.Delivered)
	}
	if res.AggregateMbps != 900 || res.SingleThreadMbps != 300 {
		t.Fatalf("mbps: aggregate=%v single=%v", res.AggregateMbps, res.SingleThreadMbps)
	}
	if res.Speedup < 2.9 || res.Speedup > 3.1 {
		t.Fatalf("speedup = %v want ~3", res.Speedup)
	}
	if res.Reorder.Reordered == 0 {
		t.Fatal("bench did not exercise reordering")
	}
}

func TestBenchCountersSurface(t *testing.T) {
	e := newEngine(t, bearer.Thread{ID: "t1", Mbps: 300, Up: true})
	_, _ = e.CreateWeave("w", "", []string{"t1"})
	if _, err := e.Bench("w", 50, 100); err != nil {
		t.Fatal(err)
	}
	if st := e.Status(); st.Delivered != 50 {
		t.Fatalf("status delivered = %d want 50", st.Delivered)
	}
	ws, ok := e.WeaveStats("w")
	if !ok {
		t.Fatal("stats missing")
	}
	if ws.BenchRuns != 1 || ws.BenchDelivered != 50 {
		t.Fatalf("bench stats = %+v", ws)
	}
}

func TestBenchValidation(t *testing.T) {
	e := newEngine(t, bearer.Thread{ID: "t1", Mbps: 100, Up: true})
	_, _ = e.CreateWeave("w", "", []string{"t1"})
	if _, err := e.Bench("w", 0, 100); err == nil {
		t.Fatal("zero packets accepted")
	}
	if _, err := e.Bench("w", 10, 0); err == nil {
		t.Fatal("zero pktBytes accepted")
	}
	if _, err := e.Bench("ghost", 10, 100); !errors.Is(err, ErrNoWeave) {
		t.Fatalf("want ErrNoWeave, got %v", err)
	}
}

func TestCompleteDrainsQueue(t *testing.T) {
	e := newEngine(t, bearer.Thread{ID: "t1", Mbps: 100, Up: true})
	_, _ = e.CreateWeave("w", "", []string{"t1"})
	_, _ = e.Strip("w", make([]byte, 500))
	before, _ := e.Thread("t1")
	if before.QueueBytes == 0 {
		t.Fatal("strip did not queue bytes")
	}
	e.Complete("t1", before.QueueBytes)
	after, _ := e.Thread("t1")
	if after.QueueBytes != 0 {
		t.Fatalf("queue = %d want 0", after.QueueBytes)
	}
}

func TestDeterministicClock(t *testing.T) {
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	e := NewEngine(nil, WithClock(func() time.Time { return fixed }), WithAnchorName("edge"))
	_ = e.AddThread(bearer.Thread{ID: "t1", Mbps: 100, Up: true})
	w, _ := e.CreateWeave("w", "", []string{"t1"})
	if !w.CreatedAt.Equal(fixed) {
		t.Fatalf("created_at = %v", w.CreatedAt)
	}
	if e.Status().Anchor != "edge" {
		t.Fatalf("anchor = %q", e.Status().Anchor)
	}
}

func TestRemoveThread(t *testing.T) {
	e := newEngine(t, bearer.Thread{ID: "t1", Mbps: 100, Up: true})
	if !e.RemoveThread("t1") || e.RemoveThread("t1") {
		t.Fatal("remove semantics wrong")
	}
	if len(e.ListThreads()) != 0 {
		t.Fatal("thread survived removal")
	}
}

// countingSink is a test egress that counts frames.
type countingSink struct {
	frames int
}

func (c *countingSink) Send([]byte) error { c.frames++; return nil }
func (c *countingSink) Name() string      { return "counting" }
