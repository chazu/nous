package actionrelationoracle_test

import (
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationoracle"
	"github.com/chazu/nous/internal/actionrelationsearch"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestIndependentCompleteEnumeratorMatchesProductionTerminalWires(t *testing.T) {
	world := actionrelations.World{
		State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "a", Value: 0}, {Name: "b", Value: 1}}, Events: []string{}},
		Actions: []actionrelations.Action{
			{Name: "seta", Kind: "set", X: "a", N: 2},
			{Name: "claima", Kind: "claim", X: "a"},
			{Name: "releaseb", Kind: "release", X: "b"},
			{Name: "event", Kind: "emit", Symbol: "x"},
		},
	}
	normalized, err := world.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	stateJSON, _ := normalized.State.CanonicalJSON()
	actions := make([][]byte, len(normalized.Actions))
	for index, action := range normalized.Actions {
		actions[index], _ = action.CanonicalJSON()
	}
	oracle, err := actionrelationoracle.CompleteTerminalDigests(stateJSON, actions)
	production, productionErr := actionrelationsearch.Search(world, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
	if err != nil || productionErr != nil || !slices.Equal(oracle, production.TerminalDigests) {
		t.Fatalf("oracle=%v production=%v errors=%v/%v", oracle, production.TerminalDigests, err, productionErr)
	}
}
