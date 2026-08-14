package api

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

// ---- tokens ----

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	var req api.TokenCreateRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	resp, err := s.createToken(req.Name, req.Role)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	s.auditReq(r, "token_create", resp.ID, fmt.Sprintf("name=%s role=%s", resp.Name, resp.Role))
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleListTokens(w http.ResponseWriter, _ *http.Request) {
	toks := s.store.ListTokens()
	out := make([]api.Token, 0, len(toks))
	for _, t := range toks {
		c := *t          // copy: never mutate store state
		c.TokenHash = "" // verifier never leaves the store
		out = append(out, c)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.revokeToken(id); err != nil {
		writeErr(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	s.auditReq(r, "token_revoke", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// ---- SIM import ---- (tier-1: bureau batches)

func (s *Server) handleImportSIMs(w http.ResponseWriter, r *http.Request) {
	var req api.SimImportRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(req.Sims) == 0 {
		writeErr(w, http.StatusBadRequest, "bad_request", "sims required")
		return
	}
	resp := &api.SimImportResponse{}
	pol := s.store.GetPolicy()
	for _, item := range req.Sims {
		if len(item.IMSI) != 15 {
			resp.Skipped = append(resp.Skipped, item.IMSI+" (bad IMSI length)")
			continue
		}
		status := item.Status
		if status == "" {
			status = "issued"
		}
		profile := item.Profile
		if profile == "" {
			profile = "lab"
		}
		apn := item.APN
		if apn == "" {
			apn = "internet"
		}
		expires := item.ExpiresAt
		if expires.IsZero() {
			expires = time.Now().Add(365 * 24 * time.Hour)
		}
		existing, exists := s.store.GetSIM(item.IMSI)
		if exists {
			existing.Status = status
			existing.APN = apn
			existing.ExpiresAt = expires
			if item.MSISDN != "" {
				existing.MSISDN = item.MSISDN
			}
			if err := s.store.UpsertSIM(existing); err != nil {
				writeErr(w, http.StatusInternalServerError, "persist", err.Error())
				return
			}
			resp.Updated++
		} else {
			sim := &api.SIM{
				IMSI:      item.IMSI,
				MSISDN:    item.MSISDN,
				Profile:   profile,
				Status:    status,
				APN:       apn,
				CreatedAt: time.Now(),
				ExpiresAt: expires,
			}
			if err := s.store.UpsertSIM(sim); err != nil {
				writeErr(w, http.StatusInternalServerError, "persist", err.Error())
				return
			}
			resp.Imported++
			// Seed the HSS when the credentials are known (lab vectors). For
			// bureau cards the Ki/OPc stay in the vendor's vault, so the HSS
			// seed must happen out-of-band (docs/sim-lifecycle).
			if sub, err := simprov.LoadTestVector(item.IMSI); err == nil {
				sub.QoSDLMbps = pol.QoSDLMbps
				sub.QoSULMbps = pol.QoSULMbps
				if herr := s.hss.Add(r.Context(), sub); herr != nil {
					log.Printf("import %s: hss seed: %v", item.IMSI, herr)
				}
			}
		}
	}
	s.auditReq(r, "sim_import", fmt.Sprintf("x%d", len(req.Sims)),
		fmt.Sprintf("imported=%d updated=%d skipped=%d", resp.Imported, resp.Updated, len(resp.Skipped)))
	writeJSON(w, http.StatusOK, resp)
}

// ---- quota / usage ----

func (s *Server) handleSetQuota(w http.ResponseWriter, r *http.Request) {
	imsi := r.PathValue("imsi")
	sim, ok := s.store.GetSIM(imsi)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such SIM")
		return
	}
	var req api.SimQuotaRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	sim.QuotaBytes = req.QuotaBytes
	if err := s.store.UpsertSIM(sim); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	if u, ok2 := s.store.GetUsage(imsi); ok2 {
		u.QuotaBytes = req.QuotaBytes
		_ = s.store.UpsertUsage(u)
	}
	s.auditReq(r, "sim_quota", imsi, fmt.Sprintf("quota=%d", req.QuotaBytes))
	writeJSON(w, http.StatusOK, sim)
}

func (s *Server) handleGetUsage(w http.ResponseWriter, r *http.Request) {
	imsi := r.PathValue("imsi")
	u, ok := s.store.GetUsage(imsi)
	if !ok {
		sim, exists := s.store.GetSIM(imsi)
		if !exists {
			writeErr(w, http.StatusNotFound, "not_found", "no such SIM")
			return
		}
		u = &api.SimUsage{IMSI: imsi, IMSIHash: api.HashIMSI(imsi), QuotaBytes: sim.QuotaBytes}
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleReconcileUsage(w http.ResponseWriter, r *http.Request) {
	imsi := r.PathValue("imsi")
	if _, ok := s.store.GetSIM(imsi); !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such SIM")
		return
	}
	var req api.SimUsageRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	u, _ := s.store.GetUsage(imsi)
	if u == nil {
		u = &api.SimUsage{IMSI: imsi, IMSIHash: api.HashIMSI(imsi)}
	}
	u.BytesUp += req.BytesUp
	u.BytesDn += req.BytesDn
	u.UpdatedAt = time.Now()
	if sim, ok2 := s.store.GetSIM(imsi); ok2 {
		u.QuotaBytes = sim.QuotaBytes
	}
	if err := s.store.UpsertUsage(u); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	s.auditReq(r, "sim_usage_reconcile", imsi, fmt.Sprintf("+%d up +%d dn", req.BytesUp, req.BytesDn))
	writeJSON(w, http.StatusOK, u)
}

// ---- alerts ----

func (s *Server) handleListAlerts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListAlerts())
}

// ---- compliance export ----

// handleComplianceExport renders the regulator-ready CSV: the audit trail
// plus a summary of SIM inventory and TX arm history.
func (s *Server) handleComplianceExport(w http.ResponseWriter, _ *http.Request) {
	var sb strings.Builder
	cw := csv.NewWriter(&sb)

	_ = cw.Write([]string{"fairwave compliance report", "generated", s.now().UTC().Format(time.RFC3339), "node", s.id.ID})
	_ = cw.Write([]string{"mode", s.cfg.Server.Mode, "country", s.cfg.Server.Country, "tx_armed", fmt.Sprintf("%v", s.txArmed)})
	_ = cw.Write(nil)

	// SIM inventory summary
	sims := s.store.ListSIMs()
	counts := map[string]int{}
	for _, sim := range sims {
		counts[sim.Status]++
	}
	_ = cw.Write([]string{"sims_total", fmt.Sprintf("%d", len(sims)),
		"issued", fmt.Sprintf("%d", counts["issued"]),
		"active", fmt.Sprintf("%d", counts["active"]),
		"suspended", fmt.Sprintf("%d", counts["suspended"]),
		"revoked", fmt.Sprintf("%d", counts["revoked"]),
		"expired", fmt.Sprintf("%d", counts["expired"])})
	_ = cw.Write(nil)

	// TX arm history from the audit trail
	_ = cw.Write([]string{"--- TX arm history (audit) ---"})
	_ = cw.Write([]string{"ts", "principal", "action", "target", "detail"})
	for _, e := range s.store.ListAudit() {
		if !strings.HasPrefix(e.Action, "tx_") {
			continue
		}
		_ = cw.Write([]string{e.TS.Format(time.RFC3339), e.Principal, e.Action, e.Target, e.Detail})
	}
	_ = cw.Write(nil)

	// Full audit trail
	_ = cw.Write([]string{"--- full audit trail ---"})
	_ = cw.Write([]string{"ts", "principal", "action", "target", "detail"})
	for _, e := range s.store.ListAudit() {
		_ = cw.Write([]string{e.TS.Format(time.RFC3339), e.Principal, e.Action, e.Target, e.Detail})
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="fairwave-compliance-%s.csv"`, s.now().UTC().Format("20060102")))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sb.String()))
}
