package actionrelationexp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/actionrelationsearch"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestRetainedManifestParsersRoundTripCanonicalRoots(t *testing.T) {
	object, err := BuildObjectBundle(ObjectScope{Curriculum: 3, Class: "authority"}, []ObjectRecord{{Kind: 1, Bytes: objectWire(1, "state")}})
	if err != nil {
		t.Fatal(err)
	}
	objectRoot, _ := object.ObjectRoot.CanonicalJSON()
	indexRoot, _ := object.IndexRoot.CanonicalJSON()
	if parsed, err := ParseObjectPackRoot(objectRoot); err != nil || !bytes.Equal(mustCanonicalObjectRoot(parsed), objectRoot) {
		t.Fatalf("object root did not round trip: %v", err)
	}
	if parsed, err := ParseIndexRoot(indexRoot); err != nil || !bytes.Equal(mustCanonicalIndexRoot(parsed), indexRoot) {
		t.Fatalf("index root did not round trip: %v", err)
	}

	transcript, err := BuildTranscript(testDigest("retained-parser")[:32], []ChargedCall{{Phase: 2, Operation: 13, Status: 1, SourceTaskDigest: testAuthorityDigest("task"), Payload: []any{"certificate-applicable", testAuthorityDigest("state"), testAuthorityDigest("occurrence")}, OutputDigests: []string{testAuthorityDigest("output")}}})
	if err != nil {
		t.Fatal(err)
	}
	journalRoot, _ := transcript.JournalRoot.CanonicalJSON()
	if parsed, err := ParseTranscriptRoot(journalRoot); err != nil || !bytes.Equal(mustCanonicalTranscriptRoot(parsed), journalRoot) {
		t.Fatalf("transcript root did not round trip: %v", err)
	}

	tableRow := make([]byte, tableRecordSizes[102])
	copy(tableRow[0:32], bytes.Repeat([]byte{1}, 32))
	copy(tableRow[32:64], bytes.Repeat([]byte{2}, 32))
	tableRow[64], tableRow[65] = 1, 1
	table, err := BuildTableBundle(3, "no-guard", 102, [][]byte{tableRow})
	if err != nil {
		t.Fatal(err)
	}
	tableRoot, _ := table.Manifest.CanonicalJSON()
	if parsed, err := ParseTableManifest(tableRoot); err != nil || !bytes.Equal(mustCanonicalTableManifest(parsed), tableRoot) {
		t.Fatalf("table manifest did not round trip: %v", err)
	}
}

func TestRetainedManifestParsersRejectNoncanonicalWire(t *testing.T) {
	object, err := BuildObjectBundle(ObjectScope{Curriculum: 0, Class: "authority"}, []ObjectRecord{{Kind: 1, Bytes: objectWire(1, "state")}})
	if err != nil {
		t.Fatal(err)
	}
	canonical, _ := object.ObjectRoot.CanonicalJSON()
	corrupt := append([]byte(" "), canonical...)
	if _, err := ParseObjectPackRoot(corrupt); err == nil {
		t.Fatal("accepted noncanonical object-root whitespace")
	}
}

func TestRetainedRunReplayClosesReservationsTypesAndWork(t *testing.T) {
	runID := testDigest("retained-run-replay")[:32]
	left, _ := json.Marshal([]any{"finite-action-state/v1", []any{[]any{"c0", 0}}, []any{}})
	right, _ := json.Marshal([]any{"finite-action-state/v1", []any{[]any{"c0", 1}}, []any{}})
	leftDigest, rightDigest := shaHex(left), shaHex(right)
	equality, _ := json.Marshal([]any{"action-state-equality-row/v1", leftDigest, rightDigest, false, "valid"})
	equalityDigest := shaHex(equality)
	reservation, err := actionrelationledger.BuildReservation(runID, testAuthorityDigest("utility-task"), []uint8{14}, 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	transcript, err := BuildTranscript(runID, []ChargedCall{{
		Phase: 2, Operation: 14, Status: 1, SourceTaskDigest: reservation.Digest,
		Payload: []any{"certificate-equality", leftDigest, rightDigest}, OutputDigests: []string{equalityDigest},
	}})
	if err != nil {
		t.Fatal(err)
	}
	calls, err := decodeRetainedCalls(transcript)
	if err != nil {
		t.Fatal(err)
	}
	objects := map[string]retainedObjectValue{
		leftDigest:         {kind: 1, canonical: left},
		rightDigest:        {kind: 1, canonical: right},
		equalityDigest:     {kind: 40, canonical: equality},
		reservation.Digest: {kind: 27, canonical: reservation.Canonical},
	}
	authority := retainedRunAuthority{curriculum: 0, phase: 2, workTerminal: zeroObjectDigest, work: [12]int{9: 1}, initialWork: [12]int{0: 2}, terminal: "completed", policy: "complete", worldOrdinal: 5, remaining: 1_999_997}
	record := RunEvidenceRecord{RunID: runID}
	if err := verifyRetainedRunReplay(record, authority, calls, objects, nil, nil); err != nil {
		t.Fatal(err)
	}

	partial, err := actionrelationledger.BuildReservation(runID, testAuthorityDigest("partial-task"), []uint8{14, 14}, 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	partialCalls := slicesCloneCalls(calls)
	partialCalls[0].source = partial.Digest
	partialObjects := cloneRetainedObjects(objects)
	partialObjects[partial.Digest] = retainedObjectValue{kind: 27, canonical: partial.Canonical}
	if err := verifyRetainedRunReplay(record, authority, partialCalls, partialObjects, nil, nil); err == nil {
		t.Fatal("accepted partially consumed compound reservation")
	}

	delete(objects, rightDigest)
	if err := verifyRetainedRunReplay(record, authority, calls, objects, nil, nil); err == nil {
		t.Fatal("accepted payload digest without its named typed leaf")
	}
}

func TestCertificateAttemptFinalizationMatchesDigestSortedPair(t *testing.T) {
	state := testAuthorityDigest("state")
	left := testAuthorityDigest("left")
	right := testAuthorityDigest("right")
	root := testAuthorityDigest("root")
	minimum, maximum := left, right
	if minimum > maximum {
		minimum, maximum = maximum, minimum
	}
	wire, _ := json.Marshal([]any{"cache-finalization", testAuthorityDigest("world"), "static-rw-sleep", state, minimum, maximum, testAuthorityDigest("miss"), testAuthorityDigest("attempt"), root})
	var payload []json.RawMessage
	if err := json.Unmarshal(wire, &payload); err != nil {
		t.Fatal(err)
	}
	attempt := decodedCertificateAttempt{State: state, A: maximum, B: minimum, Operation: root}
	if !certificateAttemptMatchesFinalization(attempt, payload, root) {
		t.Fatal("rejected canonical attempt orientation differing from digest-sorted cache pair")
	}
	attempt.B = testAuthorityDigest("other")
	if certificateAttemptMatchesFinalization(attempt, payload, root) {
		t.Fatal("accepted different certificate attempt pair")
	}
}

func TestRetainedPartialSearchGraphRequiresExactSemanticPrefix(t *testing.T) {
	authority, calls, objects, structural := retainedPartialSearchFixture(t, 0)
	if err := verifyRetainedPartialSearchGraph(authority, calls, objects, structural, nil); err != nil {
		t.Fatalf("valid partial DFS prefix: %v", err)
	}
	if err := verifyRetainedStructuralCompleteness(authority, calls, objects, nil, structural, nil); err != nil {
		t.Fatalf("valid partial structural set: %v", err)
	}

	_, nonPrefixCalls, nonPrefixObjects, nonPrefixStructural := retainedPartialSearchFixture(t, 1)
	if err := verifyRetainedPartialSearchGraph(authority, nonPrefixCalls, nonPrefixObjects, nonPrefixStructural, nil); err == nil {
		t.Fatal("accepted a completed later sibling without the semantic first sibling")
	}

	if err := verifyRetainedPartialSearchGraph(authority, calls[:len(calls)-1], objects, structural, nil); err == nil {
		t.Fatal("accepted a retained terminal without its charged construction call")
	}
	wrongRoot := authority
	wrongRoot.initialState = testAuthorityDigest("wrong-fixture-state")
	if err := verifyRetainedPartialSearchGraph(wrongRoot, calls, objects, structural, nil); err == nil {
		t.Fatal("accepted a partial DFS rooted outside the fixture initial state")
	}
	extraObjects := cloneRetainedObjects(objects)
	extraStructural := mapsCloneBool(structural)
	extraState, _ := (actionrelations.State{Cells: []actionrelations.Cell{{Name: "extra", Value: 0}}}).CanonicalJSON()
	extraDigest := shaHex(extraState)
	extraObjects[extraDigest] = retainedObjectValue{kind: 1, canonical: extraState}
	extraStructural[fmt.Sprintf("1:%s", extraDigest)] = true
	if err := verifyRetainedStructuralCompleteness(authority, calls, extraObjects, nil, extraStructural, nil); err == nil {
		t.Fatal("accepted an attributed but unproduced utility object")
	}
}

func TestRetainedStructuralCompletenessOwnsCurrentAndTailWitnessesExactly(t *testing.T) {
	authority, calls, objects, structural := retainedPartialSearchFixture(t, 0)
	authority.policy = "static-rw-sleep"
	world := authority.world
	state := authority.initialState
	taken, sleeper := authority.initialOccurrences[0], authority.initialOccurrences[1]
	footprint, _ := json.Marshal([]any{
		"action-static-footprint-row/v1", world, testAuthorityDigest("current-witness-node"), state, taken, sleeper,
		testAuthorityDigest("current-witness-a-facts"), testAuthorityDigest("current-witness-b-facts"), true, "valid",
	})
	if err := ValidateObject(48, footprint); err != nil {
		t.Fatal(err)
	}
	footprintDigest := shaHex(footprint)
	objects[footprintDigest] = retainedObjectValue{kind: 48, canonical: footprint}
	witness, _ := json.Marshal([]any{"static-witness/v1", footprintDigest})
	witnessDigest := shaHex(witness)
	objects[witnessDigest] = retainedObjectValue{kind: 15, canonical: witness}
	witnessKey := fmt.Sprintf("15:%s", witnessDigest)
	structural[witnessKey] = true
	calls = append(calls, retainedCall{operation: 24, outputs: []string{footprintDigest}})

	completed := &retainedDFSWitnessAuthority{current: map[string]bool{witnessKey: true}, completed: map[string]bool{witnessKey: true}}
	if err := verifyRetainedStructuralCompleteness(authority, calls, objects, nil, structural, completed); err != nil {
		t.Fatalf("completed cache-hit witness was not owned: %v", err)
	}
	tail := &retainedDFSWitnessAuthority{current: map[string]bool{witnessKey: true}, completed: map[string]bool{}}
	if err := verifyRetainedStructuralCompleteness(authority, calls, objects, nil, structural, tail); err != nil {
		t.Fatalf("unique budget tail witness was not owned: %v", err)
	}
	completedRun := authority
	completedRun.terminal = "completed"
	if err := verifyRetainedStructuralCompleteness(completedRun, calls, objects, nil, structural, tail); err == nil {
		t.Fatal("accepted an incomplete tail witness in a completed run")
	}
	if err := verifyRetainedStructuralCompleteness(authority, calls, objects, nil, structural, &retainedDFSWitnessAuthority{current: map[string]bool{}, completed: map[string]bool{}}); err == nil {
		t.Fatal("accepted a valid-looking witness absent from ordered DFS authority")
	}
}

func TestRetainedRunObjectsPreservePhysicalScope(t *testing.T) {
	stateCanonical, _ := (actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}}}).CanonicalJSON()
	stateDigest := shaHex(stateCanonical)
	truthCanonical, _ := json.Marshal([]any{"action-scorer-truth-shard/v1", testAuthorityDigest("world")})
	truthDigest := shaHex(truthCanonical)
	reservationCanonical, _ := json.Marshal([]any{"compound-work-reservation/v1", testAuthorityDigest("reservation")})
	reservationDigest := shaHex(reservationCanonical)
	foreignDigest := testAuthorityDigest("foreign")
	scopes := map[string]map[string]retainedObjectValue{
		"authority": {
			stateDigest: {kind: 1, canonical: stateCanonical},
			truthDigest: {kind: 29, canonical: truthCanonical},
		},
		"utility": {
			reservationDigest: {kind: 27, canonical: reservationCanonical},
		},
		"acquisition-nous-preboundary": {
			foreignDigest: {kind: 1, canonical: stateCanonical},
		},
	}
	objects, _, err := retainedObjectsForRun(retainedRunAuthority{phase: 2, policy: "complete"}, scopes)
	if err != nil {
		t.Fatal(err)
	}
	if objects[stateDigest].kind != 1 || objects[reservationDigest].kind != 27 {
		t.Fatal("run lost an authorized fixture or utility object")
	}
	if _, ok := objects[truthDigest]; ok {
		t.Fatal("utility replay gained scorer-truth authority")
	}
	if _, ok := objects[foreignDigest]; ok {
		t.Fatal("utility replay crossed into an unrelated acquisition scope")
	}
	otherStateCanonical, _ := (actionrelations.State{Cells: []actionrelations.Cell{{Name: "other", Value: 1}}}).CanonicalJSON()
	otherStateDigest := shaHex(otherStateCanonical)
	curricula := []map[string]map[string]retainedObjectValue{
		scopes,
		{"authority": {otherStateDigest: {kind: 1, canonical: otherStateCanonical}}},
	}
	objects, _, err = retainedObjectsForRun(retainedRunAuthority{curriculum: 0, phase: 2, policy: "complete"}, curricula[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := objects[otherStateDigest]; ok {
		t.Fatal("utility replay borrowed an authority object from another curriculum")
	}
}

func TestRetainedAcquisitionTableOutputsRequireExactScopeKindAndSlot(t *testing.T) {
	record := make([]byte, tableRecordSizes[103])
	copy(record[0:32], bytes.Repeat([]byte{1}, 32))
	copy(record[32:64], bytes.Repeat([]byte{3}, 32))
	copy(record[64:96], bytes.Repeat([]byte{2}, 32))
	binary.BigEndian.PutUint16(record[96:98], 7)
	record[99] = 1
	if err := ValidateTableRecord(103, record); err != nil {
		t.Fatal(err)
	}
	leafDigest := TableLeafDigest(103, 7, record)
	digest := fmt.Sprintf("%x", leafDigest)
	authority := retainedRunAuthority{curriculum: 2, phase: 1, policy: "nous"}
	call := retainedCall{operation: 1, outputs: []string{testAuthorityDigest("guard"), digest}}
	tables := map[string][]retainedTableLeaf{
		digest: {{kind: 103, curriculum: 2, scope: "nous", ordinal: 7, record: record}},
	}
	if !retainedAcquisitionTableOutput(authority, call, 1, digest, tables) {
		t.Fatal("rejected exact acquisition-bound table output")
	}
	for name, mutate := range map[string]func(retainedRunAuthority, retainedCall, map[string][]retainedTableLeaf) (retainedRunAuthority, retainedCall, map[string][]retainedTableLeaf){
		"wrong curriculum": func(a retainedRunAuthority, c retainedCall, m map[string][]retainedTableLeaf) (retainedRunAuthority, retainedCall, map[string][]retainedTableLeaf) {
			a.curriculum++
			return a, c, m
		},
		"wrong scope": func(a retainedRunAuthority, c retainedCall, m map[string][]retainedTableLeaf) (retainedRunAuthority, retainedCall, map[string][]retainedTableLeaf) {
			a.policy = "no-guard"
			return a, c, m
		},
		"wrong kind": func(a retainedRunAuthority, c retainedCall, m map[string][]retainedTableLeaf) (retainedRunAuthority, retainedCall, map[string][]retainedTableLeaf) {
			m[digest][0].kind = 104
			return a, c, m
		},
		"wrong operation": func(a retainedRunAuthority, c retainedCall, m map[string][]retainedTableLeaf) (retainedRunAuthority, retainedCall, map[string][]retainedTableLeaf) {
			c.operation = 2
			return a, c, m
		},
		"wrong slot": func(a retainedRunAuthority, c retainedCall, m map[string][]retainedTableLeaf) (retainedRunAuthority, retainedCall, map[string][]retainedTableLeaf) {
			return a, c, m
		},
		"duplicate leaf": func(a retainedRunAuthority, c retainedCall, m map[string][]retainedTableLeaf) (retainedRunAuthority, retainedCall, map[string][]retainedTableLeaf) {
			m[digest] = append(m[digest], m[digest][0])
			return a, c, m
		},
	} {
		t.Run(name, func(t *testing.T) {
			copyTables := map[string][]retainedTableLeaf{digest: slices.Clone(tables[digest])}
			a, c, m := mutate(authority, call, copyTables)
			index := 1
			if name == "wrong slot" {
				index = 0
			}
			if retainedAcquisitionTableOutput(a, c, index, digest, m) {
				t.Fatal("accepted table output outside its exact physical boundary")
			}
		})
	}
}

func TestRetainedBudgetReplayUsesExactFrozenCrossing(t *testing.T) {
	runID := testDigest("retained-budget-crossing")[:32]
	cap := 4_096
	before := cap - 2
	rejected, err := actionrelationledger.BuildReservation(runID, testAuthorityDigest("rejected-task"), []uint8{11, 11}, before, cap)
	if err != nil || rejected.Status != "rejected-cap" {
		t.Fatalf("rejected reservation: %v", err)
	}
	terminalReservation, err := actionrelationledger.BuildTerminalReservation(runID, testAuthorityDigest("terminal-task"), before, cap)
	if err != nil {
		t.Fatal(err)
	}
	work := [12]int{}
	work[0], work[11] = before, 1
	terminalWire, _ := json.Marshal([]any{"action-work-terminal/v1", runID, 2, rejected.Digest, "budget-exhausted", work, before + 1, 0})
	terminalDigest := shaHex(terminalWire)
	payloadWire, _ := json.Marshal([]any{"budget-terminal", rejected.Digest})
	var payload []json.RawMessage
	_ = json.Unmarshal(payloadWire, &payload)
	calls := []retainedCall{{sequence: 0, phase: 2, operation: 19, counter: 12, status: 1, source: terminalReservation.Digest, payload: payload, outputs: []string{terminalDigest}}}
	objects := map[string]retainedObjectValue{
		rejected.Digest:            {kind: 27, canonical: rejected.Canonical},
		terminalReservation.Digest: {kind: 27, canonical: terminalReservation.Canonical},
		terminalDigest:             {kind: 49, canonical: terminalWire},
	}
	authority := retainedRunAuthority{
		phase: 2, terminal: "budget-exhausted", policy: "complete", worldOrdinal: 5,
		workTerminal: terminalDigest, work: [12]int{11: 1}, initialWork: [12]int{0: before},
		terminalSet: zeroObjectDigest, remaining: 0,
	}
	record := RunEvidenceRecord{RunID: runID, WorkTerminal: terminalDigest}
	if err := verifyRetainedRunReplay(record, authority, calls, objects, nil, nil); err != nil {
		t.Fatalf("exact cap crossing: %v", err)
	}

	earlyWire, _ := json.Marshal([]any{"compound-work-reservation/v1", runID, testAuthorityDigest("early-task"), []uint8{11}, before, before, "rejected-cap"})
	earlyDigest := shaHex(earlyWire)
	earlyTerminalWire, _ := json.Marshal([]any{"action-work-terminal/v1", runID, 2, earlyDigest, "budget-exhausted", work, before + 1, 0})
	earlyTerminalDigest := shaHex(earlyTerminalWire)
	earlyPayloadWire, _ := json.Marshal([]any{"budget-terminal", earlyDigest})
	_ = json.Unmarshal(earlyPayloadWire, &calls[0].payload)
	calls[0].outputs = []string{earlyTerminalDigest}
	earlyObjects := cloneRetainedObjects(objects)
	earlyObjects[earlyDigest] = retainedObjectValue{kind: 27, canonical: earlyWire}
	earlyObjects[earlyTerminalDigest] = retainedObjectValue{kind: 49, canonical: earlyTerminalWire}
	authority.workTerminal = earlyTerminalDigest
	record.WorkTerminal = earlyTerminalDigest
	if err := verifyRetainedRunReplay(record, authority, calls, earlyObjects, nil, nil); err == nil {
		t.Fatal("accepted a rejected-cap terminal before the frozen crossing")
	}
}

func retainedPartialSearchFixture(t *testing.T, selected int) (retainedRunAuthority, []retainedCall, map[string]retainedObjectValue, map[string]bool) {
	t.Helper()
	state := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 0}}}
	occurrences, err := actionrelations.AssignOccurrences([]actionrelations.SemanticAction{
		{Kind: "add", XRole: "c0", N: 1},
		{Kind: "add", XRole: "c1", N: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(occurrences, func(a, b actionrelations.Occurrence) int {
		aBytes, _ := a.CanonicalJSON()
		bBytes, _ := b.CanonicalJSON()
		return bytes.Compare(aBytes, bBytes)
	})
	if selected < 0 || selected >= len(occurrences) {
		t.Fatal("invalid selected occurrence")
	}
	taken, other := occurrences[selected], occurrences[1-selected]
	takenDigest, _ := taken.Digest()
	otherDigest, _ := other.Digest()

	emptyProof, _ := actionrelationsearch.BuildProofMap(nil)
	rootRemaining, err := actionrelationsearch.BuildRemaining(occurrences)
	if err != nil {
		t.Fatal(err)
	}
	rootNode, err := actionrelationsearch.BuildSearchNode(state, rootRemaining, emptyProof)
	if err != nil {
		t.Fatal(err)
	}
	childState, outcome, err := actionrelations.Apply(state, taken.Action)
	if err != nil || outcome != "applied" {
		t.Fatalf("root transition: %s %v", outcome, err)
	}
	childRemaining, err := actionrelationsearch.BuildRemaining([]actionrelations.Occurrence{other})
	if err != nil {
		t.Fatal(err)
	}
	childNode, err := actionrelationsearch.BuildSearchNode(childState, childRemaining, emptyProof)
	if err != nil {
		t.Fatal(err)
	}
	terminalState, outcome, err := actionrelations.Apply(childState, other.Action)
	if err != nil || outcome != "applied" {
		t.Fatalf("child transition: %s %v", outcome, err)
	}
	emptyRemaining, err := actionrelationsearch.BuildRemaining(nil)
	if err != nil {
		t.Fatal(err)
	}
	terminalNode, err := actionrelationsearch.BuildSearchNode(terminalState, emptyRemaining, emptyProof)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := actionrelationsearch.BuildTerminalBehavior(terminalState, nil)
	if err != nil {
		t.Fatal(err)
	}
	terminalSet, err := actionrelationsearch.BuildTerminalSet([]string{terminal.Digest})
	if err != nil {
		t.Fatal(err)
	}
	terminalSubtree, err := actionrelationsearch.BuildSubtreeRoot(terminalNode.Digest, nil)
	if err != nil {
		t.Fatal(err)
	}
	childEdge, err := actionrelationsearch.BuildSearchEdge(childNode.Digest, otherDigest, nil, terminalNode.Digest)
	if err != nil {
		t.Fatal(err)
	}
	childCompletion, err := actionrelationsearch.BuildCompletedSubtree(childNode.Digest, otherDigest, childEdge, terminalSubtree, terminalSet)
	if err != nil {
		t.Fatal(err)
	}
	childSubtree, err := actionrelationsearch.BuildSubtreeRoot(childNode.Digest, []string{childCompletion.Digest})
	if err != nil {
		t.Fatal(err)
	}
	rootEdge, err := actionrelationsearch.BuildSearchEdge(rootNode.Digest, takenDigest, nil, childNode.Digest)
	if err != nil {
		t.Fatal(err)
	}
	rootCompletion, err := actionrelationsearch.BuildCompletedSubtree(rootNode.Digest, takenDigest, rootEdge, childSubtree, terminalSet)
	if err != nil {
		t.Fatal(err)
	}

	objects := map[string]retainedObjectValue{}
	structural := map[string]bool{}
	addBytes := func(kind uint16, canonical []byte) string {
		digest := shaHex(canonical)
		objects[digest] = retainedObjectValue{kind: kind, canonical: canonical}
		structural[fmt.Sprintf("%d:%s", kind, digest)] = true
		return digest
	}
	addEvidence := func(kind uint16, value actionrelationsearch.EvidenceObject) {
		addBytes(kind, value.Canonical)
	}
	for _, value := range []actionrelations.State{state, childState, terminalState} {
		canonical, _ := value.CanonicalJSON()
		addBytes(1, canonical)
	}
	for _, value := range occurrences {
		canonical, _ := value.CanonicalJSON()
		addBytes(3, canonical)
	}
	for _, value := range []actionrelationsearch.EvidenceObject{rootRemaining, childRemaining, emptyRemaining} {
		addEvidence(5, value)
	}
	addEvidence(19, emptyProof)
	for _, value := range []actionrelationsearch.EvidenceObject{rootNode, childNode, terminalNode} {
		addEvidence(20, value)
	}
	for _, value := range []actionrelationsearch.EvidenceObject{rootEdge, childEdge} {
		addEvidence(21, value)
	}
	for _, value := range []actionrelationsearch.EvidenceObject{rootCompletion, childCompletion} {
		addEvidence(22, value)
	}
	addEvidence(23, terminal)
	addEvidence(24, terminalSet)
	for _, value := range []actionrelationsearch.EvidenceObject{childSubtree, terminalSubtree} {
		addEvidence(25, value)
	}
	for digest, object := range objects {
		if slices.Contains([]uint16{1, 3, 20, 23}, object.kind) {
			delete(structural, fmt.Sprintf("%d:%s", object.kind, digest))
		}
	}

	stateDigest, _ := state.Digest()
	childStateDigest, _ := childState.Digest()
	terminalStateDigest, _ := terminalState.Digest()
	semanticDigests := make([]string, len(occurrences))
	for index, occurrence := range occurrences {
		semanticDigests[index], _ = occurrence.Digest()
	}
	payload := func(values ...any) []json.RawMessage {
		wire, _ := json.Marshal(values)
		var result []json.RawMessage
		_ = json.Unmarshal(wire, &result)
		return result
	}
	calls := []retainedCall{
		{sequence: 0, operation: 16, payload: payload("search-node-lookup", stateDigest, rootRemaining.Digest, emptyProof.Digest), outputs: []string{rootNode.Digest}},
		{sequence: 1, operation: 23, payload: payload("search-applicable", testAuthorityDigest("world"), "complete", rootNode.Digest, stateDigest, semanticDigests[0])},
		{sequence: 2, operation: 23, payload: payload("search-applicable", testAuthorityDigest("world"), "complete", rootNode.Digest, stateDigest, semanticDigests[1])},
		{sequence: 3, operation: 16, payload: payload("search-node-lookup", childStateDigest, childRemaining.Digest, emptyProof.Digest), outputs: []string{childNode.Digest}},
		{sequence: 4, operation: 23, payload: payload("search-applicable", testAuthorityDigest("world"), "complete", childNode.Digest, childStateDigest, otherDigest)},
		{sequence: 5, operation: 16, payload: payload("search-node-lookup", terminalStateDigest, emptyRemaining.Digest, emptyProof.Digest), outputs: []string{terminalNode.Digest}},
		{sequence: 6, operation: 19, payload: payload("terminal-construct"), outputs: []string{terminal.Digest}},
	}
	callIDs := make([]string, len(calls))
	for index := range calls {
		calls[index].callID = testAuthorityDigest(fmt.Sprintf("partial-call-%d-%d", selected, index))
		callIDs[index] = calls[index].callID
	}
	runRoot, err := BuildOperationRange(testDigest(fmt.Sprintf("partial-run-%d", selected))[:32], 2, 0, callIDs)
	if err != nil {
		t.Fatal(err)
	}
	addBytes(46, runRoot.Canonical)
	initialOccurrences := slices.Clone(semanticDigests)
	slices.Sort(initialOccurrences)
	authority := retainedRunAuthority{terminal: "budget-exhausted", terminalSet: zeroObjectDigest, policy: "complete", world: testAuthorityDigest("world"), operationRoot: runRoot.Digest, initialState: stateDigest, initialOccurrences: initialOccurrences}
	return authority, calls, objects, structural
}

func mapsCloneBool(values map[string]bool) map[string]bool {
	result := make(map[string]bool, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func slicesCloneCalls(values []retainedCall) []retainedCall {
	result := make([]retainedCall, len(values))
	copy(result, values)
	return result
}

func cloneRetainedObjects(values map[string]retainedObjectValue) map[string]retainedObjectValue {
	result := make(map[string]retainedObjectValue, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func mustCanonicalObjectRoot(value ObjectPackRoot) []byte {
	data, _ := value.CanonicalJSON()
	return data
}
func mustCanonicalIndexRoot(value IndexRoot) []byte { data, _ := value.CanonicalJSON(); return data }
func mustCanonicalTranscriptRoot(value TranscriptRoot) []byte {
	data, _ := value.CanonicalJSON()
	return data
}
func mustCanonicalTableManifest(value TableManifest) []byte {
	data, _ := value.CanonicalJSON()
	return data
}
