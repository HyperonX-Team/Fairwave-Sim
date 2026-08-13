<h1 align="center">Fairwave - 피자 박스 속 커뮤니티 캐리어</h1>

<p align="center">
  <strong>오픈소스 프라이빗 LTE: 이더넷에 꽂고, 4G/5G를 내보내고, 이웃을 맞이하세요.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **법률 및 주파수 경고.** Fairwave는 기본적으로 **랩 / 무RF 모드**(제로 IF 루프백만)로 작동합니다.
> 적절한 허가 없이 셀룰러 대역에서 송신하는 것은 **대부분의 관할권에서 불법입니다**.
> 라이선스, SAS 허가, 실내 제한 및 형식 승인에 대한 책임은 전적으로 귀하에게 있습니다.
> HyperonX와 기여자들은 합법적인 사설 네트워크, 연구, 공유 주파수 제도만을 위해 소프트웨어를
> **있는 그대로** 제공합니다. [docs/spectrum-and-law/](docs/spectrum-and-law/index.md) 참조.

**다른 언어:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## 고장난 시스템

모바일 연결은 단 하나의 카르텔입니다: SIM 하나, 캐리어 하나, 계약 하나, 폐쇄된 정원.
커버리지 맵은 마케팅 브로슈어에 불과하고, 시골 거리는 그 맵에서 떨어지고, 아파트는 신호를 잃으며,
그리고 당신 - 돈을 내는 사람 - 은 당신을 서비스하는 인프라를 하나도 소유하지 않습니다. 국내 캐리어가
가격을 올리거나 타워가 침묵하면, 당신의 유일한 선택은… 같은 타워와 같은 조건의 다른 독점뿐입니다.

당신 주머니의 모뎀은 20미터 떨어진 기지국과 대화할 수 있습니다. 그것이 *당신의* 기지국과 대화하지
못하는 유일한 이유는, 그 기지국이 당신 것이 되는 것이 결코 허용되지 않았기 때문입니다.

## HyperonX의 해결책

**Fairwave는 커뮤니티 캐리어입니다: 피자 박스에 들어가고 일반 이더넷에 꽂히는 완전한 오픈소스 스몰셀.**

카페, 주택 협동조합, 마을 회관, 동네 - 누구나 하나를 운영할 수 있습니다:

1. SDR이 연결된 미니 PC(x86 또는 ARM)에서 Fairwave 이미지를 부팅합니다.
2. `fairwave node init`을 실행하고 규제 체크리스트에 응답합니다(허가를 증명할 때까지 TX는 꺼져 있습니다).
3. 커뮤니티를 위한 Fairwave SIM을 발급합니다 - 당신이 소유한 카드, 당신이 통제하는 자격 증명.
4. 휴대폰이 연결됩니다. 트래픽은 가능하면 로컬에 머물고, 여러 피자 박스는 메시를 이룹니다.
   원할 때 인터넷은 안전한 WireGuard 터널을 타고 나갑니다.

검증된 오픈 인프라 - **Open5GS**(EPC)와 **srsRAN**(eNB/gNB) - 위에 Go 컨트롤 플레인, 운영자
대시보드, 오프라인 SIM 발급, 그리고 제로 RF로 전체 캐리어를 Docker에서 실행하는 랩 모드를 갖췄습니다.

## 무엇을 얻을 수 있나요

- **캐리어 그 자체가 되는 피자 박스**: `fairwave-control`이 관리하는 노드 ID, 등록, 라이프사이클
  (`provision → register → on-air → peer → breakout`).
- **진짜 LTE 어태치**: Open5GS EPC + srsRAN eNB, 구성 가능한 PLMN, 트래킹 에어리어,
  `internet` + `ims` APN, 엣지 로컬 브레이크아웃.
- **Fairwave SIM 운영**: 오프라인 퍼스트 발급기; Ki/OPc 생성, 카드 뷰로용 CSV/JSON 배치,
  HSS/UDM 기록, 폐기 및 교체 관리. 랩과 프로덕션은 엄격히 분리.
- **랩 eSIM(SM-DP+)**: 자체 SGP.22 형태 프로파일 서버와 소프트웨어 eUICC -
  암호화된 바운드 프로파일 패키지, QR 활성화 코드(`LPA:1$...`), 하드웨어 없이 CI 검증되는
  완전한 다운로드 사이클. 설계상 랩 전용; GSMA 적합성은 오픈 항목으로 추적.
- **이웃 메시**: mDNS 디스커버리, mTLS 제어, WireGuard 데이터 플레인, 라우트 교환.
- **운영자 포털**: 로컬 퍼스트 대시보드 - UE(프라이버시 보호), 백홀, 피어, 랩 모드.
- **완전한 랩 모드**: srsUE와 함께 순수 IP(zmq)로 전체 네트워크 실행 - 무선 없음, 라이선스 없음, CI 친화적.
- **코드 속 규제 게이트**: TX 아밍에는 국가 코드, 라이선스 확인, 주파수 허용 목록이 필요합니다.
  그 외에는 컴파일 시 거부.

## 왜 기존 캐리어에 위협인가

캐리어 잠금은 *로컬 커버리지에 대해 선택 사항*이 됩니다. 카페도, 협동조합도, 마을도 각자 셀을 운영하고
메시로 연결할 수 있다면, 국가 SIM은 단지 로밍 옵션이지 문지기가 아닙니다. 오프로드, 중립 호스트 접근,
커뮤니티 가격 요금제는 더 이상 이론이 아닙니다. Fairwave는 통신 카르텔을 복제하지 않습니다 -
카르텔의 로컬 독점에 도전할 수 있게 만듭니다.

## 왜 지금 가능한가

- **Open5GS**와 **srsRAN**은 성숙하고, 프로덕션에 가깝고, 활발히 개발되고 있습니다.
- **SDR**(USRP, LimeSDR, BladeRF)은 수백 달러로 스몰셀 출력의 LTE 대역을 커버합니다.
- **공유 주파수는 현실입니다**: 미국의 CBRS, 영국/유럽의 로컬 라이선스, 사설 LTE 대역.
- **사설 LTE + Wi-Fi 통화**는 규제가 없는 척하는 대신 복사할 수 있는 합법적 템플릿을 제공합니다.
- 미니 PC급 하드웨어(그리고 개발용 Raspberry Pi CM4/5 + HAT)는 저렴하고 충분합니다.

---

## 퀵스타트 - 30분 안에 첫 UE 어태치(무선 없음, 라이선스 없음)

> 요구 사항: Docker Engine 24+, 8GB RAM. 모든 것이 컨테이너에서 실행되며, 아무것도 송신하지 않습니다.
> 전체 데이터 경로(UE IP + ping)는 네이티브 Linux를 사용하세요; Windows/macOS의 Docker Desktop은
> EPC 측 어태치 체크를 모두 통과합니다([docs](docs/tutorials/lab-attach.md) 참조).

```bash
./scripts/bootstrap.sh      # 툴체인 확인/설치 (Go, Docker, pre-commit)
make lab-up                 # 이미지 빌드, EPC + eNB + srsUE 기동, 어태치 체크 실행
make status                 # 한눈에 보는 상태: mme, sgwu/upf, enb, ue1
```

`make lab-up`은 다음을 모두 확인(및 출력)합니다:

1. Open5GS MME + HSS 실행 중
2. eNB S1-MME가 MME에 연결됨
3. 랩 PLMN에서 UE RRC 연결 + 랜덤 액세스
4. UE NAS 인증 + 보안 모드(HSS와 milenage)
5. MME가 기본 EPS 베어러를 만들고 Attach Accept 전송(UE IP 할당)

그런 다음 내부를 살펴보세요:

```bash
fairwave node status                     # 이 클러스터의 컨트롤 플레인 뷰
fairwave sim issue --count 2 --profile lab  # 랩 SIM 2장 발급
fairwave spectrum check --country US --band n48 --indoor  # 스펙트럼 게이트 데모
make lab-down                            # 모두 중지
```

(ZMQ 타이밍이 안정적인 Linux 호스트에서는 `docker exec -it ue1 ping -c3 10.45.0.1`로
`tun_srsue` 터널을 통해 전체 데이터 경로를 테스트할 수 있습니다.)

위의 모든 것은 **RF 사일런트**입니다: 무선은 `srsran/zmq` 가상 RF 장치로 에뮬레이션됩니다.
실제 하드웨어는 [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md)를 따르세요 -
법적 체크리스트를 완료하기 전에는 TX를 켤 수 없습니다.

## 문서

전체 사이트: **`make docs-serve` 실행** - 또는 [`docs/`](docs/index.md) 아래 트리에서 읽으세요.

| 여기서 시작 | 그 다음 |
|---|---|
| [비전](docs/vision.md) | [아키텍처](docs/architecture/overview.md) |
| [퀵스타트(30분, 무선 없음)](docs/tutorials/quickstart-no-rf.md) | [카페 파일럿(2시간, 법적 체크리스트 포함)](docs/tutorials/cafe-pilot.md) |
| [스펙트럼과 법](docs/spectrum-and-law/index.md) | [위협 모델](design/threat-model.md) |
| [SIM 라이프사이클](docs/sim-lifecycle/index.md) | [피어링 패브릭](docs/peering/index.md) |
| [API 레퍼런스](docs/api/index.md) | [ADR](docs/adr/0000-index.md) |

## 상태

**랩 릴리스 `v0.1.0`**: EPC + zmq RAN + srsUE 어태치가 엔드투엔드로 동작; 컨트롤 플레인, CLI,
SIM 발급기, 랩 eSIM(SM-DP+), 문서 사이트가 기능합니다. 또한 CHF 기반 사용량 측정을 갖춘 free5GC 5G SA 코어가 옵션으로 함께 제공됩니다(`core: free5gc`). ZMQ gNB/UE 랩 프로필과 CI 어태치 테스트 포함. 실제 RF 경로는 개발 하드웨어에서 검증
되었지만 **기본적으로 비활성화**되어 있습니다. [로드맵](design/roadmap.md) 참조.

## Fairwave가 아닌 것

- **IMSI 캐처가 아닙니다** - 수동 조회가 없습니다; UE는 당신이 발급한 자격 증명으로 인증해야 합니다.
- **주파수 무법지대가 아닙니다** - TX는 게이트되며, 규제 우회 기능을 거부합니다.
- **무료 국내 캐리어가 아닙니다** - 선택적 브레이크아웃이 있는 로컬 커버리지입니다.
- **긴급 통화의 대체물이 아닙니다** - 모든 배포에서 911/112 동작을 계획하세요([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## 기여와 거버넌스

기여자를 환영합니다. [CONTRIBUTING.md](CONTRIBUTING.md)(코드 스타일, DCO, 테스트),
[GOVERNANCE.md](GOVERNANCE.md)(의사 결정 방식), [SECURITY.md](SECURITY.md)
(취약점 공개; 위협 모델), [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)를 읽어보세요.
라이선스: **Apache-2.0**([LICENSE](LICENSE)); 타사 귀속은 [NOTICE](NOTICE).

---

<p align="center">
  <sub>HyperonX 팀과 Fairwave 커뮤니티가 만들었습니다. 공기는 모두의 것입니다.</sub>
</p>
