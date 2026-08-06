# Changelog

All notable changes to Fairwave are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versions follow [SemVer](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Peering mesh data-plane runbook (v0.3 preview)
- CBRS/SAS client interface stubs (M4 preview)
- **Lab eSIM (SM-DP+) stack (`core/esim/`):** SGP.22-shaped remote SIM
  provisioning with P-256 ECDH key agreement, AES-128-CMAC (NIST SP 800-38B
  KAT-verified), counter-mode KDF, and AES-128-CBC encrypted bound profile
  packages; ES9+ endpoints (`initiateAuthentication` ... `cancelSession`);
  software eUICC running the full download loop for CI; QR activation codes
  (`LPA:1$...`); CLI `fairwave esim issue|serve`; file-backed profile
  registry (0600). Lab-only: dummy MCC 999 vectors, JSON transport pending
  GSMA conformance. See docs/adr/0013-esim.md.
- **README + landing page translations:** README in Arabic, Chinese,
  Spanish, French, German, and Hindi; landing page (website/index.html)
  gained a language switcher, the lab eSIM feature card, and corrected
  GitHub links.
- **More translations (12 new languages):** README and landing page now
  also in Italian, Portuguese, Russian, Japanese, Korean, Turkish, Polish,
  Dutch, Ukrainian, Swedish, Indonesian, and Vietnamese (19 languages
  total).
- **Automatic HSS write-back:** SIM issuance and revocation now seed
  Open5GS automatically (`core/sim-ops/hsswrite`): `mongosh` upsert driver
  (hss-init.sh document shape) and `dbctl` driver, both via docker exec so
  Ki/OPc never leave the node; wired into the control plane (`/v1/sims`
  issue/revoke) and `fairwave esim issue --hss-driver`; config
  `hss.driver`/`hss.container` + `FAIRWAVE_HSS_*` env; lab compose config
  enables it.

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
  Desktop/WSL2 (UE PHY loses subframe sync - see docs/tutorials/lab-attach.md).
- 5G SA/NSA stubs behind flags; no production SIM bureau integration.
- eSIM/LPA and roaming SEPP/IPX are documented future work.

[Unreleased]: https://github.com/HyperonX-Team/Fairwave-Sim/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/HyperonX-Team/Fairwave-Sim/releases/tag/v0.1.0
