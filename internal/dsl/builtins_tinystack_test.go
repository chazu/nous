package dsl

import (
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

func tinyStackTestStore() *unit.Store {
	store := protocolTestStore("")
	stackMarker := unit.New("StackVocabulary")
	stackMarker.Set("isA", []string{"Vocabulary", "Anything"})
	stackMarker.Set("dslExtension", "stack")
	store.Put(stackMarker)
	return store
}

func TestTinyStackWordsAreStrictAndScoped(t *testing.T) {
	base := NewVM(protocolTestStore(""), agenda.New(), nil)
	if _, err := base.Execute(`1 1 list-of stack-valid?`); err == nil {
		t.Fatal("unselected VM exposed stack word")
	}
	if _, err := base.Execute(`"anything" synth-candidate-count`); err == nil {
		t.Fatal("unselected VM exposed program-synthesis word")
	}
	vm := NewVM(tinyStackTestStore(), agenda.New(), nil)
	value, err := vm.Execute(`2 3 2 list-of "over" stack-exec-op "add" stack-exec-op`)
	got, ok := strictIntList(value)
	if err != nil || !ok || len(got) != 2 || got[0] != 2 || got[1] != 5 {
		t.Fatalf("over add = (%v,%v)", value, err)
	}
	value, err = vm.Execute(`1 2 3 4 5 5 list-of stack-input-valid?`)
	if err != nil || value.Kind() != VBool || value.AsBool() {
		t.Fatalf("depth-five input validity = (%v,%v), want false", value, err)
	}
	value, err = vm.Execute(`1 2 3 4 5 5 list-of stack-valid?`)
	if err != nil || value.Kind() != VBool || !value.AsBool() {
		t.Fatalf("depth-five runtime stack validity = (%v,%v), want true", value, err)
	}
	for _, program := range []string{
		`1.5 1 list-of stack-valid?`,
		`"1" 1 list-of stack-valid?`,
		`0 list-of "dup" stack-exec-op`,
		`1 1 list-of "unknown" stack-exec-op`,
	} {
		value, err := vm.Execute(program)
		if err != nil {
			t.Fatal(err)
		}
		if value.Kind() == VBool {
			if value.AsBool() {
				t.Fatalf("invalid %q classified true", program)
			}
		} else if !value.IsNil() {
			t.Fatalf("invalid %q = %v, want nil", program, value)
		}
	}
}
