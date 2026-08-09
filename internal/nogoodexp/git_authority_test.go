package nogoodexp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGitAuthorityIgnoresInheritedGitMetadataOverrides(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_DIR", t.TempDir())
	t.Setenv("GIT_WORK_TREE", t.TempDir())
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "index"))
	t.Setenv("GIT_OBJECT_DIRECTORY", t.TempDir())
	top, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	if top != root {
		t.Fatalf("hardened Git top=%q want=%q", top, root)
	}
}

func TestCommittedReaderUsesRegularGitBlobBytes(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	head, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	committed, err := readCommittedRegularBlob(repositoryAuthority{root: root, head: head}, filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	working, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || !bytes.Equal(committed, working) {
		t.Fatalf("committed regular blob mismatch: %v", err)
	}
}

func TestReviewedFilesystemRejectsIgnoredInputsAndSymlinks(t *testing.T) {
	for _, relative := range []string{filepath.Join("internal", "hidden.go"), filepath.Join("domains", "nogoods", "hidden.cue")} {
		t.Run(relative, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, relative)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("ignored input"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := verifyReviewedFilesystemInputs(root, map[string]string{}); err == nil {
				t.Fatal("ignored compiler/runtime input escaped the reviewed surface")
			}
		})
	}
	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "target")
		if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, "linked.go")); err != nil {
			t.Fatal(err)
		}
		if err := verifyReviewedFilesystemInputs(root, map[string]string{}); err == nil {
			t.Fatal("symlink escaped the reviewed source surface")
		}
	})
}

func TestEvidencePathsRejectSymlinkedLeavesAndParents(t *testing.T) {
	t.Run("leaf", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "target")
		if err := os.WriteFile(target, []byte("report"), 0o644); err != nil {
			t.Fatal(err)
		}
		base := filepath.Join(root, ".nous")
		if err := os.Mkdir(base, 0o755); err != nil {
			t.Fatal(err)
		}
		leaf := filepath.Join(base, "report.json")
		if err := os.Symlink(target, leaf); err != nil {
			t.Fatal(err)
		}
		if err := requireRegularPath(root, leaf, false); err == nil {
			t.Fatal("symlinked evidence leaf was accepted")
		}
	})
	t.Run("parent", func(t *testing.T) {
		root := t.TempDir()
		target := t.TempDir()
		if err := os.WriteFile(filepath.Join(target, "report.json"), []byte("report"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, ".nous")); err != nil {
			t.Fatal(err)
		}
		if err := requireRegularPath(root, filepath.Join(root, ".nous", "report.json"), false); err == nil {
			t.Fatal("symlinked evidence parent was accepted")
		}
	})
}
