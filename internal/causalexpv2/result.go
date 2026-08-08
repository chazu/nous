package causalexpv2

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const ResultsDirectoryName = "nous-results"

func resultPath(commonDirectory string, panel Panel) string {
	return filepath.Join(commonDirectory, ResultsDirectoryName, "active-causal-diagnosis-v3-"+string(panel)+".json")
}

// PublishEvaluationReport publishes validation or locked output with
// exclusive-create semantics after rebinding it to the same clean candidate C.
// A locked publication also reopens the published validation result and proves
// the self-digest which minted its capability has not changed.
func (cap *attemptCapability) publishEvaluationReport(ctx context.Context, repoRoot string, reportBytes []byte) error {
	if cap == nil {
		return errors.New("missing evaluation attempt capability")
	}
	if cap.replayOnly {
		return errors.New("replay regeneration has no evaluation publication authority")
	}
	cap.mu.Lock()
	if cap.consumed || (cap.record.Panel != PanelValidation && cap.record.Panel != PanelLocked) || cap.record.State != "started" {
		cap.mu.Unlock()
		return errors.New("invalid evaluation publication capability")
	}
	if len(cap.generated) != cap.record.SeedRange.Count {
		cap.mu.Unlock()
		return errors.New("attempt has not generated its complete seed range")
	}
	record, validationDigest := cap.record, cap.validationDigest
	cap.mu.Unlock()

	fail := func(cause error) error {
		if transitionErr := writeAttemptState(cap, "failed"); transitionErr != nil {
			return fmt.Errorf("%w; also failed to preserve attempt failure: %v", cause, transitionErr)
		}
		return cause
	}
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return fail(err)
	}
	if !state.Clean || state.Head != record.ExecutableCommit {
		return fail(errors.New("evaluation publication is not bound to the same clean candidate commit"))
	}
	if state.Root != cap.repositoryRoot || state.CommonDir != cap.commonDirectory {
		return fail(errors.New("evaluation publication capability belongs to another repository"))
	}
	verified, err := contextuallyVerifyEvaluationEvidence(ctx, state.Root, reportBytes)
	if err != nil {
		return fail(err)
	}
	report := verified.Report
	cap.mu.Lock()
	expectedReportDigest := cap.expectedReportDigest
	cap.mu.Unlock()
	if expectedReportDigest == "" || report.ReportDigest != expectedReportDigest {
		return fail(errors.New("evaluation report was not produced by this execution capability"))
	}
	if Panel(report.Panel) != record.Panel || report.ImplementationCommit != record.ExecutableCommit || report.PretrainingCommit != record.PretrainingCommit {
		return fail(errors.New("evaluation report provenance does not match attempt"))
	}
	if record.Panel == PanelLocked {
		validationBytes, readErr := os.ReadFile(resultPath(state.CommonDir, PanelValidation))
		if readErr != nil {
			return fail(fmt.Errorf("reopen validation result: %w", readErr))
		}
		validationVerified, verifyErr := contextuallyVerifyEvaluationEvidence(ctx, state.Root, validationBytes)
		validation := validationVerified.Report
		if verifyErr != nil || !mechanicallyValid(validation) || validation.ReportDigest != validationDigest || validation.ImplementationCommit != record.ExecutableCommit {
			return fail(errors.New("locked publication lost its validation self-digest or candidate binding"))
		}
	}
	directory := filepath.Join(state.CommonDir, ResultsDirectoryName)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fail(err)
	}
	path := resultPath(state.CommonDir, record.Panel)
	if err := writeExclusiveSynced(path, reportBytes); err != nil {
		return fail(fmt.Errorf("exclusive result publication: %w", err))
	}
	if err := syncDirectory(directory); err != nil {
		return fail(err)
	}
	cap.mu.Lock()
	cap.proof.PublishedDigest = report.ReportDigest
	if err := persistAttemptProofLocked(cap); err != nil {
		cap.mu.Unlock()
		return fail(err)
	}
	cap.mu.Unlock()
	return writeAttemptState(cap, "published")
}
