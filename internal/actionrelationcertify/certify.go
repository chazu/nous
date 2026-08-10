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
}

func Execute(store *unit.Store, state actionrelations.State, a, b actionrelations.Occurrence, witness []byte, operationRoot, token string) (Result, error) {
	stateJSON, err := state.CanonicalJSON()
	if err != nil {
		return Result{}, err
	}
	a, b, err = actionrelations.CanonicalPair(a, b)
	if err != nil || a == b {
		return Result{}, actionrelations.ErrInvalid
	}
	aJSON, _ := a.CanonicalJSON()
	bJSON, _ := b.CanonicalJSON()
	name := "AR.CertificateRequest." + token
	if store.Has(name) {
		digest := sha256.Sum256(append(append(append(stateJSON, aJSON...), bJSON...), witness...))
		name += "." + hex.EncodeToString(digest[:8])
		if store.Has(name) {
			return Result{}, fmt.Errorf("certificate request name occupied")
		}
	}
	request := unit.New(name)
	request.Set("isA", []string{"ActionRelationCertificateRequest", "Anything"})
	request.Set("state", string(stateJSON))
	request.Set("aOccurrence", string(aJSON))
	request.Set("bOccurrence", string(bJSON))
	request.Set("witness", string(witness))
	request.Set("operationRoot", operationRoot)
	store.Put(request)
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return Result{}, err
	}
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: request.Name, SlotName: "arCertify"})
	if eng.LastError != nil {
		return Result{}, eng.LastError
	}
	return Result{Request: request.Name, Terminal: request.GetString("certificateTerminal"), Certificate: request.GetString("certificateUnit")}, nil
}
