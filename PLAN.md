# Kan Plan

Created: 2026-02-21
Updated: 2026-02-27
Status: Execution plan locked; immediate next action is collaborative test closeout

## 1) Primary Goal

Finish `kan` as a reliable local-first planning system for human + agent collaboration, with:
1. stable TUI workflows,
2. strict mutation guardrails,
3. MCP/HTTP parity for critical flows,
4. evidence-backed validation and closeout.

## 2) Canonical Active Docs

1. `PLAN.md` (this file): execution plan and phase/task tracker.
2. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`: canonical collaborative validation evidence.
3. `COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md`: remediation requirements and checkpoints.
4. `MCP_FULL_TESTER_AGENT_RUNBOOK.md`: canonical MCP full-sweep run protocol.
5. `MCP_DOGFOODING_WORKSHEET.md`: MCP/HTTP dogfooding worksheet.
6. `PARALLEL_AGENT_RUNBOOK.md`: subagent orchestration policy.

## 3) Locked Constraints And References

### 3.1 Locked Constraints

1. Path portability rules:
   - no absolute-path export,
   - portable refs only (`root_alias` + relative paths),
   - import fails on unresolved required refs/root mappings.
2. Project linkage model stays `workspace_linked = true|false`.
3. Non-user mutations remain lease-gated and fail-closed.
4. Completion contracts remain required for completion semantics.
5. Attention/blocker escalation remains required for unresolved consensus/approval flows.

### 3.2 MCP References (Required)

1. MCP tool discovery/update:
   - https://modelcontextprotocol.io/legacy/concepts/tools#tool-discovery-and-updates
2. MCP roots/client concepts:
   - https://modelcontextprotocol.io/specification/2025-03-26/client/roots
   - https://modelcontextprotocol.io/docs/learn/client-concepts
3. MCP-Go:
   - https://github.com/mark3labs/mcp-go
   - Context7 id: `/mark3labs/mcp-go`

## 4) Global Subagent Execution Contract (Applies To Every Phase)

1. Orchestrator/integrator is the only writer for `PLAN.md` phase status and completion markers.
2. Each phase is split into parallel lanes with non-overlapping lock scopes.
3. Worker lanes run scoped checks only (`just test-pkg <pkg>`); no repo-wide gates in worker lanes.
4. Integrator runs repo-wide gates (`just check`, `just ci`, `just test-golden`) at phase integration points.
5. Worker handoff must include files changed, commands run, outcomes, acceptance checklist, and unresolved risks.
6. No lane closes without explicit acceptance evidence.

## 5) Phase Plan (Complete Execution Sequence)

## Phase 0: Collaborative Test Closeout (Immediate Next Action)

Objective:
- finish all collaborative test work and update worksheet evidence to current truth.

Tasks:
1. `P0-T01` Run remaining manual TUI validation for C4/C6/C9/C10/C11/C12/C13.
2. `P0-T02` Run archived/search/keybinding targeted checks and record PASS/FAIL/BLOCKED.
3. `P0-T03` Re-run focused MCP checks for known failures (`kan_restore_task`, `capture_state` readiness).
4. `P0-T04` Capture logging/help discoverability evidence (`./kan --help`, `./kan serve --help`, runtime log parity).
5. `P0-T05` Fill all blank checkpoints and sign-off blocks in `MCP_DOGFOODING_WORKSHEET.md`.
6. `P0-T06` Update `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md` with final evidence paths and verdict.

Parallel lane split:
1. `P0-LA` (TUI manual validation lane)
   - lock scope: `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`, `.tmp/**` evidence artifacts.
2. `P0-LB` (MCP/HTTP verification lane)
   - lock scope: `MCP_DOGFOODING_WORKSHEET.md`, `.tmp/**` protocol/evidence artifacts.
3. `P0-LC` (logging/help verification lane)
   - lock scope: `.tmp/**` logging artifacts, worksheet evidence rows for logging sections.

Exit criteria:
1. All P0 tasks have explicit PASS/FAIL/BLOCKED outcomes with evidence.
2. No blank sign-off fields remain in active worksheets.
3. Open failures are converted into explicit implementation tasks in Phase 1.

## Phase 1: Critical Remediation Fixes

Objective:
- fix currently known blockers from collaborative validation.

Tasks:
1. `P1-T01` Fix `kan_restore_task` MCP contract/guard mismatch.
2. `P1-T02` Fix logging discoverability and runtime log-sink parity gaps.
3. `P1-T03` Implement deterministic external-mutation refresh behavior in active TUI views.
4. `P1-T04` Complete notifications/notices behavior requirements (global count, quick-nav, drill-in).
5. `P1-T05` Reconcile archived/search/key policy behavior with expected UX.

Parallel lane split:
1. `P1-LA` (transport contract lane)
   - lock scope: `internal/adapters/server/mcpapi/**`, `internal/adapters/server/httpapi/**`, related tests.
2. `P1-LB` (TUI notices/refresh lane)
   - lock scope: `internal/tui/**`, related tests/golden fixtures.
3. `P1-LC` (logging/help lane)
   - lock scope: `cmd/kan/**`, `internal/adapters/server/**`, `internal/config/**`, related tests.

Exit criteria:
1. P1 defects are closed with test evidence.
2. P0 failed checks are re-run and pass or are explicitly reclassified with rationale.

## Phase 2: Contract And Data-Model Hardening

Objective:
- lock unresolved design contracts that block stable MCP/HTTP closeout.

Tasks:
1. `P2-T01` Finalize attention storage model (`table` vs embedded JSON) and migration plan.
2. `P2-T02` Finalize attention taxonomy and lifecycle/override semantics.
3. `P2-T03` Finalize pagination/cursor contract for attention and related list surfaces.
4. `P2-T04` Finalize unresolved MCP contract decisions from prior open-question sets.
5. `P2-T05` Close snapshot portability completeness gaps for collaboration-grade import/export.
6. `P2-T06` Carry unresolved override-token documentation obligations into active docs.

Parallel lane split:
1. `P2-LA` (domain/app contract lane)
   - lock scope: `internal/domain/**`, `internal/app/**`, tests.
2. `P2-LB` (storage/schema lane)
   - lock scope: `internal/adapters/storage/sqlite/**`, migration/test fixtures.
3. `P2-LC` (transport schema/docs lane)
   - lock scope: `internal/adapters/server/**`, `README.md`, `PLAN.md`, MCP worksheets.

Exit criteria:
1. Contract decisions are encoded in code/tests/docs.
2. No unresolved “open contract” placeholders remain for in-scope MVP behavior.

## Phase 3: Full Validation And Gate Pass

Objective:
- produce final evidence-backed quality pass for current scope.

Tasks:
1. `P3-T01` Run `just check`.
2. `P3-T02` Run `just ci`.
3. `P3-T03` Run `just test-golden`.
4. `P3-T04` Execute MCP full-sweep per `MCP_FULL_TESTER_AGENT_RUNBOOK.md` and capture final report.
5. `P3-T05` Re-run collaborative worksheet and dogfooding worksheet with final verdicts.

Parallel lane split:
1. `P3-LA` (automated-gates lane)
   - lock scope: test outputs and `.tmp/**` gate artifacts.
2. `P3-LB` (MCP runbook lane)
   - lock scope: MCP run artifacts/report files.
3. `P3-LC` (manual validation lane)
   - lock scope: collaborative worksheet evidence rows/screenshots.

Exit criteria:
1. Required gates pass.
2. Worksheets have final, non-blank verdicts.
3. Remaining risks are explicitly documented with owner/next step.

## Phase 4: Docs Finalization And Closeout

Objective:
- finalize accurate active docs and remove stale narrative drift.

Tasks:
1. `P4-T01` Ensure `README.md` and `AGENTS.md` reflect actual current behavior.
2. `P4-T02` Ensure `PLAN.md` statuses match worksheet/runbook evidence.
3. `P4-T03` Remove or archive stale planning/status statements that conflict with final evidence.
4. `P4-T04` Produce final closeout summary and commit sequencing plan.

Parallel lane split:
1. `P4-LA` (product docs lane)
   - lock scope: `README.md`, `CONTRIBUTING.md`.
2. `P4-LB` (process docs lane)
   - lock scope: `AGENTS.md`, `PARALLEL_AGENT_RUNBOOK.md`.
3. `P4-LC` (plan/worksheet lane)
   - lock scope: `PLAN.md`, collab worksheets/worklogs.

Exit criteria:
1. Active docs are internally consistent.
2. No stale “not implemented” statements remain for implemented behavior.

## Phase 5: Deferred Roadmap (Not In Immediate Finish Scope)

Objective:
- preserve future work without blocking finish of current scope.

Tasks:
1. `P5-T01` Advanced import/export divergence reconciliation tooling.
2. `P5-T02` Broader policy-driven tool-surface controls and template expansion.
3. `P5-T03` Multi-user/team auth-tenancy and security hardening.

Parallel lane split:
1. `P5-LA` (import/export research lane).
2. `P5-LB` (policy/template lane).
3. `P5-LC` (security/tenancy lane).

Exit criteria:
1. Roadmap items are explicitly scoped and non-blocking for current finish target.

## 6) Immediate Next Action Lock

The very next work to run is **Phase 0: Collaborative Test Closeout**.
No new feature phase should start until Phase 0 produces updated evidence and explicit task outcomes.

## 7) Definition Of Done For Current Finish Target

1. Phase 0 through Phase 4 are complete.
2. Known blocking defects from collaborative validation are closed or explicitly accepted with owner + follow-up.
3. `just check`, `just ci`, and `just test-golden` pass on the final integrated state.
4. Collaborative and dogfooding worksheets have final non-blank sign-off verdicts.
5. Active docs are accurate and mutually consistent.

## 8) Lightweight Execution Log

### 2026-02-27: PLAN Restructure For Full Phase/Lane Execution

Objective:
- convert `PLAN.md` into a complete phase/task plan with explicit parallel-lane execution for every phase.

Result:
- phases, task IDs, lane lock scopes, and exit criteria are now defined end-to-end,
- collaborative test closeout is explicitly locked as immediate next action.

Test status:
- `test_not_applicable` (docs-only change).

### 2026-02-27: Phase 0 Collaborative Closeout Run (in progress)

Objective:
- execute Phase 0 closeout checks, capture fresh evidence, and update active worksheets with explicit PASS/FAIL/BLOCKED outcomes.

Evidence root:
- `.tmp/phase0-collab-20260227_141800/`

Commands run and outcomes:
1. `just check` -> PASS (`.tmp/phase0-collab-20260227_141800/just_check.txt`)
2. `just ci` -> PASS (`.tmp/phase0-collab-20260227_141800/just_ci.txt`)
3. `just test-golden` -> PASS (`.tmp/phase0-collab-20260227_141800/just_test_golden.txt`)
4. `just build` -> PASS with environment warning (`.tmp/phase0-collab-20260227_141800/just_build.txt`)
5. `./kan --help` -> FAIL help discoverability (`.tmp/phase0-collab-20260227_141800/help_kan.txt`)
6. `./kan serve --help` -> FAIL help discoverability / startup side-effect path (`.tmp/phase0-collab-20260227_141800/help_kan_serve.txt`)
7. `curl http://127.0.0.1:18080/healthz` -> PASS (`.tmp/phase0-collab-20260227_141800/healthz.headers`, `.tmp/phase0-collab-20260227_141800/healthz.txt`)
8. `curl http://127.0.0.1:18080/readyz` -> PASS (`.tmp/phase0-collab-20260227_141800/readyz.headers`, `.tmp/phase0-collab-20260227_141800/readyz.txt`)

Focused MCP checks and outcomes:
1. `capture_state` readiness -> PASS
   - evidence: `.tmp/phase0-collab-20260227_141800/http_capture_state_project.headers`, `.tmp/phase0-collab-20260227_141800/http_capture_state_project.json`, `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`
2. `kan_restore_task` known failure repro -> FAIL (`mutation lease is required`)
   - evidence: `.tmp/phase0-collab-20260227_141800/mcp_focused_checks.md`
3. Guardrail failure matrix probes -> MIXED
   - M2.1 (missing/invalid lease tuple): PASS
   - M2.2 (scope mismatch rejection): FAIL (scope-type/scope-id mismatch accepted in one probe)
   - evidence: `.tmp/phase0-collab-20260227_141800/guardrail_failure_checks.md`
4. Completion guard probe -> PASS
   - unresolved blocker prevented `progress -> done`; transition succeeded after resolver step
   - evidence: `.tmp/phase0-collab-20260227_141800/completion_guard_check.md`
5. Resume/hash short loop probe -> PASS
   - state hash changed on mutation and returned to baseline post-cleanup
   - evidence: `.tmp/phase0-collab-20260227_141800/capture_state_hash_loop.md`

Blockers currently open:
1. CLI help discoverability remains broken (`./kan --help`, `./kan serve --help`).
2. `kan_restore_task` MCP contract mismatch remains unresolved.
3. Manual collaborative TUI checks remain pending user execution (C4/C6/C9/C10/C11/C12/C13 and archived/search/key policy checks).
4. Additional user-directed remediation requirements must be carried into fix phase:
   - first-launch config bootstrap should copy `config.example.toml` when config is missing,
   - help UX should be implemented with Charm/Fang styled output.

Current status:
- Phase 0 remains open until manual collaborative checks are completed and worksheet sign-offs are finalized.
- `MCP_DOGFOODING_WORKSHEET.md` has no blank sign-off fields; remaining blocked rows now carry explicit blocker statements and evidence paths.
- Section 0 user execution update recorded:
  - M0.2 runtime launch marked PASS by user,
  - M0.3 hierarchy IDs captured via MCP and unresolved user-action fixture item seeded,
  - early manual findings logged (C4 fail, C6 fail, C10 fail; others pending).
- Section 1 execution update recorded:
  - M1.1 (`capture_state` all required scopes) PASS,
  - M1.2 (`requires_user_action` blocker highlight in summary) PASS.
- Section 2 execution update recorded:
  - M2.1 PASS,
  - M2.2 FAIL (scope mismatch still accepted),
  - M2.3 PASS.

File edits in this checkpoint:
1. `MCP_DOGFOODING_WORKSHEET.md`
   - filled all USER NOTES blocks and final sign-off fields with explicit status + evidence references for this run.
2. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - added Section 12 Phase 0 tracker with current task statuses and blockers.
3. `PLAN.md`
   - logged command evidence, focused-check outcomes, blockers, and worksheet status for the active Phase 0 run.

Process contract update from user:
1. Continue section-by-section collaborative test walkthrough and note capture.
2. Preserve user intent with full detail in active markdown docs; normalize wording only when needed for technically correct terminology.
3. Final step of testing process will run subagents + Context7 (+ web research as needed) to propose fixes, then record proposals only after explicit user+agent consensus.

Additional restore-surface design requirement:
1. During fix-proposal phase, evaluate whether restore should be generalized (`restore` + explicit node/scope type arg) versus task-only surface, while ensuring required guardrail tuple fields and id/name gatekeeping semantics are consistently enforced.

### 2026-02-27: Remote E2EE Architecture + Roadmap Draft

Objective:
- produce a detailed roadmap for optional remote org collaboration with strict E2EE data handling while preserving local-first OSS usage.

Commands run and outcomes:
1. `rg --files -g'*.md' | sort` -> PASS (identified doc targets)
2. `sed -n '1,360p' PLAN.md` -> PASS (loaded active plan/worklog context)
3. `rg -n "export|import|snapshot|remote|tenancy|auth|sync|sqlite|postgres|file|attachment|project_roots" ...` -> PASS (collected active constraints from canonical docs)
4. Context7 lookup:
   - `resolve-library-id sqlite` -> PASS
   - `resolve-library-id postgresql` -> PASS
   - `query-docs /websites/sqlite_cli` -> PASS
   - `query-docs /websites/postgresql_current` -> PASS
5. Web-source verification:
   - SQLite (`isolation`, `wal`, `useovernet`, `backup`, `lang_vacuum`, `sessionintro`) -> PASS
   - PostgreSQL (`ddl-rowsecurity`, `sql-listen`, `sql-notify`) -> PASS
   - S3 (`Welcome`, `using-presigned-url`, `Versioning`) -> PASS
   - WebSocket RFC6455 -> PASS
   - libsodium `secretstream` -> PASS

File edits in this checkpoint:
1. `REMOTE_E2EE_ROADMAP.md`
   - added full architecture decision, live-update model, file/blob strategy, OSS execution phases, hosted-service roadmap, and aligned references.
2. `PLAN.md`
   - added this checkpoint with command/test evidence.

Test status:
- `test_not_applicable` (docs-only changes; no code/test behavior modified).

### 2026-02-28: Rename Wave Kickoff (hakoll project + koll command)

Objective:
- execute full no-compat rename from `kan` to `hakoll` (project/repo identity) and `koll` (runtime command/binary), then revalidate parity with `just check` and `just ci`.

Commands run and outcomes:
1. `git status --short` -> PASS (`README.md` staged change only).
2. `git add README.md && git commit -m "docs: note hakoll naming origin in README"` -> PASS (`aadf95c`).
3. Context7:
   - `resolve-library-id` for Go (`/golang/go`) -> PASS.
   - `query-docs` on module path/import rename implications -> PASS.
4. Orchestration scan:
   - `pwd && ls -la` -> PASS.
   - `sed -n '1,220p' Justfile` -> PASS.
   - `rg -n "\bkan\b|cmd/kan|/kan\b|kan\." -S --hidden --glob '!**/.git/**'` -> PASS.
5. Spawned three subagents (code/runtime, tests/fixtures, docs/automation) and collected inventories -> PASS.

File edits in this checkpoint:
1. `PLAN.md`
   - added rename-wave kickoff checkpoint and command evidence.

Test status:
- `test_not_applicable` (planning/orchestration checkpoint; implementation in progress).

### 2026-02-28: Rename Wave Implementation Complete (No Compatibility Layer)

Objective:
- complete the all-at-once rename from `kan` to `hakoll` (project/repo/module identity) and `koll` (runtime command/binary/tool namespace), with no compatibility aliases.

Subagent lane execution and outcomes:
1. `R1-core-cli` (core CLI/module/build/path surfaces) -> PASS
   - scope delivered: `go.mod`, `cmd/koll/**` (from `cmd/kan/**`), `internal/platform/**`, `internal/config/**`, `internal/tui/**`, `Justfile`, `.goreleaser.yml`, `.github/workflows/ci.yml`, `.gitignore`, `config.example.toml`, `cmd/headerlab/main.go`.
2. `R2-runtime-mcp` (server/app/domain/storage surfaces) -> PASS
   - scope delivered: `internal/adapters/server/**`, `internal/adapters/storage/sqlite/**`, `internal/app/**`, `internal/domain/**`.
3. `R3-docs-ops` (docs/runbooks/worksheets/tapes) -> PASS
   - scope delivered: `README.md`, `AGENTS.md`, `MCP_*`, `COLLAB*`, `REMOTE_E2EE_ROADMAP.md`, `vhs/**`.

Commands run and outcomes:
1. Integrator gate run `just check` -> FAIL (verify-sources pathspec before staging renamed `cmd/koll/*` files).
2. Context7 re-consult (Go rename/staging implications) -> PASS.
3. Staged rename paths and reran `just check` -> FAIL (`gofmt required for cmd/koll/main.go`).
4. Context7 re-consult (gofmt workflow) -> PASS.
5. `just fmt` -> PASS.
6. `just check` -> PASS.
7. `just ci` -> PASS.
8. Final cleanup of lingering test sample tokens (`kan` -> `hakoll`) in:
   - `internal/adapters/storage/sqlite/repo_test.go`
   - `internal/app/service_test.go`
   - `internal/adapters/server/mcpapi/handler_test.go`
9. Post-cleanup verification:
   - `just check` -> PASS.
   - `just ci` -> PASS.

File edits in this checkpoint:
1. `PLAN.md`
   - added full rename implementation checkpoint with subagent evidence and gate outcomes.

Test status:
- `just check` PASS
- `just ci` PASS

### 2026-02-28: Post-Integration Docs Correction

Objective:
- resolve a docs regression introduced during rename sweep where absolute local links in the remote roadmap pointed at a non-existent workspace path.

Commands run and outcomes:
1. `rg -n "/Users/.*/personal/hakoll|/Users/.*/personal/kan" REMOTE_E2EE_ROADMAP.md ...` -> PASS (identified hardcoded absolute links).
2. Patched `REMOTE_E2EE_ROADMAP.md` links to repo-relative paths -> PASS.

File edits in this checkpoint:
1. `REMOTE_E2EE_ROADMAP.md`
   - replaced hardcoded absolute paths with repo-relative markdown links.
2. `PLAN.md`
   - recorded post-integration docs correction checkpoint.

Test status:
- `test_not_applicable` (docs-only correction; no runtime/code behavior change).

### 2026-02-28: Phase 0 Section 2 Post-Fix Rerun (in progress, blocker persists)

Objective:
- rerun Section 2 guardrail checks after app-layer + scope-mapping fixes, then update worksheets/evidence before deciding next remediation lane.

Commands/tools run and outcomes:
1. `just test-pkg ./internal/app` -> PASS (`ok ... internal/app (cached)`).
2. `kan_create_task` probe (`actor_type=agent`, missing tuple) -> PASS expected failure (`invalid_request` requiring guard tuple fields).
3. `kan_create_task` probe (`actor_type=agent` + malformed lease token) -> PASS expected failure (`guardrail_failed ... mutation lease is invalid`).
4. `kan_issue_capability_lease` on fixture project -> PASS (issued instance `2c83f1cb-fba9-40e0-b274-84705dc5e73d`).
5. `kan_raise_attention_item` scope-mismatch probe (`scope_type=task`, `scope_id=<project_id>`) -> FAIL (unexpected acceptance; persisted `5956394b-f73a-4522-8530-ec53ec00082c`).
6. `kan_create_task` cross-project mismatch probe using fixture-scoped lease -> PASS expected failure (`guardrail_failed ... mutation lease is invalid`).
7. M2.3 completion contract probe:
   - created task `d6fe3b4a-369c-4212-b049-90630e71fc1f` in progress,
   - raised blocker `a264b6fd-15bc-427f-9972-f6f5273807ae`,
   - move to done blocked (expected),
   - resolve blocker + retry move -> PASS.
8. Cleanup:
   - resolved mismatch probe item `5956394b-f73a-4522-8530-ec53ec00082c`,
   - hard-deleted probe task `d6fe3b4a-369c-4212-b049-90630e71fc1f`,
   - revoked lease `2c83f1cb-fba9-40e0-b274-84705dc5e73d`.
9. Runtime freshness check -> FLAGGED:
   - `ls -l ./kan internal/app/attention_capture.go internal/app/kind_capability.go`
   - binary mtime `2026-02-27 14:40` predates modified source mtimes (`17:13`, `17:16`), so the rerun may have exercised a stale running server.
10. Explorer subagent root-cause pass -> COMPLETED (no edits):
   - call-chain traced from MCP handler to `Service.RaiseAttentionItem` and `validateCapabilityScopeTuple`,
   - recommended next step: restart/reload runtime and re-run M2.2 before additional code edits; if still failing, add deterministic tuple guard.
11. `just build` -> PASS with known non-fatal Go stat-cache warning; rebuilt binary mtime now `2026-02-27 17:34`.

Result summary:
1. M2.1 PASS.
2. M2.2 FAIL (still open; fail-closed behavior not enforced for `scope_type=task` + project ID).
3. M2.3 PASS.

File edits in this checkpoint:
1. `.tmp/phase0-collab-20260227_141800/manual/section2_guardrail_evidence_20260227.md`
   - appended 2026-02-28 rerun with IDs, outcomes, and cleanup.
2. `MCP_DOGFOODING_WORKSHEET.md`
   - updated M2.1/M2.2/M2.3 notes and final sign-off notes to reflect post-fix rerun outcomes.
3. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - updated Section 12.8 with explicit 2026-02-28 rerun status and persisted M2.2 blocker.

Current status:
- Phase 0 remains open; Section 2 cannot be closed due to persistent M2.2 failure.
- M2.2 runtime result is currently confounded by stale-binary risk and needs one clean rerun on a refreshed server process.
- Binary is refreshed locally; next required action is restarting `./kan serve ...` and rerunning M2.2 immediately.
- Per section-by-section policy, next step is targeted remediation of M2.2 before advancing to later sections.

### 2026-02-28: Section 2 Post-Restart Recheck + CI Gate

Objective:
- verify M2.2 on a freshly restarted runtime and confirm repo-level gate status before deciding commit readiness.

Commands/tools run and outcomes:
1. `kan_raise_attention_item` mismatch probe (`scope_type=task`, `scope_id=<project_id>`) -> PASS expected fail-closed (`not_found`, no persistence).
2. `kan_issue_capability_lease` + cross-project guarded mutation probe -> PASS expected fail-closed (`mutation lease is invalid`), lease revoked.
3. `kan_list_attention_items` open project scope check -> PASS (no unexpected open items after probe).
4. `just test-pkg ./internal/app` -> PASS.
5. `just ci` -> PASS (exit 0; coverage lines still above policy thresholds).

Result summary:
1. M2.2 fail-closed behavior is now confirmed after restart.
2. Section 2 gate status: M2.1 PASS, M2.2 PASS, M2.3 PASS.
3. Phase 0 overall remains open due to separate known blockers (help/first-launch/restore + pending manual collaborative TUI sections).

File edits in this checkpoint:
1. `.tmp/phase0-collab-20260227_141800/manual/section2_guardrail_evidence_20260227.md`
   - appended post-restart verification outcome.
2. `.tmp/phase0-collab-20260227_141800/manual/section2_post_restart_20260228.md`
   - added focused post-restart probe transcript and gate outcomes.
3. `MCP_DOGFOODING_WORKSHEET.md`
   - updated M2.2 to PASS and adjusted final blocking list accordingly.
4. `COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md`
   - updated Section 12.8 with post-restart M2.2 PASS evidence.

### 2026-02-27: AGENTS Flow Update (Section-by-Section Fix-As-We-Go)

Objective:
- align repository agent policy with user-directed collaborative flow:
  - test one section,
  - fix findings immediately,
  - revalidate section before moving forward.

Commands run and outcomes:
1. `rg -n "Testing Guidelines|Parallel/Subagent Mode|Temporary Next-Step Directive|..." AGENTS.md` -> PASS
2. `sed -n '1,260p' AGENTS.md` + `sed -n '260,520p' AGENTS.md` -> PASS
3. Updated `AGENTS.md` to lock section-by-section remediation loop and consensus-before-implementation workflow.
4. `rg -n "Locked execution flow|section-by-section remediation|..." AGENTS.md` -> PASS (verified insertions)

File edits in this checkpoint:
1. `AGENTS.md`
   - added temporary-phase locked execution flow for section-by-section remediation with subagent/context7/web research + consensus + scoped tests + section rerun.
   - added testing-guideline rules preventing advancement before section revalidation.

Test status:
- `test_not_applicable` (process/docs-only change).

### 2026-02-27: Restore Task Guardrail Contract Investigation

Objective:
- trace `kan_restore_task` (`kan.restore_task`) guardrail failure (`mutation lease is required`) across MCP registration, common adapter contracts, and app guard enforcement.

Commands run and outcomes:
1. `rg -n "restore_task|kan_restore_task|mutation lease is required|lease"` -> PASS (identified MCP/tool + guardrail references)
2. `rg -n "delete_task|move_task|update_task|actor"` -> PASS (identified tuple-capable mutation tools for comparison)
3. `nl -ba internal/adapters/server/mcpapi/extended_tools.go` (scoped ranges) -> PASS
4. `nl -ba internal/adapters/server/common/mcp_surface.go` -> PASS
5. `nl -ba internal/adapters/server/common/app_service_adapter_mcp.go` (scoped ranges) -> PASS
6. `nl -ba internal/app/service.go` + `internal/app/kind_capability.go` (scoped ranges) -> PASS
7. `nl -ba internal/adapters/server/common/app_service_adapter.go` + `internal/adapters/server/mcpapi/handler.go` -> PASS
8. `nl -ba internal/domain/errors.go` + `internal/domain/task.go` -> PASS
9. `nl -ba Justfile` -> PASS (startup recipe review requirement)

Findings summary:
1. `kan.restore_task` MCP registration only accepts `task_id` and calls `tasks.RestoreTask(ctx, taskID)` with no actor/lease tuple.
2. Common task-service contract and adapter method signature for restore accept only `task_id`, unlike update/move/delete request structs that include `ActorLeaseTuple`.
3. App `RestoreTask` still enforces mutation guardrails using persisted `task.UpdatedByType`; when that actor type is non-user and no guard tuple is attached to context, enforcement returns `domain.ErrMutationLeaseRequired`.
4. Error mapping converts this to MCP-visible `guardrail_failed: ... mutation lease is required`.

File edits in this checkpoint:
1. `PLAN.md`
   - added investigation worklog entry with command evidence and root-cause chain.

Test status:
- `test_not_applicable` (investigation/docs-only; no code changes).

### 2026-02-27: Remote Roadmap Update (HTTP-Only Runtime + Fang/Cobra Plan)

Objective:
- update remote roadmap with newly agreed runtime decisions:
  - HTTP-only MCP for now,
  - `kan` launches TUI with local-server ensure/reuse behavior,
  - default local endpoint `127.0.0.1:5437` with auto-fallback,
  - user endpoint selection in CLI/TUI,
  - Fang/Cobra migration,
  - phase/lane plan for parallel subagents.

Commands run and outcomes:
1. `Context7 resolve-library-id fang` -> PASS
2. `Context7 resolve-library-id cobra` -> PASS
3. `Context7 query-docs /charmbracelet/fang` -> PASS
4. `Context7 query-docs /spf13/cobra` -> PASS
5. Spawned explorer subagents for:
   - serve/runtime lifecycle verification (PASS),
   - current help/UX friction and recommendations (PASS)
6. `sed -n '1,320p' REMOTE_E2EE_ROADMAP.md` -> PASS (loaded current roadmap prior to patching)
7. `Context7 resolve-library-id mcp-go` + `query-docs /mark3labs/mcp-go` -> PASS (validated transport suitability/limits for HTTP-first decision)

File edits in this checkpoint:
1. `REMOTE_E2EE_ROADMAP.md`
   - added locked 2026-02-27 runtime/transport decisions,
   - added local runtime modes, endpoint fallback policy, and supervisor behavior,
   - added `R-CLI` phase for Fang/Cobra + server orchestration,
   - added explicit parallel lane map for subagent execution,
   - updated milestones and references.
2. `PLAN.md`
   - added this checkpoint with evidence and outcomes.

Test status:
- `test_not_applicable` (docs-only changes; no code/test behavior modified).

### 2026-02-28: R-CLI-FANG-01 Integrated (Fang/Cobra CLI Migration)

Objective:
- replace stdlib `flag` CLI parsing in `cmd/koll` with Fang/Cobra, improve help/error UX, and remove orphaned parser code paths.

Commands/tools run and outcomes:
1. Context7 `resolve-library-id` + `query-docs` for `/charmbracelet/fang` and `/spf13/cobra` -> PASS (captured Execute/RunE/help/error patterns).
2. Spawned worker lane `R-CLI-FANG-01` (lock scope: `cmd/koll/**`, `go.mod`, `go.sum`) -> PASS.
3. Worker lane package check loop:
   - `just test-pkg ./cmd/koll` baseline -> PASS
   - post-migration `just test-pkg ./cmd/koll` -> FAIL (missing `go.sum` entry)
   - dependency fetch for missing checksum + `just fmt` + rerun `just test-pkg ./cmd/koll` -> PASS
4. Integrator verification:
   - `just check` -> PASS
   - `just ci` -> PASS
5. Runtime smoke:
   - `./koll --help` -> PASS (styled root help)
   - `./koll serve --help` -> PASS (styled subcommand help)
   - `./koll --badflag` -> PASS (styled error + guidance + existing `error: ...` line)

File edits in this checkpoint:
1. `cmd/koll/main.go`
   - migrated to Cobra command tree executed by Fang;
   - removed stdlib `flag` parser flow and related orphaned helpers;
   - preserved `tui` default, `serve`, `export`, `import`, and `paths` command behavior.
2. `cmd/koll/main_test.go`
   - updated/added help coverage for Fang/Cobra output behavior.
3. `go.mod`, `go.sum`
   - added Fang/Cobra dependencies and required checksum entries.

Current status:
- CLI adapter migration is integrated locally and gated (`just check` + `just ci` passing).
- No remaining orphaned stdlib `flag` parser path in `cmd/koll/main.go`.

### 2026-02-28: Fang Output Refinement (Paths + Error Surface)

Objective:
- ensure command output/error surfaces are Fang-styled where practical, including `koll paths` presentation and removal of duplicate plain error output.

Commands run and outcomes:
1. Context7 `query-docs /charmbracelet/fang` (output/error handler styling confirmation) -> PASS.
2. `go doc github.com/charmbracelet/fang` + `go doc -all github.com/charmbracelet/fang` -> PASS (validated available APIs/Styles surface).
3. `just fmt && just test-pkg ./cmd/koll` -> PASS.
4. `just ci` -> PASS.
5. Runtime smoke:
   - `./koll paths` -> PASS (styled titled key/value output).
   - `./koll --badflag` -> PASS (Fang-styled error block, no extra plain `error:` suffix).

File edits in this checkpoint:
1. `cmd/koll/main.go`
   - removed duplicate top-level plain error print in `main`;
   - added `writePathsOutput` using Fang default color scheme + lipgloss rendering;
   - routed `paths` command through styled renderer.
2. `cmd/koll/main_test.go`
   - updated `TestRunPathsCommand` assertions for titled/styled paths output semantics.

Current status:
- `paths` output and CLI error surface are now aligned with Fang-style rendering expectations.
