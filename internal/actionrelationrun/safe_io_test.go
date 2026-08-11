package actionrelationrun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuthorityWriteRejectsSymlinkedNousAncestor(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".nous")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".nous", "escaped.json")
	if err := writeExclusiveAuthority(path, []byte("authority")); err == nil {
		t.Fatal("authority write followed a symlinked .nous ancestor")
	}
	if _, err := os.Lstat(filepath.Join(outside, "escaped.json")); !os.IsNotExist(err) {
		t.Fatalf("symlink escape created outside file: %v", err)
	}
}

func TestAuthorityWriteRejectsSymlinkLeaf(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, ".nous")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	leaf := filepath.Join(directory, "authority.json")
	if err := os.Symlink(target, leaf); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusiveAuthority(leaf, []byte("replacement")); err == nil {
		t.Fatal("authority write followed a symlink leaf")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "original" {
		t.Fatalf("symlink target changed: %q %v", data, err)
	}
}

func TestAtomicAuthorityInstallIsIdempotentAndNoncorrective(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".nous", "authority.json")
	linked, err := installAtomicNoFollow(path, []byte("first"), 0o644, 0o755)
	if err != nil || !linked {
		t.Fatalf("first install: linked=%v err=%v", linked, err)
	}
	linked, err = installAtomicNoFollow(path, []byte("first"), 0o644, 0o755)
	if err != nil || linked {
		t.Fatalf("idempotent install: linked=%v err=%v", linked, err)
	}
	if _, err := installAtomicNoFollow(path, []byte("second"), 0o644, 0o755); err == nil {
		t.Fatal("atomic install corrected existing authority")
	}
	data, err := readRegularNoFollow(path, 0o644)
	if err != nil || string(data) != "first" {
		t.Fatalf("authority changed: %q %v", data, err)
	}
}

func TestStagedLinkCommitsExactInodeAndCleansPending(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(root, "receipt.pending")
	final := filepath.Join(root, "receipt.json")
	if err := writeExclusiveAuthority(staged, []byte("published")); err != nil {
		t.Fatal(err)
	}
	committed, err := linkStagedNoFollow(staged, final, []byte("published"), 0o644)
	if err != nil || !committed {
		t.Fatalf("commit staged receipt: committed=%v err=%v", committed, err)
	}
	if err := requireAbsentNoFollow(staged); err != nil {
		t.Fatalf("pending link survived commit: %v", err)
	}
	data, err := readRegularNoFollow(final, 0o644)
	if err != nil || string(data) != "published" {
		t.Fatalf("committed receipt changed: %q %v", data, err)
	}
}
