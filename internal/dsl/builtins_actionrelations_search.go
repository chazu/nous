package dsl

import (
	"encoding/json"
	"slices"

	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

var actionRelationSearchPolicies = []string{
	"complete", "lexical-order", "static-rw-sleep", "dynamic-diamond-sleep",
	"nous-guarded-sleep", "no-guard-sleep", "learned-no-use",
}

func bARSearchApplicable(vm *VM) error {
	requested, occurrenceValue, stateValue, nodeValue, policyValue, worldValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	values := []Value{requested, occurrenceValue, stateValue, nodeValue, policyValue, worldValue}
	for _, value := range values {
		if value.Kind() != VString {
			vm.push(Nil())
			return nil
		}
	}
	state, occurrence, ok := arStateOccurrence(stateValue, occurrenceValue)
	node := vm.Store.Get(nodeValue.AsString())
	stateDigest, _ := state.Digest()
	occurrenceDigest, _ := occurrence.Digest()
	if !ok || !validSearchNodeContext(vm, node, worldValue.AsString(), policyValue.AsString(), stateDigest, occurrenceDigest) {
		vm.push(Nil())
		return nil
	}
	applicable, err := actionrelations.Applicable(state, occurrence.Action)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	wire, _ := json.Marshal([]any{"action-applicability-row/v1", stateDigest, occurrenceDigest, applicable, "valid"})
	name, err := arStoreCanonical(vm, requested.AsString(), "ActionApplicabilityRow", wire, map[string]any{
		"stateDigest": stateDigest, "occurrenceDigest": occurrenceDigest, "applicable": applicable,
		"worldDigest": worldValue.AsString(), "policy": policyValue.AsString(), "nodeUnit": node.Name,
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	if err := recordActionRelation(vm, 23, 10, "search-applicable", [][]byte{
		[]byte(worldValue.AsString()), []byte(policyValue.AsString()), []byte(node.GetString("canonicalObject")),
		[]byte(stateValue.AsString()), []byte(occurrenceValue.AsString()),
	}, [][]byte{wire}); err != nil {
		return err
	}
	vm.push(StringVal(name))
	return nil
}

func bARStaticFootprint(vm *VM) error {
	requested, bFactsValue, aFactsValue, bOccurrenceValue, aOccurrenceValue, stateValue, nodeValue, worldValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	values := []Value{requested, bFactsValue, aFactsValue, bOccurrenceValue, aOccurrenceValue, stateValue, nodeValue, worldValue}
	for _, value := range values {
		if value.Kind() != VString {
			vm.push(Nil())
			return nil
		}
	}
	state, a, aOK := arStateOccurrence(stateValue, aOccurrenceValue)
	_, b, bOK := arStateOccurrence(stateValue, bOccurrenceValue)
	stateDigest, _ := state.Digest()
	aDigest, _ := a.Digest()
	bDigest, _ := b.Digest()
	node := vm.Store.Get(nodeValue.AsString())
	aFacts, bFacts := vm.Store.Get(aFactsValue.AsString()), vm.Store.Get(bFactsValue.AsString())
	if !aOK || !bOK || aDigest == bDigest || !validSearchNodeContext(vm, node, worldValue.AsString(), "static-rw-sleep", stateDigest, aDigest) || !slices.Contains(node.GetStrings("remainingOccurrenceDigests"), bDigest) ||
		!validStaticFacts(vm, aFacts, stateDigest, aDigest) || !validStaticFacts(vm, bFacts, stateDigest, bDigest) {
		vm.push(Nil())
		return nil
	}
	left, leftErr := actionrelations.ParseLocalFacts([]byte(aFacts.GetString("facts")))
	right, rightErr := actionrelations.ParseLocalFacts([]byte(bFacts.GetString("facts")))
	result, evalErr := actionrelations.EvaluateAtom("read-write-disjoint", left, right)
	if leftErr != nil || rightErr != nil || evalErr != nil {
		vm.push(Nil())
		return nil
	}
	wire, _ := json.Marshal([]any{"action-static-footprint-row/v1", worldValue.AsString(), node.GetString("objectDigest"), stateDigest, aDigest, bDigest, aFacts.GetString("objectDigest"), bFacts.GetString("objectDigest"), result, "valid"})
	name, err := arStoreCanonical(vm, requested.AsString(), "ActionStaticFootprintRow", wire, map[string]any{
		"worldDigest": worldValue.AsString(), "nodeUnit": node.Name, "stateDigest": stateDigest,
		"aOccurrenceDigest": aDigest, "bOccurrenceDigest": bDigest, "aFacts": aFacts.Name, "bFacts": bFacts.Name, "result": result,
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	if err := recordActionRelation(vm, 24, 10, "static-footprint", [][]byte{
		[]byte(worldValue.AsString()), []byte(node.GetString("canonicalObject")), []byte(stateValue.AsString()),
		[]byte(aOccurrenceValue.AsString()), []byte(bOccurrenceValue.AsString()), []byte(aFacts.GetString("canonicalObject")), []byte(bFacts.GetString("canonicalObject")),
	}, [][]byte{wire}); err != nil {
		return err
	}
	vm.push(StringVal(name))
	return nil
}

func validSearchNodeContext(vm *VM, node *unit.Unit, worldDigest, policy, stateDigest, occurrenceDigest string) bool {
	return vm != nil && vm.Store != nil && node != nil && vm.Store.IsA(node.Name, "ActionRelationSearchNode") &&
		actionrelationsDigest(worldDigest) && slices.Contains(actionRelationSearchPolicies, policy) &&
		node.GetString("worldDigest") == worldDigest && node.GetString("policy") == policy && node.GetString("stateDigest") == stateDigest &&
		slices.Contains(node.GetStrings("remainingOccurrenceDigests"), occurrenceDigest)
}

func validStaticFacts(vm *VM, facts *unit.Unit, stateDigest, occurrenceDigest string) bool {
	return facts != nil && vm.Store.IsA(facts.Name, "ActionLocalFacts") && facts.GetString("stateDigest") == stateDigest && facts.GetString("occurrenceDigest") == occurrenceDigest
}
