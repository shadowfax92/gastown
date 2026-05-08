# Changelog

All notable changes to the Gas Town project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2026-05-06

### Added

- **Convoy completion + cross-feature dep notifications** — Convoy completion and
  cross-feature dependency resolution now fire notifications, surfacing milestone
  events without polling (#3838, gt-wfs-55hsg).

### Fixed

- **Scheduler guards** — `scheduleTicket` now skips closed and tombstone tickets,
  preventing wasted dispatch attempts and "no such ticket" errors (hq-ki2,
  #3840). Mirrors the existing `runSling` / `executeSling` guards.
- **Mail inbox Dolt churn** — Reduced repeated Dolt commits on inbox reads
  (gt-05ld, #3830).
- **Polish** — Rename `DeliveryAckLabelSequenceIdempotent` to clarify intent
  (gt-ekuh).
- **`jsonl_git_backup` plugin** — Forward `USER`, `LOGNAME`, and `HOME` env to
  git children so the plugin can run under uid-501 wrapper contexts (gt-zt1w).
- **Resilience under Dolt memory pressure** — Subprocess timeout and parallel
  feature scan so `gt status` and related commands degrade gracefully when Dolt is
  under load instead of hanging.
- **`bd` 1.0+ compatibility** — `bd init --prefix <X>` now persists the prefix
  directly (previously relied on the removed `bd config set ticket_prefix` call).
- **Cross-feature mail delivery** — Delivery acks routed via bd prefix routing;
  Create routes via BEADS_DIR instead of `--repo` to prevent pthread deadlock;
  zero-copy fix for routed writes.
- **Daemon hardening** — Filter messaging tickets, guard cross-feature prefix, fail
  prime fast on bad state. `hasAssignedOpenWork` checked via `--repo` (gt-fcw)
  and before reaping working-ticket-lookup-failed agents. Legacy flat agent
  layout supported with regression coverage.
- **Agent / sling lifecycle** — Auto-burn orphan molecules from tickets stuck
  in hooked-with-no-assignee (gh-3697); use `IsAgentAlive` in `Start` zombie
  check (hq-k1ot); accept dog pool targets in deferred mode (aa-4yf2);
  CLOSED ticket treated as terminal in `check-recovery` (aa-55d8); zombie-scan
  skips restart if agent branch already merged (aa-apw); persist convoy
  fields for single-ticket dispatch.
- **Convoy** — `bdListChildren` falls back to deps table when the primary path
  is unavailable (#3700); cross-feature wave staging regression test (hq-mtc).
- **Doctor** — Role-aware Stop hook check for agents (#3648); detect and
  recover corrupt `.repo.git` bare repos (gt-61twf).
- **Install / dolt** — Fail fast when Dolt is unavailable during install;
  clamp Dolt idle session `wait_timeout` to prevent connection exhaustion
  (gh-3623); switch remaining DDL call sites to server connection (#3641);
  atomic `settings.json` write to prevent concurrent-spawn corruption (gh-3500).
- **Hooks / agent config** — Pass agent `Args` to `ResolveProcessNames` at all
  call sites; resolve agent process names through wrapper commands; preserve
  built-in opencode preset fields in overrides; sync opencode hooks in nested
  agent worktrees (gt-hii); reject non-ticket args before `bd show` (#3701).
- **Done / safety nets** — Auto-pop orphaned stashes in gt-pvx safety net;
  auto-save uncommitted implementation work (gt-pvx safety net); honor
  explicit target in done contamination check (gt-nmt); verify pushed commits
  before ticket closure.
- **Mail / archive** — Tolerate GC'd ticket IDs in archive and `archive --stale`
  (aa-6hv); clear satisfied mail reply reminders (gt-niu2); isolate
  `NotifyMergeOutcome` tests from production mail.
- **Plugin runner** — Explicit `gate=manual` skip in `dispatchPlugins`
  (hq-suin); dispatcher skips idle dogs with leaked tmux sessions (gt-o24);
  `stuck-agent-dog` uses `gt hook show` to inspect other agents' hooks.
- **Test infrastructure** — Don't skip non-Dolt daemon tests when Docker is
  unavailable (gt-kw4449); enforce standalone formula singleton at sling
  boundary (gt-3kir); `gt prime` renders formula steps for both town- and
  feature-level formulas; nightly integration test failures fixed for wl-commons
  (closes antns1/fergus#336).
- **Lint** — Correct British spellings to American English; `gt status` no
  longer hangs in `bd` probe.
- **Plugin portability** — `dolt-archive` and `dolt-backup` no longer rely on
  the bash 4 `mapfile` builtin, restoring production DB discovery on macOS
  systems where `/bin/bash` (3.2.57) is invoked.

## [1.0.1] - 2026-04-25

### Fixed

- **`gt dog done` closes accumulated plugin mails** — Plugin dispatch mails sent
  by the daemon to dogs were never closed after execution, causing dog inboxes
  to accumulate hundreds of open "Plugin: X" tickets. On every UserPromptSubmit
  hook, `gt mail check --inject` re-injected ALL open mails, ballooning agent
  context to 60-70% and causing compaction/hook conflicts that froze the senior engineer.
  `gt dog done` now archives all open "Plugin: " mails from the dog's inbox
  before clearing work and terminating the session.

## [1.0.0] - 2026-04-02

### Added

- **Windows platform support** — Cherry-picked Windows support: platform-specific
  signal handling, process management, tmux descendant tracking, and estop split
  into OS-specific files.
- **Workflow formula type** — `gt formula run` now supports `type = "workflow"`
  formulas with interactive step execution.
- **Release Engineer PR merge strategy** — New `merge_strategy=pr` option uses `gh pr merge`
  instead of direct push, enabling GitHub's native merge queue.
- **`/engineers-commit` skill** — Canonical engineers commit workflow as a Claude skill.
- **Rate-limit watchdog plugin** — Auto-estop on API 429 rate limit errors.
- **`gt mail send --from` flag** — Relay/bridge use case for mail forwarding.
- **`gt mail mark-read --all`** — Mark all inbox messages as read at once.
- **`RequireTownEnv` test helper** — Integration test guard with documentation
  for GH#2717.
- **Prefix collision checking** — `gt feature add` and `gt feature adopt` now detect
  prefix collisions before creation.
- **Ticket description in PR body** — PR body now includes ticket description and
  diff stat for richer context.
- **Dolt commit freshness health check** — Dolt health metrics now include commit
  freshness monitoring.
- **Default effort level config** — `CLAUDE_CODE_EFFORT_LEVEL` configurable for
  all agents.
- **`gt dolt pull`** — New command for pulling Dolt remotes.

### Changed

- **CI: Windows smoke tests** — Replaced Windows unit tests with lighter smoke
  tests for faster CI.
- **Release Engineer requires review** — `require_review=true` now blocks merge until PR
  has an approving review.
- **Product Manager approval for scope expansion** — Agents must get product manager approval
  before expanding molecule scope.
- **Agent PreToolUse guard** — Blocks `sudo` and system package installs in
  agent sessions.
- **Patrol formulas use feature-prefixed vars** — Template variables for agent ticket
  IDs are now feature-prefixed.
- **Agent auto-checkout** — Sessions auto-checkout a fresh branch when started
  on the default branch.
- **Makefile OOM fixes** — Strip flags and codesign removal to prevent OOM kills
  during builds.

### Fixed

- **SQL injection in dolt_remotes** — Escaped SQL string in remote name query
  (security fix).
- **ACP integration test flakiness** — Resolved CleanExit and FullLoop test
  races.
- **QA Engineer zombie detection** — Distinguish ticket lookup failure from
  closed/reaped tickets.
- **Scheduler capacity counting** — Exclude idle agents from capacity count.
- **Nested town root detection** — `FindTownRoot` now returns outermost town
  root for nested features.
- **Convoy +Inf metadata** — Fix detection and flip-flop in convoy metadata.
- **Carry branch builds** — Support `carry/*` branches in build infrastructure.
- **Unsigned binary handling** — Refuse to run unsigned binary instead of just
  warning.
- **Shell hook shebang** — `shell-hook.sh` shebang now matches registered shell.
- **Sling context routing** — Route sling-context wisp to target feature instead of
  HQ.
- **Feed timestamps** — Display feed timestamps in local timezone instead of UTC.
- **Engineers status across features** — `gt engineers status` now shows all features.
- **Agent CLAUDE.md commit guard** — Prevent agents from committing Gas Town
  overlay CLAUDE.md.
- **PR branch deletion guard** — Guard PR branch deletion and add review approval
  check.
- **Lint fixes** — Resolve unconvert, unparam, and misspell warnings.
- **Git identity in worktrees** — Propagate global git identity into agent
  worktrees.
- **Sparse-checkout deletions** — Ignore sparse-checkout deletions in git status.
- **Tickets config parsing** — Ignore `(not set)` in tickets config output.
- **Plugin heartbeat path** — Check `heartbeat.json` instead of legacy
  `.senior engineer-heartbeat`.
- **Dog mail race condition** — Send dog mail before session start to prevent
  race.
- **Agent dashboard drops** — Use local prefix registry in dashboard
  FetchWorkers.
- **Dolt TCP ping fallback** — Always TCP-ping dolt port as last resort in
  IsRunning.
- **ENABLED_RIGS unbound variable** — Initialize array to avoid error with
  `set -u`.
- **Claude project dir path encoding** — Encode underscores as hyphens in
  project directory path.
- **Hook template PATH export** — Replace `export PATH` with `{{GT_BIN}}` in all
  hook templates.

## [0.13.0] - 2026-03-29

### Added

- **Directives and overlays** — New `gt prime` directive loader and overlay
  system with CLI commands (`gt directive`, `gt overlay`), formula overlay
  support, and doctor health check for overlay integrity.
- **Gate ticket instruction template** — Gate tickets now carry structured
  instruction templates with GitHub API client support.
- **Merge queue step dependencies** — `gt mq submit` enforces molecule step
  dependency ordering before submission.
- **Convoy watch/unwatch** — `gt convoy watch` and `gt convoy unwatch` for
  opt-in completion notifications on convoy progress.
- **Convoy merge queue panel** — Feed view now shows merge queue status in
  convoy panels.
- **Patrol scan CLI** — `gt patrol scan` detects zombie and stalled agents
  from the command line.
- **Checkpoint dog** — New `checkpoint_dog` auto-commits WIP changes in
  agent worktrees periodically.
- **Crash recovery on startup** — `gt up` detects and recovers orphaned hooked
  tickets left by crashed sessions.
- **Post-squash gate phase** — Release Engineer adds a pre-push validation gate after
  squash merging.
- **Release Engineer auto\_push config** — New `auto_push` feature setting controls whether
  release engineer pushes after merge.
- **PR feedback patrol formula** — `mol-pr-feedback-patrol` formula for
  automated PR review triage.
- **Configurable tmux theming** — Window tint and `window-style` theming with
  resolver; Product Manager gets terminal-default theme.
- **`gt changelog` command** — Generate changelogs from the CLI with tests.
- **Wasteland stamps and pilot cohorts** — `gt wl stamp`, `gt wl stamps`
  commands and `pilot_cohort` column for HOP pilot program.
- **Wasteland scorekeeper** — Charsheet, scorekeeper, and stamp loop
  integration tests.
- **`gt wl show <work-id>`** — Structured work-item detail view with
  auto-fetch.
- **`gt default-agent list`** — New subcommand to list available agent presets.
- **Disabled patrols setting** — `disabled_patrols` town config to suppress
  patrols without editing daemon.json.
- **Dolt failover/failback** — Multi-host Dolt setups can failover and
  failback between primary and replica.
- **`.no-sync` marker files** — Drop a `.no-sync` file in a database directory
  to exclude it from sync.
- **`/done` slash command** — Agents can invoke `/done` with a Stop hook
  safety net for clean lifecycle exit.
- **Sling `--review-only` flag** — Prevent assignee from merging; report back
  only.
- **Copilot agent support** — GitHub Copilot CLI documented and preset updated
  for GA release (Feb 2026).
- **Unique agent namepool** — Agent names are now globally unique across
  features via shared namepool with auto-assigned themes.
- **Handoff restart prompt** — `gt handoff` now prompts the user before
  restarting the session.
- **Patrol effort tuning** — Idle patrol cycles now run at reduced reasoning
  effort; configurable per-formula with `effort_idle` and `effort_active`.
- **Longer patrol backoff** — Max backoff increased from 5m to 15m for idle
  patrols, reducing cost by ~66% for dormant features.
- **Formula/path discoverability** — Reference docs for formulas, tickets CLI,
  and Dolt injected into agent context to eliminate discovery tax.

### Changed

- **Tickets dependency** upgraded to v0.62.0.
- **Compactor-dog threshold** — Default compaction threshold raised from 500 to
  2000 to reduce unnecessary compactions.
- **Dolt startup timeout** — Scales dynamically by database count (5s per DB)
  instead of fixed timeout.
- **Dolt SIGTERM→SIGKILL timeout** — Increased from 5s to 30s for graceful
  shutdown of large databases.
- **Agent CLAUDE.md provisioning** — Lifecycle instructions provisioned on
  all spawn paths including worktree reuse, with `gt done` reminders injected
  at startup and after compaction.
- **Boot and dog cost tiers** — Boot and dog roles now tracked in the cost tier
  system.
- **Plugin database discovery** — Plugins auto-discover databases instead of
  using hardcoded lists; reaper uses `DiscoverDatabases` with proper error
  handling.
- **Dolt `dolt_transaction_commit` disabled** — Prevents read-only commit
  storms on busy servers.

### Fixed

- **Daemon tickets compatibility guard** — `gt daemon run` now fail-fast checks
  workspace tickets schema compatibility before Convoy polling starts, and
  `gt daemon start` surfaces the startup mismatch directly instead of only
  telling operators to inspect logs.
- **Dolt server stability** — Fixed thundering herd in `doltserver.Start()`,
  port-squatter detection and kill on startup, `cmd.Dir` set on all CLI/SQL
  invocations to prevent stray `.doltcfg` directories, and timing race in
  startup sequence.
- **Security hardening** — Ticket ID suffix validation enforced, formula
  variables use ticket IDs instead of user-supplied titles, and
  `--subject`/`--args` sanitized before tmux pane injection.
- **Tmux reliability** — Replaced timing-based Enter delivery with
  verification-based retry, detect and dismiss Claude Code Rewind menu during
  nudge delivery, restored per-town socket isolation, and added flock-based
  cross-process nudge lock to prevent interleaved delivery.
- **Windows fixes** — Atomic counter in `generateStampID` for timer resolution,
  pipe deadlock prevention in `prime_test`, process group test skips, and
  multiple CI test stabilizations.
- **Agent lifecycle** — Skip crash/zombie alerts for done/nuked agents,
  use `IsIdle` instead of `IsAtPrompt` for startup nudge verify, clean dirty
  worktree before reuse, kill session unconditionally when reusing idle
  agents, and wire operational config into startup nudge loop.
- **Release Engineer fixes** — Use commit SHA instead of branch name for MR dedup,
  supersede MR on same-branch re-submission, check `no_merge` flag before
  merging, close task tickets after successful merge, wait for CI in PR mode,
  and filter MR listings by feature to prevent cross-feature contamination.
- **Convoy fixes** — Use Unix epoch instead of zero time for initial event poll,
  stranded scan checks completion status, create legs in target feature tickets,
  and cross-feature dependency routing uses town root.
- **Cross-town safety** — Prevent orphan cleanup from killing agents on other
  towns' sockets, distinguish sibling Gas Town instances from test zombies.
- **Dog and daemon** — Clear agent identity env vars at startup, prevent
  duplicate Product Manager spawns during `gt up`, auto-clear hung dogs and orphan
  sessions, include dogs in startup retry loop, prevent daemon restart during
  `gt down`, and respect global default agent for dog spawns.
- **Doctor improvements** — Avoid slow `filepath.Walk` on Docker bind mounts,
  stale `sql-server.info` detection, hooks-sync check detects stale Gemini
  settings, route misclassified wisp fixes by workdir, and repair relocated
  worktree gitdir paths.
- **Mail and communication** — Drain crashed agent notifications, prefer
  `GT_TOWN_ROOT` env var for town root detection, fall back to explicit agent
  workspaces for mail delivery.
- **Dolt plugins** — `dolt-archive` uses `while-read` loops for bash 3.2
  compatibility (macOS), `dolt-backup` uses `$HOME/gt` as `GT_ROOT` fallback,
  named Docker volume prevents journal corruption on macOS, and `grep -v`
  exit code handled under `pipefail`.
- **Formula and molecule** — Cap backoff before overflow in `await-event` and
  `await-signal`, inject `merge_strategy` from feature settings into formula vars,
  propagate `base_branch` to MR target in `gt done` and `gt mq submit`.
- **Sling** — Serialize concurrent hook writes with per-assignee flock,
  `--dry-run` detects tmux session collision before spawn, guard `sha[:8]`
  slice against short hashes.
- **Config and identity** — Dog sessions inherit env vars from base agent, custom
  agents inherit Session/Tmux from preset, `CLAUDE_CONFIG_DIR` respected in
  `gt costs`, feature prefix pattern refresh when stale, propagate
  `BEADS_DOLT_SERVER_HOST` to subprocesses, and repair PROJECT IDENTITY
  MISMATCH after crash.
- **Guard and compliance** — Block agents from pushing directly to main.
- **Misc** — `formatPeriod` returns "Week of" on Mondays instead of "Today",
  sync `agent_state` between column and description on transitions, validate
  git URL before engineers clone, `--flat` flag on all `bd list --json` calls to
  guarantee JSON output, `gt upgrade` repairs missing identity tickets, and
  `CLAUDE.local.md` added to gitignore patterns.

### Removed

- **Session-hygiene plugin** — Removed entirely after causing repeated engineers
  session kills.
- **`--no-history` flag** — Removed from identity ticket creation in favor of
  proper ephemeral ticket support.
- **Hardcoded database lists** — Reaper and plugin database discovery replaced
  with dynamic `DiscoverDatabases`.
- **Legacy `gt` database** — Removed from reaper fallback list.

## [0.12.1] - 2026-03-15

### Added

- **Agent Client Protocol (ACP)** — New protocol for structured agent
  communication with propulsion tfeatureger detection and output suppression.
- **gt mountain** — Stage, label, and launch epic work in one command.
- **gt assign** — One-shot ticket creation + hook for direct agent assignment.
- **Convoy --from-epic** — `gt convoy create --from-epic` stages epic children
  into convoy waves with automatic validation ticket.
- **Typed memories** — `gt remember --type feedback/project/user/reference` for
  categorized agent memory storage.
- **Repo-sourced feature settings** — `.gastown/settings.json` in repos auto-configures
  feature behavior (test gates, merge strategy).
- **exec-wrapper plugin type** — Plugins can now wrap agent execution.
- **Prior attempt context** — Agents receive context from previous failed
  attempts when re-dispatched.
- **Spider Protocol** — Fraud detection for Wasteland stamp system.

### Changed

- **Reaper plugin receipt cleanup** — Plugin run receipts now fast-tracked for
  closure (1h) instead of waiting for 7-day stale ticket AutoClose.
- **Dog dispatch handler** — Daemon lifecycle defaults include handler for
  direct dog dispatch.
- **Formula v2** — mol-idea-to-plan with iterative review rounds and inline
  eval/smoke-test ticket creation.

### Fixed

- **Idle patrol CPU burn** — Patrol agents no longer burn CPU/tokens in handoff
  restart loops.
- **Compactor-dog false positives** — Fixed concurrent write detection and hash
  validation for Dolt base32 format.
- **Dolt server stability** — Fixed stale socket cleanup, server ownership
  detection, rogue process race on restart, idle-monitor orphans on `gt down`.
- **Cross-feature wisp contamination** — MQ list filtered by feature to prevent leaks.
- **Agent lifecycle** — Fixed idle reuse with live sessions, CRASHED_POLECAT
  alerts for closed tickets, spawn storm dedup.
- **Session prefix parsing** — Fixed hq- prefix collision and feature-level fallback.
- **Unicode handling** — Fixed parse errors in `gt compact`.
- **Non-Claude agent support** — Liveness env vars, idle-wait instructions, and
  nudge startup prompts for Gemini/Codex runtimes.
- **Test isolation** — 5 tests isolated from live Dolt server; sleep sessions
  used in cleanup tests to avoid .zshrc interference.
- **QA Engineer completion notifications** — Product Manager now notified on agent completion.
- **Shell quoting** — Agent args properly quoted, model flags respected.
- **Exponential backoff** — Convoy event poller backs off on Dolt errors.
- **Docker** — Added tini for zombie process reaping in containers.

## [0.12.0] - 2026-03-11

### Added

- **Event-driven agent lifecycle** — Agents now use FIX_NEEDED feedback loop
  with awaiting_verdict state, replacing polling-based lifecycle (gt-k0h).
- **Cross-database convoy resolution** — CLI-side dep resolution for multi-feature
  towns where bd SQL JOINs fail across databases (GH#2624, GH#2625).
- **Plugin sync** — `gt plugin sync` auto-deploys plugins after build (hq-o9gna).
- **Compactor dog** — Executable `run.sh` for Dolt database compaction with
  DoltHub remote sync, validation, and dry-run support.
- **GitHub sheriff v2** — Single API call PR categorization with structured output.
- **Mail reply reminders** — Deferred nudge delivery for unanswered mail.
- **Git hygiene dog** — Automated repo cleanup plugin (gt-cdm).
- **Engineers agent assignment** — Town-level `engineers_agents` config for per-engineers
  agent runtime selection.
- **Partial clones** — `gt feature add` supports `--reference` for submodule init
  and sparse checkout.
- **Formula composition** — `extends` and `compose/expand` support for formulas.
- **Background nudge poller** — Queue-based nudge delivery for non-Claude agents.
- **Review command** — `/review` with A-F grading and release engineer integration (#2636).
- **Escalation channels** — Email, Slack, SMS, and log notification channels.
- **Pressure checks** — Opt-in CPU/memory pressure gating before agent spawns.
- **MVGT integration guide** — Comprehensive Wasteland federation guide for
  non-Gas-Town systems.
- **Engineers specialization design** — Capability-based dispatch design doc.

### Changed

- **Release Engineer merge strategy** — Configurable direct vs PR mode (gt-fln).
- **Agent lifecycle patrol** — Redesigned formula for event-driven model.
- **Session hygiene** — Converted from plugin.md to deterministic `run.sh`.
- **DND auto-reset** — Muted mode auto-resets on `gt up`.
- **Nudge degradation** — Wait-idle gracefully degrades to queue for agents
  without prompt detection.

### Fixed

- **Install bootstrap** — `gt install` now waits for MySQL readiness and always
  passes `--server-port` to `bd init` (GH#2572, GH#2573).
- **Feature add database creation** — `InitFeature` runs CREATE DATABASE on live server
  before schema migration.
- **Boot triage loop** — Removed ZFC-violating decision engine that consumed
  unbounded tokens on failed installs.
- **Agent spawn storm** — Two-layer circuit breaker caps respawns and total
  active agents (clown show #22).
- **Standing-order tickets** — Protected from AutoClose reaper and agent
  removal status reset.
- **Tmux socket split-brain** — Prevented nudge failures from socket mismatch
  (GH#2442).
- **Reaper Sprintf bugs** — Fixed format string tickets and missing schema guard
  (GH#2469).
- **bd JSON corruption** — Strip bd stdout warnings before JSON parsing.
- **Remote branch deletion** — Restricted to agent branches only (GH#2669).
- **EnsureAllMetadata** — Uses feature name and correct DB prefix (GH#2668).
- **Ephemeral tickets** — Auto-purge closed ephemeral tickets on session end.
- **MR back-link** — Source ticket linked to MR ticket on creation.
- **Convoy routing** — Town root and BEADS_DIR properly stripped for bd
  subprocess calls.
- **CI stability** — Resolved lint warnings (unparam, misspell) and 5 test
  failures on main.

## [0.11.0] - 2026-03-05

### Added

- **Docker support** — docker-compose and Dockerfile for containerized deployment.
- **Cursor hooks** — Agent agent integration for Cursor IDE sessions.
- **Context-budget guard** — External script to prevent context window overflow (#2008).
- **Cascade close** — `bd close --cascade` closes parent and all children with cycle
  guard and depth limit (#998).
- **Schema evolution** — `gt wl sync` supports Wasteland schema changes (gp-c7e).
- **Dashboard enrichment** — Convoy panel shows progress %, ready/active counts,
  and assignees.
- **Agent slot env** — `POLECAT_SLOT` environment variable for test isolation (#954).

### Changed

- **Tickets dependency** upgraded from v0.57.0 to v0.59.0.
- **Hook installers consolidated** — Per-agent hook installer packages replaced with
  generic declarative system (gt-071h).
- **Agent preset registry** — Hardcoded `isKnownAgent` switch replaced with
  `config.IsKnownPreset` (gt-7r3c).
- **Reaper TTLs shortened** — Auto-close 7d, purge 3d (previously longer).
- **`CreateOptions.Type` deprecated** in favor of Labels.

### Removed

- **`gt swarm` command** — Deprecated command and `internal/swarm` package removed (#1170).
- **Tickets Classic legacy code** — Remaining SQLite/JSONL/sync code paths removed.
- **Vestigial `sync.mode` plumbing** — Dead config removed.

### Fixed

- **Serial killer bug** — Removed hung session detection that was killing healthy
  QA engineeres and refineries (f3d47a96). Stuck agent detection moved to Dog plugin
  (5a5deaac).
- **Sling race condition** — Hook write visibility ensured before agent startup
  (GH#2389).
- **Release Engineer** — PostMerge now uses `ForceCloseWithReason` for source ticket (GH#2321).
- **Engineers mail prefix** — Regression test added for engineers mail send prefix mismatch
  (gt-brip).
- **bd JSON guard** — Non-JSON output from bd v0.58.0 handled in remaining parsers
  (gt-ac0i).
- **CI release guards** — Blocks `go.mod` replace directives in releases (gt-qex2).
- **go vet** — Pre-existing failures on main resolved (gt-77xe).
- **Branch contamination** — Preflight check added to `gt done` (#2220).
- **Agent nuke** — Uses `ClonePath` for best-effort push (hq-9pcb0).
- **Agent state** — JSON list state reconciled with session liveness.
- **Convoy** — External tracked IDs resolved during launch collection.
- **`gt done`** — Correct feature used when Claude Code resets shell cwd. Tolerates
  Gas Town runtime artifacts in worktrees (#2382).
- **Dolt server** — Server-side timeouts prevent CLOSE_WAIT accumulation (#2287).
- **Daemon** — 5-minute grace period before auto-closing empty convoys (GH#2303).
- **Sling TTL** — Prevents permanent scheduling blocks (GH#2279).
- **Tmux** — Refresh cycle bindings when prefix pattern is stale (#2300).
- **Patrol** — Cap stale cleanup and break early on active patrol found (#2285).
- **Reaper** — Correct database name; O(n*m) correlated EXISTS replaced with LEFT
  JOIN anti-pattern.
- **Hook show** — Normalized targets; prefer hooked ticket over stale agent hook.
- **Tmux socket** — Derived from town name instead of defaulting to "default".
- **Gitignore** — Broadened patterns for Cursor runtime artifacts and Gas Town
  infrastructure directories.
- **Feature remove** — Shows actionable guidance for orphaned feature directories.
- **CI** — Lint errors, Windows test failures, proxy log truncation fixed.
- **Product Manager clone** — Reuses bare repo as reference for faster cloning (#1059).
- **Prefix registry** — Reloaded on heartbeat to prevent ghost sessions (#2338).
- **Dog molecule** — JSON parsing fix for `bd show --children` output.
- **`--allow-stale`** — Made conditional on bd version support.

## [0.10.0] - 2026-03-03

_Release contained incremental fixes between v0.9.0 and v0.10.0. See git log for details._

## [0.9.0] - 2026-03-01

### Added

- **Batch-then-bisect merge queue** — Bors-style MQ batches MRs, runs tests on
  the tip, and binary-bisects on failure. GatesParallel runs test + lint
  concurrently per MR.
- **Persistent agents** — Agent identity and sandbox survive across work
  assignments. Sessions are ephemeral; identity accumulates forever. `gt done`
  transitions to idle instead of nuking.
- **Compactor Dog** — Daily Dolt commit history flattening via `DOLT_RESET --soft`.
  Runs on live server, no downtime. Configurable threshold. Includes surgical
  interactive rebase for advanced use.
- **Doctor Dog** — Automated health monitoring patrol for Dolt server. Detects
  zombies, orphan databases, and stale locks. Reports structured data for agent
  decision-making (ZFC-compliant).
- **JSONL Dog** — Spike detection and pollution firewall for JSONL backup exports.
  Scrubs test artifacts before git commit.
- **Wisp Reaper Dog** — Automated wisp GC with DELETE of closed wisps >7d, auto-close
  of stale tickets >30d, and mail purge >7d.
- **Root-only wisps** — Formula steps no longer materialized as individual database
  rows. Single root wisp per formula. Cuts ~6,000 ephemeral rows/day from Dolt.
- **Six-stage data lifecycle** — CREATE, LIVE, CLOSE, DECAY, COMPACT, FLATTEN,
  all automated via Dogs. `EnsureLifecycleDefaults()` auto-populates daemon.json
  on startup.
- **`gt maintain`** — One-command Dolt maintenance (flatten + gc).
- **`gt vitals`** — Unified health dashboard command.
- **`gt upgrade`** — Post-binary migration command for config propagation.
- **`gt convoy stage`** — Stage convoys with `--title` flag and smart defaults.
- **`gt mol await-signal`** — Alias for step-level signal awaiting.
- **`gt daemon clear-backoff`** — Reset exponential backoff on daemon restarts.
- **`gt mq post-merge`** — Branch cleanup after release engineer merge.
- **`gt mail drain`** — Drain inbox command.
- **Sandbox sync** — Branch-only agent reuse with `pool-init`.
- **Per-worker agent selection** — `worker_agents` config for engineers members.
- **Dangerous-command guard hook** — Blocks `cp -i`, `mv -i`, `rm -i` in agent sessions.
- **Log rotation** — Daemon and Dolt server logs now rotate with gzip compression.
- **Did-you-mean suggestions** — Unknown subcommands suggest closest match.
- **`gt version --verbose/--short`** — Version display options.
- **Town-root CLAUDE.md version check** — `gt doctor` detects stale CLAUDE.md files.
- **Lifecycle defaults doctor check** — Detects missing daemon.json patrol entries.

### Changed

- **OperationalConfig for config-driven thresholds (ZFC)** — Hardcoded thresholds
  for hung sessions, stale claims, max retries, GUPP violations, and crash-loop
  backoff are now in `operational.json`. Go code reads config; agents set values.
- **Nudge-first communication** — Protocol messages (LIFECYCLE, WORK_DONE,
  merge notifications) changed from permanent mail to ephemeral wisps or nudges.
  Reduces Dolt commit volume by ~80% for patrol traffic.
- **Default Dolt log level** changed from debug to info.
- **Convoy IDs** shortened from 5 chars to 3.
- **Mail purge age** reduced from 30 days to 7 days.
- **Compactor threshold** bumped from 500 to 10,000 commits.
- **Default max Dolt connections** bumped from 200 to 1,000.
- **Dolt flatten runs on live server** — No maintenance window needed.

### Fixed

- **Mail delivery** — Fixed `--id` flag breaking all mail creation paths. Fixed
  recipient validation to include pinned and inactive agents. Added auto-nudge
  on mail delivery to idle agents.
- **QA Engineer patrol** — Stopped nuking idle agents. Stopped auto-closing permanent
  tickets (ZFC violation). Replaced screen-scraping with structured signals for
  stalled agent detection.
- **Agent lifecycle** — Prevent sleepwalking completions with zero commits. Normalize
  CWD to git root before tickets resolution. Preserve remote branches when MR pending.
  Close orphan step wisps before closing root molecule.
- **Release Engineer** — Removed dead `findTownRoot` filesystem inference (ZFC). Removed
  hardcoded severity/priority logic. Made stale-claim thresholds config-driven.
  Extracted typed `ConvoyFields` accessor.
- **Boot triage** — Removed ZFC-violating decision engine from degraded boot triage.
- **PID detection** — Replaced PID signal probing with heartbeat-based liveness.
  Replaced ps string matching with nonce-based PID files. Replaced per-PID pgrep
  with single ps-based process tree scan.
- **tmux** — Use default socket instead of per-town socket. Set `window-size=latest`
  on detached sessions. Auto-dismiss workspace trust dialog blocking agent sessions.
- **Convoy** — Prevent auto-close of stuck convoys. Recovery sweep after Dolt
  reconnect. Check parked AND docked features in dispatch. Prevent double-spawn from
  stale feed. Idempotent close handling.
- **Doctor** — Fixed silent failure for agent-tickets and misclassified-wisps checks.
  Repair tickets redirect targets with missing config.yaml. Handle worktree branch
  conflicts. Skip feature dirs whose .tickets symlinks to town root.
- **Test isolation** — Migrate test infrastructure to testcontainers. Ephemeral Dolt
  server per test suite. `BEADS_TEST_MODE=1` enforcement. Dropped 45 zombie test
  servers (7GB RAM).
- **Feature lifecycle** — Enforce dock/park across all startup, patrol, and sling paths.
  Dogs correctly use plugin lookup. Formula lookup falls back to embedded for
  non-gastown features.
- **Handoff** — Deterministic git state in handoff context. Fix socket confusion
  using caller socket. Preserve conversation context on PreCompact cycle.
- **Windows** — Cross-compilation fixes for `syscall.Flock`, `syscall.SIGUSR2`,
  bash-dependent tests, `GOPATH/bin` creation, batch mock caret escaping.
- **Hardcoded strings** — Replaced ~50 hardcoded state/status string comparisons
  with typed enums across QA engineer, release engineer, and tickets packages.
- **CI** — Fixed lint errors (errcheck, misspell) and integration test port collision.

## [0.8.0] - 2026-02-23

### Added

#### Work Queue & Dispatch Engine
- **`gt queue` CLI and dispatch engine** — Enqueue work items for daemon-driven dispatch
- **`gt queue epic`** — Bulk enqueue of epic children
- **`gt convoy queue`** — Bulk enqueue of convoy-tracked tickets
- **`gt sling --queue`** — Enqueue path for asynchronous dispatch
- **Queue daemon heartbeat step** for reliable dispatch processing
- **Enqueue-time validation** and enhanced metadata on queued items
- **Config-driven capacity scheduler** with sling context tickets

#### Telemetry & Observability (OTEL)
- **Full OpenTelemetry instrumentation** across gt, bd, and Claude Code
- **VictoriaMetrics and VictoriaLogs integration** for metrics and log export
- **Lifecycle events, duration histograms, and operation traces**
- **Claude Code tool content and user prompt logging** via OTLP
- **`gt.* OTEL` resource attributes**, prime context events, and session metadata
- **Command usage telemetry** and `gt metrics` reader

#### Dog Subsystem (Senior Engineer Workers)
- **Handler patrol** for Dog lifecycle and plugin dispatch
- **Session-hygiene Dog plugin** for zombie tmux cleanup
- **Idle dog reaping** — Kill stale tmux sessions and trim oversized pools
- **Stale working dog detection** for dogs stuck with idle sessions
- **Wisp compaction formula** for Dogs
- **Shutdown dance molecule** — Death warrant execution state machine for Dogs

#### Wisps & Ephemeral Storage
- **Wisps table migration** for agent tickets (ephemeral SQLite store)
- **Agent ticket readers updated** to query wisps table
- **Closed wisp GC** added to patrol cycles to prevent accumulation

#### Convoy & Sling Improvements
- **`gt convoy stage` and `gt convoy launch`** commands
- **Auto-resolve feature from ticket prefixes** in batch mode
- **Reject deferred/post-launch tickets** at sling time
- **Auto-check convoy completion** on `bd close`
- **Senior Engineer feed-stranded command** for auto-feeding stranded convoys
- **Notify senior engineer of convoy-eligible merges** for immediate feeding

#### QA Engineer & Release Engineer
- **Configurable quality gates** before merge in release engineer
- **MR queue nonempty tracking** in QA engineer check-release engineer step
- **Ticket respawn count tracking** for spawn storm detection
- **Verify MR ticket exists** before sending MERGE_READY
- **Release Engineer nudges senior engineer** after merge to check stranded convoys
- **Auto-close completed convoys** after merge

#### Agent Providers & Runtime
- **Pi agent provider** with unit tests
- **Promptfoo model comparison framework** for patrol agents
- **Cost-tier presets** for model selection
- **Ralphcat opt-in loop mode** for multi-step workflows

#### Dashboard & UX
- **Rename Workers panel to Agents** in dashboard
- **Session terminal preview** in dashboard Sessions panel
- **Mail threading** in dashboard inbox
- **Convoy drill-down** — Expand rows to show tracked tickets
- **Date+time timestamps** instead of relative ago format

#### CLI & Workflow
- **`gt handoff --cycle`** — Full session replacement flag
- **`gt handoff --agent`** — Explicit runtime selection
- **`gt engineers start --resume`** — Resume flag for engineers sessions
- **`gt engineers at --reset`** — Branch switch opt-in with reset flag
- **`gt senior engineer pending`** — AI-based spawn observation command
- **`gt patrol step-drift`** subcommand
- **`gt up --json`** output flag
- **`gt migrate-ticket-labels`** command
- **Desire-path fixes** for feature settings, engineers list, mail directory
- **Idle-aware notification** in mail router with auto-nudge on send

#### Infrastructure
- **WorktreeCreate/WorktreeRemove hook event support**
- **Remote Dolt server support** (`gt dolt`)
- **Dolt server identity verification and restart command**
- **Orphaned Dolt branch cleanup**
- **Nix flake** and integration with bump-version script
- **Boot block scraper VM deployment script**
- **Tickets-redirect-target doctor check**
- **Unified zombie session detection** with 3-level health check
- **Emit-event command** and nudgeRelease Engineer event wiring
- **`gt mol step await-event`** for channel-based event subscription
- **Doctor check** for in_progress tickets with NULL assignee
- **PR creation guard** blocking direct pushes to steveyegge/gastown
- **Orphan scan** for agent worktrees with unmerged branches
- **Wasteland CLI command suite** (`gt wl`)
- **GitHub Sheriff Senior Engineer plugin** for CI failure monitoring

### Changed

- **Removed Dolt branch-per-agent infrastructure** — Dead code from pre-server-mode era
- **Removed BD_BRANCH from session manager and agent spawn** — Server mode eliminates branch isolation
- **Removed JSONL fallback code** — Dolt is now the sole backend
- **Removed deprecated role ticket TTL layer** from compact
- **Removed deprecated role ticket lookups** that blocked QA engineer startup
- **Stale JSONL references** cleaned from documentation and comments
- **Molecule squash `--no-digest`** flag to skip digest creation
- **Dog startup honors role_agents config**
- **Handoff restart honors role_agents config**

### Fixed

#### Dolt & Storage Stability
- **Replace dolthub/dolt with nil-check fork** to fix SEGVs
- **Prevent agent spawns from creating orphan databases**
- **Drop orphaned `tickets_<prefix>` databases** after `bd init`
- **Isolate Dolt integration tests** from production server
- **Dynamic port for test Dolt server** to avoid production collision
- **Restore BEADS_DIR stripping** in `buildRunEnv()` lost in merge
- **Strip BD_BRANCH from wisp creation and cook steps**

#### Agent & Session Management
- **LookupEnv for GT_AGENT** to prevent tmux env contamination
- **KillSession idempotent** + ensureProduct ManagerInfra on attach
- **Detect non-agent CWD** when Claude Code resets shell to product manager/feature
- **Filter getCurrentWork by assignee** to prevent shared feature tickets leaking across sessions
- **Clean up stale molecules during agent nuke** to unblock re-sling
- **Set GT_PROCESS_NAMES in tmux env** for all session types
- **Resolve agent config from --agent override** for Codex agent startup
- **Tell agents to close ticket explicitly** when nothing to commit
- **Silent costs-record skip** for non-GT sessions

#### Test Infrastructure
- **Flaky TestFindActivePatrol tests** caused by bd daemon contamination
- **Isolated patrol tests** from shared Dolt database
- **Convoy/epic test failures** and TestValidateRecipient resolved
- **Extract requireDoltServer** into shared testutil package
- **gotestsum + JUnit test failure reporting** in CI

#### Patrol & Boot
- **Remove redundant Status.Running field** (ZFC compliance)
- **Prefer gastown repo over engineers features** in stale binary check
- **Session-name-format doctor check** for outdated session names
- **Handoff warn on uncommitted/unpushed work** before session cycling
- **Helpful error for `gt mail read` with no args** instead of cobra usage

## [0.7.0] - 2026-02-15

### Added

#### Convoy Ownership & Merge Strategies
- **Convoy ownership model** — `--owned` flag for `gt convoy create` and `gt sling`
- **Merge strategy selection** — `--merge` flag with `direct`, `mr`, and `local` strategies
- **`gt convoy land`** — New command for owned convoy cleanup and completion
- **Skip QA engineer/release engineer registration** for owned+direct convoys (faster dispatch)
- **Ownership and merge strategy display** in `gt convoy status` and `gt convoy list`

#### Agent Resilience & Lifecycle
- **`gt done` checkpoint-based resilience** — Recovery from session death mid-completion
- **Agent factory** — Data-driven preset registry replaces provider switch statements
- **Gemini CLI integration** — First-class Gemini CLI as runtime adapter
- **GitHub Copilot CLI integration** — Copilot CLI as runtime adapter
- **Non-destructive nudge delivery** — Queue and wait-idle modes prevent message loss
- **Auto-dismiss stalled agent permission prompts** — QA Engineer detects and clears stuck prompts
- **Dead engineers agent detection** — Detect dead engineers agents on startup and restart them
- **Remote hook attach** — `gt hook attach` with remote target support

#### Dashboard & Web UI
- **Rich activity timeline** — Chronological view with filtering
- **Mobile-friendly responsive layout** — Dashboard works on small screens
- **Toast notifications and escalation actions** — Interactive escalation UI
- **Escape key closes expanded panels** — Keyboard navigation improvement

#### QA Engineer & Patrol
- **JSON patrol receipts** for stale/orphan verdicts — Structured patrol output
- **Orphaned molecule detection** — Detect and close orphaned `mol-agent-work` molecules
- **IN_PROGRESS tickets assigned to dead agents** — Automatic detection and recovery
- **Deterministic stale/orphan receipt ordering** — Consistent patrol results

#### Infrastructure
- **Submodule support** — Worktree and release engineer merge queue support for git submodules
- **Merge queue `--verify` flag** — Detect orphaned/missing merge queue entries
- **Cost digest aggregate-only payload** — Fixes Dolt column overflow
- **Feature-specific tickets prefix for tmux session names** — Better multi-feature session isolation
- **Product Manager GT_ROLE Task tool guard** — Block Task tool for Product Manager via GT_ROLE check
- **Server-side database creation** during `gt feature add` with ticket_prefix setup

### Changed

- **Tickets Classic dead code removed** — -924 lines of SQLite/JSONL/sync code eliminated
- **Agent provider consolidated** — Data-driven preset registry replaces switch statements
- **Session prefix renamed** — Registry-based prefixes replace hardcoded `gt-*` patterns
- **Agent config resolution** — Moved mutex to package config for thread safety
- **Molecule step readiness** — Delegated to `bd ready --mol` instead of custom logic

### Fixed

#### Reliability & Race Conditions
- **Options cache and command concurrency** race conditions in web dashboard
- **Feed curator** race conditions with RWMutex protection
- **TUI convoy** concurrent access guarded with RWMutex
- **TUI feed** concurrent access guarded with RWMutex
- **Dolt backoff** — Thread-safe jitter using `math/rand/v2`
- **Concurrent Start()** and feed file access protection
- **QA Engineer manager** race condition fix

#### Agent & Session Management
- **Nudge delivery** — Unique claim suffix prevents Windows race in concurrent Drain
- **Signal stop hook** — Prevent infinite loop with state-based dedup
- **Agent zero-commit completion** blocked — Must have at least one commit
- **Molecule step instructions** — Use `bd mol current` instead of `bd ready`
- **Role inference** — Don't infer RoleProduct Manager from town root cwd
- **Boot role ticket ID** — Add RoleBoot case to buildAgentTicketID and ActorString
- **IsAgentRunning replaced with IsAgentAlive** — More accurate agent status
- **Stale prime help text** updated with town root regression tests
- **Sling validation** — Allow agent/engineers shorthand, validate before dispatch fork

#### Convoy & Workflow
- **Convoy lifecycle guards** — Extended to batch auto-close and synthesis paths
- **Empty convoy handling** — Auto-close and flag in stranded detection
- **Formalized lifecycle transition guards** for convoys

#### Feature & Infrastructure
- **Feature remove kills tmux sessions** — Clean up sessions on feature removal
- **Feature adopt** — Init `.tickets/` when no existing database found
- **Revert shared-DB for untracked-tickets features** — Fixes ticket creation breakage
- **Install preserves existing configs** — `town.json` and `features.json` kept on re-install
- **Orphaned dolt server** detected and stopped during `gt install`
- **Doctor dolt check** — Uses platform-appropriate mock binaries, adds dolt binary check
- **Doctor dolt-server-reachable** reads host/port from metadata instead of hardcoding
- **IPv6 safety** and accurate feature count in doctor
- **Feature remove** aborts on kill failures, propagates session check errors
- **Spurious `go build` warning** fixed for Homebrew installs

#### Senior Engineer & Dogs
- **Senior Engineer scoped zombie/orphan detection** to Gas Town workspace only
- **Senior Engineer heartbeat** surfaced in `gt senior engineer status`
- **Senior Engineer loop-or-exit** step updated with squash/create-wisp/hook cycle
- **Dog agent tickets** — Added description for mail routing

#### Other Fixes
- **Overflow agent names** — Remove feature prefix
- **QA Engineer per-label `--set-labels=` pattern** — Improved tests
- **Feed auto-disable follow** when stdout is not a TTY
- **Mail inject** — Improved output wording and test coverage
- **Codex config** — Replace invalid `--yolo` with `--dangerously-bypass-approvals-and-sandbox`
- **Cross-prefix tickets routing** via `runWithRouting` for slot ops
- **Tmux `-u` flag** added to remaining client-side callsites
- **JSON output** — Return `[]` instead of `null` for empty slices
- **Windows CI** cut from ~13 min to ~4 min
- **Dolt server auto-start** in `gt start`
- 50+ additional bug fixes from community contributions

## [0.6.0] - 2026-02-15

### Added

#### Dolt-Native Architecture
- **Complete SQLite-to-Dolt migration** - Gas Town is now Dolt-native; all SQLite code removed
- **`gt dolt` command** - Server management (start, stop, status, migrate, rollback, sync, init-feature)
- **`gt install` consolidation** - Folds Dolt identity, HQ database, and server start into single command
- **Branch-per-agent write isolation** - Each agent gets its own Dolt branch to prevent write conflicts
- **Proactive Dolt health alerting** - Daemon monitors server health with dedicated 30s ticker
- **Auto-create DoltHub repos and configure remotes** - `gt dolt sync` pushes to DoltHub
- **Dolt remotes patrol** - Periodic push to git remotes for federation readiness
- **Max-connections and admission control** - Prevents connection storms on Dolt server

#### Dashboard & Web UI
- **Comprehensive UX overhaul** - 13 interactive data panels with engineers notifications
- **SSE real-time updates** - Replaces 10s polling with server-sent events
- **Interactive command palette** - Autocomplete, recent history, contextual suggestions
- **Interactive convoy management** - Create, close, feed convoys from dashboard
- **Interactive hook management** - View and manage hooks from dashboard
- **Ticket management actions** - Work panel detail view with sling buttons
- **Convoy titles alongside IDs** - Better convoy identification

#### Daemon & Supervision
- **launchd/systemd supervision support** - OS-native daemon management
- **Exponential backoff for agent restarts** - Prevents restart storms
- **Product Manager daemon supervision** - Daemon manages Product Manager session lifecycle
- **Boot watchdog** - Ephemeral dog that triages Senior Engineer state on each daemon tick

#### Molecule & Workflow System
- **DAG visualization** (`gt mol dag`) - Visualize molecule dependency graphs
- **Fan-out/gather pattern** - Parallel steps in patrol workflows
- **Wisp compaction** (`gt compact`) - TTL-based wisp lifecycle management
- **Key Record Chronicle (KRC)** - Forensic value decay model for ephemeral data
- **Wisp promotion criteria** - Helpers for squash-to-persistent decisions
- **Formula variable declarations** - Proper `[vars]` sections in all formulas

#### Hooks Management
- **Centralized hook management** - `gt hooks sync`, `gt hooks diff`, `gt hooks list`
- **Per-matcher merge logic** - Fine-grained hook configuration
- **Hook registry integration** - Hooks wired into `gt feature add` and `gt doctor`

#### Agent Lifecycle
- **Persistent agent identity model** - Agent tickets survive nuke; identity accumulates forever
- **Auto re-dispatch recovered tickets** - Senior Engineer recovers work from dead agents
- **QA Engineer resets abandoned tickets** - Dead agent detection tfeaturegers work recovery
- **Auto-respawn hooks** - Product Manager sessions survive tmux detach
- **Signal stop handler** (`gt signal stop`) - Turn-boundary messaging for clean stops
- **PID tracking for spawned agents** - Better process management

#### Convoy System
- **Completion notifications** - Push convoy completion to active Product Manager session
- **Auto-close empty convoys** - Empty 0/0 convoys auto-closed on create
- **`--merge` and `--owned` flags** for `gt convoy create` and `gt sling`
- **Safety checks on `gt convoy close`** with `--force` override
- **Reactive convoy continuation feeding** - Observer auto-feeds convoys

#### CLI Improvements
- **`--stdin` flag** - Shell-quoting-safe message bodies for mail, nudge, handoff, escalate, sling
- **`--auto` flag for handoff** - PreCompact auto-handoff support
- **`gt hook clear`** - Alias for `gt unhook`
- **`gt dog clear` and `gt warrant`** - Dog management commands
- **`gt feature settings`** - Interactive feature settings management
- **`--adopt` flag for `gt feature add`** - Register existing directories
- **Enhanced `--help` text** - Long descriptions added to 30+ commands
- **Dark mode CLI theme support** - Configurable terminal themes
- **Agent switcher keybinding** - `C-b g` for tmux agent switching
- **`gt prime` compact/resume detection** - Lighter post-compaction priming
- **Command quick-reference in CLAUDE.md** - Auto-generated per role

#### Community Contributions
- **Containerized E2E tests** - Docker-based install and daemon testing
- **Integration branch enhancements** - End-to-end integration branches across the pipeline
- **Stale claim timeout in release engineer** - Prevents stuck MRs
- **Serialize main pushes with merge slot** - Prevents push conflicts
- **Agent-agnostic zombie detection** - Works with any AI agent, not just Claude
- **Configurable CLI name** (`GT_COMMAND` env var) - For custom installations
- **Compaction reporting** - Daily digest and weekly rollup

### Changed

- **Dolt is the only backend** - All SQLite code removed; `--no-daemon` flag deprecated
- **Settings moved to `settings.local.json`** - Cleaner separation from repo config
- **`gt status --fast` optimized** - From ~5s to ~2s
- **Release Engineer squash merge** - Closed MRs excluded from queue output
- **Priority-based mail notifications** - Prevents agent derailment from low-priority mail
- **Compaction reporting** - Automated daily digest and weekly rollup
- **Formula template rendering** - Go text/template for convoy prompts
- **Centralized configuration** - Hardcoded timeouts and thresholds moved to TownSettings

### Fixed

#### Reliability & Race Conditions
- **flock-based locking** across molecule attach, events/feed writes, engineers files, and lock acquisition
- **TOCTOU guards** on Dolt server startup, worktree operations, cleanup actions, and FindFeatureTicketsDir
- **Atomic writes** for catalog, routes, settings, and per-ticket files
- **Thread-safe NotificationManager** with mutex protection
- **Deadlock elimination** in daemon restart backoff tests
- **Process group termination** using verified member enumeration instead of blind PGID kill

#### Security Hardening
- **Input validation** for web dashboard handlers, group names, dog names, ticket creation
- **Path traversal prevention** in dog names, feature names, and ticket operations
- **Shell injection prevention** via session name validation and branch name sanitization
- **Rejected flag-like titles** to prevent garbage ticket creation from malformed commands

#### Dolt Backend
- **Read-only state auto-recovery** - Commands detect and recover from Dolt read-only mode
- **Split-brain prevention** when `bd` used before Dolt server starts
- **Exponential backoff with jitter** for Dolt retries (10 attempts)
- **Database verification** after migration and server start
- **Orphaned database detection and cleanup** in `.dolt-data/`

#### Session & Process Management
- **Agent nuke improvements** - Close open MRs, verify worktree removal, handle cd'd shells
- **Zombie detection** - tmux-alive-but-agent-dead detection, cleanup_status handling
- **Respawn protection** - Prevents destroying unmerged MR work on respawn
- **Nudge backoff** - Correct cap, reduced timeout, transient error retry
- **Session name parsing** - Handles hyphenated feature names correctly
- **NBSP normalization** - Fixes Claude Code readiness detection

#### Cross-Feature Operations
- **Tickets routing fixes** - Correct prefix detection, redirect topology verification
- **Cross-feature agent ticket operations** routed to correct database
- **Convoy tracking** - Proper external ticket status refresh and stranded detection
- **Doctor checks** continue on error in agent/feature ticket fix methods

#### Many more fixes
- 200+ bug fixes from community contributions and internal development
- See `git log v0.5.0..v0.6.0` for complete details

## [0.5.0] - 2026-01-22

### Added

#### Mail Improvements
- **Numeric index support for `gt mail read`** - Read messages by inbox position (e.g., `gt mail read 1`)
- **`gt mail hook` alias** - Shortcut for `gt hook attach` from mail context
- **`--body` alias for `--message`** - More intuitive flag in `gt mail send` and `gt mail reply`
- **Multiple message IDs in delete** - `gt mail delete msg1 msg2 msg3`
- **Positional message arg in reply** - `gt mail reply <id> "message"` without --message flag
- **`--all` flag for inbox** - Show all messages including read
- **Parallel inbox queries** - ~6x speedup for mail inbox

#### Command Aliases
- **`gt bd`** - Alias for `gt ticket`
- **`gt work`** - Alias for `gt hook`
- **`--comment` alias for `--reason`** - In `gt close`
- **`read` alias for `show`** - In `gt ticket`

#### Configuration & Agents
- **OpenCode as built-in agent preset** - Configure with `gt config set agent opencode`
- **Config-based role definition system** - Roles defined in config, not tickets
- **Env field in RuntimeConfig** - Custom environment variables for agent presets
- **ShellQuote helper** - Safe env var escaping for shell commands

#### Infrastructure
- **Senior Engineer status line display** - Shows senior engineer icon in product manager status line
- **Configurable agent branch naming** - Template-based branch naming
- **Hook registry and install command** - Manage Claude Code hooks via `gt hooks`
- **Doctor auto-fix capability** - SessionHookCheck can auto-repair
- **`gt orphans kill` command** - Clean up orphaned Claude processes
- **Zombie-scan command for senior engineer** - tmux-verified process cleanup
- **Initial prompt for autonomous patrol startup** - Better agent priming

#### Release Engineer & Merging
- **Squash merge for cleaner history** - Eliminates redundant merge commits
- **Redundant observers** - QA Engineer and Release Engineer both watch convoys

### Fixed

#### Engineers & Session Stability
- **Don't kill pane processes on new sessions** - Prevents destroying fresh shells
- **Auto-recover from stale tmux pane references** - Recreates sessions automatically
- **Preserve GT_AGENT across session restarts** - Handoff maintains identity

#### Process Management
- **KillPaneProcesses kills pane process itself** - Not just descendants
- **Kill pane processes before all RespawnPane calls** - Prevents orphan leaks
- **Shutdown reliability improvements** - Multiple fixes for clean shutdown
- **Senior Engineer spawns immediately after killing stuck session**

#### Convoy & Routing
- **Pass convoy ID to convoy check command** - Correct ID propagation
- **Multi-repo routing for custom types** - Correct tickets routing across repos
- **Normalize agent ID trailing slash** - Consistent ID handling

#### Miscellaneous
- **Sling auto-apply mol-agent-work** - Auto-attach on open agent tickets
- **Wisp orphan lifecycle bug** - Proper cleanup of abandoned wisps
- **Misclassified wisp detection** - Defense-in-depth filtering
- **Cross-account session access in seance** - Talk to predecessors across accounts
- **Many more bug fixes** - See git log for full details

## [0.4.0] - 2026-01-19

_Changelog not documented at release time. See git log v0.3.1..v0.4.0 for changes._

## [0.3.1] - 2026-01-18

_Changelog not documented at release time. See git log v0.3.0..v0.3.1 for changes._

## [0.3.0] - 2026-01-17

### Added

#### Release Automation
- **`gastown-release` molecule formula** - Workflow for releases with preflight checks, CHANGELOG/info.go updates, local install, and daemon restart

#### New Commands
- **`gt show`** - Inspect ticket contents and metadata
- **`gt cat`** - Display ticket content directly
- **`gt orphans list/kill`** - Detect and clean up orphaned Claude processes
- **`gt convoy close`** - Manual convoy closure command
- **`gt commit`** - Wrapper for git commit with ticket awareness
- **`gt trail`** - View commit trail for current work
- **`gt mail ack`** - Alias for mark-read command

#### Plugin System
- **Plugin discovery and management** - `gt plugin run`, `gt plugin history`
- **`gt dispatch --plugin`** - Execute plugins via dispatch command

#### Messaging Infrastructure (Tickets-Native)
- **Queue tickets** - New ticket type for message queues
- **Channel tickets** - Pub/sub messaging with retention
- **Group tickets** - Group management for messaging
- **Address resolution** - Resolve agent addresses for mail routing
- **`gt mail claim`** - Claim messages from queues

#### Agent Identity
- **`gt agent identity show`** - Display CV summary for agents
- **Worktree setup hooks** - Inject local configurations into worktrees

#### Performance & Reliability
- **Parallel agent startup** - Faster boot with concurrency limit
- **Event-driven convoy completion** - Senior Engineer checks convoy status on events
- **Automatic orphan cleanup** - Detect and kill orphaned Claude processes
- **Namepool auto-theming** - Themes selected per feature based on name hash

### Changed

- **MR tracking via tickets** - Removed mrqueue package, MRs now stored as tickets
- **Desire-path commands** - Added agent ergonomics shortcuts
- **Explicit escalation in templates** - Agent templates include escalation instructions
- **NamePool state is transient** - InUse state no longer persisted to config

### Fixed

#### Process Management
- **Kill process tree on shutdown** - Prevents orphaned Claude processes
- **Explicit pane process kill** - Prevents setsid orphans in tmux
- **Session survival verification** - Verify session survives startup before returning
- **Batch session queries** - Improved performance in `gt down`
- **Prevent tmux server exit** - `gt down` no longer kills tmux server

#### Tickets & Routing
- **Agent ticket prefix alignment** - Force multi-hyphen IDs for consistency
- **hq- prefix for town-level tickets** - Groups, channels use correct prefix
- **CreatedAt for group/channel tickets** - Proper timestamps on creation
- **Routes.jsonl protection** - Doctor check for feature-level routing tickets
- **Clear BEADS_DIR in auto-convoys** - Prevent prefix inheritance tickets

#### Mail & Communication
- **Channel routing in router.Send()** - Mail correctly routes to channels
- **Filter unread in tickets mode** - Correct unread message filtering
- **Town root detection** - Use workspace.Find for consistent detection

#### Session & Lifecycle
- **Idle Agent Heresy warnings** - Templates warn against idle waiting
- **Direct push prohibition for agents** - Explicit in templates
- **Handoff working directory** - Use correct QA engineer directory
- **Dead agent handling in sling** - Detect and handle dead agents
- **gt done self-cleaning** - Kill tmux session on completion

#### Doctor & Diagnostics
- **Zombie session detection** - Detect dead Claude processes in tmux
- **sqlite3 availability check** - Verify sqlite3 is installed
- **Clone divergence check** - Remove blocking git fetch

#### Build & Platform
- **Windows build support** - Platform-specific process/signal handling
- **macOS codesigning** - Sign binary after install

### Documentation

- **Idle Agent Heresy** - Document the anti-pattern of waiting for work
- **Ticket ID vs Ticket ID** - Clarify terminology in README
- **Explicit escalation** - Add escalation guidance to agent templates
- **Getting Started placement** - Fix README section ordering

## [0.2.6] - 2026-01-12

### Added

#### Escalation System
- **Unified escalation system** - Complete escalation implementation with severity levels, routing, and tracking (gt-i9r20)
- **Escalation config schema alignment** - Configuration now matches design doc specifications

#### Agent Identity & Management
- **`gt agent identity` subcommand group** - Agent ticket management commands for agent lifecycle
- **AGENTS.md fallback copy** - Agents automatically copy AGENTS.md from product manager/feature for context bootstrapping
- **`--debug` flag for `gt engineers at`** - Debug mode for engineers attachment troubleshooting
- **Boot role detection in priming** - Proper context injection for boot role agents (#370)

#### Statusline Improvements
- **Per-agent-type health tracking** - Statusline now shows health status per agent type (#344)
- **Visual feature grouping** - Features sorted by activity with visual grouping in tmux statusline (#337)

#### Mail & Communication
- **`gt mail show` alias** - Alternative command for reading mail (#340)

#### Developer Experience
- **`gt stale` command** - Check for stale binaries and version mismatches

### Changed

- **Refactored statusline** - Merged session loops and removed dead code for cleaner implementation
- **Refactored sling.go** - Split 1560-line file into 7 focused modules for maintainability
- **Magic numbers extracted** - Suggest package now uses named constants (#353)

### Fixed

#### Configuration & Environment
- **Empty GT_ROOT/BEADS_DIR not exported** - AgentEnv no longer exports empty environment variables (#385)
- **Inherited BEADS_DIR prefix mismatch** - Prevent inherited BEADS_DIR from causing prefix mismatches (#321)

#### Tickets & Routing
- **routes.jsonl corruption prevention** - Added protection against routes.jsonl corruption with doctor check for feature-level tickets (#377)
- **Tracked tickets init after clone** - Initialize tickets database for tracked tickets after git clone (#376)
- **Feature root from TicketsPath()** - Correctly return feature root to respect redirect system

#### Sling & Formula
- **Feature and ticket vars in formula-on-ticket mode** - Pass both variables correctly (#382)
- **Engineers member shorthand resolution** - Resolve engineers members correctly with shorthand paths
- **Removed obsolete --naked flag** - Cleanup of deprecated sling option

#### Doctor & Diagnostics
- **Role tickets check with shared definitions** - Doctor now validates role tickets using shared role definitions (#378)
- **Filter bd "Note:" messages** - Custom types check no longer confused by bd informational output (#381)

#### Installation & Setup
- **gt:role label on role tickets** - Role tickets now properly labeled during creation (#383)
- **Fetch ofeaturein after refspec config** - Bare clones now fetch after configuring refspec (#384)
- **Allow --wrappers in existing town** - No longer recreates HQ unnecessarily (#366)

#### Session & Lifecycle
- **Fallback instructions in start/restart beacons** - Session beacons now include fallback instructions
- **Handoff recognizes agent session pattern** - Correctly handles gt-<feature>-<name> session names (#373)
- **gt done resilient to missing agent tickets** - No longer fails when agent tickets don't exist
- **MR tickets as ephemeral wisps** - Create MR tickets as ephemeral wisps for proper cleanup
- **Auto-detect cleanup status** - Prevents premature agent nuke (#361)
- **Delete remote agent branches after merge** - Release Engineer cleans up remote branches (#369)

#### Costs & Events
- **Query all tickets locations for session events** - Cost tracking finds events across locations (#374)

#### Linting & Quality
- **errcheck and unparam violations resolved** - Fixed linting errors
- **NudgeSession for all agent notifications** - Mail now uses consistent notification method

### Documentation

- **Agent three-state model** - Clarified working/stalled/zombie states
- **Name pool vs agent pool** - Clarified misconception about pools
- **Plugin and escalation system designs** - Added design documentation
- **Documentation reorganization** - Concepts, design, and examples structure
- **gt prime clarification** - Clarified that gt prime is context recovery, not session start (GH #308)
- **Formula package documentation** - Comprehensive package docs
- **Various godoc additions** - GenerateMRIDWithTime, isAutonomousRole, formatInt, nil sentinel pattern
- **Tickets ticket ID format** - Clarified format in README (gt-uzx2c)
- **Stale agent identity description** - Fixed outdated documentation

### Tests

- **AGENTS.md worktree tests** - Test coverage for AGENTS.md in worktrees
- **Comprehensive test coverage** - Added tests for 5 packages (#351)
- **Sling test for bd empty output** - Fixed test for empty output handling

### Deprecated

- **`gt agent add`** - Added migration warning for deprecated command

### Contributors

Thanks to all contributors for this release:
- @JeremyKalmus - Various contributions (#364)
- @boshu2 - Formula package documentation (#343), PR documentation (#352)
- @sauerdaniel - Agent mail notification fix (#347)
- @abhijit360 - Assign model to role (#368)
- @julianknutsen - Tickets path fix (#334)

## [0.2.5] - 2026-01-11

### Added
- **`gt mail mark-read`** - Mark messages as read without opening them (desire path)
- **`gt down --agents`** - Shut down agents without affecting other components
- **Self-cleaning agent model** - Agents self-nuke on completion, QA engineer tracks leases
- **`gt prime --state` validation** - Flag exclusivity checks for cleaner CLI

### Changed
- **Removed `gt stop`** - Use `gt down --agents` instead (cleaner semantics)
- **Policy-neutral templates** - engineers.md.tmpl checks remote ofeaturein for PR policy
- **Refactored prime.go** - Split 1833-line file into logical modules

### Fixed
- **Agent re-spawn** - CreateOrReopenAgentTicket handles agent lifecycle correctly (#333)
- **Vim mode compatibility** - tmux sends Escape before Enter for vim users
- **Worktree default branch** - Uses feature's configured default branch (#325)
- **Agent ticket type** - Sets --type=agent when creating agent tickets
- **Bootstrap priming** - Reduced AGENTS.md to bootstrap pointer, fixed CLAUDE.md templates

### Documentation
- Updated QA engineer help text for self-cleaning model
- Updated daemon comments for self-cleaning model
- Policy-aware PR guidance in engineers template

## [0.2.4] - 2026-01-10

Priming subsystem overhaul and Zero Framework Cognition (ZFC) improvements.

### Added

#### Priming Subsystem
- **PRIME.md provisioning** - Auto-provision PRIME.md at feature level so all workers inherit Gas Town context (GUPP, hooks, propulsion) (#hq-5z76w)
- **Post-handoff detection** - `gt prime` detects handoff marker and outputs "HANDOFF COMPLETE" warning to prevent handoff loop bug (#hq-ukjrr)
- **Priming health checks** - `gt doctor` validates priming subsystem: SessionStart hook, gt prime command, PRIME.md presence, CLAUDE.md size (#hq-5scnt)
- **`gt prime --dry-run`** - Preview priming without side effects
- **`gt prime --state`** - Output session state (normal, post-handoff, crash-recovery, autonomous)
- **`gt prime --explain`** - Add [EXPLAIN] tags for debugging priming decisions

#### Formula & Configuration
- **Feature-level default formulas** - Configure default formula at feature level (#297)
- **QA Engineer --agent/--env overrides** - Override agent and environment variables for QA engineer (#293, #294)

#### Developer Experience
- **UX system import** - Comprehensive UX system from tickets (#311)
- **Explicit handoff instructions** - Clearer nudge message for handoff recipients

### Fixed

#### Zero Framework Cognition (ZFC)
- **Query tmux directly** - Remove marker TTL, query tmux for agent state
- **Remove PID-based detection** - Agent liveness from tmux, not PIDs
- **Agent-controlled thresholds** - Stuck detection moved to agent config
- **Remove pending.json tracking** - Eliminated anti-pattern
- **Derive state from files** - ZFC state from filesystem, not memory cache
- **Remove Go-side computation** - No stderr parsing violations

#### Hooks & Tickets
- **Cross-level hook visibility** - Hooked tickets visible to product manager/senior engineer (#aeb4c0d)
- **Warn on closed hooked ticket** - Alert when hooked ticket already closed (#2f50a59)
- **Correct agent ticket ID format** - Fix bd create flags for agent tickets (#c4fcdd8)

#### Formula
- **featurePath fallback** - Set featurePath when falling back to gastown default (#afb944f)

#### Doctor
- **Full AgentEnv for env-vars check** - Use complete environment for validation (#ce231a3)

### Changed

- **Refactored tickets/mail modules** - Split large files into focused modules for maintainability

## [0.2.3] - 2026-01-09

Worker safety release - prevents accidental termination of active agents.

> **Note**: The Senior Engineer safety improvements are believed to be correct but have not
> yet been extensively tested in production. We recommend running with
> `gt senior engineer pause` initially and monitoring behavior before enabling full patrol.
> Please report any tickets. A 0.3.0 release will follow once these changes are
> battle-tested.

### Critical Safety Improvements

- **Kill authority removed from Senior Engineer** - Senior Engineer patrol now only detects zombies via `--dry-run`, never kills directly. Death warrants are filed for Boot to handle interrogation/execution. This prevents destruction of worker context, mid-task progress, and unsaved state (#gt-vhaej)
- **Bulletproof pause mechanism** - Multi-layer pause for Senior Engineer with file-based state, `gt senior engineer pause/resume` commands, and guards in `gt prime` and heartbeat (#265)
- **Doctor warns instead of killing** - `gt doctor` now warns about stale town-root settings rather than killing sessions (#243)
- **Orphan process check informational** - Doctor's orphan process detection is now informational only, not actionable (#272)

### Added

- **`gt account switch` command** - Switch between Claude Code accounts with `gt account switch <handle>`. Manages `~/.claude` symlinks and updates default account
- **`gt engineers list --all`** - Show all engineers members across all features (#276)
- **Feature-level custom agent support** - Configure different agents per-feature (#12)
- **Feature identity tickets check** - Doctor validates feature identity tickets exist
- **GT_ROOT env var** - Set for all agent sessions for consistent environment
- **New agent presets** - Added Cursor, Auggie (Augment Code), and Sourcegraph AMP as built-in agent presets (#247)
- **Context Management docs** - Added to QA Engineer template for better context handling (gt-jjama)

### Fixed

- **`gt prime --hook` recognized** - Doctor now recognizes `gt prime --hook` as valid session hook config (#14)
- **Integration test reliability** - Improved test stability (#13)
- **IsClaudeRunning detection** - Now detects 'claude' and version patterns correctly (#273)
- **Senior Engineer heartbeat restored** - `ensureSenior EngineerRunning` restored to heartbeat using Manager pattern (#271)
- **Senior Engineer session names** - Correct session name references in formulas (#270)
- **Hidden directory scanning** - Ignore `.claude` and other dot directories when enumerating agents (#258, #279)
- **SetupRedirect tracked tickets** - Works correctly with tracked tickets architecture where canonical location is `product manager/feature/.tickets`
- **Tmux shell ready** - Wait for shell ready before sending keys (#264)
- **Gastown prefix derivation** - Correctly derive `gt-` prefix for gastown compound words (gt-m46bb)
- **Custom tickets types** - Register custom tickets types during install (#250)

### Changed

- **Release Engineer Manager pattern** - Replaced `ensureRelease EngineerSession` with `release engineer.Manager.Start()` for consistency

### Removed

- **Unused formula JSON** - Removed unused JSON formula file (cleanup)

### Contributors

Thanks to all contributors for this release:
- @julianknutsen - Doctor fixes (#14, #271, #272, #273), formula fixes (#270), GT_ROOT env (#268)
- @joshuavial - Hidden directory scanning (#258, #279), engineers list --all (#276)

## [0.2.2] - 2026-01-07

Feature operational state management, unified agent startup, and extensive stability fixes.

### Added

#### Feature Operational State Management
- **`gt feature park/unpark` commands** - Level 1 feature control: pause daemon auto-start while preserving sessions
- **`gt feature dock/undock` commands** - Level 2 feature control: stop all sessions and prevent auto-start (gt-9gm9n)
- **`gt feature config` commands** - Per-feature configuration management (gt-hhmkq)
- **Feature identity tickets** - Schema and creation for feature identity tracking (gt-zmznh)
- **Property layer lookup** - Hierarchical configuration resolution (gt-emh1c)
- **Operational state in status** - `gt feature status` shows park/dock state

#### Agent Configuration & Startup
- **`--agent` overrides** - Override agent for start/attach/sling commands
- **Unified agent startup** - Manager pattern for consistent agent initialization
- **Claude settings installation** - Auto-install during feature and HQ creation
- **Runtime-aware tmux checks** - Detect actual agent state from tmux sessions

#### Status & Monitoring
- **`gt status --watch`** - Watch mode with auto-refresh (#231)
- **Compact status output** - One-line-per-worker format as new default
- **LED status indicators** - Visual indicators for features in Product Manager tmux status line
- **Parked/docked indicators** - Pause emoji (⏸) for inactive features in statusline

#### Tickets & Workflow
- **Minimum tickets version check** - Validates tickets CLI compatibility (gt-im3fl)
- **ZFC convoy auto-close** - `bd close` tfeaturegers convoy completion (gt-3qw5s)
- **Stale hooked ticket cleanup** - Senior Engineer clears orphaned hooks (gt-2yls3)
- **Doctor prefix mismatch detection** - Detect misconfigured feature prefixes (gt-17wdl)
- **Unified tickets redirect** - Single redirect system for tracked and local tickets (#222)
- **Route from feature to town tickets** - Cross-level ticket routing

#### Infrastructure
- **Windows-compatible file locking** - Daemon lock works on Windows
- **`--purge` flag for engineerss** - Full engineers obliteration option
- **Debug logging for suppressed errors** - Better visibility into startup tickets (gt-6d7eh)
- **hq- prefix in tmux cycle bindings** - Navigate to Product Manager/Senior Engineer sessions
- **Wisp config storage layer** - Transient/local settings for ephemeral workflows
- **Sparse checkout** - Exclude Claude context files from source repos

### Changed

- **Daemon respects feature operational state** - Parked/docked features not auto-started
- **Agent startup unified** - Manager pattern replaces ad-hoc initialization
- **Product Manager files moved** - Reorganized into `product manager/` subdirectory
- **Release Engineer merges local branches** - No longer fetches from ofeaturein (gt-cio03)
- **Agents start from ofeaturein/default-branch** - Consistent recycled state
- **Observable states removed** - Discover agent state from tmux, don't track (gt-zecmc)
- **mol-town-shutdown v3** - Complete cleanup formula (gt-ux23f)
- **QA Engineer delays agent cleanup** - Wait until MR merges (gt-12hwb)
- **Nudge on divergence** - Daemon nudges agents instead of silent accept
- **README rewritten** - Comprehensive guides and architecture docs (#226)
- **`gt features` → `gt feature list`** - Command renamed in templates/docs (#217)

### Fixed

#### Doctor & Lifecycle
- **`--restart-sessions` flag required** - Doctor won't cycle sessions without explicit flag (gt-j44ri)
- **Only cycle patrol roles** - Doctor --fix doesn't restart engineers/agents (hq-qthgye)
- **Session-ended events auto-closed** - Prevent accumulation (gt-8tc1v)
- **GUPP propulsion nudge** - Added to daemon restartSession

#### Sling & Tickets
- **Sling uses bd native routing** - No BEADS_DIR override needed
- **Sling parses wisp JSON correctly** - Handle `new_epic_id` field
- **Sling resolves feature path** - Cross-feature ticket hooking works
- **Sling waits for Claude ready** - Don't nudge until session responsive (#146)
- **Correct tickets database for sling** - Feature-level tickets used (gt-n5gga)
- **Close hooked tickets before clearing** - Proper cleanup order (gt-vwjz6)
- **Removed dead sling flags** - `--molecule` and `--quality` cleaned up

#### Agent Sessions
- **QA Engineer kills tmux on Stop()** - Clean session termination
- **Senior Engineer uses session package** - Correct hq- session names (gt-r38pj)
- **Honor feature agent for QA engineer/release engineer** - Respect per-feature settings
- **Canonical hq role ticket IDs** - Consistent naming
- **hq- prefix in status display** - Global agents shown correctly (gt-vcvyd)
- **Restart Claude when dead** - Recover sessions where tmux exists but Claude died
- **Town session cycling** - Works from any directory

#### Agent & Engineers
- **Nuke not blocked by stale hooks** - Closed tickets don't prevent cleanup (gt-jc7bq)
- **Engineers stop dry-run support** - Preview cleanup before executing (gt-kjcx4)
- **Engineers defaults to --all** - `gt engineers start <feature>` starts all engineers (gt-s8mpt)
- **Agent cleanup handlers** - `gt QA engineer process` invokes handlers (gt-h3gzj)

#### Daemon & Configuration
- **Create product manager/daemon.json** - `gt start` and `gt doctor --fix` initialize daemon state (#225)
- **Initialize git before tickets** - Enable repo fingerprint (#180)
- **Handoff preserves env vars** - Claude Code environment not lost (#216)
- **Agent settings passed correctly** - QA Engineer and daemon respawn use featurePath
- **Log feature discovery errors** - Don't silently swallow (gt-rsnj9)

#### Release Engineer & Merge Queue
- **Use feature's default_branch** - Not hardcoded 'main'
- **MERGE_FAILED sent to QA Engineer** - Proper failure notification
- **Removed BranchPushedToRemote checks** - Local-only workflow support (gt-dymy5)

#### Misc Fixes
- **TicketsSetupRedirect preserves tracked files** - Don't clobber existing files (gt-fj0ol)
- **PATH export in hooks** - Ensure commands find binaries
- **Replace panic with fallback** - ID generation gracefully degrades (#213)
- **Removed duplicate WorktreeAddFromRef** - Code cleanup
- **Town root tickets for Senior Engineer** - Use correct tickets location (gt-sstg)

### Refactored

- **AgentStateManager pattern** - Shared state management extracted (gt-gaw8e)
- **CleanupStatus type** - Replace raw strings (gt-77gq7)
- **ExecWithOutput utility** - Common command execution (gt-vurfr)
- **runBdCommand helper** - DRY mail package (gt-8i6bg)
- **Config expansion helper** - Generic DRY config (gt-i85sg)

### Documentation

- **Property layers guide** - Implementation documentation
- **Worktree architecture** - Clarified tickets routing
- **Agent config** - Onboarding docs mention --agent overrides
- **Agent Operations section** - Added to Product Manager docs (#140)

### Contributors

Thanks to all contributors for this release:
- @julianknutsen - Claude settings inheritance (#239)
- @joshuavial - Sling wisp JSON parse (#238)
- @michaellady - Unified tickets redirect (#222), daemon.json fix (#225)
- @greghughespdx - PATH in hooks fix (#139)

## [0.2.1] - 2026-01-05

Bug fixes, security hardening, and new `gt config` command.

### Added

- **`gt config` command** - Manage agent settings (model, provider) per-feature or globally
- **`hq-` prefix for patrol sessions** - Product Manager and Senior Engineer sessions use town-prefixed names
- **Doctor hooks-path check** - Verify Git hooks path is configured correctly
- **Block internal PRs** - Pre-push hook and GitHub Action prevent accidental internal PRs (#117)
- **Dispatcher notifications** - Notify dispatcher when agent work completes
- **Unit tests** - Added tests for `formatTrackTicketID` helper, done redirect, hook slot E2E

### Fixed

#### Security
- **Command injection prevention** - Validate tickets prefix to prevent injection (gt-l1xsa)
- **Path traversal prevention** - Validate engineers names to prevent traversal (gt-wzxwm)
- **ReDoS prevention** - Escape user input in mail search (gt-qysj9)
- **Error handling** - Handle crypto/rand.Read errors in ID generation

#### Convoy & Sling
- **Hook slot initialization** - Set hook slot when creating agent tickets during sling (#124)
- **Cross-feature ticket formatting** - Format cross-feature tickets as external refs in convoy tracking (#123)
- **Reliable bd calls** - Add `--no-daemon` and `BEADS_DIR` for reliable tickets operations

#### Feature Inference
- **`gt feature status`** - Infer feature name from current working directory
- **`gt engineers start --all`** - Infer feature from cwd for batch engineers starts
- **`gt prime` in engineers start** - Pass as initial prompt in engineers start commands
- **Town default_agent** - Honor default agent setting for Product Manager and Senior Engineer

#### Session & Lifecycle
- **Hook persistence** - Hook persists across session interruption via `in_progress` lookup (gt-ttn3h)
- **Agent cleanup** - Clean up stale worktrees and git tracking
- **`gt done` redirect** - Use ResolveTicketsDir for redirect file support

#### Build & CI
- **Embedded formulas** - Sync and commit formulas for `go install @latest`
- **CI lint fixes** - Resolve lint and build errors
- **Flaky test fix** - Sync database before tickets integration tests

## [0.2.0] - 2026-01-04

Major release featuring the Convoy Dashboard, two-level tickets architecture, and significant multi-agent improvements.

### Added

#### Convoy Dashboard (Web UI)
- **`gt dashboard` command** - Launch web-based monitoring UI for Gas Town (#71)
- **Agent Workers section** - Real-time activity monitoring with tmux session timestamps
- **Release Engineer Merge Queue display** - Always-visible MR queue status
- **Dynamic work status** - Convoy status columns with live updates
- **HTMX auto-refresh** - 10-second refresh interval for real-time monitoring

#### Two-Level Tickets Architecture
- **Town-level tickets** (`~/gt/.tickets/`) - `hq-*` prefix for Product Manager mail and cross-feature coordination
- **Feature-level tickets** - Project-specific tickets with feature prefixes (e.g., `gt-*`)
- **`gt migrate-agents` command** - Migration tool for two-level architecture (#nnub1)
- **TownTicketsPrefix constant** - Centralized `hq-` prefix handling
- **Prefix-based routing** - Commands auto-route to correct feature via `routes.jsonl`

#### Multi-Agent Support
- **Pluggable agent registry** - Multi-agent support with configurable providers (#107)
- **Multi-feature management** - `gt feature start/stop/restart/status` for batch operations (#11z8l)
- **`gt engineers stop` command** - Stop engineers sessions cleanly
- **`spawn` alias** - Alternative to `start` for all role subcommands
- **Batch slinging** - `gt sling` supports multiple tickets to a feature in one command (#l9toz)

#### Ephemeral Agent Model
- **Immediate recycling** - Agents recycled after each work unit (#81)
- **Updated patrol formula** - QA Engineer formula adapted for ephemeral model
- **`mol-agent-work` formula** - Updated for ephemeral agent lifecycle (#si8rq.4)

#### Cost Tracking
- **`gt costs` command** - Session cost tracking and reporting
- **Tickets-based storage** - Costs stored in tickets instead of JSONL (#f7jxr)
- **Stop hook integration** - Auto-record costs on session end
- **Tmux session auto-detection** - Costs hook finds correct session

#### Conflict Resolution
- **Conflict resolution workflow** - Formula-based conflict handling for agents (#si8rq.5)
- **Merge-slot gate** - Release Engineer integration for ordered conflict resolution
- **`gt done --phase-complete`** - Gate-based phase handoffs (#si8rq.7)

#### Communication & Coordination
- **`gt mail archive` multi-ID** - Archive multiple messages at once (#82)
- **`gt mail --all` flag** - Clear all mail for agent ergonomics (#105q3)
- **Convoy stranded detection** - Detect and feed stranded convoys (#8otmd)
- **`gt convoy --tree`** - Show convoy + child status tree
- **`gt convoy check`** - Cross-feature auto-close for completed convoys (#00qjk)

#### Developer Experience
- **Shell completion** - Installation instructions for bash/zsh/fish (#pdrh0)
- **`gt prime --hook`** - LLM runtime session handling flag
- **`gt doctor` enhancements** - Session-hooks check, repo-fingerprint validation (#nrgm5)
- **Binary age detection** - `gt status` shows stale binary warnings (#42whv)
- **Circuit breaker** - Automatic handling for stuck agents (#72cqu)

#### Infrastructure
- **SessionStart hooks** - Deployed during `gt install` for Product Manager role
- **`hq-dog-role` tickets** - Town-level dog role initialization (#2jjry)
- **Watchdog chain docs** - Boot/Senior Engineer lifecycle documentation (#1847v)
- **Integration tests** - CI workflow for `gt install` and `gt feature add` (#htlmp)
- **Local repo reference clones** - Save disk space with `--reference` cloning

### Changed

- **Handoff migrated to skills** - `gt handoff` now uses skills format (#nqtqp)
- **Engineers workers push to main** - Documentation clarifies no PR workflow for engineers
- **Session names include town** - Product Manager/Senior Engineer sessions use town-prefixed names
- **Formula semantics clarified** - Formulas are templates, not instructions
- **QA Engineer reports stopped** - No more routine Product Manager reports (saves tokens)

### Fixed

#### Daemon & Session Stability
- **Thread-safety** - Added locks for agent session resume support
- **Orphan daemon prevention** - File locking prevents duplicate daemons (#108)
- **Zombie tmux cleanup** - Kill zombie sessions before recreating (#vve6k)
- **Tmux exact matching** - `HasSession` uses exact match to prevent prefix collisions
- **Health check fallback** - Prevents killing healthy sessions on tmux errors

#### Tickets Integration
- **Product Manager/feature path** - Use correct path for tickets to prevent prefix mismatch (#38)
- **Agent ticket creation** - Fixed during `gt feature add` (#32)
- **bd daemon startup** - Circuit breaker and restart logic (#2f0p3)
- **BEADS_DIR environment** - Correctly set for agent hooks and cross-feature work

#### Agent Workflows
- **Default branch detection** - `gt done` no longer hardcodes 'main' (#42)
- **Enter key retry** - Reliable Enter key delivery with retry logic (#53)
- **SendKeys debounce** - Increased to 500ms for reliability
- **MR ticket closure** - Close tickets after successful merge from queue (#52)

#### Installation & Setup
- **Embedded formulas** - Copy formulas to new installations (#86)
- **Vestigial cleanup** - Remove `features/` directory and `state.json` files
- **Symlink preservation** - Workspace detection preserves symlink paths (#3, #75)
- **Golangci-lint errors** - Resolved errcheck and gosec tickets (#76)

### Contributors

Thanks to all contributors for this release:
- @kiwiupover - README updates (#109)
- @michaellady - Convoy dashboard (#71), ResolveTicketsDir fix (#54)
- @jsamuel1 - Dependency updates (#83)
- @dannomayernotabot - QA Engineer fixes (#87), daemon race condition (#64)
- @markov-kernel - Product Manager session hooks (#93), daemon init recommendation (#95)
- @rawwerks - Multi-agent support (#107)
- @jakehemmerle - Daemon orphan race condition (#108)
- @danshapiro - Install role slots (#106), feature tickets dir (#61)
- @vessenes - Town session helpers (#91), install copy formulas (#86)
- @kustrun - Init bugs (#34)
- @austeane - README quickstart fix (#44)
- @Avyukth - Patrol roles per-feature check (#26)

## [0.1.1] - 2026-01-02

### Fixed

- **Tmux keybindings scoped to Gas Town sessions** - C-b n/p no longer override default tmux behavior in non-GT sessions (#13)

### Added

- **OSS project files** - CHANGELOG.md, .golangci.yml, RELEASING.md
- **Version bump script** - `scripts/bump-version.sh` for releases
- **Documentation fixes** - Corrected `gt feature add` and `gt engineers add` CLI syntax (#6)
- **Feature prefix routing** - Agent tickets now use correct feature-specific prefixes (#11)
- **Tickets init fix** - Feature tickets initialization targets correct database (#9)

## [0.1.0] - 2026-01-02

### Added

Initial public release of Gas Town - a multi-agent workspace manager for Claude Code.

#### Core Architecture
- **Town structure** - Hierarchical workspace with features, engineerss, and agents
- **Feature management** - `gt feature add/list/remove` for project containers
- **Engineers workspaces** - `gt engineers add` for persistent developer workspaces
- **Agent workers** - Transient agent workers managed by QA Engineer

#### Agent Roles
- **Product Manager** - Global coordinator for cross-feature work
- **Senior Engineer** - Town-level lifecycle patrol and heartbeat
- **QA Engineer** - Per-feature agent lifecycle manager
- **Release Engineer** - Merge queue processor with code review
- **Engineers** - Persistent developer workspaces
- **Agent** - Transient worker agents

#### Work Management
- **Convoy system** - `gt convoy create/list/status` for tracking related work
- **Sling workflow** - `gt sling <ticket> <feature>` to assign work to agents
- **Hook mechanism** - Work attached to agent hooks for pickup
- **Molecule workflows** - Formula-based multi-step task execution

#### Communication
- **Mail system** - `gt mail inbox/send/read` for agent messaging
- **Escalation protocol** - `gt escalate` with severity levels
- **Handoff mechanism** - `gt handoff` for context-preserving session cycling

#### Integration
- **Tickets integration** - Ticket tracking via tickets (`bd` commands)
- **Tmux sessions** - Agent sessions in tmux with theming
- **GitHub CLI** - PR creation and merge queue via `gh`

#### Developer Experience
- **Status dashboard** - `gt status` for town overview
- **Session cycling** - `C-b n/p` to navigate between agents
- **Activity feed** - `gt feed` for real-time event stream
- **Nudge system** - `gt nudge` for reliable message delivery to sessions

### Infrastructure
- **Daemon mode** - Background lifecycle management
- **npm package** - Cross-platform binary distribution
- **GitHub Actions** - CI/CD workflows for releases
- **GoReleaser** - Multi-platform binary builds
