package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// mmeInfo returns the documented MME /ue-info shape for one connected UE.
func mmeInfo(t *testing.T) []byte {
	t.Helper()
	doc := map[string]any{
		"items": []any{
			map[string]any{
				"supi":     "999991234567001",
				"cm_state": "connected",
				"pdn": []any{
					map[string]any{"apn": "internet", "qci": 9, "pdu_state": "active"},
				},
			},
		},
		"pager": map[string]any{"page": 0, "page_size": 100, "count": 1},
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNoneSource(t *testing.T) {
	sessions, err := (None{}).Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Fatalf("none source must yield no sessions, got %d", len(sessions))
	}
}

func TestOpen5GSParsesMMEInfo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ue-info", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "-1" {
			t.Errorf("expected page=-1, got %q", r.URL.Query().Get("page"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mmeInfo(t))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	o := NewOpen5GS(Open5GSConfig{MMEURL: srv.URL})
	sessions, err := o.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %d, want 1: %+v", len(sessions), sessions)
	}
	s := sessions[0]
	wantHash := api.HashIMSI("999991234567001")
	if s.IMSIHash != wantHash {
		t.Fatalf("hash = %s, want %s", s.IMSIHash, wantHash)
	}
	if s.APN != "internet" || s.Phase != "active" {
		t.Fatalf("session = %+v", s)
	}
}

func TestOpen5GSEnrichesIPFromSMF(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ue-info", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(mmeInfo(t))
	})
	mux.HandleFunc("/pdu-info", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"supi":"999991234567001","pdu":[{"dnn":"internet","ipv4":"10.45.0.11","pdu_state":"active"}]}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	o := NewOpen5GS(Open5GSConfig{MMEURL: srv.URL, SMFURL: srv.URL})
	sessions, err := o.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].IP != "10.45.0.11" {
		t.Fatalf("expected IP enrichment, got %+v", sessions)
	}
}

func TestOpen5GSPollError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	o := NewOpen5GS(Open5GSConfig{MMEURL: srv.URL})
	if _, err := o.Poll(context.Background()); err == nil {
		t.Fatal("poll must surface non-200 responses")
	}
}
