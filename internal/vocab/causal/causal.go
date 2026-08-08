// Package causal implements the pure production semantics for the bounded
// three-variable active causal-diagnosis vocabulary.
package causal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

const (
	Variables      = 3
	MaximumPool    = 32
	MaximumActions = 10
	EpisodeCostCap = 1000
	InvalidScore   = 1001
	ProfileVersion = "causal-profile/v1"
	ExperimentV1   = "active-causal-diagnosis/v1"
)

type Node struct {
	Parents   []int  `json:"parents"`
	Mechanism string `json:"mechanism"`
}

type Hypothesis struct {
	Nodes []Node `json:"nodes"`
}

type Action struct {
	Variable int
	Value    int
}

var actions = []Action{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}, {2, 1}}

func Actions() []Action { return append([]Action(nil), actions...) }

func (a Action) Code() string {
	if !a.Valid() {
		return ""
	}
	return fmt.Sprintf("do:%d=%d", a.Variable, a.Value)
}

func (a Action) Valid() bool {
	return a.Variable >= 0 && a.Variable < Variables && (a.Value == 0 || a.Value == 1)
}

func ParseAction(code string) (Action, error) {
	for _, action := range actions {
		if action.Code() == code {
			return action, nil
		}
	}
	return Action{}, fmt.Errorf("invalid causal action %q", code)
}

func mechanismValid(arity int, mechanism string) bool {
	switch arity {
	case 0:
		return mechanism == "constant-0" || mechanism == "constant-1"
	case 1:
		return mechanism == "copy" || mechanism == "negate"
	case 2:
		return mechanism == "and" || mechanism == "or" || mechanism == "xor"
	}
	return false
}

func Validate(h Hypothesis) error {
	if len(h.Nodes) != Variables {
		return fmt.Errorf("nodes=%d, want %d", len(h.Nodes), Variables)
	}
	for position, node := range h.Nodes {
		if len(node.Parents) > 2 || !mechanismValid(len(node.Parents), node.Mechanism) {
			return fmt.Errorf("node %d has invalid mechanism/arity", position)
		}
		previous := -1
		for _, parent := range node.Parents {
			if parent <= previous || parent < 0 || parent >= position {
				return fmt.Errorf("node %d has noncanonical parent %d", position, parent)
			}
			previous = parent
		}
	}
	return nil
}

func Code(h Hypothesis) (string, error) {
	if err := Validate(h); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(h)
	return string(encoded), err
}

func Parse(code string) (Hypothesis, error) {
	var h Hypothesis
	decoder := json.NewDecoder(strings.NewReader(code))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&h); err != nil {
		return h, err
	}
	if decoder.More() {
		return h, errors.New("trailing hypothesis data")
	}
	canonical, err := Code(h)
	if err != nil {
		return h, err
	}
	if canonical != code {
		return h, errors.New("noncanonical hypothesis encoding")
	}
	return h, nil
}

func Enumerate() []Hypothesis {
	zero := []Node{{Parents: []int{}, Mechanism: "constant-0"}, {Parents: []int{}, Mechanism: "constant-1"}}
	one := func(parent int) []Node {
		return []Node{{Parents: []int{parent}, Mechanism: "copy"}, {Parents: []int{parent}, Mechanism: "negate"}}
	}
	two := []Node{{Parents: []int{0, 1}, Mechanism: "and"}, {Parents: []int{0, 1}, Mechanism: "or"}, {Parents: []int{0, 1}, Mechanism: "xor"}}
	var universe []Hypothesis
	for _, a := range zero {
		for _, b := range append(append([]Node{}, zero...), one(0)...) {
			third := append([]Node{}, zero...)
			third = append(third, one(0)...)
			third = append(third, one(1)...)
			third = append(third, two...)
			for _, c := range third {
				universe = append(universe, Hypothesis{Nodes: []Node{cloneNode(a), cloneNode(b), cloneNode(c)}})
			}
		}
	}
	sort.Slice(universe, func(i, j int) bool { a, _ := Code(universe[i]); b, _ := Code(universe[j]); return a < b })
	return universe
}

func cloneNode(n Node) Node {
	return Node{Parents: append([]int(nil), n.Parents...), Mechanism: n.Mechanism}
}

func Evaluate(h Hypothesis, intervention *Action) ([3]int, error) {
	var result [3]int
	if err := Validate(h); err != nil {
		return result, err
	}
	if intervention != nil && !intervention.Valid() {
		return result, errors.New("invalid intervention")
	}
	for position, node := range h.Nodes {
		if intervention != nil && intervention.Variable == position {
			result[position] = intervention.Value
			continue
		}
		switch node.Mechanism {
		case "constant-0":
			result[position] = 0
		case "constant-1":
			result[position] = 1
		case "copy":
			result[position] = result[node.Parents[0]]
		case "negate":
			result[position] = 1 - result[node.Parents[0]]
		case "and":
			result[position] = result[node.Parents[0]] & result[node.Parents[1]]
		case "or":
			result[position] = result[node.Parents[0]] | result[node.Parents[1]]
		case "xor":
			result[position] = result[node.Parents[0]] ^ result[node.Parents[1]]
		}
	}
	return result, nil
}

func OutcomeCode(outcome [3]int) string {
	return fmt.Sprintf("%d%d%d", outcome[0], outcome[1], outcome[2])
}

func PredictCode(code, actionCode string) (string, error) {
	h, err := Parse(code)
	if err != nil {
		return "", err
	}
	a, err := ParseAction(actionCode)
	if err != nil {
		return "", err
	}
	o, err := Evaluate(h, &a)
	if err != nil {
		return "", err
	}
	return OutcomeCode(o), nil
}

type Cell struct {
	Outcome    string   `json:"outcome"`
	Hypotheses []string `json:"hypotheses"`
}

func Partition(posterior []string, actionCode string) ([]Cell, error) {
	if len(posterior) == 0 || len(posterior) > MaximumPool {
		return nil, errors.New("invalid posterior size")
	}
	byOutcome := map[string][]string{}
	seen := map[string]bool{}
	for _, code := range posterior {
		if seen[code] {
			return nil, errors.New("duplicate hypothesis")
		}
		seen[code] = true
		outcome, err := PredictCode(code, actionCode)
		if err != nil {
			return nil, err
		}
		byOutcome[outcome] = append(byOutcome[outcome], code)
	}
	var cells []Cell
	for value := 0; value < 8; value++ {
		outcome := fmt.Sprintf("%03b", value)
		if hypotheses := byOutcome[outcome]; len(hypotheses) > 0 {
			sort.Strings(hypotheses)
			cells = append(cells, Cell{Outcome: outcome, Hypotheses: hypotheses})
		}
	}
	return cells, nil
}

func Filter(posterior []string, actionCode, outcome string) ([]string, error) {
	cells, err := Partition(posterior, actionCode)
	if err != nil {
		return nil, err
	}
	for _, cell := range cells {
		if cell.Outcome == outcome {
			return append([]string(nil), cell.Hypotheses...), nil
		}
	}
	return nil, errors.New("outcome absent from partition")
}

func Signature(code string) (string, error) {
	parts := make([]string, 0, 6)
	for _, action := range actions {
		outcome, err := PredictCode(code, action.Code())
		if err != nil {
			return "", err
		}
		parts = append(parts, outcome)
	}
	return strings.Join(parts, "/"), nil
}

func CompleteClass(initial, posterior []string) bool {
	if len(posterior) <= 1 {
		return false
	}
	want, err := Signature(posterior[0])
	if err != nil {
		return false
	}
	members := make([]string, 0)
	for _, code := range initial {
		signature, err := Signature(code)
		if err != nil {
			return false
		}
		if signature == want {
			members = append(members, code)
		}
	}
	sort.Strings(members)
	actual := append([]string(nil), posterior...)
	sort.Strings(actual)
	return strings.Join(members, "\x00") == strings.Join(actual, "\x00")
}

type Features struct {
	ExpectedNumerator int
	Worst             int
	EntropyProduct    *big.Int
	Cost              int
	Repeat            int
}

func FeaturesFor(posterior []string, action string, cost int, repeated bool) (Features, error) {
	cells, err := Partition(posterior, action)
	if err != nil {
		return Features{}, err
	}
	f := Features{Cost: cost, EntropyProduct: big.NewInt(1)}
	if repeated {
		f.Repeat = 1
	}
	for _, cell := range cells {
		n := len(cell.Hypotheses)
		f.ExpectedNumerator += n * n
		if n > f.Worst {
			f.Worst = n
		}
		f.EntropyProduct.Mul(f.EntropyProduct, new(big.Int).Exp(big.NewInt(int64(n)), big.NewInt(int64(n)), nil))
	}
	return f, nil
}

var featureOrder = []string{"E", "W", "H", "C", "R"}

type Rule struct{ Primary, Mode, Secondary string }

func Rules() []Rule {
	var rules []Rule
	for _, primary := range featureOrder {
		for _, mode := range []string{"raw", "gain"} {
			for _, secondary := range featureOrder {
				if secondary != primary {
					rules = append(rules, Rule{primary, mode, secondary})
				}
			}
		}
	}
	return rules
}

func (r Rule) Code() string { return fmt.Sprintf("P=%s;M=%s;S=%s", r.Primary, r.Mode, r.Secondary) }

func ParseRule(code string) (Rule, error) {
	for _, rule := range Rules() {
		if rule.Code() == code {
			return rule, nil
		}
	}
	return Rule{}, fmt.Errorf("invalid acquisition rule %q", code)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func rawCompare(feature string, a, b Features) int {
	switch feature {
	case "E":
		return cmpInt(a.ExpectedNumerator, b.ExpectedNumerator)
	case "W":
		return cmpInt(a.Worst, b.Worst)
	case "H":
		return a.EntropyProduct.Cmp(b.EntropyProduct)
	case "C":
		return cmpInt(a.Cost, b.Cost)
	case "R":
		return cmpInt(a.Repeat, b.Repeat)
	}
	return 0
}

// Compare returns -1 when a is preferred, +1 when b is preferred, and zero
// for an exact primary/secondary tie. Semantic action code is deliberately not
// included so callers can retain the complete tie set.
func Compare(rule Rule, posteriorSize int, a, b Features) int {
	primary := 0
	if rule.Mode == "gain" && (rule.Primary == "E" || rule.Primary == "W" || rule.Primary == "H") {
		switch rule.Primary {
		case "E":
			primary = -cmpInt((posteriorSize*posteriorSize-a.ExpectedNumerator)*b.Cost, (posteriorSize*posteriorSize-b.ExpectedNumerator)*a.Cost)
		case "W":
			primary = -cmpInt((posteriorSize-a.Worst)*b.Cost, (posteriorSize-b.Worst)*a.Cost)
		case "H":
			n := big.NewInt(int64(posteriorSize))
			left := new(big.Int).Mul(new(big.Int).Exp(n, big.NewInt(int64(posteriorSize*b.Cost)), nil), new(big.Int).Exp(b.EntropyProduct, big.NewInt(int64(a.Cost)), nil))
			right := new(big.Int).Mul(new(big.Int).Exp(n, big.NewInt(int64(posteriorSize*a.Cost)), nil), new(big.Int).Exp(a.EntropyProduct, big.NewInt(int64(b.Cost)), nil))
			primary = -left.Cmp(right)
		}
	} else {
		primary = rawCompare(rule.Primary, a, b)
	}
	if primary != 0 {
		return primary
	}
	return rawCompare(rule.Secondary, a, b)
}

func Digest(domain string, value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(domain))
	h.Write([]byte{0})
	h.Write(encoded)
	return hex.EncodeToString(h.Sum(nil)), nil
}
