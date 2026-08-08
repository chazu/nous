package kuberepairexp

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/kuberepairfixture"
)

func TestDevelopmentTrialIsMechanicallyValidAndDeterministic(t *testing.T) {
	first, err := Run("../../domains", "development")
	if err != nil {
		t.Fatal(err)
	}
	if !first.IntegrityValid || first.Training.Tasks != 3 || first.Component.Tasks != 12 || first.TwoFeature.Tasks != 6 || first.ThreeFeature.Tasks != 6 {
		t.Fatalf("invalid development report: %#v", first)
	}
	if first.Training.CandidatesCreated != 568 || first.Training.TerminalCalls != 375 || first.Training.CallLogSHA256 != "2571d8b818b56b5008c6c41b6a899646c209380d6992dd25b55e48830d5f7b18" {
		t.Fatalf("unreconciled training accounting: %#v", first.Training)
	}
	wantPhaseAIDs := []string{"label-761001-1", "label-761002-3", "label-761003-7", "label-761004-0", "no-solution-761005", "unrelated-761006"}
	for index, want := range wantPhaseAIDs {
		if first.PhaseA.Tasks[index].ID != want {
			t.Fatalf("phase A task %d = %q, want %q", index, first.PhaseA.Tasks[index].ID, want)
		}
	}
	if first.Outcome != "valid-null" || first.Power == nil || first.Power.Accepted || first.Power.Power != 0 {
		t.Fatalf("unexpected development feasibility result: outcome=%s power=%#v", first.Outcome, first.Power)
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

func TestProtectedPanelsRefuseAfterFailedPowerGate(t *testing.T) {
	if _, err := Run("../../domains", "validation"); err == nil {
		t.Fatal("validation ran despite failed development power gate")
	}
	if _, err := Run("../../domains", "locked"); err == nil {
		t.Fatal("locked panel bypassed guarded entrypoint")
	}
}

func TestWorstCaseEligiblePlanAndTerminalCallBounds(t *testing.T) {
	caseData, err := kuberepairfixture.Seed()
	if err != nil {
		t.Fatal(err)
	}
	plans, events, err := enumeratePlans(caseData)
	if err != nil {
		t.Fatal(err)
	}
	if len(caseData.Edits) != 8 || len(plans) > 400 || 1+len(plans) > 401 {
		t.Fatalf("worst case edits=%d eligible=%d calls=%d", len(caseData.Edits), len(plans), 1+len(plans))
	}
	if events.PlansEmitted != len(plans) || events.PlanApplications < len(plans) {
		t.Fatalf("unreconciled construction events: %#v", events)
	}
}

func TestTrainingCreditValidationRejectsSelfConsistentProvenanceForgery(t *testing.T) {
	cases, err := kuberepairfixture.Training(750001)
	if err != nil {
		t.Fatal(err)
	}
	caseData := withOpaqueHandle(cases[0], rootHex(750001), 0)
	store, selected, _, cleanup, err := discover("../../domains", caseData)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	creditEngine := engine.New(store, agenda.New())
	creditEngine.Out = io.Discard
	creditEngine.VM.Out = io.Discard
	creditEngine.MutConfig.Enabled = false
	creditEngine.MaxCycles = 16
	if err := creditEngine.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	program := store.Get(selected[0])
	component := store.Get(program.GetStrings("components")[0])
	validate := func() error {
		return validateTrainingCredit(store, program, component,
			component.GetString("creditFeatureKey"), component.GetString("creditRelationKey"),
			component.GetString("creditFeatureSubject"), component.GetString("creditRelationSubject"))
	}
	if err := validate(); err != nil {
		t.Fatalf("valid provenance rejected: %v", err)
	}

	originalDecision := program.GetString("creditDecision")
	program.Set("creditDecision", "sha256:v1:forged")
	if err := validate(); err == nil {
		t.Fatal("forged concrete decision accepted")
	}
	program.Set("creditDecision", originalDecision)

	originalCreditors := program.GetStrings("creditors")
	program.Set("creditors", []string{"H-ForgedSynthesis", component.Name})
	if err := validate(); err == nil {
		t.Fatal("forged synthesis creditor accepted")
	}
	program.Set("creditors", originalCreditors)

	originalSemantics := program.GetStrings("semanticSequence")
	program.Set("semanticSequence", []string{"forged-opcode"})
	if err := validate(); err == nil {
		t.Fatal("forged semantic sequence accepted")
	}
	program.Set("semanticSequence", originalSemantics)
}
