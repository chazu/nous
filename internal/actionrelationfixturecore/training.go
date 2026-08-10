// Package actionrelationfixturecore constructs deterministic, public training
// curricula. It does not run CUE, a search policy, or a learned artifact.
package actionrelationfixturecore

import (
	"fmt"

	"github.com/chazu/nous/internal/actionrelationoracle"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

const TrainingCount = 16

type Case struct {
	Ordinal     int
	State       []byte
	AOccurrence []byte
	BOccurrence []byte
	Label       string
}

// Training returns eight disjoint-add commutations followed by one witness for
// each negative diagnostic, with the second conflict reserved for event order.
func Training() ([]Case, error) {
	positives, err := positiveCases()
	if err != nil {
		return nil, err
	}
	negatives, err := negativeCases()
	if err != nil {
		return nil, err
	}
	cases := append(positives, negatives...)
	for index := range cases {
		cases[index].Ordinal = index
	}
	if len(cases) != TrainingCount {
		return nil, fmt.Errorf("training count %d", len(cases))
	}
	return cases, nil
}

func positiveCases() ([]Case, error) {
	var cases []Case
	for left := 0; left <= 3 && len(cases) < 8; left++ {
		for right := 0; right <= 3 && len(cases) < 8; right++ {
			for _, aN := range []int{-2, -1, 1, 2} {
				for _, bN := range []int{-2, -1, 1, 2} {
					state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: left}, {Name: "c1", Value: right}}}
					a := actionrelations.SemanticAction{Kind: "add", XRole: "c0", N: aN}
					b := actionrelations.SemanticAction{Kind: "add", XRole: "c1", N: bN}
					candidate, label, err := makeCase(state, a, b)
					if err != nil {
						return nil, err
					}
					if label == "commutes" {
						cases = append(cases, candidate)
						if len(cases) == 8 {
							return cases, nil
						}
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("only %d positive witnesses", len(cases))
}

func negativeCases() ([]Case, error) {
	wanted := []string{"a-enables-b", "b-enables-a", "a-disables-b", "b-disables-a", "mutual-disables", "inapplicable", "conflicts"}
	found := map[string]Case{}
	actions := actionAlphabet()
	for cells := 1; cells <= 2 && len(found) < len(wanted); cells++ {
		combinations := 1
		for range cells {
			combinations *= 4
		}
		for values := 0; values < combinations && len(found) < len(wanted); values++ {
			state := stateFor(cells, values, 0)
			for left := 0; left < len(actions) && len(found) < len(wanted); left++ {
				for right := left; right < len(actions) && len(found) < len(wanted); right++ {
					candidate, label, err := makeCase(state, actions[left], actions[right])
					if err != nil {
						continue
					}
					if contains(wanted, label) {
						if _, exists := found[label]; !exists {
							found[label] = candidate
						}
					}
				}
			}
		}
	}
	result := make([]Case, 0, 8)
	for _, label := range wanted {
		candidate, ok := found[label]
		if !ok {
			return nil, fmt.Errorf("no witness for %s", label)
		}
		result = append(result, candidate)
	}
	eventConflict, label, err := makeCase(
		stateFor(1, 0, 0),
		actionrelations.SemanticAction{Kind: "emit", Symbol: "a"},
		actionrelations.SemanticAction{Kind: "emit", Symbol: "b"},
	)
	if err != nil || label != "conflicts" {
		return nil, fmt.Errorf("event conflict: label=%s err=%v", label, err)
	}
	return append(result, eventConflict), nil
}

func makeCase(state actionrelations.State, left, right actionrelations.SemanticAction) (Case, string, error) {
	occurrences, err := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{left, right})
	if err != nil {
		return Case{}, "", err
	}
	a, b, err := actionrelations.CanonicalPair(occurrences[0], occurrences[1])
	if err != nil || a == b {
		return Case{}, "", actionrelations.ErrInvalid
	}
	stateJSON, _ := state.CanonicalJSON()
	aJSON, _ := a.CanonicalJSON()
	bJSON, _ := b.CanonicalJSON()
	observation, err := actionrelationoracle.Observe(stateJSON, mustActionJSON(a.Action), mustActionJSON(b.Action))
	if err != nil {
		return Case{}, "", err
	}
	return Case{State: stateJSON, AOccurrence: aJSON, BOccurrence: bJSON, Label: observation.Label}, observation.Label, nil
}

func mustActionJSON(action actionrelations.SemanticAction) []byte {
	data, _ := action.CanonicalJSON()
	return data
}

func actionAlphabet() []actionrelations.SemanticAction {
	roles := []string{"c0", "c1"}
	var actions []actionrelations.SemanticAction
	for _, x := range roles {
		for _, n := range []int{-2, -1, 1, 2} {
			actions = append(actions, actionrelations.SemanticAction{Kind: "add", XRole: x, N: n})
		}
		for _, n := range []int{0, 1, 2, 3} {
			actions = append(actions, actionrelations.SemanticAction{Kind: "set", XRole: x, N: n}, actionrelations.SemanticAction{Kind: "check", XRole: x, N: n})
		}
		actions = append(actions, actionrelations.SemanticAction{Kind: "claim", XRole: x}, actionrelations.SemanticAction{Kind: "release", XRole: x})
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
	actions = append(actions, actionrelations.SemanticAction{Kind: "emit", Symbol: "a"}, actionrelations.SemanticAction{Kind: "emit", Symbol: "b"})
	return actions
}

func stateFor(cells, encoded, events int) actionrelations.State {
	row := make([]actionrelations.Cell, cells)
	for index := range row {
		row[index] = actionrelations.Cell{Name: fmt.Sprintf("c%d", index), Value: encoded % 4}
		encoded /= 4
	}
	trace := make([]string, events)
	for index := range trace {
		trace[index] = "e"
	}
	return actionrelations.State{Cells: row, Events: trace}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
