package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestObservationAssemblerFoldsExplicitCommutingDiamond(t *testing.T) {
	store := actionRelationTestStore()
	vm := &VM{Store: store}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c1", N: 1}}
	stateJSON, _ := state.CanonicalJSON()
	aJSON, _ := a.CanonicalJSON()
	bJSON, _ := b.CanonicalJSON()
	aInitial, afterA := recordARTransition(t, vm, stateJSON, aJSON, "A.Initial")
	bInitial, afterB := recordARTransition(t, vm, stateJSON, bJSON, "B.Initial")
	bAfterA, ab := recordARTransition(t, vm, afterA, bJSON, "B.AfterA")
	aAfterB, ba := recordARTransition(t, vm, afterB, aJSON, "A.AfterB")
	equality := recordAREquality(t, vm, ab, ba, "AB.Equals.BA")
	vm.stack = []Value{
		StringVal(string(stateJSON)), StringVal(string(aJSON)), StringVal(string(bJSON)),
		StringVal(aInitial), StringVal(bInitial), StringVal(bAfterA), StringVal(aAfterB), StringVal(equality),
		StringVal("commutes"), StringVal("AR.Observation.Commutes"),
	}
	if err := bARObservationAssemble(vm); err != nil {
		t.Fatal(err)
	}
	name := vm.pop().AsString()
	observation := store.Get(name)
	if observation == nil || observation.GetString("label") != "commutes" || observation.GetString("objectDigest") == "" {
		t.Fatalf("observation=%#v", observation)
	}
	vm.stack = []Value{
		StringVal(string(stateJSON)), StringVal(string(aJSON)), StringVal(string(bJSON)),
		StringVal(aInitial), StringVal(bInitial), StringVal(bAfterA), StringVal(aAfterB), StringVal(equality),
		StringVal("conflicts"), StringVal("AR.Observation.Forged"),
	}
	if err := bARObservationAssemble(vm); err != nil || !vm.pop().IsNil() || store.Has("AR.Observation.Forged") {
		t.Fatal("assembler accepted a forged label")
	}
}

func TestObservationAssemblerRequiresOnlyLegalEnablingCrossStep(t *testing.T) {
	store := actionRelationTestStore()
	vm := &VM{Store: store}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}}}
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "check", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 1}}
	stateJSON, _ := state.CanonicalJSON()
	aJSON, _ := a.CanonicalJSON()
	bJSON, _ := b.CanonicalJSON()
	aInitial, _ := recordARTransition(t, vm, stateJSON, aJSON, "Enable.A.Initial")
	bInitial, afterB := recordARTransition(t, vm, stateJSON, bJSON, "Enable.B.Initial")
	aAfterB, _ := recordARTransition(t, vm, afterB, aJSON, "Enable.A.AfterB")
	vm.stack = []Value{
		StringVal(string(stateJSON)), StringVal(string(aJSON)), StringVal(string(bJSON)),
		StringVal(aInitial), StringVal(bInitial), StringVal(""), StringVal(aAfterB), StringVal(""),
		StringVal("b-enables-a"), StringVal("AR.Observation.Enables"),
	}
	if err := bARObservationAssemble(vm); err != nil {
		t.Fatal(err)
	}
	if value := vm.pop(); value.Kind() != VString || store.Get(value.AsString()).GetString("label") != "b-enables-a" {
		t.Fatalf("enabling observation=%v", value)
	}
}

func TestGuardRowsNameFactsAndFoldInLiteralOrder(t *testing.T) {
	store := actionRelationTestStore()
	vm := &VM{Store: store}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c1", N: 1}}
	stateJSON, _ := state.CanonicalJSON()
	aJSON, _ := a.CanonicalJSON()
	bJSON, _ := b.CanonicalJSON()
	aInitial, afterA := recordARTransition(t, vm, stateJSON, aJSON, "Guard.A.Initial")
	bInitial, afterB := recordARTransition(t, vm, stateJSON, bJSON, "Guard.B.Initial")
	bAfterA, ab := recordARTransition(t, vm, afterA, bJSON, "Guard.B.AfterA")
	aAfterB, ba := recordARTransition(t, vm, afterB, aJSON, "Guard.A.AfterB")
	equality := recordAREquality(t, vm, ab, ba, "Guard.Equality")
	vm.stack = []Value{StringVal(string(stateJSON)), StringVal(string(aJSON)), StringVal(string(bJSON)), StringVal(aInitial), StringVal(bInitial), StringVal(bAfterA), StringVal(aAfterB), StringVal(equality), StringVal("commutes"), StringVal("Guard.Observation")}
	_ = bARObservationAssemble(vm)
	observationName := vm.pop().AsString()
	vm.stack = []Value{StringVal(string(stateJSON)), StringVal(string(aJSON)), StringVal("Guard.Facts.A")}
	_ = bARActionFacts(vm)
	aFacts := vm.pop().AsString()
	vm.stack = []Value{StringVal(string(stateJSON)), StringVal(string(bJSON)), StringVal("Guard.Facts.B")}
	_ = bARActionFacts(vm)
	bFacts := vm.pop().AsString()
	store.SetSlot(observationName, "aFacts", aFacts)
	store.SetSlot(observationName, "bFacts", bFacts)
	guard := actionrelations.Guard{Literals: []actionrelations.Literal{{Atom: "read-write-disjoint", Polarity: true}}}
	guardJSON, _ := guard.CanonicalJSON()
	vm.stack = []Value{StringVal(string(guardJSON)), StringVal(observationName), StringVal(aFacts), StringVal(bFacts), StringVal("read-write-disjoint"), BoolVal(true), StringVal("Guard.Literal")}
	if err := bARGuardMatch(vm); err != nil {
		t.Fatal(err)
	}
	literalName := vm.pop().AsString()
	if !store.Get(literalName).GetBool("result") {
		t.Fatal("expected true literal")
	}
	vm.stack = []Value{StringVal(string(guardJSON)), StringVal(observationName), ListVal([]Value{StringVal(literalName)}), StringVal("Guard.Result")}
	if err := bARGuardResult(vm); err != nil {
		t.Fatal(err)
	}
	resultName := vm.pop().AsString()
	if !store.Get(resultName).GetBool("result") {
		t.Fatal("expected true conjunction")
	}
}

func TestCertificateAssemblerRequiresCompleteExplicitDiamond(t *testing.T) {
	store := actionRelationTestStore()
	vm := &VM{Store: store}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c1", N: 1}}
	stateJSON, _ := state.CanonicalJSON()
	aJSON, _ := a.CanonicalJSON()
	bJSON, _ := b.CanonicalJSON()
	aInitial, afterA := recordARTransition(t, vm, stateJSON, aJSON, "Cert.A.Initial")
	bInitial, afterB := recordARTransition(t, vm, stateJSON, bJSON, "Cert.B.Initial")
	bAfterA, ab := recordARTransition(t, vm, afterA, bJSON, "Cert.B.AfterA")
	aAfterB, ba := recordARTransition(t, vm, afterB, aJSON, "Cert.A.AfterB")
	equality := recordAREquality(t, vm, ab, ba, "Cert.Equality")
	witnessDigest := sha256.Sum256([]byte("dynamic-candidate"))
	witnessJSON, _ := json.Marshal([]any{"dynamic-witness/v1", "all-pairs", hex.EncodeToString(witnessDigest[:])})
	operationDigest := sha256.Sum256([]byte("certificate-operations"))
	operationRoot := hex.EncodeToString(operationDigest[:])
	vm.stack = []Value{
		StringVal(string(stateJSON)), StringVal(string(aJSON)), StringVal(string(bJSON)), StringVal(string(witnessJSON)),
		StringVal(aInitial), StringVal(bInitial), StringVal(bAfterA), StringVal(aAfterB), StringVal(equality),
		StringVal(string(aJSON)), StringVal(operationRoot), StringVal("AR.Certificate"),
	}
	if err := bARCertificateAssemble(vm); err != nil {
		t.Fatal(err)
	}
	attempt := store.Get(vm.pop().AsString())
	if attempt == nil || attempt.GetString("result") != "certified" {
		t.Fatalf("attempt=%#v", attempt)
	}
	if certificate := store.Get(attempt.GetString("certificateUnit")); certificate == nil || certificate.GetString("representativeDigest") == "" {
		t.Fatalf("certificate=%#v", certificate)
	}
	vm.stack = []Value{
		StringVal(string(stateJSON)), StringVal(string(aJSON)), StringVal(string(bJSON)), StringVal(string(witnessJSON)),
		StringVal(aInitial), StringVal(bInitial), StringVal(""), StringVal(aAfterB), StringVal(equality),
		StringVal(string(aJSON)), StringVal(operationRoot), StringVal("AR.Certificate.Forged"),
	}
	if err := bARCertificateAssemble(vm); err != nil {
		t.Fatal(err)
	}
	forged := store.Get(vm.pop().AsString())
	if forged == nil || forged.GetString("result") != "invalid" || forged.GetString("certificateUnit") != "" {
		t.Fatal("omitted crossed transition did not produce an invalid attempt")
	}
}

func recordARTransition(t *testing.T, vm *VM, stateJSON, occurrenceJSON []byte, prefix string) (string, []byte) {
	t.Helper()
	vm.stack = []Value{StringVal(string(stateJSON)), StringVal(string(occurrenceJSON)), StringVal(prefix + ".Applicable")}
	if err := bARApplicable(vm); err != nil {
		t.Fatal(err)
	}
	applicability := vm.pop().AsString()
	vm.stack = []Value{StringVal(string(stateJSON)), StringVal(string(occurrenceJSON)), StringVal(applicability), StringVal(prefix + ".Transition"), StringVal(prefix + ".State")}
	if err := bARApply(vm); err != nil {
		t.Fatal(err)
	}
	result := vm.pop().AsList()
	transition := result[0].AsString()
	if result[1].AsString() == "" {
		return transition, nil
	}
	return transition, []byte(vm.Store.Get(result[1].AsString()).GetString("state"))
}

func recordAREquality(t *testing.T, vm *VM, left, right []byte, name string) string {
	t.Helper()
	vm.stack = []Value{StringVal(string(left)), StringVal(string(right)), StringVal(name)}
	if err := bARStateEqual(vm); err != nil {
		t.Fatal(err)
	}
	return vm.pop().AsString()
}
