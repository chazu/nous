package actionrelationrun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedSecretWriteIsExclusiveAndModeExact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := writeExclusiveSyncedMode(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatal("secret mode changed")
	}
	if err := writeExclusiveSyncedMode(path, []byte("replacement"), 0o600); err == nil {
		t.Fatal("secret writer overwrote an attempt root")
	}
}

func TestProtectedOutputPreflightAllowsOnlyCommittedClaimAtPrepare(t *testing.T) {
	root := t.TempDir()
	claim := filepath.Join(root, ".nous", "actionrelations-v1-validation-claim.json")
	if err := os.MkdirAll(filepath.Dir(claim), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claim, []byte("claim"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := requireProtectedOutputsAbsent(root, "validation", true); err != nil {
		t.Fatal(err)
	}
	if err := requireProtectedOutputsAbsent(root, "validation", false); err == nil {
		t.Fatal("claim stage accepted an existing claim")
	}
}
