package actionrelationacquire

import (
	"testing"

	"github.com/chazu/nous/internal/actionrelationwire"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestCUEAcquisitionEvidenceCardinalities(t *testing.T) {
	run, err := Execute("../../domains", "test")
	if err != nil {
		t.Fatal(err)
	}
	if run.Observations != 16 || run.Candidates != 451 || run.Edges != 450 || run.LiteralRows != 13920 || run.GuardResults != 7216 || run.CandidateResults != 451 || run.Winners < 1 || run.Artifact == "" {
		t.Fatalf("run=%+v", run)
	}
	codes := map[uint16]int{}
	for _, record := range run.MeterRecords {
		codes[record.Code]++
	}
	if codes[1] != 1 || codes[2] != 450 || codes[3] != 450 || codes[7] != 13920 || codes[8] != 1 || codes[20] != 451 || codes[22] != 7216 {
		t.Fatalf("meter code counts=%v", codes)
	}
	experiment := run.Store.Get(run.Experiment)
	if len(experiment.GetStrings("presentationViewUnits")) != 32 || len(experiment.GetStrings("normalizationProofUnits")) != 32 || len(experiment.GetStrings("viewEvidenceUnits")) != 32 {
		t.Fatal("acquisition did not retain the two presentation lineages per observation")
	}
	observationDigests := make([]string, 16)
	for index, name := range experiment.GetStrings("observationUnits") {
		observationDigests[index] = run.Store.Get(name).GetString("objectDigest")
	}
	semanticRoot, _ := actionrelationwire.RootDigest("semantic-training", observationDigests)
	if experiment.GetString("semanticTrainingRoot") != semanticRoot {
		t.Fatal("semantic training root does not commit the ordered observation cores")
	}
	viewRows := make([]any, 0, 32)
	for _, name := range experiment.GetStrings("viewEvidenceUnits") {
		view := run.Store.Get(name)
		viewRows = append(viewRows, []any{view.GetString("observationDigest"), view.GetInt("bank"), view.GetString("objectDigest")})
	}
	viewRoot, _ := actionrelationwire.RootDigest("view-evidence", viewRows)
	if experiment.GetString("viewEvidenceRoot") != viewRoot {
		t.Fatal("view root does not commit ordered observation/bank/evidence rows")
	}
	winner := run.Store.Get(experiment.GetStrings("winnerResultUnits")[0])
	candidate := run.Store.Get(winner.GetString("candidate"))
	guard, err := actionrelations.ParseGuard([]byte(candidate.GetString("guard")))
	// The alias pattern already fixes the two add roles as distinct, so the
	// minimum-literal representative is extensionally the empty guard.
	if err != nil || len(guard.Literals) != 0 {
		t.Fatalf("unexpected disjoint-add winner guard=%#v err=%v", guard, err)
	}
	artifactUnit := run.Store.Get(run.Artifact)
	artifact, err := actionrelations.ParseArtifact([]byte(artifactUnit.GetString("artifact")))
	if err != nil {
		t.Fatal(err)
	}
	resolved := map[string]actionrelations.Relation{}
	for _, name := range artifactUnit.GetStrings("relationUnits") {
		relation, err := actionrelations.ParseRelation([]byte(run.Store.Get(name).GetString("relation")))
		if err != nil {
			t.Fatal(err)
		}
		digest, _ := relation.Digest()
		resolved[digest] = relation
	}
	if err := artifact.ValidateResolved(resolved); err != nil {
		t.Fatalf("resolved artifact: %v", err)
	}
}
