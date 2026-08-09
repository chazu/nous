package transformexp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

type transformLifecycleState struct {
	policy          string
	partials        map[string]transformschema.Partial
	allocPhase      map[string]string
	parents         map[string]string
	closures        []string
	survivor        string
	frozenArtifact  string
	history         []TransformOperation
	pendingAttach   int
	activeCandidate string
	factorResults   map[string]bool
}

func newTransformLifecycleState(policy string) *transformLifecycleState {
	return &transformLifecycleState{policy: policy, partials: map[string]transformschema.Partial{}, allocPhase: map[string]string{}, parents: map[string]string{}, pendingAttach: -1, factorResults: map[string]bool{}}
}

func (s *transformLifecycleState) observe(operation TransformOperation, objects map[string][]byte) (returnErr error) {
	sequence := len(s.history)
	defer func() { s.history = append(s.history, operation) }()
	if s.pendingAttach >= 0 {
		if operation.Operation != "evidence-link" || sequence != s.pendingAttach {
			return errors.New("schema application certificate lacks final evidence attachment")
		}
		s.pendingAttach = -1
	}
	switch operation.Operation {
	case "candidate-allocate":
		if operation.Outcome != "allocated" {
			return nil
		}
		digest := operation.Outputs[0]
		if partial, err := transformschema.ParsePartial(objects[digest]); err == nil {
			s.partials[digest] = partial
			s.allocPhase[digest] = operation.Phase
			return nil
		}
	case "refine":
		if operation.Outcome == "refined" {
			partial, err := transformschema.ParsePartial(objects[operation.Outputs[0]])
			if err != nil || partial.Stage > 1 && (len(s.closures) < partial.Stage-1 || operation.Inputs[0] != s.survivor) {
				return errors.New("refinement crossed an unclosed stage")
			}
			s.parents[operation.Outputs[0]] = operation.Inputs[0]
			s.activeCandidate = operation.Outputs[0]
		}
	case "evidence-link":
		if s.activeCandidate != "" && oneOfString(operation.Phase, "target", "anchor", "scope", "old-guard", "locality") {
			var attempt []json.RawMessage
			var version, status, kind string
			if json.Unmarshal(objects[operation.Outputs[0]], &attempt) != nil || len(attempt) != 7 || json.Unmarshal(attempt[0], &version) != nil || version != "transform-evidence-attempt/v1" || json.Unmarshal(attempt[1], &status) != nil || status != "attached" || json.Unmarshal(attempt[2], &kind) != nil || kind != "atom" {
				return errors.New("factor result lacks attached atom evidence")
			}
			atomBytes := bytes.Clone(attempt[3])
			atomKind, value, err := decodeTransformAtom(atomBytes)
			boolean, ok := value.(bool)
			if err != nil || atomKind != "boolean" || !ok {
				return errors.New("factor result evidence is not boolean")
			}
			s.factorResults[s.activeCandidate] = boolean
			s.activeCandidate = ""
		}
	case "schema-application":
		last, err := s.validateApplicationCertificate(sequence, operation, objects)
		if err != nil {
			return err
		}
		s.pendingAttach = last
	case "verify":
		if len(operation.Inputs) != 1 {
			return nil
		}
		if objectVersion(objects[operation.Inputs[0]], "transform-closure/v1") {
			return s.observeClosure(operation, objects)
		}
		if objectVersion(objects[operation.Inputs[0]], "transform-schema/v1") && operation.Phase == "freeze" {
			kind, value, err := decodeTransformAtom(objects[operation.Outputs[0]])
			boolean, ok := value.(bool)
			if err != nil || kind != "boolean" || !ok || !boolean || operation.Outcome != "verified" || s.frozenArtifact != "" && s.frozenArtifact != operation.Inputs[0] {
				return errors.New("invalid frozen schema verification")
			}
			s.frozenArtifact = operation.Inputs[0]
		}
	case "terminal":
		if s.pendingAttach >= 0 {
			return errors.New("terminal precedes application evidence attachment")
		}
		if len(operation.Inputs) != 1 || !oneOfString(objectWireVersion(objects[operation.Inputs[0]]), "transform-closure/v1", "transform-schema/v1", "transform-store-boundary/v1") {
			return errors.New("terminal input is not a closure, schema, or store boundary")
		}
		if s.policy == string(NousRefine) || s.policy == string(NoEqualityGuard) {
			switch operation.Outcome {
			case "completed":
				if len(s.closures) != 5 {
					return errors.New("completed production acquisition lacks five stage closures")
				}
				if s.frozenArtifact == "" || operation.Inputs[0] != s.frozenArtifact {
					return errors.New("production terminal does not name frozen artifact")
				}
			case "no-discovery":
				if len(s.closures) == 0 || s.survivor != "" || s.frozenArtifact != "" {
					return errors.New("no-discovery terminal lacks a rejected stage closure")
				}
			}
		}
	}
	return nil
}

func (s *transformLifecycleState) validateApplicationCertificate(sequence int, operation TransformOperation, objects map[string][]byte) (int, error) {
	var application []json.RawMessage
	if json.Unmarshal(objects[operation.Outputs[0]], &application) != nil || len(application) != 3 {
		return 0, errors.New("invalid application certificate container")
	}
	var certificate []json.RawMessage
	if json.Unmarshal(application[2], &certificate) != nil || len(certificate) != 12 {
		return 0, errors.New("invalid application certificate")
	}
	var version, schemaDigest, inputDigest, outputDigest, terminal string
	var requestID, definitionID int
	var referenceIDs []int
	var guards []bool
	var editDigests []string
	var first, last int
	targets := []any{&version, &schemaDigest, &inputDigest, &requestID, &definitionID, &referenceIDs, &guards, &editDigests, &outputDigest, &terminal, &first, &last}
	for index := range targets {
		if json.Unmarshal(certificate[index], targets[index]) != nil {
			return 0, errors.New("application certificate field")
		}
	}
	if version != "transform-certificate/v1" || schemaDigest != operation.Inputs[1] || inputDigest != operation.Inputs[0] || first < 0 || last != sequence+1 || first > sequence {
		return 0, errors.New("application certificate event range mismatch")
	}
	forest, forestErr := transformschema.ParseForest(objects[operation.Inputs[0]])
	schema, schemaErr := transformschema.ParseSchema(objects[operation.Inputs[1]])
	result, applyErr := schema.Apply(forest)
	if forestErr != nil || schemaErr != nil || applyErr != nil {
		return 0, errors.New("application certificate inputs do not replay")
	}
	trace := append(slices.Clone(s.history[first:]), operation)
	if len(trace) < len(forest.Nodes)+2 {
		return 0, errors.New("application certificate trace is incomplete")
	}
	seenNodes := map[int]bool{}
	var tracedGuards []bool
	var tracedEdits []string
	for index, actual := range trace {
		if actual.Phase != operation.Phase || index < len(forest.Nodes) && actual.Operation != "node" || index == len(trace)-1 && actual.Operation != "schema-application" || index != len(trace)-1 && oneOfString(actual.Operation, "schema-application", "replay-application", "evidence-link", "terminal") {
			return 0, errors.New("application certificate trace shape mismatch")
		}
		if actual.Operation == "node" {
			_, idValue, err := decodeTransformAtom(objects[actual.Inputs[1]])
			id, ok := jsonInteger(idValue)
			if err != nil || !ok || seenNodes[id] {
				return 0, errors.New("application certificate node coverage mismatch")
			}
			seenNodes[id] = true
		}
		if actual.Operation == "schema-predicate" {
			kind, value, err := decodeTransformAtom(objects[actual.Outputs[0]])
			boolean, ok := value.(bool)
			if err != nil || kind != "boolean" || !ok {
				return 0, errors.New("application certificate guard mismatch")
			}
			tracedGuards = append(tracedGuards, boolean)
			selectorKind, selector, _ := decodeTransformAtom(objects[actual.Inputs[2]])
			if selectorKind == "selector" && selector == "edit-no-op" {
				tracedEdits = append(tracedEdits, actual.Inputs[3])
			}
		}
	}
	if len(seenNodes) != len(forest.Nodes) || !slices.Equal(guards, tracedGuards) || !slices.Equal(editDigests, tracedEdits) {
		return 0, errors.New("application certificate evidence summary mismatch")
	}
	wantRequest, wantDefinition, wantReferences, wantEdits := expectedCertificateBindings(forest, schema)
	wantOutput := ""
	if result.Output != nil {
		output, _ := result.Output.CanonicalJSON()
		wantOutput = digestBytes(output)
	}
	if requestID != wantRequest || definitionID != wantDefinition || !slices.Equal(referenceIDs, wantReferences) || !slices.Equal(editDigests, wantEdits) || outputDigest != wantOutput || terminal != result.Terminal || operation.Outcome != result.Terminal {
		return 0, fmt.Errorf("application certificate semantic fields mismatch: binding=%d/%d refs=%v edits=%v output=%s terminal=%s want=%d/%d refs=%v edits=%v output=%s terminal=%s", requestID, definitionID, referenceIDs, editDigests, outputDigest, terminal, wantRequest, wantDefinition, wantReferences, wantEdits, wantOutput, result.Terminal)
	}
	return last, nil
}

func expectedCertificateBindings(forest transformschema.Forest, schema transformschema.Schema) (int, int, []int, []string) {
	requestID, definitionID := -1, -1
	var requests, definitions []transformschema.Node
	byID := map[int]transformschema.Node{}
	for _, node := range forest.Nodes {
		byID[node.ID] = node
		if node.Kind == "request" {
			requests = append(requests, node)
		}
		if node.Kind == "definition" {
			definitions = append(definitions, node)
		}
	}
	if len(requests) != 1 {
		return requestID, definitionID, []int{}, []string{}
	}
	request := requests[0]
	requestID = request.ID
	var candidates []transformschema.Node
	for _, definition := range definitions {
		if schema.Anchor == "request-target" && definition.ID == request.Target || schema.Anchor == "from-value" && definition.Value == request.From || schema.Anchor == "first-local" && definition.Parent == request.Parent {
			candidates = append(candidates, definition)
			if schema.Anchor == "first-local" {
				break
			}
		}
	}
	if len(candidates) != 1 {
		return requestID, definitionID, []int{}, []string{}
	}
	definition := candidates[0]
	definitionID = definition.ID
	if schema.Locality == "required" && definition.Parent != request.Parent {
		return requestID, definitionID, []int{}, []string{}
	}
	var edits []transformschema.Edit
	if schema.Targets == "definition" || schema.Targets == "definition+references" {
		edits = append(edits, transformschema.Edit{Target: definition.ID, Value: request.To})
	}
	var references []int
	if schema.Targets == "references" || schema.Targets == "definition+references" {
		for _, node := range forest.Nodes {
			if node.Kind == "reference" && node.Target == definition.ID && (schema.ReferenceScope == "global" || node.Parent == request.Parent) && (schema.OldGuard == "any" || node.Value == request.From) {
				references = append(references, node.ID)
				edits = append(edits, transformschema.Edit{Target: node.ID, Value: request.To})
			}
		}
	}
	slices.Sort(references)
	slices.SortFunc(edits, func(a, b transformschema.Edit) int { return a.Target - b.Target })
	if len(edits) < 1 || len(edits) > transformschema.MaxEdits {
		return requestID, definitionID, []int{}, []string{}
	}
	digests := make([]string, len(edits))
	noOp := false
	for index, edit := range edits {
		wire, _ := json.Marshal([]any{"set-value/v1", edit.Target, edit.Value})
		digests[index] = digestBytes(wire)
		if byID[edit.Target].Value == edit.Value {
			noOp = true
		}
	}
	if noOp {
		return requestID, definitionID, []int{}, digests
	}
	return requestID, definitionID, references, digests
}

func objectWireVersion(data []byte) string {
	var wire []json.RawMessage
	var version string
	if json.Unmarshal(data, &wire) != nil || len(wire) == 0 || json.Unmarshal(wire[0], &version) != nil {
		return ""
	}
	return version
}

func (s *transformLifecycleState) observeClosure(operation TransformOperation, objects map[string][]byte) error {
	var wire []json.RawMessage
	var version, stage, parent, survivor string
	var alternatives [][]json.RawMessage
	data := objects[operation.Inputs[0]]
	if json.Unmarshal(data, &wire) != nil || len(wire) != 5 || json.Unmarshal(wire[0], &version) != nil || version != "transform-closure/v1" || json.Unmarshal(wire[1], &stage) != nil || json.Unmarshal(wire[2], &parent) != nil || json.Unmarshal(wire[3], &alternatives) != nil || json.Unmarshal(wire[4], &survivor) != nil {
		return errors.New("invalid closure wire")
	}
	stages := []string{"target", "anchor", "scope", "old-guard", "locality"}
	if len(s.closures) >= len(stages) || stage != stages[len(s.closures)] {
		return fmt.Errorf("closure stage order mismatch: got %q after %v", stage, s.closures)
	}
	stageIndex := len(s.closures)
	expectedParent := s.survivor
	if stageIndex == 0 {
		var roots []string
		for digest, partial := range s.partials {
			if partial.Stage == 0 && s.allocPhase[digest] == "target" {
				roots = append(roots, digest)
			}
		}
		if len(roots) == 1 {
			expectedParent = roots[0]
		}
	}
	expected := []string{}
	for digest, partial := range s.partials {
		if partial.Stage == stageIndex+1 && s.allocPhase[digest] == stage {
			expected = append(expected, digest)
		}
	}
	slices.Sort(expected)
	actual := make([]string, len(alternatives))
	survivors := 0
	for index, row := range alternatives {
		var alternative, result, status string
		if len(row) != 3 || json.Unmarshal(row[0], &alternative) != nil || json.Unmarshal(row[1], &result) != nil || json.Unmarshal(row[2], &status) != nil || !digestString(alternative) || !digestString(result) || !oneOfString(status, "survivor", "counterexample", "ablated-ineligible", "redundant-noncanonical") {
			return errors.New("invalid closure alternative")
		}
		actual[index] = alternative
		if s.parents[alternative] != parent {
			return errors.New("closure alternative has wrong refinement parent")
		}
		matched, authenticated := s.factorResults[alternative]
		if !authenticated || matched != (status == "survivor") {
			return errors.New("closure status is not derived from attached factor evidence")
		}
		boolean, _ := json.Marshal([]any{"transform-atom/v1", "boolean", matched})
		if digestBytes(boolean) != result {
			return errors.New("closure result commitment mismatch")
		}
		if matched {
			survivors++
			if survivor != alternative {
				return errors.New("closure survivor commitment mismatch")
			}
		}
	}
	valid := parent != "" && parent == expectedParent && slices.Equal(actual, expected) && len(expected) == []int{3, 3, 2, 2, 2}[stageIndex] && survivors == 1
	kind, value, err := decodeTransformAtom(objects[operation.Outputs[0]])
	boolean, ok := value.(bool)
	if err != nil || kind != "boolean" || !ok || boolean != valid || operation.Outcome != map[bool]string{true: "verified", false: "rejected"}[valid] {
		return fmt.Errorf("closure verification result mismatch")
	}
	if valid {
		s.survivor = survivor
	} else {
		s.survivor = ""
	}
	s.closures = append(s.closures, operation.Inputs[0])
	return nil
}

func objectVersion(data []byte, want string) bool {
	var wire []json.RawMessage
	var version string
	return json.Unmarshal(data, &wire) == nil && len(wire) != 0 && json.Unmarshal(wire[0], &version) == nil && version == want && bytes.Equal(data, mustJSONRaw(wire))
}

func mustJSONRaw(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}
