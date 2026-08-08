package causalv2

import (
	"sort"
	"strings"
	"testing"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

func handPublicFixture(t *testing.T) PublicFixture {
	t.Helper()
	models := causal.Enumerate()
	pool := make([]string, 32)
	for i := range pool {
		pool[i], _ = causal.Code(models[i])
	}
	sort.Strings(pool)
	counts := map[string]int{}
	for _, code := range pool {
		h, _ := causal.Parse(code)
		outcome, _ := causal.Evaluate(h, nil)
		counts[causal.OutcomeCode(outcome)]++
	}
	passive := ""
	for outcome, count := range counts {
		if count >= 8 && (passive == "" || outcome < passive) {
			passive = outcome
		}
	}
	if passive == "" {
		t.Fatal("hand pool has no legal passive posterior")
	}
	var posterior []string
	for _, code := range pool {
		h, _ := causal.Parse(code)
		outcome, _ := causal.Evaluate(h, nil)
		if causal.OutcomeCode(outcome) == passive {
			posterior = append(posterior, code)
		}
	}
	token, err := PublicToken("development", 112005, 0)
	if err != nil {
		t.Fatal(err)
	}
	presentation := make([]int, 32)
	for i := range presentation {
		presentation[i] = 31 - i
	}
	fixture := PublicFixture{
		Seed: 112005, GeneratorAttempt: 0, Cohort: "balanced",
		Aliases: []string{"node-λ", "node-v", "node-w"}, Costs: []int{20, 30, 40}, PassiveOutcome: passive,
		Pool: pool, Presentation: presentation, InitialPosterior: posterior,
		UniformRandomActions: []string{"do:0=0", "do:0=1", "do:1=0", "do:1=1", "do:2=0", "do:2=1", "do:0=0", "do:0=1", "do:1=0", "do:1=1"}, OpaqueToken: token,
	}
	if err := SignPublicFixture(&fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestFixtureAndProfileRoundTrip(t *testing.T) {
	fixture := handPublicFixture(t)
	encoded, _ := CanonicalJSON(fixture)
	verified, err := VerifyPublicFixtureForPanel(encoded, "development")
	if err != nil || verified.FixtureDigest != fixture.FixtureDigest {
		t.Fatalf("VerifyPublicFixture=%v, %v", verified, err)
	}
	if err := VerifyPreregisteredFixtureContext(verified, "development"); err != nil {
		t.Fatal(err)
	}

	private := PrivateFixture{PublicFixture: fixture, HiddenHypothesis: fixture.InitialPosterior[0]}
	if err := SignPrivateFixture(&private); err != nil {
		t.Fatal(err)
	}
	privateBytes, _ := CanonicalJSON(private)
	if _, err := VerifyPrivateFixture(privateBytes); err != nil {
		t.Fatal(err)
	}

	profile := Profile{ProfileVersion: ProfileDomain, Manifest: PreregisteredManifest(), Panel: "development", Seed: fixture.Seed, AcquisitionCode: causal.Rules()[0].Code(), FixtureDigest: fixture.FixtureDigest}
	if err := SignProfile(&profile); err != nil {
		t.Fatal(err)
	}
	profileBytes, _ := CanonicalJSON(profile)
	if _, err := VerifyProfileForFixture(profileBytes, fixture.FixtureDigest); err != nil {
		t.Fatal(err)
	}

	central := CentralProfile{CentralProfileVersion: CentralProfileDomain, Manifest: PreregisteredManifest(), PlanCommit: strings.Repeat("a", 40), PretrainingCommit: strings.Repeat("b", 40)}
	if err := SignCentralProfile(&central); err != nil {
		t.Fatal(err)
	}
	centralBytes, _ := CanonicalJSON(central)
	if _, err := VerifyCentralProfile(centralBytes); err != nil {
		t.Fatal(err)
	}
}

func TestFixtureRejectsDigestAndPresentationCorruption(t *testing.T) {
	fixture := handPublicFixture(t)
	fixture.Presentation[0] = fixture.Presentation[1]
	if err := SignPublicFixture(&fixture); err == nil {
		t.Fatal("accepted duplicate presentation position")
	}
	fixture = handPublicFixture(t)
	fixture.FixtureDigest = strings.Repeat("0", 64)
	encoded, _ := CanonicalJSON(fixture)
	if _, err := VerifyPublicFixture(encoded); err == nil {
		t.Fatal("accepted corrupt fixture digest")
	}
}
