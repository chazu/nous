package actionrelationfixture

import (
	"errors"
	"testing"
)

func TestAttemptMeterReservesEventsBeforeItsNamedPredicate(t *testing.T) {
	context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: 0, CurriculumSeed: 851001, Attempt: 0}
	meter, err := BeginAttemptMeter(context)
	if err != nil {
		t.Fatal(err)
	}
	var escaped func() error
	if err := meter.RunPhase(func(reserve func() error) (bool, error) {
		escaped = reserve
		if err := reserve(); err != nil {
			return false, err
		}
		if err := reserve(); err != nil {
			return false, err
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if meter.TotalWork() != 69 || len(meter.phases) != 2 || meter.phases[1].StartWork != 66 || meter.phases[1].EndWork != 69 {
		t.Fatalf("work=%d phases=%+v", meter.TotalWork(), meter.phases)
	}
	if err := escaped(); err == nil {
		t.Fatal("phase reservation remained usable after its semantic event block")
	}
}

func TestAttemptMeterStopsOnRejectedPredicateAndSealsLedger(t *testing.T) {
	context := DrawContext{Panel: "validation", Authority: "validation-public-v1", Curriculum: 0, CurriculumSeed: 852001, Attempt: 0}
	meter, err := BeginAttemptMeter(context)
	if err != nil {
		t.Fatal(err)
	}
	err = meter.RunPhase(func(reserve func() error) (bool, error) {
		if err := reserve(); err != nil {
			return false, err
		}
		return false, nil
	})
	if !errors.Is(err, ErrGeneratorPredicate) {
		t.Fatalf("err=%v", err)
	}
	if err := meter.RunPhase(func(func() error) (bool, error) { return true, nil }); err == nil {
		t.Fatal("meter executed work after failed predicate")
	}
	ledger, err := meter.Close()
	if err != nil || ledger.Terminal != "rejected" || ledger.TotalWork != 68 {
		t.Fatalf("ledger=%+v err=%v", ledger, err)
	}
}

func TestAttemptMeterRejectsBeforeCrossingPhysicalCap(t *testing.T) {
	context := DrawContext{Panel: "development", Authority: "development-public-v1", Curriculum: 0, CurriculumSeed: 851001, Attempt: 1}
	meter, err := BeginAttemptMeter(context)
	if err != nil {
		t.Fatal(err)
	}
	meter.cap = 67
	err = meter.RunPhase(func(reserve func() error) (bool, error) {
		if err := reserve(); err != nil {
			return false, err
		}
		// The next semantic event is refused before it can execute.
		if err := reserve(); err != nil {
			return false, err
		}
		return true, nil
	})
	if !errors.Is(err, ErrGeneratorWorkCap) || meter.TotalWork() != 67 {
		t.Fatalf("err=%v work=%d", err, meter.TotalWork())
	}
}
