package actionrelationutility

import (
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationsearch"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestCompleteUtilityDFSUsesOnlyReservedCUESemantics(t *testing.T) {
	world := actionrelations.World{
		State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "x", Value: 0}, {Name: "y", Value: 0}}},
		Actions: []actionrelations.Action{
			{Name: "left", Kind: "add", X: "x", N: 1},
			{Name: "right", Kind: "add", X: "y", N: 1},
		},
	}
	run, err := ExecuteComplete("../../domains", world, "development", "authority", 3, 0, 4096, "complete")
	if err != nil {
		t.Fatal(err)
	}
	oracle, err := actionrelationsearch.Search(world, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
	if err != nil || !slices.Equal(run.Search.TerminalDigests, oracle.TerminalDigests) {
		t.Fatalf("terminal mismatch got=%v want=%v err=%v", run.Search.TerminalDigests, oracle.TerminalDigests, err)
	}
	if err := actionrelationsearch.VerifyResultEvidence(run.Search); err != nil {
		t.Fatal(err)
	}
	if actionrelationexp.ValidateObject(46, run.RunRoot.Canonical) != nil || len(run.Transcript.CallIDs) != len(run.Records) {
		t.Fatal("complete utility run lacks an exact charged operation range")
	}
	seen := map[uint16]bool{}
	hit := false
	for _, record := range run.Records {
		if record.SourceTaskDigest == "" {
			t.Fatal("utility DFS call lacks pre-execution reservation")
		}
		if !slices.Contains([]uint16{11, 16, 19, 23}, record.Code) {
			t.Fatalf("unexpected complete-policy operation %d", record.Code)
		}
		seen[record.Code] = true
		hit = hit || record.Code == 16 && record.Status == 3
	}
	for _, code := range []uint16{11, 16, 19, 23} {
		if !seen[code] {
			t.Fatalf("complete utility DFS omitted operation %d", code)
		}
	}
	if !hit {
		t.Fatal("complete utility DFS did not retain its exact node-dedup hit")
	}
}
