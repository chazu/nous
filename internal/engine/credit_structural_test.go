package engine

import (
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/credit"
	"github.com/chazu/nous/internal/unit"
)

func TestWorthGrowthDerivesStructuralFeatureCredit(t *testing.T) {
	store := unit.NewStore()
	store.Put(unit.New("Anything"))
	store.Put(unit.New(credit.Category))
	for _, name := range []string{"H-Search", "Edit-A", "Edit-B"} {
		u := unit.New(name)
		u.SetWorth(500)
		store.Put(u)
	}
	featureA := unit.New("Feature-A")
	featureA.SetWorth(500)
	featureA.Set("creditFeatureKey", "feature/a")
	store.Put(featureA)
	featureB := unit.New("Feature-B")
	featureB.SetWorth(500)
	featureB.Set("creditFeatureKey", "feature/b")
	store.Put(featureB)
	relation := unit.New("Relation-Reference")
	relation.SetWorth(500)
	relation.Set("creditRelationKey", "relation/reference")
	store.Put(relation)
	for name, feature := range map[string]string{"Edit-A": "Feature-A", "Edit-B": "Feature-B"} {
		component := store.Get(name)
		component.Set("creditFeatureSubject", feature)
		component.Set("creditFeatureKey", store.Get(feature).GetString("creditFeatureKey"))
		component.Set("creditRelationSubject", relation.Name)
		component.Set("creditRelationKey", relation.GetString("creditRelationKey"))
	}
	program := unit.New("Selected-Program")
	program.SetWorth(800)
	program.Set("creationWorth", 500)
	program.Set("lastRewardedWorth", 500)
	program.Set("components", []string{"Edit-A", "Edit-B"})
	program.Set("creditors", []string{"H-Search", "Edit-A", "Edit-B"})
	program.Set("creditRoles", []string{"synthesis", "step-1", "step-2"})
	program.Set("creditContext", "test/structural/v1")
	program.Set("creditDecision", "concrete-decision")
	program.Set("synthesisMethod", "ordered/v1")
	store.Put(program)

	eng := New(store, agenda.New())
	eng.TaskNum = 7
	eng.rewardForWorthGrowth()
	decision, err := credit.StructuralDecisionKey("ordered/v1", []string{"feature/a", "feature/b"})
	if err != nil {
		t.Fatal(err)
	}
	checks := map[credit.Tuple]int{
		credit.DecisionTuple("test/structural/v1", decision):                      300,
		{Context: "test/structural/v1", Subject: "Feature-A", Role: "component"}:  150,
		{Context: "test/structural/v1", Subject: "Feature-A", Role: "step-1"}:     150,
		{Context: "test/structural/v1", Subject: "Feature-B", Role: "component"}:  150,
		{Context: "test/structural/v1", Subject: "Feature-B", Role: "step-2"}:     150,
		{Context: "test/structural/v1", Subject: relation.Name, Role: "relation"}: 300,
	}
	for tuple, want := range checks {
		if got := credit.RewardTotal(store, tuple); got != want {
			t.Fatalf("credit %#v = %d, want %d", tuple, got, want)
		}
	}
	if featureA.Worth() != 650 || featureB.Worth() != 650 {
		t.Fatalf("feature worths = %d,%d", featureA.Worth(), featureB.Worth())
	}
}

func TestMalformedStructuralDeclarationIsAtomic(t *testing.T) {
	store := unit.NewStore()
	store.Put(unit.New("Anything"))
	store.Put(unit.New(credit.Category))
	for _, name := range []string{"H", "Edit", "Feature", "Relation"} {
		u := unit.New(name)
		u.SetWorth(500)
		store.Put(u)
	}
	store.Get("Edit").Set("creditFeatureSubject", "Feature")
	store.Get("Edit").Set("creditFeatureKey", "feature/a")
	store.Get("Edit").Set("creditRelationSubject", "Relation")
	store.Get("Edit").Set("creditRelationKey", "relation/a")
	store.Get("Feature").Set("creditFeatureKey", "different")
	store.Get("Relation").Set("creditRelationKey", "relation/a")
	program := unit.New("Program")
	program.SetWorth(800)
	program.Set("creationWorth", 500)
	program.Set("lastRewardedWorth", 500)
	program.Set("components", []string{"Edit"})
	program.Set("creditors", []string{"H", "Edit"})
	program.Set("creditRoles", []string{"synthesis", "step-1"})
	program.Set("creditContext", "test/atomic/v1")
	program.Set("creditDecision", "concrete")
	program.Set("synthesisMethod", "ordered/v1")
	store.Put(program)
	New(store, agenda.New()).rewardForWorthGrowth()
	if store.Get("Feature").Worth() != 500 || credit.RewardTotal(store, credit.Tuple{Context: "test/atomic/v1", Subject: "Feature", Role: "component"}) != 0 {
		t.Fatal("malformed structural declaration produced partial credit")
	}
}
