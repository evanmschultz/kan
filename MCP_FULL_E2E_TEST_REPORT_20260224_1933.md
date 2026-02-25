# MCP Full E2E Test Report

- Run date: 2026-02-24 19:33 (local)
- Orchestrator mode: user-approved option 1 (orchestrator-run MCP/HTTP execution due subagent localhost restrictions)
- Artifact root: `.tmp/mcp-e2e-20260224_185728/`
- Lane `.tmp/mcp_lane_*.md` files: deleted after synthesis per runbook contract
- Final report owner file: this document

## 0. Remediation Update (2026-02-25)

This report captured a failing run. A follow-up remediation pass was executed with fresh evidence.

Remediation evidence root:
- `.tmp/verify-d1-d6-20260224_200201/`

Remediation status by defect:
1. D1 (`capture_state` hash parity): fixed and verified.
   - Result: `D1_MATCH=true` (MCP and HTTP `state_hash` equal).
2. D2 (`revoke_all_capability_leases` unknown scope): fixed and verified.
   - Result: `not_found` error returned (`D2_ERR` non-empty not_found message).
3. D3 (`create_comment` unknown target): fixed and verified.
   - Result: `not_found` error returned (`D3_ERR` non-empty not_found message).
4. D4 (`update_task` title-only update): fixed and verified.
   - Result: `D4_OK=true`.
5. D5 (P0 clean-instance precondition): addressed and verified.
   - Result: `P0_COUNT=0` on isolated temp DB.
   - Process hardening added in runbook: `MCP_FULL_TESTER_AGENT_RUNBOOK.md` now requires isolated temp DB and hard-stop on non-empty P0.
6. D6 (TUI hierarchy focus discoverability): addressed in TUI + tests.
   - Result: board info line now shows hierarchy level, child count, and `f/F` focus guidance.
   - Test coverage: `TestModelViewShowsSubtreeDiscoverabilityHint`, `TestModelProjectionFocusBreadcrumbMode`, `TestModelFocusSubtreeRendersBoardForHierarchyLevels`.

Gate status after remediation:
- `just check`: pass
- `just ci`: pass
- `just test-golden`: pass

## 1. Server/Config Confirmation

- Confirmed server command:
  - `./kan serve --http 127.0.0.1:18080 --api-endpoint /api/v1 --mcp-endpoint /mcp`
- Confirmed HTTP bind/port: `127.0.0.1:18080`
- Confirmed API endpoint: `/api/v1`
- Confirmed MCP endpoint: `/mcp`
- `.codex/config.toml` MCP URL:
  - `http://127.0.0.1:18080/mcp`
- Runtime MCP URL match: `confirmed`

## 2. Lane Summaries

- Lane A (`.tmp/mcp_lane_a_transport.md`): pass=9 fail=0 blocked=0
- Lane B (`.tmp/mcp_lane_b_capture_attention.md`): pass=38 fail=1 blocked=0
- Lane C (`.tmp/mcp_lane_c_projects_tasks_search.md`): pass=51 fail=1 blocked=0
- Lane D (`.tmp/mcp_lane_d_kinds_leases_comments.md`): pass=39 fail=2 blocked=0
- Lane E (`.tmp/mcp_lane_e_parity_gates.md`): pass=6 fail=1 blocked=0

## 3. Full Tool Matrix (30 MCP Tools)

| tool | total_cases | pass | fail | blocked |
|---|---:|---:|---:|---:|
| kan.capture_state | 12 | 12 | 0 | 0 |
| kan.create_comment | 6 | 5 | 1 | 0 |
| kan.create_project | 5 | 5 | 0 | 0 |
| kan.create_task | 10 | 10 | 0 | 0 |
| kan.delete_task | 4 | 4 | 0 | 0 |
| kan.get_bootstrap_guide | 1 | 1 | 0 | 0 |
| kan.get_project_dependency_rollup | 3 | 3 | 0 | 0 |
| kan.heartbeat_capability_lease | 4 | 4 | 0 | 0 |
| kan.issue_capability_lease | 4 | 4 | 0 | 0 |
| kan.list_attention_items | 10 | 10 | 0 | 0 |
| kan.list_child_tasks | 4 | 4 | 0 | 0 |
| kan.list_comments_by_target | 3 | 3 | 0 | 0 |
| kan.list_kind_definitions | 2 | 2 | 0 | 0 |
| kan.list_project_allowed_kinds | 2 | 2 | 0 | 0 |
| kan.list_project_change_events | 4 | 4 | 0 | 0 |
| kan.list_projects | 3 | 2 | 1 | 0 |
| kan.list_tasks | 4 | 4 | 0 | 0 |
| kan.move_task | 4 | 4 | 0 | 0 |
| kan.raise_attention_item | 6 | 6 | 0 | 0 |
| kan.renew_capability_lease | 5 | 5 | 0 | 0 |
| kan.reparent_task | 4 | 4 | 0 | 0 |
| kan.resolve_attention_item | 3 | 3 | 0 | 0 |
| kan.restore_task | 3 | 3 | 0 | 0 |
| kan.revoke_all_capability_leases | 5 | 4 | 1 | 0 |
| kan.revoke_capability_lease | 3 | 3 | 0 | 0 |
| kan.search_task_matches | 3 | 3 | 0 | 0 |
| kan.set_project_allowed_kinds | 3 | 3 | 0 | 0 |
| kan.update_project | 4 | 4 | 0 | 0 |
| kan.update_task | 4 | 3 | 1 | 0 |
| kan.upsert_kind_definition | 4 | 4 | 0 | 0 |

## 4. Protocol/Stateless Findings (Lane A)

- `initialize` behavior is deterministic across:
  - current protocol `2025-06-18`
  - legacy protocol `2024-11-05`
  - unsupported future protocol (server falls back to a server-selected version)
  - missing `protocolVersion` (server still returns deterministic response)
- `tools/list` included all 30 expected tool names.
- No `Mcp-Session-Id` response header observed in stateless mode.
- Requests still succeed with bogus `Mcp-Session-Id` request header.
- Unknown method response is deterministic across repeated calls.
- Invalid JSON-RPC envelope response is deterministic across repeated calls.

Evidence:
- `.tmp/mcp-e2e-20260224_185728/transport-*`

## 5. Guardrail/Gatekeeping Findings

Gate commands (Lane E):
- `just check`: pass (exit 0)
- `just ci`: pass (exit 0)
- `just test-golden`: pass (exit 0)

Guardrail observations:
- Non-user mutation without lease tuple failed closed (`invalid_request` requiring lease tuple): pass behavior.
- Bad lease token / wrong instance combinations failed closed: pass behavior.
- `kan.revoke_all_capability_leases` accepted unknown scope tuple (`scope_type=task`, unknown `scope_id`) and returned `updated=true`: fail-closed expectation mismatch.

Evidence:
- `.tmp/mcp-e2e-20260224_185728/parity-gate-just-check.log`
- `.tmp/mcp-e2e-20260224_185728/parity-gate-just-ci.log`
- `.tmp/mcp-e2e-20260224_185728/parity-gate-just-test-golden.log`
- `.tmp/mcp-e2e-20260224_185728/policy-guardrail-*`
- `.tmp/mcp-e2e-20260224_185728/policy-lease-*`

## 6. HTTP Parity Findings

Checked parity targets:
- `capture_state`: fail (state hash mismatch between MCP and HTTP for same scope tuple)
- `attention list`: pass
- `attention raise`: pass
- `attention resolve`: pass

Evidence:
- `.tmp/mcp-e2e-20260224_185728/parity-mcp-capture.json`
- `.tmp/mcp-e2e-20260224_185728/parity-http-capture.json`
- `.tmp/mcp-e2e-20260224_185728/parity-mcp-attn-*.json`
- `.tmp/mcp-e2e-20260224_185728/parity-http-attn-*.json`

## 7. Defects / Findings

### D1 (High): `capture_state` parity hash mismatch between MCP and HTTP
- Severity: High
- Repro:
  1. MCP: `kan.capture_state` with identical project/scope tuple.
  2. HTTP: `GET /api/v1/capture_state` with same tuple.
  3. Compare `state_hash`.
- Expected: same deterministic hash for same underlying state tuple.
- Actual: hashes differ consistently.
- Evidence:
  - `.tmp/mcp-e2e-20260224_185728/parity-mcp-capture.json`
  - `.tmp/mcp-e2e-20260224_185728/parity-http-capture.json`

### D2 (High): `kan.revoke_all_capability_leases` does not fail closed for unknown scope tuple
- Severity: High
- Repro:
  1. Call `kan.revoke_all_capability_leases` with valid `project_id`, `scope_type=task`, and unknown `scope_id`.
- Expected: deterministic not_found/guardrail failure.
- Actual: success payload with `updated=true`.
- Evidence:
  - `.tmp/mcp-e2e-20260224_185728/policy-guardrail-scope-mismatch.json`

### D3 (Medium): `kan.create_comment` accepts unknown `target_id`
- Severity: Medium
- Repro:
  1. Call `kan.create_comment` with real `project_id`, `target_type=task`, and unknown `target_id`.
- Expected: not_found or guardrail failure.
- Actual: comment is created successfully.
- Evidence:
  - `.tmp/mcp-e2e-20260224_185728/policy-comment-create-notfound.json`

### D4 (Medium): `kan.update_task` minimal update path fails on existing task
- Severity: Medium
- Repro:
  1. Call `kan.update_task` with valid `task_id` + `title` only.
- Expected: success update.
- Actual: `invalid_request` with `invalid priority`.
- Evidence:
  - `.tmp/mcp-e2e-20260224_185728/work-update-task-min.json`

### D5 (Medium): P0 clean-instance precondition was not satisfied
- Severity: Medium (test-run integrity)
- Repro:
  1. P0 check `kan.list_projects` expected empty.
- Expected: zero projects.
- Actual: non-empty (`projects=6`) due pre-existing/mutated runtime DB.
- Evidence:
  - `.tmp/mcp-e2e-20260224_185728/capture-attn-p0-list-projects.json`

### D6 (UX Design): TUI branch navigation/discoverability gap
- Severity: Medium (UX/design)
- User-observed behavior:
  - Branch data exists in DB and is visible only after specific project navigation flow.
  - No direct/obvious way to enter a branch and inspect its nested contents from the shown view.
- Evidence:
  - User-provided screenshots in this run (Image #1 and Image #2)
  - DB verification during run showed branch row present in `work_items`.

## 8. Final Verdict (Historical Run)

- Historical run verdict (this file's original execution window): `fail`
- Current status after remediation update at top of file: `superseded` by the 2026-02-25 remediation evidence.

Reason:
- This section reflects the original failing run only.
- See `0. Remediation Update (2026-02-25)` for current defect status and passing gate evidence.

## 9. Explicit User Validation Checklist (Historical)

1. Historical checklist retained for traceability of the original failing run.
2. Use the remediation section and current runbook for present-state validation.
