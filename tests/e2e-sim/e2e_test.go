// Package e2esim contains the lab end-to-end integration test harness.
// It requires the compose lab to be up (`make lab-up`); it asserts the
// control-plane view of the attach. Run with: make test-e2e-lab
package e2esim

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

const baseURL = "http://localhost:8080"

// labClient returns an authed client, or skips the test when the lab is
// not reachable.
func labClient(t *testing.T, authed bool) *http.Client {
	t.Helper()
	c := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest("GET", baseURL+"/v1/healthz", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Skipf("control plane not reachable (%v); run make lab-up first", err)
	}
	resp.Body.Close()
	if authed {
		if os.Getenv("FW_ADMIN_TOKEN") == "" {
			t.Skip("FW_ADMIN_TOKEN not set (make lab-up prints the control-plane token)")
		}
	}
	return c
}

func doGet(t *testing.T, c *http.Client, path string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest("GET", baseURL+path, nil)
	if tok := os.Getenv("FW_ADMIN_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Skipf("control plane not reachable (%v); run make lab-up first", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestControlPlaneHealth(t *testing.T) {
	c := labClient(t, false)
	code, body := doGet(t, c, "/v1/healthz")
	if code != 200 {
		t.Skipf("healthz: %d %s (is the Fairwave control plane up? run make lab-up)", code, body)
	}
	var h map[string]bool
	if err := json.Unmarshal([]byte(body), &h); err != nil || !h["ok"] {
		t.Fatalf("unexpected health payload: %s", body)
	}
}

func TestStatusReflectsLab(t *testing.T) {
	c := labClient(t, true)
	code, body := doGet(t, c, "/v1/status")
	if code != 200 {
		t.Fatalf("status: %d %s", code, body)
	}
	var st api.Status
	if err := json.Unmarshal([]byte(body), &st); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if st.Mode != "lab" {
		t.Fatalf("mode: %q (want lab) — is the control plane running in lab mode?", st.Mode)
	}
	if st.TxArmed {
		t.Fatal("TX must never be armed in lab mode")
	}
}

func TestLabSIMList(t *testing.T) {
	c := labClient(t, true)
	code, body := doGet(t, c, "/v1/sims")
	if code != 200 {
		t.Fatalf("sims: %d %s", code, body)
	}
	var sims []api.SIM
	if err := json.Unmarshal([]byte(body), &sims); err != nil {
		t.Fatalf("decode sims: %v", err)
	}
	t.Logf("SIMs in store: %d", len(sims))
}
