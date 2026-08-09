package nogoodexp

import (
	"fmt"
	"os"
	"path/filepath"
)

type ChunkManifest struct {
	Policy        string `json:"policy"`
	RawSize       int    `json:"raw_size"`
	RawSHA256     string `json:"raw_sha256"`
	GzipSize      int    `json:"gzip_size"`
	GzipSHA256    string `json:"gzip_sha256"`
	EventCount    int64  `json:"event_count"`
	FirstSequence int64  `json:"first_sequence"`
	LastSequence  int64  `json:"last_sequence"`
}

type ExecutionManifest struct {
	ExecutionRole         string          `json:"execution_role"`
	AcquisitionEventCount int64           `json:"acquisition_event_count"`
	AcquisitionVector     [12]int64       `json:"acquisition_vector"`
	AcquisitionWork       int64           `json:"acquisition_work"`
	Policies              []ChunkManifest `json:"policies"`
}

type RootManifest struct {
	Panel                  string `json:"panel"`
	PrimaryExecutionSHA256 string `json:"primary_execution_sha256"`
	AuditExecutionSHA256   string `json:"audit_execution_sha256"`
	FixtureBundleSize      int    `json:"fixture_bundle_size"`
	FixtureBundleSHA256    string `json:"fixture_bundle_sha256"`
	ReportPayloadSHA256    string `json:"report_payload_sha256"`
	FinalReportReference   string `json:"final_report_reference"`
}

type EvidenceBundle struct {
	PrimaryManifest     ExecutionManifest
	AuditManifest       ExecutionManifest
	PrimaryManifestJSON []byte
	AuditManifestJSON   []byte
	RootManifest        RootManifest
	RootManifestJSON    []byte
}

func buildEvidenceBundle(panel string, fixtureJSON []byte, primary, audit PanelExecution, payloadDigest string) (EvidenceBundle, error) {
	if _, err := decodeFixtureBundle(panel, fixtureJSON); err != nil {
		return EvidenceBundle{}, fmt.Errorf("fixture evidence: %w", err)
	}
	primaryManifest, err := executionManifest(primary)
	if err != nil {
		return EvidenceBundle{}, err
	}
	auditManifest, err := executionManifest(audit)
	if err != nil {
		return EvidenceBundle{}, err
	}
	if totalManifestEvents(primaryManifest)+totalManifestEvents(auditManifest) > 16000000 || totalManifestRaw(primaryManifest)+totalManifestRaw(auditManifest) > 2147483648 || totalManifestGzip(primaryManifest)+totalManifestGzip(auditManifest) > 2148000000 {
		return EvidenceBundle{}, fmt.Errorf("%s transcript bundle exceeds a hard cap", panel)
	}
	primaryJSON, err := canonicalJSON(primaryManifest)
	if err != nil {
		return EvidenceBundle{}, err
	}
	auditJSON, err := canonicalJSON(auditManifest)
	if err != nil {
		return EvidenceBundle{}, err
	}
	root := RootManifest{
		Panel: panel, PrimaryExecutionSHA256: digestHex(primaryJSON), AuditExecutionSHA256: digestHex(auditJSON),
		FixtureBundleSize: len(fixtureJSON), FixtureBundleSHA256: digestHex(fixtureJSON),
		ReportPayloadSHA256: payloadDigest, FinalReportReference: fmt.Sprintf("nogoods-v2-%s-report.json", panel),
	}
	rootJSON, err := canonicalJSON(root)
	if err != nil {
		return EvidenceBundle{}, err
	}
	return EvidenceBundle{primaryManifest, auditManifest, primaryJSON, auditJSON, root, rootJSON}, nil
}

func executionManifest(execution PanelExecution) (ExecutionManifest, error) {
	manifest := ExecutionManifest{ExecutionRole: execution.Role, AcquisitionVector: execution.AcquisitionVector, AcquisitionWork: execution.AcquisitionWork}
	for _, policy := range execution.Policies {
		decoded, err := DecodeTranscript(policy.Transcript.Raw)
		if err != nil || decoded.Vector != policy.Transcript.Vector {
			return ExecutionManifest{}, fmt.Errorf("decode %s transcript: %w", policy.Policy, err)
		}
		eventCount := int64(0)
		for _, count := range decoded.Vector {
			eventCount += count
		}
		last := int64(-1)
		if eventCount > 0 {
			last = eventCount - 1
		}
		manifest.Policies = append(manifest.Policies, ChunkManifest{
			Policy: policy.Policy, RawSize: len(policy.Transcript.Raw), RawSHA256: digestHex(policy.Transcript.Raw),
			GzipSize: len(policy.Transcript.Gzip), GzipSHA256: digestHex(policy.Transcript.Gzip),
			EventCount: eventCount, FirstSequence: 0, LastSequence: last,
		})
		if policy.Policy == "nous-generalized" {
			manifest.AcquisitionEventCount = execution.AcquisitionWork
		}
	}
	if len(manifest.Policies) != PolicyCount {
		return ExecutionManifest{}, fmt.Errorf("execution manifest contains %d policies", len(manifest.Policies))
	}
	if totalManifestEvents(manifest) > 8000000 || totalManifestRaw(manifest) > 1073741824 || totalManifestGzip(manifest) > 1074000000 {
		return ExecutionManifest{}, fmt.Errorf("%s execution transcript exceeds a hard cap", execution.Role)
	}
	return manifest, nil
}

func totalManifestEvents(manifest ExecutionManifest) int64 {
	var total int64
	for _, policy := range manifest.Policies {
		total += policy.EventCount
	}
	return total
}

func totalManifestRaw(manifest ExecutionManifest) int64 {
	var total int64
	for _, policy := range manifest.Policies {
		total += int64(policy.RawSize)
	}
	return total
}

func totalManifestGzip(manifest ExecutionManifest) int64 {
	var total int64
	for _, policy := range manifest.Policies {
		total += int64(policy.GzipSize)
	}
	return total
}

func PersistDevelopmentEvidence(repoRoot string, evidence DevelopmentEvidence) error {
	return persistEvidence(repoRoot, "development", evidence, false)
}

func persistEvidence(repoRoot, panel string, evidence DevelopmentEvidence, transcriptRootClaimed bool) error {
	base := filepath.Join(repoRoot, ".nous")
	reportPath := filepath.Join(base, fmt.Sprintf("nogoods-v2-%s-report.json", panel))
	transcriptRoot := filepath.Join(base, fmt.Sprintf("nogoods-v2-%s-transcripts", panel))
	if err := requireAbsent(reportPath); err != nil {
		return err
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	if transcriptRootClaimed {
		info, err := os.Lstat(transcriptRoot)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("claimed transcript root is invalid: %w", err)
		}
		entries, err := os.ReadDir(transcriptRoot)
		if err != nil || len(entries) != 1 || entries[0].Name() != "fixtures.json" || entries[0].IsDir() {
			return fmt.Errorf("claimed transcript root lacks its sole fixture bundle: %w", err)
		}
		materialized, err := os.ReadFile(filepath.Join(transcriptRoot, "fixtures.json"))
		if err != nil || !bytesEqual(materialized, evidence.FixtureJSON) {
			return fmt.Errorf("claimed fixture bundle changed: %w", err)
		}
	} else {
		if err := requireAbsent(transcriptRoot); err != nil {
			return err
		}
		if err := os.Mkdir(transcriptRoot, 0o755); err != nil {
			return err
		}
		if err := writeExclusive(filepath.Join(transcriptRoot, "fixtures.json"), evidence.FixtureJSON, 0o644); err != nil {
			return err
		}
	}
	for _, execution := range []PanelExecution{evidence.Primary, evidence.Audit} {
		roleDir := filepath.Join(transcriptRoot, execution.Role)
		if err := os.Mkdir(roleDir, 0o755); err != nil {
			return err
		}
		for _, policy := range execution.Policies {
			if err := writeExclusive(filepath.Join(roleDir, policy.Policy+".ngt.gz"), policy.Transcript.Gzip, 0o644); err != nil {
				return err
			}
		}
		manifestJSON := evidence.Bundle.PrimaryManifestJSON
		if execution.Role == "audit" {
			manifestJSON = evidence.Bundle.AuditManifestJSON
		}
		if err := writeExclusive(filepath.Join(roleDir, "execution-manifest.json"), manifestJSON, 0o644); err != nil {
			return err
		}
		if err := syncDirectory(roleDir); err != nil {
			return err
		}
	}
	if len(evidence.FixtureJSON) == 0 || len(evidence.FixtureJSON) > FixtureBundleByteCap || digestHex(evidence.FixtureJSON) != evidence.Bundle.RootManifest.FixtureBundleSHA256 {
		return fmt.Errorf("fixture bundle does not match root manifest")
	}
	if err := writeExclusive(filepath.Join(transcriptRoot, "manifest.json"), evidence.Bundle.RootManifestJSON, 0o644); err != nil {
		return err
	}
	if err := syncDirectory(transcriptRoot); err != nil {
		return err
	}
	if err := writeExclusive(reportPath, evidence.ReportJSON, 0o644); err != nil {
		return err
	}
	return syncDirectory(base)
}

func claimDevelopmentEvidenceRoot(repoRoot string) error {
	base := filepath.Join(repoRoot, ".nous")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(transcriptPath(repoRoot, "development"), 0o755); err != nil {
		return err
	}
	return syncDirectory(base)
}

func materializeClaimedFixtureBundle(repoRoot, panel string, fixtureJSON []byte) ([]byte, error) {
	if _, err := decodeFixtureBundle(panel, fixtureJSON); err != nil {
		return nil, err
	}
	root := transcriptPath(repoRoot, panel)
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		return nil, fmt.Errorf("claimed transcript root is not empty before fixture materialization: %w", err)
	}
	path := filepath.Join(root, "fixtures.json")
	if err := writeExclusive(path, fixtureJSON, 0o644); err != nil {
		return nil, err
	}
	if err := syncDirectory(root); err != nil {
		return nil, err
	}
	materialized, err := os.ReadFile(path)
	if err != nil || !bytesEqual(materialized, fixtureJSON) {
		return nil, fmt.Errorf("materialized fixture bundle changed: %w", err)
	}
	return materialized, nil
}

func requireAbsent(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("evidence path already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeExclusive(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
