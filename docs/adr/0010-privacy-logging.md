---
title: ADR-0010: UE Identifier Minimization
---

# ADR-0010: UE Identifier Minimization (IMSI Never Logged)

- Status: Accepted
- Date: 2026-01-08
- Related: ADR-0006 (SIM vault), ADR-0007 (operator auth)
- Scope: all logging, metrics, dashboards, audit trail, and API responses

## Context

The EPC must reference subscribers (sessions, SIM states, attach events), and operations need to correlate records. But raw IMSI is PII-class data: it identifies a subscriber globally and is exactly what IMSI-catcher-style systems harvest. Fairwave's stated position ([privacy](../security/privacy.md), [regulator FAQ](../reference/faq-regulators.md)) is that subscriber identifiers in observability are minimized.

Full IMSI in logs was a risk; hashing with a full SHA-256 was a usability risk (operators cannot eyeball-correlate and hash-catalog attacks against full 256-bit hashes of a small IMSI space are trivially brute-forceable given any side channel). We need a scheme that is *practically* useful to the operator and *deliberately* not a universal key.

## Decision

- **Raw IMSI is never written to logs, metrics, dashboards, API responses, or audit records.** (Ki/OPc were already banned by ADR-0006.)
- **The sole identifier in observable output is the truncated hash:** `sha256(imsi)`, first **12 lowercase hex characters** (48 bits), e.g. `9f2c41b07d3a`.
- Where the control plane must address a record internally (HSS keys, vault), the full credential lives in the encrypted vault (ADR-0006); observability layers only ever see the truncation.
- Session records, `/v1/sims`, `/v1/sessions`, metrics labels, and audit entries all use the 12-hex form. The provisioning CLI prints the hash, not the IMSI, unless `--show-imsi` is explicitly passed for a locally-supervised operation.
- `fairwave doctor --privacy` lints config and log sinks for cleartext IMSI patterns and flags violations.

## Consequences

Positive:

- An exposed log set does not enumerate subscribers; correlation to a real person requires knowledge of the IMSI (and building a local mapping — which is the operator's lawful metadata, not the project's).
- 12-hex truncation is short enough to paste/eyeball and long enough to make casual brute-forcing over the full IMSI space expensive-ish, while *documented* as not a security boundary for a determined adversary.
- Regulators and carriers get a concrete, auditable claim: "no cleartext IMSI in observability."

Negative:

- Operators must fetch IMSI ↔ hash mapping from the vault for forensics; that mapping is itself sensitive.
- Some EPC internals (Open5GS MME logs) will still emit IMSI because we do not control third-party logs; we document suppression recipes rather than promise perfection.
- Truncation is a minimization technique, not anonymity: we say so explicitly everywhere the hash appears.

## Alternatives Considered

- **Full SHA-256 of IMSI:** rejected — no operational difference for legitimate correlation, and implies stronger identity guarantees than it delivers.
- **Random per-SIM opaque IDs:** rejected — breaks cross-record correlation that operators need (same SIM across sessions).
- **Plaintext IMSI in internal-only logs:** rejected — logs leak through copy/paste and support requests; the rule must be structural, not "internal only".
- **No identifiers at all:** rejected — operations (revocation, troubleshooting) become impossible.

## Related

- [Privacy](../security/privacy.md) · [Regulator FAQ](../reference/faq-regulators.md) · ADR-0006 · ADR-0007
