# MCP Design And Plan (Phase 11 Execution Wave 1)

Date: 2026-02-24  
Status: Active orchestrator execution/worklog hub (wave in progress)

## 1) Purpose

This file is the dedicated Phase 11 design and execution hub for MCP + HTTP.

Goals:
- reconcile all pre-MCP consensus docs with current code reality;
- define and execute a practical architecture for HTTP-first + `stateless mcp-go`;
- identify risks/warnings that could cause missed work or task-loss across docs/agents;
- run the wave with lock-safe parallel lanes and evidence-backed checkpoints.

Scope guard for this file:
- this is the active execution tracker for the current pre-MCP closeout + MCP wave;
- lane planning, lock ownership, checkpoint evidence, blockers, and decisions are tracked here;
- roadmap-only items stay explicitly deferred.

## 2) Inputs Reconciled

Primary docs reviewed:
- `PLAN.md` (planning/architecture + Phase 10/11 contract sections; worklog ignored except failure patterns)
- `PRE_PHASE11_CLOSEOUT_DISCUSSION.md`
- `PRE_MCP_EXECUTION_WAVES.md`
- `PRE_MCP_CONSENSUS.md`
- `Pre_MCP_User_NOTES.md`
- `USER_RESPONSE.md`
- `response.md`

Current code reviewed:
- `cmd/kan/main.go`
- `internal/app/service.go`
- `internal/app/ports.go`
- `internal/app/kind_capability.go`
- `internal/app/snapshot.go`
- `internal/domain/*.go` (kind, capability, task, project, comment, errors, workitem)
- `internal/adapters/storage/sqlite/repo.go`
- `internal/config/config.go`
- `README.md`
- `AGENTS.md`

Context7 references reviewed:
- `mcp-go` transport/tool registration patterns
- MCP specification (tools capability and `list_changed` behavior)
- Go standard library `context` patterns (used here for server cancellation planning)

## 2.1) Orchestrator Activation (2026-02-24)

Wave launch prompt lock:
- finish all remaining pre-MCP non-roadmap work;
- implement only explicitly locked MCP/HTTP features with HTTP-first and stateless MCP;
- preserve strict guardrails (lease tuple enforcement, id/name/scope checks, fail-closed ambiguity, completion contract blocking, user-action blocker surfacing);
- keep user-in-loop checkpoints and produce dogfooding-ready worksheets.

Current mode for this file:
- active orchestrator control plane;
- single source for MCP-wave objective, lanes, lock map, checkpoint evidence, blockers, and decisions.

## 2.2) Locked Decisions Revalidated Before Execution

The following are treated as locked and in-scope for this wave:
- strict task/state system (not vector memory surrogate);
- level-scoped operations must support:
  - `project|branch|phase|subphase|task|subtask`;
- `capture_state` is summary-first recovery context:
  - overview + implications + follow-up pointers, not full history dump;
- runtime-enforced kind/type semantics via DB catalog + project allowlists + schema validation;
- strict non-user mutation gatekeeping via lease tuple + id/name/scope validation;
- fail-closed behavior on ambiguity and scope/token mismatch;
- completion contract enforcement and unresolved blocker/user-action surfacing;
- REST/tool-style contracts with response-size controls and pagination defaults.

## 2.3) Unresolved Roadmap Items Explicitly Deferred In This Wave

Do not implement in this wave unless user re-prioritizes:
- advanced import/export transport-closure concerns (branch/commit-aware divergence reconciliation and richer conflict tooling);
- remote/team auth-tenancy and advanced security policies;
- override-token hardening beyond current MVP warning/guardrail baseline;
- kind/template historical versioning;
- dynamic MCP tool-surface policy beyond initial static surface;
- roadmap template-library expansion and richer auto-generated file intelligence.

## 2.4) Current-Code Constraints Carried Into Lane Planning

Must-fix-before-http constraints from `PRE_MCP_FULL_CODE_REVIEW.md`:
- canonical root enforcement currently can fall back to search roots/CWD;
- attachment normalization permits unrestricted absolute path when root is missing;
- snapshot export is not full MCP closure (comments/kinds/capability/cursor-attention context missing);
- transport plan lacked measurable anti-loss acceptance criteria before this wave kickoff.

Additional implementation constraints:
- hotspot files requiring serialized ownership:
  - `internal/tui/model.go`
  - `internal/app/service.go`
  - `internal/adapters/storage/sqlite/repo.go`
- worker lanes run package-scoped checks only (`just test-pkg <pkg>`); integrator owns repo gates.

## 2.5) Wave Objective And Exit Criteria

Wave objective:
- close remaining pre-MCP hardening gaps;
- ship non-roadmap locked HTTP and stateless MCP capability;
- ship TUI blocker/warning integration by level;
- ship user-readable dogfooding worksheets for TUI and MCP-primary flows.

Mandatory exit criteria:
- all lane acceptance criteria met with evidence;
- docs synchronized incrementally (`README.md`, planning docs, test worksheets);
- integrator passes both `just check` and `just ci`;
- user validates dogfooding checkpoints before final wave closeout.

## 2.6) Parallel Lane Plan (Required A-E)

Lane A: pre-MCP hardening gaps (must-fix-before-http)
- objective:
  - enforce fail-closed root/path behavior and close must-fix gate items.
- acceptance:
  - no permissive fallback for missing canonical root in write/attach flows;
  - attachment normalization rejects missing/invalid root boundary;
  - hardening behavior has tests and README notes.
- worker test scope:
  - `just test-pkg ./internal/tui`
  - `just test-pkg ./internal/app`
- primary lock scope:
  - `internal/tui/model.go`
  - `internal/tui/model_test.go`
  - `internal/app/snapshot.go`

Lane B: HTTP API implementation (non-roadmap locked features)
- objective:
  - add HTTP-first transport slice for locked read/mutation contracts, including `capture_state`.
- acceptance:
  - API endpoints under `/api/v1` with summary-first defaults, pagination, and structured errors;
  - strict lease/scope enforcement for non-user mutations;
  - deterministic contract tests for key read/write paths.
- worker test scope:
  - `just test-pkg ./internal/adapters/server/httpapi`
  - `just test-pkg ./internal/app`
- primary lock scope:
  - `internal/adapters/server/httpapi/**`
  - `internal/adapters/server/common/**`
  - `cmd/kan/main.go`

Lane C: MCP stateless adapter/tooling (non-roadmap locked features)
- objective:
  - add stateless MCP HTTP adapter and tool glue for locked surfaces, including `kan.capture_state`.
- acceptance:
  - MCP endpoint mounted with stateless behavior;
  - tool inputs enforce scope/lease rules and summary-first outputs;
  - guardrail failures return deterministic tool errors.
- worker test scope:
  - `just test-pkg ./internal/adapters/server/mcpapi`
  - `just test-pkg ./internal/app`
- primary lock scope:
  - `internal/adapters/server/mcpapi/**`
  - `internal/adapters/server/mcpapi/**/*_test.go`

Lane D: TUI warning/panel/count/search integration by level
- objective:
  - expose unresolved blocker/user-action visibility across level-scoped workflows.
- acceptance:
  - row-level warning indicator + visible unresolved count;
  - compact current-scope panel for unresolved items;
  - search/filter integration supports `project|branch|phase|subphase|task|subtask`.
- worker test scope:
  - `just test-pkg ./internal/tui`
- primary lock scope:
  - `internal/tui/model.go` (serialized after Lane A release)
  - `internal/tui/model_test.go` (serialized after Lane A release)
  - `internal/tui/**/*attention*`

Lane E: docs/tests/worksheets synchronization
- objective:
  - keep docs aligned as code lands and ship dogfooding-ready worksheets.
- acceptance:
  - incremental `README.md` updates per merged slice;
  - planning docs aligned without noisy duplicate logs;
  - one complete TUI worksheet update and one MCP-primary worksheet created/updated.
- worker test scope:
  - `test_not_applicable` for docs-only checkpoints (must log rationale)
- primary lock scope:
  - `README.md`
  - `PLAN.md`
  - `PRE_MCP_CONSENSUS.md`
  - `PRE_MCP_EXECUTION_WAVES.md`
  - `MCP_DESIGN_AND_PLAN.md`
  - `TUI_MANUAL_TEST_WORKSHEET.md`
  - `MCP_DOGFOODING_WORKSHEET.md` (new if absent)

## 2.7) File-Lock Map And Serialization Rules

Zero-overlap policy:
- no concurrent edits to the same file across lanes;
- hotspot ownership is explicit and serialized by checkpoint.

Serialized hotspots:
- `internal/tui/model.go`: Lane A first, then Lane D.
- `cmd/kan/main.go`: Lane B first, then Lane C.
- `internal/app/service.go`: assigned only with explicit temporary lock grant from orchestrator.
- `internal/adapters/storage/sqlite/repo.go`: assigned only with explicit temporary lock grant from orchestrator.

Lock transfer rule:
- lane cannot start edits on a serialized hotspot until prior owner checkpoint is reviewed and integrated by orchestrator.

## 2.8) Checkpoint Ledger (Active Worklog)

Checkpoint format (required):
- checkpoint id;
- objective;
- lock owner(s);
- files edited;
- commands/tests and outcomes;
- blockers/decisions;
- next step.

### Checkpoint O-00 (Orchestrator kickoff)

- objective:
  - initialize wave execution tracker, lane plan, and lock map from current consensus.
- lock owner(s):
  - orchestrator only (`MCP_DESIGN_AND_PLAN.md`).
- files edited:
  - `MCP_DESIGN_AND_PLAN.md` (status switch + execution sections).
- commands/tests and outcomes:
  - `sed -n '1,220p' Justfile` -> pass (recipes confirmed for test/check/ci gates).
  - `sed -n '1,260p' PLAN.md` -> pass (baseline planning context loaded).
  - `sed -n '1,260p' PRE_MCP_CONSENSUS.md` -> pass (locked decisions loaded).
  - `sed -n '1,260p' PRE_MCP_EXECUTION_WAVES.md` -> pass (wave structure context loaded).
  - `sed -n '1,260p' PRE_PHASE11_CLOSEOUT_DISCUSSION.md` -> pass (closeout constraints loaded).
  - `sed -n '1,320p' PRE_MCP_FULL_CODE_REVIEW.md` -> pass (must-fix and transport-entry gates loaded).
  - `sed -n '1,240p' Pre_MCP_User_NOTES.md` -> pass (goal intent and historical decision context loaded).
  - tests: `test_not_applicable` (docs/process-only checkpoint).
- blockers/decisions:
  - decision: run required lane structure A-E exactly as requested;
  - decision: serialize hotspots explicitly to preserve zero-overlap safety;
  - blocker: pending user validation of lane/lock plan before worker kickoff.
- next step:
  - user checkpoint on lane scope/priority, then spawn worker lanes with contract-compliant prompts.

### Checkpoint O-01 (Planning-doc alignment)

- objective:
  - align planning docs to temporary wave directive so active execution/worklog ownership is unambiguous.
- lock owner(s):
  - orchestrator (`PLAN.md`, `PRE_MCP_CONSENSUS.md`).
- files edited:
  - `PLAN.md` (worklog governance updated to point MCP-wave execution logs to this file).
  - `PRE_MCP_CONSENSUS.md` (execution handoff note + scope guard wording aligned to active wave).
- commands/tests and outcomes:
  - `rg -n "worklog|execution ledger|MCP_DESIGN_AND_PLAN|PLAN.md is the only active|single source|canonical" PLAN.md PRE_MCP_CONSENSUS.md` -> pass (conflicting text identified).
  - `nl -ba PLAN.md | sed -n '1,220p'` -> pass (target section verified before edit).
  - `nl -ba PRE_MCP_CONSENSUS.md | sed -n '1,240p'` -> pass (target section verified before edit).
  - tests: `test_not_applicable` (docs/process-only checkpoint; no code-path changes).
- blockers/decisions:
  - decision: keep roadmap/intent in `PLAN.md` and active MCP-wave execution checkpoints in `MCP_DESIGN_AND_PLAN.md`.
  - blocker: pending user approval to begin worker-lane execution sequencing.
- next step:
  - finalize worker-lane prompts (A-E) and begin with Lane A hardening gate.

### Checkpoint O-02 (Lane A slice: fail-closed resource-root hardening)

- objective:
  - close must-fix root/attachment fallback gaps so task resource writes fail closed without project-root mapping.
- lock owner(s):
  - orchestrator acting as integrator for Lane A hotspot scope (`internal/tui/model.go`, `internal/tui/model_test.go`, `README.md`).
- files edited:
  - `internal/tui/model.go`
    - task resource-picker entry points now block when project root mapping is missing;
    - attachment normalization now errors when root is missing/invalid/not-directory;
    - non-task picker flows use browse-root fallback helper;
    - task-info help text no longer advertises cwd fallback for attach.
  - `internal/tui/model_test.go`
    - added/updated tests for strict root lookup and blocked task attachment flows when root mapping is missing;
    - expanded normalization tests for empty-root and non-directory-root failure branches.
  - `README.md`
    - clarified `paths.search_roots` use for root-selection browse flows only;
    - documented that task resource attachments require project-root mapping.
- commands/tests and outcomes:
  - Context7:
    - `resolve-library-id("go standard library")` -> pass.
    - `query-docs(... filepath Abs/Rel/Clean root-bound checks ...)` -> pass.
  - verification:
    - `just fmt` -> pass.
    - `just test-pkg ./internal/tui` -> pass (`ok ... 70.500s`).
    - `just check` -> pass.
    - `just ci` -> pass.
- blockers/decisions:
  - decision: canonical fail-closed behavior now enforced for task resource attachment when project root mapping is missing.
  - decision: keep `paths.search_roots` fallback for non-task browse flows (bootstrap/project root selection), not for task attachments.
  - blocker: Lane A remaining sub-slice is snapshot export/import closure expansion for MCP/HTTP context parity.
- next step:
  - execute Lane A sub-slice for snapshot closure gap, then open Lane B HTTP implementation kickoff.

### Checkpoint O-03 (Lane A slice: snapshot closure expansion)

- objective:
  - close the pre-HTTP snapshot portability gap by including locked policy/context entities needed for MCP/HTTP rehydration.
- lock owner(s):
  - orchestrator acting as integrator for Lane A app-snapshot scope (`internal/app/snapshot.go`, `internal/app/snapshot_test.go`).
- files edited:
  - `internal/app/snapshot.go`
    - expanded snapshot shape to include:
      - kind catalog definitions,
      - project allowed-kind closure,
      - comments,
      - capability leases;
    - added export collection helpers for project/task-scoped comments and scope-mapped capability leases;
    - added import upsert helpers for kind definitions, comments, and capability leases;
    - added validation + deterministic sort coverage for new snapshot sections.
  - `internal/app/snapshot_test.go`
    - expanded export/import tests to verify closure payload and round-trip persistence for kinds/allowlists/comments/capability leases.
  - `README.md`
    - documented expanded snapshot export closure contents (kind catalog/allowlists/comments/capability leases).
- commands/tests and outcomes:
  - Context7:
    - `query-docs(/websites/pkg_go_dev_go1_25_3, deterministic sort/time normalization query)` -> pass.
  - verification:
    - `just fmt` -> pass.
    - `just test-pkg ./internal/app` -> pass.
    - `just check` -> pass.
    - `just ci` -> pass.
- blockers/decisions:
  - decision: Lane A hardening gates from pre-review findings are now implemented in code and covered by tests.
  - note: snapshot now carries core kind/comment/capability closure; remaining transport-facing pagination/sizing behavior stays in Lane B/C.
- next step:
  - begin Lane B HTTP API implementation (locked non-roadmap features, summary-first + strict guardrails).

### Checkpoint O-04 (Parallel full-scope audit synthesis)

- objective:
  - run a parallel multi-lane audit pass for docs/prompt/code coverage, capture all gaps, and consolidate findings into one root report.
- lock owner(s):
  - orchestrator (report synthesis + checkpoint logging).
- files edited:
  - `FULL_PARALLEL_AUDIT.md` (new consolidated report artifact).
  - `MCP_DESIGN_AND_PLAN.md` (checkpoint evidence).
  - removed lane temp artifacts:
    - `.tmp/wave_audit_lane_a_prompt_docs.md`
    - `.tmp/wave_audit_lane_b_http_mcp.md`
    - `.tmp/wave_audit_lane_c_guardrails_core.md`
    - `.tmp/wave_audit_lane_d_tui.md`
    - `.tmp/wave_audit_lane_e_storage_snapshot.md`
    - `.tmp/wave_audit_lane_f_tests_docs_dogfood.md`
- commands/tests and outcomes:
  - subagent completion polling:
    - `functions.wait` across six lane agents -> pass (all completed and wrote lane reports).
  - lane report extraction:
    - `sed -n '1,260p' .tmp/wave_audit_lane_*.md` -> pass.
  - verifier checks:
    - `nl -ba README.md | sed -n '1,80p'` -> pass (scope text still pre-MCP-only).
    - `nl -ba PRE_MCP_EXECUTION_WAVES.md | sed -n '1,80p'` -> pass (scope still marked pre-MCP-only).
    - `ls internal/adapters` -> pass (`storage` only, no server adapters yet).
    - `rg -n "capture_state|CaptureState"` -> pass (plan-only references; no code implementation).
    - `rg -n "subphase" internal` -> pass (no `subphase` support in code).
    - `rg -n "attention" internal/tui` -> pass (no unresolved-attention UI code yet).
    - `ls MCP_DOGFOODING_WORKSHEET.md` -> fail expected (artifact missing).
  - cleanup:
    - `rm -f .tmp/wave_audit_lane_*.md` (explicit file list) -> pass.
  - tests:
    - `test_not_applicable` (audit/report-only checkpoint; no behavior changes introduced).
- blockers/decisions:
  - blocker: docs/prompt alignment drift remains (README + PRE_MCP_EXECUTION_WAVES still pre-MCP-only while wave is active).
  - blocker: MCP worksheet required by lane acceptance is missing.
  - blocker: locked transport slice still missing (`/api/v1`, stateless MCP adapter, `capture_state`, attention model, `subphase` scope support).
  - decision: consolidated findings are now tracked in `FULL_PARALLEL_AUDIT.md` as the wave-level audit baseline before next implementation slice.
- next step:
  - review consolidated findings with user, lock issue priority/order, then execute implementation lanes with zero-overlap lock scopes.

### Checkpoint O-05 (Lane W-C docs + worksheets alignment)

- objective:
  - align README/planning/worksheet artifacts with the active MCP wave and ship runnable dogfooding worksheets.
- lock owner(s):
  - worker lane `W-C` (`README.md`, `PLAN.md`, `PRE_MCP_CONSENSUS.md`, `PRE_MCP_EXECUTION_WAVES.md`, `MCP_DESIGN_AND_PLAN.md`, `TUI_MANUAL_TEST_WORKSHEET.md`, `MCP_DOGFOODING_WORKSHEET.md`).
- files edited:
  - `README.md`
    - removed pre-MCP-only wording drift; clarified active-wave in-progress MCP/HTTP scope and explicit roadmap-only transport-closure deferrals.
  - `PLAN.md`
    - updated top-level scope guard to reflect active-wave override while keeping advanced import/export transport closure concerns roadmap-only.
  - `PRE_MCP_CONSENSUS.md`
    - added explicit active-wave defer note for advanced import/export transport closure concerns.
  - `PRE_MCP_EXECUTION_WAVES.md`
    - marked as historical pre-MCP baseline and redirected active execution ownership to this file.
  - `TUI_MANUAL_TEST_WORKSHEET.md`
    - normalized all note anchors to explicit `pass|fail|blocked`;
    - added hierarchy focus-path + branch/phase/subphase board checks with deterministic expected outcomes.
  - `MCP_DOGFOODING_WORKSHEET.md` (new)
    - added runnable user+agent MCP-primary worksheet covering `capture_state`, guardrail failures, blocker/user-action verification, level-scoped search/filter behavior, and resume-after-context-loss flows.
  - `MCP_DESIGN_AND_PLAN.md`
    - expanded Lane E lock scope to include `PRE_MCP_EXECUTION_WAVES.md`;
    - logged this checkpoint evidence.
- commands/tests and outcomes:
  - Context7:
    - `resolve-library-id("markdown", "Need concise, runnable markdown worksheet formatting best practices for test checklists with pass/fail steps and clear sections.")` -> pass.
    - `query-docs(/websites/daringfireball_net_projects_markdown, checklist formatting query)` -> pass.
  - doc verification:
    - `wc -l README.md PLAN.md PRE_MCP_CONSENSUS.md PRE_MCP_EXECUTION_WAVES.md MCP_DESIGN_AND_PLAN.md TUI_MANUAL_TEST_WORKSHEET.md MCP_DOGFOODING_WORKSHEET.md` -> pass.
    - `rg -n "pre-MCP|roadmap|import|export|transport|capture_state|subphase|worksheet" README.md PLAN.md PRE_MCP_CONSENSUS.md PRE_MCP_EXECUTION_WAVES.md MCP_DESIGN_AND_PLAN.md TUI_MANUAL_TEST_WORKSHEET.md MCP_DOGFOODING_WORKSHEET.md` -> pass (alignment targets located and rechecked post-edit).
    - `nl -ba <lane-file> | sed -n '<target-range>'` on all lane files -> pass (target sections reviewed before/after edits).
  - tests:
    - `test_not_applicable` (docs-only lane; no runtime/code-path change in this checkpoint).
- blockers/decisions:
  - decision: active wave docs now explicitly allow locked non-roadmap MCP/HTTP delivery while keeping advanced import/export transport-closure concerns roadmap-only.
  - decision: worksheets use explicit `pass|fail|blocked` anchors only; blank/ambiguous checkpoints are sign-off blockers.
  - blocker: none in this lane scope.
- next step:
  - handoff to orchestrator for integration review and wave-level sequencing with code lanes.

### Checkpoint O-06 (Parallel lane integration: W1/W2 + orchestrator W3/W4)

- objective:
  - integrate transport wiring fixes, raise package coverage above floor, and verify hierarchy-focus behavior for branch/phase/subphase flows.
- lock owner(s):
  - worker lane `W1` (transport wiring + scope support),
  - worker lane `W2` (HTTP/MCP coverage tests),
  - orchestrator fallback lanes `W3` (sqlite coverage tests) and `W4` (TUI hierarchy verification) due agent-thread cap.
- files edited:
  - transport/app wiring:
    - `cmd/kan/main.go`
    - `cmd/kan/main_test.go`
    - `internal/adapters/server/common/types.go`
    - `internal/adapters/server/common/capture.go`
    - `internal/adapters/server/common/app_service_adapter.go` (new)
    - `internal/adapters/server/mcpapi/handler.go`
  - coverage/test lanes:
    - `internal/adapters/server/httpapi/handler_test.go`
    - `internal/adapters/server/mcpapi/handler_test.go`
    - `internal/adapters/storage/sqlite/repo_test.go`
  - consolidation:
    - `FULL_PARALLEL_AUDIT.md` (rewritten as current consolidated lane report),
    - removed temporary handoff files from `.tmp/`.
- commands/tests and outcomes:
  - worker lane scoped tests:
    - `just test-pkg ./cmd/kan` -> pass
    - `just test-pkg ./internal/adapters/server/httpapi` -> pass
    - `just test-pkg ./internal/adapters/server/mcpapi` -> pass
    - `just test-pkg ./internal/adapters/storage/sqlite` -> pass
  - orchestrator scoped verification:
    - `just test-pkg ./internal/tui` -> pass
    - `just fmt` -> pass
    - `just check` -> pass
    - `just ci` -> pass
  - notable coverage outcomes from `just ci`:
    - `internal/adapters/server/httpapi`: `94.1%`
    - `internal/adapters/server/mcpapi`: `85.2%`
    - `internal/adapters/storage/sqlite`: `70.6%`
- blockers/decisions:
  - decision: `capture_state` transport now accepts `project|branch|phase|subphase|task|subtask` and remains fail-closed on invalid tuples.
  - decision: serve-mode now wires non-nil app-backed attention service and app-backed capture_state adapter.
  - decision: MCP capture tool scope enum now exposes full locked scope set.
  - note: `resolve_attention_item.reason` is transport-visible but not yet persisted by app service.
  - blocker: none for gate completion.
- next step:
  - complete user-guided dogfooding worksheet runs (TUI + MCP-primary) and collect pass/fail evidence.

### Checkpoint O-07 (Docs/worklog closeout alignment for this turn)

- objective:
  - align root docs/worklog artifacts with current integrated state and remove stale temp-lane artifacts.
- lock owner(s):
  - orchestrator (`README.md`, `FULL_PARALLEL_AUDIT.md`, `MCP_DESIGN_AND_PLAN.md`).
- files edited:
  - `README.md`:
    - added serve-mode feature bullet (`/api/v1` + stateless `/mcp`);
    - corrected scope tuple text to include `subphase`;
    - updated wave wording to implemented + dogfooding closeout.
  - `FULL_PARALLEL_AUDIT.md`:
    - replaced stale baseline with current consolidated parallel lane report and gate evidence.
  - `.tmp/`:
    - removed lane handoff markdown files after consolidation.
- commands/tests and outcomes:
  - `ls -la .tmp` -> pass (only non-report directories remain after cleanup).
  - `just check` -> pass.
  - `just ci` -> pass.
- blockers/decisions:
  - decision: keep this file as the active checkpoint ledger; consolidated audit details live in `FULL_PARALLEL_AUDIT.md`.
  - blocker: final wave closure still requires user validation on worksheets per temporary directive.
- next step:
  - run collaborative worksheet validation with user and then request AGENTS temporary-directive reduction/removal guidance on completion confirmation.

### Checkpoint O-08 (Second independent readiness audit wave)

- objective:
  - run a fresh subagent audit set focused on goals, code quality, and go/no-go readiness before collaborative worksheet execution.
- lock owner(s):
  - orchestrator (`SECOND_PARALLEL_READINESS_AUDIT.md`, `MCP_DESIGN_AND_PLAN.md` checkpoint logging).
- files edited:
  - `SECOND_PARALLEL_READINESS_AUDIT.md` (new consolidated readiness report).
  - `MCP_DESIGN_AND_PLAN.md` (this checkpoint entry).
- commands/tests and outcomes:
  - independent subagent lanes completed for:
    - directive/docs alignment,
    - core/server quality,
    - TUI behavior readiness,
    - gate/worksheet readiness.
  - supporting gate evidence in this wave:
    - `just check` -> pass.
    - `just ci` -> pass.
- blockers/decisions:
  - blocker (quality): `capture_state` currently accepts full scope tuple but work rollups remain effectively project-wide for non-project scopes; this is recorded as a material correctness gap in `SECOND_PARALLEL_READINESS_AUDIT.md`.
  - decision: proceed with user checkpoint on whether to run worksheets now with known issue tracked, or patch scope-rollup behavior first.
- next step:
  - user decision: execute worksheet pass now or authorize immediate scope-rollup fix pass first.

## 3) Locked Baseline Carried Forward

From prior consensus (no change in this file):
- Kinds are DB-defined runtime enums (`kind_catalog` + project `allowed_kinds`) with hard pre-write validation.
- JSON schema validation is runtime, cached, and required before persistence.
- Non-user mutations are lease-gated (name/instance/token/scope validation).
- Overlapping orchestrators are hard-blocked by default with policy-gated override token flow.
- One canonical project root path is the write boundary for filesystem resources.
- Comments/descriptions are markdown source and rendered in TUI.
- REST/tool-style contracts are preferred over GraphQL for MVP.
- Thread/comment payloads should be recent-window + pagination by default in transport phase.
- Attention/blocker model is a required MCP-phase build target.
- Level-scoped operations are first-class:
  - `project`, `branch`, `phase`, `subphase`, `task`, `subtask`.
- A summary-first recovery call is required:
  - `capture_state` returns level-scoped overview context so agent/user can reorient fast after context loss/compaction.

## 4) Current Code Reality (What We Already Have)

### 4.1 App Core Strengths

- Service-layer operations are already centralized in `internal/app`.
- Domain and service enforce:
  - kind/allowlist checks,
  - schema validation,
  - capability lease enforcement,
  - completion-transition guardrails for lifecycle transitions.
- SQLite adapter already persists:
  - projects, columns, tasks/work_items,
  - comments,
  - kind catalog + project allowlists,
  - capability leases,
  - change events.

### 4.2 Existing Gaps For MCP/HTTP Phase

- No HTTP server exists today (only TUI + import/export/paths CLI flow in `cmd/kan/main.go`).
- No MCP transport adapter exists today.
- No dedicated `attention_items` storage model exists yet.
- No agent cursor/ack storage model exists yet for branch-scoped delta delivery.
- Snapshot export does not currently include full MCP-relevant closure:
  - comments, kind catalog closure bundles, capability state, config mappings.
- No API-facing request/response schema package exists yet.

## 5) Warning Ledger (Failure + Task-Loss Prevention)

This section exists to prevent context loss and hidden failures as we shift from docs to implementation.

### 5.1 Resolved But Important Failure Patterns

- Windows CRLF formatting drift caused `gofmt` failures in CI; fixed with `.gitattributes` + Windows checkout normalization.
- CI jobs were canceled on `main` due concurrency policy; fixed with branch-aware cancellation policy.
- Strict lease gating broke tests when actor identity assumptions were incorrect; tests were updated to use proper actor/lease paths.

### 5.2 Design Warnings To Carry Into MCP Phase

- If attention/blocker records are not first-class, agent blockers can be lost in prose threads.
- If completion cannot be blocked by unresolved blockers/required approvals, progress integrity degrades.
- If pagination and recent-window defaults are not explicit, response size can destabilize MCP clients.
- If tool contracts do not explicitly instruct escalation behavior, orchestrators/subagents can bypass user-consensus loops.
- If we do not enforce “no completion with unresolved required children/checklists” at service boundaries, UI/agent flows will drift.

### 5.3 Required Anti-Loss Rule (Design Lock Proposal)

Proposal to lock before coding:
- a node cannot transition to `done` while it has unresolved required completion criteria or unresolved blocking attention items;
- parent completion remains blocked if required children are incomplete;
- every blocked transition must return machine-readable reasons so agents can raise/resolve attention records.

### 5.4 Current Intent Drift To Fix Before/With Transport Work

Code audit findings that should be handled so transport does not encode the wrong behavior:
- Canonical root enforcement gap:
  - resource attachment logic can fall back to search roots/current directory when project root mapping is absent.
  - write/attach flows should fail deterministically when project root mapping is missing.
  - references: `internal/tui/model.go` (`resourcePickerRootForCurrentProject`, `normalizeAttachmentPathWithinRoot`).
- Hierarchy context visibility gap:
  - projection breadcrumb shows parent task-title chain, but does not explicitly surface branch/phase context.
  - navigation intent requires obvious `project -> branch -> phase -> ...` context in UI.
  - reference: `internal/tui/model.go` (`projectionBreadcrumb`).
- Kind-bootstrap strictness caveat:
  - initial allowlist seeding can insert built-in kinds when catalog/allowlist are empty.
  - runtime enum policy is DB-driven; fallback behavior should remain explicit and tightly controlled.
  - reference: `internal/app/kind_capability.go` (`initializeProjectAllowedKinds`).

## 6) Proposed Target Architecture (HTTP + Stateless MCP-Go)

## 6.1 Process Model

One process, one app core, two transport adapters:
- REST-style HTTP API for deterministic automation/programmatic integration.
- MCP adapter (served over HTTP) for LLM tool ecosystems.

Both call the same application service layer; no transport-specific business logic in handlers.

## 6.2 Package/Boundary Plan

Proposed new packages:
- `internal/adapters/server/httpapi`:
  - REST handlers, request decode/validate, response encode.
- `internal/adapters/server/mcpapi`:
  - MCP server setup, tool registration, tool handler glue to app services.
- `internal/adapters/server/common`:
  - shared transport middleware helpers (request id, body limits, error mapping).

No change to hexagonal direction:
- adapters call app service interfaces;
- domain/app remain transport-agnostic.

## 6.3 Server Entry Plan

Add CLI mode (example):
- `kan serve --http :8080 --mcp-endpoint /mcp --api-endpoint /api/v1`

Runtime composition:
- one `http.ServeMux`;
- mount REST endpoints under `/api/v1/...`;
- mount MCP streamable HTTP endpoint under `/mcp`;
- include `/healthz` and `/readyz`.

## 6.4 Stateless MCP-Go Plan

Transport choice:
- use streamable HTTP mode with stateless behavior enabled;
- avoid server-side session correctness dependencies.

Planned behavior:
- each tool call carries explicit scope/actor/lease arguments where required;
- request context is rebuilt per call from explicit args + HTTP metadata;
- cursor correctness is persisted in DB (not in-memory session state).

## 7) HTTP API Contract Plan (v1 Direction)

## 7.1 Endpoint Groups

- Projects:
  - list/get/create/update
- State capture:
  - capture level-scoped overview context (`capture_state`) with expansion hooks for deeper follow-up calls
- Work graph:
  - list items by scope,
  - get item context,
  - create/update/move/reparent/transition
- Kinds:
  - list/get/upsert,
  - set/list project allowlist
- Capability leases:
  - issue/heartbeat/renew/revoke/revoke-all
- Threads/comments:
  - create/list by target (with recent-window/pagination)
- Change feed:
  - list changes since cursor,
  - ack cursor
- Attention/blockers:
  - create/list/update/resolve/ack

## 7.2 Request Envelope Direction

For mutation endpoints, include:
- actor identity: `actor_type`, `agent_name`, `agent_instance_id` (if non-user);
- lease tuple: `lease_token`;
- scope context: `project_id`, `branch_id`, `scope_type`, `scope_id`;
- optional `override_token` where policy allows.

## 7.3 Response Sizing Defaults

Default response strategy:
- `view=summary` by default;
- explicit `view=full` for deep payloads;
- `limit`, `cursor`, and `recent_window` arguments for thread/attention/change feeds;
- deterministic ordering + stable cursor semantics.
- `capture_state` is summary-first by design:
  - return goal/status/blockers/comment highlights and "what needs user action now",
  - include explicit follow-up pointers/cursors for deeper calls.

## 7.4 Error Contract Direction

Return structured errors with:
- stable code (for programmatic handling),
- readable message,
- context payload (field/path/scope),
- optional remediation hint.

Map domain errors deterministically:
- lease errors,
- kind/schema errors,
- scope violations,
- transition blocked conditions.

## 7.5 `capture_state` Contract (MVP-Required)

Purpose:
- one call that returns a deterministic level-scoped overview so an agent/user can quickly re-acquire project intent and active risks.
- this is not full-history reconstruction; it is a reorientation bundle with clear next calls.

Level tuple (required):
- `project_id`
- `branch_id`
- `scope_type`: `project|branch|phase|subphase|task|subtask`
- `scope_id`

Input envelope (required):
- actor tuple + lease tuple (non-user calls),
- level tuple,
- optional `view=summary|full`,
- optional `include=` list for controlled expansion.

Output sections (summary default):
- `goal_overview`:
  - current node description/intended outcome,
  - active status rollups relevant to the level.
- `attention_overview`:
  - unresolved blockers/consensus/approval items,
  - `requires_user_action` highlights for agent-stopped work.
- `work_overview`:
  - open/in-progress child items and incomplete completion requirements.
- `comment_overview`:
  - important/recent comments and thread pointers.
- `warnings_overview`:
  - unresolved dependency/blocking signals and consistency warnings.
- `resume_hints`:
  - recommended next calls (for details),
  - cursors/tokens/anchors for deterministic drill-down.

Stability fields:
- `captured_at`
- `scope_path` (project -> branch -> ... -> target)
- `state_hash`
- `last_change_event_id`

## 8) MCP Tool Plan (Stateless HTTP MCP)

## 8.1 Tool Naming Direction

Use explicit namespaced tools, e.g.:
- `kan.list_projects`
- `kan.capture_state`
- `kan.get_branch_context`
- `kan.search_items`
- `kan.update_item`
- `kan.issue_capability_lease`
- `kan.list_attention_items`
- `kan.raise_attention_item`
- `kan.resolve_attention_item`
- `kan.list_changes_since`
- `kan.ack_changes`

## 8.2 Tool Contract Rules

Each mutation tool must:
- require explicit actor + lease tuple for non-user actions;
- validate scope pre-write;
- include escalation guidance:
  - if blocked on consensus/approval, raise attention record and ask user.

Each read tool must:
- support payload sizing parameters (`summary/full`, `limit`, `cursor`, `include`);
- return enough metadata for agents to understand blocking conditions.

`kan.capture_state` must:
- be level-scoped and lease/scope validated;
- return summary-first reorientation data (goal/status/blockers/comments/warnings);
- include deterministic pointers for follow-up calls instead of overloading one response.

## 8.3 Tool Discovery / List Changed Strategy

MVP direction:
- keep tool surface static for first MCP implementation slice;
- declare tool capability; optional `listChanged` can be false initially.

Roadmap direction:
- if tool sets become policy/dynamic, adopt `notifications/tools/list_changed` flow with explicit compatibility notes.

## 8.4 Tool Metadata/Schema Loading Strategy

MVP direction:
- keep tool names static;
- keep tool output summary-first to reduce context pressure;
- rely on `tools/list` metadata + schema as the source of current tool contracts.

Refresh strategy:
- client/orchestrator refreshes tool metadata at startup and on `notifications/tools/list_changed`.
- tool docs should clearly separate summary fields vs expandable detail fields.

## 9) Data Model Additions Needed Before Transport Build

## 9.1 Attention Items (Required)

Add first-class `attention_items` model (proposed):
- `id`
- `project_id`
- `scope_type` (`project|branch|phase|subphase|task|subtask`)
- `scope_id`
- `state` (`open|acknowledged|resolved`)
- `kind` (`blocker|consensus_required|approval_required|risk_note`)
- `summary`
- `body_markdown`
- `requires_user_action`
- actor audit fields (`created_by_*`, `ack_by_*`, `resolved_by_*`, timestamps)

## 9.2 Cursor/Ack Model (Required)

Add branch-scope cursor storage (proposed):
- `agent_instance_id`
- `branch_scope_id`
- `last_acked_change_event_id`
- `updated_at`

Guarantee:
- no ack => deterministic resend;
- ack => cursor advance.

## 9.3 Optional (Likely Useful)

- request audit metadata table for HTTP/MCP envelope diagnostics.
- transport error/event correlation IDs in change/attention writes.

## 9.4 Capture-State Query Support (Required For Performance)

Need query/index support so `capture_state` is fast and deterministic:
- scope-path resolution (project/branch/phase/subphase/task/subtask ancestry),
- unresolved attention by scope with `requires_user_action` prioritization,
- open child-work rollups + completion requirement summary,
- recent/important comment window retrieval,
- unresolved dependency/blocking rollups for warning summary.

## 10) TUI Prerequisites For MCP-Ready Coordination

Required UI additions (already consensus-locked, still pending implementation):
- warning indicator on rows with open attention entries;
- always-visible compact attention panel for current scope;
- search + quick action + command palette hooks for attention filtering;
- paginated full attention list viewer for user triage.
- unresolved attention count in visible status/header region.
- panel/search must support level filtering across:
  - `project`, `branch`, `phase`, `subphase`, `task`, `subtask`.

## 11) Implementation Phasing (Recommended)

## Phase 11.0 (Current)
- lock contracts and open decisions in this file.

## Phase 11.1
- add DB/schema + app service support for attention items and cursor/ack model;
- add domain tests and repo tests.

## Phase 11.2
- add REST read endpoints (summary-first, pagination, sizing rules), including `capture_state`.

## Phase 11.3
- add REST mutation endpoints with strict lease/scope enforcement.

## Phase 11.4
- add MCP adapter using stateless streamable HTTP and map tools to app services, including `kan.capture_state`.

## Phase 11.5
- add TUI attention panel/warnings/filter integration.

## Phase 11.6
- contract docs, template updates (`AGENTS.md`/`CLAUDE.md` generation rules), CI and manual worksheet refresh.

## 12) Testing + Quality Gate Plan

Required coverage per implementation slice:
- `internal/domain`: enum/state validation, attention lifecycle, cursor rules.
- `internal/app`: guard enforcement, transition blocking, pagination and sizing.
- `internal/adapters/storage/sqlite`: migrations + deterministic query ordering.
- `internal/adapters/server/httpapi`: request decode/validate/error mapping.
- `internal/adapters/server/mcpapi`: tool argument validation + result envelopes.
- `internal/tui`: attention indicators/panel behavior and command hooks.

Gate strategy:
- lane-level `just test-pkg <pkg>`;
- final integrator `just ci`;
- manual worksheet updates for new TUI behavior.

## 13) Open Questions For Discussion

Important: answer Section 13.1 first (general goal lock).  
After those are locked, move to Section 13.2 (specific contracts).

## 13.1 General Goal-Alignment Questions (Answer First)

1. Is the primary Phase 11 objective internal dogfooding for agent workflows, or external integration readiness?
2. Should REST and MCP ship in the same milestone, or should REST precede MCP by one slice?
3. Do you want strict “no dynamic tool set” for MVP (static tools only), even if project policy differs?
4. Should attention/blocker records be mandatory for all unresolved approvals, or optional but strongly recommended?
5. Should unresolved `approval_required` and `consensus_required` both block `progress -> done` by default?
6. Is user-first review of blocker queues required before any agent can request override?
7. Should all non-user mutations require lease tuples always, or allow explicit project-level opt-out for local-only runs?
8. For dogfooding, do you want MCP exposed only on localhost by default (`127.0.0.1`)?
9. Should we lock a single canonical API version for MVP (`/api/v1` and tool schema version `v1`)?
10. Should branch context remain required in all delta/attention APIs, even when only one branch exists?
11. Is “fail closed” the default for any ambiguous scope/token mismatch?
12. Should we treat missing project root mapping as hard fail for file-path resources in all transports?
13. Should we lock “one canonical writable root path” in transport contracts exactly as in TUI policy?
14. Is preserving human auditability more important than minimizing payload size in ambiguous cases?
15. Should we require explicit user-approval metadata on every override action from day one?

## 13.2 Specific Contract Questions (After 13.1 Is Locked)

1. For stateless MCP HTTP, do you want one endpoint `/mcp` only, or versioned endpoint (`/mcp/v1`)?
2. Should REST use JSON body-only mutation envelopes, or allow key headers for lease tuple fields?
3. Default `limit` for list endpoints: 25, 50, or 100?
4. Default thread `recent_window`: 20, 50, or 100 comments?
5. Default attention list order: newest first by `created_at`, or priority by kind then time?
6. Cursor format preference: opaque token vs numeric event id + scope tuple?
7. Should `ack_changes` accept partial ack by event id, or only “ack all delivered” semantics?
8. Should we include `unmet_completion_requirements` in every item-context response by default?
9. Should attention `acknowledged` state still block completion, or only `open` with blocking kinds?
10. Do you want separate attention kinds for `dependency_blocked` vs general `blocker`?
11. Should attention records support child-thread comments, or stay single markdown body in MVP?
12. Should attention resolution require reason text always?
13. Should override actions require both override token and reason markdown?
14. Should we support per-project override-token rotation in MVP, or roadmap only?
15. Should lease renewals require same token + instance id only, or also enforce role/scope echo params?
16. Do you want auto-heartbeat on every successful mutation (current behavior) preserved for HTTP/MCP?
17. Should we expose `list_capability_leases` endpoint/tool to users and orchestrators in MVP?
18. Should subagent lease issuance require parent lease id always, or allow direct issue by user actor?
19. Should equal-scope delegation require explicit flag in each issuance call even if project policy allows it?
20. Should REST and MCP responses include lightweight policy hints (e.g., `override_allowed`, `requires_user_action`)?
21. Should `kind_payload` be returned raw always, or only with `include=kind_payload`?
22. Should tool outputs include markdown-rendered previews, or raw markdown only?
23. Should thread/comment list include author actor metadata by default in summary view?
24. Should `search_items` include archived by default in MCP, or explicit opt-in only?
25. Should we enforce request size limits uniformly (e.g., 1MB body) across REST and MCP?
26. Should we expose health endpoints without auth/lease checks in MVP?
27. Should we provide a `dry_run=true` mode for mutating tools/endpoints in MVP or roadmap?
28. Should we emit transport-level audit events for read operations or only mutations?
29. Should project export include attention/cursor state in MVP or roadmap?
30. Should TUI attention panel show only unresolved requiring user action, or all open records?
31. Should command palette include dedicated attention commands (`attention-open`, `attention-resolve`) in first TUI slice?
32. Should unresolved attention count be shown in project picker rows as well?
33. Should we include branch mismatch hints in import warnings, even before full reconciliation tools exist?
34. Should we lock an MCP tool deprecation policy now (e.g., one minor version grace window)?
35. Should generated `AGENTS.md`/`CLAUDE.md` templates include a mandatory “raise attention before guessing” section?
36. For `capture_state`, what should be the default "important comments" window (for example top 5 + last 10)?
37. Should `capture_state` include full node description by default, or truncated summary with explicit expand flag?
38. Should `capture_state` return separate `agent_blockers` and `user_action_required` arrays, or one unified attention list with flags?
39. Should `capture_state` include descendant-depth controls (`depth=0|1|2|all`) for subphase-heavy trees?
40. Should `capture_state` include contract-failure evidence snippets by default, or only IDs + follow-up pointers?

## 14) Initial Recommendation Snapshot (So We Have A Starting Point)

- Start with REST + MCP in same process, but sequence build as REST-first then MCP adapter.
- Keep tool surface static in first MCP slice (no dynamic list-changed behavior yet).
- Make attention table first-class (not embedded JSON) for filtering, pagination, and audit quality.
- Use opaque cursor tokens backed by persisted `(agent_instance_id, branch_scope_id, event_id)` state.
- Keep strict lease gating for non-user mutations with fail-closed behavior.
- Default to summary responses with explicit expansion options.
- Keep localhost-only default bind for serve mode in MVP.
- Make `capture_state` a first-class MVP read surface for deterministic context reorientation.

## 15) Sources

- `PRE_MCP_CONSENSUS.md`
- `PRE_MCP_EXECUTION_WAVES.md`
- `PRE_PHASE11_CLOSEOUT_DISCUSSION.md`
- `PLAN.md`
- `Pre_MCP_User_NOTES.md`
- `USER_RESPONSE.md`
- `response.md`
- `cmd/kan/main.go`
- `internal/app/service.go`
- `internal/app/kind_capability.go`
- `internal/app/snapshot.go`
- `internal/adapters/storage/sqlite/repo.go`
- `internal/domain/*.go`
- Context7:
  - `mcp-go` streamable HTTP + tool registration docs
  - MCP 2025-11-25 specification (tools capability/list_changed)
  - Go standard library context references
