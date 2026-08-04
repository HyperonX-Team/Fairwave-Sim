<h1 align="center">Fairwave - operator komunitas dalam kotak pizza</h1>

<p align="center">
  <strong>LTE privat sumber terbuka: colokkan ke Ethernet, pancarkan 4G, sambut tetangga Anda.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **PERINGATAN HUKUM DAN SPEKTRUM.** Fairwave berjalan secara default dalam mode **laboratorium / tanpa RF** (hanya loopback IF nol).
> Mentransmisikan pada pita seluler tanpa otorisasi yang sah **ilegal di sebagian besar yurisdiksi**.
> Anda sepenuhnya bertanggung jawab atas lisensi, hibah SAS, pembatasan dalam ruangan, dan persetujuan tipe.
> HyperonX dan kontributor menyediakan perangkat lunak **apa adanya** hanya untuk jaringan privat yang sah, riset,
> dan rezim spektrum bersama. Lihat [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Baca dalam:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Tiếng Việt](README.vi.md)

---

## Sistem yang rusak

Konektivitas seluler adalah kartel satu orang: satu SIM, satu operator, satu kontrak, satu taman berdinding.
Peta cakupan hanyalah brosur pemasaran; jalan-jalan pedesaan jatuh darinya; apartemen kehilangan sinyal;
dan Anda - orang yang membayar - tidak memiliki satu pun infrastruktur yang melayani Anda. Jika operator
nasional Anda menaikkan harga atau menaranya diam, satu-satunya pilihan Anda adalah… monopoli lain dengan
menara yang sama dan ketentuan yang sama.

Modem di saku Anda dapat berbicara dengan stasiun pangkalan 20 meter jauhnya. Satu-satunya alasan ia tidak
berbicara dengan stasiun pangkalan *Anda* adalah bahwa stasiun itu tidak pernah diizinkan menjadi milik Anda.

## Solusi HyperonX

**Fairwave adalah operator komunitas: sel kecil lengkap sumber terbuka yang muat dalam kotak pizza
dan dicolokkan ke Ethernet biasa.**

Sebuah kafe, koperasi perumahan, balai desa, sebuah lingkungan - siapa pun dapat menjalankan satu:

1. Nyalakan citra Fairwave di mini-PC (x86 atau ARM) dengan SDR terhubung.
2. Jalankan `fairwave node init`, jawab daftar periksa regulasi (TX tetap mati sampai Anda membuktikan otorisasi).
3. Terbitkan SIM Fairwave untuk komunitas Anda - kartu yang Anda miliki, kredensial yang Anda kendalikan.
4. Ponsel terhubung. Lalu lintas tetap lokal bila memungkinkan; beberapa kotak pizza membentuk mesh;
   akses internet melalui terowongan WireGuard yang aman saat Anda menginginkannya.

Dibangun di atas infrastruktur terbuka yang terbukti - **Open5GS** (EPC) dan **srsRAN** (eNB/gNB) - dengan
bidang kendali Go, dasbor operator, penyediaan SIM offline, dan mode laboratorium yang menjalankan seluruh
operator dalam Docker dengan nol RF.

## Apa yang Anda dapatkan

- **Kotak pizza yang menjadi operator**: identitas node, pendaftaran, siklus hidup
  (`provision → register → on-air → peer → breakout`) dikelola oleh `fairwave-control`.
- **Lampiran LTE nyata**: Open5GS EPC + srsRAN eNB, PLMN yang dapat dikonfigurasi, area pelacakan,
  APN `internet` + `ims`, breakout lokal di tepi.
- **Operasi SIM Fairwave**: penyedia offline-first; menghasilkan Ki/OPc, batch CSV/JSON untuk
  biro kartu, menulis ke HSS/UDM, kontrol pencabutan dan penggantian. Lab dan produksi dipisahkan secara ketat.
- **eSIM laboratorium (SM-DP+)**: server profil berbentuk SGP.22 Anda sendiri dan eUICC perangkat lunak -
  paket profil terikat terenkripsi, kode aktivasi QR (`LPA:1$...`), siklus unduh lengkap terverifikasi
  CI tanpa perangkat keras. Hanya lab secara desain; kepatuhan GSMA dilacak sebagai item terbuka.
- **Mesh lingkungan**: penemuan mDNS, kontrol mTLS, bidang data WireGuard, pertukaran rute.
- **Portal operator**: dasbor lokal-first - UE (menjaga privasi), backhaul, peer, mode laboratorium.
- **Mode laboratorium lengkap**: seluruh jaringan di atas IP murni (zmq) dengan srsUE - tanpa radio, tanpa lisensi, ramah CI.
- **Gerbang regulasi dalam kode**: mengaktifkan TX memerlukan kode negara, pengakuan lisensi,
  dan daftar putih frekuensi. Jika tidak, penolakan saat kompilasi.

## Mengapa mengancam operator yang ada

Penguncian operator menjadi *opsional untuk cakupan lokal*. Ketika kafe, koperasi, dan desa masing-masing
dapat menjalankan sel dan menghubungkannya dalam mesh, SIM nasional hanyalah opsi roaming, bukan penjaga gerbang.
Offload, akses host netral, dan paket harga komunitas tidak lagi teoretis. Fairwave tidak meniru kartel telekomunikasi -
ia membuat monopoli lokal kartel dapat diperebutkan.

## Mengapa layak sekarang

- **Open5GS** dan **srsRAN** matang, mendekati produksi, dan dikembangkan secara aktif.
- **SDR** (USRP, LimeSDR, BladeRF) mencakup pita LTE dengan daya sel kecil untuk ratusan dolar.
- **Spektrum bersama itu nyata**: CBRS di AS, lisensi lokal di Inggris/UE, pita LTE privat.
- **LTE privat + panggilan Wi-Fi** memberikan templat legal untuk ditiru alih-alih berpura-pura regulasi tidak ada.
- Perangkat keras kelas mini-PC (dan Raspberry Pi CM4/5 + HAT untuk pengembangan) murah dan memadai.

---

## Mulai cepat - lampiran UE pertama dalam <30 menit (tanpa RF, tanpa lisensi)

> Persyaratan: Docker Engine 24+, RAM 8 GB. Semuanya berjalan dalam kontainer; tidak ada yang memancarkan.
> Untuk jalur data lengkap (IP UE + ping) gunakan Linux asli; Docker Desktop di
> Windows/macOS melewati semua pemeriksaan lampiran sisi EPC (lihat [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # memeriksa/memasang toolchain (Go, Docker, pre-commit)
make lab-up                 # membangun citra, menjalankan EPC + eNB + srsUE, menjalankan pemeriksaan lampiran
make status                 # kesehatan sekilas: mme, sgwu/upf, enb, ue1
```

`make lab-up` memverifikasi (dan mencetak) semua hal berikut:

1. Open5GS MME + HSS berjalan
2. eNB S1-MME terhubung ke MME
3. Koneksi RRC UE + akses acak di PLMN laboratorium
4. Autentikasi NAS UE + mode keamanan (milenage terhadap HSS)
5. MME membuat pembawa EPS default dan mengirim Attach Accept (IP UE dialokasikan)

Lalu lihat ke dalam:

```bash
fairwave node status                     # tampilan bidang kendali kluster ini
fairwave sim issue --count 2 --profile lab  # terbitkan dua SIM lab
fairwave spectrum check --country US --band n48 --indoor  # demo gerbang spektrum
make lab-down                            # hentikan semuanya
```

(Pada host Linux dengan timing ZMQ stabil, `docker exec -it ue1 ping -c3 10.45.0.1`
menguji jalur data lengkap melalui terowongan `tun_srsue`.)

Semua di atas **senyap RF**: radio disimulasikan dengan perangkat RF virtual `srsran/zmq`.
Untuk perangkat keras asli, ikuti [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
yang tidak akan mengizinkan Anda mengaktifkan TX tanpa menyelesaikan daftar periksa hukum.

## Dokumentasi

Situs lengkap: **jalankan `make docs-serve`** - atau baca di pohon di bawah [`docs/`](docs/index.md).

| Mulai di sini | Lalu |
|---|---|
| [Visi](docs/vision.md) | [Arsitektur](docs/architecture/overview.md) |
| [Mulai cepat (30 mnt, tanpa RF)](docs/tutorials/quickstart-no-rf.md) | [Pilot kafe (2 jam, dengan daftar periksa hukum)](docs/tutorials/cafe-pilot.md) |
| [Spektrum dan hukum](docs/spectrum-and-law/index.md) | [Model ancaman](design/threat-model.md) |
| [Siklus hidup SIM](docs/sim-lifecycle/index.md) | [Kain peering](docs/peering/index.md) |
| [Referensi API](docs/api/index.md) | [ADR](docs/adr/0000-index.md) |

## Status

**Rilis lab `v0.1.0`**: lampiran EPC + zmq RAN + srsUE berfungsi end-to-end; bidang kendali, CLI,
penyedia SIM, eSIM lab (SM-DP+) dan situs dokumentasi berfungsi. Jalur RF nyata tervalidasi pada perangkat
keras pengembangan tetapi **dinonaktifkan secara default**. Lihat [peta jalan](design/roadmap.md).

## Apa yang Fairwave BUKAN

- **Bukan penangkap IMSI** - tidak ada interogasi pasif; UE harus mengautentikasi dengan kredensial yang Anda sediakan.
- **Bukan tanah liar spektrum** - TX digerbangi, dan kami menolak fitur penghindaran regulasi.
- **Bukan operator nasional gratis** - ini cakupan lokal dengan breakout opsional.
- **Bukan pengganti panggilan darurat** - rencanakan perilaku 911/112 di setiap penempatan ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Berkontribusi dan tata kelola

Kami menyambut kontributor. Baca [CONTRIBUTING.md](CONTRIBUTING.md) (gaya kode, DCO, pengujian),
[GOVERNANCE.md](GOVERNANCE.md) (cara keputusan dibuat), [SECURITY.md](SECURITY.md)
(pengungkapan kerentanan; model ancaman), dan [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Lisensi: **Apache-2.0** ([LICENSE](LICENSE)); atribusi pihak ketiga di [NOTICE](NOTICE).

---

<p align="center">
  <sub>Dibangun oleh tim HyperonX dan komunitas Fairwave. Udara untuk semua orang.</sub>
</p>
