package ruleinductionexp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
	rivocab "github.com/chazu/nous/internal/vocab/ruleinduction"
)

func TestDevelopmentPoliciesRunThroughProductionVocabulary(t *testing.T) {
	report, err := RunDevelopment("../../domains")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Policies) != len(policies) {
		t.Fatalf("policies = %d", len(report.Policies))
	}
	if !report.Mechanical.AllValid || report.Status == "invalid" {
		t.Fatalf("development mechanics/status = %+v/%s", report.Mechanical, report.Status)
	}
	for _, policy := range report.Policies {
		if policy.Identified != 16 || policy.Correct != 16 || policy.HeldOutCorrect != policy.HeldOutTotal || policy.HeldOutTotal != 16*128 {
			t.Fatalf("policy %s identified/correct/heldout = %d/%d/%d/%d", policy.Name, policy.Identified, policy.Correct, policy.HeldOutCorrect, policy.HeldOutTotal)
		}
		for _, fixture := range policy.Fixtures {
			if !fixture.HeldOutStoreUnchanged {
				t.Fatalf("policy %s seed %d mutated the production store while scoring held-out data", policy.Name, fixture.Seed)
			}
		}
	}
	var direct, shared PolicyReport
	for _, policy := range report.Policies {
		if policy.Name == LFFDirect {
			direct = policy
		}
		if policy.Name == SharedLibrary {
			shared = policy
		}
	}
	if shared.FixedPointWork >= direct.FixedPointWork || shared.Stage2Candidates >= direct.Stage2Candidates {
		t.Fatalf("shared work/candidates %d/%d, direct %d/%d", shared.FixedPointWork, shared.Stage2Candidates, direct.FixedPointWork, direct.Stage2Candidates)
	}
}

func TestTranscriptActionReplayIsIdempotent(t *testing.T) {
	fixture, err := Generate("development", 11001, Beneficial)
	if err != nil {
		t.Fatal(err)
	}
	_, err = runPolicyHook("../../domains", fixture, SharedLibrary, func(store *unit.Store, experiment *unit.Unit) error {
		var action *unit.Unit
		for _, name := range store.All() {
			u := store.Get(name)
			if u.GetString("experiment") == experiment.Name && u.GetString("artifactKind") == "transcript" {
				action = u
				break
			}
		}
		if action == nil {
			return fmt.Errorf("transcript action missing")
		}
		before, _ := store.CanonicalJSON()
		vm := dsl.NewVM(store, agenda.New(), nil)
		stage := fmt.Sprintf("stage%d", action.GetInt("stageIndex"))
		program := fmt.Sprintf("%q %q %q %q %d ri-record-action", experiment.Name, stage, action.GetString("action"), action.GetString("actionSemantic"), action.GetInt("domainWork"))
		value, err := vm.Execute(program)
		if err != nil || !value.AsBool() {
			return fmt.Errorf("replay: value=%v err=%v", value, err)
		}
		after, _ := store.CanonicalJSON()
		if string(before) != string(after) {
			return fmt.Errorf("replay mutated store")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFrozenHarmfulSensitivityLedger(t *testing.T) {
	fixture := HarmfulSensitivityFixture()
	stage1 := []string{"03", "14", "04", "05", "12", "13", "15", "23", "24", "25", "01", "02", "34", "35", "45"}
	stage2 := []string{"14", "03", "04", "05", "12", "13", "15", "23", "24", "25", "01", "02", "34", "35", "45"}
	options := runOptions{stage1Queue: stage1, stage2Queue: stage2}
	direct, err := runPolicyMode("../../domains", fixture, LFFDirect, options, nil)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := runPolicyMode("../../domains", fixture, SharedLibrary, options, nil)
	if err != nil {
		t.Fatal(err)
	}
	if direct.Stage1Definition != "03" || direct.Stage2Definition != "14" || shared.Stage1Definition != "03" || shared.Stage2Definition != "14" || !shared.FellBack {
		t.Fatalf("direct=%s/%s shared=%s/%s fallback=%t", direct.Stage1Definition, direct.Stage2Definition, shared.Stage1Definition, shared.Stage2Definition, shared.FellBack)
	}
	wantDirect := WorkReport{PartialAST: 298, FixedPoint: 838, Cache: 2, AllocationProbes: 406, ArtifactEnvelopes: 12992, TranscriptDigest: 1984, Selection: 4, Total: 16524}
	wantShared := WorkReport{PartialAST: 298, FixedPoint: 842, Cache: 3, AllocationProbes: 420, ArtifactEnvelopes: 13440, TranscriptDigest: 2016, Selection: 5, Total: 17024}
	if direct.Work != wantDirect || shared.Work != wantShared {
		t.Fatalf("ledger drift: direct=%+v want=%+v shared=%+v want=%+v", direct.Work, wantDirect, shared.Work, wantShared)
	}
	if shared.Work.Total <= direct.Work.Total || float64(shared.Work.Total)/float64(direct.Work.Total) > 2 || shared.CandidatesExecuted != direct.CandidatesExecuted+1 {
		t.Fatalf("direct work/candidates=%d/%d shared=%d/%d", direct.Work.Total, direct.CandidatesExecuted, shared.Work.Total, shared.CandidatesExecuted)
	}
}

func TestReportJSONHasNoNullCollectionsAndFitsCap(t *testing.T) {
	report := Report{
		ReportVersion: "rule-induction-report/v1",
		Manifest:      PreregisteredManifest(),
		PlanCommit:    "test",
		Panel:         "development",
		Status:        "invalid",
		Policies:      []PolicyReport{},
		Contrasts:     []ContrastReport{},
		Limitations:   []string{},
	}
	encoded, err := report.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), ": null") {
		t.Fatalf("report contains a null collection: %s", encoded)
	}
	if len(encoded) > report.Manifest.ReportByteCap {
		t.Fatalf("report bytes = %d, cap = %d", len(encoded), report.Manifest.ReportByteCap)
	}
}

func TestLockedPanelCannotUsePublicRunner(t *testing.T) {
	if _, err := RunPanel("../../domains", "locked"); err == nil {
		t.Fatal("public runner accepted locked panel without commit provenance")
	}
}

func TestNoSolutionControlTerminatesWithoutPromotion(t *testing.T) {
	fixture, err := GenerateNoSolution(51001)
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range policies {
		report, err := runPolicy("../../domains", fixture, policy)
		if err != nil {
			t.Fatalf("policy %s: %v", policy, err)
		}
		if report.Terminal != "no-solution" || report.Stage2Definition != "" || !report.ExperimentComplete || !report.AgendaDrained || report.CandidatesExecuted < 15 {
			t.Fatalf("policy=%s terminal=%s definition=%q complete=%t drained=%t executed=%d", policy, report.Terminal, report.Stage2Definition, report.ExperimentComplete, report.AgendaDrained, report.CandidatesExecuted)
		}
		encoded, err := json.Marshal(report)
		if err != nil || strings.Contains(string(encoded), ":null") {
			t.Fatalf("policy=%s nullable no-solution report: %s err=%v", policy, encoded, err)
		}
	}
}

func TestAdversarialControlSuiteFailsClosed(t *testing.T) {
	controls := runControls("../../domains")
	if !allControlsValid(controls) {
		t.Fatalf("controls = %+v", controls)
	}
}

func TestTranscriptProfilePruneAndBudgetCorruptionFailClosed(t *testing.T) {
	fixture, err := Generate("development", 11001, Beneficial)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*unit.Store, *unit.Unit) error{
		"delete transcript": func(store *unit.Store, experiment *unit.Unit) error {
			for _, name := range store.All() {
				if store.Get(name).GetString("experiment") == experiment.Name && store.Get(name).GetString("artifactKind") == "transcript" {
					store.Delete(name)
					return nil
				}
			}
			return fmt.Errorf("transcript missing")
		},
		"duplicate transcript": func(store *unit.Store, experiment *unit.Unit) error {
			for _, name := range store.All() {
				if store.Get(name).GetString("experiment") == experiment.Name && store.Get(name).GetString("artifactKind") == "transcript" {
					store.Put(cloneUnit(store.Get(name), name+"-duplicate"))
					return nil
				}
			}
			return fmt.Errorf("transcript missing")
		},
		"forge digest": func(store *unit.Store, experiment *unit.Unit) error {
			for _, name := range store.All() {
				if store.Get(name).GetString("experiment") == experiment.Name && store.Get(name).GetString("artifactKind") == "transcript" {
					store.Get(name).Set("prefixDigest", "sha256:"+strings.Repeat("0", 64))
					return nil
				}
			}
			return fmt.Errorf("transcript missing")
		},
		"stale profile": func(store *unit.Store, experiment *unit.Unit) error {
			store.Get(experiment.GetString("projectionUnit")).Set("stageProfileKey", experiment.GetString("stage1ProfileKey"))
			return nil
		},
		"stage one mutation": func(store *unit.Store, experiment *unit.Unit) error {
			for _, name := range store.All() {
				u := store.Get(name)
				if u.GetString("experiment") == experiment.Name && u.GetString("artifactKind") == "partial" && u.GetInt("stageIndex") == 1 {
					u.Set("partial", "0000")
					return nil
				}
			}
			return fmt.Errorf("stage-one partial missing")
		},
		"unsound prune": func(store *unit.Store, experiment *unit.Unit) error {
			u := unit.New("RI.UnsoundPrune")
			u.Set("isA", []string{experiment.GetString("pruneCategory"), "Anything"})
			u.Set("experiment", experiment.Name)
			u.Set("experimentProfileKey", experiment.GetString("experimentProfileKey"))
			u.Set("stageIndex", 2)
			u.Set("stageProfileKey", experiment.GetString("stage2ProfileKey"))
			u.Set("artifactKind", "prune")
			u.Set("semanticKey", rivocab.ArtifactSemanticKey("prune", "unsound"))
			store.Put(u)
			return nil
		},
		"budget charge": func(store *unit.Store, experiment *unit.Unit) error {
			for _, name := range store.All() {
				if store.Get(name).GetString("experiment") == experiment.Name && store.Get(name).GetString("artifactKind") == "transcript" {
					store.Get(name).Set("domainWork", 500001)
					return nil
				}
			}
			return fmt.Errorf("transcript missing")
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			if !corruptionRejected("../../domains", fixture, mutate) {
				t.Fatal("corruption was not rejected")
			}
		})
	}
}

func TestFailedFrozenProjectionCorruptionFailsClosed(t *testing.T) {
	fixture := HarmfulSensitivityFixture()
	if !corruptionRejected("../../domains", fixture, func(store *unit.Store, experiment *unit.Unit) error {
		projection := store.Get(experiment.GetString("projectionUnit"))
		if projection == nil || projection.GetInt("failureCount") == 0 {
			return fmt.Errorf("failed frozen projection missing")
		}
		store.Delete(projection.Name)
		return nil
	}) {
		t.Fatal("failed frozen projection deletion was not rejected")
	}
	if !corruptionRejected("../../domains", fixture, func(_ *unit.Store, experiment *unit.Unit) error {
		experiment.Set("fellBack", false)
		return nil
	}) {
		t.Fatal("erased fallback state was not rejected")
	}
}

func TestAllocatorRejectsAuthoritativeSlotConflict(t *testing.T) {
	fixture := HarmfulSensitivityFixture()
	checked := false
	_, _ = runPolicyHook("../../domains", fixture, SharedLibrary, func(store *unit.Store, experiment *unit.Unit) error {
		projection := store.Get(experiment.GetString("projectionUnit"))
		if projection == nil || projection.GetInt("failureCount") == 0 {
			return fmt.Errorf("failed frozen projection missing")
		}
		vm := dsl.NewVM(store, agenda.New(), nil)
		semantic := "projection:" + experiment.GetString("frozenCode")
		program := fmt.Sprintf("%q %q %q %q %q ri-artifact-name", experiment.Name, "stage2", "candidate", semantic, projection.Name)
		exact, err := vm.Execute(program)
		if err != nil || exact.IsNil() || exact.AsString() != projection.Name {
			return fmt.Errorf("exact reuse failed: value=%v err=%v", exact, err)
		}
		projection.Set("failureCount", projection.GetInt("failureCount")+1)
		conflict, err := vm.Execute(program)
		if err != nil || !conflict.IsNil() {
			return fmt.Errorf("conflicting reuse accepted: value=%v err=%v", conflict, err)
		}
		checked = true
		return fmt.Errorf("stop after allocator witness")
	})
	if !checked {
		t.Fatal("allocator conflict witness did not run")
	}
}
