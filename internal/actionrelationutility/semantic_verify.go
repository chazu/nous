package actionrelationutility

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationoracle"
	"github.com/chazu/nous/internal/actionrelationsearch"
)

type parsedNode struct{ state, remaining, proofMap string }
type parsedEdge struct {
	digest, parent, taken, child string
	propagations                 []string
}
type parsedPropagation struct{ digest, parent, taken, sleeper, source, authority, certificate, successor, childRemaining string }
type parsedCompleted struct{ digest, parent, taken, edge, subtree, terminalSet string }
type parsedSubtree struct {
	digest, node string
	preorder     []string
}
type parsedTerminal struct {
	digest, state, terminal string
	remaining               []string
}

type semanticOccurrence struct {
	canonical []byte
	action    []byte
}

const semanticZeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

func VerifyCertificateDecisionSemantics(stateJSON, aOccurrenceJSON, bOccurrenceJSON []byte, operationRows []string, result, certificateDigest string, certificateCanonical []byte) error {
	if actionrelationoracle.ValidateState(stateJSON) != nil {
		return fmt.Errorf("certificate decision lacks state preimage")
	}
	a, err := parseSemanticOccurrence(aOccurrenceJSON)
	if err != nil {
		return err
	}
	b, err := parseSemanticOccurrence(bOccurrenceJSON)
	if err != nil || bytes.Compare(a.canonical, b.canonical) >= 0 {
		return fmt.Errorf("certificate decision pair is not canonical and distinct")
	}
	stateDigest, aDigest, bDigest := digestBytesText(stateJSON), digestBytesText(aOccurrenceJSON), digestBytesText(bOccurrenceJSON)
	aInitial, err := actionrelationoracle.Apply(stateJSON, a.action)
	if err != nil {
		return err
	}
	bInitial, err := actionrelationoracle.Apply(stateJSON, b.action)
	if err != nil {
		return err
	}
	rowDigest := func(row []any) string {
		canonical, _ := json.Marshal(row)
		return digestBytesText(canonical)
	}
	appRow := func(inputState, occurrence string, applicable bool) string {
		return rowDigest([]any{"action-applicability-row/v1", inputState, occurrence, applicable, "valid"})
	}
	transitionRow := func(inputState, occurrence, applicability string, transition actionrelationoracle.Transition) string {
		outcome, output := "inapplicable", semanticZeroDigest
		if transition.Applicable {
			outcome, output = "applied", digestBytesText(transition.State)
		}
		return rowDigest([]any{"action-transition-row/v1", inputState, occurrence, applicability, output, outcome})
	}
	aApp := appRow(stateDigest, aDigest, aInitial.Applicable)
	bApp := appRow(stateDigest, bDigest, bInitial.Applicable)
	wantRows := []string{aApp, bApp, transitionRow(stateDigest, aDigest, aApp, aInitial), transitionRow(stateDigest, bDigest, bApp, bInitial)}
	if aInitial.Applicable && bInitial.Applicable {
		bAfterA, err := actionrelationoracle.Apply(aInitial.State, b.action)
		if err != nil {
			return err
		}
		aAfterB, err := actionrelationoracle.Apply(bInitial.State, a.action)
		if err != nil {
			return err
		}
		aState, bState := digestBytesText(aInitial.State), digestBytesText(bInitial.State)
		bCrossApp := appRow(aState, bDigest, bAfterA.Applicable)
		aCrossApp := appRow(bState, aDigest, aAfterB.Applicable)
		wantRows = append(wantRows, bCrossApp, aCrossApp, transitionRow(aState, bDigest, bCrossApp, bAfterA), transitionRow(bState, aDigest, aCrossApp, aAfterB))
		if bAfterA.Applicable && aAfterB.Applicable {
			abDigest, baDigest := digestBytesText(bAfterA.State), digestBytesText(aAfterB.State)
			wantRows = append(wantRows, rowDigest([]any{"action-state-equality-row/v1", abDigest, baDigest, bytes.Equal(bAfterA.State, aAfterB.State), "valid"}))
		}
	}
	if !slices.Equal(operationRows, wantRows) {
		return fmt.Errorf("certificate operation rows do not independently reconstruct")
	}
	observation, err := actionrelationoracle.Observe(stateJSON, a.action, b.action)
	if err != nil {
		return err
	}
	if observation.Label == "commutes" {
		if result != "certified" || !digestTextUtility(certificateDigest) {
			return fmt.Errorf("commuting decision was not certified")
		}
		return verifySemanticCertificate(certificateCanonical, certificateDigest, stateDigest, aDigest, bDigest, a, b, observation)
	}
	if result != "not-certified" || certificateDigest != semanticZeroDigest || len(certificateCanonical) != 0 {
		return fmt.Errorf("noncommuting decision was falsely certified")
	}
	return nil
}

// VerifyResultSemantics independently replays every retained transition,
// terminal, sleeping proof, and certified local diamond from digest-addressed
// state, occurrence, and certificate preimages.
func VerifyResultSemantics(result actionrelationsearch.Result, canonicalByDigest map[string][]byte) error {
	if err := actionrelationsearch.VerifyResultEvidence(result); err != nil {
		return err
	}
	remaining := map[string][]string{}
	for _, object := range result.RemainingSets {
		row, _ := semanticEvidenceRow(object, 2)
		var values []string
		_ = json.Unmarshal(row[1], &values)
		remaining[object.Digest] = values
	}
	proofs := map[string][]actionrelationsearch.ProofEntry{}
	for _, object := range result.ProofMaps {
		row, _ := semanticEvidenceRow(object, 2)
		var raw [][]json.RawMessage
		_ = json.Unmarshal(row[1], &raw)
		for _, fields := range raw {
			proofs[object.Digest] = append(proofs[object.Digest], actionrelationsearch.ProofEntry{SleeperDigest: stringValue(fields[0]), PropagationDigest: stringValue(fields[1])})
		}
	}
	nodes := map[string]parsedNode{}
	for _, object := range result.Nodes {
		row, _ := semanticEvidenceRow(object, 4)
		nodes[object.Digest] = parsedNode{state: stringValue(row[1]), remaining: stringValue(row[2]), proofMap: stringValue(row[3])}
	}
	edges := map[string]parsedEdge{}
	edgesByParent := map[string][]parsedEdge{}
	for _, object := range result.SearchEdges {
		row, _ := semanticEvidenceRow(object, 5)
		parsed := parsedEdge{digest: object.Digest, parent: stringValue(row[1]), taken: stringValue(row[2]), child: stringValue(row[4])}
		_ = json.Unmarshal(row[3], &parsed.propagations)
		edges[object.Digest] = parsed
	}
	subtrees := map[string]parsedSubtree{}
	for _, object := range result.SubtreeRoots {
		row, _ := semanticEvidenceRow(object, 3)
		parsed := parsedSubtree{digest: object.Digest, node: stringValue(row[1])}
		_ = json.Unmarshal(row[2], &parsed.preorder)
		subtrees[object.Digest] = parsed
	}
	completed := map[string]parsedCompleted{}
	for _, object := range result.CompletedSubtrees {
		row, _ := semanticEvidenceRow(object, 7)
		parsed := parsedCompleted{digest: object.Digest, parent: stringValue(row[1]), taken: stringValue(row[2]), edge: stringValue(row[3]), subtree: stringValue(row[4]), terminalSet: stringValue(row[5])}
		completed[object.Digest] = parsed
	}
	for _, subtree := range subtrees {
		for _, completionDigest := range subtree.preorder {
			edge := edges[completed[completionDigest].edge]
			edgesByParent[subtree.node] = append(edgesByParent[subtree.node], edge)
		}
	}
	terminalsByNode := map[string]parsedTerminal{}
	for _, object := range result.TerminalBehaviors {
		row, _ := semanticEvidenceRow(object, 4)
		var stateJSON json.RawMessage
		var occurrenceDigests []string
		_ = json.Unmarshal(row[1], &stateJSON)
		_ = json.Unmarshal(row[2], &occurrenceDigests)
		remainingCanonical, _ := json.Marshal([]any{"remaining-occurrences/v1", occurrenceDigests})
		terminalsByNode[digestBytesText(stateJSON)+digestBytesText(remainingCanonical)] = parsedTerminal{digest: object.Digest, state: digestBytesText(stateJSON), remaining: occurrenceDigests, terminal: stringValue(row[3])}
	}
	propagations := map[string]parsedPropagation{}
	for _, object := range result.Propagations {
		row, _ := semanticEvidenceRow(object, 9)
		propagations[object.Digest] = parsedPropagation{
			digest: object.Digest, parent: stringValue(row[1]), taken: stringValue(row[2]), sleeper: stringValue(row[3]),
			source: stringValue(row[4]), authority: stringValue(row[5]), certificate: stringValue(row[6]), successor: stringValue(row[7]), childRemaining: stringValue(row[8]),
		}
	}
	states := map[string][]byte{}
	occurrences := map[string]semanticOccurrence{}
	resolveState := func(digest string) ([]byte, error) {
		if data := states[digest]; data != nil {
			return data, nil
		}
		data := canonicalByDigest[digest]
		if digestBytesText(data) != digest || actionrelationoracle.ValidateState(data) != nil {
			return nil, fmt.Errorf("missing independent state preimage %s", digest)
		}
		states[digest] = data
		return data, nil
	}
	resolveOccurrence := func(digest string) (semanticOccurrence, error) {
		if value := occurrences[digest]; value.canonical != nil {
			return value, nil
		}
		data := canonicalByDigest[digest]
		value, err := parseSemanticOccurrence(data)
		if err != nil || digestBytesText(data) != digest {
			return semanticOccurrence{}, fmt.Errorf("missing independent occurrence preimage %s", digest)
		}
		occurrences[digest] = value
		return value, nil
	}
	for nodeDigest, node := range nodes {
		stateJSON, err := resolveState(node.state)
		if err != nil {
			return err
		}
		remainingOccurrences := remaining[node.remaining]
		enabled := make([]string, 0, len(remainingOccurrences))
		for _, digest := range remainingOccurrences {
			occurrence, err := resolveOccurrence(digest)
			if err != nil {
				return err
			}
			transition, err := actionrelationoracle.Apply(stateJSON, occurrence.action)
			if err != nil {
				return err
			}
			if transition.Applicable {
				enabled = append(enabled, digest)
			}
		}
		slices.SortFunc(enabled, func(a, b string) int { return bytes.Compare(occurrences[a].canonical, occurrences[b].canonical) })
		sleepers := map[string]bool{}
		for _, proof := range proofs[node.proofMap] {
			sleepers[proof.SleeperDigest] = true
			if !slices.Contains(enabled, proof.SleeperDigest) {
				return fmt.Errorf("node retains a disabled sleeper")
			}
		}
		wantTaken := make([]string, 0, len(enabled))
		for _, digest := range enabled {
			if !sleepers[digest] {
				wantTaken = append(wantTaken, digest)
			}
		}
		gotEdges := edgesByParent[nodeDigest]
		gotTaken := make([]string, len(gotEdges))
		for index, edge := range gotEdges {
			gotTaken[index] = edge.taken
			taken, err := resolveOccurrence(edge.taken)
			if err != nil {
				return err
			}
			transition, err := actionrelationoracle.Apply(stateJSON, taken.action)
			if err != nil || !transition.Applicable || digestBytesText(transition.State) != nodes[edge.child].state {
				return fmt.Errorf("search edge is not the independent transition")
			}
		}
		if !slices.Equal(gotTaken, wantTaken) {
			return fmt.Errorf("search edges omit or reorder enabled non-sleepers")
		}
		terminal := terminalsByNode[node.state+node.remaining]
		if len(enabled) == 0 {
			if terminal.digest == "" {
				return fmt.Errorf("deadlocked node lacks terminal behavior")
			}
		} else if len(wantTaken) == 0 {
			if terminal.digest != "" {
				return fmt.Errorf("sleep-blocked node was reported as terminal")
			}
		} else if terminal.digest != "" {
			return fmt.Errorf("nonterminal node retained a terminal behavior")
		}
	}
	for _, propagation := range propagations {
		parent := nodes[propagation.parent]
		stateJSON, err := resolveState(parent.state)
		if err != nil {
			return err
		}
		taken, err := resolveOccurrence(propagation.taken)
		if err != nil {
			return err
		}
		sleeper, err := resolveOccurrence(propagation.sleeper)
		if err != nil {
			return err
		}
		observation, err := actionrelationoracle.Observe(stateJSON, taken.action, sleeper.action)
		if err != nil || observation.Label != "commutes" {
			return fmt.Errorf("sleep propagation lacks an independent local diamond")
		}
		if err := verifySemanticCertificate(canonicalByDigest[propagation.certificate], propagation.certificate, parent.state, propagation.taken, propagation.sleeper, taken, sleeper, observation); err != nil {
			return err
		}
		if propagation.source == "earlier-sibling" {
			order := edgesByParent[propagation.parent]
			if edgeIndex(order, propagation.sleeper) < 0 || edgeIndex(order, propagation.sleeper) >= edgeIndex(order, propagation.taken) {
				return fmt.Errorf("earlier-sibling proof does not precede the taken branch")
			}
		}
	}
	return nil
}

func parseSemanticOccurrence(data []byte) (semanticOccurrence, error) {
	var row []json.RawMessage
	var version string
	var ordinal int
	if json.Unmarshal(data, &row) != nil || len(row) != 3 || json.Unmarshal(row[0], &version) != nil || version != "action-occurrence/v1" || json.Unmarshal(row[2], &ordinal) != nil || ordinal < 0 || ordinal >= 8 || actionrelationoracle.ValidateAction(row[1]) != nil {
		return semanticOccurrence{}, fmt.Errorf("invalid semantic occurrence")
	}
	want, _ := json.Marshal(row)
	if !bytes.Equal(want, data) {
		return semanticOccurrence{}, fmt.Errorf("noncanonical semantic occurrence")
	}
	return semanticOccurrence{canonical: data, action: slices.Clone(row[1])}, nil
}

func verifySemanticCertificate(data []byte, certificateDigest, stateDigest, takenDigest, sleeperDigest string, taken, sleeper semanticOccurrence, observation actionrelationoracle.Observation) error {
	var row []json.RawMessage
	if json.Unmarshal(data, &row) != nil || len(row) != 10 || stringValue(row[0]) != "local-diamond-certificate/v1" || digestBytesText(data) != certificateDigest {
		return fmt.Errorf("sleep propagation lacks certificate preimage")
	}
	aDigest, bDigest := takenDigest, sleeperDigest
	abState, baState := observation.AB, observation.BA
	if bytes.Compare(taken.canonical, sleeper.canonical) > 0 {
		aDigest, bDigest = bDigest, aDigest
		abState, baState = baState, abState
	}
	var equal bool
	if stringValue(row[1]) != stateDigest || stringValue(row[2]) != aDigest || stringValue(row[3]) != bDigest || !digestTextUtility(stringValue(row[4])) || stringValue(row[5]) != digestBytesText(abState) || stringValue(row[6]) != digestBytesText(baState) || json.Unmarshal(row[7], &equal) != nil || !equal || stringValue(row[8]) != aDigest || !digestTextUtility(stringValue(row[9])) {
		return fmt.Errorf("certificate does not reconstruct its independent diamond")
	}
	want, _ := json.Marshal(row)
	if !bytes.Equal(want, data) {
		return fmt.Errorf("noncanonical certificate preimage")
	}
	return nil
}

func semanticEvidenceRow(object actionrelationsearch.EvidenceObject, length int) ([]json.RawMessage, error) {
	if object.Digest != digestBytesText(object.Canonical) {
		return nil, fmt.Errorf("semantic evidence digest changed")
	}
	var row []json.RawMessage
	if json.Unmarshal(object.Canonical, &row) != nil || len(row) != length {
		return nil, fmt.Errorf("invalid semantic evidence row")
	}
	want, _ := json.Marshal(row)
	if !bytes.Equal(want, object.Canonical) {
		return nil, fmt.Errorf("noncanonical semantic evidence row")
	}
	return row, nil
}

func edgeIndex(edges []parsedEdge, taken string) int {
	for index, edge := range edges {
		if edge.taken == taken {
			return index
		}
	}
	return -1
}

func stringValue(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}
