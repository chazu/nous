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
	parent, taken, child string
	propagations         []string
}

func VerifyResultEvidence(result Result) error {
	collections := []struct {
		tag     string
		objects []EvidenceObject
	}{
		{"remaining-occurrences/v1", result.RemainingSets}, {"sleep-proof-map/v1", result.ProofMaps},
		{"sleep-search-node/v1", result.Nodes}, {"sleep-search-edge/v1", result.SearchEdges},
		{"sleep-propagation-core/v1", result.Propagations}, {"completed-subtree/v1", result.CompletedSubtrees},
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
		if err != nil {
			return err
		}
		parsed := parsedNode{}
		if json.Unmarshal(row[1], &parsed.state) != nil || json.Unmarshal(row[2], &parsed.remaining) != nil || json.Unmarshal(row[3], &parsed.proofMap) != nil || !digestText(parsed.state) || tags[parsed.remaining] != "remaining-occurrences/v1" || tags[parsed.proofMap] != "sleep-proof-map/v1" {
			return fmt.Errorf("invalid retained search node")
		}
		nodes[node.Digest] = parsed
	}

	edges := map[string]parsedEdge{}
	edgeByParentTaken := map[string]parsedEdge{}
	for _, edge := range result.SearchEdges {
		row, err := evidenceRow(edge, 5)
		if err != nil {
			return err
		}
		parsed := parsedEdge{}
		if json.Unmarshal(row[1], &parsed.parent) != nil || json.Unmarshal(row[2], &parsed.taken) != nil || json.Unmarshal(row[3], &parsed.propagations) != nil || json.Unmarshal(row[4], &parsed.child) != nil || nodes[parsed.parent].state == "" || nodes[parsed.child].state == "" || !digestText(parsed.taken) || !allDigests(parsed.propagations) {
			return fmt.Errorf("invalid retained search edge")
		}
		childProofs := proofRows[nodes[parsed.child].proofMap]
		want := make([]string, len(childProofs))
		for index, entry := range childProofs {
			want[index] = entry.PropagationDigest
		}
		if !slices.Equal(parsed.propagations, want) {
			return fmt.Errorf("edge/proof-map propagation mismatch")
		}
		edges[edge.Digest] = parsed
		edgeByParentTaken[parsed.parent+parsed.taken] = parsed
	}

	completed := map[string]bool{}
	for _, value := range result.CompletedSubtrees {
		row, err := evidenceRow(value, 6)
		if err != nil {
			return err
		}
		var parent, taken, subtree, terminalSet, status string
		if json.Unmarshal(row[1], &parent) != nil || json.Unmarshal(row[2], &taken) != nil || json.Unmarshal(row[3], &subtree) != nil || json.Unmarshal(row[4], &terminalSet) != nil || json.Unmarshal(row[5], &status) != nil || status != "completed" || nodes[parent].state == "" || !digestText(taken) || tags[subtree] != "sleep-subtree-root/v1" || tags[terminalSet] != "sleep-terminal-set/v1" {
			return fmt.Errorf("invalid completed subtree")
		}
		edge := edgeByParentTaken[parent+taken]
		if edge.child == "" {
			return fmt.Errorf("completed subtree lacks parent edge")
		}
		rootRow, _ := evidenceRow(objects[subtree], 3)
		var subtreeNode string
		_ = json.Unmarshal(rootRow[1], &subtreeNode)
		if subtreeNode != edge.child {
			return fmt.Errorf("completed subtree names wrong child")
		}
		completed[value.Digest] = true
	}

	for _, propagation := range result.Propagations {
		row, err := evidenceRow(propagation, 9)
		if err != nil {
			return err
		}
		var parent, taken, sleeper, source, authority, certificate, successor, childRemaining string
		values := []*string{&parent, &taken, &sleeper, &source, &authority, &certificate, &successor, &childRemaining}
		for index := range values {
			if json.Unmarshal(row[index+1], values[index]) != nil {
				return fmt.Errorf("invalid propagation field")
			}
		}
		edge := edgeByParentTaken[parent+taken]
		if edge.child == "" || taken == sleeper || !slices.Contains(edge.propagations, propagation.Digest) || nodes[edge.child].state != successor || nodes[edge.child].remaining != childRemaining || !digestText(certificate) || source == "prior-sleep" && tags[authority] != "sleep-propagation-core/v1" || source == "earlier-sibling" && !completed[authority] || source != "prior-sleep" && source != "earlier-sibling" {
			return fmt.Errorf("invalid propagation authority chain")
		}
	}

	for _, subtree := range result.SubtreeRoots {
		row, err := evidenceRow(subtree, 3)
		if err != nil {
			return err
		}
		var node string
		var preorder []string
		if json.Unmarshal(row[1], &node) != nil || json.Unmarshal(row[2], &preorder) != nil || nodes[node].state == "" || !allDigests(preorder) {
			return fmt.Errorf("invalid subtree root")
		}
		for _, digest := range preorder {
			if edges[digest].parent == "" {
				return fmt.Errorf("subtree references unknown edge")
			}
		}
	}
	for _, terminalSet := range result.TerminalSets {
		row, err := evidenceRow(terminalSet, 2)
		if err != nil {
			return err
		}
		var terminals []string
		if json.Unmarshal(row[1], &terminals) != nil || !sortedUniqueOrEmpty(terminals) {
			return fmt.Errorf("invalid terminal set rows")
		}
		for _, digest := range terminals {
			if tags[digest] != "action-terminal/v1" {
				return fmt.Errorf("terminal set references unknown behavior")
			}
		}
	}
	rootRow, _ := evidenceRow(result.RootSubtree, 3)
	var rootNode string
	_ = json.Unmarshal(rootRow[1], &rootNode)
	if rootNode != result.RootNodeDigest {
		return fmt.Errorf("root subtree/node mismatch")
	}
	terminalRow, _ := evidenceRow(result.TerminalSet, 2)
	var terminalDigests []string
	_ = json.Unmarshal(terminalRow[1], &terminalDigests)
	if !slices.Equal(terminalDigests, result.TerminalDigests) {
		return fmt.Errorf("result terminal-set mismatch")
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
