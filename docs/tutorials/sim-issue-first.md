---
title: Issue Your First SIM
---

# Issuing Your First SIM

This tutorial walks through the full SIM lifecycle from the operator's point of view: mint a lab SIM, inspect the provisioner output, load it into the HSS, and attach a device.

> Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands without proper authorization is illegal in most jurisdictions. You are solely responsible for licenses, SAS grants, indoor restrictions, and type approval. HyperonX and contributors provide software as-is for lawful private networks, research, and shared-spectrum regimes only.

## Step 1: Issue a lab SIM

With the lab stack up (see [quickstart](quickstart-no-rf.md)), mint one SIM:

```bash
fairwave sim issue --profile lab --count 1 --label first-sim
```

Expected output:

```
✓ issued 1 lab SIM
  IMSI    9999912345678901  (lab, 15 digits)
  ICCID   8999990123456789012
  output  sims/2026-08-02/
          sims.csv  sims.json  hss-hook.sh
  NOTE    Ki/OPc material is hashed at rest (sha256, 12 hex).
          It is never committed, logged, or sent over the network
          in plaintext by the provisioner.
```

The `--label` is operator metadata only; it is not written to the HSS.

## Step 2: Inspect the output directory

The provisioner is offline-first: it writes everything you need to hand the credential to a card bureau or load it into an EPC.

```bash
ls sims/2026-08-02/
cat sims/2026-08-02/sims.json
```

```json
[
  {
    "imsi": "9999912345678901",
    "iccid": "8999990123456789012",
    "profile": "lab",
    "apn": ["internet", "ims"],
    "plmn": "999-99",
    "ki_sha256_12": "9f2c41b07d3a",
    "opc_sha256_12": "c8e61a9d05b4",
    "issued_at": "2026-08-02T09:00:00Z"
  }
]
```

Notes:

- `sims.csv` is the bureau-facing form (no Ki, no OPc — see the [bureau runbook](../sim-lifecycle/bureau-runbook.md)).
- `sims.json` is machine-readable for tooling.
- `hss-hook.sh` is the provisioner hook that loads the Ki/OPc into the Open5GS HSS database locally.
- Only `*_sha256_12` hashes ever appear in committed or logged output. Full Ki/OPc material lives only in the protected bundle files that never leave the machine by default.

## Step 3: Load into the HSS

In lab mode the hook runs automatically. To run it manually (e.g. after a reset):

```bash
bash sims/2026-08-02/hss-hook.sh
```

```
[open5gs-hss] adding subscriber 9999912345678901 (profile lab)
[open5gs-hss] 1 subscriber present in HSS
```

The hook inserts the Ki/OPc into the HSS database via the Open5GS internal API. For production deployments the equivalent mechanism targets the UDM interface; the [provisioner doc](../sim-lifecycle/provisioner.md) describes both.

## Step 4: Attach a device

### Option A — srsUE (lab, no SIM card needed)

The lab UE container reads the minted IMSI from the shared subscriber store:

```bash
docker compose -f deployments/lab/docker-compose.yml logs -f ue
```

```
[srsue] Network attach successful.
[srsue] PDN connectivity complete. IP: 10.45.0.2
```

### Option B — a real handset with a physical SIM card

This requires a **provisioned SIM card** (a physical card whose Ki/OPc match the HSS record) and a node with RF armed. A physical card is *not* provisioned by `fairwave sim issue` alone — the provisioner output must go to a card bureau that personalizes the card with the same Ki/OPc (see [bureau runbook](../sim-lifecycle/bureau-runbook.md)).

Workflow for a real handset:

1. `fairwave sim issue --profile prod` (separate profile from lab; never mix).
2. Send the encrypted bundle to your card bureau for personalization.
3. On the node: set country code, acknowledge the license, and arm TX (`fairwave spectrum check`, then `fairwave tx arm` — see [spectrum gate ADR](../adr/0008-spectrum-gate.md)).
4. Insert the card, dial nothing, watch the node UI for the attach: handset shows signal; control plane `/v1/sessions` lists the new session.

Expected control-plane session:

```bash
curl -s http://127.0.0.1:8080/v1/sessions | jq
```

```json
{
  "sessions": [
    {
      "imsi_sha256_12": "9f2c41b07d3a",
      "apn": "internet",
      "ue_ip": "10.45.0.3",
      "state": "active",
      "started_at": "2026-08-02T10:12:00Z"
    }
  ]
}
```

Note the session list exposes the IMSI **hash**, never the IMSI itself — see [privacy ADR](../adr/0010-privacy-logging.md).

## Differences: srsUE vs physical handset

| Aspect | srsUE (lab) | Physical handset (prod) |
| --- | --- | --- |
| SIM | Virtual credentials in HSS | Provisioned physical card |
| RF | ZMQ virtual radio | Real SDR, TX gated |
| Attach | Automatic at container start | Manual (phone boots, scans) |
| Band/EARFCN | Fixed in lab config | Must match your spectrum profile |
| Roaming/NAS quirks | srsUE is forgiving | Handsets enforce PLMN/TAC lists |

## Next steps

- [SIM lifecycle overview](../sim-lifecycle/index.md) — the full state machine.
- [Provisioner architecture](../sim-lifecycle/provisioner.md) — crypto and formats.
- [Bureau runbook](../sim-lifecycle/bureau-runbook.md) — turning JSON into cards.
- [Revocation](../sim-lifecycle/revocation.md) — taking a SIM away.
