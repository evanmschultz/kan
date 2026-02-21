# Kan TUI Plan + Worklog

Created: 2026-02-21  
Status: In progress (Phase 0-6 active, MCP-oriented roadmap expansion added)  
Execution gate: Planning update only in this step (no code changes)

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
- Canonical lifecycle states are fixed and non-configurable for now:
  - `todo`, `progress`, `done`, `archived`.
- Completion contracts are required across all work-item levels (phase/task/subtask/etc):
  - transition into `progress` and `done` must evaluate contract criteria.
  - LLM-driven transitions must receive contract context and validation feedback.

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

## Phase 5: UX + Workflow Expansion (Current)
- Config system expansion:
  - richer app config (key overrides, search defaults, dev/prod paths)
  - tracked example config template and docs for installed/dev use
- Project lifecycle UX:
  - create/edit projects from TUI
  - project metadata support (description + timeline fields)
- Search + filtering:
  - cross-project search
  - state-aware filtering (`todo`, `progress`, `done`, `archived`)
- Help + onboarding:
  - richer branded help screen (Fang-inspired presentation)
  - first-run onboarding flow
- Board interaction upgrades:
  - command palette
  - quick action menu
  - multi-select + bulk actions
  - activity log panel
  - undo/redo basics
- Card/field UX upgrades:
  - priority picker (done)
  - due date picker modal
  - label suggestions / picker
- Planning features:
  - optional swimlane grouping modes
  - WIP limit visibility and warnings
- Deliverable: production-usable daily-driver UX with configurable workflow semantics.

## Phase 6: UX Remediation + Search/State Overhaul (Approved Next)
This phase is intentionally split into small execution chunks so we can ship incrementally and keep VHS + teatest feedback tight.

### Phase 6.1: Search Form UX and Focus Model
- Convert search to a single coherent modal with focusable controls in this exact order:
  - query -> states -> scope -> archived -> apply.
- Remove duplicated state/query labeling in the search modal (no duplicate prompt text and external labels).
- Keep `tab` / `shift+tab` as the primary focus movement; keep direct hotkeys as optional shortcuts.
- Ensure scope and archived are not hidden-only toggles; both must be keyboard-focusable controls.
- Clarify actions:
  - `apply search`: run current query + filters.
  - `clear query`: clear only query text.
  - `reset filters`: reset query + states + scope + archived back to configured defaults.
- Acceptance:
  - No repeated labels in search UI.
  - User can tab through every control including scope and archived.
  - `clear query` and `reset filters` have distinct behavior and help text.

### Phase 6.2: Canonical State Selector and Filter Model
- Replace free-text state search entry with a multi-select state selector.
- State options are canonical and fixed:
  - `todo`
  - `progress`
  - `done`.
- Include configured filter defaults from TOML as initial active selection (from canonical set only).
- Archived behavior:
  - keep explicit archived toggle in search controls.
  - avoid duplicate "archived" state rendering if already represented in filter text.
- Ensure lifecycle states use canonical enum semantics in storage/app/TUI.
- Acceptance:
  - Search modal never shows duplicate states.
  - State filtering is consistent across projects because lifecycle states are global and fixed.

### Phase 6.3: Command Palette Usability Upgrade
- Command palette should filter commands live as user types (fuzzy/substring acceptable).
- Enter executes highlighted command; full command text should not be required.
- Tab autocompletes top match.
- Show short descriptions and aliases for commands.
- Add search-context hints:
  - examples: `reset filters`, `clear query`, `search all projects`, `search current project`.
- Define explicit command semantics for `clear search`:
  - either remove it in favor of `clear query` + `reset filters`, or keep as alias with documented target behavior.
- Acceptance:
  - Typing narrows command list immediately.
  - Enter runs highlighted entry with no ambiguity.

### Phase 6.4: Board List Marker and Card Visual Consistency
- Restore kancli-style row semantics:
  - only focused task gets the left accent marker bar.
  - remove global marker from every row.
  - do not mix `>` with universal bars in a way that creates double indicators.
- Keep card text compact and readable with predictable truncation.
- Ensure focused row styling remains clear in both active and inactive columns.
- Acceptance:
  - Exactly one row marker per focused list (or none when empty).
  - Dense lists remain readable and visually stable.

### Phase 6.5: Confirmation Modals for Destructive/State-Changing Actions
- Add confirmation modal flows for:
  - archive task
  - hard delete task
  - soft delete/default delete action (if mapped separately)
  - optional restore confirmation (configurable)
- Provide clear copy including task title and action impact.
- Modal actions:
  - confirm
  - cancel
  - keyboard + mouse support.
- Add config gates:
  - `confirm.archive`
  - `confirm.delete`
  - `confirm.hard_delete`
  - `confirm.restore` (optional)
- Acceptance:
  - No destructive action executes without configured confirmation behavior.

### Phase 6.6: Modal Layering and Interaction Consistency
- Keep all primary modals (new/edit/info/help/search/confirm) centered in full viewport (X/Y center).
- Ensure background layout does not shift when modal opens.
- Keep `enter` mapped to task info (same behavior as `i`) in board mode.
- Info modal remains read-first; `e` from info enters edit modal for selected task.
- Remove repeated instructional footer text when it duplicates field labels already visible in the form.
- Acceptance:
  - Modal centering is consistent across viewport sizes.
  - Enter and `i` are functionally equivalent in board mode.

### Phase 6.7: Due DateTime, Validation, and Urgency Signals
- Keep current lightweight due picker approach (fast list shortcuts + direct typing).
- Do not adopt external calendar dependency in this phase.
- Support datetime input while keeping date-only compatibility:
  - date-only accepted
  - datetime accepted (user-typed for precision control).
- Warn in-form when parsed datetime is already in the past.
- Add urgency logic:
  - `OVERDUE` when `now > due_at`
  - `DUE SOON` within configured windows.
- Add board summary status row using this format:
  - `<overdue_count> overdue * <due_soon_count> due soon`
  - example: `3 overdue * 5 due soon`
- Acceptance:
  - User can set date-only or datetime.
  - Past-due warning appears before save.
  - Summary row reflects current filtered task set.

### Phase 6.8: Due Window Config and TOML Surface
- Add/confirm TOML keys for urgency thresholds and display:
  - `ui.due_soon_windows = ["24h", "1h"]` (user configurable)
  - `ui.show_due_summary = true`
- Keep defaults sane, and allow full override.
- Ensure parse errors report exact key and invalid value.
- Acceptance:
  - Users can customize due-soon thresholds without code changes.
  - Invalid durations fail validation with clear error messages.

### Phase 6.9: Lifecycle Simplification and Column Mapping Alignment
- Remove configurable lifecycle state taxonomies for this phase.
- Keep canonical lifecycle states:
  - `todo`
  - `progress`
  - `done`
  - `archived`.
- Map board columns/sections to canonical lifecycle states deterministically.
- If display labels vary by project, treat them as presentation aliases only (not data-state identities).
- Acceptance:
  - No project-specific state definitions can diverge lifecycle logic.
  - Search, board rendering, and transitions all operate on one canonical state model.

### Phase 6.10: Help + Command Discoverability Pass
- Keep Fang-style help modal but update command sections to reflect new search commands and confirmations.
- Ensure help explains:
  - info modal (`i`/`enter`)
  - edit handoff (`e`)
  - search apply vs clear query vs reset filters
  - destructive action confirmations.
- Add a short "search quick guide" block in help.
- Acceptance:
  - Help content matches actual runtime bindings and command semantics.

### Phase 6.11: Test and VHS Coverage Expansion for New UX Contracts
- Add/update teatest coverage for:
  - search focus traversal including scope/archived controls
  - state multi-select filtering
  - command palette live filtering and enter/Tab behavior
  - destructive action confirm modal flow
  - due datetime parsing + past warning + due summary rendering
  - list marker behavior (only focused row bar).
- Add/update VHS scripts for before/after visual validation of these flows.
- Acceptance:
  - CI coverage floors remain above package thresholds.
  - VHS artifacts show expected modal centering and list marker behavior.

### Phase 6.12: Execution Order and Checkpoints
- 6.1 + 6.2 first (search UX + state model) because they unblock filter correctness.
- 6.3 next (command palette discoverability).
- 6.4 + 6.5 + 6.6 as a UI consistency batch.
- 6.7 + 6.8 + 6.9 as due/config/state semantics batch.
- 6.10 + 6.11 finalize docs/test/contracts.
- 6.12 checkpoint review:
  - confirm all search semantics
  - confirm no duplicate state rendering
  - confirm canonical fixed lifecycle states are enforced end-to-end
  - confirm due summary output:
    - `3 overdue * 5 due soon` format
  - confirm no unresolved UX regressions before returning to remaining stretch items (multi-select, activity log, swimlanes, undo/redo).

## Phase 7: Rich Planning Data Model (MCP-Ready Local Foundation)
Goal: evolve from a simple kanban board into a planning/coordination system that can later back an MCP HTTP service for human + agent collaboration.

### Phase 7.1: Work Item Schema Evolution
- Move toward a generic hierarchical `work_items` core (instead of task-only mental model):
  - `id`, `project_id`, `parent_id`, `kind`, `title`, `summary`, `description`, `lifecycle_state`, `priority`, `position`.
  - lifecycle fields: `created_at`, `updated_at`, `started_at`, `completed_at`, `archived_at`, `canceled_at`.
  - ownership/attribution fields: `created_by_actor`, `updated_by_actor`, `updated_by_type` (`user|agent|system`).
- `lifecycle_state` is a fixed enum:
  - `todo`
  - `progress`
  - `done`
  - `archived`.
- `kind` should be configurable at project level (`phase`, `task`, `subtask`, `milestone`, `note`, `decision`, etc.).
- Keep compatibility adapters so current board/task flows keep working while data model expands.

### Phase 7.2: Rich Task/Work Item Fields for LLM + Dev Context
- Add fields designed for execution context and handoff quality:
  - `objective`
  - `implementation_notes_user`
  - `implementation_notes_agent`
  - `acceptance_criteria`
  - `definition_of_done`
  - `validation_plan`
  - `blocked_reason`
  - `risk_notes`
  - `command_snippets`
  - `expected_outputs`
  - `decision_log`
  - `related_items` (cross-links)
  - `transition_notes` (why it moved states)
- Add typed "context blocks" with importance:
  - `context_type`: `note|constraint|decision|reference|warning|runbook`.
  - `importance`: `low|normal|high|critical`.
- Goal: every work item can carry enough context to be safely resumed by an LLM agent.

### Phase 7.3: Resources and File References
- Add first-class `resource_refs` associated to project/phase/task:
  - `resource_type`: `local_file|local_dir|url|doc|ticket|snippet`.
  - `location`: path/URL.
  - `path_mode`: `relative|absolute`.
  - `base_alias`: project root alias for relative paths.
  - `title`, `notes`, `tags`, `last_verified_at`.
- Add file picker UX for local path references.
- Add "attach related paths" flow:
  - single file
  - directory
  - multi-select where feasible.
- Preserve both local portability and future remote portability by normalizing path refs to project-root-relative when possible.

### Phase 7.4: Project Root Path Strategy
- Primary recommendation:
  - keep machine-specific root paths in TOML config, keyed by stable project slug/ID.
  - store only stable project identity and relative references in DB.
- Proposed config shape:
  - `project_roots.<project_slug> = "/abs/path/to/project"`
  - optional multiple roots/aliases for monorepos.
- Benefits:
  - easier sharing/export of DB snapshots across machines.
  - less path churn in persisted data.
- Open fallback:
  - optionally persist last-known root path in DB as hint only (never source of truth).

### Phase 7.5: Configurable Label System (Project and Subscope Aware)
- Introduce label sets with scoped inheritance:
  - project defaults
  - phase-level extensions/overrides
  - optional item-level ad hoc labels.
- Label metadata:
  - name, color, icon, description, scope, suggested states/kinds.
- Add configurable label picker behavior:
  - suggested labels by scope/state/kind
  - recent labels
  - required labels (optional policy).

### Phase 7.6: Completion Contracts and Transition Guards (All Levels)
- Add first-class completion contract fields to every work item kind:
  - `start_criteria` (what must be true to move `todo -> progress`)
  - `completion_criteria` (what must be true to move `progress -> done`)
  - `completion_checklist` (structured checklist items with status)
  - `completion_evidence` (links, notes, command outputs, artifacts)
  - `completion_notes` (final human/agent summary)
  - `completion_policy` (per-kind policy knobs, including child requirements).
- Contract inheritance and override:
  - project-level default contracts by `kind`
  - phase/item-level overrides
  - explicit inheritance markers so behavior is deterministic.
- Transition guard behavior:
  - on `todo -> progress`, evaluate `start_criteria` and return unmet items.
  - on `progress -> done`, evaluate `completion_criteria`, checklist, and evidence requirements.
  - for parent items, enforce child completion policy (default: required children must be `done`).
- LLM-specific requirement:
  - when LLM attempts state transitions, context response must include active contract + unmet checks.
  - reject transition with structured reason when contract fails.

### Phase 7 Acceptance
- Rich context fields and resources are storable/queryable without breaking existing flows.
- Relative path references resolve through TOML project roots.
- Label scopes work at least at project + phase levels.
- Completion contracts are available for every item kind and validated on transitions.
- LLM transition attempts receive explicit contract context and validation results.

## Phase 8: Nested Work Graph and Configurable Hierarchy
Goal: support deep planning structures beyond flat tasks, controlled by user/project configuration.

### Phase 8.1: Hierarchy Model
- Support parent/child nesting for all work item kinds.
- Configurable naming for hierarchy levels:
  - example: `Project -> Phase -> Task -> Subtask`
  - example: `Project -> Milestone -> Story -> Task`.
- Configurable maximum depth per project.
- Preserve ordering at each sibling level.

### Phase 8.2: Board and Navigation Behavior for Nested Structures
- Define board projection modes:
  - phase board
  - task board within selected phase
  - flattened "current execution queue" view.
- Add breadcrumb context in UI for nested location.
- Add quick jump between parent, siblings, and children.

### Phase 8.3: Rollups and Dependency Basics
- Roll up progress and urgency from children to parents.
- Track dependency links:
  - `depends_on`
  - `blocked_by`.
- Surface blocked state in board/search results.

### Phase 8.4: Nested Completion Semantics
- Enforce completion behavior at every nesting level:
  - child-level contracts gate child transitions.
  - parent-level contracts can require specific child kinds or counts to be complete.
- Parent state progression defaults:
  - `todo -> progress` allowed when parent `start_criteria` passes (and optional child-start policy passes).
  - `progress -> done` requires parent contract + required child completion policy pass.
- Provide rollup diagnostics:
  - unmet parent criteria
  - unmet child criteria
  - blocked dependencies preventing completion.

### Phase 8 Acceptance
- Users can create and manage at least 3-level nested structures.
- Views remain usable with keyboard/mouse navigation.
- Parent progress reflects child state changes.
- Parent completion status is contract-driven and deterministic across nested levels.

## Phase 9: Memory Nodes and Operating Rules (Roadmap; No MCP/HTTP Yet)
Goal: capture persistent "remember this" context for dev + agent behavior at project/phase/item scopes.

### Phase 9.1: Memory Node Model
- Add `memory_nodes` with scope:
  - `project`
  - `phase`
  - `item` (task/subtask/decision/etc).
- Memory node fields:
  - `title`
  - `body`
  - `memory_type` (`rule|preference|warning|runbook|constraint|environment`)
  - `priority`
  - `active`
  - `created_by_actor`, `updated_by_actor`.

### Phase 9.2: Memory Resolution Rules
- Memory composition when loading a scope should merge:
  - project-level active memory
  - phase-level active memory
  - item-level active memory.
- Add deterministic ordering (priority + recency + scope proximity).
- Add conflict notes for contradictory memory entries.

### Phase 9.3: Documentation/Policy Preparation (for later MCP use)
- Add explicit roadmap doc requirements:
  - MCP instructions should require agent to read active memory nodes before work.
  - AGENTS policy should be generated/updated from active memory nodes where appropriate.
  - agent response templates should include "memory considered" evidence.
- Keep this as roadmap/spec only until MCP transport exists.

### Phase 9 Acceptance
- Memory nodes can be authored and retrieved by scope.
- Clear merge semantics exist for project/phase/item contexts.
- Roadmap docs define how memory should influence future agent execution policy.

## Phase 10: Actor-Aware Change Tracking and Agent Delta Feeds (Roadmap)
Goal: capture all edits with provenance and prepare incremental sync semantics for future MCP responses.

### Phase 10.1: Change Event Log
- Add `change_events` ledger for all mutable entities:
  - `event_id`, `entity_type`, `entity_id`, `project_id`
  - `actor_type` (`user|agent|system`)
  - `actor_id` / `actor_name`
  - `operation` (`create|update|delete|archive|restore|reorder|transition`)
  - `transition_from`, `transition_to` (for transition operations)
  - `contract_check_result` (`pass|fail|override`)
  - `field_changes` (structured diff)
  - `occurred_at`.
- Keep events append-only for audit and replay potential.

### Phase 10.2: Agent Cursor/Checkpoint Tracking
- Add per-agent cursor:
  - `agent_id`
  - `project_id`
  - `last_seen_event_id`
  - `last_sync_at`.
- Query contract (future MCP):
  - "changes since cursor"
  - "unseen new tasks/items"
  - "unseen memory node updates"
  - "unseen state transitions"
  - "unseen contract updates and transition validation failures."

### Phase 10.3: LLM Notification Semantics (Future MCP Response Shape)
- When an agent requests item/project context, response should include:
  - current snapshot
  - summary of changes since agent cursor
  - explicit list of new entities unseen by the agent
  - memory node deltas since last sync
  - active completion contract for targeted items
  - unmet contract checks relevant to requested transition actions.
- Include edit attribution summaries:
  - "user changed X"
  - "agent changed Y"
  - "system changed Z".

### Phase 10 Acceptance
- Every meaningful write emits an actor-attributed event.
- Per-agent cursor model is defined and testable locally.
- Delta payload contract is documented for later MCP implementation.

## Phase 11: MCP/HTTP Integration Roadmap (Not Implemented Yet)
Goal: define the path to expose this system as an MCP-capable planning backend.

### Phase 11.1: Transport and API Boundaries
- Keep app core transport-agnostic; expose use cases through ports.
- Add future adapter plan:
  - HTTP server
  - MCP tool/schema layer
  - auth and tenancy model (future multi-user/remote mode).

### Phase 11.2: Candidate MCP Tool Surface (Roadmap)
- `list_projects`
- `get_project_context`
- `search_items`
- `get_item_context`
- `create_item` / `update_item`
- `append_memory_node`
- `list_changes_since`
- `ack_changes`.

### Phase 11.3: Safety and Data Hygiene Requirements
- Path safety:
  - enforce project root allowlist.
  - reject path traversal outside configured roots.
- Secret handling:
  - redaction pipeline for known secret patterns in notes/resources.
- Context budgets:
  - ranked/truncated context blocks with deterministic priority.

### Phase 11 Acceptance
- Clear API/tool contracts exist before network implementation.
- Security constraints are designed before exposing remote interfaces.

## Roadmap Consensus Defaults (Locked)
- Project root paths:
  - TOML source of truth with optional DB hint (`project_roots.<project_slug>` authoritative).
- Nesting model:
  - generic `work_items` + `kind` + `parent_id` (no fixed-table hierarchy model).
- Memory precedence:
  - merge `project -> phase -> item`, nearest scope wins, conflicts preserved explicitly.
- Label scope:
  - project + phase first, item-level ad hoc later.
- Actor attribution:
  - always capture `actor_type`; optional `actor_id` and `session_id`.
- History policy:
  - append-only event envelope, optional retention for older detailed diffs.
- Lifecycle model:
  - fixed non-configurable lifecycle states: `todo`, `progress`, `done`, `archived`.
- Completion contracts:
  - mandatory contract support at every work-item level.
  - transition guards required for `todo -> progress` and `progress -> done`.
  - LLM transition responses must include active contract and unmet checks.

## Remaining Open Questions (Next Refinement Round)
- Transition enforcement mode:
  - should user-initiated transitions be hard-blocked on contract failure, or allow explicit override with reason?
- Child completion policy defaults:
  - should all parents require all children complete, or allow configurable policies by `kind`?
- Evidence schema:
  - should `completion_evidence` be free-form text only, structured entries, or both?
- Nesting depth and performance guardrails:
  - default max depth and practical limits for very large trees.
- Template bootstrapping:
  - what default templates ship for software projects, phases, and subtasks?
- Collaboration readiness:
  - what minimum auth/identity model is required before enabling remote shared mode later?

## Additional Roadmap Gaps to Track
- Template system:
  - reusable project/phase/task templates with default memory nodes, labels, resources, and completion contracts.
- Dependency and scheduling aids:
  - lightweight critical-path visibility and blocker surfacing.
- Bulk maintenance tools:
  - batch transitions, label apply/remove, due-date shifts, and contract assignment.
- Context quality signals:
  - stale resource warnings, missing acceptance criteria warnings, and missing evidence warnings.
- Review workflows:
  - daily/weekly digest summaries for human and agent handoff.
- Remote collaboration foundation:
  - conflict strategy, permission boundaries, and data ownership model.

## Parallel Workstream Candidates (Can Start Now)
- Safe to run now with low coupling to roadmap-phase schema work:
  - Phase 6.1 (search form UX + focus model)
  - Phase 6.2 (canonical state selector and filter model)
  - Phase 6.4 (list marker/card visual consistency)
  - Phase 6.5 (confirmation modals)
  - Phase 6.6 (modal consistency + `enter`/`i` behavior checks)
  - Phase 6.10 (help/discoverability refresh)
  - Phase 6.11 (test/VHS expansion for the above).
- Can run in parallel if kept interface-driven:
  - Phase 6.7 + 6.8 (due datetime + due window config).
- Should wait for schema implementation kickoff:
  - deep data-model work under Phase 7+ (work_items migration, completion contracts, and event ledger extensions).

## 12) Definition of Done (MVP)
- Multi-project board is fully usable from TUI.
- SQLite persistence is reliable and migration-backed.
- Search/filter and import/export are present.
- `vim` + arrow keys both work.
- Mouse select/scroll works.
- CI passes on macOS/Linux/Windows.
- TUI behavior covered by teatest.
- `just ci` is the single local/CI quality gate.

## 13) Worklog
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
- [x] 2026-02-21: Added golden-style TUI regression fixtures:
  - `internal/tui/model_teatest_test.go` captures final terminal output and asserts with Charm golden helper
  - added fixtures under `internal/tui/testdata/*.golden`
- [x] 2026-02-21: Hardened CI + release verification pipeline:
  - `ci.yml` now has concurrency control, Go cache, and a release snapshot verification job
  - `release.yml` now has concurrency control and Go cache
  - added local `just release-check` recipe for goreleaser snapshot validation
- [x] 2026-02-21: Added release-facing docs and golden test ergonomics:
  - added `README.md` with run/config/keymap/import-export/dev workflow docs
  - added `just test-golden` and `just test-golden-update` recipes
- [x] 2026-02-21: Prioritized app UX over release execution:
  - release/homebrew work remains roadmap-only
  - shifted active implementation back to TUI product polish
- [x] 2026-02-21: TUI layout refresh pass with VHS verification:
  - enabled alt-screen rendering for proper full-screen UX
  - added project tabs row and column task counts
  - introduced persistent overview/selection panel
  - improved edit modal guidance formatting and placement
  - updated golden fixtures after visual/layout changes
- [x] 2026-02-21: TUI layout stabilization pass (VHS + CI verified):
  - removed duplicate single-project tabs row (tabs only render for multi-project sessions)
  - stopped inline modal flow insertion that was pushing/clipping content
  - mode overlays now occupy the side panel (wide) or replace stacked panel region (narrow)
  - status line is hidden for idle `ready` state to reduce visual noise
  - help bar is now guaranteed to stay bottom-anchored via content height fitting
  - updated picker click offset + board top math for optional tabs row
  - refreshed golden fixtures and re-ran `just vhs` + `just ci`
- [x] 2026-02-21: Task editor and card rendering ergonomics pass:
  - fixed edit-mode prefill trap causing `expected 5 fields` after typing a full edit payload
  - edit mode now starts with empty input + visible current-value template guidance
  - corrected board column sizing to use actual board pane width in split layout
  - constrained card metadata width to prevent multi-line card row wrapping
  - revalidated via VHS captures (`board`, `workflow`) and `just ci`
- [x] 2026-02-21: Kancli-inspired visual pass (from `.tmp/kancli`):
  - adopted kancli-style palette (blue accent `62`, muted gray help/text)
  - changed board columns to hidden-border baseline with rounded border only on focused column
  - simplified main layout so board uses full width (closer to kancli’s column-first composition)
  - moved active input overlays above the board so edit/search inputs stay visible on dense boards
  - retuned column width heuristics and metadata truncation for compact list-style cards
  - updated golden fixtures and revalidated with `just vhs` and `just ci`
- [x] 2026-02-21: Modal/input reliability and coverage recovery pass:
  - added explicit AGENTS guardrail to query Context7 before code/test fixes
  - improved add/edit form defaults to placeholders (prevents `mediumhigh` / `-2026-...` append traps)
  - strengthened form navigation key handling (`tab`, `ctrl+i`, `shift+tab`, arrows) and focused field highlighting
  - centered modal overlays above board content and refreshed TUI golden fixtures
  - expanded TUI tests for modal rendering, parser helpers, row mapping, and validation branches
  - restored TUI coverage gate: `internal/tui` now 77.5% in `just ci`
- [x] 2026-02-21: UX refinement slice from direct review feedback:
  - added `i` task-info modal (read-first flow) with `e` shortcut to enter edit modal
  - kept direct `e` in board list for immediate edit
  - changed priority entry to an in-form picker (`h/l` or `←/→`) instead of free text typing
  - removed redundant modal footer legends that duplicated field explanations
  - restyled task list rows with a left-side marker bar to better match kancli card feel
  - recentered overlays across full viewport composition and refreshed goldens/VHS
- [x] 2026-02-21: Priority fixup requested by user:
  - updated `AGENTS.md` to strict Context7-first + `just`-only test rules
  - made modal overlays truly viewport-centered
  - mapped `enter` to open task info (same as `i`) and added test coverage
- [x] 2026-02-21: Phase 5 implementation kickoff:
  - [x] config expansion + example template + dev/prod path logic
  - [x] project create/edit metadata flows
  - [x] cross-project + state-aware search/filter
  - [x] Fang-style help/onboarding presentation
  - [x] command palette + quick actions
  - [ ] multi-select + bulk actions
  - [x] due picker + label suggestions
  - [ ] WIP warnings + swimlanes + activity log + undo/redo
- [ ] Next: close remaining Phase 5 stretch features (multi-select/bulk, activity log, undo/redo, swimlane/grouping polish) after UX baseline is stable.
- [x] 2026-02-21: Completed Phase 5 slices 1-5 baseline:
  - Slice 1: config/platform foundation + example TOML + docs
  - Slice 2: project metadata domain/storage/app + create/edit workflows
  - Slice 3: cross-project/state-aware search wired through app and TUI
  - Slice 4: Fang-style help modal + project management UX polish
  - Slice 5: command palette + quick actions + full verification (`just ci`, `just vhs`)
- [x] 2026-02-21: Completed Phase 5 Slice 1 details:
  - dev/prod path resolution via `--dev` / `KAN_DEV_MODE` and namespaced app paths via `--app` / `KAN_APP_NAME`
  - new `kan paths` command for active config/data/db path introspection
  - added `config.example.toml` with state/search/key defaults and override precedence notes
  - updated `README.md` and `Justfile` (`run-dev`, `paths`, `test-pkg`, `check-llm`)
  - added config tests for board/search/key overrides and state validation
- [x] 2026-02-21: Completed Phase 5 Slice 2 details:
  - added project metadata in domain (`owner`, `icon`, `color`, `homepage`, `tags`)
  - persisted metadata via SQLite `projects.metadata_json` with migration-safe column add
  - wired app service create/update project APIs and TUI add/edit project modals
  - snapshot import/export now preserves project metadata
- [x] 2026-02-21: Completed Phase 5 Slice 3 details:
  - added app-level `SearchTaskMatches` with cross-project/state-aware filtering
  - added TUI search modal fields for query + states
  - added toggles for cross-project scope (`ctrl+p`) and archived inclusion (`ctrl+a`)
  - added search-results modal with jump-to-project/task flow
- [x] 2026-02-21: Completed Phase 5 Slice 4 details:
  - replaced plain help with Fang-style centered overlay and workflow hints
  - improved modal composition so overlays render above board/help without layout shift
  - added `i`/`enter` task-info modal flow with `e` handoff into edit mode
- [x] 2026-02-21: Completed Phase 5 Slice 5 details:
  - added command palette (`:`) with project/search/help/archive commands
  - added quick actions menu (`.`) for common task actions
  - expanded TUI test coverage and refreshed golden outputs
  - verified with `just ci` and `just vhs`
- [x] 2026-02-21: Worklog update (current pass):
  - consulted Context7 for Bubble Tea/Bubbles/Lip Gloss patterns before edits
  - updated `AGENTS.md` to stricter Context7-first + `just`-recipe workflow template
  - reconciled Plan tracker status to reflect implemented Phase 5 baseline and pending stretch items
- [x] 2026-02-21: UX feature follow-up implementation:
  - added centered due-date picker modal (`ctrl+d` from task form due field)
  - wired due picker keyboard flow (`j/k`, `enter`, `esc`) and return-focus behavior
  - added label suggestion hints in task form based on existing project labels
  - added TUI tests for due picker flow and label-suggestion rendering
- [x] 2026-02-21: Verification + remediation loop:
  - `just test-pkg ./internal/tui` initially failed due sandbox cache path permissions; reran with local `GOCACHE`
  - fixed compile issue (`tea.KeyCtrlD` not available) by supporting `D` as due-picker shortcut in forms
  - golden output changed; refreshed fixtures with `just test-golden-update` and revalidated with cache-busted test run
  - full gate passed via `GOCACHE=$(pwd)/.go-cache just ci` (coverage floor preserved)
  - visual regression pass re-run with `just vhs` after one transient VHS tool panic; second run succeeded
- [x] 2026-02-21: Final verification commands:
  - `GOCACHE=$(pwd)/.go-cache just test-golden`
  - `just paths` (verified default dev-mode path resolution output)
- [x] 2026-02-21: Planning-only update (no code changes):
  - expanded Phase 6 into 12 detailed subphases covering search/state UX, command palette behavior, destructive-action confirmations, list marker rules, modal consistency, due datetime warnings, due summary row, and TOML-configurable due-soon windows
  - documented date picker decision: keep current lightweight picker + typed datetime support, no external datepicker dependency for now
- [x] 2026-02-21: MCP-oriented roadmap expansion (planning only; no code changes):
  - added Phase 7-11 roadmap for rich planning data model, nesting, memory nodes, actor-attributed change feed, and future MCP/HTTP integration
  - added open questions with recommended defaults to preserve context and guide next consensus round
  - documented parallel-safe workstreams another agent can execute while architecture details are finalized
- [x] 2026-02-21: Consensus lock + completion-contract planning update (planning only; no code changes):
  - locked canonical lifecycle states to `todo|progress|done|archived` and removed roadmap direction toward configurable state taxonomies
  - added completion contract requirements for all work-item levels, including transition guard behavior for `todo -> progress` and `progress -> done`
  - added explicit LLM transition context requirement: contract + unmet checks must be returned when state updates are attempted
  - expanded remaining-open-questions and parallel-workstream guidance to reflect new consensus
- [x] 2026-02-21: Repo-wide comment/docstring hardening pass (active)
  - objective: enforce explicit comment/docstring coverage across all Go code blocks, including tests
  - command: `ls -la` (repo inventory) -> success
  - command: `sed -n '1,220p' Justfile` (startup recipe review) -> success
  - command: `sed -n '1,260p' PLAN.md` + `tail -n 120 PLAN.md` (active worklog context) -> success
  - command: `sed -n '1,260p' README.md` + `sed -n '1,260p' AGENTS.md` -> success
  - command: `rg --files -g'*.go'` -> 35 Go files discovered
  - command: `rg -n '^func |^type |^var |^const |^//' -g'*.go' internal cmd` (declaration density scan) -> success
  - command: `mcp__context7-mcp__resolve-library-id` for Go doc-comment guidance -> failed (`Monthly quota exceeded`); proceeding with established idiomatic Go conventions for this pass
  - next: update `AGENTS.md` rule text, then execute code-wide comment/docstring edits + `just` verification
- [x] 2026-02-21: Repo-wide comment/docstring hardening pass (completed)
  - edit: updated `AGENTS.md` rule to require idiomatic comments/docstrings for every code block, including tests
  - command: generated declaration audit via `/tmp/missing_docs.go` -> reported 239 missing declaration comments before fixes
  - edit: executed automated insertion tool `/tmp/add_docs.go` across all `.go` files to add missing comments on:
    - all top-level `type`/`const`/`var` declarations
    - all functions and methods (including `Test*` and test helpers)
  - command: post-pass verification via `/tmp/missing_docs_allfuncs.go` -> `TOTAL 0`
  - command: `just fmt` -> success
  - command: `just check-llm` -> success
  - result: repository now has declaration-level doc comments across implementation and test code; build/test/coverage gate remains green
  - refinement: replaced low-quality templated phrasing with cleaner idiomatic sentence forms using `/tmp/refine_docs.go`
  - command: `just fmt` (post-refinement) -> success
  - command: `just check-llm` (post-refinement) -> success (tests, coverage floor, and build-all all green)
  - command: `go run /tmp/missing_docs_allfuncs.go` -> `TOTAL 0` (no missing declaration/method comments)
  - note: `cmd/kan/*.go` are currently matched by `.gitignore` pattern `kan`, so these comment updates are present in workspace but do not appear in `git status`
- [x] 2026-02-21: Investigated nested-module import diagnostic in `third_party/teatest_v2`
  - observed root module resolves `github.com/charmbracelet/x/exp/golden`, but nested module lacked `go.sum`
  - added `third_party/teatest_v2/go.sum` so editor/module resolution can load `golden` and `bubbletea` imports inside the local teatest patch module
- [x] 2026-02-21: Documented and fixed nested `third_party/teatest_v2` module metadata
  - request context: explain purpose of `third_party/` and resolve editor `packages.Load` error for `github.com/charmbracelet/x/exp/golden`
  - command: `go list` in root -> success; command in `third_party/teatest_v2` -> `go: updates to go.mod needed`
  - root cause: nested module metadata drift (`third_party/teatest_v2/go.mod` required additional indirect requirements and committed `go.sum` for reproducible package loading)
  - remediation: ran `cd third_party/teatest_v2 && go mod tidy` and kept resulting `go.mod` + `go.sum`
  - verification: `cd third_party/teatest_v2 && GOCACHE=/Users/evanschultz/Documents/Code/personal/kan/.go-cache go list ./...` -> success
  - verification: `cd third_party/teatest_v2 && GOCACHE=/Users/evanschultz/Documents/Code/personal/kan/.go-cache go list -deps -test ./...` -> success
  - docs: added `third_party/teatest_v2/README.md` with purpose, wiring, maintenance, error troubleshooting, and removal criteria
  - note: Context7 attempted again before edits and failed due monthly quota (`Monthly quota exceeded`)
  - command: `just check-llm` -> success (all packages cached green, coverage floor preserved)
- [x] 2026-02-21: CI shell failure remediation (`just ci` on GitHub ubuntu)
  - symptom: workflow failed before running recipes with `just could not find the shell: ...` for `zsh`
  - root cause: `Justfile` hard-coded `set shell := ["zsh", ...]`, but ubuntu runner image does not guarantee `zsh`
  - edit: changed `Justfile` shell to `bash` (`set shell := ["bash", "-eu", "-o", "pipefail", "-c"]`)
  - command: `just ci` -> success locally after change
  - note: Context7 attempted before edit and unavailable due quota (`Monthly quota exceeded`)
- [x] 2026-02-21: CI `fmt` recipe portability fix after ubuntu runner failure
  - symptom: `just ci` failed on ubuntu with `rg: command not found` in `fmt`
  - root cause: `fmt` depended on ripgrep being installed in runner image
  - initial attempted fix used shell fallback + make-style `$$` escaping; this caused local syntax/runtime issues and was corrected
  - final fix: `fmt` now formats tracked Go files via `git ls-files '*.go'` and `gofmt -w "$@"` (no ripgrep dependency)
  - verification: `just ci` passes locally after final fix
  - note: Context7 was retried before edits and unavailable due quota (`Monthly quota exceeded`)
- [x] 2026-02-21: CI matrix remediation for shell portability + TUI golden determinism
  - investigation scope: `Justfile`, `.github/workflows/ci.yml`, `internal/tui/model_teatest_test.go`, golden fixtures, and recent CI logs
  - command: `gh run list ...` failed in sandbox (`error connecting to api.github.com`), so run-log analysis used provided GitHub log excerpts + local reproduction
  - online/docs check: searched Just shell behavior docs; verified `windows-shell` setting guidance and bash/powershell defaults
  - root cause #1 (windows): workflow step ran `just ci` with default shell on `windows-latest`, causing bash resolution through WSL with no distro installed
  - fix #1: set `shell: bash` on the CI step that executes `just ci` in `.github/workflows/ci.yml`
  - root cause #2 (linux/macos): TUI golden tests asserted raw terminal control streams that vary by terminal capability (`TERM`) across runners
  - fix #2: in golden tests, force deterministic terminal env via `teatest.WithProgramOptions(tea.WithEnvironment([]string{"TERM=dumb"))` and regenerate goldens
  - commands run:
    - `just fmt` -> success
    - `GOCACHE=$(pwd)/.go-cache just test-golden-update` -> success
    - `GOCACHE=$(pwd)/.go-cache just test-golden` -> success
    - `TERM=xterm-256color GOCACHE=$(pwd)/.go-cache just test-golden` (after testcache clean) -> success
    - `GOCACHE=$(pwd)/.go-cache just check-llm` -> success
  - note: Context7 retried before edits and remained unavailable (`Monthly quota exceeded`)
- [x] 2026-02-21: Windows-only CI shell remediation follow-up
  - symptom: `windows-latest` still failed with `Windows Subsystem for Linux has no installed distributions` while running `just ci`
  - root cause refinement: `just` recipes invoke `bash`; on hosted windows this can resolve to WSL shim when not forced through Git Bash
  - edit: split CI run step by OS and run windows job via explicit Git Bash invocation:
    - unix: `shell: bash`, `run: just ci`
    - windows: `shell: pwsh`, `& "C:\Program Files\Git\bin\bash.exe" -lc "just ci"`
  - expected result: windows matrix run bypasses WSL shim and executes recipes in Git Bash
