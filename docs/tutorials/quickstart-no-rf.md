---
title: Quickstart (No RF, 30 Minutes)
---

# Quickstart: Full Lab Stack Without RF (30 minutes)

This tutorial gets a complete private LTE network running on your laptop in about 30 minutes. Nothing is transmitted: the eNB and UE talk to each other over srsRAN's ZMQ virtual radio, inside Docker, end to end.

## Prerequisites

| Requirement | Minimum | Notes |
| --- | --- | --- |
| Docker Engine | 24+ | Docker Compose v2 included |
| RAM | 8 GB | 16 GB recommended |
| Disk | 10 GB free | Container images |
| Go | 1.22+ | Only needed to build `fairwave-cli` |
| OS | Linux, macOS, or Windows (WSL2) | Lab stack runs in containers |

You do **not** need an SDR, a SIM card, a license, or any RF gear for this tutorial.

## 1. Clone and bootstrap

```bash
git clone https://github.com/hyperonx/fairwave.git
cd fairwave
./scripts/bootstrap.sh
```

Expected output (abridged):

```
[fairwave] checking prerequisites... ok (docker 26.1.4, 16 GB RAM)
[fairwave] pulling images (open5gs, srsran, fairwave-control)...
[fairwave] initializing mongo seed data (plmn 999-99, tac 7)...
[fairwave] bootstrap complete in 42s
```

`bootstrap.sh` verifies the environment and writes the local lab config under `deploy/config/`.

## 2. Build the CLI

```bash
make build
fairwave version
```

Expected output:

```
fairwave 0.1.0
```

If you do not want to build from source, a signed release binary is available in the v0.1.0 release assets (see [release signing](../security/release-signing.md)).

## 3. Bring the lab up

```bash
make lab-up
```

Expected output (abridged):

```
[+] Running 6/6
 Container fairwave-mongo-1          Started
 Container fairwave-open5gs-1        Started
 Container fairwave-enb-1            Started
 Container fairwave-ue-1             Started
 Container fairwave-control-plane-1  Started
lab stack up: control-plane http://127.0.0.1:8080/v1/healthz
```

This starts Mongo, Open5GS, the ZMQ eNB, the ZMQ UE, and the control plane. No SDR is touched. Verify:

```bash
make status
```

```
STATE: on-air
PLMN: 999-99   TAC: 7
eNB:  connected (zmq virtual radio)
UE:   registered
APNs: internet, ims
```

## 4. Mint a lab SIM

```bash
fairwave sim issue --profile lab --count 1
```

Expected output:

```
issued 1 lab SIM
  IMSI    9999912345678901  (lab)
  ICCID   8999990123456789012
  output  sims/2026-08-02/  (sims.csv, sims.json, hss-hook.sh)
  NOTE    Ki/OPc written to local files only; never committed.
```

The provisioner writes Ki/OPc material locally, hashes them at rest, and renders a HSS load script. The lab profile is auto-loaded into the Open5GS HSS in lab mode via the provisioner hook.

## 5. Verify the attach

`make lab-up` runs the assert script (`tests/e2e-sim/assert-lab-up.sh`), which verifies:

1. Open5GS MME + HSS running
2. eNB S1-MME connected to the MME
3. UE RRC connection + random access on the lab PLMN
4. UE NAS authentication + security mode (milenage against HSS)
5. MME creates the default EPS bearer and allocates a UE IP

Watch it live:

```bash
docker compose -f deploy/docker-compose.yml logs ue1 | grep -E "RRC|Attach|Security"
docker compose -f deploy/docker-compose.yml logs open5gs | grep -E "Bearer added|Attach accept"
```

Expected lines (EPC side completes in all environments):

```
[ue1] RRC Connected
[ue1] Random Access Complete.     c-rnti=0x48, ta=0
[ue1] Received Security Mode Command ... eia: 128-EIA2
[open5gs] Attach accept
[open5gs] Bearer added (EBI=5 IMSI=999991234567001)
```

> [!NOTE]
> The final hop - UE IP visible on `tun_srsue` and ping through it - needs
> stable ZMQ timing. It passes on native Linux. Under Docker Desktop
> (Windows/macOS) the UE PHY can lose subframe sync and the attach accept
> is not delivered over the virtual radio, even though the EPC-side attach
> completes. See [lab attach deep dive](lab-attach.md) for details and the
> host recommendation.

## 6. Tear down

```bash
make lab-down
```

The lab leaves no persistent RF state behind. To wipe subscriber data entirely, run `docker compose -f deploy/docker-compose.yml down -v` (or `make lab-down` which does the same).

## What you just ran

| Layer | Component |
| --- | --- |
| EPC | Open5GS (MME/SGW/PGW/HSS) |
| Radio | srsRAN eNB on ZMQ virtual radio |
| Handset | srsUE on ZMQ virtual radio |
| Control plane | fairwave-control + fairwave-agent |
| Operator UI | Available at `http://127.0.0.1:8081` |

## Next steps

- Deep dive: [lab attach internals and troubleshooting](lab-attach.md).
- Issue your first SIM properly: [SIM issuance tutorial](sim-issue-first.md).
- Two nodes, one mesh: [two-box peering](two-box-peering.md).
- Change PLMN, TAC, bands: [customization](customization.md).
