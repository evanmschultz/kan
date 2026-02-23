# Kan TUI Manual Test Worksheet (Delta: Bootstrap + Roots + Threads)

Use this worksheet to validate only the behavior added/fixed after the previous full worksheet pass.
Run against a clean DB and clean config path.

## 0) Setup

### 0.1 Clean state launch

Actions:

1. Remove any prior temp DB/config files.
2. Launch app with isolated config + DB paths.

Commands:

```bash
rm -f /tmp/kan-delta.db /tmp/kan-delta.toml
KAN_DB_PATH=/tmp/kan-delta.db KAN_CONFIG=/tmp/kan-delta.toml just run
```

Expected:

- App opens `Startup Setup Required` modal before project picker.
- No stdin/terminal prompt flow appears outside TUI.

### USER NOTES D0.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 1) Startup Bootstrap Modal

### 1.1 Mandatory modal behavior

Actions:

1. On bootstrap modal, press `esc`.

Expected:

- Modal remains open (cannot dismiss while required fields are incomplete).

### USER NOTES D1.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 1.2 Identity + actor type entry

Actions:

1. Enter display name.
2. Move focus with `tab`.
3. Change default actor using `h/l`.

Expected:

- Display name input updates normally.
- Actor options cycle only between `user|agent|system`.

### USER NOTES D1.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 1.3 Add global search roots via fuzzy picker

Actions:

1. Focus `global search roots` section.
2. Press `a` or `ctrl+r` to open picker.
3. Use filter input to narrow entries.
4. Select a directory and add it.

Expected:

- Picker opens in a directory context that is easy to navigate.
- Filtering narrows visible entries.
- Selected root is added to modal list.

### USER NOTES D1.3-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 1.4 Remove and re-add root

Actions:

1. In roots list, select an entry.
2. Press `d`.
3. Re-add one root through picker.

Expected:

- Selected root is removed.
- Re-adding works and list index remains stable.

### USER NOTES D1.4-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 1.5 Save bootstrap and transition

Actions:

1. Move focus to `[ save settings ]`.
2. Press `enter`.

Expected:

- Modal closes.
- Project picker opens immediately.
- `N` in picker opens new-project form.

### USER NOTES D1.5-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 1.6 Config persistence verification

Actions:

1. Quit app.
2. Inspect saved TOML.

Commands:

```bash
sed -n '1,220p' /tmp/kan-delta.toml
```

Expected:

- `[identity]` contains `display_name` and `default_actor_type`.
- `[paths]` contains non-empty `search_roots`.

### USER NOTES D1.6-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 2) Picker-First Launch Regression

### 2.1 Launch with persisted config

Actions:

1. Relaunch with same `KAN_DB_PATH` and `KAN_CONFIG`.

Expected:

- App opens project picker first (not board directly).
- Picker still supports `N` new project path.

### USER NOTES D2.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 3) Post-First-Run Bootstrap Settings Command

### 3.1 Open bootstrap settings from command palette

Actions:

1. From board, open `:` command palette.
2. run `bootstrap-settings`.

Expected:

- Identity + roots modal opens in non-mandatory mode.
- `esc` cancels normally in this non-startup context.

### USER NOTES D3.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 3.2 Edit and save updated identity/roots

Actions:

1. Reopen `bootstrap-settings`.
2. Change display name and actor.
3. Add/remove roots.
4. Save.

Expected:

- Save succeeds and modal closes.
- Next thread comments use updated ownership defaults.

### USER NOTES D3.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 4) Search-Root Fallback in Resource Picker

### 4.1 Task-form resource picker fallback

Precondition:

- Current project has no explicit project root mapping.

Actions:

1. Open add/edit task modal.
2. Trigger resource picker (`ctrl+r`).
3. Observe initial picker root and use filter.

Expected:

- Picker starts from global search-root fallback.
- Fuzzy filter behaves consistently while browsing.

### USER NOTES D4.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 5) Thread Mode + Ownership Attribution

### 5.1 Project thread render and comment post

Actions:

1. Command palette: `thread-project`.
2. Confirm description renders markdown.
3. Enter markdown comment and submit (`enter`).

Expected:

- Thread full-screen view shows markdown-rendered description.
- New comment appears with `[actor] author` metadata and timestamp.

### USER NOTES D5.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 5.2 Work-item thread from task info

Actions:

1. Open task info (`i`/`enter`).
2. Press `c` to open thread.
3. Press `esc`.

Expected:

- Thread opens for selected work item target.
- `esc` returns to task info.

### USER NOTES D5.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 6) Focused Regression Spot Checks

### 6.1 Command palette windowing stability

Actions:

1. Open command palette.
2. Hold `j` through many entries.

Expected:

- Selected command row remains visible in the current window.
- No rendering jump that hides selected row.

### USER NOTES D6.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 6.2 Thread comment append stability

Actions:

1. Open an existing thread with at least one prior comment.
2. Post a new comment.

Expected:

- Existing comments remain visible.
- New comment appends (no overwrite of prior in-memory thread list).

### USER NOTES D6.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 7) Final Artifact/Log Sanity

Actions:

1. Quit app.
2. Check runtime artifact locations.

Commands:

```bash
git status --short
ls -la .kan 2>/dev/null || true
ls -la cmd/kan/.kan 2>/dev/null || true
```

Expected:

- Runtime logs remain under repo-root `.kan/log/` (dev mode).
- No unexpected `cmd/kan/.kan` runtime directory.

### USER NOTES D7.1-N1

- Pass/Fail:
- Evidence:
- Notes:
