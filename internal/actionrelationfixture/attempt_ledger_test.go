package actionrelationfixture

import "testing"

func TestAttemptLedgerRecomputesDrawsAndClosesPhaseAuthority(t *testing.T) {
	context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: 3, CurriculumSeed: 851004, Attempt: 0}
	draws, err := precommitDraws(context)
	if err != nil {
		t.Fatal(err)
	}
	phases := []GeneratorPhase{{Name: "draw-precommit", StartWork: 0, EndWork: 66, Predicate: "exact-66-draws", Status: "passed"}}
	work := 66
	for _, vocabulary := range generatorPhaseVocabulary[1:] {
		phases = append(phases, GeneratorPhase{Name: vocabulary.name, StartWork: work, EndWork: work + 1, Predicate: vocabulary.predicate, Status: "passed"})
		work++
	}
	ledger, err := sealAttemptLedger(context, draws, phases, "accepted")
	if err != nil {
		t.Fatal(err)
	}
	if ledger.TotalWork != 73 || ledger.Digest == "" || len(ledger.Canonical) > 65536 {
		t.Fatalf("ledger work=%d digest=%q bytes=%d", ledger.TotalWork, ledger.Digest, len(ledger.Canonical))
	}

	forged := ledger
	forged.Draws.Draws = append([]Draw(nil), ledger.Draws.Draws...)
	forged.Draws.Draws[7].U64++
	if err := VerifyAttemptLedger(forged); err == nil {
		t.Fatal("accepted a forged retained draw")
	}

	gapped := ledger
	gapped.Phases = append([]GeneratorPhase(nil), ledger.Phases...)
	gapped.Phases[3].StartWork++
	if err := VerifyAttemptLedger(gapped); err == nil {
		t.Fatal("accepted a gap in generator work")
	}
}

func TestAttemptLedgerStopsAtFirstFailedPredicate(t *testing.T) {
	context := DrawContext{Panel: "validation", Authority: "validation-public-v1", Curriculum: 2, CurriculumSeed: 852003, Attempt: 4}
	draws, err := precommitDraws(context)
	if err != nil {
		t.Fatal(err)
	}
	phases := []GeneratorPhase{
		{Name: "draw-precommit", StartWork: 0, EndWork: 66, Predicate: "exact-66-draws", Status: "passed"},
		{Name: "family-universe", StartWork: 66, EndWork: 125, Predicate: "complete-family-universe", Status: "failed"},
	}
	ledger, err := sealAttemptLedger(context, draws, phases, "rejected")
	if err != nil {
		t.Fatal(err)
	}
	if ledger.TotalWork != 125 {
		t.Fatalf("work=%d", ledger.TotalWork)
	}
	ledger.Phases = append(ledger.Phases, GeneratorPhase{Name: "identifiability", StartWork: 125, EndWork: 126, Predicate: "identifiable", Status: "passed"})
	if err := VerifyAttemptLedger(ledger); err == nil {
		t.Fatal("accepted work after a failed predicate")
	}
}
