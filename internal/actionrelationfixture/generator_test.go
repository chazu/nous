package actionrelationfixture

import (
	"slices"
	"testing"
)

func TestMeasuredGeneratorProducesAcceptedLedgerBeforeFixtureAssembly(t *testing.T) {
	context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: 7, CurriculumSeed: 851008, Attempt: 0}
	generated, err := GenerateAttempt(context, nil)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Ledger.Terminal != "accepted" || len(generated.Ledger.Phases) != len(generatorPhaseVocabulary) || generated.Ledger.TotalWork <= 66 || generated.Fixture.Digest == "" || generated.Fixture.AttemptLedgers[0] != generated.Ledger.Digest {
		t.Fatalf("ledger work=%d terminal=%q phases=%d fixture=%q", generated.Ledger.TotalWork, generated.Ledger.Terminal, len(generated.Ledger.Phases), generated.Fixture.Digest)
	}
	for index, phase := range generated.Ledger.Phases {
		if phase.Status != "passed" || index > 0 && phase.EndWork <= phase.StartWork {
			t.Fatalf("phase %d=%+v", index, phase)
		}
	}
	again, err := GenerateAttempt(context, nil)
	if err != nil || generated.Ledger.Digest != again.Ledger.Digest || generated.Fixture.Digest != again.Fixture.Digest || !slices.Equal(generated.TrainingAuthority.CoreDigests, again.TrainingAuthority.CoreDigests) {
		t.Fatal("measured generator is not deterministic")
	}
}

func TestMeasuredGeneratorAcceptsEveryFrozenFamilyWithinPhysicalCap(t *testing.T) {
	for family := 0; family < 8; family++ {
		context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: family, CurriculumSeed: 851001 + family, Attempt: 0}
		generated, err := GenerateAttempt(context, nil)
		if err != nil {
			t.Fatalf("family %d: %v", family, err)
		}
		if generated.Curriculum.Family != family || generated.Ledger.TotalWork >= GeneratorAttemptWorkCap || generated.Fixture.Family != family {
			t.Fatalf("family %d work=%d fixture-family=%d", family, generated.Ledger.TotalWork, generated.Fixture.Family)
		}
		t.Logf("family %d generator work %d", family, generated.Ledger.TotalWork)
	}
}
