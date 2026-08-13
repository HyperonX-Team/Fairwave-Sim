package agent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendPostsTelemetry(t *testing.T) {
	var (
		gotPath string
		gotAuth string
		gotBody []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := New(Config{ControlURL: srv.URL, Token: "secret-token"})
	h := Health{NodeID: "node-1", Load1: 0.5, Watchdog: "ok", Radio: "zmq"}
	if err := a.send(h); err != nil {
		t.Fatalf("send: %v", err)
	}

	if gotPath != "/v1/telemetry" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("auth = %q", gotAuth)
	}
	var decoded Health
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if decoded.NodeID != "node-1" {
		t.Fatalf("body = %+v", decoded)
	}
}

func TestSendReportsServerErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	a := New(Config{ControlURL: srv.URL})
	if err := a.send(Health{NodeID: "n"}); err == nil {
		t.Fatal("401 must surface as an error")
	}
}

func TestSendNoopWithoutControlURL(t *testing.T) {
	a := New(Config{})
	if err := a.send(Health{NodeID: "n"}); err != nil {
		t.Fatalf("no control url must be a no-op: %v", err)
	}
}
