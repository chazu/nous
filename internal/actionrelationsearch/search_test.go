package actionrelationsearch

import (
	"slices"
	"testing"

	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestCertifiedPoliciesPreserveTerminalBehaviors(t *testing.T) {
	world := actionrelations.World{
		State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "a", Value: 0}, {Name: "b", Value: 0}, {Name: "c", Value: 0}}},
		Actions: []actionrelations.Action{
			{Name: "one", Kind: "set", X: "a", N: 1},
			{Name: "two", Kind: "set", X: "b", N: 2},
			{Name: "three", Kind: "set", X: "c", N: 3},
		},
	}
	complete, err := Search(world, Complete, Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range []Policy{Lexical, StaticSleep, DynamicSleep} {
		result, err := Search(world, policy, Artifact{})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(result.TerminalDigests, complete.TerminalDigests) {
			t.Fatalf("%s terminals=%v complete=%v", policy, result.TerminalDigests, complete.TerminalDigests)
		}
		if policy != Lexical && result.SleepPropagations == 0 {
			t.Fatalf("%s produced no certified sleep propagation: result=%+v complete=%+v", policy, result, complete)
		}
	}
}

func TestConflictingActionsAreNeverSleepPruned(t *testing.T) {
	world := actionrelations.World{
		State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "x", Value: 0}}},
		Actions: []actionrelations.Action{
			{Name: "low", Kind: "set", X: "x", N: 1},
			{Name: "high", Kind: "set", X: "x", N: 3},
		},
	}
	complete, _ := Search(world, Complete, Artifact{})
	dynamic, _ := Search(world, DynamicSleep, Artifact{})
	if !slices.Equal(complete.TerminalDigests, dynamic.TerminalDigests) || dynamic.SleepPropagations != 0 {
		t.Fatalf("complete=%+v dynamic=%+v", complete, dynamic)
	}
}
