<h1 align="center">Fairwave - een community carrier in een pizzadoos</h1>

<p align="center">
  <strong>Open-source privaat LTE: steek het in de ethernet, zend 4G uit, verwelkom je buren.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **JURIDISCHE EN SPECTRUM WAARSCHUWING.** Fairwave werkt standaard in de **lab / geen-RF**-modus (alleen zero-IF-loopback).
> Zenden op mobiele banden zonder de juiste machtiging is **in de meeste landen illegaal**.
> U bent zelf verantwoordelijk voor licenties, SAS-vergunningen, binnenbeperkingen en typegoedkeuring.
> HyperonX en bijdragers leveren de software **zoals deze is** uitsluitend voor legale privénetwerken, onderzoek
> en gedeeld-spectrumregimes. Zie [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Lees in:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## Het kapotte systeem

Mobiele connectiviteit is een kartel van één: één simkaart, één carrier, één contract, één ommuurde tuin.
Dekkingskaarten zijn marketingbrochures; plattelandsstraten vallen eruit; appartementen verliezen signaal;
en jij - de persoon die betaalt - bezit geen van de infrastructuur die jou bedient. Als je nationale carrier
de prijs verhoogt of zijn toren zwijgt, is je enige optie… een ander monopolie met dezelfde torens en dezelfde voorwaarden.

De modem in je zak kan praten met een basisstation op 20 meter. De enige reden dat hij niet met *jouw*
basisstation praat, is dat dat basisstation nooit van jou mocht zijn.

## De HyperonX-oplossing

**Fairwave is de community carrier: een complete open-source kleine cel die in een pizzadoos past
en op gewone ethernet wordt aangesloten.**

Een café, een wooncoöperatie, een dorpshuis, een buurt - iedereen kan er één runnen:

1. Start de Fairwave-image op een mini-pc (x86 of ARM) met een aangesloten SDR.
2. Voer `fairwave node init` uit, beantwoord de regelgevingschecklist (TX blijft uit totdat je machtiging bewijst).
3. Geef Fairwave-simkaarten uit aan je community - kaarten die jij bezit, credentials die jij beheert.
4. Telefoons maken verbinding. Verkeer blijft waar mogelijk lokaal; meerdere pizzadozen vormen een mesh;
   internettoegang gaat via een veilige WireGuard-tunnel wanneer je dat wilt.

Gebouwd op bewezen open infrastructuur - **Open5GS** (EPC) en **srsRAN** (eNB/gNB) - met een Go
controlelaag, een operator-dashboard, offline simkaartvoorziening en een labmodus die de hele carrier
in Docker draait met nul RF.

## Wat je krijgt

- **Een pizzadoos die een carrier is**: node-identiteit, inschrijving, levenscyclus
  (`provision → register → on-air → peer → breakout`) beheerd door `fairwave-control`.
- **Echte LTE-attach**: Open5GS EPC + srsRAN eNB, configureerbaar PLMN, tracking areas,
  `internet` + `ims` APN's, lokale breakout aan de rand.
- **Fairwave-simkaartoperaties**: offline-first provisioner; genereert Ki/OPc, CSV/JSON-batches voor
  kaartbureaus, schrijft naar HSS/UDM, intrek- en wisselcontroles. Lab en productie strikt gescheiden.
- **Lab-eSIM (SM-DP+)**: je eigen SGP.22-vormige profielserver en een software-eUICC -
  versleutelde gebonden profielpakketten, QR-activeringscodes (`LPA:1$...`), volledige downloadcyclus
  CI-geverifieerd zonder hardware. Alleen lab door ontwerp; GSMA-conformiteit gevolgd als openstaande punten.
- **Buurt-mesh**: mDNS-detectie, mTLS-besturing, WireGuard-dataplane, route-uitwisseling.
- **Operatorportaal**: local-first dashboard - UEs (privacybehoudend), backhaul, peers, labmodus.
- **Volledige labmodus**: het hele netwerk op puur IP (zmq) met srsUE - zonder radio, zonder licentie, CI-vriendelijk.
- **Regelgevende poorten in code**: TX-scherper stellen vereist landcode, licentiebevestiging
  en frequentiewitte-lijst. Anders compile-time weigering.

## Waarom het gevestigde carriers bedreigt

Carrier-lock-in wordt *optioneel voor lokale dekking*. Wanneer het café, de coöperatie en het dorp elk een cel
kunnen runnen en ze in een mesh verbinden, is de nationale simkaart slechts een roamingoptie, niet de poortwachter.
Offload, neutral-host-toegang en community-prijsplannen worden niet langer theoretisch. Fairwave kopieert het
telecomkartel niet - het maakt het lokale monopolie van het kartel betwistbaar.

## Waarom het nu haalbaar is

- **Open5GS** en **srsRAN** zijn volwassen, productie-nabij en actief ontwikkeld.
- **SDR's** (USRP, LimeSDR, BladeRF) dekken LTE-bandden op small-cell-vermogen voor honderden dollars.
- **Gedeeld spectrum is echt**: CBRS in de VS, lokale licenties in VK/EU, privé-LTE-bandden.
- **Privé-LTE + Wi-Fi-bellen** geven legale sjablonen om te kopiëren in plaats van te doen alsof regulering niet bestaat.
- Mini-pc-hardware (en Raspberry Pi CM4/5 + HAT's voor ontwikkeling) is goedkoop en voldoende.

---

## Snelstart - eerste UE-attach in <30 minuten (zonder RF, zonder licentie)

> Vereisten: Docker Engine 24+, 8 GB RAM. Alles draait in containers; niets zendt.
> Gebruik voor het volledige datapad (UE-IP + ping) native Linux; Docker Desktop op
> Windows/macOS doorstaat alle EPC-zijde attach-checks (zie [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # controleert/installeert toolchains (Go, Docker, pre-commit)
make lab-up                 # bouwt images, start EPC + eNB + srsUE, voert attach-checks uit
make status                 # gezondheid in één oogopslag: mme, sgwu/upf, enb, ue1
```

`make lab-up` controleert (en print) alle volgende punten:

1. Open5GS MME + HSS draaien
2. eNB S1-MME verbonden met de MME
3. UE-RRC-verbinding + willekeurige toegang op het lab-PLMN
4. UE-NAS-authenticatie + beveiligingsmodus (milenage tegen HSS)
5. De MME maakt de standaard-EPS-bearer aan en stuurt Attach Accept (UE-IP toegewezen)

Kijk dan naar binnen:

```bash
fairwave node status                     # controlelaag-weergave van dit cluster
fairwave sim issue --count 2 --profile lab  # geef twee lab-simkaarten uit
fairwave spectrum check --country US --band n48 --indoor  # spectrum-poortdemo
make lab-down                            # stop alles
```

(Op Linux-hosts met stabiele ZMQ-timing oefent `docker exec -it ue1 ping -c3 10.45.0.1`
het volledige datapad over de `tun_srsue`-tunnel.)

Al het bovenstaande is **RF-stil**: de radio wordt geëmuleerd met virtuele `srsran/zmq`-apparaten.
Voor echte hardware volg je [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
dat je TX niet laat inschakelen zonder de juridische checklist af te ronden.

## Documentatie

Volledige site: **voer `make docs-serve` uit** - of lees in de boom onder [`docs/`](docs/index.md).

| Begin hier | Daarna |
|---|---|
| [Visie](docs/vision.md) | [Architectuur](docs/architecture/overview.md) |
| [Snelstart (30 min, zonder RF)](docs/tutorials/quickstart-no-rf.md) | [Café-pilot (2 uur, met juridische checklist)](docs/tutorials/cafe-pilot.md) |
| [Spectrum en recht](docs/spectrum-and-law/index.md) | [Bedreigingsmodel](design/threat-model.md) |
| [Simkaartlevenscyclus](docs/sim-lifecycle/index.md) | [Peering-weefsel](docs/peering/index.md) |
| [API-referentie](docs/api/index.md) | [ADR's](docs/adr/0000-index.md) |

## Status

**Labrelease `v0.1.0`**: EPC + zmq-RAN + srsUE-attach werkt end-to-end; controlelaag, CLI,
simkaartvoorziening, lab-eSIM (SM-DP+) en de documentatiesite zijn functioneel. Echte RF-paden zijn
gevalideerd op ontwikkelhardware maar **standaard uitgeschakeld**. Zie de [roadmap](design/roadmap.md).

## Wat Fairwave NIET is

- **Geen IMSI-catcher** - geen passief ondervragen; UEs moeten authenticeren met credentials die jij hebt voorzien.
- **Geen spectrum-wildwest** - TX is geporteerd, en we weigeren regelgevingsbypass-functies.
- **Geen gratis nationale carrier** - het is lokale dekking met optionele breakout.
- **Geen vervanging voor noodoproepen** - plan 911/112-gedrag in elke implementatie ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Bijdragen en bestuur

We verwelkomen bijdragers. Lees [CONTRIBUTING.md](CONTRIBUTING.md) (codestijl, DCO, tests),
[GOVERNANCE.md](GOVERNANCE.md) (hoe beslissingen worden genomen), [SECURITY.md](SECURITY.md)
(kwetsbaarheidsmelding; bedreigingsmodel) en [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Licentie: **Apache-2.0** ([LICENSE](LICENSE)); attriuties van derden in [NOTICE](NOTICE).

---

<p align="center">
  <sub>Gebouwd door het HyperonX-team en de Fairwave-community. De lucht is voor iedereen.</sub>
</p>
