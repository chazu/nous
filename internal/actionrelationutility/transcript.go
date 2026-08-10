package actionrelationutility

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

func BuildTranscript(store *unit.Store, runID string, records []dsl.ActionRelationMeterRecord) (actionrelationexp.TranscriptBundle, error) {
	if store == nil {
		return actionrelationexp.TranscriptBundle{}, fmt.Errorf("missing utility Store")
	}
	if err := verifyUtilityReservationSources(store, records); err != nil {
		return actionrelationexp.TranscriptBundle{}, err
	}
	calls := make([]actionrelationexp.ChargedCall, len(records))
	for index, record := range records {
		call, err := translateUtilityRecord(record)
		if err != nil {
			return actionrelationexp.TranscriptBundle{}, fmt.Errorf("translate utility call %d: %w", index, err)
		}
		calls[index] = call
	}
	return actionrelationexp.BuildTranscript(runID, calls)
}

func (s *Session) OperationRoot(firstSequence int) (actionrelationexp.OperationRoot, error) {
	records, err := s.Snapshot()
	if err != nil || firstSequence < 0 || firstSequence >= len(records) {
		return actionrelationexp.OperationRoot{}, fmt.Errorf("invalid utility operation range")
	}
	transcript, err := BuildTranscript(s.Store, s.RunID, records)
	if err != nil {
		return actionrelationexp.OperationRoot{}, err
	}
	return actionrelationexp.BuildOperationRange(s.RunID, 2, uint32(firstSequence), transcript.CallIDs[firstSequence:])
}

func verifyUtilityReservationSources(store *unit.Store, records []dsl.ActionRelationMeterRecord) error {
	type sourceState struct {
		reservation actionrelationledger.Reservation
		next        int
	}
	sources := map[string]*sourceState{}
	for sequence, record := range records {
		state := sources[record.SourceTaskDigest]
		if state == nil {
			var found *unit.Unit
			for _, name := range store.All() {
				u := store.Get(name)
				if u.GetString("objectDigest") == record.SourceTaskDigest && store.IsA(u.Name, "CompoundWorkReservation") {
					found = u
					break
				}
			}
			if found == nil {
				return fmt.Errorf("utility call %d lacks retained reservation", sequence)
			}
			reservation, err := actionrelationledger.ParseReservation([]byte(found.GetString("canonicalObject")))
			if err != nil || actionrelationledger.VerifyReservation(reservation, 2_000_000) != nil || reservation.Status != "reserved" || reservation.Digest != record.SourceTaskDigest {
				return fmt.Errorf("utility call %d has invalid reservation", sequence)
			}
			state = &sourceState{reservation: reservation}
			sources[record.SourceTaskDigest] = state
		}
		if state.next >= len(state.reservation.OperationCodes) || state.reservation.OperationCodes[state.next] != uint8(record.Code) {
			return fmt.Errorf("utility call %d violates its compound reservation", sequence)
		}
		state.next++
	}
	for digest, state := range sources {
		if state.next != len(state.reservation.OperationCodes) {
			return fmt.Errorf("utility reservation %s is incomplete", digest)
		}
	}
	return nil
}

func translateUtilityRecord(record dsl.ActionRelationMeterRecord) (actionrelationexp.ChargedCall, error) {
	code := uint8(record.Code)
	if code < 9 || code > 25 || record.SourceTaskDigest == "" {
		return actionrelationexp.ChargedCall{}, fmt.Errorf("invalid utility meter record")
	}
	call := actionrelationexp.ChargedCall{Phase: 2, Operation: code, Status: 1, SourceTaskDigest: record.SourceTaskDigest}
	digest := func(data []byte) string {
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:])
	}
	canonicalRow := func(data []byte, length int) ([]json.RawMessage, error) {
		var row []json.RawMessage
		if json.Unmarshal(data, &row) != nil || len(row) != length {
			return nil, fmt.Errorf("invalid utility evidence row")
		}
		canonical, _ := json.Marshal(row)
		if !bytes.Equal(canonical, data) {
			return nil, fmt.Errorf("noncanonical utility evidence row")
		}
		return row, nil
	}
	stringAt := func(row []json.RawMessage, index int) string {
		var value string
		_ = json.Unmarshal(row[index], &value)
		return value
	}
	output := func(index int) (string, error) {
		if index >= len(record.Outputs) {
			return "", fmt.Errorf("missing utility output")
		}
		return digest(record.Outputs[index]), nil
	}
	switch code {
	case 12:
		row, err := canonicalRow(firstOutput(record), 6)
		if err != nil || stringAt(row, 0) != "action-transition-row/v1" {
			return call, fmt.Errorf("invalid certificate transition")
		}
		call.Payload = []any{"certificate-apply", stringAt(row, 1), stringAt(row, 2), stringAt(row, 3)}
		for index := range record.Outputs {
			value, _ := output(index)
			call.OutputDigests = append(call.OutputDigests, value)
		}
	case 13:
		row, err := canonicalRow(firstOutput(record), 5)
		if err != nil || stringAt(row, 0) != "action-applicability-row/v1" {
			return call, fmt.Errorf("invalid certificate applicability")
		}
		call.Payload = []any{"certificate-applicable", stringAt(row, 1), stringAt(row, 2)}
		value, _ := output(0)
		call.OutputDigests = []string{value}
	case 14:
		row, err := canonicalRow(firstOutput(record), 5)
		if err != nil || stringAt(row, 0) != "action-state-equality-row/v1" {
			return call, fmt.Errorf("invalid certificate equality")
		}
		call.Payload = []any{"certificate-equality", stringAt(row, 1), stringAt(row, 2)}
		value, _ := output(0)
		call.OutputDigests = []string{value}
	case 23:
		if len(record.Inputs) != 5 {
			return call, fmt.Errorf("invalid search applicability context")
		}
		row, err := canonicalRow(firstOutput(record), 5)
		if err != nil || stringAt(row, 0) != "action-applicability-row/v1" {
			return call, fmt.Errorf("invalid search applicability")
		}
		call.Payload = []any{"search-applicable", string(record.Inputs[0]), string(record.Inputs[1]), digest(record.Inputs[2]), stringAt(row, 1), stringAt(row, 2)}
		value, _ := output(0)
		call.OutputDigests = []string{value}
	case 24:
		row, err := canonicalRow(firstOutput(record), 10)
		if err != nil || stringAt(row, 0) != "action-static-footprint-row/v1" {
			return call, fmt.Errorf("invalid static footprint")
		}
		call.Payload = []any{"static-footprint", stringAt(row, 1), stringAt(row, 2), stringAt(row, 3), stringAt(row, 4), stringAt(row, 5), stringAt(row, 6), stringAt(row, 7)}
		value, _ := output(0)
		call.OutputDigests = []string{value}
	default:
		return call, fmt.Errorf("utility operation %d translator not implemented", code)
	}
	return call, nil
}

func firstOutput(record dsl.ActionRelationMeterRecord) []byte {
	if len(record.Outputs) == 0 {
		return nil
	}
	return record.Outputs[0]
}
