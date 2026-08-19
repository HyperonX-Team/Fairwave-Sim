---
title: eSIM Security Considerations
---

# eSIM Security Considerations

This page covers the security properties of Fairwave's eSIM surface
(lab SM-DP+ server, software eUICC, profile registry) so operators can
make informed decisions before exposing it beyond the lab.

## Threat model (lab server)

The embedded SM-DP+ (`fairwave esim serve` and the control-plane
`/es9plus/` mount) is a **lab implementation**:

- Wire transport is JSON over HTTPS by default; pass `--tls-cert/--tls-key`
  to `fairwave esim serve` to serve HTTPS directly. When the control plane
  fronts it, terminate TLS there.
- Cryptographic operations use real primitives (P-256 ECDH, AES-128-CMAC,
  AES-128-CBC) but the byte-level layout is lab-defined pending GSMA
  conformance. Do not assume interoperability with production LPAs.
- Profile payloads carry Milenage KI/OPc. The registry file is 0600 and
  can be encrypted at rest with AES-GCM via `registry.OpenWithKey` /
  `KeyFromPassphrase`. Treat the passphrase like a SIM vault KEK.

## Credential source

By default, `fairwave esim issue` resolves credentials from the three
dummy lab vectors (`simprov.LoadTestVector`). The control plane exposes a
`ProfileSource` hook (`ESIMOptions.ProfileSource`) so an operator can
inject a production/HSM-backed resolver. **Never wire a production
credential source without first enabling registry encryption and a
persistent session store.**

## Registry and sessions

- Registry entries can be encrypted at rest with AES-GCM (AES-256 key via
  `KeyFromPassphrase` or raw 16/24/32-byte key). Metadata (tokens,
  timestamps) stays readable; only the profile payload is encrypted.
- SM-DP+ sessions are in-memory by default. Enable the file-backed store
  (`ESIMOptions.SessionStorePath`) so a reboot does not abort an in-flight
  download exchange. Session files are written with 0600 and contain
  ephemeral ECDH private keys — protect them accordingly.

## Lab vs. production boundaries

- The `profile.NewLabProfile` gate refuses non-lab profile classes. A
  production path must use a different constructor and a credential source
  that does not depend on `LoadTestVector`.
- Activation codes are single-use by default and carry a configurable TTL.
- The software eUICC is a reference implementation; it does not replace a
  physical eUICC in a phone. Physical-phone download requires ASN.1/DER
  ES9+ transport conformance (GSMA SGP.22), which is an open item.

## Operational guidance

1. Run the SM-DP+ behind the control plane or a reverse proxy; do not
   expose `/es9plus/` directly to the internet in the lab.
2. Use `--tls-cert/--tls-key` on `fairwave esim serve` for any external
   LPA testing.
3. Enable registry encryption (`OpenWithKey`) before issuing any profile
   that carries real credentials.
4. Enable the file-backed session store if the server restarts during
   business hours.
5. Audit the `docs/sim-lifecycle/esim.md` alternatives if you need a
   production eSIM path today (certified SM-DP+ partner, bureau
   factory-injection, or licensed MVNO/MVNE).

## Related

- [eSIM and LPA](../sim-lifecycle/esim.md) · [SIM lifecycle overview](../sim-lifecycle/index.md)
- [ADR-0013 (eSIM scope)](../adr/0013-esim.md) · [ADR-0006 (SIM vault)](../adr/0006-sim-vault.md)
- [Operator auth](operator-auth.md) · [Privacy](privacy.md)
