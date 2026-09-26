package api

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/bearer"
)

// ---- threads ----

func (s *Server) handleHydraStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.hydra.Status())
}

func (s *Server) handleHydraListThreads(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.hydra.ListThreads())
}

func (s *Server) handleHydraAddThread(w http.ResponseWriter, r *http.Request) {
	var req api.HydraThreadRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t := bearer.Thread{
		ID:      req.ID,
		Box:     req.Box,
		SIM:     req.SIM,
		Mbps:    req.Mbps,
		RTTms:   req.RTTms,
		LossPct: req.LossPct,
		Up:      req.Up,
	}
	if err := s.hydra.AddThread(t); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	s.auditReq(r, "hydra_thread_upsert", req.ID, fmt.Sprintf("mbps=%.1f up=%v", req.Mbps, req.Up))
	stored, ok := s.hydra.Thread(req.ID)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "internal", "thread vanished after upsert")
		return
	}
	if err := s.store.UpsertHydraThread(&stored); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, stored)
}

func (s *Server) handleHydraDeleteThread(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.hydra.RemoveThread(id) {
		writeErr(w, http.StatusNotFound, "not_found", "no such thread")
		return
	}
	if err := s.store.DeleteHydraThread(id); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	s.auditReq(r, "hydra_thread_remove", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// handleHydraThreadHealth ingests a running node's measured link health.
// It is deliberately not audited: it fires on a heartbeat cadence and
// would otherwise drown the append-only regulatory trail.
func (s *Server) handleHydraThreadHealth(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req api.HydraThreadHealth
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if !s.hydra.SetThreadHealth(id, req.Mbps, req.RTTms, req.LossPct, req.Up) {
		writeErr(w, http.StatusNotFound, "not_found", "no such thread")
		return
	}
	if t, ok := s.hydra.Thread(id); ok {
		if err := s.store.UpsertHydraThread(&t); err != nil {
			writeErr(w, http.StatusInternalServerError, "persist", err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "up": req.Up})
}

// ---- weaves ----

func (s *Server) handleHydraListWeaves(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.hydra.ListWeaves())
}

func (s *Server) handleHydraGetWeave(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	wv, ok := s.hydra.GetWeave(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such weave")
		return
	}
	writeJSON(w, http.StatusOK, wv)
}

func (s *Server) handleHydraCreateWeave(w http.ResponseWriter, r *http.Request) {
	var req api.HydraWeaveRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	wv, err := s.hydra.CreateWeave(req.ID, req.Anchor, req.Threads)
	if err != nil {
		writeErr(w, http.StatusConflict, "hydra_weave", err.Error())
		return
	}
	s.auditReq(r, "hydra_weave_create", wv.ID, fmt.Sprintf("anchor=%s threads=%d", wv.Anchor, len(wv.Threads)))
	if err := s.store.UpsertHydraWeave(&wv); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, wv)
}

func (s *Server) handleHydraDeleteWeave(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.hydra.DeleteWeave(id) {
		writeErr(w, http.StatusNotFound, "not_found", "no such weave")
		return
	}
	if err := s.store.DeleteHydraWeave(id); err != nil {
		writeErr(w, http.StatusInternalServerError, "persist", err.Error())
		return
	}
	s.auditReq(r, "hydra_weave_delete", id, "")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHydraWeaveStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stats, ok := s.hydra.WeaveStats(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such weave")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// ---- lab data path ----

// handleHydraStrip assigns one payload to a thread and returns the framed
// packet, letting an operator drive the weave by hand in the lab.
func (s *Server) handleHydraStrip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req api.HydraStripRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	payload, err := base64.StdEncoding.DecodeString(req.PayloadB64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "payload_b64 is not valid base64")
		return
	}
	res, err := s.hydra.Strip(id, payload)
	if err != nil {
		writeErr(w, http.StatusConflict, "hydra_strip", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, api.HydraStripResponse{
		WeaveID:  res.WeaveID,
		ThreadID: res.ThreadID,
		Seq:      res.Seq,
		FrameLen: res.FrameLen,
		FrameB64: base64.StdEncoding.EncodeToString(res.Frame),
	})
}

// handleHydraIngest feeds a framed packet back into reassembly.
func (s *Server) handleHydraIngest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req api.HydraIngestRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	frame, err := base64.StdEncoding.DecodeString(req.FrameB64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "frame_b64 is not valid base64")
		return
	}
	n, err := s.hydra.Ingest(id, frame)
	if err != nil {
		writeErr(w, http.StatusConflict, "hydra_ingest", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, api.HydraIngestResponse{Delivered: n})
}

// handleHydraBench runs the synthetic stripe/reassemble bench.
func (s *Server) handleHydraBench(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req api.HydraBenchRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Packets == 0 {
		req.Packets = 300
	}
	if req.PktBytes == 0 {
		req.PktBytes = 1200
	}
	res, err := s.hydra.Bench(id, req.Packets, req.PktBytes)
	if err != nil {
		writeErr(w, http.StatusConflict, "hydra_bench", err.Error())
		return
	}
	s.auditReq(r, "hydra_bench", id, fmt.Sprintf("packets=%d speedup=%.2f", res.Packets, res.Speedup))
	writeJSON(w, http.StatusOK, res)
}
