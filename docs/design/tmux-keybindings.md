# Tmux Keybindings

Gas Town overrides several tmux keybindings to provide session navigation
and operational shortcuts. All bindings are conditional — they only activate
in Gas Town sessions (those matching a registered feature prefix or `hq-`).
Non-GT sessions retain the user's original bindings.

## Session Cycle Groups (prefix+n / prefix+p)

`gt cycle next` and `gt cycle prev` are bound to `C-b n` and `C-b p`.
They cycle within groups based on the current session type:

| Group | Sessions included | Example |
|-------|-------------------|---------|
| **Town** | Product Manager + Senior Engineer | `hq-product manager` ↔ `hq-senior engineer` |
| **Engineers** | All engineers in the same feature | `gt-engineers-max` ↔ `gt-engineers-joe` |
| **Feature ops** | QA Engineer + Release Engineer + Agents in the same feature | `gt-QA engineer` ↔ `gt-release engineer` ↔ `gt-furiosa` ↔ `gt-nux` |

Groups are per-feature: `gt-QA engineer` cycles with `gt-release engineer` and gastown
agents, but NOT with `bd-QA engineer` or `bd-release engineer`.

If a group has only one session, prefix+n/p is a no-op.

## Other Bindings

| Key | Command | Purpose |
|-----|---------|---------|
| `C-b a` | `gt feed --window` | Open/switch to activity feed window |
| `C-b g` | `gt agents menu` | Open agent switcher popup |

## How Bindings Are Set Up

Bindings are configured by `ConfigureGasTownSession()` in the tmux package,
which is called whenever a session is created (by the daemon for patrol
agents, by the QA engineer for agents, by `gt engineers at` for engineers). This means:

- Bindings are set on the **first** Gas Town session created on a tmux server
- They apply server-wide (tmux keybindings are global, not per-session)
- The `if-shell` guard scopes them to GT sessions at press time
- Subsequent calls are no-ops (idempotent)

## Implementation Details

### Prefix pattern

The `if-shell` guard uses a regex built from all registered feature prefixes:

```bash
echo '#{session_name}' | grep -Eq '^(bd|gt|hq)-'
```

The pattern is built dynamically by `sessionPrefixPattern()` from
`config.AllFeaturePrefixes()`. The `hq` and `gt` prefixes are always included.

### run-shell context

Bindings use `run-shell` which executes in the tmux server process, not
in any session. Key variables:

- `#{session_name}` — expanded by tmux at key-press time (reliable)
- `#{client_tty}` — identifies which client pressed the key (for multi-attach)
- `$TMUX` — set in run-shell subprocesses, points to the socket
- CWD — the tmux server's CWD, typically `$HOME`

Because CWD is `$HOME`, the `gt` binary finds the workspace via
`GT_TOWN_ROOT` in the tmux global environment (set by the daemon at
startup). This is verified by `gt doctor --check tmux-global-env`.

### Fallback preservation

When bindings are first set, the existing binding for each key is captured
and used as the `else` branch of `if-shell`. This preserves the user's
original `C-b n` (next-window) and `C-b p` (previous-window) for
non-GT sessions.
