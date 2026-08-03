---
title: fairwave-cli
---

# fairwave-cli

`fairwave-cli` is the operator's command surface against `fairwave-control` REST API v1. It talks JSON over the local API; it never touches core/RAN config directly.

## Command Reference

| Command | Action |
|---|---|
| `fairwave node init` | create identity + mesh CA on this box |
| `fairwave node status` | lifecycle phase, subsystems, agent heartbeat |
| `fairwave node join <bootstrap-token>` | enroll into a neighborhood (mTLS) |
| `fairwave node leave` | leave mesh, revoke own cert |
| `fairwave sim issue --count N --profile lab` | issue N SIMs (IMSI 15 digits, Ki/OPc to vault) |
| `fairwave sim revoke <sim-id>` | revoke a SIM (HSS purge, vault flag) |
| `fairwave peer list` | mesh peers, link state, route table |
| `fairwave spectrum check --band 3 --earfcn 1200` | band/EARFCN pre-flight vs policy |
| `fairwave doctor` | full self-check: API, store, SDR, sync, gate state |
| `fairwave version` | release string + build metadata |

Global flags: `--endpoint` (default `http://127.0.0.1:8080`), `--output json|table` (default table).

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

## Related

- API surface: `docs/architecture/control-plane.md`
- SIM model: `docs/architecture/mobile-core.md`
- Legal gates: `docs/spectrum-and-law/index.md`
