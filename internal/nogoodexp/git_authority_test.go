package nogoodexp

import (
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
