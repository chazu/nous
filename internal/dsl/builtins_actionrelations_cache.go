package dsl

import (
	"encoding/json"

	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func bARCacheFinalize(vm *VM) error {
	requested, operationRootValue, attemptValue, missCallValue, bOccurrenceValue, aOccurrenceValue, stateValue, policyValue, worldValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	values := []Value{requested, operationRootValue, attemptValue, missCallValue, bOccurrenceValue, aOccurrenceValue, stateValue, policyValue, worldValue}
	for _, value := range values {
		if value.Kind() != VString {
			vm.push(Nil())
			return nil
		}
	}
	state, stateErr := actionrelations.ParseState([]byte(stateValue.AsString()))
	a, aErr := actionrelations.ParseOccurrence([]byte(aOccurrenceValue.AsString()))
	b, bErr := actionrelations.ParseOccurrence([]byte(bOccurrenceValue.AsString()))
	orderedA, orderedB, pairErr := actionrelations.CanonicalPair(a, b)
	attempt := vm.Store.Get(attemptValue.AsString())
	stateDigest, _ := state.Digest()
	aDigest, _ := a.Digest()
	bDigest, _ := b.Digest()
	minDigest, maxDigest := aDigest, bDigest
	if minDigest > maxDigest {
		minDigest, maxDigest = maxDigest, minDigest
	}
	if stateErr != nil || aErr != nil || bErr != nil || pairErr != nil || orderedA != a || orderedB != b ||
		!actionrelationsDigest(worldValue.AsString()) || !actionrelationsDigest(missCallValue.AsString()) || !actionrelationsDigest(operationRootValue.AsString()) ||
		!slicesContainsString(actionRelationSearchPolicies, policyValue.AsString()) || attempt == nil || !vm.Store.IsA(attempt.Name, "ActionRelationCertificateAttempt") ||
		attempt.GetString("stateDigest") != stateDigest || attempt.GetString("aOccurrenceDigest") != aDigest || attempt.GetString("bOccurrenceDigest") != bDigest || attempt.GetString("operationRoot") != operationRootValue.AsString() || attempt.GetString("status") != "valid" {
		vm.push(Nil())
		return nil
	}
	result := attempt.GetString("result")
	certificateDigest := attempt.GetString("certificateDigest")
	if result != "certified" && result != "not-certified" || result == "certified" && !actionrelationsDigest(certificateDigest) || result == "not-certified" && certificateDigest != actionRelationZeroDigest {
		vm.push(Nil())
		return nil
	}
	wire, _ := json.Marshal([]any{"certificate-cache-row/v3", worldValue.AsString(), policyValue.AsString(), stateDigest, minDigest, maxDigest, missCallValue.AsString(), attempt.GetString("objectDigest"), operationRootValue.AsString(), result, certificateDigest, "valid"})
	name, err := arStoreCanonical(vm, requested.AsString(), "ActionCertificateCacheRow", wire, map[string]any{
		"worldDigest": worldValue.AsString(), "policy": policyValue.AsString(), "stateDigest": stateDigest,
		"minOccurrenceDigest": minDigest, "maxOccurrenceDigest": maxDigest,
		"aOccurrenceDigest": aDigest, "bOccurrenceDigest": bDigest, "missLookupCallID": missCallValue.AsString(),
		"attemptUnit": attempt.Name, "operationRoot": operationRootValue.AsString(), "result": result, "certificateDigest": certificateDigest,
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	if err := recordActionRelation(vm, 25, 11, "certificate-cache-finalize", nil, [][]byte{wire}); err != nil {
		return err
	}
	vm.push(StringVal(name))
	return nil
}

func slicesContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
