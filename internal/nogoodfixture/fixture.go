// Package nogoodfixture constructs deterministic public CSP fixtures. Cohort
// metadata is retained only by the experiment wrapper and is never serialized
// into ProblemJSON.
package nogoodfixture

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/chazu/nous/internal/vocab/nogoods"
)

const SeedAuthority = "part3/nogoods/v1"

type Cohort string

const (
	Reusable         Cohort = "reusable"
	NearMiss         Cohort = "near-miss"
	Irrelevant       Cohort = "irrelevant"
	IndependentUnsat Cohort = "independent-unsat"
)

type Task struct {
	Panel         string
	Ordinal       int
	Seed          int
	Cohort        Cohort
	CohortOrdinal int
	Template      int
	MissingBit    int
	ProblemJSON   []byte
	Decision      nogoods.Literal
}

type panelShape struct {
	SeedsStart int
	Counts     [4]int
}

var panelShapes = map[string]panelShape{
	"development": {SeedsStart: 832001, Counts: [4]int{56, 24, 8, 8}},
	"validation":  {SeedsStart: 833001, Counts: [4]int{112, 48, 16, 16}},
}

var cohorts = [4]Cohort{Reusable, NearMiss, Irrelevant, IndependentUnsat}

// Panel constructs the public development or validation stream.
func Panel(panel string) ([]Task, error) {
	shape, ok := panelShapes[panel]
	if !ok {
		return nil, fmt.Errorf("unknown nogood panel %q", panel)
	}
	total := 0
	for _, count := range shape.Counts {
		total += count
	}
	tasks := make([]Task, 0, total)
	global := 0
	for cohortIndex, cohort := range cohorts {
		for local := 0; local < shape.Counts[cohortIndex]; local++ {
			task, err := utilityTask(panel, shape.SeedsStart+global, global, cohort, local, nil)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, task)
			global++
		}
	}
	return tasks, nil
}

// LockedPanel constructs the private-root stream after the caller has claimed
// its one-shot guard. The root is never included in Task or public bytes.
func LockedPanel(root string) ([]Task, error) {
	if len(root) != 64 {
		return nil, fmt.Errorf("locked root must be 64 lowercase hex characters")
	}
	for _, c := range root {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return nil, fmt.Errorf("locked root must be 64 lowercase hex characters")
		}
	}
	counts := [4]int{312, 48, 12, 12}
	var tasks []Task
	global := 0
	for cohortIndex, cohort := range cohorts {
		for local := 0; local < counts[cohortIndex]; local++ {
			task, err := utilityTask("locked", 0, global, cohort, local, root)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, task)
			global++
		}
	}
	return tasks, nil
}

func utilityTask(panel string, seed, global int, cohort Cohort, local int, lockedRoot any) (Task, error) {
	root := any(seed)
	ordinal := 0
	if panel == "locked" {
		root = lockedRoot
		ordinal = global
	}
	variablePositions := permutation(8, stream(panel, root, ordinal, "variable-positions"))
	variableAliases := permutation(8, stream(panel, root, ordinal, "variable-aliases"))
	colorPositions := permutation(4, stream(panel, root, ordinal, "color-positions"))
	colorAliases := permutation(4, stream(panel, root, ordinal, "color-aliases"))

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
	if cohort == Reusable || cohort == NearMiss {
		for d := 3; d < 8; d++ {
			roleEdges = append(roleEdges, [2]int{0, d})
		}
		motif := [3][2]int{{0, 1}, {0, 2}, {1, 2}}
		missing := -1
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
		return Task{}, fmt.Errorf("construct utility task %d: %w", global, err)
	}
	return Task{Panel: panel, Ordinal: global, Seed: seed, Cohort: cohort, CohortOrdinal: local, Template: template, MissingBit: func() int {
		if cohort == NearMiss {
			return local % 3
		}
		return -1
	}(), ProblemJSON: encoded, Decision: nogoods.Literal{Variable: rolePosition[0], Color: blocked}}, nil
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

func stream(panel string, root any, ordinal int, purpose string) *rand.Rand {
	encoded, err := json.Marshal([]any{SeedAuthority, panel, root, ordinal, purpose})
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(encoded)
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
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
