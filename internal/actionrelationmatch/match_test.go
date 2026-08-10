package actionrelationmatch

import (
	"testing"

	"github.com/chazu/nous/internal/actionrelationacquire"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestOrdinaryCUEMatchesEveryRetainedRelationUnanimously(t *testing.T) {
	acquisition, err := actionrelationacquire.Execute("../../domains", "match-test")
	if err != nil {
		t.Fatal(err)
	}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "add", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "add", XRole: "c1", N: 1}}
	result, err := Execute(acquisition.Store, acquisition.Artifact, state, a, b, "positive")
	if err != nil || result.Terminal != "completed" || !result.Matched || result.Barrier == "" {
		t.Fatalf("positive=%+v err=%v", result, err)
	}
	other := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c1", N: 1}}
	result, err = Execute(acquisition.Store, acquisition.Artifact, state, a, other, "negative")
	if err != nil || result.Terminal != "completed" || result.Matched {
		t.Fatalf("negative=%+v err=%v", result, err)
	}
}
