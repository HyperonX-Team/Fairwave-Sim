---
title: ADR-0007: Operator Authentication
---

# ADR-0007: Operator Authentication (WebAuthn + TOTP + Bootstrap Tokens)

- Status: Accepted
- Date: 2025-12-15
- Related: ADR-0010, ADR-0012
- Scope: human access to the control plane, CLI, and operator UI

## Context

The control plane can mint SIMs, arm TX, revoke peers, and read sessions - a powerful and sensitive surface. Password-based auth is the wrong default for community-operated nodes: passwords are shared, phishable, and stored badly. We also need a first-enrollment path that does not ship with a default credential, and a fallback for operators whose devices cannot do WebAuthn. See [operator auth](../security/operator-auth.md) for the operational page.

## Decision

- **WebAuthn/FIDO2 passkeys are the primary operator authentication factor.** The control plane stores only public keys and credential IDs; no secrets at rest.
- **TOTP (RFC 6238) is the fallback second factor** where WebAuthn is unavailable; no static passwords are ever accepted.
- **Bootstrap tokens provide first enrollment only**: admin-minted, single-use, TTL-bounded (default 15 min, hard max 24 h), role-scoped. They are consumed by the first successful WebAuthn registration and cannot be reused or extended.
- Sessions are short-lived JWTs (15 min) with device-bound refresh tokens (30-day sliding cap), idle timeout 30 min, and kill-anywhere (`logout --all`).
- Authorization is server-side RBAC (viewer / operator / sim:admin / node:admin / auditor); roles attach to sessions, not tokens.
- Every auth event (invite, register, login, logout, role change, lockout) is written to the append-only audit log with principal, role, session id, and source.

## Consequences

Positive:

- Phishing resistance where passkeys are used; no credential database to leak.
- Token exposure is bounded by single-use + short TTL; session theft is mitigated by device-bound refresh tokens.
- RBAC gives community operators a defensible separation (e.g. auditor without session data).
- Audit log makes enrollment and lockout events reconstructible.

Negative:

- WebAuthn needs a supporting browser/OS; older kiosk setups fall back to TOTP with a worse security story.
- Lost passkey = admin-mediated re-enrollment (documented procedure).
- Token TTL friction on slow ops teams (re-invite required) - accepted as a feature.

## Alternatives Considered

- **Username/password + TOTP:** rejected - password database remains the attack surface and phishing vector.
- **Client certificates only:** rejected - good for nodes/CLI (mTLS), impractical for human browser sessions.
- **Magic-link email auth:** rejected - email is out of scope for offline-first nodes and creates dependency on mail infra.
- **Unlimited-lifetime bootstrap tokens:** rejected - contradicts the threat model (credential-dump scenarios).

## Related

- [Operator auth](../security/operator-auth.md) · [Security overview](../security/index.md) · ADR-0010 · ADR-0012
