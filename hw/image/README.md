# Golden image plan (Debian 12 node OS)

The node OS is a minimal Debian 12 (bookworm) image built offline with
debootstrap, tuned for a small-cell host. Build it with
`hw/image/golden-image.sh` on a Debian/Ubuntu machine (or a container), then
flash `image.img` to the SBC's eMMC/SD.

## What the image contains

- **Kernel**: `linux-image-arm64` (or amd64) + `linux-headers`, firmware
  `firmware-linux` (wireless NICs — mostly to `rfkill` them off), SDR udev
  rules (`uhd-host`, `liblimesuite-dev`, `bladerf`), `wireguard`, `jq`.
- **Kernel cmdline** (`/etc/default/grub` or extlinux):
  `isolcpus=2-3 nohz_full=2-3 rcu_nocbs=2-3 hugepagesz=1G hugepages=8`
  — dedicated cores for srsENB PHY threads; hugepages reserve for DPDK-style
  capture if ever used.
- **rfkill at boot**: `fairwave-rfkill.service` blocks WLAN/BT radios at
  power-on; the SDR never starts with TX capability — the control plane must
  arm TX explicitly and the RF gate must pass first. TX-off is the default
  invariant of the image.
- **systemd units**: `fairwave-control.service`, `fairwave-agent.service`
  (see files in this directory) run as the unprivileged `fairwave` user.
- **Users**: `fairwave` (service), `admin` (SSH, keys injected by
  `firstboot.sh`).

## Boot flow

1. `firstboot.sh` runs on first power-up (systemd `firstboot.service`):
   set hostname, generate SSH host keys, seed `/etc/fairwave/` from the
   provisioning bundle, ensure TX stays off, then disable itself.
2. `fairwave-agent` starts, reads its config, enrolls with the control plane
   (`POST /v1/nodes/{id}/enroll`) using the bootstrap token from
   `/etc/fairwave/env`.
3. The control plane renders the node's configs (open5gs, srsRAN) — never
   baked into the image — and the agent applies them.

## Never baked in

Hostnames, IPs, PLMN, SIM material, WireGuard keys, tokens, license
acknowledgments, and any RF configuration. The image is hardware-agnostic
and trivially reproducible; all identity arrives at first boot.
