package causalexpv2

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/chazu/nous/internal/causalv2"
	"golang.org/x/sys/unix"
)

const (
	ReplayInputVersion = "causal-replay-input/v3"
	replayInputFD      = 3
	replayOutputFD     = 4
)

// ReplayInput is the complete transient authority-free input consumed by the
// detached replay worker. Field order is part of its canonical encoding.
type ReplayInput struct {
	ReplayInputVersion string           `json:"replay_input_version"`
	PlanCommit         string           `json:"plan_commit"`
	PretrainingCommit  string           `json:"pretraining_commit"`
	EvidenceCommit     string           `json:"evidence_commit"`
	TrainingDigest     string           `json:"training_digest"`
	BundleDigest       string           `json:"bundle_digest"`
	Fixtures           []PrivateFixture `json:"fixtures"`
	CorruptionFixture  PrivateFixture   `json:"corruption_fixture"`
	ReplayInputDigest  string           `json:"replay_input_digest"`
}

func finalizeReplayInput(input *ReplayInput) ([]byte, error) {
	if input == nil {
		return nil, errors.New("nil replay input")
	}
	input.ReplayInputDigest = ""
	digest, err := causalv2.Digest(ReplayInputVersion, *input)
	if err != nil {
		return nil, err
	}
	input.ReplayInputDigest = digest
	encoded, err := causalv2.CanonicalJSON(*input)
	if err != nil {
		return nil, err
	}
	if _, err := verifyReplayInput(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

func verifyReplayInput(encoded []byte) (ReplayInput, error) {
	input, err := causalv2.StrictDecode[ReplayInput](encoded)
	if err != nil {
		return input, fmt.Errorf("replay input is not canonical: %w", err)
	}
	if input.ReplayInputVersion != ReplayInputVersion || input.PlanCommit != PlanCommit {
		return input, errors.New("replay input has the wrong version or plan commit")
	}
	if !canonicalHex(input.PretrainingCommit, 20) || !canonicalHex(input.EvidenceCommit, 20) || !canonicalHex(input.TrainingDigest, 32) || !canonicalHex(input.BundleDigest, 32) {
		return input, errors.New("replay input has invalid provenance digests")
	}
	manifest := causalv2.PreregisteredManifest()
	if len(input.Fixtures) != manifest.TrainingSeeds.Count {
		return input, errors.New("replay input does not contain exactly 12 training fixtures")
	}
	for index, fixture := range input.Fixtures {
		wantSeed := manifest.TrainingSeeds.Start + int64(index)*manifest.TrainingSeeds.Step
		if fixture.PublicFixture.Seed != wantSeed {
			return input, errors.New("replay training fixtures are not in exact seed order")
		}
		if _, err := causalv2.VerifyPrivateFixture(mustCanonical(fixture)); err != nil {
			return input, fmt.Errorf("replay training fixture %d: %w", index, err)
		}
		publicBytes := mustCanonical(fixture.PublicFixture)
		if _, err := causalv2.VerifyPublicFixtureForPanel(publicBytes, string(PanelTraining)); err != nil {
			return input, fmt.Errorf("replay training fixture %d panel: %w", index, err)
		}
		if err := causalv2.VerifyPreregisteredFixtureContext(fixture.PublicFixture, string(PanelTraining)); err != nil {
			return input, fmt.Errorf("replay training fixture %d context: %w", index, err)
		}
	}
	if input.CorruptionFixture.PublicFixture.Seed != manifest.DevelopmentSeeds.Start {
		return input, errors.New("replay corruption fixture is not development seed 112001")
	}
	if _, err := causalv2.VerifyPrivateFixture(mustCanonical(input.CorruptionFixture)); err != nil {
		return input, fmt.Errorf("replay corruption fixture: %w", err)
	}
	corruptionPublicBytes := mustCanonical(input.CorruptionFixture.PublicFixture)
	if _, err := causalv2.VerifyPublicFixtureForPanel(corruptionPublicBytes, string(PanelDevelopment)); err != nil {
		return input, fmt.Errorf("replay corruption fixture panel: %w", err)
	}
	if err := causalv2.VerifyPreregisteredFixtureContext(input.CorruptionFixture.PublicFixture, string(PanelDevelopment)); err != nil {
		return input, fmt.Errorf("replay corruption fixture context: %w", err)
	}
	want := input
	want.ReplayInputDigest = ""
	digest, err := causalv2.Digest(ReplayInputVersion, want)
	if err != nil || input.ReplayInputDigest != digest {
		return input, errors.New("replay input digest mismatch")
	}
	return input, nil
}

func canonicalHex(value string, bytes int) bool {
	if len(value) != bytes*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == bytes && value == hex.EncodeToString(decoded)
}

// ReplayRegenerate is the no-argument worker entry point. It reads one
// canonical input to EOF from inherited FD 3 and writes only through inherited
// directory FD 4. It has no fixture-generation or publication authority.
func ReplayRegenerate(ctx context.Context) error {
	inputFile := os.NewFile(replayInputFD, "causal-replay-input")
	outputDirectory := os.NewFile(replayOutputFD, "causal-replay-output")
	if inputFile == nil || outputDirectory == nil {
		return errors.New("replay descriptors 3 and 4 are required")
	}
	defer inputFile.Close()
	defer outputDirectory.Close()

	manifest := causalv2.PreregisteredManifest()
	maximumInputBytes := int64((manifest.TrainingSeeds.Count+1)*manifest.TrainingFixtureByteCap + 65536)
	encoded, err := io.ReadAll(io.LimitReader(inputFile, maximumInputBytes+1))
	if err != nil || int64(len(encoded)) > maximumInputBytes {
		return errors.New("replay input exceeds its bounded transport")
	}
	input, err := verifyReplayInput(encoded)
	if err != nil {
		return err
	}
	if err := requireEmptyDirectoryDescriptor(outputDirectory); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	state, err := resolveGitState(ctx, cwd)
	if err != nil {
		return err
	}
	if state.Root != cwd || state.Head != input.PretrainingCommit || !state.Clean {
		return errors.New("replay regeneration is not running in clean detached E")
	}
	if _, err := gitStringOutput(ctx, state.Root, "symbolic-ref", "-q", "HEAD"); err == nil {
		return errors.New("replay regeneration worktree is not detached")
	}
	if err := requireAncestor(ctx, state.Root, input.PretrainingCommit, input.EvidenceCommit); err != nil {
		return err
	}
	if err := verifyEvidenceCommitShape(ctx, state.Root, input.PretrainingCommit, input.EvidenceCommit); err != nil {
		return err
	}
	capability := replayAttemptCapability(input, state)
	evidence, err := regenerateTrainingEvidence(ctx, state.Root, capability)
	if err != nil {
		return err
	}
	report, err := VerifyTrainingReportBytes(evidence.Report)
	if err != nil || report.TrainingDigest != input.TrainingDigest {
		return errors.New("regenerated replay report has the wrong training digest")
	}
	bundle, err := VerifyTrainingBundleBytes(evidence.Bundle)
	if err != nil || bundle.BundleDigest != input.BundleDigest {
		return errors.New("regenerated replay bundle has the wrong bundle digest")
	}
	if err := writeReplayOutputAt(outputDirectory, TrainingReportName, evidence.Report); err != nil {
		return err
	}
	if err := writeReplayOutputAt(outputDirectory, TrainingEpisodesName, evidence.Bundle); err != nil {
		return err
	}
	return outputDirectory.Sync()
}

func requireEmptyDirectoryDescriptor(directory *os.File) error {
	if directory == nil {
		return errors.New("replay output descriptor is absent")
	}
	info, err := directory.Stat()
	if err != nil || !info.IsDir() {
		return errors.New("replay output descriptor is not a directory")
	}
	names, err := directory.Readdirnames(1)
	if err != nil && !errors.Is(err, io.EOF) {
		return errors.New("replay output directory is unreadable")
	}
	if len(names) != 0 {
		return errors.New("replay output directory is not empty")
	}
	return nil
}

func writeReplayOutputAt(directory *os.File, name string, data []byte) error {
	if name != TrainingReportName && name != TrainingEpisodesName {
		return errors.New("replay output name is not allowlisted")
	}
	fd, err := unix.Openat(int(directory.Fd()), name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return fmt.Errorf("create exclusive replay output %s: %w", name, err)
	}
	file := os.NewFile(uintptr(fd), name)
	if file == nil {
		_ = unix.Close(fd)
		return errors.New("open replay output descriptor")
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	return err
}

func replayAttemptCapability(input ReplayInput, state gitState) *attemptCapability {
	manifest := causalv2.PreregisteredManifest()
	return &attemptCapability{
		record: AttemptRecord{
			AttemptVersion:    ReplayCapabilityVersion,
			PlanCommit:        PlanCommit,
			PretrainingCommit: input.PretrainingCommit,
			Panel:             PanelTraining,
			SeedRange:         SeedRange{manifest.TrainingSeeds.Start, manifest.TrainingSeeds.Count, manifest.TrainingSeeds.Step},
			ExecutableCommit:  input.PretrainingCommit,
			State:             "started",
		},
		repositoryRoot:          state.Root,
		commonDirectory:         state.CommonDir,
		generated:               make(map[int64]string),
		replayFixtures:          append([]PrivateFixture(nil), input.Fixtures...),
		replayCorruptionFixture: &input.CorruptionFixture,
		replayOnly:              true,
	}
}
