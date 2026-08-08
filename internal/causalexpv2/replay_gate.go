package causalexpv2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/chazu/nous/internal/causalv2"
)

const (
	replayRecordVersion      = "causal-replay-success/v4"
	ReplayRepairPlanCommit   = "0162f3eb651049958827da2a9616b0b4b79b512d"
	failedV3ReplayRecordName = "active-causal-diagnosis-v3-replay.json"
)

type replaySuccessRecord struct {
	ReplayVersion       string `json:"replay_version"`
	PlanCommit          string `json:"plan_commit"`
	PretrainingCommit   string `json:"pretraining_commit"`
	EvidenceCommit      string `json:"evidence_commit"`
	CandidateCommit     string `json:"candidate_commit"`
	CandidateDiffDigest string `json:"candidate_diff_digest"`
	TrainingDigest      string `json:"training_digest"`
	BundleDigest        string `json:"bundle_digest"`
	CreatedUTC          string `json:"created_utc"`
	State               string `json:"state"`
}

func replayRecordPath(commonDirectory string) string {
	return filepath.Join(commonDirectory, "nous-attempts", "active-causal-diagnosis-v4-replay.json")
}

func candidateDiffDigest(ctx context.Context, repositoryRoot, evidenceCommit string) (string, error) {
	if err := verifyProtectedGitEnvironment(); err != nil {
		return "", err
	}
	command := exec.CommandContext(ctx, pinnedGitPath, "-C", repositoryRoot, "diff", "--binary", evidenceCommit, "--")
	command.Env = protectedGitCommandEnvironment()
	diff, err := command.Output()
	if err != nil {
		return "", err
	}
	return causalv2.Digest("causal-replay-candidate-diff/v4", struct {
		EvidenceCommit string `json:"evidence_commit"`
		Diff           []byte `json:"diff"`
	}{evidenceCommit, diff})
}

// ExecuteReplay owns the fixed E3 regeneration while HEAD is exact X4 with the
// one uncommitted constants edit that will become C4.
func ExecuteReplay(ctx context.Context, repoRoot string) (returnErr error) {
	if err := orchestrationAvailable(); err != nil {
		return err
	}
	if err := verifyProtectedGitEnvironment(); err != nil {
		return err
	}
	if FrozenTrainingReportCommit == "" {
		return errors.New("training evidence commit is not frozen")
	}
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return err
	}
	if err := verifyPinnedProtectedRuntime(ctx, state.Root); err != nil {
		return err
	}
	if err := requireReplayRetrySlotsAbsent(state); err != nil {
		return err
	}
	if err := verifyCandidateConstantsState(ctx, state, FrozenTrainingReportCommit); err != nil {
		return err
	}
	builder, err := pinnedReplayBuilder(ctx, state.Root)
	if err != nil {
		return err
	}
	if err := preflightReplayBuild(ctx, state.Root, replayPretrainingCommit, builder); err != nil {
		return err
	}
	state, err = resolveGitState(ctx, state.Root)
	if err != nil {
		return err
	}
	if err := verifyPinnedGitRepositoryState(ctx, state); err != nil {
		return err
	}
	if err := requireReplayRetrySlotsAbsent(state); err != nil {
		return err
	}
	record, err := structuralReplayRecord(ctx, state)
	if err != nil {
		return err
	}
	if err := createReplayRecord(state.CommonDir, record); err != nil {
		return err
	}
	defer func() {
		if returnErr != nil {
			record.State = "failed"
			returnErr = errors.Join(returnErr, persistReplayRecord(state.CommonDir, record))
		}
	}()
	capability, err := mintReplayCapability(ctx, state.Root, FrozenTrainingReportCommit)
	if err != nil {
		return err
	}
	if capability.pretrainingCommit != record.PretrainingCommit || capability.evidenceCommit != record.EvidenceCommit || capability.reportDigest != record.TrainingDigest || capability.bundleDigest != record.BundleDigest {
		return errors.New("minted replay capability differs from started v4 receipt")
	}
	if _, err := capability.Replay(ctx); err != nil {
		return err
	}
	record.State = "succeeded"
	return persistReplayRecord(state.CommonDir, record)
}

func requireReplayRetrySlotsAbsent(state gitState) error {
	paths := []string{
		replayRecordPath(state.CommonDir),
		attemptRecordPath(state.CommonDir, PanelValidation),
		attemptProofRecordPath(state.CommonDir, PanelValidation),
		attemptRecordPath(state.CommonDir, PanelLocked),
		attemptProofRecordPath(state.CommonDir, PanelLocked),
		resultPath(state.CommonDir, PanelValidation),
		resultPath(state.CommonDir, PanelLocked),
	}
	for _, path := range paths {
		if err := requireAbsent(path); err != nil {
			return fmt.Errorf("v4 replay preflight collision: %w", err)
		}
	}
	return nil
}

func structuralReplayRecord(ctx context.Context, state gitState) (replaySuccessRecord, error) {
	if err := verifyCandidateConstantsState(ctx, state, replayEvidenceCommit); err != nil {
		return replaySuccessRecord{}, err
	}
	if err := verifyEvidenceCommitShape(ctx, state.Root, replayPretrainingCommit, replayEvidenceCommit); err != nil {
		return replaySuccessRecord{}, err
	}
	replayEvidenceReads.Add(1)
	reportBytes, err := gitFile(ctx, state.Root, replayEvidenceCommit, filepath.ToSlash(filepath.Join(TrainingEvidenceDirectory, TrainingReportName)))
	if err != nil {
		return replaySuccessRecord{}, err
	}
	bundleBytes, err := gitFile(ctx, state.Root, replayEvidenceCommit, filepath.ToSlash(filepath.Join(TrainingEvidenceDirectory, TrainingEpisodesName)))
	if err != nil {
		return replaySuccessRecord{}, err
	}
	report, err := causalv2.StrictDecode[TrainingReport](reportBytes)
	if err != nil || !bytes.Equal(reportBytes, mustCanonical(report)) {
		return replaySuccessRecord{}, errors.New("R3 training report is not structurally canonical")
	}
	bundle, err := causalv2.StrictDecode[TrainingBundle](bundleBytes)
	if err != nil || !bytes.Equal(bundleBytes, mustCanonical(bundle)) {
		return replaySuccessRecord{}, errors.New("R3 training bundle is not structurally canonical")
	}
	if report.ReportVersion != "causal-training-report/v2" || report.PlanCommit != PlanCommit || report.PretrainingCommit != replayPretrainingCommit || report.TrainingDigest != FrozenTrainingDigest || report.EpisodeBundleDigest != bundle.BundleDigest || bundle.BundleVersion != "causal-training-episode-bundle/v2" || bundle.PlanCommit != PlanCommit || bundle.PretrainingCommit != replayPretrainingCommit {
		return replaySuccessRecord{}, errors.New("structural R3 provenance does not match frozen identity")
	}
	diffDigest, err := candidateDiffDigest(ctx, state.Root, replayEvidenceCommit)
	if err != nil {
		return replaySuccessRecord{}, err
	}
	return replaySuccessRecord{ReplayVersion: replayRecordVersion, PlanCommit: ReplayRepairPlanCommit, PretrainingCommit: replayPretrainingCommit, EvidenceCommit: replayEvidenceCommit, CandidateCommit: state.Head, CandidateDiffDigest: diffDigest, TrainingDigest: report.TrainingDigest, BundleDigest: bundle.BundleDigest, CreatedUTC: time.Now().UTC().Format(time.RFC3339), State: "started"}, nil
}

func createReplayRecord(commonDirectory string, record replaySuccessRecord) error {
	directory := filepath.Dir(replayRecordPath(commonDirectory))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	encoded, err := causalv2.CanonicalJSON(record)
	if err != nil {
		return err
	}
	if err := writeExclusiveSynced(replayRecordPath(commonDirectory), encoded); err != nil {
		return fmt.Errorf("create exclusive replay record: %w", err)
	}
	return syncDirectory(directory)
}

func persistReplayRecord(commonDirectory string, record replaySuccessRecord) error {
	encoded, err := causalv2.CanonicalJSON(record)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(replayRecordPath(commonDirectory), os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	if _, err = file.Write(encoded); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = syncDirectory(filepath.Dir(replayRecordPath(commonDirectory)))
	}
	return err
}

func requireReplaySuccess(ctx context.Context, state gitState, report TrainingReport, bundle TrainingBundle) error {
	return verifyReplaySuccessRecord(ctx, state, report, bundle, FrozenTrainingReportCommit)
}

func verifyReplaySuccessRecord(ctx context.Context, state gitState, report TrainingReport, bundle TrainingBundle, evidenceCommit string) error {
	encoded, err := os.ReadFile(replayRecordPath(state.CommonDir))
	if err != nil {
		return fmt.Errorf("read replay success record: %w", err)
	}
	record, err := causalv2.StrictDecode[replaySuccessRecord](encoded)
	if err != nil || !bytes.Equal(encoded, mustCanonical(record)) {
		return errors.New("replay success record is not canonical")
	}
	diffDigest, err := candidateDiffDigest(ctx, state.Root, evidenceCommit)
	if err != nil {
		return err
	}
	parent, parentErr := gitStringOutput(ctx, state.Root, "rev-parse", state.Head+"^")
	if parentErr != nil || verifyCandidateConstantsState(ctx, state, evidenceCommit) != nil || record.ReplayVersion != replayRecordVersion || record.PlanCommit != ReplayRepairPlanCommit || record.State != "succeeded" || record.PretrainingCommit != report.PretrainingCommit || record.EvidenceCommit != evidenceCommit || record.CandidateCommit != parent || record.CandidateDiffDigest != diffDigest || record.TrainingDigest != report.TrainingDigest || record.BundleDigest != bundle.BundleDigest {
		return errors.New("replay success is not bound to E, R, C, and committed evidence")
	}
	return nil
}
