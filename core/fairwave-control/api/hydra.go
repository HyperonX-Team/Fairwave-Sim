package api

// This file holds the Hydra (cooperative bearer multiplexing) northbound
// wire shapes: the CLI, UI, and lab bench use them to manage threads and
// weaves. See docs/architecture/hydra.md.

// HydraThreadRequest adds or updates one bearer thread.
type HydraThreadRequest struct {
	ID      string  `json:"id"`
	Box     string  `json:"box,omitempty"`
	SIM     string  `json:"sim,omitempty"`
	Mbps    float64 `json:"mbps"`
	RTTms   float64 `json:"rtt_ms"`
	LossPct float64 `json:"loss_pct"`
	Up      bool    `json:"up"`
}

// HydraWeaveRequest binds a set of threads to an anchor.
type HydraWeaveRequest struct {
	ID      string   `json:"id"`
	Anchor  string   `json:"anchor,omitempty"`
	Threads []string `json:"threads"`
}

// HydraStripRequest asks the engine to assign one payload to a thread.
type HydraStripRequest struct {
	PayloadB64 string `json:"payload_b64"`
}

// HydraStripResponse is the scheduler's assignment, including the framed
// packet so the lab can feed it back through ingest.
type HydraStripResponse struct {
	WeaveID  string `json:"weave_id"`
	ThreadID string `json:"thread_id"`
	Seq      uint64 `json:"seq"`
	FrameLen int    `json:"frame_len"`
	FrameB64 string `json:"frame_b64"`
}

// HydraIngestRequest feeds one framed packet back into reassembly.
type HydraIngestRequest struct {
	FrameB64 string `json:"frame_b64"`
}

// HydraIngestResponse reports how many payloads became deliverable.
type HydraIngestResponse struct {
	Delivered int `json:"delivered"`
}

// HydraBenchRequest runs a synthetic stripe/reassemble bench.
type HydraBenchRequest struct {
	Packets  int `json:"packets"`
	PktBytes int `json:"pkt_bytes"`
}

// HydraThreadHealth is a running data-plane node's measured view of one
// thread, reported back to the control plane so the operator sees live
// link health rather than only the declared values.
type HydraThreadHealth struct {
	Mbps    float64 `json:"mbps"`
	RTTms   float64 `json:"rtt_ms"`
	LossPct float64 `json:"loss_pct"`
	Up      bool    `json:"up"`
}
