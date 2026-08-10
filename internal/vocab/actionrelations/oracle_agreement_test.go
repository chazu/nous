package actionrelations_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/chazu/nous/internal/actionrelationoracle"
	"github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestProductionAgreesWithIndependentOracle(t *testing.T) {
	actions := boundedActions()
	states := boundedStates(3, []int{0, 8})
	for _, state := range states {
		stateJSON, _ := state.CanonicalJSON()
		for _, action := range actions {
			actionJSON, _ := action.CanonicalJSON()
			oracle, err := actionrelationoracle.Apply(stateJSON, actionJSON)
			if err != nil {
				t.Fatalf("oracle: %v", err)
			}
			production, outcome, err := actionrelations.Apply(state, action)
			if err != nil {
				t.Fatalf("production: %v", err)
			}
			productionJSON, _ := production.CanonicalJSON()
			if oracle.Applicable != (outcome == "applied") || !bytes.Equal(oracle.State, productionJSON) {
				t.Fatalf("disagreement state=%s action=%s oracle=%#v production=%s/%s", stateJSON, actionJSON, oracle, outcome, productionJSON)
			}
		}
	}
}

func TestOracleClassificationTableAgainstExplicitProductionSteps(t *testing.T) {
	actions := boundedActions()
	for _, state := range boundedStates(2, []int{0, 7, 8}) {
		stateJSON, _ := state.CanonicalJSON()
		for _, a := range actions {
			aJSON, _ := a.CanonicalJSON()
			for _, b := range actions {
				bJSON, _ := b.CanonicalJSON()
				observation, err := actionrelationoracle.Observe(stateJSON, aJSON, bJSON)
				if err != nil {
					t.Fatal(err)
				}
				label, ab, ba := explicitClassification(t, state, a, b)
				if observation.Label != label || !bytes.Equal(observation.AB, ab) || !bytes.Equal(observation.BA, ba) {
					t.Fatalf("classification disagreement state=%s a=%s b=%s oracle=%#v production=%s", stateJSON, aJSON, bJSON, observation, label)
				}
			}
		}
	}
}

func explicitClassification(t *testing.T, state actionrelations.State, a, b actionrelations.SemanticAction) (string, []byte, []byte) {
	t.Helper()
	sa, ao, err := actionrelations.Apply(state, a)
	if err != nil {
		t.Fatal(err)
	}
	sb, bo, err := actionrelations.Apply(state, b)
	if err != nil {
		t.Fatal(err)
	}
	aOK, bOK := ao == "applied", bo == "applied"
	if !aOK && !bOK {
		return "inapplicable", nil, nil
	}
	if !aOK {
		_, outcome, _ := actionrelations.Apply(sb, a)
		if outcome == "applied" {
			return "b-enables-a", nil, nil
		}
		return "inapplicable", nil, nil
	}
	if !bOK {
		_, outcome, _ := actionrelations.Apply(sa, b)
		if outcome == "applied" {
			return "a-enables-b", nil, nil
		}
		return "inapplicable", nil, nil
	}
	sab, abo, _ := actionrelations.Apply(sa, b)
	sba, bao, _ := actionrelations.Apply(sb, a)
	if abo != "applied" && bao != "applied" {
		return "mutual-disables", nil, nil
	}
	if abo != "applied" {
		return "a-disables-b", nil, nil
	}
	if bao != "applied" {
		return "b-disables-a", nil, nil
	}
	ab, _ := sab.CanonicalJSON()
	ba, _ := sba.CanonicalJSON()
	if bytes.Equal(ab, ba) {
		return "commutes", ab, ba
	}
	return "conflicts", ab, ba
}

func boundedStates(maxCells int, traceLengths []int) []actionrelations.State {
	var states []actionrelations.State
	for cells := 1; cells <= maxCells; cells++ {
		combinations := 1
		for range cells {
			combinations *= 4
		}
		for encoded := 0; encoded < combinations; encoded++ {
			values := encoded
			row := make([]actionrelations.Cell, cells)
			for index := range cells {
				row[index] = actionrelations.Cell{Name: fmt.Sprintf("c%d", index), Value: values % 4}
				values /= 4
			}
			for _, length := range traceLengths {
				events := make([]string, length)
				for index := range events {
					events[index] = "e"
				}
				states = append(states, actionrelations.State{Cells: row, Events: events})
			}
		}
	}
	return states
}

func boundedActions() []actionrelations.SemanticAction {
	roles := []string{"c0", "c1", "c2"}
	var actions []actionrelations.SemanticAction
	for _, x := range roles {
		for _, n := range []int{-2, -1, 1, 2} {
			actions = append(actions, actionrelations.SemanticAction{Kind: "add", XRole: x, N: n})
		}
		for _, n := range []int{0, 1, 2, 3} {
			actions = append(actions, actionrelations.SemanticAction{Kind: "set", XRole: x, N: n})
			actions = append(actions, actionrelations.SemanticAction{Kind: "check", XRole: x, N: n})
		}
		actions = append(actions,
			actionrelations.SemanticAction{Kind: "claim", XRole: x},
			actionrelations.SemanticAction{Kind: "release", XRole: x},
		)
		for _, y := range roles {
			if x == y {
				continue
			}
			for _, n := range []int{1, 2} {
				actions = append(actions, actionrelations.SemanticAction{Kind: "transfer", XRole: x, YRole: y, N: n})
			}
			actions = append(actions, actionrelations.SemanticAction{Kind: "swap", XRole: x, YRole: y})
		}
	}
	actions = append(actions,
		actionrelations.SemanticAction{Kind: "emit", Symbol: "a"},
		actionrelations.SemanticAction{Kind: "emit", Symbol: "b"},
	)
	return actions
}
