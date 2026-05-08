# Formula Resolution Architecture

> **Status: Partially implemented** — Basic formula resolution works. Tier enforcement, Mol Mall integration, and HOP federation are planned.

> Where formulas live, how they're found, and how they'll scale to Mol Mall

## The Problem

Formulas currently exist in multiple locations with no clear precedence:
- `internal/formula/formulas/` (source of truth, embedded in binary)
- `.tickets/formulas/` (provisioned at runtime by `gt install`)
- Engineers directories have their own `.tickets/formulas/` (diverging copies)

When an agent runs `bd cook mol-agent-work`, which version do they get?

## Design Goals

1. **Predictable resolution** - Clear precedence rules
2. **Local customization** - Override system defaults without forking
3. **Project-specific formulas** - Committed workflows for collaborators
4. **Mol Mall ready** - Architecture supports remote formula installation
5. **Federation ready** - Formulas are shareable across towns via HOP (Highway Operations Protocol)

## Three-Tier Resolution

```
┌─────────────────────────────────────────────────────────────────┐
│                     FORMULA RESOLUTION ORDER                     │
│                    (most specific wins)                          │
└─────────────────────────────────────────────────────────────────┘

TIER 1: PROJECT (feature-level)
  Location: <project>/.tickets/formulas/
  Source:   Committed to project repo
  Use case: Project-specific workflows (deploy, test, release)
  Example:  ~/gt/gastown/.tickets/formulas/mol-gastown-release.formula.toml

TIER 2: TOWN (user-level)
  Location: ~/gt/.tickets/formulas/
  Source:   Mol Mall installs, user customizations
  Use case: Cross-project workflows, personal preferences
  Example:  ~/gt/.tickets/formulas/mol-agent-work.formula.toml (customized)

TIER 3: SYSTEM (embedded)
  Location: Compiled into gt binary
  Source:   internal/formula/formulas/ at build time
  Use case: Defaults, blessed patterns, fallback
  Example:  mol-agent-work.formula.toml (factory default)
```

### Resolution Algorithm

```go
func ResolveFormula(name string, cwd string) (Formula, Tier, error) {
    // Tier 1: Project-level (walk up from cwd to find .tickets/formulas/)
    if projectDir := findProjectRoot(cwd); projectDir != "" {
        path := filepath.Join(projectDir, ".tickets", "formulas", name+".formula.toml")
        if f, err := loadFormula(path); err == nil {
            return f, TierProject, nil
        }
    }

    // Tier 2: Town-level
    townDir := getTownRoot() // ~/gt or $GT_HOME
    path := filepath.Join(townDir, ".tickets", "formulas", name+".formula.toml")
    if f, err := loadFormula(path); err == nil {
        return f, TierTown, nil
    }

    // Tier 3: Embedded (system)
    if f, err := loadEmbeddedFormula(name); err == nil {
        return f, TierSystem, nil
    }

    return nil, 0, ErrFormulaNotFound
}
```

### Why This Order

**Project wins** because:
- Project maintainers know their workflows best
- Collaborators get consistent behavior via git
- CI/CD uses the same formulas as developers

**Town is middle** because:
- User customizations override system defaults
- Mol Mall installs don't require project changes
- Cross-project consistency for the user

**System is fallback** because:
- Always available (compiled in)
- Factory reset target
- The "blessed" versions

## Formula Identity

### Current Format

```toml
formula = "mol-agent-work"
version = 4
description = "..."
```

### Extended Format (Mol Mall Ready)

```toml
[formula]
name = "mol-agent-work"
version = "4.0.0"                          # Semver
author = "steve@gastown.io"                # Author identity
license = "MIT"
repository = "https://github.com/steveyegge/gastown"

[formula.registry]
uri = "hop://molmall.gastown.io/formulas/mol-agent-work@4.0.0"
checksum = "sha256:abc123..."              # Integrity verification
signed_by = "steve@gastown.io"             # Optional signing

[formula.capabilities]
# What capabilities does this formula exercise? Used for agent routing.
primary = ["go", "testing", "code-review"]
secondary = ["git", "ci-cd"]
```

### Version Resolution

When multiple versions exist:

```bash
bd cook mol-agent-work          # Resolves per tier order
bd cook mol-agent-work@4        # Specific major version
bd cook mol-agent-work@4.0.0    # Exact version
bd cook mol-agent-work@latest   # Explicit latest
```

## Engineers Directory Problem

### Current State

Engineers directories (`gastown/engineers/max/`) are git worktrees of the featureged repo. They have:
- Their own `.tickets/formulas/` (from the worktree)
- These can diverge from `product manager/feature/.tickets/formulas/`

### The Fix

Engineers should NOT have their own formula copies. Options:

**Option A: Symlink/Redirect**
```bash
# engineers/max/.tickets/formulas -> ../../product manager/feature/.tickets/formulas
```
All engineers share the feature's formulas.

**Option B: Provision on Demand**
Engineers directories don't have `.tickets/formulas/`. Resolution falls through to:
1. Town-level (~/gt/.tickets/formulas/)
2. System (embedded)

**Option C: Gitignore Exclusion**
Exclude `.tickets/formulas/` from engineers worktrees via `.gitignore`.

**Recommendation: Option B** - Engineers shouldn't need project-level formulas. They work on the project, they don't define its workflows.

## Commands

### Existing

```bash
bd formula list              # Available formulas (should show tier)
bd formula show <name>       # Formula details
bd cook <formula>            # Formula → Proto
```

### Enhanced

```bash
# List with tier information
bd formula list
  mol-agent-work          v4    [project]
  mol-agent-code-review   v1    [town]
  mol-QA engineer-patrol        v2    [system]

# Show resolution path
bd formula show mol-agent-work --resolve
  Resolving: mol-agent-work
  ✓ Found at: ~/gt/gastown/.tickets/formulas/mol-agent-work.formula.toml
  Tier: project
  Version: 4

  Resolution path checked:
  1. [project] ~/gt/gastown/.tickets/formulas/ ← FOUND
  2. [town]    ~/gt/.tickets/formulas/
  3. [system]  <embedded>

# Override tier for testing
bd cook mol-agent-work --tier=system    # Force embedded version
bd cook mol-agent-work --tier=town      # Force town version
```

### Future (Mol Mall)

```bash
# Install from Mol Mall
gt formula install mol-code-review-strict
gt formula install mol-code-review-strict@2.0.0
gt formula install hop://acme.corp/formulas/mol-deploy

# Manage installed formulas
gt formula list --installed              # What's in town-level
gt formula upgrade mol-agent-work      # Update to latest
gt formula pin mol-agent-work@4.0.0    # Lock version
gt formula uninstall mol-code-review-strict
```

## Migration Path

### Phase 1: Resolution Order (Now)

1. Implement three-tier resolution in `bd cook`
2. Add `--resolve` flag to show resolution path
3. Update `bd formula list` to show tiers
4. Fix engineers directories (Option B)

### Phase 2: Town-Level Formulas

1. Establish `~/gt/.tickets/formulas/` as town formula location
2. Add `gt formula` commands for managing town formulas
3. Support manual installation (copy file, track in `.installed.json`)

### Phase 3: Mol Mall Integration

1. Define registry API (see mol-mall-design.md)
2. Implement `gt formula install` from remote
3. Add version pinning and upgrade flows
4. Add integrity verification (checksums, optional signing)

### Phase 4: Federation (HOP)

1. Add capability tags to formula schema
2. Track formula execution for agent accountability
3. Enable federation (cross-town formula sharing via Highway Operations Protocol)
4. Author attribution and validation records

## Related Documents

- [Mol Mall Design](mol-mall-design.md) - Registry architecture
- [Molecules](../concepts/molecules.md) - Formula → Proto → Mol lifecycle
