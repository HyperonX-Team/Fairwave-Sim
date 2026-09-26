package reorder

import "testing"

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

func TestAlignToFirstSequence(t *testing.T) {
	b := New(8)
	// A weave that starts at sequence 100 must still deliver.
	got := b.Push(100, []byte("a"))
	if len(got) != 1 {
		t.Fatalf("align failed: %v", got)
	}
	if b.Next() != 101 {
		t.Fatalf("next = %d", b.Next())
	}
}

func TestPayloadIsCopied(t *testing.T) {
	b := New(8)
	b.Push(1, []byte("a")) // align + deliver seq 1
	buf := []byte("mutate-me")
	b.Push(3, buf) // buffered as a copy
	copy(buf, []byte("XXXXX"))
	got := b.Push(2, []byte("b")) // drains 2 and 3
	if len(got) != 2 || string(got[1]) != "mutate-me" {
		t.Fatalf("payload aliased caller buffer: %q", got)
	}
}
