package dsl

import (
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func actionRelationTestStore() *unit.Store {
	store := protocolTestStore("")
	marker := unit.New("GuardedActionRelationVocabulary")
	marker.Set("isA", []string{"Vocabulary", "Anything"})
	marker.Set("dslExtension", "actionrelations")
	store.Put(marker)
	return store
}

func TestActionRelationWordsAreScoped(t *testing.T) {
	empty := protocolTestStore("")
	if _, ok := NewVM(empty, agenda.New(), nil).words["ar-apply"]; ok {
		t.Fatal("action-relation word leaked into base VM")
	}
	vm := NewVM(actionRelationTestStore(), agenda.New(), nil)
	if err := vm.InitError(); err != nil {
		t.Fatal(err)
	}
	if _, ok := vm.words["ar-apply"]; !ok {
		t.Fatal("scoped word absent")
	}
}

func TestActionRelationPrimitiveWordsExposeOnlySingleSteps(t *testing.T) {
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}}}
	action := actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 3}
	occurrence := actionrelations.Occurrence{Action: action}
	stateJSON, _ := state.CanonicalJSON()
	actionJSON, _ := action.CanonicalJSON()
	occurrenceJSON, _ := occurrence.CanonicalJSON()
	vm := &VM{stack: []Value{StringVal(string(stateJSON)), StringVal(string(actionJSON))}}
	if err := bARApplicable(vm); err != nil || !vm.pop().AsBool() {
		t.Fatalf("applicable err=%v", err)
	}
	vm.stack = []Value{StringVal(string(stateJSON)), StringVal(string(actionJSON)), BoolVal(true)}
	if err := bARApply(vm); err != nil {
		t.Fatal(err)
	}
	if value := vm.pop(); value.Kind() != VString {
		t.Fatalf("apply=%v", value)
	}
	vm.stack = []Value{StringVal(string(stateJSON)), StringVal(string(occurrenceJSON))}
	if err := bARActionFacts(vm); err != nil {
		t.Fatal(err)
	}
	value := vm.pop()
	if value.Kind() != VString {
		t.Fatalf("facts=%v", value)
	}
	if _, err := actionrelations.ParseLocalFacts([]byte(value.AsString())); err != nil {
		t.Fatalf("facts parse: %v value=%s", err, value.AsString())
	}
}

func TestActionRelationGuardWordsTraverseOneEdge(t *testing.T) {
	pattern := actionrelations.Pattern{Kinds: []string{"add", "add"}, Roles: []int{0, -1, 1, -1}}
	patternJSON, _ := pattern.CanonicalJSON()
	vm := &VM{stack: []Value{StringVal(string(patternJSON))}}
	if err := bARGuardRoot(vm); err != nil {
		t.Fatal(err)
	}
	root := vm.pop()
	if root.Kind() != VString {
		t.Fatalf("root=%v", root)
	}
	vm.stack = []Value{root, StringVal("read-write-disjoint"), BoolVal(true)}
	if err := bARGuardExtend(vm); err != nil {
		t.Fatal(err)
	}
	child := vm.pop()
	if child.Kind() != VString {
		t.Fatalf("child=%v", child)
	}
	guard, err := actionrelations.ParseGuard([]byte(child.AsString()))
	if err != nil || len(guard.Literals) != 1 {
		t.Fatalf("guard=%#v err=%v", guard, err)
	}
}
