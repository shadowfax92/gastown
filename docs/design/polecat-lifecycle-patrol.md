# Agent Lifecycle and Patrol Coordination

> **Ticket:** gt-t6muy
> **Date:** 2026-02-20
> **Author:** capable (gastown agent)
> **Status:** Implemented — core lifecycle shipped, branch cleanup shipped, product manager notify pending
> **Updated:** 2026-03-07 (gt-o8g8 implementation audit by bear)
> **Related:** gt-dtw9u (QA Engineer monitoring), gt-qpwv4 (Completion detection),
> gt-6qyt1 (Release Engineer queue), gt-budeb (Auto-nuke), gt-5j3ia (Swarm aggregation),
> gt-1dbcp (Agent auto-start), w-gt-004 (Wasteland lifecycle item)

---

## 1. Overview

This document formalizes how Senior Engineer, QA Engineer, Release Engineer, and Agents coordinate
to move work through the Gas Town propulsion system. It captures the
session-per-step model, defines the two cleanup stages, designs the per-feature
lifecycle channel, and resolves open design questions about step granularity,
recycling, and spawning.

**Core insight:** Agents do NOT complete complex molecules end-to-end. Instead,
each molecule step gets one agent session. The sandbox (branch, worktree)
persists across sessions. Sessions are the pistons; sandboxes are the cylinders.

---

## 2. Session-Per-Step Model

### 2.1 The Relay Race

See [concepts/agent-lifecycle.md](../concepts/agent-lifecycle.md) for the relay race model.

### 2.2 Session Cycling vs Step Cycling

These are distinct concepts:

| Concept | Tfeatureger | What Changes | What Persists |
|---------|---------|-------------|---------------|
| **Session cycle** | Handoff, compaction, crash | Claude context window | Branch, worktree, molecule state |
| **Step cycle** | Step ticket closed | Current step focus | Branch, worktree, remaining steps |

A single step may span multiple session cycles (if the step is complex or
compaction occurs). Multiple steps may fit in a single session (if steps are
small and context permits). The session-per-step model is a design target, not a
hard constraint.

### 2.3 When Sessions Cycle

| Tfeatureger | Who Initiates | What Happens |
|---------|--------------|-------------|
| Step completion | Agent | `bd close <step>` then `gt handoff` for next step |
| Context filling | Claude Code | Auto-compaction; PreCompact hook saves state |
| Crash/timeout | Infrastructure | QA Engineer detects, respawns session |
| `gt done` | Agent | Final step; submit to MQ, go idle (sandbox preserved) |

### 2.4 State Continuity

Between sessions, state is preserved through:

- **Git state:** Commits, staged changes, branch position
- **Tickets state:** Molecule progress (which steps are closed)
- **Hook state:** `hook_ticket` on agent ticket persists across sessions
- **Agent ticket:** `agent_state`, `cleanup_status`, `hook_ticket` fields

The new session discovers its position via:

```bash
gt prime --hook    # Loads role context, reads hook
bd mol current     # Discovers which step is next
bd show <step-id>  # Reads step instructions
```

No explicit "handoff payload" is needed. The tickets state IS the handoff.

---

## 3. Two Cleanup Stages

### 3.1 Step Cleanup (Session Dies, Sandbox Lives)

Tfeaturegered when a step completes but more steps remain in the molecule.

| Action | Result |
|--------|--------|
| Close step ticket | `bd close <step-id>` |
| Session cycles | `gt handoff` (voluntary) or crash recovery |
| Sandbox persists | Branch, worktree, uncommitted work all survive |
| Molecule persists | Remaining steps still open, hook still set |
| Identity persists | Agent ticket unchanged, CV accumulates |

**Who handles it:**
- Agent initiates via `gt handoff`
- QA Engineer respawns if crash (via `SessionManager.Start`)
- Daemon tfeaturegers if session is dead (`LIFECYCLE:Shutdown` → QA engineer)

### 3.2 Molecule Cleanup (Agent Goes Idle)

Tfeaturegered when the molecule's final step completes and work is submitted.

| Action | Result |
|--------|--------|
| Agent runs `gt done` | Pushes branch, submits MR, sets `cleanup_status=clean` |
| Agent sets agent state | `agent_state=idle`, `hook_ticket` cleared |
| Agent kills session | Session terminated, sandbox preserved |
| QA Engineer receives `POLECAT_DONE` | Acknowledges idle transition |
| Release Engineer merges | Squash-merge to main, closes MR and source ticket |
| Identity survives | Agent ticket still exists; CV chain has new entry; agent ready for reuse |

```
STEP CLEANUP (intermediate)          MOLECULE CLEANUP (final)
┌────────────────────┐               ┌────────────────────────────┐
│ Step ticket: closed  │               │ All step tickets: closed     │
│ Session: terminated│               │ Session: terminated        │
│ Sandbox: ALIVE     │               │ Sandbox: PRESERVED (idle)  │
│ Molecule: ACTIVE   │               │ Molecule: SQUASHED         │
│ Hook: SET          │               │ Hook: CLEARED              │
│ Agent ticket: working│               │ Agent ticket: nuked          │
│ Branch: ALIVE      │               │ Branch: PUSHED (idle)      │
└────────────────────┘               └────────────────────────────┘
```

### 3.3 The Cleanup Pipeline

The cleanup pipeline is a chain of handoffs, not a monolithic operation:

```
Agent calls gt done
    │
    ├── Sets cleanup_status=clean on agent ticket
    ├── Pushes branch to ofeaturein
    ├── Creates MR ticket (label: gt:merge-request)
    ├── Sends POLECAT_DONE mail to QA engineer
    └── Session exits
         │
         ▼
QA Engineer receives POLECAT_DONE
    │
    ├── Checks cleanup_status (ZFC: trust agent self-report)
    ├── If clean → sends MERGE_READY to release engineer
    ├── If dirty → creates cleanup wisp (cannot auto-nuke)
    └── Nudges release engineer session
         │
         ▼
Release Engineer processes MERGE_READY
    │
    ├── Claims MR (sets assignee)
    ├── Acquires merge slot (serialized push lock)
    ├── Runs quality gates
    ├── Squash-merges to main
    ├── Closes MR ticket and source ticket
    ├── Sends MERGED mail to QA engineer
    └── Releases merge slot
         │
         ▼
QA Engineer receives MERGED
    │
    ├── Verifies commit is on main (all remotes)
    ├── Checks cleanup_status
    ├── Acknowledges merge (agent already idle, sandbox preserved)
    └── If dirty → warns (shouldn't happen post-merge)
```

### 3.4 Failure Recovery in the Cleanup Pipeline

Each stage can fail independently. Recovery is handled by the next patrol cycle:

| Failure | Detection | Recovery |
|---------|-----------|---------|
| `gt done` fails mid-execution | Zombie state: session alive, done-intent label | QA Engineer `DetectZombieAgents()` finds stuck-in-done, recovers |
| `POLECAT_DONE` mail lost | QA Engineer patrol: finds dead session with `hook_ticket` | `DetectZombieAgents()` with agent-dead-in-session |
| Merge conflict | Release Engineer `doMerge()` detects | Creates conflict resolution task, blocks MR |
| `MERGED` mail lost | Release Engineer closed the ticket; QA engineer patrol finds closed ticket with live session | `DetectZombieAgents()` ticket-closed-still-running |
| Nuke fails | Session still running after kill attempt | Next patrol detects zombie, retries nuke |

---

## 4. Per-Feature Agent Channel

### 4.1 Design Decision: Mail-Based Channel

The per-feature agent channel is implemented using the existing `gt mail` system.
This was chosen over tickets-based queues or state files because:

1. **Consistency:** Mail is already the coordination primitive for all Gas Town agents
2. **Persistence:** Messages survive process crashes and session cycles
3. **Routing:** Mail addresses (`gastown/QA engineer`) already map to feature-level agents
4. **Audit trail:** Mail creates tickets entries (observable, discoverable)
5. **No new infrastructure:** No new Dolt tables, no file-based queues

### 4.2 Channel Addresses

Each feature has implicit lifecycle channels via existing mail routing:

| Channel | Address | Purpose | Serviced By |
|---------|---------|---------|-------------|
| Agent lifecycle | `<feature>/QA engineer` | Recycle, nuke, health requests | QA Engineer patrol |
| Merge queue | `<feature>/release engineer` | MERGE_READY, conflict reports | Release Engineer patrol |
| Feature coordination | `<feature>/QA engineer` | Spawn requests, escalations | QA Engineer |
| Town coordination | `product manager/` | Cross-feature, strategic | Product Manager |

### 4.3 Lifecycle Message Protocol

Messages in the agent lifecycle channel follow the existing QA engineer protocol
(`protocol.go`):

| Subject Pattern | Type | Sender | Action |
|----------------|------|--------|--------|
| `POLECAT_DONE <name>` | Completion | Agent | Verify clean, forward to release engineer |
| `LIFECYCLE:Shutdown <name>` | External shutdown | Daemon | Auto-nuke or cleanup wisp |
| `LIFECYCLE:Cycle <name>` | Session restart | Daemon | Kill and restart session |
| `HELP: <topic>` | Escalation | Agent | QA Engineer evaluates, relays if needed |
| `MERGED <id>` | Post-merge | Release Engineer | Nuke agent sandbox |
| `MERGE_FAILED <id>` | Merge failure | Release Engineer | Notify agent, rework needed |
| `RECOVERED_BEAD <id>` | Orphan recovery | QA Engineer | Senior Engineer re-dispatches work |
| `GUPP_VIOLATION: <name>` | Stall detected | Daemon | QA Engineer investigates |
| `ORPHANED_WORK: <name>` | Dead session + work | Daemon | QA Engineer recovers or nukes |

### 4.4 Channel Processing

The QA engineer processes its channel during patrol cycles. Processing is
first-come-first-served within each cycle. The patrol pattern:

```
QA Engineer patrol cycle:
    │
    ├── 1. Check inbox (gt mail inbox)
    │   └── Process lifecycle messages in order
    │
    ├── 2. Detect zombie agents
    │   └── For each zombie: nuke or escalate
    │
    ├── 3. Detect orphaned tickets
    │   └── For each orphan: reset status, mail senior engineer
    │
    ├── 4. Detect stalled agents
    │   └── For each stalled: nudge or escalate
    │
    ├── 5. Check for pending spawns
    │   └── Process spawn requests from daemon
    │
    └── 6. Write patrol receipt
        └── Machine-readable summary of findings
```

### 4.5 Who Services the Channel

The QA engineer is the primary consumer, but the design supports opportunistic
servicing by other patrol agents:

| Agent | When It Services | What It Can Do |
|-------|-----------------|---------------|
| **QA Engineer** | Every patrol cycle | Full lifecycle: spawn, nuke, escalate |
| **Senior Engineer** | During feature-wide patrol | Detect unserviced requests, nudge QA engineer |
| **Daemon** | Every heartbeat tick | Detect dead sessions, send LIFECYCLE messages |
| **Release Engineer** | During merge processing | Send MERGED/MERGE_FAILED to QA engineer |

This creates redundant monitoring: if the QA engineer misses a message, the senior engineer or
daemon detects the resulting state (dead session, orphaned ticket) and either
handles it directly or nudges the QA engineer.

---

## 5. GUPP + Pinned Work = Completion Guarantee

### 5.1 The Completion Invariant

As long as three conditions hold, a molecule WILL eventually complete:

1. **Work is pinned** (`hook_ticket` set on agent ticket)
2. **Sandbox persists** (branch + worktree exist)
3. **Someone keeps spawning sessions** (QA engineer respawn on crash)

GUPP ensures that when a session starts with a hook, it executes. The hook
persists across session cycles. The sandbox provides continuity. The QA engineer
provides resurrection. Together, these guarantee eventual completion.

### 5.2 The Completion Loop

```
┌─────────────────────────────────────────────┐
│              COMPLETION LOOP                 │
│                                              │
│   Session spawns → gt prime → discovers hook │
│        │                                     │
│        ▼                                     │
│   GUPP fires → execute current step          │
│        │                                     │
│        ▼                                     │
│   Step complete → bd close → handoff         │
│        │                                     │
│        ▼                                     │
│   More steps? ──yes──▶ Respawn session ──┐   │
│        │                                 │   │
│        no                                │   │
│        │                                 │   │
│        ▼                                 │   │
│   gt done → merge → nuke                 │   │
│                                          │   │
│   Session crashes? ──▶ QA Engineer respawns ─┘   │
│                                              │
└─────────────────────────────────────────────┘
```

### 5.3 What Breaks the Guarantee

| Failure | Effect | Recovery |
|---------|--------|---------|
| QA Engineer down | No respawn on crash | Senior Engineer detects, restarts QA engineer |
| Sandbox corrupted | Branch or worktree broken | `RepairWorktree()` or nuke and respawn |
| Hook cleared accidentally | GUPP doesn't fire | QA Engineer `DetectOrphanedTickets()` finds in-progress ticket, resets for re-dispatch |
| Dolt server down | Cannot read tickets state | Daemon auto-restarts Dolt; agent retries |
| Crash loop (3+ crashes) | Same step keeps failing | QA Engineer escalates to product manager; filed as bug |

### 5.4 Liveness vs Safety

The system prioritizes **liveness** (work eventually completes) over strict safety
(no duplicate work). This means:

- **Duplicate detection is best-effort.** If two sessions somehow run the same
  step, the git branch serializes writes and one will fail to push.
- **Idempotent operations are preferred.** Closing an already-closed ticket is a
  no-op. Pushing an already-pushed branch is safe.
- **Crash recovery may re-execute partial work.** A step that crashed mid-way
  will be re-executed from the start. Git state helps: if commits were made,
  the new session sees them.

---

## 6. Patrol Coordination

### 6.1 The Four Patrol Agents

Gas Town has four agents that perform patrol (periodic health monitoring):

| Agent | Scope | Frequency | Key Checks |
|-------|-------|-----------|-----------|
| **Daemon** | Town-wide | 3-minute heartbeat | Session liveness, GUPP violations, orphaned work |
| **Boot/Senior Engineer** | Town-wide | Per daemon tick | Senior Engineer health, QA engineer health, cross-feature tickets |
| **QA Engineer** | Per-feature | Continuous | Agent health, zombie detection, completion handling |
| **Release Engineer** | Per-feature | On demand | Merge queue processing, conflict detection |

### 6.2 Patrol Overlap as Resilience

Multiple agents observing overlapping state is intentional redundancy:

```
               Daemon                          Senior Engineer
           (mechanical)                    (intelligent)
                │                               │
    ┌───────────┼───────────┐       ┌──────────┼──────────┐
    │           │           │       │          │          │
 Session    GUPP         Orphan   QA Engineer   Release Engineer    Cross-feature
 liveness   violations   work    health    health      convoy
    │           │           │       │          │
    └───────────┤           │       │          │
                │           │       │          │
                ▼           ▼       ▼          ▼
              QA Engineer               QA Engineer    Release Engineer
           (per-feature patrol)      (responds)   (responds)
                │
    ┌───────────┼───────────┐
    │           │           │
 Zombie      Orphaned     Stalled
 detection   tickets        agents
```

**Key property:** If any single patrol agent fails, the others detect the
resulting state degradation and compensate. The daemon detects dead sessions.
The senior engineer detects dead QA engineeres. The QA engineer detects dead agents.

### 6.3 Information Flow Between Patrol Agents

```
Daemon ───LIFECYCLE:──────▶ QA Engineer inbox
Daemon ───GUPP_VIOLATION:─▶ QA Engineer inbox
Daemon ───ORPHANED_WORK:──▶ QA Engineer inbox

Senior Engineer ◀──heartbeat.json──── Daemon
Senior Engineer ───nudge────────────▶ QA Engineer (if stale)
Senior Engineer ───nudge────────────▶ Release Engineer (if stale)

QA Engineer ──MERGE_READY:────▶ Release Engineer inbox
QA Engineer ──RECOVERED_BEAD:─▶ Senior Engineer (for re-dispatch)
QA Engineer ──patrol receipt───▶ Tickets (audit trail)

Release Engineer ─MERGED:─────────▶ QA Engineer inbox
Release Engineer ─MERGE_FAILED:───▶ QA Engineer inbox
Release Engineer ─convoy check─────▶ Senior Engineer (for stranded convoys)
```

### 6.4 Convergent State

All patrol agents converge on the same observable state: tickets (via Dolt), git
(via branches and worktrees), and tmux (via session liveness). No agent maintains
private state that others depend on. This is the "discover, don't track" principle
applied to monitoring.

If state diverges (e.g., a message is lost), the next patrol cycle re-derives
state from observables and self-heals.

---

## 7. Resolved Design Questions

### Q1: Spoon-Feeding and Step Granularity

**Question:** How many logical steps per physical molecule step? How many steps
per agent session?

**Answer:** Use formulas to define granularity, and let context pressure determine
session boundaries.

**Step granularity guidelines:**

| Step Type | Granularity | Example |
|-----------|-------------|---------|
| Setup / teardown | One physical step | "Set up working branch" |
| Implementation | One per logical unit | "Implement the solution" (may span sessions) |
| Verification | One per check type | "Run quality checks", "Self-review" |
| Handoff | One per lifecycle event | "Commit changes", "Submit work" |

The `mol-agent-work` formula currently uses 10 steps. This is appropriate for
most work because:

- Each step has clear entry/exit criteria
- Steps are independently resumable (a crash mid-step loses at most one step's work)
- Context stays focused (one step's instructions, not the whole molecule)

**Session-per-step is a guideline, not a rule.** A agent may complete multiple
steps in one session if context permits. The key constraint is that each step
is closed individually (no batch-closing — the Batch-Closure Heresy).

**Anti-patterns:**
- Steps so small they're just `git add` commands (overhead exceeds value)
- Steps so large they exhaust context (implementation + testing + review in one step)
- Steps that can't be independently resumed (step 3 requires step 2's context window)

### Q2: Mechanical vs Agent-Driven Recycling

**Question:** When is mechanical intervention (daemon-driven) appropriate vs
agent-driven (agent requests its own recycle)?

**Answer:** Prefer explicit self-recycling. Use mechanical intervention only as a
safety net.

**The spectrum:**

```
AGENT-DRIVEN (preferred)              MECHANICAL (safety net)
├── gt done (agent goes idle)       ├── Daemon detects dead session
├── gt handoff (agent self-cycles)  ├── Daemon detects GUPP violation
├── gt escalate (agent asks help)   ├── QA Engineer zombie sweep
└── HELP mail (agent signals)       └── Senior Engineer restart on stale heartbeat
```

**Design principle:** The agent is the authority on its own state. External
intervention should only occur when the agent cannot speak for itself (dead
session, hung process, stuck-in-done).

**Concrete thresholds (agent-determined, not hardcoded):**

The daemon uses broad thresholds for safety-net detection:
- **GUPP violation:** 30 minutes with `hook_ticket` but no progress
- **Hung session:** 30 minutes of no tmux output (`HungSessionThresholdMinutes`)
- **Stuck-in-done:** 60 seconds with `done-intent` label

These thresholds are intentionally generous. The goal is to catch truly stuck
agents, not agents that are thinking hard. False positives (the "Senior Engineer
murder spree" bug) are worse than slow detection.

**The murder spree lesson:** Mechanical detection of "stuck" is fragile because
distinguishing "thinking deeply" from "hung" requires intelligence. This is why
Boot exists (intelligent triage) and why the daemon's thresholds are conservative.
Only the QA engineer (an AI agent) should make judgment calls about whether a agent
is truly stuck.

### Q3: Channel Implementation

**Question:** Mail-based, tickets-based, or state file?

**Answer:** Mail-based. See [Section 4](#4-per-feature-agent-channel) for full design.

**Why not tickets-based (special ticket type)?**
- Tickets tickets are durable work artifacts. Lifecycle requests are transient signals.
- Creating/closing tickets for "recycle me" adds unnecessary Dolt write pressure.
- Mail is already the coordination primitive and has the featureht lifecycle (read → process → delete).

**Why not state files (feature/agent-queue.json)?**
- State files require explicit locking for concurrent access.
- No audit trail (file gets overwritten).
- Doesn't integrate with existing patrol patterns (agents already check mail).
- Recovery after crash is harder (partially-written JSON).

### Q4: Who Spawns the Next Step?

**Question:** After a agent completes a step and hands off, who spawns the
next session to continue the molecule?

**Answer:** The QA engineer, tfeaturegered by either handoff detection or daemon lifecycle
request.

**The spawn chain:**

```
Agent completes step
    │
    ├── Closes step ticket
    ├── Calls gt handoff (creates handoff mail)
    └── Session exits
         │
         ▼
Daemon heartbeat tick
    │
    ├── Detects dead agent session
    ├── Finds hook_ticket still set (work isn't done)
    └── Tfeaturegers session restart
         │
         ▼
SessionManager.Start()
    │
    ├── Creates new tmux session in existing worktree
    ├── Injects env vars (GT_POLECAT, GT_RIG)
    ├── SessionStart hook fires: gt prime --hook
    └── New session discovers next step via bd mol current
```

**Current implementation:** The daemon's `processLifecycleRequests()` handles
this. When a session dies but the hook is still set, the daemon either sends a
`LIFECYCLE:` message to the QA engineer or directly restarts the session (depending
on configuration). Agent startup is handled end-to-end by the GUPP/beacon
flow (SessionManager → StartupNudge → BuildStartupPrompt → SessionStart hook
→ gt prime).

**Future (AT integration):** The QA engineer spawns replacement teammates directly
via `Teammate({ operation: "spawn" })`. The SubagentStop hook detects teammate
death and tfeaturegers respawn. See `docs/design/QA engineer-at-team-lead.md` for details.

---

## 8. Edge Cases and Failure Modes

### 8.1 The Stuck-in-Done Zombie

A agent runs `gt done` but the session hangs before cleanup completes.

**Detection:** QA Engineer `DetectZombieAgents()` checks for `done-intent` label
older than 60 seconds with a live session.

**Recovery:** QA Engineer kills the session and continues the cleanup pipeline
(verify `cleanup_status`, forward to release engineer if MR exists).

### 8.2 The Orphaned Sandbox

A agent directory exists but no tmux session and no `hook_ticket`.

**Detection:** `Manager.ReconcilePool()` finds directories without sessions.
`DetectStaleAgents()` identifies sandboxes far behind main with no work.

**Recovery:** If no uncommitted work and no active MR, nuke the sandbox. If
uncommitted work exists, escalate (someone needs to decide if the work matters).

### 8.3 The Split-Brain Merge

The release engineer starts merging while the agent is still pushing.

**Prevention:** The `cleanup_status=clean` field on the agent ticket serializes
this. The QA engineer only sends `MERGE_READY` after verifying the agent has
exited and the branch is clean. The merge slot provides additional serialization.

### 8.4 The Infinite Cycle

A step keeps failing and the session keeps restarting.

**Detection:** Track crash count per agent (via `ReconcilePool` or
ephemeral state). Three crashes on the same step tfeaturegers escalation.

**Recovery:** QA Engineer stops respawning, creates a bug ticket, mails the product manager.
The molecule stays in its current state (recoverable when the bug is fixed).

### 8.5 Concurrent Agents on Same Ticket

Should not happen because the hook is exclusive (one `hook_ticket` per agent ticket,
one agent ticket per agent name). But if it does:

**Prevention:** Git branch naming includes a unique suffix (`@<timestamp>`).
The TOCTOU guard in `DetectZombieAgents()` (records `detectedAt`, re-verifies
before destructive action) prevents racing between detection and action.

**Recovery:** The second session fails to push (branch diverged) and escalates.

---

## 9. Future: AT Integration Impact

The Agent Teams (AT) integration (see `docs/design/QA engineer-at-team-lead.md`)
changes the transport layer but preserves the lifecycle model:

| Aspect | Current (tmux) | Future (AT) |
|--------|---------------|-------------|
| Session management | tmux sessions | AT teammates |
| Spawning | `SessionManager.Start()` | `Teammate({ operation: "spawn" })` |
| Health monitoring | tmux liveness + pane output | AT lifecycle hooks (SubagentStop) |
| Messaging | `gt nudge` (tmux send-keys) | AT messaging |
| Cleanup | Session kill (sandbox preserved) | `Teammate({ operation: "requestShutdown" })` (sandbox preserved) |

**What stays the same:**
- Tickets as the durable ledger
- Molecules as workflow templates
- `gt done` as the agent idle signal
- Two-stage cleanup (step vs molecule)
- Mail for cross-feature communication
- The completion guarantee (GUPP + pinned work + respawn)

**What changes:**
- The QA engineer becomes an AT team lead (delegate mode)
- Zombie detection becomes structural (hooks vs polling)
- Agent-to-agent isolation is hook-enforced, not tmux-enforced
- Real-time coordination moves from tmux to AT (ephemeral), reducing Dolt pressure

---

## 10. Implementation Status (gt-o8g8 audit, 2026-03-07)

### Shipped

All core lifecycle operations are implemented and running in production:

| Operation | Command/Component | Key Implementation |
|-----------|------------------|-------------------|
| Spawn/assign | `gt sling` | `sling.go`, `agent_spawn.go` — finds idle agent or allocates new slot |
| Work execution | `gt prime --hook` | Session discovers hook via `bd mol current`, GUPP fires |
| Session cycling | `gt handoff` | `handoff.go` — all roles, preserves sandbox and identity |
| Step completion | `bd close` + `gt handoff` | Step cleanup: session dies, sandbox lives |
| Work submission | `gt done` | `done.go` — push, MR, sandbox sync, set idle |
| Idle agent reuse | `gt sling` | `agent/manager.go`: `FindIdleAgent()` + `ReuseIdleAgent()` — branch-only repair |
| Zombie detection | QA Engineer patrol | `QA engineer/handlers.go`: `DetectZombieAgents()` — restart-first, no auto-nuke |
| Stale detection | QA Engineer patrol | `agent/manager.go`: `DetectStaleAgents()` — tmux-based, protects paused states |
| Orphan recovery | QA Engineer patrol | `QA engineer/handlers.go`: `DetectOrphanedTickets()` — reset and re-dispatch |
| Cleanup pipeline | Mail-based | POLECAT_DONE → QA Engineer → MERGE_READY → Release Engineer → MERGED |
| Merge queue | Release Engineer | Squash-merge, close MR and ticket, convoy check |

### Pending

| Feature | Description | Impact |
|---------|-------------|--------|
| Release Engineer notifies product manager after merge | PRs #2436/#2437 closed; branch cleanup shipped, product manager notify not yet | Unblocks dependent work dispatch |

### Deferred (design only)

| Feature | Rationale for deferral |
|---------|----------------------|
| Pool size enforcement | On-demand allocation works; fixed pool is optimization, not correctness |
| `gt agent pool init` | Agents created naturally by first `gt sling`; pre-allocation unnecessary |
| `ReconcilePool()` | QA Engineer patrol already detects state drift via zombie/stale/orphan checks |

---

## 11. Summary

See [concepts/agent-lifecycle.md](../concepts/agent-lifecycle.md) for the
complete lifecycle model (three layers, four states, persistent agent design).
This document covers the implementation details: cleanup stages, mail channels,
patrol coordination, and edge case handling.
