package actionrelationscore

import (
	"bytes"
	"strings"
	"testing"
)

func TestPublicPanelContainsNoPrivateScorerOrGeneratorFields(t *testing.T) {
	sealed, err := PrepareDevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	public := sealed.Public()
	for _, forbidden := range []string{"Truth", "Label", "Family", "Stratum", "Ledger", "Attempt", "CurriculumSeed", "ScorerRoot", "AcceptedAttempt"} {
		if strings.Contains(string(public.Canonical()), forbidden) {
			t.Fatalf("public panel exposes private field %q", forbidden)
		}
	}
	parsed, err := ParsePublicPanel(bytes.NewReader(public.Canonical()), int64(len(public.Canonical())), public.Digest())
	if err != nil || parsed.FixtureDigest() != sealed.Fixture().Digest {
		t.Fatalf("parse public panel: %v", err)
	}
}
