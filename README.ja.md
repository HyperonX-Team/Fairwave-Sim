<h1 align="center">Fairwave - ピザボックスに入ったコミュニティ・キャリア</h1>

<p align="center">
  <strong>オープンソースのプライベートLTE：イーサネットに挿して、4G/5Gを出し、隣人を迎え入れよう。</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **法的・周波数に関する警告。** Fairwaveはデフォルトで**ラボ／無RFモード**（ゼロIFループバックのみ）で動作します。
> 適切な許可なしに携帯電話バンドで送信することは、**ほとんどの法域で違法です**。
> ライセンス、SAS許諾、屋内制限、型式認証の責任はすべてあなたにあります。
> HyperonXとコントリビューターは、合法的なプライベートネットワーク、研究、共有スペクトラム制度の
> ためだけに、ソフトウェアを**現状のまま**提供します。[docs/spectrum-and-law/](docs/spectrum-and-law/index.md) を参照。

**他の言語:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## 壊れたシステム

モバイル接続は一人だけのカルテルです：SIM1枚、キャリア1社、契約1つ、閉ざされた庭。
カバレッジマップはマーケティングのパンフレット；地方の通りはそこから落ち、アパートは信号を失い、
そしてあなた（お金を払う人）は、自分を支えるインフラのどれも所有していません。国内キャリアが値上げ
したら、あるいはその塔が沈黙したら、あなたの唯一の選択肢は…同じ塔と同じ条件の別の独占企業です。

あなたのポケットのモデムは、20メートル先の基地局と話せます。それが*あなたの*基地局と話さない唯一の
理由は、その基地局があなたのものになることを決して許されなかったからです。

## HyperonXの解決策

**Fairwaveはコミュニティ・キャリアです：ピザボックスに収まり、普通のイーサネットに挿すだけの、
完全なオープンソース・スモールセル。**

カフェ、住宅協同組合、公民館、町内会 - 誰でも運用できます：

1. SDRを接続したミニPC（x86またはARM）でFairwaveイメージを起動。
2. `fairwave node init` を実行し、規制チェックリストに回答（許可を証明するまで送信はオフ）。
3. コミュニティにFairwave SIMを発行 - あなたが所有するカード、あなたが管理する認証情報。
4. 電話が接続。トラフィックは可能な限りローカルに留まり、複数のピザボックスがメッシュを形成。
   必要なときは安全なWireGuardトンネルでインターネットに抜けます。

実績あるオープン基盤 - **Open5GS**（EPC）と**srsRAN**（eNB/gNB）- の上に、Goのコントロールプレーン、
オペレーターダッシュボード、オフラインSIM発行、ゼロRFでキャリア全体をDockerで動かすラボモードを搭載。

## 得られるもの

- **キャリアそのものになるピザボックス**：`fairwave-control` が管理するノードID、登録、ライフサイクル
  （`provision → register → on-air → peer → breakout`）。
- **本物のLTEアタッチ**：Open5GS EPC + srsRAN eNB、設定可能なPLMN、トラッキングエリア、
  `internet` + `ims` APN、エッジでのローカルブレークアウト。
- **Fairwave SIM運用**：オフラインファーストの発行者；Ki/OPcを生成し、カードビューロー向けにCSV/JSONバッチ、
  HSS/UDMへの書き込み、失効・交換管理。ラボと本番は厳格に分離。
- **ラボeSIM（SM-DP+）**：自前のSGP.22型プロファイルサーバーとソフトウェアeUICC -
  暗号化されたバウンドプロファイルパッケージ、QRアクティベーションコード（`LPA:1$...`）、
  ハードウェアなしでCI検証される完全なダウンロードサイクル。設計上ラボのみ。GSMA適合性は未解決項目として追跡。
- **近隣メッシュ**：mDNSディスカバリー、mTLS制御、WireGuardデータプレーン、ルート交換。
- **オペレーターポータル**：ローカルファーストのダッシュボード - UE（プライバシー保護）、バックホール、ピア、ラボモード。
- **完全なラボモード**：ネットワーク全体を純粋なIP（zmq）+ srsUEで実行 - 無線なし、ライセンス不要、CIフレンドリー。
- **コード内の規制ゲート**：送信の武装化には国コード、ライセンスの確認、周波数ホワイトリストが必要。
  それ以外はコンパイル時に拒否。

## なぜ既存キャリアへの脅威なのか

ローカルカバレッジにおいて、キャリアの囲い込みは*任意*になります。カフェも協同組合も村も、それぞれ
セルを運用してメッシュでつなげるなら、国内SIMは単なるローミングオプションであり、門番ではありません。
オフロード、中立ホストアクセス、コミュニティ価格のプランは理論でなくなります。Fairwaveは通信カルテルを
再現しません - カルテルのローカル独占に挑戦可能にします。

## なぜ今なら可能か

- **Open5GS**と**srsRAN**は成熟し、本番に近く、活発に開発されています。
- **SDR**（USRP、LimeSDR、BladeRF）は数百ドルでスモールセル出力のLTEバンドをカバー。
- **共有スペクトラムは現実**：米国のCBRS、英国／EUのローカルライセンス、プライベートLTEバンド。
- **プライベートLTE + Wi-Fiコーリング**は、規制がないふりをする代わりにコピーできる合法的なテンプレートを提供。
- ミニPCクラスのハードウェア（開発用のRaspberry Pi CM4/5 + HATも）は安価で十分。

---

## クイックスタート - 30分以内に最初のUEアタッチ（無線なし、ライセンス不要）

> 要件：Docker Engine 24+、8GB RAM。すべてコンテナで動作し、何も送信しません。
> 完全なデータパス（UE IP + ping）にはネイティブLinuxを使用；Windows/macOSのDocker Desktopは
> EPC側のアタッチチェックをすべて通過します（[docs](docs/tutorials/lab-attach.md)参照）。

```bash
./scripts/bootstrap.sh      # ツールチェーンの確認/インストール（Go、Docker、pre-commit）
make lab-up                 # イメージ構築、EPC + eNB + srsUE起動、アタッチチェック実行
make status                 # ひと目でわかる健全性：mme、sgwu/upf、enb、ue1
```

`make lab-up` は以下をすべて検証（および表示）します：

1. Open5GS MME + HSSが稼働
2. eNB S1-MMEがMMEに接続
3. ラボPLMNでのUE RRC接続 + ランダムアクセス
4. UE NAS認証 + セキュリティモード（HSSとのmilenage）
5. MMEがデフォルトEPSベアラーを作成しAttach Acceptを送信（UE IP割り当て）

中を覗いてみましょう：

```bash
fairwave node status                     # このクラスターのコントロールプレーン表示
fairwave sim issue --count 2 --profile lab  # ラボSIMを2枚発行
fairwave spectrum check --country US --band n48 --indoor  # スペクトラムゲートのデモ
make lab-down                            # すべて停止
```

（ZMQタイミングが安定したLinuxホストでは、`docker exec -it ue1 ping -c3 10.45.0.1` で
`tun_srsue` トンネル経由の完全なデータパスを確認できます。）

上記はすべて**無線サイレント**です：無線は `srsran/zmq` 仮想RFデバイスでエミュレートされます。
実ハードウェアは [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) に従ってください -
法的チェックリストを完了しなければTXを有効化できません。

## ドキュメント

完全なサイト：**`make docs-serve` を実行** - または [`docs/`](docs/index.md) 以下でツリー内を読む。

| ここから始める | 次に |
|---|---|
| [ビジョン](docs/vision.md) | [アーキテクチャ](docs/architecture/overview.md) |
| [クイックスタート（30分、無線なし）](docs/tutorials/quickstart-no-rf.md) | [カフェパイロット（2時間、法的チェックリスト付き）](docs/tutorials/cafe-pilot.md) |
| [スペクトラムと法律](docs/spectrum-and-law/index.md) | [脅威モデル](design/threat-model.md) |
| [SIMライフサイクル](docs/sim-lifecycle/index.md) | [ピアリングファブリック](docs/peering/index.md) |
| [APIリファレンス](docs/api/index.md) | [ADR](docs/adr/0000-index.md) |

## ステータス

**ラボリリース `v0.1.0`**：EPC + zmq RAN + srsUEアタッチがエンドツーエンドで動作；コントロールプレーン、
CLI、SIM発行者、ラボeSIM（SM-DP+）、ドキュメントサイトが機能。さらに、free5GC ベースの 5G SA コアと CHF ベースの利用量計測がオプションで同梱されます（`core: free5gc`）。ZMQ の gNB/UE ラボプロファイルと CI 接続テスト付き。実RFパスは開発ハードウェアで検証済み
ですが、**デフォルトで無効**です。[ロードマップ](design/roadmap.md)を参照。

## Fairwaveが「ではない」もの

- **IMSIキャッチャーではない** - 受動的傍受はありません；UEはあなたが発行した認証情報で認証しなければなりません。
- **スペクトラムの無法地帯ではない** - TXはゲートされ、規制回避機能は拒否します。
- **無料の全国キャリアではない** - オプションのブレークアウト付きローカルカバレッジです。
- **緊急通報の代替ではない** - すべての展開で911/112の挙動を計画してください（[docs/ops/incident-response.md](docs/ops/incident-response.md)）。

## 貢献とガバナンス

コントリビューターを歓迎します。[CONTRIBUTING.md](CONTRIBUTING.md)（コードスタイル、DCO、テスト）、
[GOVERNANCE.md](GOVERNANCE.md)（意思決定の仕組み）、[SECURITY.md](SECURITY.md)
（脆弱性開示；脅威モデル）、[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) をお読みください。
ライセンス：**Apache-2.0**（[LICENSE](LICENSE)）；サードパーティの帰属は [NOTICE](NOTICE)。

---

<p align="center">
  <sub>HyperonXチームとFairwaveコミュニティによって作られました。空気はみんなのもの。</sub>
</p>
