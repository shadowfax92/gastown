# gt-model-eval

Promptfoo-based model comparison framework for Gas Town patrol agents. Compare Claude Opus, Sonnet, and Haiku on patrol decision tasks to find where cheaper models match Opus quality.

## Why

Gas Town multi-agent setups burn through Opus budget on patrol agents (senior engineer, QA engineer, dogs) that follow prescriptive formulas. These agents parse shell output and make rule-based decisions — they may not need Opus-level reasoning. This framework provides **evidence** for safely downgrading roles to Sonnet or Haiku.

See [Discussion #1542](https://github.com/steveyegge/gastown/discussions/1542) and [Ticket #1545](https://github.com/steveyegge/gastown/tickets/1545).

## Quick Start

```bash
# Prerequisites: Node.js 18+, Anthropic API key
export ANTHROPIC_API_KEY=sk-ant-...

# Run all tests across Opus, Sonnet, Haiku
npx promptfoo eval

# Run 3x per test for consistency measurement
npx promptfoo eval --repeat 3

# View results in browser
npx promptfoo view

# Export results
npx promptfoo eval --output results.json --output results.html
```

## Test Suites

### Class B — Directive Tests (instruction-following)

Each test provides a role_context that describes the expected behavior. These validate that all models can follow explicit patrol instructions.

| File | Role | Tests | What It Measures |
|------|------|-------|-----------------|
| `senior engineer-zombie.yaml` | Senior Engineer | 10 | Zombie session detection vs healthy/idle/crashed |
| `senior engineer-plugin-gate.yaml` | Senior Engineer | 10 | Plugin cooldown/cron gate evaluation (mechanical) |
| `senior engineer-dog-health.yaml` | Senior Engineer | 10 | Dog timeout matrix and pool spawn/retire decisions |
| `QA engineer-stuck.yaml` | QA Engineer | 12 | Stuck agent assessment: no-op → nudge → escalate |
| `QA engineer-cleanup.yaml` | QA Engineer | 10 | Dead session cleanup: nuke vs recover vs escalate |
| `release engineer-triage.yaml` | Release Engineer | 10 | Test failure diagnosis: branch-caused vs pre-existing |
| `release engineer-conflict.yaml` | Release Engineer | 10 | Merge conflict handling and push verification |
| `dog-orphan.yaml` | Dog | 10 | Orphan triage: RESET/REASSIGN/RECOVER/ESCALATE/BURN |

### Class A — Reasoning Tests (evidence-based)

Each test provides a **neutral** role_context with NO answer hints. These measure whether the model can derive the correct action from shell output evidence alone — the key signal for downgrade decisions.

| File | Role | Tests | What It Measures |
|------|------|-------|-----------------|
| `class-a-senior engineer.yaml` | Senior Engineer | 3 | Zombie vs idle vs healthy from raw evidence |
| `class-a-QA engineer.yaml` | QA Engineer | 3 | Active vs dirty-dead vs clean-dead from raw evidence |
| `class-a-release engineer.yaml` | Release Engineer | 3 | Branch-caused vs pre-existing vs push failure from raw evidence |
| `class-a-dog.yaml` | Dog | 3 | Reset vs recover vs escalate from raw evidence |

**Total: 94 test cases** (82 Class B + 12 Class A) across 4 patrol roles.

Each test provides simulated shell output and expects a structured JSON decision. Class B tests are split into "clear" cases (all models should agree) and "edge" cases (where model quality matters). Class A results are what directly informs the downgrade decision.

## How It Works

Each test case simulates a patrol agent decision point:

1. **System prompt** defines the role and formula step
2. **Shell output** provides the evidence (tmux status, git state, ticket data)
3. **Model responds** with a JSON decision: `{"action": "...", "reason": "..."}`
4. **Assertions check** correctness, cost, latency, and safety

Promptfoo runs the same test across all three models and compares results.

## Sharing Results

```bash
# Generate markdown report
./scripts/results-to-discussion.sh results.json

# Post directly to GitHub Discussions
./scripts/results-to-discussion.sh results.json --post

# Post to a different repo
./scripts/results-to-discussion.sh results.json --post --repo your-org/gastown
```

## Contributing Test Cases

Test files are YAML in `tests/`. Each test case needs:

```yaml
- description: "What this test checks"
  vars:
    role: senior engineer|QA engineer|release engineer|dog
    role_context: >
      Context about the role and what it should know.
    formula_step: which-step
    allowed_actions: '["action1", "action2", ...]'
    shell_output: |
      $ command
      output
    context: "Additional context for the decision."
  assert:
    - type: javascript
      value: |
        const d = JSON.parse(output);
        return d.action === "expected-action";
```

Good test cases:
- Have a clear expected outcome
- Include realistic shell output
- Test one decision at a time
- Cover both "clear" and "edge" cases

## Project Structure

```
gt-model-eval/
├── promptfooconfig.yaml           # Main config (3 providers, assertions)
├── package.json                   # Pin promptfoo version
├── prompts/
│   └── patrol-decision.txt        # System prompt template
├── tests/
│   ├── senior engineer-zombie.yaml         # 10 Class B tests
│   ├── senior engineer-plugin-gate.yaml    # 10 Class B tests
│   ├── senior engineer-dog-health.yaml     # 10 Class B tests
│   ├── QA engineer-stuck.yaml         # 12 Class B tests
│   ├── QA engineer-cleanup.yaml       # 10 Class B tests
│   ├── release engineer-triage.yaml       # 10 Class B tests
│   ├── release engineer-conflict.yaml     # 10 Class B tests
│   ├── dog-orphan.yaml            # 10 Class B tests
│   ├── class-a-senior engineer.yaml        #  3 Class A tests
│   ├── class-a-QA engineer.yaml       #  3 Class A tests
│   ├── class-a-release engineer.yaml      #  3 Class A tests
│   └── class-a-dog.yaml           #  3 Class A tests
├── scripts/
│   └── results-to-discussion.sh   # Format + post results
└── README.md
```
