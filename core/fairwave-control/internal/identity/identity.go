// Package identity manages the node's long-lived identity: an Ed25519
// keypair, a self-signed x509 certificate for mTLS, and bootstrap tokens
// for enrollment. The private key never leaves the box and is stored with
// 0600 permissions.
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// Identity is the node's persistent identity material.
type Identity struct {
	dir     string
	PrivKey ed25519.PrivateKey
	PubKey  ed25519.PublicKey
	CertPEM []byte
	Cert    *x509.Certificate
	ID      string // short id = sha256(pubkey)[:12]
}

// LoadOrCreate loads identity from dir, or generates a fresh one.
func LoadOrCreate(dir string) (*Identity, error) {
	keyPath := filepath.Join(dir, "node.key")
	certPath := filepath.Join(dir, "node.crt")
	if data, err := os.ReadFile(keyPath); err == nil {
		block, _ := pem.Decode(data)
		if block != nil {
			priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err == nil {
				ed, ok := priv.(ed25519.PrivateKey)
				if ok {
					id := &Identity{dir: dir, PrivKey: ed, PubKey: ed.Public().(ed25519.PublicKey)}
					if certData, err := os.ReadFile(certPath); err == nil {
						id.CertPEM = certData
						if cb, _ := pem.Decode(certData); cb != nil {
							if c, err := x509.ParseCertificate(cb.Bytes); err == nil {
								id.Cert = c
							}
						}
					}
					id.ID = ShortID(id.PubKey)
					return id, nil
				}
			}
		}
		return nil, fmt.Errorf("corrupt node.key: regenerate by deleting the file (loses node identity!)")
	}
	// fresh identity
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	id := &Identity{dir: dir, PrivKey: priv, PubKey: priv.Public().(ed25519.PublicKey)}
	id.ID = ShortID(id.PubKey)

	// self-signed cert for mTLS (control plane ↔ agent/CLI)
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "fairwave-node-" + id.ID, Organization: []string{"HyperonX Fairwave"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, id.PubKey, id.PrivKey)
	if err != nil {
		return nil, err
	}
	id.Cert = tpl
	id.CertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalPKCS8PrivateKey(id.PrivKey)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(certPath, id.CertPEM, 0o644); err != nil {
		return nil, err
	}
	return id, nil
}

// ShortID derives a human-usable 12-hex-char id from a public key.
func ShortID(pub ed25519.PublicKey) string {
	// We hash to avoid leaking key material in the id.
	// (This is not cryptography-critical: it's an identifier.)
	hash := sha256.Sum256(pub)
	return hex.EncodeToString(hash[:6])
}

// Sign signs data with the node key.
func (i *Identity) Sign(data []byte) []byte {
	return ed25519.Sign(i.PrivKey, data)
}

// Verify checks a signature against a public key.
func Verify(pub ed25519.PublicKey, data, sig []byte) bool {
	return ed25519.Verify(pub, data, sig)
}

// GenerateBootstrapToken creates a random enrollment token.
func GenerateBootstrapToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
