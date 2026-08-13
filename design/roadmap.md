# Fairwave Roadmap

> [!IMPORTANT]
> **Legal banner** - Fairwave defaults to lab/no-RF mode. Roadmap numbers are aspirational;
> nothing is permitted to bypass spectrum gates.

## Milestone Overview

| # | Milestone | Target | Key outcomes |
|---|-----------|--------|--------------|
| M0 | Lab core + zmq UE attach + docs skeleton | v0.1.0 | CI-verified full attach in no-RF mode |
| M1 | SIM provisioner + operator CLI/UI | v0.2.0 | Offline batch pro-SIMs, revoke, show node status |
| M2 | Neighborhood mesh peering | v0.3.0 | Two-box attach, mTLS ctrl WG data |
| M3 | Hardware pizza-box images + BOM | v0.4.0 | Flashable images & bench-tested SDR notes |
| M4 | CBRS/SAS integration hooks | v0.5.0 | Region flags + SAS client stub + docs |
| M5 | Wi-Fi calling / ePDG path | v0.6.0 | Handoff to national EPC via ipsec, where lawful |
| M6 | Neutral-host multi-PLMN | v0.7.0 | Broadcast multiple PLMN ids; roaming prep |

## Shipped ahead of the table (5G core + fair use)

The free5GC 5G SA core landed ahead of the milestone numbering: `core: free5gc`
switches the control plane to free5GC (AMF/SMF/UPF/NRF/PCF/NSSF/AUSF/UDM/UDR) with
UDR write-back, AMF OAM session polling, and **fair-use metering from the core**
(free5GC CHF CDR files; GTP-U tap on 4G) feeding per-SIM quotas with auto-suspend.
Deployment: `deploy/docker-compose.5g.yml`, configs in `core/free5gc/`, RAN profiles
`core/ran/gnb.zmq.yml` + `ue5g.zmq.yml`. 4G/Open5GS remains the default path.

## Detailed gates

### M0 - v0.1.0 (lab)
- Open5GS + srsRAN in zmq mode attach on CI
- `fairwave node init`/`status`/`sim issue --lab`
- Operator UI skeleton + captive portal
- Threat model + ADRs + spectrum matrix draft
- Pre-commit / CI / SBOM / cosign-signed containers

### M1 - v0.2.0 (SIM ops)
- Offline-first provisioner batch CSV/JSON for card bureaus
- QR/activation-code generator (lab-only profiles only)
- HSS/UDM write-back via REST hook (no private Ki/OPc → web processes)
- CLI: `fairwave sim revoke`, `fairwave sim list`

### M2 - v0.3.0 (peering)
- mDNS rendezvous, optional static peer seeds
- mTLS certificate issuance per node via control-plane CA
- WireGuard full-mesh at up to 5 nodes; beyond 5, hub-and-spoke guide
- Route exchange: UE pools advertised via labels (BGP-lite)

### M3 - v0.4.0 (hardware)
- BOM tiers: Dev (mini-PC+SDR), Community (NUC/CM4+HAT), CBRS (with certified radio)
- Golden image scripts: Debian Bookworm, locked kernel, isolcpus, hugepages, rfkill off
- Enclosure docs + thermals + GPSDO outline

### M4 - v0.5.0 (CBRS/SAS)
- Region flags: `--region=US-CBRS`
- SAS client interface + mock implementation
- Docs: how to contract with a certified SAS provider; build-time go tag `-tags cbrs`

### M5 - v0.6.0 (ePDG)
- Open ePDG integration (strongSwan ePDG profile) - lab mode + operator-authored config
- Docs: lawful-intercept caveats, emergency-call routing
- Real-world pilot: **`lab-pilot` profile only**

### M6 - v0.7.0 (neutral host)
- Multi-PLMN broadcasting in lab
- Roaming SEPP/IPX stubs (docs-only; never shipped)
- Compliance guidance docs

## Non-goals (permanent)

- Free nationwide MNO coverage
- IMSI-catching / unwanted tracking tools
- LTE broadcasting for piracy signals
- Bypassing lawful interception rules where applicable
- Spectrum misuse: jamming, power-amp hacks supporting out-of-band TX

## Timeline (indicative)

Community-paced: each milestone assumes review bandwidth and a live CI lab. We prioritize
correctness, regulatory posture, and documentation quality over speed.

| Version | Est. date | Notes |
|---------|-----------|-------|
| v0.1.0 | 2026-08-30 | This release |
| v0.2.0 | 2026-11-30 | Simulator-core done, SIM batcher stable |
| v0.3.0 | 2027-03-31 | Peering MVP |
| v0.4.0 | 2027-06-30 | HW images ready |
| v0.5.0 | 2027-09-30 | CBRS hooks |
| v0.6.0 | 2027-12-31 | ePDG |
| v0.7.0 | 2028-06-30 | Neutral host |
