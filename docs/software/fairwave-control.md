---
title: fairwave-control
---

# fairwave-control

`fairwave-control` is the control plane: REST API v1, reconcile loop, enrollment, policy, TX gate, SIM/eSIM ops, and fair-use metering. It is the only writer to core (Open5GS / free5GC) and srsRAN configuration. Architecture: `docs/architecture/control-plane.md`.

## Flags

```
fairwave-control [flags]

  --config string       path to YAML config (default /etc/fairwave/control.yaml)
  --data-dir string     data + keys directory (default /var/lib/fairwave)
  --listen string       REST + metrics listen address (default 0.0.0.0:8080)
  --log-level string    debug|info|warn|error (default info)
  --version             print version and exit
```

No flags are secrets; configuration of keys happens via files, not CLI.

## Config File (YAML)

The lab config lives at `deploy/config/fairwave-control.yaml` (schema-validated against an embedded JSON Schema, with semantic checks). Shape:

```yaml
# /var/lib/fairwave/control.yaml (lab defaults shown)
server:
  listen: ":8080"
  data_dir: /var/lib/fairwave
  mode: lab              # lab | rf
  country: LAB
  log_level: info
  log_format: json       # json | console
plmn:
  mcc: "999"
  mnc: "99"
tac: 7
apns: [internet, ims]
auth:
  bootstrap_token_ttl: 5m
  admin_token_env: FAIRWAVE_ADMIN_TOKEN
southbound:
  driver: none           # docker | none
hss:
  driver: mongosh        # mongosh | dbctl | free5gc | none
  container: mongo       # container to exec the write-back into
peering:
  enabled: true
  mdns: true
policy:
  local_breakout: true
  max_ues: 128
  apns: [internet, ims]
telemetry:
  metrics: true

# Core selector: open5gs (default) or free5gc (5G SA).
core: open5gs
free5gc:
  amf_oam_url: http://amf:8000   # AMF OAM (namf-oam) base URL
  cdr_dir: ""                    # CHF CDR dir (shared volume) -> core-metered usage

# Usage collectors (feed /v1/sessions + fair-use quotas).
collector:
  enabled: false
  interval: 15s
  mme_url: http://127.0.0.2:9090  # Open5GS MME infoAPI (open5gs core)
  smf_url: ""                     # optional SMF infoAPI
  upf:
    enabled: false                # GTP-U accounting tap (needs CAP_NET_RAW)
    iface: ""                    # interface carrying GTP-U (S1-U/S5-U)

# Fair-use: auto-suspend SIMs over quota. Opt-in on purpose.
fairuse:
  enabled: false
  usage_interval: 60s
```

## Env Vars

Every config key has a `FAIRWAVE_*` override (flat, explicit mapping in `config.go`):

| Var | Effect |
|---|---|
| `FAIRWAVE_SERVER_LISTEN` / `FAIRWAVE_SERVER_DATADIR` / `FAIRWAVE_SERVER_MODE` / `FAIRWAVE_SERVER_COUNTRY` / `FAIRWAVE_SERVER_LOGLEVEL` | server basics |
| `FAIRWAVE_PLMN_MCC` / `FAIRWAVE_PLMN_MNC` / `FAIRWAVE_TAC` | PLMN/TAC |
| `FAIRWAVE_ADMIN_TOKEN` | admin bearer token (consumed by API auth, not config) |
| `FAIRWAVE_SOUTHBOUND_DRIVER` / `FAIRWAVE_HSS_DRIVER` / `FAIRWAVE_HSS_CONTAINER` | southbound + HSS write-back |
| `FAIRWAVE_PEERING_DISABLED` / `FAIRWAVE_POLICY_LOCAL_BREAKOUT` | peering/policy switches |
| `FAIRWAVE_CORE` | `open5gs` or `free5gc` |
| `FAIRWAVE_FREE5GC_AMF_OAM_URL` / `FAIRWAVE_FREE5GC_CDR_DIR` | free5GC AMF OAM + CHF CDR metering |
| `FAIRWAVE_COLLECTOR_ENABLED` / `_MME_URL` / `_SMF_URL` / `_UPF_ENABLED` / `_UPF_IFACE` | usage collectors |
| `FAIRWAVE_ESIM_ENABLED` / `_SMDP_ADDRESS` / `_SINGLE_USE` | eSIM/SM-DP+ |
| `FAIRWAVE_ALERTS_ENABLED` / `FAIRWAVE_FAIRUSE_ENABLED` | alerts + fair-use auto-suspend |

Secrets (mesh CA private key, vault key) are **never** passed via env; they live in files.

## Data Directory Layout

```
/var/lib/fairwave/
├── store/          # JSON: nodes, sims, peers, policy, lifecycle, usage
├── keys/           # node key, mesh CA (0600)
├── vault/          # SIM vault (encrypted)
├── open5gs/        # rendered core config templates
├── srsran/         # rendered eNB/gNB config
├── esim/           # eSIM activation registry
└── logs/           # rotated component logs
```

## Running: Docker vs Bare

| Mode | How | When |
|---|---|---|
| Docker Compose (default) | `compose/` brings control + core + RAN + UI + portal up together; control inside the compose network | default, v0.1 lab |
| Bare (systemd) | `fairwave-control.service`; control talks to core/RAN containers over the compose network or a fixed bridge | golden image, RF builds |

Either way the API is the same: `GET /v1/healthz` proves liveness, `GET /v1/status` proves readiness.

## First Boot Sequence

1. `fairwave node init` (CLI) → node record created.
2. Set policy (`fairwave policy set`) - APNs, band allow-list.
3. Arm TX only when a real legal basis exists (`POST /v1/tx/arm`); for lab, leave it off.
4. (Optional) enable fair-use: `fairwave sim quota <imsi> --bytes N`; usage comes from the collector (`collector.upf` tap, or `core: free5gc` + `free5gc.cdr_dir`).

## Related

- CLI: `docs/software/fairwave-cli.md`
- Agent: `docs/software/fairwave-agent.md`
- Architecture: `docs/architecture/control-plane.md`
