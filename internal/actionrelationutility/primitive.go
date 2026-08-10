// Package actionrelationutility drives the ordinary-CUE utility primitives.
// It advances only caller-supplied search work; it does not enumerate a search
// tree, classify a pair, or manufacture evidence authority.
package actionrelationutility

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type PrimitiveResult struct {
	Request string
	Row     string
	Result  bool
}

type TransitionResult struct {
	Request     string
	Row         string
	Outcome     string
	OutputState string
	ResultState actionrelations.State
}

func SearchApplicable(store *unit.Store, meterToken, nodeName, worldDigest, policy string, state actionrelations.State, occurrence actionrelations.Occurrence, token string) (PrimitiveResult, error) {
	stateJSON, occurrenceJSON, err := semanticInputs(state, occurrence)
	if err != nil {
		return PrimitiveResult{}, err
	}
	request, err := putRequest(store, "AR.SearchRequest."+token, "ActionRelationSearchRequest", meterToken, map[string]any{
		"worldDigest": worldDigest, "policy": policy, "nodeUnit": nodeName,
		"state": string(stateJSON), "occurrence": string(occurrenceJSON),
	})
	if err != nil {
		return PrimitiveResult{}, err
	}
	return executePrimitive(store, request, "arSearchApplicable")
}

func StaticFootprint(store *unit.Store, meterToken, nodeName, worldDigest string, state actionrelations.State, a, b actionrelations.Occurrence, token string) (PrimitiveResult, error) {
	stateJSON, aJSON, err := semanticInputs(state, a)
	if err != nil {
		return PrimitiveResult{}, err
	}
	_, bJSON, err := semanticInputs(state, b)
	if err != nil {
		return PrimitiveResult{}, err
	}
	request, err := putRequest(store, "AR.StaticRequest."+token, "ActionStaticFootprintRequest", meterToken, map[string]any{
		"worldDigest": worldDigest, "nodeUnit": nodeName, "state": string(stateJSON),
		"aOccurrence": string(aJSON), "bOccurrence": string(bJSON),
	})
	if err != nil {
		return PrimitiveResult{}, err
	}
	return executePrimitive(store, request, "arStaticFootprint")
}

func SearchApply(store *unit.Store, meterToken, applicabilityRow string, state actionrelations.State, occurrence actionrelations.Occurrence, token string) (TransitionResult, error) {
	stateJSON, occurrenceJSON, err := semanticInputs(state, occurrence)
	if err != nil {
		return TransitionResult{}, err
	}
	request, err := putRequest(store, "AR.TransitionRequest."+token, "ActionRelationTransitionRequest", meterToken, map[string]any{
		"state": string(stateJSON), "occurrence": string(occurrenceJSON), "applicabilityRow": applicabilityRow,
	})
	if err != nil {
		return TransitionResult{}, err
	}
	if err := runPrimitiveTask(store, request, "arSearchApply"); err != nil {
		return TransitionResult{}, err
	}
	u := store.Get(request)
	rowName, stateName := u.GetString("transitionRow"), u.GetString("outputState")
	row := store.Get(rowName)
	if row == nil || row.GetString("outcome") == "" {
		return TransitionResult{}, fmt.Errorf("search transition lacks result row")
	}
	result := TransitionResult{Request: request, Row: rowName, Outcome: row.GetString("outcome"), OutputState: stateName}
	if stateName != "" {
		stateUnit := store.Get(stateName)
		if stateUnit == nil {
			return TransitionResult{}, fmt.Errorf("search transition lacks output state")
		}
		result.ResultState, err = actionrelations.ParseState([]byte(stateUnit.GetString("canonicalObject")))
		if err != nil {
			return TransitionResult{}, err
		}
	}
	return result, nil
}

func semanticInputs(state actionrelations.State, occurrence actionrelations.Occurrence) ([]byte, []byte, error) {
	stateJSON, err := state.CanonicalJSON()
	if err != nil {
		return nil, nil, err
	}
	occurrenceJSON, err := occurrence.CanonicalJSON()
	if err != nil {
		return nil, nil, err
	}
	return stateJSON, occurrenceJSON, nil
}

func putRequest(store *unit.Store, requested, category, meterToken string, slots map[string]any) (string, error) {
	if store == nil || requested == "" || meterToken == "" {
		return "", fmt.Errorf("invalid utility primitive request")
	}
	name := requested
	if store.Has(name) {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%v", category, slots)))
		name += "." + hex.EncodeToString(digest[:8])
		if store.Has(name) {
			return "", fmt.Errorf("utility primitive request name occupied")
		}
	}
	request := unit.New(name)
	request.Set("isA", []string{category, "Anything"})
	request.Set("meterToken", meterToken)
	for key, value := range slots {
		request.Set(key, value)
	}
	store.Put(request)
	return name, nil
}

func executePrimitive(store *unit.Store, requestName, slot string) (PrimitiveResult, error) {
	request := store.Get(requestName)
	if request == nil {
		return PrimitiveResult{}, fmt.Errorf("missing utility primitive request")
	}
	if err := runPrimitiveTask(store, requestName, slot); err != nil {
		return PrimitiveResult{}, err
	}
	rowName := request.GetString("resultRow")
	row := store.Get(rowName)
	if request.GetString("terminal") != "completed" || row == nil {
		return PrimitiveResult{}, fmt.Errorf("utility primitive did not complete")
	}
	return PrimitiveResult{Request: requestName, Row: rowName, Result: row.GetBool("result") || row.GetBool("applicable")}, nil
}

func runPrimitiveTask(store *unit.Store, requestName, slot string) error {
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return err
	}
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: requestName, SlotName: slot})
	if eng.LastError != nil {
		return eng.LastError
	}
	return nil
}
