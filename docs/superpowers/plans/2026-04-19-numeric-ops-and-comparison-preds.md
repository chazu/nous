# Phase 5.9 + 5.11: Numeric Ops and Comparison Predicates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add four numeric ops (`Add`, `Multiply`, `Successor`, `Square`) and four numeric comparison predicates (`IEQP`, `IGEQ`, `IGREATERP`, `ILESSP`) as first-class seed units in the math domain.

**Architecture:** Pure data additions to two CUE files. No engine, DSL, or heuristic code changes. Existing machinery (`H-RunOnExamples`, `apply-pred`, Phase 5.10 Rarity hook, H24/H25/H26/H27/H28) picks up the new units automatically. Tests live in `internal/engine/engine_test.go` alongside existing domain-integration tests.

**Tech Stack:** CUE (seed data), Go (tests), nous DSL (defn bodies: `+`, `*`, `=`, `>=`, `>`, `<`, `dup`, `1 +` — all pre-existing builtins).

**Spec reference:** `docs/superpowers/specs/2026-04-19-numeric-ops-and-comparison-preds-design.md`

---

## File Structure

| File | Action | Purpose |
|---|---|---|
| `domains/math/predicates.cue` | Modify (append 4 units) | Add IEQP / IGEQ / IGREATERP / ILESSP |
| `domains/math/operations.cue` | Modify (append 4 units) | Add Add / Multiply / Successor / Square |
| `internal/engine/engine_test.go` | Modify (append 2 test funcs) | TestNumericComparisonPreds, TestNumericOpsAsUnits |

No new files.

---

## Task 1: Numeric Comparison Predicates

**Files:**
- Modify: `domains/math/predicates.cue` (append 4 units before closing `]`)
- Modify: `internal/engine/engine_test.go` (append `TestNumericComparisonPreds`)

### - [ ] Step 1: Write the failing test

Append to the end of `internal/engine/engine_test.go` (after the last existing test function, before EOF):

```go
// Phase 5.11: four numeric comparison predicates must exist as first-class
// Pred units with correct isA, domain, range, and executable defns. The
// Phase 5.10 Rarity hook in apply-op should populate rarity on invocation.
func TestNumericComparisonPreds(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	names := []string{"IEQP", "IGEQ", "IGREATERP", "ILESSP"}
	for _, n := range names {
		u := eng.Store.Get(n)
		if u == nil {
			t.Fatalf("pred %q not loaded from domains/math/predicates.cue", n)
		}
		isA := u.GetStrings("isA")
		hasBinary, hasPred := false, false
		for _, t := range isA {
			if t == "BinaryPred" {
				hasBinary = true
			}
			if t == "Pred" {
				hasPred = true
			}
		}
		if !hasBinary || !hasPred {
			t.Errorf("%s isA=%v; want BinaryPred and Pred", n, isA)
		}
		if got := u.GetStrings("domain"); len(got) != 2 || got[0] != "Number" || got[1] != "Number" {
			t.Errorf("%s domain=%v; want [Number Number]", n, got)
		}
		if u.GetString("defn") == "" {
			t.Errorf("%s has empty defn", n)
		}
	}

	// Truth-table spot checks via apply-pred.
	type tc struct {
		prog string
		want bool
	}
	cases := []tc{
		{`5 3 "IGREATERP" apply-pred`, true},
		{`3 5 "IGREATERP" apply-pred`, false},
		{`3 5 "ILESSP" apply-pred`, true},
		{`5 3 "ILESSP" apply-pred`, false},
		{`3 3 "IEQP" apply-pred`, true},
		{`3 4 "IEQP" apply-pred`, false},
		{`5 5 "IGEQ" apply-pred`, true},
		{`5 6 "IGEQ" apply-pred`, false},
	}
	for _, c := range cases {
		v, err := eng.VM.Execute(c.prog)
		if err != nil {
			t.Fatalf("Execute(%q) error: %v", c.prog, err)
		}
		if v.Truthy() != c.want {
			t.Errorf("Execute(%q) = %v; want %v", c.prog, v.Truthy(), c.want)
		}
	}

	// Phase 5.10 Rarity hook: IGREATERP was called twice (one true, one false).
	u := eng.Store.Get("IGREATERP")
	r, ok := u.Get("rarity").([]any)
	if !ok || len(r) != 3 {
		t.Fatalf("IGREATERP.rarity = %v; want 3-element list", u.Get("rarity"))
	}
	// r[1]=numT, r[2]=numF — stored as int per updateRarity in builtins_math.go
	numT, numTOk := r[1].(int)
	numF, numFOk := r[2].(int)
	if !numTOk || !numFOk {
		t.Fatalf("IGREATERP.rarity counters have unexpected types: %T %T", r[1], r[2])
	}
	if numT < 1 || numF < 1 {
		t.Errorf("IGREATERP.rarity counters numT=%d numF=%d; want both >=1", numT, numF)
	}
}
```

### - [ ] Step 2: Run test to verify it fails

```bash
cd /Users/chazu/dev/go/nous
go test ./internal/engine/ -run TestNumericComparisonPreds -v
```

Expected: FAIL with `pred "IEQP" not loaded from domains/math/predicates.cue`.

If you see a compile error about unused imports or undefined symbols, resolve before moving on.

### - [ ] Step 3: Add the four predicate units

Open `domains/math/predicates.cue`. It currently ends with the `AlwaysNIL` unit followed by `]` on line 72. Insert the following four units after `AlwaysNIL`'s closing `},` and before the final `]`:

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

Indentation is a single tab per the existing file's style. Match exactly.

### - [ ] Step 4: Run test to verify it passes

```bash
go test ./internal/engine/ -run TestNumericComparisonPreds -v
```

Expected: PASS.

If it fails with a CUE load error, run `go test ./internal/cueload/... -v` to surface the schema problem, then fix the offending unit. Likely causes: a stray comma, wrong indent, or missing required slot.

### - [ ] Step 5: Commit

```bash
git add domains/math/predicates.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.11 — numeric comparison predicates as units

IEQP / IGEQ / IGREATERP / ILESSP land as BinaryPred units in the math
domain. Phase 5.10 Rarity hook populates rarity on invocation; H24 can
now discover rare numeric predicates on Number-valued categories.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Numeric Operations

**Files:**
- Modify: `domains/math/operations.cue` (append 4 units before closing `]`)
- Modify: `internal/engine/engine_test.go` (append `TestNumericOpsAsUnits`)

### - [ ] Step 1: Write the failing test

Append to the end of `internal/engine/engine_test.go`:

```go
// Phase 5.9: four numeric ops must exist as first-class Op units. The
// existence/shape assertions catch CUE schema regressions; the
// H-RunOnExamples firing assertion catches defn-body errors (the ops
// must actually produce correct results through the full pipeline).
func TestNumericOpsAsUnits(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	// Existence and shape.
	want := map[string]struct {
		isAEntries []string // required isA entries (subset match)
		domain     []string
		rng        []string
	}{
		"Add":       {[]string{"BinaryOp", "Op"}, []string{"Number", "Number"}, []string{"Number"}},
		"Multiply":  {[]string{"BinaryOp", "Op"}, []string{"Number", "Number"}, []string{"Number"}},
		"Successor": {[]string{"UnaryOp", "Op"}, []string{"Number"}, []string{"Number"}},
		"Square":    {[]string{"UnaryOp", "Op"}, []string{"Number"}, []string{"Number"}},
	}
	for n, spec := range want {
		u := eng.Store.Get(n)
		if u == nil {
			t.Fatalf("op %q not loaded from domains/math/operations.cue", n)
		}
		isA := u.GetStrings("isA")
		for _, req := range spec.isAEntries {
			found := false
			for _, got := range isA {
				if got == req {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s isA=%v; missing %q", n, isA, req)
			}
		}
		if got := u.GetStrings("domain"); !stringSliceEq(got, spec.domain) {
			t.Errorf("%s domain=%v; want %v", n, got, spec.domain)
		}
		if got := u.GetStrings("range"); !stringSliceEq(got, spec.rng) {
			t.Errorf("%s range=%v; want %v", n, got, spec.rng)
		}
		if u.GetString("defn") == "" {
			t.Errorf("%s has empty defn", n)
		}
	}

	// Defn correctness via apply-op (no H-RunOnExamples needed for this check —
	// avoids dependency on Number-instance seeding).
	type tc struct {
		prog string
		want int
	}
	cases := []tc{
		{`2 3 "Add" apply-op`, 5},
		{`4 5 "Multiply" apply-op`, 20},
		{`7 "Successor" apply-op`, 8},
		{`6 "Square" apply-op`, 36},
	}
	for _, c := range cases {
		v, err := eng.VM.Execute(c.prog)
		if err != nil {
			t.Fatalf("Execute(%q) error: %v", c.prog, err)
		}
		if got := v.AsInt(); got != c.want {
			t.Errorf("Execute(%q) = %d; want %d", c.prog, v.AsInt(), c.want)
		}
	}

	// End-to-end: H-RunOnExamples on Successor produces an applic when
	// Number has concrete instance units. Seed two Number instances with
	// data; fire H-RunOnExamples on Successor; assert an applic was recorded.
	seedNumberInstance(t, eng, "Num-3", 3)
	seedNumberInstance(t, eng, "Num-5", 5)

	eng.fireUnitRule("H-RunOnExamples", "Successor")

	succ := eng.Store.Get("Successor")
	applics, _ := succ.Get("applics").([]map[string]any)
	if len(applics) == 0 {
		t.Fatalf("Successor.applics empty after H-RunOnExamples; want >=1 entry")
	}
	// At least one applic should reference one of our seeded Number instances.
	foundSrc := false
	for _, a := range applics {
		args, _ := a["args"].([]any)
		for _, arg := range args {
			if s, _ := arg.(string); s == "Num-3" || s == "Num-5" {
				foundSrc = true
			}
		}
	}
	if !foundSrc {
		t.Errorf("Successor.applics did not reference Num-3 or Num-5; applics=%v", applics)
	}
}

func stringSliceEq(a, b []string) bool {
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

// seedNumberInstance creates a Number-typed instance unit with the given
// data value and appends it to Number.examples (so H-RunOnExamples picks
// it up). Mirrors what H-Generate would produce for real.
func seedNumberInstance(t *testing.T, eng *Engine, name string, data int) {
	t.Helper()
	u := unit.New(name)
	u.Set("isA", []string{"Number", "Anything"})
	u.SetWorth(500)
	u.Set("data", data)
	eng.Store.Put(u)

	num := eng.Store.Get("Number")
	existing := num.Get("examples")
	var ex []any
	if l, ok := existing.([]any); ok {
		ex = l
	}
	ex = append(ex, name)
	eng.Store.SetSlot("Number", "examples", ex)
}

// (helper toIntForTest not needed — Value.AsInt() covers apply-op results.)
```

If the test file already has a `stringSliceEq` helper (check with `grep -n "^func stringSliceEq" internal/engine/engine_test.go`), drop the duplicate from this task.

### - [ ] Step 2: Run test to verify it fails

```bash
go test ./internal/engine/ -run TestNumericOpsAsUnits -v
```

Expected: FAIL with `op "Add" not loaded from domains/math/operations.cue`.

### - [ ] Step 3: Add the four operation units

Open `domains/math/operations.cue`. It currently ends after the `Restrict` unit (line 92 `},`) followed by `]` on line 93. Insert the following four units after `Restrict`'s closing `},` and before the final `]`:

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

Match existing indentation (single tab).

### - [ ] Step 4: Run test to verify it passes

```bash
go test ./internal/engine/ -run TestNumericOpsAsUnits -v
```

Expected: PASS.

### - [ ] Step 5: Commit

```bash
git add domains/math/operations.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.9 — numeric ops as units

Add / Multiply / Successor / Square land as Op units in the math domain
with seeded examples. H-RunOnExamples picks them up automatically; no
engine or DSL changes needed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Regression check and status update

**Files:**
- Modify: `docs/eurisko-parity-phases.md` (mark 5.9 and 5.11 COMPLETE)

### - [ ] Step 1: Run the full test suite

```bash
cd /Users/chazu/dev/go/nous
go test ./... 2>&1 | tail -60
```

Expected: all packages PASS. The new units change seed counts; if any test is pinned to a specific unit count (`Store.Count()`) and fails, that test's expectation needs updating to account for the 8 new units. Report the failing test name and expected/actual counts rather than changing it unilaterally — the reviewer should decide whether the pin was load-bearing.

Known candidate: `TestEngineCreatesUnits` uses `finalCount > initialCount` (monotonic), which is unaffected. A grep of test files for `Count()` finds only monotonic uses; no pinned counts. If a different test breaks, surface it.

### - [ ] Step 2: Run a 100-cycle math engine smoke run

```bash
cd /Users/chazu/dev/go/nous
go build -o nous ./cmd/nous 2>&1 | tail -5
./nous -domains-dir ./domains -domain math -cycles 100 -verbosity 0 2>&1 | tail -20
```

Expected: clean exit, non-zero unit growth, no panic, no runaway agenda. Record the terminal summary (final unit count, applics-count, mutation count) in your report for the reviewer — this is observational, not asserted. Look especially for any Rarity lines referencing the new preds, and any applics accruing on Add/Multiply/Successor/Square.

If the run panics or hangs, capture the stack/trailing output and stop. Do not proceed to the doc update.

### - [ ] Step 3: Update the phase tracking doc

Edit `docs/eurisko-parity-phases.md`:

Find the line for 5.9 (currently `**5.9: Numeric operations as units**` near line 211). Replace the body with:

```markdown
**5.9: Numeric operations as units** -- COMPLETE
Add, Multiply, Successor, Square landed as seed units in `domains/math/operations.cue` (Phase 5.9 + 5.11 plan, 2026-04-19). Each has worth 500, BinaryOp/UnaryOp isA as appropriate, [Number]→[Number] signatures, and seeded raw-literal examples matching the GCD/DivisorsOf precedent. H-RunOnExamples picks them up via the existing pipeline; no engine changes.
```

Find the line for 5.11 (currently `**5.11: H25-H28 -- Predicate set analysis** -- PARTIAL ...`). Update the PARTIAL marker: leave the existing H27/H28-COMPLETE / H25-H26-deferred framing unchanged, but add a new paragraph at the end of its section:

```markdown
**Numeric comparison predicates** -- COMPLETE (2026-04-19)
IEQP, IGEQ, IGREATERP, ILESSP landed in `domains/math/predicates.cue` as BinaryPred units with [Number, Number]→TruthValue. Phase 5.10 Rarity hook populates rarity on invocation. Transpose variants (EURISKO pairs IGEQ↔IGREATERP and ILESSP) deferred to Phase 5.6A.
```

Also update the summary table at the bottom of the file — the row `| 5 | Type hierarchy + operations | 12 | Not started |` should become `| 5 | Type hierarchy + operations | 12 | PARTIAL (5.2, 5.6 C.1/C.2, 5.9, 5.10, 5.11 numeric-preds, 5.11 H27/H28) |` — matching the style of the existing Phase 7 entry. If the exact wording of nearby rows has drifted, follow their current pattern.

### - [ ] Step 4: Update the memory index

Edit `/Users/chazu/.claude/projects/-Users-chazu-dev-go-nous/memory/project_nous_phases.md`. Update the description frontmatter field and the status paragraph to reflect that 5.9 and the numeric-comparison-preds portion of 5.11 are now complete as of 2026-04-19. Keep the existing "pudl ↔ nous integration" note and "Next natural moves" — but swap the first "Next" entry (currently `5.9 + 5.11`) for `5.6A Transpose` since those have now shipped.

### - [ ] Step 5: Commit

```bash
git add docs/eurisko-parity-phases.md
# note: memory file is outside the repo; commit the in-repo change only
git commit -m "$(cat <<'EOF'
docs: mark Phase 5.9 COMPLETE and numeric comparison preds in 5.11

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Completion Criteria

- [ ] `go test ./internal/engine/ -run "TestNumericComparisonPreds|TestNumericOpsAsUnits" -v` passes
- [ ] `go test ./...` passes with no regressions
- [ ] 100-cycle math smoke run exits cleanly
- [ ] Phase doc shows 5.9 COMPLETE and 5.11 numeric-preds COMPLETE
- [ ] Memory index updated
- [ ] Three commits on branch (one per task)

---

## Notes for the implementing engineer

- The CUE files use tab indentation with a single leading tab per unit entry (look at the existing `AlwaysNIL` / `Restrict` units for reference). Preserve this exactly — CUE is whitespace-tolerant but `gofmt`-style checks on the CI side prefer consistency.
- `apply-op` and `apply-pred` are the same builtin under the hood (see `internal/dsl/builtins_math.go:517-520`). `apply-pred` is only semantically distinct when reading heuristic DSL. Either name works in tests.
- The Phase 5.10 Rarity hook in `bApplyOp` (`internal/dsl/builtins_math.go:473-475`) fires on any unit whose `isA` includes "Pred" — so all four new comparison preds inherit rarity tracking with zero additional work. That's why the test asserts on `IGREATERP.rarity` after invoking `apply-pred`.
- `Store.IsA(name, "Pred")` checks transitive isA membership; with `isA: ["BinaryPred", "Pred", "MathPred", "Anything"]` the direct membership matches. No need for a Pred → MathPred inheritance chain.
- `H-RunOnExamples` iterates `domType1.examples` (the first domain type's examples slot) and filters to entries where `"data" get-slot nil !=` — raw ints in `Number.examples` get filtered out; only instance units with `data` populated pass. That's why Task 2's test seeds `Num-3` / `Num-5` manually. In production, `H-Generate` produces `Number-gen-*` units that serve the same role.
- If a step's expected command output differs substantially from what you see, stop and report rather than pressing on. The plan is only as good as its assumptions about the current tree.
