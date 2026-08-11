package actionrelationsearch

import (
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationoracle"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func oracleTestCertifier(state actionrelations.State, left, right actionrelations.Occurrence) (bool, error) {
	stateJSON, _ := state.CanonicalJSON()
	leftJSON, _ := left.Action.CanonicalJSON()
	rightJSON, _ := right.Action.CanonicalJSON()
	observation, err := actionrelationoracle.Observe(stateJSON, leftJSON, rightJSON)
	return err == nil && observation.Label == "commutes", err
}

func TestCertifiedPoliciesPreserveTerminalBehaviors(t *testing.T) {
	world := actionrelations.World{
		State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "a", Value: 0}, {Name: "b", Value: 0}, {Name: "c", Value: 0}}},
		Actions: []actionrelations.Action{
			{Name: "one", Kind: "set", X: "a", N: 1},
			{Name: "two", Kind: "set", X: "b", N: 2},
			{Name: "three", Kind: "set", X: "c", N: 3},
		},
	}
	complete, err := Search(world, Complete, Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyResultEvidence(complete); err != nil {
		t.Fatalf("complete evidence: %v", err)
	}
	for _, policy := range []Policy{Lexical, StaticSleep, DynamicSleep} {
		result, err := SearchWithCertifier(world, policy, Artifact{}, oracleTestCertifier)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(result.TerminalDigests, complete.TerminalDigests) {
			t.Fatalf("%s terminals=%v complete=%v", policy, result.TerminalDigests, complete.TerminalDigests)
		}
		if err := VerifyResultEvidence(result); err != nil {
			t.Fatalf("%s evidence: %v", policy, err)
		}
		if policy != Lexical && result.SleepPropagations == 0 {
			t.Fatalf("%s produced no certified sleep propagation: result=%+v complete=%+v", policy, result, complete)
		}
	}
}

func TestEvidenceProducingCertifierClosesProofDAG(t *testing.T) {
	world := actionrelations.World{
		State:   actionrelations.State{Cells: []actionrelations.Cell{{Name: "a", Value: 0}, {Name: "b", Value: 0}, {Name: "c", Value: 0}}},
		Actions: []actionrelations.Action{{Name: "one", Kind: "set", X: "a", N: 1}, {Name: "two", Kind: "set", X: "b", N: 2}, {Name: "three", Kind: "set", X: "c", N: 3}},
	}
	result, err := SearchWithEvidenceAdapters(world, DynamicSleep, Artifact{}, nil, func(state actionrelations.State, left, right actionrelations.Occurrence) (CertificateDecision, error) {
		stateDigest, _ := state.Digest()
		leftDigest, _ := left.Digest()
		rightDigest, _ := right.Digest()
		return CertificateDecision{Certified: true, CertificateDigest: testEvidenceDigest(stateDigest + leftDigest + rightDigest)}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.CertificateEvidenceBound || len(result.Propagations) == 0 || len(result.CompletedSubtrees) == 0 {
		t.Fatalf("missing evidence-producing search records: %+v", result)
	}
	if err := VerifyResultEvidence(result); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceProducingCertifierRejectsMissingCertificateDigest(t *testing.T) {
	world := actionrelations.World{State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "a", Value: 0}, {Name: "b", Value: 0}}}, Actions: []actionrelations.Action{{Name: "one", Kind: "set", X: "a", N: 1}, {Name: "two", Kind: "set", X: "b", N: 2}}}
	_, err := SearchWithEvidenceAdapters(world, DynamicSleep, Artifact{}, nil, func(actionrelations.State, actionrelations.Occurrence, actionrelations.Occurrence) (CertificateDecision, error) {
		return CertificateDecision{Certified: true}, nil
	})
	if err == nil {
		t.Fatal("accepted certified decision without retained certificate")
	}
}

func TestConflictingActionsAreNeverSleepPruned(t *testing.T) {
	world := actionrelations.World{
		State: actionrelations.State{Cells: []actionrelations.Cell{{Name: "x", Value: 0}}},
		Actions: []actionrelations.Action{
			{Name: "low", Kind: "set", X: "x", N: 1},
			{Name: "high", Kind: "set", X: "x", N: 3},
		},
	}
	complete, _ := Search(world, Complete, Artifact{})
	dynamic, _ := SearchWithCertifier(world, DynamicSleep, Artifact{}, oracleTestCertifier)
	if !slices.Equal(complete.TerminalDigests, dynamic.TerminalDigests) || dynamic.SleepPropagations != 0 {
		t.Fatalf("complete=%+v dynamic=%+v", complete, dynamic)
	}
}
