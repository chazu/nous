package transformexp

import (
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/transformbaseline"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
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

func transcriptFromBaselineEvents(events []transformbaseline.Event, c policyCurriculum, ordinal int, policy Policy, terminal string, schema []byte) (TransformTranscriptBundle, error) {
	sink, err := newTransformTranscriptSink(ordinal, string(policy), c.PolicyTokens[policy], policyManifestDigest(c, policy))
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	for index, event := range events {
		if event.Operation == "evidence-link" {
			if len(event.Inputs) != 1 {
				return TransformTranscriptBundle{}, fmt.Errorf("baseline event %d invalid evidence boundary", index)
			}
			if emitErr := sink.EmitEvidenceLink(event.Phase, event.Inputs[0]); emitErr != nil {
				return TransformTranscriptBundle{}, emitErr
			}
			continue
		}
		if err := sink.EmitValues(event.Operation, event.Phase, event.Outcome, event.Category, event.Inputs, event.Outputs); err != nil {
			return TransformTranscriptBundle{}, fmt.Errorf("baseline event %d: %w", index, err)
		}
	}
	input := schema
	if _, parseErr := transformschema.ParseSchema(input); parseErr != nil {
		input, _ = json.Marshal([]any{"transform-store-boundary/v1", "freeze", digestBytes(schema)})
	}
	terminalBytes, _ := json.Marshal([]any{"transform-terminal/v1", terminal, sink.Work + 1, sink.Applications, len(sink.Events)})
	if err := sink.EmitValues("terminal", "terminal", terminal, 11, [][]byte{input}, [][]byte{terminalBytes}); err != nil {
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
			if len(record.Inputs) != 1 {
				return TransformTranscriptBundle{}, fmt.Errorf("meter %d invalid evidence boundary", index)
			}
			if emitErr := sink.EmitEvidenceLink(record.Phase, record.Inputs[0]); emitErr != nil {
				return TransformTranscriptBundle{}, fmt.Errorf("meter %d: %w", index, emitErr)
			}
			continue
		}
		if err := sink.EmitValues(record.Operation, record.Phase, record.Outcome, int(record.Category), record.Inputs, record.Outputs); err != nil {
			return TransformTranscriptBundle{}, fmt.Errorf("meter %d %s/%s/%s: %w", index, record.Phase, record.Operation, record.Outcome, err)
		}
	}
	terminal := run.Terminal
	if terminal == "" {
		terminal = "no-discovery"
	}
	storeBytes, _ := run.Store.CanonicalJSON()
	inputBytes, _ := json.Marshal([]any{"transform-store-boundary/v1", "freeze", digestBytes(storeBytes)})
	if run.Artifact != "" {
		inputBytes = []byte(run.Store.Get(run.Artifact).GetString("schema"))
	}
	terminalBytes, _ := json.Marshal([]any{"transform-terminal/v1", terminal, sink.Work + 1, sink.Applications, len(sink.Events)})
	if err := sink.EmitValues("terminal", "terminal", terminal, 11, [][]byte{inputBytes}, [][]byte{terminalBytes}); err != nil {
		return TransformTranscriptBundle{}, err
	}
	return sink.Bundle()
}
