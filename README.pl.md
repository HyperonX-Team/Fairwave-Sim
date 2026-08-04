<h1 align="center">Fairwave - operator społecznościowy w pudełku po pizzy</h1>

<p align="center">
  <strong>Otwarta prywatna sieć LTE: podłącz do Ethernetu, emituj 4G, witaj sąsiadów.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **OSTRZEŻENIE PRAWNE I DOTYCZĄCE WIDMA.** Fairwave domyślnie działa w trybie **laboratoryjnym / bez RF** (tylko pętla zerowego IF).
> Nadawanie na pasmach komórkowych bez odpowiedniego zezwolenia jest **nielegalne w większości jurysdykcji**.
> Jesteś wyłącznie odpowiedzialny za licencje, zezwolenia SAS, ograniczenia wewnątrz pomieszczeń i homologację typu.
> HyperonX i współtwórcy udostępniają oprogramowanie **tak jak jest** wyłącznie dla legalnych sieci prywatnych, badań
> i reżimów współdzielonego widma. Zobacz [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Czytaj w:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## Zepsuty system

Łączność mobilna to kartel jednego: jedna karta SIM, jeden operator, jedna umowa, jeden zamknięty ogród.
Mapy zasięgu to broszury marketingowe; wiejskie ulice z nich wypadają; mieszkania tracą sygnał;
a ty - osoba płacąca - nie posiadasz żadnej z infrastruktur, które cię obsługują. Jeśli twój krajowy operator
podniesie ceny lub jego maszt zamilknie, twoją jedyną opcją jest… inny monopol z tymi samymi masztami i tymi samymi warunkami.

Modem w twojej kieszeni może rozmawiać ze stacją bazową 20 metrów dalej. Jedynym powodem, dla którego nie rozmawia
z *twoją* stacją bazową, jest to, że tej stacji nigdy nie pozwolono być twoją.

## Rozwiązanie HyperonX

**Fairwave to operator społecznościowy: kompletna, otwartoźródłowa mała stacja komórkowa, która mieści się
w pudełku po pizzy i podłącza do zwykłego Ethernetu.**

Kawiarnia, spółdzielnia mieszkaniowa, wiejska świetlica, osiedle - każdy może prowadzić jedną:

1. Uruchom obraz Fairwave na mini-PC (x86 lub ARM) z podłączonym SDR.
2. Wykonaj `fairwave node init`, odpowiedz na listę kontrolną regulacyjną (TX pozostaje wyłączony, dopóki nie udowodnisz zezwolenia).
3. Wydawaj karty SIM Fairwave swojej społeczności - karty, które posiadasz, poświadczenia, które kontrolujesz.
4. Telefony się łączą. Ruch pozostaje lokalny, gdy to możliwe; kilka pudełek po pizzy tworzy siatkę;
   dostęp do internetu idzie bezpiecznym tunelem WireGuard, gdy chcesz.

Zbudowane na sprawdzonej otwartej infrastrukturze - **Open5GS** (EPC) i **srsRAN** (eNB/gNB) - z płaszczyzną
kontrolną w Go, panelem operatora, offline'owym dostarczaniem kart SIM i trybem laboratoryjnym, który uruchamia
całego operatora w Dockerze z zerowym RF.

## Co otrzymujesz

- **Pudełko po pizzy, które jest operatorem**: tożsamość węzła, rejestracja, cykl życia
  (`provision → register → on-air → peer → breakout`) zarządzany przez `fairwave-control`.
- **Prawdziwe dołączenie LTE**: Open5GS EPC + srsRAN eNB, konfigurowalny PLMN, obszary śledzenia,
  APN `internet` + `ims`, lokalny breakout na krawędzi.
- **Operacje SIM Fairwave**: dostarczanie offline-first; generuje Ki/OPc, partie CSV/JSON dla
  biur kart, zapis do HSS/UDM, kontrola unieważniania i wymiany. Laboratorium i produkcja ściśle rozdzielone.
- **Laboratoryjna eSIM (SM-DP+)**: własny serwer profili w kształcie SGP.22 i programowa eUICC -
  szyfrowane pakiety powiązanych profili, kody aktywacyjne QR (`LPA:1$...`), pełny cykl pobierania
  zweryfikowany w CI bez sprzętu. Tylko laboratorium z założenia; zgodność GSMA śledzona jako otwarte pozycje.
- **Sąsiedzka siatka**: wykrywanie mDNS, kontrola mTLS, płaszczyzna danych WireGuard, wymiana tras.
- **Portal operatora**: lokalny pulpit - UE (z poszanowaniem prywatności), backhaul, peerzy, tryb laboratorium.
- **Pełny tryb laboratoryjny**: cała sieć na czystym IP (zmq) z srsUE - bez radia, bez licencji, przyjazny CI.
- **Bramy regulacyjne w kodzie**: uzbrojenie TX wymaga kodu kraju, potwierdzenia licencji
  i białej listy częstotliwości. W przeciwnym razie kompilacyjna odmowa.

## Dlaczego zagraża obecnym operatorom

Zamknięcie operatora staje się *opcjonalne dla lokalnego zasięgu*. Gdy kawiarnia, spółdzielnia i wieś mogą
prowadzić każda własną komórkę i połączyć je w siatkę, krajowa karta SIM jest tylko opcją roamingu, a nie
odźwiernym. Offload, dostęp neutralnego hosta i plany w cenach społecznościowych przestają być teorią.
Fairwave nie kopiuje kartelu telekomunikacyjnego - czyni lokalny monopol kartelu kwestionowalnym.

## Dlaczego to możliwe teraz

- **Open5GS** i **srsRAN** są dojrzałe, bliskie produkcji i aktywnie rozwijane.
- **SDR-y** (USRP, LimeSDR, BladeRF) pokrywają pasma LTE z mocą małej komórki za kilkaset dolarów.
- **Współdzielone widmo jest realne**: CBRS w USA, lokalne licencje w Wielkiej Brytanii/UE, prywatne pasma LTE.
- **Prywatne LTE + połączenia Wi-Fi** dają legalne szablony do skopiowania zamiast udawania, że regulacje nie istnieją.
- Sprzęt klasy mini-PC (oraz Raspberry Pi CM4/5 + HAT do rozwoju) jest tani i wystarczający.

---

## Szybki start - pierwsze dołączenie UE w <30 minut (bez RF, bez licencji)

> Wymagania: Docker Engine 24+, 8 GB RAM. Wszystko działa w kontenerach; nic nie nadaje.
> Do pełnej ścieżki danych (IP UE + ping) użyj natywnego Linuksa; Docker Desktop na
> Windows/macOS przechodzi wszystkie kontrole dołączenia po stronie EPC (zobacz [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # sprawdza/instaluje toolchainy (Go, Docker, pre-commit)
make lab-up                 # buduje obrazy, uruchamia EPC + eNB + srsUE, wykonuje kontrole dołączenia
make status                 # zdrowie na pierwszy rzut oka: mme, sgwu/upf, enb, ue1
```

`make lab-up` potwierdza (i wypisuje) wszystko poniżej:

1. Open5GS MME + HSS działają
2. eNB S1-MME połączony z MME
3. Połączenie RRC UE + dostęp losowy w laboratoryjnym PLMN
4. Uwierzytelnianie NAS UE + tryb bezpieczeństwa (milenage względem HSS)
5. MME tworzy domyślny nośnik EPS i wysyła Attach Accept (IP UE przydzielony)

Potem zajrzyj do środka:

```bash
fairwave node status                     # widok płaszczyzny kontrolnej tego klastra
fairwave sim issue --count 2 --profile lab  # wydaj dwie laboratoryjne karty SIM
fairwave spectrum check --country US --band n48 --indoor  # demo bramy widma
make lab-down                            # zatrzymaj wszystko
```

(Na hostach Linux ze stabilnym taktowaniem ZMQ `docker exec -it ue1 ping -c3 10.45.0.1`
ćwiczy pełną ścieżkę danych przez tunel `tun_srsue`.)

Wszystko powyżej jest **ciche radiowo**: radio jest emulowane wirtualnymi urządzeniami `srsran/zmq`.
Dla prawdziwego sprzętu postępuj zgodnie z [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
które nie pozwoli włączyć TX bez ukończenia listy kontrolnej prawnej.

## Dokumentacja

Pełna witryna: **uruchom `make docs-serve`** - lub czytaj w drzewie pod [`docs/`](docs/index.md).

| Zacznij tutaj | Potem |
|---|---|
| [Wizja](docs/vision.md) | [Architektura](docs/architecture/overview.md) |
| [Szybki start (30 min, bez RF)](docs/tutorials/quickstart-no-rf.md) | [Pilot w kawiarni (2 h, z listą kontrolną prawną)](docs/tutorials/cafe-pilot.md) |
| [Widmo i prawo](docs/spectrum-and-law/index.md) | [Model zagrożeń](design/threat-model.md) |
| [Cykl życia karty SIM](docs/sim-lifecycle/index.md) | [Tkanina peeringu](docs/peering/index.md) |
| [Referencja API](docs/api/index.md) | [ADR-y](docs/adr/0000-index.md) |

## Status

**Wersja laboratoryjna `v0.1.0`**: dołączenie EPC + zmq RAN + srsUE działa end-to-end; płaszczyzna kontrolna,
CLI, dostarczanie SIM, laboratoryjna eSIM (SM-DP+) i witryna dokumentacji są funkcjonalne. Prawdziwe ścieżki
RF są zweryfikowane na sprzęcie deweloperskim, ale **domyślnie wyłączone**. Zobacz [mapę drogową](design/roadmap.md).

## Czym Fairwave NIE jest

- **Nie jest łapaczem IMSI** - brak biernego przesłuchiwania; UE muszą uwierzytelniać się poświadczeniami, które dostarczyłeś.
- **Nie jest Dzikim Zachodem widma** - TX jest bramkowany, a my odrzucamy funkcje obchodzenia regulacji.
- **Nie jest darmowym krajowym operatorem** - to lokalny zasięg z opcjonalnym breakout.
- **Nie zastępuje połączeń alarmowych** - zaplanuj zachowanie 911/112 w każdym wdrożeniu ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Współpraca i zarządzanie

Zapraszamy współtwórców. Przeczytaj [CONTRIBUTING.md](CONTRIBUTING.md) (styl kodu, DCO, testy),
[GOVERNANCE.md](GOVERNANCE.md) (jak podejmowane są decyzje), [SECURITY.md](SECURITY.md)
(ujawnianie podatności; model zagrożeń) i [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Licencja: **Apache-2.0** ([LICENSE](LICENSE)); atrybucje stron trzecich w [NOTICE](NOTICE).

---

<p align="center">
  <sub>Zbudowane przez zespół HyperonX i społeczność Fairwave. Powietrze należy do wszystkich.</sub>
</p>
