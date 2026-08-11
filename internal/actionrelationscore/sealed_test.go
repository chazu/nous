package actionrelationscore

import (
	"bytes"
	"testing"
)

func TestSealedDevelopmentPanelRoundTripsWithoutConstructorAuthority(t *testing.T) {
	sealed, err := PrepareDevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	canonical := sealed.Canonical()
	parsed, err := ParseSealedPanel(bytes.NewReader(canonical), int64(len(canonical)), sealed.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Panel() != "development" || parsed.Authority() != "development-public-v1" || parsed.Fixture().Digest != sealed.Fixture().Digest || len(parsed.attempts) != 16 {
		t.Fatal("sealed development panel authority changed during round trip")
	}
	canonical[len(canonical)-1] ^= 1
	if _, err := ParseSealedPanel(bytes.NewReader(canonical), int64(len(canonical)), sealed.Digest()); err == nil {
		t.Fatal("sealed panel parser accepted changed bytes")
	}
}
