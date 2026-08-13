<h1 align="center">Fairwave - en gemenskapsoperatör i en pizzakartong</h1>

<p align="center">
  <strong>Öppen källkod privat LTE: koppla in i ethernet, sänd 4G/5G, välkomna dina grannar.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **JURIDISK OCH SPEKTRUMVARNING.** Fairwave kör som standard i **lab / ingen-RF**-läge (endast noll-IF-slinga).
> Att sända på mobila band utan vederbörlig auktorisation är **olagligt i de flesta jurisdiktioner**.
> Du är ensam ansvarig för licenser, SAS-tillstånd, inomhusbegränsningar och typgodkännande.
> HyperonX och bidragsgivare tillhandahåller programvaran **i befintligt skick** endast för lagliga privata nätverk, forskning
> och regimer för delat spektrum. Se [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Läs på:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## Det trasiga systemet

Mobilanslutning är ett enmans-kartell: ett SIM, en operatör, ett kontrakt, en muromgärdad trädgård.
Täckningskartor är marknadsföringsbroschyrer; landsbygdsgator faller ur dem; lägenheter förlorar signal;
och du - personen som betalar - äger ingen av infrastrukturerna som betjänar dig. Om din nationella operatör
höjer priset eller dess torn tystnar, är ditt enda val… ett annat monopol med samma torn och samma villkor.

Modemet i fickan kan prata med en basstation 20 meter bort. Den enda anledningen till att det inte pratar
med *din* basstation är att den stationen aldrig tilläts vara din.

## HyperonX-lösningen

**Fairwave är gemenskapsoperatören: en komplett öppen källkods-småcell som får plats i en pizzakartong
och kopplas in i vanlig ethernet.**

Ett café, en bostadsrättsförening, en bygdegård, en stadsdel - vem som helst kan driva en:

1. Starta Fairwave-avbilden på en mini-PC (x86 eller ARM) med en ansluten SDR.
2. Kör `fairwave node init`, svara på den regulatoriska checklistan (TX förblir avstängd tills du bevisar auktorisation).
3. Ge ut Fairwave-SIM till din gemenskap - kort du äger, referenser du kontrollerar.
4. Telefoner ansluter. Trafiken förblir lokal när det är möjligt; flera pizzakartonger bildar ett mesh;
   internetuppkoppling går via en säker WireGuard-tunnel när du vill.

Byggt på beprövad öppen infrastruktur - **Open5GS** (EPC) och **srsRAN** (eNB/gNB) - med en Go-kontrollplans,
en operatörspanel, offline-SIM-försörjning och ett lab-läge som kör hela operatören i Docker med noll RF.

## Vad du får

- **En pizzakartong som är en operatör**: nodidentitet, registrering, livscykel
  (`provision → register → on-air → peer → breakout`) hanterad av `fairwave-control`.
- **Äkta LTE-anslutning**: Open5GS EPC + srsRAN eNB, konfigurerbart PLMN, spårningsområden,
  `internet` + `ims`-APN:er, lokal breakout i kanten.
- **Fairwave-SIM-drift**: offline-first-försörjare; genererar Ki/OPc, CSV/JSON-batcher för
  kortbyråer, skriver till HSS/UDM, återkallnings- och byteskontroller. Lab och produktion strikt åtskilda.
- **Lab-eSIM (SM-DP+)**: din egen SGP.22-formade profils server och en programvaru-eUICC -
  krypterade bundna profils paket, QR-aktiveringskoder (`LPA:1$...`), fullständig nedladdningscykel
  CI-verifierad utan hårdvara. Endast labb genom design; GSMA-efterlevnad följs som öppna punkter.
- **Grannskaps-mesh**: mDNS-upptäckt, mTLS-kontroll, WireGuard-dataplan, ruttutbyte.
- **Operatörsportal**: lokal-first-panel - UE:n (integritetsbevarande), backhaul, peers, lab-läge.
- **Fullständigt lab-läge**: hela nätverket på ren IP (zmq) med srsUE - utan radio, utan licens, CI-vänligt.
- **Regulatoriska grindar i koden**: TX-skarpsättning kräver landskod, licensbekräftelse
  och frekvensvitlista. Annars kompileringstidsvägran.

## Varför det hotar etablerade operatörer

Operatörslås blir *valfritt för lokal täckning*. När caféet, föreningen och byn var och en kan driva en cell
och koppla dem i mesh, är det nationella SIM:et bara ett roamingalternativ, inte grindvakten. Offload,
neutral-host-åtkomst och gemenskapsprissatta abonnemang slutar vara teori. Fairwave replikerar inte
telekomkartellen - det gör kartellens lokala monopol utmaningsbart.

## Varför det är möjligt nu

- **Open5GS** och **srsRAN** är mogna, nära produktion och aktivt utvecklade.
- **SDR:er** (USRP, LimeSDR, BladeRF) täcker LTE-band med småcellseffekt för hundratals dollar.
- **Delat spektrum är verkligt**: CBRS i USA, lokala licenser i Storbritannien/EU, privata LTE-band.
- **Privat LTE + Wi-Fi-samtal** ger lagliga mallar att kopiera i stället för att låtsas att reglering inte finns.
- Mini-PC-klassens hårdvara (och Raspberry Pi CM4/5 + HAT för utveckling) är billig och tillräcklig.

---

## Snabbstart - första UE-anslutning på <30 minuter (utan RF, utan licens)

> Krav: Docker Engine 24+, 8 GB RAM. Allt körs i containrar; ingenting sänder.
> För fullständig dataväg (UE-IP + ping) använd nativ Linux; Docker Desktop på
> Windows/macOS klarar alla EPC-sidiga anslutningskontroller (se [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # kontrollerar/installerar verktygskedjor (Go, Docker, pre-commit)
make lab-up                 # bygger avbilder, startar EPC + eNB + srsUE, kör anslutningskontroller
make status                 # hälsa i ett ögonkast: mme, sgwu/upf, enb, ue1
```

`make lab-up` verifierar (och skriver ut) allt av följande:

1. Open5GS MME + HSS körs
2. eNB S1-MME ansluten till MME
3. UE-RRC-anslutning + slumpmässig åtkomst på labb-PLMN:et
4. UE-NAS-autentisering + säkerhetsläge (milenage mot HSS)
5. MME skapar standard-EPS-bäraren och skickar Attach Accept (UE-IP tilldelad)

Titta sedan inuti:

```bash
fairwave node status                     # kontrollplansvy över detta kluster
fairwave sim issue --count 2 --profile lab  # utfärda två lab-SIM
fairwave spectrum check --country US --band n48 --indoor  # spektrumgrindsdemo
make lab-down                            # stoppa allt
```

(På Linux-värdar med stabil ZMQ-timing övar `docker exec -it ue1 ping -c3 10.45.0.1`
den fullständiga datavägen över `tun_srsue`-tunneln.)

Allt ovan är **RF-tyst**: radion emuleras med virtuella `srsran/zmq`-enheter.
För riktig hårdvara, följ [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
som inte låter dig aktivera TX utan att slutföra den juridiska checklistan.

## Dokumentation

Fullständig webbplats: **kör `make docs-serve`** - eller läs i trädet under [`docs/`](docs/index.md).

| Börja här | Sedan |
|---|---|
| [Vision](docs/vision.md) | [Arkitektur](docs/architecture/overview.md) |
| [Snabbstart (30 min, utan RF)](docs/tutorials/quickstart-no-rf.md) | [Cafépilot (2 timmar, med juridisk checklista)](docs/tutorials/cafe-pilot.md) |
| [Spektrum och lag](docs/spectrum-and-law/index.md) | [Hotmodell](design/threat-model.md) |
| [SIM-livscykel](docs/sim-lifecycle/index.md) | [Peering-väv](docs/peering/index.md) |
| [API-referens](docs/api/index.md) | [ADR:er](docs/adr/0000-index.md) |

## Status

**Labbrelease `v0.1.0`**: EPC + zmq-RAN + srsUE-anslutning fungerar end-to-end; kontrollplans, CLI,
SIM-försörjare, labb-eSIM (SM-DP+) och dokumentationssajten är funktionella. Dessutom levereras en free5GC 5G SA-kärna med CHF-baserad förbrukningsmätning (valfritt, `core: free5gc`), med ZMQ gNB/UE-labbprofiler och ett CI-attach-test. Riktiga RF-vägar är validerade
på utvecklingshårdvara men **avstängda som standard**. Se [färdplanen](design/roadmap.md).

## Vad Fairwave INTE är

- **Inte en IMSI-fångare** - ingen passiv förfrågan; UE:n måste autentisera med referenser du försörjt.
- **Inte en spektrumvilda västern** - TX är grindad, och vi vägrar regleringskringgåendefunktioner.
- **Inte en gratis nationell operatör** - det är lokal täckning med valfri breakout.
- **Inte ett alternativ till nödsamtal** - planera 911/112-beteende i varje driftsättning ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Bidra och styrning

Vi välkomnar bidragsgivare. Läs [CONTRIBUTING.md](CONTRIBUTING.md) (kodstil, DCO, tester),
[GOVERNANCE.md](GOVERNANCE.md) (hur beslut fattas), [SECURITY.md](SECURITY.md)
(sårbarhetsrapportering; hotmodell) och [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Licens: **Apache-2.0** ([LICENSE](LICENSE)); tredjepartsattribution i [NOTICE](NOTICE).

---

<p align="center">
  <sub>Byggt av HyperonX-teamet och Fairwave-gemenskapen. Luften är till för alla.</sub>
</p>
