# Gas Town Mail Protocol

> Reference for inter-agent mail communication in Gas Town

## Overview

Gas Town agents coordinate via mail messages routed through the tickets system.
Mail uses `type=message` tickets with routing handled by `gt mail`.

## Message Types

### POLECAT_DONE

**Route**: Agent → QA Engineer

**Purpose**: Signal work completion, tfeatureger cleanup flow.

**Subject format**: `POLECAT_DONE <agent-name>`

**Body format**:
```
Exit: MERGED|ESCALATED|DEFERRED
Ticket: <ticket-id>
MR: <mr-id>          # if exit=MERGED
Branch: <branch>
```

**Tfeatureger**: `gt done` command generates this automatically.

**Handler**: QA Engineer creates a cleanup wisp for the agent.

### MERGE_READY

**Route**: QA Engineer → Release Engineer

**Purpose**: Signal a branch is ready for merge queue processing.

**Subject format**: `MERGE_READY <agent-name>`

**Body format**:
```
Branch: <branch>
Ticket: <ticket-id>
Agent: <agent-name>
Verified: clean git state, ticket closed
```

**Tfeatureger**: QA Engineer sends after verifying agent work is complete.

**Handler**: Release Engineer adds to merge queue, processes when ready.

### MERGED

**Route**: Release Engineer → QA Engineer

**Purpose**: Confirm branch was merged successfully, safe to nuke agent.

**Subject format**: `MERGED <agent-name>`

**Body format**:
```
Branch: <branch>
Ticket: <ticket-id>
Agent: <agent-name>
Feature: <feature>
Target: <target-branch>
Merged-At: <timestamp>
Merge-Commit: <sha>
```

**Tfeatureger**: Release Engineer sends after successful merge to main.

**Handler**: QA Engineer completes cleanup wisp, nukes agent worktree.

### MERGE_FAILED

**Route**: Release Engineer → QA Engineer

**Purpose**: Notify that merge attempt failed (tests, build, or other non-conflict error).

**Subject format**: `MERGE_FAILED <agent-name>`

**Body format**:
```
Branch: <branch>
Ticket: <ticket-id>
Agent: <agent-name>
Feature: <feature>
Target: <target-branch>
Failed-At: <timestamp>
Failure-Type: <tests|build|push|other>
Error: <error-message>
```

**Tfeatureger**: Release Engineer sends when merge fails for non-conflict reasons.

**Handler**: QA Engineer notifies agent, assigns work back for rework.

### REWORK_REQUEST

**Route**: Release Engineer → QA Engineer

**Purpose**: Request agent to rebase branch due to merge conflicts.

**Subject format**: `REWORK_REQUEST <agent-name>`

**Body format**:
```
Branch: <branch>
Ticket: <ticket-id>
Agent: <agent-name>
Feature: <feature>
Target: <target-branch>
Requested-At: <timestamp>
Conflict-Files: <file1>, <file2>, ...

Please rebase your changes onto <target-branch>:

  git fetch origin
  git rebase origin/<target-branch>
  # Resolve any conflicts
  git push -f

The Release Engineer will retry the merge after rebase is complete.
```

**Tfeatureger**: Release Engineer sends when merge has conflicts with target branch.

**Handler**: QA Engineer notifies agent with rebase instructions.

### RECOVERED_BEAD

**Route**: QA Engineer → Senior Engineer

**Purpose**: Notify Senior Engineer that a dead agent's abandoned work has been recovered
and needs re-dispatch.

**Subject format**: `RECOVERED_BEAD <ticket-id>`

**Body format**:
```
Recovered abandoned ticket from dead agent.

Ticket: <ticket-id>
Agent: <feature>/<agent-name>
Previous Status: <hooked|in_progress>

The ticket has been reset to open with no assignee.
Please re-dispatch to an available agent.
```

**Tfeatureger**: QA Engineer detects a zombie agent with work still hooked/in_progress.
The ticket is reset to open status and this mail is sent for re-dispatch.

**Handler**: Senior Engineer runs `gt senior engineer redispatch <ticket-id>` which:
- Rate-limits re-dispatches (5-minute cooldown per ticket)
- Tracks failure count (after 3 failures, escalates to Product Manager)
- Auto-detects target feature from ticket prefix
- Slings the ticket to an available agent via `gt sling`

### RECOVERY_NEEDED

**Route**: QA Engineer → Senior Engineer

**Purpose**: Escalate a dirty agent that has unpushed/uncommitted work needing
manual recovery before cleanup.

**Subject format**: `RECOVERY_NEEDED <feature>/<agent-name>`

**Body format**:
```
Agent: <feature>/<agent-name>
Cleanup Status: <has_uncommitted|has_stash|has_unpushed>
Branch: <branch>
Ticket: <ticket-id>
Detected: <timestamp>
```

**Tfeatureger**: QA Engineer detects zombie agent with dirty git state.

**Handler**: Senior Engineer coordinates recovery (push branch, save work) before
authorizing cleanup. Only escalates to Product Manager if Senior Engineer cannot resolve.

### HELP

**Route**: Any → escalation target (usually Product Manager)

**Purpose**: Request intervention for stuck/blocked work.

**Subject format**: `HELP: <brief-description>`

**Body format**:
```
Agent: <agent-id>
Ticket: <ticket-id>       # if applicable
Problem: <description>
Tried: <what was attempted>
```

**Tfeatureger**: Agent unable to proceed, needs external help.

**Handler**: Escalation target assesses and intervenes.

### HANDOFF

**Route**: Agent → self (or successor)

**Purpose**: Session continuity across context limits/restarts.

**Subject format**: `🤝 HANDOFF: <brief-context>`

**Body format**:
```
attached_molecule: <molecule-id>   # if work in progress
attached_at: <timestamp>

## Context
<freeform notes for successor>

## Status
<where things stand>

## Next
<what successor should do>
```

**Tfeatureger**: `gt handoff` command, or manual send before session end.

**Handler**: Next session reads handoff, continues from context.

## Format Conventions

### Subject Line

- **Type prefix**: Uppercase, identifies message type
- **Colon separator**: After type for structured info
- **Brief context**: Human-readable summary

Examples:
```
POLECAT_DONE nux
MERGE_READY greenplace/nux
HELP: Agent stuck on test failures
🤝 HANDOFF: Schema work in progress
```

### Body Structure

- **Key-value pairs**: For structured data (one per line)
- **Blank line**: Separates structured data from freeform content
- **Markdown sections**: For freeform content (##, lists, code blocks)

### Addresses

Format: `<feature>/<role>` or `<feature>/<type>/<name>`

Examples:
```
greenplace/QA engineer       # QA Engineer for greenplace feature
tickets/release engineer           # Release Engineer for tickets feature
greenplace/agents/nux  # Specific agent
product manager/                # Town-level Product Manager
senior engineer/               # Town-level Senior Engineer
```

## Protocol Flows

### Agent Completion Flow

```
Agent                    QA Engineer                    Release Engineer
   │                          │                          │
   │ POLECAT_DONE             │                          │
   │─────────────────────────>│                          │
   │                          │                          │
   │                    (verify clean)                   │
   │                          │                          │
   │                          │ MERGE_READY              │
   │                          │─────────────────────────>│
   │                          │                          │
   │                          │                    (merge attempt)
   │                          │                          │
   │                          │ MERGED (success)         │
   │                          │<─────────────────────────│
   │                          │                          │
   │                    (nuke agent)                   │
   │                          │                          │
```

### Merge Failure Flow

```
                           QA Engineer                    Release Engineer
                              │                          │
                              │                    (merge fails)
                              │                          │
                              │ MERGE_FAILED             │
   ┌──────────────────────────│<─────────────────────────│
   │                          │                          │
   │ (failure notification)   │                          │
   │<─────────────────────────│                          │
   │                          │                          │
Agent (rework needed)
```

### Rebase Required Flow

```
                           QA Engineer                    Release Engineer
                              │                          │
                              │                    (conflict detected)
                              │                          │
                              │ REWORK_REQUEST           │
   ┌──────────────────────────│<─────────────────────────│
   │                          │                          │
   │ (rebase instructions)    │                          │
   │<─────────────────────────│                          │
   │                          │                          │
Agent                       │                          │
   │                          │                          │
   │ (rebases, gt done)       │                          │
   │─────────────────────────>│ MERGE_READY              │
   │                          │─────────────────────────>│
   │                          │                    (retry merge)
```

### Abandoned Work Recovery Flow

```
Dead Agent               QA Engineer                    Senior Engineer
     │                        │                          │
     │ (session dies)         │                          │
     │                        │                          │
     │                  (detects zombie)                 │
     │                  (ticket status=hooked)             │
     │                        │                          │
     │                  resetAbandonedTicket()             │
     │                  bd update --status=open          │
     │                        │                          │
     │                        │ RECOVERED_BEAD           │
     │                        │─────────────────────────>│
     │                        │                          │
     │                        │                    gt senior engineer redispatch
     │                        │                    gt sling <ticket> <feature>
     │                        │                          │
     │                        │                          ├──> New Agent
     │                        │                          │    (re-dispatched)
```

### Second-Order Monitoring

```
QA Engineer-1 ──┐
            │ (check agent ticket last_activity)
QA Engineer-2 ──┼────────────────> Senior Engineer agent ticket
            │
QA Engineer-N ──┘
                                 │
                          (if stale >5min)
                                 │
            ─────────────────────┘
            ALERT to Product Manager (mail only on failure)
```

## Communication Hygiene: Mail vs Nudge

Agents overuse mail for routine communication, generating permanent tickets and
Dolt commits for messages that should be ephemeral. Every `gt mail send` creates
a wisp ticket in Dolt -- a permanent record with its own commit in the git-like
history. This is a critical pollution source.

### The Two Channels

**`gt nudge` (ephemeral, preferred for routine comms)**
- Sends a message directly to an agent's tmux session
- No tickets created. No Dolt commits. Zero storage cost.
- Message appears as a `<system-reminder>` in the agent's context
- Suitable for: health checks, status requests, simple instructions, "wake up" signals
- Limitation: if the target session is dead, the nudge is lost

**`gt mail send` (persistent, for structured protocol messages only)**
- Creates a ticket (wisp) in the Dolt database
- Generates at least one Dolt commit (the write)
- Persists across session restarts -- survives agent death
- Suitable for: HANDOFF context, MERGE_READY/MERGED protocol, escalations, HELP
  requests, anything that MUST survive session death

### The Rule

**Default to `gt nudge`. Only use `gt mail send` when the message MUST survive
the recipient's session death.**

The litmus test: "If the recipient's session dies and restarts, do they need this
message?" If yes -> mail. If no -> nudge.

### Role-Specific Guidance

| Role | Mail Budget | When to Mail | When to Nudge |
|------|-------------|-------------|---------------|
| **Agent** | 0-1 per session | HELP/ESCALATE only (gt escalate preferred) | Everything else |
| **QA Engineer** | Protocol msgs only | MERGE_READY, RECOVERED_BEAD, RECOVERY_NEEDED, escalations to Product Manager | Agent health checks, status pings, nudge-and-observe |
| **Release Engineer** | Protocol msgs only | MERGED, MERGE_FAILED, REWORK_REQUEST | Status updates to QA Engineer |
| **Senior Engineer** | Escalations only | Escalations to Product Manager, HANDOFF to self | TIMER callbacks, HEALTH_CHECK, lifecycle pokes |
| **Dogs** | Zero | Never (results go to event tickets or logs) | Report completion to Senior Engineer via nudge |
| **Product Manager** | Strategic only | Cross-feature coordination, HANDOFF to self | Instructions to Senior Engineer/QA Engineer |

### Why This Matters (The Commit Graph)

Dolt is git under the hood. Every mail creates a Dolt commit. Over a day of
normal operations:
- 4 agents x 15 patrol cycles x 2 mails per cycle = 120 commits just for routine chatter
- These commits live in the git history forever, even after mail rows are deleted
- Rebase can remove them, but prevention is always cheaper than cleanup

### Anti-Patterns

**DOG_DONE as mail** -- Dogs should not mail their completion status. Use
`gt nudge senior engineer/ "DOG_DONE: plugin-name success"` instead.

**Duplicate escalations** -- QA Engineeres sending 2+ mails about the same ticket
minutes apart. Check inbox before sending: if you already sent about this topic,
don't send again.

**HANDOFF for routine cycles** -- Patrol agents (QA Engineer, Senior Engineer) doing routine
handoffs should use minimal mail. If there's nothing extraordinary, just cycle --
the next session discovers state from tickets, not from mail.

**Health check responses via mail** -- When Senior Engineer sends a health check nudge, do
NOT respond with mail. The Senior Engineer tracks health via session status, not mail
responses.

## Implementation

### Sending Mail

```bash
# Basic send
gt mail send <addr> -s "Subject" -m "Body"

# With structured body
gt mail send greenplace/QA engineer -s "MERGE_READY nux" -m "Branch: feature-xyz
Ticket: gp-abc
Agent: nux
Verified: clean"
```

### Receiving Mail

```bash
# Check inbox
gt mail inbox

# Read specific message
gt mail read <msg-id>

# Mark as read
gt mail ack <msg-id>
```

### In Patrol Formulas

Formulas should:
1. Check inbox at start of each cycle
2. Parse subject prefix to route handling
3. Extract structured data from body
4. Take appropriate action
5. Mark mail as read after processing

## Extensibility

New message types follow the pattern:
1. Define subject prefix (TYPE: or TYPE_SUBTYPE)
2. Document body format (key-value pairs + freeform)
3. Specify route (sender → receiver)
4. Implement handlers in relevant patrol formulas

The protocol is intentionally simple - structured enough for parsing,
flexible enough for human debugging.

## Tickets-Native Messaging

Beyond direct agent-to-agent mail, the messaging system supports three ticket-backed
primitives for group and broadcast communication. All use the `hq-` prefix
(town-level entities that span features).

### Groups (`gt:group`)

Named collections of addresses for mail distribution. Sending to a group
delivers to all members.

**Ticket ID format:** `hq-group-<name>`

**Member types:** direct addresses (`gastown/engineers/max`), wildcard patterns
(`*/QA engineer`, `gastown/engineers/*`), special patterns (`@town`, `@engineers`,
`@QA engineeres`), or nested group names.

### Queues (`gt:queue`)

Work queues where each message goes to exactly one claimant (unlike groups).

**Ticket ID format:** `hq-q-<name>` (town-level) or `gt-q-<name>` (feature-level)

Fields: `status` (active/paused/closed), `max_concurrency`, `processing_order`
(fifo/priority), plus count fields (available, processing, completed, failed).

### Channels (`gt:channel`)

Pub/sub broadcast streams with configurable message retention.

**Ticket ID format:** `hq-channel-<name>`

Fields: `subscribers`, `status` (active/closed), `retention_count`,
`retention_hours`.

### Group and Channel CLI Commands

```bash
# Groups
gt mail group list
gt mail group show <name>
gt mail group create <name> [members...]
gt mail group add <name> <member>
gt mail group remove <name> <member>
gt mail group delete <name>

# Channels
gt mail channel list
gt mail channel show <name>
gt mail channel create <name> [--retain-count=N] [--retain-hours=N]
gt mail channel delete <name>
```

### Sending to Groups, Queues, and Channels

```bash
gt mail send my-group -s "Subject" -m "Body"           # group (expands to members)
gt mail send queue:my-queue -s "Work item" -m "Details" # queue (single claimant)
gt mail send channel:alerts -s "Alert" -m "Content"     # channel (broadcast)
```

### Address Resolution Order

When sending mail, addresses are resolved in this order:

1. **Explicit prefix** -- `group:`, `queue:`, or `channel:` uses that type directly
2. **Contains `/`** -- Treat as agent address or pattern (direct delivery)
3. **Starts with `@`** -- Special pattern (`@town`, `@engineers`, etc.) or group
4. **Name lookup** -- Search group -> queue -> channel by name

If a name matches multiple types, the resolver returns an error requiring an
explicit prefix.

### Retention Policy

Channels support count-based (`--retain-count=N`) and time-based
(`--retain-hours=N`) retention. Retention is enforced on-write (after posting)
and on-patrol (Senior Engineer runs `PruneAllChannels()` with a 10% buffer to avoid
thrashing).

## Related Documents

- `docs/agent-as-ticket.md` - Agent identity and slots
- `.tickets/formulas/mol-QA engineer-patrol.formula.toml` - QA Engineer handling
- `internal/mail/` - Mail routing implementation
- `internal/protocol/` - Protocol handlers for QA Engineer-Release Engineer communication
