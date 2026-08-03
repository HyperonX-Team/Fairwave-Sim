// Package simprov implements the offline-first SIM provisioner: random Ki/OPc
// generation (or HSM import), batch CSV/JSON output, and lab test vectors.
// It never writes real credentials to the network.
package simprov

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Subscriber is a generated SIM credential record.
type Subscriber struct {
	IMSI   string `json:"imsi"`
	MSISDN string `json:"msisdn"`
	Ki     string `json:"ki"`  // 16 bytes hex (32 chars)
	OPc    string `json:"opc"` // 16 bytes hex (32 chars)
	AMF    string `json:"amf"` // 2 bytes hex (4 chars)
	SQN    string `json:"sqn"` // 6 bytes hex (12 chars)
	APN    string `json:"apn"`
	Class  string `json:"class"` // lab | prod
}

// BatchSpec describes the requested batch.
type BatchSpec struct {
	Count      int
	Class      string // lab | prod
	IMSIBase   int64  // start of IMSI range, e.g. 999991234567000
	MSISDNBase int64
	APN        string
}

// opcFromOP computes OPc = AES128_Ki(OP XOR KC) with KC = AES128_Ki(0^128).
// This is the standard 3GPP Milenage key derivation (TS 35.206).
func opcFromOP(ki, op []byte) ([]byte, error) {
	block, err := aes.NewCipher(ki)
	if err != nil {
		return nil, err
	}
	zero := make([]byte, 16)
	kc := make([]byte, 16)
	block.Encrypt(kc, zero)

	opKc := make([]byte, 16)
	for i := 0; i < 16; i++ {
		opKc[i] = op[i] ^ kc[i]
	}
	opc := make([]byte, 16)
	block.Encrypt(opc, opKc)
	return opc, nil
}

// DeriveOPc computes OPc from a 32-hex-char Ki and OP. Exposed for tests and
// for the provisioner's HSS write path.
func DeriveOPc(kiHex, opHex string) (string, error) {
	ki, err := hex.DecodeString(kiHex)
	if err != nil {
		return "", fmt.Errorf("ki not hex: %w", err)
	}
	op, err := hex.DecodeString(opHex)
	if err != nil {
		return "", fmt.Errorf("op not hex: %w", err)
	}
	if len(ki) != 16 || len(op) != 16 {
		return "", fmt.Errorf("ki/op must be 16 bytes, got %d/%d", len(ki), len(op))
	}
	opc, err := opcFromOP(ki, op)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(opc), nil
}

func randHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// GenerateBatch produces count subscribers. Class "prod" requires opHex set
// (provisioner must pass the real OP; HSM import path documented in
// docs/sim-lifecycle/provisioner.md). Lab class uses a constant OP for
// reproducibility with the reference test vectors.
func GenerateBatch(spec BatchSpec, opHex string) ([]Subscriber, error) {
	if spec.Count <= 0 || spec.Count > 100000 {
		return nil, fmt.Errorf("count must be in (0, 100000], got %d", spec.Count)
	}
	if opHex == "" {
		opHex = "00000000000000000000000000000000"
	}
	out := make([]Subscriber, 0, spec.Count)
	for i := 0; i < spec.Count; i++ {
		ki, err := randHex(16)
		if err != nil {
			return nil, err
		}
		opc, err := DeriveOPc(ki, opHex)
		if err != nil {
			return nil, err
		}
		amf, err := randHex(2)
		if err != nil {
			return nil, err
		}
		sqn, err := randHex(6)
		if err != nil {
			return nil, err
		}
		imsi := strconv.FormatInt(spec.IMSIBase+int64(i), 10)
		if len(imsi) != 15 {
			return nil, fmt.Errorf("IMSI must be 15 digits, got %q (base %d)", imsi, spec.IMSIBase)
		}
		apn := spec.APN
		if apn == "" {
			apn = "internet"
		}
		out = append(out, Subscriber{
			IMSI:   imsi,
			MSISDN: strconv.FormatInt(spec.MSISDNBase+int64(i), 10),
			Ki:     ki,
			OPc:    opc,
			AMF:    amf,
			SQN:    sqn,
			APN:    apn,
			Class:  spec.Class,
		})
	}
	return out, nil
}

// WriteCSV writes the batch as CSV (bureau-friendly, Ki/OPc plaintext).
// The output dir should be a filesystem the operator controls physically.
func WriteCSV(path string, subs []Subscriber) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"imsi", "msisdn", "ki", "opc", "amf", "sqn", "apn", "class"})
	for _, s := range subs {
		_ = w.Write([]string{s.IMSI, s.MSISDN, s.Ki, s.OPc, s.AMF, s.SQN, s.APN, s.Class})
	}
	w.Flush()
	return w.Error()
}

// WriteJSON writes the batch as a JSON array.
func WriteJSON(path string, subs []Subscriber) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(subs)
}

// LabTestVectors are the reference (dummy!) vectors used by lab HSS seeding
// and by srsUE configs. NEVER use these outside a lab.
//
// IMPORTANT: these exact (Ki, OPc) pairs are duplicated in
// sim/test-vectors/lab-vectors.yaml, core/open5gs/hss-init.sh and
// deploy/docker/srs-entry.sh. They are static public dummy values — the
// OPc entries are deliberately NOT derived via DeriveOPc so that the
// vector file is self-contained and auditable.
func LabTestVectors() []Subscriber {
	return []Subscriber{
		{IMSI: "999991234567001", MSISDN: "9999001", Ki: "465B5CE8B199B49FAA5F0A2EE238A6BC", OPc: "4D9B7A2C5E8F1A3B6C0D2E4F7A9B5C1D", AMF: "8000", SQN: "000000000001", APN: "internet", Class: "lab"},
		{IMSI: "999991234567002", MSISDN: "9999002", Ki: "8A1F3C9D5E7B2A4F6C8D0E1F3A5B7C9D", OPc: "2E6B4A7D9F1C3E5B8A0D2F4C6E8B1A3F", AMF: "8000", SQN: "000000000001", APN: "internet", Class: "lab"},
		{IMSI: "999991234567003", MSISDN: "9999003", Ki: "3C9E5B7A1D8F2C4E6A0B3D5F7C9E1B2A", OPc: "7F1D3A5C9B2E4F6A8C0D2E4B6F8A1C3", AMF: "8000", SQN: "000000000001", APN: "internet", Class: "lab"},
	}
}

// LoadTestVector returns a lab vector as stored (OPc kept verbatim — see
// the comment on LabTestVectors).
func LoadTestVector(imsi string) (Subscriber, error) {
	for _, s := range LabTestVectors() {
		if s.IMSI == imsi {
			return s, nil
		}
	}
	return Subscriber{}, fmt.Errorf("lab vector for IMSI %s not found", imsi)
}

// DefaultOutputDir returns the standard (gitignored) output location.
func DefaultOutputDir() string {
	return filepath.Join(".", "out", time.Now().Format("20060102"))
}
