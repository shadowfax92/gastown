# OpenTelemetry Architecture

## Overview

Gas Town uses OpenTelemetry (OTel) for structured observability of all agent operations. Telemetry is emitted via standard OTLP HTTP to any compatible backend (metrics, logs).

**Backend-agnostic design**: The system emits standard OpenTelemetry Protocol (OTLP) — any OTLP v1.x+ compatible backend can consume it. You are **not obligated** to use VictoriaMetrics/VictoriaLogs; these are simply development defaults.

**Best-effort design**: Telemetry initialization errors are returned but do not affect normal GT operation. The system remains functional even when telemetry is unavailable.

---

## Quick Setup

Set at least one endpoint variable to activate telemetry — both endpoints unset means telemetry is completely disabled (no instrumentation code runs):

```bash
# Full local setup (recommended)
export GT_OTEL_METRICS_URL=http://localhost:8428/opentelemetry/api/v1/push
export GT_OTEL_LOGS_URL=http://localhost:9428/insert/opentelemetry/v1/logs

# Opt-in features
export GT_LOG_BD_OUTPUT=true      # Include bd stdout/stderr in bd.call records
export GT_LOG_AGENT_OUTPUT=true   # Stream Claude conversation turns to logs (PR #2199)
```

**Local backends (Docker):**
```bash
docker run -d -p 8428:8428 victoriametrics/victoria-metrics
docker run -d -p 9428:9428 victoriametrics/victoria-logs
```

**Verify:** `gt prime` should emit a `prime` event visible at `http://localhost:9428/select/vmui`.

---

## Implementation Status

### Core Telemetry (Main ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Core OTel initialization | ✅ Main | `telemetry.Init()`, providers setup |
| Metrics export (counters) | ✅ Main | 18 metric instruments |
| Metrics export (histograms) | ✅ Main | `gastown.bd.duration_ms` histogram |
| Logs export (any OTLP backend) | ✅ Main | OTLP logs exporter |
| Subprocess correlation | ✅ Main | `OTEL_RESOURCE_ATTRIBUTES` via `SetProcessOTELAttrs()` |

### Session Lifecycle (Main ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| **Session lifecycle** | ✅ Main | `session.start`/`session.stop` events (tmux lifecycle) |
| **Agent instantiation** | ❌ Roadmap | `agent.instantiate` event — no `RecordAgentInstantiate` function exists |

### Workflow & Work Events (Main ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Prompt/nudge telemetry | ✅ Main | `prompt.send` and `nudge` events |
| BD operation telemetry | ✅ Main | `bd.call` events (stdout/stderr opt-in via `GT_LOG_BD_OUTPUT=true`) |
| Mail telemetry | ✅ Main | `mail` operations (operation + status only; no message payload) |
| Sling/done telemetry | ✅ Main | `sling` and `done` events |
| GT prime telemetry | ✅ Main | `prime` + `prime.context` events |
| Work context in `prime` | 🔲 PR #2199 | `work_feature`, `work_ticket`, `work_mol` on `prime` events |

### Agent Lifecycle (Main ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Agent lifecycle telemetry | ✅ Main | `agent.spawn`/`agent.remove` |
| Agent state telemetry | ✅ Main | `agent.state_change` events |
| Daemon restart telemetry | ✅ Main | `daemon.restart` events |
| Agent spawn metric | ✅ Main | `gastown.agent.spawns.total` |

### Molecule Lifecycle

| Feature | Status | Notes |
|---------|--------|-------|
| Molecule lifecycle telemetry | ❌ Roadmap | `mol.cook`/`mol.wisp`/`mol.squash`/`mol.burn` — no `RecordMol*` functions exist |
| Ticket creation telemetry | ❌ Roadmap | `ticket.create` — no `RecordTicketCreate` function exists |
| Formula instantiation telemetry | ✅ Main | `formula.instantiate` |
| Convoy telemetry | ✅ Main | `convoy.create` events |

### Agent Events (PR #2199)

| Feature | Status | Notes |
|---------|--------|-------|
| **Agent conversation events** | 🔲 PR #2199 | `agent.event` per conversation turn (text/tool_use/tool_result/thinking) |
| **Token usage tracking** | 🔲 PR #2199 | `agent.usage` per assistant turn (input/output/cache_read/cache_creation) |
| **Cloud session correlation** | 🔲 PR #2199 | `native_session_id` linking Claude to GT telemetry |
| **Agent logging daemon** | 🔲 PR #2199 | `gt agent-log` detached process for JSONL streaming |
| **`run.id` on all events** | 🔲 PR #2199 | `WithRunID`/`RunIDFromCtx`; `addRunID()` injects `run.id` into every log record |

**Activation (PR #2199)**: Requires `GT_LOG_AGENT_OUTPUT=true` AND `GT_OTEL_LOGS_URL` set.

---

## Roadmap

### P0 — Critical (blocking accurate attribution)

**Work context injection at `gt prime`** — implemented in PR #2199

Agents are **generic agents** — they have no fixed feature. `GT_RIG` at session start reflects an allocation feature or is empty, which is meaningless for attributing work. The actual work context (which feature, ticket, and molecule a agent is processing) is only known at each `gt prime` invocation.

A single agent session goes through multiple `gt prime` cycles, each on a potentially different feature and ticket:

```
session start → feature="" (generic, no work yet)
gt prime #1   → work_feature="gastown", work_ticket="sg-05iq", work_mol="mol-agent-work"
  bd.call, mail, sling, done  ← carry work context from prime #1
gt prime #2   → work_feature="sfgastown", work_ticket="sg-g8vs", work_mol="mol-agent-work"
  bd.call, mail, sling, done  ← carry work context from prime #2
```

Fix: at each `gt prime`, inject `GT_WORK_RIG`, `GT_WORK_BEAD`, `GT_WORK_MOL` into the **tmux session environment** (via `SetEnvironment`), not just emit them as log attributes. This ensures all subsequent subprocesses (`bd`, mail, agent logging) inherit the current work context automatically until the next prime overwrites it.

New attributes emitted on the `prime` event and carried by all events until the next prime:

| Attribute | Type | Description |
|---|---|---|
| `work_feature` | string | feature whose ticket is on the hook |
| `work_ticket` | string | ticket ID currently hooked |
| `work_mol` | string | molecule ID if the ticket is a molecule step; empty otherwise |

---

### P1 — High value

**Token cost metric (`gastown.token.cost_usd`)**

Compute dollar cost per run from token counts using Claude model pricing. Emit as a Gauge metric at session end. Enables per-feature and per-ticket cost dashboards.

New metric: `gastown.token.cost_usd{feature, role, agent_type}` — accumulated cost per session.
New event attribute on `agent.usage`: `cost_usd` — cost of the current turn.

---

**Go runtime/process metrics (low effort)**

The OTel Go SDK has a contrib package (`go.opentelemetry.io/contrib/instrumentation/runtime`) that auto-emits goroutine count, GC pause duration, heap usage, and memory allocations. Activation is ~5 lines of code in `telemetry.Init()`.

New metrics: `process.runtime.go.goroutines`, `process.runtime.go.gc.pause_ns`, `process.runtime.go.mem.heap_alloc_bytes`

---

**Release Engineer queue telemetry**

The Release Engineer's merge queue is a central health indicator but currently completely dark to observability. Expose:

| Metric | Type | Description |
|--------|------|-------------|
| `gastown.release engineer.queue_depth` | Gauge | pending items in merge queue |
| `gastown.release engineer.item_age_ms` | Histogram | age of oldest item in queue |
| `gastown.release engineer.dispatch_latency_ms` | Histogram | time between enqueue and dispatch |

New log event: `release engineer.dispatch` with `ticket_id`, `queue_depth`, `wait_ms`, `status`.

---

**Distributed Traces (OTel Traces SDK)**

Currently the waterfall relies on `run.id` as a manual correlation key across flat log records. Replacing this with proper OTel Traces would enable:

- Visual waterfall in Jaeger / Grafana Tempo
- Automatic parent → child span attribution (no manual run.id joins)
- P95/P99 latency per operation derived from spans, not histograms

Architecture: each agent session spawn creates a **root span** (`gt.session`). Child spans are created for `bd.call`, `mail`, `sling`, `done`. The `run.id` becomes the trace ID. `GT_RUN` propagation becomes W3C `traceparent` header injection.

This is a significant effort (requires `go.opentelemetry.io/otel/trace` + tracer provider + exporter) but would be the single highest-impact observability improvement.

---

### P2 — Medium value

**Scheduler dispatch telemetry**

Expose the capacity-controlled dispatch cycle:

```mermaid
flowchart TD
    A[Query Pending<br/>Readiness Filter] --> B{Capacity Available?}
    B -->|Yes| C[Plan Dispatch<br/>ToDispatch array]
    B -->|No| D[Skip All<br/>Reason: capacity]
    C --> E[Execute Dispatch<br/>gt sling / gt prime]
    E --> F{Dispatch Result}
    F -->|Success| G[Close Sling Context<br/>OnSuccess]
    F -->|Failure| H[Retry / Quarantine<br/>OnFailure]
    G --> I[Log: scheduler.dispatch_cycle<br/>dispatched++, status=ok]
    H --> I
    D --> I
    I --> J[Log: scheduler.queue_depth<br/>current pending count]
    J --> K[Log: scheduler.capacity_usage<br/>free/used slots]
```

New metrics: `scheduler.dispatch_cycle` (dispatched/failed/skipped counts), `scheduler.queue_depth` (histogram), `scheduler.capacity_usage` (gauge).

---

**`done` event enrichment**

Currently `done` carries only `exit_type` and `status`. Adding work context enables per-feature completion analysis:

New attributes: `feature`, `ticket_id`, `time_to_complete_ms` (wall time from session start to done).

---

**QA Engineer patrol cycle telemetry**

Each QA engineer patrol cycle should emit: duration, stale sessions detected, restarts tfeaturegered. Enables trend analysis on QA engineer health.

New event: `QA engineer.patrol` with `duration_ms`, `stale_count`, `restart_count`, `status`.

---

**Dolt health metrics**

Dolt tickets are only detected at spawn time today. Exposing health metrics continuously:

New metrics: `gastown.dolt.connections`, `gastown.dolt.query_duration_ms` (histogram), `gastown.dolt.replication_lag_ms`.

---

### P3 — Nice to have

| Item | Description |
|------|-------------|
| **Senior Engineer watchdog telemetry** | State machine transitions in the senior engineer watchdog chain |
| **Engineers session tracking** | Engineers session cycle events: start, push, done, idle |
| **Git operation telemetry** | Track clone, checkout, fetch duration per agent session |
| **OTel W3C Baggage** | Replace `GT_RUN` env var propagation with W3C Baggage for standard cross-process context |
| **Retry pattern detection** | Alert when a agent's error rate exceeds threshold across runs |

---

## Components

### 1. Initialization (`internal/telemetry/telemetry.go`)

The `telemetry.Init()` function sets up OTel providers on process startup:

```go
provider, err := telemetry.Init(ctx, "gastown", version)
if err != nil {
    // Log and continue — telemetry is best-effort
}
defer provider.Shutdown(ctx)
```

**Exact signature**: `func Init(ctx context.Context, serviceName, serviceVersion string) (*Provider, error)`

**Providers:**
- **Metrics**: Any OTLP-compatible metrics backend via `otlpmetrichttp` exporter
- **Logs**: Any OTLP-compatible logs backend via `otlploghttp` exporter

**Default endpoints** (when GT_OTEL_* variables are not set):
- Metrics: `http://localhost:8428/opentelemetry/api/v1/push`
- Logs: `http://localhost:9428/insert/opentelemetry/v1/logs`

> **Note**: These defaults target VictoriaMetrics/VictoriaLogs for local development convenience. Gas Town uses standard OTLP — you can override endpoints to use any OTLP v1.x+ compatible backend (Prometheus, Grafana Mimir, Datadog, New Relic, Grafana Cloud, Loki, OpenTelemetry Collector, etc.).

**OTLP Compatibility**:
- Uses standard OpenTelemetry Protocol (OTLP) over HTTP
- Protobuf encoding (VictoriaMetrics, Prometheus, and others accept this)
- Compatible with any backend that supports OTLP v1.x+

**Resource attributes** (set at init time by the OTel SDK):

| OTel attribute | Source |
|---|---|
| `service.name` | `"gastown"` (hardcoded at call site) |
| `service.version` | GT binary version |
| `host.name`, `host.arch` | `resource.WithHost()` — OTel SDK reads system hostname |
| `os.type`, `os.version`, `os.description` | `resource.WithOS()` — OTel SDK reads OS info |

**Custom resource attributes** (via `OTEL_RESOURCE_ATTRIBUTES` env var, set by `SetProcessOTELAttrs()`):

| Attribute | Source env var | Notes |
|---|---|---|
| `gt.role` | `GT_ROLE` | Agent role (e.g. `gastown/agents/Toast`) |
| `gt.feature` | `GT_RIG` | Feature name |
| `gt.actor` | `BD_ACTOR` | BD actor/identity |
| `gt.agent` | `GT_POLECAT` or `GT_CREW` | Agent name |
| `gt.session` | `GT_SESSION` | Tmux session name — **PR #2199** |
| `gt.run_id` | `GT_RUN` | Run UUID — **PR #2199** |
| `gt.work_feature` | `GT_WORK_RIG` | Work feature at last prime — **PR #2199** |
| `gt.work_ticket` | `GT_WORK_BEAD` | Hooked ticket at last prime — **PR #2199** |
| `gt.work_mol` | `GT_WORK_MOL` | Molecule step at last prime — **PR #2199** |

---

### 2. Recording Layer (`internal/telemetry/recorder.go`)

The recorder provides type-safe functions for emitting all GT telemetry events. Each function emits:

1. **OTel metric counter** (→ VictoriaMetrics, aggregated)
2. **OTel log record** (→ VictoriaLogs, full detail)

> **`run.id` on log records**: On main, log records do not carry `run.id`. After PR #2199
> merges, `addRunID(ctx, &r)` will be called on every log record, injecting `run.id` from
> context (set via `WithRunID`) or from the `GT_RUN` env var.

#### Recording Pattern

```go
func RecordSomething(ctx context.Context, args ..., err error) {
    initInstruments() // Lazy-load OTel instruments
    status := statusStr(err) // "ok" or "error"
    inst.somethingTotal.Add(ctx, 1, metric.WithAttributes(
        attribute.String("status", status),
        attribute.String("label", value),
    ))
    emit(ctx, "something", severity(err),
        otellog.String("key1", value1),
        otellog.String("key2", value2),
        otellog.String("status", status),
        errKV(err), // Empty string or error message
    )
}
```

#### Instrument Types

| Type | Description | Example |
|------|-------------|---------|
| Counters | Total counts per attribute combination | `gastown.agent.spawns.total{status="ok"}` |
| Histograms | Distribution of measurements (latency, duration) | `gastown.bd.duration_ms` |
| Log records | Structured events with full payload | `prime`, `mail`, `agent.event` (PR #2199) |

---

### 3. Context Propagation

#### Subprocess Integration (`internal/telemetry/subprocess.go`)

Two mechanisms ensure subprocess telemetry is correlated:

**1. Process-level inheritance** (`SetProcessOTELAttrs`):
- Called once at GT startup
- Sets `OTEL_RESOURCE_ATTRIBUTES` in process environment
- All `exec.Command()` subprocesses inherit these env vars automatically

**2. Manual injection** (`OTELEnvForSubprocess`):
- For callers building `cmd.Env` explicitly (overriding `os.Environ`)
- Returns pre-built env slice with:
  - `OTEL_RESOURCE_ATTRIBUTES` (GT context attributes)
  - `BD_OTEL_METRICS_URL` (mirrors `GT_OTEL_METRICS_URL`)
  - `BD_OTEL_LOGS_URL` (mirrors `GT_OTEL_LOGS_URL`)
  - `GT_RUN` (run ID for correlation — **PR #2199**)

#### Run ID Correlation (PR #2199)

On main, there is no run-level correlation key in log records. PR #2199 adds:

- `GT_RUN` env var — UUID generated at agent spawn
- `gt.run_id` in `OTEL_RESOURCE_ATTRIBUTES` — carried by all subprocesses
- `WithRunID(ctx, runID)` / `RunIDFromCtx(ctx)` — Go context carrier
- `addRunID(ctx, &record)` — called in every emit, injects `run.id` into log record

**Query example (after PR #2199):** Retrieve all events for a single session run
```logsql
run.id:uuid-1234
```

---

### 4. Agent Logging (PR #2199)

> **Status: PR #2199 (`otel-p0-work-context`)** — not on main. The files below are added in PR #2199 and do not exist at `ofeaturein/main`.

**Opt-in feature**: `GT_LOG_AGENT_OUTPUT=true` streams native AI agent JSONL to VictoriaLogs.

**How it works:**
1. `ActivateAgentLogging()` (`internal/session/agent_logging_unix.go`) spawns detached `gt agent-log` process
2. Uses `Setsid` so it survives parent process exit
3. PID file at `/tmp/gt-agentlog-<session>.pid` ensures single instance
4. `--since=now-60s` filters to only this session's Claude instance
5. `gt agent-log` (`internal/cmd/agent_log.go`) tails JSONL files and emits `RecordAgentEvent` for each
6. `internal/agentlog/` package — adapters for claudecode and opencode JSONL formats

**Events emitted:**
- `agent.event`: One record per conversation turn (text, tool_use, tool_result, thinking)
- `agent.usage`: Token usage per assistant turn (input, output, cache stats)

**Session name in telemetry:**
- `session`: Tmux session name (e.g., `gt-gastown-Toast`)
- `native_session_id`: Claude Code JSONL filename UUID

---

## Environment Variables

### GT-Level Variables

| Variable | Set by | Description |
|----------|---------|-------------|
| `GT_OTEL_METRICS_URL` | Operator | OTLP metrics endpoint (default: localhost:8428) |
| `GT_OTEL_LOGS_URL` | Operator | OTLP logs endpoint (default: localhost:9428) |
| `GT_LOG_BD_OUTPUT` | Operator | **Opt-in**: Include bd stdout/stderr in `bd.call` records |
| `GT_LOG_AGENT_OUTPUT` | Operator | **Opt-in (PR #2199)**: Stream Claude conversation events |

### Session Context Variables (Set by `session.StartSession`)

| Variable | Values / Format | Description |
|----------|-----------------|-------------|
| `GT_ROLE` | `<feature>/agents/<name>` · `product manager` · `tickets/QA engineer` | Agent role for identity parsing |
| `GT_RIG` | `gastown`, `tickets` | Feature name (empty for town-level agents) |
| `GT_POLECAT` | `Toast`, `Shadow`, `Furiosa` | Agent name (feature-specific) |
| `GT_CREW` | `max`, `jane` | Engineers member name |
| `GT_SESSION` | `gt-gastown-Toast`, `hq-product manager` | Tmux session name |
| `GT_AGENT` | Preset names: `claude`, `gemini`, `codex`, `cursor`, `copilot`, `opencode`, … | Agent override (if specified) |
| `GT_RUN` | UUID v4 | **PR #2199** — Run identifier, primary waterfall correlation key |
| `GT_ROOT` | `/Users/pa/gt` | Town root path |
| `CLAUDE_CONFIG_DIR` | `~/gt/.claude` | Runtime config directory (for agent overrides) |
| `BD_ACTOR` | `<feature>/agents/<name>` | BD actor identity (git author) |
| `GIT_AUTHOR_NAME` | Agent name | Git author name |
| `GIT_CEILING_DIRECTORIES` | Town root | Git ceiling (prevents repo traversal) |

---

## Event Types

See [OTel Data Model](otel-data-model.md) for the complete event schema, attribute tables, and metric reference.

---

## Monitoring Gaps

### Currently Monitored ✅

| Area | Coverage |
|-------|----------|
| Agent session lifecycle | Full (start, stop, respawn) |
| Tmux prompts/nudges | Full (content length, debouncing — content not logged) |
| BD operations | Full (all BD CLI calls) |
| Mail operations | Full (operation + status; message payload not recorded) |
| Agent lifecycle | Full (spawn, remove, state changes) |
| Formula instantiation | Full (formula name, ticket ID) |
| Convoy tracking | Full (auto-convoy creation) |
| Daemon restarts | Full (QA engineer/senior engineer-initiated) |
| GT prime operations | Full (with formula context) |
| Agent conversation events | 🔲 PR #2199 — requires `GT_LOG_AGENT_OUTPUT=true` |
| Token usage | 🔲 PR #2199 — requires `GT_LOG_AGENT_OUTPUT=true` |

### Not Currently Monitored ❌

| Area | Notes | Operational Impact |
|-------|-------|-------------------|
| **Generic agent work context** | **Critical gap** — see [Generic Agent Work Context](#generic-agent-work-context-️) below | No work attribution on any event between two `gt prime` calls; token costs unattributable |
| **Agent instantiation** | No `agent.instantiate` event (roadmap) | Cannot anchor a run to a specific agent spawn |
| **Molecule lifecycle** | No `mol.cook/wisp/squash/burn` events (roadmap) | Cannot observe formula-to-wisp pipeline |
| **Ticket creation** | No `ticket.create` event (roadmap) | Cannot trace child ticket graph during molecule instantiation |
| Dolt server health | Handled by pre-spawn health checks, but not exposed to telemetry | Database tickets only detected at spawn time; no real-time health monitoring |
| Release Engineer merge queue | Internal operation, not surfaced via telemetry | Cannot monitor merge backlog or detect bottlenecks |
| Scheduler dispatch logs | Capacity-controlled dispatch cycles not exposed to telemetry | Cannot track dispatch efficiency, queue depth, or capacity utilization |
| Engineers worktree operations | No explicit tracking of engineers session cycles | Cannot track engineers efficiency or session patterns |
| Git operations (clone, checkout, etc.) | Git author/name is set, but individual operations not tracked | Cannot diagnose git-related failures or track repository operations |
| Resource usage (CPU, memory, disk) | Not instrumented — consider OTel process metrics | Cannot detect resource exhaustion or capacity planning needs |
| Network activity | Not instrumented (Claude API calls logged by agent, but external traffic not) | Cannot diagnose network tickets or detect unusual external connections |
| Cross-feature worktree operations | Worktrees are created/managed but operations not tracked | Cannot correlate worktree lifecycle with work items |
| QA Engineer monitoring loops | Health checks happen but not exposed to observability | Cannot monitor QA engineer health trends or detect degraded performance |
| Senior Engineer watchdog chain | Internal state machine, not currently exposed to observability | Cannot track senior engineer health or detect daemon failures |

---

## Generic Agent Work Context ⚠️

**Critical gap**: Agents are generic agents with no fixed feature. `gt.feature` in resource attributes reflects the allocation feature (or is empty), which has no bearing on the actual work being done. Work context is only determined at each `gt prime` invocation — and changes with every new work assignment.

This means all events emitted between two `gt prime` calls (`bd.call`, `mail`, `sling`, `done`) have no work attribution today. You cannot answer "which ticket did this `bd.call` serve?" from current telemetry.

**Impact**:
- `gt.feature` resource attribute is the allocation feature, not the work feature — misleading for multi-feature agents
- Token usage (`agent.usage`, PR #2199) cannot be attributed to a specific ticket, feature, or molecule
- `bd.call`, `mail`, `done` events carry no indication of which work item tfeaturegered them

**Proposed solution** (see [Roadmap P0](#p0--critical-blocking-accurate-attribution)):
- At each `gt prime`, write `GT_WORK_RIG`, `GT_WORK_BEAD`, `GT_WORK_MOL` into the tmux session via `SetEnvironment` — all subprocesses inherit automatically
- Emit `work_feature`, `work_ticket`, `work_mol` on the `prime` event
- All events emitted after a `prime` (until the next one) carry the current work context via the inherited env vars

---

## Data Model

See [OTel Data Model](otel-data-model.md) for complete schema of all events.

> The data model is independent of backend — any OTLP-compatible consumer can parse and query these events.

---

## Queries

### Metrics (Any OTLP-compatible backend)

> These examples use PromQL/MetricsQL syntax, as supported by VictoriaMetrics, Prometheus, Grafana Mimir, etc.

**Total counts by status:**
```promql
sum(rate(gastown_agent_spawns_total[5m])) by (status)
sum(rate(gastown_bd_calls_total[5m])) by (subcommand, status)
```

> **Naming note**: OTel SDK metric names use dot notation (e.g. `gastown.bd.calls.total`).
> Prometheus-compatible backends export these with underscores (e.g. `gastown_bd_calls_total`).
> Use underscore form in PromQL queries.

**Latency distributions:**
```promql
histogram_quantile(0.5, rate(gastown_bd_duration_ms_bucket[5m])) by (subcommand)
histogram_quantile(0.95, rate(gastown_bd_duration_ms_bucket[5m])) by (subcommand)
histogram_quantile(0.99, rate(gastown_bd_duration_ms_bucket[5m])) by (subcommand)
```

**Session activity:**
```promql
sum(increase(gastown_session_starts_total[1h])) by (role)
sum(increase(gastown_done_total[1h])) by (exit_type)
```

### VictoriaLogs (Structured Logs)

**Find all events from a agent:**
```logsql
gt.agent:Toast
```

**Error analysis:**
```logsql
status:error
_msg:bd.call AND status:error
_msg:session.stop AND status:error
```

**Agent lifecycle:**
```logsql
_msg:agent.spawn
_msg:agent.remove
_msg:agent.state_change AND new_state:working
```

### Debugging Examples

**Track a agent working across multiple features:**
```logsql
gt.agent:Toast
```
Shows all events from agent Toast, regardless of feature assignment.

**Identify sessions with high error rates:**
```logsql
_msg:bd.call AND status:error
```

**Find all events for a run (after PR #2199):**
```logsql
run.id:uuid-1234
```

---

## Related Documentation

- [OTel Data Model](otel-data-model.md) — Complete event schema
- [Agent Lifecycle](../../concepts/agent-lifecycle.md) — Persistent agent model
- [Overview](../../overview.md) — Role taxonomy and architecture
- [Reference](../../reference.md) — Environment variables and commands

## Backends Compatible with OTLP

| Backend | Notes |
|---------|-------|
| **VictoriaMetrics** | Default for metrics (localhost:8428) — open source. Override with `GT_OTEL_METRICS_URL` to use any OTLP-compatible backend. |
| **VictoriaLogs** | Default for logs (localhost:9428) — open source. Override with `GT_OTEL_LOGS_URL` to use any OTLP-compatible backend. |
| **Prometheus** | Supports OTLP via remote_write receiver — open source |
| **Grafana Mimir** | Supports OTLP via write endpoint — open source |
| **Loki** | Requires OTLP bridge (Loki uses different format) — open source |
| **OpenTelemetry Collector** | Universal forwarder to any backend (recommended for production) — open source |

**Production Recommendation**: For production deployments, consider using the OpenTelemetry Collector as a sidecar. The Collector provides:
- Single agent for all telemetry
- Advanced processing and batching
- Support for multiple backends simultaneously
- Better resource efficiency than per-process exporters

---

## Appendix: Source Reference Audit

Audited against `ofeaturein/main` @ `2d8d71ee35fafda3bbdf353683692bfcc9165476`

### Initialization (`internal/telemetry/telemetry.go`)

| Claim | Source |
|-------|--------|
| `func Init(ctx context.Context, serviceName, serviceVersion string) (*Provider, error)` | `telemetry.go:104` |
| `func (p *Provider) Shutdown(ctx context.Context) error` | `telemetry.go:68` |
| `EnvMetricsURL = "GT_OTEL_METRICS_URL"` | `telemetry.go:36` |
| `EnvLogsURL = "GT_OTEL_LOGS_URL"` | `telemetry.go:39` |
| `DefaultMetricsURL = "http://localhost:8428/opentelemetry/api/v1/push"` | `telemetry.go:42` |
| `DefaultLogsURL = "http://localhost:9428/insert/opentelemetry/v1/logs"` | `telemetry.go:45` |
| `semconv.ServiceName(serviceName)` — resource attr | `telemetry.go:129` |
| `semconv.ServiceVersion(serviceVersion)` — resource attr | `telemetry.go:130` |
| `resource.WithHost()` — produces `host.name`, `host.arch` | `telemetry.go:132` |
| `resource.WithOS()` — produces `os.type`, `os.version`, `os.description` | `telemetry.go:133` |
| Both endpoints unset → telemetry disabled (no providers created) | `telemetry.go:115–120` |

### Subprocess correlation (`internal/telemetry/subprocess.go`)

| Claim | Source |
|-------|--------|
| `func buildGTResourceAttrs() string` | `subprocess.go:11` |
| `GT_ROLE` → `gt.role` | `subprocess.go:13` |
| `GT_RIG` → `gt.feature` | `subprocess.go:16` |
| `BD_ACTOR` → `gt.actor` | `subprocess.go:19` |
| `GT_POLECAT` → `gt.agent` | `subprocess.go:23` |
| `GT_CREW` → `gt.agent` (fallback) | `subprocess.go:25` |
| `func SetProcessOTELAttrs()` | `subprocess.go:42` |
| Sets `OTEL_RESOURCE_ATTRIBUTES` | `subprocess.go:48` |
| Sets `BD_OTEL_METRICS_URL` | `subprocess.go:52` |
| Sets `BD_OTEL_LOGS_URL` | `subprocess.go:54` |
| `func OTELEnvForSubprocess() []string` | `subprocess.go:66` |

### Recording (`internal/telemetry/recorder.go`)

| Claim | Source |
|-------|--------|
| `func emit(ctx, body, sev, attrs...)` | `recorder.go:133` |
| `initInstruments()` — all instruments initialized here | `recorder.go:59` |
| `GT_LOG_BD_OUTPUT` gates stdout/stderr logging | `recorder.go:208` |
| `RecordBDCall` / `bd.call` event | `recorder.go:187` |
| `RecordSessionStart` / `session.start` event | `recorder.go:218` |
| `RecordSessionStop` / `session.stop` event | `recorder.go:236` |
| `RecordPromptSend` / `prompt.send` event — `keys_len` only, content not logged | `recorder.go:250` |
| `RecordPaneRead` / `pane.read` event | `recorder.go:266` |
| `RecordPrime` / `prime` event | `recorder.go:282` |
| `RecordPrimeContext` / `prime.context` event | `recorder.go:305` |
| `RecordAgentStateChange` / `agent.state_change` — `has_hook_ticket` bool | `recorder.go:318` |
| `RecordAgentSpawn` / `agent.spawn` event | `recorder.go:338` |
| `RecordAgentRemove` / `agent.remove` event | `recorder.go:352` |
| `RecordSling` / `sling` event | `recorder.go:366` |
| `RecordMail` / `mail` event — `operation`, `status`, `error` only | `recorder.go:381` |
| `RecordNudge` / `nudge` event | `recorder.go:398` |
| `RecordDone` / `done` event | `recorder.go:413` |
| `RecordDaemonRestart` / `daemon.restart` event | `recorder.go:431` |
| `RecordFormulaInstantiate` / `formula.instantiate` event | `recorder.go:442` |
| `RecordConvoyCreate` / `convoy.create` event | `recorder.go:460` |
| `RecordPaneOutput` / `pane.output` event | `recorder.go:477` |

### Absent functions and features (confirmed by grep on `ofeaturein/main`)

| Claim | Verification |
|-------|-------------|
| `RecordAgentInstantiate` / `agent.instantiate` — does not exist | `grep -r "RecordAgentInstantiate\|agent\.instantiate" internal/ → zero matches` |
| `RecordMolCook` / `mol.cook` etc. — do not exist | `grep -r "RecordMol\|mol\.cook\|mol\.wisp\|mol\.squash\|mol\.burn" internal/ → zero matches` |
| `RecordTicketCreate` / `ticket.create` — does not exist | `grep -r "RecordTicketCreate\|ticket\.create" internal/ → zero matches` |
| `WithRunID` / `RunIDFromCtx` — do not exist on main | `grep -r "WithRunID\|RunIDFromCtx" internal/telemetry/ → zero matches` |
| `GT_RUN` — does not exist on main | `grep -r "GT_RUN" internal/ → zero matches` |
| `GT_LOG_AGENT_OUTPUT` — does not exist on main | `grep -r "GT_LOG_AGENT_OUTPUT" . → zero matches` |
| `gt.session` / `gt.run_id` in resource attrs — not in subprocess.go on main | confirmed: `subprocess.go` has only `gt.role`, `gt.feature`, `gt.actor`, `gt.agent` |
| `agent_logging_unix.go` — does not exist on main | `find internal/session/ -name "agent_logging*" → zero results` |
| `agent_log.go` — does not exist on main | `find internal/cmd/ -name "agent_log*" → zero results` |
| `telemetry.IsActive()` — does not exist on main | `grep -r "IsActive" internal/telemetry/ → zero matches` |

### PromQL naming convention

OTel SDK uses dot notation; Prometheus-compatible backends export with underscores.

| SDK name | PromQL / MetricsQL name |
|----------|------------------------|
| `gastown.bd.calls.total` | `gastown_bd_calls_total` |
| `gastown.bd.duration_ms` | `gastown_bd_duration_ms_bucket` / `_sum` / `_count` |
| `gastown.agent.spawns.total` | `gastown_agent_spawns_total` |
| `gastown.session.starts.total` | `gastown_session_starts_total` |
| `gastown.done.total` | `gastown_done_total` |

### PR #2199 additions (commit `8b88de15`, not yet on main)

| Claim | Source |
|-------|--------|
| `RecordAgentEvent` / `agent.event` | added in `8b88de15` |
| `RecordAgentTokenUsage` / `agent.usage` | added in `8b88de15` |
| `gastown.agent.events.total` Counter | added in `8b88de15` |
| `WithRunID` / `RunIDFromCtx` / `addRunID` | added in `8b88de15` |
| `gt.session`, `gt.run_id`, `gt.work_*` in resource attrs | `subprocess.go` updated in `8b88de15` |
| `GT_RUN` propagation to subprocesses | `subprocess.go` updated in `8b88de15` |
| `injectWorkContext` / `setTmuxWorkContext` in `prime.go` | added in `8b88de15` |
| `internal/agentlog/` package | new in `8b88de15` |
| `internal/cmd/agent_log.go` | new in `8b88de15` |
| `internal/session/agent_logging_unix.go` | new in `8b88de15` |
| `GT_LOG_AGENT_OUTPUT` env var | new in `8b88de15` |
| `telemetry.IsActive()` | added in `8b88de15` |
