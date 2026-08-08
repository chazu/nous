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

const replayRecordVersion = "causal-replay-success/v3"

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
	return filepath.Join(commonDirectory, "nous-attempts", "active-causal-diagnosis-v3-replay.json")
}

func candidateDiffDigest(ctx context.Context, repositoryRoot, evidenceCommit string) (string, error) {
	command := exec.CommandContext(ctx, "git", "-C", repositoryRoot, "diff", "--binary", evidenceCommit, "--")
	diff, err := command.Output()
	if err != nil {
		return "", err
	}
	return causalv2.Digest("causal-replay-candidate-diff/v3", struct {
		EvidenceCommit string `json:"evidence_commit"`
		Diff           []byte `json:"diff"`
	}{evidenceCommit, diff})
}

// ExecuteReplay owns the fixed E regeneration while HEAD is R with the exact
// uncommitted three-constant edit that will become C.
func ExecuteReplay(ctx context.Context, repoRoot string) (returnErr error) {
	if err := orchestrationAvailable(); err != nil {
		return err
	}
	if FrozenTrainingReportCommit == "" {
		return errors.New("training evidence commit is not frozen")
	}
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return err
	}
	capability, err := mintReplayCapability(ctx, state.Root, FrozenTrainingReportCommit)
	if err != nil {
		return err
	}
	diffDigest, err := candidateDiffDigest(ctx, state.Root, capability.evidenceCommit)
	if err != nil {
		return err
	}
	record := replaySuccessRecord{ReplayVersion: replayRecordVersion, PlanCommit: PlanCommit, PretrainingCommit: capability.pretrainingCommit, EvidenceCommit: capability.evidenceCommit, CandidateCommit: state.Head, CandidateDiffDigest: diffDigest, TrainingDigest: capability.reportDigest, BundleDigest: capability.bundleDigest, CreatedUTC: time.Now().UTC().Format(time.RFC3339), State: "started"}
	if err := createReplayRecord(state.CommonDir, record); err != nil {
		return err
	}
	defer func() {
		if returnErr != nil {
			record.State = "failed"
			returnErr = errors.Join(returnErr, persistReplayRecord(state.CommonDir, record))
		}
	}()
	if _, err := capability.Replay(ctx); err != nil {
		return err
	}
	record.State = "succeeded"
	return persistReplayRecord(state.CommonDir, record)
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
	if record.ReplayVersion != replayRecordVersion || record.PlanCommit != PlanCommit || record.State != "succeeded" || record.PretrainingCommit != report.PretrainingCommit || record.EvidenceCommit != evidenceCommit || record.CandidateCommit != evidenceCommit || record.CandidateDiffDigest != diffDigest || record.TrainingDigest != report.TrainingDigest || record.BundleDigest != bundle.BundleDigest {
		return errors.New("replay success is not bound to E, R, C, and committed evidence")
	}
	return nil
}
