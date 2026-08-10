package actionrelationsearch

import (
	"encoding/json"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestSearchEvidenceBuildsFrozenAcyclicProofChain(t *testing.T) {
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}, Events: []string{}}
	occurrences, err := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{{Kind: "add", XRole: "c0", N: 1}, {Kind: "add", XRole: "c1", N: 1}})
	if err != nil {
		t.Fatal(err)
	}
	remaining, err := BuildRemaining(occurrences)
	if err != nil {
		t.Fatal(err)
	}
	emptyProof, err := BuildProofMap(nil)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := BuildSearchNode(state, remaining, emptyProof)
	if err != nil {
		t.Fatal(err)
	}
	next, outcome, err := actionrelations.Apply(state, occurrences[0].Action)
	if err != nil || outcome != "applied" {
		t.Fatalf("apply: %s %v", outcome, err)
	}
	childRemaining, err := BuildRemaining(occurrences[1:])
	if err != nil {
		t.Fatal(err)
	}
	taken, _ := occurrences[0].Digest()
	sleeper, _ := occurrences[1].Digest()
	propagation, err := BuildPropagation(parent.Digest, taken, sleeper, "earlier-sibling", testEvidenceDigest("completed sibling"), testEvidenceDigest("certificate"), mustStateDigest(next), childRemaining.Digest)
	if err != nil {
		t.Fatal(err)
	}
	entry := ProofEntry{SleeperDigest: sleeper, PropagationDigest: propagation.Digest}
	proofMap, err := BuildProofMap([]ProofEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	child, err := BuildSearchNode(next, childRemaining, proofMap)
	if err != nil {
		t.Fatal(err)
	}
	edge, err := BuildSearchEdge(parent.Digest, taken, []ProofEntry{entry}, child.Digest)
	if err != nil {
		t.Fatal(err)
	}
	terminalSet, _ := BuildTerminalSet(nil)
	subtree, _ := BuildSubtreeRoot(child.Digest, nil)
	completed, err := BuildCompletedSubtree(parent.Digest, taken, subtree, terminalSet)
	if err != nil {
		t.Fatal(err)
	}
	for name, object := range map[string]EvidenceObject{"remaining": remaining, "proof": proofMap, "parent": parent, "propagation": propagation, "child": child, "edge": edge, "subtree": subtree, "terminal-set": terminalSet, "completed": completed} {
		if object.Digest != shaHex(object.Canonical) || !json.Valid(object.Canonical) {
			t.Fatalf("%s is not canonical digest authority", name)
		}
	}
	for kind, object := range map[uint16]EvidenceObject{5: remaining, 18: propagation, 19: proofMap, 20: parent, 21: edge, 22: completed, 24: terminalSet, 25: subtree} {
		if err := actionrelationexp.ValidateObject(kind, object.Canonical); err != nil {
			t.Fatalf("kind %d: %v", kind, err)
		}
	}
}

func TestTerminalBehaviorRejectsEnabledRemainder(t *testing.T) {
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}}, Events: []string{}}
	occurrences, _ := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{{Kind: "add", XRole: "c0", N: 1}})
	if _, err := BuildTerminalBehavior(state, occurrences); err == nil {
		t.Fatal("accepted enabled remainder as a deadlock")
	}
	blocked, _ := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{{Kind: "check", XRole: "c0", N: 3}})
	terminal, err := BuildTerminalBehavior(state, blocked)
	if err != nil || !taggedEvidence(terminal, "action-terminal/v1") {
		t.Fatalf("deadlock: %v", err)
	}
	if err := actionrelationexp.ValidateObject(23, terminal.Canonical); err != nil {
		t.Fatal(err)
	}
	complete, err := BuildTerminalBehavior(state, nil)
	if err != nil || complete.Digest == terminal.Digest {
		t.Fatalf("complete: %v", err)
	}
}

func TestProofMapRejectsDuplicateSleeperAndPropagationRejectsAlias(t *testing.T) {
	digest := testEvidenceDigest("same")
	if _, err := BuildProofMap([]ProofEntry{{SleeperDigest: digest, PropagationDigest: testEvidenceDigest("a")}, {SleeperDigest: digest, PropagationDigest: testEvidenceDigest("b")}}); err == nil {
		t.Fatal("accepted duplicate proof-map sleeper")
	}
	if _, err := BuildPropagation(testEvidenceDigest("parent"), digest, digest, "prior-sleep", testEvidenceDigest("proof"), testEvidenceDigest("certificate"), testEvidenceDigest("state"), testEvidenceDigest("remaining")); err == nil {
		t.Fatal("accepted propagation from an occurrence to itself")
	}
}

func testEvidenceDigest(marker string) string { return shaHex([]byte(marker)) }

func mustStateDigest(state actionrelations.State) string { digest, _ := state.Digest(); return digest }
