# Phase 5.6 A + B: Transpose and Compose Meta-Operations

**Date:** 2026-04-19
**Status:** Design approved, ready for implementation plan
**Plan ref:** `docs/eurisko-parity-phases.md` §5.6 A (Transpose) and §5.6 B (Compose)

## Motivation

EURISKO's meta-operation layer lets heuristics build new ops from existing ops. Two of the most load-bearing meta-ops are Transpose (swap a binary op's arguments) and Compose (chain f then g). Together they multiply the op space: once Add/Multiply/Successor/Square exist as units (shipped in Phase 5.9), Transpose and Compose can synthesize Transpose-SetDifference, Compose-Successor-Successor (+2), Compose-Square-Successor, etc.

This phase also supersedes the ad-hoc `H-CheckDomain` SelfCompose branch in `domains/math/heuristics.cue`, which creates `SelfCompose-<op>` shell units without executable defns. A proper Compose builtin gives those units real defns.

## Scope

**In scope:**
- DSL builtin `transpose-op (opName -- newOpName | nil)`
- DSL builtin `compose-ops (fName gName -- newOpName | nil)`
- Heuristic `H-Transpose` (CUE)
- Heuristic `H-Compose` (CUE)
- Deletion of H-CheckDomain's SelfCompose branch (lines ~210-234)
- Four Go tests

**Out of scope (deferred):**
- Explicit commutativity detection (future H-DetectCommutative) — H19-EliminateDuplicates handles it reactively
- Compose with predicates (Restrict variant) — Phase 5.6D
- Multi-step composition (compose of compose) — emergent from iteration
- Phase 5.6C remainder items — already complete (C.1 + C.2)

## Design

### 5.6A — Transpose

#### DSL builtin

Add to `internal/dsl/builtins_math.go`:

```go
// transpose-op: ( opName -- newOpName | nil )
// Creates Transpose-<opName> for any BinaryOp: domain reversed, defn
// prefixed with `swap`. Idempotent — if Transpose-<op> already exists,
// returns its name without modifying. Returns nil on precondition
// failure (not a BinaryOp, missing defn, wrong-arity domain).
func bTransposeOp(vm *VM) error {
    opName := vm.pop().AsString()
    u := vm.Store.Get(opName)
    if u == nil {
        vm.push(Nil())
        return nil
    }
    if !vm.Store.IsA(opName, "BinaryOp") {
        vm.push(Nil())
        return nil
    }
    defn := u.GetString("defn")
    if defn == "" {
        vm.push(Nil())
        return nil
    }
    domain := u.GetStrings("domain")
    if len(domain) != 2 {
        vm.push(Nil())
        return nil
    }

    newName := "Transpose-" + opName
    if vm.Store.Has(newName) {
        vm.push(StringValue(newName))
        return nil
    }

    newU := unit.New(newName)
    newU.Set("isA", []string{"BinaryOp", "Op", "MathOp", "Anything"})
    newU.SetWorth(500)
    newU.Set("domain", []string{domain[1], domain[0]})
    newU.Set("range", u.GetStrings("range"))
    newU.Set("defn", "swap "+defn)
    newU.Set("creditors", []string{"H-Transpose"})
    vm.Store.Put(newU)
    // Wire generalizations via SetSlot so the inverse (specializations)
    // auto-updates on the parent unit.
    vm.Store.SetSlot(newName, "generalizations", []any{opName})
    vm.push(StringValue(newName))
    return nil
}
```

Registered in the `builtins` map: `builtins["transpose-op"] = bTransposeOp`.

#### Heuristic

Add to `domains/common/heuristics.cue`:

```cue
{
    name:    "H-Transpose"
    worth:   500
    isA: ["Heuristic", "Anything"]
    english: "Create transposed version of binary ops"
    overallRecord: {successes: 0, failures: 0}
    ifPotentiallyRelevant: #"""
        "ArgU" @ "BinaryOp" isa?
        "ArgU" @ "defn" get-slot nil !=
        and
        "ArgU" @ "transposed" get-slot nil =
        and
        """#
    thenCompute: #"""
        "ArgU" @ transpose-op
        "newOp" !
        "newOp" @ nil !=
        if
            400 "newOp" @ "examples" "Examples for transposed op" add-task
            "Transposed " "ArgU" @ concat print
        then
        true "ArgU" @ "transposed" set-slot
        """#
},
```

One-shot per op via `transposed` flag. On fire: call builtin, schedule examples task on new unit.

### 5.6B — Compose

#### DSL builtin

Add to `internal/dsl/builtins_math.go`:

```go
// compose-ops: ( fName gName -- newOpName | nil )
// Creates Compose-<f>-<g> when range(f) matches domain(g) as ordered
// string slices. Composed defn chains apply-op on f then g. Arity of
// the result matches f's arity; range matches g's. Idempotent.
func bComposeOps(vm *VM) error {
    gName := vm.pop().AsString()
    fName := vm.pop().AsString()
    f := vm.Store.Get(fName)
    g := vm.Store.Get(gName)
    if f == nil || g == nil {
        vm.push(Nil())
        return nil
    }
    fDefn := f.GetString("defn")
    gDefn := g.GetString("defn")
    if fDefn == "" || gDefn == "" {
        vm.push(Nil())
        return nil
    }
    fRange := f.GetStrings("range")
    gDomain := g.GetStrings("domain")
    if !stringSlicesEqual(fRange, gDomain) {
        vm.push(Nil())
        return nil
    }

    newName := "Compose-" + fName + "-" + gName
    if vm.Store.Has(newName) {
        vm.push(StringValue(newName))
        return nil
    }

    // Pick arity bucket from f (number of args in).
    fDomain := f.GetStrings("domain")
    arityBucket := "UnaryOp"
    if len(fDomain) == 2 {
        arityBucket = "BinaryOp"
    } else if len(fDomain) != 1 {
        vm.push(Nil())
        return nil
    }

    newU := unit.New(newName)
    newU.Set("isA", []string{arityBucket, "Op", "MathOp", "Anything"})
    newU.SetWorth(500)
    newU.Set("domain", append([]string{}, fDomain...))
    newU.Set("range", append([]string{}, g.GetStrings("range")...))
    newU.Set("defn", fmt.Sprintf(`"%s" apply-op "%s" apply-op`, fName, gName))
    newU.Set("creditors", []string{"H-Compose"})
    vm.Store.Put(newU)
    vm.Store.SetSlot(newName, "generalizations", []any{fName, gName})
    vm.push(StringValue(newName))
    return nil
}

// stringSlicesEqual compares two []string for elementwise equality.
func stringSlicesEqual(a, b []string) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }
    return true
}
```

Registered: `builtins["compose-ops"] = bComposeOps`.

Note: if a `stringSlicesEqual` helper already exists in `builtins_math.go` or a sibling file, reuse it and drop the duplicate.

#### Heuristic

Add to `domains/common/heuristics.cue`:

```cue
{
    name:    "H-Compose"
    worth:   500
    isA: ["Heuristic", "Anything"]
    english: "Compose pairs of ops with matching range/domain"
    overallRecord: {successes: 0, failures: 0}
    ifPotentiallyRelevant: #"""
        "ArgU" @ "Op" isa?
        "ArgU" @ "defn" get-slot nil !=
        and
        "ArgU" @ "composed" get-slot nil =
        and
        """#
    thenCompute: #"""
        0 "composeCount" !
        "Op" examples
        each
            it "g" !
            "composeCount" @ 3 <
            if
                "ArgU" @ "g" @ compose-ops
                "newOp" !
                "newOp" @ nil !=
                if
                    400 "newOp" @ "examples" "Examples for composed op" add-task
                    "Composed " "ArgU" @ concat " . " concat "g" @ concat print
                    "composeCount" @ 1 + "composeCount" !
                then
            then
        end
        true "ArgU" @ "composed" set-slot
        """#
},
```

One-shot per op via `composed` flag. Caps at 3 new composes per firing. Iterates `Op`'s examples slot — same enumeration pattern used by H-RunOnExamples.

Note: `compose-ops` itself checks range/domain compatibility, so the heuristic doesn't need to pre-filter. The nil return from the builtin short-circuits.

### H-CheckDomain SelfCompose branch deletion

Delete lines ~210-234 of `domains/math/heuristics.cue` — the entire `thenCompute` block of `H-CheckDomain` that creates `SelfCompose-<op>` shell units. H-CheckDomain can be removed entirely since its only job was the SelfCompose creation; confirm no other code references it before deletion.

`grep -r "H-CheckDomain" domains/ internal/` should return only the unit definition itself and any test that specifically tests H-CheckDomain. If any real dependency surfaces, keep the H-CheckDomain unit stub with an empty `thenCompute` and note the deprecation; otherwise delete the whole unit definition.

### Tests

All in `internal/engine/engine_test.go`:

#### `TestTransposeOp`
Unit-level builtin test:
1. Load math domain.
2. Call `transpose-op` on `SetDifference` via `eng.VM.Execute(`"SetDifference" transpose-op`)`.
3. Assert `Transpose-SetDifference` exists; isA contains BinaryOp/Op; domain=[Set,Set]; range=[Set]; defn starts with `swap `; generalizations=[SetDifference].
4. Call `transpose-op` again — returns same name, no duplicate unit.
5. Call on `DivisorsOf` (UnaryOp) — returns nil, no unit created.

#### `TestComposeOps`
Unit-level builtin test:
1. Load math domain.
2. Call `compose-ops` on `(DivisorsOf, SetUnion)` — range(DivisorsOf)=[Set] but domain(SetUnion)=[Set,Set]; mismatch → nil.
3. Call on `(Successor, Successor)` — both [Number]→[Number]; match. Verify unit `Compose-Successor-Successor` exists; isA contains UnaryOp/Op; domain=[Number]; range=[Number]; defn contains both `apply-op` calls.
4. Execute the composed defn: `5 "Compose-Successor-Successor" apply-op` → 7.
5. Call on `(Add, Square)` — range(Add)=[Number], domain(Square)=[Number]; match. Verify unit; execute `3 4 "Compose-Add-Square" apply-op` → 49.

#### `TestHTransposeFires`
Engine-level: focus SetDifference, fire `H-Transpose`, assert `Transpose-SetDifference` created and `SetDifference.transposed == true`.

#### `TestHComposeFires`
Engine-level: focus Successor, fire `H-Compose`, assert `Compose-Successor-Successor` exists AND count of new Compose-* units is between 1 and 3. Assert `Successor.composed == true`.

### File Structure

| File | Action | Purpose |
|---|---|---|
| `internal/dsl/builtins_math.go` | Modify | Add `bTransposeOp`, `bComposeOps`, registrations |
| `domains/common/heuristics.cue` | Modify | Add H-Transpose, H-Compose units |
| `domains/math/heuristics.cue` | Modify | Delete H-CheckDomain's SelfCompose branch |
| `internal/engine/engine_test.go` | Modify | Add 4 tests |

No new files.

## Rationale

**Why string-slice equality for range/domain match, not subsumption?**
A strict equality check is simple, correct, and matches EURISKO's behavior in practice. Subsumption (e.g., `range(f) ⊆ domain(g)` via isA traversal) would let Compose fire more often but risks incoherent composes where type narrowing happens silently. Defer to a future refinement.

**Why iterate `Op`'s examples slot rather than all Op-isA units in the store?**
`Op.examples` is the canonical list EURISKO uses for this kind of enumeration, and H-RunOnExamples uses the same pattern. Consistent with existing heuristic style.

**Why cap Compose at 3 per firing?**
Without a cap a single H-Compose firing on a heavily-connected op could create dozens of composes. The cap (matching H8's cap-3 and H20's cap-3) keeps agenda pressure bounded. If useful composes are truncated, the one-shot flag means we only do this once per op — so bumping the cap is the right follow-up, not removing the flag.

**Why `swap <defn>` for Transpose?**
`swap` is a stack-op that flips the top two values. For a BinaryOp whose defn expects `[a, b]` on the stack, `swap <defn>` executes on `[b, a]` instead — exactly the transposed behavior.

**Why delete H-CheckDomain?**
Its SelfCompose output was shell units without defns — they couldn't actually be invoked. H-Compose with f=g produces real, executable self-composes. Keeping both would generate duplicate units with different names (`Compose-f-f` vs `SelfCompose-f`), confusing H19-EliminateDuplicates.

## Risk

Low-medium:
- New DSL builtins — straightforward Go code, well-scoped.
- New CUE heuristics — pattern-matched to existing H-RunOnExamples/H8/H20.
- H-CheckDomain deletion — verify no tests depend on `SelfCompose-*` unit names before deleting. If any do, update them to expect `Compose-*` names.
- Worst case: CUE load error on typo (caught on first test run).

## Expected runtime effects (observational)

After a 300-cycle math-domain run we expect:
- At least one Transpose-* unit per non-commutative BinaryOp (SetDifference, GCD is commutative but will be pruned by H19).
- At least one Compose-* unit per Op whose range matches some domain (Compose-Successor-Successor, Compose-Add-Square, Compose-DivisorsOf-*, etc.).
- H19 should kill Transpose variants of commutative ops (Add, Multiply, SetUnion, SetIntersect) once their applics prove structurally identical.
- Total unit count growth of roughly +5-10 meta-op units in a typical run.

None of these are asserted by the tests — they are observational signals for the wrap-up report.
