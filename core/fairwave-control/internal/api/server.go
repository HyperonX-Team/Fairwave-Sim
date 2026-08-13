// Package api implements the northbound REST API for fairwave-control.
// Auth: Bearer token from FAIRWAVE_ADMIN_TOKEN (or auto-generated on first
// run and written to data/admin_token, 0600). In lab mode with no token
// configured, only loopback is accepted.
package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/collector"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/config"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/identity"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/store"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/hsswrite"
	"github.com/prometheus/client_golang/prometheus"
)

// Version is set at build time via -ldflags.
var Version = "0.1.0"

// SpectrumChecker is the policy gate interface the API depends on, kept
// narrow so the API layer is testable with a fake checker.
type SpectrumChecker interface {
	Check(country, band string, indoor bool, licenseRef string) Verdict
}

// Verdict mirrors core/policy.Verdict.
type Verdict struct {
	Allowed bool
	Reasons []string
}

// Server hosts the control-plane REST API.
type Server struct {
	cfg       *config.ControlConfig
	store     *store.Store
	id        *identity.Identity
	adminTok  string
	txArmed   bool
	started   time.Time
	spectrum  SpectrumChecker
	hss       hsswrite.Writer
	collector collector.Source
	esim      *esimSvc
	now       func() time.Time
	mux       *http.ServeMux

	mu sync.Mutex // guards fired alerts + the usage delta snapshot
	// fired remembers which alert keys have already fired, so recovery and
	// re-fire are edge-triggered rather than level-triggered spam.
	fired map[string]bool
	// lastUsage remembers the previous per-UE byte snapshot so usage can be
	// accumulated from deltas reported by the session source.
	lastUsage map[string]usagePoint
}

// usagePoint is one UE's byte counters at the last accumulation.
type usagePoint struct {
	up uint64
	dn uint64
}

// Options configures optional server components (collector, eSIM server).
type Options struct {
	// Collector produces live session snapshots (defaults to None).
	Collector collector.Source
	// ESIM enables the SM-DP+ server + activation-code registry. Nil
	// disables the eSIM surface (routes answer 503).
	ESIM *ESIMOptions
}

// ESIMOptions configures the embedded lab SM-DP+ server.
type ESIMOptions struct {
	Enabled      bool
	RegistryPath string        // defaults to <data_dir>/esim/registry.json
	SMDPAddress  string        // host[:port] embedded in activation codes
	SMDPID       string        //
	SingleUse    bool          // activation codes die after one download
	CodeTTL      time.Duration // activation codes expire after this long (0 = no code-level expiry)
}

// New creates the API server with default options. sc may be nil (all
// spectrum checks denied); hss may be nil (SIM ops are store-only).
func New(cfg *config.ControlConfig, st *store.Store, id *identity.Identity, sc SpectrumChecker, hss hsswrite.Writer) *Server {
	return NewWithOptions(cfg, st, id, sc, hss, Options{})
}

// NewWithOptions creates the API server with optional collector and eSIM
// components wired in.
func NewWithOptions(cfg *config.ControlConfig, st *store.Store, id *identity.Identity, sc SpectrumChecker, hss hsswrite.Writer, opts Options) *Server {
	if hss == nil {
		hss = hsswrite.None{}
	}
	if opts.Collector == nil {
		opts.Collector = collector.None{}
	}
	s := &Server{
		cfg:       cfg,
		store:     st,
		id:        id,
		spectrum:  sc,
		hss:       hss,
		collector: opts.Collector,
		txArmed:   st.GetTxArmed(),
		started:   time.Now(),
		now:       time.Now,
		fired:     map[string]bool{},
		lastUsage: map[string]usagePoint{},
	}
	if opts.ESIM != nil {
		s.esim = newEsimSvc(cfg, opts.ESIM)
	}
	s.adminTok = strings.TrimSpace(os.Getenv(cfg.Auth.AdminTokenEnv))
	if s.adminTok == "" {
		tokPath := filepath.Join(cfg.Server.DataDir, "admin_token")
		if data, err := os.ReadFile(tokPath); err == nil {
			s.adminTok = strings.TrimSpace(string(data))
		} else {
			buf := make([]byte, 24)
			if _, err := rand.Read(buf); err == nil {
				s.adminTok = hex.EncodeToString(buf)
				_ = os.MkdirAll(cfg.Server.DataDir, 0o750)
				_ = os.WriteFile(tokPath, []byte(s.adminTok), 0o600)
				log.Printf("generated admin token -> %s (0600, store it safely)", tokPath)
			}
		}
	}
	s.registerMetrics()
	s.mux = http.NewServeMux()
	s.routes()
	return s
}

// Token returns the admin token (for CLI bootstrap / docs).
func (s *Server) Token() string { return s.adminTok }

// Handler returns the HTTP handler for the whole API surface.
func (s *Server) Handler() http.Handler {
	return s.recoverMW(s.authMW(s.logMW(s.mux)))
}

// RunBackground drives the periodic jobs: session collection, SIM expiry
// sweeping, usage accumulation, and alert evaluation. It blocks until ctx
// is done; the caller runs it in a goroutine.
func (s *Server) RunBackground(ctx context.Context) {
	collInterval := s.cfg.Collector.Interval
	if collInterval <= 0 {
		collInterval = 15 * time.Second
	}
	usageInterval := s.cfg.FairUse.UsageInterval
	if usageInterval <= 0 {
		usageInterval = time.Minute
	}

	sweep := time.NewTicker(time.Minute)
	defer sweep.Stop()
	usage := time.NewTicker(usageInterval)
	defer usage.Stop()
	alerts := time.NewTicker(30 * time.Second)
	defer alerts.Stop()

	var coll <-chan time.Time
	if _, ok := s.collector.(collector.None); !ok {
		t := time.NewTicker(collInterval)
		defer t.Stop()
		coll = t.C
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-sweep.C:
			s.ExpireOnce(ctx)
		case <-usage.C:
			if err := s.AccumulateUsage(ctx); err != nil {
				log.Printf("usage: %v", err)
			}
		case <-alerts.C:
			if s.cfg.Alerts.Enabled {
				s.EvaluateAlerts(ctx)
			}
		case <-coll:
			if err := s.CollectOnce(ctx); err != nil {
				log.Printf("collector: %v", err)
			}
		}
	}
}

// CollectOnce polls the session source and replaces the session table,
// marking issued SIMs active when their UE attaches.
func (s *Server) CollectOnce(ctx context.Context) error {
	sessions, err := s.collector.Poll(ctx)
	if err != nil {
		return err
	}
	if err := s.store.ReplaceSessions(sessions); err != nil {
		return err
	}
	s.refreshMetrics()
	// A UE attaching is the first evidence its SIM works: promote issued
	// SIMs to active when their (hashed) IMSI shows up in a session.
	for _, sess := range sessions {
		for _, sim := range s.store.ListSIMs() {
			if sim.Status == "issued" && api.HashIMSI(sim.IMSI) == sess.IMSIHash {
				sim.Status = "active"
				if err := s.store.UpsertSIM(sim); err != nil {
					log.Printf("mark sim %s active: %v", sim.IMSI, err)
				}
			}
		}
	}
	return nil
}

// ExpireOnce revokes SIMs whose expiry has passed. Called periodically by
// RunBackground; exported for tests.
func (s *Server) ExpireOnce(ctx context.Context) {
	now := s.now().UTC()
	for _, sim := range s.store.ListSIMs() {
		if sim.Status == "revoked" || sim.Status == "expired" {
			continue
		}
		if sim.ExpiresAt.IsZero() || now.Before(sim.ExpiresAt) {
			continue
		}
		sim.Status = "expired"
		if err := s.store.UpsertSIM(sim); err != nil {
			log.Printf("expire sim %s: %v", sim.IMSI, err)
			continue
		}
		if err := s.hss.Remove(ctx, sim.IMSI); err != nil {
			log.Printf("expire sim %s: hss remove: %v", sim.IMSI, err)
		}
		s.audit("system", "sim_expired", sim.IMSI, "auto-revoked at expiry")
	}
	s.refreshMetrics()
}

// audit appends one entry to the append-only audit log under the given
// principal (token name, "admin", or "system" for background jobs).
func (s *Server) audit(principal, action, target, detail string) {
	id := make([]byte, 4)
	if _, err := rand.Read(id); err != nil {
		id = []byte("audit")
	}
	e := api.AuditEntry{
		ID:        hex.EncodeToString(id),
		TS:        s.now().UTC(),
		Action:    action,
		Target:    target,
		Detail:    detail,
		Principal: principal,
	}
	if err := s.store.AppendAudit(e); err != nil {
		log.Printf("audit %s %s: %v", action, target, err)
	}
}

// auditReq is audit() for request handlers: it attributes the action to
// the authenticated principal from the request context.
func (s *Server) auditReq(r *http.Request, action, target, detail string) {
	s.audit(principalFrom(r), action, target, detail)
}

// ---- tokens ----

// createToken mints a scoped API token. The raw secret is returned once;
// only its SHA-256 hash is stored.
func (s *Server) createToken(name string, role api.TokenRole) (*api.TokenCreateResponse, error) {
	if !role.Valid() {
		return nil, fmt.Errorf("role must be admin, operator or viewer")
	}
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	secret := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(secret))
	t := &api.Token{
		ID:        fmt.Sprintf("tok-%s", hex.EncodeToString([]byte(secret[:3]))),
		Name:      name,
		Role:      role,
		TokenHash: hex.EncodeToString(sum[:]),
		CreatedAt: s.now().UTC(),
	}
	if err := s.store.UpsertToken(t); err != nil {
		return nil, err
	}
	return &api.TokenCreateResponse{
		ID: t.ID, Name: t.Name, Role: t.Role, Token: secret, CreatedAt: t.CreatedAt,
	}, nil
}

func (s *Server) revokeToken(id string) error {
	return s.store.RevokeToken(id)
}

// ---- usage accumulation ----

// AccumulateUsage folds per-UE byte deltas (reported by the session
// source) into per-SIM usage totals and, when fair-use is enabled,
// suspends SIMs that exceed their quota. Deltas are computed against the
// previous snapshot, so a source that does not report bytes (the Open5GS
// infoAPI) simply contributes zero until a richer metering source is
// wired in.
func (s *Server) AccumulateUsage(ctx context.Context) error {
	if len(s.store.ListSessions()) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	sessions := s.store.ListSessions()
	cur := map[string]usagePoint{}
	for _, sess := range sessions {
		p := cur[sess.IMSIHash]
		p.up += sess.BytesUp
		p.dn += sess.BytesDn
		cur[sess.IMSIHash] = p
	}
	for hash, c := range cur {
		// Sessions carry monotonic cumulative counters, so usage is the
		// delta vs the previous reading. A never-seen hash starts at zero:
		// the first reading is real usage, not a baseline.
		prev := s.lastUsage[hash]
		dUp, dDn := uint64(0), uint64(0)
		if c.up > prev.up {
			dUp = c.up - prev.up
		}
		if c.dn > prev.dn {
			dDn = c.dn - prev.dn
		}
		s.lastUsage[hash] = c
		if dUp == 0 && dDn == 0 {
			continue
		}
		imsi := ""
		for _, sim := range s.store.ListSIMs() {
			if api.HashIMSI(sim.IMSI) == hash {
				imsi = sim.IMSI
				break
			}
		}
		if imsi == "" {
			continue
		}
		u, _ := s.store.GetUsage(imsi)
		if u == nil {
			u = &api.SimUsage{IMSI: imsi, IMSIHash: hash}
		}
		u.BytesUp += dUp
		u.BytesDn += dDn
		u.QuotaBytes = quotaFor(s.store, imsi)
		u.UpdatedAt = now
		if err := s.store.UpsertUsage(u); err != nil {
			return err
		}
		if s.cfg.FairUse.Enabled {
			if err := s.enforceQuota(ctx, imsi, u); err != nil {
				log.Printf("quota %s: %v", imsi, err)
			}
		}
	}
	return nil
}

// enforceQuota suspends a SIM whose usage meets its quota (when the SIM is
// currently usable) and raises an alert so the operator can react.
func (s *Server) enforceQuota(ctx context.Context, imsi string, u *api.SimUsage) error {
	if u.QuotaBytes == 0 {
		return nil
	}
	total := u.BytesUp + u.BytesDn
	sim, ok := s.store.GetSIM(imsi)
	if !ok || sim.Status != "active" && sim.Status != "issued" {
		return nil
	}
	if total < u.QuotaBytes {
		return nil
	}
	sim.Status = "suspended"
	if err := s.store.UpsertSIM(sim); err != nil {
		return err
	}
	s.audit("system", "sim_suspend", imsi, "data quota reached")
	s.fireAlertLocked(api.AlertWarning, "quota:"+imsi, "SIM reached its data quota and was suspended", imsi, "")
	return nil
}

// ---- alert engine ----

// EvaluateAlerts checks every threshold against current state and fires
// (or resolves) alerts edge-triggered, delivering webhooks on transitions.
func (s *Server) EvaluateAlerts(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stale := s.cfg.Telemetry.StaleAfter
	if stale <= 0 {
		stale = 90 * time.Second
	}
	now := s.now().UTC()
	seen := map[string]bool{}

	// node health thresholds
	for _, h := range s.store.ListHealth() {
		up := now.Sub(h.TS) <= stale
		if !up {
			key := "node-down:" + h.NodeID
			seen[key] = true
			s.fireAlertLocked(api.AlertCritical, key, "node stopped reporting heartbeats", h.NodeID, h.NodeID)
		}
		if h.SDRTempC != nil && *h.SDRTempC > s.cfg.Alerts.TempHighC {
			key := "sdr-temp:" + h.NodeID
			seen[key] = true
			s.fireAlertLocked(api.AlertWarning, key, fmt.Sprintf("SDR temperature %.1fC above threshold %.1fC", *h.SDRTempC, s.cfg.Alerts.TempHighC), h.NodeID, h.NodeID)
		}
		if h.Watchdog != "" && h.Watchdog != "ok" {
			key := "watchdog:" + h.NodeID
			seen[key] = true
			s.fireAlertLocked(api.AlertWarning, key, "watchdog reports: "+h.Watchdog, h.NodeID, h.NodeID)
		}
		if !h.FreqCheck {
			key := "freq:" + h.NodeID
			seen[key] = true
			s.fireAlertLocked(api.AlertCritical, key, "frequency plan check failed", h.NodeID, h.NodeID)
		}
	}

	// SIM expiry warnings
	warn := time.Duration(s.cfg.Alerts.SimExpiryWarnDays) * 24 * time.Hour
	if warn > 0 {
		for _, sim := range s.store.ListSIMs() {
			if sim.Status == "revoked" || sim.Status == "expired" || sim.ExpiresAt.IsZero() {
				continue
			}
			if sim.ExpiresAt.After(now) && sim.ExpiresAt.Sub(now) <= warn {
				key := "sim-expiry:" + sim.IMSI
				seen[key] = true
				s.fireAlertLocked(api.AlertWarning, key, fmt.Sprintf("SIM expires %s", sim.ExpiresAt.Format(time.RFC3339)), sim.IMSI, "")
			}
		}
	}

	// UE capacity
	p := s.store.GetPolicy()
	if p.MaxUEs > 0 {
		ueCount := len(s.store.ListSessions())
		pct := s.cfg.Alerts.UesCapacityPct
		if pct <= 0 {
			pct = 90
		}
		if ueCount*100 >= p.MaxUEs*pct {
			key := "ues-capacity"
			seen[key] = true
			s.fireAlertLocked(api.AlertWarning, key, fmt.Sprintf("UE count %d reached %d%% of capacity %d", ueCount, pct, p.MaxUEs), "", "")
		}
	}

	// resolve alerts whose condition cleared
	for key := range s.fired {
		if seen[key] {
			continue
		}
		s.resolveAlertLocked(ctx, key, now)
	}
	s.refreshMetrics()
}

// fireAlertLocked records + delivers an alert if it has not already fired.
// Caller must hold s.mu.
func (s *Server) fireAlertLocked(sev api.AlertSeverity, key, message, target, node string) {
	if s.fired[key] {
		return
	}
	s.fired[key] = true
	a := &api.Alert{
		ID:       fmt.Sprintf("al-%s", hex.EncodeToString(randBytes(4))),
		Key:      key,
		Severity: sev,
		Message:  message,
		Target:   target,
		Node:     node,
		TS:       s.now().UTC(),
	}
	if err := s.store.AppendAlert(a); err != nil {
		log.Printf("alert %s: %v", key, err)
	}
	s.deliverWebhook(a, false)
}

// resolveAlertLocked marks a previously fired alert as resolved.
// Caller must hold s.mu.
func (s *Server) resolveAlertLocked(ctx context.Context, key string, now time.Time) {
	delete(s.fired, key)
	for _, a := range s.store.ActiveAlerts() {
		if a.Key != key {
			continue
		}
		if err := s.store.ResolveAlert(a.ID, now); err != nil {
			log.Printf("resolve alert %s: %v", key, err)
		}
		s.deliverWebhook(a, true)
		return
	}
}

// deliverWebhook POSTs an alert (or resolution) to every configured
// webhook. Fire-and-forget: delivery failures are logged, never fatal.
func (s *Server) deliverWebhook(a *api.Alert, resolved bool) {
	if len(s.cfg.Alerts.Webhooks) == 0 {
		return
	}
	event := map[string]any{
		"event":    "alert",
		"id":       a.ID,
		"severity": string(a.Severity),
		"message":  a.Message,
		"target":   a.Target,
		"node":     a.Node,
		"ts":       a.TS,
		"resolved": resolved,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return
	}
	for _, url := range s.cfg.Alerts.Webhooks {
		go func(u string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
			if err != nil {
				log.Printf("webhook %s: %v", u, err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("webhook %s: %v", u, err)
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(url)
	}
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(i)
		}
	}
	return b
}

// quotaFor reads the quota stored on the SIM record.
func quotaFor(st *store.Store, imsi string) uint64 {
	if sim, ok := st.GetSIM(imsi); ok {
		return sim.QuotaBytes
	}
	return 0
}

// ---- metrics ----

var (
	metricsOnce sync.Once

	mNodes    = prometheus.NewGauge(prometheus.GaugeOpts{Name: "fairwave_nodes", Help: "registered nodes"})
	mSIMs     = prometheus.NewGauge(prometheus.GaugeOpts{Name: "fairwave_sims", Help: "SIMs in store"})
	mPeers    = prometheus.NewGauge(prometheus.GaugeOpts{Name: "fairwave_peers", Help: "known peers"})
	mSessions = prometheus.NewGauge(prometheus.GaugeOpts{Name: "fairwave_sessions", Help: "active sessions"})
	mTxArmed  = prometheus.NewGauge(prometheus.GaugeOpts{Name: "fairwave_tx_armed", Help: "1 if TX is armed"})
	mTxDenied = prometheus.NewCounter(prometheus.CounterOpts{Name: "fairwave_tx_denied_total", Help: "denied TX arm attempts"})
	mSpectrum = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "fairwave_spectrum_checks_total", Help: "spectrum checks"}, []string{"allowed"})
)

func (s *Server) registerMetrics() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(mNodes, mSIMs, mPeers, mSessions, mTxArmed, mTxDenied, mSpectrum)
	})
}

func (s *Server) refreshMetrics() {
	mNodes.Set(float64(len(s.store.ListNodes())))
	mSIMs.Set(float64(len(s.store.ListSIMs())))
	mPeers.Set(float64(len(s.store.ListPeers())))
	mSessions.Set(float64(len(s.store.ListSessions())))
	if s.txArmed {
		mTxArmed.Set(1)
	} else {
		mTxArmed.Set(0)
	}
}

var _ = api.LifecyclePhase("") // keep import linkage for API types used in routes
