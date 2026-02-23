# Kan TUI Plan + Worklog

Created: 2026-02-21  
Status: In progress (Phase 0-6 active, MCP-oriented roadmap expansion added)  
Execution gate: Planning update only in this step (no code changes)

## 1) Product Goal

Build a polished, Charm-style Kanban TUI with local SQLite persistence, multiple projects, customizable columns, strong keyboard support (`vim` + arrows), mouse support, and cross-platform releases (macOS/Linux/Windows).

Scope guard:

- Pre-Phase-11 implementation is local/TUI-first.
- MCP/HTTP integration and external system sync are roadmap-only and intentionally deferred.

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

- `PLAN.md` is the only active execution/worklog ledger for this repository.
- `PRE_PHASE11_CLOSEOUT_DISCUSSION.md` is a decision register and discussion memory artifact, not a step-by-step execution ledger.
- In parallel/subagent mode, only the orchestrator/integrator writes lock ownership, checkpoint progression, and completion state in `PLAN.md`.
- Worker subagents provide evidence handoffs; the orchestrator ingests those into `PLAN.md`.
- Every checkpoint entry in `PLAN.md` must include:
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

- Add first-class attention fields on work items:
    - `attention_state` enum: `none|note|unresolved`.
    - `attention_note`.
    - `attention_set_by`, `attention_set_at`.
    - `attention_cleared_by`, `attention_cleared_at`.
- Delivery contract for future branch-context reads:
    - always return `changes_since_cursor` for all meaningful edits at or below the requested branch scope.
    - always return `active_attention_items` (open `note|unresolved`) in the same response.
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
    - one active attention record per work item (`state + note + audit metadata`).
    - many items may be concurrently flagged; item-level history remains in `change_events`.
- Default transition gate:
    - `attention_state=unresolved` blocks `progress -> done` by default.
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
- `clear_attention_state`
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
    - unresolved attention blocks `progress -> done` by default.
    - policy may allow explicit override with required actor-attributed reason.
- Attention cardinality:
    - one active attention record per work item.
    - event history captures prior set/clear operations.
- Branch-context delta scope:
    - include branch and descendants changes since cursor.
    - include project-level/config changes only when they affect branch execution context.
    - exclude unrelated project/global chatter from default payloads.
- Agent attention defaults:
    - agents can set `note|unresolved` by default.
    - clearing `unresolved` requires user approval by default (configurable).
- Session metadata:
    - optional diagnostics only; never correctness-critical.

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
    - one active attention record per item (`none|note|unresolved`), with set/clear audit metadata.
    - unresolved attention blocks completion transitions by default.
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

## Subagent Execution Model (Single Branch, No Worktrees)

This repository can run multiple subagents in parallel on one branch, but only with strict orchestration discipline.
Reference: see `PARALLEL_AGENT_RUNBOOK.md` for the generic reusable paradigm and templates.

### Roles

- Orchestrator agent:
    - decomposes work, assigns lanes, tracks lock ownership, and coordinates retries.
- Worker subagents:
    - implement one scoped lane and return patch artifacts plus test notes.
- Integrator agent:
    - the only actor allowed to apply patches to the shared branch.
    - runs integration tests and resolves conflicts.

### Orchestrator Prompt Contract (Required)

- Every worker-lane assignment must include:
    - lane id and single bounded objective,
    - lock scope (allowed file globs) and explicit out-of-scope paths,
    - concrete acceptance criteria for that slice,
    - architecture constraints (hexagonal dependency directions and hotspot restrictions),
    - required `just` commands for lane verification and TDD declaration (`tests-first` or justified exception),
    - doc/comment requirements for touched Go declarations and non-obvious logic blocks,
    - Context7 requirement and fallback behavior when Context7 is unavailable.
- Worker prompts must explicitly forbid:
    - edits outside lane lock,
    - direct `go test`,
    - cross-layer architecture violations unless explicitly required by lane objective.

### Locking and Edit Ownership

- Use lock entries in `PLAN.md` before starting each lane:
    - `lock_id`
    - owner (agent/lane)
    - file globs
    - acceptance target
    - `start_time`
    - `heartbeat`
    - `expires_at`.
- Subagents must not edit outside their lock scope.
- Hotspot files require serialized ownership:
    - `internal/tui/model.go`
    - `internal/app/service.go`
    - `internal/adapters/storage/sqlite/repo.go`.

### Single-Branch Parallel Protocol

- Subagents run in parallel for analysis, code edits, and patch generation.
- Integrator applies patches one-by-one in deterministic order.
- After each applied patch:
    - run package-level checks for touched areas (`just test-pkg <pkg>`),
    - then continue to next patch.
- End each wave with full gate:
    - `just ci`.

### Permission Escalation and Recovery Loop

- Subagents inherit sandbox policy and cannot do interactive approval prompts.
- If a subagent hits a permission-gated action:
    - action fails and returns to orchestrator context.
    - orchestrator reports exact failure and required approval.
    - user approves at parent level.
    - orchestrator reruns blocked command or resumes/restarts subagent lane with updated permissions.
- Progress continuation rule:
    - subagent resumes from last completed checkpoint recorded in `PLAN.md`.

### Delivery Unit and Checkpointing

- Smallest mergeable unit per lane:
    - one acceptance criterion or one tightly-coupled slice.
- Each lane update must record:
    - lane id + checkpoint id
    - files touched
    - commands executed
    - test outcomes
    - acceptance checklist (pass/fail per criterion)
    - architecture-boundary compliance note
    - doc/comment compliance note
    - unresolved risks.
- Integrator may reject a lane patch if:
    - lock violation
    - missing tests
    - failing acceptance criteria.
    - missing architecture/doc-comment compliance evidence.

### Suggested Wave Order for Current Roadmap

- Wave A:
    - Phase 6.1, 6.2, 6.4, 6.5, 6.6, 6.10, 6.11.
- Wave B:
    - Phase 6.7, 6.8 (due datetime + config).
- Wave C (serialized core-model wave):
    - Phase 7.1, 7.2, 7.6, 8.4.
- Wave D (serialized event/policy wave):
    - Phase 10.1, 10.2, 10.3, 10.4.
- Wave E:
    - docs, polish, and final validation.

### Acceptance for Parallel Execution

- No unresolved lock collisions.
- All applied patches have checkpoint records in `PLAN.md`.
- Permission-gated failures have explicit remediation notes.
- Final integrated branch passes `just ci`.

## Orchestrator Bootstrap (Active)

Single-branch parallel execution is now bootstrapped. This section is the source of truth for lane ownership and integration order.

### Active Locks

| lock_id | lane                | owner_role          | scope                                                                                           | objective                                                                                                          | status   | start_time | heartbeat  | expires_at |
| ------- | ------------------- | ------------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ | -------- | ---------- | ---------- | ---------- |
| L-A     | Wave A / Lane A     | worker-subagent     | `internal/tui/*`, `vhs/*`                                                                       | Phase 6.1/6.2/6.4/6.5/6.6/6.10/6.11                                                                                | verified | 2026-02-21 | 2026-02-21 | closed     |
| L-B     | Wave A / Lane B     | worker-subagent     | `internal/config/*`, `config.example.toml`, `README.md`                                         | Phase 6.7/6.8 config + due windows alignment                                                                       | verified | 2026-02-21 | 2026-02-21 | closed     |
| L-C     | Wave 1 / Lane C     | worker-subagent     | `internal/adapters/storage/sqlite/**`, `internal/app/**`, `internal/domain/**`                  | pre-Phase-11 backend foundations (`work_items`, dependency rollups, change event ledger)                           | verified | 2026-02-22 | 2026-02-22 | closed     |
| L-D     | Wave 1 / Lane D     | worker-subagent     | `internal/tui/**`, `internal/config/**`, `cmd/kan/main.go`, `config.example.toml`               | Phase 5 stretch UX (multi-select/bulk, activity log, undo/redo, grouping + WIP)                                    | verified | 2026-02-22 | 2026-02-22 | closed     |
| L-E     | Wave 2 / Lane E     | worker-subagent     | `internal/tui/**`, `internal/config/**`, `cmd/kan/main.go`, `config.example.toml`               | remaining pre-Phase-11 UI gaps (resource picker, label inheritance UI, projection breadcrumbs, dependency rollups) | verified | 2026-02-22 | 2026-02-22 | closed     |
| L-F     | Wave 3 / Lane F     | worker-subagent     | `internal/tui/**`                                                                               | durable activity log modal from persisted `change_events`                                                          | verified | 2026-02-22 | 2026-02-22 | closed     |
| L-G     | Wave 3 / Lane G     | worker-subagent     | `cmd/kan/**`, `internal/config/**`, `config.example.toml`, `README.md`                          | pre-Phase-11 logging baseline (`charmbracelet/log`, dev file logging)                                              | verified | 2026-02-22 | 2026-02-22 | closed     |
| L-J     | Wave 3 / Lane J     | worker-subagent     | `internal/app/**`                                                                               | cleanup: remove unused `SearchTasks` runtime path                                                                  | verified | 2026-02-22 | 2026-02-22 | closed     |
| L-I     | Integrator          | integrator          | shared-branch apply scope                                                                       | serialized patch apply + validation + `just ci` gate                                                               | verified | 2026-02-22 | 2026-02-22 | closed     |
| L-QA    | Audit / Remediation | orchestrator+worker | `internal/tui/**`, `internal/app/**`, `internal/adapters/storage/sqlite/**`, `vhs/*`, `PLAN.md` | independent quality audit + pre-Phase-11 completion reconciliation                                                 | verified | 2026-02-22 | 2026-02-22 | closed     |

### Lane State Machine

- `planned`
- `in_progress`
- `ready_for_integration`
- `integrated`
- `verified`
- `closed`.

### Integration Queue (Initial)

1. Lane A patch slice 1 (`6.1` search focus/controls)
2. Lane A patch slice 2 (`6.2` canonical state selector)
3. Lane A patch slice 3 (`6.4` list marker cleanup)
4. Lane A patch slice 4 (`6.5` confirmations)
5. Lane A patch slice 5 (`6.6` modal consistency)
6. Lane A patch slice 6 (`6.10` help copy)
7. Lane A patch slice 7 (`6.11` tests/VHS)
8. Lane B patch slice (`6.7` + `6.8` due datetime/config alignment)
9. Integrator full wave verification (`just ci`).

### Permission Escalation Queue

- Record each permission-gated failure as:
    - lane
    - blocked command
    - reason
    - approval requested
    - remediation outcome.

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
    - [x] multi-select + bulk actions
    - [x] due picker + label suggestions
    - [x] WIP warnings + swimlanes + activity log + undo/redo
- [x] Next: close remaining Phase 5 stretch features (multi-select/bulk, activity log, undo/redo, swimlane/grouping polish) after UX baseline is stable.
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
- [x] 2026-02-21: Consensus hardening update (planning only; no code changes):
    - locked UI state labels to `To Do|In Progress|Done|Archived` while preserving canonical internal enum values
    - added dangerous-mode authorization roadmap (TOML policy, startup warnings, audit requirements)
    - defined `system` actor as internal automation with mandatory attribution metadata
    - converted transition/evidence/child-policy/versioning items from open questions into locked defaults
- [x] 2026-02-21: Subagent orchestration planning update (planning only; no code changes):
    - added single-branch/no-worktree subagent execution model with lock ownership, integrator-only patch application, and wave sequencing
    - documented permission-failure escalation loop (subagent fail -> parent approval -> retry/resume)
    - linked current phase set to practical multi-agent wave ordering
    - added `PARALLEL_AGENT_RUNBOOK.md` with project-agnostic parallel execution framework, templates, and external resources
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
- [x] 2026-02-21: Final windows CI shell-resolution fix for `just`
    - symptom: windows job still invoked WSL shim (`System32\\bash.exe`) even when launching `just ci` from Git Bash
    - root cause: `Justfile` global `shell := ["bash", ...]` allows Windows process resolution to prefer `C:\\Windows\\System32\\bash.exe` before PATH
    - fix: added `set windows-shell := ["C:/Program Files/Git/bin/bash.exe", "-eu", "-o", "pipefail", "-c"]` to force Git Bash for all `just` recipes on Windows
    - command: `just ci` -> success locally
    - doc references used: `just` settings docs for `windows-shell` behavior and Windows shell precedence
    - note: Context7 retried before edit and unavailable (`Monthly quota exceeded`)
    - follow-up cleanup: windows CI step now runs `just ci` directly in `pwsh`; `windows-shell` in `Justfile` controls recipe shell path explicitly
- [x] 2026-02-21: Windows path-separator test portability fix (`internal/platform`)
    - user request: explain and then fix windows-only `internal/platform` test failures
    - docs consulted first:
        - Go `path/filepath` package docs (`filepath.Join` uses OS-specific separators)
        - fallback source due Context7 quota block: `go doc path/filepath` locally
    - Context7 retry before edit failed (`Monthly quota exceeded`)
    - root cause: tests asserted hardcoded Unix-style path strings while `PathsFor` uses `filepath.Join`, which emits `\\` separators on windows hosts
    - edit: updated expected paths in `internal/platform/paths_test.go` to use `filepath.Join` for Linux/Darwin/fallback cases
    - command: `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/platform` -> success
    - command: `GOCACHE=$(pwd)/.go-cache just check-llm` -> success
- [x] 2026-02-21: Investigated release-snapshot failure (`couldn't find main file: stat cmd/kan`)
    - user request: investigate/discuss only, no fixes yet
    - evidence:
        - `.goreleaser.yml` expects build `main: ./cmd/kan`
        - `git ls-files` returns no tracked files under `cmd/`
        - `git check-ignore -v cmd/kan/main.go` shows ignore rule `.gitignore:4:kan`
        - `git status --ignored` shows `!! cmd/` (entire directory ignored)
    - root cause: ignore pattern `kan` is currently matching `cmd/kan`, so entrypoint sources exist locally but are not versioned; CI checkout does not contain them
    - why test-matrix can still look mostly green: `go test ./...` in CI only sees tracked packages from checkout, so it does not validate missing ignored entrypoint package
    - why release check fails: goreleaser explicitly resolves `./cmd/kan`, which is absent in clean CI checkout
- [x] 2026-02-21: Repository integrity audit for ignored/untracked source drift (investigation-only)
    - objective: verify whether ignored-but-required source exists beyond `cmd/kan`
    - command: compared local vs tracked Go files (`find ... '*.go'` vs `git ls-files '*.go'`)
        - untracked go files detected: `cmd/kan/main.go`, `cmd/kan/main_test.go`, `.tmp/kancli/*` (scratch area)
    - command: per-dir tracked audit (`cmd`, `internal`, `.github`, `third_party`, `vhs`)
        - only unexpected untracked source under first-class dirs: `cmd/kan/*.go`
    - command: `git check-ignore -v cmd/kan/main.go` -> matched by `.gitignore` pattern `kan`
    - conclusion: primary repo-integrity issue is broad ignore rule `kan` unintentionally masking `cmd/kan`; no second ignored source tree found under core project dirs
- [x] 2026-02-21: Parallel governance hardening (docs/process only; no code changes)
    - updated `AGENTS.md` with explicit subagent-parallel policy:
        - single-writer `PLAN.md` rule
        - worker-lane lock boundaries
        - integrator-only shared-branch apply
        - permission-failure escalation loop
        - lane closeout requires integrator verification
    - updated `PARALLEL_AGENT_RUNBOOK.md` with mandatory bootstrap rule:
        - before parallel execution in any repo, update that repo's `AGENTS.md` to encode execution policy
    - initialized orchestrator bootstrap block in `PLAN.md`:
        - active lock table
        - lane state machine
        - initial integration queue
        - permission escalation queue format
- [x] 2026-02-21: Codex subagent config readiness check (investigation/discussion only; no code changes)
    - objective: verify whether current `~/.codex/config.toml` requires changes to use subagents in this repo
    - local docs/process reviewed:
        - `PLAN.md` subagent execution model and lock protocol (`Subagent Execution Model`, `Orchestrator Bootstrap`)
        - `PARALLEL_AGENT_RUNBOOK.md` bootstrap and approval-escalation requirements
    - commands:
        - `codex --version` -> `codex-cli 0.104.0`
        - `codex features list` -> `multi_agent experimental false`
    - online source check:
        - OpenAI Codex multi-agent docs indicate subagents require explicit enablement via `/experimental` or config `features.multi_agent = true`
        - OpenAI config basics confirm per-project settings load only for trusted projects; this repo is already configured as trusted
    - conclusion:
        - process docs in this repo are already aligned for subagent operation
        - config update still needed to persist enablement: set `features.multi_agent = true` (or enable each session via `/experimental`)
- [x] 2026-02-21: Subagent smoke run after config update (runtime verification; no code changes)
    - objective: verify subagent execution path by delegating a real repo task
    - startup policy check:
        - reviewed `Justfile` to confirm repo automation baseline before execution
    - commands/actions:
        - spawned explorer subagent `019c8236-9ed4-7c62-8993-d9098eb99cb9`
        - task: read `README.md` and return purpose/architecture/commands/caveats summary
        - waited for completion and captured handoff response
        - closed subagent handle after successful completion
    - result:
        - subagent completed successfully and returned requested README summary
        - delegation path is working in current session
- [x] 2026-02-21: Subagent prompt-policy hardening pass (docs/process only; no product code changes)
    - objective: tighten orchestrator-to-worker prompting so lanes consistently enforce hexagonal boundaries, TDD intent, doc/comment quality, and evidence-rich handoffs
    - commands/context review:
        - `sed -n '1,320p' AGENTS.md`
        - `sed -n '730,930p' PLAN.md`
        - `sed -n '1,320p' PARALLEL_AGENT_RUNBOOK.md`
        - `nl -ba AGENTS.md | sed -n '1,260p'`
        - `nl -ba PLAN.md | sed -n '730,900p'`
        - `nl -ba PARALLEL_AGENT_RUNBOOK.md | sed -n '1,280p'`
    - edits:
        - `AGENTS.md`: clarified Context7 fallback behavior; strengthened doc/comment requirement wording; added required orchestrator prompt contract and required worker handoff contract
        - `PLAN.md`: added required orchestrator prompt contract in subagent model; expanded checkpoint evidence requirements; aligned active lock schema to include `start_time`, `heartbeat`, `expires_at`
        - `PARALLEL_AGENT_RUNBOOK.md`: added generic orchestrator assignment contract, expanded lane handoff template with acceptance/compliance fields, and expanded bootstrap checklist to include assignment + handoff requirements
    - verification:
        - docs-only change; no Go source touched and no tests required for this pass
- [x] 2026-02-21: Execution wave complete (non-roadmap implementation only; no MCP/HTTP)
    - scope lock:
        - implement concrete UX/config fixes from Phase 6 and outstanding non-roadmap product behavior before Phase 11
        - explicitly exclude MCP transport and roadmap-only schema phases
    - startup checks completed:
        - reviewed `Justfile` recipes (tests via `just` only)
        - re-reviewed `PLAN.md` phase checklist and lock table
        - consulted Context7 for Bubble Tea/Bubbles input and modal patterns before edits
    - Lane B implementation (`internal/config/*`, `config.example.toml`, `cmd/kan/main.go`):
        - locked canonical lifecycle search-state validation to `todo|progress|done|archived`
        - removed runtime dependence on configurable `board.states`
        - added confirmation config surface:
            - `confirm.delete`
            - `confirm.archive`
            - `confirm.hard_delete`
            - `confirm.restore`
        - added due-summary UI config surface:
            - `ui.due_soon_windows`
            - `ui.show_due_summary`
        - updated command wiring to pass confirm/UI settings into TUI and dropped `stateTemplatesFromConfig` path
    - Lane A implementation (`internal/tui/*`):
        - rebuilt search modal interaction model:
            - focus order `query -> states -> scope -> archived -> apply`
            - canonical multi-select state toggles (no free-text state input)
            - clear-query and reset-filters command semantics
        - upgraded command palette:
            - live filtered commands while typing
            - highlighted selection execution on enter
            - tab autocomplete for top match
            - descriptions + aliases shown inline
            - clear-query and reset-filters command aliases wired
        - fixed board row marker semantics:
            - removed `>` indicator
            - only focused item uses the vertical accent bar
        - added confirmation modal for destructive/state-changing actions
            - archive, hard delete, default delete mode, restore (config gated)
            - `d` now respects `confirm.delete` independently from `confirm.archive`/`confirm.hard_delete`
        - kept modal overlays centered and non-shifting
        - added due datetime support (typed date or datetime), due-past warning hint, and due summary row:
            - `<overdue_count> overdue * <due_soon_count> due soon`
        - updated help overlay copy for new search/command/confirmation semantics
    - tests and verification:
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/config` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/tui` -> pass (after fixture + behavior updates)
        - `GOCACHE=$(pwd)/.go-cache just test-golden-update` -> pass (golden fixtures refreshed)
        - `GOCACHE=$(pwd)/.go-cache just test` -> pass
        - `GOCACHE=$(pwd)/.go-cache just ci` -> pass
            - package coverage floors remain >= 70% (TUI now 72.7%)
        - targeted regression fix:
            - search query typing preserved while in query focus (no `h/l` shortcut interception)
    - operational note:
        - observed non-fatal Go stat-cache write warning under sandboxed module cache during final `just ci`; command still exited successfully.
- [x] 2026-02-21: Phase 7/8 foundation slice implemented (non-MCP runtime features only)
    - objective:
        - implement pre-Phase-11 local feature foundations after Phase 6:
            - rich work-item/task context model
            - nesting foundations (parent/child + kind)
            - completion-contract transition guards
            - project-root and label-policy TOML surfaces
    - implementation summary:
        - domain model expansion (`internal/domain/*`):
            - added canonical lifecycle/actor/kind modeling:
                - `LifecycleState` (`todo|progress|done|archived`)
                - `ActorType` (`user|agent|system`)
                - `WorkKind` defaults (`task|subtask|phase|decision|note`)
            - added rich task metadata model:
                - context blocks (typed + importance)
                - resource references (local/url/doc/ticket/snippet with path mode)
                - completion contracts (start criteria, completion criteria, checklist, evidence, policy)
            - added task-level operations:
                - lifecycle transition timestamp management
                - planning metadata updates
                - parent reassignment
                - unmet criteria derivation for start/completion checks
        - app-layer behavior (`internal/app/*`):
            - `CreateTaskInput` now supports `parent_id`, `kind`, metadata, actor attribution
            - move transitions now enforce completion contracts:
                - `todo -> progress` checks unmet start criteria
                - `progress -> done` checks completion criteria/checklist and optional child-done policy
            - added nesting use cases:
                - `ListChildTasks`
                - `ReparentTask`
            - lifecycle resolution now maps from canonical column state semantics while preserving backward compatibility
            - snapshot compatibility updated to include rich fields while auto-defaulting missing legacy values
        - sqlite adapter updates (`internal/adapters/storage/sqlite/repo.go`):
            - migrated `tasks` schema with compatibility-safe additive columns:
                - `parent_id`, `kind`, `lifecycle_state`
                - `metadata_json`
                - actor attribution columns
                - lifecycle timestamp columns (`started_at`, `completed_at`, `canceled_at`)
            - added parent index and scan/create/update support for new fields
        - config + wiring updates:
            - added TOML surfaces:
                - `[project_roots]`
                - `[labels]` + `[labels.projects]`
            - added label allowlist policy:
                - `labels.enforce_allowed`
            - added config helper:
                - `AllowedLabels(project_slug)`
            - updated runtime wiring to pass label policy into TUI
            - updated `config.example.toml` with project-root and label-policy examples
        - TUI behavior updates (`internal/tui/*`):
            - added subtask creation action:
                - `s` key (`new subtask`) from selected task
                - command palette commands: `new-task`, `new-subtask`, `edit-task`
            - task form now supports label suggestions from config allowlists (global + project-scoped)
            - optional label enforcement in form submit path when enabled in TOML
            - hierarchical task ordering in column rendering (parent before children) with indentation
            - help/workflow text updated for subtask flow
            - task info modal includes kind/state/parent/objective/completion summary hints
    - test/verification trail:
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/domain` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/config` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/app` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/adapters/storage/sqlite` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-golden-update` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/tui` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test` -> pass
        - `GOCACHE=$(pwd)/.go-cache just ci` -> pass
            - coverage floors restored:
                - `internal/app` 70.6%
                - `internal/domain` 77.0%
                - `internal/tui` 70.4%
    - operational note:
        - non-fatal Go stat-cache write warning persists under sandboxed module cache during `just ci`; command still exits successfully.
    - remaining pre-Phase-11 gaps not completed in this slice:
        - full first-class `work_items` table migration (currently task-compatible extension, not full table replacement)
        - explicit file-picker UX for resource attachment
        - full project/phase-scoped label inheritance UI (foundation is in TOML + task form enforcement/suggestions)
        - dedicated nested board projections/breadcrumb mode switching beyond current parent/child ordering + subtask creation
        - dependency graph UX (`depends_on`, `blocked_by`) and rollup visualizations
    - resolution note:
        - these gaps were closed in 2026-02-22 Wave 1/2 integrations (`L-C`, `L-D`, `L-E`)
- [x] 2026-02-21: Post-slice coverage recovery + gate verification
    - command: `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/domain` -> pass
    - command: `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/app` -> pass
    - command: `GOCACHE=$(pwd)/.go-cache just ci` -> pass
    - result:
        - restored per-package coverage floor compliance after schema/model expansion
        - final gate remains green with known non-fatal sandbox stat-cache write warning.
- [x] 2026-02-21: Runtime migration hotfix for legacy DBs missing `tasks.parent_id`
    - user-reported failure:
        - `just run` -> `error: migrate sqlite: SQL logic error: no such column: parent_id (1)`
    - diagnostics:
        - default runtime DB path is dev-mode platform path, not repo root:
            - `go run ./cmd/kan paths` -> `/Users/evanschultz/Library/Application Support/kan-dev/kan-dev.db`
        - legacy DB verified to lack `parent_id`:
            - `sqlite3 ... \"PRAGMA table_info(tasks);\"` showed old schema without added columns
        - root cause:
            - migration attempted `CREATE INDEX ... tasks(project_id, parent_id)` before additive `ALTER TABLE` statements for legacy `tasks`
    - docs consulted before edit:
        - Context7 (`/websites/sqlite_cli`) for SQLite index/column behavior and migration ordering constraints
    - code fix:
        - moved `idx_tasks_project_parent` creation out of initial statement batch so it runs only after legacy column-add migration path
        - file: `internal/adapters/storage/sqlite/repo.go`
    - regression test:
        - added legacy-schema migration test that seeds old `tasks` table and verifies `parent_id` exists after `Open`
        - file: `internal/adapters/storage/sqlite/repo_test.go`
    - verification:
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/adapters/storage/sqlite` -> pass
        - legacy runtime upgrade simulation:
            - create old schema db via `sqlite3`
            - run `KAN_DB_PATH=<legacy.db> go run ./cmd/kan export --out -`
            - result: success
        - `GOCACHE=$(pwd)/.go-cache just ci` -> pass
    - operational notes:
        - local sandbox cannot write to user home data path, so local verification used `KAN_DB_PATH` in writable temp path
        - non-fatal Go stat-cache warning remains under sandboxed module cache; `just ci` exits success
- [x] 2026-02-21: Orchestrator wave kickoff (current run)
    - diagnosed `just test-pkg ./cmd/` failure:
        - root cause: recipe passed package arg directly and `./cmd/` has no Go files; package path is `./cmd/kan`
        - fix: `test-pkg` now auto-expands directory inputs with no local Go files to `<dir>/...`
        - verification:
            - `just test-pkg ./cmd/` -> pass
            - `just test-pkg ./cmd/kan` -> pass
    - launched parallel worker lanes:
        - `L-C`: backend pre-Phase-11 foundations
        - `L-D`: TUI Phase 5 stretch UX features
- [x] 2026-02-22: Wave 1 backend lane integrated (`L-C`)
    - implemented canonical persistence + audit foundation:
        - added `work_items` table migration and non-destructive legacy bridge from `tasks`
        - switched task CRUD queries to canonical `work_items`
        - added `change_events` ledger table and event emission for task mutations
        - added app/domain surface for project activity events and dependency rollup summaries
    - verification:
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/domain` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/app` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/adapters/storage/sqlite` -> pass
    - lane closure:
        - fixed failing fixture (`labels_json`) during lane finalization and re-verified package tests
- [x] 2026-02-22: Wave 1 TUI lane integrated (`L-D`)
    - completed Phase 5 stretch UX in TUI:
        - multi-select and bulk actions (move/archive/delete) with command palette + quick action integration
        - activity log modal with bounded retention
        - undo/redo basics for reversible operations
        - grouping/WIP warning rendering and configurable key overrides
    - verification:
        - `GOCACHE=$(pwd)/.go-cache just test-golden-update` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/tui` -> pass
- [x] 2026-02-22: Wave 2 advanced TUI lane integrated (`L-E`)
    - closed remaining pre-Phase-11 UI gaps:
        - resource picker modal (filesystem browse + attach resource refs to task metadata)
        - inherited label picker + source visibility (`global/project/phase`)
        - subtree projection mode with breadcrumb (`f` focus / `F` clear)
        - dependency rollup summaries and task dependency hints in task info modal
    - wiring updates:
        - passed `project_roots` config into TUI options
        - updated help/keymap surfaces for new flows
    - verification:
        - `GOCACHE=$(pwd)/.go-cache just test-golden-update` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/tui` -> pass
- [x] 2026-02-22: Coverage recovery + final integrator gate
    - issue:
        - `just ci` initially failed with `internal/tui` coverage below floor (`68.5%`, then `69.5%`, then `69.8%`)
    - remediation:
        - added targeted helper/edge-path tests in `internal/tui/model_test.go` and `internal/tui/keymap_test.go`
        - refreshed formatting and reran package tests
    - final verification:
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/tui` -> pass
        - `GOCACHE=$(pwd)/.go-cache just ci` -> pass
            - `internal/tui` coverage restored to `70.1%`
    - operational note:
        - earlier VHS panic in sandboxed runtime was resolved; rerun with approved `vhs` prefix now succeeds locally.
- [x] 2026-02-22: Post-integration audit + VHS visual inspection (L-QA, current run)
    - objectives:
        - verify AGENTS/RUNBOOK contracts enforce subagent Context7 and package-scoped `just test-pkg` behavior.
        - run independent quality review against pre-Phase-11 completion claims.
        - validate VHS artifacts visually (not test output only).
    - reviewer subagent:
        - spawned explorer reviewer lane and collected findings.
        - reviewer output summary:
            - missing durable activity log wiring:
                - TUI activity modal reads in-memory `activityLog` only.
                - persisted `change_events` APIs exist but are not consumed by TUI.
            - missing first-run onboarding flow:
                - Phase 5 plan includes onboarding but no dedicated onboarding mode/flag exists in TUI state machine.
            - dead/unused path:
                - `internal/app/service.go` `SearchTasks` is not used by runtime clients (test-only usage).
    - VHS validation:
        - commands:
            - `vhs vhs/board.tape` -> pass
            - `vhs vhs/workflow.tape` -> pass
        - visual verification:
            - extracted representative frames from `.artifacts/vhs/board.gif` and `.artifacts/vhs/workflow.gif`.
            - confirmed centered modal rendering and board/help overlays appear in captured frames.
            - confirmed first-frame shell prompt screenshots are expected tape startup frames, not app runtime failures.
    - gate:
        - `GOCACHE=$(pwd)/.go-cache just ci` -> pass
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./cmd/` -> pass
    - resulting status adjustment:
        - pre-Phase-11 is functionally close but not fully complete against `PLAN.md` acceptance text.
        - open gaps to close before claiming full completion:
            - durable activity log modal backed by persisted change events.
            - first-run onboarding flow.
            - optional cleanup: remove or integrate currently-unused `SearchTasks` path.
- [x] 2026-02-22: Manual worksheet synthesis + external research discussion packet
    - user input source reviewed:
        - `TUI_MANUAL_TEST_WORKSHEET.md` notes across every section (0-15).
    - external research completed for decision support:
        - Bubble Tea/Bubbles/Lip Gloss capabilities (fuzzy list filtering, filepicker, placement, mouse options)
        - task-manager behavior references (taskwarrior-tui, taskwarrior docs, Trello, ClickUp)
        - path portability and multi-root workflows (`os.UserConfigDir`, `filepath.Rel`, `git-worktree`)
        - datepicker compatibility (`ethanefung/bubble-datepicker` dependency versions)
    - artifact created:
        - `PRE_PHASE11_CLOSEOUT_DISCUSSION.md`
    - artifact scope:
        - explicit pre-Phase-11 remaining gaps
        - section-by-section mapping of worksheet notes to implementation/discussion items
        - ordered decision list for final consensus round
        - source links for each researched area
- [x] 2026-02-22: Manual worksheet deep pass refinement (line-referenced)
    - re-read all user notes from `TUI_MANUAL_TEST_WORKSHEET.md` with line references and reconciled against current code paths.
    - expanded `PRE_PHASE11_CLOSEOUT_DISCUSSION.md` with:
        - direct worksheet line mapping for each raised concern,
        - current-behavior vs closeout-direction notes,
        - architecture-first clarifying question set for final consensus lock.
- [x] 2026-02-22: Consensus and architecture-discussion expansion (second discussion pass)
    - captured latest user consensus points (global-first model, pointer-only resources, fuzzy everywhere, onboarding moved to roadmap).
    - restored missing dropped discussion points ("2-4") in closeout doc and tied them to implementation implications.
    - added ASCII option sets with pros/cons for:
        - archived item UX,
        - hierarchy/subtree rendering model,
        - dependency authoring/visibility model.
    - added big-picture architecture questions to drive decisions before low-level execution.
    - added updated external references for MCP roots, Trello/Jira archive behavior, ClickUp/Linear dependency and hierarchy patterns.
- [x] 2026-02-22: Third-pass consensus capture (track-only clarification + roots/import policy + SQLite direction)
    - updated `PRE_PHASE11_CLOSEOUT_DISCUSSION.md` with:
        - explicit clarification that kan is track/planning-only; execution permissioning is external policy context.
        - project path policy (`linked/unlinked`) and agent eligibility semantics.
        - strict portable import/export root-resolution flow with local TOML mapping and bypass warning path.
        - locked decisions for archive/hierarchy/dependency first-ships (A1/B1/C1).
        - SQLite-first storage direction and graph-db deferment to roadmap.
        - lifecycle/lexicon consistency notes and recovered missing 5-7 discussion items.
- [x] 2026-02-22: Fourth-pass consensus cleanup (terminology + scope lock)
    - removed confusing capability terminology from closeout discussion:
        - replaced `planning_ready`/`automation_context_ready` with `workspace_linked` only.
    - tightened import policy in closeout discussion:
        - strict-fail unresolved relative path refs in pre-Phase-11.
        - advanced divergence reconciliation moved to roadmap.
    - updated `PLAN.md`/`README.md` scope statements:
        - local/TUI-first implementation now.
        - MCP/HTTP and external connectors explicitly roadmap-only.
- [x] 2026-02-22: Fifth-pass consensus lock (user feedback reconciliation)
    - refined closeout doc policy for linked/unlinked projects:
        - unlinked projects remain valid for planning.
        - URL resources allowed when unlinked; filesystem path resources require workspace link.
    - locked import semantics for pre-Phase-11:
        - strict fail on unresolved relative path refs under mapped roots.
        - branch/path divergence resolution deferred to roadmap.
    - clarified architecture direction:
        - SQLite relational model remains primary.
        - single local DB default + project-scoped export/import for sharing.
- [x] 2026-02-22: Final-decision lock in pre-Phase-11 closeout doc
    - updated `PRE_PHASE11_CLOSEOUT_DISCUSSION.md` to mark Section 15.8 as authoritative locked decisions.
    - collapsed confusing execution wording to explicit planning-only semantics for `kan`.
    - added Section 15.9 with only two truly unresolved non-roadmap items for explicit user lock:
        - dependency default scope,
        - repo identity mismatch behavior.
- [x] 2026-02-22: Final unresolved pre-Phase-11 locks captured from user decision
    - dependency default scope locked:
        - same-branch default, cross-branch explicit opt-in.
    - repo identity mismatch behavior locked:
        - warning + explicit user-confirm continue.
    - `PRE_PHASE11_CLOSEOUT_DISCUSSION.md` Section 15.9 updated from unresolved to resolved.
- [x] 2026-02-22: Phase 11 roadmap consensus update (attention signals + branch delta delivery)
    - updated `PLAN.md`:
        - added attention-state roadmap contract (`none|note|unresolved`) with audit metadata fields.
        - added branch-scoped delta delivery contract (`changes_since_cursor` + `active_attention_items`).
        - documented cursor semantics (`agent_id + branch_id`, explicit ack, deterministic resend without ack).
        - clarified `session_id` as optional diagnostic metadata, not correctness-critical.
        - expanded candidate Phase 11 tool surface (`list_branches`, `get_branch_context`, `set_attention_state`, `clear_attention_state`).
        - added Phase 11 open-contract questions for ack shape and unresolved gating policy.
        - locked lifecycle display mapping note (`progress` internal, `In Progress` display).
    - updated `PRE_PHASE11_CLOSEOUT_DISCUSSION.md`:
        - corrected stale dependency wording (`same-branch` default).
        - corrected lifecycle terminology drift (`progress` canonical, `In Progress` display label).
        - added Section 15.10 capturing roadmap-only consensus for attention-state and branch-context delta delivery.
- [x] 2026-02-22: Phase 11/docs lock refresh (delivery defaults + logging policy alignment)
    - updated `PLAN.md`:
        - converted Phase 11 contract questions into locked defaults for cursor/ack and attention behavior.
        - locked one-active-attention-per-item policy and unresolved-by-default completion gating.
        - locked branch-feed scope policy (branch descendants + context-relevant project/config deltas only).
        - added logging policy section:
            - adopt `github.com/charmbracelet/log`,
            - styled/colorized terminal logs,
            - developer local log-file output,
            - post-MCP observability expansion deferred.
    - updated `PRE_PHASE11_CLOSEOUT_DISCUSSION.md`:
        - added Section 15.11 with locked Phase 11 delivery defaults.
        - added Section 15.12 with logging baseline lock (pre-Phase-11 in-scope).
- [x] 2026-02-22: Added explicit Phase 11.0 MCP design gate (docs-only)
    - updated `PLAN.md`:
        - inserted `Phase 11.0` mandatory research/discussion gate before `Phase 11.1`.
        - locked requirement to review `mcp-go`, stateless HTTP serving, hexagonal adapter fit, and dynamic tool discovery/update behavior before implementation.
        - documented dogfooding intent and open contract questions for pre-implementation review.
    - updated `PRE_PHASE11_CLOSEOUT_DISCUSSION.md`:
        - added Section 15.13 locking the same Phase 11.0 gate and reference link.
- [x] 2026-02-22: Logging scope correction + agent policy hardening (docs-only)
    - updated `AGENTS.md`:
        - logging is now mandatory implementation scope with `github.com/charmbracelet/log`.
        - dev mode requires workspace-local `.kan/log/` file logging.
        - troubleshooting requires inspecting local logs.
        - reinforced idiomatic error bubbling/wrapping expectations.
    - updated `PLAN.md`:
        - moved logging baseline from deferred roadmap framing to explicit pre-Phase-11 in-scope requirements.
        - kept broader observability pipeline work deferred until after MCP/HTTP.
- [x] 2026-02-22: Worklog governance lock sync across planning docs (orchestrator docs pass)
    - objective:
        - make execution-ledger ownership and checkpoint evidence requirements explicit across repository docs.
    - files updated:
        - `PLAN.md`
        - `PRE_PHASE11_CLOSEOUT_DISCUSSION.md`
        - `PARALLEL_AGENT_RUNBOOK.md`
        - `AGENTS.md`
    - commands run:
        - `rg -n "Worklog Governance \\(Locked\\)|closeout decision register|Worklog Source-of-Truth Split|test_not_applicable" PLAN.md PRE_PHASE11_CLOSEOUT_DISCUSSION.md PARALLEL_AGENT_RUNBOOK.md AGENTS.md` -> pass
    - tests:
        - `test_not_applicable` (docs-only update; no code-path changes)
    - status:
        - governance wording is now aligned: `PLAN.md` is single execution ledger, closeout file is decision register, and orchestrator/integrator is single writer for checkpoint state.
- [x] 2026-02-22: Wave 3 remediation integration (`L-F`, `L-G`, `L-J`)
    - `L-F` durable activity log:
        - `internal/tui/model.go` now loads persisted project events via `ListProjectChangeEvents`.
        - activity modal surfaces persisted `change_events` and keeps graceful fallback text when fetch fails.
        - tests updated in `internal/tui/model_test.go` for persisted log rendering and failure handling.
    - `L-G` logging baseline:
        - integrated `github.com/charmbracelet/log` in `cmd/kan/main.go`.
        - added dual sink behavior:
            - styled/colorized console logs,
            - dev file logs in workspace-local `.kan/log/`.
        - added TOML logging config surface in `internal/config/config.go` and `config.example.toml`.
        - documented behavior in `README.md`; tests added in `cmd/kan/main_test.go` and `internal/config/config_test.go`.
    - `L-J` cleanup:
        - removed unused `SearchTasks` method from `internal/app/service.go`.
        - updated `internal/app/service_test.go` to use `SearchTaskMatches`.
    - verification:
        - `GOCACHE=$(pwd)/.kan/go-build-cache just test-pkg ./internal/tui` -> pass
        - `GOCACHE=$(pwd)/.kan/go-build-cache just test-pkg ./internal/config` -> pass
        - `GOCACHE=$(pwd)/.kan/go-build-cache just test-pkg ./internal/app` -> pass
        - `GOCACHE=$(pwd)/.kan/go-build-cache just test-pkg ./cmd/kan` -> pass
        - `GOCACHE=$(pwd)/.kan/go-build-cache just test-pkg ./cmd/` -> pass
        - `GOCACHE=$(pwd)/.kan/go-build-cache just ci` -> pass
    - runtime/vhs confirmation:
        - `GOCACHE=$(pwd)/.kan/go-build-cache KAN_DB_PATH=/tmp/kan-run-check.db just run` startup/migration verified with fresh DB.
        - `vhs vhs/board.tape` -> pass
        - `vhs vhs/workflow.tape` -> pass
        - extracted mid-timeline frames with `ffmpeg` from `.artifacts/vhs/*.gif`; confirmed board/help/task modals render correctly (first `> K` frame is expected tape startup).
    - status:
        - pre-Phase-11 closeout blockers from earlier QA pass are resolved.
        - first-run onboarding remains intentionally roadmap-only per locked decision register.
- [x] 2026-02-22: Independent reviewer re-audit (`CR-2`) after Wave 3 integration
    - objective:
        - confirm no hidden quality regressions and verify pre-Phase-11 completion claims against code + docs.
    - findings summary:
        - no high/medium defects reported.
        - durable activity log, reload-config/paths-roots, logging baseline, and `SearchTasks` cleanup verified in code/tests.
        - remaining UX concerns in `PRE_PHASE11_CLOSEOUT_DISCUSSION.md` are treated as backlog/roadmap context, not pre-Phase-11 blockers.
    - status:
        - pre-Phase-11 implementation scope remains closed.
- [ ] 2026-02-22: Re-open pre-Phase-11 remediation from live user QA (orchestrator reset)
    - objective:
        - re-open previously closed pre-Phase-11 scope based on confirmed runtime UX regressions and workflow gaps.
        - execute fixes under single-orchestrator control; only spawn fresh subagents after lock reset and lane scoping.
    - trigger:
        - user manual run screenshots + notes identified blocker regressions not captured by existing VHS coverage.
    - blockers to resolve before re-close:
        - board list behavior:
            - subtasks must not render inline in board rows; board row should show compact progress count (`done/total`) only.
            - selected-row marker semantics must be restored (focused row only), without clutter from global row markers.
            - long column lists must maintain fixed viewport height and scroll with cursor/wheel so focused row stays visible.
        - task info/subtask behavior:
            - subtask visibility must be consistent regardless of parent task column/state (`todo|progress|done|archived`).
            - task info modal should expose subtasks list and allow drill-in edit flow via modal stack.
        - runtime artifact hygiene:
            - `.kan/` runtime logs must not appear under package test directories (`cmd/kan/.kan`).
            - local caches/runtime artifacts must be gitignored appropriately (`.kan/`, `.go-mod-cache/`).
            - dev log location policy needs deterministic behavior for `just run`, tests, and VHS.
        - verification assets:
            - refresh VHS coverage to include scrolling/selection and subtask modal expectations.
            - regenerate `TUI_MANUAL_TEST_WORKSHEET.md` with machine-readable note anchors:
                - `### USER NOTES Sx.y-Nz`
    - orchestrator policy for this wave:
        - `PLAN.md` remains single-writer worklog.
        - no stale/orphan subagent sessions reused; only newly spawned lanes after baseline update.
    - lane orchestration evidence:
        - attempted fresh worker spawn for runtime hygiene lane (`L-RH1`) failed due environment agent-capacity limit (`max 6`).
        - fallback policy activated: orchestrator executes lane work directly until fresh capacity is available.
    - status:
        - in progress (docs baseline updated; code remediation pending).
    - progress update (2026-02-22, orchestrator):
        - rewrote `TUI_MANUAL_TEST_WORKSHEET.md` with strict machine-readable anchors in every section:
            - `### USER NOTES Sx.y-Nz`
        - added targeted regression VHS scenarios:
            - `vhs/regression_subtasks.tape`
            - `vhs/regression_scroll.tape`
        - regenerated and reviewed artifacts:
            - `.artifacts/vhs/regression_subtasks.gif`
            - `.artifacts/vhs/regression_scroll.gif`
            - frame samples under `.artifacts/vhs/review/regression_subtasks-*.png` and `.artifacts/vhs/review/regression_scroll-*.png`
        - confirmed captures show:
            - board rows hide subtasks and show compact `done/total` metadata on parent row,
            - task-info modal still shows subtasks after parent move to `In Progress`,
            - constrained-height list scroll follows selection.
        - refreshed docs:
            - `README.md` now explicitly documents human↔agent collaboration intent and manual-test anchor workflow.
        - fixed key-behavior gap from user QA:
            - `[`/`]` now route to bulk move when multi-select is active, and keep single-task behavior when no selection set exists.
            - added regression test:
                - `TestModelBulkMoveKeysUseSelection` in `internal/tui/model_test.go`.
    - verification log (2026-02-22, orchestrator):
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/tui` -> pass (cached)
        - `GOCACHE=$(pwd)/.go-cache just test-pkg ./cmd/kan` -> pass (cached)
        - `just vhs vhs/regression_subtasks.tape` -> pass
        - `just vhs vhs/regression_scroll.tape` -> pass
        - `GOCACHE=$(pwd)/.go-cache just ci` -> pass
        - note: `just ci` emitted a non-fatal module stat-cache write warning under sandboxed filesystem permissions; command still exited 0 and coverage/build gates passed.
        - post-fix rerun:
            - `GOCACHE=$(pwd)/.go-cache just test-pkg ./internal/tui` -> pass
            - `GOCACHE=$(pwd)/.go-cache just ci` -> pass
    - progress update (2026-02-22, orchestrator):
        - user requested new development-environment helpers in `Justfile`.
        - added descriptive comments for all recipe blocks so `just --list` shows clear intent.
        - added new recipes:
            - `init-dev-config`: create/copy dev config at resolved `./kan --dev paths` config path when missing.
            - `clean-dev`: remove resolved dev data directory returned by `./kan --dev paths`.
    - verification log (2026-02-22, orchestrator):
        - `just --list` -> pass; recipe comments rendered as expected.
        - `just --dry-run init-dev-config` -> pass; command expansion validated.
        - `just --dry-run clean-dev` -> pass; command expansion validated.
    - progress update (2026-02-22, orchestrator):
        - user runtime report: `just run` failed after `just init-dev-config` with:
            - `load config ".../config.toml": database path is required`
        - root cause:
            - `config.example.toml` intentionally uses `database.path = ""`,
            - `config.Load` merged TOML over defaults and treated blank `database.path` as an explicit empty override.
        - remediation:
            - `internal/config/config.go`: preserve resolved default database path when TOML `database.path` is blank.
            - `internal/config/config_test.go`: added `TestLoadBlankDatabasePathFallsBackToDefault`.
            - `Justfile`: fixed shell expansion in `init-dev-config`/`clean-dev` (use shell `$...` instead of `$$...`), which previously produced PID-prefixed paths such as `16116cfg`.
    - verification log (2026-02-22, orchestrator):
        - `just test-pkg ./internal/config` -> pass
        - `just test-pkg ./cmd/kan` -> pass
        - user-reported confirmation:
            - `just init-dev-config` created `/Users/evanschultz/Library/Application Support/kan-dev/config.toml`
            - `just ci` -> pass
    - progress update (2026-02-22, orchestrator, Wave A kickoff):
        - objective:
            - begin implementation of remaining pre-Phase-11 worksheet/user-note remediations with parallel lanes where lock scopes allow.
        - command/test evidence:
            - `ls -la .kan/log` -> pass (`.kan/log/kan-20260222.log` present)
            - `tail -n 120 .kan/log/kan-20260222.log` -> pass
                - finding: clean startup/migration/TUI loop lifecycle events; no runtime failure signatures in recent log tail.
            - Context7 consults (required pre-edit):
                - `resolve-library-id` for `github.com/charmbracelet/bubbles` -> pass (`/charmbracelet/bubbles`)
                - `resolve-library-id` for `github.com/charmbracelet/bubbletea` -> pass (`/charmbracelet/bubbletea`)
                - `query-docs` `/charmbracelet/bubbles` (textinput suggestions + list behavior) -> pass
                - `query-docs` `/charmbracelet/bubbletea` (modal/key update patterns) -> pass
        - lane orchestration:
            - spawned `L-PA` (startup/default-project behavior lane):
                - lock: `cmd/kan/main.go`, `cmd/kan/main_test.go`, `internal/app/service.go`, `internal/app/service_test.go`
            - spawned `L-TUIA` (Wave A TUI behavior lane):
                - lock: `internal/tui/model.go`, `internal/tui/model_test.go` (+ `internal/tui/keymap_test.go`/`internal/tui/model_teatest_test.go` if required)
        - status:
            - in progress (awaiting worker handoffs for integration and gate verification).
    - progress update (2026-02-23, orchestrator, Wave A integration + remediation):
        - integrated worker handoffs and completed direct follow-up edits for worksheet-driven UX gaps.
        - startup/default-project behavior:
            - removed startup auto-create/default-project path so empty DB no longer silently creates `Inbox`.
            - first-run empty state now auto-opens `New Project` modal when no projects exist.
        - task/project flow fixes:
            - task creation now returns and applies `focusTaskID` so cursor follows newly created tasks.
            - project creation/edit now supports `root_path` field and persists via project-root callback.
            - new-project refresh now applies `pendingProjectID` correctly so stale prior-project tasks are not shown.
        - command/search/actions fixes:
            - command palette `search-all` and `search-project` now open search mode with scope applied.
            - command/quick-actions overlays use windowed rendering when list exceeds modal height.
            - quick actions are now state-aware, enabled actions sort first, disabled actions render with reason and are blocked from execution.
        - task detail/form UX fixes:
            - `esc` in task info now steps back to parent before closing.
            - `esc` in normal board mode now clears subtree focus when active.
            - focused row styling now uses fuchsia (`212`) and focused+selected cues remain visible.
            - task due signals now show in board/task-info contexts (`!YYYY-MM-DD` + warning line).
            - due picker now includes datetime presets (`today 17:00 UTC`, `tomorrow 09:00 UTC`).
            - label autocomplete accept shortcut added (`ctrl+y`) and hints updated.
            - task form suggestions now merge inherited + project-observed labels for deterministic autocomplete acceptance.
            - resource picker hint copy reduced to single attach action wording (`enter/a` semantics).
        - domain rule hardening:
            - `MoveTask` completion validation now blocks moving to `done` when any subtask remains incomplete.
    - failure/remediation log (2026-02-23, orchestrator):
        - `GOCACHE=$(pwd)/.go-cache-tui just test-pkg ./internal/tui` failed:
            - compile error: `projectAccentColor` returned `lipgloss.Color` type.
            - remediation:
                - updated signature to `color.Color`.
            - Context7 re-consult (required after failure):
                - `/charmbracelet/lipgloss` color typing/usage.
        - `GOCACHE=$(pwd)/.go-cache-tui just test-pkg ./internal/tui` failed:
            - failing tests:
                - `TestModelNoProjectsBootstrapsProjectForm` (overlay assertion too strict),
                - `TestTaskFormCtrlYAcceptsLabelSuggestion` (suggestion source mismatch).
            - remediation:
                - aligned first-run test with modal-first rendering expectation,
                - added robust `isCtrlY` detection,
                - merged task-form suggestion sources (inherited + project-observed) for autocomplete.
            - Context7 re-consults (required after failures):
                - `/charmbracelet/bubbletea` key handling (`ctrl+` inputs),
                - `/charmbracelet/bubbles` textinput suggestions behavior.
        - `GOCACHE=$(pwd)/.go-cache-ci just ci` failed:
            - `coverage` recipe awk parser was non-portable and misparsed output.
            - remediation:
                - hardened `Justfile` coverage awk to parse only `^ok` coverage lines with POSIX-safe `sub(...)` extraction.
            - Context7 re-consult (required after failure):
                - `/casey/just` cross-platform recipe guidance.
    - verification log (2026-02-23, orchestrator):
        - `GOCACHE=$(pwd)/.go-cache-tui just test-pkg ./internal/tui` -> pass
        - `GOCACHE=$(pwd)/.go-cache-app just test-pkg ./internal/app` -> pass
        - `GOCACHE=$(pwd)/.go-cache-cmd just test-pkg ./cmd/kan` -> pass
        - `GOCACHE=$(pwd)/.go-cache-ci just ci` -> pass
            - note: non-fatal Go stat-cache permission warning emitted under sandboxed module cache write path; command exited 0.
    - status:
        - in progress (Wave A fixes integrated; remaining worksheet-note closeout planning pending for unresolved product-scope items).
    - progress update (2026-02-23, orchestrator, Wave B closeout integration):
        - objective:
            - close remaining worksheet-driven pre-Phase-11 UX/behavior gaps and regenerate a clean manual retest worksheet.
        - Context7 + docs evidence (required pre-edit):
            - `resolve-library-id` for `charmbracelet bubbletea` -> pass (`/charmbracelet/bubbletea`)
            - `query-docs` `/charmbracelet/bubbletea` (key handling for ctrl/esc/back semantics) -> pass
            - `resolve-library-id` for `pelletier go-toml v2` -> no direct match returned by Context7 catalog
            - fallback source recorded per policy:
                - `go doc github.com/pelletier/go-toml/v2.Marshal` -> pass
                - `go doc github.com/pelletier/go-toml/v2.Unmarshal` -> pass
        - implementation updates:
            - `internal/tui/model.go`:
                - task-info mode now supports:
                    - `s` to open subtask form for the current task,
                    - `[`/`]` to move the currently focused task/subtask left/right from within task-info.
                - task-info hints/mode prompt updated for discoverability (`s subtask`, `[/] move`).
                - resource-picker hint copy tightened for clearer `enter/a` semantics.
            - `internal/tui/model_test.go`:
                - added regression coverage:
                    - `TestModelCommandPaletteFuzzyAbbreviationExecutesNewSubtask`
                    - `TestModelLabelsConfigCommandSave`
                    - `TestModelTaskInfoAllowsSubtaskCreation`
                    - `TestModelTaskInfoMovesCurrentTaskWithBrackets`
            - `internal/config/config_test.go`:
                - added `UpsertAllowedLabels` persistence and validation coverage:
                    - write/update/clear behavior,
                    - missing-file clear noop,
                    - invalid input rejection.
            - `cmd/kan/main_test.go`:
                - added `TestPersistAllowedLabelsRoundTrip` to verify CLI persistence helper wiring.
            - `TUI_MANUAL_TEST_WORKSHEET.md`:
                - fully rewritten as a clean pre-Phase-11 retest worksheet:
                    - preserved machine-readable anchors (`### USER NOTES Sx.y-N1`),
                    - removed stale historical note clutter,
                    - added explicit checks for:
                        - first-run no-default-project behavior,
                        - due datetime + warnings,
                        - dependency/task-info/subtask flows,
                        - create/edit resource attach,
                        - fuzzy command abbreviations,
                        - labels-config + project-root picker workflows.
    - verification log (2026-02-23, orchestrator, Wave B):
        - `just fmt` -> pass
        - package lanes (parallel):
            - `GOCACHE=$(pwd)/.go-cache-config just test-pkg ./internal/config` -> pass
            - `GOCACHE=$(pwd)/.go-cache-cmd just test-pkg ./cmd/kan` -> pass
            - `GOCACHE=$(pwd)/.go-cache-app just test-pkg ./internal/app` -> pass
            - `GOCACHE=$(pwd)/.go-cache-domain just test-pkg ./internal/domain` -> pass
            - `GOCACHE=$(pwd)/.go-cache-tui just test-pkg ./internal/tui` -> pass
        - full gate:
            - `GOCACHE=$(pwd)/.go-cache-ci just ci` -> pass
            - note: non-fatal Go module stat-cache permission warning surfaced under sandbox restrictions; command exited 0 and all gates passed.
    - status:
        - in progress (implementation + worksheet refresh complete; awaiting user manual worksheet run for final closeout sign-off).
    - progress update (2026-02-23, orchestrator, Wave C visual polish + runtime highlight control):
        - objective:
            - address live QA feedback on focused-row marker rendering and provide command-palette control for highlight color.
        - Context7 evidence (required pre-edit):
            - `resolve-library-id` `charmbracelet/lipgloss` -> pass (`/charmbracelet/lipgloss`)
            - `resolve-library-id` `charmbracelet/bubbletea` -> pass (`/charmbracelet/bubbletea`)
            - `query-docs` `/charmbracelet/lipgloss` (dynamic foreground style patterns) -> pass
            - `query-docs` `/charmbracelet/bubbletea` (input-mode key handling patterns) -> pass
        - implementation updates:
            - `internal/tui/model.go`:
                - removed duplicate marker rendering on secondary/meta lines (marker now title-line only).
                - added runtime-configurable focused-row highlight color model state with default `212`.
                - added command-palette command:
                    - `highlight-color` (aliases: `set-highlight`, `focus-color`).
                - added highlight-color modal/input mode (`enter` save, `esc` cancel, blank resets to default).
                - selected-row styles now use configured highlight color instead of hardcoded `212`.
            - `internal/tui/model_test.go`:
                - added `TestModelCommandPaletteHighlightColorApplies`.
                - added `TestModelSelectionMarkerOnlyOnTitleLine`.
            - `internal/tui/testdata/TestModelGoldenBoardOutput.golden` and `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`:
                - refreshed to match marker-on-title-only rendering.
            - `TUI_MANUAL_TEST_WORKSHEET.md`:
                - updated section checks for:
                    - single marker on title row only,
                    - command-palette `highlight-color` runtime verification.
        - failure/remediation log:
            - `just test-pkg ./internal/tui` initially failed due sandbox cache permission (`~/Library/Caches/go-build` write denied).
            - reran with approval outside sandbox; tests then failed only on expected golden snapshots after marker behavior change.
            - mandatory Context7 re-consult completed after failed test run.
            - resolved by running `just test-golden-update` and re-running `just test-pkg ./internal/tui`.
    - verification log (2026-02-23, orchestrator, Wave C):
        - `just fmt` -> pass
        - `just test-golden-update` -> pass
        - `just test-pkg ./internal/tui` -> pass
        - `just ci` -> pass
    - status:
        - in progress (code + worksheet updated; awaiting full user worksheet pass to close re-opened pre-Phase-11 remediation item).
- [x] 2026-02-23: Focused multi-select marker polish (star-only indicator)
    - objective:
        - remove non-focused multi-select row background highlight so star markers are the only multi-select indicator.
    - files updated:
        - `internal/tui/model.go`
    - implementation notes:
        - changed non-focused multi-select style from filled background to no additional styling.
        - focused/selected behavior remains unchanged (`│*` marker + focused highlight color).
    - commands/tests:
        - `just test-pkg ./internal/tui` -> initial run blocked by sandbox cache permission.
        - reran with approval outside sandbox:
            - `just test-pkg ./internal/tui` -> pass.
- [x] 2026-02-23: TUI runtime logs file-only through shutdown
    - objective:
        - prevent runtime log lines from printing into the TUI/terminal on close; keep TUI-mode runtime logging in `.kan/log/*.log`.
    - Context7 evidence (required pre-edit):
        - `resolve-library-id` for `github.com/charmbracelet/log` -> pass (`/charmbracelet/log`)
        - `query-docs` `/charmbracelet/log` (output sink control and logger output configuration) -> pass
    - files updated:
        - `cmd/kan/main.go`
        - `cmd/kan/main_test.go`
    - implementation notes:
        - guarded close-time warning emission so terminal output remains muted whenever the TUI command disables the console sink.
        - added regression test to assert TUI mode writes runtime lifecycle events to dev log file while `stderr` remains empty.
    - commands/tests:
        - `just fmt` -> pass
        - `just test-pkg ./cmd/kan` -> initial run blocked by sandbox cache permission (`~/Library/Caches/go-build` write denied)
        - reran with approval outside sandbox:
            - `just test-pkg ./cmd/kan` -> pass
- [x] 2026-02-23: Task-info checklist completion + due-time clarity sweep
    - objective:
        - resolve user-reported pre-Phase-11 UX gaps for task-info/subtask completion controls, focused state visibility, and due-time entry discoverability.
    - runtime/log review:
        - `tail -n 120 .kan/log/kan-20260223.log` -> pass; recent entries show clean startup/command-loop/project-root lifecycle events with no crash signatures.
    - Context7 evidence (required pre-edit):
        - `resolve-library-id` + `query-docs` for:
            - `github.com/charmbracelet/bubbles` (`/charmbracelet/bubbles`) checklist/list styling guidance.
            - `github.com/charmbracelet/lipgloss` (`/charmbracelet/lipgloss`) list item style composition for focused metadata rows.
            - `github.com/charmbracelet/bubbletea` (`/charmbracelet/bubbletea`) key handling patterns (`msg.String()`/space handling).
    - files updated:
        - `internal/tui/model.go`
        - `internal/tui/model_test.go`
        - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`
        - `TUI_MANUAL_TEST_WORKSHEET.md`
    - implementation notes:
        - task-info modal:
            - subtasks now render checklist-style rows (`[ ]` / `[x]`) with explicit per-row `state` + `complete` metadata.
            - added `space` completion toggle in task-info for focused subtask (done <-> active column).
            - updated task-info shortcuts/hints to clarify movement as `[ or ]` and document checklist toggle.
        - state visibility:
            - task-info header now shows focused item lifecycle state and completion flag (`complete: yes/no`).
            - board-side task details now include lifecycle state metadata.
        - due-time discoverability:
            - due-field hints now explicitly document accepted date/time formats (including `RFC3339`) and UTC default behavior.
            - task-info adds explicit due-time input guidance.
        - test realism:
            - fake service `MoveTask` now updates `LifecycleState` based on destination column naming to mirror app behavior in TUI tests.
        - worksheet refresh:
            - updated manual checks under existing anchors (`S2.3`, `S3.1`, `S3.2`, `S3.3`, `S9.1`, `S10.1`) for checklist toggling, state visibility, due format hints, and `[ or ]` wording.
    - failure/remediation log:
        - `just test-pkg ./internal/tui` initial run blocked by sandbox cache permission; reran with approval.
        - rerun surfaced failures:
            - golden help snapshot drift due intentional `[ or ]` hint wording update.
            - checklist toggle test no-op from space key string mismatch.
            - board subtask-progress test mismatch after lifecycle resolution change.
        - mandatory Context7 re-consult completed after failed test run (Bubble Tea key string handling for spacebar).
        - remediation:
            - handled both `" "` and `"space"` in task-info toggle branch.
            - restored lifecycle precedence to explicit task state (column fallback only when state is empty).
            - updated fake-service move lifecycle mapping and refreshed golden output.
    - verification log:
        - `just fmt` -> pass
        - `just test-golden-update` -> pass
        - `just test-pkg ./internal/tui` -> pass
        - `just ci` -> pass
- [x] 2026-02-23: Dependency/blocker inspector modal deepening + worksheet sync
    - objective:
        - ensure dependency/blocker modal supports inspect + add/remove + jump workflows with linked refs always visible (including archived/missing refs), and document the flow for manual verification.
    - Context7 evidence (required pre-edit + post-failure re-consults):
        - `resolve-library-id` for `charm bubbles` -> pass (`/charmbracelet/bubbles`)
        - `query-docs` `/charmbracelet/bubbles` (list/delegate/filter/key patterns) -> pass
        - `query-docs` `/charmbracelet/bubbletea` (key handling/testing and shortcut behavior) -> pass; re-consulted after each failing test/coverage iteration.
    - files updated:
        - `internal/tui/model.go`
        - `internal/tui/model_test.go`
        - `TUI_MANUAL_TEST_WORKSHEET.md`
    - implementation notes:
        - dependency inspector data loading:
            - linked refs from `depends_on`/`blocked_by` are now pinned at the top of the modal list.
            - linked refs remain visible even when archived or filtered out of search matches.
            - unresolved/missing linked IDs render as explicit placeholder rows so they can be inspected/removed.
        - dependency inspector rendering:
            - added linked-ref pinning hint in modal.
            - details pane now falls back to match `state_id` (including `missing`) when lifecycle state is empty.
        - manual worksheet:
            - section `2.1` now validates `ctrl+o` dependency picker flow plus CSV fallback.
            - section `3.1` explicitly validates `b` dependency inspector with pinned refs, details, add/remove, and jump-to-task.
            - section `10.1` includes dependency/blocker modal workflow regression expectation.
        - regression coverage:
            - added dependency inspector tests for pinned linked refs, archived/missing visibility, task-info jump/apply flow, form `ctrl+o` integration, filter controls, and state-id helper behavior.
    - failure/remediation log:
        - `just test-pkg ./internal/tui` initially failed in new tests due:
            - batch-load assumptions in tests,
            - reserved key collisions in query typing,
            - filter-state sequence assumptions for list toggles.
        - remediations:
            - test flows now explicitly apply dependency-load msg where needed,
            - switched query test input to non-conflicting keys,
            - restored list-eligible filter state before list-toggle assertions.
        - `just ci` initially failed coverage gate:
            - `internal/tui` at `67.3%`, then `69.7%`.
        - remediation:
            - expanded dependency-inspector branch coverage until `internal/tui` reached `71.2%`.
    - verification log:
        - `just fmt` -> pass
        - `just test-pkg ./internal/tui` -> pass
        - `just ci` -> pass
- [x] 2026-02-23: Dependency self-reference guard
    - objective:
        - prevent a task from depending on or being blocked by itself in dependency inspector flows.
    - files updated:
        - `internal/tui/model.go`
        - `internal/tui/model_test.go`
    - implementation notes:
        - added dependency-id sanitization to strip owner task ID when opening inspector and when applying/saving.
        - excluded owner task ID from pinned linked refs as well as search results.
        - added explicit key-path guards (`d`/`b`/active toggle) to block self-selection if encountered.
    - regression coverage:
        - extended `TestModelDependencyInspectorPinsLinkedRefsAndAppliesEdits` to assert owner ID is excluded from candidate list and stripped on save.
    - verification log:
        - `just fmt` -> pass
        - `just test-pkg ./internal/tui` -> pass
        - `just ci` -> pass
- [x] 2026-02-23: Dependency/search modal key-routing hardening
    - objective:
        - fix dependency inspector key conflicts so query typing does not trigger action shortcuts, constrain relation toggles to list rows, and align `j/k` + arrows with focus navigation behavior.
    - Context7 evidence (required pre-edit + post-failure re-consults):
        - `query-docs` `/charmbracelet/bubbletea` for key-routing/focus-handling patterns -> pass
        - re-consulted after failing test iterations per policy.
    - files updated:
        - `internal/tui/model.go`
        - `internal/tui/model_test.go`
    - implementation notes:
        - dependency inspector:
            - `x`/`a` shortcuts are ignored while query input is focused (`dependencyFocus == 0`) so those keys type into search.
            - `d` and `b` now execute only when list focus is active (`dependencyFocus == 4`).
            - `j/k` and up/down now move modal focus like tab/backtab; when list focus is active they navigate list rows.
        - search modal:
            - `j/k` and up/down now move focus between controls instead of inserting query characters.
    - regression coverage:
        - added `TestModelDependencyInspectorInputAndListKeyRouting`.
        - added `TestModelSearchFocusNavigationWithJK`.
        - adjusted `TestModelDependencyInspectorFilterControls` for updated focus/shortcut behavior.
    - failure/remediation log:
        - initial `just test-pkg ./internal/tui` failures came from stale test assumptions after shortcut scope changes.
        - remediated by updating test focus sequencing and query/list setup.
    - verification log:
        - `just fmt` -> pass
        - `just test-pkg ./internal/tui` -> pass
        - `just ci` -> pass
- [x] 2026-02-23: Pre-MCP consensus lock captured in executable wave plan
    - objective:
        - convert latest user-locked consensus (startup picker-first behavior, first-run config bootstrap, root-search UX, markdown description/comments with ownership, glamour full-screen thread view) into a new pre-MCP execution document with parallel subagent waves.
    - runtime/log review:
        - `ls -1t .kan/log | head -n 3` -> pass (`kan-20260223.log` present)
        - `tail -n 120 .kan/log/kan-20260223.log` -> pass; no crash signatures in recent startup/shutdown/config update lifecycle lines.
    - Context7 evidence (planning input for external library contract):
        - `resolve-library-id` for `github.com/charmbracelet/glamour` -> pass (`/charmbracelet/glamour`)
        - `query-docs` `/charmbracelet/glamour` (renderer creation, style options, word wrap, ANSI output integration) -> pass
        - `query-docs` `/charmbracelet/glamour` (reusable renderer patterns, JSON/custom style options) -> pass
    - files updated:
        - `PRE_MCP_EXECUTION_WAVES.md`
        - `PLAN.md`
    - implementation notes:
        - added a locked pre-MCP execution spec with:
            - startup-first project picker contract (always opens; always offers `New Project`),
            - first-run bootstrap ordering (identity + global root-search config before picker),
            - root-search and project-root UX requirements,
            - markdown description + ownership-attributed comment model across project/phase/task/subtask variants,
            - glamour-based full-screen description/thread rendering contract,
            - 7-wave parallel execution breakdown with lane scopes, acceptance criteria, and subagent handoff requirements.
    - verification log:
        - `test_not_applicable`: docs/process-only checkpoint; no Go/runtime behavior changed in this step, so package tests were not run.
- [ ] 2026-02-23: Pre-MCP implementation kickoff (subagent orchestration)
    - objective:
        - implement locked pre-MCP consensus in code using parallel worker lanes:
            - first-run identity + global root-search bootstrap,
            - picker-first launch + always-available in-picker project creation,
            - ownership-attributed markdown comments persisted across project/work-item targets,
            - full-screen glamour-rendered description/comment thread with compose input.
    - runtime/log review:
        - `ls -1t .kan/log | head -n 3` -> pass (`kan-20260223.log` present)
        - `tail -n 120 .kan/log/kan-20260223.log` -> pass (no crash signatures in recent startup/shutdown logs).
    - Context7 evidence (required pre-edit):
        - `resolve-library-id` `charmbracelet/bubbletea` -> pass (`/charmbracelet/bubbletea`)
        - `query-docs` `/charmbracelet/bubbletea` (modal state/key routing patterns) -> pass
        - `resolve-library-id` `charmbracelet/glamour` -> pass (`/charmbracelet/glamour`)
        - `query-docs` `/charmbracelet/glamour` (renderer lifecycle, style/width options) -> pass
        - `resolve-library-id` `github.com/pelletier/go-toml/v2` -> unavailable in Context7 result set
    - fallback source evidence (required when Context7 unavailable):
        - `go doc github.com/pelletier/go-toml/v2` -> pass (fallback API/source reference for config-table read/write helpers).
    - discovery command evidence:
        - `sed -n '1,260p' Justfile` -> pass (just recipes confirmed as canonical test/build gate)
        - `sed -n '1,260p' PLAN.md` -> pass
        - `sed -n '1,260p' PRE_PHASE11_CLOSEOUT_DISCUSSION.md` -> pass
        - `sed -n '1,260p' TUI_MANUAL_TEST_WORKSHEET.md` -> pass
        - explorer lane reports (startup flow, config/bootstrap gaps, comments/storage gaps) -> pass
    - lane lock plan (single-branch, lock discipline):
        - `W11-A` (config + startup bootstrap + picker-first routing):
            - lock scope: `internal/config/*.go`, `cmd/kan/main.go`, `cmd/kan/main_test.go`, `internal/tui/options.go`, `internal/tui/model.go`, `internal/tui/model_test.go`, `config.example.toml`
            - hotspot ownership in this lane: `internal/tui/model.go`
        - `W11-B` (comments domain/app/storage):
            - lock scope: `internal/domain/*.go`, `internal/app/*.go`, `internal/adapters/storage/sqlite/*.go`
            - hotspot ownership in this lane: `internal/app/service.go`, `internal/adapters/storage/sqlite/repo.go`
        - `W11-C` (TUI markdown thread/glamour integration + tests + worksheet updates):
            - lock scope: `internal/tui/*.go`, `internal/tui/testdata/*.golden`, `TUI_MANUAL_TEST_WORKSHEET.md`
            - starts after `W11-A`+`W11-B` integration because it depends on new runtime config + comments API.
    - status:
        - in progress (pre-edit orchestration complete; next step is worker-lane execution and integration checkpoints).
    - progress update (2026-02-23, orchestrator, lane handoff integration before W11-C):
        - `W11-B` handoff ingested:
            - comments domain + app + sqlite persistence landed (`internal/domain/comment.go`, `internal/app/service.go`, `internal/adapters/storage/sqlite/repo.go` + tests).
            - worker evidence:
                - `just test-pkg ./internal/domain` -> pass
                - `just test-pkg ./internal/app` -> pass
                - `just test-pkg ./internal/adapters/storage/sqlite` -> pass
        - `W11-A` handoff ingested:
            - startup bootstrap + picker-first launch + in-picker new-project flow + config identity/search-roots support landed.
            - worker reported intermediate `internal/tui` failures during migration to picker-first tests, then applied option-driven launch picker fix.
            - orchestrator verification rerun:
                - `just test-pkg ./internal/tui` -> pass
        - status:
            - ready to execute `W11-C` (full-screen glamour markdown thread + comment compose + ownership rendering in TUI) on top of integrated lanes.
    - progress update (2026-02-23, orchestrator, post-user CI feedback + next execution order lock):
        - user-provided gate evidence:
            - `just ci` failed in `./internal/tui` with:
                - `TestModelThreadModeProjectAndPostCommentUsesConfiguredIdentity`
                - `TestModelCommandPaletteWindowedRendering`
        - locked execution order requested by user:
            1. update markdown trackers with current state + remaining work.
            2. fix failing tests and re-confirm `just ci`.
            3. commit checkpoint (user will push).
            4. use subagents to finish remaining UX scope:
                - first-launch inputs must be in-TUI modal flow (visual style aligned with existing modal system; no CLI stdin prompt flow),
                - search-root path selection must be fuzzy and easy to navigate/add,
                - no `huh` dependency; use Bubble Tea/Bubbles/Lip Gloss v2 only,
                - separate lane to update `README.md` and incorporate `fang` usage context added by user.
            5. after implementation + passing `just ci`, create a new targeted manual worksheet focused only on newly changed/fixed areas since the prior worksheet pass.
        - immediate next step:
            - apply minimal code fixes for the two failing TUI tests, then rerun `just ci`.
- [x] 2026-02-23: Ordered checkpoint A (docs update + CI regression fixes)
    - objective:
        - satisfy user-ordered pre-implementation sequence:
            1) update markdown trackers,
            2) clear latest TUI regressions,
            3) re-confirm full `just ci` gate,
            4) commit checkpoint before next subagent wave.
    - files updated (this checkpoint):
        - `PLAN.md`
        - `PRE_MCP_EXECUTION_WAVES.md`
        - `internal/tui/thread_mode.go`
        - `internal/tui/model_test.go`
    - implementation notes:
        - documented current state + remaining required UX scope in planning docs.
        - fixed thread-mode in-memory comment append regression by ensuring thread-load command runs as the primary returned command while still setting composer focus state.
        - hardened command-palette windowed rendering test expectation to assert the currently selected command row dynamically (resilient to command-list growth).
    - verification log:
        - `just test-pkg ./internal/tui` -> pass
        - `just ci` -> pass
    - next step:
        - commit checkpoint, then launch subagent waves for remaining UX scope (TUI first-run modal bootstrap, fuzzy root picker UX, README/Fang lane), followed by new targeted manual worksheet creation.
- [x] 2026-02-23: Windows search-root assertion portability fix (targeted)
    - objective:
        - fix `windows-latest` CI failures in `cmd/kan` caused by OS-specific path separator assertions in search-root tests.
    - Context7 evidence (required pre-edit):
        - `resolve-library-id` `go standard library filepath` -> pass (`/websites/pkg_go_dev_go1_25_3`)
        - `query-docs` `/websites/pkg_go_dev_go1_25_3` (filepath.Clean cross-platform behavior) -> pass
    - files updated:
        - `cmd/kan/main_test.go`
    - implementation notes:
        - updated two search-root assertions to compare against `filepath.Clean("/tmp/code")` and `filepath.Clean("/tmp/docs")` instead of hardcoded `/tmp/...` strings.
    - command/test evidence:
        - `just test-pkg ./cmd/kan` -> fail
            - failure observed at `TestRunBootstrapPromptsAndPersistsMissingFields` with empty persisted display name.
        - `git status --short` -> pass; concurrent in-flight edits present in:
            - `README.md`
            - `cmd/kan/main.go`
            - `cmd/kan/main_test.go`
            - `internal/tui/model.go`
            - `internal/tui/options.go`
        - `git diff --stat` -> pass; substantial concurrent diff detected, likely affecting bootstrap test flow.
        - `just test-pkg ./cmd/kan` (rerun after concurrent updates settled) -> pass
        - `just ci` -> pass
    - status:
        - complete; portability assertion fix verified with repository gate.
- [x] 2026-02-23: Ordered checkpoint B (subagent completion wave: native bootstrap modal + fuzzy roots + README/Fang + delta worksheet)
    - objective:
        - complete remaining user-requested UX/docs scope after checkpoint A commit:
            - replace startup stdin bootstrap prompts with native TUI modal flow,
            - make global search-root setup/editing fuzzy and low-friction,
            - update README with accurate current behavior and Fang context,
            - produce a new targeted manual worksheet for only changed/fixed areas.
    - subagent lane results:
        - `W12-DE` (cmd+tui startup/bootstrap/root UX) -> pass handoff
            - key changes:
                - removed `cmd/kan` stdin bootstrap prompt flow,
                - introduced TUI `modeBootstrapSettings` (mandatory on first run when required fields missing),
                - added bootstrap save callback wiring in `cmd/kan/main.go` and TUI options,
                - added fuzzy root add/remove flow via resource picker integration,
                - added command-palette command `bootstrap-settings` (`setup`, `identity-roots`).
            - worker verification:
                - `just test-pkg ./cmd/kan` -> pass
                - `just test-pkg ./internal/tui` -> pass
        - `W12-F` (README/Fang docs lane) -> pass handoff
            - key changes:
                - `README.md` updated for picker-first startup, bootstrap identity/search roots, thread mode/ownership details, and accurate Fang status/context.
            - verification:
                - `test_not_applicable` (docs-only lane)
    - integrator verification:
        - `just test-pkg ./cmd/kan` -> pass
        - `just test-pkg ./internal/tui` -> pass
        - `just ci` -> pass
    - docs/worksheet outputs:
        - created `TUI_MANUAL_TEST_WORKSHEET_DELTA_BOOTSTRAP_THREADS.md` for post-change targeted manual validation.
    - status:
        - complete for this ordered wave; ready for user manual delta pass.
- [x] 2026-02-23: First-run actor scope correction + resource-picker parent navigation UX fix
    - objective:
        - fix two pre-manual-test launch issues requested by user:
            1) keep startup onboarding actor as human (`user`) while preserving `agent|system` ownership support elsewhere,
            2) make `ctrl+r` browse/path-picker flows show current directory path and allow traversal to higher directories.
    - Context7 evidence (required):
        - pre-edit consult:
            - `resolve-library-id` `charmbracelet/bubbletea` -> pass (`/charmbracelet/bubbletea`)
            - `query-docs` `/charmbracelet/bubbletea` (key-routing/update handler patterns for modal/input coexistence) -> pass
        - failure-triggered re-consult:
            - `resolve-library-id` `Go standard library filepath` -> pass (`/websites/pkg_go_dev_go1_25_3`)
            - `query-docs` `/websites/pkg_go_dev_go1_25_3` (parent navigation/path semantics context) -> pass
    - files updated:
        - `internal/tui/model.go`
        - `internal/tui/options.go`
        - `internal/tui/model_test.go`
        - `cmd/kan/main.go`
        - `cmd/kan/main_test.go`
    - implementation notes:
        - actor handling:
            - restored full actor support (`user|agent|system`) for runtime identity/ownership attribution.
            - mandatory startup bootstrap now always persists `default_actor_type = user`.
            - startup modal keeps actor row visible but locks actor mutation during mandatory onboarding.
        - resource picker/path UX:
            - removed root-bound clamping in `listResourcePickerEntries`; picker can navigate to higher parent directories.
            - picker modal now renders absolute `current:` directory path in header (shared by all path-picker backflows).
            - on directory loads, default selection skips `..` when present to avoid accidental parent selection.
            - for root-selection flows (`add/edit project`, `paths/roots`, `bootstrap settings`), `a` now chooses the current open directory explicitly.
    - failure/remediation log:
        - `just test-pkg ./cmd/kan` -> fail
            - `TestRunBootstrapModalPersistsMissingFields` expected selected root, got parent directory after introducing `..` row.
            - remediation: root-selection `a` behavior changed to choose current directory.
        - `just test-pkg ./internal/tui` -> fail
            - `TestModelBootstrapSettingsCommandPaletteRootsEditing` parent directory selected unexpectedly.
            - `TestModelResourcePickerAttachFromTaskInfoAndEdit` attached `local_dir` due shifted default index with `..` row.
            - remediation: skip default selection over `..` and adjust root-selection attach behavior.
    - verification log:
        - `just test-pkg ./cmd/kan` -> pass
        - `just test-pkg ./internal/tui` -> pass
        - `just ci` -> pass
    - status:
        - complete; ready for user manual verification pass.
