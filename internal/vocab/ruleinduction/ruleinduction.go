// Package ruleinduction implements the bounded Horn semantics used by the
// relational rule-induction vocabulary. It is deliberately independent of the
// unit store, DSL, engine, and experiment harness.
package ruleinduction

import (
	"errors"
	"fmt"
	"sort"
)

const (
	ConstantCount   = 8
	PredicateCount  = 3
	GroundPairCount = ConstantCount * ConstantCount
)

type Relation [GroundPairCount]bool

func PairIndex(x, y int) (int, bool) {
	if x < 0 || x >= ConstantCount || y < 0 || y >= ConstantCount {
		return 0, false
	}
	return x*ConstantCount + y, true
}

func (r *Relation) Add(x, y int) bool {
	index, ok := PairIndex(x, y)
	if !ok || r[index] {
		return false
	}
	r[index] = true
	return true
}

func (r Relation) Has(x, y int) bool {
	index, ok := PairIndex(x, y)
	return ok && r[index]
}

func (r Relation) Count() int {
	count := 0
	for _, present := range r {
		if present {
			count++
		}
	}
	return count
}

func (r Relation) Signature() string {
	encoded := make([]byte, GroundPairCount)
	for index, present := range r {
		encoded[index] = '0'
		if present {
			encoded[index] = '1'
		}
	}
	return string(encoded)
}

type ClauseKind uint8

const (
	Identity ClauseKind = iota
	TailRec
)

type Clause struct {
	Kind       ClauseKind
	Background int
}

func (c Clause) Valid() bool {
	return (c.Kind == Identity || c.Kind == TailRec) && c.Background >= 0 && c.Background < PredicateCount
}

func (c Clause) Code() int { return int(c.Kind)*PredicateCount + c.Background }

type Definition struct {
	Clauses [2]Clause
}

var ErrInvalidDefinition = errors.New("invalid rule-induction definition")

func Normalize(def Definition) (Definition, error) {
	if !def.Clauses[0].Valid() || !def.Clauses[1].Valid() {
		return Definition{}, ErrInvalidDefinition
	}
	if def.Clauses[0].Code() == def.Clauses[1].Code() {
		return Definition{}, ErrInvalidDefinition
	}
	if def.Clauses[1].Code() < def.Clauses[0].Code() {
		def.Clauses[0], def.Clauses[1] = def.Clauses[1], def.Clauses[0]
	}
	return def, nil
}

func (d Definition) Code() (string, error) {
	normalized, err := Normalize(d)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d%d", normalized.Clauses[0].Code(), normalized.Clauses[1].Code()), nil
}

func EnumerateDefinitions() []Definition {
	definitions := make([]Definition, 0, 15)
	for first := 0; first < 6; first++ {
		for second := first + 1; second < 6; second++ {
			definitions = append(definitions, Definition{Clauses: [2]Clause{
				{Kind: ClauseKind(first / PredicateCount), Background: first % PredicateCount},
				{Kind: ClauseKind(second / PredicateCount), Background: second % PredicateCount},
			}})
		}
	}
	return definitions
}

type Work struct {
	Substitutions     int `json:"substitutions"`
	BodyLookups       int `json:"body_lookups"`
	InsertionAttempts int `json:"insertion_attempts"`
	NovelInserts      int `json:"novel_inserts"`
	Normalization     int `json:"normalization"`
	ExampleLookups    int `json:"example_lookups"`
}

func (w Work) FixedPointTotal() int {
	return w.Substitutions + w.BodyLookups + w.InsertionAttempts + w.NovelInserts + w.Normalization + w.ExampleLookups
}

type pair struct{ x, y int }

// Evaluate computes the least fixed point of one explicit definition using the
// preregistered deterministic work-list algorithm.
func Evaluate(def Definition, background [PredicateCount]Relation) (Relation, Work, error) {
	normalized, err := Normalize(def)
	if err != nil {
		return Relation{}, Work{}, err
	}
	work := Work{Normalization: 16}
	var result Relation
	queue := make([]pair, 0, GroundPairCount)
	insert := func(x, y int) {
		work.InsertionAttempts++
		if result.Add(x, y) {
			work.NovelInserts++
			queue = append(queue, pair{x: x, y: y})
		}
	}

	for _, clause := range normalized.Clauses {
		if clause.Kind != Identity {
			continue
		}
		for x := 0; x < ConstantCount; x++ {
			for y := 0; y < ConstantCount; y++ {
				work.Substitutions++
				work.BodyLookups++
				if background[clause.Background].Has(x, y) {
					insert(x, y)
				}
			}
		}
	}

	for cursor := 0; cursor < len(queue); cursor++ {
		current := queue[cursor]
		for _, clause := range normalized.Clauses {
			if clause.Kind != TailRec {
				continue
			}
			for x := 0; x < ConstantCount; x++ {
				work.Substitutions++
				work.BodyLookups++
				if background[clause.Background].Has(x, current.x) {
					insert(x, current.y)
				}
			}
		}
	}
	return result, work, nil
}

type Example struct {
	X, Y     int
	Positive bool
}

func Score(relation Relation, examples []Example, work *Work) (exact bool, falsePositive, falseNegative int) {
	exact = true
	for _, example := range examples {
		work.ExampleLookups++
		actual := relation.Has(example.X, example.Y)
		if actual == example.Positive {
			continue
		}
		exact = false
		if actual {
			falsePositive++
		} else {
			falseNegative++
		}
	}
	return exact, falsePositive, falseNegative
}

// StructurallySubsumes implements the v1 theory comparison. Variables and
// links are fixed by the metarules, so theta-subsumption reduces to matching an
// explicit normalized clause under the at-most-three legal Z images.
func StructurallySubsumes(general, specific Definition) (bool, int, error) {
	g, err := Normalize(general)
	if err != nil {
		return false, 0, err
	}
	s, err := Normalize(specific)
	if err != nil {
		return false, 0, err
	}
	work := 0
	for _, wanted := range s.Clauses {
		matched := false
		for _, offered := range g.Clauses {
			for zImage := 0; zImage < 3; zImage++ {
				work++ // theta substitution
				work += 4
				if zImage == 2 && offered == wanted {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false, work, nil
		}
	}
	return true, work, nil
}

type Partial struct {
	// Fields are clause0 kind/background then clause1 kind/background. -1 is
	// the sole unbound representation.
	Fields [4]int
}

func RootPartial() Partial { return Partial{Fields: [4]int{-1, -1, -1, -1}} }

func (p Partial) Bound() int {
	for index, field := range p.Fields {
		if field == -1 {
			return index
		}
	}
	return len(p.Fields)
}

func (p Partial) Valid() bool {
	bound := p.Bound()
	for index, field := range p.Fields {
		if index >= bound {
			if field != -1 {
				return false
			}
			continue
		}
		if index%2 == 0 {
			if field < 0 || field > 1 {
				return false
			}
		} else if field < 0 || field >= PredicateCount {
			return false
		}
	}
	return true
}

func RefineOne(p Partial) ([]Partial, error) {
	if !p.Valid() {
		return nil, errors.New("invalid partial definition")
	}
	hole := p.Bound()
	if hole == len(p.Fields) {
		return nil, nil
	}
	choices := 2
	if hole%2 == 1 {
		choices = PredicateCount
	}
	children := make([]Partial, 0, choices)
	for choice := 0; choice < choices; choice++ {
		child := p
		child.Fields[hole] = choice
		children = append(children, child)
	}
	return children, nil
}

func (p Partial) Definition() (Definition, error) {
	if !p.Valid() || p.Bound() != len(p.Fields) {
		return Definition{}, ErrInvalidDefinition
	}
	return Normalize(Definition{Clauses: [2]Clause{
		{Kind: ClauseKind(p.Fields[0]), Background: p.Fields[1]},
		{Kind: ClauseKind(p.Fields[2]), Background: p.Fields[3]},
	}})
}

func CanonicalCodes(definitions []Definition) ([]string, error) {
	codes := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		code, err := definition.Code()
		if err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes, nil
}
