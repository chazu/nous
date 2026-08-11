package actionrelationsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
)

type parsedNode struct {
	state, remaining, proofMap string
}

type parsedEdge struct {
	digest, parent, taken, child string
	propagations                 []string
}

type parsedPropagation struct {
	digest, parent, taken, sleeper, source, authority, certificate, successor, childRemaining string
}

type parsedCompleted struct {
	digest, parent, taken, edge, subtree, terminalSet string
}

type parsedSubtree struct {
	digest, node string
	preorder     []string
}

type parsedTerminal struct {
	digest, state, terminal string
	remaining               []string
}

type verifiedSubtree struct {
	terminals []string
	histories int
}

func VerifyResultEvidence(result Result) error {
	collections := []struct {
		tag     string
		objects []EvidenceObject
	}{
		{"remaining-occurrences/v1", result.RemainingSets}, {"sleep-proof-map/v1", result.ProofMaps},
		{"sleep-search-node/v1", result.Nodes}, {"sleep-search-edge/v1", result.SearchEdges},
		{"sleep-propagation-core/v1", result.Propagations}, {"completed-subtree/v2", result.CompletedSubtrees},
		{"action-terminal/v1", result.TerminalBehaviors}, {"sleep-subtree-root/v1", result.SubtreeRoots},
		{"sleep-terminal-set/v1", result.TerminalSets},
	}
	objects := map[string]EvidenceObject{}
	tags := map[string]string{}
	for _, collection := range collections {
		for _, object := range collection.objects {
			if !taggedEvidence(object, collection.tag) || objects[object.Digest].Digest != "" {
				return fmt.Errorf("invalid or duplicate %s evidence", collection.tag)
			}
			objects[object.Digest], tags[object.Digest] = object, collection.tag
		}
	}
	if tags[result.RootNodeDigest] != "sleep-search-node/v1" || tags[result.RootSubtree.Digest] != "sleep-subtree-root/v1" || tags[result.TerminalSet.Digest] != "sleep-terminal-set/v1" {
		return fmt.Errorf("missing root search evidence")
	}

	remainingRows := map[string][]string{}
	for _, value := range result.RemainingSets {
		row, err := evidenceRow(value, 2)
		var occurrences []string
		if err != nil || json.Unmarshal(row[1], &occurrences) != nil || !sortedUniqueOrEmpty(occurrences) {
			return fmt.Errorf("invalid remaining-occurrence rows")
		}
		remainingRows[value.Digest] = occurrences
	}

	proofRows := map[string][]ProofEntry{}
	for _, proofMap := range result.ProofMaps {
		row, err := evidenceRow(proofMap, 2)
		if err != nil {
			return err
		}
		var rawRows [][]json.RawMessage
		if json.Unmarshal(row[1], &rawRows) != nil {
			return fmt.Errorf("invalid proof-map rows")
		}
		entries := make([]ProofEntry, len(rawRows))
		for index, raw := range rawRows {
			if len(raw) != 2 || json.Unmarshal(raw[0], &entries[index].SleeperDigest) != nil || json.Unmarshal(raw[1], &entries[index].PropagationDigest) != nil || tags[entries[index].PropagationDigest] != "sleep-propagation-core/v1" || index > 0 && entries[index].SleeperDigest <= entries[index-1].SleeperDigest {
				return fmt.Errorf("invalid retained proof-map entry")
			}
		}
		proofRows[proofMap.Digest] = entries
	}

	nodes := map[string]parsedNode{}
	for _, node := range result.Nodes {
		row, err := evidenceRow(node, 4)
		parsed := parsedNode{}
		if err != nil || json.Unmarshal(row[1], &parsed.state) != nil || json.Unmarshal(row[2], &parsed.remaining) != nil || json.Unmarshal(row[3], &parsed.proofMap) != nil || !digestText(parsed.state) || tags[parsed.remaining] != "remaining-occurrences/v1" || tags[parsed.proofMap] != "sleep-proof-map/v1" {
			return fmt.Errorf("invalid retained search node")
		}
		remainingSet := makeSet(remainingRows[parsed.remaining])
		for _, proof := range proofRows[parsed.proofMap] {
			if !remainingSet[proof.SleeperDigest] {
				return fmt.Errorf("proof-map sleeper is absent from node remaining set")
			}
		}
		nodes[node.Digest] = parsed
	}

	edges := map[string]parsedEdge{}
	edgeByParentTaken := map[string]parsedEdge{}
	for _, edge := range result.SearchEdges {
		row, err := evidenceRow(edge, 5)
		parsed := parsedEdge{digest: edge.Digest}
		if err != nil || json.Unmarshal(row[1], &parsed.parent) != nil || json.Unmarshal(row[2], &parsed.taken) != nil || json.Unmarshal(row[3], &parsed.propagations) != nil || json.Unmarshal(row[4], &parsed.child) != nil || nodes[parsed.parent].state == "" || nodes[parsed.child].state == "" || !digestText(parsed.taken) || !allDigests(parsed.propagations) {
			return fmt.Errorf("invalid retained search edge")
		}
		key := parsed.parent + parsed.taken
		if edgeByParentTaken[key].digest != "" {
			return fmt.Errorf("duplicate parent/taken search edge")
		}
		parentRemaining := remainingRows[nodes[parsed.parent].remaining]
		childRemaining := remainingRows[nodes[parsed.child].remaining]
		if !slices.Contains(parentRemaining, parsed.taken) || !slices.Equal(removeDigest(parentRemaining, parsed.taken), childRemaining) || proofEntry(proofRows[nodes[parsed.parent].proofMap], parsed.taken).SleeperDigest != "" {
			return fmt.Errorf("search edge does not remove one non-sleeping occurrence")
		}
		childProofs := proofRows[nodes[parsed.child].proofMap]
		want := make([]string, len(childProofs))
		for index, entry := range childProofs {
			want[index] = entry.PropagationDigest
		}
		if !slices.Equal(parsed.propagations, want) {
			return fmt.Errorf("edge/proof-map propagation mismatch")
		}
		edges[edge.Digest], edgeByParentTaken[key] = parsed, parsed
	}

	terminalSets := map[string][]string{}
	for _, terminalSet := range result.TerminalSets {
		row, err := evidenceRow(terminalSet, 2)
		var terminals []string
		if err != nil || json.Unmarshal(row[1], &terminals) != nil || !sortedUniqueOrEmpty(terminals) {
			return fmt.Errorf("invalid terminal set rows")
		}
		for _, digest := range terminals {
			if tags[digest] != "action-terminal/v1" {
				return fmt.Errorf("terminal set references unknown behavior")
			}
		}
		terminalSets[terminalSet.Digest] = terminals
	}

	terminals := map[string]parsedTerminal{}
	terminalByNodeKey := map[string]string{}
	for _, terminal := range result.TerminalBehaviors {
		row, err := evidenceRow(terminal, 4)
		var stateJSON json.RawMessage
		parsed := parsedTerminal{digest: terminal.Digest}
		if err != nil || json.Unmarshal(row[1], &stateJSON) != nil || json.Unmarshal(row[2], &parsed.remaining) != nil || json.Unmarshal(row[3], &parsed.terminal) != nil || !sortedUniqueOrEmpty(parsed.remaining) || parsed.terminal != "complete" && parsed.terminal != "deadlock" || (len(parsed.remaining) == 0) != (parsed.terminal == "complete") {
			return fmt.Errorf("invalid terminal behavior")
		}
		parsed.state = shaHex(stateJSON)
		remainingWire := evidenceWire([]any{"remaining-occurrences/v1", parsed.remaining})
		key := parsed.state + remainingWire.Digest
		if terminalByNodeKey[key] != "" {
			return fmt.Errorf("duplicate terminal behavior authority")
		}
		terminals[terminal.Digest], terminalByNodeKey[key] = parsed, terminal.Digest
	}

	subtrees := map[string]parsedSubtree{}
	subtreeByNode := map[string]string{}
	for _, subtree := range result.SubtreeRoots {
		row, err := evidenceRow(subtree, 3)
		parsed := parsedSubtree{digest: subtree.Digest}
		if err != nil || json.Unmarshal(row[1], &parsed.node) != nil || json.Unmarshal(row[2], &parsed.preorder) != nil || nodes[parsed.node].state == "" || !allDigests(parsed.preorder) {
			return fmt.Errorf("invalid subtree root")
		}
		if subtreeByNode[parsed.node] != "" {
			return fmt.Errorf("multiple retained subtrees for one search node")
		}
		subtrees[subtree.Digest], subtreeByNode[parsed.node] = parsed, subtree.Digest
	}

	completed := map[string]parsedCompleted{}
	completedByParentTaken := map[string]parsedCompleted{}
	completedByEdge := map[string]parsedCompleted{}
	terminalSetForSubtree := map[string]string{result.RootSubtree.Digest: result.TerminalSet.Digest}
	for _, value := range result.CompletedSubtrees {
		row, err := evidenceRow(value, 7)
		parsed := parsedCompleted{digest: value.Digest}
		var status string
		if err != nil || json.Unmarshal(row[1], &parsed.parent) != nil || json.Unmarshal(row[2], &parsed.taken) != nil || json.Unmarshal(row[3], &parsed.edge) != nil || json.Unmarshal(row[4], &parsed.subtree) != nil || json.Unmarshal(row[5], &parsed.terminalSet) != nil || json.Unmarshal(row[6], &status) != nil || status != "completed" || nodes[parsed.parent].state == "" || !digestText(parsed.taken) || subtrees[parsed.subtree].node == "" {
			return fmt.Errorf("invalid completed subtree")
		}
		if _, ok := terminalSets[parsed.terminalSet]; !ok {
			return fmt.Errorf("completed subtree references unknown terminal set")
		}
		edge := edges[parsed.edge]
		if edge.child == "" || edgeByParentTaken[parsed.parent+parsed.taken].digest != parsed.edge || subtrees[parsed.subtree].node != edge.child {
			return fmt.Errorf("completed subtree names wrong child")
		}
		if prior := terminalSetForSubtree[parsed.subtree]; prior != "" && prior != parsed.terminalSet {
			return fmt.Errorf("subtree has conflicting terminal-set authority")
		}
		key := parsed.parent + parsed.taken
		if completedByParentTaken[key].digest != "" || completedByEdge[parsed.edge].digest != "" {
			return fmt.Errorf("duplicate completed subtree")
		}
		terminalSetForSubtree[parsed.subtree] = parsed.terminalSet
		completed[value.Digest], completedByParentTaken[key] = parsed, parsed
		completedByEdge[parsed.edge] = parsed
	}
	for _, subtree := range subtrees {
		seen := map[string]bool{}
		for _, digest := range subtree.preorder {
			completion := completed[digest]
			if completion.digest == "" || completion.parent != subtree.node || seen[completion.taken] {
				return fmt.Errorf("subtree references invalid direct completion")
			}
			seen[completion.taken] = true
		}
	}

	propagations := map[string]parsedPropagation{}
	for _, propagation := range result.Propagations {
		row, err := evidenceRow(propagation, 9)
		parsed := parsedPropagation{digest: propagation.Digest}
		values := []*string{&parsed.parent, &parsed.taken, &parsed.sleeper, &parsed.source, &parsed.authority, &parsed.certificate, &parsed.successor, &parsed.childRemaining}
		for index := range values {
			if err != nil || json.Unmarshal(row[index+1], values[index]) != nil {
				return fmt.Errorf("invalid propagation field")
			}
		}
		edge := edgeByParentTaken[parsed.parent+parsed.taken]
		childProof := proofEntry(proofRows[nodes[edge.child].proofMap], parsed.sleeper)
		if edge.child == "" || parsed.taken == parsed.sleeper || childProof.PropagationDigest != parsed.digest || nodes[edge.child].state != parsed.successor || nodes[edge.child].remaining != parsed.childRemaining || !digestText(parsed.certificate) {
			return fmt.Errorf("invalid propagation edge binding")
		}
		if parsed.source == "prior-sleep" {
			if proofEntry(proofRows[nodes[parsed.parent].proofMap], parsed.sleeper).PropagationDigest != parsed.authority {
				return fmt.Errorf("prior-sleep propagation does not extend the parent proof")
			}
		} else if parsed.source == "earlier-sibling" {
			if completed[parsed.authority].digest == "" || completed[parsed.authority] != completedByParentTaken[parsed.parent+parsed.sleeper] {
				return fmt.Errorf("earlier-sibling propagation lacks the exact completed branch")
			}
		} else {
			return fmt.Errorf("invalid propagation source")
		}
		propagations[propagation.Digest] = parsed
	}

	rootSubtree := subtrees[result.RootSubtree.Digest]
	if rootSubtree.node != result.RootNodeDigest || len(proofRows[nodes[result.RootNodeDigest].proofMap]) != 0 {
		return fmt.Errorf("root subtree/node/proof authority mismatch")
	}
	visited := map[string]map[string]bool{
		"nodes": {}, "remaining": {}, "proofs": {}, "edges": {}, "propagations": {}, "completed": {}, "subtrees": {}, "terminal-sets": {}, "terminals": {},
	}
	verified := map[string]verifiedSubtree{}
	visiting := map[string]bool{}
	var verifySubtree func(string) (verifiedSubtree, error)
	verifySubtree = func(subtreeDigest string) (verifiedSubtree, error) {
		if value, ok := verified[subtreeDigest]; ok {
			return value, nil
		}
		if visiting[subtreeDigest] {
			return verifiedSubtree{}, fmt.Errorf("cyclic subtree authority")
		}
		visiting[subtreeDigest] = true
		defer delete(visiting, subtreeDigest)
		subtree := subtrees[subtreeDigest]
		node := nodes[subtree.node]
		setDigest := terminalSetForSubtree[subtreeDigest]
		if node.state == "" || setDigest == "" {
			return verifiedSubtree{}, fmt.Errorf("unbound subtree authority")
		}
		visited["subtrees"][subtreeDigest], visited["nodes"][subtree.node] = true, true
		visited["remaining"][node.remaining], visited["proofs"][node.proofMap], visited["terminal-sets"][setDigest] = true, true, true
		if len(subtree.preorder) == 0 {
			terminalDigest := terminalByNodeKey[node.state+node.remaining]
			want := []string{}
			histories := 0
			if terminalDigest != "" {
				want, histories = []string{terminalDigest}, 1
				visited["terminals"][terminalDigest] = true
			}
			if !slices.Equal(terminalSets[setDigest], want) {
				return verifiedSubtree{}, fmt.Errorf("leaf terminal set does not match its node")
			}
			value := verifiedSubtree{terminals: want, histories: histories}
			verified[subtreeDigest] = value
			return value, nil
		}
		seenTaken := map[string]bool{}
		var union []string
		histories := 0
		for _, completionDigest := range subtree.preorder {
			completedRow := completed[completionDigest]
			edge := edges[completedRow.edge]
			if completedRow.parent != subtree.node || edge.parent != subtree.node || edge.taken != completedRow.taken || seenTaken[edge.taken] {
				return verifiedSubtree{}, fmt.Errorf("subtree child list is not a direct ordered partition")
			}
			seenTaken[edge.taken] = true
			child, err := verifySubtree(completedRow.subtree)
			if err != nil {
				return verifiedSubtree{}, err
			}
			visited["edges"][edge.digest], visited["completed"][completedRow.digest] = true, true
			for _, propagation := range edge.propagations {
				if propagations[propagation].digest == "" {
					return verifiedSubtree{}, fmt.Errorf("edge references unresolved propagation")
				}
				visited["propagations"][propagation] = true
			}
			union = append(union, child.terminals...)
			histories += child.histories
		}
		slices.Sort(union)
		union = slices.Compact(union)
		if !slices.Equal(terminalSets[setDigest], union) {
			return verifiedSubtree{}, fmt.Errorf("subtree terminal set is not the exact child union")
		}
		value := verifiedSubtree{terminals: union, histories: histories}
		verified[subtreeDigest] = value
		return value, nil
	}
	root, err := verifySubtree(result.RootSubtree.Digest)
	if err != nil {
		return err
	}
	if !slices.Equal(root.terminals, result.TerminalDigests) || !slices.Equal(root.terminals, terminalSets[result.TerminalSet.Digest]) || root.histories != result.HistoryCount {
		return fmt.Errorf("result terminal or history summary mismatch")
	}
	for name, values := range map[string]map[string]any{
		"nodes": keys(nodes), "remaining": keys(remainingRows), "proofs": keys(proofRows), "edges": keys(edges),
		"propagations": keys(propagations), "completed": keys(completed), "subtrees": keys(subtrees), "terminal-sets": keys(terminalSets), "terminals": keys(terminals),
	} {
		for digest := range values {
			if !visited[name][digest] {
				return fmt.Errorf("unreachable retained %s evidence", name)
			}
		}
	}
	if result.ConstructedNodes != len(nodes) || result.Edges != len(edges) || result.NodeLookups != len(edges)+1 || result.SleepPropagations != len(propagations) {
		return fmt.Errorf("search result counters do not reconstruct from evidence")
	}
	return nil
}

func evidenceRow(value EvidenceObject, length int) ([]json.RawMessage, error) {
	if value.Digest != shaHex(value.Canonical) {
		return nil, fmt.Errorf("evidence digest mismatch")
	}
	var row []json.RawMessage
	if json.Unmarshal(value.Canonical, &row) != nil || len(row) != length {
		return nil, fmt.Errorf("invalid evidence row")
	}
	canonical, _ := json.Marshal(row)
	if !bytes.Equal(canonical, value.Canonical) {
		return nil, fmt.Errorf("noncanonical evidence")
	}
	return row, nil
}

func sortedUniqueOrEmpty(values []string) bool {
	for index, value := range values {
		if !digestText(value) || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}

func removeDigest(values []string, digest string) []string {
	result := make([]string, 0, len(values)-1)
	removed := false
	for _, value := range values {
		if !removed && value == digest {
			removed = true
			continue
		}
		result = append(result, value)
	}
	return result
}

func proofEntry(entries []ProofEntry, sleeper string) ProofEntry {
	for _, entry := range entries {
		if entry.SleeperDigest == sleeper {
			return entry
		}
	}
	return ProofEntry{}
}

func keys[K comparable, V any](values map[K]V) map[K]any {
	result := make(map[K]any, len(values))
	for key := range values {
		result[key] = nil
	}
	return result
}
