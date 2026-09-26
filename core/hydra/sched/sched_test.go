package sched

import (
	"errors"
	"math"
	"testing"
)

func TestProjectedQueueing(t *testing.T) {
	// 100 Mbps, no loss, no RTT, 1 MB queued + 1 MB packet = 2 MB.
	// 2_000_000 bytes * 8 / 100e6 = 0.16 s.
	got := Projected(Thread{ID: "a", Mbps: 100, QueueBytes: 1_000_000}, 1_000_000)
	if math.Abs(got-0.16) > 1e-9 {
		t.Fatalf("projected = %v want 0.16", got)
	}
}

func TestProjectedLossInflation(t *testing.T) {
	base := Projected(Thread{ID: "a", Mbps: 100}, 1_000_000)
	lossy := Projected(Thread{ID: "a", Mbps: 100, LossPct: 50}, 1_000_000)
	if math.Abs(lossy-2*base) > 1e-9 {
		t.Fatalf("lossy = %v want %v (2x)", lossy, base)
	}
}

func TestProjectedRTT(t *testing.T) {
	got := Projected(Thread{ID: "a", Mbps: 1000, RTTms: 40}, 0)
	if math.Abs(got-0.02) > 1e-9 {
		t.Fatalf("projected = %v want 0.02 (rtt/2)", got)
	}
}

func TestProjectedZeroRateIsInfinite(t *testing.T) {
	if !math.IsInf(Projected(Thread{ID: "a", Mbps: 0}, 100), 1) {
		t.Fatal("zero-rate thread must be unusable")
	}
}

func TestPickFastestWhenIdle(t *testing.T) {
	threads := []Thread{
		{ID: "slow", Mbps: 50, RTTms: 20},
		{ID: "fast", Mbps: 300, RTTms: 20},
		{ID: "medium", Mbps: 150, RTTms: 20},
	}
	got, err := Pick(threads, 1400)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "fast" {
		t.Fatalf("picked %s want fast", got.ID)
	}
}

func TestPickAvoidsCongestedFastLink(t *testing.T) {
	// The fast link is buried under a big queue; the medium link wins.
	threads := []Thread{
		{ID: "fast", Mbps: 300, RTTms: 20, QueueBytes: 10_000_000},
		{ID: "medium", Mbps: 150, RTTms: 20, QueueBytes: 0},
	}
	got, err := Pick(threads, 1400)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "medium" {
		t.Fatalf("picked %s want medium", got.ID)
	}
}

func TestPickPrefersLowRTTOnEqualRate(t *testing.T) {
	threads := []Thread{
		{ID: "far", Mbps: 100, RTTms: 80},
		{ID: "near", Mbps: 100, RTTms: 10},
	}
	got, _ := Pick(threads, 1400)
	if got.ID != "near" {
		t.Fatalf("picked %s want near", got.ID)
	}
}

func TestPickNoThreads(t *testing.T) {
	if _, err := Pick(nil, 1400); !errors.Is(err, ErrNoThread) {
		t.Fatalf("want ErrNoThread, got %v", err)
	}
}

func TestPickAllZeroRate(t *testing.T) {
	if _, err := Pick([]Thread{{ID: "dead", Mbps: 0}}, 1400); !errors.Is(err, ErrNoThread) {
		t.Fatalf("want ErrNoThread, got %v", err)
	}
}

func TestRankOrdering(t *testing.T) {
	threads := []Thread{
		{ID: "c", Mbps: 10},
		{ID: "a", Mbps: 300},
		{ID: "b", Mbps: 100},
	}
	got := Rank(threads, 1400)
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("rank[%d] = %s want %s", i, got[i].ID, want[i])
		}
	}
}

func TestDeterministicTieBreak(t *testing.T) {
	threads := []Thread{
		{ID: "z", Mbps: 100, RTTms: 20},
		{ID: "a", Mbps: 100, RTTms: 20},
	}
	got, _ := Pick(threads, 1400)
	if got.ID != "a" {
		t.Fatalf("tie break picked %s want a", got.ID)
	}
}
