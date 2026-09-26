// Package reorder implements the Hydra bounded sliding-window reassembler.
//
// Packets striped across independent links arrive out of order. The buffer
// holds them until the missing earlier sequence numbers arrive, then
// releases every contiguous payload in order.
//
// Two properties keep the weave live on unreliable links:
//
//   - The window is bounded, so a packet further than Window ahead of the
//     next expected sequence is dropped rather than buffered forever.
//   - The window starts at a known sequence (NewAt), so a packet that
//     arrives late is held rather than mistaken for a duplicate. Aligning
//     to the first packet seen would silently discard the true first packet
//     whenever a later one won the race.
//
// A gap timeout additionally prevents a single lost packet from stalling
// the stream forever: when the expected sequence has not arrived within
// GapTimeout while later packets are waiting, the gap is skipped and the
// stream resumes. The skipped packet is lost - correct for the unreliable
// datagram semantics Hydra carries, where the layer above retransmits.
package reorder

import (
	"sync"
	"time"
)

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
	// Skipped counts sequence numbers abandoned by the gap timeout.
	Skipped uint64 `json:"skipped"`
	// Resyncs counts re-alignments after the far end restarted.
	Resyncs uint64 `json:"resyncs"`
	// Reordered counts packets that arrived ahead of the next expected
	// sequence (i.e. required buffering).
	Reordered uint64 `json:"reordered"`
	// MaxDepth is the high-water mark of buffered packets.
	MaxDepth uint64 `json:"max_depth"`
}

// Buffer is a bounded, concurrency-safe reorder window for one weave.
type Buffer struct {
	mu         sync.Mutex
	window     uint64
	next       uint64
	gapTimeout time.Duration
	gapSince   time.Time
	staleRun   uint64
	now        func() time.Time
	slots      map[uint64][]byte
	stats      Stats
}

// DefaultWindow is the reassembly window used when New is called with 0.
// It is sized for a gigabit weave across links with spread RTTs.
const DefaultWindow uint64 = 4096

// DefaultGapTimeout is how long a missing sequence is waited for before it
// is skipped. It is generous enough never to fire during ordinary
// reordering and short enough to bound the stall from a real loss.
const DefaultGapTimeout = time.Second

// ResyncAfter is how many consecutive stale packets (sequence numbers
// below the expected one, with nothing buffered) trigger a re-alignment to
// the sender's new sequence. It recovers a weave after the far end
// restarted and its sequence reset to 1.
const ResyncAfter = 64

// New creates a buffer whose first expected sequence is 1.
func New(window uint64) *Buffer { return NewAt(window, 1) }

// NewAt creates a buffer whose first expected sequence is start. A zero
// window selects DefaultWindow.
func NewAt(window, start uint64) *Buffer {
	if window == 0 {
		window = DefaultWindow
	}
	return &Buffer{
		window: window,
		next:   start,
		now:    time.Now,
		slots:  map[uint64][]byte{},
	}
}

// SetGapTimeout configures how long a missing sequence is waited for before
// it is skipped (0 disables gap recovery).
func (b *Buffer) SetGapTimeout(d time.Duration) {
	b.mu.Lock()
	b.gapTimeout = d
	b.mu.Unlock()
}

// Push inserts a packet and returns every payload that is now deliverable
// in order.
func (b *Buffer) Push(seq uint64, payload []byte) [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stats.Received++

	// Recover from a stalled gap before inserting: if the expected sequence
	// has not arrived within the timeout and later packets are buffered,
	// abandon it and resume from the lowest buffered sequence.
	if b.gapTimeout > 0 && len(b.slots) > 0 && !b.gapSince.IsZero() &&
		b.now().Sub(b.gapSince) > b.gapTimeout {
		if lowest := b.minBuffered(); lowest > b.next {
			b.stats.Skipped += lowest - b.next
			b.next = lowest
		}
		b.gapSince = time.Time{}
	}

	switch {
	case seq < b.next:
		// A stale packet. If nothing is buffered and this keeps happening,
		// the far end almost certainly restarted and reset its sequence;
		// re-align rather than discarding its whole stream. Otherwise it
		// is an ordinary duplicate.
		if len(b.slots) == 0 && b.staleRun+1 >= ResyncAfter {
			b.stats.Resyncs++
			b.next = seq
			b.staleRun = 0
		} else {
			if len(b.slots) == 0 {
				b.staleRun++
			}
			b.stats.Duplicates++
			return nil
		}
	case seq >= b.next+b.window:
		b.stats.TooFar++
		return nil
	}
	if _, dup := b.slots[seq]; dup {
		b.stats.Duplicates++
		return nil
	}
	if seq > b.next {
		b.stats.Reordered++
		b.staleRun = 0
		if b.gapSince.IsZero() {
			b.gapSince = b.now()
		}
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
	if len(out) > 0 {
		b.gapSince = time.Time{}
		b.staleRun = 0
	}
	return out
}

// minBuffered returns the lowest buffered sequence. Caller holds the lock.
func (b *Buffer) minBuffered() uint64 {
	var lowest uint64
	first := true
	for seq := range b.slots {
		if first || seq < lowest {
			lowest = seq
			first = false
		}
	}
	return lowest
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
