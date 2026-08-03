# ADR 0001: Mobile core - Open5GS instead of Magma (or free5GC)

- Status: Accepted
- Date: 2026-08-02

## Context

Fairwave needs a 4G EPC and a future path to 5G GC. Candidates:

- **Open5GS** - C, standalone binaries, tiny memory footprint, REST-ish configs, active community, 4G+5G SA in one repo.
- **Magma** - Python/Go gateway and cloud-native design; heavier ops burden, more moving parts.
- **free5GC** - pure Go; strong 5G focus, smaller 4G track record.
- **OAI CN** - very complete but operationally complex, many knobs.

## Decision

Use **Open5GS** as the default mobile core, driven by `fairwave-control` via configuration
templating, Helm, and Compose.

## Consequences

- [+] Small memory footprint - fits pizza-box targets (NUC, CM4/5).
- [+] Single repo with MME+SGW+PGW+HSS+PCRF+SMF+UPF+UDM+AUSF; simpler mental model.
- [+] AGPL-3.0 is acceptable for Fairwave's posture (we configure and drive; no source modification by default).
- [+] Fast gNB/eNB interoperability via srsRAN containers.
- [!] No built-in northbound REST for provisioning; we must implement it in `fairwave-control`.
- [!] Upstream CVEs propagate; hence pinned digests, SBOM, cosign signing.

## Alternatives considered

- **Magma** - rejected: ops burden too large for a pizza-box, cloud-first model opposes local-first.
- **free5GC** - deferred: could be the 5G SA path later via a feature flag; 4G remains Open5GS.
- **OAI CN** - rejected: complexity too high for community operators; revisit later for RU/DU split research.
