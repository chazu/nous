package transformexp

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
)

type PolicyReportRow struct {
	Ordinal            int    `json:"ordinal"`
	Family             int    `json:"family"`
	Policy             Policy `json:"policy"`
	Terminal           string `json:"terminal"`
	Work               int64  `json:"work"`
	Applications       int    `json:"applications"`
	SchemaSHA256       string `json:"schema_sha256"`
	HeldoutCorrect     int    `json:"heldout_correct"`
	HeldoutCorrectBits string `json:"heldout_correct_bits"`
	FalseApplications  int    `json:"false_applications"`
	NonmatchingWork    int64  `json:"nonmatching_work"`
	TranscriptSHA256   string `json:"transcript_sha256,omitempty"`
}

type SafePanelReport struct {
	Version               string             `json:"version"`
	Panel                 string             `json:"panel"`
	PlanCommit            string             `json:"plan_commit"`
	Manifest              json.RawMessage    `json:"manifest"`
	FixtureRootDigest     string             `json:"fixture_root_digest,omitempty"`
	PrimaryManifestDigest string             `json:"primary_manifest_digest,omitempty"`
	AuditManifestDigest   string             `json:"audit_manifest_digest,omitempty"`
	EvidenceGraphDigest   string             `json:"evidence_graph_digest,omitempty"`
	Rows                  []PolicyReportRow  `json:"rows"`
	Inference             transformInference `json:"inference"`
	Competence            CompetenceReport   `json:"competence"`
	DualExecutionEqual    bool               `json:"dual_execution_equal"`
	TranscriptHashesEqual bool               `json:"transcript_hashes_equal"`
	MechanicallyValid     bool               `json:"mechanically_valid"`
	Limitations           []string           `json:"limitations"`
}

type panelArtifacts struct {
	Primary map[string]TransformTranscriptBundle
	Audit   map[string]TransformTranscriptBundle
}

func (r SafePanelReport) JSON() ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(output.Bytes(), []byte("\n")), nil
}

func runSafePanel(domainsDir, panel string, curricula []curriculum, authority uint64) (SafePanelReport, error) {
	if panel != "safe" || len(curricula) == 0 {
		return SafePanelReport{}, fmt.Errorf("safe runner cannot execute panel %q", panel)
	}
	report, _, err := runPanelDetailed(domainsDir, panel, curricula, authority)
	return report, err
}

func runPanelDetailed(domainsDir, panel string, curricula []curriculum, authority uint64) (SafePanelReport, panelArtifacts, error) {
	return runPanelDetailedWithPairs(domainsDir, panel, curricula, authority, nil)
}

func runPanelDetailedWithPairs(domainsDir, panel string, curricula []curriculum, authority uint64, lockedPairs [][2]uint64) (SafePanelReport, panelArtifacts, error) {
	if len(curricula) == 0 {
		return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("empty panel")
	}
	report := SafePanelReport{Version: "transform-schema-trials/safe-v1", Panel: panel, PlanCommit: PlanCommit, Manifest: json.RawMessage(PreregisteredManifestJSON), DualExecutionEqual: true, TranscriptHashesEqual: true}
	artifacts := panelArtifacts{Primary: map[string]TransformTranscriptBundle{}, Audit: map[string]TransformTranscriptBundle{}}
	paired := make([]pairedTransformRow, len(curricula))
	for index, c := range curricula {
		if c.Ordinal != index {
			return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("noncanonical curriculum ordinal")
		}
		outcomes := map[Policy]PolicyOutcome{}
		for _, policy := range empiricalPolicies {
			outcome, err := executePolicy(domainsDir, c, policy)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("curriculum %d policy %s: %w", c.Ordinal, policy, err)
			}
			audit, err := executePolicy(domainsDir, c, policy)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("curriculum %d policy %s audit: %w", c.Ordinal, policy, err)
			}
			if outcome.Terminal != audit.Terminal || outcome.Applications != audit.Applications || outcome.HeldoutCorrect != audit.HeldoutCorrect || outcome.HeldoutCorrectBits != audit.HeldoutCorrectBits || outcome.FalseApplications != audit.FalseApplications || outcome.NonmatchingWork != audit.NonmatchingWork || !bytes.Equal(outcome.Schema, audit.Schema) {
				report.DualExecutionEqual = false
			}
			if !bytes.Equal(outcome.Transcript.Raw, audit.Transcript.Raw) || !bytes.Equal(outcome.Transcript.Gzip, audit.Transcript.Gzip) || !equalTransformObjects(outcome.Transcript.Objects, audit.Transcript.Objects) {
				report.TranscriptHashesEqual = false
			}
			key := fmt.Sprintf("%s/%03d", policy, c.Ordinal)
			artifacts.Primary[key] = outcome.Transcript
			artifacts.Audit[key] = audit.Transcript
			outcomes[policy] = outcome
			work := int64(outcome.TrainingWork)
			transcriptDigest := ""
			if len(outcome.Transcript.Raw) != 0 {
				work = outcome.Transcript.Work
				transcriptDigest = digestBytes(outcome.Transcript.Raw)
			}
			if work == 0 {
				work = int64(outcome.Applications*80 + 1)
			}
			schemaDigest := ""
			if len(outcome.Schema) != 0 {
				schemaDigest = digestBytes(outcome.Schema)
			}
			nonmatchingWork := outcome.NonmatchingWork
			if outcome.Terminal != "completed" && outcome.HeldoutCorrectBits == 0 {
				nonmatchingWork = 12000
			}
			bits := outcome.HeldoutCorrectBits
			report.Rows = append(report.Rows, PolicyReportRow{c.Ordinal, c.Family, policy, outcome.Terminal, work, outcome.Applications, schemaDigest, outcome.HeldoutCorrect, hex.EncodeToString([]byte{bits}), outcome.FalseApplications, nonmatchingWork, transcriptDigest})
		}
		nous, pbe := outcomes[NousRefine], outcomes[BoundedPBE]
		paired[index] = pairedTransformRow{index, c.Family, nous.HeldoutCorrect == 8, pbe.HeldoutCorrect == 8, nous.FalseApplications, nous.NonmatchingWork, pbe.NonmatchingWork}
	}
	var err error
	report.Inference, err = computeTransformInferenceWithPairs(paired, panel, authority, lockedPairs, 10000, 10000)
	if err != nil {
		return SafePanelReport{}, panelArtifacts{}, err
	}
	report.Competence, err = runTransformCompetence()
	if err != nil {
		return SafePanelReport{}, panelArtifacts{}, err
	}
	report.Limitations = []string{"evidence graph and protected repository authority are not yet implemented", "safe runner is not protected-panel evidence"}
	slices.Sort(report.Limitations)
	report.MechanicallyValid = false
	return report, artifacts, nil
}

func equalTransformObjects(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for digest, value := range left {
		if !bytes.Equal(value, right[digest]) {
			return false
		}
	}
	return true
}
