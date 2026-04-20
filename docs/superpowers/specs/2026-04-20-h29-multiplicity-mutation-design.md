# Phase 5.12 — H29 Multiplicity Mutation

**Date:** 2026-04-20
**Scope:** EURISKO parity phase 5.12 (`docs/eurisko-parity-phases.md`).

## Goal

Add H29, the heuristic that generates new examples of a multi-element structure by randomly mutating the element multiplicities of existing examples. H29 is the first heuristic to dispatch on the instance-level `MultEleStruc` classification added in Phase 5.4 — it validates that the classification machinery actually drives behavior.

## Why this is the right shape

EURISKO's H29 fires when focus lands on an `examples` task for a MultEleStruc. It iterates existing example children; for each element in a child's data, it randomly drops, keeps, or duplicates. The resulting mutated list becomes a new example child. This gives the system a source of novel multiset shapes without requiring external data.

Today nous has no MultEleStruc instance with populated examples: `SortedList` is the only seeded MultEleStruc and it has no data. H29 would sit dormant forever. We therefore seed a canonical multiset (`BagOfTallies`) with three example children carrying real multi-element data. The mutation mechanism is a single new DSL builtin (`mutate-multiplicities`); H29 itself is a standard heuristic unit following the H25/H26/H27 pattern.

## Components

### 1. Seed units (`domains/math/sets.cue`)

Append four new units inside `units: [...]`:

```cue
{
    name:    "BagOfTallies"
    worth:   500
    isA: ["MultEleStruc", "UnOrdStruc", "NonEmptyStruc", "Bag", "Structure", "MathObj", "Anything"]
    english: "A bag with duplicate elements — a canonical multiset for mutation experiments"
    data: [1, 1, 2, 2, 2, 3, 5]
    examples: ["Bag-ex-tally-a", "Bag-ex-tally-b", "Bag-ex-tally-c"]
}
{
    name:    "Bag-ex-tally-a"
    worth:   300
    isA: ["Bag", "Structure", "MathObj", "Anything"]
    data: [1, 1, 2, 3]
}
{
    name:    "Bag-ex-tally-b"
    worth:   300
    isA: ["Bag", "Structure", "MathObj", "Anything"]
    data: [2, 2, 4, 5, 5]
}
{
    name:    "Bag-ex-tally-c"
    worth:   300
    isA: ["Bag", "Structure", "MathObj", "Anything"]
    data: [3, 3, 3, 7]
}
```

`BagOfTallies.isA` tags the classification categories at instance level, consistent with the Phase 5.4 decision. The three child units carry real multi-element data with duplicates; H29 mutates each child's data to produce new sibling children.

### 2. DSL builtin (`internal/dsl/builtins_math.go`)

One new builtin registered in `init()`:

```go
builtins["mutate-multiplicities"] = bMutateMult
```

Implementation:

```go
// mutate-multiplicities ( list -- list' )
// For each element: with equal probability drop, keep, or keep+duplicate.
// Uses VM.Rng for reproducibility when callers seed the engine's RNG.
func bMutateMult(vm *VM) error {
    in := vm.pop().AsList()
    out := make([]Value, 0, len(in)+2)
    for _, el := range in {
        switch vm.Rng.Intn(3) {
        case 0:
            // drop
        case 1:
            out = append(out, el)
        case 2:
            out = append(out, el, el)
        }
    }
    vm.push(ListVal(out))
    return nil
}
```

Probabilities are hardcoded at 1/3 each, matching EURISKO's `Randomp` trichotomy. No tuning slot; if future work needs adjustable rates, make it a heuristic slot read by H29 rather than a per-call argument.

### 3. H29 heuristic (`domains/common/heuristics.cue`)

Append inside the `units: [...]` list:

```cue
{
    name:    "H29"
    worth:   500
    isA: ["Heuristic", "Anything"]
    english: "New examples of a MultEleStruc can be found by randomly mutating element multiplicities in known examples"
    overallRecord: {successes: 0, failures: 0}
    h29Cap: 5
    ifWorkingOnTask: #"""
        "CurUnit" @ "MultEleStruc" isa?
        "CurSlot" @ "examples" =
        and
        "CurUnit" @ "h29Ran" get-slot nil =
        and
        """#
    thenCompute: #"""
        "CurUnit" @ true "h29Ran" set-slot

        "CurUnit" @ "examples" get-slot "srcExs" !
        "H29" "h29Cap" get-slot "cap" !
        0 "made" !

        "srcExs" @ each
            "made" @ "cap" @ < if
                it "srcName" !
                "srcName" @ "data" get-slot mutate-multiplicities "newData" !
                "newData" @ list-length 0 > if
                    "Bag-ex-H29-" "CurUnit" @ concat "-" concat "made" @ concat "newName" !
                    "newName" @ unit-exists? not if
                        "newName" @ "Bag" create-unit drop
                        "newData" @ "newName" @ "data" set-slot
                        "newName" @ "CurUnit" @ "examples" add-to-slot
                        "H29" "newName" @ "creditors" set-slot
                        "made" @ 1 + "made" !
                    then
                then
            then
        end

        "made" @ 0 > if
            "H29: created " "made" @ concat " new multiplicity-mutated examples for " concat "CurUnit" @ concat print
        then
        """#
}
```

Gate: CurUnit isA MultEleStruc AND CurSlot == "examples" AND `h29Ran` not yet set. One-shot per source unit via the flag (matches the pattern used by `specTaskAdded`, `genTaskAdded`, `h8Ran`).

Bound: `h29Cap` (default 5) is a slot on H29 so it can be tuned by later mutation without editing CUE.

Output: each surviving mutation becomes a fresh `Bag-ex-H29-<CurUnit>-<N>` unit, added to CurUnit's `examples` via the inverse-maintained `add-to-slot` builtin so `isAExamples` (if relevant) wires up automatically. Creditor set so H-SemanticDup and analytical heuristics know who made the unit.

### 4. Tests

**DSL unit test** (`internal/dsl/builtins_math_test.go`):

`TestMutateMultiplicities` seeds `vm.Rng` with a fixed seed and:
- asserts the function is deterministic with the seed (same input → same output across multiple calls after re-seeding);
- asserts every element in the output was present in the input (no foreign values introduced);
- asserts an empty input produces an empty output;
- runs the mutator 100× on `[1, 2, 3]` and confirms at least one output longer than 3 (duplication occurred) and at least one output shorter than 3 (a drop occurred). Statistical regression guard.

**Engine tests** (`internal/engine/engine_test.go`):

- `TestH29FiresOnBagOfTallies`: run the engine long enough for an `examples` task on `BagOfTallies` to fire H29. After the run, `BagOfTallies.examples` has more than three children, and each added child's `data` slot is a non-empty list drawn from the source tally alphabet (`{1, 2, 3, 4, 5, 7}` across the three children).
- `TestH29OneShotGuard`: directly invoke the H29 firing path on the same CurUnit twice; second pass adds no new examples because `h29Ran` is set.

### What this unlocks

- **First live exercise of Phase 5.4 classification dispatch.** H29 gates on `IsA(CurUnit, "MultEleStruc")` — a real behavioral test for the instance-level tagging decision.
- **Material for H-SemanticDup and H19.** Mutation collisions and semantic duplicates get pruned by the existing post-hoc machinery.
- **Material for H22 / H23 / H24.** More examples on a non-Set MultEleStruc enlarges the pool H24 can search for rare predicates.
- **Creditor chain for meta-op analysis.** `creditors: [H29]` on mutated children means later heuristics can trace provenance.

## Out of scope

- `MultEleStrucInsert` as a first-class op unit. EURISKO represents the mutator as a named op; we inline the builtin in H29 to keep the surface area small. If a heuristic later wants to reason about the mutator as a unit, add it then.
- Tunable mutation probabilities via slot. Hardcoded 1/3 each is enough for MVP.
- An H29-Seeder that schedules examples tasks on idle MultEleStruc units (H24-Seeder pattern). Defer; add if H29 stays dormant in practice.
- Additional MultEleStruc seed units beyond BagOfTallies. One seed with three children is enough to verify the mechanism.

## Risk / invariants

- **Empty mutations.** When the RNG drops every element the mutator returns `[]`. H29 guards with `"newData" @ list-length 0 >` and skips; no empty-data Bag children get created.
- **Name collisions.** `unit-exists?` guards name reuse under repeated firings (though `h29Ran` makes that unreachable in normal operation).
- **Randomness reproducibility.** The mutator uses `vm.Rng`; tests seed the engine's RNG deterministically. Production runs inherit whatever seed the engine was configured with.
- **Double inverse wiring.** `add-to-slot` walks `Examples`/`isAExamples` inverses. `Bag-ex-H29-*` children will acquire `isAExamples = [BagOfTallies]` automatically — matches pattern of H27/H28.
- **Interaction with H19/H-SemanticDup.** Mutations whose data matches an existing sibling (e.g., the RNG happens to reproduce `Bag-ex-tally-a`'s data) land as a separate unit with the same `data`. H19-Criterial compares data and may kill them. That is the intended behavior.
