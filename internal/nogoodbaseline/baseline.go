// Package nogoodbaseline contains conventional CSP policies implemented from
// canonical public JSON. It imports no production or Nous heuristic package.
package nogoodbaseline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
)

type Literal struct {
	Variable int
	Color    int
}

type Event struct {
	Category   int
	Transition string
	Operands   []int
}

type Result struct {
	Satisfied bool
	Witness   []int
	Work      int64
	Vector    [12]int64
	Events    []Event
}

type variable struct {
	Alias  string `json:"alias"`
	Domain []int  `json:"domain"`
}
type edge struct {
	Left, Right int
}

func (e *edge) UnmarshalJSON(data []byte) error {
	var value struct{ Left, Right int }
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	e.Left, e.Right = value.Left, value.Right
	return nil
}
func (e edge) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Left  int `json:"left"`
		Right int `json:"right"`
	}{e.Left, e.Right})
}

type assignedLiteral struct {
	Variable int `json:"variable"`
	Color    int `json:"color"`
}
type problem struct {
	Version      string            `json:"version"`
	ColorAliases []string          `json:"color_aliases"`
	Variables    []variable        `json:"variables"`
	Edges        []edge            `json:"edges"`
	Assignment   []assignedLiteral `json:"assignment"`
}

type meter struct {
	result *Result
}

func (m meter) charge(category int, transition string, operands ...int) {
	m.result.Work++
	m.result.Vector[category-1]++
	m.result.Events = append(m.result.Events, Event{Category: category, Transition: transition, Operands: slices.Clone(operands)})
}

type solver struct {
	p            problem
	domains      [][]int
	assignment   []int
	explanations []map[int]map[int]bool
	degree       []int
	meter        meter
}

// MACCBJ answers one fixed-branch completion query using maintaining arc
// consistency, MRV/degree ordering, exact local conflict sets, and
// conflict-directed backjump returns.
func MACCBJ(data []byte, decision Literal) (Result, error) {
	p, err := parse(data)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	s := newSolver(p, &result)
	for _, item := range p.Assignment {
		s.assignment[item.Variable] = item.Color
	}
	rootDomains := cloneDomains(s.domains)
	rootExplanations := cloneExplanations(s.explanations)
	s.meter.charge(2, "decision-domain-read", decision.Variable, decision.Color)
	s.meter.charge(3, "decision-propose", decision.Variable, decision.Color)
	if !contains(s.domains[decision.Variable], decision.Color) || s.assignment[decision.Variable] >= 0 {
		return Result{}, fmt.Errorf("invalid supplied decision")
	}
	s.assignment[decision.Variable] = decision.Color
	s.meter.charge(3, "decision-bind", decision.Variable, decision.Color)
	for _, color := range slices.Clone(s.domains[decision.Variable]) {
		if color == decision.Color {
			continue
		}
		s.remove(decision.Variable, color, map[int]bool{decision.Variable: true})
		s.meter.charge(5, "domain-empty-check", decision.Variable)
	}
	if !s.assignedEdgesConsistent(decision.Variable) {
		s.finishRoot(&result, decision, rootDomains, rootExplanations)
		return result, nil
	}
	queue := s.initialQueue(decision.Variable)
	failure := s.ac3(queue)
	if failure == nil {
		var witness []int
		witness, failure = s.search()
		if witness != nil {
			result.Satisfied = true
			result.Witness = witness
			s.meter.charge(11, "witness-check")
		}
	}
	if !result.Satisfied {
		// The root wrapper seals the concrete decision nogood before restore.
		s.meter.charge(12, "nogood-lookup", decision.Variable, decision.Color)
		s.meter.charge(12, "nogood-write", decision.Variable, decision.Color)
		if failure != nil {
			s.meter.charge(7, "root-project", decision.Variable)
		}
	}
	s.finishRoot(&result, decision, rootDomains, rootExplanations)
	return result, nil
}

func newSolver(p problem, result *Result) *solver {
	s := &solver{p: p, domains: make([][]int, len(p.Variables)), assignment: make([]int, len(p.Variables)), explanations: make([]map[int]map[int]bool, len(p.Variables)), degree: make([]int, len(p.Variables)), meter: meter{result}}
	for i, v := range p.Variables {
		s.domains[i] = slices.Clone(v.Domain)
		s.assignment[i] = -1
		s.explanations[i] = map[int]map[int]bool{}
	}
	for _, e := range p.Edges {
		s.degree[e.Left]++
		s.degree[e.Right]++
	}
	return s
}

func (s *solver) finishRoot(result *Result, decision Literal, domains [][]int, explanations []map[int]map[int]bool) {
	for variable := range s.domains {
		for _, color := range domains[variable] {
			if !contains(s.domains[variable], color) {
				s.meter.charge(5, "domain-restore", variable, color)
			}
		}
	}
	s.domains = domains
	s.explanations = explanations
	s.assignment[decision.Variable] = -1
	s.meter.charge(3, "decision-unbind", decision.Variable, decision.Color)
	s.meter.charge(11, "terminal-classification")
	s.meter.charge(12, "terminal-record-write")
}

func (s *solver) search() ([]int, map[int]bool) {
	variable := s.chooseVariable()
	if variable < 0 {
		if s.completeConsistent() {
			return slices.Clone(s.assignment), nil
		}
		return nil, map[int]bool{}
	}
	conflicts := map[int]bool{}
	for _, color := range slices.Clone(s.domains[variable]) {
		domains := cloneDomains(s.domains)
		explanations := cloneExplanations(s.explanations)
		assignment := slices.Clone(s.assignment)
		s.meter.charge(3, "assignment-propose", variable, color)
		s.assignment[variable] = color
		s.meter.charge(3, "assignment-bind", variable, color)
		for _, other := range slices.Clone(s.domains[variable]) {
			if other != color {
				s.remove(variable, other, map[int]bool{variable: true})
				s.meter.charge(5, "domain-empty-check", variable)
			}
		}
		if !s.assignedEdgesConsistent(variable) {
			failure := s.assignedNeighborSet(variable)
			failure[variable] = true
			s.restore(domains, explanations, assignment)
			for member := range failure {
				if member != variable {
					conflicts[member] = true
					s.meter.charge(7, "conflict-insert", member)
				}
			}
			continue
		}
		failure := s.ac3(s.initialQueue(variable))
		if failure == nil {
			witness, nested := s.search()
			if witness != nil {
				return witness, nil
			}
			failure = nested
		}
		s.restore(domains, explanations, assignment)
		if !failure[variable] {
			s.meter.charge(7, "backjump", variable)
			return nil, failure
		}
		for member := range failure {
			if member != variable {
				conflicts[member] = true
				s.meter.charge(7, "conflict-union", member)
			}
		}
	}
	return nil, conflicts
}

func (s *solver) restore(domains [][]int, explanations []map[int]map[int]bool, assignment []int) {
	for variable := range domains {
		for _, color := range domains[variable] {
			if !contains(s.domains[variable], color) {
				s.meter.charge(5, "domain-restore", variable, color)
			}
		}
	}
	for variable, color := range s.assignment {
		if color >= 0 && assignment[variable] < 0 {
			s.meter.charge(3, "assignment-unbind", variable, color)
		}
	}
	s.domains = domains
	s.explanations = explanations
	s.assignment = assignment
}

func (s *solver) chooseVariable() int {
	best := -1
	for variable := range s.domains {
		if s.assignment[variable] >= 0 {
			continue
		}
		if best < 0 || len(s.domains[variable]) < len(s.domains[best]) || len(s.domains[variable]) == len(s.domains[best]) && (s.degree[variable] > s.degree[best] || s.degree[variable] == s.degree[best] && variable < best) {
			best = variable
		}
	}
	return best
}

type arc struct{ x, y int }

func (s *solver) initialQueue(variable int) []arc {
	neighbors := s.neighbors(variable)
	sort.Slice(neighbors, func(i, j int) bool {
		if s.degree[neighbors[i]] != s.degree[neighbors[j]] {
			return s.degree[neighbors[i]] > s.degree[neighbors[j]]
		}
		return neighbors[i] < neighbors[j]
	})
	queue := make([]arc, 0, len(neighbors))
	for _, neighbor := range neighbors {
		queue = append(queue, arc{neighbor, variable})
		s.meter.charge(6, "arc-enqueue", neighbor, variable)
	}
	return queue
}

func (s *solver) ac3(queue []arc) map[int]bool {
	queued := map[arc]bool{}
	for _, item := range queue {
		queued[item] = true
	}
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		delete(queued, item)
		s.meter.charge(6, "arc-dequeue", item.x, item.y)
		s.meter.charge(6, "arc-revise", item.x, item.y)
		changed := false
		for _, xColor := range slices.Clone(s.domains[item.x]) {
			supported := false
			for _, yColor := range s.domains[item.y] {
				s.meter.charge(4, "inequality", item.x, xColor, item.y, yColor)
				if xColor != yColor {
					supported = true
					break
				}
			}
			if !supported {
				explanation := map[int]bool{}
				for _, original := range s.p.Variables[item.y].Domain {
					if original != xColor {
						for member := range s.explanations[item.y][original] {
							explanation[member] = true
							s.meter.charge(7, "explanation-union", member)
						}
					}
				}
				s.remove(item.x, xColor, explanation)
				s.meter.charge(5, "domain-empty-check", item.x)
				changed = true
				if len(s.domains[item.x]) == 0 {
					return s.wipeout(item.x)
				}
			}
		}
		if changed {
			neighbors := s.neighbors(item.x)
			sort.Slice(neighbors, func(i, j int) bool {
				if s.degree[neighbors[i]] != s.degree[neighbors[j]] {
					return s.degree[neighbors[i]] > s.degree[neighbors[j]]
				}
				return neighbors[i] < neighbors[j]
			})
			for _, neighbor := range neighbors {
				next := arc{neighbor, item.x}
				if neighbor != item.y && !queued[next] {
					queue = append(queue, next)
					queued[next] = true
					s.meter.charge(6, "arc-enqueue", next.x, next.y)
				}
			}
		}
	}
	return nil
}

func (s *solver) remove(variable, color int, explanation map[int]bool) {
	index := slices.Index(s.domains[variable], color)
	if index < 0 {
		return
	}
	s.domains[variable] = slices.Delete(s.domains[variable], index, index+1)
	s.explanations[variable][color] = cloneSet(explanation)
	s.meter.charge(5, "domain-delete", variable, color)
}
func (s *solver) wipeout(variable int) map[int]bool {
	out := map[int]bool{}
	for _, color := range s.p.Variables[variable].Domain {
		for member := range s.explanations[variable][color] {
			out[member] = true
			s.meter.charge(7, "wipeout-union", member)
		}
	}
	return out
}
func (s *solver) neighbors(variable int) []int {
	var out []int
	for _, e := range s.p.Edges {
		if e.Left == variable {
			out = append(out, e.Right)
		} else if e.Right == variable {
			out = append(out, e.Left)
		}
	}
	return out
}
func (s *solver) assignedEdgesConsistent(variable int) bool {
	for _, neighbor := range s.neighbors(variable) {
		if s.assignment[neighbor] >= 0 {
			s.meter.charge(4, "assigned-inequality", variable, s.assignment[variable], neighbor, s.assignment[neighbor])
			if s.assignment[neighbor] == s.assignment[variable] {
				return false
			}
		}
	}
	return true
}
func (s *solver) assignedNeighborSet(variable int) map[int]bool {
	out := map[int]bool{}
	for _, n := range s.neighbors(variable) {
		if s.assignment[n] >= 0 {
			out[n] = true
		}
	}
	return out
}
func (s *solver) completeConsistent() bool {
	for i, c := range s.assignment {
		if c < 0 || !contains(s.p.Variables[i].Domain, c) {
			return false
		}
	}
	for _, e := range s.p.Edges {
		if s.assignment[e.Left] == s.assignment[e.Right] {
			return false
		}
	}
	return true
}

func cloneDomains(source [][]int) [][]int {
	out := make([][]int, len(source))
	for i := range source {
		out[i] = slices.Clone(source[i])
	}
	return out
}
func cloneSet(source map[int]bool) map[int]bool {
	out := map[int]bool{}
	for k, v := range source {
		out[k] = v
	}
	return out
}
func cloneExplanations(source []map[int]map[int]bool) []map[int]map[int]bool {
	out := make([]map[int]map[int]bool, len(source))
	for i := range source {
		out[i] = map[int]map[int]bool{}
		for color, set := range source[i] {
			out[i][color] = cloneSet(set)
		}
	}
	return out
}
func contains(values []int, value int) bool { return slices.Contains(values, value) }

func parse(data []byte) (problem, error) {
	var p problem
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return problem{}, err
	}
	encoded, _ := json.Marshal(p)
	if !bytes.Equal(encoded, data) || p.Version != "finite-neq-csp/v1" || len(p.Variables) == 0 || len(p.Variables) > 8 || len(p.ColorAliases) == 0 || len(p.ColorAliases) > 4 {
		return problem{}, fmt.Errorf("invalid public CSP")
	}
	for _, v := range p.Variables {
		if len(v.Domain) == 0 {
			return problem{}, fmt.Errorf("empty domain")
		}
	}
	return p, nil
}
