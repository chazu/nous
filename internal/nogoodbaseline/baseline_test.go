package nogoodbaseline

import (
	"reflect"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/nogoodoracle"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

func TestMACCBJMatchesIndependentOracleAcrossDevelopment(t *testing.T) {
	tasks, err := nogoodfixture.DevelopmentPanel()
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

func TestMACCBJRootWrapperCoversEveryFailureOrigin(t *testing.T) {
	problemJSON := func(problem nogoods.Problem) []byte {
		encoded, err := problem.CanonicalJSON()
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}
	colors := []string{"c0", "c1", "c2"}
	cases := []struct {
		name    string
		problem nogoods.Problem
	}{
		{"assigned-edge", nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: colors, Variables: []nogoods.Variable{{Alias: "a", Domain: []int{0, 1}}, {Alias: "b", Domain: []int{0}}}, Edges: []nogoods.Edge{{Left: 0, Right: 1}}, Assignment: []nogoods.Literal{{Variable: 1, Color: 0}}}},
		{"initial-ac3", nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: colors, Variables: []nogoods.Variable{{Alias: "a", Domain: []int{0, 1}}, {Alias: "b", Domain: []int{0}}}, Edges: []nogoods.Edge{{Left: 0, Right: 1}}, Assignment: []nogoods.Literal{}}},
		{"recursive", nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: colors, Variables: []nogoods.Variable{{Alias: "a", Domain: []int{0, 1, 2}}, {Alias: "b", Domain: []int{0, 1, 2}}, {Alias: "c", Domain: []int{0, 1, 2}}, {Alias: "d", Domain: []int{0, 1, 2}}}, Edges: []nogoods.Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}, {Left: 0, Right: 3}, {Left: 1, Right: 2}, {Left: 1, Right: 3}, {Left: 2, Right: 3}}, Assignment: []nogoods.Literal{}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := MACCBJ(problemJSON(testCase.problem), Literal{Variable: 0, Color: 0})
			if err != nil || result.Satisfied {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			transitions := make([]string, len(result.Events))
			for index, event := range result.Events {
				transitions[index] = event.Transition
			}
			if !slices.Contains(transitions, "nogood-write") || !slices.Contains(transitions, "root-project") || !slices.Equal(transitions[len(transitions)-3:], []string{"decision-unbind", "terminal-classification", "terminal-record-write"}) {
				t.Fatalf("root wrapper trace is incomplete: %v", transitions)
			}
		})
	}
}

func TestMACCBJRetainsAndConsultsTaskLocalExactNogoods(t *testing.T) {
	for colorCount := 2; colorCount <= 4; colorCount++ {
		for variableCount := colorCount + 1; variableCount <= min(8, colorCount+3); variableCount++ {
			problem := nogoods.Problem{Version: nogoods.ProblemVersion, Assignment: []nogoods.Literal{}}
			for color := 0; color < colorCount; color++ {
				problem.ColorAliases = append(problem.ColorAliases, "c"+string(rune('0'+color)))
			}
			for variable := 0; variable < variableCount; variable++ {
				domain := make([]int, colorCount)
				for color := range domain {
					domain[color] = color
				}
				problem.Variables = append(problem.Variables, nogoods.Variable{Alias: "v" + string(rune('a'+variable)), Domain: domain})
			}
			for left := 0; left < variableCount; left++ {
				for right := left + 1; right < variableCount; right++ {
					problem.Edges = append(problem.Edges, nogoods.Edge{Left: left, Right: right})
				}
			}
			if len(problem.Edges) > nogoods.MaxEdges {
				continue
			}
			encoded, err := problem.CanonicalJSON()
			if err != nil {
				t.Fatal(err)
			}
			result, err := MACCBJ(encoded, Literal{Variable: 0, Color: 0})
			if err != nil {
				t.Fatal(err)
			}
			write := -1
			for index, event := range result.Events {
				if event.Transition == "nogood-write" {
					write = index
				}
				if event.Transition == "nogood-lookup" && write >= 0 && index > write {
					return
				}
			}
		}
	}
	t.Fatal("bounded clique microtraces never consulted a retained exact nogood")
}

func TestMACCBJResumeIsExactPostDecisionContinuation(t *testing.T) {
	tasks, err := nogoodfixture.DevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		decision := Literal{Variable: task.Decision.Variable, Color: task.Decision.Color}
		standalone, err := MACCBJ(task.ProblemJSON, decision)
		if err != nil {
			t.Fatal(err)
		}
		resumed, err := MACCBJResume(task.ProblemJSON, decision)
		if err != nil {
			t.Fatal(err)
		}
		problem, err := parse(task.ProblemJSON)
		if err != nil {
			t.Fatal(err)
		}
		prefixEvents := 3 + 2*(len(problem.Variables[decision.Variable].Domain)-1)
		if standalone.Satisfied != resumed.Satisfied || !reflect.DeepEqual(standalone.Witness, resumed.Witness) || !reflect.DeepEqual(standalone.Events[prefixEvents:], resumed.Events) {
			t.Fatalf("task %d resume is not the exact post-decision continuation", task.Ordinal)
		}
	}
}

func TestConventionalPoliciesMatchIndependentOracleAcrossDevelopment(t *testing.T) {
	tasks, err := nogoodfixture.DevelopmentPanel()
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
