---
title: eSIM and LPA
---

# eSIM and LPA: What Fairwave Supports, Honestly

Fairwave ships a **lab SM-DP+ server and a software eUICC** (`core/esim/`)
that implement the full SGP.22-shaped profile download loop, CI-verified
without hardware. This page stays honest about the line between "runs in the
lab" and "GSMA-certified production eSIM".

## What the GSMA ecosystem is

| Piece | What it is | Openness |
| --- | --- | --- |
| eUICC | The embedded SIM chip (a computer on a card) | Hardware, spec published |
| LPA | Software on the device that downloads profiles | Spec published; implementations are often closed |
| RSP (SGP.22 / SGP.32) | The download protocol between LPA and server | Published, but server-side SM-DP+ is certified/licensed |
| SM-DP+ | Server that prepares and delivers profiles | Certified operators/vendors for production |

**Bottom line:** production eSIM still requires GSMA certification and a
certified operator contract. Fairwave's stack is a lab implementation of the
flow with real cryptography, clearly labeled as such (see
[ADR-0013](../adr/0013-esim.md)).

## What Fairwave does support now

1. **Lab SM-DP+ server** (`fairwave esim serve`): the ES9+ endpoints
   (`initiateAuthentication`, `authenticateClient`,
   `getBoundProfilePackage`, `confirmOrder`, `handleNotification`,
   `cancelSession`) with P-256 ECDH key agreement, AES-128-CMAC integrity
   (NIST known-answer tested), and encrypted bound profile packages.
2. **Profile minting** (`fairwave esim issue`): mints an ICCID + activation
   code (`LPA:1$smdp$token`) for a lab vector SIM, writes a QR PNG, and
   registers the profile in a local 0600-protected registry.
3. **Software eUICC** (`core/esim/euicc`): a Go eUICC that performs the full
   download - key agreement, MAC verification, decrypt, install - so the
   loop is testable end-to-end in CI, and as a reference for ports to real
   eSIM modules.
4. **End-to-end lab runbook**: issue a code, run the server, download with
   the software eUICC ([esim-first tutorial](../tutorials/esim-first.md)).
   Physical-phone download works once the phone's LPA accepts the transport;
   see the tutorial's step 5 for attach status.

## What Fairwave does NOT do (yet)

- **No GSMA-certified transport.** The wire messages are JSON over HTTPS,
  not the SGP.22 ASN.1/DER transport. Physical phone LPAs speak DER, so
  real-phone installation is the open conformance item, not something the
  lab stack silently claims.
- **No carrier applet in profiles.** A physical phone can download the
  profile package but needs a USIM applet to attach; the software eUICC
  does not need one because it interprets the profile directly. This is a
  hardware/module milestone, not a protocol one.
- **No production IMSIs, no bypass flags.** Profile issuance accepts only
  the dummy lab vectors (MCC 999). Production issuance waits for GSMA
  certification.
- **No KEK-wrapped registry encryption yet.** The registry file is 0600 but
  plaintext; the ADR-0006 encrypted vault applies to it in a follow-up.

## Alternatives to a full SM-DP+ for production

1. **Factory-injected eUICC profiles** via your card bureau (works today,
   matches the bundle flow).
2. **A commercial SM-DP+ service** - your IMSI ranges and profiles are
   hosted by a certified provider; Fairwave's provisioner output is the
   source data you upload (CSV/JSON interchange).
3. **A community "virtual operator" arrangement** - a licensed MVNO/MVNE
   partner hosts profiles; you keep the network, they keep RSP compliance.
4. **Plain physical SIMs** - the honest default for community networks; see
   the bureau runbook.

## Timeline

- v0.1 (lab): QR/activation-code generation only, clearly labeled demo.
- v0.2 (current): lab SM-DP+ + software eUICC, full download loop in CI,
  phone download validation runbook.
- M4+: interchange with one certified SM-DP+ (contract + imports) if a
  partner exists; ASN.1 transport + applet integration tracked as open
  items.
- No milestone commits to running a GSMA-certified SM-DP+ without the
  certification program.

## Related

- [ADR-0013](../adr/0013-esim.md) · [SIM lifecycle overview](index.md) ·
  [Provisioner](provisioner.md) · [Bureau runbook](bureau-runbook.md)
