// Package profile models the eSIM carrier profile: the metadata and USIM
// application files that an SM-DP+ encrypts into a Bound Profile Package
// (BPP) and that a target eUICC installs.
//
// This is a LAB implementation. The EF set is the minimal subset needed to
// install and (with the software eUICC) attach to the Fairwave lab network.
// Byte-level details marked "lab-defined" follow the shape of 3GPP
// TS 131.102 / GSMA SGP.22 but are ours until validated against GSMA
// conformance tooling and a physical phone (docs/adr/0013-esim.md).
package profile

import (
	"crypto/rand"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

// Version is the lab profile payload format version.
const Version = 1

// Profile is the operator-facing carrier profile delivered to an eUICC.
// KI/OPc are part of the encrypted payload and never cross the wire in
// plaintext; the API surface only exposes metadata.
type Profile struct {
	Version             int       `json:"version"`
	ICCID               string    `json:"iccid"` // 20 digits, Luhn check digit
	EID                 string    `json:"eid"`   // target eUICC, 32 digits
	ProfileName         string    `json:"profile_name"`
	ProfileOwner        string    `json:"profile_owner"`
	ServiceProviderName string    `json:"service_provider_name"`
	SMDPID              string    `json:"smdp_id"`
	Class               string    `json:"class"` // lab | prod
	IMSI                string    `json:"imsi"`  // 15 digits
	MSISDN              string    `json:"msisdn"`
	APN                 string    `json:"apn"`
	KI                  string    `json:"ki,omitempty"`  // 16 bytes hex
	OPc                 string    `json:"opc,omitempty"` // 16 bytes hex
	AMF                 string    `json:"amf,omitempty"`
	SQN                 string    `json:"sqn,omitempty"`
	UST                 []byte    `json:"ust,omitempty"` // EF.UST content
	AD                  []byte    `json:"ad,omitempty"`  // EF.AD content
	CreatedAt           time.Time `json:"created_at"`
	ExpiresAt           time.Time `json:"expires_at"`
}

// Option is a functional option for NewLabProfile.
type Option func(*Profile)

// WithProfileName sets the display name of the profile.
func WithProfileName(name string) Option {
	return func(p *Profile) { p.ProfileName = name }
}

// WithExpiry sets the profile expiry.
func WithExpiry(exp time.Time) Option {
	return func(p *Profile) { p.ExpiresAt = exp }
}

// WithEID pins the profile to a specific eUICC.
func WithEID(eid string) Option {
	return func(p *Profile) { p.EID = eid }
}

// NewLabProfile builds a lab profile from a provisioned SIM subscriber.
// It derives a fresh ICCID and copies the (dummy) Milenage credentials into
// the encrypted payload. EID stays empty until a target eUICC is chosen.
func NewLabProfile(sub simprov.Subscriber, smdpID string, opts ...Option) (*Profile, error) {
	if sub.Class != "lab" {
		return nil, fmt.Errorf("profile: only lab-class subscribers can mint eSIM profiles, got %q", sub.Class)
	}
	iccid, err := NewICCID("99901")
	if err != nil {
		return nil, err
	}
	p := &Profile{
		Version:             Version,
		ICCID:               iccid,
		ProfileName:         "Fairwave Lab " + sub.MSISDN,
		ProfileOwner:        "Fairwave Lab",
		ServiceProviderName: "Fairwave",
		SMDPID:              smdpID,
		Class:               sub.Class,
		IMSI:                sub.IMSI,
		MSISDN:              sub.MSISDN,
		APN:                 sub.APN,
		KI:                  sub.Ki,
		OPc:                 sub.OPc,
		AMF:                 sub.AMF,
		SQN:                 sub.SQN,
		UST:                 []byte{0x00, 0x00, 0x00, 0x00, 0x00},
		AD:                  []byte{0x00, 0x00},
		CreatedAt:           time.Now().UTC(),
		ExpiresAt:           time.Now().UTC().Add(365 * 24 * time.Hour),
	}
	for _, o := range opts {
		o(p)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

// Validate checks structural invariants of the profile.
func (p *Profile) Validate() error {
	if p.ICCID == "" || !ValidLuhn(p.ICCID) || len(p.ICCID) < 19 {
		return fmt.Errorf("profile: bad ICCID %q", p.ICCID)
	}
	if p.EID != "" && (len(p.EID) != 32 || !ValidLuhn(p.EID)) {
		return fmt.Errorf("profile: bad EID %q", p.EID)
	}
	if len(p.IMSI) != 15 {
		return fmt.Errorf("profile: IMSI must be 15 digits, got %q", p.IMSI)
	}
	if p.Class != "lab" && p.Class != "prod" {
		return fmt.Errorf("profile: class must be lab or prod, got %q", p.Class)
	}
	if p.Class == "lab" && (len(p.KI) != 32 || len(p.OPc) != 32) {
		return fmt.Errorf("profile: lab profile needs 16-byte hex KI/OPc")
	}
	return nil
}

// Luhn computes the standard mod-10 check digit for data.
func Luhn(data string) int {
	sum := 0
	double := true
	for i := len(data) - 1; i >= 0; i-- {
		d := int(data[i] - '0')
		if d < 0 || d > 9 {
			return -1
		}
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return (10 - sum%10) % 10
}

// ValidLuhn reports whether the last digit of s is the Luhn check of the
// preceding digits.
func ValidLuhn(s string) bool {
	if len(s) < 2 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return Luhn(s[:len(s)-1]) == int(s[len(s)-1]-'0')
}

// NewICCID mints a 20-digit ICCID for the given MNO code (e.g. "99901").
func NewICCID(mno string) (string, error) {
	if len(mno) > 14 {
		return "", fmt.Errorf("iccid: mno code too long: %q", mno)
	}
	payload := "89" + mno + randDigits(19-2-len(mno))
	return payload + fmt.Sprintf("%d", Luhn(payload)), nil
}

// NewEID mints a 32-digit EID for a software or hardware eUICC. Lab-defined:
// 30 data digits after the GSMA "8904" test prefix plus two Luhn check
// digits.
func NewEID() (string, error) {
	data := "8904" + randDigits(26)
	return data + fmt.Sprintf("%d%d", Luhn(data), Luhn(data+fmt.Sprintf("%d", Luhn(data)))), nil
}

func randDigits(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("0", n)
	}
	for i := range buf {
		buf[i] = '0' + buf[i]%10
	}
	return string(buf)
}

// ICCIDFile returns the EF.ICCID (2F02) BCD content (10 bytes).
func (p *Profile) ICCIDFile() ([]byte, error) {
	bcd, err := bcdEncodeSwapped(p.ICCID)
	if err != nil {
		return nil, err
	}
	if len(bcd) != 10 {
		return nil, fmt.Errorf("iccid file: expected 10 bytes, got %d", len(bcd))
	}
	return bcd, nil
}

// IMSIFile returns the EF.IMSI (6F07) content: length byte + 8 bytes of
// swapped-BCD digits (last nibble set to 0xF, TS 131.102 layout).
func (p *Profile) IMSIFile() ([]byte, error) {
	if len(p.IMSI) != 15 {
		return nil, fmt.Errorf("imsi file: IMSI must be 15 digits, got %q", p.IMSI)
	}
	nibbles := make([]byte, 16)
	for i := 0; i < 15; i++ {
		d := p.IMSI[i] - '0'
		if d > 9 {
			return nil, fmt.Errorf("imsi file: non-digit in %q", p.IMSI)
		}
		nibbles[i] = d
	}
	nibbles[15] = 0xF
	out := make([]byte, 9)
	out[0] = 0x09
	for i := 0; i < 8; i++ {
		out[1+i] = nibbles[2*i+1]<<4 | nibbles[2*i]
	}
	return out, nil
}

// bcdEncodeSwapped encodes digits as swapped-nibble BCD: (d1,d2) -> 0xd2d1.
func bcdEncodeSwapped(digits string) ([]byte, error) {
	if len(digits)%2 != 0 {
		return nil, fmt.Errorf("bcd: odd digit count %d", len(digits))
	}
	out := make([]byte, len(digits)/2)
	for i := 0; i < len(digits); i += 2 {
		hi := digits[i+1] - '0'
		lo := digits[i] - '0'
		if hi > 9 || lo > 9 {
			return nil, fmt.Errorf("bcd: non-digit in %q", digits)
		}
		out[i/2] = hi<<4 | lo
	}
	return out, nil
}

// MarshalPayload serializes the profile to its canonical JSON form; this is
// the plaintext that gets encrypted into the BPP.
func (p *Profile) MarshalPayload() ([]byte, error) {
	return json.Marshal(p)
}

// UnmarshalPayload parses a payload produced by MarshalPayload.
func UnmarshalPayload(data []byte) (*Profile, error) {
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("profile payload: %w", err)
	}
	return &p, nil
}

// EncryptedProfile is the SGP.22-shaped envelope for a bound profile.
// Lab-defined wire layout: seqCounter || ciphertext with a full-length MAC
// over those two values.
type EncryptedProfile struct {
	SeqCounter int
	Ciphertext []byte
	MAC        []byte
}

type encryptedProfileASN1 struct {
	SeqCounter int
	Ciphertext []byte
	MAC        []byte
}

type bppASN1 struct {
	Version int
	Profile encryptedProfileASN1
}

// EncodeBPP serializes the bound profile package (DER envelope).
func EncodeBPP(ep EncryptedProfile) ([]byte, error) {
	return asn1.Marshal(bppASN1{
		Version: Version,
		Profile: encryptedProfileASN1(ep),
	})
}

// EncodeMACInput builds the MAC input for a bound profile package: the DER
// encoding of SEQUENCE { seqCounter, ciphertext }. Both SM-DP+ and eUICC
// must compute this identically (lab-defined, see EncodeBPP).
func EncodeMACInput(seqCounter int, ciphertext []byte) ([]byte, error) {
	return asn1.Marshal(struct {
		SeqCounter int
		Ciphertext []byte
	}{seqCounter, ciphertext})
}

// DecodeBPP parses a bound profile package produced by EncodeBPP.
func DecodeBPP(data []byte) (*EncryptedProfile, error) {
	var b bppASN1
	rest, err := asn1.Unmarshal(data, &b)
	if err != nil {
		return nil, fmt.Errorf("bpp: %w", err)
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("bpp: %d trailing bytes", len(rest))
	}
	if b.Version != Version {
		return nil, fmt.Errorf("bpp: unsupported version %d", b.Version)
	}
	return &EncryptedProfile{
		SeqCounter: b.Profile.SeqCounter,
		Ciphertext: b.Profile.Ciphertext,
		MAC:        b.Profile.MAC,
	}, nil
}
