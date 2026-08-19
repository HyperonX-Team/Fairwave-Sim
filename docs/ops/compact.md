---
title: Compact Profile (4 GB Laptops)
---

# Compact Profile: the Lab on a 4 GB RAM Laptop

The full no-RF lab (`make lab-up`, docker-compose.yml) expects an 8 GB+ host.
The **compact profile** (`make compact-up`) layers a memory-tuned override on
the same compose file so the whole stack fits comfortably in a 4 GB budget.

## What it changes

`deploy/docker-compose.compact.yml` is layered **on top of** the base compose.
Only these things differ from the full lab:

| Service | Memory limit | Notes |
|---|---|---|
| mongo | 600m | WiredTiger cache also capped in the base compose |
| open5gs | 600m | launches only the 6 attach-path daemons (see below) |
| enb | 1024m | srsENB PHY - the largest single consumer |
| ue1 | 256m | |
| control-plane | 256m | `GOMEMLIMIT`/`GOMAXPROCS` bound the Go heap |

Open5GS normally starts **all eight** daemons (`nrfd mmed sgwcd sgwud smfd
upfd hssd pcrfd`). The compact override sets `FW_DAEMONS` to the six the
4G attach path actually exercises (`mmed sgwcd sgwud smfd upfd hssd`),
omitting `nrfd` and `pcrfd`, which saves several hundred MB of RSS.

## Running it

```bash
make compact-up        # bring up + assert attach, tuned for 4 GB
make compact-status    # compose ps + UE tail
make compact-down      # stop and wipe compact volumes
```

Or directly:

```bash
docker compose \
  -f deploy/docker-compose.yml \
  -f deploy/docker-compose.compact.yml up -d --build
```

## Scope notes

- The compact profile targets the **4G lab** (docker-compose.yml). The 5G SA
  stack (docker-compose.5g.yml) runs ~15 containers and is not the 4 GB path;
  use it on 16 GB+ hosts.
- Build-time parallelism in the Dockerfiles is capped (`-j2`) so building on
  a 4 GB laptop does not OOM the compiler.
- A future memory pass could shrink the srsENB PHY working set (e.g. a
  5 MHz cell, `n_prb = 25` in `core/ran/enb.zmq.yml`) at the cost of attach
  fidelity; the current 10 MHz cell keeps the reference topology.
