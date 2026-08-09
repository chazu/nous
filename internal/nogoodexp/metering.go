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

func acquisitionTranscript(run TrainingRun, preflight []TranscriptEvent) []TranscriptEvent {
	events := slices.Clone(preflight)
	store := run.Store
	appendUnits := func(category string, task uint32, vector [12]int64) {
		count := len(store.Examples(category)) - 1
		for index := 0; index < count; index++ {
			events = append(events, fixedVectorEvents(task, vector, category)...)
		}
	}
	appendUnits("NogoodCandidate", 0x80000004, [12]int64{0: 1, 11: 1})
	appendUnits("NogoodRefinement", 0x80000004, [12]int64{0: 1, 11: 1})
	appendUnits("NogoodBinding", 0x80000004, [12]int64{0: 1, 1: 1, 11: 1})
	appendUnits("NogoodResult", 0x80000004, [12]int64{8: 1, 11: 1})
	appendUnits("NogoodEvidence", 0x80000004, [12]int64{11: 2})
	appendUnits("NogoodEvidenceBarrier", 0x80000004, [12]int64{9: 2, 11: 1})
	appendUnits("NogoodPromotionProof", 0x90000000, [12]int64{8: 1, 11: 1})
	events = append(events, fixedVectorEvents(0x90000018, [12]int64{7: 1, 11: 3}, "artifact-freeze")...)
	return events
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

func bridgeTranscript(taskOrdinal uint32, disposition Disposition) []TranscriptEvent {
	if disposition.Status == "propose-prune" {
		return fixedVectorEvents(taskOrdinal, [12]int64{3, 23, 3, 3, 3, 0, 0, 10, 1, 25, 1, 53}, "learned-prune")
	}
	roles := int64(len(disposition.Store.Examples("NogoodRoleCandidate")) - 1)
	pairs := int64(len(disposition.Store.Examples("NogoodPairProposal")) - 1)
	bindings := int64(len(disposition.Store.Examples("NogoodBinding")) - 1)
	artifactChecks := int64(0)
	artifactRecords := int64(0)
	if len(disposition.Store.Examples("NogoodArtifact")) > 1 && bindings > 0 {
		artifactChecks = 10
		artifactRecords = 2
	}
	storeWork := int64(26) + 7 + roles + bindings + artifactRecords + 10
	return fixedVectorEvents(taskOrdinal, [12]int64{roles + pairs, 17, 0, 0, 0, 0, 0, artifactChecks, 0, 0, 0, storeWork}, "bridge-resume")
}
