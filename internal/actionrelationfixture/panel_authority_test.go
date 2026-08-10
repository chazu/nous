package actionrelationfixture

import (
	"bytes"
	"testing"

	"github.com/chazu/nous/internal/actionrelationcap"
)

func TestDevelopmentPanelFixtureHasOneExactRootAndAllCurricula(t *testing.T) {
	attempts, fixture, err := GenerateDevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 16 || len(fixture.CurriculumRoots) != 16 || len(fixture.Canonical) > 4096 || fixture.Digest == "" {
		t.Fatalf("attempts=%d roots=%d bytes=%d digest=%q", len(attempts), len(fixture.CurriculumRoots), len(fixture.Canonical), fixture.Digest)
	}
	if parsed, err := ParsePanelFixture(fixture.Canonical); err != nil || parsed.Digest != fixture.Digest {
		t.Fatalf("parse fixture: %v", err)
	}
	again, err := SealPanelFixture("development", "development-public-v1", attempts)
	if err != nil || again.Digest != fixture.Digest || !bytes.Equal(again.Canonical, fixture.Canonical) {
		t.Fatal("development panel fixture is not deterministic")
	}
	for curriculum, attempt := range attempts {
		if fixture.CurriculumRoots[curriculum] != attempt.Fixture.Digest {
			t.Fatalf("curriculum root %d changed ordinal", curriculum)
		}
	}
}

func TestProtectedPanelAPIsRejectZeroCapability(t *testing.T) {
	if _, _, err := GenerateProtectedPanel(actionrelationcap.Token{}); err == nil {
		t.Fatal("protected generator accepted zero capability")
	}
	if _, err := SealPanelFixture("locked", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", nil); err == nil {
		t.Fatal("public fixture sealer accepted locked panel")
	}
}
