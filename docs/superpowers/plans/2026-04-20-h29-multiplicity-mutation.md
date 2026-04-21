# Phase 5.12 H29 Multiplicity Mutation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement H29 — the heuristic that generates new MultEleStruc examples by randomly mutating element multiplicities of existing examples.

**Architecture:** One new DSL builtin (`mutate-multiplicities`) + one seed unit family (`BagOfTallies` + three example children) + one CUE heuristic unit. No engine changes required; H29 fires via the existing `ifWorkingOnTask` pipeline. Tests cover DSL determinism, heuristic firing, and engine-level end-to-end growth.

**Tech Stack:** Go, CUE, Forth-like DSL VM, EURISKO-style unit engine.

**Spec:** `docs/superpowers/specs/2026-04-20-h29-multiplicity-mutation-design.md`

---

## File Structure

- `internal/dsl/builtins_math.go` — add `mutate-multiplicities` builtin (~15 lines).
- `internal/dsl/builtins_math_test.go` — `TestMutateMultiplicities`.
- `domains/math/sets.cue` — add `BagOfTallies`, `Bag-ex-tally-a`, `Bag-ex-tally-b`, `Bag-ex-tally-c`.
- `domains/common/heuristics.cue` — add `H29` unit.
- `internal/engine/engine_test.go` — add `TestH29FiresOnBagOfTallies` + `TestH29OneShotGuard`.
- `docs/eurisko-parity-phases.md` — mark 5.12 COMPLETE.

---

### Task 1: `mutate-multiplicities` DSL builtin

**Files:**
- Modify: `internal/dsl/builtins_math.go`
- Modify: `internal/dsl/builtins_math_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/dsl/builtins_math_test.go`:

```go
func TestMutateMultiplicities(t *testing.T) {
	// Every output element must come from the input alphabet.
	t.Run("element_preservation", func(t *testing.T) {
		vm := testVM(t)
		inputAlphabet := map[int]bool{1: true, 2: true, 3: true, 5: true}
		for i := 0; i < 50; i++ {
			v, err := vm.Execute(`1 2 3 5 4 list-of mutate-multiplicities`)
			if err != nil {
				t.Fatalf("exec: %v", err)
			}
			for _, e := range v.AsList() {
				if !inputAlphabet[e.AsInt()] {
					t.Errorf("foreign element %d in output %v", e.AsInt(), v.AsList())
				}
			}
		}
	})

	// Empty input → empty output, no error.
	t.Run("empty_input", func(t *testing.T) {
		vm := testVM(t)
		v, err := vm.Execute(`0 list-of mutate-multiplicities`)
		if err != nil {
			t.Fatalf("exec: %v", err)
		}
		if len(v.AsList()) != 0 {
			t.Errorf("empty input should produce empty output, got %v", v.AsList())
		}
	})

	// Over 100 runs on a 3-element input, expect both a longer-than-3 output
	// (at least one duplicate) and a shorter-than-3 output (at least one drop).
	t.Run("distribution_shows_drops_and_duplicates", func(t *testing.T) {
		vm := testVM(t)
		sawLonger := false
		sawShorter := false
		for i := 0; i < 200; i++ {
			v, err := vm.Execute(`1 2 3 3 list-of mutate-multiplicities`)
			if err != nil {
				t.Fatalf("exec: %v", err)
			}
			n := len(v.AsList())
			if n > 3 {
				sawLonger = true
			}
			if n < 3 {
				sawShorter = true
			}
			if sawLonger && sawShorter {
				return
			}
		}
		if !sawLonger {
			t.Error("200 runs never produced a longer-than-3 output — duplicate branch unreachable?")
		}
		if !sawShorter {
			t.Error("200 runs never produced a shorter-than-3 output — drop branch unreachable?")
		}
	})
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -run TestMutateMultiplicities -v`
Expected: FAIL with unknown word `mutate-multiplicities`.

- [ ] **Step 3: Register and implement**

In `internal/dsl/builtins_math.go`, inside `init()` — find the block registering list builtins near the existing `builtins["but-last"] = bButLast` (added in the previous phase) and add on a nearby line:

```go
	builtins["mutate-multiplicities"] = bMutateMult
```

Then append to the file (after the other `bOSet*` / `bButLast` functions):

```go
// mutate-multiplicities ( list -- list' )
// For each element: with equal probability drop, keep, or keep+duplicate.
// Uses VM.Rng so test runs are deterministic when the engine RNG is seeded.
func bMutateMult(vm *VM) error {
	in := vm.pop().AsList()
	out := make([]Value, 0, len(in)+2)
	for _, el := range in {
		switch vm.Rng.Intn(3) {
		case 0:
			// drop
		case 1:
			out = append(out, el)
		case 2:
			out = append(out, el, el)
		}
	}
	vm.push(ListVal(out))
	return nil
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -run TestMutateMultiplicities -v`
Expected: PASS.

Regression: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/dsl/builtins_math.go internal/dsl/builtins_math_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.12 — mutate-multiplicities DSL builtin

1/3 drop, 1/3 keep, 1/3 keep+duplicate per element. Uses VM.Rng so the
engine's seeded RNG makes heuristic runs reproducible.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: `BagOfTallies` seed unit family

**Files:**
- Modify: `domains/math/sets.cue`
- Modify: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestBagOfTalliesSeedsLoad verifies the H29 seed family loads from CUE:
// BagOfTallies (canonical multiset) with three example children carrying
// real multi-element data with duplicates.
func TestBagOfTalliesSeedsLoad(t *testing.T) {
	eng, _ := testEngine(t)

	bag := eng.Store.Get("BagOfTallies")
	if bag == nil {
		t.Fatal("BagOfTallies not loaded")
	}
	isA := bag.GetStrings("isA")
	wantTags := []string{"MultEleStruc", "UnOrdStruc", "NonEmptyStruc", "Bag"}
	isAMap := make(map[string]bool, len(isA))
	for _, t := range isA {
		isAMap[t] = true
	}
	for _, want := range wantTags {
		if !isAMap[want] {
			t.Errorf("BagOfTallies.isA missing %q; got %v", want, isA)
		}
	}
	examples := bag.GetStrings("examples")
	wantChildren := []string{"Bag-ex-tally-a", "Bag-ex-tally-b", "Bag-ex-tally-c"}
	exMap := make(map[string]bool, len(examples))
	for _, e := range examples {
		exMap[e] = true
	}
	for _, want := range wantChildren {
		if !exMap[want] {
			t.Errorf("BagOfTallies.examples missing %q; got %v", want, examples)
		}
	}

	for _, name := range wantChildren {
		child := eng.Store.Get(name)
		if child == nil {
			t.Errorf("%s child not loaded", name)
			continue
		}
		// data should be a non-empty list of ints
		data := child.Get("data")
		switch d := data.(type) {
		case []int:
			if len(d) == 0 {
				t.Errorf("%s.data empty", name)
			}
		case []any:
			if len(d) == 0 {
				t.Errorf("%s.data empty", name)
			}
		default:
			t.Errorf("%s.data unexpected type %T: %v", name, data, data)
		}
	}
}
```

- [ ] **Step 2: Confirm failure**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestBagOfTalliesSeedsLoad -v`
Expected: FAIL — none of the units exist.

- [ ] **Step 3: Add seed units to CUE**

Append to `domains/math/sets.cue` inside the `units: [...]` list (before closing `]`):

```cue
	{
		name:    "BagOfTallies"
		worth:   500
		isA: ["MultEleStruc", "UnOrdStruc", "NonEmptyStruc", "Bag", "Structure", "MathObj", "Anything"]
		english: "A bag with duplicate elements — a canonical multiset for mutation experiments"
		data: [1, 1, 2, 2, 2, 3, 5]
		examples: ["Bag-ex-tally-a", "Bag-ex-tally-b", "Bag-ex-tally-c"]
	},
	{
		name:    "Bag-ex-tally-a"
		worth:   300
		isA: ["Bag", "Structure", "MathObj", "Anything"]
		data: [1, 1, 2, 3]
	},
	{
		name:    "Bag-ex-tally-b"
		worth:   300
		isA: ["Bag", "Structure", "MathObj", "Anything"]
		data: [2, 2, 4, 5, 5]
	},
	{
		name:    "Bag-ex-tally-c"
		worth:   300
		isA: ["Bag", "Structure", "MathObj", "Anything"]
		data: [3, 3, 3, 7]
	},
```

- [ ] **Step 4: Confirm pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestBagOfTalliesSeedsLoad -v`
Expected: PASS.

Regression: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add domains/math/sets.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.12 — BagOfTallies seed family

Canonical multiset with three example children carrying real multi-element
data. Gives H29 immediate material to mutate. Instance-level MultEleStruc
tagging consistent with Phase 5.4 decision.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: H29 heuristic

**Files:**
- Modify: `domains/common/heuristics.cue`
- Modify: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestH29UnitLoads verifies the H29 unit is loaded from CUE with the
// expected slot shape before exercising the firing pipeline.
func TestH29UnitLoads(t *testing.T) {
	eng, _ := testEngine(t)
	h := eng.Store.Get("H29")
	if h == nil {
		t.Fatal("H29 not loaded")
	}
	isA := h.GetStrings("isA")
	foundHeuristic := false
	for _, s := range isA {
		if s == "Heuristic" {
			foundHeuristic = true
		}
	}
	if !foundHeuristic {
		t.Errorf("H29.isA missing Heuristic; got %v", isA)
	}
	if cap, _ := h.Get("h29Cap").(int); cap != 5 {
		t.Errorf("H29.h29Cap: want 5, got %v", h.Get("h29Cap"))
	}
	if prog := h.GetString("ifWorkingOnTask"); prog == "" {
		t.Error("H29.ifWorkingOnTask empty")
	}
	if prog := h.GetString("thenCompute"); prog == "" {
		t.Error("H29.thenCompute empty")
	}
}

// TestH29FiresOnBagOfTallies directly invokes H29's task-firing path on
// a BagOfTallies examples task and verifies new mutated example children
// appear with non-empty data drawn from the input alphabet.
func TestH29FiresOnBagOfTallies(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	bag := eng.Store.Get("BagOfTallies")
	if bag == nil {
		t.Fatal("BagOfTallies not loaded")
	}
	beforeExs := bag.GetStrings("examples")
	beforeCount := len(beforeExs)
	if beforeCount < 3 {
		t.Fatalf("expected ≥3 seed examples, got %d", beforeCount)
	}

	task := &agenda.Task{
		Priority: 500,
		UnitName: "BagOfTallies",
		SlotName: "examples",
		Reasons:  []string{"test: H29 firing"},
	}
	fired, _, _ := eng.fireTaskRule("H29", task)
	if !fired {
		t.Fatal("H29 did not fire on BagOfTallies examples task")
	}

	afterExs := eng.Store.Get("BagOfTallies").GetStrings("examples")
	if len(afterExs) <= beforeCount {
		t.Fatalf("expected more than %d examples after H29 firing, got %d (%v)", beforeCount, len(afterExs), afterExs)
	}

	// New children should all start with the "Bag-ex-H29-" prefix and have
	// non-empty data drawn from {1,2,3,4,5,7} (the union of source alphabets).
	sourceAlphabet := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 7: true}
	newCount := 0
	for _, name := range afterExs {
		isNew := true
		for _, old := range beforeExs {
			if name == old {
				isNew = false
				break
			}
		}
		if !isNew {
			continue
		}
		newCount++
		if !strings.HasPrefix(name, "Bag-ex-H29-") {
			t.Errorf("new example %q does not have H29 prefix", name)
		}
		child := eng.Store.Get(name)
		if child == nil {
			t.Errorf("new example %q not in store", name)
			continue
		}
		data := child.Get("data")
		var intList []int
		switch d := data.(type) {
		case []int:
			intList = d
		case []any:
			for _, v := range d {
				if i, ok := v.(int); ok {
					intList = append(intList, i)
				}
			}
		}
		if len(intList) == 0 {
			t.Errorf("%s.data empty — H29 should skip empty mutations", name)
		}
		for _, v := range intList {
			if !sourceAlphabet[v] {
				t.Errorf("%s.data has foreign element %d; data=%v", name, v, intList)
			}
		}
	}
	if newCount == 0 {
		t.Error("no new H29-created examples detected")
	}
	if newCount > 5 {
		t.Errorf("H29 created %d new examples, exceeds h29Cap=5", newCount)
	}
}

// TestH29OneShotGuard verifies the h29Ran flag prevents repeat firings
// on the same source unit.
func TestH29OneShotGuard(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	task := &agenda.Task{
		Priority: 500,
		UnitName: "BagOfTallies",
		SlotName: "examples",
		Reasons:  []string{"test: first pass"},
	}
	eng.fireTaskRule("H29", task)
	firstPass := len(eng.Store.Get("BagOfTallies").GetStrings("examples"))

	// Second pass on the same unit should be a no-op.
	task2 := &agenda.Task{
		Priority: 500,
		UnitName: "BagOfTallies",
		SlotName: "examples",
		Reasons:  []string{"test: second pass"},
	}
	eng.fireTaskRule("H29", task2)
	secondPass := len(eng.Store.Get("BagOfTallies").GetStrings("examples"))

	if secondPass != firstPass {
		t.Errorf("h29Ran guard failed: first pass left %d examples, second pass left %d", firstPass, secondPass)
	}
}
```

- [ ] **Step 2: Confirm failure**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run "TestH29" -v`
Expected: FAIL — H29 not loaded.

- [ ] **Step 3: Add H29 to heuristics.cue**

In `domains/common/heuristics.cue`, append inside the `units: [...]` list (before the closing `]`):

```cue
	{
		name:    "H29"
		worth:   500
		isA: ["Heuristic", "Anything"]
		english: "New examples of a MultEleStruc can be found by randomly mutating element multiplicities in known examples"
		overallRecord: {successes: 0, failures: 0}
		h29Cap: 5
		ifWorkingOnTask: #"""
			"CurUnit" @ "MultEleStruc" isa?
			"CurSlot" @ "examples" =
			and
			"CurUnit" @ "h29Ran" get-slot nil =
			and
			"""#
		thenCompute: #"""
			"CurUnit" @ true "h29Ran" set-slot

			"CurUnit" @ "examples" get-slot "srcExs" !
			"H29" "h29Cap" get-slot "cap" !
			0 "made" !

			"srcExs" @ each
				"made" @ "cap" @ < if
					it "srcName" !
					"srcName" @ "data" get-slot mutate-multiplicities "newData" !
					"newData" @ list-length 0 > if
						"Bag-ex-H29-" "CurUnit" @ concat "-" concat "made" @ concat "newName" !
						"newName" @ unit-exists? not if
							"newName" @ "Bag" create-unit drop
							"newData" @ "newName" @ "data" set-slot
							"newName" @ "CurUnit" @ "examples" add-to-slot
							"H29" "newName" @ "creditors" set-slot
							"made" @ 1 + "made" !
						then
					then
				then
			end

			"made" @ 0 > if
				"H29: created " "made" @ concat " new multiplicity-mutated examples for " concat "CurUnit" @ concat print
			then
			"""#
	},
```

**Note:** the exact location within the CUE `units: [...]` list matters for readability only; CUE is unordered. Place H29 near the other high-numbered heuristics (H27/H28) for grep-friendliness.

- [ ] **Step 4: Confirm pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run "TestH29" -v`
Expected: all three TestH29* tests PASS.

Regression: `cd /Users/chazu/dev/go/nous && go test ./...`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add domains/common/heuristics.cue internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
feat: Phase 5.12 — H29 multiplicity mutation heuristic

Fires on MultEleStruc examples tasks; iterates existing example children
and creates up to h29Cap new children with element multiplicities mutated
via mutate-multiplicities. One-shot per source via h29Ran flag. First live
exercise of the Phase 5.4 instance-level classification tagging.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Full engine smoke test

**Files:**
- Modify: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the test**

Append to `internal/engine/engine_test.go`:

```go
// TestH29FiresViaEngineLoop is an end-to-end smoke test: the engine's
// main loop should schedule an examples task on BagOfTallies and H29
// should fire, growing the examples list. Regression guard against
// breaking the dispatch chain (SeedInitialAgenda → task focus → H29 gate).
func TestH29FiresViaEngineLoop(t *testing.T) {
	eng, _ := testEngine(t)
	eng.SeedInitialAgenda()
	eng.MaxCycles = 150
	eng.Verbosity = 0

	beforeCount := len(eng.Store.Get("BagOfTallies").GetStrings("examples"))

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	afterCount := len(eng.Store.Get("BagOfTallies").GetStrings("examples"))
	if afterCount <= beforeCount {
		t.Errorf("engine loop did not grow BagOfTallies.examples: before=%d after=%d", beforeCount, afterCount)
	}
}
```

- [ ] **Step 2: Run the test**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestH29FiresViaEngineLoop -v`

**If it passes:** proceed to commit.

**If it fails with "did not grow":** investigate before adjusting. Possible causes:
- `SeedInitialAgenda` may not seed `examples` tasks on arbitrary MultEleStruc units — check what tasks it queues. If BagOfTallies never gets an examples task, H29 can't fire via the loop (even though direct invocation works via Task 3's test).
- An earlier heuristic may mutate `h29Ran` before H29's gate sees it (unlikely given one-shot semantics).
- Focus-budget exhaustion: 150 cycles may not be enough; try 300.

Stop and report if the issue is structural (e.g., SeedInitialAgenda doesn't cover MultEleStruc). Do not paper over by forcing the task onto the agenda — the loop-based test should pass through normal pipeline behavior.

- [ ] **Step 3: Full regression**

Run: `cd /Users/chazu/dev/go/nous && go test ./...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/engine/engine_test.go
git commit -m "$(cat <<'EOF'
test: Phase 5.12 — H29 fires via the engine main loop

End-to-end: SeedInitialAgenda queues a task for BagOfTallies, the engine
picks it up, and H29 grows the examples list. Regression guard against
breaking the dispatch chain between agenda seeding and H29's gate.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Mark Phase 5.12 COMPLETE in plan doc

**Files:**
- Modify: `docs/eurisko-parity-phases.md`

- [ ] **Step 1: Update Phase 5.12 section**

Find:
```markdown
**5.12: H29 -- Multiplicity mutation**
Once MultEleStruc exists, implement element multiplicity mutation for generating new examples.
```

Replace with:
```markdown
**5.12: H29 -- Multiplicity mutation** -- COMPLETE (2026-04-20)

H29 landed in `domains/common/heuristics.cue`. Fires on `ifWorkingOnTask` when CurUnit isA MultEleStruc and CurSlot == examples; iterates existing example children and creates up to `h29Cap` (default 5) new children with element multiplicities mutated via new `mutate-multiplicities` DSL builtin (1/3 drop, 1/3 keep, 1/3 duplicate per element). One-shot per source via `h29Ran` flag. Seed family `BagOfTallies` + three children in `domains/math/sets.cue` gives H29 immediate material. First live exercise of the Phase 5.4 instance-level classification dispatch.
```

- [ ] **Step 2: Update summary table**

Find (around line 320):
```markdown
| 5 | Type hierarchy + operations | 12 | PARTIAL (5.1, 5.2, 5.3 partial, 5.4 partial, 5.6 A/B/C.1/C.2, 5.9, 5.10, 5.11 numeric-preds + H27/H28) |
```

Replace with:
```markdown
| 5 | Type hierarchy + operations | 12 | PARTIAL (5.1, 5.2, 5.3 partial, 5.4 partial, 5.6 A/B/C.1/C.2, 5.9, 5.10, 5.11 numeric-preds + H27/H28, 5.12) |
```

- [ ] **Step 3: Commit**

```bash
git add docs/eurisko-parity-phases.md
git commit -m "$(cat <<'EOF'
docs: mark Phase 5.12 (H29 multiplicity mutation) COMPLETE

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Post-Implementation

After all tasks:
1. `go test ./...` from repo root — all green.
2. Update memory (`project_nous_phases.md`) to reflect 5.12 complete.
3. Merge `phase-5.12-h29` to main.
