// free5GC live-session source.
//
// free5GC's AMF exposes an OAM HTTP service (namf-oam) alongside its SBI
// server (default :8000). `GET /namf-oam/v1/registered-ue-context`
// returns every registered UE with its SUPI and PDU sessions - the
// 5G-core equivalent of Open5GS's /ue-info infoAPI (see
// https://git.unl.edu/pqc-free5gc/amf for the route + response shape).
//
// The response items carry Go-default JSON field names (no tags):
//
//	[{"AccessType":"3GPP_ACCESS","Supi":"imsi-...","Guti":"...",
//	  "Mcc":"999","Mnc":"99","Tac":"1",
//	  "PduSessions":[{"PduSessionId":"1","SmContextRef":"...","Sst":"1","Sd":"010203","Dnn":"internet"}],
//	  "CmState":"CONNECTED"}]
//
// The parser is tolerant of casing so it survives free5GC minor version
// drift. Failures are never fatal: a transient poll error leaves the
// previous snapshot in place.
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// Free5GCConfig configures the free5GC AMF OAM driver.
type Free5GCConfig struct {
	// AMFOAMURL is the base URL of the AMF OAM server, e.g.
	// http://amf:8000 (the SBI/OAM port).
	AMFOAMURL string
	// Client is the HTTP client; a default with a sane timeout is used
	// when nil.
	Client *http.Client
	// Now is the clock used to timestamp sessions (injectable for tests).
	Now func() time.Time
}

// Free5GC polls the AMF OAM API for registered UEs.
type Free5GC struct {
	cfg Free5GCConfig
}

// NewFree5GC builds the free5GC driver.
func NewFree5GC(cfg Free5GCConfig) *Free5GC {
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Free5GC{cfg: cfg}
}

// Poll implements Source: it fetches the registered UE contexts and maps
// each to a session (SUPI -> hashed IMSI, DNN -> APN).
func (f *Free5GC) Poll(ctx context.Context) ([]api.Session, error) {
	if f.cfg.AMFOAMURL == "" {
		return nil, fmt.Errorf("collector: free5gc amf_oam_url required")
	}
	u := strings.TrimRight(f.cfg.AMFOAMURL, "/") + "/namf-oam/v1/registered-ue-context"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.cfg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("collector: free5gc get %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("collector: free5gc get %s: %s", u, resp.Status)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("collector: free5gc read %s: %w", u, err)
	}
	return f.parse(raw)
}

// parse turns the OAM response into sessions, accepting both the
// capitalized Go-default keys and lowercase variants.
func (f *Free5GC) parse(raw []byte) ([]api.Session, error) {
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("collector: free5gc parse registered-ue-context: %w", err)
	}
	now := f.cfg.Now().UTC()
	out := make([]api.Session, 0, len(items))
	for _, item := range items {
		supi := jsonString(item, "Supi", "supi")
		if supi == "" {
			continue
		}
		imsi := strings.TrimPrefix(supi, "imsi-")
		if imsi == "" {
			continue
		}
		phase := jsonString(item, "CmState", "cmState")
		if phase == "" {
			phase = "connected"
		}
		phase = strings.ToLower(phase)
		out = append(out, api.Session{
			IMSIHash: api.HashIMSI(imsi),
			APN:      firstDNN(item),
			Phase:    phase,
			Created:  now,
		})
	}
	return out, nil
}

// firstDNN extracts the DNN of the UE's first PDU session (tolerating
// both key casings).
func firstDNN(item map[string]json.RawMessage) string {
	for _, key := range []string{"PduSessions", "pduSessions"} {
		var sessions []map[string]json.RawMessage
		if err := json.Unmarshal(item[key], &sessions); err == nil && len(sessions) > 0 {
			if dnn := jsonString(sessions[0], "Dnn", "dnn"); dnn != "" {
				return dnn
			}
		}
	}
	return ""
}

// jsonString returns the first non-empty string field under any of the
// candidate keys.
func jsonString(item map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		var s string
		if err := json.Unmarshal(item[k], &s); err == nil && s != "" {
			return s
		}
	}
	return ""
}
