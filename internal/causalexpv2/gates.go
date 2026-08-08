package causalexpv2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/chazu/nous/internal/causalv2"
	"golang.org/x/sys/unix"
)

const FrozenConstantsPath = "internal/causalexpv2/freeze.go"

type ReplayCapability struct {
	mu                sync.Mutex
	repositoryRoot    string
	pretrainingCommit string
	evidenceCommit    string
	seeds             SeedRange
	reportDigest      string
	bundleDigest      string
	reportBytes       []byte
	bundleBytes       []byte
	replayInputBytes  []byte
	buildExecutable   regenerationExecutable
	workerExecutable  regenerationExecutable
	used              bool
}

type ReplayResult struct {
	ReportEqual bool
	BundleEqual bool
}

type regenerationExecutable struct {
	Path       string
	PrefixArgs []string
}

// MintReplayCapability verifies the committed evidence at R and proves that
// the candidate differs from R only by the allowlisted constants source.
func mintReplayCapability(ctx context.Context, repoRoot, evidenceCommit string) (*ReplayCapability, error) {
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return nil, err
	}
	if err := requireAncestor(ctx, state.Root, evidenceCommit, state.Head); err != nil {
		return nil, err
	}
	if err := verifyCandidateConstantsState(ctx, state, evidenceCommit); err != nil {
		return nil, err
	}
	reportBytes, err := gitFile(ctx, state.Root, evidenceCommit, filepath.ToSlash(filepath.Join(TrainingEvidenceDirectory, TrainingReportName)))
	if err != nil {
		return nil, err
	}
	bundleBytes, err := gitFile(ctx, state.Root, evidenceCommit, filepath.ToSlash(filepath.Join(TrainingEvidenceDirectory, TrainingEpisodesName)))
	if err != nil {
		return nil, err
	}
	verified, err := contextuallyVerifyTrainingEvidence(ctx, state.Root, reportBytes, bundleBytes)
	if err != nil {
		return nil, fmt.Errorf("contextually verify committed training evidence: %w", err)
	}
	report, bundle := verified.Report, verified.Bundle
	if err := verifyEvidenceCommitShape(ctx, state.Root, report.PretrainingCommit, evidenceCommit); err != nil {
		return nil, err
	}
	if err := requireUsableTrainingReport(report); err != nil {
		return nil, err
	}
	if err := verifyEmptyFreezeAt(ctx, state.Root, report.PretrainingCommit); err != nil {
		return nil, err
	}
	if err := verifyEmptyFreezeAt(ctx, state.Root, evidenceCommit); err != nil {
		return nil, err
	}
	if err := verifyFrozenCandidate(state.Root, report, evidenceCommit); err != nil {
		return nil, err
	}
	misePath, err := exec.LookPath("mise")
	if err != nil {
		return nil, errors.New("fixed replay launcher mise is unavailable")
	}
	executable, err := validateRegenerationExecutable(regenerationExecutable{Path: misePath, PrefixArgs: []string{"exec", "--", "go"}})
	if err != nil {
		return nil, err
	}
	input := ReplayInput{ReplayInputVersion: ReplayInputVersion, PlanCommit: PlanCommit, PretrainingCommit: report.PretrainingCommit, EvidenceCommit: evidenceCommit, TrainingDigest: report.TrainingDigest, BundleDigest: bundle.BundleDigest, Fixtures: append([]PrivateFixture(nil), bundle.Fixtures...), CorruptionFixture: verified.CorruptionFixture}
	inputBytes, err := finalizeReplayInput(&input)
	if err != nil {
		return nil, fmt.Errorf("construct canonical replay input: %w", err)
	}
	return &ReplayCapability{repositoryRoot: state.Root, pretrainingCommit: report.PretrainingCommit, evidenceCommit: evidenceCommit, seeds: SeedRange{122001, 12, 1}, reportDigest: report.TrainingDigest, bundleDigest: bundle.BundleDigest, reportBytes: reportBytes, bundleBytes: bundleBytes, replayInputBytes: inputBytes, buildExecutable: executable}, nil
}

func verifyCandidateConstantsState(ctx context.Context, state gitState, evidenceCommit string) error {
	if state.Clean {
		if state.Head == evidenceCommit {
			return errors.New("clean evidence commit has no frozen constants edit")
		}
		parent, err := gitStringOutput(ctx, state.Root, "rev-parse", state.Head+"^")
		if err != nil || parent != evidenceCommit {
			return errors.New("candidate C is not the direct child of evidence commit R")
		}
		changed, err := gitStringOutput(ctx, state.Root, "diff", "--name-only", evidenceCommit, state.Head, "--")
		if err != nil || strings.TrimSpace(changed) != FrozenConstantsPath {
			return errors.New("candidate commit diff from R is not the one-file constants edit")
		}
		return nil
	}
	if state.Head != evidenceCommit {
		return errors.New("dirty replay must run directly on evidence commit R")
	}
	status, err := gitStringOutput(ctx, state.Root, "status", "--porcelain", "--untracked-files=all")
	// gitOutput trims the porcelain record's single leading index-column
	// space; an unstaged-only edit therefore has exactly this representation.
	if err != nil || status != "M "+FrozenConstantsPath {
		return errors.New("dirty replay worktree is not the exact unstaged constants edit on R")
	}
	changed, err := gitStringOutput(ctx, state.Root, "diff", "--name-only", evidenceCommit, "--")
	if err != nil || strings.TrimSpace(changed) != FrozenConstantsPath {
		return errors.New("dirty replay diff from R is not the one-file constants edit")
	}
	return nil
}

func verifyEvidenceCommitShape(ctx context.Context, repoRoot, pretrainingCommit, evidenceCommit string) error {
	parent, err := gitStringOutput(ctx, repoRoot, "rev-parse", evidenceCommit+"^")
	if err != nil || parent != pretrainingCommit {
		return errors.New("evidence commit R is not the direct child of pretraining commit E")
	}
	paths := []string{TrainingEvidenceDirectory + "/" + TrainingEpisodesName, TrainingEvidenceDirectory + "/" + TrainingReportName}
	for _, path := range paths {
		if _, err := gitStringOutput(ctx, repoRoot, "cat-file", "-e", pretrainingCommit+":"+path); err == nil {
			return errors.New("canonical evidence path already existed at pretraining commit E")
		}
	}
	changed, err := gitStringOutput(ctx, repoRoot, "diff-tree", "--no-commit-id", "--name-status", "-r", evidenceCommit)
	if err != nil {
		return err
	}
	want := []string{"A\t" + paths[0], "A\t" + paths[1]}
	lines := strings.Split(strings.TrimSpace(changed), "\n")
	slices.Sort(lines)
	if !slices.Equal(lines, want) {
		return errors.New("evidence commit R does not add exactly the two previously absent canonical evidence files")
	}
	return nil
}

func validateRegenerationExecutable(executable regenerationExecutable) (regenerationExecutable, error) {
	if executable.Path == "" || !filepath.IsAbs(executable.Path) {
		return executable, errors.New("regeneration executable must be an absolute path")
	}
	resolved, err := filepath.EvalSymlinks(executable.Path)
	if err != nil {
		return executable, err
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return executable, errors.New("regeneration executable is not an executable regular file")
	}
	executable.Path = resolved
	executable.PrefixArgs = append([]string(nil), executable.PrefixArgs...)
	return executable, nil
}

// Replay owns the fresh detached worktree, fixed regeneration invocation,
// byte comparison, and cleanup. It cannot publish or mutate attempt state.
func (cap *ReplayCapability) Replay(ctx context.Context) (result ReplayResult, returnErr error) {
	if cap == nil {
		return ReplayResult{}, errors.New("replay capability missing or consumed")
	}
	cap.mu.Lock()
	defer cap.mu.Unlock()
	if cap.used {
		return ReplayResult{}, errors.New("replay capability missing or consumed")
	}
	cap.used = true
	base, err := os.MkdirTemp("", "nous-causal-v2-replay-")
	if err != nil {
		return ReplayResult{}, err
	}
	worktree, output := filepath.Join(base, "worktree"), filepath.Join(base, "output")
	if err := runGit(ctx, cap.repositoryRoot, "worktree", "add", "--detach", worktree, cap.pretrainingCommit); err != nil {
		_ = os.RemoveAll(base)
		return ReplayResult{}, err
	}
	defer func() {
		if cleanupErr := cleanupReplayWorktree(cap.repositoryRoot, worktree, base); cleanupErr != nil {
			returnErr = errors.Join(returnErr, cleanupErr)
		}
	}()
	detached, err := resolveGitState(ctx, worktree)
	if err != nil || detached.Head != cap.pretrainingCommit || !detached.Clean {
		return ReplayResult{}, errors.New("replay worktree is not clean at recorded pretraining commit")
	}
	if err := os.Mkdir(output, 0o700); err != nil {
		return ReplayResult{}, err
	}
	input, err := verifyReplayInput(cap.replayInputBytes)
	if err != nil || input.PretrainingCommit != cap.pretrainingCommit || input.EvidenceCommit != cap.evidenceCommit || input.TrainingDigest != cap.reportDigest || input.BundleDigest != cap.bundleDigest {
		return ReplayResult{}, errors.New("replay capability input is invalid or changed")
	}
	worker := cap.workerExecutable
	if worker.Path == "" {
		worker, err = buildReplayWorker(ctx, cap.buildExecutable, worktree, filepath.Join(base, "causal-v2-replay-worker"))
		if err != nil {
			return ReplayResult{}, err
		}
	}
	inputRead, inputWrite, err := os.Pipe()
	if err != nil {
		return ReplayResult{}, err
	}
	defer inputRead.Close()
	defer inputWrite.Close()
	outputHandle, err := os.Open(output)
	if err != nil {
		return ReplayResult{}, err
	}
	defer outputHandle.Close()
	args := append([]string(nil), worker.PrefixArgs...)
	command := exec.CommandContext(ctx, worker.Path, args...)
	command.Dir = worktree
	command.Env = append([]string(nil), os.Environ()...)
	command.ExtraFiles = []*os.File{inputRead, outputHandle}
	var workerOutput bytes.Buffer
	command.Stdout, command.Stderr = &workerOutput, &workerOutput
	if err := command.Start(); err != nil {
		return ReplayResult{}, fmt.Errorf("start replay worker: %w", err)
	}
	_ = inputRead.Close()
	writeErr := writeAllAndClose(inputWrite, cap.replayInputBytes)
	waitErr := command.Wait()
	if writeErr != nil || waitErr != nil {
		return ReplayResult{}, fmt.Errorf("regeneration executable failed: %w: %s", errors.Join(writeErr, waitErr), strings.TrimSpace(workerOutput.String()))
	}
	detached, err = resolveGitState(ctx, worktree)
	if err != nil || detached.Head != cap.pretrainingCommit || !detached.Clean {
		return ReplayResult{}, errors.New("regeneration executable changed the detached pretraining worktree")
	}
	if err := requireExactReplayOutputs(outputHandle); err != nil {
		return ReplayResult{}, err
	}
	generatedReport, err := readReplayOutputAt(outputHandle, TrainingReportName, int64(causalv2.PreregisteredManifest().ReportByteCap))
	if err != nil {
		return ReplayResult{}, fmt.Errorf("read regenerated report: %w", err)
	}
	generatedBundle, err := readReplayOutputAt(outputHandle, TrainingEpisodesName, int64(causalv2.PreregisteredManifest().TrainingEpisodeBundleByteCap))
	if err != nil {
		return ReplayResult{}, fmt.Errorf("read regenerated bundle: %w", err)
	}
	result = ReplayResult{bytes.Equal(cap.reportBytes, generatedReport), bytes.Equal(cap.bundleBytes, generatedBundle)}
	if !result.ReportEqual || !result.BundleEqual {
		return result, errors.New("regenerated evidence differs from committed evidence")
	}
	return result, nil
}

func buildReplayWorker(ctx context.Context, builder regenerationExecutable, worktree, output string) (regenerationExecutable, error) {
	if builder.Path == "" {
		return regenerationExecutable{}, errors.New("replay worker build executable is missing")
	}
	args := append([]string(nil), builder.PrefixArgs...)
	args = append(args, "build", "-o", output, "./internal/causalexpv2/replayexec")
	command := exec.CommandContext(ctx, builder.Path, args...)
	command.Dir = worktree
	command.Env = append([]string(nil), os.Environ()...)
	if outputBytes, err := command.CombinedOutput(); err != nil {
		return regenerationExecutable{}, fmt.Errorf("build fixed replay worker: %w: %s", err, strings.TrimSpace(string(outputBytes)))
	}
	return validateRegenerationExecutable(regenerationExecutable{Path: output})
}

func writeAllAndClose(file *os.File, data []byte) error {
	for len(data) != 0 {
		written, err := file.Write(data)
		if err != nil {
			_ = file.Close()
			return err
		}
		data = data[written:]
	}
	return file.Close()
}

func requireExactReplayOutputs(directory *os.File) error {
	if _, err := directory.Seek(0, 0); err != nil {
		return err
	}
	names, err := directory.Readdirnames(-1)
	if err != nil {
		return err
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{TrainingEpisodesName, TrainingReportName}) {
		return errors.New("replay worker did not produce exactly the two allowlisted outputs")
	}
	return nil
}

func readReplayOutputAt(directory *os.File, name string, maximum int64) ([]byte, error) {
	fd, err := unix.Openat(int(directory.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	if file == nil {
		_ = unix.Close(fd)
		return nil, errors.New("open replay output descriptor")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximum {
		return nil, errors.New("replay output is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(data)) > maximum {
		return nil, errors.New("replay output exceeds byte cap")
	}
	return data, nil
}

func cleanupReplayWorktree(repositoryRoot, worktree, base string) error {
	removeErr := runGit(context.Background(), repositoryRoot, "worktree", "remove", "--force", worktree)
	directoryErr := os.RemoveAll(base)
	if removeErr != nil {
		pruneErr := runGit(context.Background(), repositoryRoot, "worktree", "prune", "--expire", "now")
		return errors.Join(removeErr, directoryErr, pruneErr)
	}
	return directoryErr
}

type FrozenTrainingIdentity struct {
	EvidenceCommit string
	TrainingDigest string
	FrozenRule     string
}

func beginValidationAttempt(ctx context.Context, repoRoot string) (*attemptCapability, error) {
	manifest := causalv2.PreregisteredManifest()
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return nil, err
	}
	if !state.Clean {
		return nil, errors.New("validation requires clean candidate commit")
	}
	if FrozenTrainingReportCommit == "" || FrozenTrainingDigest == "" || FrozenRule == "" {
		return nil, errors.New("training constants are not frozen")
	}
	if err := requireAncestor(ctx, state.Root, FrozenTrainingReportCommit, state.Head); err != nil {
		return nil, err
	}
	changed, err := gitStringOutput(ctx, state.Root, "diff", "--name-only", FrozenTrainingReportCommit, state.Head, "--")
	if err != nil || strings.TrimSpace(changed) != FrozenConstantsPath {
		return nil, errors.New("candidate is not the constants-only child of training evidence")
	}
	reportPath := filepath.Join(state.Root, TrainingEvidenceDirectory, TrainingReportName)
	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}
	bundleBytes, err := os.ReadFile(filepath.Join(state.Root, TrainingEvidenceDirectory, TrainingEpisodesName))
	if err != nil {
		return nil, err
	}
	verified, err := contextuallyVerifyTrainingEvidence(ctx, state.Root, reportBytes, bundleBytes)
	if err != nil || verified.Report.TrainingDigest != FrozenTrainingDigest || verified.Report.SelectedRule != FrozenRule {
		return nil, errors.New("frozen training digest does not match contextually verified committed evidence")
	}
	report, bundle := verified.Report, verified.Bundle
	if err := verifyEvidenceCommitShape(ctx, state.Root, report.PretrainingCommit, FrozenTrainingReportCommit); err != nil {
		return nil, err
	}
	if err := requireUsableTrainingReport(report); err != nil {
		return nil, err
	}
	if err := verifyFrozenCandidate(state.Root, report, FrozenTrainingReportCommit); err != nil {
		return nil, err
	}
	replayCapability, err := mintReplayCapability(ctx, state.Root, FrozenTrainingReportCommit)
	if err != nil {
		return nil, fmt.Errorf("mint synchronous validation replay: %w", err)
	}
	replayResult, err := replayCapability.Replay(ctx)
	if err != nil || !replayResult.ReportEqual || !replayResult.BundleEqual {
		return nil, fmt.Errorf("synchronous validation replay failed: %w", err)
	}
	// The receipt is diagnostic provenance, never authority: the synchronous
	// detached byte replay above has already established the gate independently.
	if err := requireReplaySuccess(ctx, state, report, bundle); err != nil {
		return nil, fmt.Errorf("prior dirty-R replay receipt: %w", err)
	}
	return beginAttempt(beginAttemptOptions{time.Now(), state, PanelValidation, SeedRange{manifest.ValidationSeeds.Start, manifest.ValidationSeeds.Count, manifest.ValidationSeeds.Step}, report.PretrainingCommit, resultPath(state.CommonDir, PanelValidation)})
}

func requireUsableTrainingReport(report TrainingReport) error {
	if report.Status != "valid" || !report.Mechanical.AllValid || !report.Mechanical.AllCapsValid || !report.Mechanical.CreditRecomputed || !report.Mechanical.SelectionVerified || report.Mechanical.OracleDisagreements != 0 {
		return errors.New("training evidence is not mechanically valid and freeze-eligible")
	}
	for _, certificate := range report.ControlBundle.Certificates {
		if !certificate.Passed {
			return errors.New("training evidence contains a failed control")
		}
	}
	return nil
}

// BeginLockedAttempt mints locked generation authority only from a strictly
// reconstructed, mechanically valid validation report for the same clean C.
func beginLockedAttempt(ctx context.Context, repoRoot string) (*attemptCapability, error) {
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return nil, err
	}
	validationBytes, err := os.ReadFile(resultPath(state.CommonDir, PanelValidation))
	if err != nil {
		return nil, fmt.Errorf("read published validation report: %w", err)
	}
	verifiedValidation, err := contextuallyVerifyEvaluationEvidence(ctx, state.Root, validationBytes)
	if err != nil {
		return nil, err
	}
	validation := verifiedValidation.Report
	attemptBytes, err := os.ReadFile(attemptRecordPath(state.CommonDir, PanelValidation))
	if err != nil {
		return nil, fmt.Errorf("read validation attempt provenance: %w", err)
	}
	attempt, err := causalv2.StrictDecode[AttemptRecord](attemptBytes)
	if err != nil || !bytes.Equal(attemptBytes, mustCanonical(attempt)) {
		return nil, errors.New("validation attempt provenance is not canonical")
	}
	proofBytes, err := os.ReadFile(attemptProofRecordPath(state.CommonDir, PanelValidation))
	if err != nil {
		return nil, fmt.Errorf("read validation attempt proof: %w", err)
	}
	proof, err := causalv2.StrictDecode[AttemptProofRecord](proofBytes)
	if err != nil || !bytes.Equal(proofBytes, mustCanonical(proof)) || proof.ProofVersion != "causal-attempt-proof/v2" || proof.Panel != PanelValidation {
		return nil, errors.New("validation attempt proof is not canonical")
	}
	manifest := causalv2.PreregisteredManifest()
	if attempt.State != "published" || attempt.Panel != PanelValidation || attempt.ExecutableCommit != state.Head || proof.PublishedDigest != validation.ReportDigest || len(proof.GeneratedFixtures) != manifest.ValidationSeeds.Count {
		return nil, errors.New("validation result is not bound to its published attempt")
	}
	for index := 0; index < manifest.ValidationSeeds.Count; index++ {
		seed := manifest.ValidationSeeds.Start + int64(index)*manifest.ValidationSeeds.Step
		fixture, generateErr := generate(PanelValidation, seed, index)
		if generateErr != nil || proof.GeneratedFixtures[fmt.Sprintf("%d", seed)] != fixture.PublicFixture.FixtureDigest {
			return nil, errors.New("validation attempt fixture provenance does not reconstruct")
		}
	}
	if !state.Clean || validation.Panel != "validation" || validation.ImplementationCommit != state.Head || !mechanicallyValid(validation) {
		return nil, errors.New("validation report cannot authorize locked panel")
	}
	capability, err := beginAttempt(beginAttemptOptions{time.Now(), state, PanelLocked, SeedRange{manifest.LockedSeeds.Start, manifest.LockedSeeds.Count, manifest.LockedSeeds.Step}, validation.PretrainingCommit, resultPath(state.CommonDir, PanelLocked)})
	if err == nil {
		capability.validationDigest = validation.ReportDigest
	}
	return capability, err
}

func mechanicallyValid(report EvaluationReport) bool {
	m := report.Mechanical
	if !m.AllValid || !m.DependencyBoundary || !m.ProfileValid || !m.TranscriptValid || !m.TrainingFreezeValid || m.OracleDisagreements != 0 || !m.AllCapsValid {
		return false
	}
	for _, meter := range m.Meters {
		if !meter.Valid {
			return false
		}
	}
	for _, certificate := range report.ControlBundle.Certificates {
		if !certificate.Passed {
			return false
		}
	}
	return true
}

func gitStringOutput(ctx context.Context, root string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func gitFile(ctx context.Context, root, commit, path string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", "-C", root, "show", commit+":"+path)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("read %s at %s: %w", path, commit, err)
	}
	return output, nil
}

func runGit(ctx context.Context, root string, args ...string) error {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
