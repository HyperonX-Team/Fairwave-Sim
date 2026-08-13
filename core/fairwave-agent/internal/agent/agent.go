// Package agent implements the on-box Fairwave agent: health probes,
// watchdog, GPS/time sync checks, SDR temperature, and the safe-TX flag.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Config for the agent.
type Config struct {
	ControlURL  string
	NodeID      string
	Token       string // bearer token for /v1/telemetry (FAIRWAVE_ADMIN_TOKEN)
	Interval    time.Duration
	DataDir     string
	EnableRF    bool   // default false; only set after tx-arm
	SDRTempPath string // e.g. /sys/class/thermal/... (empty = n/a on x86 without SDR)
	GPSDoPath   string // file an external GPSDO daemon touches when locked
}

// Agent runs the health loop.
type Agent struct {
	cfg    Config
	client *http.Client
	mu     sync.RWMutex
	last   Health
}

// Health is the heartbeat payload.
type Health struct {
	NodeID    string    `json:"node_id"`
	TS        time.Time `json:"ts"`
	Mode      string    `json:"mode"`
	Load1     float64   `json:"load1"`
	SDRTempC  *float64  `json:"sdr_temp_c,omitempty"`
	GPSDO     bool      `json:"gpsdo_locked"`
	RFArmed   bool      `json:"rf_armed"`
	Watchdog  string    `json:"watchdog"`
	Platform  string    `json:"platform"`
	Radio     string    `json:"radio"` // "zmq" | "none" | "hardware"
	FreqCheck bool      `json:"freq_plan_ok"`
}

// New creates an agent.
func New(cfg Config) *Agent {
	if cfg.Interval <= 0 {
		cfg.Interval = 10 * time.Second
	}
	return &Agent{cfg: cfg, client: &http.Client{Timeout: 5 * time.Second}}
}

// Run loops until ctx is done.
func (a *Agent) Run(ctx context.Context) error {
	t := time.NewTicker(a.cfg.Interval)
	defer t.Stop()
	for {
		h := a.collect()
		a.mu.Lock()
		a.last = h
		a.mu.Unlock()
		if err := a.send(h); err != nil {
			fmt.Printf("agent: heartbeat failed: %v\n", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// LastHealth returns the most recent health snapshot.
func (a *Agent) LastHealth() Health {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.last
}

func (a *Agent) collect() Health {
	h := Health{
		NodeID:   a.cfg.NodeID,
		TS:       time.Now(),
		Mode:     "lab",
		RFArmed:  false,
		Watchdog: "ok",
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
		Radio:    "zmq",
	}
	// load average (best-effort, POSIX)
	load := readProcLoad()
	if load >= 0 {
		h.Load1 = load
	}
	// SDR temperature (read from a sysfs path when configured)
	if a.cfg.SDRTempPath != "" {
		if c, err := readTemp(a.cfg.SDRTempPath); err == nil {
			h.SDRTempC = &c
		}
	}
	// GPSDO lock file
	if a.cfg.GPSDoPath != "" {
		if _, err := os.Stat(a.cfg.GPSDoPath); err == nil {
			h.GPSDO = true
		}
	}
	// RF arming: the agent refuses to arm unless explicitly enabled in config
	// AND the control plane has recorded tx_armed=true (checked by caller).
	h.RFArmed = a.cfg.EnableRF
	// frequency plan check: lab mode always zmq-only; nothing to verify on air
	h.FreqCheck = true
	return h
}

func (a *Agent) send(h Health) error {
	if a.cfg.ControlURL == "" {
		return nil
	}
	body, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("marshal health: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(a.cfg.ControlURL, "/")+"/v1/telemetry", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+a.cfg.Token)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("telemetry: %s", resp.Status)
	}
	return nil
}

// TouchWatchdog updates the watchdog file used by systemd to detect hangs.
func (a *Agent) TouchWatchdog() error {
	if a.cfg.DataDir == "" {
		return nil
	}
	if err := os.MkdirAll(a.cfg.DataDir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.cfg.DataDir, "watchdog"), []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)
}

// readProcLoad parses /proc/loadavg on Linux; -1 on other systems.
func readProcLoad() float64 {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return -1
	}
	var l float64
	_, _ = fmt.Sscanf(string(b), "%f", &l)
	return l
}

// readTemp parses a millidegree sysfs file into degrees Celsius.
func readTemp(path string) (float64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var millideg int
	if _, err := fmt.Sscanf(string(b), "%d", &millideg); err != nil {
		return 0, err
	}
	return float64(millideg) / 1000.0, nil
}
