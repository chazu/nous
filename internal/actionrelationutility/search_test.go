package actionrelationutility

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationacquire"
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
	if run.Terminal != "completed" || run.WorkTotal != len(run.Records) {
		t.Fatalf("complete utility work total=%d records=%d terminal=%s", run.WorkTotal, len(run.Records), run.Terminal)
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

func TestArtifactFreeUtilityCarriesPriorLifecycleAndPhysicalWork(t *testing.T) {
	world := independentUtilityWorld()
	initial := [12]int{3, 2}
	run, err := ExecutePolicyContinuing("../../domains", world, actionrelationsearch.Complete, "development", "authority", 3, 1, initial, WorkBudget{LifecycleCap: 4096, PhysicalCap: 4096, PriorPhysical: 7}, "complete-continuing")
	if err != nil {
		t.Fatal(err)
	}
	current, err := MeterWorkVector(run.Records)
	if err != nil {
		t.Fatal(err)
	}
	for index := range current {
		if run.WorkVector[index] != initial[index]+current[index] {
			t.Fatalf("counter %d = %d want %d", index+1, run.WorkVector[index], initial[index]+current[index])
		}
	}
	if run.InitialWork != initial || run.PriorPhysical != 7 || run.WorkTotal != 5+len(run.Records) {
		t.Fatalf("continuation authority = %+v", run)
	}
}

func TestLearnedNousUtilityLoadsFrozenArtifactAndUsesCUEBarrierBeforeSleep(t *testing.T) {
	session, err := actionrelationacquire.BeginFor("../../domains", "learned-utility", 0, 6, "development", actionrelationexp.PlanCommit)
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := actionrelationexp.CompleteAcquisition(session, 6)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := actionrelationexp.BuildAcquisitionBoundary(acquisition, 6, "nous")
	if err != nil {
		t.Fatal(err)
	}
	world := independentUtilityWorld()
	complete, _ := actionrelationsearch.Search(world, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
	initialWork, _ := MeterWorkVector(acquisition.Run.MeterRecords)
	run, err := ExecuteLearnedPolicy(acquisition.Run.Store, acquisition.Run.Artifact, boundary.BoundaryUnit, world, actionrelationsearch.NousSleep, "development", actionrelationexp.PlanCommit, 6, 0, initialWork, 2_000_000, "learned-nous")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(run.Search.TerminalDigests, complete.TerminalDigests) || len(run.Search.Propagations) == 0 || !run.Search.CertificateEvidenceBound {
		t.Fatalf("learned=%+v complete=%v", run.Search, complete.TerminalDigests)
	}
	if run.WorkTotal != len(acquisition.Run.MeterRecords)+len(run.Records) {
		t.Fatalf("learned lifecycle total=%d acquisition=%d utility=%d", run.WorkTotal, len(acquisition.Run.MeterRecords), len(run.Records))
	}
	seen := map[uint16]bool{}
	firstPairApplicable, firstCacheLookup := -1, -1
	for sequence, record := range run.Records {
		seen[record.Code] = true
		if firstPairApplicable < 0 && record.Code == 21 {
			firstPairApplicable = sequence
		}
		if firstCacheLookup < 0 && record.Code == 18 {
			firstCacheLookup = sequence
		}
	}
	for _, code := range []uint16{9, 10, 21, 18, 25} {
		if !seen[code] {
			t.Fatalf("learned utility omitted operation %d", code)
		}
	}
	if len(run.Records) == 0 || run.Records[0].Code != 10 || firstPairApplicable < 0 || firstCacheLookup <= firstPairApplicable {
		t.Fatalf("learned utility ordering applicable=%d cache=%d records=%#v", firstPairApplicable, firstCacheLookup, run.Records)
	}
	control, err := ExecuteLearnedPolicy(acquisition.Run.Store, acquisition.Run.Artifact, boundary.BoundaryUnit, world, actionrelationsearch.LearnedNoUse, "development", actionrelationexp.PlanCommit, 6, 1, initialWork, 2_000_000, "learned-no-use")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(control.Search.TerminalDigests, complete.TerminalDigests) || control.Search.SleepPropagations != 0 || len(control.Records) == 0 || control.Records[0].Code != 10 {
		t.Fatalf("learned-no-use did not load the artifact then explore completely: %+v", control.Search)
	}
	for _, record := range control.Records[1:] {
		if slices.Contains([]uint16{9, 15, 18, 21, 25}, record.Code) {
			t.Fatalf("learned-no-use performed artifact eligibility work %d", record.Code)
		}
	}
}

func TestNoGuardUtilityUsesItsSeparateRootOnlyAcquisitionAuthority(t *testing.T) {
	session, err := actionrelationacquire.BeginNoGuardFor("../../domains", "no-guard-utility", 0, 7, "development", actionrelationexp.PlanCommit)
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := actionrelationexp.CompleteNoGuardAcquisition(session, 7)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := actionrelationexp.BuildAcquisitionBoundary(acquisition, 7, "no-guard")
	if err != nil {
		t.Fatal(err)
	}
	world := independentUtilityWorld()
	complete, _ := actionrelationsearch.Search(world, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
	initialWork, _ := MeterWorkVector(acquisition.Run.MeterRecords)
	run, err := ExecuteLearnedPolicy(acquisition.Run.Store, acquisition.Run.Artifact, boundary.BoundaryUnit, world, actionrelationsearch.NoGuardSleep, "development", actionrelationexp.PlanCommit, 7, 0, initialWork, 2_000_000, "no-guard")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(run.Search.TerminalDigests, complete.TerminalDigests) || len(run.Search.Propagations) == 0 {
		t.Fatalf("no-guard=%+v complete=%v", run.Search, complete.TerminalDigests)
	}
	seen := map[uint16]bool{}
	for _, record := range run.Records {
		seen[record.Code] = true
	}
	for _, code := range []uint16{9, 10, 21, 18, 25} {
		if !seen[code] {
			t.Fatalf("no-guard utility omitted operation %d", code)
		}
	}
	if seen[15] {
		t.Fatal("root-only no-guard artifact emitted learned literal work")
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

func TestUtilityBudgetExhaustionRejectsWholeBlockAndEmitsKind49(t *testing.T) {
	run, err := ExecuteComplete("../../domains", independentUtilityWorld(), "development", "authority", 8, 0, 2, "budget")
	if err != nil {
		t.Fatal(err)
	}
	if run.Terminal != "budget-exhausted" || actionrelationexp.ValidateObject(49, run.WorkTerminal.Canonical) != nil || run.WorkTotal != 2 || run.WorkVector[10] != 1 || run.WorkVector[11] != 1 {
		t.Fatalf("budget run=%+v", run)
	}
	if len(run.Records) != 2 || run.Records[0].Code != 16 || run.Records[1].Code != 19 {
		t.Fatalf("budget records=%#v", run.Records)
	}
	var terminalWire []any
	if json.Unmarshal(run.WorkTerminal.Canonical, &terminalWire) != nil || len(terminalWire) != 8 || terminalWire[4] != "budget-exhausted" || terminalWire[7] != float64(0) {
		t.Fatalf("terminal=%v", terminalWire)
	}
	rejectedDigest := terminalWire[3].(string)
	if run.Records[1].SourceTaskDigest == rejectedDigest {
		t.Fatal("rejected compound reservation was used as charged authority")
	}
}

func TestUtilityPhysicalCapIsSeparateAndCumulativeAcrossWorlds(t *testing.T) {
	run, err := ExecutePolicyWithBudget(
		"../../domains", independentUtilityWorld(), actionrelationsearch.Complete,
		"development", "authority", 8, 1,
		WorkBudget{LifecycleCap: 2_000_000, PhysicalCap: 4096, PriorPhysical: 4095},
		"physical-budget",
	)
	if err != nil {
		t.Fatal(err)
	}
	if run.Terminal != "budget-exhausted" || run.PhysicalWork != 1 || run.PriorPhysical+run.PhysicalWork != 4096 || run.WorkTotal != 1 {
		t.Fatalf("physical-cap run=%+v", run)
	}
	if len(run.Records) != 1 || run.Records[0].Code != 19 {
		t.Fatalf("physical-cap records=%#v", run.Records)
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
