package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/euicc"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/collector"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/config"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/identity"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/store"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

type fakeChecker struct{}

func (fakeChecker) Check(country, band string, _ bool, licenseRef string) Verdict {
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

func TestTelemetryAndHealth(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	temp := 41.2
	w := doJSON(t, h, "POST", "/v1/telemetry", tok, api.NodeHealth{
		NodeID: "node-1", Load1: 0.42, SDRTempC: &temp, GPSDO: true, Radio: "zmq", Watchdog: "ok",
	})
	if w.Code != 200 {
		t.Fatalf("telemetry: %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, h, "GET", "/v1/health", tok, nil)
	if w.Code != 200 {
		t.Fatalf("health: %d", w.Code)
	}
	var health []api.NodeHealth
	_ = json.Unmarshal(w.Body.Bytes(), &health)
	if len(health) != 1 || health[0].NodeID != "node-1" {
		t.Fatalf("health: %+v", health)
	}
	if !health[0].Up {
		t.Fatal("fresh heartbeat must be up")
	}
}

func TestSimSuspendResume(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 1})
	if w.Code != 201 {
		t.Fatalf("issue: %d", w.Code)
	}
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)
	imsi := sims[0].IMSI

	w = doJSON(t, h, "POST", "/v1/sims/"+imsi+"/suspend", tok, nil)
	if w.Code != 200 {
		t.Fatalf("suspend: %d %s", w.Code, w.Body.String())
	}
	var sim api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sim)
	if sim.Status != "suspended" {
		t.Fatalf("status after suspend: %s", sim.Status)
	}

	w = doJSON(t, h, "POST", "/v1/sims/"+imsi+"/resume", tok, nil)
	_ = json.Unmarshal(w.Body.Bytes(), &sim)
	if sim.Status != "active" {
		t.Fatalf("status after resume: %s", sim.Status)
	}

	// single-SIM lookup
	w = doJSON(t, h, "GET", "/v1/sims/"+imsi, tok, nil)
	if w.Code != 200 {
		t.Fatalf("get sim: %d", w.Code)
	}
}

func TestTxDisarmAndAudit(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	// lab mode: arm is always denied, but the attempt must be recorded
	w := doJSON(t, h, "POST", "/v1/tx/arm", tok, api.TxArmRequest{
		Country: "US", Band: "n48", Acknowledgment: "I hold authorization for this transmission",
	})
	if w.Code != 200 {
		t.Fatalf("arm: %d", w.Code)
	}
	var resp api.TxArmResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Armed {
		t.Fatal("lab must refuse")
	}

	w = doJSON(t, h, "POST", "/v1/tx/disarm", tok, nil)
	if w.Code != 200 {
		t.Fatalf("disarm: %d", w.Code)
	}

	w = doJSON(t, h, "GET", "/v1/audit", tok, nil)
	if w.Code != 200 {
		t.Fatalf("audit: %d", w.Code)
	}
	var entries []api.AuditEntry
	_ = json.Unmarshal(w.Body.Bytes(), &entries)
	found := false
	for _, e := range entries {
		if e.Action == "tx_arm" {
			found = true
		}
	}
	if !found {
		t.Fatalf("audit log missing tx_arm entry: %+v", entries)
	}
}

func TestEsimIssueListRevoke(t *testing.T) {
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
	srv := NewWithOptions(cfg, st, id, fakeChecker{}, nil, Options{
		ESIM: &ESIMOptions{
			Enabled: true, RegistryPath: filepath.Join(dir, "esim", "registry.json"),
			SMDPAddress: "fairwave.local:8443", SMDPID: "fairwave-esim",
		},
	})
	tok := "test-token"
	h := srv.Handler()

	// issue a lab SIM (IMSI base == first lab vector)
	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 1})
	if w.Code != 201 {
		t.Fatalf("issue: %d %s", w.Code, w.Body.String())
	}
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)
	imsi := sims[0].IMSI

	w = doJSON(t, h, "POST", "/v1/esim/issue", tok, api.EsimIssueRequest{IMSI: imsi})
	if w.Code != 201 {
		t.Fatalf("esim issue: %d %s", w.Code, w.Body.String())
	}
	var resp api.EsimIssueResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ActivationCode == "" || resp.ICCID == "" || resp.SMDPAddress == "" {
		t.Fatalf("esim issue response incomplete: %+v", resp)
	}
	if resp.QRPNGBase64 == "" {
		t.Fatal("expected QR payload")
	}
	if _, err := base64.StdEncoding.DecodeString(resp.QRPNGBase64); err != nil {
		t.Fatalf("qr not valid base64: %v", err)
	}

	// es9+ surface is mounted without auth; use a structurally valid EID so
	// the endpoint answers a real protocol error instead of a validation one.
	eid, err := profile.NewEID()
	if err != nil {
		t.Fatal(err)
	}
	w = doJSON(t, h, "POST", "/es9plus/initiateAuthentication", "", map[string]string{
		"activation_code": "LPA:1$" + resp.SMDPAddress + "$" + resp.ActivationCode,
		"eid":             eid,
	})
	if w.Code == 401 {
		t.Fatal("es9plus must not require the admin token")
	}
	if w.Code == 503 {
		t.Fatal("es9plus must be served when esim is enabled")
	}

	w = doJSON(t, h, "GET", "/v1/esim/codes", tok, nil)
	if w.Code != 200 {
		t.Fatalf("esim codes: %d", w.Code)
	}
	var codes []api.EsimCode
	_ = json.Unmarshal(w.Body.Bytes(), &codes)
	if len(codes) != 1 || codes[0].IMSI != imsi {
		t.Fatalf("codes: %+v", codes)
	}

	w = doJSON(t, h, "POST", "/v1/esim/revoke", tok, api.EsimRevokeRequest{ActivationCode: resp.ActivationCode})
	if w.Code != 200 {
		t.Fatalf("esim revoke: %d %s", w.Code, w.Body.String())
	}
}

// staticSource is a collector.Source returning a fixed snapshot.
type staticSource struct{ sessions []api.Session }

func (s staticSource) Poll(context.Context) ([]api.Session, error) { return s.sessions, nil }

func TestCollectorMarksSimActive(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 1})
	if w.Code != 201 {
		t.Fatalf("issue: %d", w.Code)
	}
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)
	imsi := sims[0].IMSI
	if sims[0].Status != "issued" {
		t.Fatalf("fresh sim status: %s", sims[0].Status)
	}

	srv.collector = staticSource{sessions: []api.Session{{
		IMSIHash: api.HashIMSI(imsi), APN: "internet", Phase: "active",
	}}}
	if err := srv.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	got, ok := srv.store.GetSIM(imsi)
	if !ok || got.Status != "active" {
		t.Fatalf("SIM should be active after attach: %+v", got)
	}
	if n := len(srv.store.ListSessions()); n != 1 {
		t.Fatalf("sessions = %d, want 1", n)
	}
}

func TestEsimDisabledReturns503(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()
	w := doJSON(t, h, "POST", "/v1/esim/issue", tok, api.EsimIssueRequest{IMSI: "999991234567001"})
	if w.Code != 503 {
		t.Fatalf("esim disabled must 503, got %d", w.Code)
	}
}

func newEsimServer(t *testing.T, singleUse bool) (*Server, string) {
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
	srv := NewWithOptions(cfg, st, id, fakeChecker{}, nil, Options{
		ESIM: &ESIMOptions{
			Enabled: true, RegistryPath: filepath.Join(dir, "esim", "registry.json"),
			SMDPAddress: "fairwave.local:8443", SMDPID: "fairwave-esim", SingleUse: singleUse,
		},
	})
	return srv, "test-token"
}

// TestEsimSingleUseCode runs the full download twice: the first succeeds,
// the second must be refused because the code is single-use.
func TestEsimSingleUseCode(t *testing.T) {
	srv, tok := newEsimServer(t, true)
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 1})
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)

	w = doJSON(t, h, "POST", "/v1/esim/issue", tok, api.EsimIssueRequest{IMSI: sims[0].IMSI})
	if w.Code != 201 {
		t.Fatalf("esim issue: %d %s", w.Code, w.Body.String())
	}
	var resp api.EsimIssueResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	code := "LPA:1$" + resp.SMDPAddress + "$" + resp.ActivationCode

	// Run the loop against the server's own handler (which mounts /es9plus).
	hs := httptest.NewServer(h)
	defer hs.Close()
	eu, err := euicc.New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eu.Download(context.Background(), hs.URL, code, hs.Client()); err != nil {
		t.Fatalf("first download: %v", err)
	}
	// A second eUICC must not be able to use the same code.
	eu2, err := euicc.New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eu2.Download(context.Background(), hs.URL, code, hs.Client()); err == nil {
		t.Fatal("second download of a single-use code must fail")
	}
}

func TestTokensRBAC(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	// admin mints an operator and a viewer token
	w := doJSON(t, h, "POST", "/v1/tokens", tok, api.TokenCreateRequest{Name: "ops-1", Role: api.RoleOperator})
	if w.Code != 201 {
		t.Fatalf("create operator token: %d %s", w.Code, w.Body.String())
	}
	var opTok api.TokenCreateResponse
	_ = json.Unmarshal(w.Body.Bytes(), &opTok)
	if opTok.Token == "" {
		t.Fatal("secret must be returned once")
	}

	w = doJSON(t, h, "POST", "/v1/tokens", tok, api.TokenCreateRequest{Name: "view-1", Role: api.RoleViewer})
	var viewTok api.TokenCreateResponse
	_ = json.Unmarshal(w.Body.Bytes(), &viewTok)

	// viewer: read ok, write forbidden
	if w := doJSON(t, h, "GET", "/v1/status", viewTok.Token, nil); w.Code != 200 {
		t.Fatalf("viewer read: %d", w.Code)
	}
	if w := doJSON(t, h, "POST", "/v1/sims", viewTok.Token, api.SimIssueRequest{Profile: "lab", Count: 1}); w.Code != 403 {
		t.Fatalf("viewer write must 403, got %d", w.Code)
	}

	// listing tokens must not invalidate them (the hash is stripped for the
	// response; the stored record must keep it)
	if w := doJSON(t, h, "GET", "/v1/tokens", tok, nil); w.Code != 200 {
		t.Fatalf("token list: %d", w.Code)
	}
	if w := doJSON(t, h, "GET", "/v1/status", opTok.Token, nil); w.Code != 200 {
		t.Fatalf("token must survive a list call, got %d", w.Code)
	}

	// operator: mutating sims ok, admin surfaces forbidden
	if w := doJSON(t, h, "POST", "/v1/sims", opTok.Token, api.SimIssueRequest{Profile: "lab", Count: 1}); w.Code != 201 {
		t.Fatalf("operator write: %d %s", w.Code, w.Body.String())
	}
	if w := doJSON(t, h, "POST", "/v1/tx/disarm", opTok.Token, nil); w.Code != 403 {
		t.Fatalf("operator tx disarm must 403, got %d", w.Code)
	}
	if w := doJSON(t, h, "GET", "/v1/audit", opTok.Token, nil); w.Code != 403 {
		t.Fatalf("operator audit must 403, got %d", w.Code)
	}
	if w := doJSON(t, h, "GET", "/v1/tokens", opTok.Token, nil); w.Code != 403 {
		t.Fatalf("operator tokens must 403, got %d", w.Code)
	}

	// audit attribution: the operator's action is attributed to ops-1
	w = doJSON(t, h, "GET", "/v1/audit", tok, nil)
	var entries []api.AuditEntry
	_ = json.Unmarshal(w.Body.Bytes(), &entries)
	found := false
	for _, e := range entries {
		if e.Action == "sim_issue" && e.Principal == "ops-1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("audit missing ops-1 attribution: %+v", entries)
	}

	// revoke the operator token: it stops working
	w = doJSON(t, h, "DELETE", "/v1/tokens/"+opTok.ID, tok, nil)
	if w.Code != 204 {
		t.Fatalf("revoke token: %d", w.Code)
	}
	if w := doJSON(t, h, "GET", "/v1/status", opTok.Token, nil); w.Code != 401 {
		t.Fatalf("revoked token must 401, got %d", w.Code)
	}
}

func TestSimImport(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims/import", tok, api.SimImportRequest{Sims: []api.SimImportItem{
		{IMSI: "999991234567010", MSISDN: "9999010", Profile: "prod", APN: "internet"},
		{IMSI: "999991234567011", MSISDN: "9999011", Profile: "lab", APN: "ims"},
		{IMSI: "123", APN: "internet"}, // invalid, skipped
	}})
	if w.Code != 200 {
		t.Fatalf("import: %d %s", w.Code, w.Body.String())
	}
	var resp api.SimImportResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Imported != 2 || len(resp.Skipped) != 1 {
		t.Fatalf("import result: %+v", resp)
	}
	if sim, ok := srv.store.GetSIM("999991234567010"); !ok || sim.Profile != "prod" {
		t.Fatalf("imported sim: %+v", sim)
	}
	// re-import updates
	w = doJSON(t, h, "POST", "/v1/sims/import", tok, api.SimImportRequest{Sims: []api.SimImportItem{
		{IMSI: "999991234567010", Status: "suspended"},
	}})
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Updated != 1 {
		t.Fatalf("re-import should update: %+v", resp)
	}
}

func TestQuotaAndUsage(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 1})
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)
	imsi := sims[0].IMSI

	w = doJSON(t, h, "POST", "/v1/sims/"+imsi+"/quota", tok, api.SimQuotaRequest{QuotaBytes: 1000})
	if w.Code != 200 {
		t.Fatalf("quota: %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, h, "POST", "/v1/sims/"+imsi+"/usage", tok, api.SimUsageRequest{BytesUp: 400, BytesDn: 200})
	if w.Code != 200 {
		t.Fatalf("reconcile: %d %s", w.Code, w.Body.String())
	}

	w = doJSON(t, h, "GET", "/v1/sims/"+imsi+"/usage", tok, nil)
	var u api.SimUsage
	_ = json.Unmarshal(w.Body.Bytes(), &u)
	if u.BytesUp != 400 || u.BytesDn != 200 || u.QuotaBytes != 1000 {
		t.Fatalf("usage: %+v", u)
	}
}

// TestFairUseAutoSuspend: with fair-use enabled, a SIM whose usage meets its
// quota is suspended automatically.
func TestFairUseAutoSuspend(t *testing.T) {
	srv, tok := newTestServer(t)
	srv.cfg.FairUse.Enabled = true
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 1})
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)
	imsi := sims[0].IMSI
	_ = doJSON(t, h, "POST", "/v1/sims/"+imsi+"/quota", tok, api.SimQuotaRequest{QuotaBytes: 1000})

	srv.collector = staticSource{sessions: []api.Session{{
		IMSIHash: api.HashIMSI(imsi), APN: "internet", Phase: "active", BytesUp: 500,
	}}}
	if err := srv.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	// first accumulation establishes the baseline
	if err := srv.AccumulateUsage(context.Background()); err != nil {
		t.Fatal(err)
	}
	// traffic grows past the quota -> delta trips the cap
	srv.collector = staticSource{sessions: []api.Session{{
		IMSIHash: api.HashIMSI(imsi), APN: "internet", Phase: "active", BytesUp: 1500,
	}}}
	if err := srv.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := srv.AccumulateUsage(context.Background()); err != nil {
		t.Fatal(err)
	}

	sim, ok := srv.store.GetSIM(imsi)
	if !ok {
		t.Fatal("sim missing")
	}
	if sim.Status != "suspended" {
		t.Fatalf("SIM should be suspended over quota, got %s", sim.Status)
	}
}

func TestAlertsFireAndResolve(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	// stale heartbeat -> node-down alert
	old := time.Now().Add(-10 * time.Minute)
	temp := 30.0
	_ = doJSON(t, h, "POST", "/v1/telemetry", tok, api.NodeHealth{NodeID: "n1", TS: old, SDRTempC: &temp, Watchdog: "ok", FreqCheck: true})
	srv.EvaluateAlerts(context.Background())

	w := doJSON(t, h, "GET", "/v1/alerts", tok, nil)
	if w.Code != 200 {
		t.Fatalf("alerts: %d", w.Code)
	}
	var alerts []api.Alert
	_ = json.Unmarshal(w.Body.Bytes(), &alerts)
	if len(alerts) != 1 || alerts[0].Key != "node-down:n1" || alerts[0].Resolved {
		t.Fatalf("alerts: %+v", alerts)
	}

	// fresh heartbeat -> the same alert resolves, nothing new fires
	fresh := time.Now()
	_ = doJSON(t, h, "POST", "/v1/telemetry", tok, api.NodeHealth{NodeID: "n1", TS: fresh, SDRTempC: &temp, Watchdog: "ok", FreqCheck: true})
	srv.EvaluateAlerts(context.Background())

	w = doJSON(t, h, "GET", "/v1/alerts", tok, nil)
	_ = json.Unmarshal(w.Body.Bytes(), &alerts)
	if len(alerts) != 1 || !alerts[0].Resolved {
		t.Fatalf("alert should resolve: %+v", alerts)
	}
}

func TestComplianceExport(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()
	_ = doJSON(t, h, "POST", "/v1/tx/arm", tok, api.TxArmRequest{Country: "US", Band: "n48"})

	w := doJSON(t, h, "GET", "/v1/compliance/export", tok, nil)
	if w.Code != 200 {
		t.Fatalf("compliance: %d %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "fairwave compliance report") || !strings.Contains(body, "tx_arm") {
		t.Fatalf("compliance body incomplete: %.200s", body)
	}
}

func TestBackupRestoreRoundtrip(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()
	_ = doJSON(t, h, "POST", "/v1/nodes", tok, api.Node{Name: "cafe1", Country: "LAB"})

	w := doJSON(t, h, "GET", "/v1/backup", tok, nil)
	if w.Code != 200 {
		t.Fatalf("backup: %d %s", w.Code, w.Body.String())
	}
	archive := w.Body.Bytes()
	if len(archive) == 0 {
		t.Fatal("empty backup")
	}

	// restore into a fresh server's data dir
	dir2 := t.TempDir()
	cfg := config.Default()
	cfg.Server.DataDir = dir2
	id2, _ := identity.LoadOrCreate(filepath.Join(dir2, "identity"))
	st2, _ := store.Open(filepath.Join(dir2, "state"))
	srv2 := New(cfg, st2, id2, fakeChecker{}, nil)
	req := httptest.NewRequest("POST", "/v1/restore", bytes.NewReader(archive))
	req.Header.Set("Authorization", "Bearer test-token")
	w2 := httptest.NewRecorder()
	srv2.Handler().ServeHTTP(w2, req)
	if w2.Code != 200 {
		t.Fatalf("restore: %d %s", w2.Code, w2.Body.String())
	}

	// the restored state is on disk (in-memory state needs a restart)
	st3, err := store.Open(filepath.Join(dir2, "state"))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(st3.ListNodes()); n != 1 {
		t.Fatalf("restored nodes = %d, want 1", n)
	}
}

// ---- core-metered usage (GTP-U accounting) ----

// gtpFeed delivers synthetic GTP packets to a collector.UPF source.
type gtpFeed struct{ ch chan []byte }

func (f *gtpFeed) Next() ([]byte, error) {
	pkt, ok := <-f.ch
	if !ok {
		return nil, io.EOF
	}
	return pkt, nil
}

func (f *gtpFeed) Close() error {
	close(f.ch)
	return nil
}

// minimal GTP frame builders (mirror collector's test helpers) - enough to
// drive the accounting path end to end.
func gtpIMSI(imsi string) []byte {
	out := make([]byte, 0, (len(imsi)+1)/2)
	for i := 0; i < len(imsi); i += 2 {
		hi := imsi[i] - '0'
		lo := byte(0xF)
		if i+1 < len(imsi) {
			lo = imsi[i+1] - '0'
		}
		out = append(out, hi<<4|lo)
	}
	return out
}

func gtpIE(t byte, val []byte) []byte {
	out := make([]byte, 4+len(val))
	out[0] = t
	out[2], out[3] = byte(len(val)>>8), byte(len(val))
	copy(out[4:], val)
	return out
}

func gtpFTEID(ifaceType byte, teid uint32) []byte {
	v := make([]byte, 9)
	v[0] = ifaceType | 0x10
	binary.BigEndian.PutUint32(v[1:5], teid)
	return v
}

func gtpWrap(gtp []byte) []byte {
	total := 20 + 8 + len(gtp)
	out := make([]byte, total)
	out[0] = 0x45
	binary.BigEndian.PutUint16(out[2:4], uint16(total))
	out[9] = 17
	binary.BigEndian.PutUint16(out[24:26], uint16(8+len(gtp)))
	copy(out[28:], gtp)
	return out
}

// gtpCreateSession builds an S5-C Create Session Request learning the SGW
// S5-U TEID (interface type 5 = uplink).
func gtpCreateSession(imsi string, teid uint32) []byte {
	bearer := gtpIE(93, gtpIE(87, gtpFTEID(5, teid)))
	msg := make([]byte, 8+len(bearer)+4+8)
	msg[0] = 0x20
	msg[1] = 32
	ies := append(gtpIE(1, gtpIMSI(imsi)), bearer...)
	binary.BigEndian.PutUint16(msg[2:4], uint16(len(ies)))
	copy(msg[8:], ies)
	return gtpWrap(msg)
}

// gtpGPDU builds a GTPv1-U user-data packet carrying n payload bytes.
func gtpGPDU(teid uint32, n int) []byte {
	gtp := make([]byte, 8+n)
	gtp[0] = 0x30
	gtp[1] = 255
	binary.BigEndian.PutUint16(gtp[2:4], uint16(n))
	binary.BigEndian.PutUint32(gtp[4:8], teid)
	return gtpWrap(gtp)
}

func waitCoreBytes(t *testing.T, srv *Server, imsi string, wantBytes uint64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if err := srv.CollectOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		for _, s := range srv.store.ListSessions() {
			if s.IMSIHash == api.HashIMSI(imsi) && s.BytesUp+s.BytesDn >= wantBytes {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for core-metered bytes")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestCoreMeteredUsageAutoSuspend drives the full path: GTP frames from a
// feed -> per-UE byte counters -> usage fold -> fair-use auto-suspend.
func TestCoreMeteredUsageAutoSuspend(t *testing.T) {
	srv, tok := newTestServer(t)
	srv.cfg.FairUse.Enabled = true
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 1})
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)
	imsi := sims[0].IMSI
	if w := doJSON(t, h, "POST", "/v1/sims/"+imsi+"/quota", tok, api.SimQuotaRequest{QuotaBytes: 1000}); w.Code != 200 {
		t.Fatalf("quota: %d %s", w.Code, w.Body.String())
	}

	feed := &gtpFeed{ch: make(chan []byte, 64)}
	upf := collector.NewUPF(collector.UPFConfig{PacketSource: feed, Now: time.Now})
	srv.collector = upf
	defer upf.Close()

	// UE attaches and pushes 500 bytes of uplink traffic.
	feed.ch <- gtpCreateSession(imsi, 0x1111)
	feed.ch <- gtpGPDU(0x1111, 500)
	waitCoreBytes(t, srv, imsi, 500)
	if err := srv.AccumulateUsage(context.Background()); err != nil {
		t.Fatal(err)
	}
	u, _ := srv.store.GetUsage(imsi)
	if u == nil || u.BytesUp != 500 {
		t.Fatalf("usage after fold: %+v", u)
	}

	// Traffic crosses the 1000-byte quota -> auto-suspend, audited.
	feed.ch <- gtpGPDU(0x1111, 600)
	waitCoreBytes(t, srv, imsi, 1100)
	if err := srv.AccumulateUsage(context.Background()); err != nil {
		t.Fatal(err)
	}
	sim, ok := srv.store.GetSIM(imsi)
	if !ok || sim.Status != "suspended" {
		t.Fatalf("core-metered usage must auto-suspend over quota, got %+v", sim)
	}
	found := false
	for _, e := range srv.store.ListAudit() {
		if e.Action == "sim_suspend" && e.Detail == "data quota reached" {
			found = true
		}
	}
	if !found {
		t.Fatal("quota suspension must be audited")
	}
}

// ---- core-metered usage (free5GC CHF CDRs) ----

// cdrTLV is a minimal BER TLV builder for the free5GC CHF CDR fixture
// (mirrors the encoder in free5gc/chf cdr/asn; see the collector tests for
// the full layout). Only the fields the collector reads are emitted.
func cdrTLV(id byte, content []byte) []byte {
	out := []byte{id}
	if len(content) < 128 {
		out = append(out, byte(len(content)))
	} else {
		n := 1
		for l := len(content); l > 255; l >>= 8 {
			n++
		}
		out = append(out, byte(n)|0x80)
		for i := n - 1; i >= 0; i-- {
			out = append(out, byte(len(content)>>(8*i)))
		}
	}
	return append(out, content...)
}

func cdrInt(n int64) []byte {
	var out []byte
	for n > 127 {
		out = append([]byte{byte(n & 0xff)}, out...)
		n >>= 8
	}
	return append([]byte{byte(n)}, out...)
}

// cdrUsageRecord builds one ChargingRecord (context [1] SET) with the
// subscriber identifier and a single rating-group usage carrying ul/dn
// bytes - exactly what the collector needs to report per-UE totals.
func cdrUsageRecord(imsi string, ul, dn int64) []byte {
	sub := cdrTLV(0xA2, append(cdrTLV(0x80, cdrInt(1)), cdrTLV(0x81, []byte(imsi))...))
	used := cdrTLV(0x30, append(append(cdrTLV(0x84, cdrInt(ul+dn)),
		cdrTLV(0x85, cdrInt(ul))...), cdrTLV(0x86, cdrInt(dn))...))
	muu := cdrTLV(0x30, append(cdrTLV(0x80, cdrInt(1)), cdrTLV(0xA1, used)...))
	return cdrTLV(0xA1, append(append(cdrTLV(0x80, cdrInt(200)),
		sub...), cdrTLV(0xA5, muu)...))
}

// cdrFile wraps one or more records in a TS 32.297 container (52-byte
// header + 4-byte record headers), as free5GC's CHF writes them.
func cdrFile(records ...[]byte) []byte {
	header := make([]byte, 52)
	binary.BigEndian.PutUint32(header[4:8], 52)
	binary.BigEndian.PutUint32(header[18:22], uint32(len(records)))
	var body []byte
	for _, rec := range records {
		hdr := make([]byte, 4)
		binary.BigEndian.PutUint16(hdr[0:2], uint16(len(rec)))
		hdr[3] = 1 << 5
		body = append(body, hdr...)
		body = append(body, rec...)
	}
	binary.BigEndian.PutUint32(header[0:4], uint32(52+len(body)))
	return append(header, body...)
}

// TestCDRMeteredUsageAutoSuspend drives the full free5GC path: CHF CDR
// files on a shared volume -> CDR collector -> usage fold -> fair-use
// auto-suspend. This is the Phase 3 replacement for the GTP-U tap.
func TestCDRMeteredUsageAutoSuspend(t *testing.T) {
	cdrDir := t.TempDir()
	imsi := "999991234567001"

	srv, tok := newTestServer(t)
	srv.cfg.FairUse.Enabled = true
	srv.collector = collector.NewCDR(collector.CDRConfig{Dir: cdrDir, Now: time.Now})
	h := srv.Handler()

	w := doJSON(t, h, "POST", "/v1/sims", tok, api.SimIssueRequest{Profile: "lab", Count: 1})
	var sims []api.SIM
	_ = json.Unmarshal(w.Body.Bytes(), &sims)
	if w := doJSON(t, h, "POST", "/v1/sims/"+imsi+"/quota", tok, api.SimQuotaRequest{QuotaBytes: 1000}); w.Code != 200 {
		t.Fatalf("quota: %d %s", w.Code, w.Body.String())
	}

	// The CHF writes the first usage snapshot: 500 up / 300 dn.
	if err := os.WriteFile(filepath.Join(cdrDir, "chf-test.cdr"), cdrFile(cdrUsageRecord(imsi, 500, 300)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := srv.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := srv.AccumulateUsage(context.Background()); err != nil {
		t.Fatal(err)
	}
	u, _ := srv.store.GetUsage(imsi)
	if u == nil || u.BytesUp != 500 || u.BytesDn != 300 {
		t.Fatalf("usage after first fold: %+v", u)
	}

	// The CHF rewrites the snapshot past the 1000-byte quota (800 up /
	// 300 dn = 1100 total) -> delta trips the cap -> auto-suspend, audited.
	if err := os.WriteFile(filepath.Join(cdrDir, "chf-test.cdr"), cdrFile(cdrUsageRecord(imsi, 800, 300)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := srv.CollectOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := srv.AccumulateUsage(context.Background()); err != nil {
		t.Fatal(err)
	}
	sim, ok := srv.store.GetSIM(imsi)
	if !ok || sim.Status != "suspended" {
		t.Fatalf("CDR-metered usage must auto-suspend over quota, got %+v", sim)
	}
	found := false
	for _, e := range srv.store.ListAudit() {
		if e.Action == "sim_suspend" && e.Detail == "data quota reached" {
			found = true
		}
	}
	if !found {
		t.Fatal("quota suspension must be audited")
	}
}

func TestBackupRestoreWithPassphrase(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	req := httptest.NewRequest("GET", "/v1/backup", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("X-Fairwave-Passphrase", "hunter2")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("encrypted backup: %d %s", w.Code, w.Body.String())
	}
	archive := w.Body.Bytes()

	// wrong passphrase must fail
	dir := t.TempDir()
	id2, _ := identity.LoadOrCreate(filepath.Join(dir, "identity"))
	st2, _ := store.Open(filepath.Join(dir, "state"))
	srv2 := New(config.Default(), st2, id2, fakeChecker{}, nil)
	r := httptest.NewRequest("POST", "/v1/restore", bytes.NewReader(archive))
	r.Header.Set("Authorization", "Bearer test-token")
	r.Header.Set("X-Fairwave-Passphrase", "wrong")
	wr := httptest.NewRecorder()
	srv2.Handler().ServeHTTP(wr, r)
	if wr.Code == 200 {
		t.Fatal("wrong passphrase must fail restore")
	}

	// correct passphrase succeeds
	r2 := httptest.NewRequest("POST", "/v1/restore", bytes.NewReader(archive))
	r2.Header.Set("Authorization", "Bearer test-token")
	r2.Header.Set("X-Fairwave-Passphrase", "hunter2")
	wr2 := httptest.NewRecorder()
	srv2.Handler().ServeHTTP(wr2, r2)
	if wr2.Code != 200 {
		t.Fatalf("restore with passphrase: %d %s", wr2.Code, wr2.Body.String())
	}
}
