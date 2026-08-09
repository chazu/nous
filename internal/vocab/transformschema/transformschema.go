// Package transformschema implements the bounded, pure semantics for the
// transformation-schema vocabulary. It contains no search, fixture, engine,
// experiment, or oracle code.
package transformschema

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

const (
	ForestVersion  = "typed-reference-forest/v1"
	EditVersion    = "set-value/v1"
	ProgramVersion = "concrete-program/v1"
	SchemaVersion  = "transform-schema/v1"
	PartialVersion = "transform-partial/v1"
	MaxNodes       = 12
	MaxEdits       = 4
	MaxForestBytes = 2048
)

var ErrInvalid = errors.New("invalid transformation-schema value")

type Node struct {
	ID     int
	Kind   string
	Parent int
	Key    string
	Value  string
	From   string
	To     string
	Target int
}

type Forest struct{ Nodes []Node }

func ParseForest(data []byte) (Forest, error) {
	if len(data) > MaxForestBytes {
		return Forest{}, ErrInvalid
	}
	v, err := decodeOne(data)
	if err != nil {
		return Forest{}, err
	}
	outer, ok := v.([]any)
	if !ok || len(outer) != 2 || outer[0] != ForestVersion {
		return Forest{}, ErrInvalid
	}
	rows, ok := outer[1].([]any)
	if !ok {
		return Forest{}, ErrInvalid
	}
	f := Forest{Nodes: make([]Node, len(rows))}
	for i, raw := range rows {
		row, ok := raw.([]any)
		if !ok || len(row) != 8 {
			return Forest{}, ErrInvalid
		}
		id, ok0 := exactInt(row[0])
		kind, ok1 := row[1].(string)
		parent, ok2 := exactInt(row[2])
		key, ok3 := row[3].(string)
		value, ok4 := row[4].(string)
		from, ok5 := row[5].(string)
		to, ok6 := row[6].(string)
		target, ok7 := exactInt(row[7])
		if !(ok0 && ok1 && ok2 && ok3 && ok4 && ok5 && ok6 && ok7) {
			return Forest{}, ErrInvalid
		}
		f.Nodes[i] = Node{id, kind, parent, key, value, from, to, target}
	}
	if err := f.Validate(); err != nil {
		return Forest{}, err
	}
	return f, nil
}

func (f Forest) Validate() error {
	if len(f.Nodes) == 0 || len(f.Nodes) > MaxNodes {
		return ErrInvalid
	}
	byID := make([]*Node, len(f.Nodes))
	counts := map[string]int{}
	for i := range f.Nodes {
		n := &f.Nodes[i]
		if n.ID < 0 || n.ID >= len(f.Nodes) || byID[n.ID] != nil || !validToken(n.Kind, 16) {
			return ErrInvalid
		}
		byID[n.ID] = n
		counts[n.Kind]++
		for _, s := range []string{n.Key, n.Value, n.From, n.To} {
			if s != "" && !validToken(s, 16) {
				return ErrInvalid
			}
		}
	}
	if counts["group"] > 2 || counts["request"] > 2 || counts["definition"] > 2 || counts["reference"] > 6 {
		return ErrInvalid
	}
	keys := map[int]map[string]bool{}
	for _, n := range byID {
		switch n.Kind {
		case "group":
			if n.Parent != -1 || n.Key != "" || n.Value != "" || n.From != "" || n.To != "" || n.Target != -1 {
				return ErrInvalid
			}
		case "request":
			if !childOfGroup(n, byID) || n.Key == "" || n.Value != "" || n.From == "" || n.To == "" || !definitionTarget(n.Target, byID) {
				return ErrInvalid
			}
		case "definition":
			if !childOfGroup(n, byID) || n.Key == "" || n.Value == "" || n.From != "" || n.To != "" || n.Target != -1 {
				return ErrInvalid
			}
		case "reference":
			if !childOfGroup(n, byID) || n.Key == "" || n.Value == "" || n.From != "" || n.To != "" || !definitionTarget(n.Target, byID) {
				return ErrInvalid
			}
		case "decoy":
			if !childOfGroup(n, byID) || n.Key == "" || n.Value == "" || n.From != "" || n.To != "" || n.Target != -1 {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
		if n.Kind != "group" {
			if keys[n.Parent] == nil {
				keys[n.Parent] = map[string]bool{}
			}
			if keys[n.Parent][n.Key] {
				return ErrInvalid
			}
			keys[n.Parent][n.Key] = true
		}
	}
	return nil
}

func childOfGroup(n *Node, byID []*Node) bool {
	return n.Parent >= 0 && n.Parent < len(byID) && byID[n.Parent].Kind == "group"
}

func definitionTarget(id int, byID []*Node) bool {
	return id >= 0 && id < len(byID) && byID[id].Kind == "definition"
}

func validToken(s string, max int) bool {
	if s == "" || len(s) > max {
		return false
	}
	for _, c := range []byte(s) {
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

func (f Forest) CanonicalJSON() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	nodes := slices.Clone(f.Nodes)
	slices.SortFunc(nodes, func(a, b Node) int { return a.ID - b.ID })
	rows := make([]any, len(nodes))
	for i, n := range nodes {
		rows[i] = []any{n.ID, n.Kind, n.Parent, n.Key, n.Value, n.From, n.To, n.Target}
	}
	return json.Marshal([]any{ForestVersion, rows})
}

func (f Forest) Digest() (string, error) {
	b, err := f.CanonicalJSON()
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

type Edit struct {
	Target int
	Value  string
}

func (e Edit) wire() []any { return []any{EditVersion, e.Target, e.Value} }

type Program struct{ Edits []Edit }

func ParseProgram(data []byte) (Program, error) {
	v, err := decodeOne(data)
	if err != nil {
		return Program{}, err
	}
	outer, ok := v.([]any)
	if !ok || len(outer) != 2 || outer[0] != ProgramVersion {
		return Program{}, ErrInvalid
	}
	rows, ok := outer[1].([]any)
	if !ok {
		return Program{}, ErrInvalid
	}
	p := Program{Edits: make([]Edit, len(rows))}
	for i, raw := range rows {
		row, ok := raw.([]any)
		if !ok || len(row) != 3 || row[0] != EditVersion {
			return Program{}, ErrInvalid
		}
		target, ok1 := exactInt(row[1])
		value, ok2 := row[2].(string)
		if !ok1 || !ok2 {
			return Program{}, ErrInvalid
		}
		p.Edits[i] = Edit{target, value}
	}
	if err := p.Validate(); err != nil {
		return Program{}, err
	}
	canonical, _ := p.CanonicalJSON()
	if !bytes.Equal(canonical, data) {
		return Program{}, ErrInvalid
	}
	return p, nil
}

func (p Program) Validate() error {
	if len(p.Edits) < 1 || len(p.Edits) > MaxEdits {
		return ErrInvalid
	}
	previous := -1
	for _, e := range p.Edits {
		if e.Target < 0 || e.Target <= previous || !validToken(e.Value, 16) {
			return ErrInvalid
		}
		previous = e.Target
	}
	return nil
}

func (p Program) CanonicalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	rows := make([]any, len(p.Edits))
	for i, e := range p.Edits {
		rows[i] = e.wire()
	}
	return json.Marshal([]any{ProgramVersion, rows})
}

func (p Program) Apply(f Forest) (Forest, error) {
	if err := f.Validate(); err != nil {
		return Forest{}, err
	}
	if err := p.Validate(); err != nil {
		return Forest{}, err
	}
	out := Forest{Nodes: slices.Clone(f.Nodes)}
	byID := make(map[int]*Node, len(out.Nodes))
	for i := range out.Nodes {
		byID[out.Nodes[i].ID] = &out.Nodes[i]
	}
	for _, e := range p.Edits {
		n := byID[e.Target]
		if n == nil || (n.Kind != "definition" && n.Kind != "reference") || n.Value == e.Value {
			return Forest{}, ErrInvalid
		}
		n.Value = e.Value
	}
	return out, out.Validate()
}

type Schema struct {
	Anchor         string
	Targets        string
	ReferenceScope string
	OldGuard       string
	Locality       string
}

var (
	anchors    = []string{"request-target", "from-value", "first-local"}
	targets    = []string{"definition", "references", "definition+references"}
	scopes     = []string{"local", "global"}
	oldGuards  = []string{"equals-from", "any"}
	localities = []string{"required", "none"}
)

func (s Schema) Validate() error {
	if !slices.Contains(anchors, s.Anchor) || !slices.Contains(targets, s.Targets) || !slices.Contains(scopes, s.ReferenceScope) || !slices.Contains(oldGuards, s.OldGuard) || !slices.Contains(localities, s.Locality) {
		return ErrInvalid
	}
	return nil
}

func (s Schema) CanonicalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal([]any{SchemaVersion, s.Anchor, s.Targets, s.ReferenceScope, s.OldGuard, s.Locality})
}

func ParseSchema(data []byte) (Schema, error) {
	v, err := decodeOne(data)
	if err != nil {
		return Schema{}, err
	}
	r, ok := v.([]any)
	if !ok || len(r) != 6 || r[0] != SchemaVersion {
		return Schema{}, ErrInvalid
	}
	s := Schema{}
	var oks [5]bool
	s.Anchor, oks[0] = r[1].(string)
	s.Targets, oks[1] = r[2].(string)
	s.ReferenceScope, oks[2] = r[3].(string)
	s.OldGuard, oks[3] = r[4].(string)
	s.Locality, oks[4] = r[5].(string)
	if slices.Contains(oks[:], false) || s.Validate() != nil {
		return Schema{}, ErrInvalid
	}
	canonical, _ := s.CanonicalJSON()
	if !bytes.Equal(canonical, data) {
		return Schema{}, ErrInvalid
	}
	return s, nil
}

type Certificate struct {
	RequestID    int
	DefinitionID int
	ReferenceIDs []int
	Edits        []Edit
}

type Result struct {
	Terminal    string
	Output      *Forest
	Certificate Certificate
}

func (s Schema) Apply(f Forest) (Result, error) {
	if err := s.Validate(); err != nil {
		return Result{Terminal: "invalid-input"}, err
	}
	if err := f.Validate(); err != nil {
		return Result{Terminal: "invalid-input"}, err
	}
	nodes := slices.Clone(f.Nodes)
	slices.SortFunc(nodes, func(a, b Node) int { return a.ID - b.ID })
	requests := filter(nodes, func(n Node) bool { return n.Kind == "request" })
	if len(requests) != 1 {
		return Result{Terminal: "abstain/request-count"}, nil
	}
	rq := requests[0]
	defs := filter(nodes, func(n Node) bool { return n.Kind == "definition" })
	var candidates []Node
	switch s.Anchor {
	case "request-target":
		candidates = filter(defs, func(n Node) bool { return n.ID == rq.Target })
	case "from-value":
		candidates = filter(defs, func(n Node) bool { return n.Value == rq.From })
	case "first-local":
		candidates = filter(defs, func(n Node) bool { return n.Parent == rq.Parent })
		if len(candidates) > 1 {
			candidates = candidates[:1]
		}
	}
	if len(candidates) != 1 {
		return Result{Terminal: "abstain/anchor"}, nil
	}
	def := candidates[0]
	if s.Locality == "required" && def.Parent != rq.Parent {
		return Result{Terminal: "abstain/locality"}, nil
	}
	refs := filter(nodes, func(n Node) bool {
		if n.Kind != "reference" || n.Target != def.ID {
			return false
		}
		if s.ReferenceScope == "local" && n.Parent != rq.Parent {
			return false
		}
		return s.OldGuard == "any" || n.Value == rq.From
	})
	var edits []Edit
	if s.Targets == "definition" || s.Targets == "definition+references" {
		edits = append(edits, Edit{def.ID, rq.To})
	}
	if s.Targets == "references" || s.Targets == "definition+references" {
		for _, ref := range refs {
			edits = append(edits, Edit{ref.ID, rq.To})
		}
	}
	slices.SortFunc(edits, func(a, b Edit) int { return a.Target - b.Target })
	if len(edits) == 0 || len(edits) > MaxEdits {
		return Result{Terminal: "abstain/expansion"}, nil
	}
	byID := map[int]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	for _, e := range edits {
		if byID[e.Target].Value == e.Value {
			return Result{Terminal: "abstain/no-op"}, nil
		}
	}
	out, err := (Program{Edits: edits}).Apply(f)
	if err != nil {
		return Result{Terminal: "invalid-input"}, err
	}
	return Result{Terminal: "applied", Output: &out, Certificate: Certificate{rq.ID, def.ID, ids(refs), edits}}, nil
}

func filter(values []Node, keep func(Node) bool) []Node {
	out := make([]Node, 0, len(values))
	for _, v := range values {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func ids(nodes []Node) []int {
	out := make([]int, len(nodes))
	for i, n := range nodes {
		out[i] = n.ID
	}
	return out
}

func Schemas() []Schema {
	out := make([]Schema, 0, 72)
	for _, a := range anchors {
		for _, t := range targets {
			for _, sc := range scopes {
				for _, g := range oldGuards {
					for _, l := range localities {
						out = append(out, Schema{a, t, sc, g, l})
					}
				}
			}
		}
	}
	return out
}

type Partial struct {
	Stage          int
	Targets        string
	Anchor         string
	ReferenceScope string
	OldGuard       string
	Locality       string
}

func (p Partial) Refine(value string) (Partial, error) {
	choices := [][]string{targets, anchors, scopes, oldGuards, localities}
	if p.Stage < 0 || p.Stage >= len(choices) || !slices.Contains(choices[p.Stage], value) {
		return Partial{}, ErrInvalid
	}
	n := p
	switch p.Stage {
	case 0:
		n.Targets = value
	case 1:
		n.Anchor = value
	case 2:
		n.ReferenceScope = value
	case 3:
		n.OldGuard = value
	case 4:
		n.Locality = value
	}
	n.Stage++
	return n, nil
}

func (p Partial) CanonicalJSON() ([]byte, error) {
	if p.Stage < 0 || p.Stage > 5 {
		return nil, ErrInvalid
	}
	fields := []string{p.Targets, p.Anchor, p.ReferenceScope, p.OldGuard, p.Locality}
	choices := [][]string{targets, anchors, scopes, oldGuards, localities}
	for i, field := range fields {
		if i < p.Stage {
			if !slices.Contains(choices[i], field) {
				return nil, ErrInvalid
			}
		} else if field != "" {
			return nil, ErrInvalid
		}
	}
	return json.Marshal([]any{PartialVersion, p.Stage, p.Targets, p.Anchor, p.ReferenceScope, p.OldGuard, p.Locality})
}

func decodeOne(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var v any
	if err := decoder.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, ErrInvalid
	}
	return v, nil
}

func exactInt(v any) (int, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, err := n.Int64()
	if err != nil || int64(int(i)) != i {
		return 0, false
	}
	return int(i), true
}
