package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/lifecycle"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/simops"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /v1/healthz", s.handleHealthz)
	m.HandleFunc("GET /v1/version", s.handleVersion)
	m.HandleFunc("GET /v1/status", s.handleStatus)
	m.HandleFunc("POST /v1/lifecycle/transition", s.handleLifecycle)

	m.HandleFunc("GET /v1/nodes", s.handleListNodes)
	m.HandleFunc("POST /v1/nodes", s.handleCreateNode)
	m.HandleFunc("GET /v1/nodes/{id}", s.handleGetNode)
	m.HandleFunc("POST /v1/nodes/{id}/enroll", s.handleEnrollNode)
	m.HandleFunc("POST /v1/nodes/{id}/leave", s.handleLeaveNode)

	m.HandleFunc("GET /v1/sims", s.handleListSIMs)
	m.HandleFunc("POST /v1/sims", s.handleIssueSIMs)
	m.HandleFunc("POST /v1/sims/{imsi}/revoke", s.handleRevokeSIM)

	m.HandleFunc("GET /v1/peers", s.handleListPeers)
	m.HandleFunc("POST /v1/peers", s.handleAddPeer)
	m.HandleFunc("DELETE /v1/peers/{id}", s.handleDeletePeer)

	m.HandleFunc("GET /v1/sessions", s.handleListSessions)
	m.HandleFunc("GET /v1/policy", s.handleGetPolicy)
	m.HandleFunc("PUT /v1/policy", s.handlePutPolicy)

	m.HandleFunc("POST /v1/spectrum/check", s.handleSpectrumCheck)
	m.HandleFunc("POST /v1/tx/arm", s.handleTxArm)

	m.HandleFunc("GET /metrics", promhttp.Handler().ServeHTTP)
}

// ---- middleware ----

func (s *Server) logMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s (%s)", r.RemoteAddr, r.Method, r.URL.Path, time.Since(start))
	})
}

// authMW requires a Bearer token; with no configured token it allows
// loopback only (safe lab default).
func (s *Server) authMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/healthz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok == "" && s.adminTok == "" {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil || net.ParseIP(host).IsLoopback() {
				next.ServeHTTP(w, r)
				return
			}
			writeErr(w, http.StatusUnauthorized, "auth_required", "admin token not configured; loopback only")
			return
		}
		if s.adminTok != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(s.adminTok)) == 1 {
			next.ServeHTTP(w, r)
			return
		}
		writeErr(w, http.StatusUnauthorized, "auth_required", "invalid or missing bearer token")
	})
}

func (s *Server) recoverMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic serving %s: %v", r.URL.Path, rec)
				writeErr(w, http.StatusInternalServerError, "internal", "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, errCode, msg string) {
	writeJSON(w, code, api.ErrorBody{Error: api.ErrorDetail{Code: errCode, Message: msg}})
}

func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	return dec.Decode(v)
}

// ---- handlers ----

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": Version,
		"mode":    s.cfg.Server.Mode,
		"country": s.cfg.Server.Country,
		"node_id": s.id.ID,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	phase := api.PhaseProvision
	if nodes := s.store.ListNodes(); len(nodes) > 0 {
		phase = nodes[0].Phase
	}
	st := api.Status{
		Version:   Version,
		Mode:      s.cfg.Server.Mode,
		Phase:     string(phase),
		TxArmed:   s.txArmed,
		Country:   s.cfg.Server.Country,
		Nodes:     len(s.store.ListNodes()),
		UEs:       len(s.store.ListSessions()),
		Peers:     len(s.store.ListPeers()),
		UptimeSec: int64(time.Since(s.started).Seconds()),
	}
	s.refreshMetrics()
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handleLifecycle(w http.ResponseWriter, r *http.Request) {
	var req api.LifecycleTransitionRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	nodes := s.store.ListNodes()
	if len(nodes) == 0 {
		writeErr(w, http.StatusConflict, "no_node", "create a node first")
		return
	}
	n := nodes[0]
	if err := lifecycle.Transition(n.Phase, req.Phase); err != nil {
		writeErr(w, http.StatusConflict, "bad_transition", err.Error())
		return
	}
	n.Phase = req.Phase
	n.UpdatedAt = time.Now()
	if err := s.store.UpsertNode(n); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// ---- nodes ----

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListNodes())
}

func (s *Server) handleCreateNode(w http.ResponseWriter, r *http.Request) {
	var req api.Node
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("node-%d", time.Now().UnixNano())
	}
	if req.Country == "" {
		req.Country = s.cfg.Server.Country
	}
	req.Phase = api.PhaseProvision
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	if err := s.store.UpsertNode(&req); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n, ok := s.store.GetNode(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such node")
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleEnrollNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n, ok := s.store.GetNode(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such node")
		return
	}
	var req api.EnrollRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.BootstrapToken == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "bootstrap_token required")
		return
	}
	n.Phase = api.PhaseRegister
	n.UpdatedAt = time.Now()
	if err := s.store.UpsertNode(n); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleLeaveNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.store.GetNode(id); !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such node")
		return
	}
	if err := s.store.DeleteNode(id); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- sims ----

func (s *Server) handleListSIMs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListSIMs())
}

func (s *Server) handleIssueSIMs(w http.ResponseWriter, r *http.Request) {
	var req api.SimIssueRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Count <= 0 || req.Count > 1000 {
		writeErr(w, http.StatusBadRequest, "bad_request", "count must be 1..1000")
		return
	}
	if req.Profile == "" {
		req.Profile = "lab"
	}
	subs, err := simops.GenerateBatch(simops.BatchSpec{
		Count:    req.Count,
		Class:    req.Profile,
		IMSIBase: 999991234567001,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	for _, sub := range subs {
		// Write back to the HSS first so a failed store never leaves an
		// orphan subscriber; on store failure roll the HSS entry back.
		if err := s.hss.Add(r.Context(), sub); err != nil {
			writeErr(w, http.StatusBadGateway, "hss_writeback", err.Error())
			return
		}
		sim := &api.SIM{
			IMSI:      sub.IMSI,
			MSISDN:    sub.MSISDN,
			Profile:   req.Profile,
			Status:    "issued",
			APN:       sub.APN,
			SQN:       sub.SQN,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
		}
		if err := s.store.UpsertSIM(sim); err != nil {
			_ = s.hss.Remove(r.Context(), sub.IMSI)
			writeErr(w, http.StatusInternalServerError, "persist", err.Error())
			return
		}
	}
	writeJSON(w, http.StatusCreated, s.store.ListSIMs())
}

func (s *Server) handleRevokeSIM(w http.ResponseWriter, r *http.Request) {
	imsi := r.PathValue("imsi")
	sim, ok := s.store.GetSIM(imsi)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such SIM")
		return
	}
	// Remove from the HSS first; only mark revoked once the network can
	// no longer accept the SIM (idempotent on retry).
	if err := s.hss.Remove(r.Context(), imsi); err != nil {
		writeErr(w, http.StatusBadGateway, "hss_writeback", err.Error())
		return
	}
	sim.Status = "revoked"
	if err := s.store.UpsertSIM(sim); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sim)
}

// ---- peers ----

func (s *Server) handleListPeers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListPeers())
}

func (s *Server) handleAddPeer(w http.ResponseWriter, r *http.Request) {
	var req api.Peer
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("peer-%d", time.Now().UnixNano())
	}
	req.Status = "pending"
	req.LastSeen = time.Now()
	if err := s.store.UpsertPeer(&req); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (s *Server) handleDeletePeer(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeletePeer(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- sessions / policy ----

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListSessions())
}

func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.GetPolicy())
}

func (s *Server) handlePutPolicy(w http.ResponseWriter, r *http.Request) {
	var p api.Policy
	if err := readJSON(r, &p); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if p.MaxUEs <= 0 {
		p.MaxUEs = 128
	}
	if len(p.APNs) == 0 {
		p.APNs = []string{"internet"}
	}
	if err := s.store.SetPolicy(&p); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.store.GetPolicy())
}

// ---- spectrum / tx ----

func (s *Server) handleSpectrumCheck(w http.ResponseWriter, r *http.Request) {
	var req api.SpectrumCheckRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	verdict := Verdict{Allowed: false, Reasons: []string{"no spectrum registry configured"}}
	if s.spectrum != nil {
		verdict = s.spectrum.Check(req.Country, req.Band, req.Indoor, req.LicenseRef)
	}
	if verdict.Allowed {
		mSpectrum.WithLabelValues("allowed").Inc()
	} else {
		mSpectrum.WithLabelValues("denied").Inc()
	}
	writeJSON(w, http.StatusOK, api.SpectrumCheckResponse{Allowed: verdict.Allowed, Reasons: verdict.Reasons})
}

func (s *Server) handleTxArm(w http.ResponseWriter, r *http.Request) {
	var req api.TxArmRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	var reasons []string
	allowed := true

	if s.cfg.Server.Mode == "lab" {
		allowed = false
		reasons = append(reasons, "server.mode=lab: RF TX is disabled by configuration")
	}
	if req.Acknowledgment != "I hold authorization for this transmission" {
		allowed = false
		reasons = append(reasons, "acknowledgment phrase missing or wrong")
	}
	if s.spectrum != nil {
		v := s.spectrum.Check(req.Country, req.Band, true, req.LicenseRef)
		if !v.Allowed {
			allowed = false
			reasons = append(reasons, v.Reasons...)
		}
	}
	if allowed {
		s.txArmed = true
		s.refreshMetrics()
	} else {
		mTxDenied.Inc()
	}
	writeJSON(w, http.StatusOK, api.TxArmResponse{Armed: allowed, Reasons: reasons})
}
