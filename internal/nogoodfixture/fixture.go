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

	"github.com/chazu/nous/internal/nogoodfixturecore"
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

type PromotionCase struct {
	Ordinal     int
	ProblemJSON []byte
	Decision    nogoods.Literal
	Binding     nogoods.Binding
	Completion  nogoods.Completion
}

// PromotionCases is the complete injective substitution set for the three
// color roles. Each record is explicit input to the heuristic; the production
// vocabulary only evaluates one supplied record at a time.
func PromotionCases() ([]PromotionCase, error) {
	cases := make([]PromotionCase, 0, 24)
	for blocked := 0; blocked < 4; blocked++ {
		for escape := 0; escape < 4; escape++ {
			for only := 0; only < 4; only++ {
				if blocked == escape || blocked == only || escape == only {
					continue
				}
				problem := nogoods.Problem{
					Version:      nogoods.ProblemVersion,
					ColorAliases: []string{"pc0", "pc1", "pc2", "pc3"},
					Variables: []nogoods.Variable{
						{Alias: "pa", Domain: sorted(blocked, escape)},
						{Alias: "px", Domain: sorted(blocked, only)},
						{Alias: "py", Domain: sorted(blocked, only)},
					},
					Edges:      []nogoods.Edge{{Left: 0, Right: 1}, {Left: 0, Right: 2}, {Left: 1, Right: 2}},
					Assignment: []nogoods.Literal{},
				}
				encoded, err := problem.CanonicalJSON()
				if err != nil {
					return nil, fmt.Errorf("construct promotion case: %w", err)
				}
				cases = append(cases, PromotionCase{
					Ordinal: len(cases), ProblemJSON: encoded,
					Decision:   nogoods.Literal{Variable: 0, Color: blocked},
					Binding:    nogoods.Binding{Anchor: 0, X: 1, Y: 2, Blocked: blocked, Escape: escape, Only: only},
					Completion: nogoods.Completion{XColor: only, YColor: only},
				})
			}
		}
	}
	return cases, nil
}

// Training constructs the four public examples from which the schema must be
// learned. The hidden MissingBit field is for fixture/oracle audits only; it is
// never serialized into the problem object or inserted into a Nous store.
func Training() ([]Task, error) {
	tasks := make([]Task, 0, 4)
	for ordinal, seed := range []int{831001, 831002, 831003, 831004} {
		task, err := trainingTask(seed, ordinal)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func RandomControlMask() int {
	return int(stream("training", 831001, 0, "random-control").Uint64N(7))
}

func trainingTask(seed, ordinal int) (Task, error) {
	variablePositions := permutation(3, stream("training", seed, 0, "variable-positions"))
	variableAliases := permutation(3, stream("training", seed, 0, "variable-aliases"))
	colorPositions := permutation(4, stream("training", seed, 0, "color-positions"))
	colorAliases := permutation(4, stream("training", seed, 0, "color-aliases"))
	blocked, escape := colorPositions[0], colorPositions[1]
	if blocked > escape {
		blocked, escape = escape, blocked
	}
	only := colorPositions[2]

	aliases := make([]string, 3)
	for descriptor, source := range variableAliases {
		aliases[descriptor] = fmt.Sprintf("ta%d", source)
	}
	colors := make([]string, 4)
	for descriptor, source := range colorAliases {
		colors[descriptor] = fmt.Sprintf("tc%d", source)
	}
	roleDomains := [3][]int{sorted(blocked, escape), sorted(blocked, only), sorted(blocked, only)}
	variables := make([]nogoods.Variable, 3)
	for role, descriptor := range variablePositions {
		variables[descriptor] = nogoods.Variable{Alias: aliases[descriptor], Domain: roleDomains[role]}
	}
	anchorDescriptor := variablePositions[0]
	xDescriptor, yDescriptor := variablePositions[1], variablePositions[2]
	if xDescriptor > yDescriptor {
		xDescriptor, yDescriptor = yDescriptor, xDescriptor
	}
	descriptorEdges := [3]nogoods.Edge{
		canonicalEdge(anchorDescriptor, xDescriptor),
		canonicalEdge(anchorDescriptor, yDescriptor),
		canonicalEdge(xDescriptor, yDescriptor),
	}
	missing := -1
	if ordinal > 0 {
		missing = ordinal - 1
	}
	edges := make([]nogoods.Edge, 0, 3)
	for bit, edge := range descriptorEdges {
		if bit != missing {
			edges = append(edges, edge)
		}
	}
	slices.SortFunc(edges, func(a, b nogoods.Edge) int {
		if a.Left != b.Left {
			return a.Left - b.Left
		}
		return a.Right - b.Right
	})
	problem := nogoods.Problem{
		Version:      nogoods.ProblemVersion,
		ColorAliases: colors,
		Variables:    variables,
		Edges:        edges,
		Assignment:   []nogoods.Literal{},
	}
	encoded, err := problem.CanonicalJSON()
	if err != nil {
		return Task{}, fmt.Errorf("construct training task %d: %w", ordinal, err)
	}
	return Task{
		Panel: "training", Ordinal: ordinal, Seed: seed, MissingBit: missing,
		ProblemJSON: encoded,
		Decision:    nogoods.Literal{Variable: variablePositions[0], Color: blocked},
	}, nil
}

func canonicalEdge(left, right int) nogoods.Edge {
	if left > right {
		left, right = right, left
	}
	return nogoods.Edge{Left: left, Right: right}
}

type panelShape struct {
	SeedsStart int
	Counts     [4]int
}

var cohorts = [4]Cohort{Reusable, NearMiss, Irrelevant, IndependentUnsat}

// DevelopmentPanel constructs the public development stream. Protected panel
// construction is owned by the experiment guard.
func DevelopmentPanel() ([]Task, error) {
	const panel = "development"
	shape := panelShape{SeedsStart: 832001, Counts: [4]int{56, 24, 8, 8}}
	total := 0
	for _, count := range shape.Counts {
		total += count
	}
	tasks := make([]Task, 0, total)
	global := 0
	for cohortIndex, cohort := range cohorts {
		for local := 0; local < shape.Counts[cohortIndex]; local++ {
			task, err := utilityTask(panel, shape.SeedsStart+global, global, cohort, local)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, task)
			global++
		}
	}
	return tasks, nil
}

func utilityTask(panel string, seed, global int, cohort Cohort, local int) (Task, error) {
	coreCohort := map[Cohort]nogoodfixturecore.Cohort{
		Reusable: nogoodfixturecore.Reusable, NearMiss: nogoodfixturecore.NearMiss,
		Irrelevant: nogoodfixturecore.Irrelevant, IndependentUnsat: nogoodfixturecore.IndependentUnsat,
	}[cohort]
	constructed, err := nogoodfixturecore.Construct(global, coreCohort, local, func(purpose string) *rand.Rand {
		return stream(panel, seed, 0, purpose)
	})
	if err != nil {
		return Task{}, err
	}
	return Task{
		Panel: panel, Ordinal: global, Seed: seed, Cohort: cohort, CohortOrdinal: local,
		Template: constructed.Template, MissingBit: constructed.MissingBit,
		ProblemJSON: constructed.ProblemJSON, Decision: constructed.Decision,
	}, nil
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
