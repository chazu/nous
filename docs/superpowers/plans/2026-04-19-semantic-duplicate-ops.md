# Semantic Duplicate Operation Detection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent and prune semantically-redundant operations (Transpose variants of commutative ops, and any generated op whose observed applics reproduce a generalization's behavior) via a sampling check at Transpose creation time (Part B) plus a post-hoc detector heuristic (Part A).

**Architecture:** One new Go DSL builtin (`applics-redundant?`), one extension to an existing Go DSL builtin (`bTransposeOp`), one new CUE heuristic (`H-SemanticDup`). No engine-package changes. Sub-VM execution via the existing `subExecute` helper.

**Tech Stack:** Go (builtins + tests), CUE (heuristic), nous DSL (concatenative stack language for heuristic bodies).

**Spec reference:** `docs/superpowers/specs/2026-04-19-semantic-duplicate-ops-design.md`

---

## File Structure

| File | Action | Purpose |
|---|---|---|
| `internal/dsl/builtins_math.go` | Modify | Extend `bTransposeOp` with commutativity sampling; add `bApplicsRedundant` + registration + helpers |
| `domains/common/heuristics.cue` | Modify (append 1 unit) | Add `H-SemanticDup` |
| `internal/engine/engine_test.go` | Modify (append 4 tests) | TestApplicsRedundantBuiltin, TestTransposeOpSkipsCommutative, TestTransposeOpFallbackNoSamples, TestHSemanticDupKillsRedundant |
| `docs/eurisko-parity-phases.md` | Modify (section addendum) | Document the followup as completed |

No new files.

---

## Task 1: `applics-redundant?` DSL builtin + test

**Files:**
- Modify: `internal/dsl/builtins_math.go` (append function + registration + helpers)
- Modify: `internal/engine/engine_test.go` (append `TestApplicsRedundantBuiltin`)

### - [ ] Step 1: Write the failing test

Append to the end of `internal/engine/engine_test.go`:

```go
// Semantic duplicate detection: applics-redundant? returns true iff
// every applic on the unit has an output that matches what parent's
// defn produces on the same args. Gates on >=3 applics to avoid
// premature kills on sparse evidence.
func TestApplicsRedundantBuiltin(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	// Build a synthetic unit whose recorded applics match what Add would
	// produce. Args reference Number instance units; outputs are also
	// Number instance units whose `data` equals the expected sum.
	mkNum := func(name string, v int) {
		u := unit.New(name)
		u.Set("isA", []string{"Number", "Anything"})
		u.SetWorth(500)
		u.Set("data", v)
		eng.Store.Put(u)
	}
	mkNum("N2", 2)
	mkNum("N3", 3)
	mkNum("N5", 5)
	mkNum("N7", 7)
	mkNum("N8", 8)
	mkNum("N10", 10)

	// "FakeAdd" has 3 applics that match Add(a,b)=a+b exactly.
	fa := unit.New("FakeAdd")
	fa.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	fa.Set("creditors", []string{"TestSeed"})
	fa.Set("generalizations", []string{"Add"})
	fa.Set("applics", []map[string]any{
		{"args": []string{"N2", "N3"}, "output": "N5", "direct": true},
		{"args": []string{"N3", "N5"}, "output": "N8", "direct": true},
		{"args": []string{"N3", "N7"}, "output": "N10", "direct": true},
	})
	eng.Store.Put(fa)

	v, err := eng.VM.Execute(`"FakeAdd" "Add" applics-redundant?`)
	if err != nil {
		t.Fatalf("applics-redundant? FakeAdd Add: %v", err)
	}
	if !v.Truthy() {
		t.Errorf("FakeAdd vs Add: expected redundant=true")
	}

	// "DivergentAdd" has 3 applics but one output is wrong.
	da := unit.New("DivergentAdd")
	da.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	da.Set("creditors", []string{"TestSeed"})
	da.Set("generalizations", []string{"Add"})
	da.Set("applics", []map[string]any{
		{"args": []string{"N2", "N3"}, "output": "N5", "direct": true},
		{"args": []string{"N3", "N5"}, "output": "N8", "direct": true},
		{"args": []string{"N3", "N7"}, "output": "N2", "direct": true}, // wrong: 3+7 != 2
	})
	eng.Store.Put(da)

	v2, err := eng.VM.Execute(`"DivergentAdd" "Add" applics-redundant?`)
	if err != nil {
		t.Fatalf("applics-redundant? DivergentAdd Add: %v", err)
	}
	if v2.Truthy() {
		t.Errorf("DivergentAdd vs Add: expected redundant=false")
	}

	// Insufficient evidence: fewer than 3 applics → false.
	sparse := unit.New("SparseAdd")
	sparse.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	sparse.Set("creditors", []string{"TestSeed"})
	sparse.Set("generalizations", []string{"Add"})
	sparse.Set("applics", []map[string]any{
		{"args": []string{"N2", "N3"}, "output": "N5", "direct": true},
		{"args": []string{"N3", "N5"}, "output": "N8", "direct": true},
	})
	eng.Store.Put(sparse)

	v3, err := eng.VM.Execute(`"SparseAdd" "Add" applics-redundant?`)
	if err != nil {
		t.Fatalf("applics-redundant? SparseAdd Add: %v", err)
	}
	if v3.Truthy() {
		t.Errorf("SparseAdd vs Add: expected false (only 2 applics)")
	}

	// Missing target: unknown parent → false.
	v4, err := eng.VM.Execute(`"FakeAdd" "Nonexistent" applics-redundant?`)
	if err != nil {
		t.Fatalf("applics-redundant? FakeAdd Nonexistent: %v", err)
	}
	if v4.Truthy() {
		t.Errorf("FakeAdd vs Nonexistent: expected false (missing parent)")
	}
}
```

### - [ ] Step 2: Run test to verify it fails

```bash
cd /Users/chazu/dev/go/nous
go test ./internal/engine/ -run TestApplicsRedundantBuiltin -v
```

Expected: FAIL with `unknown word: applics-redundant?`.

### - [ ] Step 3: Implement the builtin

In `internal/dsl/builtins_math.go`, add registration in the `init()` function (after `builtins["compose-ops"]`):

```go
	builtins["applics-redundant?"] = bApplicsRedundant
```

At the end of the file, add:

```go
// applics-redundant?: ( unitName parentName -- bool )
// Returns true iff every applic on unitName has an output that matches
// what parentName's defn produces on the same args. Requires >=3 applics
// for meaningful evidence; returns false on sparse evidence, missing
// targets, or any mismatch.
func bApplicsRedundant(vm *VM) error {
	parentName := vm.pop().AsString()
	unitName := vm.pop().AsString()

	u := vm.Store.Get(unitName)
	parent := vm.Store.Get(parentName)
	if u == nil || parent == nil {
		vm.push(BoolVal(false))
		return nil
	}
	parentDefn := parent.GetString("defn")
	if parentDefn == "" {
		vm.push(BoolVal(false))
		return nil
	}

	applicsRaw, ok := u.Get("applics").([]map[string]any)
	if !ok || len(applicsRaw) < 3 {
		vm.push(BoolVal(false))
		return nil
	}

	for _, a := range applicsRaw {
		// Extract args as []string regardless of whether stored as []string
		// or []any (both storage paths occur — []string from record-applic,
		// []any from some test seeds).
		var args []string
		switch v := a["args"].(type) {
		case []string:
			args = v
		case []any:
			for _, x := range v {
				if s, ok := x.(string); ok {
					args = append(args, s)
				}
			}
		}
		outName, _ := a["output"].(string)
		if len(args) == 0 || outName == "" {
			vm.push(BoolVal(false))
			return nil
		}

		// Resolve args to their data values, run parent.defn, compare
		// against the recorded output unit's data.
		sub := NewVM(vm.Store, vm.Ag, vm.Rng)
		sub.Out = vm.Out
		for k, val := range vm.env {
			sub.env[k] = val
		}
		argsOK := true
		for _, argName := range args {
			argU := vm.Store.Get(argName)
			if argU == nil {
				argsOK = false
				break
			}
			data := argU.Get("data")
			if data == nil {
				argsOK = false
				break
			}
			sub.stack = append(sub.stack, anyToValue(data))
		}
		if !argsOK {
			vm.push(BoolVal(false))
			return nil
		}

		parentResult, err := subExecute(sub, parentDefn)
		if err != nil {
			vm.push(BoolVal(false))
			return nil
		}

		outU := vm.Store.Get(outName)
		if outU == nil {
			vm.push(BoolVal(false))
			return nil
		}
		outData := outU.Get("data")
		if outData == nil {
			vm.push(BoolVal(false))
			return nil
		}
		observed := anyToValue(outData)

		if !semanticValuesEqual(parentResult, observed) {
			vm.push(BoolVal(false))
			return nil
		}
	}

	vm.push(BoolVal(true))
	return nil
}

// semanticValuesEqual compares two Values with set-semantics for lists
// (order-insensitive, dedupe-insensitive for int lists) and strict
// equality otherwise. Used by commutativity sampling and applics-redundant?
// so Transpose-Add(a,b) == Add(b,a) returns true when outputs are the
// same set regardless of element order.
func semanticValuesEqual(a, b Value) bool {
	if a.Kind() == VList && b.Kind() == VList {
		// Normalize both through toIntSet (sort + dedupe). Works for our
		// current int-valued list domain; if non-int lists appear later,
		// this fallback would need broadening.
		as, bs := toIntSet(a), toIntSet(b)
		if len(as) != len(bs) {
			return false
		}
		for i := range as {
			if as[i] != bs[i] {
				return false
			}
		}
		return true
	}
	return a.Equal(b)
}
```

`toIntSet`, `anyToValue`, `subExecute`, `NewVM`, and the `Value.Equal` method all exist already in the `dsl` package. `Value.Kind()` is at `internal/dsl/value.go:41`.

### - [ ] Step 4: Run test to verify it passes

```bash
go test ./internal/engine/ -run TestApplicsRedundantBuiltin -v
```

Expected: PASS.

If the FakeAdd case fails (`expected redundant=true`), debug by temporarily logging inside `bApplicsRedundant`: `fmt.Fprintf(vm.Out, "applic %d: parent produced %v, observed %v\n", i, parentResult, observed)`. The most likely failure is a missing N2/N3/N5 lookup — ensure the test seeds them via `mkNum` before calling the builtin.

### - [ ] Step 5: Commit

```bash
git add internal/dsl/builtins_math.go internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: applics-redundant? builtin for semantic dup detection

New DSL builtin compares a unit's observed applic outputs against what
a parent op's defn would produce on the same args. Gates on >=3 applics
for meaningful evidence. Foundation for H-SemanticDup heuristic.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Extend `bTransposeOp` with commutativity sampling

**Files:**
- Modify: `internal/dsl/builtins_math.go` (extend `bTransposeOp` after existing gates, before unit creation)
- Modify: `internal/engine/engine_test.go` (append `TestTransposeOpSkipsCommutative`, `TestTransposeOpFallbackNoSamples`)

### - [ ] Step 1: Write the failing tests

Append to `internal/engine/engine_test.go`:

```go
// Part B: transpose-op skips creation when sampling proves commutativity.
// Add is commutative (a+b == b+a); Transpose-Add should not be created.
// SetDifference is non-commutative; Transpose-SetDifference should be
// created normally.
func TestTransposeOpSkipsCommutative(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	// Ensure Number has enough data-bearing instance units for sampling.
	// Seed 3 with known int data.
	for i, v := range []int{2, 3, 5} {
		name := fmt.Sprintf("Ncomm%d", i)
		u := unit.New(name)
		u.Set("isA", []string{"Number", "Anything"})
		u.SetWorth(500)
		u.Set("data", v)
		eng.Store.Put(u)
		num := eng.Store.Get("Number")
		ex := num.Get("examples")
		var exs []any
		if l, ok := ex.([]any); ok {
			exs = l
		}
		exs = append(exs, name)
		eng.Store.SetSlot("Number", "examples", exs)
	}

	// Commutative: transpose-op on Add → nil, unit not created.
	v, err := eng.VM.Execute(`"Add" transpose-op`)
	if err != nil {
		t.Fatalf("transpose-op Add: %v", err)
	}
	if !v.IsNil() {
		t.Errorf("transpose-op Add: expected nil (commutative), got %v", v)
	}
	if eng.Store.Has("Transpose-Add") {
		t.Error("Transpose-Add should not exist after commutativity detected")
	}

	// Non-commutative: transpose-op on SetDifference → succeeds.
	v2, err := eng.VM.Execute(`"SetDifference" transpose-op`)
	if err != nil {
		t.Fatalf("transpose-op SetDifference: %v", err)
	}
	if v2.AsString() != "Transpose-SetDifference" {
		t.Errorf("transpose-op SetDifference: expected Transpose-SetDifference, got %v", v2)
	}
}

// Part B fallback: when the domain type has fewer than 2 data-bearing
// examples, sampling can't conclude commutativity — fall back to
// creating the Transpose unconditionally.
func TestTransposeOpFallbackNoSamples(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	// Create a synthetic type with no examples and a synthetic BinaryOp
	// whose domain is that type.
	tt := unit.New("TestType")
	tt.Set("isA", []string{"Anything"})
	tt.SetWorth(500)
	eng.Store.Put(tt)

	synth := unit.New("SynthOp")
	synth.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	synth.SetWorth(500)
	synth.Set("domain", []string{"TestType", "TestType"})
	synth.Set("range", []string{"TestType"})
	synth.Set("defn", "+")
	synth.Set("creditors", []string{"TestSeed"})
	eng.Store.Put(synth)

	v, err := eng.VM.Execute(`"SynthOp" transpose-op`)
	if err != nil {
		t.Fatalf("transpose-op SynthOp: %v", err)
	}
	if v.AsString() != "Transpose-SynthOp" {
		t.Errorf("expected Transpose-SynthOp (fallback path), got %v", v)
	}
	if !eng.Store.Has("Transpose-SynthOp") {
		t.Error("Transpose-SynthOp not created in fallback path")
	}
}
```

If `fmt` is not already imported in `engine_test.go`, add it.

### - [ ] Step 2: Run tests to verify they fail

```bash
go test ./internal/engine/ -run "TestTransposeOpSkipsCommutative|TestTransposeOpFallbackNoSamples" -v
```

Expected:
- `TestTransposeOpSkipsCommutative` FAIL — Transpose-Add gets created because current `bTransposeOp` doesn't sample.
- `TestTransposeOpFallbackNoSamples` PASS — the current code already creates unconditionally; this test is documenting the preserved fallback behavior. If it unexpectedly fails, investigate why SynthOp isn't being accepted by the existing gates.

### - [ ] Step 3: Extend `bTransposeOp`

Find `bTransposeOp` in `internal/dsl/builtins_math.go`. Currently the function looks like:

```go
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

	// ... unit construction ...
}
```

Insert the commutativity check between the `newName` idempotency block and the unit construction. Add:

```go
	// Part B: commutativity sampling. When domain[0] == domain[1], sample
	// up to 3 (a,b) pairs of data-bearing examples from the domain type's
	// examples slot, run defn on (a,b) and (b,a). If all pairs agree, the
	// op is commutative on observed inputs — skip creating the Transpose.
	// Skip the check entirely if the domain has asymmetric types (transpose
	// is always semantically distinct there) or if fewer than 2 samples are
	// available (can't conclude anything from sparse evidence).
	if domain[0] == domain[1] {
		samples := drawDataBearingExamples(vm.Store, domain[0], 3)
		if len(samples) >= 2 {
			if commutativeOnSamples(vm, defn, samples) {
				vm.push(Nil())
				return nil
			}
		}
	}
```

At the end of the file, add the two helpers:

```go
// drawDataBearingExamples walks typeName.examples, resolves each entry
// to a unit, and collects up to n distinct `data` values as Values.
// Raw non-unit entries (e.g. int literals pre-dating H-Generate) are
// ignored. Used by bTransposeOp for commutativity sampling.
func drawDataBearingExamples(store *unit.Store, typeName string, n int) []Value {
	u := store.Get(typeName)
	if u == nil {
		return nil
	}
	ex := u.Get("examples")
	var list []any
	switch v := ex.(type) {
	case []any:
		list = v
	case []string:
		for _, s := range v {
			list = append(list, s)
		}
	default:
		return nil
	}

	var out []Value
	for _, item := range list {
		if len(out) >= n {
			break
		}
		name, ok := item.(string)
		if !ok {
			continue
		}
		iu := store.Get(name)
		if iu == nil {
			continue
		}
		data := iu.Get("data")
		if data == nil {
			continue
		}
		out = append(out, anyToValue(data))
	}
	return out
}

// commutativeOnSamples tests whether `defn` produces the same result on
// (a, b) and (b, a) for every pair drawn from samples. Uses semantic
// equality (set-equal for lists, == for primitives). Returns true iff
// all observed pairs commute.
func commutativeOnSamples(vm *VM, defn string, samples []Value) bool {
	pairs := 0
	for i := 0; i < len(samples); i++ {
		for j := i + 1; j < len(samples); j++ {
			if pairs >= 3 {
				return true
			}
			a, b := samples[i], samples[j]

			sub1 := NewVM(vm.Store, vm.Ag, vm.Rng)
			sub1.Out = vm.Out
			for k, val := range vm.env {
				sub1.env[k] = val
			}
			sub1.stack = append(sub1.stack, a, b)
			r1, err := subExecute(sub1, defn)
			if err != nil {
				return false
			}

			sub2 := NewVM(vm.Store, vm.Ag, vm.Rng)
			sub2.Out = vm.Out
			for k, val := range vm.env {
				sub2.env[k] = val
			}
			sub2.stack = append(sub2.stack, b, a)
			r2, err := subExecute(sub2, defn)
			if err != nil {
				return false
			}

			if !semanticValuesEqual(r1, r2) {
				return false
			}
			pairs++
		}
	}
	return pairs > 0
}
```

`semanticValuesEqual` was added in Task 1. `unit.Store`, `anyToValue`, `subExecute`, `NewVM` already exist.

### - [ ] Step 4: Run tests to verify they pass

```bash
go test ./internal/engine/ -run "TestTransposeOpSkipsCommutative|TestTransposeOpFallbackNoSamples" -v
```

Expected: both PASS.

Now run the full engine suite — some tests previously passed that assumed Transpose-Add would get created may now fail:

```bash
go test ./internal/engine/ 2>&1 | tail -10
```

If `TestTransposeOp` (Task 1 of the Transpose/Compose phase) still expects `Transpose-SetDifference` (non-commutative — should still be created): PASS. If it tests any commutative op and expects the Transpose to exist, investigate. The earlier test only exercises SetDifference and DivisorsOf; both should still work.

If `TestHTransposeFires` still expects `Transpose-SetDifference`: PASS (non-commutative target).

If `TestHComposeFires` breaks: unlikely since Compose doesn't go through the commutativity check. But verify.

If any test breaks, STOP and report which one and why before proceeding.

### - [ ] Step 5: Commit

```bash
git add internal/dsl/builtins_math.go internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: transpose-op commutativity sampling

Before creating Transpose-<op>, sample up to 3 pairs of data-bearing
examples from the domain type (when symmetric) and check whether the op
produces identical outputs on (a,b) and (b,a). If all pairs commute,
return nil — no Transpose needed. Falls back to creating the Transpose
when fewer than 2 samples are available.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: `H-SemanticDup` heuristic + test

**Files:**
- Modify: `domains/common/heuristics.cue` (append unit after H-Compose)
- Modify: `internal/engine/engine_test.go` (append `TestHSemanticDupKillsRedundant`)

### - [ ] Step 1: Write the failing test

Append to `internal/engine/engine_test.go`:

```go
// Part A: H-SemanticDup kills ops whose observed applics are fully
// reproduced by a generalization. Seed Transpose-Add-like applics that
// match Add's output and fire the heuristic; expect Transpose-Add to
// be killed and Add to survive.
func TestHSemanticDupKillsRedundant(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H-SemanticDup") {
		t.Fatal("H-SemanticDup heuristic not loaded from common/heuristics.cue")
	}

	// Seed data-bearing Number instances for arg/output resolution.
	mkNum := func(name string, v int) {
		u := unit.New(name)
		u.Set("isA", []string{"Number", "Anything"})
		u.SetWorth(500)
		u.Set("data", v)
		eng.Store.Put(u)
	}
	mkNum("Nd2", 2)
	mkNum("Nd3", 3)
	mkNum("Nd5", 5)
	mkNum("Nd7", 7)
	mkNum("Nd8", 8)
	mkNum("Nd10", 10)

	// Manually construct a Transpose-Add-like unit whose applics exactly
	// reproduce Add's behavior on 3 input pairs. (We skip H-Transpose here
	// because Part B would prevent creation — this test exercises Part A
	// on a manually-seeded unit.)
	ta := unit.New("FakeTranspose-Add")
	ta.Set("isA", []string{"BinaryOp", "Op", "MathOp", "Anything"})
	ta.SetWorth(500)
	ta.Set("domain", []string{"Number", "Number"})
	ta.Set("range", []string{"Number"})
	ta.Set("defn", "swap +")
	ta.Set("creditors", []string{"H-Transpose"})
	ta.Set("applics", []map[string]any{
		{"args": []string{"Nd2", "Nd3"}, "output": "Nd5", "direct": true},
		{"args": []string{"Nd3", "Nd5"}, "output": "Nd8", "direct": true},
		{"args": []string{"Nd3", "Nd7"}, "output": "Nd10", "direct": true},
	})
	eng.Store.Put(ta)
	eng.Store.SetSlot("FakeTranspose-Add", "generalizations", []string{"Add"})

	eng.fireUnitRule("H-SemanticDup", "FakeTranspose-Add")

	if eng.Store.Has("FakeTranspose-Add") {
		t.Error("FakeTranspose-Add should have been killed by H-SemanticDup")
	}
	if !eng.Store.Has("Add") {
		t.Error("Add (parent) should survive — H-SemanticDup must not touch it")
	}
}
```

### - [ ] Step 2: Run test to verify it fails

```bash
go test ./internal/engine/ -run TestHSemanticDupKillsRedundant -v
```

Expected: FAIL with `H-SemanticDup heuristic not loaded from common/heuristics.cue`.

### - [ ] Step 3: Add H-SemanticDup heuristic

Open `domains/common/heuristics.cue`. Append a new unit before the closing `]` (after H-Compose from Phase 5.6B):

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
			true "active" !
			"ArgU" @ "generalizations" get-slot
			each
				it "parent" !
				"active" @
				if
					"ArgU" @ "parent" @ applics-redundant?
					if
						"H-SemanticDup: " "ArgU" @ concat " redundant vs " concat "parent" @ concat " — killing" concat print
						"ArgU" @ kill-unit
						false "active" !
					then
				then
			end
			true "ArgU" @ "semDupChecked" set-slot
			"""#
	},
```

Match the indentation of the surrounding heuristic units (single tab per entry). If you're unsure about placement, put it right after the H-Compose unit (added in the Phase 5.6 A+B work).

### - [ ] Step 4: Run test to verify it passes

```bash
go test ./internal/engine/ -run TestHSemanticDupKillsRedundant -v
```

Expected: PASS.

Also run the full engine suite:

```bash
go test ./internal/engine/ 2>&1 | tail -10
```

Expected: all PASS.

### - [ ] Step 5: Commit

```bash
git add domains/common/heuristics.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: H-SemanticDup heuristic for post-hoc op duplicate detection

Fires on Ops with >=3 applics and non-seed creditors. Checks whether
any generalization reproduces the unit's observed applic outputs via
applics-redundant? builtin; kills the unit on match. Covers Transpose
variants that escaped creation-time commutativity sampling, plus any
Compose-* that collapses to its generalization's behavior.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Regression + doc updates

**Files:**
- Modify: `docs/eurisko-parity-phases.md` (append followup note to Phase 5.6)

### - [ ] Step 1: Run the full test suite

```bash
cd /Users/chazu/dev/go/nous
go test ./... 2>&1 | tail -15
```

Expected: all packages PASS. Report the tail verbatim. If anything fails, STOP and report.

### - [ ] Step 2: 100-cycle math smoke run

```bash
go build -o nous ./cmd/nous 2>&1 | tail -3
./nous run -domains-dir ./domains -domain math -cycles 100 -v 0 2>&1 | tee /tmp/smoke-semdup.txt > /dev/null

# Observational counts
echo "Transpose count: $(grep -c "^Transposed " /tmp/smoke-semdup.txt)"
echo "SemanticDup kills: $(grep -c "^H-SemanticDup:" /tmp/smoke-semdup.txt)"
echo "Transpose-* surviving: $(./nous ls -domains-dir ./domains -domain math 2>/dev/null | grep -c "^Transpose-" || echo 'n/a')"
tail -20 /tmp/smoke-semdup.txt
```

Note: `./nous run` and `./nous ls` flag shapes may differ from the above — verify with `./nous --help` / `./nous run --help` and adapt. If the subcommand is simply `./nous` (no `run`), drop the `run`. The prior phase's Task 5 used the same binary and flags `-domains-dir -domain -cycles -v 0`; match whatever was current then.

Expected observations:
- Far fewer `Transposed <op>` lines for commutative ops (Add, Multiply, SetUnion, SetIntersect should be absent from the Transposed log)
- Some `H-SemanticDup: ... killing` lines may appear if any Compose-* or fallback-created Transpose-* accumulates matching applics
- Non-commutative Transpose variants (Transpose-SetDifference, Transpose-GCD if GCD had asymmetric domains — actually GCD is commutative so it should NOT appear either) should still be created

Capture concrete numbers for the Task 4 commit message.

### - [ ] Step 3: Update phase tracking doc

Edit `docs/eurisko-parity-phases.md`. In the Phase 5.6 section, find the "A Transpose" paragraph (marked COMPLETE 2026-04-19) and append to its body:

```markdown
Followup (2026-04-19): commutativity sampling added to `transpose-op` — op.defn is run on (a,b) and (b,a) for up to 3 sample pairs drawn from the domain type's data-bearing examples. If all pairs agree, no Transpose is created. Plus new `H-SemanticDup` heuristic kills Transpose/Compose units whose observed applics are fully reproduced by any generalization. Spec: `docs/superpowers/specs/2026-04-19-semantic-duplicate-ops-design.md`. Closes the "H19 doesn't prune commutative Transposes" observation from the Phase 5.6 A+B smoke run.
```

No changes to the summary table are needed — this is a followup to an already-COMPLETE phase entry, not a new phase.

### - [ ] Step 4: Update memory index

Edit `/Users/chazu/.claude/projects/-Users-chazu-dev-go-nous/memory/project_nous_phases.md`. Update:
- Frontmatter `description` to reflect 2026-04-19 semantic-dup detection landed
- In Complete/Status section: note `H-SemanticDup` heuristic + `transpose-op` commutativity sampling as landed 2026-04-19
- "Next natural moves" list: unchanged (5.3+5.4 remains top candidate; this was a side-quest followup)

Keep the pudl ↔ nous integration note.

### - [ ] Step 5: Commit

```bash
cd /Users/chazu/dev/go/nous
git add docs/eurisko-parity-phases.md
git commit -m "$(cat <<'EOF'
docs: note semantic duplicate detection followup to Phase 5.6

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Completion Criteria

- [ ] `go test ./internal/engine/ -run "TestApplicsRedundantBuiltin|TestTransposeOpSkipsCommutative|TestTransposeOpFallbackNoSamples|TestHSemanticDupKillsRedundant" -v` passes
- [ ] `go test ./...` passes with no regressions
- [ ] 100-cycle math smoke run exits cleanly
- [ ] Commutative BinaryOps (Add, Multiply, SetUnion, SetIntersect) do NOT produce Transpose-* units in the smoke run
- [ ] Non-commutative BinaryOps (SetDifference) DO produce Transpose-* units
- [ ] Four commits on branch (three feature + one doc)

---

## Notes for the implementing engineer

- `Value.IsNil()` (`internal/dsl/value.go:43`) is the preferred Nil check in tests.
- `anyToValue` (`internal/dsl/builtins.go:1368`) converts Go values (int, float64, string, []any, []string) to DSL Values. Use it when stacking arg data onto a sub-VM.
- `subExecute` (`internal/dsl/builtins_math.go:574`) runs a DSL program on a sub-VM and returns the top-of-stack Value.
- `toIntSet` (elsewhere in `builtins_math.go`) normalizes an int-list Value to a sorted, deduplicated `[]int`. Used by `semanticValuesEqual` for list comparisons.
- `Store.SetSlot(name, slot, value)` triggers inverse maintenance; use it (not `u.Set`) for `generalizations` / `specializations`. Pass `[]string` (NOT `[]any`) for inverse-triggered slots — this was established in Phase 5.6.
- The applic storage format in `record-applic` (`internal/dsl/builtins.go:1149`) stores args as `[]string`. Test seeds may store as `[]any`. The `bApplicsRedundant` implementation handles both via type-switch.
- `kill-unit` DSL builtin (`internal/dsl/builtins.go:57`) removes a unit from the store.
- If test-run output differs substantially from what the plan predicts, stop and report rather than pressing on.
