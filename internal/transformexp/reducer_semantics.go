package transformexp

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/chazu/nous/internal/transformbaseline"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

func validateTransformSemantics(operation TransformOperation, objects map[string][]byte) error {
	inputs := valuesForDigests(operation.Inputs, objects)
	outputs := valuesForDigests(operation.Outputs, objects)
	switch operation.Operation {
	case "node", "parent", "target":
		return validateFactSemantics(operation, inputs, outputs)
	case "compare":
		left, _, err := decodeTransformAtom(inputs[0])
		if err != nil {
			return err
		}
		right, _, err := decodeTransformAtom(inputs[1])
		if err != nil {
			return err
		}
		if left != right {
			if operation.Outcome != "invalid-input" || len(outputs) != 0 {
				return errors.New("different-kind atoms were compared: " + left + " vs " + right)
			}
			return nil
		}
		equal := bytes.Equal(inputs[0], inputs[1])
		return validateBooleanResult(operation, outputs, equal, map[bool]string{true: "true", false: "false"}, left, right)
	case "candidate-allocate":
		if operation.Outcome == "allocated" && (len(outputs) != 1 || !bytes.Equal(inputs[0], outputs[0])) {
			return errors.New("allocated candidate output differs from input")
		}
	case "refine":
		partial, err := transformschema.ParsePartial(inputs[0])
		if err != nil {
			return err
		}
		kind, value, err := decodeTransformAtom(inputs[1])
		choice, ok := value.(string)
		if err != nil || kind != "enum" || !ok {
			return errors.New("refine choice is not an enum atom")
		}
		next, refineErr := partial.Refine(choice)
		if refineErr != nil {
			if operation.Outcome != "rejected" && operation.Outcome != "invalid-input" {
				return errors.New("invalid refinement was accepted")
			}
			return nil
		}
		encoded, _ := next.CanonicalJSON()
		if operation.Outcome != "refined" || len(outputs) != 1 || !bytes.Equal(encoded, outputs[0]) {
			return errors.New("refinement result mismatch")
		}
	case "edit-validate":
		forest, forestErr := transformschema.ParseForest(inputs[0])
		edit, editErr := decodeTransformEdit(inputs[1])
		status := "invalid-input"
		if forestErr == nil && editErr == nil {
			status = "valid"
			for _, node := range forest.Nodes {
				if node.ID == edit.Target && node.Value == edit.Value {
					status = "no-op"
				}
			}
			if _, err := (transformschema.Program{Edits: []transformschema.Edit{edit}}).Apply(forest); err != nil && status != "no-op" {
				status = "invalid-input"
			}
		}
		want, _ := json.Marshal([]any{"transform-edit-status/v1", status, digestBytes(inputs[1])})
		if operation.Outcome != status || len(outputs) != 1 || !bytes.Equal(want, outputs[0]) {
			return errors.New("edit validation mismatch")
		}
	case "edit-apply":
		forest, forestErr := transformschema.ParseForest(inputs[0])
		edit, editErr := decodeTransformEdit(inputs[1])
		var next transformschema.Forest
		applyErr := errors.New("invalid edit application input")
		if forestErr == nil && editErr == nil {
			next, applyErr = (transformschema.Program{Edits: []transformschema.Edit{edit}}).Apply(forest)
		}
		if forestErr != nil || editErr != nil || applyErr != nil {
			if operation.Outcome != "invalid-input" || len(outputs) != 0 {
				return errors.New("invalid edit application was accepted")
			}
			return nil
		}
		want, _ := next.CanonicalJSON()
		if operation.Outcome != "applied" || len(outputs) != 1 || !bytes.Equal(want, outputs[0]) {
			return errors.New("edit application mismatch")
		}
	case "schema-predicate":
		return validateSchemaPredicate(operation, inputs, outputs)
	case "output-compare":
		left, leftErr := transformschema.ParseForest(inputs[0])
		right, rightErr := transformschema.ParseForest(inputs[1])
		kind, value, atomErr := decodeTransformAtom(inputs[2])
		id, idOK := jsonInteger(value)
		if leftErr != nil || rightErr != nil || atomErr != nil || kind != "id" || !idOK || len(left.Nodes) != len(right.Nodes) || id < 0 || id >= len(left.Nodes) {
			if operation.Outcome != "invalid-input" {
				return errors.New("invalid node comparison was accepted")
			}
			return nil
		}
		var leftNode, rightNode *transformschema.Node
		for index := range left.Nodes {
			if left.Nodes[index].ID == id {
				leftNode = &left.Nodes[index]
			}
			if right.Nodes[index].ID == id {
				rightNode = &right.Nodes[index]
			}
		}
		if leftNode == nil || rightNode == nil {
			if operation.Outcome != "invalid-input" {
				return errors.New("missing node comparison was accepted")
			}
			return nil
		}
		leftBytes, _ := json.Marshal(leftNode)
		rightBytes, _ := json.Marshal(rightNode)
		return validateBooleanResult(operation, outputs, bytes.Equal(leftBytes, rightBytes), map[bool]string{true: "equal", false: "different"}, nil, nil)
	case "schema-application":
		return validateSchemaApplication(operation, inputs, outputs)
	case "replay-application":
		application, err := transformbaseline.Replay(inputs[1], "", inputs[0])
		if err != nil {
			return err
		}
		outputDigest := ""
		if len(application.Output) != 0 {
			outputDigest = digestBytes(application.Output)
		}
		want, _ := json.Marshal([]any{"transform-result/v1", application.Terminal, outputDigest})
		if operation.Outcome != application.Terminal || len(outputs) != 1 || !bytes.Equal(want, outputs[0]) {
			return errors.New("replay result mismatch")
		}
	case "evidence-link":
		return validateEvidenceAttempt(operation, objects, outputs)
	case "canonicalize":
		_, semanticErr := transformSemanticKind(inputs[0])
		if operation.Outcome == "invalid-input" {
			if semanticErr == nil || len(outputs) != 0 {
				return errors.New("canonicalize rejected a valid semantic value")
			}
			return nil
		}
		if semanticErr != nil || len(outputs) != 1 || !bytes.Equal(inputs[0], outputs[0]) {
			return errors.New("canonicalize changed or accepted an invalid value")
		}
	case "hash":
		if operation.Outcome == "invalid-input" {
			if _, semanticErr := transformSemanticKind(inputs[0]); semanticErr == nil || len(outputs) != 0 {
				return errors.New("hash rejected a valid semantic value")
			}
			return nil
		}
		if len(outputs) != 1 {
			return errors.New("hash result arity")
		}
		kind, value, err := decodeTransformAtom(outputs[0])
		if err != nil || kind != "digest" || value != digestBytes(inputs[0]) {
			return errors.New("hash result mismatch")
		}
	}
	return nil
}

func valuesForDigests(digests []string, objects map[string][]byte) [][]byte {
	values := make([][]byte, len(digests))
	for index, digest := range digests {
		values[index] = objects[digest]
	}
	return values
}

func validateFactSemantics(operation TransformOperation, inputs, outputs [][]byte) error {
	forest, forestErr := transformschema.ParseForest(inputs[0])
	kind, value, atomErr := decodeTransformAtom(inputs[1])
	id, idOK := jsonInteger(value)
	if forestErr != nil || atomErr != nil || kind != "id" || !idOK {
		return errors.New("fact input mismatch")
	}
	var node *transformschema.Node
	for index := range forest.Nodes {
		if forest.Nodes[index].ID == id {
			node = &forest.Nodes[index]
		}
	}
	var want []byte
	wantOutcome := "ok"
	switch operation.Operation {
	case "node":
		if node == nil {
			wantOutcome = "invalid-input"
		} else {
			want, _ = json.Marshal([]any{"transform-node-facts/v1", node.Kind, node.Value, node.From, node.To})
		}
	case "parent":
		if node == nil {
			wantOutcome = "invalid-input"
		} else if node.Parent < 0 {
			wantOutcome = "absent"
		} else {
			want, _ = json.Marshal([]any{"transform-parent-facts/v1", node.Parent, node.Key})
		}
	case "target":
		target := -1
		if node == nil {
			wantOutcome = "invalid-input"
		} else if node.Target < 0 {
			wantOutcome = "absent"
		} else {
			target = node.Target
			want, _ = json.Marshal([]any{"transform-atom/v1", "id", target})
		}
	}
	if operation.Outcome != wantOutcome {
		return errors.New("fact outcome mismatch")
	}
	if want == nil {
		if len(outputs) != 0 {
			return errors.New("failed fact operation produced output")
		}
		return nil
	}
	if len(outputs) != 1 || !bytes.Equal(want, outputs[0]) {
		return errors.New("fact result mismatch")
	}
	return nil
}

func validateSchemaPredicate(operation TransformOperation, inputs, outputs [][]byte) error {
	forest, err := transformschema.ParseForest(inputs[0])
	if err != nil {
		return validateInvalidPredicate(operation, outputs)
	}
	schema, err := transformschema.ParseSchema(inputs[1])
	if err != nil {
		return validateInvalidPredicate(operation, outputs)
	}
	selectorKind, selectorValue, err := decodeTransformAtom(inputs[2])
	selector, ok := selectorValue.(string)
	if err != nil || selectorKind != "selector" || !ok {
		return validateInvalidPredicate(operation, outputs)
	}
	subjectKind, subjectValue, subjectErr := decodeTransformAtom(inputs[3])
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
	var candidates []transformschema.Node
	if len(requests) == 1 {
		request := requests[0]
		for _, definition := range definitions {
			if schema.Anchor == "request-target" && definition.ID == request.Target || schema.Anchor == "from-value" && definition.Value == request.From || schema.Anchor == "first-local" && definition.Parent == request.Parent {
				candidates = append(candidates, definition)
				if schema.Anchor == "first-local" {
					break
				}
			}
		}
	}
	result := false
	switch selector {
	case "request-count":
		count, ok := jsonInteger(subjectValue)
		result = subjectErr == nil && subjectKind == "count" && ok && count == len(requests) && len(requests) == 1
	case "anchor-candidate":
		n, ok := jsonInteger(subjectValue)
		if subjectKind == "count" {
			result = subjectErr == nil && ok && n == len(candidates) && len(candidates) == 1
		} else if subjectKind == "id" {
			result = subjectErr == nil && ok && len(candidates) == 1 && n == candidates[0].ID
		}
	case "anchor-locality":
		id, ok := jsonInteger(subjectValue)
		definition := byID[id]
		result = subjectErr == nil && subjectKind == "id" && ok && len(requests) == 1 && definition.Kind == "definition" && (schema.Locality == "none" || definition.Parent == requests[0].Parent)
	case "reference-target", "reference-scope", "reference-old-guard":
		id, ok := jsonInteger(subjectValue)
		reference := byID[id]
		if subjectErr == nil && subjectKind == "id" && ok && len(requests) == 1 && len(candidates) == 1 && reference.Kind == "reference" {
			switch selector {
			case "reference-target":
				result = reference.Target == candidates[0].ID
			case "reference-scope":
				result = schema.ReferenceScope == "global" || reference.Parent == requests[0].Parent
			case "reference-old-guard":
				result = schema.OldGuard == "any" || reference.Value == requests[0].From
			}
		}
	case "expansion-bound":
		count, ok := jsonInteger(subjectValue)
		result = subjectErr == nil && subjectKind == "count" && ok && count >= 1 && count <= transformschema.MaxEdits
	case "edit-no-op":
		edit, editErr := decodeTransformEdit(inputs[3])
		node := byID[edit.Target]
		result = editErr == nil && (node.Kind == "definition" || node.Kind == "reference") && node.Value != edit.Value
	default:
		return validateInvalidPredicate(operation, outputs)
	}
	if subjectErr != nil && selector != "edit-no-op" {
		return validateInvalidPredicate(operation, outputs)
	}
	return validateBooleanResult(operation, outputs, result, map[bool]string{true: "true", false: "false"}, nil, nil)
}

func validateInvalidPredicate(operation TransformOperation, outputs [][]byte) error {
	if operation.Outcome != "invalid-input" || len(outputs) != 0 {
		return errors.New("invalid schema predicate was accepted")
	}
	return nil
}

func validateSchemaApplication(operation TransformOperation, inputs, outputs [][]byte) error {
	forest, err := transformschema.ParseForest(inputs[0])
	if err != nil {
		return err
	}
	schema, err := transformschema.ParseSchema(inputs[1])
	if err != nil {
		return err
	}
	result, err := schema.Apply(forest)
	if err != nil && result.Terminal != "invalid-input" {
		return err
	}
	outputDigest := ""
	if result.Output != nil {
		encoded, _ := result.Output.CanonicalJSON()
		outputDigest = digestBytes(encoded)
	}
	var application []any
	if json.Unmarshal(outputs[0], &application) != nil || len(application) != 3 || application[0] != "transform-schema-application/v1" {
		return errors.New("schema application wire")
	}
	resultWire, _ := json.Marshal(application[1])
	wantResult, _ := json.Marshal([]any{"transform-result/v1", result.Terminal, outputDigest})
	if operation.Outcome != result.Terminal || !bytes.Equal(resultWire, wantResult) {
		return errors.New("schema application result mismatch")
	}
	return nil
}

func validateEvidenceAttempt(operation TransformOperation, objects map[string][]byte, outputs [][]byte) error {
	var row []any
	if json.Unmarshal(outputs[0], &row) != nil || len(row) != 7 || row[0] != "transform-evidence-attempt/v1" || row[1] != operation.Outcome || row[4] != operation.Inputs[0] || row[5] != operation.Inputs[1] || row[6] != operation.Inputs[2] {
		return errors.New("evidence attempt mismatch")
	}
	encoded, _ := json.Marshal(row[3])
	kind, err := transformSemanticKind(encoded)
	claimedKind, ok := row[2].(string)
	if err != nil || !ok || claimedKind != kind || digestBytes(encoded) != operation.Inputs[0] || len(objects[operation.Inputs[2]]) == 0 {
		return errors.New("evidence attempt changed value or kind")
	}
	return nil
}

func validateBooleanResult(operation TransformOperation, outputs [][]byte, result bool, outcomes map[bool]string, _ any, _ any) error {
	if len(outputs) != 1 {
		return errors.New("boolean operation has no result")
	}
	kind, value, err := decodeTransformAtom(outputs[0])
	actual, ok := value.(bool)
	if err != nil || kind != "boolean" || !ok || actual != result || operation.Outcome != outcomes[result] {
		return errors.New("boolean operation result mismatch")
	}
	return nil
}

func decodeTransformAtom(data []byte) (string, any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var row []any
	if decoder.Decode(&row) != nil || len(row) != 3 || row[0] != "transform-atom/v1" {
		return "", nil, errors.New("atom wire")
	}
	encoded, _ := json.Marshal(row)
	if !bytes.Equal(encoded, data) {
		return "", nil, errors.New("noncanonical atom")
	}
	kind, ok := row[1].(string)
	if !ok {
		return "", nil, errors.New("atom kind")
	}
	if err := validateTransformAtom(kind, row[2]); err != nil {
		return "", nil, err
	}
	return kind, row[2], nil
}

func validateTransformAtom(kind string, value any) error {
	switch kind {
	case "id", "count":
		n, ok := jsonInteger(value)
		if !ok || kind == "id" && (n < -1 || n >= transformschema.MaxNodes) || kind == "count" && (n < 0 || n > transformschema.MaxNodes) {
			return errors.New("bounded integer atom")
		}
	case "id-set":
		rows, ok := value.([]any)
		last := -1
		if !ok || len(rows) > 6 {
			return errors.New("id-set atom")
		}
		for _, row := range rows {
			id, ok := jsonInteger(row)
			if !ok || id < 0 || id >= transformschema.MaxNodes || id <= last {
				return errors.New("noncanonical id-set atom")
			}
			last = id
		}
	case "scoped-id", "scoped-id-set":
		row, ok := value.([]any)
		if !ok || len(row) != 2 {
			return errors.New("scoped id atom")
		}
		digest, ok := row[0].(string)
		if !ok || !digestString(digest) {
			return errors.New("scoped id forest digest")
		}
		if kind == "scoped-id" {
			n, ok := jsonInteger(row[1])
			if !ok || n < -1 || n >= transformschema.MaxNodes {
				return errors.New("scoped id value")
			}
			break
		}
		ids, ok := row[1].([]any)
		last := -1
		if !ok || len(ids) > 6 {
			return errors.New("scoped id-set value")
		}
		for _, item := range ids {
			id, ok := jsonInteger(item)
			if !ok || id < 0 || id >= transformschema.MaxNodes || id <= last {
				return errors.New("noncanonical scoped id-set")
			}
			last = id
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("boolean atom")
		}
	case "digest":
		text, ok := value.(string)
		if !ok || !digestString(text) {
			return errors.New("digest atom")
		}
	case "selector":
		text, ok := value.(string)
		if !ok || !oneOfString(text, "request-count", "anchor-candidate", "anchor-locality", "reference-target", "reference-scope", "reference-old-guard", "expansion-bound", "edit-no-op") {
			return errors.New("selector atom")
		}
	case "kind", "key":
		if _, ok := value.(string); !ok {
			return errors.New("string atom")
		}
	case "enum":
		if _, ok := value.(string); !ok && value != nil {
			return errors.New("enum atom")
		}
	case "scalar":
		if _, stringOK := value.(string); !stringOK {
			if _, listOK := value.([]any); !listOK {
				return errors.New("scalar atom")
			}
		}
	default:
		return errors.New("unknown atom kind")
	}
	return nil
}

func decodeTransformEdit(data []byte) (transformschema.Edit, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var row []any
	if decoder.Decode(&row) != nil || len(row) != 3 || row[0] != "set-value/v1" {
		return transformschema.Edit{}, errors.New("edit wire")
	}
	target, ok := jsonInteger(row[1])
	value, valueOK := row[2].(string)
	if !ok || !valueOK {
		return transformschema.Edit{}, errors.New("edit fields")
	}
	return transformschema.Edit{Target: target, Value: value}, nil
}

func jsonInteger(value any) (int, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	integer, err := number.Int64()
	return int(integer), err == nil && int64(int(integer)) == integer
}
