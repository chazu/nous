package nogoodexp

import (
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/nogoodbaseline"
)

func baselineTranscript(taskOrdinal uint32, result nogoodbaseline.Result) ([]TranscriptEvent, error) {
	events := make([]TranscriptEvent, 0, len(result.Events))
	for _, source := range result.Events {
		if source.Category < 1 || source.Category > 12 {
			return nil, fmt.Errorf("invalid baseline category %d", source.Category)
		}
		operation := source.Transition
		outcome := "ok"
		operand := func(index int) int {
			if index < len(source.Operands) {
				return source.Operands[index]
			}
			return 0
		}
		variable, color := fmt.Sprintf("v:%d", operand(0)), fmt.Sprintf("c:%d", operand(1))
		event := TranscriptEvent{Category: uint8(source.Category), TaskOrdinal: taskOrdinal}
		switch source.Category {
		case 1:
			event.Code = 1
			event.Operands = [8]TranscriptOperand{ID("candidate"), OptionalID(""), Number(int32(operand(0))), Number(int32(operand(1))), ID(operation), ID(outcome), Omitted(), Omitted()}
		case 2:
			event.Code = 2
			event.Operands = [8]TranscriptOperand{ID(variable), ID(operation), ID(color), ID(outcome), Omitted(), Omitted(), Omitted(), Omitted()}
		case 3:
			event.Code = 3
			event.Operands = [8]TranscriptOperand{ID(variable), ID(color), Number(0), ID(operation), ID(outcome), Omitted(), Omitted(), Omitted()}
		case 4:
			event.Code = 4
			event.Operands = [8]TranscriptOperand{ID(variable), ID(color), ID(fmt.Sprintf("v:%d", operand(2))), ID(fmt.Sprintf("c:%d", operand(3))), ID(outcome), Omitted(), Omitted(), Omitted()}
		case 5:
			event.Code = 5
			event.Operands = [8]TranscriptOperand{ID(variable), ID(color), ID(operation), OptionalID(""), ID(outcome), Omitted(), Omitted(), Omitted()}
		case 6:
			event.Code = 6
			event.Operands = [8]TranscriptOperand{ID(variable), ID(fmt.Sprintf("v:%d", operand(1))), OptionalID(""), Number(0), ID(operation), ID(outcome), Omitted(), Omitted()}
		case 7:
			event.Code = 7
			event.Operands = [8]TranscriptOperand{ID("conflict:" + operation), OptionalID(variable), OptionalID(""), Number(0), ID(operation), ID(outcome), Omitted(), Omitted()}
		case 8:
			event.Code = 8
			event.Operands = [8]TranscriptOperand{OptionalID("artifact"), OptionalID("binding"), Number(int32(operand(0))), OptionalID(""), OptionalID(""), ID(operation), ID(outcome), Omitted()}
		case 9:
			event.Code = 9
			event.Operands = [8]TranscriptOperand{ID("binding"), ID("completion"), OptionalID(""), OptionalID(""), OptionalID(""), ID(operation), ID(outcome), Omitted()}
		case 10:
			event.Code = 10
			event.Operands = [8]TranscriptOperand{ID("certificate"), OptionalID(""), ID(operation), OptionalID(""), OptionalID(""), ID(operation), ID(outcome), Omitted()}
		case 11:
			event.Code = 11
			event.Operands = [8]TranscriptOperand{ID(operation), Number(1), Omitted(), Omitted(), Omitted(), Omitted(), Omitted(), Omitted()}
		case 12:
			event.Code = 18
			event.Operands = [8]TranscriptOperand{ID("record:" + operation), OptionalID(""), Number(0), ID(operation), ID(outcome), Omitted(), Omitted(), Omitted()}
		}
		events = append(events, event)
	}
	return events, nil
}

func fixedVectorEvents(taskOrdinal uint32, vector [12]int64, prefix string) []TranscriptEvent {
	var events []TranscriptEvent
	for category, count := range vector {
		for index := int64(0); index < count; index++ {
			operation := fmt.Sprintf("%s:c%d", prefix, category+1)
			source := nogoodbaseline.Event{Category: category + 1, Transition: operation, Operands: []int{int(index), category}}
			mapped, _ := baselineTranscript(taskOrdinal, nogoodbaseline.Result{Events: []nogoodbaseline.Event{source}})
			events = append(events, mapped[0])
		}
	}
	return events
}

func acquisitionTranscript(run TrainingRun, preflight []TranscriptEvent) ([]TranscriptEvent, error) {
	events := slices.Clone(preflight)
	for index, record := range run.MeterRecords {
		mapped, err := baselineTranscript(0x80000000, nogoodbaseline.Result{Events: []nogoodbaseline.Event{{Category: int(record.Category), Transition: record.Operation, Operands: []int{index}}}})
		if err != nil {
			return nil, err
		}
		events = append(events, mapped[0])
	}
	return events, nil
}

func transcriptVector(events []TranscriptEvent) (vector [12]int64) {
	for _, event := range events {
		vector[event.Category-1]++
	}
	return vector
}

func transcriptWork(events []TranscriptEvent) int64 { return int64(len(events)) }

func appendEvents(first, second []TranscriptEvent) []TranscriptEvent {
	return append(slices.Clone(first), second...)
}

func bridgeTranscript(taskOrdinal uint32, disposition Disposition) ([]TranscriptEvent, error) {
	if len(disposition.MeterRecords) == 0 {
		return nil, fmt.Errorf("bridge emitted no verifier-owned meter records")
	}
	events := make([]TranscriptEvent, 0, len(disposition.MeterRecords))
	for index, record := range disposition.MeterRecords {
		if record.Category < 1 || record.Category > 12 || record.Operation == "" {
			return nil, fmt.Errorf("invalid bridge meter record %d", index)
		}
		mapped, err := baselineTranscript(taskOrdinal, nogoodbaseline.Result{Events: []nogoodbaseline.Event{{Category: int(record.Category), Transition: record.Operation, Operands: []int{index}}}})
		if err != nil {
			return nil, err
		}
		events = append(events, mapped[0])
	}
	return events, nil
}
