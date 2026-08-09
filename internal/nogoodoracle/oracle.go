// Package nogoodoracle independently parses and exhaustively solves the public
// finite inequality CSP encoding. It intentionally imports no production,
// fixture, experiment, engine, seed, DSL, or CUE package.
package nogoodoracle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
)

type variable struct {
	Alias  string `json:"alias"`
	Domain []int  `json:"domain"`
}

type edge struct {
	Left  int `json:"left"`
	Right int `json:"right"`
}

type literal struct {
	Variable int `json:"variable"`
	Color    int `json:"color"`
}

type problem struct {
	Version      string     `json:"version"`
	ColorAliases []string   `json:"color_aliases"`
	Variables    []variable `json:"variables"`
	Edges        []edge     `json:"edges"`
	Assignment   []literal  `json:"assignment"`
}

type Literal struct {
	Variable int
	Color    int
}

type Result struct {
	Satisfiable bool
	Solutions   [][]int
	Digest      string
	Work        int64
}

func Enumerate(data []byte, decision Literal) (Result, error) {
	p, err := parse(data)
	if err != nil {
		return Result{}, err
	}
	assignment := make([]int, len(p.Variables))
	for index := range assignment {
		assignment[index] = -1
	}
	for _, item := range p.Assignment {
		assignment[item.Variable] = item.Color
	}
	if decision.Variable < 0 || decision.Variable >= len(assignment) || !slices.Contains(p.Variables[decision.Variable].Domain, decision.Color) || assignment[decision.Variable] >= 0 {
		return Result{}, fmt.Errorf("invalid decision")
	}
	assignment[decision.Variable] = decision.Color
	if !consistent(p, assignment) {
		return finalize(nil, 1), nil
	}
	var solutions [][]int
	var work int64
	var visit func(int)
	visit = func(variableIndex int) {
		for variableIndex < len(assignment) && assignment[variableIndex] >= 0 {
			variableIndex++
		}
		if variableIndex == len(assignment) {
			work++
			if consistent(p, assignment) {
				solutions = append(solutions, slices.Clone(assignment))
			}
			return
		}
		for _, color := range p.Variables[variableIndex].Domain {
			work++
			assignment[variableIndex] = color
			if consistent(p, assignment) {
				visit(variableIndex + 1)
			}
			assignment[variableIndex] = -1
		}
	}
	visit(0)
	return finalize(solutions, work), nil
}

func finalize(solutions [][]int, work int64) Result {
	if solutions == nil {
		solutions = [][]int{}
	}
	encoded, _ := json.Marshal(solutions)
	digest := sha256.Sum256(encoded)
	return Result{Satisfiable: len(solutions) > 0, Solutions: solutions, Digest: hex.EncodeToString(digest[:]), Work: work}
}

func parse(data []byte) (problem, error) {
	var p problem
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return problem{}, err
	}
	canonical, _ := json.Marshal(p)
	if !bytes.Equal(canonical, data) || p.Version != "finite-neq-csp/v1" || len(p.ColorAliases) == 0 || len(p.ColorAliases) > 4 || len(p.Variables) == 0 || len(p.Variables) > 8 || len(p.Edges) > 18 {
		return problem{}, fmt.Errorf("invalid public CSP")
	}
	for _, variable := range p.Variables {
		if len(variable.Domain) == 0 {
			return problem{}, fmt.Errorf("empty domain")
		}
		previous := -1
		for _, color := range variable.Domain {
			if color < 0 || color >= len(p.ColorAliases) || color <= previous {
				return problem{}, fmt.Errorf("invalid domain")
			}
			previous = color
		}
	}
	previous := edge{Left: -1, Right: -1}
	for index, current := range p.Edges {
		if current.Left < 0 || current.Right >= len(p.Variables) || current.Left >= current.Right || index > 0 && !edgeLess(previous, current) {
			return problem{}, fmt.Errorf("invalid edge")
		}
		previous = current
	}
	previousVariable := -1
	for _, item := range p.Assignment {
		if item.Variable < 0 || item.Variable >= len(p.Variables) || item.Variable <= previousVariable || !slices.Contains(p.Variables[item.Variable].Domain, item.Color) {
			return problem{}, fmt.Errorf("invalid assignment")
		}
		previousVariable = item.Variable
	}
	return p, nil
}

func edgeLess(a, b edge) bool { return a.Left < b.Left || a.Left == b.Left && a.Right < b.Right }

func consistent(p problem, assignment []int) bool {
	for _, current := range p.Edges {
		if assignment[current.Left] >= 0 && assignment[current.Right] >= 0 && assignment[current.Left] == assignment[current.Right] {
			return false
		}
	}
	return true
}
