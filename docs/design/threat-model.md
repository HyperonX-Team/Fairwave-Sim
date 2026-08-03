# Threat Model

> [!IMPORTANT]
> **Legal banner** - Fairwave defaults to lab/no-RF mode. Transmitting on
> cellular bands without proper authorization is illegal in most
> jurisdictions. This threat model covers lawful private LTE deployments
> only.

The authoritative threat model lives in the repository root:
**[design/threat-model.md](threat-model.md)**.

It covers:

- STRIDE analysis of the control plane, agent, SIM vault, peering fabric,
  operator portal, RF path (lab), and metadata logs
- Asset inventory (Ki/OPc, operator credentials, mTLS CA, WireGuard keys,
  UE metadata, spectrum-attestation tokens)
- Trust boundaries and actors
- Explicit out-of-scope items (IMSI catchers, jamming, LI bypass)
- Privacy commitments (no IMSI in logs, hash-at-rest, consent-to-onboard)
- Reporting channel: security@hyperonx.io

Read the full document before contributing security-sensitive code.
