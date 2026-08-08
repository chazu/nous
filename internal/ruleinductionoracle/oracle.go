// Package ruleinductionoracle independently checks the bounded rule-induction
// experiment. It intentionally does not import the production vocabulary.
package ruleinductionoracle

import "fmt"

const size = 8

type Relation [size * size]bool

func (r *Relation) Add(x, y int) bool {
	if x < 0 || x >= size || y < 0 || y >= size || r[x*size+y] {
		return false
	}
	r[x*size+y] = true
	return true
}

func (r Relation) Has(x, y int) bool {
	return x >= 0 && x < size && y >= 0 && y < size && r[x*size+y]
}

func (r Relation) Signature() string {
	b := make([]byte, size*size)
	for i, present := range r {
		b[i] = '0'
		if present {
			b[i] = '1'
		}
	}
	return string(b)
}

type Clause struct {
	Recursive  bool
	Background int
}

type Definition [2]Clause

type JointTheory struct {
	Stage1 Definition
	Stage2 Definition
	Shared bool
}

func Definitions() []Definition {
	result := make([]Definition, 0, 15)
	for first := 0; first < 6; first++ {
		for second := first + 1; second < 6; second++ {
			result = append(result, Definition{
				{Recursive: first >= 3, Background: first % 3},
				{Recursive: second >= 3, Background: second % 3},
			})
		}
	}
	return result
}

// JointTheories independently enumerates the 225 factored task-local theory
// pairs plus the 15 single-definition shared-invention theories.
func JointTheories() []JointTheory {
	definitions := Definitions()
	result := make([]JointTheory, 0, 240)
	for _, first := range definitions {
		for _, second := range definitions {
			result = append(result, JointTheory{Stage1: first, Stage2: second})
		}
	}
	for _, definition := range definitions {
		result = append(result, JointTheory{Stage1: definition, Stage2: definition, Shared: true})
	}
	return result
}

func Code(definition Definition) string {
	code := func(clause Clause) int {
		if clause.Recursive {
			return 3 + clause.Background
		}
		return clause.Background
	}
	first, second := code(definition[0]), code(definition[1])
	if second < first {
		first, second = second, first
	}
	return fmt.Sprintf("%d%d", first, second)
}

// Evaluate uses repeated full scans, deliberately unlike production's work
// list. The finite monotone relation reaches the same least fixed point.
func Evaluate(definition Definition, background [3]Relation) Relation {
	var result Relation
	for {
		changed := false
		for _, clause := range definition {
			if !clause.Recursive {
				for x := 0; x < size; x++ {
					for y := 0; y < size; y++ {
						if background[clause.Background].Has(x, y) && result.Add(x, y) {
							changed = true
						}
					}
				}
				continue
			}
			for x := 0; x < size; x++ {
				for z := 0; z < size; z++ {
					if !background[clause.Background].Has(x, z) {
						continue
					}
					for y := 0; y < size; y++ {
						if result.Has(z, y) && result.Add(x, y) {
							changed = true
						}
					}
				}
			}
		}
		if !changed {
			return result
		}
	}
}

type Example struct {
	X, Y     int
	Positive bool
}

func Exact(relation Relation, examples []Example) bool {
	for _, example := range examples {
		if relation.Has(example.X, example.Y) != example.Positive {
			return false
		}
	}
	return true
}

func ExactCodes(background [3]Relation, examples []Example) []string {
	result := []string{}
	for _, definition := range Definitions() {
		if Exact(Evaluate(definition, background), examples) {
			result = append(result, Code(definition))
		}
	}
	return result
}

// StructurallySubsumes independently checks the v1 metarule theories. Fixed
// X/Y links and the sole Z variable make two normalized two-clause theories
// mutually subsume only when their clause sets are identical.
func StructurallySubsumes(general, specific Definition) bool {
	return Code(general) == Code(specific)
}

func ConstraintSound(direction string, failed, candidate Definition, background [3]Relation) bool {
	structural := false
	switch direction {
	case "too-general":
		structural = StructurallySubsumes(candidate, failed)
	case "too-specific":
		structural = StructurallySubsumes(failed, candidate)
	default:
		return false
	}
	if !structural {
		return true
	}
	failedRelation, candidateRelation := Evaluate(failed, background), Evaluate(candidate, background)
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			if direction == "too-general" && failedRelation.Has(x, y) && !candidateRelation.Has(x, y) {
				return false
			}
			if direction == "too-specific" && candidateRelation.Has(x, y) && !failedRelation.Has(x, y) {
				return false
			}
		}
	}
	return true
}
