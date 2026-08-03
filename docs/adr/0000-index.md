---
title: Architecture Decision Records
---

# ADR index

Fairwave records significant architectural decisions as ADRs (Michael Nygard format, kept short).
An ADR is immutable once accepted; superseding decisions get a new ADR that links back.

| # | Title | Status | Date |
|---|-------|--------|------|
| [0001](0001-open5gs-vs-magma.md) | Mobile core: Open5GS instead of Magma | Accepted | 2026-08-02 |
| [0002](0002-4g-first.md) | Ship 4G EPC first; gate 5G SA/NSA | Accepted | 2026-08-02 |
| [0003](0003-control-plane-language.md) | Control plane language: Go (not Rust) | Accepted | 2026-08-02 |
| [0004](0004-wireguard-vs-ipsec.md) | Peering data plane: WireGuard | Accepted | 2026-08-02 |
| [0005](0005-local-breakout-default.md) | Breakout: edge NAT local-first, hub optional | Accepted | 2026-08-02 |
| [0006](0006-sim-vault.md) | SIM credential storage: HSM-ready AES-GCM at rest | Accepted | 2026-08-02 |
| [0007](0007-operator-auth.md) | Operator auth: WebAuthn + TOTP + bootstrap token TTL | Accepted | 2026-08-02 |
| [0008](0008-spectrum-gate.md) | TX gating: compile-time flag + runtime ack + allow-list | Accepted | 2026-08-02 |
| [0009](0009-rf-backend.md) | RF backend: srsRAN Project 4G-first, OAI as alt | Accepted | 2026-08-02 |
| [0010](0010-privacy-logging.md) | UE identifier minimization: hash-at-rest, no IMSI in logs | Accepted | 2026-08-02 |
| [0011](0011-captive-portal.md) | Onboarding: captive portal anchor, not an ePDG | Accepted | 2026-08-02 |
| [0012](0012-config-format.md) | Config: three-file YAML + env; jsonschema validation | Accepted | 2026-08-02 |

## Conventions

Each ADR has **Context → Decision → Consequences → Alternatives considered**.
Rejected-but-plausible options stay written down so the next contributor doesn't re-litigate them.
