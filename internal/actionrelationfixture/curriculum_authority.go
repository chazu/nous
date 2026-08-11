package actionrelationfixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationfixturecore"
)

type CurriculumFixture struct {
	Panel               string
	Curriculum          int
	Family              int
	AcceptedAttempt     int
	TrainingCoreDigests []string
	ViewEvidenceDigests []string
	WorldDigests        []string
	AttemptLedgers      []string
	Canonical           []byte
	Digest              string
}

func SealCurriculumFixture(context DrawContext, curriculum Curriculum, truth CurriculumTruth, ledgers []AttemptLedger) (CurriculumFixture, error) {
	if context.Panel != "development" {
		return CurriculumFixture{}, fmt.Errorf("protected fixture sealing requires guarded capability")
	}
	if err := validateDrawContext(context); err != nil || curriculum.Family != context.Curriculum%8 || curriculum.WithinFamilyOrdinal != context.Curriculum/8 || len(curriculum.Worlds) != 6 || len(truth.Worlds) != 6 {
		return CurriculumFixture{}, fmt.Errorf("invalid curriculum fixture input")
	}
	wantTruth, err := sealCurriculumTruthMeasured(curriculum, nil)
	if err != nil || truth.Root != wantTruth.Root {
		return CurriculumFixture{}, fmt.Errorf("curriculum scorer truth changed")
	}
	training, err := SealTrainingAuthority(curriculum.Family)
	if err != nil {
		return CurriculumFixture{}, err
	}
	return assembleCurriculumFixture(context, curriculum, truth, ledgers, training)
}

func assembleCurriculumFixture(context DrawContext, curriculum Curriculum, truth CurriculumTruth, ledgers []AttemptLedger, training TrainingAuthority) (CurriculumFixture, error) {
	if len(ledgers) != context.Attempt+1 || len(ledgers) < 1 {
		return CurriculumFixture{}, fmt.Errorf("attempt-ledger sequence does not reach accepted attempt")
	}
	ledgerDigests := make([]string, len(ledgers))
	for attempt, ledger := range ledgers {
		wantContext := context
		wantContext.Attempt = attempt
		if !equalDrawContexts(ledger.Context, wantContext) || VerifyAttemptLedger(ledger) != nil || attempt < context.Attempt && ledger.Terminal != "rejected" || attempt == context.Attempt && ledger.Terminal != "accepted" {
			return CurriculumFixture{}, fmt.Errorf("invalid attempt ledger %d", attempt)
		}
		ledgerDigests[attempt] = ledger.Digest
	}
	worldDigests := make([]string, 6)
	for slot, view := range curriculum.Worlds {
		if view.Slot != slot || view.Stratum != []string{actionrelationfixturecore.PositiveEffect, actionrelationfixturecore.PositiveEffect, actionrelationfixturecore.Neutral, actionrelationfixturecore.Neutral, actionrelationfixturecore.Adverse, actionrelationfixturecore.Adverse}[slot] || truth.Worlds[slot].WorldDigest != view.Core.Digest || VerifyWorldTruth(truth.Worlds[slot]) != nil {
			return CurriculumFixture{}, fmt.Errorf("invalid fixture world %d", slot)
		}
		worldDigests[slot] = view.Core.Digest
	}
	fixture := CurriculumFixture{
		Panel: context.Panel, Curriculum: context.Curriculum, Family: curriculum.Family, AcceptedAttempt: context.Attempt,
		TrainingCoreDigests: training.CoreDigests, ViewEvidenceDigests: training.ViewEvidenceDigests,
		WorldDigests: worldDigests, AttemptLedgers: ledgerDigests,
	}
	canonical, err := curriculumFixtureWire(fixture)
	if err != nil {
		return CurriculumFixture{}, err
	}
	fixture.Canonical, fixture.Digest = canonical, digestBytes(canonical)
	if err := VerifyCurriculumFixture(fixture); err != nil {
		return CurriculumFixture{}, err
	}
	return fixture, nil
}

func VerifyCurriculumFixture(fixture CurriculumFixture) error {
	if fixture.Panel != "development" && fixture.Panel != "validation" && fixture.Panel != "locked" || fixture.Curriculum < 0 || fixture.Family != fixture.Curriculum%8 || fixture.AcceptedAttempt < 0 || fixture.AcceptedAttempt > 31 || len(fixture.TrainingCoreDigests) != 16 || len(fixture.ViewEvidenceDigests) != 32 || len(fixture.WorldDigests) != 6 || len(fixture.AttemptLedgers) != fixture.AcceptedAttempt+1 {
		return fmt.Errorf("invalid curriculum-fixture shape")
	}
	for _, values := range [][]string{fixture.TrainingCoreDigests, fixture.ViewEvidenceDigests, fixture.WorldDigests, fixture.AttemptLedgers} {
		seen := map[string]bool{}
		for _, value := range values {
			if !digestText(value) || seen[value] {
				return fmt.Errorf("invalid or duplicate curriculum-fixture digest")
			}
			seen[value] = true
		}
	}
	want, err := curriculumFixtureWire(fixture)
	if err != nil || !bytes.Equal(want, fixture.Canonical) || fixture.Digest != digestBytes(fixture.Canonical) || len(fixture.Canonical) > 65536 {
		return fmt.Errorf("invalid curriculum-fixture wire")
	}
	return nil
}

func curriculumFixtureWire(fixture CurriculumFixture) ([]byte, error) {
	return json.Marshal([]any{
		"actionrelation-curriculum-fixture/v1", fixture.Panel, fixture.Curriculum, fixture.Family,
		fixture.AcceptedAttempt, slices.Clone(fixture.TrainingCoreDigests), slices.Clone(fixture.ViewEvidenceDigests),
		slices.Clone(fixture.WorldDigests), slices.Clone(fixture.AttemptLedgers),
	})
}
