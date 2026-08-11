package actionrelationutility

import (
	"testing"

	"github.com/chazu/nous/internal/actionrelationsearch"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestSemanticVerifierRejectsStructurallyValidFalseDeadlock(t *testing.T) {
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}}, Events: []string{}}
	occurrences, err := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{{Kind: "add", XRole: "c0", N: 1}})
	if err != nil {
		t.Fatal(err)
	}
	remaining, _ := actionrelationsearch.BuildRemaining(occurrences)
	proofMap, _ := actionrelationsearch.BuildProofMap(nil)
	node, _ := actionrelationsearch.BuildSearchNode(state, remaining, proofMap)
	// The structural constructor accepts a supplied applicability vector; the
	// independent semantic verifier must prove that vector rather than trust it.
	terminal, err := actionrelationsearch.BuildTerminalBehaviorFromApplicability(state, occurrences, []bool{false})
	if err != nil {
		t.Fatal(err)
	}
	terminalSet, _ := actionrelationsearch.BuildTerminalSet([]string{terminal.Digest})
	subtree, _ := actionrelationsearch.BuildSubtreeRoot(node.Digest, nil)
	result := actionrelationsearch.Result{
		Policy: actionrelationsearch.Complete, TerminalDigests: []string{terminal.Digest}, NodeLookups: 1, ConstructedNodes: 1, HistoryCount: 1,
		RootNodeDigest: node.Digest, RootSubtree: subtree, TerminalSet: terminalSet,
		Nodes: []actionrelationsearch.EvidenceObject{node}, RemainingSets: []actionrelationsearch.EvidenceObject{remaining}, ProofMaps: []actionrelationsearch.EvidenceObject{proofMap},
		TerminalBehaviors: []actionrelationsearch.EvidenceObject{terminal}, SubtreeRoots: []actionrelationsearch.EvidenceObject{subtree}, TerminalSets: []actionrelationsearch.EvidenceObject{terminalSet},
	}
	if err := actionrelationsearch.VerifyResultEvidence(result); err != nil {
		t.Fatalf("false deadlock should be structurally closed: %v", err)
	}
	stateJSON, _ := state.CanonicalJSON()
	stateDigest, _ := state.Digest()
	occurrenceJSON, _ := occurrences[0].CanonicalJSON()
	occurrenceDigest, _ := occurrences[0].Digest()
	if err := VerifyResultSemantics(result, map[string][]byte{stateDigest: stateJSON, occurrenceDigest: occurrenceJSON}); err == nil {
		t.Fatal("semantic verifier accepted an enabled occurrence as deadlocked")
	}
}

func TestStructuralVerifierRejectsMissingCompletedBranchAndForgedSummary(t *testing.T) {
	world := actionrelations.World{
		State:   actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}}, Events: []string{}},
		Actions: []actionrelations.Action{{Name: "a", Kind: "add", X: "c0", N: 1}},
	}
	result, err := actionrelationsearch.Search(world, actionrelationsearch.Complete, actionrelationsearch.Artifact{})
	if err != nil || actionrelationsearch.VerifyResultEvidence(result) != nil {
		t.Fatalf("valid complete result: %v", err)
	}
	withoutBranch := result
	withoutBranch.CompletedSubtrees = nil
	if err := actionrelationsearch.VerifyResultEvidence(withoutBranch); err == nil {
		t.Fatal("structural verifier accepted an edge without completed-subtree authority")
	}
	wrongSummary := result
	wrongSummary.TerminalDigests = nil
	if err := actionrelationsearch.VerifyResultEvidence(wrongSummary); err == nil {
		t.Fatal("structural verifier accepted a forged terminal summary")
	}
}
