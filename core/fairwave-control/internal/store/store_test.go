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

func TestHealthPersist(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	temp := 42.5
	if err := s.UpsertHealth(&api.NodeHealth{NodeID: "n1", Load1: 0.75, SDRTempC: &temp, GPSDO: true}); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open(dir)
	h, ok := s2.GetHealth("n1")
	if !ok || h.Load1 != 0.75 || h.SDRTempC == nil || *h.SDRTempC != 42.5 || !h.GPSDO {
		t.Fatalf("health not persisted: %+v", h)
	}
}

func TestAuditAppendOnly(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	_ = s.AppendAudit(api.AuditEntry{ID: "a1", Action: "tx_arm"})
	_ = s.AppendAudit(api.AuditEntry{ID: "a2", Action: "sim_revoke"})
	s2, _ := Open(dir)
	entries := s2.ListAudit()
	if len(entries) != 2 {
		t.Fatalf("audit len = %d, want 2", len(entries))
	}
	// most recent first
	if entries[0].Action != "sim_revoke" || entries[1].Action != "tx_arm" {
		t.Fatalf("audit order: %+v", entries)
	}
}

func TestTxStatePersist(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	if s.GetTxArmed() {
		t.Fatal("fresh store must start disarmed")
	}
	if err := s.SetTxArmed(true); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open(dir)
	if !s2.GetTxArmed() {
		t.Fatal("tx armed state not persisted")
	}
}

func TestSessionsReplace(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	if err := s.ReplaceSessions([]api.Session{{
		IMSIHash: api.HashIMSI("999991234567001"), APN: "internet", Phase: "active",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceSessions(nil); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open(dir)
	if n := len(s2.ListSessions()); n != 0 {
		t.Fatalf("sessions after replace-nil = %d, want 0", n)
	}
}

func TestTokensPersistAndLookup(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	tok := &api.Token{ID: "tok-1", Name: "ops", Role: api.RoleOperator, TokenHash: "abc123"}
	if err := s.UpsertToken(tok); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open(dir)
	got, ok := s2.TokenByHash("abc123")
	if !ok || got.Name != "ops" || got.Role != api.RoleOperator {
		t.Fatalf("token lookup: %+v %v", got, ok)
	}
	if err := s2.RevokeToken("tok-1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s2.TokenByHash("abc123"); ok {
		t.Fatal("revoked token must not authenticate")
	}
}

func TestUsagePersist(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	_ = s.UpsertUsage(&api.SimUsage{IMSI: "999991234567001", BytesUp: 10, BytesDn: 5, QuotaBytes: 100})
	s2, _ := Open(dir)
	u, ok := s2.GetUsage("999991234567001")
	if !ok || u.BytesUp != 10 || u.QuotaBytes != 100 {
		t.Fatalf("usage: %+v", u)
	}
}

func TestAlertsPersistAndResolve(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	now := time.Now().UTC()
	a := &api.Alert{ID: "al-1", Key: "node-down:n1", Severity: api.AlertCritical, Message: "down", TS: now}
	if err := s.AppendAlert(a); err != nil {
		t.Fatal(err)
	}
	s2, _ := Open(dir)
	if n := len(s2.ActiveAlerts()); n != 1 {
		t.Fatalf("active alerts = %d, want 1", n)
	}
	if err := s2.ResolveAlert("al-1", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if n := len(s2.ActiveAlerts()); n != 0 {
		t.Fatalf("active alerts after resolve = %d, want 0", n)
	}
}
