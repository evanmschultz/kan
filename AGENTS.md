# AGENTS.md

## Mission
Build and maintain `kan` as a polished, local-first Kanban TUI with Charm v2 tooling, SQLite persistence, and cross-platform CI.

## Locked Technical Decisions
- UI stack: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`.
- Persistence: SQLite via `modernc.org/sqlite` (no CGO).
- Architecture: hexagonal boundaries (`domain` -> `app` -> adapters/TUI).
- Quality gate: every package must stay above 70% test coverage.

## Architecture Rules
- Keep domain types/rules in `internal/domain` with no framework dependencies.
- Put use-case orchestration in `internal/app` behind port interfaces.
- Keep SQLite details in `internal/adapters/storage/sqlite` only.
- Keep TUI state/render/input handling in `internal/tui`.
- Avoid leaking adapter concerns into `domain` or `app`.

## Workflow Rules
- Prefer `just` recipes as the entrypoint:
  - `just fmt`
  - `just test`
  - `just test-tui`
  - `just ci`
  - `just vhs`
- For local visual QA, generate GIF previews with VHS before handing off major TUI changes.

## Testing + TDD
- Add/adjust tests with each behavior change; default to TDD for new logic.
- Keep tea-driven behavior tests in `internal/tui` (including teatest smoke).
- Run `just ci` before considering a slice done.

## UX Guardrails
- Help keys should render in a bottom help bar.
- Support both vim keys and arrow keys.
- Preserve mouse scroll/click behavior in board and picker flows.
- Favor modal-style interactions for text entry and picker modes.

## Planning Discipline
- `PLAN.md` is the source of truth for phased delivery and worklog.
- When a slice is completed, update `PLAN.md` worklog in the same change.
