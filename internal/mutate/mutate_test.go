package mutate

import (
	"math/rand"
	"testing"

	"github.com/chazu/nous/internal/unit"
)

func TestMutateProducesOutput(t *testing.T) {
	store := unit.NewStore()
	rng := rand.New(rand.NewSource(42))
	m := New(rng, store)

	program := `"ArgU" @ "Op" isa? "ArgU" @ "defn" get-slot nil != and`

	// Run many mutations to exercise different paths
	mutated := 0
	for i := 0; i < 50; i++ {
		result, op := m.Mutate(program)
		if op != nil {
			mutated++
			if result == "" {
				t.Error("mutation produced empty result")
			}
			_ = result
		}
	}
	if mutated == 0 {
		t.Error("expected at least one successful mutation in 50 attempts")
	}
}

func TestMutateDiversity(t *testing.T) {
	store := unit.NewStore()
	store.Put(unit.New("Op"))
	store.Put(unit.New("BinaryOp"))

	u := unit.New("Op")
	u.Set("isA", []string{"Anything"})
	u.Set("specializations", []string{"BinaryOp"})
	store.Put(u)

	rng := rand.New(rand.NewSource(99))
	m := New(rng, store)

	program := `"ArgU" @ "Op" isa? 200 > and`

	kinds := make(map[string]bool)
	for i := 0; i < 100; i++ {
		_, op := m.Mutate(program)
		if op != nil {
			kinds[op.Kind] = true
		}
	}

	// We should see at least 3 different mutation types
	if len(kinds) < 3 {
		t.Errorf("expected at least 3 mutation kinds, got %v", kinds)
	}
}

func TestValidate(t *testing.T) {
	store := unit.NewStore()
	store.Put(unit.New("Anything"))

	// Valid program
	if !Validate(`1 2 +`, store) {
		t.Error("1 2 + should be valid")
	}

	// Invalid program (unknown word)
	if Validate(`1 2 blarghhh`, store) {
		t.Error("unknown word should be invalid")
	}

	// Abort is valid
	if !Validate(`abort`, store) {
		t.Error("abort should be valid")
	}

	// Unbalanced if is invalid
	if Validate(`true if 1`, store) {
		t.Error("unbalanced if should be invalid")
	}
}

func TestWidenNarrow(t *testing.T) {
	store := unit.NewStore()
	rng := rand.New(rand.NewSource(42))
	m := New(rng, store)

	program := `100 200 >`

	widens := 0
	narrows := 0
	for i := 0; i < 100; i++ {
		_, op := m.Mutate(program)
		if op != nil {
			switch op.Kind {
			case "widen":
				widens++
			case "narrow":
				narrows++
			}
		}
	}
	if widens == 0 {
		t.Error("expected at least one widen mutation")
	}
	if narrows == 0 {
		t.Error("expected at least one narrow mutation")
	}
}
