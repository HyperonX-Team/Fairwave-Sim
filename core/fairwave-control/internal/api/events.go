package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// snapshotEvent is the live payload pushed over SSE to the dashboard.
// It is intentionally small: full tables still come from the REST
// endpoints; this is the 5s heartbeat that makes the UI feel live
// without 9-way polling.
type snapshotEvent struct {
	Now       time.Time `json:"now"`
	Version   string    `json:"version"`
	Mode      string    `json:"mode"`
	Phase     string    `json:"phase"`
	TxArmed   bool      `json:"tx_armed"`
	Nodes     int       `json:"nodes"`
	SIMs      int       `json:"sims"`
	Sessions  int       `json:"sessions"`
	Peers     int       `json:"peers"`
	Alerts    int       `json:"alerts_active"`
	Critical  int       `json:"alerts_critical"`
	UpNodes   int       `json:"nodes_up"`
	BytesUp   uint64    `json:"bytes_up"`
	BytesDn   uint64    `json:"bytes_dn"`
	UptimeSec int64     `json:"uptime_sec"`
}

// handleEvents serves GET /v1/events as Server-Sent Events.
// Query: ?interval=5s (default 5s, min 1s, max 60s).
// Auth: same RBAC as other GETs (viewer OK). The stream ends when the
// client disconnects; each message is `event: snapshot` + JSON data.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	interval := 5 * time.Second
	if q := r.URL.Query().Get("interval"); q != "" {
		if d, err := time.ParseDuration(q); err == nil {
			if d < time.Second {
				d = time.Second
			}
			if d > time.Minute {
				d = time.Minute
			}
			interval = d
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "internal", "streaming unsupported")
		return
	}

	// Initial comment keeps proxies from buffering; client ignores it.
	fmt.Fprintf(w, ": fairwave live events (interval %s)\n\n", interval)
	flusher.Flush()

	t := time.NewTicker(interval)
	defer t.Stop()

	send := func() bool {
		snap := s.buildSnapshot()
		data, err := json.Marshal(snap)
		if err != nil {
			return true
		}
		if _, err := fmt.Fprintf(w, "event: snapshot\ndata: %s\n\n", data); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !send() {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-t.C:
			if !send() {
				return
			}
		}
	}
}

func (s *Server) buildSnapshot() snapshotEvent {
	phase := string(api.PhaseProvision)
	if nodes := s.store.ListNodes(); len(nodes) > 0 {
		phase = string(nodes[0].Phase)
	}
	var up, crit int
	for _, a := range s.store.ActiveAlerts() {
		if a.Severity == api.AlertCritical {
			crit++
		}
	}
	active := len(s.store.ActiveAlerts())

	stale := s.cfg.Telemetry.StaleAfter
	if stale <= 0 {
		stale = 90 * time.Second
	}
	now := s.now().UTC()
	for _, h := range s.store.ListHealth() {
		if now.Sub(h.TS) <= stale {
			up++
		}
	}

	var bytesUp, bytesDn uint64
	for _, sess := range s.store.ListSessions() {
		bytesUp += sess.BytesUp
		bytesDn += sess.BytesDn
	}

	return snapshotEvent{
		Now:       now,
		Version:   Version,
		Mode:      s.cfg.Server.Mode,
		Phase:     phase,
		TxArmed:   s.txArmed,
		Nodes:     len(s.store.ListNodes()),
		SIMs:      len(s.store.ListSIMs()),
		Sessions:  len(s.store.ListSessions()),
		Peers:     len(s.store.ListPeers()),
		Alerts:    active,
		Critical:  crit,
		UpNodes:   up,
		BytesUp:   bytesUp,
		BytesDn:   bytesDn,
		UptimeSec: int64(time.Since(s.started).Seconds()),
	}
}
