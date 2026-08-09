package transformexp

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestProtectedReportWireIsCanonicalStrictAndDigestBound(t *testing.T) {
	payload := protectedPayload{
		Panel:                "development",
		ImplementationCommit: "0123456789abcdef0123456789abcdef01234567",
		Manifest:             json.RawMessage(PreregisteredManifestJSON),
		FixtureRoot:          digestBytes([]byte("fixture")),
		PrimaryManifest:      digestBytes([]byte("primary")),
		AuditManifest:        digestBytes([]byte("audit")),
		EvidenceGraph:        digestBytes([]byte("graph")),
		Competence:           CompetenceReport{351, 25272, 7020, true},
		CompetenceRoot:       digestBytes([]byte("competence")),
		Rows:                 []PolicyReportRow{{Ordinal: 0, Family: 0, Policy: NousRefine, Terminal: "completed", Work: 1, Applications: 1, SchemaSHA256: digestBytes([]byte("schema")), HeldoutCorrectBits: "ff"}},
		Inference:            transformInference{Point: rationalPoint{1, 1}, Lower: rationalPoint{1, 1}, Upper: rationalPoint{1, 1}, PValue: rationalPoint{1, 100}, NousSuccesses: 1},
		Power:                transformPower{1600, 2000, true},
		Gates:                [12]bool{true, true, true, true, true, true, true, true, true, true, true, true},
		Limitations:          []string{},
	}
	payloadBytes, err := payload.wire()
	if err != nil {
		t.Fatal(err)
	}
	report := protectedReport{"interim-power-authorized", digestBytes(payloadBytes), payload}
	encoded, err := canonicalProtectedReport(report)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeProtectedReport(encoded)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, _ := canonicalProtectedReport(decoded)
	if !bytes.Equal(encoded, reencoded) || encoded[0] != '[' {
		t.Fatal("protected report did not round-trip as canonical array wire")
	}
	mutated := bytes.Replace(encoded, []byte(report.PayloadDigest), []byte(digestBytes([]byte("wrong"))), 1)
	if _, err := decodeProtectedReport(mutated); err == nil {
		t.Fatal("payload digest mutation was accepted")
	}
	if _, err := decodeProtectedReport(append(encoded, '\n')); err == nil {
		t.Fatal("noncanonical trailing whitespace was accepted")
	}
}

func TestProtectedClassificationUsesFrozenLockedCriteria(t *testing.T) {
	passing := transformInference{Point: rationalPoint{13, 128}, Lower: rationalPoint{1, 128}, PValue: rationalPoint{1, 2001}, NousSuccesses: 103, NonmatchingNous: 100, NonmatchingPBE: 100}
	if got, _ := protectedClassification("locked", passing, transformPower{1600, 2000, true}); got != "valid-positive" {
		t.Fatalf("passing classification = %s", got)
	}
	passing.Lower.Numerator = 0
	if got, _ := protectedClassification("locked", passing, transformPower{1600, 2000, true}); got != "valid-null" {
		t.Fatalf("zero lower bound classification = %s", got)
	}
}
