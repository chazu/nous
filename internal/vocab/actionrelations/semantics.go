package actionrelations

import (
	"bytes"
	"encoding/json"
	"slices"
	"strconv"
)

// Applicable reports whether one action may run at one state. Invalid input is
// distinct from a well-formed action that names a cell absent from the state.
func Applicable(state State, action SemanticAction) (bool, error) {
	if err := state.Validate(); err != nil {
		return false, err
	}
	if err := action.Validate(); err != nil {
		return false, err
	}
	x, xOK := state.Value(action.XRole)
	y, yOK := state.Value(action.YRole)
	switch action.Kind {
	case "add":
		return xOK && x+action.N >= 0 && x+action.N <= MaxCellValue, nil
	case "set", "swap":
		return action.Kind == "set" && xOK || action.Kind == "swap" && xOK && yOK, nil
	case "transfer":
		return xOK && yOK && x >= action.N && y+action.N <= MaxCellValue, nil
	case "claim":
		return xOK && x == 0, nil
	case "release":
		return xOK && x == 1, nil
	case "check":
		return xOK && x == action.N, nil
	case "emit":
		return len(state.Events) < MaxEvents, nil
	default:
		return false, ErrInvalid
	}
}

// Apply executes exactly one semantic action. It returns "inapplicable" with
// the original state when the action is valid but cannot run.
func Apply(state State, action SemanticAction) (State, string, error) {
	applicable, err := Applicable(state, action)
	if err != nil {
		return State{}, "", err
	}
	if !applicable {
		return state, "inapplicable", nil
	}
	next := State{Cells: slices.Clone(state.Cells), Events: slices.Clone(state.Events)}
	set := func(role string, value int) {
		index, _ := slices.BinarySearchFunc(next.Cells, role, func(cell Cell, target string) int {
			return bytes.Compare([]byte(cell.Name), []byte(target))
		})
		next.Cells[index].Value = value
	}
	x, _ := state.Value(action.XRole)
	y, _ := state.Value(action.YRole)
	switch action.Kind {
	case "add":
		set(action.XRole, x+action.N)
	case "set":
		set(action.XRole, action.N)
	case "transfer":
		set(action.XRole, x-action.N)
		set(action.YRole, y+action.N)
	case "swap":
		set(action.XRole, y)
		set(action.YRole, x)
	case "claim":
		set(action.XRole, 1)
	case "release":
		set(action.XRole, 0)
	case "check":
	case "emit":
		next.Events = append(next.Events, action.Symbol)
	}
	return next, "applied", nil
}

func CompareStates(left, right State) (int, error) {
	a, err := left.CanonicalJSON()
	if err != nil {
		return 0, err
	}
	b, err := right.CanonicalJSON()
	if err != nil {
		return 0, err
	}
	return bytes.Compare(a, b), nil
}

type LocalFacts struct {
	StateDigest      string
	OccurrenceDigest string
	Kind             string
	PrimaryRole      int
	SecondaryRole    int
	ArgumentPresent  bool
	ArgumentValue    int
	Symbol           string
	ReadRoles        []int
	WriteRoles       []int
	PrimaryValue     int
	SecondaryValue   int
	TraceLength      int
}

func ParseLocalFacts(data []byte) (LocalFacts, error) {
	v, err := decodeOne(data)
	if err != nil {
		return LocalFacts{}, err
	}
	row, ok := v.([]any)
	if !ok || len(row) != 14 || row[0] != LocalFactsVersion {
		return LocalFacts{}, ErrInvalid
	}
	stateDigest, a := row[1].(string)
	occurrenceDigest, b := row[2].(string)
	kind, c := row[3].(string)
	primary, d := exactInt(row[4])
	secondary, e := exactInt(row[5])
	argumentPresent, f := row[6].(bool)
	argumentValue, g := exactInt(row[7])
	symbol, h := row[8].(string)
	reads, i := intList(row[9])
	writes, j := intList(row[10])
	primaryValue, k := exactInt(row[11])
	secondaryValue, l := exactInt(row[12])
	traceLength, m := exactInt(row[13])
	if !(a && b && c && d && e && f && g && h && i && j && k && l && m) {
		return LocalFacts{}, ErrInvalid
	}
	facts := LocalFacts{stateDigest, occurrenceDigest, kind, primary, secondary, argumentPresent, argumentValue, symbol, reads, writes, primaryValue, secondaryValue, traceLength}
	if err := facts.Validate(); err != nil {
		return LocalFacts{}, err
	}
	canonical, _ := facts.CanonicalJSON()
	if !bytes.Equal(canonical, data) {
		return LocalFacts{}, ErrInvalid
	}
	return facts, nil
}

func Facts(state State, occurrence Occurrence) (LocalFacts, error) {
	if err := state.Validate(); err != nil {
		return LocalFacts{}, err
	}
	if _, err := occurrence.CanonicalJSON(); err != nil {
		return LocalFacts{}, err
	}
	action := occurrence.Action
	primary, primaryValue, err := roleAndValue(state, action.XRole)
	if err != nil {
		return LocalFacts{}, err
	}
	secondary, secondaryValue, err := roleAndValue(state, action.YRole)
	if err != nil {
		return LocalFacts{}, err
	}
	facts := LocalFacts{
		Kind: action.Kind, PrimaryRole: primary, SecondaryRole: secondary,
		PrimaryValue: primaryValue, SecondaryValue: secondaryValue,
		TraceLength: len(state.Events), Symbol: action.Symbol,
		ReadRoles: []int{}, WriteRoles: []int{},
	}
	switch action.Kind {
	case "add", "claim", "release":
		facts.ReadRoles, facts.WriteRoles = []int{primary}, []int{primary}
	case "set":
		facts.WriteRoles = []int{primary}
	case "transfer", "swap":
		facts.ReadRoles = sortedUnique(primary, secondary)
		facts.WriteRoles = slices.Clone(facts.ReadRoles)
	case "check":
		facts.ReadRoles = []int{primary}
	case "emit":
		facts.WriteRoles = []int{-2}
	}
	if action.Kind == "add" || action.Kind == "set" || action.Kind == "transfer" || action.Kind == "check" {
		facts.ArgumentPresent = true
		facts.ArgumentValue = action.N
	}
	facts.StateDigest, err = state.Digest()
	if err != nil {
		return LocalFacts{}, err
	}
	facts.OccurrenceDigest, err = occurrence.Digest()
	if err != nil {
		return LocalFacts{}, err
	}
	return facts, facts.Validate()
}

func (f LocalFacts) Validate() error {
	if !validDigest(f.StateDigest) || !validDigest(f.OccurrenceDigest) || !oneString(f.Kind, "add", "set", "transfer", "swap", "claim", "release", "check", "emit") ||
		f.PrimaryRole < -1 || f.PrimaryRole > 2 || f.SecondaryRole < -1 || f.SecondaryRole > 2 ||
		f.PrimaryValue < -1 || f.PrimaryValue > MaxCellValue || f.SecondaryValue < -1 || f.SecondaryValue > MaxCellValue ||
		f.TraceLength < 0 || f.TraceLength > MaxEvents {
		return ErrInvalid
	}
	if f.PrimaryRole == -1 != (f.PrimaryValue == -1) || f.SecondaryRole == -1 != (f.SecondaryValue == -1) {
		return ErrInvalid
	}
	if !f.ArgumentPresent && f.ArgumentValue != 0 || f.Kind != "emit" && f.Symbol != "" || f.Kind == "emit" && !validIdentifier(f.Symbol) {
		return ErrInvalid
	}
	if !validFootprint(f.ReadRoles) || !validFootprint(f.WriteRoles) {
		return ErrInvalid
	}
	action := SemanticAction{Kind: f.Kind, XRole: roleName(f.PrimaryRole), YRole: roleName(f.SecondaryRole), Symbol: f.Symbol}
	if f.ArgumentPresent {
		action.N = f.ArgumentValue
	}
	if err := action.Validate(); err != nil {
		return ErrInvalid
	}
	var reads, writes []int
	switch f.Kind {
	case "add", "claim", "release":
		reads, writes = []int{f.PrimaryRole}, []int{f.PrimaryRole}
	case "set":
		writes = []int{f.PrimaryRole}
	case "transfer", "swap":
		reads = sortedUnique(f.PrimaryRole, f.SecondaryRole)
		writes = slices.Clone(reads)
	case "check":
		reads = []int{f.PrimaryRole}
	case "emit":
		writes = []int{-2}
	}
	wantsArgument := oneString(f.Kind, "add", "set", "transfer", "check")
	if f.ArgumentPresent != wantsArgument || !slices.Equal(f.ReadRoles, reads) || !slices.Equal(f.WriteRoles, writes) {
		return ErrInvalid
	}
	return nil
}

func (f LocalFacts) CanonicalJSON() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal([]any{LocalFactsVersion, f.StateDigest, f.OccurrenceDigest, f.Kind,
		f.PrimaryRole, f.SecondaryRole, f.ArgumentPresent, f.ArgumentValue, f.Symbol,
		nonNilInts(f.ReadRoles), nonNilInts(f.WriteRoles), f.PrimaryValue, f.SecondaryValue, f.TraceLength})
}

func (f LocalFacts) Digest() (string, error) { return digestCanonical(f.CanonicalJSON()) }

func roleAndValue(state State, role string) (int, int, error) {
	if role == "" {
		return -1, -1, nil
	}
	if !validRole(role) {
		return 0, 0, ErrInvalid
	}
	value, ok := state.Value(role)
	if !ok {
		return 0, 0, ErrInvalid
	}
	return int(role[1] - '0'), value, nil
}

func sortedUnique(values ...int) []int {
	slices.Sort(values)
	return slices.Compact(values)
}

func validFootprint(values []int) bool {
	for index, value := range values {
		if value < -2 || value > 2 || value == -1 || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}

func intList(value any) ([]int, bool) {
	raw, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := make([]int, len(raw))
	for index, item := range raw {
		integer, ok := exactInt(item)
		if !ok {
			return nil, false
		}
		result[index] = integer
	}
	return result, true
}

func nonNilInts(values []int) []int {
	if values == nil {
		return []int{}
	}
	return values
}

func oneString(value string, allowed ...string) bool { return slices.Contains(allowed, value) }

func roleName(role int) string {
	if role < 0 {
		return ""
	}
	return "c" + strconv.Itoa(role)
}
