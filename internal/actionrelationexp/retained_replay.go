package actionrelationexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/actionrelationoracle"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/actionrelationwire"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type retainedTableLeaf struct {
	kind       uint16
	curriculum int
	scope      string
	manifest   string
	ordinal    uint32
	record     []byte
}

type retainedCall struct {
	sequence       int
	callID         string
	envelopeDigest string
	phase          uint8
	operation      uint8
	status         uint8
	counter        uint8
	source         string
	payload        []json.RawMessage
	outputs        []string
}

type retainedSemanticTableIndex struct {
	byKind     map[uint16]map[string][]retainedTableLeaf
	references map[int]map[int]retainedTableLeaf
}

func (index retainedSemanticTableIndex) resolve(sequence, payloadIndex int, kind uint16, digest string) []retainedTableLeaf {
	if reference, ok := index.references[sequence][payloadIndex]; ok {
		return []retainedTableLeaf{reference}
	}
	return index.byKind[kind][digest]
}

func decodeRetainedCalls(bundle TranscriptBundle) ([]retainedCall, error) {
	if len(bundle.JournalFiles) != len(bundle.InputFiles) || len(bundle.JournalFiles) != len(bundle.DetailFiles) {
		return nil, fmt.Errorf("unaligned transcript files")
	}
	result := make([]retainedCall, 0, bundle.JournalRoot.TotalRecords)
	for shardOrdinal := range bundle.JournalFiles {
		shard := bundle.JournalRoot.Shards[shardOrdinal]
		inputs, err := parseInputFrames(bundle.InputFiles[shardOrdinal].Data, shard.RecordCount)
		if err != nil {
			return nil, err
		}
		for local := 0; local < shard.RecordCount; local++ {
			sequence := int(shard.FirstSequence) + local
			journal := bundle.JournalFiles[shardOrdinal].Data[len(JournalHeader)+local*JournalRowBytes:][:JournalRowBytes]
			detail := bundle.DetailFiles[shardOrdinal].Data[len(DetailHeader)+local*DetailRowBytes:][:DetailRowBytes]
			var envelope []json.RawMessage
			var version, source string
			var phase, operation uint8
			if json.Unmarshal(inputs[local], &envelope) != nil || len(envelope) != 5 ||
				json.Unmarshal(envelope[0], &version) != nil || version != "action-charged-input/v1" ||
				json.Unmarshal(envelope[1], &phase) != nil || json.Unmarshal(envelope[2], &operation) != nil ||
				json.Unmarshal(envelope[3], &source) != nil {
				return nil, fmt.Errorf("invalid retained envelope %d", sequence)
			}
			var payload []json.RawMessage
			if json.Unmarshal(envelope[4], &payload) != nil || len(payload) == 0 {
				return nil, fmt.Errorf("invalid retained payload %d", sequence)
			}
			callHash := sha256.Sum256(journal)
			envelopeHash := sha256.Sum256(inputs[local])
			outputCount := int(detail[75])
			outputs := make([]string, outputCount)
			for index := range outputs {
				outputs[index] = hex.EncodeToString(detail[128+index*32 : 160+index*32])
			}
			result = append(result, retainedCall{
				sequence: sequence, callID: hex.EncodeToString(callHash[:]), envelopeDigest: hex.EncodeToString(envelopeHash[:]),
				phase: phase, operation: operation, status: journal[3], counter: journal[4], source: source,
				payload: payload, outputs: outputs,
			})
			if binary.BigEndian.Uint32(journal[8:12]) != uint32(sequence) || detail[34] != phase || detail[35] != operation || !bytes.Equal(detail[:32], callHash[:]) {
				return nil, fmt.Errorf("retained call identity mismatch %d", sequence)
			}
		}
	}
	if len(result) != bundle.JournalRoot.TotalRecords {
		return nil, fmt.Errorf("retained call count mismatch")
	}
	return result, nil
}

func buildRetainedSemanticTableIndex(authority retainedRunAuthority, calls []retainedCall, objects map[string]retainedObjectValue, tables map[string][]retainedTableLeaf) (retainedSemanticTableIndex, error) {
	result := retainedSemanticTableIndex{byKind: map[uint16]map[string][]retainedTableLeaf{}, references: map[int]map[int]retainedTableLeaf{}}
	add := func(kind uint16, canonical []byte, leaf retainedTableLeaf) {
		if result.byKind[kind] == nil {
			result.byKind[kind] = map[string][]retainedTableLeaf{}
		}
		digest := shaHex(canonical)
		result.byKind[kind][digest] = append(result.byKind[kind][digest], leaf)
	}
	leaf := func(digest string, kind uint16) (retainedTableLeaf, error) {
		var found retainedTableLeaf
		for _, value := range tables[digest] {
			if value.curriculum == authority.curriculum && value.scope == authority.policy && value.kind == kind {
				if found.record != nil {
					return retainedTableLeaf{}, fmt.Errorf("semantic index output resolves more than once")
				}
				found = value
			}
		}
		if found.record == nil {
			return retainedTableLeaf{}, fmt.Errorf("semantic index output lacks table kind %d", kind)
		}
		return found, nil
	}
	for _, values := range tables {
		for _, value := range values {
			if value.curriculum == authority.curriculum && value.scope == authority.policy && value.kind == 105 {
				canonical, err := observationCanonical(value.record)
				if err != nil {
					return retainedSemanticTableIndex{}, err
				}
				add(105, canonical, value)
			}
		}
	}
	pendingApplicability := map[string][]retainedTableLeaf{}
	for _, call := range calls {
		d := func(index int) string { return rawText(call.payload[index]) }
		boolAt := func(index int) bool { var value bool; _ = json.Unmarshal(call.payload[index], &value); return value }
		intAt := func(index int) int { var value int; _ = json.Unmarshal(call.payload[index], &value); return value }
		output := func(index int, kind uint16) (retainedTableLeaf, error) {
			if index >= len(call.outputs) {
				return retainedTableLeaf{}, fmt.Errorf("semantic index call %d lacks output %d", call.sequence, index)
			}
			return leaf(call.outputs[index], kind)
		}
		switch call.operation {
		case 1:
			candidate, err := output(1, 103)
			if err != nil {
				return retainedSemanticTableIndex{}, err
			}
			wire, _ := json.Marshal([]any{"action-guard-candidate/v1", call.outputs[0], "", d(1), 0, 0})
			add(103, wire, candidate)
		case 2:
			candidate, err := output(0, 103)
			guard := objects[d(2)]
			parsed, guardErr := actionrelations.ParseGuard(guard.canonical)
			if err != nil || guard.kind != 7 || guardErr != nil {
				return retainedSemanticTableIndex{}, fmt.Errorf("semantic index candidate does not resolve")
			}
			wire, _ := json.Marshal([]any{"action-guard-candidate/v1", d(2), d(3), d(1), intAt(4), len(parsed.Literals)})
			add(103, wire, candidate)
		case 4:
			transition, err := output(0, 107)
			if err != nil {
				return retainedSemanticTableIndex{}, err
			}
			pending := pendingApplicability[d(3)]
			if len(pending) == 0 {
				return retainedSemanticTableIndex{}, fmt.Errorf("training transition lacks preceding applicability leaf")
			}
			result.references[call.sequence] = map[int]retainedTableLeaf{3: pending[0]}
			pendingApplicability[d(3)] = pending[1:]
			resultState, outcome := zeroObjectDigest, "inapplicable"
			if transition.record[3] == 1 {
				resultState, outcome = digestAt(transition.record, 68), "applied"
			}
			wire, _ := json.Marshal([]any{"action-transition-row/v1", d(1), d(2), d(3), resultState, outcome})
			add(107, wire, transition)
		case 5:
			applicability, err := output(0, 107)
			if err != nil {
				return retainedSemanticTableIndex{}, err
			}
			wire, _ := json.Marshal([]any{"action-applicability-row/v1", d(1), d(2), applicability.record[3] == 1, "valid"})
			add(107, wire, applicability)
			digest := shaHex(wire)
			pendingApplicability[digest] = append(pendingApplicability[digest], applicability)
		case 6:
			equality, err := output(0, 107)
			if err != nil {
				return retainedSemanticTableIndex{}, err
			}
			wire, _ := json.Marshal([]any{"action-state-equality-row/v1", d(1), d(2), equality.record[3] == 1, "valid"})
			add(107, wire, equality)
		case 7:
			literal, err := output(0, 101)
			if err != nil {
				return retainedSemanticTableIndex{}, err
			}
			wire, _ := json.Marshal([]any{"action-guard-literal-row/v1", d(1), d(2), d(3), d(4), d(5), boolAt(6), literal.record[67] == 1})
			add(101, wire, literal)
		case 20:
			candidateResult, err := output(0, 108)
			if err != nil {
				return retainedSemanticTableIndex{}, err
			}
			var guardResults []string
			_ = json.Unmarshal(call.payload[2], &guardResults)
			wire, _ := json.Marshal([]any{"action-candidate-result/v1", d(1), guardResults, int(candidateResult.record[64]), int(candidateResult.record[65]), candidateResult.record[66] == 1, candidateResult.record[67] == 1, d(3)})
			add(108, wire, candidateResult)
		case 22:
			guardResult, err := output(0, 102)
			if err != nil {
				return retainedSemanticTableIndex{}, err
			}
			var literals []string
			_ = json.Unmarshal(call.payload[3], &literals)
			wire, _ := json.Marshal([]any{"action-guard-result/v1", d(1), d(2), literals, guardResult.record[64] == 1})
			add(102, wire, guardResult)
		}
	}
	for _, pending := range pendingApplicability {
		if len(pending) != 0 {
			return retainedSemanticTableIndex{}, fmt.Errorf("training applicability leaf lacks its transition")
		}
	}
	return result, nil
}

func candidateObjectDigest(record []byte) string {
	parent := ""
	if !allZero(record[32:64]) {
		parent = digestAt(record, 32)
	}
	wire, _ := json.Marshal([]any{"action-guard-candidate/v1", digestAt(record, 0), parent, digestAt(record, 64), int(binary.BigEndian.Uint16(record[96:98])), int(record[98])})
	return shaHex(wire)
}

func observationCanonical(record []byte) ([]byte, error) {
	if ValidateTableRecord(105, record) != nil {
		return nil, fmt.Errorf("invalid semantic observation record")
	}
	row := []any{"action-pair-observation/v1", digestAt(record, 4), digestAt(record, 36), digestAt(record, 68)}
	for bit, start := range []int{100, 132, 164, 196, 228, 260} {
		if record[3]&(1<<bit) != 0 {
			row = append(row, nil)
		} else {
			row = append(row, digestAt(record, start))
		}
	}
	labels := []string{"", "commutes", "a-enables-b", "b-enables-a", "a-disables-b", "b-disables-a", "mutual-disables", "inapplicable", "conflicts", "invalid"}
	row = append(row, labels[int(record[1])])
	return json.Marshal(row)
}

func verifyRetainedRunReplay(record RunEvidenceRecord, authority retainedRunAuthority, calls []retainedCall, objects map[string]retainedObjectValue, tables map[string][]retainedTableLeaf, structural map[string]bool) error {
	if len(calls) == 0 || authority.phase < 1 || authority.phase > 2 {
		return fmt.Errorf("empty or invalid run authority")
	}
	semanticTables := retainedSemanticTableIndex{}
	if authority.phase == 1 {
		var err error
		semanticTables, err = buildRetainedSemanticTableIndex(authority, calls, objects, tables)
		if err != nil {
			return err
		}
	}
	var work [12]int
	for _, call := range calls {
		if call.sequence < 0 || call.phase != authority.phase || call.counter != operationCounters[call.operation] {
			return fmt.Errorf("call %d differs from run phase/counter", call.sequence)
		}
		work[call.counter-1]++
		if err := verifyRetainedCallTypes(authority, call, objects, tables, semanticTables); err != nil {
			return fmt.Errorf("call %d operation %d: %w", call.sequence, call.operation, err)
		}
		if err := verifyRetainedCallSemantics(authority, call, objects, tables, semanticTables); err != nil {
			return fmt.Errorf("call %d semantics: %w", call.sequence, err)
		}
	}
	if work != authority.work {
		return fmt.Errorf("charged work vector does not match score row")
	}

	total := sumInts(authority.initialWork[:])
	seen := map[string]bool{}
	for cursor := 0; cursor < len(calls); {
		first := calls[cursor]
		object, ok := objects[first.source]
		if !ok || object.kind != 27 || seen[first.source] {
			return fmt.Errorf("call %d lacks unique retained reservation", cursor)
		}
		reservation, err := decodeReservation(object.canonical)
		if err != nil || reservation.Digest != first.source || reservation.RunID != record.RunID || reservation.Status != "reserved" || reservation.TotalBefore != total {
			return fmt.Errorf("call %d has invalid reservation authority", cursor)
		}
		if authority.phase == 1 {
			if len(reservation.OperationCodes) != 1 || reservation.TaskDigest != actionrelationledger.TaskDigest(record.RunID, cursor, first.operation) {
				return fmt.Errorf("acquisition reservation %d is not its exact single call", cursor)
			}
		}
		if cursor+len(reservation.OperationCodes) > len(calls) {
			return fmt.Errorf("reservation at %d is only partially consumed", cursor)
		}
		for index, code := range reservation.OperationCodes {
			call := calls[cursor+index]
			if call.source != first.source || call.operation != code {
				return fmt.Errorf("reservation at %d does not exactly cover its call block", cursor)
			}
		}
		seen[first.source] = true
		total = reservation.TotalAfter
		cursor += len(reservation.OperationCodes)
	}
	if total != sumInts(authority.initialWork[:])+len(calls) {
		return fmt.Errorf("reservation totals do not conserve charged work")
	}
	if authority.phase == 1 {
		if err := verifyRetainedAcquisitionClosure(record.RunID, authority, calls, objects, tables); err != nil {
			return err
		}
	}
	if authority.phase == 2 && authority.terminal == "completed" && structural != nil {
		if err := verifyRetainedCacheRanges(record.RunID, calls, objects); err != nil {
			return err
		}
		if err := verifyRetainedSearchGraph(authority, calls, objects, structural); err != nil {
			return err
		}
	}

	if authority.workTerminal == zeroObjectDigest {
		if record.WorkTerminal != "" || authority.terminal == "budget-exhausted" {
			return fmt.Errorf("missing score-row work terminal")
		}
		return nil
	}
	if record.WorkTerminal != authority.workTerminal || authority.terminal != "budget-exhausted" || authority.phase != 2 {
		return fmt.Errorf("unexpected work-terminal authority")
	}
	terminalObject, ok := objects[authority.workTerminal]
	if !ok || terminalObject.kind != 49 {
		return fmt.Errorf("work terminal does not resolve")
	}
	terminal, err := decodeWorkTerminal(mustObjectRow(terminalObject.canonical))
	if err != nil || terminal.RunID != record.RunID || terminal.Phase != 2 || terminal.Work != addWork(authority.initialWork, authority.work) || terminal.Total != total {
		return fmt.Errorf("work terminal does not reconstruct cumulative work")
	}
	last := calls[len(calls)-1]
	if last.operation != 19 || last.status != 1 || !slices.Equal(last.outputs, []string{authority.workTerminal}) || payloadTag(last.payload) != "budget-terminal" || len(last.payload) != 2 || rawText(last.payload[1]) != terminal.Rejected {
		return fmt.Errorf("work terminal is not the final aligned code-19 output")
	}
	rejectedObject, ok := objects[terminal.Rejected]
	if !ok || rejectedObject.kind != 27 || seen[terminal.Rejected] {
		return fmt.Errorf("rejected reservation is missing or was charged")
	}
	rejected, err := decodeReservation(rejectedObject.canonical)
	terminalReservation, terminalErr := decodeReservation(objects[last.source].canonical)
	if err != nil || terminalErr != nil || rejected.Status != "rejected-cap" || rejected.RunID != record.RunID || rejected.TotalBefore != terminalReservation.TotalBefore || rejected.TotalAfter != rejected.TotalBefore || !slices.Equal(terminalReservation.OperationCodes, []uint8{19}) || terminalReservation.TotalAfter != rejected.TotalBefore+1 || rejected.TotalBefore+len(rejected.OperationCodes) < terminalReservation.TotalAfter {
		return fmt.Errorf("budget terminal reservations do not reconstruct")
	}
	return nil
}

func verifyRetainedCacheRanges(runID string, calls []retainedCall, objects map[string]retainedObjectValue) error {
	callByID := map[string]retainedCall{}
	finalized := map[string]string{}
	misses := map[string]bool{}
	keyFor := func(payload []json.RawMessage) string {
		return rawText(payload[1]) + rawText(payload[2]) + rawText(payload[3]) + rawText(payload[4]) + rawText(payload[5])
	}
	for _, call := range calls {
		callByID[call.callID] = call
		if call.operation == 18 && call.status == 1 {
			key := keyFor(call.payload)
			if misses[call.callID] || finalized[key] != "" {
				return fmt.Errorf("duplicate cache miss or miss after finalization")
			}
			misses[call.callID] = true
		}
		if call.operation == 18 && call.status == 3 {
			key := keyFor(call.payload)
			if finalized[key] == "" || !slices.Equal(call.outputs, []string{finalized[key]}) {
				return fmt.Errorf("cache hit precedes or differs from finalization")
			}
		}
		if call.operation != 25 {
			continue
		}
		p := call.payload
		key := keyFor(p)
		missID, attemptDigest, rootDigest := rawText(p[6]), rawText(p[7]), rawText(p[8])
		miss, ok := callByID[missID]
		if !ok || !misses[missID] || miss.operation != 18 || miss.status != 1 || len(miss.outputs) != 0 || keyFor(miss.payload) != key || finalized[key] != "" {
			return fmt.Errorf("cache finalization lacks its unique prior miss")
		}
		rootObject, rootOK := objects[rootDigest]
		if !rootOK || rootObject.kind != 46 {
			return fmt.Errorf("certificate range root does not resolve")
		}
		root, err := decodeOperationRoot(mustObjectRow(rootObject.canonical))
		if err != nil || root.Variant != "range" || root.RunID != runID || root.Phase != 2 || root.Count < 1 || int(root.Start)+root.Count != call.sequence || int(root.Start) > miss.sequence || miss.sequence >= call.sequence {
			return fmt.Errorf("certificate operation range has wrong bounds")
		}
		callIDs := make([]string, root.Count)
		operationRows := []string{}
		missCount := 0
		for index := 0; index < root.Count; index++ {
			item := calls[int(root.Start)+index]
			callIDs[index] = item.callID
			if item.operation == 18 {
				missCount++
				if item.callID != missID || item.status != 1 || len(item.outputs) != 0 {
					return fmt.Errorf("certificate range contains wrong cache miss")
				}
			}
			if item.operation == 25 {
				return fmt.Errorf("certificate range includes finalization")
			}
			if item.operation == 12 || item.operation == 13 || item.operation == 14 {
				if len(item.outputs) == 0 {
					return fmt.Errorf("certificate proof operation lacks row")
				}
				operationRows = append(operationRows, item.outputs[0])
			}
		}
		wantRoot, err := BuildOperationRange(runID, 2, root.Start, callIDs)
		if err != nil || wantRoot.Digest != rootDigest || missCount != 1 {
			return fmt.Errorf("certificate operation root does not reconstruct")
		}
		attemptObject, attemptOK := objects[attemptDigest]
		if !attemptOK || attemptObject.kind != 44 {
			return fmt.Errorf("certificate attempt does not resolve")
		}
		attempt, err := decodeCertificateAttempt(mustObjectRow(attemptObject.canonical))
		if err != nil || !certificateAttemptMatchesFinalization(attempt, p, rootDigest) {
			return fmt.Errorf("certificate attempt differs from finalization")
		}
		stateObject, stateOK := objects[attempt.State]
		aObject, aOK := objects[attempt.A]
		bObject, bOK := objects[attempt.B]
		if !stateOK || stateObject.kind != 1 || !aOK || aObject.kind != 3 || !bOK || bObject.kind != 3 {
			return fmt.Errorf("certificate semantic preimages do not resolve")
		}
		var certificate []byte
		if attempt.Certificate != zeroObjectDigest {
			certificateObject, ok := objects[attempt.Certificate]
			if !ok || certificateObject.kind != 17 {
				return fmt.Errorf("certified attempt lacks certificate preimage")
			}
			certificate = certificateObject.canonical
		}
		if err := VerifyCertificateDecisionSemantics(stateObject.canonical, aObject.canonical, bObject.canonical, operationRows, attempt.Result, attempt.Certificate, certificate); err != nil {
			return fmt.Errorf("certificate decision: %w", err)
		}
		finalized[key] = call.outputs[0]
		delete(misses, missID)
	}
	if len(misses) != 0 {
		return fmt.Errorf("completed run retains unfinalized cache miss")
	}
	return nil
}

func certificateAttemptMatchesFinalization(attempt decodedCertificateAttempt, payload []json.RawMessage, rootDigest string) bool {
	if len(payload) != 9 {
		return false
	}
	pair := []string{attempt.A, attempt.B}
	return attempt.State == rawText(payload[3]) && attempt.Operation == rootDigest && slices.Min(pair) == rawText(payload[4]) && slices.Max(pair) == rawText(payload[5])
}

func verifyRetainedCallSemantics(authority retainedRunAuthority, call retainedCall, objects map[string]retainedObjectValue, tables map[string][]retainedTableLeaf, semanticTables retainedSemanticTableIndex) error {
	if call.status != 1 && !(call.status == 3 && (call.operation == 16 || call.operation == 18)) {
		return fmt.Errorf("typed retained call has invalid semantic status")
	}
	p := call.payload
	if authority.phase == 1 {
		return verifyRetainedAcquisitionCall(authority.policy, authority.curriculum, call, objects, tables, semanticTables)
	}
	d := func(index int) string { return rawText(p[index]) }
	canonical := func(digest string, kind uint16) ([]byte, error) {
		object, ok := objects[digest]
		if !ok || object.kind != kind || shaHex(object.canonical) != digest {
			return nil, fmt.Errorf("digest does not resolve kind %d", kind)
		}
		return object.canonical, nil
	}
	exactOutput := func(rows ...[]byte) error {
		want := make([]string, len(rows))
		for index, row := range rows {
			want[index] = shaHex(row)
		}
		if !slices.Equal(call.outputs, want) {
			return fmt.Errorf("outputs do not equal semantic reconstruction")
		}
		return nil
	}
	applicability := func(stateDigest, occurrenceDigest string) ([]byte, bool, error) {
		state, err := canonical(stateDigest, 1)
		if err != nil {
			return nil, false, err
		}
		occurrence, err := canonical(occurrenceDigest, 3)
		if err != nil {
			return nil, false, err
		}
		action, err := occurrenceAction(occurrence)
		if err != nil {
			return nil, false, err
		}
		transition, err := actionrelationoracle.Apply(state, action)
		if err != nil {
			return nil, false, err
		}
		row, _ := json.Marshal([]any{"action-applicability-row/v1", stateDigest, occurrenceDigest, transition.Applicable, "valid"})
		return row, transition.Applicable, nil
	}

	switch call.operation {
	case 9:
		return verifyRelationMatchCall(call, objects)
	case 10:
		artifact, err := canonical(d(2), 10)
		if err != nil || len(call.outputs) != 1 || call.outputs[0] != d(2) {
			return fmt.Errorf("artifact load does not return its frozen artifact")
		}
		return exactOutput(artifact)
	case 11, 12:
		state, err := canonical(d(1), 1)
		if err != nil {
			return err
		}
		occurrence, err := canonical(d(2), 3)
		if err != nil {
			return err
		}
		appRow, applicable, err := applicability(d(1), d(2))
		if err != nil || shaHex(appRow) != d(3) {
			return fmt.Errorf("transition names wrong applicability authority")
		}
		action, _ := occurrenceAction(occurrence)
		transition, err := actionrelationoracle.Apply(state, action)
		if err != nil || transition.Applicable != applicable {
			return fmt.Errorf("transition does not replay")
		}
		outcome, resultDigest := "inapplicable", zeroObjectDigest
		if transition.Applicable {
			outcome, resultDigest = "applied", shaHex(transition.State)
		}
		row, _ := json.Marshal([]any{"action-transition-row/v1", d(1), d(2), d(3), resultDigest, outcome})
		if transition.Applicable {
			return exactOutput(row, transition.State)
		}
		return exactOutput(row)
	case 13, 21:
		row, _, err := applicability(d(1), d(2))
		if err != nil {
			return err
		}
		return exactOutput(row)
	case 14:
		left, err := canonical(d(1), 1)
		if err != nil {
			return err
		}
		right, err := canonical(d(2), 1)
		if err != nil {
			return err
		}
		row, _ := json.Marshal([]any{"action-state-equality-row/v1", d(1), d(2), bytes.Equal(left, right), "valid"})
		return exactOutput(row)
	case 15:
		return verifyLiteralCall(call, objects)
	case 16:
		row, _ := json.Marshal([]any{"sleep-search-node/v1", d(1), d(2), d(3)})
		return exactOutput(row)
	case 17:
		if len(call.outputs) == 0 {
			return nil
		}
		proof, err := canonical(d(2), 19)
		if err != nil {
			return err
		}
		var row []json.RawMessage
		var entries [][]string
		if json.Unmarshal(proof, &row) != nil || len(row) != 2 || json.Unmarshal(row[1], &entries) != nil {
			return fmt.Errorf("proof-map lookup has invalid map")
		}
		found := ""
		for _, entry := range entries {
			if len(entry) == 2 && entry[0] == d(3) {
				found = entry[1]
			}
		}
		if found == "" || !slices.Equal(call.outputs, []string{found}) {
			return fmt.Errorf("proof-map lookup output is not the named sleeper entry")
		}
	case 18:
		if d(1) != authority.world || d(2) != authority.policy {
			return fmt.Errorf("certificate cache lookup crossed run context")
		}
		if call.status == 1 {
			if len(call.outputs) != 0 {
				return fmt.Errorf("cache miss produced an output")
			}
			return nil
		}
		if len(call.outputs) != 1 || !cacheRowMatches(objects[call.outputs[0]].canonical, p) {
			return fmt.Errorf("cache hit does not return the exact retained key")
		}
	case 19:
		if payloadTag(p) == "budget-terminal" {
			return nil
		}
		state, err := canonical(d(1), 1)
		if err != nil {
			return err
		}
		remainingBytes, err := canonical(d(2), 5)
		if err != nil {
			return err
		}
		var remainingRow []json.RawMessage
		var remaining []string
		var applicabilityDigests []string
		if json.Unmarshal(remainingBytes, &remainingRow) != nil || len(remainingRow) != 2 || json.Unmarshal(remainingRow[1], &remaining) != nil || json.Unmarshal(p[3], &applicabilityDigests) != nil || len(applicabilityDigests) != len(remaining) {
			return fmt.Errorf("terminal remaining/applicability coverage changed")
		}
		semanticOrder := slices.Clone(remaining)
		slices.SortFunc(semanticOrder, func(a, b string) int { return bytes.Compare(objects[a].canonical, objects[b].canonical) })
		for index, digest := range applicabilityDigests {
			want, applicable, err := applicability(d(1), semanticOrder[index])
			if err != nil || applicable || shaHex(want) != digest {
				return fmt.Errorf("terminal names non-deadlocked applicability")
			}
		}
		terminal := "deadlock"
		if len(remaining) == 0 {
			terminal = "complete"
		}
		row, _ := json.Marshal([]any{"action-terminal/v1", json.RawMessage(state), remaining, terminal})
		return exactOutput(row)
	case 23:
		if d(1) != authority.world || d(2) != authority.policy {
			return fmt.Errorf("search applicability crossed run context")
		}
		node, err := canonical(d(3), 20)
		if err != nil {
			return err
		}
		var nodeRow []json.RawMessage
		var nodeState, remainingDigest string
		if json.Unmarshal(node, &nodeRow) != nil || len(nodeRow) != 4 || json.Unmarshal(nodeRow[1], &nodeState) != nil || json.Unmarshal(nodeRow[2], &remainingDigest) != nil || nodeState != d(4) {
			return fmt.Errorf("search applicability node/state mismatch")
		}
		remainingBytes, err := canonical(remainingDigest, 5)
		if err != nil {
			return err
		}
		var remainingRow []json.RawMessage
		var remaining []string
		if json.Unmarshal(remainingBytes, &remainingRow) != nil || len(remainingRow) != 2 || json.Unmarshal(remainingRow[1], &remaining) != nil || !slices.Contains(remaining, d(5)) {
			return fmt.Errorf("search applicability occurrence is absent from node")
		}
		row, _, err := applicability(d(4), d(5))
		if err != nil {
			return err
		}
		return exactOutput(row)
	case 24:
		if d(1) != authority.world || d(2) == "" || authority.policy != "static-rw-sleep" {
			return fmt.Errorf("static footprint crossed run context")
		}
		aFacts, err := exactFacts(d(3), d(4), d(6), objects)
		if err != nil {
			return err
		}
		bFacts, err := exactFacts(d(3), d(5), d(7), objects)
		if err != nil {
			return err
		}
		result, err := actionrelations.EvaluateAtom("read-write-disjoint", aFacts, bFacts)
		if err != nil {
			return err
		}
		row, _ := json.Marshal([]any{"action-static-footprint-row/v1", d(1), d(2), d(3), d(4), d(5), d(6), d(7), result, "valid"})
		return exactOutput(row)
	case 25:
		if d(1) != authority.world || d(2) != authority.policy || len(call.outputs) != 1 || !cacheFinalizationMatches(objects[call.outputs[0]].canonical, p, objects) {
			return fmt.Errorf("cache finalization does not close exact key/attempt/root")
		}
	}
	return nil
}

func occurrenceAction(canonical []byte) ([]byte, error) {
	var row []json.RawMessage
	if json.Unmarshal(canonical, &row) != nil || len(row) != 3 || rawText(row[0]) != "action-occurrence/v1" {
		return nil, fmt.Errorf("invalid occurrence preimage")
	}
	action, err := json.Marshal(row[1])
	if err != nil || actionrelationoracle.ValidateAction(action) != nil {
		return nil, fmt.Errorf("invalid occurrence action")
	}
	return action, nil
}

func verifyRetainedAcquisitionCall(scope string, curriculum int, call retainedCall, objects map[string]retainedObjectValue, tables map[string][]retainedTableLeaf, semanticTables retainedSemanticTableIndex) error {
	p := call.payload
	d := func(index int) string { return rawText(p[index]) }
	leaf := func(digest string, kind uint16) (retainedTableLeaf, error) {
		var result retainedTableLeaf
		for _, value := range tables[digest] {
			if value.kind == kind && value.curriculum == curriculum && value.scope == scope {
				if result.record != nil {
					return retainedTableLeaf{}, fmt.Errorf("table leaf resolves more than once in acquisition scope")
				}
				result = value
			}
		}
		if result.record == nil {
			return retainedTableLeaf{}, fmt.Errorf("table leaf does not resolve kind %d", kind)
		}
		return result, nil
	}
	semanticLeaf := func(digest string, kind uint16, payloadIndex int) (retainedTableLeaf, error) {
		values := semanticTables.resolve(call.sequence, payloadIndex, kind, digest)
		if len(values) != 1 {
			return retainedTableLeaf{}, fmt.Errorf("semantic digest does not resolve exactly one table kind %d", kind)
		}
		return values[0], nil
	}
	object := func(digest string, kind uint16) ([]byte, error) {
		value, ok := objects[digest]
		if !ok || value.kind != kind || shaHex(value.canonical) != digest {
			return nil, fmt.Errorf("object does not resolve kind %d", kind)
		}
		return value.canonical, nil
	}
	outputLeaf := func(index int, kind uint16) (retainedTableLeaf, error) {
		if index >= len(call.outputs) {
			return retainedTableLeaf{}, fmt.Errorf("missing table output")
		}
		return leaf(call.outputs[index], kind)
	}
	intAt := func(index int) int {
		var value int
		_ = json.Unmarshal(p[index], &value)
		return value
	}
	boolAt := func(index int) bool {
		var value bool
		_ = json.Unmarshal(p[index], &value)
		return value
	}

	switch call.operation {
	case 1:
		guard, err := object(call.outputs[0], 7)
		candidate, leafErr := outputLeaf(1, 103)
		emptyGuard, _ := json.Marshal([]any{"action-guard/v1", []any{}})
		if err != nil || leafErr != nil || !bytes.Equal(guard, emptyGuard) || digestAt(candidate.record, 0) != call.outputs[0] || !allZero(candidate.record[32:64]) || digestAt(candidate.record, 64) != d(1) || binary.BigEndian.Uint16(candidate.record[96:98]) != 0 || candidate.record[98] != 0 || candidate.record[99] != 1 {
			return fmt.Errorf("guard root/candidate do not reconstruct")
		}
	case 2:
		candidate, err := outputLeaf(0, 103)
		parent, parentErr := semanticLeaf(d(3), 103, 3)
		if err != nil || parentErr != nil || digestAt(candidate.record, 0) != d(2) || digestAt(candidate.record, 32) != candidateObjectDigest(parent.record) || digestAt(candidate.record, 64) != d(1) || int(binary.BigEndian.Uint16(candidate.record[96:98])) != intAt(4) || candidate.record[99] != 1 {
			return fmt.Errorf("allocated candidate does not reconstruct")
		}
		guard, err := object(d(2), 7)
		if err != nil {
			return err
		}
		parsed, _ := actionrelations.ParseGuard(guard)
		if candidate.record[98] != byte(len(parsed.Literals)) {
			return fmt.Errorf("candidate literal count differs from guard")
		}
	case 3:
		parentBytes, err := object(d(1), 7)
		if err != nil {
			return err
		}
		parent, err := actionrelations.ParseGuard(parentBytes)
		if err != nil {
			return err
		}
		child := parent
		child.Literals = append(slices.Clone(parent.Literals), actionrelations.Literal{Atom: d(2), Polarity: boolAt(3)})
		childBytes, err := child.CanonicalJSON()
		edge, edgeErr := outputLeaf(1, 104)
		if err != nil || edgeErr != nil || call.outputs[0] != shaHex(childBytes) || digestAt(edge.record, 0) != d(1) || digestAt(edge.record, 32) != call.outputs[0] || int(binary.BigEndian.Uint16(edge.record[64:66])) != slices.Index(actionrelations.Atoms, d(2))+1 || edge.record[66] != boolByte(boolAt(3)) || edge.record[67] != 1 || int(binary.BigEndian.Uint32(edge.record[68:72])) != intAt(4) {
			return fmt.Errorf("guard refinement does not reconstruct")
		}
	case 4:
		app, err := semanticLeaf(d(3), 107, 3)
		transition, transitionErr := outputLeaf(0, 107)
		state, stateErr := object(d(1), 1)
		occurrence, occurrenceErr := object(d(2), 3)
		if err != nil || transitionErr != nil || stateErr != nil || occurrenceErr != nil || app.record[1] != 1 || digestAt(app.record, 4) != d(1) || digestAt(app.record, 36) != d(2) || transition.record[1] != 2 || digestAt(transition.record, 4) != d(1) || digestAt(transition.record, 36) != d(2) {
			return fmt.Errorf("training transition inputs do not reconstruct")
		}
		action, _ := occurrenceAction(occurrence)
		result, replayErr := actionrelationoracle.Apply(state, action)
		if replayErr != nil || app.record[3] != boolByte(result.Applicable) {
			return fmt.Errorf("training transition applicability changed")
		}
		if result.Applicable {
			if transition.record[3] != 1 || digestAt(transition.record, 68) != shaHex(result.State) || len(call.outputs) != 2 || call.outputs[1] != shaHex(result.State) {
				return fmt.Errorf("applied training transition changed output")
			}
		} else if transition.record[3] != 2 || !allZero(transition.record[68:100]) || len(call.outputs) != 1 {
			return fmt.Errorf("inapplicable training transition changed output")
		}
	case 5:
		row, err := outputLeaf(0, 107)
		state, stateErr := object(d(1), 1)
		occurrence, occurrenceErr := object(d(2), 3)
		if err != nil || stateErr != nil || occurrenceErr != nil || row.record[1] != 1 || digestAt(row.record, 4) != d(1) || digestAt(row.record, 36) != d(2) {
			return fmt.Errorf("training applicability inputs do not reconstruct")
		}
		action, _ := occurrenceAction(occurrence)
		result, replayErr := actionrelationoracle.Apply(state, action)
		if replayErr != nil || row.record[3] != boolByte(result.Applicable) {
			return fmt.Errorf("training applicability changed result")
		}
	case 6:
		row, err := outputLeaf(0, 107)
		left, leftErr := object(d(1), 1)
		right, rightErr := object(d(2), 1)
		if err != nil || leftErr != nil || rightErr != nil || row.record[1] != 3 || digestAt(row.record, 4) != d(1) || digestAt(row.record, 36) != d(2) || row.record[3] != boolByte(bytes.Equal(left, right)) {
			return fmt.Errorf("training equality changed result")
		}
	case 7:
		row, err := outputLeaf(0, 101)
		observation, observationErr := semanticLeaf(d(2), 105, 2)
		guardBytes, guardErr := object(d(1), 7)
		if err != nil || observationErr != nil || guardErr != nil || digestAt(row.record, 0) != d(1) || digestAt(row.record, 32) != d(2) || int(binary.BigEndian.Uint16(row.record[64:66])) != slices.Index(actionrelations.Atoms, d(5))+1 || row.record[66] != boolByte(boolAt(6)) {
			return fmt.Errorf("training literal identity does not reconstruct")
		}
		guard, err := actionrelations.ParseGuard(guardBytes)
		if err != nil || !slices.Contains(guard.Literals, actionrelations.Literal{Atom: d(5), Polarity: boolAt(6)}) {
			return fmt.Errorf("training literal is absent from guard")
		}
		stateDigest := digestAt(observation.record, 4)
		aOccurrence, bOccurrence := digestAt(observation.record, 36), digestAt(observation.record, 68)
		aFacts, err := exactFacts(stateDigest, aOccurrence, d(3), objects)
		if err != nil {
			return err
		}
		bFacts, err := exactFacts(stateDigest, bOccurrence, d(4), objects)
		if err != nil {
			return err
		}
		pairRoot, _ := actionrelationwire.RootDigest("local-fact-pair", []string{d(3), d(4)})
		value, err := actionrelations.EvaluateAtom(d(5), aFacts, bFacts)
		if err != nil || digestAt(row.record, 96) != pairRoot || row.record[67] != boolByte(value == boolAt(6)) {
			return fmt.Errorf("training literal changed semantic result")
		}
	case 8:
		barrierBytes, err := object(d(1), 28)
		artifactBytes, artifactErr := object(call.outputs[0], 10)
		if err != nil || artifactErr != nil {
			return fmt.Errorf("artifact freeze inputs do not resolve")
		}
		var barrierRow []json.RawMessage
		var winners []string
		if json.Unmarshal(barrierBytes, &barrierRow) != nil || len(barrierRow) != 6 || json.Unmarshal(barrierRow[4], &winners) != nil {
			return fmt.Errorf("artifact barrier does not decode")
		}
		var payloadWinners []string
		_ = json.Unmarshal(p[2], &payloadWinners)
		artifact, err := actionrelations.ParseArtifact(artifactBytes)
		if err != nil || !slices.Equal(winners, payloadWinners) || artifact.SemanticTrainingRoot != d(3) || len(artifact.RelationDigests) != len(winners) {
			return fmt.Errorf("artifact does not close barrier winners")
		}
		relations := map[string]actionrelations.Relation{}
		for _, digest := range artifact.RelationDigests {
			canonical, err := object(digest, 9)
			if err != nil {
				return err
			}
			relations[digest], err = actionrelations.ParseRelation(canonical)
			if err != nil {
				return err
			}
		}
		if artifact.ValidateResolved(relations) != nil {
			return fmt.Errorf("artifact relations do not resolve")
		}
	case 20:
		candidate, err := semanticLeaf(d(1), 103, 1)
		result, resultErr := outputLeaf(0, 108)
		var guardResults []string
		if json.Unmarshal(p[2], &guardResults) != nil || err != nil || resultErr != nil || len(guardResults) != 16 || digestAt(result.record, 0) != d(1) {
			return fmt.Errorf("candidate result inputs do not reconstruct")
		}
		for _, digest := range guardResults {
			if _, err := semanticLeaf(digest, 102, 2); err != nil {
				return err
			}
		}
		root, _ := actionrelationwire.RootDigest("guard-result-vector", guardResults)
		if digestAt(result.record, 32) != root || result.record[66] != 1 || result.record[68] != 1 || result.record[67] > 1 || result.record[64] > 16 || result.record[65] > 16 || candidate.record[98] > 2 {
			return fmt.Errorf("candidate result summary does not reconstruct")
		}
	case 22:
		guardBytes, err := object(d(1), 7)
		result, resultErr := outputLeaf(0, 102)
		var literals []string
		if json.Unmarshal(p[3], &literals) != nil || err != nil || resultErr != nil {
			return fmt.Errorf("guard-result inputs do not reconstruct")
		}
		guard, err := actionrelations.ParseGuard(guardBytes)
		if err != nil || len(literals) != len(guard.Literals) || digestAt(result.record, 0) != d(1) {
			return fmt.Errorf("guard-result literal coverage changed")
		}
		_, err = semanticLeaf(d(2), 105, 2)
		if err != nil || digestAt(result.record, 32) != d(2) {
			return fmt.Errorf("guard-result observation changed")
		}
		value := true
		for index, digest := range literals {
			literal, err := semanticLeaf(digest, 101, 3)
			if err != nil || digestAt(literal.record, 0) != d(1) || digestAt(literal.record, 32) != d(2) || int(binary.BigEndian.Uint16(literal.record[64:66])) != slices.Index(actionrelations.Atoms, guard.Literals[index].Atom)+1 || literal.record[66] != boolByte(guard.Literals[index].Polarity) {
				return fmt.Errorf("guard-result literal %d changed", index)
			}
			value = value && literal.record[67] == 1
		}
		if result.record[64] != boolByte(value) || result.record[65] != 1 {
			return fmt.Errorf("guard-result conjunction changed")
		}
	default:
		return fmt.Errorf("unsupported acquisition operation")
	}
	return nil
}

func verifyRetainedSearchGraph(authority retainedRunAuthority, calls []retainedCall, objects map[string]retainedObjectValue, structural map[string]bool) error {
	toEvidence := func(digest string, kind uint16) (actionrelationsearch.EvidenceObject, error) {
		object, ok := objects[digest]
		if !ok || object.kind != kind {
			return actionrelationsearch.EvidenceObject{}, fmt.Errorf("search digest does not resolve kind %d", kind)
		}
		return actionrelationsearch.EvidenceObject{Canonical: slices.Clone(object.canonical), Digest: digest}, nil
	}
	collections := map[uint16][]actionrelationsearch.EvidenceObject{}
	seen := map[string]bool{}
	add := func(kind uint16, digest string) error {
		key := fmt.Sprintf("%d:%s", kind, digest)
		if seen[key] {
			return nil
		}
		value, err := toEvidence(digest, kind)
		if err != nil {
			return err
		}
		seen[key] = true
		collections[kind] = append(collections[kind], value)
		return nil
	}
	for digest, object := range objects {
		if !structural[fmt.Sprintf("%d:%s", object.kind, digest)] {
			continue
		}
		switch object.kind {
		case 5, 18, 19, 21, 22, 24, 25:
			if err := add(object.kind, digest); err != nil {
				return err
			}
		}
	}
	rootNode := ""
	for _, call := range calls {
		if call.operation == 16 {
			if rootNode == "" {
				rootNode = call.outputs[0]
			}
			if err := add(20, call.outputs[0]); err != nil {
				return err
			}
		}
		if call.operation == 19 && payloadTag(call.payload) == "terminal-construct" {
			if err := add(23, call.outputs[0]); err != nil {
				return err
			}
		}
	}
	rootSubtree := actionrelationsearch.EvidenceObject{}
	for _, object := range collections[25] {
		var row []json.RawMessage
		if json.Unmarshal(object.Canonical, &row) == nil && len(row) == 3 && rawText(row[1]) == rootNode {
			if rootSubtree.Digest != "" {
				return fmt.Errorf("multiple root subtrees")
			}
			rootSubtree = object
		}
	}
	terminalSet, err := toEvidence(authority.terminalSet, 24)
	if err != nil {
		return fmt.Errorf("score-row terminal set: %w", err)
	}
	var terminalSetRow []json.RawMessage
	var terminalDigests []string
	if json.Unmarshal(terminalSet.Canonical, &terminalSetRow) != nil || len(terminalSetRow) != 2 || json.Unmarshal(terminalSetRow[1], &terminalDigests) != nil {
		return fmt.Errorf("score-row terminal set does not decode")
	}
	result := actionrelationsearch.Result{
		Policy: actionrelationsearch.Policy(authority.policy), TerminalDigests: terminalDigests,
		NodeLookups: lenOperation(calls, 16), ConstructedNodes: len(collections[20]),
		SleepPropagations: len(collections[18]), HistoryCount: authority.historyCount, Edges: len(collections[21]),
		RootNodeDigest: rootNode, RootSubtree: rootSubtree, TerminalSet: terminalSet,
		RemainingSets: collections[5], Propagations: collections[18], ProofMaps: collections[19], Nodes: collections[20],
		SearchEdges: collections[21], CompletedSubtrees: collections[22], TerminalBehaviors: collections[23],
		TerminalSets: collections[24], SubtreeRoots: collections[25],
	}
	if err := actionrelationsearch.VerifyResultEvidence(result); err != nil {
		return fmt.Errorf("search graph does not reconstruct: %w", err)
	}

	type nodeValue struct{ state, remaining, proof string }
	nodes := map[string]nodeValue{}
	for _, object := range collections[20] {
		var row []json.RawMessage
		_ = json.Unmarshal(object.Canonical, &row)
		nodes[object.Digest] = nodeValue{rawText(row[1]), rawText(row[2]), rawText(row[3])}
	}
	remaining := map[string][]string{}
	for _, object := range collections[5] {
		var row []json.RawMessage
		var values []string
		_ = json.Unmarshal(object.Canonical, &row)
		_ = json.Unmarshal(row[1], &values)
		remaining[object.Digest] = values
	}
	proofs := map[string]map[string]bool{}
	for _, object := range collections[19] {
		var row []json.RawMessage
		var entries [][]string
		_ = json.Unmarshal(object.Canonical, &row)
		_ = json.Unmarshal(row[1], &entries)
		proofs[object.Digest] = map[string]bool{}
		for _, entry := range entries {
			proofs[object.Digest][entry[0]] = true
		}
	}
	type edgeValue struct{ parent, taken, child string }
	edges := map[string]edgeValue{}
	edgeByPair := map[string]string{}
	wantTransitions := map[string]int{}
	for _, object := range collections[21] {
		var row []json.RawMessage
		_ = json.Unmarshal(object.Canonical, &row)
		edge := edgeValue{rawText(row[1]), rawText(row[2]), rawText(row[4])}
		edges[object.Digest] = edge
		edgeByPair[edge.parent+edge.taken] = object.Digest
		parent, child := nodes[edge.parent], nodes[edge.child]
		parentState := objects[parent.state].canonical
		occurrence := objects[edge.taken].canonical
		action, actionErr := occurrenceAction(occurrence)
		transition, transitionErr := actionrelationoracle.Apply(parentState, action)
		if actionErr != nil || transitionErr != nil || !transition.Applicable || shaHex(transition.State) != child.state {
			return fmt.Errorf("search edge transition does not reconstruct")
		}
		wantTransitions[parent.state+edge.taken+child.state]++
	}
	gotTransitions := map[string]int{}
	applicability := map[string]bool{}
	for _, call := range calls {
		switch call.operation {
		case 11:
			resultState := zeroObjectDigest
			if len(call.outputs) == 2 {
				resultState = call.outputs[1]
			}
			gotTransitions[rawText(call.payload[1])+rawText(call.payload[2])+resultState]++
		case 23:
			key := rawText(call.payload[3]) + rawText(call.payload[5])
			if _, duplicate := applicability[key]; duplicate {
				return fmt.Errorf("duplicate search applicability call")
			}
			var row []json.RawMessage
			var value bool
			object := objects[call.outputs[0]]
			if json.Unmarshal(object.canonical, &row) != nil || len(row) != 5 || json.Unmarshal(row[3], &value) != nil {
				return fmt.Errorf("search applicability output does not decode")
			}
			applicability[key] = value
		}
	}
	if !equalIntMap(wantTransitions, gotTransitions) {
		return fmt.Errorf("charged transitions do not exactly cover search edges")
	}
	for nodeDigest, node := range nodes {
		for _, occurrence := range remaining[node.remaining] {
			applicable, ok := applicability[nodeDigest+occurrence]
			if !ok {
				return fmt.Errorf("search node/remaining lacks applicability call")
			}
			_, hasEdge := edgeByPair[nodeDigest+occurrence]
			if hasEdge != (applicable && !proofs[node.proof][occurrence]) {
				return fmt.Errorf("search branch differs from applicability/sleep authority")
			}
			delete(applicability, nodeDigest+occurrence)
		}
	}
	if len(applicability) != 0 {
		return fmt.Errorf("unbound search applicability calls")
	}
	return nil
}

func verifyRetainedAcquisitionClosure(runID string, authority retainedRunAuthority, calls []retainedCall, objects map[string]retainedObjectValue, tables map[string][]retainedTableLeaf) error {
	type namedLeaf struct {
		digest string
		value  retainedTableLeaf
	}
	byKind := map[uint16][]namedLeaf{}
	for digest, candidates := range tables {
		for _, candidate := range candidates {
			if candidate.curriculum == authority.curriculum && candidate.scope == authority.policy {
				byKind[candidate.kind] = append(byKind[candidate.kind], namedLeaf{digest, candidate})
			}
		}
	}
	for kind, values := range byKind {
		slices.SortFunc(values, func(a, b namedLeaf) int { return int(a.value.ordinal) - int(b.value.ordinal) })
		for ordinal, value := range values {
			if value.value.ordinal != uint32(ordinal) {
				return fmt.Errorf("acquisition table %d ordinals are not complete", kind)
			}
		}
		byKind[kind] = values
	}
	wantCounts := map[uint16]int{101: 13_920, 102: 7_216, 103: 451, 104: 450, 105: 16, 106: 32, 108: 451}
	if authority.policy == "no-guard" {
		wantCounts = map[uint16]int{102: 16, 103: 1, 105: 16, 106: 32, 108: 1}
	}
	for kind, count := range wantCounts {
		if len(byKind[kind]) != count {
			return fmt.Errorf("acquisition table %d count=%d want=%d", kind, len(byKind[kind]), count)
		}
	}
	for kind := range byKind {
		if kind != 107 {
			if _, ok := wantCounts[kind]; !ok {
				return fmt.Errorf("unexpected acquisition table %d", kind)
			}
		}
	}
	trainingCount := len(byKind[107])
	if trainingCount < 64 || trainingCount > 144 {
		return fmt.Errorf("acquisition training-operation count=%d", trainingCount)
	}

	leafOutputs := map[uint16][]string{}
	for _, call := range calls {
		kind := map[uint8]uint16{1: 103, 2: 103, 3: 104, 4: 107, 5: 107, 6: 107, 7: 101, 20: 108, 22: 102}[call.operation]
		if kind != 0 {
			output := call.outputs[0]
			if call.operation == 1 || call.operation == 3 {
				output = call.outputs[1]
			}
			leafOutputs[kind] = append(leafOutputs[kind], output)
		}
	}
	for kind, values := range byKind {
		want := make([]string, len(values))
		for index, value := range values {
			want[index] = value.digest
		}
		if kind != 105 && kind != 106 && !slices.Equal(leafOutputs[kind], want) {
			return fmt.Errorf("charged outputs do not exactly cover table %d", kind)
		}
	}

	cursor := 0
	for ordinal, observation := range byKind[105] {
		record := observation.value.record
		count := 4
		if record[3]&(1<<2) == 0 {
			count += 2
		}
		if record[3]&(1<<3) == 0 {
			count += 2
		}
		if record[3]&(1<<4) == 0 && record[3]&(1<<5) == 0 {
			count++
		}
		rootDigest := digestAt(record, 292)
		rootObject, ok := objects[rootDigest]
		if !ok || rootObject.kind != 46 {
			return fmt.Errorf("observation %d operation root is absent", ordinal)
		}
		root, err := decodeOperationRoot(mustObjectRow(rootObject.canonical))
		if err != nil || root.Variant != "range" || root.RunID != runID || root.Phase != 1 || int(root.Start) != cursor || root.Count != count || cursor+count > len(calls) {
			return fmt.Errorf("observation %d operation range changed", ordinal)
		}
		wantCodes := []uint8{5, 5, 4, 4}
		if record[3]&(1<<2) == 0 {
			wantCodes = append(wantCodes, 5, 4)
		}
		if record[3]&(1<<3) == 0 {
			wantCodes = append(wantCodes, 5, 4)
		}
		if record[3]&(1<<4) == 0 && record[3]&(1<<5) == 0 {
			wantCodes = append(wantCodes, 6)
		}
		ids := make([]string, count)
		for index, code := range wantCodes {
			if calls[cursor+index].operation != code {
				return fmt.Errorf("observation %d operation schedule changed", ordinal)
			}
			ids[index] = calls[cursor+index].callID
		}
		wantRoot, err := BuildOperationRange(runID, 1, uint32(cursor), ids)
		if err != nil || wantRoot.Digest != rootDigest {
			return fmt.Errorf("observation %d operation root does not reconstruct", ordinal)
		}
		cursor += count
	}
	if cursor != trainingCount || cursor >= len(calls) || calls[cursor].operation != 1 {
		return fmt.Errorf("training ranges do not exactly precede guard allocation")
	}
	cursor++
	if authority.policy == "nous" {
		for ordinal := 1; ordinal <= 450; ordinal++ {
			if cursor+1 >= len(calls) || calls[cursor].operation != 3 || calls[cursor+1].operation != 2 {
				return fmt.Errorf("guard allocation schedule changed at %d", ordinal)
			}
			cursor += 2
		}
	}
	for candidateOrdinal, candidate := range byKind[103] {
		literalCount := int(candidate.value.record[98])
		for observation := 0; observation < 16; observation++ {
			for literal := 0; literal < literalCount; literal++ {
				if cursor >= len(calls) || calls[cursor].operation != 7 {
					return fmt.Errorf("candidate %d literal schedule changed", candidateOrdinal)
				}
				cursor++
			}
			if cursor >= len(calls) || calls[cursor].operation != 22 {
				return fmt.Errorf("candidate %d guard-result schedule changed", candidateOrdinal)
			}
			cursor++
		}
	}
	for candidate := range byKind[103] {
		if cursor >= len(calls) || calls[cursor].operation != 20 {
			return fmt.Errorf("candidate-result schedule changed at %d", candidate)
		}
		cursor++
	}
	if cursor != len(calls)-1 || calls[cursor].operation != 8 {
		return fmt.Errorf("artifact freeze is not the sole final acquisition call")
	}

	observationDigests := make([]string, len(byKind[105]))
	for index, observation := range byKind[105] {
		canonical, canonicalErr := observationCanonical(observation.value.record)
		if canonicalErr != nil {
			return canonicalErr
		}
		observationDigests[index] = shaHex(canonical)
	}
	semanticRoot, err := actionrelationwire.RootDigest("semantic-training", observationDigests)
	if err != nil || rawText(calls[cursor].payload[3]) != semanticRoot {
		return fmt.Errorf("semantic training root does not reconstruct")
	}
	viewRows := make([]any, len(byKind[106]))
	for index, view := range byKind[106] {
		record := view.value.record
		viewWire, _ := json.Marshal([]any{"action-view-evidence/v1", digestAt(record, 32), digestAt(record, 0), digestAt(record, 96)})
		viewRows[index] = []any{digestAt(record, 32), int(record[224]), shaHex(viewWire)}
	}
	viewRoot, err := actionrelationwire.RootDigest("view-evidence", viewRows)
	if err != nil {
		return err
	}
	for _, call := range calls {
		if call.operation == 20 && rawText(call.payload[3]) != viewRoot {
			return fmt.Errorf("candidate result names wrong view-evidence root")
		}
	}
	trainingWire, _ := json.Marshal([]any{"action-training-evidence/v1", semanticRoot, viewRoot})
	if object, ok := objects[shaHex(trainingWire)]; !ok || object.kind != 11 || !bytes.Equal(object.canonical, trainingWire) {
		return fmt.Errorf("acquisition lacks exact training evidence")
	}

	barrierDigest := rawText(calls[cursor].payload[1])
	barrierObject, ok := objects[barrierDigest]
	if !ok || barrierObject.kind != 28 {
		return fmt.Errorf("acquisition barrier is absent")
	}
	var barrier []json.RawMessage
	var candidateDigests, evaluationRoots, winnerDigests []string
	var edgeRoot, status string
	if json.Unmarshal(barrierObject.canonical, &barrier) != nil || len(barrier) != 6 || json.Unmarshal(barrier[1], &candidateDigests) != nil || json.Unmarshal(barrier[2], &edgeRoot) != nil || json.Unmarshal(barrier[3], &evaluationRoots) != nil || json.Unmarshal(barrier[4], &winnerDigests) != nil || json.Unmarshal(barrier[5], &status) != nil || status != "completed" {
		return fmt.Errorf("acquisition barrier does not decode")
	}
	wantCandidates := make([]string, len(byKind[103]))
	for index, candidate := range byKind[103] {
		wantCandidates[index] = candidate.digest
	}
	if !slices.Equal(candidateDigests, wantCandidates) {
		return fmt.Errorf("barrier candidate vector changed")
	}
	manifest := func(kind uint16) string {
		if len(byKind[kind]) == 0 {
			return zeroObjectDigest
		}
		return byKind[kind][0].value.manifest
	}
	wantEdge, wantEvaluations := manifest(104), []string{manifest(101), manifest(102)}
	if authority.policy == "no-guard" {
		wantEdge, wantEvaluations = zeroObjectDigest, []string{manifest(102)}
	}
	if edgeRoot != wantEdge || !slices.Equal(evaluationRoots, wantEvaluations) {
		return fmt.Errorf("barrier table roots changed")
	}
	var payloadWinners []string
	_ = json.Unmarshal(calls[cursor].payload[2], &payloadWinners)
	if !slices.Equal(winnerDigests, payloadWinners) {
		return fmt.Errorf("barrier winner vector differs from freeze call")
	}
	for _, winner := range winnerDigests {
		if !slices.ContainsFunc(byKind[108], func(value namedLeaf) bool { return value.digest == winner }) {
			return fmt.Errorf("barrier winner is not a candidate result")
		}
	}
	return nil
}

func lenOperation(calls []retainedCall, operation uint8) int {
	result := 0
	for _, call := range calls {
		if call.operation == operation {
			result++
		}
	}
	return result
}

func equalIntMap(left, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func digestAt(row []byte, start int) string {
	if start < 0 || start+32 > len(row) {
		return ""
	}
	return hex.EncodeToString(row[start : start+32])
}

func exactFacts(stateDigest, occurrenceDigest, factsDigest string, objects map[string]retainedObjectValue) (actionrelations.LocalFacts, error) {
	stateObject, stateOK := objects[stateDigest]
	occurrenceObject, occurrenceOK := objects[occurrenceDigest]
	factsObject, factsOK := objects[factsDigest]
	if !stateOK || stateObject.kind != 1 || !occurrenceOK || occurrenceObject.kind != 3 || !factsOK || factsObject.kind != 8 {
		return actionrelations.LocalFacts{}, fmt.Errorf("facts inputs do not resolve")
	}
	state, err := actionrelations.ParseState(stateObject.canonical)
	if err != nil {
		return actionrelations.LocalFacts{}, err
	}
	occurrence, err := actionrelations.ParseOccurrence(occurrenceObject.canonical)
	if err != nil {
		return actionrelations.LocalFacts{}, err
	}
	want, err := actionrelations.Facts(state, occurrence)
	if err != nil {
		return actionrelations.LocalFacts{}, err
	}
	wire, _ := want.CanonicalJSON()
	if !bytes.Equal(wire, factsObject.canonical) || shaHex(wire) != factsDigest {
		return actionrelations.LocalFacts{}, fmt.Errorf("local facts do not reconstruct")
	}
	return want, nil
}

func verifyLiteralCall(call retainedCall, objects map[string]retainedObjectValue) error {
	p := call.payload
	stateDigest, aFactsDigest, bFactsDigest := rawText(p[1]), rawText(p[2]), rawText(p[3])
	atom := rawText(p[4])
	var polarity bool
	if json.Unmarshal(p[5], &polarity) != nil {
		return fmt.Errorf("literal polarity does not decode")
	}
	aObject, aOK := objects[aFactsDigest]
	bObject, bOK := objects[bFactsDigest]
	if !aOK || aObject.kind != 8 || !bOK || bObject.kind != 8 {
		return fmt.Errorf("literal facts do not resolve")
	}
	aFacts, err := actionrelations.ParseLocalFacts(aObject.canonical)
	if err != nil {
		return err
	}
	bFacts, err := actionrelations.ParseLocalFacts(bObject.canonical)
	if err != nil {
		return err
	}
	if _, err := exactFacts(stateDigest, aFacts.OccurrenceDigest, aFactsDigest, objects); err != nil {
		return err
	}
	if _, err := exactFacts(stateDigest, bFacts.OccurrenceDigest, bFactsDigest, objects); err != nil {
		return err
	}
	result, err := actionrelations.EvaluateAtom(atom, aFacts, bFacts)
	if err != nil {
		return err
	}
	result = result == polarity
	wire, _ := json.Marshal([]any{"action-literal-evaluation-row/v1", stateDigest, aFactsDigest, bFactsDigest, atom, polarity, result, "valid"})
	if !slices.Equal(call.outputs, []string{shaHex(wire)}) {
		return fmt.Errorf("literal output does not reconstruct")
	}
	return nil
}

func verifyRelationMatchCall(call retainedCall, objects map[string]retainedObjectValue) error {
	p := call.payload
	relationDigest, stateDigest := rawText(p[1]), rawText(p[2])
	aFactsDigest, bFactsDigest := rawText(p[3]), rawText(p[4])
	aAppDigest, bAppDigest := rawText(p[5]), rawText(p[6])
	var literalDigests []string
	if json.Unmarshal(p[7], &literalDigests) != nil {
		return fmt.Errorf("relation literal vector does not decode")
	}
	relationObject, relationOK := objects[relationDigest]
	aFactsObject, aFactsOK := objects[aFactsDigest]
	bFactsObject, bFactsOK := objects[bFactsDigest]
	stateObject, stateOK := objects[stateDigest]
	if !relationOK || relationObject.kind != 9 || !aFactsOK || aFactsObject.kind != 8 || !bFactsOK || bFactsObject.kind != 8 || !stateOK || stateObject.kind != 1 {
		return fmt.Errorf("relation match inputs do not resolve")
	}
	relation, err := actionrelations.ParseRelation(relationObject.canonical)
	if err != nil || len(literalDigests) != len(relation.Guard.Literals) {
		return fmt.Errorf("relation guard/literal coverage changed")
	}
	aFacts, err := actionrelations.ParseLocalFacts(aFactsObject.canonical)
	if err != nil {
		return err
	}
	bFacts, err := actionrelations.ParseLocalFacts(bFactsObject.canonical)
	if err != nil {
		return err
	}
	if _, err := exactFacts(stateDigest, aFacts.OccurrenceDigest, aFactsDigest, objects); err != nil {
		return err
	}
	if _, err := exactFacts(stateDigest, bFacts.OccurrenceDigest, bFactsDigest, objects); err != nil {
		return err
	}
	aOccurrence, err := actionrelations.ParseOccurrence(objects[aFacts.OccurrenceDigest].canonical)
	if err != nil {
		return err
	}
	bOccurrence, err := actionrelations.ParseOccurrence(objects[bFacts.OccurrenceDigest].canonical)
	if err != nil {
		return err
	}
	pattern, err := actionrelations.PatternFor(aOccurrence, bOccurrence)
	if err != nil {
		return err
	}
	patternWire, _ := pattern.CanonicalJSON()
	relationPatternWire, _ := relation.Pattern.CanonicalJSON()
	patternResult := bytes.Equal(patternWire, relationPatternWire)
	state, err := actionrelations.ParseState(stateObject.canonical)
	if err != nil {
		return err
	}
	traceAdmissible := len(state.Events) <= 6
	aTransition, err := actionrelationoracle.Apply(stateObject.canonical, mustOccurrenceAction(objects[aFacts.OccurrenceDigest].canonical))
	if err != nil {
		return err
	}
	bTransition, err := actionrelationoracle.Apply(stateObject.canonical, mustOccurrenceAction(objects[bFacts.OccurrenceDigest].canonical))
	if err != nil {
		return err
	}
	aApp, _ := json.Marshal([]any{"action-applicability-row/v1", stateDigest, aFacts.OccurrenceDigest, aTransition.Applicable, "valid"})
	bApp, _ := json.Marshal([]any{"action-applicability-row/v1", stateDigest, bFacts.OccurrenceDigest, bTransition.Applicable, "valid"})
	if shaHex(aApp) != aAppDigest || shaHex(bApp) != bAppDigest {
		return fmt.Errorf("relation applicability does not reconstruct")
	}
	result := traceAdmissible && patternResult && aTransition.Applicable && bTransition.Applicable
	for index, literal := range relation.Guard.Literals {
		literalObject, ok := objects[literalDigests[index]]
		if !ok || literalObject.kind != 41 {
			return fmt.Errorf("relation literal %d does not resolve", index)
		}
		value, err := actionrelations.EvaluateAtom(literal.Atom, aFacts, bFacts)
		if err != nil {
			return err
		}
		value = value == literal.Polarity
		want, _ := json.Marshal([]any{"action-literal-evaluation-row/v1", stateDigest, aFactsDigest, bFactsDigest, literal.Atom, literal.Polarity, value, "valid"})
		if !bytes.Equal(want, literalObject.canonical) {
			return fmt.Errorf("relation literal %d does not reconstruct", index)
		}
		result = result && value
	}
	wire, _ := json.Marshal([]any{"action-relation-match-row/v1", relationDigest, stateDigest, aFactsDigest, bFactsDigest, aAppDigest, bAppDigest, traceAdmissible, patternResult, literalDigests, result, "valid"})
	if !slices.Equal(call.outputs, []string{shaHex(wire)}) {
		return fmt.Errorf("relation match output does not reconstruct")
	}
	return nil
}

func mustOccurrenceAction(canonical []byte) []byte {
	value, _ := occurrenceAction(canonical)
	return value
}

func cacheRowMatches(canonical []byte, payload []json.RawMessage) bool {
	var row []json.RawMessage
	if json.Unmarshal(canonical, &row) != nil || len(row) != 12 || len(payload) != 6 || rawText(row[0]) != "certificate-cache-row/v3" {
		return false
	}
	for index := 1; index <= 5; index++ {
		if rawText(row[index]) != rawText(payload[index]) {
			return false
		}
	}
	return rawText(row[11]) == "valid"
}

func cacheFinalizationMatches(canonical []byte, payload []json.RawMessage, objects map[string]retainedObjectValue) bool {
	var row []json.RawMessage
	if json.Unmarshal(canonical, &row) != nil || len(row) != 12 || len(payload) != 9 || rawText(row[0]) != "certificate-cache-row/v3" {
		return false
	}
	for index := 1; index <= 8; index++ {
		if rawText(row[index]) != rawText(payload[index]) {
			return false
		}
	}
	attemptObject, ok := objects[rawText(payload[7])]
	if !ok || attemptObject.kind != 44 {
		return false
	}
	attempt, err := decodeCertificateAttempt(mustObjectRow(attemptObject.canonical))
	return err == nil && attempt.State == rawText(payload[3]) && attempt.Operation == rawText(payload[8]) &&
		attempt.Result == rawText(row[9]) && attempt.Certificate == rawText(row[10]) && attempt.Status == rawText(row[11]) && attempt.Status == "valid"
}

func verifyRetainedCallTypes(authority retainedRunAuthority, call retainedCall, objects map[string]retainedObjectValue, tables map[string][]retainedTableLeaf, semanticTables retainedSemanticTableIndex) error {
	objectKind := func(digest string, kinds ...uint16) bool {
		object, ok := objects[digest]
		return ok && slices.Contains(kinds, object.kind)
	}
	tableKind := func(digest string, kinds ...uint16) bool {
		for _, leaf := range tables[digest] {
			if leaf.curriculum == authority.curriculum && (authority.phase != 1 || leaf.scope == authority.policy) && slices.Contains(kinds, leaf.kind) {
				return true
			}
		}
		return false
	}
	leafKind := func(digest string, objectKinds []uint16, tableKinds []uint16) bool {
		return objectKind(digest, objectKinds...) || tableKind(digest, tableKinds...)
	}
	p := call.payload
	d := func(index int) string {
		if index >= len(p) {
			return ""
		}
		return rawText(p[index])
	}
	ds := func(index int) []string {
		var values []string
		if index < len(p) {
			_ = json.Unmarshal(p[index], &values)
		}
		return values
	}
	requirePayload := func(index int, objectKinds []uint16, tableKinds []uint16) error {
		if !leafKind(d(index), objectKinds, tableKinds) {
			return fmt.Errorf("payload digest %d lacks named typed leaf", index)
		}
		return nil
	}
	requireList := func(index int, objectKinds []uint16, tableKinds []uint16) error {
		for _, digest := range ds(index) {
			if !leafKind(digest, objectKinds, tableKinds) {
				return fmt.Errorf("payload digest list %d lacks named typed leaf", index)
			}
		}
		return nil
	}
	requireSemantic := func(index int, kind uint16) error {
		if len(semanticTables.resolve(call.sequence, index, kind, d(index))) != 1 {
			return fmt.Errorf("payload digest %d lacks unique semantic table kind %d", index, kind)
		}
		return nil
	}
	requireSemanticList := func(index int, kind uint16) error {
		for _, digest := range ds(index) {
			if len(semanticTables.resolve(call.sequence, index, kind, digest)) != 1 {
				return fmt.Errorf("payload digest list %d lacks unique semantic table kind %d", index, kind)
			}
		}
		return nil
	}
	checks := map[uint8][]struct {
		index       int
		objectKinds []uint16
		tableKinds  []uint16
		list        bool
	}{
		1:  {{1, []uint16{6}, nil, false}},
		2:  {{1, []uint16{6}, nil, false}, {2, []uint16{7}, nil, false}},
		3:  {{1, []uint16{7}, nil, false}},
		4:  {{1, []uint16{1}, nil, false}, {2, []uint16{3}, nil, false}},
		5:  {{1, []uint16{1}, nil, false}, {2, []uint16{3}, nil, false}},
		6:  {{1, []uint16{1}, nil, false}, {2, []uint16{1}, nil, false}},
		7:  {{1, []uint16{7}, nil, false}, {3, []uint16{8}, nil, false}, {4, []uint16{8}, nil, false}},
		8:  {{1, []uint16{28}, nil, false}, {2, nil, []uint16{108}, true}},
		9:  {{1, []uint16{9}, nil, false}, {2, []uint16{1}, nil, false}, {3, []uint16{8}, nil, false}, {4, []uint16{8}, nil, false}, {5, []uint16{38}, nil, false}, {6, []uint16{38}, nil, false}, {7, []uint16{41}, nil, true}},
		10: {{1, []uint16{35}, nil, false}, {2, []uint16{10}, nil, false}},
		11: {{1, []uint16{1}, nil, false}, {2, []uint16{3}, nil, false}, {3, []uint16{38}, nil, false}},
		12: {{1, []uint16{1}, nil, false}, {2, []uint16{3}, nil, false}, {3, []uint16{38}, nil, false}},
		13: {{1, []uint16{1}, nil, false}, {2, []uint16{3}, nil, false}},
		14: {{1, []uint16{1}, nil, false}, {2, []uint16{1}, nil, false}},
		15: {{1, []uint16{1}, nil, false}, {2, []uint16{8}, nil, false}, {3, []uint16{8}, nil, false}},
		16: {{1, []uint16{1}, nil, false}, {2, []uint16{5}, nil, false}, {3, []uint16{19}, nil, false}},
		17: {{1, []uint16{20}, nil, false}, {2, []uint16{19}, nil, false}, {3, []uint16{3}, nil, false}},
		18: {{1, []uint16{4}, nil, false}, {3, []uint16{1}, nil, false}, {4, []uint16{3}, nil, false}, {5, []uint16{3}, nil, false}},
		20: {},
		21: {{1, []uint16{1}, nil, false}, {2, []uint16{3}, nil, false}},
		22: {{1, []uint16{7}, nil, false}},
		23: {{1, []uint16{4}, nil, false}, {3, []uint16{20}, nil, false}, {4, []uint16{1}, nil, false}, {5, []uint16{3}, nil, false}},
		24: {{1, []uint16{4}, nil, false}, {2, []uint16{20}, nil, false}, {3, []uint16{1}, nil, false}, {4, []uint16{3}, nil, false}, {5, []uint16{3}, nil, false}, {6, []uint16{8}, nil, false}, {7, []uint16{8}, nil, false}},
		25: {{1, []uint16{4}, nil, false}, {3, []uint16{1}, nil, false}, {4, []uint16{3}, nil, false}, {5, []uint16{3}, nil, false}, {7, []uint16{44}, nil, false}, {8, []uint16{46}, nil, false}},
	}
	if call.operation == 19 {
		if payloadTag(p) == "budget-terminal" {
			if err := requirePayload(1, []uint16{27}, nil); err != nil {
				return err
			}
		} else {
			for _, check := range []struct {
				index int
				kinds []uint16
			}{{1, []uint16{1}}, {2, []uint16{5}}} {
				if err := requirePayload(check.index, check.kinds, nil); err != nil {
					return err
				}
			}
			if err := requireList(3, []uint16{38}, nil); err != nil {
				return err
			}
		}
	} else {
		for _, check := range checks[call.operation] {
			var err error
			if check.list {
				err = requireList(check.index, check.objectKinds, check.tableKinds)
			} else {
				err = requirePayload(check.index, check.objectKinds, check.tableKinds)
			}
			if err != nil {
				return err
			}
		}
		if authority.phase == 1 {
			switch call.operation {
			case 2:
				if err := requireSemantic(3, 103); err != nil {
					return err
				}
			case 4:
				if err := requireSemantic(3, 107); err != nil {
					return err
				}
			case 7:
				if err := requireSemantic(2, 105); err != nil {
					return err
				}
			case 20:
				if err := requireSemantic(1, 103); err != nil {
					return err
				}
				if err := requireSemanticList(2, 102); err != nil {
					return err
				}
			case 22:
				if err := requireSemantic(2, 105); err != nil {
					return err
				}
				if err := requireSemanticList(3, 101); err != nil {
					return err
				}
			}
		}
	}
	outputKinds := map[uint8]struct {
		objects []uint16
		tables  []uint16
	}{
		1: {[]uint16{7}, []uint16{103}}, 2: {nil, []uint16{103}}, 3: {[]uint16{7}, []uint16{104}},
		4: {[]uint16{1}, []uint16{107}}, 5: {nil, []uint16{107}}, 6: {nil, []uint16{107}}, 7: {nil, []uint16{101}},
		8: {[]uint16{10}, nil}, 9: {[]uint16{42}, nil}, 10: {[]uint16{10}, nil}, 11: {[]uint16{39, 1}, nil},
		12: {[]uint16{39, 1}, nil}, 13: {[]uint16{38}, nil}, 14: {[]uint16{40}, nil}, 15: {[]uint16{41}, nil},
		16: {[]uint16{20}, nil}, 17: {[]uint16{18}, nil}, 18: {[]uint16{26}, nil}, 19: {[]uint16{23, 49}, nil},
		20: {nil, []uint16{108}}, 21: {[]uint16{38}, nil}, 22: {nil, []uint16{102}}, 23: {[]uint16{38}, nil},
		24: {[]uint16{48}, nil}, 25: {[]uint16{26}, nil},
	}
	want, ok := outputKinds[call.operation]
	if !ok {
		return fmt.Errorf("unknown operation output matrix")
	}
	for _, output := range call.outputs {
		if !leafKind(output, want.objects, want.tables) {
			return fmt.Errorf("output lacks operation-specific typed leaf")
		}
	}
	return nil
}

func rawText(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}

func payloadTag(payload []json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}
	return rawText(payload[0])
}

func addWork(left, right [12]int) [12]int {
	for index := range left {
		left[index] += right[index]
	}
	return left
}
