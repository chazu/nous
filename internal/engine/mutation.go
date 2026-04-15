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
	Enabled           bool
	Interval          int     // try mutation every N cycles
	MaxMutants        int     // max live mutant heuristics at once
	MutantWorth       int     // starting worth for mutant heuristics
	ValidateOnly      bool    // if true, only keep mutations that pass validation
	MinApplics        int     // minimum total applications before considering mutation
	MutationThreshold float64 // success ratio below which a heuristic is eligible
}

// DefaultMutationConfig returns sensible defaults.
func DefaultMutationConfig() MutationConfig {
	return MutationConfig{
		Enabled:           true,
		Interval:          10,
		MaxMutants:        20,
		MutantWorth:       400,
		ValidateOnly:      true,
		MinApplics:        10,
		MutationThreshold: 0.3,
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

	// Pick the worst-performing heuristic to mutate
	parent := e.pickWorstPerformer()
	if parent == nil {
		e.log(2, "  Mutation: no underperforming heuristics found")
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
	parentCreditors := parent.GetStrings("creditors")
	mutantCreditors := append([]string{parent.Name}, parentCreditors...)
	m.Set("creditors", mutantCreditors)
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

// pickWorstPerformer selects the heuristic with the lowest success ratio
// that has enough applications and falls below the mutation threshold.
// Ties are broken by lowest worth.
func (e *Engine) pickWorstPerformer() *unit.Unit {
	heuristics := e.Store.Examples("Heuristic")
	if len(heuristics) == 0 {
		return nil
	}

	type candidate struct {
		name  string
		ratio float64
		worth int
	}
	var candidates []candidate

	for _, name := range heuristics {
		if name == "Heuristic" {
			continue
		}
		u := e.Store.Get(name)
		if u == nil {
			continue
		}
		record := u.GetMap("overallRecord")
		if record == nil {
			continue
		}
		successes := toInt(record["successes"])
		failures := toInt(record["failures"])
		total := successes + failures
		if total < e.MutConfig.MinApplics {
			continue
		}
		ratio := float64(successes) / float64(total)
		if ratio >= e.MutConfig.MutationThreshold {
			continue
		}
		candidates = append(candidates, candidate{name, ratio, u.Worth()})
	}

	if len(candidates) == 0 {
		return nil
	}

	// Sort: lowest ratio first, then lowest worth as tiebreaker
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ratio != candidates[j].ratio {
			return candidates[i].ratio < candidates[j].ratio
		}
		return candidates[i].worth < candidates[j].worth
	})

	return e.Store.Get(candidates[0].name)
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
