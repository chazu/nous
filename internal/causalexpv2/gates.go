package causalexpv2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chazu/nous/internal/causalv2"
	"golang.org/x/sys/unix"
)

const FrozenConstantsPath = "internal/causalexpv2/freeze.go"

const (
	replayEvidenceCommit       = "7482d1b9712f49cb988623a87ec8bb1c34667a26"
	replayPretrainingCommit    = "89a9221e97dd17d6ba220a22a12d4c0328417ffb"
	pinnedGoPath               = "/Users/chazu/.local/share/mise/installs/go/1.25.12/bin/go"
	pinnedGoSHA256             = "8612de418d551a418517845c05cebdcfed49095cd08ef0a4d682bb2a5cf4896c"
	pinnedGoVersion            = "go version go1.25.12 darwin/arm64"
	pinnedGOROOT               = "/Users/chazu/.local/share/mise/installs/go/1.25.12"
	pinnedGOROOTFiles          = 14531
	pinnedGOROOTSHA256         = "77a814b12481fa12b070a905d2d6fc1ab9671b0e2866d7ffb85c9b37861d9da9"
	pinnedGitPath              = "/opt/homebrew/bin/git"
	pinnedGitSHA256            = "00ad7d9b0732c80bd8971e443a7129cf09d0957ea4c1f6cf581bbffe6c2e0505"
	pinnedGitVersion           = "git version 2.47.1"
	pinnedGitConfigSHA256      = "58610c019ec3c32186d8dc2a9e5a18b900dafdb75047d7a56475baa84d268a15"
	pinnedGitInfoExcludeSHA256 = "6671fe83b7a07c8932ee89164d1f2793b2318058eb8b98dc5c06ee0a5a3b0ec1"
	resolvedGoModSHA256        = "e5875629b398cfccd32df7604196702818eb2fc5e9b605897a0207565c64866c"
	resolvedGoSumSHA256        = "930d1ecfb0438e23115d1365f24fbddcd27fa0a97d144436eb7bafc208bbb6d4"
)

var (
	replayBuildPreflights    atomic.Int64
	replayEvidenceReads      atomic.Int64
	replayCapabilityMints    atomic.Int64
	replayInputConstructions atomic.Int64
	replayWorkerStarts       atomic.Int64
)

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
	Path        string
	PrefixArgs  []string
	Environment []string
}

// MintReplayCapability verifies the committed evidence at R and proves that
// the candidate differs from R only by the allowlisted constants source.
func mintReplayCapability(ctx context.Context, repoRoot, evidenceCommit string) (*ReplayCapability, error) {
	replayCapabilityMints.Add(1)
	if err := verifyProtectedGitEnvironment(); err != nil {
		return nil, err
	}
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
	replayEvidenceReads.Add(1)
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
	executable, err := pinnedReplayBuilder(ctx, state.Root)
	if err != nil {
		return nil, err
	}
	replayInputConstructions.Add(1)
	input := ReplayInput{ReplayInputVersion: ReplayInputVersion, PlanCommit: PlanCommit, PretrainingCommit: report.PretrainingCommit, EvidenceCommit: evidenceCommit, TrainingDigest: report.TrainingDigest, BundleDigest: bundle.BundleDigest, Fixtures: append([]PrivateFixture(nil), bundle.Fixtures...), CorruptionFixture: verified.CorruptionFixture}
	inputBytes, err := finalizeReplayInput(&input)
	if err != nil {
		return nil, fmt.Errorf("construct canonical replay input: %w", err)
	}
	return &ReplayCapability{repositoryRoot: state.Root, pretrainingCommit: report.PretrainingCommit, evidenceCommit: evidenceCommit, seeds: SeedRange{122001, 12, 1}, reportDigest: report.TrainingDigest, bundleDigest: bundle.BundleDigest, reportBytes: reportBytes, bundleBytes: bundleBytes, replayInputBytes: inputBytes, buildExecutable: executable}, nil
}

func verifyCandidateConstantsState(ctx context.Context, state gitState, evidenceCommit string) error {
	if evidenceCommit != replayEvidenceCommit {
		return errors.New("candidate is not bound to fixed R3 evidence")
	}
	if state.Clean {
		parent, err := gitStringOutput(ctx, state.Root, "rev-parse", state.Head+"^")
		if err != nil || verifyReplayRepairExecutableCommit(ctx, state.Root, parent) != nil {
			return errors.New("candidate C4 is not the direct child of exact replay-repair executable X4")
		}
		changed, err := gitStringOutput(ctx, state.Root, "diff", "--name-status", "--no-renames", parent, state.Head, "--")
		if err != nil || strings.TrimSpace(changed) != "M\t"+FrozenConstantsPath {
			return errors.New("candidate C4 is not the constants-only direct child of X4")
		}
		entry, err := gitStringOutput(ctx, state.Root, "ls-tree", state.Head, "--", FrozenConstantsPath)
		if err != nil || !strings.HasPrefix(entry, "100644 blob ") {
			return errors.New("candidate C4 constants are not a regular 100644 blob")
		}
		return nil
	}
	if err := verifyReplayRepairExecutableCommit(ctx, state.Root, state.Head); err != nil {
		return fmt.Errorf("dirty replay is not running from exact X4: %w", err)
	}
	status, err := gitStringOutput(ctx, state.Root, "status", "--porcelain", "--untracked-files=all")
	// gitOutput trims the porcelain record's single leading index-column
	// space; an unstaged-only edit therefore has exactly this representation.
	if err != nil || status != "M "+FrozenConstantsPath {
		return errors.New("dirty replay worktree is not the exact unstaged constants edit on X4")
	}
	info, err := os.Lstat(filepath.Join(state.Root, FrozenConstantsPath))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 {
		return errors.New("dirty X4 constants are not a regular 0644 file")
	}
	changed, err := gitStringOutput(ctx, state.Root, "diff", "--name-only", state.Head, "--")
	if err != nil || strings.TrimSpace(changed) != FrozenConstantsPath {
		return errors.New("dirty replay diff from X4 is not the one-file constants edit")
	}
	return nil
}

func verifyReplayRepairExecutableCommit(ctx context.Context, root, commit string) error {
	parent, err := gitStringOutput(ctx, root, "rev-parse", commit+"^")
	if err != nil || parent != ReplayRepairPlanCommit {
		return errors.New("X4 is not the direct child of the accepted replay-repair plan")
	}
	changed, err := gitStringOutput(ctx, root, "diff", "--name-status", "--no-renames", replayEvidenceCommit, commit, "--")
	if err != nil {
		return err
	}
	got := splitNonemptyLines(changed)
	slices.Sort(got)
	want := []string{
		"A\tdocs/active-causal-diagnosis-v4-replay-amendment.md",
		"M\tgo.mod",
		"M\tgo.sum",
		"M\tinternal/causalexpv2/gates.go",
		"M\tinternal/causalexpv2/provenance.go",
		"M\tinternal/causalexpv2/provenance_test.go",
		"M\tinternal/causalexpv2/replay_gate.go",
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		return fmt.Errorf("X4 diff is outside the accepted status-sensitive allowlist: got %q", got)
	}
	for _, record := range want {
		_, path, _ := strings.Cut(record, "\t")
		entry, entryErr := gitStringOutput(ctx, root, "ls-tree", commit, "--", path)
		if entryErr != nil || !strings.HasPrefix(entry, "100644 blob ") {
			return fmt.Errorf("X4 path is not a regular 100644 blob: %s", path)
		}
	}
	accepted, err := gitFile(ctx, root, ReplayRepairPlanCommit, "docs/active-causal-diagnosis-v4-replay-amendment.md")
	if err != nil {
		return err
	}
	candidate, err := gitFile(ctx, root, commit, "docs/active-causal-diagnosis-v4-replay-amendment.md")
	if err != nil || !bytes.Equal(accepted, candidate) {
		return errors.New("X4 amendment differs from accepted replay-repair plan")
	}
	for path, digest := range map[string]string{"go.mod": resolvedGoModSHA256, "go.sum": resolvedGoSumSHA256} {
		content, readErr := gitFile(ctx, root, commit, path)
		if readErr != nil || sha256Hex(content) != digest {
			return fmt.Errorf("X4 %s does not equal the accepted resolved module metadata", path)
		}
	}
	if err := verifyEmptyFreezeAt(ctx, root, commit); err != nil {
		return err
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

func pinnedReplayBuilder(ctx context.Context, repoRoot string) (regenerationExecutable, error) {
	if err := verifyProtectedGitEnvironment(); err != nil {
		return regenerationExecutable{}, err
	}
	resolved, err := filepath.EvalSymlinks(pinnedGoPath)
	if err != nil || resolved != pinnedGoPath {
		return regenerationExecutable{}, errors.New("pinned Go path does not resolve exactly")
	}
	if err := verifyRegularFileDigest(pinnedGoPath, pinnedGoSHA256); err != nil {
		return regenerationExecutable{}, fmt.Errorf("pinned Go driver: %w", err)
	}
	count, digest, err := gorootManifest(pinnedGOROOT)
	if err != nil || count != pinnedGOROOTFiles || digest != pinnedGOROOTSHA256 {
		return regenerationExecutable{}, errors.New("pinned GOROOT manifest mismatch")
	}
	version := exec.CommandContext(ctx, pinnedGoPath, "version")
	version.Env = fixedGoEnvironment("", "", "")
	output, err := version.Output()
	if err != nil || strings.TrimSpace(string(output)) != pinnedGoVersion {
		return regenerationExecutable{}, errors.New("pinned Go version mismatch")
	}
	state, err := resolveGitState(ctx, repoRoot)
	if err != nil {
		return regenerationExecutable{}, err
	}
	if err := verifyPinnedGitRepositoryState(ctx, state); err != nil {
		return regenerationExecutable{}, err
	}
	if err := verifyPinnedGitTool(ctx); err != nil {
		return regenerationExecutable{}, err
	}
	return validateRegenerationExecutable(regenerationExecutable{Path: pinnedGoPath})
}

func verifyPinnedProtectedRuntime(ctx context.Context, repoRoot string) error {
	if err := verifyProtectedGitEnvironment(); err != nil {
		return err
	}
	if runtime.Version() != "go1.25.12" || runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return errors.New("protected executable runtime is not pinned")
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("protected executable has no Go build settings")
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["CGO_ENABLED"] != "0" || settings["GOARM64"] != "v8.0" || settings["GOEXPERIMENT"] != "" || settings["DefaultGODEBUG"] != "" || settings["GOFIPS140"] != "" && settings["GOFIPS140"] != "off" {
		return errors.New("protected executable build settings are not pinned")
	}
	if settings["-buildmode"] != "exe" || settings["-compiler"] != "gc" {
		return errors.New("protected executable compiler mode is not pinned")
	}
	for key := range settings {
		if strings.HasPrefix(key, "-") && key != "-buildmode" && key != "-compiler" {
			return fmt.Errorf("protected executable has forbidden build setting %s", key)
		}
	}
	if err := verifyProtectedRuntimeEnvironment(); err != nil {
		return err
	}
	_, err := pinnedReplayBuilder(ctx, repoRoot)
	return err
}

func verifyProtectedRuntimeEnvironment() error {
	if os.Getenv("GODEBUG") != "" || os.Getenv("GOFIPS140") != "off" {
		return errors.New("protected executable runtime environment is not pinned")
	}
	return nil
}

func gorootManifest(root string) (int, string, error) {
	root = filepath.Clean(root)
	paths := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("GOROOT contains symlink %s", path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("GOROOT contains special entry %s", path)
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return 0, "", err
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return 0, "", err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return 0, "", err
		}
		for _, value := range []string{filepath.ToSlash(relative), strconv.FormatUint(uint64(info.Mode().Perm()), 8), strconv.FormatInt(info.Size(), 10)} {
			_, _ = io.WriteString(hash, value)
			_, _ = hash.Write([]byte{0})
		}
		file, err := os.Open(path)
		if err != nil {
			return 0, "", err
		}
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return 0, "", copyErr
		}
		if closeErr != nil {
			return 0, "", closeErr
		}
		_, _ = hash.Write([]byte{0})
	}
	return len(paths), hex.EncodeToString(hash.Sum(nil)), nil
}

func fixedGoEnvironment(moduleCache, buildCache, temporaryDirectory string) []string {
	goPath := "/nonexistent/nous-gopath"
	if moduleCache != "" {
		goPath = filepath.Join(filepath.Dir(moduleCache), "gopath")
	}
	return []string{
		"GOROOT=" + pinnedGOROOT,
		"GOTOOLCHAIN=local",
		"GOENV=off",
		"GOWORK=off",
		"GOFLAGS=",
		"GOEXPERIMENT=",
		"CGO_ENABLED=0",
		"GOOS=darwin",
		"GOARCH=arm64",
		"GOARM64=v8.0",
		"GODEBUG=",
		"GOFIPS140=off",
		"GOMODCACHE=" + moduleCache,
		"GOPATH=" + goPath,
		"GOCACHE=" + buildCache,
		"TMPDIR=" + temporaryDirectory,
		"GOPROXY=https://proxy.golang.org",
		"GOSUMDB=sum.golang.org",
		"GOPRIVATE=",
		"GONOPROXY=",
		"GONOSUMDB=",
	}
}

func verifyRegularFileDigest(path, want string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("path is not a regular file")
	}
	encoded, err := os.ReadFile(path)
	if err != nil || sha256Hex(encoded) != want {
		return errors.New("file digest mismatch")
	}
	return nil
}

func sha256Hex(encoded []byte) string {
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func verifyPinnedGitTool(ctx context.Context) error {
	resolved, err := filepath.EvalSymlinks(pinnedGitPath)
	if err != nil || !filepath.IsAbs(resolved) {
		return errors.New("pinned Git path does not resolve to an absolute path")
	}
	if err := verifyRegularFileDigest(resolved, pinnedGitSHA256); err != nil {
		return fmt.Errorf("pinned Git executable: %w", err)
	}
	command := exec.CommandContext(ctx, pinnedGitPath, "--version")
	command.Env = []string{"LC_ALL=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_NO_REPLACE_OBJECTS=1"}
	output, err := command.Output()
	if err != nil || strings.TrimSpace(string(output)) != pinnedGitVersion {
		return errors.New("pinned Git version mismatch")
	}
	return nil
}

func verifyProtectedGitEnvironment() error {
	want := map[string]string{
		"PATH":                   "/opt/homebrew/bin",
		"GIT_CONFIG_NOSYSTEM":    "1",
		"GIT_CONFIG_GLOBAL":      "/dev/null",
		"GIT_CONFIG_SYSTEM":      "/dev/null",
		"GIT_OPTIONAL_LOCKS":     "0",
		"GIT_NO_REPLACE_OBJECTS": "1",
		"GIT_ATTR_NOSYSTEM":      "1",
		"GIT_TERMINAL_PROMPT":    "0",
	}
	for name, expected := range want {
		if os.Getenv(name) != expected {
			return fmt.Errorf("protected Git environment has noncanonical %s", name)
		}
	}
	for _, name := range []string{"GIT_ASKPASS", "GIT_SSH_COMMAND", "GIT_CONFIG_COUNT", "GIT_CONFIG_PARAMETERS", "GIT_EXTERNAL_DIFF", "GIT_DIFF_OPTS", "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_NAMESPACE", "GIT_SHALLOW_FILE", "GIT_EXEC_PATH"} {
		if _, present := os.LookupEnv(name); present {
			return fmt.Errorf("protected Git environment contains %s", name)
		}
	}
	path, err := exec.LookPath("git")
	if err != nil || path != pinnedGitPath {
		return errors.New("protected Git lookup does not select the pinned path")
	}
	return nil
}

func protectedGitCommandEnvironment() []string {
	return []string{
		"PATH=/opt/homebrew/bin",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"LC_ALL=C",
		"TZ=UTC",
	}
}

func verifyPinnedGitRepositoryState(ctx context.Context, state gitState) error {
	if err := verifyRegularFileDigest(filepath.Join(state.CommonDir, "config"), pinnedGitConfigSHA256); err != nil {
		return fmt.Errorf("Git common config: %w", err)
	}
	config, err := os.ReadFile(filepath.Join(state.CommonDir, "config"))
	if err != nil {
		return err
	}
	lower := strings.ToLower(string(config))
	for _, denied := range []string{"include", "worktreeconfig", "fsmonitor", "hooks", "external", "filter", "showuntrackedfiles"} {
		if strings.Contains(lower, denied) {
			return fmt.Errorf("Git common config contains forbidden setting %q", denied)
		}
	}
	if err := verifyRegularFileDigest(filepath.Join(state.CommonDir, "info", "exclude"), pinnedGitInfoExcludeSHA256); err != nil {
		return fmt.Errorf("Git info exclude: %w", err)
	}
	absent := []string{
		filepath.Join(state.CommonDir, "info", "attributes"),
		filepath.Join(state.CommonDir, "info", "grafts"),
		filepath.Join(state.CommonDir, "shallow"),
		filepath.Join(state.CommonDir, "objects", "info", "alternates"),
		filepath.Join(state.CommonDir, "objects", "info", "http-alternates"),
		filepath.Join(state.CommonDir, "config.worktree"),
		filepath.Join(state.CommonDir, "refs", "replace"),
	}
	worktreeConfigs, globErr := filepath.Glob(filepath.Join(state.CommonDir, "worktrees", "*", "config.worktree"))
	if globErr != nil || len(worktreeConfigs) != 0 {
		return errors.New("Git worktree config is present")
	}
	for _, path := range absent {
		if err := requireAbsent(path); err != nil {
			return fmt.Errorf("forbidden Git repository state: %w", err)
		}
	}
	packed, err := os.ReadFile(filepath.Join(state.CommonDir, "packed-refs"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if bytes.Contains(packed, []byte(" refs/replace/")) {
		return errors.New("packed replacement refs are present")
	}
	return nil
}

func pinnedWorkerEnvironment(ctx context.Context, base string) ([]string, error) {
	if err := verifyPinnedGitTool(ctx); err != nil {
		return nil, err
	}
	bin, home, xdg := filepath.Join(base, "worker-bin"), filepath.Join(base, "worker-home"), filepath.Join(base, "worker-xdg")
	for _, directory := range []string{bin, home, xdg} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return nil, err
		}
	}
	link := filepath.Join(bin, "git")
	if err := os.Symlink(pinnedGitPath, link); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(bin)
	if err != nil || len(entries) != 1 || entries[0].Name() != "git" || entries[0].Type()&os.ModeSymlink == 0 {
		return nil, errors.New("worker private bin is not the exact pinned Git link")
	}
	resolved, err := filepath.EvalSymlinks(link)
	pinnedResolved, pinnedErr := filepath.EvalSymlinks(pinnedGitPath)
	if err != nil || pinnedErr != nil || resolved != pinnedResolved {
		return nil, errors.New("worker Git link resolves incorrectly")
	}
	environment := []string{
		"PATH=" + bin,
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + xdg,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_ATTR_NOSYSTEM=1",
		"LC_ALL=C",
		"TZ=UTC",
	}
	if err := verifyPinnedWorkerEnvironment(environment); err != nil {
		return nil, err
	}
	return environment, nil
}

func verifyPinnedWorkerEnvironment(environment []string) error {
	if len(environment) != 11 {
		return errors.New("worker environment does not have the exact allowlist")
	}
	values := map[string]string{}
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if !found || name == "" {
			return errors.New("worker environment contains a malformed entry")
		}
		if _, duplicate := values[name]; duplicate {
			return errors.New("worker environment contains a duplicate entry")
		}
		values[name] = value
	}
	want := map[string]string{"GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_SYSTEM": "/dev/null", "GIT_OPTIONAL_LOCKS": "0", "GIT_NO_REPLACE_OBJECTS": "1", "GIT_ATTR_NOSYSTEM": "1", "LC_ALL": "C", "TZ": "UTC"}
	for name, expected := range want {
		if values[name] != expected {
			return fmt.Errorf("worker environment has noncanonical %s", name)
		}
	}
	for _, name := range []string{"PATH", "HOME", "XDG_CONFIG_HOME"} {
		if values[name] == "" || !filepath.IsAbs(values[name]) {
			return fmt.Errorf("worker environment has invalid %s", name)
		}
	}
	entries, err := os.ReadDir(values["PATH"])
	if err != nil || len(entries) != 1 || entries[0].Name() != "git" || entries[0].Type()&os.ModeSymlink == 0 {
		return errors.New("worker private bin changed")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(values["PATH"], "git"))
	pinnedResolved, pinnedErr := filepath.EvalSymlinks(pinnedGitPath)
	if err != nil || pinnedErr != nil || resolved != pinnedResolved {
		return errors.New("worker private Git changed")
	}
	for _, name := range []string{"HOME", "XDG_CONFIG_HOME"} {
		contents, readErr := os.ReadDir(values[name])
		if readErr != nil || len(contents) != 0 {
			return fmt.Errorf("worker private %s is not empty", name)
		}
	}
	return nil
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
	worktrees := []string{worktree}
	defer func() {
		if cleanupErr := cleanupReplayWorktreeSet(cap.repositoryRoot, worktrees, base); cleanupErr != nil {
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
	buildWorktree := ""
	if worker.Path == "" {
		buildWorktree = filepath.Join(base, "build-worktree")
		if err := runGit(ctx, cap.repositoryRoot, "worktree", "add", "--detach", buildWorktree, cap.pretrainingCommit); err != nil {
			return ReplayResult{}, err
		}
		worktrees = append(worktrees, buildWorktree)
		worker, err = buildReplayWorker(ctx, cap.buildExecutable, buildWorktree, filepath.Join(base, "causal-v2-replay-worker"))
		if err != nil {
			return ReplayResult{}, err
		}
		if err := verifyResolvedReplayWorktree(ctx, cap.repositoryRoot, buildWorktree, cap.pretrainingCommit); err != nil {
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
	if worker.Environment != nil {
		command.Env = append([]string(nil), worker.Environment...)
	} else {
		command.Env = append([]string(nil), os.Environ()...)
	}
	command.ExtraFiles = []*os.File{inputRead, outputHandle}
	var workerOutput bytes.Buffer
	command.Stdout, command.Stderr = &workerOutput, &workerOutput
	protected := cap.buildExecutable.Path != ""
	if protected {
		if err := verifyPinnedWorkerEnvironment(worker.Environment); err != nil {
			return ReplayResult{}, err
		}
		state, stateErr := resolveGitState(ctx, cap.repositoryRoot)
		if stateErr != nil || verifyPinnedGitRepositoryState(ctx, state) != nil {
			return ReplayResult{}, errors.New("repository Git state changed before worker start")
		}
		if err := verifyResolvedReplayWorktree(ctx, cap.repositoryRoot, buildWorktree, cap.pretrainingCommit); err != nil {
			return ReplayResult{}, errors.New("resolved build worktree changed before worker start")
		}
		if err := verifyCleanReplayWorktree(ctx, worktree, cap.pretrainingCommit); err != nil {
			return ReplayResult{}, err
		}
	}
	replayWorkerStarts.Add(1)
	if err := command.Start(); err != nil {
		return ReplayResult{}, fmt.Errorf("start replay worker: %w", err)
	}
	_ = inputRead.Close()
	writeErr := writeAllAndClose(inputWrite, cap.replayInputBytes)
	waitErr := command.Wait()
	var auditErr error
	if protected {
		if err := verifyPinnedWorkerEnvironment(worker.Environment); err != nil {
			auditErr = errors.Join(auditErr, err)
		}
		if err := verifyResolvedReplayWorktree(ctx, cap.repositoryRoot, buildWorktree, cap.pretrainingCommit); err != nil {
			auditErr = errors.Join(auditErr, errors.New("regeneration executable changed resolved detached build state"))
		}
		if err := verifyCleanReplayWorktree(ctx, worktree, cap.pretrainingCommit); err != nil {
			auditErr = errors.Join(auditErr, err)
		}
		state, stateErr := resolveGitState(ctx, cap.repositoryRoot)
		if stateErr != nil || verifyPinnedGitRepositoryState(ctx, state) != nil {
			auditErr = errors.Join(auditErr, errors.New("repository Git state changed after worker exit"))
		}
	} else {
		auditErr = verifyCleanReplayWorktree(ctx, worktree, cap.pretrainingCommit)
	}
	if writeErr != nil || waitErr != nil {
		return ReplayResult{}, errors.Join(fmt.Errorf("regeneration executable failed: %w: %s", errors.Join(writeErr, waitErr), strings.TrimSpace(workerOutput.String())), auditErr)
	}
	if auditErr != nil {
		return ReplayResult{}, auditErr
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
	args = append(args, "build", "-mod=mod", "-o", output, "./internal/causalexpv2/replayexec")
	base := filepath.Dir(output)
	moduleCache, buildCache, temporaryDirectory := filepath.Join(base, "gomodcache"), filepath.Join(base, "gocache"), filepath.Join(base, "tmp")
	goPath := filepath.Join(base, "gopath")
	for _, directory := range []string{moduleCache, buildCache, temporaryDirectory, goPath} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return regenerationExecutable{}, err
		}
	}
	command := exec.CommandContext(ctx, builder.Path, args...)
	command.Dir = worktree
	command.Env = fixedGoEnvironment(moduleCache, buildCache, temporaryDirectory)
	if outputBytes, err := command.CombinedOutput(); err != nil {
		return regenerationExecutable{}, fmt.Errorf("build fixed replay worker: %w: %s", err, strings.TrimSpace(string(outputBytes)))
	}
	environment, err := pinnedWorkerEnvironment(ctx, base)
	if err != nil {
		return regenerationExecutable{}, err
	}
	executable, err := validateRegenerationExecutable(regenerationExecutable{Path: output})
	if err != nil {
		return regenerationExecutable{}, err
	}
	executable.Environment = environment
	return executable, nil
}

func preflightReplayBuild(ctx context.Context, repositoryRoot, pretrainingCommit string, builder regenerationExecutable) (returnErr error) {
	replayBuildPreflights.Add(1)
	if err := verifyProtectedGitEnvironment(); err != nil {
		return err
	}
	if pretrainingCommit != replayPretrainingCommit {
		return errors.New("build-only preflight is not bound to E3")
	}
	state, err := resolveGitState(ctx, repositoryRoot)
	if err != nil {
		return err
	}
	if err := verifyPinnedGitRepositoryState(ctx, state); err != nil {
		return err
	}
	base, err := os.MkdirTemp("", "nous-causal-v4-build-preflight-")
	if err != nil {
		return err
	}
	worktree := filepath.Join(base, "worktree")
	if err := runGit(ctx, state.Root, "worktree", "add", "--detach", worktree, pretrainingCommit); err != nil {
		_ = os.RemoveAll(base)
		return err
	}
	defer func() {
		returnErr = errors.Join(returnErr, cleanupResolvedReplayWorktree(state.Root, worktree, base))
	}()
	detached, err := resolveGitState(ctx, worktree)
	if err != nil || detached.Head != pretrainingCommit || !detached.Clean {
		return errors.New("build-only preflight did not start at clean E3")
	}
	if _, err := buildReplayWorker(ctx, builder, worktree, filepath.Join(base, "worker")); err != nil {
		return err
	}
	return verifyResolvedReplayWorktree(ctx, state.Root, worktree, pretrainingCommit)
}

func verifyResolvedReplayWorktree(ctx context.Context, repositoryRoot, worktree, pretrainingCommit string) error {
	detached, err := resolveGitState(ctx, worktree)
	if err != nil || detached.Head != pretrainingCommit || detached.Clean {
		return errors.New("resolved replay worktree has the wrong commit or clean state")
	}
	changed, err := gitStringOutput(ctx, worktree, "diff", "--name-status", "--no-renames", "HEAD", "--")
	if err != nil || changed != "M\tgo.mod\nM\tgo.sum" {
		return errors.New("resolved replay build did not change exactly go.mod and go.sum")
	}
	if command := exec.CommandContext(ctx, pinnedGitPath, "-C", worktree, "diff", "--cached", "--quiet", "HEAD", "--"); command.Run() != nil {
		return errors.New("resolved replay build staged a change")
	}
	for _, args := range [][]string{{"ls-files", "--others", "--exclude-standard"}, {"ls-files", "--others", "--ignored", "--exclude-standard"}} {
		output, err := gitStringOutput(ctx, worktree, args...)
		if err != nil || output != "" {
			return errors.New("resolved replay build created an untracked or ignored path")
		}
	}
	rootState, err := resolveGitState(ctx, repositoryRoot)
	if err != nil {
		return err
	}
	for path, digest := range map[string]string{"go.mod": resolvedGoModSHA256, "go.sum": resolvedGoSumSHA256} {
		got, readErr := os.ReadFile(filepath.Join(worktree, path))
		info, statErr := os.Lstat(filepath.Join(worktree, path))
		want, gitErr := gitFile(ctx, repositoryRoot, rootState.Head, path)
		if readErr != nil || statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 || gitErr != nil || sha256Hex(got) != digest || !bytes.Equal(got, want) {
			return fmt.Errorf("resolved replay %s differs from exact X4 metadata", path)
		}
	}
	return verifyPinnedGitRepositoryState(ctx, rootState)
}

func verifyCleanReplayWorktree(ctx context.Context, worktree, pretrainingCommit string) error {
	detached, err := resolveGitState(ctx, worktree)
	if err != nil || detached.Head != pretrainingCommit || !detached.Clean {
		return errors.New("regeneration executable changed clean detached execution worktree")
	}
	for _, args := range [][]string{{"ls-files", "--others", "--exclude-standard"}, {"ls-files", "--others", "--ignored", "--exclude-standard"}} {
		output, listErr := gitStringOutput(ctx, worktree, args...)
		if listErr != nil || output != "" {
			return errors.New("regeneration executable created an untracked or ignored execution-worktree path")
		}
	}
	return nil
}

func cleanupReplayWorktreeSet(repositoryRoot string, worktrees []string, base string) error {
	var cleanupErr error
	for index := len(worktrees) - 1; index >= 0; index-- {
		if err := runGit(context.Background(), repositoryRoot, "worktree", "remove", "--force", worktrees[index]); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	cleanupErr = errors.Join(cleanupErr, makeTreeOwnerWritable(base), os.RemoveAll(base))
	if cleanupErr != nil {
		cleanupErr = errors.Join(cleanupErr, runGit(context.Background(), repositoryRoot, "worktree", "prune", "--expire", "now"))
	}
	return cleanupErr
}

func makeTreeOwnerWritable(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return os.Chmod(path, 0o600)
	})
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

func cleanupResolvedReplayWorktree(repositoryRoot, worktree, base string) error {
	removeErr := runGit(context.Background(), repositoryRoot, "worktree", "remove", "--force", worktree)
	writableErr := makeTreeOwnerWritable(base)
	directoryErr := os.RemoveAll(base)
	if removeErr != nil {
		pruneErr := runGit(context.Background(), repositoryRoot, "worktree", "prune", "--expire", "now")
		return errors.Join(removeErr, writableErr, directoryErr, pruneErr)
	}
	return errors.Join(writableErr, directoryErr)
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
	if err := verifyCandidateConstantsState(ctx, state, FrozenTrainingReportCommit); err != nil {
		return nil, err
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
	if err != nil || !bytes.Equal(proofBytes, mustCanonical(proof)) || proof.ProofVersion != AttemptProofVersion || proof.Panel != PanelValidation {
		return nil, errors.New("validation attempt proof is not canonical")
	}
	manifest := causalv2.PreregisteredManifest()
	if !validAttemptProtocolIdentity(attempt, proof, PanelValidation) || attempt.State != "published" || attempt.ExecutableCommit != state.Head || proof.PublishedDigest != validation.ReportDigest || len(proof.GeneratedFixtures) != manifest.ValidationSeeds.Count {
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

func validAttemptProtocolIdentity(attempt AttemptRecord, proof AttemptProofRecord, panel Panel) bool {
	return attempt.AttemptVersion == AttemptVersion && attempt.PlanCommit == PlanCommit && attempt.Panel == panel && proof.ProofVersion == AttemptProofVersion && proof.Panel == panel
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
