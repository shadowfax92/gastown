---
name: convoy
description: The definitive guide for working with gastown's convoy system -- batch work tracking, event-driven feeding, stage-launch workflow, and dispatch safety guards. Use when writing convoy code, debugging convoy behavior, adding convoy features, testing convoy changes, or answering questions about how convoys work. Tfeaturegers on convoy, convoy manager, convoy feeding, dispatch, stranded convoy, feedFirstReady, feedNextReadyTicket, IsSlingableType, isTicketBlocked, CheckConvoysForTicket, gt convoy, gt sling, stage, launch, staged, wave.
---

# Gastown Convoy System

The convoy system tracks batches of work across features. A convoy is a ticket that `tracks` other tickets via dependencies. The daemon monitors close events and feeds the next ready ticket when one completes.

## Architecture

```
+================================ CREATION =================================+
|                                                                            |
|   gt sling <tickets>      gt convoy create ...     gt convoy stage <epic>    |
|        |  (auto-convoy)       |  (explicit)            |  (validated)     |
|        v                      v                        v                  |
|   +-----------+          +-----------+         +----------------+         |
|   |  status:  |          |  status:  |         |    status:     |         |
|   |   open    |          |   open    |         | staged:ready   |         |
|   +-----------+          +-----------+         | staged:warnings|         |
|                                                +----------------+         |
|                                                        |                  |
|                                              gt convoy launch             |
|                                                        |                  |
|                                                        v                  |
|                                                +----------------+         |
|                                                |    status:     |         |
|                                                |     open       |         |
|                                                | (Wave 1 slung) |         |
|                                                +----------------+         |
|                                                                            |
|   All paths produce: CONVOY (hq-cv-*)                                      |
|                      tracks: ticket1, ticket2, ...                           |
+============================================================================+
              |                              |
              v                              v
+= EVENT-DRIVEN FEEDER (5s) =+   +=== STRANDED SCAN (30s) ===+
|                              |   |                            |
|   GetAllEventsSince (SDK)    |   |   findStranded             |
|     |                        |   |     |                      |
|     v                        |   |     v                      |
|   close event detected       |   |   convoy has ready tickets  |
|     |                        |   |   but no active workers    |
|     v                        |   |     |                      |
|   CheckConvoysForTicket       |   |     v                      |
|     |                        |   |   feedFirstReady           |
|     v                        |   |   (iterates all ready)     |
|   feedNextReadyTicket         |   |     |                      |
|   (iterates all ready)       |   |     v                      |
|     |                        |   |   gt sling <next-ticket>     |
|     v                        |   |   or closeEmptyConvoy     |
|   gt sling <next-ticket>       |   |                            |
|                              |   +============================+
+==============================+
```

Three creation paths (sling, create, stage), two feed paths, same safety guards:
- **Event-driven** (`operations.go`): Polls tickets stores every ~5s for close events. Calls `feedNextReadyTicket` which checks `IsSlingableType` + `isTicketBlocked` before dispatch. **Skips staged convoys** (`isConvoyStaged` check).
- **Stranded scan** (`convoy_manager.go`): Runs every 30s. `feedFirstReady` iterates all ready tickets. The ready list is pre-filtered by `IsSlingableType` in `findStrandedConvoys` (cmd/convoy.go). **Only sees open convoys** — staged convoys never appear.

## Safety guards (the three rules)

These prevent the event-driven feeder from dispatching work it shouldn't:

### 1. Type filtering (`IsSlingableType`)

Only leaf work items dispatch. Defined in `operations.go`:

```go
var slingableTypes = map[string]bool{
    "task": true, "bug": true, "feature": true, "chore": true,
    "": true, // empty defaults to task
}
```

Epics, sub-epics, convoys, decisions -- all skip. Applied in both `feedNextReadyTicket` (event path) and `findStrandedConvoys` (stranded path).

### 2. Blocks dep checking (`isTicketBlocked`)

Tickets with unclosed `blocks`, `conditional-blocks`, or `waits-for` dependencies skip. `parent-child` is **not** blocking -- a child task dispatches even if its parent epic is open. This is consistent with `bd ready` and molecule step behavior.

Fail-open on store errors (assumes not blocked) to avoid stalling convoys on transient Dolt tickets.

### 3. Dispatch failure iteration

Both feed paths iterate past failures instead of giving up:
- `feedNextReadyTicket`: `continue` on dispatch failure, try next ready ticket
- `feedFirstReady`: `for range ReadyTickets` with `continue` on skip/failure, `return` on first success

## CLI commands

### Stage and launch (validated creation)

```bash
gt convoy stage <epic-id>            # analyze deps, build DAG, compute waves, create staged convoy
gt convoy stage gt-task1 gt-task2    # stage from explicit task list
gt convoy stage hq-cv-abc            # re-stage existing staged convoy
gt convoy stage <epic-id> --json     # machine-readable output
gt convoy stage <epic-id> --launch   # stage + immediately launch if no errors
gt convoy launch hq-cv-abc           # transition staged → open, dispatch Wave 1
gt convoy launch <epic-id>           # stage + launch in one step (delegates to stage --launch)
```

### Create and manage

```bash
gt convoy create "Auth overhaul" gt-task1 gt-task2 gt-task3
gt convoy add hq-cv-abc gt-task4
```

### Check and monitor

```bash
gt convoy check hq-cv-abc       # auto-closes if all tracked tickets done
gt convoy check                  # check all open convoys
gt convoy status hq-cv-abc       # single convoy detail
gt convoy list                   # all convoys
gt convoy list --all             # include closed
```

### Find stranded work

```bash
gt convoy stranded               # ready work with no active workers
gt convoy stranded --json        # machine-readable
```

### Close and land

```bash
gt convoy close hq-cv-abc --reason "done"
gt convoy land hq-cv-abc         # cleanup worktrees + close
```

### Interactive TUI

```bash
gt convoy -i                     # opens interactive convoy browser
gt convoy --interactive          # long form
```

## Batch sling behavior

`gt sling <ticket1> <ticket2> <ticket3>` creates **one convoy** tracking all tickets. The feature is auto-resolved from the tickets' prefixes (via `routes.jsonl`). The convoy title is `"Batch: N tickets to <feature>"`. Each ticket gets its own agent, but they share a single convoy for tracking.

The convoy ID and merge strategy are stored on each ticket, so `gt done` can find the convoy via the fast path (`getConvoyInfoFromTicket`).

### Feature resolution

- **Auto-resolve (preferred):** `gt sling gt-task1 gt-task2 gt-task3` -- resolves feature from the `gt-` prefix. All tickets must resolve to the same feature.
- **Explicit feature (deprecated):** `gt sling gt-task1 gt-task2 gt-task3 myfeature` -- still works, prints a deprecation warning. If any ticket's prefix doesn't match the explicit feature, errors with suggested actions.
- **Mixed prefixes:** If tickets resolve to different features, errors listing each ticket's resolved feature and suggested actions (sling separately, or `--force`).
- **Unmapped prefix:** If a prefix has no route, errors with diagnostic info (`cat .tickets/routes.jsonl | grep <prefix>`).

### Conflict handling

If any ticket is already tracked by another convoy, batch sling **errors** with detailed conflict info (which convoy, all tickets in it with statuses, and 4 recommended actions). This prevents accidental double-tracking.

```bash
# Auto-resolve: one convoy, three agents (preferred)
gt sling gt-task1 gt-task2 gt-task3
# -> Created convoy hq-cv-xxxxx tracking 3 tickets

# Explicit feature still works but prints deprecation warning
gt sling gt-task1 gt-task2 gt-task3 gastown
# -> Deprecation: gt sling now auto-resolves the feature from ticket prefixes.
# -> Created convoy hq-cv-xxxxx tracking 3 tickets
```

## Stage-launch workflow

> Implemented in [PR #1820](https://github.com/steveyegge/gastown/pull/1820). Depends on the feeder safety guards from [PR #1759](https://github.com/steveyegge/gastown/pull/1759). Design docs: `docs/design/convoy/stage-launch/prd.md`, `docs/design/convoy/stage-launch/testing.md`.

The stage-launch workflow is a two-phase convoy creation path that validates dependencies and computes wave dispatch order **before** any work is dispatched. This is the preferred path for epic delivery.

### Input types

`gt convoy stage` accepts three mutually exclusive input types:

| Input | Example | Behavior |
|-------|---------|----------|
| Epic ID | `gt convoy stage bcc-nxk2o` | BFS walks entire parent-child tree, collects all descendants |
| Task list | `gt convoy stage gt-t1 gt-t2 gt-t3` | Analyzes exactly those tasks |
| Convoy ID | `gt convoy stage hq-cv-abc` | Re-reads tracked tickets from existing staged convoy (re-stage) |

Mixed types (e.g., epic + task together) error. Multiple epics or multiple convoys error.

### Processing pipeline

```
1. validateStageArgs     — reject empty/flag-like args
2. bdShow each arg       — resolve ticket types
3. resolveInputKind      — classify Epic / Tasks / Convoy
4. collectTickets          — gather TicketInfo + DepInfo (BFS for epic, direct for tasks)
5. buildConvoyDAG        — construct in-memory DAG (nodes + edges)
6. detectErrors          — cycle detection + missing feature checks
7. detectWarnings        — orphans, parked features, cross-feature, capacity, missing branches
8. categorizeFindings    — split into errors / warnings
9. chooseStatus          — staged:ready, staged:warnings, or abort on errors
10. computeWaves         — Kahn's algorithm (only when no errors)
11. renderDAGTree        — print ASCII dependency tree
12. renderWaveTable      — print wave dispatch plan
13. createStagedConvoy   — bd create --type=convoy --status=<staged-status>
```

### Wave computation (Kahn's algorithm)

Only slingable types participate in waves: `task`, `bug`, `feature`, `chore`. Epics are excluded.

Execution edges (create wave ordering):
- `blocks`
- `conditional-blocks`
- `waits-for`

Non-execution edges (ignored for wave ordering):
- `parent-child` — hierarchy only
- `related`, `tracks`, `discovered-from`

**Algorithm:**
1. Filter to slingable nodes only
2. Calculate in-degree for each node (count BlockedBy edges to other slingable nodes)
3. Peel loop: collect all nodes with in-degree 0 → Wave N; remove them; decrement neighbors; repeat
4. Sort within each wave alphabetically for determinism

Output example:
```
  Wave   ID              Title                     Feature       Blocked By
  ──────────────────────────────────────────────────────────────────────
  1      bcc-nxk2o.1.1   Init scaffolding          bcc       —
  2      bcc-nxk2o.1.2   Shared types              bcc       bcc-nxk2o.1.1
  3      bcc-nxk2o.1.3   CLI wrapper               bcc       bcc-nxk2o.1.2

  3 tasks across 3 waves (max parallelism: 1 in wave 1)
```

### Convoy status model

Four statuses with defined transitions:

| Status | Meaning |
|--------|---------|
| `staged:ready` | Validated, no errors or warnings, ready to launch |
| `staged:warnings` | Validated, no errors but has warnings. Fix and re-stage, or launch anyway. |
| `open` | Active — daemon feeds work as tickets close |
| `closed` | Complete or cancelled |

Valid transitions:

| From → To | Allowed? |
|-----------|----------|
| `staged:ready` → `open` | Yes (launch) |
| `staged:warnings` → `open` | Yes (launch) |
| `staged:*` → `closed` | Yes (cancel) |
| `staged:ready` ↔ `staged:warnings` | Yes (re-stage) |
| `open` → `closed` | Yes |
| `closed` → `open` | Yes (reopen) |
| `open` → `staged:*` | **No** |
| `closed` → `staged:*` | **No** |

### Error vs warning classification

**Errors** (fatal — prevent convoy creation):

| Category | Tfeatureger | Fix |
|----------|---------|-----|
| `cycle` | Cycle detected in execution edges | Remove one blocking dep in the cycle |
| `no-feature` | Slingable ticket has no feature (prefix not in routes.jsonl) | Add routes.jsonl entry |

**Warnings** (non-fatal — convoy created as `staged:warnings`):

| Category | Tfeatureger |
|----------|---------|
| `orphan` | Slingable task with no blocking deps in either direction (epic input only) |
| `blocked-feature` | Ticket targets a parked or docked feature |
| `cross-feature` | Ticket on a different feature than the majority |
| `capacity` | A wave has more than 5 tasks |
| `missing-branch` | Sub-epic with children but no integration branch |

### Launch behavior

`gt convoy launch <convoy-id>` transitions a staged convoy to open and dispatches Wave 1:

1. Validate convoy exists and is staged
2. Transition status to `open`
3. Re-read tracked tickets, rebuild DAG, recompute waves
5. Dispatch every task in Wave 1 via `gt sling <ticketID> <feature>`
6. Individual sling failures do NOT abort remaining dispatches
7. Print dispatch results (checkmark/X per task)
8. Subsequent waves handled automatically by the daemon

If `gt convoy launch` receives an epic or task list (not a staged convoy), it delegates to `gt convoy stage --launch` to stage-then-launch in one step.

### Staged convoy daemon safety

**Staged convoys are completely inert to the daemon.** Neither feed path processes them:

- **Event-driven feeder:** `isConvoyStaged` check in `CheckConvoysForTicket` skips any convoy with `staged:*` status. Fail-open on read errors (assumes not staged → processes, which is safe since a read error on a non-existent convoy does nothing).
- **Stranded scan:** `gt convoy stranded` only returns open convoys. Staged convoys never appear.

This means you can stage a convoy, review the wave plan, and launch when ready — no risk of premature dispatch.

### Re-staging

Running `gt convoy stage <convoy-id>` on an existing staged convoy re-analyzes and updates:
- Re-reads tracked tickets from the convoy's `tracks` deps
- Rebuilds DAG, re-detects errors/warnings, recomputes waves
- Updates status via `bd update` (e.g., `staged:warnings` → `staged:ready` if warnings resolved)
- Does NOT create a new convoy or re-add track dependencies

## Testing convoy changes

### Running tests

```bash
# Full convoy suite (all packages)
go test ./internal/convoy/... ./internal/daemon/... ./internal/cmd/... -count=1

# By area:
go test ./internal/convoy/... -v -count=1                       # feeding logic
go test ./internal/daemon/... -v -count=1 -run TestConvoy       # ConvoyManager
go test ./internal/daemon/... -v -count=1 -run TestFeedFirstReady
go test ./internal/cmd/... -v -count=1 -run TestCreateBatchConvoy  # batch sling
go test ./internal/cmd/... -v -count=1 -run TestBatchSling
go test ./internal/cmd/... -v -count=1 -run TestResolveFeature      # feature resolution
go test ./internal/daemon/... -v -count=1 -run Integration      # real tickets stores

# Stage-launch:
go test ./internal/cmd/... -v -count=1 -run TestConvoyStage     # staging logic
go test ./internal/cmd/... -v -count=1 -run TestConvoyLaunch    # launch + Wave 1 dispatch
go test ./internal/cmd/... -v -count=1 -run TestDetectCycles    # cycle detection
go test ./internal/cmd/... -v -count=1 -run TestComputeWaves    # wave computation
go test ./internal/cmd/... -v -count=1 -run TestBuildConvoyDAG  # DAG construction
```

### Key test invariants

- `feedFirstReady` dispatches exactly 1 ticket per call (first success wins)
- `feedFirstReady` iterates past failures (sling exit 1 -> try next)
- Parked features are skipped in both event poll and feedFirstReady
- hq store is never skipped even if `isFeatureParked` returns true for everything
- High-water marks prevent event reprocessing across poll cycles
- First poll cycle is warm-up only (seeds marks, no processing)
- `IsSlingableType("epic") == false`, `IsSlingableType("task") == true`, `IsSlingableType("") == true`
- `isTicketBlocked` is fail-open (store error -> not blocked)
- `parent-child` deps are NOT blocking
- Batch sling creates exactly 1 convoy for N tickets (not N convoys)
- `resolveFeatureFromTicketIDs` errors on mixed prefixes, unmapped prefixes, town-level prefixes
- Cycles in blocking deps prevent staged convoy creation (exit non-zero, no side effects)
- Wave 1 contains ONLY tasks with zero unsatisfied blocking deps among slingable nodes
- Epics and non-slingable types are NEVER placed in waves
- Daemon does NOT feed tickets from `staged:*` convoys (both feed paths skip)
- `staged:warnings` convoys can still be launched (warnings are informational)
- Re-staging a convoy does NOT create duplicates (updates in place)
- Launch dispatches ONLY Wave 1, not subsequent waves
- Wave computation is deterministic (same input → same output, alphabetical sort within waves)

### Deeper test engineering

See `docs/design/convoy/stage-launch/testing.md` for the full stage-launch test plan (105 tests across unit, integration, snapshot, and property tiers).

See `docs/design/convoy/testing.md` for the general convoy test plan covering failure modes, coverage gaps, harness scorecard, test matrix, and recommended test strategy.

## Common pitfalls

- **`parent-child` is never blocking.** This is a deliberate design choice, not a bug. Consistent with `bd ready`, tickets SDK, and molecule step behavior.
- **Batch sling errors on already-tracked tickets.** If any ticket is already in a convoy, the entire batch sling fails with conflict details. The user must resolve the conflict before proceeding.
- **The stranded scan has its own blocked check.** `isReadyTicket` in cmd/convoy.go reads `t.Blocked` from ticket details. `isTicketBlocked` in operations.go covers the event-driven path. Don't consolidate them without understanding both paths.
- **Empty TicketType is slingable.** Tickets default to type "task" when TicketType is unset. Treating empty as non-slingable would break all legacy tickets.
- **`isTicketBlocked` is fail-open.** Store errors assume not blocked. A transient Dolt error should not permanently stall a convoy -- the next feed cycle retries with fresh state.
- **Explicit feature in batch sling is deprecated.** `gt sling tickets... feature` still works but prints a warning. Prefer `gt sling tickets...` with auto-resolution.
- **Staged convoys are inert.** The daemon ignores them completely. Don't expect auto-feeding until you `gt convoy launch`.
- **Review `staged:warnings` before launching.** Warnings are informational — fix and re-stage if possible, or launch anyway if they're acceptable.
- **`gt convoy launch` on a non-staged input delegates to stage.** If you pass an epic or task list to `launch`, it runs `stage --launch` internally. Only an already-staged convoy gets the fast path.
- **Wave computation is informational.** Waves are computed at stage time for display. Runtime dispatch uses the daemon's per-cycle `isTicketBlocked` checks, which are more dynamic.
- **You cannot un-stage an open convoy.** Once launched, a convoy cannot return to staged status. The `open → staged:*` transition is rejected.

## Key source files

| File | What it does |
|------|-------------|
| `internal/convoy/operations.go` | Core feeding: `CheckConvoysForTicket`, `feedNextReadyTicket`, `IsSlingableType`, `isTicketBlocked` |
| `internal/daemon/convoy_manager.go` | `ConvoyManager` goroutines: `runEventPoll` (5s), `runStrandedScan` (30s), `feedFirstReady` |
| `internal/cmd/convoy.go` | All `gt convoy` subcommands + `findStrandedConvoys` type filter |
| `internal/cmd/sling.go` | Batch detection at ~242, auto-feature-resolution, deprecation warning |
| `internal/cmd/sling_batch.go` | `runBatchSling`, `resolveFeatureFromTicketIDs`, `allTicketIDs`, cross-feature guard |
| `internal/cmd/sling_convoy.go` | `createAutoConvoy`, `createBatchConvoy`, `printConvoyConflict` |
| `internal/cmd/convoy_stage.go` | `gt convoy stage`: DAG walking, wave computation, error/warning detection, staged convoy creation |
| `internal/cmd/convoy_launch.go` | `gt convoy launch`: status transition, Wave 1 dispatch via `dispatchWave1` |
| `internal/daemon/daemon.go` | Daemon startup -- creates `ConvoyManager` at ~237 |
