package kuberepairexp

import (
	"reflect"
	"testing"
)

func TestDevelopmentTrialIsMechanicallyValidAndDeterministic(t *testing.T) {
	first, err := Run("../../domains", "development")
	if err != nil {
		t.Fatal(err)
	}
	if !first.IntegrityValid || first.Training.Tasks != 3 || first.Component.Tasks != 16 {
		t.Fatalf("invalid development report: %#v", first)
	}
	second, err := Run("../../domains", "development")
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := first.JSON()
	secondJSON, _ := second.JSON()
	if !reflect.DeepEqual(firstJSON, secondJSON) {
		t.Fatal("development reports differ")
	}
}

func TestWorstCaseLedgerBounds(t *testing.T) {
	const plans = 584
	if got := commonWork(8, plans) + contextualRankWork(plans) + plans*terminalAttemptWork; got > 231756 {
		t.Fatalf("contextual bound = %d", got)
	}
	if got := commonWork(8, plans) + constraintRankWork(plans) + plans*terminalAttemptWork; got > 329284 {
		t.Fatalf("constraint bound = %d", got)
	}
}
