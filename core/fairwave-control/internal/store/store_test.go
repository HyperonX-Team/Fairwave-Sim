package store

import (
	"testing"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
)

func TestPersistRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := &api.Node{ID: "n1", Name: "cafe", Country: "LAB", Phase: api.PhaseProvision, CreatedAt: time.Now()}
	if err := s.UpsertNode(n); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertSIM(&api.SIM{IMSI: "999991234567001", Status: "issued"}); err != nil {
		t.Fatal(err)
	}

	// reopen
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s2.GetNode("n1")
	if !ok || got.Name != "cafe" {
		t.Fatalf("node not persisted: %+v", got)
	}
	if _, ok := s2.GetSIM("999991234567001"); !ok {
		t.Fatal("sim not persisted")
	}
}

func TestPolicyPersist(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	if err := s.SetPolicy(&api.Policy{LocalBreakout: false, MaxUEs: 32, APNs: []string{"internet"}}); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open(dir)
	p := s2.GetPolicy()
	if p.LocalBreakout || p.MaxUEs != 32 {
		t.Fatalf("policy: %+v", p)
	}
}

func TestDeleteNode(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	_ = s.UpsertNode(&api.Node{ID: "gone"})
	if err := s.DeleteNode("gone"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.GetNode("gone"); ok {
		t.Fatal("node should be gone")
	}
}
