package causalexpv2

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
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
	if os.Getenv("NOUS_TEST_REPLAY_DIRTY") == "1" {
		if err := os.WriteFile("replay-worker-mutated", []byte("dirty\n"), 0o600); err != nil {
			os.Exit(22)
		}
	}
	if os.Getenv("NOUS_TEST_REPLAY_IGNORED") == "1" {
		if err := os.WriteFile("go.work", []byte("go 1.25.8\n"), 0o600); err != nil {
			os.Exit(21)
		}
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

func TestReplayAuditsWorktreeAfterWorkerFailure(t *testing.T) {
	repository, common, head := gitTestRepo(t)
	if err := os.WriteFile(filepath.Join(common, "info", "exclude"), []byte("go.work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NOUS_TEST_REPLAY_HELPER", "1")
	t.Setenv("NOUS_TEST_REPLAY_FAIL", "1")
	t.Setenv("NOUS_TEST_REPLAY_DIRTY", "1")
	_, err := replayCapabilityForTest(t, repository, head, []byte("report"), []byte("bundle")).Replay(context.Background())
	if err == nil || !strings.Contains(err.Error(), "regeneration executable failed") || !strings.Contains(err.Error(), "changed clean detached execution worktree") {
		t.Fatalf("worker failure did not retain post-exit audit error: %v", err)
	}
	assertOneWorktree(t, repository)
	t.Setenv("NOUS_TEST_REPLAY_DIRTY", "0")
	t.Setenv("NOUS_TEST_REPLAY_IGNORED", "1")
	_, err = replayCapabilityForTest(t, repository, head, []byte("report"), []byte("bundle")).Replay(context.Background())
	if err == nil || !strings.Contains(err.Error(), "untracked or ignored execution-worktree path") {
		t.Fatalf("worker failure did not audit ignored paths: %v", err)
	}
	assertOneWorktree(t, repository)
}

func TestReplaySuccessRecordBindsEToRToCandidate(t *testing.T) {
	commonDirectory := t.TempDir()
	record := replaySuccessRecord{ReplayVersion: replayRecordVersion, PlanCommit: ReplayRepairPlanCommit, PretrainingCommit: replayPretrainingCommit, EvidenceCommit: replayEvidenceCommit, CandidateCommit: strings.Repeat("c", 40), CandidateDiffDigest: strings.Repeat("d", 64), TrainingDigest: strings.Repeat("e", 64), BundleDigest: strings.Repeat("f", 64), CreatedUTC: time.Now().UTC().Format(time.RFC3339), State: "succeeded"}
	if err := createReplayRecord(commonDirectory, record); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(replayRecordPath(commonDirectory)) != "active-causal-diagnosis-v4-replay.json" {
		t.Fatal("v4 replay receipt used a non-v4 path")
	}
	encoded, err := os.ReadFile(replayRecordPath(commonDirectory))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := causalv2.StrictDecode[replaySuccessRecord](encoded)
	if err != nil || decoded.ReplayVersion != "causal-replay-success/v4" || decoded.PlanCommit != ReplayRepairPlanCommit {
		t.Fatalf("decode v4 receipt: %+v, %v", decoded, err)
	}
	legacy := record
	legacy.ReplayVersion = "causal-replay-success/v3"
	if legacy.ReplayVersion == replayRecordVersion {
		t.Fatal("v4 replay decoder identity aliases v3")
	}
	if filepath.Base(replayRecordPath(commonDirectory)) == failedV3ReplayRecordName {
		t.Fatal("v4 replay path aliases the consumed v3 receipt")
	}
}

func TestV4PinnedToolsMetadataAndEnvironment(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	state, err := resolveGitState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPinnedGitTool(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := verifyPinnedGitRepositoryState(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if err := verifyRegularFileDigest(filepath.Join(root, "go.mod"), resolvedGoModSHA256); err != nil {
		t.Fatal(err)
	}
	if err := verifyRegularFileDigest(filepath.Join(root, "go.sum"), resolvedGoSumSHA256); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GOROOT=" + pinnedGOROOT, "GOTOOLCHAIN=local", "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOEXPERIMENT=", "CGO_ENABLED=0", "GOOS=darwin", "GOARCH=arm64", "GOARM64=v8.0", "GODEBUG=", "GOFIPS140=off",
		"GOMODCACHE=/private/mod", "GOPATH=/private/gopath", "GOCACHE=/private/build", "TMPDIR=/private/tmp", "GOPROXY=https://proxy.golang.org", "GOSUMDB=sum.golang.org", "GOPRIVATE=", "GONOPROXY=", "GONOSUMDB=",
	}
	if got := fixedGoEnvironment("/private/mod", "/private/build", "/private/tmp"); !slices.Equal(got, want) {
		t.Fatalf("fixed Go environment = %q", got)
	}
	for _, hostile := range []string{"HOME=hostile", "PATH=hostile", "GOFLAGS=-mod=vendor", "GIT_CONFIG_COUNT=1"} {
		if slices.Contains(want, hostile) {
			t.Fatalf("fixed environment inherited %q", hostile)
		}
	}
	wrong := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(wrong, []byte("wrong tool\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := verifyRegularFileDigest(wrong, pinnedGoSHA256); err == nil {
		t.Fatal("tool verifier accepted wrong executable bytes")
	}
	link := filepath.Join(t.TempDir(), "tool-link")
	if err := os.Symlink(wrong, link); err != nil {
		t.Fatal(err)
	}
	if err := verifyRegularFileDigest(link, sha256Hex([]byte("wrong tool\n"))); err == nil {
		t.Fatal("tool verifier accepted a symlink")
	}
}

func setProtectedGitEnvironmentForTest(t *testing.T) {
	t.Helper()
	required := map[string]string{"PATH": "/opt/homebrew/bin", "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_SYSTEM": "/dev/null", "GIT_OPTIONAL_LOCKS": "0", "GIT_NO_REPLACE_OBJECTS": "1", "GIT_ATTR_NOSYSTEM": "1", "GIT_TERMINAL_PROMPT": "0"}
	for name, value := range required {
		t.Setenv(name, value)
	}
	for _, name := range []string{"GIT_ASKPASS", "GIT_SSH_COMMAND", "GIT_CONFIG_COUNT", "GIT_CONFIG_PARAMETERS", "GIT_EXTERNAL_DIFF", "GIT_DIFF_OPTS", "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_NAMESPACE", "GIT_SHALLOW_FILE", "GIT_EXEC_PATH"} {
		value, present := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if present {
				_ = os.Setenv(name, value)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}
}

func TestV4ProtectedGitEnvironmentRejectsHostileInputs(t *testing.T) {
	setProtectedGitEnvironmentForTest(t)
	if err := verifyProtectedGitEnvironment(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"GIT_EXTERNAL_DIFF", "GIT_CONFIG_COUNT", "GIT_CONFIG_PARAMETERS", "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES"} {
		t.Run(name, func(t *testing.T) {
			if err := os.Setenv(name, "hostile"); err != nil {
				t.Fatal(err)
			}
			if err := verifyProtectedGitEnvironment(); err == nil {
				t.Fatalf("protected Git environment accepted %s", name)
			}
			if err := os.Unsetenv(name); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestV4CandidateDigestUsesPinnedGitAndV4Domain(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	setProtectedGitEnvironmentForTest(t)
	command := exec.CommandContext(context.Background(), pinnedGitPath, "-C", root, "diff", "--binary", replayEvidenceCommit, "--")
	command.Env = protectedGitCommandEnvironment()
	diff, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	payload := struct {
		EvidenceCommit string `json:"evidence_commit"`
		Diff           []byte `json:"diff"`
	}{replayEvidenceCommit, diff}
	want, err := causalv2.Digest("causal-replay-candidate-diff/v4", payload)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := causalv2.Digest("causal-replay-candidate-diff/v3", payload)
	if err != nil {
		t.Fatal(err)
	}
	got, err := candidateDiffDigest(context.Background(), root, replayEvidenceCommit)
	if err != nil || got != want || got == legacy {
		t.Fatalf("candidate digest = %s, want v4 %s, legacy %s, err=%v", got, want, legacy, err)
	}
}

func TestV4ProtectedRuntimeEnvironmentRejectsHostileInputs(t *testing.T) {
	t.Setenv("GODEBUG", "")
	t.Setenv("GOFIPS140", "off")
	if err := verifyProtectedRuntimeEnvironment(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GODEBUG", "gctrace=1")
	if err := verifyProtectedRuntimeEnvironment(); err == nil {
		t.Fatal("protected runtime accepted hostile GODEBUG")
	}
	t.Setenv("GODEBUG", "")
	t.Setenv("GOFIPS140", "on")
	if err := verifyProtectedRuntimeEnvironment(); err == nil {
		t.Fatal("protected runtime accepted hostile GOFIPS140")
	}
}

func TestV4PinnedGOROOTManifest(t *testing.T) {
	count, digest, err := gorootManifest(pinnedGOROOT)
	if err != nil {
		t.Fatal(err)
	}
	if count != pinnedGOROOTFiles || digest != pinnedGOROOTSHA256 {
		t.Fatalf("GOROOT manifest = (%d, %s)", count, digest)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "compiler"), []byte("bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	wrongCount, wrongDigest, err := gorootManifest(root)
	if err != nil || wrongCount == pinnedGOROOTFiles || wrongDigest == pinnedGOROOTSHA256 {
		t.Fatal("GOROOT manifest accepted a different tree")
	}
	if err := os.Symlink(filepath.Join(root, "compiler"), filepath.Join(root, "symlink")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := gorootManifest(root); err == nil {
		t.Fatal("GOROOT manifest accepted a symlink")
	}
}

func TestV4PinnedGitRepositoryStateRejectsLocalInfluence(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := resolveGitState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	config, err := os.ReadFile(filepath.Join(actual.CommonDir, "config"))
	if err != nil {
		t.Fatal(err)
	}
	exclude, err := os.ReadFile(filepath.Join(actual.CommonDir, "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	makeState := func(t *testing.T) gitState {
		t.Helper()
		common := t.TempDir()
		if err := os.MkdirAll(filepath.Join(common, "info"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(common, "config"), config, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(common, "info", "exclude"), exclude, 0o600); err != nil {
			t.Fatal(err)
		}
		return gitState{CommonDir: common}
	}
	for _, test := range []struct {
		name string
		path string
		data []byte
	}{
		{"include", "config", append(append([]byte(nil), config...), []byte("\n[include]\npath = /tmp/hostile\n")...)},
		{"exclude", "info/exclude", append(append([]byte(nil), exclude...), []byte("\ninternal/\n")...)},
		{"attributes", "info/attributes", []byte("* diff=hostile\n")},
		{"graft", "info/grafts", []byte(strings.Repeat("a", 40) + " " + strings.Repeat("b", 40) + "\n")},
		{"shallow", "shallow", []byte(strings.Repeat("a", 40) + "\n")},
		{"alternate", "objects/info/alternates", []byte("/tmp/objects\n")},
		{"worktree config", "config.worktree", []byte("[core]\nworktree=/tmp\n")},
		{"replacement ref", "refs/replace/" + strings.Repeat("a", 40), []byte(strings.Repeat("b", 40) + "\n")},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := makeState(t)
			path := filepath.Join(state.CommonDir, filepath.FromSlash(test.path))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, test.data, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := verifyPinnedGitRepositoryState(context.Background(), state); err == nil {
				t.Fatal("hostile repository-local Git state was accepted")
			}
		})
	}
}

func TestV4WorkerEnvironmentIsExactAndPrivate(t *testing.T) {
	environment, err := pinnedWorkerEnvironment(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(environment) != 11 {
		t.Fatalf("worker environment has %d entries", len(environment))
	}
	for _, denied := range []string{"GIT_CONFIG_COUNT", "GIT_CONFIG_PARAMETERS", "GIT_EXTERNAL_DIFF", "GIT_DIFF_OPTS"} {
		for _, entry := range environment {
			if strings.HasPrefix(entry, denied+"=") {
				t.Fatalf("worker environment inherited %s", denied)
			}
		}
	}
	bin := strings.TrimPrefix(environment[0], "PATH=")
	entries, err := os.ReadDir(bin)
	if err != nil || len(entries) != 1 || entries[0].Name() != "git" {
		t.Fatalf("worker private bin = %v, %v", entries, err)
	}
	if err := os.Remove(filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/bin/git", filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	if err := verifyPinnedWorkerEnvironment(environment); err == nil {
		t.Fatal("worker environment accepted a substituted private Git link")
	}
}

func TestV4RetryCollisionsPrecedeAllActivityAndIgnoreV3Failure(t *testing.T) {
	common := t.TempDir()
	state := gitState{CommonDir: common}
	failedV3 := filepath.Join(common, "nous-attempts", failedV3ReplayRecordName)
	if err := os.MkdirAll(filepath.Dir(failedV3), 0o700); err != nil {
		t.Fatal(err)
	}
	sentinel := []byte("failed-v3-must-remain-opaque")
	if err := os.WriteFile(failedV3, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := requireReplayRetrySlotsAbsent(state); err != nil {
		t.Fatalf("v4 treated failed v3 receipt as a collision: %v", err)
	}
	paths := []string{replayRecordPath(common), attemptRecordPath(common, PanelValidation), attemptProofRecordPath(common, PanelValidation), attemptRecordPath(common, PanelLocked), attemptProofRecordPath(common, PanelLocked), resultPath(common, PanelValidation), resultPath(common, PanelLocked)}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("occupied"), 0o600); err != nil {
				t.Fatal(err)
			}
			replayBuildPreflights.Store(0)
			replayEvidenceReads.Store(0)
			replayCapabilityMints.Store(0)
			replayInputConstructions.Store(0)
			replayWorkerStarts.Store(0)
			if err := requireReplayRetrySlotsAbsent(state); err == nil {
				t.Fatal("occupied retry slot was accepted")
			}
			if replayBuildPreflights.Load()+replayEvidenceReads.Load()+replayCapabilityMints.Load()+replayInputConstructions.Load()+replayWorkerStarts.Load() != 0 {
				t.Fatal("retry collision permitted protected activity")
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
	got, err := os.ReadFile(failedV3)
	if err != nil || !bytes.Equal(got, sentinel) {
		t.Fatal("v4 read or changed the failed v3 receipt")
	}
}

func TestV4R3AnchoredFunctionBodyFloors(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		"gates.go:mintReplayCapability": true, "gates.go:verifyCandidateConstantsState": true, "gates.go:Replay": true, "gates.go:buildReplayWorker": true, "gates.go:beginValidationAttempt": true,
		"provenance.go:ExecuteProtectedPanel": true,
		"replay_gate.go:replayRecordPath":     true, "replay_gate.go:candidateDiffDigest": true, "replay_gate.go:ExecuteReplay": true, "replay_gate.go:createReplayRecord": true, "replay_gate.go:persistReplayRecord": true, "replay_gate.go:verifyReplaySuccessRecord": true,
		"provenance_test.go:TestReplayHelperProcess": true, "provenance_test.go:TestReplaySuccessRecordBindsEToRToCandidate": true,
	}
	allowedNew := map[string]bool{
		"gates.go:verifyReplayRepairExecutableCommit": true, "gates.go:pinnedReplayBuilder": true, "gates.go:verifyPinnedProtectedRuntime": true, "gates.go:verifyProtectedRuntimeEnvironment": true, "gates.go:gorootManifest": true, "gates.go:fixedGoEnvironment": true, "gates.go:verifyRegularFileDigest": true, "gates.go:sha256Hex": true, "gates.go:verifyPinnedGitTool": true, "gates.go:verifyProtectedGitEnvironment": true, "gates.go:protectedGitCommandEnvironment": true, "gates.go:verifyPinnedGitRepositoryState": true, "gates.go:pinnedWorkerEnvironment": true, "gates.go:verifyPinnedWorkerEnvironment": true, "gates.go:preflightReplayBuild": true, "gates.go:verifyResolvedReplayWorktree": true, "gates.go:verifyCleanReplayWorktree": true, "gates.go:cleanupReplayWorktreeSet": true, "gates.go:makeTreeOwnerWritable": true, "gates.go:cleanupResolvedReplayWorktree": true,
		"replay_gate.go:requireReplayRetrySlotsAbsent": true, "replay_gate.go:structuralReplayRecord": true,
		"provenance_test.go:TestReplayAuditsWorktreeAfterWorkerFailure": true, "provenance_test.go:TestV4PinnedToolsMetadataAndEnvironment": true, "provenance_test.go:setProtectedGitEnvironmentForTest": true, "provenance_test.go:TestV4ProtectedGitEnvironmentRejectsHostileInputs": true, "provenance_test.go:TestV4CandidateDigestUsesPinnedGitAndV4Domain": true, "provenance_test.go:TestV4ProtectedRuntimeEnvironmentRejectsHostileInputs": true, "provenance_test.go:TestV4PinnedGOROOTManifest": true, "provenance_test.go:TestV4PinnedGitRepositoryStateRejectsLocalInfluence": true, "provenance_test.go:TestV4WorkerEnvironmentIsExactAndPrivate": true, "provenance_test.go:TestV4RetryCollisionsPrecedeAllActivityAndIgnoreV3Failure": true, "provenance_test.go:TestV4R3AnchoredFunctionBodyFloors": true, "provenance_test.go:functionFloorsForTest": true, "provenance_test.go:topLevelDeclarationsForTest": true, "provenance_test.go:locateV4ExecutableCommitForTest": true, "provenance_test.go:TestV4DetachedE3BuildRegressionAndBuildOnlyPreflight": true, "provenance_test.go:TestV4ExecutableConfinementAndCandidateTopology": true, "provenance_test.go:TestV4ReplaySuccessRecordBindsR3X4C4AndDigests": true, "provenance_test.go:reconstructX4ForTest": true,
	}
	allowedChangedDeclaration := map[string]bool{"gates.go:type:regenerationExecutable": true, "replay_gate.go:const:replayRecordVersion": true}
	allowedNewDeclaration := map[string]bool{}
	for _, key := range []string{
		"gates.go:import:\"crypto/sha256\"", "gates.go:import:\"encoding/hex\"", "gates.go:import:\"io/fs\"", "gates.go:import:\"runtime\"", "gates.go:import:\"runtime/debug\"", "gates.go:import:\"sort\"", "gates.go:import:\"strconv\"", "gates.go:import:\"sync/atomic\"",
		"gates.go:const:replayEvidenceCommit", "gates.go:const:replayPretrainingCommit", "gates.go:const:pinnedGoPath", "gates.go:const:pinnedGoSHA256", "gates.go:const:pinnedGoVersion", "gates.go:const:pinnedGOROOT", "gates.go:const:pinnedGOROOTFiles", "gates.go:const:pinnedGOROOTSHA256", "gates.go:const:pinnedGitPath", "gates.go:const:pinnedGitSHA256", "gates.go:const:pinnedGitVersion", "gates.go:const:pinnedGitConfigSHA256", "gates.go:const:pinnedGitInfoExcludeSHA256", "gates.go:const:resolvedGoModSHA256", "gates.go:const:resolvedGoSumSHA256",
		"gates.go:var:replayBuildPreflights", "gates.go:var:replayEvidenceReads", "gates.go:var:replayCapabilityMints", "gates.go:var:replayInputConstructions", "gates.go:var:replayWorkerStarts",
		"provenance_test.go:import:\"go/ast\"", "provenance_test.go:import:\"go/format\"", "provenance_test.go:import:\"go/parser\"", "provenance_test.go:import:\"go/token\"", "provenance_test.go:type:functionFloorForTest",
		"replay_gate.go:const:ReplayRepairPlanCommit", "replay_gate.go:const:failedV3ReplayRecordName",
	} {
		allowedNewDeclaration[key] = true
	}
	for _, name := range []string{"gates.go", "provenance.go", "provenance_test.go", "replay_gate.go"} {
		path := "internal/causalexpv2/" + name
		baseline, err := gitFile(context.Background(), root, replayEvidenceCommit, path)
		if err != nil {
			t.Fatal(err)
		}
		current, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		before := functionFloorsForTest(t, path+"@R3", baseline)
		after := functionFloorsForTest(t, path+"@X4", current)
		beforeDeclarations := topLevelDeclarationsForTest(t, path+"@R3", baseline)
		afterDeclarations := topLevelDeclarationsForTest(t, path+"@X4", current)
		for key, declaration := range beforeDeclarations {
			currentDeclaration, present := afterDeclarations[key]
			if !present {
				t.Fatalf("R3 top-level declaration was deleted: %s:%s", name, key)
			}
			if !bytes.Equal(declaration, currentDeclaration) && !allowedChangedDeclaration[name+":"+key] {
				t.Fatalf("unlisted top-level declaration changed: %s:%s", name, key)
			}
		}
		for key := range afterDeclarations {
			if _, existed := beforeDeclarations[key]; !existed && !allowedNewDeclaration[name+":"+key] {
				t.Fatalf("unlisted top-level declaration added: %s:%s", name, key)
			}
		}
		for function, floor := range before {
			currentFloor, present := after[function]
			if !present {
				t.Fatalf("R3 function was deleted: %s:%s", name, function)
			}
			if !bytes.Equal(floor.Signature, currentFloor.Signature) {
				t.Fatalf("R3 function signature changed: %s:%s", name, function)
			}
			if allowed[name+":"+function] {
				if name == "provenance_test.go" {
					continue
				}
				if !slices.Equal(floor.EmpiricalStatements, currentFloor.EmpiricalStatements) {
					t.Fatalf("allowed function changed empirical statements or control flow: %s:%s", name, function)
				}
				continue
			}
			if !bytes.Equal(floor.Body, currentFloor.Body) {
				t.Fatalf("unlisted R3 function body changed: %s:%s", name, function)
			}
		}
		for function, floor := range after {
			if _, existed := before[function]; existed {
				continue
			}
			if !allowedNew[name+":"+function] {
				t.Fatalf("unlisted function added at X4: %s:%s", name, function)
			}
			if name != "provenance_test.go" && len(floor.EmpiricalStatements) != 0 {
				t.Fatalf("new helper reads empirical fields: %s:%s: %q", name, function, floor.EmpiricalStatements)
			}
		}
	}
	replayGate, err := os.ReadFile(filepath.Join(root, "internal/causalexpv2/replay_gate.go"))
	if err != nil {
		t.Fatal(err)
	}
	structuralBody := functionFloorsForTest(t, "replay_gate.go", replayGate)["structuralReplayRecord"].Body
	for _, forbidden := range []string{"VerifyTrainingReportBytes", "VerifyTrainingBundleBytes", "contextuallyVerifyTrainingEvidence", "requireUsableTrainingReport", "regenerateTrainingEvidence", "SelectedRule", ".Fixtures", ".Episodes", ".Actions", ".Score", ".Status"} {
		if bytes.Contains(structuralBody, []byte(forbidden)) {
			t.Fatalf("pre-receipt structural helper performs empirical work through %s", forbidden)
		}
	}
}

type functionFloorForTest struct {
	Signature           []byte
	Body                []byte
	EmpiricalStatements []string
}

func functionFloorsForTest(t *testing.T, filename string, source []byte) map[string]functionFloorForTest {
	t.Helper()
	set := token.NewFileSet()
	parsed, err := parser.ParseFile(set, filename, source, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	floors := map[string]functionFloorForTest{}
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		var signature, body bytes.Buffer
		if err := format.Node(&signature, set, function.Type); err != nil {
			t.Fatal(err)
		}
		if err := format.Node(&body, set, function.Body); err != nil {
			t.Fatal(err)
		}
		statements := []string{}
		forbidden := map[string]bool{"SelectedRule": true, "Rules": true, "WinnerTies": true, "Applications": true, "Fixtures": true, "Episodes": true, "Actions": true, "TeacherOutcomes": true, "Terminal": true, "Status": true, "RuleCode": true, "Score": true, "Cost": true, "FinalPosterior": true, "MeterItems": true, "Mechanical": true, "Controls": true, "Contrasts": true, "Cohorts": true, "Aggregates": true, "ControlBundle": true, "ControlEvidence": true, "CorruptionFixture": true, "OracleAgreements": true, "OracleDisagreements": true, "Gates": true, "DynamicBenchmark": true, "AllCapsValid": true, "Passed": true, "Valid": true}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			statement, ok := node.(ast.Stmt)
			if !ok {
				return true
			}
			if _, block := statement.(*ast.BlockStmt); block {
				return true
			}
			hasEmpirical := false
			ast.Inspect(statement, func(child ast.Node) bool {
				selector, ok := child.(*ast.SelectorExpr)
				if ok && forbidden[selector.Sel.Name] {
					hasEmpirical = true
				}
				return true
			})
			if hasEmpirical {
				var encoded bytes.Buffer
				if err := format.Node(&encoded, set, statement); err != nil {
					t.Fatal(err)
				}
				statements = append(statements, encoded.String())
			}
			return true
		})
		slices.Sort(statements)
		floors[function.Name.Name] = functionFloorForTest{Signature: signature.Bytes(), Body: body.Bytes(), EmpiricalStatements: statements}
	}
	return floors
}

func topLevelDeclarationsForTest(t *testing.T, filename string, source []byte) map[string][]byte {
	t.Helper()
	set := token.NewFileSet()
	parsed, err := parser.ParseFile(set, filename, source, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	declarations := map[string][]byte{}
	for _, declaration := range parsed.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			key := ""
			switch value := specification.(type) {
			case *ast.ImportSpec:
				key = "import:" + value.Path.Value
			case *ast.TypeSpec:
				key = "type:" + value.Name.Name
			case *ast.ValueSpec:
				names := make([]string, len(value.Names))
				for index := range value.Names {
					names[index] = value.Names[index].Name
				}
				key = strings.ToLower(general.Tok.String()) + ":" + strings.Join(names, ",")
			}
			if key == "" {
				continue
			}
			var encoded bytes.Buffer
			if err := format.Node(&encoded, set, specification); err != nil {
				t.Fatal(err)
			}
			declarations[key] = encoded.Bytes()
		}
	}
	return declarations
}

func locateV4ExecutableCommitForTest(t *testing.T, repository string) string {
	t.Helper()
	command := exec.Command("git", "-C", repository, "rev-list", "--reverse", replayEvidenceCommit+"..HEAD")
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range strings.Fields(string(output)) {
		if verifyReplayRepairExecutableCommit(context.Background(), repository, candidate) == nil {
			return candidate
		}
	}
	state, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	if state.Head == ReplayRepairPlanCommit {
		t.Skip("v4 replay-repair executable commit has not been created yet")
	}
	t.Fatal("history after the accepted v4 plan has no conforming X4 commit")
	return ""
}

func TestV4DetachedE3BuildRegressionAndBuildOnlyPreflight(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	x4 := locateV4ExecutableCommitForTest(t, root)
	state, err := resolveGitState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if state.Head != x4 {
		t.Skip("build-only preflight is exercised at clean X4 before the v4 receipt")
	}
	if !state.Clean {
		t.Skip("build-only preflight waits for the amended clean X4 commit")
	}
	setProtectedGitEnvironmentForTest(t)
	builder, err := pinnedReplayBuilder(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	base := t.TempDir()
	worktree := filepath.Join(base, "original-e3")
	if err := runGit(context.Background(), root, "worktree", "add", "--detach", worktree, replayPretrainingCommit); err != nil {
		t.Fatal(err)
	}
	worktrees := []string{worktree}
	t.Cleanup(func() { _ = cleanupReplayWorktreeSet(root, worktrees, base) })
	gitWorktree := filepath.Join(base, "git-e3")
	if err := runGit(context.Background(), root, "worktree", "add", "--detach", gitWorktree, replayPretrainingCommit); err != nil {
		t.Fatal(err)
	}
	worktrees = append(worktrees, gitWorktree)
	workerEnvironment, err := pinnedWorkerEnvironment(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range [][]string{{"rev-parse", "HEAD"}, {"rev-parse", "--show-toplevel"}, {"rev-parse", "--git-common-dir"}, {"status", "--porcelain"}, {"merge-base", "--is-ancestor", replayPretrainingCommit, replayEvidenceCommit}, {"diff-tree", "--no-commit-id", "--name-status", "-r", replayEvidenceCommit}} {
		gitCommand := exec.CommandContext(context.Background(), "git", operation...)
		gitCommand.Dir = gitWorktree
		gitCommand.Env = workerEnvironment
		gitOutput, err := gitCommand.Output()
		if err != nil {
			t.Fatalf("fixed worker Git environment cannot execute %v: %s, %v", operation, gitOutput, err)
		}
		if slices.Equal(operation, []string{"rev-parse", "HEAD"}) && strings.TrimSpace(string(gitOutput)) != replayPretrainingCommit {
			t.Fatalf("fixed worker Git environment resolved wrong E3: %s", gitOutput)
		}
	}
	for _, evidencePath := range []string{TrainingEvidenceDirectory + "/" + TrainingEpisodesName, TrainingEvidenceDirectory + "/" + TrainingReportName} {
		gitCommand := exec.CommandContext(context.Background(), "git", "cat-file", "-e", replayPretrainingCommit+":"+evidencePath)
		gitCommand.Dir = gitWorktree
		gitCommand.Env = workerEnvironment
		if err := gitCommand.Run(); err == nil {
			t.Fatalf("E3 unexpectedly contains canonical evidence path %s", evidencePath)
		}
	}
	detachedCommand := exec.CommandContext(context.Background(), "git", "symbolic-ref", "-q", "HEAD")
	detachedCommand.Dir = gitWorktree
	detachedCommand.Env = workerEnvironment
	if err := detachedCommand.Run(); err == nil {
		t.Fatal("fixed worker Git environment did not observe detached E3")
	}
	moduleCache, buildCache, temporaryDirectory := filepath.Join(base, "original-mod"), filepath.Join(base, "original-cache"), filepath.Join(base, "original-tmp")
	for _, directory := range []string{moduleCache, buildCache, temporaryDirectory} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	worker := filepath.Join(base, "original-worker")
	command := exec.CommandContext(context.Background(), builder.Path, "build", "-o", worker, "./internal/causalexpv2/replayexec")
	command.Dir = worktree
	command.Env = fixedGoEnvironment(moduleCache, buildCache, temporaryDirectory)
	if output, err := command.CombinedOutput(); err == nil {
		t.Fatalf("original detached E3 build unexpectedly succeeded without -mod=mod: %s", output)
	}
	if _, err := os.Lstat(worker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed original E3 build produced a worker")
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		encoded, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(worktree, name), encoded, 0o644); err != nil {
			t.Fatalf("prepare resolved E3 %s: %v", name, err)
		}
	}
	if err := verifyResolvedReplayWorktree(context.Background(), root, worktree, replayPretrainingCommit); err != nil {
		t.Fatalf("exact resolved E3 state rejected: %v", err)
	}
	if err := os.Chmod(filepath.Join(worktree, "go.mod"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyResolvedReplayWorktree(context.Background(), root, worktree, replayPretrainingCommit); err == nil {
		t.Fatal("resolved E3 audit accepted wrong module-file mode")
	}
	resolvedWorktree := func(t *testing.T) string {
		t.Helper()
		candidate := filepath.Join(base, fmt.Sprintf("resolved-e3-%d", len(worktrees)))
		if err := runGit(context.Background(), root, "worktree", "add", "--detach", candidate, replayPretrainingCommit); err != nil {
			t.Fatal(err)
		}
		worktrees = append(worktrees, candidate)
		for _, name := range []string{"go.mod", "go.sum"} {
			encoded, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(candidate, name), encoded, 0o644); err != nil {
				t.Fatalf("prepare %s: %v", name, err)
			}
		}
		return candidate
	}
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{"source", func(t *testing.T, candidate string) {
			path := filepath.Join(candidate, "internal/causalexpv2/generator.go")
			encoded, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(encoded, []byte("\n// changed\n")...), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"untracked", func(t *testing.T, candidate string) {
			if err := os.WriteFile(filepath.Join(candidate, "extra"), []byte("extra\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"ignored", func(t *testing.T, candidate string) {
			if err := os.WriteFile(filepath.Join(candidate, "go.work"), []byte("go 1.25.8\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"wrong hash", func(t *testing.T, candidate string) {
			if err := os.WriteFile(filepath.Join(candidate, "go.mod"), []byte("wrong\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"staged", func(t *testing.T, candidate string) {
			if err := runGit(context.Background(), candidate, "add", "go.mod"); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run("resolved mutation "+test.name, func(t *testing.T) {
			candidate := resolvedWorktree(t)
			test.mutate(t, candidate)
			if err := verifyResolvedReplayWorktree(context.Background(), root, candidate, replayPretrainingCommit); err == nil {
				t.Fatal("resolved E3 audit accepted forbidden mutation")
			}
		})
	}

	failedV3 := filepath.Join(state.CommonDir, "nous-attempts", failedV3ReplayRecordName)
	failedBefore, err := os.ReadFile(failedV3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(replayRecordPath(state.CommonDir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("v4 receipt existed before build-only preflight")
	}
	replayBuildPreflights.Store(0)
	replayEvidenceReads.Store(0)
	replayCapabilityMints.Store(0)
	replayInputConstructions.Store(0)
	replayWorkerStarts.Store(0)
	if err := preflightReplayBuild(context.Background(), root, replayPretrainingCommit, builder); err != nil {
		t.Fatal(err)
	}
	if replayBuildPreflights.Load() != 1 || replayEvidenceReads.Load() != 0 || replayCapabilityMints.Load() != 0 || replayInputConstructions.Load() != 0 || replayWorkerStarts.Load() != 0 {
		t.Fatalf("build-only preflight activity = build:%d evidence:%d mint:%d input:%d worker:%d", replayBuildPreflights.Load(), replayEvidenceReads.Load(), replayCapabilityMints.Load(), replayInputConstructions.Load(), replayWorkerStarts.Load())
	}
	if _, err := os.Lstat(replayRecordPath(state.CommonDir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("build-only preflight created a v4 receipt")
	}
	failedAfter, err := os.ReadFile(failedV3)
	if err != nil || !bytes.Equal(failedBefore, failedAfter) {
		t.Fatal("build-only preflight changed the failed v3 receipt")
	}
}

func TestV4ExecutableConfinementAndCandidateTopology(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	x4 := locateV4ExecutableCommitForTest(t, root)
	repository := cloneAtCommitForTest(t, root, x4)
	state, err := resolveGitState(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyReplayRepairExecutableCommit(context.Background(), repository, state.Head); err != nil {
		t.Fatalf("accepted X4 rejected: %v", err)
	}
	freeze := filepath.Join(repository, FrozenConstantsPath)
	if err := os.WriteFile(freeze, expectedFreezeFile("P=E;M=gain;S=C", replayEvidenceCommit, "96b1cdf7579c0a186e5cd9aeb7aaa42f0c224ffe19989bf78b5b3aa320b17fa0"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirty, err := resolveGitState(context.Background(), repository)
	if err != nil || verifyCandidateConstantsState(context.Background(), dirty, replayEvidenceCommit) != nil {
		t.Fatalf("exact dirty X4 candidate rejected: %v", err)
	}
	if err := os.Chmod(freeze, 0o755); err != nil {
		t.Fatal(err)
	}
	dirtyMode, err := resolveGitState(context.Background(), repository)
	if err != nil || verifyCandidateConstantsState(context.Background(), dirtyMode, replayEvidenceCommit) == nil {
		t.Fatal("dirty X4 accepted executable constants file")
	}
	if err := os.Chmod(freeze, 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, repository, "C4")
	clean, err := resolveGitState(context.Background(), repository)
	if err != nil || verifyCandidateConstantsState(context.Background(), clean, replayEvidenceCommit) != nil {
		t.Fatalf("exact clean C4 candidate rejected: %v", err)
	}
	badC4 := cloneAtCommitForTest(t, root, x4)
	badFreeze := filepath.Join(badC4, FrozenConstantsPath)
	if err := os.WriteFile(badFreeze, expectedFreezeFile("P=E;M=gain;S=C", replayEvidenceCommit, "96b1cdf7579c0a186e5cd9aeb7aaa42f0c224ffe19989bf78b5b3aa320b17fa0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(badFreeze, 0o755); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, badC4, "bad mode C4")
	badC4State, err := resolveGitState(context.Background(), badC4)
	if err != nil || verifyCandidateConstantsState(context.Background(), badC4State, replayEvidenceCommit) == nil {
		t.Fatal("clean C4 accepted executable constants blob")
	}

	for _, test := range []struct {
		name string
		path string
		kind string
	}{
		{"protected causal source", "internal/causalexpv2/generator.go", "edit"},
		{"domain source", "internal/causalv2/domains.go", "edit"},
		{"amendment", "docs/active-causal-diagnosis-v4-replay-amendment.md", "edit"},
		{"module hash", "go.mod", "edit"},
		{"mode", "internal/causalexpv2/gates.go", "mode"},
		{"missing", "go.sum", "missing"},
		{"additional", "README.md", "edit"},
		{"removed", "internal/causalexpv2/gates.go", "remove"},
		{"renamed", "internal/causalexpv2/gates.go", "rename"},
	} {
		t.Run(test.name, func(t *testing.T) {
			clone := reconstructX4ForTest(t, root, x4, test.path == "go.sum" && test.kind == "missing")
			path := filepath.Join(clone, filepath.FromSlash(test.path))
			switch test.kind {
			case "mode":
				if err := os.Chmod(path, 0o755); err != nil {
					t.Fatal(err)
				}
			case "edit":
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(content, []byte("\n// v4 confinement violation\n")...), 0o600); err != nil {
					t.Fatal(err)
				}
			case "remove":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "rename":
				if err := os.Rename(path, path+".renamed"); err != nil {
					t.Fatal(err)
				}
			}
			gitCommitAll(t, clone, "inject v4 confinement violation")
			bad, err := resolveGitState(context.Background(), clone)
			if err != nil {
				t.Fatal(err)
			}
			if err := verifyReplayRepairExecutableCommit(context.Background(), clone, bad.Head); err == nil {
				t.Fatal("injected X4 confinement violation was accepted")
			}
		})
	}
}

func TestV4ReplaySuccessRecordBindsR3X4C4AndDigests(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	_ = locateV4ExecutableCommitForTest(t, root)
	state, err := resolveGitState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if verifyCandidateConstantsState(context.Background(), state, replayEvidenceCommit) != nil {
		t.Skip("v4 receipt binding is exercised at clean C4")
	}
	setProtectedGitEnvironmentForTest(t)
	receiptBytes, err := os.ReadFile(replayRecordPath(state.CommonDir))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := causalv2.StrictDecode[replaySuccessRecord](receiptBytes)
	if err != nil {
		t.Fatal(err)
	}
	reportBytes, err := gitFile(context.Background(), root, replayEvidenceCommit, filepath.ToSlash(filepath.Join(TrainingEvidenceDirectory, TrainingReportName)))
	if err != nil {
		t.Fatal(err)
	}
	bundleBytes, err := gitFile(context.Background(), root, replayEvidenceCommit, filepath.ToSlash(filepath.Join(TrainingEvidenceDirectory, TrainingEpisodesName)))
	if err != nil {
		t.Fatal(err)
	}
	report, err := causalv2.StrictDecode[TrainingReport](reportBytes)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := causalv2.StrictDecode[TrainingBundle](bundleBytes)
	if err != nil {
		t.Fatal(err)
	}
	common := t.TempDir()
	state.CommonDir = common
	write := func(t *testing.T, record replaySuccessRecord) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(replayRecordPath(common)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(replayRecordPath(common), mustCanonical(record), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(t, receipt)
	if err := verifyReplaySuccessRecord(context.Background(), state, report, bundle, replayEvidenceCommit); err != nil {
		t.Fatalf("valid v4 receipt rejected: %v", err)
	}
	for _, mutate := range []func(*replaySuccessRecord){
		func(value *replaySuccessRecord) { value.ReplayVersion = "causal-replay-success/v3" },
		func(value *replaySuccessRecord) { value.PlanCommit = PlanCommit },
		func(value *replaySuccessRecord) { value.PretrainingCommit = strings.Repeat("1", 40) },
		func(value *replaySuccessRecord) { value.EvidenceCommit = strings.Repeat("2", 40) },
		func(value *replaySuccessRecord) { value.CandidateCommit = state.Head },
		func(value *replaySuccessRecord) { value.CandidateDiffDigest = strings.Repeat("3", 64) },
		func(value *replaySuccessRecord) { value.TrainingDigest = strings.Repeat("4", 64) },
		func(value *replaySuccessRecord) { value.BundleDigest = strings.Repeat("5", 64) },
		func(value *replaySuccessRecord) { value.State = "failed" },
	} {
		forged := receipt
		mutate(&forged)
		write(t, forged)
		if err := verifyReplaySuccessRecord(context.Background(), state, report, bundle, replayEvidenceCommit); err == nil {
			t.Fatal("forged v4 replay receipt was accepted")
		}
	}
	write(t, receipt)
	wrongTopology := state
	wrongTopology.Head = receipt.CandidateCommit
	if err := verifyReplaySuccessRecord(context.Background(), wrongTopology, report, bundle, replayEvidenceCommit); err == nil {
		t.Fatal("v4 replay receipt accepted wrong C4 topology")
	}
}

func reconstructX4ForTest(t *testing.T, root, x4 string, omitGoSum bool) string {
	t.Helper()
	repository := cloneAtCommitForTest(t, root, ReplayRepairPlanCommit)
	paths := []string{"go.mod", "go.sum", "internal/causalexpv2/gates.go", "internal/causalexpv2/provenance.go", "internal/causalexpv2/provenance_test.go", "internal/causalexpv2/replay_gate.go"}
	for _, path := range paths {
		if omitGoSum && path == "go.sum" {
			continue
		}
		content, err := gitFile(context.Background(), root, x4, path)
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(repository, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repository
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
