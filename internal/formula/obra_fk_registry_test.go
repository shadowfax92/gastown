package formula

import (
	"slices"
	"testing"
)

var requiredObraFKSkills = []string{
	"obra-fk-design",
	"obra-fk-exec",
	"obra-fk-debug",
	"obra-fk-debug-fix",
	"obra-fk-dev-full",
	"obra-fk-lite",
	"obra-fk-lite-plan",
	"obra-fk-lite-exec",
}

func TestValidateObraFKSkillFormulaRegistry(t *testing.T) {
	if err := ValidateObraFKSkillFormulaRegistry(); err != nil {
		t.Fatalf("ValidateObraFKSkillFormulaRegistry() error = %v", err)
	}
}

func TestObraFKSkillFormulaRegistryCoversRequiredSkills(t *testing.T) {
	for _, skill := range requiredObraFKSkills {
		contract, ok := ObraFKSkillFormulaBySkill(skill)
		if !ok {
			t.Fatalf("missing contract for %s", skill)
		}
		if contract.Skill != skill {
			t.Fatalf("lookup for %s returned %s", skill, contract.Skill)
		}
		if !slices.Contains(contract.Sessions, ObraFKSessionCodex) {
			t.Errorf("%s does not support Codex sessions", skill)
		}
		if !slices.Contains(contract.Sessions, ObraFKSessionClaude) {
			t.Errorf("%s does not support Claude sessions", skill)
		}
		if !slices.Contains(contract.InvocationNames, skill) {
			t.Errorf("%s invocations do not include the skill name", skill)
		}
	}
}

func TestObraFKSkillFormulaRegistryIsDefensiveCopy(t *testing.T) {
	registry := ObraFKSkillFormulaRegistry()
	if len(registry) == 0 {
		t.Fatal("registry is empty")
	}
	registry[0].Skill = "mutated"
	registry[0].InvocationNames[0] = "mutated"
	registry[0].Sessions[0] = "mutated"

	fresh := ObraFKSkillFormulaRegistry()
	if fresh[0].Skill == "mutated" {
		t.Fatal("registry returned shared contract structs")
	}
	if fresh[0].InvocationNames[0] == "mutated" {
		t.Fatal("registry returned shared invocation slices")
	}
	if fresh[0].Sessions[0] == "mutated" {
		t.Fatal("registry returned shared session slices")
	}
}

func TestObraFKLiteVariantsUseSourceControlledFormulas(t *testing.T) {
	liteSkills := []string{
		"obra-fk-lite",
		"obra-fk-lite-plan",
		"obra-fk-lite-exec",
	}

	for _, skill := range liteSkills {
		contract, ok := ObraFKSkillFormulaBySkill(skill)
		if !ok {
			t.Fatalf("missing contract for %s", skill)
		}
		if contract.Formula != skill {
			t.Errorf("%s formula = %q, want %q", skill, contract.Formula, skill)
		}
		if contract.Availability != ObraFKFormulaSourceControlled {
			t.Errorf("%s availability = %q, want %q", skill, contract.Availability, ObraFKFormulaSourceControlled)
		}
		if contract.FallbackFormula == "" {
			t.Errorf("%s fallback formula is empty", skill)
		}
	}
}

func TestObraFKFullVariantsArePlannedWithFallbacks(t *testing.T) {
	fullSkills := []string{
		"obra-fk-design",
		"obra-fk-exec",
		"obra-fk-debug",
		"obra-fk-debug-fix",
		"obra-fk-dev-full",
	}

	for _, skill := range fullSkills {
		contract, ok := ObraFKSkillFormulaBySkill(skill)
		if !ok {
			t.Fatalf("missing contract for %s", skill)
		}
		if contract.Availability != ObraFKFormulaPlanned {
			t.Errorf("%s availability = %q, want %q", skill, contract.Availability, ObraFKFormulaPlanned)
		}
		if contract.FallbackFormula == "" {
			t.Errorf("%s fallback formula is empty", skill)
		}
		if contract.FallbackBehavior == "" {
			t.Errorf("%s fallback behavior is empty", skill)
		}
	}
}

func TestValidateObraFKSkillFormulaContractsRejectsDuplicateInvocation(t *testing.T) {
	contracts := ObraFKSkillFormulaRegistry()
	contracts[1].InvocationNames = append(contracts[1].InvocationNames, contracts[0].InvocationNames[0])

	if err := ValidateObraFKSkillFormulaContracts(contracts); err == nil {
		t.Fatal("ValidateObraFKSkillFormulaContracts() error = nil, want duplicate invocation error")
	}
}

func TestValidateObraFKSkillFormulaContractsRejectsUnknownFallback(t *testing.T) {
	contracts := ObraFKSkillFormulaRegistry()
	contracts[0].FallbackFormula = "missing-formula"

	if err := ValidateObraFKSkillFormulaContracts(contracts); err == nil {
		t.Fatal("ValidateObraFKSkillFormulaContracts() error = nil, want unknown fallback error")
	}
}
