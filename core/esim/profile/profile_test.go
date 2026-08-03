package profile

import (
	"bytes"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

func TestLuhnKnownValues(t *testing.T) {
	// Classic Luhn example: 79927398713 is valid, 79927398714 is not.
	if got := Luhn("7992739871"); got != 3 {
		t.Fatalf("Luhn(7992739871) = %d, want 3", got)
	}
	if !ValidLuhn("79927398713") {
		t.Fatal("79927398713 must be Luhn-valid")
	}
	if ValidLuhn("79927398714") {
		t.Fatal("79927398714 must fail Luhn")
	}
	if ValidLuhn("12ab") {
		t.Fatal("non-digits must fail Luhn")
	}
	if ValidLuhn("7") {
		t.Fatal("single digit must fail Luhn")
	}
}

func TestNewICCIDValid(t *testing.T) {
	for _, mno := range []string{"99901", "00101", "999"} {
		iccid, err := NewICCID(mno)
		if err != nil {
			t.Fatal(err)
		}
		if len(iccid) != 20 {
			t.Fatalf("iccid %q: got %d digits, want 20", iccid, len(iccid))
		}
		if !ValidLuhn(iccid) {
			t.Fatalf("iccid %q fails Luhn", iccid)
		}
		if iccid[:2] != "89" {
			t.Fatalf("iccid %q must start with 89", iccid)
		}
	}
}

func TestNewEIDValid(t *testing.T) {
	eid, err := NewEID()
	if err != nil {
		t.Fatal(err)
	}
	if len(eid) != 32 {
		t.Fatalf("eid %q: got %d digits, want 32", eid, len(eid))
	}
	if !ValidLuhn(eid) {
		t.Fatalf("eid %q fails Luhn", eid)
	}
	if eid[:4] != "8904" {
		t.Fatalf("eid %q must start with 8904", eid)
	}
}

func TestNewLabProfile(t *testing.T) {
	sub, err := simprov.LoadTestVector("999991234567001")
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewLabProfile(sub, "fairwave.test")
	if err != nil {
		t.Fatal(err)
	}
	if p.IMSI != sub.IMSI || p.KI != sub.Ki || p.OPc != sub.OPc {
		t.Fatal("profile must carry the subscriber credentials")
	}
	if p.Class != "lab" {
		t.Fatalf("class = %q, want lab", p.Class)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestNewLabProfileRejectsProd(t *testing.T) {
	sub, _ := simprov.LoadTestVector("999991234567001")
	sub.Class = "prod"
	if _, err := NewLabProfile(sub, "x"); err == nil {
		t.Fatal("prod subscriber must not mint lab profiles")
	}
}

func TestIMSIAndICCIDFiles(t *testing.T) {
	sub, _ := simprov.LoadTestVector("999991234567001")
	p, err := NewLabProfile(sub, "fairwave.test")
	if err != nil {
		t.Fatal(err)
	}
	// IMSI 999991234567001 -> pairs (9,9)(9,9)(9,1)(2,3)(4,5)(6,7)(0,0)
	// swapped-BCD: 99 99 19 32 54 76 00, trailing nibble 0xF1.
	imsiFile, err := p.IMSIFile()
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x09, 0x99, 0x99, 0x19, 0x32, 0x54, 0x76, 0x00, 0xF1}
	if !bytes.Equal(imsiFile, want) {
		t.Fatalf("imsi file = %x, want %x", imsiFile, want)
	}

	iccidFile, err := p.ICCIDFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(iccidFile) != 10 {
		t.Fatalf("iccid file length = %d, want 10", len(iccidFile))
	}
	if iccidFile[0] != 0x98 { // "89" swapped
		t.Fatalf("iccid file first byte = %x, want 0x98", iccidFile[0])
	}
}

func TestPayloadRoundTrip(t *testing.T) {
	sub, _ := simprov.LoadTestVector("999991234567002")
	p, err := NewLabProfile(sub, "fairwave.test")
	if err != nil {
		t.Fatal(err)
	}
	data, err := p.MarshalPayload()
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnmarshalPayload(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.ICCID != p.ICCID || got.KI != p.KI || got.IMSI != p.IMSI {
		t.Fatal("payload round trip mismatch")
	}
}

func TestBPPRoundTrip(t *testing.T) {
	ep := EncryptedProfile{
		SeqCounter: 42,
		Ciphertext: bytes.Repeat([]byte{0xAB}, 100),
		MAC:        bytes.Repeat([]byte{0xCD}, 16),
	}
	der, err := EncodeBPP(ep)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeBPP(der)
	if err != nil {
		t.Fatal(err)
	}
	if got.SeqCounter != 42 || !bytes.Equal(got.Ciphertext, ep.Ciphertext) || !bytes.Equal(got.MAC, ep.MAC) {
		t.Fatal("BPP round trip mismatch")
	}
}

func TestDecodeBPPRejectsGarbage(t *testing.T) {
	if _, err := DecodeBPP([]byte{0x00, 0x01, 0x02}); err == nil {
		t.Fatal("expected error for garbage BPP")
	}
}
