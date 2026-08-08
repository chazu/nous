package causalexpv2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/chazu/nous/internal/causalv2"
	"golang.org/x/sys/unix"
)

const (
	cacheDiagnosticVersion          = "causal-cache-diagnostic/v1"
	cacheDiagnosticPlanCommit       = "4881641ade773a07280990f19e89277c7e12c056"
	cacheDiagnosticProgramCommit    = "e6dd2fef97a80e24ceae5b2bb531d09ae310c0aa"
	cacheDiagnosticV5Commit         = "3bce9836b60c9fc419696f0a82450af3a158ec19"
	cacheDiagnosticV5ReceiptDigest  = "2e3c67fe774b288d72ea8672f9b07a063533a377a1394ba5de082f9c316a6577"
	cacheDiagnosticInputDigest      = "f579b1ac920f57f8062dc0e030e7b708135d711161048d1ea44573d5eff21d7f"
	cacheDiagnosticInputSHA256      = "08ca98db6a5975707124e579bad34acdc9361a74f15e9e08142116b10db71649"
	cacheDiagnosticInputBytes       = 91054
	cacheDiagnosticResultDigest     = "af39dcfb8bc6b40287b2e1702bf15a5f157720ef714defac76f2a02564b1e611"
	cacheDiagnosticHypothesesDigest = "9a69dc0cfb4665bc05a91a400d5469d00253abdc6693e0cc45ca98440ada07df"
	cacheDiagnosticDecisionDigest   = "1fcb46171d7037ace6549bdcb0858dbeae433634514a2114d756c0124d4f7912"
	cacheDiagnosticSourcePath       = "internal/causalexpv2/cache_diagnostic.go"
	cacheDiagnosticPlanPath         = "docs/active-causal-diagnosis-v6-diagnostic-plan.md"
	cacheDiagnosticRecordName       = "active-causal-diagnosis-v6-cache-diagnostic.json"
)

const (
	cacheDiagnosticContractRejection    = "contract-rejection"
	cacheDiagnosticImplementationDefect = "descriptor-implementation-defect"
	cacheDiagnosticNonReproduction      = "non-reproduction"
	cacheDiagnosticUnsafeDisagreement   = "unsafe-auditor-disagreement"
	cacheDiagnosticFailure              = "diagnostic-failure"
)

const (
	cacheDiagnosticResultJSON     = `{"result_version":"causal-cache-diagnostic-result/v1","results":["contract-rejection","descriptor-implementation-defect","non-reproduction","unsafe-auditor-disagreement","diagnostic-failure"]}`
	cacheDiagnosticHypothesesJSON = `{"hypotheses_version":"causal-cache-diagnostic-hypotheses/v1","hypotheses":["diagnostic-tree-contract-rejection","reproducible-descriptor-implementation-defect","non-reproduction","unsafe-auditor-disagreement","diagnostic-failure"]}`
	cacheDiagnosticDecisionJSON   = `{"decision_version":"causal-cache-diagnostic-decision/v1","h0":"terminate-invalid","h1":"authorize-read-dirent-recovery","h2":"terminate-invalid","h3":"terminate-invalid","h4":"terminate-invalid"}`
)

type cacheDiagnosticRecord struct {
	DiagnosticVersion      string `json:"diagnostic_version"`
	PlanCommit             string `json:"plan_commit"`
	ImplementationCommit   string `json:"implementation_commit"`
	ProgramCommit          string `json:"program_commit"`
	V5PlanCommit           string `json:"v5_plan_commit"`
	V5ImplementationCommit string `json:"v5_implementation_commit"`
	PretrainingCommit      string `json:"pretraining_commit"`
	EvidenceCommit         string `json:"evidence_commit"`
	V5ReceiptDigest        string `json:"v5_receipt_digest"`
	InputDigest            string `json:"input_digest"`
	InputSHA256            string `json:"input_sha256"`
	InputBytes             int    `json:"input_bytes"`
	OperatorSHA256         string `json:"operator_sha256"`
	WorkerSHA256           string `json:"worker_sha256"`
	EnvironmentDigest      string `json:"environment_digest"`
	CandidateBodyDigest    string `json:"candidate_body_digest"`
	ResultSchemaDigest     string `json:"result_schema_digest"`
	HypothesesDigest       string `json:"hypotheses_digest"`
	DecisionDigest         string `json:"decision_digest"`
	CreatedUTC             string `json:"created_utc"`
	State                  string `json:"state"`
	WorkerStarts           int    `json:"worker_starts"`
	Result                 string `json:"result"`
}

type diagnosticOuterSnapshot struct {
	base       string
	home       string
	xdg        string
	baseDevice uint64
	baseInode  uint64
	homeDevice uint64
	homeInode  uint64
	xdgDevice  uint64
	xdgInode   uint64
}

type candidateWorkerCacheBudget struct {
	entries      int
	pathBytes    int
	logicalBytes int64
}

type referenceCacheBudget struct {
	contractEntries      int
	contractPathBytes    int
	contractLogicalBytes int64
	hardEntries          int
	hardPathBytes        int
	hardLogicalBytes     int64
}

type referenceMetadataRecord struct {
	root     byte
	path     string
	typeBits uint32
	device   uint64
	inode    uint64
	links    uint64
	size     int64
}

type referenceAuditResult struct {
	accepted bool
	digest   [32]byte
}

type cacheDiagnosticExecutionHooks struct {
	verifyExecutionState func(stage string) error
	afterStartedPersist  func() error
	bracketStep          func(step int)
}

func cacheDiagnosticRecordPath(commonDirectory string) string {
	return filepath.Join(commonDirectory, "nous-attempts", cacheDiagnosticRecordName)
}

func ExecuteCacheDiagnostic(ctx context.Context, repoRoot string) (result string, returnErr error) {
	result = cacheDiagnosticFailure
	if err := orchestrationAvailable(); err != nil {
		return result, err
	}
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return result, err
	}
	if err := verifyCacheDiagnosticInvocationRoot(repoRoot, state.Root); err != nil {
		return result, err
	}
	if err := verifyPinnedProtectedRuntime(ctx, repoRoot); err != nil {
		return result, err
	}
	if err := verifyCacheDiagnosticTopology(ctx, state); err != nil {
		return result, err
	}
	if err := verifyCacheDiagnosticSelfBinding(state.Head); err != nil {
		return result, err
	}
	if err := verifyFailedV5DiagnosticPredecessor(state.CommonDir); err != nil {
		return result, err
	}
	if err := requireDiagnosticSlotsAbsent(state.CommonDir); err != nil {
		return result, err
	}
	inputBytes, inputDigest, err := buildCacheDiagnosticInput()
	if err != nil {
		return result, err
	}
	if len(inputBytes) != cacheDiagnosticInputBytes || sha256Hex(inputBytes) != cacheDiagnosticInputSHA256 || inputDigest != cacheDiagnosticInputDigest {
		return result, errors.New("diagnostic input identity mismatch")
	}
	if err := verifyCacheDiagnosticProtocolDigests(); err != nil {
		return result, err
	}
	base, err := os.MkdirTemp("", "nous-causal-v6-cache-diagnostic-")
	if err != nil {
		return result, err
	}
	worktrees := []string{}
	receiptCreated := false
	cleanupComplete := false
	record := cacheDiagnosticRecord{}
	defer func() {
		if !cleanupComplete {
			if cleanupErr := cleanupReplayWorktreeSet(state.Root, worktrees, base); cleanupErr != nil {
				returnErr = errors.Join(returnErr, cleanupErr)
			}
			cleanupComplete = true
		}
		if returnErr != nil && receiptCreated {
			result = cacheDiagnosticFailure
			returnErr = errors.Join(returnErr, persistCacheDiagnosticTerminal(state.CommonDir, &record, "failed", cacheDiagnosticFailure))
		}
	}()

	buildWorktree := filepath.Join(base, "build-worktree")
	if err := runGit(ctx, state.Root, "worktree", "add", "--detach", buildWorktree, replayPretrainingCommit); err != nil {
		return result, err
	}
	worktrees = append(worktrees, buildWorktree)
	builder, err := pinnedReplayBuilder(ctx, state.Root)
	if err != nil {
		return result, err
	}
	worker, err := buildReplayWorker(ctx, builder, buildWorktree, filepath.Join(base, "causal-v6-diagnostic-worker"))
	if err != nil {
		return result, err
	}
	if err := verifyResolvedReplayWorktree(ctx, state.Root, buildWorktree, replayPretrainingCommit); err != nil {
		return result, err
	}
	executionWorktree := filepath.Join(base, "execution-worktree")
	if err := runGit(ctx, state.Root, "worktree", "add", "--detach", executionWorktree, replayPretrainingCommit); err != nil {
		return result, err
	}
	worktrees = append(worktrees, executionWorktree)
	if err := verifyCleanReplayWorktree(ctx, executionWorktree, replayPretrainingCommit); err != nil {
		return result, err
	}
	output := filepath.Join(base, "output")
	if err := os.Mkdir(output, 0o700); err != nil {
		return result, err
	}
	outer, err := captureDiagnosticOuterEnvelope(worker.Environment, base)
	if err != nil {
		return result, err
	}
	operatorPath, operatorDigest, err := cacheDiagnosticExecutableIdentity()
	if err != nil {
		return result, err
	}
	workerDigest, err := regularExecutableDigest(worker.Path)
	if err != nil {
		return result, err
	}
	environmentDigest, err := cacheDiagnosticEnvironmentDigest(worker.Environment, base)
	if err != nil {
		return result, err
	}
	candidateBodyDigest, err := cacheDiagnosticCandidateBodyDigest(ctx, state)
	if err != nil {
		return result, err
	}
	if err := requireDiagnosticSlotsAbsent(state.CommonDir); err != nil {
		return result, err
	}
	record = newCacheDiagnosticRecord(state.Head, operatorDigest, workerDigest, environmentDigest, candidateBodyDigest)
	if err := createCacheDiagnosticRecord(state.CommonDir, record); err != nil {
		return result, err
	}
	receiptCreated = true

	result, err = executeCacheDiagnosticWorker(ctx, state, outer, buildWorktree, executionWorktree, output, operatorPath, worker, inputBytes, &record, cacheDiagnosticExecutionHooks{})
	if err != nil {
		return result, err
	}
	cleanupErr := cleanupReplayWorktreeSet(state.Root, worktrees, base)
	cleanupComplete = true
	worktrees = nil
	base = ""
	if cleanupErr != nil {
		return cacheDiagnosticFailure, cleanupErr
	}
	if err := persistCacheDiagnosticTerminal(state.CommonDir, &record, "completed", result); err != nil {
		return cacheDiagnosticFailure, err
	}
	return result, nil
}

func newCacheDiagnosticRecord(implementationCommit, operatorDigest, workerDigest, environmentDigest, candidateBodyDigest string) cacheDiagnosticRecord {
	return cacheDiagnosticRecord{
		DiagnosticVersion: cacheDiagnosticVersion, PlanCommit: cacheDiagnosticPlanCommit,
		ImplementationCommit: implementationCommit, ProgramCommit: cacheDiagnosticProgramCommit,
		V5PlanCommit: ReplayCachePlanCommit, V5ImplementationCommit: cacheDiagnosticV5Commit,
		PretrainingCommit: replayPretrainingCommit, EvidenceCommit: replayEvidenceCommit,
		V5ReceiptDigest: cacheDiagnosticV5ReceiptDigest, InputDigest: cacheDiagnosticInputDigest,
		InputSHA256: cacheDiagnosticInputSHA256, InputBytes: cacheDiagnosticInputBytes,
		OperatorSHA256: operatorDigest, WorkerSHA256: workerDigest, EnvironmentDigest: environmentDigest,
		CandidateBodyDigest: candidateBodyDigest, ResultSchemaDigest: cacheDiagnosticResultDigest,
		HypothesesDigest: cacheDiagnosticHypothesesDigest, DecisionDigest: cacheDiagnosticDecisionDigest,
		CreatedUTC: time.Now().UTC().Format(time.RFC3339), State: "started", WorkerStarts: 0, Result: "",
	}
}

func verifyCacheDiagnosticInvocationRoot(invocationRoot, resolvedRoot string) error {
	absolute, err := filepath.Abs(invocationRoot)
	if err != nil || absolute != filepath.Clean(invocationRoot) || absolute != resolvedRoot {
		return errors.New("cache diagnostic must be invoked from the exact repository root")
	}
	physical, err := filepath.EvalSymlinks(absolute)
	if err != nil || physical != absolute {
		return errors.New("cache diagnostic must be invoked from the exact repository root")
	}
	return nil
}

func verifyCacheDiagnosticTopology(ctx context.Context, state gitState) error {
	if !state.Clean {
		return errors.New("cache diagnostic requires a clean worktree")
	}
	parent, err := gitStringOutput(ctx, state.Root, "rev-parse", state.Head+"^")
	if err != nil || parent != cacheDiagnosticPlanCommit {
		return errors.New("X6 is not the direct child of the accepted diagnostic plan")
	}
	planParent, err := gitStringOutput(ctx, state.Root, "rev-parse", cacheDiagnosticPlanCommit+"^")
	if err != nil || planParent != cacheDiagnosticProgramCommit {
		return errors.New("diagnostic plan is not the direct child of the governing program")
	}
	planDiff, err := gitStringOutput(ctx, state.Root, "diff", "--name-status", "--no-renames", cacheDiagnosticProgramCommit, cacheDiagnosticPlanCommit, "--")
	if err != nil || planDiff != "A\t"+cacheDiagnosticPlanPath {
		return errors.New("diagnostic plan commit has unexpected changes")
	}
	implementationDiff, err := gitStringOutput(ctx, state.Root, "diff", "--name-status", "--no-renames", cacheDiagnosticPlanCommit, state.Head, "--")
	want := strings.Join([]string{
		"A\tinternal/causalexpv2/cache_diagnostic.go",
		"A\tinternal/causalexpv2/cache_diagnostic_test.go",
		"A\tinternal/causalexpv2/cachediagexec/main.go",
	}, "\n")
	if err != nil || implementationDiff != want {
		return errors.New("X6 is outside the accepted three-file topology")
	}
	for _, path := range []string{cacheDiagnosticSourcePath, "internal/causalexpv2/cache_diagnostic_test.go", "internal/causalexpv2/cachediagexec/main.go", cacheDiagnosticPlanPath} {
		entry, entryErr := gitStringOutput(ctx, state.Root, "ls-tree", state.Head, "--", path)
		if entryErr != nil || !strings.HasPrefix(entry, "100644 blob ") {
			return errors.New("diagnostic topology contains a non-regular source")
		}
	}
	acceptedPlan, err := gitFile(ctx, state.Root, cacheDiagnosticPlanCommit, cacheDiagnosticPlanPath)
	if err != nil {
		return err
	}
	candidatePlan, err := gitFile(ctx, state.Root, state.Head, cacheDiagnosticPlanPath)
	if err != nil || !bytes.Equal(acceptedPlan, candidatePlan) {
		return errors.New("diagnostic plan changed after acceptance")
	}
	return nil
}

func verifyCacheDiagnosticSelfBinding(wantRevision string) error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("diagnostic executable has no build information")
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["vcs.revision"] != wantRevision || settings["vcs.modified"] != "false" {
		return errors.New("diagnostic executable is not a clean X6 build")
	}
	return nil
}

func verifyFailedV5DiagnosticPredecessor(commonDirectory string) error {
	encoded, err := os.ReadFile(filepath.Join(commonDirectory, "nous-attempts", "active-causal-diagnosis-v5-replay.json"))
	if err != nil || sha256Hex(encoded) != cacheDiagnosticV5ReceiptDigest {
		return errors.New("failed v5 diagnostic predecessor is unavailable")
	}
	return nil
}

func requireDiagnosticSlotsAbsent(commonDirectory string) error {
	paths := []string{
		cacheDiagnosticRecordPath(commonDirectory),
		attemptRecordPath(commonDirectory, PanelValidation),
		attemptProofRecordPath(commonDirectory, PanelValidation),
		attemptRecordPath(commonDirectory, PanelLocked),
		attemptProofRecordPath(commonDirectory, PanelLocked),
		resultPath(commonDirectory, PanelValidation),
		resultPath(commonDirectory, PanelLocked),
	}
	for _, path := range paths {
		if err := requireAbsent(path); err != nil {
			return errors.New("cache diagnostic slot is unavailable")
		}
	}
	return nil
}

func buildCacheDiagnosticInput() ([]byte, string, error) {
	capability := NewDiagnosticDevelopmentCapability()
	manifest := causalv2.PreregisteredManifest()
	corruption, err := capability.GenerateDevelopment(manifest.DevelopmentSeeds.Start, 0)
	if err != nil {
		return nil, "", err
	}
	fixtures := make([]PrivateFixture, manifest.TrainingSeeds.Count)
	for index := range fixtures {
		developmentSeed := manifest.DevelopmentSeeds.Start + int64(index)*manifest.DevelopmentSeeds.Step
		fixtures[index], err = capability.GenerateDevelopment(developmentSeed, index)
		if err != nil {
			return nil, "", err
		}
		fixtures[index].PublicFixture.Seed = manifest.TrainingSeeds.Start + int64(index)*manifest.TrainingSeeds.Step
		fixtures[index].PublicFixture.OpaqueToken, err = causalv2.PublicToken(string(PanelTraining), fixtures[index].PublicFixture.Seed, fixtures[index].PublicFixture.GeneratorAttempt)
		if err != nil {
			return nil, "", err
		}
		fixtures[index].PublicFixture.FixtureDigest = ""
		fixtures[index].PrivateFixtureDigest = ""
		if err := causalv2.SignPublicFixture(&fixtures[index].PublicFixture); err != nil {
			return nil, "", err
		}
		if err := causalv2.SignPrivateFixture(&fixtures[index]); err != nil {
			return nil, "", err
		}
	}
	input := ReplayInput{
		ReplayInputVersion: ReplayInputVersion, PlanCommit: PlanCommit,
		PretrainingCommit: replayPretrainingCommit, EvidenceCommit: replayEvidenceCommit,
		TrainingDigest: strings.Repeat("a", 64), BundleDigest: strings.Repeat("b", 64),
		Fixtures: fixtures, CorruptionFixture: corruption,
	}
	encoded, err := finalizeReplayInput(&input)
	return encoded, input.ReplayInputDigest, err
}

func verifyCacheDiagnosticProtocolDigests() error {
	for content, want := range map[string]string{
		cacheDiagnosticResultJSON:     cacheDiagnosticResultDigest,
		cacheDiagnosticHypothesesJSON: cacheDiagnosticHypothesesDigest,
		cacheDiagnosticDecisionJSON:   cacheDiagnosticDecisionDigest,
	} {
		if sha256Hex([]byte(content)) != want {
			return errors.New("cache diagnostic protocol digest mismatch")
		}
	}
	return nil
}

func cacheDiagnosticExecutableIdentity() (string, string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", "", err
	}
	digest, err := regularExecutableDigest(path)
	return path, digest, err
}

func regularExecutableDigest(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", errors.New("diagnostic executable identity is invalid")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sha256Hex(encoded), nil
}

func cacheDiagnosticEnvironmentDigest(environment []string, base string) (string, error) {
	values, parsedBase, err := parsePinnedWorkerEnvironment(environment)
	if err != nil || parsedBase != filepath.Clean(base) {
		return "", errors.New("diagnostic environment is invalid")
	}
	normalized := append([]string(nil), environment...)
	for index, entry := range normalized {
		name, value, _ := strings.Cut(entry, "=")
		if name == "PATH" || name == "HOME" || name == "XDG_CONFIG_HOME" {
			relative, relErr := filepath.Rel(base, value)
			if relErr != nil || strings.HasPrefix(relative, "..") {
				return "", errors.New("diagnostic environment is outside its root")
			}
			normalized[index] = name + "=$DIAGNOSTIC_ROOT/" + filepath.ToSlash(relative)
		}
	}
	if values["HOME"] == "" {
		return "", errors.New("diagnostic environment is incomplete")
	}
	encoded, err := causalv2.CanonicalJSON(normalized)
	if err != nil {
		return "", err
	}
	return sha256Hex(encoded), nil
}

func cacheDiagnosticCandidateBodyDigest(ctx context.Context, state gitState) (string, error) {
	source, err := gitFile(ctx, state.Root, state.Head, cacheDiagnosticSourcePath)
	if err != nil {
		return "", err
	}
	file, err := parser.ParseFile(token.NewFileSet(), cacheDiagnosticSourcePath, source, 0)
	if err != nil {
		return "", err
	}
	var body *ast.BlockStmt
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "candidateAuditWorkerCacheRoot" {
			body = function.Body
			break
		}
	}
	if body == nil {
		return "", errors.New("diagnostic candidate body is absent")
	}
	var normalized bytes.Buffer
	if err := format.Node(&normalized, token.NewFileSet(), body); err != nil {
		return "", err
	}
	return sha256Hex(normalized.Bytes()), nil
}

func createCacheDiagnosticRecord(commonDirectory string, record cacheDiagnosticRecord) error {
	directory := filepath.Dir(cacheDiagnosticRecordPath(commonDirectory))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	encoded, err := causalv2.CanonicalJSON(record)
	if err != nil {
		return err
	}
	if err := writeExclusiveSynced(cacheDiagnosticRecordPath(commonDirectory), encoded); err != nil {
		return errors.New("create exclusive cache diagnostic receipt")
	}
	return syncDirectory(directory)
}

func persistCacheDiagnosticRecord(commonDirectory string, record cacheDiagnosticRecord) error {
	path := cacheDiagnosticRecordPath(commonDirectory)
	directory := filepath.Dir(path)
	encoded, err := causalv2.CanonicalJSON(record)
	if err != nil {
		return err
	}
	temporary := path + ".next"
	if err := requireAbsent(temporary); err != nil {
		return err
	}
	if err := writeExclusiveSynced(temporary, encoded); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return syncDirectory(directory)
}

func persistCacheDiagnosticTerminal(commonDirectory string, record *cacheDiagnosticRecord, state, result string) error {
	if record == nil || record.WorkerStarts < 0 || record.WorkerStarts > 1 || state != "completed" && state != "failed" || state == "completed" && record.WorkerStarts != 1 {
		return errors.New("invalid cache diagnostic terminal transition")
	}
	if state == "failed" && result != cacheDiagnosticFailure {
		return errors.New("invalid cache diagnostic failure result")
	}
	record.State = state
	record.Result = result
	return persistCacheDiagnosticRecord(commonDirectory, *record)
}

func executeCacheDiagnosticWorker(ctx context.Context, state gitState, outer diagnosticOuterSnapshot, buildWorktree, executionWorktree, output, operatorPath string, worker regenerationExecutable, inputBytes []byte, record *cacheDiagnosticRecord, hooks cacheDiagnosticExecutionHooks) (string, error) {
	if record == nil {
		return cacheDiagnosticFailure, errors.New("diagnostic receipt is absent")
	}
	if err := verifyDiagnosticOuterEnvelope(worker.Environment, outer); err != nil {
		return cacheDiagnosticFailure, err
	}
	if err := verifyCacheDiagnosticExecutionState(ctx, state, buildWorktree, executionWorktree, "before", hooks); err != nil {
		return cacheDiagnosticFailure, err
	}
	inputRead, inputWrite, err := os.Pipe()
	if err != nil {
		return cacheDiagnosticFailure, err
	}
	defer inputRead.Close()
	defer inputWrite.Close()
	outputHandle, err := os.Open(output)
	if err != nil {
		return cacheDiagnosticFailure, err
	}
	defer outputHandle.Close()
	command := exec.CommandContext(ctx, worker.Path, worker.PrefixArgs...)
	command.Dir = executionWorktree
	command.Env = append([]string(nil), worker.Environment...)
	command.ExtraFiles = []*os.File{inputRead, outputHandle}
	var workerOutput bytes.Buffer
	command.Stdout, command.Stderr = &workerOutput, &workerOutput
	operatorDigest, err := regularExecutableDigest(operatorPath)
	if err != nil || operatorDigest != record.OperatorSHA256 {
		return cacheDiagnosticFailure, errors.New("diagnostic operator changed before worker start")
	}
	workerDigest, err := regularExecutableDigest(worker.Path)
	if err != nil || workerDigest != record.WorkerSHA256 {
		return cacheDiagnosticFailure, errors.New("diagnostic worker changed before worker start")
	}
	if err := command.Start(); err != nil {
		return cacheDiagnosticFailure, errors.New("diagnostic worker start failed")
	}
	record.WorkerStarts = 1
	if err := persistCacheDiagnosticRecord(state.CommonDir, *record); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return cacheDiagnosticFailure, errors.New("diagnostic worker start persistence failed")
	}
	if hooks.afterStartedPersist != nil {
		if err := hooks.afterStartedPersist(); err != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
			return cacheDiagnosticFailure, errors.New("diagnostic worker interrupted after durable start")
		}
	}
	_ = inputRead.Close()
	writeErr := writeAllAndClose(inputWrite, inputBytes)
	waitErr := command.Wait()
	exitError, exited := waitErr.(*exec.ExitError)
	if writeErr != nil || !exited || exitError.ExitCode() != 1 || strings.TrimSpace(workerOutput.String()) != "regenerated replay report has the wrong training digest" {
		return cacheDiagnosticFailure, errors.New("diagnostic worker terminal mismatch")
	}
	entries, err := os.ReadDir(output)
	if err != nil || len(entries) != 0 {
		return cacheDiagnosticFailure, errors.New("diagnostic worker published unexpected output")
	}
	if err := verifyCacheDiagnosticExecutionState(ctx, state, buildWorktree, executionWorktree, "after", hooks); err != nil {
		return cacheDiagnosticFailure, err
	}
	if err := verifyDiagnosticOuterEnvelope(worker.Environment, outer); err != nil {
		return cacheDiagnosticFailure, err
	}
	result, err := runCacheDiagnosticBracket(outer, hooks.bracketStep)
	if err != nil {
		return cacheDiagnosticFailure, err
	}
	if err := verifyDiagnosticOuterEnvelope(worker.Environment, outer); err != nil {
		return cacheDiagnosticFailure, err
	}
	return result, nil
}

func verifyCacheDiagnosticExecutionState(ctx context.Context, state gitState, buildWorktree, executionWorktree, stage string, hooks cacheDiagnosticExecutionHooks) error {
	if hooks.verifyExecutionState != nil {
		return hooks.verifyExecutionState(stage)
	}
	if err := verifyResolvedReplayWorktree(ctx, state.Root, buildWorktree, replayPretrainingCommit); err != nil {
		return err
	}
	if err := verifyCleanReplayWorktree(ctx, executionWorktree, replayPretrainingCommit); err != nil {
		return err
	}
	currentState, err := resolveGitState(ctx, state.Root)
	if err != nil || currentState.Head != state.Head || !currentState.Clean || verifyPinnedGitRepositoryState(ctx, currentState) != nil {
		return errors.New("repository changed during diagnostic worker execution")
	}
	return nil
}

func runCacheDiagnosticBracket(outer diagnosticOuterSnapshot, step func(int)) (string, error) {
	referenceA, err := referenceAuditWorkerCacheRoots(outer)
	if err != nil {
		return cacheDiagnosticFailure, err
	}
	if step != nil {
		step(1)
	}
	currentA := currentAuditWorkerCacheRoots(outer) == nil
	if step != nil {
		step(2)
	}
	candidateA := candidateAuditWorkerCacheRoots(outer) == nil
	if step != nil {
		step(3)
	}
	candidateB := candidateAuditWorkerCacheRoots(outer) == nil
	if step != nil {
		step(4)
	}
	currentB := currentAuditWorkerCacheRoots(outer) == nil
	if step != nil {
		step(5)
	}
	referenceB, err := referenceAuditWorkerCacheRoots(outer)
	if err != nil {
		return cacheDiagnosticFailure, err
	}
	if step != nil {
		step(6)
	}
	if referenceA.digest != referenceB.digest || referenceA.accepted != referenceB.accepted || currentA != currentB || candidateA != candidateB {
		return cacheDiagnosticFailure, errors.New("diagnostic metadata was not stable")
	}
	return classifyCacheDiagnostic(referenceA.accepted, currentA, candidateA), nil
}

func classifyCacheDiagnostic(referenceAccepted, currentAccepted, candidateAccepted bool) string {
	switch {
	case !referenceAccepted && !currentAccepted && !candidateAccepted:
		return cacheDiagnosticContractRejection
	case referenceAccepted && !currentAccepted && candidateAccepted:
		return cacheDiagnosticImplementationDefect
	case referenceAccepted && currentAccepted && candidateAccepted:
		return cacheDiagnosticNonReproduction
	default:
		return cacheDiagnosticUnsafeDisagreement
	}
}

func captureDiagnosticOuterEnvelope(environment []string, expectedBase string) (diagnosticOuterSnapshot, error) {
	snapshot, err := capturePinnedWorkerEnvironment(environment, expectedBase)
	if err != nil {
		return diagnosticOuterSnapshot{}, err
	}
	return diagnosticOuterSnapshot{
		base: snapshot.base, home: snapshot.home, xdg: snapshot.xdg,
		baseDevice: snapshot.baseDevice, baseInode: snapshot.baseInode,
		homeDevice: snapshot.homeDevice, homeInode: snapshot.homeInode,
		xdgDevice: snapshot.xdgDevice, xdgInode: snapshot.xdgInode,
	}, nil
}

func verifyDiagnosticOuterEnvelope(environment []string, snapshot diagnosticOuterSnapshot) error {
	values, base, err := parsePinnedWorkerEnvironment(environment)
	if err != nil || base != snapshot.base || values["HOME"] != snapshot.home || values["XDG_CONFIG_HOME"] != snapshot.xdg {
		return errors.New("diagnostic outer environment changed")
	}
	targets := []struct {
		path   string
		device uint64
		inode  uint64
	}{
		{snapshot.base, snapshot.baseDevice, snapshot.baseInode},
		{snapshot.home, snapshot.homeDevice, snapshot.homeInode},
		{snapshot.xdg, snapshot.xdgDevice, snapshot.xdgInode},
	}
	for _, target := range targets {
		info, infoErr := os.Lstat(target.path)
		var stat unix.Stat_t
		statErr := unix.Lstat(target.path, &stat)
		if infoErr != nil || statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || uint64(stat.Dev) != target.device || uint64(stat.Ino) != target.inode {
			return errors.New("diagnostic outer cache identity changed")
		}
	}
	return nil
}

func currentAuditWorkerCacheRoots(snapshot diagnosticOuterSnapshot) error {
	budget := &workerCacheBudget{}
	if err := auditWorkerCacheRoot(snapshot.home, snapshot.homeDevice, snapshot.homeInode, budget); err != nil {
		return err
	}
	return auditWorkerCacheRoot(snapshot.xdg, snapshot.xdgDevice, snapshot.xdgInode, budget)
}

func candidateAuditWorkerCacheRoots(snapshot diagnosticOuterSnapshot) error {
	budget := &candidateWorkerCacheBudget{}
	if err := candidateAuditWorkerCacheRoot(snapshot.home, snapshot.homeDevice, snapshot.homeInode, budget); err != nil {
		return err
	}
	return candidateAuditWorkerCacheRoot(snapshot.xdg, snapshot.xdgDevice, snapshot.xdgInode, budget)
}

func candidateAuditWorkerCacheRoot(root string, rootDevice, rootInode uint64, budget *candidateWorkerCacheBudget) error {
	if budget == nil {
		return errors.New("candidate cache audit failed")
	}
	rootFD, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return errors.New("candidate cache audit failed")
	}
	defer unix.Close(rootFD)
	var rootStat unix.Stat_t
	if unix.Fstat(rootFD, &rootStat) != nil || uint64(rootStat.Dev) != rootDevice || uint64(rootStat.Ino) != rootInode || rootStat.Mode&unix.S_IFMT != unix.S_IFDIR {
		return errors.New("candidate cache audit failed")
	}
	var walk func(int, int, string) error
	walk = func(directoryFD, depth int, prefix string) error {
		buffer := make([]byte, 8192)
		for {
			n, readErr := unix.ReadDirent(directoryFD, buffer)
			if readErr != nil {
				return readErr
			}
			if n == 0 {
				return nil
			}
			consumed, _, names := unix.ParseDirent(buffer[:n], -1, nil)
			if consumed != n {
				return errors.New("candidate directory parse failed")
			}
			for _, name := range names {
				if name == "." || name == ".." {
					continue
				}
				if name == "" || strings.ContainsRune(name, filepath.Separator) {
					return errors.New("candidate cache name rejected")
				}
				relative := name
				if prefix != "" {
					relative = prefix + "/" + name
				}
				budget.pathBytes += len(relative)
				if budget.pathBytes > 1<<20 || depth > 32 {
					return errors.New("candidate cache bounds exceeded")
				}
				var stat unix.Stat_t
				if unix.Fstatat(directoryFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW) != nil {
					return errors.New("candidate cache metadata changed")
				}
				switch stat.Mode & unix.S_IFMT {
				case unix.S_IFDIR:
					childFD, openErr := unix.Openat(directoryFD, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
					if openErr != nil {
						return openErr
					}
					var opened unix.Stat_t
					if unix.Fstat(childFD, &opened) != nil || opened.Dev != stat.Dev || opened.Ino != stat.Ino || opened.Mode&unix.S_IFMT != unix.S_IFDIR {
						_ = unix.Close(childFD)
						return errors.New("candidate cache directory changed")
					}
					walkErr := walk(childFD, depth+1, relative)
					closeErr := unix.Close(childFD)
					if walkErr != nil {
						return walkErr
					}
					if closeErr != nil {
						return closeErr
					}
				case unix.S_IFREG:
					budget.entries++
					if budget.entries > 4096 || stat.Nlink != 1 || stat.Size < 0 || stat.Size > (64<<20)-budget.logicalBytes {
						return errors.New("candidate cache file bounds exceeded")
					}
					budget.logicalBytes += stat.Size
				default:
					return errors.New("candidate cache entry type rejected")
				}
			}
		}
	}
	if err := walk(rootFD, 1, ""); err != nil {
		return errors.New("candidate cache audit failed")
	}
	return nil
}

func referenceAuditWorkerCacheRoots(snapshot diagnosticOuterSnapshot) (referenceAuditResult, error) {
	budget := &referenceCacheBudget{}
	records := []referenceMetadataRecord{}
	accepted := true
	for _, root := range []struct {
		label  byte
		path   string
		device uint64
		inode  uint64
	}{
		{'H', snapshot.home, snapshot.homeDevice, snapshot.homeInode},
		{'X', snapshot.xdg, snapshot.xdgDevice, snapshot.xdgInode},
	} {
		if err := referenceAuditWorkerCacheRoot(root.path, root.label, root.device, root.inode, budget, &records, &accepted); err != nil {
			return referenceAuditResult{}, err
		}
	}
	digest := referenceMetadataDigest(snapshot, records)
	return referenceAuditResult{accepted: accepted, digest: digest}, nil
}

func referenceAuditWorkerCacheRoot(root string, label byte, rootDevice, rootInode uint64, budget *referenceCacheBudget, records *[]referenceMetadataRecord, accepted *bool) error {
	if budget == nil || records == nil || accepted == nil {
		return errors.New("reference cache audit failed")
	}
	var walk func(string, int, string) error
	walk = func(directoryPath string, depth int, prefix string) error {
		var before unix.Stat_t
		if unix.Lstat(directoryPath, &before) != nil || before.Mode&unix.S_IFMT != unix.S_IFDIR {
			return errors.New("reference directory unavailable")
		}
		fd, openErr := unix.Open(directoryPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if openErr != nil {
			return openErr
		}
		file := os.NewFile(uintptr(fd), "reference-cache-directory")
		if file == nil {
			_ = unix.Close(fd)
			return errors.New("reference directory descriptor unavailable")
		}
		var opened unix.Stat_t
		if unix.Fstat(fd, &opened) != nil || opened.Dev != before.Dev || opened.Ino != before.Ino || opened.Mode&unix.S_IFMT != unix.S_IFDIR {
			_ = file.Close()
			return errors.New("reference directory identity changed")
		}
		for {
			names, readErr := file.Readdirnames(128)
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				_ = file.Close()
				return readErr
			}
			for _, name := range names {
				if name == "." || name == ".." {
					continue
				}
				if name == "" || strings.ContainsRune(name, filepath.Separator) {
					_ = file.Close()
					return errors.New("reference cache name rejected")
				}
				relative := name
				if prefix != "" {
					relative = prefix + "/" + name
				}
				budget.hardEntries++
				budget.hardPathBytes += len(relative)
				if budget.hardEntries > 8192 || budget.hardPathBytes > 2<<20 || depth+1 > 64 {
					_ = file.Close()
					return errors.New("reference diagnostic ceiling exceeded")
				}
				budget.contractPathBytes += len(relative)
				if budget.contractPathBytes > 1<<20 || depth+1 > 32 {
					*accepted = false
				}
				childPath := filepath.Join(directoryPath, name)
				var stat unix.Stat_t
				if unix.Lstat(childPath, &stat) != nil {
					_ = file.Close()
					return errors.New("reference cache metadata changed")
				}
				record := referenceMetadataRecord{root: label, path: filepath.ToSlash(relative), typeBits: uint32(stat.Mode & unix.S_IFMT), device: uint64(stat.Dev), inode: uint64(stat.Ino), links: uint64(stat.Nlink), size: stat.Size}
				*records = append(*records, record)
				switch stat.Mode & unix.S_IFMT {
				case unix.S_IFDIR:
					if err := walk(childPath, depth+1, relative); err != nil {
						_ = file.Close()
						return err
					}
				case unix.S_IFREG:
					budget.contractEntries++
					if stat.Nlink != 1 || stat.Size < 0 || budget.contractEntries > 4096 || stat.Size > (64<<20)-budget.contractLogicalBytes {
						*accepted = false
					}
					if stat.Size < 0 || stat.Size > (128<<20)-budget.hardLogicalBytes {
						_ = file.Close()
						return errors.New("reference diagnostic logical ceiling exceeded")
					}
					budget.contractLogicalBytes += stat.Size
					budget.hardLogicalBytes += stat.Size
				default:
					budget.contractEntries++
					if budget.contractEntries > 4096 {
						*accepted = false
					}
					*accepted = false
				}
			}
			if errors.Is(readErr, io.EOF) {
				break
			}
		}
		return file.Close()
	}
	var rootStat unix.Stat_t
	if unix.Lstat(root, &rootStat) != nil || uint64(rootStat.Dev) != rootDevice || uint64(rootStat.Ino) != rootInode || rootStat.Mode&unix.S_IFMT != unix.S_IFDIR {
		return errors.New("reference root identity changed")
	}
	return walk(root, 0, "")
}

func referenceMetadataDigest(snapshot diagnosticOuterSnapshot, records []referenceMetadataRecord) [32]byte {
	sort.Slice(records, func(left, right int) bool {
		if records[left].root != records[right].root {
			return records[left].root < records[right].root
		}
		return records[left].path < records[right].path
	})
	hash := sha256.New()
	writeUint64 := func(value uint64) {
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], value)
		_, _ = hash.Write(encoded[:])
	}
	for _, root := range []struct {
		label byte
		dev   uint64
		ino   uint64
	}{{'H', snapshot.homeDevice, snapshot.homeInode}, {'X', snapshot.xdgDevice, snapshot.xdgInode}} {
		_, _ = hash.Write([]byte{root.label})
		writeUint64(root.dev)
		writeUint64(root.ino)
	}
	for _, record := range records {
		_, _ = hash.Write([]byte{record.root})
		writeUint64(uint64(len(record.path)))
		_, _ = io.WriteString(hash, record.path)
		writeUint64(uint64(record.typeBits))
		writeUint64(record.device)
		writeUint64(record.inode)
		writeUint64(record.links)
		writeUint64(uint64(record.size))
	}
	var digest [32]byte
	copy(digest[:], hash.Sum(nil))
	return digest
}
