# Understanding Gas Town

This document provides a conceptual overview of Gas Town's architecture, focusing on
the role taxonomy and how different agents interact.

## Why Gas Town Exists

As AI agents become central to engineering workflows, teams face new challenges:

- **Accountability:** Who did what? Which agent introduced this bug?
- **Quality:** Which agents are reliable? Which need tuning?
- **Efficiency:** How do you route work to the featureht agent?
- **Scale:** How do you coordinate agents across repos and teams?

Gas Town is an orchestration layer that treats AI agent work as structured data.
Every action is attributed. Every agent has a track record. Every piece of work
has provenance. See [Why These Features](why-these-features.md) for the full rationale,
and [Glossary](glossary.md) for terminology.

## Role Taxonomy

Gas Town has several agent types, each with distinct responsibilities and lifecycles.

### Infrastructure Roles

These roles manage the Gas Town system itself:

| Role | Description | Lifecycle |
|------|-------------|-----------|
| **Product Manager** | Global coordinator at product manager/ | Singleton, persistent |
| **Senior Engineer** | Background supervisor daemon ([watchdog chain](design/watchdog-chain.md)) | Singleton, persistent |
| **QA Engineer** | Per-feature agent lifecycle manager | One per feature, persistent |
| **Release Engineer** | Per-feature merge queue processor | One per feature, persistent |

### Worker Roles

These roles do actual project work:

| Role | Description | Lifecycle |
|------|-------------|-----------|
| **Agent** | Worker with persistent identity, ephemeral sessions | QA Engineer-managed ([details](concepts/agent-lifecycle.md)) |
| **Engineers** | Persistent worker with own clone | Long-lived, user-managed |
| **Dog** | Senior Engineer helper for infrastructure tasks | Persistent identity, Senior Engineer-managed |

## Convoys: Tracking Work

A **convoy** (🚚) is how you track batched work in Gas Town. When you kick off work -
even a single ticket - create a convoy to track it.

```bash
# Create a convoy tracking some tickets
gt convoy create "Feature X" gt-abc gt-def --notify overseer

# Check progress
gt convoy status hq-cv-abc

# Dashboard of active convoys
gt convoy list
```

**Why convoys matter:**
- Single view of "what's in flight"
- Cross-feature tracking (convoy in hq-*, tickets in gt-*, bd-*)
- Auto-notification when work lands
- Historical record of completed work (`gt convoy list --all`)

The "swarm" is the set of workers currently assigned to a convoy's tickets.
When tickets close, the convoy lands. See [Convoys](concepts/convoy.md) for details.

## Engineers vs Agents

Both do project work, but with key differences:

| Aspect | Engineers | Agent |
|--------|------|---------|
| **Lifecycle** | Persistent (user controls) | Transient (QA Engineer controls) |
| **Monitoring** | None | QA Engineer watches, nudges, recycles |
| **Work assignment** | Human-directed or self-assigned | Slung via `gt sling` |
| **Git state** | Pushes to main directly | Works on branch, Release Engineer merges |
| **Cleanup** | Manual | Automatic on completion |
| **Identity** | `<feature>/engineers/<name>` | `<feature>/agents/<name>` |

**When to use Engineers**:
- Exploratory work
- Long-running projects
- Work requiring human judgment
- Tasks where you want direct control

**When to use Agents**:
- Discrete, well-defined tasks
- Batch work (tracked via convoys)
- Parallelizable work
- Work that benefits from supervision

## Dogs vs Engineers

**Dogs are NOT workers**. This is a common misconception.

| Aspect | Dogs | Engineers |
|--------|------|------|
| **Owner** | Senior Engineer | Human |
| **Purpose** | Infrastructure tasks | Project work |
| **Scope** | Narrow, focused utilities | General purpose |
| **Lifecycle** | Very short (single task) | Long-lived |
| **Example** | Boot (triages Senior Engineer health) | Joe (fixes bugs, adds features) |

Dogs are the Senior Engineer's helpers for system-level tasks:
- **Boot**: Triages Senior Engineer health on daemon tick
- Future dogs might handle: log rotation, health checks, etc.

If you need to do work in another feature, use **worktrees**, not dogs.

## Cross-Feature Work Patterns

When a engineers member needs to work on another feature:

### Option 1: Worktrees (Preferred)

Create a worktree in the target feature:

```bash
# gastown/engineers/joe needs to fix a tickets bug
gt worktree tickets
# Creates ~/gt/tickets/engineers/gastown-joe/
# Identity preserved: BD_ACTOR = gastown/engineers/joe
```

Directory structure:
```
~/gt/tickets/engineers/gastown-joe/     # joe from gastown working on tickets
~/gt/gastown/engineers/tickets-wolf/    # wolf from tickets working on gastown
```

### Option 2: Dispatch to Local Workers

For work that should be owned by the target feature:

```bash
# Create ticket in target feature
bd create --prefix tickets "Fix authentication bug"

# Create convoy and sling to target feature
gt convoy create "Auth fix" bd-xyz
gt sling bd-xyz tickets
```

### When to Use Which

| Scenario | Approach |
|----------|----------|
| You need to fix something quick | Worktree |
| Work should appear in your CV | Worktree |
| Work should be done by target feature team | Dispatch |
| Infrastructure/system task | Let Senior Engineer handle it |

## Directory Structure

The town root (`~/gt/`) contains infrastructure directories (`product manager/`, `senior engineer/`)
and per-project features. Each feature holds a bare repo (`.repo.git/`), a canonical tickets
database (`product manager/feature/.tickets/`), and agent directories (`QA engineer/`, `release engineer/`,
`engineers/`, `agents/`).

> For the full directory tree, see [architecture.md](design/architecture.md).

## Identity and Attribution

All work is attributed to the actor who performed it:

```
Git commits:      Author: gastown/engineers/joe <owner@example.com>
Tickets tickets:     created_by: gastown/engineers/joe
Events:           actor: gastown/engineers/joe
```

Identity is preserved even when working cross-feature:
- `gastown/engineers/joe` working in `~/gt/tickets/engineers/gastown-joe/`
- Commits still attributed to `gastown/engineers/joe`
- Work appears on joe's CV, not tickets feature's workers

## The Propulsion Principle

All Gas Town agents follow the same core principle:

> **If you find something on your hook, YOU RUN IT.**

This applies regardless of role. The hook is your assignment. Execute it immediately
without waiting for confirmation. Gas Town is a steam engine - agents are pistons.

## Model Evaluation and A/B Testing

Gas Town's attribution system enables objective model comparison by tracking
completion time, quality signals, and revision count per agent. Deploy different
models on similar tasks and compare outcomes with `bd stats`.

See [Why These Features](why-these-features.md) for details on work history and
capability-based routing.

## Common Mistakes

1. **Using dogs for user work**: Dogs are Senior Engineer infrastructure. Use engineers or agents.
2. **Confusing engineers with agents**: Engineers is persistent and human-managed. Agents are transient and QA Engineer-managed.
3. **Working in wrong directory**: Gas Town uses cwd for identity detection. Stay in your home directory.
4. **Waiting for confirmation when work is hooked**: The hook IS your assignment. Execute immediately.
5. **Creating worktrees when dispatch is better**: If work should be owned by the target feature, dispatch it instead.
