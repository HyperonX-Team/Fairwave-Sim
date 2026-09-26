// Package node runs a Hydra data plane.
//
// A node stripes payloads arriving on its ingress socket across transport
// threads, and reassembles frames arriving on those threads, delivering
// ordered payloads to its egress target. It is the runtime that turns the
// fabric engine into a running service: it owns the sockets, the per-thread
// writers paced at each link's declared capacity, the reassembly readers,
// and the liveness prober that marks dead links down and live ones up.
//
// Two nodes configured with mirrored thread sockets (an edge and an anchor)
// form a bidirectional weave. Payload boundaries are preserved end to end,
// so any datagram protocol - including the UPF's forwarded UE traffic -
// rides across the weave unchanged.
package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/bearer"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/fabric"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/shim"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/transport"
)

// Config configures a Hydra data-plane node.
type Config struct {
	// ID names this node in logs and stats.
	ID string
	// WeaveID is the weave this node serves (defaults to "default").
	WeaveID string
	// Anchor labels the egress this node forwards to.
	Anchor string
	// Ingress is the local UDP address receiving payloads to stripe.
	// Empty disables the ingress half (a pure anchor).
	Ingress string
	// Egress is the remote UDP address receiving reassembled payloads.
	// Empty discards them (a pure edge, or a test sink).
	Egress string
	// Threads are this node's independent SIM/modem/path endpoints.
	Threads []ThreadConfig
	// ProbeInterval is how often liveness probes are sent. 0 disables
	// probing (threads then stay in whatever state they were declared).
	ProbeInterval time.Duration
	// ProbeTimeout is how long an unanswered probe waits before it counts
	// as a miss (defaults to 3x ProbeInterval).
	ProbeTimeout time.Duration
	// ProbeFailures is how many consecutive misses mark a thread down
	// (defaults to 3).
	ProbeFailures int
	// QueueFrames caps in-flight frames per thread before tail-drop
	// (defaults to 1024).
	QueueFrames int
	// Logf receives diagnostics; nil discards them.
	Logf func(format string, args ...any)
}

// ThreadConfig declares one thread's endpoints and declared capacity.
type ThreadConfig struct {
	// ID names the thread; it must match on both ends of a weave.
	ID string
	// Local is the UDP address to bind (":0" for ephemeral).
	Local string
	// Remote is the peer UDP address. Empty means "learn on first packet".
	Remote string
	// Mbps is the operator-declared capacity of the underlying link.
	Mbps float64
}

// ThreadHealth is one thread's live liveness view.
type ThreadHealth struct {
	ID     string  `json:"id"`
	RTTms  float64 `json:"rtt_ms"`
	Misses int     `json:"misses"`
	Up     bool    `json:"up"`
	Mbps   float64 `json:"mbps"`
}

// Stats is a snapshot of node activity.
type Stats struct {
	NodeID        string            `json:"node_id"`
	WeaveID       string            `json:"weave_id"`
	Anchor        string            `json:"anchor"`
	IngressFrames uint64            `json:"ingress_frames"`
	IngressBytes  uint64            `json:"ingress_bytes"`
	EgressFrames  uint64            `json:"egress_frames"`
	EgressBytes   uint64            `json:"egress_bytes"`
	Dropped       uint64            `json:"dropped"`
	Delivered     uint64            `json:"delivered"`
	AggregateMbps float64           `json:"aggregate_mbps"`
	Threads       []transport.Stats `json:"threads"`
	Health        []ThreadHealth    `json:"thread_health"`
}

type threadPipe struct {
	id  string
	ep  *transport.Endpoint
	txq chan []byte

	mu       sync.Mutex
	probes   map[uint64]time.Time
	probeSeq uint64
	rttEwma  float64
	misses   int
	up       bool
}

// Node is a running Hydra data plane.
type Node struct {
	cfg      Config
	engine   *fabric.Engine
	ingress  *net.UDPConn
	egress   *net.UDPConn
	threads  map[string]*threadPipe
	order    []string
	logf     func(string, ...any)
	closing  chan struct{}
	closeOne sync.Once
	wg       sync.WaitGroup

	ingressFrames atomic.Uint64
	ingressBytes  atomic.Uint64
	egressFrames  atomic.Uint64
	egressBytes   atomic.Uint64
	dropped       atomic.Uint64
}

// udpSink is the engine's egress: reassembled payloads leave here.
type udpSink struct {
	conn *net.UDPConn
	node *Node
}

func (s *udpSink) Name() string { return "udp" }

func (s *udpSink) Send(p []byte) error {
	s.node.egressFrames.Add(1)
	s.node.egressBytes.Add(uint64(len(p)))
	if s.conn == nil {
		return nil // discard sink
	}
	_, err := s.conn.Write(p)
	return err
}

// New builds a node: it binds the ingress and egress sockets, dials every
// thread, and registers the weave. It does not start I/O; call Run.
func New(cfg Config) (*Node, error) {
	if cfg.WeaveID == "" {
		cfg.WeaveID = "default"
	}
	if cfg.Anchor == "" {
		cfg.Anchor = "local"
	}
	if cfg.QueueFrames <= 0 {
		cfg.QueueFrames = 1024
	}
	if cfg.ProbeInterval > 0 && cfg.ProbeTimeout <= 0 {
		cfg.ProbeTimeout = 3 * cfg.ProbeInterval
	}
	if cfg.ProbeFailures <= 0 {
		cfg.ProbeFailures = 3
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	if len(cfg.Threads) == 0 {
		return nil, errors.New("node: at least one thread is required")
	}

	n := &Node{
		cfg:     cfg,
		threads: map[string]*threadPipe{},
		logf:    cfg.Logf,
		closing: make(chan struct{}),
	}

	if cfg.Egress != "" {
		raddr, err := net.ResolveUDPAddr("udp", cfg.Egress)
		if err != nil {
			return nil, fmt.Errorf("node: resolve egress %q: %w", cfg.Egress, err)
		}
		conn, err := net.DialUDP("udp", nil, raddr)
		if err != nil {
			return nil, fmt.Errorf("node: dial egress %q: %w", cfg.Egress, err)
		}
		_ = conn.SetReadBuffer(4 << 20)
		_ = conn.SetWriteBuffer(4 << 20)
		n.egress = conn
	}
	n.engine = fabric.NewEngine(&udpSink{conn: n.egress, node: n}, fabric.WithAnchorName(cfg.Anchor))

	ids := make([]string, 0, len(cfg.Threads))
	for _, tc := range cfg.Threads {
		if tc.ID == "" {
			return nil, errors.New("node: thread id is required")
		}
		if _, dup := n.threads[tc.ID]; dup {
			return nil, fmt.Errorf("node: duplicate thread id %q", tc.ID)
		}
		local := tc.Local
		if local == "" {
			local = "127.0.0.1:0"
		}
		ep, err := transport.Dial(tc.ID, local, tc.Remote, tc.Mbps)
		if err != nil {
			_ = n.closeSockets()
			return nil, err
		}
		if err := n.engine.AddThread(bearer.Thread{
			ID: tc.ID, Box: cfg.ID, Mbps: tc.Mbps, Up: true,
		}); err != nil {
			_ = ep.Close()
			_ = n.closeSockets()
			return nil, err
		}
		n.threads[tc.ID] = &threadPipe{
			id:     tc.ID,
			ep:     ep,
			txq:    make(chan []byte, cfg.QueueFrames),
			probes: map[uint64]time.Time{},
			up:     true,
		}
		ids = append(ids, tc.ID)
	}
	n.order = ids

	if _, err := n.engine.CreateWeave(cfg.WeaveID, cfg.Anchor, ids); err != nil {
		_ = n.closeSockets()
		return nil, err
	}

	if cfg.Ingress != "" {
		laddr, err := net.ResolveUDPAddr("udp", cfg.Ingress)
		if err != nil {
			_ = n.closeSockets()
			return nil, fmt.Errorf("node: resolve ingress %q: %w", cfg.Ingress, err)
		}
		conn, err := net.ListenUDP("udp", laddr)
		if err != nil {
			_ = n.closeSockets()
			return nil, fmt.Errorf("node: bind ingress %q: %w", cfg.Ingress, err)
		}
		_ = conn.SetReadBuffer(4 << 20)
		_ = conn.SetWriteBuffer(4 << 20)
		n.ingress = conn
	}
	return n, nil
}

// Engine exposes the underlying weave engine (advanced use and tests).
func (n *Node) Engine() *fabric.Engine { return n.engine }

// IngressAddr returns the bound ingress address, or "" when disabled.
func (n *Node) IngressAddr() string {
	if n.ingress == nil {
		return ""
	}
	return n.ingress.LocalAddr().String()
}

// ThreadAddrs returns each thread's bound local address, so a peer node
// can be configured with the matching remotes.
func (n *Node) ThreadAddrs() map[string]string {
	out := make(map[string]string, len(n.threads))
	for id, p := range n.threads {
		out[id] = p.ep.LocalAddr().String()
	}
	return out
}

// Run starts the data plane and blocks until ctx is cancelled or Close is
// called.
func (n *Node) Run(ctx context.Context) error {
	n.start()
	select {
	case <-ctx.Done():
	case <-n.closing:
	}
	return n.Close()
}

func (n *Node) start() {
	if n.ingress != nil {
		n.wg.Add(1)
		go n.ingressLoop()
	}
	for _, id := range n.order {
		p := n.threads[id]
		n.wg.Add(2)
		go n.readLoop(p)
		go n.writeLoop(p)
	}
	if n.cfg.ProbeInterval > 0 {
		n.wg.Add(1)
		go n.probeLoop()
	}
}

// Close stops all I/O and releases the sockets. It is idempotent.
func (n *Node) Close() error {
	n.closeOne.Do(func() {
		close(n.closing)
		_ = n.closeSockets()
	})
	n.wg.Wait()
	return nil
}

func (n *Node) closeSockets() error {
	var first error
	if n.ingress != nil {
		if err := n.ingress.Close(); err != nil {
			first = err
		}
	}
	if n.egress != nil {
		if err := n.egress.Close(); err != nil && first == nil {
			first = err
		}
	}
	for _, p := range n.threads {
		if err := p.ep.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (n *Node) isClosing() bool {
	select {
	case <-n.closing:
		return true
	default:
		return false
	}
}

// ---- ingress (payload -> stripes) ----

func (n *Node) ingressLoop() {
	defer n.wg.Done()
	buf := make([]byte, transport.MaxDatagram)
	for {
		nr, _, err := n.ingress.ReadFromUDP(buf)
		if err != nil {
			if n.isClosing() {
				return
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		n.ingressFrames.Add(1)
		n.ingressBytes.Add(uint64(nr))
		// Strip immediately copies the payload into a framed packet, so the
		// read buffer can be reused safely after this call.
		res, err := n.engine.Strip(n.cfg.WeaveID, buf[:nr])
		if err != nil {
			n.dropped.Add(1)
			continue
		}
		n.enqueue(res)
	}
}

// enqueue hands a striped frame to its thread's writer. A full queue is a
// tail drop: the frame is discarded and the engine's queue accounting is
// undone so the scheduler sees the true backlog.
func (n *Node) enqueue(res fabric.StripResult) {
	p, ok := n.threads[res.ThreadID]
	if !ok {
		n.dropped.Add(1)
		n.engine.Complete(res.ThreadID, uint64(res.FrameLen))
		return
	}
	select {
	case p.txq <- res.Frame:
	default:
		n.dropped.Add(1)
		n.engine.Complete(res.ThreadID, uint64(res.FrameLen))
	}
}

// ---- thread writer (frames -> wire, paced) ----

func (n *Node) writeLoop(p *threadPipe) {
	defer n.wg.Done()
	for {
		select {
		case <-n.closing:
			return
		case frame := <-p.txq:
			if err := p.ep.Send(frame); err != nil {
				n.dropped.Add(1)
				n.engine.SetThreadHealth(p.id, 0, 0, 100, false)
				n.engine.Complete(p.id, uint64(len(frame)))
				continue
			}
			n.engine.Complete(p.id, uint64(len(frame)))
		}
	}
}

// ---- thread reader (wire -> reassembly) ----

func (n *Node) readLoop(p *threadPipe) {
	defer n.wg.Done()
	buf := make([]byte, transport.MaxDatagram)
	for {
		nr, _, err := p.ep.Recv(buf)
		if err != nil {
			if n.isClosing() {
				return
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		// Ingest copies the reassembled payload, so the read buffer is free
		// to be reused on the next iteration.
		frame := buf[:nr]
		h, _, err := shim.Decode(frame)
		if err != nil {
			continue
		}
		if h.Flags&shim.FlagProbe != 0 {
			n.handleProbe(p, h)
			continue
		}
		if _, err := n.engine.Ingest(n.cfg.WeaveID, frame); err != nil {
			n.logf("hydra node %s: ingest on thread %s: %v", n.cfg.ID, p.id, err)
		}
	}
}

// ---- liveness probing ----

func (n *Node) probeLoop() {
	defer n.wg.Done()
	ticker := time.NewTicker(n.cfg.ProbeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-n.closing:
			return
		case <-ticker.C:
			for _, id := range n.order {
				n.probeOnce(n.threads[id])
			}
		}
	}
}

// probeOnce sends one liveness probe, or reaps an outstanding one that
// timed out. A thread that misses ProbeFailures probes in a row is marked
// down; the first echo clears the miss count and marks it up again.
func (n *Node) probeOnce(p *threadPipe) {
	p.mu.Lock()
	for seq, at := range p.probes {
		if time.Since(at) > n.cfg.ProbeTimeout {
			delete(p.probes, seq)
			p.misses++
		}
	}
	misses := p.misses
	outstanding := len(p.probes)
	p.mu.Unlock()

	if misses >= n.cfg.ProbeFailures {
		n.engine.SetThreadHealth(p.id, 0, 0, 100, false)
	}
	// Keep exactly one probe in flight even while the thread is down, so
	// recovery is detectable: a black-holed link that comes back must be
	// able to prove it.
	if outstanding > 0 {
		return
	}
	p.mu.Lock()
	p.probeSeq++
	seq := p.probeSeq
	p.probes[seq] = time.Now()
	p.mu.Unlock()

	h := shim.New(fabric.WeaveHash(n.cfg.WeaveID), seq, 0)
	h.Flags = shim.FlagProbe
	if err := p.ep.Send(shim.Frame(h, nil)); err != nil {
		p.mu.Lock()
		delete(p.probes, seq)
		p.mu.Unlock()
	}
}

// handleProbe echoes an inbound probe, or records the RTT of an echo.
func (n *Node) handleProbe(p *threadPipe, h shim.Header) {
	if h.Flags&shim.FlagLast != 0 {
		p.mu.Lock()
		sent, ok := p.probes[h.Seq]
		if ok {
			delete(p.probes, h.Seq)
		}
		p.mu.Unlock()
		if !ok {
			return
		}
		rtt := float64(time.Since(sent).Microseconds()) / 1000
		p.mu.Lock()
		if p.rttEwma == 0 {
			p.rttEwma = rtt
		} else {
			p.rttEwma = 0.8*p.rttEwma + 0.2*rtt
		}
		p.misses = 0
		p.up = true
		ewma := p.rttEwma
		p.mu.Unlock()
		n.engine.SetThreadHealth(p.id, 0, ewma, 0, true)
		return
	}
	// Request: echo it back with FlagLast so the originator can match it.
	echo := shim.Header{
		Version: h.Version,
		Flags:   h.Flags | shim.FlagLast,
		WeaveID: h.WeaveID,
		Seq:     h.Seq,
		TSNanos: h.TSNanos,
	}
	if err := p.ep.Send(shim.Frame(echo, nil)); err != nil {
		n.logf("hydra node %s: probe echo thread %s: %v", n.cfg.ID, p.id, err)
	}
}

// ---- stats ----

// Stats returns a snapshot of node activity.
func (n *Node) Stats() Stats {
	st := Stats{
		NodeID:        n.cfg.ID,
		WeaveID:       n.cfg.WeaveID,
		Anchor:        n.cfg.Anchor,
		IngressFrames: n.ingressFrames.Load(),
		IngressBytes:  n.ingressBytes.Load(),
		EgressFrames:  n.egressFrames.Load(),
		EgressBytes:   n.egressBytes.Load(),
		Dropped:       n.dropped.Load(),
	}
	for _, id := range n.order {
		p := n.threads[id]
		st.Threads = append(st.Threads, p.ep.Stats())
		p.mu.Lock()
		h := ThreadHealth{ID: id, RTTms: p.rttEwma, Misses: p.misses, Up: p.up}
		p.mu.Unlock()
		if t, ok := n.engine.Thread(id); ok {
			h.Mbps = t.Mbps
			h.Up = t.Up
		}
		st.Health = append(st.Health, h)
	}
	es := n.engine.Status()
	st.AggregateMbps = es.AggregateMbps
	st.Delivered = es.Delivered
	return st
}

// ---- lab controls ----

// SetThreadLoss sets a synthetic drop percentage on a thread. It is a lab
// hook for exercising failover without touching a real radio.
func (n *Node) SetThreadLoss(id string, pct float64) bool {
	p, ok := n.threads[id]
	if !ok {
		return false
	}
	p.ep.SetLoss(pct)
	return true
}

// SetThreadRate updates a thread's declared capacity in Mbps.
func (n *Node) SetThreadRate(id string, mbps float64) bool {
	p, ok := n.threads[id]
	if !ok {
		return false
	}
	p.ep.SetRate(mbps)
	n.engine.Registry().Update(id, func(t *bearer.Thread) { t.Mbps = mbps })
	return true
}
