package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/bearer"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/fabric"
)

func TestHydraThreadAndWeaveLifecycle(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	// Empty status initially.
	w := doJSON(t, h, "GET", "/v1/hydra/status", tok, nil)
	if w.Code != 200 {
		t.Fatalf("status: %d %s", w.Code, w.Body.String())
	}
	var st fabric.Status
	_ = json.Unmarshal(w.Body.Bytes(), &st)
	if st.Threads != 0 || st.Weaves != 0 {
		t.Fatalf("initial status = %+v", st)
	}

	// Add three threads.
	for _, req := range []api.HydraThreadRequest{
		{ID: "sim-a", Box: "box-a", SIM: "999991234567001", Mbps: 300, RTTms: 18, Up: true},
		{ID: "sim-b", Box: "box-b", SIM: "999991234567002", Mbps: 300, RTTms: 22, Up: true},
		{ID: "sim-c", Box: "box-c", SIM: "999991234567003", Mbps: 300, RTTms: 30, Up: true},
	} {
		w := doJSON(t, h, "POST", "/v1/hydra/threads", tok, req)
		if w.Code != 201 {
			t.Fatalf("add thread %s: %d %s", req.ID, w.Code, w.Body.String())
		}
	}

	// Invalid thread is rejected.
	w = doJSON(t, h, "POST", "/v1/hydra/threads", tok, api.HydraThreadRequest{ID: "bad", Mbps: -1})
	if w.Code != 400 {
		t.Fatalf("negative mbps accepted: %d", w.Code)
	}

	// List + aggregate.
	w = doJSON(t, h, "GET", "/v1/hydra/threads", tok, nil)
	var threads []bearer.Thread
	_ = json.Unmarshal(w.Body.Bytes(), &threads)
	if len(threads) != 3 {
		t.Fatalf("threads = %d", len(threads))
	}

	// Create a weave over all three threads.
	w = doJSON(t, h, "POST", "/v1/hydra/weaves", tok, api.HydraWeaveRequest{
		ID: "gig", Anchor: "hub", Threads: []string{"sim-a", "sim-b", "sim-c"},
	})
	if w.Code != 201 {
		t.Fatalf("create weave: %d %s", w.Code, w.Body.String())
	}

	// Duplicate weave conflicts.
	w = doJSON(t, h, "POST", "/v1/hydra/weaves", tok, api.HydraWeaveRequest{ID: "gig", Threads: []string{"sim-a"}})
	if w.Code != 409 {
		t.Fatalf("duplicate weave: %d", w.Code)
	}

	// Status now shows the weave and 900 Mbps aggregate.
	w = doJSON(t, h, "GET", "/v1/hydra/status", tok, nil)
	_ = json.Unmarshal(w.Body.Bytes(), &st)
	if st.Weaves != 1 || st.AggregateMbps != 900 {
		t.Fatalf("status after weave = %+v", st)
	}

	// /v1/status carries the hydra summary too.
	w = doJSON(t, h, "GET", "/v1/status", tok, nil)
	var status api.Status
	_ = json.Unmarshal(w.Body.Bytes(), &status)
	if status.HydraWeaves != 1 || status.HydraAggMbps != 900 {
		t.Fatalf("status hydra fields = %+v", status)
	}

	// Delete the weave and a thread.
	if w := doJSON(t, h, "DELETE", "/v1/hydra/weaves/gig", tok, nil); w.Code != 204 {
		t.Fatalf("delete weave: %d", w.Code)
	}
	if w := doJSON(t, h, "DELETE", "/v1/hydra/threads/sim-a", tok, nil); w.Code != 204 {
		t.Fatalf("delete thread: %d", w.Code)
	}
	if w := doJSON(t, h, "DELETE", "/v1/hydra/weaves/gig", tok, nil); w.Code != 404 {
		t.Fatalf("delete missing weave: %d", w.Code)
	}
}

func TestHydraStripIngestRoundTrip(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()
	doJSON(t, h, "POST", "/v1/hydra/threads", tok, api.HydraThreadRequest{ID: "t1", Mbps: 500, Up: true})
	doJSON(t, h, "POST", "/v1/hydra/weaves", tok, api.HydraWeaveRequest{ID: "w", Threads: []string{"t1"}})

	payload := []byte("hydra-payload")
	w := doJSON(t, h, "POST", "/v1/hydra/weaves/w/strip", tok, api.HydraStripRequest{
		PayloadB64: base64.StdEncoding.EncodeToString(payload),
	})
	if w.Code != 200 {
		t.Fatalf("strip: %d %s", w.Code, w.Body.String())
	}
	var strip api.HydraStripResponse
	_ = json.Unmarshal(w.Body.Bytes(), &strip)
	if strip.ThreadID != "t1" || strip.Seq != 1 || strip.FrameB64 == "" {
		t.Fatalf("strip = %+v", strip)
	}

	// Ingest the frame back and confirm one payload is delivered.
	w = doJSON(t, h, "POST", "/v1/hydra/weaves/w/ingest", tok, api.HydraIngestRequest{FrameB64: strip.FrameB64})
	if w.Code != 200 {
		t.Fatalf("ingest: %d %s", w.Code, w.Body.String())
	}
	var ing api.HydraIngestResponse
	_ = json.Unmarshal(w.Body.Bytes(), &ing)
	if ing.Delivered != 1 {
		t.Fatalf("delivered = %d", ing.Delivered)
	}

	// Bad base64 is a client error, not a panic.
	if w := doJSON(t, h, "POST", "/v1/hydra/weaves/w/strip", tok, api.HydraStripRequest{PayloadB64: "!!!"}); w.Code != 400 {
		t.Fatalf("bad base64 strip: %d", w.Code)
	}
	if w := doJSON(t, h, "POST", "/v1/hydra/weaves/w/ingest", tok, api.HydraIngestRequest{FrameB64: "!!!"}); w.Code != 400 {
		t.Fatalf("bad base64 ingest: %d", w.Code)
	}
}

func TestHydraBenchEndpoint(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()
	for _, id := range []string{"a", "b", "c", "d"} {
		doJSON(t, h, "POST", "/v1/hydra/threads", tok, api.HydraThreadRequest{ID: id, Mbps: 300, Up: true})
	}
	doJSON(t, h, "POST", "/v1/hydra/weaves", tok, api.HydraWeaveRequest{ID: "w", Threads: []string{"a", "b", "c", "d"}})

	w := doJSON(t, h, "POST", "/v1/hydra/weaves/w/bench", tok, api.HydraBenchRequest{})
	if w.Code != 200 {
		t.Fatalf("bench: %d %s", w.Code, w.Body.String())
	}
	var res fabric.BenchResult
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Delivered != 300 {
		t.Fatalf("delivered = %d want 300", res.Delivered)
	}
	if res.AggregateMbps != 1200 {
		t.Fatalf("aggregate = %v want 1200", res.AggregateMbps)
	}
	if res.Speedup < 3.9 {
		t.Fatalf("speedup = %v want ~4", res.Speedup)
	}

	// Bench on an unknown weave is a conflict, not a 500.
	if w := doJSON(t, h, "POST", "/v1/hydra/weaves/ghost/bench", tok, api.HydraBenchRequest{}); w.Code != 409 {
		t.Fatalf("bench missing weave: %d", w.Code)
	}
}

func TestHydraPersistsAcrossRestart(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()
	doJSON(t, h, "POST", "/v1/hydra/threads", tok, api.HydraThreadRequest{ID: "p1", Mbps: 300, Up: true})
	doJSON(t, h, "POST", "/v1/hydra/threads", tok, api.HydraThreadRequest{ID: "p2", Mbps: 200, Up: true})
	doJSON(t, h, "POST", "/v1/hydra/weaves", tok, api.HydraWeaveRequest{ID: "pw", Threads: []string{"p1", "p2"}})

	// A fresh server over the same store must restore both.
	srv2 := New(srv.cfg, srv.store, srv.id, fakeChecker{}, nil)
	h2 := srv2.Handler()
	w := doJSON(t, h2, "GET", "/v1/hydra/threads", tok, nil)
	var threads []bearer.Thread
	_ = json.Unmarshal(w.Body.Bytes(), &threads)
	if len(threads) != 2 {
		t.Fatalf("restored threads = %d want 2", len(threads))
	}
	w = doJSON(t, h2, "GET", "/v1/hydra/weaves", tok, nil)
	var weaves []fabric.Weave
	_ = json.Unmarshal(w.Body.Bytes(), &weaves)
	if len(weaves) != 1 || weaves[0].ID != "pw" {
		t.Fatalf("restored weaves = %+v", weaves)
	}
	// A restored weave must be immediately usable.
	if w := doJSON(t, h2, "POST", "/v1/hydra/weaves/pw/bench", tok, api.HydraBenchRequest{Packets: 20, PktBytes: 100}); w.Code != 200 {
		t.Fatalf("bench after restore: %d %s", w.Code, w.Body.String())
	}

	// Deletion must also survive a restart.
	doJSON(t, h, "DELETE", "/v1/hydra/weaves/pw", tok, nil)
	srv3 := New(srv.cfg, srv.store, srv.id, fakeChecker{}, nil)
	if _, ok := srv3.hydra.GetWeave("pw"); ok {
		t.Fatal("deleted weave came back after restart")
	}
}

func TestHydraThreadHealth(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()
	doJSON(t, h, "POST", "/v1/hydra/threads", tok, api.HydraThreadRequest{ID: "hh", Mbps: 300, Up: true})

	w := doJSON(t, h, "POST", "/v1/hydra/threads/hh/health", tok,
		api.HydraThreadHealth{Mbps: 0, RTTms: 12.5, LossPct: 1.2, Up: true})
	if w.Code != 200 {
		t.Fatalf("health: %d %s", w.Code, w.Body.String())
	}
	th, ok := srv.hydra.Thread("hh")
	if !ok || th.RTTms != 12.5 || th.LossPct != 1.2 || th.Mbps != 300 {
		t.Fatalf("thread = %+v (mbps must be preserved when 0)", th)
	}

	// Live health must persist across a restart.
	srv2 := New(srv.cfg, srv.store, srv.id, fakeChecker{}, nil)
	th2, _ := srv2.hydra.Thread("hh")
	if th2.RTTms != 12.5 {
		t.Fatalf("restored rtt = %v", th2.RTTms)
	}

	// Unknown thread is a 404, not a 500.
	if w := doJSON(t, h, "POST", "/v1/hydra/threads/ghost/health", tok, api.HydraThreadHealth{Up: true}); w.Code != 404 {
		t.Fatalf("ghost health: %d", w.Code)
	}
}

func TestUIServedUnauthenticated(t *testing.T) {
	srv, tok := newTestServer(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><title>fairwave</title>"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv.cfg.Server.UIDir = dir
	h := srv.Handler()

	// The dashboard is served without a token...
	w := doJSON(t, h, "GET", "/", "", nil)
	if w.Code != 200 {
		t.Fatalf("ui: %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("ui content-type = %q", ct)
	}
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("ui is missing its Content-Security-Policy")
	}

	// ...but the API still is not.
	if w := doJSON(t, h, "GET", "/v1/status", "", nil); w.Code != 401 {
		t.Fatalf("api without token: %d want 401", w.Code)
	}
	if w := doJSON(t, h, "GET", "/v1/status", tok, nil); w.Code != 200 {
		t.Fatalf("api with token: %d", w.Code)
	}

	// Writing to the dashboard is refused.
	if w := doJSON(t, h, "POST", "/", "", nil); w.Code != 405 {
		t.Fatalf("ui POST: %d want 405", w.Code)
	}
}

func TestHydraRBAC(t *testing.T) {
	srv, tok := newTestServer(t)
	h := srv.Handler()

	// Mint a viewer token.
	w := doJSON(t, h, "POST", "/v1/tokens", tok, api.TokenCreateRequest{Name: "watcher", Role: api.RoleViewer})
	if w.Code != 201 {
		t.Fatalf("mint viewer: %d %s", w.Code, w.Body.String())
	}
	var created api.TokenCreateResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Viewer may read the hydra surface...
	if w := doJSON(t, h, "GET", "/v1/hydra/status", created.Token, nil); w.Code != 200 {
		t.Fatalf("viewer read: %d", w.Code)
	}
	// ...but not mutate it.
	if w := doJSON(t, h, "POST", "/v1/hydra/threads", created.Token, api.HydraThreadRequest{ID: "x", Mbps: 1}); w.Code != http.StatusForbidden {
		t.Fatalf("viewer mutate: %d want 403", w.Code)
	}
}
