package ruleinductionexp

import "testing"

func TestPreregisteredDevelopmentGeneratorFeasibility(t *testing.T) {
	for index := 0; index < 16; index++ {
		seed := int64(11001 + index)
		cohort := CohortForIndex(index)
		fixture, err := Generate("development", seed, cohort)
		if err != nil {
			t.Fatalf("seed %d cohort %s: %v", seed, cohort, err)
		}
		if len(fixture.Stage1) > 24 || len(fixture.Stage2) > 24 {
			t.Fatalf("seed %d examples %d/%d", seed, len(fixture.Stage1), len(fixture.Stage2))
		}
	}
}

func TestNoSolutionControlsAreOutsideGrammar(t *testing.T) {
	for seed := int64(51001); seed <= 51008; seed++ {
		fixture, err := GenerateNoSolution(seed)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if fixture.Cohort != NoSolution || len(exactCodes(fixture.Background, fixture.Target2)) != 0 {
			t.Fatalf("seed %d is not a no-solution fixture", seed)
		}
	}
}

func TestPreregisteredNonLockedPanelsGenerate(t *testing.T) {
	for _, panel := range []struct {
		name  string
		start int64
		count int
	}{{"training", 21001, 64}, {"validation", 31001, 32}} {
		for index := 0; index < panel.count; index++ {
			if _, err := Generate(panel.name, panel.start+int64(index), CohortForIndex(index)); err != nil {
				t.Fatalf("%s seed %d: %v", panel.name, panel.start+int64(index), err)
			}
		}
	}
}
