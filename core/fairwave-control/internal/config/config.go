// Package config loads and validates Fairwave configuration from YAML files
// plus FAIRWAVE_* environment overrides. Secrets are never read from config
// files; they come from env vars or the vault only.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaJSON []byte

// ControlConfig is the top-level fairwave-control.yaml structure.
type ControlConfig struct {
	Server struct {
		Listen    string        `yaml:"listen" json:"listen"`
		DataDir   string        `yaml:"data_dir" json:"data_dir"`
		Mode      string        `yaml:"mode" json:"mode"` // lab | rf
		Country   string        `yaml:"country" json:"country"`
		LogLevel  string        `yaml:"log_level" json:"log_level"`
		LogFormat string        `yaml:"log_format" json:"log_format"` // json | console
		Shutdown  time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"`
	} `yaml:"server" json:"server"`

	PLMN struct {
		MCC string `yaml:"mcc" json:"mcc"`
		MNC string `yaml:"mnc" json:"mnc"`
	} `yaml:"plmn" json:"plmn"`

	TAC  int      `yaml:"tac" json:"tac"`
	APNs []string `yaml:"apns" json:"apns"`

	Spectrum struct {
		OverlayPath string `yaml:"overlay_path" json:"overlay_path"`
	} `yaml:"spectrum" json:"spectrum"`

	Auth struct {
		BootstrapTokenTTL time.Duration `yaml:"bootstrap_token_ttl" json:"bootstrap_token_ttl"`
		AdminTokenEnv     string        `yaml:"admin_token_env" json:"admin_token_env"`
	} `yaml:"auth" json:"auth"`

	Southbound struct {
		Open5GSConfigDir string `yaml:"open5gs_config_dir" json:"open5gs_config_dir"`
		SrsranConfigDir  string `yaml:"srsran_config_dir" json:"srsran_config_dir"`
		Driver           string `yaml:"driver" json:"driver"` // docker | none
	} `yaml:"southbound" json:"southbound"`

	Peering struct {
		Enabled    bool     `yaml:"enabled" json:"enabled"`
		MDNS       bool     `yaml:"mdns" json:"mdns"`
		Rendezvous string   `yaml:"rendezvous" json:"rendezvous"` // host:port of optional rendezvous
		WGIface    string   `yaml:"wg_iface" json:"wg_iface"`
		WGPort     int      `yaml:"wg_port" json:"wg_port"`
		AllowedIPs []string `yaml:"allowed_ips" json:"allowed_ips"`
	} `yaml:"peering" json:"peering"`

	Policy struct {
		LocalBreakout bool     `yaml:"local_breakout" json:"local_breakout"`
		HubPeer       string   `yaml:"hub_peer" json:"hub_peer"`
		MaxUEs        int      `yaml:"max_ues" json:"max_ues"`
		APNs          []string `yaml:"apns" json:"apns"`
		QoSDLMbps     int      `yaml:"qos_dl_mbps" json:"qos_dl_mbps"`
		QoSULMbps     int      `yaml:"qos_ul_mbps" json:"qos_ul_mbps"`
	} `yaml:"policy" json:"policy"`

	Telemetry struct {
		Metrics      bool   `yaml:"metrics" json:"metrics"`
		OTLPEndpoint string `yaml:"otlp_endpoint" json:"otlp_endpoint"`
	} `yaml:"telemetry" json:"telemetry"`

	Version string `yaml:"-" json:"-"`
}

// Default returns a safe, RF-disabled configuration.
func Default() *ControlConfig {
	c := &ControlConfig{}
	c.Server.Listen = ":8080"
	c.Server.DataDir = "./data"
	c.Server.Mode = "lab"
	c.Server.Country = "LAB"
	c.Server.LogLevel = "info"
	c.Server.LogFormat = "json"
	c.Server.Shutdown = 10 * time.Second
	c.PLMN.MCC = "999"
	c.PLMN.MNC = "99"
	c.TAC = 7
	c.APNs = []string{"internet", "ims"}
	c.Auth.BootstrapTokenTTL = 5 * time.Minute
	c.Auth.AdminTokenEnv = "FAIRWAVE_ADMIN_TOKEN"
	c.Southbound.Open5GSConfigDir = "./data/open5gs"
	c.Southbound.SrsranConfigDir = "./data/srsran"
	c.Southbound.Driver = "none"
	c.Peering.Enabled = true
	c.Peering.MDNS = true
	c.Peering.WGIface = "fw-mesh"
	c.Peering.WGPort = 51820
	c.Peering.AllowedIPs = []string{"10.99.0.0/16"}
	c.Policy.LocalBreakout = true
	c.Policy.MaxUEs = 128
	c.Policy.APNs = []string{"internet", "ims"}
	c.Policy.QoSDLMbps = 50
	c.Policy.QoSULMbps = 25
	c.Telemetry.Metrics = true
	return c
}

// Load reads path, applies env overrides, and validates against the embedded
// JSON Schema. Env override naming: FAIRWAVE_SERVER_LISTEN, FAIRWAVE_TAC, ...
func Load(path string) (*ControlConfig, error) {
	c := Default()
	data := []byte{}
	if path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, c); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}
	}
	applyEnv(c)
	if err := validate(c, path); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func applyEnv(c *ControlConfig) {
	// Map env keys to fields. Deliberately explicit (no reflection) so the
	// overrides are reviewable at a glance.
	mapping := map[string]func(string){
		"FAIRWAVE_SERVER_LISTEN":     func(v string) { c.Server.Listen = v },
		"FAIRWAVE_SERVER_DATADIR":    func(v string) { c.Server.DataDir = v },
		"FAIRWAVE_SERVER_MODE":       func(v string) { c.Server.Mode = v },
		"FAIRWAVE_SERVER_COUNTRY":    func(v string) { c.Server.Country = v },
		"FAIRWAVE_SERVER_LOGLEVEL":   func(v string) { c.Server.LogLevel = v },
		"FAIRWAVE_PLMN_MCC":          func(v string) { c.PLMN.MCC = v },
		"FAIRWAVE_PLMN_MNC":          func(v string) { c.PLMN.MNC = v },
		"FAIRWAVE_TAC":               func(v string) { c.TAC, _ = strconv.Atoi(v) },
		"FAIRWAVE_MODE":              func(v string) { c.Server.Mode = v },
		"FAIRWAVE_COUNTRY":           func(v string) { c.Server.Country = v },
		"FAIRWAVE_ADMIN_TOKEN":       func(v string) { _ = v }, // consumed by api auth, not config
		"FAIRWAVE_SOUTHBOUND_DRIVER": func(v string) { c.Southbound.Driver = v },
		"FAIRWAVE_PEERING_DISABLED": func(v string) {
			if b, err := strconv.ParseBool(v); err == nil {
				c.Peering.Enabled = !b
			}
		},
		"FAIRWAVE_POLICY_LOCAL_BREAKOUT": func(v string) {
			if b, err := strconv.ParseBool(v); err == nil {
				c.Policy.LocalBreakout = b
			}
		},
	}
	for k, fn := range mapping {
		if v, ok := os.LookupEnv(k); ok {
			fn(v)
		}
	}
}

func validate(c *ControlConfig, path string) error {
	doc := map[string]interface{}{}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return err
		}
	} else {
		// No file: build a minimal doc from defaults so the schema runs.
		out, _ := yaml.Marshal(c)
		_ = yaml.Unmarshal(out, &doc)
	}
	res, err := gojsonschema.Validate(
		gojsonschema.NewBytesLoader(schemaJSON),
		gojsonschema.NewGoLoader(doc),
	)
	if err != nil {
		return fmt.Errorf("schema load: %w", err)
	}
	if !res.Valid() {
		var sb strings.Builder
		for _, e := range res.Errors() {
			sb.WriteString("- " + e.String() + "\n")
		}
		return fmt.Errorf("config invalid:\n%s", sb.String())
	}
	return nil
}

// Validate performs semantic checks the schema cannot express.
func (c *ControlConfig) Validate() error {
	if c.Server.Mode != "lab" && c.Server.Mode != "rf" {
		return fmt.Errorf("server.mode must be lab or rf (got %q)", c.Server.Mode)
	}
	if c.PLMN.MCC == "" || len(c.PLMN.MCC) != 3 {
		return fmt.Errorf("plmn.mcc must be 3 digits")
	}
	if c.PLMN.MNC == "" || (len(c.PLMN.MNC) != 2 && len(c.PLMN.MNC) != 3) {
		return fmt.Errorf("plmn.mnc must be 2 or 3 digits")
	}
	if c.Server.Country == "" {
		return fmt.Errorf("server.country required (use LAB for lab mode)")
	}
	if c.Server.Mode == "rf" && c.Server.Country == "LAB" {
		return fmt.Errorf("rf mode requires a real country; LAB is virtual-only")
	}
	if c.TAC <= 0 || c.TAC > 65535 {
		return fmt.Errorf("tac out of range: %d", c.TAC)
	}
	return nil
}
