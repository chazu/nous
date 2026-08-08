package dsl

import (
	"strings"
	"sync"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

func causalScopeTestVM() *VM {
	store := unit.NewStore()
	vocabulary := unit.New("Vocabulary")
	vocabulary.Set("isA", []string{"Anything"})
	store.Put(vocabulary)
	marker := unit.New("CausalVocabulary")
	marker.Set("isA", []string{"Vocabulary", "Anything"})
	marker.Set("dslExtension", "causal")
	store.Put(marker)
	return NewVM(store, agenda.New(), nil)
}

type testCausalScope struct{ calls int }

func (s *testCausalScope) Valid(name, slot string) bool { return name == "runtime" && slot == "task" }
func (s *testCausalScope) Begin(string, string) error   { s.calls++; return nil }
func (s *testCausalScope) Operation(string, string, ...string) (any, error) {
	s.calls++
	return true, nil
}
func (s *testCausalScope) End(string, string) error { s.calls++; return nil }

func TestCausalTaskScopeCannotBeReboundInheritedOrReused(t *testing.T) {
	vm := causalScopeTestVM()
	scope := &testCausalScope{}
	revoke, err := RegisterCausalTaskScope(vm, scope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterCausalTaskScope(vm, &testCausalScope{}); err == nil {
		t.Fatal("live causal task authority was rebound")
	}
	if got, err := vm.Execute(`"runtime" "task" causal-v2-task-valid?`); err != nil || !got.AsBool() {
		t.Fatalf("top-level registered VM failed: value=%v err=%v", got, err)
	}
	child := vm.childVM()
	if _, err := child.Execute(`"runtime" "task" causal-v2-task-valid?`); err == nil || !strings.Contains(err.Error(), "child-vm-unauthorized") {
		t.Fatalf("child VM inherited authority: %v", err)
	}
	revoke()
	revoke()
	if _, err := vm.Execute(`"runtime" "task" causal-v2-task-valid?`); err == nil || !strings.Contains(err.Error(), "child-vm-unauthorized") {
		t.Fatalf("revoked VM retained authority: %v", err)
	}
}

func TestCausalTaskScopeParallelVMIsolation(t *testing.T) {
	t.Parallel()
	var wait sync.WaitGroup
	errors := make(chan error, 32)
	start := make(chan struct{})
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			vm := causalScopeTestVM()
			scope := &testCausalScope{}
			revoke, err := RegisterCausalTaskScope(vm, scope)
			if err != nil {
				errors <- err
				return
			}
			defer revoke()
			<-start
			if _, err := vm.Execute(`"runtime" "task" causal-v2-begin-task`); err != nil {
				errors <- err
			}
		}()
	}
	close(start)
	wait.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}
