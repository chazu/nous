package actionrelationscore

import (
	"slices"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationsearch"
)

func TestDevelopmentCurriculumExecutesExactAcquisitionAndSixWorldPolicyRows(t *testing.T) {
	generated, err := actionrelationfixture.GenerateAttempt(actionrelationfixture.DrawContext{
		Panel: "development", Authority: "development-public-v1", Curriculum: 0, CurriculumSeed: 851001, Attempt: 0,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ExecuteCurriculum("../../domains", generated)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.WorldRows) != 42 || len(result.CurriculumRows) != 7 || len(result.Runs) != 7 {
		t.Fatalf("scorer cardinalities worlds=%d curricula=%d runs=%d", len(result.WorldRows), len(result.CurriculumRows), len(result.Runs))
	}
	for ordinal, row := range result.WorldRows {
		if row.WorldOrdinal != ordinal/7 || row.Policy != Policies[ordinal%7] || !row.BehaviorEqual || VerifyWorldPolicyRow(row) != nil {
			t.Fatalf("world row %d is not canonical panel order: %+v", ordinal, row)
		}
	}
	nous := result.CurriculumRows[policyIndex(actionrelationsearch.NousSleep)]
	control := result.CurriculumRows[policyIndex(actionrelationsearch.LearnedNoUse)]
	noGuard := result.CurriculumRows[policyIndex(actionrelationsearch.NoGuardSleep)]
	if nous.AcquisitionTerminal != "completed" || nous.AcquisitionWorkVector != control.AcquisitionWorkVector || nous.ArtifactDigest != control.ArtifactDigest || nous.AcquisitionWorkVector == noGuard.AcquisitionWorkVector {
		t.Fatalf("acquisition charging changed nous=%+v control=%+v no-guard=%+v", nous.AcquisitionWorkVector, control.AcquisitionWorkVector, noGuard.AcquisitionWorkVector)
	}
	for _, row := range result.CurriculumRows {
		if row.AggregateTerminal != "completed" || !row.BehaviorEqual || VerifyCurriculumPolicyRow(row) != nil || !slices.Equal(row.WorldRowDigests, policyWorldDigests(result.WorldRows, row.Policy)) {
			t.Fatalf("invalid curriculum row for %s", row.Policy)
		}
	}
	for _, row := range result.WorldRows {
		if row.Policy == actionrelationsearch.NousSleep && row.MatchCounts.UtilityFalseMatches != 0 {
			t.Fatalf("Nous false relation match in world %d", row.WorldOrdinal)
		}
		if !freshCertificateCounts(row.Policy, row) {
			t.Fatalf("certificate counts do not reconstruct for %s world %d: %+v sleeps=%d", row.Policy, row.WorldOrdinal, row.CertificateCounts, row.SleepCount)
		}
	}
	byPolicy := map[actionrelationsearch.Policy]CurriculumPolicyRow{}
	for _, row := range result.CurriculumRows {
		byPolicy[row.Policy] = row
	}
	if !immutableAcquisitionRows(byPolicy) {
		t.Fatal("post-freeze acquisition artifact identity does not reconstruct")
	}
	evidence, err := BuildCurriculumEvidence(generated, result)
	if err != nil {
		t.Fatal(err)
	}
	for scope, bundle := range map[string]actionrelationexp.ObjectBundle{
		"nous": evidence.NousPreboundary, "no-guard": evidence.NoGuardPreboundary,
		"utility": evidence.Utility, "authority": evidence.Authority,
	} {
		if err := actionrelationexp.VerifyObjectBundle(bundle); err != nil {
			t.Fatalf("%s object scope: %v", scope, err)
		}
	}
	if err := actionrelationexp.VerifyStructuralOutputMap(evidence.StructuralMap); err != nil {
		t.Fatal(err)
	}
	if len(evidence.RunEvidence) != 44 || len(evidence.Transcripts) != 44 {
		t.Fatalf("evidence cardinalities runs=%d transcripts=%d", len(evidence.RunEvidence), len(evidence.Transcripts))
	}
	wantRoot, _ := actionrelationexp.EvidenceRoot("development")
	for _, path := range []string{
		evidence.NousPreboundary.ObjectFiles[0].Path,
		result.Nous.Evidence.Tables[101].Files[0].Path,
		evidence.Transcripts[result.Nous.Evidence.Transcript.RunID].JournalFiles[0].Path,
		evidence.StructuralMap.File.Path,
	} {
		if !strings.HasPrefix(path, wantRoot+"/") {
			t.Fatalf("evidence path %q is outside canonical panel root %q", path, wantRoot)
		}
	}
	manifests, err := BuildCurriculumManifests(result, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests.ManifestFiles) != 155 || len(manifests.JournalRoots) != 44 || len(manifests.Tables) != 14 {
		t.Fatalf("manifest cardinalities files=%d journals=%d tables=%d", len(manifests.ManifestFiles), len(manifests.JournalRoots), len(manifests.Tables))
	}
	for name, mutate := range map[string]func(*WorldPolicyRow){
		"match-count":       func(row *WorldPolicyRow) { row.MatchCounts.UtilityAttempts++ },
		"certificate-count": func(row *WorldPolicyRow) { row.CertificateCounts.Attempted++ },
		"sleep-count":       func(row *WorldPolicyRow) { row.SleepCount++ },
		"behavior-equality": func(row *WorldPolicyRow) { row.BehaviorEqual = false },
	} {
		t.Run("retained-replay-rejects-summary-"+name, func(t *testing.T) {
			mutated := result
			mutated.WorldRows = slices.Clone(result.WorldRows)
			mutated.CurriculumRows = slices.Clone(result.CurriculumRows)
			row := mutated.WorldRows[0]
			mutate(&row)
			row, err = BuildWorldPolicyRow(row)
			if err != nil {
				t.Fatal(err)
			}
			mutated.WorldRows[0] = row
			curriculum := mutated.CurriculumRows[0]
			curriculum.WorldRowDigests = slices.Clone(curriculum.WorldRowDigests)
			curriculum.WorldRowDigests[0] = row.Digest
			if name == "behavior-equality" {
				curriculum.BehaviorEqual = false
			}
			curriculum, err = BuildCurriculumPolicyRow(curriculum)
			if err != nil {
				t.Fatal(err)
			}
			mutated.CurriculumRows[0] = curriculum
			if _, err := BuildCurriculumEvidence(generated, mutated); err == nil {
				t.Fatal("accepted worker-authored scorer field without retained reconstruction")
			}
		})
	}
}

func TestExportedCurriculumExecutorRejectsProtectedPanelsBeforeConstruction(t *testing.T) {
	_, err := ExecuteCurriculum("../../domains", actionrelationfixture.GeneratedAttempt{Context: actionrelationfixture.DrawContext{Panel: "validation", Authority: "validation-public-v1"}})
	if err == nil {
		t.Fatal("development helper accepted protected panel work")
	}
}

func policyWorldDigests(rows []WorldPolicyRow, policy actionrelationsearch.Policy) []string {
	var result []string
	for _, row := range rows {
		if row.Policy == policy {
			result = append(result, row.Digest)
		}
	}
	return result
}
