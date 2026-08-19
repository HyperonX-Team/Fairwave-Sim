# justfile - faster alternative to make (just install once)
set shell := ["bash", "-uc"]

version := env_var_or_default("VERSION", "0.1.0")

default:
    just --list

bootstrap:
    ./scripts/bootstrap.sh

check: fmt vet test lint

all: check

fmt:
    gofmt -l -w core apps
    go mod tidy

vet:
    go vet ./...

test:
    go test -race -count=1 ./...

lint:
    golangci-lint run --config=.golangci.yml ./...

gen:
    ./scripts/gen-docs.sh

build:
    mkdir -p bin
    go build -ldflags "-X github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/api.Version={{version}} -X github.com/HyperonX-Team/Fairwave-Sim/apps/fairwave-cli/internal/cli.Version={{version}}" -o bin/fairwave-control ./core/fairwave-control/cmd/fairwave-control
    go build -ldflags "-X github.com/HyperonX-Team/Fairwave-Sim/apps/fairwave-cli/internal/cli.Version={{version}}" -o bin/fairwave-agent ./core/fairwave-agent/cmd/fairwave-agent
    go build -ldflags "-X github.com/HyperonX-Team/Fairwave-Sim/apps/fairwave-cli/internal/cli.Version={{version}}" -o bin/fairwave ./apps/fairwave-cli/cmd/fairwave

lab-up:
    docker compose -f deploy/docker-compose.yml up -d --build
    echo "== seeding lab subscribers into HSS =="
    docker compose -f deploy/docker-compose.yml exec -T mongo bash /init/hss-init.sh || echo "[warn] HSS seed failed (manual: see docs/sim-lifecycle/bureau-runbook.md)"
    ./tests/e2e-sim/assert-lab-up.sh

lab-down:
    docker compose -f deploy/docker-compose.yml down -v

# Compact profile: the same no-RF lab tuned for a 4 GB RAM laptop. Layers the
# compact override (memory limits + FW_DAEMONS subset) on the base compose.
compact-up:
    docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.compact.yml up -d --build
    echo "== seeding lab subscribers into HSS =="
    docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.compact.yml exec -T mongo bash /init/hss-init.sh || echo "[warn] HSS seed failed (manual: see docs/sim-lifecycle/bureau-runbook.md)"
    ./tests/e2e-sim/assert-lab-up.sh

compact-down:
    docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.compact.yml down -v

compact-status:
    docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.compact.yml ps

lab-status:
    docker compose -f deploy/docker-compose.yml ps
    echo "--- attach check ---"
    docker logs ue1 2>&1 | tail -5 || true

# alias used in the README quickstart
status: lab-status

# ---- e2e (full lab: requires docker) ----
e2e: lab-up

rf-dry-run:
    docker compose -f deploy/docker-compose.rf.yml config -q && echo "rf compose OK"
    ./core/ran/check-freq.sh 2>/dev/null || echo "note: check-freq.sh not present; manual review required"

test-e2e-lab:
    echo "== e2e against live lab (requires: make lab-up) =="
    go test ./tests/e2e-sim/ -count=1 -v

docs:
    ./scripts/gen-docs.sh

docs-serve:
    cd docs && mkdocs serve

sbom:
    syft dir:. --output spdx-json --file sbom.spdx.json
    command -v cosign >/dev/null 2>&1 && cosign attest-blob --yes sbom.spdx.json >/dev/null 2>&1 || echo "cosign not present; SBOM generated unsigned"

release:
    ./scripts/release.sh {{version}}

help:
    just --list
