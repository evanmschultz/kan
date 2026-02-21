set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

default:
  @just --list

bootstrap:
  @go mod tidy
  @if ! command -v vhs >/dev/null; then \
    echo "vhs is optional for visual previews: brew install vhs"; \
  fi

fmt:
  @set -- $(git ls-files '*.go'); \
  if [ "$#" -gt 0 ]; then \
    gofmt -w "$@"; \
  fi

test:
  @go test ./...

test-unit:
  @go test ./internal/domain ./internal/app ./internal/config ./internal/platform ./internal/adapters/storage/sqlite

test-tui:
  @go test ./internal/tui

test-pkg pkg:
  @go test {{pkg}}

test-golden:
  @go test ./internal/tui -run 'Golden'

test-golden-update:
  @go test ./internal/tui -run 'Golden' -update

build:
  @go build -o ./kan ./cmd/kan

build-all:
  @go build ./...

run:
  @go run ./cmd/kan

run-dev:
  @go run ./cmd/kan --dev

paths:
  @go run ./cmd/kan paths

run-bin: build
  @./kan

export out=".artifacts/kan-export.json" db=".artifacts/kan.db": build
  @mkdir -p "$(dirname '{{out}}')"
  @KAN_DB_PATH='{{db}}' ./kan export --out '{{out}}'

import in db=".artifacts/kan.db": build
  @KAN_DB_PATH='{{db}}' ./kan import --in '{{in}}'

vhs-board: build
  @mkdir -p .artifacts/vhs
  @vhs vhs/board.tape

vhs-workflow: build
  @mkdir -p .artifacts/vhs
  @vhs vhs/workflow.tape

vhs: vhs-board vhs-workflow

clean-vhs:
  @rm -rf .artifacts/vhs

release-check:
  @if ! command -v goreleaser >/dev/null; then \
    echo "goreleaser is required for release checks (brew install goreleaser/tap/goreleaser)"; \
    exit 1; \
  fi
  @goreleaser release --snapshot --clean --skip=publish

coverage:
  @tmp=$(mktemp); \
  go test ./... -cover | tee "$tmp"; \
  awk 'BEGIN {bad=0} /coverage:/ {cov=$5; gsub("%","",cov); if ((cov+0) < 70) {print "coverage below 70%:", $2, cov "%"; bad=1}} END {exit bad}' "$tmp"; \
  rm -f "$tmp"

ci: fmt test coverage build-all

check-llm: ci
