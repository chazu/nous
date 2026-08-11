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
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationscore"
	"golang.org/x/sys/unix"
)

const isolatedStage = "isolated-policy-worker"

type isolatedPanelResult struct {
	summary actionrelationscore.PanelSummary
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
	fixtureRoot, err := os.MkdirTemp(scratchParent, ".actionrelation-fixture-")
	if err != nil {
		return isolatedPanelResult{}, err
	}
	defer os.RemoveAll(fixtureRoot)
	fixturePath := filepath.Join(fixtureRoot, "sealed-panel.json")
	if err := writeExclusiveSyncedMode(fixturePath, sealed.Canonical(), 0o444); err != nil {
		return isolatedPanelResult{}, err
	}
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
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(sealed.Panel())
	canonicalOutput := filepath.Join(prerequisites.Root, filepath.FromSlash(evidenceRoot))
	primary, err := runIsolatedWorker(ctx, prerequisites, sealed, "primary", fixturePath, primaryRoot, capBytes, []string{canonicalOutput})
	if err != nil {
		return isolatedPanelResult{}, err
	}
	audit, err := runIsolatedWorker(ctx, prerequisites, sealed, "audit", fixturePath, auditRoot, capBytes, []string{canonicalOutput, primaryRoot})
	if err != nil {
		return isolatedPanelResult{}, err
	}
	if !reflectPanelSummaries(primary.summary, audit.summary) {
		return isolatedPanelResult{}, fmt.Errorf("primary and audit isolated panel summaries differ")
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
	reachable, err := actionrelationexp.VerifyRetainedPacks(actionrelationexp.RetainedPackRefs{
		Panel: result.summary.Panel, Authority: result.summary.Authority, RunEvidence: runRef,
		ObjectRoots: result.summary.ObjectRoots, IndexRoots: result.summary.IndexRoots,
		JournalRoots: result.summary.JournalRoots, InputRoots: result.summary.InputRoots, DetailRoots: result.summary.DetailRoots,
		Tables: result.summary.Tables, StructuralMaps: result.summary.StructuralMaps, StoreBoundaries: result.summary.StoreBoundaries,
	}, read)
	if err != nil {
		return actionrelationscore.MechanicalGates{}, fmt.Errorf("retained evidence DAG: %w", err)
	}
	fixturePath := actionrelationexp.ExpectedAuthorityPath(result.summary.Panel, "fixture-root")
	fixtureBytes, err := read(fixturePath)
	if err != nil || !bytes.Equal(fixtureBytes, result.summary.Fixture.Canonical) {
		return actionrelationscore.MechanicalGates{}, fmt.Errorf("retained fixture authority changed")
	}
	reachable = append(reachable, fixturePath)
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

func runIsolatedWorker(ctx context.Context, prerequisites panelPrerequisites, sealed actionrelationscore.SealedPanel, role, fixturePath, outputRoot string, capBytes int64, deniedReadRoots []string) (isolatedPanelResult, error) {
	binary := filepath.Join(prerequisites.Root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	argv := isolatedWorkerArgv(binary, prerequisites.Root, sealed.Panel(), role, fixturePath, sealed.Digest(), outputRoot, capBytes)
	profile := isolatedSandboxProfile(binary, outputRoot, deniedReadRoots)
	command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", append([]string{"-p", profile}, argv...)...)
	command.Dir = prerequisites.Root
	command.Env = environmentStrings(competenceEnvironment)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.Output()
	if err != nil {
		return isolatedPanelResult{}, fmt.Errorf("%s isolated worker failed: %w: %s", role, err, strings.TrimSpace(stderr.String()))
	}
	canonical := bytes.TrimSuffix(stdout, []byte{'\n'})
	summary, err := actionrelationscore.ParsePanelSummary(bytes.NewReader(canonical), int64(len(canonical)))
	if err != nil || summary.Panel != sealed.Panel() || summary.Authority != sealed.Authority() || summary.Fixture.Digest != sealed.Fixture().Digest {
		return isolatedPanelResult{}, fmt.Errorf("%s isolated worker returned invalid summary", role)
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(sealed.Panel())
	writer, err := loadPanelWriter(outputRoot, evidenceRoot, capBytes)
	if err != nil {
		return isolatedPanelResult{}, fmt.Errorf("%s isolated evidence: %w", role, err)
	}
	return isolatedPanelResult{summary: summary, writer: writer}, nil
}

func ExecuteIsolatedPolicyWorker(repoRoot, panel, role, fixturePath, fixtureDigest, outputRoot string, capBytes int64, argv []string) ([]byte, error) {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	fixturePath, err = filepath.EvalSymlinks(fixturePath)
	if err != nil {
		return nil, err
	}
	outputRoot, err = filepath.EvalSymlinks(outputRoot)
	if err != nil {
		return nil, err
	}
	binary := filepath.Join(root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	want := isolatedWorkerArgv(binary, root, panel, role, fixturePath, fixtureDigest, outputRoot, capBytes)
	wantCap := map[string]int64{"development": developmentEvidenceCap, "validation": validationEvidenceCap, "locked": lockedEvidenceCap}[panel]
	wantFixtureParent := filepath.Join(root, ".nous", ".actionrelation-fixture-")
	wantOutputParent := filepath.Join(root, ".nous", ".actionrelation-"+role+"-")
	executable, executableErr := os.Executable()
	executable, evalExecutableErr := filepath.EvalSymlinks(executable)
	if !slices.Equal(argv, want) || !exactProcessEnvironment(competenceEnvironment) || role != "primary" && role != "audit" || capBytes != wantCap || !strings.HasPrefix(fixturePath, wantFixtureParent) || filepath.Base(fixturePath) != "sealed-panel.json" || !strings.HasPrefix(outputRoot, wantOutputParent) || executableErr != nil || evalExecutableErr != nil || executable != binary {
		return nil, fmt.Errorf("noncanonical isolated policy-worker invocation")
	}
	fixtureInfo, err := os.Lstat(fixturePath)
	if err != nil || !fixtureInfo.Mode().IsRegular() || fixtureInfo.Mode().Perm() != 0o444 || fixtureInfo.Size() < 1 {
		return nil, fmt.Errorf("invalid read-only sealed fixture input")
	}
	outputInfo, err := os.Lstat(outputRoot)
	if err != nil || !outputInfo.IsDir() || outputInfo.Mode().Perm() != 0o700 {
		return nil, fmt.Errorf("invalid empty isolated output namespace")
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil || len(entries) != 0 {
		return nil, fmt.Errorf("isolated output namespace is not empty")
	}
	file, err := os.Open(fixturePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	sealed, err := actionrelationscore.ParseSealedPanel(file, fixtureInfo.Size(), fixtureDigest)
	if err != nil || sealed.Panel() != panel {
		return nil, fmt.Errorf("isolated fixture authority does not reconstruct")
	}
	evidenceRoot, _ := actionrelationexp.EvidenceRoot(panel)
	writer := newPanelWriter(outputRoot, evidenceRoot, capBytes)
	fixtureAuthorityPath := actionrelationexp.ExpectedAuthorityPath(panel, "fixture-root")
	prepare := func(fixture actionrelationfixture.PanelFixture) error {
		return writer.write(actionrelationexp.EvidenceFile{Path: fixtureAuthorityPath, Mode: "100644", Data: fixture.Canonical})
	}
	consume := func(chunk actionrelationscore.PanelCurriculumEvidence) error {
		return writer.writeAll(append(slices.Clone(chunk.ManifestFiles), chunk.PackFiles...))
	}
	summary, err := actionrelationscore.ExecuteSealedPanel(filepath.Join(root, "domains"), sealed, prepare, consume)
	if err != nil {
		return nil, err
	}
	if err := writer.write(summary.RunEvidenceManifest); err != nil {
		return nil, err
	}
	if err := writer.write(summary.RunEvidence.File); err != nil {
		return nil, err
	}
	return actionrelationscore.MarshalPanelSummary(summary)
}

func isolatedWorkerArgv(binary, repoRoot, panel, role, fixturePath, fixtureDigest, outputRoot string, capBytes int64) []string {
	return []string{
		binary, "actionrelation-trials", "-stage", isolatedStage, "-panel", panel, "-repo-root", repoRoot,
		"-worker-role", role, "-fixture-input", fixturePath, "-fixture-digest", fixtureDigest,
		"-output-root", outputRoot, "-evidence-cap", strconv.FormatInt(capBytes, 10),
	}
}

func isolatedSandboxProfile(binary, outputRoot string, deniedReadRoots []string) string {
	var profile strings.Builder
	profile.WriteString("(version 1)\n(allow default)\n(deny network*)\n(deny process-exec)\n")
	profile.WriteString("(allow process-exec (literal ")
	profile.WriteString(strconv.Quote(binary))
	profile.WriteString("))\n(deny file-write*)\n")
	profile.WriteString("(allow file-write* (subpath ")
	profile.WriteString(strconv.Quote(outputRoot))
	profile.WriteString("))\n")
	for _, root := range deniedReadRoots {
		profile.WriteString("(deny file-read* (subpath ")
		profile.WriteString(strconv.Quote(root))
		profile.WriteString("))\n")
	}
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
