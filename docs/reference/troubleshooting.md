---
title: Troubleshooting
---

# Troubleshooting Reference

Symptom → cause → fix, fastest first. When unsure where you are, run `fairwave doctor` — it checks containers, ports, ZMQ sockets, control-plane connectivity, and config schema, and prints a pass/fail table.

| # | Symptom | Likely cause | Fix |
| --- | --- | --- | --- |
| 1 | UE won't attach (loops at `Attaching...`) | No subscriber in HSS for the IMSI; or IMSI not in the lab subscriber store | Mint a SIM: `fairwave sim issue --profile lab --count 1`; re-run `hss-hook.sh`; restart `ue` container |
| 2 | UE attach rejected with auth failure | Ki/OPc mismatch between SIM and HSS, or SIM profile/prefix mismatch | Re-issue the SIM and reload HSS; confirm IMSI prefix matches profile (`sim status`); check `/v1/sims` state is `activated`, not `revoked` |
| 3 | ZMQ bind error on 2100/2200 | Port clash with another process or second lab stack | `docker compose ps`; kill the other stack or change `ZMQ_PORT` env in compose; ensure `make lab-down` ran fully |
| 4 | `docker compose port` conflict (e.g. 8080/3000/27017 in use) | Host app already bound to a lab port | Remap host ports in `deployments/lab/docker-compose.yml` (`"8080:8080"` → `"18080:8080"`), update `fairwave` CLI base URL, or stop the competing app |
| 5 | PLMN wrong (handset shows no network / different MCC-MNC) | Config edit not applied; or stale HSS/UE config | Verify `network.plmn` in config; `make lab-up` to recreate; re-issue SIMs (IMSI prefix follows PLMN); check `make status` shows the intended PLMN |
| 6 | SIM rejected: "auth fail" at MME | Ki/OPc not loaded, SIM revoked, or rate-limit lockout | Check HSS has the IMSI (`docker compose logs open5gs`); `fairwave sim status --imsi`; reload hook; confirm not on any block list |
| 7 | WireGuard handshake timeout | Wrong endpoint, NAT mapping stale, firewall blocks UDP/51820, keys mismatch after `node init` | `fairwave doctor --peer <name>`; verify UDP reachability (`nc -u`); enable `PersistentKeepalive`; re-join if public keys changed |
| 8 | mDNS not seeing peers | Different L2 segment, multicast blocked, mesh name mismatch, unsigned announces dropped | Confirm both on same VLAN/subnet; check `mesh` name matches; verify announce signature key matches the peer's join-time key; fall back to `--rendezvous` |
| 9 | Control plane auth failure (401/403) | Expired bearer token, revoked cert, wrong role for the action | Re-login (WebAuthn/TOTP); `fairwave operator sessions` and re-issue; check role vs endpoint requirement in [rest.md](../api/rest.md) |
| 10 | `/metrics` empty | Prometheus scraping wrong port/path; control plane not exposing metrics in this build | Confirm `GET :8080/metrics` directly; check scraper targets the control-plane port (not the UI port 8081); metrics are on `/metrics`, not `/v1/metrics` |
| 11 | `make status` shows `state: degraded` | One container restarted or unhealthy | `docker compose ps`; check each service log; `fairwave doctor` pinpoints; restart the failing service |
| 12 | `spectrum check` returns `allowed: false` | EARFCN not in allow-list, or country not set, or license not acknowledged | Add EARFCN to the spectrum profile YAML ([customization](../tutorials/customization.md)); set country code; acknowledge license; re-run check |
| 13 | Session list shows nothing after attach | UE attached but no active bearer (APN mismatch) or PGW NAT down | Check UE APN matches config (`internet`); `docker compose logs open5gs | grep -i pdn`; restart `open5gs` |
| 14 | Control plane `500` on issue | Vault KEK missing or output dir inside git tree | Set `FW_SIM_KEK` (or HSM slot); move output dir or pass `--force-offline`; check audit log for the specific error |

## Collecting diagnostics

```bash
fairwave doctor --json > /tmp/fw-doctor.json
docker compose -f deployments/lab/docker-compose.yml logs --tail 100 enb ue open5gs
curl -s http://127.0.0.1:8080/v1/status
curl -s http://127.0.0.1:8080/v1/tx/arm
```

## Still stuck?

- Read the deep-dive for the relevant layer: [lab attach](../tutorials/lab-attach.md), [mesh runbook](../peering/mesh-runbook.md), [SIM lifecycle](../sim-lifecycle/index.md).
- Open an issue with the doctor JSON, logs (hashes fine — no credentials), and what you changed.
- Do not post Ki/OPc, IMSIs, or vault material anywhere, including issue trackers (ADR-0010).
