package actionrelationutility

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type SearchRun struct {
	RunID       string
	WorldDigest string
	Store       *unit.Store
	Search      actionrelationsearch.Result
	Records     []dsl.ActionRelationMeterRecord
	Transcript  actionrelationexp.TranscriptBundle
	RunRoot     actionrelationexp.OperationRoot
}

func ExecuteComplete(domainsDir string, world actionrelations.World, panel, authority string, curriculum, worldOrdinal, cap int, token string) (SearchRun, error) {
	return ExecutePolicy(domainsDir, world, actionrelationsearch.Complete, panel, authority, curriculum, worldOrdinal, cap, token)
}

func ExecutePolicy(domainsDir string, world actionrelations.World, policy actionrelationsearch.Policy, panel, authority string, curriculum, worldOrdinal, cap int, token string) (SearchRun, error) {
	if !slices.Contains([]actionrelationsearch.Policy{actionrelationsearch.Complete, actionrelationsearch.Lexical, actionrelationsearch.StaticSleep, actionrelationsearch.DynamicSleep, actionrelationsearch.LearnedNoUse}, policy) {
		return SearchRun{}, fmt.Errorf("unsupported utility policy %q", policy)
	}
	normalized, err := world.Normalize()
	if err != nil {
		return SearchRun{}, err
	}
	worldDigest, _ := normalized.Digest()
	runID, err := actionrelationledger.UtilityRunID(panel, authority, curriculum, string(policy), worldOrdinal, worldDigest)
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
	session, err := BeginSession(store, runID, "utility-search:"+token, 0, cap)
	if err != nil {
		return SearchRun{}, err
	}
	runner := completeRunner{
		session: session, worldDigest: worldDigest, policy: string(policy), searchPolicy: policy,
		memo: map[string]completeVisit{}, evidence: map[string]bool{}, token: token, cache: NewCertificateCache(),
	}
	summary, err := runner.visit(normalized.State, normalized.Occurrences, nil)
	if err != nil {
		session.Abort()
		return SearchRun{}, err
	}
	runner.result.Policy = policy
	runner.result.CertificateEvidenceBound = policy == actionrelationsearch.StaticSleep || policy == actionrelationsearch.DynamicSleep
	runner.result.RootNodeDigest = summary.node.Digest
	runner.result.RootSubtree = summary.subtree
	runner.result.TerminalSet = summary.terminalSet
	runner.result.TerminalDigests = slices.Clone(summary.terminals)
	if err := actionrelationsearch.VerifyResultEvidence(runner.result); err != nil {
		session.Abort()
		return SearchRun{}, err
	}
	records, err := session.Close()
	if err != nil {
		return SearchRun{}, err
	}
	transcript, err := BuildTranscript(store, runID, records)
	if err != nil {
		return SearchRun{}, err
	}
	runRoot, err := actionrelationexp.BuildOperationRange(runID, 2, 0, transcript.CallIDs)
	if err != nil {
		return SearchRun{}, err
	}
	return SearchRun{RunID: runID, WorldDigest: worldDigest, Store: store, Search: runner.result, Records: records, Transcript: transcript, RunRoot: runRoot}, nil
}

type completeVisit struct {
	node         actionrelationsearch.EvidenceObject
	terminals    []string
	edgePreorder []string
	subtree      actionrelationsearch.EvidenceObject
	terminalSet  actionrelationsearch.EvidenceObject
}

type completeRunner struct {
	session      *Session
	worldDigest  string
	policy       string
	searchPolicy actionrelationsearch.Policy
	token        string
	result       actionrelationsearch.Result
	memo         map[string]completeVisit
	evidence     map[string]bool
	cache        *CertificateCache
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
		summary := completeVisit{node: node, terminals: []string{terminal.Digest}, subtree: subtree, terminalSet: terminalSet}
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
		if r.searchPolicy == actionrelationsearch.StaticSleep || r.searchPolicy == actionrelationsearch.DynamicSleep {
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
			if !decision.Certified {
				continue
			}
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
	summary := completeVisit{node: node, terminals: terminals, edgePreorder: edgePreorder, subtree: subtree, terminalSet: terminalSet}
	r.memo[node.Digest] = summary
	return summary, nil
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
		if err != nil || !result.Result {
			return false, nil, operationStart, err
		}
		row := r.session.Store.Get(result.Row)
		witness, _ := json.Marshal([]any{"static-witness/v1", row.GetString("objectDigest")})
		return true, witness, operationStart, nil
	default:
		return false, nil, -1, nil
	}
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
	if err != nil || reservation.Status != "reserved" {
		return fmt.Errorf("reserve utility %s: %w", kind, err)
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
