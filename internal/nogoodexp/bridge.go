package nogoodexp

import (
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
	Status, Request, Artifact, Binding, Completion, Certificate, Barrier, Proposal string
	TasksPopped                                                                    int
	Store                                                                          *unit.Store
}

type BridgeExecution struct {
	base        *unit.Store
	profileHash string
	artifact    *FrozenArtifact
	authority   *ArtifactAuthority
	nextRequest int
}

func NewBridgeExecution(domainsDir string, artifact *FrozenArtifact, authority *ArtifactAuthority) (*BridgeExecution, error) {
	store := unit.NewStore()
	previousDomainsDir := seed.DomainsDir
	seed.DomainsDir = domainsDir
	err := seed.LoadDomain(store, "nogoods")
	seed.DomainsDir = previousDomainsDir
	if err != nil {
		return nil, err
	}
	profileHash, err := auditBridgeProfile(store)
	if err != nil {
		return nil, err
	}
	if artifact != nil {
		if err := installArtifact(store, *artifact, authority); err != nil {
			return nil, err
		}
		copyArtifact := *artifact
		artifact = &copyArtifact
	}
	return &BridgeExecution{base: store, profileHash: profileHash, artifact: artifact, authority: authority, nextRequest: 1}, nil
}

func auditBridgeProfile(store *unit.Store) (string, error) {
	if got := store.Examples("Heuristic"); !slices.Equal(got, []string{"Heuristic", "NG-H-ConsiderPrune"}) {
		return "", fmt.Errorf("unexpected bridge heuristic profile %v", got)
	}
	category, lane := store.Get("Heuristic"), store.Get("NG-H-ConsiderPrune")
	if category == nil || lane == nil {
		return "", fmt.Errorf("missing bridge profile unit")
	}
	profile := struct {
		StoreDigest string            `json:"store_digest"`
		Programs    map[string]string `json:"programs"`
		Shape       map[string]bool   `json:"shape"`
	}{Programs: map[string]string{}, Shape: map[string]bool{}}
	storeBytes, err := store.CanonicalJSON()
	if err != nil {
		return "", err
	}
	profile.StoreDigest = digestBytes(storeBytes)
	for _, entry := range []struct {
		name string
		u    *unit.Unit
	}{{"Heuristic", category}, {"NG-H-ConsiderPrune", lane}} {
		for _, slot := range append(unit.IfPartSlots(), unit.ThenPartSlots()...) {
			program := entry.u.GetString(slot)
			key := entry.name + "." + slot
			profile.Shape[key] = program != ""
			if program != "" {
				profile.Programs[key] = digestBytes([]byte(program))
			}
		}
	}
	if len(profile.Programs) != 2 || profile.Programs["NG-H-ConsiderPrune.ifWorkingOnTask"] == "" || profile.Programs["NG-H-ConsiderPrune.thenCompute"] == "" {
		return "", fmt.Errorf("bridge program shape drifted")
	}
	return digestJSON(profile), nil
}

func ConsiderPrune(domainsDir string, problemJSON []byte, decision nogoods.Literal, artifact *FrozenArtifact, authority *ArtifactAuthority) (Disposition, error) {
	execution, err := NewBridgeExecution(domainsDir, artifact, authority)
	if err != nil {
		return Disposition{}, err
	}
	return execution.Consider(problemJSON, decision)
}

func (execution *BridgeExecution) Consider(problemJSON []byte, decision nogoods.Literal) (Disposition, error) {
	problem, err := nogoods.ParseProblem(problemJSON)
	if err != nil {
		return Disposition{}, err
	}
	store := cloneStore(execution.base)
	requestNumber := execution.nextRequest
	execution.nextRequest++
	artifactStoreDigest := digestJSON([]string{})
	if execution.artifact != nil {
		artifactStoreDigest = execution.artifact.Digest
	}
	immutableDomains := make([][]int, len(problem.Variables))
	reducedDomains := make([][]int, len(problem.Variables))
	for index, variable := range problem.Variables {
		immutableDomains[index] = slices.Clone(variable.Domain)
		reducedDomains[index] = slices.Clone(variable.Domain)
	}
	reducedDomains[decision.Variable] = []int{decision.Color}
	for variable := range reducedDomains {
		if variable != decision.Variable && problem.EdgePresent(variable, decision.Variable) {
			reducedDomains[variable] = slices.DeleteFunc(reducedDomains[variable], func(color int) bool { return color == decision.Color })
		}
	}
	targetDigest := digestBytes(problemJSON)
	decisionDigest := digestJSON(decision)
	assignmentDigest := digestJSON(problem.Assignment)
	immutableDigest := digestJSON(immutableDomains)
	reducedDigest := digestJSON(reducedDomains)
	conflictDigest := digestJSON([]string{})
	requestMaterial := struct {
		Profile, Target, Decision, Assignment, Immutable, Reduced, Conflicts, Artifacts string
		Number                                                                          int
	}{execution.profileHash, targetDigest, decisionDigest, assignmentDigest, immutableDigest, reducedDigest, conflictDigest, artifactStoreDigest, requestNumber}
	requestDigest := digestJSON(requestMaterial)
	requestName := fmt.Sprintf("NG.Request.%08d.%s", requestNumber, requestDigest[:16])
	request := unit.New(requestName)
	request.Set("isA", []string{"NogoodRequest", "Anything"})
	request.Set("problem", string(problemJSON))
	request.Set("decisionVariable", decision.Variable)
	request.Set("decisionColor", decision.Color)
	request.Set("requestNumber", requestNumber)
	request.Set("policyProfileHash", execution.profileHash)
	request.Set("targetDigest", targetDigest)
	request.Set("decisionDigest", decisionDigest)
	request.Set("assignmentDigest", assignmentDigest)
	request.Set("immutableDomainDigest", immutableDigest)
	request.Set("reducedDomainDigest", reducedDigest)
	request.Set("exactConflictStoreDigest", conflictDigest)
	request.Set("artifactStoreDigest", artifactStoreDigest)
	request.Set("requestDigest", requestDigest)
	store.Put(request)

	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
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
		if task == nil {
			return Disposition{}, fmt.Errorf("empty bridge task")
		}
		target := store.Get(task.UnitName)
		if task.UnitName != requestName || target == nil || target.GetString("requestDigest") != requestDigest {
			return Disposition{}, fmt.Errorf("cross-request or stale bridge task")
		}
		eng.WorkOnTask(task)
		if len(eng.VM.DeletedUnits) != 0 {
			return Disposition{}, fmt.Errorf("bridge deleted units")
		}
		popped++
	}
	disposition := store.Get(request.GetString("dispositionUnit"))
	if disposition == nil {
		return Disposition{}, fmt.Errorf("missing sealed disposition")
	}
	status := disposition.GetString("status")
	result := Disposition{Status: status, Request: requestName, Artifact: disposition.GetString("artifact"), Binding: disposition.GetString("binding"), Completion: disposition.GetString("completion"), Certificate: disposition.GetString("certificate"), Barrier: disposition.GetString("barrier"), Proposal: disposition.GetString("proposal"), TasksPopped: popped, Store: store}
	if status != "resume" && status != "propose-prune" && status != "bridge-invalid" {
		return Disposition{}, fmt.Errorf("invalid disposition status %q", status)
	}
	if status == "propose-prune" {
		checks := [6]bool{
			disposition.GetBool("sealed") && disposition.GetString("request") == requestName,
			store.Get(result.Barrier) != nil && store.Get(result.Barrier).GetBool("sealed"),
			execution.artifact != nil && execution.authority != nil && execution.authority.accepts(*execution.artifact) && result.Artifact == execution.artifact.Name,
			store.Get(result.Completion) != nil && store.Get(result.Completion).GetBool("conflict"),
			disposition.GetString("requestDigest") == requestDigest && disposition.GetString("targetDigest") == targetDigest && disposition.GetString("decisionDigest") == decisionDigest,
			disposition.GetString("referencedUnitSetDigest") == referenceDigest(result),
		}
		for index, passed := range checks {
			if !passed {
				return Disposition{}, fmt.Errorf("adapter proposal check %d failed", index+1)
			}
		}
	}
	return result, nil
}

func referenceDigest(result Disposition) string {
	return digestJSON([]string{result.Request, result.Artifact, result.Binding, result.Completion, result.Certificate, result.Barrier, result.Proposal})
}

func cloneStore(source *unit.Store) *unit.Store {
	target := unit.NewStore()
	for _, name := range source.All() {
		original := source.Get(name)
		clone := unit.New(name)
		for slot, value := range original.Slots {
			clone.Set(slot, cloneSlotValue(value))
		}
		target.Put(clone)
	}
	return target
}

func cloneSlotValue(value any) any {
	switch typed := value.(type) {
	case []string:
		return slices.Clone(typed)
	case []int:
		return slices.Clone(typed)
	case []any:
		return slices.Clone(typed)
	default:
		return value
	}
}
