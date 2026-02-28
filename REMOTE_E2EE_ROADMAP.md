# Remote E2EE Architecture + Roadmap For `tillsyn`

Created: 2026-02-27  
Owner: product/architecture planning  
Status: proposed (design + execution plan)

## 1) Objective

Add optional remote collaboration for customer organizations while preserving:

1. local-first `tillsyn` UX and offline operation,
2. zero required cloud dependencies for OSS local-only users,
3. end-to-end encrypted (E2EE) customer project data such that the `tillsyn` service operator cannot read project/task/file content.

This roadmap covers two tracks:

1. OSS/runtime changes in this repo,
2. hosted remote-service buildout (control plane + encrypted sync plane).

## 2) Current Baseline (From Repo State)

1. Local-first SQLite is the current primary persistence model.
2. HTTP + MCP serve surfaces exist and are currently local/dev oriented.
3. Snapshot import/export exists (`tillsyn.snapshot.v1` JSON model).
4. Task metadata already supports resource references and project-root attachment boundaries.
5. Remote/team auth-tenancy and hardening are explicitly roadmap/deferred in active planning docs.

Primary repo references:

1. [README.md](README.md)
2. [PLAN.md](PLAN.md)
3. [internal/app/snapshot.go](internal/app/snapshot.go)
4. [internal/domain/workitem.go](internal/domain/workitem.go)
5. [internal/adapters/server/server.go](internal/adapters/server/server.go)

## 3) Required Product Constraints

1. The same `tillsyn` client must support:
    - local-only personal projects,
    - remote org projects synced from the cloud.
2. Remote org projects must live-update across authorized org members.
3. Org data content (project/task/comment/attachment payloads) must be unreadable to service operators.
4. Export/import must remain first-class and project-scoped portability must improve.
5. OSS local mode should remain simple (`just run`, local SQLite only), with remote mode as opt-in.

## 4) Architecture Decision

Use a **split-plane model**:

1. **Client data plane**: local SQLite database per user device.
2. **Remote control plane**: Postgres for identity/auth/account/org/membership/device key metadata, ACLs, sequencing metadata, billing/audit.
3. **Remote encrypted sync plane**: encrypted operation events + encrypted snapshots + encrypted file blobs stored server-side.
4. **Realtime transport**: websocket stream (or equivalent) for sequence notifications; clients pull encrypted deltas and apply locally.

Key rule:

1. Remote Postgres is **not** a plaintext mirror of project/task rows under strict E2EE.
2. The server routes/stores ciphertext and enforces access policy; clients decrypt and materialize state in local SQLite.

## 4.1 Locked Runtime + Transport Decisions (2026-02-27)

These are explicitly agreed implementation decisions for the next wave:

1. MCP transport in OSS runtime is HTTP-only for now (Streamable HTTP endpoint).
2. No stdio MCP transport in this immediate wave.
3. `till serve` remains the explicit headless runtime entrypoint.
4. `till` (TUI launch) must ensure local server availability:
    - if already running, reuse it and do not restart,
    - if not running, auto-start local server process and continue.
5. Default local server bind target is `127.0.0.1:5437`.
6. If `5437` is unavailable, auto-select another available local endpoint and surface it to the user.
7. Endpoint must be user-selectable from:
    - CLI launch flags/options,
    - TUI runtime settings.
8. HTTP-first transport is also the canonical path for:
    - future non-CLI clients,
    - hosted remote/org service endpoints,
    - browser/webapp-adjacent integration surfaces.

## 4.2 Why HTTP-Only First (Now)

1. Keeps one transport contract across local + remote.
2. Matches existing `tillsyn` implementation (`serve` is already headless; TUI is not required at runtime).
3. Reduces branching complexity in auth, observability, and test matrices.
4. Supports future non-CLI clients better than stdio-only.
5. Allows stdio to be added later as an optional compatibility transport if needed.

## 4.3 MCP-Go Viability + Known Limits (Planning Constraints)

1. `mcp-go` supports Streamable HTTP and stdio transports, and can run multiple transports in one process.
2. Current `tillsyn` implementation already uses Streamable HTTP in stateless mode for MCP.
3. HTTP-only requires a running server process; clients cannot talk to MCP when no process is bound.
4. Streamable HTTP is the right long-term path for remote/multi-client and web-facing integration.
5. Known limitation from `mcp-go` docs to track:
    - some advanced sampling-related behavior is not supported on Streamable HTTP.

## 5) Why This Fits Better Than Alternatives

### 5.1 Avoid “shared SQLite over network FS”

SQLite is excellent embedded storage, but not a network-shared database engine for concurrent multi-host writes.

### 5.2 Avoid plaintext server row model if strict E2EE is required

If server must query plaintext project content directly, the service can read customer data, violating strict E2EE.

### 5.3 Keep local mode simple

Local users still get no extra service dependencies. Remote collaboration becomes a separate opt-in capability.

## 6) Data Classification + Storage Plan

## 6.1 Control-plane data (plaintext in Postgres)

1. tenants/orgs and billing account status,
2. users, identities, sessions,
3. org memberships and roles,
4. device registrations + public keys,
5. project membership and ACL metadata,
6. event sequence indexes, idempotency keys, replay windows,
7. audit records for authz/membership/key lifecycle events.

## 6.2 Encrypted project data (ciphertext only server-side)

1. encrypted mutation events (small append-only units),
2. encrypted periodic project snapshots/checkpoints,
3. encrypted comment/task/project content payloads,
4. encrypted file manifests (metadata minimized; sensitive fields encrypted).

## 6.3 Encrypted file/blob data

1. images, PDFs, and other attachments are encrypted client-side before upload,
2. stored in object storage (S3-compatible) with immutable/object-versioned keys,
3. referenced by encrypted manifest events with integrity hashes.

## 7) Live Update Model (Detailed)

## 7.1 Write path

1. User edits remote project in TUI.
2. Client commits edit to local SQLite immediately (low latency UX).
3. Client generates deterministic mutation event:
    - `project_id`,
    - `client_event_id` (idempotency),
    - `base_seq` (last known remote seq),
    - encrypted payload,
    - signature/MAC metadata.
4. Server validates auth + org/project ACL + replay/idempotency.
5. Server persists encrypted event and assigns monotonic `project_seq`.
6. Server emits realtime notification with `{project_id, project_seq}` to subscribers.

## 7.2 Read/catch-up path

1. Client receives sequence notice (or polls if disconnected).
2. Client requests missing encrypted events since `last_applied_seq`.
3. Client decrypts, verifies integrity/signature, applies to local SQLite.
4. UI refreshes from local DB state.

## 7.3 Offline behavior

1. Client can continue local edits while offline.
2. On reconnect, client uploads queued encrypted events.
3. Conflict policy resolves divergent edits deterministically (see Section 9.5).

## 8) Local + Remote Coexistence in One TUI

Proposed project scope model in client:

1. `local:<project_id>` projects live only in local SQLite.
2. `remote:<org_id>:<project_id>` projects are locally materialized and sync-enabled.

Board/project picker behavior:

1. clear source badges (`local` / `remote`),
2. explicit online/sync status for remote projects,
3. failure states surface as attention/notices entries.

## 8.1 Local Runtime Modes

1. `till`:
    - launches TUI,
    - probes configured local HTTP server endpoint,
    - reuses existing running server if healthy,
    - otherwise auto-starts local serve process.
2. `till serve`:
    - explicit headless API/MCP server process,
    - useful for external MCP clients, scripting, and remote-sync-only sessions.
3. Future optional mode:
    - persistent daemon/service install (`launchd`/`systemd`) can be added later,
    - not required for initial rollout.

## 8.2 Endpoint Selection + Fallback Policy

1. Default bind:
    - `127.0.0.1:5437`.
2. If bind fails:
    - attempt deterministic fallback ports (for example `5438..5457`),
    - if fallback range exhausted, allocate an OS-assigned local port.
3. Chosen endpoint is written to runtime state for discovery by:
    - subsequent TUI launches,
    - local MCP client configuration helpers.
4. User controls:
    - CLI flag override for bind endpoint,
    - TUI settings screen to view/edit active endpoint.
5. Safety:
    - localhost bind only by default,
    - explicit non-local binds require deliberate user opt-in.

## 8.3 Local Server Supervisor Behavior

1. TUI startup uses health probe + lockfile/PID metadata checks to avoid duplicate server spawns.
2. If an existing process is serving expected endpoint, TUI attaches without restart.
3. If process is stale/unhealthy, TUI starts a fresh local server and updates runtime metadata.
4. TUI surfaces current endpoint and server status in runtime diagnostics/help.

## 9) Execution Plan (OSS Repo Track)

## 9.0 Phase R-CLI: Fang/Cobra Migration + Local Server Orchestration

Deliverables:

1. migrate CLI from `flag` package to Charm Fang + Cobra command tree,
2. command UX shape:
    - `till` (TUI with local-server ensure behavior),
    - `till serve` (headless server),
    - `till export` / `till import` / `till paths`.
3. deterministic help/usage behavior for root + subcommands,
4. local-server endpoint controls in CLI flags and config persistence,
5. TUI startup supervisor behavior (reuse running server; auto-start only when needed),
6. runtime endpoint diagnostics surfaced in both CLI and TUI.

Acceptance:

1. `till --help` and `till serve --help` are deterministic and side-effect free,
2. launching `till` with running server does not restart server process,
3. launching `till` without running server auto-starts server and surfaces endpoint,
4. default endpoint starts at `127.0.0.1:5437` with automatic fallback behavior,
5. regression tests cover supervisor/reuse/fallback flows.

## 9.1 Phase R0: Contract + Model Design (docs/spec only)

Deliverables:

1. remote sync protocol spec (`event envelope`, `seq`, `ack`, `idempotency`, `error taxonomy`),
2. E2EE key hierarchy spec (device keys, project data keys, wrapping/rotation),
3. file-manifest schema (`chunking`, hashes, content-type, size, encryption metadata),
4. explicit conflict resolution policy and replay semantics.

Acceptance:

1. spec reviewed with testable invariants,
2. mapped to existing domain/app boundaries,
3. no unresolved “who decrypts where” ambiguity.

## 9.2 Phase R1: Local Schema + Domain Prep

Deliverables:

1. local tables for remote sync state:
    - remote project registry,
    - per-project sync cursor,
    - outbound event queue,
    - inbound event dedupe ledger,
    - key-envelope cache.
2. domain structs for event envelope and file manifests.
3. snapshot version bump plan for remote metadata compatibility.

Acceptance:

1. migrations + tests pass,
2. existing local-only flows remain unchanged,
3. import/export backward compatibility behavior documented.

## 9.3 Phase R2: Client Sync Engine (No Hosted Hard Dependency Yet)

Deliverables:

1. sync engine module with:
    - queueing,
    - retry/backoff,
    - ordered apply,
    - idempotency,
    - checkpointing.
2. pluggable transport adapter interface (HTTP/MCP websocket/poll abstraction).
3. structured logs and notices integration for sync failures/recovery.

Acceptance:

1. deterministic integration tests for reconnect/resume scenarios,
2. crash-recovery safety for queued unapplied events,
3. no regressions in standard TUI flows.

## 9.4 Phase R3: File Attachment Remote Path

Deliverables:

1. preserve current local attachment behavior for local projects,
2. add remote attachment flow:
    - client-side encryption streaming,
    - pre-signed upload/download URLs,
    - encrypted file manifest event commit,
    - local cache policy.
3. size/quota guardrails and resumable upload behavior.

Acceptance:

1. image/pdf attach/open works across two devices in same org project,
2. service cannot render plaintext file content,
3. manifest integrity mismatches fail closed with clear recovery UX.

## 9.5 Phase R4: Conflict + Merge Strategy

Recommended policy for v1:

1. operation-based event model with deterministic reducers,
2. optimistic writes with base-seq precondition,
3. on divergence:
    - non-overlapping field edits auto-merge,
    - overlapping edits use deterministic last-writer policy plus conflict notice,
    - optional manual conflict resolution action in TUI.

Acceptance:

1. race tests across two+ clients show deterministic convergence,
2. conflict outcomes are auditable and explainable,
3. no hidden destructive overwrite.

## 9.6 Phase R5: Import/Export Evolution

Deliverables:

1. keep current JSON snapshot import/export path,
2. add project-scoped remote-aware exports:
    - decrypted portable export (user-controlled, local use/interoperability),
    - encrypted backup bundle (events + manifests + blobs metadata + key envelopes).
3. local DB-file export guidance based on SQLite backup mechanisms (safe copy workflow).

Acceptance:

1. export/import roundtrip validated for local-only and remote-linked projects,
2. unresolved root/path mappings fail closed with actionable errors,
3. portability docs updated with exact guarantees.

## 9.7 Phase R6: UX + Policy Hardening

Deliverables:

1. onboarding for remote org login/device registration,
2. clear separation of local and remote actions in command palette/help,
3. approval flow for key sharing/membership changes,
4. comprehensive operator/user runbooks.

Acceptance:

1. users can safely operate mixed local+remote projects without confusion,
2. guarded error paths always expose recovery next steps,
3. dogfooding worksheet coverage added for remote flows.

## 9.8 Parallel Subagent Lane Map (Required)

Each phase below is designed for single-branch parallel execution with non-overlapping lock scopes.

### Phase R-CLI lanes

1. `RCLI-L1` CLI framework lane:
    - scope: `cmd/till/**`, CLI tests, command/help docs.
2. `RCLI-L2` local-server supervisor lane:
    - scope: runtime process-management module(s), endpoint selection/fallback logic, related tests.
3. `RCLI-L3` TUI integration lane:
    - scope: TUI startup/runtime settings/status surfaces and tests.

### Phase R0-R2 lanes

1. `RDATA-L1` domain/app contracts:
    - scope: `internal/domain/**`, `internal/app/**`.
2. `RDATA-L2` storage/migrations:
    - scope: `internal/adapters/storage/sqlite/**`.
3. `RDATA-L3` transport adapters:
    - scope: `internal/adapters/server/**`, transport contract tests.

### Phase R3-R4 lanes

1. `RFILE-L1` encrypted attachment pipeline:
    - scope: file-manifest domain/app logic.
2. `RFILE-L2` upload/download adapter lane:
    - scope: transport + blob gateway integration.
3. `RFILE-L3` conflict/merge lane:
    - scope: reducer logic, replay/convergence tests.

### Phase R5-R6 lanes

1. `RUX-L1` import/export compatibility lane:
    - scope: snapshot/export/import code and tests.
2. `RUX-L2` onboarding/settings lane:
    - scope: TUI UX and config surfaces.
3. `RUX-L3` docs/runbook lane:
    - scope: `README.md`, MCP runbooks, dogfooding worksheets, policy docs.

Global lane rules:

1. workers run package-scoped checks only (`just test-pkg <pkg>`),
2. integrator runs `just check` and `just ci` before phase close,
3. no lane is closed without explicit acceptance evidence.

## 10) Hosted Service Roadmap (Kan Cloud Track)

## 10.1 Service components

1. Auth service (OIDC/session/token issuance),
2. Org/membership/ACL service (Postgres),
3. Sync API service (encrypted event ingest/fetch),
4. Realtime fanout service (websocket),
5. Blob gateway (presigned URL issuance, quota checks),
6. Key-management integration (wrapping keys, rotation orchestration),
7. Audit/observability pipeline.

## 10.2 Multi-tenant hardening priorities

1. RLS everywhere in Postgres control-plane tables,
2. strict tenant/org scoping in every endpoint path and query,
3. rate limits + replay protection + idempotency guarantees,
4. secure-by-default retention/backup/deletion workflows.

## 10.3 Release waves

1. Cloud Alpha:
    - small org pilot,
    - core project/task/comment sync,
    - basic file uploads (images/PDF),
    - manual key recovery policy.
2. Cloud Beta:
    - improved conflict tooling,
    - richer file handling (resume/chunk verification),
    - admin controls and audit views.
3. GA:
    - SLA/SLO targets,
    - billing integration,
    - regional data controls and documented incident response.

## 11) Alignment With Existing Active Backlog + Unresolved Findings

This roadmap does **not** replace active closeout priorities in current plan docs. It should be staged after current locked work.

## 11.1 Active `PLAN.md` alignment

1. Current immediate lock remains Phase 0 collaborative closeout.
2. Existing deferred Phase 5 already flags:
    - advanced import/export divergence reconciliation,
    - multi-user/team auth-tenancy and security hardening.
3. Remote roadmap should branch from that Phase 5 area after Phase 0-4 completion.

## 11.2 Remediation worksheet alignment

Before remote rollout, unresolved local/platform gaps should be closed first (examples from active remediation docs):

1. external mutation refresh behavior and notices/notification UX completeness,
2. logging discoverability and sink parity,
3. remaining MCP guardrail/contract mismatches.

Rationale:

1. remote sync amplifies correctness/observability needs,
2. unresolved local guardrail/visibility defects become harder to triage at remote scale.

References:

1. [COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md](COLLAB_E2E_REMEDIATION_PLAN_WORKLOG.md)
2. [COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md](COLLABORATIVE_POST_FIX_VALIDATION_WORKSHEET.md)

## 12) Initial Milestone Breakdown (Suggested)

1. Milestone M0 (2-3 weeks): R-CLI (Fang/Cobra migration + local server supervisor + endpoint controls)
2. Milestone M1 (2-4 weeks): R0 + R1
3. Milestone M2 (3-5 weeks): R2 core sync loop + convergence tests
4. Milestone M3 (3-5 weeks): R3 remote file path + encrypted manifests
5. Milestone M4 (3-4 weeks): R4 conflict tooling + R5 export/import hardening
6. Milestone M5 (2-3 weeks): R6 UX/policy hardening + dogfooding docs

## 13) Explicit Non-Goals For V1

1. Server-side plaintext content search across customer project data.
2. Server-side AI/analytics over decrypted project/task/file payloads.
3. Full CRDT framework adoption before operation-log convergence is validated.
4. Adding stdio MCP transport before HTTP-first runtime and remote/cloud path are stable.

## 14) Source References

## 14.1 External technical references

1. SQLite isolation + WAL:
    - https://sqlite.org/isolation.html
    - https://sqlite.org/wal.html
2. SQLite over network caveats:
    - https://sqlite.org/useovernet.html
3. SQLite backup/export mechanisms:
    - https://sqlite.org/backup.html
    - https://sqlite.org/lang_vacuum.html
4. SQLite session/changeset extension:
    - https://sqlite.org/sessionintro.html
5. PostgreSQL row-level security:
    - https://www.postgresql.org/docs/current/ddl-rowsecurity.html
6. PostgreSQL LISTEN/NOTIFY:
    - https://www.postgresql.org/docs/current/sql-listen.html
    - https://www.postgresql.org/docs/current/sql-notify.html
7. WebSocket protocol:
    - https://www.rfc-editor.org/rfc/rfc6455
8. S3 model references (consistency/presigned URLs/versioning):
    - https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html
    - https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html
    - https://docs.aws.amazon.com/AmazonS3/latest/userguide/Versioning.html
9. Streaming authenticated encryption reference:
    - https://doc.libsodium.org/secret-key_cryptography/secretstream
10. Optional SQLite-remote alternative references:

- https://docs.turso.tech/features/embedded-replicas/introduction
- https://litestream.io/getting-started/

11. MCP transport and architecture references:

- https://modelcontextprotocol.io/docs/learn/architecture
- https://modelcontextprotocol.io/specification/2025-03-26/basic/transports

12. Cobra documentation:

- https://github.com/spf13/cobra

13. Fang documentation:

- https://github.com/charmbracelet/fang

14. MCP-Go transport references:

- https://mcp-go.dev/transports/
- https://mcp-go.dev/transports/http
- https://mcp-go.dev/transports/stdio/
- https://mcp-go.dev/servers/advanced-sampling/

## 14.2 Context7 evidence used for this plan

1. SQLite docs corpus (`/websites/sqlite_cli`) for WAL/backup/session references.
2. PostgreSQL docs corpus (`/websites/postgresql_current`) for RLS policy semantics and operational guidance.
3. Fang docs corpus (`/charmbracelet/fang`) for Cobra integration and execution model.
4. Cobra docs corpus (`/spf13/cobra`) for command/flag/help behavior and migration planning.
