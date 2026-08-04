<h1 align="center">Fairwave - 一个装在披萨盒里的社区运营商</h1>

<p align="center">
  <strong>开源私有 LTE：插上以太网，发射 4G，欢迎你的邻居。</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **法律与频谱警告。** Fairwave 默认使用**实验室 / 无射频**模式（仅零中频回环）。
> 未经适当授权在蜂窝频段发射在**大多数司法辖区是违法的**。
> 许可证、SAS 授权、室内限制与型号核准完全由你负责。
> HyperonX 与贡献者仅按"原样"提供软件，用于合法的私有网络、研究
> 与共享频谱制度。参见 [docs/spectrum-and-law/](docs/spectrum-and-law/index.md)。

**其他语言：** [English](README.md) · [العربية](README.ar.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md)

---

## 被破坏的系统

移动连接是一个独裁联盟：一张 SIM、一家运营商、一份合同、一座围墙花园。
覆盖地图只是营销手册；乡村街道从地图上掉下去；公寓楼信号渗漏；
而你——付钱的人——不拥有任何服务你的基础设施。如果国家运营商的涨价
或基站停摆，你唯一的选择就是……换一家垄断，用着同样的铁塔和同样的条款。

你口袋里的调制解调器可以和 20 米外的基站通话。它不和你*自己的*基站通话的
唯一原因，是那座基站从来就不被允许属于你。

## HyperonX 的解法

**Fairwave 就是社区运营商：一个完整的、开源的小基站，装进披萨盒，
插上普通以太网就能用。**

咖啡馆、住房合作社、村礼堂、城镇——任何人都可以运行一个：

1. 在带 SDR 的迷你 PC（x86 或 ARM）上启动 Fairwave 镜像。
2. 运行 `fairwave node init`，回答监管清单（在证明授权之前发射保持关闭）。
3. 为你的社区签发 Fairwave SIM 卡——你拥有的卡，你掌控的凭据。
4. 手机接入。流量尽可能留在本地；多台披萨盒组成网状网络；
   需要时通过安全的 WireGuard 隧道上网。

基于成熟的开源基础设施——**Open5GS**（EPC）和 **srsRAN**（eNB/gNB）——加上
Go 控制面、运营商仪表盘、离线 SIM 签发，以及完全在 Docker 中以零射频
运行整个运营商的实验室模式。

## 你能得到什么

- **一台披萨盒就是一家运营商**：节点身份、注册、生命周期
  （`provision → register → on-air → peer → breakout`）由 `fairwave-control` 管理。
- **真正的 LTE 接入**：Open5GS EPC + srsRAN eNB，可配置 PLMN、跟踪区、
  `internet` + `ims` APN，边缘本地分流。
- **Fairwave SIM 运营**：离线优先签发器；生成 Ki/OPc，为卡商批量输出 CSV/JSON，
  写入 HSS/UDM，支持吊销与换卡。实验室与生产严格分离。
- **实验室 eSIM（SM-DP+）**：你自己的 SGP.22 形态配置服务器与软件 eUICC——
  加密的绑定配置包、QR 激活码（`LPA:1$...`），完整下载流程无需硬件即可 CI 验证。
  设计上仅限实验室；GSMA 一致性列为开放事项。
- **邻里网状网络**：mDNS 发现、mTLS 控制、WireGuard 数据面、路由交换。
- **运营商门户**：本地优先仪表盘——UE（隐私保护）、回传、对等节点、实验室模式。
- **完整实验室模式**：整个网络跑在纯 IP（zmq）上，配合 srsUE——无射频、无需许可、CI 友好。
- **代码中的监管闸门**：武装发射要求国家代码、许可确认、频率白名单。
  否则编译期内置拒绝。

## 为什么它威胁现有运营商

对本地覆盖而言，运营商的锁定变得*可有可无*。当咖啡馆、合作社和村庄
都能各自运行一个小区并互连时，全国 SIM 卡只是漫游选项，而不是守门人。
分流、中立主机接入和社区定价套餐不再只是理论。Fairwave 不是复制电信联盟——
它让联盟的本地垄断变得可竞争。

## 为什么现在可行

- **Open5GS** 和 **srsRAN** 成熟、接近生产级、持续活跃开发。
- **SDR**（USRP、LimeSDR、BladeRF）以数百美元的价格覆盖小基站功率的 LTE 频段。
- **共享频谱是真实的**：美国的 CBRS、英国/欧盟的地方许可、私有 LTE 频段。
- **私有 LTE + Wi-Fi 通话**提供了合法的可复制模板，而不是假装监管不存在。
- 迷你 PC 级硬件（以及用于开发的 Raspberry Pi CM4/5 + HAT）便宜且够用。

---

## 快速开始 - 30 分钟内首次 UE 接入（无需射频、无需许可）

> 要求：Docker Engine 24+，8 GB 内存。所有内容都在容器中运行；没有任何发射。
> 完整数据通路（UE IP + ping）请使用原生 Linux；Windows/macOS 上的 Docker Desktop
> 通过所有 EPC 侧接入检查（参见 [docs](docs/tutorials/lab-attach.md)）。

```bash
./scripts/bootstrap.sh      # 检查/安装工具链（Go、Docker、pre-commit）
make lab-up                 # 构建镜像，拉起 EPC + eNB + srsUE，运行接入检查
make status                 # 一目了然的健康状态：mme、sgwu/upf、enb、ue1
```

`make lab-up` 断言（并打印）以下所有内容：

1. Open5GS MME + HSS 运行中
2. eNB S1-MME 已连接 MME
3. UE RRC 连接 + 在实验室 PLMN 上随机接入
4. UE NAS 认证 + 安全模式（milenage 对 HSS）
5. MME 创建默认 EPS 承载并发送 Attach Accept（分配 UE IP）

然后看看内部：

```bash
fairwave node status                     # 该集群的控制面视图
fairwave sim issue --count 2 --profile lab  # 签发两张实验室 SIM
fairwave spectrum check --country US --band n48 --indoor  # 频谱闸门演示
make lab-down                            # 停止一切
```

（在 ZMQ 时序稳定的 Linux 主机上，`docker exec -it ue1 ping -c3 10.45.0.1`
会通过 `tun_srsue` 隧道完整测试数据通路。）

以上所有内容**射频静默**：无线电由 `srsran/zmq` 虚拟射频设备模拟。
真实硬件请遵循 [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md)——
不完成法律清单它不会让你启用发射。

## 文档

完整站点：**运行 `make docs-serve`**——或在树内 [`docs/`](docs/index.md) 阅读。

| 从这里开始 | 然后 |
|---|---|
| [愿景](docs/vision.md) | [架构](docs/architecture/overview.md) |
| [快速开始（30 分钟，无射频）](docs/tutorials/quickstart-no-rf.md) | [咖啡馆试点（2 小时，含法律清单）](docs/tutorials/cafe-pilot.md) |
| [频谱与法律](docs/spectrum-and-law/index.md) | [威胁模型](design/threat-model.md) |
| [SIM 生命周期](docs/sim-lifecycle/index.md) | [对等网络](docs/peering/index.md) |
| [API 参考](docs/api/index.md) | [ADR](docs/adr/0000-index.md) |

## 状态

**实验室版本 `v0.1.0`**：EPC + zmq RAN + srsUE 接入端到端可用；控制面、CLI、
SIM 签发器、实验室 eSIM（SM-DP+）与文档站均可运行。真实射频路径已在开发硬件上
验证，但**默认禁用**。参见[路线图](design/roadmap.md)。

## Fairwave 不是什么

- **不是 IMSI 捕手**——没有被动询问；UE 必须使用你签发的凭据认证。
- **不是频谱无政府状态**——发射有闸门，我们拒绝监管绕过功能。
- **不是免费的国家运营商**——它是本地覆盖 + 可选分流。
- **不是紧急呼叫的替代品**——在每个部署中规划 911/112 行为（[docs/ops/incident-response.md](docs/ops/incident-response.md)）。

## 贡献与治理

我们欢迎贡献者。阅读 [CONTRIBUTING.md](CONTRIBUTING.md)（代码风格、DCO、测试）、
[GOVERNANCE.md](GOVERNANCE.md)（决策方式）、[SECURITY.md](SECURITY.md)
（漏洞披露；威胁模型）和 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。
许可证：**Apache-2.0**（[LICENSE](LICENSE)）；第三方归属见 [NOTICE](NOTICE)。

---

<p align="center">
  <sub>由 HyperonX 团队和 Fairwave 社区构建。天空属于所有人。</sub>
</p>
