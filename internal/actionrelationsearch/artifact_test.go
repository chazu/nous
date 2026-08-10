package actionrelationsearch_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationacquire"
	"github.com/chazu/nous/internal/actionrelationcertify"
	"github.com/chazu/nous/internal/actionrelationsearch"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestLearnedArtifactFiltersCertifiedSleepWithoutChangingBehavior(t *testing.T) {
	acquisition, err := actionrelationacquire.Execute("../../domains", "search-test")
	if err != nil {
		t.Fatal(err)
	}
	artifactUnit := acquisition.Store.Get(acquisition.Artifact)
	var relations []actionrelations.Relation
	for _, name := range artifactUnit.GetStrings("relationUnits") {
		relation, err := actionrelations.ParseRelation([]byte(acquisition.Store.Get(name).GetString("relation")))
		if err != nil {
			t.Fatal(err)
		}
		relations = append(relations, relation)
	}
	world := actionrelations.World{
		State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "blue", Value: 0}, {Name: "green", Value: 0}, {Name: "red", Value: 0}}},
		Actions: []actionrelations.Action{
			{Name: "first", Kind: "add", X: "red", N: 1},
			{Name: "second", Kind: "add", X: "green", N: 1},
			{Name: "third", Kind: "add", X: "blue", N: 1},
		},
	}
	complete, err := actionrelationsearch.Search(world, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range []actionrelationsearch.Policy{actionrelationsearch.NousSleep, actionrelationsearch.NoGuardSleep} {
		result, err := actionrelationsearch.Search(world, policy, actionrelationsearch.Artifact{Relations: relations})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(result.TerminalDigests, complete.TerminalDigests) || result.SleepPropagations == 0 {
			t.Fatalf("policy=%s result=%+v complete=%+v", policy, result, complete)
		}
	}
	control, err := actionrelationsearch.Search(world, actionrelationsearch.LearnedNoUse, actionrelationsearch.Artifact{Relations: relations})
	if err != nil || !slices.Equal(control.TerminalDigests, complete.TerminalDigests) || control.SleepPropagations != 0 {
		t.Fatalf("learned-no-use=%+v err=%v", control, err)
	}
}

func TestSearchCanRequireOrdinaryCUECertificates(t *testing.T) {
	acquisition, err := actionrelationacquire.Execute("../../domains", "cue-certificate-search")
	if err != nil {
		t.Fatal(err)
	}
	world := actionrelations.World{
		State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "a", Value: 0}, {Name: "b", Value: 0}, {Name: "c", Value: 0}}},
		Actions: []actionrelations.Action{
			{Name: "one", Kind: "set", X: "a", N: 1},
			{Name: "two", Kind: "set", X: "b", N: 2},
			{Name: "three", Kind: "set", X: "c", N: 3},
		},
	}
	witnessDigest := sha256.Sum256([]byte("all-pairs"))
	witness, _ := json.Marshal([]any{"dynamic-witness/v1", "all-pairs", hex.EncodeToString(witnessDigest[:])})
	operationDigest := sha256.Sum256([]byte("cue-certificate-operations"))
	operationRoot := hex.EncodeToString(operationDigest[:])
	calls := 0
	certifier := func(state actionrelations.State, left, right actionrelations.Occurrence) (bool, error) {
		calls++
		result, err := actionrelationcertify.Execute(acquisition.Store, state, left, right, witness, operationRoot, fmt.Sprintf("search-%d", calls))
		return result.Terminal == "certified", err
	}
	complete, _ := actionrelationsearch.Search(world, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
	dynamic, err := actionrelationsearch.SearchWithCertifier(world, actionrelationsearch.DynamicSleep, actionrelationsearch.Artifact{}, certifier)
	if err != nil || calls == 0 || dynamic.SleepPropagations == 0 || !slices.Equal(dynamic.TerminalDigests, complete.TerminalDigests) {
		t.Fatalf("dynamic=%+v complete=%+v calls=%d err=%v", dynamic, complete, calls, err)
	}
}
