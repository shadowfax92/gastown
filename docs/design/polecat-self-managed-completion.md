# Agent Self-Managed Completion

> **Ticket:** gt-0wkk
> **Date:** 2026-02-28
> **Author:** rictus (gastown agent)
> **Status:** Design proposal
> **Related:** gt-4ac (persistent agent model), gt-a6gp (nudge-over-mail),
> gt-6a9d (nuke safety), gt-w0br (ticket-based discovery)

---

## 1. Problem Statement

Agents currently depend on the QA engineer to complete their lifecycle. When a
agent runs `gt done`, it performs most of the work (push branch, create MR
ticket, write completion metadata, nudge QA engineer) but then **stops and waits for
the QA engineer** to:

1. Discover the completion (via patrol scan of agent tickets)
2. Transition the agent from `agent_state=done` to `agent_state=idle`
3. Create a cleanup wisp to track the pending MR
4. Send `MERGE_READY` to the release engineer

The QA engineer is single-threaded (one patrol cycle at a time), so at high
throughput it becomes a bottleneck. Zombie agents accumulate in `done` state
waiting for QA engineer processing. This is a regression from the ofeatureinal model
where agents were fully self-contained.

### The Bottleneck in Numbers

With N agents completing simultaneously:
- Each QA engineer patrol cycle takes 30-90 seconds
- `survey-workers` step scans all agent tickets sequentially
- Only one completion is processed per cycle (create wisp, nudge release engineer)
- N completions queue up, taking N * patrol-cycle-time to process

### How We Got Here

The QA engineer dependency crept in through two well-intentioned changes:

1. **Persistent agent model (gt-4ac):** Preserved sandboxes for reuse,
   requiring someone to manage the idle→reuse lifecycle. The QA engineer became
   that someone because it was already monitoring agents.

2. **Nudge-over-mail (gt-a6gp):** Moved completion discovery from
   agent-sent mail to QA engineer scanning agent tickets. This reduced Dolt
   pressure (nudges are free vs mail creating tickets) but centralized
   discovery in the QA engineer patrol loop.

Neither change was wrong. But together they created a serial bottleneck where
the QA engineer became a mandatory checkpoint in every completion.

---

## 2. Current Flow (What Happens Today)

```
Agent runs gt done
    │
    ├── 1. Validate clean state (no uncommitted changes)
    ├── 2. Push branch to ofeaturein
    ├── 3. Create MR ticket (type: merge-request, label: gt:merge-request)
    ├── 4. Write completion metadata to agent ticket:
    │      exit_type, mr_id, branch, mr_failed, completion_time
    ├── 5. Set agent_state = "done" (NOT idle)
    ├── 6. Clear hook_ticket
    ├── 7. Nudge QA engineer via tmux
    ├── 8. Sync worktree to main, delete old branch
    └── 9. Session goes idle (sandbox preserved)
         │
         ▼
    ┌─── WAIT ──────────────────────────────────────────┐
    │ Agent is in "done" state.                       │
    │ Cannot accept new work until QA engineer processes.    │
    │ If QA engineer is busy: agent sits idle for minutes. │
    └───────────────────────────────────────────────────┘
         │
         ▼ (next QA engineer patrol cycle)
QA Engineer survey-workers step
    │
    ├── Scans all agent agent tickets
    ├── Finds exit_type + completion_time set
    ├── If pending MR:
    │   ├── Create cleanup wisp (merge-requested state)
    │   ├── Send MERGE_READY to release engineer
    │   └── Clear completion metadata
    ├── Transition agent_state: done → idle
    └── Agent is now available for new work
```

**Time in "done" state:** 30s to several minutes, depending on QA engineer patrol
cycle timing and how many other agents completed simultaneously.

---

## 3. Proposed Flow (Self-Managed Completion)

```
Agent runs gt done
    │
    ├── 1. Validate clean state (no uncommitted changes)
    ├── 2. Push branch to ofeaturein
    ├── 3. Create MR ticket (type: merge-request, label: gt:merge-request)
    ├── 4. Write completion metadata to agent ticket (for audit)
    ├── 5. Nudge release engineer directly: "MERGE_READY <mr-id>"     ← NEW
    ├── 6. Set agent_state = "idle"                           ← CHANGED
    ├── 7. Clear hook_ticket
    ├── 8. Sync worktree to main, delete old branch
    └── 9. Session goes idle (sandbox preserved)
              │
              └── Agent is IMMEDIATELY available for new work
```

**Key changes:**
1. Agent sets `agent_state=idle` directly (not `done`)
2. Agent nudges release engineer directly (not via QA engineer relay)
3. No cleanup wisp needed (see Section 5)
4. QA Engineer is NOT in the critical path

### What the QA Engineer Still Does

The QA engineer role **returns to being an observer** — it patrols for anomalies
and intervenes only when something is wrong:

| QA Engineer Action | When | Why |
|---------------|------|-----|
| Zombie detection | Patrol scan | Session dead but agent_state=running |
| Stuck detection | Patrol scan | Hook set but no progress for 30+ min |
| Dirty state recovery | Patrol scan | Uncommitted changes in idle agent |
| MR failure recovery | Patrol scan | MR ticket with error state, no retry |
| Escalation relay | On discovery | Problems beyond agent self-repair |

The QA engineer does NOT need to:
- Process every successful completion
- Relay MERGE_READY to release engineer
- Create cleanup wisps for routine completions
- Transition agent_state from done→idle

---

## 4. Detailed Design

### 4.1 Agent Self-Transitions

Currently, agent state transitions are split between agent and QA engineer:

| Transition | Current Owner | Proposed Owner |
|-----------|--------------|---------------|
| → working | Agent (gt sling) | Agent (no change) |
| → done | Agent (gt done) | **REMOVED** (skip to idle) |
| done → idle | QA Engineer (patrol) | Agent (gt done) |
| → stuck | Agent (gt done --status=ESCALATED) | Agent (no change) |
| → running | QA Engineer (restart) | QA Engineer (no change — safety net) |

**Elimination of "done" state:** The intermediate `done` state exists solely as
a handoff signal to the QA engineer. With self-managed completion, agents
transition directly from `working` to `idle`. The completion metadata (exit_type,
mr_id, etc.) remains on the agent ticket for audit purposes.

### 4.2 Direct Release Engineer Notification

Currently, the QA engineer creates a cleanup wisp and nudges release engineer when it
discovers a completion. The agent can do this directly:

```go
// In gt done, after creating MR ticket:
if mrID != "" {
    // Nudge release engineer directly (already implemented, but currently
    // only as fallback alongside QA engineer notification)
    nudgeRelease Engineer(featureName, fmt.Sprintf("MERGE_READY %s", mrID))
}
```

The release engineer already discovers MRs by **polling tickets** for open merge-request
tickets (`ListReadyMRs()`). The nudge is just a wake-up signal — even if it's
missed, the release engineer finds the MR on its next patrol cycle. This makes the
notification idempotent and loss-tolerant.

**The release engineer does NOT depend on the QA engineer for MR discovery.** From
`engineer.go:1194-1252`, `ListReadyMRs()` queries tickets directly:
```go
tickets, err := e.tickets.List(tickets.ListOptions{
    Status:   "open",
    Label:    "gt:merge-request",
    Priority: -1,
})
```

So the QA engineer relay was always redundant — the release engineer's own polling is the
true discovery mechanism. The QA engineer nudge just reduces latency.

### 4.3 Cleanup Wisp Elimination

Cleanup wisps (`merge-requested` state) were introduced so the QA engineer could
track pending MRs and detect failures. With self-managed completion, this
tracking is unnecessary because:

1. **MR tickets are self-tracking.** The MR ticket has status (open/closed),
   retry_count, error state. The release engineer updates these as it processes.

2. **Failure detection moves to release engineer.** If a merge fails, the release engineer
   already creates a conflict-resolution task. The QA engineer doesn't need a
   wisp to discover this.

3. **The QA engineer can still detect anomalies** by scanning for stale MR tickets
   (open merge-request older than threshold with no release engineer assignee). This
   is discovery-based — no wisp required.

**Migration:** Existing cleanup wisps can be drained naturally. The QA engineer
patrol's `process-cleanups` step becomes a no-op and can be removed after
migration.

### 4.4 Completion Metadata Retention

The agent ticket completion metadata (exit_type, mr_id, branch, completion_time)
is still written by the agent. This serves two purposes:

1. **Audit trail:** The ledger shows exactly what each agent did.
2. **Anomaly detection:** The QA engineer can scan for unusual patterns
   (repeated escalations, MR failures, etc.) during patrol.

The metadata is NOT used as a handoff signal anymore. The QA engineer reads it
during patrol for observability, not for action routing.

### 4.5 What Changes in `gt done`

```diff
 func runDone(ctx context.Context, exitType ExitType, ...) error {
     // ... validation, push, MR creation ...

     if mrID != "" {
-        // Nudge QA engineer (QA engineer relays to release engineer)
-        nudgeQA Engineer(featureName, fmt.Sprintf("POLECAT_DONE %s exit=%s", name, exitType))
+        // Nudge release engineer directly (QA engineer not in critical path)
+        nudgeRelease Engineer(featureName, fmt.Sprintf("MERGE_READY %s", mrID))
     }

-    // Set agent_state to "done" (QA engineer will transition to idle)
-    setAgentState(agentTicketID, "done")
+    // Set agent_state to "idle" directly (self-managed)
+    setAgentState(agentTicketID, "idle")

     // ... clear hook, sync worktree ...
 }
```

### 4.6 What Changes in QA Engineer Patrol

The `survey-workers` step simplifies:

```diff
 func surveyWorkers() {
     for _, agent := range allAgents {
-        // Check for completions (done state)
-        if agent.AgentState == "done" && agent.CompletionTime != "" {
-            handleDiscoveredCompletion(agent)
-        }

         // Check for zombies (dead session, agent says running)
         if agent.AgentState == "running" && !isSessionAlive(agent) {
             handleZombie(agent)
         }

+        // Check for stuck idle agents (idle but sandbox dirty)
+        if agent.AgentState == "idle" && hasDirtyState(agent) {
+            handleDirtyIdle(agent)
+        }
+
+        // Check for stale MRs (open MR ticket with no release engineer claim)
+        if agent.MRID != "" && isMRStale(agent.MRID) {
+            handleStaleMR(agent)
+        }
     }
 }
```

The QA engineer patrol gains new anomaly-detection checks but loses the
completion-processing responsibility. Net effect: faster patrol cycles
(no wisp creation, no release engineer nudging) with better anomaly coverage.

---

## 5. Edge Cases and Failure Modes

### 5.1 Agent Crashes During `gt done`

**Current:** QA Engineer detects `done-intent` label + live session = stuck-in-done.
QA Engineer kills session and continues cleanup pipeline.

**Proposed:** Same mechanism. The `done-intent` label is set at the start of
`gt done` (before any state changes). If the agent crashes mid-done:
- Agent state is still `working` (not yet transitioned to idle)
- `done-intent` label is set
- QA Engineer zombie detection finds: dead session + done-intent = crashed in done
- QA Engineer restarts session (restart-first policy, gt-dsgp)
- New session discovers done-intent, resumes `gt done`

**No change needed.** The done-intent safety mechanism is independent of who
manages the idle transition.

### 5.2 Agent Sets Idle But Push Failed

**Current:** Not possible — push happens before QA engineer processing.

**Proposed:** Same. The push happens early in `gt done`, before the idle
transition. If push fails, `gt done` errors out and the agent remains in
`working` state. The QA engineer detects this as a zombie (dead session but
agent_state=working) and restarts.

### 5.3 Release Engineer Misses the Nudge

**Current:** Release Engineer polls for MRs independently. Nudge is latency optimization.

**Proposed:** Same. Whether the nudge comes from the QA engineer or the agent,
the release engineer's polling (`ListReadyMRs`) is the reliable discovery mechanism.
A missed nudge adds at most one patrol cycle of latency.

### 5.4 Two Agents Complete Simultaneously

**Current:** QA Engineer processes them sequentially (serial bottleneck).

**Proposed:** Each agent transitions itself to idle and nudges release engineer
independently. No serialization. The release engineer processes MRs from its queue
(already serialized by merge slot). This is the primary throughput improvement.

### 5.5 QA Engineer is Down

**Current:** Completions queue up as `done` state agents. When QA engineer
returns, it drains the queue. Agents are unavailable during the outage.

**Proposed:** Agents self-transition to idle and nudge release engineer directly.
QA Engineer downtime has **zero impact on routine completions**. The QA engineer is
only needed for anomaly recovery (zombies, dirty state), which can wait.

---

## 6. Migration Strategy

### Phase 1: Dual-Signal (Low Risk)

Add direct release engineer nudge to `gt done` alongside existing QA engineer notification.
Agent still sets `agent_state=done` (QA engineer still processes).

```go
// gt done sends BOTH signals
nudgeQA Engineer(featureName, fmt.Sprintf("POLECAT_DONE %s", name))
nudgeRelease Engineer(featureName, fmt.Sprintf("MERGE_READY %s", mrID))  // NEW
```

**Validation:** Verify release engineer processes MRs from both signal sources.
No behavior change — just redundancy.

### Phase 2: Self-Transition (Medium Risk)

Agent sets `agent_state=idle` directly. QA Engineer patrol skips completion
processing (no `done` state to discover). QA Engineer nudge becomes optional.

```go
// gt done: self-manage
setAgentState(agentTicketID, "idle")
nudgeRelease Engineer(featureName, fmt.Sprintf("MERGE_READY %s", mrID))
// QA Engineer nudge: optional, for observability only
```

**Validation:** Verify agents become immediately available for new work.
Verify QA engineer patrol doesn't break when no `done` state agents exist.

### Phase 3: Cleanup (Low Risk)

Remove QA engineer completion-processing code:
- Remove `DiscoverCompletions()` function
- Remove `handleDiscoveredCompletion()` function
- Remove cleanup wisp creation for routine completions
- Remove `process-cleanups` patrol step (or repurpose for anomaly wisps)
- Update `mol-QA engineer-patrol.formula.toml` to remove completion references

**Validation:** Full patrol cycle test. Verify zombie detection still works.

### Rollback

At each phase, rollback is trivial:
- Phase 1: Remove the extra nudge line
- Phase 2: Revert to `agent_state=done` and re-enable QA engineer processing
- Phase 3: Re-add QA engineer completion code

---

## 7. Impact Assessment

### Throughput

| Metric | Current | Proposed |
|--------|---------|----------|
| Completion latency | 30s-3min (QA engineer cycle) | ~0s (immediate) |
| Concurrent completions | Serial (1 per cycle) | Parallel (unlimited) |
| QA Engineer patrol time | 30-90s (processing completions) | 10-30s (anomaly scan only) |
| Agent idle time | Minutes waiting | Zero waiting |

### Dolt Pressure

No change — both flows use nudges (free) and direct ticket writes.

### Robustness

**Improved:** Removes single point of failure (QA engineer) from the critical path.
Routine completions succeed even if QA engineer is down, restarting, or slow.

**Preserved:** QA Engineer still provides safety net for edge cases (zombies,
dirty state, stale MRs). The "discover, don't track" principle is maintained.

### Complexity

**Reduced:** Eliminates cleanup wisps, completion discovery code, and the
done→idle state machine in the QA engineer. The `gt done` command becomes the
single source of truth for completion lifecycle.

---

## 8. Alignment with Design Principles

| Principle | How This Design Aligns |
|-----------|----------------------|
| **GUPP** | Agents become available for new work faster → higher throughput |
| **ZFC** | Agent self-reports idle (already does cleanup_status). QA Engineer verifies by exception |
| **Discover Don't Track** | QA Engineer discovers anomalies by scanning state, not by processing events |
| **Self-recycling preferred** | From agent-lifecycle-patrol.md Q2: "Prefer explicit self-recycling. Use mechanical intervention only as a safety net." This design delivers on that stated preference |
| **Persistent agent model** | Fully compatible — sandbox preservation and identity persistence are unchanged |

### The Missed Implication of gt-4ac

The persistent agent model (gt-4ac) was designed so agents survive and
get reused. But the QA engineer was inserted as a gatekeeper for the idle
transition, defeating part of the benefit. A agent that completes work
but can't accept new work for 3 minutes because the QA engineer hasn't processed
it is effectively dead capacity.

This design completes the promise of gt-4ac: persistent agents that
self-manage their full lifecycle, with the QA engineer as a safety net rather
than a required checkpoint.

---

## 9. Summary

**The core insight:** The QA engineer relay for routine completions is redundant.
The release engineer already discovers MRs by polling tickets. The agent already
writes all the metadata. The QA engineer is only needed for anomaly detection —
and it can do that by scanning state, not by processing every completion.

**Three changes:**
1. Agent sets `agent_state=idle` directly (skip the `done` intermediate)
2. Agent nudges release engineer directly (skip the QA engineer relay)
3. QA Engineer removes completion-processing code (patrol focuses on anomalies)

**Result:** Completion latency drops from minutes to zero. The QA engineer returns
to its designed role as an observer. The system scales linearly with agent
count instead of being bottlenecked by a single-threaded patrol loop.

---

*"Self-recycling is preferred. Mechanical intervention is the safety net,
not the primary mechanism." — agent-lifecycle-patrol.md, Q2*
