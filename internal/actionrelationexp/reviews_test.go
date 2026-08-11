package actionrelationexp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommittedPlanReviewManifestClosesExactVerdictBytes(t *testing.T) {
	data, err := os.ReadFile("../../docs/actionrelations-plan-reviews.json")
	if err != nil {
		t.Fatal(err)
	}
	verdicts := map[string][]byte{}
	for _, scope := range []string{"architecture", "action-semantics", "experimental-validity"} {
		path := "docs/actionrelations-reviews/plan/round-13/" + scope + ".txt"
		verdicts[path], err = os.ReadFile(filepath.Join("../..", path))
		if err != nil {
			t.Fatal(err)
		}
	}
	manifest, err := ParseReviewManifest(data, verdicts)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Kind != "plan" || manifest.Round != 13 || manifest.ReviewedCommit != PlanCommit {
		t.Fatalf("manifest=%+v", manifest)
	}
	if err := VerifyReviewArchive("../..", manifest); err != nil {
		t.Fatal(err)
	}
	verdicts[manifest.Rows[0].VerdictPath] = []byte("ACCEPTED\n")
	if VerifyReviewManifest(manifest, verdicts) == nil {
		t.Fatal("accepted nonexact verdict bytes")
	}
}

func TestReviewArchivePreflightRejectsEveryArchiveAuthorityEscape(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{"graft", func(t *testing.T, root string) {
			writeReviewTestFile(t, filepath.Join(root, ".git/info/grafts"), strings.Repeat("a", 40)+"\n")
		}},
		{"shallow history", func(t *testing.T, root string) {
			writeReviewTestFile(t, filepath.Join(root, ".git/shallow"), strings.Repeat("a", 40)+"\n")
		}},
		{"object alternate", func(t *testing.T, root string) {
			writeReviewTestFile(t, filepath.Join(root, ".git/objects/info/alternates"), "/tmp/objects\n")
		}},
		{"external attributes", func(t *testing.T, root string) {
			writeReviewTestFile(t, filepath.Join(root, ".git/info/attributes"), "* export-ignore\n")
		}},
		{"replacement refs", func(t *testing.T, root string) {
			writeReviewTestFile(t, filepath.Join(root, ".git/refs/replace/"+strings.Repeat("a", 40)), strings.Repeat("b", 40)+"\n")
		}},
		{"archive config", func(t *testing.T, root string) { runReviewTestGit(t, root, "config", "--local", "tar.umask", "002") }},
		{"export attribute", func(t *testing.T, root string) {
			writeReviewTestFile(t, filepath.Join(root, ".gitattributes"), "fixture.txt export-subst\n")
			runReviewTestGit(t, root, "add", ".gitattributes")
			runReviewTestGit(t, root, "commit", "-q", "-m", "attributes")
		}},
		{"gitlink", func(t *testing.T, root string) {
			head := strings.TrimSpace(runReviewTestGit(t, root, "rev-parse", "HEAD"))
			runReviewTestGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+head+",nested")
			runReviewTestGit(t, root, "commit", "-q", "-m", "gitlink")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := newReviewTestRepository(t)
			test.mutate(t, root)
			manifest := reviewTestManifest(t, root)
			if err := VerifyReviewArchive(root, manifest); err == nil {
				t.Fatal("accepted archive authority escape")
			}
		})
	}
}

func TestReviewArchiveRejectsDigestSubstitution(t *testing.T) {
	root := newReviewTestRepository(t)
	manifest := reviewTestManifest(t, root)
	if err := VerifyReviewArchive(root, manifest); err != nil {
		t.Fatal(err)
	}
	manifest.ArchiveDigest = strings.Repeat("0", 64)
	if err := VerifyReviewArchive(root, manifest); err == nil {
		t.Fatal("accepted substituted archive digest")
	}
}

func newReviewTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	runReviewTestGit(t, root, "init", "-q")
	runReviewTestGit(t, root, "config", "user.email", "review@example.invalid")
	runReviewTestGit(t, root, "config", "user.name", "Review Test")
	writeReviewTestFile(t, filepath.Join(root, "fixture.txt"), "fixture\n")
	runReviewTestGit(t, root, "add", "fixture.txt")
	runReviewTestGit(t, root, "commit", "-q", "-m", "fixture")
	return root
}

func reviewTestManifest(t *testing.T, root string) ReviewManifest {
	t.Helper()
	commit := strings.TrimSpace(runReviewTestGit(t, root, "rev-parse", "HEAD"))
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	archive, err := reviewGit(root, gitPath, false, "-c", "core.attributesFile=/dev/null", "archive", "--format=tar", commit)
	if err != nil {
		t.Fatal(err)
	}
	return ReviewManifest{ReviewedCommit: commit, ArchiveDigest: shaHex(archive)}
}

func writeReviewTestFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runReviewTestGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	command.Env = []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "LC_ALL=C", "TZ=UTC"}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return string(output)
}
