// Package protocol implements the pure semantics of the finite-state protocol
// vocabulary. It has no dependency on the Nous engine or DSL.
package protocol

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Transition is one deterministic transition in a protocol machine.
type Transition struct {
	From  string
	Event string
	To    string
}

type transitionKey struct {
	state string
	event string
}

// Machine is a validated partial deterministic finite automaton.
type Machine struct {
	States      []string
	Events      []string
	Start       string
	Accepting   []string
	Transitions []Transition

	acceptSet  map[string]bool
	transition map[transitionKey]string
}

// Parse validates records and returns a machine in canonical order.
func Parse(records []string) (Machine, error) {
	states := map[string]bool{}
	events := map[string]bool{}
	accepting := map[string]bool{}
	transitions := map[transitionKey]string{}
	start := ""
	startCount := 0

	for _, record := range records {
		if record == "" || strings.IndexFunc(record, unicode.IsSpace) >= 0 {
			return Machine{}, fmt.Errorf("invalid record %q", record)
		}
		tag, value, ok := strings.Cut(record, ":")
		if !ok || value == "" {
			return Machine{}, fmt.Errorf("malformed record %q", record)
		}
		switch tag {
		case "state":
			if err := checkName(value); err != nil {
				return Machine{}, fmt.Errorf("state: %w", err)
			}
			states[value] = true
		case "event":
			if err := checkName(value); err != nil {
				return Machine{}, fmt.Errorf("event: %w", err)
			}
			events[value] = true
		case "start":
			if err := checkName(value); err != nil {
				return Machine{}, fmt.Errorf("start: %w", err)
			}
			startCount++
			start = value
		case "accept":
			if err := checkName(value); err != nil {
				return Machine{}, fmt.Errorf("accept: %w", err)
			}
			accepting[value] = true
		case "trans":
			left, to, ok := strings.Cut(value, ">")
			if !ok || strings.Contains(to, ">") {
				return Machine{}, fmt.Errorf("malformed transition %q", record)
			}
			from, event, ok := strings.Cut(left, ",")
			if !ok || strings.Contains(event, ",") {
				return Machine{}, fmt.Errorf("malformed transition %q", record)
			}
			for label, name := range map[string]string{"from": from, "event": event, "to": to} {
				if err := checkName(name); err != nil {
					return Machine{}, fmt.Errorf("transition %s: %w", label, err)
				}
			}
			key := transitionKey{state: from, event: event}
			if previous, exists := transitions[key]; exists && previous != to {
				return Machine{}, fmt.Errorf("conflicting transition %s,%s", from, event)
			}
			transitions[key] = to
		default:
			return Machine{}, fmt.Errorf("unknown record tag %q", tag)
		}
	}

	if startCount != 1 {
		return Machine{}, fmt.Errorf("expected exactly one start record, got %d", startCount)
	}
	if !states[start] {
		return Machine{}, fmt.Errorf("start state %q is undeclared", start)
	}
	for state := range accepting {
		if !states[state] {
			return Machine{}, fmt.Errorf("accepting state %q is undeclared", state)
		}
	}
	for key, to := range transitions {
		if !states[key.state] || !states[to] {
			return Machine{}, fmt.Errorf("transition %s,%s>%s references undeclared state", key.state, key.event, to)
		}
		if !events[key.event] {
			return Machine{}, fmt.Errorf("transition uses undeclared event %q", key.event)
		}
	}

	machine := Machine{
		States:      sortedKeys(states),
		Events:      sortedKeys(events),
		Start:       start,
		Accepting:   sortedKeys(accepting),
		acceptSet:   accepting,
		transition:  transitions,
		Transitions: make([]Transition, 0, len(transitions)),
	}
	for key, to := range transitions {
		machine.Transitions = append(machine.Transitions, Transition{From: key.state, Event: key.event, To: to})
	}
	sort.Slice(machine.Transitions, func(i, j int) bool {
		a, b := machine.Transitions[i], machine.Transitions[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Event != b.Event {
			return a.Event < b.Event
		}
		return a.To < b.To
	})
	return machine, nil
}

func checkName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid name %q", name)
	}
	return nil
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// Records returns the canonical encoding of m.
func (m Machine) Records() []string {
	out := make([]string, 0, len(m.States)+len(m.Events)+1+len(m.Accepting)+len(m.Transitions))
	for _, state := range m.States {
		out = append(out, "state:"+state)
	}
	for _, event := range m.Events {
		out = append(out, "event:"+event)
	}
	out = append(out, "start:"+m.Start)
	for _, state := range m.Accepting {
		out = append(out, "accept:"+state)
	}
	for _, transition := range m.Transitions {
		out = append(out, "trans:"+transition.From+","+transition.Event+">"+transition.To)
	}
	return out
}

// ReachableStates returns all states reachable from the start state.
func (m Machine) ReachableStates() []string {
	reachable := map[string]bool{m.Start: true}
	queue := []string{m.Start}
	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]
		for _, transition := range m.Transitions {
			if transition.From == state && !reachable[transition.To] {
				reachable[transition.To] = true
				queue = append(queue, transition.To)
			}
		}
	}
	return sortedKeys(reachable)
}

// RejectingTrapStates returns reachable states from which no accepting state
// can ever be reached.
func (m Machine) RejectingTrapStates() []string {
	canAccept := make(map[string]bool, len(m.Accepting))
	queue := append([]string(nil), m.Accepting...)
	for _, state := range queue {
		canAccept[state] = true
	}
	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]
		for _, transition := range m.Transitions {
			if transition.To == state && !canAccept[transition.From] {
				canAccept[transition.From] = true
				queue = append(queue, transition.From)
			}
		}
	}
	var traps []string
	for _, state := range m.ReachableStates() {
		if !canAccept[state] {
			traps = append(traps, state)
		}
	}
	return traps
}

// TrimUnreachable removes unreachable state-related records while retaining
// the declared event alphabet.
func (m Machine) TrimUnreachable() Machine {
	reachableList := m.ReachableStates()
	reachable := make(map[string]bool, len(reachableList))
	for _, state := range reachableList {
		reachable[state] = true
	}
	records := make([]string, 0)
	for _, state := range reachableList {
		records = append(records, "state:"+state)
	}
	for _, event := range m.Events {
		records = append(records, "event:"+event)
	}
	records = append(records, "start:"+m.Start)
	for _, state := range m.Accepting {
		if reachable[state] {
			records = append(records, "accept:"+state)
		}
	}
	for _, transition := range m.Transitions {
		if reachable[transition.From] && reachable[transition.To] {
			records = append(records, "trans:"+transition.From+","+transition.Event+">"+transition.To)
		}
	}
	trimmed, err := Parse(records)
	if err != nil {
		panic("protocol: internally invalid trim: " + err.Error())
	}
	return trimmed
}

// DropFirstTransition is a deterministic destructive transform used as a
// decoy in the relation-discovery experiment.
func (m Machine) DropFirstTransition() Machine {
	if len(m.Transitions) == 0 {
		return m
	}
	records := m.Records()
	needle := "trans:" + m.Transitions[0].From + "," + m.Transitions[0].Event + ">" + m.Transitions[0].To
	for i, record := range records {
		if record == needle {
			records = append(records[:i], records[i+1:]...)
			break
		}
	}
	dropped, err := Parse(records)
	if err != nil {
		panic("protocol: internally invalid transition removal: " + err.Error())
	}
	return dropped
}

// Accepts reports whether m accepts trace. Missing or unknown transitions
// enter the implicit rejecting sink.
func (m Machine) Accepts(trace []string) bool {
	state := m.Start
	sunk := false
	for _, event := range trace {
		if err := checkName(event); err != nil {
			return false
		}
		if sunk {
			continue
		}
		next, ok := m.transition[transitionKey{state: state, event: event}]
		if !ok {
			sunk = true
			continue
		}
		state = next
	}
	return !sunk && m.acceptSet[state]
}

type productState struct {
	a     string
	b     string
	aSink bool
	bSink bool
}

type productNode struct {
	state productState
	trace []string
}

// Compare decides accepted-trace language equivalence. If inequivalent,
// witness is the deterministic shortest trace accepted by exactly one side.
func Compare(a, b Machine) (equivalent bool, witness []string) {
	alphabetSet := map[string]bool{}
	for _, event := range a.Events {
		alphabetSet[event] = true
	}
	for _, event := range b.Events {
		alphabetSet[event] = true
	}
	alphabet := sortedKeys(alphabetSet)
	initial := productState{a: a.Start, b: b.Start}
	queue := []productNode{{state: initial}}
	visited := map[productState]bool{initial: true}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if accepting(a, node.state.a, node.state.aSink) != accepting(b, node.state.b, node.state.bSink) {
			return false, node.trace
		}
		for _, event := range alphabet {
			nextA, sinkA := step(a, node.state.a, node.state.aSink, event)
			nextB, sinkB := step(b, node.state.b, node.state.bSink, event)
			next := productState{a: nextA, b: nextB, aSink: sinkA, bSink: sinkB}
			if visited[next] {
				continue
			}
			visited[next] = true
			trace := append(append([]string(nil), node.trace...), event)
			queue = append(queue, productNode{state: next, trace: trace})
		}
	}
	return true, nil
}

func accepting(machine Machine, state string, sink bool) bool {
	return !sink && machine.acceptSet[state]
}

func step(machine Machine, state string, sink bool, event string) (string, bool) {
	if sink {
		return "", true
	}
	next, ok := machine.transition[transitionKey{state: state, event: event}]
	if !ok {
		return "", true
	}
	return next, false
}

// SameEncoding compares canonical protocol encodings.
func SameEncoding(a, b Machine) bool {
	return strings.Join(a.Records(), "\x00") == strings.Join(b.Records(), "\x00")
}
