# kan

A local-first Kanban TUI built with Bubble Tea v2, Bubbles v2, and Lip Gloss v2.

`kan` is designed as a better human-visible planning and verification surface than ad-hoc markdown checklists. The primary direction is human + coding-agent collaboration with explicit state, auditability, and clear completion gates, while still remaining useful as a standalone personal TUI task manager.

Current scope:
- local tracking and planning workflows (human-operated TUI).
- local runtime diagnostics with styled logging and dev-mode local log files.
- active Phase 11 wave delivery for locked MCP/HTTP slices is in progress in `MCP_DESIGN_AND_PLAN.md` (HTTP `/api/v1`, stateless MCP adapter, `capture_state`, attention/worksheet readiness).
- advanced import/export transport-closure concerns (branch/commit-aware divergence reconciliation and richer conflict tooling) remain roadmap-only unless user re-prioritizes.

Contributor workflow and CI policy: `CONTRIBUTING.md`

## Features
- Multi-project Kanban board.
- Launches into a project picker first (no auto-created default project).
- SQLite persistence (`modernc.org/sqlite`, no CGO).
- Keyboard navigation (`vim` keys + arrows) and mouse support.
- Archive-first delete flow with configurable defaults.
- Project and work-item thread mode with ownership-attributed markdown comments.
- Descriptions/comments are stored as markdown source fields and rendered in TUI views.
- Project roots are real filesystem directory mappings; resource attachment is blocked outside the allowed root.
- Runtime kind-catalog + project allowlist validation for project/task mutations.
- Runtime JSON-schema validation for kind metadata payloads (with compiled-validator caching).
- Capability-lease primitives for strict mutation locking (issue/heartbeat/renew/revoke/revoke-all).
- Serve mode for HTTP (`/api/v1`) + stateless MCP (`/mcp`) transport surfaces.
- JSON snapshot import/export.
- Configurable task field visibility.

## Active Status (2026-02-24)
Implemented now:
- Use `kan` as the canonical local planning/verification source while collaborating with an agent in terminal/chat.
- Keep manual QA notes in `TUI_MANUAL_TEST_WORKSHEET.md` with sectioned anchors for precise replay.
- Local-only TUI + SQLite workflows (including startup bootstrap, project picker, threads/comments, and import/export snapshots).
- Board info line includes hierarchy-aware focus guidance (`f` focus subtree, `F` return full board) with selected level and child counts for branch/phase/subphase navigation.
- Board scope rendering is level-scoped: project shows immediate project children, and focused branch/phase/subphase views show immediate children for that level (not full descendant dumps).
- Task-focused scope renders direct subtasks in the board so `f` on a task opens subtask-level board context.
- Board path context is always visible above columns (`path: project -> ...`) and updates on each `f` drill-down.
- Board cards now include hierarchy markers in metadata (`[branch|...]` / `[phase|...]`) so branch/phase rows are visually distinct from task rows.
- Wide layouts render a right-side notices panel with unresolved attention summary, selected-item context, and recent activity hints.
- `n` now respects active focus scope: in focused branch/phase/subphase it creates a child in that scope, and in focused task scope it creates a subtask.
- Kind-catalog bootstrap + project `allowed_kinds` enforcement is active for project/task write paths.
- Project-level `kind` and task-level `scope` persistence are active (`project|branch|phase|subphase|task|subtask` semantics enforced by kind rules).
- Kind template system actions can auto-append checklist items and auto-create child work items during task creation.
- Capability-lease/mutation-guard enforcement scaffolding is active in app/service write paths for non-user actors.

Wave-locked MCP/HTTP direction (implemented and in active dogfooding closeout):
- Transport/tool direction is REST/tool-style with markdown description/comment fields documented as markdown-write text.
- `capture_state` is a summary-first recovery surface for level-scoped workflows.
- Attention/blocker signaling direction is node-scoped with user-action visibility and paginated scope queries for user/agent coordination.
- Transport-level lease/scope request contracts enforce non-user mutation guardrails.
- MCP tool surface now includes:
  - bootstrap guidance: `kan.get_bootstrap_guide`
  - projects: `kan.list_projects`, `kan.create_project`, `kan.update_project`
  - tasks/work graph: `kan.list_tasks`, `kan.create_task`, `kan.update_task`, `kan.move_task`, `kan.delete_task`, `kan.restore_task`, `kan.reparent_task`, `kan.list_child_tasks`, `kan.search_task_matches`
  - capture/attention: `kan.capture_state`, `kan.list_attention_items`, `kan.raise_attention_item`, `kan.resolve_attention_item`
  - change/dependency context: `kan.list_project_change_events`, `kan.get_project_dependency_rollup`
  - kinds/allowlists: `kan.list_kind_definitions`, `kan.upsert_kind_definition`, `kan.set_project_allowed_kinds`, `kan.list_project_allowed_kinds`
  - capability leases: `kan.issue_capability_lease`, `kan.heartbeat_capability_lease`, `kan.renew_capability_lease`, `kan.revoke_capability_lease`, `kan.revoke_all_capability_leases`
  - comments: `kan.create_comment`, `kan.list_comments_by_target`
  - empty-instance `capture_state` now returns deterministic `bootstrap_required` signaling, and agents can call `kan.get_bootstrap_guide` for next steps.
  - parity/guardrail notes:
    - `capture_state.state_hash` is stable across MCP/HTTP calls for unchanged underlying state (timestamp jitter excluded from hash input);
    - `kan.revoke_all_capability_leases` fails closed on invalid/unknown scope tuples;
    - `kan.create_comment` fails closed when the target does not exist in the referenced project;
    - `kan.update_task` title-only updates preserve existing priority when `priority` is omitted.

Roadmap-only in the active wave (explicitly deferred):
- advanced import/export transport closure concerns (branch/commit-aware divergence reconciliation and conflict tooling),
- remote/team auth-tenancy expansion and additional security hardening,
- dynamic tool-surface policy and broader template-library expansion.

Dangerous limitation note (pre-hardening, design warning):
- In future policy-controlled override flows, orchestrator calls may receive override-token material.
- That design currently assumes orchestrator adherence to user policy/guidance; treat overrides as explicit user-approved actions only.

## Run
```bash
just run
```

Or build once and run the binary:
```bash
just build
./kan
```

## Startup Behavior
- TUI launch opens the project picker before normal board mode.
- If no projects exist yet, the picker stays open and supports `N` to create the first project.
- On TUI startup, missing required bootstrap fields are prompted and persisted:
  - `identity.display_name`
  - one default path (stored as the single active entry in `paths.search_roots`)

## CLI Commands
Export current data:
```bash
./kan export --out /tmp/kan.json
```

Snapshot export includes:
- projects, columns, tasks/work-items
- kind catalog definitions + project allowed-kind closure
- comments/threads
- capability leases

Import snapshot:
```bash
./kan import --in /tmp/kan.json
```

Include only active records in export:
```bash
./kan export --out /tmp/kan-active.json --include-archived=false
```

## Config
`kan` loads TOML config from platform defaults, or from `--config` / `KAN_CONFIG`.

Database path precedence:
1. `--db`
2. `KAN_DB_PATH`
3. TOML `database.path`
4. platform default path

Path resolution controls:
- `--app` / `KAN_APP_NAME` to namespace paths (default `kan`)
- `--dev` / `KAN_DEV_MODE` to use `<app>-dev` path roots
- `kan paths` prints the resolved config/data/db paths for the current environment
- `identity.default_actor_type` (`user|agent|system`) + `identity.display_name` are defaults for new thread comment ownership
- `paths.search_roots` stores one active default path used by bootstrap and path-pickers
- task resource attachments require a configured per-project root mapping (`project_roots`)
- dev mode logging writes to workspace-local `.kan/log/` when `logging.dev_file.enabled = true`
  - relative dev log dirs are anchored to the nearest workspace root marker (`go.mod` or `.git`)
- logging level is controlled by TOML `logging.level` (`debug|info|warn|error|fatal`)

Example:
```toml
[database]
path = ""

[delete]
default_mode = "archive" # archive | hard

[task_fields]
show_priority = true
show_due_date = true
show_labels = true
show_description = false

[board]
show_wip_warnings = true
group_by = "none" # none | priority | state

[search]
cross_project = false
include_archived = false
states = ["todo", "progress", "done"] # plus optional "archived"

[identity]
display_name = "" # required at TUI startup bootstrap
default_actor_type = "user" # user | agent | system

[paths]
search_roots = [] # bootstrap writes one active default path entry

[logging]
level = "info"

[logging.dev_file]
enabled = true
dir = ".kan/log"
```

Full template: `config.example.toml`

## Key Controls
- `h/l` or `←/→`: move column
- `j/k` or `↓/↑`: move task
- `n`: new task
- `e`: edit task
- `i` or `enter`: task info modal
- `c` (in task info): open thread for the selected work item
- `d` (in task form due field): open due-date picker
- `f`: focus selected subtree (including empty scopes)
- `F`: return to full board
- `p`: project picker
- `N` (in project picker): new project
- `:`: command palette
- `/`: search
- `d`: delete using configured default mode
- `a`: archive task
- `D`: hard delete task
- `u`: restore task
- `t`: toggle archived visibility
- `v`: toggle text-selection mode (copy-friendly mouse selection)
- `?`: toggle expanded help
- `q`: quit

Command palette highlights:
- `new-branch`, `edit-branch`, `archive-branch`, `restore-branch`, `delete-branch`
- `new-phase`, `new-subphase`
- `new-project`, `edit-project`, `archive-project`, `restore-project`, `delete-project`
- while subtree focus is active, `new-branch` is blocked and shows a warning modal; clear focus (`F`) first

## Thread Mode
- Open project thread from command palette with `thread-project` (`project-thread` alias).
- Open selected work-item thread with `thread-item` (`item-thread` / `task-thread` aliases), or `c` from task info.
- Supported thread targets: project, task, subtask, phase, decision, and note.
- New comments use configured identity defaults; invalid/empty identity safely falls back to `[user] kan-user`.

## Fang Context
Fang is Charmbracelet's experimental batteries-included wrapper for Cobra CLIs.
`kan` does not currently integrate Fang or Cobra for CLI command execution.
Current usage is Fang-inspired help copy/style in the in-app command reference overlay.

## Developer Workflow
Primary commands:
```bash
just fmt
just test-pkg ./internal/app
just check
just test
just ci
```

For contribution policy, pre-push expectations, and branch-protection recommendations, see `CONTRIBUTING.md`.

VHS visual regression captures:
```bash
just vhs
just vhs vhs/regression_subtasks.tape
just vhs vhs/regression_scroll.tape
```

Golden tests:
```bash
just test-golden
just test-golden-update
```

## CI
GitHub Actions runs split gates:
- matrix smoke checks on macOS/Linux/Windows via `just check`
- full Linux gate via `just ci`
- Goreleaser snapshot validation after the full Linux gate
