package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationoracle"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/actionrelationwire"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

var errRetainedDFSExhausted = errors.New("retained DFS exhausted")

type retainedDFSVisit struct {
	node, subtree, terminalSet string
	terminals                  []string
	histories                  int
}

type retainedDFSReplay struct {
	runID      string
	authority  retainedRunAuthority
	calls      []retainedCall
	objects    map[string]retainedObjectValue
	structural map[string]bool
	cursor     int
	memo       map[string]retainedDFSVisit
	consumed   map[string]bool
}

func verifyRetainedOrderedDFS(runID string, authority retainedRunAuthority, calls []retainedCall, objects map[string]retainedObjectValue, structural map[string]bool) error {
	r := &retainedDFSReplay{runID: runID, authority: authority, calls: calls, objects: objects, structural: structural, memo: map[string]retainedDFSVisit{}, consumed: map[string]bool{}}
	exhausted := false
	if authority.policy == "nous-guarded-sleep" || authority.policy == "no-guard-sleep" || authority.policy == "learned-no-use" {
		boundary := ""
		for digest, object := range objects {
			if object.kind != 35 {
				continue
			}
			value, err := decodeStoreBoundary(mustObjectRow(object.canonical))
			if err == nil && value.Curriculum == authority.curriculum && ((authority.policy == "no-guard-sleep" && value.Scope == "no-guard") || (authority.policy != "no-guard-sleep" && value.Scope == "nous")) {
				boundary = digest
			}
		}
		if boundary == "" {
			return fmt.Errorf("ordered DFS lacks its artifact boundary")
		}
		if _, err := r.consumeTask("artifact-load", []any{boundary, authority.artifact}, []uint8{10}); err != nil {
			if !errors.Is(err, errRetainedDFSExhausted) {
				return err
			}
			exhausted = true
		}
	}
	root := retainedDFSVisit{}
	if !exhausted {
		emptyProof := map[string]string{}
		var err error
		root, err = r.visit(authority.initialState, slices.Clone(authority.initialOccurrences), emptyProof)
		exhausted = errors.Is(err, errRetainedDFSExhausted)
		if err != nil && !exhausted {
			return err
		}
	}
	if authority.terminal == "completed" {
		if exhausted || r.cursor != len(calls) || root.terminalSet != authority.terminalSet || root.histories != authority.historyCount {
			return fmt.Errorf("completed transcript is not one exact rooted DFS")
		}
	} else if authority.terminal == "budget-exhausted" {
		if !exhausted || r.cursor != len(calls) {
			return fmt.Errorf("budget transcript is not one exact rooted DFS prefix")
		}
	} else {
		return fmt.Errorf("ordered DFS has unknown terminal")
	}
	for key := range structural {
		separator := -1
		for index := range key {
			if key[index] == ':' {
				separator = index
				break
			}
		}
		if separator < 1 {
			continue
		}
		var kind uint16
		if _, scanErr := fmt.Sscanf(key[:separator], "%d", &kind); scanErr != nil {
			continue
		}
		if slices.Contains([]uint16{5, 18, 19, 21, 22, 24, 25}, kind) && !r.consumed[key] {
			return fmt.Errorf("ordered DFS carries unreachable structural object %s", key)
		}
	}
	return nil
}

func (r *retainedDFSReplay) consumeTask(kind string, fields []any, codes []uint8) ([]retainedCall, error) {
	wire, _ := json.Marshal([]any{"actionrelation-utility-task/v1", r.runID, kind, fields, codes})
	return r.consumeReservation(kind, shaHex(wire), codes)
}

func (r *retainedDFSReplay) consumeReservation(kind, taskDigest string, codes []uint8) ([]retainedCall, error) {
	if r.cursor < len(r.calls) && r.cursor == len(r.calls)-1 && r.calls[r.cursor].operation == 19 && payloadTag(r.calls[r.cursor].payload) == "budget-terminal" {
		last := r.calls[r.cursor]
		terminal := r.objects[last.outputs[0]]
		value, terminalErr := decodeWorkTerminal(mustObjectRow(terminal.canonical))
		rejectedObject := r.objects[value.Rejected]
		rejected, rejectedErr := decodeReservation(rejectedObject.canonical)
		terminalReservationObject := r.objects[last.source]
		terminalReservation, reservationErr := decodeReservation(terminalReservationObject.canonical)
		terminalWire, _ := json.Marshal([]any{"actionrelation-budget-terminal-task/v1", r.runID, value.Rejected})
		if terminalErr != nil || rejectedErr != nil || reservationErr != nil || terminal.kind != 49 || rejectedObject.kind != 27 || terminalReservationObject.kind != 27 ||
			rejected.TaskDigest != taskDigest || !slices.Equal(rejected.OperationCodes, codes) || terminalReservation.TaskDigest != shaHex(terminalWire) || !slices.Equal(terminalReservation.OperationCodes, []uint8{19}) {
			return nil, fmt.Errorf("budget terminal is not the unique next %s task", kind)
		}
		r.cursor++
		return nil, errRetainedDFSExhausted
	}
	if r.cursor+len(codes) > len(r.calls) {
		return nil, fmt.Errorf("ordered DFS truncates %s task", kind)
	}
	block := r.calls[r.cursor : r.cursor+len(codes)]
	reservationObject := r.objects[block[0].source]
	reservation, err := decodeReservation(reservationObject.canonical)
	if err != nil || reservationObject.kind != 27 || reservation.TaskDigest != taskDigest || !slices.Equal(reservation.OperationCodes, codes) {
		return nil, fmt.Errorf("ordered DFS %s reservation differs from exact task", kind)
	}
	for index, code := range codes {
		if block[index].operation != code || block[index].source != block[0].source {
			return nil, fmt.Errorf("ordered DFS %s task has wrong call block", kind)
		}
	}
	r.cursor += len(codes)
	return block, nil
}

func (r *retainedDFSReplay) mark(kind uint16, digest string) error {
	key := fmt.Sprintf("%d:%s", kind, digest)
	object := r.objects[digest]
	if object.kind != kind || !r.structural[key] {
		return fmt.Errorf("ordered DFS object %s does not resolve in structural authority (object-kind=%d attributed=%t)", key, object.kind, r.structural[key])
	}
	r.consumed[key] = true
	return nil
}

func (r *retainedDFSReplay) visit(state string, remaining []string, proof map[string]string) (retainedDFSVisit, error) {
	slices.Sort(remaining)
	remainingCanonical, _ := json.Marshal([]any{"remaining-occurrences/v1", remaining})
	remainingDigest := shaHex(remainingCanonical)
	entries := make([][]string, 0, len(proof))
	for sleeper, propagation := range proof {
		entries = append(entries, []string{sleeper, propagation})
	}
	slices.SortFunc(entries, func(a, b []string) int { return compareString(a[0], b[0]) })
	proofCanonical, _ := json.Marshal([]any{"sleep-proof-map/v1", entries})
	proofDigest := shaHex(proofCanonical)
	nodeCanonical, _ := json.Marshal([]any{"sleep-search-node/v1", state, remainingDigest, proofDigest})
	nodeDigest := shaHex(nodeCanonical)
	block, err := r.consumeTask("node-lookup", []any{state, remainingDigest, proofDigest}, []uint8{16})
	if err != nil {
		return retainedDFSVisit{}, err
	}
	if len(block[0].payload) != 4 || !slices.Equal(block[0].outputs, []string{nodeDigest}) || !bytes.Equal(r.objects[nodeDigest].canonical, nodeCanonical) {
		return retainedDFSVisit{}, fmt.Errorf("ordered DFS node lookup differs from constructed node")
	}
	if prior, hit := r.memo[nodeDigest]; hit {
		if block[0].status != 3 {
			return retainedDFSVisit{}, fmt.Errorf("ordered DFS memo lookup is not a cache hit")
		}
		return prior, nil
	}
	if block[0].status != 1 {
		return retainedDFSVisit{}, fmt.Errorf("ordered DFS first node lookup is not constructed")
	}
	if err := r.mark(5, remainingDigest); err != nil {
		return retainedDFSVisit{}, err
	}
	if err := r.mark(19, proofDigest); err != nil {
		return retainedDFSVisit{}, err
	}
	semanticOrder := slices.Clone(remaining)
	slices.SortFunc(semanticOrder, func(a, b string) int { return bytes.Compare(r.objects[a].canonical, r.objects[b].canonical) })
	enabled := []string{}
	applicability := map[string]string{}
	for _, occurrence := range semanticOrder {
		calls, err := r.consumeTask("enabledness", []any{nodeDigest, occurrence}, []uint8{23})
		if err != nil {
			return retainedDFSVisit{}, err
		}
		call := calls[0]
		if len(call.payload) != 6 || rawText(call.payload[3]) != nodeDigest || rawText(call.payload[4]) != state || rawText(call.payload[5]) != occurrence || len(call.outputs) != 1 {
			return retainedDFSVisit{}, fmt.Errorf("ordered DFS enabledness differs from current node")
		}
		applicability[occurrence] = call.outputs[0]
		if result, valid := retainedOutputResult(call, r.objects, 3, 4); !valid {
			return retainedDFSVisit{}, fmt.Errorf("ordered DFS enabledness is invalid")
		} else if result {
			enabled = append(enabled, occurrence)
		}
	}
	if len(enabled) == 0 {
		rows := make([]string, len(semanticOrder))
		for index, occurrence := range semanticOrder {
			rows[index] = applicability[occurrence]
		}
		calls, err := r.consumeTask("terminal", []any{state, remainingDigest, rows}, []uint8{19})
		if err != nil {
			return retainedDFSVisit{}, err
		}
		terminal := calls[0].outputs[0]
		terminalSet, _ := actionrelationsearch.BuildTerminalSet([]string{terminal})
		subtree, _ := actionrelationsearch.BuildSubtreeRoot(nodeDigest, nil)
		if err := r.mark(24, terminalSet.Digest); err != nil {
			return retainedDFSVisit{}, err
		}
		if err := r.mark(25, subtree.Digest); err != nil {
			return retainedDFSVisit{}, err
		}
		value := retainedDFSVisit{node: nodeDigest, subtree: subtree.Digest, terminalSet: terminalSet.Digest, terminals: []string{terminal}, histories: 1}
		r.memo[nodeDigest] = value
		return value, nil
	}

	enabledSet := map[string]bool{}
	for _, digest := range enabled {
		enabledSet[digest] = true
	}
	earlier := []string{}
	earlierCompletions := map[string]string{}
	var allTerminals, completionDigests []string
	histories := 0
	for _, taken := range enabled {
		if proof[taken] != "" {
			continue
		}
		transitionCalls, err := r.consumeTask("transition", []any{nodeDigest, taken, applicability[taken]}, []uint8{11})
		if err != nil {
			return retainedDFSVisit{}, err
		}
		transition := transitionCalls[0]
		if len(transition.outputs) != 2 {
			return retainedDFSVisit{}, fmt.Errorf("ordered DFS transition lacks result state")
		}
		childState := transition.outputs[1]
		childRemaining := removeDigest(remaining, taken)
		childRemainingCanonical, _ := json.Marshal([]any{"remaining-occurrences/v1", childRemaining})
		childRemainingDigest := shaHex(childRemainingCanonical)
		candidates := make([]string, 0, len(proof)+len(earlier))
		if slices.Contains([]string{"static-rw-sleep", "dynamic-diamond-sleep", "nous-guarded-sleep", "no-guard-sleep"}, r.authority.policy) {
			for sleeper := range proof {
				candidates = append(candidates, sleeper)
			}
			candidates = append(candidates, earlier...)
			slices.Sort(candidates)
			candidates = slices.Compact(candidates)
		}
		childSet := map[string]bool{}
		for _, digest := range childRemaining {
			childSet[digest] = true
		}
		childProof := map[string]string{}
		for _, sleeper := range candidates {
			if !childSet[sleeper] || !enabledSet[sleeper] {
				continue
			}
			lookup, err := r.consumeTask("proof-map-lookup", []any{nodeDigest, proofDigest, sleeper}, []uint8{17})
			if err != nil {
				return retainedDFSVisit{}, err
			}
			if prior := proof[sleeper]; prior == "" && len(lookup[0].outputs) != 0 || prior != "" && !slices.Equal(lookup[0].outputs, []string{prior}) {
				return retainedDFSVisit{}, fmt.Errorf("ordered DFS proof lookup differs from current proof map")
			}
			eligibilityStart := r.cursor
			eligible, witness, err := r.consumeEligibility(nodeDigest, state, taken, sleeper, applicability[sleeper])
			if err != nil {
				return retainedDFSVisit{}, err
			}
			if !eligible {
				continue
			}
			witnessKind := uint16(16)
			switch r.authority.policy {
			case "static-rw-sleep":
				witnessKind = 15
			case "nous-guarded-sleep", "no-guard-sleep":
				witnessKind = 14
			}
			if err := r.mark(witnessKind, shaHex(witness)); err != nil {
				return retainedDFSVisit{}, fmt.Errorf("current eligibility witness: %w", err)
			}
			certified, certificate, err := r.consumeCertificate(state, taken, sleeper, witness, eligibilityStart)
			if err != nil {
				return retainedDFSVisit{}, err
			}
			if !certified {
				continue
			}
			source, sourceAuthority := "prior-sleep", proof[sleeper]
			if sourceAuthority == "" {
				source, sourceAuthority = "earlier-sibling", earlierCompletions[sleeper]
			}
			propagation, err := actionrelationsearch.BuildPropagation(nodeDigest, taken, sleeper, source, sourceAuthority, certificate, childState, childRemainingDigest)
			if err != nil {
				return retainedDFSVisit{}, err
			}
			if err := r.mark(18, propagation.Digest); err != nil {
				return retainedDFSVisit{}, err
			}
			childProof[sleeper] = propagation.Digest
		}
		child, err := r.visit(childState, childRemaining, childProof)
		if err != nil {
			return retainedDFSVisit{}, err
		}
		proofEntries := make([]actionrelationsearch.ProofEntry, 0, len(childProof))
		for sleeper, propagation := range childProof {
			proofEntries = append(proofEntries, actionrelationsearch.ProofEntry{SleeperDigest: sleeper, PropagationDigest: propagation})
		}
		edge, err := actionrelationsearch.BuildSearchEdge(nodeDigest, taken, proofEntries, child.node)
		if err != nil {
			return retainedDFSVisit{}, err
		}
		if err := r.mark(21, edge.Digest); err != nil {
			return retainedDFSVisit{}, err
		}
		completed, err := actionrelationsearch.BuildCompletedSubtree(nodeDigest, taken, edge, actionrelationsearch.EvidenceObject{Digest: child.subtree, Canonical: r.objects[child.subtree].canonical}, actionrelationsearch.EvidenceObject{Digest: child.terminalSet, Canonical: r.objects[child.terminalSet].canonical})
		if err != nil {
			return retainedDFSVisit{}, err
		}
		if err := r.mark(22, completed.Digest); err != nil {
			return retainedDFSVisit{}, err
		}
		completionDigests = append(completionDigests, completed.Digest)
		earlierCompletions[taken] = completed.Digest
		earlier = append(earlier, taken)
		allTerminals = append(allTerminals, child.terminals...)
		histories += child.histories
	}
	slices.Sort(allTerminals)
	allTerminals = slices.Compact(allTerminals)
	terminalSet, _ := actionrelationsearch.BuildTerminalSet(allTerminals)
	subtree, _ := actionrelationsearch.BuildSubtreeRoot(nodeDigest, completionDigests)
	if err := r.mark(24, terminalSet.Digest); err != nil {
		return retainedDFSVisit{}, fmt.Errorf("node=%s enabled=%v proof=%v completions=%v terminals=%v: %w", nodeDigest, enabled, proof, completionDigests, allTerminals, err)
	}
	if err := r.mark(25, subtree.Digest); err != nil {
		return retainedDFSVisit{}, err
	}
	value := retainedDFSVisit{node: nodeDigest, subtree: subtree.Digest, terminalSet: terminalSet.Digest, terminals: allTerminals, histories: histories}
	r.memo[nodeDigest] = value
	return value, nil
}

func (r *retainedDFSReplay) consumeEligibility(node, state, taken, sleeper, applicability string) (bool, []byte, error) {
	switch r.authority.policy {
	case "dynamic-diamond-sleep":
		witness, _ := json.Marshal([]any{"dynamic-witness/v1", "all-pairs", applicability})
		return true, witness, nil
	case "static-rw-sleep":
		calls, err := r.consumeTask("static-footprint", []any{node, taken, sleeper}, []uint8{24})
		if err != nil {
			return false, nil, err
		}
		result, valid := retainedOutputResult(calls[0], r.objects, 8, 9)
		if !valid {
			return false, nil, fmt.Errorf("ordered DFS static eligibility is invalid")
		}
		if !result {
			return false, nil, nil
		}
		witness, _ := json.Marshal([]any{"static-witness/v1", calls[0].outputs[0]})
		return true, witness, nil
	case "nous-guarded-sleep", "no-guard-sleep":
		artifact, err := actionrelations.ParseArtifact(r.objects[r.authority.artifact].canonical)
		if err != nil {
			return false, nil, err
		}
		codes := []uint8{21, 21}
		for _, digest := range artifact.RelationDigests {
			relation, err := actionrelations.ParseRelation(r.objects[digest].canonical)
			if err != nil {
				return false, nil, err
			}
			for range relation.Guard.Literals {
				codes = append(codes, 15)
			}
			codes = append(codes, 9)
		}
		calls, err := r.consumeTask("learned-match", []any{node, taken, sleeper, r.authority.artifact}, codes)
		if err != nil {
			return false, nil, err
		}
		attempt := decodedCertificateAttempt{State: state}
		left, right := taken, sleeper
		leftObject, rightObject := r.objects[left], r.objects[right]
		if bytes.Compare(leftObject.canonical, rightObject.canonical) > 0 {
			left, right = right, left
		}
		attempt.A, attempt.B = left, right
		count, ok := retainedEligibilitySchedule(r.authority, attempt, taken, sleeper, calls, r.objects, false)
		if !ok || count != len(calls) {
			return false, nil, fmt.Errorf("ordered DFS learned eligibility differs from artifact")
		}
		matched := true
		matchDigests := []string{}
		for _, call := range calls {
			if call.operation == 9 {
				value, valid := retainedOutputResult(call, r.objects, 10, 11)
				matched = matched && valid && value
				matchDigests = append(matchDigests, call.outputs[0])
			}
		}
		if !matched {
			return false, nil, nil
		}
		root, err := actionrelationwire.RootDigest("unanimous-relation-matches", matchDigests)
		if err != nil {
			return false, nil, err
		}
		barrier, _ := json.Marshal([]any{"action-unanimous-use/v1", r.authority.artifact, state, taken, sleeper, root, true, "valid"})
		if err := r.mark(43, shaHex(barrier)); err != nil {
			return false, nil, fmt.Errorf("current learned barrier: %w", err)
		}
		witness, _ := json.Marshal([]any{"learned-witness/v1", shaHex(barrier)})
		return true, witness, nil
	default:
		return false, nil, nil
	}
}

func (r *retainedDFSReplay) consumeCertificate(state, taken, sleeper string, witness []byte, operationStart int) (bool, string, error) {
	pair := sortedPair(taken, sleeper)
	lookupSequence := r.cursor
	lookupWire, _ := json.Marshal([]any{"actionrelation-cache-lookup-task/v1", r.runID, r.authority.world, r.authority.policy, state, pair[0], pair[1]})
	lookupBlock, err := r.consumeReservation("certificate-cache-lookup", shaHex(lookupWire), []uint8{18})
	if err != nil {
		return false, "", err
	}
	lookup := lookupBlock[0]
	if len(lookup.payload) != 6 || rawText(lookup.payload[1]) != r.authority.world || rawText(lookup.payload[2]) != r.authority.policy || rawText(lookup.payload[3]) != state || rawText(lookup.payload[4]) != pair[0] || rawText(lookup.payload[5]) != pair[1] {
		return false, "", fmt.Errorf("ordered DFS cache lookup differs from current pair")
	}
	var cacheDigest string
	wantCertificate := ""
	if lookup.status == 3 {
		if len(lookup.outputs) != 1 {
			return false, "", fmt.Errorf("ordered DFS cache hit lacks row")
		}
		cacheDigest = lookup.outputs[0]
	} else if lookup.status == 1 {
		request := fmt.Sprintf("AR.CertificateRequest.policy-c%04d-p%02d-w%d.%05d", r.authority.curriculum, retainedPolicyOrdinal(r.authority.policy), r.authority.worldOrdinal, lookupSequence)
		consumeStage := func(stage string, codes []uint8) ([]retainedCall, error) {
			wire, _ := json.Marshal([]any{"actionrelation-certificate-stage/v1", r.runID, request, stage, codes})
			return r.consumeReservation("certificate-"+stage, shaHex(wire), codes)
		}
		initial, err := consumeStage("initial", []uint8{13, 13, 12, 12})
		if err != nil {
			return false, "", err
		}
		if len(initial[2].outputs) == 2 && len(initial[3].outputs) == 2 {
			cross, err := consumeStage("cross", []uint8{13, 13, 12, 12})
			if err != nil {
				return false, "", err
			}
			if len(cross[2].outputs) == 2 && len(cross[3].outputs) == 2 {
				if _, err := consumeStage("equality", []uint8{14}); err != nil {
					return false, "", err
				}
			}
		}
		if operationStart < 0 || operationStart >= r.cursor {
			return false, "", fmt.Errorf("ordered DFS certificate has invalid operation start")
		}
		callIDs := make([]string, r.cursor-operationStart)
		for index, call := range r.calls[operationStart:r.cursor] {
			callIDs[index] = call.callID
		}
		root, err := BuildOperationRange(r.runID, 2, uint32(operationStart), callIDs)
		if err != nil {
			return false, "", err
		}
		attemptDigest, certificateDigest, err := r.reconstructCertificateDecision(state, taken, sleeper, witness, root.Digest)
		if err != nil {
			return false, "", err
		}
		finalizeWire, _ := json.Marshal([]any{"actionrelation-cache-finalize-task/v1", r.runID, r.authority.world, r.authority.policy, state, pair[0], pair[1], lookup.callID, attemptDigest, root.Digest})
		finalize, err := r.consumeReservation("certificate-cache-finalize", shaHex(finalizeWire), []uint8{25})
		if err != nil {
			return false, "", err
		}
		if len(finalize[0].payload) != 9 || rawText(finalize[0].payload[6]) != lookup.callID || rawText(finalize[0].payload[7]) != attemptDigest || rawText(finalize[0].payload[8]) != root.Digest || len(finalize[0].outputs) != 1 {
			return false, "", fmt.Errorf("ordered DFS fresh certificate lacks exact finalization")
		}
		cacheDigest = finalize[0].outputs[0]
		wantCertificate = certificateDigest
		if row := r.objects[cacheDigest]; row.kind != 26 {
			return false, "", fmt.Errorf("ordered DFS finalization lacks cache row")
		}
	} else {
		return false, "", fmt.Errorf("ordered DFS cache lookup has invalid status")
	}
	var row []json.RawMessage
	if object := r.objects[cacheDigest]; object.kind != 26 || json.Unmarshal(object.canonical, &row) != nil || len(row) != 12 {
		return false, "", fmt.Errorf("ordered DFS cache row does not decode")
	}
	if wantCertificate != "" && rawText(row[10]) != wantCertificate {
		return false, "", fmt.Errorf("ordered DFS cache row differs from reconstructed certificate")
	}
	return rawText(row[9]) == "certified", rawText(row[10]), nil
}

func (r *retainedDFSReplay) reconstructCertificateDecision(state, taken, sleeper string, witness []byte, operationRoot string) (string, string, error) {
	left, err := actionrelations.ParseOccurrence(r.objects[taken].canonical)
	if err != nil {
		return "", "", err
	}
	right, err := actionrelations.ParseOccurrence(r.objects[sleeper].canonical)
	if err != nil {
		return "", "", err
	}
	left, right, err = actionrelations.CanonicalPair(left, right)
	if err != nil {
		return "", "", err
	}
	aDigest, _ := left.Digest()
	bDigest, _ := right.Digest()
	aAction, _ := left.Action.CanonicalJSON()
	bAction, _ := right.Action.CanonicalJSON()
	observation, err := actionrelationoracle.Observe(r.objects[state].canonical, aAction, bAction)
	if err != nil {
		return "", "", err
	}
	witnessDigest := shaHex(witness)
	result, certificateDigest := "not-certified", zeroObjectDigest
	if observation.Label == "commutes" {
		certificate, _ := json.Marshal([]any{"local-diamond-certificate/v1", state, aDigest, bDigest, witnessDigest, shaHex(observation.AB), shaHex(observation.BA), true, aDigest, operationRoot})
		certificateDigest = shaHex(certificate)
		result = "certified"
	}
	attempt, _ := json.Marshal([]any{"local-diamond-certificate-attempt/v3", state, aDigest, bDigest, witnessDigest, operationRoot, result, certificateDigest, "valid"})
	return shaHex(attempt), certificateDigest, nil
}

func retainedPolicyOrdinal(policy string) int {
	switch policy {
	case "complete":
		return 0
	case "lexical-order":
		return 1
	case "static-rw-sleep":
		return 2
	case "dynamic-diamond-sleep":
		return 3
	case "nous-guarded-sleep":
		return 4
	case "no-guard-sleep":
		return 5
	case "learned-no-use":
		return 6
	default:
		return -1
	}
}

func removeDigest(values []string, remove string) []string {
	result := make([]string, 0, len(values)-1)
	removed := false
	for _, value := range values {
		if value == remove && !removed {
			removed = true
			continue
		}
		result = append(result, value)
	}
	return result
}
