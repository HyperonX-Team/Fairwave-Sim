# Open5GS configs (lab EPC)

This directory holds the Open5GS v2.7 configuration for the Fairwave lab
EPC. It is the single source the compose/helm/ansible stacks mount at
`/etc/open5gs/open5gs.yaml`.

## What maps to what

| Service | Open5GS daemon | Config section | Talks to |
| --- | --- | --- | --- |
| `mme` | `open5gs-mmed` | `mme` (s1ap, gtpc, gummei, tai, security) | eNB (S1-MME) + HSS (S6a) + SGW (S11) |
| `sgw` | `open5gs-sgwd` | `sgw` (gtpc, gtpu) | MME (S11), PGW (S5), eNB (S1-U) |
| `pgw` | `open5gs-pgwd` | `pgw` (gtpc, gtpu, ue_pool, apn) | SGW (S5), Internet (SGi) |
| `hss` | `open5gs-hssd` | `hss` (db) | MongoDB subscriber store |
| `pcrf` | `open5gs-pcrd` | `pcrf` (freeDiameter) | Policy (lab: defaults) |

Key lab values:

- PLMN **999-99** (test range, non-routable), TAC **7** - see the MME `gummei`
  and `tai` sections.
- S1-MME binds **0.0.0.0** so the srsENB container can reach it; all S11/S5
  traffic stays on 127.0.0.1 inside the EPC container.
- PGW UE pool **10.45.0.0/16**; APNs `internet` + `ims` with public DNS.
- HSS reads subscribers from **MongoDB** (`mongodb://mongo:27017`); seed the
  lab subscribers with `hss-init.sh`.

## How fairwave-control templates it

The control plane renders site-specific configs from this file (or from its
own policy store) when a node is enrolled:

1. **PLMN**: `mcc`/`mnc` under `mme.gummei`, `mme.tai` and the `pgw` APNs are
   substituted from the site policy (`{{FW_PLMN_MCC}}` / `{{FW_PLMN_MNC}}`).
   Lab defaults are hardcoded here; production profiles must use an
   operator-assigned PLMN.
2. **TAC** and **MME address**: `tai.tac` and the advertised MME endpoint are
   derived from the site record the operator configured via the API.
3. The rendered file is written to `/etc/open5gs/open5gs.yaml` on the node
   and the EPC container reloaded.

Never hand-edit a rendered production config; treat `open5gs.yaml` in this
repo as the canonical lab source and let the control plane apply overrides.

## Seeding subscribers

```console
# mongo container (compose) - see core/open5gs/hss-init.sh
docker compose -f deploy/docker-compose.yml exec mongo sh /init/hss-init.sh
```

Only dummy lab vectors (sim/test-vectors/lab-vectors.yaml) may be loaded in
the lab. Real SIM material is minted offline by the provisioner and loaded
through its encrypted HSS hook (docs/sim-lifecycle/provisioner.md).
