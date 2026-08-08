package causalexpv2

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/chazu/nous/internal/causalv2"
)

func emptyControlRecords(bundle ControlBundle) ControlBundle {
	bundle.Certificates = []ControlCertificate{}
	return bundle
}

func emptyControlEvidenceRecords(evidence ControlEvidence) ControlEvidence {
	evidence.StaticMatrix = []causalv2.MatrixControlRow{}
	evidence.RecomputedMatrix = []causalv2.MatrixControlRow{}
	evidence.Mutation.OffMutants = []causalv2.MutantRecord{}
	evidence.Mutation.OnMutants = []causalv2.MutantRecord{}
	evidence.Corruption.BaselineArtifacts = []causalv2.Base64URL{}
	evidence.Corruption.Cases = []causalv2.CorruptionCase{}
	evidence.NoCredit.CertificateDigests = []string{}
	evidence.NoCredit.ArtifactBytes = []causalv2.Base64URL{}
	evidence.NoCredit.Aggregates = []causalv2.RuleAggregatePayload{}
	evidence.NoCredit.CentralTranscript = []causalv2.CentralTranscriptEvent{}
	evidence.NoCredit.TaskMeterItems = []causalv2.TaskMeterItem{}
	evidence.Dependency.AuditedRoots = []string{}
	evidence.Dependency.Files = []causalv2.DependencyFile{}
	evidence.Dependency.RunnerMethods = []string{}
	evidence.Dependency.RunnerFields = []causalv2.RunnerField{}
	evidence.Dependency.TeacherMethods = []string{}
	evidence.Dependency.Forbidden = []string{}
	return evidence
}

func trainingNonrecordShell(report TrainingReport) TrainingReport {
	report.Applications = []ApplicationCertificate{}
	report.TaskMeterItems = []TaskMeterItem{}
	report.ControlBundle = emptyControlRecords(report.ControlBundle)
	report.ControlEvidence = emptyControlEvidenceRecords(report.ControlEvidence)
	report.ControlEvidence.ControlEvidenceDigest = causalv2.ZeroDigest
	report.ControlEvidenceDigest = causalv2.ZeroDigest
	report.Mechanical.NonrecordBytes = "00000000"
	report.Mechanical.ReportBytes = "00000000"
	return report
}

// FinalizeTrainingReport reconstructs both fixed-width byte fields. It does
// not assign status, validity, gates, controls, or any other empirical result.
func FinalizeTrainingReport(report *TrainingReport) ([]byte, error) {
	if report == nil || report.ReportVersion != "causal-training-report/v2" || report.Panel != "training" {
		return nil, errors.New("invalid training report identity")
	}
	if err := causalv2.ValidateManifest(report.Manifest); err != nil {
		return nil, err
	}
	taskDigest, err := causalv2.TaskMeterItemsDigest(report.TaskMeterItems)
	if err != nil || taskDigest != report.TaskMeterItemsDigest {
		return nil, errors.New("task meter digest mismatch")
	}
	controlBytes, err := causalv2.CanonicalJSON(report.ControlBundle)
	if err != nil {
		return nil, err
	}
	verifiedControl, err := causalv2.VerifyControlBundle(controlBytes)
	if err != nil || verifiedControl.ControlBundleDigest != report.ControlBundleDigest {
		return nil, errors.New("control bundle mismatch")
	}
	if err := verifyExecutedControlBundle(verifiedControl); err != nil {
		return nil, err
	}
	evidenceBytes, err := causalv2.CanonicalJSON(report.ControlEvidence)
	if err != nil {
		return nil, err
	}
	verifiedEvidence, err := causalv2.VerifyControlEvidence(evidenceBytes)
	if err != nil || verifiedEvidence.ControlEvidenceDigest != report.ControlEvidenceDigest {
		return nil, errors.New("control evidence mismatch")
	}
	if err := verifyRetainedControlEvidence(verifiedControl, verifiedEvidence); err != nil {
		return nil, err
	}
	shellBytes, err := causalv2.CanonicalJSON(trainingNonrecordShell(*report))
	if err != nil {
		return nil, err
	}
	if len(shellBytes) > report.Manifest.NonrecordReportByteCap {
		return nil, errors.New("training nonrecord report shell exceeds byte cap")
	}
	report.Mechanical.NonrecordBytes, err = fixedBytes(len(shellBytes))
	if err != nil {
		return nil, err
	}
	report.Mechanical.ReportBytes = "00000000"
	preimage, err := causalv2.CanonicalJSON(*report)
	if err != nil {
		return nil, err
	}
	report.Mechanical.ReportBytes, err = fixedBytes(len(preimage))
	if err != nil {
		return nil, err
	}
	final, err := causalv2.CanonicalJSON(*report)
	if err != nil {
		return nil, err
	}
	if len(final) != len(preimage) || len(final) > report.Manifest.ReportByteCap {
		return nil, errors.New("training report byte reconstruction failed")
	}
	return final, nil
}

func VerifyTrainingReportBytes(data []byte) (TrainingReport, error) {
	report, err := causalv2.StrictDecode[TrainingReport](data)
	if err != nil {
		return report, err
	}
	wantNonrecord, err := parseFixedBytes(report.Mechanical.NonrecordBytes)
	if err != nil {
		return report, err
	}
	wantReport, err := parseFixedBytes(report.Mechanical.ReportBytes)
	if err != nil || wantReport != len(data) {
		return report, errors.New("training report byte count mismatch")
	}
	copy := report
	canonical, err := FinalizeTrainingReport(&copy)
	if err != nil || !bytes.Equal(data, canonical) || wantNonrecord != len(mustCanonical(trainingNonrecordShell(report))) {
		return report, errors.New("training report reconstruction mismatch")
	}
	return report, nil
}

func evaluationNonrecordShell(report EvaluationReport) EvaluationReport {
	report.TaskMeterItems = []TaskMeterItem{}
	report.ControlBundle = emptyControlRecords(report.ControlBundle)
	report.ControlEvidence = emptyControlEvidenceRecords(report.ControlEvidence)
	report.ControlEvidence.ControlEvidenceDigest = causalv2.ZeroDigest
	report.ControlEvidenceDigest = causalv2.ZeroDigest
	report.Policies = append([]PolicyReport(nil), report.Policies...)
	for i := range report.Policies {
		report.Policies[i].Fixtures = []EvaluationFixture{}
	}
	report.Mechanical.NonrecordBytes = "00000000"
	report.Mechanical.ReportBytes = "00000000"
	report.ReportDigest = causalv2.ZeroDigest
	return report
}

// FinalizeEvaluationReport implements the normative three-pass reconstruction:
// nonrecord shell, complete report with zero digest, then empty self-digest.
func FinalizeEvaluationReport(report *EvaluationReport) ([]byte, error) {
	if report == nil || report.ReportVersion != "causal-diagnosis-report/v2" || (report.Panel != "validation" && report.Panel != "locked" && report.Panel != "development") {
		return nil, errors.New("invalid evaluation report identity")
	}
	if err := causalv2.ValidateManifest(report.Manifest); err != nil {
		return nil, err
	}
	taskDigest, err := causalv2.TaskMeterItemsDigest(report.TaskMeterItems)
	if err != nil || taskDigest != report.TaskMeterItemsDigest {
		return nil, errors.New("task meter digest mismatch")
	}
	controlBytes, err := causalv2.CanonicalJSON(report.ControlBundle)
	if err != nil {
		return nil, err
	}
	verifiedControl, err := causalv2.VerifyControlBundle(controlBytes)
	if err != nil || verifiedControl.ControlBundleDigest != report.ControlBundleDigest {
		return nil, errors.New("control bundle mismatch")
	}
	if err := verifyExecutedControlBundle(verifiedControl); err != nil {
		return nil, err
	}
	evidenceBytes, err := causalv2.CanonicalJSON(report.ControlEvidence)
	if err != nil {
		return nil, err
	}
	verifiedEvidence, err := causalv2.VerifyControlEvidence(evidenceBytes)
	if err != nil || verifiedEvidence.ControlEvidenceDigest != report.ControlEvidenceDigest {
		return nil, errors.New("control evidence mismatch")
	}
	if err := verifyRetainedControlEvidence(verifiedControl, verifiedEvidence); err != nil {
		return nil, err
	}
	shell, err := causalv2.CanonicalJSON(evaluationNonrecordShell(*report))
	if err != nil {
		return nil, err
	}
	if len(shell) > report.Manifest.NonrecordReportByteCap {
		return nil, errors.New("evaluation nonrecord report shell exceeds byte cap")
	}
	report.Mechanical.NonrecordBytes, err = fixedBytes(len(shell))
	if err != nil {
		return nil, err
	}
	report.Mechanical.ReportBytes = "00000000"
	report.ReportDigest = causalv2.ZeroDigest
	preimage, err := causalv2.CanonicalJSON(*report)
	if err != nil {
		return nil, err
	}
	report.Mechanical.ReportBytes, err = fixedBytes(len(preimage))
	if err != nil {
		return nil, err
	}
	report.ReportDigest = ""
	report.ReportDigest, err = causalv2.Digest("causal-diagnosis-report/v2", *report)
	if err != nil {
		return nil, err
	}
	final, err := causalv2.CanonicalJSON(*report)
	if err != nil {
		return nil, err
	}
	if len(final) != len(preimage) || len(final) > report.Manifest.ReportByteCap {
		return nil, errors.New("evaluation report byte reconstruction failed")
	}
	return final, nil
}

func VerifyEvaluationReportBytes(data []byte) (EvaluationReport, error) {
	report, err := causalv2.StrictDecode[EvaluationReport](data)
	if err != nil {
		return report, err
	}
	wantReport, err := parseFixedBytes(report.Mechanical.ReportBytes)
	if err != nil || wantReport != len(data) {
		return report, errors.New("evaluation report byte count mismatch")
	}
	wantNonrecord, err := parseFixedBytes(report.Mechanical.NonrecordBytes)
	if err != nil {
		return report, err
	}
	copy := report
	canonical, err := FinalizeEvaluationReport(&copy)
	if err != nil || !bytes.Equal(data, canonical) {
		return report, errors.New("evaluation report reconstruction mismatch")
	}
	if wantNonrecord != len(mustCanonical(evaluationNonrecordShell(report))) {
		return report, errors.New("evaluation nonrecord byte count mismatch")
	}
	return report, nil
}

func mustCanonical(value any) []byte {
	encoded, err := causalv2.CanonicalJSON(value)
	if err != nil {
		panic(fmt.Sprintf("canonical JSON invariant: %v", err))
	}
	return encoded
}

func FinalizeTrainingBundle(bundle *TrainingBundle) ([]byte, error) {
	if bundle == nil || bundle.BundleVersion != "causal-training-episode-bundle/v2" {
		return nil, errors.New("invalid training bundle identity")
	}
	if err := causalv2.ValidateManifest(bundle.Manifest); err != nil {
		return nil, err
	}
	for i, fixture := range bundle.Fixtures {
		if err := VerifyPrivateFixture(fixture); err != nil {
			return nil, fmt.Errorf("fixture %d: %w", i, err)
		}
	}
	for i, episode := range bundle.Episodes {
		if err := VerifyEpisode(episode); err != nil {
			return nil, fmt.Errorf("episode %d: %w", i, err)
		}
	}
	bundle.BundleDigest = ""
	digest, err := causalv2.Digest("causal-training-episode-bundle/v2", *bundle)
	if err != nil {
		return nil, err
	}
	bundle.BundleDigest = digest
	encoded, err := causalv2.CanonicalJSON(*bundle)
	if err != nil {
		return nil, err
	}
	if len(encoded) > bundle.Manifest.TrainingEpisodeBundleByteCap {
		return nil, errors.New("training bundle exceeds byte cap")
	}
	return encoded, nil
}
