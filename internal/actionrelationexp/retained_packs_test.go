package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/chazu/nous/internal/actionrelationledger"
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
	authority := retainedRunAuthority{curriculum: 0, phase: 2, workTerminal: zeroObjectDigest, work: [12]int{9: 1}, initialWork: [12]int{0: 2}, terminal: "completed"}
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
