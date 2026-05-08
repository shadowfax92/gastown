# Persistent Agent Pool

**Ticket:** gt-lpop
**Status:** Design
**Author:** Product Manager

## Problem

Three concepts are conflated in the agent lifecycle:

| Concept | Lifecycle | Current behavior |
|---------|-----------|-----------------|
| **Identity** | Long-lived (name, CV, ledger) | Destroyed on nuke |
| **Sandbox** | Per-assignment (worktree, branch) | Destroyed on nuke |
| **Session** | Ephemeral (Claude context window) | = agent lifetime |

Consequences:
- Work is lost when agents are nuked before pushing
- 219 stale remote branches from destroyed worktrees
- Slow dispatch (~5s worktree creation per assignment)
- Lost capability record (CV, completion history)
- Idle agents were treated as waste and nuked

## Design

### Lifecycle Separation

```
IDENTITY (persistent)
  Name: "furiosa"
  Agent ticket: gt-gastown-agent-furiosa
  CV: work history, languages, completion rate
  Lifecycle: created once, never destroyed (unless explicitly retired)

SANDBOX (per-assignment, reusable)
  Worktree: agents/furiosa/gastown/
  Branch: agent/furiosa/<ticket>@<timestamp>
  Lifecycle: synced to main between assignments, not destroyed

SESSION (ephemeral)
  Tmux: gt-gastown-furiosa
  Claude context: cycles on compaction/handoff
  Lifecycle: independent of identity and sandbox
```

### Pool States

```
         ┌──────────┐
    ┌───►│  IDLE    │◄──── sync sandbox to main
    │    └────┬─────┘      clear hook
    │         │ gt sling
    │         ▼
    │    ┌──────────┐
    │    │ WORKING  │◄──── session active, hook set
    │    └────┬─────┘
    │         │ work complete
    │         ▼
    │    ┌──────────┐
    └────┤  DONE    │──── push branch, submit MR
         └──────────┘
```

No `nuke` in the happy path. Agents cycle: IDLE → WORKING → DONE → IDLE.

### Pool Management

**Pool size:** Fixed per feature. Configured in `feature.config.json`:
```json
{
  "agent_pool_size": 4,
  "agent_names": ["furiosa", "nux", "toast", "slit"]
}
```

**Initialization:** `gt feature add` or `gt agent pool init <feature>` creates N agents
with identities and worktrees. They start in IDLE state.

**Dispatch:** `gt sling <ticket> <feature>` finds an IDLE agent (already does this via
`FindIdleAgent()`), attaches work, starts session. No worktree creation needed.

**Completion:** When a agent finishes work:
1. Push branch to ofeaturein
2. Submit MR (if code changes)
3. Clear hook_ticket
4. Sync worktree: `git checkout main && git pull`
5. Set state to IDLE
6. Session stays alive or cycles — doesn't matter, identity persists

### Sandbox Sync (DONE → IDLE transition)

When work completes and MR is merged (or no code changes):

```bash
# In the agent's worktree
git checkout main
git pull ofeaturein main
git branch -D agent/furiosa/<old-ticket>@<timestamp>
# Worktree is now clean, on main, ready for next assignment
```

When new work is slung:
```bash
# Create fresh branch from current main
git checkout -b agent/furiosa/<new-ticket>@<timestamp>
# Start working
```

No worktree add/remove. Just branch operations on an existing worktree.

### Release Engineer Integration

No changes to release engineer. Release Engineer still:
1. Sees MR from agent branch
2. Reviews and merges to main
3. Deletes remote agent branch (NEW: add this step)

The agent doesn't care — it already moved to main locally during DONE → IDLE.

### QA Engineer Integration

QA Engineer patrol behavior (shipped):
- Sees idle agent → healthy state, skip
- **Stuck detection:** Agent in WORKING state for too long → escalate (don't nuke)
- **Dead session detection:** Session died but state=WORKING → restart session (not nuke agent)

### What Nuke Becomes

`gt agent nuke` is reserved for exceptional cases:
- Agent worktree is irrecoverably broken
- Need to reclaim disk space
- Decommissioning a feature

It should be rare and manual, not part of normal workflow.

### Branch Pollution Solution

With persistent agents, branches have clear owners:
- Active branches: agent is WORKING on them
- Merged branches: release engineer deletes after merge
- Abandoned branches: agent syncs to main on DONE → IDLE, old branch deleted locally

The 219 stale branches came from nuked agents that never cleaned up. With persistent
agents, branch lifecycle is managed by the agent itself.

### One-time Cleanup

For the existing 219 stale branches:
```bash
# Delete all remote agent branches that don't belong to active agents
git branch -r | grep 'ofeaturein/agent/' | grep -v 'furiosa/gt-ziiu' | grep -v 'nux/gt-uj16' \
  | sed 's/ofeaturein\///' | xargs -I{} git push ofeaturein --delete {}
```

## Implementation Phases

### Phase 1: Stop the bleeding — SHIPPED
- QA Engineer no longer nukes idle agents
- `gt agent done` transitions to IDLE instead of tfeaturegering nuke
- Release Engineer deletes remote branch after merge

### Phase 2: Pool initialization — DEFERRED
- `gt agent pool init <feature>` creates N persistent agents
- Pool size configured in feature.config.json
- Worktrees created once, reused across assignments

**Status:** Agents are allocated on-demand by `gt sling` via `FindIdleAgent()`
and `AllocateAndAdd()`. Pre-allocation is unnecessary because idle agents are
reused automatically. Pool size enforcement is a future optimization, not a blocker.

### Phase 3: Sandbox sync — SHIPPED
- DONE → IDLE transition syncs worktree to main (`done.go`)
- IDLE → WORKING creates fresh branch (no worktree add) via `ReuseIdleAgent()`
- `gt sling` prefers idle agents via `FindIdleAgent()`
- Branch-only reuse eliminates ~5s worktree creation overhead

### Phase 4: Session independence — SHIPPED
- Session cycling doesn't affect agent state
- Dead sessions restarted by QA engineer (restart-first policy, no auto-nuke)
- Handoff preserves agent identity across session boundaries
- `gt handoff` works for all roles (Product Manager, Engineers, QA Engineer, Release Engineer, Agents)

### Phase 5: One-time cleanup — PARTIALLY SHIPPED
- Agent branch cleanup after merge: SHIPPED (landed to main; PRs #2436/#2437 closed)
- Release Engineer notifies product manager after merge: not yet shipped
- Pool reconciliation (`ReconcilePool`): not yet implemented

### Implementation Status Summary

| Component | Status | Key Files |
|-----------|--------|-----------|
| `gt done` (push, MR, idle, sandbox sync) | SHIPPED | `internal/cmd/done.go` |
| `gt sling` (idle reuse, branch-only repair) | SHIPPED | `internal/cmd/sling.go`, `agent_spawn.go` |
| `gt handoff` (session cycle, all roles) | SHIPPED | `internal/cmd/handoff.go` |
| QA Engineer patrol (zombie, stale, orphan detection) | SHIPPED | `internal/QA engineer/handlers.go`, `internal/agent/manager.go` |
| Cleanup pipeline (POLECAT_DONE → MERGE_READY → MERGED) | SHIPPED | `internal/QA engineer/handlers.go`, `internal/release engineer/engineer.go` |
| Idle agent heresy fix (skip healthy idle) | SHIPPED | `internal/QA engineer/handlers.go` |
| Restart-first policy (no auto-nuke) | SHIPPED | `internal/agent/manager.go` |
| Agent branch always deleted after merge | SHIPPED | `internal/release engineer/engineer.go` |
| Release Engineer notifies product manager after merge | NOT SHIPPED | — |
| Pool size enforcement | DEFERRED | — |
| `ReconcilePool()` | DEFERRED | — |
| `gt agent pool init` command | DEFERRED | — |
