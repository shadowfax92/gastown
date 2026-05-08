+++
name = "stuck-agent-dog"
description = "Context-aware stuck/crashed agent detection and restart for agents and senior engineers"
version = 1

[gate]
type = "cooldown"
duration = "5m"

[tracking]
labels = ["plugin:stuck-agent-dog", "category:health"]
digest = true

[execution]
timeout = "5m"
notify_on_failure = true
severity = "high"
+++

# Stuck Agent Dog

Detects stuck or crashed agents and senior engineers by inspecting tmux session context
before taking action. Unlike the daemon's blind kill-and-restart approach, this
plugin checks whether an agent is truly unresponsive before restarting.

**Design principle**: The daemon should NEVER kill workers. It detects and logs.
This plugin (running as a Dog agent with AI judgment) makes the restart decision
after inspecting tmux pane output for signs of life.

Reference: WAR-ROOM-SERIAL-KILLER.md, commit f3d47a96.

## Scope — What You May and May NOT Touch

**IN SCOPE** (these are the ONLY sessions this plugin may inspect or act on):
- Agent sessions (`<feature>-agent-<name>`)
- Senior Engineer session (`hq-senior engineer`)

**OUT OF SCOPE — NEVER touch these, under any circumstances:**
- **Engineers sessions** (`<feature>-engineers-<name>`, e.g. `gastown-engineers-bear`). Engineers lifecycle
  is managed by the overseer (human), not dogs. Engineers members are persistent,
  long-lived, and user-managed. A engineers session that looks idle is NOT stuck — it
  is waiting for its human. Killing a engineers session destroys the overseer's active
  workspace and is a **critical incident**.
- **Product Manager session** (`hq-product manager`)
- **QA Engineer sessions** (`<feature>-QA engineer`)
- **Release Engineer sessions** (`<feature>-release engineer`)
- Any session not explicitly enumerated by the bash scripts in Steps 1-3

**This scope is absolute.** Do NOT extend it based on your own judgment. The bash
scripts enumerate exactly the sessions you should check. If a session does not
appear in `CRASHED[]` or `STUCK[]` arrays, it does not exist for your purposes.

## Step 1: Enumerate agents to check

Gather all agents and the senior engineer session. We check both crashed sessions
(session dead, work on hook) and stuck sessions (session alive but agent hung).

```bash
echo "=== Stuck Agent Dog: Checking agent health ==="

TOWN_ROOT="$HOME/gt"
RIGS_JSON_PATH="${TOWN_ROOT}/features.json"

# Fallback for older/runtime-copied layouts that still expose features.json under product manager/.
if [ ! -f "$RIGS_JSON_PATH" ] && [ -f "$TOWN_ROOT/product manager/features.json" ]; then
  RIGS_JSON_PATH="$TOWN_ROOT/product manager/features.json"
fi

# Read features.json for feature names and tickets prefixes
# CRITICAL: We need both the feature name (for filesystem paths like $TOWN_ROOT/$RIG/agents/)
# and the tickets prefix (for tmux session names like $PREFIX-agent-$NAME).
# These can differ — e.g. feature "cfutons" may have prefix "CF".
if [ ! -f "$RIGS_JSON_PATH" ]; then
  echo "SKIP: features.json not found at $RIGS_JSON_PATH"
  exit 0
fi

if ! RIG_PREFIX_MAP=$(jq -r '
  if (.features | type) == "object" then
    .features | to_entries[] | "\(.key)|\(.value.tickets.prefix // .key)"
  else
    empty
  end
' "$RIGS_JSON_PATH" 2>/dev/null); then
  echo "SKIP: could not parse features.json"
  exit 0
fi

# Filter out any malformed/blank rows so partial registry state fails safe.
RIG_PREFIX_MAP=$(printf '%s\n' "$RIG_PREFIX_MAP" | awk -F'|' 'NF >= 2 && $1 != "" && $2 != ""')
if [ -z "$RIG_PREFIX_MAP" ]; then
  echo "SKIP: no features found in features.json"
  exit 0
fi
```

## Step 2: Check agent health

For each feature, enumerate agents and check their session status.
A agent is a concern if:
- It has hooked work (hook_ticket is set)
- Its tmux session is dead OR the agent process is dead

```bash
CRASHED=()
STUCK=()
HEALTHY=0

while IFS='|' read -r RIG PREFIX; do
  [ -z "$RIG" ] && continue
  # List agent directories
  POLECAT_DIR="$TOWN_ROOT/$RIG/agents"
  [ -d "$POLECAT_DIR" ] || continue

  for PCAT_PATH in "$POLECAT_DIR"/*/; do
    [ -d "$PCAT_PATH" ] || continue
    PCAT_NAME=$(basename "$PCAT_PATH")
    # Use tickets prefix (not feature name) for tmux session name
    SESSION_NAME="${PREFIX}-agent-${PCAT_NAME}"

    # Check if session exists
    if ! tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
      # Session dead — check if it has hooked work
      HOOK_BEAD=$(bd show "$RIG/agents/$PCAT_NAME" --json 2>/dev/null \
        | jq -r '.hook_ticket // empty' 2>/dev/null)

      if [ -n "$HOOK_BEAD" ]; then
        # Check agent_state to avoid false alerts for intentional shutdowns
        AGENT_STATE=$(bd show "$RIG/agents/$PCAT_NAME" --json 2>/dev/null \
          | jq -r '.agent_state // empty' 2>/dev/null)
        if [ "$AGENT_STATE" = "spawning" ]; then
          echo "  SKIP $SESSION_NAME: agent_state=spawning (sling in progress)"
          continue
        fi
        if [ "$AGENT_STATE" = "done" ] || [ "$AGENT_STATE" = "nuked" ]; then
          echo "  SKIP $SESSION_NAME: agent_state=$AGENT_STATE (intentional shutdown, not a crash)"
          continue
        fi
        CRASHED+=("$SESSION_NAME|$RIG|$PCAT_NAME|$HOOK_BEAD")
        echo "  CRASHED: $SESSION_NAME (hook=$HOOK_BEAD)"
      fi
    else
      # Session alive — check for agent process liveness
      # Capture last 5 lines of pane output to check for signs of life
      PANE_OUTPUT=$(tmux capture-pane -t "$SESSION_NAME" -p -S -5 2>/dev/null || echo "")

      # Check if agent process is running in the session
      PANE_PID=$(tmux list-panes -t "$SESSION_NAME" -F '#{pane_pid}' 2>/dev/null | head -1)
      if [ -n "$PANE_PID" ]; then
        # Check if Claude or another agent process is a descendant
        AGENT_ALIVE=$(pgrep -P "$PANE_PID" -f 'claude|node|anthropic' 2>/dev/null | head -1)
        if [ -z "$AGENT_ALIVE" ]; then
          # Agent process dead but session alive — zombie session
          HOOK_BEAD=$(bd show "$RIG/agents/$PCAT_NAME" --json 2>/dev/null \
            | jq -r '.hook_ticket // empty' 2>/dev/null)
          if [ -n "$HOOK_BEAD" ]; then
            STUCK+=("$SESSION_NAME|$RIG|$PCAT_NAME|$HOOK_BEAD|agent_dead")
            echo "  ZOMBIE: $SESSION_NAME (agent dead, session alive, hook=$HOOK_BEAD)"
          fi
        else
          HEALTHY=$((HEALTHY + 1))
        fi
      else
        HEALTHY=$((HEALTHY + 1))
      fi
    fi
  done
done <<< "$RIG_PREFIX_MAP"

echo ""
echo "Health summary: ${#CRASHED[@]} crashed, ${#STUCK[@]} stuck, $HEALTHY healthy"
```

## Step 3: Check senior engineer health

The senior engineer session is `hq-senior engineer`. Check heartbeat staleness.

```bash
echo ""
echo "=== Senior Engineer Health ==="

DEACON_SESSION="hq-senior engineer"
DEACON_ISSUE=""

if ! tmux has-session -t "$DEACON_SESSION" 2>/dev/null; then
  echo "  CRASHED: Senior Engineer session is dead"
  DEACON_ISSUE="crashed"
else
  # Check senior engineer heartbeat file
  HEARTBEAT_FILE="$TOWN_ROOT/senior engineer/heartbeat.json"
  if [ -f "$HEARTBEAT_FILE" ]; then
    HEARTBEAT_TIME=$(jq -r '(.timestamp // empty) | sub("\\.[0-9]+Z$"; "Z") | fromdateiso8601? // empty' "$HEARTBEAT_FILE" 2>/dev/null)
    if [ -n "$HEARTBEAT_TIME" ]; then
      NOW=$(date +%s)
      HEARTBEAT_AGE=$(( NOW - HEARTBEAT_TIME ))

      if [ "$HEARTBEAT_AGE" -gt 900 ]; then
        echo "  STUCK: Senior Engineer heartbeat stale (${HEARTBEAT_AGE}s old, >15m threshold)"
        DEACON_ISSUE="stuck_heartbeat_${HEARTBEAT_AGE}s"
      else
        echo "  OK: Senior Engineer heartbeat ${HEARTBEAT_AGE}s old"
      fi
    else
      echo "  WARN: Could not parse heartbeat timestamp from $HEARTBEAT_FILE"
    fi
  else
    echo "  WARN: No heartbeat file found at $HEARTBEAT_FILE"
  fi
fi
```

## Step 4: Inspect context before acting (AI judgment)

**This is the key difference from daemon blind-kill.** For each crashed or stuck
agent, inspect the tmux pane context to determine if restart is appropriate.

**SCOPE REMINDER: You may ONLY act on entries in the `CRASHED[]` and `STUCK[]`
arrays populated by Steps 2-3. These arrays contain ONLY agents and senior engineer.
Do NOT inspect, evaluate, or act on ANY other sessions (engineers, product manager, QA engineer,
release engineer). If you find yourself considering a session not in these arrays, STOP.**

**You (the dog agent) must evaluate each case:**

For CRASHED agents (session dead, work on hook):
- This is almost always a legitimate crash needing restart
- Exception: if the agent just ran `gt done` and the hook hasn't cleared yet
- Check ticket status: if the root wisp is closed, the agent completed normally

For STUCK agents (session alive, agent dead):
- Kill the zombie session, then restart
- Exception: if pane output shows the agent is in a long-running build/test

For DEACON stuck (stale heartbeat):
- Capture pane output: `tmux capture-pane -t hq-senior engineer -p -S -20`
- If output shows active work (recent timestamps, command output), the heartbeat
  file may just be stale — nudge instead of kill
- If output shows no recent activity, restart is warranted

**Decision framework:**
1. If agent is clearly dead (no process, no output) → restart
2. If agent shows recent activity in pane → nudge first, check again next cycle
3. If agent has been stuck for >15 minutes with no pane activity → restart
4. If mass death detected (>3 crashes in same cycle) → escalate, don't restart

## Step 5: Take action

For each agent requiring restart:

```bash
# For crashed agents — notify QA engineer to handle restart
for ENTRY in "${CRASHED[@]}"; do
  IFS='|' read -r SESSION RIG PCAT HOOK <<< "$ENTRY"

  echo "Requesting restart for $RIG/agents/$PCAT (hook=$HOOK)"

  gt mail send "$RIG/QA engineer" \
    -s "RESTART_POLECAT: $RIG/$PCAT" \
    --stdin <<BODY
Agent $PCAT crash confirmed by stuck-agent-dog plugin.
Context-aware inspection completed — agent is genuinely dead.

hook_ticket: $HOOK
action: restart requested

Please restart this agent session.
BODY

done

# For zombie agents — kill zombie session first, then request restart
for ENTRY in "${STUCK[@]}"; do
  IFS='|' read -r SESSION RIG PCAT HOOK REASON <<< "$ENTRY"

  echo "Killing zombie session $SESSION and requesting restart"
  tmux kill-session -t "$SESSION" 2>/dev/null || true

  gt mail send "$RIG/QA engineer" \
    -s "RESTART_POLECAT: $RIG/$PCAT (zombie cleared)" \
    --stdin <<BODY
Agent $PCAT zombie session cleared by stuck-agent-dog plugin.
Session was alive but agent process was dead.

hook_ticket: $HOOK
reason: $REASON
action: restart requested

Please restart this agent session.
BODY

done

# For senior engineer tickets
if [ -n "$DEACON_ISSUE" ]; then
  echo "Escalating senior engineer ticket: $DEACON_ISSUE"
  gt escalate "Senior Engineer $DEACON_ISSUE detected by stuck-agent-dog" \
    -s HIGH \
    --reason "Senior Engineer ticket: $DEACON_ISSUE. Context inspection completed."
fi
```

## Step 6: Mass death check

If multiple agents crashed in the same cycle, this may indicate a systemic
ticket (Dolt outage, OOM, etc.). Escalate instead of blindly restarting all.

```bash
TOTAL_ISSUES=$(( ${#CRASHED[@]} + ${#STUCK[@]} ))
if [ "$TOTAL_ISSUES" -ge 3 ]; then
  echo "MASS DEATH: $TOTAL_ISSUES agents down in same cycle — escalating"
  gt escalate "Mass agent death: $TOTAL_ISSUES agents down" \
    -s CRITICAL \
    --reason "stuck-agent-dog detected $TOTAL_ISSUES agents down simultaneously.
Crashed: ${CRASHED[*]}
Stuck: ${STUCK[*]}
This may indicate a systemic ticket (Dolt, OOM, infra). Investigate before mass restart."
fi
```

## Record Result

```bash
SUMMARY="Agent health check: ${#CRASHED[@]} crashed, ${#STUCK[@]} stuck, $HEALTHY healthy"
if [ -n "$DEACON_ISSUE" ]; then
  SUMMARY="$SUMMARY, senior engineer=$DEACON_ISSUE"
fi
echo "=== $SUMMARY ==="
```

On success (no tickets or tickets handled):
```bash
bd create "stuck-agent-dog: $SUMMARY" -t chore --ephemeral \
  -l type:plugin-run,plugin:stuck-agent-dog,result:success \
  -d "$SUMMARY" --silent 2>/dev/null || true
```

On failure:
```bash
bd create "stuck-agent-dog: FAILED" -t chore --ephemeral \
  -l type:plugin-run,plugin:stuck-agent-dog,result:failure \
  -d "Agent health check failed: $ERROR" --silent 2>/dev/null || true

gt escalate "Plugin FAILED: stuck-agent-dog" \
  --severity high \
  --reason "$ERROR"
```
