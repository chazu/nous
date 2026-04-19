# Phase 5.9 + 5.11: Numeric Ops and Comparison Predicates as Units

**Date:** 2026-04-19
**Status:** Design approved, ready for implementation plan
**Plan ref:** `docs/eurisko-parity-phases.md` §5.9 and §5.11

## Motivation

The math domain currently exposes arithmetic only through DSL builtins (`+`, `*`, `=`, `>`, etc.). Heuristics cannot reason about these operations because they are not first-class units. Phase 5.9 lifts four numeric ops into the unit store; Phase 5.11 does the same for four numeric comparison predicates. Together they widen the discovery surface of the math domain without touching engine code.

Per memory `project_nous_phases.md`, discovery density is currently bottlenecked by a thin 4-predicate set. Adding 4 numeric comparison predicates roughly doubles the predicate population and gives H24/H27/H28 materially more to work with on `Number`-valued categories.

## Scope

**In scope:**
- 4 numeric op units: `Add`, `Multiply`, `Successor`, `Square`
- 4 numeric comparison pred units: `IEQP`, `IGEQ`, `IGREATERP`, `ILESSP`
- Seeded `examples` on each op unit (raw-literal pattern, matching `GCD`/`DivisorsOf`)
- Two Go tests covering existence, invocation, and Rarity population

**Out of scope (deferred per plan doc):**
- `Transpose` variants of comparison preds — land with Phase 5.6A
- New heuristics — existing H-RunOnExamples / H24 / H25 / H26 / H27 / H28 pick these up automatically
- New DSL builtins — all required primitives (`+`, `*`, `=`, `>=`, `>`, `<`, `dup`) already exist
- Engine changes

## Design

### 5.9 — Numeric ops

Add to `domains/math/operations.cue`:

```cue
{
    name:    "Add"
    worth:   500
    isA: ["BinaryOp", "Op", "MathOp", "Anything"]
    domain:  ["Number", "Number"]
    range:   ["Number"]
    english: "Sum of two numbers"
    defn:    #"""
        +
        """#
    examples: [
        {args: 2, args2: 3, result: 5},
        {args: 0, args2: 7, result: 7},
    ]
},
{
    name:    "Multiply"
    worth:   500
    isA: ["BinaryOp", "Op", "MathOp", "Anything"]
    domain:  ["Number", "Number"]
    range:   ["Number"]
    english: "Product of two numbers"
    defn:    #"""
        *
        """#
    examples: [
        {args: 3, args2: 4, result: 12},
        {args: 1, args2: 9, result: 9},
    ]
},
{
    name:    "Successor"
    worth:   500
    isA: ["UnaryOp", "Op", "MathOp", "Anything"]
    domain:  ["Number"]
    range:   ["Number"]
    english: "Next integer after n"
    defn:    #"""
        1 +
        """#
    examples: [
        {args: 0, result: 1},
        {args: 7, result: 8},
    ]
},
{
    name:    "Square"
    worth:   500
    isA: ["UnaryOp", "Op", "MathOp", "Anything"]
    domain:  ["Number"]
    range:   ["Number"]
    english: "n times n"
    defn:    #"""
        dup *
        """#
    examples: [
        {args: 3, result: 9},
        {args: 5, result: 25},
    ]
},
```

### 5.11 — Numeric comparison predicates

Append to `domains/math/predicates.cue`:

```cue
{
    name:   "IEQP"
    worth:  500
    isA: ["BinaryPred", "Pred", "MathPred", "Anything"]
    domain: ["Number", "Number"]
    range: ["TruthValue"]
    defn:   #"""
        =
        """#
},
{
    name:   "IGEQ"
    worth:  500
    isA: ["BinaryPred", "Pred", "MathPred", "Anything"]
    domain: ["Number", "Number"]
    range: ["TruthValue"]
    defn:   #"""
        >=
        """#
},
{
    name:   "IGREATERP"
    worth:  500
    isA: ["BinaryPred", "Pred", "MathPred", "Anything"]
    domain: ["Number", "Number"]
    range: ["TruthValue"]
    defn:   #"""
        >
        """#
},
{
    name:   "ILESSP"
    worth:  500
    isA: ["BinaryPred", "Pred", "MathPred", "Anything"]
    domain: ["Number", "Number"]
    range: ["TruthValue"]
    defn:   #"""
        <
        """#
},
```

## Rationale

**Why raw-literal examples, not generator-produced instance references?**
All existing ops (`GCD`, `DivisorsOf`, `SetUnion`, `SetIntersect`, `SetDifference`) seed `examples` with raw literals. `Number.examples` also uses raw integers. H-Generate's output (instance units like `Number-gen-0`) is additively appended to `Number.examples` — it does not replace the raw-literal convention and H-RunOnExamples handles both. Following precedent keeps the change minimal and coherent.

**Why worth 500?**
Matches the GCD/DivisorsOf/SetDifference tier. These numeric ops are no more or less central than GCD; bumping higher would require justification we do not have.

**Why no Transpose?**
EURISKO pairs IGEQ with IGREATERP and ILESSP via Transpose. Our Phase 5.6A (Transpose meta-op) is deferred. Seeding the preds without Transpose is coherent because (a) each has an independent, non-redundant defn and (b) when 5.6A lands it can generate Transpose-IGEQ etc. as a natural test case.

## Tests

Both in `internal/engine/` (colocated with the existing domain-integration tests):

### `TestNumericOpsAsUnits`
1. Load `domains/math` via the CUE loader.
2. Assert `Add`, `Multiply`, `Successor`, `Square` exist in the store with expected `isA` entries.
3. Seed `Number.examples = [2, 4, 6, 8, 10]`.
4. Invoke H-RunOnExamples focused on `Successor`.
5. Assert `Successor.applics` contains ≥1 entry with `output` a valid Number-typed result and `args` drawn from the examples.

### `TestNumericComparisonPreds`
1. Load `domains/math`.
2. Assert `IEQP`, `IGEQ`, `IGREATERP`, `ILESSP` exist with `isA` including `BinaryPred` and `Pred`.
3. Call `apply-pred IGREATERP [5, 3]` via the DSL VM → true.
4. Call `apply-pred ILESSP [5, 3]` via the DSL VM → false.
5. Call `apply-pred IEQP [3, 3]` → true.
6. Assert `IGREATERP.rarity` is populated (format `[freqTrue, numT, numF]`) and reflects the three calls.

## Expected runtime effects (observational, not asserted)

After a 300-cycle math-domain run we expect:
- H25/H26 to have a chance of firing on `IGREATERP`/`ILESSP` (binary preds with ≥4 `Number.examples` on each domain position)
- H24 to potentially flag `ILESSP` as rare on randomly-paired inputs
- H8 to potentially propagate applics between `Add` and `Multiply` specializations

None of these are asserted by the tests — they are discovery-density signals to watch in the next run summary.

## Risk

Extremely low. Changes are additive: new CUE units only, no code changes. Existing tests should pass unchanged. Worst-case failure mode is a domain-load error from a CUE typo — caught on first test run.

## Out-of-scope follow-ups (for future phases)

- Phase 5.6A Transpose — will generate Transpose-IGEQ, Transpose-IGREATERP as variants
- Phase 5.7 choice ops — can use these preds as filter arguments
- Numeric-range predicate synthesis (e.g. `IsPositive`) — candidate for learned predicates downstream
