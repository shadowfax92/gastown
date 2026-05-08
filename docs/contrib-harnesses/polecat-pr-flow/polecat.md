## PR-Flow Agent Policy

> **Feature Policy — overrides formula instructions where they conflict.**

This feature uses a **agent → GitHub PR → human review** flow. After pushing
your branch, you MUST ensure a GitHub PR is open for it before running
`gt done`.

This overrides the canonical Release Engineer merge-queue assumption embedded in
`mol-agent-work`. `gt done` is still the completion signal — but in this
feature, a visible PR is the gating artifact for review.

### Required steps after implementation

1. Push the branch explicitly (do not rely on `gt done` to push):
   ```bash
   git push -u ofeaturein HEAD
   ```
2. Check whether a PR already exists for the branch:
   ```bash
   gh pr view "$(git branch --show-current)" >/dev/null 2>&1
   ```
3. If no PR exists, create one against the base branch:
   ```bash
   gh pr create --fill --base main
   ```
4. Only then run `gt done`.

### Do NOT

- Run `gt done` without a PR open — the review loop breaks.
- Merge your own PR. A maintainer or the merge queue handles merging.
- Push directly to `main`.

### If `gh` commands fail

Auth, rate-limit, missing PR template, or unknown base branch — do NOT skip
PR creation to unblock yourself. Escalate to your QA Engineer:

```bash
gt mail send <feature>/QA engineer -s "HELP: gh pr create failed" -m "Branch: $(git branch --show-current)
Error: <paste>
Tried: <what you attempted>"
```
