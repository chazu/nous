package rewriteexp

import (
	"fmt"
	"math/rand"
	"testing"
)

const testDomainsDir = "../../domains"

func TestGeneratedRobustnessTrial(t *testing.T) {
	report, err := RunRobustness(testDomainsDir, 4242, 12)
	if err != nil {
		t.Fatal(err)
	}
	if report.Recovered != report.Problems || report.UniquePromotions != report.Problems || report.FalsePromotions != 0 {
		t.Fatalf("robustness recovery = %+v", report)
	}
	if report.HeldOutCases != 12*32 || report.HeldOutFailures != 0 {
		t.Fatalf("robustness held-out = %+v", report)
	}
}

func TestCreditCurriculumShowsSpecificAndNegativeTransfer(t *testing.T) {
	report, err := RunCurriculum(testDomainsDir, 4243, 30, 4)
	if err != nil {
		t.Fatal(err)
	}
	reuse := report.Cohorts["reuse-both"]
	if reuse.Learned.Solved != reuse.Learned.Tasks || reuse.Learned.MeanEvaluations > 2 {
		t.Fatalf("credit did not prioritize exact reuse: %+v", reuse)
	}
	unrelated := report.Cohorts["unrelated"]
	if unrelated.Learned.Solved != 0 || unrelated.Exhaustive.Solved != unrelated.Exhaustive.Tasks {
		t.Fatalf("negative-transfer control did not separate learned and exhaustive search: %+v", unrelated)
	}
	if report.Cohorts["reuse-one"].Exhaustive.Solved != report.Cohorts["reuse-one"].Exhaustive.Tasks {
		t.Fatalf("reuse-one corpus was not solvable: %+v", report.Cohorts["reuse-one"])
	}
}

func TestRewriteTrialsAreDeterministic(t *testing.T) {
	first, err := Run(testDomainsDir, 9001, 4, 6, 4)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(testDomainsDir, 9001, 4, 6, 4)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := first.JSON()
	secondJSON, _ := second.JSON()
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("trial reports differ:\n%s\n%s", firstJSON, secondJSON)
	}
}

func TestGeneratedNoSolutionCorpusProducesNoPromotion(t *testing.T) {
	problem, err := generateProblem(rand.New(rand.NewSource(77)), "NoSolution")
	if err != nil {
		t.Fatal(err)
	}
	problem.Examples[0].Expected = "zzzzzzzz"
	store, _, err := executeProblem(testDomainsDir, problem)
	if err != nil {
		t.Fatal(err)
	}
	if got := promotedPrograms(store); len(got) != 0 {
		t.Fatalf("unsatisfiable corpus promoted %v", got)
	}
}

func TestAmbiguousTrainingCorpusRetainsMultipleHypotheses(t *testing.T) {
	problem := problem{
		Rules: []ruleSpec{
			{Name: "Amb-P0", Left: "a", Right: "b"},
			{Name: "Amb-P1", Left: "b", Right: "c"},
			{Name: "Amb-P2", Left: "a", Right: "c"},
			{Name: "Amb-P3", Left: "e", Right: "f"},
		},
		Examples: []example{
			{Name: "Amb-E0", Input: "a", Expected: "c"},
			{Name: "Amb-E1", Input: "aa", Expected: "cc"},
			{Name: "Amb-E2", Input: "da", Expected: "dc"},
		},
	}
	store, eng, err := executeProblem(testDomainsDir, problem)
	if err != nil {
		t.Fatal(err)
	}
	promoted := promotedPrograms(store)
	if len(promoted) < 2 {
		t.Fatalf("ambiguous corpus retained %d hypotheses, want multiple", len(promoted))
	}
	distinguished := false
	for _, name := range promoted {
		value, execErr := eng.VM.Execute(fmt.Sprintf("%q %q apply-op", "b", name))
		if execErr != nil {
			t.Fatal(execErr)
		}
		if value.AsString() != "c" {
			distinguished = true
		}
	}
	if !distinguished {
		t.Fatal("held-out input did not expose any training-consistent overfit")
	}
}

func TestUnsupportedRepeatedAndThreeStepTargetsDoNotFalsePromote(t *testing.T) {
	tests := map[string]problem{
		"repeated": {
			Rules: []ruleSpec{
				{Name: "Repeat-P0", Left: "a", Right: "aa"},
				{Name: "Repeat-P1", Left: "b", Right: "c"},
				{Name: "Repeat-P2", Left: "c", Right: "d"},
				{Name: "Repeat-P3", Left: "d", Right: "e"},
			},
			Examples: []example{
				{Name: "Repeat-E0", Input: "a", Expected: "aaaa"},
				{Name: "Repeat-E1", Input: "ba", Expected: "baaaa"},
				{Name: "Repeat-E2", Input: "aaa", Expected: "aaaaaaaaaaaa"},
			},
		},
		"three-step": {
			Rules: []ruleSpec{
				{Name: "Three-P0", Left: "ab", Right: "x"},
				{Name: "Three-P1", Left: "xc", Right: "y"},
				{Name: "Three-P2", Left: "yd", Right: "z"},
				{Name: "Three-P3", Left: "x", Right: "q"},
			},
			Examples: []example{
				{Name: "Three-E0", Input: "abcd", Expected: "z"},
				{Name: "Three-E1", Input: "zabcd", Expected: "zz"},
				{Name: "Three-E2", Input: "abcdabcd", Expected: "zz"},
			},
		},
	}
	for name, problem := range tests {
		t.Run(name, func(t *testing.T) {
			store, _, err := executeProblem(testDomainsDir, problem)
			if err != nil {
				t.Fatal(err)
			}
			if got := promotedPrograms(store); len(got) != 0 {
				t.Fatalf("unsupported target false-promoted %v", got)
			}
		})
	}
}
