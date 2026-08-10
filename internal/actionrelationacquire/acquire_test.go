package actionrelationacquire

import (
	"testing"

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
	experiment := run.Store.Get(run.Experiment)
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
