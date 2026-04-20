# Phase 5.1 — OSet Type and Operations

**Date:** 2026-04-20
**Scope:** EURISKO parity phase 5.1 (`docs/eurisko-parity-phases.md`).

## Goal

Add an Ordered Set (`OSet`) type to the math domain, with operations that preserve insertion order and reject duplicates. OSet becomes a behavioral sibling of Set that is distinguishable from Set *only by its treatment of order*. That distinction is what gives H19, H-SemanticDup, and H-Transpose meaningful new material to reason about.

## Why this is the right shape

Nous stores all structured data as Go lists. The Set/List/Bag/OSet distinction is carried entirely by `isA` and by which DSL builtins the op's `defn` calls — not by storage. Existing set-* builtins dedupe but `make-set` also sorts, so Set is effectively canonicalized to sorted form. OSet therefore needs:

- Its own order-preserving builtins (not reusing `set-union` et al., which call `make-set` and would destroy order).
- Instance data whose order is *not* sorted, so order-preservation is observable.
- Subtype relationship to Set (EURISKO-faithful) — OSet instances flow through Set ops, losing order; that's the productive intermingling H-SemanticDup is designed to discover and reason about.

## Components

### 1. Type unit (`domains/math/types.cue`)

```cue
{
    name: "OSet"
    worth: 600
    isA: ["Set", "Structure", "MathObj", "Anything"]
    english: "An ordered collection with no duplicate elements"
    specializations: ["OSetOfNumbers", "OSetOfPrimesDesc"]
    defn: "is-list?"
}
```

Also append `"OSet"` to `Set.specializations`.

### 2. Instance units (`domains/math/sets.cue`)

```cue
{
    name: "OSetOfNumbers"
    worth: 500
    isA: ["OSet", "Set", "Structure", "MathObj", "Anything"]
    english: "The integers from 1 to 20 in ascending order"
    data: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
}
{
    name: "OSetOfPrimesDesc"
    worth: 500
    isA: ["OSet", "Set", "Structure", "MathObj", "Anything"]
    english: "Primes under 20 in descending order"
    data: [19, 17, 13, 11, 7, 5, 3, 2]
}
```

The descending order on `OSetOfPrimesDesc` is load-bearing: it is what makes order-preservation visible in the seed data.

### 3. Operation units (`domains/math/operations.cue`)

Five new units, worth 500 each, mirroring the shape of `SetUnion` etc:

| Unit | Category | Domain | Range | defn |
|---|---|---|---|---|
| `OSetUnion` | BinaryOp | [OSet, OSet] | [OSet] | `oset-union` |
| `OSetIntersect` | BinaryOp | [OSet, OSet] | [OSet] | `oset-intersect` |
| `OSetInsert` | BinaryOp | [OSet, Anything] | [OSet] | `oset-insert` |
| `OSetDelete` | BinaryOp | [OSet, Anything] | [OSet] | `oset-delete` |
| `OSetEqual` | BinaryPred | [OSet, OSet] | [TruthValue] | `oset-equal?` |

Each gets 2 seeded examples demonstrating order preservation.

### 4. DSL builtins (`internal/dsl/builtins_math.go`)

Five new builtins, one per op unit:

- `oset-union ( a b -- c )` — `a` then elements of `b` not already in `a`, in `b`'s order.
- `oset-intersect ( a b -- c )` — elements of `a` that also appear in `b`, preserving `a`'s order.
- `oset-insert ( oset x -- oset' )` — append `x` if not present; no-op if already present (no reordering).
- `oset-delete ( oset x -- oset' )` — remove `x` if present, preserving order.
- `oset-equal? ( a b -- bool )` — true iff same length and element-wise equal (strict order-sensitive).

All implementations are linear-scan O(n·m), consistent with existing `set-*` builtins. No `make-oset` is needed — `oset-union a []` already functions as a dedupe-preserving-order helper. OSetDifference is intentionally omitted to match the EURISKO 5.1 list; can be added later if a heuristic needs it.

### 5. Tests

**DSL unit tests (`internal/dsl/builtins_math_test.go`):**
- `oset-union` preserves left order and appends right novelty: `[3 1 2] [4 2 5] oset-union → [3 1 2 4 5]`.
- `oset-insert` is a no-op on existing element.
- `oset-delete` preserves surrounding order.
- Divergence test: `[1 2] [2 1] oset-equal?` is false; `[1 2] [2 1] set-equal?` is true.

**Engine smoke test:** one test that loads the math domain, runs a handful of cycles, and asserts OSetUnion's recorded applics contain an order-preserved output (e.g. applied to OSetOfPrimesDesc × OSetOfNumbers, output starts with 19, 17, 13…).

## What this unlocks

- **H-SemanticDup correctness case:** OSetUnion is *not* a duplicate of SetUnion — `([2,1],[2,1]) → [2,1]` vs `→ [1,2]`. The heuristic currently cannot observe this distinction because no order-disagreeing op exists. This phase gives it the test case it was built for.
- **H-Transpose:** applied to OSetUnion will yield a genuinely distinct `Transpose-OSetUnion` (order-preserving ops are not commutative on output order even when commutative on membership). Applied to SetUnion, the new commutativity sampling correctly prunes it.
- **Specialization pipeline:** new type gives H-Specialize / H6 fresh material; OSet instances flow through Set ops, producing observable Set→OSet demotions that heuristics can notice.

## Out of scope

- No hand-wired H-Specialize paths; pipeline discovers specializations organically.
- No `ReverseOSet` op — natural mutation target, left for H-Transpose / future H-Reverse to discover.
- No interaction with Phase 5.3 (projections) or 5.4 (structure classification); that wiring happens when those phases land.

## Risk / invariants

- **Subtype intermingling:** OSet `isA ["Set", ...]` means Set-domain ops accept OSet inputs, producing Set outputs (sorted, order lost). This is intended; see "Why this is the right shape" above. No guard needed.
- **H19 should not collapse OSet instances:** duplicate detection compares criterial slots (`data`). OSetOfNumbers and SetOfNumbers have the same `data` but differ in `isA`, so H19 (which scopes duplicate search to a shared isA category) will correctly see them as siblings only when both belong to a common category (Set, Structure). If H19 flags them, that is itself an interesting discovery, not a bug — defer any gating until observed in a run.
- **Existing set-* builtins are untouched.** No risk to existing tests.
