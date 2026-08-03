---
title: ADR-0008: TX Spectrum Gate
---

# ADR-0008: TX Gating — Three Independent Layers

- Status: Accepted
- Date: 2025-12-20
- Related: ADR-0002, ADR-0009
- Scope: every path that can cause RF emission from a Fairwave node

## Context

Fairwave is an open-source project whose default stance is "transmits nothing." Real deployments are legal only where the operator holds spectrum rights (licensed, SAS-granted GAA/PPA, or lawful experimental regimes). Software cannot grant rights, but it can make accidental or casual transmission hard and make the gate state inspectable. The risk profile: a community member clones the repo, plugs in an SDR, and keys up on a wrong frequency.

A single toggle is insufficient — the failure of one check must not arm TX.

## Decision

**Real RF is gated by three independent layers; all three must pass before the transmitter can be armed:**

1. **Compile-time gate.** The RF path is excluded from default builds. `FEATURE_RF` (explicit build flag) is required to compile any code that can drive an SDR. Release binaries ship RF-disabled.
2. **Runtime acknowledgment.** Each node carries a persisted country code and a license-acknowledgment record set interactively by the operator (`fairwave tx arm` flow / UI). Acknowledgment is per-node, per-country, time-stamped, and audit-logged.
3. **Frequency allow-list.** Arming requires a validated spectrum profile (JSON-Schema-checked YAML, see ADR-0012 and `/design/spectrum-matrix.md`) whose channels include the exact EARFCN(s) in use for the node's country. Unknown channels → not allowed.

Enforcement points: `POST /v1/spectrum/check` returns per-gate verdicts; `POST /v1/tx/arm` refuses with `409` unless all three pass; the agent re-evaluates the gates on config reload and disarms on any change. `GET /v1/tx/arm` exposes gate state read-only.

## Consequences

Positive:

- Default build is physically incapable of transmitting; the claim is checkable by code inspection.
- Compromise of one layer (e.g. a forged config) does not arm TX; two layers must fail.
- Gate state is public and auditable — regulators and MNOs can verify (see [regulator FAQ](../reference/faq-regulators.md)).
- Re-evaluation on reload catches config drift automatically.

Negative:

- Operators must deliberately re-arm after every config change touching radio — friction by design.
- Compile-time flag complicates "SDR on laptop" experiments slightly (they need a custom build — correct).
- The gate does not enforce *legal* use — it only forces informed, deliberate actions. Docs must keep saying so.

## Alternatives Considered

- **Single runtime flag:** rejected — one bit flipped by a script or misconfig arms TX.
- **Hardware fuse/dongle:** rejected — non-portable, prevents legitimate research builds.
- **Cloud authorization service (SAS-style check-in):** rejected — offline-first requirement; SAS compliance itself remains the operator's obligation (CBRS: optional-but-mandatory-by-rules GAA check-in is out of Fairwave's scope).
- **No gate at all (trust operators):** rejected — conflicts with the project's public-safety stance and regulator questions.

## Related

- [Spectrum matrix](/design/spectrum-matrix.md) · [Customization](../tutorials/customization.md) · ADR-0002 · ADR-0009
