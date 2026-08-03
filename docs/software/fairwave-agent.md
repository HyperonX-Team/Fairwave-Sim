---
title: fairwave-agent
---

# fairwave-agent

`fairwave-agent` is the on-box sensor and watchdog. It runs natively (systemd) even when the stack is containerized, because it must probe hardware, not containers.

## Duties

1. **Heartbeat** - periodic signed heartbeat to `fairwave-control` (default 10 s). The control plane tracks `fairwave_agent_heartbeat_age_seconds`; a stale heartbeat flips `/v1/status` to degraded and trips alert rules.
2. **Health probes**:

| Probe | Source | Metric |
|---|---|---|
| CPU temperature | thermal zone / `lm-sensors` | `fairwave_agent_cpu_temp_celsius` |
| GPSDO lock | SDR status via driver probe | `fairwave_agent_gpsdo_lock` |
| NTP offset | `timedatectl` / `ntpstat` | `fairwave_agent_ntp_offset_seconds` |
| SDR temperature | `uhd_usrp_get_...` / LimeSuite temp | `fairwave_agent_sdr_temp_celsius` |
| Disk/IO | `/proc` + sysfs | `fairwave_agent_disk_bytes`, `fairwave_agent_io_busy_ratio` |

3. **Watchdog** - periodic re-check (timer unit `fairwave-watchdog.timer`):

- control-plane endpoint reachable?
- core/RAN containers healthy (via docker healthchecks)?
- SDR still probing?
- on failure: collect evidence (journal slice), restart service or notify; never silently ignore.

4. **Safe-TX flag** - the agent holds the *hardware* side of the TX gate: `safe_tx` is only true when the control plane's gate is armed AND the armed band matches the eNB config AND rfkill is in the expected state. The eNB process itself is only allowed to start when the agent asserts `safe_tx`.

## Behavior Rules

- Agent never writes core/RAN config; it only reports and supervises.
- Agent restarts with the box; it does not resolve config drift itself.
- Logs are JSON lines (zerolog), `component=agent`; no subscriber material ever (`docs/architecture/telemetry.md`).
- Probe failures are reported, not "fixed" by the agent - remediation is operator-driven.

## Example Log Line

```json
{"time":"2026-08-02T12:00:01Z","level":"warn","component":"agent","event":"gpsdo_unlocked","msg":"GPSDO lost lock","sdr_temp_c":42}
```

## Configuration

| Env | Default | Purpose |
|---|---|---|
| `FAIRWAVE_CONTROL_URL` | `http://127.0.0.1:8080` | control endpoint |
| `FAIRWAVE_AGENT_INTERVAL` | `10s` | heartbeat interval |
| `FAIRWAVE_AGENT_SDR` | auto-detect | which driver to probe (uhd/limesuite/bladerf) |
| `FAIRWAVE_WATCHDOG_PERIOD` | `30s` | watchdog timer period |

## Related

- Telemetry: `docs/architecture/telemetry.md`
- Golden image units: `docs/hardware/image.md`
- Incident runbooks: `docs/ops/incident-response.md`
