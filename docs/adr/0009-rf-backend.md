---
title: ADR-0009: RF Backend - srsRAN Project
---

# ADR-0009: RF Backend - srsRAN Project, 4G-First, ZMQ Lab Default

- Status: Accepted
- Date: 2025-12-22
- Related: ADR-0002 (4G first), ADR-0008 (TX gate)
- Scope: choice of radio software stack (eNB/UE) and lab radio transport

## Context

Fairwave needs an eNB (and a UE for testing) that: speaks LTE against Open5GS (ADR-0001), supports the SDR families we target (USRP, LimeSDR, BladeRF), and supports a no-RF lab path. The candidate ecosystems are srsRAN Project, OAI (OpenAirInterface), and (for the lab only) custom virtual-radio stubs.

srsRAN Project: active, permissive (AGPL-3.0), mature LTE eNB/gNB + srsUE, first-class ZMQ virtual radio support, native UHD/LimeSuite/BladeRF drivers. OAI: strong 5G research momentum, but heavier build/config surface, historically rockier LTE-as-a-service story for small deployments.

## Decision

- **srsRAN Project is the RF backend** for eNB (srsENB) and test UE (srsUE) in v0.1 and through M6, LTE-only per ADR-0002.
- **ZMQ virtual radio is the lab default**: eNB ↔ UE run over ZeroMQ TCP sockets in Docker; no SDR, no emissions. `make lab-up` boots exactly this.
- The SDR driver layer is runtime-selected (UHD / LimeSDR / BladeRF) behind the same interface the ZMQ transport uses; TX gating (ADR-0008) applies identically regardless of driver.
- OAI remains a documented alternative for operators who need it (interop experiments), but it is not packaged or supported in releases.

## Consequences

Positive:

- One stack to test end-to-end in CI (ZMQ in Docker makes attach tests hermetic and fast).
- The lab and prod radio paths share config surface (EARFCNs, bands, TAC) - customization doc stays single-sourced.
- srsRAN's active community matches Fairwave's maintenance model; bugs surface early.
- ZMQ lab means the quickstart is genuinely no-RF and runs anywhere with Docker.

Negative:

- AGPL-3.0 of srsRAN requires network-service deployments to share modifications if offered to users - acceptable for an AGPL project itself, but documented for commercial operators.
- srsENB/srsUE maturity lags 5G paths (irrelevant while ADR-0002 holds).
- OAI interop gaps cannot be supported by us; they are the operator's experiment.

## Alternatives Considered

- **OAI as primary:** rejected for v0.1 - heavier ops surface, weaker out-of-box LTE/EPC fit for our audience; re-evaluate only if a 5G milestone materializes (and see ADR-0002's flag).
- **Custom minimal virtual radio instead of ZMQ:** rejected - ZMQ is maintained upstream, realistic (real IQ over TCP), and already how srsRAN tests itself.
- **Third-party bundled binaries (e.g. Amarisoft):** rejected - licensing costs and closed core contradict the open-source project.

## Related

- ADR-0001 · ADR-0002 · ADR-0008 · [Lab attach tutorial](../tutorials/lab-attach.md) · [Roadmap](/design/roadmap.md)
