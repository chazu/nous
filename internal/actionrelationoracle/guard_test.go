package actionrelationoracle_test

import (
	"testing"

	"github.com/chazu/nous/internal/actionrelationoracle"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestIndependentOracleAgreesOnAllNormalizedGuards(t *testing.T) {
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 2}, {Name: "c2", Value: 3}}, Events: []string{"a"}}
	actions := []actionrelations.SemanticAction{
		{Kind: "add", XRole: "c0", N: 1}, {Kind: "add", XRole: "c0", N: -1},
		{Kind: "transfer", XRole: "c1", YRole: "c2", N: 1}, {Kind: "swap", XRole: "c0", YRole: "c2"},
		{Kind: "emit", Symbol: "a"}, {Kind: "emit", Symbol: "b"},
	}
	stateJSON, _ := state.CanonicalJSON()
	guards := actionrelations.EnumerateGuards()
	if len(guards) != 451 {
		t.Fatalf("guard count=%d", len(guards))
	}
	for left := range actions {
		for right := range actions {
			if left == right {
				continue
			}
			occurrences, _ := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{actions[left], actions[right]})
			aFacts, err := actionrelations.Facts(state, occurrences[0])
			if err != nil {
				t.Fatal(err)
			}
			bFacts, _ := actionrelations.Facts(state, occurrences[1])
			aJSON, _ := occurrences[0].Action.CanonicalJSON()
			bJSON, _ := occurrences[1].Action.CanonicalJSON()
			for ordinal, guard := range guards {
				production, err := guard.Evaluate(aFacts, bFacts)
				guardJSON, _ := guard.CanonicalJSON()
				oracle, oracleErr := actionrelationoracle.EvaluateGuard(stateJSON, aJSON, bJSON, guardJSON)
				if err != nil || oracleErr != nil || production != oracle {
					t.Fatalf("pair=%d/%d guard=%d production=%v/%v oracle=%v/%v", left, right, ordinal, production, err, oracle, oracleErr)
				}
			}
		}
	}
}
