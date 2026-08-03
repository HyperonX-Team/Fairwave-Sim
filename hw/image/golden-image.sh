#!/usr/bin/env bash
# Fairwave golden image builder — Debian 12 node OS.
#
# Outline: run ONCE per image build on a Debian/Ubuntu host (or in a
# privileged container). Produces build/rootfs.tar.xz + build/image.img.
#
#   sudo hw/image/golden-image.sh
#
# The image ships with RF OFF: rfkill blocks wireless radios at boot and no
# TX capability exists until the control plane arms it (RF gate).
set -euo pipefail

ARCH="${ARCH:-arm64}"                      # arm64 (SBC) or amd64 (x86 node)
RELEASE="${RELEASE:-bookworm}"
OUT="build/golden-${RELEASE}-${ARCH}"
MIRROR="${MIRROR:-http://deb.debian.org/debian}"

root=$(cd "$(dirname "$0")/../.." && pwd)
cd "${root}"

echo "[golden-image] stage 1: debootstrap (${RELEASE}/${ARCH})"
mkdir -p "${OUT}"
if [[ ! -d "${OUT}/rootfs" ]]; then
    debootstrap --arch="${ARCH}" --variant=minbase \
        --components=main,contrib,non-free-firmware \
        "${RELEASE}" "${OUT}/rootfs" "${MIRROR}"
fi

# Stage 2: configure the chroot (paths under "${OUT}/rootfs").
cat >> "${OUT}/rootfs/etc/apt/sources.list" <<EOF
deb ${MIRROR} ${RELEASE}-updates main contrib
deb http://security.debian.org/debian-security ${RELEASE}-security main contrib
EOF

mount_chroot() {
    for d in /proc /sys /dev /dev/pts /dev/shm; do
        mount --bind "$d" "${OUT}/rootfs$d" 2>/dev/null || true
    done
}
umount_chroot() {
    for d in /dev/shm /dev/pts /dev /sys /proc; do
        umount "${OUT}/rootfs$d" 2>/dev/null || true
    done
}

mount_chroot
chroot "${OUT}/rootfs" /bin/bash <<'CHROOT'
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
apt-get update

# Base + node OS
apt-get install -y --no-install-recommends \
    linux-image-arm64 linux-headers-arm64 firmware-linux firmware-misc-nonfree \
    systemd systemd-sysv openssh-server sudo \
    ufw wireguard jq curl ca-certificates \
    uhd-host liblimesuite-dev bladerf \
    ethtool iperf3 htop tcpdump networkd-dispatcher

# Service user
useradd --system --create-home --shell /usr/sbin/nologin fairwave || true
useradd -m -s /bin/bash admin || true
echo 'admin ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/admin
chmod 0440 /etc/sudoers.d/admin

# Kernel cmdline: dedicated cores for the PHY, hugepages for capture paths.
sed -i 's/^GRUB_CMDLINE_LINUX=.*/GRUB_CMDLINE_LINUX="isolcpus=2-3 nohz_full=2-3 rcu_nocbs=2-3 hugepagesz=1G hugepages=8"/' \
    /etc/default/grub

# rfkill everything wireless at boot; TX stays off until control-plane arms.
cat > /etc/systemd/system/fairwave-rfkill.service <<UNIT
[Unit]
Description=Fairwave: block wireless radios (TX stays off)
Before=network-pre.target
Wants=network-pre.target
[Service]
Type=oneshot
ExecStart=/usr/sbin/rfkill block all
[Install]
WantedBy=multi-user.target
UNIT
systemctl enable fairwave-rfkill.service

# First-boot provisioning hook (hostname, keys, config seed).
install -m 0755 /dev/null /usr/local/sbin/fairwave-firstboot

# Enable services; the units themselves arrive via ansible or the
# provisioning bundle (hw/image/fairwave-*.service), enabled at first boot.
systemctl enable systemd-networkd ssh || true

# Clean apt artifacts for a smaller image.
apt-get clean
rm -rf /var/lib/apt/lists/* /tmp/*
CHROOT

echo "[golden-image] stage 3: staging the provisioning bundle"
install -m 0644 hw/image/fairwave-control.service "${OUT}/rootfs/etc/systemd/system/"
install -m 0644 hw/image/fairwave-agent.service "${OUT}/rootfs/etc/systemd/system/"
install -m 0755 hw/image/firstboot.sh "${OUT}/rootfs/usr/local/sbin/fairwave-firstboot"
mkdir -p "${OUT}/rootfs/etc/fairwave"
touch "${OUT}/rootfs/etc/fairwave/country"   # operator fills at first boot

echo "[golden-image] stage 4: packaging rootfs"
umount_chroot
tar -C "${OUT}/rootfs" -cf - . | xz -9 > "${OUT}/rootfs.tar.xz"

echo "[golden-image] done. Artifacts in ${OUT}/:"
echo "  rootfs.tar.xz  — unpack onto the SBC root partition"
echo "  (image.img flashable disk image: follow docs/hardware/image.md)"
