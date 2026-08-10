package dsl

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func bARCandidateResult(vm *VM) error {
	requestedValue, viewRootValue, rowsValue, candidateValue := vm.pop(), vm.pop(), vm.pop(), vm.pop()
	if requestedValue.Kind() != VString || viewRootValue.Kind() != VString || rowsValue.Kind() != VList || candidateValue.Kind() != VString || vm.Store == nil || !actionrelationsDigest(viewRootValue.AsString()) {
		vm.push(Nil())
		return nil
	}
	candidate := vm.Store.Get(candidateValue.AsString())
	if candidate == nil || !vm.Store.IsA(candidate.Name, "ActionGuardCandidate") || len(rowsValue.AsList()) != 16 {
		vm.push(Nil())
		return nil
	}
	experiment := vm.Store.Get(candidate.GetString("experiment"))
	if experiment == nil || len(experiment.GetStrings("observationUnits")) != 16 {
		vm.push(Nil())
		return nil
	}
	guard, guardErr := actionrelations.ParseGuard([]byte(candidate.GetString("guard")))
	pattern, patternErr := actionrelations.ParsePattern([]byte(candidate.GetString("pattern")))
	if guardErr != nil || patternErr != nil {
		vm.push(Nil())
		return nil
	}
	guardDigest, _ := guard.Digest()
	patternDigest, _ := pattern.Digest()
	rowDigests := make([]string, 16)
	var positives, negatives []string
	positiveCoverage, falseMatches, negativeCoverage := 0, 0, 0
	for index, value := range rowsValue.AsList() {
		if value.Kind() != VString {
			vm.push(Nil())
			return nil
		}
		row := vm.Store.Get(value.AsString())
		observationName := experiment.GetStrings("observationUnits")[index]
		observation := vm.Store.Get(observationName)
		if row == nil || observation == nil || !vm.Store.IsA(row.Name, "ActionGuardResult") || row.GetString("guardDigest") != guardDigest || row.GetString("observationDigest") != observation.GetString("objectDigest") {
			vm.push(Nil())
			return nil
		}
		rowDigests[index] = row.GetString("objectDigest")
		eligibleContext := observation.GetString("patternDigest") == patternDigest && observation.GetBool("bothInitiallyApplicable") && observation.GetInt("traceLength") <= 6
		if observation.GetString("label") != "commutes" {
			negativeCoverage++
			negatives = append(negatives, observation.GetString("objectDigest"))
		}
		if eligibleContext && row.GetBool("result") {
			if observation.GetString("label") == "commutes" {
				positiveCoverage++
				positives = append(positives, observation.GetString("objectDigest"))
			} else {
				falseMatches++
			}
		}
	}
	slices.Sort(positives)
	slices.Sort(negatives)
	eligible := positiveCoverage > 0 && falseMatches == 0
	wire, _ := json.Marshal([]any{"action-candidate-result/v1", candidate.GetString("objectDigest"), rowDigests, positiveCoverage, negativeCoverage, true, eligible, viewRootValue.AsString()})
	name, err := arStoreCanonical(vm, requestedValue.AsString(), "ActionGuardCandidateResult", wire, map[string]any{
		"candidate": candidate.Name, "candidateDigest": candidate.GetString("objectDigest"), "positiveCoverage": positiveCoverage, "negativeCoverage": negativeCoverage,
		"falseMatches": falseMatches, "wrapperCoverageComplete": true, "eligible": eligible, "guardResults": valuesToStrings(rowsValue.AsList()),
		"positiveObservationDigests": positives, "negativeObservationDigests": negatives, "viewEvidenceRoot": viewRootValue.AsString(), "ordinal": candidate.GetInt("ordinal"), "literalCount": candidate.GetInt("literalCount"), "experiment": experiment.Name,
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	if err := recordActionRelation(vm, 20, 5, "candidate-result", [][]byte{[]byte(candidate.GetString("canonicalObject"))}, [][]byte{wire}); err != nil {
		return err
	}
	vm.push(StringVal(name))
	return nil
}

func bARCloseGuardSearch(vm *VM) error {
	requestedValue, winnerLeavesValue, evaluationRootsValue, edgeRootValue, candidateLeavesValue, winnersValue, resultsValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	if requestedValue.Kind() != VString || winnerLeavesValue.Kind() != VList || evaluationRootsValue.Kind() != VList || edgeRootValue.Kind() != VString || candidateLeavesValue.Kind() != VList || winnersValue.Kind() != VList || resultsValue.Kind() != VList || vm.Store == nil || len(resultsValue.AsList()) == 0 || len(winnersValue.AsList()) == 0 || len(winnerLeavesValue.AsList()) != len(winnersValue.AsList()) || !actionrelationsDigest(edgeRootValue.AsString()) {
		vm.push(Nil())
		return nil
	}
	resultNames := valuesToStrings(resultsValue.AsList())
	firstResult := vm.Store.Get(resultNames[0])
	firstCandidate := (*unit.Unit)(nil)
	if firstResult != nil {
		firstCandidate = vm.Store.Get(firstResult.GetString("candidate"))
	}
	experiment := (*unit.Unit)(nil)
	if firstCandidate != nil {
		experiment = vm.Store.Get(firstCandidate.GetString("experiment"))
	}
	if experiment == nil {
		vm.push(Nil())
		return nil
	}
	wantCandidates, wantEvaluations, wantEdgeRoot := 451, 2, true
	if experiment.GetString("scope") == "no-guard" {
		wantCandidates, wantEvaluations, wantEdgeRoot = 1, 1, false
	}
	if len(resultNames) != wantCandidates || len(candidateLeavesValue.AsList()) != wantCandidates || len(evaluationRootsValue.AsList()) != wantEvaluations || wantEdgeRoot && edgeRootValue.AsString() == actionRelationZeroDigest || !wantEdgeRoot && edgeRootValue.AsString() != actionRelationZeroDigest {
		vm.push(Nil())
		return nil
	}
	winnerNames := valuesToStrings(winnersValue.AsList())
	candidateLeaves := valuesToStrings(candidateLeavesValue.AsList())
	winnerLeaves := valuesToStrings(winnerLeavesValue.AsList())
	evaluationRoots := valuesToStrings(evaluationRootsValue.AsList())
	for _, digest := range append(append(append([]string{}, candidateLeaves...), winnerLeaves...), evaluationRoots...) {
		if !actionrelationsDigest(digest) {
			vm.push(Nil())
			return nil
		}
	}
	maxPositive, minLiterals := -1, 3
	for ordinal, name := range resultNames {
		result := vm.Store.Get(name)
		if result == nil || !vm.Store.IsA(name, "ActionGuardCandidateResult") || result.GetInt("ordinal") != ordinal {
			vm.push(Nil())
			return nil
		}
		candidate := vm.Store.Get(result.GetString("candidate"))
		if candidate == nil || candidate.GetString("tableLeafDigest") != candidateLeaves[ordinal] {
			vm.push(Nil())
			return nil
		}
		if result.GetBool("eligible") {
			positive, literals := result.GetInt("positiveCoverage"), result.GetInt("literalCount")
			if positive > maxPositive || positive == maxPositive && literals < minLiterals {
				maxPositive, minLiterals = positive, literals
			}
		}
	}
	var expectedWinners []string
	for _, name := range resultNames {
		result := vm.Store.Get(name)
		if result.GetBool("eligible") && result.GetInt("positiveCoverage") == maxPositive && result.GetInt("literalCount") == minLiterals {
			expectedWinners = append(expectedWinners, name)
		}
	}
	if !slices.Equal(winnerNames, expectedWinners) {
		vm.push(Nil())
		return nil
	}
	for index, name := range winnerNames {
		result := vm.Store.Get(name)
		if result == nil || result.GetInt("ordinal") < 0 || result.GetInt("ordinal") >= len(candidateLeaves) || result.GetString("tableLeafDigest") != winnerLeaves[index] {
			vm.push(Nil())
			return nil
		}
	}
	wire, _ := json.Marshal([]any{"action-guard-search-barrier/v1", candidateLeaves, edgeRootValue.AsString(), evaluationRoots, winnerLeaves, "completed"})
	name, err := arStoreCanonical(vm, requestedValue.AsString(), "ActionGuardSearchBarrier", wire, map[string]any{
		"candidateResults": resultNames, "winnerResults": winnerNames, "candidateLeafDigests": candidateLeaves, "edgeTableRoot": edgeRootValue.AsString(), "evaluationTableRoots": evaluationRoots, "winnerLeafDigests": winnerLeaves, "maximumPositiveCoverage": maxPositive, "minimumLiteralCount": minLiterals, "status": "completed",
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(name))
	return nil
}

func bARFreezeRelation(vm *VM) error {
	requestedValue, trainingRootValue, barrierValue := vm.pop(), vm.pop(), vm.pop()
	if requestedValue.Kind() != VString || trainingRootValue.Kind() != VString || barrierValue.Kind() != VString || vm.Store == nil || !actionrelationsDigest(trainingRootValue.AsString()) {
		vm.push(Nil())
		return nil
	}
	barrier := vm.Store.Get(barrierValue.AsString())
	if barrier == nil || !vm.Store.IsA(barrier.Name, "ActionGuardSearchBarrier") {
		vm.push(Nil())
		return nil
	}
	winners := barrier.GetStrings("winnerResults")
	relations := make([]actionrelations.Relation, len(winners))
	relationNames := make([]string, len(winners))
	for index, winnerName := range winners {
		winner := vm.Store.Get(winnerName)
		candidate := vm.Store.Get(winner.GetString("candidate"))
		pattern, patternErr := actionrelations.ParsePattern([]byte(candidate.GetString("pattern")))
		guard, guardErr := actionrelations.ParseGuard([]byte(candidate.GetString("guard")))
		if patternErr != nil || guardErr != nil {
			vm.push(Nil())
			return nil
		}
		relation := actionrelations.Relation{Pattern: pattern, Guard: guard, PositiveObservations: winner.GetStrings("positiveObservationDigests"), NegativeObservations: winner.GetStrings("negativeObservationDigests")}
		canonical, err := relation.CanonicalJSON()
		if err != nil {
			vm.push(Nil())
			return nil
		}
		atoms := make([]string, len(guard.Literals))
		polarities := make([]any, len(guard.Literals))
		for literalIndex, literal := range guard.Literals {
			atoms[literalIndex], polarities[literalIndex] = literal.Atom, literal.Polarity
		}
		name, err := arStoreCanonical(vm, fmt.Sprintf("%s.Relation.%d", requestedValue.AsString(), index), "GuardedActionRelation", canonical, map[string]any{
			"relation": string(canonical), "candidateResult": winnerName, "pattern": string(mustCanonicalPattern(pattern)), "guard": string(mustCanonicalGuard(guard)), "atoms": atoms, "polarities": polarities,
		})
		if err != nil {
			vm.push(Nil())
			return nil
		}
		relations[index], relationNames[index] = relation, name
	}
	artifact, err := actionrelations.NewArtifact(relations, trainingRootValue.AsString())
	if err != nil {
		vm.push(Nil())
		return nil
	}
	canonical, _ := artifact.CanonicalJSON()
	name, err := arStoreCanonical(vm, requestedValue.AsString(), "GuardedActionArtifact", canonical, map[string]any{
		"artifact": string(canonical), "relationUnits": relationNames, "semanticTrainingRoot": trainingRootValue.AsString(), "barrier": barrier.Name,
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	if err := recordActionRelation(vm, 8, 6, "artifact-freeze", [][]byte{[]byte(barrier.GetString("canonicalObject"))}, [][]byte{canonical}); err != nil {
		return err
	}
	vm.push(StringVal(name))
	return nil
}
