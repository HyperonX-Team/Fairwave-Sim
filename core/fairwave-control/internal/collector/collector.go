// Package collector turns live network state from Open5GS into
// privacy-preserving session records for the control plane.
//
// The driver talks to the Open5GS "infoAPI" - JSON endpoints embedded in
// the MME/SMF HTTP servers (the same port Prometheus metrics use,
// default :9090; see https://open5gs.org/open5gs/docs/tutorial/07-infoAPI-UE-gNB-session-data/).
// MME exposes /ue-info (connected LTE UEs + PDNs), SMF exposes /pdu-info
// (5G PDU sessions with IPs). Failures are reported to the caller and are
// never fatal: a transient poll error leaves the previous snapshot in
// place.
//
// The default source (None) is a safe no-op - no collector, no sessions -
// matching the lab default where nothing is configured.
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// Source produces the live session snapshot. Implementations must be safe
// for concurrent use by the polling loop only.
type Source interface {
	Poll(ctx context.Context) ([]api.Session, error)
}

// None is the safe default: no collector configured.
type None struct{}

// Poll implements Source with no sessions.
func (None) Poll(context.Context) ([]api.Session, error) { return nil, nil }

// Open5GSConfig configures the Open5GS infoAPI driver.
type Open5GSConfig struct {
	// MMEURL is the base URL of the MME infoAPI server, e.g.
	// http://127.0.0.2:9090. Required for 4G (EPS) session collection.
	MMEURL string
	// SMFURL is the optional base URL of the SMF infoAPI server; when set,
	// PDU sessions are merged in to enrich IP addresses.
	SMFURL string
	// Client is the HTTP client; a default with a sane timeout is used when
	// nil.
	Client *http.Client
	// Now is the clock used to timestamp sessions (injectable for tests).
	Now func() time.Time
}

// Open5GS polls the Open5GS infoAPI for connected UEs and their sessions.
type Open5GS struct {
	cfg Open5GSConfig
}

// NewOpen5GS builds the Open5GS driver.
func NewOpen5GS(cfg Open5GSConfig) *Open5GS {
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Open5GS{cfg: cfg}
}

// Poll implements Source: it fetches /ue-info from the MME and, when
// configured, /pdu-info from the SMF, merging IPs and deduplicating.
func (o *Open5GS) Poll(ctx context.Context) ([]api.Session, error) {
	if o.cfg.MMEURL == "" {
		return nil, fmt.Errorf("collector: mme_url required")
	}
	mmeSessions, err := o.pollMME(ctx)
	if err != nil {
		return nil, err
	}
	ips := map[string]string{}
	if o.cfg.SMFURL != "" {
		ips, err = o.pollSMFIPs(ctx)
		if err != nil {
			return nil, err
		}
	}
	now := o.cfg.Now().UTC()
	seen := map[string]bool{}
	var out []api.Session
	for _, s := range mmeSessions {
		if s.IP == "" {
			s.IP = ips[s.IMSIHash]
		}
		key := s.IMSIHash + "|" + s.APN + "|" + s.IP
		if seen[key] {
			continue
		}
		seen[key] = true
		s.Created = now
		out = append(out, s)
	}
	return out, nil
}

func (o *Open5GS) pollMME(ctx context.Context) ([]api.Session, error) {
	raw, err := o.get(ctx, o.cfg.MMEURL, "/ue-info")
	if err != nil {
		return nil, err
	}
	var info mmeUEInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("collector: parse mme /ue-info: %w", err)
	}
	now := o.cfg.Now().UTC()
	var out []api.Session
	for _, ue := range info.Items {
		imsi := strings.TrimPrefix(ue.SUPI, "imsi-")
		if imsi == "" {
			continue
		}
		hash := api.HashIMSI(imsi)
		if len(ue.PDN) == 0 {
			out = append(out, api.Session{
				IMSIHash: hash,
				APN:      "",
				Phase:    or(ue.CMState, "connected"),
				Created:  now,
			})
			continue
		}
		for _, p := range ue.PDN {
			out = append(out, api.Session{
				IMSIHash: hash,
				APN:      p.APN,
				IP:       p.IPV4,
				Phase:    or(p.State, or(ue.CMState, "connected")),
				Created:  now,
			})
		}
	}
	return out, nil
}

func (o *Open5GS) pollSMFIPs(ctx context.Context) (map[string]string, error) {
	raw, err := o.get(ctx, o.cfg.SMFURL, "/pdu-info")
	if err != nil {
		return nil, err
	}
	var info smfPDUInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("collector: parse smf /pdu-info: %w", err)
	}
	ips := map[string]string{}
	for _, ue := range info.Items {
		imsi := strings.TrimPrefix(ue.SUPI, "imsi-")
		if imsi == "" {
			continue
		}
		hash := api.HashIMSI(imsi)
		for _, p := range ue.PDU {
			if p.IPV4 != "" {
				ips[hash] = p.IPV4
				break
			}
		}
	}
	return ips, nil
}

func (o *Open5GS) get(ctx context.Context, base, path string) ([]byte, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("collector: bad base url %q: %w", base, err)
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	q := u.Query()
	q.Set("page", "-1") // disable paging: we want the full snapshot
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.cfg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("collector: get %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("collector: get %s: %s", u, resp.Status)
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("collector: read %s: %w", u, err)
	}
	return buf, nil
}

// mmeUEInfo is the tolerant subset of the MME /ue-info shape.
type mmeUEInfo struct {
	Items []struct {
		SUPI    string `json:"supi"`
		CMState string `json:"cm_state"`
		PDN     []struct {
			APN   string `json:"apn"`
			IPV4  string `json:"ipv4"`
			State string `json:"pdu_state"`
		} `json:"pdn"`
	} `json:"items"`
}

// smfPDUInfo is the tolerant subset of the SMF /pdu-info shape.
type smfPDUInfo struct {
	Items []struct {
		SUPI string `json:"supi"`
		PDU  []struct {
			DNN  string `json:"dnn"`
			IPV4 string `json:"ipv4"`
		} `json:"pdu"`
	} `json:"items"`
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
