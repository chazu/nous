package dsl

import (
	"bytes"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

func testVM(t *testing.T) *VM {
	t.Helper()
	s := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(s, ag)
	vm.Out = &bytes.Buffer{}
	return vm
}

func TestArithmetic(t *testing.T) {
	vm := testVM(t)
	tests := []struct {
		prog string
		want int
	}{
		{"3 4 +", 7},
		{"10 3 -", 7},
		{"6 7 *", 42},
		{"20 4 /", 5},
		{"17 5 mod", 2},
	}
	for _, tt := range tests {
		v, err := vm.Execute(tt.prog)
		if err != nil {
			t.Errorf("%s: %v", tt.prog, err)
			continue
		}
		if v.AsInt() != tt.want {
			t.Errorf("%s: got %d, want %d", tt.prog, v.AsInt(), tt.want)
		}
	}
}

func TestComparison(t *testing.T) {
	vm := testVM(t)
	tests := []struct {
		prog string
		want bool
	}{
		{"3 3 =", true},
		{"3 4 =", false},
		{"3 4 <", true},
		{"4 3 >", true},
		{"3 3 <=", true},
		{"3 3 >=", true},
	}
	for _, tt := range tests {
		v, err := vm.Execute(tt.prog)
		if err != nil {
			t.Errorf("%s: %v", tt.prog, err)
			continue
		}
		if v.AsBool() != tt.want {
			t.Errorf("%s: got %v, want %v", tt.prog, v.AsBool(), tt.want)
		}
	}
}

func TestVariables(t *testing.T) {
	vm := testVM(t)
	v, err := vm.Execute(`42 "x" ! "x" @`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsInt() != 42 {
		t.Errorf("got %d, want 42", v.AsInt())
	}
}

func TestIfThenEnd(t *testing.T) {
	vm := testVM(t)

	// Forth-style: <cond> if <true-body> then
	v, err := vm.Execute(`true if 42 then`)
	if err != nil {
		t.Fatalf("true branch: %v", err)
	}
	if v.AsInt() != 42 {
		t.Errorf("true branch: got %v, want 42", v)
	}

	// False branch - should not push
	v, err = vm.Execute(`false if 42 then`)
	if err != nil {
		t.Fatalf("false branch: %v", err)
	}
	if !v.IsNil() {
		t.Errorf("false branch: got %v, want nil", v)
	}

	// With else
	v, err = vm.Execute(`false if 1 else 2 then`)
	if err != nil {
		t.Fatalf("else branch: %v", err)
	}
	if v.AsInt() != 2 {
		t.Errorf("else branch: got %v, want 2", v)
	}

	// Multiline if (the pattern used in heuristics)
	v, err = vm.Execute(`
		1 0 >
		if
			99
		then
	`)
	if err != nil {
		t.Fatalf("multiline if: %v", err)
	}
	if v.AsInt() != 99 {
		t.Errorf("multiline if: got %v, want 99", v)
	}

	// Nested if
	v, err = vm.Execute(`true if true if 7 then then`)
	if err != nil {
		t.Fatalf("nested if: %v", err)
	}
	if v.AsInt() != 7 {
		t.Errorf("nested if: got %v, want 7", v)
	}
}

func TestEach(t *testing.T) {
	vm := testVM(t)

	// Create a unit with a list
	u := unit.New("TestUnit")
	u.Set("items", []string{"a", "b", "c"})
	vm.Store.Put(u)

	v, err := vm.Execute(`"TestUnit" "items" get-slot each it print end true`)
	if err != nil {
		t.Fatalf("each: %v", err)
	}
	if !v.Truthy() {
		t.Error("expected true after each loop")
	}
}

func TestStoreOps(t *testing.T) {
	vm := testVM(t)

	u := unit.New("Foo")
	u.Set("worth", 500)
	u.Set("isA", []string{"Anything"})
	vm.Store.Put(u)

	// get-slot
	v, err := vm.Execute(`"Foo" "worth" get-slot`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsInt() != 500 {
		t.Errorf("get-slot: got %d, want 500", v.AsInt())
	}

	// isa?
	v, err = vm.Execute(`"Foo" "Anything" isa?`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.AsBool() {
		t.Error("isa? should be true")
	}

	// create-unit
	v, err = vm.Execute(`"Bar" "Anything" create-unit`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsString() != "Bar" {
		t.Errorf("create-unit: got %v", v)
	}
	if !vm.Store.Has("Bar") {
		t.Error("Bar should exist")
	}
}

func TestAddTask(t *testing.T) {
	vm := testVM(t)

	_, err := vm.Execute(`500 "MyUnit" "examples" "test reason" add-task`)
	if err != nil {
		t.Fatal(err)
	}
	if vm.Ag.Len() != 1 {
		t.Errorf("expected 1 task, got %d", vm.Ag.Len())
	}
	task := vm.Ag.Pop()
	if task.Priority != 500 || task.UnitName != "MyUnit" || task.SlotName != "examples" {
		t.Errorf("unexpected task: %+v", task)
	}
}

func TestComments(t *testing.T) {
	vm := testVM(t)
	v, err := vm.Execute(`
		# This is a comment
		42
		# Another comment
	`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsInt() != 42 {
		t.Errorf("got %d, want 42", v.AsInt())
	}
}

func TestAbort(t *testing.T) {
	vm := testVM(t)
	_, err := vm.Execute(`abort`)
	if !IsAbort(err) {
		t.Errorf("expected AbortError, got %v", err)
	}
}
