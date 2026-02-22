# Parallel Agent Execution Runbook

This document describes a general-purpose paradigm for running multiple coding agents in parallel across many project types.
It is intentionally framework-agnostic and can be adapted to CLI apps, backend services, frontends, data tools, and infra repos.

## 1) Purpose
- Increase throughput with parallel agent lanes without sacrificing correctness.
- Prevent merge thrash and silent regressions when multiple agents touch one codebase.
- Provide predictable handling for permission-gated commands and tool failures.

## 2) Execution Modes
- `Mode A: Single branch, no worktrees`:
  - fastest startup.
  - highest collision risk.
  - requires strict lock ownership and integrator discipline.
- `Mode B: Multiple worktrees/branches`:
  - safer isolation.
  - more setup overhead.
  - preferred when available.

This runbook fully supports Mode A and scales to Mode B with minimal changes.

## 3) Role Model
- Orchestrator:
  - owns decomposition, lane assignment, lock registry, and global prioritization.
- Worker subagents:
  - implement one bounded lane each.
  - produce patch artifacts plus evidence of verification.
- Integrator:
  - only role that applies patches to the shared target branch.
  - runs gate checks and resolves conflicts.
- Human maintainer:
  - resolves approvals and policy questions.
  - signs off on final integrated state.

In small teams, one person/agent can play multiple roles, but responsibilities stay distinct.

## 4) Core Principles
- One lane, one responsibility.
- Small patches, frequent integration.
- Explicit lock ownership for files and directories.
- Integrator-only branch mutation in Mode A.
- Gate every applied patch with automated checks.
- Record every failure and remediation in a live worklog.

## 5) Work Decomposition Strategy
- Split by architecture boundaries first:
  - domain/model
  - application/service
  - adapters/storage
  - UI/presentation
  - config/docs/release.
- Prefer independent acceptance slices over large feature blobs.
- Flag hotspots early (large or frequently touched files) and serialize them.

### Decomposition Checklist
- Does the lane have a single acceptance target?
- Are touched files mostly disjoint from other lanes?
- Are test expectations clear for this lane?
- Is rollback simple if integration fails?

## 6) Lock Registry Model
- Keep an active lock table in the primary worklog (for example `PLAN.md`).
- Minimum lock fields:
  - `lock_id`
  - `owner`
  - `scope` (file globs)
  - `objective`
  - `start_time`
  - `heartbeat`
  - `expires_at`.
- Lock rules:
  - no edits outside lock scope.
  - no concurrent owners on hotspot files.
  - stale locks require explicit reclaim note.

## 6.1) Worklog Source-of-Truth Split
- For repos with multiple planning documents, explicitly designate:
  - one execution ledger (single-writer, checkpoint-by-checkpoint progress),
  - one or more decision registers (policy/consensus history).
- Execution ledger requirements:
  - lane locks, checkpoint ids, command/test evidence, failure/remediation notes, and completion markers.
  - only orchestrator/integrator writes this ledger in single-branch mode.
- Decision register requirements:
  - rationale, alternatives, unresolved questions, and policy locks.
  - must not be treated as live checkpoint log.
- Recommended bootstrap check:
  - fail parallel start if this split is not declared in project docs.

## 7) Lane Contract
Each worker lane should deliver:
- patch artifact (or exact file diff summary),
- touched file list,
- commands executed,
- test evidence,
- unresolved risks,
- follow-up notes for integration.

### Orchestrator Assignment Contract
- Before dispatching a worker lane, orchestrator prompt should include:
  - lane id and one bounded objective,
  - lock scope (in-scope globs + explicit out-of-scope),
  - concrete acceptance criteria,
  - architecture constraints and allowed dependency directions,
  - required test plan and command constraints,
  - explicit worker test scope: package-level checks via `just test-pkg <pkg>` for touched packages only,
  - explicit full-gate ownership: repo-wide suites (`just test`, `just ci`) are integrator/orchestrator responsibilities unless explicitly delegated,
  - documentation/comment expectations for touched code,
  - required evidence/handoff format,
  - required doc-source behavior:
    - Context7 consult before first code edit,
    - Context7 re-consult after any failing test/runtime error before the next edit,
    - fallback source recording when Context7 is unavailable.
- Prompts should explicitly forbid:
  - edits outside lock scope,
  - unapproved destructive actions,
  - undocumented deviations from architecture or test policy.

### Lane “Done” Criteria
- Acceptance objective met.
- Local package-scoped tests for touched packages pass (`just test-pkg ...`).
- No lock violations.
- Handoff notes are complete.

## 8) Integration Protocol (Mode A)
- Integrator applies one lane patch at a time.
- After each applied patch:
  - run targeted checks for touched areas,
  - fix immediate conflicts before applying next lane.
- End each wave with full-repo gate.
- If gate fails:
  - isolate failing lane via git diff and test scope,
  - revert only lane-specific patch if needed,
  - reopen lane with precise failure context.

## 9) Permission and Approval Handling
Some tools/subagents cannot prompt interactively for approvals.
Use a parent-loop escalation model:
- worker hits permission-gated action,
- action fails and reports to orchestrator,
- orchestrator asks human for exact approval,
- once approved:
  - orchestrator runs blocked command directly, or
  - reruns/resumes worker from the last checkpoint.

### Approval Hygiene
- Request narrow, prefix-scoped approvals.
- Log why approval was needed and what was run.
- Prefer least privilege and short-lived capability.

## 10) Checkpoint and Resume Model
- Every lane maintains checkpoints:
  - `checkpoint_id`
  - completed steps
  - pending steps
  - blockers
  - next command.
- On interruption:
  - resume from latest checkpoint instead of replaying entire lane.
- On retry:
  - include failure signature and remediation notes to avoid repeated dead ends.

## 11) Testing and Quality Gates
- Lane-level checks:
  - package-scoped tests and linters for touched areas.
- Wave-level checks:
  - broader suite after a batch of lanes.
- Final integration gate:
  - canonical CI-equivalent command.

### Suggested Gate Ladder
- `Gate 1`: format + static checks on touched files.
- `Gate 2`: targeted package tests.
- `Gate 3`: repo-wide validation/build.

## 12) Conflict Management
- Expected conflict classes:
  - textual merge overlap,
  - semantic interface drift,
  - test/golden artifact drift.
- Conflict process:
  - freeze new patch application,
  - resolve in integrator context,
  - rerun affected lane checks,
  - continue queue.

### Hotspot Tactics
- Serialize hotspots by lock.
- Extract interfaces early to reduce lane coupling.
- Split giant files before heavy parallelization when possible.

## 13) Safety and Risk Controls
- Never allow destructive repository operations without explicit approval.
- Keep audit trail for:
  - approvals,
  - destructive actions,
  - policy overrides.
- For autonomous/dangerous modes:
  - default disabled,
  - show persistent warning,
  - require strong attribution in change logs.

## 14) Observability and Throughput Metrics
- Track:
  - lane cycle time,
  - merge conflict rate,
  - gate pass/fail ratio,
  - approval-block frequency,
  - rework percentage.
- Use metrics to adjust:
  - lane size,
  - hotspot serialization,
  - decomposition strategy.

## 15) Reusable Templates

### Lock Entry Template
```markdown
| lock_id | owner | scope | objective | start | heartbeat | expires |
|---|---|---|---|---|---|---|
| L-001 | agent-a | internal/tui/* | search modal focus loop | 14:10 | 14:22 | 15:00 |
```

### Lane Handoff Template
```markdown
Lane: L-001
Checkpoint: CP-03
Objective: search modal focus loop
Files: internal/tui/model.go, internal/tui/model_test.go
Acceptance:
- [x] focus order matches spec
- [x] scope toggle keyboard-focusable
Architecture Compliance: no cross-layer boundary violations
Doc/Comment Compliance: updated for touched declarations/non-obvious logic
Commands: just test-pkg ./internal/tui
Result: pass
Risks: none
Next: integrator apply + full wave gate
```

### Failure/Remediation Template
```markdown
Failure: permission denied for <command prefix>
Impact: lane blocked at checkpoint CP-03
Requested approval: <exact scope>
Remediation: approval granted; command rerun by integrator; lane resumed
```

## 16) Recommended Rollout Pattern
- Step 1: pilot with 2-3 lanes and strict locking.
- Step 2: tune lane size and gate cadence from metrics.
- Step 3: expand to more lanes only after stable pass rates.
- Step 4: codify learned policies in project docs.

## 16.1) Required Project Bootstrap Before Parallel Runs
- Before running parallel agents in any repository:
  - update that repo's `AGENTS.md` (or equivalent agent policy file) to explicitly encode:
    - single-writer worklog policy,
    - lock ownership rules,
    - integrator-only patch application for single-branch mode,
    - permission-failure escalation loop,
    - gate requirements (lane checks + final integration gate),
    - explicit Context7 policy for worker prompts (pre-edit + post-failure re-check),
    - explicit worker test policy (`just test-pkg` for lanes, full gate by integrator),
    - orchestrator assignment contract requirements,
    - worker handoff evidence requirements.
- Do not start parallel execution until this policy is committed/accepted by maintainers.
- Rationale:
  - avoids implicit process drift,
  - keeps behavior reproducible across different projects and goals,
  - prevents worker agents from applying incompatible assumptions.

## 17) Anti-Patterns to Avoid
- Parallel edits on the same hotspot file without lock serialization.
- Applying multiple lane patches before any test/gate run.
- Missing handoff artifacts.
- Large multi-feature patches with unclear acceptance.
- Ignoring permission failures and “manual guess” continuation.

## 18) Resource Links
- OpenAI Codex multi-agent docs:
  - https://developers.openai.com/codex/multi-agent
- OpenAI Codex worktrees docs:
  - https://developers.openai.com/codex/app/worktrees
- Codex app introduction:
  - https://openai.com/index/introducing-the-codex-app/
- OpenAI Codex API docs:
  - https://developers.openai.com/codex
- Git worktree reference:
  - https://git-scm.com/docs/git-worktree
- Trunk-based development overview:
  - https://trunkbaseddevelopment.com/
