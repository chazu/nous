package nogoodfixturecore

import (
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/vocab/nogoods"
)

type CompetenceResult struct {
	ProblemJSON  []byte
	Decision     nogoods.Literal
	Kind         string
	WantProposal bool
	Mutation     string
}

func ConstructCompetence(ordinal int, stream Stream) (CompetenceResult, error) {
	kinds := []string{
		"full", "external-context", "missing-0", "missing-1", "missing-2", "wrong-decision", "pair-domain-three", "duplicate-completion",
		"missing-0-1", "missing-0-2", "missing-1-2", "anchor-domain-three", "unequal-pair-domains", "cross-decision", "stale-target", "color-position-audit",
	}
	if ordinal < 0 || ordinal >= len(kinds) {
		return CompetenceResult{}, fmt.Errorf("competence ordinal %d is out of range", ordinal)
	}
	kind := kinds[ordinal]
	n := 3
	if kind == "external-context" {
		n = 8
	}
	variablePositions := permutation(n, stream("variable-positions"))
	variableAliases := permutation(n, stream("variable-aliases"))
	colorPositions := permutation(4, stream("color-positions"))
	colorAliases := permutation(4, stream("color-aliases"))
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
		return CompetenceResult{}, err
	}
	decisionColor := blocked
	if kind == "wrong-decision" {
		decisionColor = escape
	}
	want := kind == "full" || kind == "external-context" || kind == "duplicate-completion" || kind == "cross-decision" || kind == "stale-target" || kind == "color-position-audit"
	mutation := ""
	switch kind {
	case "duplicate-completion":
		mutation = "duplicate-completion"
	case "cross-decision":
		mutation = "cross-decision"
	case "stale-target":
		mutation = "stale-target"
	}
	return CompetenceResult{
		ProblemJSON: encoded, Decision: nogoods.Literal{Variable: variablePositions[0], Color: decisionColor},
		Kind: kind, WantProposal: want, Mutation: mutation,
	}, nil
}
