package seed

import (
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

func TestActionRelationDomainLoadsScopedVocabularyPack(t *testing.T) {
	DomainsDir = "../../domains"
	store := unit.NewStore()
	if err := LoadDomain(store, "actionrelations"); err != nil {
		t.Fatal(err)
	}
	marker := store.Get("GuardedActionRelationVocabulary")
	if marker == nil || marker.GetString("dslExtension") != "actionrelations" {
		t.Fatalf("action-relation vocabulary marker = %#v", marker)
	}
	if !store.IsA("ActionGuardCandidate", "Anything") || !store.IsA("ValidFiniteActionState", "Op") {
		t.Fatal("action-relation category/operation ontology was not loaded")
	}
	vm := dsl.NewVM(store, agenda.New(), nil)
	if err := vm.InitError(); err != nil {
		t.Fatal(err)
	}
}
