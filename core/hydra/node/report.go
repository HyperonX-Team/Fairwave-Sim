package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ReportLoop periodically sends this node's measured thread health to a
// Fairwave control plane until ctx is cancelled. It is how a running data
// plane makes live link health visible to the operator instead of leaving
// only the declared values in the store.
func ReportLoop(ctx context.Context, n *Node, base, token string, interval time.Duration, logf func(string, ...any)) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ReportOnce(ctx, client, n, base, token); err != nil {
				logf("hydra report: %v", err)
			}
		}
	}
}

// ReportOnce posts the node's current thread health to the control plane.
func ReportOnce(ctx context.Context, client *http.Client, n *Node, base, token string) error {
	st := n.Stats()
	if len(st.Health) == 0 {
		return nil
	}
	base = strings.TrimRight(base, "/")
	var firstErr error
	for _, h := range st.Health {
		loss := 0.0
		if !h.Up {
			loss = 100
		}
		body, err := json.Marshal(map[string]any{
			"mbps":     h.Mbps,
			"rtt_ms":   h.RTTms,
			"loss_pct": loss,
			"up":       h.Up,
		})
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		url := fmt.Sprintf("%s/v1/hydra/threads/%s/health", base, h.ID)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 400 && firstErr == nil {
			firstErr = fmt.Errorf("thread %s: http %d", h.ID, resp.StatusCode)
		}
	}
	return firstErr
}
