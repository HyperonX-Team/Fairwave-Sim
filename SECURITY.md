# Security Policy

> [!IMPORTANT]
> **Legal banner** — Fairwave defaults to lab/no-RF mode. Transmitting on cellular
> bands without authorization is illegal in most jurisdictions. You are solely responsible
> for licenses, SAS grants, indoor restrictions, and type approval. HyperonX and
> contributors provide software as-is for lawful private networks, research, and
> shared-spectrum regimes only.

## Supported Versions

Security fixes are provided for the latest minor release branch. Old branches receive
fixes only for remote code execution or SIM-credential compromise.

| Version | Supported |
|---------|-----------|
| 0.1.x   | yes (lab) |
| <0.1    | no        |

## Reporting a Vulnerability

**Do NOT open a public issue for security vulnerabilities.**

- Email: **security@hyperonx.io** (PGP key in [.well-known/security.txt](website/public/.well-known/security.txt))
- GitHub private vulnerability reporting: https://github.com/HyperonX-Team/Fairwave-Sim/security/advisories/new

You should receive acknowledgement within **72 hours**. We will work with you to
coordinate disclosure.

## Safe Harbor

Good-faith security research against **your own authorized lab/network** is safe. Do not
test against public operator networks. Any activity involving unauthorized radio
transmission, interception of traffic, or IMSI catching is out of scope and will be
reported as required by law.

## What we consider security bugs

- Remote code execution in fairwave-control, fairwave-agent, or the operator portal
- Bypass of TX gating / country-code allow-list logic
- Exposure or leakage of Ki/OPc (even hashed) from logs, metrics, or API responses
- Cleartext credentials on the host filesystem outside `secure/` volumes
- Default passwords or backdoor accounts in `deploy/` images
- Peering mTLS/WireGuard authentication bypass

## What is explicitly out of scope

- Attacks requiring physical access to the SIM/HSM (except stopgap notes)
- Attacks requiring RF jamming / close-range radio spoofing (unless software-side)
- Vulnerabilities in Open5GS/srsRAN themselves (report upstream; we track them)
- Findings requiring the lab-only PLMN `999-99` in production contexts

## Hardening baseline

- All control-plane APIs require mTLS or JWT with rotation
- SIM credentials encrypted at rest (NaCl secretbox over per-cluster KEK)
- Zero IMSI in logs: we log IMSI-SHA256(truncated 12) only
- Configuration files are minimal: no secrets in YAML, env-only references
- Releases signed with Cosign; SBOM (SPDX/CycloneDX) attached
