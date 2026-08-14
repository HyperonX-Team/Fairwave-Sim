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
| `fairwave node health` | per-node agent health (heartbeats) |
| `fairwave node join <endpoint>` | join a neighboring box's mesh (not implemented yet - M2; errors) |
| `fairwave node leave [--yes]` | decommission node(s); requires `--yes` (or TTY confirmation) |
| `fairwave sim issue --count N --profile lab` | issue N SIMs (IMSI 15 digits, Ki/OPc to vault) |
| `fairwave sim revoke <imsi>` | revoke a SIM (HSS purge, vault flag) |
| `fairwave sim suspend <imsi>` / `resume <imsi>` | suspend/resume a SIM (fair-use enforcement) |
| `fairwave sim get <imsi>` / `list` | show one SIM (JSON) / list all SIMs (JSON) |
| `fairwave sim export --format csv|json [--lab-creds]` | bureau batch (Ki/OPc merged from vault only) |
| `fairwave sim import <file.csv\|file.json>` | import a bureau batch (CSV or JSON only, never Ki/OPc) |
| `fairwave sim quota <imsi> --bytes N` | set a fair-use data allowance (0 = unlimited) |
| `fairwave sim usage <imsi>` | accumulated up/down bytes vs quota |
| `fairwave esim issue --imsi ... [--qr]` | mint an eSIM profile + activation code (lab vectors) |
| `fairwave esim list` / `revoke <code>` | activation-code registry |
| `fairwave esim serve` | run the lab SM-DP+ server (ES9+) |
| `fairwave peer list` / `add <name> <endpoint>` | mesh peers |
| `fairwave spectrum check --country --band [--indoor] [--license-ref]` | band/country pre-flight vs policy |
| `fairwave spectrum arm` / `disarm` | arm/clear the TX gate (arm requires acknowledgment; lab refuses) |
| `fairwave policy get` / `set` | routing/QoS policy (local breakout, max UEs, QoS caps) |
| `fairwave compliance [--out FILE]` | regulator-ready compliance report (CSV) |
| `fairwave backup` / `restore` | full state backup (optionally AES-256-GCM encrypted) / restore |
| `fairwave token create|list|revoke` | scoped API tokens (admin) |
| `fairwave alerts [--watch]` | operational alerts (active first, poll mode) |
| `fairwave audit [--limit N]` | append-only operator audit trail |
| `fairwave doctor` | local self-check: toolchains, control-plane reachability, RF-mode safety |
| `fairwave version` | release string + build metadata |

Global flags: `--control` (default `http://localhost:8080`, env `FAIRWAVE_CONTROL`),
`--token` (admin bearer token or `FAIRWAVE_ADMIN_TOKEN`), `--data-dir` (default `./data`;
holds the admin token file and the standalone eSIM registry).

## Example Outputs

```text
$ fairwave version
fairwave 0.1.0
```

```text
$ fairwave node status
fairwave-control 0.1.0
  mode:      lab
  phase:     on-air
  country:   US
  tx armed:  false
  nodes:     1
  ues:       2
  peers:     1
  node cafe-01          id=node-abc123          phase=on-air    tx=false
```

```text
$ fairwave sim issue --count 2 --profile lab
issued SIM imsi=999991234567890 profile=lab status=activated apn=internet
issued SIM imsi=999991234567891 profile=lab status=activated apn=internet
note: Ki/OPc were generated in the vault; export via sim export (bureau runbook)
```

```text
$ fairwave peer list
peer cafe-02          status=up      endpoint=10.60.0.2:51820 allowed=[10.60.0.0/24]
peer lab-box          status=up      endpoint=10.60.1.2:51820 allowed=[10.60.1.0/24]
```

```text
$ fairwave spectrum check --country US --band n48 --indoor
spectrum check US/n48 indoor=true: ALLOWED
```

## Rules for the CLI

- It never prints Ki/OPc, bootstrap tokens, or private keys (`doctor` redacts).
- Destructive commands (`node leave`, `sim revoke`) require confirmation: pass `--yes`, or answer the interactive `[y/N]` prompt. Without `--yes` on a non-TTY stdout the command refuses.
- Exit codes: `0` success, `1` general error, `2` usage error (bad/missing arguments or flags), `3` policy/legal gate refused (e.g. `spectrum arm` denied).
- `node join` is a stub: it exits `1` with `peering is not yet implemented (M2)` until the M2 mesh milestone lands.
- `sim import` accepts only `.csv` and `.json` files; any other extension is rejected with a usage error.
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
