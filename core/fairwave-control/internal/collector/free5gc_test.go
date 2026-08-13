package collector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// oamResponse is the documented AMF OAM /registered-ue-context shape:
// Go-default capitalized keys (no json tags) plus the SUPI prefix.
const oamResponse = `[
  {
    "AccessType": "3GPP_ACCESS",
    "Supi": "imsi-999991234567001",
    "Guti": "999-99-0001-01",
    "Mcc": "999",
    "Mnc": "99",
    "Tac": "1",
    "PduSessions": [
      {
        "PduSessionId": "1",
        "SmContextRef": "smContextRef-1",
        "Sst": "1",
        "Sd": "010203",
        "Dnn": "internet"
      }
    ],
    "CmState": "CONNECTED"
  },
  {
    "AccessType": "3GPP_ACCESS",
    "Supi": "imsi-999991234567002",
    "Guti": "999-99-0002-01",
    "CmState": "IDLE"
  },
  {
    "Supi": "imsi-999991234567003",
    "CmState": "CONNECTED"
  }
]`

func TestFree5GCParsesOAM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/namf-oam/v1/registered-ue-context" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(oamResponse))
	}))
	defer srv.Close()

	f := NewFree5GC(Free5GCConfig{AMFOAMURL: srv.URL})
	sessions, err := f.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 3 {
		t.Fatalf("sessions = %d, want 3: %+v", len(sessions), sessions)
	}
	byHash := map[string]api.Session{}
	for _, s := range sessions {
		byHash[s.IMSIHash] = s
	}
	first := byHash[api.HashIMSI("999991234567001")]
	if first.APN != "internet" || first.Phase != "connected" {
		t.Fatalf("first session = %+v", first)
	}
	second := byHash[api.HashIMSI("999991234567002")]
	if second.Phase != "idle" {
		t.Fatalf("second session phase = %q, want idle", second.Phase)
	}
	third := byHash[api.HashIMSI("999991234567003")]
	if third.APN != "" {
		t.Fatalf("third session apn = %q, want empty (no PDU session)", third.APN)
	}
}

func TestFree5GCToleratesLowercaseKeys(t *testing.T) {
	raw := `[{"supi":"imsi-999991234567004","cmState":"CONNECTED","pduSessions":[{"dnn":"ims"}]}]`
	f := NewFree5GC(Free5GCConfig{})
	sessions, err := f.parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(sessions))
	}
	if sessions[0].APN != "ims" {
		t.Fatalf("apn = %q, want ims", sessions[0].APN)
	}
}

func TestFree5GCPollError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := NewFree5GC(Free5GCConfig{AMFOAMURL: srv.URL})
	if _, err := f.Poll(context.Background()); err == nil {
		t.Fatal("poll must surface non-200 responses")
	}
}

func TestFree5GCRequiresURL(t *testing.T) {
	f := NewFree5GC(Free5GCConfig{})
	if _, err := f.Poll(context.Background()); err == nil {
		t.Fatal("missing amf_oam_url must fail")
	}
}
