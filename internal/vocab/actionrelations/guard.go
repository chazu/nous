package actionrelations

import (
	"bytes"
	"encoding/json"
	"slices"
)

var Atoms = []string{
	"read-write-disjoint",
	"primary-same",
	"secondary-same",
	"a-primary-b-secondary",
	"a-secondary-b-primary",
	"argument-equal",
	"argument-opposite",
	"symbol-equal",
	"shared-value-zero",
	"shared-value-max",
	"a-primary-zero",
	"a-primary-max",
	"b-primary-zero",
	"b-primary-max",
	"combined-adds-in-bounds",
}

type Pattern struct {
	Kinds []string
	Roles []int
}

func PatternFor(left, right Occurrence) (Pattern, error) {
	left, right, err := CanonicalPair(left, right)
	if err != nil {
		return Pattern{}, err
	}
	roles := []string{left.Action.XRole, left.Action.YRole, right.Action.XRole, right.Action.YRole}
	aliases := make([]int, len(roles))
	next := 0
	seen := map[string]int{}
	for index, role := range roles {
		if role == "" {
			aliases[index] = -1
			continue
		}
		alias, ok := seen[role]
		if !ok {
			alias = next
			next++
			seen[role] = alias
		}
		aliases[index] = alias
	}
	return Pattern{Kinds: []string{left.Action.Kind, right.Action.Kind}, Roles: aliases}, nil
}

func CanonicalPair(left, right Occurrence) (Occurrence, Occurrence, error) {
	a, err := left.CanonicalJSON()
	if err != nil {
		return Occurrence{}, Occurrence{}, err
	}
	b, err := right.CanonicalJSON()
	if err != nil {
		return Occurrence{}, Occurrence{}, err
	}
	if bytes.Compare(a, b) > 0 {
		return right, left, nil
	}
	return left, right, nil
}

func (p Pattern) Validate() error {
	if len(p.Kinds) != 2 || len(p.Roles) != 4 {
		return ErrInvalid
	}
	for _, kind := range p.Kinds {
		if !oneString(kind, "add", "set", "transfer", "swap", "claim", "release", "check", "emit") {
			return ErrInvalid
		}
	}
	next := 0
	seen := map[int]bool{}
	for _, role := range p.Roles {
		if role == -1 {
			continue
		}
		if role < 0 || role > 3 {
			return ErrInvalid
		}
		if !seen[role] {
			if role != next {
				return ErrInvalid
			}
			seen[role] = true
			next++
		}
	}
	return nil
}

func (p Pattern) CanonicalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal([]any{PatternVersion, p.Kinds, p.Roles})
}

func (p Pattern) Digest() (string, error) { return digestCanonical(p.CanonicalJSON()) }

type Literal struct {
	Atom     string
	Polarity bool
}

type Guard struct {
	Literals []Literal
}

func (g Guard) Validate() error {
	if len(g.Literals) > 2 {
		return ErrInvalid
	}
	previous := -1
	for _, literal := range g.Literals {
		rank := atomRank(literal.Atom)
		if rank < 0 || rank <= previous {
			return ErrInvalid
		}
		previous = rank
	}
	return nil
}

func (g Guard) CanonicalJSON() ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	literals := make([]any, len(g.Literals))
	for index, literal := range g.Literals {
		literals[index] = []any{literal.Atom, literal.Polarity}
	}
	return json.Marshal([]any{GuardVersion, literals})
}

func (g Guard) Digest() (string, error) { return digestCanonical(g.CanonicalJSON()) }

func (g Guard) Parent() (Guard, bool, error) {
	if err := g.Validate(); err != nil {
		return Guard{}, false, err
	}
	if len(g.Literals) == 0 {
		return Guard{}, false, nil
	}
	return Guard{Literals: slices.Clone(g.Literals[:len(g.Literals)-1])}, true, nil
}

func (g Guard) Evaluate(left, right LocalFacts) (bool, error) {
	if err := g.Validate(); err != nil {
		return false, err
	}
	if err := left.Validate(); err != nil {
		return false, err
	}
	if err := right.Validate(); err != nil || left.StateDigest != right.StateDigest {
		return false, ErrInvalid
	}
	for _, literal := range g.Literals {
		value, err := EvaluateAtom(literal.Atom, left, right)
		if err != nil || value != literal.Polarity {
			return false, err
		}
	}
	return true, nil
}

func EnumerateGuards() []Guard {
	guards := []Guard{{}}
	for _, atom := range Atoms {
		for _, polarity := range []bool{false, true} {
			guards = append(guards, Guard{Literals: []Literal{{Atom: atom, Polarity: polarity}}})
		}
	}
	for left := 0; left < len(Atoms); left++ {
		for right := left + 1; right < len(Atoms); right++ {
			for _, a := range []bool{false, true} {
				for _, b := range []bool{false, true} {
					guards = append(guards, Guard{Literals: []Literal{{Atom: Atoms[left], Polarity: a}, {Atom: Atoms[right], Polarity: b}}})
				}
			}
		}
	}
	return guards
}

func EvaluateAtom(atom string, a, b LocalFacts) (bool, error) {
	if atomRank(atom) < 0 || a.StateDigest != b.StateDigest {
		return false, ErrInvalid
	}
	if err := a.Validate(); err != nil {
		return false, err
	}
	if err := b.Validate(); err != nil {
		return false, err
	}
	switch atom {
	case "read-write-disjoint":
		return disjoint(a.WriteRoles, union(b.ReadRoles, b.WriteRoles)) && disjoint(b.WriteRoles, union(a.ReadRoles, a.WriteRoles)), nil
	case "primary-same":
		return presentEqual(a.PrimaryRole, b.PrimaryRole), nil
	case "secondary-same":
		return presentEqual(a.SecondaryRole, b.SecondaryRole), nil
	case "a-primary-b-secondary":
		return presentEqual(a.PrimaryRole, b.SecondaryRole), nil
	case "a-secondary-b-primary":
		return presentEqual(a.SecondaryRole, b.PrimaryRole), nil
	case "argument-equal":
		return a.ArgumentPresent && b.ArgumentPresent && a.ArgumentValue == b.ArgumentValue, nil
	case "argument-opposite":
		return a.Kind == "add" && b.Kind == "add" && a.ArgumentValue+b.ArgumentValue == 0, nil
	case "symbol-equal":
		return a.Kind == "emit" && b.Kind == "emit" && a.Symbol == b.Symbol, nil
	case "shared-value-zero", "shared-value-max":
		role, ok := lowestSharedRole(a, b)
		if !ok {
			return false, nil
		}
		value, ok := factValue(a, role)
		if !ok {
			value, ok = factValue(b, role)
		}
		if !ok {
			return false, ErrInvalid
		}
		if atom == "shared-value-zero" {
			return value == 0, nil
		}
		return value == MaxCellValue, nil
	case "a-primary-zero":
		return a.PrimaryRole >= 0 && a.PrimaryValue == 0, nil
	case "a-primary-max":
		return a.PrimaryRole >= 0 && a.PrimaryValue == MaxCellValue, nil
	case "b-primary-zero":
		return b.PrimaryRole >= 0 && b.PrimaryValue == 0, nil
	case "b-primary-max":
		return b.PrimaryRole >= 0 && b.PrimaryValue == MaxCellValue, nil
	case "combined-adds-in-bounds":
		if a.Kind != "add" || b.Kind != "add" || !presentEqual(a.PrimaryRole, b.PrimaryRole) {
			return false, nil
		}
		value := a.PrimaryValue
		return value+a.ArgumentValue >= 0 && value+a.ArgumentValue <= MaxCellValue &&
			value+b.ArgumentValue >= 0 && value+b.ArgumentValue <= MaxCellValue &&
			value+a.ArgumentValue+b.ArgumentValue >= 0 && value+a.ArgumentValue+b.ArgumentValue <= MaxCellValue, nil
	default:
		return false, ErrInvalid
	}
}

func atomRank(atom string) int { return slices.Index(Atoms, atom) }

func presentEqual(left, right int) bool { return left >= 0 && right >= 0 && left == right }

func union(left, right []int) []int {
	values := append(slices.Clone(left), right...)
	slices.Sort(values)
	return slices.Compact(values)
}

func disjoint(left, right []int) bool {
	for _, value := range left {
		if _, found := slices.BinarySearch(right, value); found {
			return false
		}
	}
	return true
}

func referenced(f LocalFacts) []int {
	values := union(f.ReadRoles, f.WriteRoles)
	return slices.DeleteFunc(values, func(value int) bool { return value < 0 })
}

func lowestSharedRole(a, b LocalFacts) (int, bool) {
	for _, role := range referenced(a) {
		if _, found := slices.BinarySearch(referenced(b), role); found {
			return role, true
		}
	}
	return 0, false
}

func factValue(f LocalFacts, role int) (int, bool) {
	if f.PrimaryRole == role {
		return f.PrimaryValue, true
	}
	if f.SecondaryRole == role {
		return f.SecondaryValue, true
	}
	return 0, false
}
