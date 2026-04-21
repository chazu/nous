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

// Phase 7.2: run-generator iterates a unit's generator to produce N values.
func TestRunGenerator(t *testing.T) {
	vm := testVM(t)

	// Counting generator: start at 0, step = +1.
	counter := unit.New("Counter")
	counter.Set("isA", []string{"MathObj", "Anything"})
	counter.Set("generator", map[string]any{
		"initial": []any{0},
		"step":    "1 +",
	})
	vm.Store.Put(counter)

	v, err := vm.Execute(`"Counter" 5 run-generator`)
	if err != nil {
		t.Fatal(err)
	}
	list := v.AsList()
	if len(list) != 5 {
		t.Fatalf("expected 5 values, got %d: %v", len(list), list)
	}
	for i, el := range list {
		if el.AsInt() != i {
			t.Errorf("list[%d]: got %d, want %d", i, el.AsInt(), i)
		}
	}

	// Generator with 2 initial values.
	fib := unit.New("FibLike")
	fib.Set("isA", []string{"MathObj", "Anything"})
	fib.Set("generator", map[string]any{
		"initial": []any{1, 1},
		"step":    "2 *", // not actual Fib; just doubles each step
	})
	vm.Store.Put(fib)
	v, _ = vm.Execute(`"FibLike" 4 run-generator`)
	list = v.AsList()
	if len(list) != 4 || list[0].AsInt() != 1 || list[1].AsInt() != 1 ||
		list[2].AsInt() != 2 || list[3].AsInt() != 4 {
		t.Errorf("doubling generator: got %v, want [1 1 2 4]", list)
	}

	// Unit with no generator slot -> empty list.
	bare := unit.New("Bare")
	bare.Set("isA", []string{"Anything"})
	vm.Store.Put(bare)
	v, _ = vm.Execute(`"Bare" 3 run-generator`)
	if len(v.AsList()) != 0 {
		t.Errorf("no generator: expected empty list, got %v", v.AsList())
	}
}

func TestOSetOps(t *testing.T) {
	vm := testVM(t)

	// Lists are constructed via `list-of` — push N elements, then N, then call.
	cases := []struct {
		name string
		prog string
		want []int
	}{
		// Order preservation: a then b's novelty in b's order
		{"union-preserves-order", `3 1 2 3 list-of  4 2 5 3 list-of  oset-union`, []int{3, 1, 2, 4, 5}},
		// Intersect preserves a's order
		{"intersect-preserves-a-order", `3 1 2 4 4 list-of  4 2 2 list-of  oset-intersect`, []int{2, 4}},
		// Insert: present → no-op
		{"insert-present-noop", `3 1 2 3 list-of  1 oset-insert`, []int{3, 1, 2}},
		// Insert: absent → append at end
		{"insert-absent-appends", `3 1 2 3 list-of  7 oset-insert`, []int{3, 1, 2, 7}},
		// Delete: preserves surrounding order
		{"delete-preserves-order", `3 1 2 4 4 list-of  1 oset-delete`, []int{3, 2, 4}},
	}
	for _, tc := range cases {
		v, err := vm.Execute(tc.prog)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		got := v.AsList()
		if len(got) != len(tc.want) {
			t.Errorf("%s: want %v got %v", tc.name, tc.want, got)
			continue
		}
		for i, w := range tc.want {
			if got[i].AsInt() != w {
				t.Errorf("%s: at %d want %d got %d (full: %v)", tc.name, i, w, got[i].AsInt(), got)
				break
			}
		}
	}

	// Divergence: oset-equal? is order-sensitive, set-equal? is not
	v, err := vm.Execute(`1 2 2 list-of  2 1 2 list-of  oset-equal?`)
	if err != nil || v.AsBool() {
		t.Errorf("oset-equal? [1 2] [2 1]: want false, got %v (err=%v)", v, err)
	}
	v, err = vm.Execute(`1 2 2 list-of  2 1 2 list-of  set-equal?`)
	if err != nil || !v.AsBool() {
		t.Errorf("set-equal? [1 2] [2 1]: want true, got %v (err=%v)", v, err)
	}
	v, err = vm.Execute(`1 2 3 3 list-of  1 2 3 3 list-of  oset-equal?`)
	if err != nil || !v.AsBool() {
		t.Errorf("oset-equal? identical: want true, got %v (err=%v)", v, err)
	}
}

func TestButLast(t *testing.T) {
	vm := testVM(t)

	cases := []struct {
		name string
		prog string
		want []int
	}{
		{"basic", `3 1 2 3 list-of but-last`, []int{3, 1}},
		{"two-element", `1 2 2 list-of but-last`, []int{1}},
		{"single-element", `1 1 list-of but-last`, []int{}},
	}
	for _, tc := range cases {
		v, err := vm.Execute(tc.prog)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		got := v.AsList()
		if len(got) != len(tc.want) {
			t.Errorf("%s: len want %d got %d (%v)", tc.name, len(tc.want), len(got), got)
			continue
		}
		for i, w := range tc.want {
			if got[i].AsInt() != w {
				t.Errorf("%s: at %d want %d got %d", tc.name, i, w, got[i].AsInt())
			}
		}
	}

	// Empty input → empty output, no error.
	v, err := vm.Execute(`0 list-of but-last`)
	if err != nil {
		t.Errorf("empty: %v", err)
	}
	if len(v.AsList()) != 0 {
		t.Errorf("empty: want [] got %v", v.AsList())
	}
}

func TestMutateMultiplicities(t *testing.T) {
	// Every output element must come from the input alphabet.
	t.Run("element_preservation", func(t *testing.T) {
		vm := testVM(t)
		inputAlphabet := map[int]bool{1: true, 2: true, 3: true, 5: true}
		for i := 0; i < 50; i++ {
			v, err := vm.Execute(`1 2 3 5 4 list-of mutate-multiplicities`)
			if err != nil {
				t.Fatalf("exec: %v", err)
			}
			for _, e := range v.AsList() {
				if !inputAlphabet[e.AsInt()] {
					t.Errorf("foreign element %d in output %v", e.AsInt(), v.AsList())
				}
			}
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		vm := testVM(t)
		v, err := vm.Execute(`0 list-of mutate-multiplicities`)
		if err != nil {
			t.Fatalf("exec: %v", err)
		}
		if len(v.AsList()) != 0 {
			t.Errorf("empty input should produce empty output, got %v", v.AsList())
		}
	})

	t.Run("distribution_shows_drops_and_duplicates", func(t *testing.T) {
		vm := testVM(t)
		sawLonger := false
		sawShorter := false
		for i := 0; i < 200; i++ {
			v, err := vm.Execute(`1 2 3 3 list-of mutate-multiplicities`)
			if err != nil {
				t.Fatalf("exec: %v", err)
			}
			n := len(v.AsList())
			if n > 3 {
				sawLonger = true
			}
			if n < 3 {
				sawShorter = true
			}
			if sawLonger && sawShorter {
				return
			}
		}
		if !sawLonger {
			t.Error("200 runs never produced a longer-than-3 output — duplicate branch unreachable?")
		}
		if !sawShorter {
			t.Error("200 runs never produced a shorter-than-3 output — drop branch unreachable?")
		}
	})
}
