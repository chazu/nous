package dsl

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

// These words expose only one-object or one-step operations. In particular,
// none classifies a pair or executes both orders of a diamond.
func init() {
	registerVocabularyWords("actionrelations", map[string]builtinFn{
		"ar-state-valid?":       bARStateValid,
		"ar-action-valid?":      bARActionValid,
		"ar-action-facts":       bARActionFacts,
		"ar-applicable?":        bARApplicable,
		"ar-apply":              bARApply,
		"ar-state-equal?":       bARStateEqual,
		"ar-guard-root":         bARGuardRoot,
		"ar-guard-extend":       bARGuardExtend,
		"ar-candidate-allocate": bARCandidateAllocate,
		"ar-guard-match":        bARGuardMatch,
		"ar-guard-result":       bARGuardResult,
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
	requestedValue, patternValue := vm.pop(), vm.pop()
	if patternValue.Kind() != VString || requestedValue.Kind() != VString || vm.Store == nil {
		vm.push(Nil())
		return nil
	}
	pattern, err := actionrelations.ParsePattern([]byte(patternValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	guard := actionrelations.Guard{}
	data, _ := guard.CanonicalJSON()
	name, err := arStoreCandidate(vm, requestedValue.AsString(), pattern, guard, "", 0)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(ListVal([]Value{StringVal(string(data)), StringVal(name)}))
	return nil
}

func bARGuardExtend(vm *VM) error {
	requestedValue, ordinalValue, polarityValue, atomValue, guardValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	if guardValue.Kind() != VString || atomValue.Kind() != VString || polarityValue.Kind() != VBool || ordinalValue.Kind() != VInt || requestedValue.Kind() != VString || vm.Store == nil {
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
	parentDigest, _ := guard.Digest()
	childDigest, _ := child.Digest()
	edgeWire, _ := json.Marshal([]any{"action-guard-refinement/v1", parentDigest, childDigest, atomValue.AsString(), polarityValue.AsBool(), ordinalValue.AsInt()})
	name, err := arStoreCanonical(vm, requestedValue.AsString(), "ActionGuardRefinement", edgeWire, map[string]any{
		"parentGuard": string(mustCanonicalGuard(guard)), "childGuard": string(data), "atom": atomValue.AsString(), "polarity": polarityValue.AsBool(), "ordinal": ordinalValue.AsInt(),
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(ListVal([]Value{StringVal(string(data)), StringVal(name)}))
	return nil
}

func bARCandidateAllocate(vm *VM) error {
	requestedValue, ordinalValue, parentValue, guardValue, patternValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	if requestedValue.Kind() != VString || ordinalValue.Kind() != VInt || parentValue.Kind() != VString || guardValue.Kind() != VString || patternValue.Kind() != VString || vm.Store == nil {
		vm.push(Nil())
		return nil
	}
	pattern, patternErr := actionrelations.ParsePattern([]byte(patternValue.AsString()))
	guard, guardErr := actionrelations.ParseGuard([]byte(guardValue.AsString()))
	ordinal := ordinalValue.AsInt()
	if patternErr != nil || guardErr != nil || ordinal < 1 || ordinal > 450 || len(guard.Literals) == 0 {
		vm.push(Nil())
		return nil
	}
	parent := vm.Store.Get(parentValue.AsString())
	if parent == nil || !vm.Store.IsA(parent.Name, "ActionGuardCandidate") || parent.GetString("objectDigest") == "" {
		vm.push(Nil())
		return nil
	}
	name, err := arStoreCandidate(vm, requestedValue.AsString(), pattern, guard, parent.Name, ordinal)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(name))
	return nil
}

func arStoreCandidate(vm *VM, requested string, pattern actionrelations.Pattern, guard actionrelations.Guard, parent string, ordinal int) (string, error) {
	patternDigest, _ := pattern.Digest()
	guardDigest, _ := guard.Digest()
	parentDigest := ""
	if parent != "" {
		unit := vm.Store.Get(parent)
		if unit == nil {
			return "", fmt.Errorf("missing parent candidate")
		}
		parentDigest = unit.GetString("objectDigest")
	}
	wire, _ := json.Marshal([]any{"action-guard-candidate/v1", guardDigest, parentDigest, patternDigest, ordinal, len(guard.Literals)})
	return arStoreCanonical(vm, requested, "ActionGuardCandidate", wire, map[string]any{
		"pattern": string(mustCanonicalPattern(pattern)), "guard": string(mustCanonicalGuard(guard)), "parentCandidate": parent, "ordinal": ordinal, "literalCount": len(guard.Literals),
	})
}

func arStoreCanonical(vm *VM, requested, category string, canonical []byte, slots map[string]any) (string, error) {
	if requested == "" {
		return "", fmt.Errorf("empty requested name")
	}
	digestBytes := sha256.Sum256(canonical)
	digest := hex.EncodeToString(digestBytes[:])
	name := requested
	if occupied := vm.Store.Get(name); occupied != nil && (occupied.GetString("canonicalObject") != string(canonical) || !vm.Store.IsA(name, category)) {
		name = requested + "." + digest[:16]
		if collision := vm.Store.Get(name); collision != nil && (collision.GetString("canonicalObject") != string(canonical) || !vm.Store.IsA(name, category)) {
			name = requested + "." + digest
			if vm.Store.Has(name) {
				return "", fmt.Errorf("content-derived name occupied")
			}
		}
	}
	if existing := vm.Store.Get(name); existing != nil {
		return name, nil
	}
	u := unit.New(name)
	u.Set("isA", []string{category, "Anything"})
	u.Set("canonicalObject", string(canonical))
	u.Set("objectDigest", digest)
	for key, value := range slots {
		u.Set(key, value)
	}
	vm.Store.Put(u)
	vm.NewUnits = append(vm.NewUnits, name)
	return name, nil
}

func mustCanonicalPattern(pattern actionrelations.Pattern) []byte {
	data, _ := pattern.CanonicalJSON()
	return data
}
func mustCanonicalGuard(guard actionrelations.Guard) []byte {
	data, _ := guard.CanonicalJSON()
	return data
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
