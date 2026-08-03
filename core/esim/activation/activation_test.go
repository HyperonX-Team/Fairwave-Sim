package activation

import (
	"bytes"
	"testing"
)

func TestParseRoundTrip(t *testing.T) {
	in := "LPA:1$fairwave.local:8443$ABC123DEF456"
	c, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if c.SMDPAddress != "fairwave.local:8443" || c.ActivationCode != "ABC123DEF456" {
		t.Fatalf("parsed wrong: %+v", c)
	}
	if c.String() != in {
		t.Fatalf("round trip mismatch: %q != %q", c.String(), in)
	}
}

func TestParseWithConfirmationCode(t *testing.T) {
	for _, in := range []string{
		"LPA:1$smdp.example.com$ACTIVATION$$1234",
		"LPA:1$smdp.example.com$ACTIVATION$1234", // tolerated non-canonical
	} {
		c, err := Parse(in)
		if err != nil {
			t.Fatal(err)
		}
		if c.ConfirmationCode != "1234" {
			t.Fatalf("confirmation code = %q, want 1234", c.ConfirmationCode)
		}
		if c.String() != "LPA:1$smdp.example.com$ACTIVATION$$1234" {
			t.Fatalf("canonical render = %q", c.String())
		}
	}
}

func TestParseRejectsMalformed(t *testing.T) {
	bad := []string{
		"",
		"LPA:2$smdp$code",
		"LPA:1$",
		"LPA:1$$",
		"LPA:1$smdp$",
		"LPA:1$smdp$code$more$parts$x",
		"LPA:1$smdp$bad code with space",
		"LPA:1$smdp$cödé",
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Fatalf("expected error for %q", s)
		}
	}
}

func TestValidToken(t *testing.T) {
	for _, s := range []string{"ABC12345", "a1b2c3d4e5f6g7h8"} {
		if !ValidToken(s) {
			t.Fatalf("%q must be a valid token", s)
		}
	}
	for _, s := range []string{"short", "has space!", "has$dollar", "ümlaut", ""} {
		if ValidToken(s) {
			t.Fatalf("%q must be rejected", s)
		}
	}
}

func TestQRProducesPNG(t *testing.T) {
	c, err := Parse("LPA:1$fairwave.local$QRCODE12345")
	if err != nil {
		t.Fatal(err)
	}
	png, err := c.QR()
	if err != nil {
		t.Fatal(err)
	}
	if len(png) < 100 {
		t.Fatalf("png too small: %d bytes", len(png))
	}
	if !bytes.Equal(png[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		t.Fatal("not a PNG signature")
	}
}
