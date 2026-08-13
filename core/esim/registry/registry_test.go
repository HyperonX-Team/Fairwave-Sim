package registry

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

func labProfile(t *testing.T) *profile.Profile {
	t.Helper()
	sub, err := simprov.LoadTestVector("999991234567001")
	if err != nil {
		t.Fatal(err)
	}
	p, err := profile.NewLabProfile(sub, "smdp.fairwave.test")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEntryGetterAndExpiry(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(filepath.Join(dir, "reg.json"))
	if err != nil {
		t.Fatal(err)
	}
	exp := time.Now().Add(time.Hour)
	if err := r.AddWithExpiry("TOKEN1234", labProfile(t), exp); err != nil {
		t.Fatal(err)
	}
	e, err := r.Entry("TOKEN1234")
	if err != nil {
		t.Fatal(err)
	}
	if e.ExpiresAt == nil || !e.ExpiresAt.Equal(exp) {
		t.Fatalf("expiry not stored: %v", e.ExpiresAt)
	}

	// before expiry: allowed
	if _, err := r.ResolvePolicy("TOKEN1234", true, exp.Add(-time.Minute)); err != nil {
		t.Fatalf("resolve before expiry: %v", err)
	}
	// after expiry: refused
	if _, err := r.ResolvePolicy("TOKEN1234", true, exp.Add(time.Minute)); !errors.Is(err, ErrActivationCodeExpired) {
		t.Fatalf("resolve after expiry: %v, want ErrActivationCodeExpired", err)
	}
}

func TestSingleUseEnforcement(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(filepath.Join(dir, "reg.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Add("TOKEN5678", labProfile(t)); err != nil {
		t.Fatal(err)
	}
	if err := r.MarkDownloaded("TOKEN5678"); err != nil {
		t.Fatal(err)
	}
	// single-use refused after download
	if _, err := r.ResolvePolicy("TOKEN5678", true, time.Now()); !errors.Is(err, ErrActivationCodeUsed) {
		t.Fatalf("second resolve: %v, want ErrActivationCodeUsed", err)
	}
	// non-single-use still resolves
	if _, err := r.ResolvePolicy("TOKEN5678", false, time.Now()); err != nil {
		t.Fatalf("resolve without single-use: %v", err)
	}
	// plain Resolve is unaffected (back-compat)
	if _, err := r.Resolve("TOKEN5678"); err != nil {
		t.Fatalf("plain resolve: %v", err)
	}
}
