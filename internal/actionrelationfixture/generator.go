package actionrelationfixture

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationfixturecore"
)

type GeneratedAttempt struct {
	Context           DrawContext
	Training          []actionrelationfixturecore.Case
	TrainingAuthority TrainingAuthority
	Catalogs          SkeletonCatalogs
	Curriculum        Curriculum
	Truth             CurriculumTruth
	Ledger            AttemptLedger
	AttemptLedgers    []AttemptLedger
	Fixture           CurriculumFixture
}

func GenerateAttempt(context DrawContext, prior []AttemptLedger) (GeneratedAttempt, error) {
	if context.Panel != "development" {
		return GeneratedAttempt{}, fmt.Errorf("protected attempt construction requires guarded capability")
	}
	return generateAttempt(context, prior)
}

func generateAttempt(context DrawContext, prior []AttemptLedger) (GeneratedAttempt, error) {
	result := GeneratedAttempt{Context: context}
	if len(prior) != context.Attempt || context.Attempt > 31 {
		return result, fmt.Errorf("prior attempt sequence does not match context")
	}
	priorWork := 0
	for attempt, ledger := range prior {
		want := context
		want.Attempt = attempt
		if !equalDrawContexts(ledger.Context, want) || VerifyAttemptLedger(ledger) != nil || ledger.Terminal != "rejected" {
			return result, fmt.Errorf("invalid prior attempt %d", attempt)
		}
		priorWork += ledger.TotalWork
	}
	if priorWork+66 > GeneratorCurriculumCap {
		return result, fmt.Errorf("curriculum generator work cap exhausted before draw precommit")
	}
	meter, err := beginAttemptMeter(context)
	if err != nil {
		return result, err
	}
	finishFailure := func(cause error) (GeneratedAttempt, error) {
		ledger, closeErr := meter.Close()
		if closeErr == nil {
			result.Ledger = ledger
			result.AttemptLedgers = append(append([]AttemptLedger(nil), prior...), ledger)
		}
		if closeErr != nil {
			return result, fmt.Errorf("%v; close rejected attempt: %w", cause, closeErr)
		}
		return result, cause
	}

	var pool []actionrelationfixturecore.Case
	if err := meter.RunPhase(func(reserve func() error) (bool, error) {
		pool, err = actionrelationfixturecore.BuildTrainingPool(context.Curriculum%8, reserve)
		return err == nil && len(pool) >= 16, err
	}); err != nil {
		return finishFailure(err)
	}

	guard, err := actionrelationfixturecore.LatentGuard(context.Curriculum % 8)
	if err != nil {
		return finishFailure(err)
	}
	if err := meter.RunPhase(func(reserve func() error) (bool, error) {
		err := actionrelationfixturecore.VerifyFamilyGuardMeasured(context.Curriculum%8, guard, reserve)
		return err == nil, err
	}); err != nil {
		return finishFailure(err)
	}

	if err := meter.RunPhase(func(reserve func() error) (bool, error) {
		result.Training, err = actionrelationfixturecore.SelectTrainingPool(pool, reserve)
		if err != nil {
			return false, err
		}
		result.TrainingAuthority, err = SealTrainingAuthorityFromCases(result.Training, reserve)
		return err == nil && len(result.TrainingAuthority.CoreDigests) == 16 && len(result.TrainingAuthority.ViewEvidenceDigests) == 32, err
	}); err != nil {
		return finishFailure(err)
	}

	if err := meter.RunPhase(func(reserve func() error) (bool, error) {
		result.Catalogs, err = BuildSkeletonCatalogs(context.Curriculum%8, reserve)
		passed := err == nil && len(result.Catalogs.Positive) >= 2 && len(result.Catalogs.Neutral) >= 2 && len(result.Catalogs.Adverse) >= 2
		return passed, err
	}); err != nil {
		return finishFailure(err)
	}

	if err := meter.RunPhase(func(reserve func() error) (bool, error) {
		result.Curriculum, err = buildCurriculumFromCatalogs(context, meter.Draws(), result.Catalogs, reserve)
		return err == nil && len(result.Curriculum.Worlds) == 6, err
	}); err != nil {
		return finishFailure(err)
	}

	if err := meter.RunPhase(func(reserve func() error) (bool, error) {
		result.Truth, err = sealCurriculumTruthMeasured(result.Curriculum, reserve)
		return err == nil && len(result.Truth.Worlds) == 6 && result.Truth.Root != "", err
	}); err != nil {
		return finishFailure(err)
	}

	if err := meter.RunPhase(func(reserve func() error) (bool, error) {
		err := preflightGeneratedAttempt(result, reserve)
		return err == nil, err
	}); err != nil {
		return finishFailure(err)
	}

	result.Ledger, err = meter.Close()
	if err != nil {
		return result, err
	}
	if priorWork+result.Ledger.TotalWork > GeneratorCurriculumCap {
		return result, fmt.Errorf("accepted attempt crossed curriculum generator work cap")
	}
	ledgers := append(append([]AttemptLedger(nil), prior...), result.Ledger)
	result.AttemptLedgers = ledgers
	result.Fixture, err = assembleCurriculumFixture(context, result.Curriculum, result.Truth, ledgers, result.TrainingAuthority)
	if err != nil {
		return result, err
	}
	return result, nil
}

func preflightGeneratedAttempt(result GeneratedAttempt, reserve actionrelationfixturecore.WorkReservation) error {
	for _, values := range [][]string{
		result.TrainingAuthority.CoreDigests,
		result.TrainingAuthority.ViewEvidenceDigests,
	} {
		for _, digest := range values {
			if err := reserve(); err != nil {
				return err
			}
			if !digestText(digest) {
				return fmt.Errorf("invalid training preflight digest")
			}
		}
	}
	if len(result.Curriculum.Worlds) != 6 || len(result.Truth.Worlds) != 6 {
		return fmt.Errorf("incomplete world preflight")
	}
	for slot, world := range result.Truth.Worlds {
		if err := reserve(); err != nil {
			return err
		}
		if world.WorldDigest != result.Curriculum.Worlds[slot].Core.Digest || VerifyWorldTruth(world) != nil {
			return fmt.Errorf("world %d failed evidence preflight", slot)
		}
		for _, shard := range world.Shards {
			if err := reserve(); err != nil {
				return err
			}
			if len(shard.Canonical) > maximumTruthShardBytes || shard.Digest != digestBytes(shard.Canonical) {
				return fmt.Errorf("truth shard failed evidence preflight")
			}
		}
	}
	return nil
}

// VerifyGeneratedAttempt reconstructs a sealed attempt without exposing any
// constructor. It is used when a policy worker reopens a supervisor-produced
// read-only fixture bundle after all seed authority has been destroyed.
func VerifyGeneratedAttempt(result GeneratedAttempt) error {
	if err := validateDrawContext(result.Context); err != nil || result.Ledger.Terminal != "accepted" || len(result.AttemptLedgers) != result.Context.Attempt+1 {
		return fmt.Errorf("invalid generated attempt authority")
	}
	if !equalDrawContexts(result.Curriculum.Draws.Context, result.Context) || result.Curriculum.Family != result.Context.Curriculum%8 || result.Curriculum.WithinFamilyOrdinal != result.Context.Curriculum/8 || len(result.Curriculum.Worlds) != 6 {
		return fmt.Errorf("invalid generated curriculum")
	}
	if len(result.Training) != actionrelationfixturecore.TrainingCount {
		return fmt.Errorf("invalid generated training cardinality")
	}
	wantTraining, err := SealTrainingAuthorityFromCases(result.Training, nil)
	if err != nil || !slices.Equal(wantTraining.CoreDigests, result.TrainingAuthority.CoreDigests) || !slices.Equal(wantTraining.ViewEvidenceDigests, result.TrainingAuthority.ViewEvidenceDigests) {
		return fmt.Errorf("generated training authority changed")
	}
	for attempt, ledger := range result.AttemptLedgers {
		wantContext := result.Context
		wantContext.Attempt = attempt
		if !equalDrawContexts(ledger.Context, wantContext) || VerifyAttemptLedger(ledger) != nil || attempt < result.Context.Attempt && ledger.Terminal != "rejected" || attempt == result.Context.Attempt && ledger.Terminal != "accepted" {
			return fmt.Errorf("invalid generated attempt ledger %d", attempt)
		}
	}
	if result.Ledger.Digest != result.AttemptLedgers[len(result.AttemptLedgers)-1].Digest {
		return fmt.Errorf("generated terminal ledger changed")
	}
	wantTruth, err := sealCurriculumTruthMeasured(result.Curriculum, nil)
	if err != nil || wantTruth.Root != result.Truth.Root {
		return fmt.Errorf("generated scorer truth changed")
	}
	wantFixture, err := assembleCurriculumFixture(result.Context, result.Curriculum, result.Truth, result.AttemptLedgers, result.TrainingAuthority)
	if err != nil || wantFixture.Digest != result.Fixture.Digest || !bytes.Equal(wantFixture.Canonical, result.Fixture.Canonical) {
		return fmt.Errorf("generated curriculum fixture changed")
	}
	return nil
}
