package shim

import (
	"errors"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	h := New(0xDEADBEEF, 42, 1_700_000_000_000_000_000)
	h.Flags = FlagProbe | FlagLast
	frame := Frame(h, []byte("payload-bytes"))

	got, payload, err := Decode(frame)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != h {
		t.Fatalf("header mismatch:\n got %+v\nwant %+v", got, h)
	}
	if string(payload) != "payload-bytes" {
		t.Fatalf("payload = %q", payload)
	}
	if len(frame) != HeaderLen+len("payload-bytes") {
		t.Fatalf("frame len = %d", len(frame))
	}
}

func TestDecodeShort(t *testing.T) {
	if _, _, err := Decode(make([]byte, HeaderLen-1)); !errors.Is(err, ErrShort) {
		t.Fatalf("want ErrShort, got %v", err)
	}
}

func TestDecodeBadMagic(t *testing.T) {
	frame := Frame(New(1, 1, 0), nil)
	frame[0] = 0x00
	if _, _, err := Decode(frame); !errors.Is(err, ErrMagic) {
		t.Fatalf("want ErrMagic, got %v", err)
	}
}

func TestDecodeBadVersion(t *testing.T) {
	frame := Frame(New(1, 1, 0), nil)
	frame[2] = Version + 1
	if _, _, err := Decode(frame); !errors.Is(err, ErrVersion) {
		t.Fatalf("want ErrVersion, got %v", err)
	}
}

func TestEmptyPayload(t *testing.T) {
	frame := Frame(New(7, 9, 123), nil)
	h, payload, err := Decode(frame)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if h.Seq != 9 || len(payload) != 0 {
		t.Fatalf("seq=%d payload=%d", h.Seq, len(payload))
	}
}
