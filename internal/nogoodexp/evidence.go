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

func buildEvidenceBundle(panel string, primary, audit PanelExecution, payloadDigest string) (EvidenceBundle, error) {
	primaryManifest, err := executionManifest(primary)
	if err != nil {
		return EvidenceBundle{}, err
	}
	auditManifest, err := executionManifest(audit)
	if err != nil {
		return EvidenceBundle{}, err
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
		ReportPayloadSHA256: payloadDigest, FinalReportReference: fmt.Sprintf("nogoods-v1-%s-report.json", panel),
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
	return manifest, nil
}

func PersistDevelopmentEvidence(repoRoot string, evidence DevelopmentEvidence) error {
	base := filepath.Join(repoRoot, ".nous")
	reportPath := filepath.Join(base, "nogoods-v1-development-report.json")
	transcriptRoot := filepath.Join(base, "nogoods-v1-development-transcripts")
	if err := requireAbsent(reportPath); err != nil {
		return err
	}
	if err := requireAbsent(transcriptRoot); err != nil {
		return err
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(transcriptRoot, 0o755); err != nil {
		return err
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
	}
	if err := writeExclusive(filepath.Join(transcriptRoot, "manifest.json"), evidence.Bundle.RootManifestJSON, 0o644); err != nil {
		return err
	}
	return writeExclusive(reportPath, evidence.ReportJSON, 0o644)
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
	return file.Close()
}
