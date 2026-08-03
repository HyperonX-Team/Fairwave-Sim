# Spectrum & Regulatory Matrix

> [!IMPORTANT]
> **Legal banner** - Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands
> without authorization is illegal in most jurisdictions. You are solely responsible for
> licenses, SAS grants, indoor restrictions, and type approval. HyperonX and contributors
> provide software as-is for lawful private networks, research, and shared-spectrum regimes only.

This matrix is a **starting point, not legal advice**. Always verify current rules with
your national regulator before enabling TX. Fairwave enforces per-country allow-lists in
code; the configuration layer will refuse to set ARFCN/band outside the allowed ranges.

## Region / Regime Overview

| Region | Regime | Band / Frequency (LTE) | Typical Power Limits | Fairwave posture |
|--------|--------|------------------------|----------------------|-------------------|
| USA | Private LTE / CBRS | Band 48 (3550–3700 MHz) | ≤10W EIRP (SAS-controlled) | Lab mode + SAS client stub; certified SAS integration roadmap (M4) |
| USA | MLS/WCS / unlicensed | Band 46 (5150–5925 MHz), Band 28 lab attenuated | Strict EMI/EMC rules | Lab-only; attenuated bench test only |
| UK | Shared Access License | Band 1 (2100 MHz), Band 3 (1800 MHz), Band 7 (2600 MHz), Band 38 (2600 MHz) | 2–10W EIRP depending on geo | Ofcom SAL handled by operator; Fairwave gates by country code |
| UK | Local access licensing | Band 3 / 38; dynamic | Site-specific | Manual config; requires proof-of-license |
| DE / FR etc | Experimental / campus | Band 3/7/38/40, limited | ≤1W EIRP for campus | Lab mode default; local regulators must approve |
| India | In-building captive use | Band 3/40 | <5W with paperwork | Lab default; deployment needs DoT private-captive approval |
| Australia | ACMA apparatus licence | Band 28/3/7/40 | 1–5W typical | Lab default; licence-holder gates TX |

## Spectrum-Hardening Design

Fairwave's software **soft-gates TX** via three independent layers:

### Layer 1 - Compile mode

```bash
FAIRWAVE_RF_MODE=none      # default: no RF support compiled in
FAIRWAVE_RF_MODE=lab-zmq   # virtual zmq only (CI)
FAIRWAVE_RF_MODE=hardware  # + enable_tx knob needed at runtime
```

### Layer 2 - Runtime country gate

`country_code` must match a list of MCCs for which Fairwave has curated band plans. Note
Fairwave does not force every MNC/MCC; those are your responsibility as operator.

### Layer 3 - Frequency allow-list per profile

Each profile in `core/policy` declares:
- `bands`: allowed LTE band numbers
- `earfcn_min/max`: permitted EARFCN range
- `max_eirp_dbm`: hard cap
- `attenuation_required`: truthy if you must guarantee lab/attenuated operation

Control plane rejects any TX attempt outside the allow-list and refuses to commit the
generated Open5GS config to disk unless `node.is_lab=true && country_code == XX` matches.

## Private-LTE vs Wi-Fi calling posture

- **Private LTE:** Local PLMN, local HSS, SIP barred except via explicit policy.
- **Wi-Fi calling fallback:** Handset prefers Wi-Fi when both Fairwave and national SIM
  are present. Fairwave does (future) support ePDG hard-stubs (M5) but ships with no
  national-EPC integration; tunnels are your responsibility.

## Technical summary by country

### USA - CBRS (48/3550–3700)

- Requires SAS client: proof-of-certified-GAA/PPA.
- Fairwave stub has SAS **interface** + **mock**, with signed key exchange stub; distribution
  is subject to SAS vendor contract (CommScope, Google, Amdocs, Federated, etc.).
- Self-hosted open SAS is not permitted-SAS must be certified per FCC Part 96.

### UK - Shared Access License (SAL)

- Operator completes Ofcom online form; receives a licence with permitted frequency/power.
- Fairwave gates: `country_code=UK`, profile has `ofcom_licence_ref`, license number stored
  as part of tx-arm signed blob.

### EU - various local / experimental licenses

- Most regimes require bundle with detailed technical annex; Fairwave accepts PDF upload
  as docs-only (not machine-parsed) and requires `licence_annex_version` token to sign.

## Measuring / enforcement

- `fairwave spectrum check` queries the local agent, returns:
  - `is_lab` status
  - `country_code` and whether `armable` (spec-list match) is true
  - `granted_bands` by the active profile
- Attempt to `tx_arm=true` outside allow-list is refused, audited, and results in Web-UI warning.

## FAQ for operators

**Q: I want indoor private LTE on my own campus but have no licence yet; can I TX?**
**A:** No. Fairwave will stay in zmq mode. Get a licence, then send tx-arm sign-off.

**Q: Can Fairwave simulate a bad actor's TX pattern?**
**A:** That is not a project goal. Software patterns that primarily enable unauthorized
transmission will be rejected during code review.

**Q: My country isn't listed.**
**A:** Contribute a profile under `core/policy` (YAML + docs). The row must reference a real
regulator document. Community-sourced band data without a legal reference will not merge.
