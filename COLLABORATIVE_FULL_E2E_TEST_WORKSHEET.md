# Collaborative Full E2E Test Worksheet (User + Agent)

Date: __________
Tester agent/session: __________
User: __________
Run artifact dir: `.tmp/collab-e2e-<timestamp>/`

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
- Result: __________
- Evidence: ______________________________________________

### A2. MCP bootstrap guide is available
- Tool: `kan.get_bootstrap_guide`
- Expected: deterministic setup guidance returned, not internal error
- Result: __________
- Evidence: ______________________________________________

### A3. `capture_state` on empty instance returns bootstrap-required guidance
- Tool: `kan.capture_state`
- Expected: bootstrap-required behavior, actionable next steps
- Result: __________
- Evidence: ______________________________________________

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
- project_id: ____________________________________________
- branch_id: _____________________________________________
- phase_id: ______________________________________________
- subphase_id: ___________________________________________
- task_id: _______________________________________________
- top_task_id: ___________________________________________

Evidence: ________________________________________________

## 6. Section C: TUI Behavior Validation (Manual + Agent Guided)

Launch TUI against the same DB:

```bash
./kan --db "$DB_PATH"
```

### C1. Project scope renders immediate children only
- In project board, verify nested descendants are not flattened into the same scope.
- Expected visible at project scope: top-level branch(es) + top-level task(s).
- Result: __________
- Evidence (screenshot/log): _____________________________

### C2. Path line is visible above board
- Expected: `path: <project>` at project scope.
- Result: __________
- Evidence: ______________________________________________

### C3. Focus drill-down (`f`) moves one level at a time
- On branch: `f` -> board shows immediate branch children.
- On phase: `f` -> board shows immediate phase children.
- On subphase: `f` -> board shows immediate subphase children.
- `F` returns to broader scope/full board.
- Result: __________
- Evidence: ______________________________________________

### C4. Enter/info behavior on hierarchy items
- Press `enter` (or `i`) on branch/phase/subphase.
- Expected: info modal opens with metadata and supports `f` for scope drill-down.
- Result: __________
- Evidence: ______________________________________________

### C5. Hierarchy markers render in card metadata
- Expected examples: `[branch|...]`, `[phase|...]`.
- Result: __________
- Evidence: ______________________________________________

### C6. Right-side notices panel appears on wide layout
- Expected panel shows:
  - attention summary,
  - selected-item context,
  - recent activity hint.
- Result: __________
- Evidence: ______________________________________________

### C7. Scoped create (`n`) follows focused level
- From focused branch/phase/subphase boards, press `n` and create one item each.
- Expected: each new item is created as a child of that focused scope (not project root).
- From focused task board, press `n`.
- Expected: new item is created as a direct subtask of that task.
- Result: __________
- Evidence: ______________________________________________

### C8. Focus no-op on leaf
- On a task/subtask with no children, press `f`.
- Expected: no navigation change and no empty focused board.
- Result: __________
- Evidence: ______________________________________________

## 7. Section D: HTTP/MCP Parity + Guardrails

### D1. `capture_state` parity hash MCP vs HTTP
- Expected: same `state_hash` for same scope and unchanged state.
- Result: __________
- Evidence: ______________________________________________

### D2. Unknown scope revoke-all fails closed
- Tool: `kan.revoke_all_capability_leases` with unknown scope tuple.
- Expected: `not_found` (or equivalent fail-closed error).
- Result: __________
- Evidence: ______________________________________________

### D3. Unknown comment target fails closed
- Tool: `kan.create_comment` with non-existent target.
- Expected: `not_found` (or equivalent fail-closed error).
- Result: __________
- Evidence: ______________________________________________

### D4. `update_task` title-only preserves priority
- Expected: priority unchanged when omitted.
- Result: __________
- Evidence: ______________________________________________

## 8. Section E: MCP Tool Sweep (All Tools)

Mark each tool `PASS` when minimally validated with real call evidence.

### E1. Bootstrap/Capture/Attention
- [ ] `kan.get_bootstrap_guide`  Evidence: ____________________
- [ ] `kan.capture_state`  Evidence: ___________________________
- [ ] `kan.list_attention_items`  Evidence: ____________________
- [ ] `kan.raise_attention_item`  Evidence: ____________________
- [ ] `kan.resolve_attention_item`  Evidence: _________________

### E2. Projects
- [ ] `kan.list_projects`  Evidence: ___________________________
- [ ] `kan.create_project`  Evidence: _________________________
- [ ] `kan.update_project`  Evidence: _________________________

### E3. Tasks/Hierarchy/Search
- [ ] `kan.list_tasks`  Evidence: ______________________________
- [ ] `kan.create_task`  Evidence: _____________________________
- [ ] `kan.update_task`  Evidence: _____________________________
- [ ] `kan.move_task`  Evidence: _______________________________
- [ ] `kan.delete_task`  Evidence: _____________________________
- [ ] `kan.restore_task`  Evidence: ____________________________
- [ ] `kan.reparent_task`  Evidence: ___________________________
- [ ] `kan.list_child_tasks`  Evidence: ________________________
- [ ] `kan.search_task_matches`  Evidence: _____________________

### E4. Change/Dependency/Kinds
- [ ] `kan.list_project_change_events`  Evidence: ______________
- [ ] `kan.get_project_dependency_rollup`  Evidence: ___________
- [ ] `kan.list_kind_definitions`  Evidence: ___________________
- [ ] `kan.upsert_kind_definition`  Evidence: __________________
- [ ] `kan.set_project_allowed_kinds`  Evidence: _______________
- [ ] `kan.list_project_allowed_kinds`  Evidence: ______________

### E5. Capability Leases
- [ ] `kan.issue_capability_lease`  Evidence: __________________
- [ ] `kan.heartbeat_capability_lease`  Evidence: ______________
- [ ] `kan.renew_capability_lease`  Evidence: __________________
- [ ] `kan.revoke_capability_lease`  Evidence: _________________
- [ ] `kan.revoke_all_capability_leases`  Evidence: ____________

### E6. Comments
- [ ] `kan.create_comment`  Evidence: __________________________
- [ ] `kan.list_comments_by_target`  Evidence: _________________

## 9. Section F: Final Quality Gates

Run and record:

```bash
just check
just ci
just test-golden
```

- `just check`: PASS / FAIL  Evidence: _________________________
- `just ci`: PASS / FAIL  Evidence: ____________________________
- `just test-golden`: PASS / FAIL  Evidence: ___________________

## 10. Final Verdict

- Overall: PASS / FAIL / BLOCKED
- Blocking issues:
  1. __________________________________________
  2. __________________________________________
- Follow-up actions:
  1. __________________________________________
  2. __________________________________________
