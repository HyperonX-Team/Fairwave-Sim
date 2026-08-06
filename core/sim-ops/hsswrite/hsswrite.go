// Package hsswrite pushes issued SIM credentials into the Open5GS HSS so a
// freshly provisioned SIM (physical or eSIM) can attach without manual
// seeding. The credentials never cross the network: the drivers execute
// commands inside the local containers (docker exec), exactly like the
// reference seed script core/open5gs/hss-init.sh.
//
// ADR-0006 rules still apply: the write-back runs on the node, the caller
// supplies the credentials from the vault/provisioner, and nothing is ever
// logged. The process list briefly contains the Ki/OPc (docker exec
// argv) - acceptable in the lab, documented in docs/sim-lifecycle.
package hsswrite

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

// Writer pushes subscriber records into the HSS.
type Writer interface {
	// Add upserts the subscriber (IMSI, Ki/OPc, APN) into the HSS.
	Add(ctx context.Context, sub simprov.Subscriber) error
	// Remove deletes the subscriber by IMSI (idempotent).
	Remove(ctx context.Context, imsi string) error
}

// runner is the exec seam, injectable for tests.
type runner func(ctx context.Context, name string, args ...string) ([]byte, error)

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Driver names understood by New.
const (
	DriverMongosh = "mongosh" // upsert document via mongosh in the mongo container
	DriverDBCTL   = "dbctl"   // open5gs-dbctl add/remove in the open5gs container
	DriverNone    = "none"
)

// DefaultDBURI is the Open5GS subscriber database URI used by the lab
// compose (matches core/open5gs/hss-init.sh).
const DefaultDBURI = "mongodb://localhost:27017/open5gs"

// New builds the writer for a driver name. Unknown or empty drivers return
// the safe no-op (None).
func New(driver, container string) Writer {
	switch driver {
	case DriverMongosh:
		return &Mongosh{Container: container, DBURI: DefaultDBURI, run: execRunner}
	case DriverDBCTL:
		return &DBCTL{Container: container, run: execRunner}
	default:
		return None{}
	}
}

// None is a safe no-op writer (default when no driver is configured).
type None struct{}

// Add implements Writer.
func (None) Add(context.Context, simprov.Subscriber) error { return nil }

// Remove implements Writer.
func (None) Remove(context.Context, string) error { return nil }

// Mongosh upserts the full subscriber document (the hss-init.sh shape)
// via mongosh inside the mongo container.
type Mongosh struct {
	Container string
	DBURI     string
	run       runner
}

// Add implements Writer: db.subscribers.updateOne(... upsert).
func (m *Mongosh) Add(ctx context.Context, sub simprov.Subscriber) error {
	doc := map[string]any{
		"imsi":                     sub.IMSI,
		"msisdn":                   []string{sub.MSISDN},
		"access_restriction_data":  32,
		"network_access_mode":      0,
		"subscriber_status":        0,
		"subscribed_rau_tau_timer": 12,
		"security":                 map[string]any{"k": sub.Ki, "opc": sub.OPc, "amf": sub.AMF, "sqn": sub.SQN},
		"apn_list": []any{map[string]any{
			"apn": sub.APN,
			"qos": map[string]any{"class_id": 9, "priority_level": 8, "preemption_vulnerability": 1, "preemption_capability": 1},
			"ambr": map[string]any{
				"uplink":   map[string]any{"value": 1, "unit": 8},
				"downlink": map[string]any{"value": 1, "unit": 8},
			},
			"type": 0,
		}},
		"slice": []any{map[string]any{
			"sst":               1,
			"default_indicator": true,
			"session":           []any{map[string]any{"name": sub.APN, "type": 3}},
		}},
		"schema_version": 1,
		"__v":            0,
	}
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("hsswrite: build document: %w", err)
	}
	eval := fmt.Sprintf(`db.subscribers.updateOne({imsi: "%s"}, {$set: %s}, {upsert: true})`, sub.IMSI, docJSON)
	out, err := m.run(ctx, "docker", "exec", m.Container, "mongosh", "--quiet", m.DBURI, "--eval", eval)
	if err != nil {
		return fmt.Errorf("hsswrite: mongosh add %s: %w: %s", sub.IMSI, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove implements Writer: db.subscribers.deleteOne(...).
func (m *Mongosh) Remove(ctx context.Context, imsi string) error {
	eval := fmt.Sprintf(`db.subscribers.deleteOne({imsi: "%s"})`, imsi)
	out, err := m.run(ctx, "docker", "exec", m.Container, "mongosh", "--quiet", m.DBURI, "--eval", eval)
	if err != nil {
		return fmt.Errorf("hsswrite: mongosh remove %s: %w: %s", imsi, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// DBCTL wraps open5gs-dbctl inside the open5gs container.
type DBCTL struct {
	Container string
	run       runner
}

// Add implements Writer: open5gs-dbctl add <imsi> <ki> <opc>.
func (d *DBCTL) Add(ctx context.Context, sub simprov.Subscriber) error {
	out, err := d.run(ctx, "docker", "exec", d.Container, "open5gs-dbctl", "add", sub.IMSI, sub.Ki, sub.OPc)
	if err != nil {
		return fmt.Errorf("hsswrite: dbctl add %s: %w: %s", sub.IMSI, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove implements Writer: open5gs-dbctl remove <imsi>.
func (d *DBCTL) Remove(ctx context.Context, imsi string) error {
	out, err := d.run(ctx, "docker", "exec", d.Container, "open5gs-dbctl", "remove", imsi)
	if err != nil {
		return fmt.Errorf("hsswrite: dbctl remove %s: %w: %s", imsi, err, strings.TrimSpace(string(out)))
	}
	return nil
}
