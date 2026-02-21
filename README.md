# kan

A local-first Kanban TUI built with Bubble Tea v2, Bubbles v2, and Lip Gloss v2.

## Features
- Multi-project Kanban board.
- SQLite persistence (`modernc.org/sqlite`, no CGO).
- Keyboard navigation (`vim` keys + arrows) and mouse support.
- Archive-first delete flow with configurable defaults.
- JSON snapshot import/export.
- Configurable task field visibility.

## Run
```bash
just run
```

Or build once and run the binary:
```bash
just build
./kan
```

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

[[board.states]]
id = "todo"
name = "To Do"
wip_limit = 0
position = 0

[search]
cross_project = false
include_archived = false
states = ["todo", "progress", "done"] # plus optional "archived"
```

Full template: `config.example.toml`

## Key Controls
- `h/l` or `←/→`: move column
- `j/k` or `↓/↑`: move task
- `n`: new task
- `e`: edit task
- `i` or `enter`: task info modal
- `ctrl+d` or `D` (in task form due field): open due-date picker
- `p`: project picker
- `/`: search
- `d`: delete using configured default mode
- `a`: archive task
- `D`: hard delete task
- `u`: restore task
- `t`: toggle archived visibility
- `?`: toggle expanded help
- `q`: quit

## Developer Workflow
Primary commands:
```bash
just fmt
just test-pkg ./internal/app
just test
just ci
```

Golden tests:
```bash
just test-golden
just test-golden-update
```

## CI
GitHub Actions runs matrix CI on macOS, Linux, and Windows via `just ci`, plus a Goreleaser snapshot validation job.
