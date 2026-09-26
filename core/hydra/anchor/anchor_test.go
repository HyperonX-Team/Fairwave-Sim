package anchor

import "testing"

func TestSinkCounts(t *testing.T) {
	s := NewSink()
	if s.Name() != "sink" {
		t.Fatalf("name = %q", s.Name())
	}
	if err := s.Send([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := s.Send([]byte("world!")); err != nil {
		t.Fatal(err)
	}
	if s.Frames() != 2 || s.Bytes() != 11 {
		t.Fatalf("frames=%d bytes=%d", s.Frames(), s.Bytes())
	}
}
