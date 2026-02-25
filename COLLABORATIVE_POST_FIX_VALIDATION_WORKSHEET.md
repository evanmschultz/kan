# Collaborative Post-Fix Validation Worksheet

Date: __________  
Tester agent/session: __________  
User: __________  
Artifact dir: `.tmp/collab-post-fix-<timestamp>/`

## 1) Objective

Validate that all remediations from `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md` are complete and that previously failing collaborative E2E findings are resolved.

Rule: mark every step `PASS` / `FAIL` / `BLOCKED` with evidence path.

## 2) Preconditions

1. Implementation changes merged locally.
2. Full gates already passed:
   - `just check`
   - `just ci`
   - `just test-golden`
3. Server/TUI launched against clean validation DB snapshot.

Record command/evidence:
- gates evidence: ______________________________________
- runtime command: _____________________________________
- health check evidence: ________________________________

## 3) Gatekeeping + Identity Regression

1. Agent lease tuple cannot submit `actor_type=user`.
2. Cross-project/scope mutation remains fail-closed.
3. User and agent attribution fields are distinct and correct.
4. Spawned worker scope enforcement still holds.

- Result: PASS / FAIL / BLOCKED
- Evidence: _____________________________________________

## 4) TUI Regression Sweep (Previously Failing Areas)

1. C4: contextual `esc` back behavior + info modal return origin.
2. C6: notifications panel present with level scope + global count + quick-nav + warning/error surfacing.
3. C9: typing keys preserved in text inputs; selection/copy key redesign works; due picker include-time + fuzzy date works.
4. C10: emoji input support + icon behavior is defined and visible.
5. C11: `?` help available in all modals with correct scoped shortcuts.
6. C12: branch path flow, focused-empty phase/subphase creation behavior, `new-branch` block under focus.
7. C13: archive/restore/project lifecycle semantics and search archived behavior (decoupled from global visibility toggle as expected).

- Result: PASS / FAIL / BLOCKED
- Evidence: _____________________________________________

## 5) Archived/Search/Keybinding Targeted Checks

1. Archived projects hidden by default where expected; explicit picker visibility control present.
2. `.` quick action restore works.
3. Archive action key policy matches expected behavior (no plain `a` archive trigger).
4. Search archived filter returns archived matches when selected without requiring unrelated global toggle coupling.

- Result: PASS / FAIL / BLOCKED
- Evidence: _____________________________________________

## 6) Logging + Notifications Quality

1. MCP/runtime operations produce useful logs with `charmbracelet/log`.
2. Important warnings/errors bubble to notifications panel and quick info modal.
3. No silent failures for key operations.

- Result: PASS / FAIL / BLOCKED
- Evidence: _____________________________________________

## 7) Final Post-Fix Verdict

- Overall: PASS / FAIL / BLOCKED
- Remaining blockers:
  1. __________________________________________
  2. __________________________________________
- Follow-up actions:
  1. __________________________________________
  2. __________________________________________
