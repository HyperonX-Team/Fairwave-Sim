package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// passphraseHeader carries the optional backup passphrase. It is only
// meaningful over TLS in production; in the lab the archive is obfuscated,
// not strongly protected - treat it as such (see docs/ops).
const passphraseHeader = "X-Fairwave-Passphrase"

// createBackup tars + gzips the whole store directory (state JSON files,
// eSIM registry, identity, admin token) and, when a passphrase is given,
// encrypts the stream with AES-256-GCM (key = SHA-256 of the passphrase).
func (s *Server) createBackup(passphrase string) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	dir := s.store.DataDir()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("backup: walk %s: %w", dir, err)
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	if passphrase == "" {
		return buf.Bytes(), nil
	}
	return encryptArchive(buf.Bytes(), passphrase)
}

// restoreBackup validates the archive before touching the data dir: it is
// unpacked into a fresh staging dir that is only swapped in once the whole
// archive has been read successfully.
func (s *Server) restoreBackup(data []byte, passphrase string) error {
	raw := data
	if passphrase != "" {
		dec, err := decryptArchive(raw, passphrase)
		if err != nil {
			return err
		}
		raw = dec
	}
	dir := s.store.DataDir()
	staging, err := os.MkdirTemp("", "fairwave-restore-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("restore: not a gzip archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("restore: read archive: %w", err)
		}
		name := filepath.Clean(hdr.Name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("restore: refusing unsafe path %q", hdr.Name)
		}
		dst := filepath.Join(staging, name)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(dst, 0o750); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		f, err := os.Create(dst)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return fmt.Errorf("restore: %s: %w", hdr.Name, err)
		}
		f.Close()
	}

	// Swap the validated staging dir in. The control plane keeps running on
	// the in-memory state; a restart is required for the restored data to
	// take effect (the CLI prints this).
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.Rename(staging, dir)
}

func encryptArchive(plain []byte, passphrase string) ([]byte, error) {
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nonce, nonce, plain, nil)
	out := make([]byte, 0, len(sealed)+8)
	out = append(out, []byte("FWBAK1\x00\x00")...) // magic + version
	out = append(out, sealed...)
	return out, nil
}

func decryptArchive(data []byte, passphrase string) ([]byte, error) {
	const magic = "FWBAK1\x00\x00"
	if len(data) < len(magic) || !bytes.Equal(data[:len(magic)], []byte(magic)) {
		return nil, fmt.Errorf("restore: not an encrypted Fairwave backup (missing magic)")
	}
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	sealed := data[len(magic):]
	if len(sealed) < gcm.NonceSize() {
		return nil, fmt.Errorf("restore: encrypted payload too short")
	}
	nonce := sealed[:gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, sealed[gcm.NonceSize():], nil)
	if err != nil {
		return nil, fmt.Errorf("restore: decrypt failed (wrong passphrase?): %w", err)
	}
	return plain, nil
}

// ---- handlers ----

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	data, err := s.createBackup(r.Header.Get(passphraseHeader))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "backup", err.Error())
		return
	}
	s.auditReq(r, "backup", "", "archive created")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="fairwave-backup-%s.tar.gz"`, s.now().UTC().Format("20060102")))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 64<<20)) // 64 MB cap
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(data) == 0 {
		writeErr(w, http.StatusBadRequest, "bad_request", "empty archive")
		return
	}
	if err := s.restoreBackup(data, r.Header.Get(passphraseHeader)); err != nil {
		writeErr(w, http.StatusBadRequest, "restore", err.Error())
		return
	}
	s.auditReq(r, "restore", "", "state restored; restart the control plane to load it")
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "restored",
		"message": "restart fairwave-control to load the restored state",
	})
}
