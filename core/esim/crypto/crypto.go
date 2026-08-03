// Package crypto implements the SGP.22-shaped cryptographic primitives used
// by the Fairwave eSIM stack (SM-DP+ server and software eUICC):
//
//   - AES-128-CMAC (NIST SP 800-38B)
//   - counter-mode key derivation based on AES-CMAC (NIST SP 800-108,
//     SGP.22 section 5.1.1 shape)
//   - session key derivation from an ECDH shared secret (K1..K4)
//   - AES-128-CBC with PKCS#7 padding for profile payload encryption
//
// NOTE: this is a LAB implementation. The byte-level context/label layout
// follows the SGP.22 structure but is defined locally until it is validated
// against the GSMA conformance tooling with a physical phone. See
// docs/adr/0013-esim.md. Never use these keys outside a lab.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
)

// rb is the AES-128 CMAC subkey constant (x^128 bitmask, NIST SP 800-38B).
var rb = [16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x87}

func cmacSubkeys(key []byte) (k1, k2 [16]byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return k1, k2, err
	}
	zero := make([]byte, 16)
	l := make([]byte, 16)
	block.Encrypt(l, zero)
	k1 = shiftLeft(l)
	if l[0]&0x80 != 0 {
		for i := 0; i < 16; i++ {
			k1[i] ^= rb[i]
		}
	}
	l2 := k1[:]
	k2 = shiftLeft(l2)
	if k1[0]&0x80 != 0 {
		for i := 0; i < 16; i++ {
			k2[i] ^= rb[i]
		}
	}
	return k1, k2, nil
}

func shiftLeft(in []byte) (out [16]byte) {
	var carry byte
	for i := 15; i >= 0; i-- {
		out[i] = in[i]<<1 | carry
		carry = in[i] >> 7
	}
	return out
}

// CMAC computes the full-length AES-128-CMAC of msg (NIST SP 800-38B).
func CMAC(key, msg []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, fmt.Errorf("cmac: key must be 16 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	k1, k2, err := cmacSubkeys(key)
	if err != nil {
		return nil, err
	}

	blocks := len(msg) / 16
	rem := len(msg) % 16

	var last [16]byte
	if rem == 0 && blocks > 0 {
		copy(last[:], msg[(blocks-1)*16:])
		for i := 0; i < 16; i++ {
			last[i] ^= k1[i]
		}
	} else {
		copy(last[:], msg[blocks*16:])
		last[rem] = 0x80
		for i := 0; i < 16; i++ {
			last[i] ^= k2[i]
		}
	}

	x := make([]byte, 16)
	// CBC chain over every block except the last. A partial trailing block
	// means the message has blocks+1 blocks in total.
	chainBlocks := blocks
	if rem == 0 {
		chainBlocks--
	}
	for i := 0; i < chainBlocks; i++ {
		y := make([]byte, 16)
		for j := 0; j < 16; j++ {
			y[j] = msg[i*16+j] ^ x[j]
		}
		block.Encrypt(x, y)
	}
	y := make([]byte, 16)
	for j := 0; j < 16; j++ {
		y[j] = last[j] ^ x[j]
	}
	out := make([]byte, 16)
	block.Encrypt(out, y)
	return out, nil
}

// KDF derives length bytes using the AES-CMAC counter-mode KDF
// (NIST SP 800-108 style, SGP.22 section 5.1.1 shape):
//
//	Ki = CMAC(key, context || BE32(i)) for i = 1, 2, ...
//
// Output is the concatenation truncated to length (multiple of 16).
//
// The KDF key must be 16 bytes. P-256 ECDH shared secrets are 32 bytes;
// those are folded to 16 bytes by XOR-ing the two halves. The fold is a
// lab-defined detail pending GSMA conformance testing (see
// docs/adr/0013-esim.md); both SM-DP+ and eUICC apply the same rule.
func KDF(key, context []byte, length int) ([]byte, error) {
	k, err := cmacKey(key)
	if err != nil {
		return nil, err
	}
	if length <= 0 || length%16 != 0 {
		return nil, fmt.Errorf("kdf: length must be a positive multiple of 16, got %d", length)
	}
	out := make([]byte, 0, length)
	counter := make([]byte, 4)
	for i := 1; len(out) < length; i++ {
		binary.BigEndian.PutUint32(counter, uint32(i))
		m := make([]byte, 0, len(context)+4)
		m = append(m, context...)
		m = append(m, counter...)
		block, err := CMAC(k, m)
		if err != nil {
			return nil, err
		}
		out = append(out, block...)
	}
	return out[:length], nil
}

// cmacKey normalizes the KDF key to 16 bytes: 16-byte keys pass through,
// 32-byte keys (P-256 shared secrets) are folded by XOR-ing halves.
func cmacKey(key []byte) ([]byte, error) {
	switch len(key) {
	case 16:
		return key, nil
	case 32:
		folded := make([]byte, 16)
		for i := 0; i < 16; i++ {
			folded[i] = key[i] ^ key[i+16]
		}
		return folded, nil
	default:
		return nil, fmt.Errorf("kdf: key must be 16 or 32 bytes, got %d", len(key))
	}
}

// DeriveSessionKey derives a single 16-byte session key from the shared
// secret using the given label (SGP.22 labels: 1=enc, 2=mac, 3=kek, 4=dek).
// The context is the label byte right-padded to 16 bytes.
func DeriveSessionKey(shared []byte, label byte) ([16]byte, error) {
	var out [16]byte
	ctx := make([]byte, 16)
	ctx[0] = label
	derived, err := KDF(shared, ctx, 16)
	if err != nil {
		return out, err
	}
	copy(out[:], derived)
	return out, nil
}

// SessionKeys holds the four SGP.22-style session keys derived from the
// ECDH shared secret.
type SessionKeys struct {
	Enc [16]byte // K1 - transport encryption
	Mac [16]byte // K2 - message integrity
	Kek [16]byte // K3 - key encryption (reserved, lab)
	Dek [16]byte // K4 - profile payload encryption
}

// DeriveSessionKeys derives K1..K4 from the ECDH shared secret.
func DeriveSessionKeys(shared []byte) (SessionKeys, error) {
	var keys SessionKeys
	var err error
	if keys.Enc, err = DeriveSessionKey(shared, 1); err != nil {
		return keys, err
	}
	if keys.Mac, err = DeriveSessionKey(shared, 2); err != nil {
		return keys, err
	}
	if keys.Kek, err = DeriveSessionKey(shared, 3); err != nil {
		return keys, err
	}
	if keys.Dek, err = DeriveSessionKey(shared, 4); err != nil {
		return keys, err
	}
	return keys, nil
}

// Encrypt encrypts pt with AES-128-CBC + PKCS#7 under the given key and
// returns IV || ciphertext (random IV prepended).
func Encrypt(key, pt []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, fmt.Errorf("encrypt: key must be 16 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(pt, block.BlockSize())
	iv := make([]byte, block.BlockSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	out := make([]byte, len(iv)+len(padded))
	copy(out, iv)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out[len(iv):], padded)
	return out, nil
}

// Decrypt reverses Encrypt. Input must be IV || ciphertext.
func Decrypt(key, ct []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, fmt.Errorf("decrypt: key must be 16 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ct) < block.BlockSize()*2 || len(ct)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("decrypt: bad ciphertext length %d", len(ct))
	}
	iv := ct[:block.BlockSize()]
	body := ct[block.BlockSize():]
	out := make([]byte, len(body))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, body)
	unpadded, err := pkcs7Unpad(out, block.BlockSize())
	if err != nil {
		return nil, err
	}
	return unpadded, nil
}

func pkcs7Pad(data []byte, size int) []byte {
	pad := size - len(data)%size
	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}

func pkcs7Unpad(data []byte, size int) ([]byte, error) {
	if len(data) == 0 || len(data)%size != 0 {
		return nil, errors.New("unpad: bad length")
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > size || pad > len(data) {
		return nil, errors.New("unpad: bad padding")
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, errors.New("unpad: bad padding bytes")
		}
	}
	return data[:len(data)-pad], nil
}

// ECDHKeyPair generates a P-256 (secp256r1) keypair. SGP.22 requires P-256
// when P-256 is supported by the eUICC; the lab stack uses P-256 only.
func ECDHKeyPair() (*ecdh.PrivateKey, error) {
	return ecdh.P256().GenerateKey(rand.Reader)
}

// ECDHPublic marshals a public key to its uncompressed SEC1 65-byte form
// (the wire format used in ES9+ messages).
func ECDHPublic(pub *ecdh.PublicKey) []byte {
	return pub.Bytes()
}

// ECDHShared computes the ECDH shared secret between priv and peer.
func ECDHShared(priv *ecdh.PrivateKey, peer []byte) ([]byte, error) {
	pub, err := ecdh.P256().NewPublicKey(peer)
	if err != nil {
		return nil, fmt.Errorf("ecdh: bad peer key: %w", err)
	}
	return priv.ECDH(pub)
}

// VerifyMAC constant-time-compares a computed MAC against the received one.
func VerifyMAC(expected, got []byte) bool {
	return len(expected) == len(got) && subtle.ConstantTimeCompare(expected, got) == 1
}
