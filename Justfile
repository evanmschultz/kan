set shell := ["bash", "-eu", "-o", "pipefail", "-c"]
set windows-shell := ["C:/Program Files/Git/bin/bash.exe", "-eu", "-o", "pipefail", "-c"]

[private]
verify-sources:
  @git ls-files --error-unmatch cmd/kan/main.go cmd/kan/main_test.go >/dev/null

fmt:
  @set -- $(git ls-files '*.go'); \
  if [ "$#" -gt 0 ]; then \
    gofmt -w "$@"; \
  fi

test:
  @go test ./...

test-pkg pkg:
  @pkg="{{pkg}}"; \
  if [ -d "$pkg" ]; then \
    if ls "$pkg"/*.go >/dev/null 2>&1; then \
      go test "$pkg"; \
    else \
      go test "$pkg/..."; \
    fi; \
  else \
    go test "$pkg"; \
  fi

test-golden:
  @go test ./internal/tui -run 'Golden'

test-golden-update:
  @go test ./internal/tui -run 'Golden' -update

build:
  @go build -o ./kan ./cmd/kan

run:
  @go run ./cmd/kan

[private]
coverage:
  @tmp=$(mktemp); \
  go test ./... -cover | tee "$tmp"; \
  awk 'BEGIN {bad=0} /coverage:/ {cov=$5; gsub("%","",cov); if ((cov+0) < 70) {print "coverage below 70%:", $2, cov "%"; bad=1}} END {exit bad}' "$tmp"; \
  rm -f "$tmp"

ci: verify-sources fmt test coverage build
