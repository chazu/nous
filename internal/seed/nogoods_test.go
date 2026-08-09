package seed

import (
	"slices"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

func TestNogoodDomainLoadsBoundedVocabularyPack(t *testing.T) {
	DomainsDir = "../../domains"
	store := unit.NewStore()
	if err := LoadDomain(store, "nogoods"); err != nil {
		t.Fatal(err)
	}
	marker := store.Get("ConstraintNogoodVocabulary")
	if marker == nil || marker.GetString("dslExtension") != "nogoods" {
		t.Fatalf("nogood vocabulary marker = %#v", marker)
	}
	if !store.IsA("NogoodCandidate", "Anything") || !store.IsA("ValidNogoodProblem", "Op") {
		t.Fatal("nogood category/operation ontology was not loaded")
	}
	if got := store.Examples("Heuristic"); !slices.Equal(got, []string{"Heuristic", "NG-H-ConsiderPrune"}) {
		t.Fatalf("bridge heuristic set = %v", got)
	}
	heuristic := store.Get("NG-H-ConsiderPrune")
	if heuristic.GetString("ifWorkingOnTask") == "" || heuristic.GetString("thenCompute") == "" {
		t.Fatal("bridge heuristic programs are empty")
	}
	vm := dsl.NewVM(store, agenda.New(), nil)
	if err := vm.InitError(); err != nil {
		t.Fatal(err)
	}
}
