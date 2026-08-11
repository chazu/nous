package actionrelationrun

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/chazu/nous/internal/actionrelationcompetence"
	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationscore"
	"golang.org/x/sys/unix"
)

const isolatedStage = "isolated-policy-worker"

type isolatedPanelResult struct {
	summary actionrelationscore.PanelSummary
	policy  actionrelationscore.PolicyPanelSummary
	writer  *panelWriter
	gates   actionrelationscore.MechanicalGates
}

func executeIsolatedPair(ctx context.Context, prerequisites panelPrerequisites, sealed actionrelationscore.SealedPanel, capBytes int64) (isolatedPanelResult, error) {
	if sealed.Panel() != "development" && sealed.Panel() != "validation" && sealed.Panel() != "locked" {
		return isolatedPanelResult{}, fmt.Errorf("invalid sealed panel")
	}
	scratchParent := filepath.Join(prerequisites.Root, ".nous")
	if err := checkDirectoryNoFollow(scratchParent); err != nil {
		return isolatedPanelResult{}, fmt.Errorf("unsafe panel scratch namespace: %w", err)
	}
	inputRoot, binary, fixturePath, scorerPath, err := prepareIsolatedInputs(prerequisites, sealed)
	if err != nil {
		return isolatedPanelResult{}, err
	}
	defer removeIsolatedInputs(inputRoot)
	primaryRoot, err := os.MkdirTemp(scratchParent, ".actionrelation-primary-")
	if err != nil {
		return isolatedPanelResult{}, err
	}
	defer os.RemoveAll(primaryRoot)
	auditRoot, err := os.MkdirTemp(scratchParent, ".actionrelation-audit-")
	if err != nil {
		return isolatedPanelResult{}, err
	}
	defer os.RemoveAll(auditRoot)
	primary, err := runIsolatedWorker(ctx, prerequisites, sealed, binary, inputRoot, "primary", fixturePath, primaryRoot, capBytes)
	if err != nil {
		return isolatedPanelResult{}, err
	}
	audit, err := runIsolatedWorker(ctx, prerequisites, sealed, binary, inputRoot, "audit", fixturePath, auditRoot, capBytes)
	if err != nil {
		return isolatedPanelResult{}, err
	}
	if !reflectPolicyPanelSummaries(primary.policy, audit.policy) {
		return isolatedPanelResult{}, fmt.Errorf("primary and audit isolated policy summaries differ")
	}
	if err := comparePanelFiles(primary.writer, audit.writer); err != nil {
		return isolatedPanelResult{}, err
	}
	primarySealed, err := reopenIsolatedScorer(scorerPath, sealed.Digest())
	if err != nil {
		return isolatedPanelResult{}, err
	}
	primary.summary, err = finalizeIsolatedPolicyPanel(primarySealed, primary.policy, primary.writer)
	if err != nil {
		return isolatedPanelResult{}, fmt.Errorf("primary isolated scoring: %w", err)
	}
	auditSealed, err := reopenIsolatedScorer(scorerPath, sealed.Digest())
	if err != nil {
		return isolatedPanelResult{}, err
	}
	audit.summary, err = finalizeIsolatedPolicyPanel(auditSealed, audit.policy, audit.writer)
	if err != nil {
		return isolatedPanelResult{}, fmt.Errorf("audit isolated scoring: %w", err)
	}
	if !reflectPanelSummaries(primary.summary, audit.summary) {
		return isolatedPanelResult{}, fmt.Errorf("primary and audit isolated scorer summaries differ")
	}
	if err := comparePanelFiles(primary.writer, audit.writer); err != nil {
		return isolatedPanelResult{}, err
	}
	gates, err := deriveIsolatedGates(prerequisites, primary)
	if err != nil {
		return isolatedPanelResult{}, err
	}
	gates.PrimaryAuditEqual = true
	primary.gates = gates
	if err := promotePanelWriter(primary.writer, prerequisites.Root); err != nil {
		return isolatedPanelResult{}, err
	}
	return primary, nil
}

func prepareIsolatedInputs(prerequisites panelPrerequisites, sealed actionrelationscore.SealedPanel) (string, string, string, string, error) {
	root, err := os.MkdirTemp(filepath.Join(prerequisites.Root, ".nous"), ".actionrelation-inputs-")
	if err != nil {
		return "", "", "", "", err
	}
	fail := func(err error) (string, string, string, string, error) {
		removeIsolatedInputs(root)
		return "", "", "", "", err
	}
	binarySource := filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	binaryBytes, err := readRegularNoFollow(binarySource, 0o755)
	if err != nil || digestBytes(binaryBytes) != prerequisites.Build.BinaryDigest {
		return fail(fmt.Errorf("reviewed executable is not an exact no-follow regular file"))
	}
	binary := filepath.Join(root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	if err := writeExclusiveSyncedMode(binary, binaryBytes, 0o555); err != nil {
		return fail(err)
	}
	domainCount := 0
	for _, row := range prerequisites.Build.SourceRows {
		if row.Role != "domain" || !strings.HasPrefix(row.Path, "domains/actionrelations/") {
			continue
		}
		if row.GitMode != "100644" {
			return fail(fmt.Errorf("reviewed actionrelations domain has noncanonical mode"))
		}
		data, readErr := readRegularNoFollow(filepath.Join(prerequisites.Root, filepath.FromSlash(row.Path)), 0o644)
		if readErr != nil || int64(len(data)) != row.ByteLength || digestBytes(data) != row.Digest {
			return fail(fmt.Errorf("reviewed domain input changed: %s", row.Path))
		}
		if writeErr := writeExclusiveSyncedMode(filepath.Join(root, filepath.FromSlash(row.Path)), data, 0o444); writeErr != nil {
			return fail(writeErr)
		}
		domainCount++
	}
	if domainCount != 3 {
		return fail(fmt.Errorf("reviewed actionrelations domain snapshot is incomplete"))
	}
	fixturePath := filepath.Join(root, "public-panel.json")
	if err := writeExclusiveSyncedMode(fixturePath, sealed.Public().Canonical(), 0o444); err != nil {
		return fail(err)
	}
	scorerPath := filepath.Join(root, "private-scorer.json")
	if err := writeExclusiveSyncedMode(scorerPath, sealed.Canonical(), 0o400); err != nil {
		return fail(err)
	}
	for _, directory := range []string{filepath.Dir(binary), filepath.Join(root, "domains", "actionrelations"), filepath.Join(root, "domains"), filepath.Join(root, ".nous"), root} {
		if err := os.Chmod(directory, 0o555); err != nil {
			return fail(err)
		}
	}
	return root, binary, fixturePath, scorerPath, nil
}

func reopenIsolatedScorer(path, digest string) (actionrelationscore.SealedPanel, error) {
	data, err := readRegularNoFollow(path, 0o400)
	if err != nil {
		return actionrelationscore.SealedPanel{}, fmt.Errorf("reopen private scorer: %w", err)
	}
	sealed, err := actionrelationscore.ParseSealedPanel(bytes.NewReader(data), int64(len(data)), digest)
	if err != nil {
		return actionrelationscore.SealedPanel{}, fmt.Errorf("reconstruct private scorer: %w", err)
	}
	return sealed, nil
}

func finalizeIsolatedPolicyPanel(sealed actionrelationscore.SealedPanel, policy actionrelationscore.PolicyPanelSummary, writer *panelWriter) (actionrelationscore.PanelSummary, error) {
	return actionrelationscore.FinalizePolicyPanel(sealed, policy, writer.read, writer.writeAll)
}

func removeIsolatedInputs(root string) {
	if root == "" {
		return
	}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && entry.IsDir() {
			_ = os.Chmod(path, 0o700)
		}
		return nil
	})
	_ = os.RemoveAll(root)
}

func deriveIsolatedGates(prerequisites panelPrerequisites, result isolatedPanelResult) (actionrelationscore.MechanicalGates, error) {
	if err := actionrelationcompetence.VerifyRoot(prerequisites.Competence); err != nil {
		return actionrelationscore.MechanicalGates{}, fmt.Errorf("competence authority does not reconstruct: %w", err)
	}
	gates, err := actionrelationscore.DerivePanelMechanicalGates(result.summary)
	if err != nil {
		return actionrelationscore.MechanicalGates{}, err
	}
	read := func(path string) ([]byte, error) {
		retained, ok := result.writer.files[path]
		if !ok {
			return nil, fmt.Errorf("retained path is absent: %s", path)
		}
		data, err := readRegularNoFollow(retained.Path, 0o644)
		if err != nil || int64(len(data)) != retained.Bytes || digestBytes(data) != retained.Digest {
			return nil, fmt.Errorf("retained path changed during verification: %s", path)
		}
		return data, nil
	}
	runRef, err := actionrelationexp.Reference(result.summary.RunEvidenceManifest.Path, result.summary.RunEvidenceManifest.Data)
	if err != nil {
		return actionrelationscore.MechanicalGates{}, err
	}
	fixtureRef, err := actionrelationexp.Reference(actionrelationexp.ExpectedAuthorityPath(result.summary.Panel, "fixture-root"), result.summary.Fixture.Canonical)
	if err != nil {
		return actionrelationscore.MechanicalGates{}, err
	}
	reachable, err := actionrelationexp.VerifyRetainedPacks(actionrelationexp.RetainedPackRefs{
		Panel: result.summary.Panel, Authority: result.summary.Authority, Fixture: fixtureRef, RunEvidence: runRef,
		ObjectRoots: result.summary.ObjectRoots, IndexRoots: result.summary.IndexRoots,
		JournalRoots: result.summary.JournalRoots, InputRoots: result.summary.InputRoots, DetailRoots: result.summary.DetailRoots,
		Tables: result.summary.Tables, StructuralMaps: result.summary.StructuralMaps, StoreBoundaries: result.summary.StoreBoundaries,
	}, read)
	if err != nil {
		return actionrelationscore.MechanicalGates{}, fmt.Errorf("retained evidence DAG: %w", err)
	}
	slices.Sort(reachable)
	if !slices.Equal(reachable, mapKeys(result.writer.files)) {
		return actionrelationscore.MechanicalGates{}, fmt.Errorf("retained evidence contains unreachable files")
	}
	gates.AuthorityClosure = true
	return gates, nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func runIsolatedWorker(ctx context.Context, prerequisites panelPrerequisites, sealed actionrelationscore.SealedPanel, binary, inputRoot, role, fixturePath, outputRoot string, capBytes int64) (isolatedPanelResult, error) {
	argv := isolatedWorkerArgv(binary, inputRoot, sealed.Panel(), role, fixturePath, sealed.Public().Digest(), outputRoot, capBytes)
	profile := isolatedSandboxProfile(binary, inputRoot, outputRoot)
	command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", append([]string{"-p", profile}, argv...)...)
	command.Dir = inputRoot
	command.Env = environmentStrings(competenceEnvironment)
	command.ExtraFiles = nil
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.Output()
	if err != nil {
		return isolatedPanelResult{}, fmt.Errorf("%s isolated worker failed: %w: %s", role, err, strings.TrimSpace(stderr.String()))
	}
	canonical := bytes.TrimSuffix(stdout, []byte{'\n'})
	summary, err := actionrelationscore.ParsePolicyPanelSummary(bytes.NewReader(canonical), int64(len(canonical)))
	if err != nil || actionrelationscore.VerifyPolicyPanelSummaryForPublic(summary, sealed.Public()) != nil {
		return isolatedPanelResult{}, fmt.Errorf("%s isolated worker returned invalid summary", role)
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(sealed.Panel())
	writer, err := loadPanelWriter(outputRoot, evidenceRoot, capBytes)
	if err != nil {
		return isolatedPanelResult{}, fmt.Errorf("%s isolated evidence: %w", role, err)
	}
	return isolatedPanelResult{policy: summary, writer: writer}, nil
}

func ExecuteIsolatedPolicyWorker(repoRoot, panel, role, publicPath, publicDigest, outputRoot string, capBytes int64, argv []string) ([]byte, error) {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	publicPath, err = filepath.EvalSymlinks(publicPath)
	if err != nil {
		return nil, err
	}
	outputRoot, err = filepath.EvalSymlinks(outputRoot)
	if err != nil {
		return nil, err
	}
	binary := filepath.Join(root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	want := isolatedWorkerArgv(binary, root, panel, role, publicPath, publicDigest, outputRoot, capBytes)
	wantCap := map[string]int64{"development": developmentEvidenceCap, "validation": validationEvidenceCap, "locked": lockedEvidenceCap}[panel]
	wantOutputPrefix := ".actionrelation-" + role + "-"
	executable, executableErr := os.Executable()
	executable, evalExecutableErr := filepath.EvalSymlinks(executable)
	if !slices.Equal(argv, want) || !exactProcessEnvironment(competenceEnvironment) || role != "primary" && role != "audit" || capBytes != wantCap || publicPath != filepath.Join(root, "public-panel.json") || !strings.HasPrefix(filepath.Base(outputRoot), wantOutputPrefix) || executableErr != nil || evalExecutableErr != nil || executable != binary {
		return nil, fmt.Errorf("noncanonical isolated policy-worker invocation")
	}
	publicInfo, err := os.Lstat(publicPath)
	if err != nil || !publicInfo.Mode().IsRegular() || publicInfo.Mode().Perm() != 0o444 || publicInfo.Size() < 1 {
		return nil, fmt.Errorf("invalid read-only public policy input")
	}
	outputInfo, err := os.Lstat(outputRoot)
	if err != nil || !outputInfo.IsDir() || outputInfo.Mode().Perm() != 0o700 {
		return nil, fmt.Errorf("invalid empty isolated output namespace")
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil || len(entries) != 0 {
		return nil, fmt.Errorf("isolated output namespace is not empty")
	}
	file, err := os.Open(publicPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	public, err := actionrelationscore.ParsePublicPanel(file, publicInfo.Size(), publicDigest)
	if err != nil || public.Panel() != panel {
		return nil, fmt.Errorf("isolated public policy authority does not reconstruct")
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(panel)
	writer := newPanelWriter(outputRoot, evidenceRoot, capBytes)
	consume := func(chunk actionrelationscore.PolicyPanelCurriculumEvidence) error {
		return writer.writeAll(append(slices.Clone(chunk.ManifestFiles), chunk.PackFiles...))
	}
	summary, err := actionrelationscore.ExecutePublicPanel(filepath.Join(root, "domains"), public, consume)
	if err != nil {
		return nil, err
	}
	if err := writer.write(summary.RunEvidenceManifest); err != nil {
		return nil, err
	}
	if err := writer.write(summary.RunEvidence.File); err != nil {
		return nil, err
	}
	return actionrelationscore.MarshalPolicyPanelSummary(summary)
}

func isolatedWorkerArgv(binary, repoRoot, panel, role, fixturePath, fixtureDigest, outputRoot string, capBytes int64) []string {
	return []string{
		binary, "actionrelation-trials", "-stage", isolatedStage, "-panel", panel, "-repo-root", repoRoot,
		"-worker-role", role, "-public-input", fixturePath, "-public-digest", fixtureDigest,
		"-output-root", outputRoot, "-evidence-cap", strconv.FormatInt(capBytes, 10),
	}
}

func isolatedSandboxProfile(binary, inputRoot, outputRoot string) string {
	var profile strings.Builder
	profile.WriteString("(version 1)\n(deny default)\n")
	profile.WriteString("(allow process-exec (literal ")
	profile.WriteString(strconv.Quote(binary))
	profile.WriteString("))\n(allow sysctl-read)\n")
	profile.WriteString("(allow file-read* (literal \"/\") (subpath \"/usr\") (subpath \"/System\") (subpath \"/private/var/db\") (literal \"/dev/null\") (literal ")
	profile.WriteString(strconv.Quote(binary))
	profile.WriteString(") (literal ")
	profile.WriteString(strconv.Quote(filepath.Join(inputRoot, "public-panel.json")))
	profile.WriteString(") (subpath ")
	profile.WriteString(strconv.Quote(filepath.Join(inputRoot, "domains", "actionrelations")))
	profile.WriteString("))\n")
	profile.WriteString("(allow file-read-metadata (subpath ")
	profile.WriteString(strconv.Quote(inputRoot))
	profile.WriteString("))\n")
	profile.WriteString("(allow file-write* (subpath ")
	profile.WriteString(strconv.Quote(outputRoot))
	profile.WriteString("))\n")
	return profile.String()
}

func loadPanelWriter(root, panelRoot string, capBytes int64) (*panelWriter, error) {
	writer := newPanelWriter(root, panelRoot, capBytes)
	physicalRoot := filepath.Join(root, filepath.FromSlash(panelRoot))
	err := filepath.WalkDir(physicalRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("isolated evidence contains symlink: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 {
			return fmt.Errorf("isolated evidence is not mode-100644 regular file: %s", path)
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); !ok || stat.Nlink != 1 {
			return fmt.Errorf("isolated evidence has nonexclusive inode: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		logical := filepath.ToSlash(relative)
		if !strings.HasPrefix(logical, panelRoot+"/") || filepath.Clean(logical) != logical || writer.files[logical].Path != "" {
			return fmt.Errorf("invalid isolated evidence path: %s", logical)
		}
		digest, err := digestFile(path)
		if err != nil {
			return err
		}
		writer.total += info.Size()
		if writer.total > capBytes {
			return fmt.Errorf("isolated panel evidence exceeds capacity")
		}
		writer.files[logical] = retainedFile{Path: path, Bytes: info.Size(), Digest: digest}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return writer, nil
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func promotePanelWriter(writer *panelWriter, repoRoot string) error {
	source := filepath.Join(writer.physicalRoot, filepath.FromSlash(writer.panelRoot))
	target := filepath.Join(repoRoot, filepath.FromSlash(writer.panelRoot))
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("panel evidence target already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	sourceParent, sourceLeaf, err := openParentNoFollow(source, false, 0)
	if err != nil {
		return err
	}
	defer unix.Close(sourceParent)
	targetParent, targetLeaf, err := openParentNoFollow(target, false, 0)
	if err != nil {
		return err
	}
	defer unix.Close(targetParent)
	if err := unix.RenameatxNp(sourceParent, sourceLeaf, targetParent, targetLeaf, unix.RENAME_EXCL); err != nil {
		return fmt.Errorf("publish compared primary evidence: %w", err)
	}
	if err := unix.Fsync(sourceParent); err != nil {
		return err
	}
	if err := unix.Fsync(targetParent); err != nil {
		return err
	}
	writer.physicalRoot = repoRoot
	for logical, retained := range writer.files {
		retained.Path = filepath.Join(repoRoot, filepath.FromSlash(logical))
		writer.files[logical] = retained
	}
	return nil
}

func reflectPanelSummaries(left, right actionrelationscore.PanelSummary) bool {
	leftBytes, leftErr := actionrelationscore.MarshalPanelSummary(left)
	rightBytes, rightErr := actionrelationscore.MarshalPanelSummary(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func reflectPolicyPanelSummaries(left, right actionrelationscore.PolicyPanelSummary) bool {
	leftBytes, leftErr := actionrelationscore.MarshalPolicyPanelSummary(left)
	rightBytes, rightErr := actionrelationscore.MarshalPolicyPanelSummary(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}
