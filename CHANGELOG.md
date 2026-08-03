# Changelog

All notable changes to Fairwave are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versions follow [SemVer](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Peering mesh data-plane runbook (v0.3 preview)
- CBRS/SAS client interface stubs (M4 preview)

## [0.1.0] - 2026-08-02

### Added

- **Lab core (M0):** Open5GS 2.7.7 EPC (MME/SGW-C/SGW-U/SMF/UPF/HSS/PCRF/NRF)
  + srsRAN eNB/UE over ZMQ virtual radio, all in Docker Compose.
- **Control plane:** `fairwave-control` (Go): node identity, enrollment,
  lifecycle state machine, northbound REST API, Prometheus metrics,
  config with JSON-Schema validation, file-backed state store.
- **Agent:** `fairwave-agent` health heartbeat, watchdog, safe-TX flag.
- **CLI:** `fairwave` (Go): `node init|status|join|leave`, `sim issue|revoke|list`,
  `peer list|add`, `spectrum check`, `tx arm`, `policy get|set`, `doctor`, `version`.
- **SIM provisioner:** offline-first Ki/OPc generation (AES-based Milenage OPc
  derivation), batch CSV/JSON output, lab test vectors (dummy, non-routable IMSIs).
- **Spectrum gate:** country allow-lists, per-band EIRP caps, license-ref
  requirement, acknowledgment phrase; RF TX is disabled by default (lab mode).
- **Operator UI + captive portal:** local-first static dashboards (no build step).
- **Docs site:** 69 pages (vision, architecture, hardware, spectrum & law, ops,
  SIM lifecycle, peering, security, API, tutorials, reference, 12 ADRs).
- **CI:** GitHub Actions (build/test/vet, compose gate validation, docs sanity,
  gitleaks), pre-commit hooks, CodeQL, release workflow with SBOM + cosign.
- **Deploy assets:** Compose (lab + RF-gated), Helm chart, Ansible role,
  Terraform stub, Dockerfiles for Open5GS/srsRAN/control-plane.

### Fixed

- Open5GS v2.7 `freeDiameter` cert generation for lab TLS paths.
- PCRF mongoc crash (missing `db_uri` in pcrf.yaml).
- SMF SBI/HTTP2 retry noise (EPC-only SMF without SBI client).
- Store persistence (`.json` suffix mismatch).

### Known limitations (v0.1)

- The final ZMQ data-path hop (UE IP on `tun_srsue`, ping) requires stable
  low-jitter scheduling; verified on native Linux, degraded under Docker
  Desktop/WSL2 (UE PHY loses subframe sync — see docs/tutorials/lab-attach.md).
- 5G SA/NSA stubs behind flags; no production SIM bureau integration.
- eSIM/LPA and roaming SEPP/IPX are documented future work.

[Unreleased]: https://github.com/HyperonX-Team/Fairwave-Sim/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/HyperonX-Team/Fairwave-Sim/releases/tag/v0.1.0
