package node

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// startSink binds a UDP socket that receives reassembled payloads.
func startSink(t *testing.T) *net.UDPConn {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadBuffer(4 << 20)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// pairOpts configures a two-node weave for tests.
type pairOpts struct {
	threads      int
	mbps         float64
	anchorEgress string
	edgeEgress   string
	probe        bool
}

// startPair brings up a fully bidirectional anchor and edge sharing one
// weave. Both have an ingress; egress is wired to the supplied sinks.
func startPair(t *testing.T, o pairOpts) (*Node, *Node) {
	t.Helper()
	anchorThreads := make([]ThreadConfig, 0, o.threads)
	for i := 1; i <= o.threads; i++ {
		anchorThreads = append(anchorThreads, ThreadConfig{
			ID: fmt.Sprintf("t%d", i), Local: "127.0.0.1:0", Mbps: o.mbps,
		})
	}
	anchorCfg := Config{
		ID: "anchor", WeaveID: "w", Anchor: "anchor",
		Ingress: "127.0.0.1:0", Egress: o.anchorEgress,
		Threads: anchorThreads, QueueFrames: 4096,
	}
	anchor, err := New(anchorCfg)
	if err != nil {
		t.Fatalf("anchor: %v", err)
	}
	actx, acancel := context.WithCancel(context.Background())
	go func() { _ = anchor.Run(actx) }()
	t.Cleanup(func() { acancel(); _ = anchor.Close() })

	addrs := anchor.ThreadAddrs()
	edgeThreads := make([]ThreadConfig, 0, o.threads)
	for i := 1; i <= o.threads; i++ {
		id := fmt.Sprintf("t%d", i)
		edgeThreads = append(edgeThreads, ThreadConfig{
			ID: id, Local: "127.0.0.1:0", Remote: addrs[id], Mbps: o.mbps,
		})
	}
	edgeCfg := Config{
		ID: "edge", WeaveID: "w", Anchor: "anchor",
		Ingress: "127.0.0.1:0", Egress: o.edgeEgress,
		Threads: edgeThreads, QueueFrames: 4096,
	}
	if o.probe {
		edgeCfg.ProbeInterval = 20 * time.Millisecond
		edgeCfg.ProbeTimeout = 50 * time.Millisecond
		edgeCfg.ProbeFailures = 2
	}
	edge, err := New(edgeCfg)
	if err != nil {
		t.Fatalf("edge: %v", err)
	}
	ectx, ecancel := context.WithCancel(context.Background())
	go func() { _ = edge.Run(ectx) }()
	t.Cleanup(func() { ecancel(); _ = edge.Close() })

	return anchor, edge
}

// dialIngress returns a connected socket that writes payloads into a node's
// ingress.
func dialIngress(t *testing.T, n *Node) *net.UDPConn {
	t.Helper()
	raddr, err := net.ResolveUDPAddr("udp", n.IngressAddr())
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// sendSeq writes n payloads whose first four bytes are a big-endian index.
// It paces in small batches so the receiving socket buffer absorbs the
// burst; the weave's own rate limiters remain the throughput bottleneck.
func sendSeq(t *testing.T, ing *net.UDPConn, n, size int) {
	t.Helper()
	payload := make([]byte, size)
	for i := 0; i < n; i++ {
		binary.BigEndian.PutUint32(payload[:4], uint32(i))
		if _, err := ing.Write(payload); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		if i%25 == 24 {
			time.Sleep(200 * time.Microsecond)
		}
	}
}

// collect reads exactly n payloads from the sink and returns their leading
// sequence numbers in arrival order.
func collect(t *testing.T, sink *net.UDPConn, n int, timeout time.Duration) []uint32 {
	t.Helper()
	seqs := make([]uint32, 0, n)
	buf := make([]byte, 2048)
	deadline := time.Now().Add(timeout)
	for len(seqs) < n {
		if time.Now().After(deadline) {
			t.Fatalf("sink timed out after %d/%d payloads", len(seqs), n)
		}
		_ = sink.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		nr, _, err := sink.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			t.Fatalf("sink read: %v", err)
		}
		if nr < 4 {
			continue
		}
		seqs = append(seqs, binary.BigEndian.Uint32(buf[:4]))
	}
	return seqs
}

func assertOrdered(t *testing.T, seqs []uint32) {
	t.Helper()
	for i, s := range seqs {
		if s != uint32(i) {
			t.Fatalf("payload %d out of order: got seq %d", i, s)
		}
	}
}

func TestNodeDeliversOrderedAcrossThreads(t *testing.T) {
	sink := startSink(t)
	anchor, edge := startPair(t, pairOpts{threads: 3, mbps: 20, anchorEgress: sink.LocalAddr().String()})

	ing := dialIngress(t, edge)
	const n = 600
	sendSeq(t, ing, n, 800)
	assertOrdered(t, collect(t, sink, n, 10*time.Second))

	st := edge.Stats()
	if len(st.Threads) != 3 {
		t.Fatalf("threads = %d", len(st.Threads))
	}
	for _, ts := range st.Threads {
		if ts.SentFrames == 0 {
			t.Fatalf("thread %s carried no traffic: %+v", ts.ID, ts)
		}
	}
	if st.AggregateMbps != 60 {
		t.Fatalf("aggregate = %v want 60", st.AggregateMbps)
	}
	if st.Dropped != 0 {
		t.Fatalf("dropped %d frames", st.Dropped)
	}
	if as := anchor.Stats(); as.EgressFrames < n {
		t.Fatalf("anchor egress frames = %d want >= %d", as.EgressFrames, n)
	}
}

func TestNodeIsBidirectional(t *testing.T) {
	anchorSink := startSink(t)
	edgeSink := startSink(t)
	anchor, edge := startPair(t, pairOpts{
		threads: 2, mbps: 50,
		anchorEgress: anchorSink.LocalAddr().String(),
		edgeEgress:   edgeSink.LocalAddr().String(),
	})

	const n = 150

	// Edge -> anchor first; this also teaches the anchor its peer addresses.
	sendSeq(t, dialIngress(t, edge), n, 400)
	assertOrdered(t, collect(t, anchorSink, n, 10*time.Second))

	// Anchor -> edge over the same weave, now that peers are known.
	sendSeq(t, dialIngress(t, anchor), n, 400)
	assertOrdered(t, collect(t, edgeSink, n, 10*time.Second))
}

func TestNodeProbeMarksThreadDownAndRecovers(t *testing.T) {
	sink := startSink(t)
	_, edge := startPair(t, pairOpts{threads: 3, mbps: 50, anchorEgress: sink.LocalAddr().String(), probe: true})

	waitUp := func(id string, want bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if th, ok := edge.Engine().Thread(id); ok && th.Up == want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		th, _ := edge.Engine().Thread(id)
		t.Fatalf("thread %s up=%v, want %v", id, th.Up, want)
	}

	waitUp("t2", true)

	if !edge.SetThreadLoss("t2", 100) {
		t.Fatal("SetThreadLoss: unknown thread")
	}
	waitUp("t2", false)

	if !edge.SetThreadLoss("t2", 0) {
		t.Fatal("SetThreadLoss: unknown thread")
	}
	waitUp("t2", true)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, h := range edge.Stats().Health {
			if h.ID == "t2" && h.RTTms > 0 {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no RTT measured for t2")
}

// TestRunFromFileConfig builds both ends of a weave from on-disk YAML, the
// same path the fairwave-hydra daemon uses, and moves a stream across it.
func TestRunFromFileConfig(t *testing.T) {
	dir := t.TempDir()
	sink := startSink(t)

	anchorYAML := fmt.Sprintf(`
node_id: anchor
weave: w
anchor: hub
ingress: 127.0.0.1:0
egress: %s
probe_interval: 50ms
probe_failures: 3
threads:
  - {id: t1, local: "127.0.0.1:0", mbps: 50}
  - {id: t2, local: "127.0.0.1:0", mbps: 50}
`, sink.LocalAddr().String())
	anchorPath := filepath.Join(dir, "anchor.yaml")
	if err := os.WriteFile(anchorPath, []byte(anchorYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	afc, err := LoadFile(anchorPath)
	if err != nil {
		t.Fatal(err)
	}
	acfg, err := afc.Config()
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := New(acfg)
	if err != nil {
		t.Fatal(err)
	}
	actx, acancel := context.WithCancel(context.Background())
	go func() { _ = anchor.Run(actx) }()
	t.Cleanup(func() { acancel(); _ = anchor.Close() })

	addrs := anchor.ThreadAddrs()
	edgeYAML := fmt.Sprintf(`
node_id: edge
weave: w
anchor: hub
ingress: 127.0.0.1:0
probe_interval: 50ms
threads:
  - {id: t1, local: "127.0.0.1:0", remote: "%s", mbps: 50}
  - {id: t2, local: "127.0.0.1:0", remote: "%s", mbps: 50}
`, addrs["t1"], addrs["t2"])
	edgePath := filepath.Join(dir, "edge.yaml")
	if err := os.WriteFile(edgePath, []byte(edgeYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	efc, err := LoadFile(edgePath)
	if err != nil {
		t.Fatal(err)
	}
	ecfg, err := efc.Config()
	if err != nil {
		t.Fatal(err)
	}
	edge, err := New(ecfg)
	if err != nil {
		t.Fatal(err)
	}
	ectx, ecancel := context.WithCancel(context.Background())
	go func() { _ = edge.Run(ectx) }()
	t.Cleanup(func() { ecancel(); _ = edge.Close() })

	const n = 200
	sendSeq(t, dialIngress(t, edge), n, 600)
	assertOrdered(t, collect(t, sink, n, 10*time.Second))
}

func TestNodeAggregateBeatsSingleThread(t *testing.T) {
	measure := func(threads int) time.Duration {
		sink := startSink(t)
		_, edge := startPair(t, pairOpts{threads: threads, mbps: 20, anchorEgress: sink.LocalAddr().String()})
		ing := dialIngress(t, edge)
		const n = 800
		start := time.Now()
		sendSeq(t, ing, n, 800)
		collect(t, sink, n, 15*time.Second)
		return time.Since(start)
	}

	single := measure(1)
	triple := measure(3)
	// Three 20 Mbps threads should be ~3x faster; require at least 1.5x to
	// stay robust on a loaded CI runner.
	if triple >= single*3/2 {
		t.Fatalf("aggregate not faster: 1 thread=%v, 3 threads=%v", single, triple)
	}
}
