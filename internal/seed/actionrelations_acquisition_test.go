package seed

import (
	"io"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestActionRelationCUEAllocatesCompleteGuardSpace(t *testing.T) {
	store, experiment := allocateActionRelationGuardSpace(t, false)
	if !experiment.GetBool("guardSpaceAllocated") || len(experiment.GetStrings("candidateUnits")) != 451 || len(experiment.GetStrings("edgeUnits")) != 450 {
		t.Fatalf("allocated=%v candidates=%d edges=%d terminal=%q", experiment.GetBool("guardSpaceAllocated"), len(experiment.GetStrings("candidateUnits")), len(experiment.GetStrings("edgeUnits")), experiment.GetString("terminal"))
	}
	for ordinal, name := range experiment.GetStrings("candidateUnits") {
		candidate := store.Get(name)
		if candidate == nil || candidate.GetInt("ordinal") != ordinal {
			t.Fatalf("candidate ordinal %d: %#v", ordinal, candidate)
		}
		if _, err := actionrelations.ParseGuard([]byte(candidate.GetString("guard"))); err != nil {
			t.Fatalf("candidate %d guard: %v", ordinal, err)
		}
	}
}

func TestActionRelationCUEGuardSpaceIsStableWithOccupiedNames(t *testing.T) {
	baselineStore, baseline := allocateActionRelationGuardSpace(t, false)
	occupiedStore, occupied := allocateActionRelationGuardSpace(t, true)
	if occupiedStore.Get("AR.Candidate.AR.Test.Experiment.0").GetString("owner") != "user" || occupiedStore.Get("AR.Edge.AR.Test.Experiment.0").GetString("owner") != "user" {
		t.Fatal("occupied unit was overwritten")
	}
	for index, baselineName := range baseline.GetStrings("candidateUnits") {
		actualName := occupied.GetStrings("candidateUnits")[index]
		if baselineStore.Get(baselineName).GetString("objectDigest") != occupiedStore.Get(actualName).GetString("objectDigest") {
			t.Fatalf("candidate identity changed at %d", index)
		}
	}
	for index, baselineName := range baseline.GetStrings("edgeUnits") {
		actualName := occupied.GetStrings("edgeUnits")[index]
		if baselineStore.Get(baselineName).GetString("objectDigest") != occupiedStore.Get(actualName).GetString("objectDigest") {
			t.Fatalf("edge identity changed at %d", index)
		}
	}
}

func TestActionRelationCUEAssemblesVisibleTrainingDiamond(t *testing.T) {
	DomainsDir = "../../domains"
	store := unit.NewStore()
	if err := LoadDomain(store, "actionrelations"); err != nil {
		t.Fatal(err)
	}
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	a := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c0", N: 1}}
	b := actionrelations.Occurrence{Action: actionrelations.SemanticAction{Kind: "set", XRole: "c1", N: 1}}
	stateJSON, _ := state.CanonicalJSON()
	aJSON, _ := a.CanonicalJSON()
	bJSON, _ := b.CanonicalJSON()
	experiment := unit.New("AR.Observe.Experiment")
	experiment.Set("isA", []string{"ActionRelationExperiment", "Anything"})
	experiment.Set("expectedObservationCount", 1)
	pattern := actionrelations.Pattern{Kinds: []string{"set", "set"}, Roles: []int{0, -1, 1, -1}}
	patternJSON, _ := pattern.CanonicalJSON()
	experiment.Set("pattern", string(patternJSON))
	store.Put(experiment)
	training := unit.New("AR.Training.0")
	training.Set("isA", []string{"ActionRelationTrainingCase", "Anything"})
	training.Set("experiment", experiment.Name)
	training.Set("state", string(stateJSON))
	training.Set("aOccurrence", string(aJSON))
	training.Set("bOccurrence", string(bJSON))
	training.Set("label", "commutes")
	store.Put(training)
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: "arObserve"})
	if eng.LastError != nil {
		t.Fatal(eng.LastError)
	}
	observations := experiment.GetStrings("observationUnits")
	if !experiment.GetBool("observationsAssembled") || len(observations) != 1 || store.Get(observations[0]).GetString("label") != "commutes" {
		t.Fatalf("assembled=%v observations=%v terminal=%q", experiment.GetBool("observationsAssembled"), observations, experiment.GetString("terminal"))
	}
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: "arAllocate"})
	if eng.LastError != nil {
		t.Fatal(eng.LastError)
	}
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: "arEvaluate"})
	if eng.LastError != nil {
		t.Fatal(eng.LastError)
	}
	if !experiment.GetBool("guardsEvaluated") || len(experiment.GetStrings("guardResultUnits")) != 451 || len(experiment.GetStrings("literalRowUnits")) != 870 {
		t.Fatalf("evaluated=%v results=%d literals=%d terminal=%q", experiment.GetBool("guardsEvaluated"), len(experiment.GetStrings("guardResultUnits")), len(experiment.GetStrings("literalRowUnits")), experiment.GetString("terminal"))
	}
}

func allocateActionRelationGuardSpace(t *testing.T, occupied bool) (*unit.Store, *unit.Unit) {
	t.Helper()
	DomainsDir = "../../domains"
	store := unit.NewStore()
	if err := LoadDomain(store, "actionrelations"); err != nil {
		t.Fatal(err)
	}
	pattern := actionrelations.Pattern{Kinds: []string{"add", "add"}, Roles: []int{0, -1, 1, -1}}
	patternJSON, _ := pattern.CanonicalJSON()
	experiment := unit.New("AR.Test.Experiment")
	experiment.Set("isA", []string{"ActionRelationExperiment", "Anything"})
	experiment.Set("pattern", string(patternJSON))
	store.Put(experiment)
	if occupied {
		for _, name := range []string{"AR.Candidate.AR.Test.Experiment.0", "AR.Candidate.AR.Test.Experiment.17", "AR.Edge.AR.Test.Experiment.0", "AR.Edge.AR.Test.Experiment.333"} {
			placeholder := unit.New(name)
			placeholder.Set("owner", "user")
			store.Put(placeholder)
		}
	}
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: "arAllocate"})
	if eng.LastError != nil {
		t.Fatal(eng.LastError)
	}
	return store, experiment
}
