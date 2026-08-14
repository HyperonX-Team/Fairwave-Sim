---
title: Software Inventory
---

# Software Inventory

Fairwave's software is split into repo paths (one repository, multiple components) with a clean boundary: control plane, agent, CLI, UI, portal, and the containerized core/RAN stack.

## Inventory

| Path | Component | Language | What it does |
|---|---|---|---|
| `core/fairwave-control/` | fairwave-control | Go | identity, enrollment, reconcile loop, REST API v1, southbound drivers |
| `core/fairwave-agent/` | fairwave-agent | Go | hardware probes, heartbeat, watchdog, safe-TX flag |
| `apps/fairwave-cli/` | fairwave-cli | Go | operator commands against `/v1` |
| `apps/fairwave-ui/` | Operator UI | web (local-first) | dashboard: coverage, UEs, backhaul, peering, lab toggle |
| `apps/captive-portal/` | Captive portal | web | onboarding for Wi-Fi calling / non-cellular devices |
| `core/open5gs/` | Open5GS stack | containers | EPC: MME/SGW/PGW/HSS/PCRF |
| `core/ran/` | srsRAN stack | containers | eNB (and srsUE for lab) |
| `deploy/` | Docker Compose + ops | - | single-command lab stack, v0.1 default (compose, helm, ansible, terraform) |
| `docs/` | documentation | markdown | this site |
| `design/` | design docs | markdown | roadmap, threat model, spectrum matrix |
| `mkdocs.yml` | site config | - | docs rendering |

## Control Plane Dependencies

- Go (component language decision: `docs/adr/0003-control-plane-language.md`)
- Open5GS containers managed via southbound driver (config templating + reload)
- srsRAN eNB containers supervised via southbound driver
- Prometheus client library for `/metrics`; zerolog for structured logs
- OTLP exporter: stubbed, not wired (`docs/architecture/telemetry.md`)

## Container Images

| Image | Source | Tag policy |
|---|---|---|
| `fairwave/control` | built from `core/fairwave-control/` | per release (v0.1.0…) |
| `fairwave/agent` | built from `core/fairwave-agent/` | per release |
| `fairwave/ui` | built from `apps/fairwave-ui/` | per release |
| `fairwave/portal` | built from `apps/captive-portal/` | per release |
| `open5gs:dev` | pinned Open5GS build | pinned, hash-pinned in compose |
| `srsran:dev` | pinned srsRAN build | pinned, hash-pinned in compose |

## Build Matrix

| Platform | OS | Arch | Builds |
|---|---|---|---|
| CI (all PRs) | Linux | amd64 | control, agent, cli, ui, portal |
| CI (nightly) | Linux | arm64 | same, cross-compiled |
| Release | Linux | amd64 + arm64 | binaries + images + compose bundle |
| Golden image | Debian 12 | amd64 + arm64 | image playbook (`docs/hardware/image.md`) |

Release `v0.1.0` ships the lab stack (zmq, no RF). M-milestones gate RF features (`design/roadmap.md`).

## Related

- Control plane: `docs/software/fairwave-control.md`
- Agent: `docs/software/fairwave-agent.md`
- CLI: `docs/software/fairwave-cli.md`
- UI: `docs/software/operator-ui.md`
- Portal: `docs/software/captive-portal.md`
