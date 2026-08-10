package dsl

import (
	"encoding/hex"
	"encoding/json"

	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

// bARObservationAssemble only folds already materialized transition/equality
// rows. It never executes an action or asks production code to classify a pair.
func bARObservationAssemble(vm *VM) error {
	requested, labelValue, equalityValue, aAfterBValue, bAfterAValue, bInitialValue, aInitialValue, bOccurrenceValue, aOccurrenceValue, stateValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	values := []Value{requested, labelValue, equalityValue, aAfterBValue, bAfterAValue, bInitialValue, aInitialValue, bOccurrenceValue, aOccurrenceValue, stateValue}
	for _, value := range values {
		if value.Kind() != VString {
			vm.push(Nil())
			return nil
		}
	}
	if vm.Store == nil {
		vm.push(Nil())
		return nil
	}
	state, stateErr := actionrelations.ParseState([]byte(stateValue.AsString()))
	aOccurrence, aErr := actionrelations.ParseOccurrence([]byte(aOccurrenceValue.AsString()))
	bOccurrence, bErr := actionrelations.ParseOccurrence([]byte(bOccurrenceValue.AsString()))
	orderedA, orderedB, pairErr := actionrelations.CanonicalPair(aOccurrence, bOccurrence)
	if stateErr != nil || aErr != nil || bErr != nil || pairErr != nil || orderedA != aOccurrence || orderedB != bOccurrence || aOccurrence == bOccurrence {
		vm.push(Nil())
		return nil
	}
	stateDigest, _ := state.Digest()
	aDigest, _ := aOccurrence.Digest()
	bDigest, _ := bOccurrence.Digest()
	aInitial := arTransition(vm, aInitialValue.AsString())
	bInitial := arTransition(vm, bInitialValue.AsString())
	if !validInitialTransition(aInitial, stateDigest, aDigest) || !validInitialTransition(bInitial, stateDigest, bDigest) {
		vm.push(Nil())
		return nil
	}
	bAfterA := arOptionalTransition(vm, bAfterAValue.AsString())
	aAfterB := arOptionalTransition(vm, aAfterBValue.AsString())
	equality := arOptionalEquality(vm, equalityValue.AsString())
	label, abDigest, baDigest, valid := reconstructObservation(aInitial, bInitial, bAfterA, aAfterB, equality, aDigest, bDigest)
	if !valid || label != labelValue.AsString() {
		vm.push(Nil())
		return nil
	}
	wire := []any{"action-pair-observation/v1", stateDigest, aDigest, bDigest,
		aInitial.objectDigest, bInitial.objectDigest, optionalDigest(bAfterA), optionalDigest(aAfterB), optionalString(abDigest), optionalString(baDigest), label}
	canonical, _ := json.Marshal(wire)
	name, err := arStoreCanonical(vm, requested.AsString(), "ActionRelationObservation", canonical, map[string]any{
		"stateDigest": stateDigest, "aOccurrenceDigest": aDigest, "bOccurrenceDigest": bDigest, "label": label,
		"aInitialRow": aInitial.name, "bInitialRow": bInitial.name, "bAfterARow": optionalName(bAfterA), "aAfterBRow": optionalName(aAfterB), "equalityRow": equalityValue.AsString(),
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(name))
	return nil
}

type arTransitionEvidence struct {
	name, objectDigest, stateDigest, occurrenceDigest, outcome, outputDigest string
	valid                                                                    bool
}

type arEqualityEvidence struct {
	leftDigest, rightDigest string
	equal                   bool
	valid                   bool
}

func arTransition(vm *VM, name string) arTransitionEvidence {
	u := vm.Store.Get(name)
	if u == nil || !vm.Store.IsA(name, "ActionTransitionRow") || !actionrelationsDigest(u.GetString("objectDigest")) {
		return arTransitionEvidence{}
	}
	return arTransitionEvidence{name: name, objectDigest: u.GetString("objectDigest"), stateDigest: u.GetString("stateDigest"), occurrenceDigest: u.GetString("occurrenceDigest"), outcome: u.GetString("outcome"), outputDigest: u.GetString("outputStateDigest"), valid: true}
}

func arOptionalTransition(vm *VM, name string) *arTransitionEvidence {
	if name == "" {
		return nil
	}
	row := arTransition(vm, name)
	return &row
}

func arOptionalEquality(vm *VM, name string) *arEqualityEvidence {
	if name == "" {
		return nil
	}
	u := vm.Store.Get(name)
	if u == nil || !vm.Store.IsA(name, "ActionStateEqualityRow") {
		return &arEqualityEvidence{}
	}
	return &arEqualityEvidence{leftDigest: u.GetString("leftStateDigest"), rightDigest: u.GetString("rightStateDigest"), equal: u.GetBool("equal"), valid: true}
}

func validInitialTransition(row arTransitionEvidence, stateDigest, occurrenceDigest string) bool {
	return row.valid && row.stateDigest == stateDigest && row.occurrenceDigest == occurrenceDigest && oneOutcome(row)
}

func oneOutcome(row arTransitionEvidence) bool {
	return row.outcome == "applied" && actionrelationsDigest(row.outputDigest) || row.outcome == "inapplicable" && row.outputDigest == ""
}

func reconstructObservation(aInitial, bInitial arTransitionEvidence, bAfterA, aAfterB *arTransitionEvidence, equality *arEqualityEvidence, aDigest, bDigest string) (string, string, string, bool) {
	aOK, bOK := aInitial.outcome == "applied", bInitial.outcome == "applied"
	if !aOK && !bOK {
		return "inapplicable", "", "", bAfterA == nil && aAfterB == nil && equality == nil
	}
	if !aOK {
		valid := bAfterA == nil && validCross(aAfterB, bInitial.outputDigest, aDigest) && equality == nil
		if valid && aAfterB.outcome == "applied" {
			return "b-enables-a", "", "", true
		}
		return "inapplicable", "", "", valid
	}
	if !bOK {
		valid := aAfterB == nil && validCross(bAfterA, aInitial.outputDigest, bDigest) && equality == nil
		if valid && bAfterA.outcome == "applied" {
			return "a-enables-b", "", "", true
		}
		return "inapplicable", "", "", valid
	}
	if !validCross(bAfterA, aInitial.outputDigest, bDigest) || !validCross(aAfterB, bInitial.outputDigest, aDigest) {
		return "", "", "", false
	}
	abOK, baOK := bAfterA.outcome == "applied", aAfterB.outcome == "applied"
	if !abOK && !baOK {
		return "mutual-disables", "", "", equality == nil
	}
	if !abOK {
		return "a-disables-b", "", "", equality == nil
	}
	if !baOK {
		return "b-disables-a", "", "", equality == nil
	}
	if equality == nil || !equality.valid || equality.leftDigest != bAfterA.outputDigest || equality.rightDigest != aAfterB.outputDigest {
		return "", "", "", false
	}
	if equality.equal {
		return "commutes", bAfterA.outputDigest, aAfterB.outputDigest, true
	}
	return "conflicts", bAfterA.outputDigest, aAfterB.outputDigest, true
}

func validCross(row *arTransitionEvidence, stateDigest, occurrenceDigest string) bool {
	return row != nil && row.valid && row.stateDigest == stateDigest && row.occurrenceDigest == occurrenceDigest && oneOutcome(*row)
}

func optionalDigest(row *arTransitionEvidence) any {
	if row == nil {
		return nil
	}
	return row.objectDigest
}

func optionalName(row *arTransitionEvidence) string {
	if row == nil {
		return ""
	}
	return row.name
}

func optionalString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func actionrelationsDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
