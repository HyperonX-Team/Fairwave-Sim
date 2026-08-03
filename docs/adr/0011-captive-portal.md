---
title: ADR-0011: Captive Portal Onboarding
---

# ADR-0011: Captive Portal Onboarding (Anchor 302, Not ePDG - Yet)

- Status: Accepted
- Date: 2026-01-15
- Related: ADR-0005 (local breakout), ADR-0012 (config)
- Scope: subscriber-facing onboarding UX for v0.1–M5

## Context

A private network's subscribers need a way to onboard: enroll their device, see a privacy notice, get help, and (in future) authenticate for walled-garden access. Two candidate architectures existed:

1. **ePDG (Evolved Packet Data Gateway)-style onboarding** - a standards-grade path where the UE's data plane is steered into a portal via an Access Point Name. Full ePDG is heavy: additional network functions, certificates, IPSec/IKEv2, and interop quirks with consumer handsets.
2. **Anchor/302 portal** - the network's DNS/edge redirects unauthenticated subscribers' HTTP(S) requests to a portal served by the node, on the local breakout path (ADR-0005). No new core functions; works with any handset that opens a browser.

## Decision

- **v0.1 through M4 ship the anchor/302 portal**: local breakout edge answers the PGW-assigned default route, serves a captive page (privacy notice per [privacy](../security/privacy.md), network info, support links), and records minimal session metadata (hashes only, ADR-0010).
- The portal is part of the control plane stack (`control-plane` service), served at the operator UI's port `8081`.
- **The ePDG interface contract is documented** (in `/design/`) with the goal of an M5 opt-in ePDG path: APN steering, IKEv2/IPsec prerequisites, certificate requirements, and the exact control-plane hooks it must use. Nothing is implemented in v0.1.
- Until ePDG exists, walled-garden policies are enforced at the anchor (allow/deny by subscriber state), not by APN.

## Consequences

Positive:

- Onboarding ships in v0.1 with zero new core functions; the quickstart and lab are unaffected.
- Privacy notice placement is native (portal page) rather than bolted on.
- The ePDG contract gives integrators a stable target and keeps the door open without fake scope.

Negative:

- Anchor 302 only catches HTTP/S traffic to reachable hosts; apps that never resolve an internet host won't see the portal (documented limitation).
- Consumer handsets' captive-portal detection is inconsistent across OEMs - the portal must also work as a plain URL.
- ePDG remains unimplemented; claims of "APN-grade onboarding" are M5+ and clearly labeled as such.

## Alternatives Considered

- **ePDG-first:** rejected - biggest scope item with the least v0.1 user value; interop risk with consumer devices.
- **SMS/USSD onboarding:** rejected - needs an SMSC and roaming-grade signalling; out of scope for private cells.
- **No onboarding surface (UI only):** rejected - privacy notice and walled-garden need a subscriber-facing page.

## Related

- ADR-0005 · [Privacy](../security/privacy.md) · [Roadmap](/design/roadmap.md) (M5: ePDG contract)
