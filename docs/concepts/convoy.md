# Convoys

Convoys are the primary unit for tracking batched work across features.

## Quick Start

```bash
# Create a convoy tracking some tickets
gt convoy create "Feature X" gt-abc gt-def --notify overseer

# Check progress
gt convoy status hq-cv-abc

# List active convoys (the dashboard)
gt convoy list

# See all convoys including landed ones
gt convoy list --all
```

## Concept

A **convoy** is a persistent tracking unit that monitors related tickets across
multiple features. When you kick off work - even a single ticket - a convoy tracks it
so you can see when it lands and what was included.

```
                 🚚 Convoy (hq-cv-abc)
                         │
            ┌────────────┼────────────┐
            │            │            │
            ▼            ▼            ▼
       ┌─────────┐  ┌─────────┐  ┌─────────┐
       │ gt-xyz  │  │ gt-def  │  │ bd-abc  │
       │ gastown │  │ gastown │  │  tickets  │
       └────┬────┘  └────┬────┘  └────┬────┘
            │            │            │
            ▼            ▼            ▼
       ┌─────────┐  ┌─────────┐  ┌─────────┐
       │  nux    │  │ furiosa │  │  amber  │
       │(agent)│  │(agent)│  │(agent)│
       └─────────┘  └─────────┘  └─────────┘
                         │
                    "the swarm"
                    (ephemeral)
```

## Convoy vs Swarm

| Concept | Persistent? | ID | Description |
|---------|-------------|-----|-------------|
| **Convoy** | Yes | hq-cv-* | Tracking unit. What you create, track, get notified about. |
| **Swarm** | No | None | Ephemeral. "The workers currently on this convoy's tickets." |
| **Stranded Convoy** | Yes | hq-cv-* | A convoy with ready work but no agents assigned. Needs attention. |

When you "kick off a swarm", you're really:
1. Creating a convoy (the tracking unit)
2. Assigning agents to the tracked tickets
3. The "swarm" is just those agents while they're working

When tickets close, the convoy lands and notifies you. The swarm dissolves.

## Convoy Lifecycle

```
OPEN ──(all tickets close)──► LANDED/CLOSED
  ↑                              │
  └──(add more tickets)───────────┘
       (auto-reopens)
```

| State | Description |
|-------|-------------|
| `open` | Active tracking, work in progress |
| `closed` | All tracked tickets closed, notification sent |

Adding tickets to a closed convoy reopens it automatically.

## Commands

### Create a Convoy

```bash
# Track multiple tickets across features
gt convoy create "Deploy v2.0" gt-abc bd-xyz --notify gastown/joe

# Track a single ticket (still creates convoy for dashboard visibility)
gt convoy create "Fix auth bug" gt-auth-fix

# With default notification (from config)
gt convoy create "Feature X" gt-a gt-b gt-c
```

### Add Tickets

```bash
# Add tickets to existing convoy
gt convoy add hq-cv-abc gt-new-ticket
gt convoy add hq-cv-abc gt-ticket1 gt-ticket2 gt-ticket3

# Adding to closed convoy requires reopening first
bd update hq-cv-abc --status=open
gt convoy add hq-cv-abc gt-followup-fix
```

### Check Status

```bash
# Show tickets and active workers (the swarm)
gt convoy status hq-abc

# All active convoys (the dashboard)
gt convoy status
```

Example output:
```
🚚 hq-cv-abc: Deploy v2.0

  Status:    ●
  Progress:  2/4 completed
  Created:   2025-12-30T10:15:00-08:00

  Tracked Tickets:
    ✓ gt-xyz: Update API endpoint [task]
    ✓ bd-abc: Fix validation [bug]
    ○ bd-ghi: Update docs [task]
    ○ gt-jkl: Deploy to prod [task]
```

### List Convoys (Dashboard)

```bash
# Active convoys (default) - the primary attention view
gt convoy list

# All convoys including landed
gt convoy list --all

# Only landed convoys
gt convoy list --status=closed

# JSON output
gt convoy list --json
```

Example output:
```
Convoys

  🚚 hq-cv-w3nm6: Feature X ●
  🚚 hq-cv-abc12: Bug fixes ●

Use 'gt convoy status <id>' for detailed view.
```

## Notifications

When a convoy lands (all tracked tickets closed), subscribers are notified:

```bash
# Explicit subscriber
gt convoy create "Feature X" gt-abc --notify gastown/joe

# Multiple subscribers
gt convoy create "Feature X" gt-abc --notify product manager/ --notify --human
```

Notification content:
```
🚚 Convoy Landed: Deploy v2.0 (hq-cv-abc)

Tickets (3):
  ✓ gt-xyz: Update API endpoint
  ✓ gt-def: Add validation
  ✓ bd-abc: Update docs

Duration: 2h 15m
```

## Create from Epic

Auto-discover tracked tickets from an existing epic's children. Useful when
a planning/decomposition tool has already structured work as an epic with
child implementation tickets.

```bash
# Auto-discover children from epic
gt convoy create --from-epic gt-epic-abc

# Override the convoy name (defaults to epic title)
gt convoy create --from-epic gt-epic-abc "Custom convoy name"

# Combine with other flags
gt convoy create --from-epic gt-epic-abc --owned --merge=direct
```

**How it works:**
1. Verifies the given ticket is an epic
2. BFS-walks the parent-child hierarchy to find slingable descendants
3. Creates a standard convoy (`hq-cv-*`) tracking all slingable children (task, bug, feature, chore)

Non-slingable types (sub-epics, decisions) are recursed into but never
tracked directly. Only leaf work items appear in the convoy.

## Auto-Convoy on Sling

When you sling a single ticket without an existing convoy:

```bash
gt sling bd-xyz tickets/amber
```

This auto-creates a convoy so all work appears in the dashboard:
1. Creates convoy: "Work: bd-xyz"
2. Tracks the ticket
3. Assigns the agent

Even "swarm of one" gets convoy visibility.

## Cross-Feature Tracking

Convoys live in town-level tickets (`hq-cv-*` prefix) and can track tickets from any feature:

```bash
# Track tickets from multiple features
gt convoy create "Full-stack feature" \
  gt-frontend-abc \
  gt-backend-def \
  bd-docs-xyz
```

The `tracks` relation is:
- **Non-blocking**: doesn't affect ticket workflow
- **Additive**: can add tickets anytime
- **Cross-feature**: convoy in hq-*, tickets in gt-*, bd-*, etc.

## Convoy vs Feature Status

| View | Scope | Shows |
|------|-------|-------|
| `gt convoy status [id]` | Cross-feature | Tickets tracked by convoy + workers |
| `gt feature status <feature>` | Single feature | All workers in feature + their convoy membership |

Use convoys for "what's the status of this batch of work?"
Use feature status for "what's everyone in this feature working on?"

## See Also

- [Propulsion Principle](propulsion-principle.md) - Worker execution model
- [Mail Protocol](../design/mail-protocol.md) - Notification delivery
