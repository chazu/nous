package transformexp

import (
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/transformbaseline"
)

func transformMeterWork(records []dsl.TransformMeterRecord) (int64, [12]int64, error) {
	var vector [12]int64
	for _, record := range records {
		if record.Category >= uint8(len(vector)) {
			return 0, vector, fmt.Errorf("invalid transformation meter category %d", record.Category)
		}
		vector[record.Category]++
	}
	work, err := workForVector(vector)
	return work, vector, err
}

func baselineEventWork(events []transformbaseline.Event) int64 {
	var vector [12]int64
	for _, event := range events {
		if event.Category >= 0 && event.Category < len(vector) {
			vector[event.Category]++
		}
	}
	work, _ := workForVector(vector)
	return work
}

func baselineEventsFromTransformMeter(records []dsl.TransformMeterRecord) []transformbaseline.Event {
	events := make([]transformbaseline.Event, len(records))
	for i, record := range records {
		events[i] = transformbaseline.Event{Category: int(record.Category), Operation: record.Operation, Phase: record.Phase, Outcome: record.Outcome, Inputs: record.Inputs, Outputs: record.Outputs}
	}
	return events
}

func transcriptFromBaselineEvents(events []transformbaseline.Event, c curriculum, policy Policy, terminal string, schema []byte) (TransformTranscriptBundle, error) {
	sink, err := newTransformTranscriptSink(c.Ordinal, string(policy), c.PolicyTokens[policy], policyManifestDigest(c, policy))
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	for index, event := range events {
		if event.Operation == "evidence-link" {
			if len(event.Inputs) != 1 || !sink.lastAttach || sink.lastOutput == "" || sink.lastObject == "" {
				return TransformTranscriptBundle{}, fmt.Errorf("baseline event %d invalid evidence boundary", index)
			}
			attemptedDigest, admitErr := sink.Admit(event.Inputs[0])
			if admitErr != nil {
				return TransformTranscriptBundle{}, admitErr
			}
			var attempted any
			if json.Unmarshal(event.Inputs[0], &attempted) != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("baseline event %d evidence JSON", index)
			}
			attemptBytes, _ := json.Marshal([]any{"transform-evidence-attempt/v1", "attached", "result", attempted, attemptedDigest, sink.lastOutput, sink.lastObject})
			attemptDigest, admitErr := sink.Admit(attemptBytes)
			if admitErr != nil {
				return TransformTranscriptBundle{}, admitErr
			}
			if emitErr := sink.Emit(TransformOperation{"evidence-link", event.Phase, []string{attemptedDigest, sink.lastOutput, sink.lastObject}, []string{attemptDigest}, "attached", 10}); emitErr != nil {
				return TransformTranscriptBundle{}, emitErr
			}
			continue
		}
		inputs := make([]string, len(event.Inputs))
		for i, value := range event.Inputs {
			inputs[i], err = sink.Admit(value)
			if err != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("baseline event %d input %d: %w", index, i, err)
			}
		}
		outputs := make([]string, len(event.Outputs))
		for i, value := range event.Outputs {
			outputs[i], err = sink.Admit(value)
			if err != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("baseline event %d output %d: %w", index, i, err)
			}
		}
		if err := sink.Emit(TransformOperation{event.Operation, event.Phase, inputs, outputs, event.Outcome, event.Category}); err != nil {
			return TransformTranscriptBundle{}, fmt.Errorf("baseline event %d: %w", index, err)
		}
	}
	input := schema
	if len(input) == 0 {
		input, _ = json.Marshal([]any{"transform-atom/v1", "enum", "no-schema"})
	}
	inputDigest, err := sink.Admit(input)
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	terminalBytes, _ := json.Marshal([]any{"transform-terminal/v1", terminal, sink.Work + 1, sink.Applications, len(sink.Events)})
	terminalDigest, err := sink.Admit(terminalBytes)
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	if err := sink.Emit(TransformOperation{"terminal", "terminal", []string{inputDigest}, []string{terminalDigest}, terminal, 11}); err != nil {
		return TransformTranscriptBundle{}, err
	}
	return sink.Bundle()
}

func transcriptFromAcquisition(run acquisitionRun, ordinal int, policy Policy, token, manifestDigest string) (TransformTranscriptBundle, error) {
	sink, err := newTransformTranscriptSink(ordinal, string(policy), token, manifestDigest)
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	for index, record := range run.MeterRecords {
		if record.Operation == "evidence-link" {
			if len(record.Inputs) != 1 || !sink.lastAttach || sink.lastOutput == "" || sink.lastObject == "" {
				return TransformTranscriptBundle{}, fmt.Errorf("meter %d invalid evidence boundary", index)
			}
			attemptedDigest, admitErr := sink.Admit(record.Inputs[0])
			if admitErr != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("meter %d evidence value: %w", index, admitErr)
			}
			var attempted any
			if json.Unmarshal(record.Inputs[0], &attempted) != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("meter %d evidence value is not JSON", index)
			}
			attemptBytes, _ := json.Marshal([]any{"transform-evidence-attempt/v1", "attached", "result", attempted, attemptedDigest, sink.lastOutput, sink.lastObject})
			attemptDigest, admitErr := sink.Admit(attemptBytes)
			if admitErr != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("meter %d evidence attempt: %w", index, admitErr)
			}
			operation := TransformOperation{"evidence-link", record.Phase, []string{attemptedDigest, sink.lastOutput, sink.lastObject}, []string{attemptDigest}, "attached", 10}
			if emitErr := sink.Emit(operation); emitErr != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("meter %d: %w", index, emitErr)
			}
			continue
		}
		inputs := make([]string, len(record.Inputs))
		for i, value := range record.Inputs {
			inputs[i], err = sink.Admit(value)
			if err != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("meter %d input %d: %w", index, i, err)
			}
		}
		outputs := make([]string, len(record.Outputs))
		for i, value := range record.Outputs {
			outputs[i], err = sink.Admit(value)
			if err != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("meter %d output %d: %w", index, i, err)
			}
		}
		operation := TransformOperation{record.Operation, record.Phase, inputs, outputs, record.Outcome, int(record.Category)}
		if err := sink.Emit(operation); err != nil {
			return TransformTranscriptBundle{}, fmt.Errorf("meter %d: %w", index, err)
		}
	}
	terminal := run.Terminal
	if terminal == "" {
		terminal = "no-discovery"
	}
	inputBytes := []byte(run.Store.Get(run.Root).GetString("partial"))
	if run.Artifact != "" {
		inputBytes = []byte(run.Store.Get(run.Artifact).GetString("schema"))
	}
	inputDigest, err := sink.Admit(inputBytes)
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	terminalBytes, _ := json.Marshal([]any{"transform-terminal/v1", terminal, sink.Work + 1, sink.Applications, len(sink.Events)})
	terminalDigest, err := sink.Admit(terminalBytes)
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	if err := sink.Emit(TransformOperation{"terminal", "terminal", []string{inputDigest}, []string{terminalDigest}, terminal, 11}); err != nil {
		return TransformTranscriptBundle{}, err
	}
	return sink.Bundle()
}
