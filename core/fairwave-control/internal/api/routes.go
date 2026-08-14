package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/activation"
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
	m.HandleFunc("GET /v1/sims/{imsi}", s.handleGetSIM)
	m.HandleFunc("POST /v1/sims", s.handleIssueSIMs)
	m.HandleFunc("POST /v1/sims/{imsi}/revoke", s.handleRevokeSIM)
	m.HandleFunc("POST /v1/sims/{imsi}/suspend", s.handleSuspendSIM)
	m.HandleFunc("POST /v1/sims/{imsi}/resume", s.handleResumeSIM)
	m.HandleFunc("POST /v1/sims/import", s.handleImportSIMs)
	m.HandleFunc("POST /v1/sims/{imsi}/quota", s.handleSetQuota)
	m.HandleFunc("GET /v1/sims/{imsi}/usage", s.handleGetUsage)
	m.HandleFunc("POST /v1/sims/{imsi}/usage", s.handleReconcileUsage)

	m.HandleFunc("GET /v1/peers", s.handleListPeers)
	m.HandleFunc("POST /v1/peers", s.handleAddPeer)
	m.HandleFunc("DELETE /v1/peers/{id}", s.handleDeletePeer)

	m.HandleFunc("GET /v1/sessions", s.handleListSessions)
	m.HandleFunc("GET /v1/policy", s.handleGetPolicy)
	m.HandleFunc("PUT /v1/policy", s.handlePutPolicy)

	m.HandleFunc("POST /v1/spectrum/check", s.handleSpectrumCheck)
	m.HandleFunc("POST /v1/tx/arm", s.handleTxArm)
	m.HandleFunc("POST /v1/tx/disarm", s.handleTxDisarm)

	m.HandleFunc("POST /v1/telemetry", s.handleTelemetry)
	m.HandleFunc("GET /v1/health", s.handleListHealth)
	m.HandleFunc("GET /v1/audit", s.handleAudit)
	m.HandleFunc("GET /v1/alerts", s.handleListAlerts)

	m.HandleFunc("POST /v1/tokens", s.handleCreateToken)
	m.HandleFunc("GET /v1/tokens", s.handleListTokens)
	m.HandleFunc("DELETE /v1/tokens/{id}", s.handleRevokeToken)

	m.HandleFunc("GET /v1/backup", s.handleBackup)
	m.HandleFunc("POST /v1/restore", s.handleRestore)
	m.HandleFunc("GET /v1/compliance/export", s.handleComplianceExport)

	m.HandleFunc("POST /v1/esim/issue", s.handleEsimIssue)
	m.HandleFunc("GET /v1/esim/codes", s.handleEsimList)
	m.HandleFunc("GET /v1/esim/codes/{token}/qr", s.handleEsimQR)
	m.HandleFunc("POST /v1/esim/revoke", s.handleEsimRevoke)

	// ES9+ (eUICC download flow) is served by the embedded SM-DP+. It does
	// its own challenge/response crypto and must be reachable by a phone's
	// LPA, so authMW bypasses this subtree (like /metrics).
	m.Handle("/es9plus/", s.esimHandler())

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

// principalKey is the context key carrying the authenticated principal
// name (token name, "admin", or "loopback").
type principalKey struct{}

// principalFrom returns the authenticated principal recorded by authMW.
func principalFrom(r *http.Request) string {
	if p, ok := r.Context().Value(principalKey{}).(string); ok && p != "" {
		return p
	}
	return "unknown"
}

// authMW authenticates the request, resolves the role, and enforces the
// per-route RBAC rules. The admin token (or unauthenticated loopback in
// lab mode) is RoleAdmin; scoped tokens get their own role. The principal
// name is stored in the context for audit attribution.
func (s *Server) authMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/healthz" || r.URL.Path == "/metrics" || strings.HasPrefix(r.URL.Path, "/es9plus/") {
			next.ServeHTTP(w, r)
			return
		}
		role := api.RoleAdmin
		principal := "admin"
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		switch {
		case s.adminTok != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(s.adminTok)) == 1:
			// admin token: full access
		case tok == "" && s.adminTok == "":
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil || !net.ParseIP(host).IsLoopback() {
				writeErr(w, http.StatusUnauthorized, "auth_required", "admin token not configured; loopback only")
				return
			}
			principal = "loopback"
		default:
			sum := sha256.Sum256([]byte(tok))
			t, ok := s.store.TokenByHash(hex.EncodeToString(sum[:]))
			if !ok {
				writeErr(w, http.StatusUnauthorized, "auth_required", "invalid or missing bearer token")
				return
			}
			role = t.Role
			principal = t.Name
		}
		if err := authorize(role, r.Method, r.URL.Path); err != nil {
			writeErr(w, http.StatusForbidden, "forbidden", err.Error())
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, principal)))
	})
}

// authorize enforces the least-privilege matrix:
//   - admin: everything
//   - operator: mutating SIM/eSIM/peer/policy/lifecycle + telemetry
//   - viewer: read-only GETs
//
// Regulatory and key-management surfaces (TX, audit, compliance, tokens,
// backup/restore) are admin-only regardless of role.
func authorize(role api.TokenRole, method, path string) error {
	if role == api.RoleAdmin {
		return nil
	}
	adminOnly := func() bool {
		return strings.HasPrefix(path, "/v1/tokens") ||
			strings.HasPrefix(path, "/v1/backup") ||
			strings.HasPrefix(path, "/v1/restore") ||
			strings.HasPrefix(path, "/v1/tx/arm") ||
			strings.HasPrefix(path, "/v1/tx/disarm") ||
			strings.HasPrefix(path, "/v1/compliance") ||
			path == "/v1/audit"
	}
	if adminOnly() {
		return fmt.Errorf("this surface requires the admin token")
	}
	if method == http.MethodGet {
		return nil // viewer can read
	}
	if role == api.RoleViewer {
		return fmt.Errorf("viewer tokens are read-only")
	}
	return nil // operator
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

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": Version,
		"mode":    s.cfg.Server.Mode,
		"country": s.cfg.Server.Country,
		"node_id": s.id.ID,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
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
	s.auditReq(r, "lifecycle_transition", n.ID, fmt.Sprintf("%s -> %s", n.Phase, req.Phase))
	writeJSON(w, http.StatusOK, n)
}

// ---- nodes ----

func (s *Server) handleListNodes(w http.ResponseWriter, _ *http.Request) {
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
	s.auditReq(r, "node_enroll", id, "")
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
	s.auditReq(r, "node_leave", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// ---- sims ----

func (s *Server) handleListSIMs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListSIMs())
}

func (s *Server) handleGetSIM(w http.ResponseWriter, r *http.Request) {
	imsi := r.PathValue("imsi")
	sim, ok := s.store.GetSIM(imsi)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such SIM")
		return
	}
	writeJSON(w, http.StatusOK, sim)
}

// handleSuspendSIM deactivates a SIM without revoking its credentials.
// It stays in the store so it can be resumed later.
func (s *Server) handleSuspendSIM(w http.ResponseWriter, r *http.Request) {
	imsi := r.PathValue("imsi")
	sim, ok := s.store.GetSIM(imsi)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such SIM")
		return
	}
	if sim.Status != "active" && sim.Status != "issued" {
		writeErr(w, http.StatusConflict, "bad_state", fmt.Sprintf("cannot suspend a %s SIM", sim.Status))
		return
	}
	sim.Status = "suspended"
	if err := s.store.UpsertSIM(sim); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	s.auditReq(r, "sim_suspend", imsi, "")
	writeJSON(w, http.StatusOK, sim)
}

// handleResumeSIM reactivates a suspended SIM.
func (s *Server) handleResumeSIM(w http.ResponseWriter, r *http.Request) {
	imsi := r.PathValue("imsi")
	sim, ok := s.store.GetSIM(imsi)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such SIM")
		return
	}
	if sim.Status != "suspended" {
		writeErr(w, http.StatusConflict, "bad_state", fmt.Sprintf("cannot resume a %s SIM", sim.Status))
		return
	}
	sim.Status = "active"
	if err := s.store.UpsertSIM(sim); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	s.auditReq(r, "sim_resume", imsi, "")
	writeJSON(w, http.StatusOK, sim)
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
	// The per-subscriber AMBR caps come from the operator policy (fair-use).
	pol := s.store.GetPolicy()
	for i := range subs {
		subs[i].QoSDLMbps = pol.QoSDLMbps
		subs[i].QoSULMbps = pol.QoSULMbps
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
	s.auditReq(r, "sim_issue", fmt.Sprintf("x%d", req.Count), req.Profile)
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
	s.auditReq(r, "sim_revoke", imsi, "removed from HSS")
	writeJSON(w, http.StatusOK, sim)
}

// ---- peers ----

func (s *Server) handleListPeers(w http.ResponseWriter, _ *http.Request) {
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
	s.auditReq(r, "peer_add", req.ID, req.Endpoint)
	writeJSON(w, http.StatusCreated, req)
}

func (s *Server) handleDeletePeer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeletePeer(id); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	s.auditReq(r, "peer_delete", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// ---- sessions / policy ----

func (s *Server) handleListSessions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListSessions())
}

func (s *Server) handleGetPolicy(w http.ResponseWriter, _ *http.Request) {
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
	s.auditReq(r, "policy_update", "", fmt.Sprintf("breakout=%v max_ues=%d qos=%d/%d", p.LocalBreakout, p.MaxUEs, p.QoSDLMbps, p.QoSULMbps))
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
		if err := s.store.SetTxArmed(true); err != nil {
			writeErr(w, http.StatusInternalServerError, "persist", err.Error())
			return
		}
		for _, n := range s.store.ListNodes() {
			n.TxArmed = true
			_ = s.store.UpsertNode(n)
		}
		s.refreshMetrics()
	}
	// Every arm attempt lands in the audit log - including denials - so the
	// regulatory trail shows what was requested, when, and why it was refused.
	s.auditReq(r, "tx_arm", fmt.Sprintf("%s/%s", req.Country, req.Band),
		fmt.Sprintf("allowed=%v indoor=%v license=%q", allowed, true, req.LicenseRef))
	if !allowed {
		mTxDenied.Inc()
	}
	writeJSON(w, http.StatusOK, api.TxArmResponse{Armed: allowed, Reasons: reasons})
}

// handleTxDisarm clears the armed flag and records the action. Disarming
// is always allowed (it can only reduce risk).
func (s *Server) handleTxDisarm(w http.ResponseWriter, r *http.Request) {
	if !s.txArmed {
		writeJSON(w, http.StatusOK, api.TxArmResponse{Armed: false, Reasons: nil})
		return
	}
	s.txArmed = false
	if err := s.store.SetTxArmed(false); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	for _, n := range s.store.ListNodes() {
		n.TxArmed = false
		_ = s.store.UpsertNode(n)
	}
	s.auditReq(r, "tx_disarm", "", "")
	s.refreshMetrics()
	writeJSON(w, http.StatusOK, api.TxArmResponse{Armed: false, Reasons: nil})
}

// ---- telemetry / health ----

// handleTelemetry ingests an agent heartbeat. The agent and the control
// plane share the NodeHealth JSON shape, so no transformation is needed.
func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	var h api.NodeHealth
	if err := readJSON(r, &h); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if h.NodeID == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "node_id required")
		return
	}
	if h.TS.IsZero() {
		h.TS = time.Now().UTC()
	}
	if err := s.store.UpsertHealth(&h); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleListHealth returns the latest heartbeat per node, with Up computed
// from recency so the dashboard can show stale/dead boxes.
func (s *Server) handleListHealth(w http.ResponseWriter, _ *http.Request) {
	stale := s.cfg.Telemetry.StaleAfter
	if stale <= 0 {
		stale = 90 * time.Second
	}
	out := s.store.ListHealth()
	now := s.now().UTC()
	for _, h := range out {
		h.Up = now.Sub(h.TS) <= stale
	}
	writeJSON(w, http.StatusOK, out)
}

// ---- audit ----

func (s *Server) handleAudit(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListAudit())
}

// ---- esim ----

// esimHandler returns the embedded SM-DP+ handler (503 when disabled).
func (s *Server) esimHandler() http.Handler {
	if s.esim == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeErr(w, http.StatusServiceUnavailable, "esim_disabled", "eSIM server is not configured")
		})
	}
	return s.esim.Handler()
}

func (s *Server) handleEsimIssue(w http.ResponseWriter, r *http.Request) {
	if s.esim == nil || !s.esim.enabled {
		writeErr(w, http.StatusServiceUnavailable, "esim_disabled", "eSIM server is not configured")
		return
	}
	var req api.EsimIssueRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	resp, err := s.issueEsim(&req, principalFrom(r))
	if err != nil {
		writeErr(w, http.StatusConflict, "esim_issue", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleEsimList(w http.ResponseWriter, _ *http.Request) {
	if s.esim == nil || !s.esim.enabled {
		writeErr(w, http.StatusServiceUnavailable, "esim_disabled", "eSIM server is not configured")
		return
	}
	writeJSON(w, http.StatusOK, s.listEsim())
}

func (s *Server) handleEsimQR(w http.ResponseWriter, r *http.Request) {
	if s.esim == nil || !s.esim.enabled {
		writeErr(w, http.StatusServiceUnavailable, "esim_disabled", "eSIM server is not configured")
		return
	}
	token := r.PathValue("token")
	var code *activation.Code
	for _, e := range s.esim.reg.List() {
		if e.Token == token {
			code = activation.New(s.esim.address, token)
			break
		}
	}
	if code == nil {
		writeErr(w, http.StatusNotFound, "not_found", "no such activation code")
		return
	}
	png, err := code.QR()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

func (s *Server) handleEsimRevoke(w http.ResponseWriter, r *http.Request) {
	var req api.EsimRevokeRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := s.revokeEsim(req.ActivationCode, principalFrom(r)); err != nil {
		writeErr(w, http.StatusConflict, "esim_revoke", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
