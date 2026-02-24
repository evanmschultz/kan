# Kan TUI Manual Test Worksheet (Delta: Bootstrap + Roots + Threads)

Use this worksheet to validate only the behavior added/fixed after the previous full worksheet pass.
Run against a clean DB and clean config path.
This refresh explicitly carries forward unfinished delta checks from the previous run.

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

### 0.2 Carry-forward unfinished delta checklist

Actions:

1. Treat every anchor listed below as required for this rerun.
2. Re-run each section and set each listed USER NOTES block to explicit `pass` or `fail`.
3. For every `fail`, add reproduction details and the blocking observation in that section's notes.

Carry-forward anchors (must all be completed):

- `D0.1-N1`
- `D1.1-N1`, `D1.2-N1`, `D1.3-N1`, `D1.4-N1`, `D1.5-N1`, `D1.6-N1`
- `D2.1-N1`
- `D3.1-N1`, `D3.2-N1`
- `D4.1-N1`, `D4.2-N1`
- `D5.1-N1`, `D5.2-N1`
- `D6.1-N1`, `D6.2-N1`, `D6.3-N1`
- `D7.1-N1`

Expected:

- No listed anchor remains blank.
- Delta worksheet is actionable as a complete rerun artifact (not partial notes).

### USER NOTES D0.2-N1

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
3. In startup-required mode, try changing default actor using `h/l`.

Expected:

- Display name input updates normally.
- Mandatory startup mode keeps default actor locked to `user`.
- Actor row remains visible and lock behavior is clear.

### USER NOTES D1.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 1.3 Add global search roots via fuzzy picker

Actions:

1. Focus `global search roots` section.
2. Press `a` or `ctrl+r` to open picker.
3. Use filter input to narrow entries; use arrow keys for navigation and `left/right` for parent/child traversal.
4. Press `ctrl+a` to choose the current directory as a root.

Expected:

- Picker opens in a directory context that is easy to navigate and shows current path.
- Filtering narrows visible entries.
- Typing into filter does not trigger unrelated hotkeys.
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
3. Re-add one root through picker (`a`/`ctrl+r`, then `ctrl+a` in picker).

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
3. Observe initial picker root, use filter input, and navigate with arrow keys.
4. Use `ctrl+a` attach behavior and confirm expected attach target.

Expected:

- Picker starts from global search-root fallback.
- Fuzzy filter behaves consistently while browsing.
- Filter typing keeps text-input priority.
- `ctrl+a` behavior is deterministic for the active picker context.

### USER NOTES D4.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 4.2 Project-root boundary gate for resource attach

Precondition:

- Current project has explicit root mapping set via `paths-roots`.

Actions:

1. Set a project root path where you can browse to a parent/sibling path outside that root.
2. Open task info and trigger resource picker (`r`).
3. Navigate outside the configured root using `..`.
4. Attempt to attach an out-of-root file/dir.
5. Repeat from add/edit task form picker (`ctrl+r`).

Expected:

- Browsing outside root may be visible for navigation context.
- Attach operation is blocked when selected path is outside configured project root.
- Status message reports root-boundary failure (`resource path is outside allowed root`).
- No new out-of-root resource ref is saved to task metadata.

### USER NOTES D4.2-N1

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

### 6.3 Fuzzy backend parity follow-up

Precondition:

- Have tasks with titles/labels that allow abbrev/fuzzy checks (for example `roadmap parser cleanup`, label `backend-fuzzy`).

Actions:

1. In command palette (`:`), type `ns` and verify `new-subtask` ranking remains stable.
2. In search modal (`/`), query `rdmp` and capture matching tasks.
3. In dependency inspector (`ctrl+o`), run `rdmp` and compare candidate set to search modal.
4. Toggle state/archive filters and confirm query behavior updates consistently.

Expected:

- Command palette abbreviation matching remains predictable.
- Backend search behavior used by search/dependency flows matches fuzzy-policy expectations.
- State/archive filters still gate results strictly.

### USER NOTES D6.3-N1

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
