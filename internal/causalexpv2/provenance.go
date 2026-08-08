// Package causalexpv2 owns the offline experiment and provenance boundary for
// active causal diagnosis v2. It deliberately does not provide an online
// runner; online execution belongs to internal/causalrun.
package causalexpv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/chazu/nous/internal/causalv2"
)

const (
	PlanCommit              = "b4183595a9769882ad4673c606e9a35560cfb95c"
	BaselineCommit          = "44f19d2823307906a8a0393e13a995d00d33f639"
	AttemptVersion          = "causal-attempt/v3"
	AttemptProofVersion     = "causal-attempt-proof/v3"
	ReplayCapabilityVersion = "causal-replay/v3"

	TrainingEvidenceDirectory = "docs/evidence/active-causal-diagnosis-v3"
	TrainingReportName        = "training.json"
	TrainingEpisodesName      = "training-episodes.json"
)

type Panel string

const (
	PanelDevelopment Panel = "development"
	PanelTraining    Panel = "training"
	PanelValidation  Panel = "validation"
	PanelLocked      Panel = "locked"
)

func (p Panel) protected() bool {
	return p == PanelTraining || p == PanelValidation || p == PanelLocked
}

type SeedRange struct {
	Start int64 `json:"start"`
	Count int   `json:"count"`
	Step  int64 `json:"step"`
}

type AttemptRecord struct {
	AttemptVersion    string    `json:"attempt_version"`
	PlanCommit        string    `json:"plan_commit"`
	PretrainingCommit string    `json:"pretraining_commit"`
	Panel             Panel     `json:"panel"`
	SeedRange         SeedRange `json:"seed_range"`
	ExecutableCommit  string    `json:"executable_commit"`
	CreatedUTC        string    `json:"created_utc"`
	State             string    `json:"state"`
}

type AttemptProofRecord struct {
	ProofVersion      string            `json:"proof_version"`
	Panel             Panel             `json:"panel"`
	GeneratedFixtures map[string]string `json:"generated_fixtures"`
	PublishedDigest   string            `json:"published_digest"`
}

// attemptCapability is intentionally unexported. Protected fixture generation
// can only receive one from the checked constructors in this package.
type attemptCapability struct {
	mu                      sync.Mutex
	record                  AttemptRecord
	recordPath              string
	proof                   AttemptProofRecord
	proofPath               string
	repositoryRoot          string
	commonDirectory         string
	consumed                bool
	generated               map[int64]string
	validationDigest        string
	expectedReportDigest    string
	expectedBundleDigest    string
	replayFixtures          []PrivateFixture
	replayCorruptionFixture *PrivateFixture
	replayOnly              bool
}

type DiagnosticDevelopmentCapability struct{ marker struct{} }

func NewDiagnosticDevelopmentCapability() DiagnosticDevelopmentCapability {
	return DiagnosticDevelopmentCapability{}
}

// ExecuteProtectedPanel is the sole operational boundary for protected v3
// commands. The unchanged semantic executors may produce detailed internal
// errors; none of those values cross this boundary before atomic publication.
func ExecuteProtectedPanel(ctx context.Context, repoRoot string, panel Panel) error {
	if err := verifyPinnedProtectedRuntime(ctx, repoRoot); err != nil {
		return fmt.Errorf("causal %s panel failed before publication", panel)
	}
	return executeProtectedPanelWith(ctx, repoRoot, panel, func(ctx context.Context, repoRoot string, panel Panel) error {
		switch panel {
		case PanelTraining:
			return ExecuteTraining(ctx, repoRoot)
		case PanelValidation:
			return ExecuteValidation(ctx, repoRoot)
		case PanelLocked:
			return ExecuteLocked(ctx, repoRoot)
		default:
			return fmt.Errorf("panel %q is not protected", panel)
		}
	})
}

func executeProtectedPanelWith(ctx context.Context, repoRoot string, panel Panel, run func(context.Context, string, Panel) error) error {
	if !panel.protected() || run == nil {
		return errors.New("invalid protected panel invocation")
	}
	if err := run(ctx, repoRoot, panel); err != nil {
		return fmt.Errorf("causal %s panel failed before publication", panel)
	}
	return nil
}

type gitState struct {
	Root      string
	CommonDir string
	Head      string
	Clean     bool
}

func resolveGitState(ctx context.Context, root string) (gitState, error) {
	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	top, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return gitState{}, err
	}
	common, err := run("rev-parse", "--git-common-dir")
	if err != nil {
		return gitState{}, err
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(top, common)
	}
	head, err := run("rev-parse", "HEAD")
	if err != nil {
		return gitState{}, err
	}
	status, err := run("status", "--porcelain")
	if err != nil {
		return gitState{}, err
	}
	return gitState{Root: top, CommonDir: filepath.Clean(common), Head: head, Clean: status == ""}, nil
}

func requireAncestor(ctx context.Context, root, ancestor, descendant string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "merge-base", "--is-ancestor", ancestor, descendant)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("commit %s is not an ancestor of %s", ancestor, descendant)
	}
	return nil
}

type beginAttemptOptions struct {
	Now               time.Time
	Git               gitState
	Panel             Panel
	Seeds             SeedRange
	PretrainingCommit string
	EvidenceDirectory string
}

func beginAttempt(opts beginAttemptOptions) (*attemptCapability, error) {
	if !opts.Panel.protected() {
		return nil, fmt.Errorf("panel %q does not use an acceptance attempt", opts.Panel)
	}
	if !opts.Git.Clean {
		return nil, errors.New("acceptance panel requires a clean worktree")
	}
	wantSeeds, err := panelSeedRange(opts.Panel)
	if err != nil || opts.Seeds != wantSeeds {
		return nil, errors.New("attempt seed range is not the exact preregistered panel")
	}
	if opts.PretrainingCommit == "" || opts.Git.Head == "" || opts.Git.CommonDir == "" {
		return nil, errors.New("incomplete attempt provenance")
	}
	if opts.Panel == PanelTraining && opts.PretrainingCommit != opts.Git.Head {
		return nil, errors.New("training pretraining commit must equal HEAD")
	}
	if opts.EvidenceDirectory != "" {
		if _, err := os.Lstat(opts.EvidenceDirectory); err == nil {
			return nil, fmt.Errorf("evidence path already exists: %s", opts.EvidenceDirectory)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	dir := filepath.Join(opts.Git.CommonDir, "nous-attempts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := attemptRecordPath(opts.Git.CommonDir, opts.Panel)
	record := AttemptRecord{
		AttemptVersion:    AttemptVersion,
		PlanCommit:        PlanCommit,
		PretrainingCommit: opts.PretrainingCommit,
		Panel:             opts.Panel,
		SeedRange:         opts.Seeds,
		ExecutableCommit:  opts.Git.Head,
		CreatedUTC:        opts.Now.UTC().Format(time.RFC3339),
		State:             "started",
	}
	proof := AttemptProofRecord{ProofVersion: AttemptProofVersion, Panel: opts.Panel, GeneratedFixtures: map[string]string{}}
	data, err := canonicalJSON(record)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create exclusive attempt record: %w", err)
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			// A created record is intentionally preserved and transitioned to
			// failed on every locally recoverable write/fsync failure.
			record.State = "failed"
			if failedBytes, encodeErr := canonicalJSON(record); encodeErr == nil {
				if failedFile, openErr := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0); openErr == nil {
					_, _ = failedFile.Write(failedBytes)
					_ = failedFile.Sync()
					_ = failedFile.Close()
				}
			}
			_ = syncDirectory(dir)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	if err := syncDirectory(dir); err != nil {
		return nil, err
	}
	proofPath := attemptProofRecordPath(opts.Git.CommonDir, opts.Panel)
	proofBytes, err := canonicalJSON(proof)
	if err != nil {
		return nil, err
	}
	if err := writeExclusiveSynced(proofPath, proofBytes); err != nil {
		return nil, fmt.Errorf("create exclusive attempt proof record: %w", err)
	}
	if err := syncDirectory(dir); err != nil {
		return nil, err
	}
	ok = true
	return &attemptCapability{record: record, recordPath: path, proof: proof, proofPath: proofPath, repositoryRoot: opts.Git.Root, commonDirectory: opts.Git.CommonDir, generated: make(map[int64]string)}, nil
}

func attemptRecordPath(commonDirectory string, panel Panel) string {
	return filepath.Join(commonDirectory, "nous-attempts", "active-causal-diagnosis-v3-"+string(panel)+".json")
}

func attemptProofRecordPath(commonDirectory string, panel Panel) string {
	return filepath.Join(commonDirectory, "nous-attempts", "active-causal-diagnosis-v3-"+string(panel)+"-proof.json")
}

func panelSeedRange(panel Panel) (SeedRange, error) {
	manifest := causalv2.PreregisteredManifest()
	switch panel {
	case PanelTraining:
		return SeedRange{manifest.TrainingSeeds.Start, manifest.TrainingSeeds.Count, manifest.TrainingSeeds.Step}, nil
	case PanelValidation:
		return SeedRange{manifest.ValidationSeeds.Start, manifest.ValidationSeeds.Count, manifest.ValidationSeeds.Step}, nil
	case PanelLocked:
		return SeedRange{manifest.LockedSeeds.Start, manifest.LockedSeeds.Count, manifest.LockedSeeds.Step}, nil
	default:
		return SeedRange{}, errors.New("panel has no protected seed range")
	}
}

// GenerateFixture is available only on a durable protected-panel attempt.
// The capability and its exact preregistered seed range are checked before the
// generator sees the panel name or seed.
func (cap *attemptCapability) generateFixture(seed int64, index int) (PrivateFixture, error) {
	if cap == nil {
		return PrivateFixture{}, errors.New("missing protected-panel capability")
	}
	cap.mu.Lock()
	defer cap.mu.Unlock()
	if cap.consumed || cap.record.State != "started" {
		return PrivateFixture{}, errors.New("protected-panel capability is consumed")
	}
	range_ := cap.record.SeedRange
	if index < 0 || index >= range_.Count || seed != range_.Start+int64(index)*range_.Step {
		return PrivateFixture{}, errors.New("seed/index outside attempt range")
	}
	if cap.generated[seed] != "" {
		return PrivateFixture{}, errors.New("fixture already generated in this attempt")
	}
	var fixture PrivateFixture
	var err error
	if cap.replayOnly {
		if cap.record.Panel != PanelTraining || len(cap.replayFixtures) != cap.record.SeedRange.Count {
			return PrivateFixture{}, errors.New("invalid replay fixture source")
		}
		fixture = cap.replayFixtures[index]
		if fixture.PublicFixture.Seed != seed {
			return PrivateFixture{}, errors.New("supplied replay fixture has wrong seed")
		}
		if _, err = causalv2.VerifyPrivateFixture(mustCanonical(fixture)); err != nil {
			return PrivateFixture{}, fmt.Errorf("supplied replay fixture: %w", err)
		}
	} else {
		if cap.replayFixtures != nil || cap.replayCorruptionFixture != nil {
			return PrivateFixture{}, errors.New("non-replay attempt has a replay fixture source")
		}
		fixture, err = generate(cap.record.Panel, seed, index)
	}
	if err != nil {
		return PrivateFixture{}, err
	}
	cap.generated[seed] = fixture.PublicFixture.FixtureDigest
	if cap.proof.GeneratedFixtures == nil {
		cap.proof.GeneratedFixtures = map[string]string{}
	}
	cap.proof.GeneratedFixtures[fmt.Sprintf("%d", seed)] = fixture.PublicFixture.FixtureDigest
	// Replay regeneration has no acceptance attempt record; real protected
	// panels durably record every opened fixture before returning it.
	if cap.proofPath != "" {
		if err := persistAttemptProofLocked(cap); err != nil {
			cap.record.State = "failed"
			_ = persistAttemptRecordLocked(cap)
			cap.consumed = true
			return PrivateFixture{}, fmt.Errorf("persist generated fixture digest: %w", err)
		}
	}
	return fixture, nil
}

func (cap *attemptCapability) Fail() error { return writeAttemptState(cap, "failed") }

func writeAttemptState(cap *attemptCapability, state string) error {
	if cap == nil || (state != "published" && state != "failed") {
		return errors.New("invalid attempt transition")
	}
	if cap.replayOnly {
		return errors.New("replay regeneration has no attempt-state authority")
	}
	cap.mu.Lock()
	defer cap.mu.Unlock()
	if cap.consumed {
		return errors.New("attempt capability already consumed")
	}
	cap.record.State = state
	err := persistAttemptRecordLocked(cap)
	if err == nil {
		cap.consumed = true
	}
	return err
}

func persistAttemptRecordLocked(cap *attemptCapability) error {
	data, err := canonicalJSON(cap.record)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(cap.recordPath, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = syncDirectory(filepath.Dir(cap.recordPath))
	}
	return err
}

func persistAttemptProofLocked(cap *attemptCapability) error {
	data, err := canonicalJSON(cap.proof)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(cap.proofPath, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = syncDirectory(filepath.Dir(cap.proofPath))
	}
	return err
}

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// BeginTrainingAttempt performs the real repository checks and creates the
// durable, exclusive marker before returning any generation authority.
func beginTrainingAttempt(ctx context.Context, repoRoot string) (*attemptCapability, error) {
	manifest := causalv2.PreregisteredManifest()
	seeds := SeedRange{manifest.TrainingSeeds.Start, manifest.TrainingSeeds.Count, manifest.TrainingSeeds.Step}
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return nil, err
	}
	if err := requireAncestor(ctx, state.Root, PlanCommit, state.Head); err != nil {
		return nil, err
	}
	if err := verifyV3ExecutableConfinement(ctx, state); err != nil {
		return nil, err
	}
	if err := verifyV3CollisionAbsence(state); err != nil {
		return nil, err
	}
	if err := preflightDependencyProof(state.Root, state.Head); err != nil {
		return nil, err
	}
	return beginAttempt(beginAttemptOptions{
		Now:               time.Now(),
		Git:               state,
		Panel:             PanelTraining,
		Seeds:             seeds,
		PretrainingCommit: state.Head,
		EvidenceDirectory: filepath.Join(state.Root, TrainingEvidenceDirectory),
	})
}

var v3ExecutableChanges = []string{
	"A\tdocs/active-causal-diagnosis-v3-amendment.md",
	"A\tinternal/causalv2/dependency_verify.go",
	"A\tinternal/causalv2/dependency_verify_test.go",
	"M\tcmd/nous/main.go",
	"M\tinternal/causalexpv2/dependency_evidence.go",
	"M\tinternal/causalexpv2/dependency_evidence_test.go",
	"M\tinternal/causalexpv2/gates.go",
	"M\tinternal/causalexpv2/provenance.go",
	"M\tinternal/causalexpv2/provenance_test.go",
	"M\tinternal/causalexpv2/publication.go",
	"M\tinternal/causalexpv2/replay_gate.go",
	"M\tinternal/causalexpv2/replay_hook.go",
	"M\tinternal/causalexpv2/result.go",
}

func verifyV3ExecutableConfinement(ctx context.Context, state gitState) error {
	if !state.Clean {
		return errors.New("v3 executable confinement requires a clean worktree")
	}
	changed, err := gitStringOutput(ctx, state.Root, "diff", "--name-status", "--no-renames", BaselineCommit, state.Head, "--")
	if err != nil {
		return err
	}
	got := splitNonemptyLines(changed)
	slices.Sort(got)
	want := append([]string{}, v3ExecutableChanges...)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		return fmt.Errorf("v3 executable diff is outside the accepted status-sensitive allowlist: got %q", got)
	}
	for _, record := range want {
		status, path, ok := strings.Cut(record, "\t")
		if !ok {
			return errors.New("invalid internal v3 executable allowlist")
		}
		headEntry, err := gitStringOutput(ctx, state.Root, "ls-tree", state.Head, "--", path)
		if err != nil || !strings.HasPrefix(headEntry, "100644 blob ") {
			return fmt.Errorf("v3 executable path is not a regular 100644 blob: %s", path)
		}
		baselineEntry, baselineErr := gitStringOutput(ctx, state.Root, "ls-tree", BaselineCommit, "--", path)
		switch status {
		case "A":
			if baselineErr != nil || baselineEntry != "" {
				return fmt.Errorf("v3 added path existed at the baseline: %s", path)
			}
		case "M":
			if baselineErr != nil || !strings.HasPrefix(baselineEntry, "100644 blob ") {
				return fmt.Errorf("v3 modified path was not an existing regular 100644 blob: %s", path)
			}
		default:
			return fmt.Errorf("invalid v3 executable status %q", status)
		}
	}
	acceptedAmendment, err := gitFile(ctx, state.Root, PlanCommit, "docs/active-causal-diagnosis-v3-amendment.md")
	if err != nil {
		return err
	}
	executableAmendment, err := gitFile(ctx, state.Root, state.Head, "docs/active-causal-diagnosis-v3-amendment.md")
	if err != nil || !bytes.Equal(acceptedAmendment, executableAmendment) {
		return errors.New("v3 amendment differs from the accepted plan-commit blob")
	}
	protected := []string{
		"domains/causal", "internal/causalrun", "internal/causalcurriculum", "internal/causaldpproof",
		"internal/causalexpv2/contextual.go", "internal/causalexpv2/control_adapter.go", "internal/causalexpv2/control_verify.go",
		"internal/causalexpv2/curriculum_adapter.go", "internal/causalexpv2/evaluation_executor.go", "internal/causalexpv2/evidence.go",
		"internal/causalexpv2/executor.go", "internal/causalexpv2/freeze.go", "internal/causalexpv2/freeze_verify.go",
		"internal/causalexpv2/generator.go", "internal/causalexpv2/meter.go", "internal/causalexpv2/report.go",
		"internal/causalexpv2/statistics.go", "internal/causalexpv2/training_executor.go", "internal/causalexpv2/verify.go",
	}
	args := append([]string{"diff", "--quiet", BaselineCommit, state.Head, "--"}, protected...)
	command := exec.CommandContext(ctx, "git", append([]string{"-C", state.Root}, args...)...)
	if err := command.Run(); err != nil {
		return errors.New("v3 executable changed a protected empirical or runtime path")
	}
	if err := verifyEmptyFreezeAt(ctx, state.Root, state.Head); err != nil {
		return fmt.Errorf("v3 executable freeze constants: %w", err)
	}
	return nil
}

func splitNonemptyLines(value string) []string {
	if value == "" {
		return []string{}
	}
	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func verifyV3CollisionAbsence(state gitState) error {
	paths := []string{
		filepath.Join(state.Root, TrainingEvidenceDirectory),
		filepath.Join(state.Root, filepath.Dir(TrainingEvidenceDirectory), ".active-causal-diagnosis-v3.staging"),
		replayRecordPath(state.CommonDir),
		resultPath(state.CommonDir, PanelValidation),
		resultPath(state.CommonDir, PanelLocked),
	}
	for _, panel := range []Panel{PanelTraining, PanelValidation, PanelLocked} {
		paths = append(paths, attemptRecordPath(state.CommonDir, panel), attemptProofRecordPath(state.CommonDir, panel))
	}
	// V2 common-directory records are intentionally never named here. These
	// worktree paths are checked for absence only; their bytes are never opened.
	paths = append(paths,
		filepath.Join(state.Root, "docs/evidence/active-causal-diagnosis-v2"),
		filepath.Join(state.Root, "docs/evidence/.active-causal-diagnosis-v2.staging"),
		filepath.Join(state.Root, ResultsDirectoryName, "active-causal-diagnosis-v2-validation.json"),
		filepath.Join(state.Root, ResultsDirectoryName, "active-causal-diagnosis-v2-locked.json"),
	)
	for _, path := range paths {
		if err := requireAbsent(path); err != nil {
			return fmt.Errorf("v3 preflight collision: %w", err)
		}
	}
	return nil
}

func canonicalJSON(value any) ([]byte, error) {
	// Kept local so provenance can be tested before causalv2 proof primitives
	// are linked. The encoder is intentionally identical to the accepted codec.
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return []byte(strings.TrimSuffix(b.String(), "\n")), nil
}
