# Pre-MCP Full Code Review

Date: 2026-02-24  
Status: Completed (code + docs readiness review; no runtime feature edits in this pass)

## 1) Purpose

Provide a single, high-signal readiness review for pre-Phase-11 and pre-MCP state:
- validate alignment with locked product intent;
- identify what is truly complete vs drifted;
- classify findings by transport impact (`must_fix_before_http`, `safe_to_defer`, `already_aligned`);
- give a concrete recommendation on starting HTTP-first + stateless MCP.

## 2) Review Scope

Reviewed artifacts:
- `internal/domain/**`
- `internal/app/**`
- `internal/adapters/storage/sqlite/**`
- `internal/tui/**`
- `internal/config/**`
- `cmd/kan/main.go`
- `README.md`
- `PLAN.md`
- `PRE_MCP_CONSENSUS.md`
- `PRE_MCP_EXECUTION_WAVES.md`
- `MCP_DESIGN_AND_PLAN.md`
- `TUI_MANUAL_TEST_WORKSHEET.md`

## 3) Method

- Parallel audit lanes were run via explorer subagents:
  - Lane `R1`: domain/app/storage invariants and gatekeeping.
  - Lane `R2`: TUI hierarchy render + warnings + command/search discoverability.
  - Lane `R3`: export/import/path/portability and HTTP-forward risk.
  - Lane `R4`: docs alignment and single-source-of-truth drift.
- Orchestrator validated key claims directly in code after lane handoff.
- Baseline quality gate executed: `just ci` (pass).

## 4) External Standards Used (Context7)

Resolved and reviewed before this pass:
- `/mark3labs/mcp-go`
- `/websites/modelcontextprotocol_io_specification_2025-11-25`

Relevant transport constraints carried into recommendation:
- streamable HTTP + stateless mode is supported in `mcp-go`;
- server must declare tools capability;
- tool discovery uses `tools/list` (with pagination support);
- tool-set updates should notify with `notifications/tools/list_changed`.

## 5) Findings (Severity-Ranked)

### 5.1 must_fix_before_http

1. Canonical root enforcement can silently fall back to search roots/CWD.
- Why this matters:
  - remote HTTP-first mutations can target unintended filesystem scope when project root mapping is missing.
  - this conflicts with strict root-boundary intent.
- Evidence:
  - `internal/tui/model.go:3324` (`resourcePickerRootForCurrentProject`) falls back through search roots, default root, then `"."`.
  - `cmd/kan/main.go:341` bootstrap gating checks identity + search roots, not per-project canonical root mapping.

2. Attachment normalization allows unrestricted absolute path when root is empty.
- Why this matters:
  - no fail-closed behavior for missing root boundary in write/attach flow.
- Evidence:
  - `internal/tui/model.go:3528` (`normalizeAttachmentPathWithinRoot`) returns absolute path when `root == ""` instead of rejecting.

3. Snapshot is not a full-fidelity portability bundle for upcoming remote workflows.
- Why this matters:
  - export/import currently misses key entities required for complete policy/context reconstruction (comments, kind catalog closure, capability state, change events).
  - this is risky if treated as collaboration-grade transport now.
- Evidence:
  - `internal/app/snapshot.go:18` snapshot shape includes projects/columns/tasks only.
  - sqlite schema includes additional entities not present in snapshot bundle:
    - `kind_catalog`, `project_allowed_kinds`, `comments`, `change_events`, capability lease tables in `internal/adapters/storage/sqlite/repo.go:171`.

4. Phase-11 design gate lacks measurable acceptance criteria for transport-critical promises.
- Why this matters:
  - design contains direction but no hard entry/exit checks for anti-loss invariants (attention blocking, cursor/ack behavior, lease enforcement proof points).
- Evidence:
  - planning intent exists in `MCP_DESIGN_AND_PLAN.md`, but no acceptance checklist tied to test evidence for those promises.

### 5.2 safe_to_defer

1. Hierarchy breadcrumb does not explicitly show branch/phase labels.
- Impact:
  - navigation works, but explicit scope readability is weaker than target intent.
- Evidence:
  - `internal/tui/model.go:6491` (`projectionBreadcrumb`) builds parent-title chain only.

2. Blocker/dependency warning prominence is weaker than desired.
- Impact:
  - counts exist, but unresolved warnings are not elevated into dedicated high-contrast attention UI in current TUI.
- Evidence:
  - compact dependency summary exists in `internal/tui/model.go:6519`, but no dedicated attention panel/count channel yet.

3. `SetProjectAllowedKinds` can omit the current project kind and lock future updates.
- Impact:
  - recoverable configuration trap; should be guarded.
- Evidence:
  - rewrite path in `internal/app/kind_capability.go:149` does not force-retain project’s current kind.

4. Search-root normalization remains path-cleaning, not deterministic absolute binding.
- Impact:
  - remote daemon/process working directory can alter interpretation of relative roots.
- Evidence:
  - `internal/config/config.go:366` (`normalize`) calls search-root normalizer; absolute binding is deferred to later usage sites.

### 5.3 already_aligned

1. Runtime kind validation is DB-driven with pre-write enforcement.
- Evidence:
  - `internal/app/kind_capability.go:496` (`validateTaskKind`).

2. JSON-schema payload validation is compiled/cached and enforced.
- Evidence:
  - `internal/app/kind_capability.go:590`
  - `internal/app/schema_validator.go:62`

3. Non-user mutation writes are lease-gated and fail closed.
- Evidence:
  - `internal/app/kind_capability.go:381` (`enforceMutationGuard`).
  - tuple identity checks include name + instance + token.

4. Lifecycle transition guards enforce completion contracts.
- Evidence:
  - `internal/app/service.go:384` (`MoveTask`) enforces start/completion criteria and child completion constraints.

5. Runtime audit/change tracking exists for task mutations.
- Evidence:
  - actor-normalized event writing in `internal/adapters/storage/sqlite/repo.go:1236`.

6. Command/search discoverability is strong for pre-MCP UX.
- Evidence:
  - command palette definitions and help overlays in `internal/tui/model.go:2159`, `internal/tui/model.go:6788`.

7. Quality gate baseline is currently healthy.
- Evidence:
  - `just ci` passed in this review pass.

## 6) Grand Goal Coverage Matrix

Legend:
- `done`: implemented and aligned.
- `partial`: implemented with notable drift.
- `planned`: intentionally not implemented pre-MCP, but planning contracts exist.
- `missing`: required but not yet represented sufficiently.

1. Strict kind/type semantics with DB-defined runtime enum behavior: `done`.
2. Hard pre-write gatekeeping with actor/lease tuple enforcement: `done`.
3. Deterministic completion contracts for progress/done transitions: `done`.
4. Canonical writable root boundary for path/resource operations: `partial`.
5. Explicit hierarchy context clarity (project/branch/phase/task/subtask) in TUI render: `partial`.
6. Attention/blocker model and warning panel/count pipeline: `planned`.
7. Export/import portability closure for remote collaboration safety: `partial`.
8. HTTP-first stateless transport design with MCP extension path: `planned`.
9. Tool discovery/update contract (`tools/list`, `list_changed`) strategy: `planned`.
10. Single-source-of-truth docs with reduced worklog noise: `done` (improved), with minor acceptance-check clarity gap.

## 7) Recommendation: Can HTTP-First Start Now?

Short answer: **Yes, with entry gates.**

Recommended approach:
- Start HTTP-first implementation now for core non-path mutation/read flows and shared service contract extraction.
- Keep stateless MCP adapter in same architecture stream, but behind explicit transport entry criteria.

Required entry gates before enabling path-sensitive remote mutations:
1. Fail closed when project root mapping is missing for any path/resource write surface.
2. Remove permissive root-empty normalization behavior for attachments.
3. Define and lock transport acceptance criteria for:
  - lease tuple enforcement;
  - completion blocking behavior;
  - attention/blocker anti-loss behavior;
  - cursor/ack semantics and pagination defaults.

If those three gates are addressed, small TUI polish gaps should **not** block HTTP/MCP progress.

## 8) Recommended Immediate Work Split

1. Pre-transport hardening slice (small, high impact):
- root-mapping fail-closed path policy;
- attachment normalization strictness;
- design-gate acceptance checklist in `MCP_DESIGN_AND_PLAN.md`.

2. HTTP-first implementation slice:
- `/api/v1` read/write surfaces over existing app services;
- strict error schema + scope/lease validation pass-through.

3. Stateless MCP slice:
- static tool names in MVP;
- dynamic tool metadata/schema generation from current DB policy state;
- emit `tools/list_changed` only when tool metadata surface changes materially.

4. Attention/blocker slice (pre-MCP transport dependency):
- add storage + service APIs;
- expose list/create/resolve with pagination and scope filters;
- TUI count + panel + marker integration by current level.

## 9) Final Go/No-Go

- **Go** for HTTP-first architecture and scaffolding.
- **Conditional Go** for remote/path-sensitive mutation exposure: only after `must_fix_before_http` items 1 and 2 are closed.
- Keep snapshot expansion and extra TUI clarity work in the next slices, not as a stop-everything blocker.
