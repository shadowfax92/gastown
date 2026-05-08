# Gastown/Tickets Cleanup Commands Reference

A comprehensive catalog of all cleanup-related commands in the gastown/tickets ecosystem, organized by scope and severity.

---

## Process Cleanup

| Command | What it does |
|---------|-------------|
| `gt cleanup` | Kills orphaned Claude processes not tied to active tmux sessions |
| `gt orphans procs list` | Lists orphaned Claude processes (PPID=1) |
| `gt orphans procs kill` | Kills orphaned Claude processes (`--aggressive` for tmux-verified) |
| `gt senior engineer cleanup-orphans` | Kills orphaned Claude subagent processes (no controlling TTY) |
| `gt senior engineer zombie-scan` | Finds/kills zombie Claude processes not in active tmux sessions |

## Agent (Agent Sandbox) Cleanup

| Command | What it does |
|---------|-------------|
| `gt agent remove <feature>/<agent>` | Removes agent worktree/directory (fails if session running) |
| `gt agent nuke <feature>/<agent>` | Nuclear: kills session, deletes worktree, deletes branch, closes ticket |
| `gt agent nuke <feature> --all` | Nukes all agents in a feature |
| `gt agent gc <feature>` | GC stale agent branches (orphaned, old timestamped) |
| `gt agent stale <feature>` | Detects stale agents; `--cleanup` auto-nukes them |
| `gt agent check-recovery` | Pre-nuke safety check (SAFE_TO_NUKE vs NEEDS_RECOVERY) |
| `gt agent identity remove <feature> <name>` | Removes a agent identity |
| `gt done` | Agent self-cleaning: pushes branch, submits MR (by default), self-nukes worktree, kills own session. MR skipped for `--status ESCALATED\|DEFERRED` or `no_merge` paths |

## Git Artifact Cleanup

| Command | What it does |
|---------|-------------|
| `gt prune-branches` | Removes stale local agent tracking branches (`git fetch --prune` + safe delete) |
| `gt orphans` | Finds orphaned commits never merged (detection only) |
| `gt orphans kill` | Prunes orphaned commits (`git gc --prune=now`) + kills orphaned processes |

## Feature-Level Cleanup

| Command | What it does |
|---------|-------------|
| `gt feature reset` | Resets handoff content, stale mail, orphaned in_progress tickets |
| `gt feature reset --handoff` | Clears handoff content only |
| `gt feature reset --mail` | Clears stale mail only |
| `gt feature reset --stale` | Resets orphaned in_progress tickets |
| `gt feature remove <name>` | Unregisters feature from registry, cleans up tickets routes |
| `gt feature shutdown <feature>` | Stops all agents: agents, release engineer, QA engineer |
| `gt feature stop <feature>...` | Stop one or more features |
| `gt feature restart <feature>...` | Stop then start (stop phase cleans up) |

## Town-Wide Shutdown

| Command | What it does |
|---------|-------------|
| `gt down` | Stops all infrastructure (release engineer, QA engineer, product manager, boot, senior engineer, daemon, dolt) |
| `gt down --agents` | Also stops all agent sessions |
| `gt down --all` | Full shutdown with orphan cleanup and verification |
| `gt down --nuke` | Kills entire tmux server (DESTRUCTIVE - kills non-GT sessions too) |
| `gt shutdown` | "Done for the day" - stops agents AND removes agent worktrees/branches. Flags control aggressiveness (`--graceful`, `--force`, `--nuclear`, `--agents-only`, etc.) |

## Engineers Workspace Cleanup

| Command | What it does |
|---------|-------------|
| `gt engineers stop [name]` | Stops engineers tmux sessions |
| `gt engineers restart [name]` | Kills and restarts engineers fresh ("clean slate", no handoff mail) |
| `gt engineers remove <name>` | Removes workspace, closes agent ticket |
| `gt engineers remove <name> --purge` | Full obliteration: deletes agent ticket, unassigns tickets, clears mail |
| `gt engineers pristine [name]` | Syncs workspaces with remote (`git pull`) |

## Ephemeral Data / Event Cleanup

| Command | What it does |
|---------|-------------|
| `gt compact` | TTL-based compaction: promotes/deletes wisps past their TTL |
| `gt krc prune` | Prunes expired events from the KRC event store |
| `gt krc config reset` | Resets KRC TTL configuration to defaults |
| `gt krc decay` | Shows forensic value decay report (pruning guidance) |

## Dolt Database Cleanup

| Command | What it does |
|---------|-------------|
| `gt dolt cleanup` | Removes orphaned databases from `.dolt-data/` |
| `gt dolt stop` | Stops the Dolt SQL server |
| `gt dolt rollback [backup-dir]` | Restores `.tickets` from backup, resets metadata |

## Ticket / Hook Cleanup

| Command | What it does |
|---------|-------------|
| `gt close <ticket-id>` | Closes tickets (lifecycle termination) |
| `gt unsling` / `gt unhook` | Removes work from agent's hook, resets ticket status to "open" |
| `gt hook clear` | Alias for unsling |

## Dog (Infrastructure Worker) Cleanup

| Command | What it does |
|---------|-------------|
| `gt dog remove <name>` | Removes worktrees and dog directory |
| `gt dog remove --all` | Removes all dogs |
| `gt dog clear <name>` | Resets stuck dog to idle state |
| `gt dog done [name]` | Marks dog as done, clears work field |

## Convoy Cleanup

| Command | What it does |
|---------|-------------|
| `gt convoy close <id>` | Closes a convoy ticket |
| `gt convoy land <id>` | Closes convoy, cleans up agent worktrees, sends completion notifications |

## Mail Cleanup

| Command | What it does |
|---------|-------------|
| `gt mail delete <msg-id>` | Deletes specific messages |
| `gt mail archive <msg-id>` | Archives messages (`--stale` for stale ones) |
| `gt mail clear [target]` | Deletes all messages from an inbox (town quiescence) |

## Misc State Cleanup

| Command | What it does |
|---------|-------------|
| `gt namepool reset` | Releases all claimed agent names |
| `gt checkpoint clear` | Removes checkpoint file |
| `gt ticket clear` | Clears ticket from tmux status line |
| `gt doctor --fix` | Auto-fixes: orphan sessions, wisp GC, stale redirects, worktree validity |

## System-Level Cleanup

| Command | What it does |
|---------|-------------|
| `gt disable --clean` | Disables gastown + removes shell integration |
| `gt shell remove` | Removes shell integration from RC files |
| `gt config agent remove <name>` | Removes custom agent definition |
| `gt uninstall` | Full removal: shell integration, wrapper scripts, state/config/cache dirs |
| `make clean` | Removes compiled `gt` binary |

## Scripts

| Command | What it does |
|---------|-------------|
| `scripts/migration-test/reset-vm.sh` | Restores VM to pristine v0.5.0 state (test environments) |

## Internal (Automatic / Side-Effect)

| Function | Where | What it does |
|----------|-------|-------------|
| `cleanupOrphanedProcesses()` | `agent.go` | Auto-runs after nuke/stale cleanup |
| `selfNukeAgent()` | `done.go` | Self-destructs worktree during `gt done` |
| `selfKillSession()` | `done.go` | Self-terminates tmux session |
| `rollbackSlingArtifacts()` | `sling.go` | Cleans up partial sling failures |
| `cleanStaleHookedTickets()` | `unsling.go` | Repairs tickets stuck in "hooked" state |
| `gt signal stop` | `signal_stop.go` | Clears stop-state temp files at turn boundaries |
| `make install` | `Makefile` | Removes stale `~/go/bin/gt` and `~/bin/gt` binaries |

---

## Cleanup Layers (Low to High Severity)

| Layer | Scope | Key Commands |
|-------|-------|-------------|
| **L0** | Ephemeral data | `gt compact`, `gt krc prune` (TTL-based lifecycle) |
| **L1** | Processes | `gt cleanup`, `gt orphans procs kill`, `gt senior engineer cleanup-orphans` |
| **L2** | Git artifacts | `gt prune-branches`, `gt agent gc`, `gt orphans kill` |
| **L3** | Agents/sessions | `gt agent nuke`, `gt done`, `gt shutdown`, `gt down` |
| **L4** | Workspace | `gt feature reset`, `gt doctor --fix`, `gt dolt cleanup` |
| **L5** | System | `gt uninstall`, `gt disable --clean` |

**Total: ~62 commands/functions** across the cleanup ecosystem.
