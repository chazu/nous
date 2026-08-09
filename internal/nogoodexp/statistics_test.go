package nogoodexp

import (
	"math"
	"slices"
	"testing"
)

func TestDevelopmentRandomizationGoldenDraws(t *testing.T) {
	rng := statisticsStream("development", 832001, 0, "randomization/nous-vs-mac")
	got := make([]uint64, 8)
	for index := range got {
		got[index] = rng.Uint64N(2)
	}
	want := []uint64{0, 1, 1, 0, 1, 0, 0, 0}
	if !slices.Equal(got, want) {
		t.Fatalf("replicate-zero draws = %v, want %v", got, want)
	}
}

func TestInferenceRejectsSignedOverflow(t *testing.T) {
	execution := PanelExecution{Panel: "development", Role: "primary", AcquisitionWork: math.MaxInt64}
	for _, policy := range RequiredPolicies {
		execution.Policies = append(execution.Policies, PolicyExecution{Policy: policy})
	}
	learnedIndex, macIndex := slices.Index(RequiredPolicies, "nous-generalized"), slices.Index(RequiredPolicies, "mac-cbj")
	for ordinal := 0; ordinal < 96; ordinal++ {
		cohort := "reusable"
		if ordinal >= 88 {
			cohort = "independent-unsat"
		} else if ordinal >= 80 {
			cohort = "irrelevant"
		} else if ordinal >= 56 {
			cohort = "near-miss"
		}
		execution.Policies[macIndex].Tasks = append(execution.Policies[macIndex].Tasks, TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: 1})
		execution.Policies[learnedIndex].Tasks = append(execution.Policies[learnedIndex].Tasks, TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: 2})
	}
	if _, err := InferDevelopment(execution); err == nil {
		t.Fatal("overflowing inference input was accepted")
	}
}

func TestInferenceIsDeterministicAndUsesAllFrozenStrata(t *testing.T) {
	execution := PanelExecution{Panel: "development", Role: "primary", AcquisitionWork: 500}
	for _, policy := range RequiredPolicies {
		execution.Policies = append(execution.Policies, PolicyExecution{Policy: policy})
	}
	learnedIndex, macIndex := slices.Index(RequiredPolicies, "nous-generalized"), slices.Index(RequiredPolicies, "mac-cbj")
	for ordinal := 0; ordinal < 96; ordinal++ {
		cohort := "reusable"
		if ordinal >= 88 {
			cohort = "independent-unsat"
		} else if ordinal >= 80 {
			cohort = "irrelevant"
		} else if ordinal >= 56 {
			cohort = "near-miss"
		}
		mac := TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: 200, PruneSound: true}
		learnedWork := int64(125)
		if cohort != "reusable" {
			learnedWork = 215
		}
		learned := TaskOutcome{Ordinal: ordinal, Cohort: cohort, Work: learnedWork, PruneSound: true}
		execution.Policies[macIndex].Tasks = append(execution.Policies[macIndex].Tasks, mac)
		execution.Policies[learnedIndex].Tasks = append(execution.Policies[learnedIndex].Tasks, learned)
	}
	first, err := InferDevelopment(execution)
	if err != nil {
		t.Fatal(err)
	}
	second, err := InferDevelopment(execution)
	if err != nil || first != second {
		t.Fatalf("inference not deterministic: %#v %#v %v", first, second, err)
	}
	if first.Point.Denominator != 96*200 || first.RandomizationP.Denominator != 10001 {
		t.Fatalf("inference denominators = %#v", first)
	}
}
