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
	var identity struct {
		ExperimentVersion    string `json:"experiment_version"`
		SeedAuthority        string `json:"seed_authority"`
		CostVersion          string `json:"cost_version"`
		ReportVersion        string `json:"report_version"`
		NoMatchCap           int    `json:"no_match_bridge_overhead_cap"`
		FixtureBundleByteCap int    `json:"fixture_bundle_byte_cap"`
	}
	if err := json.Unmarshal([]byte(PreregisteredManifestJSON), &identity); err != nil {
		t.Fatal(err)
	}
	if identity.ExperimentVersion != "nogoods/v2" || identity.SeedAuthority != "part3/nogoods/v1" || identity.CostVersion != "nogood-lifecycle-events/v2" || identity.ReportVersion != ReportVersion || identity.NoMatchCap != NoMatchBridgeOverheadCap || identity.FixtureBundleByteCap != FixtureBundleByteCap || PlanCommit != "23ba4ff097c1c6ded9f488eabf4d96d4eccfbea3" {
		t.Fatalf("V2 manifest identity drifted: %#v plan=%s", identity, PlanCommit)
	}
}

func TestDevelopmentEvidenceIsDualDeterministicAndPersistsAllChunks(t *testing.T) {
	evidence, err := buildDevelopmentEvidence("../../domains", "development-test", func(PanelExecution) (PowerEstimate, error) {
		return PowerEstimate{Passing: 4, Replicates: 4, Fraction: Fraction{4, 4}, Authorized: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Report.Classification != "interim" || evidence.Report.Payload.Inference.Classification == "" || len(evidence.ReportJSON) == 0 || len(evidence.Bundle.PrimaryManifest.Policies) != PolicyCount || len(evidence.Bundle.AuditManifest.Policies) != PolicyCount {
		t.Fatalf("incomplete evidence = %#v", evidence.Report)
	}
	root := t.TempDir()
	if err := PersistDevelopmentEvidence(root, evidence); err != nil {
		t.Fatal(err)
	}
	transcriptRoot := filepath.Join(root, ".nous", "nogoods-v2-development-transcripts")
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
	fixtureBytes, err := os.ReadFile(filepath.Join(transcriptRoot, "fixtures.json"))
	if err != nil || !bytes.Equal(fixtureBytes, evidence.FixtureJSON) || digestHex(fixtureBytes) != evidence.Bundle.RootManifest.FixtureBundleSHA256 {
		t.Fatalf("persisted fixture bundle mismatch: %v", err)
	}
	reportBytes, err := os.ReadFile(filepath.Join(root, ".nous", "nogoods-v2-development-report.json"))
	if err != nil || !bytes.Equal(reportBytes, evidence.ReportJSON) {
		t.Fatalf("persisted report mismatch: %v", err)
	}
	if err := PersistDevelopmentEvidence(root, evidence); err == nil {
		t.Fatal("development evidence overwrote an existing attempt")
	}
	if err := verifyEvidenceFiles(root, "development", evidence.Report); err != nil {
		t.Fatalf("persisted evidence did not verify: %v", err)
	}
	chunk := filepath.Join(transcriptRoot, "audit", RequiredPolicies[0]+".ngt.gz")
	corrupt, err := os.ReadFile(chunk)
	if err != nil {
		t.Fatal(err)
	}
	corrupt[len(corrupt)-1] ^= 1
	if err := os.WriteFile(chunk, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyEvidenceFiles(root, "development", evidence.Report); err == nil {
		t.Fatal("evidence verifier accepted a corrupt transcript chunk")
	}
}

func TestTerminalClassificationTableIsClosed(t *testing.T) {
	positive := Inference{Classification: "valid-positive"}
	null := Inference{Classification: "valid-null"}
	authorized := PowerEstimate{Authorized: true}
	if stageClassification("development", positive, PowerEstimate{}) != "valid-null" || stageClassification("development", null, authorized) != "interim" || stageClassification("validation", positive, authorized) != "interim" || stageClassification("validation", null, authorized) != "interim" || stageClassification("locked", positive, authorized) != "valid-positive" || stageClassification("locked", null, authorized) != "valid-null" {
		t.Fatal("stage classification table drifted")
	}
}
