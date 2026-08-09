package nogoodexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/nogoodfixture"
)

type MechanicalGates struct {
	ManifestValid          bool `json:"manifest_valid"`
	CompetencePassed       bool `json:"competence_passed"`
	DualExecutionEqual     bool `json:"dual_execution_equal"`
	TranscriptHashesEqual  bool `json:"transcript_hashes_equal"`
	TranscriptConservation bool `json:"transcript_conservation"`
	OracleParity           bool `json:"oracle_parity"`
	PrunesSound            bool `json:"prunes_sound"`
}

func developmentTasks() ([]nogoodfixture.Task, error) {
	return nogoodfixture.Panel("development")
}

type SemanticPanel struct {
	Panel             string            `json:"panel"`
	AcquisitionWork   int64             `json:"acquisition_work"`
	AcquisitionVector [12]int64         `json:"acquisition_vector"`
	Policies          []PolicyExecution `json:"policies"`
}

type ReportPayload struct {
	ReportVersion        string              `json:"report_version"`
	Manifest             json.RawMessage     `json:"manifest"`
	PlanCommit           string              `json:"plan_commit"`
	ImplementationCommit string              `json:"implementation_commit"`
	Panel                string              `json:"panel"`
	Competence           CompetenceExecution `json:"competence"`
	Execution            SemanticPanel       `json:"execution"`
	Inference            Inference           `json:"inference"`
	Gates                MechanicalGates     `json:"gates"`
	Limitations          []string            `json:"limitations"`
}

type Report struct {
	Payload            ReportPayload `json:"payload"`
	PayloadSHA256      string        `json:"payload_sha256"`
	RootManifestSHA256 string        `json:"root_manifest_sha256"`
	Classification     string        `json:"classification"`
}

type DevelopmentEvidence struct {
	Report     Report
	Primary    PanelExecution
	Audit      PanelExecution
	ReportJSON []byte
	Bundle     EvidenceBundle
}

func BuildDevelopmentEvidence(domainsDir, implementationCommit string) (DevelopmentEvidence, error) {
	if err := validatePreregisteredManifest(); err != nil {
		return DevelopmentEvidence{}, err
	}
	tasks, err := developmentTasks()
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	primary, err := runPanelExecution(domainsDir, "primary", "development", tasks)
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	audit, err := runPanelExecution(domainsDir, "audit", "development", tasks)
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	primaryCompetence, err := RunCompetence(domainsDir, "development")
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	auditCompetence, err := RunCompetence(domainsDir, "development")
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	primarySemantic := semanticPanel(primary)
	auditSemantic := semanticPanel(audit)
	primaryBytes, err := canonicalJSON(primarySemantic)
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	auditBytes, err := canonicalJSON(auditSemantic)
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	competenceBytes, err := canonicalJSON(primaryCompetence)
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	auditCompetenceBytes, err := canonicalJSON(auditCompetence)
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	if !bytes.Equal(primaryBytes, auditBytes) || !bytes.Equal(competenceBytes, auditCompetenceBytes) {
		return DevelopmentEvidence{}, fmt.Errorf("development dual execution semantic mismatch")
	}
	if err := compareTranscriptBundles(primary, audit); err != nil {
		return DevelopmentEvidence{}, err
	}
	inference, err := InferDevelopment(primary)
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	gates := MechanicalGates{
		ManifestValid: true, CompetencePassed: true, DualExecutionEqual: true,
		TranscriptHashesEqual: true, TranscriptConservation: true,
		OracleParity: true, PrunesSound: true,
	}
	payload := ReportPayload{
		ReportVersion: ReportVersion, Manifest: preregisteredManifest(), PlanCommit: PlanCommit,
		ImplementationCommit: implementationCommit, Panel: "development", Competence: primaryCompetence,
		Execution: primarySemantic, Inference: inference, Gates: gates,
		Limitations: []string{"bounded blocked-pair/v1 grammar", "development evidence is not validation or locked evidence"},
	}
	payloadBytes, err := canonicalJSON(payload)
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	bundle, err := buildEvidenceBundle("development", primary, audit, digestHex(payloadBytes))
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	report := Report{Payload: payload, PayloadSHA256: digestHex(payloadBytes), RootManifestSHA256: digestHex(bundle.RootManifestJSON), Classification: inference.Classification}
	reportJSON, err := canonicalJSON(report)
	if err != nil {
		return DevelopmentEvidence{}, err
	}
	if len(reportJSON) > ReportByteCap {
		return DevelopmentEvidence{}, fmt.Errorf("development report exceeds byte cap")
	}
	return DevelopmentEvidence{Report: report, Primary: primary, Audit: audit, ReportJSON: reportJSON, Bundle: bundle}, nil
}

func semanticPanel(execution PanelExecution) SemanticPanel {
	policies := make([]PolicyExecution, len(execution.Policies))
	for index, policy := range execution.Policies {
		policies[index] = PolicyExecution{Policy: policy.Policy, Tasks: slices.Clone(policy.Tasks)}
	}
	return SemanticPanel{Panel: execution.Panel, AcquisitionWork: execution.AcquisitionWork, AcquisitionVector: execution.AcquisitionVector, Policies: policies}
}

func compareTranscriptBundles(primary, audit PanelExecution) error {
	if len(primary.Policies) != PolicyCount || len(audit.Policies) != PolicyCount {
		return fmt.Errorf("dual execution does not contain %d policies", PolicyCount)
	}
	for index := range primary.Policies {
		left, right := primary.Policies[index], audit.Policies[index]
		if left.Policy != right.Policy || !bytes.Equal(left.Transcript.Raw, right.Transcript.Raw) || !bytes.Equal(left.Transcript.Gzip, right.Transcript.Gzip) {
			offset := firstDifferentByte(left.Transcript.Raw, right.Transcript.Raw)
			return fmt.Errorf("dual transcript mismatch at policy %d (%s), %s", index, left.Policy, describeTranscriptDifference(left.Transcript.Raw, right.Transcript.Raw, offset))
		}
		decoded, err := DecodeTranscript(left.Transcript.Raw)
		if err != nil || decoded.Vector != left.Transcript.Vector {
			return fmt.Errorf("primary transcript conservation failed for %s: %v", left.Policy, err)
		}
	}
	return nil
}

func describeTranscriptDifference(left, right []byte, offset int) string {
	if len(left) < transcriptHeaderSize || len(right) < transcriptHeaderSize {
		return fmt.Sprintf("first raw offset %d", offset)
	}
	leftStart := transcriptHeaderSize + int(binary.BigEndian.Uint64(left[8:16]))
	rightStart := transcriptHeaderSize + int(binary.BigEndian.Uint64(right[8:16]))
	if offset < leftStart || offset < rightStart {
		return fmt.Sprintf("first raw offset %d before records (%d/%d)", offset, leftStart, rightStart)
	}
	leftIndex, rightIndex := (offset-leftStart)/transcriptRecordSize, (offset-rightStart)/transcriptRecordSize
	leftOffset, rightOffset := leftStart+leftIndex*transcriptRecordSize, rightStart+rightIndex*transcriptRecordSize
	if leftOffset+48 > len(left) || rightOffset+48 > len(right) {
		return fmt.Sprintf("first raw offset %d outside complete records", offset)
	}
	return fmt.Sprintf("first raw offset %d records %d/%d left=%s right=%s", offset, leftIndex, rightIndex, hex.EncodeToString(left[leftOffset:leftOffset+48]), hex.EncodeToString(right[rightOffset:rightOffset+48]))
}

func firstDifferentByte(left, right []byte) int {
	for index := 0; index < min(len(left), len(right)); index++ {
		if left[index] != right[index] {
			return index
		}
	}
	return min(len(left), len(right))
}

func canonicalJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

func digestHex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
