package reorder

import (
	"testing"
	"time"
)

func TestInOrder(t *testing.T) {
	b := New(8)
	for i := uint64(1); i <= 5; i++ {
		got := b.Push(i, []byte{byte(i)})
		if len(got) != 1 || got[0][0] != byte(i) {
			t.Fatalf("seq %d: got %v", i, got)
		}
	}
	if s := b.Stats(); s.Delivered != 5 || s.Reordered != 0 {
		t.Fatalf("stats: %+v", s)
	}
}

func TestOutOfOrderReassembly(t *testing.T) {
	b := New(8)
	// Arrive 1, 3, 2, 4 -> deliver 1, then 2 and 3 together, then 4.
	if got := b.Push(1, []byte("a")); len(got) != 1 {
		t.Fatalf("seq1: %v", got)
	}
	if got := b.Push(3, []byte("c")); got != nil {
		t.Fatalf("seq3 should buffer, got %v", got)
	}
	if b.Depth() != 1 {
		t.Fatalf("depth = %d", b.Depth())
	}
	got := b.Push(2, []byte("b"))
	if len(got) != 2 || string(got[0]) != "b" || string(got[1]) != "c" {
		t.Fatalf("drain after seq2: %v", got)
	}
	if got := b.Push(4, []byte("d")); len(got) != 1 {
		t.Fatalf("seq4: %v", got)
	}
	s := b.Stats()
	if s.Delivered != 4 || s.Reordered != 1 || s.MaxDepth != 2 {
		t.Fatalf("stats: %+v", s)
	}
}

// TestLateFirstPacketIsHeld is the regression test for the bug where the
// window aligned to the first packet seen and discarded the true first
// packet as a duplicate.
func TestLateFirstPacketIsHeld(t *testing.T) {
	b := New(8)
	if got := b.Push(2, []byte("b")); got != nil {
		t.Fatalf("seq2 should buffer, got %v", got)
	}
	if got := b.Push(3, []byte("c")); got != nil {
		t.Fatalf("seq3 should buffer, got %v", got)
	}
	got := b.Push(1, []byte("a"))
	if len(got) != 3 {
		t.Fatalf("late seq1 must release all three, got %d", len(got))
	}
	if string(got[0]) != "a" || string(got[1]) != "b" || string(got[2]) != "c" {
		t.Fatalf("order = %q", got)
	}
	if s := b.Stats(); s.Duplicates != 0 {
		t.Fatalf("late first packet counted as duplicate: %+v", s)
	}
}

func TestDuplicateAfterDelivery(t *testing.T) {
	b := New(8)
	b.Push(1, []byte("a"))
	b.Push(2, []byte("b"))
	if got := b.Push(1, []byte("a")); got != nil {
		t.Fatalf("late dup delivered: %v", got)
	}
	if s := b.Stats(); s.Duplicates != 1 || s.Delivered != 2 {
		t.Fatalf("stats: %+v", s)
	}
}

func TestDuplicateWhileBuffered(t *testing.T) {
	b := New(8)
	b.Push(1, []byte("a"))
	b.Push(3, []byte("c"))
	if got := b.Push(3, []byte("c-again")); got != nil {
		t.Fatalf("buffered dup delivered: %v", got)
	}
	if s := b.Stats(); s.Duplicates != 1 {
		t.Fatalf("stats: %+v", s)
	}
}

func TestTooFarDrop(t *testing.T) {
	b := New(4)
	b.Push(1, []byte("a"))
	if got := b.Push(1+4+1, []byte("x")); got != nil {
		t.Fatalf("beyond-window packet delivered: %v", got)
	}
	if s := b.Stats(); s.TooFar != 1 {
		t.Fatalf("stats: %+v", s)
	}
}

func TestRespectsStartSequence(t *testing.T) {
	b := NewAt(8, 100)
	if got := b.Push(100, []byte("a")); len(got) != 1 {
		t.Fatalf("start sequence not honoured: %v", got)
	}
	if got := b.Push(99, []byte("z")); got != nil {
		t.Fatalf("pre-start packet delivered: %v", got)
	}
	if s := b.Stats(); s.Duplicates != 1 {
		t.Fatalf("stats: %+v", s)
	}
}

func TestGapTimeoutSkipsLostPacket(t *testing.T) {
	b := New(8)
	now := time.Now()
	b.now = func() time.Time { return now }
	b.SetGapTimeout(100 * time.Millisecond)

	b.Push(1, []byte("a")) // delivered, next=2
	b.Push(3, []byte("c")) // buffered, gap at 2
	if b.Depth() != 1 {
		t.Fatalf("depth = %d", b.Depth())
	}
	// Sequence 2 never arrives; after the timeout the weave must resume.
	now = now.Add(200 * time.Millisecond)
	got := b.Push(4, []byte("d"))
	if len(got) != 2 || string(got[0]) != "c" || string(got[1]) != "d" {
		t.Fatalf("gap not skipped: %v", got)
	}
	if s := b.Stats(); s.Skipped != 1 {
		t.Fatalf("stats: %+v", s)
	}
}

func TestNoGapTimeoutByDefault(t *testing.T) {
	b := New(8)
	now := time.Now()
	b.now = func() time.Time { return now }
	b.Push(1, []byte("a"))
	b.Push(3, []byte("c"))
	now = now.Add(time.Hour)
	if got := b.Push(4, []byte("d")); got != nil {
		t.Fatalf("gap skipped without a configured timeout: %v", got)
	}
}

func TestResyncAfterSenderRestart(t *testing.T) {
	b := New(1 << 20)
	for i := uint64(1); i <= 100; i++ {
		b.Push(i, []byte{byte(i)})
	}
	// The sender restarts and begins again at 1. The first ResyncAfter-1
	// packets look like duplicates...
	for i := uint64(1); i < ResyncAfter; i++ {
		if got := b.Push(i, []byte{1}); got != nil {
			t.Fatalf("pre-resync packet %d delivered: %v", i, got)
		}
	}
	// ...then the buffer re-aligns and resumes delivery.
	got := b.Push(ResyncAfter, []byte("x"))
	if len(got) != 1 {
		t.Fatalf("resync did not resume delivery: %v", got)
	}
	if s := b.Stats(); s.Resyncs != 1 {
		t.Fatalf("stats: %+v", s)
	}
	// The stream continues normally after the resync.
	if got := b.Push(ResyncAfter+1, []byte("y")); len(got) != 1 {
		t.Fatalf("post-resync delivery broken: %v", got)
	}
}

func TestPayloadIsCopied(t *testing.T) {
	b := New(8)
	b.Push(1, []byte("a"))
	buf := []byte("mutate-me")
	b.Push(3, buf)
	copy(buf, []byte("XXXXX"))
	got := b.Push(2, []byte("b"))
	if len(got) != 2 || string(got[1]) != "mutate-me" {
		t.Fatalf("payload aliased caller buffer: %q", got)
	}
}
