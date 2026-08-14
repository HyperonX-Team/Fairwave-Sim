package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

const validCSV = "imsi,msisdn,profile,apn,status,expires_at\n" +
	"901700000000001,4917012345678,lab,internet,issued,2026-12-31T23:59:59Z\n" +
	"901700000000002,,prod,ims,suspended,\n"

const validJSON = `[
  {"imsi":"901700000000010","msisdn":"+4915550100","profile":"lab","apn":"internet","status":"issued"},
  {"imsi":"901700000000011","msisdn":"","profile":"prod","apn":"ims","status":"suspended","expires_at":"2027-01-01T00:00:00Z"}
]`

func TestParseSimImportCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "batch.csv")
	writeFile(t, path, validCSV)

	items, err := parseSimImport(path)
	if err != nil {
		t.Fatalf("parseSimImport(%q): %v", path, err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].IMSI != "901700000000001" || items[0].MSISDN != "4917012345678" ||
		items[0].Profile != "lab" || items[0].APN != "internet" || items[0].Status != "issued" {
		t.Errorf("row 0 parsed wrong: %+v", items[0])
	}
	if items[0].ExpiresAt.IsZero() {
		t.Error("row 0 expires_at not parsed")
	}
	// Missing optional columns must parse to empty values, never an error.
	if items[1].MSISDN != "" || items[1].Profile != "prod" || items[1].APN != "ims" {
		t.Errorf("row 1 parsed wrong: %+v", items[1])
	}
}

func TestParseSimImportJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "batch.json")
	writeFile(t, path, validJSON)

	items, err := parseSimImport(path)
	if err != nil {
		t.Fatalf("parseSimImport(%q): %v", path, err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].IMSI != "901700000000010" || items[0].APN != "internet" || items[0].Status != "issued" {
		t.Errorf("row 0 parsed wrong: %+v", items[0])
	}
	if items[1].ExpiresAt.IsZero() {
		t.Error("row 1 expires_at not parsed")
	}
}

func TestParseSimImportCSVHeaderOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.csv")
	writeFile(t, path, "imsi,msisdn,profile,apn,status\n")

	items, err := parseSimImport(path)
	if err != nil {
		t.Fatalf("header-only CSV: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
}

func TestParseSimImportRejectsBadPaths(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		path string
		want string // substring the error must contain
	}{
		{"short path", "x", "unsupported extension"},
		{"txt extension", filepath.Join(dir, "foo.txt"), "unsupported extension"},
		{"no extension", filepath.Join(dir, "foo"), "unsupported extension"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The extension check runs before the file is opened, but write a
			// real file anyway so a regression that reaches the parser would
			// surface rather than a silent "no such file".
			if tc.name != "short path" {
				writeFile(t, tc.path, validJSON)
			}
			items, err := parseSimImport(tc.path)
			if err == nil {
				t.Fatalf("parseSimImport(%q) = %d items, want error", tc.path, len(items))
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

// TestParseSimImportCaseInsensitiveExtension locks the lowercasing behavior:
// .CSV/.JSON must be accepted just like .csv/.json.
func TestParseSimImportCaseInsensitiveExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "batch.CSV")
	writeFile(t, path, validCSV)
	items, err := parseSimImport(path)
	if err != nil {
		t.Fatalf("parseSimImport(%q): %v", path, err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

// TestParseSimImportExtensionlessJSON guards against the historic
// misclassification: a file whose *name* lacks an extension must be rejected
// even if its content is valid JSON, and a .json file with JSON content must
// parse. Regression for the pre-T6 `ext == "json"` name-matching bug.
func TestParseSimImportExtensionlessJSON(t *testing.T) {
	dir := t.TempDir()

	// Content is valid JSON but the filename has no extension -> reject.
	noExt := filepath.Join(dir, "foo")
	writeFile(t, noExt, validJSON)
	if _, err := parseSimImport(noExt); err == nil {
		t.Fatal("extensionless file with JSON content must be rejected")
	}

	// Same content under a .json name -> parse.
	jsonPath := filepath.Join(dir, "foo.json")
	writeFile(t, jsonPath, validJSON)
	if _, err := parseSimImport(jsonPath); err != nil {
		t.Fatalf("foo.json with JSON content: %v", err)
	}
}
