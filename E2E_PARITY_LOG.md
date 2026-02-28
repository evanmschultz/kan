# E2E Parity Log (hakoll/koll)

## Session
- Date: 2026-02-28
- Runtime: user-started `./koll serve` on `127.0.0.1:5437`
- Known seed data: project `User_Project` exists by design

## Independent Checks (Assistant)
1. HTTP reachability
- Probe: `GET /api/v1/capture_state` without `project_id`
- Result: PASS (reachable, deterministic `400 invalid_request`)

2. MCP initialize and tool discovery
- Probe: `initialize` (`protocolVersion=2025-06-18`), then `tools/list`
- Result: PASS (`2025-06-18` negotiated; 30 tools discovered)

3. Existing project lookup
- Probe: `tools/call koll.list_projects(include_archived=true)`
- Result: PASS (`User_Project` found; expected)

4. HTTP/MCP parity: capture_state
- Scope: `project_id=10cdd734-bf41-4155-b978-b5f5f5061050`, `scope_type=project`, `view=summary`
- Result: PASS (matching `state_hash`, scope name, and task totals)

5. HTTP/MCP parity: attention list
- Scope: same project, `state=open`
- Result: PASS (matching count `0`)

6. Stateless transport behavior
- Probe: bogus `Mcp-Session-Id` header on `tools/list`
- Result: PASS (request succeeds; stateless behavior holds)

7. Error handling transport checks
- Probe: unknown method (`unknown/method`)
- Result: PASS (deterministic JSON-RPC error response)
- Probe: invalid JSON body (`{`)
- Result: PASS (`400`, deterministic parse error)

8. Initialize protocol matrix
- `2024-11-05`: PASS (accepted)
- `2099-01-01`: PASS (deterministic fallback)
- missing `protocolVersion`: PASS (deterministic default)

## Collaborative Test Plan
### User-only (UI/UX visual parity)
1. Verify bootstrap identity flow and saved display name/actor type in TUI.
2. Verify board/help/modal rendering parity (layout, keybindings, copy, no visual regressions).
3. Verify project picker and command palette interaction parity.

### Assistant-only (runtime/API parity)
1. Re-run MCP/HTTP probes after each major TUI mutation you perform.
2. Validate capture-state and attention parity after mutations.
3. Record API-side regressions and exact request/response diffs.

### Together (state transition parity)
1. You create and mutate tasks in `User_Project`; I verify MCP/HTTP state transitions immediately.
2. We validate archive/restore/move/search behavior parity between TUI actions and API outputs.
3. We log every mismatch under "Issues" with repro steps.

## Issues
- None yet.

## Updates Needed
- Revisit release default behavior so dev-mode is explicit (not implicit) for packaged/public installs.
