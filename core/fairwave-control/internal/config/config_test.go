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
