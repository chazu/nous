package actionrelationsearch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type EvidenceObject struct {
	Canonical []byte
	Digest    string
}

type ProofEntry struct {
	SleeperDigest     string
	PropagationDigest string
}

func BuildRemaining(occurrences []actionrelations.Occurrence) (EvidenceObject, error) {
	digests := occurrenceDigests(occurrences)
	if len(digests) != len(occurrences) || !sortedUniqueDigests(digests) {
		return EvidenceObject{}, fmt.Errorf("invalid remaining occurrence set")
	}
	return evidenceWire([]any{"remaining-occurrences/v1", digests}), nil
}

func BuildProofMap(entries []ProofEntry) (EvidenceObject, error) {
	entries = slices.Clone(entries)
	slices.SortFunc(entries, func(a, b ProofEntry) int {
		return bytes.Compare(mustDigest(a.SleeperDigest), mustDigest(b.SleeperDigest))
	})
	rows := make([]any, len(entries))
	for index, entry := range entries {
		if !digestText(entry.SleeperDigest) || !digestText(entry.PropagationDigest) || index > 0 && entry.SleeperDigest == entries[index-1].SleeperDigest {
			return EvidenceObject{}, fmt.Errorf("invalid proof-map entry %d", index)
		}
		rows[index] = []any{entry.SleeperDigest, entry.PropagationDigest}
	}
	return evidenceWire([]any{"sleep-proof-map/v1", rows}), nil
}

func BuildSearchNode(state actionrelations.State, remaining, proofMap EvidenceObject) (EvidenceObject, error) {
	stateDigest, err := state.Digest()
	if err != nil || !taggedEvidence(remaining, "remaining-occurrences/v1") || !taggedEvidence(proofMap, "sleep-proof-map/v1") {
		return EvidenceObject{}, fmt.Errorf("invalid search-node authority")
	}
	return evidenceWire([]any{"sleep-search-node/v1", stateDigest, remaining.Digest, proofMap.Digest}), nil
}

func BuildPropagation(parentNodeDigest, takenOccurrenceDigest, sleepingOccurrenceDigest, source, sourceAuthorityDigest, certificateDigest, successorStateDigest, childRemainingDigest string) (EvidenceObject, error) {
	if !digestText(parentNodeDigest) || !digestText(takenOccurrenceDigest) || !digestText(sleepingOccurrenceDigest) || takenOccurrenceDigest == sleepingOccurrenceDigest ||
		(source != "earlier-sibling" && source != "prior-sleep") || !digestText(sourceAuthorityDigest) || !digestText(certificateDigest) || !digestText(successorStateDigest) || !digestText(childRemainingDigest) {
		return EvidenceObject{}, fmt.Errorf("invalid sleep propagation")
	}
	return evidenceWire([]any{"sleep-propagation-core/v1", parentNodeDigest, takenOccurrenceDigest, sleepingOccurrenceDigest, source, sourceAuthorityDigest, certificateDigest, successorStateDigest, childRemainingDigest}), nil
}

func BuildSearchEdge(parentNodeDigest, takenOccurrenceDigest string, propagations []ProofEntry, childNodeDigest string) (EvidenceObject, error) {
	if !digestText(parentNodeDigest) || !digestText(takenOccurrenceDigest) || !digestText(childNodeDigest) {
		return EvidenceObject{}, fmt.Errorf("invalid search edge identity")
	}
	propagations = slices.Clone(propagations)
	slices.SortFunc(propagations, func(a, b ProofEntry) int {
		return bytes.Compare(mustDigest(a.SleeperDigest), mustDigest(b.SleeperDigest))
	})
	digests := make([]string, len(propagations))
	for index, propagation := range propagations {
		if !digestText(propagation.SleeperDigest) || !digestText(propagation.PropagationDigest) || index > 0 && propagation.SleeperDigest == propagations[index-1].SleeperDigest {
			return EvidenceObject{}, fmt.Errorf("invalid edge propagation %d", index)
		}
		digests[index] = propagation.PropagationDigest
	}
	return evidenceWire([]any{"sleep-search-edge/v1", parentNodeDigest, takenOccurrenceDigest, digests, childNodeDigest}), nil
}

func BuildTerminalBehavior(state actionrelations.State, remaining []actionrelations.Occurrence) (EvidenceObject, error) {
	stateJSON, err := state.CanonicalJSON()
	if err != nil {
		return EvidenceObject{}, err
	}
	terminal := "complete"
	if len(remaining) > 0 {
		terminal = "deadlock"
		for _, occurrence := range remaining {
			applicable, err := actionrelations.Applicable(state, occurrence.Action)
			if err != nil || applicable {
				return EvidenceObject{}, fmt.Errorf("remaining occurrence is not deadlocked")
			}
		}
	}
	return evidenceWire([]any{"action-terminal/v1", json.RawMessage(stateJSON), occurrenceDigests(remaining), terminal}), nil
}

func BuildTerminalSet(terminalDigests []string) (EvidenceObject, error) {
	terminalDigests = slices.Clone(terminalDigests)
	slices.Sort(terminalDigests)
	terminalDigests = slices.Compact(terminalDigests)
	if !allDigests(terminalDigests) {
		return EvidenceObject{}, fmt.Errorf("invalid terminal set")
	}
	return evidenceWire([]any{"sleep-terminal-set/v1", terminalDigests}), nil
}

func BuildSubtreeRoot(rootNodeDigest string, preorderEdgeDigests []string) (EvidenceObject, error) {
	if !digestText(rootNodeDigest) || !allDigests(preorderEdgeDigests) {
		return EvidenceObject{}, fmt.Errorf("invalid subtree root")
	}
	return evidenceWire([]any{"sleep-subtree-root/v1", rootNodeDigest, slices.Clone(preorderEdgeDigests)}), nil
}

func BuildCompletedSubtree(parentNodeDigest, takenOccurrenceDigest string, subtreeRoot, terminalSet EvidenceObject) (EvidenceObject, error) {
	if !digestText(parentNodeDigest) || !digestText(takenOccurrenceDigest) || !taggedEvidence(subtreeRoot, "sleep-subtree-root/v1") || !taggedEvidence(terminalSet, "sleep-terminal-set/v1") {
		return EvidenceObject{}, fmt.Errorf("invalid completed subtree")
	}
	return evidenceWire([]any{"completed-subtree/v1", parentNodeDigest, takenOccurrenceDigest, subtreeRoot.Digest, terminalSet.Digest, "completed"}), nil
}

func evidenceWire(row []any) EvidenceObject {
	canonical, _ := json.Marshal(row)
	digest := sha256.Sum256(canonical)
	return EvidenceObject{Canonical: canonical, Digest: hex.EncodeToString(digest[:])}
}

func taggedEvidence(value EvidenceObject, tag string) bool {
	if value.Digest != shaHex(value.Canonical) {
		return false
	}
	var row []json.RawMessage
	var got string
	return json.Unmarshal(value.Canonical, &row) == nil && len(row) > 0 && json.Unmarshal(row[0], &got) == nil && got == tag
}

func sortedUniqueDigests(values []string) bool {
	for index, value := range values {
		if !digestText(value) || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}

func allDigests(values []string) bool {
	for _, value := range values {
		if !digestText(value) {
			return false
		}
	}
	return true
}

func digestText(value string) bool {
	if len(value) != 64 {
		return false
	}
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == 32 && value == hex.EncodeToString(raw)
}

func mustDigest(value string) []byte {
	raw, _ := hex.DecodeString(value)
	return raw
}

func shaHex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
