// Package actionrelationmatch submits one guarded-relation use request to the
// ordinary actionrelations CUE heuristic.
package actionrelationmatch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type Result struct {
	Request  string
	Terminal string
	Matched  bool
	Barrier  string
}

func Execute(store *unit.Store, artifactName string, state actionrelations.State, a, b actionrelations.Occurrence, token string) (Result, error) {
	return ExecuteMetered(store, artifactName, state, a, b, token, "")
}

func ExecuteMetered(store *unit.Store, artifactName string, state actionrelations.State, a, b actionrelations.Occurrence, token, meterToken string) (Result, error) {
	artifact := store.Get(artifactName)
	if artifact == nil || !store.IsA(artifact.Name, "GuardedActionArtifact") {
		return Result{}, fmt.Errorf("invalid artifact unit")
	}
	stateJSON, err := state.CanonicalJSON()
	if err != nil {
		return Result{}, err
	}
	aJSON, aErr := a.CanonicalJSON()
	bJSON, bErr := b.CanonicalJSON()
	if aErr != nil || bErr != nil || a == b {
		return Result{}, actionrelations.ErrInvalid
	}
	stateDigest, _ := state.Digest()
	aDigest, _ := a.Digest()
	bDigest, _ := b.Digest()
	canonical, _ := json.Marshal([]any{"action-relation-match-request/v1", artifact.GetString("objectDigest"), stateDigest, aDigest, bDigest})
	digestBytes := sha256.Sum256(canonical)
	digest := hex.EncodeToString(digestBytes[:])
	name := "AR.MatchRequest." + token
	if store.Has(name) {
		name += "." + digest[:16]
		if store.Has(name) {
			return Result{}, fmt.Errorf("match request name occupied")
		}
	}
	request := unit.New(name)
	request.Set("isA", []string{"ActionRelationMatchRequest", "Anything"})
	request.Set("canonicalObject", string(canonical))
	request.Set("objectDigest", digest)
	request.Set("artifactUnit", artifact.Name)
	request.Set("state", string(stateJSON))
	request.Set("aOccurrence", string(aJSON))
	request.Set("bOccurrence", string(bJSON))
	if meterToken != "" {
		request.Set("meterToken", meterToken)
	}
	store.Put(request)
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return Result{}, err
	}
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: request.Name, SlotName: "arMatch"})
	if eng.LastError != nil {
		return Result{}, eng.LastError
	}
	return Result{Request: request.Name, Terminal: request.GetString("matchTerminal"), Matched: request.GetBool("matched"), Barrier: request.GetString("useBarrier")}, nil
}
