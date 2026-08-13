package collector

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// ---- byte-faithful fixture builders ----
//
// These replicate the encoding produced by free5gc/chf cdr/asn
// (ber_marshal.go) + cdr/cdrFile so the tests exercise the exact byte
// layout the CHF writes: context-specific tags for ASN.1 SET/SEQUENCE
// fields, universal SEQUENCE for nested lists, minimal INTEGER encoding.

func berTLV(id byte, content []byte) []byte {
	out := []byte{id}
	if len(content) < 128 {
		out = append(out, byte(len(content)))
	} else {
		n := 1
		for l := len(content); l > 255; l >>= 8 {
			n++
		}
		out = append(out, byte(n)|0x80)
		for i := n - 1; i >= 0; i-- {
			out = append(out, byte(len(content)>>(8*i)))
		}
	}
	return append(out, content...)
}

func ctxPrim(tag byte, content []byte) []byte { return berTLV(0x80|tag, content) }
func ctxCons(tag byte, content []byte) []byte { return berTLV(0xA0|tag, content) }
func uniSeq(content []byte) []byte            { return berTLV(0x30, content) }

// berInt is the minimal big-endian two's-complement INTEGER encoding used
// by free5gc's int64Encoder (values here are small and non-negative).
func berInt(n int64) []byte {
	var out []byte
	for n > 127 {
		out = append([]byte{byte(n & 0xff)}, out...)
		n >>= 8
	}
	return append([]byte{byte(n)}, out...)
}

func usedUnit(ul, dn, total int64) []byte {
	return uniSeq(append(append(append([]byte{},
		ctxPrim(4, berInt(total))...), // DataTotalVolume
		ctxPrim(5, berInt(ul))...), // DataVolumeUplink
		ctxPrim(6, berInt(dn))...)) // DataVolumeDownlink
}

func multiUnitUsage(rg int64, units ...[]byte) []byte {
	var containers []byte
	for _, u := range units {
		containers = append(containers, u...)
	}
	return uniSeq(append(ctxPrim(0, berInt(rg)), ctxCons(1, containers)...))
}

func subscriptionID(imsi string) []byte {
	return ctxCons(2, append(ctxPrim(0, berInt(1)), ctxPrim(1, []byte(imsi))...))
}

func pduSessionInfo(dnn string) []byte {
	return ctxCons(13, append(append(
		ctxPrim(0, berInt(1)),
		ctxPrim(6, berInt(1))...),
		ctxPrim(13, []byte(dnn))...))
}

func chargingRecord(imsi, apn string, muus ...[]byte) []byte {
	var f []byte
	f = append(f, ctxPrim(0, berInt(200))...)   // RecordType
	f = append(f, ctxPrim(1, []byte("SMF"))...) // RecordingNetworkFunctionID
	f = append(f, subscriptionID(imsi)...)
	f = append(f, ctxCons(3, ctxPrim(0, berInt(7)))...) // NFunctionConsumerInformation
	if apn != "" {
		f = append(f, pduSessionInfo(apn)...)
	}
	var list []byte
	for _, m := range muus {
		list = append(list, m...) // SEQUENCE OF: elements sit directly under [5]
	}
	f = append(f, ctxCons(5, list)...) // ListOfMultipleUnitUsage
	return ctxCons(1, f)               // ChargingRecord (SET, context [1])
}

func cdrFile(records ...[]byte) []byte {
	header := make([]byte, 52)
	binary.BigEndian.PutUint32(header[4:8], 52)
	binary.BigEndian.PutUint32(header[18:22], uint32(len(records)))
	var body []byte
	for _, rec := range records {
		hdr := make([]byte, 4)
		binary.BigEndian.PutUint16(hdr[0:2], uint16(len(rec)))
		hdr[3] = 1 << 5 // DataRecordFormat = BasicEncodingRules
		body = append(body, hdr...)
		body = append(body, rec...)
	}
	binary.BigEndian.PutUint32(header[0:4], uint32(52+len(body)))
	return append(header, body...)
}

func writeCDR(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func newCDRSource(t *testing.T, dir string) *Free5GCCDR {
	t.Helper()
	return NewCDR(CDRConfig{Dir: dir, Now: func() time.Time { return time.Unix(1700000000, 0) }})
}

func TestParseCDRFile(t *testing.T) {
	rec1 := chargingRecord("999991234567001", "internet",
		multiUnitUsage(1, usedUnit(150, 250, 400), usedUnit(50, 50, 100)))
	rec2 := chargingRecord("999991234567002", "",
		multiUnitUsage(2, usedUnit(1000, 2000, 3000)))
	data := cdrFile(rec1, rec2)

	usages, err := parseCDRFile(data)
	if err != nil {
		t.Fatalf("parseCDRFile: %v", err)
	}
	if len(usages) != 2 {
		t.Fatalf("usages = %d, want 2", len(usages))
	}
	got := map[string]cdrUsage{}
	for _, u := range usages {
		got[u.imsi] = u
	}
	u1 := got["999991234567001"]
	if u1.up != 200 || u1.dn != 300 {
		t.Fatalf("ue1 bytes = up:%d dn:%d, want 200/300", u1.up, u1.dn)
	}
	if u1.apn != "internet" {
		t.Fatalf("ue1 apn = %q, want internet", u1.apn)
	}
	u2 := got["999991234567002"]
	if u2.up != 1000 || u2.dn != 2000 {
		t.Fatalf("ue2 bytes = up:%d dn:%d, want 1000/2000", u2.up, u2.dn)
	}
}

func TestParseCDRFileRejectsGarbage(t *testing.T) {
	if _, err := parseCDRFile([]byte("not a cdr file at all")); err == nil {
		t.Fatal("short file must fail")
	}
	if _, err := parseCDRFile(cdrFile([]byte{0x01, 0x02, 0x03})); err == nil {
		t.Fatal("garbage record must fail")
	}
}

func TestCDRPollSnapshotsAndReset(t *testing.T) {
	dir := t.TempDir()
	src := newCDRSource(t, dir)

	// First snapshot: UE1 at 500 up / 300 dn, UE2 at 100 up.
	writeCDR(t, dir, "chf-aaa.cdr", cdrFile(
		chargingRecord("999991234567001", "internet", multiUnitUsage(1, usedUnit(500, 300, 800))),
		chargingRecord("999991234567002", "ims", multiUnitUsage(1, usedUnit(100, 0, 100))),
	))
	sessions, err := src.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	byHash := map[string]api.Session{}
	for _, s := range sessions {
		byHash[s.IMSIHash] = s
	}
	s1, ok := byHash[api.HashIMSI("999991234567001")]
	if !ok || s1.BytesUp != 500 || s1.BytesDn != 300 || s1.APN != "internet" {
		t.Fatalf("ue1 session: %+v", s1)
	}
	s2, ok := byHash[api.HashIMSI("999991234567002")]
	if !ok || s2.BytesUp != 100 || s2.APN != "ims" {
		t.Fatalf("ue2 session: %+v", s2)
	}

	// Snapshot reset (session released / CHF restarted): totals drop. The
	// source reports the absolute file contents; the delta-fold upstream
	// absorbs the reset.
	writeCDR(t, dir, "chf-aaa.cdr", cdrFile(
		chargingRecord("999991234567001", "internet", multiUnitUsage(1, usedUnit(50, 20, 70))),
	))
	sessions, err = src.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll after reset: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions after reset = %d, want 1 (ue2 file removed)", len(sessions))
	}
	if sessions[0].IMSIHash != api.HashIMSI("999991234567001") || sessions[0].BytesUp != 50 {
		t.Fatalf("session after reset: %+v", sessions[0])
	}
}

func TestCDRPollTolerant(t *testing.T) {
	dir := t.TempDir()
	src := newCDRSource(t, dir)

	// A garbage *.cdr file (e.g. mid-write) must not break the poll for the
	// valid one next to it.
	writeCDR(t, dir, "chf-midwrite.cdr", []byte{0x00, 0x01})
	writeCDR(t, dir, "chf-valid.cdr", cdrFile(
		chargingRecord("999991234567001", "", multiUnitUsage(1, usedUnit(10, 20, 30))),
	))
	sessions, err := src.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(sessions) != 1 || sessions[0].BytesUp != 10 || sessions[0].BytesDn != 20 {
		t.Fatalf("sessions: %+v", sessions)
	}
}

func TestCDRPollMissingDir(t *testing.T) {
	src := newCDRSource(t, filepath.Join(t.TempDir(), "does-not-exist"))
	if _, err := src.Poll(context.Background()); err == nil {
		t.Fatal("missing cdr dir must error so the previous snapshot is kept")
	}
}

func TestParseChargingRecordMissingSubscriber(t *testing.T) {
	// A record with volume but no subscriber identifier is not attributable.
	body := ctxCons(1, ctxCons(5, uniSeq(multiUnitUsage(1, usedUnit(1, 2, 3)))))
	if _, err := parseChargingRecord(body); err == nil {
		t.Fatal("record without subscriber identifier must fail")
	}
}

// fixedSource is a Source returning a fixed snapshot (test double).
type fixedSource struct{ sessions []api.Session }

func (s fixedSource) Poll(context.Context) ([]api.Session, error) { return s.sessions, nil }

func TestMultiMergesSessions(t *testing.T) {
	// AMF OAM style: live session, no bytes. CDR style: usage only.
	amf := fixedSource{sessions: []api.Session{{
		IMSIHash: api.HashIMSI("999991234567001"), APN: "internet", Phase: "connected",
	}}}
	cdr := fixedSource{sessions: []api.Session{{
		IMSIHash: api.HashIMSI("999991234567001"), APN: "", Phase: "metered", BytesUp: 400, BytesDn: 600,
	}}}
	m := Multi{amf, cdr}
	sessions, err := m.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("merged sessions = %d, want 1", len(sessions))
	}
	s := sessions[0]
	if s.BytesUp != 400 || s.BytesDn != 600 {
		t.Fatalf("merged bytes = up:%d dn:%d, want 400/600", s.BytesUp, s.BytesDn)
	}
	if s.APN != "internet" {
		t.Fatalf("merged apn = %q, want internet (from live source)", s.APN)
	}
	if s.Phase != "connected" {
		t.Fatalf("merged phase = %q, want connected (from live source)", s.Phase)
	}
}
