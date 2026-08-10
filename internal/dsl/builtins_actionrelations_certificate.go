package dsl

import (
	"encoding/json"

	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

// bARCertificateAssemble folds one explicitly supplied diamond. Unlike the
// oracle and search packages it cannot discover, classify, or enumerate pairs.
func bARCertificateAssemble(vm *VM) error {
	requested, operationRootValue, representativeValue, equalityValue, aAfterBValue, bAfterAValue, bInitialValue, aInitialValue, witnessValue, bOccurrenceValue, aOccurrenceValue, stateValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	values := []Value{requested, operationRootValue, representativeValue, equalityValue, aAfterBValue, bAfterAValue, bInitialValue, aInitialValue, witnessValue, bOccurrenceValue, aOccurrenceValue, stateValue}
	for _, value := range values {
		if value.Kind() != VString {
			vm.push(Nil())
			return nil
		}
	}
	if vm.Store == nil || !actionrelationsDigest(operationRootValue.AsString()) {
		vm.push(Nil())
		return nil
	}
	state, stateErr := actionrelations.ParseState([]byte(stateValue.AsString()))
	a, aErr := actionrelations.ParseOccurrence([]byte(aOccurrenceValue.AsString()))
	b, bErr := actionrelations.ParseOccurrence([]byte(bOccurrenceValue.AsString()))
	representative, representativeErr := actionrelations.ParseOccurrence([]byte(representativeValue.AsString()))
	orderedA, orderedB, pairErr := actionrelations.CanonicalPair(a, b)
	if stateErr != nil || aErr != nil || bErr != nil || representativeErr != nil || pairErr != nil || orderedA != a || orderedB != b || representative != a {
		vm.push(Nil())
		return nil
	}
	witness, witnessErr := parseARWitness([]byte(witnessValue.AsString()))
	if witnessErr != nil {
		vm.push(Nil())
		return nil
	}
	stateDigest, _ := state.Digest()
	aDigest, _ := a.Digest()
	bDigest, _ := b.Digest()
	aInitial, bInitial := arTransition(vm, aInitialValue.AsString()), arTransition(vm, bInitialValue.AsString())
	bAfterA, aAfterB := arOptionalTransition(vm, bAfterAValue.AsString()), arOptionalTransition(vm, aAfterBValue.AsString())
	equality := arOptionalEquality(vm, equalityValue.AsString())
	if !validInitialTransition(aInitial, stateDigest, aDigest) || !validInitialTransition(bInitial, stateDigest, bDigest) {
		vm.push(Nil())
		return nil
	}
	label, abDigest, baDigest, valid := reconstructObservation(aInitial, bInitial, bAfterA, aAfterB, equality, aDigest, bDigest)
	if !valid || label != "commutes" {
		vm.push(Nil())
		return nil
	}
	wire, _ := json.Marshal([]any{actionrelations.CertificateVersion, stateDigest, aDigest, bDigest, witness, abDigest, baDigest, true, aDigest, operationRootValue.AsString()})
	name, err := arStoreCanonical(vm, requested.AsString(), "ActionRelationCertificate", wire, map[string]any{
		"stateDigest": stateDigest, "aOccurrenceDigest": aDigest, "bOccurrenceDigest": bDigest, "witness": witnessValue.AsString(),
		"abStateDigest": abDigest, "baStateDigest": baDigest, "representativeDigest": aDigest, "operationRoot": operationRootValue.AsString(),
		"aInitialRow": aInitial.name, "bInitialRow": bInitial.name, "bAfterARow": bAfterA.name, "aAfterBRow": aAfterB.name, "equalityRow": equalityValue.AsString(),
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(name))
	return nil
}

func parseARWitness(data []byte) ([]any, error) {
	var row []any
	if json.Unmarshal(data, &row) != nil || len(row) != 2 && len(row) != 3 {
		return nil, actionrelations.ErrInvalid
	}
	version, ok := row[0].(string)
	if !ok {
		return nil, actionrelations.ErrInvalid
	}
	switch version {
	case "learned-witness/v1", "static-witness/v1":
		if len(row) != 2 || !digestAny(row[1]) {
			return nil, actionrelations.ErrInvalid
		}
	case "dynamic-witness/v1":
		if len(row) != 3 || row[1] != "all-pairs" || !digestAny(row[2]) {
			return nil, actionrelations.ErrInvalid
		}
	default:
		return nil, actionrelations.ErrInvalid
	}
	return row, nil
}

func digestAny(value any) bool {
	text, ok := value.(string)
	return ok && actionrelationsDigest(text)
}
