package actionrelationcompetence

import "testing"

func TestSafeCompetenceUniverse(t *testing.T) {
	report, err := Run()
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.Sequences != 40320 || report.Steps != 322560 {
		t.Fatalf("report=%+v", report)
	}
}

func TestSequenceCompetenceRetainsAllCaseAndResultRows(t *testing.T) {
	report, evidence, err := RunSequenceEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || len(evidence.Cases) != MaximumSequences || len(evidence.Results) != MaximumSequences || VerifyEvidence(evidence) != nil {
		t.Fatalf("report=%+v cases=%d results=%d", report, len(evidence.Cases), len(evidence.Results))
	}
}

func TestFullCompetenceEvidenceRetainsAllNormalizedGuardTruth(t *testing.T) {
	report, evidence, err := RunEvidence()
	if err != nil {
		t.Fatal(err)
	}
	// 4 one-cell states * 15 actions + 16 two-cell states * 35 actions +
	// 64 three-cell states * 61 actions.
	wantTransitions := 4*15 + 16*35 + 64*61
	wantDiamonds := 61 * 61
	wantSearches := 8 * 3
	want := MaximumSequences + 16*451 + wantDiamonds + wantSearches + wantTransitions
	if !report.Passed || len(evidence.Cases) != want || len(evidence.Results) != want || VerifyEvidence(evidence) != nil {
		t.Fatalf("report=%+v cases=%d want=%d", report, len(evidence.Cases), want)
	}
}
