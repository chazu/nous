package actionrelationexp

import (
	"bytes"
	"testing"
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
