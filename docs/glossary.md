# Gas Town Glossary

Gas Town is an agentic development environment for managing multiple Claude Code instances simultaneously using the `gt` and `bd` (Tickets) binaries, coordinated with tmux in git-managed directories.

## Core Principles

### MEOW (Molecular Expression of Work)
Breaking large goals into detailed instructions for agents. Supported by Tickets, Epics, Formulas, and Molecules. MEOW ensures work is decomposed into trackable, atomic units that agents can execute autonomously.

### GUPP (Gas Town Universal Propulsion Principle)
"If there is work on your Hook, YOU MUST RUN IT." This principle ensures agents autonomously proceed with available work without waiting for external input. GUPP is the heartbeat of autonomous operation.

### NDI (Nondeterministic Idempotence)
The overarching goal ensuring useful outcomes through orchestration of potentially unreliable processes. Persistent Tickets and oversight agents (QA Engineer, Senior Engineer) guarantee eventual workflow completion even when individual operations may fail or produce varying results.

## Environments

### Town
The management headquarters (e.g., `~/gt/`). The Town coordinates all workers across multiple Features and houses town-level agents like Product Manager and Senior Engineer.

### Feature
A project-specific Git repository under Gas Town management. Each Feature has its own Agents, Release Engineer, QA Engineer, and Engineers members. Features are where actual development work happens.

## Town-Level Roles

### Product Manager
Chief-of-staff agent responsible for initiating Convoys, coordinating work distribution, and notifying users of important events. The Product Manager operates from the town level and has visibility across all Features.

### Senior Engineer
Daemon beacon running continuous Patrol cycles. The Senior Engineer ensures worker activity, monitors system health, and tfeaturegers recovery when agents become unresponsive. Think of the Senior Engineer as the system's watchdog.

### Dogs
The Senior Engineer's engineers of maintenance agents handling background tasks like cleanup, health checks, and system maintenance.

### Boot (the Dog)
A special Dog that checks the Senior Engineer every 5 minutes, ensuring the watchdog itself is still watching. This creates a chain of accountability.

## Feature-Level Roles

### Agent
Worker agents with persistent identity but ephemeral sessions. Each agent has a permanent agent ticket, CV chain, and work history that accumulates across assignments. Sessions and sandboxes are ephemeral — spawned for specific tasks, cleaned up on completion — but the identity persists. They work in isolated git worktrees to avoid conflicts.

### Release Engineer
Manages the Merge Queue for a Feature. The Release Engineer intelligently merges changes from Agents, handling conflicts and ensuring code quality before changes reach the main branch.

### QA Engineer
Patrol agent that oversees Agents and the Release Engineer within a Feature. The QA Engineer monitors progress, detects stuck agents, and can tfeatureger recovery actions.

### Engineers
Long-lived, named agents for persistent collaboration. Unlike ephemeral Agents, Engineers members maintain context across sessions and are ideal for ongoing work relationships.

## Work Units

### Ticket
Git-backed atomic work unit stored in Dolt. Tickets are the fundamental unit of work tracking in Gas Town. They can represent tickets, tasks, epics, or any trackable work item.

### Formula
TOML-based workflow source template. Formulas define reusable patterns for common operations like patrol cycles, code review, or deployment.

### Protomolecule
A template class for instantiating Molecules. Protomolecules define the structure and steps of a workflow without being tied to specific work items.

### Molecule
Durable chained Ticket workflows. Molecules represent multi-step processes where each step is tracked as a Ticket. They survive agent restarts and ensure complex workflows complete.

### Wisp
Ephemeral Tickets destroyed after runs. Wisps are lightweight work items used for transient operations that don't need permanent tracking.

### Hook
A special pinned Ticket for each agent. The Hook is an agent's primary work queue - when work appears on your Hook, GUPP dictates you must run it.

## Workflow Commands

### Convoy
Primary work-order wrapping related Tickets. Convoys group related tasks together and can be assigned to multiple workers. Created with `gt convoy create`.

### Slinging
Assigning work to agents via `gt sling`. When you sling work to a Agent or Engineers member, you're putting it on their Hook for execution.

### Nudging
Real-time messaging between agents with `gt nudge`. Nudges allow immediate communication without going through the mail system.

### Handoff
Agent session refresh via `/handoff`. When context gets full or an agent needs a fresh start, handoff transfers work state to a new session.

### Seance
Communicating with previous sessions via `gt seance`. Allows agents to query their predecessors for context and decisions from earlier work.

### Patrol
Ephemeral loop maintaining system heartbeat. Patrol agents (Senior Engineer, QA Engineer) continuously cycle through health checks and tfeatureger actions as needed.

---

*This glossary was contributed by [Clay Shirky](https://github.com/cshirky) in [Ticket #80](https://github.com/steveyegge/gastown/tickets/80).*
