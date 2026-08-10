// Package actionrelationrun owns the operational boundary between ordinary
// prerequisite construction and the protected action-relations panels.
package actionrelationrun

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/chazu/nous/internal/actionrelationcompetence"
	"github.com/chazu/nous/internal/actionrelationexp"
)

const competenceRootDocument = "docs/actionrelations-competence-root.json"

var competenceEnvironment = []actionrelationexp.EnvironmentRow{
	{Key: "GOMAXPROCS", Value: "1"},
	{Key: "LC_ALL", Value: "C"},
	{Key: "PATH", Value: "/opt/homebrew/bin:/usr/bin:/bin"},
	{Key: "TZ", Value: "UTC"},
}

// PreparePrerequisites builds from the accepted implementation, writes build
// authority, and invokes that exact binary for exhaustive competence.
func PreparePrerequisites(ctx context.Context, repoRoot string) (actionrelationcompetence.Root, error) {
	build, err := actionrelationexp.PrepareBuildAuthority(ctx, repoRoot)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	if err := writeExclusiveAuthority(filepath.Join(root, filepath.FromSlash(actionrelationexp.BuildAuthorityPath)), build.Canonical); err != nil {
		return actionrelationcompetence.Root{}, fmt.Errorf("persist build authority: %w", err)
	}
	binary := filepath.Join(root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	argv := []string{binary, "actionrelation-trials", "-stage", "competence", "-repo-root", root}
	command := exec.CommandContext(ctx, argv[0], argv[1:]...)
	command.Dir, command.Env = root, environmentStrings(competenceEnvironment)
	output, err := command.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return actionrelationcompetence.Root{}, fmt.Errorf("competence subprocess: %s", strings.TrimSpace(string(exit.Stderr)))
		}
		return actionrelationcompetence.Root{}, err
	}
	encoded, err := os.ReadFile(filepath.Join(root, competenceRootDocument))
	if err != nil || !bytes.Equal(bytes.TrimSpace(output), encoded) {
		return actionrelationcompetence.Root{}, fmt.Errorf("competence subprocess output does not equal retained root")
	}
	return LoadCompetenceRoot(root, encoded, build)
}

// ExecuteCompetence is reachable only through the exact reviewed-binary argv
// and three-variable environment minted by PreparePrerequisites.
func ExecuteCompetence(repoRoot string, argv []string) (actionrelationcompetence.Root, error) {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	binary := filepath.Join(root, filepath.FromSlash(actionrelationexp.PanelBinaryPath))
	wantArgv := []string{binary, "actionrelation-trials", "-stage", "competence", "-repo-root", root}
	if !slices.Equal(argv, wantArgv) || !exactProcessEnvironment(competenceEnvironment) {
		return actionrelationcompetence.Root{}, fmt.Errorf("noncanonical competence invocation")
	}
	buildBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(actionrelationexp.BuildAuthorityPath)))
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	build, err := actionrelationexp.ParseBuildAuthority(buildBytes)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	if err := verifyCompetenceBuild(root, build, binary); err != nil {
		return actionrelationcompetence.Root{}, err
	}
	_, evidence, err := actionrelationcompetence.RunEvidence()
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	caseRef, err := actionrelationexp.Reference(".nous/actionrelations-v1-competence-evidence/cases-root.json", evidence.CaseManifest.Canonical)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	resultRef, err := actionrelationexp.Reference(".nous/actionrelations-v1-competence-evidence/results-root.json", evidence.ResultManifest.Canonical)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	buildRef, err := actionrelationexp.Reference(actionrelationexp.BuildAuthorityPath, build.Canonical)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	value, err := actionrelationcompetence.BuildRoot(actionrelationcompetence.Root{
		SourceRoot: build.SourceRoot, BinaryDigest: build.BinaryDigest, BuildAuthority: buildRef,
		CommandArgv: slices.Clone(argv), Environment: slices.Clone(competenceEnvironment), Evidence: evidence,
		CaseManifestRef: caseRef, ResultManifestRef: resultRef,
	})
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	if err := persistCompetenceEvidence(root, evidence); err != nil {
		return actionrelationcompetence.Root{}, err
	}
	if err := writeExclusiveAuthority(filepath.Join(root, competenceRootDocument), value.Canonical); err != nil {
		return actionrelationcompetence.Root{}, err
	}
	return LoadCompetenceRoot(root, value.Canonical, build)
}

func verifyCompetenceBuild(root string, build actionrelationexp.BuildAuthority, binary string) error {
	if err := actionrelationexp.VerifySourceCheckout(root, build.SourceRows); err != nil {
		return err
	}
	binaryBytes, err := os.ReadFile(binary)
	if err != nil || digest(binaryBytes) != build.BinaryDigest {
		return fmt.Errorf("competence binary differs from build authority")
	}
	goBytes, err := os.ReadFile(build.GoExecutablePath)
	if err != nil || digest(goBytes) != build.GoExecutableDigest {
		return fmt.Errorf("competence Go executable differs from build authority")
	}
	miseBytes, err := os.ReadFile(filepath.Join(root, "mise.toml"))
	if err != nil || digest(miseBytes) != build.MiseTomlDigest || string(miseBytes) != actionrelationexp.CanonicalMiseToml {
		return fmt.Errorf("competence mise authority differs from build authority")
	}
	command := exec.Command(build.GoExecutablePath, "version", "-m", binary)
	command.Dir, command.Env = root, []string{"GOENV=off", "GOWORK=off", "LC_ALL=C", "TZ=UTC"}
	versionM, err := command.Output()
	if err != nil || digest(versionM) != build.GoVersionMDigest {
		return fmt.Errorf("competence binary module identity differs from build authority")
	}
	return nil
}

func persistCompetenceEvidence(root string, evidence actionrelationcompetence.Evidence) error {
	directory := filepath.Join(root, ".nous", "actionrelations-v1-competence-evidence")
	if _, err := os.Lstat(directory); err == nil {
		return fmt.Errorf("competence evidence namespace already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Mkdir(directory, 0o755); err != nil {
		return err
	}
	files := make([]actionrelationcompetence.EvidenceFile, 0, len(evidence.CaseFiles)+len(evidence.ResultFiles)+2)
	files = append(files, evidence.CaseFiles...)
	files = append(files, evidence.ResultFiles...)
	files = append(files,
		actionrelationcompetence.EvidenceFile{Path: ".nous/actionrelations-v1-competence-evidence/cases-root.json", Mode: "100644", Data: evidence.CaseManifest.Canonical},
		actionrelationcompetence.EvidenceFile{Path: ".nous/actionrelations-v1-competence-evidence/results-root.json", Mode: "100644", Data: evidence.ResultManifest.Canonical},
	)
	total := 0
	for _, file := range files {
		if file.Mode != "100644" || filepath.Dir(file.Path) != ".nous/actionrelations-v1-competence-evidence" {
			return fmt.Errorf("invalid competence evidence path")
		}
		total += len(file.Data)
		if total > 65*1024*1024 {
			return fmt.Errorf("competence evidence exceeds reservation")
		}
		path := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := writeExclusiveSynced(path, file.Data); err != nil {
			return err
		}
	}
	if err := syncDirectory(directory); err != nil {
		return err
	}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Path)))
		if err != nil || !bytes.Equal(data, file.Data) {
			return fmt.Errorf("competence evidence readback mismatch: %s", file.Path)
		}
	}
	return nil
}

func LoadCompetenceRoot(repoRoot string, data []byte, build actionrelationexp.BuildAuthority) (actionrelationcompetence.Root, error) {
	directory := filepath.Join(repoRoot, ".nous", "actionrelations-v1-competence-evidence")
	entries, err := os.ReadDir(directory)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	files := map[string][]byte{}
	var casesManifest, resultsManifest []byte
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 {
			return actionrelationcompetence.Root{}, fmt.Errorf("invalid retained competence path: %s", entry.Name())
		}
		encoded, readErr := os.ReadFile(path)
		if readErr != nil {
			return actionrelationcompetence.Root{}, readErr
		}
		switch entry.Name() {
		case "cases-root.json":
			casesManifest = encoded
		case "results-root.json":
			resultsManifest = encoded
		default:
			relative := ".nous/actionrelations-v1-competence-evidence/" + entry.Name()
			files[relative] = encoded
		}
	}
	evidence, err := actionrelationcompetence.ParseEvidence(casesManifest, resultsManifest, files)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	value, err := actionrelationcompetence.ParseRoot(data, evidence)
	if err != nil {
		return actionrelationcompetence.Root{}, err
	}
	buildRef, _ := actionrelationexp.Reference(actionrelationexp.BuildAuthorityPath, build.Canonical)
	if value.SourceRoot != build.SourceRoot || value.BinaryDigest != build.BinaryDigest || value.BuildAuthority != buildRef || !slices.Equal(value.Environment, competenceEnvironment) {
		return actionrelationcompetence.Root{}, fmt.Errorf("competence root differs from build execution authority")
	}
	return value, nil
}

func writeExclusiveAuthority(path string, data []byte) error {
	if info, err := os.Lstat(path); err == nil {
		existing, readErr := os.ReadFile(path)
		if readErr == nil && info.Mode().IsRegular() && info.Mode().Perm() == 0o644 && bytes.Equal(existing, data) {
			return nil
		}
		return fmt.Errorf("authority path already exists with different bytes: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := writeExclusiveSynced(path, data); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func writeExclusiveSynced(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			// Preserve a partial exclusive authority file as evidence of failure.
			_ = file.Sync()
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func exactProcessEnvironment(rows []actionrelationexp.EnvironmentRow) bool {
	want := environmentStrings(rows)
	have := os.Environ()
	slices.Sort(have)
	return slices.Equal(have, want)
}

func environmentStrings(rows []actionrelationexp.EnvironmentRow) []string {
	result := make([]string, len(rows))
	for index, row := range rows {
		result[index] = row.Key + "=" + row.Value
	}
	return result
}

func digest(data []byte) string {
	ref, err := actionrelationexp.Reference("digest", data)
	if err != nil {
		return ""
	}
	return ref.Digest
}
