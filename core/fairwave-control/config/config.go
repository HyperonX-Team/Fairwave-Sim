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

	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/hsswrite"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaJSON []byte

// ControlConfig is the top-level fairwave-control.yaml structure.
type ControlConfig struct {
	// Core selects the mobile-core backend the control plane supervises.
	// open5gs keeps the existing 4G/5G lab core; free5gc switches the HSS
	// write-back and session collector to free5GC (5G SA).
	Core string `yaml:"core" json:"core"` // open5gs | free5gc

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

	HSS struct {
		Driver    string `yaml:"driver" json:"driver"`       // mongosh | dbctl | none
		Container string `yaml:"container" json:"container"` // container to exec into
	} `yaml:"hss" json:"hss"`

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
		Metrics      bool          `yaml:"metrics" json:"metrics"`
		OTLPEndpoint string        `yaml:"otlp_endpoint" json:"otlp_endpoint"`
		StaleAfter   time.Duration `yaml:"stale_after" json:"stale_after"` // node marked down after this silence
	} `yaml:"telemetry" json:"telemetry"`

	Collector struct {
		Enabled  bool          `yaml:"enabled" json:"enabled"`
		Interval time.Duration `yaml:"interval" json:"interval"`
		MMEURL   string        `yaml:"mme_url" json:"mme_url"`
		SMFURL   string        `yaml:"smf_url" json:"smf_url"`
		UPF      struct {
			Enabled bool   `yaml:"enabled" json:"enabled"` // per-UE GTP-U accounting tap
			Iface   string `yaml:"iface" json:"iface"`     // interface carrying GTP-U (S1-U/S5-U)
		} `yaml:"upf" json:"upf"`
	} `yaml:"collector" json:"collector"`

	// Free5GC holds the free5GC-specific knobs, active when core is
	// "free5gc".
	Free5GC struct {
		AMFOAMURL string `yaml:"amf_oam_url" json:"amf_oam_url"` // AMF OAM HTTP base, e.g. http://amf:8000
		CDRDir    string `yaml:"cdr_dir" json:"cdr_dir"`         // CHF CDR dir (shared volume); enables core-metered usage
	} `yaml:"free5gc" json:"free5gc"`

	ESIM struct {
		Enabled      bool          `yaml:"enabled" json:"enabled"`
		RegistryPath string        `yaml:"registry_path" json:"registry_path"`
		SMDPAddress  string        `yaml:"smdp_address" json:"smdp_address"` // host[:port] embedded in activation codes
		SMDPID       string        `yaml:"smdp_id" json:"smdp_id"`
		SingleUse    bool          `yaml:"single_use" json:"single_use"` // codes die after one download
		CodeTTL      time.Duration `yaml:"code_ttl" json:"code_ttl"`     // codes expire after this long
	} `yaml:"esim" json:"esim"`

	Alerts struct {
		Enabled           bool     `yaml:"enabled" json:"enabled"`
		Webhooks          []string `yaml:"webhooks" json:"webhooks"`
		TempHighC         float64  `yaml:"temp_high_c" json:"temp_high_c"`
		SimExpiryWarnDays int      `yaml:"sim_expiry_warn_days" json:"sim_expiry_warn_days"`
		UesCapacityPct    int      `yaml:"ues_capacity_pct" json:"ues_capacity_pct"`
	} `yaml:"alerts" json:"alerts"`

	FairUse struct {
		Enabled       bool          `yaml:"enabled" json:"enabled"`
		UsageInterval time.Duration `yaml:"usage_interval" json:"usage_interval"`
	} `yaml:"fairuse" json:"fairuse"`

	Version string `yaml:"-" json:"-"`
}

// Default returns a safe, RF-disabled configuration.
func Default() *ControlConfig {
	c := &ControlConfig{}
	c.Core = "open5gs"
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
	c.HSS.Driver = "none"
	c.HSS.Container = "mongo"
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
	c.Telemetry.StaleAfter = 90 * time.Second
	c.Collector.Enabled = false
	c.Collector.Interval = 15 * time.Second
	c.Collector.MMEURL = "http://127.0.0.2:9090" // Open5GS MME infoAPI (lab compose pins the MME to .2)
	c.Collector.SMFURL = "http://127.0.0.4:9090" // optional: SMF infoAPI for IP enrichment
	c.Collector.UPF.Enabled = false              // requires CAP_NET_RAW on the tap interface
	c.Collector.UPF.Iface = ""
	c.Free5GC.AMFOAMURL = "http://amf:8000" // free5GC AMF OAM (SBI) port
	c.Free5GC.CDRDir = ""                   // CHF CDR dir; unset = no core-metered usage
	c.ESIM.Enabled = true
	c.ESIM.RegistryPath = "" // defaults to <data_dir>/esim/registry.json
	c.ESIM.SMDPAddress = "fairwave.local:8443"
	c.ESIM.SMDPID = "fairwave-esim"
	c.ESIM.SingleUse = true
	c.ESIM.CodeTTL = 7 * 24 * time.Hour
	c.Alerts.Enabled = true
	c.Alerts.TempHighC = 85
	c.Alerts.SimExpiryWarnDays = 14
	c.Alerts.UesCapacityPct = 90
	c.FairUse.Enabled = false // auto-suspend is opt-in: it is a heavy hammer
	c.FairUse.UsageInterval = 60 * time.Second
	return c
}

// Load reads path, applies env overrides, and validates against the embedded
// JSON Schema. Env override naming: FAIRWAVE_SERVER_LISTEN, FAIRWAVE_TAC, ...
func Load(path string) (*ControlConfig, error) {
	c := Default()
	if path != "" {
		data, err := os.ReadFile(path)
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
		"FAIRWAVE_HSS_DRIVER":        func(v string) { c.HSS.Driver = v },
		"FAIRWAVE_HSS_CONTAINER":     func(v string) { c.HSS.Container = v },
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
		"FAIRWAVE_COLLECTOR_ENABLED": func(v string) {
			if b, err := strconv.ParseBool(v); err == nil {
				c.Collector.Enabled = b
			}
		},
		"FAIRWAVE_COLLECTOR_MME_URL": func(v string) { c.Collector.MMEURL = v },
		"FAIRWAVE_COLLECTOR_SMF_URL": func(v string) { c.Collector.SMFURL = v },
		"FAIRWAVE_COLLECTOR_UPF_ENABLED": func(v string) {
			if b, err := strconv.ParseBool(v); err == nil {
				c.Collector.UPF.Enabled = b
			}
		},
		"FAIRWAVE_COLLECTOR_UPF_IFACE": func(v string) { c.Collector.UPF.Iface = v },
		"FAIRWAVE_CORE":                func(v string) { c.Core = v },
		"FAIRWAVE_FREE5GC_AMF_OAM_URL": func(v string) { c.Free5GC.AMFOAMURL = v },
		"FAIRWAVE_FREE5GC_CDR_DIR":     func(v string) { c.Free5GC.CDRDir = v },
		"FAIRWAVE_ESIM_ENABLED": func(v string) {
			if b, err := strconv.ParseBool(v); err == nil {
				c.ESIM.Enabled = b
			}
		},
		"FAIRWAVE_ESIM_SMDP_ADDRESS": func(v string) { c.ESIM.SMDPAddress = v },
		"FAIRWAVE_ESIM_SINGLE_USE": func(v string) {
			if b, err := strconv.ParseBool(v); err == nil {
				c.ESIM.SingleUse = b
			}
		},
		"FAIRWAVE_ALERTS_ENABLED": func(v string) {
			if b, err := strconv.ParseBool(v); err == nil {
				c.Alerts.Enabled = b
			}
		},
		"FAIRWAVE_FAIRUSE_ENABLED": func(v string) {
			if b, err := strconv.ParseBool(v); err == nil {
				c.FairUse.Enabled = b
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
	switch c.Core {
	case "open5gs", "free5gc":
	default:
		return fmt.Errorf("core must be open5gs or free5gc (got %q)", c.Core)
	}
	switch c.HSS.Driver {
	case hsswrite.DriverNone, hsswrite.DriverMongosh, hsswrite.DriverDBCTL, hsswrite.DriverFree5GC:
	default:
		return fmt.Errorf("hss.driver must be mongosh, dbctl, free5gc or none (got %q)", c.HSS.Driver)
	}
	if c.HSS.Driver != hsswrite.DriverNone && c.HSS.Container == "" {
		return fmt.Errorf("hss.container required when hss.driver is %q", c.HSS.Driver)
	}
	if c.Core == "free5gc" && c.Collector.Enabled && c.Free5GC.AMFOAMURL == "" {
		return fmt.Errorf("free5gc.amf_oam_url required when core is free5gc and collector.enabled is true")
	}
	if c.Free5GC.CDRDir != "" {
		if c.Core != "free5gc" {
			return fmt.Errorf("free5gc.cdr_dir requires core free5gc")
		}
		if !c.Collector.Enabled {
			return fmt.Errorf("free5gc.cdr_dir requires collector.enabled (the CDR meter runs in the collector)")
		}
	}
	if c.Collector.Enabled && c.Core != "free5gc" && c.Collector.MMEURL == "" {
		return fmt.Errorf("collector.mme_url required when collector.enabled is true (open5gs core)")
	}
	if c.Collector.UPF.Enabled && c.Collector.UPF.Iface == "" {
		return fmt.Errorf("collector.upf.iface required when collector.upf.enabled is true")
	}
	if c.ESIM.Enabled && c.ESIM.SMDPAddress == "" {
		return fmt.Errorf("esim.smdp_address required when esim.enabled is true")
	}
	if c.Alerts.UesCapacityPct <= 0 || c.Alerts.UesCapacityPct > 100 {
		return fmt.Errorf("alerts.ues_capacity_pct must be in (0,100]")
	}
	if c.FairUse.Enabled && c.FairUse.UsageInterval <= 0 {
		return fmt.Errorf("fairuse.usage_interval must be positive when fairuse.enabled")
	}
	return nil
}
