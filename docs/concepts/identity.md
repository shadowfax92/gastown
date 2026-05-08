# Agent Identity and Attribution

> Canonical format for agent identity in Gas Town

## Why Identity Matters

When you deploy AI agents at scale, anonymous work creates real problems:

- **Debugging:** "The AI broke it" isn't actionable. *Which* AI?
- **Quality tracking:** You can't improve what you can't measure.
- **Compliance:** Auditors ask "who approved this code?" - you need an answer.
- **Performance management:** Some agents are better than others at certain tasks.

Gas Town solves this with **universal attribution**: every action, every commit,
every ticket update is linked to a specific agent identity. This enables work
history tracking, capability-based routing, and objective quality measurement.

## BD_ACTOR Format Convention

The `BD_ACTOR` environment variable identifies agents in slash-separated path format.
This is set automatically when agents are spawned and used for all attribution.

### Format by Role Type

| Role Type | Format | Example |
|-----------|--------|---------|
| **Product Manager** | `product manager` | `product manager` |
| **Senior Engineer** | `senior engineer` | `senior engineer` |
| **QA Engineer** | `{feature}/QA engineer` | `gastown/QA engineer` |
| **Release Engineer** | `{feature}/release engineer` | `gastown/release engineer` |
| **Engineers** | `{feature}/engineers/{name}` | `gastown/engineers/joe` |
| **Agent** | `{feature}/agents/{name}` | `gastown/agents/toast` |

### Why Slashes?

The slash format mirrors filesystem paths and enables:
- Hierarchical parsing (extract feature, role, name)
- Consistent mail addressing (`gt mail send gastown/QA engineer`)
- Path-like routing in tickets operations
- Visual clarity about agent location

## Attribution Model

Gas Town uses three fields for complete provenance:

### Git Commits

```bash
GIT_AUTHOR_NAME="gastown/engineers/joe"      # Who did the work (agent)
GIT_AUTHOR_EMAIL="steve@example.com"    # Who owns the work (overseer)
```

Result in git log:
```
abc123 Fix bug (gastown/engineers/joe <steve@example.com>)
```

**Interpretation**:
- The agent `gastown/engineers/joe` authored the change
- The work belongs to the workspace owner (`steve@example.com`)
- Both are preserved in git history forever

### Tickets Records

```json
{
  "id": "gt-xyz",
  "created_by": "gastown/engineers/joe",
  "updated_by": "gastown/QA engineer"
}
```

The `created_by` field is populated from `BD_ACTOR` when creating tickets.
The `updated_by` field tracks who last modified the record.

### Event Logging

All events include actor attribution:

```json
{
  "ts": "2025-01-15T10:30:00Z",
  "type": "sling",
  "actor": "gastown/engineers/joe",
  "payload": { "ticket": "gt-xyz", "target": "gastown/agents/toast" }
}
```

## Environment Setup

Gas Town uses a centralized `config.AgentEnv()` function to set environment
variables consistently across all agent spawn paths (managers, daemon, boot).

### Example: Agent Environment

```bash
# Set automatically for agent 'toast' in feature 'gastown'
export GT_ROLE="agent"
export GT_RIG="gastown"
export GT_POLECAT="toast"
export BD_ACTOR="gastown/agents/toast"
export GIT_AUTHOR_NAME="gastown/agents/toast"
export GT_ROOT="/home/user/gt"
export BEADS_DIR="/home/user/gt/gastown/.tickets"
export BEADS_AGENT_NAME="gastown/toast"
```

### Example: Engineers Environment

```bash
# Set automatically for engineers member 'joe' in feature 'gastown'
export GT_ROLE="engineers"
export GT_RIG="gastown"
export GT_CREW="joe"
export BD_ACTOR="gastown/engineers/joe"
export GIT_AUTHOR_NAME="gastown/engineers/joe"
export GT_ROOT="/home/user/gt"
export BEADS_DIR="/home/user/gt/gastown/.tickets"
export BEADS_AGENT_NAME="gastown/joe"
```

### Manual Override

For local testing or debugging:

```bash
export BD_ACTOR="gastown/engineers/debug"
bd create --title="Test ticket"  # Will show created_by: gastown/engineers/debug
```

See [reference.md](reference.md#environment-variables) for the complete
environment variable reference.

## Identity Parsing

The format supports programmatic parsing:

```go
// identityToBDActor converts daemon identity to BD_ACTOR format
// Town level: product manager, senior engineer
// Feature level: {feature}/QA engineer, {feature}/release engineer
// Workers: {feature}/engineers/{name}, {feature}/agents/{name}
```

| Input | Parsed Components |
|-------|-------------------|
| `product manager` | role=product manager |
| `senior engineer` | role=senior engineer |
| `gastown/QA engineer` | feature=gastown, role=QA engineer |
| `gastown/release engineer` | feature=gastown, role=release engineer |
| `gastown/engineers/joe` | feature=gastown, role=engineers, name=joe |
| `gastown/agents/toast` | feature=gastown, role=agent, name=toast |

## Audit Queries

Attribution enables powerful audit queries:

```bash
# All work by an agent
bd audit --actor=gastown/engineers/joe

# All work in a feature
bd audit --actor=gastown/*

# All agent work
bd audit --actor=*/agents/*

# Git history by agent
git log --author="gastown/engineers/joe"
```

## Design Principles

1. **Agents are not anonymous** - Every action is attributed
2. **Work is owned, not authored** - Agent creates, overseer owns
3. **Attribution is permanent** - Git commits preserve history
4. **Format is parseable** - Enables programmatic analysis
5. **Consistent across systems** - Same format in git, tickets, events

## CV and Skill Accumulation

### Human Identity is Global

The global identifier is your **email** - it's already in every git commit. No separate "entity ticket" needed.

```
steve@example.com                ← global identity (from git author)
├── Town A (home)                ← workspace
│   ├── gastown/engineers/joe         ← agent executor
│   └── gastown/agents/toast   ← agent executor
└── Town B (work)                ← workspace
    └── acme/agents/nux        ← agent executor
```

### Agent vs Owner

| Field | Scope | Purpose |
|-------|-------|---------|
| `BD_ACTOR` | Local (town) | Agent attribution for debugging |
| `GIT_AUTHOR_EMAIL` | Global | Human identity for CV |
| `created_by` | Local | Who created the ticket |
| `owner` | Global | Who owns the work |

**Agents execute. Humans own.** The agent name in `completed-by: gastown/agents/toast` is executor attribution. The CV credits the human owner (`steve@example.com`).

### Agents Have Persistent Identities

Agents have **persistent identities but ephemeral sessions**. Like employees who
clock in/out: each work session is fresh (new tmux, new worktree), but the identity
persists across sessions.

- **Identity (persistent)**: Agent ticket, CV chain, work history
- **Session (ephemeral)**: Claude instance, context window
- **Sandbox (ephemeral)**: Git worktree, branch

Work credits the agent identity, enabling:
- Performance tracking per agent
- Capability-based routing (send Go work to agents with Go track records)
- Model comparison (A/B test different models via different agents)

See [agent-lifecycle.md](agent-lifecycle.md#agent-identity) for details.

### Skills Are Derived

Your CV emerges from querying work evidence:

```bash
# All work by owner (across all agents)
git log --author="steve@example.com"
bd list --owner=steve@example.com

# Skills derived from evidence
# - .go files touched → Go skill
# - ticket tags → domain skills
# - commit patterns → activity types
```

### Multi-Town Aggregation

A human with multiple towns has one CV:

```bash
# Future: federated CV query
bd cv steve@example.com
# Discovers all towns, aggregates work, derives skills
```

See `~/gt/docs/hop/decisions/008-identity-model.md` for architectural rationale.

## Enterprise Use Cases

### Compliance and Audit

```bash
# Who touched this file in the last 90 days?
git log --since="90 days ago" -- path/to/sensitive/file.go

# All changes by a specific agent
bd audit --actor=gastown/agents/toast --since=2025-01-01
```

### Performance Tracking

```bash
# Completion rate by agent
bd stats --group-by=actor

# Average time to completion
bd stats --actor=gastown/agents/* --metric=cycle-time
```

### Model Comparison

When agents use different underlying models, attribution enables A/B comparison:

```bash
# Tag agents by model
# gastown/agents/claude-1 uses Claude
# gastown/agents/gpt-1 uses GPT-4

# Compare quality signals
bd stats --actor=gastown/agents/claude-* --metric=revision-count
bd stats --actor=gastown/agents/gpt-* --metric=revision-count
```

Lower revision counts suggest higher first-pass quality.
