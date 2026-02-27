# Collaborative E2E Remediation Plan + Worklog

Created: 2026-02-25  
Owner: orchestrator (Codex)  
Status: planning locked, implementation not started  
Primary test source: `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`  
Recovered baseline source: `.tmp/old-worksheet-audit/COLLABORATIVE_FULL_E2E_TEST_WORKSHEET.HEAD.md`  
Evidence root: `.tmp/collab-e2e-20260225_080750/`

## 1) Scope + Non-Negotiables

1. Include all user-stated expectations and fixes captured during the collaborative run.
2. Execute with parallel subagents where lock scopes allow it.
3. Preserve gatekeeping boundaries (orchestrator vs worker, lease scope restrictions, actor attribution).
4. Use `just` recipes for all checks/tests.
5. Finalization order is locked:
   1. all code/test changes integrated,
   2. `just check`, `just ci`, `just test-golden` passing,
   3. create/update and execute a post-fix collaborative validation worksheet,
   4. commit implementation + test + validation record (record-preserving commit),
   5. update `README.md` + other docs/templates,
   6. clean up old/outdated markdown/worklog files,
   7. commit docs/cleanup pass.

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
4. After gates pass, run post-fix collaborative validation worksheet:
   - `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - record PASS/FAIL/BLOCKED per step with evidence paths.

## 6) Finalization Sequence (Locked by user request)

## Phase A: Record-preserving commit (after tests + post-fix worksheet)

1. Integrate implementation changes.
2. Run:
   - `just check`
   - `just ci`
   - `just test-golden`
3. Execute and complete:
   - `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
4. Commit implementation + test + validation record so there is a durable pre-cleanup snapshot.

## Phase B: Docs/MD cleanup pass (after record commit)

1. Update `README.md` and other affected docs/templates with final behavior and keymaps.
2. Clean up old/outdated markdown/worklog files that are superseded.
3. Validate references and doc integrity after cleanup.
4. Commit docs/cleanup changes as a separate follow-up commit.

## 7) Worklog

## Checkpoint 000 (2026-02-25)

1. Created this remediation plan/worklog per user request.
2. Imported complete expectation inventory from collaborative worksheet + C13 final findings.
3. Defined parallel lane setup and locked finalization order:
   - tests first (`just` gates),
   - then post-fix collaborative validation worksheet,
   - then record-preserving commit,
   - then docs update + outdated-file cleanup as a second commit.

Next checkpoint:
1. spawn worker lanes `L1-GATEKEEP`, `L2-LOGGING`, `L3-TUI-HOTSPOT` with lock scopes and acceptance criteria.

## Checkpoint 001 (2026-02-26)

1. User clarification received:
   - external padding must be small/equal for the whole TUI surface, not centered board/notices block behavior.
2. Research evidence captured before edit:
   - Context7: `/charmbracelet/lipgloss` placement/padding guidance (`PlaceHorizontal`, `Width`, `Padding` behavior).
   - Web docs search: Lip Gloss README/API references for `PlaceHorizontal`, block width, and padding/margin behavior.
3. Implemented layout adjustment:
   - added global constant `tuiOuterHorizontalPadding = 1` and applied it in both render branches (empty-project and normal board render) so header/board/notices/status/help share equal small outer gutters.
   - removed prior centering behavior for the board/notices block.
4. Verification commands and outcomes:
   - `just test-golden-update` (PASS)
   - `just fmt` (PASS)
   - `just check` (PASS)
   - `just ci` (PASS)
5. File touches:
   - `internal/tui/model.go`
   - `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
   - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`

## Checkpoint 002 (2026-02-26)

1. User-reported layout issues addressed:
   - `(1)` right-side outside gap larger than desired near notices panel.
   - `(2)` gap between `Done` and `Notices` appeared doubled vs inter-column gaps.
2. Root causes:
   - doubled separator came from both `MarginRight(1)` on columns and an explicit `" "` join spacer between body and notices panel.
   - panel-width allocation at narrower widths could leave visible right slack while panel remained visible.
3. Implementation updates:
   - removed explicit spacer in board+notices join.
   - switched layout width math to use inner content width (`m.width - 2*tuiOuterHorizontalPadding`).
   - adjusted notices-panel width selection to use available remainder after minimum board width with lower minimum width (`24`) to reduce right slack before hide-threshold behavior.
4. Test adjustments:
   - relaxed strict notices-panel string assertions in `internal/tui/model_test.go` where wrapping/truncation is expected at narrower panel widths.
5. Verification commands and outcomes:
   - `just fmt` (PASS)
   - `just test-golden-update` (PASS)
   - `just check` (PASS)
   - `just ci` (PASS)
6. File touches:
   - `internal/tui/model.go`
   - `internal/tui/model_test.go`
   - `internal/tui/testdata/TestModelGoldenBoardOutput.golden`
   - `internal/tui/testdata/TestModelGoldenHelpExpandedOutput.golden`

## 8) Live Remediation Execution Board (User-Locked Tracking Flow)

Tracking protocol (locked for this remediation wave):
1. Every remediation task keeps two independent checkboxes:
   - `Subagent complete`: worker marks `[x]` only after lane deliverables + lane evidence are ready.
   - `Orchestrator check`: orchestrator marks `[x]` only after code/test/doc review confirms completeness.
2. Worker updates must happen in this file under assigned task sections.
3. If orchestrator check fails, task returns to worker with explicit gap list.
4. If worker handoff is incomplete, spawn another worker lane for gap closure and keep an evidence trail here.

### 8.1 Backlog (Fix-All Scope)

| Task ID | Scope | Requirements | Lane | Status |
|---|---|---|---|---|
| T-001 | Notifications design checkpoint | REQ-004 | W-NOTIFY-DESIGN | pending |
| T-002 | Notifications panel implementation (always visible + two-part + quick-nav + drill-in) | REQ-002, REQ-003, user always-visible requirement | W-NOTIFY-UI | pending |
| T-003 | External MCP/HTTP auto-refresh in TUI | REQ-001 | W-REFRESH | pending |
| T-004 | Logging/help discoverability + sink parity | REQ-005, REQ-006, REQ-007, REQ-008, REQ-009, REQ-010 | W-LOGGING | pending |
| T-005 | Restore-task guard/transport contract | REQ-027 | W-GUARD-RESTORE | pending |
| T-006 | Archive/search/project-archived UX policy alignment | REQ-019, REQ-021, REQ-031 | W-ARCHIVE-UX | pending |
| T-007 | Agent approval notification flow | REQ-026 | W-APPROVAL-UX | pending |
| T-008 | Docs/template policy alignment | REQ-032, REQ-033, REQ-034, REQ-035, REQ-036 | W-DOCS-POLICY | pending |
| T-009 | Automated test expansion for all changed behaviors | Supports T-001..T-008 | W-TESTS | pending |
| T-010 | Full collaborative validation rerun + gates + evidence finalization | Sections C/D/E + `just check` + `just ci` + `just test-golden` | W-VALIDATION | pending |

### 8.2 Task Detail Cards

#### T-001 — Notifications Design Checkpoint (ASCII + Clarifying Questions)
- [x] Subagent complete
- [x] Orchestrator check
- Lane: `W-NOTIFY-DESIGN`
- Deliverables:
  1. ASCII layouts for desktop + narrow terminal behavior.
  2. Clarifying questions that must be answered before coding.
  3. Accepted UX contract for level scope, global count, quick-nav, and quick info modal behavior.
- Evidence:
  - planning note path(s): `pending`
- Implementation slices:
  1. Draft right-panel desktop ASCII with persistent visibility, header count area, feed list, and detail pane affordances.
  2. Draft narrow-terminal fallback ASCII with deterministic collapse/priority rules and preserved quick-nav actions.
  3. Record clarifying-question set and convert answers into an explicit UX contract checklist for implementation lanes.
  4. Add sign-off criteria mapping contract points to `T-002` implementation tasks.
- Acceptance checks:
  1. Desktop and narrow ASCII layouts both specify placement, focus behavior, and quick-nav interaction points.
  2. Clarifying questions explicitly cover level scope, global unresolved count, quick-nav labels, and quick info drill-in behavior.
  3. UX contract statements are testable (binary pass/fail wording) and traceable to `T-002` deliverables.
  4. `Orchestrator check` is marked only after direct orchestrator review of this handoff.
- Risks/open questions:
  1. Narrow-width behavior may conflict with help modal and existing overlay layering rules.
  2. Quick-nav key bindings may overlap with existing board or modal shortcuts.
  3. Need explicit decision on when global unresolved count updates if feed filtering is level-scoped.
- Planned test commands:
  - Execution now: `test_not_applicable` (planning/docs-only lane; no code changes).
  - Implementation stage: `just test-pkg ./internal/tui`
  - Implementation stage: `just test-pkg ./internal/app`
  - Integration stage: `just check`
  - Integration stage: `just ci`

#### T-002 — Notifications Panel Implementation (Always Visible + Two-Part)
- [ ] Subagent complete
- [ ] Orchestrator check
- Lane: `W-NOTIFY-UI`
- Deliverables:
  1. Always-visible right-side notifications region (no hidden-by-default behavior on normal widths).
  2. Two-part notifications workflow (feed + detail) with bottom quick-nav actions.
  3. Level-scoped items plus global unresolved count.
  4. Quick info modal drill-in from notification items.
- Evidence:
  - code/test path(s): `pending`
- Implementation slices:
  1. Introduce persistent notifications panel layout in normal widths without hidden-by-default toggles.
  2. Implement two-part workflow: feed list region plus selected-item detail region with stable focus transitions.
  3. Wire bottom quick-nav actions for notification traversal and drill-in launch to quick info modal.
  4. Connect level-scoped feed filtering and global unresolved count aggregation to existing state sources.
  5. Add/update TUI tests for rendering, keyboard navigation, focus handoff, and modal drill-in behavior.
- Acceptance checks:
  1. Panel remains visible at supported normal widths and does not require a toggle to appear.
  2. Feed and detail regions render concurrently with deterministic selection behavior.
  3. Quick-nav actions function with vim keys and arrow-key flows without regressing existing controls.
  4. Level-scoped feed and global unresolved count are both visible and update correctly on state changes.
  5. `Orchestrator check` remains unchecked pending review.
- Risks/open questions:
  1. Layout pressure on smaller terminals may degrade board readability without finalized breakpoint rules from `T-001`.
  2. Selection-state synchronization between feed/detail/modal may introduce stale-detail edge cases.
  3. Existing mouse interaction paths may require additional handling to preserve click/scroll behavior.
- Planned test commands:
  - Execution now: `test_not_applicable` (planning/docs-only lane; no code changes).
  - Implementation stage: `just test-pkg ./internal/tui`
  - Implementation stage: `just test-pkg ./internal/domain`
  - Integration stage: `just check`
  - Integration stage: `just ci`

#### T-003 — External MCP/HTTP Auto-Refresh
- [x] Subagent complete
- [x] Orchestrator check
- Lane: `W-REFRESH`
- Deliverables:
  1. Out-of-band MCP/HTTP mutations appear in current TUI view without project switch workaround.
  2. Refresh policy documented (event-driven and/or interval fallback).
  3. Tests proving refresh in focused and non-focused scopes.
- Evidence:
  - code/test path(s): `pending`
- Implementation slices:
  1. Define refresh trigger path for MCP/HTTP mutations to invalidate and rehydrate active view state.
  2. Add interval fallback refresh loop with bounded cadence and no-focus starvation prevention.
  3. Ensure refresh applies to current scope and parent rollups without requiring project navigation changes.
  4. Add focused/non-focused scope tests validating eventual consistency after external mutations.
  5. Document refresh policy tradeoffs and operational guardrails in implementation notes.
- Acceptance checks:
  1. External mutation events appear in active TUI scope without manual project switch.
  2. Fallback interval refresh updates stale views when direct event signaling is unavailable.
  3. Focused and non-focused refresh behavior is covered by automated tests with deterministic assertions.
  4. Refresh behavior does not regress local in-session mutation visibility.
  5. `Orchestrator check` is marked only after direct orchestrator review of this handoff.
- Risks/open questions:
  1. Event-driven refresh may miss mutations if adapter signaling is lossy or delayed.
  2. Overly aggressive interval cadence could increase render churn in large boards.
  3. Need clear precedence rules when local optimistic state and external updates race.
- Planned test commands:
  - Execution now: `test_not_applicable` (planning/docs-only lane; no code changes).
  - Implementation stage: `just test-pkg ./internal/tui`
  - Implementation stage: `just test-pkg ./internal/app`
  - Implementation stage: `just test-pkg ./internal/adapters/storage/sqlite`
  - Integration stage: `just check`
  - Integration stage: `just ci`

#### T-004 — Logging/Help/Sink Parity
- [ ] Subagent complete
- [ ] Orchestrator check
- Lane: `W-LOGGING`
- Deliverables:
  1. Clear debug activation path (`--help` discoverability and documented command path).
  2. Runtime + MCP guardrail warnings/errors mirrored to file sink and stdout/stderr as required.
  3. No silent failure path for critical mutation errors.
- Evidence:
  - code/test path(s): `pending`

#### T-005 — MCP Restore Guard Contract
- [ ] Subagent complete
- [ ] Orchestrator check
- Lane: `W-GUARD-RESTORE`
- Deliverables:
  1. `kan_restore_task` request path enforces/provides required actor+lease tuple.
  2. Guardrail behavior aligns with actor model (user vs agent).
  3. Automated test coverage for this contract.
- Evidence:
  - code/test path(s): `pending`

#### T-006 — Archive/Search/Project Archived UX
- [ ] Subagent complete
- [ ] Orchestrator check
- Lane: `W-ARCHIVE-UX`
- Deliverables:
  1. Archive action policy aligned with quick actions + command palette expectations.
  2. Project-level archived visibility UX reworked and documented.
  3. Marker meanings remain clear and validated in help/legend.
- Evidence:
  - code/test path(s): `pending`

#### T-007 — Agent Approval Notification Flow
- [ ] Subagent complete
- [ ] Orchestrator check
- Lane: `W-APPROVAL-UX`
- Deliverables:
  1. Approval request notifications for agent identity/scope grants.
  2. Path for user approval or orchestrator continuation when user authorizes from client flow.
  3. Guardrail-compatible audit trail of approvals.
- Evidence:
  - code/test path(s): `pending`

#### T-008 — Docs/Template Policy Alignment
- [ ] Subagent complete
- [ ] Orchestrator check
- Lane: `W-DOCS-POLICY`
- Deliverables:
  1. `AGENTS.md`/template guidance covers DB-first update behavior and approval/scope expectations.
  2. Single-source-of-truth guidance updated in README/docs.
  3. Explicit policy for adaptable per-project fields (bounded flexibility).
- Evidence:
  - docs path(s): `pending`

#### T-009 — Automated Test Expansion
- [ ] Subagent complete
- [ ] Orchestrator check
- Lane: `W-TESTS`
- Deliverables:
  1. Tests added/updated for notifications, refresh, logging/help, guard restore contract, and archive UX.
  2. Lane-level `just test-pkg` evidence and final integrator gates ready.
- Evidence:
  - test path(s): `pending`

#### T-010 — Final Collaborative Validation + Gates
- [ ] Subagent complete
- [ ] Orchestrator check
- Lane: `W-VALIDATION`
- Deliverables:
  1. Collaborative worksheet rerun completed with PASS/FAIL/BLOCKED per step + evidence.
  2. Final quality gates: `just check`, `just ci`, `just test-golden`.
  3. Final verdict, blocker list, and follow-up actions captured.
- Evidence:
  - worksheet/evidence path(s): `pending`

## 9) Subagent Plan-Wave 0 (Detailed Plan Drafting Before Code)

Purpose:
1. Build implementation-ready detail for T-001..T-010.
2. Keep this file as the single planning tracker with subagent and orchestrator checkboxes.
3. Ensure no requirement from the recovered worksheet and post-fix worksheet is lost.

Plan-wave lanes:
1. `P0-A` (notifications + refresh): T-001, T-002, T-003.
2. `P0-B` (logging + guard): T-004, T-005.
3. `P0-C` (archive/approval/docs/tests/validation): T-006..T-010.

Subagent update rule for Plan-Wave 0:
1. Assigned subagent appends one short handoff block under each owned task card:
   - proposed implementation slices,
   - acceptance checks,
   - risk/open questions,
   - recommended test commands.
2. Subagent then marks only `Subagent complete` for planning handoff if the handoff is complete.
3. Orchestrator reviews and marks `Orchestrator check` after quality/completeness verification.
