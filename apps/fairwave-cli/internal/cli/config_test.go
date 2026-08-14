package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalValidConfig mirrors the structure of deploy/config/fairwave-control.yaml
// and satisfies both the embedded JSON schema (server/plmn/tac required) and the
// semantic rules in config.Validate.
const minimalValidConfig = `server:
  listen: ":8080"
  data_dir: /tmp/fairwave-test-data
  mode: lab
  country: LAB
  log_level: info
  log_format: console
plmn:
  mcc: "999"
  mnc: "99"
tac: 7
apns: [internet, ims]
auth:
  bootstrap_token_ttl: 5m
  admin_token_env: FAIRWAVE_ADMIN_TOKEN
southbound:
  driver: none
hss:
  driver: none
peering:
  enabled: true
  mdns: false
policy:
  local_breakout: true
  max_ues: 128
  apns: [internet, ims]
telemetry:
  metrics: true
`

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fairwave-control.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	return path
}

func runConfigValidate(t *testing.T, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	root := newTestRoot(t)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"config", "validate"}, args...))
	_, err := root.ExecuteC()
	return &out, err
}

func decodeValidResult(t *testing.T, out *bytes.Buffer) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout: %q", err, out.String())
	}
	return got
}

func TestConfigValidateValid(t *testing.T) {
	path := writeConfig(t, minimalValidConfig)
	out, err := runConfigValidate(t, path)
	if err != nil {
		t.Fatalf("config validate %s: unexpected error: %v", path, err)
	}
	got := decodeValidResult(t, out)
	if got["valid"] != true {
		t.Fatalf("valid = %v, want true (stdout: %s)", got["valid"], out.String())
	}
	if got["path"] != path {
		t.Fatalf("path = %v, want %q", got["path"], path)
	}
	if _, present := got["error"]; present {
		t.Fatalf("successful validation must not carry an error field: %s", out.String())
	}
}

func TestConfigValidateInvalidMode(t *testing.T) {
	path := writeConfig(t, strings.Replace(minimalValidConfig, "mode: lab", "mode: invalid", 1))
	out, err := runConfigValidate(t, path)
	if err == nil {
		t.Fatal("expected a validation error for mode: invalid")
	}
	got := decodeValidResult(t, out)
	if got["valid"] != false {
		t.Fatalf("valid = %v, want false (stdout: %s)", got["valid"], out.String())
	}
	if got["path"] != path {
		t.Fatalf("path = %v, want %q", got["path"], path)
	}
	if msg, ok := got["error"].(string); !ok || msg == "" {
		t.Fatalf("error field missing or empty: %s", out.String())
	}
}

func TestConfigValidateMissingFile(t *testing.T) {
	out, err := runConfigValidate(t, "/nonexistent/fairwave-control.yaml")
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	got := decodeValidResult(t, out)
	if got["valid"] != false {
		t.Fatalf("valid = %v, want false (stdout: %s)", got["valid"], out.String())
	}
	if got["path"] != "/nonexistent/fairwave-control.yaml" {
		t.Fatalf("path = %v, want /nonexistent/fairwave-control.yaml", got["path"])
	}
	if msg, ok := got["error"].(string); !ok || msg == "" {
		t.Fatalf("error field missing or empty: %s", out.String())
	}
}

func TestConfigValidateOmittedPathIsUsageError(t *testing.T) {
	_, err := runConfigValidate(t)
	if err == nil {
		t.Fatal("expected an error when <path> is omitted")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("errors.Is(%q, ErrUsage) = false; omitted path is a usage error (exit 2)", err)
	}
}
