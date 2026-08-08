package ruleinductionexp

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/ruleinductionoracle"
	"github.com/chazu/nous/internal/unit"
	rivocab "github.com/chazu/nous/internal/vocab/ruleinduction"
)

func freshComplete(store *unit.Store, experiment *unit.Unit) (bool, error) {
	vm := dsl.NewVM(store, agenda.New(), nil)
	if err := vm.InitError(); err != nil {
		return false, err
	}
	value, err := vm.Execute(strconv.Quote(experiment.Name) + " ri-experiment-complete?")
	if err != nil {
		return false, err
	}
	return value.AsBool(), nil
}

func cloneUnit(source *unit.Unit, name string) *unit.Unit {
	clone := unit.New(name)
	for slot, value := range source.Slots {
		clone.Set(slot, value)
	}
	return clone
}

func corruptionRejected(domainsDir string, fixture Fixture, mutate func(*unit.Store, *unit.Unit) error) bool {
	checked := false
	_, _ = runPolicyHook(domainsDir, fixture, SharedLibrary, func(store *unit.Store, experiment *unit.Unit) error {
		valid, err := freshComplete(store, experiment)
		if err != nil || !valid {
			return fmt.Errorf("pristine verification: valid=%t err=%v", valid, err)
		}
		if err := mutate(store, experiment); err != nil {
			return err
		}
		valid, err = freshComplete(store, experiment)
		checked = err == nil && !valid
		if !checked {
			return fmt.Errorf("corruption was accepted: valid=%t err=%v", valid, err)
		}
		return nil
	})
	return checked
}

func runControls(domainsDir string) ControlReport {
	fixture, err := Generate("development", 11001, Beneficial)
	if err != nil {
		return ControlReport{}
	}
	baseline, err := runPolicy(domainsDir, fixture, SharedLibrary)
	if err != nil {
		return ControlReport{}
	}
	repeated, repeatErr := runPolicy(domainsDir, fixture, SharedLibrary)

	aliasFixture := fixture
	for left, right := 0, len(aliasFixture.ConstantAliases)-1; left < right; left, right = left+1, right-1 {
		aliasFixture.ConstantAliases[left], aliasFixture.ConstantAliases[right] = aliasFixture.ConstantAliases[right], aliasFixture.ConstantAliases[left]
	}
	aliasFixture.PredicateAliases[0], aliasFixture.PredicateAliases[2] = aliasFixture.PredicateAliases[2], aliasFixture.PredicateAliases[0]
	aliasRun, aliasErr := runPolicy(domainsDir, aliasFixture, SharedLibrary)
	opaqueAlias := aliasErr == nil && aliasRun.Stage1Definition == baseline.Stage1Definition && aliasRun.Stage2Definition == baseline.Stage2Definition && aliasRun.Work == baseline.Work && aliasRun.Accuracy == baseline.Accuracy

	alternate, alternateErr := runPolicyMode(domainsDir, fixture, SharedLibrary, runOptions{alternateDescriptor: true}, nil)
	occupied, occupiedErr := runPolicyMode(domainsDir, fixture, SharedLibrary, runOptions{occupiedName: true}, nil)
	mutated, mutationErr := runPolicyMode(domainsDir, fixture, SharedLibrary, runOptions{mutation: true}, nil)
	directBaseline, directErr := runPolicy(domainsDir, fixture, LFFDirect)
	stage1Queue := queue(fixture.Panel, fixture.Seed, 1, LFFDirect)
	stage2Queue := queue(fixture.Panel, fixture.Seed, 2, LFFDirect)
	stage1Exact := ruleinductionoracle.ExactCodes(fixture.Background, fixture.Stage1)
	stage2Exact := ruleinductionoracle.ExactCodes(fixture.Background, fixture.Stage2)
	removeStage1 := ""
	for _, code := range stage1Queue {
		if !slices.Contains(stage1Exact, code) {
			removeStage1 = code
			break
		}
	}
	removeStage2 := firstExact(stage2Queue, stage2Exact)
	omitted, omittedErr := runPolicyMode(domainsDir, fixture, LFFDirect, runOptions{stage1Queue: withoutCode(stage1Queue, removeStage1), stage2Queue: withoutCode(stage2Queue, removeStage2)}, nil)

	caseFixture := fixture
	for index := range caseFixture.Edges {
		reverseEdges(caseFixture.Edges[index])
	}
	caseRun, caseErr := runPolicy(domainsDir, caseFixture, SharedLibrary)

	controls := ControlReport{
		OpaqueAlias:           opaqueAlias,
		AlternateDescriptor:   alternateErr == nil && alternate.Stage1Definition == baseline.Stage1Definition && alternate.Stage2Definition == baseline.Stage2Definition && alternate.Accuracy == baseline.Accuracy,
		CaseOrder:             caseErr == nil && caseRun.Stage1Definition == baseline.Stage1Definition && caseRun.Stage2Definition == baseline.Stage2Definition && caseRun.Work == baseline.Work,
		OccupiedName:          occupiedErr == nil && occupied.Stage1Definition == baseline.Stage1Definition && occupied.Stage2Definition == baseline.Stage2Definition && occupied.Accuracy == baseline.Accuracy,
		MutationInert:         mutationErr == nil && reflect.DeepEqual(mutated, baseline),
		AlternateQueueOmit:    directErr == nil && omittedErr == nil && omitted.ExperimentProfileKey != directBaseline.ExperimentProfileKey && omitted.Stage1Definition == directBaseline.Stage1Definition && omitted.Stage2Definition == "" && omitted.Terminal == "no-solution" && omitted.ExperimentComplete,
		HeldoutStoreImmutable: baseline.HeldOutStoreUnchanged,
	}
	controls.CandidateInsertCorruption = corruptionRejected(domainsDir, fixture, func(store *unit.Store, experiment *unit.Unit) error {
		source := store.Get(experiment.GetString("projectionUnit"))
		if source == nil {
			return fmt.Errorf("projection missing")
		}
		rogue := cloneUnit(source, "RI.RogueCandidate")
		rogue.Set("semanticKey", rivocab.ArtifactSemanticKey("candidate", "candidate:rogue"))
		rogue.Set("chargeIndex", source.GetInt("chargeIndex"))
		store.Put(rogue)
		return nil
	})
	controls.CandidateDeleteCorruption = corruptionRejected(domainsDir, fixture, func(store *unit.Store, experiment *unit.Unit) error {
		if store.Delete(experiment.GetString("projectionUnit")) == nil {
			return fmt.Errorf("projection missing")
		}
		return nil
	})
	controls.CandidateDuplicateCorruption = corruptionRejected(domainsDir, fixture, func(store *unit.Store, experiment *unit.Unit) error {
		for _, name := range store.All() {
			u := store.Get(name)
			if u.GetString("experiment") == experiment.Name && u.GetString("artifactKind") == "evidence" && u.GetInt("stageIndex") == 2 {
				store.Put(cloneUnit(u, u.Name+"-duplicate"))
				return nil
			}
		}
		return fmt.Errorf("stage-2 evidence missing")
	})
	controls.CategoryInjection = corruptionRejected(domainsDir, fixture, func(store *unit.Store, experiment *unit.Unit) error {
		for _, name := range store.All() {
			u := store.Get(name)
			if u.GetString("experiment") == experiment.Name && u.GetString("artifactKind") == "partial" && u.GetInt("stageIndex") == 1 {
				rogue := cloneUnit(u, "RI.CategoryInjection")
				rogue.Set("partial", "0000")
				rogue.Set("semanticKey", rivocab.ArtifactSemanticKey("partial", "partial:0000"))
				store.Put(rogue)
				return nil
			}
		}
		return fmt.Errorf("stage-1 partial missing")
	})
	controls.EvidencePositiveFlip = validEvidencePerturbation(domainsDir, fixture, true)
	controls.EvidenceNegativeFlip = validEvidencePerturbation(domainsDir, fixture, false)
	harmful := HarmfulSensitivityFixture()
	harmfulDirect, harmfulDirectErr := runPolicy(domainsDir, harmful, LFFDirect)
	harmfulShared, harmfulSharedErr := runPolicy(domainsDir, harmful, SharedLibrary)
	controls.WrongContext = harmfulDirectErr == nil && harmfulSharedErr == nil && harmfulShared.FellBack && harmfulShared.Stage2Definition == harmfulDirect.Stage2Definition && float64(harmfulShared.Work.Total)/float64(harmfulDirect.Work.Total) <= 2
	controls.DeterministicJSON = repeatErr == nil && reflect.DeepEqual(baseline, repeated) && deterministicReportEncoding(baseline, repeated)
	return controls
}

func validEvidencePerturbation(domainsDir string, fixture Fixture, polarity bool) bool {
	baseline, err := runPolicy(domainsDir, fixture, LFFDirect)
	if err != nil {
		return false
	}
	stage2Queue := queue(fixture.Panel, fixture.Seed, 2, LFFDirect)
	for index, example := range fixture.Stage2 {
		if example.Positive != polarity {
			continue
		}
		changed := fixture
		changed.Stage2 = append([]ruleinductionoracle.Example(nil), fixture.Stage2...)
		changed.Stage2[index].Positive = !changed.Stage2[index].Positive
		ties := ruleinductionoracle.ExactCodes(changed.Background, changed.Stage2)
		want := firstExact(stage2Queue, ties)
		wantTerminal := "identified"
		if want == "" {
			wantTerminal = "no-solution"
		}
		if want == baseline.Stage2Definition && wantTerminal == baseline.Terminal {
			continue
		}
		report, runErr := runPolicy(domainsDir, changed, LFFDirect)
		return runErr == nil && report.Stage2ProfileKey != baseline.Stage2ProfileKey && report.Stage2Definition == want && report.Terminal == wantTerminal && slices.Equal(report.Stage2ExactTies, ties)
	}
	return false
}

func reverseEdges(edges []Edge) {
	for left, right := 0, len(edges)-1; left < right; left, right = left+1, right-1 {
		edges[left], edges[right] = edges[right], edges[left]
	}
}

func flipExample(store *unit.Store, experiment string, polarity bool) error {
	descriptor := store.Get(experiment)
	corpus := store.Get(descriptor.GetString("stage2CorpusUnit"))
	if corpus == nil {
		return fmt.Errorf("stage-2 corpus missing")
	}
	records := append([]string(nil), corpus.GetStrings("examples")...)
	suffix := ":" + strconv.FormatBool(polarity)
	for index, record := range records {
		if strings.HasSuffix(record, suffix) {
			records[index] = strings.TrimSuffix(record, suffix) + ":" + strconv.FormatBool(!polarity)
			corpus.Set("examples", records)
			return nil
		}
	}
	return fmt.Errorf("stage-2 example with polarity %t missing", polarity)
}

func deterministicReportEncoding(firstFixture, secondFixture FixtureReport) bool {
	firstReport := Report{ReportVersion: "rule-induction-report/v1", Manifest: PreregisteredManifest(), ImplementationCommit: "test", PlanCommit: planCommit, Panel: "development", Status: "invalid", Policies: []PolicyReport{{Name: SharedLibrary, Fixtures: []FixtureReport{firstFixture}, Cohorts: []AggregateReport{}}}, Contrasts: []ContrastReport{}, Limitations: []string{}}
	secondReport := firstReport
	secondReport.Policies = []PolicyReport{{Name: SharedLibrary, Fixtures: []FixtureReport{secondFixture}, Cohorts: []AggregateReport{}}}
	first, firstErr := firstReport.JSON()
	second, secondErr := secondReport.JSON()
	return firstErr == nil && secondErr == nil && string(first) == string(second) && len(first) <= firstReport.Manifest.ReportByteCap
}
