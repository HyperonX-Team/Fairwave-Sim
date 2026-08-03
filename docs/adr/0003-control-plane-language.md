# ADR 0003: Control plane language — Go (not Rust)

- Status: Accepted
- Date: 2026-08-02

## Context

The Fairwave control plane (`fairwave-control`, `fairwave-agent`, `fairwave-cli`) is a
long-running daemon: it holds state, drives process configs, exposes REST+gRPC, and performs
background reconciliation. Candidate languages:

- **Go** — mature ops ecosystem (Prometheus, gRPC, Cobra, Docker client, Zerolog), fast compile, low barrier to contributor entry.
- **Rust** — stronger memory safety without a GC; modern async (tokio), but higher learning curve and slower iteration in a community dominated by network-ops contributors.

## Decision

Use **Go** for all control-plane components. If a truly unsafe hot path emerges (SIM-crypto
on constrained box, parser hardening), migrate that path to a Rust FFI later.

## Consequences

- [+] Faster development velocity for a community ops tool.
- [+] Trivially cross-compiles to ARM (CM4, CM5, routers).
- [+] Ecosystem alignment: Prometheus, gRPC, container tooling.
- [!] Runtime GC pauses must be measured under heavy signaling; if they appear in benchmarks
  we rework hot paths (object pooling, arena buffers) rather than default to unsafe Rust.
- [!] Type safety over `map[string]interface{}` config must be enforced with `go-playground/validator` and codegen.

## Alternatives considered

- **Rust** — excellent safety, but slower onboarding for community contributors and typical
  workloads here are I/O-bound where Go's GC is acceptable. Decision: Go now, Rust later for
  proven hot paths, via explicit hot-path ADR.
