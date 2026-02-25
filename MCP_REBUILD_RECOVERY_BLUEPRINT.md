# MCP Rebuild Recovery Blueprint (From `c92e213`)

This file reconstructs the full MCP/HTTP wave work so it can be re-applied or rebuilt after rollback/loss.

## 1) Source of Truth

- MCP/HTTP wave implementation commit: `c92e213`
- Commit title: `feat: ship mcp/http wave integration and e2e agent runbook`
- Parent commit (pre-MCP implementation): `1a6c62e`

## 2) Fastest Recovery Options

### Option A: Re-apply entire MCP wave commit (recommended)

```bash
git checkout <target-branch>
git cherry-pick c92e213
```

If conflicts occur:

```bash
git status
# resolve conflicts
git add <resolved-files>
git cherry-pick --continue
```

### Option B: Export and apply as patch

```bash
git format-patch -1 c92e213 --stdout > /tmp/c92e213.patch
git checkout <target-branch>
git apply --index /tmp/c92e213.patch
git commit -m "reapply mcp/http wave from c92e213"
```

### Option C: Manual rebuild (order and scope below)

Use sections 4-9 in this file.

## 3) Full File Manifest Changed By `c92e213`

```text
M	AGENTS.md
A	FULL_PARALLEL_AUDIT.md
A	MCP_DESIGN_AND_PLAN.md
A	MCP_DOGFOODING_WORKSHEET.md
A	MCP_HTTP_E2E_AGENT_REPORT.md
M	PLAN.md
M	PRE_MCP_CONSENSUS.md
M	PRE_MCP_EXECUTION_WAVES.md
A	PRE_MCP_FULL_CODE_REVIEW.md
M	PRE_PHASE11_CLOSEOUT_DISCUSSION.md
M	README.md
A	SECOND_PARALLEL_READINESS_AUDIT.md
M	TUI_MANUAL_TEST_WORKSHEET.md
M	cmd/kan/main.go
M	cmd/kan/main_test.go
M	go.mod
M	go.sum
A	internal/adapters/server/common/app_service_adapter.go
A	internal/adapters/server/common/capture.go
A	internal/adapters/server/common/types.go
A	internal/adapters/server/httpapi/handler.go
A	internal/adapters/server/httpapi/handler_test.go
A	internal/adapters/server/mcpapi/handler.go
A	internal/adapters/server/mcpapi/handler_test.go
A	internal/adapters/server/server.go
M	internal/adapters/storage/sqlite/repo.go
M	internal/adapters/storage/sqlite/repo_test.go
A	internal/app/attention_capture.go
A	internal/app/attention_capture_test.go
M	internal/app/kind_capability.go
M	internal/app/kind_capability_test.go
M	internal/app/ports.go
M	internal/app/service.go
M	internal/app/service_test.go
M	internal/app/snapshot.go
M	internal/app/snapshot_test.go
A	internal/domain/attention.go
A	internal/domain/attention_level_test.go
M	internal/domain/capability.go
M	internal/domain/errors.go
M	internal/domain/kind.go
A	internal/domain/level.go
M	internal/tui/model.go
M	internal/tui/model_test.go
M	internal/tui/options.go
```

## 4) What Was Implemented (Behavior-Level)

### 4.1 CLI/Runtime Serve Mode

- Added `serve` flow flags and wiring in `cmd/kan/main.go`:
  - `--http` default `127.0.0.1:8080`
  - `--api-endpoint` default `/api/v1`
  - `--mcp-endpoint` default `/mcp`
- `serve` injects a shared app-service adapter into HTTP and MCP transports.

### 4.2 Unified HTTP Server Composition

- New server composition in `internal/adapters/server/server.go`:
  - health endpoints: `/healthz`, `/readyz`
  - API endpoint mounted at configured `/api/v1`
  - MCP endpoint mounted at configured `/mcp`
- Fail-closed normalization:
  - endpoint path normalization
  - explicit rejection when API endpoint == MCP endpoint

### 4.3 HTTP API Surface (REST)

Implemented in `internal/adapters/server/httpapi/handler.go`:

- `GET /capture_state`
- `GET /attention/items`
- `POST /attention/items`
- `POST /attention/items/{id}/resolve`

Fail-closed handling implemented:
- strict JSON decoding (`DisallowUnknownFields`)
- trailing payload rejection
- structured error envelopes (`invalid_request`, `not_found`, `method_not_allowed`, etc.)

### 4.4 MCP Surface (mcp-go Streamable HTTP, Stateless)

Implemented in `internal/adapters/server/mcpapi/handler.go`:

- Streamable HTTP transport + stateless mode:
  - `WithEndpointPath(...)`
  - `WithStateLess(true)`
- Registered MCP tools:
  1. `kan.capture_state`
  2. `kan.list_attention_items`
  3. `kan.raise_attention_item`
  4. `kan.resolve_attention_item`

Error mapping done via `toolResultFromError(...)` with fail-closed categories.

### 4.5 Transport-Agnostic Adapter Layer

Added `internal/adapters/server/common/*`:

- `types.go`: request/response DTOs and shared error sentinels
- `capture.go`: capture-state request normalization
- `app_service_adapter.go`: maps transport contracts into app/domain service calls

Key behavior from adapter normalization:
- Valid scope types: `project|branch|phase|subphase|task|subtask`
- Default behavior when `scope_type` missing: project scope
- For project scope, `scope_id` defaults to `project_id`

### 4.6 Domain/App/Storage Changes

- New domain attention model and scope-level constructs:
  - `internal/domain/attention.go`
  - `internal/domain/level.go`
  - `internal/domain/attention_level_test.go`
- New app-level capture/attention service logic:
  - `internal/app/attention_capture.go`
  - `internal/app/attention_capture_test.go`
- SQLite repository support for attention/capture flows:
  - `internal/adapters/storage/sqlite/repo.go`
  - `internal/adapters/storage/sqlite/repo_test.go`
- Snapshot/import-export updates to include new entities/fields.

## 5) Net Diff Size

```text
 AGENTS.md                                          |   21 +
 FULL_PARALLEL_AUDIT.md                             |  110 ++
 MCP_DESIGN_AND_PLAN.md                             |  978 ++++++++++
 MCP_DOGFOODING_WORKSHEET.md                        |  370 ++++
 MCP_HTTP_E2E_AGENT_REPORT.md                       |  280 +++
 PLAN.md                                            | 1922 +-------------------
 PRE_MCP_CONSENSUS.md                               |  182 +-
 PRE_MCP_EXECUTION_WAVES.md                         |   34 +-
 PRE_MCP_FULL_CODE_REVIEW.md                        |  206 +++
 PRE_PHASE11_CLOSEOUT_DISCUSSION.md                 |    7 +
 README.md                                          |   34 +-
 SECOND_PARALLEL_READINESS_AUDIT.md                 |   44 +
 TUI_MANUAL_TEST_WORKSHEET.md                       |  274 +--
 cmd/kan/main.go                                    |   49 +-
 cmd/kan/main_test.go                               |   95 +
 go.mod                                             |   12 +
 go.sum                                             |   29 +
 .../adapters/server/common/app_service_adapter.go  |  429 +++++
 internal/adapters/server/common/capture.go         |  336 ++++
 internal/adapters/server/common/types.go           |  198 ++
 internal/adapters/server/httpapi/handler.go        |  316 ++++
 internal/adapters/server/httpapi/handler_test.go   |  717 ++++++++
 internal/adapters/server/mcpapi/handler.go         |  249 +++
 internal/adapters/server/mcpapi/handler_test.go    |  693 +++++++
 internal/adapters/server/server.go                 |  158 ++
 internal/adapters/storage/sqlite/repo.go           |  382 +++-
 internal/adapters/storage/sqlite/repo_test.go      |  301 +++
 internal/app/attention_capture.go                  |  325 ++++
 internal/app/attention_capture_test.go             |  210 +++
 internal/app/kind_capability.go                    |    8 +-
 internal/app/kind_capability_test.go               |   25 +
 internal/app/ports.go                              |    4 +
 internal/app/service.go                            |    3 +
 internal/app/service_test.go                       |   96 +
 internal/app/snapshot.go                           |  616 ++++++-
 internal/app/snapshot_test.go                      |  106 ++
 internal/domain/attention.go                       |  294 +++
 internal/domain/attention_level_test.go            |  139 ++
 internal/domain/capability.go                      |   12 +-
 internal/domain/errors.go                          |    5 +
 internal/domain/kind.go                            |   13 +-
 internal/domain/level.go                           |  170 ++
 internal/tui/model.go                              |  488 ++++-
 internal/tui/model_test.go                         |  386 +++-
 internal/tui/options.go                            |    8 +
 45 files changed, 9073 insertions(+), 2261 deletions(-)
```

## 6) Test Coverage Added In Wave

Primary new/expanded test files:

- `cmd/kan/main_test.go` (serve command wiring)
- `internal/adapters/server/httpapi/handler_test.go`
- `internal/adapters/server/mcpapi/handler_test.go`
- `internal/app/attention_capture_test.go`
- `internal/domain/attention_level_test.go`
- plus updates across app/domain/sqlite/tui tests for integration impact

## 7) Docs/Planning/Worksheets Added Or Updated In Wave

New or heavily updated artifacts included:

- `MCP_DESIGN_AND_PLAN.md`
- `MCP_DOGFOODING_WORKSHEET.md`
- `MCP_HTTP_E2E_AGENT_REPORT.md`
- `PRE_MCP_FULL_CODE_REVIEW.md`
- `FULL_PARALLEL_AUDIT.md`
- `SECOND_PARALLEL_READINESS_AUDIT.md`
- updates to `README.md`, `PLAN.md`, consensus/wave/pre-phase docs, and TUI worksheet

## 8) Known Gaps / Contract Mismatches Observed During Dogfooding

These were observed in execution reports and should be considered during rebuild:

1. MCP tool surface is intentionally narrow (4 tools only).
2. No CRUD tools for branch/phase/subphase/task/subtask exposed over MCP yet.
3. Empty-instance onboarding is weak (no dedicated bootstrap/info MCP tool).
4. HTTP vs MCP contract mismatch observed for attention scope defaults:
   - HTTP path accepted missing `scope_type/scope_id` by defaulting to project scope,
   - MCP `kan.raise_attention_item` requires explicit `scope_type/scope_id`.
5. Resume hints include relations that are not currently registered as MCP tools (documentation/contract alignment needed).

## 9) Manual Rebuild Order (If Not Cherry-Picking)

Use this sequence to reduce churn and conflicts:

1. Domain primitives
   - `internal/domain/errors.go`
   - `internal/domain/capability.go`
   - `internal/domain/kind.go`
   - add `internal/domain/level.go`
   - add `internal/domain/attention.go`

2. App service layer
   - `internal/app/ports.go`
   - `internal/app/service.go`
   - add `internal/app/attention_capture.go`
   - update capability/snapshot/service files impacted by new scope/attention behavior

3. Storage layer
   - `internal/adapters/storage/sqlite/repo.go`
   - `internal/adapters/storage/sqlite/repo_test.go`

4. Transport common layer
   - add `internal/adapters/server/common/types.go`
   - add `internal/adapters/server/common/capture.go`
   - add `internal/adapters/server/common/app_service_adapter.go`

5. HTTP + MCP adapters
   - add `internal/adapters/server/httpapi/handler.go`
   - add `internal/adapters/server/mcpapi/handler.go`
   - add `internal/adapters/server/server.go`

6. CLI integration
   - `cmd/kan/main.go`
   - `cmd/kan/main_test.go`

7. Dependency/module updates
   - `go.mod`
   - `go.sum`

8. TUI integration updates
   - `internal/tui/model.go`
   - `internal/tui/model_test.go`
   - `internal/tui/options.go`

9. Planning/docs/worksheets
   - reapply docs listed in section 7

## 10) Validation Gates After Rebuild

Run in order:

```bash
just test-pkg ./internal/adapters/server/httpapi
just test-pkg ./internal/adapters/server/mcpapi
just test-pkg ./internal/app
just test-pkg ./internal/domain
just test-pkg ./internal/adapters/storage/sqlite
just check
just ci
```

If TUI changes alter golden outputs:

```bash
just test-golden
# if expected changes:
just test-golden-update
just test-golden
```

## 11) Practical Recommendation

Because `c92e213` is still present and readable in this repository, **cherry-pick is the fastest and least lossy recovery path**. Use manual rebuild only if you intentionally want to recompose the wave incrementally.
