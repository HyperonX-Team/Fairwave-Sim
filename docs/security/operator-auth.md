---
title: Operator Authentication
---

# Operator Authentication: Tokens, Passkeys, TOTP, RBAC

Operator access to the control plane is the crown-jewel surface: it can mint SIMs, arm TX, revoke peers, and read sessions. ADR-0007 sets the direction: **no shared passwords; WebAuthn passkeys primary, TOTP fallback, bootstrap tokens only for first enrollment with a short TTL.**

## Bootstrap tokens: first enrollment

A new operator cannot authenticate with anything until enrolled. Enrollment flow:

1. A `sim:admin` mints a token:

```bash
fairwave operator invite --email alice@example.org --role operator --ttl 15m
```

```
token: fw-op-3f9c... (expires in 15 minutes, single use)
```

2. Alice calls the control plane with the token; the response is a WebAuthn registration challenge (no password is ever set).

3. Alice's passkey (hardware or platform authenticator) is registered; the token is invalidated on use.

Rules:

- Default TTL 15 minutes, hard max 24 h; tokens are single-use and scoped to the role in the invite.
- Issuance and use are both audit-logged (principal, token hash, role, IP).
- No API exists to extend a token; re-invite instead.
- Invites cannot be revoked via CLI before expiry; they are single-use and TTL-bounded.

## WebAuthn passkeys (primary)

- Registration and assertion use WebAuthn/FIDO2; the control plane stores only the public key and credential ID - no secrets at rest.
- Resident credentials (discoverable) preferred; device-bound passkeys encouraged for `sim:admin`.
- Fallback if a browser/OS cannot do WebAuthn: TOTP enrollment (below) - never a static password.

## TOTP (fallback)

- TOTP (RFC 6238, 30 s window, 8 digits, SHA-1 with per-account secret) is available as a second factor or as the sole factor where WebAuthn is impractical.
- Secrets are stored hashed (HMAC of the secret with the cluster KEK); QR provisioning only at enrollment time.
- Failed-verification rate limit: 5 attempts / 5 min per account, then lockout requiring admin reset (audit-logged).

## Session handling

- **Access tokens:** short-lived JWTs (15 min) signed by the control plane key; refreshed with a refresh token bound to the session (30 day max, sliding).
- **Sessions list:** the operator UI lists active sessions; an admin can kill any session from there (there is no CLI equivalent - the CLI manages API tokens with `fairwave token create|list|revoke`).
- **Idle timeout:** 30 minutes for UI sessions, 15 minutes for API-only.
- **Device binding:** refresh tokens are bound to the client's key (DPoP-style); token theft without the key is useless.
- **Logout everywhere:** the UI's sign-out-everywhere invalidates all sessions for the account; CLI-issued tokens are revoked individually with `fairwave token revoke <id>`.

## RBAC roles

| Role | Capabilities (prefix grants) | Example actions |
| --- | --- | --- |
| `viewer` | read-only on status/peers/sessions/metrics | Watch dashboard |
| `operator` | viewer + sessions, policy, spectrum check, sim issue/revoke (non-admin) | Run lab, issue SIMs |
| `sim:admin` | operator + sim wipe, profile changes, range revokes | Manage SIM ranges |
| `node:admin` | operator + node join/leave/revoke, wg rotate, mesh CA ops | Run the mesh |
| `auditor` | read-only + audit log, no session data | Compliance |

Role enforcement is server-side on every API call (`/v1/...`); UI hiding is cosmetic only. Role changes require the change actor to be `sim:admin` or higher and are audit-logged.

## Audit log

Every sensitive action appends an entry (see [revocation audit](../sim-lifecycle/revocation.md) for an example):

- What: action + target (hashed identifiers).
- Who: principal + role at time of action.
- When: monotonic timestamp.
- Where: source IP + session id.
- Result: applied / rejected / rate-limited.

Retention: 400 days default (configurable), exportable as JSONL by auditors only. The audit log itself is append-only (read replica + hash chain per day).

## Failure modes

| Scenario | Behaviour |
| --- | --- |
| Lost all passkeys | Admin re-invites; TOTP remains as fallback; identity re-verified out of band |
| Account lockout after TOTP attempts | Admin reset, audit entry, forced passkey re-registration |
| Token leaked | Single-use + TTL bounds exposure; `fairwave token revoke <id>` for issued ones |
| Session stolen | Device-bound refresh token; kill the session from the UI (no CLI equivalent) |

## Related

- ADR-0007 · [Security overview](index.md) · [REST auth](../api/index.md#authentication)
