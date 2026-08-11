package actionrelationexp

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/actionrelationsearch"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestRetainedOrderedDFSRejectsReorderedLifecycle(t *testing.T) {
	runID := testDigest("ordered-dfs-run")[:32]
	world := testAuthorityDigest("ordered-dfs-world")
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	occurrences, err := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{{Kind: "transfer", XRole: "c0", YRole: "c1", N: 1}})
	if err != nil {
		t.Fatal(err)
	}
	stateCanonical, _ := state.CanonicalJSON()
	stateDigest, _ := state.Digest()
	occurrenceCanonical, _ := occurrences[0].CanonicalJSON()
	occurrenceDigest, _ := occurrences[0].Digest()
	remaining, _ := actionrelationsearch.BuildRemaining(occurrences)
	proof, _ := actionrelationsearch.BuildProofMap(nil)
	node, _ := actionrelationsearch.BuildSearchNode(state, remaining, proof)
	terminal, _ := actionrelationsearch.BuildTerminalBehaviorFromApplicability(state, occurrences, []bool{false})
	terminalSet, _ := actionrelationsearch.BuildTerminalSet([]string{terminal.Digest})
	subtree, _ := actionrelationsearch.BuildSubtreeRoot(node.Digest, nil)
	appCanonical, _ := json.Marshal([]any{"action-applicability-row/v1", stateDigest, occurrenceDigest, false, "valid"})
	appDigest := shaHex(appCanonical)

	objects := map[string]retainedObjectValue{
		stateDigest:        {kind: 1, canonical: stateCanonical},
		occurrenceDigest:   {kind: 3, canonical: occurrenceCanonical},
		remaining.Digest:   {kind: 5, canonical: remaining.Canonical},
		proof.Digest:       {kind: 19, canonical: proof.Canonical},
		node.Digest:        {kind: 20, canonical: node.Canonical},
		terminal.Digest:    {kind: 23, canonical: terminal.Canonical},
		terminalSet.Digest: {kind: 24, canonical: terminalSet.Canonical},
		subtree.Digest:     {kind: 25, canonical: subtree.Canonical},
		appDigest:          {kind: 38, canonical: appCanonical},
	}
	structural := map[string]bool{
		fmt.Sprintf("5:%s", remaining.Digest):    true,
		fmt.Sprintf("19:%s", proof.Digest):       true,
		fmt.Sprintf("24:%s", terminalSet.Digest): true,
		fmt.Sprintf("25:%s", subtree.Digest):     true,
	}
	payload := func(values ...any) []json.RawMessage {
		wire, _ := json.Marshal(values)
		var result []json.RawMessage
		_ = json.Unmarshal(wire, &result)
		return result
	}
	task := func(sequence int, kind string, fields []any, code uint8) string {
		wire, _ := json.Marshal([]any{"actionrelation-utility-task/v1", runID, kind, fields, []uint8{code}})
		reservation, err := actionrelationledger.BuildReservation(runID, shaHex(wire), []uint8{code}, sequence, 100)
		if err != nil {
			t.Fatal(err)
		}
		objects[reservation.Digest] = retainedObjectValue{kind: 27, canonical: reservation.Canonical}
		return reservation.Digest
	}
	rows := []string{appDigest}
	calls := []retainedCall{
		{sequence: 0, operation: 16, status: 1, source: task(0, "node-lookup", []any{stateDigest, remaining.Digest, proof.Digest}, 16), payload: payload("search-node-lookup", stateDigest, remaining.Digest, proof.Digest), outputs: []string{node.Digest}},
		{sequence: 1, operation: 23, status: 1, source: task(1, "enabledness", []any{node.Digest, occurrenceDigest}, 23), payload: payload("search-applicable", world, "complete", node.Digest, stateDigest, occurrenceDigest), outputs: []string{appDigest}},
		{sequence: 2, operation: 19, status: 1, source: task(2, "terminal", []any{stateDigest, remaining.Digest, rows}, 19), payload: payload("terminal-construct", stateDigest, remaining.Digest, rows), outputs: []string{terminal.Digest}},
	}
	authority := retainedRunAuthority{phase: 2, terminal: "completed", terminalSet: terminalSet.Digest, historyCount: 1, world: world, policy: "complete", initialState: stateDigest, initialOccurrences: []string{occurrenceDigest}}
	if err := verifyRetainedOrderedDFS(runID, authority, calls, objects, structural); err != nil {
		t.Fatalf("valid ordered DFS: %v", err)
	}
	forgedObjects := cloneRetainedObjects(objects)
	forgedReservation, err := actionrelationledger.BuildReservation(runID, testAuthorityDigest("wrong-task"), []uint8{23}, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	forgedObjects[calls[1].source] = retainedObjectValue{kind: 27, canonical: forgedReservation.Canonical}
	if err := verifyRetainedOrderedDFS(runID, authority, calls, forgedObjects, structural); err == nil {
		t.Fatal("accepted a lifecycle block under a different reserved task")
	}
	reordered := slices.Clone(calls)
	reordered[1], reordered[2] = reordered[2], reordered[1]
	if err := verifyRetainedOrderedDFS(runID, authority, reordered, objects, structural); err == nil {
		t.Fatal("accepted a terminal construction before the current node's enabledness scan")
	}
}

func TestRetainedOrderedDFSAcceptsOnlyExactEarlyBudgetTask(t *testing.T) {
	runID := testDigest("ordered-early-budget")[:32]
	tables := make([]string, 8)
	for index := range tables {
		tables[index] = testAuthorityDigest(fmt.Sprintf("table-%d", index))
	}
	boundaryCanonical, _ := json.Marshal([]any{"action-store-boundary/v3", 0, "nous", tables, testAuthorityDigest("object-set"), testAuthorityDigest("index")})
	boundaryDigest := shaHex(boundaryCanonical)
	artifactCanonical, _ := json.Marshal([]any{"guarded-action-artifact/v1", []string{testAuthorityDigest("relation")}, testAuthorityDigest("training")})
	artifactDigest := shaHex(artifactCanonical)
	taskWire, _ := json.Marshal([]any{"actionrelation-utility-task/v1", runID, "artifact-load", []any{boundaryDigest, artifactDigest}, []uint8{10}})
	rejected, err := actionrelationledger.BuildReservation(runID, shaHex(taskWire), []uint8{10}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	terminalTask, _ := json.Marshal([]any{"actionrelation-budget-terminal-task/v1", runID, rejected.Digest})
	terminalReservation, err := actionrelationledger.BuildTerminalReservation(runID, shaHex(terminalTask), 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	work := [12]int{}
	work[11] = 1
	terminalCanonical, _ := json.Marshal([]any{"action-work-terminal/v1", runID, 2, rejected.Digest, "budget-exhausted", work, 1, 0})
	terminalDigest := shaHex(terminalCanonical)
	objects := map[string]retainedObjectValue{
		boundaryDigest:             {kind: 35, canonical: boundaryCanonical},
		artifactDigest:             {kind: 10, canonical: artifactCanonical},
		rejected.Digest:            {kind: 27, canonical: rejected.Canonical},
		terminalReservation.Digest: {kind: 27, canonical: terminalReservation.Canonical},
		terminalDigest:             {kind: 49, canonical: terminalCanonical},
	}
	payloadWire, _ := json.Marshal([]any{"budget-terminal", rejected.Digest})
	var payload []json.RawMessage
	_ = json.Unmarshal(payloadWire, &payload)
	calls := []retainedCall{{sequence: 0, operation: 19, status: 1, source: terminalReservation.Digest, payload: payload, outputs: []string{terminalDigest}}}
	authority := retainedRunAuthority{curriculum: 0, phase: 2, terminal: "budget-exhausted", policy: "learned-no-use", artifact: artifactDigest}
	if err := verifyRetainedOrderedDFS(runID, authority, calls, objects, map[string]bool{}); err != nil {
		t.Fatalf("valid early budget prefix: %v", err)
	}
	forged, err := actionrelationledger.BuildReservation(runID, testAuthorityDigest("wrong-next-task"), []uint8{10}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	objects[rejected.Digest] = retainedObjectValue{kind: 27, canonical: forged.Canonical}
	if err := verifyRetainedOrderedDFS(runID, authority, calls, objects, map[string]bool{}); err == nil {
		t.Fatal("accepted early exhaustion for a different next task")
	}
}

func TestRetainedLearnedEligibilityRequiresEveryArtifactRelationInOrder(t *testing.T) {
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	occurrences, err := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{{Kind: "add", XRole: "c0", N: 1}, {Kind: "add", XRole: "c1", N: 1}})
	if err != nil {
		t.Fatal(err)
	}
	a, b, err := actionrelations.CanonicalPair(occurrences[0], occurrences[1])
	if err != nil {
		t.Fatal(err)
	}
	stateCanonical, _ := state.CanonicalJSON()
	stateDigest, _ := state.Digest()
	aCanonical, _ := a.CanonicalJSON()
	bCanonical, _ := b.CanonicalJSON()
	aDigest, _ := a.Digest()
	bDigest, _ := b.Digest()
	pattern, _ := actionrelations.PatternFor(a, b)
	relation := actionrelations.Relation{Pattern: pattern, Guard: actionrelations.Guard{}, PositiveObservations: []string{}, NegativeObservations: []string{}}
	relationCanonical, err := relation.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	relationDigest, _ := relation.Digest()
	artifact, err := actionrelations.NewArtifact([]actionrelations.Relation{relation}, testAuthorityDigest("training-root"))
	if err != nil {
		t.Fatal(err)
	}
	artifactCanonical, _ := artifact.CanonicalJSON()
	artifactDigest := shaHex(artifactCanonical)
	aFacts, _ := actionrelations.Facts(state, a)
	bFacts, _ := actionrelations.Facts(state, b)
	aFactsCanonical, _ := aFacts.CanonicalJSON()
	bFactsCanonical, _ := bFacts.CanonicalJSON()
	aFactsDigest, bFactsDigest := shaHex(aFactsCanonical), shaHex(bFactsCanonical)
	aAppCanonical, _ := json.Marshal([]any{"action-applicability-row/v1", stateDigest, aDigest, true, "valid"})
	bAppCanonical, _ := json.Marshal([]any{"action-applicability-row/v1", stateDigest, bDigest, true, "valid"})
	aAppDigest, bAppDigest := shaHex(aAppCanonical), shaHex(bAppCanonical)
	matchCanonical, _ := json.Marshal([]any{"action-relation-match-row/v1", relationDigest, stateDigest, aFactsDigest, bFactsDigest, aAppDigest, bAppDigest, true, true, []string{}, true, "valid"})
	matchDigest := shaHex(matchCanonical)
	objects := map[string]retainedObjectValue{
		stateDigest:    {kind: 1, canonical: stateCanonical},
		aDigest:        {kind: 3, canonical: aCanonical},
		bDigest:        {kind: 3, canonical: bCanonical},
		aFactsDigest:   {kind: 8, canonical: aFactsCanonical},
		bFactsDigest:   {kind: 8, canonical: bFactsCanonical},
		relationDigest: {kind: 9, canonical: relationCanonical},
		artifactDigest: {kind: 10, canonical: artifactCanonical},
		aAppDigest:     {kind: 38, canonical: aAppCanonical},
		bAppDigest:     {kind: 38, canonical: bAppCanonical},
		matchDigest:    {kind: 42, canonical: matchCanonical},
	}
	payload := func(values ...any) []json.RawMessage {
		wire, _ := json.Marshal(values)
		var result []json.RawMessage
		_ = json.Unmarshal(wire, &result)
		return result
	}
	calls := []retainedCall{
		{operation: 21, status: 1, payload: payload("relation-instance-applicable", stateDigest, aDigest), outputs: []string{aAppDigest}},
		{operation: 21, status: 1, payload: payload("relation-instance-applicable", stateDigest, bDigest), outputs: []string{bAppDigest}},
		{operation: 9, status: 1, payload: payload("relation-match", relationDigest, stateDigest, aFactsDigest, bFactsDigest, aAppDigest, bAppDigest, []string{}), outputs: []string{matchDigest}},
	}
	authority := retainedRunAuthority{policy: "nous-guarded-sleep", artifact: artifactDigest}
	attempt := decodedCertificateAttempt{State: stateDigest, A: aDigest, B: bDigest}
	if count, ok := retainedEligibilitySchedule(authority, attempt, calls, objects, true); !ok || count != len(calls) {
		t.Fatal("valid learned eligibility schedule did not reconstruct")
	}
	if _, ok := retainedEligibilitySchedule(authority, attempt, calls[:2], objects, true); ok {
		t.Fatal("accepted learned eligibility with its artifact relation omitted")
	}
	reordered := slices.Clone(calls)
	reordered[1], reordered[2] = reordered[2], reordered[1]
	if _, ok := retainedEligibilitySchedule(authority, attempt, reordered, objects, true); ok {
		t.Fatal("accepted learned eligibility outside artifact order")
	}
}
