# Repository Guidelines

You are a senior Go dev. YOU ALWAYS:

- ALWAYS automatically use Context7 for code generation and library documentation before writing a single line of code.
- ALWAYS re-run Context7 before any code edit after a test failure or runtime error.
- Write idiomatic Go comments and docstrings for every code block, including all production and test code (`*_test.go`), with no uncommented behavior blocks.
- Review the `Justfile` at startup to align on recipes, environment expectations, and cleanup patterns.
- Run `just check-llm` whenever you touch Go code (unless the user explicitly approves a narrower suite).
- Add or use package-scoped Justfile test recipes for fast iteration, then run `just check-llm` before final confirmation.
- Run tests only via `just` recipes. Do not run `go test` directly from the agent.
- If a needed test command does not exist in `Justfile`, add a recipe first (or ask the user which recipe to use), then run that recipe.
- If dependency updates require network access, ask the user to run `go get` and related module commands in their own shell.
- Never use dependency-fetch sandbox workarounds (for example `GOPROXY=direct`, `GOSUMDB=off`, or checksum bypass flags).
- Never delete files or directories without explicit user approval.
- Keep all active worklogs/spec notes in `worklogs/` at repository root.
- Treat the active worklog as both execution plan and progress ledger; update it continuously while you work.

## Project Structure

- `cmd/kan`: CLI/TUI entrypoint.
- `internal/domain`: core entities and invariants.
- `internal/app`: application services and use-cases (ports-first, hexagonal core).
- `internal/adapters/storage/sqlite`: SQLite persistence adapter.
- `internal/config`: TOML loading, defaults, validation.
- `internal/platform`: OS-specific config/data/db path resolution.
- `internal/tui`: Bubble Tea/Bubbles/Lip Gloss presentation layer.
- `vhs/`: tracked VHS tapes used for visual regression checks.
- `.artifacts/`: generated local outputs (VHS gifs, exports, temporary build outputs).
- `PLAN.md`: active roadmap + integrated worklog.

## Build and Run

- `just run`: run app from source (`go run ./cmd/kan`).
- `just run-dev`: run app with dev path isolation (`--dev`).
- `just build`: build local binary `./kan`.
- `just run-bin`: build and run local binary.
- `just paths`: print resolved config/data/db paths.
- `just fmt`: format Go files.
- `just test`, `just test-unit`, `just test-tui`, `just test-pkg <pkg>`: test entrypoints.
- `just test-golden`, `just test-golden-update`: golden fixture validation/update.
- `just vhs`, `just vhs-board`, `just vhs-workflow`: visual snapshots.
- `just ci`: canonical local gate (format, tests, coverage floor, build-all).
- `just check-llm`: alias to strongest local gate in this repo.

## Worktrees

- Worktrees are optional but supported.
- If a worktree path is requested by the user, always `cd` into that exact path before editing, testing, or committing.
- Do not hard-code worktree names.
- Do not run completion/cleanup git actions (push, merge, rebase, worktree removal, branch deletion) without explicit user approval in the current conversation.

## Worklogs

- Use `PLAN.md` and/or files in `worklogs/` as the live execution ledger.
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
- Use `just vhs` for visual UX verification when layout/modal/help behavior changes.
- VHS artifacts are for visual inspection only; they are not source-of-truth logic.
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
