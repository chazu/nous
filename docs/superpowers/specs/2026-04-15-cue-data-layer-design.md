# CUE Data Layer Design

**Date:** 2026-04-15
**Status:** Approved

## Overview

Move domain data (unit definitions, type hierarchies, heuristic programs) from hardcoded Go into CUE files loaded at runtime. The stack DSL is unchanged -- heuristic programs are string values in CUE fields. This establishes the infrastructure for all subsequent EURISKO parity work.

## Decisions

- **Schema strictness:** Known slots enumerated with types, open for extensions (option B). Catches typos in common slot names while allowing novel slots.
- **File organization:** Grouped by category (option B). ~5-8 files per domain, organized by concept type (types, operations, heuristics, etc.).
- **Domain discovery:** Convention-based (option A). `-domain math` loads `domains/common/` then `domains/math/`. Add a domain by creating a directory.
- **Loading strategy:** Layered. Domains are embedded in the binary via `//go:embed` as defaults. A `-domains-dir` flag loads from a filesystem path instead, for development without recompilation. Schema is always embedded.

## CUE Schema

`domains/schema.cue` defines `#Unit`:

```cue
package domains

#Unit: {
    name:  string
    worth: int & >=0 & <=1000

    isA: [...string]

    // Known slots with types
    english?:           string
    abbrev?:            string
    domain?:            [...string] | string
    range?:             [...string] | string
    arity?:             int
    defn?:              string
    data?:              [...int] | [...string]
    examples?:          _
    nonExamples?:       _
    generalizations?:   [...string]
    specializations?:   [...string]
    creditors?:         [...string]
    inverse?:           string
    status?:            string

    // Heuristic program slots (stack DSL strings)
    ifPotentiallyRelevant?:      string
    ifTrulyRelevant?:            string
    ifWorkingOnTask?:            string
    ifFinishedWorkingOnTask?:    string
    thenCompute?:                string
    thenAddToAgenda?:            string
    thenDefineNewConcepts?:      string
    thenDeleteOldConcepts?:      string
    thenPrintToUser?:            string
    thenConjecture?:             string
    thenModifySlots?:            string

    // Open for novel slots
    ...
}
```

Domain files contribute to a `units` list:

```cue
package math

import "domains"

units: [...domains.#Unit]
```

Each CUE file in a domain directory defines its own `units` list. The Go loader reads each file individually and concatenates the results -- it does not rely on CUE cross-file unification of lists, which has constraining (not concatenating) semantics. Each file is a self-contained CUE instance validated against the schema.

## Go Loader

New package `internal/cueload/` with:

```go
type UnitDef struct {
    Name  string
    Worth int
    IsA   []string
    Slots map[string]any
}

func LoadDomain(domainDir string) ([]UnitDef, error)
```

The loader:
1. Reads all `.cue` files from the directory (filesystem or embedded)
2. Evaluates as a CUE instance
3. Extracts the `units` list
4. For each entry: pulls `name`, `worth`, `isA` as typed fields, puts everything else into `Slots`
5. Returns the list, or error on CUE validation failure

Creating a unit from `UnitDef`: `unit.New(def.Name)`, `SetWorth(def.Worth)`, `Set("isA", def.IsA)`, iterate `def.Slots` and `Set` each.

## Layered Loading

`domains/schema.cue` is always embedded (`//go:embed`). Domain `.cue` files are embedded as defaults.

A `-domains-dir` CLI flag points to a filesystem directory. When set, the loader reads from the filesystem path for the requested domain. When not set, falls back to the embedded copy.

Development workflow: edit CUE files, run `nous run -domain math -domains-dir ./domains`, see changes immediately without recompilation.

Distribution: the binary is self-contained. `nous run -domain math` loads from embedded files.

## File Layout

```
domains/
  schema.cue
  common/
    types.cue             -- Anything, Heuristic, Slot, Op, Pred, BinaryOp, UnaryOp, etc.
  math/
    types.cue             -- Structure, Set, List, Bag, Number, MathConcept, MathObj, etc.
    sets.cue              -- SetOfPrimes, SetOfEvens, SetOfOdds, SetOfNumbers, EmptySet
    operations.cue        -- SetUnion, SetIntersect, SetDifference, GCD, DivisorsOf, Compose, Restrict
    predicates.cue        -- MemberOf, SubsetOf, SetEqual
    numbers.cue           -- EvenNum, OddNum, PrimeNum, PerfectNum, SquareNum, TruthValue
    conjectures.cue       -- GoldbachConjecture
    heuristics.cue        -- All 12 math heuristics
  observations/
    types.cue             -- Observation, DerivedFact, Conjecture, ScopeHotspot
    heuristics.cue        -- All 6 observation heuristics
```

## Migration

`internal/seed/math.go` content moves to `domains/math/*.cue`. `internal/seed/heuristics.go` content moves to `domains/math/heuristics.cue`. `internal/seed/observations.go` content moves to `domains/observations/*.cue`.

`internal/seed/registry.go` is updated to call `cueload.LoadDomain` instead of the Go domain loaders. The old Go seed files are deleted once regression tests pass.

Untouched: DSL builtins, engine, mutate, unit, agenda, pudlbridge packages.

## Testing

- CUE validation: test that malformed unit definitions are rejected (bad worth, missing name)
- Loader: test that CUE files produce the same units as the current Go seed code
- Regression: run the 100-cycle math domain, verify identical behavior (same conjectures, same unit kills, same HAvoid rules, same final worth rankings)
- Filesystem override: test that `-domains-dir` loads from disk and that missing directories fall back to embedded
