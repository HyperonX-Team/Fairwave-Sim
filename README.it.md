<h1 align="center">Fairwave - un operatore comunitario in una scatola di pizza</h1>

<p align="center">
  <strong>LTE privato open source: collegalo all'Ethernet, emetti 4G, accogli i tuoi vicini.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **AVVISO LEGALE E DI SPETTRO.** Fairwave funziona per impostazione predefinita in modalità **laboratorio / senza RF** (solo loopback a FI zero).
> Trasmettere su bande cellulari senza la dovuta autorizzazione è **illegale nella maggior parte dei Paesi**.
> Sei l'unico responsabile di licenze, concessioni SAS, restrizioni indoor e omologazione di tipo.
> HyperonX e i contributori forniscono il software **così com'è** solo per reti private legali, ricerca
> e regimi di spettro condiviso. Vedi [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Leggi in:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## Il sistema rotto

La connettività mobile è un cartello di uno: una SIM, un operatore, un contratto, un giardino recintato.
Le mappe di copertura sono brochure di marketing; le strade rurali ne cadono fuori; gli appartamenti perdono il segnale;
e tu - la persona che paga - non possiedi nessuna delle infrastrutture che ti servono. Se il tuo operatore
nazionale alza i prezzi o la sua torre tace, la tua unica opzione è… un monopolio diverso con le stesse torri e le stesse condizioni.

Il modem in tasca può parlare con una stazione base a 20 metri. L'unica ragione per cui non parla con la
*tua* stazione base è che quella stazione non è mai stata autorizzata a essere tua.

## La soluzione HyperonX

**Fairwave è l'operatore comunitario: una small cell completa e open source che sta in una scatola di
pizza e si collega a una normale Ethernet.**

Un bar, una cooperativa edilizia, una sala del paese, un quartiere - chiunque può gestirne una:

1. Avvia l'immagine Fairwave su un mini-PC (x86 o ARM) con un SDR collegato.
2. Esegui `fairwave node init`, rispondi alla checklist normativa (la TX resta spenta finché non dimostri l'autorizzazione).
3. Emetti SIM Fairwave per la tua comunità - carte che possiedi, credenziali che controlli.
4. I telefoni si collegano. Il traffico resta locale quando possibile; più scatole di pizza formano una mesh;
   l'accesso a internet viaggia su un tunnel WireGuard sicuro quando vuoi.

Costruito su infrastruttura aperta collaudata - **Open5GS** (EPC) e **srsRAN** (eNB/gNB) - con un piano di
controllo in Go, un dashboard operatore, provisioning SIM offline e una modalità laboratorio che esegue
l'intero operatore in Docker con zero RF.

## Cosa ottieni

- **Una scatola di pizza che è un operatore**: identità del nodo, enrollement, ciclo di vita
  (`provision → register → on-air → peer → breakout`) gestito da `fairwave-control`.
- **Attach LTE reale**: Open5GS EPC + srsRAN eNB, PLMN configurabile, tracking area,
  APN `internet` + `ims`, breakout locale al bordo.
- **Operazioni SIM Fairwave**: provisioner offline-first; genera Ki/OPc, batch CSV/JSON per
  le agenzie di carte, scrive in HSS/UDM, controlli di revoca e sostituzione. Lab e produzione rigorosamente separati.
- **eSIM di laboratorio (SM-DP+)**: il tuo server di profili in stile SGP.22 e una eUICC software -
  pacchetti di profilo cifrati, codici di attivazione QR (`LPA:1$...`), ciclo di download completo
  verificato in CI senza hardware. Solo lab per progetto; la conformità GSMA è tracciata come voci aperte.
- **Mesh di vicinato**: discovery mDNS, controllo mTLS, piano dati WireGuard, scambio di route.
- **Portale operatore**: dashboard local-first - UE (rispettoso della privacy), backhaul, peer, modalità lab.
- **Modalità laboratorio completa**: tutta la rete su IP puro (zmq) con srsUE - senza radio, senza licenza, amico della CI.
- **Cancelli normativi nel codice**: armare la TX richiede codice paese, riconoscimento licenza
  e whitelist delle frequenze. Rifiuto compilato altrimenti.

## Perché minaccia gli incumbent

Il lock-in dell'operatore diventa *opzionale per la copertura locale*. Quando il bar, la cooperativa e il
paese possono gestire ciascuno una cella e collegarle in mesh, la SIM nazionale è solo un'opzione di roaming,
non il custode. L'offload, l'accesso neutral host e i piani a prezzo comunitario smettono di essere teoria.
Fairwave non replica il cartello delle telecomunicazioni - rende contendibile il monopolio locale del cartello.

## Perché è fattibile ora

- **Open5GS** e **srsRAN** sono maturi, vicini alla produzione e sviluppati attivamente.
- **Gli SDR** (USRP, LimeSDR, BladeRF) coprono le bande LTE a potenza di small cell per poche centinaia di dollari.
- **Lo spettro condiviso è reale**: CBRS negli USA, licenze locali in UK/UE, bande LTE private.
- **LTE privato + chiamate Wi-Fi** offrono modelli legali da copiare invece di fingere che la regolamentazione non esista.
- L'hardware classe mini-PC (e Raspberry Pi CM4/5 + HAT per lo sviluppo) è economico e sufficiente.

---

## Avvio rapido - primo attach UE in <30 minuti (senza RF, senza licenza)

> Requisiti: Docker Engine 24+, 8 GB di RAM. Tutto gira in container; nulla trasmette.
> Per il percorso dati completo (IP UE + ping) usa Linux nativo; Docker Desktop su
> Windows/macOS supera tutti i controlli di attach lato EPC (vedi [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # verifica/installa le toolchain (Go, Docker, pre-commit)
make lab-up                 # costruisce le immagini, avvia EPC + eNB + srsUE, esegue i controlli di attach
make status                 # salute a colpo d'occhio: mme, sgwu/upf, enb, ue1
```

`make lab-up` verifica (e stampa) tutto quanto segue:

1. Open5GS MME + HSS in esecuzione
2. eNB S1-MME connesso al MME
3. Connessione RRC dell'UE + accesso casuale sul PLMN di laboratorio
4. Autenticazione NAS dell'UE + modalità di sicurezza (milenage contro HSS)
5. Il MME crea il bearer EPS predefinito e invia Attach Accept (IP UE assegnato)

Poi guarda dentro:

```bash
fairwave node status                     # vista del piano di controllo di questo cluster
fairwave sim issue --count 2 --profile lab  # emette due SIM di laboratorio
fairwave spectrum check --country US --band n48 --indoor  # demo del cancello spettrale
make lab-down                            # ferma tutto
```

(Su host Linux con timing ZMQ stabile, `docker exec -it ue1 ping -c3 10.45.0.1`
esercita il percorso dati completo sul tunnel `tun_srsue`.)

Tutto quanto sopra è **silenzioso in RF**: la radio è emulata con dispositivi virtuali `srsran/zmq`.
Per hardware reale, segui [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
che non ti lascerà attivare la TX senza completare la checklist legale.

## Documentazione

Sito completo: **esegui `make docs-serve`** - oppure leggilo nell'albero sotto [`docs/`](docs/index.md).

| Inizia qui | Poi |
|---|---|
| [Visione](docs/vision.md) | [Architettura](docs/architecture/overview.md) |
| [Avvio rapido (30 min, senza RF)](docs/tutorials/quickstart-no-rf.md) | [Pilota bar (2 ore, con checklist legale)](docs/tutorials/cafe-pilot.md) |
| [Spettro e legge](docs/spectrum-and-law/index.md) | [Modello di minacce](design/threat-model.md) |
| [Ciclo di vita SIM](docs/sim-lifecycle/index.md) | [Tessuto di peering](docs/peering/index.md) |
| [Riferimento API](docs/api/index.md) | [ADR](docs/adr/0000-index.md) |

## Stato

**Versione di laboratorio `v0.1.0`**: l'attach EPC + zmq RAN + srsUE funziona end-to-end; piano di controllo,
CLI, provisioner SIM, eSIM di laboratorio (SM-DP+) e il sito di documentazione sono funzionali. I percorsi
RF reali sono validati su hardware di sviluppo ma **disabilitati per impostazione predefinita**.
Vedi la [roadmap](design/roadmap.md).

## Cosa Fairwave NON è

- **Non è un IMSI catcher** - nessuna interrogazione passiva; le UE devono autenticarsi con credenziali da te provisionate.
- **Non è un Far West spettrale** - la TX è cancellata, e rifiutiamo funzioni di bypass normativo.
- **Non è un operatore nazionale gratuito** - è copertura locale con breakout opzionale.
- **Non è un sostituto delle chiamate di emergenza** - pianifica il comportamento 911/112 in ogni deployment ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Contribuire e governance

Diamo il benvenuto ai contributori. Leggi [CONTRIBUTING.md](CONTRIBUTING.md) (stile del codice, DCO, test),
[GOVERNANCE.md](GOVERNANCE.md) (come si prendono le decisioni), [SECURITY.md](SECURITY.md)
(divulgazione delle vulnerabilità; modello di minacce) e [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Licenza: **Apache-2.0** ([LICENSE](LICENSE)); attribuzioni di terze parti in [NOTICE](NOTICE).

---

<p align="center">
  <sub>Costruito dal team HyperonX e dalla comunità Fairwave. L'aria è di tutti.</sub>
</p>
