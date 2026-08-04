<h1 align="center">Fairwave - ein Community-Carrier in einer Pizzabox</h1>

<p align="center">
  <strong>Open-Source-Privat-LTE: einstecken, 4G senden, Nachbarn willkommen heißen.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **RECHTLICHER UND SPEKTRUM-HINWEIS.** Fairwave läuft standardmäßig im **Lab / No-RF-Modus** (nur Zero-IF-Rückkopplung).
> Das Senden auf Mobilfunkbändern ohne ordnungsgemäße Genehmigung ist **in den meisten Ländern illegal**.
> Sie sind allein verantwortlich für Lizenzen, SAS-Grants, Indoor-Beschränkungen und Typprüfung.
> HyperonX und Mitwirkende stellen die Software **wie besehen** nur für rechtmäßige private Netze, Forschung
> und Shared-Spectrum-Regime bereit. Siehe [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Lesen in:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## Das kaputte System

Mobile Konnektivität ist ein Ein-Mann-Kartell: eine SIM, ein Carrier, ein Vertrag, ein ummauerter Garten.
Abdeckungskarten sind Marketing-Broschüren; ländliche Straßen fallen aus ihnen heraus; Wohnungen verlieren das Signal;
und Sie - die zahlende Person - besitzen keine der Infrastrukturen, die Sie versorgen. Wenn Ihr nationaler Carrier
den Preis erhöht oder sein Turm schweigt, bleibt Ihnen nur… ein anderes Monopol mit denselben Türmen und denselben Bedingungen.

Das Modem in Ihrer Tasche kann mit einer Basisstation 20 Meter entfernt sprechen. Der einzige Grund, warum es nicht mit
*Ihrer* Basisstation spricht, ist, dass diese Basisstation nie erlaubt war, Ihnen zu gehören.

## Der HyperonX-Fix

**Fairwave ist der Community-Carrier: eine vollständige Open-Source-Kleinzelle, die in eine Pizzabox passt
und an gewöhnliches Ethernet angeschlossen wird.**

Ein Café, eine Wohngenossenschaft, ein Gemeindesaal, ein Stadtviertel - jeder kann eine betreiben:

1. Starten Sie das Fairwave-Image auf einem Mini-PC (x86 oder ARM) mit angeschlossenem SDR.
2. Führen Sie `fairwave node init` aus und beantworten Sie den Regulierungs-Checklist (TX bleibt aus, bis Sie eine Autorisierung nachweisen).
3. Stellen Sie Fairwave-SIMs für Ihre Community aus - Karten, die Ihnen gehören, Zugangsdaten, die Sie kontrollieren.
4. Telefone verbinden sich. Verkehr bleibt wo möglich lokal; mehrere Pizzaboxen bilden ein Mesh;
   Internet-Breakout fährt bei Bedarf über einen sicheren WireGuard-Tunnel.

Gebaut auf bewährter offener Infrastruktur - **Open5GS** (EPC) und **srsRAN** (eNB/gNB) - mit einer Go-Control-Plane,
einem Operator-Dashboard, Offline-SIM-Provisionierung und einem Lab-Modus, der den gesamten Carrier
mit null HF in Docker betreibt.

## Was Sie bekommen

- **Eine Pizzabox, die ein Carrier ist**: Node-Identität, Enrollment, Lebenszyklus
  (`provision → register → on-air → peer → breakout`) verwaltet von `fairwave-control`.
- **Echter LTE-Attach**: Open5GS EPC + srsRAN eNB, konfigurierbares PLMN, Tracking Areas,
  `internet` + `ims` APNs, lokaler Breakout am Edge.
- **Fairwave-SIM-Betrieb**: Offline-First-Provisionierer; generiert Ki/OPc, CSV/JSON-Batches für
  Kartenbüros, schreibt HSS/UDM, Sperr- und Tauschkontrollen. Lab und Produktion strikt getrennt.
- **Lab-eSIM (SM-DP+)**: Ihr eigener SGP.22-förmiger Profilserver und eine Software-eUICC -
  verschlüsselte Bound-Profile-Pakete, QR-Aktivierungscodes (`LPA:1$...`), kompletter Download-Zyklus
  CI-verifiziert ohne Hardware. Designbedingt nur Lab; GSMA-Konformität als offene Punkte verfolgt.
- **Nachbarschafts-Mesh**: mDNS-Discovery, mTLS-Steuerung, WireGuard-Datenebene, Routenaustausch.
- **Operator-Portal**: Local-First-Dashboard - UEs (datenschutzerhaltend), Backhaul, Peers, Lab-Modus.
- **Voller Lab-Modus**: das gesamte Netz auf reinem IP (zmq) mit srsUE - ohne Funk, ohne Lizenz, CI-freundlich.
- **Regulierungs-Gates im Code**: TX-Scharfschaltung erfordert Ländercode, Lizenzbestätigung
  und Frequenz-Whitelist. Andernfalls verweigert die Kompilierung.

## Warum es die etablierten Anbieter bedroht

Carrier-Lock-in wird *optional für lokale Abdeckung*. Wenn Café, Genossenschaft und Dorf jeweils eine Zelle
betreiben und sie vermaschen können, ist die nationale SIM nur noch eine Roaming-Option, nicht mehr der Torwächter.
Offload, Neutral-Host-Zugang und Community-Preismodelle hören auf, theoretisch zu sein. Fairwave kopiert nicht
das Telekom-Kartell - es macht das lokale Monopol des Kartells angreifbar.

## Warum es jetzt machbar ist

- **Open5GS** und **srsRAN** sind ausgereift, produktionsnah und aktiv weiterentwickelt.
- **SDRs** (USRP, LimeSDR, BladeRF) decken LTE-Bänder mit Kleinzellen-Leistung für hunderte Dollar ab.
- **Shared Spectrum ist real**: CBRS in den USA, lokale Lizenzen in UK/EU, private LTE-Bänder.
- **Privates LTE + Wi-Fi Calling** liefern legale Vorlagen zum Kopieren, statt so zu tun, als gäbe es keine Regulierung.
- Mini-PC-Hardware (und Raspberry Pi CM4/5 + HATs für Entwicklung) ist günstig und ausreichend.

---

## Schnellstart - erster UE-Attach in <30 Minuten (ohne HF, ohne Lizenz)

> Voraussetzungen: Docker Engine 24+, 8 GB RAM. Alles läuft in Containern; nichts sendet.
> Für den vollständigen Datenpfad (UE-IP + Ping) natives Linux verwenden; Docker Desktop auf
> Windows/macOS besteht alle EPC-seitigen Attach-Checks (siehe [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # prüft/installiert Toolchains (Go, Docker, pre-commit)
make lab-up                 # baut Images, startet EPC + eNB + srsUE, führt Attach-Checks aus
make status                 # Gesundheit auf einen Blick: mme, sgwu/upf, enb, ue1
```

`make lab-up` prüft (und druckt) alle folgenden Punkte:

1. Open5GS MME + HSS laufen
2. eNB S1-MME mit dem MME verbunden
3. UE-RRC-Verbindung + Random Access auf dem Lab-PLMN
4. UE-NAS-Authentifizierung + Sicherheitsmodus (Milenage gegen HSS)
5. MME erstellt den Standard-EPS-Bearer und sendet Attach Accept (UE-IP zugewiesen)

Dann schauen Sie hinein:

```bash
fairwave node status                     # Control-Plane-Ansicht dieses Clusters
fairwave sim issue --count 2 --profile lab  # zwei Lab-SIMs ausstellen
fairwave spectrum check --country US --band n48 --indoor  # Spektrum-Gate-Demo
make lab-down                            # alles stoppen
```

(Auf Linux-Hosts mit stabilem ZMQ-Timing übt `docker exec -it ue1 ping -c3 10.45.0.1`
den vollständigen Datenpfad über den `tun_srsue`-Tunnel.)

Alles oben Genannte ist **HF-still**: Das Funkgerät wird mit virtuellen `srsran/zmq`-Geräten emuliert.
Für echte Hardware folgen Sie [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
das lässt Sie TX nicht aktivieren, ohne die rechtliche Checkliste abzuschließen.

## Dokumentation

Komplette Website: **`make docs-serve` ausführen** - oder im Baum unter [`docs/`](docs/index.md) lesen.

| Hier starten | Dann |
|---|---|
| [Vision](docs/vision.md) | [Architektur](docs/architecture/overview.md) |
| [Schnellstart (30 Min., ohne HF)](docs/tutorials/quickstart-no-rf.md) | [Café-Pilot (2 Std., mit rechtlicher Checkliste)](docs/tutorials/cafe-pilot.md) |
| [Spektrum & Recht](docs/spectrum-and-law/index.md) | [Bedrohungsmodell](design/threat-model.md) |
| [SIM-Lebenszyklus](docs/sim-lifecycle/index.md) | [Peering-Geflecht](docs/peering/index.md) |
| [API-Referenz](docs/api/index.md) | [ADRs](docs/adr/0000-index.md) |

## Status

**Lab-Version `v0.1.0`**: EPC + zmq-RAN + srsUE-Attach funktioniert Ende-zu-Ende; Control-Plane, CLI,
SIM-Provisionierer, Lab-eSIM (SM-DP+) und die Docs-Website sind funktionsfähig. Echte RF-Pfade sind auf
Entwicklungshardware validiert, aber **standardmäßig deaktiviert**. Siehe [Roadmap](design/roadmap.md).

## Was Fairwave NICHT ist

- **Kein IMSI-Catcher** - keine passive Abfrage; UEs müssen sich mit von Ihnen bereitgestellten Zugangsdaten authentifizieren.
- **Kein Spektrum-Wildwuchs** - TX ist gated, und wir lehnen Regulierungs-Umgehungsfunktionen ab.
- **Kein kostenloser nationaler Carrier** - es ist lokale Abdeckung mit optionalem Breakout.
- **Kein Ersatz für Notrufe** - planen Sie 911/112-Verhalten in jedem Deployment ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Mitwirken und Governance

Wir begrüßen Mitwirkende. Lesen Sie [CONTRIBUTING.md](CONTRIBUTING.md) (Codestil, DCO, Tests),
[GOVERNANCE.md](GOVERNANCE.md) (wie Entscheidungen getroffen werden), [SECURITY.md](SECURITY.md)
(Schwachstellenmeldung; Bedrohungsmodell) und [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Lizenz: **Apache-2.0** ([LICENSE](LICENSE)); Drittanbieter-Attribution in [NOTICE](NOTICE).

---

<p align="center">
  <sub>Gebaut vom HyperonX-Team und der Fairwave-Community. Die Luft gehört allen.</sub>
</p>
