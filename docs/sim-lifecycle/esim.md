---
title: eSIM and LPA
---

# eSIM and LPA: What Fairwave Supports, Honestly

This page is deliberately honest about eSIM. The GSMA ecosystem around eSIM - the LPA (Local Profile Assistant), the SM-DP+ servers that generate profiles, and the RSP (Remote SIM Provisioning) protocol - is largely proprietary and licensing-bound. Fairwave does not pretend otherwise.

## What the GSMA ecosystem is

| Piece | What it is | Openness |
| --- | --- | --- |
| eUICC | The embedded SIM chip (a computer on a card) | Hardware, spec published |
| LPA | Software on the device that downloads profiles | Spec published; implementations are often closed |
| RSP (SGP.22 / SGP.32) | The download protocol between LPA and server | Published, but server-side SM-DP+ is certified/licensed |
| SM-DP+ | Server that prepares and delivers profiles | Certified operators/vendors; not something you run casually |

**Bottom line:** you cannot "just run a production SM-DP+" from an open-source project in v0.1. Doing so would violate GSMA compliance expectations and interoperate with nothing.

## What Fairwave does support

1. **Activation/QR code generation.** For provisioning workflows, the operator UI and `fairwave sim issue` can emit an activation code in the `LPA:1$smdp://...$...` shape, plus a QR rendering, for *handset-side capture*. The code references a profile that still has to be made available through a real SM-DP+ - Fairwave generates the paperwork, not the backend.
2. **Lab profile note.** In lab mode, the LPA/QR is purely cosmetic scaffolding: the virtual UE uses HSS credentials directly, never eUICC download. Treat QR output as a format demo until a real SM-DP+ backend exists.
3. **Pre-provisioned eUICC with physical profile injection (future).** Some eUICC vendors support injecting a profile at the factory alongside physical personalization. The bureau workflow (see [bureau runbook](bureau-runbook.md)) can carry an "eUICC lot" variant of the bundle; this is the most realistic eSIM path for community networks today.

## What Fairwave does NOT do

- No production RSP server (no SM-DP+ implementation, no GSMA certification).
- No claim of interoperability with operator eSIM catalogs.
- No LPA implementation - that runs on the handset, out of our control.
- No bypassing of device/OS eSIM approval flows.

## Alternatives to a full SM-DP+

If you need eSIM in production, options, in rough order of practicality:

1. **Factory-injected eUICC profiles** via your card bureau (works today, matches the bundle flow).
2. **A commercial SM-DP+ service** - your IMSI ranges and profiles are hosted by a certified provider; Fairwave's provisioner output is the source data you upload to them (CSV/JSON interchange).
3. **A community "virtual operator" arrangement** - a licensed MVNO/MVNE partner hosts profiles; you keep the network, they keep RSP compliance.
4. **Plain physical SIMs** - the honest default for community networks; see the bureau runbook.

## Timeline

- v0.1 (lab): QR/activation-code generation only, clearly labeled demo.
- M4+: interchange with one certified SM-DP+ (contract + imports) if a partner exists; otherwise the feature stays "tooling only".
- No milestone commits to running our own RSP server.

## Related

- [SIM lifecycle overview](index.md) · [Provisioner](provisioner.md) · [Bureau runbook](bureau-runbook.md)
