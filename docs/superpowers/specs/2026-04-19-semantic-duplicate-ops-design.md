# Semantic Duplicate Operation Detection

**Date:** 2026-04-19
**Status:** Design approved, ready for implementation plan
**Follows from:** Phase 5.6 A+B merged 2026-04-19; smoke run observed that H19 doesn't prune `Transpose-Add` / `Transpose-Multiply` / `Transpose-SetUnion` etc.

## Motivation

After Phase 5.6 A (Transpose) and B (Compose) shipped, a 100-cycle math-domain smoke run created 12 Transpose-* units including variants of commutative ops (Add, Multiply, SetUnion, SetIntersect). None were pruned by H19. Systematic investigation found three independent reasons:

1. **H19-EliminateDuplicates** gates on `data != nil`; operations have no `data` slot — gate always fails.
2. **H19Criterial** only iterates `new-units`, which is populated exclusively during task-phase (`WorkOnTask`) firing. H-Transpose creates units in unit-focus — they never enter `new-units`.
3. **H19Criterial's** comparison treats `Defn` as a criterial slot and compares by string equality; `"swap +"` ≠ `"+"` syntactically, even though they compute the same function when `+` is commutative.

Commutativity is a semantic property. The codebase currently has no mechanism that recognizes it. This spec adds two complementary mechanisms: one preventive (at Transpose creation) and one detective (post-hoc, for any Op).

## Scope

**In scope:**
- Modify `bTransposeOp` (Go) to sample-test commutativity before creating the Transpose variant when `domain[0] == domain[1]`.
- New Go DSL builtin `applics-redundant? (unitName parentName -- bool)`.
- New CUE heuristic `H-SemanticDup` using that builtin to kill ops whose observed applics are fully redundant against any generalization.
- Four Go tests.

**Out of scope:**
- Multi-parent requirements tuning (current design: ALL generalizations must match to kill; sensible default, can refine later).
- Tunable sample-count via CUE config slot (hardcode 3 in Go; revisit if a future phase needs it).
- Pruning Restrict-derived or future meta-op families that don't go through Transpose/Compose.
- Touching H19 / H19Criterial themselves — they remain correct for their actual purpose (data-unit duplicates).

## Design

### Part B: Preventive commutativity check in `transpose-op`

**Location:** `internal/dsl/builtins_math.go`, modifying `bTransposeOp`.

After the existing precondition gates (unit exists, is BinaryOp, has defn, domain len 2), and before the new-unit construction:

```
if domain[0] == domain[1]:
    samples = drawDataBearingExamples(domain[0], 3)
    if len(samples) >= 2:
        allEqual = true
        for each pair (a, b) in C(samples, 2) — the 2-combinations (i<j) — capped at 3 pairs:
            r1 = runDefn(f.defn, [a, b])
            r2 = runDefn(f.defn, [b, a])
            if !valuesEqual(r1, r2):
                allEqual = false
                break
        if allEqual:
            push Nil; return   // treat as "commutative → no transpose needed"
// else: proceed to create Transpose-<op> as before
```

**`drawDataBearingExamples(typeName, n)`**: walks `<typeName>.examples`, resolves each entry to a unit, checks for populated `data` slot, collects up to `n` distinct data values. Skips raw non-unit entries and entries without data.

**`valuesEqual`** for the commutativity check: use `set-equal?` semantics when both values are lists, `==` for ints, `==` for strings. Reject (treat as unequal) on type mismatch. This matches the semantics H-Conjecture already uses for SetEqual.

**Fallback behavior:** if fewer than 2 data-bearing examples are available for the domain type, skip the check entirely and create the Transpose (we can't confidently call the op commutative without evidence). This preserves existing behavior on cold domains.

**When `domain[0] != domain[1]`:** skip the check entirely. Transpose of asymmetric-domain ops is always potentially useful and commutativity doesn't semantically apply.

### Part A: `applics-redundant?` builtin + `H-SemanticDup` heuristic

#### DSL builtin

**Location:** `internal/dsl/builtins_math.go`.

```
applics-redundant? ( unitName parentName -- bool )
```

Behavior:
- Fetch `unitName.applics` and `parentName.defn`.
- If either is empty/missing → push false (can't verify redundancy without evidence).
- If `unitName.applics` has fewer than 3 entries → push false (insufficient evidence; avoid killing fresh units).
- For each applic `{args, output}` in `unitName.applics`:
  - Resolve each arg-name in `args` to its unit's `data` slot (same resolution `apply-op-args` uses).
  - Run `parent.defn` on those values via sub-VM.
  - Resolve `output` to its unit's `data` slot.
  - Compare using the same `valuesEqual` helper from Part B.
  - If any applic mismatches → push false and return.
- If all applics match → push true.

Applic entries whose args reference missing units are treated as non-matching (conservative — don't kill on ambiguous evidence).

#### Heuristic

**Location:** `domains/common/heuristics.cue`.

```cue
{
    name:    "H-SemanticDup"
    worth:   600
    isA: ["Heuristic", "Anything"]
    english: "Kill ops whose observed applics are fully reproduced by a generalization"
    overallRecord: {successes: 0, failures: 0}
    ifPotentiallyRelevant: #"""
        "ArgU" @ "Op" isa?
        "ArgU" @ "creditors" get-slot nil !=
        and
        "ArgU" @ "generalizations" get-slot nil !=
        and
        "ArgU" @ "applics" get-slot nil !=
        and
        "ArgU" @ "applics" get-slot list-length 3 >=
        and
        "ArgU" @ "semDupChecked" get-slot nil =
        and
        """#
    thenCompute: #"""
        true "killed" !
        "ArgU" @ "generalizations" get-slot
        each
            it "parent" !
            "killed" @ not
            if else
                "ArgU" @ "parent" @ applics-redundant?
                if
                    "H-SemanticDup: " "ArgU" @ concat " redundant vs " concat "parent" @ concat " — killing" concat print
                    "ArgU" @ kill-unit
                    false "killed" !
                then
            then
        end
        true "ArgU" @ "semDupChecked" set-slot
        """#
},
```

**Semantics:** iterate generalizations (for Transpose-Add, that's [Add]; for Compose-f-g, that's [f, g]). If ANY generalization proves redundant, kill. (Single-parent redundancy is enough; a Compose-Successor-Identity would be redundant against Successor alone.) The `killed` sentinel stops iteration early once a kill happens to avoid redundant work.

**Creditors requirement:** gates on `creditors != nil` — prevents killing hand-seeded base ops, which have no creditor. Only machine-generated ops are eligible.

**One-shot flag:** `semDupChecked` prevents re-firing even if the kill was blocked (ArgU already dead). Avoids churning the agenda.

**Worth 600:** below H19-EliminateDuplicates (700) but above H-RunOnExamples (750 start, drops) — enough to fire when focus lands but not to dominate.

### Tests

All in `internal/engine/engine_test.go`:

#### `TestTransposeOpSkipsCommutative`
1. Load math domain; Number.examples seeded with N-1..N-20 instance units (existing seed).
2. Call `transpose-op` on `Add`. `Add` is `x,y → x+y`; commutative. Assert returns Nil and `Transpose-Add` is NOT in the store.
3. Call `transpose-op` on `SetDifference`. Non-commutative. Assert returns `"Transpose-SetDifference"` and the unit exists.

#### `TestTransposeOpFallbackNoSamples`
1. Load math domain.
2. Create a synthetic BinaryOp `SynthOp` with domain `[TestType, TestType]`, range `[TestType]`, defn `+`. Do NOT seed TestType.examples.
3. Call `transpose-op` on SynthOp. Assert returns `"Transpose-SynthOp"` and the unit exists (fallback path).

#### `TestApplicsRedundantBuiltin`
1. Load math domain.
2. Manually construct applics on a synthetic unit that match what Add would produce on the same args.
3. Call `"FakeAdd" "Add" applics-redundant?`. Assert pushes true.
4. Construct applics with divergent outputs. Assert pushes false.
5. Construct a unit with fewer than 3 applics. Assert pushes false (insufficient evidence).

#### `TestHSemanticDupKillsRedundant`
1. Load math domain.
2. Manually create `Transpose-Add` unit with applics that match Add's (simulating post-H-RunOnExamples state).
3. Fire `H-SemanticDup` on Transpose-Add.
4. Assert `Transpose-Add` no longer in store (killed).
5. Assert `Add` still in store (parent untouched).

### File Structure

| File | Action | Purpose |
|---|---|---|
| `internal/dsl/builtins_math.go` | Modify | Extend `bTransposeOp` with commutativity sampling; add `bApplicsRedundant` + helpers |
| `domains/common/heuristics.cue` | Modify (append 1 unit) | Add H-SemanticDup |
| `internal/engine/engine_test.go` | Modify (append 4 tests) | 4 tests |

No new files.

## Rationale

**Why sample at creation (B) AND detect post-hoc (A) rather than one or the other?**
- B alone misses Compose redundancies (e.g., Compose-Successor-Identity if Identity ever exists) and any Transpose that happens to be non-commutative on its typed domain but produces identical outputs on all observed inputs within a session.
- A alone creates cruft first and cleans up later, costing store space and focus cycles.
- Together: B cheap-guards the common case (Transpose of commutative op), A catches everything else generically.

**Why 3 samples, not 1 or 5?**
- 1 is insufficient — a single coincidental equality (e.g., 2+2=2+2, then suddenly Add is "commutative" per identical-args) would produce false positives.
- Pairs drawn from 3 distinct values give 3 distinct (a,b) ordered tuples with a≠b — enough to distinguish commutative from non-commutative on typical arithmetic/set ops.
- 5 is gratuitous for our current domain; the cost is sub-VM execution time, but no meaningful accuracy gain.

**Why `creditors != nil` gate on H-SemanticDup?**
- Seed units (Add, Multiply, SetUnion, ...) have no creditor in the CUE seed. They're load-bearing and must not be killable by a detector looking at their own applics. The creditors gate ensures only machine-generated ops are eligible.

**Why not fix H19Criterial instead?**
- H19Criterial's purpose is syntactic criterial-slot matching — a legitimate duplicate-detection mechanism for hand-created specializations that happen to be identical (`{isA: [Op], domain: [Set, Set], range: [Set], defn: "set-union"}` × 2 is a genuine duplicate). Bending it to do semantic comparison would muddy its responsibility. H-SemanticDup earns its own name.

**Why not touch H19-EliminateDuplicates?**
- H19-EliminateDuplicates handles data-unit duplicates (two Set units with identical elements). That's a different detection problem with different evidence (the `data` slot). Extending it to also handle ops would blur the single-responsibility line. Better to have a new heuristic with a clear name.

## Risk

Low-medium:
- `bTransposeOp` gains a conditional sampling branch — guarded by `domain[0]==domain[1]` so asymmetric-domain ops are unaffected. Fallback path preserves existing behavior on cold domains.
- `bApplicsRedundant` executes arbitrary defns via sub-VM. Must not mutate the main VM's state; reuses the `subExecute` pattern from `bApplyOpArgs`.
- `H-SemanticDup` kills units. The 3-applic minimum and `creditors != nil` gate limit blast radius. Worst-case: a flaky defn that happened to produce identical outputs on 3 observations gets falsely killed — acceptable, since the evidence genuinely pointed to equivalence.

## Expected runtime effects (observational)

After a 300-cycle math run we expect:
- Transpose-Add, Transpose-Multiply, Transpose-SetUnion, Transpose-SetIntersect NOT created (caught by Part B).
- Transpose-SetDifference, Transpose-GCD (non-commutative) created normally.
- If any Compose-* unit turns out to be redundant (e.g., chaining with a no-op identity), Part A kills it once applics accumulate.
- Expected kill count visible in the log: `H-SemanticDup: <name> redundant vs <parent>` lines.
- Total Transpose-* unit count should drop from 12 (observed pre-fix) to roughly 2-4.
