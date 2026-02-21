# Kan TUI Plan + Worklog

Created: 2026-02-21  
Status: In progress (Phase 0-3 complete, Phase 4 underway)  
Execution gate: Active implementation

## 1) Product Goal
Build a polished, Charm-style Kanban TUI with local SQLite persistence, multiple projects, customizable columns, strong keyboard support (`vim` + arrows), mouse support, and cross-platform releases (macOS/Linux/Windows).

## 2) Confirmed Decisions
- UI stack: Bubble Tea v2 + Bubbles v2 + Lip Gloss v2.
- Start point: use `bubbletea-app-template` as architecture seed, then expand.
- Persistence: SQLite, single-user local-first.
- Projects: multiple projects are core.
- Columns: customizable per project.
- Task data: rich fields supported, but fields can be optional/configurable.
- MVP includes search/filter.
- MVP includes import/export.
- Delete/archive behavior: user-selectable.
- Input modes: `vim` keys + arrow keys, plus mouse support.
- CI target OS: macOS, Linux, Windows (manual dev testing on macOS).
- Quality requirement: Teatest included for TUI behavior tests.
- Tooling requirement: use `just` recipes as the primary dev/CI interface.

## 3) Open Decisions To Lock Before Coding
No open decisions. Locked defaults:
- SQLite driver: `modernc.org/sqlite` (no CGO).
- Delete behavior:
  - default mode is `archive`.
  - archived tasks are easy to restore.
  - hard delete is explicit and available from task actions/CLI.
- Task field defaults:
  - required: `title`.
  - visible by default on board cards: `priority`, `due_date`, `labels`.
  - `description` optional and edited in full task editor.
  - system/internal fields (`id`, `position`, timestamps, archive metadata) not shown by default.

## 4) Technical Architecture

## 4.1 Project Layout
```text
cmd/kan/main.go
internal/app/                # app bootstrapping + runtime wiring
internal/tui/                # root model, screens, components, keymaps, styles
internal/domain/             # entities + business rules
internal/adapters/storage/sqlite/ # schema and repository implementation
internal/config/             # config load/merge/validation
internal/cli/                # non-TUI commands (project/import/export/doctor)
internal/platform/           # user config/data path handling per OS
migrations/                  # embedded SQL migrations
testdata/                    # fixtures/golden outputs
.github/workflows/           # CI/release
Justfile
```

## 4.2 Data Model (Initial)
- `projects`
  - `id`, `slug`, `name`, `description`, `created_at`, `updated_at`, `archived_at`
- `columns`
  - `id`, `project_id`, `name`, `wip_limit`, `position`, `created_at`, `updated_at`, `archived_at`
- `tasks`
  - `id`, `project_id`, `column_id`, `position`, `title`, `description`, `priority`, `due_at`, `status`, `created_at`, `updated_at`, `archived_at`
- `labels`
  - `id`, `project_id`, `name`, `color`, `created_at`
- `task_labels`
  - `task_id`, `label_id`
- `app_meta`
  - schema version and migration metadata (if needed beyond migration tool state)

## 4.3 Config + Paths
- Config file: TOML.
- Path resolution:
  - macOS: config in `~/Library/Application Support/kan/config.toml`, data in `~/Library/Application Support/kan/kan.db`.
  - Linux: config in `$XDG_CONFIG_HOME/kan/config.toml` (fallback `~/.config/kan/config.toml`), data in `$XDG_DATA_HOME/kan/kan.db` (fallback `~/.local/share/kan/kan.db`).
  - Windows: `%AppData%/kan/config.toml`, data in `%LocalAppData%/kan/kan.db`.
- Overrides:
  - flags: `--config`, `--db`
  - env: `KAN_CONFIG`, `KAN_DB_PATH`
  - precedence: flag > env > TOML > defaults

## 4.4 TUI UX Plan
- Screens:
  - Project Switcher
  - Board View
  - Task Editor Modal
  - Command Palette
  - Filter/Search Bar
  - Confirm dialogs
- Navigation:
  - `h/j/k/l` and arrow equivalents
  - `tab` / `shift+tab` focus cycling
  - `enter` open/edit
  - `n` new task, `p` project picker, `/` search
  - `[` and `]` or `H`/`L` for move across columns
  - `K`/`J` for reorder within column
- Mouse:
  - click to focus/select
  - scroll within lists
  - drag/move behavior staged (MVP: click/scroll; drag in phase 2 if stable)
- Visual quality:
  - centralized theme tokens (spacing, borders, accents, state colors)
  - consistent status/help bar
  - clean empty/loading/error states

## 5) Teatest + Bubble Tea v2 Strategy
Known risk: ecosystem path/version drift between latest `charm.land/*` v2 stack and upstream `x/exp/teatest/v2`.

Plan:
- Start with `teatest` integration from day 1.
- If upstream resolves cleanly, depend directly.
- If mismatch persists, pin and patch locally:
  - `third_party/teatest_v2` with minimal import-path compatibility fix.
  - `go.mod replace` for deterministic CI.
- Track removal task to drop patch when upstream aligns.

Acceptance:
- TUI smoke tests run on all CI OS jobs.
- Golden output tests for core views.
- Input event tests for keyboard navigation and project switching.

## 6) `just`-First Developer Workflow
Planned recipes:
- `just bootstrap` -> install tools and sync modules
- `just fmt` -> format code
- `just lint` -> run linters
- `just test` -> all tests
- `just test-unit` -> domain/config/storage unit tests
- `just test-tui` -> teatest/golden TUI tests
- `just test-int` -> sqlite integration tests
- `just build` -> local build
- `just run` -> run app
- `just ci` -> deterministic CI entrypoint (`fmt-check + lint + tests + build`)
- `just release-check` -> local goreleaser snapshot check

Rule: GitHub Actions should call `just` recipes, not duplicate shell logic per workflow.

## 7) CI + Release Plan
- `ci.yml` on push/PR:
  - matrix: `ubuntu-latest`, `macos-latest`, `windows-latest`
  - steps: checkout, setup-go, cache, `just ci`
- `lint.yml` (optional split) or include lint in `ci.yml`
- `release.yml` on tags (`v*.*.*`):
  - goreleaser builds for all targets
  - publish checksums + archives
  - Homebrew formula/tap publish flow
- `snapshot.yml` on `main` (optional):
  - goreleaser snapshot artifacts for early verification

## 8) Phased Delivery Plan

## Phase 0: Foundation
- Bootstrap repo from template shape.
- Upgrade/lock v2 stack.
- Add `Justfile`, basic CI skeleton, project layout.
- Add config path and DB path resolver.
- Deliverable: app starts, key handling works, CI green.

## Phase 1: Persistence Core
- Add migrations and sqlite repository layer.
- Implement projects, columns, tasks CRUD.
- Add seed/default project on first run.
- Deliverable: data survives restarts.

## Phase 2: Board UX MVP
- Project switcher and board rendering.
- Create/edit/move/reorder tasks.
- Search/filter in board.
- Basic mouse support (select/scroll).
- Deliverable: complete keyboard-driven Kanban workflow.

## Phase 3: Config + Import/Export + Delete Policies
- Configurable fields and behavior via TOML.
- User-selectable delete/archive mode.
- JSON import/export commands.
- Deliverable: customizable and portable local workflow.

## Phase 4: Quality + Release
- Expand teatest coverage + golden tests.
- Cross-platform CI hardening.
- Goreleaser + Homebrew packaging.
- Deliverable: first public tagged release.

## 9) Definition of Done (MVP)
- Multi-project board is fully usable from TUI.
- SQLite persistence is reliable and migration-backed.
- Search/filter and import/export are present.
- `vim` + arrow keys both work.
- Mouse select/scroll works.
- CI passes on macOS/Linux/Windows.
- TUI behavior covered by teatest.
- `just ci` is the single local/CI quality gate.

## 10) Worklog
- [x] 2026-02-21: Planning confirmed with v2 direction.
- [x] 2026-02-21: Decision recorded to pause coding until explicit approval.
- [x] 2026-02-21: Defaults finalized: no-CGO sqlite, archive-first delete mode, default task field visibility.
- [x] 2026-02-21: Bootstrapped Go module and hexagonal package structure (`domain`, `app`, `adapters`, `tui`, `config`, `platform`).
- [x] 2026-02-21: Implemented app/service core with archive/hard-delete behavior and default project bootstrap.
- [x] 2026-02-21: Implemented SQLite adapter (modernc, no CGO) with schema creation and CRUD for projects/columns/tasks.
- [x] 2026-02-21: Implemented TOML config defaults + validation and platform path resolution logic.
- [x] 2026-02-21: Implemented Bubble Tea v2 TUI board model with keyboard handling.
- [x] 2026-02-21: Added Charm teatest v2 smoke testing via local compatibility patch (`third_party/teatest_v2` + `go.mod replace`).
- [x] 2026-02-21: Added `Justfile` recipes and cross-platform GitHub Actions CI + release workflow scaffold.
- [x] 2026-02-21: Enforced and verified per-package coverage floor >70% using `just ci`.
- [x] 2026-02-21: Completed first Phase 2 interaction slice:
  - quick-add (`n`)
  - task selection (`j`/`k`, mouse wheel)
  - move between columns (`[` / `]`)
  - rename (`e`)
  - archive/hard-delete/restore (`d` / `D` / `u`)
  - search mode (`/`)
  - project cycling (`p` / `P`)
  - archived visibility toggle (`t`)
  - mouse click column selection
- [x] 2026-02-21: Completed Phase 2 polish pass:
  - interactive project picker mode (`p`/`P`, `j`/`k`, `enter`, `esc`, mouse wheel/click)
  - full-field task editor mode (`e`) with structured edit input:
    `title | description | priority | due | labels`
  - app-layer `UpdateTask` use case added for full task mutation through ports
  - improved mouse behavior while modal modes are active (picker-aware wheel/click handling)
- [x] 2026-02-21: Continued Phase 2 refinement:
  - switched to Bubbles v2 key/help model with bottom-anchored help bar (`?` toggles full help)
  - added modal-style overlays for input/picker flows to improve in-TUI editing ergonomics
  - added selected-task detail panel for better board readability
- [x] 2026-02-21: Added visual QA workflow with Charm VHS:
  - `just vhs-board`, `just vhs-workflow`, `just vhs`
  - `vhs/*.tape` recordings for quick interaction smoke previews
- [x] 2026-02-21: Added repo-level `AGENTS.md` with architecture/testing/workflow guardrails.
- [x] 2026-02-21: Completed Phase 3 import/export + configurable UX behavior:
  - new CLI subcommands:
    - `kan export --out <file> [--include-archived]`
    - `kan import --in <file>`
  - app-layer snapshot export/import use cases with validation and upsert behavior
  - environment overrides wired: `KAN_CONFIG`, `KAN_DB_PATH` (flag > env > TOML/defaults)
  - TUI rendering now honors `task_fields` config toggles for card/details metadata
  - TUI delete key now uses configurable default delete mode, with explicit archive (`a`) and hard delete (`D`)
  - added `just export` / `just import` recipes
- [x] 2026-02-21: Started Phase 4 quality pass:
  - expanded teatest coverage for interaction states (full-help toggle and project picker flow)
  - improved TUI package coverage from 73.7% to 77.1%
- [ ] Next: Phase 4 quality/release hardening (teatest/golden depth, CI enhancements, release polish).
