package engine

import (
	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

// fireTaskRule fires a heuristic's IfTaskParts against a task, then executes ThenParts if all pass.
// Returns (fired, abort).
func (e *Engine) fireTaskRule(heuristic string, task *agenda.Task) (bool, bool) {
	h := e.Store.Get(heuristic)
	if h == nil {
		return false, false
	}

	e.VM.SetEnv("ArgU", dsl.StringVal(task.UnitName))

	// Check ifPotentiallyRelevant first (quick filter)
	if prog := h.GetString("ifPotentiallyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true
			}
			e.log(3, "    %s.ifPotentiallyRelevant error: %v", heuristic, err)
			return false, false
		}
		if !v.Truthy() {
			return false, false
		}
	}

	// Check ifTrulyRelevant
	if prog := h.GetString("ifTrulyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true
			}
			e.log(3, "    %s.ifTrulyRelevant error: %v", heuristic, err)
			return false, false
		}
		if !v.Truthy() {
			return false, false
		}
	}

	// Check ifWorkingOnTask
	if prog := h.GetString("ifWorkingOnTask"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true
			}
			e.log(3, "    %s.ifWorkingOnTask error: %v", heuristic, err)
			return false, false
		}
		if !v.Truthy() {
			return false, false
		}
	}

	// All conditions passed — execute ThenParts
	return true, e.executeThenParts(h, heuristic)
}

// fireUnitRule fires a heuristic against a unit (Level 2: when agenda is empty).
// Uses ifPotentiallyRelevant and ifTrulyRelevant, then ThenParts.
func (e *Engine) fireUnitRule(heuristic string, targetUnit string) (bool, bool) {
	h := e.Store.Get(heuristic)
	if h == nil {
		return false, false
	}

	e.VM.SetEnv("ArgU", dsl.StringVal(targetUnit))
	e.VM.SetEnv("CurUnit", dsl.StringVal(targetUnit))

	// Check ifPotentiallyRelevant
	if prog := h.GetString("ifPotentiallyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true
			}
			e.log(3, "    %s.ifPotentiallyRelevant error: %v", heuristic, err)
			return false, false
		}
		if !v.Truthy() {
			return false, false
		}
	}

	// Check ifTrulyRelevant
	if prog := h.GetString("ifTrulyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true
			}
			e.log(3, "    %s.ifTrulyRelevant error: %v", heuristic, err)
			return false, false
		}
		if !v.Truthy() {
			return false, false
		}
	}

	// All conditions passed — execute ThenParts
	return true, e.executeThenParts(h, heuristic)
}

// executeThenParts runs all ThenPart slots of a heuristic. Returns true if abort.
func (e *Engine) executeThenParts(h *unit.Unit, heuristicName string) bool {
	for _, slot := range unit.ThenPartSlots() {
		prog := h.GetString(slot)
		if prog == "" {
			continue
		}
		_, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true
			}
			e.log(3, "    %s.%s error: %v", heuristicName, slot, err)
		}
	}
	return false
}
