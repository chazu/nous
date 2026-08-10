package actionrelationmatch

import (
	"testing"

	"github.com/chazu/nous/internal/actionrelationacquire"
	"github.com/chazu/nous/internal/actionrelationexp"
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

func TestLearnedMatchRetainsExactUtilityDecoderWires(t *testing.T) {
	acquisition, err := actionrelationacquire.ExecuteFamily("../../domains", "match-wire", 1)
	if err != nil {
		t.Fatal(err)
	}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}, {Name: "c2", Value: 0}}, Events: []string{}}
	occurrences, _ := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{{Kind: "add", XRole: "c0", N: 1}, {Kind: "add", XRole: "c0", N: 1}})
	result, err := Execute(acquisition.Store, acquisition.Artifact, state, occurrences[0], occurrences[1], "wire")
	if err != nil || !result.Matched {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	barrier := acquisition.Store.Get(result.Barrier)
	if barrier == nil || actionrelationexp.ValidateObject(43, []byte(barrier.GetString("canonicalObject"))) != nil {
		t.Fatal("unanimous-use barrier does not decode as kind 43")
	}
	request := acquisition.Store.Get(result.Request)
	for _, rowName := range request.GetStrings("matchRows") {
		row := acquisition.Store.Get(rowName)
		if err := actionrelationexp.ValidateObject(42, []byte(row.GetString("canonicalObject"))); err != nil {
			t.Fatal(err)
		}
		for _, literalName := range row.GetStrings("literalRows") {
			literal := acquisition.Store.Get(literalName)
			if err := actionrelationexp.ValidateObject(41, []byte(literal.GetString("canonicalObject"))); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, name := range []string{"Match.App.A." + result.Request, "Match.App.B." + result.Request} {
		row := acquisition.Store.Get(name)
		if row == nil {
			t.Fatalf("missing applicability row %q", name)
		}
		if err := actionrelationexp.ValidateObject(38, []byte(row.GetString("canonicalObject"))); err != nil {
			t.Fatal(err)
		}
	}
}
