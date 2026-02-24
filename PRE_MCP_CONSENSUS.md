# Pre-MCP Consensus Register (Active)

Date: 2026-02-23
Status: Active consensus register for pre-MCP locks; MCP-wave execution is tracked separately.

Execution handoff note (2026-02-24):
- this file remains the locked decision register;
- MCP/HTTP implementation execution is now tracked in `MCP_DESIGN_AND_PLAN.md` under the temporary wave directive.

## 1) Purpose

Capture what is fully locked right now, what is intentionally roadmap-only, and which decisions are still open.
This file is meant to replace ambiguity and reduce context loss between discussions.

## 2) Source Inputs

- `PLAN.md` (non-worklog planning/architecture sections)
- `PRE_MCP_EXECUTION_WAVES.md`
- `PRE_PHASE11_CLOSEOUT_DISCUSSION.md` (decision register sections)
- `Pre_MCP_User_NOTES.md`
- Current code state in `internal/`, `cmd/`, `config.example.toml`

## 3) Scope Guard

- This register captures decisions locked before MCP implementation started.
- MCP/HTTP transport/tool execution now occurs in the active wave tracked by `MCP_DESIGN_AND_PLAN.md`.
- Advanced import/export transport-closure concerns remain roadmap-only during the active wave unless user re-prioritizes.
- No remote auth/tenancy implementation in this phase.
- Build now with extension points for future team-sharing and remote operation.

## 4) Locked Consensus (Build Now)

### 4.1 Terminology

- Use `kind` for node classification.
- Use `work item` as the generic node family term.
- Keep markdown source-of-truth for descriptions/comments in storage; render in TUI view-time.

### 4.2 Kinds Model

- Add DB-root `kind_catalog` as a dictionary of reusable kind definitions.
- Add project-scoped `allowed_kinds` references to `kind_catalog`.
- `kind_catalog` entries must include `applies_to` constraints.
- Baseline `applies_to` set: `project | branch | phase | task | subtask`.
- `n/a` is optional and only exists when explicitly created by user/orchestrator.
- Kinds/templates are cross-project reusable from day one.
- Kind definitions should support JSON-schema-driven validation payloads in DB (not hardcoded static enum lists).
- Kind/template definitions should also support template-intent metadata for:
  - auto-created actions/checklists/work items,
  - auto-fill generation for `AGENTS.md`/`CLAUDE.md` sections (initially planned, roadmap expansion later).
- Kinds must enforce hard validation rules:
  - CRUD write attempts with unknown/disallowed kinds fail before DB write.
  - Failures are logged and bubbled up as wrapped errors.

### 4.3 Dynamic Enum Contract (Important)

- Kinds are not compile-time Go enums because values are DB-defined at runtime.
- Implementation model:
  - Use string-backed domain types at API boundaries.
  - Validate against DB-loaded `kind_catalog` + project `allowed_kinds` on every write path before persistence.
  - Apply JSON-schema validation for kind-scoped payload/metadata where configured.
  - Treat validated values as runtime enums per DB state.
- JSON marshalling helps transport/storage only; it does not replace policy/schema validation.

### 4.4 Hierarchy Direction

- Baseline hierarchy remains: `Project -> Branch -> Phase -> Task -> Subtask`.
- MVP branch representation uses unified work-item modeling (`kind=branch`) with enforced invariants.
- Projects can be typed.
- Hierarchy and kind are used together:
  - parent and child can be different kinds,
  - child kind must still satisfy `applies_to` + policy constraints.
- Subphase/subbranch remain hierarchy + parent linkage, with room for small kind-specific metadata differences if needed for usability.
- TUI rendering/usability for nested structures is a first-class requirement.
- Parent/child navigation must re-render context clearly (breadcrumb/path intent) for branch/phase descent.

### 4.5 Templates + System Actions

- Kind templates can auto-create child work and metadata as a system action.
- Auto-generated children are editable by user and orchestrator.
- System-created items must be actor-attributed and auditable.
- Purpose of kind templates must be explicit in docs/tooling:
  - drive automatic operational scaffolding (for example tests/docs/git/workflow checks),
  - drive deterministic file-section autofill behavior for `AGENTS.md`/`CLAUDE.md` guidance.

### 4.6 Path/Root Contract (System Directory Paths)

- Project root is a real system filesystem directory path.
- One canonical writable project root for now.
- Root path must exist and must be a directory.
- If missing, return actionable error guidance (create/fix path) before continuing.
- Project roots are used for:
  - resource attachment resolution,
  - gatekeeping scope enforcement for agent file operations.
- Agent rule:
  - no resource attach outside allowed root scope for that agent level.
- Exception:
  - orchestrator can create new Kan project records from current working directory flow when allowed by policy.

### 4.7 Gatekeeping + Locking (Non-Auth Capability Model)

- Gatekeeping is strict operational locking, not soft advisory checks.
- Every agent mutation must provide valid `name/id` pair and scope token.
- Invalid/missing pair blocks before DB mutation.
- Blocked attempts are logged and bubbled up.
- Names are repeatable display metadata.
- `agent_instance_id` (token identity) is unique and authoritative.
- Overlapping orchestrators at same scope:
  - hard prevention by default,
  - override allowed only through explicit policy + explicit acknowledgement flow.
- Token lifecycle (MVP):
  - short-lived capability leases + heartbeat renewal,
  - expired lease blocks mutation calls until explicit renew/revive action.
- Revive behavior:
  - both orchestrators and subagents can be renewed/revived through explicit user-approved flow.
  - expiry cause and renewal requirement must be logged and bubbled up clearly.
- Scope delegation:
  - subagents are narrower-than-parent by default,
  - equal-scope delegation allowed only via explicit policy + warning path.
- Emergency safety:
  - one-shot revoke-all at project/branch scope is in MVP.

### 4.8 Override Safety

- Default behavior requires explicit user approval for overlap override.
- Override pathway must be actor-attributed and auditable.
- Project-level policy controls may allow orchestrator override behavior.
- Generated AGENTS/CLAUDE guidance should default to "ask user before override."
- MVP dangerous limitation (explicitly documented):
  - orchestrator calls may receive override token material based on project policy and user instructions,
  - system assumes orchestrator follows user policy in generated guidance.
- This limitation and recommended user practice must be called out in future MCP/HTTP README/tool docs.

### 4.9 Search/Matching Consistency

- Fuzzy behavior should be unified now (pre-MCP), including backend task search behavior.
- Avoid mixed substring-only backend semantics when TUI uses fuzzy ranking contracts.

### 4.10 HTTP/Tool Contract Direction

- REST/tool-style contracts first (not GraphQL in MVP).
- Summary-by-default responses with explicit expansion args.
- Comment/thread responses should carry enough task context for agent usefulness (description + relevant metadata + comments window).
- Descriptions/comments are markdown text fields and should be documented as markdown-write fields in tool contracts.
- This section remains the lock baseline; execution now proceeds in `MCP_DESIGN_AND_PLAN.md`.

### 4.11 Standards/Policy Profile in DB

- Project-level standards/policy data should be storable and discoverable in DB.
- This includes conventions like logging, error handling, testing style, architecture rules.
- This profile is intended to support AGENTS/CLAUDE file generation and updates.

### 4.12 Error Handling and Logging

- Use idiomatic wrapped errors (`%w`).
- Reject invalid gatekeeping/kind operations before persistence boundary.
- Log failures with context via `github.com/charmbracelet/log`.
- Bubble errors up to caller surfaces.

### 4.13 Thread Payload Default (Design-Ready Lock)

- For future MCP/HTTP payloads, default thread delivery should be recent-window + pagination.
- Full-history default behavior is not MVP; treat as roadmap/optional policy evolution.
- Response shape should support explicit expansion controls for deeper history retrieval.

### 4.14 JSON-Schema Execution Strategy (Best-Practice MVP Lock)

- Use a maintainable and testable runtime-validation pipeline:
  - validate kind/policy constraints first,
  - then apply JSON-schema payload validation,
  - then perform persistence write.
- Use compiled-schema caching keyed by stable schema identity (for example kind + schema hash/version) to avoid repeated compile overhead.
- Validation failures must return deterministic structured errors and be logged with context.
- Keep implementation intentionally simple and safe in MVP; roadmap can add advanced optimization/evolution controls.

### 4.15 Kind Metadata Breadth (Customizable MVP Lock)

- MVP should provide:
  - a minimal common metadata surface,
  - plus schema-validated extension payloads customizable by user/orchestrator per kind.
- Users/orchestrators can customize kind template behavior and metadata expectations through catalog definitions and project allowlists.

### 4.16 Mutation Guard Strictness (MVP Lock)

- Non-user mutations are strictly lease-gated by default when capability locking is enabled.
- Agent/system writes without a valid guard tuple (`agent_name`, `agent_instance_id`, `lease_token`) fail before persistence.
- Scope/identity/token mismatch fails before persistence and is logged with context.

## 5) Roadmap-Locked (Not MVP Build Target Yet)

### 5.1 Export/Import Portability

- Build MVP with export/import extensibility in mind.
- Later project export must include closure bundle of referenced kinds/templates (not IDs only).
- Import should fail on unresolved required root mappings and unresolved required references.
- Advanced MCP/HTTP transport closure concerns for import/export remain roadmap-only in the current wave (branch/commit divergence reconciliation and richer conflict tooling).
- Cross-OS guidance:
  - SQLite files are generally portable,
  - safest sharing path is snapshot/export workflows with explicit resolution steps.

### 5.2 Optional Path Expansion

- Keep one writable root now.
- Future option: additional read/search roots per project for reference-only workflows.

### 5.3 Override-Token Hardening

- Future policy/transport hardening should reduce accidental token abuse risks:
  - stronger conditional logic for when override material is exposed,
  - finer-grained policy controls and safer template defaults.

### 5.4 Kind Versioning

- Versioned kind/template history for old items is roadmap-only (not MVP).

### 5.5 Default Template Catalog Expansion (Agents/Claude)

- Built-in template library expansion is roadmap:
  - richer default sections,
  - guided placeholders like "talk to user and decide X/Y/Z, then edit/remove this block",
  - user-level controls for visibility, selection, and customization.

### 5.6 Advanced Team/Remote Security

- Current gatekeeping is non-auth capability control.
- Future remote/team user auth/tenancy extension is roadmap-only.

### 5.7 Agent Attention/Blocker Signaling (MCP-Phase Build Target)

- Add a DB-level attention/blocker model so agents can raise "cannot proceed without consensus/approval" signals on specific nodes.
- Attention entries must be level-scoped (`project|branch|phase|task|subtask`) and capability-gated for create/update/list access.
- TUI roadmap behavior for this model:
  - warning indicator in list rows for nodes with open attention/blocker entries,
  - small always-visible panel showing current-level attention items requiring user action,
  - filter integration through search, quick actions (`.`), and command palette (`:`).
- Future MCP/HTTP contracts must expose:
  - create/update/resolve attention calls per node,
  - list APIs with level filters + pagination/expansion controls,
  - actor attribution and audit metadata.
- All MCP tool definitions should include explicit escalation guidance:
  - when blocked on consensus/approval, raise node-scoped attention/blocker records using the attention tool surface.
- Template guidance direction:
  - `AGENTS.md` / `CLAUDE.md` templates should instruct orchestrators/subagents to raise attention signals when consensus/approval is required before proceeding.

## 6) Remaining Open Questions (Roadmap)

1. Post-MVP thread-delivery policy:
   - whether and when to allow full-history default by policy.
2. Post-MVP schema evolution controls:
   - richer version migration workflows, compatibility modes, and alternate validators.
3. Post-MVP typed metadata expansion:
   - whether to add richer first-class typed fields for branch/phase beyond extension payloads.
4. Attention schema shape:
   - dedicated `attention_items` table vs embedded JSON payload field strategy.
5. TUI attention panel scope:
   - exact panel placement/size behavior across board, thread, and modal-heavy contexts.
6. Attention severity/state taxonomy:
   - required enum set (`blocker|consensus|approval|risk`, etc.) and lifecycle states (`open|ack|resolved`).
7. MCP attention pagination defaults:
   - default recent-window size, cursor shape, and expansion semantics for full history.

## 7) Decision Notes for Next Discussion

- The kind model is interpreted as:
  - global reusable catalog entries,
  - project-scoped allowlists,
  - hard pre-write validation,
  - template-driven auto actions,
  - full actor attribution.
- Root paths are interpreted as system directories used for both attachment behavior and gatekeeping boundaries.

## 8) Final Task (Do Last)

- [x] Create `MCP_DESIGN_AND_PLAN.md` only after this pre-MCP register is sufficiently locked.
- [x] That MCP design/planning file must explicitly reconcile:
  - `PLAN.md`,
  - `PRE_PHASE11_CLOSEOUT_DISCUSSION.md`,
  - `PRE_MCP_EXECUTION_WAVES.md`,
  - `PRE_MCP_CONSENSUS.md`,
  - `Pre_MCP_User_NOTES.md`,
  - and current code/runtime state.
- [x] The MCP design file must include: contract shape, tool boundaries, gating/locking model, payload sizing rules, portability/import behavior assumptions, roadmap tie-ins, explicit open risks, and explicit explanation of kind-template purpose for auto-actions and `AGENTS.md`/`CLAUDE.md` autofill workflows.

## 9) Canonical Sources (Locked)

Use this source-of-truth split going forward:
- `PLAN.md`: primary roadmap and project intent ledger.
- `PRE_MCP_CONSENSUS.md` (this file): locked pre-MCP consensus decisions.
- `MCP_DESIGN_AND_PLAN.md`: Phase 11.0 design gate, open questions, and transport planning.

Execution checkpoint history was intentionally trimmed from this file to reduce context overload.
