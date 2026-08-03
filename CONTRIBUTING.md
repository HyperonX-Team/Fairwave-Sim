# Contributing to Fairwave

> [!IMPORTANT]
> **Legal banner** — Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands
> without authorization is illegal in most jurisdictions. Contributions that bypass
> authorization gates will not be merged.

Thanks for helping us pry open the last mile. This guide gets you from clone to green
CI in under an hour.

## Ways to contribute

- **Report bugs** — open an issue, pick a template
- **Fix docs** — factual corrections, typos, better tutorials all welcome
- **Write ADRs** — contributing an Architecture Decision Record helps us preserve knowledge
- **Code** — see below
- **Test on hardware** — SDR + pilot notes under [hw/](hw/) are highly valued

## Development setup

```bash
git clone https://github.com/HyperonX-Team/Fairwave-Sim.git
cd Fairwave-Sim
./scripts/bootstrap.sh       # installs Go, Docker helpers, pre-commit
make lab-up                  # smoke runs the whole stack in no-RF mode
make check                   # fmt + lint + unit tests + secrets scan
```

Install pre-commit hooks:

```bash
pip install pre-commit
pre-commit install
```

## PR standards

- **Conventional Commits**: `feat:`, `fix:`, `docs:`, `refactor:`, `perf:`, `test:`, `ci:`, `chore:`, `revert:`
- Keep PRs small and focused (<400 LOC where possible)
- Add or update tests that cover the change
- Update **docs** when behavior changes
- For **API changes**, regenerate stubs: `make genapi`

## Testing requirements

- `go test ./...` must pass
- `make lab-up` must bring the stack up on clean machines
- End-to-end: `make test-e2e-lab` must pass
- For RF-touching PRs: rfdry-run test (`make rf-dry-run`) required; **never** attach real radios in CI

## Code style

- **Go**: use `gofmt`/`gofumpt`; ban `fmt.Println`, prefer structured logging; context deadlines everywhere; no global state. 
- **Rust** (if introduced later): no `unwrap()` in production code paths; `thiserror`/`anyhow`; `clippy` enforced.
- **Shell**: only POSIX-compatible or `bash` with explicit safety (`set -euo pipefail`).

## DCO / sign-off

All commits must be signed off (`git commit -s`). By signing off, you certify the
Developer Certificate of Origin 1.1. We do not use a CLA.

## Security-sensitive PRs

Anything that:
- touches `tx_arm`, spectrum gating, SIM provisioning, crypto, auth
- modifies Dockerfiles, base images, or system-level scripts
- changes HTTP/gRPC auth

...requires a **second-review** by a maintainer listed as codeowner for that path and
**must include a threat-model note** in the PR description.
