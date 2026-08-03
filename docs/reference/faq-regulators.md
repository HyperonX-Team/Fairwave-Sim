---
title: FAQ for Regulators
---

# FAQ for Regulators

> Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands without proper authorization is illegal in most jurisdictions. You are solely responsible for licenses, SAS grants, indoor restrictions, and type approval. HyperonX and contributors provide software as-is for lawful private networks, research, and shared-spectrum regimes only.

This page answers the questions regulators actually ask about Fairwave. It is written for people who review radio, privacy, and network equipment — no 3GPP required.

## Q: Is Fairwave an IMSI catcher / stingray?

**No.** An IMSI catcher impersonates a network to harvest identifiers. Fairwave is the opposite: it operates a *bona fide, individually identifiable network* (its own PLMN, e.g. `999-99`), with subscribers it issued itself and stored in its own HSS. It does not scan, impersonate, or passively collect foreign IMSIs. It has no capability to camp devices onto a fake cell. See [What Fairwave is NOT](not-fairwave.md).

## Q: Does Fairwave intercept or decrypt traffic?

**No.** The EPC routes subscriber packets; it does not decrypt them (encryption is 3GPP end-to-end between handset and network — any operator can, by architecture, see its own subscribers' plaintext traffic at the PGW, but Fairwave ships no DPI, no payload inspection, and no lawful-interception backend). The only identification used is the operator's own IMSI hash for session accounting ([privacy](../security/privacy.md), ADR-0010).

## Q: How is spectrum access gated?

Three independent layers, all required to arm RF ([ADR-0008](../adr/0008-spectrum-gate.md)):

1. **Compile-time gate:** RF is disabled in the default build; a deliberate build flag enables the RF path.
2. **Runtime acknowledgment:** the operator must set a country code and acknowledge the license terms on each node.
3. **Frequency allow-list:** a validated spectrum profile ([spectrum matrix](/design/spectrum-matrix.md)) must contain the exact EARFCN before `tx/arm` returns success.

Inspectable at runtime: `GET /v1/tx/arm` shows which gates are open. The default build transmits nothing, ever.

## Q: How is lawful interception handled?

- **No LI backend ships.** There is no built-in interception facility, no remote activation path, and no government backdoor.
- Where law requires it, LI must be added as a documented, configuration-level integration (exporting session metadata/CDRs to an operator-provided LI system) — an operator-side, jurisdiction-specific decision, not a Fairwave feature. We document the interfaces; we do not implement LI.
- We will not add undocumented collection. Code and configuration are public; claims can be verified against the repo.

## Q: What about emergency calls (911/112)?

Fairwave is a private network and does not provide emergency-call routing. Handsets will not treat a private PLMN as their home network for emergency purposes; there is no guarantee that 911/112 works over a Fairwave network, and it must not be relied upon. Nodes should not be deployed where their signal could interfere with public-safety communications. Users of any real deployment need a fallback (public network) path. See [not-fairwave](not-fairwave.md).

## Q: Is the equipment type-approved?

- Fairwave is open-source software on commodity hardware; it is not itself a type-approved radio product.
- Any concrete deployment (SDR + amplifier + antenna) must comply with local equipment approval (e.g. FCC/CE type acceptance, ETSI EN 301 908 for LTE base stations) — that responsibility rests with the deploying operator.
- For shared-spectrum regimes (e.g. US CBRS GAA), operators must comply with SAS grants, indoor restrictions, and EIRP limits before transmission.

## Q: What data is retained, and for how long?

Defaults: session records 30 days, audit log 400 days, metrics 90 days ([privacy](../security/privacy.md)). Records identify subscribers only by truncated SHA-256 hashes of IMSI, not raw IMSI. Retention is configurable by the operator; the defaults exist because operations need them, and we publish them.

## Q: Who is accountable when something goes wrong?

The operator of a node is the accountable entity for its radio behaviour, subscriber data, and compliance — the same as with any base station. Fairwave's role is documented software plus auditable defaults; the [threat model](/design/threat-model.md) and ADRs state exactly what the software does and refuses to do.

## Q: Can I audit these claims?

Yes: the code, configs, ADRs, and this documentation are public. The runtime exposes gate state (`/v1/tx/arm`), policy, and audit log (append-only). Nothing about the RF path is hidden behind binaries or obfuscation.

## Related

- [Not Fairwave](not-fairwave.md) · [Privacy](../security/privacy.md) · [Spectrum gate ADR](../adr/0008-spectrum-gate.md) · [Carrier FAQ](faq-carriers.md)
