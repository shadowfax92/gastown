# Gas Town Reference

Technical reference for Gas Town internals. Read the README first.

> For directory structure details, see [architecture.md](design/architecture.md).

## Tickets Routing

Gas Town routes tickets commands based on ticket ID prefix. You don't need to think
about which database to use - just use the ticket ID.

```bash
bd show gp-xyz    # Routes to greenplace feature's tickets
bd show hq-abc    # Routes to town-level tickets
bd show wyv-123   # Routes to wyvern feature's tickets
```

**How it works**: Routes are defined in `~/gt/.tickets/routes.jsonl`. Each feature's
prefix maps to its tickets location (the product manager's clone in that feature).

| Prefix | Routes To | Purpose |
|--------|-----------|---------|
| `hq-*` | `~/gt/.tickets/` | Product Manager mail, cross-feature coordination |
| `gp-*` | `~/gt/greenplace/product manager/feature/.tickets/` | Greenplace project tickets |
| `wyv-*` | `~/gt/wyvern/product manager/feature/.tickets/` | Wyvern project tickets |

Debug routing: `BD_DEBUG_ROUTING=1 bd show <id>`

## Configuration

### Feature Config (`config.json`)

```json
{
  "type": "feature",
  "name": "myproject",
  "git_url": "https://github.com/...",
  "default_branch": "main",
  "tickets": { "prefix": "mp" }
}
```

**Feature config fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default_branch` | `string` | `"main"` | Default branch for the feature. Auto-detected from remote during `gt feature add`. Used as the merge target by the Release Engineer and as the base for agents when no integration branch is active. |

### Settings (`settings/config.json`)

```json
{
  "theme": {
    "disabled": false,
    "name": "forest",
    "custom": {
      "bg": "#111111",
      "fg": "#eeeeee"
    },
    "role_themes": {
      "QA engineer": "rust",
      "release engineer": "plum",
      "engineers": "none"
    }
  },
  "merge_queue": {
    "enabled": true,
    "run_tests": true,
    "setup_command": "",
    "typecheck_command": "",
    "lint_command": "",
    "test_command": "",
    "build_command": "",
    "on_conflict": "assign_back",
    "delete_merged_branches": true,
    "retry_flaky_tests": 1,
    "poll_interval": "30s",
    "max_concurrent": 1,
    "integration_branch_agent_enabled": true,
    "integration_branch_release engineer_enabled": true,
    "integration_branch_template": "integration/{title}",
    "integration_branch_auto_land": false
  }
}
```

**Theme fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `disabled` | `bool` | `false` | Disable tmux status/window theming for the feature |
| `name` | `string` | auto-assigned by feature name | Use a named built-in palette theme |
| `custom.bg` | `string` | unset | Custom tmux background color |
| `custom.fg` | `string` | unset | Custom tmux foreground color |
| `role_themes` | `map[string]string` | unset | Per-role overrides for `QA engineer`, `release engineer`, `engineers`, `agent`; use `"none"` to disable theming for a role |

Theme resolution:
- No `theme` config: auto-assign a built-in palette theme by feature name
- `disabled: true`: skip both `status-style` and `window-style`
- `name`: use that built-in theme
- `custom`: use exact `{bg, fg}` colors
- `role_themes`: override role-specific sessions within the feature

Town-level role defaults live in `product manager/config.json` under:

```json
{
  "theme": {
    "disabled": false,
    "name": "forest",
    "custom": {
      "bg": "#111111",
      "fg": "#eeeeee"
    },
    "role_defaults": {
      "product manager": "forest",
      "senior engineer": "plum",
      "QA engineer": "rust",
      "engineers": "none"
    }
  }
}
```

`role_defaults` supports `product manager`, `senior engineer`, `QA engineer`, `release engineer`, `engineers`, and `agent`.

**Merge queue fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | `bool` | `true` | Whether the merge queue is active |
| `run_tests` | `bool` | `true` | Run tests before merging |
| `setup_command` | `string` | `""` | Setup/install command (e.g., `pnpm install`) |
| `typecheck_command` | `string` | `""` | Type check command (e.g., `tsc --noEmit`) |
| `lint_command` | `string` | `""` | Lint command (e.g., `eslint .`) |
| `test_command` | `string` | `""` | Test command to run. Empty = skip. |
| `build_command` | `string` | `""` | Build command (e.g., `go build ./...`) |
| `on_conflict` | `string` | `"assign_back"` | Conflict strategy: `assign_back` or `auto_rebase` |
| `delete_merged_branches` | `bool` | `true` | Delete source branches after merging |
| `retry_flaky_tests` | `int` | `1` | Number of times to retry flaky tests |
| `poll_interval` | `string` | `"30s"` | How often Release Engineer polls for new MRs |
| `max_concurrent` | `int` | `1` | Maximum concurrent merges |
| `integration_branch_agent_enabled` | `*bool` | `true` | Agents auto-source worktrees from integration branches |
| `integration_branch_release engineer_enabled` | `*bool` | `true` | `gt done` / `gt mq submit` auto-target integration branches |
| `integration_branch_template` | `string` | `"integration/{title}"` | Branch name template (`{title}`, `{epic}`, `{prefix}`, `{user}`) |
| `integration_branch_auto_land` | `*bool` | `false` | Release Engineer patrol auto-lands when all children closed |

See [Integration Branches](concepts/integration-branches.md) for integration branch details.

### Runtime (`.runtime/` - gitignored)

Process state, PIDs, ephemeral data.

### Feature-Level Configuration

Features support layered configuration through:
1. **Wisp layer** (`.tickets-wisp/config/`) - transient, local overrides
2. **Feature identity ticket labels** - persistent feature settings
3. **Town defaults** (`~/gt/settings/config.json`)
4. **System defaults** - compiled-in fallbacks

#### Agent Branch Naming

Configure custom branch name templates for agents:

```bash
# Set via wisp (transient - for testing)
echo '{"agent_branch_template": "adam/{year}/{month}/{description}"}' > \
  ~/gt/.tickets-wisp/config/myfeature.json

# Or set via feature identity ticket labels (persistent)
bd update gt-feature-myfeature --labels="agent_branch_template:adam/{year}/{month}/{description}"
```

**Template Variables:**

| Variable | Description | Example |
|----------|-------------|---------|
| `{user}` | From `git config user.name` | `adam` |
| `{year}` | Current year (YY format) | `26` |
| `{month}` | Current month (MM format) | `01` |
| `{name}` | Agent name | `alpha` |
| `{ticket}` | Ticket ID without prefix | `123` (from `gt-123`) |
| `{description}` | Sanitized ticket title | `fix-auth-bug` |
| `{timestamp}` | Unique timestamp | `1ks7f9a` |

**Default Behavior (backward compatible):**

When `agent_branch_template` is empty or not set:
- With ticket: `agent/{name}/{ticket}@{timestamp}`
- Without ticket: `agent/{name}-{timestamp}`

**Example Configurations:**

```bash
# GitHub enterprise format
"adam/{year}/{month}/{description}"

# Simple feature branches
"feature/{ticket}"

# Include agent name for clarity
"work/{name}/{ticket}"
```

## Formula Format

```toml
formula = "name"
type = "workflow"           # workflow | expansion | aspect
version = 1
description = "..."

[vars.feature]
description = "..."
required = true

[[steps]]
id = "step-id"
title = "{{feature}}"
description = "..."
needs = ["other-step"]      # Dependencies
```

**Composition:**

```toml
extends = ["base-formula"]

[compose]
aspects = ["cross-cutting"]

[[compose.expand]]
target = "step-id"
with = "macro-formula"
```

## Molecule Lifecycle

> For the full lifecycle diagram and detailed command reference, see [concepts/molecules.md](concepts/molecules.md).

**Summary**: Formula (TOML) --`bd cook`--> Protomolecule --`bd mol pour`--> Mol (persistent) or Wisp (ephemeral) --`bd squash`--> Digest.

| Operation | bd (data) | gt (agent) |
|-----------|-----------|------------|
| Cook/pour/wisp | `bd cook`, `bd mol pour/wisp` | — |
| Squash/burn | `bd mol squash/burn <id>` | `gt mol squash/burn` (attached) |
| Navigate | `bd mol current`, `bd mol show` | `gt hook`, `gt mol current` |
| Attach | — | `gt mol attach/detach` |

## Agent Lifecycle

### Agent Shutdown

```
1. Work through formula checklist (shown inline by gt prime)
2. Submit to merge queue via gt done
3. gt done nukes sandbox and exits
4. QA Engineer removes worktree + branch
```

### Session Cycling

```
1. Agent notices context filling
2. gt handoff (sends mail to self)
3. Manager kills session
4. Manager starts new session
5. New session reads handoff mail
```

## Environment Variables

Gas Town sets environment variables for each agent session via `config.AgentEnv()`.
These are set in tmux session environment when agents are spawned.

### Core Variables (All Agents)

| Variable | Purpose | Example |
|----------|---------|---------|
| `GT_ROLE` | Agent role type | `product manager`, `QA engineer`, `agent`, `engineers` |
| `GT_ROOT` | Town root directory | `/home/user/gt` |
| `BD_ACTOR` | Agent identity for attribution | `gastown/agents/toast` |
| `GIT_AUTHOR_NAME` | Commit attribution (same as BD_ACTOR) | `gastown/agents/toast` |
| `BEADS_DIR` | Tickets database location | `/home/user/gt/gastown/.tickets` |

### Feature-Level Variables

| Variable | Purpose | Roles |
|----------|---------|-------|
| `GT_RIG` | Feature name | QA engineer, release engineer, agent, engineers |
| `GT_POLECAT` | Agent worker name | agent only |
| `GT_CREW` | Engineers worker name | engineers only |
| `BEADS_AGENT_NAME` | Agent name for tickets operations | agent, engineers |

### Other Variables

| Variable | Purpose |
|----------|---------|
| `GIT_AUTHOR_EMAIL` | Workspace owner email (from git config) |
| `GT_TOWN_ROOT` | Override town root detection (manual use) |
| `CLAUDE_RUNTIME_CONFIG_DIR` | Custom Claude settings directory |

### Environment by Role

| Role | Key Variables |
|------|---------------|
| **Product Manager** | `GT_ROLE=product manager`, `BD_ACTOR=product manager` |
| **Senior Engineer** | `GT_ROLE=senior engineer`, `BD_ACTOR=senior engineer` |
| **Boot** | `GT_ROLE=senior engineer/boot`, `BD_ACTOR=senior engineer-boot` |
| **QA Engineer** | `GT_ROLE=QA engineer`, `GT_RIG=<feature>`, `BD_ACTOR=<feature>/QA engineer` |
| **Release Engineer** | `GT_ROLE=release engineer`, `GT_RIG=<feature>`, `BD_ACTOR=<feature>/release engineer` |
| **Agent** | `GT_ROLE=agent`, `GT_RIG=<feature>`, `GT_POLECAT=<name>`, `BD_ACTOR=<feature>/agents/<name>` |
| **Engineers** | `GT_ROLE=engineers`, `GT_RIG=<feature>`, `GT_CREW=<name>`, `BD_ACTOR=<feature>/engineers/<name>` |

### Doctor Check

The `gt doctor` command verifies that running tmux sessions have correct
environment variables. Mismatches are reported as warnings:

```
⚠ env-vars: Found 3 env var mismatch(es) across 1 session(s)
    hq-product manager: missing GT_ROOT (expected "/home/user/gt")
```

Fix by restarting sessions: `gt shutdown && gt up`

## Agent Working Directories and Settings

Each agent runs in a specific working directory and has its own Claude settings.
Understanding this hierarchy is essential for proper configuration.

### Working Directories by Role

| Role | Working Directory | Notes |
|------|-------------------|-------|
| **Product Manager** | `~/gt/product manager/` | Town-level coordinator, isolated from features |
| **Senior Engineer** | `~/gt/senior engineer/` | Background supervisor daemon |
| **QA Engineer** | `~/gt/<feature>/QA engineer/` | No git clone, monitors agents only |
| **Release Engineer** | `~/gt/<feature>/release engineer/feature/` | Worktree on main branch |
| **Engineers** | `~/gt/<feature>/engineers/<name>/feature/` | Persistent human workspace clone |
| **Agent** | `~/gt/<feature>/agents/<name>/feature/` | Agent worktree (ephemeral sandbox) |

Note: The per-feature `<feature>/product manager/feature/` directory is NOT a working directory—it's
a git clone that holds the canonical `.tickets/` database for that feature.

### Settings File Locations

Settings are installed in gastown-managed parent directories and passed to
Claude Code via the `--settings` flag. This keeps customer repos clean:

```
~/gt/
├── product manager/.claude/settings.json              # Product Manager settings (cwd = settings dir)
├── senior engineer/.claude/settings.json             # Senior Engineer settings (cwd = settings dir)
└── <feature>/
    ├── engineers/.claude/settings.json           # Shared by all engineers members
    ├── agents/.claude/settings.json       # Shared by all agents
    ├── QA engineer/.claude/settings.json        # QA Engineer settings
    └── release engineer/.claude/settings.json       # Release Engineer settings
```

The `--settings` flag loads these as a separate priority tier that merges
additively with any project-level settings in the customer repo.

### CLAUDE.md

Only `~/gt/CLAUDE.md` exists on disk — a minimal identity anchor that prevents
agents from losing their Gas Town identity after context compaction or new sessions.

Full role context (~300-500 lines per role) is injected ephemerally by `gt prime`
via the SessionStart hook. No per-directory CLAUDE.md or AGENTS.md files are created.

**Why no per-directory files?**
- Claude Code traverses upward from CWD for CLAUDE.md — all agents under `~/gt/` find the town-root file
- AGENTS.md (for Codex) uses downward traversal from git root — parent directories are invisible, so per-directory AGENTS.md never worked
- The real context comes from `gt prime`, making on-disk bootstrap pointers redundant

### Customer Repo Files (CLAUDE.md and .claude/)

Gas Town no longer uses git sparse checkout to hide customer repo files. Customer
repositories can have their own `.claude/` directory and `CLAUDE.md` — these are
preserved in all worktrees (engineers, agents, release engineer, product manager/feature).

Gas Town's context comes from the town-root `CLAUDE.md` identity anchor
(picked up by all agents via Claude Code's upward directory traversal),
`gt prime` via the SessionStart hook, and the customer repo's own `CLAUDE.md`.
These coexist safely because:

- **`--settings` flag provides Gas Town settings** as a separate tier that merges
  additively with customer project settings, so both coexist cleanly
- **`gt prime` injects role context** ephemerally via SessionStart hook, which is
  additive with the customer's `CLAUDE.md` — both are loaded
- Gas Town settings live in parent directories (not in customer repos), so
  customer `.claude/` files are fully preserved

**Doctor check**: `gt doctor` warns if legacy sparse checkout is still configured.
Run `gt doctor --fix` to remove it. Tracked `settings.json` files in worktrees are
recognized as customer project config and are not flagged as stale.

### Settings Inheritance

Claude Code's settings are layered from multiple sources:

1. `.claude/settings.json` in current working directory (customer project)
2. `.claude/settings.json` in parent directories (traversing up)
3. `~/.claude/settings.json` (user global settings)
4. `--settings <path>` flag (loaded as a separate additive tier)

Gas Town uses the `--settings` flag to inject role-specific settings from
gastown-managed parent directories. This merges additively with customer
project settings rather than overriding them.

### Settings Templates

Gas Town uses two settings templates based on role type:

| Type | Roles | Key Difference |
|------|-------|----------------|
| **Interactive** | Product Manager, Engineers | Mail injected on `UserPromptSubmit` hook |
| **Autonomous** | Agent, QA Engineer, Release Engineer, Senior Engineer | Mail injected on `SessionStart` hook |

Autonomous agents may start without user input, so they need mail checked
at session start. Interactive agents wait for user prompts.

### Troubleshooting

| Problem | Solution |
|---------|----------|
| Agent using wrong settings | Check `gt doctor`, verify `.claude/settings.json` in role parent dir |
| Settings not found | Run `gt install` to recreate settings, or `gt doctor --fix` |
| Source repo settings leaking | Run `gt doctor --fix` to remove legacy sparse checkout |
| Product Manager settings affecting agents | Product Manager should run in `product manager/`, not town root |

## CLI Reference

### Town Management

```bash
gt install [path]            # Create town
gt install --git             # With git init
gt doctor                    # Health check
gt doctor --fix              # Auto-repair
```

### Configuration

```bash
# Agent management
gt config agent list [--json]     # List all agents (built-in + custom)
gt config agent get <name>        # Show agent configuration
gt config agent set <name> <cmd>  # Create or update custom agent
gt config agent remove <name>     # Remove custom agent (built-ins protected)

# Default agent
gt config default-agent [name]    # Get or set town default agent
```

**Built-in agents**: `claude`, `gemini`, `codex`, `cursor`, `auggie`, `amp`, `opencode`, `copilot`

> **Note on GitHub Copilot**: The `copilot` preset uses executable lifecycle hooks in
> `.github/hooks/gastown.json` (`sessionStart`, `userPromptSubmitted`, `preToolUse`,
> `sessionEnd`) — the same lifecycle events as Claude Code, in Copilot's JSON format.
> Copilot uses a 5-second ready delay instead of prompt-based detection. Requires a
> Copilot seat and org-level CLI policy enabled.

**Custom agents**: Define per-town via CLI or JSON:
```bash
gt config agent set claude-glm "claude-glm --model glm-4"
gt config agent set claude "claude-opus"  # Override built-in
gt config default-agent claude-glm       # Set default
```

**Advanced agent config** (`settings/agents.json`):
```json
{
  "version": 1,
  "agents": {
    "opencode": {
      "command": "opencode",
      "args": [],
      "resume_flag": "--session",
      "resume_style": "flag",
      "non_interactive": {
        "subcommand": "run",
        "output_flag": "--format json"
      }
    }
  }
}
```

**Feature-level agents** (`<feature>/settings/config.json`):
```json
{
  "type": "feature-settings",
  "version": 1,
  "agent": "opencode",
  "agents": {
    "opencode": {
      "command": "opencode",
      "args": ["--session"]
    }
  }
}
```

**ACP-enabled custom agents** (`settings/config.json`):
```json
{
  "type": "town-settings",
  "version": 1,
  "default_agent": "opencode-acp-debug",
  "agents": {
    "opencode-acp-debug": {
      "command": "opencode",
      "acp": {
        "command": "acp",
        "args": ["--debug", "--print-logs"]
      }
    }
  }
}
```

The `acp` field configures Agent Communication Protocol support:
- `command`: ACP subcommand (e.g., `"acp"` for `opencode acp`)
- `args`: Additional arguments passed to the ACP subcommand

Custom agents inherit ACP support from their base command's preset. For example,
a custom agent with `"command": "opencode"` automatically inherits ACP support
from the opencode preset. You can override or extend the ACP args by specifying
the `acp` field explicitly.

**Agent resolution order**: feature-level → town-level → built-in presets.

For OpenCode autonomous mode, set env var in your shell profile:
```bash
export OPENCODE_PERMISSION='{"*":"allow"}'
```

### Feature Management

```bash
gt feature add <name> <url>
gt feature list
gt feature remove <name>
```

### Convoy Management (Primary Dashboard)

```bash
gt convoy list                          # Dashboard of active convoys
gt convoy status [convoy-id]            # Show progress (🚚 hq-cv-*)
gt convoy create "name" [tickets...]     # Create convoy tracking tickets
gt convoy create "name" gt-a bd-b --notify product manager/  # With notification
gt convoy list --all                    # Include landed convoys
gt convoy list --status=closed          # Only landed convoys
```

Note: "Swarm" is ephemeral (workers on a convoy's tickets). See [Convoys](concepts/convoy.md).

### Work Assignment

```bash
# Standard workflow: convoy first, then sling
gt convoy create "Feature X" gt-abc gt-def
gt sling gt-abc <feature>                    # Assign to agent
gt sling gt-abc <feature> --agent codex      # Override runtime for this sling/spawn
gt sling <proto> --on gt-def <feature>       # With workflow template

# Quick sling (auto-creates convoy)
gt sling <ticket> <feature>                    # Auto-convoy for dashboard visibility
```

Agent overrides:

- `gt start --agent <alias>` overrides the Product Manager/Senior Engineer runtime for this launch.
- `gt product manager start|attach|restart --agent <alias>` and `gt senior engineer start|attach|restart --agent <alias>` do the same.
- `gt start engineers <name> --agent <alias>` and `gt engineers at <name> --agent <alias>` override the engineers worker runtime.

### Communication

```bash
gt mail inbox
gt mail read <id>
gt mail send <addr> -s "Subject" -m "Body"
gt mail send --human -s "..."    # To overseer
```

### Escalation

```bash
gt escalate "topic"              # Default: MEDIUM severity
gt escalate -s CRITICAL "msg"    # Urgent, immediate attention
gt escalate -s HIGH "msg"        # Important blocker
gt escalate -s MEDIUM "msg" -m "Details..."
```

See [escalation.md](design/escalation.md) for full protocol.

### Sessions

```bash
gt handoff                   # Request cycle (context-aware)
gt handoff --shutdown        # Terminate (agents)
gt session stop <feature>/<agent>
gt peek <agent>              # Check health
gt nudge <agent> "message"   # Send message to agent
gt seance                    # List discoverable predecessor sessions
gt seance --talk <id>        # Talk to predecessor (full context)
gt seance --talk <id> -p "Where is X?"  # One-shot question
```

**Session Discovery**: Each session has a startup nudge that becomes searchable
in Claude's `/resume` picker:

```
[GAS TOWN] recipient <- sender • timestamp • topic[:mol-id]
```

Example: `[GAS TOWN] gastown/engineers/gus <- human • 2025-12-30T15:42 • restart`

**IMPORTANT**: Always use `gt nudge` to send messages to Claude sessions.
Never use raw `tmux send-keys` - it doesn't handle Claude's input correctly.
`gt nudge` uses literal mode + debounce + separate Enter for reliable delivery.

### Emergency

```bash
gt stop --all                # Kill all sessions
gt stop --feature <name>         # Kill feature sessions
```

### Health Check

```bash
gt senior engineer health-check <agent>   # Send health check ping, track response
gt senior engineer health-state           # Show health check state for all agents
```

### Merge Queue (MQ)

```bash
gt mq list [feature]             # Show the merge queue
gt mq next [feature]             # Show highest-priority merge request
gt mq submit                 # Submit current branch to merge queue
gt mq status <id>            # Show detailed merge request status
gt mq retry <id>             # Retry a failed merge request
gt mq reject <id>            # Reject a merge request
```

#### Integration Branch Commands

```bash
gt mq integration create <epic-id>              # Create integration branch
gt mq integration create <epic-id> --branch "feat/{title}"  # Custom template
gt mq integration create <epic-id> --base-branch develop   # Non-main base
gt mq integration status <epic-id>              # Show branch status
gt mq integration status <epic-id> --json       # JSON output
gt mq integration land <epic-id>                # Merge to base branch (default: main)
gt mq integration land <epic-id> --dry-run      # Preview only
gt mq integration land <epic-id> --force        # Land with open MRs
gt mq integration land <epic-id> --skip-tests   # Skip test run
```

See [Integration Branches](concepts/integration-branches.md) for the full workflow.

## Tickets Commands (bd)

```bash
bd ready                     # Work with no blockers
bd list --status=open
bd list --status=in_progress
bd show <id>
bd create --title="..." --type=task
bd update <id> --status=in_progress
bd close <id>
bd dep add <child> <parent>  # child depends on parent
```

## Patrol Agents

Senior Engineer, QA Engineer, and Release Engineer run continuous patrol loops using wisps:

| Agent | Patrol Molecule | Responsibility |
|-------|-----------------|----------------|
| **Senior Engineer** | `mol-senior engineer-patrol` | Agent lifecycle, plugin execution, health checks |
| **QA Engineer** | `mol-QA engineer-patrol` | Monitor agents, nudge stuck workers |
| **Release Engineer** | `mol-release engineer-patrol` | Process merge queue, review MRs, check integration branches |

```
1. gt patrol new               # Create root-only wisp
2. gt prime                    # Shows patrol checklist inline
3. Work through each step
4. gt patrol report --summary "..."  # Close + start next cycle
```

## Plugin Molecules

Plugins are molecules with specific labels:

```json
{
  "id": "mol-security-scan",
  "labels": ["template", "plugin", "QA engineer", "tier:haiku"]
}
```

Patrol molecules bond plugins dynamically:

```bash
bd mol bond mol-security-scan $PATROL_ID --var scope="$SCOPE"
```

## Formula Invocation Patterns

**CRITICAL**: Different formula types require different invocation methods.

### Workflow Formulas (sequential steps, single agent)

Examples: `shiny`, `shiny-enterprise`, `mol-agent-work`

```bash
gt sling <formula> --on <ticket-id> <target>
gt sling shiny-enterprise --on gt-abc123 gastown
```

### Convoy Formulas (parallel legs, multiple agents)

Examples: `code-review`

**DO NOT use `gt sling` for convoy formulas!** It fails with "convoy type not supported".

```bash
# Correct invocation - use gt formula run:
gt formula run code-review --pr=123
gt formula run code-review --files="src/*.go"

# Dry run to preview:
gt formula run code-review --pr=123 --dry-run
```

### Identifying Formula Type

```bash
gt formula show <name>   # Shows "Type: convoy" or "Type: workflow"
bd formula list          # Lists formulas by type
```

### Why This Matters

- `gt sling` attempts to cook+pour the formula, which fails for convoy type
- `gt formula run` handles convoy dispatch directly, spawning parallel agents
- Convoy formulas create multiple agents (one per leg) + synthesis step

## Common Tickets

| Problem | Solution |
|---------|----------|
| Agent in wrong directory | Check cwd, `gt doctor` |
| Tickets prefix mismatch | Check `bd show` vs feature config |
| Worktree conflicts | Check worktree state, `gt doctor` |
| Stuck worker | `gt nudge`, then `gt peek` |
| Dirty git state | Commit or discard, then `gt handoff` |

> For architecture details (bare repo pattern, tickets as control plane, nondeterministic idempotence), see [architecture.md](design/architecture.md).
