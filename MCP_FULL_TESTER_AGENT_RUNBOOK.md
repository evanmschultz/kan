# MCP Full Tester Agent Runbook

Date: 2026-02-25
Owner: tester-orchestrator agent
Status: ready_to_run

## 1) Objective
Run a full MCP-first, end-to-end validation sweep against the local `kan` server and produce one evidence-backed final report.

Hard requirement: cover **every currently registered MCP tool**, protocol/stateless behavior, guardrail/fail-closed paths, and HTTP parity checks where relevant.

Current MCP tool surface to validate:
1. `kan.capture_state`
2. `kan.get_bootstrap_guide`
3. `kan.list_projects`
4. `kan.create_project`
5. `kan.update_project`
6. `kan.list_tasks`
7. `kan.create_task`
8. `kan.update_task`
9. `kan.move_task`
10. `kan.delete_task`
11. `kan.restore_task`
12. `kan.reparent_task`
13. `kan.list_child_tasks`
14. `kan.search_task_matches`
15. `kan.list_project_change_events`
16. `kan.get_project_dependency_rollup`
17. `kan.list_kind_definitions`
18. `kan.upsert_kind_definition`
19. `kan.set_project_allowed_kinds`
20. `kan.list_project_allowed_kinds`
21. `kan.issue_capability_lease`
22. `kan.heartbeat_capability_lease`
23. `kan.renew_capability_lease`
24. `kan.revoke_capability_lease`
25. `kan.revoke_all_capability_leases`
26. `kan.create_comment`
27. `kan.list_comments_by_target`
28. `kan.list_attention_items`
29. `kan.raise_attention_item`
30. `kan.resolve_attention_item`

## 2) Mandatory Safety + Approval Policy
1. If a command needs sandbox escalation, stop and ask the user with a human-readable reason.
2. No workaround behavior if blocked by permissions/network/sandbox.
3. No destructive cleanup outside test temp directories without user approval.
4. If behavior is ambiguous product policy, pause and ask user before changing expectations.

## 3) Mandatory User Checkpoint Before Any Network Calls
Ask user to start and confirm server first.

Required user confirmations:
1. Exact running serve command.
2. HTTP bind/port.
3. API endpoint path.
4. MCP endpoint path.
5. Confirmation that `.codex/config.toml` MCP URL matches runtime endpoint.

Required runtime endpoint shape:
- `http://127.0.0.1:<port>/mcp`

## 4) Files and Reporting Contract
Create and maintain exactly these artifacts:
1. Final root report (single source of truth):
   - `MCP_FULL_E2E_TEST_REPORT_<YYYYMMDD_HHMM>.md`
2. Lane handoff notes (`.tmp`, one per lane):
   - `.tmp/mcp_lane_a_transport.md`
   - `.tmp/mcp_lane_b_capture_attention.md`
   - `.tmp/mcp_lane_c_projects_tasks_search.md`
   - `.tmp/mcp_lane_d_kinds_leases_comments.md`
   - `.tmp/mcp_lane_e_parity_gates.md`
3. Raw call artifacts:
   - `.tmp/mcp-e2e-<timestamp>/`

After integration:
1. compile all lane findings into the single root report,
2. delete `.tmp/mcp_lane_*.md` lane files,
3. keep raw protocol artifacts under `.tmp/mcp-e2e-<timestamp>/`.

## 5) Parallel Subagent Plan (Required)
Run lanes in parallel with non-overlapping output locks.

Lane A (transport/protocol/stateless)
- Lock scope: `.tmp/mcp_lane_a_transport.md`, `.tmp/mcp-e2e-*/transport-*`
- Acceptance: protocol/version/init/tools-list/session-header/stateless checks complete.

Lane B (capture + attention + bootstrap)
- Lock scope: `.tmp/mcp_lane_b_capture_attention.md`, `.tmp/mcp-e2e-*/capture-attn-*`
- Acceptance: all `capture_state`, bootstrap, and attention tool options + errors complete.

Lane C (projects/tasks/search/change/dependency)
- Lock scope: `.tmp/mcp_lane_c_projects_tasks_search.md`, `.tmp/mcp-e2e-*/work-*`
- Acceptance: project/task/search/change/rollup tool options + errors complete.

Lane D (kinds/allowlist/leases/comments)
- Lock scope: `.tmp/mcp_lane_d_kinds_leases_comments.md`, `.tmp/mcp-e2e-*/policy-*`
- Acceptance: kind, lease, and comment tool options + guardrail checks complete.

Lane E (parity + gates + report synthesis)
- Lock scope: `.tmp/mcp_lane_e_parity_gates.md`, final root report
- Acceptance: HTTP/MCP parity checks + `just check` + `just ci` + `just test-golden` complete.

## 6) Test Data Strategy
Run two phases:

Phase P0: Empty instance validation
1. Start with clean DB and no projects.
2. Validate empty-instance behavior:
   - `kan.list_projects` returns empty.
   - `kan.capture_state` with unknown project returns deterministic error (`not_found` or `bootstrap_required` class).
   - `kan.get_bootstrap_guide` returns actionable setup guidance.

Phase P1: Seeded hierarchy validation
Seed deterministic data for all scope levels:
- project
- branch
- phase
- subphase
- task
- subtask

Also create:
- at least 2 attention items (`requires_user_action` true/false)
- comments on at least one task target
- at least one dependency edge/blocking signal

Record all IDs in report:
- `PROJECT_ID`, `BRANCH_ID`, `PHASE_ID`, `SUBPHASE_ID`, `TASK_ID`, `SUBTASK_ID`, `ATTN_ID_*`, `LEASE_INSTANCE_ID`, `LEASE_TOKEN`.

## 7) MCP Protocol and Stateless Matrix (Lane A)
Required checks:
1. `initialize` with current protocol.
2. `initialize` with `2024-11-05` legacy protocol.
3. `initialize` with unsupported future protocol (verify deterministic fallback/error behavior).
4. `initialize` missing `protocolVersion` (record exact behavior).
5. `tools/list` includes all 30 tool names above.
6. no `Mcp-Session-Id` header in stateless mode.
7. calls still work when bogus `Mcp-Session-Id` header is sent.
8. unknown method handling is deterministic.
9. invalid JSON-RPC envelope handling is deterministic.

## 8) Tool-by-Tool Coverage Matrix
For each tool, run:
1. success path with minimal required args,
2. success path with optional args,
3. missing-required-arg failure,
4. invalid value/type failure,
5. not-found/guardrail path where applicable.

### 8.1 `kan.capture_state`
Cover scope tuple variants:
- `project`, `branch`, `phase`, `subphase`, `task`, `subtask`

Failure matrix:
- missing `project_id`
- invalid `scope_type`
- invalid `view`
- project scope mismatch (`scope_id != project_id`)
- non-project missing `scope_id`

### 8.2 Bootstrap tool
- `kan.get_bootstrap_guide` response shape + actionable fields.

### 8.3 Attention tools
- `kan.list_attention_items`: all scope levels + `state` filters (`open`, `acknowledged`, `resolved`) + invalid state.
- `kan.raise_attention_item`: required and optional fields; missing requireds; invalid scope tuple.
- `kan.resolve_attention_item`: required `id`, optional fields, unknown id.

### 8.4 Project/task/search/change tools
- `kan.list_projects`, `kan.create_project`, `kan.update_project`
- `kan.list_tasks`, `kan.create_task`, `kan.update_task`, `kan.move_task`, `kan.delete_task`, `kan.restore_task`, `kan.reparent_task`, `kan.list_child_tasks`
- `kan.search_task_matches`
- `kan.list_project_change_events`
- `kan.get_project_dependency_rollup`

### 8.5 Kind/allowlist tools
- `kan.list_kind_definitions`
- `kan.upsert_kind_definition`
- `kan.set_project_allowed_kinds`
- `kan.list_project_allowed_kinds`

### 8.6 Lease tools
- `kan.issue_capability_lease`
- `kan.heartbeat_capability_lease`
- `kan.renew_capability_lease`
- `kan.revoke_capability_lease`
- `kan.revoke_all_capability_leases`

Guardrail checks:
- non-user mutation calls without valid lease tuple should fail closed.
- bad lease token / wrong instance / scope mismatch should fail closed.

### 8.7 Comment tools
- `kan.create_comment`
- `kan.list_comments_by_target`

## 9) HTTP Parity Checks (Lane E)
For same scope tuple and entities, compare MCP outputs against HTTP endpoints where available:
1. `capture_state`
2. `attention` list/raise/resolve
3. key fields parity: scope path, counts, IDs, state transitions, error categories

## 10) Evidence Format (Strict)
Each test row must include:
1. test id
2. command/payload
3. HTTP status
4. structured response assertion
5. expected vs actual
6. result: `pass|fail|blocked`
7. artifact file paths

No blank result rows allowed.

## 11) Required Command Gates
After lane integration and before final verdict:
1. `just check`
2. `just ci`
3. `just test-golden`

Do not mark complete if any gate fails.

## 12) Final Deliverable Structure
Final report must include:
1. server/config confirmation (including `.codex/config.toml` MCP URL match)
2. all lane summaries
3. full tool matrix pass/fail table
4. protocol/stateless findings
5. guardrail/gatekeeping findings
6. parity findings
7. defects (severity + reproduction + evidence)
8. final verdict: `pass|fail|blocked`
9. explicit user validation checklist

## 13) Prompt Discipline for Worker Subagents
Each worker prompt must include:
1. lane objective
2. lock scope + explicit out-of-scope
3. acceptance criteria
4. required evidence format
5. stop-for-approval policy
6. no-code-change rule unless explicitly authorized by orchestrator

## 14) Non-Negotiables
1. Do not mutate product code in this run unless user explicitly authorizes patching defects.
2. This run is validation-first and evidence-first.
3. If any test cannot be run, mark `blocked` with concrete blocker and required user action.
