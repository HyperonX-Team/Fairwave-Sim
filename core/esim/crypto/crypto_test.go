package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// NIST SP 800-38B section D.1.1/D.1.2 known-answer vectors for AES-128-CMAC.
var cmacKAT = []struct {
	key, msg, mac string
}{
	{
		key: "2b7e151628aed2a6abf7158809cf4f3c",
		msg: "",
		mac: "bb1d6929e95937287fa37d129b756746",
	},
	{
		key: "2b7e151628aed2a6abf7158809cf4f3c",
		msg: "6bc1bee22e409f96e93d7e117393172a",
		mac: "070a16b46b4d4144f79bdd9dd04a287c",
	},
	{
		key: "2b7e151628aed2a6abf7158809cf4f3c",
		msg: "6bc1bee22e409f96e93d7e117393172aae2d8a571e03ac9c9eb76fac45af8e5130c81c46a35ce411",
		mac: "dfa66747de9ae63030ca32611497c827",
	},
}

func TestCMACKnownAnswerVectors(t *testing.T) {
	for i, tc := range cmacKAT {
		key, _ := hex.DecodeString(tc.key)
		msg, _ := hex.DecodeString(tc.msg)
		want, _ := hex.DecodeString(tc.mac)
		got, err := CMAC(key, msg)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("case %d: got %x, want %x", i, got, want)
		}
	}
}

func TestKDFDeterministic(t *testing.T) {
	key, _ := hex.DecodeString("2b7e151628aed2a6abf7158809cf4f3c")
	a, err := KDF(key, []byte{0x01, 0x02, 0x03}, 32)
	if err != nil {
		t.Fatal(err)
	}
	b, err := KDF(key, []byte{0x01, 0x02, 0x03}, 32)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("KDF not deterministic")
	}
	if len(a) != 32 {
		t.Fatalf("KDF length: got %d, want 32", len(a))
	}
	c, err := KDF(key, []byte{0x01, 0x02}, 32)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, c) {
		t.Fatal("KDF context not mixed in")
	}
}

func TestKDFLengthValidation(t *testing.T) {
	key := make([]byte, 16)
	if _, err := KDF(key, nil, 7); err == nil {
		t.Fatal("expected error for non-multiple-of-16 length")
	}
	if _, err := KDF(make([]byte, 8), nil, 16); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestDeriveSessionKeys(t *testing.T) {
	shared := bytes.Repeat([]byte{0x5a}, 32)
	keys, err := DeriveSessionKeys(shared)
	if err != nil {
		t.Fatal(err)
	}
	if keys.Enc == keys.Dek {
		t.Fatal("session keys must differ across labels")
	}
	again, err := DeriveSessionKeys(shared)
	if err != nil {
		t.Fatal(err)
	}
	if keys.Enc != again.Enc || keys.Mac != again.Mac {
		t.Fatal("session key derivation not deterministic")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 16)
	cases := [][]byte{
		{},
		[]byte("hello"),
		bytes.Repeat([]byte("x"), 16),
		bytes.Repeat([]byte("x"), 17),
		bytes.Repeat([]byte("x"), 255),
	}
	for i, pt := range cases {
		ct, err := Encrypt(key, pt)
		if err != nil {
			t.Fatalf("case %d: encrypt: %v", i, err)
		}
		if bytes.Equal(ct, pt) {
			t.Fatalf("case %d: ciphertext equals plaintext", i)
		}
		got, err := Decrypt(key, ct)
		if err != nil {
			t.Fatalf("case %d: decrypt: %v", i, err)
		}
		if !bytes.Equal(got, pt) {
			t.Fatalf("case %d: round trip mismatch: got %x want %x", i, got, pt)
		}
	}
}

func TestDecryptTamperDetect(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 16)
	ct, err := Encrypt(key, []byte("profile payload"))
	if err != nil {
		t.Fatal(err)
	}
	ct[len(ct)-1] ^= 0x01
	if _, err := Decrypt(key, ct); err == nil {
		t.Fatal("expected padding error on tampered ciphertext")
	}
}

func TestECDHSharedSecretAgrees(t *testing.T) {
	a, err := ECDHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	b, err := ECDHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	sa, err := ECDHShared(a, ECDHPublic(b.PublicKey()))
	if err != nil {
		t.Fatal(err)
	}
	sb, err := ECDHShared(b, ECDHPublic(a.PublicKey()))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sa, sb) {
		t.Fatal("ECDH shared secrets differ")
	}
	if len(sa) != 32 {
		t.Fatalf("P-256 shared secret: got %d bytes, want 32", len(sa))
	}
}

func TestVerifyMAC(t *testing.T) {
	if !VerifyMAC([]byte("abcdefghijklmnop"), []byte("abcdefghijklmnop")) {
		t.Fatal("equal MACs must verify")
	}
	if VerifyMAC([]byte("abcdefghijklmnop"), []byte("abcdefghijklmnopq")) {
		t.Fatal("length mismatch must fail")
	}
	if VerifyMAC([]byte("abcdefghijklmnop"), []byte("aaaaaaaaaaaaaaaa")) {
		t.Fatal("content mismatch must fail")
	}
}
