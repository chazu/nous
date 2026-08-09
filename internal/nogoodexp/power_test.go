package nogoodexp

import (
	"slices"
	"testing"
)

func TestDevelopmentPowerSimulationIsDeterministicAndUsesLockedCells(t *testing.T) {
	execution := PanelExecution{Panel: "development", Role: "primary", AcquisitionWork: 100}
	for _, policy := range RequiredPolicies {
		execution.Policies = append(execution.Policies, PolicyExecution{Policy: policy})
	}
	learnedIndex, macIndex := slices.Index(RequiredPolicies, "nous-generalized"), slices.Index(RequiredPolicies, "mac-cbj")
	for ordinal := 0; ordinal < DevelopmentTaskCount; ordinal++ {
		cohort := "reusable"
		if ordinal >= 88 {
			cohort = "independent-unsat"
		} else if ordinal >= 80 {
			cohort = "irrelevant"
		} else if ordinal >= 56 {
			cohort = "near-miss"
		}
		execution.Policies[macIndex].Tasks = append(execution.Policies[macIndex].Tasks, TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: 200})
		learnedWork := int64(100)
		if cohort != "reusable" {
			learnedWork = 205
		}
		execution.Policies[learnedIndex].Tasks = append(execution.Policies[learnedIndex].Tasks, TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: learnedWork})
	}
	first, err := estimateDevelopmentPower(execution, 3, 40)
	if err != nil {
		t.Fatal(err)
	}
	second, err := estimateDevelopmentPower(execution, 3, 40)
	if err != nil || first != second {
		t.Fatalf("power simulation is not deterministic: %#v %#v %v", first, second, err)
	}
	if first.Replicates != 3 || first.Fraction.Denominator != 3 {
		t.Fatalf("power shape = %#v", first)
	}
	count := 0
	for _, cell := range orderedCells() {
		count += lockedCellCount(cell)
	}
	if count != LockedTaskCount {
		t.Fatalf("locked cell counts sum to %d", count)
	}
}
