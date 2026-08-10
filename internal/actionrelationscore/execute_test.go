package actionrelationscore

import (
	"slices"
	"testing"

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
