---
title: Hydra - Cooperative Bearer Multiplexing
---

# Hydra: many SIMs, one link

!!! info "Lab-safe by default"
    Hydra moves no RF. It is a control-plane and data-plane multiplexing
    subsystem: in the lab it is driven over ZMQ and the HTTP bench, and on
    real hardware the same calls sit behind the GTP-U datapath. Nothing in
    Hydra bypasses the [spectrum gate](security.md) or the TX arm rules.

## The problem Hydra solves

A single SIM in a single modem has a ceiling. A 20 MHz LTE carrier tops out
near 100 Mbps; a USRP B200 cannot host the 100 MHz NR carrier that reaches a
gigabit. The usual answer is "buy a faster link". Hydra takes the other path:
**if the core owns the identity and the radio, it can multiplex one
subscriber across many independent threads and reassemble the flow in order.**

- One box: ~300 Mbps (its own modem).
- Two boxes: ~600 Mbps.
- Four boxes: **~1.2 Gbps**.
- Eight boxes: **~2.4 Gbps**.

The ceiling stops being any single radio and becomes **the sum of the
threads**, until the anchor's uplink saturates.

## Concepts

| Term | Meaning |
|---|---|
| **Thread** | One independent SIM + modem + path. The unit of capacity. |
| **Weave** | A set of threads bound to one anchor, carrying one logical flow. |
| **Anchor** | The node with real transit. It reassembles the weave and egresses in order. |
| **Shim** | The 24-byte per-packet sequencing header that makes reassembly possible. |

## How it works

```
        UE sees ONE flow (standard TCP/QUIC - no client changes)
                              │
  [UE]──NR──>[Box A radio]────┤
                              │      HYDRA FABRIC (over the WireGuard mesh)
  [SIM2][Modem2][Box B]───────┼──► [ Weave Anchor ] ──► Internet
  [SIM3][Modem3][Box C]───────┤      reorder + egress
  [SIM4][eSIM profile 4]──────┘
```

1. **Strip.** Each outbound packet gets a monotonic sequence number and a
   shim header, then is assigned to a thread.
2. **Schedule.** The **delay-inflation scheduler** picks the thread whose
   *projected completion time* is earliest, given queue depth, rate, loss,
   and one-way delay. Because it minimises completion time rather than RTT,
   it maximises goodput while keeping consecutive packets close in time -
   which keeps the reorder window shallow.
3. **Reassemble.** The anchor feeds inbound frames into a bounded sliding
   window and releases every contiguous payload in order.
4. **Egress.** Reassembled payloads leave the anchor as one ordinary flow.

### Why this is only possible for Fairwave

Consumer link bonding is a terms-of-service violation because the user owns
neither the SIMs nor the core. Fairwave issues its own SIMs
(`core/sim-ops`) and runs its own core (Open5GS/freeGC). Multiplexing its
own credentialed bearers is simply its own network using its own
credentials.

## Packages

| Package | Responsibility |
|---|---|
| `core/hydra/shim` | Fixed-width sequencing header (encode/decode). |
| `core/hydra/sched` | Delay-inflation scheduler (projected completion time). |
| `core/hydra/reorder` | Bounded sliding-window reassembler + stats. |
| `core/hydra/bearer` | Thread registry, capacity aggregation, queue accounting. |
| `core/hydra/fabric` | Weave engine: strip, reassemble, ingest, bench, status. |
| `core/hydra/anchor` | Egress interface (real uplink, or a counting sink in the lab). |

## CLI

```bash
# Declare three threads (three SIMs on three boxes).
fairwave hydra thread-add --id sim-a --box box-a --mbps 300 --rtt 18 --up
fairwave hydra thread-add --id sim-b --box box-b --mbps 300 --rtt 22 --up
fairwave hydra thread-add --id sim-c --box box-c --mbps 300 --rtt 30 --up

# Weave them into one logical link at the anchor.
fairwave hydra weave-create --id gig --anchor hub --threads sim-a,sim-b,sim-c

# Prove the ceiling became the sum.
fairwave hydra bench gig

# Inspect.
fairwave hydra status
fairwave hydra weave-stats gig
```

`fairwave hydra bench` reports the single-thread capacity, the aggregate,
and the speedup:

```
hydra bench: gig
  packets:        300 (367200 bytes)
  delivered:      300
  single thread:  300 Mbps
  aggregate:      900 Mbps
  speedup:        3.00x
  reorder events: 150 (max depth 150)
```

## REST surface

Mutations are operator-role; reads are viewer-role.

| Method | Path | Purpose |
|---|---|---|
| GET | `/v1/hydra/status` | Weave summary |
| GET/POST | `/v1/hydra/threads` | List / add threads |
| DELETE | `/v1/hydra/threads/{id}` | Remove a thread |
| GET/POST | `/v1/hydra/weaves` | List / create weaves |
| GET/DELETE | `/v1/hydra/weaves/{id}` | Get / delete a weave |
| GET | `/v1/hydra/weaves/{id}/stats` | Live weave counters |
| POST | `/v1/hydra/weaves/{id}/strip` | Lab: strip one payload |
| POST | `/v1/hydra/weaves/{id}/ingest` | Lab: reassemble one frame |
| POST | `/v1/hydra/weaves/{id}/bench` | Synthetic bench |

Metrics: `fairwave_hydra_threads`, `fairwave_hydra_threads_up`,
`fairwave_hydra_weaves`, `fairwave_hydra_aggregate_mbps`.

## Honest limits

- **Reorder costs memory and a little latency.** The window is bounded; a
  packet beyond it is dropped rather than stalled.
- **Heterogeneous NATs need the anchor.** The anchor is a mesh member with
  real transit; the existing WireGuard fabric carries the threads.
- **One thread is one ceiling.** Hydra cannot beat physics - it beats islands.
- **Carrier terms of service** only bind when you do not own the SIMs. On a
  Fairwave deployment you do, which is what makes weaving lawful here.

## Related

- Control plane: [control-plane](control-plane.md)
- Peering (the thread transport): [peering](peering.md)
- SIM operations: [SIM lifecycle](../sim-lifecycle/index.md)
