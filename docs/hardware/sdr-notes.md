---
title: SDR Notes
---

# SDR Notes

Fairwave supports three SDR families. All are USB 3.0 devices; none are FCC/CE-certified as cellular transmitters by themselves — real RF requires the `tx/arm` gate, a legal basis, and (for CBRS) a certified path (`docs/spectrum-and-law/`).

## Comparison

| | USRP B200/B200mini | USRP B210 | LimeSDR | LimeSDR Mini | bladeRF x40/x115 |
|---|---|---|---|---|---|
| Frequency | 70 MHz–6 GHz | 70 MHz–6 GHz | 10 MHz–3.5 GHz | 10 MHz–3.5 GHz | 300 MHz–3.8 GHz |
| RF channels | 1×1 | 2×2 | 2×2 | 1×1 | 1×1 |
| ADC/DAC | 12-bit | 12-bit | 12-bit | 12-bit | 12-bit |
| Bandwidth | 56 MHz | 56 MHz | 61.44 MHz | 61.44 MHz | 28/56 MHz |
| Driver | UHD | UHD | LimeSuite | LimeSuite | bladeRF-cli / libbladeRF |
| GPSDO | external ref in | external ref in | none (PPS via GPIO) | none | external ref in |
| Price (approx) | $750 / $1,000 | $1,600 | $450 | $250 | $420–$650 |
| Notes | B200mini tuned for LTE-class duty | 2×2 MIMO capable | wideband, FPGA | lightweight version | FPGA, open firmware |

## Drivers

| SDR | Driver | Package |
|---|---|---|
| B200/B210 | UHD (usrp package in Debian, or built from source) | `uhd` |
| LimeSDR | LimeSuite (`limesuite`), `LimeSDR_Util` | `limesuite` |
| bladeRF | `bladeRF-cli`, libbladeRF | `bladerf` |

- Verify with the vendor utility before starting the stack: `uhd_usrp_probe`, `LimeUtil --find`, `bladeRF-cli -p`.
- srsRAN builds must be compiled against the same driver major versions used by the eNB image; mixed ABI versions produce silent underrun/overrun errors.

## Clocking

- **B200 family:** external 10 MHz + PPS reference (OctoClock or Microphase) for real RF; GPSDO discipline recommended.
- **LimeSDR:** no reference input; acceptable for lab/experimental duty with NTP, not for TDD-band alignment.
- **bladeRF:** external reference input on x115; x40 has no ref input.
- `fairwave-agent` reports `gpsdo_lock` and `ntp_offset_seconds` so sync health is visible (`docs/architecture/telemetry.md`).

## Gains and Calibration

- TX gain is set in the eNB config, not by UI sliders; the policy cap is the binding limit.
- Calibration: B200 self-calibrates at runtime; LimeSDR benefits from periodic `LimeUtil --calibrate`; bladeRF is calibrated per-unit with stored values.
- Watch AGC: keep RX gain fixed during attach tests; AGC chasing during bursts breaks logs correlation.

## Bench Testing

- **Attenuator requirement:** any TX-capable SDR must have a 30–40 dB attenuator between its SMA and a receiver/antenna. Full LTE TX into an antenna indoors can exceed legal EIRP and will desensitize your own RX.
- Loopback check: SDR TX → attenuator → SDR RX; verify srsUE attach before going near air.
- Cable quality matters: > 3 m USB3 runs cause USB errors and `underrun` in the eNB logs — keep SDR adjacent to the box (`docs/hardware/index.md`).

## Related

- Radio stack: `docs/architecture/ran.md`
- BOMs: `docs/hardware/bom-tiers.md`
- Enclosure/thermal: `docs/hardware/enclosure.md`
