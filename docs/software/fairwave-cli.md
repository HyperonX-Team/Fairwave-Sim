---
title: fairwave-cli
---

# fairwave-cli

`fairwave-cli` is the operator's command surface against `fairwave-control` REST API v1. It talks JSON over the local API; it never touches core/RAN config directly.

## Command Reference

| Command | Action |
|---|---|
| `fairwave node init [--name --country --role]` | create a node record on the control plane |
| `fairwave node status` | control-plane + node status (phase, TX gate, UE/peer counts) |
| `fairwave node health` | per-node agent heartbeats (load, radio, GPSDO, watchdog) |
| `fairwave node join <endpoint>` | join a neighboring box's mesh (peering MVP) |
| `fairwave node leave` | decommission node(s) |
| `fairwave sim issue --count N --profile lab` | issue N SIMs (IMSI 15 digits, Ki/OPc to vault) |
| `fairwave sim revoke <imsi>` | revoke a SIM (HSS purge, vault flag) |
| `fairwave sim suspend <imsi>` / `resume <imsi>` | suspend/resume a SIM (fair-use enforcement) |
| `fairwave sim get <imsi>` / `list` | show one SIM / list all SIMs |
| `fairwave sim export --format csv|json [--lab-creds]` | bureau batch (Ki/OPc merged from vault only) |
| `fairwave sim import <file>` | import a bureau batch (CSV/JSON, never Ki/OPc) |
| `fairwave sim quota <imsi> --bytes N` | set a fair-use data allowance (0 = unlimited) |
| `fairwave sim usage <imsi>` | accumulated up/down bytes vs quota |
| `fairwave esim issue --imsi ... [--qr]` | mint an eSIM profile + activation code (lab vectors) |
| `fairwave esim list` / `revoke <code>` | activation-code registry |
| `fairwave esim serve` | run the lab SM-DP+ server (ES9+) |
| `fairwave peer list` / `add <name> <endpoint>` | mesh peers |
| `fairwave spectrum check --country --band` | band/country pre-flight vs policy |
| `fairwave spectrum arm` / `disarm` | arm/clear the TX gate (arm requires acknowledgment; lab refuses) |
| `fairwave policy get` / `set` | routing/QoS policy (local breakout, max UEs, QoS caps) |
| `fairwave compliance export` | regulator-ready compliance report (CSV) |
| `fairwave backup` / `restore` | full state backup (optionally AES-256-GCM encrypted) / restore |
| `fairwave token create|list|revoke` | scoped API tokens (admin) |
| `fairwave alerts [--watch]` | operational alerts (active first, poll mode) |
| `fairwave audit [--limit N]` | append-only operator audit trail |
| `fairwave doctor` | full self-check: API, store, SDR, sync, gate state |
| `fairwave version` | release string + build metadata |

Global flags: `--control` (default `http://localhost:8080`, env `FAIRWAVE_CONTROL`),
`--token` (admin bearer token or `FAIRWAVE_ADMIN_TOKEN`), `--data-dir` (default `./data`;
holds the admin token file and the standalone eSIM registry).

## Example Outputs

```text
$ fairwave node status
Node:      cafe-01
Phase:     on-air
PLMN:      999-99   TAC: 7
Core:      open5gs 5.0  UP (5/5 services)
RAN:       srsran eNB UP  (zmq)
TX gate:   NOT ARMED (lab mode)
Agent:     heartbeat 2s ago, 51C, NTP +1.1ms
```

```text
$ fairwave sim issue --count 2 --profile lab
SIM  sim-8f3c1  IMSI 999991234567890  profile lab  ACTIVE
SIM  sim-9d21b  IMSI 999991234567891  profile lab  ACTIVE
Ki/OPc written to SIM vault; never printed.
```

```text
$ fairwave peer list
PEER       STATE   WIREGUARD   ROUTES
cafe-02    UP      est. 2m     10.60.0.0/24 via cafe-02
lab-box    UP      est. 5d     10.60.1.0/24 via lab-box
```

```text
$ fairwave spectrum check --band 3 --earfcn 1200
Band 3 EARFCN 1200 (20 MHz): ALLOWED for region profile "lab"
EIRP cap (policy): not set -> gate will not arm until set.
```

## Rules for the CLI

- It never prints Ki/OPc, bootstrap tokens, or private keys (`doctor` redacts).
- Destructive commands (`node leave`, `sim revoke`) require `--yes` confirmation.
- Exit codes: 0 success, 1 failure, 2 usage error, 3 policy/legal-block (gate refuses).
- IMSIs in output are full-length for operator convenience; the store only keeps hashes.

## Usage metering workflow (fair-use caps)

```bash
fairwave sim issue --count 1 --profile lab
fairwave sim quota 999991234567001 --bytes 1073741824   # 1 GiB allowance
fairwave sim usage 999991234567001                      # up=... dn=... total=... quota=...
```

Usage comes from the core: the 4G GTP-U tap or (in 5G mode) the free5GC CHF CDR files
(`core: free5gc` + `free5gc.cdr_dir`). Exceeding the quota auto-suspends the SIM and
writes an audit entry; `sim resume` brings it back.

## Related

- API surface: `docs/architecture/control-plane.md`
- SIM model: `docs/architecture/mobile-core.md`
- Legal gates: `docs/spectrum-and-law/index.md`
