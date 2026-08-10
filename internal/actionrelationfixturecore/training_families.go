package actionrelationfixturecore

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationoracle"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TrainingFamily(family int) ([]Case, error) {
	if family < 0 || family >= len(FamilyNames) {
		return nil, fmt.Errorf("invalid training family")
	}
	if family == 0 {
		return Training()
	}
	positives, err := familyPositiveCases(family)
	if err != nil {
		return nil, err
	}
	negatives, err := negativeCases()
	if err != nil {
		return nil, err
	}
	switch family {
	case 1, 7:
		hard, label, err := makeCase(
			actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 2}, {Name: "c1", Value: 0}, {Name: "c2", Value: 0}}, Events: []string{}},
			action("add", "c0", "", 1, ""), action("add", "c0", "", 1, ""),
		)
		if err != nil || label != "mutual-disables" {
			return nil, fmt.Errorf("bounded-add hard negative: %s %v", label, err)
		}
		negatives[4] = hard
	case 2:
		hard, label, err := makeCase(
			actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}, {Name: "c2", Value: 0}}, Events: []string{}},
			action("set", "c0", "", 1, ""), action("set", "c0", "", 2, ""),
		)
		if err != nil || label != "conflicts" {
			return nil, fmt.Errorf("equal-set hard negative: %s %v", label, err)
		}
		negatives[6] = hard
	case 6:
		hard, label, err := makeCase(
			actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}, {Name: "c2", Value: 0}}, Events: []string{}},
			action("emit", "", "", 0, "e0"), action("emit", "", "", 0, "e1"),
		)
		if err != nil || label != "conflicts" {
			return nil, fmt.Errorf("identical-emit hard negative: %s %v", label, err)
		}
		negatives[7] = hard
	}
	result := append(positives, negatives...)
	for index := range result {
		result[index].Ordinal = index
	}
	return result, nil
}

func familyPositiveCases(family int) ([]Case, error) {
	pairs := familyPairs(family)
	byCore := map[string]Case{}
	for values := 0; values < 64; values++ {
		for traceLength := 0; traceLength <= 6; traceLength++ {
			events := make([]string, traceLength)
			for index := range events {
				events[index] = "e3"
			}
			state := actionrelations.State{Cells: []actionrelations.Cell{
				{Name: "c0", Value: values / 16}, {Name: "c1", Value: values / 4 % 4}, {Name: "c2", Value: values % 4},
			}, Events: events}
			for _, pair := range pairs {
				testCase, label, err := makeCase(state, pair[0], pair[1])
				if err != nil || label != "commutes" {
					continue
				}
				normalizedState, _ := actionrelations.ParseState(testCase.State)
				left, _ := actionrelations.ParseOccurrence(testCase.AOccurrence)
				right, _ := actionrelations.ParseOccurrence(testCase.BOccurrence)
				matched, err := latentMatch(family, normalizedState, left, right)
				if err != nil || !matched {
					continue
				}
				key := string(testCase.State) + string(testCase.AOccurrence) + string(testCase.BOccurrence)
				byCore[key] = testCase
			}
		}
	}
	keys := make([]string, 0, len(byCore))
	for key := range byCore {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int { return bytes.Compare([]byte(a), []byte(b)) })
	if len(keys) < 8 {
		return nil, fmt.Errorf("family %d has only %d positive cores", family, len(keys))
	}
	if family == 1 || family == 7 {
		var zero, nonzero []string
		for _, key := range keys {
			testCase := byCore[key]
			state, _ := actionrelations.ParseState(testCase.State)
			left, _ := actionrelations.ParseOccurrence(testCase.AOccurrence)
			value, _ := state.Value(left.Action.XRole)
			if value == 0 {
				zero = append(zero, key)
			} else {
				nonzero = append(nonzero, key)
			}
		}
		if len(zero) < 4 || len(nonzero) < 4 {
			return nil, fmt.Errorf("family %d lacks value-balanced positives", family)
		}
		keys = append(slices.Clone(zero[:4]), nonzero[:4]...)
	}
	result := make([]Case, 8)
	for index := range result {
		result[index] = byCore[keys[index]]
	}
	return result, nil
}

func familyPairs(family int) [][2]actionrelations.SemanticAction {
	var result [][2]actionrelations.SemanticAction
	roles := []string{"c0", "c1", "c2"}
	addArguments := []int{-2, -1, 1, 2}
	switch family {
	case 0:
		for _, leftRole := range roles {
			for _, rightRole := range roles {
				if leftRole == rightRole {
					continue
				}
				for _, leftN := range addArguments {
					for _, rightN := range addArguments {
						result = append(result, [2]actionrelations.SemanticAction{action("add", leftRole, "", leftN, ""), action("add", rightRole, "", rightN, "")})
					}
				}
			}
		}
	case 1, 7:
		for _, role := range roles {
			for _, leftN := range addArguments {
				for _, rightN := range addArguments {
					result = append(result, [2]actionrelations.SemanticAction{action("add", role, "", leftN, ""), action("add", role, "", rightN, "")})
				}
			}
		}
	case 2:
		for _, role := range roles {
			for leftN := 0; leftN <= 3; leftN++ {
				for rightN := 0; rightN <= 3; rightN++ {
					result = append(result, [2]actionrelations.SemanticAction{action("set", role, "", leftN, ""), action("set", role, "", rightN, "")})
				}
			}
		}
	case 3:
		for _, symbol := range []string{"e0", "e1", "e2", "e3"} {
			for _, role := range roles {
				for _, n := range addArguments {
					result = append(result, [2]actionrelations.SemanticAction{action("emit", "", "", 0, symbol), action("add", role, "", n, "")})
				}
			}
		}
	case 4:
		for _, x := range roles {
			for _, y := range roles {
				if x != y {
					swap := action("swap", x, y, 0, "")
					result = append(result, [2]actionrelations.SemanticAction{swap, swap})
				}
			}
		}
	case 5:
		for _, symbol := range []string{"e0", "e1", "e2", "e3"} {
			for _, x := range roles {
				for _, y := range roles {
					if x == y {
						continue
					}
					for _, n := range []int{1, 2} {
						result = append(result, [2]actionrelations.SemanticAction{action("emit", "", "", 0, symbol), action("transfer", x, y, n, "")})
					}
				}
			}
		}
	case 6:
		for _, left := range []string{"e0", "e1", "e2", "e3"} {
			for _, right := range []string{"e0", "e1", "e2", "e3"} {
				result = append(result, [2]actionrelations.SemanticAction{action("emit", "", "", 0, left), action("emit", "", "", 0, right)})
			}
		}
	}
	return result
}

func VerifyFamilyGuard(family int, guard actionrelations.Guard) error {
	if family < 0 || family >= len(FamilyNames) {
		return fmt.Errorf("invalid guard family")
	}
	for values := 0; values < 64; values++ {
		for traceLength := 0; traceLength <= 6; traceLength++ {
			events := make([]string, traceLength)
			for index := range events {
				events[index] = "e3"
			}
			state := actionrelations.State{Cells: []actionrelations.Cell{
				{Name: "c0", Value: values / 16}, {Name: "c1", Value: values / 4 % 4}, {Name: "c2", Value: values % 4},
			}, Events: events}
			stateJSON, _ := state.CanonicalJSON()
			for pairOrdinal, pair := range familyPairs(family) {
				occurrences, err := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{pair[0], pair[1]})
				if err != nil {
					return err
				}
				left, right, err := actionrelations.CanonicalPair(occurrences[0], occurrences[1])
				if err != nil {
					return err
				}
				truth, err := latentMatch(family, state, left, right)
				if err != nil {
					return err
				}
				bothApplicable := true
				for _, occurrence := range []actionrelations.Occurrence{left, right} {
					actionJSON, _ := occurrence.Action.CanonicalJSON()
					transition, err := actionrelationoracle.Apply(stateJSON, actionJSON)
					if err != nil {
						return err
					}
					bothApplicable = bothApplicable && transition.Applicable
				}
				leftFacts, _ := actionrelations.Facts(state, left)
				rightFacts, _ := actionrelations.Facts(state, right)
				matched, err := guard.Evaluate(leftFacts, rightFacts)
				if err != nil {
					return err
				}
				predicted := bothApplicable && matched
				if predicted != truth {
					return fmt.Errorf("family %d assignment %d trace %d pair %d predicted=%t truth=%t", family, values, traceLength, pairOrdinal, predicted, truth)
				}
			}
		}
	}
	return nil
}
