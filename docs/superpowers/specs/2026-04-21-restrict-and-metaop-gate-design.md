# Phase 5.6 D — Restrict + MetaOpHeuristic gate refactor

**Date:** 2026-04-21
**Phase:** EURISKO parity 5.6 D (partial — Restrict only; InvertOp deferred)
**Companion docs:** [eurisko-parity-phases.md](../../eurisko-parity-phases.md), [2026-04-19-semantic-duplicate-ops-design.md](2026-04-19-semantic-duplicate-ops-design.md), [2026-04-19-transpose-and-compose-design.md](2026-04-19-transpose-and-compose-design.md)

## Goal

Ship the third meta-operation (`Restrict`) alongside a small refactor that turns the H-SemanticDup creditor gate from a hard-coded allowlist into a proper `MetaOpHeuristic` isA category. InvertOp is explicitly out of scope for this slice.

## Why now

Three motivations converge:

1. **EURISKO parity completeness.** Transpose and Compose shipped 2026-04-19; Restrict rounds out the meta-op triad with the narrowing counterpart. In EURISKO, Restrict has worth 600 (higher than Transpose/Compose at 500) — the narrowed-domain perspective is load-bearing for discoveries like "Add restricted to primes → PrimeAddition."
2. **Unblocks the H-SemanticDup gate followup.** The current `creditors contains H-Transpose or H-Compose` allowlist in `domains/common/heuristics.cue:1140-1142` is a known wart (see `docs/eurisko-parity-phases.md` Phase 5.6 followup). Adding a third meta-op without refactoring the gate would entrench the wart; doing both together amortizes the cost.
3. **Exercises the Phase 5.4 classification machinery.** `MetaOpHeuristic` as an isA category follows the same pattern as OrdStruc/UnOrdStruc/MultEleStruc — queryable via `store.IsA`, extensible, no per-output slot marker churn.

## Non-goals

- **InvertOp.** EURISKO worth is 100, our phase doc flags it "genuinely complex" (requires inverse-search infra). Deferred to a future slice.
- **Restrict-chain depth limits.** Restrict-of-Restrict is allowed; the one-shot `restrictRan` flag is per-unit, not per-lineage. If chain explosion is observed in smoke runs, add a cap later.
- **Replacing H6-Specialize.** H6 narrows via `restrictedTo` (applics-level filter); H-Restrict creates a first-class narrowed Op unit. Both coexist — they target different outcomes.

## Design decisions

### Restrict firing gate

H-Restrict fires `ifPotentiallyRelevant` on a unit `f` when **all** of:

- `f isA Op`
- `f.applics` length ≥ 1 (Option B from brainstorming — requires recorded behavior to narrow against)
- ∃ position `i` in `f.domain` where the type `f.domain[i]` has ≥ 1 immediate specialization in the store
- `f.restrictRan != true` (one-shot per unit)

Rationale: H-Restrict's value is "re-examine `f` on a narrower slice." With zero recorded applics there's nothing to re-examine; the new unit just burns agenda slots. Restricting to *immediate* specializations (not transitive descendants) avoids walking arbitrarily deep isA chains on abstract domain types like `Anything` or `MathConcept`.

### `restrict-op` builtin behavior

Stack: `( opName -- newName | "" )`.

1. Resolve `f = Store.Get(opName)`. If nil or not `isA Op`, push `""`, return.
2. Collect candidate positions: indices `i` where `Store.Specializations(f.domain[i])` is non-empty.
3. If no candidates, push `""`, return.
4. Random-choose position `i`; random-choose specialization `s` of `f.domain[i]`.
5. Construct `newDomain = f.domain` with `[i] := s`.
6. Gensym `newName = "Restrict-<opName>-<N>"`.
7. Create unit with slots:
   - `isA`: copy of `f.isA` (matching Transpose/Compose convention — the gate lives on the *heuristic*, not the output unit, so no extra marker on the Op itself)
   - `domain`: `newDomain`
   - `range`: copy of `f.range`
   - `arity`: `f.arity`
   - `defn`: delegating bytecode that calls parent via `apply-op-args` (exact form matches `transpose-op`'s delegation pattern)
   - `worth`: average of `f.worth` and the static H-Restrict worth (600), matching EURISKO's `AverageWorths` convention
   - `creditors`: `["H-Restrict"]`
   - `generalizations`: `[opName]`
   - `extensions`: `[opName]` (EURISKO convention from `EURUNITS:848`)
8. Store the new unit. Set `f.restrictRan = true`.
9. Push `newName`.

### `MetaOpHeuristic` category

New category unit (location: `domains/common/heuristics.cue` alongside existing category units, or `domains/common/categories.cue` if that's where classification markers live — implementation task picks based on file state).

```
name: "MetaOpHeuristic"
isA:  ["Category", "Anything"]
english: "Heuristic category: produces new Op units via meta-operation (Transpose/Compose/Restrict/...)."
```

No `specializations` slot (following the instance-level-tagging discipline from Phase 5.4 — category gets populated via `isA` declarations on the heuristics themselves, not inverse wiring).

H-Transpose, H-Compose, H-Restrict each gain `MetaOpHeuristic` in their `isA` lists.

### H-SemanticDup gate refactor

`ifPotentiallyRelevant` in `domains/common/heuristics.cue:1136-1152` replaces:

```
"ArgU" @ "creditors" get-slot "H-Transpose" list-contains
"ArgU" @ "creditors" get-slot "H-Compose"   list-contains
or
```

with a creditors-walk gated on `isA MetaOpHeuristic`. Candidate DSL form (exact idiom during implementation):

```
"ArgU" @ "creditors" get-slot
any (it "MetaOpHeuristic" isa?)
```

If `any` / `isa?` helpers are missing, implementation task either adds them or loop-unrolls with `each`.

### `applics-redundant?` domain precheck

New guard, first check after unit/parent nil checks in `internal/dsl/builtins_math.go` `bApplicsRedundant`:

- Read `u.domain` and `parent.domain` as `[]string`.
- Compare as **multisets** (sort both, then element-equal).
- If unequal, push `false` and return — the unit and its parent operate on structurally different domains, so behavioral match on sampled applics does not imply redundancy.

Critical: **multiset**, not ordered list. Transpose swaps domain order (`[A,B]` → `[B,A]`); those must compare equal so commutative-Transpose killing (the 2026-04-19 behavior) survives. Restrict narrows one position (`[Number, Number]` → `[Integer, Number]`); those compare unequal, so Restrict is auto-exempt.

## Architecture

### Files touched

- **New Go code**
  - `internal/dsl/builtins_math.go` — add `bRestrictOp`, register `restrict-op`. Add multiset domain-equality precheck to `bApplicsRedundant`.
- **New CUE content**
  - `domains/common/heuristics.cue` (or `categories.cue`) — add `MetaOpHeuristic` category unit.
  - `domains/common/heuristics.cue` — add `H-Restrict` heuristic; append `MetaOpHeuristic` to the `isA` lists of H-Transpose, H-Compose, H-Restrict; swap H-SemanticDup's gate.
- **New tests**
  - `internal/engine/engine_test.go` — seven new tests (enumerated in Testing).
- **Doc updates (in the phase close-out commit, not this spec)**
  - `docs/eurisko-parity-phases.md` — mark 5.6 D partial-complete, note InvertOp deferred.

### Component boundaries

`restrict-op` is a pure Go DSL builtin with no engine-package dependencies. It reuses the `subExecute` + `vm.Store` + `vm.Rng` machinery already used by `transpose-op` and `compose-ops`. It does **not** call into `internal/engine/*` directly.

`H-Restrict` is a CUE heuristic that runs entirely inside the DSL VM via `restrict-op`. No engine-package changes.

The gate refactor is a localized edit: one `ifPotentiallyRelevant` body, one new `bApplicsRedundant` precheck, one category unit. No cross-file coupling beyond existing Store lookups.

## Data flow (summary)

```
Task on Op f (any slot)
  → H-Restrict.ifPotentiallyRelevant passes (see gate above)
  → thenCompute: "f" restrict-op
      → picks position i + specialization s
      → creates "Restrict-f-N" with narrowed domain, delegating defn
      → sets f.restrictRan = true

Later: task on Restrict-f-N accumulates ≥3 applics
  → H-SemanticDup.ifPotentiallyRelevant passes:
       creditors walk finds H-Restrict isA MetaOpHeuristic
       generalizations, applics checks pass
  → thenCompute walks generalizations:
       "Restrict-f-N" "f" applics-redundant?
         → domain multiset differs (narrowed) → returns false
       → Restrict-f-N survives.

Separately: task on Transpose-Add accumulates applics, Add is commutative
  → applics-redundant? sees domain multisets [Number,Number] == [Number,Number] → continues to output-matching → all match → returns true → Transpose-Add killed (regression preserved).
```

## Testing

All tests live in `internal/engine/engine_test.go`.

1. **`TestRestrictOpCreatesNarrowedUnit`** — unit-level on the builtin. Seed Add (Number×Number→Number) + Integer as specialization of Number. Call `"Add" restrict-op`. Assert the pushed name exists, has generalizations `[Add]`, creditor `H-Restrict`, domain is either `[Integer, Number]` or `[Number, Integer]`.
2. **`TestHRestrictFiresOnEligibleOp`** — engine-loop smoke. Seed Add + ≥1 Add applic + Integer specialization. Tick engine. Assert `Restrict-Add-*` unit exists and `Add.restrictRan == true`. Tick again. Assert exactly one Restrict-Add unit (one-shot behavior).
3. **`TestHRestrictSkipsOpWithoutApplics`** — seed Add with zero applics. Tick. Assert no Restrict-Add unit created.
4. **`TestHRestrictSkipsOpWithoutSpecializableDomain`** — seed op whose every domain type has no specializations. Tick. Assert no Restrict unit created.
5. **`TestSemanticDupExemptsRestrict`** — construct a Restrict-Add unit directly (narrower domain, delegating defn, ≥3 applics whose outputs match parent Add). Tick H-SemanticDup. Assert Restrict-Add survives (`applics-redundant?` returns false due to domain mismatch).
6. **`TestSemanticDupStillKillsCommutativeTranspose`** — regression. Seed commutative Add + Transpose-Add (reversed domain `[Number,Number]`, which multiset-equals parent's). Tick. Assert Transpose-Add killed.
7. **`TestMetaOpHeuristicCategoryWalk`** — `store.IsA("H-Restrict", "MetaOpHeuristic")` returns true; same for `H-Transpose` and `H-Compose`.

## Known risks / open questions

- **Worth averaging.** Using `average(f.worth, 600)` matches EURISKO's `AverageWorths` convention but may be worth tuning if Restrict outputs dominate the agenda. First-run observable; tune post-smoke.
- **Set-equal domain compare on non-Op units.** `applics-redundant?` reads `domain` as `[]string`. If a future unit type has `domain` in a different shape, the precheck should degrade to "domains unequal → false" defensively — aligning with the existing defensive defaults in `bApplicsRedundant`. Implementation note, not a blocker.
- **Immediate vs transitive specializations.** Restrict picks only immediate specializations. If smoke runs show the narrowing is too shallow, the policy can widen in a followup without spec changes.

## Followup items (out of scope for this slice)

- Phase 5.6 D' — **InvertOp** with a real inverse-search algorithm. Defer until at least one concrete demand emerges (e.g., a discovery heuristic that would benefit from Subtract-from-Add).
- If new meta-op heuristics land later (H-Curry, H-Distribute, H-Coalesce), they just need `isA: [..., MetaOpHeuristic]` — the gate refactor already accommodates them with zero further edits.
