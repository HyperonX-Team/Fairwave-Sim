package smdp

import (
	"crypto/ecdh"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// fileSession is the serializable on-disk form of a Session. It differs
// from Session only in that the ephemeral ECDH private key is stored as
// bytes so it survives a reboot; the in-memory Session reconstructs it.
type fileSession struct {
	TransactionID  string
	ActivationCode string
	EID            string
	ICCID          string
	Status         SessionStatus
	SeqCounter     int

	EuiccChallenge  []byte
	EuiccEKPb       []byte
	ServerChallenge []byte
	ServerEphemeral []byte // PKCS8 bytes of the P-256 private key
	Keys            [4][16]byte

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (f *fileSession) toSession() (*Session, error) {
	s := &Session{
		TransactionID:  f.TransactionID,
		ActivationCode: f.ActivationCode,
		EID:            f.EID,
		ICCID:          f.ICCID,
		Status:         f.Status,
		SeqCounter:     f.SeqCounter,
		EuiccChallenge:  f.EuiccChallenge,
		EuiccEKPb:       f.EuiccEKPb,
		ServerChallenge: f.ServerChallenge,
		CreatedAt:       f.CreatedAt,
		UpdatedAt:       f.UpdatedAt,
	}
	s.Keys.Enc = f.Keys[0]
	s.Keys.Mac = f.Keys[1]
	s.Keys.Kek = f.Keys[2]
	s.Keys.Dek = f.Keys[3]
	if len(f.ServerEphemeral) > 0 {
		priv, err := newPrivateKey(f.ServerEphemeral)
		if err != nil {
			return nil, fmt.Errorf("load ephemeral key: %w", err)
		}
		s.ServerEphemeral = priv
	}
	return s, nil
}

func fromSession(s *Session) (*fileSession, error) {
	f := &fileSession{
		TransactionID:  s.TransactionID,
		ActivationCode: s.ActivationCode,
		EID:            s.EID,
		ICCID:          s.ICCID,
		Status:         s.Status,
		SeqCounter:     s.SeqCounter,
		EuiccChallenge:  s.EuiccChallenge,
		EuiccEKPb:       s.EuiccEKPb,
		ServerChallenge: s.ServerChallenge,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
	f.Keys[0] = s.Keys.Enc
	f.Keys[1] = s.Keys.Mac
	f.Keys[2] = s.Keys.Kek
	f.Keys[3] = s.Keys.Dek
	if s.ServerEphemeral != nil {
		f.ServerEphemeral = privateKeyBytes(s.ServerEphemeral)
	}
	return f, nil
}

// fileStore is a directory-backed Store implementation. Each session is a
// JSON file named <dir>/<transaction>.json (0600). It is a drop-in
// replacement for MemStore: restarting the server no longer loses a
// half-finished download exchange.
type fileStore struct {
	mu  sync.Mutex
	dir string
}

// NewFileStore returns a Store that persists sessions under dir (created
// with 0700 if missing).
func NewFileStore(dir string) (Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &fileStore{dir: dir}, nil
}

func (f *fileStore) path(tx string) string { return filepath.Join(f.dir, tx+".json") }

func (f *fileStore) CreateSession(s *Session) error {
	if s == nil {
		return errors.New("smdp: nil session")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	fs, err := fromSession(s)
	if err != nil {
		return err
	}
	return writeJSONFile(f.path(s.TransactionID), fs, 0o600)
}

func (f *fileStore) GetSession(transactionID string) (*Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := os.ReadFile(f.path(transactionID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrUnknownSession, transactionID)
		}
		return nil, err
	}
	var fs fileSession
	if err := json.Unmarshal(data, &fs); err != nil {
		return nil, fmt.Errorf("smdp: corrupt session file: %w", err)
	}
	return fs.toSession()
}

func (f *fileStore) UpdateSession(s *Session) error {
	return f.CreateSession(s)
}

func (f *fileStore) DeleteSession(transactionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := os.Remove(f.path(transactionID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// newPrivateKey reconstructs a P-256 private key from its PKCS8 bytes.
func newPrivateKey(b []byte) (*ecdh.PrivateKey, error) {
	return ecdh.P256().NewPrivateKey(b)
}

// privateKeyBytes serializes a P-256 private key to PKCS8 bytes.
func privateKeyBytes(k *ecdh.PrivateKey) []byte {
	return k.Bytes()
}

// writeJSONFile atomically writes data to path with the given mode.
func writeJSONFile(path string, v any, mode os.FileMode) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
