# Phase 5.3 + 5.4 — Projections and Structure Classification

**Date:** 2026-04-20
**Scope:** EURISKO parity phases 5.3 + 5.4 (`docs/eurisko-parity-phases.md`).

## Goal

Add six structure-classification categories (`OrdStruc`, `UnOrdStruc`, `MultEleStruc`, `NoMultEleStruc`, `EmptyStruc`, `NonEmptyStruc`) as marker units in the type hierarchy, and six projection operations (`Proj1`, `Proj2`, `FirstEle`, `LastEle`, `AllButFirst`, `AllButLast`) that use those categories as domain restrictions. Together these two phases give H-Specialize a richer structural vocabulary — it can now restrict a domain to "any ordered structure" rather than only to concrete types.

## Why this is the right shape

The two phases are tightly linked: projections such as `FirstEle` only make sense on structures with a definite order, so their `domain` slot naturally points at a classification category (`OrdStruc`), which must exist first. EURISKO's convention is that classifications live in the `isA` chain — a two-line CUE change per type gives immediate traction to existing `store.IsA` and `is-a?`. No new gate machinery.

We pick a deliberately tight slice: six classification dimensions (the three binary axes: order, multiplicity, emptiness) and six projections (the four ordered-structure projections + the two OPair projections). `SecondEle`/`ThirdEle`/`AllButSecond`/`AllButThird` are redundant with `rest`-chain composition; `SetOfSets`/`StructureOfStructures` higher-order categories have no instances yet. Both groups are deferred.

## Components

### 1. Classification category units (`domains/math/types.cue`)

Six new category units appended to `units: [...]`:

```cue
{
    name: "OrdStruc"
    worth: 500
    isA: ["Structure", "MathObj", "Anything"]
    english: "A structure whose elements have a definite order"
    specializations: ["OSet", "List"]
}
{
    name: "UnOrdStruc"
    worth: 500
    isA: ["Structure", "MathObj", "Anything"]
    english: "A structure whose elements have no definite order"
    specializations: ["Set", "Bag"]
}
{
    name: "MultEleStruc"
    worth: 500
    isA: ["Structure", "MathObj", "Anything"]
    english: "A structure that may contain duplicates"
    specializations: ["List", "Bag"]
}
{
    name: "NoMultEleStruc"
    worth: 500
    isA: ["Structure", "MathObj", "Anything"]
    english: "A structure that rejects duplicate elements"
    specializations: ["Set", "OSet"]
}
{
    name: "EmptyStruc"
    worth: 400
    isA: ["Structure", "MathObj", "Anything"]
    english: "A structure containing no elements"
    specializations: ["EmptySet"]
}
{
    name: "NonEmptyStruc"
    worth: 400
    isA: ["Structure", "MathObj", "Anything"]
    english: "A structure containing at least one element"
    specializations: ["SetOfNumbers", "SetOfPrimes", "SetOfEvens", "SetOfOdds",
                      "OSetOfNumbers", "OSetOfPrimesDesc"]
}
```

No `defn` — pure marker categories. `store.IsA(u, "OrdStruc")` walks the chain; that is the only access path we need.

### 2. Updated type/instance isA lists

Modify existing units' `isA` to include the relevant classifications (prepended before `"Structure"`):

- `Set` (`domains/math/types.cue`) — add `"UnOrdStruc", "NoMultEleStruc"`
- `List` — add `"OrdStruc", "MultEleStruc"`
- `Bag` — add `"UnOrdStruc", "MultEleStruc"`
- `OSet` — add `"OrdStruc", "NoMultEleStruc"`
- `EmptySet` (`domains/math/sets.cue`) — add `"EmptyStruc"`
- Non-empty instance units (`SetOfNumbers`, `SetOfPrimes`, `SetOfEvens`, `SetOfOdds`, `OSetOfNumbers`, `OSetOfPrimesDesc`) — add `"NonEmptyStruc"`

Each instance unit already lists its parent type first; we prepend the classification tags at the top of its `isA` list to follow EURISKO convention.

**Design call:** `NonEmptyStruc` is tagged on instances, not on `Set`/`List`/etc., because `EmptySet` is a Set and tagging Set with `NonEmptyStruc` would be false. Runtime-created instances won't auto-inherit `NonEmptyStruc` unless the creating heuristic tags them. Acceptable for MVP; revisit if a heuristic starts depending on non-emptiness for runtime-created units.

### 3. Projection operation units (`domains/math/operations.cue`)

Six new units:

| Unit | Category | Domain | Range | defn |
|---|---|---|---|---|
| `Proj1` | UnaryOp | [OPair] | [Anything] | `first` |
| `Proj2` | UnaryOp | [OPair] | [Anything] | `rest first` |
| `FirstEle` | UnaryOp | [OrdStruc] | [Anything] | `first` |
| `LastEle` | UnaryOp | [OrdStruc] | [Anything] | `last` |
| `AllButFirst` | UnaryOp | [OrdStruc] | [OrdStruc] | `rest` |
| `AllButLast` | UnaryOp | [OrdStruc] | [OrdStruc] | `but-last` |

All worth 500, `isA: [UnaryOp, Op, MathOp, Anything]`. Two seeded examples each:

- `Proj1` / `Proj2` — applied to an OPair-shaped `[a, b]` returning `a` / `b`
- `FirstEle` — `[3, 1, 2] → 3`; `[19, 17, 13] → 19`
- `LastEle` — `[3, 1, 2] → 2`; `[19, 17, 13] → 13`
- `AllButFirst` — `[3, 1, 2] → [1, 2]`; `[19, 17, 13] → [17, 13]`
- `AllButLast` — `[3, 1, 2] → [3, 1]`; `[19, 17, 13] → [19, 17]`

### 4. New DSL builtin (`internal/dsl/builtins_math.go`)

One new builtin:

```go
builtins["but-last"] = bButLast

func bButLast(vm *VM) error {
    list := vm.pop().AsList()
    if len(list) == 0 {
        vm.push(ListVal(nil))
        return nil
    }
    vm.push(ListVal(list[:len(list)-1]))
    return nil
}
```

`Proj2`'s `rest first` composition is fine — no dedicated `second` builtin.

### 5. Tests

**DSL unit test** (`internal/dsl/builtins_math_test.go`):
- `TestButLast` — `3 1 2 3 list-of but-last` → `[3, 1]`; empty list → empty list.

**Engine tests** (`internal/engine/engine_test.go`):
- `TestStructureClassificationCategoriesLoad` — six classification categories present in store with correct isA.
- `TestStructureClassificationTagsPropagate` — `store.IsA(OSet, "OrdStruc")`, `IsA(Set, "UnOrdStruc")`, `IsA(OSetOfPrimesDesc, "NonEmptyStruc")`, `IsA(List, "MultEleStruc")` all return true; `IsA(Set, "OrdStruc")` returns false.
- `TestProjectionUnitsLoad` — six projection op units present with correct domain/range/defn.
- `TestFirstEleAppliedToOSetOfPrimesDesc` — engine smoke test; run a few cycles, confirm FirstEle.applics contains an entry for OSetOfPrimesDesc with output 19. (Regression guard against breaking OrdStruc domain dispatch.)

### What this unlocks

- **H-Specialize on classification dimensions.** Existing H6-Specialize reads `restrictedTo`; it can now carry `restrictedTo = "OrdStruc"` to create an op that only runs on ordered structures. That's a structural specialization move that previously wasn't representable.
- **FirstEle / LastEle as discovery tools.** Applying FirstEle to a unit whose `data` is ordered lets H20 cross-apply it to other OrdStruc sibling ops' args. The pipeline discovers what "first element" means across List, OSet, OPair projections automatically.
- **Sets up Phase 5.12 H29.** H29's precondition is that `MultEleStruc` exists as a classification. After this phase it does.
- **Sets up Phase 5.6 D Restrict.** Meta-op Restrict can now plausibly take `OrdStruc` as a restriction; the target category exists.

## Out of scope

- H29 multiplicity mutation — deferred to Phase 5.12.
- `SecondEle`, `ThirdEle`, `AllButSecond`, `AllButThird` — redundant with rest-chain composition; add when a heuristic demands them.
- `SetOfSets`, `StructureOfStructures` higher-order categories — no concrete instances; defer.
- Runtime auto-tagging of new instances with `NonEmptyStruc` — acceptable MVP hole.

## Risk / invariants

- **Multi-parent isA.** `Set.isA` now has five entries (`UnOrdStruc, NoMultEleStruc, Structure, MathObj, Anything`). Multi-parent is already the pattern for ops (`BinaryOp, Op, MathOp, Anything`), and `store.IsA` walks the chain via BFS — no change required. Ordering within isA doesn't affect chain lookups.
- **EmptySet must not tag NonEmptyStruc.** The MVP tags NonEmpty at instance level to avoid this bug.
- **Projection on empty structure.** `FirstEle` on `EmptySet` relies on `bFirst` returning `Nil()`; no guard needed. H24 can notice the always-nil pattern and flag it later.
- **No new engine code required.** All changes live in CUE data + one DSL builtin + tests. Store's existing isA chain, domain resolution in `apply-op`, and H-RunOnExamples pipeline already handle the rest.
