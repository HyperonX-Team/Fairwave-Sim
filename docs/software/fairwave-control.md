---
title: fairwave-control
---

# fairwave-control

`fairwave-control` is the control plane: REST API v1, reconcile loop, enrollment, policy, TX gate. It is the only writer to Open5GS/srsRAN configuration. Architecture: `docs/architecture/control-plane.md`.

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

```yaml
# /etc/fairwave/control.yaml
node:
  name: cafe-01
  role: standalone        # standalone | peer | hub
  plmn: "999-99"
  tac: 7
core:
  open5gs:
    image: open5gs:dev
    apns: [internet, ims]
ran:
  srsran:
    image: srsran:dev
    device: zmq          # zmq | b200 | b210 | limesdr | bladerf
    band_profile: lab     # lab | community | cbrs
mesh:
  advertise: true
  wireguard_port: 51820
policy:
  default_breakout: local # local | wireguard
security:
  bootstrap_token_ttl: 10m
```

## Env Vars

| Var | Effect |
|---|---|
| `FAIRWAVE_CONFIG` | config file path (overrides `--config`) |
| `FAIRWAVE_DATA_DIR` | data dir |
| `FAIRWAVE_LISTEN` | listen address |
| `FAIRWAVE_LOG_LEVEL` | log level |
| `FAIRWAVE_OTEL_ENDPOINT` | OTLP endpoint (stub; not wired in v0.1) |
| `FAIRWAVE_MESH_CA_DIR` | mesh CA directory (default under data dir) |

Secrets (mesh CA private key, vault key) are **never** passed via env; they live in files.

## Data Directory Layout

```
/var/lib/fairwave/
├── store/          # JSON: nodes, sims, peers, policy, lifecycle
├── keys/           # node key, mesh CA (0600)
├── vault/          # SIM vault (encrypted)
├── open5gs/        # rendered core config templates
├── srsran/         # rendered eNB config
└── logs/           # rotated component logs
```

## Running: Docker vs Bare

| Mode | How | When |
|---|---|---|
| Docker Compose (default) | `compose/` brings control + core + RAN + UI + portal up together; control inside the compose network | default, v0.1 lab |
| Bare (systemd) | `fairwave-control.service`; control talks to core/RAN containers over the compose network or a fixed bridge | golden image, RF builds |

Either way the API is the same: `GET /v1/healthz` proves liveness, `GET /v1/status` proves readiness.

## First Boot Sequence

1. `fairwave node init` (CLI) → identity + mesh CA generated.
2. `fairwave node register` → enrollment, lifecycle → `register`.
3. Set policy (`fairwave policy set`) — APNs, band allow-list.
4. Arm TX only when a real legal basis exists (`POST /v1/tx/arm`); for lab, leave it off.

## Related

- CLI: `docs/software/fairwave-cli.md`
- Agent: `docs/software/fairwave-agent.md`
- Architecture: `docs/architecture/control-plane.md`
