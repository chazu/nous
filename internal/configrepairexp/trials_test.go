package configrepairexp

import (
	"testing"

	configvocab "github.com/chazu/nous/internal/vocab/configrepair"
)

func TestScenarioCatalogHasTwentyBalancedCases(t *testing.T) {
	problems := scenarios()
	if len(problems) != 20 {
		t.Fatalf("scenario count = %d, want 20", len(problems))
	}
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, problem := range problems {
		if seen[problem.ID] {
			t.Fatalf("duplicate scenario ID %s", problem.ID)
		}
		seen[problem.ID] = true
		counts[problem.Technology]++
		repairs := 0
		for _, spec := range problem.Assignments {
			if spec.Required {
				repairs++
			}
		}
		if repairs == 0 || repairs > 3 {
			t.Fatalf("%s target repair count = %d, want 1..3", problem.ID, repairs)
		}
		if !configvocab.ValidSchema(schemaData(problem)) {
			t.Fatalf("%s generated invalid schema", problem.ID)
		}
	}
	if counts["kubernetes"] != 10 || counts["terraform"] != 10 {
		t.Fatalf("technology counts = %v", counts)
	}
}

func TestRealityGateRunsNousAndMatchesBoundedBaseline(t *testing.T) {
	report, err := Run("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	if report.NousExpectedPlansRecovered != 20 || report.NousUniquePromotions != 20 {
		t.Fatalf("Nous recoveries/unique = %d/%d, want 20/20", report.NousExpectedPlansRecovered, report.NousUniquePromotions)
	}
	if report.HeldOutCases != 40 || report.HeldOutFailures != 0 {
		t.Fatalf("held-out cases/failures = %d/%d, want 40/0", report.HeldOutCases, report.HeldOutFailures)
	}
	if report.UnsafeCandidates == 0 || report.UnsafeCandidatesRejected != report.UnsafeCandidates {
		t.Fatalf("unsafe candidates/rejected = %d/%d", report.UnsafeCandidates, report.UnsafeCandidatesRejected)
	}
	if report.BaselineExpectedPlans != 20 || report.ExactNousBaselineAgreements != 20 {
		t.Fatalf("baseline recoveries/agreements = %d/%d, want 20/20", report.BaselineExpectedPlans, report.ExactNousBaselineAgreements)
	}
}
