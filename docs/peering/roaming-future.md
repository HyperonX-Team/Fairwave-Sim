---
title: Roaming and Multi-Operator Interconnect (Future)
---

# Roaming and Multi-Operator Interconnect: Future Work

This page is deliberately unambitious. Fairwave v0.1.0 does **not** do inter-operator roaming, and the roadmap does not claim it. Here is the honest state and what it would take.

## What exists today

- **Intra-mesh peering**: UEs of one operator traverse another operator's nodes only within a single administrative mesh, under explicit policy (see [peering](index.md)). This is not roaming — no settlement, no inter-operator identities.
- **PLMN separation**: each deployment serves its own PLMN; there is no N26/interface exchange between PLMNs.
- **No SEPP**: the Security Edge Protection Proxy (5G) or GRX/IPX connectivity required for real-world inter-operator signalling does not exist in the codebase.

## What "roaming" would require (honest list)

1. **Contractual basis.** Roaming is a commercial and legal arrangement: settlement agreements, fraud handling, LI obligations, data protection (GDPR/DPA scope). Software cannot supply this.
2. **SEPP or MAP/GRX interconnects.** For LTE: S6a/S8 over a trusted GRX/IPX exchange with certificate chains (STKS, network certificates). For 5G: SEPP with security parameter negotiation (PRINS). Fairwave has neither implemented nor certified.
3. **HSS/UDM federation.** S6a/S13d reachability, roaming IMSI ranges (MCC/MNC of visited networks), VLR/MME addressing, and AVP-level compatibility with host network elements.
4. **Charging and settlement.** Offline charging (CGa/CDR export) and inter-operator TAP/NA records — none shipped.
5. **LI and lawful access.** Host-country requirements for retained data and interception of roaming subscribers must be met by *both* operators; see the [regulator FAQ](../reference/faq-regulators.md).
6. **Testing.** Actual interop testing with a real MNO's core or a certified test lab — not simulated.

## Roadmap posture

| Milestone | Scope |
| --- | --- |
| v0.1 (lab) | No roaming. Mesh peering within one admin domain only. |
| M4 | Interchange formats documented (CDR export schema) for community settlement experiments. |
| M5 | SEPP interface contract documented as an integration point (ADR-0011 style contract), no implementation committed. |
| M6+ | Only with a licensed partner: pilot of one-to-one roaming via a certified IPX provider. |

Nothing in M0–M6 implies roaming is shipped. If a page or talk says otherwise, it is wrong.

## What Fairwave-compatible boxes can do lawfully today

- **Offload / neutral host experiments**: one operator's node serving subscribers of another *only* through lawful MVNO-style arrangements (subscribers hosted in the local HSS under contract). This is the realistic path: local breakout with hosted identities, not inter-operator signalling.
- **Community settlement prototypes**: CDR-style accounting via the sessions API for voluntary clearing between community operators — data-plane settlements, no core-to-core interconnect.

## If you are an MNO reading this

- Fairwave meshes are private networks, not a roaming threat; there is no inter-PLMN path out of the box (see [carrier FAQ](../reference/faq-carriers.md)).
- The integration surfaces that exist — Open5GS-standard S6a/API, documented CDR exports in M4 — are the only interop points.

## Related

- [Peering overview](index.md) · [Carrier FAQ](../reference/faq-carriers.md) · [Not Fairwave](../reference/not-fairwave.md)
