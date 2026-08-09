package nogoodexp

import (
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
