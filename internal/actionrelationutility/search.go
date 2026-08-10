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
	normalized, err := world.Normalize()
	if err != nil {
		return SearchRun{}, err
	}
	worldDigest, _ := normalized.Digest()
	runID, err := actionrelationledger.UtilityRunID(panel, authority, curriculum, string(actionrelationsearch.Complete), worldOrdinal, worldDigest)
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
		session: session, worldDigest: worldDigest, policy: string(actionrelationsearch.Complete),
		memo: map[string]completeVisit{}, evidence: map[string]bool{}, token: token,
	}
	summary, err := runner.visit(normalized.State, normalized.Occurrences)
	if err != nil {
		session.Abort()
		return SearchRun{}, err
	}
	runner.result.Policy = actionrelationsearch.Complete
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
	session     *Session
	worldDigest string
	policy      string
	token       string
	result      actionrelationsearch.Result
	memo        map[string]completeVisit
	evidence    map[string]bool
}

func (r *completeRunner) visit(state actionrelations.State, remaining []actionrelations.Occurrence) (completeVisit, error) {
	remainingObject, err := actionrelationsearch.BuildRemaining(remaining)
	if err != nil {
		return completeVisit{}, err
	}
	proofMap, _ := actionrelationsearch.BuildProofMap(nil)
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
	var terminals, edgePreorder []string
	for _, taken := range enabled {
		takenDigest, _ := taken.Digest()
		rowName := applicabilityRows[indexOccurrence(remaining, takenDigest)]
		if err := r.reserveTask("transition", []any{node.Digest, takenDigest, r.session.Store.Get(rowName).GetString("objectDigest")}, []uint8{11}); err != nil {
			return completeVisit{}, err
		}
		transition, err := SearchApply(r.session.Store, r.session.MeterToken, rowName, state, taken, fmt.Sprintf("%s.%05d", r.token, r.session.Sequence))
		if err != nil || transition.Outcome != "applied" {
			return completeVisit{}, fmt.Errorf("enabled utility transition failed: %v", err)
		}
		childRemaining := removeOccurrence(remaining, takenDigest)
		child, err := r.visit(transition.ResultState, childRemaining)
		if err != nil {
			return completeVisit{}, err
		}
		edge, err := actionrelationsearch.BuildSearchEdge(node.Digest, takenDigest, nil, child.node.Digest)
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
