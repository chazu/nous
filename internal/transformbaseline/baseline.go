// Package transformbaseline contains conventional transformation learners. It
// intentionally does not import the production vocabulary, DSL, engine, or
// oracle.
package transformbaseline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"slices"

	"github.com/chazu/nous/internal/transformfixturecore"
)

type Result struct {
	Terminal     string
	Schema       []byte
	Applications int
	Ties         [][]byte
}

type schema struct{ anchor, targets, scope, guard, locality string }
type node struct {
	id, parent, target         int
	kind, key, value, from, to string
}
type forest struct{ nodes []node }

var errInvalid = errors.New("invalid baseline input")

// BoundedPBE performs canonical minimum-description enumeration. At most 40
// training applications are available because eight credits are reserved.
func BoundedPBE(trainingBytes []byte) (Result, error) {
	training, err := transformfixturecore.ParseTraining(trainingBytes)
	if err != nil {
		return Result{}, err
	}
	candidates := schemas()
	slices.SortFunc(candidates, compareSchema)
	return enumerate(training, candidates, true)
}

func RandomPBE(trainingBytes []byte, seed1, seed2 uint64) (Result, error) {
	training, err := transformfixturecore.ParseTraining(trainingBytes)
	if err != nil {
		return Result{}, err
	}
	candidates := schemas()
	rand.New(rand.NewPCG(seed1, seed2)).Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	return enumerate(training, candidates, false)
}

func enumerate(training transformfixturecore.Training, candidates []schema, retainTier bool) (Result, error) {
	result := Result{}
	foundCost := -1
	for _, candidate := range candidates {
		if foundCost >= 0 && (!retainTier || description(candidate) != foundCost) {
			break
		}
		exact := true
		for _, c := range training.Cases {
			if result.Applications >= 40 {
				return Result{Terminal: "budget-exhausted", Applications: result.Applications}, nil
			}
			result.Applications++
			terminal, output, err := apply(c.Before, candidate)
			if err != nil {
				return Result{}, err
			}
			match := c.Kind == "positive" && terminal == "applied" && bytes.Equal(output, c.After) ||
				c.Kind == "abstain" && len(output) == 0 && len(terminal) > 8 && terminal[:8] == "abstain/"
			if !match {
				exact = false
				break
			}
		}
		if exact {
			encoded := encodeSchema(candidate)
			if foundCost < 0 {
				foundCost = description(candidate)
				result.Schema = encoded
			}
			result.Ties = append(result.Ties, encoded)
			if !retainTier {
				break
			}
		}
	}
	if len(result.Schema) == 0 {
		result.Terminal = "no-discovery"
	} else {
		result.Terminal = "completed"
	}
	return result, nil
}

func schemas() []schema {
	var out []schema
	for _, a := range []string{"request-target", "from-value", "first-local"} {
		for _, t := range []string{"definition", "references", "definition+references"} {
			for _, s := range []string{"local", "global"} {
				for _, g := range []string{"equals-from", "any"} {
					for _, l := range []string{"required", "none"} {
						out = append(out, schema{a, t, s, g, l})
					}
				}
			}
		}
	}
	return out
}

func description(s schema) int {
	cost := map[string]int{"request-target": 1, "from-value": 2, "first-local": 3, "definition": 1, "references": 1, "definition+references": 2, "local": 1, "global": 2, "equals-from": 2, "any": 1, "required": 2, "none": 1}
	return cost[s.anchor] + cost[s.targets] + cost[s.scope] + cost[s.guard] + cost[s.locality]
}

func compareSchema(a, b schema) int {
	if d := description(a) - description(b); d != 0 {
		return d
	}
	orders := [][]string{{"request-target", "from-value", "first-local"}, {"definition", "references", "definition+references"}, {"local", "global"}, {"equals-from", "any"}, {"required", "none"}}
	av, bv := []string{a.anchor, a.targets, a.scope, a.guard, a.locality}, []string{b.anchor, b.targets, b.scope, b.guard, b.locality}
	for i := range av {
		if d := slices.Index(orders[i], av[i]) - slices.Index(orders[i], bv[i]); d != 0 {
			return d
		}
	}
	return 0
}

func encodeSchema(s schema) []byte {
	b, _ := json.Marshal([]any{"transform-schema/v1", s.anchor, s.targets, s.scope, s.guard, s.locality})
	return b
}

func apply(data []byte, s schema) (string, []byte, error) {
	f, err := parseForest(data)
	if err != nil {
		return "invalid-input", nil, err
	}
	var reqs, defs []node
	for _, n := range f.nodes {
		if n.kind == "request" {
			reqs = append(reqs, n)
		}
		if n.kind == "definition" {
			defs = append(defs, n)
		}
	}
	if len(reqs) != 1 {
		return "abstain/request-count", nil, nil
	}
	rq := reqs[0]
	var matches []node
	for _, d := range defs {
		if s.anchor == "request-target" && d.id == rq.target || s.anchor == "from-value" && d.value == rq.from || s.anchor == "first-local" && d.parent == rq.parent {
			matches = append(matches, d)
			if s.anchor == "first-local" {
				break
			}
		}
	}
	if len(matches) != 1 {
		return "abstain/anchor", nil, nil
	}
	def := matches[0]
	if s.locality == "required" && def.parent != rq.parent {
		return "abstain/locality", nil, nil
	}
	var editIDs []int
	if s.targets == "definition" || s.targets == "definition+references" {
		editIDs = append(editIDs, def.id)
	}
	if s.targets == "references" || s.targets == "definition+references" {
		for _, n := range f.nodes {
			if n.kind == "reference" && n.target == def.id && (s.scope == "global" || n.parent == rq.parent) && (s.guard == "any" || n.value == rq.from) {
				editIDs = append(editIDs, n.id)
			}
		}
	}
	slices.Sort(editIDs)
	if len(editIDs) == 0 || len(editIDs) > 4 {
		return "abstain/expansion", nil, nil
	}
	for _, id := range editIDs {
		if f.nodes[id].value == rq.to {
			return "abstain/no-op", nil, nil
		}
	}
	for _, id := range editIDs {
		f.nodes[id].value = rq.to
	}
	return "applied", encodeForest(f), nil
}

func parseForest(data []byte) (forest, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return forest{}, errInvalid
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return forest{}, errInvalid
	}
	o, ok := v.([]any)
	if !ok || len(o) != 2 || o[0] != "typed-reference-forest/v1" {
		return forest{}, errInvalid
	}
	rows, ok := o[1].([]any)
	if !ok || len(rows) == 0 || len(rows) > 12 {
		return forest{}, errInvalid
	}
	f := forest{nodes: make([]node, len(rows))}
	seen := make([]bool, len(rows))
	keys := map[int]map[string]bool{}
	for _, raw := range rows {
		r, ok := raw.([]any)
		if !ok || len(r) != 8 {
			return forest{}, errInvalid
		}
		id, a := integer(r[0])
		kind, b := r[1].(string)
		parent, c := integer(r[2])
		key, e := r[3].(string)
		value, g := r[4].(string)
		from, h := r[5].(string)
		to, j := r[6].(string)
		target, k := integer(r[7])
		if !(a && b && c && e && g && h && j && k) || id < 0 || id >= len(rows) || seen[id] {
			return forest{}, errInvalid
		}
		seen[id] = true
		f.nodes[id] = node{id, parent, target, kind, key, value, from, to}
	}
	for _, n := range f.nodes {
		if !oneOf(n.kind, "group", "request", "definition", "reference", "decoy") {
			return forest{}, errInvalid
		}
		for _, x := range []string{n.key, n.value, n.from, n.to} {
			if x != "" && !lower(x) {
				return forest{}, errInvalid
			}
		}
		if n.kind == "group" {
			if n.parent != -1 || n.key != "" || n.value != "" || n.from != "" || n.to != "" || n.target != -1 {
				return forest{}, errInvalid
			}
			continue
		}
		if n.parent < 0 || n.parent >= len(f.nodes) || f.nodes[n.parent].kind != "group" || n.key == "" {
			return forest{}, errInvalid
		}
		if keys[n.parent] == nil {
			keys[n.parent] = map[string]bool{}
		}
		if keys[n.parent][n.key] {
			return forest{}, errInvalid
		}
		keys[n.parent][n.key] = true
		if n.kind == "request" && (n.value != "" || n.from == "" || n.to == "" || !definition(f, n.target)) {
			return forest{}, errInvalid
		}
		if (n.kind == "definition" || n.kind == "decoy") && (n.value == "" || n.from != "" || n.to != "" || n.target != -1) {
			return forest{}, errInvalid
		}
		if n.kind == "reference" && (n.value == "" || n.from != "" || n.to != "" || !definition(f, n.target)) {
			return forest{}, errInvalid
		}
	}
	canonical := encodeForest(f)
	if !bytes.Equal(canonical, data) {
		return forest{}, fmt.Errorf("%w: noncanonical", errInvalid)
	}
	return f, nil
}

func encodeForest(f forest) []byte {
	rows := make([]any, len(f.nodes))
	for i, n := range f.nodes {
		rows[i] = []any{n.id, n.kind, n.parent, n.key, n.value, n.from, n.to, n.target}
	}
	b, _ := json.Marshal([]any{"typed-reference-forest/v1", rows})
	return b
}
func integer(v any) (int, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, e := n.Int64()
	return int(i), e == nil && int64(int(i)) == i
}
func lower(v string) bool {
	if v == "" || len(v) > 16 {
		return false
	}
	for _, c := range []byte(v) {
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}
func oneOf(v string, values ...string) bool { return slices.Contains(values, v) }
func definition(f forest, id int) bool {
	return id >= 0 && id < len(f.nodes) && f.nodes[id].kind == "definition"
}
