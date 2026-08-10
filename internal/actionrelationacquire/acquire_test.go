package actionrelationacquire

import (
	"fmt"
	"testing"

	"github.com/chazu/nous/internal/actionrelationfixturecore"
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

func TestCUEAcquisitionLearnsEveryFrozenFamilyGuard(t *testing.T) {
	wants := []string{"", "combined-adds-in-bounds", "argument-equal", "", "", "", "symbol-equal", "combined-adds-in-bounds"}
	for family, wantAtom := range wants {
		run, err := ExecuteFamily("../../domains", fmt.Sprintf("family-%d", family), family)
		if err != nil {
			t.Fatalf("family %d: %v", family, err)
		}
		experiment := run.Store.Get(run.Experiment)
		winners := experiment.GetStrings("winnerResultUnits")
		if len(winners) != 1 {
			var guards []string
			for _, winner := range winners {
				candidate := run.Store.Get(run.Store.Get(winner).GetString("candidate"))
				guards = append(guards, candidate.GetString("guard"))
			}
			t.Fatalf("family %d winners=%d guards=%v", family, len(winners), guards)
		}
		candidate := run.Store.Get(run.Store.Get(winners[0]).GetString("candidate"))
		guard, err := actionrelations.ParseGuard([]byte(candidate.GetString("guard")))
		if err != nil || wantAtom == "" && len(guard.Literals) != 0 || wantAtom != "" && (len(guard.Literals) != 1 || guard.Literals[0].Atom != wantAtom || !guard.Literals[0].Polarity) {
			t.Fatalf("family %d guard=%+v want atom %q err=%v", family, guard, wantAtom, err)
		}
		if err := actionrelationfixturecore.VerifyFamilyGuard(family, guard); err != nil {
			t.Fatalf("family %d extensional guard: %v", family, err)
		}
	}
}

func TestNoGuardAcquisitionExecutesOnlyRootSchedule(t *testing.T) {
	run, err := ExecuteNoGuard("../../domains", "no-guard", 0)
	if err != nil {
		t.Fatal(err)
	}
	if run.Observations != 16 || run.Candidates != 1 || run.Edges != 0 || run.LiteralRows != 0 || run.GuardResults != 16 || run.CandidateResults != 1 || run.Winners != 1 || run.Artifact == "" {
		t.Fatalf("run=%+v", run)
	}
	codes := map[uint16]int{}
	for _, record := range run.MeterRecords {
		codes[record.Code]++
	}
	training := codes[4] + codes[5] + codes[6]
	if training != 130 || len(run.MeterRecords) != training+19 || codes[1] != 1 || codes[22] != 16 || codes[20] != 1 || codes[8] != 1 || codes[2] != 0 || codes[3] != 0 || codes[7] != 0 {
		t.Fatalf("no-guard operation schedule=%v total=%d", codes, len(run.MeterRecords))
	}
	experiment := run.Store.Get(run.Experiment)
	winner := run.Store.Get(experiment.GetStrings("winnerResultUnits")[0])
	candidate := run.Store.Get(winner.GetString("candidate"))
	guard, err := actionrelations.ParseGuard([]byte(candidate.GetString("guard")))
	if err != nil || len(guard.Literals) != 0 {
		t.Fatalf("guard=%+v err=%v", guard, err)
	}
}
