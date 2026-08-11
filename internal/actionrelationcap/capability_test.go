package actionrelationcap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestZeroCapabilityIsInert(t *testing.T) {
	var token Token
	if _, ok := token.Panel(); ok {
		t.Fatal("zero token has panel")
	}
	if _, ok := token.Authority(); ok {
		t.Fatal("zero token has authority")
	}
	if _, ok := token.CurriculumSeed(0); ok {
		t.Fatal("zero token has seed")
	}
}

func TestLockedCapabilitySeparatesPublicCommitmentFromHMACRoot(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	token := Token{grant: &grant{panel: "locked", authority: digestBytes(secret), secret: secret}}
	seed, ok := token.CurriculumSeed(3)
	if !ok || seed == token.grant.authority {
		t.Fatal("locked seed did not use private preimage")
	}
	value, _ := hex.DecodeString(seed.(string))
	if len(value) != sha256.Size {
		t.Fatal("locked seed is not an HMAC digest")
	}
	if !token.VerifyCurriculumSeed(3, seed) || token.VerifyCurriculumSeed(3, token.grant.authority) || token.VerifyCurriculumSeed(4, seed) {
		t.Fatal("locked seed verification did not bind the exact curriculum HMAC")
	}
}

func TestValidationCapabilityVerifiesOnlyExactPublicSeed(t *testing.T) {
	token := Token{grant: &grant{panel: "validation", authority: validationAuthority}}
	if !token.VerifyCurriculumSeed(7, 852008) || token.VerifyCurriculumSeed(7, 852007) || token.VerifyCurriculumSeed(7, "852008") {
		t.Fatal("validation seed verification did not bind type and curriculum")
	}
}

func TestCapabilityAllowsExactlyOneSealedFixtureConstruction(t *testing.T) {
	token := Token{grant: &grant{panel: "validation", authority: validationAuthority}}
	if _, _, ok := token.BeginConstruction(); !ok {
		t.Fatal("sealed fixture construction was not authorized")
	}
	if _, _, ok := token.BeginConstruction(); ok {
		t.Fatal("capability authorized fixture regeneration")
	}
	if err := token.Destroy(); err != nil {
		t.Fatal(err)
	}
	if _, ok := token.Panel(); ok {
		t.Fatal("destroyed capability retained panel authority")
	}
}

func TestDestroyErasesAndDeletesLockedRoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "locked.root")
	secret := []byte("0123456789abcdef0123456789abcdef")
	if err := os.WriteFile(path, secret, 0o600); err != nil {
		t.Fatal(err)
	}
	token := Token{grant: &grant{panel: "locked", authority: digestBytes(secret), secret: bytes.Clone(secret), secretPath: path}}
	if err := token.Destroy(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("destroy did not delete the erased locked root")
	}
	if len(token.grant.secret) != 0 || !token.grant.destroyed {
		t.Fatal("destroy retained in-memory locked authority")
	}
}

func TestReleaseForRetryPreservesDiskRootButErasesMemory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "locked.root")
	secret := []byte("0123456789abcdef0123456789abcdef")
	if err := os.WriteFile(path, secret, 0o600); err != nil {
		t.Fatal(err)
	}
	token := Token{grant: &grant{panel: "locked", authority: digestBytes(secret), secret: bytes.Clone(secret), secretPath: path}}
	token.ReleaseForRetry()
	if len(token.grant.secret) != 0 || !token.grant.destroyed {
		t.Fatal("retry release retained in-memory authority")
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, secret) {
		t.Fatalf("retry release consumed disk authority: %v", err)
	}
}

func TestSecretIORejectsSymlinkedAncestor(t *testing.T) {
	root := t.TempDir()
	out := t.TempDir()
	secret := []byte("0123456789abcdef0123456789abcdef")
	if err := os.WriteFile(filepath.Join(out, "locked.root"), secret, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(out, filepath.Join(root, "secrets")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "secrets", "locked.root")
	if _, err := readSecretNoFollow(path); err == nil {
		t.Fatal("secret read followed a symlinked ancestor")
	}
	if err := eraseSecretFile(path); err == nil {
		t.Fatal("secret erasure followed a symlinked ancestor")
	}
	data, err := os.ReadFile(filepath.Join(out, "locked.root"))
	if err != nil || !bytes.Equal(data, secret) {
		t.Fatalf("outside secret changed: %v", err)
	}
}

func TestAttemptCommitmentsAndSecretLocationsAreCanonical(t *testing.T) {
	if !digestText(ValidationAttemptCommitment()) {
		t.Fatal("invalid validation attempt commitment")
	}
	if authority := LockedClaimAuthority("0123456789012345678901234567890123456789", digestBytes([]byte("source"))); !digestText(authority) {
		t.Fatal("locked claim authority is not a seedless digest")
	}
	claim := digestBytes([]byte("claim"))
	location, commitment, err := LockedSecretLocation(claim)
	if err != nil || location != "nous-actionrelations-v1/secrets/locked-"+claim+".root" || commitment != digestBytes([]byte(location)) {
		t.Fatal("locked location authority changed")
	}
}
