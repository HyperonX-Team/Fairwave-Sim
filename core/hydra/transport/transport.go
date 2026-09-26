// Package transport moves Hydra shim frames between weave endpoints over
// UDP.
//
// One Endpoint is one thread: an independent SIM, modem, or path. Each
// endpoint owns a UDP socket, an optional declared capacity (a token
// bucket paced at the operator-declared link rate), and an optional loss
// simulation hook for lab work. Endpoints learn their peer address from
// the first inbound datagram, so a weave becomes bidirectional as soon as
// traffic flows in either direction - no out-of-band address exchange.
package transport

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Errors returned by the transport.
var (
	// ErrClosed is returned by Send/Recv after Close.
	ErrClosed = errors.New("transport: endpoint closed")
	// ErrTooLarge is returned for a frame larger than a UDP datagram.
	ErrTooLarge = errors.New("transport: frame larger than a UDP datagram")
	// ErrNoPeer is returned by Send before a peer address is known.
	ErrNoPeer = errors.New("transport: no peer address")
)

// MaxDatagram is the largest frame the transport will send.
const MaxDatagram = 65507

// Stats is a snapshot of an endpoint's counters.
type Stats struct {
	ID         string  `json:"id"`
	Local      string  `json:"local"`
	Remote     string  `json:"remote,omitempty"`
	Mbps       float64 `json:"mbps"`
	SentFrames uint64  `json:"sent_frames"`
	SentBytes  uint64  `json:"sent_bytes"`
	RecvFrames uint64  `json:"recv_frames"`
	RecvBytes  uint64  `json:"recv_bytes"`
	Dropped    uint64  `json:"dropped"`
	LossPct    float64 `json:"loss_pct"`
}

// Endpoint is one thread's UDP transport.
type Endpoint struct {
	id   string
	conn *net.UDPConn

	mu       sync.Mutex
	remote   *net.UDPAddr
	tokens   float64
	last     time.Time
	rateBps  float64 // 0 = unlimited
	lossPct  float64
	rng      *rand.Rand
	closed   bool
	closedCh chan struct{}

	sentFrames atomic.Uint64
	sentBytes  atomic.Uint64
	recvFrames atomic.Uint64
	recvBytes  atomic.Uint64
	dropped    atomic.Uint64
}

// Dial binds a local UDP socket for a thread and optionally pins the peer
// address. A local address of "127.0.0.1:0" (or ":0") binds an ephemeral
// port. rateMbps declares the thread's capacity; 0 means unlimited.
func Dial(id, local, remote string, rateMbps float64) (*Endpoint, error) {
	laddr, err := net.ResolveUDPAddr("udp", local)
	if err != nil {
		return nil, fmt.Errorf("transport: resolve local %q: %w", local, err)
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, fmt.Errorf("transport: bind %q: %w", local, err)
	}
	// A weave aggregates bursts from many threads; a large socket buffer
	// keeps the kernel from dropping datagrams before the reader drains
	// them. Failures are non-fatal (the OS may cap the request).
	_ = conn.SetReadBuffer(4 << 20)
	_ = conn.SetWriteBuffer(4 << 20)
	e := &Endpoint{
		id:       id,
		conn:     conn,
		last:     time.Now(),
		closedCh: make(chan struct{}),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	if remote != "" {
		raddr, err := net.ResolveUDPAddr("udp", remote)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("transport: resolve remote %q: %w", remote, err)
		}
		e.remote = raddr
	}
	e.SetRate(rateMbps)
	return e, nil
}

// ID returns the thread id.
func (e *Endpoint) ID() string { return e.id }

// LocalAddr returns the bound local address.
func (e *Endpoint) LocalAddr() net.Addr { return e.conn.LocalAddr() }

// SetRate updates the declared capacity in Mbps (0 = unlimited).
func (e *Endpoint) SetRate(mbps float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if mbps <= 0 {
		e.rateBps = 0
		return
	}
	e.rateBps = mbps * 1e6 / 8
}

// SetLoss sets a synthetic drop probability in percent (lab hook).
func (e *Endpoint) SetLoss(pct float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lossPct = pct
}

// SetRemote pins the peer address.
func (e *Endpoint) SetRemote(addr string) error {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("transport: resolve remote %q: %w", addr, err)
	}
	e.mu.Lock()
	e.remote = raddr
	e.mu.Unlock()
	return nil
}

// Peer returns the current peer address, or "" if not yet known.
func (e *Endpoint) Peer() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.remote == nil {
		return ""
	}
	return e.remote.String()
}

// Send transmits one frame, blocking until the thread's capacity allows
// it. A synthetic loss roll may drop the frame; that is counted, not an
// error. Send is safe for concurrent use.
func (e *Endpoint) Send(frame []byte) error {
	if len(frame) > MaxDatagram {
		return fmt.Errorf("%w: %d bytes", ErrTooLarge, len(frame))
	}
	if err := e.wait(len(frame)); err != nil {
		return err
	}
	e.mu.Lock()
	if e.lossPct > 0 && e.rng.Float64()*100 < e.lossPct {
		e.mu.Unlock()
		e.dropped.Add(1)
		return nil
	}
	raddr := e.remote
	e.mu.Unlock()
	if raddr == nil {
		return ErrNoPeer
	}
	if _, err := e.conn.WriteToUDP(frame, raddr); err != nil {
		return err
	}
	e.sentFrames.Add(1)
	e.sentBytes.Add(uint64(len(frame)))
	return nil
}

// wait blocks until the token bucket has room for n bytes.
func (e *Endpoint) wait(n int) error {
	for {
		e.mu.Lock()
		if e.closed {
			e.mu.Unlock()
			return ErrClosed
		}
		rate := e.rateBps
		if rate <= 0 {
			e.mu.Unlock()
			return nil
		}
		now := time.Now()
		e.tokens += now.Sub(e.last).Seconds() * rate
		e.last = now
		if e.tokens > rate { // cap the bucket at one second of capacity
			e.tokens = rate
		}
		if e.tokens >= float64(n) {
			e.tokens -= float64(n)
			e.mu.Unlock()
			return nil
		}
		need := (float64(n) - e.tokens) / rate
		e.mu.Unlock()
		d := time.Duration(need * float64(time.Second))
		if d < time.Millisecond {
			d = time.Millisecond
		}
		select {
		case <-e.closedCh:
			return ErrClosed
		case <-time.After(d):
		}
	}
}

// Recv reads one frame into buf, returning its length and source address.
// The peer address is learned on the first datagram when unset.
func (e *Endpoint) Recv(buf []byte) (int, *net.UDPAddr, error) {
	n, addr, err := e.conn.ReadFromUDP(buf)
	if err != nil {
		e.mu.Lock()
		closed := e.closed
		e.mu.Unlock()
		if closed {
			return 0, nil, ErrClosed
		}
		return 0, nil, err
	}
	if addr != nil {
		e.mu.Lock()
		if e.remote == nil {
			e.remote = addr
		}
		e.mu.Unlock()
	}
	e.recvFrames.Add(1)
	e.recvBytes.Add(uint64(n))
	return n, addr, nil
}

// Stats returns a snapshot of the endpoint counters.
func (e *Endpoint) Stats() Stats {
	e.mu.Lock()
	mbps := e.rateBps * 8 / 1e6
	loss := e.lossPct
	remote := ""
	if e.remote != nil {
		remote = e.remote.String()
	}
	e.mu.Unlock()
	return Stats{
		ID:         e.id,
		Local:      e.conn.LocalAddr().String(),
		Remote:     remote,
		Mbps:       mbps,
		SentFrames: e.sentFrames.Load(),
		SentBytes:  e.sentBytes.Load(),
		RecvFrames: e.recvFrames.Load(),
		RecvBytes:  e.recvBytes.Load(),
		Dropped:    e.dropped.Load(),
		LossPct:    loss,
	}
}

// Close shuts the endpoint down, unblocking any pending Send or Recv.
func (e *Endpoint) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	close(e.closedCh)
	e.mu.Unlock()
	return e.conn.Close()
}
