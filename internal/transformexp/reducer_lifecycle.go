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
	policy         string
	partials       map[string]transformschema.Partial
	allocPhase     map[string]string
	parents        map[string]string
	closures       []string
	survivor       string
	frozenArtifact string
}

func newTransformLifecycleState(policy string) *transformLifecycleState {
	return &transformLifecycleState{policy: policy, partials: map[string]transformschema.Partial{}, allocPhase: map[string]string{}, parents: map[string]string{}}
}

func (s *transformLifecycleState) observe(operation TransformOperation, objects map[string][]byte) error {
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
			s.parents[operation.Outputs[0]] = operation.Inputs[0]
		}
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
		if s.policy == string(NousRefine) || s.policy == string(NoEqualityGuard) {
			if len(s.closures) != 5 {
				return errors.New("production acquisition lacks five stage closures")
			}
			if operation.Outcome == "completed" && (s.frozenArtifact == "" || operation.Inputs[0] != s.frozenArtifact) {
				return errors.New("production terminal does not name frozen artifact")
			}
		}
	}
	return nil
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
		matched := status == "survivor"
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
