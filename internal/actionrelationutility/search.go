package actionrelationutility

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/actionrelationmatch"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type SearchRun struct {
	RunID             string
	WorldDigest       string
	Policy            actionrelationsearch.Policy
	Store             *unit.Store
	Search            actionrelationsearch.Result
	Records           []dsl.ActionRelationMeterRecord
	Transcript        actionrelationexp.TranscriptBundle
	RunRoot           actionrelationexp.OperationRoot
	Terminal          string
	WorkTerminal      WorkTerminal
	WorkVector        [12]int
	InitialWork       [12]int
	WorkTotal         int
	ProofRoots        []actionrelationexp.OperationRoot
	PhysicalWork      int
	PriorPhysical     int
	StructuralObjects []actionrelationexp.ObjectRecord
}

type WorkBudget struct {
	LifecycleCap  int
	PhysicalCap   int
	PriorPhysical int
}

func ExecuteComplete(domainsDir string, world actionrelations.World, panel, authority string, curriculum, worldOrdinal, cap int, token string) (SearchRun, error) {
	return ExecutePolicy(domainsDir, world, actionrelationsearch.Complete, panel, authority, curriculum, worldOrdinal, cap, token)
}

func ExecutePolicy(domainsDir string, world actionrelations.World, policy actionrelationsearch.Policy, panel, authority string, curriculum, worldOrdinal, cap int, token string) (SearchRun, error) {
	return ExecutePolicyWithBudget(domainsDir, world, policy, panel, authority, curriculum, worldOrdinal, WorkBudget{LifecycleCap: cap, PhysicalCap: physicalPolicyCap(policy)}, token)
}

func ExecutePolicyWithBudget(domainsDir string, world actionrelations.World, policy actionrelationsearch.Policy, panel, authority string, curriculum, worldOrdinal int, budget WorkBudget, token string) (SearchRun, error) {
	return ExecutePolicyContinuing(domainsDir, world, policy, panel, authority, curriculum, worldOrdinal, [12]int{}, budget, token)
}

// ExecutePolicyContinuing executes an artifact-free policy while carrying the
// exact lifecycle work from earlier worlds in the same curriculum-policy run.
// The returned Records and PhysicalWork cover only this world; WorkVector and
// WorkTotal include initialWork.
func ExecutePolicyContinuing(domainsDir string, world actionrelations.World, policy actionrelationsearch.Policy, panel, authority string, curriculum, worldOrdinal int, initialWork [12]int, budget WorkBudget, token string) (SearchRun, error) {
	if !slices.Contains([]actionrelationsearch.Policy{actionrelationsearch.Complete, actionrelationsearch.Lexical, actionrelationsearch.StaticSleep, actionrelationsearch.DynamicSleep, actionrelationsearch.LearnedNoUse}, policy) {
		return SearchRun{}, fmt.Errorf("unsupported utility policy %q", policy)
	}
	normalized, err := world.Normalize()
	if err != nil {
		return SearchRun{}, err
	}
	store := unit.NewStore()
	previous := seed.DomainsDir
	seed.DomainsDir = domainsDir
	err = seed.LoadDomain(store, "actionrelations")
	seed.DomainsDir = previous
	if err != nil {
		return SearchRun{}, err
	}
	return executePolicyOnStore(store, normalized, policy, panel, authority, curriculum, worldOrdinal, initialWork, budget, token, "", "")
}

func ExecuteLearnedPolicy(store *unit.Store, artifactName, boundaryName string, world actionrelations.World, policy actionrelationsearch.Policy, panel, authority string, curriculum, worldOrdinal int, initialWork [12]int, cap int, token string) (SearchRun, error) {
	return ExecuteLearnedPolicyWithBudget(store, artifactName, boundaryName, world, policy, panel, authority, curriculum, worldOrdinal, initialWork, WorkBudget{LifecycleCap: cap, PhysicalCap: physicalPolicyCap(policy)}, token)
}

func ExecuteLearnedPolicyWithBudget(store *unit.Store, artifactName, boundaryName string, world actionrelations.World, policy actionrelationsearch.Policy, panel, authority string, curriculum, worldOrdinal int, initialWork [12]int, budget WorkBudget, token string) (SearchRun, error) {
	if policy != actionrelationsearch.NousSleep && policy != actionrelationsearch.NoGuardSleep && policy != actionrelationsearch.LearnedNoUse {
		return SearchRun{}, fmt.Errorf("unsupported learned utility policy %q", policy)
	}
	artifact, boundary := store.Get(artifactName), store.Get(boundaryName)
	if artifact == nil || boundary == nil || !store.IsA(artifact.Name, "GuardedActionArtifact") || !store.IsA(boundary.Name, "ActionStoreBoundary") {
		return SearchRun{}, fmt.Errorf("invalid learned utility authority")
	}
	normalized, err := world.Normalize()
	if err != nil {
		return SearchRun{}, err
	}
	return executePolicyOnStore(store, normalized, policy, panel, authority, curriculum, worldOrdinal, initialWork, budget, token, artifactName, boundaryName)
}

func executePolicyOnStore(store *unit.Store, normalized actionrelations.NormalizedWorld, policy actionrelationsearch.Policy, panel, authority string, curriculum, worldOrdinal int, initialWork [12]int, budget WorkBudget, token, artifactName, boundaryName string) (SearchRun, error) {
	initialTotal := sumWorkVector(initialWork)
	if budget.LifecycleCap < 1 || budget.PhysicalCap < 1 || budget.PriorPhysical < 0 || budget.PriorPhysical >= budget.PhysicalCap {
		return SearchRun{}, fmt.Errorf("invalid utility work budget")
	}
	effectiveCap := initialTotal + budget.PhysicalCap - budget.PriorPhysical
	if budget.LifecycleCap < effectiveCap {
		effectiveCap = budget.LifecycleCap
	}
	if initialTotal >= effectiveCap {
		return SearchRun{}, fmt.Errorf("utility budget exhausted before run start")
	}
	worldDigest, _ := normalized.Digest()
	runID, err := actionrelationledger.UtilityRunID(panel, authority, curriculum, string(policy), worldOrdinal, worldDigest)
	if err != nil {
		return SearchRun{}, err
	}
	session, err := BeginSession(store, runID, "utility-search:"+token, initialTotal, effectiveCap)
	if err != nil {
		return SearchRun{}, err
	}
	if err := session.SetInitialWorkVector(initialWork); err != nil {
		session.Abort()
		return SearchRun{}, err
	}
	runner := completeRunner{
		session: session, worldDigest: worldDigest, policy: string(policy), searchPolicy: policy,
		memo: map[string]completeVisit{}, evidence: map[string]bool{}, token: token, cache: NewCertificateCache(), artifactName: artifactName, proofRootSeen: map[string]bool{}, structuralSeen: map[string]bool{},
	}
	if artifactName != "" {
		if err := runner.chargeArtifactLoad(boundaryName, artifactName); err != nil {
			session.Abort()
			return SearchRun{}, err
		}
	}
	summary, err := runner.visit(normalized.State, normalized.Occurrences, nil)
	if err != nil {
		var exhausted *budgetExhaustedError
		if errors.As(err, &exhausted) {
			terminal, terminalErr := session.TerminateBudget(exhausted.Reservation)
			if terminalErr != nil {
				session.Abort()
				return SearchRun{}, terminalErr
			}
			return finishSearchRun(session, runID, worldDigest, policy, budget.PriorPhysical, runner.result, runner.proofRoots, runner.structural, "budget-exhausted", terminal)
		}
		session.Abort()
		return SearchRun{}, err
	}
	runner.result.Policy = policy
	runner.result.CertificateEvidenceBound = policy == actionrelationsearch.StaticSleep || policy == actionrelationsearch.DynamicSleep || policy == actionrelationsearch.NousSleep || policy == actionrelationsearch.NoGuardSleep
	runner.result.RootNodeDigest = summary.node.Digest
	runner.result.RootSubtree = summary.subtree
	runner.result.TerminalSet = summary.terminalSet
	runner.result.TerminalDigests = slices.Clone(summary.terminals)
	runner.result.HistoryCount = summary.historyCount
	if err := actionrelationsearch.VerifyResultEvidence(runner.result); err != nil {
		session.Abort()
		return SearchRun{}, err
	}
	return finishSearchRun(session, runID, worldDigest, policy, budget.PriorPhysical, runner.result, runner.proofRoots, runner.structural, "completed", WorkTerminal{})
}

func finishSearchRun(session *Session, runID, worldDigest string, policy actionrelationsearch.Policy, priorPhysical int, result actionrelationsearch.Result, proofRoots []actionrelationexp.OperationRoot, extraStructural []actionrelationexp.ObjectRecord, terminal string, workTerminal WorkTerminal) (SearchRun, error) {
	initialWork := session.InitialWork
	records, err := session.Close()
	if err != nil {
		return SearchRun{}, err
	}
	transcript, err := BuildTranscript(session.Store, runID, records)
	if err != nil {
		return SearchRun{}, err
	}
	runRoot, err := actionrelationexp.BuildOperationRange(runID, 2, 0, transcript.CallIDs)
	if err != nil {
		return SearchRun{}, err
	}
	workVector, err := MeterWorkVector(records)
	if err != nil {
		return SearchRun{}, err
	}
	for index := range workVector {
		workVector[index] += initialWork[index]
	}
	structural, err := collectStructuralObjects(result, proofRoots, runRoot, extraStructural)
	if err != nil {
		return SearchRun{}, err
	}
	run := SearchRun{RunID: runID, WorldDigest: worldDigest, Policy: policy, Store: session.Store, Search: result, Records: records, Transcript: transcript, RunRoot: runRoot, Terminal: terminal, WorkTerminal: workTerminal, WorkVector: workVector, InitialWork: initialWork, WorkTotal: sumWorkVector(workVector), ProofRoots: slices.Clone(proofRoots), PhysicalWork: len(records), PriorPhysical: priorPhysical, StructuralObjects: structural}
	if err := VerifySearchRun(run); err != nil {
		return SearchRun{}, err
	}
	return run, nil
}

func physicalPolicyCap(policy actionrelationsearch.Policy) int {
	switch policy {
	case actionrelationsearch.DynamicSleep, actionrelationsearch.NousSleep:
		return 8192
	case actionrelationsearch.Complete, actionrelationsearch.Lexical, actionrelationsearch.StaticSleep, actionrelationsearch.NoGuardSleep, actionrelationsearch.LearnedNoUse:
		return 4096
	default:
		return 0
	}
}

type completeVisit struct {
	node         actionrelationsearch.EvidenceObject
	terminals    []string
	edgePreorder []string
	subtree      actionrelationsearch.EvidenceObject
	terminalSet  actionrelationsearch.EvidenceObject
	historyCount int
}

type completeRunner struct {
	session        *Session
	worldDigest    string
	policy         string
	searchPolicy   actionrelationsearch.Policy
	token          string
	result         actionrelationsearch.Result
	memo           map[string]completeVisit
	evidence       map[string]bool
	cache          *CertificateCache
	artifactName   string
	proofRoots     []actionrelationexp.OperationRoot
	proofRootSeen  map[string]bool
	structural     []actionrelationexp.ObjectRecord
	structuralSeen map[string]bool
}

type budgetExhaustedError struct {
	Reservation actionrelationledger.Reservation
}

func (e *budgetExhaustedError) Error() string {
	return "utility budget exhausted"
}

func (r *completeRunner) visit(state actionrelations.State, remaining []actionrelations.Occurrence, proofs []actionrelationsearch.ProofEntry) (completeVisit, error) {
	remainingObject, err := actionrelationsearch.BuildRemaining(remaining)
	if err != nil {
		return completeVisit{}, err
	}
	proofMap, err := actionrelationsearch.BuildProofMap(proofs)
	if err != nil {
		return completeVisit{}, err
	}
	node, err := actionrelationsearch.BuildSearchNode(state, remainingObject, proofMap)
	if err != nil {
		return completeVisit{}, err
	}
	r.result.NodeLookups++
	hit := r.memo[node.Digest].node.Digest != ""
	if err := r.chargeNodeLookup(state, remainingObject, proofMap, node, hit); err != nil {
		return completeVisit{}, err
	}
	if hit {
		return r.memo[node.Digest], nil
	}
	r.record(&r.result.RemainingSets, remainingObject)
	r.record(&r.result.ProofMaps, proofMap)
	r.record(&r.result.Nodes, node)
	r.result.ConstructedNodes++
	stateDigest, _ := state.Digest()
	nodeName := "AR.SearchNode." + node.Digest
	nodeUnit := unit.New(nodeName)
	nodeUnit.Set("isA", []string{"ActionRelationSearchNode", "Anything"})
	nodeUnit.Set("canonicalObject", string(node.Canonical))
	nodeUnit.Set("objectDigest", node.Digest)
	nodeUnit.Set("worldDigest", r.worldDigest)
	nodeUnit.Set("policy", r.policy)
	nodeUnit.Set("stateDigest", stateDigest)
	nodeUnit.Set("remainingOccurrenceDigests", occurrenceDigests(remaining))
	r.session.Store.Put(nodeUnit)

	enabled := make([]actionrelations.Occurrence, 0, len(remaining))
	applicabilityRows := make([]string, len(remaining))
	applicabilityValues := make([]bool, len(remaining))
	for index, occurrence := range remaining {
		occurrenceDigest, _ := occurrence.Digest()
		if err := r.reserveTask("enabledness", []any{node.Digest, occurrenceDigest}, []uint8{23}); err != nil {
			return completeVisit{}, err
		}
		result, err := SearchApplicable(r.session.Store, r.session.MeterToken, nodeName, r.worldDigest, r.policy, state, occurrence, fmt.Sprintf("%s.%05d", r.token, r.session.Sequence))
		if err != nil {
			return completeVisit{}, err
		}
		r.result.ApplicabilityChecks++
		applicabilityRows[index] = result.Row
		applicabilityValues[index] = result.Result
		if result.Result {
			enabled = append(enabled, occurrence)
		}
	}
	if len(enabled) == 0 {
		terminal, err := actionrelationsearch.BuildTerminalBehaviorFromApplicability(state, remaining, applicabilityValues)
		if err != nil {
			return completeVisit{}, err
		}
		if err := r.chargeTerminal(stateDigest, remainingObject.Digest, applicabilityRows, terminal); err != nil {
			return completeVisit{}, err
		}
		r.record(&r.result.TerminalBehaviors, terminal)
		terminalSet, _ := actionrelationsearch.BuildTerminalSet([]string{terminal.Digest})
		subtree, _ := actionrelationsearch.BuildSubtreeRoot(node.Digest, nil)
		r.record(&r.result.TerminalSets, terminalSet)
		r.record(&r.result.SubtreeRoots, subtree)
		summary := completeVisit{node: node, terminals: []string{terminal.Digest}, subtree: subtree, terminalSet: terminalSet, historyCount: 1}
		r.memo[node.Digest] = summary
		return summary, nil
	}
	priorProofs := map[string]string{}
	for _, proof := range proofs {
		priorProofs[proof.SleeperDigest] = proof.PropagationDigest
	}
	sleeperSet := stringSet(proofSleeperDigests(proofs))
	var earlier []actionrelations.Occurrence
	earlierSubtrees := map[string]string{}
	var terminals, edgePreorder []string
	historyCount := 0
	for _, taken := range enabled {
		takenDigest, _ := taken.Digest()
		if sleeperSet[takenDigest] {
			continue
		}
		rowName := applicabilityRows[indexOccurrence(remaining, takenDigest)]
		if err := r.reserveTask("transition", []any{node.Digest, takenDigest, r.session.Store.Get(rowName).GetString("objectDigest")}, []uint8{11}); err != nil {
			return completeVisit{}, err
		}
		transition, err := SearchApply(r.session.Store, r.session.MeterToken, rowName, state, taken, fmt.Sprintf("%s.%05d", r.token, r.session.Sequence))
		if err != nil || transition.Outcome != "applied" {
			return completeVisit{}, fmt.Errorf("enabled utility transition failed: %v", err)
		}
		childRemaining := removeOccurrence(remaining, takenDigest)
		childRemainingObject, err := actionrelationsearch.BuildRemaining(childRemaining)
		if err != nil {
			return completeVisit{}, err
		}
		var candidateDigests []string
		if r.searchPolicy == actionrelationsearch.StaticSleep || r.searchPolicy == actionrelationsearch.DynamicSleep || r.searchPolicy == actionrelationsearch.NousSleep || r.searchPolicy == actionrelationsearch.NoGuardSleep {
			candidateDigests = append(proofSleeperDigests(proofs), occurrenceDigests(earlier)...)
			slices.Sort(candidateDigests)
			candidateDigests = slices.Compact(candidateDigests)
		}
		childRemainingSet := stringSet(occurrenceDigests(childRemaining))
		var childProofs []actionrelationsearch.ProofEntry
		for _, candidateDigest := range candidateDigests {
			if !childRemainingSet[candidateDigest] || !containsOccurrence(enabled, candidateDigest) {
				continue
			}
			candidate, ok := findOccurrence(remaining, candidateDigest)
			if !ok {
				return completeVisit{}, fmt.Errorf("missing sleep candidate")
			}
			priorDigest := priorProofs[candidateDigest]
			if err := r.chargeProofLookup(node.Digest, proofMap.Digest, candidateDigest, priorDigest); err != nil {
				return completeVisit{}, err
			}
			eligible, witness, operationStart, err := r.eligibility(nodeName, node.Digest, state, taken, candidate, applicabilityRows[indexOccurrence(remaining, candidateDigest)])
			if err != nil {
				return completeVisit{}, err
			}
			if !eligible {
				continue
			}
			decision, err := CertifyCached(r.session, r.cache, r.worldDigest, r.policy, state, taken, candidate, witness, operationStart, fmt.Sprintf("%s.%05d", r.token, r.session.Sequence))
			if err != nil {
				return completeVisit{}, err
			}
			if !decision.CacheHit {
				cacheRow := r.session.Store.Get(decision.CacheRow)
				if cacheRow == nil {
					return completeVisit{}, fmt.Errorf("fresh certificate decision lacks cache row")
				}
				attempt := r.session.Store.Get(cacheRow.GetString("attemptUnit"))
				if attempt == nil {
					return completeVisit{}, fmt.Errorf("fresh certificate decision lacks attempt preimage")
				}
				if err := r.recordStructuralUnit(44, attempt); err != nil {
					return completeVisit{}, err
				}
				if decision.Certified {
					certificate := r.unitByDigest(decision.CertificateDigest)
					if certificate == nil {
						return completeVisit{}, fmt.Errorf("certified decision lacks certificate preimage")
					}
					if err := r.recordStructuralUnit(17, certificate); err != nil {
						return completeVisit{}, err
					}
				}
			}
			if !decision.Certified {
				r.recordProofRoot(decision.OperationRoot)
				continue
			}
			r.recordProofRoot(decision.OperationRoot)
			source, sourceAuthority := "prior-sleep", priorDigest
			if sourceAuthority == "" {
				source, sourceAuthority = "earlier-sibling", earlierSubtrees[candidateDigest]
			}
			successorDigest, _ := transition.ResultState.Digest()
			propagation, err := actionrelationsearch.BuildPropagation(node.Digest, takenDigest, candidateDigest, source, sourceAuthority, decision.CertificateDigest, successorDigest, childRemainingObject.Digest)
			if err != nil {
				return completeVisit{}, err
			}
			r.record(&r.result.Propagations, propagation)
			childProofs = append(childProofs, actionrelationsearch.ProofEntry{SleeperDigest: candidateDigest, PropagationDigest: propagation.Digest})
			r.result.SleepPropagations++
		}
		child, err := r.visit(transition.ResultState, childRemaining, childProofs)
		if err != nil {
			return completeVisit{}, err
		}
		edge, err := actionrelationsearch.BuildSearchEdge(node.Digest, takenDigest, childProofs, child.node.Digest)
		if err != nil {
			return completeVisit{}, err
		}
		r.record(&r.result.SearchEdges, edge)
		edgePreorder = append(edgePreorder, edge.Digest)
		edgePreorder = append(edgePreorder, child.edgePreorder...)
		terminals = append(terminals, child.terminals...)
		historyCount += child.historyCount
		r.result.Edges++
		completed, _ := actionrelationsearch.BuildCompletedSubtree(node.Digest, takenDigest, child.subtree, child.terminalSet)
		r.record(&r.result.CompletedSubtrees, completed)
		earlierSubtrees[takenDigest] = completed.Digest
		earlier = append(earlier, taken)
	}
	slices.Sort(terminals)
	terminals = slices.Compact(terminals)
	terminalSet, _ := actionrelationsearch.BuildTerminalSet(terminals)
	subtree, _ := actionrelationsearch.BuildSubtreeRoot(node.Digest, edgePreorder)
	r.record(&r.result.TerminalSets, terminalSet)
	r.record(&r.result.SubtreeRoots, subtree)
	summary := completeVisit{node: node, terminals: terminals, edgePreorder: edgePreorder, subtree: subtree, terminalSet: terminalSet, historyCount: historyCount}
	r.memo[node.Digest] = summary
	return summary, nil
}

func (r *completeRunner) recordProofRoot(root actionrelationexp.OperationRoot) {
	if root.Digest == "" || r.proofRootSeen[root.Digest] {
		return
	}
	r.proofRootSeen[root.Digest] = true
	r.proofRoots = append(r.proofRoots, root)
}

func (r *completeRunner) recordStructuralBytes(kind uint16, canonical []byte) error {
	if actionrelationexp.ValidateObject(kind, canonical) != nil {
		return fmt.Errorf("invalid structural object kind %d", kind)
	}
	hash := sha256.Sum256(canonical)
	digest := hex.EncodeToString(hash[:])
	key := fmt.Sprintf("%d:%s", kind, digest)
	if !r.structuralSeen[key] {
		r.structuralSeen[key] = true
		r.structural = append(r.structural, actionrelationexp.ObjectRecord{Kind: kind, Bytes: slices.Clone(canonical)})
	}
	return nil
}

func (r *completeRunner) recordStructuralUnit(kind uint16, value *unit.Unit) error {
	if value == nil {
		return fmt.Errorf("missing structural object kind %d", kind)
	}
	canonical := []byte(value.GetString("canonicalObject"))
	hash := sha256.Sum256(canonical)
	if value.GetString("objectDigest") != hex.EncodeToString(hash[:]) {
		return fmt.Errorf("structural object kind %d changed digest", kind)
	}
	return r.recordStructuralBytes(kind, canonical)
}

func (r *completeRunner) recordMatchStructural(result actionrelationmatch.Result) error {
	request := r.session.Store.Get(result.Request)
	if request == nil {
		return fmt.Errorf("learned match lacks request")
	}
	for _, name := range []string{request.GetString("aFacts"), request.GetString("bFacts")} {
		if err := r.recordStructuralUnit(8, r.session.Store.Get(name)); err != nil {
			return err
		}
	}
	return r.recordStructuralUnit(43, r.session.Store.Get(result.Barrier))
}

func (r *completeRunner) unitByDigest(digest string) *unit.Unit {
	for _, name := range r.session.Store.All() {
		value := r.session.Store.Get(name)
		if value != nil && value.GetString("objectDigest") == digest {
			return value
		}
	}
	return nil
}

func collectStructuralObjects(result actionrelationsearch.Result, proofRoots []actionrelationexp.OperationRoot, runRoot actionrelationexp.OperationRoot, extra []actionrelationexp.ObjectRecord) ([]actionrelationexp.ObjectRecord, error) {
	objects := slices.Clone(extra)
	appendEvidence := func(kind uint16, values []actionrelationsearch.EvidenceObject) {
		for _, value := range values {
			objects = append(objects, actionrelationexp.ObjectRecord{Kind: kind, Bytes: slices.Clone(value.Canonical)})
		}
	}
	appendEvidence(5, result.RemainingSets)
	appendEvidence(19, result.ProofMaps)
	appendEvidence(18, result.Propagations)
	appendEvidence(21, result.SearchEdges)
	appendEvidence(22, result.CompletedSubtrees)
	appendEvidence(24, result.TerminalSets)
	appendEvidence(25, result.SubtreeRoots)
	for _, root := range append(slices.Clone(proofRoots), runRoot) {
		objects = append(objects, actionrelationexp.ObjectRecord{Kind: 46, Bytes: slices.Clone(root.Canonical)})
	}
	type keyed struct {
		record actionrelationexp.ObjectRecord
		digest string
	}
	unique := map[string]keyed{}
	for _, object := range objects {
		if actionrelationexp.ValidateObject(object.Kind, object.Bytes) != nil {
			return nil, fmt.Errorf("invalid structural output kind %d", object.Kind)
		}
		hash := sha256.Sum256(object.Bytes)
		digest := hex.EncodeToString(hash[:])
		key := fmt.Sprintf("%05d:%s", object.Kind, digest)
		unique[key] = keyed{object, digest}
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	resultObjects := make([]actionrelationexp.ObjectRecord, len(keys))
	for index, key := range keys {
		resultObjects[index] = unique[key].record
	}
	return resultObjects, nil
}

func (r *completeRunner) eligibility(nodeName, nodeDigest string, state actionrelations.State, taken, candidate actionrelations.Occurrence, candidateApplicabilityRow string) (bool, []byte, int, error) {
	switch r.searchPolicy {
	case actionrelationsearch.DynamicSleep:
		r.result.EligibilityChecks++
		row := r.session.Store.Get(candidateApplicabilityRow)
		if row == nil {
			return false, nil, -1, fmt.Errorf("missing dynamic candidate row")
		}
		witness, _ := json.Marshal([]any{"dynamic-witness/v1", "all-pairs", row.GetString("objectDigest")})
		if err := r.recordStructuralBytes(16, witness); err != nil {
			return false, nil, -1, err
		}
		return true, witness, -1, nil
	case actionrelationsearch.StaticSleep:
		r.result.EligibilityChecks++
		operationStart := r.session.Sequence
		takenDigest, _ := taken.Digest()
		candidateDigest, _ := candidate.Digest()
		if err := r.reserveTask("static-footprint", []any{nodeDigest, takenDigest, candidateDigest}, []uint8{24}); err != nil {
			return false, nil, operationStart, err
		}
		result, err := StaticFootprint(r.session.Store, r.session.MeterToken, nodeName, r.worldDigest, state, taken, candidate, fmt.Sprintf("%s.%05d", r.token, r.session.Sequence))
		if err != nil {
			return false, nil, operationStart, err
		}
		row := r.session.Store.Get(result.Row)
		if row == nil {
			return false, nil, operationStart, fmt.Errorf("static footprint lacks row")
		}
		if err := r.recordStructuralUnit(8, r.session.Store.Get(row.GetString("aFacts"))); err != nil {
			return false, nil, operationStart, err
		}
		if err := r.recordStructuralUnit(8, r.session.Store.Get(row.GetString("bFacts"))); err != nil {
			return false, nil, operationStart, err
		}
		if !result.Result {
			return false, nil, operationStart, nil
		}
		witness, _ := json.Marshal([]any{"static-witness/v1", row.GetString("objectDigest")})
		if err := r.recordStructuralBytes(15, witness); err != nil {
			return false, nil, operationStart, err
		}
		return true, witness, operationStart, nil
	case actionrelationsearch.NousSleep, actionrelationsearch.NoGuardSleep:
		r.result.EligibilityChecks++
		operationStart := r.session.Sequence
		artifact := r.session.Store.Get(r.artifactName)
		if artifact == nil {
			return false, nil, operationStart, fmt.Errorf("missing learned artifact")
		}
		codes := []uint8{21, 21}
		for _, relationName := range artifact.GetStrings("relationUnits") {
			relation := r.session.Store.Get(relationName)
			if relation == nil {
				return false, nil, operationStart, fmt.Errorf("missing learned relation")
			}
			for range relation.GetStrings("atoms") {
				codes = append(codes, 15)
			}
			codes = append(codes, 9)
		}
		takenDigest, _ := taken.Digest()
		candidateDigest, _ := candidate.Digest()
		if err := r.reserveTask("learned-match", []any{nodeDigest, takenDigest, candidateDigest, artifact.GetString("objectDigest")}, codes); err != nil {
			return false, nil, operationStart, err
		}
		result, err := actionrelationmatch.ExecuteMetered(r.session.Store, r.artifactName, state, taken, candidate, fmt.Sprintf("%s.%05d", r.token, r.session.Sequence), r.session.MeterToken)
		if err != nil || result.Terminal != "completed" || !result.Matched {
			if err == nil && result.Request != "" {
				if structuralErr := r.recordMatchStructural(result); structuralErr != nil {
					return false, nil, operationStart, structuralErr
				}
			}
			return false, nil, operationStart, err
		}
		if err := r.recordMatchStructural(result); err != nil {
			return false, nil, operationStart, err
		}
		barrier := r.session.Store.Get(result.Barrier)
		if barrier == nil {
			return false, nil, operationStart, fmt.Errorf("learned match lacks unanimous barrier")
		}
		witness, _ := json.Marshal([]any{"learned-witness/v1", barrier.GetString("objectDigest")})
		if err := r.recordStructuralBytes(14, witness); err != nil {
			return false, nil, operationStart, err
		}
		return true, witness, operationStart, nil
	default:
		return false, nil, -1, nil
	}
}

func (r *completeRunner) chargeArtifactLoad(boundaryName, artifactName string) error {
	boundary, artifact := r.session.Store.Get(boundaryName), r.session.Store.Get(artifactName)
	if boundary == nil || artifact == nil {
		return fmt.Errorf("missing artifact load authority")
	}
	boundaryDigest, artifactDigest := boundary.GetString("objectDigest"), artifact.GetString("objectDigest")
	if err := r.reserveTask("artifact-load", []any{boundaryDigest, artifactDigest}, []uint8{10}); err != nil {
		return err
	}
	return dsl.ChargeActionRelationMeter(r.session.MeterToken, 10, 7, "artifact-load", [][]byte{[]byte(boundaryDigest), []byte(artifactDigest)}, [][]byte{[]byte(artifact.GetString("canonicalObject"))})
}

func (r *completeRunner) chargeProofLookup(parentNodeDigest, proofMapDigest, sleeperDigest, propagationDigest string) error {
	if err := r.reserveTask("proof-map-lookup", []any{parentNodeDigest, proofMapDigest, sleeperDigest}, []uint8{17}); err != nil {
		return err
	}
	var outputs [][]byte
	if propagationDigest != "" {
		for _, propagation := range r.result.Propagations {
			if propagation.Digest == propagationDigest {
				outputs = [][]byte{propagation.Canonical}
				break
			}
		}
		if len(outputs) == 0 {
			return fmt.Errorf("proof-map lookup lacks retained propagation")
		}
	}
	return dsl.ChargeActionRelationMeter(r.session.MeterToken, 17, 11, "proof-map-lookup", [][]byte{[]byte(parentNodeDigest), []byte(proofMapDigest), []byte(sleeperDigest)}, outputs)
}

func (r *completeRunner) chargeNodeLookup(state actionrelations.State, remaining, proofMap, node actionrelationsearch.EvidenceObject, hit bool) error {
	stateDigest, _ := state.Digest()
	if err := r.reserveTask("node-lookup", []any{stateDigest, remaining.Digest, proofMap.Digest}, []uint8{16}); err != nil {
		return err
	}
	status := uint8(1)
	if hit {
		status = 3
	}
	return dsl.ChargeActionRelationMeterStatus(r.session.MeterToken, 16, 11, status, "search-node-lookup", [][]byte{[]byte(stateDigest), []byte(remaining.Digest), []byte(proofMap.Digest)}, [][]byte{node.Canonical})
}

func (r *completeRunner) chargeTerminal(stateDigest, remainingDigest string, applicabilityRows []string, terminal actionrelationsearch.EvidenceObject) error {
	digests := make([]string, len(applicabilityRows))
	for index, name := range applicabilityRows {
		digests[index] = r.session.Store.Get(name).GetString("objectDigest")
	}
	if err := r.reserveTask("terminal", []any{stateDigest, remainingDigest, digests}, []uint8{19}); err != nil {
		return err
	}
	vector, _ := json.Marshal(digests)
	return dsl.ChargeActionRelationMeter(r.session.MeterToken, 19, 12, "terminal-construct", [][]byte{[]byte(stateDigest), []byte(remainingDigest), vector}, [][]byte{terminal.Canonical})
}

func (r *completeRunner) reserveTask(kind string, fields []any, codes []uint8) error {
	wire, _ := json.Marshal([]any{"actionrelation-utility-task/v1", r.session.RunID, kind, fields, codes})
	digest := sha256.Sum256(wire)
	reservation, err := r.session.Reserve(hex.EncodeToString(digest[:]), codes)
	if err != nil {
		return fmt.Errorf("reserve utility %s: %w", kind, err)
	}
	if reservation.Status == "rejected-cap" {
		return &budgetExhaustedError{Reservation: reservation}
	}
	if reservation.Status != "reserved" {
		return fmt.Errorf("reserve utility %s returned %s", kind, reservation.Status)
	}
	return nil
}

func (r *completeRunner) record(target *[]actionrelationsearch.EvidenceObject, object actionrelationsearch.EvidenceObject) {
	if !r.evidence[object.Digest] {
		r.evidence[object.Digest] = true
		*target = append(*target, object)
	}
}

func occurrenceDigests(occurrences []actionrelations.Occurrence) []string {
	result := make([]string, len(occurrences))
	for index, occurrence := range occurrences {
		result[index], _ = occurrence.Digest()
	}
	slices.Sort(result)
	return result
}

func indexOccurrence(occurrences []actionrelations.Occurrence, digest string) int {
	for index, occurrence := range occurrences {
		current, _ := occurrence.Digest()
		if current == digest {
			return index
		}
	}
	return -1
}

func removeOccurrence(occurrences []actionrelations.Occurrence, digest string) []actionrelations.Occurrence {
	result := make([]actionrelations.Occurrence, 0, len(occurrences)-1)
	removed := false
	for _, occurrence := range occurrences {
		current, _ := occurrence.Digest()
		if !removed && current == digest {
			removed = true
			continue
		}
		result = append(result, occurrence)
	}
	return result
}

func proofSleeperDigests(proofs []actionrelationsearch.ProofEntry) []string {
	result := make([]string, len(proofs))
	for index, proof := range proofs {
		result[index] = proof.SleeperDigest
	}
	slices.Sort(result)
	return result
}

func stringSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func findOccurrence(occurrences []actionrelations.Occurrence, digest string) (actionrelations.Occurrence, bool) {
	index := indexOccurrence(occurrences, digest)
	if index < 0 {
		return actionrelations.Occurrence{}, false
	}
	return occurrences[index], true
}

func containsOccurrence(occurrences []actionrelations.Occurrence, digest string) bool {
	return indexOccurrence(occurrences, digest) >= 0
}
