package actionrelationscore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationutility"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type CurriculumEvidence struct {
	Curriculum         int
	NousPreboundary    actionrelationexp.ObjectBundle
	NoGuardPreboundary actionrelationexp.ObjectBundle
	Utility            actionrelationexp.ObjectBundle
	Authority          actionrelationexp.ObjectBundle
	StructuralMap      actionrelationexp.StructuralOutputMap
	RunEvidence        []actionrelationexp.RunEvidenceRecord
	Transcripts        map[string]actionrelationexp.TranscriptBundle
}

func BuildCurriculumEvidence(generated actionrelationfixture.GeneratedAttempt, scored CurriculumResult) (CurriculumEvidence, error) {
	policyResult := scored
	policyResult.WorldRows = nil
	policyResult.CurriculumRows = nil
	result, err := BuildPolicyCurriculumEvidence(policyResult)
	if err != nil {
		return CurriculumEvidence{}, err
	}
	if generated.Context.Panel != scored.Panel || generated.Context.Authority != scored.Authority || generated.Context.Curriculum != scored.Curriculum || generated.Curriculum.Family != scored.Family || len(scored.WorldRows) != 42 || len(scored.CurriculumRows) != 7 {
		return result, fmt.Errorf("scoring and fixture authority differ")
	}
	authorityRecords, err := curriculumAuthorityObjects(generated, scored)
	if err != nil {
		return result, err
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(scored.Panel)
	result.Authority, err = actionrelationexp.BuildObjectBundleAt(evidenceRoot, actionrelationexp.ObjectScope{Curriculum: scored.Curriculum, Class: "authority"}, authorityRecords)
	if err != nil {
		return result, fmt.Errorf("authority object bundle: %w", err)
	}
	var tables []actionrelationexp.TableBundle
	for _, acquisition := range []Acquisition{scored.Nous, scored.NoGuard} {
		kinds := []uint16{101, 102, 103, 104, 105, 106, 107, 108}
		if acquisition.Boundary.Scope == "no-guard" {
			kinds = []uint16{102, 103, 105, 106, 107, 108}
		}
		for _, kind := range kinds {
			tables = append(tables, acquisition.Evidence.Tables[kind])
		}
	}
	if err := actionrelationexp.VerifyCurriculumReplay(actionrelationexp.CurriculumReplay{
		Panel: scored.Panel, Authority: scored.Authority, Curriculum: scored.Curriculum,
		Objects: []actionrelationexp.ObjectBundle{result.NousPreboundary, result.NoGuardPreboundary, result.Utility, result.Authority},
		Tables:  tables, StructuralMap: result.StructuralMap, RunEvidence: result.RunEvidence, Transcripts: result.Transcripts,
	}); err != nil {
		return result, fmt.Errorf("curriculum evidence replay: %w", err)
	}
	return result, nil
}

// BuildPolicyCurriculumEvidence closes only policy-produced evidence. It is
// safe to call inside the public worker because scorer truth, score rows,
// generator ledgers, and fixture preimages are added by the supervisor later.
func BuildPolicyCurriculumEvidence(scored CurriculumResult) (CurriculumEvidence, error) {
	result := CurriculumEvidence{Curriculum: scored.Curriculum, Transcripts: map[string]actionrelationexp.TranscriptBundle{}}
	if scored.Panel == "" || scored.Authority == "" || scored.Curriculum < 0 || len(scored.WorldRows) != 0 || len(scored.CurriculumRows) != 0 {
		return result, fmt.Errorf("invalid unscored policy evidence")
	}
	if scored.Nous.Boundary.Verify(scored.Nous.Evidence) != nil || scored.NoGuard.Boundary.Verify(scored.NoGuard.Evidence) != nil {
		return result, fmt.Errorf("invalid acquisition preboundary")
	}
	evidenceRoot, err := actionrelationexp.EvidenceRoot(scored.Panel)
	if err != nil {
		return result, err
	}
	result.NousPreboundary, result.NoGuardPreboundary = scored.Nous.Boundary.Preboundary, scored.NoGuard.Boundary.Preboundary

	runIDs := []string{scored.Nous.Evidence.Transcript.RunID, scored.NoGuard.Evidence.Transcript.RunID}
	attributions := map[string]actionrelationexp.StructuralAttribution{}
	addAttribution := func(runID string, object actionrelationexp.ObjectRecord) error {
		if actionrelationexp.ValidateObject(object.Kind, object.Bytes) != nil {
			return fmt.Errorf("invalid structural attribution kind %d", object.Kind)
		}
		digest := objectDigest(object.Bytes)
		key := fmt.Sprintf("%05d:%s", object.Kind, digest)
		row := attributions[key]
		row.Kind, row.Digest = object.Kind, digest
		if !slices.Contains(row.RunIDs, runID) {
			row.RunIDs = append(row.RunIDs, runID)
		}
		attributions[key] = row
		return nil
	}
	for _, acquisition := range []Acquisition{scored.Nous, scored.NoGuard} {
		for _, object := range acquisitionStructuralObjects(acquisition) {
			if err := addAttribution(acquisition.Evidence.Transcript.RunID, object); err != nil {
				return result, err
			}
		}
		result.Transcripts[acquisition.Evidence.Transcript.RunID] = acquisition.Evidence.Transcript.Transcript
	}
	var utilityRecords []actionrelationexp.ObjectRecord
	for _, policy := range Policies {
		runs := scored.Runs[policy]
		if len(runs) != 6 {
			return result, fmt.Errorf("policy %s lacks six evidence runs", policy)
		}
		for _, run := range runs {
			if actionrelationutility.VerifySearchRun(run) != nil {
				return result, fmt.Errorf("invalid utility run %s", run.RunID)
			}
			runIDs = append(runIDs, run.RunID)
			result.Transcripts[run.RunID] = run.Transcript
			for _, object := range run.StructuralObjects {
				if err := addAttribution(run.RunID, object); err != nil {
					return result, err
				}
				utilityRecords = append(utilityRecords, object)
			}
			charged, err := utilityChargedObjects(run)
			if err != nil {
				return result, err
			}
			utilityRecords = append(utilityRecords, charged...)
		}
	}
	slices.Sort(runIDs)
	if len(runIDs) != 44 {
		return result, fmt.Errorf("curriculum does not have 44 run IDs")
	}
	attributionRows := make([]actionrelationexp.StructuralAttribution, 0, len(attributions))
	for _, row := range attributions {
		slices.Sort(row.RunIDs)
		attributionRows = append(attributionRows, row)
	}
	structuralMap, err := actionrelationexp.BuildStructuralOutputMapAt(evidenceRoot, scored.Curriculum, runIDs, attributionRows)
	if err != nil {
		return result, err
	}
	result.StructuralMap = structuralMap

	utilityRecords = uniqueObjectRecords(utilityRecords)
	result.Utility, err = actionrelationexp.BuildObjectBundleAt(evidenceRoot, actionrelationexp.ObjectScope{Curriculum: scored.Curriculum, Class: "utility"}, utilityRecords)
	if err != nil {
		return result, fmt.Errorf("utility object bundle: %w", err)
	}
	for _, acquisition := range []Acquisition{scored.Nous, scored.NoGuard} {
		record, err := acquisitionRunEvidence(acquisition, structuralMap.RunRoots[acquisition.Evidence.Transcript.RunID])
		if err != nil {
			return result, err
		}
		result.RunEvidence = append(result.RunEvidence, record)
	}
	for _, policy := range Policies {
		for _, run := range scored.Runs[policy] {
			record, err := utilityRunEvidence(run, structuralMap.RunRoots[run.RunID])
			if err != nil {
				return result, err
			}
			result.RunEvidence = append(result.RunEvidence, record)
		}
	}
	slices.SortFunc(result.RunEvidence, func(a, b actionrelationexp.RunEvidenceRecord) int { return compareText(a.RunID, b.RunID) })
	return result, nil
}

func acquisitionStructuralObjects(acquisition Acquisition) []actionrelationexp.ObjectRecord {
	run := acquisition.Evidence.Run
	if run.Store == nil {
		return nil
	}
	experiment := run.Store.Get(run.Experiment)
	var result []actionrelationexp.ObjectRecord
	add := func(kind uint16, name string) {
		if value := run.Store.Get(name); value != nil {
			result = append(result, actionrelationexp.ObjectRecord{Kind: kind, Bytes: []byte(value.GetString("canonicalObject"))})
		}
	}
	if experiment != nil {
		add(28, experiment.GetString("guardSearchBarrier"))
		add(11, experiment.GetString("trainingEvidenceUnit"))
		for _, name := range experiment.GetStrings("presentationViewUnits") {
			add(12, name)
		}
		for _, name := range experiment.GetStrings("normalizationProofUnits") {
			add(13, name)
		}
		artifact := run.Store.Get(run.Artifact)
		if artifact != nil {
			for _, name := range artifact.GetStrings("relationUnits") {
				add(9, name)
			}
		}
	}
	for _, root := range append(slices.Clone(acquisition.Evidence.Transcript.ObservationRoots), acquisition.Evidence.Transcript.RunRoot) {
		result = append(result, actionrelationexp.ObjectRecord{Kind: 46, Bytes: slices.Clone(root.Canonical)})
	}
	return uniqueObjectRecords(result)
}

func utilityChargedObjects(run actionrelationutility.SearchRun) ([]actionrelationexp.ObjectRecord, error) {
	var result []actionrelationexp.ObjectRecord
	for _, record := range run.Records {
		for _, output := range record.Outputs {
			kind := kindForCanonical(output)
			if kind == 0 {
				return nil, fmt.Errorf("run %s has untyped charged output", run.RunID)
			}
			// Artifact loads resolve to the acquisition preboundary and must not
			// duplicate that object into the utility scope.
			if kind != 10 {
				result = append(result, actionrelationexp.ObjectRecord{Kind: kind, Bytes: slices.Clone(output)})
			}
		}
		reservation := storeUnitByDigest(run.Store, record.SourceTaskDigest)
		if reservation == nil || !run.Store.IsA(reservation.Name, "CompoundWorkReservation") {
			return nil, fmt.Errorf("run %s lacks reservation preimage", run.RunID)
		}
		result = append(result, actionrelationexp.ObjectRecord{Kind: 27, Bytes: []byte(reservation.GetString("canonicalObject"))})
	}
	if run.Terminal == "budget-exhausted" {
		rejected := run.WorkTerminal.RejectedReservation
		if rejected.Digest == "" || rejected.Digest != objectDigest(rejected.Canonical) {
			return nil, fmt.Errorf("run %s lacks rejected reservation preimage", run.RunID)
		}
		result = append(result, actionrelationexp.ObjectRecord{Kind: 27, Bytes: slices.Clone(rejected.Canonical)})
	}
	return uniqueObjectRecords(result), nil
}

func curriculumAuthorityObjects(generated actionrelationfixture.GeneratedAttempt, scored CurriculumResult) ([]actionrelationexp.ObjectRecord, error) {
	return curriculumAuthorityObjectsDelayed(generated, scored, scored.Nous.Boundary.Canonical, scored.NoGuard.Boundary.Canonical)
}

func curriculumAuthorityObjectsDelayed(generated actionrelationfixture.GeneratedAttempt, scored CurriculumResult, nousBoundary, noGuardBoundary []byte) ([]actionrelationexp.ObjectRecord, error) {
	result := []actionrelationexp.ObjectRecord{
		{Kind: 35, Bytes: slices.Clone(nousBoundary)},
		{Kind: 35, Bytes: slices.Clone(noGuardBoundary)},
		{Kind: 47, Bytes: slices.Clone(generated.Fixture.Canonical)},
	}
	for _, ledger := range generated.AttemptLedgers {
		result = append(result, actionrelationexp.ObjectRecord{Kind: 36, Bytes: slices.Clone(ledger.Canonical)})
	}
	for _, truth := range generated.Truth.Worlds {
		for _, shard := range truth.Shards {
			result = append(result, actionrelationexp.ObjectRecord{Kind: 29, Bytes: slices.Clone(shard.Canonical)})
		}
	}
	for _, row := range scored.WorldRows {
		result = append(result, actionrelationexp.ObjectRecord{Kind: 32, Bytes: slices.Clone(row.Canonical)})
	}
	for _, row := range scored.CurriculumRows {
		result = append(result, actionrelationexp.ObjectRecord{Kind: 33, Bytes: slices.Clone(row.Canonical)})
		result = append(result, actionrelationexp.ObjectRecord{Kind: 46, Bytes: slices.Clone(row.OperationRoot.Canonical)})
	}
	for _, view := range generated.Curriculum.Worlds {
		normalized, err := (actionrelations.World{State: view.State, Actions: view.Actions}).Normalize()
		if err != nil {
			return nil, err
		}
		state, _ := normalized.State.CanonicalJSON()
		world, _ := normalized.CanonicalJSON()
		result = append(result, actionrelationexp.ObjectRecord{Kind: 1, Bytes: state}, actionrelationexp.ObjectRecord{Kind: 4, Bytes: world})
		for _, action := range normalized.Actions {
			canonical, _ := action.CanonicalJSON()
			result = append(result, actionrelationexp.ObjectRecord{Kind: 2, Bytes: canonical})
		}
		for _, occurrence := range normalized.Occurrences {
			canonical, _ := occurrence.CanonicalJSON()
			result = append(result, actionrelationexp.ObjectRecord{Kind: 3, Bytes: canonical})
		}
	}
	return uniqueObjectRecords(result), nil
}

func acquisitionRunEvidence(acquisition Acquisition, structuralRoot string) (actionrelationexp.RunEvidenceRecord, error) {
	transcript := acquisition.Evidence.Transcript.Transcript
	journal, err := transcript.JournalRoot.Digest()
	if err != nil {
		return actionrelationexp.RunEvidenceRecord{}, err
	}
	input, _ := transcript.InputRoot.Digest()
	detail, _ := transcript.DetailRoot.Digest()
	charged, err := actionrelationexp.ChargedOutputsRoot(transcript)
	if err != nil {
		return actionrelationexp.RunEvidenceRecord{}, err
	}
	return actionrelationexp.RunEvidenceRecord{RunID: transcript.RunID, JournalRoot: journal, InputRoot: input, DetailRoot: detail, OperationRoot: acquisition.Evidence.Transcript.RunRoot.Digest, ChargedRoot: charged, StructuralRoot: structuralRoot}, nil
}

func utilityRunEvidence(run actionrelationutility.SearchRun, structuralRoot string) (actionrelationexp.RunEvidenceRecord, error) {
	journal, err := run.Transcript.JournalRoot.Digest()
	if err != nil {
		return actionrelationexp.RunEvidenceRecord{}, err
	}
	input, _ := run.Transcript.InputRoot.Digest()
	detail, _ := run.Transcript.DetailRoot.Digest()
	charged, err := actionrelationexp.ChargedOutputsRoot(run.Transcript)
	if err != nil {
		return actionrelationexp.RunEvidenceRecord{}, err
	}
	return actionrelationexp.RunEvidenceRecord{RunID: run.RunID, JournalRoot: journal, InputRoot: input, DetailRoot: detail, OperationRoot: run.RunRoot.Digest, ChargedRoot: charged, StructuralRoot: structuralRoot, WorkTerminal: run.WorkTerminal.Digest}, nil
}

func kindForCanonical(canonical []byte) uint16 {
	var row []json.RawMessage
	var tag string
	if json.Unmarshal(canonical, &row) != nil || len(row) == 0 || json.Unmarshal(row[0], &tag) != nil {
		return 0
	}
	return map[string]uint16{
		"finite-action-state/v1": 1, "finite-action-semantic/v1": 2, "action-occurrence/v1": 3,
		"guarded-action-artifact/v1": 10, "action-local-facts/v1": 8, "local-diamond-certificate/v1": 17,
		"sleep-propagation-core/v1": 18, "sleep-search-node/v1": 20, "action-terminal/v1": 23,
		"certificate-cache-row/v3": 26, "action-applicability-row/v1": 38, "action-transition-row/v1": 39,
		"action-state-equality-row/v1": 40, "action-literal-evaluation-row/v1": 41, "action-relation-match-row/v1": 42,
		"action-static-footprint-row/v1": 48, "action-work-terminal/v1": 49,
	}[tag]
}

func uniqueObjectRecords(records []actionrelationexp.ObjectRecord) []actionrelationexp.ObjectRecord {
	seen := map[string]actionrelationexp.ObjectRecord{}
	for _, record := range records {
		key := objectDigest(record.Bytes)
		if prior, ok := seen[key]; ok && (prior.Kind != record.Kind || !slices.Equal(prior.Bytes, record.Bytes)) {
			panic("action-relation object digest collision")
		}
		seen[key] = record
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]actionrelationexp.ObjectRecord, len(keys))
	for index, key := range keys {
		result[index] = seen[key]
	}
	return result
}

func storeUnitByDigest(store *unit.Store, digest string) *unit.Unit {
	if store == nil {
		return nil
	}
	for _, name := range store.All() {
		value := store.Get(name)
		if value != nil && value.GetString("objectDigest") == digest {
			return value
		}
	}
	return nil
}

func objectDigest(canonical []byte) string {
	value := sha256.Sum256(canonical)
	return hex.EncodeToString(value[:])
}

func compareText(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
