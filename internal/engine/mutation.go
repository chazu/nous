package engine

import (
	"fmt"
	"sort"

	"github.com/chazu/nous/internal/mutate"
	"github.com/chazu/nous/internal/unit"
)

// MutationConfig controls how often and how aggressively the engine
// mutates heuristics.
type MutationConfig struct {
	Enabled      bool
	Interval     int     // try mutation every N cycles
	MaxMutants   int     // max live mutant heuristics at once
	MutantWorth  int     // starting worth for mutant heuristics
	ValidateOnly bool    // if true, only keep mutations that pass validation
}

// DefaultMutationConfig returns sensible defaults.
func DefaultMutationConfig() MutationConfig {
	return MutationConfig{
		Enabled:      true,
		Interval:     10,
		MaxMutants:   20,
		MutantWorth:  400,
		ValidateOnly: true,
	}
}

// tryMutateHeuristic picks a heuristic weighted by worth, mutates one of
// its program slots, validates the result, and creates a new heuristic unit.
func (e *Engine) tryMutateHeuristic() {
	if e.mutator == nil || e.MutConfig.MaxMutants <= 0 {
		return
	}

	// Count existing mutants
	mutantCount := 0
	for _, name := range e.Store.All() {
		u := e.Store.Get(name)
		if u != nil && u.GetString("mutant_of") != "" {
			mutantCount++
		}
	}
	if mutantCount >= e.MutConfig.MaxMutants {
		e.log(2, "  Mutation: at mutant cap (%d/%d)", mutantCount, e.MutConfig.MaxMutants)
		return
	}

	// Pick a heuristic to mutate, weighted by worth
	parent := e.pickHeuristicByWorth()
	if parent == nil {
		return
	}

	// Pick a program slot to mutate
	slot, prog := e.pickProgramSlot(parent)
	if prog == "" {
		return
	}

	// Apply mutation
	mutated, op := e.mutator.Mutate(prog)
	if op == nil {
		return
	}

	// Validate
	if e.MutConfig.ValidateOnly && !mutate.Validate(mutated, e.Store) {
		e.log(3, "  Mutation: invalid mutant of %s.%s (%s), discarded", parent.Name, slot, op.Kind)
		return
	}

	// Create the mutant heuristic
	mutantName := fmt.Sprintf("M-%s-%d", parent.Name, e.cycle)
	if e.Store.Has(mutantName) {
		return
	}

	m := unit.New(mutantName)
	m.SetWorth(e.MutConfig.MutantWorth)
	m.Set("isA", []string{"Heuristic", "MutantHeuristic", "Anything"})
	m.Set("creditors", []string{parent.Name})
	m.Set("mutant_of", parent.Name)
	m.Set("mutation_op", op.Kind)
	m.Set("mutation_slot", slot)
	m.Set("mutation_from", op.From)
	m.Set("mutation_to", op.To)
	m.Set("mutation_cycle", e.cycle)
	m.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})

	// Copy all program slots from parent, replacing the mutated one
	for _, s := range programSlots() {
		p := parent.GetString(s)
		if p != "" {
			if s == slot {
				m.Set(s, mutated)
			} else {
				m.Set(s, p)
			}
		}
	}

	if parent.GetString("english") != "" {
		m.Set("english", fmt.Sprintf("Mutant of %s (%s %s.%s: %s→%s)",
			parent.Name, op.Kind, parent.Name, slot, op.From, op.To))
	}

	e.Store.Put(m)
	e.log(1, "  Mutation: created %s (%s on %s.%s: %s → %s)",
		mutantName, op.Kind, parent.Name, slot, op.From, op.To)
}

// pickHeuristicByWorth selects a heuristic weighted by worth.
// Higher-worth heuristics are more likely to be mutated (they're
// more likely to be good starting points).
func (e *Engine) pickHeuristicByWorth() *unit.Unit {
	heuristics := e.Store.Examples("Heuristic")
	if len(heuristics) == 0 {
		return nil
	}

	type candidate struct {
		name  string
		worth int
	}
	var candidates []candidate
	totalWorth := 0
	for _, name := range heuristics {
		u := e.Store.Get(name)
		if u == nil {
			continue
		}
		// Skip meta-heuristic types (HAvoid rules, the Heuristic type itself)
		if name == "Heuristic" {
			continue
		}
		w := u.Worth()
		if w <= 0 {
			continue
		}
		candidates = append(candidates, candidate{name, w})
		totalWorth += w
	}
	if len(candidates) == 0 || totalWorth == 0 {
		return nil
	}

	// Sort for determinism, then weighted random selection
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].name < candidates[j].name
	})

	r := e.rng.Intn(totalWorth)
	cumulative := 0
	for _, c := range candidates {
		cumulative += c.worth
		if r < cumulative {
			return e.Store.Get(c.name)
		}
	}
	return e.Store.Get(candidates[len(candidates)-1].name)
}

// pickProgramSlot returns a random non-empty program slot from a heuristic.
func (e *Engine) pickProgramSlot(h *unit.Unit) (string, string) {
	var slots []string
	for _, s := range programSlots() {
		if h.GetString(s) != "" {
			slots = append(slots, s)
		}
	}
	if len(slots) == 0 {
		return "", ""
	}
	slot := slots[e.rng.Intn(len(slots))]
	return slot, h.GetString(slot)
}

func programSlots() []string {
	return []string{
		"ifPotentiallyRelevant",
		"ifTrulyRelevant",
		"ifWorkingOnTask",
		"ifFinishedWorkingOnTask",
		"thenCompute",
		"thenAddToAgenda",
		"thenDefineNewConcepts",
		"thenDeleteOldConcepts",
		"thenPrintToUser",
		"thenConjecture",
	}
}
