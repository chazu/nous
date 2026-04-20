# Phase 5.1 OSet Type and Operations — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an Ordered Set (`OSet`) type with order-preserving operations to the math domain, enabling heuristics to distinguish order-preserving from order-insensitive behavior.

**Architecture:** Five new DSL builtins (`oset-union`, `oset-intersect`, `oset-insert`, `oset-delete`, `oset-equal?`) that linear-scan inputs without sorting. One new type unit (`OSet` ⊂ `Set`), two instance units (one ascending, one descending — the non-sorted one is load-bearing for observability), five op units parallel to the existing `SetUnion`/`SetIntersect`/`SetEqual` shape.

**Tech Stack:** Go, CUE (for domain data), internal Forth-like DSL VM, EURISKO-style unit+heuristic engine.

**Spec:** `docs/superpowers/specs/2026-04-20-oset-type-and-ops-design.md`

---

## File Structure

- `internal/dsl/builtins_math.go` — add `oset-*` builtins + one helper `toIntListPreserve` (dedupe, preserve order). Register in `init()`. ~80 lines added.
- `internal/dsl/builtins_math_test.go` — 1 new test function `TestOSetOps` covering order-preservation, dedupe, OSetEqual vs SetEqual divergence.
- `domains/math/types.cue` — add `OSet` unit; append `"OSet"` to `Set.specializations`.
- `domains/math/sets.cue` — add `OSetOfNumbers`, `OSetOfPrimesDesc` instance units.
- `domains/math/operations.cue` — add `OSetUnion`, `OSetIntersect`, `OSetInsert`, `OSetDelete`, `OSetEqual` op units.
- `internal/engine/engine_test.go` — add `TestOSetUnionPreservesOrder` smoke test.

---

### Task 1: OSet DSL builtins

**Files:**
- Modify: `internal/dsl/builtins_math.go` (register + implement at end of file)
- Modify: `internal/dsl/builtins_math_test.go` (add TestOSetOps)

- [ ] **Step 1: Write the failing test**

Add to `internal/dsl/builtins_math_test.go`:

```go
func TestOSetOps(t *testing.T) {
	vm := testVM(t)

	// Lists are constructed via `list-of` — push N elements, then N, then call.
	cases := []struct {
		name string
		prog string
		want []int
	}{
		// Order preservation: a then b's novelty in b's order
		{"union-preserves-order", `3 1 2 3 list-of  4 2 5 3 list-of  oset-union`, []int{3, 1, 2, 4, 5}},
		// Intersect preserves a's order
		{"intersect-preserves-a-order", `3 1 2 4 4 list-of  4 2 2 list-of  oset-intersect`, []int{2, 4}},
		// Insert: present → no-op
		{"insert-present-noop", `3 1 2 3 list-of  1 oset-insert`, []int{3, 1, 2}},
		// Insert: absent → append at end
		{"insert-absent-appends", `3 1 2 3 list-of  7 oset-insert`, []int{3, 1, 2, 7}},
		// Delete: preserves surrounding order
		{"delete-preserves-order", `3 1 2 4 4 list-of  1 oset-delete`, []int{3, 2, 4}},
	}
	for _, tc := range cases {
		v, err := vm.Execute(tc.prog)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		got := v.AsList()
		if len(got) != len(tc.want) {
			t.Errorf("%s: want %v got %v", tc.name, tc.want, got)
			continue
		}
		for i, w := range tc.want {
			if got[i].AsInt() != w {
				t.Errorf("%s: at %d want %d got %d (full: %v)", tc.name, i, w, got[i].AsInt(), got)
				break
			}
		}
	}

	// Divergence: oset-equal? is order-sensitive, set-equal? is not
	v, err := vm.Execute(`1 2 2 list-of  2 1 2 list-of  oset-equal?`)
	if err != nil || v.AsBool() {
		t.Errorf("oset-equal? [1 2] [2 1]: want false, got %v (err=%v)", v, err)
	}
	v, err = vm.Execute(`1 2 2 list-of  2 1 2 list-of  set-equal?`)
	if err != nil || !v.AsBool() {
		t.Errorf("set-equal? [1 2] [2 1]: want true, got %v (err=%v)", v, err)
	}
	// Same order → equal
	v, err = vm.Execute(`1 2 3 3 list-of  1 2 3 3 list-of  oset-equal?`)
	if err != nil || !v.AsBool() {
		t.Errorf("oset-equal? identical: want true, got %v (err=%v)", v, err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -run TestOSetOps -v`
Expected: FAIL with errors like `oset-union: unknown word` or similar.

- [ ] **Step 3: Implement the builtins**

In `internal/dsl/builtins_math.go`, inside `init()` just after the existing set registrations (around line 64, after `builtins["set-empty?"] = ...`), add:

```go
	// OSet operations — order-preserving, duplicate-rejecting
	builtins["oset-union"] = bOSetUnion
	builtins["oset-intersect"] = bOSetIntersect
	builtins["oset-insert"] = bOSetInsert
	builtins["oset-delete"] = bOSetDelete
	builtins["oset-equal?"] = bOSetEqual
```

At the end of the file (after the existing `bMakeSet` function ends around line 345), append:

```go
// OSet operations — ordered sets: unique elements, order-preserving.
// Internal representation is the same []Value as sets/lists; the distinction
// is behavioral (these builtins don't sort and reject duplicates by O(n)
// linear scan rather than by map-based dedupe).

// toIntListPreserve returns a's element ints in original order, with
// duplicates removed (first occurrence wins).
func toIntListPreserve(v Value) []int {
	list := v.AsList()
	seen := make(map[int]bool, len(list))
	out := make([]int, 0, len(list))
	for _, el := range list {
		n := el.AsInt()
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

func intListToValue(s []int) Value {
	vals := make([]Value, len(s))
	for i, n := range s {
		vals[i] = IntVal(n)
	}
	return ListVal(vals)
}

func bOSetUnion(vm *VM) error {
	b := toIntListPreserve(vm.pop())
	a := toIntListPreserve(vm.pop())
	inA := make(map[int]bool, len(a))
	for _, n := range a {
		inA[n] = true
	}
	result := append([]int(nil), a...)
	for _, n := range b {
		if !inA[n] {
			result = append(result, n)
			inA[n] = true
		}
	}
	vm.push(intListToValue(result))
	return nil
}

func bOSetIntersect(vm *VM) error {
	b := toIntListPreserve(vm.pop())
	a := toIntListPreserve(vm.pop())
	inB := make(map[int]bool, len(b))
	for _, n := range b {
		inB[n] = true
	}
	var result []int
	for _, n := range a {
		if inB[n] {
			result = append(result, n)
		}
	}
	vm.push(intListToValue(result))
	return nil
}

func bOSetInsert(vm *VM) error {
	x := vm.pop().AsInt()
	a := toIntListPreserve(vm.pop())
	for _, n := range a {
		if n == x {
			vm.push(intListToValue(a))
			return nil
		}
	}
	vm.push(intListToValue(append(a, x)))
	return nil
}

func bOSetDelete(vm *VM) error {
	x := vm.pop().AsInt()
	a := toIntListPreserve(vm.pop())
	result := make([]int, 0, len(a))
	for _, n := range a {
		if n != x {
			result = append(result, n)
		}
	}
	vm.push(intListToValue(result))
	return nil
}

func bOSetEqual(vm *VM) error {
	b := toIntListPreserve(vm.pop())
	a := toIntListPreserve(vm.pop())
	if len(a) != len(b) {
		vm.push(BoolVal(false))
		return nil
	}
	for i := range a {
		if a[i] != b[i] {
			vm.push(BoolVal(false))
			return nil
		}
	}
	vm.push(BoolVal(true))
	return nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -run TestOSetOps -v`
Expected: PASS.

Also run the full dsl test suite to catch regressions:

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -v`
Expected: all PASS (existing `TestSetOps`, `TestNumberPredicates`, etc. unaffected).

- [ ] **Step 5: Commit**

```bash
git add internal/dsl/builtins_math.go internal/dsl/builtins_math_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.1 — oset-* DSL builtins

Order-preserving, duplicate-rejecting variants of set operations.
Distinguishes OSetUnion from SetUnion behaviorally — required for
H-SemanticDup to see meaningful differences between the two.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: OSet type unit

**Files:**
- Modify: `domains/math/types.cue`
- Modify: `internal/engine/engine_test.go` (verify domain load)

- [ ] **Step 1: Write the failing test**

Add to `internal/engine/engine_test.go` (at end of file):

```go
// TestOSetTypeUnitLoads verifies the OSet type unit is loaded from CUE
// with the correct isA hierarchy (subtype of Set).
func TestOSetTypeUnitLoads(t *testing.T) {
	eng, _ := testEngine(t)
	oset := eng.Store.Get("OSet")
	if oset == nil {
		t.Fatal("OSet unit not loaded from domain")
	}
	isA, _ := oset.Get("isA").([]any)
	got := make(map[string]bool, len(isA))
	for _, v := range isA {
		if s, ok := v.(string); ok {
			got[s] = true
		}
	}
	for _, want := range []string{"Set", "Structure", "MathObj", "Anything"} {
		if !got[want] {
			t.Errorf("OSet.isA missing %q; got %v", want, isA)
		}
	}

	// Set should list OSet as a specialization
	set := eng.Store.Get("Set")
	specs, _ := set.Get("specializations").([]any)
	foundOSet := false
	for _, s := range specs {
		if s == "OSet" {
			foundOSet = true
			break
		}
	}
	if !foundOSet {
		t.Errorf("Set.specializations missing OSet; got %v", specs)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestOSetTypeUnitLoads -v`
Expected: FAIL with `OSet unit not loaded from domain`.

- [ ] **Step 3: Add the OSet type unit in CUE**

In `domains/math/types.cue`, append a new entry inside `units: [...]` (before the closing `]` on line 36):

```cue
	{
		name:    "OSet"
		worth:   600
		isA: ["Set", "Structure", "MathObj", "Anything"]
		english: "An ordered collection with no duplicate elements"
		specializations: ["OSetOfNumbers", "OSetOfPrimesDesc"]
		defn: #"""
			is-list?
			"""#
	},
```

Also modify the existing `Set` unit (lines 8–17) to include `"OSet"` in its `specializations`:

Change:
```cue
		specializations: ["EmptySet", "SetOfNumbers", "SetOfPrimes", "SetOfEvens"]
```
to:
```cue
		specializations: ["EmptySet", "SetOfNumbers", "SetOfPrimes", "SetOfEvens", "OSet"]
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestOSetTypeUnitLoads -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add domains/math/types.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.1 — OSet type unit (Set specialization)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: OSet instance units

**Files:**
- Modify: `domains/math/sets.cue`
- Modify: `internal/engine/engine_test.go` (verify data)

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestOSetInstanceUnitsLoad verifies OSetOfNumbers and OSetOfPrimesDesc
// are present with correct data. OSetOfPrimesDesc's descending order is
// load-bearing — it makes order preservation observable in seed data.
func TestOSetInstanceUnitsLoad(t *testing.T) {
	eng, _ := testEngine(t)

	nums := eng.Store.Get("OSetOfNumbers")
	if nums == nil {
		t.Fatal("OSetOfNumbers not loaded")
	}
	data, _ := nums.Get("data").([]any)
	if len(data) != 20 {
		t.Errorf("OSetOfNumbers.data: want 20 elements, got %d", len(data))
	}

	primes := eng.Store.Get("OSetOfPrimesDesc")
	if primes == nil {
		t.Fatal("OSetOfPrimesDesc not loaded")
	}
	pdata, _ := primes.Get("data").([]any)
	// Expect descending order: first element > last element
	if len(pdata) < 2 {
		t.Fatalf("OSetOfPrimesDesc.data too short: %v", pdata)
	}
	first, _ := pdata[0].(int)
	last, _ := pdata[len(pdata)-1].(int)
	if first <= last {
		t.Errorf("OSetOfPrimesDesc not descending: first=%d last=%d", first, last)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestOSetInstanceUnitsLoad -v`
Expected: FAIL with `OSetOfNumbers not loaded`.

- [ ] **Step 3: Add the instance units in CUE**

In `domains/math/sets.cue`, append two new entries inside the `units: [...]` array (before the closing `]`):

```cue
	{
		name:    "OSetOfNumbers"
		worth:   500
		isA: ["OSet", "Set", "Structure", "MathObj", "Anything"]
		english: "The integers from 1 to 20 in ascending order"
		data: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
	},
	{
		name:    "OSetOfPrimesDesc"
		worth:   500
		isA: ["OSet", "Set", "Structure", "MathObj", "Anything"]
		english: "Primes under 20 in descending order"
		data: [19, 17, 13, 11, 7, 5, 3, 2]
	},
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestOSetInstanceUnitsLoad -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add domains/math/sets.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.1 — OSet instance units

OSetOfNumbers (ascending) and OSetOfPrimesDesc (descending) seed data
makes order preservation observable in first-cycle runs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: OSet operation units

**Files:**
- Modify: `domains/math/operations.cue`
- Modify: `internal/engine/engine_test.go` (verify ops load)

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestOSetOperationUnitsLoad verifies the five OSet op units are loaded
// with correct domain/range and defn hooks to the new DSL builtins.
func TestOSetOperationUnitsLoad(t *testing.T) {
	eng, _ := testEngine(t)
	wantOps := map[string]struct {
		domain []string
		rangeT []string
		defn   string
	}{
		"OSetUnion":     {[]string{"OSet", "OSet"}, []string{"OSet"}, "oset-union"},
		"OSetIntersect": {[]string{"OSet", "OSet"}, []string{"OSet"}, "oset-intersect"},
		"OSetInsert":    {[]string{"OSet", "Anything"}, []string{"OSet"}, "oset-insert"},
		"OSetDelete":    {[]string{"OSet", "Anything"}, []string{"OSet"}, "oset-delete"},
		"OSetEqual":     {[]string{"OSet", "OSet"}, []string{"TruthValue"}, "oset-equal?"},
	}
	for name, want := range wantOps {
		u := eng.Store.Get(name)
		if u == nil {
			t.Errorf("%s not loaded", name)
			continue
		}
		dom, _ := u.Get("domain").([]any)
		if len(dom) != len(want.domain) {
			t.Errorf("%s.domain: want %v got %v", name, want.domain, dom)
		} else {
			for i, d := range want.domain {
				if dom[i] != d {
					t.Errorf("%s.domain[%d]: want %q got %v", name, i, d, dom[i])
				}
			}
		}
		rng, _ := u.Get("range").([]any)
		if len(rng) != len(want.rangeT) || rng[0] != want.rangeT[0] {
			t.Errorf("%s.range: want %v got %v", name, want.rangeT, rng)
		}
		defn, _ := u.Get("defn").(string)
		if !strings.Contains(defn, want.defn) {
			t.Errorf("%s.defn: want contains %q, got %q", name, want.defn, defn)
		}
	}
}
```

(Note: `strings` is already imported in this file — no import changes needed.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestOSetOperationUnitsLoad -v`
Expected: FAIL with `OSetUnion not loaded` (etc).

- [ ] **Step 3: Add the op units in CUE**

In `domains/math/operations.cue`, append inside the `units: [...]` array:

```cue
	{
		name:    "OSetUnion"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["OSet", "OSet"]
		range: ["OSet"]
		english: "Combine two ordered sets, preserving first's order then appending second's novel elements"
		defn:    #"""
			oset-union
			"""#
		examples: [
			{args: [3, 1, 2], args2: [4, 2, 5], result: [3, 1, 2, 4, 5]},
			{args: [19, 17], args2: [2, 17], result: [19, 17, 2]},
		]
	},
	{
		name:    "OSetIntersect"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["OSet", "OSet"]
		range: ["OSet"]
		english: "Elements common to both ordered sets, in first's order"
		defn:    #"""
			oset-intersect
			"""#
		examples: [
			{args: [3, 1, 2, 4], args2: [4, 2], result: [2, 4]},
			{args: [19, 17, 13], args2: [13, 19], result: [19, 13]},
		]
	},
	{
		name:    "OSetInsert"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["OSet", "Anything"]
		range: ["OSet"]
		english: "Append element to ordered set if not already present"
		defn:    #"""
			oset-insert
			"""#
		examples: [
			{args: [3, 1, 2], args2: 7, result: [3, 1, 2, 7]},
			{args: [3, 1, 2], args2: 1, result: [3, 1, 2]},
		]
	},
	{
		name:    "OSetDelete"
		worth:   500
		isA: ["BinaryOp", "Op", "MathOp", "Anything"]
		domain: ["OSet", "Anything"]
		range: ["OSet"]
		english: "Remove element from ordered set, preserving remaining order"
		defn:    #"""
			oset-delete
			"""#
		examples: [
			{args: [3, 1, 2, 4], args2: 1, result: [3, 2, 4]},
			{args: [19, 17, 13], args2: 99, result: [19, 17, 13]},
		]
	},
	{
		name:    "OSetEqual"
		worth:   500
		isA: ["BinaryPred", "Pred", "Op", "MathOp", "Anything"]
		domain: ["OSet", "OSet"]
		range: ["TruthValue"]
		english: "True iff two ordered sets contain the same elements in the same order"
		defn:    #"""
			oset-equal?
			"""#
		examples: [
			{args: [1, 2, 3], args2: [1, 2, 3], result: true},
			{args: [1, 2], args2: [2, 1], result: false},
		]
	},
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestOSetOperationUnitsLoad -v`
Expected: PASS.

Also run the full engine test suite to catch regressions:

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add domains/math/operations.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.1 — OSet operation units

OSetUnion, OSetIntersect, OSetInsert, OSetDelete, OSetEqual as BinaryOp /
BinaryPred units with seeded order-divergent examples.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: End-to-end smoke test — OSetUnion preserves order through H-RunOnExamples

**Files:**
- Modify: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestOSetUnionPreservesOrderViaEngine runs the engine long enough for
// H-RunOnExamples to apply OSetUnion to its seed data and asserts the
// recorded output preserves left-argument order. Regression guard: if a
// refactor accidentally routes OSet ops through set-union canonicalization,
// this test fails.
func TestOSetUnionPreservesOrderViaEngine(t *testing.T) {
	eng, _ := testEngine(t)
	eng.MaxCycles = 80
	eng.Verbosity = 0

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	op := eng.Store.Get("OSetUnion")
	if op == nil {
		t.Fatal("OSetUnion missing after run")
	}
	applics, _ := op.Get("applics").([]map[string]any)
	if len(applics) == 0 {
		t.Fatalf("OSetUnion recorded no applics in %d cycles — H-RunOnExamples never fired on it", eng.MaxCycles)
	}

	// Look for an applic where left arg starts with a non-ascending head
	// (OSetOfPrimesDesc → [19, 17, ...]) and verify output begins the same.
	foundOrderEvidence := false
	for _, ap := range applics {
		args, _ := ap["args"].([]string)
		if len(args) != 2 {
			continue
		}
		if args[0] != "OSetOfPrimesDesc" {
			continue
		}
		// Resolve the output unit's data
		outName, _ := ap["output"].(string)
		if outName == "" {
			continue
		}
		outU := eng.Store.Get(outName)
		if outU == nil {
			continue
		}
		data, _ := outU.Get("data").([]any)
		if len(data) < 2 {
			continue
		}
		first, _ := data[0].(int)
		if first == 19 {
			foundOrderEvidence = true
			break
		}
	}
	if !foundOrderEvidence {
		t.Errorf("No OSetUnion applic with OSetOfPrimesDesc as left arg produced an output starting with 19; order preservation not verified. Applics: %v", applics)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails or passes**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestOSetUnionPreservesOrderViaEngine -v`

This test may PASS immediately if all prior tasks are wired correctly (the pipeline already knows how to apply ops to their domain instances via H-RunOnExamples). If it fails, the most likely cause is that H-RunOnExamples doesn't pick up OSetOfPrimesDesc as a left-arg candidate. Investigate before proceeding.

**If the test fails with "OSetUnion recorded no applics":** this indicates H-RunOnExamples didn't find suitable input pairs. Increase `eng.MaxCycles` to 200 and re-run. If still no applics, the issue is upstream — H-RunOnExamples' candidate-selection for OSet-domain ops needs debugging. Flag this and stop; do not hack around it.

**If the test passes:** order preservation is confirmed end-to-end.

- [ ] **Step 3: Run the full test suite**

Run: `cd /Users/chazu/dev/go/nous && go test ./...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
test: Phase 5.1 — OSetUnion order-preservation smoke test

Regression guard: if a refactor routes OSet ops through set-union
canonicalization, this test fails by observing OSetOfPrimesDesc's
descending order is lost in OSetUnion output.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Update plan doc to mark 5.1 COMPLETE

**Files:**
- Modify: `docs/eurisko-parity-phases.md`

- [ ] **Step 1: Update Phase 5.1 section**

In `docs/eurisko-parity-phases.md`, change the Phase 5.1 heading and body (around lines 169–171):

```markdown
**5.1: OSet type and operations** -- COMPLETE (2026-04-20)

`OSet` added as `Set` specialization in `domains/math/types.cue`. Five op units (OSetUnion, OSetIntersect, OSetInsert, OSetDelete, OSetEqual) with corresponding order-preserving DSL builtins. Seed instances `OSetOfNumbers` (ascending) and `OSetOfPrimesDesc` (descending); the descending seed makes order preservation observable to heuristics. Engine smoke test (`TestOSetUnionPreservesOrderViaEngine`) guards against silent regression to canonicalizing set-*. OSetDifference and ReverseOSet deferred.
```

Also update the Phase 5 row in the summary table (around line 320):

Change:
```markdown
| 5 | Type hierarchy + operations | 12 | PARTIAL (5.2, 5.6 A/B/C.1/C.2, 5.9, 5.10, 5.11 numeric-preds + H27/H28) |
```
to:
```markdown
| 5 | Type hierarchy + operations | 12 | PARTIAL (5.1, 5.2, 5.6 A/B/C.1/C.2, 5.9, 5.10, 5.11 numeric-preds + H27/H28) |
```

- [ ] **Step 2: Commit**

```bash
git add docs/eurisko-parity-phases.md
git commit -m "$(cat <<'EOF'
docs: mark Phase 5.1 (OSet) COMPLETE

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Post-Implementation

After all tasks complete:

1. Run `go test ./...` from repo root — all green.
2. Update the auto-memory index entry for `project_nous_phases.md` to reflect 5.1 complete.
3. Next up per the plan: Phases 5.3 (projections) + 5.4 (structure classification) — separate brainstorming + spec + plan cycle.
