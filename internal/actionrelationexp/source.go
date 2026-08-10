package actionrelationexp

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// CollectSourceRows retains every tracked source, test, toolchain, lane-plan,
// and umbrella-plan blob from the accepted implementation commit. It never
// substitutes working-tree bytes for reviewed bytes.
func CollectSourceRows(repoRoot, implementationCommit string) ([]SourceRow, error) {
	if !commitText(implementationCommit) {
		return nil, fmt.Errorf("invalid implementation commit")
	}
	root, gitPath, gitDir, commonDir, err := reviewRepository(repoRoot)
	if err != nil {
		return nil, err
	}
	if err := reviewArchivePreflight(root, gitPath, gitDir, commonDir, implementationCommit); err != nil {
		return nil, err
	}
	tree, err := reviewGit(root, gitPath, true, "ls-tree", "-r", "-z", "--long", implementationCommit)
	if err != nil {
		return nil, fmt.Errorf("list implementation sources: %w", err)
	}
	var rows []SourceRow
	for _, entry := range bytes.Split(tree, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		metadata, rawPath, ok := bytes.Cut(entry, []byte{'\t'})
		fields := bytes.Fields(metadata)
		if !ok || len(fields) != 4 {
			return nil, fmt.Errorf("invalid implementation tree entry")
		}
		path := string(rawPath)
		role := ActionRelationSourceRole(path)
		if role == "" {
			continue
		}
		if string(fields[1]) != "blob" {
			return nil, fmt.Errorf("source is not a blob: %s", path)
		}
		size, sizeErr := strconv.ParseInt(string(fields[3]), 10, 64)
		if sizeErr != nil || size < 0 {
			return nil, fmt.Errorf("invalid source size: %s", path)
		}
		data, readErr := reviewGit(root, gitPath, true, "cat-file", "blob", string(fields[2]))
		if readErr != nil || int64(len(data)) != size {
			return nil, fmt.Errorf("read implementation source %s", path)
		}
		rows = append(rows, SourceRow{Path: path, GitMode: string(fields[0]), GitBlobOID: string(fields[2]), ByteLength: size, Digest: shaHex(data), Role: role})
	}
	if _, err := sourceRowWires(rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func ActionRelationSourceRole(path string) string {
	switch path {
	case "docs/guarded-action-relations-vocabulary-plan.md":
		return "plan"
	case "docs/vocabulary-research-program-v3.md", "docs/vocabulary-research-roadmap.md":
		return "umbrella"
	case "go.mod", "go.sum", "mise.toml":
		return "toolchain"
	}
	if strings.HasSuffix(path, "_test.go") {
		return "test"
	}
	if strings.HasSuffix(path, ".cue") {
		return "domain"
	}
	for _, extension := range []string{".go", ".c", ".h", ".s", ".S"} {
		if strings.HasSuffix(path, extension) {
			return "compiler-input"
		}
	}
	return ""
}

// VerifySourceCheckout proves that the build reads the reviewed source set and
// that no tracked, untracked, or ignored compiler input has been introduced.
func VerifySourceCheckout(repoRoot string, rows []SourceRow) error {
	root, gitPath, _, _, err := reviewRepository(repoRoot)
	if err != nil {
		return err
	}
	if _, err := sourceRowWires(rows); err != nil {
		return err
	}
	want := make(map[string]SourceRow, len(rows))
	for _, row := range rows {
		want[row.Path] = row
		pathList, listErr := reviewGit(root, gitPath, true, "ls-files", "-z", "--", row.Path)
		if listErr != nil || !bytes.Equal(pathList, append([]byte(row.Path), 0)) {
			return fmt.Errorf("source is not uniquely tracked: %s", row.Path)
		}
		physical := filepath.Join(root, filepath.FromSlash(row.Path))
		info, statErr := os.Lstat(physical)
		if statErr != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("source is not a regular file: %s", row.Path)
		}
		resolved, resolveErr := filepath.EvalSymlinks(physical)
		if resolveErr != nil || resolved != physical {
			return fmt.Errorf("source path traverses a symlink: %s", row.Path)
		}
		data, readErr := os.ReadFile(physical)
		if readErr != nil || int64(len(data)) != row.ByteLength || shaHex(data) != row.Digest {
			return fmt.Errorf("source differs from reviewed blob: %s", row.Path)
		}
		mode := "100644"
		if info.Mode()&0o111 != 0 {
			mode = "100755"
		}
		if mode != row.GitMode {
			return fmt.Errorf("source mode differs from reviewed blob: %s", row.Path)
		}
	}
	tracked, err := reviewGit(root, gitPath, true, "ls-files", "-z")
	if err != nil {
		return err
	}
	for _, rawPath := range bytes.Split(tracked, []byte{0}) {
		path := string(rawPath)
		if role := ActionRelationSourceRole(path); role != "" {
			row, present := want[path]
			if !present || row.Role != role {
				return fmt.Errorf("reviewed source set omits tracked input: %s", path)
			}
		}
	}
	// Excluding declared evidence namespaces prevents a pre-existing retained
	// panel from turning a bounded preflight into a walk over millions of leaves.
	pathspec := []string{".", ":(exclude).nous/**", ":(exclude).maki/**", ":(exclude)runs/**", ":(exclude)nous", ":(exclude)go.work", ":(exclude)go.work.sum"}
	for _, ignored := range []bool{false, true} {
		args := []string{"ls-files", "--others", "-z", "--exclude-standard"}
		if ignored {
			args = append(args, "--ignored")
		}
		args = append(args, "--")
		args = append(args, pathspec...)
		others, listErr := reviewGit(root, gitPath, true, args...)
		if listErr != nil {
			return listErr
		}
		for _, rawPath := range bytes.Split(others, []byte{0}) {
			path := string(rawPath)
			if path != "" && ActionRelationSourceRole(path) != "" {
				return fmt.Errorf("untracked compiler input is forbidden: %s", path)
			}
		}
	}
	return nil
}

var standardNonInputPaths = []string{
	".git/hooks",
	".maki",
	".nous/actionrelations-v1-competence-evidence",
	".nous/actionrelations-v1-development-evidence",
	".nous/actionrelations-v1-development-report.json",
	".nous/actionrelations-v1-development-terminal-receipt.json",
	".nous/actionrelations-v1-locked-claim.json",
	".nous/actionrelations-v1-locked-evidence",
	".nous/actionrelations-v1-locked-report.json",
	".nous/actionrelations-v1-locked-running.json",
	".nous/actionrelations-v1-locked-terminal-receipt.json",
	".nous/actionrelations-v1-validation-claim.json",
	".nous/actionrelations-v1-validation-evidence",
	".nous/actionrelations-v1-validation-report.json",
	".nous/actionrelations-v1-validation-running.json",
	".nous/actionrelations-v1-validation-terminal-receipt.json",
	".nous/bin",
	"go.work",
	"go.work.sum",
	"nous",
	"runs",
}

func SnapshotNonInputs(repoRoot string) ([]NonInputRow, error) {
	root, _, gitDir, _, err := reviewRepository(repoRoot)
	if err != nil {
		return nil, err
	}
	paths := slices.Clone(standardNonInputPaths)
	entries, readErr := os.ReadDir(filepath.Join(root, ".nous"))
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return nil, readErr
	}
	for _, entry := range entries {
		path := ".nous/" + entry.Name()
		if !slices.Contains(paths, path) {
			if ActionRelationSourceRole(path) != "" {
				return nil, fmt.Errorf("compiler input occupies unread namespace: %s", path)
			}
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	rows := make([]NonInputRow, len(paths))
	for index, path := range paths {
		physical := filepath.Join(root, filepath.FromSlash(path))
		if path == ".git/hooks" {
			physical = filepath.Join(gitDir, "hooks")
		}
		status := "present-not-read"
		info, statErr := os.Lstat(physical)
		if errors.Is(statErr, os.ErrNotExist) {
			status = "absent"
		} else if statErr != nil {
			return nil, statErr
		} else if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("non-input path is a symlink: %s", path)
		}
		rows[index] = NonInputRow{Path: path, Status: status}
	}
	if _, err := nonInputRowWires(rows); err != nil {
		return nil, err
	}
	return rows, nil
}
