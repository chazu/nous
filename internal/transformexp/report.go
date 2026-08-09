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
	Rows                  []PolicyReportRow  `json:"rows"`
	Inference             transformInference `json:"inference"`
	Competence            CompetenceReport   `json:"competence"`
	DualExecutionEqual    bool               `json:"dual_execution_equal"`
	TranscriptHashesEqual bool               `json:"transcript_hashes_equal"`
	MechanicallyValid     bool               `json:"mechanically_valid"`
	Limitations           []string           `json:"limitations"`
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
	report := SafePanelReport{Version: "transform-schema-trials/safe-v1", Panel: panel, PlanCommit: PlanCommit, Manifest: json.RawMessage(PreregisteredManifestJSON), DualExecutionEqual: true, TranscriptHashesEqual: true}
	paired := make([]pairedTransformRow, len(curricula))
	for index, c := range curricula {
		if c.Ordinal != index {
			return SafePanelReport{}, fmt.Errorf("noncanonical curriculum ordinal")
		}
		outcomes := map[Policy]PolicyOutcome{}
		for _, policy := range empiricalPolicies {
			outcome, err := executePolicy(domainsDir, c, policy)
			if err != nil {
				return SafePanelReport{}, fmt.Errorf("curriculum %d policy %s: %w", c.Ordinal, policy, err)
			}
			audit, err := executePolicy(domainsDir, c, policy)
			if err != nil {
				return SafePanelReport{}, fmt.Errorf("curriculum %d policy %s audit: %w", c.Ordinal, policy, err)
			}
			if outcome.Terminal != audit.Terminal || outcome.Applications != audit.Applications || outcome.HeldoutCorrect != audit.HeldoutCorrect || outcome.FalseApplications != audit.FalseApplications || outcome.NonmatchingWork != audit.NonmatchingWork || !bytes.Equal(outcome.Schema, audit.Schema) {
				report.DualExecutionEqual = false
			}
			if !bytes.Equal(outcome.Transcript.Raw, audit.Transcript.Raw) || !bytes.Equal(outcome.Transcript.Gzip, audit.Transcript.Gzip) {
				report.TranscriptHashesEqual = false
			}
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
			bits := byte(0)
			if outcome.HeldoutCorrect == 8 {
				bits = 0xff
			}
			report.Rows = append(report.Rows, PolicyReportRow{c.Ordinal, c.Family, policy, outcome.Terminal, work, outcome.Applications, schemaDigest, outcome.HeldoutCorrect, hex.EncodeToString([]byte{bits}), outcome.FalseApplications, outcome.NonmatchingWork, transcriptDigest})
		}
		nous, pbe := outcomes[NousRefine], outcomes[BoundedPBE]
		paired[index] = pairedTransformRow{index, c.Family, nous.HeldoutCorrect == 8, pbe.HeldoutCorrect == 8, nous.FalseApplications, nous.NonmatchingWork, pbe.NonmatchingWork}
		if paired[index].NonmatchingPBEWork == 0 {
			paired[index].NonmatchingPBEWork = int64(4 * 80)
		}
	}
	var err error
	report.Inference, err = computeTransformInference(paired, panel, authority, 10000, 10000)
	if err != nil {
		return SafePanelReport{}, err
	}
	report.Competence, err = runTransformCompetence()
	if err != nil {
		return SafePanelReport{}, err
	}
	report.Limitations = []string{"evidence graph and protected repository authority are not yet implemented", "safe runner is not protected-panel evidence"}
	slices.Sort(report.Limitations)
	report.MechanicallyValid = false
	return report, nil
}
