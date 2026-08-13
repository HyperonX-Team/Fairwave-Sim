package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultValid(t *testing.T) {
	c := Default()
	if err := c.Validate(); err != nil {
		t.Fatalf("default config must validate: %v", err)
	}
}

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "control.yaml")
	content := `
server:
  listen: ":9090"
  data_dir: "./data"
  mode: lab
  country: LAB
plmn:
  mcc: "999"
  mnc: "99"
tac: 7
apns: [internet, ims]
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Server.Listen != ":9090" {
		t.Fatalf("listen: %s", c.Server.Listen)
	}
	if c.TAC != 7 {
		t.Fatalf("tac: %d", c.TAC)
	}
}

func TestLoadRejectsInvalidMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	content := "server:\n  mode: rf\n  country: LAB\nplmn:\n  mcc: \"999\"\n  mnc: \"99\"\ntac: 7\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("rf+LAB must be rejected")
	}
}

func TestLoadRejectsBadSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema-bad.yaml")
	content := "server:\n  mode: lab\n  country: LAB\nplmn:\n  mcc: \"12\"\n  mnc: \"99\"\ntac: 7\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("2-digit MCC must fail schema validation")
	}
}

func TestEnvOverrides(t *testing.T) {
	os.Setenv("FAIRWAVE_SERVER_LISTEN", ":7777")
	os.Setenv("FAIRWAVE_TAC", "42")
	os.Setenv("FAIRWAVE_PLMN_MCC", "310")
	defer func() {
		os.Unsetenv("FAIRWAVE_SERVER_LISTEN")
		os.Unsetenv("FAIRWAVE_TAC")
		os.Unsetenv("FAIRWAVE_PLMN_MCC")
	}()
	c, err := Load("")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Server.Listen != ":7777" {
		t.Fatalf("listen: %s", c.Server.Listen)
	}
	if c.TAC != 42 {
		t.Fatalf("tac: %d", c.TAC)
	}
	if c.PLMN.MCC != "310" {
		t.Fatalf("mcc: %s", c.PLMN.MCC)
	}
}

func TestHSSDriverValidation(t *testing.T) {
	c := Default()
	c.HSS.Driver = "mongosh"
	c.HSS.Container = "mongo"
	if err := c.Validate(); err != nil {
		t.Fatalf("mongosh driver should validate: %v", err)
	}
	c.HSS.Container = ""
	if err := c.Validate(); err == nil {
		t.Fatal("missing container with a real driver must fail")
	}
	c.HSS.Container = "mongo"
	c.HSS.Driver = "sanity-check"
	if err := c.Validate(); err == nil {
		t.Fatal("unknown driver must fail")
	}
	c.HSS.Driver = "none"
	c.HSS.Container = ""
	if err := c.Validate(); err != nil {
		t.Fatalf("none driver must validate without container: %v", err)
	}
}

func TestUPFCollectorValidation(t *testing.T) {
	c := Default()
	c.Collector.UPF.Enabled = true
	c.Collector.UPF.Iface = ""
	if err := c.Validate(); err == nil {
		t.Fatal("upf enabled without iface must fail")
	}
	c.Collector.UPF.Iface = "fwnet"
	if err := c.Validate(); err != nil {
		t.Fatalf("upf with iface must validate: %v", err)
	}
}

func TestEnvOverridesUPF(t *testing.T) {
	cfg := Default()
	t.Setenv("FAIRWAVE_COLLECTOR_UPF_ENABLED", "true")
	t.Setenv("FAIRWAVE_COLLECTOR_UPF_IFACE", "eth1")
	applyEnv(cfg)
	if !cfg.Collector.UPF.Enabled || cfg.Collector.UPF.Iface != "eth1" {
		t.Fatalf("upf env overrides not applied: %+v", cfg.Collector.UPF)
	}
}

func TestFree5GCCoreValidation(t *testing.T) {
	c := Default()
	c.Core = "free5gc"
	if err := c.Validate(); err != nil {
		t.Fatalf("free5gc core with defaults must validate: %v", err)
	}
	// free5gc collector needs the AMF OAM URL.
	c.Collector.Enabled = true
	c.Free5GC.AMFOAMURL = ""
	if err := c.Validate(); err == nil {
		t.Fatal("free5gc collector without amf_oam_url must fail")
	}
	c.Free5GC.AMFOAMURL = "http://amf:8000"
	if err := c.Validate(); err != nil {
		t.Fatalf("free5gc collector with amf_oam_url must validate: %v", err)
	}
	// free5gc core does not require the open5gs MME URL.
	c.Collector.MMEURL = ""
	if err := c.Validate(); err != nil {
		t.Fatalf("free5gc core must not require mme_url: %v", err)
	}
	// unknown core must fail.
	c.Core = "nextepc"
	if err := c.Validate(); err == nil {
		t.Fatal("unknown core must fail")
	}
}

func TestFree5GCHSSDriverValidation(t *testing.T) {
	c := Default()
	c.HSS.Driver = "free5gc"
	c.HSS.Container = "mongo"
	if err := c.Validate(); err != nil {
		t.Fatalf("free5gc hss driver must validate: %v", err)
	}
}

func TestEnvOverridesFree5GC(t *testing.T) {
	cfg := Default()
	t.Setenv("FAIRWAVE_CORE", "free5gc")
	t.Setenv("FAIRWAVE_FREE5GC_AMF_OAM_URL", "http://amf:8000")
	t.Setenv("FAIRWAVE_FREE5GC_CDR_DIR", "/var/fairwave/chf-cdr")
	applyEnv(cfg)
	if cfg.Core != "free5gc" || cfg.Free5GC.AMFOAMURL != "http://amf:8000" || cfg.Free5GC.CDRDir != "/var/fairwave/chf-cdr" {
		t.Fatalf("free5gc env overrides not applied: core=%s amf=%s cdr=%s", cfg.Core, cfg.Free5GC.AMFOAMURL, cfg.Free5GC.CDRDir)
	}
}

func TestFree5GCCDRValidation(t *testing.T) {
	c := Default()
	c.Core = "free5gc"
	c.Collector.Enabled = true
	c.Free5GC.CDRDir = "/var/fairwave/chf-cdr"
	if err := c.Validate(); err != nil {
		t.Fatalf("free5gc with cdr_dir must validate: %v", err)
	}
	// CDR meter runs in the collector: it must be enabled.
	c.Collector.Enabled = false
	if err := c.Validate(); err == nil {
		t.Fatal("cdr_dir without collector.enabled must fail")
	}
	c.Collector.Enabled = true
	// cdr_dir only makes sense with the free5GC core.
	c.Core = "open5gs"
	if err := c.Validate(); err == nil {
		t.Fatal("cdr_dir on an open5gs core must fail")
	}
}

func TestEnvOverridesHSS(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FAIRWAVE_HSS_DRIVER", "dbctl")
	t.Setenv("FAIRWAVE_HSS_CONTAINER", "open5gs")
	c, err := Load(filepath.Join(dir, "nope.yaml"))
	if err == nil {
		t.Fatal("expected load error for missing file")
	}
	_ = c
	// Load from a valid file with env overrides.
	cfg := Default()
	applyEnv(cfg)
	if cfg.HSS.Driver != "dbctl" || cfg.HSS.Container != "open5gs" {
		t.Fatalf("env overrides not applied: %+v", cfg.HSS)
	}
}
