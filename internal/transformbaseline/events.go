package transformbaseline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Event struct {
	Category  int
	Operation string
	Phase     string
	Outcome   string
	Inputs    [][]byte
	Outputs   [][]byte
}

func ReplayMetered(programBatchBytes []byte, token string, forestBytes []byte, phase string) (Application, []Event, error) {
	application, err := Replay(programBatchBytes, token, forestBytes)
	if err != nil {
		return Application{}, nil, err
	}
	outputDigest := ""
	if len(application.Output) != 0 {
		outputDigest = baselineDigest(application.Output)
	}
	result, _ := json.Marshal([]any{"transform-result/v1", application.Terminal, outputDigest})
	return application, []Event{{11, "replay-application", phase, application.Terminal, [][]byte{forestBytes, programBatchBytes}, [][]byte{result}}}, nil
}

func ApplySchemaMetered(forestBytes, schemaBytes []byte, phase string) (Application, []Event, error) {
	s, err := parseSchema(schemaBytes)
	if err != nil {
		return Application{}, nil, err
	}
	terminal, output, err := apply(forestBytes, s)
	if err != nil {
		return Application{}, nil, err
	}
	return Application{terminal, output}, applicationEvents(forestBytes, schemaBytes, s, terminal, output, phase), nil
}

func CompareOutputsMetered(left, right []byte, phase string) (bool, []Event, error) {
	leftForest, leftErr := parseForest(left)
	rightForest, rightErr := parseForest(right)
	if leftErr != nil || rightErr != nil || len(leftForest.nodes) != len(rightForest.nodes) {
		return false, []Event{{9, "output-compare", phase, "invalid-input", [][]byte{left, right}, nil}}, nil
	}
	equal := true
	events := make([]Event, len(leftForest.nodes))
	for i := range leftForest.nodes {
		nodeEqual := leftForest.nodes[i] == rightForest.nodes[i]
		if !nodeEqual {
			equal = false
		}
		outcome := "different"
		if nodeEqual {
			outcome = "equal"
		}
		events[i] = Event{9, "output-compare", phase, outcome, [][]byte{left, right}, [][]byte{baselineAtom("boolean", nodeEqual)}}
	}
	return equal, events, nil
}

func applicationEvents(forestBytes, schemaBytes []byte, s schema, terminal string, output []byte, phase string) []Event {
	f, err := parseForest(forestBytes)
	if err != nil || terminal != "applied" {
		result, _ := json.Marshal([]any{"transform-result/v1", terminal, ""})
		certificate, _ := json.Marshal([]any{"transform-certificate/v1", baselineDigest(schemaBytes), baselineDigest(forestBytes), -1, -1, []int{}, []bool{}, []string{}, "", terminal, -1, -1})
		application, _ := json.Marshal([]any{"transform-schema-application/v1", json.RawMessage(result), json.RawMessage(certificate)})
		return []Event{{11, "schema-application", phase, terminal, [][]byte{forestBytes, schemaBytes}, [][]byte{application}}}
	}
	var requests, definitions, references []node
	for _, n := range f.nodes {
		switch n.kind {
		case "request":
			requests = append(requests, n)
		case "definition":
			definitions = append(definitions, n)
		case "reference":
			references = append(references, n)
		}
	}
	rq := requests[0]
	var definition node
	for _, candidate := range definitions {
		if s.anchor == "request-target" && candidate.id == rq.target || s.anchor == "from-value" && candidate.value == rq.from || s.anchor == "first-local" && candidate.parent == rq.parent {
			definition = candidate
			break
		}
	}
	var events []Event
	for _, n := range f.nodes {
		facts, _ := json.Marshal([]any{"transform-node-facts/v1", n.kind, n.value, n.from, n.to})
		events = append(events, Event{0, "node", phase, "ok", [][]byte{forestBytes, baselineAtom("id", n.id)}, [][]byte{facts}})
	}
	parentNodes := append(append([]node(nil), references...), rq, definition)
	if s.anchor == "first-local" {
		parentNodes = append(parentNodes, definition)
	}
	for _, n := range parentNodes {
		facts, _ := json.Marshal([]any{"transform-parent-facts/v1", n.parent, n.key})
		events = append(events, Event{1, "parent", phase, "ok", [][]byte{forestBytes, baselineAtom("id", n.id)}, [][]byte{facts}})
	}
	targetNodes := append([]node(nil), references...)
	if s.anchor == "request-target" {
		targetNodes = append(targetNodes, rq)
	}
	for _, n := range targetNodes {
		events = append(events, Event{2, "target", phase, "ok", [][]byte{forestBytes, baselineAtom("id", n.id)}, [][]byte{baselineAtom("id", n.target)}})
	}
	var editWires [][]byte
	var selectedReferences []node
	if s.targets == "definition" || s.targets == "definition+references" {
		wire, _ := json.Marshal([]any{"set-value/v1", definition.id, rq.to})
		editWires = append(editWires, wire)
	}
	if s.targets == "references" || s.targets == "definition+references" {
		for _, n := range references {
			if n.target == definition.id && (s.scope == "global" || n.parent == rq.parent) && (s.guard == "any" || n.value == rq.from) {
				selectedReferences = append(selectedReferences, n)
				wire, _ := json.Marshal([]any{"set-value/v1", n.id, rq.to})
				editWires = append(editWires, wire)
			}
		}
	}
	predicates := 4 + 3*len(references) + len(editWires)
	if s.anchor != "request-target" {
		predicates++
	}
	for index := range predicates {
		events = append(events, Event{8, "schema-predicate", phase, "true", [][]byte{forestBytes, schemaBytes, baselineAtom("enum", "guard"), baselineAtom("id", index)}, [][]byte{baselineAtom("boolean", true)}})
	}
	editDigests := make([]string, len(editWires))
	for i, wire := range editWires {
		editDigests[i] = baselineDigest(wire)
		status, _ := json.Marshal([]any{"transform-edit-status/v1", "valid", editDigests[i]})
		events = append(events,
			Event{6, "edit-validate", phase, "valid", [][]byte{forestBytes, wire}, [][]byte{status}},
			Event{7, "edit-apply", phase, "applied", [][]byte{forestBytes, wire}, [][]byte{output}},
		)
	}
	result, _ := json.Marshal([]any{"transform-result/v1", terminal, baselineDigest(output)})
	guards := make([]bool, predicates)
	for i := range guards {
		guards[i] = true
	}
	var referenceIDs []int
	for _, n := range selectedReferences {
		referenceIDs = append(referenceIDs, n.id)
	}
	certificate, _ := json.Marshal([]any{"transform-certificate/v1", baselineDigest(schemaBytes), baselineDigest(forestBytes), rq.id, definition.id, referenceIDs, guards, editDigests, baselineDigest(output), terminal, -1, -1})
	application, _ := json.Marshal([]any{"transform-schema-application/v1", json.RawMessage(result), json.RawMessage(certificate)})
	events = append(events,
		Event{11, "schema-application", phase, terminal, [][]byte{forestBytes, schemaBytes}, [][]byte{application}},
		Event{10, "evidence-link", phase, "attached", [][]byte{result}, nil},
	)
	return events
}

func outputComparisonEvents(left, right []byte, phase string) []Event {
	_, events, _ := CompareOutputsMetered(left, right, phase)
	return events
}

func baselineAtom(kind string, value any) []byte {
	b, _ := json.Marshal([]any{"transform-atom/v1", kind, value})
	return b
}

func baselineDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
