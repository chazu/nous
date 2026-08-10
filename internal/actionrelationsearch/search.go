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

	"github.com/chazu/nous/internal/actionrelationoracle"
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
	Policy              Policy
	TerminalDigests     []string
	NodeLookups         int
	ConstructedNodes    int
	ApplicabilityChecks int
	EligibilityChecks   int
	CertificateChecks   int
	CertificateHits     int
	SleepPropagations   int
	Edges               int
}

func (r Result) Work() int {
	return r.NodeLookups + r.ApplicabilityChecks + r.EligibilityChecks + r.CertificateChecks + r.Edges
}

type Artifact struct {
	Relations []actionrelations.Relation
}

func Search(world actionrelations.World, policy Policy, artifact Artifact) (Result, error) {
	normalized, err := world.Normalize()
	if err != nil {
		return Result{}, err
	}
	if !onePolicy(policy) {
		return Result{}, fmt.Errorf("unknown policy %q", policy)
	}
	search := searcher{policy: policy, artifact: artifact, memo: map[string][]string{}, certificateCache: map[string]bool{}}
	terminals, err := search.visit(normalized.State, normalized.Occurrences, nil)
	if err != nil {
		return Result{}, err
	}
	slices.Sort(terminals)
	search.result.TerminalDigests = slices.Compact(terminals)
	search.result.Policy = policy
	return search.result, nil
}

type searcher struct {
	policy           Policy
	artifact         Artifact
	result           Result
	memo             map[string][]string
	certificateCache map[string]bool
}

func (s *searcher) visit(state actionrelations.State, remaining []actionrelations.Occurrence, sleepers []string) ([]string, error) {
	s.result.NodeLookups++
	key, err := nodeKey(state, remaining, sleepers)
	if err != nil {
		return nil, err
	}
	if terminals, ok := s.memo[key]; ok {
		return slices.Clone(terminals), nil
	}
	s.result.ConstructedNodes++
	enabled := make([]actionrelations.Occurrence, 0, len(remaining))
	for _, occurrence := range remaining {
		s.result.ApplicabilityChecks++
		ok, err := actionrelations.Applicable(state, occurrence.Action)
		if err != nil {
			return nil, err
		}
		if ok {
			enabled = append(enabled, occurrence)
		}
	}
	if len(enabled) == 0 {
		terminal, err := terminalDigest(state, remaining)
		if err != nil {
			return nil, err
		}
		s.memo[key] = []string{terminal}
		return []string{terminal}, nil
	}
	sleeperSet := makeSet(sleepers)
	var earlier []actionrelations.Occurrence
	var terminals []string
	for _, taken := range enabled {
		takenDigest, _ := taken.Digest()
		if sleeperSet[takenDigest] {
			continue
		}
		next, outcome, err := actionrelations.Apply(state, taken.Action)
		if err != nil || outcome != "applied" {
			return nil, fmt.Errorf("enabled transition failed: %s %w", outcome, err)
		}
		childRemaining := removeOccurrence(remaining, takenDigest)
		candidateDigests := append(slices.Clone(sleepers), occurrenceDigests(earlier)...)
		slices.Sort(candidateDigests)
		candidateDigests = slices.Compact(candidateDigests)
		childRemainingSet := makeSet(occurrenceDigests(childRemaining))
		var childSleepers []string
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
				return nil, err
			}
			if !eligible {
				continue
			}
			certified, err := s.certify(state, taken, candidate)
			if err != nil {
				return nil, err
			}
			if certified {
				childSleepers = append(childSleepers, candidateDigest)
				s.result.SleepPropagations++
			}
		}
		slices.Sort(childSleepers)
		childTerminals, err := s.visit(next, childRemaining, childSleepers)
		if err != nil {
			return nil, err
		}
		terminals = append(terminals, childTerminals...)
		s.result.Edges++
		earlier = append(earlier, taken)
	}
	slices.Sort(terminals)
	terminals = slices.Compact(terminals)
	s.memo[key] = slices.Clone(terminals)
	return terminals, nil
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

func (s *searcher) certify(state actionrelations.State, left, right actionrelations.Occurrence) (bool, error) {
	left, right, err := actionrelations.CanonicalPair(left, right)
	if err != nil {
		return false, err
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
	stateJSON, _ := state.CanonicalJSON()
	leftJSON, _ := left.Action.CanonicalJSON()
	rightJSON, _ := right.Action.CanonicalJSON()
	observation, err := actionrelationoracle.Observe(stateJSON, leftJSON, rightJSON)
	if err != nil {
		return false, err
	}
	result := observation.Label == "commutes"
	s.certificateCache[key] = result
	return result, nil
}

func nodeKey(state actionrelations.State, remaining []actionrelations.Occurrence, sleepers []string) (string, error) {
	stateDigest, err := state.Digest()
	if err != nil {
		return "", err
	}
	wire, _ := json.Marshal([]any{"sleep-search-node-key/v1", stateDigest, occurrenceDigests(remaining), sleepers})
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:]), nil
}

func terminalDigest(state actionrelations.State, remaining []actionrelations.Occurrence) (string, error) {
	stateDigest, err := state.Digest()
	if err != nil {
		return "", err
	}
	wire, _ := json.Marshal([]any{"sleep-terminal-behavior/v1", stateDigest, occurrenceDigests(remaining)})
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:]), nil
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
