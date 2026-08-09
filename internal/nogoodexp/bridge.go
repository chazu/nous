package nogoodexp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

type Disposition struct {
	Status      string
	Request     string
	Artifact    string
	Binding     string
	Completion  string
	Certificate string
	TasksPopped int
	Store       *unit.Store
}

func ConsiderPrune(domainsDir string, problemJSON []byte, decision nogoods.Literal, artifact *FrozenArtifact) (Disposition, error) {
	if _, err := nogoods.ParseProblem(problemJSON); err != nil {
		return Disposition{}, err
	}
	store := unit.NewStore()
	previousDomainsDir := seed.DomainsDir
	seed.DomainsDir = domainsDir
	err := seed.LoadDomain(store, "nogoods")
	seed.DomainsDir = previousDomainsDir
	if err != nil {
		return Disposition{}, err
	}
	if got := store.Examples("Heuristic"); !slices.Equal(got, []string{"Heuristic", "NG-H-ConsiderPrune"}) {
		return Disposition{}, fmt.Errorf("unexpected bridge heuristic profile %v", got)
	}
	artifactDigest := "empty"
	if artifact != nil {
		if err := installArtifact(store, *artifact); err != nil {
			return Disposition{}, err
		}
		artifactDigest = artifact.Digest
	}
	requestMaterial := struct {
		Problem  json.RawMessage `json:"problem"`
		Decision nogoods.Literal `json:"decision"`
		Artifact string          `json:"artifact"`
	}{Problem: json.RawMessage(problemJSON), Decision: decision, Artifact: artifactDigest}
	encodedRequest, _ := json.Marshal(requestMaterial)
	requestHash := sha256.Sum256(encodedRequest)
	requestDigest := hex.EncodeToString(requestHash[:])
	requestName := "NG.Request." + requestDigest[:24]
	request := unit.New(requestName)
	request.Set("isA", []string{"NogoodRequest", "Anything"})
	request.Set("problem", string(problemJSON))
	request.Set("decisionVariable", decision.Variable)
	request.Set("decisionColor", decision.Color)
	request.Set("requestDigest", requestDigest)
	request.Set("artifactStoreDigest", artifactDigest)
	store.Put(request)

	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return Disposition{}, err
	}
	ag.Push(&agenda.Task{Priority: 800, UnitName: requestName, SlotName: "ngConsiderPrune", Reasons: []string{"Consider frozen blocked-pair artifact"}})
	popped := 0
	for ag.Len() > 0 {
		if popped >= TrainingTaskCap {
			return Disposition{}, fmt.Errorf("bridge task cap exceeded")
		}
		task := ag.Pop()
		if task == nil || task.UnitName != requestName {
			return Disposition{}, fmt.Errorf("cross-request or empty bridge task")
		}
		eng.WorkOnTask(task)
		if len(eng.VM.DeletedUnits) != 0 {
			return Disposition{}, fmt.Errorf("bridge deleted units")
		}
		popped++
	}
	dispositionName := request.GetString("dispositionUnit")
	disposition := store.Get(dispositionName)
	if disposition == nil || !disposition.GetBool("sealed") || disposition.GetString("request") != requestName || disposition.GetString("requestDigest") != requestDigest {
		return Disposition{}, fmt.Errorf("missing or invalid sealed disposition")
	}
	status := disposition.GetString("status")
	if status != "resume" && status != "propose-prune" && status != "bridge-invalid" {
		return Disposition{}, fmt.Errorf("invalid disposition status %q", status)
	}
	result := Disposition{Status: status, Request: requestName, Artifact: disposition.GetString("artifact"), Binding: disposition.GetString("binding"), Completion: disposition.GetString("completion"), Certificate: disposition.GetString("certificate"), TasksPopped: popped, Store: store}
	if status == "propose-prune" {
		if artifact == nil || result.Artifact != artifact.Name || store.Get(result.Binding) == nil || store.Get(result.Completion) == nil || store.Get(result.Certificate) == nil || !store.Get(result.Certificate).GetBool("valid") {
			return Disposition{}, fmt.Errorf("incomplete prune proposal")
		}
	}
	return result, nil
}
