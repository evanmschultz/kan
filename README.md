# kan

A local-first Kanban TUI built with Bubble Tea v2, Bubbles v2, and Lip Gloss v2.

`kan` is designed as a better human-visible planning and verification surface than ad-hoc markdown checklists. The primary direction is human + coding-agent collaboration with explicit state, auditability, and clear completion gates, while still remaining useful as a standalone personal TUI task manager.

Current scope:
- local tracking and planning workflows (human-operated TUI).
- local runtime diagnostics with styled logging and dev-mode local log files.
- MCP/HTTP integrations are roadmap items and are not implemented yet.

## Features
- Multi-project Kanban board.
- Launches into a project picker first (no auto-created default project).
- SQLite persistence (`modernc.org/sqlite`, no CGO).
- Keyboard navigation (`vim` keys + arrows) and mouse support.
- Archive-first delete flow with configurable defaults.
- Project and work-item thread mode with markdown comments and ownership metadata.
- JSON snapshot import/export.
- Configurable task field visibility.

## Human-Agent Workflow (Current + Roadmap Direction)
- **Today (pre-Phase 11):** use `kan` as the canonical local planning/verification source while collaborating with an agent in terminal/chat.
- **Current best practice:** keep manual QA notes in `TUI_MANUAL_TEST_WORKSHEET.md` with sectioned anchors so findings are precise and replayable.
- **Roadmap (Phase 11+):** expose the same project/branch/phase/task state through MCP/HTTP so agents can consume authoritative updates instead of fragile markdown-only status files.

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
  - at least one `paths.search_roots` entry

## CLI Commands
Export current data:
```bash
./kan export --out /tmp/kan.json
```

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
- `paths.search_roots` are global fallback roots for the resource picker when no per-project root is configured
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
search_roots = [] # required at TUI startup bootstrap

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
- `ctrl+d` or `D` (in task form due field): open due-date picker
- `p`: project picker
- `N` (in project picker): new project
- `:`: command palette
- `/`: search
- `d`: delete using configured default mode
- `a`: archive task
- `D`: hard delete task
- `u`: restore task
- `t`: toggle archived visibility
- `?`: toggle expanded help
- `q`: quit

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
just test
just ci
```

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
GitHub Actions runs matrix CI on macOS, Linux, and Windows via `just ci`, plus a Goreleaser snapshot validation job.
