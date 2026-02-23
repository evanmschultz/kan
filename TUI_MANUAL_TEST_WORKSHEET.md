# Kan TUI Manual Test Worksheet (Pre-Phase 11 Closeout Retest)

Use this worksheet for a fresh end-to-end validation pass of all pre-Phase-11 behavior.
Run against a clean DB. Capture screenshots/GIFs for any failures.

## 0) Setup

### 0.1 Environment bootstrap

Actions:

1. Start with a clean DB path.
2. Run the app in a terminal at least 140x45.
3. If no projects exist, confirm first-run immediately opens project creation flow.

Commands:

```bash
rm -f /tmp/kan-manual-test.db
KAN_DB_PATH=/tmp/kan-manual-test.db just run
```

Expected:

- App launches without migration/runtime error.
- Board + status + bottom help render.
- Empty DB does not auto-create a default project; `New Project` flow is shown.

### USER NOTES S0.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 0.2 Artifact/log sanity

Actions:

1. Quit app.
2. Verify local runtime artifact locations.

Commands:

```bash
git status --short
ls -la .kan 2>/dev/null || true
ls -la cmd/kan/.kan 2>/dev/null || true
```

Expected:

- Runtime logs are under repo-root `.kan/log/`.
- No `cmd/kan/.kan` runtime artifact directory is created.
- Generated artifacts remain gitignored.

### USER NOTES S0.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 1) Board UX and Navigation

### 1.1 Column and cursor navigation

Actions:

1. Move columns with `h/l` and left/right arrows.
2. Move rows with `j/k` and up/down arrows.

Expected:

- Focus changes correctly with vim keys and arrows.
- No erratic cursor jumps.

### USER NOTES S1.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 1.2 Long-list viewport follow

Precondition:

- Have 15+ tasks in one column.

Actions:

1. Hold `j` to move selection beyond visible rows.
2. Add new tasks until list exceeds viewport.
3. Move back with `k`.

Expected:

- Column viewport stays bounded; board height does not grow.
- Focused row stays visible while scrolling.
- Newly created task becomes focused after create and remains visible.

### USER NOTES S1.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 1.3 Row marker semantics and styling

Actions:

1. Select a task with `space`.
2. Keep cursor on that same row.
3. Move focus to another selected row.

Expected:

- Focused row uses fuchsia highlight.
- Focused+selected row keeps selection cue (does not lose selected style).
- Selection styling is distinct from plain focus.

### USER NOTES S1.3-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 2) Task Create/Edit Modal UX

### 2.1 Add task modal centered overlay

Actions:

1. Press `n`.
2. Verify centered overlay behavior.
3. In labels field, use `ctrl+l` and `ctrl+y` suggestion acceptance.
4. Use `ctrl+d` due picker.
5. Use `ctrl+r` to stage resource refs.
6. Fill dependency fields (`depends_on`, `blocked_by`, `blocked_reason`).

Expected:

- Modal is centered and does not push board layout.
- Tab order is deterministic.
- Label suggestion and picker are usable.
- Resource attach from create flow works.
- Dependency values are accepted in add flow.

### USER NOTES S2.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 2.2 Edit task modal behavior

Actions:

1. Select task and press `e`.
2. Validate prefilled values (title/description/priority/due/labels/dependencies/resources).
3. Update fields and save.

Expected:

- Existing values load correctly.
- No stale/duplicated field values.
- Save persists updates including dependency and resource metadata edits.

### USER NOTES S2.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 2.3 Priority and due behavior

Actions:

1. Cycle priority with `h/l` in priority field.
2. Enter due date only (`YYYY-MM-DD`).
3. Enter due datetime (`YYYY-MM-DD HH:MM` and `YYYY-MM-DDTHH:MM`).
4. Enter past due datetime.

Expected:

- Priority picker remains keyboard-friendly.
- Date and datetime inputs are accepted.
- Past due warning appears before save.
- Due warnings surface in board/task-info context after save.

### USER NOTES S2.3-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 3) Task Info + Subtask Drill Flow

### 3.1 Open info from list

Actions:

1. Select task.
2. Press `i`.
3. Close and repeat with `enter`.

Expected:

- Both `i` and `enter` open task info.
- Modal remains centered.
- Task-info hints include edit, dependency edit, resource attach, subtask, and move shortcuts.

### USER NOTES S3.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 3.2 Subtask visibility and completion model

Precondition:

- Parent task with subtasks.

Actions:

1. Observe parent row on board.
2. Open parent task info.
3. Use `enter` on a subtask to drill in.
4. Use `[` / `]` in task-info to move subtask state.

Expected:

- Board row hides inline subtasks and shows compact progress (`done/total`).
- Task info shows subtasks list.
- Subtasks can be progressed/completed from task-info context.
- Parent move to `done` is blocked while any subtask remains incomplete.

### USER NOTES S3.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 3.3 Subtask drill-in and step-back consistency

Actions:

1. In task info, use `j/k` to highlight subtask.
2. Press `enter` to open subtask detail.
3. Press `backspace` to parent.
4. Press `esc` repeatedly to step back then close.

Expected:

- Drill-in works.
- `backspace` and `esc` behave as one-step-back navigation.
- Subtasks remain visible/accessible regardless of parent column/state.

### USER NOTES S3.3-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 4) Resource Attachment UX

### 4.1 Attach resource from task info

Actions:

1. Open task info.
2. Press `r`.
3. Use picker filter typing and `ctrl+u` clear.
4. Attach file and directory entries.

Expected:

- Picker opens centered.
- Filter narrows entries.
- Attach behavior is explicit and predictable.
- Attached refs appear in task info.

### USER NOTES S4.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 4.2 Attach resource from create/edit task forms

Actions:

1. Open add-task (`n`) and edit-task (`e`) modals.
2. Use `ctrl+r` in each modal.
3. Stage resources and save.

Expected:

- Attach flow works from add and edit modals.
- Staged refs persist after save.

### USER NOTES S4.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 5) Search + Command Palette

### 5.1 Search modal ergonomics

Actions:

1. Press `/`.
2. Tab across query/state/scope/archive controls.
3. Apply filters and inspect results.

Expected:

- Focus order is deterministic.
- Search results update correctly.
- Clear query vs reset filters remain distinct.

### USER NOTES S5.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 5.2 Command palette filtering and execution

Actions:

1. Press `:`.
2. Try fuzzy/abbrev queries (for example `ns` for `new-subtask`).
3. Execute `search-all` and `search-project` from palette.
4. Scroll through palette list beyond first page.

Expected:

- Fuzzy command ranking behaves predictably.
- Enter executes highlighted command.
- `search-all` and `search-project` open search mode with correct scope.
- Windowed scrolling keeps highlighted command visible.

### USER NOTES S5.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 5.3 Quick actions menu state-awareness

Actions:

1. Press `.` with no multi-selection.
2. Observe enabled vs disabled actions.
3. Select tasks with `space` and reopen quick actions.
4. Execute bulk actions.

Expected:

- State-irrelevant actions appear disabled with reason.
- Enabled actions sort first.
- Disabled actions cannot execute.
- Bulk actions become available when selection exists.

### USER NOTES S5.3-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 6) Multi-Select + Bulk Operations

### 6.1 Selection controls

Actions:

1. Toggle select on multiple tasks with `space`.
2. Clear selection with `esc`.

Expected:

- Selected set updates predictably.
- Selection indicators stay clear and stable.

### USER NOTES S6.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 6.2 Bulk move/archive/delete

Actions:

1. Select 2+ tasks.
2. Run bulk move left/right (`[`/`]` with selection).
3. Run bulk archive/delete and confirm.

Expected:

- Bulk actions apply to selected set.
- Confirm modal appears for destructive operations.
- Non-applicable actions are blocked with clear status.

### USER NOTES S6.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 7) Undo/Redo + Activity Log

### 7.1 Undo/redo

Actions:

1. Perform several mutating actions.
2. Press `z` for undo.
3. Press `Z` for redo.

Expected:

- Undo/redo sequence is deterministic.
- User-facing status is clear for non-undoable cases.

### USER NOTES S7.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 7.2 Activity log modal

Actions:

1. Press `g`.
2. Review persisted recent entries.

Expected:

- Modal opens centered.
- Entries are readable and ordered.

### USER NOTES S7.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 8) Project Management + Paths + Labels Config

### 8.1 Create/edit project

Actions:

1. Press `N` create project.
2. Set metadata fields including `color` and `root_path`.
3. Save and confirm selected project/task list refresh behavior.
4. Press `M` to edit and resave.

Expected:

- Create/edit both work.
- New project selection refresh is immediate (no stale prior-project tasks).
- Accent color changes are visible in project-scoped styling.
- Root path is editable in form.

### USER NOTES S8.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 8.2 Project picker behavior

Actions:

1. Press `p`/`P`.
2. Switch across projects.

Expected:

- Picker opens correctly.
- Selection applies and board reloads for selected project.

### USER NOTES S8.2-N1

- Pass/Fail:
- Evidence:
- Notes:

---

### 8.3 Project root mapping + labels config workflows

Actions:

1. Open paths/roots flow from command palette (`paths-roots`).
2. Use `ctrl+r` directory picker and filter; save valid root.
3. Validate invalid path error handling.
4. Open labels config (`labels-config`) and update global/project labels.

Expected:

- Root mapping accepts valid directories and blocks invalid paths with clear error.
- Directory picker supports fuzzy filtering for easier selection.
- Labels config saves and applies global/project defaults.

### USER NOTES S8.3-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 9) Help + Discoverability

### 9.1 Help overlay content

Actions:

1. Press `?`.
2. Validate listed keys and workflows against runtime behavior.

Expected:

- Help renders as centered overlay.
- Key hints match actual behavior (including task-info/subtask/dependency/resource flows).

### USER NOTES S9.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## 10) Final Regression Sweep

### 10.1 End-to-end core flow

Actions:

1. Create project and set root path.
2. Create parent task and subtasks.
3. Edit dependencies and labels defaults.
4. Move tasks across columns and verify completion rules.
5. Attach resources from create/edit/info flows.
6. Use search, command palette, quick actions, and bulk operations.
7. Use undo/redo and review activity log.
8. Restart app and verify persistence.

Expected:

- No crashes.
- No data loss.
- Behavior matches sections above.

### USER NOTES S10.1-N1

- Pass/Fail:
- Evidence:
- Notes:

---

## Final Sign-off

- Overall result: Pass / Pass with minor issues / Fail
- Critical bugs:
- Non-critical UX issues:
- Suggested next priorities:
- Tester:
- Date:
