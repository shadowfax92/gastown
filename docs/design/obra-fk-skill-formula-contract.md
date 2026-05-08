# Obra FK Skill/Formula Registry Contract

Status: implemented as a static registry in `internal/formula`.

## Purpose

Obra FK skills are the user-facing names that Codex and Claude sessions can
invoke. Gas Town formulas are the executable workflows that a hooked session can
run. The registry is the contract between those layers: every supported skill
must name the formula it prefers, the invocation aliases agents may recognize,
the session runtimes covered by the contract, and the fallback behavior to use
when the preferred formula is not available.

This contract is intentionally data-only. Doctor checks and skill installers can
consume it later without coupling the registry to formula resolution or embedded
formula provisioning.

## Metadata

Each registry row contains:

- `Skill`: canonical skill name.
- `Formula`: preferred Gas Town formula name.
- `Availability`: `source-controlled` when the formula is expected to resolve
  through the normal formula tiers, or `planned` when the skill name is reserved
  before its dedicated formula exists.
- `InvocationNames`: accepted names such as `obra-fk-lite-exec` and
  `/obra-fk-lite-exec`.
- `Sessions`: supported runtime contracts. Current rows cover both `codex` and
  `claude`.
- `Phase`: coarse workflow phase for UI and routing.
- `RequiresAcceptedDesign`: true when execution should begin from an accepted
  design or feature breakdown.
- `FallbackFormula`: formula to run when the preferred formula is unavailable.
- `FallbackBehavior`: concise operator instructions for Codex/Claude sessions.

## Registry

| Skill | Formula | Availability | Phase | Requires accepted design | Fallback formula |
| --- | --- | --- | --- | --- | --- |
| `obra-fk-design` | `obra-fk-design` | `planned` | `design` | no | `obra-fk-lite-plan` |
| `obra-fk-exec` | `obra-fk-exec` | `planned` | `execution` | yes | `obra-fk-lite-exec` |
| `obra-fk-debug` | `obra-fk-debug` | `planned` | `debug` | no | `tdd-cycle` |
| `obra-fk-debug-fix` | `obra-fk-debug-fix` | `planned` | `debug-fix` | no | `tdd-cycle` |
| `obra-fk-dev-full` | `obra-fk-dev-full` | `planned` | `full-development` | no | `obra-fk-lite` |
| `obra-fk-lite` | `obra-fk-lite` | `source-controlled` | `lite-end-to-end` | no | `mol-polecat-work` |
| `obra-fk-lite-plan` | `obra-fk-lite-plan` | `source-controlled` | `lite-planning` | no | `mol-idea-to-plan` |
| `obra-fk-lite-exec` | `obra-fk-lite-exec` | `source-controlled` | `lite-execution` | yes | `mol-polecat-work` |

Every row accepts the canonical skill name and the slash-command form as
invocation aliases. For example, `obra-fk-lite-exec` and `/obra-fk-lite-exec`
both target the same contract.

## Runtime Behavior

Codex and Claude sessions follow the same resolution order:

1. If the preferred formula resolves, run it.
2. If the preferred formula is `planned` or cannot be resolved, run the
   `FallbackFormula`.
3. If formula execution is unavailable in that runtime, execute the matching
   skill instructions manually and preserve the same outputs in beads, notes,
   design fields, commits, and `gt done` submission.

Fallbacks are chosen to preserve behavior rather than names. For example,
`obra-fk-design` falls back to `obra-fk-lite-plan` because both produce an
accepted design and executable beads. Debug skills fall back to `tdd-cycle`
because the contract needs a reproduce, test, fix, verify loop until dedicated
debug formulas exist.

## Validation Rules

`ValidateObraFKSkillFormulaRegistry` enforces the contract shape:

- skill names are present and unique;
- formula names are present;
- availability is one of the known enum values;
- invocation aliases are present and unique across the registry;
- sessions are known runtime identifiers;
- phase, fallback formula, and fallback behavior are present;
- fallback formulas either point to another registered formula or to one of the
  generic fallbacks: `mol-idea-to-plan`, `mol-polecat-work`, or `tdd-cycle`.

Doctor checks should call the same validation helper before checking local
filesystem availability for source-controlled formulas.
