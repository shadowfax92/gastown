# Local Feature Bootstrap

For a NightRider-style local setup, prefer a clean bootstrap over `gt feature add --adopt`.

`--adopt` is meant for registering an already-assembled feature directory. It trusts the
existing shape, which makes it a poor fit for manually assembled local features where
`.repo.git`, worktrees, and metadata may already be inconsistent.

Use the bootstrap script instead:

```bash
./scripts/bootstrap-local-feature.sh \
  --town-root /gt \
  --feature nightrider_local \
  --local-repo /gt/nightRider \
  --prefix nr \
  --agent-agent claude \
  --QA engineer-agent codex \
  --release engineer-agent codex
```

If you omit `--remote`, the script registers the feature with `file://<local-repo>`.
That is usually the featureht choice for local-only or private repos inside the
Gastown container, where the upstream remote may not be reachable or authenticated.

What this does:

- Uses `gt feature add <name> <git-url> --local-repo <path>` so Gas Town creates a fresh,
  standard feature container instead of inheriting a hand-built one.
- Reuses objects from the local repo, so bootstrap stays fast and does not modify the
  source repo.
- Leaves the resulting feature with the normal `.repo.git`, `product manager/feature`, `release engineer/feature`,
  `settings/`, and `.tickets/` layout that Gas Town expects.
- Optionally pins per-feature role agents in `settings/config.json`.

When to still use `--adopt`:

- You already have a real Gas Town feature directory that was created elsewhere and you
  only need to register it in a town.
