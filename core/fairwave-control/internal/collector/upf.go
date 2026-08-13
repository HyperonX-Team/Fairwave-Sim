// GTP-U/GTP-C accounting: per-UE byte counters measured from the core's
// user plane.
//
// Open5GS does not export per-UE traffic bytes anywhere (its Prometheus
// metrics only carry aggregate per-QoS-level data volumes on the N3/N9
// interface; see https://open5gs.org/open5gs/docs/tutorial/04-metrics-prometheus/).
// To meter per-UE usage honestly we measure the user plane directly:
//
//   - a GTPv1-U (G-PDU) tap counts payload bytes per TEID as they transit
//     an interface carrying UE traffic (S1-U between the eNB and SGW-U, or
//     S5-U between SGW-U and PGW-U);
//   - a GTPv2-C (S11/S5) snoop learns the TEID -> IMSI map from session
//     signaling (Create/Modify/Delete Session), so every counted byte can
//     be attributed to a subscriber;
//   - the accountant exposes cumulative UL/DL bytes per IMSI as sessions,
//     which the control plane's existing usage pipeline (delta fold +
//     fair-use auto-suspend) consumes unchanged.
//
// Packet capture needs CAP_NET_RAW (root) on the host running the tap. The
// packet source is pluggable so tests can feed synthetic GTP frames.
package collector

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"golang.org/x/sys/unix"
)

// ---- GTP message constants (TS 29.281 / TS 29.274) ----

const (
	gtpv1GPDU = 255 // GTPv1-U user data

	// GTPv2-C message types (S11 / S5 / S8).
	gtpv2CreateSessionReq  = 32
	gtpv2CreateSessionResp = 33
	gtpv2ModifyBearerReq   = 34
	gtpv2ModifyBearerResp  = 35
	gtpv2DeleteSessionReq  = 36
	gtpv2DeleteSessionResp = 37

	// GTPv2-C IE types.
	ieIMSI      = 1
	ieFTEID     = 87
	ieBearerCtx = 93
	iePDNConn   = 96 // PDN Connection (recursed for bearer contexts)

	// F-TEID interface types we account (TS 29.274 §8.20.1).
	ftS1UEnb = 0 // S1-U eNodeB GTP-U   -> uplink TEID on S1-U
	ftS1USGW = 1 // S1-U SGW GTP-U      -> downlink TEID on S1-U
	ftS5S8SG = 5 // S5/S8 SGW GTP-U     -> uplink TEID on S5-U
	ftS5S8PG = 6 // S5/S8 PGW GTP-U     -> downlink TEID on S5-U
)

const (
	dirUL = 1
	dirDL = 2
)

// PacketSource delivers raw IP packets (network-layer header included).
// Implementations block until a packet is available; Close unblocks them.
type PacketSource interface {
	Next() ([]byte, error)
	Close() error
}

// ---- GTP parsing ----

// gtpFrame is the parsed common GTP header.
type gtpFrame struct {
	version   byte // 1 (GTPv1) or 2 (GTPv2-C)
	msgType   byte
	teid      uint32
	payload   []byte // bytes after the mandatory header
	payloadOK bool
}

// parseGTP extracts the GTP header from a UDP datagram payload.
//
// Octet 1 layout (TS 29.060/29.274): bits 6-8 hold the version, bit 5 the
// protocol type. GTPv1 (any PT) therefore reads as nibble 3, GTPv2-C as
// nibble 2; anything else is rejected.
func parseGTP(data []byte) (gtpFrame, bool) {
	if len(data) < 8 {
		return gtpFrame{}, false
	}
	var version byte
	switch data[0] >> 4 {
	case 3: // GTPv1 (version 1 + PT)
		version = 1
	case 2: // GTPv2-C
		version = 2
	default:
		return gtpFrame{}, false
	}
	f := gtpFrame{
		version: version,
		msgType: data[1],
		teid:    binary.BigEndian.Uint32(data[4:8]),
	}
	length := int(binary.BigEndian.Uint16(data[2:4]))
	end := 8 + length
	if end > len(data) || end < 8 {
		return gtpFrame{}, false
	}
	f.payload = data[8:end]
	f.payloadOK = true
	return f, true
}

// parseUDPGTP walks an IP packet (v4 or v6) and returns its GTP frame if
// the transport is UDP and the payload is GTP.
func parseUDPGTP(pkt []byte) (gtpFrame, bool) {
	if len(pkt) == 0 {
		return gtpFrame{}, false
	}
	var udp []byte
	switch pkt[0] >> 4 {
	case 4: // IPv4
		if len(pkt) < 20 {
			return gtpFrame{}, false
		}
		ihl := int(pkt[0]&0x0F) * 4
		if ihl < 20 || ihl+8 > len(pkt) || pkt[9] != 17 /* UDP */ {
			return gtpFrame{}, false
		}
		udp = pkt[ihl:]
	case 6: // IPv6
		if len(pkt) < 48 || pkt[6] != 17 /* UDP */ {
			return gtpFrame{}, false
		}
		udp = pkt[40:]
	default:
		return gtpFrame{}, false
	}
	// UDP header: sport(2) dport(2) length(2) csum(2); payload follows.
	if len(udp) < 8 {
		return gtpFrame{}, false
	}
	return parseGTP(udp[8:])
}

// ---- GTPv2-C IE walker ----

// fteid is one learned GTP-U tunnel endpoint.
type fteid struct {
	ifaceType byte
	teid      uint32
}

// parseGTPv2C parses a GTPv2-C message into its IMSI and the F-TEIDs it
// carries (recursing one level into Bearer Context / PDN Connection IEs,
// which is where S1-U/S5-U F-TEIDs live).
func parseGTPv2C(f gtpFrame) (imsi string, teids []fteid) {
	if f.version != 2 {
		return "", nil
	}
	walk(f.payload, 0, &imsi, &teids)
	return imsi, teids
}

// walk iterates top-level IEs; nested is 0 for top level, 1 for bearer
// contexts (deeper nesting is ignored - S1-U/S5-U F-TEIDs never sit deeper).
func walk(ies []byte, nested int, imsi *string, teids *[]fteid) {
	for len(ies) >= 4 {
		ieType := ies[0]
		ieLen := int(binary.BigEndian.Uint16(ies[2:4]))
		value := ies[4:]
		if ieLen > len(value) {
			return // malformed; stop
		}
		value = value[:ieLen]
		switch ieType {
		case ieIMSI:
			if *imsi == "" {
				*imsi = decodeIMSI(value)
			}
		case ieFTEID:
			if t, ok := parseFTEID(value); ok {
				*teids = append(*teids, t)
			}
		case ieBearerCtx, iePDNConn:
			if nested == 0 {
				walk(value, 1, imsi, teids)
			}
		}
		ies = ies[4+ieLen:]
	}
}

// parseFTEID decodes IE type 87 (F-TEID): interface type (low nibble of
// byte 0), 4-byte TEID, then the IP address(es).
func parseFTEID(v []byte) (fteid, bool) {
	if len(v) < 5 {
		return fteid{}, false
	}
	return fteid{
		ifaceType: v[0] & 0x0F,
		teid:      binary.BigEndian.Uint32(v[1:5]),
	}, true
}

// decodeIMSI decodes the 15-digit BCD IMSI IE value (8 octets).
func decodeIMSI(v []byte) string {
	if len(v) == 0 {
		return ""
	}
	var out []byte
	for _, b := range v {
		hi, lo := b>>4, b&0x0F
		if hi <= 9 {
			out = append(out, '0'+hi)
		}
		if lo <= 9 {
			out = append(out, '0'+lo)
		}
	}
	if len(out) > 15 {
		out = out[:15] // IMSI is 15 digits (last octet has spare bits)
	}
	return string(out)
}

// dirFor maps an F-TEID interface type to an accounting direction.
func dirFor(t fteid) (byte, bool) {
	switch t.ifaceType {
	case ftS1UEnb, ftS5S8SG:
		return dirUL, true
	case ftS1USGW, ftS5S8PG:
		return dirDL, true
	default:
		return 0, false
	}
}

// ---- accountant ----

// upfAccountant learns TEID -> IMSI bindings from GTP-C and folds GTP-U
// payload bytes into per-IMSI UL/DL totals.
type upfAccountant struct {
	mu      sync.Mutex
	byTEID  map[uint32]teidInfo   // teid -> owner + direction
	byIMSI  map[string]*imsiBytes // cumulative bytes per IMSI
	unknown uint64                // bytes on TEIDs we have not learned yet
	now     func() time.Time
}

type teidInfo struct {
	imsi string
	dir  byte
}

type imsiBytes struct {
	up, dn uint64
}

func newAccountant(now func() time.Time) *upfAccountant {
	return &upfAccountant{
		byTEID: map[uint32]teidInfo{},
		byIMSI: map[string]*imsiBytes{},
		now:    now,
	}
}

// observe consumes one IP packet. GTP-C messages update the TEID map;
// GTP-U G-PDUs accumulate bytes.
func (a *upfAccountant) observe(pkt []byte) {
	f, ok := parseUDPGTP(pkt)
	if !ok {
		return
	}
	switch {
	case f.version == 1 && f.msgType == gtpv1GPDU:
		a.count(f.teid, uint64(len(f.payload)))
	case f.version == 2:
		a.learn(f)
	}
}

func (a *upfAccountant) count(teid uint32, n uint64) {
	if n == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	info, ok := a.byTEID[teid]
	if !ok {
		a.unknown += n
		return
	}
	ib := a.byIMSI[info.imsi]
	if ib == nil {
		ib = &imsiBytes{}
		a.byIMSI[info.imsi] = ib
	}
	if info.dir == dirUL {
		ib.up += n
	} else {
		ib.dn += n
	}
}

// learn folds one GTPv2-C message into the TEID map.
func (a *upfAccountant) learn(f gtpFrame) {
	imsi, teids := parseGTPv2C(f)
	a.mu.Lock()
	defer a.mu.Unlock()
	switch f.msgType {
	case gtpv2CreateSessionReq, gtpv2CreateSessionResp, gtpv2ModifyBearerReq, gtpv2ModifyBearerResp:
		if imsi == "" {
			return
		}
		if a.byIMSI[imsi] == nil {
			a.byIMSI[imsi] = &imsiBytes{} // attached UE: report even at zero bytes
		}
		for _, t := range teids {
			dir, ok := dirFor(t)
			if !ok {
				continue
			}
			a.byTEID[t.teid] = teidInfo{imsi: imsi, dir: dir}
		}
	case gtpv2DeleteSessionReq, gtpv2DeleteSessionResp:
		if imsi == "" {
			return
		}
		for teid, info := range a.byTEID {
			if info.imsi == imsi {
				delete(a.byTEID, teid)
			}
		}
	}
}

// snapshot returns cumulative per-IMSI bytes. Zero-byte sessions are
// dropped once their TEIDs are gone (detached UEs), while any UE with
// bytes is always reported - billing must never lose a counter.
func (a *upfAccountant) snapshot() (map[string]api.Session, uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now().UTC()
	live := map[string]bool{}
	for _, info := range a.byTEID {
		live[info.imsi] = true
	}
	out := map[string]api.Session{}
	for imsi, ib := range a.byIMSI {
		if ib.up == 0 && ib.dn == 0 && !live[imsi] {
			delete(a.byIMSI, imsi)
			continue
		}
		out[imsi] = api.Session{
			IMSIHash: api.HashIMSI(imsi),
			APN:      "internet",
			Phase:    "active",
			BytesUp:  ib.up,
			BytesDn:  ib.dn,
			Created:  now,
		}
	}
	return out, a.unknown
}

// ---- raw socket packet source ----

// RawSocket reads IP packets from a network interface with an AF_PACKET
// datagram socket (no link-layer header). Requires CAP_NET_RAW.
type RawSocket struct {
	fd   int
	ifi  int
	name string
}

// NewRawSocket binds an AF_PACKET socket to the named interface. The
// socket is SOCK_DGRAM so each read yields a full IP packet.
func NewRawSocket(iface string) (*RawSocket, error) {
	if iface == "" {
		return nil, fmt.Errorf("collector: upf iface required")
	}
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("collector: interface %q: %w", iface, err)
	}
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM, int(htons(unix.ETH_P_ALL)))
	if err != nil {
		return nil, fmt.Errorf("collector: AF_PACKET socket (CAP_NET_RAW required): %w", err)
	}
	sa := &unix.SockaddrLinklayer{Protocol: htons(unix.ETH_P_ALL), Ifindex: ifi.Index}
	if err := unix.Bind(fd, sa); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("collector: bind %s: %w", iface, err)
	}
	return &RawSocket{fd: fd, ifi: ifi.Index, name: iface}, nil
}

func (r *RawSocket) Name() string { return r.name }

// Next reads the next IP packet.
func (r *RawSocket) Next() ([]byte, error) {
	buf := make([]byte, 65536)
	for {
		n, _, err := unix.Recvfrom(r.fd, buf, 0)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			return buf[:n], nil
		}
	}
}

// Close releases the socket.
func (r *RawSocket) Close() error {
	return unix.Close(r.fd)
}

func htons(v uint16) uint16 {
	return v<<8 | v>>8
}

// ---- UPF source ----

// UPFConfig configures the GTP accounting source.
type UPFConfig struct {
	// Iface is the interface carrying GTP-U (S1-U or S5-U), e.g. "fwnet".
	Iface string
	// PacketSource overrides the raw socket (tests). When nil a RawSocket
	// on Iface is opened on first use.
	PacketSource PacketSource
	// Now is the clock (injectable for tests).
	Now func() time.Time
}

// UPF meters per-UE usage from the core user plane. It implements Source:
// Poll returns cumulative per-IMSI UL/DL bytes as sessions, feeding the
// control plane's delta-based usage pipeline.
type UPF struct {
	cfg  UPFConfig
	acc  *upfAccountant
	src  PacketSource
	once sync.Once
	stop chan struct{}
	done chan struct{}

	unknownOnce sync.Once
}

// NewUPF builds the GTP accounting source.
func NewUPF(cfg UPFConfig) *UPF {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	u := &UPF{
		cfg:  cfg,
		acc:  newAccountant(cfg.Now),
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	return u
}

// start opens the packet source and runs the tap goroutine exactly once.
func (u *UPF) start() error {
	var err error
	u.once.Do(func() {
		src := u.cfg.PacketSource
		if src == nil {
			src, err = NewRawSocket(u.cfg.Iface)
			if err != nil {
				return
			}
		}
		u.src = src
		go u.tap(src)
	})
	return err
}

// tap consumes packets until the source fails or the source is closed.
func (u *UPF) tap(src PacketSource) {
	defer close(u.done)
	for {
		pkt, err := src.Next()
		if err != nil {
			return
		}
		select {
		case <-u.stop:
			return
		default:
		}
		u.acc.observe(pkt)
	}
}

// Poll implements Source: it returns the current cumulative per-IMSI
// usage snapshot. Sessions are reported even with zero bytes (the UE is
// attached); the usage pipeline ignores zero deltas.
func (u *UPF) Poll(ctx context.Context) ([]api.Session, error) {
	if err := u.start(); err != nil {
		return nil, err
	}
	sessions, unknown := u.acc.snapshot()
	if unknown > 0 {
		u.unknownOnce.Do(func() {
			log.Printf("collector.upf: %d bytes on unlearned TEIDs (tap started mid-session?); per-UE attribution is incomplete until GTP-C session setup is seen", unknown)
		})
	}
	out := make([]api.Session, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, s)
	}
	return out, nil
}

// Close stops the tap and releases the socket. Safe to call even if the
// source was never started (or failed to start).
func (u *UPF) Close() error {
	select {
	case <-u.stop:
	default:
		close(u.stop)
	}
	if u.src != nil {
		_ = u.src.Close()
	}
	select {
	case <-u.done:
	case <-time.After(2 * time.Second):
	}
	return nil
}
