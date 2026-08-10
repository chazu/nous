package actionrelationexp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceRowsComeFromCommitAndCheckoutMustMatch(t *testing.T) {
	root := newReviewTestRepository(t)
	writeReviewTestFile(t, filepath.Join(root, "main.go"), "package main\n")
	writeReviewTestFile(t, filepath.Join(root, "main_test.go"), "package main\n")
	writeReviewTestFile(t, filepath.Join(root, "fixture.cue"), "package fixture\n")
	runReviewTestGit(t, root, "add", ".")
	runReviewTestGit(t, root, "commit", "-q", "-m", "sources")
	commit := strings.TrimSpace(runReviewTestGit(t, root, "rev-parse", "HEAD"))
	rows, err := CollectSourceRows(root, commit)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Role != "domain" || rows[1].Role != "compiler-input" || rows[2].Role != "test" {
		t.Fatalf("rows=%+v", rows)
	}
	if err := VerifySourceCheckout(root, rows); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if VerifySourceCheckout(root, rows) == nil {
		t.Fatal("accepted dirty reviewed source")
	}
}

func TestSourceCheckoutRejectsUntrackedCompilerInput(t *testing.T) {
	root := newReviewTestRepository(t)
	writeReviewTestFile(t, filepath.Join(root, "main.go"), "package main\n")
	runReviewTestGit(t, root, "add", "main.go")
	runReviewTestGit(t, root, "commit", "-q", "-m", "source")
	commit := strings.TrimSpace(runReviewTestGit(t, root, "rev-parse", "HEAD"))
	rows, err := CollectSourceRows(root, commit)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewTestFile(t, filepath.Join(root, "injected.go"), "package injected\n")
	if VerifySourceCheckout(root, rows) == nil {
		t.Fatal("accepted untracked compiler input")
	}
}

func TestNonInputSnapshotIsClosedAndRejectsSymlink(t *testing.T) {
	root := newReviewTestRepository(t)
	rows, err := SnapshotNonInputs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(standardNonInputPaths) || rows[0].Path != ".git/hooks" || rows[len(rows)-1].Path != "runs" {
		t.Fatalf("rows=%+v", rows)
	}
	if err := os.Symlink("missing", filepath.Join(root, ".maki")); err != nil {
		t.Fatal(err)
	}
	if _, err := SnapshotNonInputs(root); err == nil {
		t.Fatal("accepted symlink non-input namespace")
	}
}
