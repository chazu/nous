package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/actionrelationoracle"
	"github.com/chazu/nous/internal/actionrelationsearch"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestRetainedCertificateFinalizationTailRequiresProducedPreimages(t *testing.T) {
	runID := testDigest("retained-finalization-tail")[:32]
	world := testAuthorityDigest("retained-finalization-world")
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	occurrences, err := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{
		{Kind: "set", XRole: "c0", N: 1},
		{Kind: "set", XRole: "c1", N: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(occurrences, func(a, b actionrelations.Occurrence) int {
		aBytes, _ := a.CanonicalJSON()
		bBytes, _ := b.CanonicalJSON()
		return bytes.Compare(aBytes, bBytes)
	})
	stateCanonical, _ := state.CanonicalJSON()
	stateDigest, _ := state.Digest()
	aCanonical, _ := occurrences[0].CanonicalJSON()
	bCanonical, _ := occurrences[1].CanonicalJSON()
	aDigest, _ := occurrences[0].Digest()
	bDigest, _ := occurrences[1].Digest()
	objects := map[string]retainedObjectValue{
		stateDigest: {kind: 1, canonical: stateCanonical},
		aDigest:     {kind: 3, canonical: aCanonical},
		bDigest:     {kind: 3, canonical: bCanonical},
	}
	payload := func(values ...any) []json.RawMessage {
		wire, _ := json.Marshal(values)
		var result []json.RawMessage
		_ = json.Unmarshal(wire, &result)
		return result
	}
	calls := []retainedCall{}
	total := 0
	appendBlock := func(task []any, codes []uint8) {
		wire, _ := json.Marshal(task)
		reservation, buildErr := actionrelationledger.BuildReservation(runID, shaHex(wire), codes, total, 100)
		if buildErr != nil || reservation.Status != "reserved" {
			t.Fatalf("reservation: %v", buildErr)
		}
		objects[reservation.Digest] = retainedObjectValue{kind: 27, canonical: reservation.Canonical}
		for _, code := range codes {
			sequence := len(calls)
			calls = append(calls, retainedCall{sequence: sequence, operation: code, status: 1, source: reservation.Digest, callID: testAuthorityDigest(fmt.Sprintf("finalization-call-%d", sequence))})
		}
		total += len(codes)
	}
	pair := sortedPair(aDigest, bDigest)
	lookupWire := []any{"actionrelation-cache-lookup-task/v1", runID, world, "dynamic-diamond-sleep", stateDigest, pair[0], pair[1]}
	appendBlock(lookupWire, []uint8{18})
	calls[0].payload = payload("certificate-cache-lookup", world, "dynamic-diamond-sleep", stateDigest, pair[0], pair[1])
	request := "AR.CertificateRequest.policy-c0000-p03-w0.00000"
	for _, stage := range []string{"initial", "cross"} {
		codes := []uint8{13, 13, 12, 12}
		appendBlock([]any{"actionrelation-certificate-stage/v1", runID, request, stage, codes}, codes)
		calls[len(calls)-2].outputs = []string{testAuthorityDigest(stage + "-left-state"), testAuthorityDigest(stage + "-left-outcome")}
		calls[len(calls)-1].outputs = []string{testAuthorityDigest(stage + "-right-state"), testAuthorityDigest(stage + "-right-outcome")}
	}
	appendBlock([]any{"actionrelation-certificate-stage/v1", runID, request, "equality", []uint8{14}}, []uint8{14})
	callIDs := make([]string, len(calls))
	for index := range calls {
		callIDs[index] = calls[index].callID
	}
	root, err := BuildOperationRange(runID, 2, 0, callIDs)
	if err != nil {
		t.Fatal(err)
	}
	witness, _ := json.Marshal([]any{"dynamic-witness/v1", "all-pairs", testAuthorityDigest("retained-finalization-applicability")})
	aAction, _ := occurrences[0].Action.CanonicalJSON()
	bAction, _ := occurrences[1].Action.CanonicalJSON()
	observation, err := actionrelationoracle.Observe(stateCanonical, aAction, bAction)
	if err != nil || observation.Label != "commutes" {
		t.Fatalf("observation: %+v %v", observation, err)
	}
	certificate, _ := json.Marshal([]any{"local-diamond-certificate/v1", stateDigest, aDigest, bDigest, shaHex(witness), shaHex(observation.AB), shaHex(observation.BA), true, aDigest, root.Digest})
	certificateDigest := shaHex(certificate)
	attempt, _ := json.Marshal([]any{"local-diamond-certificate-attempt/v3", stateDigest, aDigest, bDigest, shaHex(witness), root.Digest, "certified", certificateDigest, "valid"})
	attemptDigest := shaHex(attempt)
	for digest, object := range map[string]retainedObjectValue{
		root.Digest:       {kind: 46, canonical: root.Canonical},
		attemptDigest:     {kind: 44, canonical: attempt},
		certificateDigest: {kind: 17, canonical: certificate},
	} {
		objects[digest] = object
	}
	finalizeWire, _ := json.Marshal([]any{"actionrelation-cache-finalize-task/v1", runID, world, "dynamic-diamond-sleep", stateDigest, pair[0], pair[1], calls[0].callID, attemptDigest, root.Digest})
	rejected, err := actionrelationledger.BuildReservation(runID, shaHex(finalizeWire), []uint8{25}, total, total+1)
	if err != nil || rejected.Status != "rejected-cap" {
		t.Fatalf("rejected finalization: %v", err)
	}
	objects[rejected.Digest] = retainedObjectValue{kind: 27, canonical: rejected.Canonical}
	terminalTask, _ := json.Marshal([]any{"actionrelation-budget-terminal-task/v1", runID, rejected.Digest})
	terminalReservation, err := actionrelationledger.BuildTerminalReservation(runID, shaHex(terminalTask), total, total+1)
	if err != nil {
		t.Fatal(err)
	}
	objects[terminalReservation.Digest] = retainedObjectValue{kind: 27, canonical: terminalReservation.Canonical}
	work := [12]int{}
	work[11] = total + 1
	terminal, _ := json.Marshal([]any{"action-work-terminal/v1", runID, 2, rejected.Digest, "budget-exhausted", work, total + 1, 0})
	terminalDigest := shaHex(terminal)
	objects[terminalDigest] = retainedObjectValue{kind: 49, canonical: terminal}
	calls = append(calls, retainedCall{sequence: total, operation: 19, status: 1, source: terminalReservation.Digest, payload: payload("budget-terminal", rejected.Digest), outputs: []string{terminalDigest}, callID: testAuthorityDigest("finalization-terminal-call")})
	structural := map[string]bool{
		fmt.Sprintf("46:%s", root.Digest):       true,
		fmt.Sprintf("44:%s", attemptDigest):     true,
		fmt.Sprintf("17:%s", certificateDigest): true,
	}
	r := &retainedDFSReplay{
		runID: runID, authority: retainedRunAuthority{curriculum: 0, worldOrdinal: 0, world: world, policy: "dynamic-diamond-sleep"},
		calls: calls, objects: objects, structural: structural, consumed: map[string]bool{}, preFinalizationTail: map[string]bool{},
	}
	certified, _, err := r.consumeCertificate(stateDigest, aDigest, bDigest, witness, 0)
	if !errors.Is(err, errRetainedDFSExhausted) || certified || r.cursor != len(calls) {
		t.Fatalf("finalization tail replay: certified=%t cursor=%d err=%v", certified, r.cursor, err)
	}
	wantTail := map[string]bool{
		fmt.Sprintf("46:%s", root.Digest):       true,
		fmt.Sprintf("44:%s", attemptDigest):     true,
		fmt.Sprintf("17:%s", certificateDigest): true,
	}
	if !reflect.DeepEqual(r.preFinalizationTail, wantTail) {
		t.Fatalf("tail authority=%v want=%v", r.preFinalizationTail, wantTail)
	}
	authority := r.authority
	authority.phase, authority.terminal, authority.operationRoot = 2, "budget-exhausted", root.Digest
	ordered := &retainedDFSWitnessAuthority{current: map[string]bool{}, completed: map[string]bool{}, preFinalizationTail: r.preFinalizationTail}
	if err := verifyRetainedStructuralCompleteness(authority, calls, objects, nil, structural, ordered); err != nil {
		t.Fatalf("retained finalization tail completeness: %v", err)
	}
	for key := range wantTail {
		missing := mapsCloneBool(structural)
		delete(missing, key)
		if err := verifyRetainedStructuralCompleteness(authority, calls, objects, nil, missing, ordered); err == nil {
			t.Fatalf("accepted finalization tail without %s: %v", key, err)
		}
	}
}

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
	if count, ok := retainedEligibilitySchedule(authority, attempt, aDigest, bDigest, calls, objects, true); !ok || count != len(calls) {
		t.Fatal("valid learned eligibility schedule did not reconstruct")
	}
	if _, ok := retainedEligibilitySchedule(authority, attempt, aDigest, bDigest, calls[:2], objects, true); ok {
		t.Fatal("accepted learned eligibility with its artifact relation omitted")
	}
	reordered := slices.Clone(calls)
	reordered[1], reordered[2] = reordered[2], reordered[1]
	if _, ok := retainedEligibilitySchedule(authority, attempt, aDigest, bDigest, reordered, objects, true); ok {
		t.Fatal("accepted learned eligibility outside artifact order")
	}
	reverseMatchCanonical, _ := json.Marshal([]any{"action-relation-match-row/v1", relationDigest, stateDigest, bFactsDigest, aFactsDigest, bAppDigest, aAppDigest, true, true, []string{}, true, "valid"})
	reverseMatchDigest := shaHex(reverseMatchCanonical)
	objects[reverseMatchDigest] = retainedObjectValue{kind: 42, canonical: reverseMatchCanonical}
	reversed := []retainedCall{
		{operation: 21, status: 1, payload: payload("relation-instance-applicable", stateDigest, bDigest), outputs: []string{bAppDigest}},
		{operation: 21, status: 1, payload: payload("relation-instance-applicable", stateDigest, aDigest), outputs: []string{aAppDigest}},
		{operation: 9, status: 1, payload: payload("relation-match", relationDigest, stateDigest, bFactsDigest, aFactsDigest, bAppDigest, aAppDigest, []string{}), outputs: []string{reverseMatchDigest}},
	}
	if count, ok := retainedEligibilitySchedule(authority, attempt, bDigest, aDigest, reversed, objects, true); !ok || count != len(reversed) {
		t.Fatal("valid reverse-oriented learned eligibility did not reconstruct")
	}
	if _, ok := retainedEligibilitySchedule(authority, attempt, aDigest, bDigest, reversed, objects, true); ok {
		t.Fatal("accepted learned eligibility under the wrong taken/sleeper orientation")
	}
}

func TestRetainedStaticCacheUseReconstructsFreshNodeWitness(t *testing.T) {
	world := testAuthorityDigest("static-world")
	state := testAuthorityDigest("static-state")
	taken := testAuthorityDigest("static-taken")
	sleeper := testAuthorityDigest("static-sleeper")
	aFacts := testAuthorityDigest("static-a-facts")
	bFacts := testAuthorityDigest("static-b-facts")
	objects := map[string]retainedObjectValue{}
	makeUse := func(node string) ([]retainedCall, []byte) {
		footprint, _ := json.Marshal([]any{"action-static-footprint-row/v1", world, node, state, taken, sleeper, aFacts, bFacts, true, "valid"})
		if err := ValidateObject(48, footprint); err != nil {
			t.Fatal(err)
		}
		digest := shaHex(footprint)
		objects[digest] = retainedObjectValue{kind: 48, canonical: footprint}
		calls := []retainedCall{{operation: 24, outputs: []string{digest}}}
		witness, kind, ok := retainedCurrentEligibilityWitness(retainedRunAuthority{policy: "static-rw-sleep"}, taken, sleeper, calls, objects)
		if !ok || kind != 15 {
			t.Fatal("static use did not reconstruct its current witness")
		}
		objects[shaHex(witness)] = retainedObjectValue{kind: kind, canonical: witness}
		return calls, witness
	}
	_, original := makeUse(testAuthorityDigest("static-node-original"))
	currentCalls, current := makeUse(testAuthorityDigest("static-node-current"))
	if bytes.Equal(original, current) {
		t.Fatal("static witnesses failed to bind their distinct current nodes")
	}
	rebuilt, kind, ok := retainedCurrentEligibilityWitness(retainedRunAuthority{policy: "static-rw-sleep"}, taken, sleeper, currentCalls, objects)
	if !ok || kind != 15 || !bytes.Equal(rebuilt, current) || bytes.Equal(rebuilt, original) || objects[shaHex(rebuilt)].kind != 15 {
		t.Fatal("cache reuse did not retain the fresh current witness independently of the cached attempt")
	}
}
