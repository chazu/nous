// Package actionrelationcertify submits one public certificate request to the
// ordinary actionrelations CUE heuristic.
package actionrelationcertify

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

type Result struct {
	Request     string
	Terminal    string
	Certificate string
	Attempt     string
}

func Execute(store *unit.Store, state actionrelations.State, a, b actionrelations.Occurrence, witness []byte, operationRoot, token string) (Result, error) {
	request, err := Begin(store, state, a, b, witness, token, "")
	if err != nil {
		return Result{}, err
	}
	if err := RunInitial(store, request); err != nil {
		return Result{}, err
	}
	if err := RunCross(store, request); err != nil {
		return Result{}, err
	}
	if err := RunEquality(store, request); err != nil {
		return Result{}, err
	}
	return Complete(store, request, operationRoot)
}

func Begin(store *unit.Store, state actionrelations.State, a, b actionrelations.Occurrence, witness []byte, token, meterToken string) (string, error) {
	stateJSON, err := state.CanonicalJSON()
	if err != nil {
		return "", err
	}
	a, b, err = actionrelations.CanonicalPair(a, b)
	if err != nil || a == b {
		return "", actionrelations.ErrInvalid
	}
	aJSON, _ := a.CanonicalJSON()
	bJSON, _ := b.CanonicalJSON()
	name := "AR.CertificateRequest." + token
	if store.Has(name) {
		digest := sha256.Sum256(append(append(append(stateJSON, aJSON...), bJSON...), witness...))
		name += "." + hex.EncodeToString(digest[:8])
		if store.Has(name) {
			return "", fmt.Errorf("certificate request name occupied")
		}
	}
	request := unit.New(name)
	request.Set("isA", []string{"ActionRelationCertificateRequest", "Anything"})
	request.Set("state", string(stateJSON))
	request.Set("aOccurrence", string(aJSON))
	request.Set("bOccurrence", string(bJSON))
	request.Set("witness", string(witness))
	if meterToken != "" {
		request.Set("meterToken", meterToken)
	}
	store.Put(request)
	return request.Name, nil
}

func RunInitial(store *unit.Store, request string) error {
	return executeStage(store, request, "arCertifyInitial", "initialTerminal")
}

func CrossOperationCodes(store *unit.Store, request string) []uint8 {
	u := store.Get(request)
	if u != nil && u.GetString("afterAUnit") != "" && u.GetString("afterBUnit") != "" {
		return []uint8{13, 13, 12, 12}
	}
	return nil
}

func RunCross(store *unit.Store, request string) error {
	return executeStage(store, request, "arCertifyCross", "crossTerminal")
}

func EqualityOperationCodes(store *unit.Store, request string) []uint8 {
	u := store.Get(request)
	if u != nil && u.GetString("abUnit") != "" && u.GetString("baUnit") != "" {
		return []uint8{14}
	}
	return nil
}

func RunEquality(store *unit.Store, request string) error {
	return executeStage(store, request, "arCertifyEquality", "equalityTerminal")
}

func Complete(store *unit.Store, request, operationRoot string) (Result, error) {
	u := store.Get(request)
	if u == nil {
		return Result{}, fmt.Errorf("missing certificate request")
	}
	u.Set("operationRoot", operationRoot)
	if err := executeStage(store, request, "arCertifyAssemble", "certificateTerminal"); err != nil {
		return Result{}, err
	}
	return Result{Request: request, Terminal: u.GetString("certificateTerminal"), Certificate: u.GetString("certificateUnit"), Attempt: u.GetString("certificateAttemptUnit")}, nil
}

func executeStage(store *unit.Store, request, slot, terminalSlot string) error {
	u := store.Get(request)
	if u == nil {
		return fmt.Errorf("missing certificate request")
	}
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return err
	}
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: request, SlotName: slot})
	if eng.LastError != nil {
		return eng.LastError
	}
	if u.GetString(terminalSlot) == "" {
		return fmt.Errorf("certificate stage %s did not complete", slot)
	}
	return nil
}
