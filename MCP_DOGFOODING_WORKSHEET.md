# MCP + HTTP Dogfooding Worksheet (Active Wave)

Use this worksheet for user+agent validation of active-wave MCP/HTTP behavior.

Pass/fail rule for all `USER NOTES` blocks:
- `Pass/Fail` must be set to exactly one of `pass`, `fail`, or `blocked`.
- Blank `Pass/Fail` values are invalid and block sign-off.
- `blocked` requires the exact blocker and required user action.

## 0) Environment + Preflight

### 0.1 Build and serve-surface preflight

Actions:

1. Build the binary.
2. Check whether `serve` mode is present.
3. Record the exact serve command/flags available in your build.

Commands:

```bash
just build
./kan serve --help
```

Expected:

- Build succeeds.
- `serve` help output is available.
- Help output documents HTTP and MCP endpoint options (or equivalent).

### USER NOTES M0.1-N1

- Pass/Fail (set one: pass|fail|blocked): fail
- Evidence (required): `.tmp/phase0-collab-20260227_141800/just_build.txt`, `.tmp/phase0-collab-20260227_141800/help_kan.txt`, `.tmp/phase0-collab-20260227_141800/help_kan_serve.txt`, `.tmp/phase0-collab-20260227_141800/phase0_preflight_summary.md`
- Notes: Build succeeded, but help-surface validation failed. `./kan --help` returned `error: flag: help requested`, and `./kan serve --help` entered startup/open-db flow instead of printing stable help. Remediation requirement: implement a Charm/Fang-based help surface for usable, styled CLI help output.

---

### 0.2 Start isolated dogfood runtime

Actions:

1. Start `kan` in a fresh DB/config pair.
2. Bind locally only.
3. Keep terminal output for evidence capture.

Command template (adjust flags to match `--help` output):

```bash
KAN_DB_PATH=/tmp/kan-mcp-dogfood.db \
KAN_CONFIG=/tmp/kan-mcp-dogfood.toml \
./kan serve --http 127.0.0.1:8080 --api-endpoint /api/v1 --mcp-endpoint /mcp
```

Expected:

- Service starts without panic.
- Startup logs include bound address and endpoint paths.
- Runtime errors are emitted as structured logs.

### USER NOTES M0.2-N1

- Pass/Fail (set one: pass|fail|blocked): blocked
- Evidence (required): `.tmp/phase0-collab-20260227_141800/port_18080_listener.txt`, `.tmp/phase0-collab-20260227_141800/healthz.headers`, `.tmp/phase0-collab-20260227_141800/healthz.txt`, `.tmp/phase0-collab-20260227_141800/readyz.headers`, `.tmp/phase0-collab-20260227_141800/readyz.txt`
- Notes: Active runtime is up and healthy on `127.0.0.1:18080`, but strict isolated dogfood DB/config launch for this worksheet step still needs an explicit fresh-runtime user run.

---

### 0.3 Seed hierarchy fixture for level-scoped checks

Actions:

1. In another terminal, run the TUI against the same DB/config.
2. Create one project with this hierarchy:
   - `branch -> phase -> subphase -> task -> subtask`
3. Add at least one open blocker/approval-required item in the same branch.
4. Capture IDs needed for API/tool calls.

Command:

```bash
KAN_DB_PATH=/tmp/kan-mcp-dogfood.db KAN_CONFIG=/tmp/kan-mcp-dogfood.toml just run
```

Expected:

- Fixture exists with all required scope levels.
- At least one unresolved blocker/user-action case exists for validation.

### USER NOTES M0.3-N1

- Pass/Fail (set one: pass|fail|blocked): blocked
- Evidence (required): `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md`
- Notes: Hierarchy fixture creation in TUI (`branch -> phase -> subphase -> task -> subtask`) and user-action blocker seeding remain pending manual user execution.

---

## 1) `capture_state` Flows

### 1.1 Scope-by-scope `capture_state` responses

Actions:

1. Call `capture_state` for each scope level:
   - `project`, `branch`, `phase`, `subphase`, `task`, `subtask`.
2. Use equivalent REST endpoint or MCP tool call.
3. Record one response per level.

REST example payload:

```json
{
  "actor_type": "user",
  "project_id": "<project_id>",
  "branch_id": "<branch_id>",
  "scope_type": "subphase",
  "scope_id": "<subphase_id>",
  "view": "summary"
}
```

Expected:

- Every level returns a deterministic summary-first bundle.
- Each response includes scope path and resume-oriented context.
- Follow-up pointers/cursors are present for deeper calls.

### USER NOTES M1.1-N1

- Pass/Fail (set one: pass|fail|blocked): blocked
- Evidence (required): `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`, `.tmp/phase0-collab-20260227_141800/http_capture_state_project.headers`, `.tmp/phase0-collab-20260227_141800/http_capture_state_project.json`
- Notes: Project-level capture_state readiness is confirmed (deterministic hash + resume hints), but scope-by-scope coverage for `branch/phase/subphase/task/subtask` is pending seeded hierarchy.

---

### 1.2 `capture_state` includes blocker/user-action highlights

Actions:

1. Ensure at least one unresolved blocker requiring user action exists.
2. Call `capture_state` at the affected scope.
3. Verify blocker visibility in summary output.

Expected:

- Response surfaces unresolved blocker/approval/consensus items.
- `requires_user_action` items are visible in the summary response.

### USER NOTES M1.2-N1

- Pass/Fail (set one: pass|fail|blocked): blocked
- Evidence (required): `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md`
- Notes: Requires unresolved blocker fixture tied to real hierarchy scope before validation of `requires_user_action` highlights in capture summaries.

---

## 2) Guardrail Failure Checks (Expected Errors)

### 2.1 Non-user mutation without valid lease tuple

Actions:

1. Issue a non-user mutation call without lease tuple fields.
2. Repeat with malformed lease token.

Expected:

- Call fails closed.
- Error response is structured and explains the guardrail failure.
- No persistence side effects occur.

### USER NOTES M2.1-N1

- Pass/Fail (set one: pass|fail|blocked): pass
- Evidence (required): `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`, `.tmp/phase0-collab-20260227_141800/guardrail_failure_checks.md`
- Notes: Non-user mutations without a valid lease tuple fail closed as expected, including no-tuple (`mutation lease is required`) and malformed-token (`mutation lease is invalid`) cases.

---

### 2.2 Scope mismatch/ambiguity rejection

Actions:

1. Call mutation with mismatched `scope_type` and `scope_id`.
2. Call mutation with mismatched project/branch tuple.

Expected:

- Calls fail closed with deterministic error code/message.
- Errors clearly identify scope mismatch and remediation direction.

### USER NOTES M2.2-N1

- Pass/Fail (set one: pass|fail|blocked): fail
- Evidence (required): `.tmp/phase0-collab-20260227_141800/guardrail_failure_checks.md`
- Notes: Mixed outcome. Cross-project lease mismatch correctly failed closed, but a scope mismatch probe (`scope_type=task` with project-id scope_id) was accepted and persisted, violating deterministic mismatch rejection expectation.

---

### 2.3 Completion guard with unresolved blockers

Actions:

1. Attempt `progress -> done` transition while unresolved blocking attention exists.
2. Resolve blocker.
3. Retry transition.

Expected:

- First transition is blocked with explicit reason.
- Transition succeeds only after blocker is resolved.

### USER NOTES M2.3-N1

- Pass/Fail (set one: pass|fail|blocked): pass
- Evidence (required): `.tmp/phase0-collab-20260227_141800/completion_guard_check.md`
- Notes: Transition to done was blocked while unresolved blocker attention existed, and succeeded only after blocker resolution.

---

## 3) Blocker/User-Action Panel + Warning Verification

### 3.1 TUI warning indicator + compact panel

Actions:

1. Open TUI against the same dataset.
2. Navigate to scope with unresolved blocker/user-action entries.
3. Verify warning indicator and compact panel visibility.

Expected:

- Rows with unresolved blocker/user-action items show warning state.
- Compact panel reflects current-scope unresolved items.
- Panel updates when scope focus changes.

### USER NOTES M3.1-N1

- Pass/Fail (set one: pass|fail|blocked): blocked
- Evidence (required): `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md`
- Notes: Requires user-driven TUI validation of warning indicators and compact unresolved panel behavior.

---

### 3.2 Resolve flow parity across transport + TUI

Actions:

1. Resolve an attention/blocker item through HTTP/MCP.
2. Refresh TUI scope.
3. Confirm UI and transport output agree.

Expected:

- Resolved item disappears from unresolved panel/list.
- `capture_state` and TUI panel show matching unresolved counts.

### USER NOTES M3.2-N1

- Pass/Fail (set one: pass|fail|blocked): blocked
- Evidence (required): `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md`
- Notes: Requires combined transport mutation + TUI refresh parity verification with interactive UI evidence.

---

## 4) Level-Scoped Search/Filter Behavior

### 4.1 Scope filter coverage across all levels

Actions:

1. Run search/filter at each scope level:
   - `project`, `branch`, `phase`, `subphase`, `task`, `subtask`.
2. Use one query that should match descendants and one that should not.

Expected:

- Scope selector supports all required levels.
- Results honor selected scope boundaries.
- Widening scope increases candidate set deterministically.

### USER NOTES M4.1-N1

- Pass/Fail (set one: pass|fail|blocked): blocked
- Evidence (required): `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md`
- Notes: Scope-filter behavior across all hierarchy levels depends on seeded hierarchy and interactive query validation.

---

### 4.2 Search/filter parity between API/MCP and TUI

Actions:

1. Run the same query/scope tuple in transport and TUI.
2. Compare returned/matched item IDs.

Expected:

- Query behavior is consistent across surfaces.
- Any known intentional differences are documented in evidence.

### USER NOTES M4.2-N1

- Pass/Fail (set one: pass|fail|blocked): blocked
- Evidence (required): `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md`
- Notes: API/MCP versus TUI query parity still requires manual cross-surface comparison once hierarchy fixture is in place.

---

## 5) Resume-After-Context-Loss Scenarios

### 5.1 Agent reorientation after context loss

Actions:

1. Simulate context loss (new session or cleared conversation state).
2. Call `capture_state` for current working scope.
3. Use only returned summary + resume hints to continue a pending task.

Expected:

- Agent can restate current goal, blockers, and next action without full-history replay.
- Resume hints are sufficient to request the next deterministic follow-up call.

### USER NOTES M5.1-N1

- Pass/Fail (set one: pass|fail|blocked): pass
- Evidence (required): `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`
- Notes: capture_state summary includes sufficient goal/scope context and explicit resume hints (`list_attention_items`, `list_project_change_events`, `list_child_tasks`) to continue work after context loss.

---

### 5.2 Cursor/hash stability across short resume loops

Actions:

1. Capture state.
2. Perform one mutation.
3. Capture state again.
4. Verify state hash/event pointers moved predictably.

Expected:

- `last_change_event_id`/state-tracking fields update monotonically.
- No stale snapshot or duplicated resume pointer behavior appears.

### USER NOTES M5.2-N1

- Pass/Fail (set one: pass|fail|blocked): pass
- Evidence (required): `.tmp/phase0-collab-20260227_141800/capture_state_hash_loop.md`
- Notes: State hash changed predictably after mutation and returned to baseline after cleanup, demonstrating stable short-loop resume behavior.

---

## Final Sign-off

- Overall result (set one): `fail`
- Blocking defects: help discoverability path failures (`./kan --help`, `./kan serve --help`) plus required Charm/Fang help redesign, missing first-launch config bootstrap requirement (copy `config.example.toml` when config is absent), `kan_restore_task` guardrail contract mismatch, scope-mismatch rejection gap in M2.2, and unresolved manual TUI/fixture-dependent sections.
- Non-blocking defects: environment warning during `just build` (`go` stat-cache write permission warning) did not fail build.
- Required user actions before next wave checkpoint: run manual TUI hierarchy/notification/search/keybinding checks and attach evidence paths under `.tmp/phase0-collab-20260227_141800/manual/`.
- Tester(s): Codex (agent) + evanschultz (user pending manual steps)
- Date (`YYYY-MM-DD`): 2026-02-27

### USER NOTES MF.1-N1

- Pass/Fail (set one: pass|fail|blocked): fail
- Evidence (required): `.tmp/phase0-collab-20260227_141800/phase0_preflight_summary.md`, `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`, `.tmp/phase0-collab-20260227_141800/guardrail_failure_checks.md`, `.tmp/phase0-collab-20260227_141800/completion_guard_check.md`, `.tmp/phase0-collab-20260227_141800/phase0_manual_steps.md`
- Notes: Agent-completable checks were executed and recorded; final wave sign-off is blocked until user-driven collaborative TUI checks are completed and open blocking defects are resolved.
