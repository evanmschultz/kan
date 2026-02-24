# Kan TUI Plan + Worklog

Created: 2026-02-21  
Status: In progress (Phase 0-6 baseline + active Phase 11 Wave 1 execution tracked in `MCP_DESIGN_AND_PLAN.md`)  
Execution gate: Planning update only in this step (no code changes)

## 1) Product Goal

Build a polished, Charm-style Kanban TUI with local SQLite persistence, multiple projects, customizable columns, strong keyboard support (`vim` + arrows), mouse support, and cross-platform releases (macOS/Linux/Windows).

Scope guard:

- Pre-Phase-11 baseline implementation is local/TUI-first.
- Temporary active-wave override (2026-02-24): locked non-roadmap MCP/HTTP slices are in progress in `MCP_DESIGN_AND_PLAN.md`.
- Advanced import/export transport-closure concerns (branch/commit-aware divergence reconciliation and richer conflict tooling) remain roadmap-only unless user re-prioritizes.
- External system sync remains roadmap-only and intentionally deferred.

## 1.1) Canonical Document Set (Locked)

- `PLAN.md`: primary product intent, roadmap, and current planning source-of-truth.
- `PRE_MCP_CONSENSUS.md`: locked pre-MCP decision register.
- `MCP_DESIGN_AND_PLAN.md`: active Phase 11 execution/worklog hub for MCP/HTTP slices (design + lanes + checkpoint evidence).
- Historical execution/checkpoint notes are intentionally trimmed from non-canonical docs to reduce context overload.

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
- Kan is the authoritative planning source in current scope.
- External integrations (Git, GitHub, Jira, Slack, etc.) are roadmap-only; do not design/block current implementation around them.
- Logging baseline is in current scope:
    - use `github.com/charmbracelet/log`,
    - keep styled local console logs,
    - support dev-mode workspace log files under `.kan/log/`.
- Canonical lifecycle states are fixed and non-configurable for now:
    - `todo`, `progress`, `done`, `archived`.
- UI display labels for lifecycle states are fixed:
    - `To Do`, `In Progress`, `Done`, `Archived`.
    - internal stored value remains `todo|progress|done|archived` (for deterministic queries and MCP payloads).
- Completion contracts are required across all work-item levels (phase/task/subtask/etc):
    - transition into `progress` and `done` must evaluate contract criteria.
    - LLM-driven transitions must receive contract context and validation feedback.
- Workspace linking model is intentionally simple:
    - `workspace_linked = true|false` at project scope.
    - no extra "automation-ready" state in pre-Phase-11.
- Import safety policy (pre-Phase-11):
    - never export absolute paths.
    - import fails if referenced relative file/dir resources cannot be resolved from mapped roots.
    - advanced divergence reconciliation is roadmap-only.

## 2.1) Worklog Governance (Locked)

- Temporary wave override (2026-02-24):
    - MCP-oriented execution checkpoints are tracked in `MCP_DESIGN_AND_PLAN.md`.
    - `PLAN.md` remains the canonical roadmap/intent document and keeps only high-signal alignment notes.
- `PRE_PHASE11_CLOSEOUT_DISCUSSION.md` is a decision register and discussion memory artifact, not a step-by-step execution ledger.
- In parallel/subagent mode, only the orchestrator/integrator writes lock ownership, checkpoint progression, and completion state in the active wave worklog file.
- Worker subagents provide evidence handoffs; the orchestrator ingests those into the active wave worklog file.
- Every checkpoint entry in the active wave worklog must include:
    - objective and lane/checkpoint id,
    - files touched and why,
    - commands run,
    - test/check outcomes (or explicit `test_not_applicable` with reason),
    - failures/remediation,
    - current status and next step.
- No lane is marked complete without acceptance evidence and verification notes.
- No wave is marked complete without integrator closeout and successful `just ci`.

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
- Display names should be:
    - `To Do`
    - `In Progress`
    - `Done`.
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

- Remove configurable lifecycle state definitions for this phase.
- Keep canonical lifecycle states:
    - `todo`
    - `progress`
    - `done`
    - `archived`.
- Map board columns/sections to canonical lifecycle states deterministically.
- Use fixed display labels (`To Do`, `In Progress`, `Done`, `Archived`) for consistency.
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
- `system` actor definition:
    - internal app automation (migration/backfill/normalization/rule-driven maintenance), never anonymous.
    - must include stable `actor_id` (ex: `kan-system`) and source metadata.

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

### Phase 10.4: Attention Signals and Branch-Scoped Delivery Contract

- Add first-class attention records (node-scoped, capability-gated), not a single inline flag:
    - `attention_items` model keyed by level scope:
        - `scope_type`: `project|branch|phase|task|subtask`
        - `scope_id`
        - `state`: `open|acknowledged|resolved`
        - `kind`: `blocker|consensus_required|approval_required|risk_note` (extensible)
        - `summary`, `body_markdown`
        - actor audit (`created_by`, `created_at`, `resolved_by`, `resolved_at`)
        - optional `requires_user_action` marker.
- Delivery contract for future branch-context reads:
    - always return `changes_since_cursor` for all meaningful edits at or below the requested branch scope.
    - always return `active_attention_items` (open, unresolved attention entries) in the same response.
- Event-noise policy:
    - emit events for committed mutations only (`save`, `move`, `transition`, `archive`, `restore`, `delete`, contract/policy updates).
    - never emit per-keystroke modal typing events.
- Cursor/ack semantics:
    - cursor scope is `(agent_id, branch_id)`.
    - cursor advances only on explicit ack.
    - no ack means deterministic resend of the same unseen delta set.
- Session identifiers:
    - `session_id` may be carried optionally for diagnostics.
    - correctness and delivery guarantees are branch-scoped and do not require session affinity.
- Active-attention shape:
    - multiple active attention records are allowed per node and per branch context.
    - history remains auditable in `change_events` plus attention record lifecycle fields.
- Default transition gate:
    - unresolved blocker/approval-required attention blocks `progress -> done` by default.
    - explicit override remains policy-controlled and must be actor-attributed with reason.

### Phase 10.5: Authorization Model and Dangerous Mode Policy

- Default-safe authorization policy:
    - agent destructive actions require user approval by default.
    - agent transitions into `progress` and `done` require completion-contract validation by default.
- Add TOML policy surface for action-level permissions:
    - `permissions.agent.require_user_approval_for = [...]`
    - `permissions.agent.allow_without_approval = [...]`
    - `permissions.agent.allow_transition_on_contract_failure = false` (default).
- Add explicit dangerous-mode TOML section (default disabled):
    - `dangerous.agent.enabled = false`
    - `dangerous.agent.allow_destructive_without_approval = false`
    - `dangerous.agent.allow_transition_without_contract_pass = false`
    - `dangerous.agent.startup_warning = true`.
- Dangerous mode behavior:
    - when enabled, app must show persistent warning in help/status and startup.
    - all dangerous-mode actions must still be fully actor-attributed in `change_events`.
- Implementation requirement:
    - update `config.example.toml`, config validation, docs, and runtime permission checks together in one slice.

### Phase 10 Acceptance

- Every meaningful write emits an actor-attributed event.
- Per-agent cursor model is defined and testable locally.
- Delta payload contract is documented for later MCP implementation.
- Authorization policy and dangerous-mode config are validated and documented.
- Attention state model and branch-scoped delivery contract are documented for MCP-phase implementation.

## Phase 11: MCP/HTTP Integration Roadmap (Not Implemented Yet)

Goal: define the path to expose this system as an MCP-capable planning backend.

### Phase 11.0: Mandatory Research/Design Gate (No Build Yet)

- Before any MCP implementation work in this phase:
    - run a focused design review on `mcp-go` + stateless HTTP-served MCP architecture.
    - map the MCP adapter shape to existing hexagonal boundaries (app core remains transport-agnostic).
    - validate that the schema/contracts already locked in Phases 7-10 satisfy this adapter model.
- Research/discussion scope to explicitly cover:
    - dynamic tool discovery/update behavior and payload implications:
        - https://modelcontextprotocol.io/legacy/concepts/tools#tool-discovery-and-updates
    - tool-loading strategy for minimizing context-window overhead.
    - branch-scoped delta and attention delivery behavior in MCP responses.
- Dogfooding requirement:
    - this MCP slice must be planned so `kan` can be used to dogfood the broader system this project will belong to.
- Open questions to settle before coding 11.1+:
    - stateless request contract shape and cursor/ack lifecycle details.
    - whether any session metadata is needed beyond diagnostics.
    - dynamic tool refresh triggers and compatibility constraints for clients.
- Implementation guard:
    - do not start 11.1+ implementation until this 11.0 review/discussion is complete and documented.

### Phase 11.1: Transport and API Boundaries

- Keep app core transport-agnostic; expose use cases through ports.
- Add future adapter plan:
    - HTTP server
    - MCP tool/schema layer
    - auth and tenancy model (future multi-user/remote mode).

### Phase 11.2: Candidate MCP Tool Surface (Roadmap)

- `list_projects`
- `list_branches`
- `get_project_context`
- `get_branch_context`
- `search_items`
- `get_item_context`
- `create_item` / `update_item`
- `append_memory_node`
- `set_attention_state`
- `resolve_attention_item`
- `list_attention_items`
- `ack_attention_item`
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

### Phase 11.4: Delivery and Attention Defaults (Locked)

- Cursor advancement:
    - `ack_changes` is the canonical cursor-advance mechanism.
    - `get_branch_context` must not advance cursor unless an explicit ack behavior is requested by contract.
- Attention gating:
    - unresolved blocker/approval-required attention blocks `progress -> done` by default.
    - policy may allow explicit override with required actor-attributed reason.
- Attention cardinality:
    - many active attention records may exist per node.
    - event history captures create/ack/resolve operations with actor attribution.
- Branch-context delta scope:
    - include branch and descendants changes since cursor.
    - include project-level/config changes only when they affect branch execution context.
    - exclude unrelated project/global chatter from default payloads.
- Agent attention defaults:
    - agents can raise blocker/consensus/approval/risk attention records on scoped nodes.
    - resolving blocker/approval-required attention requires user approval by default (configurable).
- Session metadata:
    - optional diagnostics only; never correctness-critical.

### Phase 11.5: Attention UX + Query Contract Prerequisites

- TUI updates required for attention readiness:
    - list-row warning indicator when node has open attention/blocker entries.
    - always-visible compact attention panel that updates by currently rendered scope level.
    - filter integration for attention signals via search flow, quick actions (`.`), and command palette (`:`).
- DB/query requirements:
    - paginated scope query for attention records:
        - filters: `scope_type`, `scope_id`, `state`, `kind`, `requires_user_action`
        - pagination: `limit`, `cursor` (or equivalent deterministic page token)
        - sort: newest-first default with explicit alternate ordering support.
    - gatekeeping:
        - attention create/update/list operations must respect capability scope leases.
- MCP/tooling contract requirement:
    - all MCP tool definitions (read and write) must include a standard note that agents can escalate via node attention/blocker records when blocked on consensus/approval.
    - node-mutating tools must document the expected escalation path and required node identifiers for attention creation/update calls.
    - tool docs and generated agent templates must include "raise attention + ask user" behavior for unresolved consensus/approval paths.

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
    - always capture `actor_type`; require `actor_id` for writes; optional `session_id` for diagnostics only.
    - branch-scoped delivery correctness must not depend on session identifiers.
- History policy:
    - append-only event envelope, optional retention for older detailed diffs.
- Attention policy:
    - node-scoped `attention_items` records (project/branch/phase/task/subtask) with audit metadata.
    - unresolved blocker/approval-required attention blocks completion transitions by default.
- Lifecycle model:
    - fixed non-configurable lifecycle states: `todo`, `progress`, `done`, `archived`.
    - UI display label for `progress` is always `In Progress`.
- Completion contracts:
    - mandatory contract support at every work-item level.
    - transition guards required for `todo -> progress` and `progress -> done`.
    - LLM transition responses must include active contract and unmet checks.
- Transition enforcement:
    - default hard block for agent transitions when contract checks fail.
    - user override allowed with explicit confirmation and reason (logged).
- Evidence schema:
    - hybrid model: structured evidence entries + free-form notes.
- Parent-child completion default:
    - required children must be `done` before parent can move to `done` (unless explicit policy override).
- Contract versioning:
    - transitions record `contract_version` and evaluated snapshot for auditability.
- Dangerous mode:
    - exists as explicit TOML opt-in, defaults off, includes startup and in-app warnings.
- Actor identity:
    - `system` is internal automation, must be identified and audited like all other actors.
- Delta delivery policy:
    - branch-scoped changes are delivered via cursor/ack flow.
    - `ack_changes` is canonical cursor advance.
    - include only context-relevant project/config deltas in branch feeds.
- Logging policy:
    - runtime logging is required now (not deferred).
    - canonical package is `github.com/charmbracelet/log`.
    - dev mode must emit local file logs under workspace `.kan/log/`.

## Pre-Phase-11 Logging Baseline (In Scope Now)

Goal: make troubleshooting and runtime diagnostics reliable during current local/TUI development.

### Logging Baseline Requirements

- Adopt `github.com/charmbracelet/log` as the canonical runtime logger.
- Use styled/colorized terminal logs for local developer visibility.
- Log meaningful runtime operations end-to-end:
    - startup/config/path resolution,
    - DB open/migration and persistence failures,
    - mutating actions and transition guard failures.
- Add file logging in dev mode:
    - config-controlled dev mode flag enables workspace-local logs.
    - default dev log directory is `.kan/log/` under the current working directory.
    - include sensible defaults for file name, level, and append behavior.
- Ensure loggable failures are visible and diagnosable:
    - retain wrapped error chains (`%w`) for upstream handling.
    - log adapter/runtime boundary failures with enough context to reproduce.
- Document troubleshooting workflow:
    - where log files live in dev mode,
    - how to increase log verbosity,
    - how to correlate status-bar errors with log entries.

### Post-MCP Observability Expansion (Roadmap)

- After MCP/HTTP implementation is stable:
    - add service-oriented observability strategy (structured event export, metrics, tracing).
    - evaluate sinks/collectors and retention policies for team/shared deployments.
    - define correlation between MCP requests, branch context calls, and change-event writes.

## Known Intent Gaps (Code Audit: 2026-02-24)

- Full review ledger:
    - `PRE_MCP_FULL_CODE_REVIEW.md` (severity-ranked findings + HTTP/MCP entry-gate recommendation).
- Canonical root enforcement drift:
    - resource picker logic can fall back to search roots/current directory when project root mapping is missing.
    - intent requires deterministic hard-fail for write/attach flows that need project root scope.
    - references: `internal/tui/model.go` (`resourcePickerRootForCurrentProject`, `normalizeAttachmentPathWithinRoot`).
- Hierarchy context visibility drift:
    - projection breadcrumb currently resolves parent task title chain, but does not explicitly render branch/phase context labels.
    - intent requires clear hierarchy context while navigating project -> branch -> phase -> task -> subtask.
    - reference: `internal/tui/model.go` (`projectionBreadcrumb`).
- Kind seeding fallback strictness:
    - project allowlist bootstrap can seed built-in kinds when catalog/allowlist are empty.
    - intent is runtime DB-driven enum strictness; fallback behavior should stay explicit and tightly controlled.
    - reference: `internal/app/kind_capability.go` (`initializeProjectAllowedKinds`).

## Remaining Open Questions (Next Refinement Round)

- Nesting depth and performance guardrails:
    - default max depth and practical limits for very large trees.
- Template bootstrapping:
    - what default templates ship for software projects, phases, and subtasks?
- Collaboration readiness:
    - what minimum auth/identity model is required before enabling remote shared mode later?
- Dangerous mode operational UX:
    - should dangerous mode require a typed confirmation phrase each launch, or only a persistent warning banner?

## Additional Roadmap Gaps to Track

- External integrations and sync:
    - Git/GitHub/Jira/Slack and other API connectors are roadmap-only.
    - evaluate authoritative-sync policy and conflict ownership before implementation.
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
- Policy ergonomics:
    - role presets (`strict`, `balanced`, `dangerous`) to simplify TOML setup for common workflows.
- Config implementation guardrails:
    - example TOML, migration notes, and validation errors must stay synchronized across releases.
- Import reconciliation:
    - future guided resolution for path/branch divergence across shared snapshots.
    - pre-Phase-11 remains strict fail on unresolved relative refs.

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
