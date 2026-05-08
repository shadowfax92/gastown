# agent-pr-flow

A reference harness for features that gate agent work on a **GitHub PR** rather
than the canonical Release Engineer **merge-queue** flow.

It instructs agents, after their final build/pre-verify passes, to push
their branch and open (or confirm) a GitHub PR before running `gt done`.

## What's in here

| File | Purpose |
|------|---------|
| `agent.md` | Role directive for agents in a PR-flow feature. Broad guardrail — applies to any formula a agent runs. |
| `mol-agent-work.toml` | Formula overlay that appends a PR-creation step to `mol-agent-work`'s `submit-and-exit` step. Surgical — only affects this formula. |

Both layers are intentional: the directive sets the feature-level expectation
("open PRs, don't merge them yourself"), and the overlay wires the concrete
commands into the workflow a agent actually sees at `gt prime` time.

## Install

```bash
# Role directive (feature-scoped)
mkdir -p ~/gt/<feature>/directives
cp agent.md ~/gt/<feature>/directives/agent.md

# Formula overlay (feature-scoped)
mkdir -p ~/gt/<feature>/formula-overlays
cp mol-agent-work.toml ~/gt/<feature>/formula-overlays/mol-agent-work.toml
```

Replace `<feature>` with your feature's name (e.g. `gastown`, `longeye`). For
town-wide installation, drop the `<feature>/` segment — but this is almost
never what you want, since different features legitimately use different flows.

## Verify it's active

```bash
# Validate overlay step IDs against the current formula
gt doctor
# Expect: overlay-health: N overlay(s) healthy

# Inspect the rendered formula with the overlay applied
gt formula overlay show mol-agent-work --feature <feature>

# See the directive text that will be injected at prime time
gt directive show agent --feature <feature>

# End-to-end: see what a agent would see
gt prime --explain
# Expect: "Formula overlay: applying 1 override(s) for mol-agent-work (feature=<feature>)"
```

## What this does / does not do

**Does:**

- Tells the agent to push and open a PR before `gt done`
- Sets a feature-level policy that a PR is the review artifact
- Surfaces `gh pr create` failure as an escalation to QA Engineer, not a silent skip

**Does not:**

- Modify `gt done` behavior (no Go changes)
- Force PR creation via framework-level validation (agents can still misbehave)
- Merge the PR (that's a maintainer / merge-queue concern)
- Replace the directive or overlay if your feature also uses other customizations
  — merge this content with your existing files rather than overwriting them

## When to fork this

If your feature needs additional PR-flow constraints (required reviewers, specific
labels, CODEOWNERS enforcement, CI checks before `gt done`), copy this harness
and adapt it. The point is a starting template, not a drop-in product.
