package transformexp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewManifestCanonicalRoundTripAndScopeOrder(t *testing.T) {
	commit := "0123456789abcdef0123456789abcdef01234567"
	manifest := ImplementationReviewManifest{
		Version: "transform-implementation-reviews/v1", PlanCommit: PlanCommit, ImplementationCommit: commit,
		Reviews:        []ImplementationReview{{"architecture", "accepted", commit}, {"semantics", "accepted", commit}, {"experiment", "accepted", commit}},
		ProtectedPaths: map[string]string{"go.mod": strings.Repeat("a", 64)},
	}
	encoded, err := encodeReviewManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeReviewManifest(encoded)
	if err != nil || decoded.ImplementationCommit != commit {
		t.Fatalf("round trip failed: %+v %v", decoded, err)
	}
	wrong := strings.Replace(string(encoded), `"architecture"`, `"experiment"`, 1)
	if _, err := decodeReviewManifest([]byte(wrong)); err == nil {
		t.Fatal("wrong review scope order was accepted")
	}
}

func TestAttemptReceiptIsExclusiveAndMonotone(t *testing.T) {
	root := t.TempDir()
	authority := repositoryAuthority{Root: root, Head: strings.Repeat("a", 40), Reviews: ImplementationReviewManifest{ImplementationCommit: strings.Repeat("b", 40)}}
	receipt, err := claimAttempt(authority, "validation")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := claimAttempt(authority, "validation"); err == nil {
		t.Fatal("duplicate receipt claim succeeded")
	}
	if err := startAttempt(root, receipt, "", ""); err != nil {
		t.Fatal(err)
	}
	fixture, report, graph := strings.Repeat("c", 64), strings.Repeat("d", 64), strings.Repeat("e", 64)
	if err := finalizeAttempt(root, receipt, "published", fixture, report, graph); err != nil {
		t.Fatal(err)
	}
	if err := finalizeAttempt(root, receipt, "invalid", "", "", ""); err == nil {
		t.Fatal("published receipt transitioned")
	}
	data, err := os.ReadFile(receiptPath(root, "validation"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"published"`) || !strings.Contains(string(data), fixture) {
		t.Fatalf("final receipt lost committed fields: %s", data)
	}
}

func TestReviewedFilesystemRejectsIgnoredSourceAndSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	protected := map[string]string{"go.mod": strings.Repeat("a", 64)}
	if err := verifyReviewedFilesystem(root, protected); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.go"), []byte("package ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyReviewedFilesystem(root, protected); err == nil {
		t.Fatal("unreviewed compiler input was accepted")
	}
	if err := os.Remove(filepath.Join(root, "ignored.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("go.mod", filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := verifyReviewedFilesystem(root, protected); err == nil {
		t.Fatal("repository symlink was accepted")
	}
}

func TestProtectedPanelConstructorsHaveExactlyOneProductionCaller(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{"developmentPanel(": 0, "validationPanel(": 0, "lockedPanel(": 0}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		for surface := range counts {
			counts[surface] += strings.Count(string(data), surface)
		}
	}
	for surface, count := range counts {
		if count != 2 {
			t.Fatalf("%s production occurrences = %d, want definition plus one guarded call", surface, count)
		}
	}
}
