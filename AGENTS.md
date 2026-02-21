# Repository Guidelines

This file defines instructions for coding agents working in this repository. It is not runtime behavior for `kan`.

You are a senior Go dev. YOU ALWAYS:

- ALWAYS use Context7 for library and API documentation before writing any code.
- ALWAYS re-run Context7 after any test failure or runtime error before making the next edit.
- Write idiomatic Go docstrings and comments for all non-obvious behavior in production and test code, including behavior blocks in `*_test.go`.
- Review `Justfile` at startup and use its recipes as the source of truth for local automation.
- Run tests/checks through `just` recipes only; do not run `go test` directly from the agent.
- When you touch Go code, finish by running `just ci` unless the user explicitly approves a narrower suite.
- Add package-scoped `Justfile` recipes when needed for fast iteration, then still finish with `just ci`.
- If dependency updates need network access, ask the user to run `go get` and module update commands in their own shell.
- Never use dependency-fetch bypasses (for example `GOPROXY=direct`, `GOSUMDB=off`, or checksum bypass flags).
- Never delete files or directories without explicit user approval.
- Keep the active execution/work log in `PLAN.md`. Use `worklogs/` only when the user explicitly asks for split logs.

## Project Structure

- `cmd/kan`: CLI/TUI entrypoint.
- `internal/domain`: core entities and invariants.
- `internal/app`: application services and use-cases (ports-first, hexagonal core).
- `internal/adapters/storage/sqlite`: SQLite persistence adapter.
- `internal/config`: TOML loading, defaults, validation.
- `internal/platform`: OS-specific config/data/db path resolution.
- `internal/tui`: Bubble Tea/Bubbles/Lip Gloss presentation layer.
- `.artifacts/`: generated local outputs (exports, temporary build outputs).
- `PLAN.md`: active roadmap and execution/work log.

## Build and Run

- `just run`: run app from source (`go run ./cmd/kan`).
- `just build`: build local binary `./kan`.
- `just fmt`: format Go files.
- `just test`, `just test-pkg <pkg>`: test entrypoints.
- `just test-golden`, `just test-golden-update`: golden fixture validation/update.
- `just ci`: canonical local gate (source verification, format, tests, coverage floor, build).

## Worktrees

- Worktrees are optional but supported.
- If a worktree path is requested by the user, always `cd` into that exact path before editing, testing, or committing.
- Do not hard-code worktree names.
- Do not run completion/cleanup git actions (push, merge, rebase, worktree removal, branch deletion) without explicit user approval in the current conversation.

## Worklogs

- Use `PLAN.md` as the live execution ledger.
- Keep updates step-by-step while work is in progress. At minimum log:
  - current objective/plan,
  - each command/test run and outcome,
  - each file edit and why,
  - each failure and remediation,
  - current status and next step.

## Tech Stack

- Go 1.26+
- Bubble Tea v2, Bubbles v2, Lip Gloss v2
- SQLite (`modernc.org/sqlite`, no CGO)
- TOML config (`github.com/pelletier/go-toml/v2`)

## Core Coding Paradigms

- Hexagonal architecture (ports/adapters), interface-first boundaries, dependency inversion.
- Ship small, testable increments; prioritize maintainability and pragmatic MVP progress.
- TDD-first where practical: tests before implementation for new behavior.
- Preserve Go idioms: clear naming, wrapped errors (`fmt.Errorf("...: %w", err)`), import grouping stdlib -> third-party -> local.
- Keep TUI mode transitions explicit and test-covered.

## Testing Guidelines

- Tests are co-located as `*_test.go`.
- Prefer table-driven tests and behavior-oriented assertions.
- Run package-focused loops with `just test-pkg <pkg>` during implementation.
- For substantial TUI changes, update or add tea-driven tests and golden fixtures.
- Coverage below 70% is a hard failure.
- Build/test execution must go through `just` recipes only.

## UX Guardrails

- Help bar stays bottom-anchored in normal mode.
- Expanded help is a centered modal overlay (Fang-inspired style).
- Add/edit/info/project/search overlays are centered and do not push board content.
- Support both vim keys and arrow keys.
- Mouse wheel/click behavior must continue to function.
- Keep modal copy concise and avoid redundant field explanations.

## Release and Security

- Keep release/Homebrew work in roadmap unless explicitly requested for execution.
- Keep secrets out of config files committed to the repository.
- Prefer environment overrides for machine-local sensitive settings.
