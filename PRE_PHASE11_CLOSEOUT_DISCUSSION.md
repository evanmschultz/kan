# Pre-Phase 11 Closeout Discussion

Purpose: capture closeout decisions from your manual worksheet notes and preserve the locked pre-Phase-11 consensus baseline before Phase 11 begins.

Date: 2026-02-22

---

## 0) Consensus So Far (Captured From Latest Discussion)

These are now treated as locked unless we explicitly change them together.

Note:
- Earlier per-section "Open questions" in this file are historical discussion capture.
- Active, authoritative decisions for pre-Phase-11 are in Section 15.8.
- Section 15.9 is resolved.
- Current authoritative addenda are Sections 15.10-15.13.
- This file is the closeout decision register, not the execution worklog.
- Step-by-step execution checkpoints, command/test evidence, and lane completion state are maintained in `PLAN.md` by the orchestrator/integrator.

1. Runtime model:
- Global-first control plane is preferred.
- We still support local context/scoping, but as a mode/view, not as separate product identity.

2. Path + sharing model (this was one of the dropped 2-4 items):
- Team-sharing requires portable references; absolute paths must not be exported.
- Local machines map project roots locally.

3. Attachment semantics (dropped 2-4 item):
- Pointer/reference only (no file copy/snapshot by default).

4. Fuzzy behavior (dropped 2-4 item):
- Fuzzy everywhere for search/filter/find flows.

5. Resource picker keys:
- Keep both `enter` and `a`; behavior must be explicit and predictable.

6. Multi-select actions:
- Invalid actions should be hidden (not merely listed and failing later).

7. Activity log/control:
- You want full functionality now (not passive log only).

8. Onboarding:
- Move to roadmap (not pre-Phase-11 closeout requirement).

9. Config:
- Add `:reload-config` now.
- Also add UI support to manage path/root mapping.

---

## 1) Is `PLAN.md` Mostly Finished?

Yes. Pre-Phase-11 execution scope is complete; remaining items in this file are either:
- historical discussion capture from earlier closeout passes, or
- explicit roadmap items for Phase 11+.

Current assessment:

- Implemented and verified:
  - Phase 3 core config/import-export/delete behavior
  - Phase 5 UX expansion (including command palette, quick actions, multi-select, centered modals, help modal)
  - Phase 6 search/state/command-palette remediations
  - major pre-Phase-11 foundations from Phases 7/8/10 (local-only, no MCP transport)
  - durable activity log UI backed by persisted `change_events`
  - runtime `:reload-config` behavior and `paths/roots` edit modal
  - pre-Phase-11 logging baseline (`github.com/charmbracelet/log` + dev file logs under `.kan/log/`)
  - optional cleanup formerly listed in closeout (`SearchTasks` path removed)
- Intentionally not required for pre-Phase-11 (locked):
  - first-run onboarding is roadmap-only.

Conclusion:

- Pre-Phase-11 is in closeout-complete state; this document remains as the decision register for what was locked and why.

---

## 2) What Is Missing to Complete Pre-Phase 11

No remaining pre-Phase-11 blockers are open.

This section is preserved as historical closeout context from earlier passes. Items listed below were discussion drivers and have since been either:
- implemented and verified, or
- explicitly moved to roadmap with locked scope boundaries.

Authoritative execution status and command/test evidence are in `PLAN.md`.

### A. Product/UX behavior gaps

- Project metadata clarity:
  - unclear `icon` behavior
  - unclear `csv tags` behavior
  - project root/path editing/discovery still confusing
- Task form clarity:
  - due datetime format discoverability and time-entry UX still weak
  - label inheritance hints are information-dense and visually crowded
- Resource attachment UX:
  - `enter` behavior is ambiguous versus `a`
  - path scope policy (project-root vs global fs browsing) needs final decision
  - non-existent typed path handling needs explicit error and guidance
- Search + command + quick actions:
  - fuzzy matching expectations are not met for command abbreviations (`ns`, etc.)
  - quick-action menu should be context-sensitive (single vs multi-select)
- Bulk actions and archived visibility:
  - archived toggle does not provide an obvious "where did archived go?" view
  - selection highlight style and clarity need polish
- Subtree/nesting/dependencies:
  - current subtree mode is technically present but not visually intuitive enough
  - dependency entry/edit UX is not discoverable
- Restore behavior:
  - `u` appears non-functional in user workflow and needs behavior/feedback hardening

### B. System behavior gaps

- Activity log persistence in UI:
  - current modal relies on in-memory log entries
  - persisted event ledger exists but is not surfaced in the modal
- Onboarding:
  - no explicit first-run mode despite plan requirement
- Optional cleanup:
  - remove or integrate currently unused runtime search path (`SearchTasks`)

### C. Config/runtime operations gaps

- Config discoverability:
  - users asked "how do I set group_by/WIP limits?" repeatedly
- Hot reload:
  - no config hot-reload (currently restart-driven config)
- No TUI config editor:
  - currently TOML-first only

---

## 3) Worksheet Notes by Section (Every Section)

This section maps your notes literally across all worksheet sections.

## 0) Test Setup

Observed:

- No blocker noted.

Discussion:

- Keep deterministic temp DB guidance as-is.

Open questions:

- Do we want an in-app `:diagnostics` command that prints DB path + config path + active project root to reduce setup confusion?

## 1) Startup + Baseline Navigation

Observed:

- Startup and keyboard nav passed.

Discussion:

- No functional gap.
- Keep this as baseline regression gate.

Open questions:

- None.

## 2) Project Management

### 2.1 Create project

Observed:

- confusion about icon text rendering
- confusion about `csv tags`
- concern that project local path/root is not obvious enough
- shareability concern: relative paths on export/import

Discussion:

- We need to explicitly define project root UX:
  - what users edit
  - what gets stored in DB
  - what is export-safe
- We need visible inline help on icon/tags meaning.

Open questions:

- Should `icon` be:
  - plain short text glyph
  - emoji allowed
  - optional and hidden if empty?
- Should `tags` be a simple discoverability/filter aid only, or drive defaults (labels/resources/search scope)?
- Should project root live primarily in TOML (`project_roots`) with DB storing only project slug/metadata references?
- Should export include:
  - root alias only
  - root alias + optional path
  - alias only by default, path optionally included via explicit flag?

### 2.2 Edit project

Observed:

- works after discovery; discoverability weak.

Discussion:

- `M edit project` is functional but discoverability can improve in help/status lines and quick actions.

Open questions:

- Should project edit be reachable from:
  - command palette default suggestions
  - project picker context action
  - top-row hint when switching projects?

### 2.3 Project picker

Observed:

- works
- "Why is Inbox default project?"
- display concern about showing both `p/P`.

Discussion:

- `Inbox` as default is common for "capture-first" workflow, but this should be explained once.
- `p/P` dual notation vs single key is a UI copy and future keyspace decision.

Open questions:

- Keep `Inbox` default always, or make default project name configurable?
- Keep dual-case hint text (`p/P`) or canonicalize displayed help to one binding while still accepting aliases?

## 3) Task Create/Edit + Due + Labels

### 3.1 Create task

Observed:

- due format/time entry not clear enough
- hints visually obscured by inherited-label block

Discussion:

- We need explicit due examples in-field and compact progressive hints.
- "show more help" model likely better than always-on dense hint text.

Open questions:

- Should due field accept:
  - `YYYY-MM-DD`
  - `YYYY-MM-DD HH:MM`
  - RFC3339
  - natural language tokens (`tomorrow 5pm`)?
- Should time default to local timezone display but store UTC?

### 3.2 Edit task

Observed:

- resource attach liked, but user wants same in create flow
- desire for resource navigation beyond project root in some cases
- request to ensure attachment is pointer/reference, not file copy
- major strategic question: global multi-project daemon-like usage vs per-project local usage

Discussion:

- This is a core product strategy decision that impacts root/path policy, security defaults, and MCP-era behavior.

Open questions:

- Should resource attach in create mode be:
  - available immediately (`ctrl+r`)
  - deferred until first save?
- Path policy default:
  - project-root constrained
  - workspace constrained
  - unrestricted filesystem
  - constrained-by-default with explicit "dangerous mode" override?
- Attachment semantics:
  - pointer-only (recommended)
  - optional snapshot/copy mode?
- Multi-project runtime model:
  - single global kan instance managing many roots
  - per-project instance only
  - hybrid mode (single instance with opt-in multiple roots)?

### 3.3 Due datetime warnings

Observed:

- passed.

Discussion:

- Keep warning behavior; expand with explicit timezone note and due-time docs.

Open questions:

- Should warning become blocking validation in strict mode?

## 4) Task Info + Resource Picker

### 4.1 Info modal

Observed:

- passed (`i` and `enter`).

Discussion:

- keep as-is.

Open questions:

- None.

### 4.2 Resource attach flow

Observed:

- user expected `enter` to attach; actual flow uses `a`
- no clear warning that `enter` might navigate/open instead of attach

Discussion:

- Need one explicit interaction contract:
  - `enter` on file should attach by default or
  - `enter` only opens and `a` attaches, but must be clearly shown.

Open questions:

- Preferred default:
  - `enter` attach file / open dir
  - `enter` open (all), `a` attach (explicit)?
- For empty dirs, should attach-current-dir on `enter` be enabled?

## 5) Label Inheritance + Label Picker

### 5.1 and 5.2

Observed:

- both references point to earlier concerns:
  - inherited labels explanation density
  - clarity of picker behavior

Discussion:

- inheritance is useful but currently verbose/noisy.
- needs compact, layered explainability.

Open questions:

- Should inherited labels be:
  - collapsed by default with toggle to expand
  - always shown but source-badged in one line?
- Should picker prioritize:
  - phase > project > global
  - or stable alphabetical grouping?

## 6) Search Modal + Filtering

### 6.1 / 6.2

Observed:

- behavior works but user wants fuzzy live feedback and stronger intuitiveness

Discussion:

- Current model is deterministic and stateful; fuzzy layer can improve speed.
- Need consistent fuzzy semantics across search, command palette, quick actions, and resource picker.

Open questions:

- Should board search use:
  - fuzzy by default
  - exact+contains by default with fuzzy toggle?
- Should state/scope filters remain strict while query is fuzzy?

## 7) Command Palette + Quick Actions

### 7.1 Command palette

Observed:

- live filtering exists
- wants abbreviation-style fuzzy matching (`ns` -> `new-subtask`)
- requests file/resource search fuzzy and explicit invalid-path errors

Discussion:

- command execution should never require full command spelling.
- fuzzy ranking and alias boosting should be explicit.

Open questions:

- ranking preference:
  - aliases exact > command prefix > fuzzy score?
- should palette accept short "verbs" (`new sub`, `proj edit`) via tokenized fuzzy?

### 7.2 Quick actions

Observed:

- works, but needs simplification + fuzzy filtering.

Discussion:

- quick actions should be a constrained, context-aware palette, not a full command mirror.

Open questions:

- single command system with scoped views, or maintain separate palette + quick actions?

## 8) Multi-Select + Bulk Actions

### 8.1 Select/unselect

Observed:

- appears to work.

Discussion:

- keep.

Open questions:

- None.

### 8.2 Bulk operations

Observed:

- archived toggle feedback insufficient (status changes, but user cannot "see" archived context clearly)
- user confusion on bracket mappings
- selected marker style disliked
- bulk behavior perceived inconsistent
- quick actions should disable non-bulk-safe commands when multiple selected

Discussion:

- this is a high-priority UX correctness pass.

Open questions:

- archived UI:
  - inline archive column
  - archive overlay/modal
  - project archive list view?
- multi-select visual:
  - pink accent + left rail only
  - keep symbol marker + color
  - both?
- action availability:
  - hard disable invalid actions
  - show disabled with explanation?

## 9) Undo/Redo + Activity Log

### 9.1 Undo/redo

Observed:

- works and makes sense.

Discussion:

- keep.

Open questions:

- None.

### 9.2 Activity log modal

Observed:

- request for interactive control (rollback/jump), not passive log
- future MCP delta implications noted

Discussion:

- This aligns with audit finding: current UI log should consume persisted events.
- "time travel" must be policy-controlled (safe vs destructive operations).

Open questions:

- Should activity log support:
  - inspect only
  - inspect + replay
  - inspect + rollback to checkpoint?
- rollback granularity:
  - per action
  - per task
  - per project snapshot?

## 10) Subtree Projection + Breadcrumb

### 10.1 Focus subtree

Observed:

- feature feels technically present but not conceptually useful enough
- user wants stronger nested/subphase visualization and clearer model

Discussion:

- this is a product model + rendering problem, not just a keybinding issue.
- requires explicit hierarchy semantics:
  - task vs phase vs subtask vs subphase.

Open questions:

- primary hierarchy model:
  - generic work-item tree with `kind`
  - separate task/phase object families?
- board visualization:
  - lane + indented tree rows
  - dual pane (tree left, details right)
  - subtree-only board with breadcrumb + depth indicator?

### 10.2 Clear subtree focus

Observed:

- no blocker noted.

Discussion:

- keep but improve focus-state discoverability.

Open questions:

- Should persistent breadcrumb always show whether focus mode is active?

## 11) Dependency + Rollup Visualization

### 11.1 / 11.2

Observed:

- user cannot discover how to add/edit/use dependencies
- unclear purpose and operational impact

Discussion:

- we need dependency authoring UX in create/edit/task-info flows.
- dependency value should be obvious in board and detail views.

Open questions:

- dependency entry UI:
  - id-based input
  - picker from tasks
  - both?
- should blocked tasks be visually demoted/highlighted?

## 12) Destructive Action Confirmations

### 12.1

Observed:

- `u` reported as no-op.

Discussion:

- likely state-context mismatch (no archived selection / no restore candidate), but UX feedback is insufficient.

Open questions:

- Should `u` always open a restore chooser when no direct candidate exists?

## 13) Grouping + WIP Warnings (Config-Driven)

### 13.1 / 13.2 / 13.3

Observed:

- repeated confusion: "how do I configure this, what does it do, why use it"
- concern about no hot reload
- concern about no TUI config editor

Discussion:

- this is primarily discoverability + operability debt.

Open questions:

- config operations:
  - keep restart-required and document clearly
  - add `:reload-config`
  - add optional file watch/hot reload?
- should we add a minimal in-TUI config editor for common knobs only?

## 14) Help Modal + Discoverability

Observed:

- no blocking note.

Discussion:

- still opportunity to add "task-based walkthrough" mode.

Open questions:

- Should help have tabs:
  - navigation
  - editing
  - search/commands
  - hierarchy/dependencies?

## 15) Final Regression Sweep

Observed:

- no final sign-off entered yet.

Discussion:

- treat this as pending until unresolved sections above are closed.

Open questions:

- None.

---

## 4) Research Findings and How They Affect Decisions

This is the online research summary mapped to your concerns.

### A. Fuzzy matching and command/search UX

Findings:

- Bubbles `list` default filter already uses `sahilm/fuzzy`.
- `SetFilterText` and custom filter behavior are built into the component.
- `taskwarrior-tui` explicitly advertises live filter updates and multiple selection.

Implication:

- We should standardize one fuzzy matching policy across:
  - command palette
  - quick actions
  - resource picker
  - task search query input.

### B. File/resource picker behavior

Findings:

- Bubbles has a `filepicker` model with explicit keymap (`Open`, `Select`) and helper methods (`DidSelectFile`).
- Current custom picker can be retained, but behavior should match user expectations from filepicker conventions.

Implication:

- We should pick one clear key contract and keep it consistent everywhere:
  - "enter opens directories and selects files", or
  - "enter opens, `a` explicitly attaches."

### C. Modal centering and overlay behavior

Findings:

- Lip Gloss provides `Place` / `PlaceHorizontal` / `PlaceVertical` for strict centering.
- Additional-bubbles includes overlay/modal components (`rmhubbert/bubbletea-overlay`) if we want stronger compositing.

Implication:

- Keep current centered modal approach using lipgloss placement primitives unless we hit composition limits.

### D. Multi-project path strategy and portability

Findings:

- Go stdlib `os.UserConfigDir` provides cross-platform config root conventions.
- Go stdlib `filepath.Rel` defines lexical relative-path behavior and error cases.
- `git worktree` official docs confirm linked-worktree semantics and worktree-specific config support.

Implication:

- Best portability model:
  - canonical project root alias + per-machine TOML mapping
  - store resource refs as relative-to-alias by default
  - keep optional absolute path only when explicitly allowed.

### E. Hierarchy/nesting patterns

Findings:

- ClickUp emphasizes explicit hierarchy + subtasks + nested subtasks with breadcrumbs.
- Trello relies on checklists inside cards (flat card with inline subtasks), not full nested tree.
- Taskwarrior uses dependency model (`depends`) and blocked/unblocked states, less visual hierarchy by default.

Implication:

- For kan, a mixed model fits:
  - explicit parent/child for hierarchy
  - dependencies orthogonal to hierarchy
  - clear breadcrumb/focus mode + subtree board view.

### F. Datepicker package compatibility

Findings:

- `ethanefung/bubble-datepicker` latest README is current, but `go.mod` still pins Bubble Tea v0.24.2, Bubbles v0.16.1, Lip Gloss v0.7.1.

Implication:

- Not a drop-in dependency for our v2 stack without adaptation/forking.
- keeping current picker + improving datetime UX is still the lowest-risk path.

---

## 5) Proposed Decisions to Finalize (For Our Next Discussion)

These are the highest-leverage decisions to unlock a fast closeout.

1. Runtime scope model:
- single global kan instance with many projects, or per-project instance by default?

2. Path security model:
- constrained to project root by default, optional "dangerous mode" for wider fs traversal?

3. Resource attach semantics:
- pointer-only references (recommended) with no file copy?

4. Fuzzy policy:
- one unified fuzzy engine across command/search/resource/quick-actions?

5. Archived visibility:
- explicit archive view/modal vs implicit filtering only?

6. Activity log:
- read-only persisted timeline now, interactive rollback later?

7. Onboarding:
- one first-run modal/tutorial required before claiming pre-Phase-11 complete?

8. Hierarchy semantics:
- generic work-item tree with task/phase kinds as one unified model?

9. Dependency UX:
- picker-based dependency editing in task form/info vs raw id entry?

10. Config operability:
- keep restart-required with stronger docs + `:reload-config`, or invest in hot reload now?

---

## 6) Suggested Closeout Sequence (After Discussion)

1. Lock decisions from Section 5.
2. Implement critical correctness:
- durable activity log UI
- onboarding
- `u` restore behavior/feedback
- bulk action correctness + context-aware action menus.
3. Implement UX clarity:
- due/time format guidance
- resource attach key contract
- project/path explanations and docs.
4. Implement discoverability:
- dependency authoring UX
- grouping/WIP and config guidance surfaces.
5. Re-run:
- `just ci`
- VHS tapes
- manual worksheet re-test and sign-off.

---

## 7) External Sources

Charm stack and UI primitives:

- Bubble Tea v2 docs: https://pkg.go.dev/charm.land/bubbletea/v2
- Bubble Tea mouse options: https://pkg.go.dev/github.com/charmbracelet/bubbletea
- Bubbles list (fuzzy filtering): https://pkg.go.dev/charm.land/bubbles/v2/list
- Bubbles module overview (help, list, filepicker): https://pkg.go.dev/charm.land/bubbles/v2
- Bubbles filepicker docs: https://pkg.go.dev/charm.land/bubbles/v2/filepicker
- Lip Gloss placement utilities: https://pkg.go.dev/github.com/charmbracelet/lipgloss/v2
- Additional Bubbles catalog: https://github.com/charm-and-friends/additional-bubbles
- bubbletea-overlay package docs: https://pkg.go.dev/github.com/rmhubbert/bubbletea-overlay

Datepicker compatibility:

- bubble-datepicker package page: https://pkg.go.dev/github.com/ethanefung/bubble-datepicker
- bubble-datepicker `go.mod` (dependency versions): https://raw.githubusercontent.com/ethanefung/bubble-datepicker/main/go.mod

Task-manager behavior references:

- taskwarrior-tui features/readme: https://github.com/kdheepak/taskwarrior-tui
- Taskwarrior task man page (`depends`, status filters): https://taskwarrior.org/docs/man/task.1/
- Taskwarrior FAQ (`BLOCKED`, `UNBLOCKED`): https://taskwarrior.org/support/faq/
- Taskwarrior UDA docs: https://taskwarrior.org/docs/udas/

Hierarchy and subtasks references:

- ClickUp hierarchy overview: https://help.clickup.com/hc/en-us/articles/13856392825367-Intro-to-the-Hierarchy
- ClickUp task/subtask overview: https://help.clickup.com/hc/en-us/articles/10552031987735-Task-View-3-0-overview
- Trello checklists management: https://support.atlassian.com/trello/docs/adding-checklists-to-cards/
- Trello advanced checklist due dates: https://support.atlassian.com/trello/docs/how-to-use-advanced-checklists-to-set-due-dates

Path and portability references:

- Go `os.UserConfigDir`: https://pkg.go.dev/os@go1.25.4
- Go `filepath.Rel`: https://pkg.go.dev/path/filepath@go1.26.0
- Git worktree official docs: https://git-scm.com/docs/git-worktree.html

Config hot-reload considerations:

- fsnotify (cross-platform watcher): https://github.com/fsnotify/fsnotify

---

## 8) Round 2: Direct Responses to Worksheet Notes (Line-Referenced)

This pass maps your comments to exact worksheet lines and current code behavior.

### A. Project metadata/path concerns

Worksheet notes:

- `TUI_MANUAL_TEST_WORKSHEET.md:83` (`icon text`, `csv tags`, missing project path logic)
- `TUI_MANUAL_TEST_WORKSHEET.md:101` (edit project discoverability)
- `TUI_MANUAL_TEST_WORKSHEET.md:121` (`Inbox` default and `p/P` display)

Current behavior:

- Project form captures `icon` and `tags`, but board/header/info views do not render them yet.
- Project roots are currently TOML-driven (`project_roots`), not project-form fields.

Closeout direction:

- Add project-root visibility/edit path in project UX (at least read-only + jump-to-config in this slice).
- Render icon/tag metadata where it actually helps (project picker/details, not cluttered board header).

### B. Task form clarity + due time

Worksheet notes:

- `TUI_MANUAL_TEST_WORKSHEET.md:142` (due format/time unclear, inherited-label hints crowded)
- `TUI_MANUAL_TEST_WORKSHEET.md:161` (resource attach in create flow + global/path policy)

Current behavior:

- Time format is supported in parser, but hints are dense and easy to miss in crowded forms.
- Resource attachment exists in edit/info; create flow is incomplete ergonomically.

Closeout direction:

- Keep datetime support, simplify hint hierarchy, and add clearer due examples inline.
- Enable attach flow consistently in both create and edit where possible.

### C. Resource picker behavior

Worksheet notes:

- `TUI_MANUAL_TEST_WORKSHEET.md:227` (`enter` ambiguity vs `a`)
- `TUI_MANUAL_TEST_WORKSHEET.md:342` (typed invalid path and file-search expectations)

Current behavior:

- `enter` can open directories and may attach by context; `a` explicitly attaches.
- No explicit typed path validation feedback model in picker UI.

Closeout direction:

- Adopt explicit key contract and surface it clearly.
- Add typed path validation + "not found" feedback.
- Add fuzzy filter in picker list.

### D. Search/palette/actions fuzzy requests

Worksheet notes:

- `TUI_MANUAL_TEST_WORKSHEET.md:296`, `318`, `342`, `362` (fuzzy everywhere)

Current behavior:

- Current command filtering uses substring matching, not robust fuzzy abbreviation behavior.
- Search flow is functional but not "fuzzy-first."

Closeout direction:

- Unify fuzzy behavior across:
  - command palette
  - quick actions
  - resource picker
  - task query search.

### E. Multi-select, archive visibility, action validity

Worksheet notes:

- `TUI_MANUAL_TEST_WORKSHEET.md:406` (archive visibility, bulk reliability, selection styling, quick-action validity filtering)

Current behavior:

- archive toggle is status-driven and not discoverable enough visually.
- quick actions include operations that are nonsensical for multi-select context.

Closeout direction:

- context-sensitive quick-action list for multi-select.
- clearer selected styling (accent rail, remove noisy symbols).
- explicit archived-items view path (modal/list), not only toggle text.

### F. Activity log expectations

Worksheet notes:

- `TUI_MANUAL_TEST_WORKSHEET.md:451` (interactive control, not passive view; forward MCP audit implications)

Current behavior:

- UI log is mostly in-memory and not full interactive control.

Closeout direction:

- treat log as operational timeline with controlled replay/revert actions.
- persist and surface actor-aware changes consistently.

### G. Hierarchy and dependency discoverability

Worksheet notes:

- `TUI_MANUAL_TEST_WORKSHEET.md:475` (subtree mode not conveying nested workflow)
- `TUI_MANUAL_TEST_WORKSHEET.md:514`, `533` (dependency creation/use unclear)

Current behavior:

- subtree filter exists but does not yet communicate structure strongly enough.
- dependency data renders in info but authoring is not discoverable in normal flow.

Closeout direction:

- improve nested visual semantics (expand/collapse and depth context).
- add dependency authoring UX and explain dependency effects in UI.

### H. Restore key behavior

Worksheet notes:

- `TUI_MANUAL_TEST_WORKSHEET.md:556` (`u` appears no-op)

Current behavior:

- restore requires archived selection or prior archived context; this can appear like no-op without context.

Closeout direction:

- add restore chooser/fallback and explicit status guidance when no restore candidate exists.

### I. Config operability

Worksheet notes:

- `TUI_MANUAL_TEST_WORKSHEET.md:578`, `594`, `614` (how to use grouping/WIP; no reload; no in-TUI config updates)

Current behavior:

- config is TOML-driven and restart-based.

Closeout direction:

- improve discoverability and add explicit config reload command path before full hot-reload.

---

## 9) Refined Clarifying Questions (Architecture-First)

These are the decision questions to finalize before implementation.

1. Global vs local runtime:
- Should kan be a global multi-project control plane by default, with optional "current-dir scoped view" as a mode rather than a separate app model?

2. Project identity:
- Should project identity be anchored on Git remote + logical slug, so local paths can differ per teammate while exports remain portable?

3. Root mapping:
- Should exports contain only stable root aliases and relative paths (default), with absolute paths allowed only in explicit dangerous mode?

4. Worktree model:
- For one repo with many worktrees/branches, do you want:
  - one project with many execution contexts, or
  - separate projects per worktree/branch?

5. Resource attach key contract:
- Confirm preferred interaction:
  - `enter`: open dir / attach file
  - `a`: force attach selected entry
  - both shown in hints.

6. Archived experience:
- Should archived items be a dedicated modal/list (Trello-style) instead of only inline board toggling?

7. Multi-select action policy:
- Should invalid actions be hidden entirely in multi-select contexts, or shown disabled with reason text?

8. Activity log control:
- For "full functionality now," should we allow:
  - revert one action
  - revert to checkpoint
  - both, with hard-delete revert gated by policy?

9. Hierarchy semantics:
- Should we standardize on unified work-item tree (`phase`, `task`, `subtask`) with shared lifecycle and completion rules?

10. Dependency behavior:
- Should dependencies be same-project only by default, with cross-project dependencies as an opt-in advanced mode?

11. Config operations:
- Should we ship `:reload-config` first and postpone full file-watcher hot reload, or do you want both now?

---

## 10) Big-Picture Architecture Questions (Before Execution Details)

These questions are intentionally not low-level. They define what kan is for.

1. Source of truth:
- Is kan the authoritative planning system, or should it eventually sync/mirror external systems (Jira/Linear/GitHub issues) with kan as an operator-friendly local facade?

2. Agent handoff model:
- When an agent reconnects, should kan provide:
  - only "delta since last cursor" summaries, or
  - delta + current authoritative snapshot (recommended)?

3. Context budget strategy:
- For LLM efficiency, should kan emit:
  - compact semantic diffs (status/owner/dependency changes),
  - with optional drill-down into full event history only on demand?

4. Boundary ownership:
- Should boundaries be rooted at:
  - project level only,
  - project + branch/worktree execution context,
  - or phase lanes within project?

5. Team model:
- For future remote/team mode, do we want:
  - optimistic local edits with server reconciliation,
  - or strict server-authoritative edits with local cache?

6. Dangerous operations policy:
- Should dangerous-mode permissions be:
  - global toggle,
  - per project,
  - per tool/action class (delete, hard-delete, path escape, etc.)?

---

## 11) Path/Root Strategy (Global Tool + Team Shareable)

Your stated goal is global, multi-project operation with safe boundaries and team portability.
The clean model is:

- DB stores stable identifiers and portable refs:
  - `project_slug`
  - optional repo identity (URL, provider, default branch)
  - resource refs as `{root_alias, relative_path}`
- Local TOML stores machine-specific root resolutions:
  - `[project_roots]`
  - `slug = "/Users/.../repo"`

### Import Resolution Flow (Recommended)

On import, if any `root_alias` cannot be resolved locally:

1. open "Resolve Project Roots" screen
2. block normal board use until each required root is mapped or deferred explicitly
3. verify mapping (directory exists + optional repo identity check)
4. write resolved paths to local TOML only
5. continue

This directly matches your requirement: no absolute export, team-specific local resolution, and explicit user control.

### Why this matches best practice

- MCP roots are intended as context boundaries, typically managed by workspace/project UI.
- Roots are advisory at protocol level, so app-level + OS-level checks still matter.
- Multi-root workspace tools (like VS Code) use layered settings (global + workspace/folder), which maps well to TOML root mapping.

---

## 12) ASCII UX Options (Archived + Hierarchy + Dependencies)

These options are meant for decision-making, not final styling.

### A) Archived Items UX

Option A1: Full-screen Archive View (recommended)

```text
kan  Inbox  [archive]
┌────────────────────────────────────────────────────────────────────────────┐
│ Archived Work Items (Project: Inbox)                                      │
│ Filter: [text fuzzy________________]  State: [archived]                   │
│                                                                            │
│ > Draft roadmap                      archived 2026-02-21 by user          │
│   Refactor parser                    archived 2026-02-20 by agent         │
│                                                                            │
│ [enter] inspect   [u] restore   [D] hard delete   [/] filter   [esc] back │
└────────────────────────────────────────────────────────────────────────────┘
```

Pros:
- clear destination for `t`/archive workflows
- scales well for many archived items
- aligns with Trello/Jira "archived items" style

Cons:
- mode switch (not inline with board)

Option A2: Center modal archive list

```text
             ┌──────────────── Archived ────────────────┐
             │ > Draft roadmap       archived 2d ago    │
             │   Refactor parser     archived 3d ago    │
             │                                           │
             │ [u] restore  [D] delete  [esc] close     │
             └───────────────────────────────────────────┘
```

Pros:
- quick to implement
- keeps board context visible behind modal

Cons:
- cramped for large archives
- harder to support rich filtering/history

Option A3: Fourth board column "Archived"

```text
[To Do]   [In Progress]   [Done]   [Archived]
```

Pros:
- no mode switch

Cons:
- mixes active and inactive lifecycle visually
- can clutter board and confuse WIP flow

### B) Hierarchy / Subtree UX

Option B1: Board with indented tree rows (recommended baseline)

```text
[To Do]
> Phase: API Cleanup
  ├─ Task: Update handlers
  │  ├─ Subtask: add tests
  │  └─ Subtask: docs
  └─ Task: remove legacy routes
```

Pros:
- minimal disruption to existing kanban view
- clear parent/child context
- works with subtree focus

Cons:
- deep nesting can get visually dense

Option B2: Two-pane mode (tree left, board right)

```text
┌────────────── Hierarchy ──────────────┐ ┌──── Board (selected node) ────┐
│ Phase: API Cleanup                     │ │ To Do      In Progress   Done  │
│  ├─ Update handlers                    │ │ ...filtered to selected subtree │
│  └─ Remove legacy routes               │ │                                  │
└────────────────────────────────────────┘ └──────────────────────────────────┘
```

Pros:
- strongest mental model for nested work
- easy subtree navigation

Cons:
- more complex UI/state management
- less space per panel in small terminals

Option B3: Parent-centric detail modal with subtask list

```text
Task Info: Phase API Cleanup
- child items:
  [ ] Update handlers
  [ ] Remove legacy routes
```

Pros:
- simple incremental step

Cons:
- weaker board-level hierarchy visibility

### C) Dependency UX

Option C1: Relation picker in task edit/info (recommended)

```text
Dependencies
depends_on: [ + add relation ]   (fuzzy task picker)
blocked_by: [ + add relation ]
blocked_reason: [text________________]
```

Pros:
- discoverable, explicit, low ambiguity
- consistent with ClickUp/Linear relationship editing patterns

Cons:
- needs careful cycle/invalid-edge validation

Option C2: Command-driven dependency add

```text
: dep add blocked-by <task>
: dep add depends-on <task>
```

Pros:
- fast for power users

Cons:
- poor discoverability for most users

Recommendation:
- ship C1 first, add C2 alias commands later.

---

## 13) Responses to Your Latest Notes (2-4, 7, 11, 12, 13, and #10 follow-up)

### "2-4 went away" recovery

Re-stated cleanly:

- (2) Path boundaries should be local-resolved, share-safe, and import-blocking until mapped.
- (3) Pointer-only resource references are correct.
- (4) Unified fuzzy behavior everywhere is correct.

### Archived discussion (#7)

Best initial choice:
- A1 full-screen archive view.

Reason:
- Trello and Jira both expose archived work in explicit archived destinations rather than silently in active boards.

### Hierarchy discussion (#11)

Best initial choice:
- B1 (indented tree rows) first, with optional B2 later if needed.

Reason:
- lower implementation risk, keeps current board mental model, still supports subtrees/phases.

### Dependency discussion (#12)

Meaning in this app:
- `depends_on`: items this item requires before progress/completion.
- `blocked_by`: explicit blockers (often mirrors depends_on but can capture external blockers too).

Recommended policy:
- same-branch dependencies by default.
- cross-branch dependencies configurable/opt-in.

### Config operations (#13 and prior #10 follow-up)

Proposed immediate scope:
- implement `:reload-config`.
- add "Paths/Roots" config modal in TUI for local mapping changes.
- keep full fs watcher hot reload optional/deferred unless you want it in this same closeout wave.

---

## 14) Updated Source Notes for the Above Decisions

- MCP roots/boundaries and UI model:
  - https://modelcontextprotocol.io/specification/2025-03-26/client/roots
  - https://modelcontextprotocol.io/docs/learn/client-concepts
- Multi-root configuration layering patterns:
  - https://code.visualstudio.com/docs/editing/workspaces/multi-root-workspaces
- Trello archived-items model:
  - https://support.atlassian.com/trello/docs/archiving-and-deleting-cards/
- Jira archived work item model:
  - https://support.atlassian.com/jira-software-cloud/docs/archive-an-issue/
- ClickUp dependencies model:
  - https://help.clickup.com/hc/en-us/articles/6309155073303-Intro-to-Dependency-Relationships
- Linear parent/sub-issue + dependency relations:
  - https://linear.app/docs/parent-and-sub-issues
  - https://linear.app/docs/issue-relations
- Taskwarrior dependency emphasis:
  - https://taskwarrior.org/docs/best-practices/

---

## 15) Third-Pass Consensus Update (2026-02-22)

This section captures the latest decisions from today so they are not lost.

### 15.1 Clarification: "Track-only vs execution-enabled"

- `kan` itself is always planning/tracking software.
- It does not execute code-change workflows directly.
- Any execution policy belongs to external orchestration and remains roadmap-only.

### 15.2 Project Path Policy (Required vs Optional)

- Project path/root is optional for personal/local task tracking.
- Project path/root should be represented with one explicit state only:
  - `workspace_linked = true|false`
- No additional "automation eligibility" flag is needed in pre-Phase-11.
- TUI UX:
  - On project create/edit, show a warning when no path is configured.
  - Mark unlinked projects visually so users understand file/dir resource attachments are limited.
  - If unlinked, allow URL resources but block filesystem path resources.

### 15.3 Import/Export and Root Resolution

- Never export absolute paths.
- Export portable refs only (`root_alias`, relative resource paths, optional repo identity metadata).
- Import flow must block and validate all references:
  1. detect unresolved aliases,
  2. open root-resolution modal queue,
  3. verify local directory exists,
  4. verify referenced relative files/dirs resolve under the mapped root,
  5. optionally verify repo identity and expected branch metadata,
  6. write local mappings to TOML only,
  7. fail import if validation fails.
- Pre-Phase-11 policy:
  - strict fail for unresolved relative path references (no partial import success).
  - advanced divergence-repair flows are roadmap only.
- Future flexibility (roadmap):
  - add guided reconciliation when imports diverge by branch/commit/path layout.
- Local-only mode:
  - supported for projects created locally without linked roots.
  - for imported shared project snapshots with path references, unresolved refs must fail in pre-Phase-11.

### 15.4 Archived/Hierarchy/Dependency Direction

- Archived UX: proceed with full-screen archive view first (A1).
- Hierarchy UX: proceed with board tree-first model (B1) before full-screen phase board nesting.
- Dependency UX: relation picker in task edit/info first (C1), command aliases later.

### 15.5 Data Model Direction

- Keep SQLite for pre-Phase-11 and early product maturity.
- Keep schema relational with explicit relation tables (parent-child, dependencies, blockers, resources, events).
- Do not move to graph DB for current scope.
- Recommended operational model:
  - single local DB by default,
  - project-scoped export/import packages for sharing and merge workflows.
- Roadmap: optional service mode and/or Postgres backend later; design storage interfaces now to keep migration tractable.

### 15.6 Lifecycle and Terminology

- Canonical lifecycle remains fixed: `todo | progress | done | archived`.
- UI display label for `progress` is `In Progress`.
- Keep labels/config/user customization around filtering and presentation, not lifecycle taxonomy.
- Add a lexicon section (docs + UI copy) so terminology is consistent across levels.
- Proposed neutral hierarchy terms:
  - Portfolio (optional future)
  - Project
  - Phase
  - Task
  - Subtask
- Optional Swedish-theme aliases can be presentation-only, not schema names.

### 15.7 Recovered Missing Discussion Items (5-7)

- (5) Undo model: checkpoint-style undo/revert is preferred over noisy per-keystroke history.
- (6) Agent handoff/event model: summarize meaningful final-state deltas since last cursor, not every intermediate edit.
- (7) Nested work UX: keep board continuity now; evaluate deeper full-screen nested views as follow-up after initial hierarchy hardening.

### 15.8 Open Questions to Resolve Next

Final locked decisions for pre-Phase-11:

1. Import safety:
- strict fail on unresolved relative file/dir references in imported shared snapshots.
- never export absolute paths.
- advanced divergence repair remains roadmap-only.

2. Workspace link model:
- single project flag only: `workspace_linked = true|false`.
- unlinked projects remain valid for planning.
- URL resources allowed while unlinked; filesystem path resources blocked until linked.

3. Data/backend:
- SQLite relational remains the implemented backend.
- single local DB default + project-scoped export/import sharing.
- graph DB is out-of-scope.

4. Hierarchy and lifecycle:
- standard hierarchy: `Project -> Branch -> Phase -> Task -> Subtask`.
- default branch `main` auto-created.
- projects are not nested; branches are not nested.
- phases can nest (subphases).
- tasks can have subtasks only.
- canonical lifecycle stays fixed: `todo | progress | done | archived`.
- UI display label for `progress` is `In Progress`.

5. Completion gating:
- moving to `done` must enforce child-complete checks and completion-contract checks.
- this applies across nested levels by role (phase/task/subtask semantics).

6. UX direction:
- archived first ship: full-screen archive view (A1).
- hierarchy first ship: tree-in-board rendering (B1).
- dependencies first ship: relation picker in info/edit (C1).

7. Terminology:
- ship English-first terminology in UI/docs.
- Swedish/brand-style labels, if added, are presentation-only and roadmap-level.

8. Scope guard:
- MCP/HTTP integration and external connectors (GitHub/Jira/Slack/etc.) are roadmap-only.
- no pre-Phase-11 implementation work in those areas.

### 15.9 Truly Unresolved (Non-Roadmap, Needs Final User Lock)

Resolved (2026-02-22):

1. Dependency scope default:
- locked to option A:
  - same-branch only by default,
  - cross-branch explicit opt-in.

2. Repo identity check strictness:
- locked to option B:
  - warning + explicit user-confirm continue when configured repo identity mismatch is detected.

### 15.10 Phase 11 Roadmap Consensus Addendum (Attention + Delta Delivery)

- This is roadmap-only for MCP/HTTP phases; no pre-Phase-11 transport implementation.
- Add first-class attention fields for work items:
  - `attention_state` enum: `none|note|unresolved`
  - `attention_note`
  - `attention_set_by`, `attention_set_at`
  - `attention_cleared_by`, `attention_cleared_at`
- Branch-scoped context contract for future MCP calls:
  - every branch-context request returns meaningful changes at/below that branch since cursor
  - same response includes active attention items (all open `note|unresolved`)
- Delivery/cursor semantics:
  - cursor scope is `(agent_id, branch_id)`
  - cursor advances only on explicit ack
  - without ack, same unseen changes are resent deterministically
- Event noise policy:
  - capture committed writes/events only
  - do not capture per-keystroke modal typing
- Session metadata:
  - may be included as optional diagnostics
  - is not required for correctness
  - branch scope is authoritative for delivery guarantees

### 15.11 Locked Delivery Defaults for Phase 11 Planning

- Cursor and ack:
  - `ack_changes` is the canonical cursor-advance mechanism.
  - default reads (`get_branch_context`) do not advance cursor automatically.
  - no ack means deterministic resend of unseen branch-scoped changes.
- Attention defaults:
  - one active attention record per work item (`none|note|unresolved` + note + audit metadata).
  - `unresolved` blocks `progress -> done` by default.
  - override path is policy-controlled and must be actor-attributed with reason.
- Delta scope policy:
  - include changes at/below branch scope.
  - include project/config changes only when they affect that branch context.
  - exclude unrelated project/global churn from default branch payload.
- Agent attention behavior:
  - agents may set attention flags by default.
  - clearing `unresolved` requires user approval by default (configurable policy).

### 15.12 Logging Baseline Lock (Pre-Phase-11, In Scope Now)

- Logging is not deferred:
  - adopt `github.com/charmbracelet/log` in current implementation scope.
  - leverage color/styling for local developer visibility.
  - log meaningful runtime operations and failures, not just terminal status hints.
- Dev-mode local file logging is required now:
  - configuration must expose a dev mode toggle.
  - default dev log directory should be workspace-local `.kan/log/`.
- Error-handling expectation:
  - keep idiomatic wrapped error chains and bubble errors across boundaries.
  - log boundary failures with enough context for troubleshooting.
- Service observability expansion:
  - broader observability pipeline work remains deferred until after MCP/HTTP slices.

### 15.13 Phase 11.0 Gate Lock (MCP-Go + Stateless HTTP Design Review)

- Before Phase 11 implementation starts, require a dedicated 11.0 research/discussion gate:
  - no coding in this gate.
  - review `mcp-go` stateless HTTP-served MCP patterns in the context of current hexagonal architecture.
  - review dynamic tool discovery/update behavior:
    - https://modelcontextprotocol.io/legacy/concepts/tools#tool-discovery-and-updates
- Goal:
  - ensure this MCP plan is suitable for internal dogfooding as part of the broader system roadmap.
- Guidance:
  - existing schema and contract locks should largely support this, but the agent must review all current design artifacts and surface any new findings before 11.1+ coding.

### 15.14 Re-opened Pre-Phase-11 Blockers (Live QA Override)

Status (2026-02-22):
- pre-Phase-11 is re-opened due to user-confirmed runtime UX regressions.
- prior closeout claims remain historical, but are superseded by this blocker list until resolved and re-verified.

Locked remediation set for this re-open:

1. Board row/subtask rendering:
- subtasks are hidden from board row listing.
- board rows show only parent work item summary + compact subtask completion count (`x/y`) near metadata.
- selected/focused marker applies only to focused row, not every row.

2. Column viewport/scroll behavior:
- board columns keep fixed height independent of task count.
- cursor navigation and wheel scrolling must keep focused row inside visible viewport window.
- no off-screen selection drift.

3. Task info/subtask consistency:
- subtask data must remain visible in info flow regardless of parent column/state.
- task info modal should be the entry point for subtask exploration/editing (modal drill-in).

4. Runtime artifact hygiene:
- workspace/runtime artifacts are gitignored and deterministic:
  - `.kan/`
  - `.go-mod-cache/`
- dev logs should not materialize under package directories during test execution.

5. Verification/documentation refresh:
- VHS scenarios must be expanded/updated to catch the above regressions.
- manual worksheet must be reissued with machine-readable user note anchors:
  - `### USER NOTES Sx.y-Nz`

Execution note:
- `PLAN.md` is the sole execution/worklog ledger for this remediation wave.
- this closeout doc remains a decision register and blocker definition source.

### 15.15 Product-Intent Clarification (Locked for Pre-Phase-11 docs)

`kan` is not just a generic board UI; it is intended to be the human-readable, auditable coordination layer for developer + coding-agent workflows that currently degrade when managed with markdown-only checklists.

Locked documentation implications:
- pre-Phase-11 behavior and docs should already optimize for clear human↔agent handoff quality, even before MCP transport exists.
- manual test communication should use stable anchored notes to reduce ambiguity and context loss.
- roadmap language for MCP/HTTP should preserve this contract: authoritative state + explicit completion semantics + visible deltas, not free-form status prose.

Operational doc lock:
- `TUI_MANUAL_TEST_WORKSHEET.md` uses section anchors in the required form:
  - `### USER NOTES Sx.y-Nz`
  - this is now the preferred shorthand format for tester↔agent discussion during pre-Phase-11 closeout.
