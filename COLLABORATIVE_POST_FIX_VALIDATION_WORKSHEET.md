# Collaborative Post-Fix Validation Worksheet

Date: 2026-02-25  
Tester agent/session: Codex collaborative test agent  
User: evanschultz  
Artifact dir: `.tmp/collab-post-fix-20260225_143243/`

This is the active worksheet for post-fix validation.  
`COLLABORATIVE_FULL_E2E_TEST_WORKSHEET.md` is historical context only; do not record new run results there.
`TUI_MANUAL_TEST_WORKSHEET.md` was retired on 2026-02-27; unresolved TUI carry-forward items are now tracked in Section 11 of this worksheet.
Use `MCP_FULL_TESTER_AGENT_RUNBOOK.md` as the canonical MCP full-sweep procedure and evidence contract.

## 1) Objective

Validate that all remediations from `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md` are complete and that previously failing collaborative E2E findings are resolved.

Rule: mark every step `PASS` / `FAIL` / `BLOCKED` with evidence path.

Run ledger policy (locked for this validation pass):
- This worksheet is the single narrative source of truth for test status and findings.
- `.tmp/collab-post-fix-20260225_143243/` stores raw command/log artifacts only.
- External summary markdown files under `.tmp` are non-authoritative and should not be used as the run narrative.

## 2) Preconditions

1. Implementation changes merged locally.
2. Full gates already passed:
   - `just check`
   - `just ci`
   - `just test-golden`
3. Server/TUI launched against clean validation DB snapshot.
4. Debug logging can be enabled without manual TOML edits (CLI flag and/or env override), or this remains an explicit FAIL item.
5. Logging discoverability check captured from help output:
   - `./kan --help`
   - `./kan serve --help`

Record command/evidence:
- gates evidence: `.tmp/collab-post-fix-20260225_143243/just_check.txt`, `.tmp/collab-post-fix-20260225_143243/just_ci.txt`, `.tmp/collab-post-fix-20260225_143243/just_test_golden.txt` (PASS)
- runtime command: user-launched `./kan serve --http 127.0.0.1:18080 --api-endpoint /api/v1 --mcp-endpoint /mcp` (listener evidence: `.tmp/collab-post-fix-20260225_143243/port_18080_listener.txt`)
- health check evidence: `.tmp/collab-post-fix-20260225_143243/healthz.txt`, `.tmp/collab-post-fix-20260225_143243/readyz.txt` (both `{"status":"ok"}`)
- help output evidence: `.tmp/collab-post-fix-20260225_143243/help_kan.txt`, `.tmp/collab-post-fix-20260225_143243/help_kan_serve.txt`
- initial-empty-project evidence: `.tmp/collab-post-fix-20260225_143243/precondition_initial_projects_empty.txt`

Precondition status:
1. Implementation merge present: PASS (`git log -1 => 368b908`).
2. Full gates passed: PASS.
3. Clean validation DB snapshot: PASS (inferred from initial `kan_list_projects => []` before automated setup mutations).
4. Debug logging enablement without manual TOML edits: FAIL (manual config edit required).
5. Help-output capture: PASS (captured, but content quality fails in Section 6).

## 3) Gatekeeping + Identity Regression

1. Agent lease tuple cannot submit `actor_type=user`.
2. Cross-project/scope mutation remains fail-closed.
3. User and agent attribution fields are distinct and correct.
4. Spawned worker scope enforcement still holds.

- Result: PASS
- Evidence: `.tmp/collab-post-fix-20260225_143243/section3_attribution_extract.txt`, `.tmp/collab-post-fix-20260225_143243/precondition_initial_projects_empty.txt`, MCP call transcript in this run
- Detailed outcomes:
  - actor spoof guard:
    - attempted guarded mutation with `actor_type=user` + lease tuple
    - response: `invalid_request: actor_type=user cannot be used with guarded mutation tuple`
    - status: PASS
  - cross-project guard:
    - lease scoped to project `7b742526-30d6-4e5a-b483-9b64432fe765`
    - in-scope update succeeded on that project
    - cross-project update on `a4dd0ea4-93a0-4529-a8ba-2bef09d0f95f` failed with `guardrail_failed ... mutation lease is invalid`
    - status: PASS
  - attribution separation:
    - user-created task: `CreatedByActor=kan-user`, `UpdatedByType=user`
    - agent-created task: `CreatedByActor=orchestrator-main`, `UpdatedByType=agent`
    - status: PASS
  - real worker scope probe:
    - worker agent id: `019c96cc-ee7b-7ad1-8cde-ea0a49dd52f4`
    - worker cross-project update attempt failed closed with `guardrail_failed ... mutation lease is invalid`
    - status: PASS

## 4) TUI Regression Sweep (Previously Failing Areas)

1. C4: contextual `esc` back behavior + info modal return origin.
2. C6: notifications panel present with level scope + global count + quick-nav + warning/error surfacing.
3. C9: typing keys preserved in text inputs; selection/copy key redesign works; due picker include-time + fuzzy date works.
4. C10: emoji input support + icon behavior is defined and visible.
5. C11: `?` help available in all modals with correct scoped shortcuts.
6. C12: branch path flow, focused-empty phase/subphase creation behavior, `new-branch` block under focus.
7. C13: archive/restore/project lifecycle semantics and search archived behavior (decoupled from global visibility toggle as expected).

- Result: BLOCKED
- Evidence: `.tmp/collab-post-fix-20260225_143243/user_manual_notes.txt`, automated-only context from `.tmp/collab-post-fix-20260225_143243/just_check.txt`
- Notes:
  - Manual TUI verification required for C4/C6/C9/C10/C11/C12/C13.
  - User note captured: bootstrap completed with display name `Evan`; user observed typing `v` worked in bootstrap input.
  - Root-cause code review note for missing notifications panel UX:
    - current UI renders a single right-side `Notices` panel only when width threshold passes; there is no dedicated two-part notifications implementation with global count + quick-nav workflow.
    - width gate is hard-coded in `noticesPanelWidth` and can hide the panel on narrower terminals.
  - Root-cause code review note for stale board state after external mutations:
    - no periodic/event-driven refresh loop exists for external MCP/HTTP mutations.
    - `loadData` is triggered from local UI actions (project switch, explicit actions), so out-of-band writes can remain stale until a user action forces reload.

## 5) Archived/Search/Keybinding Targeted Checks

1. Archived projects hidden by default where expected; explicit picker visibility control present.
2. `.` quick action restore works.
3. Archive action key policy matches expected behavior (no plain `a` archive trigger).
4. Search archived filter returns archived matches when selected without requiring unrelated global toggle coupling.

- Result: BLOCKED
- Evidence: `.tmp/collab-post-fix-20260225_143243/just_check.txt`
- Notes:
  - Requires manual TUI behavior verification (project picker visibility, quick-action restore UX, archive key policy, archived-search semantics in UI).

## 6) Logging + Notifications Quality

1. MCP/runtime operations produce useful logs with `charmbracelet/log`.
2. Important warnings/errors bubble to notifications panel and quick info modal.
3. No silent failures for key operations.
4. Debug logging activation path is ergonomic for test runs (no mandatory manual config-file edit).
5. Help output clearly documents debug logging activation path (`--log-level` and/or env/config route).

- Result: FAIL
- Evidence: `.tmp/collab-post-fix-20260225_143243/help_kan.txt`, `.tmp/collab-post-fix-20260225_143243/help_kan_serve.txt`, `.tmp/collab-post-fix-20260225_143243/runtime_log_full.log`, `.tmp/collab-post-fix-20260225_143243/runtime_log_mcp_filter.txt`, `.tmp/collab-post-fix-20260225_143243/runtime_log_gatekeeping_ops_search.txt`, `.tmp/collab-post-fix-20260225_143243/user_stdout_logging_observation.txt`
- Detailed outcomes:
  - logging help discoverability:
    - `./kan --help` produced `error: flag: help requested` (no usable help text)
    - `./kan serve --help` attempted startup/open path and failed on runtime path instead of printing subcommand help
    - status: FAIL
  - runtime logs exist:
    - `charmbracelet/log` output is present in file sink for startup/command flow
    - status: PASS
  - MCP operation visibility in file sink:
    - filtered search of file sink showed no clear MCP mutation operation entries from the automated gatekeeping calls
    - status: FAIL
  - stdout vs file sink parity:
    - user observed guardrail mutation errors on serve stdout/stderr (`lease project mismatch`, `lease not found`)
    - same operation-level signals were not reliably mirrored into `.kan/log` file sink
    - status: FAIL
  - notifications panel bubbling:
    - requires manual TUI validation
    - status: BLOCKED
- Locked issue:
  - guardrail/MCP mutation errors are observable on serve stdout/stderr but are not reliably mirrored into `.kan/log` file sink; sink parity must be fixed.
  - requested notifications-panel redesign process (ASCII-art proposal + clarifying questions before implementation) was not executed prior to this run; this is a process miss and remains required before building the new panel design.

## 7) Final Post-Fix Verdict

- Overall: FAIL
- Remaining blockers:
  1. Manual TUI regression sweep (Sections 4 and 5) not executed yet; required user-driven validation remains.
  2. Logging/help discoverability gaps remain (no usable CLI help path for debug activation; MCP mutation errors currently split between stdout and file sink, with missing file-sink parity for key MCP operations).
  3. MCP `kan_restore_task` fails guardrail path (`mutation lease is required`) and currently exposes no actor/lease tuple fields to satisfy guardrails.
- Follow-up actions:
  1. Run full manual TUI post-fix validation section-by-section and record PASS/FAIL per item in this worksheet with screenshots/log evidence.
  2. Implement/fix logging discoverability + MCP operation logging visibility, then rerun Section 6 and re-check final verdict.
  3. Fix MCP restore-task transport contract/guard context (or guardrail policy) so restore works under intended actor model; rerun Section 8 tool sweep.

## 8) Automated MCP Tool Sweep

Automated sweep completed without manual TUI input.

- Result: FAIL
- Evidence: MCP call transcript in this run, `.tmp/collab-post-fix-20260225_143243/export_snapshot_after_sweep_project.json`, `.tmp/collab-post-fix-20260225_143243/columns.tsv`, `.tmp/collab-post-fix-20260225_143243/cleanup_notes.md`
- Tool/behavior matrix:
  - PASS:
    - `kan_get_bootstrap_guide`
    - `kan_list_kind_definitions`
    - `kan_upsert_kind_definition`
    - `kan_create_project`
    - `kan_list_projects`
    - `kan_issue_capability_lease`
    - `kan_create_task`
    - `kan_list_tasks`
    - `kan_list_child_tasks`
    - `kan_update_task`
    - `kan_move_task`
    - `kan_reparent_task` (valid parent path)
    - `kan_delete_task` (archive and hard modes)
    - `kan_search_task_matches`
    - `kan_create_comment`
    - `kan_list_comments_by_target`
    - `kan_raise_attention_item` (valid kind)
    - `kan_list_attention_items`
    - `kan_resolve_attention_item`
    - `kan_set_project_allowed_kinds`
    - `kan_list_project_allowed_kinds`
    - `kan_update_project`
    - `kan_get_project_dependency_rollup`
    - `kan_list_project_change_events`
    - `kan_heartbeat_capability_lease`
    - `kan_renew_capability_lease`
    - `kan_revoke_capability_lease`
    - `kan_revoke_all_capability_leases`
  - FAIL:
    - `kan_restore_task`
      - response: `guardrail_failed ... mutation lease is required`
      - MCP surface currently does not provide actor/lease tuple fields for restore, creating a contract mismatch with guardrails.
  - Expected edge validation errors (not counted as failures):
    - invalid attention kind `risk` -> `invalid_request`
    - invalid reparent empty parent id -> `invalid_request`
    - update blocked when `project` kind removed from allowlist -> expected fail-closed
- Cleanup note:
  - Sweep tasks were hard-deleted with a temporary cleanup lease.
  - Sweep/gatekeeping projects remain because MCP surface currently has no project delete/archive/restore tool.

## 9) Locked Planning Notes (User)

Planning-required product direction captured during this validation run:
- single-source-of-truth strategy:
  - Kan should continue moving toward one DB-backed operational source of truth and avoid markdown sprawl for active execution state.
  - Agents should update existing DB state for the current scope instead of creating duplicate records/notes by default.
- adaptable field model:
  - plan flexible per-project fields to support better user/agent/subagent conversations and agent-to-agent collaboration.
  - keep flexibility bounded with strong defaults/standards to avoid schema and UX chaos.
- agent behavior policy/docs:
  - `AGENTS.md` guidance should explicitly require agents to fetch current DB state for the relevant scope/idea, then update existing records before creating new entries.
  - this needs a dedicated planning/design pass with concrete rules and conflict-resolution behavior.
- status: PLANNING REQUIRED (not a completed test item yet)

## 10) Comprehensive Existence/Gap Audit (Recovered Old Worksheet + Explorer Review)

Audit method:
- recovered deleted worksheet from git:
  - `git show HEAD:COLLABORATIVE_FULL_E2E_TEST_WORKSHEET.md`
  - rerun evidence (2026-02-26): `git show HEAD:COLLABORATIVE_FULL_E2E_TEST_WORKSHEET.md > .tmp/old-worksheet-audit/COLLABORATIVE_FULL_E2E_TEST_WORKSHEET.HEAD.md`
- explorer subagent requirement extraction:
  - agent id: `019c9918-59a6-70d1-8a2d-62ff8d0a7826`
- explorer subagent code-gap audit:
  - agent id: `019c9918-7518-7352-b164-17d87e39e9bc`
- explorer subagent test/process coverage audit:
  - agent id: `019c9918-9802-7b22-9697-42c6fb2edcea`
- rerun parallel explorer sweep (2026-02-26):
  - requirement extraction: `019c9922-34bd-7ad0-a09f-1d44f2b5f548`
  - code existence/gap audit: `019c9922-34f8-7331-9456-65924e82992e`
  - test/docs/process audit: `019c9922-34e1-7a11-82b9-93c40eb1d670`
  - focused verification sweep (refresh + notifications): `019c9921-b2fc-7480-9b8f-ea1615c87e4c`

Legend:
- `IMPLEMENTED` = exists in current codebase.
- `PARTIAL` = some behavior exists, requested target behavior still missing.
- `MISSING` = not implemented.
- `PLANNING` = explicitly captured as planning requirement, not implementation-complete.

| Req ID | Requirement (from recovered collaborative expectations) | Status |
|---|---|---|
| REQ-001 | External MCP-originated changes refresh current project without project switch workaround | MISSING |
| REQ-002 | Level-scoped notifications panel with global count and quick-nav controls | PARTIAL |
| REQ-003 | Important runtime/MCP warnings/errors surfaced to notifications + quick info modal | PARTIAL |
| REQ-004 | ASCII-art + clarifying-question checkpoint before notifications redesign | MISSING |
| REQ-005 | Full runtime + MCP logging with meaningful bubbling | PARTIAL |
| REQ-006 | MCP/runtime operations produce useful logs with charm/log | PARTIAL |
| REQ-007 | No silent failures for key operations | PARTIAL |
| REQ-008 | Debug logging activation path ergonomic without manual config edits | MISSING |
| REQ-009 | Help output clearly documents debug logging activation path | MISSING |
| REQ-010 | Stdout/stderr guardrail logs mirrored to `.kan/log` file sink (sink parity) | MISSING |
| REQ-011 | `v` must not break typing in text inputs | IMPLEMENTED |
| REQ-012 | Typing keys preserved in inputs; emoji support in text fields | IMPLEMENTED |
| REQ-013 | Selection mode moved off `v` to control chord direction | IMPLEMENTED |
| REQ-014 | Contextual `esc`/info return-to-origin behavior | IMPLEMENTED |
| REQ-015 | Due picker include-time toggle works | IMPLEMENTED |
| REQ-016 | Due picker fuzzy date behavior (example `2-2`) | IMPLEMENTED |
| REQ-017 | Entity-specific archive confirm labels | IMPLEMENTED |
| REQ-018 | `.` quick action restore for archived items | IMPLEMENTED |
| REQ-019 | No plain `a` archive trigger; archive/restore via quick action + command palette | MISSING |
| REQ-020 | Archived projects hidden by default with explicit picker toggle | IMPLEMENTED |
| REQ-021 | Reworked project-level archived UX (`t` mismatch concern) | MISSING |
| REQ-022 | Archived search filter decoupled from unrelated global visibility toggle | IMPLEMENTED |
| REQ-023 | Agent spoof prevention (`actor_type=user` with guard tuple blocked) | IMPLEMENTED |
| REQ-024 | User vs agent attribution correctness | IMPLEMENTED |
| REQ-025 | Strict scope guardrails (orchestrator/worker) | IMPLEMENTED |
| REQ-026 | Notification-driven approval UX for agent identity/scope requests | MISSING |
| REQ-027 | MCP restore-task guard/transport contract works under intended actor model | MISSING |
| REQ-028 | `?` help available in all modals/screens | IMPLEMENTED |
| REQ-029 | Help includes copy/paste/select and modal-specific guidance | IMPLEMENTED |
| REQ-030 | Icon feature behavior/purpose visibly functional and defined | IMPLEMENTED |
| REQ-031 | Marker meaning clarity (`!3`, overdue markers) | IMPLEMENTED |
| REQ-032 | Template/docs encode approval/scope flow (`AGENTS.md` + template path) | MISSING |
| REQ-033 | Single DB-backed source-of-truth direction (reduce md sprawl) | PLANNING |
| REQ-034 | Agents update existing scoped DB state before creating duplicates | PLANNING |
| REQ-035 | Adaptable per-project conversation fields with bounded standards | PLANNING |
| REQ-036 | Detailed policy/planning for DB-first agent update behavior in `AGENTS.md` | PLANNING |

Highest-risk unresolved items (priority):
1. REQ-001: no external mutation auto-refresh in TUI (stale board risk).
2. REQ-002/003: requested two-part notifications workflow (global count + quick-nav + drill-in) absent.
3. REQ-010: stdout/file sink parity gap for MCP guardrail logs.
4. REQ-027: `kan_restore_task` MCP contract mismatch (guardrail requires lease, tool path lacks required tuple).
5. REQ-019/021: archive-key and project archived UX policy mismatches still open.

## 11) TUI Carry-Forward (Migrated From Retired Worksheet)

Source:
- retired worksheet: `TUI_MANUAL_TEST_WORKSHEET.md` (retired 2026-02-27)
- extracted unresolved anchors from latest prior run before retirement.

Legend:
- `OPEN` means unresolved and still requires implementation or validation evidence in this worksheet.
- `MIGRATED` means source anchor was moved here and should not be tracked in a separate TUI worksheet anymore.

| TUI-CF ID | Migrated issue | Prior anchor(s) | Maps to requirement(s) | Status |
|---|---|---|---|---|
| TUI-CF-01 | Due date/time UX clarity remains unclear in modal flows. | `S2.1`, `S2.3` | REQ-015, REQ-016 | OPEN / MIGRATED |
| TUI-CF-02 | Contextual `esc` back-stack behavior still needs full manual pass confirmation. | `S3.1` | REQ-014 | OPEN / MIGRATED |
| TUI-CF-03 | Modal help/key guidance consistency remains insufficient in some flows. | `S3.2` | REQ-004, REQ-005, REQ-028, REQ-029 | OPEN / MIGRATED |
| TUI-CF-04 | Subtask save return-origin behavior requires explicit validation/fix confirmation. | `S3.3` | REQ-014 | OPEN / MIGRATED |
| TUI-CF-05 | Text-input key handling/select-mode collisions need explicit revalidation in all forms. | `S4.1` note mismatch | REQ-001, REQ-002, REQ-003, REQ-011, REQ-012, REQ-013 | OPEN / MIGRATED |
| TUI-CF-06 | Project color/accent behavior remains unclear/insufficiently validated. | `S8.1` | REQ-030 | OPEN / MIGRATED |
| TUI-CF-07 | Labels-config scope behavior still has unresolved bug notes. | `S8.3` | REQ-032 | OPEN / MIGRATED |
| TUI-CF-08 | Consolidated rerun anchors were incomplete and blocked final sign-off in the retired worksheet. | `D0.1`..`D7.1` | REQ-001..REQ-032 | OPEN / MIGRATED |

Closure rule:
- A `TUI-CF-*` item can only be closed in this worksheet with a dated PASS/FAIL decision and concrete evidence path(s).

## 12) Phase 0 Closeout Run (2026-02-27)

Run artifact root:
- `.tmp/phase0-collab-20260227_141800/`

### 12.1 Phase 0 task tracker

| Task | Status | Evidence | Notes |
|---|---|---|---|
| P0-T01 Manual TUI validation for C4/C6/C9/C10/C11/C12/C13 | BLOCKED | `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md`, `.tmp/phase0-collab-20260227_141800/manual/checklist.md` | Requires user-driven TUI execution and screenshot/log capture in this active run. |
| P0-T02 Archived/search/keybinding targeted checks | BLOCKED | `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md`, `.tmp/phase0-collab-20260227_141800/manual/checklist.md` | Requires manual UX verification in running TUI session. |
| P0-T03 Focused MCP rerun (`kan_restore_task`, `capture_state`) | FAIL | `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`, `.tmp/phase0-collab-20260227_141800/http_capture_state_project.json` | `capture_state` readiness passes; `kan_restore_task` still fails guardrail path (`mutation lease is required`). |
| P0-T04 Logging/help discoverability evidence capture | FAIL | `.tmp/phase0-collab-20260227_141800/phase0_preflight_summary.md`, `.tmp/phase0-collab-20260227_141800/help_kan.txt`, `.tmp/phase0-collab-20260227_141800/help_kan_serve.txt`, `.tmp/phase0-collab-20260227_141800/runtime_log_focus_filter.txt` | Help output path remains broken; operation-level log parity remains insufficient in this probe. Remediation requirements now include Charm/Fang-based help UX and first-launch config bootstrap behavior (copy default example config when missing). |
| P0-T05 Fill blank checkpoints/sign-offs in `MCP_DOGFOODING_WORKSHEET.md` | PASS | `MCP_DOGFOODING_WORKSHEET.md` + run artifacts under `.tmp/phase0-collab-20260227_141800/` | Completed: all USER NOTES rows and final sign-off fields now have explicit `pass`/`fail`/`blocked` values with evidence paths. |
| P0-T06 Update this worksheet with final evidence and verdict | BLOCKED | `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`, `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md` | Worksheet updated with current evidence and blockers, but final closeout verdict is blocked on pending user-driven manual collaborative checks. |

### 12.2 Automated checks executed in this run

1. `just check` -> PASS (`.tmp/phase0-collab-20260227_141800/just_check.txt`)
2. `just ci` -> PASS (`.tmp/phase0-collab-20260227_141800/just_ci.txt`)
3. `just test-golden` -> PASS (`.tmp/phase0-collab-20260227_141800/just_test_golden.txt`)
4. `just build` -> PASS with environment warning (`.tmp/phase0-collab-20260227_141800/just_build.txt`)
5. runtime listener/health checks -> PASS (`.tmp/phase0-collab-20260227_141800/port_18080_listener.txt`, `.tmp/phase0-collab-20260227_141800/healthz.txt`, `.tmp/phase0-collab-20260227_141800/readyz.txt`)

### 12.3 Immediate blockers called out

1. `./kan --help` and `./kan serve --help` do not provide expected discoverable help output in this run.
2. `kan_restore_task` remains unusable in current MCP surface due guardrail tuple mismatch.
3. Remaining TUI-centric collaborative checks require explicit user interaction and cannot be auto-validated by agent-only execution.

### 12.4 User-Directed Additions And Process Contract

Additional remediation requirements captured from user direction:
1. First-launch bootstrap behavior:
   - when launching `kan` for the first time and config file is missing, copy from the default example config (`config.example.toml`) instead of creating a short/minimal config stub.
2. Help/CLI UX behavior:
   - replace current failing `--help` behavior with a designed help surface using Charm/Fang so help output is readable, attractive, and discoverable.
3. `kan_restore_task` contract remediation:
   - close the mutation guardrail mismatch by aligning MCP restore transport with actor/lease tuple expectations enforced by the service guardrail path.

Execution/process contract for remaining Phase 0 testing:
1. Run collaborative validation section-by-section.
2. User provides detailed notes/evidence for each section.
3. Agent records notes verbatim in active markdown worksheets without shortening detail.
4. Agent updates PASS/FAIL/BLOCKED outcomes immediately after each section handoff.
5. No move to feature/fix implementation until Phase 0 closeout evidence is complete.
6. Final step of this testing process:
   - run subagents for code inspection,
   - run Context7 for relevant library/API guidance,
   - run targeted web research where needed,
   - propose fix options,
   - add agreed proposals to active markdown only after user+agent consensus.

### 12.5 `kan_restore_task` Mutation Guardrail Root-Cause Summary (Explorer Audit)

Observed failure:
- `kan_restore_task` returns `guardrail_failed ... mutation lease is required` for agent-attributed archived tasks.

Code-level explanation:
1. MCP `kan.restore_task` currently accepts only `task_id` and calls restore without actor/lease tuple.
2. Restore path in MCP/common surfaces does not carry actor lease data like update/move/delete paths.
3. Restore adapter path does not attach mutation guard context before invoking service restore.
4. Service restore enforces mutation guard using persisted `UpdatedByType`; for non-user actor types this requires a valid lease tuple.
5. Without that tuple in context, guardrail returns `ErrMutationLeaseRequired`, mapped to MCP `guardrail_failed`.

Primary evidence references from explorer review:
1. `internal/adapters/server/mcpapi/extended_tools.go`
2. `internal/adapters/server/common/mcp_surface.go`
3. `internal/adapters/server/common/app_service_adapter_mcp.go`
4. `internal/app/service.go`
5. `internal/app/kind_capability.go`
