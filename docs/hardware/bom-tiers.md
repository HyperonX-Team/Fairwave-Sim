---
title: BOM Tiers
---

# BOM Tiers

Three reference bills of materials. Prices are approximate street prices in USD, mid-2026, and vary by region and availability. No SKUs are guaranteed; use them as sizing anchors.

## Dev Tier - Lab Only (~$200)

No RF. srsRAN zmq + srsUE, Open5GS, control plane - all in Docker Compose.

| Part | Example | Price |
|---|---|---|
| x86 mini-PC, 4 GB RAM, 256 GB NVMe | Beelink MINI S12 / N100-class (4 GB) | $150 |
| - or - 8 GB RAM variant (Dev Tier recommended) | Beelink MINI S12 / N100-class (8 GB) | $170 |
| USB 3.0 flash (unattended-upgrades log, optional) | 32 GB | $10 |
| Ethernet cable, PSU included | - | incl. |
| **Total** | | **~$150–180** |

SDR not required; add any tier-2 SDR later without rework.

## Community Tier - Private Network (~$450–850)

| Part | Example | Price |
|---|---|---|
| Compute: Intel NUC 8/10 class, 16 GB, 500 GB | NUC8i3 / similar | $350 |
| - or - CM4 (4/8 GB) + PoE HAT + carrier | Raspberry Pi CM4 + IO board | $180–220 |
| SDR (choose one) | LimeSDR Mini | $250 |
| | USRP B200mini (premium path) | $1,000 |
| GPSDO reference (recommended for RF) | OctoClock / Tallysman PPS board | $100–300 |
| 30 dB SMA attenuator + cables (bench) | generic kit | $25 |
| Enclosure | see `enclosure.md` | $60–120 |
| **Total (LimeSDR Mini)** | | **~$450–850** |

## CBRS US - Certified Small Cell (~$2,000+)

Licensed shared access on 3.5 GHz (FCC Part 96) - **not** any USB SDR. The radio path must be certified; see `docs/spectrum-and-law/cbrs.md` for obligations.

| Part | Example | Price |
|---|---|---|
| Compute (NUC class, 16 GB) | NUC13 / similar | $400 |
| Certified small-cell radio path | certified 3.5 GHz small cell / eNB-class unit | $1,200–2,500 |
| SAS client | certified SAS client (software or vendor-hosted) | $0–500/yr |
| GPS (required by Part 96) | GPSDO with PPS, antenna | $150–300 |
| Enclosure + RF cabling | per `enclosure.md` | $100–150 |
| **Total** | | **~$2,000–3,900** |

### Certification Obligations (Non-Negotiable)

- The radio must be **FCC-certified** for the band and output class used.
- A **certified SAS client** must be used; Fairwave's own mock SAS is for development only and never grants authority to transmit.
- Every spectrum grant must come from a real SAS instance against the certified client.
- EIRP must stay within the grant and the permit; the box's `tx/arm` policy must be set accordingly.

## Notes Common to All Tiers

- USB 3.0: SDR should own a dedicated controller; avoid sharing with bulk storage.
- PSUs: 12 V/5 V derived from quality supplies; a 30–60 W budget covers mini-PC + SDR.
- Prices exclude antennas (user-provided, band-specific).
