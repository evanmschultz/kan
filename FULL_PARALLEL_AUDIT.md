# Full Parallel Audit + Fix Consolidation (2026-02-24)

## Scope
This report consolidates the current execution wave after parallel lane work, code integration review, and full gate verification.

Primary user asks covered in this run:
- identify what was missing from docs/prompt and code,
- fix transport + TUI hierarchy focus concerns,
- ensure stateless MCP via `mcp-go`,
- produce lane handoff files in `.tmp/`, consolidate in root, then remove temp files.

## Lane Execution Summary

### Lane W1 (subagent) - Transport Integration
Objective:
- wire serve-mode HTTP/MCP to app-level `capture_state` + attention APIs with full level scope support.

Delivered:
- `cmd/kan/main.go`: `runServe` now wires `servercommon.NewAppServiceAdapter(svc)` as both `CaptureState` and `Attention`.
- `cmd/kan/main_test.go`: serve wiring test now asserts non-nil attention dependency.
- `internal/adapters/server/common/types.go`: added canonical scope constants and `SupportedScopeTypes()` for `project|branch|phase|subphase|task|subtask`.
- `internal/adapters/server/common/capture.go`: `normalizeCaptureStateRequest` now accepts full scope set and remains fail-closed.
- `internal/adapters/server/common/app_service_adapter.go` (new): maps transport contracts to app `CaptureState`/attention operations.
- `internal/adapters/server/mcpapi/handler.go`: `kan.capture_state` tool enum uses full supported scope types.

Scoped tests:
- `just test-pkg ./cmd/kan` pass
- `just test-pkg ./internal/adapters/server/mcpapi` pass

### Lane W2 (subagent) - HTTP/MCP Coverage Lift
Objective:
- raise coverage in server adapters by hitting uncovered branches and fail paths.

Delivered:
- `internal/adapters/server/httpapi/handler_test.go`: added route/method guards, unavailable services, malformed payloads, error mapping, helper branch tests.
- `internal/adapters/server/mcpapi/handler_test.go`: added nil-dependency checks, config normalization, nil handler behavior, tool error mapping, and optional attention-tool execution/error tests.

Scoped tests:
- `just test-pkg ./internal/adapters/server/httpapi` pass
- `just test-pkg ./internal/adapters/server/mcpapi` pass

### Lane W3 (orchestrator fallback lane) - SQLite Coverage Lift
Reason:
- agent thread limit blocked additional concurrent worker spawn.

Delivered:
- `internal/adapters/storage/sqlite/repo_test.go`: added attention validation/not-found/error-path tests and migration checks for attention table/index.

Scoped tests:
- `just test-pkg ./internal/adapters/storage/sqlite` pass

### Lane W4 (orchestrator fallback lane) - TUI Hierarchy Focus Verification
Reason:
- agent thread limit blocked additional concurrent worker spawn.

Delivered:
- verified existing coverage in `internal/tui/model_test.go` for:
  - focus path + parent line rendering,
  - branch/phase/subphase focused subtree board rendering,
  - level-scoped search filtering behavior.
- no code edits required in this lane.

Scoped tests:
- `just test-pkg ./internal/tui` pass

## Missing Items Review (Docs/Prompt + Code)

### Missing from docs/prompt (current status)
Addressed:
- docs now explicitly keep advanced import/export transport-closure concerns as roadmap-only while allowing locked MCP/HTTP wave delivery.
- MCP dogfooding worksheet exists and covers capture_state, guardrails, blocker/user-action flows, level-scoped behavior, and resume-after-context-loss scenarios.

Still to keep in mind:
- historical sections in `MCP_DESIGN_AND_PLAN.md` still include older “not implemented yet” design text; current truth should be interpreted from latest checkpoints and gate evidence.

### Code issues (current status)
Fixed in this run:
- `mcp-go` stateless MCP transport is active (`WithStateLess(true)` retained).
- serve wiring now uses app-backed capture/attention adapter (not `nil` attention).
- scope-type support now includes `project|branch|phase|subphase|task|subtask` in capture-state normalization and MCP schema.
- HTTP/MCP and sqlite coverage floors now pass CI thresholds.
- TUI hierarchy focus/path behavior is present and test-covered.

Residual note:
- `ResolveAttentionItemRequest.reason` is accepted at transport level but not persisted/used by app service yet.

## Gate Evidence

Scoped package checks run during integration:
- `just test-pkg ./cmd/kan` pass
- `just test-pkg ./internal/adapters/server/httpapi` pass
- `just test-pkg ./internal/adapters/server/mcpapi` pass
- `just test-pkg ./internal/adapters/storage/sqlite` pass
- `just test-pkg ./internal/tui` pass

Wave gates:
- `just check` pass
- `just ci` pass

Coverage highlights from `just ci`:
- `internal/adapters/server/httpapi`: 94.1%
- `internal/adapters/server/mcpapi`: 85.2%
- `internal/adapters/storage/sqlite`: 70.6%

## Consolidation Source Files
This consolidated report was compiled from:
- `.tmp/w1_transport_handoff.md`
- `.tmp/w2_http_mcp_tests_handoff.md`
- `.tmp/w3_sqlite_tests_handoff.md`
- `.tmp/w4_tui_verification_handoff.md`
