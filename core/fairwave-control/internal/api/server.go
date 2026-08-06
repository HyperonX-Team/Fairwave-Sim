// Package api implements the northbound REST API for fairwave-control.
// Auth: Bearer token from FAIRWAVE_ADMIN_TOKEN (or auto-generated on first
// run and written to data/admin_token, 0600). In lab mode with no token
// configured, only loopback is accepted.
package api

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
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
	cfg      *config.ControlConfig
	store    *store.Store
	id       *identity.Identity
	adminTok string
	txArmed  bool
	started  time.Time
	spectrum SpectrumChecker
	hss      hsswrite.Writer
	mux      *http.ServeMux
}

// New creates the API server. sc may be nil (all spectrum checks denied).
// hss may be nil (SIM issuance/revocation is store-only).
func New(cfg *config.ControlConfig, st *store.Store, id *identity.Identity, sc SpectrumChecker, hss hsswrite.Writer) *Server {
	if hss == nil {
		hss = hsswrite.None{}
	}
	s := &Server{
		cfg:      cfg,
		store:    st,
		id:       id,
		spectrum: sc,
		hss:      hss,
		started:  time.Now(),
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
