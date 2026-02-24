# MCP + HTTP End-to-End Agent Execution Report

Status: `not_started`  
Owner Agent: `<fill>`  
Date: `<fill>`  
Branch/Commit Tested: `<fill>`

## 1) Mission
Run the full HTTP + MCP end-to-end test sweep and fill this file with evidence.

Hard requirements:
- Exercise **every HTTP route and every MCP tool option** in this file.
- Validate success paths, fail-closed paths, and method/shape errors.
- Keep results evidence-based (commands + output snippets).
- Do not mark complete unless every required row is filled `pass|fail|blocked`.

## 2) Completion Gate
All of the following must be true:
1. Every table row in sections 6-11 has `Result` filled.
2. `just check` and `just ci` pass in this working tree.
3. Parity checks in section 10 are completed.
4. Known issues (if any) are clearly documented in section 12 with reproduction steps.

## 3) Approval Checkpoints (pause and ask user)
Stop and ask user before continuing if any of these occur:
1. A command requests escalated permissions/sandbox override.
2. Any endpoint/tool behavior contradicts expected guardrails in a way that could hide data-loss or scope ambiguity.
3. You need to run destructive cleanup outside temp test directories.
4. You hit a mismatch that looks like a product decision rather than a bug.

Approval log:
- `A1` `<fill if used>`
- `A2` `<fill if used>`

## 4) Environment Setup
Record exact commands you ran.

```bash
just build
TMP_DIR="$(mktemp -d)"
DB_PATH="$TMP_DIR/kan-e2e.db"
KAN_BIN="./kan"

# Start server in one shell (or tmux pane)
$KAN_BIN --config config.example.toml --db "$DB_PATH" serve --http 127.0.0.1:18080 --api-endpoint /api/v1 --mcp-endpoint /mcp
```

In another shell:

```bash
BASE_HTTP="http://127.0.0.1:18080"
API="$BASE_HTTP/api/v1"
MCP="$BASE_HTTP/mcp"

# JSON-RPC helper
rpc() {
  curl -sS "$MCP" \
    -H 'content-type: application/json' \
    -d "$1"
}
```

Setup notes/evidence:
- Command output snippets: `<fill>`
- Server startup logs snippet: `<fill>`

## 5) Test Data Seed
Create minimal deterministic fixture:
- 1 project
- 1 branch node
- 1 phase node
- 1 subphase node
- 1 task node
- 1 subtask node
- At least 2 attention records (one `requires_user_action=true`, one false)

Seed commands/evidence:
- `<fill>`

Entity IDs captured:
- `PROJECT_ID=<fill>`
- `BRANCH_ID=<fill>`
- `PHASE_ID=<fill>`
- `SUBPHASE_ID=<fill>`
- `TASK_ID=<fill>`
- `SUBTASK_ID=<fill>`
- `ATTN_ID_OPEN=<fill>`
- `ATTN_ID_OPEN_2=<fill>`

## 6) HTTP Surface Coverage

### 6.1 Health/Readiness + Route Guards
| ID | Check | Command | Expected | Result (`pass|fail|blocked`) | Evidence |
|---|---|---|---|---|---|
| H-1 | `/healthz` | `curl -i "$BASE_HTTP/healthz"` | 200 + `{"status":"ok"}` | `<fill>` | `<fill>` |
| H-2 | `/readyz` | `curl -i "$BASE_HTTP/readyz"` | 200 + `{"status":"ok"}` | `<fill>` | `<fill>` |
| H-3 | unknown route | `curl -i "$API/nope"` | 404 structured error | `<fill>` | `<fill>` |
| H-4 | capture_state wrong method | `curl -i -X POST "$API/capture_state"` | 405 + `Allow: GET` | `<fill>` | `<fill>` |
| H-5 | attention/items wrong method | `curl -i -X DELETE "$API/attention/items"` | 405 + `Allow: GET, POST` | `<fill>` | `<fill>` |
| H-6 | resolve wrong method | `curl -i -X GET "$API/attention/items/$ATTN_ID_OPEN/resolve"` | 405 + `Allow: POST` | `<fill>` | `<fill>` |

### 6.2 `GET /capture_state` (every option)
Scope types to test: `project branch phase subphase task subtask`

| ID | Variant | Command | Expected | Result | Evidence |
|---|---|---|---|---|---|
| C-1 | defaults (`scope_type` omitted, `view` omitted) | `curl -sS "$API/capture_state?project_id=$PROJECT_ID"` | 200, summary defaults | `<fill>` | `<fill>` |
| C-2 | `view=full` | `curl -sS "$API/capture_state?project_id=$PROJECT_ID&view=full"` | 200 full accepted | `<fill>` | `<fill>` |
| C-3 | `scope_type=project` + matching `scope_id` | `curl -sS "$API/capture_state?project_id=$PROJECT_ID&scope_type=project&scope_id=$PROJECT_ID"` | 200 | `<fill>` | `<fill>` |
| C-4 | `scope_type=branch` | `curl -sS "$API/capture_state?project_id=$PROJECT_ID&scope_type=branch&scope_id=$BRANCH_ID"` | 200 | `<fill>` | `<fill>` |
| C-5 | `scope_type=phase` | `curl -sS "$API/capture_state?project_id=$PROJECT_ID&scope_type=phase&scope_id=$PHASE_ID"` | 200 | `<fill>` | `<fill>` |
| C-6 | `scope_type=subphase` | `curl -sS "$API/capture_state?project_id=$PROJECT_ID&scope_type=subphase&scope_id=$SUBPHASE_ID"` | 200 | `<fill>` | `<fill>` |
| C-7 | `scope_type=task` | `curl -sS "$API/capture_state?project_id=$PROJECT_ID&scope_type=task&scope_id=$TASK_ID"` | 200 | `<fill>` | `<fill>` |
| C-8 | `scope_type=subtask` | `curl -sS "$API/capture_state?project_id=$PROJECT_ID&scope_type=subtask&scope_id=$SUBTASK_ID"` | 200 | `<fill>` | `<fill>` |
| C-9 | missing `project_id` | `curl -i "$API/capture_state"` | 400 invalid_request | `<fill>` | `<fill>` |
| C-10 | invalid `view` | `curl -i "$API/capture_state?project_id=$PROJECT_ID&view=bad"` | 400 invalid_request | `<fill>` | `<fill>` |
| C-11 | invalid `scope_type` | `curl -i "$API/capture_state?project_id=$PROJECT_ID&scope_type=bad&scope_id=x"` | 400 invalid_request | `<fill>` | `<fill>` |
| C-12 | project `scope_id` mismatch | `curl -i "$API/capture_state?project_id=$PROJECT_ID&scope_type=project&scope_id=wrong"` | 400 invalid_request | `<fill>` | `<fill>` |
| C-13 | non-project missing `scope_id` | `curl -i "$API/capture_state?project_id=$PROJECT_ID&scope_type=branch"` | 400 invalid_request | `<fill>` | `<fill>` |

### 6.3 Attention HTTP APIs (every option)

#### `GET /attention/items`
| ID | Variant | Command | Expected | Result | Evidence |
|---|---|---|---|---|---|
| A-1 | required only | `curl -sS "$API/attention/items?project_id=$PROJECT_ID"` | 200 list | `<fill>` | `<fill>` |
| A-2 | `scope_type=project` + `scope_id` | `curl -sS "$API/attention/items?project_id=$PROJECT_ID&scope_type=project&scope_id=$PROJECT_ID"` | 200 | `<fill>` | `<fill>` |
| A-3 | `scope_type=branch` | `curl -sS "$API/attention/items?project_id=$PROJECT_ID&scope_type=branch&scope_id=$BRANCH_ID"` | 200 | `<fill>` | `<fill>` |
| A-4 | `scope_type=phase` | `curl -sS "$API/attention/items?project_id=$PROJECT_ID&scope_type=phase&scope_id=$PHASE_ID"` | 200 | `<fill>` | `<fill>` |
| A-5 | `scope_type=subphase` | `curl -sS "$API/attention/items?project_id=$PROJECT_ID&scope_type=subphase&scope_id=$SUBPHASE_ID"` | 200 | `<fill>` | `<fill>` |
| A-6 | `scope_type=task` | `curl -sS "$API/attention/items?project_id=$PROJECT_ID&scope_type=task&scope_id=$TASK_ID"` | 200 | `<fill>` | `<fill>` |
| A-7 | `scope_type=subtask` | `curl -sS "$API/attention/items?project_id=$PROJECT_ID&scope_type=subtask&scope_id=$SUBTASK_ID"` | 200 | `<fill>` | `<fill>` |
| A-8 | `state=open` | `curl -sS "$API/attention/items?project_id=$PROJECT_ID&scope_type=task&scope_id=$TASK_ID&state=open"` | 200 filtered | `<fill>` | `<fill>` |
| A-9 | `state=acknowledged` | `curl -sS "$API/attention/items?project_id=$PROJECT_ID&scope_type=task&scope_id=$TASK_ID&state=acknowledged"` | 200 filtered | `<fill>` | `<fill>` |
| A-10 | `state=resolved` | `curl -sS "$API/attention/items?project_id=$PROJECT_ID&scope_type=task&scope_id=$TASK_ID&state=resolved"` | 200 filtered | `<fill>` | `<fill>` |
| A-11 | missing `project_id` | `curl -i "$API/attention/items"` | 400 invalid_request | `<fill>` | `<fill>` |
| A-12 | invalid state | `curl -i "$API/attention/items?project_id=$PROJECT_ID&state=bad"` | fail-closed error | `<fill>` | `<fill>` |

#### `POST /attention/items`
| ID | Variant | Command | Expected | Result | Evidence |
|---|---|---|---|---|---|
| A-13 | required fields only | `curl -i -X POST "$API/attention/items" -H 'content-type: application/json' -d '{"project_id":"'$PROJECT_ID'","scope_type":"task","scope_id":"'$TASK_ID'","kind":"risk_note","summary":"http raise required only"}'` | 201 | `<fill>` | `<fill>` |
| A-14 | optional `body_markdown` | same + `body_markdown` | 201 | `<fill>` | `<fill>` |
| A-15 | optional `requires_user_action=true` | same + bool true | 201 | `<fill>` | `<fill>` |
| A-16 | optional `requires_user_action=false` | same + bool false | 201 | `<fill>` | `<fill>` |
| A-17 | invalid json | malformed body | 400 invalid_request | `<fill>` | `<fill>` |
| A-18 | unknown field | add unknown field | 400 invalid_request | `<fill>` | `<fill>` |
| A-19 | trailing payload | body + extra json | 400 invalid_request | `<fill>` | `<fill>` |
| A-20 | missing required fields (project/scope/kind/summary) | omit each (4 separate calls) | fail-closed errors | `<fill>` | `<fill>` |

#### `POST /attention/items/{id}/resolve`
| ID | Variant | Command | Expected | Result | Evidence |
|---|---|---|---|---|---|
| A-21 | empty body | `curl -i -X POST "$API/attention/items/$ATTN_ID_OPEN/resolve" -H 'content-type: application/json' -d ''` | 200 | `<fill>` | `<fill>` |
| A-22 | `resolved_by` only | payload with `resolved_by` | 200 | `<fill>` | `<fill>` |
| A-23 | `reason` only | payload with `reason` | 200 | `<fill>` | `<fill>` |
| A-24 | both optional fields | payload with both | 200 | `<fill>` | `<fill>` |
| A-25 | malformed json | malformed body | 400 invalid_request | `<fill>` | `<fill>` |
| A-26 | unknown id | random id | 404 not_found | `<fill>` | `<fill>` |

## 7) MCP Surface Coverage

### 7.1 Initialize + Tool Discovery
| ID | Check | JSON-RPC payload | Expected | Result | Evidence |
|---|---|---|---|---|---|
| M-1 | initialize | `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-05","clientInfo":{"name":"e2e","version":"1.0.0"}}}` | success | `<fill>` | `<fill>` |
| M-2 | tools/list | `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` | includes 4 tools: capture/list/raise/resolve attention | `<fill>` | `<fill>` |
| M-3 | stateless header check | inspect response headers | no `Mcp-Session-Id` header | `<fill>` | `<fill>` |

### 7.2 `kan.capture_state` (every option)
| ID | Variant | JSON args | Expected | Result | Evidence |
|---|---|---|---|---|---|
| M-4 | required only | `{"project_id":"$PROJECT_ID"}` | success | `<fill>` | `<fill>` |
| M-5 | `view=full` | + `view:"full"` | success | `<fill>` | `<fill>` |
| M-6 | `scope_type=project` | + scope tuple | success | `<fill>` | `<fill>` |
| M-7 | `scope_type=branch` | tuple | success | `<fill>` | `<fill>` |
| M-8 | `scope_type=phase` | tuple | success | `<fill>` | `<fill>` |
| M-9 | `scope_type=subphase` | tuple | success | `<fill>` | `<fill>` |
| M-10 | `scope_type=task` | tuple | success | `<fill>` | `<fill>` |
| M-11 | `scope_type=subtask` | tuple | success | `<fill>` | `<fill>` |
| M-12 | missing required `project_id` | `{}` | tool error | `<fill>` | `<fill>` |
| M-13 | invalid `view` | `view:"bad"` | invalid_request error | `<fill>` | `<fill>` |
| M-14 | unsupported `scope_type` | `scope_type:"bad"` | invalid_request error | `<fill>` | `<fill>` |
| M-15 | project scope mismatch | `scope_type:"project",scope_id:"wrong"` | invalid_request error | `<fill>` | `<fill>` |

### 7.3 `kan.list_attention_items` (every option)
| ID | Variant | JSON args | Expected | Result | Evidence |
|---|---|---|---|---|---|
| M-16 | required only | `project_id` | success | `<fill>` | `<fill>` |
| M-17 | with `scope_type/scope_id` per level | each level value | success | `<fill>` | `<fill>` |
| M-18 | `state=open` | + state | success | `<fill>` | `<fill>` |
| M-19 | `state=acknowledged` | + state | success | `<fill>` | `<fill>` |
| M-20 | `state=resolved` | + state | success | `<fill>` | `<fill>` |
| M-21 | missing `project_id` | `{}` | tool error | `<fill>` | `<fill>` |
| M-22 | invalid `state` | `state:"bad"` | invalid_request | `<fill>` | `<fill>` |

### 7.4 `kan.raise_attention_item` (every option)
| ID | Variant | JSON args | Expected | Result | Evidence |
|---|---|---|---|---|---|
| M-23 | required only | project/scope/kind/summary | success | `<fill>` | `<fill>` |
| M-24 | `body_markdown` set | + body_markdown | success | `<fill>` | `<fill>` |
| M-25 | `requires_user_action=true` | + bool true | success | `<fill>` | `<fill>` |
| M-26 | `requires_user_action=false` | + bool false | success | `<fill>` | `<fill>` |
| M-27 | missing required arg matrix | omit each required field one at a time | tool error each | `<fill>` | `<fill>` |

### 7.5 `kan.resolve_attention_item` (every option)
| ID | Variant | JSON args | Expected | Result | Evidence |
|---|---|---|---|---|---|
| M-28 | required only | `{id}` | success | `<fill>` | `<fill>` |
| M-29 | with `resolved_by` | + resolved_by | success | `<fill>` | `<fill>` |
| M-30 | with `reason` | + reason | success | `<fill>` | `<fill>` |
| M-31 | with both optional fields | + both | success | `<fill>` | `<fill>` |
| M-32 | missing required `id` | `{}` | tool error | `<fill>` | `<fill>` |
| M-33 | unknown `id` | random id | not_found/tool error | `<fill>` | `<fill>` |

## 8) Custom Endpoint Option Coverage
Validate serve flags themselves:

| ID | Check | Command | Expected | Result | Evidence |
|---|---|---|---|---|---|
| O-1 | custom bind/endpoint paths | restart server with `--http 127.0.0.1:18081 --api-endpoint /custom-api --mcp-endpoint /custom-mcp` | health + API + MCP work on custom paths | `<fill>` | `<fill>` |
| O-2 | endpoint collision fail-closed | set same path for api and mcp | startup error | `<fill>` | `<fill>` |

## 9) Guardrail/Fail-Closed Verifications
| ID | Scenario | Expected | Result | Evidence |
|---|---|---|---|---|
| G-1 | malformed JSON on HTTP mutation | 400 invalid_request | `<fill>` | `<fill>` |
| G-2 | unsupported scope tuple | invalid_request | `<fill>` | `<fill>` |
| G-3 | unknown route/tool arg failures | deterministic structured errors | `<fill>` | `<fill>` |
| G-4 | missing required MCP arg | tool error with clear message | `<fill>` | `<fill>` |

## 10) HTTP vs MCP Parity Checks
For the same target scope tuple, compare key fields returned by both surfaces.

| ID | Scope | Fields compared | Result | Evidence |
|---|---|---|---|---|
| P-1 | project | `scope_path`, `requested_scope_type`, `goal_overview.project_id` | `<fill>` | `<fill>` |
| P-2 | branch | same fields + attention counts | `<fill>` | `<fill>` |
| P-3 | phase | same fields + work overview | `<fill>` | `<fill>` |
| P-4 | subphase | same fields + work overview | `<fill>` | `<fill>` |
| P-5 | task | same fields + work overview | `<fill>` | `<fill>` |
| P-6 | subtask | same fields + work overview | `<fill>` | `<fill>` |

## 11) Final Gates
Run after all above tests:

```bash
just check
just ci
```

| Gate | Result | Evidence |
|---|---|---|
| just check | `<fill>` | `<fill>` |
| just ci | `<fill>` | `<fill>` |

## 12) Defects / Risks / Follow-ups
Record every failure or suspicious behavior.

### Defect D-1
- Summary: `<fill>`
- Repro steps: `<fill>`
- Expected: `<fill>`
- Actual: `<fill>`
- Severity: `<fill>`

### Defect D-2
- Summary: `<fill>`
- Repro steps: `<fill>`
- Expected: `<fill>`
- Actual: `<fill>`
- Severity: `<fill>`

## 13) Final Verdict
- Overall: `pass|fail|blocked`
- Ready for user sign-off: `yes|no`
- If `no`, list blockers:
  1. `<fill>`
  2. `<fill>`

