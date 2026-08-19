---
title: Glossary
---

# Glossary

Terms used across the Fairwave docs. Line-oriented; 3GPP-heavy where noted.

- **eNB** - Evolved NodeB: the LTE base station that talks to handsets over the air and to the EPC over S1.
- **gNB** - Next-Generation NodeB: the 5G NR base station. Fairwave is 4G-first; gNB is future-gated (ADR-0002).
- **EPC** - Evolved Packet Core: the 4G core network (MME, SGW, PGW, HSS) that handles registration, bearers, and data routing.
- **E-UTRAN** - Evolved UMTS Terrestrial Radio Access Network: the radio side of LTE (eNB + air interface).
- **MME** - Mobility Management Entity: the EPC's control-plane brain (attach, authentication orchestration, paging, tracking areas).
- **SGW** - Serving Gateway: anchors user-plane bearers, routes between eNB and PGW.
- **PGW** - Packet Data Network Gateway: assigns UE IPs, applies NAT/policy, is the network's internet edge.
- **HSS** - Home Subscriber Server: holds subscriber credentials (Ki/OPc, IMSI, profiles) and authenticates SIMs.
- **PCRF** - Policy and Charging Rules Function: policy/QoS decisions in the 4G core; Open5GS supports a basic profile.
- **UDM** - Unified Data Management: the 5G analogue of the HSS; the roadmap target for subscriber stores.
- **AMF** - Access and Mobility Management Function: 5G analogue of the MME. Shipped as part of the free5GC 5G core (`core: free5gc`).
- **GTP-U** - GPRS Tunnelling Protocol, User plane: tunnels user packets between eNB/SGW/PGW.
- **APN** - Access Point Name: the named "gateway" a UE connects through; Fairwave defaults are `internet` and `ims`.
- **EARFCN** - E-UTRA Absolute Radio Frequency Channel Number: numeric channel identifier for LTE carriers.
- **PLMN** - Public Land Mobile Network: identified by MCC+MNC; Fairwave default is `999-99`.
- **MCC** - Mobile Country Code: 3 digits; `999` is the test/dead MCC (never assigned to a real country).
- **MNC** - Mobile Network Code: 2–3 digits; `99` in the default PLMN.
- **TAC** - Tracking Area Code: locates UEs for paging; Fairwave default is `7`.
- **IMSI** - International Mobile Subscriber Identity: the 15-digit subscriber identifier on a SIM.
- **Ki** - The secret 128-bit key stored on the SIM and in the HSS; never leaves the vault in cleartext.
- **OPc** - Operator variant of the OP key; derived from Ki + operator OP per 3GPP; stored like Ki.
- **SQN** - Sequence Number: used in authentication (AKA) to resist replay; managed by HSS and SIM.
- **AUX** - Not standard 3GPP; in Fairwave docs, auxiliary provisioning data (ICCID, labels) that accompanies a credential.
- **SIM** - Subscriber Identity Module: the credential (physical card or virtual) that authenticates a device.
- **eSIM / LPA** - eSIM: embedded SIM; LPA: the device software that downloads profiles from an SM-DP+. Fairwave runs a lab SM-DP+ server (see [eSIM](../sim-lifecycle/esim.md)); production eSIM requires GSMA certification.
- **CBRS** - Citizens Broadband Radio Service: US shared-spectrum band (3550–3700 MHz).
- **SAS** - Spectrum Access System: the US database/enforcement system granting CBRS channel access.
- **GAA** - General Authorized Access: CBRS tier usable without individual license but under SAS rules.
- **PPA** - Priority Access License: CBRS tier above GAA, auctioned, still SAS-coordinated.
- **mDNS** - Multicast DNS: LAN service discovery; Fairwave uses `_fairwave._udp.local`.
- **WireGuard** - Modern UDP VPN protocol; Fairwave's data-plane mesh fabric (ADR-0004).
- **NAT** - Network Address Translation; the edge mechanism behind Fairwave's default local breakout.
- **PoE** - Power over Ethernet: how a field small-cell node is typically powered.
- **GPSDO** - GPS-Disciplined Oscillator: precise timing reference for real RF; required before real deployments.
- **SDR** - Software-Defined Radio: USRP, LimeSDR, BladeRF - the radio front-ends Fairwave supports.
- **ZMQ** - ZeroMQ: the virtual-radio transport between srsENB and srsUE in lab mode.
- **EIRP** - Equivalent Isotropically Radiated Power: the regulator-facing transmit power figure; spectrum profiles carry EIRP limits.
- **SEPP** - Security Edge Protection Proxy: 5G inter-operator signalling gateway; future-work only ([roaming](../peering/roaming-future.md)).
- **IPX** - IP eXchange: the carrier-grade interconnect network roaming traffic flows over; future-work only.
- **LI** - Lawful Interception: regulated interception capability; handled per jurisdiction, see [regulator FAQ](../reference/faq-regulators.md).
- **SPOF** - Single Point of Failure: a design anti-pattern Fairwave avoids (e.g. control plane without standby).
