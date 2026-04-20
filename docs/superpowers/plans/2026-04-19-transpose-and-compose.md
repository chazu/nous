# Phase 5.6 A+B: Transpose and Compose Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two meta-operation constructors — `transpose-op` and `compose-ops` — as DSL builtins, plus two heuristics (`H-Transpose`, `H-Compose`) that invoke them. Replaces the ad-hoc `H-CheckDomain` SelfCompose path.

**Architecture:** Two new Go builtins in `internal/dsl/builtins_math.go`, two new CUE heuristics in `domains/common/heuristics.cue`, one deletion from `domains/math/heuristics.cue`. No engine-package changes.

**Tech Stack:** Go (builtins + tests), CUE (heuristics), nous DSL (stack-based concatenative language for defn bodies).

**Spec reference:** `docs/superpowers/specs/2026-04-19-transpose-and-compose-design.md`

---

## File Structure

| File | Action | Purpose |
|---|---|---|
| `internal/dsl/builtins_math.go` | Modify | Add `bTransposeOp` and `bComposeOps` + registrations |
| `domains/common/heuristics.cue` | Modify (append 2 units) | Add H-Transpose and H-Compose |
| `domains/math/heuristics.cue` | Modify (delete unit) | Remove H-CheckDomain (lines 195-235) |
| `internal/engine/engine_test.go` | Modify (append 4 tests) | TestTransposeOp, TestHTransposeFires, TestComposeOps, TestHComposeFires |
| `docs/eurisko-parity-phases.md` | Modify | Mark 5.6 A + B COMPLETE |

No new files.

---

## Task 1: `transpose-op` DSL builtin + unit test

**Files:**
- Modify: `internal/dsl/builtins_math.go` (append function + registration)
- Modify: `internal/engine/engine_test.go` (append `TestTransposeOp`)

### - [ ] Step 1: Write the failing test

Append to the end of `internal/engine/engine_test.go`:

```go
// Phase 5.6A: transpose-op creates Transpose-<op> for any BinaryOp,
// reversing the domain and prefixing the defn with `swap`. Idempotent —
// calling twice returns the same unit name.
func TestTransposeOp(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	// Non-commutative BinaryOp → creates Transpose variant.
	v, err := eng.VM.Execute(`"SetDifference" transpose-op`)
	if err != nil {
		t.Fatalf("transpose-op SetDifference: %v", err)
	}
	if v.AsString() != "Transpose-SetDifference" {
		t.Fatalf("returned name = %q; want Transpose-SetDifference", v.AsString())
	}

	tu := eng.Store.Get("Transpose-SetDifference")
	if tu == nil {
		t.Fatal("Transpose-SetDifference unit not in store")
	}

	isA := tu.GetStrings("isA")
	hasBinary, hasOp := false, false
	for _, tag := range isA {
		if tag == "BinaryOp" {
			hasBinary = true
		}
		if tag == "Op" {
			hasOp = true
		}
	}
	if !hasBinary || !hasOp {
		t.Errorf("isA=%v; want BinaryOp and Op", isA)
	}

	// SetDifference.domain = [Set, Set] — reversed still [Set, Set], but the
	// defn should have `swap ` prefix distinguishing the operation.
	if got := tu.GetStrings("domain"); len(got) != 2 || got[0] != "Set" || got[1] != "Set" {
		t.Errorf("domain=%v; want [Set Set]", got)
	}
	if got := tu.GetStrings("range"); len(got) != 1 || got[0] != "Set" {
		t.Errorf("range=%v; want [Set]", got)
	}
	defn := tu.GetString("defn")
	if defn == "" || defn[:5] != "swap " {
		t.Errorf("defn=%q; want prefix 'swap '", defn)
	}
	if gens := tu.GetStrings("generalizations"); len(gens) != 1 || gens[0] != "SetDifference" {
		t.Errorf("generalizations=%v; want [SetDifference]", gens)
	}

	// Idempotency: second call returns same name, does not duplicate.
	v2, err := eng.VM.Execute(`"SetDifference" transpose-op`)
	if err != nil {
		t.Fatalf("second transpose-op: %v", err)
	}
	if v2.AsString() != "Transpose-SetDifference" {
		t.Errorf("second call returned %q; want Transpose-SetDifference", v2.AsString())
	}

	// UnaryOp → nil (domain length != 2).
	v3, err := eng.VM.Execute(`"DivisorsOf" transpose-op`)
	if err != nil {
		t.Fatalf("transpose-op DivisorsOf: %v", err)
	}
	if !v3.IsNil() {
		t.Errorf("transpose-op on UnaryOp: expected nil, got %v", v3)
	}
	if eng.Store.Has("Transpose-DivisorsOf") {
		t.Error("Transpose-DivisorsOf should not exist (precondition failure)")
	}

	// Semantic check: invoking the transposed defn swaps args.
	// SetDifference: set-diff takes (a, b) → a\b. Transposed: b\a.
	// Feed [1,2,3] and [2,3,4]: SetDifference gives [1]; Transpose gives [4].
	v4, err := eng.VM.Execute(`3 iota 2 3 4 3 list-of "Transpose-SetDifference" apply-op`)
	if err != nil {
		t.Fatalf("apply transposed: %v", err)
	}
	// Result should be a list containing 4 (the element in second arg not in first).
	list := v4.AsList()
	has4 := false
	for _, x := range list {
		if x.AsInt() == 4 {
			has4 = true
		}
	}
	if !has4 {
		t.Errorf("Transpose-SetDifference([0,1,2], [2,3,4]) = %v; expected contains 4", list)
	}
}
```

`Value.IsNil()` is defined in `internal/dsl/value.go:43` and needs no additional import — `eng.VM.Execute` already returns a `dsl.Value` in scope.

### - [ ] Step 2: Run test to verify it fails

```bash
cd /Users/chazu/dev/go/nous
go test ./internal/engine/ -run TestTransposeOp -v
```

Expected: FAIL with `unknown word: transpose-op` OR `Transpose-SetDifference unit not in store`.

### - [ ] Step 3: Implement the builtin

Open `internal/dsl/builtins_math.go`. In the `init()` function, add registration right after the existing `builtins["apply-pred"]` line (around line 82):

```go
	builtins["transpose-op"] = bTransposeOp
```

At the end of the file (after the last function), add:

```go
// transpose-op: ( opName -- newOpName | nil )
// Creates Transpose-<opName> for any BinaryOp: domain reversed, defn
// prefixed with `swap`. Idempotent — if Transpose-<op> already exists,
// returns its name without modifying. Returns nil on precondition failure
// (not a BinaryOp, missing defn, wrong-arity domain).
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
		vm.push(StringVal(newName))
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
	vm.Store.SetSlot(newName, "generalizations", []any{opName})
	vm.push(StringVal(newName))
	return nil
}
```

The `unit` package is already imported at the top of the file (line 7: `"github.com/chazu/nous/internal/unit"`). `StringVal` and `Nil` are in `internal/dsl/value.go`.

### - [ ] Step 4: Run test to verify it passes

```bash
go test ./internal/engine/ -run TestTransposeOp -v
```

Expected: PASS.

If it fails with a semantic check on the transposed output (the `has4` assertion), debug by adding a `t.Logf("transposed result: %v", list)` to see what `apply-op` actually returned. Common causes: `list-of` pushed args in wrong order, or `set-diff` has different semantics than assumed. The assertion "contains 4" is deliberately weak — if it still fails, the issue is deeper.

### - [ ] Step 5: Commit

```bash
git add internal/dsl/builtins_math.go internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.6A — transpose-op DSL builtin

New builtin creates Transpose-<op> for any BinaryOp, reversing the
domain and prefixing the defn with `swap`. Idempotent. Returns nil on
UnaryOps or missing defn.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: H-Transpose heuristic + engine-level test

**Files:**
- Modify: `domains/common/heuristics.cue` (append unit)
- Modify: `internal/engine/engine_test.go` (append `TestHTransposeFires`)

### - [ ] Step 1: Write the failing test

Append to `internal/engine/engine_test.go`:

```go
// Phase 5.6A: H-Transpose fires on unit-focus of a BinaryOp, creates the
// Transpose variant via the transpose-op builtin, and marks the op as
// transposed to prevent re-firing.
func TestHTransposeFires(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H-Transpose") {
		t.Fatal("H-Transpose heuristic not loaded from common/heuristics.cue")
	}

	eng.fireUnitRule("H-Transpose", "SetDifference")

	if !eng.Store.Has("Transpose-SetDifference") {
		t.Fatal("Transpose-SetDifference not created after H-Transpose fired")
	}
	sd := eng.Store.Get("SetDifference")
	if tr, _ := sd.Get("transposed").(bool); !tr {
		t.Errorf("SetDifference.transposed = %v; want true", sd.Get("transposed"))
	}

	// Re-firing should not create anything new (one-shot guard).
	preCount := eng.Store.Count()
	eng.fireUnitRule("H-Transpose", "SetDifference")
	if eng.Store.Count() != preCount {
		t.Errorf("re-firing H-Transpose created new units; pre=%d post=%d", preCount, eng.Store.Count())
	}
}
```

### - [ ] Step 2: Run test to verify it fails

```bash
go test ./internal/engine/ -run TestHTransposeFires -v
```

Expected: FAIL with `H-Transpose heuristic not loaded from common/heuristics.cue`.

### - [ ] Step 3: Add the heuristic

Open `domains/common/heuristics.cue`. Find the last unit in the file (check with `tail -30`). Append a new unit before the closing `]`:

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

Match the indentation of existing heuristics in the file (single tab per entry).

### - [ ] Step 4: Run test to verify it passes

```bash
go test ./internal/engine/ -run TestHTransposeFires -v
```

Expected: PASS.

Also run the full engine test package to catch any side effect:

```bash
go test ./internal/engine/ -v 2>&1 | tail -10
```

All tests should still PASS. If a test that was working now fails — likely `TestEngineRuns` or `TestEngineCreatesUnits` — it may be because H-Transpose now fires during normal runs. Investigate before proceeding.

### - [ ] Step 5: Commit

```bash
git add domains/common/heuristics.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.6A — H-Transpose heuristic

Fires on unit-focus of any BinaryOp with a defn. Invokes transpose-op
and schedules examples task on the new Transpose-<op> unit. One-shot per
op via `transposed` flag.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: `compose-ops` DSL builtin + unit test

**Files:**
- Modify: `internal/dsl/builtins_math.go` (append function + registration)
- Modify: `internal/engine/engine_test.go` (append `TestComposeOps`)

### - [ ] Step 1: Write the failing test

Append to `internal/engine/engine_test.go`:

```go
// Phase 5.6B: compose-ops creates Compose-<f>-<g> when range(f) == domain(g)
// as ordered string slices. Composed defn chains apply-op on f then g.
// Idempotent. Returns nil on mismatch or missing defns.
func TestComposeOps(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	// Mismatch: range(DivisorsOf)=[Set], domain(SetUnion)=[Set,Set] → nil.
	v, err := eng.VM.Execute(`"DivisorsOf" "SetUnion" compose-ops`)
	if err != nil {
		t.Fatalf("compose-ops DivisorsOf SetUnion: %v", err)
	}
	if !v.IsNil() {
		t.Errorf("mismatch case returned %v; want nil", v)
	}
	if eng.Store.Has("Compose-DivisorsOf-SetUnion") {
		t.Error("Compose-DivisorsOf-SetUnion should not exist on mismatch")
	}

	// Match: range(Successor)=[Number], domain(Successor)=[Number] → self-compose.
	v2, err := eng.VM.Execute(`"Successor" "Successor" compose-ops`)
	if err != nil {
		t.Fatalf("compose-ops Successor Successor: %v", err)
	}
	if v2.AsString() != "Compose-Successor-Successor" {
		t.Fatalf("returned %q; want Compose-Successor-Successor", v2.AsString())
	}
	cu := eng.Store.Get("Compose-Successor-Successor")
	if cu == nil {
		t.Fatal("Compose-Successor-Successor not in store")
	}
	isA := cu.GetStrings("isA")
	hasUnary, hasOp := false, false
	for _, tag := range isA {
		if tag == "UnaryOp" {
			hasUnary = true
		}
		if tag == "Op" {
			hasOp = true
		}
	}
	if !hasUnary || !hasOp {
		t.Errorf("isA=%v; want UnaryOp and Op", isA)
	}
	if got := cu.GetStrings("domain"); len(got) != 1 || got[0] != "Number" {
		t.Errorf("domain=%v; want [Number]", got)
	}
	if got := cu.GetStrings("range"); len(got) != 1 || got[0] != "Number" {
		t.Errorf("range=%v; want [Number]", got)
	}
	gens := cu.GetStrings("generalizations")
	if len(gens) != 2 || gens[0] != "Successor" || gens[1] != "Successor" {
		t.Errorf("generalizations=%v; want [Successor Successor]", gens)
	}

	// Semantic check: Compose-Successor-Successor(5) should be 7.
	v3, err := eng.VM.Execute(`5 "Compose-Successor-Successor" apply-op`)
	if err != nil {
		t.Fatalf("apply Compose-Successor-Successor: %v", err)
	}
	if v3.AsInt() != 7 {
		t.Errorf("Compose(Successor,Successor)(5) = %d; want 7", v3.AsInt())
	}

	// Binary match: range(Add)=[Number], domain(Square)=[Number] → binary compose.
	v4, err := eng.VM.Execute(`"Add" "Square" compose-ops`)
	if err != nil {
		t.Fatalf("compose-ops Add Square: %v", err)
	}
	if v4.AsString() != "Compose-Add-Square" {
		t.Fatalf("returned %q; want Compose-Add-Square", v4.AsString())
	}
	cu2 := eng.Store.Get("Compose-Add-Square")
	if cu2 == nil {
		t.Fatal("Compose-Add-Square not in store")
	}
	if got := cu2.GetStrings("domain"); len(got) != 2 || got[0] != "Number" || got[1] != "Number" {
		t.Errorf("Compose-Add-Square domain=%v; want [Number Number]", got)
	}
	// Compose-Add-Square(3, 4) = Square(Add(3,4)) = Square(7) = 49.
	v5, err := eng.VM.Execute(`3 4 "Compose-Add-Square" apply-op`)
	if err != nil {
		t.Fatalf("apply Compose-Add-Square: %v", err)
	}
	if v5.AsInt() != 49 {
		t.Errorf("Compose(Add,Square)(3,4) = %d; want 49", v5.AsInt())
	}

	// Idempotency: second call returns same unit.
	v6, err := eng.VM.Execute(`"Add" "Square" compose-ops`)
	if err != nil {
		t.Fatalf("second compose-ops: %v", err)
	}
	if v6.AsString() != "Compose-Add-Square" {
		t.Errorf("second call returned %q; want Compose-Add-Square", v6.AsString())
	}
}
```

### - [ ] Step 2: Run test to verify it fails

```bash
go test ./internal/engine/ -run TestComposeOps -v
```

Expected: FAIL with `unknown word: compose-ops`.

### - [ ] Step 3: Implement the builtin

In `internal/dsl/builtins_math.go`, add registration after `builtins["transpose-op"]` in `init()`:

```go
	builtins["compose-ops"] = bComposeOps
```

At the end of the file, add:

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
		vm.push(StringVal(newName))
		return nil
	}

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
	vm.push(StringVal(newName))
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

Add `"fmt"` to the import block at the top of `internal/dsl/builtins_math.go`. If `stringSlicesEqual` already exists elsewhere in the `dsl` package (check with `grep -rn "func stringSlicesEqual" internal/dsl/`), drop the local copy and use the existing one.

### - [ ] Step 4: Run test to verify it passes

```bash
go test ./internal/engine/ -run TestComposeOps -v
```

Expected: PASS.

If `apply-op` on Compose-Add-Square returns an unexpected value, debug with `t.Logf("got: %v", v5.AsAny())`. The likely issue if it fails: the defn's quoted string escaping inside CUE/Go — verify the unit's defn slot contains literally `"Add" apply-op "Square" apply-op` (with quote marks) by reading it in the test.

### - [ ] Step 5: Commit

```bash
git add internal/dsl/builtins_math.go internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.6B — compose-ops DSL builtin

New builtin creates Compose-<f>-<g> when range(f) matches domain(g).
Composed defn chains apply-op on f then g. Arity matches f's; range
matches g's. Idempotent. Returns nil on range/domain mismatch.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: H-Compose heuristic + delete H-CheckDomain + engine-level test

**Files:**
- Modify: `domains/common/heuristics.cue` (append unit)
- Modify: `domains/math/heuristics.cue` (delete H-CheckDomain unit, lines ~195-235)
- Modify: `internal/engine/engine_test.go` (append `TestHComposeFires`)

### - [ ] Step 1: Write the failing test

Append to `internal/engine/engine_test.go`:

```go
// Phase 5.6B: H-Compose fires on unit-focus of any Op, iterates candidate
// g's with matching range/domain, creates Compose-<ArgU>-<g> via compose-ops.
// Caps at 3 new composes per firing. One-shot per source op.
func TestHComposeFires(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H-Compose") {
		t.Fatal("H-Compose heuristic not loaded from common/heuristics.cue")
	}

	preCount := 0
	for _, name := range eng.Store.All() {
		if len(name) >= 8 && name[:8] == "Compose-" {
			preCount++
		}
	}

	eng.fireUnitRule("H-Compose", "Successor")

	// Successor's range [Number] matches any UnaryOp or first-domain [Number]
	// prefix — so at minimum Compose-Successor-Successor must exist.
	if !eng.Store.Has("Compose-Successor-Successor") {
		t.Fatal("Compose-Successor-Successor not created after H-Compose fired on Successor")
	}

	// Count new Compose-* units; should be between 1 and 3 (cap).
	postCount := 0
	for _, name := range eng.Store.All() {
		if len(name) >= 8 && name[:8] == "Compose-" {
			postCount++
		}
	}
	created := postCount - preCount
	if created < 1 || created > 3 {
		t.Errorf("created %d Compose-* units; want 1..3 (cap-3)", created)
	}

	// One-shot flag set.
	sc := eng.Store.Get("Successor")
	if c, _ := sc.Get("composed").(bool); !c {
		t.Errorf("Successor.composed = %v; want true", sc.Get("composed"))
	}

	// Re-firing is a no-op for new unit creation.
	mid := eng.Store.Count()
	eng.fireUnitRule("H-Compose", "Successor")
	if eng.Store.Count() != mid {
		t.Errorf("re-firing H-Compose created new units; mid=%d post=%d", mid, eng.Store.Count())
	}
}
```

### - [ ] Step 2: Run test to verify it fails

```bash
go test ./internal/engine/ -run TestHComposeFires -v
```

Expected: FAIL with `H-Compose heuristic not loaded from common/heuristics.cue`.

### - [ ] Step 3: Add H-Compose heuristic

Open `domains/common/heuristics.cue`. Append another unit before the closing `]` (after the H-Transpose unit added in Task 2):

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

### - [ ] Step 4: Delete H-CheckDomain

Open `domains/math/heuristics.cue`. Delete the entire H-CheckDomain unit (lines 195-235 in the current file — the whole entry from the opening `{` before `name: "H-CheckDomain"` through its closing `},`). Verify with `grep -n "H-CheckDomain" domains/math/heuristics.cue` — should return no matches.

Sanity check before deleting: `grep -rn "H-CheckDomain\|SelfCompose" internal/ domains/ 2>/dev/null` should show references only in this file and in doc files (not in any `.go` file). Confirmed safe per plan analysis — no Go tests reference `SelfCompose-*` unit names.

### - [ ] Step 5: Run test to verify it passes

```bash
go test ./internal/engine/ -run TestHComposeFires -v
```

Expected: PASS.

Also run the full engine suite:

```bash
go test ./internal/engine/ 2>&1 | tail -5
```

Expected: all PASS. If any previously-passing test now fails — especially tests that might observe unit counts or SelfCompose-* names — investigate before committing.

### - [ ] Step 6: Commit

```bash
git add domains/common/heuristics.cue domains/math/heuristics.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.6B — H-Compose heuristic, remove H-CheckDomain

H-Compose iterates Op.examples with matching range/domain, caps at 3
new composes per firing. Supersedes H-CheckDomain's SelfCompose branch
which produced shell units without defns.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Regression check and doc updates

**Files:**
- Modify: `docs/eurisko-parity-phases.md` (mark 5.6 A + B COMPLETE)

### - [ ] Step 1: Run the full test suite

```bash
cd /Users/chazu/dev/go/nous
go test ./... 2>&1 | tail -15
```

Expected: all packages PASS. If any test fails, STOP and report the failure. Do not fix unilaterally.

### - [ ] Step 2: 100-cycle math smoke run

```bash
go build -o nous ./cmd/nous 2>&1 | tail -3
./nous -domains-dir ./domains -domain math -cycles 100 -verbosity 0 2>&1 | tail -30
```

Expected: clean exit. Observe in the output:
- Any `Transposed <op>` lines (H-Transpose firings)
- Any `Composed <f> . <g>` lines (H-Compose firings)
- Whether H19-EliminateDuplicates kills any Transpose-* variants of commutative ops (Add, Multiply, SetUnion, SetIntersect)
- Unit count growth vs baseline (prior phase runs created ~220 units from ~170 seed)

If the run panics, kill within 30s, capture tail, STOP.

### - [ ] Step 3: Update phase tracking doc

Edit `docs/eurisko-parity-phases.md`. Find the section for `**5.6: Meta-operations with algorithms** -- PARTIAL`. The current body mentions C.1 and C.2 complete, and A/B/D deferred. Update A and B entries:

Find this line:
```
- **A Transpose** -- deferred. `transpose-op` builtin + H-Transpose heuristic that creates `Transpose-<op>` variants for non-commutative binary ops.
```

Replace with:
```
- **A Transpose** -- COMPLETE (2026-04-19). `transpose-op` builtin + H-Transpose heuristic create `Transpose-<op>` for any BinaryOp; domain reversed, defn prefixed with `swap`. Commutativity handled reactively by H19-EliminateDuplicates. Plan: `docs/superpowers/plans/2026-04-19-transpose-and-compose.md`.
```

Find this line:
```
- **B Compose** -- deferred. `compose-ops` builtin synthesizing a new op whose defn chains `apply-op(f)` then `apply-op(g)`, with range/domain compatibility checks. Supersedes the ad-hoc H-CheckDomain SelfCompose code path.
```

Replace with:
```
- **B Compose** -- COMPLETE (2026-04-19). `compose-ops` builtin creates `Compose-<f>-<g>` when range(f) == domain(g) as ordered string slices. Composed defn chains apply-op on f then g. H-Compose iterates Op.examples capped at 3/firing. H-CheckDomain deleted (its SelfCompose branch produced shell units without defns; falls out naturally from H-Compose with f=g).
```

Update the overall PARTIAL marker to reflect A+B complete:
```
**5.6: Meta-operations with algorithms** -- PARTIAL (A, B, C.1, C.2 complete; D deferred)
```

Update the summary table at the bottom of the file. Find the Phase 5 row (should currently be `PARTIAL (5.2, 5.6 C.1/C.2, 5.9, 5.10, 5.11 numeric-preds + H27/H28)`) and update to:
```
| 5 | Type hierarchy + operations | 12 | PARTIAL (5.2, 5.6 A/B/C.1/C.2, 5.9, 5.10, 5.11 numeric-preds + H27/H28) |
```

### - [ ] Step 4: Update memory index

Edit `/Users/chazu/.claude/projects/-Users-chazu-dev-go-nous/memory/project_nous_phases.md`:
- Update frontmatter `description` to reflect 5.6 A+B complete as of 2026-04-19
- In status section: add 5.6A, 5.6B to the Complete list
- "Next natural moves": promote `Phase 5.3 + 5.4 — projections + structure classification` to #1, add `Phase 5.1 — OSet type + ops` as #2, `Phase 5.5/5.7/5.8 — per-type/choice/logical ops as units` as #3

Keep pudl ↔ nous integration note.

### - [ ] Step 5: Commit

```bash
cd /Users/chazu/dev/go/nous
git add docs/eurisko-parity-phases.md
git commit -m "$(cat <<'EOF'
docs: mark Phase 5.6 A+B (Transpose, Compose) COMPLETE

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Completion Criteria

- [ ] `go test ./internal/engine/ -run "TestTransposeOp|TestHTransposeFires|TestComposeOps|TestHComposeFires" -v` passes
- [ ] `go test ./...` passes with no regressions
- [ ] 100-cycle math smoke run exits cleanly
- [ ] H-CheckDomain gone, no SelfCompose-* references in .go files
- [ ] Phase doc shows 5.6 A and B COMPLETE
- [ ] Memory index updated
- [ ] Five commits on branch (one per task)

---

## Notes for the implementing engineer

- `StringVal` and `Nil` are the Value constructors in `internal/dsl/value.go:33,37`. Not `StringValue`.
- The `unit` package is imported in `internal/dsl/builtins_math.go` line 7. `fmt` is NOT imported — add it in Task 3.
- `vm.Store.SetSlot(unitName, slotName, value)` triggers inverse maintenance; use it (not `u.Set(...)`) for slots with defined inverses like `generalizations` / `specializations`.
- The DSL's `"Op" examples` form works because `Examples.inverse = "IsA"` in `domains/common/slots.cue:44` — so every unit with `isA` entries auto-populates the parent category's `examples` slot.
- `apply-op` pops args then opName; check the order carefully when writing test programs.
- `fireUnitRule(heuristicName, targetUnitName)` is unexported but tests live in the same `engine` package and can call it directly.
- For H-Transpose's `transposed` and H-Compose's `composed` one-shot flags: stored as `true` bool; `get-slot` returns nil (the slot is genuinely absent) before first firing, and `true` after.
- `Value.IsNil()` is the canonical way to check for nil (method at `internal/dsl/value.go:43`). Prefer it over enum comparisons.
- If any step's output differs substantially from the expected result, stop and report rather than pressing on.
