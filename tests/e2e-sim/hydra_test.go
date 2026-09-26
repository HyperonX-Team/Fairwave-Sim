package e2esim

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/fabric"
)

// doJSONReq performs an authenticated JSON request against the live lab
// control plane, skipping when it is unreachable.
func doJSONReq(t *testing.T, method, path string, in, out any) int {
	t.Helper()
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok := os.Getenv("FW_ADMIN_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("control plane not reachable (%v); run make lab-up first", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if out != nil && len(data) > 0 {
		_ = json.Unmarshal(data, out)
	}
	return resp.StatusCode
}

// TestHydraWeaveBenchAgainstLab drives the full Hydra surface (threads ->
// weave -> bench -> cleanup) against a live lab control plane and asserts
// the aggregate ceiling is the sum of the threads.
func TestHydraWeaveBenchAgainstLab(t *testing.T) {
	labClient(t, true) // skips when the lab or token is absent

	suffix := os.Getenv("FW_HYDRA_SUFFIX")
	if suffix == "" {
		suffix = "e2e"
	}
	weaveID := "e2e-" + suffix
	threadIDs := []string{"e2e-a-" + suffix, "e2e-b-" + suffix, "e2e-c-" + suffix}

	// Clean up any leftovers from a previous run.
	doJSONReq(t, http.MethodDelete, "/v1/hydra/weaves/"+weaveID, nil, nil)
	for _, id := range threadIDs {
		doJSONReq(t, http.MethodDelete, "/v1/hydra/threads/"+id, nil, nil)
	}

	for _, id := range threadIDs {
		var th map[string]any
		if code := doJSONReq(t, http.MethodPost, "/v1/hydra/threads",
			api.HydraThreadRequest{ID: id, Mbps: 300, RTTms: 20, LossPct: 0.1, Up: true}, &th); code != 201 {
			t.Fatalf("add thread %s: %d", id, code)
		}
	}

	var wv fabric.Weave
	if code := doJSONReq(t, http.MethodPost, "/v1/hydra/weaves",
		api.HydraWeaveRequest{ID: weaveID, Threads: threadIDs}, &wv); code != 201 {
		t.Fatalf("create weave: %d", code)
	}

	var res fabric.BenchResult
	if code := doJSONReq(t, http.MethodPost, "/v1/hydra/weaves/"+weaveID+"/bench",
		api.HydraBenchRequest{Packets: 300, PktBytes: 1200}, &res); code != 200 {
		t.Fatalf("bench: %d", code)
	}
	if res.Delivered != 300 {
		t.Fatalf("delivered = %d want 300", res.Delivered)
	}
	if res.AggregateMbps < 900 {
		t.Fatalf("aggregate = %.0f Mbps want >= 900", res.AggregateMbps)
	}
	if res.Speedup < 2.9 {
		t.Fatalf("speedup = %.2fx want >= 2.9", res.Speedup)
	}
	t.Logf("hydra e2e: %d packets | single=%.0f Mbps aggregate=%.0f Mbps speedup=%.2fx",
		res.Packets, res.SingleThreadMbps, res.AggregateMbps, res.Speedup)

	// Cleanup so repeated runs stay idempotent.
	if code := doJSONReq(t, http.MethodDelete, "/v1/hydra/weaves/"+weaveID, nil, nil); code != 204 {
		t.Fatalf("delete weave: %d", code)
	}
	for _, id := range threadIDs {
		if code := doJSONReq(t, http.MethodDelete, "/v1/hydra/threads/"+id, nil, nil); code != 204 {
			t.Fatalf("delete thread %s: %d", id, code)
		}
	}
}
