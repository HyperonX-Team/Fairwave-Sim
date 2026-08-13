package hsswrite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

// fakeRunner records invocations and returns canned output.
type fakeRunner struct {
	calls [][]string
	err   error
	out   []byte
}

func (f *fakeRunner) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	return f.out, f.err
}

func labSub() simprov.Subscriber {
	sub, _ := simprov.LoadTestVector("999991234567001")
	return sub
}

func TestMongoshAddBuildsUpsert(t *testing.T) {
	sub := labSub()
	fr := &fakeRunner{}
	m := &Mongosh{Container: "mongo", DBURI: DefaultDBURI, run: fr.run}

	if err := m.Add(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if len(fr.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fr.calls))
	}
	args := fr.calls[0]
	want := []string{"docker", "exec", "mongo", "mongosh", "--quiet", DefaultDBURI, "--eval"}
	for i, w := range want {
		if args[i] != w {
			t.Fatalf("arg %d = %q, want %q", i, args[i], w)
		}
	}
	eval := args[len(args)-1]
	if !strings.Contains(eval, `updateOne({imsi: "999991234567001"}`) {
		t.Fatalf("eval missing imsi: %s", eval)
	}
	if !strings.Contains(eval, `"k":"`+sub.Ki) || !strings.Contains(eval, `"opc":"`+sub.OPc) {
		t.Fatal("eval must carry ki/opc (they stay on the node)")
	}
	if !strings.Contains(eval, `"apn":"`+sub.APN) {
		t.Fatal("eval must carry the apn")
	}
	if !strings.Contains(eval, `{upsert: true}`) {
		t.Fatal("eval must upsert")
	}
}

func TestMongoshAddWritesPolicyAMBR(t *testing.T) {
	sub := labSub()
	sub.QoSDLMbps = 50
	sub.QoSULMbps = 25
	fr := &fakeRunner{}
	m := &Mongosh{Container: "mongo", DBURI: DefaultDBURI, run: fr.run}
	if err := m.Add(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	eval := fr.calls[0][len(fr.calls[0])-1]
	if !strings.Contains(eval, `"downlink":{"unit":8,"value":50}`) ||
		!strings.Contains(eval, `"uplink":{"unit":8,"value":25}`) {
		t.Fatalf("AMBR caps missing from eval: %s", eval)
	}
}

func TestMongoshAddDefaultsAMBR(t *testing.T) {
	fr := &fakeRunner{}
	m := &Mongosh{Container: "mongo", DBURI: DefaultDBURI, run: fr.run}
	if err := m.Add(context.Background(), labSub()); err != nil { // QoS fields zero
		t.Fatal(err)
	}
	eval := fr.calls[0][len(fr.calls[0])-1]
	if !strings.Contains(eval, `"unit":8,"value":1`) {
		t.Fatalf("default AMBR missing from eval: %s", eval)
	}
}

func TestMongoshAddPropagatesError(t *testing.T) {
	fr := &fakeRunner{err: errors.New("exec failed"), out: []byte("boom")}
	m := &Mongosh{Container: "mongo", DBURI: DefaultDBURI, run: fr.run}
	err := m.Add(context.Background(), labSub())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error must include command output: %v", err)
	}
}

func TestMongoshRemove(t *testing.T) {
	fr := &fakeRunner{}
	m := &Mongosh{Container: "mongo", DBURI: DefaultDBURI, run: fr.run}
	if err := m.Remove(context.Background(), "999991234567009"); err != nil {
		t.Fatal(err)
	}
	eval := fr.calls[0][len(fr.calls[0])-1]
	if !strings.Contains(eval, `deleteOne({imsi: "999991234567009"})`) {
		t.Fatalf("eval = %s", eval)
	}
}

func TestDBCTLAddAndRemove(t *testing.T) {
	sub := labSub()
	fr := &fakeRunner{}
	d := &DBCTL{Container: "open5gs", run: fr.run}

	if err := d.Add(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	args := fr.calls[0]
	want := []string{"docker", "exec", "open5gs", "open5gs-dbctl", "add", sub.IMSI, sub.Ki, sub.OPc}
	for i, w := range want {
		if args[i] != w {
			t.Fatalf("add arg %d = %q, want %q", i, args[i], w)
		}
	}

	if err := d.Remove(context.Background(), sub.IMSI); err != nil {
		t.Fatal(err)
	}
	args = fr.calls[1]
	if args[len(args)-2] != "remove" || args[len(args)-1] != sub.IMSI {
		t.Fatalf("remove args = %v", args)
	}
}

func TestNewDriverSelection(t *testing.T) {
	if _, ok := New(DriverMongosh, "mongo").(*Mongosh); !ok {
		t.Fatal("mongosh driver must return *Mongosh")
	}
	if _, ok := New(DriverDBCTL, "open5gs").(*DBCTL); !ok {
		t.Fatal("dbctl driver must return *DBCTL")
	}
	if _, ok := New(DriverNone, "").(None); !ok {
		t.Fatal("none driver must return None")
	}
	if _, ok := New("bogus", "").(None); !ok {
		t.Fatal("unknown driver must fall back to None")
	}
	if _, ok := New("", "").(None); !ok {
		t.Fatal("empty driver must fall back to None")
	}
}

func TestNoneIsSafeNoop(t *testing.T) {
	n := None{}
	if err := n.Add(context.Background(), labSub()); err != nil {
		t.Fatalf("None.Add must not fail: %v", err)
	}
	if err := n.Remove(context.Background(), "999991234567001"); err != nil {
		t.Fatalf("None.Remove must not fail: %v", err)
	}
}
