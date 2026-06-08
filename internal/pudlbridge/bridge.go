// Package pudlbridge connects nous to pudl's bitemporal fact store
// and Datalog evaluator for Mode 2 operation.
package pudlbridge

import (
	"encoding/json"
	"fmt"

	"github.com/chazu/pudl/pkg/eval"
	"github.com/chazu/pudl/pkg/factstore"

	"github.com/chazu/nous/internal/unit"
)

// Bridge provides read/write access to pudl's fact store and Datalog evaluator.
// Query execution lives on factstore.Store.Query; the bridge holds the store and
// adapts results into nous units.
type Bridge struct {
	store *factstore.Store
}

// New opens a bridge to the pudl store at the given config directory.
func New(pudlDir string) (*Bridge, error) {
	store, err := factstore.Open(pudlDir)
	if err != nil {
		return nil, fmt.Errorf("open pudl: %w", err)
	}
	return &Bridge{store: store}, nil
}

// Close releases the database connection.
func (b *Bridge) Close() error {
	return b.store.Close()
}

// QueryFacts evaluates rules and returns matching tuples for a relation.
func (b *Bridge) QueryFacts(rules []eval.Rule, relation string, constraints map[string]interface{}) ([]eval.Tuple, error) {
	return b.store.Query(factstore.QueryOptions{
		Relation:    relation,
		Constraints: constraints,
		Rules:       rules,
	})
}

// ScanFacts returns raw facts for a relation (no rule evaluation). A query with
// no producing rule returns matching base facts directly.
func (b *Bridge) ScanFacts(relation string) ([]eval.Tuple, error) {
	return b.store.Query(factstore.QueryOptions{Relation: relation})
}

// WriteFact records a fact in the bitemporal store.
func (b *Bridge) WriteFact(relation string, args map[string]interface{}, source string) (factstore.Fact, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return factstore.Fact{}, fmt.Errorf("marshal args: %w", err)
	}

	return b.store.AddFact(factstore.Fact{
		Relation: relation,
		Args:     string(argsJSON),
		Source:   source,
	})
}

// LoadObservations reads all current observations from pudl and creates
// units in the nous store. Each observation becomes a unit with isA=["Observation"]
// and slots populated from the fact's args.
func (b *Bridge) LoadObservations(s *unit.Store) (int, error) {
	tuples, err := b.ScanFacts("observation")
	if err != nil {
		return 0, fmt.Errorf("scan observations: %w", err)
	}

	count := 0
	for _, t := range tuples {
		name := observationName(t, count)
		if s.Has(name) {
			continue
		}

		u := unit.New(name)
		u.Set("isA", []string{"Observation", "Anything"})

		// Map args to slots
		for k, v := range t.Args {
			u.Set(k, v)
		}

		// Set worth from the observation's worth field, default 500
		worth := 500
		if w, ok := t.Args["worth"]; ok {
			if wf, ok := w.(float64); ok {
				worth = int(wf * 1000)
			}
		}
		u.Set("worth", worth)

		s.Put(u)
		count++
	}

	return count, nil
}

// LoadDerived evaluates rules and loads derived facts as units. The current
// query API is per-relation, so the rules' distinct head relations are each
// queried and their derived tuples collected — equivalent to evaluating the
// whole program and taking every derived fact.
func (b *Bridge) LoadDerived(s *unit.Store, rules []eval.Rule) (int, error) {
	heads := make([]string, 0, len(rules))
	seen := make(map[string]bool, len(rules))
	for _, r := range rules {
		if !seen[r.Head.Rel] {
			seen[r.Head.Rel] = true
			heads = append(heads, r.Head.Rel)
		}
	}

	count := 0
	for _, rel := range heads {
		idb, err := b.store.Query(factstore.QueryOptions{Relation: rel, Rules: rules})
		if err != nil {
			return count, fmt.Errorf("evaluate %s: %w", rel, err)
		}
		for _, t := range idb {
			name := derivedName(t, count)
			if s.Has(name) {
				continue
			}

			u := unit.New(name)
			u.Set("isA", []string{t.Relation, "DerivedFact", "Anything"})
			u.Set("worth", 400)

			for k, v := range t.Args {
				u.Set(k, v)
			}

			s.Put(u)
			count++
		}
	}

	return count, nil
}

func observationName(t eval.Tuple, idx int) string {
	if desc, ok := t.Args["description"].(string); ok {
		if len(desc) > 40 {
			desc = desc[:40]
		}
		return fmt.Sprintf("Obs-%s", sanitize(desc))
	}
	return fmt.Sprintf("Obs-%d", idx)
}

func derivedName(t eval.Tuple, idx int) string {
	name := t.Relation
	for k, v := range t.Args {
		name += fmt.Sprintf("-%s=%v", k, v)
		if len(name) > 60 {
			break
		}
	}
	return sanitize(name)
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
			out = append(out, c)
		case c == ' ':
			out = append(out, '-')
		}
	}
	return string(out)
}
