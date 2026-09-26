// Package anchor defines where a reassembled Hydra weave leaves for
// transit. The anchor owns the real internet uplink; every thread in the
// weave tunnels inbound until its packets reach the anchor, which restores
// their original order and forwards them as one ordinary flow.
package anchor

import "sync"

// Egress is the terminal for in-order weave payloads.
type Egress interface {
	// Send delivers one reassembled payload, in the order it was produced.
	Send(payload []byte) error
	// Name identifies the egress in status output.
	Name() string
}

// Sink counts payloads without transmitting them. It is the default
// egress for the lab and for tests.
type Sink struct {
	mu     sync.Mutex
	frames uint64
	bytes  uint64
}

// NewSink creates an empty counting sink.
func NewSink() *Sink { return &Sink{} }

// Send counts the payload and discards it.
func (s *Sink) Send(payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames++
	s.bytes += uint64(len(payload))
	return nil
}

// Name implements Egress.
func (s *Sink) Name() string { return "sink" }

// Frames returns the number of payloads received.
func (s *Sink) Frames() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frames
}

// Bytes returns the number of payload bytes received.
func (s *Sink) Bytes() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bytes
}
