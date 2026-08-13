// free5GC CHF CDR usage source.
//
// free5GC's SMF relays per-PDU-session usage reports (PFCP URR) to the
// CHF, which writes them as TS 32.297 CDR files - one per subscriber,
// named chf-<sha256(supi)>.cdr - into the CHF container's CGF directory
// (see free5gc/chf: internal/sbi/processor/cdr.go dumpCdrFile; the write
// path is hardcoded to /tmp). Each file is a binary container: a fixed
// file header, then per-record 4-byte headers plus an ASN.1 BER
// ChargingRecord (TS 32.255) body carrying the subscriber identifier and
// the per-rating-group data volumes (DataVolumeUplink/Downlink).
//
// The CHF rewrites the whole file on every charging update, so a snapshot
// holds the *current* cumulative volumes per subscriber. The source
// reports those absolute totals as session counters; the delta-fold in
// AccumulateUsage absorbs the resets (deltas clamp at zero and catch up
// when the next snapshot grows past the last reading). No raw sockets, no
// CAP_NET_RAW: usage is measured from the core itself.
package collector

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

// CDRConfig configures the free5GC CHF CDR driver.
type CDRConfig struct {
	// Dir is the directory the CHF writes CDR files into (a volume shared
	// with the CHF container's /tmp), e.g. /var/fairwave/chf-cdr.
	Dir string
	// Now is the clock used to timestamp sessions (injectable for tests).
	Now func() time.Time
}

// Free5GCCDR reads CHF CDR files and reports per-SIM usage as session
// counters (absolute totals, matching the UPF tap contract).
type Free5GCCDR struct {
	cfg CDRConfig
}

// NewCDR builds the free5GC CHF CDR driver.
func NewCDR(cfg CDRConfig) *Free5GCCDR {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Free5GCCDR{cfg: cfg}
}

// Poll implements Source: it scans the CDR directory, parses every *.cdr
// file, and returns one session per subscriber carrying the current
// cumulative bytes. Unparseable files (mid-write, or a foreign file
// matching the glob) are skipped; a missing directory is an error so the
// caller keeps the previous snapshot.
func (c *Free5GCCDR) Poll(ctx context.Context) ([]api.Session, error) {
	if c.cfg.Dir == "" {
		return nil, fmt.Errorf("collector: cdr_dir required")
	}
	entries, err := os.ReadDir(c.cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("collector: cdr: %w", err)
	}
	now := c.cfg.Now().UTC()
	byIMSI := map[string]*api.Session{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".cdr") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(c.cfg.Dir, e.Name()))
		if err != nil {
			continue
		}
		usages, err := parseCDRFile(data)
		if err != nil {
			continue // partial write or not a CDR file: try next poll
		}
		for _, u := range usages {
			if u.imsi == "" {
				continue
			}
			s, ok := byIMSI[u.imsi]
			if !ok {
				s = &api.Session{IMSIHash: api.HashIMSI(u.imsi), Phase: "metered", Created: now}
				byIMSI[u.imsi] = s
			}
			s.BytesUp += u.up
			s.BytesDn += u.dn
			if s.APN == "" {
				s.APN = u.apn
			}
		}
	}
	out := make([]api.Session, 0, len(byIMSI))
	for _, s := range byIMSI {
		out = append(out, *s)
	}
	return out, nil
}

// cdrUsage is one subscriber's usage carried by one CDR record.
type cdrUsage struct {
	imsi string
	apn  string
	up   uint64
	dn   uint64
}

// ---- TS 32.297 container (mirrors free5gc/chf cdr/cdrFile) ----

// parseCDRFile decodes a TS 32.297 CDR file: the fixed header, then one
// BER-encoded ChargingRecord per record. Volumes of records belonging to
// the same subscriber are summed.
func parseCDRFile(data []byte) ([]cdrUsage, error) {
	if len(data) < 52 {
		return nil, fmt.Errorf("cdr: file too short (%d bytes)", len(data))
	}
	// Fixed 52-byte header, then optional CDR routing filter + private
	// extension (mirrors free5gc/chf cdrFile.Decoding). free5GC writes
	// zero-length extensions, so this is defensive for other producers.
	routingLen := int(binary.BigEndian.Uint16(data[48:50]))
	off := 50 + routingLen
	if off+2 > len(data) {
		return nil, fmt.Errorf("cdr: truncated private-extension length")
	}
	privLen := int(binary.BigEndian.Uint16(data[off : off+2]))
	off += 2 + privLen
	if off < 52 || off > len(data) {
		return nil, fmt.Errorf("cdr: bad header length %d", off)
	}
	// Release identifier extension bytes (only when release == 7; free5GC
	// writes release 0, so this is defensive).
	if off < len(data) && data[8]>>5 == 7 {
		off++
	}
	if off < len(data) && data[9]>>5 == 7 {
		off++
	}

	num := int(binary.BigEndian.Uint32(data[18:22]))
	if num == 0 || num > 1<<16 {
		return nil, fmt.Errorf("cdr: implausible record count %d", num)
	}
	var out []cdrUsage
	for i := 0; i < num; i++ {
		if off+4 > len(data) {
			return nil, fmt.Errorf("cdr: truncated record header")
		}
		bodyLen := int(binary.BigEndian.Uint16(data[off : off+2]))
		release := data[off+2] >> 5
		recOff := off + 4
		if release == 7 {
			recOff++ // release identifier extension
		}
		if recOff+bodyLen > len(data) {
			return nil, fmt.Errorf("cdr: record %d body out of range", i)
		}
		u, err := parseChargingRecord(data[recOff : recOff+bodyLen])
		if err != nil {
			return nil, err
		}
		out = append(out, u)
		off = recOff + bodyLen
	}
	return out, nil
}

// ---- ASN.1 BER decoding (tolerant subset of TS 32.255) ----

const (
	classUniversal = 0
	classContext   = 2
)

// tlv is one BER tag-length-value element.
type tlv struct {
	class       int
	constructed bool
	tag         int
	content     []byte
}

// parseTLV reads one BER TLV and returns the remainder.
func parseTLV(b []byte) (tlv, []byte, error) {
	var t tlv
	if len(b) < 2 {
		return t, nil, fmt.Errorf("ber: short element")
	}
	id := b[0]
	t.class = int(id >> 6)
	t.constructed = id&0x20 != 0
	t.tag = int(id & 0x1f)
	off := 1
	if t.tag == 0x1f { // high tag number form
		t.tag = 0
		for {
			if off >= len(b) {
				return t, nil, fmt.Errorf("ber: truncated tag")
			}
			t.tag = t.tag<<7 | int(b[off]&0x7f)
			last := b[off]&0x80 == 0
			off++
			if last {
				break
			}
		}
	}
	if off >= len(b) {
		return t, nil, fmt.Errorf("ber: truncated length")
	}
	length := int64(b[off])
	off++
	if length&0x80 != 0 {
		n := int(length & 0x7f)
		if n == 0 || n > 4 || off+n > len(b) {
			return t, nil, fmt.Errorf("ber: bad long-form length")
		}
		length = 0
		for i := 0; i < n; i++ {
			length = length<<8 | int64(b[off+i])
		}
		off += n
	}
	if length < 0 || off+int(length) > len(b) {
		return t, nil, fmt.Errorf("ber: value out of range")
	}
	t.content = b[off : off+int(length)]
	return t, b[off+int(length):], nil
}

// parseChildren splits a constructed element's content into TLVs.
func parseChildren(content []byte) ([]tlv, error) {
	var out []tlv
	rest := content
	for len(rest) > 0 {
		t, r, err := parseTLV(rest)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
		rest = r
	}
	return out, nil
}

// findTag returns the first context-class element with the given tag.
func findTag(tlvs []tlv, tag int) (tlv, bool) {
	for _, t := range tlvs {
		if t.class == classContext && t.tag == tag {
			return t, true
		}
	}
	return tlv{}, false
}

// findChild is findTag inside a constructed element's content.
func findChild(t tlv, tag int) (tlv, bool) {
	children, err := parseChildren(t.content)
	if err != nil {
		return tlv{}, false
	}
	return findTag(children, tag)
}

// uintValue decodes a non-negative INTEGER TLV (DataVolumeOctets).
func uintValue(t tlv, ok bool) uint64 {
	if !ok {
		return 0
	}
	var v uint64
	for _, b := range t.content {
		v = v<<8 | uint64(b)
	}
	return v
}

// parseChargingRecord decodes one BER ChargingRecord body: the subscriber
// identifier (SubscriptionID [2] -> SubscriptionIDData [1]) and the data
// volumes (ListOfMultipleUnitUsage [5] -> UsedUnitContainers [1] ->
// DataVolumeUplink/Downlink [5]/[6]). The optional APN (DNN) is read from
// PDUSessionChargingInformation [13] -> DataNetworkNameIdentifier [13].
// Unknown or malformed optional fields are ignored; only the identifier
// and the volume TLVs matter, so minor free5GC version drift survives.
func parseChargingRecord(body []byte) (cdrUsage, error) {
	var u cdrUsage
	top, err := parseChildren(body)
	if err != nil {
		return u, err
	}
	// The body is the CHFRecord CHOICE: context [1] wrapping the
	// ChargingRecord SET. Be tolerant of the outer wrapper.
	rec, ok := findTag(top, 1)
	if !ok && len(top) == 1 {
		rec = top[0]
	} else if !ok {
		return u, fmt.Errorf("cdr: missing ChargingRecord wrapper [1]")
	}
	fields, err := parseChildren(rec.content)
	if err != nil {
		return u, err
	}
	if sub, ok := findTag(fields, 2); ok { // SubscriberIdentifier
		if data, ok := findChild(sub, 1); ok { // SubscriptionIDData (UTF8String)
			u.imsi = string(data.content)
		}
	}
	if u.imsi == "" {
		return u, fmt.Errorf("cdr: record without subscriber identifier")
	}
	if psi, ok := findTag(fields, 13); ok { // PDUSessionChargingInformation
		if dnn, ok := findChild(psi, 13); ok { // DataNetworkNameIdentifier
			u.apn = string(dnn.content)
		}
	}
	if list, ok := findTag(fields, 5); ok { // ListOfMultipleUnitUsage
		if muus, err := parseChildren(list.content); err == nil {
			for _, muu := range muus { // MultipleUnitUsage (SEQUENCE)
				mFields, err := parseChildren(muu.content)
				if err != nil {
					continue
				}
				containers, ok := findTag(mFields, 1) // UsedUnitContainers
				if !ok {
					continue
				}
				if cFields, err := parseChildren(containers.content); err == nil {
					for _, uc := range cFields { // UsedUnitContainer (SEQUENCE)
						ucFields, err := parseChildren(uc.content)
						if err != nil {
							continue
						}
						u.up += uintValue(findTag(ucFields, 5)) // DataVolumeUplink
						u.dn += uintValue(findTag(ucFields, 6)) // DataVolumeDownlink
					}
				}
			}
		}
	}
	return u, nil
}

// Multi merges the snapshots of several sources into one session table.
// It exists so the free5GC metering pair - the AMF OAM live-session source
// and the CHF CDR usage source - can run as a single collector.
type Multi []Source

// Poll implements Source: it polls every child and merges the sessions,
// keeping one record per UE (richest wins: bytes from the metering source,
// APN/phase from the live-session source).
func (m Multi) Poll(ctx context.Context) ([]api.Session, error) {
	var all []api.Session
	for _, s := range m {
		sessions, err := s.Poll(ctx)
		if err != nil {
			return nil, err
		}
		all = append(all, sessions...)
	}
	return mergeSessions(all), nil
}

// mergeSessions dedupes sessions by IMSI hash, keeping the richest fields.
func mergeSessions(in []api.Session) []api.Session {
	byHash := map[string]*api.Session{}
	for i := range in {
		s := in[i]
		cur, ok := byHash[s.IMSIHash]
		if !ok {
			cp := s
			byHash[s.IMSIHash] = &cp
			continue
		}
		if s.BytesUp > cur.BytesUp {
			cur.BytesUp = s.BytesUp
		}
		if s.BytesDn > cur.BytesDn {
			cur.BytesDn = s.BytesDn
		}
		if cur.APN == "" {
			cur.APN = s.APN
		}
		if cur.IP == "" {
			cur.IP = s.IP
		}
		if cur.Phase == "" || cur.Phase == "metered" {
			cur.Phase = s.Phase
		}
	}
	out := make([]api.Session, 0, len(byHash))
	for _, s := range byHash {
		out = append(out, *s)
	}
	return out
}
