// Package ruleinductionexp drives the staged rule-induction experiment through
// production store artifacts and an independent oracle.
package ruleinductionexp

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/chazu/nous/internal/ruleinductionoracle"
)

type Cohort string

const (
	Beneficial Cohort = "beneficial"
	Neutral    Cohort = "neutral"
	Harmful    Cohort = "harmful"
	NoSolution Cohort = "no-solution"
)

type Edge struct{ X, Y int }

type graph struct {
	Relation ruleinductionoracle.Relation
	Edges    []Edge
	Attempt  int
}

func GenerateNoSolution(seed int64) (Fixture, error) {
	fixture, err := Generate("development", seed, Beneficial)
	if err != nil {
		return Fixture{}, err
	}
	fixture.Cohort = NoSolution
	fixture.Target2 = symmetricClosure(fixture.Background[0])
	fixture.HeldTarget2 = symmetricClosure(fixture.HeldOut[0])
	if exact := exactCodes(fixture.Background, fixture.Target2); len(exact) != 0 {
		return Fixture{}, fmt.Errorf("no-solution seed %d has exact definitions %v", seed, exact)
	}
	fixture.Stage2, err = separatingExamples(fixture.Background, fixture.Target2)
	if err != nil {
		return Fixture{}, err
	}
	return fixture, nil
}

func symmetricClosure(input ruleinductionoracle.Relation) ruleinductionoracle.Relation {
	result := input
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			if input.Has(x, y) {
				result.Add(y, x)
			}
		}
	}
	return result
}

func HarmfulSensitivityFixture() Fixture {
	edges := [3][]Edge{
		{{0, 1}, {1, 2}, {1, 3}, {3, 4}, {5, 6}, {6, 7}, {0, 5}, {2, 4}},
		{{7, 6}, {6, 5}, {6, 4}, {4, 3}, {2, 1}, {1, 0}, {7, 2}, {5, 3}},
		{{0, 4}, {0, 5}, {0, 6}, {0, 7}, {1, 4}, {1, 5}, {1, 6}, {1, 7}},
	}
	fixture := Fixture{Panel: "sensitivity", Seed: 0, Cohort: Harmful, Edges: edges, HeldEdges: edges,
		Stage1: []ruleinductionoracle.Example{{X: 0, Y: 1, Positive: true}, {X: 0, Y: 2, Positive: true}, {X: 0, Y: 0}, {X: 1, Y: 0}},
		Stage2: []ruleinductionoracle.Example{{X: 1, Y: 0, Positive: true}, {X: 2, Y: 0, Positive: true}, {X: 0, Y: 0}, {X: 0, Y: 1}},
	}
	for predicate := range edges {
		for _, edge := range edges[predicate] {
			fixture.Background[predicate].Add(edge.X, edge.Y)
			fixture.HeldOut[predicate].Add(edge.X, edge.Y)
		}
	}
	fixture.Target1 = ruleinductionoracle.Evaluate(definition("03"), fixture.Background)
	fixture.Target2 = ruleinductionoracle.Evaluate(definition("14"), fixture.Background)
	fixture.HeldTarget1, fixture.HeldTarget2 = fixture.Target1, fixture.Target2
	for index := range fixture.ConstantAliases {
		fixture.ConstantAliases[index] = fmt.Sprintf("manual-k-%d", index)
	}
	for index := range fixture.PredicateAliases {
		fixture.PredicateAliases[index] = fmt.Sprintf("manual-r-%d", index)
	}
	return fixture
}

type Fixture struct {
	Panel            string
	Seed             int64
	Cohort           Cohort
	Background       [3]ruleinductionoracle.Relation
	HeldOut          [3]ruleinductionoracle.Relation
	Edges            [3][]Edge
	HeldEdges        [3][]Edge
	Attempts         [3]int
	HeldAttempts     [3]int
	Stage1           []ruleinductionoracle.Example
	Stage2           []ruleinductionoracle.Example
	Target1          ruleinductionoracle.Relation
	Target2          ruleinductionoracle.Relation
	HeldTarget1      ruleinductionoracle.Relation
	HeldTarget2      ruleinductionoracle.Relation
	ConstantAliases  [8]string
	PredicateAliases [3]string
}

func CohortForIndex(index int) Cohort {
	switch index % 16 {
	case 10, 11, 12:
		return Neutral
	case 13, 14, 15:
		return Harmful
	default:
		return Beneficial
	}
}

func streamRNG(panel string, seed int64, stream string, attempt int) *rand.Rand {
	material := fmt.Sprintf("closure-pairs/v1|%s|%d|%s|%d", panel, seed, stream, attempt)
	sum := sha256.Sum256([]byte(material))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(sum[0:8]), binary.BigEndian.Uint64(sum[8:16])))
}

func sampledGraph(panel string, seed int64, stream string, minDepth, maxDepth, startAttempt int) (graph, error) {
	all := make([]Edge, 0, 56)
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			if x != y {
				all = append(all, Edge{X: x, Y: y})
			}
		}
	}
	for attempt := startAttempt; attempt < 100; attempt++ {
		rng := streamRNG(panel, seed, stream, attempt)
		pairs := append([]Edge(nil), all...)
		rng.Shuffle(len(pairs), func(i, j int) { pairs[i], pairs[j] = pairs[j], pairs[i] })
		count := 8 + rng.IntN(11)
		edges := pairs[:count]
		var relation ruleinductionoracle.Relation
		outdegree := [8]int{}
		for _, edge := range edges {
			relation.Add(edge.X, edge.Y)
			outdegree[edge.X]++
		}
		branch := false
		for _, degree := range outdegree {
			branch = branch || degree >= 2
		}
		closure := transitiveClosure(relation)
		depth := longestSimplePath(relation)
		if branch && relationCount(closure) < 64 && depth >= minDepth && depth <= maxDepth {
			return graph{Relation: relation, Edges: append([]Edge(nil), edges...), Attempt: attempt}, nil
		}
	}
	return graph{}, fmt.Errorf("%s seed %d stream %s exhausted attempts", panel, seed, stream)
}

func transitiveClosure(input ruleinductionoracle.Relation) ruleinductionoracle.Relation {
	result := input
	for k := 0; k < 8; k++ {
		for x := 0; x < 8; x++ {
			for y := 0; y < 8; y++ {
				if result.Has(x, k) && result.Has(k, y) {
					result.Add(x, y)
				}
			}
		}
	}
	return result
}

func relationCount(relation ruleinductionoracle.Relation) int {
	count := 0
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			if relation.Has(x, y) {
				count++
			}
		}
	}
	return count
}

func longestSimplePath(relation ruleinductionoracle.Relation) int {
	best := 0
	var visit func(int, uint8, int)
	visit = func(node int, seen uint8, depth int) {
		if depth > best {
			best = depth
		}
		for next := 0; next < 8; next++ {
			bit := uint8(1 << next)
			if relation.Has(node, next) && seen&bit == 0 {
				visit(next, seen|bit, depth+1)
			}
		}
	}
	for start := 0; start < 8; start++ {
		visit(start, uint8(1<<start), 0)
	}
	return best
}

func definition(code string) ruleinductionoracle.Definition {
	for _, candidate := range ruleinductionoracle.Definitions() {
		if ruleinductionoracle.Code(candidate) == code {
			return candidate
		}
	}
	panic("unknown definition " + code)
}

func exactCodes(background [3]ruleinductionoracle.Relation, target ruleinductionoracle.Relation) []string {
	var codes []string
	for _, candidate := range ruleinductionoracle.Definitions() {
		if ruleinductionoracle.Evaluate(candidate, background) == target {
			codes = append(codes, ruleinductionoracle.Code(candidate))
		}
	}
	return codes
}

func separatingExamples(background [3]ruleinductionoracle.Relation, target ruleinductionoracle.Relation) ([]ruleinductionoracle.Example, error) {
	all := make([]ruleinductionoracle.Example, 0, 64)
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			all = append(all, ruleinductionoracle.Example{X: x, Y: y, Positive: target.Has(x, y)})
		}
	}
	selected := make([]ruleinductionoracle.Example, 0, 24)
	for _, polarity := range []bool{true, false} {
		for _, example := range all {
			if example.Positive == polarity {
				selected = append(selected, example)
				if countPolarity(selected, polarity) == 2 {
					break
				}
			}
		}
	}
	for _, candidate := range ruleinductionoracle.Definitions() {
		relation := ruleinductionoracle.Evaluate(candidate, background)
		if relation == target || distinguished(relation, selected) {
			continue
		}
		for _, example := range all {
			if relation.Has(example.X, example.Y) != example.Positive {
				selected = append(selected, example)
				break
			}
		}
	}
	if len(selected) > 24 || countPolarity(selected, true) < 2 || countPolarity(selected, false) < 2 {
		return nil, fmt.Errorf("separator has %d examples", len(selected))
	}
	return selected, nil
}

func countPolarity(examples []ruleinductionoracle.Example, polarity bool) int {
	count := 0
	for _, example := range examples {
		if example.Positive == polarity {
			count++
		}
	}
	return count
}

func distinguished(relation ruleinductionoracle.Relation, examples []ruleinductionoracle.Example) bool {
	for _, example := range examples {
		if relation.Has(example.X, example.Y) != example.Positive {
			return true
		}
	}
	return false
}

func Generate(panel string, seed int64, cohort Cohort) (Fixture, error) {
	var lastErr error
	for fixtureAttempt := 0; fixtureAttempt < 100; fixtureAttempt++ {
		fixture, err := generateAt(panel, seed, cohort, fixtureAttempt)
		if err == nil {
			return fixture, nil
		}
		lastErr = err
	}
	return Fixture{}, fmt.Errorf("%s seed %d exhausted fixture attempts: %w", panel, seed, lastErr)
}

func generateAt(panel string, seed int64, cohort Cohort, startAttempt int) (Fixture, error) {
	fixture := Fixture{Panel: panel, Seed: seed, Cohort: cohort}
	constantRNG := streamRNG(panel, seed, "constant-aliases", 0)
	for index := range fixture.ConstantAliases {
		fixture.ConstantAliases[index] = fmt.Sprintf("k-%016x", constantRNG.Uint64())
	}
	predicateRNG := streamRNG(panel, seed, "predicate-aliases", 0)
	for index := range fixture.PredicateAliases {
		fixture.PredicateAliases[index] = fmt.Sprintf("r-%016x", predicateRNG.Uint64())
	}
	specs := [3][2]int{{1, 6}, {1, 6}, {1, 6}}
	if cohort == Beneficial || cohort == Harmful {
		specs[0] = [2]int{3, 6}
	}
	if cohort == Harmful {
		specs[1] = [2]int{3, 6}
	}
	for position := 0; position < 3; position++ {
		if cohort == Neutral && position > 0 {
			fixture.Background[position] = fixture.Background[0]
			fixture.Edges[position] = append([]Edge(nil), fixture.Edges[0]...)
			fixture.Attempts[position] = fixture.Attempts[0]
			continue
		}
		graph, err := sampledGraph(panel, seed, fmt.Sprintf("training-graph-%d", position), specs[position][0], specs[position][1], startAttempt)
		if err != nil {
			return Fixture{}, err
		}
		fixture.Background[position], fixture.Edges[position], fixture.Attempts[position] = graph.Relation, graph.Edges, graph.Attempt
	}
	for position := 0; position < 3; position++ {
		if cohort == Neutral && position > 0 {
			fixture.HeldOut[position] = fixture.HeldOut[0]
			fixture.HeldEdges[position] = append([]Edge(nil), fixture.HeldEdges[0]...)
			fixture.HeldAttempts[position] = fixture.HeldAttempts[0]
			continue
		}
		graph, err := sampledGraph(panel, seed, fmt.Sprintf("heldout-graph-%d", position), specs[position][0], specs[position][1], startAttempt)
		if err != nil {
			return Fixture{}, err
		}
		fixture.HeldOut[position], fixture.HeldEdges[position], fixture.HeldAttempts[position] = graph.Relation, graph.Edges, graph.Attempt
	}
	code1, code2 := "03", "03"
	if cohort == Harmful {
		code2 = "14"
	}
	fixture.Target1 = ruleinductionoracle.Evaluate(definition(code1), fixture.Background)
	fixture.Target2 = ruleinductionoracle.Evaluate(definition(code2), fixture.Background)
	fixture.HeldTarget1 = ruleinductionoracle.Evaluate(definition(code1), fixture.HeldOut)
	fixture.HeldTarget2 = ruleinductionoracle.Evaluate(definition(code2), fixture.HeldOut)
	if cohort != Neutral {
		if got := exactCodes(fixture.Background, fixture.Target1); len(got) != 1 || got[0] != code1 {
			return Fixture{}, fmt.Errorf("stage 1 exact codes %v", got)
		}
		if got := exactCodes(fixture.Background, fixture.Target2); len(got) != 1 || got[0] != code2 {
			return Fixture{}, fmt.Errorf("stage 2 exact codes %v", got)
		}
	}
	var err error
	fixture.Stage1, err = separatingExamples(fixture.Background, fixture.Target1)
	if err != nil {
		return Fixture{}, err
	}
	fixture.Stage2, err = separatingExamples(fixture.Background, fixture.Target2)
	if err != nil {
		return Fixture{}, err
	}
	return fixture, nil
}

func CanonicalFacts(fixture Fixture) []string {
	var facts []string
	for predicate, edges := range fixture.Edges {
		for _, edge := range edges {
			facts = append(facts, fmt.Sprintf("%d:%d:%d", predicate, edge.X, edge.Y))
		}
	}
	sort.Strings(facts)
	return facts
}
