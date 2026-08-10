package actionrelationcap

import (
	"crypto/sha256"
	"encoding/hex"
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
}

func TestCapabilityAllowsExactlyPrimaryAndAuditConstruction(t *testing.T) {
	token := Token{grant: &grant{panel: "validation", authority: validationAuthority}}
	for use := 0; use < 2; use++ {
		if _, _, ok := token.BeginConstruction(); !ok {
			t.Fatalf("construction %d was not authorized", use)
		}
	}
	if _, _, ok := token.BeginConstruction(); ok {
		t.Fatal("capability authorized a third construction")
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
