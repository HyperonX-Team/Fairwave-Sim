# BladeRF notes (x40 / x115 / micro)

## Setup (Debian 12 / golden image)

```console
sudo apt-get install -y bladerf libbladerf-dev bladerf-firmware-fx3
bladeRF-cli -i
# In the CLI:  info        → hardware + firmware versions
#              firmware load /usr/share/Nuand/bladeRF/bladeRF_fw_*.img  (only if needed)
#              flash load  ...  (only when upgrading the FX3/FPGA - not routine)
```

srsRAN 4G uses BladeRF via SoapySDR (`device_name=soapy`) with the
`libbladeRF` backend, or directly via the `bladeRF` device name. Fairwave
templates use the Soapy path; `SoapySDRUtil --find` must list the device
before starting srsENB.

## bladeRF-cli

| Command | Meaning |
| --- | --- |
| `info` | Versions, serial, FPGA loaded |
| `print` | Current gain/freq/sample-rate state |
| `set frequency 3.55G` | TX/RX center frequency |
| `set samplerate 23.04e6` | LTE-compatible rate |
| `rx config ...` / `tx config ...` | Streaming (lab RX measurements) |
| `version` | Firmware/FPGA versions |

## libbladeRF

- x40: 40 MS/s, 1×1, 300 MHz–3.8 GHz - enough for band 3/48 RX and low-rate
  TX in community trials.
- x115: 115 MS/s, 1×1, broader frequency range.
- micro A4: 2×2 MIMO, 61.44 MS/s, 70 MHz–6 GHz - the dev-tier BladeRF.
- All: 12-bit ADCs/DACs, tuneable gain; `set gain 50` etc. Check
  `hw/bom/community-bom.csv` for the x40 line.

## Gain / sample rate

- `tx_gain` 50–60 dB community tier start (again gated by the RF gate -
  never transmit without it).
- Keep sample rates LTE multiples: 23.04e6 works across srsRAN configs.
- BladeRF's FX3 USB3 path is sensitive to host scheduling: run srsUE/srsENB
  on the isolated cores the golden image provides (`isolcpus=2-3`).

## Synchronization

- No onboard GPSDO on the classic x40; micro A4 has optional external clock
  input (10 MHz). For CBRS-grade timing use the B210 path - BladeRF is a
  lab/community instrument.

## Troubleshooting

- "Failed to open device" → permissions: add user to `plugdev`/`bladerf`
  group or reinstall `bladerf-firmware-fx3` (udev rules).
- USB3 negotiation fail → try another port/cable; `lsusb -t` should show
  `5000M` for a bladeRF bus.
- FPGA half-loaded (`info` shows stale version) → re-run `bladeRF-cli -i`,
  `flash load` the matching FPGA, then power-cycle.
