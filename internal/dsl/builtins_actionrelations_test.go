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
	occurrenceJSON, _ := occurrence.CanonicalJSON()
	vm := &VM{Store: actionRelationTestStore(), stack: []Value{StringVal(string(stateJSON)), StringVal(string(occurrenceJSON)), StringVal("AR.Applicable")}}
	if err := bARApplicable(vm); err != nil {
		t.Fatalf("applicable err=%v", err)
	}
	applicabilityName := vm.pop().AsString()
	if !vm.Store.Get(applicabilityName).GetBool("applicable") {
		t.Fatal("expected applicable row")
	}
	vm.stack = []Value{StringVal(string(stateJSON)), StringVal(string(occurrenceJSON)), StringVal(applicabilityName), StringVal("AR.Transition"), StringVal("AR.State.Next")}
	if err := bARApply(vm); err != nil {
		t.Fatal(err)
	}
	if value := vm.pop(); value.Kind() != VList || vm.Store.Get(value.AsList()[1].AsString()).GetString("state") == "" {
		t.Fatalf("apply=%v", value)
	}
	vm.stack = []Value{StringVal(string(stateJSON)), StringVal(string(occurrenceJSON)), StringVal("AR.Facts")}
	if err := bARActionFacts(vm); err != nil {
		t.Fatal(err)
	}
	value := vm.pop()
	if value.Kind() != VString || vm.Store.Get(value.AsString()) == nil {
		t.Fatalf("facts=%v", value)
	}
	if _, err := actionrelations.ParseLocalFacts([]byte(vm.Store.Get(value.AsString()).GetString("facts"))); err != nil {
		t.Fatalf("facts parse: %v value=%s", err, value.AsString())
	}
}

func TestActionRelationGuardWordsTraverseOneEdge(t *testing.T) {
	pattern := actionrelations.Pattern{Kinds: []string{"add", "add"}, Roles: []int{0, -1, 1, -1}}
	patternJSON, _ := pattern.CanonicalJSON()
	vm := &VM{Store: actionRelationTestStore(), stack: []Value{StringVal(string(patternJSON)), StringVal("AR.Candidate.Root")}}
	if err := bARGuardRoot(vm); err != nil {
		t.Fatal(err)
	}
	rootResult := vm.pop()
	if rootResult.Kind() != VList || len(rootResult.AsList()) != 2 {
		t.Fatalf("root result=%v", rootResult)
	}
	root := rootResult.AsList()[0]
	rootName := rootResult.AsList()[1]
	if root.Kind() != VString || rootName.Kind() != VString {
		t.Fatalf("root=%v", root)
	}
	vm.stack = []Value{root, StringVal("read-write-disjoint"), BoolVal(true), IntVal(0), StringVal("AR.Edge.0")}
	if err := bARGuardExtend(vm); err != nil {
		t.Fatal(err)
	}
	childResult := vm.pop()
	if childResult.Kind() != VList || len(childResult.AsList()) != 2 {
		t.Fatalf("child result=%v", childResult)
	}
	child := childResult.AsList()[0]
	guard, err := actionrelations.ParseGuard([]byte(child.AsString()))
	if err != nil || len(guard.Literals) != 1 {
		t.Fatalf("guard=%#v err=%v", guard, err)
	}
	vm.stack = []Value{StringVal(string(patternJSON)), child, rootName, IntVal(1), StringVal("AR.Candidate.1")}
	if err := bARCandidateAllocate(vm); err != nil || vm.pop().Kind() != VString {
		t.Fatalf("candidate allocate: %v", err)
	}
}

func TestActionRelationStoreUsesContentSuffixOnOccupiedName(t *testing.T) {
	pattern := actionrelations.Pattern{Kinds: []string{"add", "add"}, Roles: []int{0, -1, 1, -1}}
	patternJSON, _ := pattern.CanonicalJSON()
	store := actionRelationTestStore()
	occupied := unit.New("AR.Candidate.Root")
	occupied.Set("owner", "user")
	store.Put(occupied)
	vm := &VM{Store: store, stack: []Value{StringVal(string(patternJSON)), StringVal(occupied.Name)}}
	if err := bARGuardRoot(vm); err != nil {
		t.Fatal(err)
	}
	result := vm.pop()
	actual := result.AsList()[1].AsString()
	if actual == occupied.Name || store.Get(occupied.Name).GetString("owner") != "user" || store.Get(actual).GetString("objectDigest") == "" {
		t.Fatalf("occupied=%q actual=%q", occupied.Name, actual)
	}
}
