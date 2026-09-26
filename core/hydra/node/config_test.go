package node

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFileAndConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hydra.yaml")
	yamlDoc := `
node_id: edge-1
weave: w
anchor: hub
ingress: 127.0.0.1:7777
egress: 127.0.0.1:8888
probe_interval: 2s
probe_timeout: 6s
probe_failures: 4
queue_frames: 2048
report: http://127.0.0.1:8080
report_interval: 3s
threads:
  - id: t1
    local: 127.0.0.1:10001
    remote: 127.0.0.1:20001
    mbps: 300
  - id: t2
    local: 127.0.0.1:10002
    remote: 127.0.0.1:20002
    mbps: 200
`
	if err := os.WriteFile(path, []byte(yamlDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	fc, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := fc.Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ID != "edge-1" || cfg.WeaveID != "w" || cfg.Anchor != "hub" {
		t.Fatalf("identity = %+v", cfg)
	}
	if cfg.ProbeInterval != 2*time.Second || cfg.ProbeTimeout != 6*time.Second {
		t.Fatalf("probe durations = %v / %v", cfg.ProbeInterval, cfg.ProbeTimeout)
	}
	if cfg.ProbeFailures != 4 || cfg.QueueFrames != 2048 {
		t.Fatalf("tuning = %+v", cfg)
	}
	if len(cfg.Threads) != 2 || cfg.Threads[0].ID != "t1" || cfg.Threads[0].Mbps != 300 {
		t.Fatalf("threads = %+v", cfg.Threads)
	}
	if got := fc.ResolveReportInterval(); got != 3*time.Second {
		t.Fatalf("report interval = %v", got)
	}
}

func TestConfigRejectsNoThreads(t *testing.T) {
	fc := &FileConfig{}
	if _, err := fc.Config(); err == nil {
		t.Fatal("expected an error for a config with no threads")
	}
}

func TestConfigRejectsBadDuration(t *testing.T) {
	fc := &FileConfig{ProbeInterval: "soon", Threads: []FileThread{{ID: "t1"}}}
	if _, err := fc.Config(); err == nil {
		t.Fatal("expected an error for a malformed probe_interval")
	}
}

func TestConfigRejectsThreadWithoutID(t *testing.T) {
	fc := &FileConfig{Threads: []FileThread{{Mbps: 100}}}
	if _, err := fc.Config(); err == nil {
		t.Fatal("expected an error for a thread with no id")
	}
}

func TestResolveReportIntervalDefault(t *testing.T) {
	fc := &FileConfig{}
	if got := fc.ResolveReportInterval(); got != 5*time.Second {
		t.Fatalf("default report interval = %v", got)
	}
	fc.ReportInterval = "not-a-duration"
	if got := fc.ResolveReportInterval(); got != 5*time.Second {
		t.Fatalf("malformed report interval did not fall back: %v", got)
	}
}

func TestLoadFileMissing(t *testing.T) {
	if _, err := LoadFile(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected an error for a missing config")
	}
}
