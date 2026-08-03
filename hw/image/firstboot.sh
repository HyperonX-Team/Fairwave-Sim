#!/usr/bin/env bash
# Fairwave first-boot provisioning (runs once, then disables itself).
#
# Installed as /usr/local/sbin/fairwave-firstboot by the golden image and
# invoked by a systemd oneshot unit (firstboot.service). Inputs come from a
# provisioning bundle the operator drops at /etc/fairwave/firstboot.env:
#   FW_HOSTNAME=<node hostname>
#   FW_BOOTSTRAP_TOKEN=<enroll token>          (one-time)
#   FW_CONTROL_ENDPOINT=http://<control-ip>:8080
#   FW_COUNTRY=<ISO-3166 alpha-2, empty in lab>
#
# INVARIANT: TX stays OFF. This script never writes an ack file, never sets
# FAIRWAVE_RF_MODE=hardware, and rfkill remains blocking until the control
# plane explicitly arms TX through the RF gate.
set -euo pipefail

ENV_FILE=/etc/fairwave/firstboot.env
[[ -f "${ENV_FILE}" ]] || { echo "[firstboot] no bundle at ${ENV_FILE}; skipping provisioning"; exit 0; }
source "${ENV_FILE}"

echo "[firstboot] hostname: ${FW_HOSTNAME:-<unset>}"

# 1. Hostname + hosts
if [[ -n "${FW_HOSTNAME:-}" ]]; then
    hostnamectl set-hostname "${FW_HOSTNAME}"
    sed -i "s/^127\.0\.1\.1.*/127.0.1.1\t${FW_HOSTNAME}/" /etc/hosts
fi

# 2. SSH host keys (generate once; never shipped in the image)
if [[ ! -f /etc/ssh/ssh_host_ed25519_key ]]; then
    ssh-keygen -q -A
    systemctl restart ssh || true
fi

# 3. Control-plane config seed (identity arrives from the bundle; keys and
#    secrets live in /etc/fairwave/env - mode 0600).
mkdir -p /etc/fairwave /var/lib/fairwave /var/lib/fairwave-agent
cat > /etc/fairwave/fairwave-control.yaml <<EOF
version: 1
mode: lab
listen_addr: ":8080"
data_dir: /var/lib/fairwave
rf:
  mode: lab
  country_file: /etc/fairwave/country
  ack_file: /etc/fairwave/tx-ack
EOF

umask 077
touch /etc/fairwave/env
chmod 0600 /etc/fairwave/env
chown -R fairwave:fairwave /etc/fairwave /var/lib/fairwave /var/lib/fairwave-agent

# 4. Agent bootstrap token (if provided in the bundle)
if [[ -n "${FW_BOOTSTRAP_TOKEN:-}" ]]; then
    sed -i "/^FAIRWAVE_AGENT_TOKEN=/d" /etc/fairwave/env
    echo "FAIRWAVE_AGENT_TOKEN=${FW_BOOTSTRAP_TOKEN}" >> /etc/fairwave/env
fi

# 5. Ensure TX stays off: rfkill block, no ack file, no hardware mode.
rfkill block all
rm -f /etc/fairwave/tx-ack
sed -i "/^FAIRWAVE_RF_MODE=/d" /etc/fairwave/env
echo "FAIRWAVE_RF_MODE=lab" >> /etc/fairwave/env

# 6. Enable node services (units shipped in the image).
systemctl enable fairwave-control.service fairwave-agent.service

# 7. Consume the bundle, disable this script.
rm -f "${ENV_FILE}"
systemctl disable fairwave-firstboot.service 2>/dev/null || true
echo "[firstboot] provisioning complete - TX remains OFF until the RF gate is passed."
