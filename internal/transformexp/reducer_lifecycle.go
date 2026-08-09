package transformexp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/transformbaseline"
	"github.com/chazu/nous/internal/transformfixturecore"
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
	pendingEvidence *TransformOperation
	activeCandidate string
	factorStart     int
	factorResults   map[string]bool
	acquireBefore   string
	acquireAfter    string
	acquireEdits    []transformschema.Edit
	acquireCompare  int
	programs        []reconstructedProgram
	batchVerified   bool
	trainingCases   map[string]transformfixturecore.TrainingCase
	trainingOrder   []transformfixturecore.TrainingCase
	afterEvidence   []TransformOperation
	pendingCompare  []TransformOperation
}

type reconstructedProgram struct {
	before transformschema.Forest
	after  transformschema.Forest
	edits  []transformschema.Edit
}

func newTransformLifecycleState(policy string, training []byte) (*transformLifecycleState, error) {
	state := &transformLifecycleState{policy: policy, partials: map[string]transformschema.Partial{}, allocPhase: map[string]string{}, parents: map[string]string{}, pendingAttach: -1, factorStart: -1, factorResults: map[string]bool{}, trainingCases: map[string]transformfixturecore.TrainingCase{}}
	if len(training) == 0 {
		return state, nil
	}
	fixture, err := transformfixturecore.ParseTraining(training)
	if err != nil {
		return nil, err
	}
	for _, item := range fixture.Cases {
		state.trainingCases[digestBytes(item.Before)] = item
		state.trainingOrder = append(state.trainingOrder, item)
	}
	return state, nil
}

func decodeLifecycleEdit(data []byte) (transformschema.Edit, error) {
	programBytes, err := json.Marshal([]any{"concrete-program/v1", []json.RawMessage{data}})
	if err != nil {
		return transformschema.Edit{}, err
	}
	program, err := transformschema.ParseProgram(programBytes)
	if err != nil || len(program.Edits) != 1 {
		return transformschema.Edit{}, errors.New("invalid acquired edit")
	}
	return program.Edits[0], nil
}

func (s *transformLifecycleState) reconstructFactor(partial transformschema.Partial, phase string, trace []TransformOperation, objects map[string][]byte) (bool, error) {
	wantStage := map[string]int{"target": 1, "anchor": 2, "scope": 3, "old-guard": 4, "locality": 5}[phase]
	if wantStage == 0 || partial.Stage != wantStage || len(s.programs) != 4 {
		return false, errors.New("factor evidence has wrong stage or incomplete acquired programs")
	}
	var scans []transformschema.Forest
	var err error
	if phase == "target" {
		if err := requireTargetObservations(trace, s.programs, objects); err != nil {
			return false, err
		}
	} else {
		scans, err = completeFactorScans(trace, objects)
		if err != nil {
			return false, err
		}
	}
	if err := s.requireFactorComparisons(partial, phase, trace, objects); err != nil {
		return false, err
	}
	if phase == "locality" {
		if len(scans) != 4 {
			return false, fmt.Errorf("locality factor observed %d complete training forests, want 4", len(scans))
		}
		var expectedNegatives []transformschema.Forest
		for _, item := range s.trainingOrder {
			if item.Kind == "abstain" {
				forest, parseErr := transformschema.ParseForest(item.Before)
				if parseErr != nil {
					return false, parseErr
				}
				expectedNegatives = append(expectedNegatives, forest)
			}
		}
		sawWrongContext := false
		for index, forest := range scans {
			if len(expectedNegatives) != 0 {
				if len(expectedNegatives) != len(scans) {
					return false, errors.New("locality factor row count differs from training source")
				}
				got, _ := forest.CanonicalJSON()
				want, _ := expectedNegatives[index].CanonicalJSON()
				if !bytes.Equal(got, want) {
					return false, fmt.Errorf("locality factor training rows are reordered at %d", index)
				}
			}
			requests := lifecycleNodesOfKind(forest, "request")
			if len(requests) == 1 {
				request := requests[0]
				definition, ok := lifecycleNodeByID(forest, request.Target)
				if ok && definition.Kind == "definition" && request.Parent != definition.Parent {
					sawWrongContext = true
				}
			}
		}
		return partial.Locality == "required" && sawWrongContext, nil
	}
	requiresProgramScans := phase != "target" && !(phase == "scope" && partial.Targets == "definition") && !(phase == "old-guard" && partial.Targets == "definition")
	if requiresProgramScans {
		if len(scans) != 4 {
			return false, fmt.Errorf("factor observed %d complete program scans in %s, want 4", len(scans), phase)
		}
		for index, program := range s.programs {
			got, _ := scans[index].CanonicalJSON()
			want, _ := program.before.CanonicalJSON()
			if !bytes.Equal(got, want) {
				return false, fmt.Errorf("factor program rows are missing or reordered at %d in %s", index, phase)
			}
		}
	}
	exact := true
	for _, program := range s.programs {
		matched, matchErr := factorMatchesProgram(partial, phase, program)
		if matchErr != nil {
			return false, matchErr
		}
		exact = exact && matched
	}
	if s.policy == string(NoEqualityGuard) && phase == "old-guard" && partial.OldGuard == "equals-from" {
		exact = false
	}
	return exact, nil
}

func (s *transformLifecycleState) requireFactorComparisons(partial transformschema.Partial, phase string, trace []TransformOperation, objects map[string][]byte) error {
	atom := func(kind string, value any) []byte { return mustJSON([]any{"transform-atom/v1", kind, value}) }
	type pair struct{ left, right []byte }
	for index, operation := range trace {
		if operation.Phase != phase {
			return fmt.Errorf("%s factor operation %d is relabeled %s", phase, index, operation.Phase)
		}
	}
	definitionNormalization := (phase == "scope" || phase == "old-guard") && partial.Targets == "definition"
	if definitionNormalization {
		canonical := "local"
		candidate := partial.ReferenceScope
		if phase == "old-guard" {
			canonical, candidate = "any", partial.OldGuard
		}
		expected := pair{atom("enum", candidate), atom("enum", canonical)}
		matched, observations := 0, 0
		for _, operation := range trace {
			if oneOfString(operation.Operation, "node", "parent", "target") {
				observations++
			}
			if operation.Operation == "compare" && len(operation.Inputs) == 2 && bytes.Equal(objects[operation.Inputs[0]], expected.left) && bytes.Equal(objects[operation.Inputs[1]], expected.right) {
				matched++
			}
		}
		if observations != 0 || matched != 1 {
			return fmt.Errorf("%s definition-only normalization has observations=%d comparisons=%d", phase, observations, matched)
		}
		return requireFactorAggregate(trace, 1, objects)
	}
	if phase == "target" {
		return requireTargetFactorProvenance(partial, trace, s.programs, objects)
	}
	blocks, err := completeFactorScanBlocks(trace, objects)
	if err != nil || len(blocks) != 4 {
		return fmt.Errorf("%s factor scan blocks: %w", phase, err)
	}
	for _, operation := range trace[:blocks[0].start] {
		if oneOfString(operation.Operation, "node", "parent", "target") {
			return fmt.Errorf("%s factor has structural evidence before its first row", phase)
		}
	}
	expectedByBlock := make([][]pair, 4)
	wantKind := "id"
	if phase == "locality" {
		for index, block := range blocks {
			forest := block.forest
			requests := lifecycleNodesOfKind(forest, "request")
			if len(requests) == 1 {
				definition, ok := lifecycleNodeByID(forest, requests[0].Target)
				if ok {
					expectedByBlock[index] = []pair{{atom("id", requests[0].Parent), atom("id", definition.Parent)}}
				}
			}
		}
	} else {
		for index, program := range s.programs {
			request, definitionID, editedReferences, signature, err := lifecycleProgramFacts(program)
			if err != nil {
				return err
			}
			switch phase {
			case "anchor":
				expectedByBlock[index] = []pair{{atom("id", resolveLifecycleAnchor(program.before, request, partial.Anchor)), atom("id", definitionID)}}
			case "scope":
				wantKind = "id-set"
				expectedByBlock[index] = []pair{
					{atom("id-set", projectLifecycleReferences(program.before, request, definitionID, partial.ReferenceScope, "equals-from")), atom("id-set", editedReferences)},
					{atom("id-set", projectLifecycleReferences(program.before, request, definitionID, partial.ReferenceScope, "any")), atom("id-set", editedReferences)}}
			case "old-guard":
				wantKind = "id-set"
				expectedByBlock[index] = []pair{{atom("id-set", projectLifecycleReferences(program.before, request, definitionID, partial.ReferenceScope, partial.OldGuard)), atom("id-set", editedReferences)}}
			default:
				return errors.New("unsupported factor comparison phase")
			}
			_ = signature
		}
	}
	for index, block := range blocks {
		end := len(trace)
		if index+1 < len(blocks) {
			end = blocks[index+1].start
		}
		var actual []pair
		lastObservation, firstTypedComparison := -1, -1
		for offset, operation := range trace[block.start:end] {
			if oneOfString(operation.Operation, "node", "parent", "target") {
				lastObservation = offset
			}
			if operation.Operation != "compare" || len(operation.Inputs) != 2 || len(operation.Outputs) != 1 {
				continue
			}
			leftKind, _, leftErr := decodeTransformAtom(objects[operation.Inputs[0]])
			rightKind, _, rightErr := decodeTransformAtom(objects[operation.Inputs[1]])
			if leftErr == nil && rightErr == nil && leftKind == wantKind && rightKind == wantKind {
				if firstTypedComparison < 0 {
					firstTypedComparison = offset
				}
				actual = append(actual, pair{objects[operation.Inputs[0]], objects[operation.Inputs[1]]})
			}
		}
		if len(actual) != 0 && firstTypedComparison <= lastObservation {
			return fmt.Errorf("%s row %d comparison precedes its final structural observation", phase, index)
		}
		if len(actual) != len(expectedByBlock[index]) {
			forestBytes, _ := block.forest.CanonicalJSON()
			return fmt.Errorf("%s row %d forest=%s authenticated %d comparisons %v, want %d", phase, index, forestBytes, len(actual), actual, len(expectedByBlock[index]))
		}
		for comparison := range actual {
			if !bytes.Equal(actual[comparison].left, expectedByBlock[index][comparison].left) || !bytes.Equal(actual[comparison].right, expectedByBlock[index][comparison].right) {
				return fmt.Errorf("%s row %d comparison %d has wrong operands", phase, index, comparison)
			}
		}
		var program *reconstructedProgram
		if phase != "locality" {
			program = &s.programs[index]
		}
		if err := requireFactorObservationBlock(phase, partial, block.forest, program, trace[block.start:end], objects); err != nil {
			return fmt.Errorf("%s row %d observations: %w", phase, index, err)
		}
	}
	totalExpected := 0
	for _, expected := range expectedByBlock {
		totalExpected += len(expected)
	}
	return requireFactorAggregate(trace, totalExpected, objects)
}

type factorEvidenceStep struct {
	operation, forest string
	id                int
	left, right       []byte
}

func requireTargetFactorProvenance(partial transformschema.Partial, trace []TransformOperation, programs []reconstructedProgram, objects map[string][]byte) error {
	atom := func(kind string, value any) []byte { return mustJSON([]any{"transform-atom/v1", kind, value}) }
	var expected []factorEvidenceStep
	var comparisonPairs [][2][]byte
	for _, program := range programs {
		forest, _ := program.before.CanonicalJSON()
		forestDigest := digestBytes(forest)
		_, _, _, signature, err := lifecycleProgramFacts(program)
		if err != nil {
			return err
		}
		for _, edit := range program.edits {
			expected = append(expected, factorEvidenceStep{operation: "node", forest: forestDigest, id: edit.Target})
		}
		left, right := atom("enum", partial.Targets), atom("enum", signature)
		expected = append(expected, factorEvidenceStep{operation: "compare", left: left, right: right})
		comparisonPairs = append(comparisonPairs, [2][]byte{left, right})
	}
	var actual []factorEvidenceStep
	for _, operation := range trace {
		switch operation.Operation {
		case "node":
			forest, id, ok := operationForestID(operation, objects)
			if !ok {
				return errors.New("target observation has invalid forest/id source")
			}
			actual = append(actual, factorEvidenceStep{operation: "node", forest: forest, id: id})
		case "parent", "target":
			return fmt.Errorf("target factor contains extra %s observation", operation.Operation)
		case "compare":
			if len(operation.Inputs) != 2 {
				continue
			}
			left, right := objects[operation.Inputs[0]], objects[operation.Inputs[1]]
			for _, pair := range comparisonPairs {
				if bytes.Equal(left, pair[0]) && bytes.Equal(right, pair[1]) {
					actual = append(actual, factorEvidenceStep{operation: "compare", left: left, right: right})
					break
				}
			}
		}
	}
	if len(actual) != len(expected) {
		return fmt.Errorf("target provenance steps=%d want=%d", len(actual), len(expected))
	}
	for index := range expected {
		if actual[index].operation != expected[index].operation || actual[index].forest != expected[index].forest || actual[index].id != expected[index].id || !bytes.Equal(actual[index].left, expected[index].left) || !bytes.Equal(actual[index].right, expected[index].right) {
			return fmt.Errorf("target provenance differs at step %d", index)
		}
	}
	return requireFactorAggregate(trace, len(programs), objects)
}

func requireFactorAggregate(trace []TransformOperation, stageComparisons int, objects map[string][]byte) error {
	comparisons := 0
	for _, operation := range trace {
		if operation.Operation == "compare" {
			comparisons++
		}
	}
	if comparisons != stageComparisons+1 || len(trace) == 0 {
		return fmt.Errorf("factor comparisons=%d, want %d stage plus one aggregate", comparisons, stageComparisons)
	}
	aggregate := trace[len(trace)-1]
	if aggregate.Operation != "compare" || len(aggregate.Inputs) != 2 || len(aggregate.Outputs) != 1 || aggregate.Outputs[0] != aggregate.Inputs[0] {
		return errors.New("factor aggregate comparison is not the final proof operation")
	}
	leftKind, leftValue, leftErr := decodeTransformAtom(objects[aggregate.Inputs[0]])
	rightKind, rightValue, rightErr := decodeTransformAtom(objects[aggregate.Inputs[1]])
	left, leftOK := leftValue.(bool)
	right, rightOK := rightValue.(bool)
	if leftErr != nil || rightErr != nil || leftKind != "boolean" || rightKind != "boolean" || !leftOK || !rightOK || !right || aggregate.Outcome != map[bool]string{true: "true", false: "false"}[left] {
		return errors.New("factor aggregate comparison does not bind claimed boolean to true")
	}
	return nil
}

func requireFactorObservationBlock(phase string, partial transformschema.Partial, forest transformschema.Forest, program *reconstructedProgram, trace []TransformOperation, objects map[string][]byte) error {
	encoded, _ := forest.CanonicalJSON()
	forestDigest := digestBytes(encoded)
	type observation struct {
		operation string
		id        int
	}
	var expected []observation
	appendObservation := func(operation string, id int) { expected = append(expected, observation{operation, id}) }
	for id := 0; id < transformschema.MaxNodes; id++ {
		appendObservation("node", id)
		node, ok := lifecycleNodeByID(forest, id)
		if !ok {
			continue
		}
		switch phase {
		case "scope", "old-guard":
			if node.Kind == "request" || node.Kind == "reference" || node.Kind == "definition" {
				appendObservation("parent", id)
			}
			if node.Kind == "reference" {
				appendObservation("target", id)
			}
		case "anchor":
			if node.Kind == "definition" {
				appendObservation("parent", id)
			}
		}
	}
	requestNodes := lifecycleNodesOfKind(forest, "request")
	if len(requestNodes) != 1 {
		if phase != "locality" {
			return errors.New("factor source lacks one request")
		}
	} else {
		request := requestNodes[0]
		switch phase {
		case "anchor":
			if program == nil {
				return errors.New("anchor source lacks program")
			}
			for _, edit := range program.edits {
				appendObservation("node", edit.Target)
				node, _ := lifecycleNodeByID(forest, edit.Target)
				if node.Kind == "reference" {
					appendObservation("target", edit.Target)
				}
			}
			switch partial.Anchor {
			case "request-target":
				appendObservation("target", request.ID)
			case "from-value":
				appendObservation("node", request.ID)
			case "first-local":
				appendObservation("parent", request.ID)
			}
		case "locality":
			appendObservation("target", request.ID)
			appendObservation("parent", request.ID)
			appendObservation("parent", request.Target)
		}
	}
	var actual []observation
	for _, operation := range trace {
		if !oneOfString(operation.Operation, "node", "parent", "target") {
			continue
		}
		gotForest, id, ok := operationForestID(operation, objects)
		if !ok || gotForest != forestDigest {
			return errors.New("structural observation is not bound to row forest")
		}
		actual = append(actual, observation{operation.Operation, id})
	}
	if !slices.Equal(actual, expected) {
		return fmt.Errorf("structural observation order/count differs: got=%v want=%v", actual, expected)
	}
	return nil
}

func operationForestID(operation TransformOperation, objects map[string][]byte) (string, int, bool) {
	if len(operation.Inputs) != 2 {
		return "", 0, false
	}
	if _, err := transformschema.ParseForest(objects[operation.Inputs[0]]); err != nil {
		return "", 0, false
	}
	kind, value, err := decodeTransformAtom(objects[operation.Inputs[1]])
	id, ok := jsonInteger(value)
	return operation.Inputs[0], id, err == nil && kind == "id" && ok
}

func lifecycleProgramFacts(program reconstructedProgram) (transformschema.Node, int, []int, string, error) {
	byID := map[int]transformschema.Node{}
	var request *transformschema.Node
	for _, node := range program.before.Nodes {
		byID[node.ID] = node
		if node.Kind == "request" {
			copy := node
			request = &copy
		}
	}
	if request == nil {
		return transformschema.Node{}, -1, nil, "", errors.New("program lacks request")
	}
	definitionID := -1
	definition, reference := false, false
	var references []int
	for _, edit := range program.edits {
		node, ok := byID[edit.Target]
		if !ok {
			return transformschema.Node{}, -1, nil, "", errors.New("program edit target absent")
		}
		if node.Kind == "definition" {
			definition, definitionID = true, node.ID
		} else if node.Kind == "reference" {
			reference, definitionID = true, node.Target
			references = append(references, node.ID)
		}
	}
	slices.Sort(references)
	signature := "definition+references"
	if definition && !reference {
		signature = "definition"
	} else if reference && !definition {
		signature = "references"
	}
	return *request, definitionID, references, signature, nil
}

func requireTargetObservations(trace []TransformOperation, programs []reconstructedProgram, objects map[string][]byte) error {
	type observation struct {
		forest string
		id     int
	}
	var observed []observation
	for _, operation := range trace {
		if operation.Operation != "node" || operation.Outcome != "ok" || len(operation.Inputs) != 2 {
			continue
		}
		_, value, err := decodeTransformAtom(objects[operation.Inputs[1]])
		id, ok := jsonInteger(value)
		if err == nil && ok {
			observed = append(observed, observation{operation.Inputs[0], id})
		}
	}
	var expected []observation
	for _, program := range programs {
		forestBytes, _ := program.before.CanonicalJSON()
		forestDigest := digestBytes(forestBytes)
		for _, edit := range program.edits {
			expected = append(expected, observation{forestDigest, edit.Target})
		}
	}
	if !slices.Equal(observed, expected) {
		return errors.New("target factor edited-node observations are missing, extra, or reordered")
	}
	return nil
}

func factorMatchesProgram(partial transformschema.Partial, phase string, program reconstructedProgram) (bool, error) {
	byID := map[int]transformschema.Node{}
	var request *transformschema.Node
	for index := range program.before.Nodes {
		node := program.before.Nodes[index]
		byID[node.ID] = node
		if node.Kind == "request" {
			copy := node
			request = &copy
		}
	}
	if request == nil {
		return false, errors.New("acquired program forest lacks request")
	}
	editedKinds := map[string]bool{}
	definitionID := -1
	var editedReferences []int
	for _, edit := range program.edits {
		node, ok := byID[edit.Target]
		if !ok || edit.Value != request.To {
			return false, errors.New("acquired program edit is not local to its request")
		}
		editedKinds[node.Kind] = true
		switch node.Kind {
		case "definition":
			if definitionID >= 0 && definitionID != node.ID {
				return false, errors.New("acquired program has multiple definition anchors")
			}
			definitionID = node.ID
		case "reference":
			if definitionID >= 0 && definitionID != node.Target {
				return false, errors.New("acquired program has inconsistent reference anchors")
			}
			definitionID = node.Target
			editedReferences = append(editedReferences, node.ID)
		default:
			return false, errors.New("acquired program edits a non-transform node")
		}
	}
	slices.Sort(editedReferences)
	switch phase {
	case "target":
		return partial.Targets == "definition" && editedKinds["definition"] && !editedKinds["reference"] || partial.Targets == "references" && editedKinds["reference"] && !editedKinds["definition"] || partial.Targets == "definition+references" && editedKinds["definition"] && editedKinds["reference"], nil
	case "anchor":
		return resolveLifecycleAnchor(program.before, *request, partial.Anchor) == definitionID, nil
	case "scope":
		if partial.Targets == "definition" {
			return partial.ReferenceScope == "local", nil
		}
		for _, guard := range []string{"equals-from", "any"} {
			if slices.Equal(projectLifecycleReferences(program.before, *request, definitionID, partial.ReferenceScope, guard), editedReferences) {
				return true, nil
			}
		}
		return false, nil
	case "old-guard":
		if partial.Targets == "definition" {
			return partial.OldGuard == "any", nil
		}
		return slices.Equal(projectLifecycleReferences(program.before, *request, definitionID, partial.ReferenceScope, partial.OldGuard), editedReferences), nil
	default:
		return false, errors.New("unsupported factor phase")
	}
}

func resolveLifecycleAnchor(forest transformschema.Forest, request transformschema.Node, anchor string) int {
	switch anchor {
	case "request-target":
		return request.Target
	case "from-value":
		selected := -1
		for _, node := range forest.Nodes {
			if node.Kind == "definition" && node.Value == request.From {
				if selected >= 0 {
					return -2
				}
				selected = node.ID
			}
		}
		return selected
	case "first-local":
		selected := -1
		for _, node := range forest.Nodes {
			if node.Kind == "definition" && node.Parent == request.Parent && (selected < 0 || node.ID < selected) {
				selected = node.ID
			}
		}
		return selected
	default:
		return -3
	}
}

func projectLifecycleReferences(forest transformschema.Forest, request transformschema.Node, definitionID int, scope, guard string) []int {
	var result []int
	for _, node := range forest.Nodes {
		if node.Kind == "reference" && node.Target == definitionID && (scope == "global" || node.Parent == request.Parent) && (guard == "any" || node.Value == request.From) {
			result = append(result, node.ID)
		}
	}
	slices.Sort(result)
	return result
}

func lifecycleNodesOfKind(forest transformschema.Forest, kind string) []transformschema.Node {
	var result []transformschema.Node
	for _, node := range forest.Nodes {
		if node.Kind == kind {
			result = append(result, node)
		}
	}
	return result
}

func lifecycleNodeByID(forest transformschema.Forest, id int) (transformschema.Node, bool) {
	for _, node := range forest.Nodes {
		if node.ID == id {
			return node, true
		}
	}
	return transformschema.Node{}, false
}

type factorScanBlock struct {
	forest     transformschema.Forest
	start, end int
}

func completeFactorScans(trace []TransformOperation, objects map[string][]byte) ([]transformschema.Forest, error) {
	blocks, err := completeFactorScanBlocks(trace, objects)
	if err != nil {
		return nil, err
	}
	result := make([]transformschema.Forest, len(blocks))
	for index := range blocks {
		result[index] = blocks[index].forest
	}
	return result, nil
}

func completeFactorScanBlocks(trace []TransformOperation, objects map[string][]byte) ([]factorScanBlock, error) {
	var result []factorScanBlock
	forestDigest := ""
	nextID, start := 0, -1
	for index, operation := range trace {
		if operation.Operation != "node" || len(operation.Inputs) != 2 {
			continue
		}
		_, value, err := decodeTransformAtom(objects[operation.Inputs[1]])
		id, ok := jsonInteger(value)
		if err != nil || !ok || id < 0 || id >= transformschema.MaxNodes {
			return nil, errors.New("factor node observation has invalid id")
		}
		if forestDigest == "" {
			if id != 0 {
				continue
			}
			forestDigest = operation.Inputs[0]
			nextID = 0
			start = index
		}
		if forestDigest != operation.Inputs[0] || id != nextID {
			forestDigest = ""
			nextID = 0
			start = -1
			if id == 0 {
				forestDigest = operation.Inputs[0]
				nextID = 1
				start = index
			}
			continue
		}
		nextID++
		if nextID == transformschema.MaxNodes {
			forest, err := transformschema.ParseForest(objects[forestDigest])
			if err != nil {
				return nil, err
			}
			result = append(result, factorScanBlock{forest: forest, start: start, end: index + 1})
			forestDigest = ""
			nextID = 0
			start = -1
		}
	}
	return result, nil
}

func (s *transformLifecycleState) observe(operation TransformOperation, objects map[string][]byte) (returnErr error) {
	sequence := len(s.history)
	defer func() { s.history = append(s.history, operation) }()
	if len(s.pendingCompare) != 0 {
		want := s.pendingCompare[0]
		if !equalLifecycleOperation(operation, want) {
			return errors.New("applied positive training output comparisons differ from committed fixture")
		}
		s.pendingCompare = s.pendingCompare[1:]
		return nil
	}
	if s.pendingAttach >= 0 {
		if operation.Operation != "evidence-link" || sequence != s.pendingAttach {
			return errors.New("schema application certificate lacks final evidence attachment")
		}
		if s.pendingEvidence == nil || !equalApplicationEvidence(operation, *s.pendingEvidence) {
			return errors.New("schema application evidence differs from exact reconstructed trace")
		}
		s.pendingAttach = -1
		s.pendingEvidence = nil
		s.pendingCompare = s.afterEvidence
		s.afterEvidence = nil
	}
	switch operation.Operation {
	case "candidate-allocate":
		digest := operation.Inputs[0]
		_, alreadyAllocated := s.partials[digest]
		if operation.Outcome == "duplicate" {
			if !alreadyAllocated {
				return errors.New("candidate duplicate has no prior allocation")
			}
			return nil
		}
		if operation.Outcome != "allocated" {
			_, partialErr := transformschema.ParsePartial(objects[digest])
			_, schemaErr := transformschema.ParseSchema(objects[digest])
			if partialErr == nil || schemaErr == nil {
				return errors.New("valid candidate was rejected")
			}
			return nil
		}
		if alreadyAllocated {
			return errors.New("candidate was allocated twice")
		}
		digest = operation.Outputs[0]
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
			s.factorStart = sequence + 1
		}
	case "edit-validate":
		if operation.Phase == "acquire" && operation.Outcome == "valid" {
			if s.acquireBefore == "" {
				s.acquireBefore = operation.Inputs[0]
			}
			edit, err := decodeLifecycleEdit(objects[operation.Inputs[1]])
			if err != nil {
				return err
			}
			s.acquireEdits = append(s.acquireEdits, edit)
		}
	case "output-compare":
		if operation.Phase == "acquire" && len(s.acquireEdits) != 0 {
			before, beforeErr := transformschema.ParseForest(objects[s.acquireBefore])
			after, afterErr := transformschema.ParseForest(objects[operation.Inputs[1]])
			program := transformschema.Program{Edits: slices.Clone(s.acquireEdits)}
			actual, applyErr := program.Apply(before)
			actualBytes, canonicalErr := actual.CanonicalJSON()
			kind, selector, selectorErr := decodeTransformAtom(objects[operation.Inputs[2]])
			id, idOK := jsonInteger(selector)
			if beforeErr != nil || afterErr != nil || applyErr != nil || canonicalErr != nil || selectorErr != nil || kind != "id" || !idOK || operation.Outcome != "equal" || !bytes.Equal(actualBytes, objects[operation.Inputs[0]]) || !bytes.Equal(actualBytes, objects[operation.Inputs[1]]) || id != s.acquireCompare || s.acquireAfter != "" && s.acquireAfter != operation.Inputs[1] {
				return errors.New("concrete program acquisition does not reconstruct")
			}
			s.acquireAfter = operation.Inputs[1]
			s.acquireCompare++
			if s.acquireCompare == len(after.Nodes) {
				s.programs = append(s.programs, reconstructedProgram{before, after, slices.Clone(s.acquireEdits)})
				s.acquireBefore = ""
				s.acquireAfter = ""
				s.acquireEdits = nil
				s.acquireCompare = 0
			}
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
			partial, exists := s.partials[s.activeCandidate]
			if !exists || s.factorStart < 0 || s.factorStart > len(s.history) {
				return errors.New("factor result has no allocated active partial")
			}
			expected, err := s.reconstructFactor(partial, operation.Phase, s.history[s.factorStart:], objects)
			if err != nil {
				return err
			}
			if boolean != expected {
				return errors.New("factor result differs from independently reconstructed observations")
			}
			s.factorResults[s.activeCandidate] = expected
			s.activeCandidate = ""
			s.factorStart = -1
		}
	case "schema-application":
		evidence, err := s.validateApplicationCertificate(sequence, operation, objects)
		if err != nil {
			return err
		}
		s.pendingAttach = sequence + 1
		s.pendingEvidence = &evidence
		if operation.Phase == "training-validate" && operation.Outcome == "applied" && len(s.trainingCases) != 0 {
			item, exists := s.trainingCases[operation.Inputs[0]]
			if !exists {
				return errors.New("training application input is absent from committed fixture")
			}
			if item.Kind == "positive" {
				var application []json.RawMessage
				var resultWire []json.RawMessage
				var outputDigest string
				if json.Unmarshal(objects[operation.Outputs[0]], &application) != nil || len(application) != 3 || json.Unmarshal(application[1], &resultWire) != nil || len(resultWire) != 3 || json.Unmarshal(resultWire[2], &outputDigest) != nil {
					return errors.New("training application result projection")
				}
				_, expected, compareErr := transformbaseline.CompareOutputsMetered(objects[outputDigest], item.After, "training-validate")
				if compareErr != nil {
					return compareErr
				}
				s.afterEvidence = make([]TransformOperation, len(expected))
				for index := range expected {
					s.afterEvidence[index] = lifecycleOperationFromBaseline(expected[index])
				}
			}
		}
	case "verify":
		if len(operation.Inputs) != 1 {
			return nil
		}
		if objectVersion(objects[operation.Inputs[0]], "transform-closure/v1") {
			if operation.Phase != "freeze" {
				return errors.New("closure verification outside freeze phase")
			}
			return s.observeClosure(operation, objects)
		}
		if objectVersion(objects[operation.Inputs[0]], "transform-program-batch/v1") && operation.Phase == "acquire" {
			if s.batchVerified || len(s.programs) != 4 || s.acquireBefore != "" || s.acquireAfter != "" || len(s.acquireEdits) != 0 || s.acquireCompare != 0 {
				return errors.New("program batch verified before four complete acquisitions")
			}
			batch, err := transformfixturecore.ParseProgramBatch(objects[operation.Inputs[0]])
			kind, value, atomErr := decodeTransformAtom(objects[operation.Outputs[0]])
			verified, ok := value.(bool)
			if err != nil || atomErr != nil || kind != "boolean" || !ok || !verified || operation.Outcome != "verified" {
				return errors.New("invalid program batch verification")
			}
			expectedRows := make([]transformfixturecore.ProgramRow, len(s.programs))
			for index, program := range s.programs {
				beforeBytes, _ := program.before.CanonicalJSON()
				programBytes, _ := (transformschema.Program{Edits: slices.Clone(program.edits)}).CanonicalJSON()
				fixture, exists := s.trainingCases[digestBytes(beforeBytes)]
				if len(s.trainingCases) != 0 && (!exists || fixture.Kind != "positive") {
					return errors.New("verified program has no positive fixture row")
				}
				if exists {
					expectedRows[index] = transformfixturecore.ProgramRow{Token: fixture.Token, BeforeDigest: digestBytes(beforeBytes), Program: programBytes}
				} else {
					matched := false
					for _, row := range batch.Rows {
						if row.BeforeDigest == digestBytes(beforeBytes) && bytes.Equal(row.Program, programBytes) {
							expectedRows[index] = row
							matched = true
							break
						}
					}
					if !matched {
						return errors.New("verified batch omits a promoted program")
					}
				}
			}
			expectedBytes, expectedErr := (transformfixturecore.ProgramBatch{Rows: expectedRows}).CanonicalJSON()
			actualBytes, actualErr := batch.CanonicalJSON()
			if expectedErr != nil || actualErr != nil || !bytes.Equal(expectedBytes, actualBytes) {
				return errors.New("program batch differs from promoted programs")
			}
			s.batchVerified = true
			return nil
		}
		if objectVersion(objects[operation.Inputs[0]], "transform-schema/v1") && operation.Phase == "freeze" {
			kind, value, err := decodeTransformAtom(objects[operation.Outputs[0]])
			boolean, ok := value.(bool)
			schema, schemaErr := transformschema.ParseSchema(objects[operation.Inputs[0]])
			survivor, survivorOK := s.partials[s.survivor]
			production := s.policy == string(NousRefine) || s.policy == string(NoEqualityGuard)
			exactAssembly := schemaErr == nil
			if production {
				exactAssembly = survivorOK && survivor.Stage == 5 && len(s.closures) == 5 && schemaErr == nil && schema.Anchor == survivor.Anchor && schema.Targets == survivor.Targets && schema.ReferenceScope == survivor.ReferenceScope && schema.OldGuard == survivor.OldGuard && schema.Locality == survivor.Locality
			}
			if err != nil || kind != "boolean" || !ok || !boolean || operation.Outcome != "verified" || s.frozenArtifact != "" || !exactAssembly {
				return errors.New("invalid frozen schema verification")
			}
			s.frozenArtifact = operation.Inputs[0]
			return nil
		}
		return errors.New("unsupported verify input or phase")
	case "terminal":
		if s.pendingAttach >= 0 || len(s.pendingCompare) != 0 || len(s.afterEvidence) != 0 {
			return errors.New("terminal precedes application evidence attachment")
		}
		if len(operation.Inputs) != 1 || !oneOfString(objectWireVersion(objects[operation.Inputs[0]]), "transform-closure/v1", "transform-schema/v1", "transform-store-boundary/v1") {
			return errors.New("terminal input is not a closure, schema, or store boundary")
		}
		if len(s.programs) == 4 && !s.batchVerified {
			return errors.New("terminal precedes exact program-batch verification")
		}
		if oneOfString(s.policy, string(PositiveLGG), string(ConcreteReplay)) && !s.batchVerified {
			return errors.New("acquisition-backed baseline lacks batch verification")
		}
		if s.policy == string(NousRefine) || s.policy == string(NoEqualityGuard) {
			switch operation.Outcome {
			case "completed":
				if len(s.closures) != 5 {
					return errors.New("completed production acquisition lacks five stage closures")
				}
				if s.frozenArtifact == "" || objectWireVersion(objects[operation.Inputs[0]]) != "transform-store-boundary/v1" {
					return errors.New("production terminal lacks frozen artifact and Store boundary")
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

func (s *transformLifecycleState) validateApplicationCertificate(sequence int, operation TransformOperation, objects map[string][]byte) (TransformOperation, error) {
	var application []json.RawMessage
	if json.Unmarshal(objects[operation.Outputs[0]], &application) != nil || len(application) != 3 {
		return TransformOperation{}, errors.New("invalid application certificate container")
	}
	var certificate []json.RawMessage
	if json.Unmarshal(application[2], &certificate) != nil || len(certificate) != 12 {
		return TransformOperation{}, errors.New("invalid application certificate")
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
			return TransformOperation{}, errors.New("application certificate field")
		}
	}
	if version != "transform-certificate/v1" || schemaDigest != operation.Inputs[1] || inputDigest != operation.Inputs[0] || first < 0 || last != sequence+1 || first > sequence {
		return TransformOperation{}, errors.New("application certificate event range mismatch")
	}
	forest, forestErr := transformschema.ParseForest(objects[operation.Inputs[0]])
	schema, schemaErr := transformschema.ParseSchema(objects[operation.Inputs[1]])
	result, applyErr := schema.Apply(forest)
	if forestErr != nil || schemaErr != nil || applyErr != nil {
		return TransformOperation{}, errors.New("application certificate inputs do not replay")
	}
	trace := append(slices.Clone(s.history[first:]), operation)
	if len(trace) < len(forest.Nodes)+2 {
		return TransformOperation{}, errors.New("application certificate trace is incomplete")
	}
	seenNodes := map[int]bool{}
	var tracedGuards []bool
	var tracedEdits []string
	for index, actual := range trace {
		if actual.Phase != operation.Phase || index < len(forest.Nodes) && actual.Operation != "node" || index == len(trace)-1 && actual.Operation != "schema-application" || index != len(trace)-1 && oneOfString(actual.Operation, "schema-application", "replay-application", "evidence-link", "terminal") {
			return TransformOperation{}, errors.New("application certificate trace shape mismatch")
		}
		if actual.Operation == "node" {
			_, idValue, err := decodeTransformAtom(objects[actual.Inputs[1]])
			id, ok := jsonInteger(idValue)
			if err != nil || !ok || seenNodes[id] {
				return TransformOperation{}, errors.New("application certificate node coverage mismatch")
			}
			seenNodes[id] = true
		}
		if actual.Operation == "schema-predicate" {
			kind, value, err := decodeTransformAtom(objects[actual.Outputs[0]])
			boolean, ok := value.(bool)
			if err != nil || kind != "boolean" || !ok {
				return TransformOperation{}, errors.New("application certificate guard mismatch")
			}
			tracedGuards = append(tracedGuards, boolean)
			selectorKind, selector, _ := decodeTransformAtom(objects[actual.Inputs[2]])
			if selectorKind == "selector" && selector == "edit-no-op" {
				tracedEdits = append(tracedEdits, actual.Inputs[3])
			}
		}
	}
	if len(seenNodes) != len(forest.Nodes) || !slices.Equal(guards, tracedGuards) || !slices.Equal(editDigests, tracedEdits) {
		return TransformOperation{}, errors.New("application certificate evidence summary mismatch")
	}
	wantRequest, wantDefinition, wantReferences, wantEdits := expectedCertificateBindings(forest, schema)
	wantOutput := ""
	if result.Output != nil {
		output, _ := result.Output.CanonicalJSON()
		wantOutput = digestBytes(output)
	}
	if requestID != wantRequest || definitionID != wantDefinition || !slices.Equal(referenceIDs, wantReferences) || !slices.Equal(editDigests, wantEdits) || outputDigest != wantOutput || terminal != result.Terminal || operation.Outcome != result.Terminal {
		return TransformOperation{}, fmt.Errorf("application certificate semantic fields mismatch: binding=%d/%d refs=%v edits=%v output=%s terminal=%s want=%d/%d refs=%v edits=%v output=%s terminal=%s", requestID, definitionID, referenceIDs, editDigests, outputDigest, terminal, wantRequest, wantDefinition, wantReferences, wantEdits, wantOutput, result.Terminal)
	}
	_, expectedEvents, err := transformbaseline.ApplySchemaMeteredAt(objects[operation.Inputs[0]], objects[operation.Inputs[1]], operation.Phase, first)
	if err != nil || len(expectedEvents) != len(trace)+1 {
		return TransformOperation{}, errors.New("application certificate does not cover the exact reconstructed trace")
	}
	for index, actual := range trace {
		expected := lifecycleOperationFromBaseline(expectedEvents[index])
		if !equalLifecycleOperation(actual, expected) {
			return TransformOperation{}, fmt.Errorf("application certificate trace differs at event %d: got %s/%s want %s/%s", index, actual.Operation, actual.Outcome, expected.Operation, expected.Outcome)
		}
	}
	return lifecycleOperationFromBaseline(expectedEvents[len(expectedEvents)-1]), nil
}

func lifecycleOperationFromBaseline(event transformbaseline.Event) TransformOperation {
	inputs := make([]string, len(event.Inputs))
	for index, value := range event.Inputs {
		inputs[index] = digestBytes(value)
	}
	outputs := make([]string, len(event.Outputs))
	for index, value := range event.Outputs {
		outputs[index] = digestBytes(value)
	}
	return TransformOperation{event.Operation, event.Phase, inputs, outputs, event.Outcome, event.Category}
}

func equalLifecycleOperation(left, right TransformOperation) bool {
	return left.Operation == right.Operation && left.Phase == right.Phase && left.Outcome == right.Outcome && left.Category == right.Category && slices.Equal(left.Inputs, right.Inputs) && slices.Equal(left.Outputs, right.Outputs)
}

func equalApplicationEvidence(actual, baseline TransformOperation) bool {
	return actual.Operation == "evidence-link" && baseline.Operation == "evidence-link" && actual.Phase == baseline.Phase && actual.Outcome == baseline.Outcome && actual.Category == baseline.Category && len(actual.Inputs) == 3 && len(baseline.Inputs) == 1 && actual.Inputs[0] == baseline.Inputs[0] && len(actual.Outputs) == 1
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
		partial := s.partials[alternative]
		wantStatus := "counterexample"
		if matched {
			wantStatus = "survivor"
		} else if s.policy == string(NoEqualityGuard) && stage == "old-guard" && partial.OldGuard == "equals-from" {
			wantStatus = "ablated-ineligible"
		} else if stage == "scope" && partial.Targets == "definition" && partial.ReferenceScope == "global" || stage == "old-guard" && partial.Targets == "definition" && partial.OldGuard == "equals-from" {
			wantStatus = "redundant-noncanonical"
		}
		if status != wantStatus {
			return fmt.Errorf("closure alternative status %q, want %q", status, wantStatus)
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
