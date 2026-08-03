---
title: Operations
---

# Operations

Operations are local-first: one person, one box, ssh + `fairwave-cli`. This section covers running a single box through a cluster, with runbooks for the common paths.

## Single Box to Cluster

| Scale | What changes | Runbook |
|---|---|---|
| 1 box, lab | zmq stack, no RF, sims issued, srsUE attaches | `cafe-pilot.md` (same flow, lab) |
| 1 box, RF (community) | SDR configured, gate armed, physical UEs | `index.md` + `monitoring.md` |
| 2–5 boxes, mesh | peers join, hub breakout, WireGuard | `docs/architecture/peering.md` |
| CBRS deployment | certified path, SAS client, grants | `docs/spectrum-and-law/cbrs.md` |

The lifecycle (provision → register → on-air → peer → breakout) is the same at every scale; only the gates differ.

## Runbook Index

| Runbook | When |
|---|---|
| `cafe-pilot.md` | two-hour pilot: preflight → install → SIM issue → UE attach → verify → teardown |
| `incident-response.md` | UEs can't attach, box offline, security event, gate bypass suspicion |
| `backup-restore.md` | before upgrades, after enrollment, DR |
| `monitoring.md` | Prometheus rules, alerting, log retention, privacy rules |

## Operating Principles

1. **Lab by default.** Any box without a completed `tx/arm` runs no RF. Prefer `zmq` until the authorization story is real.
2. **Everything local.** No cloud console; the UI and API live on the box. Remote access is ssh over your own network (or the mesh, mTLS).
3. **Evidence over assertion.** Logs, grants, gate decisions, and checklists are kept (`docs/spectrum-and-law/compliance-checklist.md`).
4. **Update deliberately.** `unattended-upgrades` covers Debian security only; Fairwave stack updates are explicit (`docs/hardware/image.md`).

## Baseline Operations

```text
ssh fairwave@<box>
fairwave node status        # phase, subsystems, gate
fairwave doctor             # deep self-check
fairwave peer list          # mesh state
journalctl -u fairwave-agent -f
docker compose -f compose/docker-compose.lab.yml logs -f
```

## Related

- Hardware deploy: `docs/hardware/index.md`
- Golden image: `docs/hardware/image.md`
- Software: `docs/software/index.md`
