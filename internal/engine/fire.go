package engine

import (
	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/unit"
)

// fireTaskRule fires a heuristic's IfTaskParts against a task, then executes ThenParts if all pass.
// Returns (fired, abort, produced).
func (e *Engine) fireTaskRule(heuristic string, task *agenda.Task) (bool, bool, bool) {
	h := e.Store.Get(heuristic)
	if h == nil {
		return false, false, false
	}

	e.VM.SetEnv("ArgU", dsl.StringVal(task.UnitName))

	// Check ifAboutToWorkOnTask first — pre-flight veto slot used by HAvoid
	// rules (H12) and HAvoidIfWorking to abort the whole task before any
	// other heuristic fires. Falsy result means "this heuristic doesn't
	// match, skip it"; calling `abort` from inside aborts the task.
	if prog := h.GetString("ifAboutToWorkOnTask"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, false
			}
			e.log(3, "    %s.ifAboutToWorkOnTask error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	// Check ifPotentiallyRelevant first (quick filter)
	if prog := h.GetString("ifPotentiallyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, false
			}
			e.log(3, "    %s.ifPotentiallyRelevant error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	// Check ifTrulyRelevant
	if prog := h.GetString("ifTrulyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, false
			}
			e.log(3, "    %s.ifTrulyRelevant error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	// Check ifWorkingOnTask
	if prog := h.GetString("ifWorkingOnTask"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, false
			}
			e.log(3, "    %s.ifWorkingOnTask error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	// All conditions passed — execute ThenParts
	abort, produced := e.executeThenParts(h, heuristic)
	return true, abort, produced
}

// fireFinishedRule runs a heuristic's ifFinishedWorkingOnTask slot if present.
// Executed after all ThenParts of the current task have completed.
// Used by HAvoid2/HAvoid3 (H13/H14) to kill bad newly-created units.
func (e *Engine) fireFinishedRule(heuristic string, task *agenda.Task) {
	h := e.Store.Get(heuristic)
	if h == nil {
		return
	}
	prog := h.GetString("ifFinishedWorkingOnTask")
	if prog == "" {
		return
	}
	e.VM.SetEnv("ArgU", dsl.StringVal(task.UnitName))
	_, err := e.VM.Execute(prog)
	if err != nil && !dsl.IsAbort(err) {
		e.log(3, "    %s.ifFinishedWorkingOnTask error: %v", heuristic, err)
	}
}

// fireUnitRule fires a heuristic against a unit (Level 2: when agenda is empty).
// Uses ifPotentiallyRelevant and ifTrulyRelevant, then ThenParts.
// Returns (fired, abort, produced).
//
// Heuristics whose only conditions are task-phase (ifWorkingOnTask,
// ifFinishedWorkingOnTask) are skipped in unit-focus — they're slot-triggered
// by design and have no meaningful unit-focus trigger.
func (e *Engine) fireUnitRule(heuristic string, targetUnit string) (bool, bool, bool) {
	h := e.Store.Get(heuristic)
	if h == nil {
		return false, false, false
	}

	potProg := h.GetString("ifPotentiallyRelevant")
	relProg := h.GetString("ifTrulyRelevant")
	if potProg == "" && relProg == "" {
		return false, false, false
	}

	e.VM.SetEnv("ArgU", dsl.StringVal(targetUnit))
	e.VM.SetEnv("CurUnit", dsl.StringVal(targetUnit))
	// CurSlot must not leak from the previous task dispatch.
	e.VM.SetEnv("CurSlot", dsl.Nil())

	// Check ifPotentiallyRelevant
	if prog := potProg; prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, false
			}
			e.log(3, "    %s.ifPotentiallyRelevant error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	// Check ifTrulyRelevant
	if prog := relProg; prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, false
			}
			e.log(3, "    %s.ifTrulyRelevant error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	// All conditions passed — execute ThenParts
	abort, produced := e.executeThenParts(h, heuristic)
	return true, abort, produced
}

// executeThenParts runs all ThenPart slots of a heuristic.
// Returns (abort, producedOutput). producedOutput is true if the store grew
// or the agenda grew as a result of executing the ThenParts.
func (e *Engine) executeThenParts(h *unit.Unit, heuristicName string) (bool, bool) {
	storeBefore := e.Store.Count()
	agendaBefore := e.Agenda.Len()

	for _, slot := range unit.ThenPartSlots() {
		prog := h.GetString(slot)
		if prog == "" {
			continue
		}
		_, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				// Abort counts neither as success nor failure — it's
				// a deliberate veto signal, not an execution error.
				return true, false
			}
			e.log(3, "    %s.%s error: %v", heuristicName, slot, err)
			e.LastError = err
			e.trackThenPartRecord(heuristicName, slot, false)
			continue
		}
		e.trackThenPartRecord(heuristicName, slot, true)
	}

	produced := e.Store.Count() > storeBefore || e.Agenda.Len() > agendaBefore
	return false, produced
}
