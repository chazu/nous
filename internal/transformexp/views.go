package transformexp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/transformfixturecore"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

// policyCurriculum is the complete policy-visible view. It deliberately has
// no family, generator seed, accepted attempt, latent schema, or expected
// output field.
type policyCurriculum struct {
	Panel            string
	PanelCommitment  string
	Training         []byte
	HeldoutDigest    string
	PolicyTokens     map[Policy]string
	PolicyRandomness map[Policy][2]uint64
}

// scorerCurriculum is decoded only by orchestration after policy execution.
type scorerCurriculum struct {
	Family          int
	SeedCommitment  string
	AcceptedAttempt int
	Latent          []byte
	Expected        []expectedCase
}

func policyQueueBytes(c curriculum) []byte {
	rows := make([]any, len(empiricalPolicies))
	for index, policy := range empiricalPolicies {
		randomness := c.PolicyRandomness[policy]
		rows[index] = []any{policy, c.PolicyTokens[policy], fmt.Sprintf("%016x", randomness[0]), fmt.Sprintf("%016x", randomness[1])}
	}
	return mustJSON([]any{"transform-policy-queue/v1", rows})
}

func policyQueueBytesFromView(c policyCurriculum) []byte {
	rows := make([]any, len(empiricalPolicies))
	for index, policy := range empiricalPolicies {
		randomness := c.PolicyRandomness[policy]
		rows[index] = []any{policy, c.PolicyTokens[policy], fmt.Sprintf("%016x", randomness[0]), fmt.Sprintf("%016x", randomness[1])}
	}
	return mustJSON([]any{"transform-policy-queue/v1", rows})
}

func scorerFixtureBytes(c curriculum) ([]byte, error) {
	var latent any
	if err := json.Unmarshal(c.Latent, &latent); err != nil {
		return nil, err
	}
	expected := make([]any, len(c.Expected))
	for index, value := range c.Expected {
		var output any
		if len(value.Output) != 0 {
			if err := json.Unmarshal(value.Output, &output); err != nil {
				return nil, err
			}
		}
		expected[index] = []any{value.Token, value.Terminal, output}
	}
	return mustJSON([]any{"transform-scorer-curriculum/v1", c.Family, c.SeedCommitment, c.AcceptedAttempt, latent, expected}), nil
}

func decodePolicyView(c curriculum) (policyCurriculum, error) {
	training, err := transformfixturecore.ParseTraining(bytes.Clone(c.Training))
	if err != nil {
		return policyCurriculum{}, err
	}
	trainingBytes, err := training.CanonicalJSON()
	if err != nil {
		return policyCurriculum{}, err
	}
	queue := policyQueueBytes(c)
	var wire []json.RawMessage
	if json.Unmarshal(queue, &wire) != nil || len(wire) != 2 {
		return policyCurriculum{}, fmt.Errorf("invalid policy queue")
	}
	var version string
	var rows [][]json.RawMessage
	if json.Unmarshal(wire[0], &version) != nil || version != "transform-policy-queue/v1" || json.Unmarshal(wire[1], &rows) != nil || len(rows) != len(empiricalPolicies) {
		return policyCurriculum{}, fmt.Errorf("invalid policy queue wire")
	}
	view := policyCurriculum{Panel: c.Panel, PanelCommitment: c.PanelCommitment, Training: trainingBytes, HeldoutDigest: digestBytes(c.Heldout), PolicyTokens: map[Policy]string{}, PolicyRandomness: map[Policy][2]uint64{}}
	for index, row := range rows {
		var policy Policy
		var token, first, second string
		if len(row) != 4 || json.Unmarshal(row[0], &policy) != nil || policy != empiricalPolicies[index] || json.Unmarshal(row[1], &token) != nil || json.Unmarshal(row[2], &first) != nil || json.Unmarshal(row[3], &second) != nil || len(token) != 16 {
			return policyCurriculum{}, fmt.Errorf("invalid policy queue row %d", index)
		}
		firstBytes, firstErr := hex.DecodeString(first)
		secondBytes, secondErr := hex.DecodeString(second)
		if firstErr != nil || secondErr != nil || len(firstBytes) != 8 || len(secondBytes) != 8 {
			return policyCurriculum{}, fmt.Errorf("invalid policy randomness")
		}
		view.PolicyTokens[policy] = token
		view.PolicyRandomness[policy] = [2]uint64{binary.BigEndian.Uint64(firstBytes), binary.BigEndian.Uint64(secondBytes)}
	}
	return view, nil
}

// decodeHeldoutInputs is the release boundary: callers invoke it only after a
// policy has frozen its schema or concrete replay batch.
func decodeHeldoutInputs(c curriculum) ([]byte, error) {
	heldout, err := transformfixturecore.ParseHeldout(bytes.Clone(c.Heldout))
	if err != nil {
		return nil, err
	}
	return heldout.CanonicalJSON()
}

func decodeScorerView(c curriculum) (scorerCurriculum, error) {
	encoded, err := scorerFixtureBytes(c)
	if err != nil {
		return scorerCurriculum{}, err
	}
	var wire []json.RawMessage
	if json.Unmarshal(encoded, &wire) != nil || len(wire) != 6 {
		return scorerCurriculum{}, fmt.Errorf("invalid scorer wire")
	}
	var version string
	view := scorerCurriculum{}
	var expectedRows [][]json.RawMessage
	if json.Unmarshal(wire[0], &version) != nil || version != "transform-scorer-curriculum/v1" || json.Unmarshal(wire[1], &view.Family) != nil || json.Unmarshal(wire[2], &view.SeedCommitment) != nil || json.Unmarshal(wire[3], &view.AcceptedAttempt) != nil || json.Unmarshal(wire[5], &expectedRows) != nil {
		return scorerCurriculum{}, fmt.Errorf("invalid scorer values")
	}
	latentBytes := bytes.Clone(wire[4])
	if _, err := transformschema.ParseSchema(latentBytes); err != nil {
		return scorerCurriculum{}, err
	}
	view.Latent = latentBytes
	for _, row := range expectedRows {
		var item expectedCase
		if len(row) != 3 || json.Unmarshal(row[0], &item.Token) != nil || json.Unmarshal(row[1], &item.Terminal) != nil {
			return scorerCurriculum{}, fmt.Errorf("invalid scorer expectation")
		}
		if string(row[2]) != "null" {
			item.Output = bytes.Clone(row[2])
		}
		view.Expected = append(view.Expected, item)
	}
	if !slices.IsSortedFunc(view.Expected, func(a, b expectedCase) int { return bytes.Compare([]byte(a.Token), []byte(b.Token)) }) || !bytes.Equal(encoded, mustJSON([]any{"transform-scorer-curriculum/v1", view.Family, view.SeedCommitment, view.AcceptedAttempt, json.RawMessage(view.Latent), scorerExpectedWire(view.Expected)})) {
		return scorerCurriculum{}, fmt.Errorf("noncanonical scorer")
	}
	return view, nil
}

func scorerExpectedWire(values []expectedCase) []any {
	rows := make([]any, len(values))
	for index, value := range values {
		var output any
		if len(value.Output) != 0 {
			_ = json.Unmarshal(value.Output, &output)
		}
		rows[index] = []any{value.Token, value.Terminal, output}
	}
	return rows
}
