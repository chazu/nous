package nogoodbaseline

import (
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/nogoodoracle"
)

func TestMACCBJMatchesIndependentOracleAcrossDevelopment(t *testing.T) {
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	minimumReusable := int64(^uint64(0) >> 1)
	for _, task := range tasks {
		got, err := MACCBJ(task.ProblemJSON, Literal{Variable: task.Decision.Variable, Color: task.Decision.Color})
		if err != nil {
			t.Fatalf("task %d: %v", task.Ordinal, err)
		}
		want, err := nogoodoracle.Enumerate(task.ProblemJSON, nogoodoracle.Literal{Variable: task.Decision.Variable, Color: task.Decision.Color})
		if err != nil {
			t.Fatal(err)
		}
		if got.Satisfied != want.Satisfiable {
			t.Fatalf("task %d cohort %s: MAC=%v oracle=%v", task.Ordinal, task.Cohort, got.Satisfied, want.Satisfiable)
		}
		if got.Satisfied && !containsSolution(want.Solutions, got.Witness) {
			t.Fatalf("task %d returned non-oracle witness %v", task.Ordinal, got.Witness)
		}
		if got.Work != int64(len(got.Events)) {
			t.Fatalf("task %d work=%d events=%d", task.Ordinal, got.Work, len(got.Events))
		}
		var vector int64
		for _, count := range got.Vector {
			vector += count
		}
		if vector != got.Work {
			t.Fatalf("task %d vector=%d work=%d", task.Ordinal, vector, got.Work)
		}
		if task.Cohort == nogoodfixture.Reusable && got.Work < minimumReusable {
			minimumReusable = got.Work
		}
	}
	if minimumReusable < 150 {
		t.Fatalf("minimum reusable work=%d, want >=150", minimumReusable)
	}
	t.Logf("minimum reusable MAC-CBJ work=%d", minimumReusable)
}

func TestConventionalPoliciesMatchIndependentOracleAcrossDevelopment(t *testing.T) {
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		t.Fatal(err)
	}
	policies := map[string]func([]byte, Literal) (Result, error){
		"chronological":    Chronological,
		"forward-checking": ForwardChecking,
	}
	for name, policy := range policies {
		t.Run(name, func(t *testing.T) {
			for _, task := range tasks {
				got, err := policy(task.ProblemJSON, Literal{Variable: task.Decision.Variable, Color: task.Decision.Color})
				if err != nil {
					t.Fatalf("task %d: %v", task.Ordinal, err)
				}
				want, err := nogoodoracle.Enumerate(task.ProblemJSON, nogoodoracle.Literal{Variable: task.Decision.Variable, Color: task.Decision.Color})
				if err != nil {
					t.Fatal(err)
				}
				if got.Satisfied != want.Satisfiable || got.Satisfied && !containsSolution(want.Solutions, got.Witness) {
					t.Fatalf("task %d cohort %s: result=%v witness=%v oracle=%v", task.Ordinal, task.Cohort, got.Satisfied, got.Witness, want.Satisfiable)
				}
				if got.Work != int64(len(got.Events)) {
					t.Fatalf("task %d work/event mismatch %d/%d", task.Ordinal, got.Work, len(got.Events))
				}
			}
		})
	}
}

func containsSolution(solutions [][]int, witness []int) bool {
	for _, solution := range solutions {
		if len(solution) != len(witness) {
			continue
		}
		equal := true
		for index := range solution {
			if solution[index] != witness[index] {
				equal = false
				break
			}
		}
		if equal {
			return true
		}
	}
	return false
}
