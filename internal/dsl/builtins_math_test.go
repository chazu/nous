package dsl

import (
	"bytes"
	"fmt"
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
	vm := NewVM(s, ag, nil)
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
	vm := NewVM(s, ag, nil)
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

func TestRarityTracking(t *testing.T) {
	vm := testVM(t)
	// Create a simple predicate unit that returns whether input > 0.
	pred := unit.New("IsPositive")
	pred.Set("isA", []string{"UnaryPred", "Pred", "Anything"})
	pred.Set("defn", `0 >`)
	vm.Store.Put(pred)

	// Apply it to several inputs via direct stack push. apply-op pops
	// a single arg for unary ops (isA UnaryPred or checked via BinaryOp/BinaryPred branch).
	// UnaryPred isn't in the binary branch of bApplyOp, so it takes one arg.
	for _, n := range []int{5, 3, -1, 0, -7, 2} {
		_, err := vm.Execute(fmt.Sprintf(`%d "IsPositive" apply-op drop`, n))
		if err != nil {
			t.Fatal(err)
		}
	}

	// Expect 3 trues (5, 3, 2) and 3 falses (-1, 0, -7)
	rarity, ok := vm.Store.Get("IsPositive").Get("rarity").([]any)
	if !ok {
		t.Fatalf("rarity not populated: got %T", vm.Store.Get("IsPositive").Get("rarity"))
	}
	if len(rarity) != 3 {
		t.Fatalf("rarity len: got %d, want 3", len(rarity))
	}
	freq, _ := rarity[0].(float64)
	numT, _ := rarity[1].(int)
	numF, _ := rarity[2].(int)
	if numT != 3 || numF != 3 {
		t.Errorf("rarity counts: got T=%d F=%d, want T=3 F=3", numT, numF)
	}
	if freq != 0.5 {
		t.Errorf("rarity freq: got %v, want 0.5", freq)
	}
}

// Phase 5.6 C.1: type predicates (is-int?, is-list?) work as DSL builtins
// and can be used as the defn of type units so apply-pred on Set/Number
// returns a type check result.
func TestTypePredicates(t *testing.T) {
	vm := testVM(t)

	// Builtins directly.
	cases := []struct {
		prog string
		want bool
	}{
		{"5 is-int?", true},
		{"5 is-list?", false},
		{"3 iota is-list?", true},
		{"3 iota is-int?", false},
		{`"hello" is-string?`, true},
		{"5 is-string?", false},
	}
	for _, c := range cases {
		v, err := vm.Execute(c.prog)
		if err != nil {
			t.Errorf("%q: %v", c.prog, err)
			continue
		}
		if v.AsBool() != c.want {
			t.Errorf("%q: got %v, want %v", c.prog, v.AsBool(), c.want)
		}
	}

	// Simulated type unit with defn = "is-list?"; apply-pred on it should
	// return true for lists, false for ints. This is the Phase 5.6 pattern:
	// type units carry their defn as an executable predicate.
	setType := unit.New("Set")
	setType.Set("isA", []string{"Structure", "Anything"})
	setType.Set("defn", "is-list?")
	vm.Store.Put(setType)

	v, err := vm.Execute(`3 iota "Set" apply-pred`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.AsBool() {
		t.Error("apply-pred Set on a list: expected true")
	}
	v, err = vm.Execute(`42 "Set" apply-pred`)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsBool() {
		t.Error("apply-pred Set on an int: expected false")
	}
}
