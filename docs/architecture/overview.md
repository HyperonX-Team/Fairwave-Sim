---
title: Architecture Overview
---

# Architecture Overview

Fairwave is an open-source community small-cell: a "carrier in a pizza box." An x86/ARM mini-PC drives a software-defined radio and runs the LTE core, RAN, control plane, and operator tooling as containers. One box is a private network. Several boxes, joined over a mesh, become a neighborhood network.

The default mode is **lab/no-RF**: srsRAN runs on the `zmq` virtual radio and srsUE replaces physical handsets, all inside Docker Compose. Real RF is disabled until a country code, license acknowledgment, and an allow-listed band are configured (the `tx/arm` gate).

## System Diagram

```mermaid
flowchart LR
    UE["UE / srsUE (lab)"] -->|LTE-Uu| eNB["srsRAN eNB"]
    eNB -->|S1-MME / S1-U| EPC["Open5GS EPC"]
    EPC --> BRK["Local Breakout NAT"]
    BRK --> NET["Internet / LAN"]

    CTRL["fairwave-control"] -->|reconcile| eNB
    CTRL -->|reconcile| EPC
    CTRL -->|northbound REST /v1| UI["Operator UI"]
    CTRL -->|northbound REST| CLI["fairwave-cli"]
    AGENT["fairwave-agent"] -->|heartbeat + metrics| CTRL
    AGENT -->|probes| HW["SDR / GPSDO / thermals"]
    PORTAL["Captive Portal"] --> UE

    PEER["Peer box"] <-->|mDNS + mTLS + WireGuard| CTRL
```

## Component Table

| Component | Role | Runs as |
|---|---|---|
| srsRAN eNB | LTE radio: LTE-Uu toward UEs, S1 toward EPC | container |
| srsRAN srsUE | Synthetic UE for lab tests (zmq loopback) | container |
| Open5GS EPC | MME, SGW, PGW, HSS, PCRF | containers (one per service) |
| fairwave-control | Identity, enrollment, reconcile loop, REST API v1 | container / systemd |
| fairwave-agent | On-box health probes, heartbeat, watchdog | systemd |
| fairwave-cli | Operator commands against the REST API | binary |
| Operator UI | Local-first dashboard | container |
| Captive portal | Onboarding for non-cellular devices / Wi-Fi calling | container |

## Data Plane vs Control Plane

- **Data plane:** UE ↔ eNB (LTE-Uu) ↔ S1-U ↔ SGW/PGW ↔ local breakout NAT ↔ Internet. User traffic never hairpins through a cloud.
- **Control plane:** NAS/S1AP signaling stays inside the box; `fairwave-control` reconciles configuration into Open5GS and srsRAN; policy (APN, breakout, band allow-list) lives in the control plane.
- **Management plane:** `fairwave-agent` → `fairwave-control` → CLI/UI, all on-box; peer boxes connect over an mTLS mesh with a WireGuard data plane.

## Lifecycle Phases

```mermaid
stateDiagram-v2
    [*] --> provision
    provision --> register: node init + enrollment
    register --> on_air: tx/arm gate cleared
    on_air --> peer: mesh join
    peer --> breakout: hub / NAT route
    peer --> register: rollback
```

`provision → register → on-air → peer → breakout`. The transition API is `lifecycle/transition` under `/v1`; each phase is gated by prerequisite state (see `docs/architecture/control-plane.md`).

## Defaults

- PLMN `999-99` (lab), TAC `7`, APNs `internet` and `ims`.
- Local breakout (edge NAT) by default; WireGuard tunnel optional (`docs/adr/0005-local-breakout-default.md`).
- Release `v0.1.0` targets lab mode; milestones M0–M6 in `design/roadmap.md`.

## Related

- Control plane: `docs/architecture/control-plane.md`
- Core network: `docs/architecture/mobile-core.md`
- Radio: `docs/architecture/ran.md`
- Security model: `docs/architecture/security.md`, `design/threat-model.md`
