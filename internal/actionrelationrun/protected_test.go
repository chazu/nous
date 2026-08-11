package actionrelationrun

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chazu/nous/internal/actionrelationcap"
	"github.com/chazu/nous/internal/actionrelationexp"
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

func TestLockedRecoveryFinishesUnlinkAfterSecretWasAlreadyZeroed(t *testing.T) {
	gitCommon := t.TempDir()
	claim := actionrelationexp.Claim{Digest: digest([]byte("claim"))}
	location, locationDigest, err := actionrelationcap.LockedSecretLocation(claim.Digest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(gitCommon, filepath.FromSlash(location))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, 32), 0o600); err != nil {
		t.Fatal(err)
	}
	running := actionrelationexp.Running{AttemptCommitment: digest([]byte("original secret")), SecretLocationDigest: &locationDigest}
	if err := eraseRecoverySecret(panelPrerequisites{GitCommonDir: gitCommon}, "locked", claim, running); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("zeroed recovery secret remains: %v", err)
	}
}
