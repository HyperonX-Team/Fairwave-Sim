---
title: Compliance Checklist
---

# Compliance Checklist

> Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands without proper authorization is illegal in most jurisdictions. You are solely responsible for licenses, SAS grants, indoor restrictions, and type approval. HyperonX and contributors provide software as-is for lawful private networks, research, and shared-spectrum regimes only.

Run this checklist **before** any real-RF session and keep the completed copy with the deployment logs. Every item maps to a field Fairwave's `tx/arm` gate can enforce (`docs/spectrum-and-law/index.md`).

## Operator Side

- [ ] Country code recorded in policy (gate refuses without it).
- [ ] License reference recorded: licence number, grant ID, or registration — even if "none applies, lab only".
- [ ] Band allow-list populated: only bands you may transmit on; EARFCNs checked against `design/spectrum-matrix.md`.
- [ ] Indoor/outdoor declared and matches what the authorization allows.
- [ ] EIRP cap set per band profile and below the authorization's limit.
- [ ] Site/registration reference (campus, experimental, or CBRS site) recorded.
- [ ] SAS grant reference present and current (CBRS only; approved SAS, not the mock).

## Hardware Side

- [ ] Attenuator (30–40 dB) in the TX path for bench work — mandatory, non-negotiable (`docs/hardware/sdr-notes.md`).
- [ ] GPS reference locked (real RF deployments; PPS verified) where the regime requires it.
- [ ] Antenna/feeder known to be band-correct; EIRP math includes cable + antenna gain.
- [ ] Type approval: radio path certified for the regime (CBRS: Part 96 CBSD; EU: RED conformity where applicable).

## Software Side

- [ ] `tx/arm` completed with correct country, license ref, band list — or explicitly left unarmed for lab.
- [ ] Agent `safe_tx` asserted; eNB process only runs when asserted (`docs/software/fairwave-agent.md`).
- [ ] `rfkill` state matches policy (authorized bands only; not blanket-off).
- [ ] Logging/audit on: gate decisions, arm events, grant heartbeats, phase transitions are retained.

## Sign-Off

| Field | Value |
|---|---|
| Operator |  |
| Date |  |
| Country / region |  |
| Authorization reference |  |
| Band(s) / EARFCNs |  |
| EIRP cap |  |
| Indoor / outdoor |  |
| Notes |  |

Print or copy this page as a PDF, sign it, and file it with the deployment. In an incident, this checklist is your first exhibit (`docs/ops/incident-response.md`).
