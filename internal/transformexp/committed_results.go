package transformexp

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chazu/nous/internal/transformfixturecore"
)

type committedHeldoutResult struct {
	Token, Terminal, OutputDigest string
	Work                          int64
}

type committedHeldoutScore struct {
	Bits              string
	FalseApplications int
	NonmatchingWork   int64
}

// reconstructHeldoutResults derives the result commitment from the admitted
// application objects and charged events. Report fields are deliberately not
// inputs to this function.
func reconstructHeldoutResults(raw []byte, objects map[string][]byte, heldoutBytes []byte) ([]byte, error) {
	heldout, err := transformfixturecore.ParseHeldout(heldoutBytes)
	if err != nil {
		return nil, err
	}
	type observed struct {
		terminal, output string
		work             int64
	}
	var observations []observed
	var pending int64
	awaitingAttachment := false
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, EventByteCap+1), EventByteCap+1)
	for scanner.Scan() {
		event, err := parseTransformEvent(scanner.Bytes())
		if err != nil {
			return nil, err
		}
		operationBytes := objects[event.Object]
		operation, err := parseTransformOperation(operationBytes)
		if err != nil {
			return nil, err
		}
		if operation.Phase != "heldout" {
			continue
		}
		pending += lifecycleCharges[operation.Category]
		if operation.Operation == "schema-application" || operation.Operation == "replay-application" {
			terminal, output, err := decodeCommittedApplication(operation, objects)
			if err != nil {
				return nil, err
			}
			observations = append(observations, observed{terminal: terminal, output: output})
			awaitingAttachment = true
			continue
		}
		if awaitingAttachment {
			if operation.Operation != "evidence-link" {
				return nil, errors.New("heldout application lacks immediate evidence link")
			}
			observations[len(observations)-1].work = pending
			pending = 0
			awaitingAttachment = false
		}
	}
	if err := scanner.Err(); err != nil || awaitingAttachment || pending != 0 {
		return nil, errors.New("heldout application sequence mismatch")
	}
	if len(observations) == 0 {
		return nil, nil
	}
	if len(observations) != len(heldout.Cases) {
		return nil, errors.New("heldout application count mismatch")
	}
	rows := make([]any, len(observations))
	for index, observation := range observations {
		rows[index] = []any{heldout.Cases[index].Token, observation.terminal, observation.output, observation.work}
	}
	return mustJSON([]any{"transform-heldout-results/v1", rows}), nil
}

func decodeCommittedApplication(operation TransformOperation, objects map[string][]byte) (string, string, error) {
	data := objects[operation.Outputs[0]]
	var wire []json.RawMessage
	if json.Unmarshal(data, &wire) != nil {
		return "", "", errors.New("invalid application object")
	}
	var result []json.RawMessage
	if operation.Operation == "schema-application" {
		if len(wire) != 3 {
			return "", "", errors.New("invalid schema application object")
		}
		var version string
		if json.Unmarshal(wire[0], &version) != nil || version != "transform-schema-application/v1" {
			return "", "", errors.New("invalid schema application version")
		}
		if json.Unmarshal(wire[1], &result) != nil {
			return "", "", errors.New("invalid schema application result")
		}
	} else {
		result = wire
	}
	if len(result) != 3 {
		return "", "", errors.New("invalid application result")
	}
	var version, terminal, output string
	if json.Unmarshal(result[0], &version) != nil || version != "transform-result/v1" || json.Unmarshal(result[1], &terminal) != nil || json.Unmarshal(result[2], &output) != nil || output != "" && !digestString(output) {
		return "", "", errors.New("invalid application result fields")
	}
	return terminal, output, nil
}

func reconstructArtifactDigest(raw []byte, objects map[string][]byte, policy Policy, terminal string) (string, error) {
	if terminal != "completed" {
		return "", nil
	}
	wantOperation := "schema-application"
	if policy == ConcreteReplay {
		wantOperation = "replay-application"
	}
	artifact := ""
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, EventByteCap+1), EventByteCap+1)
	for scanner.Scan() {
		event, err := parseTransformEvent(scanner.Bytes())
		if err != nil {
			return "", err
		}
		operation, err := parseTransformOperation(objects[event.Object])
		if err != nil {
			return "", err
		}
		if operation.Phase != "heldout" || operation.Operation != wantOperation {
			continue
		}
		candidate := operation.Inputs[1]
		version := "transform-schema/v1"
		if policy == ConcreteReplay {
			version = "transform-program-batch/v1"
		}
		if !objectVersion(objects[candidate], version) {
			return "", errors.New("heldout application names invalid frozen artifact")
		}
		if artifact == "" {
			artifact = candidate
		} else if artifact != candidate {
			return "", errors.New("heldout applications changed frozen artifact")
		}
	}
	if err := scanner.Err(); err != nil || artifact == "" {
		return "", errors.New("completed policy lacks heldout frozen artifact")
	}
	if policy == ConcreteReplay {
		return "", nil
	}
	return artifact, nil
}

func scoreCommittedHeldout(results, scorerBytes []byte, policyTerminal string) (committedHeldoutScore, error) {
	if policyTerminal != "completed" {
		if len(results) != 0 {
			return committedHeldoutScore{}, errors.New("noncompleted policy has heldout results")
		}
		return committedHeldoutScore{Bits: "00", NonmatchingWork: LifecycleWorkCap}, nil
	}
	var resultWire []json.RawMessage
	var version string
	var rows [][]json.RawMessage
	if json.Unmarshal(results, &resultWire) != nil || len(resultWire) != 2 || json.Unmarshal(resultWire[0], &version) != nil || version != "transform-heldout-results/v1" || json.Unmarshal(resultWire[1], &rows) != nil || len(rows) != 8 {
		return committedHeldoutScore{}, errors.New("invalid heldout results wire")
	}
	var scorerWire []json.RawMessage
	var expectedRows [][]json.RawMessage
	if json.Unmarshal(scorerBytes, &scorerWire) != nil || len(scorerWire) != 6 || json.Unmarshal(scorerWire[0], &version) != nil || version != "transform-scorer-curriculum/v1" || json.Unmarshal(scorerWire[5], &expectedRows) != nil || len(expectedRows) != 8 {
		return committedHeldoutScore{}, errors.New("invalid scorer evidence")
	}
	type expected struct{ terminal, output string }
	truth := map[string]expected{}
	for _, row := range expectedRows {
		var token, terminal string
		if len(row) != 3 || json.Unmarshal(row[0], &token) != nil || json.Unmarshal(row[1], &terminal) != nil {
			return committedHeldoutScore{}, errors.New("invalid scorer expectation")
		}
		output := ""
		if string(row[2]) != "null" {
			var value any
			if json.Unmarshal(row[2], &value) != nil {
				return committedHeldoutScore{}, errors.New("invalid scorer output")
			}
			output = digestBytes(mustJSON(value))
		}
		if _, duplicate := truth[token]; duplicate {
			return committedHeldoutScore{}, errors.New("duplicate scorer token")
		}
		truth[token] = expected{terminal, output}
	}
	var bits byte
	result := committedHeldoutScore{}
	previous := ""
	for index, row := range rows {
		var token, terminal, output string
		var work int64
		if len(row) != 4 || json.Unmarshal(row[0], &token) != nil || json.Unmarshal(row[1], &terminal) != nil || json.Unmarshal(row[2], &output) != nil || json.Unmarshal(row[3], &work) != nil || token <= previous || work <= 0 {
			return committedHeldoutScore{}, errors.New("invalid heldout result row")
		}
		want, ok := truth[token]
		if !ok {
			return committedHeldoutScore{}, fmt.Errorf("heldout token %q is not scored", token)
		}
		correct := want.terminal == "applied" && terminal == "applied" && output == want.output || want.terminal == "abstain" && strings.HasPrefix(terminal, "abstain/")
		if correct {
			bits |= 1 << index
		}
		if want.terminal == "abstain" {
			result.NonmatchingWork += work
			if terminal == "applied" {
				result.FalseApplications++
			}
		}
		previous = token
	}
	result.Bits = hex.EncodeToString([]byte{bits})
	return result, nil
}
