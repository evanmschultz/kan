# Pre-MCP Execution Waves (Consensus Lock)

Date: 2026-02-23
Status: Locked for execution planning
Scope: Pre-MCP only (local app, no MCP/HTTP transport)

## 1) Objective

Implement the remaining product-intent UX/data capabilities before MCP work starts:

- Always-open project picker on app launch with first-class project creation in that picker.
- First-run configuration bootstrap for user identity + global root-search base path(s).
- Description + comment capabilities for project/phase/task/subtask-level coordination.
- Full-screen markdown rendering using `github.com/charmbracelet/glamour`.
- Ownership-attributed comments for solo user+agent and team scenarios.
- Parallelized execution plan with lock scopes and acceptance gates.

## 2) Locked Decisions

### 2.1 Launch and Startup Flow

- On every launch, open in `Project Picker` mode first.
- Project picker must always include an explicit `New Project` action.
- Selecting `New Project` opens project creation immediately without leaving picker flow.
- If no projects exist, picker still opens and foregrounds the `New Project` action.

### 2.2 First-Run Bootstrap (before project picker)

On first launch (or missing required identity/root-search config), run a short bootstrap wizard before the picker:

1. Identity setup:
- collect display name used for authored comments.
- default actor role is `user`.

2. Root-search setup:
- collect one or more global root-search base paths (for example `~/Documents`, `~/Documents/code`).
- save to `config.toml`.
- enable an easy TUI picker for editing these later.

3. Continue to project picker.

### 2.3 Root Path UX

- Keep project-specific `root_path` mapping support.
- Add global root-search base paths to speed/ease discovery in path pickers.
- Path pickers must support fuzzy filtering and intuitive navigation.

### 2.4 Description and Comments

- Projects, phases, tasks, subtasks, and equivalent work-item variants must support:
- markdown description text (stored as raw string in DB).
- markdown comments (stored as raw string in DB).

- Rendering:
- all markdown is rendered in full-screen views using `glamour`.
- descriptions and comments are human-authored markdown source in storage; rendering is view-time.

### 2.5 Comment Ownership

Every comment must include ownership metadata:

- actor category: `user | agent | system`.
- author display name (from config identity, overridable where needed).
- timestamps.

This supports solo user+agent collaboration and team multi-author collaboration.

### 2.6 Full-Screen Description + Thread View

- Add a full-screen markdown view per entity (project/phase/task/subtask/etc.).
- View includes:
- rendered description section (scrollable).
- rendered comments thread (scrollable).
- input field to submit a new comment.
- input submission appends new comment with ownership metadata.

## 3) Technical Contract

## 3.1 Config Additions

Add new config sections:

- `[identity]`
- `display_name = "..."`
- `default_actor_type = "user"` (`user|agent|system`)

- `[paths]`
- `search_roots = ["..."]` (global roots for fuzzy pickers)

Notes:

- Existing project root mappings remain under `[project_roots]`.
- First-run wizard writes defaults when missing.

## 3.2 Data Model Additions

Description fields:

- Reuse existing description string fields where present.
- Ensure all required entity types expose description consistently in app/domain APIs.

Comments:

- Add persisted comments table (or equivalent adapter-backed entity) keyed by target type + target id.
- Minimum columns:
- `id`, `project_id`, `target_type`, `target_id`, `body_markdown`, `actor_type`, `author_name`, `created_at`, `updated_at`.

- Add list/create comment operations in app service boundary.

## 3.3 Glamour Rendering Contract

Context7 references confirm:

- reusable renderer instances via `glamour.NewTermRenderer(...)`.
- style selection via `WithAutoStyle`/`WithStandardStyle` and configurable word wrap.
- optional custom JSON styles if needed later.

Implementation direction:

- create one renderer component/service in TUI layer.
- cache/reuse renderer instance; rebuild on width/style changes when required.
- render markdown source safely into ANSI for full-screen views.

## 4) Parallel Wave Plan

## Wave 0: Architecture + Contracts Lock

Goal:

- finalize config keys, comment schema, and TUI mode contracts before implementation.

Parallel lanes:

- `W0-A` Config/domain contract draft.
- `W0-B` Storage migration plan + repository API additions.
- `W0-C` TUI mode/key contract draft for picker/bootstrap/markdown thread view.

Acceptance:

- contracts documented and cross-checked with existing architecture boundaries.

## Wave 1: Launch Flow + Picker Rewrite

Goal:

- app always starts in project picker and supports in-picker project creation.

Parallel lanes:

- `W1-A` Startup mode routing and picker-first boot.
- `W1-B` Picker UX (fuzzy filter, create action row, deterministic focus behavior).
- `W1-C` Regression tests for boot/picker/new-project path.

Lock scope hints:

- `internal/tui/model.go`
- `internal/tui/model_test.go`
- `cmd/kan/main.go`, `cmd/kan/main_test.go` (only if startup wiring requires)

Acceptance:

- every app launch opens picker.
- picker always exposes `New Project`.
- create-from-picker flow works on empty and non-empty workspaces.

## Wave 2: First-Run Config Bootstrap + Root Search Base Paths

Goal:

- first-run wizard collects identity + global search roots before picker.

Parallel lanes:

- `W2-A` Config schema and load/normalize/validate updates.
- `W2-B` Bootstrap TUI flow and persistence wiring.
- `W2-C` Path picker integration using global `paths.search_roots` with fuzzy UX.

Lock scope hints:

- `internal/config/config.go`, `internal/config/config_test.go`
- `cmd/kan/main.go`, `cmd/kan/main_test.go`
- `internal/tui/model.go`, `internal/tui/model_test.go`
- `config.example.toml`

Acceptance:

- missing identity/search_roots triggers bootstrap once.
- values persist to config and are used by TUI pickers.
- user can edit these from TUI after first run.

## Wave 3: Comments Domain + Persistence

Goal:

- add ownership-attributed markdown comments to all target entities.

Parallel lanes:

- `W3-A` Domain/app service interfaces for comments.
- `W3-B` SQLite migration + repository implementation.
- `W3-C` App service tests for create/list comments by target.

Lock scope hints:

- `internal/domain/*`
- `internal/app/service.go`, `internal/app/service_test.go`
- `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`

Acceptance:

- create/list comments works for project/phase/task/subtask targets.
- ownership metadata always persisted.

## Wave 4: Full-Screen Markdown Description + Comment Thread View

Goal:

- implement full-screen markdown rendering and comment composition flow.

Parallel lanes:

- `W4-A` TUI full-screen mode(s) for description + comments.
- `W4-B` Glamour renderer adapter integration and width/style handling.
- `W4-C` Input + submit comment flow + ownership badge rendering.

Lock scope hints:

- `internal/tui/model.go`, `internal/tui/model_test.go`
- optional `internal/tui/markdown.go` renderer helper

Acceptance:

- full-screen view opens from project/phase/task/subtask contexts.
- description and comments render as markdown via glamour.
- comment input appends persisted, ownership-attributed comment.

## Wave 5: Entity Coverage + Consistency Sweep

Goal:

- ensure the same description/comment behavior across all supported entity levels.

Parallel lanes:

- `W5-A` Project-level description/thread wiring.
- `W5-B` Phase-level description/thread wiring.
- `W5-C` Task/subtask-level description/thread wiring and navigation consistency.

Acceptance:

- uniform UX and ownership rendering across entity levels.

## Wave 6: Verification, Worksheet Refresh, and Gate

Goal:

- update tests/docs/manual worksheet and close wave with repo gate.

Parallel lanes:

- `W6-A` Package tests and golden updates.
- `W6-B` Manual worksheet update for new flows.
- `W6-C` Docs cleanup and help overlay updates.

Acceptance:

- package lanes pass via `just test-pkg <pkg>`.
- integrator runs `just ci` successfully.
- worksheet has explicit anchors for bootstrap, picker-first startup, markdown thread views, ownership labels, and comment submission.

## 5) Subagent Parallel Contract (Execution)

Each worker lane must include:

- lane id and single acceptance objective.
- lock scope file globs and explicit out-of-scope paths.
- Context7 checkpoint before first code edit.
- Context7 re-consult after every failed test/runtime error.
- package-scoped checks using only `just test-pkg <pkg>`.
- Go doc comments and inline comments for non-obvious logic.
- handoff evidence:
- files changed and why.
- commands run with pass/fail.
- acceptance checklist pass/fail.
- architecture boundary compliance note.
- unresolved risks/blockers.

Integrator responsibilities:

- merge verified lane outputs.
- keep single-writer `PLAN.md` worklog updates.
- run final `just ci` before marking wave complete.

## 6) Initial Suggested Lane Bundle

Safe first bundle for parallel kickoff:

- `Bundle-A`: W1-A + W1-B + W1-C (startup picker behavior).
- `Bundle-B`: W2-A + W2-C (config + search roots/path picker).
- `Bundle-C`: W3-A + W3-B + W3-C (comment persistence contract).

Then:

- W4 lanes begin after W3 contract lands.
- W5 and W6 follow as integration sweeps.

## 7) Risks and Mitigations

- Risk: TUI mode complexity/regressions from added full-screen views.
- Mitigation: isolate new modes, add table-driven mode/key routing tests, keep deterministic focus transitions.

- Risk: markdown rendering performance on large threads.
- Mitigation: reuse glamour renderer; lazy render and viewport/windowing for long comment timelines.

- Risk: schema churn during active feature work.
- Mitigation: land comment schema in one migration wave and avoid re-shaping table contract mid-wave.

- Risk: shortcut conflicts in text-input contexts.
- Mitigation: text-input-first key routing rule and regression tests for every modal with text fields.

## 8) Out of Scope (Still Locked)

- MCP/HTTP transport and external connector execution.
- external service synchronization logic.
- non-local backend migrations beyond current pre-MCP scope.

## 7) Current Execution Note (2026-02-23)

Implementation has started with subagent lanes and produced integrated changes for:
- config identity + search roots,
- startup bootstrap logic,
- picker-first launch behavior,
- comments domain/app/sqlite plumbing,
- initial thread-mode markdown rendering.

Current known gaps before closeout:
- startup bootstrap must move from terminal prompt UX into a first-run TUI modal flow consistent with existing overlays,
- search-root selection UX needs fuzzy, low-friction picker interactions,
- README needs an explicit update lane aligned with newly added `fang` dependency/usage direction,
- regression failures from the latest integrated TUI test run must be cleared and `just ci` must pass.

Execution is continuing in ordered checkpoints captured in `PLAN.md`.
