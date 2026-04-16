package seed

import (
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

// TestCriterialSlotsBuiltin verifies that criterial-slots returns real slot
// names for a unit after domain load. Slot definition units are named
// "Domain", "Range", etc. (capitalized), but slot keys on units are lowercase
// ("domain", "range"). The builtin must bridge this case gap.
func TestCriterialSlotsBuiltin(t *testing.T) {
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}

	vm := dsl.NewVM(store, agenda.New(), nil)
	v, err := vm.Execute(`"SetIntersect" criterial-slots`)
	if err != nil {
		t.Fatal(err)
	}
	got := v.AsList()
	if len(got) == 0 {
		t.Fatalf("criterial-slots on SetIntersect returned empty; expected domain/range/defn/arity")
	}

	have := map[string]bool{}
	for _, s := range got {
		have[s.AsString()] = true
	}
	for _, want := range []string{"domain", "range", "defn"} {
		if !have[want] {
			t.Errorf("criterial-slots missing %q; got %v", want, got)
		}
	}
}
