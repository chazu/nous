package actionrelationfixture

import (
	"bytes"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/actionrelationwire"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestCurriculumTruthIsSealedBeforePoliciesAndPartitionsExactRows(t *testing.T) {
	context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: 0, CurriculumSeed: 851001, Attempt: 0}
	curriculum, err := BuildCurriculum(context)
	if err != nil {
		t.Fatal(err)
	}
	truth, err := SealCurriculumTruth(curriculum)
	if err != nil || len(truth.Worlds) != 6 || truth.Root == "" {
		t.Fatalf("worlds=%d root=%q err=%v", len(truth.Worlds), truth.Root, err)
	}
	for slot, worldTruth := range truth.Worlds {
		if err := VerifyWorldTruth(worldTruth); err != nil {
			t.Fatalf("world %d: %v", slot, err)
		}
		for _, shard := range worldTruth.Shards {
			if actionrelationexp.ValidateObject(29, shard.Canonical) != nil {
				t.Fatalf("world %d shard %d does not decode as kind 29", slot, shard.Ordinal)
			}
		}
		view := curriculum.Worlds[slot]
		complete, err := actionrelationsearch.Search(actionrelations.World{State: view.State, Actions: view.Actions}, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
		if err != nil || !equalStrings(worldTruth.Terminals, complete.TerminalDigests) {
			t.Fatalf("world %d terminals=%v complete=%v err=%v", slot, worldTruth.Terminals, complete.TerminalDigests, err)
		}
	}
	var rootRows []any
	for _, worldTruth := range truth.Worlds {
		for _, shard := range worldTruth.Shards {
			rootRows = append(rootRows, []any{worldTruth.WorldDigest, shard.Ordinal, shard.Digest})
		}
	}
	// Curriculum fixture order is intentionally not scorer-root order.
	for left := 0; left < len(rootRows); left++ {
		for right := left + 1; right < len(rootRows); right++ {
			a := rootRows[left].([]any)
			b := rootRows[right].([]any)
			if a[0].(string) > b[0].(string) || a[0] == b[0] && a[1].(int) > b[1].(int) {
				rootRows[left], rootRows[right] = rootRows[right], rootRows[left]
			}
		}
	}
	wantRoot, err := actionrelationwire.RootDigest("scorer-shards", rootRows)
	if err != nil || truth.Root != wantRoot {
		t.Fatalf("scorer root=%q want=%q err=%v", truth.Root, wantRoot, err)
	}
	again, err := SealCurriculumTruth(curriculum)
	if err != nil || again.Root != truth.Root || !bytes.Equal(again.Worlds[0].Shards[0].Canonical, truth.Worlds[0].Shards[0].Canonical) {
		t.Fatal("sealed truth is not deterministic")
	}
}

func TestWorldTruthRejectsPairReordering(t *testing.T) {
	context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: 1, CurriculumSeed: 851002, Attempt: 0}
	curriculum, err := BuildCurriculum(context)
	if err != nil {
		t.Fatal(err)
	}
	truth, err := SealWorldTruth(curriculum.Worlds[0].Core.World)
	if err != nil || len(truth.PairRows) < 2 {
		t.Fatalf("rows=%d shards=%d err=%v", len(truth.PairRows), len(truth.Shards), err)
	}
	truth.PairRows[0], truth.PairRows[1] = truth.PairRows[1], truth.PairRows[0]
	if err := VerifyWorldTruth(truth); err == nil {
		t.Fatal("truth verifier accepted reordered pair labels")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}
