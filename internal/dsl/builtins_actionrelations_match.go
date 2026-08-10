package dsl

import (
	"bytes"
	"encoding/json"

	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func bARPatternMatch(vm *VM) error {
	requested, literalRowsValue, bAppValue, aAppValue, bFactsValue, aFactsValue, bOccurrenceValue, aOccurrenceValue, stateValue, relationValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	values := []Value{requested, literalRowsValue, bAppValue, aAppValue, bFactsValue, aFactsValue, bOccurrenceValue, aOccurrenceValue, stateValue, relationValue}
	for index, value := range values {
		if index == 1 {
			if value.Kind() != VList {
				vm.push(Nil())
				return nil
			}
		} else if value.Kind() != VString {
			vm.push(Nil())
			return nil
		}
	}
	relationUnit := vm.Store.Get(relationValue.AsString())
	state, stateErr := actionrelations.ParseState([]byte(stateValue.AsString()))
	a, aErr := actionrelations.ParseOccurrence([]byte(aOccurrenceValue.AsString()))
	b, bErr := actionrelations.ParseOccurrence([]byte(bOccurrenceValue.AsString()))
	if relationUnit == nil || !vm.Store.IsA(relationUnit.Name, "GuardedActionRelation") || stateErr != nil || aErr != nil || bErr != nil {
		vm.push(Nil())
		return nil
	}
	relation, relationErr := actionrelations.ParseRelation([]byte(relationUnit.GetString("relation")))
	pattern, patternErr := actionrelations.PatternFor(a, b)
	patternJSON, _ := pattern.CanonicalJSON()
	relationPatternJSON, _ := relation.Pattern.CanonicalJSON()
	stateDigest, _ := state.Digest()
	aDigest, _ := a.Digest()
	bDigest, _ := b.Digest()
	aApp, bApp := vm.Store.Get(aAppValue.AsString()), vm.Store.Get(bAppValue.AsString())
	aFacts, bFacts := vm.Store.Get(aFactsValue.AsString()), vm.Store.Get(bFactsValue.AsString())
	if !validMatchApplicabilityRow(vm, aApp, stateDigest, aDigest) || !validMatchApplicabilityRow(vm, bApp, stateDigest, bDigest) || !validMatchFacts(vm, aFacts, stateDigest, aDigest) || !validMatchFacts(vm, bFacts, stateDigest, bDigest) {
		vm.push(Nil())
		return nil
	}
	validInputs := relationErr == nil && patternErr == nil && len(state.Events) <= 6 && bytes.Equal(patternJSON, relationPatternJSON) &&
		aApp.GetBool("applicable") && bApp.GetBool("applicable")
	guardDigest, _ := relation.Guard.Digest()
	rows := literalRowsValue.AsList()
	guardTrue := len(rows) == len(relation.Guard.Literals)
	rowDigests := make([]string, len(rows))
	for index, rowValue := range rows {
		if rowValue.Kind() != VString || index >= len(relation.Guard.Literals) {
			guardTrue = false
			continue
		}
		row := vm.Store.Get(rowValue.AsString())
		literal := relation.Guard.Literals[index]
		if row == nil || !vm.Store.IsA(row.Name, "ActionGuardLiteralRow") || row.GetString("guardDigest") != guardDigest || row.GetString("atom") != literal.Atom || row.GetBool("polarity") != literal.Polarity || !row.GetBool("result") {
			guardTrue = false
			continue
		}
		rowDigests[index] = row.GetString("objectDigest")
	}
	result := validInputs && guardTrue
	wire, _ := json.Marshal([]any{"action-relation-match/v1", relationUnit.GetString("objectDigest"), stateDigest, aFacts.GetString("objectDigest"), bFacts.GetString("objectDigest"), aApp.GetString("objectDigest"), bApp.GetString("objectDigest"), rowDigests, result})
	name, err := arStoreCanonical(vm, requested.AsString(), "ActionRelationMatchRow", wire, map[string]any{"relation": relationUnit.Name, "result": result, "stateDigest": stateDigest, "literalRows": valuesToStrings(rows)})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(name))
	return nil
}

func bARCloseRelationUse(vm *VM) error {
	requested, rowsValue, artifactValue := vm.pop(), vm.pop(), vm.pop()
	if requested.Kind() != VString || rowsValue.Kind() != VList || artifactValue.Kind() != VString || vm.Store == nil {
		vm.push(Nil())
		return nil
	}
	artifact := vm.Store.Get(artifactValue.AsString())
	if artifact == nil || !vm.Store.IsA(artifact.Name, "GuardedActionArtifact") || len(rowsValue.AsList()) != len(artifact.GetStrings("relationUnits")) {
		vm.push(Nil())
		return nil
	}
	all := true
	digests := make([]string, len(rowsValue.AsList()))
	for index, value := range rowsValue.AsList() {
		row := vm.Store.Get(value.AsString())
		if value.Kind() != VString || row == nil || !vm.Store.IsA(row.Name, "ActionRelationMatchRow") || row.GetString("relation") != artifact.GetStrings("relationUnits")[index] {
			vm.push(Nil())
			return nil
		}
		digests[index] = row.GetString("objectDigest")
		all = all && row.GetBool("result")
	}
	wire, _ := json.Marshal([]any{"action-relation-use-barrier/v1", artifact.GetString("objectDigest"), digests, all})
	name, err := arStoreCanonical(vm, requested.AsString(), "ActionRelationUseBarrier", wire, map[string]any{"artifact": artifact.Name, "matchRows": valuesToStrings(rowsValue.AsList()), "result": all})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(ListVal([]Value{StringVal(name), BoolVal(all)}))
	return nil
}

func validMatchApplicabilityRow(vm *VM, u *unit.Unit, stateDigest, occurrenceDigest string) bool {
	return u != nil && vm.Store.IsA(u.Name, "ActionApplicabilityRow") && u.GetString("stateDigest") == stateDigest && u.GetString("occurrenceDigest") == occurrenceDigest
}

func validMatchFacts(vm *VM, u *unit.Unit, stateDigest, occurrenceDigest string) bool {
	return u != nil && vm.Store.IsA(u.Name, "ActionLocalFacts") && u.GetString("stateDigest") == stateDigest && u.GetString("occurrenceDigest") == occurrenceDigest
}
