package dsl

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

const actionRelationZeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

// These words expose only one-object or one-step operations. In particular,
// none classifies a pair or executes both orders of a diamond.
func init() {
	registerVocabularyWords("actionrelations", map[string]builtinFn{
		"ar-state-valid?":         bARStateValid,
		"ar-action-valid?":        bARActionValid,
		"ar-action-facts":         bARActionFacts,
		"ar-applicable?":          bARApplicable,
		"ar-apply":                bARApply,
		"ar-state-equal?":         bARStateEqual,
		"ar-guard-root":           bARGuardRoot,
		"ar-guard-extend":         bARGuardExtend,
		"ar-candidate-allocate":   bARCandidateAllocate,
		"ar-guard-match":          bARGuardMatch,
		"ar-guard-result":         bARGuardResult,
		"ar-observation-assemble": bARObservationAssemble,
		"ar-candidate-result":     bARCandidateResult,
		"ar-close-guard-search":   bARCloseGuardSearch,
		"ar-freeze-relation":      bARFreezeRelation,
		"ar-certificate-assemble": bARCertificateAssemble,
		"ar-pattern-match":        bARPatternMatch,
		"ar-close-relation-use":   bARCloseRelationUse,
		"ar-meter":                bARMeter,
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
	requestedValue, occurrenceValue, stateValue := vm.pop(), vm.pop(), vm.pop()
	if requestedValue.Kind() != VString || occurrenceValue.Kind() != VString || stateValue.Kind() != VString || vm.Store == nil {
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
	name, err := arStoreCanonical(vm, requestedValue.AsString(), "ActionLocalFacts", data, map[string]any{
		"stateDigest": facts.StateDigest, "occurrenceDigest": facts.OccurrenceDigest, "facts": string(data),
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(StringVal(name))
	return nil
}

func bARApplicable(vm *VM) error {
	requestedValue, occurrenceValue, stateValue := vm.pop(), vm.pop(), vm.pop()
	state, occurrence, ok := arStateOccurrence(stateValue, occurrenceValue)
	if !ok || requestedValue.Kind() != VString || vm.Store == nil {
		vm.push(Nil())
		return nil
	}
	applicable, err := actionrelations.Applicable(state, occurrence.Action)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	stateDigest, _ := state.Digest()
	occurrenceDigest, _ := occurrence.Digest()
	wire, _ := json.Marshal([]any{"action-applicability-row/v1", stateDigest, occurrenceDigest, applicable, "valid"})
	name, err := arStoreCanonical(vm, requestedValue.AsString(), "ActionApplicabilityRow", wire, map[string]any{
		"stateDigest": stateDigest, "occurrenceDigest": occurrenceDigest, "applicable": applicable,
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	code := actionRelationPhaseCode(vm, 5, 13, 21, 5)
	counter := uint8(4)
	if code != 5 {
		counter = 10
	}
	if err := recordActionRelation(vm, code, counter, "applicable", [][]byte{[]byte(stateValue.AsString()), []byte(occurrenceValue.AsString())}, [][]byte{wire}); err != nil {
		return err
	}
	vm.push(StringVal(name))
	return nil
}

func bARApply(vm *VM) error {
	stateRequest, transitionRequest, applicabilityValue, occurrenceValue, stateValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	state, occurrence, ok := arStateOccurrence(stateValue, occurrenceValue)
	if !ok || applicabilityValue.Kind() != VString || transitionRequest.Kind() != VString || stateRequest.Kind() != VString || vm.Store == nil {
		vm.push(Nil())
		return nil
	}
	applicability := vm.Store.Get(applicabilityValue.AsString())
	stateDigest, _ := state.Digest()
	occurrenceDigest, _ := occurrence.Digest()
	if applicability == nil || !vm.Store.IsA(applicability.Name, "ActionApplicabilityRow") || applicability.GetString("stateDigest") != stateDigest || applicability.GetString("occurrenceDigest") != occurrenceDigest {
		vm.push(Nil())
		return nil
	}
	next, outcome, err := actionrelations.Apply(state, occurrence.Action)
	if err != nil || (outcome == "applied") != applicability.GetBool("applicable") {
		vm.push(Nil())
		return nil
	}
	outputDigest := ""
	outputName := ""
	if outcome == "applied" {
		data, _ := next.CanonicalJSON()
		outputDigest, _ = next.Digest()
		outputName, err = arStoreCanonical(vm, stateRequest.AsString(), "FiniteActionState", data, map[string]any{"state": string(data), "stateDigest": outputDigest})
		if err != nil {
			vm.push(Nil())
			return nil
		}
	}
	applicabilityDigest := applicability.GetString("objectDigest")
	resultStateDigest := outputDigest
	if resultStateDigest == "" {
		resultStateDigest = actionRelationZeroDigest
	}
	wire, _ := json.Marshal([]any{"action-transition-row/v1", stateDigest, occurrenceDigest, applicabilityDigest, resultStateDigest, outcome})
	transitionName, err := arStoreCanonical(vm, transitionRequest.AsString(), "ActionTransitionRow", wire, map[string]any{
		"stateDigest": stateDigest, "occurrenceDigest": occurrenceDigest, "applicabilityRow": applicability.Name, "outcome": outcome, "outputState": outputName, "outputStateDigest": outputDigest,
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	code := actionRelationPhaseCode(vm, 4, 12, 11, 4)
	outputs := [][]byte{wire}
	if outputName != "" {
		outputs = append(outputs, []byte(vm.Store.Get(outputName).GetString("canonicalObject")))
	}
	counter := uint8(3)
	if code == 11 {
		counter = 8
	} else if code == 12 {
		counter = 9
	}
	if err := recordActionRelation(vm, code, counter, "apply", [][]byte{[]byte(stateValue.AsString()), []byte(occurrenceValue.AsString())}, outputs); err != nil {
		return err
	}
	vm.push(ListVal([]Value{StringVal(transitionName), StringVal(outputName)}))
	return nil
}

func bARStateEqual(vm *VM) error {
	requestedValue, rightValue, leftValue := vm.pop(), vm.pop(), vm.pop()
	if leftValue.Kind() != VString || rightValue.Kind() != VString || requestedValue.Kind() != VString || vm.Store == nil {
		vm.push(Nil())
		return nil
	}
	left, leftErr := actionrelations.ParseState([]byte(leftValue.AsString()))
	right, rightErr := actionrelations.ParseState([]byte(rightValue.AsString()))
	if leftErr != nil || rightErr != nil {
		vm.push(Nil())
		return nil
	}
	a, _ := left.CanonicalJSON()
	b, _ := right.CanonicalJSON()
	equal := bytes.Equal(a, b)
	leftDigest, _ := left.Digest()
	rightDigest, _ := right.Digest()
	wire, _ := json.Marshal([]any{"action-state-equality-row/v1", leftDigest, rightDigest, equal, "valid"})
	name, err := arStoreCanonical(vm, requestedValue.AsString(), "ActionStateEqualityRow", wire, map[string]any{
		"leftStateDigest": leftDigest, "rightStateDigest": rightDigest, "equal": equal,
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	code := actionRelationPhaseCode(vm, 6, 14, 14, 6)
	counter := uint8(4)
	if code != 6 {
		counter = 10
	}
	if err := recordActionRelation(vm, code, counter, "state-equality", [][]byte{[]byte(leftValue.AsString()), []byte(rightValue.AsString())}, [][]byte{wire}); err != nil {
		return err
	}
	vm.push(StringVal(name))
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
	if _, err := arStoreCanonical(vm, requestedValue.AsString()+".Guard", "ActionGuard", data, map[string]any{"guard": string(data)}); err != nil {
		vm.push(Nil())
		return nil
	}
	name, err := arStoreCandidate(vm, requestedValue.AsString(), pattern, guard, "", 0)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	candidate := vm.Store.Get(name)
	if err := recordActionRelation(vm, 1, 1, "guard-root", [][]byte{[]byte(patternValue.AsString())}, [][]byte{data, []byte(candidate.GetString("canonicalObject"))}); err != nil {
		return err
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
	if _, err := arStoreCanonical(vm, requestedValue.AsString()+".Guard", "ActionGuard", data, map[string]any{"guard": string(data)}); err != nil {
		vm.push(Nil())
		return nil
	}
	name, err := arStoreCanonical(vm, requestedValue.AsString(), "ActionGuardRefinement", edgeWire, map[string]any{
		"parentGuard": string(mustCanonicalGuard(guard)), "childGuard": string(data), "atom": atomValue.AsString(), "polarity": polarityValue.AsBool(), "ordinal": ordinalValue.AsInt(),
	})
	if err != nil {
		vm.push(Nil())
		return nil
	}
	if err := recordActionRelation(vm, 3, 2, "guard-extend", [][]byte{mustCanonicalGuard(guard)}, [][]byte{data, edgeWire}); err != nil {
		return err
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
	if err := recordActionRelation(vm, 2, 1, "candidate-allocate", [][]byte{[]byte(patternValue.AsString()), []byte(guardValue.AsString())}, [][]byte{[]byte(vm.Store.Get(name).GetString("canonicalObject"))}); err != nil {
		return err
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
	atoms := make([]string, len(guard.Literals))
	polarities := make([]any, len(guard.Literals))
	for index, literal := range guard.Literals {
		atoms[index], polarities[index] = literal.Atom, literal.Polarity
	}
	return arStoreCanonical(vm, requested, "ActionGuardCandidate", wire, map[string]any{
		"pattern": string(mustCanonicalPattern(pattern)), "guard": string(mustCanonicalGuard(guard)), "parentCandidate": parent, "ordinal": ordinal, "literalCount": len(guard.Literals), "atoms": atoms, "polarities": polarities,
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
	requestedValue, polarityValue, atomValue, rightValue, leftValue, observationValue, guardValue := vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop(), vm.pop()
	if requestedValue.Kind() != VString || rightValue.Kind() != VString || leftValue.Kind() != VString || observationValue.Kind() != VString || guardValue.Kind() != VString || polarityValue.Kind() != VBool || atomValue.Kind() != VString || vm.Store == nil {
		vm.push(Nil())
		return nil
	}
	guard, guardErr := actionrelations.ParseGuard([]byte(guardValue.AsString()))
	observation := vm.Store.Get(observationValue.AsString())
	leftUnit, rightUnit := vm.Store.Get(leftValue.AsString()), vm.Store.Get(rightValue.AsString())
	if guardErr != nil || observation == nil || leftUnit == nil || rightUnit == nil || !actionRelationFactContext(vm, observation.Name) || observation.GetString("aFacts") != leftUnit.Name || observation.GetString("bFacts") != rightUnit.Name || !actionrelationsDigest(observation.GetString("objectDigest")) {
		vm.push(Nil())
		return nil
	}
	literal := actionrelations.Literal{Atom: atomValue.AsString(), Polarity: polarityValue.AsBool()}
	if !slices.Contains(guard.Literals, literal) {
		vm.push(Nil())
		return nil
	}
	left, leftErr := actionrelations.ParseLocalFacts([]byte(leftUnit.GetString("facts")))
	right, rightErr := actionrelations.ParseLocalFacts([]byte(rightUnit.GetString("facts")))
	value, err := actionrelations.EvaluateAtom(atomValue.AsString(), left, right)
	if leftErr != nil || rightErr != nil || err != nil {
		vm.push(Nil())
		return nil
	}
	result := value == polarityValue.AsBool()
	guardDigest, _ := guard.Digest()
	code := actionRelationPhaseCode(vm, 7, 15, 15, 7)
	category := "ActionGuardLiteralRow"
	wireRow := []any{"action-guard-literal-row/v1", guardDigest, observation.GetString("objectDigest"), leftUnit.GetString("objectDigest"), rightUnit.GetString("objectDigest"), literal.Atom, literal.Polarity, result}
	if code == 15 {
		category = "ActionLiteralEvaluationRow"
		wireRow = []any{"action-literal-evaluation-row/v1", left.StateDigest, leftUnit.GetString("objectDigest"), rightUnit.GetString("objectDigest"), literal.Atom, literal.Polarity, result, "valid"}
	}
	wire, _ := json.Marshal(wireRow)
	name, storeErr := arStoreCanonical(vm, requestedValue.AsString(), category, wire, map[string]any{
		"guardDigest": guardDigest, "observationDigest": observation.GetString("objectDigest"), "aFactsDigest": leftUnit.GetString("objectDigest"), "bFactsDigest": rightUnit.GetString("objectDigest"), "atom": literal.Atom, "polarity": literal.Polarity, "result": result,
	})
	if storeErr != nil {
		vm.push(Nil())
		return nil
	}
	counter := uint8(5)
	if code == 15 {
		counter = 10
	}
	if err := recordActionRelation(vm, code, counter, "guard-literal", [][]byte{[]byte(guardValue.AsString()), []byte(leftUnit.GetString("facts")), []byte(rightUnit.GetString("facts"))}, [][]byte{wire}); err != nil {
		return err
	}
	vm.push(StringVal(name))
	return nil
}

func bARGuardResult(vm *VM) error {
	requestedValue, rowsValue, observationValue, guardValue := vm.pop(), vm.pop(), vm.pop(), vm.pop()
	if requestedValue.Kind() != VString || guardValue.Kind() != VString || observationValue.Kind() != VString || rowsValue.Kind() != VList || vm.Store == nil {
		vm.push(Nil())
		return nil
	}
	guard, err := actionrelations.ParseGuard([]byte(guardValue.AsString()))
	observation := vm.Store.Get(observationValue.AsString())
	rows := rowsValue.AsList()
	if err != nil || observation == nil || !actionRelationFactContext(vm, observation.Name) || !actionrelationsDigest(observation.GetString("objectDigest")) || len(rows) != len(guard.Literals) {
		vm.push(Nil())
		return nil
	}
	guardDigest, _ := guard.Digest()
	result := true
	rowDigests := make([]string, len(rows))
	for index, row := range rows {
		if row.Kind() != VString {
			vm.push(Nil())
			return nil
		}
		u := vm.Store.Get(row.AsString())
		literal := guard.Literals[index]
		if u == nil || !vm.Store.IsA(u.Name, "ActionGuardLiteralRow") || u.GetString("guardDigest") != guardDigest || u.GetString("observationDigest") != observation.GetString("objectDigest") || u.GetString("atom") != literal.Atom || u.GetBool("polarity") != literal.Polarity {
			vm.push(Nil())
			return nil
		}
		rowDigests[index] = u.GetString("objectDigest")
		result = result && u.GetBool("result")
	}
	wire, _ := json.Marshal([]any{"action-guard-result/v1", guardDigest, observation.GetString("objectDigest"), rowDigests, result})
	name, storeErr := arStoreCanonical(vm, requestedValue.AsString(), "ActionGuardResult", wire, map[string]any{
		"guardDigest": guardDigest, "observationDigest": observation.GetString("objectDigest"), "literalRows": valuesToStrings(rows), "result": result,
	})
	if storeErr != nil {
		vm.push(Nil())
		return nil
	}
	if err := recordActionRelation(vm, 22, 5, "guard-result", [][]byte{[]byte(guardValue.AsString())}, [][]byte{wire}); err != nil {
		return err
	}
	vm.push(StringVal(name))
	return nil
}

func valuesToStrings(values []Value) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = value.AsString()
	}
	return result
}

func actionRelationFactContext(vm *VM, name string) bool {
	return vm.Store.IsA(name, "ActionRelationObservation") || vm.Store.IsA(name, "ActionRelationMatchRequest")
}

func arStateOccurrence(stateValue, occurrenceValue Value) (actionrelations.State, actionrelations.Occurrence, bool) {
	if stateValue.Kind() != VString || occurrenceValue.Kind() != VString {
		return actionrelations.State{}, actionrelations.Occurrence{}, false
	}
	state, stateErr := actionrelations.ParseState([]byte(stateValue.AsString()))
	occurrence, occurrenceErr := actionrelations.ParseOccurrence([]byte(occurrenceValue.AsString()))
	return state, occurrence, stateErr == nil && occurrenceErr == nil
}
