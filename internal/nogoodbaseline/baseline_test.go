package nogoodbaseline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
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
		wantErr string
	}{
		{"assigned-edge", nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: colors, Variables: []nogoods.Variable{{Alias: "a", Domain: []int{0, 1}}, {Alias: "b", Domain: []int{0, 1}}, {Alias: "c", Domain: []int{0}}}, Edges: []nogoods.Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}}, Assignment: []nogoods.Literal{{Variable: 1, Color: 1}, {Variable: 2, Color: 0}}}, "[2]"},
		{"initial-ac3", nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: colors, Variables: []nogoods.Variable{{Alias: "a", Domain: []int{0, 1}}, {Alias: "b", Domain: []int{0}}}, Edges: []nogoods.Edge{{Left: 0, Right: 1}}, Assignment: []nogoods.Literal{}}, ""},
		{"recursive", nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: colors, Variables: []nogoods.Variable{{Alias: "a", Domain: []int{0, 1, 2}}, {Alias: "b", Domain: []int{0, 1, 2}}, {Alias: "c", Domain: []int{0, 1, 2}}, {Alias: "d", Domain: []int{0, 1, 2}}}, Edges: []nogoods.Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}, {Left: 0, Right: 3}, {Left: 1, Right: 2}, {Left: 1, Right: 3}, {Left: 2, Right: 3}}, Assignment: []nogoods.Literal{}}, ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := MACCBJ(problemJSON(testCase.problem), Literal{Variable: 0, Color: 0})
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("assigned-edge projection error=%v, want exact conflicting endpoint %s", err, testCase.wantErr)
				}
				return
			}
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
			lastWrite := -1
			for index, event := range result.Events {
				if event.Transition == "nogood-write" {
					lastWrite = index
				}
			}
			if lastWrite < 0 || !slices.Equal(result.Events[lastWrite].Operands, []int{0, 0}) {
				t.Fatalf("root wrapper sealed operands %v, want distinguished literal [0 0]", result.Events[lastWrite].Operands)
			}
		})
	}
}

func TestMACCBJRetainsAndScansTaskLocalExactNogoods(t *testing.T) {
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

func TestMACCBJCommittedGoldenMicrotraces(t *testing.T) {
	tasks, err := nogoodfixture.DevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	canonical := func(problem nogoods.Problem) []byte {
		encoded, encodeErr := problem.CanonicalJSON()
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		return encoded
	}
	edge := nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: []string{"c0", "c1"}, Variables: []nogoods.Variable{{Alias: "a", Domain: []int{0, 1}}, {Alias: "b", Domain: []int{0, 1}}}, Edges: []nogoods.Edge{{Left: 0, Right: 1}}, Assignment: []nogoods.Literal{}}
	singleton := nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: []string{"c0", "c1"}, Variables: []nogoods.Variable{{Alias: "a", Domain: []int{0, 1}}, {Alias: "b", Domain: []int{0}}}, Edges: []nogoods.Edge{{Left: 0, Right: 1}}, Assignment: []nogoods.Literal{}}
	type golden struct {
		name     string
		problem  []byte
		decision Literal
		vector   [12]int64
		digest   string
	}
	goldens := []golden{
		{name: "satisfiable-edge", problem: canonical(edge), decision: Literal{0, 0}, vector: [12]int64{0, 3, 5, 5, 6, 6, 1, 0, 0, 0, 2, 1}, digest: "6d7fe652410be17e3e34a1596603d854fa79282d305965b583185713aca0d150"},
		{name: "unsatisfiable-equal-singleton-edge", problem: canonical(singleton), decision: Literal{0, 0}, vector: [12]int64{0, 1, 3, 1, 6, 3, 3, 0, 0, 0, 1, 3}, digest: "a3877c6fb7c3a8a682102f8d4cc61c70e77c0c87426ec854b5a5c151677a7fc1"},
		{name: "blocked-pair", problem: tasks[0].ProblemJSON, decision: Literal{tasks[0].Decision.Variable, tasks[0].Decision.Color}, vector: [12]int64{0, 1, 3, 89, 27, 73, 11, 0, 0, 0, 1, 3}, digest: "6f6028bfd3b74384afb5d04b1c6236525dbb79e527ce15aa04c1f0a2210518a4"},
		{name: "nonchronological-backjump", problem: tasks[88].ProblemJSON, decision: Literal{tasks[88].Decision.Variable, tasks[88].Decision.Color}, vector: [12]int64{0, 1, 6, 1, 12, 6, 2, 0, 0, 0, 1, 3}, digest: "ee07dc4d48a8ed0d766eb69d7c92ce9a966cb15a6989df05c6a1620b9b16c456"},
		{name: "sibling-activation-conflict-isolation", problem: canonical(cliqueProblem(3, 4)), decision: Literal{0, 0}, vector: [12]int64{0, 1, 9, 43, 39, 61, 33, 0, 0, 0, 1, 8}, digest: "7b8efaed1a0bf8c91ebc2959dc7a38c1720e04d3990c4f9d7c7328f57f1bcfc7"},
	}
	for _, item := range goldens {
		t.Run(item.name, func(t *testing.T) {
			result, runErr := MACCBJ(item.problem, item.decision)
			if runErr != nil {
				t.Fatal(runErr)
			}
			material, marshalErr := json.Marshal(struct {
				Satisfied bool      `json:"satisfied"`
				Witness   []int     `json:"witness,omitempty"`
				Vector    [12]int64 `json:"vector"`
				Events    []Event   `json:"events"`
			}{result.Satisfied, result.Witness, result.Vector, result.Events})
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			digest := sha256.Sum256(material)
			gotDigest := hex.EncodeToString(digest[:])
			if result.Vector != item.vector || gotDigest != item.digest {
				t.Fatalf("golden drift vector=%v digest=%s", result.Vector, gotDigest)
			}
			transitions := make([]string, len(result.Events))
			for index, event := range result.Events {
				transitions[index] = event.Transition
			}
			switch item.name {
			case "satisfiable-edge":
				if !slices.Contains(transitions, "complete-domain-read") || !slices.Contains(transitions, "complete-inequality") {
					t.Fatal("complete-assignment audit is absent from satisfiable golden")
				}
			case "nonchronological-backjump":
				if !slices.Contains(transitions, "backjump") {
					t.Fatal("nonchronological golden contains no backjump")
				}
			}
		})
	}
}

func TestTaskLocalExactNogoodHitCommittedGolden(t *testing.T) {
	problem := cliqueProblem(3, 4)
	encoded, err := problem.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{}
	solver := newSolver(parsed, &result)
	solver.assignment[0] = 0
	solver.domains[0] = []int{0}
	solver.materializeExactNogood(map[int]bool{0: true})
	witness, failure := solver.search()
	if witness != nil || !failure[0] {
		t.Fatalf("preloaded task-local exact nogood did not fire: witness=%v failure=%v", witness, failure)
	}
	material, err := json.Marshal(struct {
		Vector [12]int64 `json:"vector"`
		Events []Event   `json:"events"`
	}{result.Vector, result.Events})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(material)
	gotDigest := hex.EncodeToString(digest[:])
	wantVector := [12]int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	const wantDigest = "2ffc89abb56f6fe64b7a03302ecd6ac974ad628ecb0feb71ea94c679b45ad798"
	if result.Vector != wantVector || gotDigest != wantDigest {
		t.Fatalf("task-local hit golden drift vector=%v digest=%s", result.Vector, gotDigest)
	}
}

func cliqueProblem(colorCount, variableCount int) nogoods.Problem {
	problem := nogoods.Problem{Version: nogoods.ProblemVersion, Assignment: []nogoods.Literal{}}
	for color := 0; color < colorCount; color++ {
		problem.ColorAliases = append(problem.ColorAliases, fmt.Sprintf("c%d", color))
	}
	for variable := 0; variable < variableCount; variable++ {
		domain := make([]int, colorCount)
		for color := range domain {
			domain[color] = color
		}
		problem.Variables = append(problem.Variables, nogoods.Variable{Alias: fmt.Sprintf("v%d", variable), Domain: domain})
	}
	for left := 0; left < variableCount; left++ {
		for right := left + 1; right < variableCount; right++ {
			problem.Edges = append(problem.Edges, nogoods.Edge{Left: left, Right: right})
		}
	}
	return problem
}
