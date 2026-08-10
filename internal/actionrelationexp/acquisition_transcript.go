package actionrelationexp

import (
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/actionrelationacquire"
	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

type AcquisitionTranscript struct {
	RunID            string
	Reservations     []WorkReservation
	Transcript       TranscriptBundle
	ObservationRoots []OperationRoot
	RunRoot          OperationRoot
}

func AcquisitionRunID(panel, authority string, curriculum int, scope string) (string, error) {
	return actionrelationledger.AcquisitionRunID(panel, authority, curriculum, scope)
}

func BuildAcquisitionTranscript(run actionrelationacquire.Run, tables map[uint16]TableBundle, runID string) (AcquisitionTranscript, error) {
	if run.Store == nil || run.Experiment == "" || !runIDText(runID) {
		return AcquisitionTranscript{}, fmt.Errorf("invalid acquisition transcript input")
	}
	experiment := run.Store.Get(run.Experiment)
	if experiment == nil {
		return AcquisitionTranscript{}, fmt.Errorf("missing acquisition experiment")
	}
	tableKinds := []uint16{101, 102, 103, 104, 107, 108}
	if experiment.GetString("scope") == "no-guard" {
		tableKinds = []uint16{102, 103, 107, 108}
	}
	for _, kind := range tableKinds {
		if bundle, ok := tables[kind]; !ok || VerifyTableBundle(bundle) != nil {
			return AcquisitionTranscript{}, fmt.Errorf("missing acquisition transcript table %d", kind)
		}
	}
	translator := acquisitionCallTranslator{run: run, tables: tables, ordinals: map[uint16]int{}}
	calls := make([]ChargedCall, len(run.MeterRecords))
	reservations := make([]WorkReservation, len(run.MeterRecords))
	reservationNames := experiment.GetStrings("reservationUnits")
	if experiment.GetString("runID") != runID || len(reservationNames) != len(run.MeterRecords) {
		return AcquisitionTranscript{}, fmt.Errorf("acquisition reservations do not cover charged calls")
	}
	for sequence, meter := range run.MeterRecords {
		code := uint8(meter.Code)
		if meter.Status != 1 {
			return AcquisitionTranscript{}, fmt.Errorf("acquisition call %d has invalid status", sequence)
		}
		name := fmt.Sprintf("AR.Reservation.%s.%05d", runID, sequence)
		if reservationNames[sequence] != name {
			return AcquisitionTranscript{}, fmt.Errorf("acquisition reservation %d has noncanonical name", sequence)
		}
		u := run.Store.Get(name)
		if u == nil || u.GetString("objectDigest") != meter.SourceTaskDigest {
			return AcquisitionTranscript{}, fmt.Errorf("acquisition call %d lacks reserved authority", sequence)
		}
		reservation, err := actionrelationledger.ParseReservation([]byte(u.GetString("canonicalObject")))
		if err != nil || VerifyWorkReservation(reservation, acquisitionLifecycleCap) != nil ||
			reservation.Digest != meter.SourceTaskDigest || reservation.RunID != runID ||
			reservation.TaskDigest != actionrelationledger.TaskDigest(runID, sequence, code) ||
			len(reservation.OperationCodes) != 1 || reservation.OperationCodes[0] != code ||
			reservation.TotalBefore != sequence || reservation.TotalAfter != sequence+1 || reservation.Status != "reserved" {
			return AcquisitionTranscript{}, fmt.Errorf("invalid acquisition reservation %d", sequence)
		}
		call, err := translator.translate(meter, meter.SourceTaskDigest)
		if err != nil {
			return AcquisitionTranscript{}, fmt.Errorf("translate acquisition call %d: %w", sequence, err)
		}
		calls[sequence], reservations[sequence] = call, reservation
	}
	for _, kind := range tableKinds {
		if translator.ordinals[kind] != len(tables[kind].LeafDigests) {
			return AcquisitionTranscript{}, fmt.Errorf("table %d charged-output coverage %d want %d", kind, translator.ordinals[kind], len(tables[kind].LeafDigests))
		}
	}
	transcript, err := BuildTranscript(runID, calls)
	if err != nil {
		return AcquisitionTranscript{}, err
	}
	observationRoots, err := bindObservationOperationRoots(run, transcript)
	if err != nil {
		return AcquisitionTranscript{}, err
	}
	runRoot, err := BuildOperationRange(runID, 1, 0, transcript.CallIDs)
	if err != nil {
		return AcquisitionTranscript{}, err
	}
	result := AcquisitionTranscript{RunID: runID, Reservations: reservations, Transcript: transcript, ObservationRoots: observationRoots, RunRoot: runRoot}
	if err := VerifyAcquisitionTranscript(result, run); err != nil {
		return AcquisitionTranscript{}, err
	}
	return result, nil
}

type acquisitionCallTranslator struct {
	run      actionrelationacquire.Run
	tables   map[uint16]TableBundle
	ordinals map[uint16]int
}

func (t *acquisitionCallTranslator) nextLeaf(kind uint16) (string, error) {
	ordinal := t.ordinals[kind]
	bundle := t.tables[kind]
	if ordinal >= len(bundle.LeafDigests) {
		return "", fmt.Errorf("table %d leaf overflow", kind)
	}
	t.ordinals[kind] = ordinal + 1
	return bundle.LeafDigests[ordinal], nil
}

func (t *acquisitionCallTranslator) translate(meter dsl.ActionRelationMeterRecord, source string) (ChargedCall, error) {
	if meter.Code < 1 || meter.Code > 25 || int(meter.Counter) != int(operationCounters[uint8(meter.Code)]) {
		return ChargedCall{}, fmt.Errorf("invalid meter code/counter")
	}
	call := ChargedCall{Phase: 1, Operation: uint8(meter.Code), Status: 1, SourceTaskDigest: source}
	digest := func(data []byte) string { return shaHex(data) }
	row := func(data []byte) ([]json.RawMessage, error) {
		var result []json.RawMessage
		if json.Unmarshal(data, &result) != nil {
			return nil, fmt.Errorf("invalid canonical meter row")
		}
		canonical, _ := json.Marshal(result)
		if string(canonical) != string(data) {
			return nil, fmt.Errorf("noncanonical meter row")
		}
		return result, nil
	}
	str := func(raw json.RawMessage) string { var value string; _ = json.Unmarshal(raw, &value); return value }
	integer := func(raw json.RawMessage) int { var value int; _ = json.Unmarshal(raw, &value); return value }
	boolean := func(raw json.RawMessage) bool { var value bool; _ = json.Unmarshal(raw, &value); return value }
	leafOutput := func(kind uint16) error {
		leaf, err := t.nextLeaf(kind)
		if err == nil {
			call.OutputDigests = []string{leaf}
		}
		return err
	}

	switch meter.Code {
	case 1:
		if len(meter.Inputs) != 1 || len(meter.Outputs) != 2 {
			return call, fmt.Errorf("invalid guard-root meter shape")
		}
		call.Payload = []any{"guard-root", digest(meter.Inputs[0])}
		leaf, err := t.nextLeaf(103)
		if err != nil {
			return call, err
		}
		call.OutputDigests = []string{digest(meter.Outputs[0]), leaf}
	case 2:
		candidate, err := row(meter.Outputs[0])
		if err != nil || len(candidate) != 6 {
			return call, fmt.Errorf("invalid candidate allocation output")
		}
		call.Payload = []any{"candidate-allocate", digest(meter.Inputs[0]), digest(meter.Inputs[1]), str(candidate[2]), integer(candidate[4])}
		if err := leafOutput(103); err != nil {
			return call, err
		}
	case 3:
		edge, err := row(meter.Outputs[1])
		if err != nil || len(edge) != 6 {
			return call, fmt.Errorf("invalid guard extension output")
		}
		call.Payload = []any{"guard-extend", digest(meter.Inputs[0]), str(edge[3]), boolean(edge[4]), integer(edge[5])}
		leaf, err := t.nextLeaf(104)
		if err != nil {
			return call, err
		}
		call.OutputDigests = []string{digest(meter.Outputs[0]), leaf}
	case 4:
		transition, err := row(meter.Outputs[0])
		if err != nil || len(transition) != 6 {
			return call, fmt.Errorf("invalid training apply output")
		}
		call.Payload = []any{"training-apply", digest(meter.Inputs[0]), digest(meter.Inputs[1]), str(transition[3])}
		leaf, err := t.nextLeaf(107)
		if err != nil {
			return call, err
		}
		call.OutputDigests = []string{leaf}
		if len(meter.Outputs) == 2 {
			call.OutputDigests = append(call.OutputDigests, digest(meter.Outputs[1]))
		}
	case 5:
		call.Payload = []any{"training-applicable", digest(meter.Inputs[0]), digest(meter.Inputs[1])}
		if err := leafOutput(107); err != nil {
			return call, err
		}
	case 6:
		call.Payload = []any{"training-equality", digest(meter.Inputs[0]), digest(meter.Inputs[1])}
		if err := leafOutput(107); err != nil {
			return call, err
		}
	case 7:
		literal, err := row(meter.Outputs[0])
		if err != nil || len(literal) != 8 {
			return call, fmt.Errorf("invalid training literal output")
		}
		call.Payload = []any{"training-literal", digest(meter.Inputs[0]), str(literal[2]), str(literal[3]), str(literal[4]), str(literal[5]), boolean(literal[6])}
		if err := leafOutput(101); err != nil {
			return call, err
		}
	case 8:
		experiment := t.run.Store.Get(t.run.Experiment)
		barrier := t.run.Store.Get(experiment.GetString("guardSearchBarrier"))
		if barrier == nil {
			return call, fmt.Errorf("missing freeze barrier")
		}
		winners := make([]string, len(barrier.GetStrings("winnerResults")))
		for index, name := range barrier.GetStrings("winnerResults") {
			winners[index] = t.run.Store.Get(name).GetString("objectDigest")
		}
		call.Payload = []any{"artifact-freeze", barrier.GetString("objectDigest"), winners, experiment.GetString("semanticTrainingRoot")}
		call.OutputDigests = []string{digest(meter.Outputs[0])}
	case 20:
		result, err := row(meter.Outputs[0])
		if err != nil || len(result) != 8 {
			return call, fmt.Errorf("invalid candidate result output")
		}
		var guardResults []string
		if json.Unmarshal(result[2], &guardResults) != nil {
			return call, fmt.Errorf("invalid guard result vector")
		}
		call.Payload = []any{"candidate-result", digest(meter.Inputs[0]), guardResults, str(result[7])}
		if err := leafOutput(108); err != nil {
			return call, err
		}
	case 22:
		result, err := row(meter.Outputs[0])
		if err != nil || len(result) != 5 {
			return call, fmt.Errorf("invalid guard result output")
		}
		var literalDigests []string
		if json.Unmarshal(result[3], &literalDigests) != nil {
			return call, fmt.Errorf("invalid literal vector")
		}
		call.Payload = []any{"guard-result", digest(meter.Inputs[0]), str(result[2]), literalDigests}
		if err := leafOutput(102); err != nil {
			return call, err
		}
	default:
		return call, fmt.Errorf("unexpected acquisition operation %d", meter.Code)
	}
	return call, nil
}

func bindObservationOperationRoots(run actionrelationacquire.Run, transcript TranscriptBundle) ([]OperationRoot, error) {
	experiment := run.Store.Get(run.Experiment)
	if experiment == nil {
		return nil, fmt.Errorf("missing observation experiment")
	}
	roots := make([]OperationRoot, len(experiment.GetStrings("observationUnits")))
	cursor := uint32(0)
	for ordinal, name := range experiment.GetStrings("observationUnits") {
		observation := run.Store.Get(name)
		if observation == nil {
			return nil, fmt.Errorf("missing observation %d", ordinal)
		}
		aInitial, bInitial := run.Store.Get(observation.GetString("aInitialRow")), run.Store.Get(observation.GetString("bInitialRow"))
		if aInitial == nil || bInitial == nil {
			return nil, fmt.Errorf("missing initial observation transitions")
		}
		expected := []*unit.Unit{run.Store.Get(aInitial.GetString("applicabilityRow")), run.Store.Get(bInitial.GetString("applicabilityRow")), aInitial, bInitial}
		for _, transitionName := range []string{observation.GetString("bAfterARow"), observation.GetString("aAfterBRow")} {
			if transitionName != "" {
				transition := run.Store.Get(transitionName)
				if transition == nil {
					return nil, fmt.Errorf("missing cross observation transition")
				}
				expected = append(expected, run.Store.Get(transition.GetString("applicabilityRow")), transition)
			}
		}
		if equalityName := observation.GetString("equalityRow"); equalityName != "" {
			equality := run.Store.Get(equalityName)
			expected = append(expected, equality)
		}
		start := cursor
		for _, unit := range expected {
			if unit == nil || int(cursor) >= len(run.MeterRecords) || len(run.MeterRecords[cursor].Outputs) == 0 || shaHex(run.MeterRecords[cursor].Outputs[0]) != unit.GetString("objectDigest") {
				return nil, fmt.Errorf("observation %d call %d does not match ordered operation row", ordinal, cursor)
			}
			cursor++
		}
		root, err := BuildOperationRange(transcript.RunID, 1, start, transcript.CallIDs[start:cursor])
		if err != nil {
			return nil, err
		}
		observation.Set("operationRoot", root.Digest)
		roots[ordinal] = root
	}
	if int(cursor) >= len(run.MeterRecords) || run.MeterRecords[cursor].Code != 1 {
		return nil, fmt.Errorf("observation ranges do not end at allocation phase")
	}
	return roots, nil
}

func VerifyAcquisitionTranscript(value AcquisitionTranscript, run actionrelationacquire.Run) error {
	if value.RunID != value.Transcript.RunID || len(value.Reservations) != len(value.Transcript.CallIDs) || len(value.Reservations) != len(run.MeterRecords) || VerifyTranscript(value.Transcript) != nil || VerifyOperationRange(value.RunRoot, value.Transcript) != nil {
		return fmt.Errorf("invalid acquisition transcript authority")
	}
	wantRunRoot, err := BuildOperationRange(value.RunID, 1, 0, value.Transcript.CallIDs)
	if err != nil || wantRunRoot.Digest != value.RunRoot.Digest || string(wantRunRoot.Canonical) != string(value.RunRoot.Canonical) {
		return fmt.Errorf("incomplete acquisition run operation root")
	}
	sequence := 0
	for shardOrdinal, file := range value.Transcript.InputFiles {
		rows, err := parseInputFrames(file.Data, value.Transcript.InputRoot.Shards[shardOrdinal].RecordCount)
		if err != nil {
			return err
		}
		for _, envelope := range rows {
			var fields []json.RawMessage
			var source string
			if json.Unmarshal(envelope, &fields) != nil || len(fields) != 5 || json.Unmarshal(fields[3], &source) != nil || source != value.Reservations[sequence].Digest {
				return fmt.Errorf("call %d does not resolve its reservation", sequence)
			}
			sequence++
		}
	}
	if sequence != len(value.Reservations) {
		return fmt.Errorf("reservation/envelope coverage mismatch")
	}
	for sequence, reservation := range value.Reservations {
		if VerifyWorkReservation(reservation, acquisitionLifecycleCap) != nil || reservation.Status != "reserved" || reservation.TotalBefore != sequence || reservation.TotalAfter != sequence+1 || len(reservation.OperationCodes) != 1 || reservation.OperationCodes[0] != uint8(run.MeterRecords[sequence].Code) {
			return fmt.Errorf("invalid acquisition reservation %d", sequence)
		}
	}
	experiment := run.Store.Get(run.Experiment)
	if experiment == nil || len(value.ObservationRoots) != len(experiment.GetStrings("observationUnits")) {
		return fmt.Errorf("observation operation-root cardinality mismatch")
	}
	for ordinal, root := range value.ObservationRoots {
		if VerifyOperationRange(root, value.Transcript) != nil {
			return fmt.Errorf("invalid observation operation root")
		}
		observation := run.Store.Get(experiment.GetStrings("observationUnits")[ordinal])
		if observation == nil || observation.GetString("operationRoot") != root.Digest {
			return fmt.Errorf("observation does not name exact operation root")
		}
	}
	return nil
}
