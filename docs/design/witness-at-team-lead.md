# QA Engineer AT Team Lead: Implementation Spec

> **Status: Future architecture — NOT YET IMPLEMENTED**
> The current system uses tmux-based session management. This document describes
> a planned architectural change to use Claude Code Agent Teams (AT) as the
> transport layer. No code for this exists yet.

> **Ticket:** gt-ky4jf
> **Date:** 2026-02-08
> **Author:** furiosa (gastown agent)
> **Depends on:** AT spike report (gt-3nqoz), AT integration design (agent-teams-integration.md)
> **Status:** Phase 1 implementation spec

---

## Overview

This document specifies how the QA Engineer becomes an AT team lead, replacing the
current tmux-based agent session management with Claude Code Agent Teams.

The QA Engineer enters delegate mode (structurally enforced coordination-only), spawns
agent teammates for assigned work, monitors them via AT's native lifecycle hooks,
and syncs completions to tickets at task boundaries.

**What changes:** Session management layer (tmux → AT).
**What stays:** Tickets as ledger, gt mail for cross-feature, molecules/formulas, `gt done`.

---

## AT Spike Findings Summary

> Condensed from the AT spike report (gt-3nqoz, 2026-02-08, author: nux).

**Recommendation: CONDITIONAL GO for Phase 1 experiment.**

### Go/No-Go Decision Matrix

| Criterion | Status | Notes |
|-----------|--------|-------|
| Teammate working directories | WORKAROUND | PreToolUse hook for enforcement |
| Hooks fire for teammates | GO | All relevant hooks confirmed |
| Custom agent definitions | GO | `.claude/agents/*.md` works |
| Delegate mode enforcement | GO | Structural, not behavioral |
| Teammate cycling | WORKAROUND | Handoff + respawn pattern |
| Token cost acceptable | CONDITIONAL | Sonnet teammates reduce cost |
| gt/bd command access | GO | PATH via SessionStart hook |
| Task list with dependencies | GO | Native match to Gas Town workflow |

5/8 clear GO. 2 require workarounds (viable mitigations). 1 conditional on Phase 1 cost validation.

### Critical Blockers

1. **No per-teammate working directory** — AT teammates inherit lead's cwd. Workaround: `cd` in spawn prompt + PreToolUse hook (`gt validate-worktree-scope`) for structural enforcement.
2. **No session resumption for teammates** — Crashed teammates cannot resume. Workaround: PreCompact handoff + tickets state recovery + QA Engineer respawn.
3. **Token cost ~7x per teammate** — Mitigated by using Sonnet for agent teammates, Opus for QA Engineer lead only.

### Risk Register Summary

| Risk Level | Key Risks |
|------------|-----------|
| **High** | No per-teammate cwd, no session resumption, experimental feature |
| **Medium** | 7x token cost, hook compatibility gaps, AT API changes |
| **Low** | PATH/env setup, task list mapping, delegate mode gaps |

### Key Advantage

AT's file-locked task claiming eliminates Dolt write contention (estimated 80-90% reduction). This is the strongest argument for adoption.

---

## 1. QA Engineer in Delegate Mode

### What Tools the QA Engineer Keeps

In delegate mode, the QA Engineer has access to:

| Tool | Purpose |
|------|---------|
| `Teammate` | Spawn/shutdown teammates, send messages, manage team |
| `TaskCreate` | Create AT tasks for agent work |
| `TaskUpdate` | Update task status, set dependencies |
| `TaskList` | Monitor team progress |
| `TaskGet` | Read task details |
| `Bash` | **Not available** in delegate mode |
| `Read/Write/Edit` | **Not available** in delegate mode |
| `Glob/Grep` | **Not available** in delegate mode |

### The ZFC Upgrade

Current state: "QA Engineer doesn't implement" is enforced by CLAUDE.md instructions.
Agents can and do violate this under pressure.

New state: Delegate mode structurally removes implementation tools. The QA Engineer
literally *cannot* edit files. This is the strongest possible ZFC compliance —
the constraint is in the machinery, not in the instructions.

### QA Engineer Needs Bash for gt/bd Commands

**Problem:** Delegate mode removes Bash access, but the QA Engineer needs to run
`gt mail`, `bd show`, `bd close`, and other coordination commands.

**Solution options (in order of preference):**

1. **Custom agent definition with selective tools.** Create
   `.claude/agents/QA engineer-lead.md` that uses `permissionMode: delegate` but
   adds back Bash via the `tools` allowlist. This gives structural enforcement
   for file editing while preserving command access:

   ```yaml
   ---
   name: QA engineer-lead
   permissionMode: delegate
   tools: Teammate, TaskCreate, TaskUpdate, TaskList, TaskGet, Bash
   ---
   ```

   **Risk:** Bash access means the QA Engineer *could* edit files via sed/echo.
   Mitigated by: PreToolUse hook on Bash that rejects file-modifying commands.

2. **Hooks as command proxy.** The QA Engineer doesn't run commands directly.
   Instead, hooks fire at turn boundaries and execute gt/bd commands based on
   AT task state. The QA Engineer coordinates purely through AT tools; the hooks
   handle the tickets bridge.

   **Risk:** Less flexible — QA Engineer can't make ad-hoc bd queries. But it's
   the purest delegate mode implementation.

3. **Teammate as command runner.** Spawn a lightweight "ops" teammate whose
   sole job is running gt/bd commands on the QA Engineer's behalf. The QA Engineer
   sends commands via AT messaging; the ops teammate executes and returns results.

   **Risk:** Token overhead for a simple command proxy. But it preserves
   strict delegate mode for the QA Engineer.

**Recommendation:** Option 1 (custom agent with selective tools). It's pragmatic,
preserves the QA Engineer's ability to query tickets state, and the PreToolUse hook
provides sufficient guardrails. Pure delegate mode is aspirational but the
QA Engineer genuinely needs to read tickets state for coordination decisions.

### PreToolUse Guard for QA Engineer Bash

```json
{
  "PreToolUse": [{
    "matcher": "Bash",
    "hooks": [{
      "type": "command",
      "command": "gt QA engineer-bash-guard"
    }]
  }]
}
```

The `gt QA engineer-bash-guard` script:
- Allows: `gt *`, `bd *`, `git status`, `git log`, read-only commands
- Blocks: `echo >`, `cat >`, `sed -i`, `vim`, `nano`, any write operation
- Returns exit code 2 with reason on block

---

## 2. Teammate Spawn: Work Assignment → AT Task Creation

### The Spawn Flow

When work arrives (via convoy, gt sling, or direct assignment):

```
1. QA Engineer receives work (mail, convoy dispatch, bd ready)
2. QA Engineer creates AT team (if not already active)
3. For each ticket to dispatch:
   a. Create AT task with ticket details and dependencies
   b. Spawn agent teammate assigned to that task
4. Teammates self-claim tasks and begin execution
```

### Team Creation

```
Teammate({
  operation: "spawnTeam",
  team_name: "<feature-name>-work",
  description: "Agent work team for <convoy/sprint description>"
})
```

Team naming convention: `<feature>-work` for the primary work team.
One team per feature per active convoy. Multiple convoys = multiple teams
(AT limitation: one team per session, so QA Engineer manages one convoy
at a time).

### AT Task Creation from Tickets Tickets

For each ticket dispatched to a agent:

```
TaskCreate({
  subject: "<ticket title>",
  description: "Ticket: <ticket-id>\n<ticket description>\n\nWorktree: /path/to/<agent>/\nFormula: mol-agent-work",
  activeForm: "Working on <ticket title>",
  metadata: {
    "ticket_id": "<ticket-id>",
    "worktree": "/path/to/worktree",
    "molecule": "<mol-id>"
  }
})
```

**Key fields in metadata:**
- `ticket_id`: Links AT task back to the tickets ticket for sync
- `worktree`: The git worktree path this agent should use
- `molecule`: The mol-agent-work instance for this ticket

### Dependency Mapping

Tickets ticket dependencies map to AT task dependencies:

```
# If ticket B depends on ticket A:
# After creating both tasks:
TaskUpdate({
  taskId: "<task-B-id>",
  addBlockedBy: ["<task-A-id>"]
})
```

This enables AT's native self-claim: when task A completes, task B becomes
unblocked and the next idle teammate picks it up automatically.

### Agent Teammate Spawn

```
Task({
  subagent_type: "agent",
  team_name: "<feature>-work",
  name: "<agent-name>",
  model: "sonnet",
  prompt: "You are agent <name>. Your worktree is <path>.\n\nAssigned ticket: <id> - <title>\n<description>\n\nWorkflow:\n1. cd <worktree>\n2. Run `gt prime` for full context\n3. Follow mol-agent-work steps\n4. When done: commit, push, run `gt done`"
})
```

**Model selection:**
- Agent teammates: `model: "sonnet"` (execution-focused, cost-efficient)
- QA Engineer lead: Opus (judgment, coordination, quality review)
- Release Engineer teammate (Phase 2): `model: "sonnet"` (mechanical merge work)

### The `.claude/agents/agent.md` Definition

```yaml
---
name: agent
description: Gas Town agent worker agent (persistent identity, ephemeral sessions)
model: sonnet
hooks:
  SessionStart:
    - hooks:
        - type: command
          command: "export PATH=\"$HOME/go/bin:$HOME/.local/bin:$PATH\" && gt prime --hook"
  PreToolUse:
    - matcher: "Write|Edit"
      hooks:
        - type: command
          command: "gt validate-worktree-scope"
  PreCompact:
    - matcher: "auto"
      hooks:
        - type: command
          command: "gt handoff --reason compaction"
  Stop:
    - hooks:
        - type: command
          command: "gt signal stop"
---

You are a Gas Town agent (persistent identity, ephemeral sessions).

## Startup
1. `cd` to your assigned worktree (given in your spawn prompt)
2. Run `gt prime` for full context
3. Check your hook: `gt hook`
4. Follow molecule steps: `bd mol current`

## Work Protocol
- Mark steps in_progress before starting: `bd update <id> --status=in_progress`
- Close steps when done: `bd close <id>`
- Commit frequently with descriptive messages
- Never batch-close steps

## Completion
When all steps done:
1. `git status` — must be clean
2. `git push`
3. `gt done` — submits to merge queue, nukes your sandbox
```

### Worktree Assignment

Each agent teammate operates in its own git worktree. Since AT doesn't support
per-teammate working directories natively, enforcement is via:

1. **Spawn prompt:** First instruction is `cd /path/to/worktree`
2. **PreToolUse hook:** `gt validate-worktree-scope` rejects Write/Edit operations
   targeting paths outside the assigned worktree
3. **Environment variable:** `GT_WORKTREE=/path/to/worktree` set via SessionStart hook

The QA Engineer creates worktrees before spawning teammates:
```bash
git worktree add /path/to/agents/<name>/<feature> -b agent/<name>/<ticket-id>
```

This matches the current worktree management — the change is WHO creates them
(QA Engineer via AT, not `gt sling` via Go daemon).

---

## 3. Ticket Sync Protocol

### The Two-Layer Model

```
Layer 1 (AT, ephemeral):     Task claiming, status, messaging
Layer 2 (Tickets/Dolt, durable): Ticket creation, completion, audit trail
```

### Sync Points

| AT Event | Tickets Action | Tfeatureger |
|----------|-------------|---------|
| Task claimed (in_progress) | `bd update <id> --status=in_progress` | TaskCompleted hook / agent prompt |
| Task completed | `bd close <step-id>` | TaskCompleted hook |
| New ticket discovered | AT task created by QA Engineer | QA Engineer reads agent message |
| Teammate idle | Check tickets for more work | TeammateIdle hook |
| Team shutdown | Verify all tickets synced | QA Engineer cleanup routine |

### TaskCompleted Hook for Ticket Sync

The `TaskCompleted` hook fires when an AT task is marked complete. This is the
primary sync mechanism:

```bash
#!/bin/bash
# .claude/hooks/task-completed-sync.sh
# Fires on TaskCompleted hook

BEAD_ID=$(echo "$TASK_METADATA" | jq -r '.ticket_id // empty')
if [ -n "$BEAD_ID" ]; then
  export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"
  bd close "$BEAD_ID" 2>/dev/null
fi
exit 0
```

Hook configuration:
```json
{
  "TaskCompleted": [{
    "hooks": [{
      "type": "command",
      "command": ".claude/hooks/task-completed-sync.sh"
    }]
  }]
}
```

**Important:** The hook should NOT block task completion (exit 0 always). If the
`bd close` fails (Dolt contention), it will be retried at the next sync point.
The AT task list is the real-time truth; tickets catches up at boundaries.

### Agent-Side Ticket Updates

Agents still run `bd update` and `bd close` directly as part of their molecule
workflow. The TaskCompleted hook is a safety net, not the primary mechanism. This
means:

- Agent marks molecule step in_progress → `bd update --status=in_progress`
- Agent completes molecule step → `bd close <step-id>`
- AT task completion → TaskCompleted hook also fires `bd close` (idempotent)

Double-close is safe: `bd close` on an already-closed ticket is a no-op.

### Sync Verification at Team Shutdown

Before the QA Engineer shuts down the team, it verifies tickets are in sync:

```
For each AT task marked completed:
  1. Read task metadata for ticket_id
  2. Verify ticket is closed (bd show <id> | check status)
  3. If ticket still open: bd close <id> with notes
  4. If close fails: log warning, continue (Dolt retry will handle)
```

This is the "boundary sync" pattern from the integration design: AT handles
real-time coordination, tickets catches up at lifecycle boundaries (team shutdown,
convoy completion).

---

## 4. Session Cycling: Compaction → Respawn → Resume

### The Problem

AT teammates cannot be resumed after shutdown. When a teammate hits context
limits and compacts, or crashes, a new teammate must be spawned.

### The Lifecycle

```
Teammate running
    │
    ├── Context filling → PreCompact hook fires
    │   │
    │   └── gt handoff --reason compaction
    │       ├── Saves current molecule step to tickets
    │       ├── Saves progress notes
    │       └── Saves git branch state
    │
    ├── Auto-compaction occurs
    │   │
    │   └── SessionStart hook fires (source: "compact")
    │       └── gt prime --compact-resume
    │           └── Reads tickets state, restores context
    │
    └── Teammate continues with compressed context
```

### When Compaction Isn't Enough (Teammate Death)

If a teammate crashes or is shut down (not just compacted):

```
Teammate stops
    │
    └── SubagentStop hook fires on QA Engineer (lead)
        │
        ├── Read teammate's last known state from tickets
        │   └── Which molecule step was in_progress?
        │   └── What branch was being worked on?
        │
        ├── Assess: recoverable or escalate?
        │   ├── Normal completion: AT task done, tickets synced → no action
        │   ├── Incomplete work: respawn with resume context
        │   └── Repeated crashes: escalate to QA Engineer mail → Product Manager
        │
        └── If recoverable: spawn replacement teammate
            └── Task({ subagent_type: "agent", ... resume prompt ... })
```

### SubagentStop Hook (QA Engineer Side)

```json
{
  "SubagentStop": [{
    "matcher": "agent",
    "hooks": [{
      "type": "command",
      "command": "gt QA engineer-teammate-stopped"
    }]
  }]
}
```

The `gt QA engineer-teammate-stopped` script:
1. Reads the stopped agent's transcript path (available in hook input)
2. Checks AT task status — was the task completed?
3. Checks tickets — was `gt done` run?
4. If completed: no action (normal lifecycle)
5. If incomplete: outputs `{ "decision": "block", "reason": "Teammate <name> stopped before completing task <id>. Tickets state: <status>. Respawn needed." }`

The "block" decision prevents the QA Engineer from going idle, injecting the
respawn instruction as context for the QA Engineer to act on.

### Respawn Prompt Template

```
Teammate <name> stopped before completing work.

Last known state:
- Ticket: <ticket-id> (<title>)
- Molecule step: <step-id> (in_progress)
- Branch: <branch-name>
- Worktree: <path>

Spawn a replacement agent with this context. The new teammate
should read tickets state and continue from the last checkpoint.
```

### Crash Loop Prevention

Track respawn attempts per ticket. If a teammate crashes 3 times on the
same ticket:

1. Mark the AT task as blocked
2. File a ticket: `bd create --title "Agent crash loop on <ticket>" --type bug`
3. Mail the QA Engineer/Product Manager for escalation
4. Do NOT respawn — the ticket has a structural problem

Tracking: Use AT task metadata `{ "respawn_count": N }` incremented on
each respawn. This is ephemeral (dies with the team) which is correct —
crash tracking only matters during the current team session.

---

## 5. Error Handling

### Error Categories and Responses

| Error | Detection | Response |
|-------|-----------|----------|
| Teammate crash | SubagentStop hook | Respawn or escalate (see above) |
| Teammate stuck (no progress) | TeammateIdle hook | Send message asking for status |
| Test failures | TaskCompleted hook (exit 2) | Block completion, teammate must fix |
| Merge conflict | Agent messages QA Engineer | QA Engineer advises or reassigns |
| Dolt write failure | bd command exit code | Retry with backoff (existing mechanism) |
| AT team crash | QA Engineer session dies | Daemon/Boot/Senior Engineer chain detects, restarts QA Engineer |
| Worktree scope violation | PreToolUse hook | Block the operation, warn agent |

### TeammateIdle Hook

```bash
#!/bin/bash
# gt QA engineer-teammate-idle
# Fires when a teammate is about to go idle

export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"

# Check if there's more work in tickets
READY=$(bd ready --count 2>/dev/null)
if [ "$READY" -gt 0 ]; then
  echo "There is more work available. Run 'bd ready' to see unblocked tasks." >&2
  exit 2  # Block idle, send feedback
fi

# Check if gt done was run
if git log --oneline -1 | grep -q "gt done"; then
  exit 0  # Normal completion
fi

# Teammate seems genuinely idle without completing
echo "Your work doesn't appear complete. Run 'bd ready' to check remaining steps, or 'gt done' if finished." >&2
exit 2
```

### TaskCompleted Quality Gate

```bash
#!/bin/bash
# Fires on TaskCompleted hook
# Validates that work meets minimum quality before marking complete

export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"

# Check for uncommitted changes
if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
  echo "Uncommitted changes detected. Commit your work before marking complete." >&2
  exit 2
fi

# Check that the branch has been pushed
BRANCH=$(git branch --show-current 2>/dev/null)
if ! git log "ofeaturein/$BRANCH" --oneline -1 >/dev/null 2>&1; then
  echo "Branch not pushed to remote. Run 'git push' before completing." >&2
  exit 2
fi

exit 0
```

---

## 6. Convoy Mapping to AT Teams

### The Natural Mapping

| Gas Town | AT Equivalent |
|----------|--------------|
| Convoy | AT team lifecycle |
| Convoy tickets | AT tasks |
| War Feature (per-feature convoy execution) | AT team instance |
| Ready front (unblocked tickets) | Unblocked AT tasks |
| Dispatch | AT task creation + teammate spawn |
| Completion tracking | AT task list status |

### One Convoy = One AT Team Session

A convoy arrives at a feature. The QA Engineer creates an AT team for that convoy:

```
Convoy hq-abc arrives at gastown
    │
    ├── QA Engineer creates team: "gastown-convoy-abc"
    │
    ├── For each ticket in convoy:
    │   ├── Create AT task (with ticket_id in metadata)
    │   └── Set dependencies (from tickets dep graph)
    │
    ├── Spawn N agent teammates (N = min(tickets, max_agents))
    │
    ├── Teammates self-claim tasks from ready front
    │
    ├── As tasks complete:
    │   ├── Dependencies unblock next tasks
    │   ├── Idle teammates auto-claim newly ready tasks
    │   └── Tickets synced via TaskCompleted hook
    │
    └── All tasks done:
        ├── QA Engineer verifies tickets sync
        ├── QA Engineer sends convoy completion to Product Manager (gt mail)
        └── Team shutdown
```

### Multiple Convoys

AT limitation: one team per session. If a second convoy arrives while the
first is active:

**Option A: Sequential processing.** Finish convoy 1, then start convoy 2.
Simple, no concurrency tickets. Acceptable if convoy throughput is sufficient.

**Option B: Convoy queue.** The QA Engineer queues incoming convoys and processes
them in order. The queue lives in tickets (mail inbox) — the QA Engineer checks for
new convoys when the current team finishes.

**Option C: Multiple QA Engineer sessions.** The daemon spawns a second QA Engineer
session for the second convoy. Each QA Engineer manages its own AT team. This
requires the daemon to support multiple QA Engineer instances per feature.

**Recommendation:** Option A for Phase 1 (sequential). Option C for Phase 2+
if throughput demands it. The convoy queue in Option B is implicit in tickets
already (unprocessed convoy mail = queued work).

### Steady-State Worker Pool

For large convoys (20+ tickets), the QA Engineer doesn't spawn 20 teammates at once.
Instead:

```
max_teammates = 5  # configurable per feature

1. Spawn max_teammates agents
2. Create all AT tasks (with dependencies)
3. Teammates self-claim from ready front
4. As teammates complete tasks:
   - Auto-claim next unblocked task
   - No respawn needed (same teammate, new task)
5. When all tasks done: team shutdown
```

AT's self-claim mechanism is the key enabler. Teammates don't die after one
task — they pick up the next one. This eliminates the current spawn/nuke
overhead per ticket.

**When a teammate needs to cycle** (compaction), the QA Engineer spawns a
replacement, not an additional teammate. The pool size stays at max_teammates.

---

## 7. Mail Bridge: gt mail ↔ AT Messages

### The Boundary

```
                    ┌─────────────────┐
                    │    QA Engineer       │
                    │  (AT Team Lead)  │
                    │                  │
    gt mail ←──────│── Bridge ──────→ AT messaging
    (cross-feature,    │                  (intra-team,
     persistent)   │                   ephemeral)
                    └─────────────────┘
```

### Inbound: gt mail → AT message

When the QA Engineer receives gt mail relevant to an active teammate:

```
gt mail inbox
    │
    ├── From Product Manager: "Priority shift — ticket X is now P0"
    │   └── QA Engineer sends AT message to relevant teammate:
    │       Teammate({ operation: "write", target_agent_id: "<agent>",
    │                  value: "Priority update: <ticket> is now P0. Expedite." })
    │
    ├── From Release Engineer: "Merge conflict on <branch>"
    │   └── QA Engineer sends AT message to the agent on that branch:
    │       Teammate({ operation: "write", target_agent_id: "<agent>",
    │                  value: "Merge conflict detected. Rebase on main." })
    │
    └── From another feature's QA Engineer: "Dependency <ticket> is done"
        └── QA Engineer creates/unblocks AT task for downstream work
```

### Outbound: AT event → gt mail

When AT events need to reach entities outside the team:

```
Teammate completes final task
    │
    └── QA Engineer detects all tasks done
        │
        ├── gt mail send gastown/release engineer -s "MERGE_READY: <branch>"
        │   └── Release Engineer processes merge queue
        │
        ├── gt mail send product manager/ -s "CONVOY COMPLETE: hq-abc"
        │   └── Product Manager updates convoy tracking
        │
        └── gt mail send gastown/QA engineer -s "POLECAT_DONE: <name>"
            └── (Self-mail for tickets record)
```

### What Goes Where

| Communication | Channel | Why |
|--------------|---------|-----|
| QA Engineer ↔ Agent | AT messaging | Same team, real-time, ephemeral |
| Agent ↔ Agent | AT messaging | Same team, coordination chatter |
| QA Engineer → Release Engineer | gt mail | Different lifecycle, needs persistence |
| QA Engineer → Product Manager | gt mail | Cross-feature, needs persistence |
| Product Manager → QA Engineer | gt mail | Cross-feature, needs persistence |
| Agent escalation | AT message to QA Engineer, QA Engineer relays via gt mail | Bridge pattern |

### The Relay Pattern

Agents can't send gt mail directly to entities outside their team (AT
messaging is team-scoped). Instead:

```
Agent needs to escalate to Product Manager:
    │
    ├── Agent sends AT message to QA Engineer:
    │   "ESCALATE: Need Product Manager decision on auth approach"
    │
    └── QA Engineer relays via gt mail:
        gt mail send product manager/ -s "ESCALATE from agent <name>" -m "..."
```

This is analogous to the current model where agents mail the QA Engineer and
the QA Engineer escalates. The difference: AT messaging is real-time (no Dolt
sync lag), and the QA Engineer can relay immediately.

---

## 8. Configuration

### `.claude/settings.json` (Project Level)

```json
{
  "env": {
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1"
  },
  "hooks": {
    "TaskCompleted": [{
      "hooks": [{
        "type": "command",
        "command": ".claude/hooks/task-completed-sync.sh"
      }]
    }],
    "TeammateIdle": [{
      "hooks": [{
        "type": "command",
        "command": ".claude/hooks/teammate-idle-check.sh"
      }]
    }],
    "SubagentStop": [{
      "matcher": "agent",
      "hooks": [{
        "type": "command",
        "command": ".claude/hooks/teammate-stopped.sh"
      }]
    }]
  }
}
```

### `.claude/agents/QA engineer-lead.md`

```yaml
---
name: QA engineer-lead
description: Gas Town QA Engineer operating as AT team lead
model: opus
permissionMode: delegate
hooks:
  SessionStart:
    - hooks:
        - type: command
          command: "export PATH=\"$HOME/go/bin:$HOME/.local/bin:$PATH\" && gt prime --hook"
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "gt QA engineer-bash-guard"
  Stop:
    - hooks:
        - type: command
          command: "gt signal stop"
---

You are the Gas Town QA Engineer for this feature.

## Role
You coordinate agent workers. You NEVER implement code directly.
Delegate mode enforces this structurally — you cannot edit files.

## Startup
1. Check for incoming work: `gt mail inbox`, `bd ready`
2. Create AT team if work is available
3. Spawn agent teammates for each ticket
4. Monitor progress via AT task list

## During Work
- Monitor teammate progress via TaskList
- Relay cross-feature messages (gt mail ↔ AT messages)
- Handle teammate crashes (respawn or escalate)
- Enforce quality via plan approval

## Completion
- Verify all AT tasks completed
- Verify tickets are synced (all tickets closed)
- Send MERGE_READY to Release Engineer via gt mail
- Send convoy completion to Product Manager via gt mail
- Shutdown team
```

### `.claude/agents/agent.md`

See Section 2 above for the full definition.

---

## 9. What Gets Replaced

### Infrastructure Removed (Phase 1)

| Component | Replacement | Notes |
|-----------|-------------|-------|
| `gt sling` (agent spawn) | `Teammate({ operation: "spawn" })` | AT native |
| `gt agent nuke` | `Teammate({ operation: "requestShutdown" })` | AT native |
| tmux session management | AT manages teammate sessions | No more tmux for agents |
| `gt nudge` (tmux send-keys) | `Teammate({ operation: "write" })` | AT messaging |
| Zombie detection (tmux-based) | SubagentStop / TeammateIdle hooks | Structural |
| QA Engineer "are you stuck?" polling | TeammateIdle hook (automatic) | Event-driven |
| Agent-to-agent isolation | Prompt + PreToolUse hook | Behavioral → hook-enforced |

### Infrastructure Kept (Phase 1)

| Component | Why |
|-----------|-----|
| Tickets (Dolt) | Durable ledger — AT tasks are ephemeral |
| gt mail | Cross-feature communication — AT is team-scoped |
| Molecules/formulas | Work templates — AT tasks created from these |
| `gt done` | Agent self-clean — unchanged lifecycle |
| Git worktrees | Filesystem isolation — AT doesn't provide this |
| Daemon/Boot/Senior Engineer | Health monitoring — AT has no crash recovery |
| Release Engineer (separate) | Different lifecycle (Phase 2 brings it in-band) |
| Convoy tracking | Cross-feature work orders — above AT scope |

### Dolt Write Pressure Reduction

**Current:** Every `bd update`, `bd close`, `bd create` from every agent
= concurrent Dolt writes. 20 agents = 20+ concurrent commits.

**With AT:** Real-time task coordination happens in AT (file-locked, no Dolt).
Dolt writes only at boundaries:
- `bd close` when a molecule step completes (1 per task)
- `bd create` when agents discover new tickets (rare)

**Estimated reduction: 80-90%.** The remaining writes are naturally staggered
across minutes (task completions), not milliseconds (concurrent status updates).

---

## 10. QA Engineer Startup Flow (Updated)

```
QA Engineer session starts (managed by daemon)
    │
    ├── SessionStart hook: gt prime --hook
    │   └── Loads role context, checks hook
    │
    ├── Check for work:
    │   ├── gt mail inbox (convoy dispatch, priority changes)
    │   ├── bd ready (unblocked tickets)
    │   └── gt hook (hooked work)
    │
    ├── If work available:
    │   │
    │   ├── Create AT team:
    │   │   Teammate({ operation: "spawnTeam", team_name: "<feature>-work" })
    │   │
    │   ├── Create AT tasks from tickets tickets:
    │   │   For each ticket: TaskCreate({ subject, description, metadata: { ticket_id } })
    │   │   Set dependencies: TaskUpdate({ addBlockedBy: [...] })
    │   │
    │   ├── Create worktrees for agents:
    │   │   For each agent: git worktree add ...
    │   │
    │   ├── Spawn agent teammates:
    │   │   For each (up to max_teammates):
    │   │     Task({ subagent_type: "agent", team_name: "...", name: "..." })
    │   │
    │   └── Enter monitoring loop:
    │       ├── Watch AT task list for completions
    │       ├── Handle teammate crashes (SubagentStop)
    │       ├── Relay gt mail ↔ AT messages
    │       ├── Check for new convoy arrivals
    │       └── When all tasks done: cleanup and report
    │
    └── If no work:
        └── Stop hook checks for queued work periodically
            └── If work arrives: wake and create team
```

---

## 11. Phase 1 Scope and Validation Criteria

### In Scope

1. QA Engineer as AT team lead in delegate mode (with Bash for gt/bd)
2. Agent teammates with `.claude/agents/agent.md`
3. Ticket sync via TaskCompleted hook
4. Session cycling via PreCompact handoff + respawn
5. Basic error handling (crash detection, respawn, crash loop prevention)
6. Mail bridge (gt mail ↔ AT messaging)
7. Single-convoy sequential processing

### Out of Scope (Phase 2+)

1. Release Engineer as AT teammate
2. Multiple concurrent convoys
3. Cross-feature AT coordination
4. Engineers squads / shadow workers
5. Advanced plan approval workflows
6. Performance optimization (token cost tuning)

### Validation Criteria

| Criterion | Test |
|-----------|------|
| QA Engineer stays in delegate mode | Verify QA Engineer cannot write/edit files |
| Agents complete work | End-to-end: spawn → implement → push → gt done |
| Tickets sync correctly | AT task completion → bd close fires → ticket is closed |
| Session cycling works | Force compaction → new teammate resumes from tickets |
| Crash recovery works | Kill a teammate → QA Engineer detects → respawns |
| Mail bridge works | Product Manager sends mail → QA Engineer relays to agent |
| Dolt writes reduced | Measure bd command frequency: before vs after |
| Token cost acceptable | `/cost` shows < 3x overhead vs current model |
| Convoy completes | Full convoy lifecycle: dispatch → work → merge → done |

---

## 12. Migration Path

### Current Architecture → Phase 1

The transition is additive: AT runs alongside existing infrastructure during
validation. The QA Engineer can fall back to tmux-based management if AT fails.

```
Step 1: Enable AT feature flag in gastown .claude/settings.json
Step 2: Create .claude/agents/agent.md and .claude/agents/QA engineer-lead.md
Step 3: Implement hook scripts (task-completed-sync, teammate-idle, teammate-stopped)
Step 4: Implement gt QA engineer-bash-guard
Step 5: Implement gt validate-worktree-scope
Step 6: Implement gt QA engineer-teammate-stopped
Step 7: Update QA Engineer startup to create AT team instead of tmux agent sessions
Step 8: Test with 2 agents on a small convoy
Step 9: Validate all criteria above
Step 10: If validated: expand to 3-5 agents, larger convoys
```

### Rollback Plan

If Phase 1 fails:
1. Disable AT feature flag
2. QA Engineer reverts to tmux-based agent management
3. No tickets data lost (tickets sync is additive)
4. File lessons-learned ticket for Phase 1 retry

---

*"The transport changes. The ledger endures."*
