// Package nogoods implements the bounded, pure semantics used by the
// constraint/nogood vocabulary. It deliberately contains no search, role
// enumeration, learning, engine, fixture, or oracle code.
package nogoods

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"
)

const (
	ProblemVersion      = "finite-neq-csp/v1"
	SchemaVersion       = "blocked-pair/v1"
	GuardVersion        = "blocked-pair-guard/v1"
	MaxColors           = 4
	MaxVariables        = 8
	MaxEdges            = 18
	MaxAliasBytes       = 128
	FullMask       Mask = 7
)

var ErrInvalid = errors.New("invalid finite inequality CSP")

type Variable struct {
	Alias  string `json:"alias"`
	Domain []int  `json:"domain"`
}

type Edge struct {
	Left  int `json:"left"`
	Right int `json:"right"`
}

type Literal struct {
	Variable int `json:"variable"`
	Color    int `json:"color"`
}

type Problem struct {
	Version      string     `json:"version"`
	ColorAliases []string   `json:"color_aliases"`
	Variables    []Variable `json:"variables"`
	Edges        []Edge     `json:"edges"`
	Assignment   []Literal  `json:"assignment"`
}

// ParseProblem decodes canonical public JSON and rejects rather than repairs
// malformed or non-canonical input.
func ParseProblem(data []byte) (Problem, error) {
	var p Problem
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return Problem{}, fmt.Errorf("%w: decode: %v", ErrInvalid, err)
	}
	if decoder.More() {
		return Problem{}, fmt.Errorf("%w: trailing JSON", ErrInvalid)
	}
	if err := p.Validate(); err != nil {
		return Problem{}, err
	}
	encoded, err := json.Marshal(p)
	if err != nil || !slices.Equal(encoded, data) {
		return Problem{}, fmt.Errorf("%w: encoding is not canonical", ErrInvalid)
	}
	return p, nil
}

// CanonicalJSON returns the sole accepted byte encoding.
func (p Problem) CanonicalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p)
}

func (p Problem) Validate() error {
	if p.Version != ProblemVersion || len(p.ColorAliases) < 1 || len(p.ColorAliases) > MaxColors || len(p.Variables) < 1 || len(p.Variables) > MaxVariables || len(p.Edges) > MaxEdges {
		return ErrInvalid
	}
	seenAliases := map[string]bool{}
	for _, alias := range p.ColorAliases {
		if !validAlias(alias) || seenAliases[alias] {
			return ErrInvalid
		}
		seenAliases[alias] = true
	}
	for _, variable := range p.Variables {
		if !validAlias(variable.Alias) || seenAliases[variable.Alias] || len(variable.Domain) == 0 {
			return ErrInvalid
		}
		seenAliases[variable.Alias] = true
		previous := -1
		for _, color := range variable.Domain {
			if color < 0 || color >= len(p.ColorAliases) || color <= previous {
				return ErrInvalid
			}
			previous = color
		}
	}
	previousEdge := Edge{Left: -1, Right: -1}
	for i, edge := range p.Edges {
		if edge.Left < 0 || edge.Right >= len(p.Variables) || edge.Left >= edge.Right {
			return ErrInvalid
		}
		if i > 0 && !edgeLess(previousEdge, edge) {
			return ErrInvalid
		}
		previousEdge = edge
	}
	previousVariable := -1
	for _, literal := range p.Assignment {
		if literal.Variable < 0 || literal.Variable >= len(p.Variables) || literal.Variable <= previousVariable || !p.DomainContains(literal.Variable, literal.Color) {
			return ErrInvalid
		}
		previousVariable = literal.Variable
	}
	if !p.AssignmentConsistent(p.Assignment) {
		return ErrInvalid
	}
	return nil
}

func validAlias(value string) bool {
	return value != "" && len(value) <= MaxAliasBytes && utf8.ValidString(value)
}

func edgeLess(a, b Edge) bool {
	return a.Left < b.Left || a.Left == b.Left && a.Right < b.Right
}

func (p Problem) DomainContains(variable, color int) bool {
	if variable < 0 || variable >= len(p.Variables) {
		return false
	}
	return slices.Contains(p.Variables[variable].Domain, color)
}

func (p Problem) EdgePresent(left, right int) bool {
	if left > right {
		left, right = right, left
	}
	_, found := slices.BinarySearchFunc(p.Edges, Edge{Left: left, Right: right}, func(a, b Edge) int {
		if edgeLess(a, b) {
			return -1
		}
		if edgeLess(b, a) {
			return 1
		}
		return 0
	})
	return found
}

func (p Problem) AssignmentConsistent(assignment []Literal) bool {
	colors := make(map[int]int, len(assignment))
	for _, literal := range assignment {
		if !p.DomainContains(literal.Variable, literal.Color) {
			return false
		}
		if _, duplicate := colors[literal.Variable]; duplicate {
			return false
		}
		colors[literal.Variable] = literal.Color
	}
	for _, edge := range p.Edges {
		left, lok := colors[edge.Left]
		right, rok := colors[edge.Right]
		if lok && rok && left == right {
			return false
		}
	}
	return true
}

// Extend adds one literal and keeps assignment descriptor-ordered.
func (p Problem) Extend(literal Literal) (Problem, error) {
	if !p.DomainContains(literal.Variable, literal.Color) {
		return Problem{}, ErrInvalid
	}
	if slices.ContainsFunc(p.Assignment, func(existing Literal) bool { return existing.Variable == literal.Variable }) {
		return Problem{}, ErrInvalid
	}
	assignment := append(slices.Clone(p.Assignment), literal)
	slices.SortFunc(assignment, func(a, b Literal) int { return a.Variable - b.Variable })
	if !p.AssignmentConsistent(assignment) {
		return Problem{}, ErrInvalid
	}
	p.Assignment = assignment
	return p, nil
}

type Violation struct {
	Kind  string `json:"kind"`
	Left  int    `json:"left"`
	Right int    `json:"right"`
	Color int    `json:"color"`
}

// Evaluate checks one explicit assignment. It does not enumerate alternatives.
func (p Problem) Evaluate(assignment []Literal) ([]Violation, error) {
	colors := make(map[int]int, len(assignment))
	var violations []Violation
	for _, literal := range assignment {
		if literal.Variable < 0 || literal.Variable >= len(p.Variables) || !p.DomainContains(literal.Variable, literal.Color) {
			violations = append(violations, Violation{Kind: "domain", Left: literal.Variable, Right: -1, Color: literal.Color})
			continue
		}
		if _, duplicate := colors[literal.Variable]; duplicate {
			return nil, ErrInvalid
		}
		colors[literal.Variable] = literal.Color
	}
	for _, edge := range p.Edges {
		left, lok := colors[edge.Left]
		right, rok := colors[edge.Right]
		if lok && rok && left == right {
			violations = append(violations, Violation{Kind: "inequality", Left: edge.Left, Right: edge.Right, Color: left})
		}
	}
	return violations, nil
}

// SemanticKey excludes display aliases and binds descriptor positions.
func (p Problem) SemanticKey() (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	type semanticVariable struct {
		Domain []int `json:"domain"`
	}
	material := struct {
		Version    string             `json:"version"`
		Colors     int                `json:"colors"`
		Variables  []semanticVariable `json:"variables"`
		Edges      []Edge             `json:"edges"`
		Assignment []Literal          `json:"assignment"`
	}{Version: p.Version, Colors: len(p.ColorAliases), Edges: p.Edges, Assignment: p.Assignment}
	for _, variable := range p.Variables {
		material.Variables = append(material.Variables, semanticVariable{Domain: variable.Domain})
	}
	encoded, _ := json.Marshal(material)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

type Mask uint8

func (m Mask) Valid() bool { return m <= FullMask }

func RefineMask(mask Mask, bit int) (Mask, error) {
	if !mask.Valid() || bit < 0 || bit > 2 || mask&(1<<bit) != 0 {
		return 0, ErrInvalid
	}
	return mask | 1<<bit, nil
}

type Binding struct {
	Anchor  int `json:"anchor"`
	X       int `json:"x"`
	Y       int `json:"y"`
	Blocked int `json:"blocked"`
	Escape  int `json:"escape"`
	Only    int `json:"only"`
}

func GuardMatches(p Problem, decision Literal, b Binding) bool {
	if b.Anchor < 0 || b.Anchor >= len(p.Variables) || b.X < 0 || b.X >= len(p.Variables) || b.Y < 0 || b.Y >= len(p.Variables) || b.Anchor == b.X || b.Anchor == b.Y || b.X >= b.Y {
		return false
	}
	if decision != (Literal{Variable: b.Anchor, Color: b.Blocked}) || b.Blocked == b.Escape || b.Blocked == b.Only || b.Escape == b.Only {
		return false
	}
	return slices.Equal(p.Variables[b.Anchor].Domain, sortedPair(b.Blocked, b.Escape)) &&
		slices.Equal(p.Variables[b.X].Domain, sortedPair(b.Blocked, b.Only)) &&
		slices.Equal(p.Variables[b.Y].Domain, sortedPair(b.Blocked, b.Only))
}

func sortedPair(a, b int) []int {
	if a > b {
		a, b = b, a
	}
	return []int{a, b}
}

func MaskMatches(p Problem, mask Mask, b Binding) bool {
	if !mask.Valid() {
		return false
	}
	pairs := [3][2]int{{b.Anchor, b.X}, {b.Anchor, b.Y}, {b.X, b.Y}}
	for bit, pair := range pairs {
		if mask&(1<<bit) != 0 && !p.EdgePresent(pair[0], pair[1]) {
			return false
		}
	}
	return true
}

type Completion struct {
	XColor int `json:"x_color"`
	YColor int `json:"y_color"`
}

// EvaluateCompletion checks one explicit role completion and returns whether
// it conflicts with a required edge or domain.
func EvaluateCompletion(p Problem, mask Mask, b Binding, completion Completion) (bool, error) {
	if !mask.Valid() || b.X < 0 || b.Y >= len(p.Variables) {
		return false, ErrInvalid
	}
	if !p.DomainContains(b.X, completion.XColor) || !p.DomainContains(b.Y, completion.YColor) {
		return true, nil
	}
	values := [3]int{b.Blocked, completion.XColor, completion.YColor}
	for bit, pair := range [3][2]int{{0, 1}, {0, 2}, {1, 2}} {
		if mask&(1<<bit) != 0 && values[pair[0]] == values[pair[1]] {
			return true, nil
		}
	}
	return false, nil
}

type CertificateRecord struct {
	SchemaVersion string     `json:"schema_version"`
	Mask          Mask       `json:"mask"`
	Binding       Binding    `json:"binding"`
	Decision      Literal    `json:"decision"`
	Completion    Completion `json:"completion"`
	Conflict      bool       `json:"conflict"`
}

// ValidateCertificateRecord validates one already-materialized record. It does
// not assert that a collection of records is exhaustive.
func ValidateCertificateRecord(p Problem, record CertificateRecord) error {
	if record.SchemaVersion != SchemaVersion || !GuardMatches(p, record.Decision, record.Binding) || !MaskMatches(p, record.Mask, record.Binding) {
		return ErrInvalid
	}
	conflict, err := EvaluateCompletion(p, record.Mask, record.Binding, record.Completion)
	if err != nil || conflict != record.Conflict {
		return ErrInvalid
	}
	return nil
}
