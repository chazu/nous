package dsl

import (
	"bytes"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
)

func TestSetOps(t *testing.T) {
	vm := testVM(t)

	// Union
	v, err := vm.Execute(`3 iota 5 iota set-union`)
	if err != nil {
		t.Fatalf("set-union: %v", err)
	}
	if len(v.AsList()) != 5 {
		t.Errorf("set-union: expected 5 elements, got %d: %v", len(v.AsList()), v)
	}

	// Intersect
	v, err = vm.Execute(`3 iota 5 iota set-intersect`)
	if err != nil {
		t.Fatalf("set-intersect: %v", err)
	}
	if len(v.AsList()) != 3 {
		t.Errorf("set-intersect: expected 3 elements, got %v", v)
	}

	// Diff
	v, err = vm.Execute(`5 iota 3 iota set-diff`)
	if err != nil {
		t.Fatalf("set-diff: %v", err)
	}
	if len(v.AsList()) != 2 {
		t.Errorf("set-diff: expected 2 elements, got %v", v)
	}
}

func TestNumberPredicates(t *testing.T) {
	vm := testVM(t)

	tests := []struct {
		prog string
		want bool
	}{
		{"7 prime?", true},
		{"4 prime?", false},
		{"1 prime?", false},
		{"6 even?", true},
		{"7 odd?", true},
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

func TestDivisors(t *testing.T) {
	vm := testVM(t)
	v, err := vm.Execute(`12 divisors`)
	if err != nil {
		t.Fatal(err)
	}
	list := v.AsList()
	if len(list) != 6 {
		t.Errorf("divisors of 12: expected 6, got %d: %v", len(list), v)
	}
}

func TestApplyOp(t *testing.T) {
	s := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(s, ag)
	vm.Out = &bytes.Buffer{}

	// Create an operation unit with a defn
	op := unit.New("TestIntersect")
	op.Set("defn", "set-intersect")
	op.Set("isA", []string{"BinaryOp"})
	s.Put(op)
	s.Put(unit.New("BinaryOp"))

	// Push two sets and the op name, then apply
	v, err := vm.Execute(`3 iota 5 iota "TestIntersect" apply-op`)
	if err != nil {
		t.Fatalf("apply-op: %v", err)
	}
	if v.IsNil() {
		t.Fatal("apply-op returned nil")
	}
	if len(v.AsList()) != 3 {
		t.Errorf("apply-op: expected 3 elements, got %v", v)
	}
}

func TestApplyOpWithStoreData(t *testing.T) {
	s := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(s, ag)
	vm.Out = &bytes.Buffer{}

	// Simulate what H-RunOnExamples does
	s.Put(unit.New("BinaryOp"))
	op := unit.New("SetIntersect")
	op.Set("defn", "set-intersect")
	op.Set("isA", []string{"BinaryOp"})
	s.Put(op)

	primes := unit.New("SetOfPrimes")
	primes.Set("data", []int{2, 3, 5, 7, 11})
	s.Put(primes)

	evens := unit.New("SetOfEvens")
	evens.Set("data", []int{2, 4, 6, 8, 10})
	s.Put(evens)

	// Get data from units and apply
	v, err := vm.Execute(`
		"SetOfPrimes" "data" get-slot
		"SetOfEvens" "data" get-slot
		"SetIntersect" apply-op
	`)
	if err != nil {
		t.Fatalf("apply-op with store data: %v", err)
	}
	// Intersection of primes and evens should be {2}
	list := v.AsList()
	if len(list) != 1 || list[0].AsInt() != 2 {
		t.Errorf("expected [2], got %v", v)
	}
}
