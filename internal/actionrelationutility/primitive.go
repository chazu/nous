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
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return PrimitiveResult{}, err
	}
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: requestName, SlotName: slot})
	if eng.LastError != nil {
		return PrimitiveResult{}, eng.LastError
	}
	rowName := request.GetString("resultRow")
	row := store.Get(rowName)
	if request.GetString("terminal") != "completed" || row == nil {
		return PrimitiveResult{}, fmt.Errorf("utility primitive did not complete")
	}
	return PrimitiveResult{Request: requestName, Row: rowName, Result: row.GetBool("result") || row.GetBool("applicable")}, nil
}
