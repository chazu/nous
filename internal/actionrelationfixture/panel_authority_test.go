package actionrelationfixture

import (
	"bytes"
	"testing"
)

func TestDevelopmentPanelFixtureHasOneExactRootAndAllCurricula(t *testing.T) {
	attempts, fixture, err := GenerateDevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 16 || len(fixture.CurriculumRoots) != 16 || len(fixture.Canonical) > 4096 || fixture.Digest == "" {
		t.Fatalf("attempts=%d roots=%d bytes=%d digest=%q", len(attempts), len(fixture.CurriculumRoots), len(fixture.Canonical), fixture.Digest)
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

func TestLockedCurriculumSeedIsExactHMACContext(t *testing.T) {
	authority := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	seed, err := LockedCurriculumSeed(authority, 3)
	if err != nil {
		t.Fatal(err)
	}
	context := DrawContext{Panel: "locked", Authority: authority, Curriculum: 3, CurriculumSeed: seed, Attempt: 0}
	if _, err := precommitDraws(context); err != nil {
		t.Fatal(err)
	}
	context.CurriculumSeed = authority
	if _, err := precommitDraws(context); err == nil {
		t.Fatal("accepted an arbitrary locked seed digest")
	}
	context.CurriculumSeed = seed
	if _, err := PrecommitDraws(context); err == nil {
		t.Fatal("public draw API accepted locked construction")
	}
}
