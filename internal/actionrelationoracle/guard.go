package actionrelationoracle

import (
	"bytes"
	"encoding/json"
	"slices"
)

var guardAtoms = []string{
	"read-write-disjoint", "primary-same", "secondary-same", "a-primary-b-secondary", "a-secondary-b-primary",
	"argument-equal", "argument-opposite", "symbol-equal", "shared-value-zero", "shared-value-max",
	"a-primary-zero", "a-primary-max", "b-primary-zero", "b-primary-max", "combined-adds-in-bounds",
}

type oracleLiteral struct {
	atom     string
	polarity bool
}

type oracleFacts struct {
	kind               string
	primary, secondary int
	argumentPresent    bool
	argument           int
	symbol             string
	reads, writes      []int
	primaryValue       int
	secondaryValue     int
	traceLength        int
}

// EvaluateGuard independently parses and evaluates one normalized guard over
// a pair of semantic actions at one state.
func EvaluateGuard(stateJSON, aJSON, bJSON, guardJSON []byte) (bool, error) {
	s, err := parseState(stateJSON)
	if err != nil {
		return false, err
	}
	a, err := parseAction(aJSON)
	if err != nil {
		return false, err
	}
	b, err := parseAction(bJSON)
	if err != nil {
		return false, err
	}
	literals, err := parseGuard(guardJSON)
	if err != nil {
		return false, err
	}
	aFacts, err := deriveFacts(s, a)
	if err != nil {
		return false, err
	}
	bFacts, err := deriveFacts(s, b)
	if err != nil {
		return false, err
	}
	for _, literal := range literals {
		value, err := evaluateAtom(literal.atom, aFacts, bFacts)
		if err != nil || value != literal.polarity {
			return false, err
		}
	}
	return true, nil
}

func parseGuard(data []byte) ([]oracleLiteral, error) {
	value, err := decode(data)
	if err != nil {
		return nil, err
	}
	row, ok := value.([]any)
	if !ok || len(row) != 2 || row[0] != "action-guard/v1" {
		return nil, ErrInvalid
	}
	raw, ok := row[1].([]any)
	if !ok || len(raw) > 2 {
		return nil, ErrInvalid
	}
	result := make([]oracleLiteral, len(raw))
	previous := -1
	for index, item := range raw {
		pair, ok := item.([]any)
		if !ok || len(pair) != 2 {
			return nil, ErrInvalid
		}
		atom, atomOK := pair[0].(string)
		polarity, polarityOK := pair[1].(bool)
		rank := slices.Index(guardAtoms, atom)
		if !atomOK || !polarityOK || rank < 0 || rank <= previous {
			return nil, ErrInvalid
		}
		result[index], previous = oracleLiteral{atom: atom, polarity: polarity}, rank
	}
	canonical, _ := encodeGuard(result)
	if !bytes.Equal(canonical, data) {
		return nil, ErrInvalid
	}
	return result, nil
}

func encodeGuard(literals []oracleLiteral) ([]byte, error) {
	rows := make([]any, len(literals))
	for index, literal := range literals {
		rows[index] = []any{literal.atom, literal.polarity}
	}
	return json.Marshal([]any{"action-guard/v1", rows})
}

func deriveFacts(s state, a action) (oracleFacts, error) {
	primary, _, primaryOK := lookup(s, a.x)
	secondary, _, secondaryOK := lookup(s, a.y)
	primaryRole, secondaryRole := -1, -1
	if a.x != "" {
		primaryRole = int(a.x[1] - '0')
	}
	if a.y != "" {
		secondaryRole = int(a.y[1] - '0')
	}
	if a.x == "" {
		primary, primaryOK = -1, true
	}
	if a.y == "" {
		secondary, secondaryOK = -1, true
	}
	if !primaryOK || !secondaryOK {
		return oracleFacts{}, ErrInvalid
	}
	facts := oracleFacts{kind: a.kind, primary: primaryRole, secondary: secondaryRole, primaryValue: primary, secondaryValue: secondary, symbol: a.symbol, traceLength: len(s.events), reads: []int{}, writes: []int{}}
	switch a.kind {
	case "add", "claim", "release":
		facts.reads, facts.writes = []int{primaryRole}, []int{primaryRole}
	case "set":
		facts.writes = []int{primaryRole}
	case "transfer", "swap":
		facts.reads = sortedUniqueInts(primaryRole, secondaryRole)
		facts.writes = slices.Clone(facts.reads)
	case "check":
		facts.reads = []int{primaryRole}
	case "emit":
		facts.writes = []int{-2}
	default:
		return oracleFacts{}, ErrInvalid
	}
	if a.kind == "add" || a.kind == "set" || a.kind == "transfer" || a.kind == "check" {
		facts.argumentPresent, facts.argument = true, a.n
	}
	return facts, nil
}

func evaluateAtom(atom string, a, b oracleFacts) (bool, error) {
	switch atom {
	case "read-write-disjoint":
		return disjointInts(a.writes, unionInts(b.reads, b.writes)) && disjointInts(b.writes, unionInts(a.reads, a.writes)), nil
	case "primary-same":
		return presentEqualInt(a.primary, b.primary), nil
	case "secondary-same":
		return presentEqualInt(a.secondary, b.secondary), nil
	case "a-primary-b-secondary":
		return presentEqualInt(a.primary, b.secondary), nil
	case "a-secondary-b-primary":
		return presentEqualInt(a.secondary, b.primary), nil
	case "argument-equal":
		return a.argumentPresent && b.argumentPresent && a.argument == b.argument, nil
	case "argument-opposite":
		return a.kind == "add" && b.kind == "add" && a.argument+b.argument == 0, nil
	case "symbol-equal":
		return a.kind == "emit" && b.kind == "emit" && a.symbol == b.symbol, nil
	case "shared-value-zero", "shared-value-max":
		role, ok := lowestSharedRole(a, b)
		if !ok {
			return false, nil
		}
		value, ok := oracleFactValue(a, role)
		if !ok {
			value, ok = oracleFactValue(b, role)
		}
		if !ok {
			return false, ErrInvalid
		}
		if atom == "shared-value-zero" {
			return value == 0, nil
		}
		return value == 3, nil
	case "a-primary-zero":
		return a.primary >= 0 && a.primaryValue == 0, nil
	case "a-primary-max":
		return a.primary >= 0 && a.primaryValue == 3, nil
	case "b-primary-zero":
		return b.primary >= 0 && b.primaryValue == 0, nil
	case "b-primary-max":
		return b.primary >= 0 && b.primaryValue == 3, nil
	case "combined-adds-in-bounds":
		if a.kind != "add" || b.kind != "add" || !presentEqualInt(a.primary, b.primary) {
			return false, nil
		}
		value := a.primaryValue
		return value+a.argument >= 0 && value+a.argument <= 3 && value+b.argument >= 0 && value+b.argument <= 3 && value+a.argument+b.argument >= 0 && value+a.argument+b.argument <= 3, nil
	default:
		return false, ErrInvalid
	}
}

func sortedUniqueInts(values ...int) []int {
	slices.Sort(values)
	return slices.Compact(values)
}

func unionInts(left, right []int) []int {
	return sortedUniqueInts(append(slices.Clone(left), right...)...)
}

func disjointInts(left, right []int) bool {
	for _, value := range left {
		if _, found := slices.BinarySearch(right, value); found {
			return false
		}
	}
	return true
}

func presentEqualInt(left, right int) bool { return left >= 0 && right >= 0 && left == right }

func referencedRoles(f oracleFacts) []int {
	return slices.DeleteFunc(unionInts(f.reads, f.writes), func(value int) bool { return value < 0 })
}

func lowestSharedRole(a, b oracleFacts) (int, bool) {
	for _, role := range referencedRoles(a) {
		if _, found := slices.BinarySearch(referencedRoles(b), role); found {
			return role, true
		}
	}
	return 0, false
}

func oracleFactValue(f oracleFacts, role int) (int, bool) {
	if f.primary == role {
		return f.primaryValue, true
	}
	if f.secondary == role {
		return f.secondaryValue, true
	}
	return 0, false
}
