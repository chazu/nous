package actionrelationfixture

import (
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/actionrelationfixturecore"
	"github.com/chazu/nous/internal/actionrelationoracle"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

const zeroActionRelationDigest = "0000000000000000000000000000000000000000000000000000000000000000"

type TrainingAuthority struct {
	CoreDigests         []string
	ViewEvidenceDigests []string
}

func SealTrainingAuthority(family int) (TrainingAuthority, error) {
	training, err := actionrelationfixturecore.TrainingFamily(family)
	if err != nil || len(training) != 16 {
		return TrainingAuthority{}, fmt.Errorf("training family %d: %w", family, err)
	}
	authority := TrainingAuthority{CoreDigests: make([]string, len(training)), ViewEvidenceDigests: make([]string, 0, 2*len(training))}
	seen := map[string]bool{}
	for ordinal, testCase := range training {
		observation, err := sealTrainingObservation(testCase)
		if err != nil {
			return TrainingAuthority{}, fmt.Errorf("training core %d: %w", ordinal, err)
		}
		observationDigest := digestBytes(observation)
		if seen[observationDigest] {
			return TrainingAuthority{}, fmt.Errorf("duplicate training core %d", ordinal)
		}
		seen[observationDigest] = true
		authority.CoreDigests[ordinal] = observationDigest
		views, err := actionrelationfixturecore.Views(testCase)
		if err != nil || len(views) != 2 {
			return TrainingAuthority{}, fmt.Errorf("training views %d: %w", ordinal, err)
		}
		for bank, view := range views {
			if view.Bank != bank || view.SemanticWorldDigest == "" {
				return TrainingAuthority{}, fmt.Errorf("training view %d/%d changed bank order", ordinal, bank)
			}
			wire, _ := json.Marshal([]any{"action-view-evidence/v1", observationDigest, view.Digest, view.ProofDigest})
			authority.ViewEvidenceDigests = append(authority.ViewEvidenceDigests, digestBytes(wire))
		}
	}
	return authority, nil
}

type sealedTransition struct {
	digest  string
	next    actionrelations.State
	outcome string
}

func sealTrainingObservation(testCase actionrelationfixturecore.Case) ([]byte, error) {
	state, err := actionrelations.ParseState(testCase.State)
	if err != nil {
		return nil, err
	}
	a, err := actionrelations.ParseOccurrence(testCase.AOccurrence)
	if err != nil {
		return nil, err
	}
	b, err := actionrelations.ParseOccurrence(testCase.BOccurrence)
	if err != nil {
		return nil, err
	}
	left, right, err := actionrelations.CanonicalPair(a, b)
	if err != nil || left != a || right != b || a == b {
		return nil, fmt.Errorf("noncanonical training pair")
	}
	aInitial, err := sealTransition(state, a)
	if err != nil {
		return nil, err
	}
	bInitial, err := sealTransition(state, b)
	if err != nil {
		return nil, err
	}
	var bAfterA, aAfterB *sealedTransition
	if aInitial.outcome == "applied" {
		row, err := sealTransition(aInitial.next, b)
		if err != nil {
			return nil, err
		}
		bAfterA = &row
	}
	if bInitial.outcome == "applied" {
		row, err := sealTransition(bInitial.next, a)
		if err != nil {
			return nil, err
		}
		aAfterB = &row
	}
	var abDigest, baDigest any
	if bAfterA != nil && aAfterB != nil && bAfterA.outcome == "applied" && aAfterB.outcome == "applied" {
		abText, _ := bAfterA.next.Digest()
		baText, _ := aAfterB.next.Digest()
		abDigest, baDigest = abText, baText
	}
	stateJSON, _ := state.CanonicalJSON()
	aAction, _ := a.Action.CanonicalJSON()
	bAction, _ := b.Action.CanonicalJSON()
	observation, err := actionrelationoracle.Observe(stateJSON, aAction, bAction)
	if err != nil || observation.Label != testCase.Label {
		return nil, fmt.Errorf("training label changed: got %q want %q", observation.Label, testCase.Label)
	}
	stateDigest, _ := state.Digest()
	aDigest, _ := a.Digest()
	bDigest, _ := b.Digest()
	return json.Marshal([]any{
		"action-pair-observation/v1", stateDigest, aDigest, bDigest,
		aInitial.digest, bInitial.digest, optionalTransitionDigest(bAfterA), optionalTransitionDigest(aAfterB),
		abDigest, baDigest, observation.Label,
	})
}

func sealTransition(state actionrelations.State, occurrence actionrelations.Occurrence) (sealedTransition, error) {
	stateDigest, _ := state.Digest()
	occurrenceDigest, _ := occurrence.Digest()
	applicable, err := actionrelations.Applicable(state, occurrence.Action)
	if err != nil {
		return sealedTransition{}, err
	}
	applicability, _ := json.Marshal([]any{"action-applicability-row/v1", stateDigest, occurrenceDigest, applicable, "valid"})
	next, outcome, err := actionrelations.Apply(state, occurrence.Action)
	if err != nil || applicable != (outcome == "applied") {
		return sealedTransition{}, fmt.Errorf("transition applicability changed")
	}
	resultDigest := zeroActionRelationDigest
	if outcome == "applied" {
		resultDigest, _ = next.Digest()
	}
	transition, _ := json.Marshal([]any{"action-transition-row/v1", stateDigest, occurrenceDigest, digestBytes(applicability), resultDigest, outcome})
	return sealedTransition{digest: digestBytes(transition), next: next, outcome: outcome}, nil
}

func optionalTransitionDigest(row *sealedTransition) any {
	if row == nil {
		return nil
	}
	return row.digest
}
