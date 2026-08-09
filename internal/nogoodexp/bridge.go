package nogoodexp

import (
	"fmt"
	"io"
	"slices"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

type Disposition struct {
	Status, Request, Artifact, Binding, Completion, Certificate, Barrier, Proposal, Memo string
	TasksPopped                                                                          int
	Store                                                                                *unit.Store
	MeterRecords                                                                         []dsl.NogoodMeterRecord
}

func NewConcreteMemoBridge(domainsDir, exactKey string) (*BridgeExecution, error) {
	if !validDigest(exactKey) {
		return nil, fmt.Errorf("invalid concrete memo key")
	}
	execution, err := NewBridgeExecution(domainsDir, nil, nil)
	if err != nil {
		return nil, err
	}
	memo := unit.New("NG.ConcreteMemo." + exactKey[:16])
	memo.Set("isA", []string{"NogoodConcreteMemo", "Anything"})
	memo.Set("exactKey", exactKey)
	execution.base.Put(memo)
	return execution, nil
}

type BridgeExecution struct {
	base        *unit.Store
	profileHash string
	artifact    *FrozenArtifact
	authority   *ArtifactAuthority
	nextRequest int
	preflight   []TranscriptEvent
}

const committedBridgeProfileHash = "6b6275f20242bca8580b6bc9dcd459e19f37359424d1b6d8e53452b7e01a4dfc"

func NewBridgeExecution(domainsDir string, artifact *FrozenArtifact, authority *ArtifactAuthority) (*BridgeExecution, error) {
	store := unit.NewStore()
	previousDomainsDir := seed.DomainsDir
	seed.DomainsDir = domainsDir
	err := seed.LoadDomain(store, "nogoods")
	seed.DomainsDir = previousDomainsDir
	if err != nil {
		return nil, err
	}
	profileHash, preflightChecks, err := auditBridgeProfile(store)
	if err != nil {
		return nil, err
	}
	if profileHash != committedBridgeProfileHash {
		return nil, fmt.Errorf("bridge profile digest drifted: %s", profileHash)
	}
	if artifact != nil {
		if err := installArtifact(store, *artifact, authority); err != nil {
			return nil, err
		}
		copyArtifact := *artifact
		artifact = &copyArtifact
	}
	return &BridgeExecution{base: store, profileHash: profileHash, artifact: artifact, authority: authority, nextRequest: 1, preflight: profilePreflightTranscript(profileHash, preflightChecks)}, nil
}

func profilePreflightTranscript(profileHash string, checks []string) []TranscriptEvent {
	events := make([]TranscriptEvent, 0, len(checks))
	for index, operation := range checks {
		events = append(events, TranscriptEvent{Category: 12, Code: 16, TaskOrdinal: 0xffffffff, Operands: [8]TranscriptOperand{ID("NG-H-ConsiderPrune"), OptionalID(operation), Number(int32(index + 1)), ID("profile-preflight:" + profileHash[:16]), ID("ok"), Omitted(), Omitted(), Omitted()}})
	}
	return events
}

func auditBridgeProfile(store *unit.Store) (string, []string, error) {
	checks := []string{"examples:Heuristic"}
	if got := store.Examples("Heuristic"); !slices.Equal(got, []string{"Heuristic", "NG-H-ConsiderPrune"}) {
		return "", nil, fmt.Errorf("unexpected bridge heuristic profile %v", got)
	}
	category, lane := store.Get("Heuristic"), store.Get("NG-H-ConsiderPrune")
	checks = append(checks, "identity:Heuristic", "identity:NG-H-ConsiderPrune")
	if category == nil || lane == nil {
		return "", nil, fmt.Errorf("missing bridge profile unit")
	}
	profile := struct {
		StoreDigest string            `json:"store_digest"`
		Programs    map[string]string `json:"programs"`
		Shape       map[string]bool   `json:"shape"`
	}{Programs: map[string]string{}, Shape: map[string]bool{}}
	storeBytes, err := store.CanonicalJSON()
	if err != nil {
		return "", nil, err
	}
	profile.StoreDigest = digestBytes(storeBytes)
	for _, entry := range []struct {
		name string
		u    *unit.Unit
	}{{"Heuristic", category}, {"NG-H-ConsiderPrune", lane}} {
		for _, slot := range append(unit.IfPartSlots(), unit.ThenPartSlots()...) {
			program := entry.u.GetString(slot)
			checks = append(checks, "slot-read:"+entry.name+"."+slot)
			key := entry.name + "." + slot
			profile.Shape[key] = program != ""
			checks = append(checks, "slot-shape:"+key)
			if program != "" {
				// Source reads and hashes are intentionally separate audited
				// operations from the shape read above.
				source := entry.u.GetString(slot)
				checks = append(checks, "source-read:"+key)
				profile.Programs[key] = digestBytes([]byte(source))
				checks = append(checks, "source-hash:"+key)
			}
		}
	}
	if !validDigest(profile.StoreDigest) {
		return "", nil, fmt.Errorf("bridge store digest is invalid")
	}
	checks = append(checks, "compare:store-digest")
	if len(profile.Programs) != 2 || profile.Programs["NG-H-ConsiderPrune.ifWorkingOnTask"] == "" || profile.Programs["NG-H-ConsiderPrune.thenCompute"] == "" {
		return "", nil, fmt.Errorf("bridge program shape drifted")
	}
	checks = append(checks, "compare:profile-shape")
	for _, digest := range profile.Programs {
		if !validDigest(digest) {
			return "", nil, fmt.Errorf("bridge source digest is invalid")
		}
	}
	checks = append(checks, "compare:source-digests")
	if len(checks) != 54 {
		return "", nil, fmt.Errorf("bridge preflight performed %d checks, want 54", len(checks))
	}
	return digestJSON(profile), checks, nil
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
	if decision.Variable < 0 || decision.Variable >= len(problem.Variables) || !problem.DomainContains(decision.Variable, decision.Color) || slices.ContainsFunc(problem.Assignment, func(literal nogoods.Literal) bool { return literal.Variable == decision.Variable }) {
		return Disposition{}, fmt.Errorf("invalid supplied bridge decision")
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
	targetDigest := digestBytes(problemJSON)
	decisionDigest := digestJSON(decision)
	currentAssignment := append(slices.Clone(problem.Assignment), decision)
	slices.SortFunc(currentAssignment, func(a, b nogoods.Literal) int { return a.Variable - b.Variable })
	assignmentDigest := digestJSON(currentAssignment)
	immutableDigest := digestJSON(immutableDomains)
	reducedDigest := digestJSON(reducedDomains)
	conflictDigest := digestJSON([]string{})
	requestMaterial := struct {
		Profile, Target, Decision, Assignment, Immutable, Reduced, Conflicts, Artifacts string
		Number                                                                          int
	}{execution.profileHash, targetDigest, decisionDigest, assignmentDigest, immutableDigest, reducedDigest, conflictDigest, artifactStoreDigest, requestNumber}
	requestDigest := digestJSON(requestMaterial)
	requestName := fmt.Sprintf("NG.Request.%08d.%s", requestNumber, requestDigest[:16])
	meterToken := "ngm:bridge:" + requestDigest
	if err := dsl.RegisterNogoodMeter(meterToken); err != nil {
		return Disposition{}, err
	}
	defer dsl.UnregisterNogoodMeter(meterToken)
	if err := chargeMeterOperations(meterToken, 2, requestName, []string{"root-domain-read"}); err != nil {
		return Disposition{}, err
	}
	if err := chargeMeterOperations(meterToken, 3, requestName, []string{"root-propose", "root-bind"}); err != nil {
		return Disposition{}, err
	}
	if err := chargeMeterOperations(meterToken, 5, requestName, []string{"root-delete", "root-empty-check"}); err != nil {
		return Disposition{}, err
	}
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
	acceptedArtifactDigest, acceptedEvidenceDigest, acceptedPromotionDigest, acceptedProvenanceDigest := "", "", "", ""
	if execution.artifact != nil && execution.authority != nil && execution.authority.accepts(*execution.artifact) {
		acceptedArtifactDigest = execution.artifact.Digest
		acceptedEvidenceDigest = execution.artifact.EvidenceBoundaryDigest
		acceptedPromotionDigest = execution.artifact.PromotionDigest
		acceptedProvenanceDigest = execution.artifact.ProvenanceDigest
	}
	request.Set("acceptedArtifactDigest", acceptedArtifactDigest)
	request.Set("acceptedEvidenceDigest", acceptedEvidenceDigest)
	request.Set("acceptedPromotionDigest", acceptedPromotionDigest)
	request.Set("acceptedProvenanceDigest", acceptedProvenanceDigest)
	request.Set("requestDigest", requestDigest)
	concreteKey, err := concreteMemoKey(problemJSON, decision)
	if err != nil {
		return Disposition{}, err
	}
	request.Set("concreteMemoKey", concreteKey)
	request.Set("meterToken", meterToken)
	for index, variable := range problem.Variables {
		request.Set(fmt.Sprintf("domain%d", index), slices.Clone(variable.Domain))
	}
	store.Put(request)
	if err := dsl.ChargeNogoodMeter(meterToken, "request-write", requestName, requestDigest, "ok", 12); err != nil {
		return Disposition{}, err
	}

	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return Disposition{}, err
	}
	ag.Push(&agenda.Task{Priority: 800, UnitName: requestName, SlotName: "ngConsiderPrune", Reasons: []string{"Consider frozen blocked-pair artifact"}})
	if err := dsl.ChargeNogoodMeter(meterToken, "agenda-enqueue", requestName, "ngConsiderPrune", "ok", 12); err != nil {
		return Disposition{}, err
	}
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
		if err := dsl.ChargeNogoodMeter(meterToken, "agenda-dequeue", requestName, "ngConsiderPrune", "ok", 12); err != nil {
			return Disposition{}, err
		}
		if err := dsl.ChargeNogoodMeter(meterToken, "request-digest-check", requestName, requestDigest, "ok", 12); err != nil {
			return Disposition{}, err
		}
		eng.WorkOnTask(task)
		if eng.LastError != nil {
			return Disposition{}, fmt.Errorf("nogood bridge heuristic execution: %w", eng.LastError)
		}
		if len(eng.VM.DeletedUnits) != 0 {
			return Disposition{}, fmt.Errorf("bridge deleted units")
		}
		if err := chargeMeterOperations(meterToken, 12, requestName+".ngConsiderPrune", engineDispatchOperations); err != nil {
			return Disposition{}, err
		}
		popped++
	}
	disposition := store.Get(request.GetString("dispositionUnit"))
	if disposition == nil {
		return Disposition{}, fmt.Errorf("missing sealed disposition")
	}
	status := disposition.GetString("status")
	result := Disposition{Status: status, Request: requestName, Artifact: disposition.GetString("artifact"), Binding: disposition.GetString("binding"), Completion: disposition.GetString("completion"), Certificate: disposition.GetString("certificate"), Barrier: disposition.GetString("barrier"), Proposal: disposition.GetString("proposal"), Memo: disposition.GetString("memo"), TasksPopped: popped, Store: store}
	if status != "resume" && status != "propose-prune" && status != "concrete-prune" && status != "bridge-invalid" {
		return Disposition{}, fmt.Errorf("invalid disposition status %q", status)
	}
	if status == "bridge-invalid" {
		return Disposition{}, fmt.Errorf("bridge sealed an invalid disposition")
	}
	if status == "propose-prune" {
		barrier := store.Get(result.Barrier)
		referenceDigest, referenceErr := dsl.UnitSetDigest(store, []string{result.Request, result.Artifact, result.Binding, result.Completion, result.Certificate, result.Barrier, result.Proposal})
		barrierDigest := ""
		if barrier != nil && referenceErr == nil {
			barrierDigest = digestJSON([]any{barrier.GetStrings("predicateKeys"), barrier.GetStrings("predicateOutcomeKeys"), referenceDigest, requestDigest, targetDigest, decisionDigest, assignmentDigest, immutableDigest, reducedDigest, conflictDigest})
		}
		checks := [6]bool{
			disposition.GetBool("sealed") && disposition.GetString("request") == requestName,
			barrier != nil && barrier.GetBool("sealed") && barrier.GetString("barrierDigest") == barrierDigest && len(barrier.GetStrings("predicateKeys")) == 18 && !slices.Contains(barrier.GetStrings("predicateOutcomeKeys"), "false"),
			execution.artifact != nil && execution.authority != nil && execution.authority.accepts(*execution.artifact) && result.Artifact == execution.artifact.Name,
			store.Get(result.Completion) != nil && store.Get(result.Completion).GetBool("conflict"),
			disposition.GetString("requestDigest") == requestDigest && disposition.GetString("targetDigest") == targetDigest && disposition.GetString("decisionDigest") == decisionDigest,
			referenceErr == nil && disposition.GetString("referencedUnitSetDigest") == referenceDigest && disposition.GetString("barrierDigest") == barrierDigest,
		}
		for index, passed := range checks {
			if !passed {
				if index == 1 && barrier != nil {
					return Disposition{}, fmt.Errorf("adapter proposal check 2 failed: barrier digest got=%q want=%q refs=%q barrierRefs=%q keys=%v outcomes=%v", barrier.GetString("barrierDigest"), barrierDigest, referenceDigest, barrier.GetString("referencedUnitSetDigest"), barrier.GetStrings("predicateKeys"), barrier.GetStrings("predicateOutcomeKeys"))
				}
				return Disposition{}, fmt.Errorf("adapter proposal check %d failed", index+1)
			}
			if err := dsl.ChargeNogoodMeter(meterToken, fmt.Sprintf("adapter-proposal-check-%d", index+1), result.Request, result.Proposal, "ok", 10); err != nil {
				return Disposition{}, err
			}
		}
	} else if status == "concrete-prune" {
		memo := store.Get(result.Memo)
		checks := [6]bool{
			disposition.GetBool("sealed") && disposition.GetString("request") == requestName,
			memo != nil && store.IsA(result.Memo, "NogoodConcreteMemo"),
			memo != nil && memo.GetString("exactKey") == concreteKey,
			len(store.Examples("NogoodConcreteMemo")) == 2,
			disposition.GetInt("applicableCount") == 0,
			result.Artifact == "" && result.Binding == "" && result.Completion == "" && result.Certificate == "" && result.Barrier == "" && result.Proposal == "",
		}
		for index, passed := range checks {
			if !passed {
				return Disposition{}, fmt.Errorf("adapter concrete memo check %d failed", index+1)
			}
			if err := dsl.ChargeNogoodMeter(meterToken, fmt.Sprintf("adapter-concrete-check-%d", index+1), result.Request, result.Memo, "ok", 10); err != nil {
				return Disposition{}, err
			}
		}
	} else {
		resumeChecks := [6]bool{
			disposition.GetBool("sealed") && disposition.GetString("request") == requestName,
			disposition.GetString("requestDigest") == requestDigest,
			disposition.GetString("targetDigest") == targetDigest,
			disposition.GetString("decisionDigest") == decisionDigest,
			disposition.GetInt("applicableCount") == 0,
			result.Artifact == "" && result.Binding == "" && result.Completion == "" && result.Certificate == "" && result.Barrier == "" && result.Proposal == "",
		}
		for index, passed := range resumeChecks {
			if !passed {
				return Disposition{}, fmt.Errorf("adapter resume check %d failed", index+1)
			}
			if err := dsl.ChargeNogoodMeter(meterToken, fmt.Sprintf("adapter-resume-check-%d", index+1), result.Request, result.Status, "ok", 10); err != nil {
				return Disposition{}, err
			}
		}
	}
	records, err := dsl.NogoodMeterSnapshot(meterToken)
	if err != nil {
		return Disposition{}, err
	}
	result.MeterRecords = records
	return result, nil
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
