// Package activation implements eSIM activation codes (GSMA SGP.22 form:
// "LPA:1$<smdp-address>$<activation-code>$<confirmation-code?>") and their
// QR rendering. The QR content is the activation code string itself; a
// phone's camera + LPA scans it to begin a download (ES9+).
package activation

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// ErrMalformed is returned when an activation code string is invalid.
var ErrMalformed = errors.New("activation: malformed activation code")

// Code is a parsed activation code.
type Code struct {
	SMDPAddress      string // host[:port] of the SM-DP+
	ActivationCode   string // one-time code resolving to a profile
	ConfirmationCode string // optional
}

// Parse parses the SGP.22 activation-code string. The confirmation code is
// optional and separated by a "$" field which is empty when absent, so the
// canonical forms are LPA:1$addr$code, LPA:1$addr$code$$cc and (tolerated)
// LPA:1$addr$code$cc.
func Parse(s string) (*Code, error) {
	rest, ok := strings.CutPrefix(s, "LPA:1$")
	if !ok {
		return nil, fmt.Errorf("%w: missing LPA:1$ prefix", ErrMalformed)
	}
	parts := strings.Split(rest, "$")
	if len(parts) < 2 || len(parts) > 4 {
		return nil, fmt.Errorf("%w: expected LPA:1$smdp$code[$$confirmation]", ErrMalformed)
	}
	if parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("%w: empty smdp address or code", ErrMalformed)
	}
	for _, c := range parts[1] {
		if !isCodeChar(byte(c)) {
			return nil, fmt.Errorf("%w: illegal character %q in activation code", ErrMalformed, string(c))
		}
	}
	c := &Code{SMDPAddress: parts[0], ActivationCode: parts[1]}
	switch {
	case len(parts) == 3:
		c.ConfirmationCode = parts[2]
	case len(parts) == 4:
		c.ConfirmationCode = parts[3]
	}
	return c, nil
}

// New builds a Code for a random activation token of the given length.
func New(smdpAddress, activationCode string) *Code {
	return &Code{SMDPAddress: smdpAddress, ActivationCode: activationCode}
}

// String renders the SGP.22 activation-code string (canonical "$$" form
// when a confirmation code is present).
func (c *Code) String() string {
	out := "LPA:1$" + c.SMDPAddress + "$" + c.ActivationCode
	if c.ConfirmationCode != "" {
		out += "$$" + c.ConfirmationCode
	}
	return out
}

// QR renders the activation code as a PNG QR (256x256, medium error
// correction).
func (c *Code) QR() ([]byte, error) {
	png, err := qrcode.Encode(c.String(), qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("activation: qr: %w", err)
	}
	return png, nil
}

// ValidToken reports whether s is usable as an activation code: 8-64
// alphanumeric characters, no '$'.
func ValidToken(s string) bool {
	if len(s) < 8 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isCodeChar(s[i]) {
			return false
		}
	}
	return true
}

// GenerateToken mints a random activation token (base32 alphabet, no
// padding, n characters, cryptographically random).
func GenerateToken(n int) (string, error) {
	if n < 8 || n > 64 {
		return "", fmt.Errorf("activation: token length must be in [8,64], got %d", n)
	}
	rawLen := (n*5 + 7) / 8
	raw := make([]byte, rawLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	return enc.EncodeToString(raw)[:n], nil
}

func isCodeChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
