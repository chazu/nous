package actionrelationexp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const CanonicalMiseToml = "[tools]\ngo = \"1.25.12\"\n"

// PrepareBuildAuthority performs the ordinary, pre-panel reproducible build.
// It requires committed unanimous implementation reviews but does not write
// authority documents or enter any protected panel constructor.
func PrepareBuildAuthority(ctx context.Context, repoRoot string) (BuildAuthority, error) {
	root, gitPath, gitDir, commonDir, err := reviewRepository(repoRoot)
	if err != nil {
		return BuildAuthority{}, err
	}
	head, err := reviewGitText(root, gitPath, "rev-parse", "HEAD")
	if err != nil || !commitText(head) {
		return BuildAuthority{}, fmt.Errorf("invalid build HEAD")
	}
	if err := reviewArchivePreflight(root, gitPath, gitDir, commonDir, head); err != nil {
		return BuildAuthority{}, err
	}
	if err := verifyBuildGitState(gitDir); err != nil {
		return BuildAuthority{}, err
	}
	planReview, err := LoadCommittedReview(root, head, "plan")
	if err != nil {
		return BuildAuthority{}, err
	}
	implementationReview, err := LoadCommittedReview(root, head, "implementation")
	if err != nil {
		return BuildAuthority{}, err
	}
	if err := VerifyReviewArchive(root, planReview); err != nil {
		return BuildAuthority{}, fmt.Errorf("plan review archive: %w", err)
	}
	if err := VerifyReviewArchive(root, implementationReview); err != nil {
		return BuildAuthority{}, fmt.Errorf("implementation review archive: %w", err)
	}
	ancestor := exec.CommandContext(ctx, gitPath, "merge-base", "--is-ancestor", implementationReview.ReviewedCommit, head)
	ancestor.Dir, ancestor.Env = root, reviewGitEnvironment(true)
	if err := ancestor.Run(); err != nil {
		return BuildAuthority{}, fmt.Errorf("reviewed implementation is not an ancestor of build HEAD")
	}
	rows, err := CollectSourceRows(root, implementationReview.ReviewedCommit)
	if err != nil {
		return BuildAuthority{}, err
	}
	if err := VerifySourceCheckout(root, rows); err != nil {
		return BuildAuthority{}, err
	}
	sourceRoot, err := BuildSourceRoot(implementationReview.ReviewedCommit, rows)
	if err != nil {
		return BuildAuthority{}, err
	}
	miseBytes, err := os.ReadFile(filepath.Join(root, "mise.toml"))
	if err != nil || string(miseBytes) != CanonicalMiseToml {
		return BuildAuthority{}, fmt.Errorf("mise.toml is not the frozen toolchain input")
	}
	goPath, goDigest, goVersion, goos, goarch, goPathValue, moduleCache, err := resolveBuildGo(ctx, root)
	if err != nil {
		return BuildAuthority{}, err
	}
	if err := prepareBuildOutput(root); err != nil {
		return BuildAuthority{}, err
	}
	temporary := filepath.Join(root, ".nous", ".actionrelations-v1-build-scratch")
	if _, err := os.Lstat(temporary); err == nil {
		return BuildAuthority{}, fmt.Errorf("build scratch namespace already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return BuildAuthority{}, err
	}
	if err := os.Mkdir(temporary, 0o700); err != nil {
		return BuildAuthority{}, err
	}
	cleaned := false
	defer func() {
		if !cleaned {
			_ = os.RemoveAll(temporary)
		}
	}()
	cache, temp := filepath.Join(temporary, "go-cache"), filepath.Join(temporary, "tmp")
	for _, directory := range []string{cache, temp} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return BuildAuthority{}, err
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return BuildAuthority{}, err
	}
	environment := []EnvironmentRow{
		{Key: "CGO_ENABLED", Value: "0"},
		{Key: "GOARCH", Value: goarch},
		{Key: "GOCACHE", Value: cache},
		{Key: "GOENV", Value: "off"},
		{Key: "GOFLAGS", Value: ""},
		{Key: "GOMODCACHE", Value: moduleCache},
		{Key: "GOOS", Value: goos},
		{Key: "GOPATH", Value: goPathValue},
		{Key: "GOPROXY", Value: "off"},
		{Key: "GOSUMDB", Value: "off"},
		{Key: "GOTOOLCHAIN", Value: "local"},
		{Key: "GOWORK", Value: "off"},
		{Key: "HOME", Value: home},
		{Key: "LC_ALL", Value: "C"},
		{Key: "TMPDIR", Value: temp},
		{Key: "TZ", Value: "UTC"},
	}
	argv := []string{goPath, "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-o", PanelBinaryPath, "./cmd/nous"}
	command := exec.CommandContext(ctx, argv[0], argv[1:]...)
	command.Dir, command.Env = root, environmentStrings(environment)
	if output, err := command.CombinedOutput(); err != nil {
		return BuildAuthority{}, fmt.Errorf("build panel binary: %w: %s", err, strings.TrimSpace(string(output)))
	}
	binary := filepath.Join(root, filepath.FromSlash(PanelBinaryPath))
	binaryBytes, err := os.ReadFile(binary)
	if err != nil {
		return BuildAuthority{}, err
	}
	binaryInfo, err := os.Lstat(binary)
	if err != nil || !binaryInfo.Mode().IsRegular() || binaryInfo.Mode()&0o111 == 0 {
		return BuildAuthority{}, fmt.Errorf("panel binary is not a regular executable")
	}
	versionCommand := exec.CommandContext(ctx, goPath, "version", "-m", binary)
	versionCommand.Dir, versionCommand.Env = root, environmentStrings(environment)
	versionM, err := versionCommand.Output()
	if err != nil {
		return BuildAuthority{}, fmt.Errorf("inspect panel binary: %w", err)
	}
	if err := os.RemoveAll(temporary); err != nil {
		return BuildAuthority{}, fmt.Errorf("remove owned build scratch: %w", err)
	}
	cleaned = true
	gitVersion, err := reviewGit(root, gitPath, true, "--version")
	if err != nil {
		return BuildAuthority{}, err
	}
	nonInputs, err := SnapshotNonInputs(root)
	if err != nil {
		return BuildAuthority{}, err
	}
	planRef, _ := Reference(ReviewManifestPath("plan"), planReview.Canonical)
	implementationRef, _ := Reference(ReviewManifestPath("implementation"), implementationReview.Canonical)
	return BuildBuildAuthority(BuildAuthority{
		PlanCommit: PlanCommit, PlanArchiveDigest: planReview.ArchiveDigest, PlanReview: planRef,
		ImplementationCommit: implementationReview.ReviewedCommit, ImplementationArchiveDigest: implementationReview.ArchiveDigest,
		ImplementationReview: implementationRef, BuildHead: head, SourceRoot: sourceRoot, SourceRows: rows,
		GitVersion: strings.TrimSpace(string(gitVersion)), GoVersion: goVersion, GoExecutablePath: goPath,
		GoExecutableDigest: goDigest, MiseTomlDigest: shaHex(miseBytes), BuildArgv: argv, BuildEnvironment: environment,
		GOOS: goos, GOARCH: goarch, CGOEnabled: "0", BinaryPath: PanelBinaryPath,
		BinaryDigest: shaHex(binaryBytes), GoVersionMDigest: shaHex(versionM), NonInputRows: nonInputs,
	})
}

func LoadCommittedReview(repoRoot, commit, kind string) (ReviewManifest, error) {
	if !commitText(commit) || ReviewManifestPath(kind) == "" {
		return ReviewManifest{}, fmt.Errorf("invalid committed review request")
	}
	root, gitPath, _, _, err := reviewRepository(repoRoot)
	if err != nil {
		return ReviewManifest{}, err
	}
	manifestPath := ReviewManifestPath(kind)
	data, err := committedBlob(root, gitPath, commit, manifestPath)
	if err != nil {
		return ReviewManifest{}, err
	}
	var top []json.RawMessage
	var rawRows [][]json.RawMessage
	if json.Unmarshal(data, &top) != nil || len(top) != 5 || json.Unmarshal(top[4], &rawRows) != nil || len(rawRows) != 3 {
		return ReviewManifest{}, fmt.Errorf("invalid committed review manifest")
	}
	verdicts := make(map[string][]byte, 3)
	paths := []string{manifestPath}
	for index, row := range rawRows {
		var path string
		if len(row) != 6 || json.Unmarshal(row[2], &path) != nil || !canonicalRelativePath(path) {
			return ReviewManifest{}, fmt.Errorf("invalid committed review row %d", index)
		}
		verdicts[path], err = committedBlob(root, gitPath, commit, path)
		if err != nil {
			return ReviewManifest{}, err
		}
		paths = append(paths, path)
	}
	manifest, err := ParseReviewManifest(data, verdicts)
	if err != nil {
		return ReviewManifest{}, err
	}
	for _, path := range paths {
		committed := data
		if path != manifestPath {
			committed = verdicts[path]
		}
		physical := filepath.Join(root, filepath.FromSlash(path))
		info, statErr := os.Lstat(physical)
		working, readErr := os.ReadFile(physical)
		if statErr != nil || readErr != nil || !info.Mode().IsRegular() || info.Mode()&0o111 != 0 || !bytes.Equal(working, committed) {
			return ReviewManifest{}, fmt.Errorf("working review authority differs from committed blob: %s", path)
		}
	}
	return manifest, nil
}

func committedBlob(root, gitPath, commit, path string) ([]byte, error) {
	if !commitText(commit) || !canonicalRelativePath(path) {
		return nil, fmt.Errorf("invalid committed blob authority")
	}
	data, err := reviewGit(root, gitPath, true, "cat-file", "blob", commit+":"+path)
	if err != nil {
		return nil, fmt.Errorf("read committed blob %s: %w", path, err)
	}
	return data, nil
}

func resolveBuildGo(ctx context.Context, root string) (path, digest, version, goos, goarch, goPathValue, moduleCache string, err error) {
	misePath, err := exec.LookPath("mise")
	if err != nil {
		return "", "", "", "", "", "", "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", "", "", "", "", err
	}
	command := exec.CommandContext(ctx, misePath, "--no-hooks", "--no-env", "-C", root, "which", "go")
	command.Env = []string{"HOME=" + home, "MISE_NO_HOOKS=1", "MISE_NO_ENV=1", "LC_ALL=C", "TZ=UTC"}
	output, err := command.Output()
	if err != nil {
		return "", "", "", "", "", "", "", fmt.Errorf("resolve mise Go: %w", err)
	}
	path = strings.TrimSpace(string(output))
	path, err = filepath.EvalSymlinks(path)
	if err != nil || !canonicalAbsolutePath(path) {
		return "", "", "", "", "", "", "", fmt.Errorf("mise Go path is not canonical")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", "", "", "", "", err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", "", "", "", "", "", "", fmt.Errorf("mise Go is not a regular executable")
	}
	digest = shaHex(encoded)
	baseEnv := []string{"GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off", "HOME=" + home, "LC_ALL=C", "TZ=UTC"}
	versionCommand := exec.CommandContext(ctx, path, "version")
	versionCommand.Dir, versionCommand.Env = root, baseEnv
	versionBytes, err := versionCommand.Output()
	if err != nil {
		return "", "", "", "", "", "", "", err
	}
	version = strings.TrimSpace(string(versionBytes))
	if !strings.Contains(version, "go1.25.12") {
		return "", "", "", "", "", "", "", fmt.Errorf("mise Go does not match frozen 1.25.12 toolchain")
	}
	envCommand := exec.CommandContext(ctx, path, "env", "GOOS", "GOARCH", "GOPATH", "GOMODCACHE")
	envCommand.Dir, envCommand.Env = root, baseEnv
	envBytes, err := envCommand.Output()
	if err != nil {
		return "", "", "", "", "", "", "", err
	}
	lines := strings.Split(strings.TrimSpace(string(envBytes)), "\n")
	if len(lines) != 4 || !safeToken(lines[0]) || !safeToken(lines[1]) || !canonicalAbsolutePath(lines[2]) || !canonicalAbsolutePath(lines[3]) {
		return "", "", "", "", "", "", "", fmt.Errorf("invalid Go environment authority")
	}
	return path, digest, version, lines[0], lines[1], lines[2], lines[3], nil
}

func verifyBuildGitState(gitDir string) error {
	for _, name := range []string{"BISECT_LOG", "CHERRY_PICK_HEAD", "MERGE_HEAD", "REVERT_HEAD", "rebase-apply", "rebase-merge", "sequencer"} {
		if _, err := os.Lstat(filepath.Join(gitDir, name)); err == nil {
			return fmt.Errorf("Git operation in progress: %s", name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	hooks := filepath.Join(gitDir, "hooks")
	entries, err := os.ReadDir(hooks)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sample") {
			return fmt.Errorf("active Git hook is forbidden: %s", entry.Name())
		}
	}
	return nil
}

func prepareBuildOutput(root string) error {
	for _, path := range []string{".nous", ".nous/bin"} {
		physical := filepath.Join(root, filepath.FromSlash(path))
		info, err := os.Lstat(physical)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(physical, 0o755); err != nil {
				return err
			}
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe build output directory: %s", path)
		}
	}
	binary := filepath.Join(root, filepath.FromSlash(PanelBinaryPath))
	if info, err := os.Lstat(binary); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("unsafe panel binary output")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func environmentStrings(rows []EnvironmentRow) []string {
	result := make([]string, len(rows))
	for index, row := range rows {
		result[index] = row.Key + "=" + row.Value
	}
	return result
}

func reviewGitEnvironment(suppressSystemAttributes bool) []string {
	result := []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "LC_ALL=C", "TZ=UTC"}
	if suppressSystemAttributes {
		result = append(result, "GIT_ATTR_NOSYSTEM=1")
	}
	return result
}
