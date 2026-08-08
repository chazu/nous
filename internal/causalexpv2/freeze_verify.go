package causalexpv2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func expectedFreezeFile(rule, evidenceCommit, trainingDigest string) []byte {
	return []byte(fmt.Sprintf("package causalexpv2\n\nconst (\n\tFrozenRule                 = %q\n\tFrozenTrainingReportCommit = %q\n\tFrozenTrainingDigest       = %q\n)\n", rule, evidenceCommit, trainingDigest))
}

func verifyEmptyFreezeAt(ctx context.Context, repositoryRoot, commit string) error {
	encoded, err := gitFile(ctx, repositoryRoot, commit, FrozenConstantsPath)
	if err != nil {
		return err
	}
	if !bytes.Equal(encoded, expectedFreezeFile("", "", "")) {
		return errors.New("pretraining/evidence freeze.go is not the exact empty three-assignment file")
	}
	return nil
}

func verifyFrozenCandidate(repositoryRoot string, report TrainingReport, evidenceCommit string) error {
	if report.SelectedRule == "" || report.TrainingDigest == "" {
		return errors.New("training report lacks selected rule or digest")
	}
	encoded, err := os.ReadFile(filepath.Join(repositoryRoot, FrozenConstantsPath))
	if err != nil {
		return err
	}
	if !bytes.Equal(encoded, expectedFreezeFile(report.SelectedRule, evidenceCommit, report.TrainingDigest)) {
		return errors.New("candidate freeze.go is not exactly the three allowlisted assignments")
	}
	if FrozenRule != report.SelectedRule || FrozenTrainingReportCommit != evidenceCommit || FrozenTrainingDigest != report.TrainingDigest {
		return errors.New("compiled frozen constants do not equal committed training evidence")
	}
	return nil
}
