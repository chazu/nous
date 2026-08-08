// Package causaloracle is an independent enumerator, evaluator, teacher, and
// replay oracle for the active causal-diagnosis experiment. It deliberately
// does not import the production vocabulary, DSL, engine, or store.
package causaloracle

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

type Node struct {
	Parents   []int  `json:"parents"`
	Mechanism string `json:"mechanism"`
}
type Model struct {
	Nodes []Node `json:"nodes"`
}
type Action struct{ Variable, Value int }
type Cell struct {
	Outcome string
	Models  []string
}

var actionList = []Action{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}, {2, 1}}

func Actions() []Action { return append([]Action(nil), actionList...) }
func (a Action) Code() string {
	if a.Variable < 0 || a.Variable > 2 || a.Value < 0 || a.Value > 1 {
		return ""
	}
	return fmt.Sprintf("do:%d=%d", a.Variable, a.Value)
}
func ParseAction(code string) (Action, error) {
	for _, a := range actionList {
		if a.Code() == code {
			return a, nil
		}
	}
	return Action{}, errors.New("bad action")
}

func valid(m Model) bool {
	if len(m.Nodes) != 3 {
		return false
	}
	for i, n := range m.Nodes {
		ok := (len(n.Parents) == 0 && (n.Mechanism == "constant-0" || n.Mechanism == "constant-1")) || (len(n.Parents) == 1 && (n.Mechanism == "copy" || n.Mechanism == "negate")) || (len(n.Parents) == 2 && (n.Mechanism == "and" || n.Mechanism == "or" || n.Mechanism == "xor"))
		if !ok {
			return false
		}
		previous := -1
		for _, p := range n.Parents {
			if p <= previous || p < 0 || p >= i {
				return false
			}
			previous = p
		}
	}
	return true
}
func Code(m Model) (string, error) {
	if !valid(m) {
		return "", errors.New("invalid model")
	}
	b, e := json.Marshal(m)
	return string(b), e
}
func Parse(code string) (Model, error) {
	var m Model
	d := json.NewDecoder(strings.NewReader(code))
	d.DisallowUnknownFields()
	if e := d.Decode(&m); e != nil {
		return m, e
	}
	canonical, e := Code(m)
	if e != nil {
		return m, e
	}
	if canonical != code {
		return m, errors.New("noncanonical model")
	}
	return m, nil
}
func node(parents []int, mechanism string) Node {
	return Node{append([]int(nil), parents...), mechanism}
}
func Enumerate() []Model {
	z := []Node{node([]int{}, "constant-0"), node([]int{}, "constant-1")}
	o0 := []Node{node([]int{0}, "copy"), node([]int{0}, "negate")}
	o1 := []Node{node([]int{1}, "copy"), node([]int{1}, "negate")}
	t := []Node{node([]int{0, 1}, "and"), node([]int{0, 1}, "or"), node([]int{0, 1}, "xor")}
	var out []Model
	second := append(append([]Node{}, z...), o0...)
	third := append(append(append(append([]Node{}, z...), o0...), o1...), t...)
	for _, a := range z {
		for _, b := range second {
			for _, c := range third {
				out = append(out, Model{[]Node{node(a.Parents, a.Mechanism), node(b.Parents, b.Mechanism), node(c.Parents, c.Mechanism)}})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { a, _ := Code(out[i]); b, _ := Code(out[j]); return a < b })
	return out
}
func Evaluate(m Model, intervention *Action) ([3]int, error) {
	var result [3]int
	if !valid(m) {
		return result, errors.New("invalid model")
	}
	if intervention != nil && intervention.Code() == "" {
		return result, errors.New("invalid action")
	}
	for i, n := range m.Nodes {
		if intervention != nil && intervention.Variable == i {
			result[i] = intervention.Value
			continue
		}
		switch n.Mechanism {
		case "constant-0":
			result[i] = 0
		case "constant-1":
			result[i] = 1
		case "copy":
			result[i] = result[n.Parents[0]]
		case "negate":
			result[i] = 1 - result[n.Parents[0]]
		case "and":
			result[i] = result[n.Parents[0]] & result[n.Parents[1]]
		case "or":
			result[i] = result[n.Parents[0]] | result[n.Parents[1]]
		case "xor":
			result[i] = result[n.Parents[0]] ^ result[n.Parents[1]]
		}
	}
	return result, nil
}
func Outcome(o [3]int) string { return fmt.Sprintf("%d%d%d", o[0], o[1], o[2]) }
func Predict(code, action string) (string, error) {
	m, e := Parse(code)
	if e != nil {
		return "", e
	}
	a, e := ParseAction(action)
	if e != nil {
		return "", e
	}
	o, e := Evaluate(m, &a)
	return Outcome(o), e
}
func Observe(code string) (string, error) {
	m, e := Parse(code)
	if e != nil {
		return "", e
	}
	o, e := Evaluate(m, nil)
	return Outcome(o), e
}
func Partition(posterior []string, action string) ([]Cell, error) {
	by := map[string][]string{}
	for _, code := range posterior {
		o, e := Predict(code, action)
		if e != nil {
			return nil, e
		}
		by[o] = append(by[o], code)
	}
	var cells []Cell
	for i := 0; i < 8; i++ {
		o := fmt.Sprintf("%03b", i)
		if len(by[o]) > 0 {
			sort.Strings(by[o])
			cells = append(cells, Cell{o, by[o]})
		}
	}
	return cells, nil
}
func Filter(p []string, a, o string) ([]string, error) {
	cells, e := Partition(p, a)
	if e != nil {
		return nil, e
	}
	for _, c := range cells {
		if c.Outcome == o {
			return append([]string(nil), c.Models...), nil
		}
	}
	return nil, errors.New("outcome absent")
}
func Signature(code string) (string, error) {
	var parts []string
	for _, a := range actionList {
		o, e := Predict(code, a.Code())
		if e != nil {
			return "", e
		}
		parts = append(parts, o)
	}
	return strings.Join(parts, "/"), nil
}
func CompleteClass(initial, posterior []string) bool {
	if len(posterior) <= 1 {
		return false
	}
	s, e := Signature(posterior[0])
	if e != nil {
		return false
	}
	var want []string
	for _, c := range initial {
		x, e := Signature(c)
		if e != nil {
			return false
		}
		if x == s {
			want = append(want, c)
		}
	}
	a := append([]string(nil), posterior...)
	sort.Strings(a)
	sort.Strings(want)
	return strings.Join(a, "\x00") == strings.Join(want, "\x00")
}

type Teacher interface {
	Respond(token, action string) (string, error)
}
type PrivateTeacher struct{ token, hidden string }

func NewTeacher(token, hidden string) (Teacher, error) {
	if _, e := Parse(hidden); e != nil {
		return nil, e
	}
	if token == "" {
		return nil, errors.New("empty token")
	}
	return &PrivateTeacher{token, hidden}, nil
}
func (t *PrivateTeacher) Respond(token, action string) (string, error) {
	if token != t.token {
		return "", errors.New("unauthorized teacher token")
	}
	return Predict(t.hidden, action)
}

type dynamicValue struct {
	finite bool
	value  *big.Rat
	action string
}
type DynamicPolicy struct {
	Initial      []string
	Costs        [3]int
	memo         map[string]dynamicValue
	States, Work int
}

func NewDynamicPolicy(initial []string, costs [3]int) *DynamicPolicy {
	return &DynamicPolicy{Initial: append([]string(nil), initial...), Costs: costs, memo: map[string]dynamicValue{}}
}
func terminal(initial, p []string) bool { return len(p) == 1 || CompleteClass(initial, p) }
func stateKey(p []string, mask uint8) string {
	x := append([]string(nil), p...)
	sort.Strings(x)
	return fmt.Sprintf("%02x|%s", mask, strings.Join(x, "\x00"))
}
func (d *DynamicPolicy) value(p []string, mask uint8) dynamicValue {
	if terminal(d.Initial, p) {
		return dynamicValue{true, new(big.Rat), ""}
	}
	key := stateKey(p, mask)
	d.Work++
	if cached, ok := d.memo[key]; ok {
		return cached
	}
	d.States++
	var best dynamicValue
	for index, a := range actionList {
		if mask&(1<<index) != 0 {
			continue
		}
		d.Work++
		cells, e := Partition(p, a.Code())
		if e != nil || len(cells) <= 1 {
			continue
		}
		q := new(big.Rat).SetInt64(int64(d.Costs[a.Variable]))
		finite := true
		for _, cell := range cells {
			d.Work += 2
			child := d.value(cell.Models, mask|(1<<index))
			if !child.finite {
				finite = false
				break
			}
			term := new(big.Rat).Mul(new(big.Rat).SetFrac64(int64(len(cell.Models)), int64(len(p))), child.value)
			q.Add(q, term)
		}
		if finite && (!best.finite || q.Cmp(best.value) < 0 || (q.Cmp(best.value) == 0 && a.Code() < best.action)) {
			best = dynamicValue{true, new(big.Rat).Set(q), a.Code()}
		}
	}
	d.memo[key] = best
	return best
}
func (d *DynamicPolicy) Choose(p []string, consumed []string) (string, error) {
	var mask uint8
	for _, code := range consumed {
		for i, a := range actionList {
			if code == a.Code() {
				mask |= 1 << i
			}
		}
	}
	value := d.value(p, mask)
	d.Work++
	if !value.finite || value.action == "" {
		return "", errors.New("no finite dynamic action")
	}
	return value.action, nil
}
func (d *DynamicPolicy) ExpectedCost() (*big.Rat, error) {
	total := new(big.Rat)
	for _, hidden := range d.Initial {
		p := append([]string(nil), d.Initial...)
		var consumed []string
		cost := 0
		for !terminal(d.Initial, p) {
			a, e := d.Choose(p, consumed)
			if e != nil {
				return nil, e
			}
			action, _ := ParseAction(a)
			cost += d.Costs[action.Variable]
			outcome, _ := Predict(hidden, a)
			p, e = Filter(p, a, outcome)
			if e != nil {
				return nil, e
			}
			consumed = append(consumed, a)
		}
		total.Add(total, new(big.Rat).SetInt64(int64(cost)))
	}
	return total.Quo(total, new(big.Rat).SetInt64(int64(len(d.Initial)))), nil
}
