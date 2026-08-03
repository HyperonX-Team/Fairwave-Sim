---
title: Lab Attach Deep Dive
---

# Lab Attach: What `make lab-up` Actually Does

> [!NOTE]
> **Verified scope (v0.1.0 lab).** The stack is verified end-to-end through
> the EPC: UE RRC attach on the lab PLMN, NAS authentication (milenage
> against HSS), security mode, default EPS bearer creation and UE IP
> allocation by the MME. The srsENB/srsUE ZMQ loopback needs stable
> low-jitter scheduling for the final data-path hop (tun_srsue). On native
> Linux it passes reliably; under Docker Desktop (Windows/macOS, WSL2) the
> UE PHY can lose subframe sync (SYNC TRACK ret=-1) and the Attach Accept
> delivery over the radio link fails even though the EPC side completes.
> That is a host environment limitation, not a config bug - the exact same
> config is the reference topology used by the s5uishida Open5GS EPC +
> srsRAN_4G sample configs on Linux hosts. Run the lab on native Linux for
> the full data path.

This page goes one level below the quickstart: what the lab stack contains, how to read its logs, and how to prove an attach happened.

## The compose stack

`make lab-up` runs `docker compose up -d` against `deployments/lab/docker-compose.yml` (lab compose). Five services:

| Service | Image / role | Ports exposed |
| --- | --- | --- |
| `mongo` | MongoDB for the control plane and HSS-adjacent data | 27017 (internal) |
| `open5gs` | Open5GS EPC: MME, SGW, PGW, HSS in one container | 38412/SCTP (internal), web UI on 3000 |
| `enb` | srsRAN eNB, ZMQ virtual radio (`--rf.device=zmq`) | 2100 (ZMQ), S1 SCTP outbound |
| `ue` | srsUE, ZMQ virtual radio | 2200 (ZMQ), tun interface `tun_srsue` |
| `control-plane` | fairwave-control + agent, REST API, provisioning hooks | `8080` (API), `8081` (UI) |

The two ZMQ sockets (`tcp://*:2100`, `tcp://*:2200`) are the only "RF": IQ samples travel between the eNB and UE processes over TCP loopback. No SDR, no antenna, no emissions.

## Boot order and the attach timeline

1. `mongo` starts, waits for readiness.
2. `open5gs` starts, seeds HSS subscribers from `deployments/lab/subscribers.json` if present (or from the provisioner output mounted into the container).
3. `enb` starts and registers its S1 connection with the MME (Open5GS). Watch for `S1Setup` / `S1-AP connection established`.
4. `ue` starts, scans the virtual PLMN, and attaches: RRC → NAS attach → authentication against HSS → default EPS bearer.
5. `control-plane` health-checks the EPC and marks state `on-air` (lab).

## Watching the attach

```bash
cd deployments/lab

# EPC side: MME signalling
docker compose logs -f open5gs | grep -iE "attach|IMSI|S1|bearer"

# Radio side: eNB sees RRC setup complete
docker compose logs -f enb | grep -iE "RRC Connection completed|S1Setup"

# Handset side: the definitive proof
docker compose logs -f ue | grep -iE "attach|PDN|network registration"
```

### Asserting attach (assertion checklist)

| Check | Command | Success signal |
| --- | --- | --- |
| eNB up | `docker compose logs enb \| grep srsENB` | `S1 Setup procedure completed` |
| UE registered | `docker compose logs ue \| grep srsue` | `Network attach successful` |
| PDN established | `docker compose logs ue \| grep -i pdn` | `PDN connectivity complete` |
| IP handed out | `docker compose logs open5gs \| grep -i "pdn\|pool"` | UE assigned 10.45.0.x |
| End-to-end ping | `docker compose exec ue ping -c 3 10.45.0.1` | 0% packet loss |
| Control plane agrees | `curl http://127.0.0.1:8080/v1/status` | `"state":"on-air"` |

The UE's virtual tun interface is named `tun_srsue` inside the UE container; ping through it:

```bash
docker compose exec ue ping -I tun_srsue -c 3 8.8.8.8
```

Traffic exits via the PGW's NAT into your host network (local breakout), so external pings work too.

## Troubleshooting

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| UE loops at `Attaching...` | HSS has no subscriber for the IMSI | Mint a lab SIM (`fairwave sim issue --profile lab --count 1`) and restart the stack |
| `PDN connectivity` never completes | APN mismatch | Confirm UE APN is `internet` (default) and matches config |
| eNB shows `S1Setup` failing | Open5GS MME not ready | `docker compose restart enb` after open5gs is healthy |
| ZMQ bind error on 2100/2200 | Port clash with another process | Change `ZMQ_PORT` env in compose or stop the competing process |
| Control plane `state: degraded` | A container restarted | `docker compose ps`; check `fairwave doctor` output |
| No logs from `ue` | Container exited (bad config) | `docker compose logs ue --tail 50`; look for config parse errors |

When in doubt, run `fairwave doctor` - it checks containers, ports, ZMQ sockets, control-plane connectivity, and prints a pass/fail table.

## Teardown hygiene

```bash
make lab-down
```

Removes containers and the compose network. The `sims/` output directory and Mongo data volume are intentionally **kept** - delete them explicitly when you are done with a campaign, and never commit their contents.

See also: [SIM issuance](../tutorials/sim-issue-first.md), [troubleshooting reference](../reference/troubleshooting.md).
