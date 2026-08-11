package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type ReviewRow struct {
	Scope             string
	ReviewerTask      string
	VerdictPath       string
	VerdictDigest     string
	AttestationDigest string
}

type ReviewManifest struct {
	Kind           string
	Round          int
	ReviewedCommit string
	ArchiveDigest  string
	Rows           []ReviewRow
	Canonical      []byte
	Digest         string
}

func ParseReviewManifest(data []byte, verdicts map[string][]byte) (ReviewManifest, error) {
	var row []json.RawMessage
	if json.Unmarshal(data, &row) != nil || len(row) != 5 {
		return ReviewManifest{}, fmt.Errorf("invalid review manifest wire")
	}
	var version, commit, archive string
	var round int
	var rawRows [][]json.RawMessage
	if json.Unmarshal(row[0], &version) != nil || json.Unmarshal(row[1], &round) != nil || json.Unmarshal(row[2], &commit) != nil || json.Unmarshal(row[3], &archive) != nil || json.Unmarshal(row[4], &rawRows) != nil {
		return ReviewManifest{}, fmt.Errorf("invalid review manifest fields")
	}
	kind := ""
	switch version {
	case "actionrelation-plan-reviews/v1":
		kind = "plan"
	case "actionrelation-implementation-reviews/v1":
		kind = "implementation"
	default:
		return ReviewManifest{}, fmt.Errorf("unknown review manifest version")
	}
	manifest := ReviewManifest{Kind: kind, Round: round, ReviewedCommit: commit, ArchiveDigest: archive, Canonical: bytes.Clone(data), Digest: shaHex(data)}
	for _, raw := range rawRows {
		if len(raw) != 6 {
			return ReviewManifest{}, fmt.Errorf("invalid review row arity")
		}
		var scope, task, path, verdictDigest, status, attestation string
		if json.Unmarshal(raw[0], &scope) != nil || json.Unmarshal(raw[1], &task) != nil || json.Unmarshal(raw[2], &path) != nil || json.Unmarshal(raw[3], &verdictDigest) != nil || json.Unmarshal(raw[4], &status) != nil || json.Unmarshal(raw[5], &attestation) != nil || status != "accepted" {
			return ReviewManifest{}, fmt.Errorf("invalid review row fields")
		}
		manifest.Rows = append(manifest.Rows, ReviewRow{Scope: scope, ReviewerTask: task, VerdictPath: path, VerdictDigest: verdictDigest, AttestationDigest: attestation})
	}
	if err := VerifyReviewManifest(manifest, verdicts); err != nil {
		return ReviewManifest{}, err
	}
	return manifest, nil
}

func VerifyReviewManifest(value ReviewManifest, verdicts map[string][]byte) error {
	version := "actionrelation-" + value.Kind + "-reviews/v1"
	if value.Kind != "plan" && value.Kind != "implementation" || value.Round < 1 || !commitText(value.ReviewedCommit) || !digestText(value.ArchiveDigest) || len(value.Rows) != 3 || value.Digest != shaHex(value.Canonical) || len(value.Canonical) > 8192 {
		return fmt.Errorf("invalid review manifest authority")
	}
	wires := make([]any, len(value.Rows))
	scopes := []string{"architecture", "action-semantics", "experimental-validity"}
	for index, review := range value.Rows {
		path := fmt.Sprintf("docs/actionrelations-reviews/%s/round-%d/%s.txt", value.Kind, value.Round, scopes[index])
		if review.Scope != scopes[index] {
			return fmt.Errorf("invalid review row %d scope", index)
		}
		if review.ReviewerTask == "" || !isASCII(review.ReviewerTask) {
			return fmt.Errorf("invalid review row %d task", index)
		}
		if review.VerdictPath != path {
			return fmt.Errorf("invalid review row %d path", index)
		}
		if !digestText(review.VerdictDigest) || !digestText(review.AttestationDigest) {
			return fmt.Errorf("invalid review row %d digest", index)
		}
		if !bytes.Equal(verdicts[path], []byte("ACCEPTED")) || review.VerdictDigest != shaHex(verdicts[path]) {
			return fmt.Errorf("invalid review row %d verdict", index)
		}
		attestation, _ := json.Marshal([]any{"actionrelation-review-attestation/v1", value.Kind, value.Round, value.ReviewedCommit, value.ArchiveDigest, review.Scope, review.ReviewerTask, review.VerdictPath, review.VerdictDigest, "accepted"})
		if review.AttestationDigest != shaHex(attestation) {
			return fmt.Errorf("review row %d attestation mismatch: have %s want %s", index, review.AttestationDigest, shaHex(attestation))
		}
		wires[index] = []any{review.Scope, review.ReviewerTask, review.VerdictPath, review.VerdictDigest, "accepted", review.AttestationDigest}
	}
	want, _ := json.Marshal([]any{version, value.Round, value.ReviewedCommit, value.ArchiveDigest, wires})
	if !bytes.Equal(want, value.Canonical) {
		return fmt.Errorf("review manifest is not canonical")
	}
	return nil
}

func commitText(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func ReviewManifestPath(kind string) string {
	if kind == "plan" || kind == "implementation" {
		return "docs/actionrelations-" + kind + "-reviews.json"
	}
	return ""
}

// VerifyReviewArchive recomputes the exact source archive named by a review
// manifest after excluding every repository and attribute mechanism that can
// replace objects or change archive bytes outside the reviewed commit.
func VerifyReviewArchive(repoRoot string, manifest ReviewManifest) error {
	if !commitText(manifest.ReviewedCommit) || !digestText(manifest.ArchiveDigest) {
		return fmt.Errorf("invalid review archive authority")
	}
	root, gitPath, gitDir, commonDir, err := reviewRepository(repoRoot)
	if err != nil {
		return err
	}
	if err := reviewArchivePreflight(root, gitPath, gitDir, commonDir, manifest.ReviewedCommit); err != nil {
		return err
	}
	exact, err := reviewGit(root, gitPath, false, "-c", "core.attributesFile=/dev/null", "archive", "--format=tar", manifest.ReviewedCommit)
	if err != nil {
		return fmt.Errorf("archive reviewed commit: %w", err)
	}
	// System attributes are external authority. Proving that suppressing them
	// leaves the bytes unchanged catches an installed attributes file even when
	// its location is outside the repository.
	hermetic, err := reviewGit(root, gitPath, true, "-c", "core.attributesFile=/dev/null", "archive", "--format=tar", manifest.ReviewedCommit)
	if err != nil {
		return fmt.Errorf("archive reviewed commit without system attributes: %w", err)
	}
	if !bytes.Equal(exact, hermetic) {
		return fmt.Errorf("external attributes change review archive")
	}
	if shaHex(exact) != manifest.ArchiveDigest {
		return fmt.Errorf("review archive digest mismatch")
	}
	return nil
}

func reviewRepository(repoRoot string) (root, gitPath, gitDir, commonDir string, err error) {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", "", "", "", err
	}
	absRoot = filepath.Clean(absRoot)
	root, err = filepath.EvalSymlinks(absRoot)
	if err != nil || root != absRoot {
		return "", "", "", "", fmt.Errorf("review repository root is not a canonical physical path")
	}
	gitPath, err = exec.LookPath("git")
	if err != nil {
		return "", "", "", "", fmt.Errorf("resolve git: %w", err)
	}
	gitPath, err = filepath.Abs(gitPath)
	if err != nil {
		return "", "", "", "", err
	}
	top, err := reviewGitText(root, gitPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", "", "", err
	}
	top, err = filepath.EvalSymlinks(top)
	if err != nil || top != root {
		return "", "", "", "", fmt.Errorf("review archive is not at the canonical repository root")
	}
	gitDir, err = reviewGitText(root, gitPath, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", "", "", "", err
	}
	commonDir, err = reviewGitText(root, gitPath, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", "", "", "", err
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(root, commonDir)
	}
	gitDir, err = filepath.EvalSymlinks(filepath.Clean(gitDir))
	if err != nil {
		return "", "", "", "", err
	}
	commonDir, err = filepath.EvalSymlinks(filepath.Clean(commonDir))
	if err != nil {
		return "", "", "", "", err
	}
	return root, gitPath, gitDir, commonDir, nil
}

func reviewArchivePreflight(root, gitPath, gitDir, commonDir, commit string) error {
	for _, path := range uniquePaths(
		filepath.Join(gitDir, "info", "attributes"),
		filepath.Join(commonDir, "info", "attributes"),
		filepath.Join(gitDir, "info", "grafts"),
		filepath.Join(commonDir, "info", "grafts"),
		filepath.Join(gitDir, "shallow"),
		filepath.Join(commonDir, "shallow"),
		filepath.Join(commonDir, "objects", "info", "alternates"),
		filepath.Join(commonDir, "objects", "info", "http-alternates"),
		filepath.Join(commonDir, "refs", "replace"),
	) {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("forbidden review repository state: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	packed, err := os.ReadFile(filepath.Join(commonDir, "packed-refs"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if bytes.Contains(packed, []byte(" refs/replace/")) {
		return fmt.Errorf("packed replacement refs are forbidden")
	}
	config, err := reviewGit(root, gitPath, true, "config", "--local", "--null", "--list")
	if err != nil {
		return fmt.Errorf("inspect local git configuration: %w", err)
	}
	for _, field := range bytes.Split(config, []byte{0}) {
		name, _, _ := bytes.Cut(field, []byte{'\n'})
		key := strings.ToLower(string(name))
		if strings.HasPrefix(key, "tar.") || strings.HasPrefix(key, "include.") || strings.HasPrefix(key, "includeif.") || strings.HasPrefix(key, "filter.") || key == "core.attributesfile" || key == "core.alternaterefscommand" || key == "core.fsmonitor" || key == "core.hookspath" || key == "core.untrackedcache" || key == "extensions.worktreeconfig" {
			return fmt.Errorf("unsafe review git configuration: %s", key)
		}
	}
	typeName, err := reviewGitText(root, gitPath, "cat-file", "-t", commit)
	if err != nil || typeName != "commit" {
		return fmt.Errorf("reviewed object is not a commit")
	}
	tree, err := reviewGit(root, gitPath, true, "ls-tree", "-r", "-z", commit)
	if err != nil {
		return fmt.Errorf("inspect reviewed tree: %w", err)
	}
	for _, entry := range bytes.Split(tree, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		metadata, path, ok := bytes.Cut(entry, []byte{'\t'})
		fields := bytes.Fields(metadata)
		if !ok || len(fields) != 3 {
			return fmt.Errorf("invalid reviewed tree entry")
		}
		if bytes.Equal(fields[0], []byte("160000")) || bytes.Equal(fields[1], []byte("commit")) {
			return fmt.Errorf("submodule gitlink is forbidden: %s", path)
		}
		if filepath.Base(string(path)) != ".gitattributes" {
			continue
		}
		attributes, attrErr := reviewGit(root, gitPath, true, "cat-file", "blob", string(fields[2]))
		if attrErr != nil {
			return fmt.Errorf("read committed attributes %s: %w", path, attrErr)
		}
		for _, line := range bytes.Split(attributes, []byte{'\n'}) {
			line = bytes.TrimSpace(line)
			if len(line) == 0 || line[0] == '#' {
				continue
			}
			for _, token := range bytes.Fields(line) {
				name := bytes.TrimLeft(token, "-!")
				name, _, _ = bytes.Cut(name, []byte{'='})
				if bytes.Equal(name, []byte("export-ignore")) || bytes.Equal(name, []byte("export-subst")) {
					return fmt.Errorf("archive-changing attribute is forbidden: %s", path)
				}
			}
		}
	}
	return nil
}

func uniquePaths(values ...string) []string {
	slices.Sort(values)
	return slices.Compact(values)
}

func reviewGitText(root, gitPath string, args ...string) (string, error) {
	output, err := reviewGit(root, gitPath, true, args...)
	return strings.TrimSpace(string(output)), err
}

func reviewGit(root, gitPath string, suppressSystemAttributes bool, args ...string) ([]byte, error) {
	command := exec.Command(gitPath, args...)
	command.Dir = root
	command.Env = []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"LC_ALL=C",
		"TZ=UTC",
	}
	if suppressSystemAttributes {
		command.Env = append(command.Env, "GIT_ATTR_NOSYSTEM=1")
	}
	output, err := command.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, err
	}
	return output, nil
}
