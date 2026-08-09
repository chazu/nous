package transformbaseline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
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
	return ApplySchemaMeteredAt(forestBytes, schemaBytes, phase, 0)
}

func ApplySchemaMeteredAt(forestBytes, schemaBytes []byte, phase string, sequenceOffset int) (Application, []Event, error) {
	s, err := parseSchema(schemaBytes)
	if err != nil {
		return Application{}, nil, err
	}
	terminal, output, err := apply(forestBytes, s)
	if err != nil {
		return Application{}, nil, err
	}
	return Application{terminal, output}, applicationEvents(forestBytes, schemaBytes, s, terminal, output, phase, sequenceOffset), nil
}

func CompareOutputsMetered(left, right []byte, phase string) (bool, []Event, error) {
	leftForest, leftErr := parseForest(left)
	rightForest, rightErr := parseForest(right)
	if leftErr != nil || rightErr != nil || len(leftForest.nodes) != len(rightForest.nodes) {
		return false, []Event{{9, "output-compare", phase, "invalid-input", [][]byte{left, right}, nil}}, nil
	}
	equal := true
	for i := range leftForest.nodes {
		if leftForest.nodes[i] != rightForest.nodes[i] {
			equal = false
		}
	}
	events := make([]Event, len(leftForest.nodes))
	for i := range leftForest.nodes {
		outcome := "different"
		if equal {
			outcome = "equal"
		}
		events[i] = Event{9, "output-compare", phase, outcome, [][]byte{left, right}, [][]byte{baselineAtom("boolean", equal)}}
	}
	return equal, events, nil
}

func applicationEvents(forestBytes, schemaBytes []byte, s schema, terminal string, output []byte, phase string, sequenceOffset int) []Event {
	f, err := parseForest(forestBytes)
	if err != nil {
		result, _ := json.Marshal([]any{"transform-result/v1", terminal, ""})
		certificate, _ := json.Marshal([]any{"transform-certificate/v1", baselineDigest(schemaBytes), baselineDigest(forestBytes), -1, -1, []int{}, []bool{}, []string{}, "", terminal, sequenceOffset, sequenceOffset + 1})
		application, _ := json.Marshal([]any{"transform-schema-application/v1", json.RawMessage(result), json.RawMessage(certificate)})
		return []Event{{11, "schema-application", phase, terminal, [][]byte{forestBytes, schemaBytes}, [][]byte{application}}, {10, "evidence-link", phase, "attached", [][]byte{result}, nil}}
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
	var events []Event
	for _, n := range f.nodes {
		facts, _ := json.Marshal([]any{"transform-node-facts/v1", n.kind, n.value, n.from, n.to})
		events = append(events, Event{0, "node", phase, "ok", [][]byte{forestBytes, baselineAtom("id", n.id)}, [][]byte{facts}})
	}
	var guardResults []bool
	predicate := func(selector string, subject []byte, result bool) {
		outcome := "false"
		if result {
			outcome = "true"
		}
		events = append(events, Event{8, "schema-predicate", phase, outcome, [][]byte{forestBytes, schemaBytes, baselineAtom("selector", selector), subject}, [][]byte{baselineAtom("boolean", result)}})
		guardResults = append(guardResults, result)
	}
	finish := func(requestID, definitionID int, referenceIDs []int, editDigests []string) []Event {
		if referenceIDs == nil {
			referenceIDs = []int{}
		}
		if editDigests == nil {
			editDigests = []string{}
		}
		outputDigest := ""
		if len(output) != 0 {
			outputDigest = baselineDigest(output)
		}
		result, _ := json.Marshal([]any{"transform-result/v1", terminal, outputDigest})
		certificate, _ := json.Marshal([]any{"transform-certificate/v1", baselineDigest(schemaBytes), baselineDigest(forestBytes), requestID, definitionID, referenceIDs, guardResults, editDigests, outputDigest, terminal, sequenceOffset, sequenceOffset + len(events) + 1})
		application, _ := json.Marshal([]any{"transform-schema-application/v1", json.RawMessage(result), json.RawMessage(certificate)})
		events = append(events, Event{11, "schema-application", phase, terminal, [][]byte{forestBytes, schemaBytes}, [][]byte{application}}, Event{10, "evidence-link", phase, "attached", [][]byte{result}, nil})
		return events
	}
	predicate("request-count", baselineAtom("count", len(requests)), len(requests) == 1)
	if len(requests) != 1 {
		return finish(-1, -1, nil, nil)
	}
	rq := requests[0]
	parentFacts, _ := json.Marshal([]any{"transform-parent-facts/v1", rq.parent, rq.key})
	events = append(events, Event{1, "parent", phase, "ok", [][]byte{forestBytes, baselineAtom("id", rq.id)}, [][]byte{parentFacts}})
	if s.anchor == "request-target" {
		events = append(events, Event{2, "target", phase, "ok", [][]byte{forestBytes, baselineAtom("id", rq.id)}, [][]byte{baselineAtom("id", rq.target)}})
	}
	var candidates []node
	for _, candidate := range definitions {
		if s.anchor == "request-target" && candidate.id == rq.target || s.anchor == "from-value" && candidate.value == rq.from || s.anchor == "first-local" && candidate.parent == rq.parent {
			candidates = append(candidates, candidate)
			if s.anchor == "first-local" {
				break
			}
		}
	}
	predicate("anchor-candidate", baselineAtom("count", len(candidates)), len(candidates) == 1)
	if s.anchor != "request-target" {
		candidateID := -1
		if len(candidates) != 0 {
			candidateID = candidates[0].id
		}
		predicate("anchor-candidate", baselineAtom("id", candidateID), len(candidates) == 1)
	}
	if len(candidates) != 1 {
		return finish(rq.id, -1, nil, nil)
	}
	definition := candidates[0]
	definitionParent, _ := json.Marshal([]any{"transform-parent-facts/v1", definition.parent, definition.key})
	events = append(events, Event{1, "parent", phase, "ok", [][]byte{forestBytes, baselineAtom("id", definition.id)}, [][]byte{definitionParent}})
	if s.anchor == "first-local" {
		events = append(events, Event{1, "parent", phase, "ok", [][]byte{forestBytes, baselineAtom("id", definition.id)}, [][]byte{definitionParent}})
	}
	local := s.locality == "none" || definition.parent == rq.parent
	predicate("anchor-locality", baselineAtom("id", definition.id), local)
	if !local {
		return finish(rq.id, definition.id, nil, nil)
	}
	var editWires [][]byte
	var selectedReferences []node
	if s.targets == "definition" || s.targets == "definition+references" {
		wire, _ := json.Marshal([]any{"set-value/v1", definition.id, rq.to})
		editWires = append(editWires, wire)
	}
	for _, n := range references {
		parent, _ := json.Marshal([]any{"transform-parent-facts/v1", n.parent, n.key})
		events = append(events, Event{1, "parent", phase, "ok", [][]byte{forestBytes, baselineAtom("id", n.id)}, [][]byte{parent}}, Event{2, "target", phase, "ok", [][]byte{forestBytes, baselineAtom("id", n.id)}, [][]byte{baselineAtom("id", n.target)}})
		targetMatch := n.target == definition.id
		scopeMatch := s.scope == "global" || n.parent == rq.parent
		guardMatch := s.guard == "any" || n.value == rq.from
		predicate("reference-target", baselineAtom("id", n.id), targetMatch)
		predicate("reference-scope", baselineAtom("id", n.id), scopeMatch)
		predicate("reference-old-guard", baselineAtom("id", n.id), guardMatch)
		if s.targets == "references" || s.targets == "definition+references" {
			if targetMatch && scopeMatch && guardMatch {
				selectedReferences = append(selectedReferences, n)
				wire, _ := json.Marshal([]any{"set-value/v1", n.id, rq.to})
				editWires = append(editWires, wire)
			}
		}
	}
	slices.SortFunc(editWires, func(left, right []byte) int {
		var a, b []any
		_ = json.Unmarshal(left, &a)
		_ = json.Unmarshal(right, &b)
		return int(a[1].(float64) - b[1].(float64))
	})
	expansionOK := len(editWires) >= 1 && len(editWires) <= 4
	predicate("expansion-bound", baselineAtom("count", len(editWires)), expansionOK)
	if !expansionOK {
		return finish(rq.id, definition.id, nil, nil)
	}
	editDigests := make([]string, len(editWires))
	noOp := false
	for i, wire := range editWires {
		editDigests[i] = baselineDigest(wire)
		var editRow []any
		_ = json.Unmarshal(wire, &editRow)
		targetID := int(editRow[1].(float64))
		value := editRow[2].(string)
		thisNoOp := false
		for _, n := range f.nodes {
			if n.id == targetID {
				thisNoOp = n.value == value
			}
		}
		predicate("edit-no-op", wire, !thisNoOp)
		noOp = noOp || thisNoOp
	}
	if noOp {
		return finish(rq.id, definition.id, nil, editDigests)
	}
	current := forestBytes
	for i, wire := range editWires {
		if editDigests[i] != baselineDigest(wire) {
			panic("edit digest changed after no-op validation")
		}
		status, _ := json.Marshal([]any{"transform-edit-status/v1", "valid", editDigests[i]})
		one, _ := parseProgram(mustProgram(wire))
		intermediate, _ := applyProgram(current, one)
		events = append(events,
			Event{6, "edit-validate", phase, "valid", [][]byte{current, wire}, [][]byte{status}},
			Event{7, "edit-apply", phase, "applied", [][]byte{current, wire}, [][]byte{intermediate.Output}},
		)
		current = intermediate.Output
	}
	var referenceIDs []int
	for _, n := range selectedReferences {
		referenceIDs = append(referenceIDs, n.id)
	}
	return finish(rq.id, definition.id, referenceIDs, editDigests)
}

func mustProgram(edit []byte) []byte {
	var row any
	_ = json.Unmarshal(edit, &row)
	value, _ := json.Marshal([]any{"concrete-program/v1", []any{row}})
	return value
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
