// Package actionrelationsearch implements the shared deterministic DFS and its
// policy filters. Local diamonds are checked by the independent oracle; the
// production vocabulary remains free of pair classification and search.
package actionrelationsearch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type Policy string

const (
	Complete     Policy = "complete"
	Lexical      Policy = "lexical-order"
	StaticSleep  Policy = "static-rw-sleep"
	DynamicSleep Policy = "dynamic-diamond-sleep"
	NousSleep    Policy = "nous-guarded-sleep"
	NoGuardSleep Policy = "no-guard-sleep"
	LearnedNoUse Policy = "learned-no-use"
)

type Result struct {
	Policy                   Policy
	TerminalDigests          []string
	NodeLookups              int
	ConstructedNodes         int
	ApplicabilityChecks      int
	EligibilityChecks        int
	CertificateChecks        int
	CertificateHits          int
	SleepPropagations        int
	HistoryCount             int
	Edges                    int
	RootNodeDigest           string
	RootSubtree              EvidenceObject
	TerminalSet              EvidenceObject
	Nodes                    []EvidenceObject
	RemainingSets            []EvidenceObject
	ProofMaps                []EvidenceObject
	SearchEdges              []EvidenceObject
	Propagations             []EvidenceObject
	CompletedSubtrees        []EvidenceObject
	TerminalBehaviors        []EvidenceObject
	SubtreeRoots             []EvidenceObject
	TerminalSets             []EvidenceObject
	CertificateEvidenceBound bool
}

func (r Result) Work() int {
	return r.NodeLookups + r.ApplicabilityChecks + r.EligibilityChecks + r.CertificateChecks + r.Edges
}

type Artifact struct {
	Relations []actionrelations.Relation
}

type CertificateFunc func(state actionrelations.State, left, right actionrelations.Occurrence) (bool, error)
type EligibilityFunc func(state actionrelations.State, left, right actionrelations.Occurrence) (bool, error)
type CertificateEvidenceFunc func(state actionrelations.State, left, right actionrelations.Occurrence) (CertificateDecision, error)

type CertificateDecision struct {
	Certified         bool
	CertificateDigest string
}

func Search(world actionrelations.World, policy Policy, artifact Artifact) (Result, error) {
	return SearchWithCertifier(world, policy, artifact, nil)
}

func SearchWithCertifier(world actionrelations.World, policy Policy, artifact Artifact, certifier CertificateFunc) (Result, error) {
	return SearchWithAdapters(world, policy, artifact, nil, certifier)
}

func SearchWithAdapters(world actionrelations.World, policy Policy, artifact Artifact, eligibility EligibilityFunc, certifier CertificateFunc) (Result, error) {
	return searchWithEvidenceAdapters(world, policy, artifact, eligibility, certifier, nil, false)
}

func SearchWithEvidenceAdapters(world actionrelations.World, policy Policy, artifact Artifact, eligibility EligibilityFunc, certifier CertificateEvidenceFunc) (Result, error) {
	if certifier == nil && policy != Complete && policy != Lexical && policy != LearnedNoUse {
		return Result{}, fmt.Errorf("certified policy requires evidence-producing certifier")
	}
	return searchWithEvidenceAdapters(world, policy, artifact, eligibility, nil, certifier, true)
}

func searchWithEvidenceAdapters(world actionrelations.World, policy Policy, artifact Artifact, eligibility EligibilityFunc, certifier CertificateFunc, evidenceCertifier CertificateEvidenceFunc, authoritative bool) (Result, error) {
	normalized, err := world.Normalize()
	if err != nil {
		return Result{}, err
	}
	if !onePolicy(policy) {
		return Result{}, fmt.Errorf("unknown policy %q", policy)
	}
	search := searcher{policy: policy, artifact: artifact, eligibility: eligibility, certifier: certifier, evidenceCertifier: evidenceCertifier, memo: map[string]visitSummary{}, certificateCache: map[string]CertificateDecision{}, evidence: map[string]bool{}}
	summary, err := search.visit(normalized.State, normalized.Occurrences, nil)
	if err != nil {
		return Result{}, err
	}
	terminals := summary.TerminalDigests
	slices.Sort(terminals)
	search.result.TerminalDigests = slices.Compact(terminals)
	search.result.Policy = policy
	search.result.RootNodeDigest = summary.Node.Digest
	search.result.RootSubtree = summary.Subtree
	search.result.TerminalSet = summary.TerminalSet
	search.result.HistoryCount = summary.HistoryCount
	search.result.CertificateEvidenceBound = authoritative
	return search.result, nil
}

type searcher struct {
	policy            Policy
	artifact          Artifact
	certifier         CertificateFunc
	evidenceCertifier CertificateEvidenceFunc
	eligibility       EligibilityFunc
	result            Result
	memo              map[string]visitSummary
	certificateCache  map[string]CertificateDecision
	evidence          map[string]bool
}

type visitSummary struct {
	Node            EvidenceObject
	TerminalDigests []string
	EdgePreorder    []string
	Subtree         EvidenceObject
	TerminalSet     EvidenceObject
	HistoryCount    int
}

func (s *searcher) visit(state actionrelations.State, remaining []actionrelations.Occurrence, proofs []ProofEntry) (visitSummary, error) {
	s.result.NodeLookups++
	remainingObject, err := BuildRemaining(remaining)
	if err != nil {
		return visitSummary{}, err
	}
	proofMap, err := BuildProofMap(proofs)
	if err != nil {
		return visitSummary{}, err
	}
	node, err := BuildSearchNode(state, remainingObject, proofMap)
	if err != nil {
		return visitSummary{}, err
	}
	if summary, ok := s.memo[node.Digest]; ok {
		return summary, nil
	}
	s.record(&s.result.RemainingSets, remainingObject)
	s.record(&s.result.ProofMaps, proofMap)
	s.record(&s.result.Nodes, node)
	s.result.ConstructedNodes++
	enabled := make([]actionrelations.Occurrence, 0, len(remaining))
	for _, occurrence := range remaining {
		s.result.ApplicabilityChecks++
		ok, err := actionrelations.Applicable(state, occurrence.Action)
		if err != nil {
			return visitSummary{}, err
		}
		if ok {
			enabled = append(enabled, occurrence)
		}
	}
	if len(enabled) == 0 {
		terminal, err := BuildTerminalBehavior(state, remaining)
		if err != nil {
			return visitSummary{}, err
		}
		s.record(&s.result.TerminalBehaviors, terminal)
		terminalSet, _ := BuildTerminalSet([]string{terminal.Digest})
		subtree, _ := BuildSubtreeRoot(node.Digest, nil)
		s.record(&s.result.TerminalSets, terminalSet)
		s.record(&s.result.SubtreeRoots, subtree)
		summary := visitSummary{Node: node, TerminalDigests: []string{terminal.Digest}, Subtree: subtree, TerminalSet: terminalSet, HistoryCount: 1}
		s.memo[node.Digest] = summary
		return summary, nil
	}
	priorProofs := map[string]string{}
	for _, proof := range proofs {
		priorProofs[proof.SleeperDigest] = proof.PropagationDigest
	}
	sleeperSet := makeSet(proofSleeperDigests(proofs))
	var earlier []actionrelations.Occurrence
	earlierSubtrees := map[string]string{}
	var terminals []string
	var edgePreorder []string
	historyCount := 0
	for _, taken := range enabled {
		takenDigest, _ := taken.Digest()
		if sleeperSet[takenDigest] {
			continue
		}
		next, outcome, err := actionrelations.Apply(state, taken.Action)
		if err != nil || outcome != "applied" {
			return visitSummary{}, fmt.Errorf("enabled transition failed: %s %w", outcome, err)
		}
		childRemaining := removeOccurrence(remaining, takenDigest)
		childRemainingObject, err := BuildRemaining(childRemaining)
		if err != nil {
			return visitSummary{}, err
		}
		candidateDigests := append(proofSleeperDigests(proofs), occurrenceDigests(earlier)...)
		slices.Sort(candidateDigests)
		candidateDigests = slices.Compact(candidateDigests)
		childRemainingSet := makeSet(occurrenceDigests(childRemaining))
		var childProofs []ProofEntry
		for _, candidateDigest := range candidateDigests {
			if !childRemainingSet[candidateDigest] {
				continue
			}
			candidate, ok := findOccurrence(remaining, candidateDigest)
			if !ok || !containsOccurrence(enabled, candidateDigest) {
				continue
			}
			eligible, err := s.eligible(state, taken, candidate)
			if err != nil {
				return visitSummary{}, err
			}
			if !eligible {
				continue
			}
			decision, err := s.certify(state, taken, candidate)
			if err != nil {
				return visitSummary{}, err
			}
			if decision.Certified {
				source, sourceAuthority := "prior-sleep", priorProofs[candidateDigest]
				if sourceAuthority == "" {
					source, sourceAuthority = "earlier-sibling", earlierSubtrees[candidateDigest]
				}
				successorDigest, _ := next.Digest()
				propagation, err := BuildPropagation(node.Digest, takenDigest, candidateDigest, source, sourceAuthority, decision.CertificateDigest, successorDigest, childRemainingObject.Digest)
				if err != nil {
					return visitSummary{}, err
				}
				s.record(&s.result.Propagations, propagation)
				childProofs = append(childProofs, ProofEntry{SleeperDigest: candidateDigest, PropagationDigest: propagation.Digest})
				s.result.SleepPropagations++
			}
		}
		childSummary, err := s.visit(next, childRemaining, childProofs)
		if err != nil {
			return visitSummary{}, err
		}
		edge, err := BuildSearchEdge(node.Digest, takenDigest, childProofs, childSummary.Node.Digest)
		if err != nil {
			return visitSummary{}, err
		}
		s.record(&s.result.SearchEdges, edge)
		terminals = append(terminals, childSummary.TerminalDigests...)
		historyCount += childSummary.HistoryCount
		s.result.Edges++
		completed, err := BuildCompletedSubtree(node.Digest, takenDigest, edge, childSummary.Subtree, childSummary.TerminalSet)
		if err != nil {
			return visitSummary{}, err
		}
		s.record(&s.result.CompletedSubtrees, completed)
		edgePreorder = append(edgePreorder, completed.Digest)
		earlierSubtrees[takenDigest] = completed.Digest
		earlier = append(earlier, taken)
	}
	slices.Sort(terminals)
	terminals = slices.Compact(terminals)
	terminalSet, err := BuildTerminalSet(terminals)
	if err != nil {
		return visitSummary{}, err
	}
	subtree, err := BuildSubtreeRoot(node.Digest, edgePreorder)
	if err != nil {
		return visitSummary{}, err
	}
	s.record(&s.result.TerminalSets, terminalSet)
	s.record(&s.result.SubtreeRoots, subtree)
	summary := visitSummary{Node: node, TerminalDigests: slices.Clone(terminals), EdgePreorder: edgePreorder, Subtree: subtree, TerminalSet: terminalSet, HistoryCount: historyCount}
	s.memo[node.Digest] = summary
	return summary, nil
}

func (s *searcher) eligible(state actionrelations.State, left, right actionrelations.Occurrence) (bool, error) {
	if s.policy == Complete || s.policy == Lexical || s.policy == LearnedNoUse {
		return false, nil
	}
	s.result.EligibilityChecks++
	left, right, err := actionrelations.CanonicalPair(left, right)
	if err != nil {
		return false, err
	}
	leftFacts, err := actionrelations.Facts(state, left)
	if err != nil {
		return false, err
	}
	rightFacts, err := actionrelations.Facts(state, right)
	if err != nil {
		return false, err
	}
	switch s.policy {
	case DynamicSleep:
		return true, nil
	case StaticSleep:
		return actionrelations.EvaluateAtom("read-write-disjoint", leftFacts, rightFacts)
	case NousSleep, NoGuardSleep:
		if s.policy == NousSleep && s.eligibility != nil {
			return s.eligibility(state, left, right)
		}
		if len(state.Events) > 6 || len(s.artifact.Relations) == 0 {
			return false, nil
		}
		pattern, err := actionrelations.PatternFor(left, right)
		if err != nil {
			return false, err
		}
		patternJSON, _ := pattern.CanonicalJSON()
		for _, relation := range s.artifact.Relations {
			relationPattern, _ := relation.Pattern.CanonicalJSON()
			if !bytes.Equal(patternJSON, relationPattern) {
				return false, nil
			}
			guard := relation.Guard
			if s.policy == NoGuardSleep {
				guard = actionrelations.Guard{}
			}
			matched, err := guard.Evaluate(leftFacts, rightFacts)
			if err != nil || !matched {
				return false, err
			}
		}
		return true, nil
	default:
		return false, nil
	}
}

func (s *searcher) certify(state actionrelations.State, left, right actionrelations.Occurrence) (CertificateDecision, error) {
	left, right, err := actionrelations.CanonicalPair(left, right)
	if err != nil {
		return CertificateDecision{}, err
	}
	stateDigest, _ := state.Digest()
	leftDigest, _ := left.Digest()
	rightDigest, _ := right.Digest()
	key := strings.Join([]string{stateDigest, leftDigest, rightDigest}, ":")
	s.result.CertificateChecks++
	if result, ok := s.certificateCache[key]; ok {
		s.result.CertificateHits++
		return result, nil
	}
	decision := CertificateDecision{}
	if s.evidenceCertifier != nil {
		decision, err = s.evidenceCertifier(state, left, right)
		if err != nil || decision.Certified && !digestText(decision.CertificateDigest) || !decision.Certified && decision.CertificateDigest != "" {
			return CertificateDecision{}, fmt.Errorf("invalid certificate evidence: %w", err)
		}
	} else if s.certifier != nil {
		decision.Certified, err = s.certifier(state, left, right)
		if err != nil {
			return CertificateDecision{}, err
		}
		if decision.Certified {
			decision.CertificateDigest = developmentCertificateDigest(stateDigest, leftDigest, rightDigest)
		}
	} else {
		return CertificateDecision{}, fmt.Errorf("sleep policy requires an external certificate authority")
	}
	s.certificateCache[key] = decision
	return decision, nil
}

func developmentCertificateDigest(stateDigest, leftDigest, rightDigest string) string {
	wire, _ := json.Marshal([]any{"actionrelation-development-certificate/v1", stateDigest, leftDigest, rightDigest})
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:])
}

func proofSleeperDigests(proofs []ProofEntry) []string {
	result := make([]string, len(proofs))
	for index, proof := range proofs {
		result[index] = proof.SleeperDigest
	}
	slices.Sort(result)
	return result
}

func (s *searcher) record(target *[]EvidenceObject, object EvidenceObject) {
	if !s.evidence[object.Digest] {
		s.evidence[object.Digest] = true
		*target = append(*target, object)
	}
}

func occurrenceDigests(occurrences []actionrelations.Occurrence) []string {
	digests := make([]string, len(occurrences))
	for index, occurrence := range occurrences {
		digests[index], _ = occurrence.Digest()
	}
	slices.Sort(digests)
	return digests
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

func findOccurrence(occurrences []actionrelations.Occurrence, digest string) (actionrelations.Occurrence, bool) {
	for _, occurrence := range occurrences {
		current, _ := occurrence.Digest()
		if current == digest {
			return occurrence, true
		}
	}
	return actionrelations.Occurrence{}, false
}

func containsOccurrence(occurrences []actionrelations.Occurrence, digest string) bool {
	_, ok := findOccurrence(occurrences, digest)
	return ok
}

func makeSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func onePolicy(policy Policy) bool {
	return slices.Contains([]Policy{Complete, Lexical, StaticSleep, DynamicSleep, NousSleep, NoGuardSleep, LearnedNoUse}, policy)
}
