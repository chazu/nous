// Package actionrelations implements the bounded, pure semantics for the
// guarded-action-relations vocabulary. It contains no search, learning,
// fixture, experiment, engine, Store, DSL, or oracle code.
package actionrelations

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
)

const (
	StateVersion          = "finite-action-state/v1"
	ActionVersion         = "finite-action/v1"
	SemanticActionVersion = "finite-action-semantic/v1"
	OccurrenceVersion     = "action-occurrence/v1"
	WorldVersion          = "finite-action-world-core/v1"
	LocalFactsVersion     = "action-local-facts/v1"
	PatternVersion        = "action-relation-pattern/v1"
	GuardVersion          = "action-guard/v1"
	RelationVersion       = "guarded-action-relation/v1"
	ArtifactVersion       = "guarded-action-artifact/v1"
	CertificateVersion    = "local-diamond-certificate/v1"
	MaxCells              = 3
	MaxCellValue          = 3
	MaxEvents             = 8
	MaxActions            = 8
)

var ErrInvalid = errors.New("invalid action-relations value")

type Cell struct {
	Name  string
	Value int
}

type State struct {
	Cells  []Cell
	Events []string
}

func ParseState(data []byte) (State, error) {
	v, err := decodeOne(data)
	if err != nil {
		return State{}, err
	}
	row, ok := v.([]any)
	if !ok || len(row) != 3 || row[0] != StateVersion {
		return State{}, ErrInvalid
	}
	cellRows, ok := row[1].([]any)
	if !ok {
		return State{}, ErrInvalid
	}
	eventRows, ok := row[2].([]any)
	if !ok {
		return State{}, ErrInvalid
	}
	state := State{Cells: make([]Cell, len(cellRows)), Events: make([]string, len(eventRows))}
	for index, raw := range cellRows {
		item, ok := raw.([]any)
		if !ok || len(item) != 2 {
			return State{}, ErrInvalid
		}
		name, nameOK := item[0].(string)
		value, valueOK := exactInt(item[1])
		if !nameOK || !valueOK {
			return State{}, ErrInvalid
		}
		state.Cells[index] = Cell{Name: name, Value: value}
	}
	for index, raw := range eventRows {
		symbol, ok := raw.(string)
		if !ok {
			return State{}, ErrInvalid
		}
		state.Events[index] = symbol
	}
	if err := state.Validate(); err != nil {
		return State{}, err
	}
	canonical, _ := state.CanonicalJSON()
	if !bytes.Equal(canonical, data) {
		return State{}, ErrInvalid
	}
	return state, nil
}

func (s State) Validate() error {
	if len(s.Cells) < 1 || len(s.Cells) > MaxCells || len(s.Events) > MaxEvents {
		return ErrInvalid
	}
	previous := ""
	for index, cell := range s.Cells {
		if !validCellOrRole(cell.Name) || cell.Value < 0 || cell.Value > MaxCellValue || index > 0 && cell.Name <= previous {
			return ErrInvalid
		}
		previous = cell.Name
	}
	for _, event := range s.Events {
		if !validIdentifier(event) {
			return ErrInvalid
		}
	}
	return nil
}

func (s State) CanonicalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.wire())
}

func (s State) wire() []any {
	cells := make([]any, len(s.Cells))
	for index, cell := range s.Cells {
		cells[index] = []any{cell.Name, cell.Value}
	}
	events := make([]any, len(s.Events))
	for index, event := range s.Events {
		events[index] = event
	}
	return []any{StateVersion, cells, events}
}

func (s State) Digest() (string, error) { return digestCanonical(s.CanonicalJSON()) }

func (s State) Value(name string) (int, bool) {
	index, found := slices.BinarySearchFunc(s.Cells, name, func(cell Cell, target string) int {
		return bytes.Compare([]byte(cell.Name), []byte(target))
	})
	if !found {
		return 0, false
	}
	return s.Cells[index].Value, true
}

type Action struct {
	Name   string
	Kind   string
	X      string
	Y      string
	N      int
	Symbol string
}

func ParseAction(data []byte) (Action, error) {
	v, err := decodeOne(data)
	if err != nil {
		return Action{}, err
	}
	row, ok := v.([]any)
	if !ok || len(row) != 7 || row[0] != ActionVersion {
		return Action{}, ErrInvalid
	}
	name, a := row[1].(string)
	kind, b := row[2].(string)
	x, c := row[3].(string)
	y, d := row[4].(string)
	n, e := exactInt(row[5])
	symbol, f := row[6].(string)
	if !(a && b && c && d && e && f) {
		return Action{}, ErrInvalid
	}
	action := Action{Name: name, Kind: kind, X: x, Y: y, N: n, Symbol: symbol}
	if err := action.Validate(); err != nil {
		return Action{}, err
	}
	canonical, _ := action.CanonicalJSON()
	if !bytes.Equal(canonical, data) {
		return Action{}, ErrInvalid
	}
	return action, nil
}

func (a Action) Validate() error {
	if !validIdentifier(a.Name) {
		return ErrInvalid
	}
	return validateActionFields(a.Kind, a.X, a.Y, a.N, a.Symbol)
}

func (a Action) CanonicalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal([]any{ActionVersion, a.Name, a.Kind, a.X, a.Y, a.N, a.Symbol})
}

type SemanticAction struct {
	Kind   string
	XRole  string
	YRole  string
	N      int
	Symbol string
}

func ParseSemanticAction(data []byte) (SemanticAction, error) {
	v, err := decodeOne(data)
	if err != nil {
		return SemanticAction{}, err
	}
	action, err := semanticActionFromWire(v)
	if err != nil {
		return SemanticAction{}, err
	}
	canonical, _ := action.CanonicalJSON()
	if !bytes.Equal(canonical, data) {
		return SemanticAction{}, ErrInvalid
	}
	return action, nil
}

func semanticActionFromWire(v any) (SemanticAction, error) {
	row, ok := v.([]any)
	if !ok || len(row) != 6 || row[0] != SemanticActionVersion {
		return SemanticAction{}, ErrInvalid
	}
	kind, a := row[1].(string)
	x, b := row[2].(string)
	y, c := row[3].(string)
	n, d := exactInt(row[4])
	symbol, e := row[5].(string)
	if !(a && b && c && d && e) {
		return SemanticAction{}, ErrInvalid
	}
	action := SemanticAction{Kind: kind, XRole: x, YRole: y, N: n, Symbol: symbol}
	if err := action.Validate(); err != nil {
		return SemanticAction{}, err
	}
	return action, nil
}

func (a SemanticAction) Validate() error {
	if a.XRole != "" && !validRole(a.XRole) || a.YRole != "" && !validRole(a.YRole) {
		return ErrInvalid
	}
	return validateActionFields(a.Kind, a.XRole, a.YRole, a.N, a.Symbol)
}

func (a SemanticAction) CanonicalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(a.wire())
}

func (a SemanticAction) wire() []any {
	return []any{SemanticActionVersion, a.Kind, a.XRole, a.YRole, a.N, a.Symbol}
}

func (a SemanticAction) Digest() (string, error) { return digestCanonical(a.CanonicalJSON()) }

func (a Action) Semantic(roles map[string]string) (SemanticAction, error) {
	if err := a.Validate(); err != nil {
		return SemanticAction{}, err
	}
	semantic := SemanticAction{Kind: a.Kind, N: a.N, Symbol: a.Symbol}
	if a.X != "" {
		var ok bool
		semantic.XRole, ok = roles[a.X]
		if !ok {
			return SemanticAction{}, ErrInvalid
		}
	}
	if a.Y != "" {
		var ok bool
		semantic.YRole, ok = roles[a.Y]
		if !ok {
			return SemanticAction{}, ErrInvalid
		}
	}
	return semantic, semantic.Validate()
}

type Occurrence struct {
	Action  SemanticAction
	Ordinal int
}

func ParseOccurrence(data []byte) (Occurrence, error) {
	v, err := decodeOne(data)
	if err != nil {
		return Occurrence{}, err
	}
	row, ok := v.([]any)
	if !ok || len(row) != 3 || row[0] != OccurrenceVersion {
		return Occurrence{}, ErrInvalid
	}
	action, err := semanticActionFromWire(row[1])
	if err != nil {
		return Occurrence{}, err
	}
	ordinal, ok := exactInt(row[2])
	if !ok || ordinal < 0 || ordinal >= MaxActions {
		return Occurrence{}, ErrInvalid
	}
	occurrence := Occurrence{Action: action, Ordinal: ordinal}
	canonical, _ := occurrence.CanonicalJSON()
	if !bytes.Equal(canonical, data) {
		return Occurrence{}, ErrInvalid
	}
	return occurrence, nil
}

func (o Occurrence) CanonicalJSON() ([]byte, error) {
	if err := o.Action.Validate(); err != nil || o.Ordinal < 0 || o.Ordinal >= MaxActions {
		return nil, ErrInvalid
	}
	return json.Marshal(o.wire())
}

func (o Occurrence) wire() []any { return []any{OccurrenceVersion, o.Action.wire(), o.Ordinal} }

func (o Occurrence) Digest() (string, error) { return digestCanonical(o.CanonicalJSON()) }

func AssignOccurrences(actions []SemanticAction) ([]Occurrence, error) {
	if len(actions) == 0 || len(actions) > MaxActions {
		return nil, ErrInvalid
	}
	type encodedAction struct {
		action SemanticAction
		bytes  []byte
	}
	encoded := make([]encodedAction, len(actions))
	for index, action := range actions {
		data, err := action.CanonicalJSON()
		if err != nil {
			return nil, err
		}
		encoded[index] = encodedAction{action: action, bytes: data}
	}
	slices.SortFunc(encoded, func(a, b encodedAction) int { return bytes.Compare(a.bytes, b.bytes) })
	result := make([]Occurrence, len(encoded))
	ordinal := 0
	for index, item := range encoded {
		if index > 0 && !bytes.Equal(encoded[index-1].bytes, item.bytes) {
			ordinal = 0
		}
		result[index] = Occurrence{Action: item.action, Ordinal: ordinal}
		ordinal++
	}
	return result, nil
}

type World struct {
	State   State
	Actions []Action
}

type NormalizedWorld struct {
	State       State
	Actions     []SemanticAction
	Occurrences []Occurrence
	RoleMap     map[string]string
}

func (w World) Normalize() (NormalizedWorld, error) {
	if err := w.State.Validate(); err != nil || len(w.Actions) == 0 || len(w.Actions) > MaxActions {
		return NormalizedWorld{}, ErrInvalid
	}
	names := map[string]bool{}
	for _, action := range w.Actions {
		if err := action.Validate(); err != nil || names[action.Name] {
			return NormalizedWorld{}, ErrInvalid
		}
		names[action.Name] = true
		for _, cell := range []string{action.X, action.Y} {
			if cell != "" {
				if _, ok := w.State.Value(cell); !ok {
					return NormalizedWorld{}, ErrInvalid
				}
			}
		}
	}
	permutations := rolePermutations(len(w.State.Cells))
	var best []byte
	var winner NormalizedWorld
	for _, permutation := range permutations {
		roles := make(map[string]string, len(w.State.Cells))
		cells := make([]Cell, len(w.State.Cells))
		for index, cell := range w.State.Cells {
			role := "c" + strconv.Itoa(permutation[index])
			roles[cell.Name] = role
			cells[permutation[index]] = Cell{Name: role, Value: cell.Value}
		}
		actions := make([]SemanticAction, len(w.Actions))
		for index, action := range w.Actions {
			semantic, err := action.Semantic(roles)
			if err != nil {
				return NormalizedWorld{}, err
			}
			actions[index] = semantic
		}
		slices.SortFunc(actions, compareSemanticActions)
		state := State{Cells: cells, Events: slices.Clone(w.State.Events)}
		wire := []any{WorldVersion, state.wire(), semanticActionWires(actions)}
		encoded, _ := json.Marshal(wire)
		if best == nil || bytes.Compare(encoded, best) < 0 {
			occurrences, err := AssignOccurrences(actions)
			if err != nil {
				return NormalizedWorld{}, err
			}
			best = encoded
			winner = NormalizedWorld{State: state, Actions: slices.Clone(actions), Occurrences: occurrences, RoleMap: roles}
		}
	}
	return winner, nil
}

func (w NormalizedWorld) CanonicalJSON() ([]byte, error) {
	if err := w.State.Validate(); err != nil || len(w.Actions) == 0 || len(w.Actions) > MaxActions {
		return nil, ErrInvalid
	}
	for index, cell := range w.State.Cells {
		if cell.Name != "c"+strconv.Itoa(index) {
			return nil, ErrInvalid
		}
	}
	actions := slices.Clone(w.Actions)
	for _, action := range actions {
		if err := action.Validate(); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(actions, compareSemanticActions)
	return json.Marshal([]any{WorldVersion, w.State.wire(), semanticActionWires(actions)})
}

func (w NormalizedWorld) Digest() (string, error) { return digestCanonical(w.CanonicalJSON()) }

func semanticActionWires(actions []SemanticAction) []any {
	rows := make([]any, len(actions))
	for index, action := range actions {
		rows[index] = action.wire()
	}
	return rows
}

func compareSemanticActions(a, b SemanticAction) int {
	left, _ := a.CanonicalJSON()
	right, _ := b.CanonicalJSON()
	return bytes.Compare(left, right)
}

func rolePermutations(n int) [][]int {
	base := make([]int, n)
	for index := range base {
		base[index] = index
	}
	var result [][]int
	var visit func(int)
	visit = func(index int) {
		if index == n {
			result = append(result, slices.Clone(base))
			return
		}
		for candidate := index; candidate < n; candidate++ {
			base[index], base[candidate] = base[candidate], base[index]
			visit(index + 1)
			base[index], base[candidate] = base[candidate], base[index]
		}
	}
	visit(0)
	return result
}

func validateActionFields(kind, x, y string, n int, symbol string) error {
	validX := x != "" && validCellOrRole(x)
	validY := y != "" && validCellOrRole(y)
	switch kind {
	case "add":
		if !validX || y != "" || !oneInt(n, -2, -1, 1, 2) || symbol != "" {
			return ErrInvalid
		}
	case "set":
		if !validX || y != "" || n < 0 || n > 3 || symbol != "" {
			return ErrInvalid
		}
	case "transfer":
		if !validX || !validY || x == y || !oneInt(n, 1, 2) || symbol != "" {
			return ErrInvalid
		}
	case "swap":
		if !validX || !validY || x == y || n != 0 || symbol != "" {
			return ErrInvalid
		}
	case "claim", "release":
		if !validX || y != "" || n != 0 || symbol != "" {
			return ErrInvalid
		}
	case "check":
		if !validX || y != "" || n < 0 || n > 3 || symbol != "" {
			return ErrInvalid
		}
	case "emit":
		if x != "" || y != "" || n != 0 || !validIdentifier(symbol) {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}

func validIdentifier(value string) bool {
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

func validRole(value string) bool {
	return len(value) == 2 && value[0] == 'c' && value[1] >= '0' && value[1] <= '2'
}

func validCellOrRole(value string) bool { return validIdentifier(value) || validRole(value) }

func oneInt(value int, allowed ...int) bool { return slices.Contains(allowed, value) }

func decodeOne(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", ErrInvalid, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, ErrInvalid
	}
	return value, nil
}

func exactInt(value any) (int, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	integer, err := strconv.ParseInt(string(number), 10, 32)
	if err != nil || strconv.FormatInt(integer, 10) != string(number) {
		return 0, false
	}
	return int(integer), true
}

func digestCanonical(data []byte, err error) (string, error) {
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == string(bytes.ToLower([]byte(value)))
}
