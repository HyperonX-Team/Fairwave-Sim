package node

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReportOncePostsEveryThread(t *testing.T) {
	var paths, auths []string
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		auths = append(auths, r.Header.Get("Authorization"))
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		bodies = append(bodies, m)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n, err := New(Config{
		ID: "n", WeaveID: "w", Anchor: "a",
		Threads: []ThreadConfig{
			{ID: "t1", Local: "127.0.0.1:0", Mbps: 100},
			{ID: "t2", Local: "127.0.0.1:0", Mbps: 200},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = n.Close() }()

	// Trailing slash must be tolerated.
	if err := ReportOnce(context.Background(), srv.Client(), n, srv.URL+"/", "secret"); err != nil {
		t.Fatalf("report: %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("reported %d threads, want 2", len(bodies))
	}
	for i, p := range paths {
		if !strings.HasPrefix(p, "/v1/hydra/threads/") || !strings.HasSuffix(p, "/health") {
			t.Fatalf("unexpected path %q", p)
		}
		if auths[i] != "Bearer secret" {
			t.Fatalf("auth header = %q", auths[i])
		}
		if bodies[i]["up"] != true {
			t.Fatalf("thread health not up: %+v", bodies[i])
		}
	}
}

func TestReportOnceSurfacesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n, err := New(Config{ID: "n", Threads: []ThreadConfig{{ID: "t1", Local: "127.0.0.1:0", Mbps: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = n.Close() }()

	if err := ReportOnce(context.Background(), srv.Client(), n, srv.URL, ""); err == nil {
		t.Fatal("expected an error for http 500")
	}
}

func TestReportLoopStopsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n, err := New(Config{ID: "n", Threads: []ThreadConfig{{ID: "t1", Local: "127.0.0.1:0", Mbps: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = n.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		ReportLoop(ctx, n, srv.URL, "", 20*time.Millisecond, nil)
		close(done)
	}()
	time.Sleep(60 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReportLoop did not stop on cancel")
	}
}
