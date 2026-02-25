# Collaborative E2E Remediation Plan + Worklog

Created: 2026-02-25  
Owner: orchestrator (Codex)  
Status: planning locked, implementation not started  
Primary test source: `COLLABORATIVE_FULL_E2E_TEST_WORKSHEET.md`  
Evidence root: `.tmp/collab-e2e-20260225_080750/`

## 1) Scope + Non-Negotiables

1. Include all user-stated expectations and fixes captured during the collaborative run.
2. Execute with parallel subagents where lock scopes allow it.
3. Preserve gatekeeping boundaries (orchestrator vs worker, lease scope restrictions, actor attribution).
4. Use `just` recipes for all checks/tests.
5. Finalization order is locked:
   1. all code/test changes integrated,
   2. `just check`, `just ci`, `just test-golden` passing,
   3. update `README.md` + other docs/templates,
   4. clean up outdated files,
   5. commit.

## 2) Locked Expectation Inventory (Complete)

This is the full remediation inventory from Section `0.x`, Sections `C/D/E`, and final C13 findings.

| ID | Area | Required fix/expectation | Source |
|---|---|---|---|
| R-01 | Input/select mode | `v` must not break typing in text inputs (including bootstrap/startup modal). | 0.4.1, C9 |
| R-02 | Input fidelity | All typing keys must work in text inputs; emoji input supported where text is accepted. | 0.4.1, C9, C10 |
| R-03 | Select/copy keymap | Move selection mode off `v`; expected direction: control chord (`Ctrl+Y`), with `Y`/`Shift+Y` yank/copy semantics. | 0.4.12, C9 |
| R-04 | Help coverage | `?` help must be available in all modals/screens. | 0.4.11, C11 |
| R-05 | Help content | Help must include copy/paste/select shortcuts (including `Ctrl+V`) and modal-specific guidance. | 0.4.11, user notes, C11 |
| R-06 | Due picker time | `space` include-time toggle must work. | 0.4.8, C9 |
| R-07 | Due picker fuzzy | Date typing should fuzzy-filter (example `2-2` behavior). | 0.4.8, C9 |
| R-08 | Live refresh | MCP-originated changes should refresh in current project without project-switch workaround. | 0.4.4, C1 |
| R-09 | Notifications panel | Implement level-scoped notifications/notices panel with global count and quick navigation controls at panel bottom. | 0.4.6, C6 |
| R-10 | Warning/error surfacing | Important runtime/MCP warnings/errors must surface in notifications panel + quick info modal drill-in. | 0.4.5/6, C6 |
| R-11 | Logging | Full runtime + MCP logging with `github.com/charmbracelet/log`; ensure meaningful bubbling at adapter/runtime edges. | 0.4.5, C6 |
| R-12 | Nav back behavior | `esc` should step back contextually (browser-back style), not jump to project root by default. | 0.4.10, C3/C4/C8 |
| R-13 | Info modal return | Info modal must return to exact modal-open origin state. | C4 |
| R-14 | Branch creation flow | Branch creation must support required path parameter behavior. | 0.4.7, C12 |
| R-15 | Scoped phase creation | `new-phase` / `new-subphase` must work from selected and focused-empty scopes. | locked C12 requirement |
| R-16 | Focus guardrail | `new-branch` must be blocked during subtree focus with warning modal guidance. | locked C12 requirement |
| R-17 | Icon semantics | Define icon feature purpose/UX and make icon behavior visibly functional. | 0.4.13, C10 |
| R-18 | Marker clarity | Keep/clarify metadata marker meanings (`!3`, overdue marker) in help/legend. | 0.4.9, C5 |
| R-19 | Archive confirm nouns | Entity-specific confirm labels (for example `archive branch`, not `archive task`). | C13 |
| R-20 | Restore shortcut | Add `.` quick action for restoring archived items. | user C13 follow-up |
| R-21 | Archive key policy | Remove plain `a` as archive trigger; archive/restore via `.` quick action + `:` command palette. | user C13 final notes |
| R-22 | Archived project visibility | Archived projects hidden by default in top-level and picker; explicit picker toggle for archived visibility. | C13 final |
| R-23 | Project-level archived UX | Rework project-level archived toggle UX (`t` not ideal for project visibility mode). | C13 final |
| R-24 | Search archived semantics | Archived search filter must return archived matches without requiring separate global `t` coupling. | C13 final |
| R-25 | Agent spoof prevention | Agent lease tuple must not be able to persist `actor_type=user` mutations. | E5 critical finding |
| R-26 | Attribution correctness | User vs agent attribution must be distinct and correct (user identity from bootstrap persona path). | 0.6, 0.7 |
| R-27 | Gatekeeping enforcement | Keep strict scope constraints for orchestrator/workers; no out-of-scope mutations. | 0.7, spawn evidence |
| R-28 | Agent approval UX | Add notification-driven approval flow for agent identity + scope requests, with client-side approval continuation support. | 0.7 |
| R-29 | Template coverage | Update templates/docs so approval/scope flow is encoded (`AGENTS.md` plus `CLAUDE.md` path decision). | 0.7 template findings |
| R-30 | Notifications-first design process | For notifications panel changes, start with ASCII-art proposal + clarifying questions before implementation. | 0.4.14 |
| R-31 | C13 pass caveat | Keep recorded: lifecycle mostly passes, but archived visibility/search and key UX currently fail expectations. | C13 final |

## 2.1 Regression invariants (must remain true while fixing failures)

1. Project scope still renders immediate children only (C1).
2. Path line remains visible and correct (C2).
3. Focus drill-down (`f`) level behavior remains correct (C3).
4. Hierarchy markers remain present and accurate (C5).
5. Scoped create (`n`) keeps parent scope behavior (C7).
6. Empty-leaf focus + first-child create behavior stays working (C8).
7. Section D guardrails/parity remain passing.
8. Section E tool-sweep coverage remains passing.

## 3) Parallel Subagent Execution Setup

## 3.1 Roles

1. Orchestrator/integrator:
   - owns decomposition, file-lock assignment, merges, conflict resolution, final gates.
2. Worker lanes:
   - execute only within lane lock scopes.
   - run package-scoped `just test-pkg` loops only.
3. Integrator closeout:
   - runs repo-wide gates (`just check`, `just ci`, `just test-golden`) after lane integration.

## 3.2 Lane Map (Wave 1 parallel)

| Lane | Objective | Lock scope (allowed) | Explicit out-of-scope | Lane checks |
|---|---|---|---|---|
| L1-GATEKEEP | Fix actor-type spoofing and attribution enforcement (`R-25`..`R-27`) | `internal/app/**`, `internal/domain/**`, `internal/adapters/storage/sqlite/**`, `internal/adapters/server/mcpapi/**`, related tests | `internal/tui/**`, docs cleanup | `just test-pkg ./internal/app`, `just test-pkg ./internal/adapters/storage/sqlite`, `just test-pkg ./internal/adapters/server/mcpapi` |
| L2-LOGGING | MCP/runtime logging + error bubbling (`R-10`,`R-11`) | `internal/adapters/server/**`, `internal/app/**` (logging surfaces only), related tests | `internal/tui/model.go`, lifecycle/keymap behavior | `just test-pkg ./internal/adapters/server/httpapi`, `just test-pkg ./internal/adapters/server/mcpapi` |
| L3-TUI-HOTSPOT | TUI behavior fixes (`R-01`..`R-24`,`R-30`,`R-31`) | `internal/tui/**`, TUI tests/golden fixtures | app/storage guardrail code | `just test-pkg ./internal/tui`, `just test-golden` (lane-local allowed only if assigned) |

Hotspot note: `internal/tui/model.go` is serialized under `L3-TUI-HOTSPOT` ownership for the entire wave.

## 3.3 Lane sequencing constraints

1. `L1-GATEKEEP` and `L2-LOGGING` may run in parallel.
2. `L3-TUI-HOTSPOT` may run in parallel with `L1/L2` but no other lane may edit `internal/tui/model.go`.
3. Any cross-lane dependency requires orchestrator checkpoint before merge.

## 3.4 Worker prompt contract (execution-ready)

Each spawned worker prompt must include:

1. lane id + single acceptance objective.
2. lock scope and explicit out-of-scope files.
3. required tests (`just test-pkg` scoped to touched packages only).
4. Context7 checkpoints for code-change turns:
   - before first code edit,
   - again after any failed test/runtime error.
5. handoff payload:
   - files changed and why,
   - commands run and outcomes,
   - acceptance checklist pass/fail,
   - unresolved risks.

## 4) Implementation Plan by Requirement Group

## 4.1 Gatekeeping + Identity

1. Enforce actor-type/lease consistency so `actor_type=user` cannot be accepted with agent lease tuple.
2. Validate user attribution path and bootstrap persona mapping expectations.
3. Preserve and re-test fail-closed scope boundaries for orchestrator/workers.
4. Expand tests to include spawned-worker regression probes.

Acceptance:
1. Existing spoof repro fails closed.
2. Cross-project mutation remains blocked.
3. Attribution fields match actor type + identity source.

## 4.2 TUI Input/Navigation/Help/Due

1. Remove `v` typing conflict and redesign select-mode hotkeys.
2. Ensure text-entry runes pass through in all modal inputs.
3. Implement contextual `esc` back-stack behavior and modal-origin return.
4. Fix due picker include-time toggle and fuzzy date behavior.
5. Update help overlays with full modal-specific shortcut coverage.

Acceptance:
1. C4/C9/C11 reruns pass.
2. Startup/bootstrap input supports full typing without select-mode collisions.

## 4.3 Notifications + Runtime Visibility

1. Produce ASCII-art + clarifying question checkpoint before coding notifications panel.
2. Implement level-scoped notifications panel with global count and quick-nav.
3. Surface key runtime/MCP warnings/errors into notifications + quick info modals.
4. Verify logs emitted with `charmbracelet/log` and useful context.

Acceptance:
1. C6 rerun passes.
2. User can navigate notification items and inspect details without leaving workflow context.

## 4.4 Branch/Project Lifecycle + Search/Archived Semantics

1. Fix branch path parameter flow and C12 command matrix.
2. Keep `new-phase`/`new-subphase` selected + focused-empty behavior.
3. Enforce `new-branch` block under subtree focus.
4. Fix C13 archive confirm nouns and archive/restore key policy.
5. Add `.` quick restore.
6. Add explicit archived visibility control in project picker.
7. Hide archived projects by default in picker/top-level views unless explicitly enabled.
8. Decouple archived search filtering from global `t` visibility coupling.

Acceptance:
1. C12 and C13 reruns pass without caveats.
2. Archived search works when archived filter is selected even if board-level archived visibility is hidden.

## 4.5 Icon + Emoji Product Behavior

1. Define icon purpose and display surfaces (header, picker row, tabs, etc.).
2. Ensure emoji input and persistence across relevant forms.
3. Add tests and help/documentation notes.

Acceptance:
1. C10 rerun passes.

## 4.6 Templates + Approval Flow Docs

1. Update `AGENTS.md` workflow guidance for agent identity/scope approval payloads and notification expectations.
2. Decide and implement `CLAUDE.md` template handling (create or explicitly document absence/alternative).
3. Sync README and relevant planning docs.

Acceptance:
1. Template-check scenario no longer reports missing identity/scope approval guidance.

## 5) Integration + Verification Plan

1. Integrator merges lane outputs only after lane acceptance evidence is present.
2. Full repository gates after merge:
   1. `just check`
   2. `just ci`
   3. `just test-golden`
3. Manual rerun focus:
   1. C4, C6, C9, C10, C11, C12, C13
   2. gatekeeping spoof/cross-scope checks from Section E evidence.

## 6) Finalization Sequence (Locked by user request)

After tests pass and before commit:

1. Update `README.md` and other affected docs/templates with final behavior and keymaps.
2. Clean up old/outdated files that are superseded by this remediation work.
3. Confirm working tree only contains intended changes.
4. Commit.

## 7) Worklog

## Checkpoint 000 (2026-02-25)

1. Created this remediation plan/worklog per user request.
2. Imported complete expectation inventory from collaborative worksheet + C13 final findings.
3. Defined parallel lane setup and locked finalization order:
   - tests first (`just` gates),
   - then docs update + outdated-file cleanup,
   - then commit.

Next checkpoint:
1. spawn worker lanes `L1-GATEKEEP`, `L2-LOGGING`, `L3-TUI-HOTSPOT` with lock scopes and acceptance criteria.
