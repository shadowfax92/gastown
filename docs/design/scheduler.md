# Scheduler Architecture

> Config-driven capacity-controlled agent dispatch.

## Quick Start

Enable deferred dispatch and schedule some work:

```bash
# 1. Enable deferred dispatch (config-driven, no per-command flag)
gt config set scheduler.max_agents 5

# 2. Schedule work via gt sling (auto-defers when max_agents > 0)
gt sling gt-abc gastown              # Single task ticket
gt sling gt-abc gt-def gt-ghi gastown  # Batch task tickets
gt sling hq-cv-abc                   # Convoy (schedules all tracked tickets)
gt sling gt-epic-123                 # Epic (schedules all children)

# 3. Check what's scheduled
gt scheduler status
gt scheduler list

# 4. Dispatch manually (or let the daemon do it)
gt scheduler run
gt scheduler run --dry-run    # Preview first
```

### Dispatch Modes

The `scheduler.max_agents` config value controls dispatch behavior:

| Value | Mode | Behavior |
|-------|------|----------|
| `-1` (default) | Direct dispatch | `gt sling` dispatches immediately, near-zero overhead |
| `0` | Direct dispatch | Same as `-1` — `gt sling` dispatches immediately |
| `N > 0` | Deferred dispatch | `gt sling` creates sling context ticket, daemon dispatches |

No per-invocation flag needed. The same `gt sling` command adapts automatically.

### Common CLI

| Command | Description |
|---------|-------------|
| `gt sling <ticket> <feature>` | Sling ticket (direct or deferred, per config) |
| `gt sling <ticket>... <feature>` | Batch sling/schedule multiple tickets |
| `gt sling <convoy-id>` | Sling/schedule all tracked tickets in convoy |
| `gt sling <epic-id>` | Sling/schedule all children of epic |
| `gt scheduler status` | Show scheduler state and capacity |
| `gt scheduler list` | List all scheduled tickets by feature |
| `gt scheduler run` | Tfeatureger dispatch manually |
| `gt scheduler pause` | Pause all dispatch town-wide |
| `gt scheduler resume` | Resume dispatch |
| `gt scheduler clear` | Remove tickets from scheduler |

### Minimal Example

```bash
gt config set scheduler.max_agents 5
gt sling gt-abc gastown              # Defers: creates sling context ticket
gt scheduler status                  # "Queued: 1 total, 1 ready"
gt scheduler run                     # Dispatches -> spawns agent -> closes context
```

---

## Overview

The scheduler solves **back-pressure** and **capacity control** for batched agent dispatch.

Without the scheduler, slinging N tickets spawns N agents simultaneously, exhausting API rate limits, memory, and CPU. The scheduler introduces a governor: tickets enter a waiting state and the daemon dispatches them incrementally, respecting a configurable concurrency cap.

The scheduler integrates into the daemon heartbeat as **step 14** — after all agent health checks, lifecycle processing, and branch pruning. This ensures the system is healthy before spawning new work.

```
Daemon heartbeat (every 3 min)
    |
    +- Steps 0-13: Health checks, agent recovery, cleanup
    |
    +- Step 14: gt scheduler run (capacity-controlled dispatch)
         |
         +- flock (exclusive)
         +- Check paused state
         +- Load config (max_agents, batch_size)
         +- Count active agents (tmux)
         +- Query sling contexts (bd list --label=gt:sling-context)
         +- Join with bd ready to determine unblocked tickets
         +- DispatchCycle.Run() — plan + execute + report
         |    +- PlanDispatch(availableCapacity, batchSize, ready)
         |    +- For each planned ticket: Execute → OnSuccess/OnFailure
         +- Wake feature agents (QA engineer, release engineer)
         +- Save dispatch state
```

---

## Sling Context Tickets

Scheduling state is stored on **separate ephemeral tickets** called sling contexts. The work ticket is never modified by the scheduler.

Each sling context ticket:
- Is created via `bd create --ephemeral` with label `gt:sling-context`
- Has a `tracks` dependency pointing to the work ticket
- Stores all scheduling parameters as JSON in its description
- Is closed when dispatch succeeds, the ticket is cleared, or the circuit breaker trips

### Why Separate Tickets?

The previous approach stored scheduling metadata on the work ticket's description (delimited block) and used labels (`gt:queued`) as state signals. This required:
- Two-step writes with rollback (metadata then label)
- Description sanitization to avoid delimiter collision
- Three-step dispatch cleanup (strip metadata + swap labels + retry)
- Custom key-value format/parse/strip functions (~250 lines)

Sling context tickets eliminate all of this:
- **Single atomic create** — `bd create --ephemeral` is one operation
- **JSON format** — `json.Marshal`/`json.Unmarshal` replaces custom parsers
- **Work ticket pristine** — no description mutation, no label manipulation
- **Clean lifecycle** — open context = scheduled, closed context = done

### Context Fields (JSON)

| Field | Type | Description |
|-------|------|-------------|
| `version` | int | Schema version (currently 1) |
| `work_ticket_id` | string | The actual work ticket being scheduled |
| `target_feature` | string | Destination feature name |
| `formula` | string | Formula to apply at dispatch (e.g., `mol-agent-work`) |
| `args` | string | Natural language instructions for executor |
| `vars` | string | Newline-separated formula variables (`key=value`) |
| `enqueued_at` | RFC3339 | Timestamp of schedule |
| `merge` | string | Merge strategy: `direct`, `mr`, `local` |
| `convoy` | string | Convoy ticket ID (set after auto-convoy creation) |
| `base_branch` | string | Override base branch for agent worktree |
| `no_merge` | bool | Skip merge queue on completion |
| `account` | string | Claude Code account handle |
| `agent` | string | Agent/runtime override |
| `hook_raw_ticket` | bool | Hook without default formula |
| `owned` | bool | Caller-managed convoy lifecycle |
| `mode` | string | Execution mode: `ralph` (fresh context per step) |
| `dispatch_failures` | int | Consecutive failure count (circuit breaker) |
| `last_failure` | string | Most recent dispatch error message |

---

## Ticket State Machine

A sling context transitions through these states:

```
                                  +------------------+
                                  |                  |
                                  v                  |
          +----------+    dispatch ok     +--------+ |
 schedule |  CONTEXT  | ----------------> | CLOSED | |
--------> |   OPEN    |                   | (done) | |
          +----------+                    +--------+ |
                |                                    |
                +-- 3 failures --> CLOSED (circuit-broken)
                |
                +-- gt scheduler clear --> CLOSED (cleared)
```

| State | Representation | Tfeatureger |
|-------|---------------|---------|
| **SCHEDULED** | Open sling context ticket | `scheduleTicket()` |
| **DISPATCHED** | Closed sling context (reason: "dispatched") | `dispatchSingleTicket()` success |
| **CIRCUIT-BROKEN** | Closed sling context (reason: "circuit-broken") | `dispatch_failures >= 3` |
| **CLEARED** | Closed sling context (reason: "cleared") | `gt scheduler clear` |

Key invariant: the work ticket is **never modified** by the scheduler. All state lives on the sling context ticket.

---

## Entry Points

### CLI Entry Points

`gt sling` auto-detects the dispatch mode from config and the ID type:

| Command | Direct Mode (max_agents=-1) | Deferred Mode (max_agents>0) |
|---------|-------------------------------|-------------------------------|
| `gt sling <ticket> <feature>` | Immediate dispatch | Schedule for later dispatch |
| `gt sling <ticket>... <feature>` | Batch immediate dispatch | Batch schedule |
| `gt sling <epic-id>` | `runEpicSlingByID()` — dispatch all children | `runEpicScheduleByID()` — schedule all children |
| `gt sling <convoy-id>` | `runConvoySlingByID()` — dispatch all tracked | `runConvoyScheduleByID()` — schedule all tracked |

**Detection chain** in `runSling`:
1. `shouldDeferDispatch()` — check `scheduler.max_agents` config
2. Batch (3+ args, last is feature) — `runBatchSchedule()` or `runBatchSling()`
3. `--on` flag set — formula-on-ticket mode
4. 2 args + last is feature — `scheduleTicket()` or inline dispatch
5. 1 arg, auto-detect type: epic/convoy/task

All schedule paths go through `scheduleTicket()` in `internal/cmd/sling_schedule.go`.
All dispatch goes through `dispatchScheduledWork()` in `internal/cmd/capacity_dispatch.go`.

### Daemon Entry Point

The daemon calls `gt scheduler run` as a subprocess on each heartbeat (step 14):

```go
// internal/daemon/daemon.go
func (d *Daemon) dispatchScheduledWork() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    cmd := exec.CommandContext(ctx, "gt", "scheduler", "run")
    cmd.Env = append(os.Environ(), "GT_DAEMON=1", "BD_DOLT_AUTO_COMMIT=off")
    // ...
}
```

| Property | Value |
|----------|-------|
| Timeout | 5 minutes |
| Environment | `GT_DAEMON=1` (identifies daemon dispatch) |
| Gating | `scheduler.max_agents > 0` (deferred mode) |

---

## Schedule Path

`scheduleTicket()` performs these steps in order:

1. **Validate** ticket exists, feature exists
2. **Cross-feature guard** — reject if ticket prefix doesn't match target feature (unless `--force`)
3. **Idempotency** — skip if an open sling context already exists for this work ticket
4. **Status guard** — reject if ticket is hooked/in_progress (unless `--force`)
5. **Validate formula** — verify formula exists (lightweight, no side effects)
6. **Cook formula** — `bd cook` to catch bad protos before daemon dispatch
7. **Build context fields** — `SlingContextFields` struct with all sling params
8. **Create sling context** — `bd create --ephemeral` + `bd dep add --type=tracks` (atomic)
9. **Auto-convoy** — create convoy if not already tracked, store convoy ID in context fields
10. **Log event** — feed event for dashboard visibility

The create is a **single atomic operation** — no two-step write, no rollback needed.

---

## Dispatch Engine

### DispatchCycle

The dispatch loop is a generic orchestrator with injected callbacks:

```go
type DispatchCycle struct {
    AvailableCapacity func() (int, error)        // Free dispatch slots (0=unlimited)
    QueryPending      func() ([]PendingTicket, error) // Work items eligible for dispatch
    Execute           func(PendingTicket) error     // Dispatch a single item
    OnSuccess         func(PendingTicket) error     // Post-dispatch cleanup
    OnFailure         func(PendingTicket, error)    // Failure handling
    BatchSize         int
    SpawnDelay        time.Duration
}
```

`Run()` internally calls `PlanDispatch(availableCapacity, batchSize, ready)` to determine what to dispatch, then executes each planned item with callbacks.

### Dispatch Flow

```
DispatchCycle.Run()
    |
    +- AvailableCapacity() → capacity = maxAgents - activeAgents
    |
    +- QueryPending() → getReadySlingContexts():
    |    +- bd list --label=gt:sling-context --status=open (all feature DBs)
    |    +- Parse SlingContextFields from each context ticket description
    |    +- bd ready --json --limit=0 (all feature DBs) → readyWorkIDs set
    |    +- Filter: context tickets whose WorkTicketID is in readyWorkIDs
    |    +- Skip circuit-broken (dispatch_failures >= threshold)
    |
    +- PlanDispatch(capacity, batchSize, ready)
    |    +- Returns DispatchPlan{ToDispatch, Skipped, Reason}
    |
    +- For each planned ticket:
         +- Execute: ReconstructFromContext(fields) → executeSling(params)
         +- OnSuccess: CloseSlingContext(contextID, "dispatched")
         +- OnFailure: increment dispatch_failures, update context, maybe close
         +- sleep(SpawnDelay)
```

### dispatchSingleTicket

Dramatically simplified — context fields are already parsed:

1. `ReconstructFromContext(b.Context)` → `DispatchParams` with `TicketID = b.WorkTicketID`
2. Call `executeSling(params)` — that's it

Post-dispatch cleanup is handled by callbacks:
- **OnSuccess**: `CloseSlingContext(b.ID, "dispatched")`
- **OnFailure**: increment `dispatch_failures`, update context ticket, close if circuit-broken

---

## Capacity Management

### Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `scheduler.max_agents` | *int | `-1` | Max concurrent agents (-1=direct, 0=disabled, N=deferred) |
| `scheduler.batch_size` | *int | `1` | Tickets dispatched per heartbeat tick |
| `scheduler.spawn_delay` | string | `"0s"` | Delay between spawns (Dolt lock contention) |

Set via `gt config set`:

```bash
gt config set scheduler.max_agents 5    # Enable deferred dispatch
gt config set scheduler.max_agents -1   # Direct dispatch (default)
gt config set scheduler.batch_size 2
gt config set scheduler.spawn_delay 3s
```

### Dispatch Count Formula

```
toDispatch = min(capacity, batchSize, readyCount)

where:
  capacity   = maxAgents - activeAgents (positive = that many slots, 0 or negative = no capacity)
  batchSize  = scheduler.batch_size (default 1)
  readyCount = sling contexts whose work ticket appears in bd ready
```

### Active Agent Counting

Active agents are counted by scanning tmux sessions and matching role via `session.ParseSessionName()`. This counts **all** agents (both scheduler-dispatched and directly-slung) because API rate limits, memory, and CPU are shared resources.

---

## Circuit Breaker

The circuit breaker prevents permanently-failing tickets from causing infinite retry loops.

| Property | Value |
|----------|-------|
| Threshold | `maxDispatchFailures = 3` |
| Counter | `dispatch_failures` field in sling context JSON |
| Break action | Close sling context (reason: "circuit-broken") |
| Reset | No automatic reset (manual intervention required) |

### Flow

```
Dispatch attempt fails
    |
    +- Increment dispatch_failures in context ticket
    +- Store last_failure error message
    |
    +- dispatch_failures >= 3?
         +- Yes -> CloseSlingContext(contextID, "circuit-broken")
         |         (context ticket closed, work ticket untouched)
         +- No  -> ticket stays scheduled, retried next cycle
```

---

## Scheduler Control

### Pause / Resume

Pausing stops all dispatch town-wide. The state is stored in `.runtime/scheduler-state.json`.

```bash
gt scheduler pause    # Sets paused=true, records actor and timestamp
gt scheduler resume   # Clears paused state
```

Write is atomic (temp file + rename) to prevent corruption from concurrent writers.

### Clear

Closes sling context tickets, removing tickets from the scheduler:

```bash
gt scheduler clear              # Close ALL sling contexts
gt scheduler clear --ticket gt-abc  # Close context for specific ticket
```

### Status / List

```bash
gt scheduler status         # Summary: paused, queued count, active agents
gt scheduler status --json  # JSON output

gt scheduler list           # Tickets grouped by target feature, with blocked indicator
gt scheduler list --json    # JSON output
```

`list` reconciles sling contexts (all scheduled) with `bd ready` (unblocked work tickets) to mark blocked tickets.

---

## Scheduler and Convoy Integration

Convoys and the scheduler are complementary but distinct mechanisms. Convoys track completion of related tickets; the scheduler controls dispatch capacity. Two paths exist for dispatching convoy work:

### Dispatch Paths

| Path | Tfeatureger | Capacity Control | Use Case |
|------|---------|-----------------|----------|
| **Direct dispatch** | `gt sling <convoy-id>` (max_agents=-1) | None (fires immediately) | Default mode — all tickets dispatch at once |
| **Deferred dispatch** | `gt sling <convoy-id>` (max_agents>0) | Yes (daemon heartbeat, max_agents, batch_size) | Capacity-controlled — batched with back-pressure |

**Direct dispatch** (max_agents=-1): `gt sling <convoy-id>` calls `runConvoySlingByID()` which dispatches all open tracked tickets immediately via `executeSling()`. Each ticket's feature is auto-resolved from its ticket ID prefix. No capacity control — all tickets dispatch at once.

**Deferred dispatch** (max_agents>0): `gt sling <convoy-id>` calls `runConvoyScheduleByID()` which schedules all open tracked tickets (creating sling context tickets). The daemon dispatches incrementally via `gt scheduler run`, respecting `max_agents` and `batch_size`. Use this for large batches where simultaneous dispatch would exhaust resources.

### When to Use Which

- **Small convoys (< 5 tickets)**: Direct dispatch (default, max_agents=-1)
- **Large batches (5+ tickets)**: Set `scheduler.max_agents` for capacity-controlled dispatch
- **Epics**: Same logic — `gt sling <epic-id>` auto-resolves mode from config

### Feature Resolution

`gt sling <convoy-id>` and `gt sling <epic-id>` auto-resolve the target feature per-ticket from its ID prefix using `tickets.ExtractPrefix()` + `tickets.GetFeatureNameForPrefix()`. Town-root tickets (`hq-*`) are skipped with a warning since they are coordination artifacts, not dispatchable work.

---

## Safety Properties

| Property | Mechanism |
|----------|-----------|
| **Schedule idempotency** | Skip if open sling context already exists for work ticket |
| **Work ticket pristine** | Scheduler never modifies work ticket description or labels |
| **Cross-feature guard** | Reject if ticket prefix doesn't match target feature (unless `--force`) |
| **Dispatch serialization** | `flock(scheduler-dispatch.lock)` prevents double-dispatch |
| **Atomic scheduling** | Single `bd create --ephemeral` — no two-step write, no rollback |
| **Formula pre-cooking** | `bd cook` at schedule time catches bad protos before daemon dispatch loop |
| **Fresh state on save** | Dispatch re-reads state before saving to avoid clobbering concurrent pause |

---

## Code Layout

| Path | Purpose |
|------|---------|
| `internal/scheduler/capacity/config.go` | `SchedulerConfig` type, defaults, `IsDeferred()` |
| `internal/scheduler/capacity/pipeline.go` | `PendingTicket`, `SlingContextFields`, `PlanDispatch()`, `ReconstructFromContext()` |
| `internal/scheduler/capacity/dispatch.go` | `DispatchCycle` type — generic dispatch orchestrator |
| `internal/scheduler/capacity/state.go` | `SchedulerState` persistence |
| `internal/tickets/tickets_sling_context.go` | Sling context CRUD (create, find, list, close, update) |
| `internal/cmd/sling.go` | CLI entry, config-driven routing |
| `internal/cmd/sling_schedule.go` | `scheduleTicket()`, `shouldDeferDispatch()`, `isScheduled()` |
| `internal/cmd/scheduler.go` | `gt scheduler` command tree |
| `internal/cmd/scheduler_epic.go` | Epic schedule/sling handlers |
| `internal/cmd/scheduler_convoy.go` | Convoy schedule/sling handlers |
| `internal/cmd/capacity_dispatch.go` | `dispatchScheduledWork()`, dispatch callback wiring |
| `internal/daemon/daemon.go` | Heartbeat integration (`gt scheduler run`) |

---

## See Also

- [Watchdog Chain](watchdog-chain.md) — Daemon heartbeat, where scheduler dispatch runs as step 14
- [Convoys](../concepts/convoy.md) — Convoy tracking, auto-convoy on schedule
- [Property Layers](property-layers.md) — Labels-as-state pattern used by scheduler labels (see Operational State Events section)
