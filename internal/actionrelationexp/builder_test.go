package actionrelationexp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCommittedReviewRejectsDirtyAuthority(t *testing.T) {
	root := newReviewTestRepository(t)
	base := strings.TrimSpace(runReviewTestGit(t, root, "rev-parse", "HEAD"))
	archive := reviewTestManifest(t, root).ArchiveDigest
	verdicts := map[string][]byte{}
	rows := make([]any, 3)
	for index, scope := range []string{"architecture", "action-semantics", "experimental-validity"} {
		path := "docs/actionrelations-reviews/implementation/round-1/" + scope + ".txt"
		verdicts[path] = []byte("ACCEPTED")
		writeReviewTestFile(t, filepath.Join(root, path), "ACCEPTED")
		attestation, _ := json.Marshal([]any{"actionrelation-review-attestation/v1", "implementation", 1, base, archive, scope, "review/" + scope, path, shaHex(verdicts[path]), "accepted"})
		rows[index] = []any{scope, "review/" + scope, path, shaHex(verdicts[path]), "accepted", shaHex(attestation)}
	}
	canonical, _ := json.Marshal([]any{"actionrelation-implementation-reviews/v1", 1, base, archive, rows})
	writeReviewTestFile(t, filepath.Join(root, ReviewManifestPath("implementation")), string(canonical))
	runReviewTestGit(t, root, "add", "docs")
	runReviewTestGit(t, root, "commit", "-q", "-m", "reviews")
	head := strings.TrimSpace(runReviewTestGit(t, root, "rev-parse", "HEAD"))
	manifest, err := LoadCommittedReview(root, head, "implementation")
	if err != nil || !bytes.Equal(manifest.Canonical, canonical) {
		t.Fatalf("load: %v", err)
	}
	writeReviewTestFile(t, filepath.Join(root, manifest.Rows[0].VerdictPath), "ACCEPTED\n")
	if _, err := LoadCommittedReview(root, head, "implementation"); err == nil {
		t.Fatal("accepted dirty review verdict")
	}
}

func TestResolveBuildGoUsesFrozenMiseToolchain(t *testing.T) {
	path, digest, version, goos, goarch, goPathValue, moduleCache, err := resolveBuildGo(context.Background(), "../..")
	if err != nil {
		t.Fatal(err)
	}
	if path == "" || !digestText(digest) || !strings.Contains(version, "go1.25.12") || goos == "" || goarch == "" || !filepath.IsAbs(goPathValue) || !filepath.IsAbs(moduleCache) {
		t.Fatalf("toolchain=%q %q %q %q %q %q %q", path, digest, version, goos, goarch, goPathValue, moduleCache)
	}
}

func TestPrepareBuildOutputRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", filepath.Join(root, ".nous")); err != nil {
		t.Fatal(err)
	}
	if err := prepareBuildOutput(root); err == nil {
		t.Fatal("accepted symlink output namespace")
	}
}
