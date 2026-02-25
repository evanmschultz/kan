# Collaborative Full E2E Test Worksheet (User + Agent)

Date: 2026-02-25
Tester agent/session: Codex collaborative test agent
User: evanschultz
Run artifact dir: `.tmp/collab-e2e-20260225_080750/`

## 0. Locked Run Instructions + Full Conversation Capture (Do Not Lose)

This section is the complete retention block for user requirements, expectations, plan details, and observations provided during this collaborative run.

### 0.1 Original hard requirements (locked by user)

1. Do not edit code.
2. Use a fresh temp DB for this run.
3. Save raw evidence/logs under `.tmp/collab-e2e-<timestamp>/`.
4. For each worksheet step, mark `PASS` / `FAIL` / `BLOCKED` with evidence path.
5. If a step fails, capture exact repro details and continue unless blocked.
6. End with final verdict + blocker list + follow-up actions.

### 0.2 Locked execution flow (from user)

1. Read worksheet fully.
2. Start server per preconditions with fresh DB.
3. Verify health endpoint.
4. Walk sections A -> E without skipping.
5. Explicitly include/verify Section C checks:
   - C8: focus + first child creation on empty leaf scopes.
   - C12: `new-phase` and `new-subphase` from selected and focused-empty scopes.
   - C12: while subtree focus active, `new-branch` blocked with warning modal.
   - C13: branch/project create-edit-archive-restore-delete and archived visibility/search behavior.
6. Run final gates:
   - `just check`
   - `just ci`
   - `just test-golden`
7. Return structured final report.

### 0.3 Collaboration style requirements (from user)

1. Ask user to perform manual TUI actions when needed.
2. Keep instructions short and specific.
3. Confirm evidence after each major step before moving on.
4. Keep all observations in this same worksheet in the corresponding sections (no loss of detail).

### 0.4 Critical user observations captured from beginning of run

1. Startup modal/input issue observed immediately: `v` could not be typed in text input; expected all typing keys to work in input fields while still supporting selection/copy mode in modals.
2. Setup-created project state was intentional; do not treat this as invalidating collaboration progress for this run.
3. Section B completion means hierarchy seeding criteria passed only; it does not imply product UX/issues are resolved.
4. MCP-originated updates were not shown live in current project view; user observed refresh only after switching projects away/back.
5. User requested robust MCP/runtime logging visibility and meaningful warning/error surfacing to TUI notices/notifications.
6. User requested notices/notifications panel behavior:
   - level-scoped context,
   - global count,
   - quick navigation among notifications at panel bottom,
   - quick info modals for important warnings/errors.
7. User reported branch creation flow missing expected path parameter behavior.
8. User reported due picker issues:
   - `space` to include time not working,
   - desired fuzzy date input/filter behavior (example `2-2` => Feb 2 and Feb 20-range matches).
9. User asked meaning of `!3` in card metadata; clarified as unresolved attention count.
10. Navigation expectation:
   - `esc` should behave like back-step (return to prior focused/modal-open context),
   - should not unwind all the way to project root unless explicitly requested.
11. Help expectation:
   - copy/paste/select shortcuts must appear in help,
   - `?` guidance should be available in all modal contexts.
12. Keybinding expectation for selection mode:
   - `v` is a poor fit due to typing/paste conflicts,
   - suggested direction: control chord for select mode (example `Ctrl+Y`) and `Y`/`Shift+Y` copy/yank semantics.
13. Icon/emoji expectation:
   - emoji input should work across text inputs,
   - icon feature needs explicit product purpose/behavior plan (what it does and why).
14. Design/process requirement for notifications-panel fix:
   - first produce ASCII-art design proposal,
   - review context and ask clarifying questions before implementation.

### 0.5 Command/approval clarity expectation

1. When running commands that require escalation or non-obvious shell flags, explain what the command does and why it is needed for the worksheet step.
2. Clarification for shell safety prefix used in this run:
   - `set -euo pipefail` means:
     - `-e`: exit on command error,
     - `-u`: error on unset variables,
     - `pipefail`: fail pipeline if any segment fails.

### 0.6 Identity/Gatekeeping expectations (added during run)

1. Orchestrator should not mutate as `actor_type=user`; mutations by the testing/orchestration agent should be represented as `actor_type=agent` with valid lease tuple.
2. User actions should be tracked as real user identity (expected from bootstrap persona), and agent actions should be separately attributed.
3. Agent must not be able to present/label mutations as user actions.
4. Orchestrator/worker delineation must remain explicit throughout testing and implementation follow-up.
5. Subagent note for this run:
   - real spawned worker subagent was used for gatekeeping validation;
   - spawned worker id: `019c9654-fbd2-7e30-8c60-160ec0cbddc1`;
   - simulated identity-lane probes were also run using distinct lease tuples (`agent_name` + `agent_instance_id`).
6. Gatekeeping matrix evidence:
   - `.tmp/collab-e2e-20260225_080750/identity_gatekeeping_matrix.md`
7. Current orchestrator identity pair used for gated checks:
   - `agent_name=orchestrator-main`
   - `agent_instance_id=orchestrator-main-1`
   - evidence: `.tmp/collab-e2e-20260225_080750/spawn_agent_gatekeeping_matrix.md`

### 0.7 New user-locked gatekeeping + approval-flow requirements

1. Validate gatekeeping with real `spawn_agent` worker usage, not only simulated identity tuples.
2. Confirm orchestrator identity pair explicitly and prove orchestrator can be constrained to `e2e-main`.
3. Confirm workers/subagents can be constrained by scope and cannot mutate outside allowed project/scope.
4. Confirm orchestrator cannot mutate outside allowed project/scope when operating as agent with lease enforcement.
5. Validate name/id generation and attribution behavior for user vs orchestrator vs worker paths.
6. Expected UX/approval flow for agent identity + scope restrictions:
   - TUI should surface notification when agent requests permission to create/use name-id pair and scope restrictions.
   - User should be able to approve from client side without requiring explicit TUI-only approval workflow for each step once user consent is granted.
   - Orchestrator should be able to continue approvals according to that granted consent policy.
7. Template requirement:
   - this permission/scope path flow should exist in `AGENTS.md` / `CLAUDE.md` template logic used for auto-generation/update of markdown guidance;
   - subagent must verify whether current templates already encode this or if it is missing.
8. Spawn-agent validation requirement:
   - real spawned worker must validate same scope controls and attribution behavior;
   - evidence must include spawned worker agent id and raw pass/fail outcomes.
9. Template-check subagent result:
   - `AGENTS.md` has partial approval flow but does not fully encode agent identity + scoped approval payload + notification UX expectations.
   - `CLAUDE.md` template file is currently absent.
   - evidence: `.tmp/collab-e2e-20260225_080750/spawn_agent_gatekeeping_matrix.md`.

## 1. Objective

Run a full end-to-end validation of TUI + HTTP + MCP on a fresh database, with explicit guardrail and parity checks.

Pass criteria:
1. TUI hierarchy rendering and drill-down behavior match product intent.
2. MCP/HTTP core flows pass on fresh data.
3. Guardrail failures are fail-closed with expected errors.
4. Final gates pass (`just check`, `just ci`, `just test-golden`).

## 2. Runtime Preconditions (Mandatory)

Complete these before any test calls:

1. Start from repo root:
   - `/Users/evanschultz/Documents/Code/personal/kan`
2. Confirm `.codex/config.toml` MCP endpoint:
   - `mcp_servers.kan-local.url = "http://127.0.0.1:18080/mcp"`
3. Use a fresh temp DB for this run:

```bash
TMP_DIR=$(mktemp -d /tmp/kan-collab-e2e.XXXXXX)
DB_PATH="$TMP_DIR/kan-collab.db"
ART_DIR=".tmp/collab-e2e-$(date +%Y%m%d_%H%M%S)"
mkdir -p "$ART_DIR"
./kan serve --db "$DB_PATH" --http 127.0.0.1:18080 --api-endpoint /api/v1 --mcp-endpoint /mcp
```

4. In a second terminal, verify health:

```bash
curl -fsS http://127.0.0.1:18080/healthz && echo
```

5. Record runtime values:
   - server command: ______________________________________
   - HTTP bind: ___________________________________________
   - API endpoint: ________________________________________
   - MCP endpoint: ________________________________________

## 3. Collaboration Protocol

1. Work section-by-section; do not skip ahead.
2. For any command approval prompt, approve only if command matches the section step.
3. Save every raw response/log in `$ART_DIR`.
4. Mark each step `PASS` / `FAIL` / `BLOCKED` with evidence path.

## 4. Section A: Fresh-Instance Bootstrap Checks

### A1. MCP list projects is empty on fresh DB
- Expected: `projects | length == 0`
- Result: PASS (user-approved setup-seeded state accepted for this run)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_a_user_approved_state.md`

### A2. MCP bootstrap guide is available
- Tool: `kan.get_bootstrap_guide`
- Expected: deterministic setup guidance returned, not internal error
- Result: PASS
- Evidence: `.tmp/collab-e2e-20260225_080750/section_a_preflight_attempt1.md`

### A3. `capture_state` on empty instance returns bootstrap-required guidance
- Tool: `kan.capture_state`
- Expected: bootstrap-required behavior, actionable next steps
- Result: PASS (deterministic non-empty path observed and accepted for continued run)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_a_preflight_attempt1.md`

## 5. Section B: Seed Minimal Hierarchy for TUI + Guardrails

Create one project with one full hierarchy chain and one sibling top-level task.

Required structure:
1. Project: `e2e-main`
2. Branch: `Seed Branch`
3. Phase child of branch: `Seed Phase`
4. Subphase child of phase: `Seed Subphase`
5. Task child of subphase: `Seed Task`
6. Top-level task (no parent): `Top Task`

Record IDs:
- project_id: `0316b1a8-0b63-43a6-8435-001a49e2cd78` (`e2e-main`)
- branch_id: `3c83f031-dcdf-4350-a48f-cf83edac6787` (`Seed Branch`)
- phase_id: `e04bd945-f672-43a3-9d6c-b60d13c3ea50` (`Seed Phase`)
- subphase_id: `21e2e550-d8ab-4b88-adca-cb80820eb549` (`Seed Subphase`)
- task_id: `efa725a2-500d-40e9-aafe-ac5002160106` (`Seed Task`)
- top_task_id: `276ba2b0-728e-4ad1-8ce0-a21ddd1f981d` (`Top Task`)

Evidence: `.tmp/collab-e2e-20260225_080750/section_b_seed_evidence.md` (PASS for hierarchy seeding only; does not imply unresolved UX/behavior items are resolved)

## 6. Section C: TUI Behavior Validation (Manual + Agent Guided)

Launch TUI against the same DB:

```bash
./kan --db "$DB_PATH"
```

### C1. Project scope renders immediate children only
- In project board, verify nested descendants are not flattened into the same scope.
- Expected visible at project scope: top-level branch(es) + top-level task(s).
- Result: PASS (scope rendering behavior validated; refresh issue tracked separately in Section C notes)
- Evidence (screenshot/log): `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round3.md`
- Locked follow-up expectation: MCP-originated internal changes should appear without requiring project-switch refresh workaround.

### C2. Path line is visible above board
- Expected: `path: <project>` at project scope.
- Result: PASS (`path: e2e-main` visible at project scope; level/path working as expected)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c2_manual_result.md`

### C3. Focus drill-down (`f`) moves one level at a time
- On branch: `f` -> board shows immediate branch children.
- On phase: `f` -> board shows immediate phase children.
- On subphase: `f` -> board shows immediate subphase children.
- `F` returns to broader scope/full board.
- Result: PASS (focus drill-down works; back-navigation expectation tracked in C4/C8 notes)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round3.md`
- Locked follow-up expectation: `esc` should step back contextually one level/state (browser-back semantics), not jump to root.

### C4. Enter/info behavior on hierarchy items
- Press `enter` (or `i`) on branch/phase/subphase.
- Expected: info modal opens with metadata and supports `f` for scope drill-down.
- Result: FAIL (info modal opens and drill-down mostly works, but `esc` back behavior should return to modal-open origin state instead of unwinding to parent/project level)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round3.md`
- Locked follow-up expectation: info modal must remember where it was opened and back out only to that exact prior state.

### C5. Hierarchy markers render in card metadata
- Expected examples: `[branch|...]`, `[phase|...]`.
- Result: PASS (markers rendered; clarified `!3` as unresolved attention count and `!MM-DD` as overdue due marker)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round2.md`

### C6. Right-side notices panel appears on wide layout
- Expected panel shows:
  - attention summary,
  - selected-item context,
  - recent activity hint.
- Result: FAIL (notifications/notices panel not visible during validation; expected warning/error surfacing absent)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round2.md`, `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round3.md`, `.tmp/collab-e2e-20260225_080750/section_c_log_observation.md`
- Locked follow-up expectations:
  - level-scoped notifications panel with global count,
  - quick navigation controls at panel bottom,
  - warnings/errors surfaced from important runtime/MCP operations,
  - quick info modal behavior for warning/notification drill-in,
  - implementation must start with ASCII-art proposal + clarifying questions before coding.

### C7. Scoped create (`n`) follows focused level
- From focused branch/phase/subphase boards, press `n` and create one item each.
- Expected: each new item is created as a child of that focused scope (not project root).
- From focused task board, press `n`.
- Expected: new item is created as a direct subtask of that task.
- Result: PASS
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round3.md`

### C8. Focus works on empty leaf scopes
- On a task/subtask with no children, press `f`.
- Expected:
  - focused scope activates (path/focus banner visible),
  - board may be empty if no children yet,
  - `n` from this focused scope creates the first child in that scope.
- Result: PASS (deep chain validated through sub-subtask creation under user flow)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round3.md`
- Confirmed chain: `User-Branch -> user phase -> user task -> subtask -> new sub subtask`.
- Locked follow-up expectation: back behavior with focus should be contextual; current `Shift+F` project-level reset note retained.

### C9. Text-selection mode works on every screen/modal
- Toggle selection mode with `v` from board view.
- Open each modal/screen below and verify terminal copy-selection works while selection mode is enabled:
  - project picker (`p`),
  - new/edit task (`n` / `e`),
  - new/edit project (`N` / `M`),
  - due picker (`d` from task form),
  - label picker (labels field `enter`),
  - dependency inspector (`b` from task info),
  - command palette (`:`),
  - thread view (`c` from task info).
- Expected:
  - selection mode remains enabled across transitions,
  - text can be selected/copied in each screen/modal,
  - `v` toggles mode off from modal contexts as well.
- Result: FAIL (`v` conflicts with modal text entry; due picker time-toggle behavior did not work as expected; keybinding/hotkey discoverability gaps)
- Evidence: `.tmp/collab-e2e-20260225_080750/c9_text_selection_user_observation.md`, `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round2.md`
- Locked follow-up expectations:
  - all typing keys must continue to work inside text inputs (including modal/startup inputs),
  - copy/select mode should be available without stealing text-input runes,
  - due picker: `space` include-time behavior must work,
  - due picker: text should fuzzy-filter dates (example `2-2`),
  - keybinding redesign requested: use control chord for selection mode (example `Ctrl+Y`) and reserve `Y`/`Shift+Y` copy/yank semantics.

### C10. Project icon behavior + emoji support
- Create or edit a project with icon value set to an emoji (example: `🚀`).
- Expected:
  - icon is accepted and persisted,
  - icon is visible in board header, project tabs, and project picker rows.
- Result: FAIL (emoji/input support and icon functional semantics need explicit behavior definition; current icon value perceived as non-functional)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round3.md`
- Locked follow-up expectations:
  - emoji input support across text input fields,
  - explicit icon product plan required (purpose, placement, and user value) before implementation.

### C11. Screen-specific `?` help overlay
- In each major mode/modal, press `?` and verify help content is scoped to that active screen only.
- Verify at minimum:
  - board mode,
  - task form,
  - project form,
  - due picker,
  - dependency inspector,
  - command palette,
  - thread mode.
- Expected:
  - help title identifies active screen context,
  - key guidance is specific to that screen (no unrelated workflows from other modes),
  - `esc` closes help first without closing the underlying modal.
- Result: FAIL (help discoverability gaps reported for modal workflows and selection/copy shortcuts)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round2.md`
- Locked follow-up expectations:
  - `?` help affordance needs clear availability in all modals/screens,
  - help content must include copy/paste/select shortcuts and modal-specific guidance.

### C12. Phase/Subphase creation tools
- Open command palette (`:`) and run `new-phase` from:
  - a selected branch row,
  - an empty focused branch scope (`f` on branch with no children).
- Expected:
  - task form opens with phase defaults,
  - parent is that branch,
  - created item is phase-scoped.
- Run `new-subphase` from:
  - a selected phase/subphase row,
  - an empty focused phase scope (`f` on phase with no children).
- Expected:
  - task form opens with subphase defaults,
  - parent is that phase/subphase,
  - created item is subphase-scoped.
- While focused in any subtree (`f` active), run `new-branch`.
- Expected:
  - creation is blocked,
  - warning modal appears with guidance to clear focus first,
- no project-level branch is created accidentally.
- Result: FAIL (branch creation UX gap reported: missing path parameter; focused-empty/full command-matrix evidence not yet complete)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round2.md`
- Locked follow-up expectations:
  - branch creation flow must satisfy expected path parameter behavior,
  - retain requirement: `new-phase`/`new-subphase` work from selected and focused-empty scopes,
  - retain requirement: `new-branch` blocked during active subtree focus with warning modal.

### C13. Branch/Project lifecycle + archived visibility
- Use command palette (`:`) for branch lifecycle:
  - `new-branch`, `edit-branch`, `archive-branch`, `restore-branch`, `delete-branch`.
- Use command palette (`:`) for project lifecycle:
  - `new-project`, `edit-project`, `archive-project`, `restore-project`, `delete-project`.
- Expected:
  - archive/restore/delete actions persist after reload,
  - archived items are hidden when archived visibility is off,
  - archived items are visible when archived visibility is enabled (`t`),
  - search supports archived state filtering and returns archived nodes when requested.
- Result: FAIL (branch and project lifecycle actions mostly passed, but archived visibility/search semantics and archive key UX do not meet expected behavior)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_c_user_findings_round2.md`, `.tmp/collab-e2e-20260225_080750/section_c13_interim_observation.md`, `.tmp/collab-e2e-20260225_080750/section_c13_final_user_results.md`
- Interim finding captured during C13 execution:
  - archive-branch confirm modal currently labels entity as `task` (`archive task: <branch-name>`), expected `branch`.
  - restore flow requires archived visibility enabled (`t`) and `restore-branch` action on selected archived branch.
  - required UX addition: `.` quick action to restore archived items directly.
- Additional user-validated C13 findings:
  - archived projects currently remain visible in project picker/top-level project context when expected hidden-by-default.
  - project picker needs explicit archived show/hide control.
  - project-level archived visibility UX tied to `t` is not a good fit in current navigation model.
  - archive action should not be on `a`; expected archive/restore pathways are `.` quick action + `:` command palette.
  - search archived filter should return archived matches when selected, without requiring separate global `t` visibility toggle coupling.
  - all other C13 checks (including optional `reset-filters`) passed.

## 7. Section D: HTTP/MCP Parity + Guardrails

### D1. `capture_state` parity hash MCP vs HTTP
- Expected: same `state_hash` for same scope and unchanged state.
- Result: PASS (MCP and HTTP returned matching `state_hash` on same scope/state)
- Evidence: `.tmp/collab-e2e-20260225_080750/d1_http_capture_state_final.json`, `.tmp/collab-e2e-20260225_080750/section_d_partial_evidence.md`

### D2. Unknown scope revoke-all fails closed
- Tool: `kan.revoke_all_capability_leases` with unknown scope tuple.
- Expected: `not_found` (or equivalent fail-closed error).
- Result: PASS (`not_found` returned)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_d_partial_evidence.md`

### D3. Unknown comment target fails closed
- Tool: `kan.create_comment` with non-existent target.
- Expected: `not_found` (or equivalent fail-closed error).
- Result: PASS (`not_found` returned)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_d_partial_evidence.md`

### D4. `update_task` title-only preserves priority
- Expected: priority unchanged when omitted.
- Result: PASS (title-only update preserved `Priority=medium`)
- Evidence: `.tmp/collab-e2e-20260225_080750/section_d_partial_evidence.md`

## 8. Section E: MCP Tool Sweep (All Tools)

Mark each tool `PASS` when minimally validated with real call evidence.

### E1. Bootstrap/Capture/Attention
- [x] `kan.get_bootstrap_guide`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.capture_state`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.list_attention_items`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.raise_attention_item`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.resolve_attention_item`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)

### E2. Projects
- [x] `kan.list_projects`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.create_project`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.update_project`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)

### E3. Tasks/Hierarchy/Search
- [x] `kan.list_tasks`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.create_task`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.update_task`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.move_task`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.delete_task`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.restore_task`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.reparent_task`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.list_child_tasks`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.search_task_matches`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)

### E4. Change/Dependency/Kinds
- [x] `kan.list_project_change_events`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.get_project_dependency_rollup`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.list_kind_definitions`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.upsert_kind_definition`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.set_project_allowed_kinds`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.list_project_allowed_kinds`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)

### E5. Capability Leases
- [x] `kan.issue_capability_lease`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.heartbeat_capability_lease`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.renew_capability_lease`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.revoke_capability_lease`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.revoke_all_capability_leases`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- Critical gatekeeping finding:
  - agent tuple + valid lease can still submit `actor_type=user` and persist `UpdatedByType=user` with agent identity in actor fields.
  - see `.tmp/collab-e2e-20260225_080750/identity_gatekeeping_matrix.md`.
 - Spawn-agent confirmation:
  - spawned worker id `019c9654-fbd2-7e30-8c60-160ec0cbddc1` reproduced cross-project blocking and user-spoof acceptance.
  - see `.tmp/collab-e2e-20260225_080750/spawn_agent_gatekeeping_matrix.md`.

### E6. Comments
- [x] `kan.create_comment`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)
- [x] `kan.list_comments_by_target`  Evidence: `.tmp/collab-e2e-20260225_080750/section_e_tool_sweep_evidence.md` (PASS)

## 9. Section F: Final Quality Gates

Run and record:

```bash
just check
just ci
just test-golden
```

- `just check`: PASS  Evidence: `.tmp/collab-e2e-20260225_080750/section_f_just_check.log`, `.tmp/collab-e2e-20260225_080750/section_f_quality_gates_evidence.md`
- `just ci`: PASS  Evidence: `.tmp/collab-e2e-20260225_080750/section_f_just_ci.log`, `.tmp/collab-e2e-20260225_080750/section_f_quality_gates_evidence.md`
- `just test-golden`: PASS  Evidence: `.tmp/collab-e2e-20260225_080750/section_f_just_test_golden.log`, `.tmp/collab-e2e-20260225_080750/section_f_quality_gates_evidence.md`

## 10. Final Verdict

- Overall: FAIL
- Blocking issues:
  1. Agent/user attribution boundary gap: valid agent lease tuple can still submit `actor_type=user` and persist user-typed mutations (see Section E5 gatekeeping evidence).
  2. C13 archived lifecycle/search UX mismatch: archived projects visibility behavior and archived-search filter coupling require correction before expected project/archive workflows are reliable.
- Follow-up actions:
  1. Fix actor-type enforcement and approval/notification UX for agent identity + scope requests; ensure agents cannot impersonate user actor type.
  2. Resolve C-section UX failures (C4/C6/C9/C10/C11/C12/C13), prioritizing archived visibility/search semantics, archive/restore quick actions (`.` + command palette), and notices/help discoverability.
