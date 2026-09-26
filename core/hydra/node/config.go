package node

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// FileConfig is the YAML/JSON shape of a Hydra node configuration, as read
// by the fairwave-hydra daemon.
type FileConfig struct {
	// NodeID names this node in logs and health reports.
	NodeID string `yaml:"node_id" json:"node_id"`
	// Weave is the weave this node serves.
	Weave string `yaml:"weave" json:"weave"`
	// Anchor labels the egress this node forwards to.
	Anchor string `yaml:"anchor" json:"anchor"`
	// Ingress is the local UDP address receiving payloads to stripe.
	Ingress string `yaml:"ingress" json:"ingress"`
	// Egress is the remote UDP address receiving reassembled payloads.
	Egress string `yaml:"egress" json:"egress"`
	// ProbeInterval is a duration string (e.g. "2s"); empty disables probes.
	ProbeInterval string `yaml:"probe_interval" json:"probe_interval"`
	// ProbeTimeout is a duration string; empty defaults to 3x the interval.
	ProbeTimeout string `yaml:"probe_timeout" json:"probe_timeout"`
	// ProbeFailures is the consecutive-miss count that marks a thread down.
	ProbeFailures int `yaml:"probe_failures" json:"probe_failures"`
	// QueueFrames caps in-flight frames per thread before tail-drop.
	QueueFrames int `yaml:"queue_frames" json:"queue_frames"`
	// Report is the control-plane base URL for live health reports.
	Report string `yaml:"report" json:"report"`
	// Token authenticates the health reports.
	Token string `yaml:"token" json:"token"`
	// ReportInterval is a duration string; empty defaults to 5s.
	ReportInterval string `yaml:"report_interval" json:"report_interval"`
	// Threads are this node's independent SIM/modem/path endpoints.
	Threads []FileThread `yaml:"threads" json:"threads"`
}

// FileThread is one thread entry in a node configuration file.
type FileThread struct {
	ID     string  `yaml:"id" json:"id"`
	Local  string  `yaml:"local" json:"local"`
	Remote string  `yaml:"remote" json:"remote"`
	Mbps   float64 `yaml:"mbps" json:"mbps"`
}

// LoadFile reads a node configuration from a YAML or JSON file.
func LoadFile(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("node: read config %s: %w", path, err)
	}
	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("node: parse config %s: %w", path, err)
	}
	return &fc, nil
}

// Config converts the file form into a runtime Config, parsing duration
// strings and validating the thread list.
func (fc *FileConfig) Config() (Config, error) {
	if len(fc.Threads) == 0 {
		return Config{}, fmt.Errorf("node: config has no threads")
	}
	cfg := Config{
		ID:            fc.NodeID,
		WeaveID:       fc.Weave,
		Anchor:        fc.Anchor,
		Ingress:       fc.Ingress,
		Egress:        fc.Egress,
		ProbeFailures: fc.ProbeFailures,
		QueueFrames:   fc.QueueFrames,
	}
	var err error
	if fc.ProbeInterval != "" {
		if cfg.ProbeInterval, err = time.ParseDuration(fc.ProbeInterval); err != nil {
			return Config{}, fmt.Errorf("node: probe_interval %q: %w", fc.ProbeInterval, err)
		}
	}
	if fc.ProbeTimeout != "" {
		if cfg.ProbeTimeout, err = time.ParseDuration(fc.ProbeTimeout); err != nil {
			return Config{}, fmt.Errorf("node: probe_timeout %q: %w", fc.ProbeTimeout, err)
		}
	}
	for i, t := range fc.Threads {
		if t.ID == "" {
			return Config{}, fmt.Errorf("node: thread %d has no id", i)
		}
		cfg.Threads = append(cfg.Threads, ThreadConfig(t))
	}
	return cfg, nil
}

// ResolveReportInterval returns the health-report cadence (default 5s).
func (fc *FileConfig) ResolveReportInterval() time.Duration {
	if fc.ReportInterval == "" {
		return 5 * time.Second
	}
	d, err := time.ParseDuration(fc.ReportInterval)
	if err != nil || d <= 0 {
		return 5 * time.Second
	}
	return d
}
