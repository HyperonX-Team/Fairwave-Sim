// Package api defines the public API types shared by fairwave-control,
// fairwave-agent, and fairwave-cli. Keep this package free of side effects.
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// LifecyclePhase is the node lifecycle state machine step.
type LifecyclePhase string

const (
	PhaseProvision LifecyclePhase = "provision"
	PhaseRegister  LifecyclePhase = "register"
	PhaseOnAir     LifecyclePhase = "on-air"
	PhasePeer      LifecyclePhase = "peer"
	PhaseBreakout  LifecyclePhase = "breakout"
)

// AllPhases is the valid transition order (indexes are the canonical order).
var AllPhases = []LifecyclePhase{PhaseProvision, PhaseRegister, PhaseOnAir, PhasePeer, PhaseBreakout}

// Node describes one Fairwave pizza box.
type Node struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Role      string         `json:"role"` // "edge" | "hub" | "lab"
	Country   string         `json:"country"`
	Phase     LifecyclePhase `json:"phase"`
	TxArmed   bool           `json:"tx_armed"`
	Version   string         `json:"version"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// SIM is the operator-facing view of a provisioned subscriber identity.
// Ki/OPc are never returned over the API; they live in the vault only.
type SIM struct {
	IMSI       string    `json:"imsi"`
	MSISDN     string    `json:"msisdn"`
	Profile    string    `json:"profile"` // "lab" | "prod"
	Status     string    `json:"status"`  // "issued" | "active" | "suspended" | "revoked" | "expired"
	APN        string    `json:"apn"`
	SQN        string    `json:"sqn,omitempty"`
	QuotaBytes uint64    `json:"quota_bytes,omitempty"` // 0 = unlimited
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Peer is a neighboring Fairwave box reachable over the mesh.
type Peer struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Endpoint   string    `json:"endpoint"`    // host:port
	PubKey     string    `json:"pub_key"`     // WireGuard public key
	AllowedIPs []string  `json:"allowed_ips"` // UE pools / prefixes this peer serves
	Status     string    `json:"status"`      // "pending" | "up" | "down"
	LastSeen   time.Time `json:"last_seen"`
}

// Session is a UE data session (privacy-preserving: IMSI hashed at rest).
type Session struct {
	IMSIHash string    `json:"imsi_hash"`
	APN      string    `json:"apn"`
	IP       string    `json:"ip"`
	Phase    string    `json:"phase"`
	BytesUp  uint64    `json:"bytes_up"`
	BytesDn  uint64    `json:"bytes_dn"`
	Created  time.Time `json:"created"`
}

// HashIMSI returns the privacy-preserving sha256 hex digest used for
// session records. The full IMSI never leaves the subscriber store; UE
// tables and the dashboard only ever see this hash.
func HashIMSI(imsi string) string {
	sum := sha256.Sum256([]byte(imsi))
	return hex.EncodeToString(sum[:])
}

// NodeHealth is a heartbeat payload reported by the on-box agent to
// /v1/telemetry. Up is computed by the control plane from recency.
type NodeHealth struct {
	NodeID    string    `json:"node_id"`
	TS        time.Time `json:"ts"`
	Mode      string    `json:"mode"`
	Load1     float64   `json:"load1"`
	SDRTempC  *float64  `json:"sdr_temp_c,omitempty"`
	GPSDO     bool      `json:"gpsdo_locked"`
	RFArmed   bool      `json:"rf_armed"`
	Watchdog  string    `json:"watchdog"`
	Platform  string    `json:"platform"`
	Radio     string    `json:"radio"`
	FreqCheck bool      `json:"freq_plan_ok"`
	Up        bool      `json:"up"`
}

// AuditEntry is one append-only record of an operator action (TX arming,
// SIM lifecycle, peer changes, ...). The audit log never rotates or
// rewrites history; it is the regulatory trail behind the spectrum gates.
type AuditEntry struct {
	ID        string    `json:"id"`
	TS        time.Time `json:"ts"`
	Action    string    `json:"action"`
	Target    string    `json:"target,omitempty"`
	Detail    string    `json:"detail,omitempty"`
	Principal string    `json:"principal,omitempty"` // token name / "admin" / "system"
}

// TokenRole is the least-privilege tier of a scoped API token.
type TokenRole string

const (
	RoleAdmin    TokenRole = "admin"    // everything, incl. tokens/backup/TX/compliance
	RoleOperator TokenRole = "operator" // mutating SIM/eSIM/peer/policy/lifecycle ops
	RoleViewer   TokenRole = "viewer"   // read-only
)

// Valid reports whether r is a known role.
func (r TokenRole) Valid() bool {
	switch r {
	case RoleAdmin, RoleOperator, RoleViewer:
		return true
	}
	return false
}

// Token is a scoped API token. Only the SHA-256 hash is stored; the raw
// secret is returned once at creation.
type Token struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      TokenRole `json:"role"`
	TokenHash string    `json:"token_hash"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
}

// TokenCreateRequest mints a scoped token.
type TokenCreateRequest struct {
	Name string    `json:"name"`
	Role TokenRole `json:"role"`
}

// TokenCreateResponse returns the secret exactly once.
type TokenCreateResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      TokenRole `json:"role"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
}

// AlertSeverity ranks an operational alert.
type AlertSeverity string

const (
	AlertCritical AlertSeverity = "critical"
	AlertWarning  AlertSeverity = "warning"
	AlertInfo     AlertSeverity = "info"
)

// Alert is one fired (or resolved) operational condition.
type Alert struct {
	ID         string        `json:"id"`
	Key        string        `json:"key"` // stable dedup key, e.g. "sdr-temp:node-1"
	Severity   AlertSeverity `json:"severity"`
	Message    string        `json:"message"`
	Target     string        `json:"target,omitempty"`
	Node       string        `json:"node,omitempty"`
	TS         time.Time     `json:"ts"`
	Resolved   bool          `json:"resolved"`
	ResolvedAt *time.Time    `json:"resolved_at,omitempty"`
}

// SimUsage is the accumulated data usage for one SIM.
type SimUsage struct {
	IMSI       string    `json:"imsi"`
	IMSIHash   string    `json:"imsi_hash"`
	BytesUp    uint64    `json:"bytes_up"`
	BytesDn    uint64    `json:"bytes_dn"`
	QuotaBytes uint64    `json:"quota_bytes"` // 0 = unlimited
	UpdatedAt  time.Time `json:"updated_at"`
}

// SimQuotaRequest sets the fair-use data allowance for a SIM (0 = unlimited).
type SimQuotaRequest struct {
	QuotaBytes uint64 `json:"quota_bytes"`
}

// SimUsageRequest reconciles a SIM's usage totals (operator-provided, e.g.
// from a richer metering source). Values are added to the accumulated totals.
type SimUsageRequest struct {
	BytesUp uint64 `json:"bytes_up"`
	BytesDn uint64 `json:"bytes_dn"`
}

// SimImportItem is one subscriber row imported from a bureau batch.
// Credentials are never part of an import: Ki/OPc stay in the bureau's
// vault and the HSS is only seeded when a known lab vector matches.
type SimImportItem struct {
	IMSI      string    `json:"imsi"`
	MSISDN    string    `json:"msisdn"`
	Profile   string    `json:"profile"` // "lab" | "prod"
	APN       string    `json:"apn"`
	Status    string    `json:"status"` // default "issued"
	ExpiresAt time.Time `json:"expires_at"`
}

// SimImportRequest carries a bureau batch.
type SimImportRequest struct {
	Sims []SimImportItem `json:"sims"`
}

// SimImportResponse reports what the import did.
type SimImportResponse struct {
	Imported int      `json:"imported"`
	Updated  int      `json:"updated"`
	Skipped  []string `json:"skipped,omitempty"`
}

// EsimIssueRequest mints an eSIM profile for an existing SIM.
type EsimIssueRequest struct {
	IMSI    string `json:"imsi"`
	Address string `json:"address,omitempty"` // SM-DP+ address in the activation code
	EID     string `json:"eid,omitempty"`     // pin to a target eUICC (optional)
}

// EsimIssueResponse carries the activation code and, optionally, the QR
// PNG payload so a CLI can render the code without a second round trip.
type EsimIssueResponse struct {
	IMSI           string `json:"imsi"`
	ICCID          string `json:"iccid"`
	ActivationCode string `json:"activation_code"`
	SMDPAddress    string `json:"smdp_address"`
	QRPNGBase64    string `json:"qr_png_base64,omitempty"`
}

// EsimCode is the operator-facing view of a registered activation code.
type EsimCode struct {
	ActivationCode string     `json:"activation_code"`
	SMDPAddress    string     `json:"smdp_address"`
	IMSI           string     `json:"imsi"`
	ICCID          string     `json:"iccid"`
	ProfileName    string     `json:"profile_name"`
	EID            string     `json:"eid,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	DownloadedAt   *time.Time `json:"downloaded_at,omitempty"`
	ExpiresAt      time.Time  `json:"expires_at"`
}

// EsimRevokeRequest revokes a previously issued activation code.
type EsimRevokeRequest struct {
	ActivationCode string `json:"activation_code"`
}

// Policy is the operator-controlled routing/QoS policy.
type Policy struct {
	LocalBreakout bool     `json:"local_breakout"` // edge NAT by default
	HubPeer       string   `json:"hub_peer"`       // optional hub for off-site traffic
	MaxUEs        int      `json:"max_ues"`        // fair-use cap per box
	APNs          []string `json:"apns"`           // allowed APNs
	QoSDLMbps     int      `json:"qos_dl_mbps"`    // default DL cap per UE
	QoSULMbps     int      `json:"qos_ul_mbps"`
}

// Status is the /v1/status payload.
type Status struct {
	Version   string `json:"version"`
	Mode      string `json:"mode"` // "lab" | "rf"
	Phase     string `json:"phase"`
	TxArmed   bool   `json:"tx_armed"`
	Country   string `json:"country"`
	Nodes     int    `json:"nodes"`
	UEs       int    `json:"ues"`
	Peers     int    `json:"peers"`
	UptimeSec int64  `json:"uptime_sec"`
}

// SpectrumCheckRequest is the input to /v1/spectrum/check.
type SpectrumCheckRequest struct {
	Country    string `json:"country"`
	Band       string `json:"band"` // e.g. "n48", "b3"
	Indoor     bool   `json:"indoor"`
	LicenseRef string `json:"license_ref,omitempty"`
}

// SpectrumCheckResponse explains whether a TX configuration is allowed.
type SpectrumCheckResponse struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons"`
}

// TxArmRequest arms TX. Acknowledgment must match the exact gate phrase.
type TxArmRequest struct {
	Country        string `json:"country"`
	Band           string `json:"band"`
	Acknowledgment string `json:"acknowledgment"`
	LicenseRef     string `json:"license_ref"`
}

// TxArmResponse is the result of the TX arming gate.
type TxArmResponse struct {
	Armed   bool     `json:"armed"`
	Reasons []string `json:"reasons"`
}

// SimIssueRequest mints lab or prod SIM records.
type SimIssueRequest struct {
	Profile string `json:"profile"` // "lab" | "prod"
	Prefix  string `json:"prefix"`  // IMSI prefix, defaults to profile PLMN
	Count   int    `json:"count"`
}

// EnrollRequest carries a bootstrap token for node enrollment.
type EnrollRequest struct {
	BootstrapToken string `json:"bootstrap_token"`
}

// LifecycleTransitionRequest requests a phase transition.
type LifecycleTransitionRequest struct {
	Phase LifecyclePhase `json:"phase"`
}

// ErrorBody is the canonical API error envelope.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail carries a machine code and human message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
