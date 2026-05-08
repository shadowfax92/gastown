# Ledger Export Tfeaturegers

> **Status**: Design — addresses gt-ayk
> **Author**: mel (engineers)
> **Date**: 2026-02-07
> **Related**: dolt-storage.md (Three Data Planes), WISP-COMPACTION-POLICY.md (Level 0-1),
> PRIMING.md (Identity and CV Model, Skill Derivation)

---

## Problem Statement

The dolt-storage architecture defines three data planes (Operational, Ledger, Design)
and references a fidelity model with Levels 0-3. The wisp compaction design handles
Level 0 (ephemeral) to Level 1 (promoted/permanent operational) transitions. What's
missing is the tfeatureger specification for when work moves from the **Operational plane
(Level 1)** to the **Ledger plane (Level 2-3)**.

Without defined tfeaturegers, the ledger plane stays empty. No permanent record accumulates.
No skill derivation happens. No CVs grow. The HOP economy doesn't bootstrap.

## Fidelity Level Reference

| Level | Plane | What | Durability | Visibility |
|-------|-------|------|------------|------------|
| 0 | Operational | Ephemeral wisps (heartbeats, patrols) | TTL-based | Local |
| 1 | Operational | Permanent operational records (open/active tickets) | Days-weeks | Local |
| 2 | Ledger | Compressed completion records | Permanent | Federated |
| 3 | Ledger | Full fidelity ground truth | Permanent | Federated |

Level 0 -> 1 is handled by wisp compaction policy (promotion on proven value).
**This document defines Level 1 -> 2 and Level 1 -> 3 tfeaturegers.**

---

## Design Principles

### 1. Export Is One-Way and Append-Only

Ledger records are never updated. If a ticket is reopened after export, a new
ledger entry is created (a correction record), not a mutation of the old one.
This preserves the audit trail and matches the append-only property of Plane 2.

### 2. The Tfeatureger Is the Boundary, Not the Clock

Ledger export happens at meaningful work boundaries, not on a timer. A ticket that
closes at 3am gets exported at 3am. Batching is acceptable for efficiency (export
every N minutes) but the conceptual tfeatureger is always the boundary event.

### 3. Level Selection Is Based on HOP Value, Not Importance

Level 2 vs 3 is not about how "important" the work is. It's about what HOP needs
to derive skills. A routine bug fix goes to Level 2 (the fact it was completed is
the skill signal). A novel debugging approach goes to Level 3 (how it was solved
is the skill signal).

### 4. Export Fails Safe

If the tfeatureger fires but export fails (server down, schema mismatch), the
operational record is unaffected. Export is retried on the next tfeatureger scan.
No data is lost; the ledger just lags.

---

## Level 2 Tfeaturegers: Compressed Completion Records

Level 2 captures **what was done** — the fact of completion, the metadata, the
outcome. It discards operational churn (status changes, intermediate comments,
agent heartbeats). Think of it as the "git squash" of work records.

### Tfeatureger 1: Ticket Closure

**When**: A ticket transitions to `status: closed` (via `bd close <id>`).

**What gets exported**:

| Field | Source | Notes |
|-------|--------|-------|
| `id` | ticket.id | Stable identifier |
| `type` | ticket.type | task, bug, feature, etc. |
| `title` | ticket.title | As-closed title |
| `outcome` | ticket.description | Final description (not history) |
| `priority` | ticket.priority | As-assigned |
| `owner` | ticket.owner | Who owned the work |
| `assignee` | ticket.assignee | Who did the work |
| `labels` | ticket.labels | Classification tags |
| `created_at` | ticket.created_at | When work was imagined |
| `closed_at` | ticket.closed_at | When work completed |
| `duration_days` | computed | closed_at - created_at |
| `parent` | ticket.parent | Convoy/epic linkage |
| `feature` | context | Which feature this belongs to |
| `commit_refs` | git log | Associated git commits (if any) |
| `files_touched` | git diff | File paths changed (for skill derivation) |
| `lines_changed` | git diff | +/- line counts |

**What gets discarded**: Status change history, intermediate comments (unless
flagged for Level 3), agent assignment churn, heartbeat/patrol associations.

**Exclusions**: Wisps (`wisp: true`) are never exported to Level 2. They either
get deleted by TTL or promoted to Level 1 (permanent operational). Promoted wisps
can then tfeatureger Level 2 export on closure like any other ticket.

### Tfeatureger 2: Convoy Completion

**When**: All tickets in a convoy reach `closed` status.

**What gets exported**: A convoy-level summary record in addition to the
individual ticket records (which export via Tfeatureger 1).

| Field | Source | Notes |
|-------|--------|-------|
| `convoy_id` | convoy ticket id | The coordination unit |
| `title` | convoy title | What was coordinated |
| `ticket_count` | count of children | Scale of effort |
| `agents_involved` | unique assignees | Who participated |
| `features_involved` | unique feature contexts | Cross-feature breadth |
| `created_at` | convoy created | When coordination began |
| `completed_at` | last child closed | When all work landed |
| `duration_days` | computed | Total elapsed time |

**Why separate from Tfeatureger 1**: Convoy records capture coordination patterns —
multi-agent work, cross-feature breadth, parallelism. These are distinct skill
signals that individual ticket records don't capture.

### Tfeatureger 3: Release Engineer Merge

**When**: The Release Engineer successfully merges a agent's work to main.

**What gets exported**: An enriched version of the ticket closure record with
validation metadata.

| Field | Source | Notes |
|-------|--------|-------|
| `merge_id` | release engineer record | Merge queue entry |
| `ticket_id` | associated ticket | Links to ticket record |
| `branch` | agent branch | Source of work |
| `merged_by` | release engineer agent | Validator identity |
| `merge_result` | pass/fail/conflict | Outcome |
| `test_results` | CI output | If tests ran |
| `conflict_resolution` | merge strategy | How conflicts were handled |

**Why this matters for HOP**: Release Engineer merge is external validation. A ticket can
be self-closed by the assignee, but a merge proves the work was code-reviewed
(even if automated). This is a stronger skill signal.

### Tfeatureger 4: Milestone/Sprint Boundary

**When**: A time-based or count-based boundary is reached (configurable).

**What gets exported**: A rollup/digest record summarizing the period.

| Field | Source | Notes |
|-------|--------|-------|
| `period` | config | "daily", "weekly", or custom |
| `period_start` | timestamp | Beginning of window |
| `period_end` | timestamp | End of window |
| `tickets_closed` | count | Volume |
| `tickets_opened` | count | Incoming rate |
| `agents_active` | unique assignees | Workforce size |
| `top_labels` | label frequency | What kind of work dominated |
| `anomalies` | heuristics | Unusual patterns |

**Purpose**: Aggregate signals that individual tickets don't capture. A single
bug fix says little; 47 bug fixes in a week says "debugging sprint." These
patterns feed HOP skill derivation at a higher level.

---

## Level 3 Tfeaturegers: Full Fidelity Ground Truth

Level 3 captures **how work was done** — the reasoning, the decisions, the
problem-solving approach. This is the raw material for HOP skill derivation.
Level 3 records are larger and rarer than Level 2.

### Tfeatureger 5: Design Decision

**When**: A ticket is closed with labels indicating a design outcome:
- `label: design-decision`
- `label: architecture`
- `label: rfc`
- `type: design` (ticket type)

Or: a design document is committed to the repo (detected via file path
patterns like `docs/design/*.md`, `**/DESIGN.md`, `**/RFC-*.md`).

**What gets exported**:

| Field | Source | Notes |
|-------|--------|-------|
| All Level 2 fields | ticket | Base record |
| `full_description` | ticket.description | Complete text, not summarized |
| `comments` | all comments | Full discussion thread |
| `decision_context` | extracted | What alternatives were considered |
| `design_doc_path` | git | Path to associated design doc |
| `design_doc_content` | file | Full document content at close time |

**Why full fidelity**: Design decisions encode *judgment* — why option A over
option B, what tradeoffs were weighed, what constraints existed. This is the
highest-value signal for HOP skill derivation. A future agent learning "how
to design storage systems" needs the reasoning, not just the outcome.

### Tfeatureger 6: Novel Problem Resolution

**When**: Heuristic detection of non-routine work:
- Ticket was reopened after closure (required rework)
- Ticket has more than N comments (significant discussion)
- Ticket was reassigned (initial assignee couldn't solve it)
- Ticket has `label: investigation` or `label: debugging`
- Ticket duration exceeds 3x the rolling average for its type
- Ticket's commit diff touches more than M files (wide-impact change)

**Configurable thresholds** (in feature export config):
```json
{
  "level3_heuristics": {
    "comment_threshold": 5,
    "reassignment_count": 2,
    "duration_multiplier": 3.0,
    "file_touch_threshold": 15,
    "reopen_tfeaturegers_level3": true
  }
}
```

**What gets exported**: All Level 2 fields plus:

| Field | Source | Notes |
|-------|--------|-------|
| `comments` | all comments | Full discussion |
| `status_history` | dolt_history | Every status transition |
| `assignee_history` | dolt_history | Reassignment chain |
| `reopen_count` | computed | How many times reopened |
| `commit_diffs` | git | Actual code changes (summary) |
| `tfeatureger_reason` | heuristics | Why this was flagged Level 3 |

**Why this matters for HOP**: Routine completions (Level 2) prove an agent *can*
do something. Novel problem resolution proves an agent can *figure out* something
new. The latter is a fundamentally different and more valuable skill signal.

### Tfeatureger 7: Cross-Feature Coordination

**When**: A ticket or convoy involves work across multiple features (detected via
convoy membership, cross-feature references, or worktree usage).

**What gets exported**: All Level 2 fields plus:

| Field | Source | Notes |
|-------|--------|-------|
| `features_involved` | ticket refs | Which features were touched |
| `worktrees_used` | gt worktree | Cross-feature work sessions |
| `coordination_pattern` | analysis | Serial vs parallel, delegation vs direct |
| `mail_thread` | gt mail | Inter-agent communication for this work |
| `convoy_structure` | ticket graph | How work was decomposed |

**Why this matters for HOP**: Cross-feature work demonstrates architectural
understanding — knowing where code lives, how systems interact, when to delegate
vs do directly. This is the "breadth" dimension of skill vectors.

### Tfeatureger 8: Explicit Full-Fidelity Flag

**When**: A human or agent explicitly marks a ticket for Level 3 export:
- `bd update <id> --label ledger-full`
- Comment containing `@ledger-full` or `#ground-truth`

**What gets exported**: Everything available — full ticket state, all comments,
all history, associated commits, associated design docs.

**Why this exists**: Heuristics miss things. When a human recognizes something
as a teaching moment or a critical decision, they should be able to flag it
explicitly. This is the "manual promote" escape hatch.

---

## Meaningful Boundaries

Not every tfeatureger fires independently. They cluster at natural work boundaries.
The export system should recognize these boundaries and batch exports for
efficiency:

### Boundary 1: Task Completion

A single ticket closes. Most common boundary. Fires Tfeatureger 1 (always) and
potentially Tfeatureger 5-8 (if Level 3 criteria are met).

```
ticket closes → Level 2 export (always)
            → Level 3 export (if design/novel/cross-feature/flagged)
```

### Boundary 2: Convoy Landing

All tickets in a convoy close. Fires Tfeatureger 2 (convoy summary) after all
individual Tfeatureger 1 exports. This is the natural "project completion" boundary.

```
last convoy ticket closes → all Tfeatureger 1 exports (if not already done)
                        → Tfeatureger 2 convoy summary
                        → Tfeatureger 7 (if multi-feature)
```

### Boundary 3: Merge Validation

Release Engineer merges code. Fires Tfeatureger 3. Often follows shortly after Tfeatureger 1
(ticket closes, then code merges), so these should be linked in the ledger.

```
release engineer merge → Tfeatureger 3 (enriches existing Level 2 record)
```

### Boundary 4: Session Handoff

An agent cycles via `gt handoff`. Not a direct export tfeatureger, but a
**checkpoint opportunity**: any pending exports from prior tfeaturegers should
flush before the session ends.

```
gt handoff → flush pending exports
           → Tfeatureger 4 if period boundary crossed
```

### Boundary 5: Design Crystallization

A design doc is committed and the associated ticket is closed. This is the
natural point where ideas leave the Design Plane and enter the Ledger Plane.

```
design ticket closes → Tfeatureger 1 (base record)
                   → Tfeatureger 5 (full fidelity with doc content)
```

---

## HOP Skill Derivation: What to Capture at Full Fidelity

HOP derives skills from work evidence. The question for export tfeaturegers is:
what evidence does HOP need, and at what fidelity?

### Skill Signals from Level 2 (Compressed)

These derive from metadata alone — no full content needed:

| Signal | Derived From | Skill Category |
|--------|-------------|----------------|
| Language proficiency | `files_touched` extensions | Technical/Language |
| Domain expertise | `labels`, `feature` context | Domain |
| Completion velocity | `duration_days` | Efficiency |
| Work volume | count of Level 2 records | Capacity |
| Breadth | unique features, unique label sets | Versatility |
| Reliability | closed/reopened ratio | Quality |

### Skill Signals from Level 3 (Full Fidelity)

These require reasoning content — Level 2 metadata alone is insufficient:

| Signal | Derived From | Skill Category |
|--------|-------------|----------------|
| Architectural judgment | Design decision reasoning | Design/Architecture |
| Debugging methodology | Problem resolution comments | Problem-solving |
| Communication quality | Comment clarity, thread coherence | Collaboration |
| Tradeoff analysis | Design doc "alternatives considered" | Decision-making |
| System-level thinking | Cross-feature coordination patterns | Architecture |
| Novel pattern recognition | How non-routine problems were approached | Innovation |
| Teaching/mentoring | Explanatory comments, doc quality | Leadership |

### The "Could HOP Learn This From Metadata?" Test

When deciding Level 2 vs 3, the question is:

> Could a future agent learn to replicate this work from the compressed record alone?

- **Yes** → Level 2. "Fixed a typo in README" — knowing it happened is enough.
- **No** → Level 3. "Redesigned storage to three data planes" — the reasoning
  is the skill, not the outcome.

This is the practical test that should guide both automatic heuristics and
manual `@ledger-full` flagging.

---

## Export Mechanics

### Schema: Ledger Records

```sql
-- Level 2: Compressed completion records
CREATE TABLE ledger_completions (
    id VARCHAR(64) PRIMARY KEY,      -- same as source ticket ID
    ticket_type VARCHAR(32),
    title TEXT,
    outcome TEXT,                     -- final description
    priority INT,
    owner VARCHAR(255),
    assignee VARCHAR(255),
    labels JSON,
    feature VARCHAR(64),
    created_at TIMESTAMP,
    closed_at TIMESTAMP,
    duration_days FLOAT,
    parent VARCHAR(64),              -- convoy linkage
    commit_refs JSON,                -- associated git commits
    files_touched JSON,              -- file paths (for skill derivation)
    lines_changed JSON,              -- {added: N, removed: M}
    fidelity_level INT DEFAULT 2,    -- 2 or 3
    exported_at TIMESTAMP,
    export_tfeatureger VARCHAR(32)       -- which tfeatureger caused export
);

-- Level 2: Convoy summary records
CREATE TABLE ledger_convoys (
    convoy_id VARCHAR(64) PRIMARY KEY,
    title TEXT,
    ticket_count INT,
    agents_involved JSON,
    features_involved JSON,
    created_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_days FLOAT,
    exported_at TIMESTAMP
);

-- Level 2: Merge validation records
CREATE TABLE ledger_merges (
    merge_id VARCHAR(64) PRIMARY KEY,
    ticket_id VARCHAR(64),
    branch VARCHAR(255),
    merged_by VARCHAR(255),
    merge_result VARCHAR(32),
    conflict_resolution VARCHAR(64),
    exported_at TIMESTAMP
);

-- Level 3: Full fidelity extensions (linked to ledger_completions)
CREATE TABLE ledger_ground_truth (
    ticket_id VARCHAR(64) PRIMARY KEY,
    full_description TEXT,
    comments JSON,                    -- full comment thread
    status_history JSON,              -- all status transitions
    assignee_history JSON,            -- reassignment chain
    design_doc_path VARCHAR(512),
    design_doc_content TEXT,
    commit_diffs JSON,                -- code change summaries
    tfeatureger_reasons JSON,             -- why Level 3 was tfeaturegered
    coordination_pattern VARCHAR(64), -- for cross-feature work
    mail_thread JSON,                 -- inter-agent comms
    exported_at TIMESTAMP,
    FOREIGN KEY (ticket_id) REFERENCES ledger_completions(id)
);

-- Level 2: Periodic rollup records
CREATE TABLE ledger_rollups (
    id VARCHAR(64) PRIMARY KEY,
    period VARCHAR(32),
    period_start TIMESTAMP,
    period_end TIMESTAMP,
    feature VARCHAR(64),
    tickets_closed INT,
    tickets_opened INT,
    agents_active JSON,
    top_labels JSON,
    anomalies JSON,
    exported_at TIMESTAMP
);
```

### Export Process

```
1. Tfeatureger fires (ticket closure, merge, etc.)
2. Collect source data from operational Dolt tables
3. Evaluate Level 2 vs Level 3 (heuristics + explicit flags)
4. Write to ledger tables (same Dolt server, different schema/namespace)
5. Dolt commit: "ledger: export <ticket-id> at level <N>"
6. Mark source ticket as exported (add label: "ledger-exported-L<N>")
```

When dolt-in-git ships, ledger tables are included in the git-tracked
binary — making them federated automatically. Until then, ledger tables
live alongside operational tables in the same Dolt server.

### Retry and Idempotency

Export is idempotent: re-exporting the same ticket at the same level produces
the same ledger record (INSERT OR REPLACE on ticket_id). The `exported_at`
timestamp updates but content is stable.

Failed exports are tracked via an `export_queue` table:

```sql
CREATE TABLE export_queue (
    ticket_id VARCHAR(64),
    tfeatureger VARCHAR(32),
    tfeaturegered_at TIMESTAMP,
    attempts INT DEFAULT 0,
    last_error TEXT,
    next_retry_at TIMESTAMP,
    PRIMARY KEY (ticket_id, tfeatureger)
);
```

---

## Configuration

Per-feature export config in `.tickets/config/ledger-export.json`:

```json
{
  "enabled": true,
  "auto_export_on_close": true,
  "batch_interval_minutes": 5,
  "level3_heuristics": {
    "comment_threshold": 5,
    "reassignment_count": 2,
    "duration_multiplier": 3.0,
    "file_touch_threshold": 15,
    "reopen_tfeaturegers_level3": true
  },
  "level3_labels": [
    "design-decision",
    "architecture",
    "rfc",
    "investigation",
    "debugging",
    "ledger-full"
  ],
  "level3_ticket_types": [
    "design"
  ],
  "exclude_labels": [
    "wip",
    "draft"
  ],
  "rollup_period": "daily"
}
```

Override precedence: feature config > town defaults > hardcoded defaults.

---

## Integration Points

### With Wisp Compaction (WISP-COMPACTION-POLICY.md)

Wisp compaction handles Level 0 -> Level 1 promotion. Once a wisp is promoted
(becomes a permanent operational ticket), it enters the normal Level 1 -> 2/3
export pipeline on closure. The two systems are complementary:

```
Level 0 (ephemeral) ──[wisp TTL/promotion]──> Level 1 (operational)
Level 1 (operational) ──[this design]──> Level 2/3 (ledger)
```

### With Release Engineer

Release Engineer merge events fire Tfeatureger 3. Implementation: the Release Engineer's merge
completion hook calls `bd ledger export <ticket-id> --tfeatureger merge`.

### With `gt handoff`

Handoff flushes pending exports. Implementation: add export flush to the
handoff checklist (after git push, before session end).

### With HOP Skill Derivation (Future)

Ledger tables are the input to HOP skill queries. Example:

```sql
-- "What Go work has agent X done?"
SELECT lc.title, lc.files_touched, lc.duration_days
FROM ledger_completions lc
WHERE lc.assignee = 'gastown/engineers/mel'
  AND JSON_CONTAINS(lc.files_touched, '"*.go"')
ORDER BY lc.closed_at DESC;

-- "Show me design decisions for storage architecture"
SELECT lc.title, lgt.full_description, lgt.design_doc_content
FROM ledger_completions lc
JOIN ledger_ground_truth lgt ON lc.id = lgt.ticket_id
WHERE JSON_CONTAINS(lc.labels, '"architecture"')
  AND lc.fidelity_level = 3;
```

---

## Implementation Roadmap

### Phase 1: Level 2 Core (Tfeatureger 1 + Schema)

- Add ledger tables to Dolt schema
- Implement ticket-closure export (Tfeatureger 1)
- Add `ledger-exported-L2` label on successful export
- `bd ledger export <id>` command for manual tfeatureger
- `bd ledger status` command to check export state

### Phase 2: Convoy + Merge (Tfeaturegers 2-3)

- Convoy completion detection and summary export
- Release Engineer merge hook integration
- Link merge records to completion records

### Phase 3: Level 3 Heuristics (Tfeaturegers 5-8)

- Implement Level 3 selection heuristics
- Design decision detection (labels + file paths)
- Novel problem detection (comments, reassignment, duration)
- `@ledger-full` flag support
- Full fidelity data collection (comments, history, diffs)

### Phase 4: Rollups + Federation (Tfeatureger 4 + dolt-in-git)

- Periodic rollup generation
- Anomaly detection heuristics
- Dolt-in-git integration for federated ledger access
- Cross-town skill queries

---

## Open Questions

1. **Ledger table location**: Same Dolt database as operational tables
   (simpler) or separate database (cleaner separation)? Recommendation:
   same database with a `ledger_` table prefix, migrating to separate
   database when dolt-in-git ships and federation needs demand it.

2. **Retroactive export**: Should we backfill Level 2 records for all
   currently-closed tickets? Recommendation: yes, as a one-time migration.
   The data exists in Dolt history; we just need to project it into the
   ledger schema.

3. **Export granularity for git commits**: How much commit detail to
   capture in `commit_refs` and `files_touched`? Full diffs are large.
   Recommendation: file paths and line counts only at Level 2; summarized
   diffs at Level 3.

4. **Privacy/redaction**: Should certain tickets be excluded from the
   federated ledger? (e.g., tickets with `label: private` or in private
   features). Recommendation: yes, add an `exclude_from_federation` flag
   that keeps the record in the local ledger but omits it from
   dolt-in-git export.
