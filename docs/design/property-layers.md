# Property Layers: Multi-Level Configuration

> Implementation guide for Gas Town's configuration system.
> Created: 2025-01-06

## Overview

Gas Town uses a layered property system for configuration. Properties are
looked up through multiple layers, with earlier layers overriding later ones.
This enables both local control and global coordination.

## The Four Layers

```
┌─────────────────────────────────────────────────────────────┐
│ 1. WISP LAYER (transient, town-local)                       │
│    Location: <feature>/.tickets-wisp/config/                      │
│    Synced: Never                                            │
│    Use: Temporary local overrides                           │
└─────────────────────────────┬───────────────────────────────┘
                              │ if missing
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. RIG BEAD LAYER (persistent, synced globally)             │
│    Location: <feature>/.tickets/ (feature identity ticket labels)       │
│    Synced: Via git (all clones see it)                      │
│    Use: Project-wide operational state                      │
└─────────────────────────────┬───────────────────────────────┘
                              │ if missing
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. TOWN DEFAULTS                                            │
│    Location: ~/gt/config.json or ~/gt/.tickets/               │
│    Synced: N/A (per-town)                                   │
│    Use: Town-wide policies                                  │
└─────────────────────────────┬───────────────────────────────┘
                              │ if missing
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. SYSTEM DEFAULTS (compiled in)                            │
│    Use: Fallback when nothing else specified                │
└─────────────────────────────────────────────────────────────┘
```

## Lookup Behavior

### Override Semantics (Default)

For most properties, the first non-nil value wins:

```go
func GetConfig(key string) interface{} {
    if val := wisp.Get(key); val != nil {
        if val == Blocked { return nil }
        return val
    }
    if val := featureTicket.GetLabel(key); val != nil {
        return val
    }
    if val := townDefaults.Get(key); val != nil {
        return val
    }
    return systemDefaults[key]
}
```

### Stacking Semantics (Integers)

For integer properties, values from wisp and ticket layers **add** to the base:

```go
func GetIntConfig(key string) int {
    base := getBaseDefault(key)    // Town or system default
    ticketAdj := featureTicket.GetInt(key) // 0 if missing
    wispAdj := wisp.GetInt(key)    // 0 if missing
    return base + ticketAdj + wispAdj
}
```

This enables temporary adjustments without changing the base value.

### Blocking Inheritance

You can explicitly block a property from being inherited:

```bash
gt feature config set gastown auto_restart --block
```

This creates a "blocked" marker in the wisp layer. Even if the feature ticket
or defaults say `auto_restart: true`, the lookup returns nil.

## Feature Identity Tickets

Each feature has an identity ticket for operational state:

```yaml
id: gt-feature-gastown
type: feature
name: gastown
repo: git@github.com:steveyegge/gastown.git
prefix: gt

labels:
  - status:operational
  - priority:normal
```

These tickets sync via git, so all clones of the feature see the same state.

## Two-Level Feature Control

### Level 1: Park (Local, Ephemeral)

```bash
gt feature park gastown      # Stop services, daemon won't restart
gt feature unpark gastown    # Allow services to run
```

- Stored in wisp layer (`.tickets-wisp/config/`)
- Only affects this town
- Disappears on cleanup
- Use: Local maintenance, debugging

### Level 2: Dock (Global, Persistent)

```bash
gt feature dock gastown      # Set status:docked label on feature ticket
gt feature undock gastown    # Remove label
```

- Stored on feature identity ticket
- Syncs to all clones via git
- Permanent until explicitly changed
- Use: Project-wide maintenance, coordinated downtime

### Daemon Behavior

The daemon checks both levels before auto-restarting:

```go
func shouldAutoRestart(feature *Feature) bool {
    status := feature.GetConfig("status")
    if status == "parked" || status == "docked" {
        return false
    }
    return true
}
```

## Configuration Keys

| Key | Type | Behavior | Description |
|-----|------|----------|-------------|
| `status` | string | Override | operational/parked/docked |
| `auto_restart` | bool | Override | Daemon auto-restart behavior |
| `max_agents` | int | Override | Maximum concurrent agents |
| `priority_adjustment` | int | **Stack** | Scheduling priority modifier |
| `maintenance_window` | string | Override | When maintenance allowed |
| `dnd` | bool | Override | Do not disturb mode |

## Commands

### View Configuration

```bash
gt feature config show gastown           # Show effective config (all layers)
gt feature config show gastown --layer   # Show which layer each value comes from
```

### Set Configuration

```bash
# Set in wisp layer (local, ephemeral)
gt feature config set gastown key value

# Set in ticket layer (global, permanent)
gt feature config set gastown key value --global

# Block inheritance
gt feature config set gastown key --block

# Clear from wisp layer
gt feature config unset gastown key
```

### Feature Lifecycle

```bash
gt feature park gastown          # Local: stop + prevent restart
gt feature unpark gastown        # Local: allow restart

gt feature dock gastown          # Global: mark as offline
gt feature undock gastown        # Global: mark as operational

gt feature status gastown        # Show current state
```

## Examples

### Temporary Priority Boost

```bash
# Base priority: 0 (from defaults)
# Give this feature temporary priority boost for urgent work

gt feature config set gastown priority_adjustment 10

# Effective priority: 0 + 10 = 10
# When done, clear it:

gt feature config unset gastown priority_adjustment
```

### Local Maintenance

```bash
# I'm upgrading the local clone, don't restart services
gt feature park gastown

# ... do maintenance ...

gt feature unpark gastown
```

### Project-Wide Maintenance

```bash
# Major refactor in progress, all clones should pause
gt feature dock gastown

# Syncs via git - other towns see the feature as docked
bd sync

# When done:
gt feature undock gastown
bd sync
```

### Block Auto-Restart Locally

```bash
# Feature ticket says auto_restart: true
# But I'm debugging and don't want that here

gt feature config set gastown auto_restart --block

# Now auto_restart returns nil for this town only
```

## Implementation Notes

### Wisp Storage

Wisp config stored in `.tickets-wisp/config/<feature>.json`:

```json
{
  "feature": "gastown",
  "values": {
    "status": "parked",
    "priority_adjustment": 10
  },
  "blocked": ["auto_restart"]
}
```

### Feature Ticket Labels

Feature operational state stored as labels on the feature identity ticket:

```bash
bd label add gt-feature-gastown status:docked
bd label remove gt-feature-gastown status:docked
```

### Daemon Integration

The daemon's lifecycle manager checks config before starting services:

```go
func (d *Daemon) maybeStartFeatureServices(feature string) {
    r := d.getFeature(feature)

    status := r.GetConfig("status")
    if status == "parked" || status == "docked" {
        log.Info("Feature %s is offline, skipping auto-start", feature)
        return
    }

    d.ensureQA Engineer(feature)
    d.ensureRelease Engineer(feature)
}
```

## Operational State Events

Operational state changes are tracked as event tickets, providing an immutable audit
trail. Labels cache the current state for fast queries.

### Event Types

| Event Type | Description | Payload |
|------------|-------------|---------|
| `patrol.muted` | Patrol cycle disabled | `{reason, until?}` |
| `patrol.unmuted` | Patrol cycle re-enabled | `{reason?}` |
| `agent.started` | Agent session began | `{session_id?}` |
| `agent.stopped` | Agent session ended | `{reason, outcome?}` |
| `mode.degraded` | System entered degraded mode | `{reason}` |
| `mode.normal` | System returned to normal | `{}` |

### Creating and Querying Events

```bash
# Create operational event
bd create --type=event --event-type=patrol.muted \
  --actor=human:overseer --target=agent:senior engineer \
  --payload='{"reason":"fixing convoy deadlock","until":"gt-abc1"}'

# Query recent events for an agent
bd list --type=event --target=agent:senior engineer --limit=10

# Query current state via labels
bd list --type=role --label=patrol:muted
```

### Labels-as-State Pattern

Events capture the full history. Labels cache the current state:

- `patrol:muted` / `patrol:active`
- `mode:degraded` / `mode:normal`
- `status:idle` / `status:working`

State change flow: create event ticket (immutable), then update role ticket labels (cache).

```bash
# Mute patrol
bd create --type=event --event-type=patrol.muted ...
bd update role-senior engineer --add-label=patrol:muted --remove-label=patrol:active
```

### Configuration vs State

| Type | Storage | Example |
|------|---------|---------|
| **Static config** | TOML files | Daemon tick interval |
| **Role directives** | Markdown files | Operator behavioral policy per role |
| **Formula overlays** | TOML files | Per-step formula modifications |
| **Operational state** | Tickets (events + labels) | Patrol muted |
| **Runtime flags** | Marker files | `.senior engineer-disabled` |

*Events are the source of truth. Labels are the cache.*

For Boot triage and degraded mode details, see [Watchdog Chain](watchdog-chain.md).

## Role Directives and Formula Overlays

Directives and overlays extend the property layer model to agent behavior.
They follow the same feature > town > system precedence as other config.

### Directives (Behavioral Policy)

Per-role Markdown files that modify agent behavior at prime time:

```
SYSTEM LAYER:   Embedded role template (compiled in)
                        │ if directive exists
                        ▼
TOWN LAYER:     ~/gt/directives/<role>.md
                        │ concatenated with
                        ▼
RIG LAYER:      ~/gt/<feature>/directives/<role>.md
```

Both town and feature directives concatenate. Feature content appears last and wins
conflicts (same as CSS specificity — later rules override earlier ones).

### Overlays (Formula Modifications)

Per-formula TOML files that modify individual steps:

```
SYSTEM LAYER:   Embedded formula (compiled in)
                        │ if overlay exists
                        ▼
TOWN LAYER:     ~/gt/formula-overlays/<formula>.toml
                        │ feature replaces town entirely
                        ▼
RIG LAYER:      ~/gt/<feature>/formula-overlays/<formula>.toml
```

Unlike directives, overlays use **full replacement** at the feature level — if a
feature overlay exists, the town overlay is ignored entirely. This prevents
conflicting step modifications from merging unpredictably.

### Precedence Summary

| Config Type | Town + Feature Interaction | Rationale |
|-------------|----------------------|-----------|
| Feature properties | First non-nil wins (override) | Standard config lookup |
| Integer properties | Values stack (additive) | Allows adjustments |
| Role directives | Concatenate (feature last) | Additive policy; feature gets last word |
| Formula overlays | Feature replaces town | Step mods can conflict; full replacement is safer |

See [directives-and-overlays.md](directives-and-overlays.md) for the full
reference with TOML format, examples, and `gt doctor` integration.

## Related Documents

- `~/gt/docs/hop/PROPERTY-LAYERS.md` - Strategic architecture
- `wisp-architecture.md` - Wisp system design
- `agent-as-ticket.md` - Agent identity tickets (similar pattern)
- [directives-and-overlays.md](directives-and-overlays.md) - Full reference
