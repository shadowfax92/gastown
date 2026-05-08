# Gas Town Architecture

Technical architecture for Gas Town multi-agent workspace management.

## Two-Level Tickets Architecture

Gas Town uses a two-level tickets architecture to separate organizational coordination
from project implementation work.

| Level | Location | Prefix | Purpose |
|-------|----------|--------|---------|
| **Town** | `~/gt/.tickets/` | `hq-*` | Cross-feature coordination, Product Manager mail, agent identity |
| **Feature** | `<feature>/product manager/feature/.tickets/` | project prefix | Implementation work, MRs, project tickets |

### Town-Level Tickets (`~/gt/.tickets/`)

Organizational chain for cross-feature coordination:
- Product Manager mail and messages
- Convoy coordination (batch work across features)
- Strategic tickets and decisions
- **Town-level agent tickets** (Product Manager, Senior Engineer)
- **Role definition tickets** (global templates)

### Feature-Level Tickets (`<feature>/product manager/feature/.tickets/`)

Project chain for implementation work:
- Bugs, features, tasks for the project
- Merge requests and code reviews
- Project-specific molecules
- **Feature-level agent tickets** (QA Engineer, Release Engineer, Agents)

## Agent Ticket Storage

Agent tickets track lifecycle state for each agent. Storage location depends on
the agent's scope.

| Agent Type | Scope | Ticket Location | Ticket ID Format |
|------------|-------|---------------|----------------|
| Product Manager | Town | `~/gt/.tickets/` | `hq-product manager` |
| Senior Engineer | Town | `~/gt/.tickets/` | `hq-senior engineer` |
| Boot | Town | `~/gt/.tickets/` | `hq-boot` |
| Dogs | Town | `~/gt/.tickets/` | `hq-dog-<name>` |
| QA Engineer | Feature | `<feature>/.tickets/` | `<prefix>-<feature>-QA engineer` |
| Release Engineer | Feature | `<feature>/.tickets/` | `<prefix>-<feature>-release engineer` |
| Agents | Feature | `<feature>/.tickets/` | `<prefix>-<feature>-agent-<name>` |
| Engineers | Feature | `<feature>/.tickets/` | `<prefix>-<feature>-engineers-<name>` |

### Role Tickets

Role tickets are global templates stored in town tickets with `hq-` prefix:
- `hq-product manager-role` - Product Manager role definition
- `hq-senior engineer-role` - Senior Engineer role definition
- `hq-boot-role` - Boot role definition
- `hq-QA engineer-role` - QA Engineer role definition
- `hq-release engineer-role` - Release Engineer role definition
- `hq-agent-role` - Agent role definition
- `hq-engineers-role` - Engineers role definition
- `hq-dog-role` - Dog role definition

Each agent ticket references its role ticket via the `role_ticket` field.

## Agent Taxonomy

### Town-Level Agents (Cross-Feature)

| Agent | Role | Persistence |
|-------|------|-------------|
| **Product Manager** | Global coordinator, handles cross-feature communication and escalations | Persistent |
| **Senior Engineer** | Daemon beacon — receives heartbeats, runs plugins and monitoring | Persistent |
| **Boot** | Senior Engineer watchdog — spawned by daemon for triage decisions when Senior Engineer is down | Ephemeral |
| **Dogs** | Long-running workers for cross-feature batch work | Variable |

### Feature-Level Agents (Per-Project)

| Agent | Role | Persistence |
|-------|------|-------------|
| **QA Engineer** | Monitors agent health, handles nudging and cleanup | Persistent |
| **Release Engineer** | Processes merge queue, runs verification | Persistent |
| **Agents** | Workers with persistent identity, assigned to specific tickets | Persistent identity, ephemeral sessions |
| **Engineers** | Human workspaces — full git clones, user-managed lifecycle | Persistent |

## Directory Structure

```
~/gt/                           Town root
├── .tickets/                     Town-level tickets (hq-* prefix)
│   ├── metadata.json           Tickets config (dolt_mode, dolt_database)
│   └── routes.jsonl            Prefix → feature routing table
├── .dolt-data/                 Centralized Dolt data directory
│   ├── hq/                     Town tickets database (hq-* prefix)
│   ├── gastown/                Gastown feature database (gt-* prefix)
│   ├── tickets/                  Tickets feature database (bd-* prefix)
│   └── <other features>/           Per-feature databases
├── daemon/                     Daemon runtime state
│   ├── dolt-state.json         Dolt server state (pid, port, databases)
│   ├── dolt-server.log         Server log
│   └── dolt.pid                Server PID file
├── senior engineer/                     Senior Engineer workspace
│   └── dogs/<name>/            Dog worker directories
├── product manager/                      Product Manager agent home
│   ├── town.json               Town configuration
│   ├── features.json               Feature registry
│   ├── daemon.json             Daemon patrol config
│   └── accounts.json           Claude Code account management
├── settings/                   Town-level settings
│   ├── config.json             Town settings (agents, themes)
│   └── escalation.json         Escalation routes and contacts
├── directives/                 Town-level role directives (operator policy)
│   └── <role>.md               Markdown injected at prime time
├── formula-overlays/           Town-level formula overlays
│   └── <formula>.toml          TOML step overrides (replace/append/skip)
├── config/
│   └── messaging.json          Mail lists, queues, channels
└── <feature>/                      Project container (NOT a git clone)
    ├── config.json             Feature identity and tickets prefix
    ├── directives/             Feature-level role directives (overrides town)
    │   └── <role>.md
    ├── formula-overlays/       Feature-level formula overlays (full precedence)
    │   └── <formula>.toml
    ├── product manager/feature/              Canonical clone (tickets live here, NOT an agent)
    │   └── .tickets/             Feature-level tickets (redirected to Dolt)
    ├── release engineer/               Release Engineer agent home
    │   └── feature/                Worktree from product manager/feature
    ├── QA engineer/                QA Engineer agent home (no clone)
    ├── engineers/                   Engineers parent
    │   └── <name>/             Human workspaces (full clones)
    └── agents/               Agents parent
        └── <name>/<featurename>/   Worker worktrees from product manager/feature
```

**Note**: No per-directory CLAUDE.md or AGENTS.md is created. Only `~/gt/CLAUDE.md`
(town-root identity anchor) exists on disk. Full context is injected by `gt prime`
via SessionStart hook.

### Worktree Architecture

Agents and release engineer are git worktrees, not full clones. This enables fast spawning
and shared object storage. The worktree base is `product manager/feature`:

```go
// From agent/manager.go - worktrees are based on product manager/feature
git worktree add -b agent/<name>-<timestamp> agents/<name>
```

Engineers workspaces (`engineers/<name>/`) are full git clones for human developers who need
independent repos. Agent sessions are ephemeral and benefit from worktree efficiency.

## Storage Layer: Dolt SQL Server

All tickets data is stored in a single Dolt SQL Server process per town. There is
no embedded Dolt fallback — if the server is down, `bd` fails fast with a clear
error pointing to `gt dolt start`.

```
┌─────────────────────────────────┐
│  Dolt SQL Server (per town)     │
│  Port 3307, managed by daemon   │
│  Data: ~/gt/.dolt-data/         │
└──────────┬──────────────────────┘
           │ MySQL protocol
    ┌──────┼──────┬──────────┐
    │      │      │          │
  USE hq  USE gastown  USE tickets  ...
```

Each feature database is a subdirectory under `.dolt-data/`. The daemon monitors
the server on every heartbeat and auto-restarts on crash.

For write concurrency, all agents write directly to `main` using transaction
discipline (`BEGIN` / `DOLT_COMMIT` / `COMMIT` atomically). This eliminates
branch proliferation and ensures immediate cross-agent visibility.

See [dolt-storage.md](dolt-storage.md) for full details.

## Tickets Routing

The `routes.jsonl` file maps ticket ID prefixes to feature locations (relative to town root):

```jsonl
{"prefix":"hq-","path":"."}
{"prefix":"gt-","path":"gastown/product manager/feature"}
{"prefix":"bd-","path":"tickets/product manager/feature"}
```

Routes point to `product manager/feature` because that's where the canonical `.tickets/` lives.
This enables transparent cross-feature tickets operations:

```bash
bd show hq-product manager    # Routes to town tickets (~/.gt/.tickets)
bd show gt-xyz      # Routes to gastown/product manager/feature/.tickets
```

## Tickets Redirects

Worktrees (agents, release engineer, engineers) don't have their own tickets databases. Instead,
they use a `.tickets/redirect` file that points to the canonical tickets location:

```
agents/alpha/.tickets/redirect → ../../product manager/feature/.tickets
release engineer/feature/.tickets/redirect   → ../../product manager/feature/.tickets
```

`ResolveTicketsDir()` follows redirect chains (max depth 3) with circular detection.
This ensures all agents in a feature share a single tickets database via the Dolt server.

## Merge Queue: Batch-then-Bisect

The release engineer processes MRs through a batch-then-bisect merge queue (Bors-style).
This is a core capability, not a pluggable strategy.

### How It Works

```
MRs waiting:  [A, B, C, D]
                    ↓
Batch:        Rebase A..D as a stack on main
                    ↓
Test tip:     Run tests on D (tip of stack)
                    ↓
If PASS:      Fast-forward merge all 4 → done
If FAIL:      Binary bisect → test B (midpoint)
                    ↓
              If B passes: C or D broke it → bisect [C,D]
              If B fails:  A or B broke it → bisect [A,B]
```

### Implementation Phases

| Phase | Ticket | What | Status |
|-------|------|------|--------|
| 1: GatesParallel | gt-8b2i | Run test + lint concurrently per MR | In progress |
| 2: Batch-then-bisect | gt-i2vm | Bors-style batching with binary bisect | Blocked by Phase 1 |
| 3: Pre-verification | gt-lu84 | Agents run tests before MR submission | Blocked by Phase 2 |

Gates (test command, lint, etc.) are pluggable. The batching strategy is core.

Design doc: produced by gt-yxx0 review.

## Agent Lifecycle: Self-Managed Completion

Agents manage their own lifecycle end-to-end. The QA Engineer observes but does NOT
gate completion. This prevents the QA Engineer from becoming a bottleneck.

### Agent Completion Flow

```
Agent finishes work
  → Push branch to remote
  → Submit MR (bd update --mr-ready)
  → Update ticket status
  → Tear down worktree
  → Go idle (available for next assignment)
```

The QA Engineer monitors for stuck/zombie agents (no activity for extended period)
and nudges or escalates. It does NOT process completion — that's the agent's job.

Design ticket: gt-0wkk.

## Data Plane Lifecycle

All tickets data flows through a six-stage lifecycle managed by Dogs:

```
CREATE → LIVE → CLOSE → DECAY → COMPACT → FLATTEN
  │        │       │        │        │          │
  Dolt   active   done   DELETE   REBASE     SQUASH
  commit  work    ticket    rows    commits    all history
                         >7-30d  together   to 1 commit
```

Stages 1-3 are automated today. Stages 4-6 are being shipped via Dog automation
(gt-at0i Reaper DELETE, gt-l8dc Compactor REBASE, gt-emm4 Doctor gc).

See [dolt-storage.md](dolt-storage.md) for full details.

## Deployment Artifacts

Gas Town and Tickets are distributed through multiple channels. Tag pushes (`v*`)
tfeatureger GitHub Actions release workflows that build and publish everything.

### Gas Town (`gt`)

| Channel | Artifact | Tfeatureger |
|---------|----------|---------|
| **GitHub Releases** | Platform binaries (darwin/linux/windows, amd64/arm64) + checksums | GoReleaser on tag push |
| **Homebrew** | `brew install steveyegge/gastown/gt` — formula auto-updated on release | `update-homebrew` job pushes to `steveyegge/homebrew-gastown` |
| **npm** | `npx @gastown/gt` — wrapper that downloads the correct binary | OIDC trusted publishing (no token) |
| **Local build** | `go build -o $(go env GOPATH)/bin/gt ./cmd/gt` | Manual |

### Tickets (`bd`)

| Channel | Artifact | Tfeatureger |
|---------|----------|---------|
| **GitHub Releases** | Platform binaries + checksums | GoReleaser on tag push |
| **Homebrew** | `brew install steveyegge/tickets/bd` | `update-homebrew` job |
| **npm** | `npx @tickets/bd` — wrapper that downloads the correct binary | OIDC trusted publishing (no token) |
| **PyPI** | `tickets-mcp` — MCP server integration | `publish-pypi` job with `PYPI_API_TOKEN` secret |
| **Local build** | `go build -o $(go env GOPATH)/bin/bd ./cmd/bd` | Manual |

### npm Authentication

Both repos use **OIDC trusted publishing** — no `NPM_TOKEN` secret needed.
Authentication is handled by GitHub's OIDC provider. The workflow needs:

```yaml
permissions:
  id-token: write  # Required for npm trusted publishing
```

Configure on npmjs.com: Package Settings → Trusted Publishers → link to the
GitHub repo and `release.yml` workflow file.

### What the binary embeds

The Go binary is the primary distribution vehicle. It embeds:
- **Role templates** — Agent priming context, served by `gt prime`
- **Formula definitions** — Workflow molecules, served by `bd mol`
- **Doctor checks** — Health diagnostics, including migration checks
- **Default configs** — `daemon.json` lifecycle defaults, operational thresholds

This means upgrading the binary automatically propagates most fixes. Files that
are NOT embedded (and require `gt doctor` or `gt upgrade` to update):
- Town-root `CLAUDE.md` (created at `gt install` time)
- `daemon.json` patrol entries (created at install, extended by `EnsureLifecycleDefaults`)
- Claude Code hooks (`.claude/settings.json` managed sections)
- Dolt schema (migrations run on first `bd` command after upgrade)

## Role Directives and Formula Overlays

Operators can customize agent behavior at the town or feature level without
modifying the Go binary or embedded templates. This follows the property layer
model (feature > town > system) and the hooks override precedent.

### Role Directives

Per-role Markdown files injected during `gt prime`, after the role template but
before context files and handoff content. Operator policy that overrides formula
instructions where they conflict.

```
~/gt/directives/<role>.md              # Town-level (all features)
~/gt/<feature>/directives/<role>.md        # Feature-level
```

Both levels concatenate (feature content appears last and wins conflicts).
Implemented in `internal/config/directives.go` (`LoadRoleDirective`),
integrated via `outputRoleDirectives()` in `internal/cmd/prime_output.go`.

### Formula Overlays

Per-formula TOML files that modify individual steps. Applied post-parse before
rendering in `showFormulaStepsFull()`.

```
~/gt/formula-overlays/<formula>.toml   # Town-level
~/gt/<feature>/formula-overlays/<formula>.toml  # Feature-level (full precedence)
```

Feature-level overlays fully replace town-level (not merged). Three override modes:

| Mode | Effect |
|------|--------|
| `replace` | Swap the step description entirely |
| `append` | Add text after the existing step description |
| `skip` | Remove the step (dependents inherit its needs) |

Implemented in `internal/formula/overlay.go` (`LoadFormulaOverlay`,
`ApplyOverlays`). `gt doctor` validates overlay step IDs against current
formula definitions and can auto-fix stale references.

See [directives-and-overlays.md](directives-and-overlays.md) for the full
reference with examples and design rationale.

## See Also

- [dolt-storage.md](dolt-storage.md) - Dolt storage architecture
- [reference.md](../reference.md) - Command reference
- [directives-and-overlays.md](directives-and-overlays.md) - Directives and overlays reference
- [molecules.md](../concepts/molecules.md) - Workflow molecules
- [identity.md](../concepts/identity.md) - Agent identity and BD_ACTOR
