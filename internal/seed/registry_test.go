package seed

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
)

func TestBuildGraphVocabularyIsIndependentAndExecutable(t *testing.T) {
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "buildgraphs"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"BuildGraph", "MergeBuildGraphs", "H-RunGraphOps"} {
		if !store.Has(name) {
			t.Fatalf("buildgraphs vocabulary missing %s", name)
		}
	}
	for _, mathOnly := range []string{"MathConcept", "Set", "H1"} {
		if store.Has(mathOnly) {
			t.Fatalf("buildgraphs unexpectedly loaded math-only unit %s", mathOnly)
		}
	}

	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = 80
	eng.MutConfig.Enabled = false
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	foundResult := false
	for _, name := range store.All() {
		if strings.HasPrefix(name, "MergeBuildGraphs-on-") {
			foundResult = true
			if len(store.Get(name).GetStrings("creditors")) == 0 {
				t.Fatalf("generated graph %s has no provenance", name)
			}
		}
	}
	if !foundResult {
		t.Fatal("build-graph heuristic produced no MergeBuildGraphs result")
	}
	if len(store.Get("MergeBuildGraphs").Get("applics").([]map[string]any)) == 0 {
		t.Fatal("MergeBuildGraphs recorded no applications")
	}
}

func TestAvailableDiscoversVocabularyDirectories(t *testing.T) {
	DomainsDir = "../../domains"
	got := Available()
	for _, want := range []string{"buildgraphs", "math", "protocols", "rewrite"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Available() = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "common") {
		t.Fatalf("Available() = %q, common is not a standalone vocabulary", got)
	}
}

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
