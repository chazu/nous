package credit

import (
	"strings"
	"testing"

	"github.com/chazu/nous/internal/unit"
)

func TestUpsertAggregatesAndIsolatesTuples(t *testing.T) {
	store := contextualStore()
	tuple := Tuple{Context: "rewrite/v1", Subject: "decision", Role: "first"}
	first := Upsert(store, tuple, 25, Provenance{SourceUnit: "A", RewardTaskNum: 3})
	second := Upsert(store, tuple, 10, Provenance{SourceUnit: "B", RewardTaskNum: 4})
	if first != second || first.Worth() != 0 || first.GetInt("rewardTotal") != 35 || first.GetInt("evidenceCount") != 2 {
		t.Fatalf("aggregate = %#v", first)
	}
	if first.GetString("lastSourceUnit") != "B" || first.GetInt("lastRewardTaskNum") != 4 {
		t.Fatalf("provenance = %#v", first.Slots)
	}
	if RewardTotal(store, Tuple{Context: "other/v1", Subject: "decision", Role: "first"}) != 0 {
		t.Fatal("credit leaked across contexts")
	}
}

func TestCollisionLookupReusesTupleAcrossGaps(t *testing.T) {
	store := contextualStore()
	tuple := Tuple{Context: "rewrite/v1", Subject: "A>B", Role: "decision"}
	base := recordName(tuple)
	store.Put(unit.New(base))
	store.Put(unit.New(base + "-collision-1"))
	record := Upsert(store, tuple, 7, Provenance{})
	if record.Name != base+"-collision-2" {
		t.Fatalf("collision name = %q", record.Name)
	}
	store.Delete(base + "-collision-1")
	again := Upsert(store, tuple, 5, Provenance{})
	if again.Name != record.Name || again.GetInt("rewardTotal") != 12 {
		t.Fatalf("gap lookup allocated %q with total %d", again.Name, again.GetInt("rewardTotal"))
	}
}

func TestDeclarationBounds(t *testing.T) {
	if !ValidDeclaration("ctx", "decision", []string{"H", "P"}, []string{"synthesis", "first"}) {
		t.Fatal("valid declaration rejected")
	}
	invalid := []struct {
		context   string
		decision  string
		creditors []string
		roles     []string
	}{
		{"", "d", nil, nil},
		{"ctx", "", nil, nil},
		{"ctx", "d", []string{"P"}, nil},
		{"ctx", "d", []string{""}, []string{"first"}},
		{"ctx", "d", []string{"P"}, []string{""}},
		{strings.Repeat("x", MaxContextBytes+1), "d", nil, nil},
	}
	for _, declaration := range invalid {
		if ValidDeclaration(declaration.context, declaration.decision, declaration.creditors, declaration.roles) {
			t.Fatalf("invalid declaration accepted: %#v", declaration)
		}
	}
}

func contextualStore() *unit.Store {
	store := unit.NewStore()
	category := unit.New(Category)
	category.SetWorth(0)
	category.Set("isA", []string{"Anything"})
	store.Put(category)
	return store
}
