#!/usr/bin/env bash
# Fairwave RF gate - the final safety check before any hardware radio may run.
# Refuses (exit 1) unless ALL of the following hold:
#   1. FAIRWAVE_RF_MODE == "hardware"      (explicit operator intent)
#   2. FAIRWAVE_TX_ACK_FILE set + existing + non-empty, and containing the
#      acknowledgment line "I hold authorization for this transmission"
#      (the same text the tx/arm API requires)
#   3. a country file with a 2-letter ISO-3166 alpha-2 code exists
#      (FAIRWAVE_COUNTRY_FILE, default /etc/fairwave/country)
#
# Exits 0 with a confirmation otherwise. Wire it as a compose "rf-gate"
# service that the eNB depends on via service_completed_successfully.
set -euo pipefail

fail() {
    echo "RF GATE DENIED: $*" >&2
    exit 1
}

[[ "${FAIRWAVE_RF_MODE:-}" == "hardware" ]] || \
    fail "FAIRWAVE_RF_MODE must be 'hardware'. Transmitting RF without explicit intent is unlawful in most jurisdictions and violates Fairwave policy."

ACK_FILE="${FAIRWAVE_TX_ACK_FILE:-}"
[[ -n "${ACK_FILE}" ]] || \
    fail "FAIRWAVE_TX_ACK_FILE must point at a file holding the transmission authorization acknowledgment."
[[ -f "${ACK_FILE}" ]] || \
    fail "acknowledgment file '${ACK_FILE}' does not exist."
[[ -s "${ACK_FILE}" ]] || \
    fail "acknowledgment file '${ACK_FILE}' is empty."
grep -qi "I hold authorization for this transmission" "${ACK_FILE}" || \
    fail "acknowledgment text missing from '${ACK_FILE}'. It must contain exactly: I hold authorization for this transmission"

COUNTRY_FILE="${FAIRWAVE_COUNTRY_FILE:-/etc/fairwave/country}"
[[ -f "${COUNTRY_FILE}" ]] || \
    fail "country file '${COUNTRY_FILE}' missing. Provide a file containing the 2-letter ISO-3166 alpha-2 code (e.g. US) for the transmit site."
country="$(tr -d '[:space:]' < "${COUNTRY_FILE}")"
[[ "${country}" =~ ^[A-Za-z]{2}$ ]] || \
    fail "country file must contain exactly a 2-letter ISO code, got '${country}'."

echo "RF GATE OK: FAIRWAVE_RF_MODE=hardware, acknowledgment present in ${ACK_FILE}, country=${country}"
exit 0
