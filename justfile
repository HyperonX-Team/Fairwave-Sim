# justfile — faster alternative to make (just install once)
set shell := ["bash", "-uc"]

version := "0.1.0"

default:
    just --list

bootstrap:
    ./scripts/bootstrap.sh

check: fmt vet test lint

fmt:
    gofmt -l -w core apps
    go mod tidy

vet:
    go vet ./...

test:
    go test ./... -count=1

lint:
    golangci-lint run --config=.golangci.yml ./...

build:
    mkdir -p bin
    go build -o bin/fairwave-control ./core/fairwave-control/cmd/fairwave-control
    go build -o bin/fairwave-agent ./core/fairwave-agent/cmd/fairwave-agent
    go build -o bin/fairwave ./apps/fairwave-cli/cmd/fairwave

lab-up:
    docker compose -f deploy/docker-compose.yml up -d --build
    ./tests/e2e-sim/assert-lab-up.sh

lab-down:
    docker compose -f deploy/docker-compose.yml down -v

lab-status:
    docker compose -f deploy/docker-compose.yml ps

docs:
    ./scripts/gen-docs.sh

docs-serve:
    cd docs && mkdocs serve

rf-dry-run:
    docker compose -f deploy/docker-compose.rf.yml config -q && echo "rf compose OK"

sbom:
    syft dir:. --output spdx-json --file sbom.spdx.json

release:
    ./scripts/release.sh {{version}}

help:
    just --list
