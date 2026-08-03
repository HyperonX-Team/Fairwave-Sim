# USRP B200mini / B210 notes

## Setup (Debian 12 / golden image)

```console
# UHD from the distro is fine for B200-series (version matches in uhd_usrp_probe)
sudo apt-get install -y uhd-host libuhd-dev
sudo uhd_images_downloader          # downloads fpga/firmware images (~/.cache/uhd)
uhd_usrp_probe                      # should enumerate B2xx
```

Fairwave image uses `device_name=uhd` in `core/ran/enb.rf.yml`; the
`device_args` line pins `type=b200` and `master_clock_rate=23.04e6`
(multiple of LTE sample rates). Check the actual device: `uhd_usrp_probe
--args type=b200` lists clock rates, serial, and daughterboard.

## USB

- B200mini/B210 are USB 3.0; use a quality 0.5 m cable and a USB3 port.
  `sudo dmesg -w` shows `usb 3-1: UHD ...` on attach.
- If the device enumerates as USB2, sample rate drops to 8 MS/s and srsENB
  will fail to keep timing — check `uhd_usrp_probe` for "USB 3.0" under
  transport. No hubs in the RF path.

## Gain

- `tx_gain` (srsENB `[rf]`) maps to the B200 TX amplifier; 60 dB is a sane
  RF-tier start (see enb.rf.yml). **Lab mode: keep TX unarmed — gain only
  matters once the RF gate has passed.**
- `rx_gain` 50 dB typical; lower it if the AGC reports saturation
  (`UHD` prints `L`/`O` overflows: `D` = dropped samples).

## Sample rate and timing

- `master_clock_rate=23.04e6` supports 30.72 MS/s LTE at exact
  rate-matched ratio; 11.52e6 for 15.36 MS/s. srsRAN 4G wants the LTE-clock
  multiples; mismatched clock rates cause periodic under/overflows.
- Keep `base_srate` consistent across the ZMQ and UHD paths when switching
  radio backends for the same cell config.

## GPSDO

- B200mini has a GPSDO option; B210 requires the on-board option. Set it via
  UHD: `uhd_usrp_probe --args "type=b200,use_gpsdo=true"` — the GPSDO
  disciplines the 10 MHz + PPS. Fairwave's CBRS tier requires GPSDO
  synchronization (see `hw/bom/cbrs-bom.csv`).
- Check lock: `uhd_usrp_probe | grep -A2 -i gps` → "GPSDO: locked" expected
  after 15–60 min cold. Never trust "unlocked" for licensed operation.

## Troubleshooting

- `uhd_find_devices` empty → kernel module / cable / port issue.
- "No UHD device found" in srsENB log → run `uhd_usrp_probe` first; wrong
  `device_args` serial is the usual cause.
- FPGA image mismatch → `uhd_images_downloader && uhd_usrp_probe` re-flashes.
