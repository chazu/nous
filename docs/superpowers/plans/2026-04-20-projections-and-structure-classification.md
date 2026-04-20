# Phase 5.3 + 5.4 Projections and Structure Classification — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add six structure-classification marker categories to the type hierarchy and six projection operation units that use those categories as domain restrictions.

**Architecture:** All changes land in CUE data + one new DSL builtin (`but-last`). `store.IsA` already walks the isA chain, so classification is queryable without new engine code. Projections compose existing builtins (`first`, `last`, `rest`, `but-last`). Tests cover DSL, CUE loading, chain propagation, and engine end-to-end.

**Tech Stack:** Go, CUE, internal Forth-like DSL VM, EURISKO-style unit engine.

**Spec:** `docs/superpowers/specs/2026-04-20-projections-and-structure-classification-design.md`

---

## File Structure

- `internal/dsl/builtins_math.go` — register and implement `but-last` builtin.
- `internal/dsl/builtins_math_test.go` — `TestButLast`.
- `domains/math/types.cue` — add six classification category units; update isA of existing `Set`, `List`, `Bag`, `OSet`.
- `domains/math/sets.cue` — update isA of `EmptySet`, `SetOfNumbers`, `SetOfPrimes`, `SetOfEvens`, `SetOfOdds`, `OSetOfNumbers`, `OSetOfPrimesDesc`.
- `domains/math/operations.cue` — add six projection op units (Proj1, Proj2, FirstEle, LastEle, AllButFirst, AllButLast).
- `internal/engine/engine_test.go` — four new engine tests (classification categories load, isA propagation, projection units load, FirstEle smoke test).
- `docs/eurisko-parity-phases.md` — mark phases 5.3 + 5.4 COMPLETE.

---

### Task 1: `but-last` DSL builtin

**Files:**
- Modify: `internal/dsl/builtins_math.go`
- Modify: `internal/dsl/builtins_math_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/dsl/builtins_math_test.go`:

```go
func TestButLast(t *testing.T) {
	vm := testVM(t)

	cases := []struct {
		name string
		prog string
		want []int
	}{
		{"basic", `3 1 2 3 list-of but-last`, []int{3, 1}},
		{"two-element", `1 2 2 list-of but-last`, []int{1}},
		{"single-element", `1 1 list-of but-last`, []int{}},
	}
	for _, tc := range cases {
		v, err := vm.Execute(tc.prog)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		got := v.AsList()
		if len(got) != len(tc.want) {
			t.Errorf("%s: len want %d got %d (%v)", tc.name, len(tc.want), len(got), got)
			continue
		}
		for i, w := range tc.want {
			if got[i].AsInt() != w {
				t.Errorf("%s: at %d want %d got %d", tc.name, i, w, got[i].AsInt())
			}
		}
	}

	// Empty input → empty output, no error.
	v, err := vm.Execute(`0 list-of but-last`)
	if err != nil {
		t.Errorf("empty: %v", err)
	}
	if len(v.AsList()) != 0 {
		t.Errorf("empty: want [] got %v", v.AsList())
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -run TestButLast -v`
Expected: FAIL with unknown word `but-last`.

- [ ] **Step 3: Register and implement**

In `internal/dsl/builtins_math.go`, locate the line `builtins["reverse"] = bReverse` (around line 70) and add on a nearby line:

```go
	builtins["but-last"] = bButLast
```

Then append to the file (at the end, or beside `bLast` / `bFirst`):

```go
func bButLast(vm *VM) error {
	list := vm.pop().AsList()
	if len(list) == 0 {
		vm.push(ListVal(nil))
		return nil
	}
	vm.push(ListVal(list[:len(list)-1]))
	return nil
}
```

- [ ] **Step 4: Confirm pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -run TestButLast -v`
Expected: PASS.

Regression: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/dsl/builtins_math.go internal/dsl/builtins_math_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.3 — but-last DSL builtin

One-pass drop-last-element operator. Enables AllButLast projection
without the triple-pass reverse-rest-reverse composition.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Classification category units

**Files:**
- Modify: `domains/math/types.cue`
- Modify: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestStructureClassificationCategoriesLoad verifies the six classification
// marker categories are loaded from CUE with correct isA chains.
func TestStructureClassificationCategoriesLoad(t *testing.T) {
	eng, _ := testEngine(t)

	wantCats := map[string]struct {
		worth int
		specs []string
	}{
		"OrdStruc":       {500, []string{"OSet", "List"}},
		"UnOrdStruc":     {500, []string{"Set", "Bag"}},
		"MultEleStruc":   {500, []string{"List", "Bag"}},
		"NoMultEleStruc": {500, []string{"Set", "OSet"}},
		"EmptyStruc":     {400, []string{"EmptySet"}},
		"NonEmptyStruc":  {400, nil}, // checked loosely — instances are many
	}
	for name, want := range wantCats {
		u := eng.Store.Get(name)
		if u == nil {
			t.Errorf("%s category not loaded", name)
			continue
		}
		isA := u.GetStrings("isA")
		isAMap := make(map[string]bool, len(isA))
		for _, s := range isA {
			isAMap[s] = true
		}
		for _, parent := range []string{"Structure", "MathObj", "Anything"} {
			if !isAMap[parent] {
				t.Errorf("%s.isA missing %q; got %v", name, parent, isA)
			}
		}
		if want.specs != nil {
			specs := u.GetStrings("specializations")
			specMap := make(map[string]bool, len(specs))
			for _, s := range specs {
				specMap[s] = true
			}
			for _, s := range want.specs {
				if !specMap[s] {
					t.Errorf("%s.specializations missing %q; got %v", name, s, specs)
				}
			}
		}
	}
}
```

- [ ] **Step 2: Confirm failure**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestStructureClassificationCategoriesLoad -v`
Expected: FAIL, categories not loaded.

- [ ] **Step 3: Add the six category units in CUE**

In `domains/math/types.cue`, append inside the `units: [...]` list (before the closing `]`):

```cue
	{
		name:    "OrdStruc"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure whose elements have a definite order"
		specializations: ["OSet", "List"]
	},
	{
		name:    "UnOrdStruc"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure whose elements have no definite order"
		specializations: ["Set", "Bag"]
	},
	{
		name:    "MultEleStruc"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure that may contain duplicate elements"
		specializations: ["List", "Bag"]
	},
	{
		name:    "NoMultEleStruc"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure that rejects duplicate elements"
		specializations: ["Set", "OSet"]
	},
	{
		name:    "EmptyStruc"
		worth:   400
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure containing no elements"
		specializations: ["EmptySet"]
	},
	{
		name:    "NonEmptyStruc"
		worth:   400
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure containing at least one element"
		specializations: ["SetOfNumbers", "SetOfPrimes", "SetOfEvens", "SetOfOdds", "OSetOfNumbers", "OSetOfPrimesDesc"]
	},
```

- [ ] **Step 4: Confirm pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestStructureClassificationCategoriesLoad -v`
Expected: PASS.

Regression: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add domains/math/types.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.4 — structure classification category units

Six marker categories: OrdStruc, UnOrdStruc, MultEleStruc, NoMultEleStruc,
EmptyStruc, NonEmptyStruc. No defn — pure classifications, queryable via
store.IsA chain walks.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Tag existing types and instances with classifications

**Files:**
- Modify: `domains/math/types.cue` (Set, List, Bag, OSet)
- Modify: `domains/math/sets.cue` (EmptySet and non-empty instances)
- Modify: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestStructureClassificationTagsPropagate verifies that classification
// parent categories flow via store.IsA chain walks. Each type and instance
// should participate in the correct classification dimensions.
func TestStructureClassificationTagsPropagate(t *testing.T) {
	eng, _ := testEngine(t)

	wantTrue := []struct{ unit, cat string }{
		{"Set", "UnOrdStruc"},
		{"Set", "NoMultEleStruc"},
		{"OSet", "OrdStruc"},
		{"OSet", "NoMultEleStruc"},
		{"List", "OrdStruc"},
		{"List", "MultEleStruc"},
		{"Bag", "UnOrdStruc"},
		{"Bag", "MultEleStruc"},
		{"EmptySet", "EmptyStruc"},
		{"SetOfNumbers", "NonEmptyStruc"},
		{"OSetOfPrimesDesc", "NonEmptyStruc"},
	}
	for _, tc := range wantTrue {
		if !eng.Store.IsA(tc.unit, tc.cat) {
			t.Errorf("IsA(%q, %q): want true, got false", tc.unit, tc.cat)
		}
	}

	wantFalse := []struct{ unit, cat string }{
		{"Set", "OrdStruc"},
		{"Set", "MultEleStruc"},
		{"OSet", "UnOrdStruc"},
		{"OSet", "MultEleStruc"},
		{"EmptySet", "NonEmptyStruc"},
	}
	for _, tc := range wantFalse {
		if eng.Store.IsA(tc.unit, tc.cat) {
			t.Errorf("IsA(%q, %q): want false, got true", tc.unit, tc.cat)
		}
	}
}
```

- [ ] **Step 2: Confirm failure**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestStructureClassificationTagsPropagate -v`
Expected: FAIL — tags not yet added.

- [ ] **Step 3: Update `domains/math/types.cue`**

For each of the four type units (Set, List, Bag, OSet), prepend the classification tags at the start of the existing `isA` list (keeping existing entries intact):

- `Set`: change
  ```cue
  isA: ["Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["UnOrdStruc", "NoMultEleStruc", "Structure", "MathObj", "Anything"]
  ```

- `List`: change
  ```cue
  isA: ["Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["OrdStruc", "MultEleStruc", "Structure", "MathObj", "Anything"]
  ```

- `Bag`: change
  ```cue
  isA: ["Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["UnOrdStruc", "MultEleStruc", "Structure", "MathObj", "Anything"]
  ```

- `OSet`: change
  ```cue
  isA: ["Set", "Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["OrdStruc", "NoMultEleStruc", "Set", "Structure", "MathObj", "Anything"]
  ```

- [ ] **Step 4: Update `domains/math/sets.cue`**

- `EmptySet`: change
  ```cue
  isA: ["Set", "Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["EmptyStruc", "Set", "Structure", "MathObj", "Anything"]
  ```

- `SetOfNumbers`: change
  ```cue
  isA: ["Set", "Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["NonEmptyStruc", "Set", "Structure", "MathObj", "Anything"]
  ```

- `SetOfPrimes`: change
  ```cue
  isA: ["Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["NonEmptyStruc", "Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
  ```

- `SetOfEvens`: change
  ```cue
  isA: ["Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["NonEmptyStruc", "Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
  ```

- `SetOfOdds`: change
  ```cue
  isA: ["Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["NonEmptyStruc", "Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
  ```

- `OSetOfNumbers`: change
  ```cue
  isA: ["OSet", "Set", "Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["NonEmptyStruc", "OSet", "Set", "Structure", "MathObj", "Anything"]
  ```

- `OSetOfPrimesDesc`: change
  ```cue
  isA: ["OSet", "Set", "Structure", "MathObj", "Anything"]
  ```
  to
  ```cue
  isA: ["NonEmptyStruc", "OSet", "Set", "Structure", "MathObj", "Anything"]
  ```

- [ ] **Step 5: Confirm pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestStructureClassificationTagsPropagate -v`
Expected: PASS.

Regression: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/`
Expected: all PASS. Pay attention to pre-existing tests that read Set/List/Bag/OSet isA — any assertion that checked exact isA length or specific ordering will need to be updated.

If any existing test breaks: **stop and report**. Do not silently adjust existing assertions — flag for the human to decide whether the test's expectations were narrow and should be loosened, or whether we've changed something we didn't intend.

- [ ] **Step 6: Commit**

```bash
git add domains/math/types.cue domains/math/sets.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.4 — tag existing types + instances with classifications

Set/List/Bag/OSet pick up Ord/UnOrd + MultEle/NoMultEle parents;
EmptySet picks up EmptyStruc; non-empty seed instances pick up
NonEmptyStruc. Projections and meta-ops can now gate on these.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Projection operation units

**Files:**
- Modify: `domains/math/operations.cue`
- Modify: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go` (`strings` already imported):

```go
// TestProjectionUnitsLoad verifies the six projection op units are present
// with correct domain/range and defn hooks.
func TestProjectionUnitsLoad(t *testing.T) {
	eng, _ := testEngine(t)
	wantOps := map[string]struct {
		domain []string
		rangeT []string
		defn   string
	}{
		"Proj1":       {[]string{"OPair"}, []string{"Anything"}, "first"},
		"Proj2":       {[]string{"OPair"}, []string{"Anything"}, "rest first"},
		"FirstEle":    {[]string{"OrdStruc"}, []string{"Anything"}, "first"},
		"LastEle":     {[]string{"OrdStruc"}, []string{"Anything"}, "last"},
		"AllButFirst": {[]string{"OrdStruc"}, []string{"OrdStruc"}, "rest"},
		"AllButLast":  {[]string{"OrdStruc"}, []string{"OrdStruc"}, "but-last"},
	}
	for name, want := range wantOps {
		u := eng.Store.Get(name)
		if u == nil {
			t.Errorf("%s not loaded", name)
			continue
		}
		dom := u.GetStrings("domain")
		if len(dom) != len(want.domain) {
			t.Errorf("%s.domain: want %v got %v", name, want.domain, dom)
		} else {
			for i, d := range want.domain {
				if dom[i] != d {
					t.Errorf("%s.domain[%d]: want %q got %q", name, i, d, dom[i])
				}
			}
		}
		rng := u.GetStrings("range")
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

- [ ] **Step 2: Confirm failure**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestProjectionUnitsLoad -v`
Expected: FAIL.

- [ ] **Step 3: Add op units in CUE**

Append to `domains/math/operations.cue` inside `units: [...]`:

```cue
	{
		name:    "Proj1"
		worth:   500
		isA: ["UnaryOp", "Op", "MathOp", "Anything"]
		domain: ["OPair"]
		range: ["Anything"]
		english: "First projection of an ordered pair"
		defn:    #"""
			first
			"""#
		examples: [
			{args: [3, 7], result: 3},
			{args: [19, 2], result: 19},
		]
	},
	{
		name:    "Proj2"
		worth:   500
		isA: ["UnaryOp", "Op", "MathOp", "Anything"]
		domain: ["OPair"]
		range: ["Anything"]
		english: "Second projection of an ordered pair"
		defn:    #"""
			rest first
			"""#
		examples: [
			{args: [3, 7], result: 7},
			{args: [19, 2], result: 2},
		]
	},
	{
		name:    "FirstEle"
		worth:   500
		isA: ["UnaryOp", "Op", "MathOp", "Anything"]
		domain: ["OrdStruc"]
		range: ["Anything"]
		english: "First element of an ordered structure"
		defn:    #"""
			first
			"""#
		examples: [
			{args: [3, 1, 2], result: 3},
			{args: [19, 17, 13], result: 19},
		]
	},
	{
		name:    "LastEle"
		worth:   500
		isA: ["UnaryOp", "Op", "MathOp", "Anything"]
		domain: ["OrdStruc"]
		range: ["Anything"]
		english: "Last element of an ordered structure"
		defn:    #"""
			last
			"""#
		examples: [
			{args: [3, 1, 2], result: 2},
			{args: [19, 17, 13], result: 13},
		]
	},
	{
		name:    "AllButFirst"
		worth:   500
		isA: ["UnaryOp", "Op", "MathOp", "Anything"]
		domain: ["OrdStruc"]
		range: ["OrdStruc"]
		english: "Ordered structure with its first element removed"
		defn:    #"""
			rest
			"""#
		examples: [
			{args: [3, 1, 2], result: [1, 2]},
			{args: [19, 17, 13], result: [17, 13]},
		]
	},
	{
		name:    "AllButLast"
		worth:   500
		isA: ["UnaryOp", "Op", "MathOp", "Anything"]
		domain: ["OrdStruc"]
		range: ["OrdStruc"]
		english: "Ordered structure with its last element removed"
		defn:    #"""
			but-last
			"""#
		examples: [
			{args: [3, 1, 2], result: [3, 1]},
			{args: [19, 17, 13], result: [19, 17]},
		]
	},
```

- [ ] **Step 4: Confirm pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestProjectionUnitsLoad -v`
Expected: PASS.

Regression: `cd /Users/chazu/dev/go/nous && go test ./...`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add domains/math/operations.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.3 — projection operation units

Six UnaryOp units: Proj1 / Proj2 (OPair projections), FirstEle / LastEle /
AllButFirst / AllButLast (OrdStruc projections). Domains gated by 5.4
classification categories; defns compose existing first/last/rest/but-last
builtins.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Engine smoke test for FirstEle dispatch via OrdStruc

**Files:**
- Modify: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing/passing test**

Append to `internal/engine/engine_test.go`:

```go
// TestFirstEleAppliedToOSetOfPrimesDesc is an end-to-end smoke test: the
// engine should apply FirstEle (domain=OrdStruc) to OSetOfPrimesDesc
// (which isA OrdStruc via OSet → OrdStruc) and record an applic with
// output 19. Regression guard against breaking OrdStruc domain dispatch
// or losing the OSet → OrdStruc chain.
func TestFirstEleAppliedToOSetOfPrimesDesc(t *testing.T) {
	eng, _ := testEngine(t)
	eng.SeedInitialAgenda()
	eng.MaxCycles = 100
	eng.Verbosity = 0

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	op := eng.Store.Get("FirstEle")
	if op == nil {
		t.Fatal("FirstEle missing after run")
	}
	applics, _ := op.Get("applics").([]map[string]any)
	if len(applics) == 0 {
		t.Fatalf("FirstEle recorded no applics in %d cycles", eng.MaxCycles)
	}

	found := false
	for _, ap := range applics {
		args, _ := ap["args"].([]string)
		if len(args) != 1 || args[0] != "OSetOfPrimesDesc" {
			continue
		}
		// Output is either a scalar int (IntVal) or a unit name.
		switch out := ap["output"].(type) {
		case int:
			if out == 19 {
				found = true
			}
		case string:
			u := eng.Store.Get(out)
			if u == nil {
				continue
			}
			// data slot could be scalar int or []dsl.Value
			switch d := u.Get("data").(type) {
			case int:
				if d == 19 {
					found = true
				}
			case []dsl.Value:
				if len(d) == 1 && d[0].AsInt() == 19 {
					found = true
				}
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Errorf("No FirstEle applic on OSetOfPrimesDesc with output 19; applics=%v", applics)
	}
}
```

**Note:** This test mirrors the existing `TestOSetUnionPreservesOrderViaEngine` in shape (same `SeedInitialAgenda` pattern, similar applic walk). The exact shape of `ap["output"]` may vary — the test handles both scalar-int outputs (FirstEle returns a single element) and unit-name outputs (if H-RunOnExamples wrapped the scalar in a created unit). If after running you find outputs consistently have a third shape, adapt; do not rewrite the whole test.

- [ ] **Step 2: Run the test**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestFirstEleAppliedToOSetOfPrimesDesc -v`

**If it passes immediately:** dispatch working, commit.

**If it fails with "recorded no applics":** try increasing `eng.MaxCycles` to 200 then 400. If still no applics, FirstEle never got focus — likely a worth-ordering issue identical to the OSetUnion case from Phase 5.1. `SeedInitialAgenda` should be pushing a task; if it isn't, check whether `SeedInitialAgenda` iterates UnaryOp as well as BinaryOp.

**If it fails with "output 19 not found":** the FirstEle dispatch is going somewhere, but not returning 19. Examine the actual applics output, and either (a) accept that the actual output shape is correct and adjust the assertion, or (b) stop and report — getting a non-19 output means OrdStruc dispatch or data reading is broken.

Do not proceed until this test passes.

- [ ] **Step 3: Full regression**

Run: `cd /Users/chazu/dev/go/nous && go test ./...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
test: Phase 5.3 — FirstEle OrdStruc-dispatch smoke test

End-to-end: FirstEle applied to OSetOfPrimesDesc returns 19. Regression
guard against breaking OrdStruc domain dispatch or losing the OSet →
OrdStruc chain.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Mark phases 5.3 + 5.4 COMPLETE in plan doc

**Files:**
- Modify: `docs/eurisko-parity-phases.md`

- [ ] **Step 1: Update phase 5.3 section**

Find:
```markdown
**5.3: Projection operations**
Proj1, Proj2, FirstEle, SecondEle, ThirdEle, LastEle, AllButFirst, AllButSecond, AllButThird, AllButLast. Most map directly to existing DSL builtins (first, rest, last) but need to exist as unit concepts.
```

Replace with:
```markdown
**5.3: Projection operations** -- PARTIAL (2026-04-20)

Six of ten projection ops landed in `domains/math/operations.cue`: Proj1, Proj2 (OPair domain), FirstEle, LastEle, AllButFirst, AllButLast (OrdStruc domain via 5.4 classification). New DSL builtin `but-last` added. SecondEle, ThirdEle, AllButSecond, AllButThird deferred — redundant with `rest`-chain composition; add when a heuristic demands them. Engine smoke test `TestFirstEleAppliedToOSetOfPrimesDesc` guards OrdStruc domain dispatch.
```

- [ ] **Step 2: Update phase 5.4 section**

Find:
```markdown
**5.4: Structure type classification**
OrdStruc/UnOrdStruc, MultEleStruc/NoMultEleStruc, EmptyStruc/NonEmptyStruc, SetOfSets, StructureOfStructures. These are type-level classifications that drive H29 and per-type operation applicability.
```

Replace with:
```markdown
**5.4: Structure type classification** -- PARTIAL (2026-04-20)

Six classification marker categories added in `domains/math/types.cue`: OrdStruc, UnOrdStruc, MultEleStruc, NoMultEleStruc, EmptyStruc, NonEmptyStruc. Each has no `defn` — pure marker categories queryable via `store.IsA` chain walks. Set/List/Bag/OSet now carry the relevant ord/multiplicity tags in `isA`; EmptySet carries EmptyStruc; non-empty seed instances carry NonEmptyStruc (instance-level tag to avoid false attribution to EmptySet). SetOfSets and StructureOfStructures higher-order categories deferred — no concrete instances yet. Unblocks Phase 5.12 H29 and Phase 5.6 D Restrict.
```

- [ ] **Step 3: Update summary table**

Find (around line 320):
```markdown
| 5 | Type hierarchy + operations | 12 | PARTIAL (5.1, 5.2, 5.6 A/B/C.1/C.2, 5.9, 5.10, 5.11 numeric-preds + H27/H28) |
```

Replace with:
```markdown
| 5 | Type hierarchy + operations | 12 | PARTIAL (5.1, 5.2, 5.3 partial, 5.4 partial, 5.6 A/B/C.1/C.2, 5.9, 5.10, 5.11 numeric-preds + H27/H28) |
```

- [ ] **Step 4: Commit**

```bash
git add docs/eurisko-parity-phases.md
git commit -m "$(cat <<'EOF'
docs: mark Phases 5.3 (projections) + 5.4 (classification) PARTIAL

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Post-Implementation

After all tasks:
1. `go test ./...` from repo root — all green.
2. Update memory (`project_nous_phases.md`) to reflect 5.3 + 5.4 partial completion.
3. Merge `phase-5.3-5.4-projections-structure` to main (via `superpowers:finishing-a-development-branch` or direct merge, per user preference).
