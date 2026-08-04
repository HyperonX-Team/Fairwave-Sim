<h1 align="center">Fairwave - bir pizza kutusundaki topluluk operatörü</h1>

<p align="center">
  <strong>Açık kaynak özel LTE: Ethernet'e tak, 4G yay, komşularını karşıla.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **YASAL VE SPEKTRUM UYARISI.** Fairwave varsayılan olarak **laboratuvar / RF'siz** modda çalışır (yalnızca sıfır-IF geri döngü).
> Uygun yetkilendirme olmadan hücresel bantlarda yayın yapmak **çoğu ülkede yasadışıdır**.
> Lisanslar, SAS izinleri, iç mekan kısıtlamaları ve tip onayının tüm sorumluluğu size aittir.
> HyperonX ve katkıda bulunanlar yazılımı yalnızca yasal özel ağlar, araştırma ve paylaşımlı spektrum
> rejimleri için **olduğu gibi** sunar. Bkz. [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Şu dillerde okuyun:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## Bozuk sistem

Mobil bağlantı tek kişilik bir karteldir: bir SIM, bir operatör, bir sözleşme, duvarlarla çevrili bir bahçe.
Kapsama haritaları pazarlama broşürleridir; kırsal sokaklar onlardan düşer; apartmanlar sinyal kaybeder;
ve siz - ödeyen kişi - size hizmet eden altyapının hiçbirine sahip değilsiniz. Ulusal operatörünüz fiyat
artırırsa ya da kulesi susarsa, tek seçeneğiniz… aynı kuleler ve aynı koşullarla farklı bir tekeldir.

Cebinizdeki modem 20 metre uzaktaki bir baz istasyonuyla konuşabilir. *Sizin* baz istasyonunuzla konuşmamasının
tek nedeni, o istasyonun size ait olmasına hiçbir zaman izin verilmemiş olmasıdır.

## HyperonX çözümü

**Fairwave topluluk operatörüdür: bir pizza kutusuna sığan ve sıradan bir Ethernet'e takılan eksiksiz,
açık kaynak bir küçük hücre.**

Bir kafe, bir konut kooperatifi, bir köy salonu, bir mahalle - herkes bir tane işletebilir:

1. Bağlı bir SDR ile bir mini PC'de (x86 veya ARM) Fairwave imajını başlatın.
2. `fairwave node init` çalıştırın, düzenleyici kontrol listesine yanıt verin (yetkiyi kanıtlayana kadar TX kapalı kalır).
3. Topluluğunuz için Fairwave SIM'leri çıkarın - sahip olduğunuz kartlar, kontrol ettiğiniz kimlik bilgileri.
4. Telefonlar bağlanır. Trafik mümkün olduğunda yerel kalır; birden fazla pizza kutusu bir mesh oluşturur;
   internet erişimi istediğinizde güvenli bir WireGuard tüneliyle gider.

Kanıtlanmış açık altyapı üzerine kuruludur - **Open5GS** (EPC) ve **srsRAN** (eNB/gNB) - Go kontrol düzlemi,
operatör paneli, çevrimdışı SIM sağlama ve tüm operatörü sıfır RF ile Docker'da çalıştıran laboratuvar moduyla.

## Neler elde edersiniz

- **Operatör olan bir pizza kutusu**: `fairwave-control` tarafından yönetilen düğüm kimliği, kayıt, yaşam döngüsü
  (`provision → register → on-air → peer → breakout`).
- **Gerçek LTE bağlantısı**: Open5GS EPC + srsRAN eNB, yapılandırılabilir PLMN, takip alanları,
  `internet` + `ims` APN'leri, uçta yerel çıkış.
- **Fairwave SIM operasyonları**: çevrimdışı öncelikli sağlayıcı; Ki/OPc üretir, kart büroları için CSV/JSON
  partileri, HSS/UDM yazar, iptal ve değiştirme kontrolleri. Laboratuvar ve üretim kesin olarak ayrılmıştır.
- **Laboratuvar eSIM (SM-DP+)**: kendi SGP.22 biçimli profil sunucunuz ve bir yazılım eUICC -
  şifrelenmiş bağlı profil paketleri, QR aktivasyon kodları (`LPA:1$...`), donanım olmadan CI'da doğrulanan
  tam indirme döngüsü. Tasarım gereği yalnızca laboratuvar; GSMA uygunluğu açık maddeler olarak izlenir.
- **Komşuluk ağı**: mDNS keşfi, mTLS kontrolü, WireGuard veri düzlemi, rota değişimi.
- **Operatör portalı**: yerel öncelikli panel - UE'ler (gizlilik korumalı), geri bağlantı, eşler, laboratuvar modu.
- **Tam laboratuvar modu**: srsUE ile tüm ağ saf IP (zmq) üzerinde - radyo yok, lisans yok, CI dostu.
- **Kod içinde düzenleyici kapılar**: TX'i kurmak ülke kodu, lisans onayı ve frekans beyaz listesi gerektirir.
  Aksi halde derleme zamanı reddi.

## Neden mevcut operatörleri tehdit ediyor

Operatör kilidi *yerel kapsama için isteğe bağlı* hale gelir. Kafe, kooperatif ve köy her biri bir hücre
işletip bunları mesh ile birleştirebildiğinde, ulusal SIM sadece bir dolaşım seçeneğidir, bekçi değil.
Yük boşaltma, tarafsız ev sahibi erişimi ve topluluk fiyatlı planlar teorik olmaktan çıkar. Fairwave telekom
kartelini kopyalamaz - kartelin yerel tekeline rekabet edilebilir hale getirir.

## Neden şimdi mümkün

- **Open5GS** ve **srsRAN** olgun, üretime yakın ve aktif olarak geliştiriliyor.
- **SDR'ler** (USRP, LimeSDR, BladeRF) yüzlerce dolara küçük hücre gücünde LTE bantlarını kapsar.
- **Paylaşımlı spektrum gerçektir**: ABD'de CBRS, Birleşik Krallık/AB'de yerel lisanslar, özel LTE bantları.
- **Özel LTE + Wi-Fi aramaları**, düzenleme yokmuş gibi davranmak yerine kopyalanacak yasal şablonlar sunar.
- Mini PC sınıfı donanım (ve geliştirme için Raspberry Pi CM4/5 + HAT'lar) ucuz ve yeterlidir.

---

## Hızlı başlangıç - 30 dakikadan kısa sürede ilk UE bağlantısı (RF'siz, lisanssız)

> Gereksinimler: Docker Engine 24+, 8 GB RAM. Her şey konteynerlerde çalışır; hiçbir şey yayın yapmaz.
> Tam veri yolu (UE IP + ping) için yerel Linux kullanın; Windows/macOS'ta Docker Desktop
> EPC tarafındaki tüm bağlantı kontrollerini geçer (bkz. [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # araç zincirlerini kontrol eder/kurar (Go, Docker, pre-commit)
make lab-up                 # imajları kurar, EPC + eNB + srsUE'yi ayağa kaldırır, bağlantı kontrollerini çalıştırır
make status                 # tek bakışta sağlık: mme, sgwu/upf, enb, ue1
```

`make lab-up` aşağıdakilerin tümünü doğrular (ve yazdırır):

1. Open5GS MME + HSS çalışıyor
2. eNB S1-MME MME'ye bağlı
3. Laboratuvar PLMN'sinde UE RRC bağlantısı + rastgele erişim
4. UE NAS kimlik doğrulaması + güvenlik modu (HSS'e karşı milenage)
5. MME varsayılan EPS taşıyıcısını oluşturur ve Attach Accept gönderir (UE IP atanır)

Sonra içine bakın:

```bash
fairwave node status                     # bu kümenin kontrol düzlemi görünümü
fairwave sim issue --count 2 --profile lab  # iki laboratuvar SIM'i çıkar
fairwave spectrum check --country US --band n48 --indoor  # spektrum kapısı demosu
make lab-down                            # her şeyi durdur
```

(ZMQ zamanlaması kararlı Linux ana bilgisayarlarında `docker exec -it ue1 ping -c3 10.45.0.1`,
`tun_srsue` tüneli üzerinden tam veri yolunu test eder.)

Yukarıdakilerin tümü **RF sessizdir**: radyo `srsran/zmq` sanal RF aygıtlarıyla öykünülür.
Gerçek donanım için [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) izleyin -
yasal kontrol listesini tamamlamadan TX'i etkinleştirmenize izin vermez.

## Dokümantasyon

Tam site: **`make docs-serve` çalıştırın** - veya ağaç içinde [`docs/`](docs/index.md) altında okuyun.

| Buradan başlayın | Sonra |
|---|---|
| [Vizyon](docs/vision.md) | [Mimari](docs/architecture/overview.md) |
| [Hızlı başlangıç (30 dk, RF'siz)](docs/tutorials/quickstart-no-rf.md) | [Kafe pilotu (2 sa, yasal kontrol listeli)](docs/tutorials/cafe-pilot.md) |
| [Spektrum ve hukuk](docs/spectrum-and-law/index.md) | [Tehdit modeli](design/threat-model.md) |
| [SIM yaşam döngüsü](docs/sim-lifecycle/index.md) | [Eşleştirme dokusu](docs/peering/index.md) |
| [API referansı](docs/api/index.md) | [ADR'ler](docs/adr/0000-index.md) |

## Durum

**Laboratuvar sürümü `v0.1.0`**: EPC + zmq RAN + srsUE bağlantısı uçtan uca çalışır; kontrol düzlemi, CLI,
SIM sağlayıcı, laboratuvar eSIM (SM-DP+) ve dokümantasyon sitesi işlevseldir. Gerçek RF yolları geliştirme
donanımında doğrulanmıştır ancak **varsayılan olarak devre dışıdır**. [Yol haritasına](design/roadmap.md) bakın.

## Fairwave NE DEĞİLDİR

- **Bir IMSI avcısı değildir** - pasif sorgulama yoktur; UE'ler sizin sağladığınız kimlik bilgileriyle doğrulanmalıdır.
- **Spektrumda başıboşluk değildir** - TX kapılıdır ve düzenleyici atlatma özelliklerini reddederiz.
- **Ücretsiz bir ulusal operatör değildir** - isteğe bağlı çıkışlı yerel kapsamadır.
- **Acil aramaların yerine geçmez** - her dağıtımda 911/112 davranışını planlayın ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Katkı ve yönetişim

Katkıda bulunanları memnuniyetle karşılıyoruz. [CONTRIBUTING.md](CONTRIBUTING.md) (kod stili, DCO, testler),
[GOVERNANCE.md](GOVERNANCE.md) (kararlar nasıl alınır), [SECURITY.md](SECURITY.md)
(güvenlik açığı bildirimi; tehdit modeli) ve [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) okuyun.
Lisans: **Apache-2.0** ([LICENSE](LICENSE)); üçüncü taraf atıfları [NOTICE](NOTICE).

---

<p align="center">
  <sub>HyperonX ekibi ve Fairwave topluluğu tarafından yapıldı. Hava herkesindir.</sub>
</p>
