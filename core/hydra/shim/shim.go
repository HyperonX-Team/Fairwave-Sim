// Package shim implements the Hydra per-packet sequencing header.
//
// Every packet multiplexed across a weave carries a fixed 24-byte header
// so the terminating anchor can reassemble a single logical flow from
// packets that arrived out of order over independent SIMs, modems, and
// boxes. On a real N3 link the fields ride in the GTP-U PDU Session
// Container / extension-header area; in the lab (and in tests) the header
// is simply prepended to the payload frame.
package shim

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Wire constants.
const (
	// Magic identifies a Hydra frame ("HY").
	Magic uint16 = 0x4859
	// Version is the current Hydra shim version.
	Version uint8 = 1
	// HeaderLen is the fixed encoded header size in bytes.
	HeaderLen = 24
)

// Header flags.
const (
	// FlagNone marks an ordinary data packet.
	FlagNone uint8 = 0
	// FlagProbe marks a capacity/latency probe packet.
	FlagProbe uint8 = 1 << 0
	// FlagLast marks the final packet of a weave.
	FlagLast uint8 = 1 << 1
)

// Decode errors.
var (
	// ErrShort is returned when a frame is smaller than HeaderLen.
	ErrShort = errors.New("shim: short frame")
	// ErrMagic is returned when a frame does not start with Magic.
	ErrMagic = errors.New("shim: bad magic")
	// ErrVersion is returned for an unsupported protocol version.
	ErrVersion = errors.New("shim: unsupported version")
)

// Header is the Hydra per-packet sequencing header.
//
// Seq is the weave-scoped monotonic sequence number used for reordering;
// TSNanos carries the sender's monotonic-ish timestamp for one-way delay
// estimation by the delay-inflation scheduler.
type Header struct {
	Version uint8
	Flags   uint8
	WeaveID uint32
	Seq     uint64
	TSNanos int64
}

// New builds a header for the given packet of a weave.
func New(weaveID uint32, seq uint64, tsNanos int64) Header {
	return Header{Version: Version, WeaveID: weaveID, Seq: seq, TSNanos: tsNanos}
}

// Encode renders the header into a fixed HeaderLen-byte buffer.
func (h Header) Encode() []byte {
	b := make([]byte, HeaderLen)
	binary.BigEndian.PutUint16(b[0:2], Magic)
	b[2] = h.Version
	b[3] = h.Flags
	binary.BigEndian.PutUint32(b[4:8], h.WeaveID)
	binary.BigEndian.PutUint64(b[8:16], h.Seq)
	binary.BigEndian.PutUint64(b[16:24], uint64(h.TSNanos))
	return b
}

// Decode parses a frame, returning the header and the payload slice. The
// payload aliases b's backing array, so callers that retain it must copy.
func Decode(b []byte) (Header, []byte, error) {
	if len(b) < HeaderLen {
		return Header{}, nil, fmt.Errorf("%w: have %d want >=%d", ErrShort, len(b), HeaderLen)
	}
	if binary.BigEndian.Uint16(b[0:2]) != Magic {
		return Header{}, nil, ErrMagic
	}
	h := Header{
		Version: b[2],
		Flags:   b[3],
		WeaveID: binary.BigEndian.Uint32(b[4:8]),
		Seq:     binary.BigEndian.Uint64(b[8:16]),
		TSNanos: int64(binary.BigEndian.Uint64(b[16:24])),
	}
	if h.Version != Version {
		return Header{}, nil, fmt.Errorf("%w: got %d", ErrVersion, h.Version)
	}
	return h, b[HeaderLen:], nil
}

// Frame prepends the encoded header to payload, returning a new buffer.
func Frame(h Header, payload []byte) []byte {
	out := make([]byte, 0, HeaderLen+len(payload))
	out = append(out, h.Encode()...)
	out = append(out, payload...)
	return out
}
