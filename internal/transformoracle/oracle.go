// Package transformoracle is an independent semantic implementation used only
// after a policy terminal or by the committed competence suite. It deliberately
// does not import the production vocabulary.
package transformoracle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

var ErrInvalid = errors.New("invalid oracle transformation value")

type node struct {
	id, parent, target         int
	kind, key, value, from, to string
}

type forest struct{ nodes []node }

type Schema struct {
	Anchor, Targets, Scope, Guard, Locality string
}

type Result struct {
	Terminal string
	Output   []byte
	Edits    [][2]any
}

func ApplyProgram(forestBytes, programBytes []byte) ([]byte, error) {
	f, err := parseForest(forestBytes)
	if err != nil {
		return nil, err
	}
	v, err := decode(programBytes)
	if err != nil {
		return nil, err
	}
	row, ok := v.([]any)
	if !ok || len(row) != 2 || row[0] != "concrete-program/v1" {
		return nil, ErrInvalid
	}
	edits, ok := row[1].([]any)
	if !ok || len(edits) < 1 || len(edits) > 4 {
		return nil, ErrInvalid
	}
	positions := map[int]int{}
	for i, n := range f.nodes {
		positions[n.id] = i
	}
	previous := -1
	for _, raw := range edits {
		edit, ok := raw.([]any)
		if !ok || len(edit) != 3 || edit[0] != "set-value/v1" {
			return nil, ErrInvalid
		}
		target, targetOK := integer(edit[1])
		value, valueOK := edit[2].(string)
		position, exists := positions[target]
		if !targetOK || !valueOK || !exists || target <= previous || !lower(value) || !oneOf(f.nodes[position].kind, "definition", "reference") || f.nodes[position].value == value {
			return nil, ErrInvalid
		}
		f.nodes[position].value = value
		previous = target
	}
	canonical, err := json.Marshal(v)
	if err != nil || !bytes.Equal(canonical, programBytes) {
		return nil, ErrInvalid
	}
	return encodeForest(f)
}

func Apply(forestBytes, schemaBytes []byte) (Result, error) {
	f, err := parseForest(forestBytes)
	if err != nil {
		return Result{Terminal: "invalid-input"}, err
	}
	s, err := parseSchema(schemaBytes)
	if err != nil {
		return Result{Terminal: "invalid-input"}, err
	}
	requests := selectNodes(f.nodes, func(n node) bool { return n.kind == "request" })
	if len(requests) != 1 {
		return Result{Terminal: "abstain/request-count"}, nil
	}
	rq := requests[0]
	defs := selectNodes(f.nodes, func(n node) bool { return n.kind == "definition" })
	var matches []node
	for _, d := range defs {
		match := s.Anchor == "request-target" && d.id == rq.target ||
			s.Anchor == "from-value" && d.value == rq.from ||
			s.Anchor == "first-local" && d.parent == rq.parent
		if match {
			matches = append(matches, d)
			if s.Anchor == "first-local" {
				break
			}
		}
	}
	if len(matches) != 1 {
		return Result{Terminal: "abstain/anchor"}, nil
	}
	d := matches[0]
	if s.Locality == "required" && d.parent != rq.parent {
		return Result{Terminal: "abstain/locality"}, nil
	}
	refs := selectNodes(f.nodes, func(n node) bool {
		return n.kind == "reference" && n.target == d.id &&
			(s.Scope == "global" || n.parent == rq.parent) &&
			(s.Guard == "any" || n.value == rq.from)
	})
	var edits [][2]any
	if s.Targets == "definition" || s.Targets == "definition+references" {
		edits = append(edits, [2]any{d.id, rq.to})
	}
	if s.Targets == "references" || s.Targets == "definition+references" {
		for _, ref := range refs {
			edits = append(edits, [2]any{ref.id, rq.to})
		}
	}
	slices.SortFunc(edits, func(a, b [2]any) int { return a[0].(int) - b[0].(int) })
	if len(edits) == 0 || len(edits) > 4 {
		return Result{Terminal: "abstain/expansion"}, nil
	}
	positions := map[int]int{}
	for i, n := range f.nodes {
		positions[n.id] = i
	}
	for _, e := range edits {
		i := positions[e[0].(int)]
		if f.nodes[i].value == e[1].(string) {
			return Result{Terminal: "abstain/no-op"}, nil
		}
		f.nodes[i].value = e[1].(string)
	}
	out, err := encodeForest(f)
	if err != nil {
		return Result{Terminal: "invalid-input"}, err
	}
	return Result{Terminal: "applied", Output: out, Edits: edits}, nil
}

func parseSchema(data []byte) (Schema, error) {
	v, err := decode(data)
	if err != nil {
		return Schema{}, err
	}
	r, ok := v.([]any)
	if !ok || len(r) != 6 || r[0] != "transform-schema/v1" {
		return Schema{}, ErrInvalid
	}
	values := make([]string, 5)
	for i := range values {
		values[i], ok = r[i+1].(string)
		if !ok {
			return Schema{}, ErrInvalid
		}
	}
	s := Schema{values[0], values[1], values[2], values[3], values[4]}
	if !oneOf(s.Anchor, "request-target", "from-value", "first-local") || !oneOf(s.Targets, "definition", "references", "definition+references") || !oneOf(s.Scope, "local", "global") || !oneOf(s.Guard, "equals-from", "any") || !oneOf(s.Locality, "required", "none") {
		return Schema{}, ErrInvalid
	}
	canonical, _ := json.Marshal([]any{"transform-schema/v1", s.Anchor, s.Targets, s.Scope, s.Guard, s.Locality})
	if !bytes.Equal(canonical, data) {
		return Schema{}, ErrInvalid
	}
	return s, nil
}

func parseForest(data []byte) (forest, error) {
	if len(data) > 2048 {
		return forest{}, ErrInvalid
	}
	v, err := decode(data)
	if err != nil {
		return forest{}, err
	}
	o, ok := v.([]any)
	if !ok || len(o) != 2 || o[0] != "typed-reference-forest/v1" {
		return forest{}, ErrInvalid
	}
	rows, ok := o[1].([]any)
	if !ok || len(rows) == 0 || len(rows) > 12 {
		return forest{}, ErrInvalid
	}
	f := forest{nodes: make([]node, len(rows))}
	seen := make([]bool, len(rows))
	for i, raw := range rows {
		r, ok := raw.([]any)
		if !ok || len(r) != 8 {
			return forest{}, ErrInvalid
		}
		id, a := integer(r[0])
		kind, b := r[1].(string)
		parent, c := integer(r[2])
		key, d := r[3].(string)
		value, e := r[4].(string)
		from, g := r[5].(string)
		to, h := r[6].(string)
		target, j := integer(r[7])
		if !(a && b && c && d && e && g && h && j) || id < 0 || id >= len(rows) || seen[id] {
			return forest{}, ErrInvalid
		}
		seen[id] = true
		f.nodes[i] = node{id, parent, target, kind, key, value, from, to}
	}
	slices.SortFunc(f.nodes, func(a, b node) int { return a.id - b.id })
	if err := validateForest(f); err != nil {
		return forest{}, err
	}
	return f, nil
}

func validateForest(f forest) error {
	keys := map[int]map[string]bool{}
	counts := map[string]int{}
	for _, n := range f.nodes {
		counts[n.kind]++
		if !oneOf(n.kind, "group", "request", "definition", "reference", "decoy") {
			return ErrInvalid
		}
		for _, value := range []string{n.key, n.value, n.from, n.to} {
			if value != "" && !lower(value) {
				return ErrInvalid
			}
		}
		if n.kind == "group" {
			if n.parent != -1 || n.key != "" || n.value != "" || n.from != "" || n.to != "" || n.target != -1 {
				return ErrInvalid
			}
			continue
		}
		if n.parent < 0 || n.parent >= len(f.nodes) || f.nodes[n.parent].kind != "group" || n.key == "" {
			return ErrInvalid
		}
		if keys[n.parent] == nil {
			keys[n.parent] = map[string]bool{}
		}
		if keys[n.parent][n.key] {
			return ErrInvalid
		}
		keys[n.parent][n.key] = true
		switch n.kind {
		case "request":
			if n.value != "" || n.from == "" || n.to == "" || !definition(f, n.target) {
				return ErrInvalid
			}
		case "definition", "decoy":
			if n.value == "" || n.from != "" || n.to != "" || n.target != -1 {
				return ErrInvalid
			}
		case "reference":
			if n.value == "" || n.from != "" || n.to != "" || !definition(f, n.target) {
				return ErrInvalid
			}
		}
	}
	if counts["group"] > 2 || counts["request"] > 2 || counts["definition"] > 2 || counts["reference"] > 6 {
		return ErrInvalid
	}
	return nil
}

func encodeForest(f forest) ([]byte, error) {
	if err := validateForest(f); err != nil {
		return nil, err
	}
	rows := make([]any, len(f.nodes))
	for i, n := range f.nodes {
		rows[i] = []any{n.id, n.kind, n.parent, n.key, n.value, n.from, n.to, n.target}
	}
	return json.Marshal([]any{"typed-reference-forest/v1", rows})
}

func decode(data []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, ErrInvalid
	}
	return v, nil
}

func integer(v any) (int, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, err := n.Int64()
	return int(i), err == nil && int64(int(i)) == i
}
func oneOf(v string, values ...string) bool { return slices.Contains(values, v) }
func lower(v string) bool {
	if v == "" || len(v) > 16 {
		return false
	}
	for _, c := range []byte(v) {
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}
func definition(f forest, id int) bool {
	return id >= 0 && id < len(f.nodes) && f.nodes[id].kind == "definition"
}
func selectNodes(values []node, keep func(node) bool) []node {
	var out []node
	for _, v := range values {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}
