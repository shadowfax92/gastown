# Gate Ticket Instruction Template

This file defines the prose template embedded in every gate ticket's description.
The `mol-decompose-with-gates` formula reads this template, populates the
placeholders, and passes it as the `--description` to `bd create`.

## Placeholders

| Placeholder | Source | Description |
|-------------|--------|-------------|
| `{{gate_id}}` | Formula runtime | The ID of this gate ticket (filled after creation) |
| `{{plan_title}}` | Plan ticket title | Human-readable name of the plan/epic |
| `{{plan_ticket}}` | Formula var | The ticket ID of the parent plan/epic |
| `{{review_steps}}` | Resolved config | Rendered review step sections (see below) |
| `{{step_names}}` | Resolved config | Comma-separated list of step names for summary |

## Template

Everything below the `---` is the literal template that goes into the gate
ticket description. Placeholders are substituted at creation time.

---

## Context

This is a verification gate for plan **{{plan_title}}** (`{{plan_ticket}}`).

You are a gate agent. Your job is to execute the review steps listed below
against the implementation work that was done under this plan. You have no
memory of the implementation — everything you need is in this description and
in the code on the current branch.

**Configured review steps:** {{step_names}}

**How this gate works:**
- This ticket is blocked by all implementation tasks under the plan.
- When you receive this ticket, all implementation is complete.
- Execute each review step below in order.
- If ALL steps pass cleanly, close this gate ticket (signals plan is ready).
- If ANY step finds tickets, follow the retry loop protocol at the bottom.

## Review Steps

{{review_steps}}

<!-- Each review step is rendered from .gates.toml as:

### <step.name>

<step.description>

**Instructions:**

<step.instructions>

-->

## Retry Loop Protocol

When a review step finds tickets, do NOT close this gate ticket. Instead:

**1. File one fix ticket per ticket found:**

```bash
bd create "Fix: <concise ticket description>" \
  --type=task \
  --description="## Context
Found during gate review of {{plan_title}} ({{plan_ticket}}).
Gate ticket: {{gate_id}}
Review step: <which step found this>

## Ticket
<detailed description of what's wrong>

## Location
<file paths, line numbers, function names>

## Expected Fix
<what the fix should accomplish>

## Acceptance Criteria
- <how to verify the fix is correct>"
```

**2. Add each fix as a blocking dependency on THIS gate ticket:**

```bash
bd dep add {{gate_id}} <fix-ticket-id>
```

This ensures the gate ticket becomes blocked again until all fixes are closed.

**3. Sling each fix to a agent:**

```bash
gt sling <fix-ticket-id> <feature>
```

This dispatches the fix work to an available worker.

**4. Exit — do NOT wait for fixes:**

Clear your assignment and let your session end. You are done for now.

The system handles the rest:
- Fix agents work on each fix ticket independently.
- As each fix ticket closes, its blocking dep on this gate is satisfied.
- When ALL fix tickets close, this gate ticket becomes unblocked again.
- The stranded-ticket scan re-dispatches this gate ticket to a fresh agent
  within 30 seconds of becoming unblocked.
- The fresh agent re-runs all review steps from the top.

**Important:** Do NOT attempt to fix tickets yourself. Your role is gate review,
not implementation. File precise fix tickets so fix agents can execute without
ambiguity.

## Clean Pass Signal

When ALL review steps pass with no tickets found:

```bash
bd close {{gate_id}} --reason="Gate passed: all review steps clean"
```

Closing the gate ticket signals that the plan's implementation has passed
verification and is ready for human review.

**Before closing, verify:**
- You executed every configured review step ({{step_names}})
- No tickets were found in any step
- You are not skipping a step due to confusion or error

If you are uncertain whether something is an ticket, err on the side of filing
a fix ticket. False positives waste less time than missed defects.
