package collector

import (
	"context"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// ---- synthetic GTP frame builders ----

func ie(t byte, val []byte) []byte {
	out := make([]byte, 4+len(val))
	out[0] = t
	out[2], out[3] = byte(len(val)>>8), byte(len(val))
	copy(out[4:], val)
	return out
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func encodeIMSI(imsi string) []byte {
	out := make([]byte, 0, (len(imsi)+1)/2)
	for i := 0; i < len(imsi); i += 2 {
		hi := imsi[i] - '0'
		lo := byte(0xF)
		if i+1 < len(imsi) {
			lo = imsi[i+1] - '0'
		}
		out = append(out, hi<<4|lo)
	}
	return out
}

func fteidVal(ifaceType byte, teid uint32) []byte {
	v := make([]byte, 9)
	v[0] = ifaceType | 0x10 // IPv4 present
	binary.BigEndian.PutUint32(v[1:5], teid)
	copy(v[5:9], []byte{10, 0, 0, 1})
	return v
}

func gtpv2C(msgType byte, ies []byte) []byte {
	out := make([]byte, 8+len(ies))
	out[0] = 0x20 // GTPv2
	out[1] = msgType
	binary.BigEndian.PutUint16(out[2:4], uint16(len(ies)))
	copy(out[8:], ies)
	return out
}

func gtpv1U(teid uint32, payload []byte) []byte {
	out := make([]byte, 8+len(payload))
	out[0] = 0x30 // GTPv1-U (version bits 6-8 = 001, PT bit 5 = 1)
	out[1] = gtpv1GPDU
	binary.BigEndian.PutUint16(out[2:4], uint16(len(payload)))
	binary.BigEndian.PutUint32(out[4:8], teid)
	copy(out[8:], payload)
	return out
}

// wrapIP wraps a GTP payload in an IPv4 + UDP header (the RawSocket/feed
// contract: one IP packet per read).
func wrapIP(gtp []byte) []byte {
	total := 20 + 8 + len(gtp)
	out := make([]byte, total)
	out[0] = 0x45
	binary.BigEndian.PutUint16(out[2:4], uint16(total))
	out[9] = 17 // UDP
	binary.BigEndian.PutUint16(out[24:26], uint16(8+len(gtp)))
	copy(out[28:], gtp)
	return out
}

// S5-C Create Session Request: IMSI + bearer context carrying the SGW-U
// S5-U F-TEID (interface type 5 -> uplink).
func createSessionReq(imsi string, sgwS5UTeid uint32) []byte {
	bearer := ie(ieBearerCtx, concat(
		ie(87, fteidVal(ftS5S8SG, sgwS5UTeid)),
		ie(73, []byte{5}), // EPS bearer id
	))
	return wrapIP(gtpv2C(gtpv2CreateSessionReq, concat(ie(ieIMSI, encodeIMSI(imsi)), bearer)))
}

// S5-C Create Session Response: IMSI + bearer context with the PGW-U S5-U
// F-TEID (interface type 6 -> downlink).
func createSessionResp(imsi string, pgwS5UTeid uint32) []byte {
	bearer := ie(ieBearerCtx, concat(
		ie(87, fteidVal(ftS5S8PG, pgwS5UTeid)),
		ie(73, []byte{5}),
	))
	return wrapIP(gtpv2C(gtpv2CreateSessionResp, concat(ie(ieIMSI, encodeIMSI(imsi)), bearer)))
}

// S11 Modify Bearer Request: IMSI + bearer context with the eNB S1-U
// F-TEID (interface type 0 -> uplink on S1-U).
func modifyBearerReq(imsi string, enbS1UTeid uint32) []byte {
	bearer := ie(ieBearerCtx, concat(
		ie(87, fteidVal(ftS1UEnb, enbS1UTeid)),
		ie(73, []byte{5}),
	))
	return wrapIP(gtpv2C(gtpv2ModifyBearerReq, concat(ie(ieIMSI, encodeIMSI(imsi)), bearer)))
}

func deleteSessionReq(imsi string) []byte {
	return wrapIP(gtpv2C(gtpv2DeleteSessionReq, ie(ieIMSI, encodeIMSI(imsi))))
}

func gPDU(teid uint32, n int) []byte {
	return wrapIP(gtpv1U(teid, make([]byte, n)))
}

// ---- feed packet source ----

type feed struct{ ch chan []byte }

func (f *feed) Next() ([]byte, error) {
	pkt, ok := <-f.ch
	if !ok {
		return nil, io.EOF
	}
	return pkt, nil
}

func (f *feed) Close() error {
	close(f.ch)
	return nil
}

func newFeed() *feed { return &feed{ch: make(chan []byte, 256)} }

// ---- parser tests ----

func TestParseGTPv1UGPDU(t *testing.T) {
	pkt := gPDU(0xDEADBEEF, 1400)
	f, ok := parseUDPGTP(pkt)
	if !ok || f.version != 1 || f.msgType != gtpv1GPDU || f.teid != 0xDEADBEEF {
		t.Fatalf("parse g-pdu: %+v ok=%v", f, ok)
	}
	if len(f.payload) != 1400 {
		t.Fatalf("payload len = %d, want 1400", len(f.payload))
	}
}

func TestParseGTPv2CCreateSession(t *testing.T) {
	pkt := createSessionReq("999991234567001", 0x1111)
	f, ok := parseUDPGTP(pkt)
	if !ok || f.version != 2 || f.msgType != gtpv2CreateSessionReq {
		t.Fatalf("parse gtpv2-c: %+v ok=%v", f, ok)
	}
	imsi, teids := parseGTPv2C(f)
	if imsi != "999991234567001" {
		t.Fatalf("imsi = %q", imsi)
	}
	if len(teids) != 1 || teids[0].ifaceType != ftS5S8SG || teids[0].teid != 0x1111 {
		t.Fatalf("teids = %+v", teids)
	}
}

func TestParseNonGTPIgnored(t *testing.T) {
	// plain UDP DNS-like packet: no GTP version -> rejected
	pkt := wrapIP([]byte{0, 1, 0, 0, 0, 0, 0, 0})
	if _, ok := parseUDPGTP(pkt); ok {
		t.Fatal("non-GTP packet must be rejected")
	}
	// truncated
	if _, ok := parseUDPGTP([]byte{0x45, 0, 0, 20}); ok {
		t.Fatal("truncated packet must be rejected")
	}
}

// ---- accountant tests ----

func TestAccountAttribution(t *testing.T) {
	a := newAccountant(time.Now)
	// session setup on S5-U
	a.observe(createSessionReq("999991234567001", 0x1111))
	a.observe(createSessionResp("999991234567001", 0x2222))
	// traffic
	a.observe(gPDU(0x1111, 100)) // UL
	a.observe(gPDU(0x2222, 250)) // DL
	a.observe(gPDU(0x1111, 50))  // UL again

	sess, unknown := a.snapshot()
	if unknown != 0 {
		t.Fatalf("unknown = %d, want 0", unknown)
	}
	s, ok := sess["999991234567001"]
	if !ok {
		t.Fatalf("session missing: %+v", sess)
	}
	if s.BytesUp != 150 || s.BytesDn != 250 {
		t.Fatalf("bytes = up:%d dn:%d, want 150/250", s.BytesUp, s.BytesDn)
	}
}

func TestAccountDeleteSession(t *testing.T) {
	a := newAccountant(time.Now)
	a.observe(createSessionReq("999991234567001", 0x1111))
	a.observe(gPDU(0x1111, 100))
	a.observe(deleteSessionReq("999991234567001"))
	a.observe(gPDU(0x1111, 50)) // TEID forgotten -> unknown

	sess, unknown := a.snapshot()
	if unknown != 50 {
		t.Fatalf("unknown = %d, want 50 (teid must be forgotten on delete)", unknown)
	}
	// the IMSI's prior usage is preserved (billing must never lose bytes)
	s, ok := sess["999991234567001"]
	if !ok || s.BytesUp != 100 {
		t.Fatalf("session after delete = %+v, want preserved 100 up", s)
	}
}

func TestAccountS1UAndMultipleUEs(t *testing.T) {
	a := newAccountant(time.Now)
	// S1-U leg: DL TEID comes from S11 Create Session Response (SGW S1-U),
	// UL TEID from S11 Modify Bearer Request (eNB S1-U).
	a.observe(createSessionReq("999991234567001", 0xAAAA)) // SGW S5-U (unused here)
	a.observe(createSessionResp("999991234567001", 0xAAAA))
	a.observe(modifyBearerReq("999991234567001", 0xBBBB))
	a.observe(createSessionReq("999991234567002", 0xCCCC))
	a.observe(createSessionResp("999991234567002", 0xCCCC))
	a.observe(modifyBearerReq("999991234567002", 0xDDDD))

	a.observe(gPDU(0xBBBB, 10)) // UE1 UL
	a.observe(gPDU(0xAAAA, 20)) // UE1 DL
	a.observe(gPDU(0xDDDD, 30)) // UE2 UL
	a.observe(gPDU(0xCCCC, 40)) // UE2 DL

	sess, _ := a.snapshot()
	u1 := sess["999991234567001"]
	u2 := sess["999991234567002"]
	if u1.BytesUp != 10 || u1.BytesDn != 20 {
		t.Fatalf("ue1 = %+v", u1)
	}
	if u2.BytesUp != 30 || u2.BytesDn != 40 {
		t.Fatalf("ue2 = %+v", u2)
	}
}

// ---- UPF source (Source interface) tests ----

func waitSessions(t *testing.T, u *UPF, want func([]api.Session) bool) []api.Session {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		sess, err := u.Poll(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if want(sess) {
			return sess
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for accountant state; last=%+v", sess)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestUPFSourceReportsCumulativeBytes(t *testing.T) {
	f := newFeed()
	u := NewUPF(UPFConfig{PacketSource: f, Now: time.Now})
	defer u.Close()

	f.ch <- createSessionReq("999991234567001", 0x1111)
	// cumulative: first poll reports the attached UE with zero bytes
	waitSessions(t, u, func(s []api.Session) bool {
		for _, x := range s {
			if x.IMSIHash == api.HashIMSI("999991234567001") {
				return true
			}
		}
		return false
	})

	f.ch <- gPDU(0x1111, 400)
	f.ch <- createSessionResp("999991234567001", 0x2222)
	f.ch <- gPDU(0x2222, 600)
	waitSessions(t, u, func(s []api.Session) bool {
		for _, x := range s {
			if x.IMSIHash == api.HashIMSI("999991234567001") && x.BytesUp == 400 && x.BytesDn == 600 {
				return true
			}
		}
		return false
	})

	// more traffic: cumulative counters keep growing (the usage pipeline
	// computes deltas, so monotonic is the contract).
	f.ch <- gPDU(0x1111, 100)
	waitSessions(t, u, func(s []api.Session) bool {
		for _, x := range s {
			if x.IMSIHash == api.HashIMSI("999991234567001") && x.BytesUp == 500 {
				return true
			}
		}
		return false
	})
}

func TestUPFSourceStartFailsOnBadIface(t *testing.T) {
	u := NewUPF(UPFConfig{Iface: "definitely-not-an-iface-xyz"})
	if _, err := u.Poll(context.Background()); err == nil {
		t.Fatal("polling a nonexistent interface must fail")
	}
}
