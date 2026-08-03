# Fairwave hardware BOM tiers

Three build tiers; all prices are Q3-2026 street estimates in USD, qty-1,
before shipping/tax. `vendor_hint` is a hint, not an endorsement — a BOM is
never a purchase order. **All tiers are receive-capable in lab mode; transmit
requires the full RF gate (country + license acknowledgment + band
allow-list) regardless of what hardware is installed.**

| Tier | File | ~Total | Use case |
| --- | --- | --- | --- |
| Dev kit | `dev-bom.csv` | ~$1,700 | Contributor lab, measurements, driver work (USRP B200mini-i) |
| Community | `community-bom.csv` | ~$400–650 | Home/community lab trials (LimeSDR-mini or BladeRF x40) |
| CBRS production | `cbrs-bom.csv` | ~$2,700 + SAS fees | Licensed CBRS GAA deployment (US, part 96) |

## Price totals (rough, single unit)

- **Dev kit**: SDR 1325 + SBC 110 + memory 14 + power 22 + enclosure 45 +
  fans 16 + GPSDO 60 + antennas/cables 29 + misc 56 ≈ **$1,677**
- **Community**: SDR 150 + Pi5 80 + SD 15 + PSU 20 + enclosure 30 + fan 6 +
  cables/thermal 49 ≈ **$350** (with BladeRF x40 instead: ≈ **$680**)
- **CBRS**: B210 1530 + filter 120 + antenna 140 + LNA 90 + GPSDO 140 +
  PoE+ 35 + surge 30 + outdoor enclosure 120 + thermal 25 + interconnects 30
  ≈ **$2,260** + ~$300/yr SAS per site

## Guidance

- The SDR is the tier driver: B200mini (dev), LimeSDR-mini/BladeRF
  (community), B210 (CBRS full-duplex). See `hw/sdr-notes/` per device.
- The 260 mm pizza-box enclosure (`hw/enclosure/`) fits all tiers; CBRS swaps
  it for an outdoor-rated box.
- Buy the antenna chain only when you are legally cleared to transmit;
  lab mode needs antennas at most for RX measurement.
- Recurring costs (SAS fees, spectrum admin) are not in the unit totals.
