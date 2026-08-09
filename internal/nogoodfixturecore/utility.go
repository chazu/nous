// Package nogoodfixturecore constructs one public utility CSP from an
// externally supplied deterministic stream. It knows no panel, seed, or
// protected execution identity.
package nogoodfixturecore

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/chazu/nous/internal/vocab/nogoods"
)

type Cohort uint8

const (
	Reusable Cohort = iota
	NearMiss
	Irrelevant
	IndependentUnsat
)

type Result struct {
	ProblemJSON []byte
	Decision    nogoods.Literal
	Template    int
	MissingBit  int
}

type Stream func(purpose string) *rand.Rand

func Construct(global int, cohort Cohort, local int, stream Stream) (Result, error) {
	variablePositions := permutation(8, stream("variable-positions"))
	variableAliases := permutation(8, stream("variable-aliases"))
	colorPositions := permutation(4, stream("color-positions"))
	colorAliases := permutation(4, stream("color-aliases"))

	blocked, escape := colorPositions[0], colorPositions[1]
	if blocked > escape {
		blocked, escape = escape, blocked
	}
	only, spare := colorPositions[2], colorPositions[3]
	rolePosition := variablePositions
	roleDomains := make([][]int, 8)
	all := []int{0, 1, 2, 3}
	if cohort == Reusable || cohort == NearMiss {
		roleDomains[0] = sorted(blocked, escape)
		roleDomains[1] = sorted(blocked, only)
		roleDomains[2] = sorted(blocked, only)
	} else {
		roleDomains[0] = slices.Clone(all)
		roleDomains[1] = slices.Clone(all)
		roleDomains[2] = slices.Clone(all)
	}
	template := global % 4
	distractorDomains := [4][5][]int{
		{all, all, all, all, all},
		{sorted(blocked, escape, spare), all, sorted(escape, only, spare), all, sorted(blocked, only, spare)},
		{all, sorted(blocked, escape, only), all, sorted(escape, only, spare), all},
		{sorted(blocked, escape, spare), sorted(escape, only, spare), all, sorted(blocked, only, spare), all},
	}
	for index := 0; index < 5; index++ {
		roleDomains[index+3] = slices.Clone(distractorDomains[template][index])
	}
	if cohort == Irrelevant {
		for role := range roleDomains {
			roleDomains[role] = slices.Clone(all)
		}
	}
	if cohort == IndependentUnsat {
		for role := range roleDomains {
			roleDomains[role] = slices.Clone(all)
		}
		roleDomains[3] = []int{blocked}
		roleDomains[4] = []int{blocked}
	}

	aliases := make([]string, 8)
	for descriptor, source := range variableAliases {
		aliases[descriptor] = fmt.Sprintf("ha%d", source)
	}
	colors := make([]string, 4)
	for descriptor, source := range colorAliases {
		colors[descriptor] = fmt.Sprintf("hc%d", source)
	}
	variables := make([]nogoods.Variable, 8)
	for role, descriptor := range rolePosition {
		variables[descriptor] = nogoods.Variable{Alias: aliases[descriptor], Domain: roleDomains[role]}
	}

	roleEdges := append([][2]int(nil), distractorEdges[template]...)
	missing := -1
	if cohort == Reusable || cohort == NearMiss {
		for d := 3; d < 8; d++ {
			roleEdges = append(roleEdges, [2]int{0, d})
		}
		motif := [3][2]int{{0, 1}, {0, 2}, {1, 2}}
		if cohort == NearMiss {
			missing = local % 3
		}
		for bit, edge := range motif {
			if bit != missing {
				roleEdges = append(roleEdges, edge)
			}
		}
	} else if cohort == Irrelevant {
		for d := 3; d < 8; d++ {
			roleEdges = append(roleEdges, [2]int{0, d})
		}
		roleEdges = append(roleEdges, [2]int{0, 1}, [2]int{1, 2})
	} else {
		roleEdges = append(roleEdges, [2]int{3, 4})
	}
	edges := canonicalEdges(roleEdges, rolePosition)
	problem := nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: colors, Variables: variables, Edges: edges, Assignment: []nogoods.Literal{}}
	encoded, err := problem.CanonicalJSON()
	if err != nil {
		return Result{}, fmt.Errorf("construct utility task %d: %w", global, err)
	}
	return Result{
		ProblemJSON: encoded, Decision: nogoods.Literal{Variable: rolePosition[0], Color: blocked},
		Template: template, MissingBit: missing,
	}, nil
}

var distractorEdges = [4][][2]int{
	{{3, 4}, {3, 5}, {3, 6}, {3, 7}, {4, 6}, {4, 7}, {5, 6}, {5, 7}},
	{{3, 4}, {3, 5}, {3, 6}, {4, 5}, {4, 6}, {4, 7}, {5, 7}, {6, 7}},
	{{3, 4}, {3, 5}, {3, 6}, {3, 7}, {4, 6}, {4, 7}, {5, 6}, {5, 7}},
	{{3, 4}, {3, 5}, {3, 7}, {4, 5}, {4, 6}, {5, 6}, {5, 7}, {6, 7}},
}

func canonicalEdges(roleEdges [][2]int, positions []int) []nogoods.Edge {
	set := map[[2]int]bool{}
	for _, edge := range roleEdges {
		left, right := positions[edge[0]], positions[edge[1]]
		if left > right {
			left, right = right, left
		}
		set[[2]int{left, right}] = true
	}
	edges := make([]nogoods.Edge, 0, len(set))
	for edge := range set {
		edges = append(edges, nogoods.Edge{Left: edge[0], Right: edge[1]})
	}
	slices.SortFunc(edges, func(a, b nogoods.Edge) int {
		if a.Left != b.Left {
			return a.Left - b.Left
		}
		return a.Right - b.Right
	})
	return edges
}

func sorted(values ...int) []int {
	values = slices.Clone(values)
	slices.Sort(values)
	return values
}

func permutation(n int, source *rand.Rand) []int {
	values := make([]int, n)
	for index := range values {
		values[index] = index
	}
	for i := n - 1; i >= 1; i-- {
		j := int(source.Uint64N(uint64(i + 1)))
		values[i], values[j] = values[j], values[i]
	}
	return values
}
