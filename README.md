<h1 align="center">Fairwave — a community carrier in a pizza box</h1>

<p align="center">
  <strong>Open-source private LTE: plug it into Ethernet, emit 4G, welcome your neighbors.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **LEGAL & SPECTRUM WARNING.** Fairwave defaults to **lab / no-RF** mode (zero-IF loopback only).
> Transmitting on cellular bands without proper authorization is **illegal in most jurisdictions**.
> You are solely responsible for licenses, SAS grants, indoor restrictions, and type approval.
> HyperonX and contributors provide software **as-is** for lawful private networks, research,
> and shared-spectrum regimes only. See [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

---

## The broken system

Mobile connectivity is a cartel of one: one SIM, one carrier, one contract, one walled garden.
Coverage maps are marketing brochures; rural streets fall off them; apartments bleed signal;
and you — the person paying — own none of the infrastructure serving you. If your national
carrier's price goes up or its tower goes quiet, your only choice is… a different monopoly
with the same towers and the same terms.

The modem in your pocket can talk to a base station 20 meters away. The only reason it
doesn't talk to *your* base station is that the base station was never allowed to be yours.

## The HyperonX fix

**Fairwave is the community carrier: a complete, open-source small-cell that fits in a pizza
box and plugs into ordinary Ethernet.**

A café, a housing co-op, a village hall, a township — anyone can run one:

1. Boot the Fairwave image on a mini-PC (x86 or ARM) with an attached SDR.
2. Run `fairwave node init`, answer the regulatory checklist (TX stays off until you prove authorization).
3. Issue Fairwave SIMs to your community — cards you own, credentials you control.
4. Phones attach. Traffic stays local when possible; multiple pizza boxes peer into a mesh;
   internet breakout rides a secure WireGuard tunnel when you want it.

Built on proven open infrastructure — **Open5GS** (EPC) and **srsRAN** (eNB/gNB) — with a Go
control plane, an operator dashboard, offline SIM provisioning, and lab mode that runs the
entire carrier in Docker with zero RF.

## What you get

- **A pizza box that is a carrier**: node identity, enrollment, lifecycle
  (`provision → register → on-air → peer → breakout`) managed by `fairwave-control`.
- **Real LTE attach**: Open5GS EPC + srsRAN eNB, configurable PLMN, tracking areas,
  `internet` + `ims` APNs, local breakout at the edge.
- **Fairwave SIM ops**: offline-first provisioner; generates Ki/OPc, batches CSV/JSON for
  card bureaus, writes HSS/UDM, revocation and swap controls. Lab vs. production hard-separated.
- **Neighborhood mesh**: mDNS discovery, mTLS control, WireGuard data plane, route exchange.
- **Operator portal**: local-first dashboard — UEs (privacy-preserving), backhaul, peers, lab mode.
- **Full lab mode**: entire network on pure IP (zmq) with srsUE — no radio, no license, CI-friendly.
- **Regulatory gates in code**: TX arming requires country code, license acknowledgment,
  frequency allow-list. Compiled-in refusal otherwise.

## Why it threatens incumbents

MNO lock-in becomes *optional for local coverage*. When the café, the co-op, and the village
can each run a cell and peer them, the national SIM is just a roaming option, not the gatekeeper.
Offload, neutral-host access, and community-priced plans stop being theoretical. Fairwave doesn't
replicate the telecom cartel — it makes the cartel's local monopoly contestable.

## Why it's feasible now

- **Open5GS** and **srsRAN** are mature, production-adjacent, and actively developed.
- **SDRs** (USRP, LimeSDR, BladeRF) cover LTE bands at small-cell power for hundreds of dollars.
- **Shared spectrum is real**: CBRS in the US, local licensing in the UK/EU, private LTE bands.
- **Private LTE + Wi-Fi calling** give lawful templates to copy instead of pretending regulation doesn't exist.
- Mini-PC class hardware (and Raspberry Pi CM4/5 + HATs for dev) is cheap and adequate.

---

## Quickstart — first UE attach in <30 minutes (no RF, no license needed)

> Requirements: Docker Engine 24+, 8 GB RAM. Everything runs in containers; nothing transmits.
> For the full data path (UE IP + ping) use native Linux; Docker Desktop on
> Windows/macOS passes all EPC-side attach checks (see [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # checks/installs toolchains (Go, Docker, pre-commit)
make lab-up                 # builds images, brings up EPC + eNB + srsUE, runs attach checks
make status                 # one-glance health: mme, sgwu/upf, enb, ue1
```

`make lab-up` asserts (and prints) all of the following:

1. Open5GS MME + HSS running
2. eNB S1-MME connected to the MME
3. UE RRC connection + random access on the lab PLMN
4. UE NAS authentication + security mode (milenage vs HSS)
5. MME creates the default EPS bearer and sends Attach Accept (UE IP allocated)

Then look inside:

```bash
fairwave node status                     # control-plane view of this cluster
fairwave sim issue --count 2 --profile lab  # mint two lab SIMs
fairwave spectrum check --country US --band n48 --indoor  # spectrum gate demo
make lab-down                            # stop everything
```

(On Linux hosts with stable ZMQ timing, `docker exec -it ue1 ping -c3 10.45.0.1`
exercises the full data path over the `tun_srsue` tunnel.)

Everything above is **RF-silent**: the radio is emulated with `srsran/zmq` virtual RF devices.
For real hardware, follow [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) —
which will not let you enable TX without completing the legal checklist.

## Documentation

Full site: **run `make docs-serve`** — or read it in-tree under [`docs/`](docs/index.md).

| Start here | Then |
|---|---|
| [Vision](docs/vision.md) | [Architecture](docs/architecture/overview.md) |
| [Quickstart (30 min, no RF)](docs/tutorials/quickstart-no-rf.md) | [Café pilot (2 h, with legal checklist)](docs/tutorials/cafe-pilot.md) |
| [Spectrum & law](docs/spectrum-and-law/index.md) | [Threat model](design/threat-model.md) |
| [SIM lifecycle](docs/sim-lifecycle/index.md) | [Peering fabric](docs/peering/index.md) |
| [API reference](docs/api/index.md) | [ADRs](docs/adr/0000-index.md) |

## Status

**Lab release `v0.1.0`**: EPC + zmq RAN + srsUE attach works end-to-end; control plane, CLI,
SIM provisioner, peering MVP, and the docs site are functional. Real-RF paths are validated
on dev hardware but **disabled by default**. See the [roadmap](design/roadmap.md).

## What Fairwave is NOT

- **Not an IMSI catcher** — there is no passive interrogation; UEs must authenticate against credentials you provisioned.
- **Not a spectrum free-for-all** — TX is gated, and we refuse regulatory bypass features.
- **Not a free national carrier** — it is local coverage with optional breakout.
- **Not a replacement for emergency calling** — plan 911/112 behavior in every deployment ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Contributing & governance

We welcome contributors. Read [CONTRIBUTING.md](CONTRIBUTING.md) (code style, DCO, tests),
[GOVERNANCE.md](GOVERNANCE.md) (how decisions are made), [SECURITY.md](SECURITY.md)
(vulnerability disclosure; threat model), and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
License: **Apache-2.0** ([LICENSE](LICENSE)); third-party attribution in [NOTICE](NOTICE).

---

<p align="center">
  <sub>Built by the HyperonX team and the Fairwave community. The air is for everyone.</sub>
</p>
