package transport

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestSendRecvLearnsPeer(t *testing.T) {
	a, err := Dial("a", "127.0.0.1:0", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Dial("b", "127.0.0.1:0", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	// a must be told about b; b learns a from the first datagram.
	if err := a.SetRemote(b.LocalAddr().String()); err != nil {
		t.Fatal(err)
	}
	if err := a.Send([]byte("hello")); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 64)
	n, addr, err := b.Recv(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "hello" {
		t.Fatalf("got %q", buf[:n])
	}
	if addr == nil {
		t.Fatal("no source address")
	}
	if b.Peer() == "" {
		t.Fatal("peer not learned")
	}

	// b can now reply without being configured with a remote.
	if err := b.Send([]byte("world")); err != nil {
		t.Fatal(err)
	}
	n, _, err = a.Recv(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "world" {
		t.Fatalf("reply = %q", buf[:n])
	}
}

func TestSendWithoutPeerFails(t *testing.T) {
	a, err := Dial("a", "127.0.0.1:0", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Send([]byte("x")); !errors.Is(err, ErrNoPeer) {
		t.Fatalf("want ErrNoPeer, got %v", err)
	}
}

func TestRateLimitPacesSends(t *testing.T) {
	a, err := Dial("a", "127.0.0.1:0", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Dial("b", "127.0.0.1:0", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if err := a.SetRemote(b.LocalAddr().String()); err != nil {
		t.Fatal(err)
	}

	// 100 kB/s: ten 1000-byte frames take ~0.1s.
	a.SetRate(0.8) // 0.8 Mbps = 100 kB/s
	frame := make([]byte, 1000)
	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := a.Send(frame); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	if elapsed < 80*time.Millisecond {
		t.Fatalf("paced too fast: %v (expected >=80ms for 10kB at 100kB/s)", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("paced too slowly: %v", elapsed)
	}
	if s := a.Stats(); s.SentFrames != 10 || s.SentBytes != 10000 {
		t.Fatalf("stats = %+v", s)
	}
}

func TestLossDropsFrames(t *testing.T) {
	a, err := Dial("a", "127.0.0.1:0", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Dial("b", "127.0.0.1:0", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if err := a.SetRemote(b.LocalAddr().String()); err != nil {
		t.Fatal(err)
	}
	a.SetLoss(100)
	if err := a.Send([]byte("gone")); err != nil {
		t.Fatal(err)
	}
	if s := a.Stats(); s.Dropped != 1 || s.SentFrames != 0 {
		t.Fatalf("stats = %+v", s)
	}
}

func TestTooLarge(t *testing.T) {
	a, err := Dial("a", "127.0.0.1:0", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Send(make([]byte, MaxDatagram+1)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
}

func TestCloseUnblocksRecv(t *testing.T) {
	a, err := Dial("a", "127.0.0.1:0", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 64)
		_, _, err := a.Recv(buf)
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("want ErrClosed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Recv did not unblock on Close")
	}
}

func TestRoundTripPayloadIntegrity(t *testing.T) {
	a, _ := Dial("a", "127.0.0.1:0", "", 0)
	defer a.Close()
	b, _ := Dial("b", "127.0.0.1:0", "", 0)
	defer b.Close()
	if err := a.SetRemote(b.LocalAddr().String()); err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte{0xAB, 0xCD, 0xEF}, 400) // 1200 bytes
	if err := a.Send(payload); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 2048)
	n, _, err := b.Recv(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf[:n], payload) {
		t.Fatalf("payload corrupted: %d bytes", n)
	}
}
