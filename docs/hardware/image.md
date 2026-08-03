---
title: Golden Image
---

# Golden Image

The Fairwave golden image is the reference OS for the pizza box: Debian 12, tuned for deterministic radio processing, with the agent and control plane as first-class systemd units.

## Base

- **Debian 12 (bookworm)** bookworm, amd64 or arm64.
- Minimal install: no desktop, no cloud agents, no snap/telemetry.
- The image script is a provisioning playbook, not a one-off manual setup; reruns must be idempotent.

## Kernel

- Stock Debian `linux-image-cloud-amd64`/`arm64` or `linux-image-rt-amd64`; RT kernel recommended for real-RF srsRAN.
- Boot params:

```
isolcpus=2,3 nohz_full=2,3 rcu_nocbs=2,3
hugepagesz=1G hugepages=4
```

- CPU isolation: cores 2–3 reserved for the PHY/srsRAN threads; the agent and control plane stay on cores 0–1.
- Hugepages: 1 GB hugepages for shared memory transport (srsRAN and Open5GS data plane).
- Everything is applied via `/etc/default/grub` + `update-grub` in the image playbook, not ad hoc.

## RF Policy at OS Level

- `rfkill` must stay **enabled** for cellular radios by default.
- The image does not blanket-disable rfkill. Only an operator who has cleared the `tx/arm` gate on the band in use may unblock that band, and the change is recorded in the audit log.
- Rationale: "rfkill off" at image level would make accidental TX the default; it isn't.

## Users and Permissions

- Service account `fairwave` (system user, no shell).
- `/var/lib/fairwave/` root: `fairwave`, mode 0700; keys subdir 0600 (`docs/architecture/security.md`).
- SDR USB devices: udev rule grants `fairwave` access to the SDR vendor IDs only.

## systemd Units

| Unit | Runs | Enabled |
|---|---|---|
| `fairwave-agent.service` | agent (root-capable probes, SDR access) | yes |
| `fairwave-control.service` | control plane (bare-metal mode) | yes, when not containerized |
| `fairwave-docker.service` | Compose stack (Open5GS, srsRAN, UI, portal) | yes |
| `fairwave-watchdog.timer` | periodic watchdog re-check | yes |

Bare-metal vs containerized: the default golden image runs the stack in Docker Compose and the agent natively (`docs/software/fairwave-control.md`).

## Unattended-Upgrades

- `unattended-upgrades` enabled for Debian **security** updates only.
- Pin `fairwave-*` packages (if installed from repo) to manual: the stack updates follow Fairwave releases, not Debian cadence.
- Reboot policy: `Automatic-Reboot` off by default; the operator gets an alert instead (`docs/ops/monitoring.md`).

## Validation

The image ships with a smoke script: boots, checks kernel params (`/proc/cmdline`), hugepage count, rfkill state, systemd unit statuses, and SDR probe (`uhd_usrp_probe`/`LimeUtil --find`/`bladeRF-cli -p` as applicable). CI runs it before each release (`design/roadmap.md`).

## Related

- Deployment: `docs/ops/index.md`
- Agent duties: `docs/software/fairwave-agent.md`
- Hardware tiers: `docs/hardware/index.md`
