package nogoodexp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPreregisteredManifestIsCanonicalAndExact(t *testing.T) {
	if err := validatePreregisteredManifest(); err != nil {
		t.Fatal(err)
	}
	var reportFragment struct {
		Manifest json.RawMessage `json:"manifest"`
	}
	encoded, err := canonicalJSON(struct {
		Manifest json.RawMessage `json:"manifest"`
	}{preregisteredManifest()})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &reportFragment); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reportFragment.Manifest, []byte(PreregisteredManifestJSON)) {
		t.Fatal("report did not reproduce exact preregistered manifest")
	}
}

func TestDevelopmentEvidenceIsDualDeterministicAndPersistsAllChunks(t *testing.T) {
	evidence, err := BuildDevelopmentEvidence("../../domains", "development-test")
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Report.Classification == "" || len(evidence.ReportJSON) == 0 || len(evidence.Bundle.PrimaryManifest.Policies) != PolicyCount || len(evidence.Bundle.AuditManifest.Policies) != PolicyCount {
		t.Fatalf("incomplete evidence = %#v", evidence.Report)
	}
	root := t.TempDir()
	if err := PersistDevelopmentEvidence(root, evidence); err != nil {
		t.Fatal(err)
	}
	transcriptRoot := filepath.Join(root, ".nous", "nogoods-v1-development-transcripts")
	for _, role := range []string{"primary", "audit"} {
		entries, err := os.ReadDir(filepath.Join(transcriptRoot, role))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != PolicyCount+1 {
			t.Fatalf("%s evidence files = %d", role, len(entries))
		}
	}
	if _, err := os.Stat(filepath.Join(transcriptRoot, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	reportBytes, err := os.ReadFile(filepath.Join(root, ".nous", "nogoods-v1-development-report.json"))
	if err != nil || !bytes.Equal(reportBytes, evidence.ReportJSON) {
		t.Fatalf("persisted report mismatch: %v", err)
	}
	if err := PersistDevelopmentEvidence(root, evidence); err == nil {
		t.Fatal("development evidence overwrote an existing attempt")
	}
}
