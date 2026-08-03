---
title: Incident Response
---

# Incident Response

Runbooks for the four scenarios that actually happen to a small cell. **First rule: for RF/spectrum incidents, stop transmitting first, ask questions after.** Lab mode (zmq) incidents never involve spectrum.

## Severity Table

| Sev | Meaning | Examples | Target |
|---|---|---|---|
| S1 | Spectrum/legal exposure | gate bypass suspicion, TX outside allow-list, unauthorized band | stop TX, preserve evidence, escalate now |
| S2 | Service loss | no UE can attach, box offline, mesh down | restore service, post-incident review |
| S3 | Degraded | sync loss, backhaul poor, one peer down | fix in normal hours |
| S4 | Cosmetic | UI glitch, stale metric, docs bug | backlog |

## Runbooks

### UEs Can't Attach (S2)

1. `fairwave node status` - phase, core/RAN up?
2. `fairwave doctor` - store, API, SDR probe, sync.
3. Check agent telemetry: GPSDO lock, NTP offset, SDR temp (`fairwave_agent_*` metrics).
4. SIMs: `fairwave sim issue` / check profile matches HSS; hashes vs vault.
5. Restart eNB container; check srsRAN logs for `RRC connection` failures vs NAS rejects.
6. Verify S1 against Open5GS (MME logs); check PLMN/TAC on handset vs box.
7. If zmq lab: confirm `zmq` device set and ports not firewalled.

### Box Offline (S2)

1. Power + network: is it PoE'd, is the injector alive?
2. Console/serial if possible; else wait for agent heartbeat (10 s) or watchdog timer.
3. On boot: `fairwave node status`; check `fairwave-control` data dir integrity.
4. Restore from backup if store corruption (`docs/ops/backup-restore.md`).
5. Root cause in log slice before declaring fixed.

### Security Event (S1/S2)

1. Revoke exposure: `fairwave sim revoke` affected SIMs; drop peers (`peer list` → revoke); rotate mesh CA if key exposure suspected.
2. Contain: isolate box from LAN; preserve disk image (see Evidence).
3. Change local accounts; check WebAuthn + TOTP (`docs/architecture/security.md`).
4. Review audit log: gate events, enrollment, arm events.
5. Report per your obligations; `design/threat-model.md` describes what a box attacker can do - assume full access to box-local keys.

### Spectrum Gate Bypass Suspicion (S1)

1. **Stop TX immediately**: `fairwave tx/arm` revoke + agent `safe_tx` assert off; power the SDR if needed.
2. Preserve evidence: gate log, arm events, eNB config, rfkill state, timestamps.
3. Do NOT re-arm to "test" - that compounds the exposure.
4. Analyze: was the allow-list wrong, the gate buggy, or the config hand-edited? Hand-editing core/RAN config bypasses the gate by design of the OS layer; treat as policy breach.
5. File the compliance checklist state and regulator contact if relevant.

## Rollback

- Config rollback: `fairwave-control` store is file-backed; restore a pre-change snapshot, reconcile.
- Full rollback: restore golden image + data backup, re-enroll peers (mesh CA certs reissue).

## Evidence Preservation

| Artifact | Location |
|---|---|
| Gate + arm events | `/var/lib/fairwave/logs/`, journald slice |
| Prometheus samples | scrape dumps before cleanup |
| Compliance checklist | operator copy (sign-off page) |
| Configs rendered | `/var/lib/fairwave/open5gs/`, `srsran/` |
| Disk image | `dd` the disk or NVMe clone before remediation |

Timestamp everything; preserve original logs read-only (copy, don't edit).

## Regulator Contact (Lawful Only)

If the incident is a legal exposure: stop TX, preserve evidence, then contact the regulator through the official channel for your region (`docs/spectrum-and-law/regional.md`). Do not contact SAS/regulator from a panic state; the evidence file, not apologies, is what matters. This project does not advise on legal strategy - consult counsel.
