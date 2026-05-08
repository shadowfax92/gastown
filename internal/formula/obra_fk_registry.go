package formula

import (
	"fmt"
	"strings"
)

// ObraFKFormulaAvailability describes whether a formula is already part of
// the source-controlled formula set or is a declared future contract.
type ObraFKFormulaAvailability string

const (
	// ObraFKFormulaSourceControlled means the formula is expected to be
	// available through normal Gas Town formula resolution.
	ObraFKFormulaSourceControlled ObraFKFormulaAvailability = "source-controlled"

	// ObraFKFormulaPlanned means the skill contract is reserved, but callers
	// must follow the fallback formula until the dedicated formula lands.
	ObraFKFormulaPlanned ObraFKFormulaAvailability = "planned"
)

// ObraFKSession identifies an agent runtime that can consume the Obra FK
// skill/formula mapping.
type ObraFKSession string

const (
	// ObraFKSessionCodex is the Codex runtime.
	ObraFKSessionCodex ObraFKSession = "codex"
	// ObraFKSessionClaude is the Claude runtime.
	ObraFKSessionClaude ObraFKSession = "claude"
)

// ObraFKSkillFormulaContract defines how an Obra FK skill maps to a Gas Town
// formula and how runtimes should fall back when that formula is unavailable.
type ObraFKSkillFormulaContract struct {
	Skill                  string
	Formula                string
	Availability           ObraFKFormulaAvailability
	InvocationNames        []string
	Sessions               []ObraFKSession
	Phase                  string
	RequiresAcceptedDesign bool
	FallbackFormula        string
	FallbackBehavior       string
}

var obraFKSkillFormulaRegistry = []ObraFKSkillFormulaContract{
	{
		Skill:                  "obra-fk-design",
		Formula:                "obra-fk-design",
		Availability:           ObraFKFormulaPlanned,
		InvocationNames:        []string{"obra-fk-design", "/obra-fk-design"},
		Sessions:               []ObraFKSession{ObraFKSessionCodex, ObraFKSessionClaude},
		Phase:                  "design",
		RequiresAcceptedDesign: false,
		FallbackFormula:        "obra-fk-lite-plan",
		FallbackBehavior:       "Run obra-fk-lite-plan to normalize the vision, brainstorm options, accept the best design, and create executable beads.",
	},
	{
		Skill:                  "obra-fk-exec",
		Formula:                "obra-fk-exec",
		Availability:           ObraFKFormulaPlanned,
		InvocationNames:        []string{"obra-fk-exec", "/obra-fk-exec"},
		Sessions:               []ObraFKSession{ObraFKSessionCodex, ObraFKSessionClaude},
		Phase:                  "execution",
		RequiresAcceptedDesign: true,
		FallbackFormula:        "obra-fk-lite-exec",
		FallbackBehavior:       "Run obra-fk-lite-exec against the accepted design and current bead; keep scope tight, verify, commit, and submit through gt done.",
	},
	{
		Skill:                  "obra-fk-debug",
		Formula:                "obra-fk-debug",
		Availability:           ObraFKFormulaPlanned,
		InvocationNames:        []string{"obra-fk-debug", "/obra-fk-debug"},
		Sessions:               []ObraFKSession{ObraFKSessionCodex, ObraFKSessionClaude},
		Phase:                  "debug",
		RequiresAcceptedDesign: false,
		FallbackFormula:        "tdd-cycle",
		FallbackBehavior:       "Run tdd-cycle as the debugging loop: reproduce the failure, add or narrow a test, make the smallest fix, and rerun the target gate.",
	},
	{
		Skill:                  "obra-fk-debug-fix",
		Formula:                "obra-fk-debug-fix",
		Availability:           ObraFKFormulaPlanned,
		InvocationNames:        []string{"obra-fk-debug-fix", "/obra-fk-debug-fix"},
		Sessions:               []ObraFKSession{ObraFKSessionCodex, ObraFKSessionClaude},
		Phase:                  "debug-fix",
		RequiresAcceptedDesign: false,
		FallbackFormula:        "tdd-cycle",
		FallbackBehavior:       "Run tdd-cycle with the failure as the acceptance target; commit only the fix and any focused regression test.",
	},
	{
		Skill:                  "obra-fk-dev-full",
		Formula:                "obra-fk-dev-full",
		Availability:           ObraFKFormulaPlanned,
		InvocationNames:        []string{"obra-fk-dev-full", "/obra-fk-dev-full"},
		Sessions:               []ObraFKSession{ObraFKSessionCodex, ObraFKSessionClaude},
		Phase:                  "full-development",
		RequiresAcceptedDesign: false,
		FallbackFormula:        "obra-fk-lite",
		FallbackBehavior:       "Run obra-fk-lite end to end, then dispatch follow-up execution beads rather than attempting the full future workflow in one session.",
	},
	{
		Skill:                  "obra-fk-lite",
		Formula:                "obra-fk-lite",
		Availability:           ObraFKFormulaSourceControlled,
		InvocationNames:        []string{"obra-fk-lite", "/obra-fk-lite"},
		Sessions:               []ObraFKSession{ObraFKSessionCodex, ObraFKSessionClaude},
		Phase:                  "lite-end-to-end",
		RequiresAcceptedDesign: false,
		FallbackFormula:        "mol-polecat-work",
		FallbackBehavior:       "If the lite formula is unavailable, run mol-polecat-work and manually perform the planning and first execution dispatch described by the skill.",
	},
	{
		Skill:                  "obra-fk-lite-plan",
		Formula:                "obra-fk-lite-plan",
		Availability:           ObraFKFormulaSourceControlled,
		InvocationNames:        []string{"obra-fk-lite-plan", "/obra-fk-lite-plan"},
		Sessions:               []ObraFKSession{ObraFKSessionCodex, ObraFKSessionClaude},
		Phase:                  "lite-planning",
		RequiresAcceptedDesign: false,
		FallbackFormula:        "mol-idea-to-plan",
		FallbackBehavior:       "Run mol-idea-to-plan and persist the accepted design, breakdown, and bead dependencies that obra-fk-lite-plan would have produced.",
	},
	{
		Skill:                  "obra-fk-lite-exec",
		Formula:                "obra-fk-lite-exec",
		Availability:           ObraFKFormulaSourceControlled,
		InvocationNames:        []string{"obra-fk-lite-exec", "/obra-fk-lite-exec"},
		Sessions:               []ObraFKSession{ObraFKSessionCodex, ObraFKSessionClaude},
		Phase:                  "lite-execution",
		RequiresAcceptedDesign: true,
		FallbackFormula:        "mol-polecat-work",
		FallbackBehavior:       "Run mol-polecat-work using the accepted design as context; implement only the hooked bead, verify, commit, and submit with gt done.",
	},
}

var obraFKGenericFallbackFormulas = map[string]struct{}{
	"mol-idea-to-plan": {},
	"mol-polecat-work": {},
	"tdd-cycle":        {},
}

// ObraFKSkillFormulaRegistry returns a defensive copy of the Obra FK registry.
func ObraFKSkillFormulaRegistry() []ObraFKSkillFormulaContract {
	out := make([]ObraFKSkillFormulaContract, len(obraFKSkillFormulaRegistry))
	for i, contract := range obraFKSkillFormulaRegistry {
		out[i] = cloneObraFKSkillFormulaContract(contract)
	}
	return out
}

// ObraFKSkillFormulaBySkill looks up a contract by skill name.
func ObraFKSkillFormulaBySkill(skill string) (ObraFKSkillFormulaContract, bool) {
	for _, contract := range obraFKSkillFormulaRegistry {
		if contract.Skill == skill {
			return cloneObraFKSkillFormulaContract(contract), true
		}
	}
	return ObraFKSkillFormulaContract{}, false
}

// ValidateObraFKSkillFormulaRegistry validates the built-in Obra FK registry.
func ValidateObraFKSkillFormulaRegistry() error {
	return ValidateObraFKSkillFormulaContracts(obraFKSkillFormulaRegistry)
}

// ValidateObraFKSkillFormulaContracts validates a registry snapshot.
func ValidateObraFKSkillFormulaContracts(contracts []ObraFKSkillFormulaContract) error {
	var problems []string
	skills := make(map[string]struct{}, len(contracts))
	formulas := make(map[string]struct{}, len(contracts))
	invocations := make(map[string]string)

	for i, contract := range contracts {
		context := fmt.Sprintf("contract[%d]", i)
		if contract.Skill == "" {
			problems = append(problems, context+": skill is required")
		} else if _, exists := skills[contract.Skill]; exists {
			problems = append(problems, context+": duplicate skill "+contract.Skill)
		} else {
			skills[contract.Skill] = struct{}{}
		}

		if contract.Formula == "" {
			problems = append(problems, context+": formula is required")
		} else {
			formulas[contract.Formula] = struct{}{}
		}

		switch contract.Availability {
		case ObraFKFormulaSourceControlled, ObraFKFormulaPlanned:
		default:
			problems = append(problems, context+": unknown availability "+string(contract.Availability))
		}

		if len(contract.InvocationNames) == 0 {
			problems = append(problems, context+": at least one invocation name is required")
		}
		seenLocalInvocation := make(map[string]struct{}, len(contract.InvocationNames))
		for _, invocation := range contract.InvocationNames {
			if invocation == "" {
				problems = append(problems, context+": invocation name is required")
				continue
			}
			if _, exists := seenLocalInvocation[invocation]; exists {
				problems = append(problems, context+": duplicate invocation "+invocation)
				continue
			}
			seenLocalInvocation[invocation] = struct{}{}
			if owner, exists := invocations[invocation]; exists {
				problems = append(problems, context+": invocation "+invocation+" also belongs to "+owner)
			} else {
				invocations[invocation] = contract.Skill
			}
		}

		if len(contract.Sessions) == 0 {
			problems = append(problems, context+": at least one session is required")
		}
		for _, session := range contract.Sessions {
			switch session {
			case ObraFKSessionCodex, ObraFKSessionClaude:
			default:
				problems = append(problems, context+": unknown session "+string(session))
			}
		}

		if contract.Phase == "" {
			problems = append(problems, context+": phase is required")
		}
		if contract.FallbackFormula == "" {
			problems = append(problems, context+": fallback formula is required")
		}
		if contract.FallbackBehavior == "" {
			problems = append(problems, context+": fallback behavior is required")
		}
	}

	for i, contract := range contracts {
		if contract.FallbackFormula == "" {
			continue
		}
		if _, knownFormula := formulas[contract.FallbackFormula]; knownFormula {
			continue
		}
		if _, genericFormula := obraFKGenericFallbackFormulas[contract.FallbackFormula]; genericFormula {
			continue
		}
		problems = append(problems, fmt.Sprintf("contract[%d]: unknown fallback formula %s", i, contract.FallbackFormula))
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid Obra FK skill/formula registry: %s", strings.Join(problems, "; "))
	}
	return nil
}

func cloneObraFKSkillFormulaContract(contract ObraFKSkillFormulaContract) ObraFKSkillFormulaContract {
	contract.InvocationNames = append([]string(nil), contract.InvocationNames...)
	contract.Sessions = append([]ObraFKSession(nil), contract.Sessions...)
	return contract
}
