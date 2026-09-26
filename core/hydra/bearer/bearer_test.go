package bearer

import "testing"

func TestUpsertGetList(t *testing.T) {
	r := NewRegistry()
	if err := r.Upsert(Thread{ID: "b", Mbps: 200, Up: true}); err != nil {
		t.Fatal(err)
	}
	if err := r.Upsert(Thread{ID: "a", Mbps: 100, Up: true}); err != nil {
		t.Fatal(err)
	}
	got, ok := r.Get("a")
	if !ok || got.Mbps != 100 {
		t.Fatalf("get a: %+v %v", got, ok)
	}
	list := r.List()
	if len(list) != 2 || list[0].ID != "a" || list[1].ID != "b" {
		t.Fatalf("list not sorted: %+v", list)
	}
}

func TestUpsertValidation(t *testing.T) {
	r := NewRegistry()
	cases := []Thread{
		{ID: ""},
		{ID: "x", Mbps: -1},
		{ID: "x", RTTms: -1},
		{ID: "x", LossPct: -1},
		{ID: "x", LossPct: 101},
	}
	for i, c := range cases {
		if err := r.Upsert(c); err == nil {
			t.Fatalf("case %d: expected error for %+v", i, c)
		}
	}
}

func TestRemove(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(Thread{ID: "a", Mbps: 1})
	if !r.Remove("a") {
		t.Fatal("remove existing returned false")
	}
	if r.Remove("a") {
		t.Fatal("remove missing returned true")
	}
}

func TestAggregateMbpsCountsOnlyUp(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(Thread{ID: "a", Mbps: 300, Up: true})
	_ = r.Upsert(Thread{ID: "b", Mbps: 500, Up: false})
	_ = r.Upsert(Thread{ID: "c", Mbps: 200, Up: true})
	if got := r.AggregateMbps(); got != 500 {
		t.Fatalf("aggregate = %v want 500", got)
	}
	if len(r.Up()) != 2 {
		t.Fatalf("up = %d want 2", len(r.Up()))
	}
}

func TestCandidatesSkipsDownAndUnknown(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(Thread{ID: "up", Mbps: 100, Up: true})
	_ = r.Upsert(Thread{ID: "down", Mbps: 900, Up: false})
	cands := r.Candidates([]string{"up", "down", "ghost"})
	if len(cands) != 1 || cands[0].ID != "up" {
		t.Fatalf("candidates = %+v", cands)
	}
}

func TestQueueAccounting(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(Thread{ID: "a", Mbps: 100})
	r.AddQueue("a", 1000)
	r.AddQueue("a", 500)
	if got, _ := r.Get("a"); got.QueueBytes != 1500 {
		t.Fatalf("queue = %d want 1500", got.QueueBytes)
	}
	r.SubQueue("a", 400)
	if got, _ := r.Get("a"); got.QueueBytes != 1100 {
		t.Fatalf("queue = %d want 1100", got.QueueBytes)
	}
	// Floor at zero, and ignore unknown ids.
	r.SubQueue("a", 99999)
	if got, _ := r.Get("a"); got.QueueBytes != 0 {
		t.Fatalf("queue floor = %d want 0", got.QueueBytes)
	}
	r.AddQueue("ghost", 10)
	r.SubQueue("ghost", 10)
}

func TestUpsertStampsLastSeen(t *testing.T) {
	r := NewRegistry()
	_ = r.Upsert(Thread{ID: "a", Mbps: 1})
	got, _ := r.Get("a")
	if got.LastSeen.IsZero() {
		t.Fatal("LastSeen not stamped")
	}
}
