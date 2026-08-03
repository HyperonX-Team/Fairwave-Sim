# LimeSDR-mini notes

## Setup (Debian 12 / golden image)

```console
sudo apt-get install -y liblimesuite-dev limesuite-udev soapysdr-module-lms7
# LimeSuite builds the udev rule and a SoapySDR driver module:
SoapySDRUtil --find      # should list LimeSDR-Mini
LimeUtil --update        # flash board firmware to the packaged version
```

srsRAN 4G talks to LimeSDR via SoapySDR (`device_name=soapy` in
`core/ran/enb.rf.yml` for a hardware run). `SoapySDRUtil --probe` shows the
LMS7002M channels and sample rates.

## LMS7002M specifics

- Single-channel in the mini (1 TX + 1 RX on a shared LMS7002M); the
  community tier runs 1x1, fine for an experimental cell.
- Clock: LimeSDR-mini runs its own 30.72 MHz-derived clocks; keep
  `soapy` rate at exactly `23.04e6` for LTE (or 30.72e6 with the correct
  master clock selection). Probe: `SoapySDRUtil --probe="driver=lime"` shows
  supported rates.
- Gain settings are per-channel (`tx_gain` in srsENB `[rf]` maps to the
  Lime TX gain in dB). Start 60 dB RX-ish, 50 dB TX in RF tier - and again:
  **no TX before the RF gate.**

## Synchronization / multi-unit

- LimeSDR-mini has no onboard GPSDO or external 10 MHz input. For anything
  synchronized (SFN alignment between cells, phase-coherent MIMO), you need
  an external 10 MHz+PPS into the board - which the mini does not expose.
  This is why the CBRS tier uses the B210 + GPSDO instead.
- Two LimeSDRs on one host share USB; check `SoapySDRUtil --find` lists both
  and pass `soapy="driver=lime,serial=<serial>"` to disambiguate.

## Troubleshooting

- "Device not found" → udev rule not reloaded: `sudo udevadm control
  --reload-rules && sudo udevadm trigger`.
- Underflow/overflow (`U`/`O` in srs logs) → lower sample rate or check
  USB3; the mini is USB3-only, USB2 silently throttles.
- LMS boot failure → `LimeUtil --update` again, then `sudo poweroff` and
  cold-start the board (known quirk).
