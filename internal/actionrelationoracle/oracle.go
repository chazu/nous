// Package actionrelationoracle is an executable specification independent of
// the production actionrelations package. It deliberately duplicates parsing
// and transition logic so agreement tests can catch shared assumptions.
package actionrelationoracle

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

var ErrInvalid = errors.New("invalid oracle input")

type cell struct {
	name  string
	value int
}

type state struct {
	cells  []cell
	events []string
}

type action struct {
	kind   string
	x      string
	y      string
	n      int
	symbol string
}

type Transition struct {
	Applicable bool
	State      []byte
}

type Observation struct {
	Label string
	AB    []byte
	BA    []byte
}

func ValidateState(data []byte) error {
	_, err := parseState(data)
	return err
}

func ValidateAction(data []byte) error {
	_, err := parseAction(data)
	return err
}

func Apply(stateJSON, actionJSON []byte) (Transition, error) {
	s, err := parseState(stateJSON)
	if err != nil {
		return Transition{}, err
	}
	a, err := parseAction(actionJSON)
	if err != nil {
		return Transition{}, err
	}
	next, ok := transition(s, a)
	if !ok {
		return Transition{Applicable: false, State: bytes.Clone(stateJSON)}, nil
	}
	data, err := encodeState(next)
	return Transition{Applicable: true, State: data}, err
}

// Observe implements the frozen total classification table and never executes
// a transition represented by an n/a cell in that table.
func Observe(stateJSON, aJSON, bJSON []byte) (Observation, error) {
	s, err := parseState(stateJSON)
	if err != nil {
		return Observation{}, err
	}
	a, err := parseAction(aJSON)
	if err != nil {
		return Observation{}, err
	}
	b, err := parseAction(bJSON)
	if err != nil {
		return Observation{}, err
	}
	sa, aInitial := transition(s, a)
	sb, bInitial := transition(s, b)
	if !aInitial && !bInitial {
		return Observation{Label: "inapplicable"}, nil
	}
	if !aInitial {
		_, enabled := transition(sb, a)
		if enabled {
			return Observation{Label: "b-enables-a"}, nil
		}
		return Observation{Label: "inapplicable"}, nil
	}
	if !bInitial {
		_, enabled := transition(sa, b)
		if enabled {
			return Observation{Label: "a-enables-b"}, nil
		}
		return Observation{Label: "inapplicable"}, nil
	}
	sab, bAfterA := transition(sa, b)
	sba, aAfterB := transition(sb, a)
	if !bAfterA && !aAfterB {
		return Observation{Label: "mutual-disables"}, nil
	}
	if !bAfterA {
		return Observation{Label: "a-disables-b"}, nil
	}
	if !aAfterB {
		return Observation{Label: "b-disables-a"}, nil
	}
	ab, err := encodeState(sab)
	if err != nil {
		return Observation{}, err
	}
	ba, err := encodeState(sba)
	if err != nil {
		return Observation{}, err
	}
	label := "conflicts"
	if bytes.Equal(ab, ba) {
		label = "commutes"
	}
	return Observation{Label: label, AB: ab, BA: ba}, nil
}

func transition(s state, a action) (state, bool) {
	next := cloneState(s)
	x, xi, xOK := lookup(s, a.x)
	y, yi, yOK := lookup(s, a.y)
	switch a.kind {
	case "add":
		if !xOK || x+a.n < 0 || x+a.n > 3 {
			return s, false
		}
		next.cells[xi].value = x + a.n
	case "set":
		if !xOK {
			return s, false
		}
		next.cells[xi].value = a.n
	case "transfer":
		if !xOK || !yOK || x < a.n || y+a.n > 3 {
			return s, false
		}
		next.cells[xi].value = x - a.n
		next.cells[yi].value = y + a.n
	case "swap":
		if !xOK || !yOK {
			return s, false
		}
		next.cells[xi].value, next.cells[yi].value = y, x
	case "claim":
		if !xOK || x != 0 {
			return s, false
		}
		next.cells[xi].value = 1
	case "release":
		if !xOK || x != 1 {
			return s, false
		}
		next.cells[xi].value = 0
	case "check":
		if !xOK || x != a.n {
			return s, false
		}
	case "emit":
		if len(s.events) >= 8 {
			return s, false
		}
		next.events = append(next.events, a.symbol)
	default:
		return s, false
	}
	return next, true
}

func parseState(data []byte) (state, error) {
	v, err := decode(data)
	if err != nil {
		return state{}, err
	}
	row, ok := v.([]any)
	if !ok || len(row) != 3 || row[0] != "finite-action-state/v1" {
		return state{}, ErrInvalid
	}
	cells, ok := row[1].([]any)
	if !ok || len(cells) < 1 || len(cells) > 3 {
		return state{}, ErrInvalid
	}
	events, ok := row[2].([]any)
	if !ok || len(events) > 8 {
		return state{}, ErrInvalid
	}
	s := state{cells: make([]cell, len(cells)), events: make([]string, len(events))}
	previous := ""
	for i, raw := range cells {
		pair, ok := raw.([]any)
		if !ok || len(pair) != 2 {
			return state{}, ErrInvalid
		}
		name, nameOK := pair[0].(string)
		value, valueOK := integer(pair[1])
		if !nameOK || !valueOK || !role(name) || value < 0 || value > 3 || i > 0 && name <= previous {
			return state{}, ErrInvalid
		}
		s.cells[i], previous = cell{name: name, value: value}, name
	}
	for i, raw := range events {
		symbol, ok := raw.(string)
		if !ok || !identifier(symbol) {
			return state{}, ErrInvalid
		}
		s.events[i] = symbol
	}
	canonical, _ := encodeState(s)
	if !bytes.Equal(canonical, data) {
		return state{}, ErrInvalid
	}
	return s, nil
}

func parseAction(data []byte) (action, error) {
	v, err := decode(data)
	if err != nil {
		return action{}, err
	}
	row, ok := v.([]any)
	if !ok || len(row) != 6 || row[0] != "finite-action-semantic/v1" {
		return action{}, ErrInvalid
	}
	kind, aOK := row[1].(string)
	x, xOK := row[2].(string)
	y, yOK := row[3].(string)
	n, nOK := integer(row[4])
	symbol, sOK := row[5].(string)
	if !aOK || !xOK || !yOK || !nOK || !sOK {
		return action{}, ErrInvalid
	}
	a := action{kind: kind, x: x, y: y, n: n, symbol: symbol}
	validX, validY := role(x), role(y)
	valid := false
	switch kind {
	case "add":
		valid = validX && y == "" && (n == -2 || n == -1 || n == 1 || n == 2) && symbol == ""
	case "set", "check":
		valid = validX && y == "" && n >= 0 && n <= 3 && symbol == ""
	case "transfer":
		valid = validX && validY && x != y && (n == 1 || n == 2) && symbol == ""
	case "swap":
		valid = validX && validY && x != y && n == 0 && symbol == ""
	case "claim", "release":
		valid = validX && y == "" && n == 0 && symbol == ""
	case "emit":
		valid = x == "" && y == "" && n == 0 && identifier(symbol)
	}
	if !valid {
		return action{}, ErrInvalid
	}
	canonical, _ := encodeAction(a)
	if !bytes.Equal(canonical, data) {
		return action{}, ErrInvalid
	}
	return a, nil
}

func encodeState(s state) ([]byte, error) {
	cells := make([]any, len(s.cells))
	for i, cell := range s.cells {
		cells[i] = []any{cell.name, cell.value}
	}
	events := make([]any, len(s.events))
	for i, event := range s.events {
		events[i] = event
	}
	return json.Marshal([]any{"finite-action-state/v1", cells, events})
}

func encodeAction(a action) ([]byte, error) {
	return json.Marshal([]any{"finite-action-semantic/v1", a.kind, a.x, a.y, a.n, a.symbol})
}

func lookup(s state, name string) (int, int, bool) {
	for i, cell := range s.cells {
		if cell.name == name {
			return cell.value, i, true
		}
	}
	return 0, 0, false
}

func cloneState(s state) state {
	return state{cells: append([]cell(nil), s.cells...), events: append([]string(nil), s.events...)}
}

func decode(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var v any
	if err := decoder.Decode(&v); err != nil {
		return nil, ErrInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, ErrInvalid
	}
	return v, nil
}

func integer(value any) (int, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(string(number))
	return i, err == nil && strconv.Itoa(i) == string(number)
}

func role(value string) bool {
	return len(value) == 2 && value[0] == 'c' && value[1] >= '0' && value[1] <= '2'
}

func identifier(value string) bool {
	if len(value) < 1 || len(value) > 8 {
		return false
	}
	for index, char := range []byte(value) {
		if index == 0 && (char < 'a' || char > 'z') || index > 0 && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}
