package causalv2

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sort"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

const (
	PublicTokenDomain    = "causal-public-token/v2"
	PublicFixtureDomain  = "causal-public-fixture/v2"
	PrivateFixtureDomain = "causal-private-fixture/v2"
)

var panels = map[string]bool{"development": true, "training": true, "validation": true, "locked": true}
var cohorts = map[string]bool{"cost-skewed": true, "balanced": true, "equivalence": true, "irrelevant": true}

type PublicTokenInput struct {
	TokenVersion     string `json:"token_version"`
	Panel            string `json:"panel"`
	Seed             int64  `json:"seed"`
	GeneratorAttempt int    `json:"generator_attempt"`
}

func PublicToken(panel string, seed int64, attempt int) (string, error) {
	if !panels[panel] {
		return "", fmt.Errorf("invalid panel %q", panel)
	}
	if attempt < 0 || attempt > 4095 {
		return "", fmt.Errorf("generator attempt %d outside 0..4095", attempt)
	}
	return Digest(PublicTokenDomain, PublicTokenInput{PublicTokenDomain, panel, seed, attempt})
}

type PublicFixture struct {
	Seed                 int64    `json:"seed"`
	GeneratorAttempt     int      `json:"generator_attempt"`
	Cohort               string   `json:"cohort"`
	Aliases              []string `json:"aliases"`
	Costs                []int    `json:"costs"`
	PassiveOutcome       string   `json:"passive_outcome"`
	Pool                 []string `json:"pool"`
	Presentation         []int    `json:"presentation"`
	InitialPosterior     []string `json:"initial_posterior"`
	UniformRandomActions []string `json:"uniform_random_actions"`
	OpaqueToken          string   `json:"opaque_token"`
	FixtureDigest        string   `json:"fixture_digest"`
}

func validateOutcome(outcome string) error {
	if len(outcome) != 3 {
		return fmt.Errorf("outcome %q is not three bits", outcome)
	}
	for _, bit := range outcome {
		if bit != '0' && bit != '1' {
			return fmt.Errorf("outcome %q is not three bits", outcome)
		}
	}
	return nil
}

func validateFixtureShape(fixture PublicFixture) error {
	if fixture.GeneratorAttempt < 0 || fixture.GeneratorAttempt > 4095 {
		return errors.New("generator attempt outside 0..4095")
	}
	if !cohorts[fixture.Cohort] {
		return fmt.Errorf("invalid cohort %q", fixture.Cohort)
	}
	if len(fixture.Aliases) != 3 || len(fixture.Costs) != 3 {
		return errors.New("fixture must have exactly three aliases and costs")
	}
	seenAlias := make(map[string]bool, 3)
	for _, alias := range fixture.Aliases {
		if alias == "" || seenAlias[alias] {
			return errors.New("aliases must be nonempty and distinct")
		}
		seenAlias[alias] = true
	}
	for _, cost := range fixture.Costs {
		if cost < 1 || cost > 100 {
			return fmt.Errorf("cost %d outside 1..100", cost)
		}
	}
	if fixture.Cohort == "cost-skewed" {
		low, medium, high := 0, 0, 0
		for _, cost := range fixture.Costs {
			switch {
			case cost >= 1 && cost <= 10:
				low++
			case cost >= 30 && cost <= 50:
				medium++
			case cost >= 80 && cost <= 100:
				high++
			}
		}
		if low != 1 || medium != 1 || high != 1 {
			return errors.New("cost-skewed fixture does not have one low, medium, and high cost")
		}
	} else {
		for _, cost := range fixture.Costs {
			if cost < 20 || cost > 40 {
				return errors.New("non-cost-skewed fixture cost outside 20..40")
			}
		}
	}
	if err := validateOutcome(fixture.PassiveOutcome); err != nil {
		return err
	}
	if len(fixture.Pool) != 32 {
		return fmt.Errorf("pool has %d hypotheses, want 32", len(fixture.Pool))
	}
	if !sort.StringsAreSorted(fixture.Pool) {
		return errors.New("pool is not sorted")
	}
	seenCode := make(map[string]bool, len(fixture.Pool))
	for _, code := range fixture.Pool {
		if seenCode[code] {
			return errors.New("duplicate pool hypothesis")
		}
		seenCode[code] = true
		if _, err := causal.Parse(code); err != nil {
			return fmt.Errorf("invalid pool hypothesis: %w", err)
		}
	}
	if len(fixture.Presentation) != 32 {
		return errors.New("presentation is not a 32-element permutation")
	}
	seenPosition := make([]bool, 32)
	for _, position := range fixture.Presentation {
		if position < 0 || position >= 32 || seenPosition[position] {
			return errors.New("presentation is not a permutation of 0..31")
		}
		seenPosition[position] = true
	}
	if len(fixture.InitialPosterior) < 8 || len(fixture.InitialPosterior) > 32 || !sort.StringsAreSorted(fixture.InitialPosterior) {
		return errors.New("initial posterior must be a sorted 8..32-element array")
	}
	var expected []string
	for _, code := range fixture.Pool {
		hypothesis, _ := causal.Parse(code)
		observed, err := causal.Evaluate(hypothesis, nil)
		if err != nil {
			return err
		}
		if causal.OutcomeCode(observed) == fixture.PassiveOutcome {
			expected = append(expected, code)
		}
	}
	if !slices.Equal(fixture.InitialPosterior, expected) {
		return errors.New("initial posterior is not the passive-outcome filter of pool")
	}
	if fixture.Cohort == "irrelevant" && !hasIrrelevantVariable(fixture.InitialPosterior) {
		return errors.New("irrelevant cohort lacks an irrelevant variable")
	}
	if len(fixture.UniformRandomActions) != 10 {
		return errors.New("uniform-random prefix must contain ten actions")
	}
	for _, action := range fixture.UniformRandomActions {
		if _, err := causal.ParseAction(action); err != nil {
			return err
		}
	}
	return requireDigest("opaque_token", fixture.OpaqueToken, false)
}

func publicFixtureDigest(fixture PublicFixture) (string, error) {
	fixture.FixtureDigest = ""
	return Digest(PublicFixtureDomain, fixture)
}

func SignPublicFixture(fixture *PublicFixture) error {
	if fixture == nil {
		return errors.New("nil public fixture")
	}
	if err := validateFixtureShape(*fixture); err != nil {
		return err
	}
	digest, err := publicFixtureDigest(*fixture)
	if err != nil {
		return err
	}
	fixture.FixtureDigest = digest
	return nil
}

func VerifyPublicFixture(data []byte) (PublicFixture, error) {
	fixture, err := StrictDecode[PublicFixture](data)
	if err != nil {
		return fixture, err
	}
	if err := validateFixtureShape(fixture); err != nil {
		return fixture, err
	}
	if err := requireDigest("fixture_digest", fixture.FixtureDigest, false); err != nil {
		return fixture, err
	}
	want, err := publicFixtureDigest(fixture)
	if err != nil {
		return fixture, err
	}
	if fixture.FixtureDigest != want {
		return fixture, errors.New("public fixture digest mismatch")
	}
	return fixture, nil
}

func VerifyPublicFixtureForPanel(data []byte, panel string) (PublicFixture, error) {
	fixture, err := VerifyPublicFixture(data)
	if err != nil {
		return fixture, err
	}
	token, err := PublicToken(panel, fixture.Seed, fixture.GeneratorAttempt)
	if err != nil {
		return fixture, err
	}
	if fixture.OpaqueToken != token {
		return fixture, errors.New("public token does not match panel, seed, and attempt")
	}
	return fixture, nil
}

// VerifyPreregisteredFixtureContext proves the seed-range and cohort assignment
// required for empirical panels. Hand diagnostic fixtures intentionally use
// VerifyPublicFixtureForPanel without this empirical-only gate.
func VerifyPreregisteredFixtureContext(fixture PublicFixture, panel string) error {
	rangeForPanel, err := panelSeedRange(panel)
	if err != nil {
		return err
	}
	if fixture.Seed < rangeForPanel.Start || (fixture.Seed-rangeForPanel.Start)%rangeForPanel.Step != 0 {
		return errors.New("fixture seed is outside panel range")
	}
	index := int((fixture.Seed - rangeForPanel.Start) / rangeForPanel.Step)
	if index < 0 || index >= rangeForPanel.Count || fixture.Cohort != cohortForIndex(index) {
		return errors.New("fixture cohort does not match panel seed index")
	}
	return nil
}

type PrivateFixture struct {
	PublicFixture        PublicFixture `json:"public_fixture"`
	HiddenHypothesis     string        `json:"hidden_hypothesis"`
	PrivateFixtureDigest string        `json:"private_fixture_digest"`
}

func privateFixtureDigest(fixture PrivateFixture) (string, error) {
	fixture.PrivateFixtureDigest = ""
	return Digest(PrivateFixtureDomain, fixture)
}

func validatePrivateFixture(fixture PrivateFixture) error {
	publicBytes, err := CanonicalJSON(fixture.PublicFixture)
	if err != nil {
		return err
	}
	if _, err := VerifyPublicFixture(publicBytes); err != nil {
		return fmt.Errorf("public fixture: %w", err)
	}
	if _, err := causal.Parse(fixture.HiddenHypothesis); err != nil {
		return fmt.Errorf("hidden hypothesis: %w", err)
	}
	if !slices.Contains(fixture.PublicFixture.InitialPosterior, fixture.HiddenHypothesis) {
		return errors.New("hidden hypothesis is absent from initial posterior")
	}
	if fixture.PublicFixture.Cohort == "equivalence" {
		signature, _ := causal.Signature(fixture.HiddenHypothesis)
		count := 0
		for _, code := range fixture.PublicFixture.InitialPosterior {
			candidate, _ := causal.Signature(code)
			if candidate == signature {
				count++
			}
		}
		if count != 2 {
			return errors.New("equivalence cohort hidden class is not complete size two")
		}
	}
	return nil
}

func SignPrivateFixture(fixture *PrivateFixture) error {
	if fixture == nil {
		return errors.New("nil private fixture")
	}
	if err := validatePrivateFixture(*fixture); err != nil {
		return err
	}
	digest, err := privateFixtureDigest(*fixture)
	if err != nil {
		return err
	}
	fixture.PrivateFixtureDigest = digest
	encoded, err := CanonicalJSON(fixture)
	if err != nil {
		return err
	}
	return CheckByteCap(encoded, PreregisteredManifest().TrainingFixtureByteCap)
}

func VerifyPrivateFixture(data []byte) (PrivateFixture, error) {
	fixture, err := StrictDecode[PrivateFixture](data)
	if err != nil {
		return fixture, err
	}
	if err := validatePrivateFixture(fixture); err != nil {
		return fixture, err
	}
	if err := requireDigest("private_fixture_digest", fixture.PrivateFixtureDigest, false); err != nil {
		return fixture, err
	}
	want, err := privateFixtureDigest(fixture)
	if err != nil {
		return fixture, err
	}
	if fixture.PrivateFixtureDigest != want {
		return fixture, errors.New("private fixture digest mismatch")
	}
	if err := CheckByteCap(data, PreregisteredManifest().TrainingFixtureByteCap); err != nil {
		return fixture, err
	}
	return fixture, nil
}

func hashParts(parts ...string) string {
	h := sha256.New()
	for i, part := range parts {
		if i > 0 {
			_, _ = h.Write([]byte{0})
		}
		_, _ = h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func panelSeedRange(panel string) (SeedRange, error) {
	m := PreregisteredManifest()
	switch panel {
	case "development":
		return m.DevelopmentSeeds, nil
	case "training":
		return m.TrainingSeeds, nil
	case "validation":
		return m.ValidationSeeds, nil
	case "locked":
		return m.LockedSeeds, nil
	default:
		return SeedRange{}, fmt.Errorf("invalid panel %q", panel)
	}
}

func cohortForIndex(index int) string {
	switch index % 8 {
	case 0, 1, 2, 3:
		return "cost-skewed"
	case 4, 5:
		return "balanced"
	case 6:
		return "equivalence"
	default:
		return "irrelevant"
	}
}

func hasIrrelevantVariable(posterior []string) bool {
	for variable := 0; variable < 3; variable++ {
		irrelevant := true
		for value := 0; value < 2; value++ {
			cells, err := causal.Partition(posterior, fmt.Sprintf("do:%d=%d", variable, value))
			if err != nil || len(cells) != 1 {
				irrelevant = false
				break
			}
		}
		if irrelevant {
			return true
		}
	}
	return false
}
