---
title: ADR-0002: 4G First, 5G Gated
---

# ADR-0002: Ship 4G EPC First; Gate 5G SA/NSA Behind Flags

- Status: Accepted
- Date: 2025-11-14
- Related: ADR-0001 (Open5GS vs Magma), ADR-0009 (RF backend)
- Scope: radio access generation strategy for v0.1–M6

## Context

Fairwave's target users — community networks, labs, private small cells — overwhelmingly operate (or can lawfully operate) 4G LTE spectrum today: CBRS GAA in the US, ISM-adjacent and licensed-private LTE bands elsewhere. 5G SA/NSA brings higher complexity (NR NRU/FR2 configs, SBA core, new gNB stacks) and no widening of lawful-spectrum access for these users. The team's core competency and the Open5GS/srsRAN ecosystem (ADR-0001, ADR-0009) are LTE-mature and NR-immature.

## Decision

- **v0.1 and all M0–M6 milestones ship LTE/EPC (4G) as the only radio access.**
- 5G SA (gNB + 5G core path) and NSA are **compile-time/feature-flag gated**; no NR paths exist in the default build.
- The flag (`FEATURE_NR`) is off in release binaries. Enabling it requires a deliberate, documented build and is not part of any supported milestone.
- All subscriber, SIM, peering, and privacy machinery (vault, hashes, provisioning) is generation-agnostic by design, so a future 5G path reuses it unchanged.

## Consequences

Positive:

- One radio stack to test; the attach/regression surface stays small (lab → M6 hardening is tractable).
- CBRS GAA and private-LTE use cases are fully served by v0.1; nothing users need today is deferred.
- SIM/HSS/privacy infrastructure does not fork between generations.
- Flag-gated NR prevents accidental NR builds from being mistaken for supported software.

Negative:

- Operators with genuine 5G needs are out of scope until further notice.
- NSA (LTE anchor + NR) is the only realistic near-future path and even that stays flagged.
- Some marketing confusion: "small-cell" is sometimes assumed to mean 5G; docs must state the boundary (see [not-fairwave](../reference/not-fairwave.md)).

## Alternatives Considered

- **5G-first (NR SA):** rejected — no lawful-spectrum case for target users, ecosystem immature, scope blowup before v0.1.
- **Dual-stack from day one:** rejected — doubles test matrix and integration cost for zero v0.1 user value; flags allow later without re-architecture.
- **NSA-only (LTE anchor + NR data):** deferred — requires 5G core elements (AMF/UDM paths) that milestone M0–M6 does not schedule; revisit only with a licensed partner.

## Related

- ADR-0001 · ADR-0009 · [Roadmap](/design/roadmap.md) · [Glossary](../reference/glossary.md)
