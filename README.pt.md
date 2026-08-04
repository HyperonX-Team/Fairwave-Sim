<h1 align="center">Fairwave - uma operadora comunitária em uma caixa de pizza</h1>

<p align="center">
  <strong>LTE privado de código aberto: conecte ao Ethernet, emita 4G, dê boas-vindas aos seus vizinhos.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **AVISO LEGAL E DE ESPECTRO.** O Fairwave opera por padrão no modo **laboratório / sem RF** (somente loopback de FI zero).
> Transmitir em bandas celulares sem a devida autorização é **ilegal na maioria das jurisdições**.
> Você é o único responsável por licenças, concessões SAS, restrições indoor e homologação de tipo.
> A HyperonX e os colaboradores fornecem o software **como está** apenas para redes privadas legais, pesquisa
> e regimes de espectro compartilhado. Veja [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**Leia em:** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## O sistema quebrado

A conectividade móvel é um cartel de um: um SIM, uma operadora, um contrato, um jardim murado.
Os mapas de cobertura são folhetos de marketing; as ruas rurais caem deles; os apartamentos perdem sinal;
e você - a pessoa que paga - não é dono de nenhuma das infraestruturas que o servem. Se a sua operadora
nacional aumentar o preço ou a torre ficar em silêncio, sua única opção é… um monopólio diferente com as
mesmas torres e as mesmas condições.

O modem no seu bolso pode falar com uma estação base a 20 metros. A única razão pela qual ele não fala com
a *sua* estação base é que essa estação nunca teve permissão de ser sua.

## A solução HyperonX

**Fairwave é a operadora comunitária: uma small cell completa e de código aberto que cabe em uma caixa de
pizza e se conecta a um Ethernet comum.**

Um café, uma cooperativa habitacional, um salão de vila, um bairro - qualquer pessoa pode operar uma:

1. Inicie a imagem Fairwave em um mini-PC (x86 ou ARM) com um SDR conectado.
2. Execute `fairwave node init`, responda ao checklist regulatório (a TX permanece desligada até você provar a autorização).
3. Emita SIMs Fairwave para a sua comunidade - cartões que você possui, credenciais que você controla.
4. Os telefones se conectam. O tráfego permanece local quando possível; várias caixas de pizza formam uma mesh;
   o acesso à internet viaja por um túnel WireGuard seguro quando você quiser.

Construído sobre infraestrutura aberta comprovada - **Open5GS** (EPC) e **srsRAN** (eNB/gNB) - com um plano
de controle em Go, um painel de operadora, provisionamento offline de SIM e um modo laboratório que executa
toda a operadora em Docker com zero RF.

## O que você obtém

- **Uma caixa de pizza que é uma operadora**: identidade de nó, registro, ciclo de vida
  (`provision → register → on-air → peer → breakout`) gerenciado pelo `fairwave-control`.
- **Attach LTE real**: Open5GS EPC + srsRAN eNB, PLMN configurável, áreas de tracking,
  APNs `internet` + `ims`, breakout local na borda.
- **Operações SIM Fairwave**: provisionador offline-first; gera Ki/OPc, lotes CSV/JSON para
  bureaus de cartões, escreve em HSS/UDM, controles de revogação e troca. Laboratório e produção estritamente separados.
- **eSIM de laboratório (SM-DP+)**: seu próprio servidor de perfis em formato SGP.22 e uma eUICC de software -
  pacotes de perfil vinculado criptografados, códigos de ativação QR (`LPA:1$...`), ciclo de download completo
  verificado em CI sem hardware. Apenas laboratório por design; conformidade GSMA rastreada como itens em aberto.
- **Mesh de vizinhança**: descoberta mDNS, controle mTLS, plano de dados WireGuard, troca de rotas.
- **Portal da operadora**: painel local-first - UEs (preservando privacidade), backhaul, peers, modo laboratório.
- **Modo laboratório completo**: toda a rede em IP puro (zmq) com srsUE - sem rádio, sem licença, amigável à CI.
- **Portões regulatórios no código**: armar a TX exige código do país, reconhecimento de licença
  e whitelist de frequências. Recusa compilada caso contrário.

## Por que ameaça as operadoras incumbentes

O lock-in da operadora torna-se *opcional para a cobertura local*. Quando o café, a cooperativa e a vila
podem operar cada um uma célula e conectá-las em mesh, o SIM nacional é apenas uma opção de roaming,
não o porteiro. Offload, acesso neutral host e planos a preço comunitário deixam de ser teóricos.
O Fairwave não replica o cartel de telecomunicações - torna o monopólio local do cartel contestável.

## Por que é viável agora

- **Open5GS** e **srsRAN** são maduros, próximos da produção e desenvolvidos ativamente.
- **SDRs** (USRP, LimeSDR, BladeRF) cobrem bandas LTE a potência de small cell por centenas de dólares.
- **Espectro compartilhado é real**: CBRS nos EUA, licenciamento local no Reino Unido/UE, bandas LTE privadas.
- **LTE privado + chamadas Wi-Fi** dão modelos legais para copiar em vez de fingir que a regulação não existe.
- Hardware classe mini-PC (e Raspberry Pi CM4/5 + HATs para desenvolvimento) é barato e suficiente.

---

## Início rápido - primeiro attach de UE em <30 minutos (sem RF, sem licença)

> Requisitos: Docker Engine 24+, 8 GB de RAM. Tudo roda em contêineres; nada transmite.
> Para o caminho de dados completo (IP do UE + ping) use Linux nativo; Docker Desktop no
> Windows/macOS passa em todos os checks de attach do lado do EPC (veja [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # verifica/instala toolchains (Go, Docker, pre-commit)
make lab-up                 # constrói imagens, sobe EPC + eNB + srsUE, executa checks de attach
make status                 # saúde de relance: mme, sgwu/upf, enb, ue1
```

`make lab-up` verifica (e imprime) tudo a seguir:

1. Open5GS MME + HSS em execução
2. eNB S1-MME conectado ao MME
3. Conexão RRC do UE + acesso aleatório no PLMN de laboratório
4. Autenticação NAS do UE + modo de segurança (milenage contra HSS)
5. O MME cria o bearer EPS padrão e envia Attach Accept (IP do UE alocado)

Depois olhe para dentro:

```bash
fairwave node status                     # visão do plano de controle deste cluster
fairwave sim issue --count 2 --profile lab  # emite dois SIMs de laboratório
fairwave spectrum check --country US --band n48 --indoor  # demo do portão espectral
make lab-down                            # para tudo
```

(Em hosts Linux com timing ZMQ estável, `docker exec -it ue1 ping -c3 10.45.0.1`
exercita o caminho de dados completo sobre o túnel `tun_srsue`.)

Tudo acima é **silencioso em RF**: o rádio é emulado com dispositivos virtuais `srsran/zmq`.
Para hardware real, siga [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
que não permitirá ativar a TX sem concluir o checklist legal.

## Documentação

Site completo: **execute `make docs-serve`** - ou leia na árvore sob [`docs/`](docs/index.md).

| Comece aqui | Depois |
|---|---|
| [Visão](docs/vision.md) | [Arquitetura](docs/architecture/overview.md) |
| [Início rápido (30 min, sem RF)](docs/tutorials/quickstart-no-rf.md) | [Piloto de café (2 h, com checklist legal)](docs/tutorials/cafe-pilot.md) |
| [Espectro e lei](docs/spectrum-and-law/index.md) | [Modelo de ameaças](design/threat-model.md) |
| [Ciclo de vida do SIM](docs/sim-lifecycle/index.md) | [Tecido de peering](docs/peering/index.md) |
| [Referência da API](docs/api/index.md) | [ADRs](docs/adr/0000-index.md) |

## Status

**Versão de laboratório `v0.1.0`**: attach EPC + zmq RAN + srsUE funciona de ponta a ponta; plano de controle,
CLI, provisionador de SIM, eSIM de laboratório (SM-DP+) e o site de documentação são funcionais. Os caminhos
de RF real são validados em hardware de desenvolvimento, mas **desabilitados por padrão**.
Veja o [roadmap](design/roadmap.md).

## O que o Fairwave NÃO é

- **Não é um IMSI catcher** - sem interrogatório passivo; UEs devem se autenticar com credenciais que você provisionou.
- **Não é um faroeste espectral** - a TX é portão, e recusamos recursos de bypass regulatório.
- **Não é uma operadora nacional gratuita** - é cobertura local com breakout opcional.
- **Não é um substituto para chamadas de emergência** - planeje o comportamento 911/112 em cada implantação ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Contribuir e governança

Damos as boas-vindas a colaboradores. Leia [CONTRIBUTING.md](CONTRIBUTING.md) (estilo de código, DCO, testes),
[GOVERNANCE.md](GOVERNANCE.md) (como as decisões são tomadas), [SECURITY.md](SECURITY.md)
(divulgação de vulnerabilidades; modelo de ameaças) e [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Licença: **Apache-2.0** ([LICENSE](LICENSE)); atribuição de terceiros em [NOTICE](NOTICE).

---

<p align="center">
  <sub>Construído pela equipe HyperonX e pela comunidade Fairwave. O ar é de todos.</sub>
</p>
