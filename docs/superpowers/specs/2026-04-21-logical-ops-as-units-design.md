# Phase 5.8 — Logical operations as units

**Date:** 2026-04-21
**Phase:** EURISKO parity 5.8

## Goal

Land six logical operations as first-class seed units so existing meta-op heuristics (H-Transpose, H-Compose, H-Restrict) and predicate-analysis heuristics (H27/H28, Rarity hook) organically discover them. Unblocks logical specialization of preds (e.g. `Compose-Not-EvenNum` = OddNum up-to-semantic-dup).

Six units: **And, Or, Not, Implies, TheFirstOf, TheSecondOf**. Plus: the **LogicOp** category, two TruthValue instances (**True, False**) so applic recording has concrete outputs.

## Scope

- **In:** six logical-op units + LogicOp category + True/False instances, each with valid `defn` runnable via `apply-op`, seeded examples referencing True/False. Single new file `domains/math/logic.cue`.
- **Out:** no new heuristics. No domain/range generalization beyond TruthValue for And/Or/Not/Implies (EURISKO uses Anything; TruthValue is tighter and enables type-predicate gates to fire on them). TheFirstOf/TheSecondOf use Anything (genuinely polymorphic projections, per EURISKO).

## Design decisions

### Types + polymorphism

- **And, Or, Not, Implies**: `domain=[TruthValue, TruthValue]` (or `[TruthValue]` for Not), `range=[TruthValue]`. isA includes `BinaryOp`/`UnaryOp`, `LogicOp`, `Op`, `BinaryPred`/`UnaryPred`, `Pred`, `MathConcept`, `Anything`. EURISKO uses Anything; narrowing to TruthValue is a local judgement call to keep type-predicate composition (is-truthvalue? + And) coherent.
- **TheFirstOf, TheSecondOf**: `domain=[Anything, Anything]`, `range=[Anything]`. Genuinely polymorphic — mirrors EURISKO.

### Defns (DSL bytecode)

- `And`: `and` (single builtin)
- `Or`: `or`
- `Not`: `not`
- `Implies`: `swap not swap or` (evaluates to `(!x) || y`; stack enters as `x y`, want `x y → (!x) or y`; concretely: `swap` → `y x`; `not` → `y !x`; `swap` → `!x y`; `or` → `(!x) or y`)
- `TheFirstOf`: `swap drop` (stack `x y → x`)
- `TheSecondOf`: `drop` (stack `x y → y`)

### True / False instances

```
True  — isA=[TruthValue, Anything], data=true
False — isA=[TruthValue, Anything], data=false
```

Needed so `apply-op` on these ops can reference concrete output unit names (existing `apply-op` lookups resolve outputs by data-matching against TruthValue examples).

### LogicOp category

```
LogicOp — isA=[Category, MathConcept, MathObj, Anything], worth=500
```

No `specializations` slot (consistent with MetaOpHeuristic, OrdStruc, etc. — instance-level tagging through the logical-op units' own isA chains).

### EURISKO iSA chain (And ⊆ TheFirst/TheSecond ⊆ Or)

EURISKO declares `And.Generalizations = (TheSecondOf TheFirstOf OR)` because logically `A∧B ⇒ A`, `A∧B ⇒ B`, `A∧B ⇒ A∨B`. This is wiring for discovery heuristics, not a type-theoretic claim.

We mirror this: `And.generalizations = [TheFirstOf, TheSecondOf, Or]`. Store inverse-maintenance will populate the matching `specializations` slots automatically.

## Non-goals

- No bitwise/numeric overloads (NOT of number → boolean-of-zeroness). TruthValue-only semantics.
- No Xor, Nand, Nor, Iff — left to meta-op discovery via Compose/Not.
- No Rarity updates on logical-op applics — logical ops aren't Preds in the H24 sense (though their isA includes Pred; we don't want every And-call inflating Rarity on And itself). If the existing Rarity hook inadvertently fires because isA contains Pred, revisit in a follow-up.

## Testing

- `TestLogicalOpUnitsPresent` — all six ops + LogicOp + True/False loaded from CUE with expected isA.
- `TestAndRunsOnTruthValues` — `apply-op True False And` → returns the False unit name (or equivalent verifier).
- `TestNotRunsOnTruthValue` — `apply-op True Not` → False.
- `TestLogicOpExamplesCategory` — `store.IsA("And", "LogicOp")` + same for each logical op.
- `TestTheFirstOfReturnsFirst` / `TestTheSecondOfReturnsSecond` — polymorphic projection on two arbitrary units.

## Expected downstream effects

After load, existing heuristics should start firing on the new ops:
- **H-Transpose** on And/Or → creates Transpose-And / Transpose-Or. Commutativity sampling likely kills both as semantic dups (And/Or are commutative).
- **H-Transpose** on Implies → creates Transpose-Implies (non-commutative, survives).
- **H-Compose** on pairs where ranges/domains match → creates Compose-Not-And (= Nand), Compose-Not-Or (= Nor), etc.
- **H-Restrict** — harder to apply since TruthValue has no specializations; likely no-op.

These are observations, not success criteria for this slice.
