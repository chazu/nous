package actionrelationfixture

import (
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationacquire"
)

func TestTrainingAuthoritySealsObservationAndViewWiresDeterministically(t *testing.T) {
	authority, err := SealTrainingAuthority(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(authority.CoreDigests) != 16 || len(authority.ViewEvidenceDigests) != 32 {
		t.Fatalf("cores=%d views=%d", len(authority.CoreDigests), len(authority.ViewEvidenceDigests))
	}
	again, err := SealTrainingAuthority(1)
	if err != nil || !slices.Equal(authority.CoreDigests, again.CoreDigests) || !slices.Equal(authority.ViewEvidenceDigests, again.ViewEvidenceDigests) {
		t.Fatal("training authority is not deterministic")
	}
}

func TestTrainingAuthorityMatchesProductionObservationAndViewWires(t *testing.T) {
	authority, err := SealTrainingAuthority(1)
	if err != nil {
		t.Fatal(err)
	}
	session, err := actionrelationacquire.BeginFamily("../../domains", "fixture-authority-wire", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Abort()
	run, err := session.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	experiment := run.Store.Get(run.Experiment)
	var cores, views []string
	for _, name := range experiment.GetStrings("observationUnits") {
		cores = append(cores, run.Store.Get(name).GetString("objectDigest"))
	}
	for _, name := range experiment.GetStrings("viewEvidenceUnits") {
		views = append(views, run.Store.Get(name).GetString("objectDigest"))
	}
	if !slices.Equal(authority.CoreDigests, cores) || !slices.Equal(authority.ViewEvidenceDigests, views) {
		t.Fatal("fixture authority does not reproduce production evidence wires")
	}
}

func TestCurriculumFixtureCommitsAcceptedAttemptAndFrozenOrders(t *testing.T) {
	context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: 9, CurriculumSeed: 851010, Attempt: 0}
	curriculum, err := BuildCurriculum(context)
	if err != nil {
		t.Fatal(err)
	}
	truth, err := SealCurriculumTruth(curriculum)
	if err != nil {
		t.Fatal(err)
	}
	draws, err := PrecommitDraws(context)
	if err != nil {
		t.Fatal(err)
	}
	phases := []GeneratorPhase{{Name: "draw-precommit", StartWork: 0, EndWork: 66, Predicate: "exact-66-draws", Status: "passed"}}
	work := 66
	for _, vocabulary := range generatorPhaseVocabulary[1:] {
		phases = append(phases, GeneratorPhase{Name: vocabulary.name, StartWork: work, EndWork: work + 1, Predicate: vocabulary.predicate, Status: "passed"})
		work++
	}
	ledger, err := SealAttemptLedger(context, draws, phases, "accepted")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := SealCurriculumFixture(context, curriculum, truth, []AttemptLedger{ledger})
	if err != nil {
		t.Fatal(err)
	}
	if fixture.Digest == "" || len(fixture.TrainingCoreDigests) != 16 || len(fixture.ViewEvidenceDigests) != 32 || len(fixture.WorldDigests) != 6 || len(fixture.AttemptLedgers) != 1 {
		t.Fatalf("fixture shape=%+v", fixture)
	}
	forged := fixture
	forged.WorldDigests = slices.Clone(fixture.WorldDigests)
	forged.WorldDigests[0], forged.WorldDigests[1] = forged.WorldDigests[1], forged.WorldDigests[0]
	if err := VerifyCurriculumFixture(forged); err == nil {
		t.Fatal("accepted reordered fixture worlds")
	}
}
