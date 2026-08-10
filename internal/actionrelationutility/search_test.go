package actionrelationutility

import (
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationsearch"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestCompleteUtilityDFSUsesOnlyReservedCUESemantics(t *testing.T) {
	world := independentUtilityWorld()
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

func TestCertifiedUtilityPoliciesRetainFreshOrientedSleepProofs(t *testing.T) {
	world := independentUtilityWorld()
	complete, err := actionrelationsearch.Search(world, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range []actionrelationsearch.Policy{actionrelationsearch.DynamicSleep, actionrelationsearch.StaticSleep} {
		t.Run(string(policy), func(t *testing.T) {
			run, err := ExecutePolicy("../../domains", world, policy, "development", "authority", 4, 0, 8192, string(policy))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(run.Search.TerminalDigests, complete.TerminalDigests) || !run.Search.CertificateEvidenceBound || len(run.Search.Propagations) == 0 {
				t.Fatalf("certified result=%+v complete=%v", run.Search, complete.TerminalDigests)
			}
			if err := actionrelationsearch.VerifyResultEvidence(run.Search); err != nil {
				t.Fatal(err)
			}
			seen := map[uint16]bool{}
			priorSleepLookup := false
			for _, record := range run.Records {
				seen[record.Code] = true
				priorSleepLookup = priorSleepLookup || record.Code == 17 && len(record.Outputs) == 1
				if record.SourceTaskDigest == "" {
					t.Fatal("certified utility call lacks reservation")
				}
			}
			for _, code := range []uint16{12, 13, 14, 17, 18, 25} {
				if !seen[code] {
					t.Fatalf("certified policy omitted operation %d", code)
				}
			}
			if policy == actionrelationsearch.StaticSleep && !seen[24] {
				t.Fatal("static policy omitted its exact footprint predicate")
			}
			if !priorSleepLookup {
				t.Fatal("certified policy omitted prior-sleep proof-map authority")
			}
		})
	}
}

func independentUtilityWorld() actionrelations.World {
	return actionrelations.World{
		State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "x", Value: 0}, {Name: "y", Value: 0}, {Name: "z", Value: 0}}},
		Actions: []actionrelations.Action{
			{Name: "left", Kind: "add", X: "x", N: 1},
			{Name: "right", Kind: "add", X: "y", N: 1},
			{Name: "middle", Kind: "add", X: "z", N: 1},
		},
	}
}
