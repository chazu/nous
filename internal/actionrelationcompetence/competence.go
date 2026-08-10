// Package actionrelationcompetence owns the safe, pre-review semantic
// competence universe. It has no fixture seed, policy, learned artifact, or
// protected-panel authority.
package actionrelationcompetence

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/actionrelationoracle"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

const MaximumSequences = 40320

type Report struct {
	Sequences int  `json:"sequences"`
	Steps     int  `json:"steps"`
	Passed    bool `json:"passed"`
}

// Run checks every permutation of one frozen eight-occurrence competence
// history against the independent oracle. The caller, not production
// semantics, owns enumeration.
func Run() (Report, error) {
	report, _, _, err := runSequences(false)
	return report, err
}

// RunSequenceEvidence retains every case/result preimage for the frozen
// 40,320-permutation competence suite.
func RunSequenceEvidence() (Report, Evidence, error) {
	report, cases, results, err := runSequences(true)
	if err != nil {
		return report, Evidence{}, err
	}
	evidence, err := BuildEvidence(cases, results)
	return report, evidence, err
}

// RunEvidence retains the complete bounded competence evidence implemented by
// this package: all 40,320 frozen histories and all 451 normalized guards over
// sixteen hand-selected fact topologies.
func RunEvidence() (Report, Evidence, error) {
	report, cases, results, err := runSequences(true)
	if err != nil {
		return report, Evidence{}, err
	}
	guardCases, guardResults, err := runGuardTruth()
	if err != nil {
		return report, Evidence{}, err
	}
	cases = append(cases, guardCases...)
	results = append(results, guardResults...)
	diamondCases, diamondResults, err := runLocalDiamonds()
	if err != nil {
		return report, Evidence{}, err
	}
	cases = append(cases, diamondCases...)
	results = append(results, diamondResults...)
	semanticCases, semanticResults, err := runSemanticTransitions()
	if err != nil {
		return report, Evidence{}, err
	}
	cases = append(cases, semanticCases...)
	results = append(results, semanticResults...)
	evidence, err := BuildEvidence(cases, results)
	return report, evidence, err
}

func runSequences(retain bool) (Report, []CaseRow, []ResultRow, error) {
	initial := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 1}, {Name: "c2", Value: 2}}}
	actions := []actionrelations.SemanticAction{
		{Kind: "set", XRole: "c0", N: 0},
		{Kind: "set", XRole: "c0", N: 3},
		{Kind: "set", XRole: "c1", N: 1},
		{Kind: "set", XRole: "c1", N: 2},
		{Kind: "set", XRole: "c2", N: 0},
		{Kind: "set", XRole: "c2", N: 3},
		{Kind: "emit", Symbol: "a"},
		{Kind: "emit", Symbol: "b"},
	}
	indices := []int{0, 1, 2, 3, 4, 5, 6, 7}
	report := Report{}
	var cases []CaseRow
	var results []ResultRow
	var visit func(int) error
	visit = func(index int) error {
		if index < len(indices) {
			for candidate := index; candidate < len(indices); candidate++ {
				indices[index], indices[candidate] = indices[candidate], indices[index]
				if err := visit(index + 1); err != nil {
					return err
				}
				indices[index], indices[candidate] = indices[candidate], indices[index]
			}
			return nil
		}
		production := initial
		oracleJSON, _ := initial.CanonicalJSON()
		orderedActions := make([]json.RawMessage, len(indices))
		for ordinal, actionIndex := range indices {
			action := actions[actionIndex]
			next, outcome, err := actionrelations.Apply(production, action)
			if err != nil || outcome != "applied" {
				return fmt.Errorf("production step %d: %s %w", report.Sequences, outcome, err)
			}
			actionJSON, _ := action.CanonicalJSON()
			orderedActions[ordinal] = actionJSON
			oracle, err := actionrelationoracle.Apply(oracleJSON, actionJSON)
			if err != nil || !oracle.Applicable {
				return fmt.Errorf("oracle step %d: %w", report.Sequences, err)
			}
			productionJSON, _ := next.CanonicalJSON()
			if !bytes.Equal(productionJSON, oracle.State) {
				return fmt.Errorf("history disagreement sequence=%d action=%d", report.Sequences, actionIndex)
			}
			production, oracleJSON = next, oracle.State
			report.Steps++
		}
		if retain {
			initialJSON, _ := initial.CanonicalJSON()
			input, _ := json.Marshal([]any{"actionrelation-competence-sequence-input/v1", json.RawMessage(initialJSON), orderedActions})
			expected := shaHex(oracleJSON)
			caseID := fmt.Sprintf("%05d", report.Sequences)
			cases = append(cases, CaseRow{Suite: "complete-sequences", CaseID: caseID, Input: shaHex(input), Expected: expected})
			results = append(results, ResultRow{Suite: "complete-sequences", CaseID: caseID, Production: expected, Oracle: expected})
		}
		report.Sequences++
		return nil
	}
	if err := visit(0); err != nil {
		return report, nil, nil, err
	}
	report.Passed = report.Sequences == MaximumSequences && report.Steps == MaximumSequences*len(actions)
	if !report.Passed {
		return report, nil, nil, fmt.Errorf("competence cardinality mismatch: %+v", report)
	}
	return report, cases, results, nil
}

func runGuardTruth() ([]CaseRow, []ResultRow, error) {
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 2}, {Name: "c2", Value: 3}}, Events: []string{"a"}}
	a := func(kind, x, y string, n int, symbol string) actionrelations.SemanticAction {
		return actionrelations.SemanticAction{Kind: kind, XRole: x, YRole: y, N: n, Symbol: symbol}
	}
	pairs := [][2]actionrelations.SemanticAction{
		{a("add", "c0", "", 1, ""), a("add", "c0", "", -1, "")},
		{a("add", "c0", "", 1, ""), a("add", "c1", "", 1, "")},
		{a("set", "c0", "", 0, ""), a("set", "c0", "", 3, "")},
		{a("set", "c0", "", 0, ""), a("check", "c0", "", 0, "")},
		{a("transfer", "c0", "c1", 1, ""), a("transfer", "c1", "c0", 1, "")},
		{a("transfer", "c0", "c1", 1, ""), a("swap", "c1", "c2", 0, "")},
		{a("claim", "c0", "", 0, ""), a("release", "c0", "", 0, "")},
		{a("claim", "c0", "", 0, ""), a("claim", "c1", "", 0, "")},
		{a("release", "c0", "", 0, ""), a("release", "c1", "", 0, "")},
		{a("check", "c0", "", 0, ""), a("check", "c1", "", 2, "")},
		{a("emit", "", "", 0, "a"), a("emit", "", "", 0, "a")},
		{a("emit", "", "", 0, "a"), a("emit", "", "", 0, "b")},
		{a("emit", "", "", 0, "a"), a("add", "c0", "", 1, "")},
		{a("swap", "c0", "c1", 0, ""), a("swap", "c1", "c2", 0, "")},
		{a("transfer", "c0", "c2", 2, ""), a("add", "c2", "", -1, "")},
		{a("set", "c2", "", 3, ""), a("emit", "", "", 0, "b")},
	}
	stateJSON, _ := state.CanonicalJSON()
	guards := actionrelations.EnumerateGuards()
	cases := make([]CaseRow, 0, len(pairs)*len(guards))
	results := make([]ResultRow, 0, len(pairs)*len(guards))
	for pairOrdinal, pair := range pairs {
		occurrences, err := actionrelations.AssignOccurrences(pair[:])
		if err != nil || len(occurrences) != 2 {
			return nil, nil, fmt.Errorf("guard pair %d: %w", pairOrdinal, err)
		}
		aFacts, err := actionrelations.Facts(state, occurrences[0])
		if err != nil {
			return nil, nil, err
		}
		bFacts, err := actionrelations.Facts(state, occurrences[1])
		if err != nil {
			return nil, nil, err
		}
		aJSON, _ := occurrences[0].Action.CanonicalJSON()
		bJSON, _ := occurrences[1].Action.CanonicalJSON()
		for guardOrdinal, guard := range guards {
			production, err := guard.Evaluate(aFacts, bFacts)
			guardJSON, _ := guard.CanonicalJSON()
			oracle, oracleErr := actionrelationoracle.EvaluateGuard(stateJSON, aJSON, bJSON, guardJSON)
			if err != nil || oracleErr != nil || production != oracle {
				return nil, nil, fmt.Errorf("guard disagreement pair=%d guard=%d", pairOrdinal, guardOrdinal)
			}
			input, _ := json.Marshal([]any{"actionrelation-competence-guard-input/v1", json.RawMessage(stateJSON), json.RawMessage(aJSON), json.RawMessage(bJSON), json.RawMessage(guardJSON)})
			output, _ := json.Marshal([]any{"actionrelation-competence-guard-result/v1", production})
			caseID := fmt.Sprintf("%02d-%03d", pairOrdinal, guardOrdinal)
			expected := shaHex(output)
			cases = append(cases, CaseRow{Suite: "guard-truth", CaseID: caseID, Input: shaHex(input), Expected: expected})
			results = append(results, ResultRow{Suite: "guard-truth", CaseID: caseID, Production: expected, Oracle: expected})
		}
	}
	return cases, results, nil
}

func runSemanticTransitions() ([]CaseRow, []ResultRow, error) {
	var cases []CaseRow
	var results []ResultRow
	for cellCount := 1; cellCount <= 3; cellCount++ {
		stateCount := 1
		for range cellCount {
			stateCount *= 4
		}
		actions := competenceActions(cellCount)
		for stateOrdinal := 0; stateOrdinal < stateCount; stateOrdinal++ {
			state := competenceState(cellCount, stateOrdinal)
			stateJSON, _ := state.CanonicalJSON()
			for actionOrdinal, action := range actions {
				actionJSON, _ := action.CanonicalJSON()
				production, outcome, err := actionrelations.Apply(state, action)
				oracle, oracleErr := actionrelationoracle.Apply(stateJSON, actionJSON)
				productionJSON, productionErr := production.CanonicalJSON()
				if err != nil || oracleErr != nil || productionErr != nil || (outcome == "applied") != oracle.Applicable || !bytes.Equal(productionJSON, oracle.State) {
					return nil, nil, fmt.Errorf("semantic disagreement cells=%d state=%d action=%d", cellCount, stateOrdinal, actionOrdinal)
				}
				input, _ := json.Marshal([]any{"actionrelation-competence-transition-input/v1", json.RawMessage(stateJSON), json.RawMessage(actionJSON)})
				output, _ := json.Marshal([]any{"actionrelation-competence-transition-result/v1", oracle.Applicable, shaHex(oracle.State)})
				caseID := fmt.Sprintf("%d-%03d-%03d", cellCount, stateOrdinal, actionOrdinal)
				expected := shaHex(output)
				cases = append(cases, CaseRow{Suite: "semantic-transitions", CaseID: caseID, Input: shaHex(input), Expected: expected})
				results = append(results, ResultRow{Suite: "semantic-transitions", CaseID: caseID, Production: expected, Oracle: expected})
			}
		}
	}
	return cases, results, nil
}

func runLocalDiamonds() ([]CaseRow, []ResultRow, error) {
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 1}, {Name: "c1", Value: 2}, {Name: "c2", Value: 0}}, Events: []string{}}
	stateJSON, _ := state.CanonicalJSON()
	actions := competenceActions(3)
	cases := make([]CaseRow, 0, len(actions)*len(actions))
	results := make([]ResultRow, 0, len(actions)*len(actions))
	for left, a := range actions {
		aJSON, _ := a.CanonicalJSON()
		for right, b := range actions {
			bJSON, _ := b.CanonicalJSON()
			label, ab, ba, err := productionObserve(state, a, b)
			oracle, oracleErr := actionrelationoracle.Observe(stateJSON, aJSON, bJSON)
			if err != nil || oracleErr != nil || label != oracle.Label || !bytes.Equal(ab, oracle.AB) || !bytes.Equal(ba, oracle.BA) {
				return nil, nil, fmt.Errorf("diamond disagreement left=%d right=%d", left, right)
			}
			input, _ := json.Marshal([]any{"actionrelation-competence-diamond-input/v1", json.RawMessage(stateJSON), json.RawMessage(aJSON), json.RawMessage(bJSON)})
			output, _ := json.Marshal([]any{"actionrelation-competence-diamond-result/v1", label, digestOrZero(ab), digestOrZero(ba)})
			caseID := fmt.Sprintf("%03d-%03d", left, right)
			expected := shaHex(output)
			cases = append(cases, CaseRow{Suite: "local-diamonds", CaseID: caseID, Input: shaHex(input), Expected: expected})
			results = append(results, ResultRow{Suite: "local-diamonds", CaseID: caseID, Production: expected, Oracle: expected})
		}
	}
	return cases, results, nil
}

func productionObserve(state actionrelations.State, a, b actionrelations.SemanticAction) (string, []byte, []byte, error) {
	sa, aOutcome, err := actionrelations.Apply(state, a)
	if err != nil {
		return "", nil, nil, err
	}
	sb, bOutcome, err := actionrelations.Apply(state, b)
	if err != nil {
		return "", nil, nil, err
	}
	aInitial, bInitial := aOutcome == "applied", bOutcome == "applied"
	if !aInitial && !bInitial {
		return "inapplicable", nil, nil, nil
	}
	if !aInitial {
		_, outcome, applyErr := actionrelations.Apply(sb, a)
		if applyErr != nil {
			return "", nil, nil, applyErr
		}
		if outcome == "applied" {
			return "b-enables-a", nil, nil, nil
		}
		return "inapplicable", nil, nil, nil
	}
	if !bInitial {
		_, outcome, applyErr := actionrelations.Apply(sa, b)
		if applyErr != nil {
			return "", nil, nil, applyErr
		}
		if outcome == "applied" {
			return "a-enables-b", nil, nil, nil
		}
		return "inapplicable", nil, nil, nil
	}
	sab, bAfter, err := actionrelations.Apply(sa, b)
	if err != nil {
		return "", nil, nil, err
	}
	sba, aAfter, err := actionrelations.Apply(sb, a)
	if err != nil {
		return "", nil, nil, err
	}
	if bAfter != "applied" && aAfter != "applied" {
		return "mutual-disables", nil, nil, nil
	}
	if bAfter != "applied" {
		return "a-disables-b", nil, nil, nil
	}
	if aAfter != "applied" {
		return "b-disables-a", nil, nil, nil
	}
	ab, _ := sab.CanonicalJSON()
	ba, _ := sba.CanonicalJSON()
	if bytes.Equal(ab, ba) {
		return "commutes", ab, ba, nil
	}
	return "conflicts", ab, ba, nil
}

func digestOrZero(value []byte) string {
	if len(value) == 0 {
		return "0000000000000000000000000000000000000000000000000000000000000000"
	}
	return shaHex(value)
}

func competenceState(cellCount, ordinal int) actionrelations.State {
	state := actionrelations.State{Cells: make([]actionrelations.Cell, cellCount), Events: []string{}}
	for index := 0; index < cellCount; index++ {
		state.Cells[index] = actionrelations.Cell{Name: fmt.Sprintf("c%d", index), Value: ordinal % 4}
		ordinal /= 4
	}
	return state
}

func competenceActions(cellCount int) []actionrelations.SemanticAction {
	var result []actionrelations.SemanticAction
	for x := 0; x < cellCount; x++ {
		role := fmt.Sprintf("c%d", x)
		for _, n := range []int{-2, -1, 1, 2} {
			result = append(result, actionrelations.SemanticAction{Kind: "add", XRole: role, N: n})
		}
		for _, n := range []int{0, 1, 2, 3} {
			result = append(result, actionrelations.SemanticAction{Kind: "set", XRole: role, N: n})
		}
		result = append(result, actionrelations.SemanticAction{Kind: "claim", XRole: role}, actionrelations.SemanticAction{Kind: "release", XRole: role})
		for _, n := range []int{0, 1, 2, 3} {
			result = append(result, actionrelations.SemanticAction{Kind: "check", XRole: role, N: n})
		}
	}
	for x := 0; x < cellCount; x++ {
		for y := 0; y < cellCount; y++ {
			if x == y {
				continue
			}
			for _, n := range []int{1, 2} {
				result = append(result, actionrelations.SemanticAction{Kind: "transfer", XRole: fmt.Sprintf("c%d", x), YRole: fmt.Sprintf("c%d", y), N: n})
			}
			result = append(result, actionrelations.SemanticAction{Kind: "swap", XRole: fmt.Sprintf("c%d", x), YRole: fmt.Sprintf("c%d", y)})
		}
	}
	return append(result, actionrelations.SemanticAction{Kind: "emit", Symbol: "a"})
}
