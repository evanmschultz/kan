# Kan TUI Manual Test Worksheet (Pre-Phase 11 Closeout Retest)

Use this worksheet for a fresh end-to-end validation pass of all pre-Phase-11 behavior.
Run against a clean DB. Capture screenshots/GIFs for any failures.
For this refresh pass, keep prior USER NOTES as historical context and append a dated retest outcome in each section.

## 0) Setup

### 0.1 Environment bootstrap

Actions:

1. Start with clean DB and config paths.
2. Run the app in a terminal at least 140x45.
3. On first launch, complete `Startup Setup Required` (display name + at least one global search root).
4. Save bootstrap settings and confirm project picker opens with `New Project` access.

Commands:

```bash
rm -f /tmp/kan-manual-test.db /tmp/kan-manual-test.toml
KAN_DB_PATH=/tmp/kan-manual-test.db KAN_CONFIG=/tmp/kan-manual-test.toml just run
```

Expected:

- App launches without migration/runtime error.
- First-run opens `Startup Setup Required` before project picker.
- No terminal/stdin prompt appears outside the TUI.
- After save, project picker shows and `New Project` flow is available.

### USER NOTES S0.1-N1

- Pass/Fail: pass
- Evidence:
- Notes: on start up the project picker should show up so the user picks the project and an option to make a new on form that picker. the project picker should allow for making a new one always.///:

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

- Pass/Fail: pass
- Evidence:
- Notes:

---

### 0.3 Delta carry-forward prerequisite

Actions:

1. Open `TUI_MANUAL_TEST_WORKSHEET_DELTA_BOOTSTRAP_THREADS.md`.
2. Complete the explicit carry-forward checklist in `D0.2-N1` before final sign-off here.

Expected:

- No unresolved delta anchors remain blank.
- Full worksheet sign-off is blocked until delta carry-forward checks are completed.

### USER NOTES S0.3-N1

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

- Pass/Fail: pass
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

- Pass/Fail: pass
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
- Selection marker/cursor symbol appears on title line only (not duplicated on secondary/meta line).

### USER NOTES S1.3-N1

- Pass/Fail: pass
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
6. In `depends_on` or `blocked_by`, press `ctrl+o` to open dependency inspector.
7. In dependency inspector, use `tab` to focus list, `j/k` to navigate, review details panel, `d`/`b` to add or remove refs, and `a` to apply.
8. Also verify raw CSV entry still works for `depends_on` and `blocked_by` when typed manually.

Expected:

- Modal is centered and does not push board layout.
- Tab order is deterministic.
- Label suggestion and picker are usable.
- Resource attach from create flow works.
- Dependency inspector supports list navigation, detail inspection, add/remove toggles, and apply.
- Dependency values are accepted in add flow through both picker and raw CSV input.

### USER NOTES S2.1-N1

- Pass/Fail:
- Evidence:
- Notes: date and time should be two steps. so you pick the date and have the option to save or add a time and then there you have a picker or can type it in which as you type it would narrow the amount of options in the picker.

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

- Pass/Fail: pass
- Evidence:
- Notes: delete from file

---

### 2.3 Priority and due behavior

Actions:

1. Cycle priority with `h/l` in priority field.
2. Enter due date only (`YYYY-MM-DD`).
3. Enter due datetime (`YYYY-MM-DD HH:MM` and `YYYY-MM-DDTHH:MM`).
4. Enter past due datetime.
5. Focus the due field and verify inline hint text shows all accepted formats (`YYYY-MM-DD`, `YYYY-MM-DD HH:MM`, `YYYY-MM-DDTHH:MM`, `RFC3339`, `-`) and UTC default note.

Expected:

- Priority picker remains keyboard-friendly.
- Date and datetime inputs are accepted.
- Past due warning appears before save.
- Due field hint explicitly documents typed time support and UTC default behavior.
- Due warnings surface in board/task-info context after save.

### USER NOTES S2.3-N1

- Pass/Fail:
- Evidence:
- Notes: see above about edit and create task date picker and time information

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
- Task-info hints include edit, dependency inspector, resource attach, subtask, move (`[ / ]`), and checklist toggle (`space`) shortcuts.
- Pressing `b` opens dependency inspector with linked deps/blockers pinned at the top, list navigation, detail panel, add/remove, and `enter` jump-to-task behavior.

### USER NOTES S3.1-N1

- Pass/Fail:
- Evidence:
- Notes: esc should take you to the next higher when in subtask create or edit modal esc takes you all the way back instead of back to the task that is its parent.

---

### 3.2 Subtask visibility and completion model

Precondition:

- Parent task with subtasks.

Actions:

1. Observe parent row on board.
2. Open parent task info.
3. Confirm each subtask renders as checklist row (`[ ]` incomplete, `[x]` complete).
4. Press `space` on focused subtask to mark complete, then press `space` again to reopen.
5. Use `enter` on a subtask to drill in.
6. Use `[` / `]` in task-info to move subtask state directly.

Expected:

- Board row hides inline subtasks and shows compact progress (`done/total`).
- Task info shows checklist-style subtasks list with clear state/complete metadata.
- `space` toggles focused subtask completion between done and active columns.
- Subtasks can be progressed/completed from task-info context.
- Parent move to `done` is blocked while any subtask remains incomplete.

### USER NOTES S3.2-N1

- Pass/Fail:
- Evidence:
- Notes: need to show that 'space' is how you mark a subtask as complete. also, `?` should work on any menu aside from a text input field and show up only with the hotkeys for that menu!

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
- Task-info header metadata clearly shows state + complete status for the focused task/subtask.

### USER NOTES S3.3-N1

- Pass/Fail:
- Evidence:
- Notes: adding a new subtask should take you back to its parent not take you back to the main menu when pressing enter to save it!

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
- While filter input is focused, typing updates filter text (no hotkey hijack).
- Attach behavior is explicit and predictable.
- Attached refs appear in task info.

### USER NOTES S4.1-N1

- Pass/Fail: pass
- Evidence:
- Notes: when using the filter the hotkeys will do their thing instead of typing into the text field. we need to fix this on ALL text-input fields!

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

- Pass/Fail: pass
- Evidence:
- Notes:

---

### 4.3 Root-boundary attachment guard

Precondition:

- Current project has explicit root mapping set to a directory with at least one sibling directory outside that root.

Actions:

1. Open command palette and run `paths-roots`; set/confirm the project root path.
2. Open task info (`i`) for any task and press `r`.
3. In picker, navigate to `..` so you reach a parent/sibling path outside the configured project root.
4. Attempt to attach that out-of-root file/dir.
5. Repeat from add/edit task forms via `ctrl+r`.

Expected:

- Picker navigation can still browse parent paths for visibility.
- Attach attempt outside the configured project root is blocked.
- Status message reports root-boundary rejection (`resource path is outside allowed root`).
- No out-of-root reference is persisted in task metadata.

### USER NOTES S4.3-N1

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
4. While query input is focused, type `j/k` and confirm they are inserted as text.

Expected:

- Focus order is deterministic.
- Search results update correctly.
- Query input keeps text-input priority while focused.
- Clear query vs reset filters remain distinct.

### USER NOTES S5.1-N1

- Pass/Fail: pass
- Evidence:
- Notes:

---

### 5.2 Command palette filtering and execution

Actions:

1. Press `:`.
2. Try fuzzy/abbrev queries (for example `ns` for `new-subtask`).
3. Execute `search-all` and `search-project` from palette.
4. Execute `highlight-color` and set a custom value (for example `201`).
5. Scroll through palette list beyond first page.

Expected:

- Fuzzy command ranking behaves predictably.
- Enter executes highlighted command.
- `search-all` and `search-project` open search mode with correct scope.
- `highlight-color` updates focused-row highlight color at runtime.
- Windowed scrolling keeps highlighted command visible.

### USER NOTES S5.2-N1

- Pass/Fail: pass
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

- Pass/Fail: pass
- Evidence:
- Notes:

---

### 5.4 Fuzzy backend parity checks (consensus lock)

Precondition:

- Have tasks with predictable titles/labels: `new-subtask parser`, `roadmap parser cleanup`, and one task labeled `backend-fuzzy`.

Actions:

1. Open command palette (`:`), type `ns`, and verify `new-subtask` ranks near the top.
2. Open search (`/`) and run query `rdmp` (abbrev for roadmap).
3. Change state filters and archived toggle; rerun the same query.
4. Open dependency inspector (`ctrl+o`) from add/edit flow and run query `rdmp`.
5. Compare search results between search modal and dependency inspector for the same query and filter settings.

Expected:

- Command palette abbreviation matching remains deterministic.
- Search + dependency inspector use aligned fuzzy matching behavior (no backend-only substring divergence).
- State/archive filters remain strict regardless of fuzzy query text.
- `no matches` status appears clearly when expected.

### USER NOTES S5.4-N1

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

- Pass/Fail: pass
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

- Pass/Fail: pass
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

- Pass/Fail: pass
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

- Pass/Fail: pass
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
- Notes: color picker works, but it is border not 'accent', but we should make it possible to control the three colors, the border, the accent (highlights), and the main text color which would be. the white text for a normal unfocused task

---

### 8.2 Project picker behavior

Actions:

1. Press `p`/`P`.
2. Switch across projects.

Expected:

- Picker opens correctly.
- Selection applies and board reloads for selected project.

### USER NOTES S8.2-N1

- Pass/Fail: pass
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
- Notes: this needs to be fixed and redone, label config was supposed to be project specific, not across all projects.

---

### 8.4 Full-screen thread markdown + comments

Actions:

1. Open command palette (`:`) and run `thread-project`.
2. Verify full-screen thread shows project description rendered as markdown.
3. Run `thread-item` from command palette for the selected work item, then open task info (`i`) and press `c` to open the same item thread via shortcut.
4. In thread mode, enter markdown in the composer input and press `enter` to post.
5. Confirm posted comment metadata shows actor type, author name, and timestamp.
6. Resize terminal width and verify description/comment wrapping updates cleanly.

Expected:

- Thread mode is full-screen and non-destructive (`esc` returns to prior context).
- Description and comment bodies render via markdown styles (headings/lists/emphasis readable).
- New comment persists and appears with ownership attribution from identity defaults.
- Thread view remains readable after terminal resize.

### USER NOTES S8.4-N1

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
- Key hints match actual behavior (including `space` subtask checklist toggle and `[ / ]` move wording).

### USER NOTES S9.1-N1

- Pass/Fail: pass
- Evidence:
- Notes:

---

## 10) Final Regression Sweep

### 10.1 End-to-end core flow

Actions:

1. Create project and set root path.
2. Create parent task and subtasks, then toggle completion from task-info checklist (`space`).
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
- Dependency/blocker modal supports inspect + navigate + jump + add workflows without leaving inconsistent state.

### USER NOTES S10.1-N1

- Pass/Fail: pass
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

### USER NOTES ADDITIONAL

1. We don't have phase and sub-phase logic and capabilities. a phase should appear in the list with tasks but say they are a phase and pressing enter would change the view to the same kind of project view with tasks that are in to do in progress and done. there should be a phase info section and a way to edit add phase data with sensible good fields like tasks but better for phases. this would be separate from tasks of course. and phases can be nested. in the top near where the project's are listed like `[project 1] project 2` there should be a 'path' of where the phase is, meaning `Project | Phase | Sub-phase` and so on. phases can't be marked as complete until all their tasks are marked done, so no open todo or in progress tasks can be there to mark a phase as done. note that the phase at the project level would move like a task through the 4 'states'
2. project root_path logic. we need a way to make the path much easier and user friendly. we need a "picker" type system with a kind of fzf fuzzy path finder or something. the path can be a file. but it should be a user friendly search thing where config.toml could allow user's to set where path searching should start, for instance `.../documents/code` so the system would be faster.
