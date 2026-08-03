---
title: Telemetry
---

# Telemetry

Telemetry in Fairwave is local-first and privacy-aware: no IMSI, no subscriber content, no phone-home of subscriber data. Everything is produced on the box and consumed on the box; optional Grafana is strictly opt-in.

## Metrics

- `fairwave-control` exposes **Prometheus** text format at `GET /metrics`.
- Metric families (v0.1):

| Metric | Meaning |
|---|---|
| `fairwave_node_info{role,version}` | node identity, role (standalone/peer/hub) |
| `fairwave_lifecycle_phase` | current phase (provision…breakout) |
| `fairwave_reconcile_loop_total{result}` | reconcile runs by outcome |
| `fairwave_reconcile_errors` | outstanding reconciliation errors |
| `fairwave_peers_connected` | active mTLS peers |
| `fairwave_sessions_active` | active PDN sessions (count only) |
| `fairwave_tx_armed{band}` | TX gate state per band |
| `fairwave_agent_heartbeat_age_seconds` | staleness of agent heartbeat |
| `fairwave_agent_gpsdo_lock` | 0/1 GPSDO lock from agent |
| `fairwave_agent_ntp_offset_seconds` | NTP offset from agent |
| `fairwave_agent_cpu_temp_celsius` | die temperature from agent |

- Prometheus scrapes the control-plane endpoint; the agent forwards hardware metrics to the control plane, so there is exactly one scrape target per box.

## Structured Logs

- `fairwave-control` and `fairwave-agent` log **JSON lines** via zerolog to stdout (journald in systemd deployments, Docker logging driver in Compose).
- Required fields: `time`, `level`, `component`, `event`, `msg`. Optional context: `peer`, `sim_hash`, `phase`.
- **Never logged:** full IMSI, Ki, OPc, bootstrap tokens, private keys, local breakout payloads.
- Log retention: `docs/ops/monitoring.md`.

## Tracing (Stub)

- OpenTelemetry is stubbed in v0.1: `FAIRWAVE_OTEL_ENDPOINT` is read, spans are sketched for the reconcile loop and API calls, but no exporter is wired. The endpoint config exists to keep the tracing contract stable across releases (see `design/roadmap.md`).
- Do not deploy OTLP collectors expecting working traces in v0.1 - it is not implemented yet.

## Dashboards

- No bundled Grafana provisioning in v0.1.
- Recommended optional stack: Prometheus (Docker sidecar) + Grafana, dashboards hand-authored from the metric families above.
- The operator UI (`docs/software/operator-ui.md`) shows live status natively without Grafana; Grafana is for long-term history only.

## Alerting

- Alerting is rule-based and local: `fairwave_alertmanager` config example in `docs/ops/monitoring.md`.
- Notifications are operator-configured (email/webhook), never external by default.

## Privacy Line

Telemetry carries counts, hashes, and phase state. It cannot attribute a subscriber: session metrics are counts, UE IDs are truncated hashes. This is a deliberate design constraint, not a gap.
