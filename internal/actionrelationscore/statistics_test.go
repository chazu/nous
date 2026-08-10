package actionrelationscore

import "testing"

func TestStatDrawUsesExactCanonicalSHAAndMultiplyHighPick(t *testing.T) {
	draw := statDraw("development", "development-public-v1", "bootstrap-family-row", []int{0, 0, 0})
	if draw != 488755831655298235 {
		t.Fatalf("draw = %d", draw)
	}
	if statPick(draw, 2) != 0 || statPick(^uint64(0), 7) != 6 {
		t.Fatal("multiply-high selection changed")
	}
}

func TestExactFamilyBootstrapRandomizationAndPowerSchedule(t *testing.T) {
	pairs := make([]pairedCurriculum, 16)
	for curriculum := range pairs {
		pairs[curriculum] = pairedCurriculum{
			curriculum: curriculum, family: curriculum % 8, mechanical: true,
			nous:    CurriculumPolicyRow{SearchTotal: 80, LifecycleTotal: 90, AcquisitionWorkVector: [12]int{10}},
			dynamic: CurriculumPolicyRow{SearchTotal: 100, LifecycleTotal: 100},
		}
	}
	inference, err := inferPairs("development", "development-public-v1", pairs, 100, 1, 98, "bootstrap-family-row", "randomization-swap", nil)
	if err != nil {
		t.Fatal(err)
	}
	if inference.PrimarySearchRatio != (Fraction{1280, 1600}) || inference.LifecycleRatio != (Fraction{1440, 1600}) || inference.ConfidenceInterval[0] != (Fraction{1280, 1600}) || inference.ConfidenceInterval[1] != (Fraction{1280, 1600}) || inference.SavingCoverage != (Fraction{16, 16}) || inference.RandomizationP.Denominator != 101 {
		t.Fatalf("unexpected exact inference: %+v", inference)
	}
	for _, row := range inference.AmortizationRows {
		if row.Infinite || row.Batches != 1 {
			t.Fatalf("amortization row = %+v", row)
		}
	}
	power, successes, err := estimatePower("development-public-v1", pairs, 5, 40)
	if err != nil {
		t.Fatal(err)
	}
	if power != (Fraction{5, 5}) || successes != 5 {
		t.Fatalf("power = %+v successes=%d", power, successes)
	}
}

func TestRationalComparisonUsesCrossProductsAndTieIdentity(t *testing.T) {
	if compareFraction(Fraction{2, 3}, Fraction{4, 6}) != 0 || compareFraction(Fraction{85, 100}, Fraction{17, 20}) != 0 || compareFraction(Fraction{849999, 1000000}, Fraction{85, 100}) >= 0 {
		t.Fatal("exact fraction comparison changed")
	}
}
