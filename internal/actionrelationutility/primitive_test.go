package actionrelationutility

import (
	"strings"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestOrdinaryCUESearchPrimitivesAreContextBoundAndCharged(t *testing.T) {
	previous := seed.DomainsDir
	seed.DomainsDir = "../../domains"
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "actionrelations"); err != nil {
		t.Fatal(err)
	}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	world := actionrelations.World{State: state, Actions: []actionrelations.Action{
		{Name: "a", Kind: "add", X: "c0", N: 1}, {Name: "b", Kind: "add", X: "c1", N: 1},
	}}
	normalized, err := world.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	state = normalized.State
	occurrences := normalized.Occurrences
	worldDigest, _ := normalized.Digest()
	remaining, _ := actionrelationsearch.BuildRemaining(occurrences)
	proofMap, _ := actionrelationsearch.BuildProofMap(nil)
	nodeObject, _ := actionrelationsearch.BuildSearchNode(state, remaining, proofMap)
	stateDigest, _ := state.Digest()
	occurrenceDigests := make([]string, len(occurrences))
	for index, occurrence := range occurrences {
		occurrenceDigests[index], _ = occurrence.Digest()
	}
	node := unit.New("AR.Utility.Node")
	node.Set("isA", []string{"ActionRelationSearchNode", "Anything"})
	node.Set("canonicalObject", string(nodeObject.Canonical))
	node.Set("objectDigest", nodeObject.Digest)
	node.Set("worldDigest", worldDigest)
	node.Set("policy", "static-rw-sleep")
	node.Set("stateDigest", stateDigest)
	node.Set("remainingOccurrenceDigests", occurrenceDigests)
	store.Put(node)

	meterToken := "utility-primitives"
	plan := []dsl.ActionRelationMeterPlanEntry{
		{Code: 23, SourceTaskDigest: strings.Repeat("1", 64)},
		{Code: 11, SourceTaskDigest: strings.Repeat("2", 64)},
		{Code: 24, SourceTaskDigest: strings.Repeat("3", 64)},
	}
	if err := dsl.RegisterActionRelationMeterPlan(meterToken, plan); err != nil {
		t.Fatal(err)
	}
	defer dsl.UnregisterActionRelationMeter(meterToken)
	applicable, err := SearchApplicable(store, meterToken, node.Name, worldDigest, "static-rw-sleep", state, occurrences[0], "applicable")
	if err != nil || !applicable.Result || actionrelationexp.ValidateObject(38, []byte(store.Get(applicable.Row).GetString("canonicalObject"))) != nil {
		t.Fatalf("applicable=%+v err=%v", applicable, err)
	}
	transition, err := SearchApply(store, meterToken, applicable.Row, state, occurrences[0], "transition")
	if err != nil || transition.Outcome != "applied" || transition.OutputState == "" || actionrelationexp.ValidateObject(39, []byte(store.Get(transition.Row).GetString("canonicalObject"))) != nil {
		t.Fatalf("transition=%+v err=%v", transition, err)
	}
	footprint, err := StaticFootprint(store, meterToken, node.Name, worldDigest, state, occurrences[0], occurrences[1], "footprint")
	if err != nil || !footprint.Result || actionrelationexp.ValidateObject(48, []byte(store.Get(footprint.Row).GetString("canonicalObject"))) != nil {
		t.Fatalf("footprint=%+v err=%v", footprint, err)
	}
	if err := dsl.ActionRelationMeterPlanComplete(meterToken); err != nil {
		t.Fatal(err)
	}
	records, _ := dsl.ActionRelationMeterSnapshot(meterToken)
	if len(records) != 3 || records[0].Code != 23 || records[1].Code != 11 || records[2].Code != 24 || records[0].SourceTaskDigest != plan[0].SourceTaskDigest || records[1].SourceTaskDigest != plan[1].SourceTaskDigest || records[2].SourceTaskDigest != plan[2].SourceTaskDigest {
		t.Fatalf("records=%#v", records)
	}
}

func TestSearchApplicabilityRejectsCrossContextNode(t *testing.T) {
	previous := seed.DomainsDir
	seed.DomainsDir = "../../domains"
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "actionrelations"); err != nil {
		t.Fatal(err)
	}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}}}
	occurrences, _ := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{{Kind: "add", XRole: "c0", N: 1}})
	node := unit.New("AR.CrossContext.Node")
	node.Set("isA", []string{"ActionRelationSearchNode", "Anything"})
	node.Set("worldDigest", strings.Repeat("a", 64))
	node.Set("policy", "complete")
	stateDigest, _ := state.Digest()
	occurrenceDigest, _ := occurrences[0].Digest()
	node.Set("stateDigest", stateDigest)
	node.Set("remainingOccurrenceDigests", []string{occurrenceDigest})
	store.Put(node)
	if err := dsl.RegisterActionRelationMeterPlan("cross-context", []dsl.ActionRelationMeterPlanEntry{{Code: 23, SourceTaskDigest: strings.Repeat("b", 64)}}); err != nil {
		t.Fatal(err)
	}
	defer dsl.UnregisterActionRelationMeter("cross-context")
	if _, err := SearchApplicable(store, "cross-context", node.Name, strings.Repeat("c", 64), "complete", state, occurrences[0], "cross"); err == nil {
		t.Fatal("cross-context search node was accepted")
	}
	records, _ := dsl.ActionRelationMeterSnapshot("cross-context")
	if len(records) != 0 {
		t.Fatal("rejected context consumed a semantic operation")
	}
}
