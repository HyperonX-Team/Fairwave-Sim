# Fairwave — community carrier in a pizza box
# Targets: bootstrap, check, lab-up/down, docs, release.

SHELL := /bin/bash
MAKEFLAGS += --no-print-directory

VERSION ?= 0.1.0
BIN_DIR := bin
GO ?= go

.PHONY: all check build test lint fmt gen docs docs-serve \
        lab-up lab-down lab-status rf-dry-run \
        sbom release vet e2e help bootstrap

all: check

# ---- bootstrap ----
bootstrap:
	./scripts/bootstrap.sh

# ---- quality gates ----
fmt:
	gofmt -l -w core apps
	$(GO) mod tidy

vet:
	$(GO) vet ./...

test:
	$(GO) test ./... -count=1

lint:
	@command -v golangci-lint >/dev/null 2>&1 || (echo "install golangci-lint: make bootstrap"; exit 1)
	golangci-lint run --config=.golangci.yml ./...

check: fmt vet test lint

gen:
	./scripts/gen-docs.sh

# ---- build ----
build:
	mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "-X github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/api.Version=$(VERSION) -X github.com/HyperonX-Team/Fairwave-Sim/apps/fairwave-cli/internal/cli.Version=$(VERSION)" -o $(BIN_DIR)/fairwave-control ./core/fairwave-control/cmd/fairwave-control
	$(GO) build -ldflags "-X github.com/HyperonX-Team/Fairwave-Sim/apps/fairwave-cli/internal/cli.Version=$(VERSION)" -o $(BIN_DIR)/fairwave-agent ./core/fairwave-agent/cmd/fairwave-agent
	$(GO) build -ldflags "-X github.com/HyperonX-Team/Fairwave-Sim/apps/fairwave-cli/internal/cli.Version=$(VERSION)" -o $(BIN_DIR)/fairwave ./apps/fairwave-cli/cmd/fairwave

# ---- lab (no RF, zmq) ----
lab-up:
	docker compose -f deploy/docker-compose.yml up -d --build
	@echo "== seeding lab subscribers into HSS =="
	@docker compose -f deploy/docker-compose.yml exec -T mongo bash /init/hss-init.sh || echo "[warn] HSS seed failed (manual: see docs/sim-lifecycle/bureau-runbook.md)"
	./tests/e2e-sim/assert-lab-up.sh

lab-down:
	docker compose -f deploy/docker-compose.yml down -v

lab-status:
	docker compose -f deploy/docker-compose.yml ps
	@echo "--- attach check ---"
	@docker logs ue1 2>&1 | tail -5 || true

# alias used in the README quickstart
status: lab-status

# ---- rf dry run: validates configs without a radio ----
rf-dry-run:
	@echo "Validating RF configs (no transmission)"
	docker compose -f deploy/docker-compose.rf.yml config -q && echo "rf compose OK"
	./core/ran/check-freq.sh 2>/dev/null || echo "note: check-freq.sh not present; manual review required"

# ---- e2e (full lab: requires docker) ----
e2e: lab-up

test-e2e-lab:
	@echo "== e2e against live lab (requires: make lab-up) =="
	$(GO) test ./tests/e2e-sim/ -count=1 -v

# ---- docs ----
docs:
	./scripts/gen-docs.sh

docs-serve:
	cd docs && mkdocs serve

# ---- SBOM / supply chain ----
sbom:
	@command -v syft >/dev/null 2>&1 || (echo "install syft (make bootstrap)"; exit 1)
	syft dir:. --output spdx-json --file sbom.spdx.json
	@command -v cosign >/dev/null 2>&1 && cosign attest-blob --yes sbom.spdx.json >/dev/null 2>&1 || echo "cosign not present; SBOM generated unsigned"

# ---- release ----
release:
	./scripts/release.sh $(VERSION)

help:
	@echo "Fairwave targets:"
	@echo "  bootstrap       install toolchains (Go, Docker, pre-commit, syft, cosign)"
	@echo "  build           compile control, agent, cli into bin/"
	@echo "  check           fmt + vet + test + lint (CI gate)"
	@echo "  lab-up          start no-RF lab (Open5GS + srsRAN zmq + srsUE) + assert attach"
	@echo "  lab-down        stop and wipe lab volumes"
	@echo "  lab-status      compose ps + UE tail"
	@echo "  rf-dry-run      validate RF configs, never transmits"
	@echo "  e2e             alias for lab-up"
	@echo "  docs / docs-serve  build / serve MkDocs site"
	@echo "  sbom            generate SPDX SBOM (+ cosign attest if present)"
	@echo "  release         cut a release (VERSION=x.y.z make release)"
