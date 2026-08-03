---
title: ADR-0012: Configuration Format
---

# ADR-0012: YAML Configuration with JSON Schema Validation

- Status: Accepted
- Date: 2026-01-20
- Related: ADR-0008 (spectrum profiles), ADR-0006 (vault)
- Scope: all node, EPC, radio, spectrum, and policy configuration

## Context

Fairwave configuration spans node identity, EPC (PLMN, TAC, APNs), radio (bands, EARFCNs), spectrum profiles, policy, and peering. Early prototypes used a mix of JSON, YAML, env vars, and a custom config loader — typos produced runtime failures and spectrum profiles could not be validated before arming TX (ADR-0008 depends on validated profiles). Secrets were occasionally present in configs (violating ADR-0006 and the "no secrets committed" rule).

## Decision

- **All configuration is YAML**, validated against **JSON Schema** at load time and again in CI (`make config-lint`).
- Schema lives in `config/schemas/*.schema.json`; a schema version (`schemaVersion`) is mandatory at the top of every config file; loaders reject unknown versions (forward-incompatible changes bump the version).
- **Environment overrides** replace values without editing files: prefix `FW_`, nested keys joined with `_` (e.g. `network.plmn.mcc` → `FW_NETWORK_PLMN_MCC`). Overrides do not bypass schema validation (they are validated as a merged document).
- **Secrets are never stored in config files.** Credential-bearing fields (KEK paths, vault slots, bundle keys) reference env vars or HSM slots only; the linter fails on in-file secrets (key-like patterns).
- Load failures are fatal: a node refuses to start on invalid config rather than degrade.

## Consequences

Positive:

- Typos and type errors surface at load and in CI, not in production.
- Spectrum profiles are machine-checkable before TX arming — ADR-0008's allow-list relies on this.
- `FW_*` overrides make containers parameterizable without config mounts (compose-friendly, matches [customization](../tutorials/customization.md)).
- The no-secrets rule is enforced by lint, not convention.

Negative:

- YAML+JSON Schema adds a schema maintenance burden; each config surface change needs schema review.
- Env-var override precedence rules need documentation (they are: file < env < CLI flags, unless schema forbids).
- Some contributors prefer TOML; we standardize on YAML for consistency with the rest of the ecosystem (docker-compose, srsRAN examples).

## Alternatives Considered

- **TOML:** rejected for consistency; ecosystem convention wins.
- **Raw env-only configuration:** rejected — unreadable at scale, unvalidatable, and hostile to spectrum-profile review.
- **Config-as-code (Go structs + tags):** rejected — schema-lintable YAML keeps non-developers (community ops) able to review diffs.
- **Multiple accepted formats:** rejected — single format keeps validation and docs single-sourced.

## Related

- [Customization tutorial](../tutorials/customization.md) · [Spectrum gate ADR](0008-spectrum-gate.md) · [SIM vault ADR](0006-sim-vault.md) · [Troubleshooting](../reference/troubleshooting.md)
