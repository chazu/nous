package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationoracle"
)

type replayOccurrence struct {
	canonical []byte
	action    []byte
}

// VerifyCertificateDecisionSemantics independently reconstructs the exact
// charged proof rows and certificate decision from semantic preimages.
func VerifyCertificateDecisionSemantics(stateJSON, aOccurrenceJSON, bOccurrenceJSON []byte, operationRows []string, result, certificateDigest string, certificateCanonical []byte) error {
	if actionrelationoracle.ValidateState(stateJSON) != nil {
		return fmt.Errorf("certificate decision lacks state preimage")
	}
	a, err := parseReplayOccurrence(aOccurrenceJSON)
	if err != nil {
		return err
	}
	b, err := parseReplayOccurrence(bOccurrenceJSON)
	if err != nil || bytes.Compare(a.canonical, b.canonical) >= 0 {
		return fmt.Errorf("certificate decision pair is not canonical and distinct")
	}
	stateDigest, aDigest, bDigest := shaHex(stateJSON), shaHex(aOccurrenceJSON), shaHex(bOccurrenceJSON)
	aInitial, err := actionrelationoracle.Apply(stateJSON, a.action)
	if err != nil {
		return err
	}
	bInitial, err := actionrelationoracle.Apply(stateJSON, b.action)
	if err != nil {
		return err
	}
	rowDigest := func(row []any) string {
		canonical, _ := json.Marshal(row)
		return shaHex(canonical)
	}
	appRow := func(inputState, occurrence string, applicable bool) string {
		return rowDigest([]any{"action-applicability-row/v1", inputState, occurrence, applicable, "valid"})
	}
	transitionRow := func(inputState, occurrence, applicability string, transition actionrelationoracle.Transition) string {
		outcome, output := "inapplicable", zeroObjectDigest
		if transition.Applicable {
			outcome, output = "applied", shaHex(transition.State)
		}
		return rowDigest([]any{"action-transition-row/v1", inputState, occurrence, applicability, output, outcome})
	}
	aApp := appRow(stateDigest, aDigest, aInitial.Applicable)
	bApp := appRow(stateDigest, bDigest, bInitial.Applicable)
	wantRows := []string{aApp, bApp, transitionRow(stateDigest, aDigest, aApp, aInitial), transitionRow(stateDigest, bDigest, bApp, bInitial)}
	if aInitial.Applicable && bInitial.Applicable {
		bAfterA, err := actionrelationoracle.Apply(aInitial.State, b.action)
		if err != nil {
			return err
		}
		aAfterB, err := actionrelationoracle.Apply(bInitial.State, a.action)
		if err != nil {
			return err
		}
		aState, bState := shaHex(aInitial.State), shaHex(bInitial.State)
		bCrossApp := appRow(aState, bDigest, bAfterA.Applicable)
		aCrossApp := appRow(bState, aDigest, aAfterB.Applicable)
		wantRows = append(wantRows, bCrossApp, aCrossApp, transitionRow(aState, bDigest, bCrossApp, bAfterA), transitionRow(bState, aDigest, aCrossApp, aAfterB))
		if bAfterA.Applicable && aAfterB.Applicable {
			abDigest, baDigest := shaHex(bAfterA.State), shaHex(aAfterB.State)
			wantRows = append(wantRows, rowDigest([]any{"action-state-equality-row/v1", abDigest, baDigest, bytes.Equal(bAfterA.State, aAfterB.State), "valid"}))
		}
	}
	if !slices.Equal(operationRows, wantRows) {
		return fmt.Errorf("certificate operation rows do not independently reconstruct")
	}
	observation, err := actionrelationoracle.Observe(stateJSON, a.action, b.action)
	if err != nil {
		return err
	}
	if observation.Label == "commutes" {
		if result != "certified" || !digestText(certificateDigest) {
			return fmt.Errorf("commuting decision was not certified")
		}
		return verifyReplayCertificate(certificateCanonical, certificateDigest, stateDigest, aDigest, bDigest, a, b, observation)
	}
	if result != "not-certified" || certificateDigest != zeroObjectDigest || len(certificateCanonical) != 0 {
		return fmt.Errorf("noncommuting decision was falsely certified")
	}
	return nil
}

func parseReplayOccurrence(data []byte) (replayOccurrence, error) {
	var row []json.RawMessage
	var version string
	var ordinal int
	if json.Unmarshal(data, &row) != nil || len(row) != 3 || json.Unmarshal(row[0], &version) != nil || version != "action-occurrence/v1" || json.Unmarshal(row[2], &ordinal) != nil || ordinal < 0 || ordinal >= 8 || actionrelationoracle.ValidateAction(row[1]) != nil {
		return replayOccurrence{}, fmt.Errorf("invalid semantic occurrence")
	}
	want, _ := json.Marshal(row)
	if !bytes.Equal(want, data) {
		return replayOccurrence{}, fmt.Errorf("noncanonical semantic occurrence")
	}
	return replayOccurrence{canonical: slices.Clone(data), action: slices.Clone(row[1])}, nil
}

func verifyReplayCertificate(data []byte, certificateDigest, stateDigest, takenDigest, sleeperDigest string, taken, sleeper replayOccurrence, observation actionrelationoracle.Observation) error {
	var row []json.RawMessage
	if json.Unmarshal(data, &row) != nil || len(row) != 10 || rawText(row[0]) != "local-diamond-certificate/v1" || shaHex(data) != certificateDigest {
		return fmt.Errorf("sleep propagation lacks certificate preimage")
	}
	aDigest, bDigest := takenDigest, sleeperDigest
	abState, baState := observation.AB, observation.BA
	if bytes.Compare(taken.canonical, sleeper.canonical) > 0 {
		aDigest, bDigest = bDigest, aDigest
		abState, baState = baState, abState
	}
	var equal bool
	if rawText(row[1]) != stateDigest || rawText(row[2]) != aDigest || rawText(row[3]) != bDigest || !digestText(rawText(row[4])) || rawText(row[5]) != shaHex(abState) || rawText(row[6]) != shaHex(baState) || json.Unmarshal(row[7], &equal) != nil || !equal || rawText(row[8]) != aDigest || !digestText(rawText(row[9])) {
		return fmt.Errorf("certificate does not reconstruct its independent diamond")
	}
	want, _ := json.Marshal(row)
	if !bytes.Equal(want, data) {
		return fmt.Errorf("noncanonical certificate preimage")
	}
	return nil
}
