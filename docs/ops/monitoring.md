---
title: Monitoring
---

# Monitoring

Monitoring is local-first: Prometheus scrapes `fairwave-control` `/metrics` on the box, rules fire locally, and alerts go to an operator-configured channel. Metrics contract: `docs/architecture/telemetry.md`.

## Prometheus Rules (Examples)

```yaml
groups:
  - name: fairwave-box
    rules:
      - record: fairwave:tx_armed
        expr: fairwave_tx_armed > 0
      - alert: AgentHeartbeatStale
        expr: fairwave_agent_heartbeat_age_seconds > 60
        for: 2m
        annotations:
          summary: "agent heartbeat stale on {{ $labels.instance }}"
      - alert: SyncLost
        expr: fairwave_agent_gpsdo_lock == 0
        for: 5m
        annotations:
          summary: "GPSDO unlocked (real RF at risk)"
      - alert: ThermalWarning
        expr: fairwave_agent_cpu_temp_celsius > 80
        for: 5m
        annotations:
          summary: "CPU thermal warning (enclosure airflow?)"
      - alert: ReconcileStuck
        expr: fairwave_reconcile_errors > 0
        for: 10m
        annotations:
          summary: "reconcile loop not converging"
      - alert: NoSessionsUnexpected
        expr: fairwave_sessions_active == 0
        for: 24h
        annotations:
          summary: "no sessions for 24h — investigate or it's expected (lab?)"
```

`fairwave_agent_ntp_offset_seconds` > 50 ms for 10 min → alert in RF deployments. `fairwave_agent_sdr_temp_celsius` > 75 → hardware check.

## Alerting

- **Prometheus Alertmanager** sidecar, operator-configured receivers: email and/or generic webhook. No external SaaS by default.
- Severity mapping mirrors `docs/ops/incident-response.md`: `critical` (S1/S2), `warning` (S3), `info` (S4).
- Alert deduplication and silence windows: support/known-testing windows per box.

## Dashboards

- Grafana is optional and hand-provisioned (no bundled dashboard in v0.1); recommend three panels groups:
  1. **Coverage/sync**: `fairwave_tx_armed`, `fairwave_agent_gpsdo_lock`, NTP offset, eNB process state.
  2. **Backhaul**: uplink rate/latency, agent heartbeats, disk/IO.
  3. **UE/usage**: `fairwave_sessions_active` (counts only), attach events per hash-bucket — never IMSI.

## Log Retention

| Source | Where | Retention |
|---|---|---|
| journald (agent, control) | system | 14 days default, extend to 90 |
| component JSON logs | `/var/lib/fairwave/logs/` | 30 days, rotated 10 MB |
| Prometheus TSDB | docker volume | 15 days, or as disk allows |
| Gate/arm events | audit log (append-only) | **indefinite** — legal evidence |

Adjust retention to fit the disk; the audit log is exempt from cleanup by policy.

## Privacy in Monitoring

- **No IMSI in logs, metrics, or dashboards** — enforced at the producer (`docs/architecture/telemetry.md`), not by dashboard config.
- Session metrics are counts; UE identifiers are truncated hashes.
- Review your alerting receivers: webhook endpoints receive metric labels only, which contain no subscriber data.
- A stolen box's Prometheus DB reveals topology and counts, not subscribers — that is the point.

## Related

- Metrics contract: `docs/architecture/telemetry.md`
- Incident response: `docs/ops/incident-response.md`
- Agent probes: `docs/software/fairwave-agent.md`
