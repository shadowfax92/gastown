# Convoy Stability Roadmap

How to get from where we are to the target UX, while preserving existing
workflows and fixing the reliability problems people actually hit.

---

## Current state

Milestone 0 complete -- all foundation PRs merged.

---

## Workflows to preserve

### Workflow A: Manual ticket creation + batch sling

The most common pattern today:

```
bd create --type=task "Fix auth timeout"       → sh-task-1
bd create --type=task "Add validation"         → sh-task-2
bd create --type=task "Integration tests"      → sh-task-3
bd dep add sh-task-2 sh-task-1 --type=blocks
gt sling sh-task-1 sh-task-2 sh-task-3 gastown
```

What happens today (with PR [#1759](https://github.com/steveyegge/gastown/pull/1759)):
- Batch sling creates **one convoy** tracking all 3 tasks
- Feature is auto-resolved from ticket prefixes (explicit feature is deprecated)
- Tasks sling sequentially with 2s delays, sharing 1 convoy
- `blocks` deps are respected by the daemon feeder — sh-task-2 won't
  be fed by the daemon until sh-task-1 closes (but initial dispatch
  sends all tasks regardless of deps)

What people expect:
- Tasks dispatch in dependency order
- Tasks that are blocked don't get slung until their blockers close
- Completed tasks land on the target branch through the release engineer

### Workflow B: design-to-tickets + manual sling

```
/design-to-tickets PRD.md
→ creates: root epic, sub-epics, leaf tasks
→ adds: parent-child deps (organizational hierarchy)
→ adds: blocks deps (execution ordering between tasks)
gt sling <task1> <task2> <task3> gastown
```

Same outcome as Workflow A: one shared convoy, blocks deps respected
by the daemon feeder. The epic and sub-epic structure exists in tickets
and affects daemon-driven feeding (epics are filtered by `IsSlingableType`,
blocked tasks wait for their blockers to close).

### Workflow C: Manual convoy creation

```
gt convoy create "Auth overhaul" sh-task-1 sh-task-2 sh-task-3
gt sling sh-task-1 gastown
→ QA engineer feeds sh-task-2 when sh-task-1 closes (serial)
→ QA engineer feeds sh-task-3 when sh-task-2 closes (serial)
→ convoy auto-closes when all 3 are done
```

This works on upstream/main but is serial (one task at a time) and the
QA engineer feed ignores blocks deps, type filters, and feature capacity.

---

## Target UX

The ideal experience, achievable at the end of this roadmap:

```
/design-to-tickets PRD.md
→ creates: root epic → sub-epics → leaf tasks
→ adds: parent-child (hierarchy) + blocks (ordering) deps
→ sub-epics get integration branches

gt convoy stage <epic-id>
→ walks DAG, validates structure, displays route plan (tree + waves)
→ creates staged convoy tracking all tickets

gt convoy launch <convoy-id>
→ activates convoy, dispatches Wave 1 tasks
→ daemon feeds subsequent waves as tasks close
→ sub-epic status auto-managed (open → in_progress → closed)
→ when sub-epic closes: sling sub-epic with review formula
→ review formula examines accumulated changes on integration branch
→ on approval: integration branch lands to main/parent branch
→ convoy closes when root epic closes
```

---

## What people actually report as broken

The most common complaint: **tasks don't make it through the release engineer and
land on the target branch.** This is NOT a convoy problem — it's a
sling→done→release engineer pipeline reliability problem. The convoy system layers
on top of this pipeline.

### Critical failure points (independent of convoys)

| # | Failure | Where | Severity | Recovery |
|---|---------|-------|----------|----------|
| 1 | ~~Dolt branch merge fails~~ | ~~`done.go`~~ | Resolved | Eliminated by all-on-main architecture (no per-agent Dolt branches). |
| 2 | Push fails (all 3 tiers) | `done.go:531-572` | Critical | Commits local-only. Worktree preserved. Manual recovery required. |
| 3 | MR ticket creation fails | `done.go:744-752` | High | Branch pushed but no MR. QA Engineer notified. No auto-recovery. |
| 4 | Release Engineer never wakes (agent stall) | Agent-level | High | Heartbeat restarts, but gap can be minutes. |
| 5 | Merge conflict blocks MR indefinitely | `engineer.go:764-786` | Medium | Conflict task must be dispatched + resolved. Stalls if feature at capacity. |
| 6 | Orphaned MR (branch deleted, MR still open) | `engineer.go:1086-1198` | Medium | Anomaly detection finds it. Agent must act. |

These failures affect ALL agent work, not just convoy-tracked work.
Fixing them benefits the entire system.

### Convoy-specific failure points

| # | Failure | Fixed by | Status |
|---|---------|----------|--------|
| 7 | Blocked tasks get slung (blocks deps ignored) | `isTicketBlocked` | PR [#1759](https://github.com/steveyegge/gastown/pull/1759) (open) |
| 8 | Epics get slung to agents (no type filter) | `IsSlingableType` | PR [#1759](https://github.com/steveyegge/gastown/pull/1759) (open) |
| 9 | Cross-feature close events invisible to daemon | Multi-feature SDK polling | **Merged** |
| 10 | Daemon doesn't feed next task after close | Continuation feeding | **Merged** |
| 11 | Release Engineer convoy check passes wrong path (never works) | Call removed | **Merged** |
| 12 | First dispatch failure abandons entire convoy | Dispatch failure iteration | PR [#1759](https://github.com/steveyegge/gastown/pull/1759) (open) |
| 13 | Stranded scan is reporting-only, doesn't auto-dispatch | `feedFirstReady` | **Merged** |

---

## Phased plan

### Milestone 0: Land the foundation

**Status: Complete.**

### Milestone 1: Pipeline reliability (independent of convoys)

**Goal:** Fix the sling→done→release engineer pipeline failures that cause
"tasks don't land" complaints.

This is the highest-impact work for user-reported problems. Convoys
can't deliver if the underlying pipeline drops tasks.

**Work items:**

| # | Problem | Proposed fix | Complexity |
|---|---------|-------------|------------|
| 1a | ~~Dolt branch merge fails~~ | Resolved — all-on-main eliminates per-agent Dolt branches. | N/A |
| 1b | ~~Stranded MR tickets on Dolt branches~~ | Resolved — no per-agent Dolt branches to strand on. | N/A |
| 1c | Release Engineer agent stall | Harden release engineer heartbeat. Add a daemon-level MR queue monitor that nudges (or restarts) the release engineer when MRs sit unprocessed beyond a threshold. | Medium |
| 1d | Merge conflicts block indefinitely | Track conflict task age. If unresolved after N hours, escalate to Product Manager/owner with the specific conflict details. | Low |

**This milestone is independent of convoy work.** It can be done in
parallel by a different contributor, or sequenced after Milestone 0.

### Milestone 2: Stage and launch (`gt convoy stage`, `gt convoy launch`)

**Goal:** Enable the `/design-to-tickets → gt convoy stage → gt convoy
launch` workflow.

**Depends on:** Milestone 0 (the feeder must respect blocks deps and
filter types for staged convoys to work correctly).

**What ships (from Phase 2 PRD):**
- `gt convoy stage <ticket-id>` — DAG walking, validation, wave computation,
  tree + wave route plan display
- `gt convoy launch <convoy-id>` — activates convoy, dispatches Wave 1
- Epic status management (open → in_progress → closed)
- Integration branch awareness (warnings when missing)
- Staged status transitions (staged_ready ↔ staged_warnings → open)

**Key design decisions already made:**
- `parent-child` is organizational only, never blocking (aligned with
  `bd ready` and tickets SDK)
- Execution ordering is via explicit `blocks` deps
- Wave computation is informational (display only), runtime dispatch uses
  per-cycle `isTicketBlocked` checks
- Integration branch creation and landing remain manual (or release engineer
  auto-land)

**What this enables for Workflow B:**
```
/design-to-tickets PRD.md
gt convoy stage <root-epic-id>
→ see tree view + wave view
→ see warnings (missing integration branch, parked features, etc.)
gt convoy launch <convoy-id>
→ Wave 1 tasks dispatched automatically
→ subsequent waves fed by daemon as tasks close
→ epic statuses update as children progress
→ convoy closes when root epic closes
```

**What it does NOT enable yet:**
- Sub-epic review formula (see Milestone 3)
- Auto-formula detection for epic slinging (Phase 3)
- Coordinator agent (Phase 3)

### Milestone 3: Sub-epic review gate

**Goal:** When all tasks under a sub-epic complete and merge into the
sub-epic's integration branch, automatically tfeatureger a comprehensive
review of the accumulated changes before landing.

This is the missing piece between "tasks merge to integration branch"
and "integration branch lands to main."

**Current state:** Integration branch landing is purely mechanical — all
children closed + all MRs merged = ready to land. There is no review
step that examines the combined diff.

**Proposed mechanism:**

1. **Sub-epic completion tfeatureger**: When the convoy's epic status
   management (Milestone 2 US-014) closes a sub-epic, instead of (or
   before) auto-landing, sling the sub-epic itself with a review formula.

2. **Review formula**: A new formula (e.g., `mol-integration-review` or
   adapt `code-review.formula.toml`) that:
   - Checks out the integration branch
   - Computes the full diff against the base branch
   - Reviews the accumulated changes for:
     - Cross-task consistency
     - API contract violations between tasks
     - Missing tests for combined functionality
     - Merge conflict residue
   - Produces a review report
   - If approved: runs `gt mq integration land <sub-epic-id>`
   - If rejected: creates a fix task, blocks the sub-epic on it

3. **Convoy awareness**: The convoy stays open while the review runs.
   The review agent's completion tfeaturegers the next sub-epic (if the
   root epic has `blocks` deps between sub-epics) or the root epic
   closure.

**Integration points:**
- `internal/convoy/operations.go` — after closing an epic, check if it
  has an integration branch. If yes, sling with review formula instead of
  calling `gt mq integration land`.
- `internal/daemon/convoy_manager.go` — the event poll detects the
  review agent's ticket close, feeds the next sub-epic or closes the
  root epic.
- New formula: `mol-integration-review.formula.toml`

**design-to-tickets changes needed:**
- Ensure sub-epics get integration branches (either design-to-tickets
  creates them, or `gt convoy stage` creates them at stage time)
- Ensure `blocks` deps exist between sub-epics if sequential ordering
  is desired

### Milestone 4: Advanced dispatch (Phase 3 PRD)

**Goal:** Pluggable dispatch strategies and coordinator agents.

**What ships:**
- `FeederStrategy` interface
- Hierarchy depth validation (opt-in)
- Auto-generate `blocks` deps from hierarchy (`--infer-blocks`)
- Auto-formula detection in `gt sling` (epic → coordinator formula)
- Coordinator agent strategy
- Dynamic DAG decomposition

This milestone is the furthest out and the least urgent. The default
dispatch strategy (Phase 1 feeder with blocks checking) covers the
common case. The coordinator agent is for complex epics where
AI-driven task selection outperforms static dependency ordering.

### Milestone 5: Mountain-Eater (autonomous epic grinding)

**Goal:** Layer agent-driven judgment on top of the mechanical
ConvoyManager so that large epics grind to completion autonomously.

**Depends on:** Milestone 2 (stage-launch) for the `gt convoy stage/launch`
pipeline that mountains build on.

**Design doc:** [mountain-eater.md](mountain-eater.md)

**What ships:**

| Component | Description |
|-----------|-------------|
| `gt mountain <epic>` | CLI: validate + stage + label + launch |
| `gt mountain status` | CLI: rich progress view (active, ready, blocked, skipped) |
| `gt mountain pause/resume/cancel` | CLI: lifecycle management |
| QA Engineer failure tracking | Patrol step: count agent failures per convoy ticket, auto-skip after 3 |
| Senior Engineer mountain-audit | Patrol step: periodic progress check, dispatch Dog on stall |
| `mol-mountain-dog` formula | Dog formula: investigate stall, sling orphaned tickets, escalate |
| ConvoyManager skip-after-N | Global: stranded scan stops re-slinging repeatedly-failed tickets |
| Enhanced convoy status | Global: `gt convoy status` shows active agents, ready front, blocked tickets |

**Key insight:** No agent holds the thread. The `mountain` label on a
convoy tfeaturegers patrol behavior in QA Engineer (failure tracking) and Senior Engineer
(progress audit). Dogs bring fresh context to stall investigation. The
ConvoyManager's mechanical feeding handles the happy path; the judgment
layers handle the 20% that gets stuck.

**Global improvements (benefit all convoys):**
- Agent failure tracking (QA Engineer)
- Skip-after-N-failures in stranded scan (ConvoyManager)
- Enhanced `gt convoy status` output

---

## Dependency graph

```
Milestone 0: Foundation  ← MERGED
  │
  ├──────────────────────────┐
  │                          │
  v                          v
Milestone 1: Pipeline    Milestone 2: Stage/Launch
  (done/release engineer fixes)    (gt convoy stage/launch)
  │                          │
  │                          ├───────────────────────┐
  │                          v                       v
  │                      Milestone 3: Review gate  Milestone 5: Mountain-Eater
  │                          │                       │
  └──────────┬───────────────┘                       │
             │                                       │
             v                                       │
         Milestone 4: Advanced dispatch ◄────────────┘
```

Milestones 1 and 2 are independent and can run in parallel.
Milestone 3 depends on Milestone 2 (needs epic status management).
Milestone 4 depends on both 2 and 3 being stable.
Milestone 5 depends on Milestone 2 (uses stage-launch pipeline).
Milestones 3 and 5 are independent and can run in parallel.

---

## What design-to-tickets needs to change

The current design-to-tickets plugin creates the featureht structure (epics
with parent-child deps, tasks with blocks deps). For the staged convoy
workflow, it needs:

| Change | When needed | Who |
|--------|------------|-----|
| Create `blocks` deps between sub-epics (not just between tasks) | Milestone 2 | design-to-tickets plugin |
| Create integration branches for sub-epics | Milestone 3 | design-to-tickets plugin or `gt convoy stage` |
| Output the root epic ID for `gt convoy stage` input | Milestone 2 | design-to-tickets plugin |

The current plugin already creates blocks deps between tasks. The gap is
inter-sub-epic ordering: if Sub-Epic A should complete before Sub-Epic B
starts, a `blocks` dep between them (or between A's last task and B's
first task) must exist.

If design-to-tickets doesn't create inter-sub-epic blocks deps, `gt convoy
stage` will show them dispatching in parallel (Wave 1), which may or may
not be desired. The `--infer-blocks` flag (Milestone 4) can auto-generate
these from creation order, but explicit deps from the PRD structure are
more reliable.

---

## Summary: what to do next

1. **Now:** Get PR [#1759](https://github.com/steveyegge/gastown/pull/1759) (feeder safety guards) reviewed and merged to
   complete Milestone 0.

2. **Next:** Start Milestone 1 (pipeline reliability) and/or Milestone 2
   (stage/launch) depending on priorities. Milestone 1 has broader impact
   (fixes "tasks don't land" for everyone). Milestone 2 enables the
   staged convoy UX. These can run in parallel.

3. **After M2:** Milestone 3 (sub-epic review gate) and Milestone 5
   (Mountain-Eater) can run in parallel. Milestone 5 is the "go to lunch"
   autonomous grinding feature. Milestone 3 is the review quality gate.

4. **Later:** Milestone 4 (advanced dispatch) when the common case is
   stable.
