// Package reorder implements the Hydra bounded sliding-window reassembler.
//
// Packets striped across independent links arrive out of order. The buffer
// holds out-of-order packets until the missing earlier sequence numbers
// arrive, then releases every contiguous payload in order. The window is
// deliberately bounded: a packet further than Window ahead of the next
// expected sequence is dropped rather than buffered forever, which caps
// memory and prevents unbounded head-of-line blocking on a dead link.
package reorder

import "sync"

// Stats is a snapshot of reassembler activity for one weave.
type Stats struct {
	// Received counts every packet pushed, including duplicates.
	Received uint64 `json:"received"`
	// Delivered counts packets released in order.
	Delivered uint64 `json:"delivered"`
	// Duplicates counts packets whose sequence was already delivered or
	// already buffered.
	Duplicates uint64 `json:"duplicates"`
	// TooFar counts packets dropped for landing beyond the window.
	TooFar uint64 `json:"too_far_drops"`
	// Reordered counts packets that arrived ahead of the next expected
	// sequence (i.e. required buffering).
	Reordered uint64 `json:"reordered"`
	// MaxDepth is the high-water mark of buffered packets.
	MaxDepth uint64 `json:"max_depth"`
}

// Buffer is a bounded, concurrency-safe reorder window for one weave.
type Buffer struct {
	mu      sync.Mutex
	window  uint64
	next    uint64
	aligned bool
	slots   map[uint64][]byte
	stats   Stats
}

// DefaultWindow is the reassembly window used when New is called with 0.
const DefaultWindow uint64 = 256

// New creates a buffer with the given window. A zero window selects
// DefaultWindow.
func New(window uint64) *Buffer {
	if window == 0 {
		window = DefaultWindow
	}
	return &Buffer{window: window, slots: map[uint64][]byte{}}
}

// Push inserts a packet and returns every payload that is now deliverable
// in order. The first packet seen aligns the window to its sequence, so a
// weave may start at any sequence number.
func (b *Buffer) Push(seq uint64, payload []byte) [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stats.Received++
	if !b.aligned {
		b.next = seq
		b.aligned = true
	}

	switch {
	case seq < b.next:
		// Already delivered (or dropped) at this sequence.
		b.stats.Duplicates++
		return nil
	case seq >= b.next+b.window:
		// Beyond the window: drop rather than buffer forever.
		b.stats.TooFar++
		return nil
	}

	if _, dup := b.slots[seq]; dup {
		b.stats.Duplicates++
		return nil
	}
	if seq > b.next {
		b.stats.Reordered++
	}

	cp := make([]byte, len(payload))
	copy(cp, payload)
	b.slots[seq] = cp
	if depth := uint64(len(b.slots)); depth > b.stats.MaxDepth {
		b.stats.MaxDepth = depth
	}

	var out [][]byte
	for {
		p, ok := b.slots[b.next]
		if !ok {
			break
		}
		delete(b.slots, b.next)
		out = append(out, p)
		b.stats.Delivered++
		b.next++
	}
	return out
}

// Next returns the next expected sequence number.
func (b *Buffer) Next() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.next
}

// Depth returns the number of currently buffered packets.
func (b *Buffer) Depth() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.slots)
}

// Stats returns a snapshot of the counters.
func (b *Buffer) Stats() Stats {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stats
}
