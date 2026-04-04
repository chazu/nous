// Package engine implements the nous main loop — the agenda-driven
// heuristic interpreter directly modeled on EURISKO's control flow.
package engine

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/mutate"
	"github.com/chazu/nous/internal/unit"
)

// GraveRecord holds metadata about a killed unit for HindSight analysis.
type GraveRecord struct {
	Name      string
	IsA       []string
	Creditors []string
	Worth     int
	Slots     map[string]any // snapshot of key slots at death
	Cycle     int
}

// Engine is the nous reasoning engine.
type Engine struct {
	Store     *unit.Store
	Agenda    *agenda.Agenda
	VM        *dsl.VM
	TaskNum   int
	focused   map[string]bool
	MaxCycles int
	Verbosity int
	Out       io.Writer
	cycle     int

	// Graveyard: records of units that were killed, for HindSight analysis
	Graveyard []GraveRecord

	// Mutation
	MutConfig MutationConfig
	mutator   *mutate.Mutator
	rng       *rand.Rand
}

// New creates an engine wired to the given store and agenda.
func New(store *unit.Store, ag *agenda.Agenda) *Engine {
	vm := dsl.NewVM(store, ag)
	rng := rand.New(rand.NewSource(42))
	return &Engine{
		Store:     store,
		Agenda:    ag,
		VM:        vm,
		focused:   make(map[string]bool),
		MaxCycles: 100,
		Verbosity: 1,
		Out:       os.Stdout,
		MutConfig: DefaultMutationConfig(),
		mutator:   mutate.New(rng, store),
		rng:       rng,
	}
}

// Run is the main loop, following EURISKO's two-level control:
// Level 1: agenda-driven (pop highest-priority task, work on it)
// Level 2: unit-focused (when agenda empty, focus on highest-Worth unit)
func (e *Engine) Run(ctx context.Context) error {
	e.VM.Out = e.Out

	for e.cycle = 0; e.cycle < e.MaxCycles; e.cycle++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Level 1: agenda-driven
		if task := e.Agenda.Pop(); task != nil {
			e.log(1, "\n=== Cycle %d: Task [pri=%d] %s.%s ===", e.cycle, task.Priority, task.UnitName, task.SlotName)
			e.log(2, "  Reasons: %v", task.Reasons)
			e.WorkOnTask(task)
			e.processDeletedUnits()
			continue
		}

		// Level 2: unit-focused
		u := e.highestWorthUnfocused()
		if u == "" {
			e.log(1, "\n=== Cycle %d: All units focused, resetting ===", e.cycle)
			e.focused = make(map[string]bool)
			// If agenda is still empty after reset, we're done
			if e.Agenda.Len() == 0 {
				e.log(1, "Agenda empty and all units focused. Stopping.")
				return nil
			}
			continue
		}
		e.log(1, "\n=== Cycle %d: Focus on %s (worth=%d) ===", e.cycle, u, e.Store.Get(u).Worth())
		e.WorkOnUnit(u)
		e.focused[u] = true

		// Post-cycle: process any units deleted during this cycle
		e.processDeletedUnits()

		// Periodic mutation
		if e.MutConfig.Enabled && e.cycle > 0 && e.cycle%e.MutConfig.Interval == 0 {
			e.tryMutateHeuristic()
		}
	}

	e.log(1, "\nReached max cycles (%d). Stopping.", e.MaxCycles)
	return nil
}

// WorkOnTask tries every heuristic against the current task.
func (e *Engine) WorkOnTask(task *agenda.Task) {
	e.TaskNum++

	e.VM.SetEnv("CurUnit", dsl.StringVal(task.UnitName))
	e.VM.SetEnv("CurSlot", dsl.StringVal(task.SlotName))
	e.VM.SetEnv("CurPri", dsl.IntVal(task.Priority))
	e.VM.SetEnv("TaskNum", dsl.IntVal(e.TaskNum))

	heuristics := e.Store.Examples("Heuristic")
	for _, h := range heuristics {
		fired, abort := e.fireTaskRule(h, task)
		if fired {
			e.trackApplics(h, task.UnitName, true)
			e.log(2, "  Heuristic %s fired on task %s.%s", h, task.UnitName, task.SlotName)
		}
		if abort {
			e.log(2, "  Task aborted by %s", h)
			return
		}
	}
}

// WorkOnUnit tries every heuristic against the given unit.
func (e *Engine) WorkOnUnit(u string) {
	e.VM.SetEnv("CurUnit", dsl.StringVal(u))
	e.VM.SetEnv("ArgU", dsl.StringVal(u))

	heuristics := e.Store.Examples("Heuristic")
	for _, h := range heuristics {
		fired, _ := e.fireUnitRule(h, u)
		if fired {
			e.trackApplics(h, u, true)
			e.log(2, "  Heuristic %s fired on unit %s", h, u)
		}
	}
}

// highestWorthUnfocused returns the unit with the highest Worth that
// hasn't been focused this round. Skips heuristics and meta-units.
func (e *Engine) highestWorthUnfocused() string {
	var best string
	bestWorth := -1

	// Get all units sorted by name for determinism
	names := e.Store.All()
	sort.Strings(names)

	for _, name := range names {
		if e.focused[name] {
			continue
		}
		u := e.Store.Get(name)
		if u == nil {
			continue
		}
		// Skip heuristics for unit-focus (they fire on others, not on themselves)
		if e.Store.IsA(name, "Heuristic") {
			continue
		}
		// Skip meta-concepts (Slot, IfParts, ThenParts, etc.)
		if e.Store.IsA(name, "Slot") {
			continue
		}
		w := u.Worth()
		if w > bestWorth {
			bestWorth = w
			best = name
		}
	}
	return best
}

// processDeletedUnits handles credit assignment and HindSight for units
// killed during the current cycle.
func (e *Engine) processDeletedUnits() {
	deleted := e.VM.DeletedUnits
	if len(deleted) == 0 {
		return
	}
	e.VM.DeletedUnits = nil

	for _, name := range deleted {
		e.HandleDeletedUnit(name)
	}
}

func (e *Engine) log(level int, format string, args ...any) {
	if e.Verbosity >= level {
		fmt.Fprintf(e.Out, format+"\n", args...)
	}
}

// Cycle returns the current cycle number.
func (e *Engine) Cycle() int {
	return e.cycle
}

// Stats returns a summary of the current state.
func (e *Engine) Stats() string {
	return fmt.Sprintf("units=%d agenda=%d tasks=%d cycles=%d",
		e.Store.Count(), e.Agenda.Len(), e.TaskNum, e.cycle)
}
