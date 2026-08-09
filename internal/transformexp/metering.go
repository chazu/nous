package transformexp

import (
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/dsl"
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

func transcriptFromAcquisition(run acquisitionRun, ordinal int, policy Policy, token, manifestDigest string) (TransformTranscriptBundle, error) {
	sink, err := newTransformTranscriptSink(ordinal, string(policy), token, manifestDigest)
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	for index, record := range run.MeterRecords {
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
