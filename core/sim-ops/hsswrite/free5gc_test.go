package hsswrite

import (
	"context"
	"strings"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

func f5cLabDriver(fr *fakeRunner) *Free5GC {
	return &Free5GC{
		Container: "mongo",
		DBURI:     Free5GCDBURI,
		PlmnID:    "99999",
		SST:       1,
		SD:        "010203",
		run:       fr.run,
	}
}

func TestFree5GCAddWritesAllCollections(t *testing.T) {
	sub, _ := simprov.LoadTestVector("999991234567001")
	fr := &fakeRunner{}
	f := f5cLabDriver(fr)
	if err := f.Add(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if len(fr.calls) != 1 {
		t.Fatalf("expected 1 mongosh call, got %d", len(fr.calls))
	}
	args := fr.calls[0]
	eval := args[len(args)-1]
	if args[len(args)-3] != Free5GCDBURI {
		t.Fatalf("db uri = %q, want %q", args[len(args)-3], Free5GCDBURI)
	}

	// every record collection must be upserted, using bracket notation
	for _, coll := range []string{
		f5cAuthSubsColl, f5cAmDataColl, f5cSmDataColl, f5cSmfSelColl,
		f5cAmPolicyColl, f5cSmPolicyColl, f5cIdentityColl,
	} {
		if !strings.Contains(eval, `db["`+coll+`"].updateOne(`) {
			t.Errorf("eval missing upsert for %s", coll)
		}
	}
	if !strings.Contains(eval, "{upsert: true}") {
		t.Error("eval must upsert")
	}

	// credentials stay on the node: ki/opc must be in the eval (mongosh argv)
	if !strings.Contains(eval, `"encPermanentKey":"`+sub.Ki) || !strings.Contains(eval, `"encOpcKey":"`+sub.OPc) {
		t.Fatal("eval must carry ki/opc (they stay on the node)")
	}
	// servingPlmnId + ueId shape
	if !strings.Contains(eval, `"ueId":"imsi-999991234567001"`) || !strings.Contains(eval, `"servingPlmnId":"99999"`) {
		t.Fatalf("ueId/servingPlmnId missing: %s", eval)
	}
	// slice key and dnn
	if !strings.Contains(eval, `"01010203"`) {
		t.Fatal("smf-selection/policy slice key missing")
	}
	if !strings.Contains(eval, `"internet"`) {
		t.Fatal("dnn missing")
	}
}

func TestFree5GCAddAppliesPolicyAMBR(t *testing.T) {
	sub, _ := simprov.LoadTestVector("999991234567001")
	sub.QoSDLMbps = 50
	sub.QoSULMbps = 25
	fr := &fakeRunner{}
	f := f5cLabDriver(fr)
	if err := f.Add(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	eval := fr.calls[0][len(fr.calls[0])-1]
	if !strings.Contains(eval, `"subscribedUeAmbr":{"downlink":"50 Mbps","uplink":"25 Mbps"}`) {
		t.Fatalf("policy AMBR caps missing from eval: %s", eval)
	}
	if !strings.Contains(eval, `"sessionAmbr":{"downlink":"50 Mbps","uplink":"25 Mbps"}`) {
		t.Fatalf("session AMBR caps missing from eval: %s", eval)
	}
}

func TestFree5GCAddDefaultsAMBR(t *testing.T) {
	sub, _ := simprov.LoadTestVector("999991234567001") // QoS fields zero
	fr := &fakeRunner{}
	f := f5cLabDriver(fr)
	if err := f.Add(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	eval := fr.calls[0][len(fr.calls[0])-1]
	if !strings.Contains(eval, `"uplink":"1 Gbps"`) {
		t.Fatalf("default AMBR missing from eval: %s", eval)
	}
}

func TestFree5GCRemoveDeletesAllCollections(t *testing.T) {
	fr := &fakeRunner{}
	f := f5cLabDriver(fr)
	if err := f.Remove(context.Background(), "999991234567001"); err != nil {
		t.Fatal(err)
	}
	eval := fr.calls[0][len(fr.calls[0])-1]
	for _, coll := range []string{
		f5cAuthSubsColl, f5cAmDataColl, f5cSmDataColl, f5cSmfSelColl,
		f5cAmPolicyColl, f5cSmPolicyColl, f5cIdentityColl,
	} {
		if !strings.Contains(eval, `db["`+coll+`"].deleteMany({ueId: "imsi-999991234567001"})`) {
			t.Errorf("eval missing deleteMany for %s", coll)
		}
	}
}

func TestFree5GCPropagatesError(t *testing.T) {
	sub, _ := simprov.LoadTestVector("999991234567001")
	fr := &fakeRunner{err: context.DeadlineExceeded, out: []byte("mongo down")}
	f := f5cLabDriver(fr)
	err := f.Add(context.Background(), sub)
	if err == nil || !strings.Contains(err.Error(), "mongo down") {
		t.Fatalf("error must include command output: %v", err)
	}
}

func TestAmbrStrAndSnssaiKey(t *testing.T) {
	if ambrStr(0) != "1 Gbps" || ambrStr(50) != "50 Mbps" {
		t.Fatalf("ambrStr: %q %q", ambrStr(0), ambrStr(50))
	}
	if k := f5cSnssaiKey(1, "010203"); k != "01010203" {
		t.Fatalf("snssai key: %q", k)
	}
}
