package nogoodbaseline

import (
	"fmt"
	"slices"
)

// Chronological performs descriptor-order depth-first search with only checks
// against already assigned incident edges.
func Chronological(data []byte, decision Literal) (Result, error) {
	p, err := parse(data)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	m := meter{&result}
	assignment := make([]int, len(p.Variables))
	for index := range assignment {
		assignment[index] = -1
	}
	for _, literal := range p.Assignment {
		assignment[literal.Variable] = literal.Color
	}
	if decision.Variable < 0 || decision.Variable >= len(p.Variables) || !contains(p.Variables[decision.Variable].Domain, decision.Color) || assignment[decision.Variable] >= 0 {
		return Result{}, fmt.Errorf("invalid supplied decision")
	}
	m.charge(2, "decision-domain-read", decision.Variable, decision.Color)
	m.charge(3, "decision-propose", decision.Variable, decision.Color)
	assignment[decision.Variable] = decision.Color
	m.charge(3, "decision-bind", decision.Variable, decision.Color)
	if chronologicalConsistent(p, assignment, decision.Variable, m) {
		result.Witness = chronologicalSearch(p, assignment, m)
		result.Satisfied = result.Witness != nil
	}
	assignment[decision.Variable] = -1
	m.charge(3, "decision-unbind", decision.Variable, decision.Color)
	m.charge(11, "terminal-classification")
	m.charge(12, "terminal-record-write")
	return result, nil
}

func chronologicalSearch(p problem, assignment []int, m meter) []int {
	variable := -1
	for index, color := range assignment {
		if color < 0 {
			variable = index
			break
		}
	}
	if variable < 0 {
		m.charge(11, "witness-check")
		return slices.Clone(assignment)
	}
	for _, color := range p.Variables[variable].Domain {
		m.charge(3, "assignment-propose", variable, color)
		assignment[variable] = color
		m.charge(3, "assignment-bind", variable, color)
		if chronologicalConsistent(p, assignment, variable, m) {
			if witness := chronologicalSearch(p, assignment, m); witness != nil {
				return witness
			}
		}
		assignment[variable] = -1
		m.charge(3, "assignment-unbind", variable, color)
	}
	return nil
}

func chronologicalConsistent(p problem, assignment []int, variable int, m meter) bool {
	for _, edge := range p.Edges {
		neighbor := -1
		if edge.Left == variable {
			neighbor = edge.Right
		} else if edge.Right == variable {
			neighbor = edge.Left
		}
		if neighbor >= 0 && assignment[neighbor] >= 0 {
			m.charge(4, "assigned-inequality", variable, assignment[variable], neighbor, assignment[neighbor])
			if assignment[variable] == assignment[neighbor] {
				return false
			}
		}
	}
	return true
}

// ForwardChecking removes the assigned value from unassigned neighbors after
// every decision and uses MRV, static degree, then descriptor position.
func ForwardChecking(data []byte, decision Literal) (Result, error) {
	p, err := parse(data)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	s := newSolver(p, &result)
	for _, literal := range p.Assignment {
		s.assignment[literal.Variable] = literal.Color
	}
	if decision.Variable < 0 || decision.Variable >= len(p.Variables) || !contains(s.domains[decision.Variable], decision.Color) || s.assignment[decision.Variable] >= 0 {
		return Result{}, fmt.Errorf("invalid supplied decision")
	}
	s.meter.charge(2, "decision-domain-read", decision.Variable, decision.Color)
	s.meter.charge(3, "decision-propose", decision.Variable, decision.Color)
	s.assignment[decision.Variable] = decision.Color
	s.meter.charge(3, "decision-bind", decision.Variable, decision.Color)
	rootDomains := cloneDomains(s.domains)
	for _, other := range slices.Clone(s.domains[decision.Variable]) {
		if other != decision.Color {
			s.domains[decision.Variable] = deleteColor(s.domains[decision.Variable], other)
			s.meter.charge(5, "domain-delete", decision.Variable, other)
			s.meter.charge(5, "domain-empty-check", decision.Variable)
		}
	}
	if s.assignedEdgesConsistent(decision.Variable) && forwardPropagate(s, decision.Variable, decision.Color) {
		result.Witness = forwardSearch(s)
		result.Satisfied = result.Witness != nil
	}
	for variable := range rootDomains {
		for _, color := range rootDomains[variable] {
			if !contains(s.domains[variable], color) {
				s.meter.charge(5, "domain-restore", variable, color)
			}
		}
	}
	s.assignment[decision.Variable] = -1
	s.meter.charge(3, "decision-unbind", decision.Variable, decision.Color)
	s.meter.charge(11, "terminal-classification")
	s.meter.charge(12, "terminal-record-write")
	return result, nil
}

func forwardSearch(s *solver) []int {
	variable := s.chooseVariable()
	if variable < 0 {
		s.meter.charge(11, "witness-check")
		if s.completeConsistent() {
			return slices.Clone(s.assignment)
		}
		return nil
	}
	for _, color := range slices.Clone(s.domains[variable]) {
		domains := cloneDomains(s.domains)
		s.meter.charge(3, "assignment-propose", variable, color)
		s.assignment[variable] = color
		s.meter.charge(3, "assignment-bind", variable, color)
		for _, other := range slices.Clone(s.domains[variable]) {
			if other != color {
				s.domains[variable] = deleteColor(s.domains[variable], other)
				s.meter.charge(5, "domain-delete", variable, other)
				s.meter.charge(5, "domain-empty-check", variable)
			}
		}
		if s.assignedEdgesConsistent(variable) && forwardPropagate(s, variable, color) {
			if witness := forwardSearch(s); witness != nil {
				return witness
			}
		}
		for index := range domains {
			for _, restored := range domains[index] {
				if !contains(s.domains[index], restored) {
					s.meter.charge(5, "domain-restore", index, restored)
				}
			}
		}
		s.domains = domains
		s.assignment[variable] = -1
		s.meter.charge(3, "assignment-unbind", variable, color)
	}
	return nil
}

func forwardPropagate(s *solver, variable, color int) bool {
	for _, neighbor := range s.neighbors(variable) {
		if s.assignment[neighbor] >= 0 {
			continue
		}
		s.meter.charge(4, "forward-inequality", variable, color, neighbor, color)
		if contains(s.domains[neighbor], color) {
			s.domains[neighbor] = deleteColor(s.domains[neighbor], color)
			s.meter.charge(5, "domain-delete", neighbor, color)
			s.meter.charge(5, "domain-empty-check", neighbor)
			if len(s.domains[neighbor]) == 0 {
				return false
			}
		}
	}
	return true
}

func deleteColor(domain []int, color int) []int {
	index := slices.Index(domain, color)
	if index < 0 {
		return domain
	}
	return slices.Delete(domain, index, index+1)
}
