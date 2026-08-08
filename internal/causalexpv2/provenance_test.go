package causalexpv2

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/chazu/nous/internal/causalv2"
)

func gitTestRepo(t *testing.T) (string, string, string) {
	t.Helper()
	repository := t.TempDir()
	commands := [][]string{{"init"}, {"config", "user.email", "test@example.invalid"}, {"config", "user.name", "Nous Test"}}
	for _, args := range commands {
		command := exec.Command("git", append([]string{"-C", repository}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(repository, "README"), []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "README"}, {"commit", "-m", "initial"}} {
		command := exec.Command("git", append([]string{"-C", repository}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	state, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	return repository, state.CommonDir, state.Head
}

func TestEvidenceCommitMustBeDirectEvidenceOnlyChild(t *testing.T) {
	repository, _, pretraining := gitTestRepo(t)
	directory := filepath.Join(repository, TrainingEvidenceDirectory)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{TrainingEpisodesName, TrainingReportName} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gitCommitAll(t, repository, "evidence")
	state, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyEvidenceCommitShape(context.Background(), repository, pretraining, state.Head); err != nil {
		t.Fatalf("exact direct evidence child rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repository, "extra"), []byte("extra"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, repository, "extra descendant")
	bad, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyEvidenceCommitShape(context.Background(), repository, pretraining, bad.Head); err == nil {
		t.Fatal("non-direct evidence descendant was accepted")
	}

	preexistingRepository, _, _ := gitTestRepo(t)
	preexistingDirectory := filepath.Join(preexistingRepository, TrainingEvidenceDirectory)
	if err := os.MkdirAll(preexistingDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{TrainingEpisodesName, TrainingReportName} {
		if err := os.WriteFile(filepath.Join(preexistingDirectory, name), []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gitCommitAll(t, preexistingRepository, "preexisting evidence")
	preexisting, err := resolveGitState(context.Background(), preexistingRepository)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{TrainingEpisodesName, TrainingReportName} {
		if err := os.WriteFile(filepath.Join(preexistingDirectory, name), []byte("modified"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gitCommitAll(t, preexistingRepository, "modify evidence")
	modified, err := resolveGitState(context.Background(), preexistingRepository)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyEvidenceCommitShape(context.Background(), preexistingRepository, preexisting.Head, modified.Head); err == nil {
		t.Fatal("modified preexisting evidence paths were accepted as newly published evidence")
	}

	mixedRepository, _, mixedPretraining := gitTestRepo(t)
	mixedDirectory := filepath.Join(mixedRepository, TrainingEvidenceDirectory)
	if err := os.MkdirAll(mixedDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{TrainingEpisodesName, TrainingReportName} {
		if err := os.WriteFile(filepath.Join(mixedDirectory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(mixedRepository, "README"), []byte("modified with evidence\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, mixedRepository, "evidence plus unrelated modification")
	mixed, err := resolveGitState(context.Background(), mixedRepository)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyEvidenceCommitShape(context.Background(), mixedRepository, mixedPretraining, mixed.Head); err == nil {
		t.Fatal("evidence additions plus an unrelated modification were accepted")
	}

	deletedRepository, _, deletedPretraining := gitTestRepo(t)
	deletedDirectory := filepath.Join(deletedRepository, TrainingEvidenceDirectory)
	if err := os.MkdirAll(deletedDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{TrainingEpisodesName, TrainingReportName} {
		if err := os.WriteFile(filepath.Join(deletedDirectory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(deletedRepository, "README")); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, deletedRepository, "evidence plus unrelated deletion")
	deleted, err := resolveGitState(context.Background(), deletedRepository)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyEvidenceCommitShape(context.Background(), deletedRepository, deletedPretraining, deleted.Head); err == nil {
		t.Fatal("evidence additions plus an unrelated deletion were accepted")
	}
}

func TestProtectedAuthorizationFailuresStopBeforeGeneration(t *testing.T) {
	repository, _, _ := gitTestRepo(t)
	protectedGeneratorCalls.Store(0)
	if _, err := beginTrainingAttempt(context.Background(), repository); err == nil {
		t.Fatal("unrelated repository unexpectedly authorized training")
	}
	if got := protectedGeneratorCalls.Load(); got != 0 {
		t.Fatalf("training authorization failure opened %d protected fixtures", got)
	}
	if _, err := beginValidationAttempt(context.Background(), repository); err == nil {
		t.Fatal("empty freeze unexpectedly authorized validation")
	}
	if got := protectedGeneratorCalls.Load(); got != 0 {
		t.Fatalf("validation authorization failure opened %d protected fixtures", got)
	}
	if _, err := beginLockedAttempt(context.Background(), repository); err == nil {
		t.Fatal("missing validation result unexpectedly authorized locked panel")
	}
	if got := protectedGeneratorCalls.Load(); got != 0 {
		t.Fatalf("locked authorization failure opened %d protected fixtures", got)
	}
}

func writeSentinel(t *testing.T, path string, value []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, value, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestV3CollisionAndV2ResidueIsolation(t *testing.T) {
	repository, commonDirectory, _ := gitTestRepo(t)
	state, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := []byte("not-json;selected-rule=SENTINEL;score=999")
	v2CommonPaths := []string{filepath.Join(commonDirectory, "nous-attempts", "active-causal-diagnosis-v2-replay.json")}
	for _, panel := range []Panel{PanelTraining, PanelValidation, PanelLocked} {
		v2CommonPaths = append(v2CommonPaths,
			filepath.Join(commonDirectory, "nous-attempts", "active-causal-diagnosis-v2-"+string(panel)+".json"),
			filepath.Join(commonDirectory, "nous-attempts", "active-causal-diagnosis-v2-"+string(panel)+"-proof.json"),
		)
	}
	v2CommonPaths = append(v2CommonPaths,
		filepath.Join(commonDirectory, ResultsDirectoryName, "active-causal-diagnosis-v2-validation.json"),
		filepath.Join(commonDirectory, ResultsDirectoryName, "active-causal-diagnosis-v2-locked.json"),
	)
	for _, path := range v2CommonPaths {
		writeSentinel(t, path, sentinel)
	}
	if err := verifyV3CollisionAbsence(state); err != nil {
		t.Fatalf("opaque v2 common-directory residue blocked v3: %v", err)
	}
	for _, path := range v2CommonPaths {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, sentinel) {
			t.Fatalf("v2 common-directory sentinel changed at %s", path)
		}
	}

	v3Paths := []string{
		filepath.Join(repository, TrainingEvidenceDirectory),
		filepath.Join(repository, filepath.Dir(TrainingEvidenceDirectory), ".active-causal-diagnosis-v3.staging"),
		replayRecordPath(commonDirectory),
		resultPath(commonDirectory, PanelValidation),
		resultPath(commonDirectory, PanelLocked),
	}
	for _, panel := range []Panel{PanelTraining, PanelValidation, PanelLocked} {
		v3Paths = append(v3Paths, attemptRecordPath(commonDirectory, panel), attemptProofRecordPath(commonDirectory, panel))
	}
	for _, path := range v3Paths {
		writeSentinel(t, path, sentinel)
		protectedGeneratorCalls.Store(0)
		if err := verifyV3CollisionAbsence(state); err == nil {
			t.Fatalf("v3 collision was accepted: %s", path)
		}
		if got := protectedGeneratorCalls.Load(); got != 0 {
			t.Fatalf("v3 collision opened %d protected fixtures", got)
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, sentinel) {
			t.Fatalf("v3 collision sentinel changed at %s", path)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}

	v2WorktreePaths := []string{
		filepath.Join(repository, "docs/evidence/active-causal-diagnosis-v2"),
		filepath.Join(repository, "docs/evidence/.active-causal-diagnosis-v2.staging"),
		filepath.Join(repository, ResultsDirectoryName, "active-causal-diagnosis-v2-validation.json"),
		filepath.Join(repository, ResultsDirectoryName, "active-causal-diagnosis-v2-locked.json"),
	}
	for _, path := range v2WorktreePaths {
		writeSentinel(t, path, sentinel)
		if err := verifyV3CollisionAbsence(state); err == nil {
			t.Fatalf("v2 worktree collision was accepted: %s", path)
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, sentinel) {
			t.Fatalf("v2 worktree sentinel changed at %s", path)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}

	protectedGeneratorCalls.Store(0)
	if _, err := beginTrainingAttempt(context.Background(), repository); err == nil {
		t.Fatal("unrelated repository unexpectedly gained v3 authorization from v2 residue")
	}
	if got := protectedGeneratorCalls.Load(); got != 0 {
		t.Fatalf("failed v3 authorization opened %d protected fixtures", got)
	}
}

func TestPrepublicationFailureDoesNotExposeEmpiricalSentinels(t *testing.T) {
	repository, commonDirectory, head := gitTestRepo(t)
	state, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	manifest := causalv2.PreregisteredManifest()
	capability, err := beginAttempt(beginAttemptOptions{
		Now:               time.Unix(0, 0).UTC(),
		Git:               state,
		Panel:             PanelTraining,
		Seeds:             SeedRange{manifest.TrainingSeeds.Start, manifest.TrainingSeeds.Count, manifest.TrainingSeeds.Step},
		PretrainingCommit: head,
		EvidenceDirectory: filepath.Join(repository, TrainingEvidenceDirectory),
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < manifest.TrainingSeeds.Count; index++ {
		seed := manifest.TrainingSeeds.Start + int64(index)*manifest.TrainingSeeds.Step
		capability.generated[seed] = strings.Repeat("a", 64)
	}
	sentinels := []string{"PRIVATE_FIXTURE_SENTINEL", "SELECTED_RULE_SENTINEL", "ACTION_SENTINEL", "OUTCOME_SENTINEL", "SCORE_SENTINEL", "AGGREGATE_SENTINEL"}
	payload := []byte(strings.Join(sentinels, ":"))
	failure := capability.publishTrainingEvidence(repository, EvidenceBytes{Report: payload, Bundle: payload})
	if failure == nil {
		t.Fatal("forced prepublication failure unexpectedly succeeded")
	}
	captured := failure.Error() // Direct library execution has no stdout/stderr channel.
	for _, sentinel := range sentinels {
		if strings.Contains(captured, sentinel) {
			t.Fatalf("prepublication failure exposed %q", sentinel)
		}
	}
	for _, path := range []string{
		filepath.Join(repository, TrainingEvidenceDirectory),
		filepath.Join(repository, filepath.Dir(TrainingEvidenceDirectory), ".active-causal-diagnosis-v3.staging"),
	} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("forced failure created empirical path %s", path)
		}
	}
	recordBytes, err := os.ReadFile(attemptRecordPath(commonDirectory, PanelTraining))
	if err != nil {
		t.Fatal(err)
	}
	record, err := causalv2.StrictDecode[AttemptRecord](recordBytes)
	if err != nil || record.State != "failed" {
		t.Fatalf("forced failure record = %+v, err=%v", record, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(recordBytes, &fields); err != nil {
		t.Fatal(err)
	}
	wantFields := []string{"attempt_version", "created_utc", "executable_commit", "panel", "plan_commit", "pretraining_commit", "seed_range", "state"}
	gotFields := make([]string, 0, len(fields))
	for name := range fields {
		gotFields = append(gotFields, name)
	}
	slices.Sort(gotFields)
	if !slices.Equal(gotFields, wantFields) {
		t.Fatalf("failure record fields = %v", gotFields)
	}
	proofBytes, err := os.ReadFile(attemptProofRecordPath(commonDirectory, PanelTraining))
	if err != nil {
		t.Fatal(err)
	}
	proof, err := causalv2.StrictDecode[AttemptProofRecord](proofBytes)
	if err != nil || proof.ProofVersion != AttemptProofVersion || proof.GeneratedFixtures == nil || len(proof.GeneratedFixtures) != 0 || proof.PublishedDigest != "" {
		t.Fatalf("forced failure proof = %+v, err=%v", proof, err)
	}
	for _, sentinel := range sentinels {
		if bytes.Contains(recordBytes, []byte(sentinel)) || bytes.Contains(proofBytes, []byte(sentinel)) {
			t.Fatalf("failure provenance exposed %q", sentinel)
		}
	}
}

func TestProtectedPanelFailureHelperProcess(t *testing.T) {
	if os.Getenv("NOUS_TEST_PROTECTED_FAILURE_HELPER") != "1" {
		return
	}
	payload := os.Getenv("NOUS_TEST_PROTECTED_FAILURE_PAYLOAD")
	repository := os.Getenv("NOUS_TEST_PROTECTED_FAILURE_ROOT")
	state, stateErr := resolveGitState(context.Background(), repository)
	if stateErr != nil {
		os.Exit(28)
	}
	manifest := causalv2.PreregisteredManifest()
	capability, attemptErr := beginAttempt(beginAttemptOptions{Now: time.Unix(0, 0).UTC(), Git: state, Panel: PanelTraining, Seeds: SeedRange{manifest.TrainingSeeds.Start, manifest.TrainingSeeds.Count, manifest.TrainingSeeds.Step}, PretrainingCommit: state.Head, EvidenceDirectory: filepath.Join(repository, TrainingEvidenceDirectory)})
	if attemptErr != nil {
		os.Exit(27)
	}
	err := executeProtectedPanelWith(context.Background(), repository, PanelTraining, func(context.Context, string, Panel) error {
		return errors.New(payload)
	})
	if err == nil {
		os.Exit(29)
	}
	if failErr := capability.Fail(); failErr != nil {
		os.Exit(26)
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(23)
}

func TestProtectedPanelCommandBoundaryCapturesNoEmpiricalValues(t *testing.T) {
	sentinels := []string{"PRIVATE_FIXTURE_SENTINEL", "SELECTED_RULE_SENTINEL", "RULE_SENTINEL", "ACTION_SENTINEL", "OUTCOME_SENTINEL", "SCORE_SENTINEL", "AGGREGATE_SENTINEL"}
	payload := strings.Join(sentinels, ":")
	for _, panel := range []Panel{PanelTraining, PanelValidation, PanelLocked} {
		err := executeProtectedPanelWith(context.Background(), t.TempDir(), panel, func(context.Context, string, Panel) error {
			return errors.New(payload)
		})
		if err == nil {
			t.Fatalf("%s injected failure succeeded", panel)
		}
		for _, sentinel := range sentinels {
			if strings.Contains(err.Error(), sentinel) {
				t.Fatalf("%s returned error exposed %q", panel, sentinel)
			}
		}
	}

	filesystem, commonDirectory, _ := gitTestRepo(t)
	command := exec.Command(os.Args[0], "-test.run=^TestProtectedPanelFailureHelperProcess$")
	command.Env = append(os.Environ(),
		"NOUS_TEST_PROTECTED_FAILURE_HELPER=1",
		"NOUS_TEST_PROTECTED_FAILURE_PAYLOAD="+payload,
		"NOUS_TEST_PROTECTED_FAILURE_ROOT="+filesystem,
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	returned := command.Run()
	if returned == nil {
		t.Fatal("forced-failure helper unexpectedly succeeded")
	}
	captured := stdout.String() + stderr.String() + returned.Error()
	for _, sentinel := range sentinels {
		if strings.Contains(captured, sentinel) {
			t.Fatalf("forced-failure command exposed %q in captured channels", sentinel)
		}
	}
	rootEntries, err := os.ReadDir(filesystem)
	if err != nil {
		t.Fatal(err)
	}
	if len(rootEntries) != 2 || rootEntries[0].Name() != ".git" || rootEntries[1].Name() != "README" {
		t.Fatalf("forced-failure command changed worktree entries: %v", rootEntries)
	}
	for _, path := range []string{
		filepath.Join(filesystem, TrainingEvidenceDirectory),
		filepath.Join(filesystem, filepath.Dir(TrainingEvidenceDirectory), ".active-causal-diagnosis-v3.staging"),
		resultPath(commonDirectory, PanelValidation),
		resultPath(commonDirectory, PanelLocked),
	} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("forced-failure command created forbidden path %s", path)
		}
	}
	recordBytes, err := os.ReadFile(attemptRecordPath(commonDirectory, PanelTraining))
	if err != nil {
		t.Fatal(err)
	}
	record, err := causalv2.StrictDecode[AttemptRecord](recordBytes)
	if err != nil || record.State != "failed" || record.AttemptVersion != AttemptVersion || record.PlanCommit != PlanCommit {
		t.Fatalf("forced-failure command attempt record = %+v, err=%v", record, err)
	}
	proofBytes, err := os.ReadFile(attemptProofRecordPath(commonDirectory, PanelTraining))
	if err != nil {
		t.Fatal(err)
	}
	proof, err := causalv2.StrictDecode[AttemptProofRecord](proofBytes)
	if err != nil || proof.ProofVersion != AttemptProofVersion || proof.Panel != PanelTraining || proof.PublishedDigest != "" || proof.GeneratedFixtures == nil || len(proof.GeneratedFixtures) != 0 {
		t.Fatalf("forced-failure command proof record = %+v, err=%v", proof, err)
	}
	for _, encoded := range [][]byte{recordBytes, proofBytes} {
		for _, sentinel := range sentinels {
			if bytes.Contains(encoded, []byte(sentinel)) {
				t.Fatalf("forced-failure provenance exposed %q", sentinel)
			}
		}
	}
}

func helperExecutable(t *testing.T) regenerationExecutable {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err := validateRegenerationExecutable(regenerationExecutable{Path: path, PrefixArgs: []string{"-test.run=TestReplayHelperProcess", "--"}})
	if err != nil {
		t.Fatal(err)
	}
	return executable
}

func TestReplayHelperProcess(t *testing.T) {
	if os.Getenv("NOUS_TEST_REPLAY_HELPER") != "1" {
		return
	}
	if os.Getenv("NOUS_TEST_REPLAY_FAIL") == "1" {
		os.Exit(23)
	}
	inputFile := os.NewFile(replayInputFD, "test-replay-input")
	output := os.NewFile(replayOutputFD, "test-replay-output")
	if inputFile == nil || output == nil {
		os.Exit(25)
	}
	inputBytes, err := io.ReadAll(inputFile)
	if err != nil {
		os.Exit(26)
	}
	if _, err := verifyReplayInput(inputBytes); err != nil {
		os.Exit(24)
	}
	decode := func(name string) []byte {
		value, err := base64.StdEncoding.DecodeString(os.Getenv(name))
		if err != nil {
			os.Exit(27)
		}
		return value
	}
	if err := writeReplayOutputAt(output, TrainingReportName, decode("NOUS_TEST_REPLAY_REPORT")); err != nil {
		os.Exit(28)
	}
	if err := writeReplayOutputAt(output, TrainingEpisodesName, decode("NOUS_TEST_REPLAY_BUNDLE")); err != nil {
		os.Exit(29)
	}
}

func replayCapabilityForTest(t *testing.T, repository, head string, report, bundle []byte) *ReplayCapability {
	t.Helper()
	inputBytes := replayInputForTest(t, head)
	trainingDigest, bundleDigest := strings.Repeat("a", 64), strings.Repeat("b", 64)
	return &ReplayCapability{repositoryRoot: repository, pretrainingCommit: head, evidenceCommit: head, seeds: SeedRange{122001, 12, 1}, reportDigest: trainingDigest, bundleDigest: bundleDigest, reportBytes: report, bundleBytes: bundle, replayInputBytes: inputBytes, workerExecutable: helperExecutable(t)}
}

func replayInputForTest(t *testing.T, head string) []byte {
	t.Helper()
	development, err := NewDiagnosticDevelopmentCapability().GenerateDevelopment(causalv2.PreregisteredManifest().DevelopmentSeeds.Start, 0)
	if err != nil {
		t.Fatal(err)
	}
	fixtures := make([]PrivateFixture, causalv2.PreregisteredManifest().TrainingSeeds.Count)
	for index := range fixtures {
		developmentSeed := causalv2.PreregisteredManifest().DevelopmentSeeds.Start + int64(index)*causalv2.PreregisteredManifest().DevelopmentSeeds.Step
		fixtures[index], err = NewDiagnosticDevelopmentCapability().GenerateDevelopment(developmentSeed, index)
		if err != nil {
			t.Fatal(err)
		}
		fixtures[index].PublicFixture.Seed = causalv2.PreregisteredManifest().TrainingSeeds.Start + int64(index)*causalv2.PreregisteredManifest().TrainingSeeds.Step
		fixtures[index].PublicFixture.OpaqueToken, err = causalv2.PublicToken(string(PanelTraining), fixtures[index].PublicFixture.Seed, fixtures[index].PublicFixture.GeneratorAttempt)
		if err != nil {
			t.Fatal(err)
		}
		fixtures[index].PublicFixture.FixtureDigest = ""
		fixtures[index].PrivateFixtureDigest = ""
		if err := causalv2.SignPublicFixture(&fixtures[index].PublicFixture); err != nil {
			t.Fatal(err)
		}
		if err := causalv2.SignPrivateFixture(&fixtures[index]); err != nil {
			t.Fatal(err)
		}
	}
	trainingDigest, bundleDigest := strings.Repeat("a", 64), strings.Repeat("b", 64)
	input := ReplayInput{ReplayInputVersion: ReplayInputVersion, PlanCommit: PlanCommit, PretrainingCommit: head, EvidenceCommit: head, TrainingDigest: trainingDigest, BundleDigest: bundleDigest, Fixtures: fixtures, CorruptionFixture: development}
	inputBytes, err := finalizeReplayInput(&input)
	if err != nil {
		t.Fatal(err)
	}
	return inputBytes
}

func TestReplayInputRejectsTransportAndFixtureAttacks(t *testing.T) {
	_, _, head := gitTestRepo(t)
	valid := replayInputForTest(t, head)
	if _, err := verifyReplayInput(valid); err != nil {
		t.Fatalf("valid replay input: %v", err)
	}
	if _, err := verifyReplayInput(append(append([]byte(nil), valid...), '\n')); err == nil {
		t.Fatal("replay input accepted noncanonical trailing whitespace")
	}
	input, err := causalv2.StrictDecode[ReplayInput](valid)
	if err != nil {
		t.Fatal(err)
	}
	input.Fixtures[0], input.Fixtures[1] = input.Fixtures[1], input.Fixtures[0]
	reordered := mustCanonical(input)
	if _, err := verifyReplayInput(reordered); err == nil {
		t.Fatal("replay input accepted reordered protected fixtures")
	}
	input, _ = causalv2.StrictDecode[ReplayInput](valid)
	input.Fixtures = input.Fixtures[:len(input.Fixtures)-1]
	if _, err := verifyReplayInput(mustCanonical(input)); err == nil {
		t.Fatal("replay input accepted an incomplete protected fixture set")
	}
	input, _ = causalv2.StrictDecode[ReplayInput](valid)
	for _, hypothesis := range input.Fixtures[0].PublicFixture.InitialPosterior {
		if hypothesis != input.Fixtures[0].HiddenHypothesis {
			input.Fixtures[0].HiddenHypothesis = hypothesis
			break
		}
	}
	if _, err := verifyReplayInput(mustCanonical(input)); err == nil {
		t.Fatal("replay input accepted changed private fixture bytes")
	}
	input, _ = causalv2.StrictDecode[ReplayInput](valid)
	input.ReplayInputDigest = strings.Repeat("0", 64)
	if _, err := verifyReplayInput(mustCanonical(input)); err == nil {
		t.Fatal("replay input accepted the wrong self-digest")
	}
}

func TestReplayStateCannotFallBackToGeneratorOrPublish(t *testing.T) {
	manifest := causalv2.PreregisteredManifest()
	capability := &attemptCapability{
		record:     AttemptRecord{Panel: PanelTraining, SeedRange: SeedRange{manifest.TrainingSeeds.Start, manifest.TrainingSeeds.Count, manifest.TrainingSeeds.Step}, State: "started"},
		generated:  make(map[int64]string),
		replayOnly: true,
	}
	if _, err := capability.generateFixture(manifest.TrainingSeeds.Start, 0); err == nil {
		t.Fatal("replay state fell back to a protected fixture generator")
	}
	if err := capability.publishTrainingEvidence(t.TempDir(), EvidenceBytes{}); err == nil {
		t.Fatal("replay state acquired training publication authority")
	}
	if err := capability.publishEvaluationReport(context.Background(), t.TempDir(), nil); err == nil {
		t.Fatal("replay state acquired evaluation publication authority")
	}
	if err := capability.Fail(); err == nil {
		t.Fatal("replay state acquired attempt-state authority")
	}
}

func TestReplayOutputDescriptorRejectsOccupiedAndSymlinkNames(t *testing.T) {
	directory := t.TempDir()
	handle, err := os.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	if err := os.WriteFile(filepath.Join(directory, "occupied"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := requireEmptyDirectoryDescriptor(handle); err == nil {
		t.Fatal("replay accepted a nonempty output directory")
	}
	if err := os.Remove(filepath.Join(directory, "occupied")); err != nil {
		t.Fatal(err)
	}
	if _, err := handle.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(directory, TrainingReportName)); err != nil {
		t.Fatal(err)
	}
	if err := writeReplayOutputAt(handle, TrainingReportName, []byte("changed")); err == nil {
		t.Fatal("replay followed or replaced an output symlink")
	}
	if _, err := readReplayOutputAt(handle, TrainingReportName, 1024); err == nil {
		t.Fatal("replay parent followed an output symlink")
	}
	contents, err := os.ReadFile(outside)
	if err != nil || string(contents) != "unchanged" {
		t.Fatal("replay output symlink changed an outside file")
	}
	extraDirectory := t.TempDir()
	for _, name := range []string{TrainingReportName, TrainingEpisodesName, "unexpected"} {
		if err := os.WriteFile(filepath.Join(extraDirectory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	extraHandle, err := os.Open(extraDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer extraHandle.Close()
	if err := requireExactReplayOutputs(extraHandle); err == nil {
		t.Fatal("replay parent accepted an extra worker output")
	}
}

func assertOneWorktree(t *testing.T, repository string) {
	t.Helper()
	command := exec.Command("git", "-C", repository, "worktree", "list", "--porcelain")
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(output), "worktree "); count != 1 {
		t.Fatalf("worktree cleanup left %d registered worktrees:\n%s", count, output)
	}
}

func TestReplayOwnsWorktreeInvocationComparisonAndCleanup(t *testing.T) {
	repository, _, head := gitTestRepo(t)
	report, bundle := []byte("report-bytes"), []byte("bundle-bytes")
	t.Setenv("NOUS_TEST_REPLAY_HELPER", "1")
	t.Setenv("NOUS_TEST_REPLAY_REPORT", base64.StdEncoding.EncodeToString(report))
	t.Setenv("NOUS_TEST_REPLAY_BUNDLE", base64.StdEncoding.EncodeToString(bundle))
	capability := replayCapabilityForTest(t, repository, head, report, bundle)
	result, err := capability.Replay(context.Background())
	if err != nil || !result.ReportEqual || !result.BundleEqual {
		t.Fatalf("replay result=%+v err=%v", result, err)
	}
	assertOneWorktree(t, repository)
	if _, err := capability.Replay(context.Background()); err == nil {
		t.Fatal("single-use replay capability was reusable")
	}
}

func TestReplayCleansAfterSubprocessFailureAndMismatch(t *testing.T) {
	repository, _, head := gitTestRepo(t)
	report, bundle := []byte("report"), []byte("bundle")
	t.Setenv("NOUS_TEST_REPLAY_HELPER", "1")
	t.Setenv("NOUS_TEST_REPLAY_REPORT", base64.StdEncoding.EncodeToString(report))
	t.Setenv("NOUS_TEST_REPLAY_BUNDLE", base64.StdEncoding.EncodeToString(bundle))
	t.Setenv("NOUS_TEST_REPLAY_FAIL", "1")
	if _, err := replayCapabilityForTest(t, repository, head, report, bundle).Replay(context.Background()); err == nil {
		t.Fatal("replay accepted failed regeneration subprocess")
	}
	assertOneWorktree(t, repository)
	t.Setenv("NOUS_TEST_REPLAY_FAIL", "0")
	t.Setenv("NOUS_TEST_REPLAY_REPORT", base64.StdEncoding.EncodeToString([]byte("different")))
	result, err := replayCapabilityForTest(t, repository, head, report, bundle).Replay(context.Background())
	if err == nil || result.ReportEqual || !result.BundleEqual {
		t.Fatalf("mismatch result=%+v err=%v", result, err)
	}
	assertOneWorktree(t, repository)
}

func TestReplaySuccessRecordBindsEToRToCandidate(t *testing.T) {
	repository, commonDirectory, _ := gitTestRepo(t)
	freezePath := filepath.Join(repository, FrozenConstantsPath)
	if err := os.MkdirAll(filepath.Dir(freezePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(freezePath, expectedFreezeFile("", "", ""), 0o600); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, repository, "E")
	eState, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "evidence"), []byte("committed evidence\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, repository, "R")
	rState, _ := resolveGitState(context.Background(), repository)
	trainingDigest, bundleDigest := strings.Repeat("a", 64), strings.Repeat("b", 64)
	if err := os.WriteFile(freezePath, expectedFreezeFile("P=H;M=gain;S=C", rState.Head, trainingDigest), 0o600); err != nil {
		t.Fatal(err)
	}
	dirtyR, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyCandidateConstantsState(context.Background(), dirtyR, rState.Head); err != nil {
		t.Fatalf("prescribed uncommitted constants edit at R rejected: %v", err)
	}
	diffDigest, err := candidateDiffDigest(context.Background(), repository, rState.Head)
	if err != nil {
		t.Fatal(err)
	}
	record := replaySuccessRecord{ReplayVersion: replayRecordVersion, PlanCommit: PlanCommit, PretrainingCommit: eState.Head, EvidenceCommit: rState.Head, CandidateCommit: rState.Head, CandidateDiffDigest: diffDigest, TrainingDigest: trainingDigest, BundleDigest: bundleDigest, CreatedUTC: time.Now().UTC().Format(time.RFC3339), State: "succeeded"}
	if err := createReplayRecord(commonDirectory, record); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, repository, "C")
	cState, _ := resolveGitState(context.Background(), repository)
	if err := verifyCandidateConstantsState(context.Background(), cState, rState.Head); err != nil {
		t.Fatalf("prescribed direct constants-only child C rejected: %v", err)
	}
	report := TrainingReport{PretrainingCommit: eState.Head, TrainingDigest: trainingDigest}
	bundle := TrainingBundle{BundleDigest: bundleDigest}
	if err := verifyReplaySuccessRecord(context.Background(), cState, report, bundle, rState.Head); err != nil {
		t.Fatal(err)
	}
	legacy := record
	legacy.ReplayVersion = "causal-replay-success/v2"
	if err := persistReplayRecord(commonDirectory, legacy); err != nil {
		t.Fatal(err)
	}
	if err := verifyReplaySuccessRecord(context.Background(), cState, report, bundle, rState.Head); err == nil {
		t.Fatal("v3 accepted a v2 replay success record")
	}
	if err := persistReplayRecord(commonDirectory, record); err != nil {
		t.Fatal(err)
	}
	forged := record
	forged.CandidateCommit = cState.Head
	if err := persistReplayRecord(commonDirectory, forged); err != nil {
		t.Fatal(err)
	}
	if err := verifyReplaySuccessRecord(context.Background(), cState, report, bundle, rState.Head); err == nil {
		t.Fatal("accepted receipt forged as a replay performed after candidate commit")
	}
	if err := persistReplayRecord(commonDirectory, record); err != nil {
		t.Fatal(err)
	}
	report.TrainingDigest = strings.Repeat("c", 64)
	if err := verifyReplaySuccessRecord(context.Background(), cState, report, bundle, rState.Head); err == nil {
		t.Fatal("replay record accepted different committed evidence")
	}
}

func TestAttemptRecordRejectsCompanionProofFields(t *testing.T) {
	record := AttemptRecord{AttemptVersion: AttemptVersion, PlanCommit: PlanCommit, PretrainingCommit: strings.Repeat("a", 40), Panel: PanelValidation, SeedRange: SeedRange{Start: 132001, Count: 32, Step: 1}, ExecutableCommit: strings.Repeat("b", 40), CreatedUTC: time.Now().UTC().Format(time.RFC3339), State: "started"}
	var object map[string]any
	if err := json.Unmarshal(mustCanonical(record), &object); err != nil {
		t.Fatal(err)
	}
	object["generated_fixtures"] = map[string]string{"132001": strings.Repeat("c", 64)}
	object["published_digest"] = strings.Repeat("d", 64)
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := causalv2.StrictDecode[AttemptRecord](encoded); err == nil {
		t.Fatal("attempt record accepted proof-only fields")
	}
}

func TestV3ProtocolIdentitiesRejectV2AndViceVersa(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	baselineProvenance, err := gitFile(context.Background(), root, BaselineCommit, "internal/causalexpv2/provenance.go")
	if err != nil || !bytes.Contains(baselineProvenance, []byte(`"causal-attempt/v2"`)) || !bytes.Contains(baselineProvenance, []byte(legacyV2PlanCommit)) {
		t.Fatal("test v2 attempt decoder is not tied to baseline source identities")
	}
	baselineReplay, err := gitFile(context.Background(), root, BaselineCommit, "internal/causalexpv2/replay_hook.go")
	if err != nil || !bytes.Contains(baselineReplay, []byte(`"causal-replay-input/v2"`)) || !bytes.Contains(baselineReplay, []byte(`"causal-replay/v2"`)) {
		t.Fatal("test v2 replay decoder is not tied to baseline source identities")
	}
	baselineSuccess, err := gitFile(context.Background(), root, BaselineCommit, "internal/causalexpv2/replay_gate.go")
	if err != nil || !bytes.Contains(baselineSuccess, []byte(`"causal-replay-success/v2"`)) {
		t.Fatal("test v2 success decoder is not tied to baseline source identities")
	}

	attempt := AttemptRecord{AttemptVersion: AttemptVersion, PlanCommit: PlanCommit, Panel: PanelValidation}
	proof := AttemptProofRecord{ProofVersion: AttemptProofVersion, Panel: PanelValidation}
	if !validAttemptProtocolIdentity(attempt, proof, PanelValidation) {
		t.Fatal("v3 attempt/proof identity was rejected")
	}
	attempt.AttemptVersion = "causal-attempt/v2"
	if validAttemptProtocolIdentity(attempt, proof, PanelValidation) {
		t.Fatal("v3 accepted a v2 attempt record")
	}
	attempt.AttemptVersion = AttemptVersion
	proof.ProofVersion = "causal-attempt-proof/v2"
	if validAttemptProtocolIdentity(attempt, proof, PanelValidation) {
		t.Fatal("v3 accepted a v2 proof record")
	}
	if _, err := decodeV2AttemptRecordForTest(mustCanonical(AttemptRecord{AttemptVersion: AttemptVersion, PlanCommit: PlanCommit})); err == nil {
		t.Fatal("baseline-v2 decoder accepted a v3 attempt record")
	}
	legacyAttempt := attempt
	legacyAttempt.AttemptVersion, legacyAttempt.PlanCommit = "causal-attempt/v2", legacyV2PlanCommit
	if _, err := decodeV2AttemptRecordForTest(mustCanonical(legacyAttempt)); err != nil {
		t.Fatalf("baseline-v2 decoder rejected its own attempt identity: %v", err)
	}
	if _, err := decodeV2AttemptProofForTest(mustCanonical(AttemptProofRecord{ProofVersion: AttemptProofVersion, GeneratedFixtures: map[string]string{}})); err == nil {
		t.Fatal("baseline-v2 decoder accepted a v3 proof record")
	}
	legacyProof := proof
	legacyProof.ProofVersion = "causal-attempt-proof/v2"
	if _, err := decodeV2AttemptProofForTest(mustCanonical(legacyProof)); err != nil {
		t.Fatalf("baseline-v2 decoder rejected its own proof identity: %v", err)
	}

	encoded := replayInputForTest(t, strings.Repeat("a", 40))
	input, err := causalv2.StrictDecode[ReplayInput](encoded)
	if err != nil {
		t.Fatal(err)
	}
	input.ReplayInputVersion = "causal-replay-input/v2"
	legacyBytes := mustCanonical(input)
	if _, err := verifyReplayInput(legacyBytes); err == nil {
		t.Fatal("v3 accepted a v2 replay input")
	}
	if _, err := decodeV2ReplayInputForTest(encoded); err == nil {
		t.Fatal("baseline-v2 decoder accepted a v3 replay input")
	}
	input.PlanCommit = legacyV2PlanCommit
	if _, err := decodeV2ReplayInputForTest(mustCanonical(input)); err != nil {
		t.Fatalf("baseline-v2 decoder rejected its own replay-input identity: %v", err)
	}
	worker := replayAttemptCapability(input, gitState{Head: input.PretrainingCommit})
	if _, err := decodeV2ReplayWorkerForTest(mustCanonical(worker.record)); err == nil {
		t.Fatal("baseline-v2 decoder accepted a v3 replay worker record")
	}
	legacyWorker := worker.record
	legacyWorker.AttemptVersion, legacyWorker.PlanCommit = "causal-replay/v2", legacyV2PlanCommit
	if _, err := decodeV2ReplayWorkerForTest(mustCanonical(legacyWorker)); err != nil {
		t.Fatalf("baseline-v2 decoder rejected its own worker identity: %v", err)
	}
	success := replaySuccessRecord{ReplayVersion: replayRecordVersion, PlanCommit: PlanCommit}
	if _, err := decodeV2ReplaySuccessForTest(mustCanonical(success)); err == nil {
		t.Fatal("baseline-v2 decoder accepted a v3 replay success record")
	}
	legacySuccess := success
	legacySuccess.ReplayVersion, legacySuccess.PlanCommit = "causal-replay-success/v2", legacyV2PlanCommit
	if _, err := decodeV2ReplaySuccessForTest(mustCanonical(legacySuccess)); err != nil {
		t.Fatalf("baseline-v2 decoder rejected its own success identity: %v", err)
	}
	v3Digest, err := causalv2.Digest("causal-replay-candidate-diff/v3", struct {
		EvidenceCommit string `json:"evidence_commit"`
		Diff           []byte `json:"diff"`
	}{strings.Repeat("a", 40), []byte("same")})
	if err != nil {
		t.Fatal(err)
	}
	v2Digest, err := causalv2.Digest("causal-replay-candidate-diff/v2", struct {
		EvidenceCommit string `json:"evidence_commit"`
		Diff           []byte `json:"diff"`
	}{strings.Repeat("a", 40), []byte("same")})
	if err != nil || v2Digest == v3Digest {
		t.Fatal("candidate diff digest domains are not isolated")
	}
}

const legacyV2PlanCommit = "6a3ac6d6debb7ab4f85e6c2a12842076d6392936"

func decodeV2AttemptRecordForTest(encoded []byte) (AttemptRecord, error) {
	record, err := causalv2.StrictDecode[AttemptRecord](encoded)
	if err != nil || !bytes.Equal(encoded, mustCanonical(record)) {
		return record, errors.New("v2 attempt record is not canonical")
	}
	if record.AttemptVersion != "causal-attempt/v2" || record.PlanCommit != legacyV2PlanCommit {
		return record, errors.New("not a v2 attempt record")
	}
	return record, nil
}

func decodeV2AttemptProofForTest(encoded []byte) (AttemptProofRecord, error) {
	record, err := causalv2.StrictDecode[AttemptProofRecord](encoded)
	if err != nil || !bytes.Equal(encoded, mustCanonical(record)) {
		return record, errors.New("v2 attempt proof is not canonical")
	}
	if record.ProofVersion != "causal-attempt-proof/v2" {
		return record, errors.New("not a v2 attempt proof")
	}
	return record, nil
}

func decodeV2ReplayInputForTest(encoded []byte) (ReplayInput, error) {
	record, err := causalv2.StrictDecode[ReplayInput](encoded)
	if err != nil || !bytes.Equal(encoded, mustCanonical(record)) {
		return record, errors.New("v2 replay input is not canonical")
	}
	if record.ReplayInputVersion != "causal-replay-input/v2" || record.PlanCommit != legacyV2PlanCommit {
		return record, errors.New("not a v2 replay input")
	}
	return record, nil
}

func decodeV2ReplayWorkerForTest(encoded []byte) (AttemptRecord, error) {
	record, err := causalv2.StrictDecode[AttemptRecord](encoded)
	if err != nil || !bytes.Equal(encoded, mustCanonical(record)) {
		return record, errors.New("v2 replay worker record is not canonical")
	}
	if record.AttemptVersion != "causal-replay/v2" || record.PlanCommit != legacyV2PlanCommit {
		return record, errors.New("not a v2 replay worker record")
	}
	return record, nil
}

func decodeV2ReplaySuccessForTest(encoded []byte) (replaySuccessRecord, error) {
	record, err := causalv2.StrictDecode[replaySuccessRecord](encoded)
	if err != nil || !bytes.Equal(encoded, mustCanonical(record)) {
		return record, errors.New("v2 replay success record is not canonical")
	}
	if record.ReplayVersion != "causal-replay-success/v2" || record.PlanCommit != legacyV2PlanCommit {
		return record, errors.New("not a v2 replay success record")
	}
	return record, nil
}

func locateV3ExecutableCommitForTest(t *testing.T, repository string) string {
	t.Helper()
	command := exec.Command("git", "-C", repository, "rev-list", "--reverse", PlanCommit+"..HEAD")
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	state, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range strings.Fields(string(output)) {
		candidateState := state
		candidateState.Head = candidate
		candidateState.Clean = true
		if verifyV3ExecutableConfinement(context.Background(), candidateState) == nil {
			return candidate
		}
	}
	t.Skip("v3 executable commit has not been created yet")
	return ""
}

func cloneAtCommitForTest(t *testing.T, source, commit string) string {
	t.Helper()
	repository := filepath.Join(t.TempDir(), "repository")
	command := exec.Command("git", "clone", "--no-hardlinks", source, repository)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("clone: %v: %s", err, output)
	}
	for _, args := range [][]string{{"config", "user.email", "test@example.invalid"}, {"config", "user.name", "Nous Test"}, {"checkout", "--detach", commit}} {
		command = exec.Command("git", append([]string{"-C", repository}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	return repository
}

func TestV3ExecutableConfinementRejectsInjectedChanges(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	e3 := locateV3ExecutableCommitForTest(t, root)
	repository := cloneAtCommitForTest(t, root, e3)
	state, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyV3ExecutableConfinement(context.Background(), state); err != nil {
		t.Fatalf("accepted E3 rejected: %v", err)
	}

	cases := []struct {
		name   string
		path   string
		mode   bool
		revert bool
	}{
		{"causal domain", "domains/causal/types.cue", false, false},
		{"causal runner", "internal/causalrun/dependency.go", false, false},
		{"causal curriculum", "internal/causalcurriculum/curriculum.go", false, false},
		{"DP proof", "internal/causaldpproof/exact.go", false, false},
		{"experiment generator", "internal/causalexpv2/generator.go", false, false},
		{"non-allowlisted path", "README.md", false, false},
		{"accepted amendment blob", "docs/active-causal-diagnosis-v3-amendment.md", false, false},
		{"mode change", "cmd/nous/main.go", true, false},
		{"missing allowlisted modification", "cmd/nous/main.go", false, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			clone := cloneAtCommitForTest(t, root, e3)
			path := filepath.Join(clone, filepath.FromSlash(test.path))
			if test.mode {
				if err := os.Chmod(path, 0o755); err != nil {
					t.Fatal(err)
				}
			} else if test.revert {
				content, err := gitFile(context.Background(), clone, BaselineCommit, test.path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, content, 0o600); err != nil {
					t.Fatal(err)
				}
			} else {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(content, []byte("\n// injected confinement violation\n")...), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			gitCommitAll(t, clone, "inject confinement violation")
			bad, err := resolveGitState(context.Background(), clone)
			if err != nil {
				t.Fatal(err)
			}
			protectedGeneratorCalls.Store(0)
			if err := verifyV3ExecutableConfinement(context.Background(), bad); err == nil {
				t.Fatal("injected confinement violation was accepted")
			}
			if got := protectedGeneratorCalls.Load(); got != 0 {
				t.Fatalf("confinement violation opened %d protected fixtures", got)
			}
		})
	}
}

func gitCommitAll(t *testing.T, repository, message string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-m", message}} {
		command := exec.Command("git", append([]string{"-C", repository}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
}

func evaluationReportForTest(t *testing.T, panel Panel, seedCount int, commit string) []byte {
	t.Helper()
	tasks := make([]TaskMeterItem, 0, 960)
	for _, name := range []string{"certificate-replay", "post-selection-replay"} {
		for i := 0; i < 480; i++ {
			tasks = append(tasks, TaskMeterItem{Name: name, Subject: fmt.Sprintf("certificate-%03d", i)})
		}
	}
	taskDigest, err := causalv2.TaskMeterItemsDigest(tasks)
	if err != nil {
		t.Fatal(err)
	}
	controls := executedControlBundle(t)
	report := EvaluationReport{ReportVersion: "causal-diagnosis-report/v2", Manifest: causalv2.PreregisteredManifest(), PlanCommit: PlanCommit, PretrainingCommit: commit, ImplementationCommit: commit, Panel: string(panel), Status: "valid", ControlBundle: controls, ControlBundleDigest: controls.ControlBundleDigest, TaskMeterItems: tasks, TaskMeterItemsDigest: taskDigest, Contrasts: []Contrast{}, Limitations: []string{}}
	seedRange := report.Manifest.DevelopmentSeeds
	if panel == PanelValidation {
		seedRange = report.Manifest.ValidationSeeds
	} else if panel == PanelLocked {
		seedRange = report.Manifest.LockedSeeds
	}
	var episodeItems [][]MeterItem
	for _, policyName := range evaluationPolicies {
		policy := PolicyReport{Name: policyName, Fixtures: []EvaluationFixture{}, Cohorts: []Aggregate{}}
		for i := 0; i < seedCount; i++ {
			items := make([]MeterItem, len(causalv2.MeterNames))
			for meterIndex, name := range causalv2.MeterNames {
				items[meterIndex] = MeterItem{Name: name, Active: name == "production" || name == "teacher" || name == "oracle-audit" || name == "dp" && policyName == "dynamic-optimal"}
			}
			policy.Fixtures = append(policy.Fixtures, EvaluationFixture{Seed: seedRange.Start + int64(i)*seedRange.Step, Cohort: cohortFor(i), Actions: []string{}, MeterItems: items, AllCapsValid: true})
			episodeItems = append(episodeItems, items)
		}
		report.Policies = append(report.Policies, policy)
	}
	controlCounts := make([][15]int64, len(controls.Certificates))
	for i, certificate := range controls.Certificates {
		controlCounts[i] = certificate.MeterCounts
	}
	meters, _, err := ReconstructMeters(MeterEvaluation, episodeItems, tasks, controlCounts)
	if err != nil {
		t.Fatal(err)
	}
	report.Mechanical = EvaluationMechanical{AllValid: true, DependencyBoundary: true, ProfileValid: true, TranscriptValid: true, TrainingFreezeValid: true, Meters: meters, AllCapsValid: true}
	reconstructEvaluationDerivations(&report)
	encoded, err := FinalizeEvaluationReport(&report)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestEvaluationEvidenceRejectsWrongPreregisteredSeed(t *testing.T) {
	report := EvaluationReport{Manifest: causalv2.PreregisteredManifest(), Panel: "validation", Policies: make([]PolicyReport, len(evaluationPolicies))}
	for index, policy := range evaluationPolicies {
		report.Policies[index] = PolicyReport{Name: policy, Fixtures: make([]EvaluationFixture, report.Manifest.ValidationSeeds.Count)}
	}
	report.Policies[0].Fixtures[0].Seed = 1
	if err := VerifyEvaluationEvidence(report); err == nil {
		t.Fatal("evaluation verifier accepted caller-authored wrong seed")
	}
}

func TestExclusiveResultWriteRejectsCollision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	if err := writeExclusiveSynced(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusiveSynced(path, []byte("second")); err == nil {
		t.Fatal("exclusive result write accepted a collision")
	}
}
