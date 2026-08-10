package actionrelationfixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationcap"
	"github.com/chazu/nous/internal/actionrelationwire"
)

type PanelFixture struct {
	Panel           string
	Authority       string
	CurriculumRoots []string
	ScorerRoot      string
	Canonical       []byte
	Digest          string
}

func ParsePanelFixture(data []byte) (PanelFixture, error) {
	var fields []json.RawMessage
	var version string
	value := PanelFixture{Canonical: bytes.Clone(data), Digest: digestBytes(data)}
	if len(data) > 4096 || json.Unmarshal(data, &fields) != nil || len(fields) != 5 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-fixture-root/v2" || json.Unmarshal(fields[1], &value.Panel) != nil || json.Unmarshal(fields[2], &value.Authority) != nil || json.Unmarshal(fields[3], &value.CurriculumRoots) != nil || json.Unmarshal(fields[4], &value.ScorerRoot) != nil || VerifyPanelFixture(value) != nil {
		return PanelFixture{}, fmt.Errorf("invalid panel fixture wire")
	}
	return value, nil
}

func GenerateDevelopmentPanel() ([]GeneratedAttempt, PanelFixture, error) {
	return generatePublicPanel("development", "development-public-v1", 851001, 16)
}

// GenerateProtectedPanel is the sole protected fixture constructor. The token
// has already consumed committed running authority and the local start marker.
func GenerateProtectedPanel(token actionrelationcap.Token) ([]GeneratedAttempt, PanelFixture, error) {
	panel, authority, ok := token.BeginConstruction()
	if !ok {
		return nil, PanelFixture{}, fmt.Errorf("protected fixture requires an unconsumed authorization")
	}
	count := map[string]int{"validation": 24, "locked": 32}[panel]
	attempts := make([]GeneratedAttempt, count)
	for curriculum := 0; curriculum < count; curriculum++ {
		seed, valid := token.CurriculumSeed(curriculum)
		if !valid {
			return nil, PanelFixture{}, fmt.Errorf("protected curriculum %d lacks seed authority", curriculum)
		}
		var prior []AttemptLedger
		for attempt := 0; attempt < 32; attempt++ {
			context := DrawContext{Panel: panel, Authority: authority, Curriculum: curriculum, CurriculumSeed: seed, Attempt: attempt}
			generated, err := generateAttempt(context, prior)
			if err == nil {
				attempts[curriculum] = generated
				break
			}
			if generated.Ledger.Terminal != "rejected" {
				return nil, PanelFixture{}, fmt.Errorf("curriculum %d attempt %d: %w", curriculum, attempt, err)
			}
			prior = append(prior, generated.Ledger)
		}
		if attempts[curriculum].Fixture.Digest == "" {
			return nil, PanelFixture{}, fmt.Errorf("curriculum %d exhausted generator attempts", curriculum)
		}
	}
	fixture, err := sealPanelFixture(panel, authority, attempts)
	return attempts, fixture, err
}

func generatePublicPanel(panel, authority string, start, count int) ([]GeneratedAttempt, PanelFixture, error) {
	attempts := make([]GeneratedAttempt, count)
	for curriculum := 0; curriculum < count; curriculum++ {
		var prior []AttemptLedger
		for attempt := 0; attempt < 32; attempt++ {
			context := DrawContext{Panel: panel, Authority: authority, Curriculum: curriculum, CurriculumSeed: start + curriculum, Attempt: attempt}
			generated, err := generateAttempt(context, prior)
			if err == nil {
				attempts[curriculum] = generated
				break
			}
			if generated.Ledger.Terminal != "rejected" {
				return nil, PanelFixture{}, fmt.Errorf("curriculum %d attempt %d: %w", curriculum, attempt, err)
			}
			prior = append(prior, generated.Ledger)
		}
		if attempts[curriculum].Fixture.Digest == "" {
			return nil, PanelFixture{}, fmt.Errorf("curriculum %d exhausted generator attempts", curriculum)
		}
	}
	fixture, err := sealPanelFixture(panel, authority, attempts)
	return attempts, fixture, err
}

func SealPanelFixture(panel, authority string, attempts []GeneratedAttempt) (PanelFixture, error) {
	if panel != "development" {
		return PanelFixture{}, fmt.Errorf("protected fixture sealing requires guarded capability")
	}
	return sealPanelFixture(panel, authority, attempts)
}

func sealPanelFixture(panel, authority string, attempts []GeneratedAttempt) (PanelFixture, error) {
	wantCount, err := validatePanelFixtureAuthority(panel, authority)
	if err != nil || len(attempts) != wantCount {
		return PanelFixture{}, fmt.Errorf("invalid panel fixture cardinality")
	}
	curriculumRoots := make([]string, wantCount)
	type shardReference struct {
		worldDigest string
		ordinal     int
		digest      string
	}
	var references []shardReference
	for curriculum, attempt := range attempts {
		if validateDrawContext(attempt.Context) != nil || attempt.Context.Panel != panel || attempt.Context.Authority != authority || attempt.Context.Curriculum != curriculum || attempt.Ledger.Terminal != "accepted" || VerifyCurriculumFixture(attempt.Fixture) != nil || attempt.Fixture.Panel != panel || attempt.Fixture.Curriculum != curriculum || len(attempt.Truth.Worlds) != 6 {
			return PanelFixture{}, fmt.Errorf("invalid panel curriculum %d", curriculum)
		}
		wantTruth, err := SealCurriculumTruth(attempt.Curriculum)
		if err != nil || wantTruth.Root != attempt.Truth.Root {
			return PanelFixture{}, fmt.Errorf("panel curriculum %d changed scorer truth", curriculum)
		}
		wantTraining, err := SealTrainingAuthorityFromCases(attempt.Training, nil)
		if err != nil || !slices.Equal(wantTraining.CoreDigests, attempt.TrainingAuthority.CoreDigests) || !slices.Equal(wantTraining.ViewEvidenceDigests, attempt.TrainingAuthority.ViewEvidenceDigests) {
			return PanelFixture{}, fmt.Errorf("panel curriculum %d changed training authority", curriculum)
		}
		wantFixture, err := assembleCurriculumFixture(attempt.Context, attempt.Curriculum, attempt.Truth, attempt.AttemptLedgers, attempt.TrainingAuthority)
		if err != nil || wantFixture.Digest != attempt.Fixture.Digest {
			return PanelFixture{}, fmt.Errorf("panel curriculum %d changed fixture authority", curriculum)
		}
		curriculumRoots[curriculum] = attempt.Fixture.Digest
		for _, world := range attempt.Truth.Worlds {
			if VerifyWorldTruth(world) != nil {
				return PanelFixture{}, fmt.Errorf("invalid panel scorer world")
			}
			for _, shard := range world.Shards {
				references = append(references, shardReference{worldDigest: world.WorldDigest, ordinal: shard.Ordinal, digest: shard.Digest})
			}
		}
	}
	slices.SortFunc(references, func(a, b shardReference) int {
		if value := bytes.Compare(mustDigest(a.worldDigest), mustDigest(b.worldDigest)); value != 0 {
			return value
		}
		if a.ordinal != b.ordinal {
			return a.ordinal - b.ordinal
		}
		return bytes.Compare(mustDigest(a.digest), mustDigest(b.digest))
	})
	rootRows := make([]any, len(references))
	for index, reference := range references {
		rootRows[index] = []any{reference.worldDigest, reference.ordinal, reference.digest}
	}
	scorerRoot, err := actionrelationwire.RootDigest("scorer-shards", rootRows)
	if err != nil {
		return PanelFixture{}, err
	}
	fixture := PanelFixture{Panel: panel, Authority: authority, CurriculumRoots: curriculumRoots, ScorerRoot: scorerRoot}
	fixture.Canonical, _ = json.Marshal([]any{"actionrelation-fixture-root/v2", panel, authority, curriculumRoots, scorerRoot})
	fixture.Digest = digestBytes(fixture.Canonical)
	if err := VerifyPanelFixture(fixture); err != nil {
		return PanelFixture{}, err
	}
	return fixture, nil
}

func VerifyPanelFixture(fixture PanelFixture) error {
	wantCount, err := validatePanelFixtureAuthority(fixture.Panel, fixture.Authority)
	if err != nil || len(fixture.CurriculumRoots) != wantCount || !digestText(fixture.ScorerRoot) {
		return fmt.Errorf("invalid panel fixture shape")
	}
	seen := map[string]bool{}
	for _, digest := range fixture.CurriculumRoots {
		if !digestText(digest) || seen[digest] {
			return fmt.Errorf("invalid panel curriculum root")
		}
		seen[digest] = true
	}
	want, _ := json.Marshal([]any{"actionrelation-fixture-root/v2", fixture.Panel, fixture.Authority, fixture.CurriculumRoots, fixture.ScorerRoot})
	if !bytes.Equal(want, fixture.Canonical) || fixture.Digest != digestBytes(fixture.Canonical) || len(fixture.Canonical) > 4096 {
		return fmt.Errorf("invalid panel fixture wire")
	}
	return nil
}

func validatePanelFixtureAuthority(panel, authority string) (int, error) {
	switch panel {
	case "development":
		if authority == "development-public-v1" {
			return 16, nil
		}
	case "validation":
		if authority == "validation-public-v1" {
			return 24, nil
		}
	case "locked":
		if digestText(authority) {
			return 32, nil
		}
	}
	return 0, fmt.Errorf("invalid panel fixture authority")
}
