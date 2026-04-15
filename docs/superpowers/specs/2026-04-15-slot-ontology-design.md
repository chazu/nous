# Slot Ontology Design

**Date:** 2026-04-15
**Status:** Approved

## Overview

Make slots first-class units in the nous store, matching EURISKO's original slot ontology. Slots get properties (DataType, Inverse, SuperSlots, SubSlots, SibSlots, CriterialSlot/NonCriterialSlot classification, DontCopy, ElimSlots, Format, DoubleCheck). Inverse slot relationships are automatically maintained by the store. New DSL builtins let heuristics reason about slot structure.

## Decisions

- **Inverse maintenance:** Store-level (A). Inverse pairs loaded from slot definition units, registered with the store. `Store.SetSlot()` maintains both sides automatically on every write. Runtime-created units get correct inverses.
- **File location:** `domains/common/slots.cue` alongside base types (A). All shared infrastructure in one place.
- **Fidelity:** Full EURISKO fidelity (A). All properties defined on all slot units even if not immediately consumed. CUE data costs nothing; avoids revisiting files when future heuristics need the data.

## Slot Definition Units

`domains/common/slots.cue` defines ~30 slot units. Each is a `#Unit` with slot-specific properties.

Two classification units:
- `CriterialSlot` (worth 500, isA: ["Slot", "Anything"]) -- slots that matter for identity/equivalence
- `NonCriterialSlot` (worth 500, isA: ["Slot", "Anything"]) -- metadata slots

### Criterial Slots
Arity, Domain, Range, Examples, NonExamples, Defn, FastDefn, Alg, FastAlg, CompiledDefn, UnitizedDefn, IterativeDefn, RecursiveDefn, NecDefn, SufDefn

### Non-Criterial Slots
Worth, IsA, English, Abbrev, Creditors, Applics, IntApplics, Generalizations, Specializations, Interestingness, Rarity, OverallRecord, Inverse, Restrictions, Extensions, InDomainOf, IsRangeOf, Conjectures, ConjectureAbout, Generator, Format, WhyInt, MoreInteresting, LessInteresting, SuperSlots, SubSlots, SibSlots, DataType, DontCopy, ElimSlots, DoubleCheck

### Slot Properties

Each slot unit gets these fields:
- `dataType`: what type of value this slot holds (e.g., "Unit", "Integer", "Text", "LispPred")
- `inverse`: the inverse slot name (e.g., Domain's inverse is InDomainOf)
- `superSlots`: parent slots in the slot hierarchy
- `subSlots`: child slots
- `sibSlots`: sibling slots at the same level
- `dontCopy`: boolean -- should this slot be copied when creating specializations?
- `elimSlots`: slots that should be cleared when this slot changes
- `format`: description of the slot's value format
- `doubleCheck`: boolean -- should changes to this slot be verified?

### Inverse Pairs

| Slot | Inverse |
|---|---|
| Generalizations | Specializations |
| Domain | InDomainOf |
| Range | IsRangeOf |
| Extensions | Restrictions |
| Inverse | Inverse |
| MoreInteresting | LessInteresting |
| SuperSlots | SubSlots |
| IsA | Examples |
| IntExamples | IsAInt |
| Conjectures | ConjectureAbout |

## Store Changes

### New Fields

```go
type Store struct {
    mu       sync.RWMutex
    units    map[string]*Unit
    inverses map[string]string  // slot name -> inverse slot name
}
```

### New Methods

```go
// RegisterInverse registers a bidirectional inverse relationship.
func (s *Store) RegisterInverse(slot, inverse string)

// SetSlot sets a slot on a unit and maintains inverse relationships.
// For []string slot values with a registered inverse, appends the unit name
// to the inverse slot on each target unit.
func (s *Store) SetSlot(unitName, slotName string, value any)
```

`SetSlot` calls `u.Set(slotName, value)` then, if `slotName` has a registered inverse, extracts unit references from the value and updates the inverse slot on each referenced unit. Unit references are extracted from `[]string` values (list of unit names) or single `string` values (one unit name). Non-string values are ignored for inverse purposes.

### DSL Integration

The `bSetSlot` builtin in `builtins.go` changes from `u.Set()` to `vm.Store.SetSlot()` to get inverse maintenance for heuristic-created units.

## Loading Order

1. Load `domains/common/*.cue` via `u.Set()` (no inverse maintenance yet)
2. Load `domains/<name>/*.cue` via `u.Set()` (no inverse maintenance yet)
3. Scan loaded units for `isA` containing `"Slot"` with an `inverse` field -- register inverse pairs with the store
4. Compute initial inverse index: scan all units, for each slot with a registered inverse, build the reverse mapping
5. From this point forward, `Store.SetSlot()` maintains inverses for runtime changes

This avoids ordering issues -- everything loads normally, then inverses are computed in a single pass.

## DSL Builtins

| Builtin | Stack effect | What it does |
|---|---|---|
| `all-slots` | `(unitName -- list)` | Push list of all populated slot names for a unit |
| `criterial-slots` | `(unitName -- list)` | Push populated slots that are CriterialSlot units |
| `non-criterial-slots` | `(unitName -- list)` | Push populated slots that are NonCriterialSlot units |
| `sib-slots` | `(slotName -- list)` | Push sibSlots from the slot definition unit |
| `super-slots` | `(slotName -- list)` | Push superSlots from the slot definition unit |
| `sub-slots` | `(slotName -- list)` | Push subSlots from the slot definition unit |
| `inverse-slot` | `(slotName -- name)` | Push inverse slot name (or nil) |
| `slot-type` | `(slotName -- typeName)` | Push dataType of a slot definition |

`criterial-slots` and `non-criterial-slots` iterate a unit's populated slot keys, look up each key as a unit in the store, check if it `isA CriterialSlot` or `isA NonCriterialSlot`, and return matches.

The other builtins (`sib-slots`, `super-slots`, etc.) look up the slot name as a unit and read the corresponding property. If the slot definition unit doesn't exist, they return nil or empty list.

## Testing

- **Slot definitions:** Loading common/ produces CriterialSlot, NonCriterialSlot, and all slot units with correct properties
- **Inverse registration:** After loading math domain, Set has `isRangeOf` containing SetUnion, SetIntersect, SetDifference, DivisorsOf
- **Runtime maintenance:** `Store.SetSlot("NewOp", "range", []string{"Number"})` updates Number's isRangeOf
- **DSL builtins:** `"Domain" sib-slots` returns list containing "Range"; `"SetUnion" criterial-slots` returns populated criterial slot names
- **Regression:** 100-cycle math domain produces same behavior (slot ontology is additive)
