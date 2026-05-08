# Gas Town Escalation Protocol

> Reference for the unified escalation system in Gas Town.

## Overview

Gas Town agents escalate tickets when automated resolution is not possible.
Escalations are severity-routed, tracked as tickets, and support stale detection
with automatic re-escalation.

## Severity Levels

| Level | Priority | Description | Default Route |
|-------|----------|-------------|---------------|
| **CRITICAL** | P0 (urgent) | System-threatening, immediate attention | ticket + mail + email + SMS |
| **HIGH** | P1 (high) | Important blocker, needs human soon | ticket + mail + email |
| **MEDIUM** | P2 (normal) | Standard escalation, human at convenience | ticket + mail product manager |

## Tiered Escalation Flow

```
Agent -> gt escalate -s <SEVERITY> "description"
           |
           v
     [Senior Engineer receives]
           |
           +-- resolves --> updates ticket, re-slings work
           +-- cannot  --> forwards to Product Manager
                              +-- resolves --> updates ticket, re-slings
                              +-- cannot  --> forwards to Overseer --> resolves
```

Each tier can resolve OR forward. The chain is tracked via ticket comments.

## Configuration

Config file: `~/gt/settings/escalation.json`

### Default Configuration

```json
{
  "type": "escalation",
  "version": 1,
  "routes": {
    "medium": ["ticket", "mail:product manager"],
    "high": ["ticket", "mail:product manager", "email:human"],
    "critical": ["ticket", "mail:product manager", "email:human", "sms:human"]
  },
  "contacts": {
    "human_email": "",
    "human_sms": "",
    "slack_webhook": "",
    "smtp_host": "",
    "smtp_port": "587",
    "smtp_from": "",
    "smtp_user": "",
    "smtp_pass": "",
    "sms_webhook": ""
  },
  "stale_threshold": "4h",
  "max_reescalations": 2
}
```

### Action Types

| Action | Format | Behavior |
|--------|--------|----------|
| `ticket` | `ticket` | Create escalation ticket (always first, implicit) |
| `mail:<target>` | `mail:product manager` | Send gt mail to target |
| `email:human` | `email:human` | Send email to `contacts.human_email` |
| `sms:human` | `sms:human` | Send SMS to `contacts.human_sms` |
| `slack` | `slack` | Post to `contacts.slack_webhook` |
| `log` | `log` | Write to escalation log file |

## Escalation Tickets

Escalation tickets use `type: escalation` with structured labels for tracking.

### Label Schema

| Label | Values | Purpose |
|-------|--------|---------|
| `severity:<level>` | MEDIUM, HIGH, CRITICAL | Current severity |
| `source:<type>:<name>` | plugin:rebuild-gt, patrol:senior engineer | What tfeaturegered it |
| `acknowledged:<bool>` | true, false | Has human acknowledged |
| `reescalated:<bool>` | true, false | Has been re-escalated |
| `reescalation_count:<n>` | 0, 1, 2, ... | Times re-escalated |
| `original_severity:<level>` | MEDIUM, HIGH | Initial severity |

## Category Routing (future)

Categories provide structured routing based on the nature of the escalation.
Not yet implemented as CLI flags; currently use `--to` for explicit routing.

| Category | Description | Default Route |
|----------|-------------|---------------|
| `decision` | Multiple valid paths, need choice | Senior Engineer -> Product Manager |
| `help` | Need guidance or expertise | Senior Engineer -> Product Manager |
| `blocked` | Waiting on unresolvable dependency | Product Manager |
| `failed` | Unexpected error, can't proceed | Senior Engineer |
| `emergency` | Security or data integrity ticket | Overseer (direct) |
| `gate_timeout` | Gate didn't resolve in time | Senior Engineer |
| `lifecycle` | Worker stuck or needs recycle | QA Engineer |

## Commands

### gt escalate

Create a new escalation.

```bash
gt escalate -s <MEDIUM|HIGH|CRITICAL> "Short description" \
  [-m "Detailed explanation"] [--source="plugin:rebuild-gt"]
```

Flags: `-s` severity (required), `-m` body, `--source` origin identifier,
`--to` route to tier (senior engineer/product manager/overseer), `--dry-run`, `--json`.

### gt escalate ack

Acknowledge an escalation (prevents re-escalation).

```bash
gt escalate ack <ticket-id> [--note="Investigating"]
```

### gt escalate list

```bash
gt escalate list [--severity=...] [--stale] [--unacked] [--all] [--json]
```

### gt escalate stale

Re-escalate stale (unacked past `stale_threshold`) escalations. Bumps severity
(MEDIUM->HIGH->CRITICAL), re-executes route, respects `max_reescalations`.

```bash
gt escalate stale [--dry-run]
```

### gt escalate close

```bash
gt escalate close <ticket-id> [--reason="Fixed in commit abc123"]
```

## Integration Points

### Plugin System

Plugins use escalation for failure notification:

```bash
gt escalate -s MEDIUM "Plugin FAILED: rebuild-gt" \
  -m "$ERROR" --source="plugin:rebuild-gt"
```

### Senior Engineer Patrol

Senior Engineer uses escalation for health tickets:

```bash
if [ $unresponsive_cycles -ge 5 ]; then
  gt escalate -s HIGH "QA Engineer unresponsive: gastown" \
    -m "QA Engineer has been unresponsive for $unresponsive_cycles cycles" \
    --source="patrol:senior engineer:health-scan"
fi
```

Senior Engineer patrol also runs `gt escalate stale` periodically to catch unacked
escalations and re-escalate them.

## When to Escalate

### Agents SHOULD escalate when:

- **System errors**: Database corruption, disk full, network failures
- **Security tickets**: Unauthorized access attempts, credential exposure
- **Unresolvable conflicts**: Merge conflicts that cannot be auto-resolved
- **Ambiguous requirements**: Spec is unclear, multiple valid interpretations
- **Design decisions**: Architectural choices that need human judgment
- **Stuck loops**: Agent is stuck and cannot make progress
- **Gate timeouts**: Async conditions did not resolve in expected time

### Agents should NOT escalate for:

- **Normal workflow**: Regular work that can proceed without human input
- **Recoverable errors**: Transient failures that will auto-retry
- **Information queries**: Questions that can be answered from context

## Product Manager Startup Check

On `gt prime`, Product Manager displays pending escalations grouped by severity.
Action: review with `bd list --tag=escalation`, close with `bd close <id> --reason "..."`.


## Viewing Escalations

```bash
# List all open escalations
bd list --status=open --tag=escalation

# Filter by category
bd list --tag=escalation --tag=decision

# View specific escalation
bd show <escalation-id>

# Close resolved escalation
bd close <id> --reason "Resolved by fixing X"
```
