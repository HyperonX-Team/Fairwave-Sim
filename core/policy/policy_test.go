package policy

import (
	"os"
	"testing"
)

func TestLabProfileDeniesEverything(t *testing.T) {
	r, err := DefaultRegistry("")
	if err != nil {
		t.Fatal(err)
	}
	// The LAB profile only allows "zmq" (virtual) and requires indoor.
	v := r.Check("LAB", "zmq", true, "")
	if !v.Allowed {
		t.Fatalf("zmq in LAB should be allowed, got %v: %v", v.Allowed, v.Reasons)
	}
	v = r.Check("LAB", "zmq", false, "")
	if v.Allowed {
		t.Fatal("outdoor zmq should be denied")
	}
	v = r.Check("LAB", "n48", true, "")
	if v.Allowed {
		t.Fatal("n48 in LAB must be denied")
	}
	v = r.Check("US", "zmq", true, "")
	if v.Allowed {
		t.Fatal("zmq is not a US band")
	}
}

func TestUSCBRSRequiresLicenseRef(t *testing.T) {
	r, _ := DefaultRegistry("")
	v := r.Check("US", "n48", true, "")
	if v.Allowed {
		t.Fatal("n48 must require a license ref")
	}
	v = r.Check("US", "n48", true, "SAS-GAA-123")
	if !v.Allowed {
		t.Fatalf("n48 with SAS ref should pass: %v", v.Reasons)
	}
}

func TestUnknownCountryDenied(t *testing.T) {
	r, _ := DefaultRegistry("")
	v := r.Check("ZZ", "n48", true, "anything")
	if v.Allowed {
		t.Fatal("unknown country must deny")
	}
}

func TestIndoorOnlyBands(t *testing.T) {
	r, _ := DefaultRegistry("")
	v := r.Check("US", "n46", false, "")
	if v.Allowed {
		t.Fatal("n46 is indoor-only; outdoor must be denied")
	}
	v = r.Check("US", "n46", true, "U-NII-4")
	if !v.Allowed {
		t.Fatalf("n46 indoor with ref should pass: %v", v.Reasons)
	}
}

func TestOverlayMerge(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/overlay.json"
	if err := writeFile(path, `{"XX":{"country":"XX","mcc":"001","bands":{"b99":{"earfcn_min":1,"earfcn_max":2,"indoor_only":false,"max_eirp_dbm":0,"license_type":"lab"}}}}`); err != nil {
		t.Fatal(err)
	}
	r, err := DefaultRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	v := r.Check("XX", "b99", false, "")
	if !v.Allowed {
		t.Fatalf("overlay band should pass: %v", v.Reasons)
	}
	// builtin still present
	if _, ok := r.profiles["US"]; !ok {
		t.Fatal("overlay must not drop builtin profiles")
	}
}

func TestGatePhrase(t *testing.T) {
	if GateAckPhrase != "I hold authorization for this transmission" {
		t.Fatalf("gate phrase changed: %q", GateAckPhrase)
	}
}

func writeFile(path, content string) error {
	return osWriteFile(path, content)
}

func osWriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
