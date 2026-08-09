package transformexp

import (
	"encoding/json"
	"errors"
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

func countBaselineApplications(events []transformbaseline.Event) int {
	applications := 0
	for _, event := range events {
		if event.Operation == "schema-application" || event.Operation == "replay-application" {
			applications++
		}
	}
	return applications
}

func reserveBaselineApplication(events []transformbaseline.Event, phase string, maximumWork int64) bool {
	applicationCap := ApplicationsPerPolicy
	if phase != "heldout" {
		applicationCap -= 8
	}
	return maximumWork > 0 && countBaselineApplications(events) < applicationCap && baselineEventWork(events)+maximumWork < LifecycleWorkCap
}

func baselineEventsFromTransformMeter(records []dsl.TransformMeterRecord) []transformbaseline.Event {
	events := make([]transformbaseline.Event, len(records))
	for i, record := range records {
		events[i] = transformbaseline.Event{Category: int(record.Category), Operation: record.Operation, Phase: record.Phase, Outcome: record.Outcome, Inputs: record.Inputs, Outputs: record.Outputs}
	}
	return events
}

type applicationReservation struct {
	phase       string
	maximumWork int64
}

func applicationReservations(events []transformbaseline.Event) (map[int]applicationReservation, error) {
	reservations := map[int]applicationReservation{}
	for index, event := range events {
		switch event.Operation {
		case "replay-application":
			if _, exists := reservations[index]; exists {
				return nil, errors.New("overlapping replay application reservation")
			}
			reservations[index] = applicationReservation{event.Phase, 1}
		case "schema-application":
			if len(event.Outputs) != 1 || index+1 >= len(events) || events[index+1].Operation != "evidence-link" {
				return nil, errors.New("schema application lacks immediate evidence boundary")
			}
			var application []json.RawMessage
			var certificate []json.RawMessage
			var first, last int
			if json.Unmarshal(event.Outputs[0], &application) != nil || len(application) != 3 || json.Unmarshal(application[2], &certificate) != nil || len(certificate) != 12 || json.Unmarshal(certificate[10], &first) != nil || json.Unmarshal(certificate[11], &last) != nil || first < 0 || first > index || last != index+1 {
				return nil, errors.New("schema application certificate has invalid reservation range")
			}
			if _, exists := reservations[first]; exists {
				return nil, errors.New("overlapping schema application reservation")
			}
			maximumWork := int64(68)
			if event.Phase == "training-validate" && event.Outcome == "applied" && index+2 < len(events) && events[index+2].Operation == "output-compare" {
				maximumWork = 80
			}
			reservations[first] = applicationReservation{event.Phase, maximumWork}
		}
	}
	return reservations, nil
}

func emitMeteredEvents(sink *TransformTranscriptSink, events []transformbaseline.Event, label string) error {
	reservations, err := applicationReservations(events)
	if err != nil {
		return err
	}
	for index, event := range events {
		if reservation, ok := reservations[index]; ok {
			if err := sink.BeginApplication(reservation.phase, reservation.maximumWork); err != nil {
				return fmt.Errorf("%s event %d application reserve: %w", label, index, err)
			}
		}
		if event.Operation == "evidence-link" {
			if len(event.Inputs) != 1 {
				return fmt.Errorf("%s event %d invalid evidence boundary", label, index)
			}
			if err := sink.EmitEvidenceLink(event.Phase, event.Inputs[0]); err != nil {
				return fmt.Errorf("%s event %d: %w", label, index, err)
			}
			continue
		}
		if err := sink.EmitValues(event.Operation, event.Phase, event.Outcome, event.Category, event.Inputs, event.Outputs); err != nil {
			return fmt.Errorf("%s event %d %s/%s/%s: %w", label, index, event.Phase, event.Operation, event.Outcome, err)
		}
	}
	return nil
}

func transcriptFromBaselineEvents(events []transformbaseline.Event, c policyCurriculum, ordinal int, policy Policy, terminal string, schema, storeBytes []byte) (TransformTranscriptBundle, error) {
	sink, err := newTransformTranscriptSink(ordinal, string(policy), c.PolicyTokens[policy], policyManifestDigest(c, policy))
	if err != nil {
		return TransformTranscriptBundle{}, err
	}
	if err := emitMeteredEvents(sink, events, "baseline"); err != nil {
		return TransformTranscriptBundle{}, err
	}
	input := schema
	if len(storeBytes) != 0 {
		input, _ = json.Marshal([]any{"transform-store-boundary/v1", "freeze", digestBytes(storeBytes)})
	} else if _, parseErr := transformschema.ParseSchema(input); parseErr != nil {
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
	if err := emitMeteredEvents(sink, baselineEventsFromTransformMeter(run.MeterRecords), "meter"); err != nil {
		return TransformTranscriptBundle{}, err
	}
	terminal := run.Terminal
	if terminal == "" {
		terminal = "no-discovery"
	}
	storeBytes, _ := run.Store.CanonicalJSON()
	inputBytes, _ := json.Marshal([]any{"transform-store-boundary/v1", "freeze", digestBytes(storeBytes)})
	terminalBytes, _ := json.Marshal([]any{"transform-terminal/v1", terminal, sink.Work + 1, sink.Applications, len(sink.Events)})
	if err := sink.EmitValues("terminal", "terminal", terminal, 11, [][]byte{inputBytes}, [][]byte{terminalBytes}); err != nil {
		return TransformTranscriptBundle{}, err
	}
	return sink.Bundle()
}
