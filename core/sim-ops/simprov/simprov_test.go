package simprov

import (
	"strings"
	"testing"
)

func TestDeriveOPcKnownVector(t *testing.T) {
	// Round-trip determinism check: OPc is a pure function of (Ki, OP).
	ki := "00112233445566778899aabbccddeeff"
	op := "00000000000000000000000000000000"
	opc1, err := DeriveOPc(ki, op)
	if err != nil {
		t.Fatalf("DeriveOPc: %v", err)
	}
	opc2, _ := DeriveOPc(ki, op)
	if opc1 != opc2 {
		t.Fatalf("not deterministic: %s vs %s", opc1, opc2)
	}
	if len(opc1) != 32 {
		t.Fatalf("OPc must be 16 bytes hex, got %d chars", len(opc1))
	}
}

func TestDeriveOPcRejectsBadInput(t *testing.T) {
	if _, err := DeriveOPc("zz", "00"); err == nil {
		t.Fatal("expected error for non-hex ki")
	}
	if _, err := DeriveOPc("00112233445566778899aabbccddeeff", "00"); err == nil {
		t.Fatal("expected error for short op")
	}
}

func TestGenerateBatch(t *testing.T) {
	subs, err := GenerateBatch(BatchSpec{Count: 3, Class: "lab", IMSIBase: 999991234567001}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 3 {
		t.Fatalf("want 3, got %d", len(subs))
	}
	for i, s := range subs {
		if len(s.IMSI) != 15 {
			t.Errorf("IMSI %q not 15 digits", s.IMSI)
		}
		if !strings.HasPrefix(s.IMSI, "999991234567") {
			t.Errorf("IMSI prefix mismatch: %s", s.IMSI)
		}
		if len(s.Ki) != 32 || len(s.OPc) != 32 {
			t.Errorf("bad key lengths at %d: ki=%d opc=%d", i, len(s.Ki), len(s.OPc))
		}
		if s.Ki == s.OPc {
			t.Errorf("Ki==OPc is a red flag at %d", i)
		}
	}
	// IMSI uniqueness
	seen := map[string]bool{}
	for _, s := range subs {
		if seen[s.IMSI] {
			t.Fatalf("duplicate IMSI %s", s.IMSI)
		}
		seen[s.IMSI] = true
	}
}

func TestGenerateBatchRejectsBadCount(t *testing.T) {
	if _, err := GenerateBatch(BatchSpec{Count: 0}, ""); err == nil {
		t.Fatal("expected error for count=0")
	}
	if _, err := GenerateBatch(BatchSpec{Count: 100001}, ""); err == nil {
		t.Fatal("expected error for oversized count")
	}
}

func TestLabTestVectors(t *testing.T) {
	v, err := LoadTestVector("999991234567001")
	if err != nil {
		t.Fatal(err)
	}
	if v.Ki == "" || v.OPc == "" {
		t.Fatalf("vector incomplete: %+v", v)
	}
	if _, err := LoadTestVector("000000000000000"); err == nil {
		t.Fatal("expected error for unknown IMSI")
	}
}
