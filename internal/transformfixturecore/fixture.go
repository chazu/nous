// Package transformfixturecore owns strict, data-only policy fixture wires. It
// has no panel, seed, family, latent-schema, scorer, generator, or oracle API.
package transformfixturecore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"slices"
)

const (
	TrainingVersion = "transform-policy-curriculum/v1"
	HeldoutVersion  = "transform-heldout-inputs/v1"
	BatchVersion    = "transform-program-batch/v1"
)

var ErrInvalid = errors.New("invalid transformation fixture")

type TrainingCase struct {
	Token  string
	Kind   string
	Before []byte
	After  []byte
}

type Training struct {
	ProfileDigest string
	Cases         []TrainingCase
}

type HeldoutCase struct {
	Token  string
	Before []byte
}

type Heldout struct {
	ProfileDigest string
	Cases         []HeldoutCase
}

type ProgramRow struct {
	Token        string
	BeforeDigest string
	Program      []byte
}

type ProgramBatch struct{ Rows []ProgramRow }

func ProfileDigest() string {
	b, _ := json.Marshal([]any{"transform-profile/v1", "typed-reference-forest/v1", "set-scalar-from-request/v1", "anchor-target-scope-old-guard-locality/v1", "transform-lifecycle-events/v2", 12, 4, 72, 48, 12000})
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:])
}

func (t Training) CanonicalJSON() ([]byte, error) {
	if t.ProfileDigest != ProfileDigest() || len(t.Cases) != 8 {
		return nil, ErrInvalid
	}
	cases := slices.Clone(t.Cases)
	slices.SortFunc(cases, func(a, b TrainingCase) int { return bytes.Compare([]byte(a.Token), []byte(b.Token)) })
	rows := make([]any, len(cases))
	previous := ""
	positives, abstentions := 0, 0
	for i, c := range cases {
		if !validToken(c.Token) || c.Token == previous {
			return nil, ErrInvalid
		}
		previous = c.Token
		beforeWire, err := validateWire(c.Before, "typed-reference-forest/v1", 2048)
		if err != nil {
			return nil, ErrInvalid
		}
		var after any
		switch c.Kind {
		case "positive":
			afterWire, err := validateWire(c.After, "typed-reference-forest/v1", 2048)
			if err != nil {
				return nil, ErrInvalid
			}
			after = json.RawMessage(afterWire)
			positives++
		case "abstain":
			if len(c.After) != 0 {
				return nil, ErrInvalid
			}
			after = nil
			abstentions++
		default:
			return nil, ErrInvalid
		}
		rows[i] = []any{c.Token, c.Kind, json.RawMessage(beforeWire), after}
	}
	if positives != 4 || abstentions != 4 {
		return nil, ErrInvalid
	}
	return json.Marshal([]any{TrainingVersion, t.ProfileDigest, rows})
}

func ParseTraining(data []byte) (Training, error) {
	v, err := decode(data)
	if err != nil {
		return Training{}, err
	}
	r, ok := v.([]any)
	if !ok || len(r) != 3 || r[0] != TrainingVersion {
		return Training{}, ErrInvalid
	}
	profile, ok := r[1].(string)
	if !ok {
		return Training{}, ErrInvalid
	}
	rows, ok := r[2].([]any)
	if !ok {
		return Training{}, ErrInvalid
	}
	t := Training{ProfileDigest: profile}
	for _, raw := range rows {
		row, ok := raw.([]any)
		if !ok || len(row) != 4 {
			return Training{}, ErrInvalid
		}
		token, a := row[0].(string)
		kind, b := row[1].(string)
		before, c := json.Marshal(row[2])
		if !(a && b && c == nil) {
			return Training{}, ErrInvalid
		}
		var after []byte
		if row[3] != nil {
			after, err = json.Marshal(row[3])
			if err != nil {
				return Training{}, ErrInvalid
			}
		}
		t.Cases = append(t.Cases, TrainingCase{token, kind, before, after})
	}
	canonical, err := t.CanonicalJSON()
	if err != nil || !bytes.Equal(canonical, data) {
		return Training{}, ErrInvalid
	}
	return t, nil
}

func (h Heldout) CanonicalJSON() ([]byte, error) {
	if h.ProfileDigest != ProfileDigest() || len(h.Cases) != 8 {
		return nil, ErrInvalid
	}
	cases := slices.Clone(h.Cases)
	slices.SortFunc(cases, func(a, b HeldoutCase) int { return bytes.Compare([]byte(a.Token), []byte(b.Token)) })
	rows := make([]any, len(cases))
	previous := ""
	for i, c := range cases {
		if !validToken(c.Token) || c.Token == previous {
			return nil, ErrInvalid
		}
		previous = c.Token
		b, err := validateWire(c.Before, "typed-reference-forest/v1", 2048)
		if err != nil {
			return nil, ErrInvalid
		}
		rows[i] = []any{c.Token, json.RawMessage(b)}
	}
	return json.Marshal([]any{HeldoutVersion, h.ProfileDigest, rows})
}

func ParseHeldout(data []byte) (Heldout, error) {
	v, err := decode(data)
	if err != nil {
		return Heldout{}, err
	}
	r, ok := v.([]any)
	if !ok || len(r) != 3 || r[0] != HeldoutVersion {
		return Heldout{}, ErrInvalid
	}
	profile, ok := r[1].(string)
	if !ok {
		return Heldout{}, ErrInvalid
	}
	rows, ok := r[2].([]any)
	if !ok {
		return Heldout{}, ErrInvalid
	}
	h := Heldout{ProfileDigest: profile}
	for _, raw := range rows {
		row, ok := raw.([]any)
		if !ok || len(row) != 2 {
			return Heldout{}, ErrInvalid
		}
		token, ok := row[0].(string)
		before, marshalErr := json.Marshal(row[1])
		if !ok || marshalErr != nil {
			return Heldout{}, ErrInvalid
		}
		h.Cases = append(h.Cases, HeldoutCase{Token: token, Before: before})
	}
	canonical, err := h.CanonicalJSON()
	if err != nil || !bytes.Equal(canonical, data) {
		return Heldout{}, ErrInvalid
	}
	return h, nil
}

func (b ProgramBatch) CanonicalJSON() ([]byte, error) {
	if len(b.Rows) != 4 {
		return nil, ErrInvalid
	}
	rows := slices.Clone(b.Rows)
	slices.SortFunc(rows, func(a, c ProgramRow) int { return bytes.Compare([]byte(a.Token), []byte(c.Token)) })
	wire := make([]any, len(rows))
	previous := ""
	for i, row := range rows {
		if !validToken(row.Token) || row.Token == previous || !digest(row.BeforeDigest) {
			return nil, ErrInvalid
		}
		previous = row.Token
		p, err := validateWire(row.Program, "concrete-program/v1", 640)
		if err != nil {
			return nil, ErrInvalid
		}
		var program any
		if err := json.Unmarshal(p, &program); err != nil {
			return nil, err
		}
		wire[i] = []any{row.Token, row.BeforeDigest, program}
	}
	encoded, err := json.Marshal([]any{BatchVersion, wire})
	if err != nil || len(encoded) > 1152 {
		return nil, ErrInvalid
	}
	return encoded, nil
}

func ParseProgramBatch(data []byte) (ProgramBatch, error) {
	v, err := decode(data)
	if err != nil {
		return ProgramBatch{}, err
	}
	r, ok := v.([]any)
	if !ok || len(r) != 2 || r[0] != BatchVersion {
		return ProgramBatch{}, ErrInvalid
	}
	rows, ok := r[1].([]any)
	if !ok {
		return ProgramBatch{}, ErrInvalid
	}
	b := ProgramBatch{}
	for _, raw := range rows {
		row, ok := raw.([]any)
		if !ok || len(row) != 3 {
			return ProgramBatch{}, ErrInvalid
		}
		token, a := row[0].(string)
		beforeDigest, c := row[1].(string)
		program, marshalErr := json.Marshal(row[2])
		if !a || !c || marshalErr != nil {
			return ProgramBatch{}, ErrInvalid
		}
		b.Rows = append(b.Rows, ProgramRow{token, beforeDigest, program})
	}
	canonical, err := b.CanonicalJSON()
	if err != nil || !bytes.Equal(canonical, data) {
		return ProgramBatch{}, ErrInvalid
	}
	return b, nil
}

func validToken(s string) bool {
	if len(s) != 16 {
		return false
	}
	for _, c := range []byte(s) {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func digest(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && s == string(bytes.ToLower([]byte(s)))
}

func decode(data []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, ErrInvalid
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, ErrInvalid
	}
	return v, nil
}

func validateWire(data []byte, version string, max int) ([]byte, error) {
	if len(data) == 0 || len(data) > max {
		return nil, ErrInvalid
	}
	v, err := decode(data)
	if err != nil {
		return nil, err
	}
	r, ok := v.([]any)
	if !ok || len(r) != 2 || r[0] != version {
		return nil, ErrInvalid
	}
	canonical, err := json.Marshal(v)
	if err != nil || !bytes.Equal(canonical, data) {
		return nil, ErrInvalid
	}
	return canonical, nil
}
