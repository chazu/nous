package actionrelationscore

import (
	"bytes"
	"testing"
)

func TestV3ReportUsesExactRefsRatiosAndStageClassification(t *testing.T) {
	ref := func(name string) AuthorityRef {
		return AuthorityRef{Path: ".nous/actionrelations-v1-development-evidence/authority/" + name + ".json", Digest: testDigest(name), Mode: "100644"}
	}
	refs := ReportAuthority{
		PlanReview: ref("plan"), ImplementationReview: ref("implementation"), BuildAuthority: ref("build"),
		Competence: ref("competence"), FixtureRoot: ref("fixture"), CurriculumRowsRoot: ref("rows"), EvidencePayload: ref("payload"),
	}
	gates := MechanicalGates{true, true, true, true, true, true, true, true}
	inference := Inference{
		PrimarySearchRatio: Fraction{1280, 1600}, LifecycleRatio: Fraction{1440, 1600},
		ConfidenceInterval: [2]Fraction{{75, 100}, {90, 100}}, RandomizationP: Fraction{1, 10001},
		RandomizationExtreme: 0, SavingCoverage: Fraction{16, 16}, Power: Fraction{1600, 2000}, PowerSuccesses: 1600,
	}
	for curriculum := 0; curriculum < 16; curriculum++ {
		inference.AmortizationRows = append(inference.AmortizationRows, AmortizationRow{Curriculum: curriculum, Acquisition: 10, DynamicSearch: 100, NousSearch: 80, Batches: 1})
	}
	report, err := BuildReport("development", "development-public-v1", refs, gates, inference)
	if err != nil {
		t.Fatal(err)
	}
	if report.Classification != "interim-power-authorized" || VerifyReport(report) != nil || !bytes.HasPrefix(report.Canonical, []byte(`["actionrelation-report/v3"`)) {
		t.Fatalf("report = %+v", report)
	}
	unauthorized := inference
	unauthorized.Power = Fraction{1599, 2000}
	unauthorized.PowerSuccesses = 1599
	report, err = BuildReport("development", "development-public-v1", refs, gates, unauthorized)
	if err != nil || report.Classification != "interim-power-unauthorized" {
		t.Fatalf("unauthorized report=%+v err=%v", report, err)
	}
	invalidGates := gates
	invalidGates.FreshCertificatesValid = false
	report, err = BuildReport("development", "development-public-v1", refs, invalidGates, inference)
	if err != nil || report.Classification != "invalid" {
		t.Fatalf("invalid report=%+v err=%v", report, err)
	}
}

func TestReportRejectsUnsafeRefsAndProtectedZeroRunningReceipt(t *testing.T) {
	if (AuthorityRef{Path: "../escape", Digest: testDigest("x"), Mode: "100644"}).Verify() == nil {
		t.Fatal("accepted traversal authority ref")
	}
	locked := Inference{
		PrimarySearchRatio: Fraction{2720, 3200}, LifecycleRatio: Fraction{3360, 3200}, ConfidenceInterval: [2]Fraction{{1, 2}, {99, 100}}, RandomizationP: Fraction{499, 10001}, RandomizationExtreme: 498, SavingCoverage: Fraction{32, 32}, Power: Fraction{4, 5},
	}
	for curriculum := 0; curriculum < 32; curriculum++ {
		locked.AmortizationRows = append(locked.AmortizationRows, AmortizationRow{Curriculum: curriculum, Acquisition: 20, DynamicSearch: 100, NousSearch: 85, Batches: 2})
	}
	classification, err := reportClassification("locked", MechanicalGates{true, true, true, true, true, true, true, true}, locked)
	if err != nil || classification != "valid-positive" {
		t.Fatalf("locked boundary classification=%q err=%v", classification, err)
	}
}
