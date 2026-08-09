package nogoodfixture

import (
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/vocab/nogoods"
)

type CompetenceMutation string

const (
	NoMutation          CompetenceMutation = ""
	DuplicateCompletion CompetenceMutation = "duplicate-completion"
	CrossDecision       CompetenceMutation = "cross-decision"
	StaleTarget         CompetenceMutation = "stale-target"
)

type CompetenceCase struct {
	Task
	Kind         string
	WantProposal bool
	Mutation     CompetenceMutation
}

// Competence returns the fixed public construction/certification panel. It is
// separate from utility panels and carries no empirical-work claim.
func Competence(panel string) ([]CompetenceCase, error) {
	start, count := 0, 0
	switch panel {
	case "development":
		start, count = 831101, 8
	case "validation":
		start, count = 831201, 16
	default:
		return nil, fmt.Errorf("unknown competence panel %q", panel)
	}
	cases := make([]CompetenceCase, 0, count)
	for ordinal := 0; ordinal < count; ordinal++ {
		competenceCase, err := competenceCase(panel, start+ordinal, ordinal)
		if err != nil {
			return nil, err
		}
		cases = append(cases, competenceCase)
	}
	return cases, nil
}

func competenceCase(panel string, seed, ordinal int) (CompetenceCase, error) {
	semantic := ordinal
	if semantic >= 8 {
		semantic = ordinal
	}
	kinds := []string{
		"full", "external-context", "missing-0", "missing-1", "missing-2", "wrong-decision", "pair-domain-three", "duplicate-completion",
		"missing-0-1", "missing-0-2", "missing-1-2", "anchor-domain-three", "unequal-pair-domains", "cross-decision", "stale-target", "color-position-audit",
	}
	kind := kinds[semantic]
	n := 3
	if kind == "external-context" {
		n = 8
	}
	streamPanel := "competence-" + panel
	variablePositions := permutation(n, stream(streamPanel, seed, 0, "variable-positions"))
	variableAliases := permutation(n, stream(streamPanel, seed, 0, "variable-aliases"))
	colorPositions := permutation(4, stream(streamPanel, seed, 0, "color-positions"))
	colorAliases := permutation(4, stream(streamPanel, seed, 0, "color-aliases"))
	blocked, escape := colorPositions[0], colorPositions[1]
	if blocked > escape {
		blocked, escape = escape, blocked
	}
	only, spare := colorPositions[2], colorPositions[3]
	roleDomains := make([][]int, n)
	roleDomains[0], roleDomains[1], roleDomains[2] = sorted(blocked, escape), sorted(blocked, only), sorted(blocked, only)
	for role := 3; role < n; role++ {
		roleDomains[role] = []int{0, 1, 2, 3}
	}
	switch kind {
	case "pair-domain-three":
		roleDomains[1], roleDomains[2] = sorted(blocked, only, spare), sorted(blocked, only, spare)
	case "anchor-domain-three":
		roleDomains[0] = sorted(blocked, escape, spare)
	case "unequal-pair-domains":
		roleDomains[2] = sorted(blocked, spare)
	}
	aliases := make([]string, n)
	for descriptor, source := range variableAliases {
		aliases[descriptor] = fmt.Sprintf("ha%d", source)
	}
	colors := make([]string, 4)
	for descriptor, source := range colorAliases {
		colors[descriptor] = fmt.Sprintf("hc%d", source)
	}
	variables := make([]nogoods.Variable, n)
	for role, descriptor := range variablePositions {
		variables[descriptor] = nogoods.Variable{Alias: aliases[descriptor], Domain: slices.Clone(roleDomains[role])}
	}
	missing := map[int]bool{}
	switch kind {
	case "missing-0":
		missing[0] = true
	case "missing-1":
		missing[1] = true
	case "missing-2":
		missing[2] = true
	case "missing-0-1":
		missing[0], missing[1] = true, true
	case "missing-0-2":
		missing[0], missing[2] = true, true
	case "missing-1-2":
		missing[1], missing[2] = true, true
	}
	roleEdges := [][2]int{{0, 1}, {0, 2}, {1, 2}}
	var selected [][2]int
	for bit, edge := range roleEdges {
		if !missing[bit] {
			selected = append(selected, edge)
		}
	}
	if kind == "external-context" {
		selected = append(selected, [2]int{3, 4}, [2]int{4, 5}, [2]int{5, 6}, [2]int{6, 7})
	}
	problem := nogoods.Problem{Version: nogoods.ProblemVersion, ColorAliases: colors, Variables: variables, Edges: canonicalEdges(selected, variablePositions), Assignment: []nogoods.Literal{}}
	encoded, err := problem.CanonicalJSON()
	if err != nil {
		return CompetenceCase{}, err
	}
	decisionColor := blocked
	if kind == "wrong-decision" {
		decisionColor = escape
	}
	want := kind == "full" || kind == "external-context" || kind == "duplicate-completion" || kind == "cross-decision" || kind == "stale-target" || kind == "color-position-audit"
	mutation := NoMutation
	switch kind {
	case "duplicate-completion":
		mutation = DuplicateCompletion
	case "cross-decision":
		mutation = CrossDecision
	case "stale-target":
		mutation = StaleTarget
	}
	cohort := NearMiss
	if want {
		cohort = Reusable
	}
	return CompetenceCase{Task: Task{Panel: streamPanel, Ordinal: ordinal, Seed: seed, Cohort: cohort, ProblemJSON: encoded, Decision: nogoods.Literal{Variable: variablePositions[0], Color: decisionColor}}, Kind: kind, WantProposal: want, Mutation: mutation}, nil
}
