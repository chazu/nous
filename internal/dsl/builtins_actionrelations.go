package dsl

import (
	"bytes"

	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

// These words expose only one-object or one-step operations. In particular,
// none classifies a pair or executes both orders of a diamond.
func init() {
	registerVocabularyWords("actionrelations", map[string]builtinFn{
		"ar-state-valid?":  bARStateValid,
		"ar-action-valid?": bARActionValid,
		"ar-action-facts":  bARActionFacts,
		"ar-applicable?":   bARApplicable,
		"ar-apply":         bARApply,
		"ar-state-equal?":  bARStateEqual,
		"ar-guard-root":    bARGuardRoot,
		"ar-guard-extend":  bARGuardExtend,
		"ar-guard-match":   bARGuardMatch,
		"ar-guard-result":  bARGuardResult,
	})
}

func bARStateValid(vm *VM) error {
	value := vm.pop()
	if value.Kind() != VString {
		vm.push(BoolVal(false))
		return nil
	}
	_, err := actionrelations.ParseState([]byte(value.AsString()))
	vm.push(BoolVal(err == nil))
	return nil
}

func bARActionValid(vm *VM) error {
	value := vm.pop()
	if value.Kind() != VString {
		vm.push(BoolVal(false))
		return nil
	}
	data := []byte(value.AsString())
	_, presentationErr := actionrelations.ParseAction(data)
	_, semanticErr := actionrelations.ParseSemanticAction(data)
	_, occurrenceErr := actionrelations.ParseOccurrence(data)
	vm.push(BoolVal(presentationErr == nil || semanticErr == nil || occurrenceErr == nil))
	return nil
}

func bARActionFacts(vm *VM) error {
	occurrenceValue, stateValue := vm.pop(), vm.pop()
	if occurrenceValue.Kind() != VString || stateValue.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	state, stateErr := actionrelations.ParseState([]byte(stateValue.AsString()))
	occurrence, occurrenceErr := actionrelations.ParseOccurrence([]byte(occurrenceValue.AsString()))
	if stateErr != nil || occurrenceErr != nil {
		vm.push(Nil())
		return nil
	}
	facts, err := actionrelations.Facts(state, occurrence)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	data, _ := facts.CanonicalJSON()
	vm.push(StringVal(string(data)))
	return nil
}

func bARApplicable(vm *VM) error {
	actionValue, stateValue := vm.pop(), vm.pop()
	state, action, ok := arStateAction(stateValue, actionValue)
	if !ok {
		vm.push(BoolVal(false))
		return nil
	}
	applicable, err := actionrelations.Applicable(state, action)
	vm.push(BoolVal(err == nil && applicable))
	return nil
}

func bARApply(vm *VM) error {
	claimedValue, actionValue, stateValue := vm.pop(), vm.pop(), vm.pop()
	state, action, ok := arStateAction(stateValue, actionValue)
	if !ok || claimedValue.Kind() != VBool {
		vm.push(Nil())
		return nil
	}
	applicable, err := actionrelations.Applicable(state, action)
	if err != nil || applicable != claimedValue.AsBool() || !applicable {
		vm.push(Nil())
		return nil
	}
	next, outcome, err := actionrelations.Apply(state, action)
	if err != nil || outcome != "applied" {
		vm.push(Nil())
		return nil
	}
	data, _ := next.CanonicalJSON()
	vm.push(StringVal(string(data)))
	return nil
}

func bARStateEqual(vm *VM) error {
	rightValue, leftValue := vm.pop(), vm.pop()
	if leftValue.Kind() != VString || rightValue.Kind() != VString {
		vm.push(BoolVal(false))
		return nil
	}
	left, leftErr := actionrelations.ParseState([]byte(leftValue.AsString()))
	right, rightErr := actionrelations.ParseState([]byte(rightValue.AsString()))
	if leftErr != nil || rightErr != nil {
		vm.push(BoolVal(false))
		return nil
	}
	a, _ := left.CanonicalJSON()
	b, _ := right.CanonicalJSON()
	vm.push(BoolVal(bytes.Equal(a, b)))
	return nil
}

func bARGuardRoot(vm *VM) error {
	patternValue := vm.pop()
	if patternValue.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	if _, err := actionrelations.ParsePattern([]byte(patternValue.AsString())); err != nil {
		vm.push(Nil())
		return nil
	}
	data, _ := (actionrelations.Guard{}).CanonicalJSON()
	vm.push(StringVal(string(data)))
	return nil
}

func bARGuardExtend(vm *VM) error {
	polarityValue, atomValue, guardValue := vm.pop(), vm.pop(), vm.pop()
	if guardValue.Kind() != VString || atomValue.Kind() != VString || polarityValue.Kind() != VBool {
		vm.push(Nil())
		return nil
	}
	guard, err := actionrelations.ParseGuard([]byte(guardValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	child, err := guard.Extend(actionrelations.Literal{Atom: atomValue.AsString(), Polarity: polarityValue.AsBool()})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	data, _ := child.CanonicalJSON()
	vm.push(StringVal(string(data)))
	return nil
}

func bARGuardMatch(vm *VM) error {
	rightValue, leftValue, polarityValue, atomValue := vm.pop(), vm.pop(), vm.pop(), vm.pop()
	if rightValue.Kind() != VString || leftValue.Kind() != VString || polarityValue.Kind() != VBool || atomValue.Kind() != VString {
		vm.push(BoolVal(false))
		return nil
	}
	left, leftErr := actionrelations.ParseLocalFacts([]byte(leftValue.AsString()))
	right, rightErr := actionrelations.ParseLocalFacts([]byte(rightValue.AsString()))
	value, err := actionrelations.EvaluateAtom(atomValue.AsString(), left, right)
	vm.push(BoolVal(leftErr == nil && rightErr == nil && err == nil && value == polarityValue.AsBool()))
	return nil
}

func bARGuardResult(vm *VM) error {
	rowsValue, guardValue := vm.pop(), vm.pop()
	if guardValue.Kind() != VString || rowsValue.Kind() != VList {
		vm.push(BoolVal(false))
		return nil
	}
	guard, err := actionrelations.ParseGuard([]byte(guardValue.AsString()))
	rows := rowsValue.AsList()
	if err != nil || len(rows) != len(guard.Literals) {
		vm.push(BoolVal(false))
		return nil
	}
	for _, row := range rows {
		if row.Kind() != VBool || !row.AsBool() {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(true))
	return nil
}

func arStateAction(stateValue, actionValue Value) (actionrelations.State, actionrelations.SemanticAction, bool) {
	if stateValue.Kind() != VString || actionValue.Kind() != VString {
		return actionrelations.State{}, actionrelations.SemanticAction{}, false
	}
	state, stateErr := actionrelations.ParseState([]byte(stateValue.AsString()))
	action, actionErr := actionrelations.ParseSemanticAction([]byte(actionValue.AsString()))
	return state, action, stateErr == nil && actionErr == nil
}
