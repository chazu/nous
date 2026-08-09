package transformexp

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/chazu/nous/internal/transformoracle"
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
	Version               string                `json:"version"`
	Panel                 string                `json:"panel"`
	PlanCommit            string                `json:"plan_commit"`
	Manifest              json.RawMessage       `json:"manifest"`
	FixtureRootDigest     string                `json:"fixture_root_digest,omitempty"`
	PrimaryManifestDigest string                `json:"primary_manifest_digest,omitempty"`
	AuditManifestDigest   string                `json:"audit_manifest_digest,omitempty"`
	EvidenceGraphDigest   string                `json:"evidence_graph_digest,omitempty"`
	Rows                  []PolicyReportRow     `json:"rows"`
	Inference             transformInference    `json:"inference"`
	Competence            CompetenceReport      `json:"competence"`
	DualExecutionEqual    bool                  `json:"dual_execution_equal"`
	TranscriptHashesEqual bool                  `json:"transcript_hashes_equal"`
	MechanicallyValid     bool                  `json:"mechanically_valid"`
	Conservation          bool                  `json:"conservation"`
	OracleParity          bool                  `json:"oracle_parity"`
	ProgramsExact         bool                  `json:"programs_exact"`
	ApplicationsExact     bool                  `json:"applications_exact"`
	ArtifactFrozen        bool                  `json:"artifact_frozen"`
	HeldoutSealed         bool                  `json:"heldout_sealed"`
	GeneratorAcceptance   AcceptanceDiagnostics `json:"generator_acceptance"`
	OracleAcceptance      AcceptanceDiagnostics `json:"oracle_acceptance"`
	Limitations           []string              `json:"limitations"`
}

type AcceptanceDiagnostics struct {
	Curricula    int    `json:"curricula"`
	Applications int    `json:"applications"`
	Work         int64  `json:"work"`
	RootSHA256   string `json:"root_sha256"`
	Exact        bool   `json:"exact"`
}

type panelArtifacts struct {
	Primary         map[string]TransformTranscriptBundle
	Audit           map[string]TransformTranscriptBundle
	PrimaryStores   map[string][]byte
	AuditStores     map[string][]byte
	PrimaryPrograms map[string][]byte
	AuditPrograms   map[string][]byte
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
	curricula = cloneCurriculaForExecution(curricula)
	report := SafePanelReport{Version: "transform-schema-trials/safe-v2", Panel: panel, PlanCommit: PlanCommit, Manifest: json.RawMessage(PreregisteredManifestJSON), DualExecutionEqual: true, TranscriptHashesEqual: true, Conservation: true, OracleParity: true, ProgramsExact: true, ApplicationsExact: true, ArtifactFrozen: true, HeldoutSealed: true, GeneratorAcceptance: AcceptanceDiagnostics{Exact: true}, OracleAcceptance: AcceptanceDiagnostics{Exact: true}}
	artifacts := panelArtifacts{Primary: map[string]TransformTranscriptBundle{}, Audit: map[string]TransformTranscriptBundle{}, PrimaryStores: map[string][]byte{}, AuditStores: map[string][]byte{}, PrimaryPrograms: map[string][]byte{}, AuditPrograms: map[string][]byte{}}
	sealedScorers := make([][]byte, len(curricula))
	for index := range curricula {
		scorer, err := scorerFixtureBytes(curricula[index])
		if err != nil {
			return SafePanelReport{}, panelArtifacts{}, err
		}
		sealedScorers[index] = scorer
		eraseBytes(curricula[index].Scorer)
		eraseBytes(curricula[index].Latent)
		for expectedIndex := range curricula[index].Expected {
			eraseBytes(curricula[index].Expected[expectedIndex].Output)
		}
		curricula[index].Scorer = nil
		curricula[index].Latent = nil
		curricula[index].Expected = nil
		curricula[index].SeedCommitment = ""
		curricula[index].AcceptedAttempt = 0
	}
	defer func() {
		for index := range sealedScorers {
			eraseBytes(sealedScorers[index])
			sealedScorers[index] = nil
		}
	}()
	var generatorRows, oracleRows []any
	for index, c := range curricula {
		if c.Ordinal != index {
			return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("noncanonical curriculum ordinal")
		}
		generatorExact := c.GeneratorLedger.Applications == 72*16 && c.GeneratorLedger.Work == 109161 && digestString(c.GeneratorLedger.MatrixSHA256) && c.GeneratorLedger.Accepted
		report.GeneratorAcceptance.Curricula++
		report.GeneratorAcceptance.Applications += c.GeneratorLedger.Applications
		report.GeneratorAcceptance.Work += c.GeneratorLedger.Work
		report.GeneratorAcceptance.Exact = report.GeneratorAcceptance.Exact && generatorExact
		generatorRows = append(generatorRows, []any{c.Ordinal, c.GeneratorLedger.Applications, c.GeneratorLedger.Work, c.GeneratorLedger.MatrixSHA256, c.GeneratorLedger.Accepted})
		for _, policy := range empiricalPolicies {
			primaryView, err := decodePolicyView(c)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			outcome, err := executePolicy(domainsDir, primaryView, c.Ordinal, policy)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("curriculum %d policy %s: %w", c.Ordinal, policy, err)
			}
			primaryHeldout, err := decodeHeldoutInputs(c)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			outcome, err = executeHeldoutInputs(primaryView, primaryHeldout, outcome)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			primaryScorer, err := decodeSealedScorer(sealedScorers[index])
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			outcome, err = scorePolicyOutcome(primaryScorer, outcome)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			primaryScorerBytes := bytes.Clone(sealedScorers[index])
			outcome, err = auditPolicyOutcome(primaryView, primaryHeldout, primaryScorerBytes, outcome)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			eraseScorerView(&primaryScorer)
			eraseBytes(primaryScorerBytes)
			auditView, err := decodePolicyView(c)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			audit, err := executePolicy(domainsDir, auditView, c.Ordinal, policy)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("curriculum %d policy %s audit: %w", c.Ordinal, policy, err)
			}
			auditHeldout, err := decodeHeldoutInputs(c)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			audit, err = executeHeldoutInputs(auditView, auditHeldout, audit)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			auditScorer, err := decodeSealedScorer(sealedScorers[index])
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			audit, err = scorePolicyOutcome(auditScorer, audit)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			auditScorerBytes := bytes.Clone(sealedScorers[index])
			audit, err = auditPolicyOutcome(auditView, auditHeldout, auditScorerBytes, audit)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, err
			}
			eraseScorerView(&auditScorer)
			eraseBytes(auditScorerBytes)
			if outcome.Terminal != audit.Terminal || outcome.Applications != audit.Applications || outcome.HeldoutCorrect != audit.HeldoutCorrect || outcome.HeldoutCorrectBits != audit.HeldoutCorrectBits || outcome.FalseApplications != audit.FalseApplications || outcome.NonmatchingWork != audit.NonmatchingWork || outcome.OracleParity != audit.OracleParity || outcome.ProgramsExact != audit.ProgramsExact || !bytes.Equal(outcome.Schema, audit.Schema) {
				report.DualExecutionEqual = false
			}
			if !bytes.Equal(outcome.Transcript.Raw, audit.Transcript.Raw) || !bytes.Equal(outcome.Transcript.Gzip, audit.Transcript.Gzip) || !equalTransformObjects(outcome.Transcript.Objects, audit.Transcript.Objects) {
				report.TranscriptHashesEqual = false
			}
			manifestDigest := policyManifestDigest(primaryView, policy)
			primaryReduced, err := reduceTransformTranscriptWithTraining(outcome.Transcript.Raw, outcome.Transcript.Objects, manifestDigest, c.Training)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("primary transcript reduction curriculum %d policy %s: %w", c.Ordinal, policy, err)
			}
			auditReduced, err := reduceTransformTranscriptWithTraining(audit.Transcript.Raw, audit.Transcript.Objects, manifestDigest, c.Training)
			if err != nil {
				return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("audit transcript reduction curriculum %d policy %s: %w", c.Ordinal, policy, err)
			}
			report.Conservation = report.Conservation && primaryReduced.Vector == outcome.Transcript.Vector && primaryReduced.Work == outcome.Transcript.Work && primaryReduced.Terminal == outcome.Terminal && auditReduced.Vector == audit.Transcript.Vector && auditReduced.Work == audit.Transcript.Work && auditReduced.Terminal == audit.Terminal
			report.ApplicationsExact = report.ApplicationsExact && primaryReduced.Applications == outcome.Applications && auditReduced.Applications == audit.Applications
			report.OracleParity = report.OracleParity && outcome.OracleParity && audit.OracleParity
			report.ProgramsExact = report.ProgramsExact && outcome.ProgramsExact && audit.ProgramsExact
			report.ArtifactFrozen = report.ArtifactFrozen && (outcome.Terminal != "completed" || policy == ConcreteReplay && len(outcome.frozenReplayBatch) != 0 || policy != ConcreteReplay && len(outcome.Schema) != 0)
			report.HeldoutSealed = report.HeldoutSealed && (outcome.Terminal != "completed" || policy != NousRefine && policy != NoEqualityGuard || outcome.HeldoutStoreUnchanged)
			key := fmt.Sprintf("%s/%03d", policy, c.Ordinal)
			artifacts.Primary[key] = outcome.Transcript
			artifacts.Audit[key] = audit.Transcript
			artifacts.PrimaryStores[key] = bytes.Clone(outcome.trainingStore)
			artifacts.AuditStores[key] = bytes.Clone(audit.trainingStore)
			artifacts.PrimaryPrograms[key] = bytes.Clone(outcome.frozenPrograms)
			artifacts.AuditPrograms[key] = bytes.Clone(audit.frozenPrograms)
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
		scorerBytes := bytes.Clone(sealedScorers[index])
		oracleLedger, err := transformoracle.AuditAcceptance(c.Training, c.Heldout, scorerBytes)
		eraseBytes(scorerBytes)
		if err != nil {
			return SafePanelReport{}, panelArtifacts{}, fmt.Errorf("curriculum %d acceptance audit: %w", c.Ordinal, err)
		}
		oracleExact := oracleLedger.Applications == c.GeneratorLedger.Applications && oracleLedger.Work == c.GeneratorLedger.Work && oracleLedger.MatrixSHA256 == c.GeneratorLedger.MatrixSHA256 && oracleLedger.Accepted == c.GeneratorLedger.Accepted
		report.OracleAcceptance.Curricula++
		report.OracleAcceptance.Applications += oracleLedger.Applications
		report.OracleAcceptance.Work += oracleLedger.Work
		report.OracleAcceptance.Exact = report.OracleAcceptance.Exact && oracleExact
		oracleRows = append(oracleRows, []any{c.Ordinal, oracleLedger.Applications, oracleLedger.Work, oracleLedger.MatrixSHA256, oracleLedger.Accepted})
	}
	report.GeneratorAcceptance.RootSHA256 = digestBytes(mustJSON([]any{"transform-generator-acceptance-ledgers/v1", generatorRows}))
	report.OracleAcceptance.RootSHA256 = digestBytes(mustJSON([]any{"transform-oracle-acceptance-ledgers/v1", oracleRows}))
	paired, err := pairedRows(report.Rows, len(curricula))
	if err != nil {
		return SafePanelReport{}, panelArtifacts{}, err
	}
	report.Inference, err = computeTransformInferenceWithPairs(paired, panel, authority, lockedPairs, 10000, 10000)
	if err != nil {
		return SafePanelReport{}, panelArtifacts{}, err
	}
	report.Competence, err = runTransformCompetence(domainsDir)
	if err != nil {
		return SafePanelReport{}, panelArtifacts{}, err
	}
	report.Limitations = []string{"safe runner is not protected-panel evidence"}
	slices.Sort(report.Limitations)
	report.MechanicallyValid = report.DualExecutionEqual && report.TranscriptHashesEqual && report.Conservation && report.OracleParity && report.ProgramsExact && report.ApplicationsExact && report.ArtifactFrozen && report.HeldoutSealed && report.GeneratorAcceptance.Exact && report.OracleAcceptance.Exact
	return report, artifacts, nil
}

func cloneCurriculaForExecution(source []curriculum) []curriculum {
	out := make([]curriculum, len(source))
	for i := range source {
		out[i] = source[i]
		out[i].Training = bytes.Clone(source[i].Training)
		out[i].Heldout = bytes.Clone(source[i].Heldout)
		out[i].Latent = bytes.Clone(source[i].Latent)
		out[i].Queue = bytes.Clone(source[i].Queue)
		out[i].Scorer = bytes.Clone(source[i].Scorer)
		out[i].Expected = make([]expectedCase, len(source[i].Expected))
		for j := range source[i].Expected {
			out[i].Expected[j] = source[i].Expected[j]
			out[i].Expected[j].Output = bytes.Clone(source[i].Expected[j].Output)
		}
		out[i].PolicyTokens = maps.Clone(source[i].PolicyTokens)
		out[i].PolicyRandomness = maps.Clone(source[i].PolicyRandomness)
	}
	return out
}

func decodeSealedScorer(sealed []byte) (scorerCurriculum, error) {
	clone := bytes.Clone(sealed)
	defer eraseBytes(clone)
	return decodeScorerBytes(clone)
}

func eraseScorerView(view *scorerCurriculum) {
	eraseBytes(view.Latent)
	for index := range view.Expected {
		eraseBytes(view.Expected[index].Output)
		view.Expected[index] = expectedCase{}
	}
	view.Family = 0
	view.SeedCommitment = ""
	view.AcceptedAttempt = 0
	view.Latent = nil
	view.Expected = nil
}

func eraseBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
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
