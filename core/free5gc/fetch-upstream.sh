#!/usr/bin/env bash
# Pulls the canonical free5gc-compose NF configs that the Fairwave lab does
# not customize (nrf, ausf, udm, udr, pcf, nssf, chf, webui). The hand-adapted
# files (amfcfg.yaml, smfcfg.yaml, upfcfg.yaml) are committed and NOT
# overwritten. Run once before `docker compose -f deploy/docker-compose.5g.yml up`.
#
# The compose mounts these files read-only into each NF container, so the
# directory must be populated. Pin the ref below to a free5gc-compose release
# tag to freeze upstream drift.
set -euo pipefail

REF="${F5GC_COMPOSE_REF:-master}"
BASE="https://raw.githubusercontent.com/free5gc/free5gc-compose/${REF}/config"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for nf in nrf ausf udm udr pcf nssf chf webui; do
  url="${BASE}/${nf}cfg.yaml"
  out="${HERE}/${nf}cfg.yaml"
  echo "fetching ${url} -> ${out}"
  curl -fsSL "${url}" -o "${out}"
done

# TLS certs: the NFs expect cert/ in their working dir. Generate throwaway
# self-signed certs (lab only) if the directory is missing.
if [ ! -d "${HERE}/cert" ]; then
  echo "generating self-signed lab certs in ${HERE}/cert"
  mkdir -p "${HERE}/cert"
  for nf in amf smf nrf ausf udm udr pcf nssf chf; do
    openssl req -x509 -newkey rsa:2048 -nodes -keyout "${HERE}/cert/${nf}.key" \
      -out "${HERE}/cert/${nf}.pem" -days 3650 -subj "/CN=${nf}.free5gc.org" >/dev/null 2>&1
  done
fi

echo "done. NF configs staged in ${HERE}; review git status and commit the diff."
