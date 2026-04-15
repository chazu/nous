# Generalization/Specialization Pipeline Design

**Date:** 2026-04-15
**Status:** Approved

## Overview

Implement EURISKO's multi-step generalization/specialization pipeline. H-Specialize evolves from a direct-action heuristic to a task emitter. H6 becomes the workhorse that actually creates specialized units. H3/H5 variants provide slot selection strategies. H16/H17/H18 mirror the pipeline for generalization. Supplementary task info (Extra map) carries context between pipeline stages.

## Decisions

- **Specialize/generalize implementation:** Type-aware helpers (A). Slot ontology's `dataType` field determines how each slot is specialized/generalized. `Unit`-typed slots use the isA hierarchy. `LispPred`/`LispFn` slots delegate to the mutation system. `Integer` slots increment/decrement.
- **Supplementary task info:** `Extra map[string]any` on Task (A). Scoped to the task, no leakage across heuristic firings. Tasks with different Extra values are not merged.
- **H-Specialize evolution:** Convert to task emitter (C). Existing behavior preserved as the trigger; H6 does the work. Incremental, minimal disruption.
- **Generalization direction:** Use `generalizations` slot (matching EURISKO). Already populated via inverse maintenance from Phase 1.

## Supplementary Task Info

### Task struct change

```go
type Task struct {
    Priority int
    UnitName string
    SlotName string
    Reasons  []string
    Extra    map[string]any
}
```

### Duplicate merging update

Tasks with the same `UnitName + SlotName` but different `Extra["SlotToChange"]` values are NOT merged. Tasks with identical `Extra` contents are merged as before (max priority + 50 boost, concatenate reasons).

### VM access to current task

Add `CurrentTask *agenda.Task` field to the VM. Set by `WorkOnTask` before firing heuristics. Cleared after the task is processed.

### New DSL builtins

| Builtin | Stack effect | Purpose |
|---|---|---|
| `get-task-extra` | `(key -- value)` | Read from current task's Extra |
| `set-task-extra` | `(value key --)` | Write to current task's Extra |
| `random-choice` | `(list -- element)` | Pick random element from list |
| `random-subset` | `(list -- sublist)` | Pick random subset (each element included with 50% probability) |

The VM needs an `*rand.Rand` field for `random-choice` and `random-subset`. Seeded from the engine's RNG during VM creation.

## H-Specialize Evolution

Current behavior: for each domain type of an operation, creates specialized units directly.

New behavior: for each domain type, for each subtype in that type's specializations list, adds a task:

```
Priority: 600
UnitName: <the operation>
SlotName: "specializations"
Extra: {
    "SlotToChange": "domain",
    "SpecializeFrom": <type>,
    "SpecializeTo": <subtype>,
}
Reason: "Specialize domain type"
```

H-Specialize stops creating units. It becomes a trigger that feeds the pipeline.

## H6: Specialize a Given Slot (worth 700)

Fires when: `SlotName == "specializations"` AND `Extra["SlotToChange"]` is set.

Reads from task extras:
- `SlotToChange`: which slot to modify (e.g., "domain")
- `SpecializeFrom`: the current value to replace (e.g., "Set")
- `SpecializeTo`: the narrower replacement (e.g., "SetOfPrimes")

Creates a new unit:
- Name: `<parent>-on-<SpecializeTo>`
- isA: same as parent
- All slots copied from parent
- The changed slot: replaces `SpecializeFrom` with `SpecializeTo` in the value
- creditors: the heuristic that triggered the specialization task
- overallRecord: `{successes: 0, failures: 0}`
- Adds task to gather examples for the new unit

For `Unit`-typed slots (`domain`, `range`): value is `[]string`, replace the matching entry.
For other slot types: not handled in initial implementation (future: mutation for `defn`, increment for `arity`).

Skip if the specialized unit already exists (idempotent).

## H3: Random Slot Selection (worth 101)

Fires when: `SlotName == "specializations"` AND `Extra["SlotToChange"]` is NOT set.

Action: pick one random slot from the unit's populated criterial slots. Add a new task with `SlotToChange` set to the chosen slot, `SpecializeFrom` and `SpecializeTo` determined by looking up the current slot value's specializations.

Low worth because H5Criterial and H5Good are better strategies.

## H5/H5Criterial/H5Good: Targeted Slot Selection

Three variants, all fire on "specializations" tasks without `SlotToChange`:

**H5 (worth 151):** Random subset of all populated slots. Adds one task per selected slot.

**H5Criterial (worth 700):** Only criterial slots (uses `criterial-slots` builtin). More focused than H5. Subsumes H3 and H5.

**H5Good (worth 700):** Slots weighted by the slot definition unit's worth. High-worth slots (Domain at 600, Range at 600) are preferred over low-worth slots (English at 300).

Each variant:
1. Selects a set of slot names
2. For each selected slot, looks up the current value
3. For `Unit`-typed slots with `[]string` values, picks a random element from the list to specialize (e.g., for `domain: ["Set", "Set"]`, picks one of the two "Set" entries)
4. Finds specializations of the chosen type
5. Adds tasks with `SlotToChange`, `SpecializeFrom`, `SpecializeTo` set

## H16: Generalization Trigger (worth 600)

Fires during unit-focus on operations with some good applics.

Condition: success ratio > 0.1 in applics (at least some good results, suggesting generalization might find more).

Action: adds a "generalizations" task to the agenda:
```
Priority: 500
UnitName: <the operation>
SlotName: "generalizations"
Reason: "Operation has some good results, try generalizing"
```

## H17: Choose Slots to Generalize (worth 600)

Fires when: `SlotName == "generalizations"` AND `Extra["SlotToChange"]` is NOT set.

Action: picks a random subset of criterial slots. For each, looks up the current value's `generalizations` list to find candidates. Adds tasks with `SlotToChange`, `GeneralizeFrom`, `GeneralizeTo` set.

## H18: Generalize a Given Slot (worth 704)

Fires when: `SlotName == "generalizations"` AND `Extra["SlotToChange"]` is set.

Reads from task extras:
- `SlotToChange`: which slot to modify
- `GeneralizeFrom`: the current value
- `GeneralizeTo`: the wider replacement

Creates a new unit with the generalized slot value. Same mechanics as H6 but in the opposite direction.

Name: `<parent>-gen-<GeneralizeTo>`

## Heuristic Registration

All new heuristics defined in CUE at `domains/math/heuristics.cue` (append to existing file) and `domains/common/heuristics.cue` (for H6, H3, H5 variants which are domain-independent).

Domain-independent heuristics (work on any domain):
- H3, H5, H5Criterial, H5Good, H6, H17, H18

Domain-specific heuristics (need applics data):
- H16 (checks applics ratio, could be domain-independent but following EURISKO's placement)

## Testing

- **Supplementary task info:** Task with Extra is created, popped, Extra readable. Merge behavior correct (same Extra = merge, different Extra = don't merge).
- **H-Specialize evolution:** Verify it emits tasks instead of creating units directly. Tasks have correct Extra fields.
- **H6:** Given a task with SlotToChange/SpecializeFrom/SpecializeTo, creates correct specialized unit.
- **Pipeline integration:** H-Specialize emits task -> H6 picks it up -> specialized unit created. Same result as before but through the pipeline.
- **H17/H18:** Generalization task -> H17 picks slot -> H18 creates generalized unit.
- **Regression:** 100-cycle math run still produces specializations (now via pipeline instead of direct creation).
