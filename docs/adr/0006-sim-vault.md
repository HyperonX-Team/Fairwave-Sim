---
title: ADR-0006: SIM Credential Vault
---

# ADR-0006: SIM Credential Vault (Ki/OPc at Rest)

- Status: Accepted
- Date: 2025-12-02
- Related: ADR-0010 (privacy logging), ADR-0012 (config format)
- Scope: storage and handling of SIM credentials (Ki, OPc)

## Context

Fairwave's provisioner generates Ki/OPc for every SIM. These are the subscriber's network credentials: whoever holds them authenticates as the subscriber. The project rules are absolute - Ki/OPc are never committed, never logged, never sent in plaintext (see [provisioner](../sim-lifecycle/provisioner.md)). But they must be stored at rest long enough to load into HSS/UDM and to re-issue deterministic bundles for bureaus.

Naive options (plaintext JSON next to the CSV, env-var dump, gitignored-but-readable files) fail the project's own standard. We need real encryption with a key the software cannot invent.

## Decision

- **All SIM credential material is encrypted at rest with AES-256-GCM.**
- The data encryption key (DEK) is per-vault and randomly generated; each record (Ki, OPc) is encrypted with a unique nonce.
- The DEK is wrapped by a per-cluster **Key Encryption Key (KEK)** supplied from environment (`FW_SIM_KEK`) or an HSM-backed key path. The KEK is never written to disk by Fairwave code.
- The vault exposes an **HSM-ready interface** (`VaultProvider` with `Get/Sign/Decrypt` semantics) so deployments can drop in PKCS#11/HashiCorp-backed providers without changing the provisioner, HSS hook, or control plane.
- One-time decryption semantics for bundle transport: a bundle can be decrypted once by the HSS hook; further decrypts require an explicit operator action (audit-logged).
- Key rotation: re-wrapping rotates the DEK; full re-encryption is supported offline (no network) and is the documented procedure for personnel change.

## Consequences

Positive:

- Credential material at rest is unreadable without the KEK, even with full disk access.
- HSM-ready interface lets stricter deployments keep KEKs out of the OS entirely.
- Rotate-and-rewrap workflow gives operators an auditable response to key exposure.
- Nonce-per-record + GCM prevents replay of old ciphertexts.

Negative:

- KEK loss is vault loss: unrecoverable by design. Operators must have a documented KEK backup/ceremony (note in [SIM lifecycle](../sim-lifecycle/index.md)).
- Slight operational friction: every HSS load needs the KEK present (env or HSM) at load time.
- Env-var KEK on a shared host is only as strong as host isolation; docs must say so.

## Alternatives Considered

- **Plaintext files with restrictive permissions:** rejected - violates the no-plaintext-at-rest rule; permissions alone are not crypto.
- **OS keychain / DPAPI-style keying:** rejected - not portable across the x86/ARM Linux node fleet and Dockerized lab.
- **Cloud KMS:** rejected - offline-first is a project requirement (provisioner must work air-gapped); KMS is a fallback provider, not the default.
- **Software-embedded key:** rejected outright - security theater.

## Related

- ADR-0010 · [Provisioner](../sim-lifecycle/provisioner.md) · [Bureau runbook](../sim-lifecycle/bureau-runbook.md)
