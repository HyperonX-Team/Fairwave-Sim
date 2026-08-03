package euicc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/smdp"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

const testActivationCode = "EUICCTEST1234"

func e2eServer(t *testing.T, imsi string) (*httptest.Server, *smdp.MemStore) {
	t.Helper()
	store := smdp.NewMemStore()
	sub, err := simprov.LoadTestVector(imsi)
	if err != nil {
		t.Fatal(err)
	}
	// An activation code resolves to ONE stable profile (like a production
	// SM-DP+ registry): mint it once and return the same record.
	p, err := profile.NewLabProfile(sub, "smdp.fairwave.test")
	if err != nil {
		t.Fatal(err)
	}
	srv := smdp.NewServer("smdp.fairwave.test", store, func(ac string) (*profile.Profile, error) {
		if ac != testActivationCode {
			return nil, smdp.ErrUnknownActivationCode
		}
		return p, nil
	})
	return httptest.NewServer(srv.Handler()), store
}

func download(t *testing.T, e *EUICC, url string) *InstalledProfile {
	t.Helper()
	p, err := e.Download(context.Background(), url, "LPA:1$smdp.fairwave.test$"+testActivationCode, nil)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEnableDisableList(t *testing.T) {
	hs, _ := e2eServer(t, "999991234567001")
	defer hs.Close()
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	p := download(t, e, hs.URL)
	if err := e.DisableProfile(p.Profile.ICCID); err != nil {
		t.Fatal(err)
	}
	if e.Profiles[p.Profile.ICCID].Enabled {
		t.Fatal("profile must be disabled")
	}
	if err := e.EnableProfile(p.Profile.ICCID); err != nil {
		t.Fatal(err)
	}
	if !e.Profiles[p.Profile.ICCID].Enabled {
		t.Fatal("profile must be enabled again")
	}
	if err := e.DisableProfile("89000000000000000000"); err == nil {
		t.Fatal("disabling an unknown profile must fail")
	}
	if got := len(e.ListProfiles()); got != 1 {
		t.Fatalf("list = %d, want 1", got)
	}
}

func TestTwoEUICCDownloadsIndependent(t *testing.T) {
	hs, _ := e2eServer(t, "999991234567001")
	defer hs.Close()
	e1, _ := New()
	e2, _ := New()
	p1 := download(t, e1, hs.URL)
	p2 := download(t, e2, hs.URL)
	// One activation code resolves to one profile record: both eUICCs must
	// install the same ICCID (duplicate handling is per-eUICC).
	if p1.Profile.ICCID != p2.Profile.ICCID {
		t.Fatal("an activation code must resolve to a single profile ICCID")
	}
	if e1.EID == e2.EID {
		t.Fatal("two eUICCs must have distinct EIDs")
	}
	if p1.Profile.EID != "" {
		t.Fatal("unpinned profile must not carry an EID")
	}
}

func TestEIDPinnedProfile(t *testing.T) {
	sub, _ := simprov.LoadTestVector("999991234567002")
	store := smdp.NewMemStore()
	e, _ := New()
	srv := smdp.NewServer("smdp.fairwave.test", store, func(ac string) (*profile.Profile, error) {
		if ac != testActivationCode {
			return nil, smdp.ErrUnknownActivationCode
		}
		return profile.NewLabProfile(sub, "smdp.fairwave.test", profile.WithEID(e.EID))
	})
	hs := httptest.NewServer(srv.Handler())
	defer hs.Close()

	p := download(t, e, hs.URL)
	if p.Profile.EID != e.EID {
		t.Fatalf("pinned EID = %s, want %s", p.Profile.EID, e.EID)
	}

	// A different eUICC must be refused.
	other, _ := New()
	_, err := other.Download(context.Background(), hs.URL, "LPA:1$smdp.fairwave.test$"+testActivationCode, hs.Client())
	if err == nil {
		t.Fatal("EID-pinned profile must be refused for another eUICC")
	}
	if got := len(other.ListProfiles()); got != 0 {
		t.Fatalf("other eUICC installed %d profiles", got)
	}
}

func TestDuplicateDownloadRejected(t *testing.T) {
	hs, _ := e2eServer(t, "999991234567001")
	defer hs.Close()
	e, _ := New()
	p := download(t, e, hs.URL)
	_, err := e.Download(context.Background(), hs.URL, "LPA:1$smdp.fairwave.test$"+testActivationCode, hs.Client())
	if err == nil {
		t.Fatal("duplicate install must fail")
	}
	if p.Profile.ICCID != e.ListProfiles()[0].Profile.ICCID {
		t.Fatal("original profile must remain installed")
	}
}

func TestDownloadServerUnreachable(t *testing.T) {
	e, _ := New()
	client := &http.Client{}
	_, err := e.Download(context.Background(), "http://127.0.0.1:1", "LPA:1$smdp.fairwave.test$"+testActivationCode, client)
	if err == nil {
		t.Fatal("expected failure for unreachable server")
	}
	if got := len(e.ListProfiles()); got != 0 {
		t.Fatalf("profiles installed = %d, want 0", got)
	}
}
