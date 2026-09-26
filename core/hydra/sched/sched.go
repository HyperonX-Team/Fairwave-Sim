// Package sched implements the Hydra delay-inflation scheduler.
//
// Conventional multipath schedulers trade goodput against reordering: a
// low-RTT policy ships packets onto the fastest link and reorders badly;
// a flowlet policy avoids reordering but underuses capacity. The
// delay-inflation scheduler avoids the trade entirely by assigning each
// packet to the thread whose *projected completion time* is earliest,
// given current queue depth, link rate, expected retransmissions from
// loss, and one-way delay. Because it minimises completion time rather
// than RTT, it maximises aggregate goodput while keeping consecutive
// packets close together in time, which in turn keeps the reorder window
// shallow.
package sched

import (
	"errors"
	"math"
	"sort"
)

// ErrNoThread is returned when no candidate can carry a packet.
var ErrNoThread = errors.New("sched: no available thread")

// Thread is one scheduling candidate: an independent SIM/modem/path with
// its current state. QueueBytes is the number of bytes already assigned to
// this thread and not yet fully delivered.
type Thread struct {
	ID         string
	Mbps       float64
	RTTms      float64
	LossPct    float64
	QueueBytes uint64
}

// Projected returns the expected time, in seconds from now, for a packet
// of pktBytes to be fully delivered over t.
//
// service = (queue + packet) / rate / (1 - loss)   (queueing + retransmits)
// delay   = rtt / 2                                 (one-way propagation)
func Projected(t Thread, pktBytes uint64) float64 {
	if t.Mbps <= 0 {
		return math.Inf(1)
	}
	rate := t.Mbps * 1e6 / 8 // bytes per second
	loss := t.LossPct / 100
	switch {
	case loss < 0:
		loss = 0
	case loss > 0.99:
		loss = 0.99
	}
	bytes := float64(t.QueueBytes + pktBytes)
	service := bytes / rate / (1 - loss)
	return service + t.RTTms/2000
}

// Pick returns the thread with the earliest projected completion time.
// Ties are broken by thread ID so scheduling is deterministic.
func Pick(threads []Thread, pktBytes uint64) (Thread, error) {
	if len(threads) == 0 {
		return Thread{}, ErrNoThread
	}
	best := threads[0]
	bestT := Projected(best, pktBytes)
	for _, t := range threads[1:] {
		if pt := Projected(t, pktBytes); pt < bestT || (pt == bestT && t.ID < best.ID) {
			best, bestT = t, pt
		}
	}
	if math.IsInf(bestT, 1) {
		return Thread{}, ErrNoThread
	}
	return best, nil
}

// Rank returns the candidates ordered by ascending projected completion
// time. It is the whole ordering behind Pick, exposed for diagnostics and
// tests.
func Rank(threads []Thread, pktBytes uint64) []Thread {
	out := make([]Thread, len(threads))
	copy(out, threads)
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := Projected(out[i], pktBytes), Projected(out[j], pktBytes)
		if pi == pj {
			return out[i].ID < out[j].ID
		}
		return pi < pj
	})
	return out
}
