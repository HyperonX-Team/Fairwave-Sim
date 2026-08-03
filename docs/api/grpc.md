---
title: gRPC Note
---

# gRPC Note (Honest Status)

Fairwave's API story has two layers, and the second one is not shipped yet.

## What exists

- **REST (canonical, v0.1):** the OpenAPI contract at `api/openapi.yaml` is the source of truth. Everything in [rest.md](rest.md) works today.
- **Protos (contract-only):** service definitions live in `api/proto/*.proto` and express the *intended* RPC surface: control plane services for status, SIMs, peers, policy, spectrum, TX arm, and lifecycle transitions.

Proto services (names as defined in `api/proto/`):

| Service | Notable RPCs |
| --- | --- |
| `fairwave.v1.Health` | `Healthz`, `Version` |
| `fairwave.v1.Node` | `GetStatus`, `ListNodes` |
| `fairwave.v1.Sims` | `Issue`, `List`, `Revoke`, `Wipe` |
| `fairwave.v1.Peers` | `List`, `Join`, `Leave` |
| `fairwave.v1.Policy` | `Get`, `Update` |
| `fairwave.v1.Spectrum` | `Check` |
| `fairwave.v1.TxArm` | `Get`, `Set` |
| `fairwave.v1.Lifecycle` | `Transition` |

## Codegen

- Protos are linted and generated with **buf** in CI (`buf lint`, `buf generate`).
- Generated stubs are committed in `api/gen/` so downstream consumers (agent, CLI, UI) build without buf locally.
- Breaking proto changes require a new package version (`fairwave.v2`) — same rule as REST.

## Current status and roadmap

- **REST is canonical for v0.1.** No gRPC server is served in the v0.1.0 lab release; the protos are aspirational contracts used to shape the service boundaries.
- **gRPC gateway (M1):** roadmap milestone M1 adds a gRPC server with the gateway serving REST/JSON on the same endpoints. Until M1, do not expect `grpcurl` to work against a running node.
- Protos and REST share semantics; the OpenAPI file remains the tie-breaker where they diverge.

## What this means for you

- If you integrate today: use REST against `:8080/v1/...`.
- If you plan ahead: import `api/gen/` for typed clients, and expect them to start working against live servers at M1.
- If you saw gRPC mentioned in a talk: that is the roadmap, not the release.

## Related

- [API overview](index.md) · [REST reference](rest.md) · `api/openapi.yaml` · `/design/roadmap.md`
