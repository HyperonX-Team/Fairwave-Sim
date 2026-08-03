// Package api defines the public API types shared by fairwave-control,
// fairwave-agent, and fairwave-cli. Keep this package free of side effects.
package api

import "time"

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
	IMSI      string    `json:"imsi"`
	MSISDN    string    `json:"msisdn"`
	Profile   string    `json:"profile"` // "lab" | "prod"
	Status    string    `json:"status"`  // "issued" | "active" | "revoked" | "expired"
	APN       string    `json:"apn"`
	SQN       string    `json:"sqn,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
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
