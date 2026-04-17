package seed

import (
	"testing"

	"github.com/chazu/nous/internal/unit"
)

func TestPhase6SlotsLoaded(t *testing.T) {
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}

	// Phase 6 slot units must all be present.
	for _, name := range []string{
		"Interestingness", "Rarity", "IntExamples", "IsAInt",
		"WhyInt", "MoreInteresting", "LessInteresting",
	} {
		if !store.Has(name) {
			t.Errorf("missing Phase 6 slot unit: %s", name)
		}
	}

	// IntExamples is a sub-slot of Examples.
	intEx := store.Get("IntExamples")
	if supers := intEx.GetStrings("superSlots"); len(supers) != 1 || supers[0] != "Examples" {
		t.Errorf("IntExamples.superSlots: got %v, want [Examples]", supers)
	}

	// IntExamples <-> IsAInt inverse wiring, both directions. The inverse
	// slot value gets list-ified during load by computeInverseIndex
	// (the self-inversing "inverse" slot cascades), so read via GetStrings.
	firstInv := func(u *unit.Unit) string {
		if ss := u.GetStrings("inverse"); len(ss) > 0 {
			return ss[0]
		}
		return u.GetString("inverse")
	}
	if got := firstInv(intEx); got != "IsAInt" {
		t.Errorf("IntExamples.inverse: got %q, want IsAInt", got)
	}
	if got := firstInv(store.Get("IsAInt")); got != "IntExamples" {
		t.Errorf("IsAInt.inverse: got %q, want IntExamples", got)
	}
	if got := firstInv(store.Get("MoreInteresting")); got != "LessInteresting" {
		t.Errorf("MoreInteresting.inverse: got %q, want LessInteresting", got)
	}
	if got := firstInv(store.Get("LessInteresting")); got != "MoreInteresting" {
		t.Errorf("LessInteresting.inverse: got %q, want MoreInteresting", got)
	}

	// Inverse registration must actually work at the Store level — setting
	// intExamples on one unit should mirror to isAInt on the target.
	cat := unit.New("TestCategory")
	cat.Set("isA", []string{"Anything"})
	store.Put(cat)
	inst := unit.New("TestInstance")
	inst.Set("isA", []string{"Anything"})
	store.Put(inst)
	store.SetSlot("TestCategory", "intExamples", []string{"TestInstance"})
	back := store.Get("TestInstance").GetStrings("isAInt")
	if len(back) != 1 || back[0] != "TestCategory" {
		t.Errorf("IntExamples inverse maintenance failed: TestInstance.isAInt = %v", back)
	}

	// Examples.subSlots should include IntExamples (already declared in slots.cue).
	ex := store.Get("Examples")
	subs := ex.GetStrings("subSlots")
	found := false
	for _, s := range subs {
		if s == "IntExamples" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Examples.subSlots: got %v, want to contain IntExamples", subs)
	}
}
