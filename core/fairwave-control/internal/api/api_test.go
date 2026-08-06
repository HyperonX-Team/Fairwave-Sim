package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/config"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/identity"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/store"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

type fakeChecker struct{}

func (fakeChecker) Check(country, band string, indoor bool, licenseRef string) Verdict {
	if country == "US" && band == "n48" && licenseRef != "" {
		return Verdict{Allowed: true, Reasons: nil}
	}
	return Verdict{Allowed: false, Reasons: []string{"denied by fake checker"}}
}

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Server.DataDir = dir
	cfg.Server.Mode = "lab"
	os.Setenv("FAIRWAVE_ADMIN_TOKEN", "test-token")
	t.Cleanup(func() { os.Unsetenv("FAIRWAVE_ADMIN_TOKEN") })
	id, err := identity.LoadOrCreate(filepath.Join(dir, "identity"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dir, "state"))
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, st, id, fakeChecker{}, nil)
	return srv, "test-token"
}

func doJSON(t *testing.T, h http.Handler, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestHealthzAndAuth(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	if w := doJSON(t, h, "GET", "/v1/healthz", "", nil); w.Code != 200 {
		t.Fatalf("healthz: %d", w.Code)
	}
	if w := doJSON(t, h, "GET", "/v1/status", "", nil); w.Code != 401 {
		t.Fatalf("status without token must 401, got %d", w.Code)
	}
	if w := doJSON(t, h, "GET", "/v1/status", tok, nil); w.Code != 200 {
		t.Fatalf("status with token: %d", w.Code)
	}
}

func TestNodeLifecycle(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/nodes", tok, api.Node{Name: "cafe1", Country: "LAB"})
	if w.Code != 201 {
		t.Fatalf("create node: %d %s", w.Code, w.Body.String())
	}
	var node api.Node
	_ = json.Unmarshal(w.Body.Bytes(), &node)
	if node.Phase != api.PhaseProvision {
		t.Fatalf("initial phase: %s", node.Phase)
	}

	// enroll
	w = doJSON(t, h, "POST", "/v1/nodes/"+node.ID+"/enroll", tok, api.EnrollRequest{BootstrapToken: "tok"})
	if w.Code != 200 {
		t.Fatalf("enroll: %d", w.Code)
	}
	_ = json.Unmarshal(w.Body.Bytes(), &node)
	if node.Phase != api.PhaseRegister {
		t.Fatalf("phase after enroll: %s", node.Phase)
	}

	// transition on-air
	w = doJSON(t, h, "POST", "/v1/lifecycle/transition", tok, api.LifecycleTransitionRequest{Phase: api.PhaseOnAir})
	if w.Code != 200 {
		t.Fatalf("transition: %d %s", w.Code, w.Body.String())
	}

	// skip is rejected
	w = doJSON(t, h, "POST", "/v1/lifecycle/transition", tok, api.LifecycleTransitionRequest{Phase: api.PhaseBreakout})
	if w.Code != 409 {
		t.Fatalf("skip transition must 409, got %d", w.Code)
	}
}

func TestSimIssueAndRevoke(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 2})
	if w.Code != 201 {
		t.Fatalf("issue: %d %s", w.Code, w.Body.String())
	}
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)
	if len(sims) != 2 {
		t.Fatalf("want 2 sims, got %d", len(sims))
	}
	if sims[0].Status != "issued" {
		t.Fatalf("status: %s", sims[0].Status)
	}

	w = doJSON(t, h, "POST", "/v1/sims/"+sims[0].IMSI+"/revoke", tok, nil)
	if w.Code != 200 {
		t.Fatalf("revoke: %d", w.Code)
	}
	var revoked api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &revoked)
	if revoked.Status != "revoked" {
		t.Fatalf("revoked status: %s", revoked.Status)
	}
}

func TestTxArmGates(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	// lab mode refuses regardless of ack
	w := doJSON(t, h, "POST", "/v1/tx/arm", tok, api.TxArmRequest{
		Country: "US", Band: "n48", Acknowledgment: "I hold authorization for this transmission",
	})
	var resp api.TxArmResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Armed {
		t.Fatal("lab mode must refuse TX arm")
	}

	// wrong ack phrase refused
	w = doJSON(t, h, "POST", "/v1/tx/arm", tok, api.TxArmRequest{Country: "US", Band: "n48", Acknowledgment: "no"})
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Armed {
		t.Fatal("wrong ack must refuse")
	}
}

func TestSpectrumCheck(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/spectrum/check", tok, api.SpectrumCheckRequest{Country: "US", Band: "n48", Indoor: true})
	var resp api.SpectrumCheckResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Allowed {
		t.Fatal("no license ref must deny")
	}

	w = doJSON(t, h, "POST", "/v1/spectrum/check", tok, api.SpectrumCheckRequest{Country: "US", Band: "n48", Indoor: true, LicenseRef: "SAS-GAA-1"})
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Allowed {
		t.Fatalf("with license ref should allow: %v", resp.Reasons)
	}
}

func TestPolicyRoundTrip(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	w := doJSON(t, h, "PUT", "/v1/policy", tok, api.Policy{LocalBreakout: false, MaxUEs: 64, APNs: []string{"internet"}})
	if w.Code != 200 {
		t.Fatalf("put policy: %d", w.Code)
	}
	var p api.Policy
	w = doJSON(t, h, "GET", "/v1/policy", tok, nil)
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	if p.MaxUEs != 64 || p.LocalBreakout {
		t.Fatalf("policy round trip: %+v", p)
	}
}

// recordingHSS captures Add/Remove calls and can be forced to fail.
type recordingHSS struct {
	added   []string
	removed []string
	failAdd bool
}

func (r *recordingHSS) Add(_ context.Context, sub simprov.Subscriber) error {
	if r.failAdd {
		return errors.New("hss down")
	}
	r.added = append(r.added, sub.IMSI)
	return nil
}

func (r *recordingHSS) Remove(_ context.Context, imsi string) error {
	r.removed = append(r.removed, imsi)
	return nil
}

func TestSIMIssueWritesBackToHSS(t *testing.T) {
	srv, tok := newTestServer(t)
	hss := &recordingHSS{}
	srv.hss = hss
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 2})
	if w.Code != 201 {
		t.Fatalf("issue: %d %s", w.Code, w.Body.String())
	}
	if len(hss.added) != 2 {
		t.Fatalf("hss.Add calls = %d, want 2", len(hss.added))
	}
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)
	if len(sims) != 2 {
		t.Fatalf("issued sims = %d, want 2", len(sims))
	}
	for i, sim := range sims {
		if sim.IMSI != hss.added[i] {
			t.Fatalf("hss write-back IMSI %s != issued %s", hss.added[i], sim.IMSI)
		}
	}

	imsi := sims[0].IMSI
	w = doJSON(t, h, "POST", "/v1/sims/"+imsi+"/revoke", tok, nil)
	if w.Code != 200 {
		t.Fatalf("revoke: %d %s", w.Code, w.Body.String())
	}
	if len(hss.removed) != 1 || hss.removed[0] != imsi {
		t.Fatalf("hss.Remove calls = %v, want [%s]", hss.removed, imsi)
	}
}

func TestSIMIssueFailsWhenHSSUnavailable(t *testing.T) {
	srv, tok := newTestServer(t)
	srv.hss = &recordingHSS{failAdd: true}
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 1})
	if w.Code != 502 {
		t.Fatalf("issue with dead HSS: %d, want 502", w.Code)
	}
	if n := len(srv.store.ListSIMs()); n != 0 {
		t.Fatalf("no SIM must be stored when HSS write-back fails, got %d", n)
	}
}
