# Phase 5.6 D — Restrict + MetaOpHeuristic Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the Restrict meta-operation (new DSL builtin + CUE heuristic) and refactor H-SemanticDup's creditor gate from a hard-coded allowlist into a proper `MetaOpHeuristic` isA category, with a multiset domain-equality precheck that auto-exempts Restrict outputs while preserving commutative-Transpose killing.

**Architecture:** One new Go DSL builtin (`restrict-op`, analogous to `transpose-op`/`compose-ops`), one extension to `applics-redundant?` (multiset domain precheck), three CUE edits (new `MetaOpHeuristic` category, new `H-Restrict` heuristic, gate refactor on `H-SemanticDup`). No engine-package changes.

**Tech Stack:** Go 1.x, CUE (domain data), internal DSL (stack VM with `subExecute` helper), existing `internal/unit/store.go` Store API.

**Spec:** `docs/superpowers/specs/2026-04-21-restrict-and-metaop-gate-design.md`.

---

## File Structure

- **Modify:** `internal/dsl/builtins_math.go` — add `bRestrictOp` (registered as `restrict-op`); extend `bApplicsRedundant` with multiset domain precheck.
- **Modify:** `domains/common/heuristics.cue` — add `MetaOpHeuristic` category unit; add `H-Restrict` heuristic; add `MetaOpHeuristic` to `isA` of H-Transpose, H-Compose, H-Restrict; refactor `H-SemanticDup.ifPotentiallyRelevant` creditors check.
- **Modify:** `internal/engine/engine_test.go` — add seven new tests (enumerated below).
- **Modify (close-out only):** `docs/eurisko-parity-phases.md` — mark Phase 5.6 D partial-complete (Restrict done, InvertOp deferred).

---

## Task 1: `MetaOpHeuristic` category unit

**Files:**
- Modify: `domains/common/heuristics.cue` (add new unit near top of `units:` list, adjacent to other shared scaffolding)

- [ ] **Step 1: Add the category unit**

Insert into `domains/common/heuristics.cue` at a clean location (e.g., just before the first `name: "H-..."` unit):

```cue
{
	name:    "MetaOpHeuristic"
	worth:   400
	isA: ["Anything"]
	english: "Heuristic category: produces new Op units via meta-operation (Transpose/Compose/Restrict/...). Consumed by H-SemanticDup to gate behavioral-duplicate checks."
},
```

- [ ] **Step 2: Build and confirm CUE loads**

Run: `go build ./...`
Expected: success, no CUE validation errors.

- [ ] **Step 3: Commit**

```bash
git add domains/common/heuristics.cue
git commit -m "feat: Phase 5.6 D — MetaOpHeuristic category unit"
```

---

## Task 2: Tag H-Transpose + H-Compose `isA MetaOpHeuristic`, add regression test

**Files:**
- Modify: `domains/common/heuristics.cue:1076,1100` (H-Transpose and H-Compose `isA` lines)
- Modify: `internal/engine/engine_test.go` (append new test)

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestMetaOpHeuristicCategoryWalk verifies that H-Transpose and H-Compose
// (and later H-Restrict) have MetaOpHeuristic in their isA chain, so the
// H-SemanticDup gate can walk creditors.isA MetaOpHeuristic.
func TestMetaOpHeuristicCategoryWalk(t *testing.T) {
	eng := newTestEngine(t)
	for _, h := range []string{"H-Transpose", "H-Compose"} {
		if !eng.Store.IsA(h, "MetaOpHeuristic") {
			t.Errorf("%s should isA MetaOpHeuristic", h)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestMetaOpHeuristicCategoryWalk -v`
Expected: FAIL — `H-Transpose should isA MetaOpHeuristic` (and H-Compose).

- [ ] **Step 3: Edit isA lists**

In `domains/common/heuristics.cue`, change:

```cue
	name:    "H-Transpose"
	worth:   500
	isA: ["Heuristic", "Anything"]
```

to:

```cue
	name:    "H-Transpose"
	worth:   500
	isA: ["Heuristic", "MetaOpHeuristic", "Anything"]
```

Same edit on the `H-Compose` unit — change `isA: ["Heuristic", "Anything"]` to `isA: ["Heuristic", "MetaOpHeuristic", "Anything"]`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestMetaOpHeuristicCategoryWalk -v`
Expected: PASS.

- [ ] **Step 5: Run full engine tests to confirm no regression**

Run: `go test ./internal/engine/ -v 2>&1 | tail -30`
Expected: all existing tests still pass.

- [ ] **Step 6: Commit**

```bash
git add domains/common/heuristics.cue internal/engine/engine_test.go
git commit -m "feat: Phase 5.6 D — tag H-Transpose/H-Compose as MetaOpHeuristic"
```

---

## Task 3: Multiset domain-equality precheck in `applics-redundant?`

**Files:**
- Modify: `internal/dsl/builtins_math.go:832-852` (head of `bApplicsRedundant`)
- Test: new test stanza in `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestApplicsRedundantDomainMismatch verifies that applics-redundant? returns
// false when the unit's domain is a different multiset from the parent's,
// even if outputs would otherwise match. This is the precheck that exempts
// Restrict from H-SemanticDup killing.
func TestApplicsRedundantDomainMismatch(t *testing.T) {
	eng := newTestEngine(t)

	// Seed a delegating child with NARROWER domain than Add.
	// Add.domain = [Number, Number]; child.domain = [PrimeNum, Number].
	// Child's defn delegates to Add, so outputs would match — but domains
	// differ as multisets, so applics-redundant? must return false.
	child := unit.New("Restricted-Add")
	child.Set("isA", []string{"BinaryOp", "Op", "MathOp", "Anything"})
	child.SetWorth(500)
	child.Set("domain", []string{"PrimeNum", "Number"})
	child.Set("range", []string{"Number"})
	child.Set("defn", `"Add" apply-op-args`)
	child.Set("creditors", []string{"H-Restrict"})
	// Three applics whose outputs match parent Add.
	child.Set("applics", []map[string]any{
		{"args": []string{"N-2", "N-3"}, "output": "N-5"},
		{"args": []string{"N-5", "N-7"}, "output": "N-12"},
		{"args": []string{"N-3", "N-11"}, "output": "N-14"},
	})
	eng.Store.Put(child)
	eng.Store.SetSlot("Restricted-Add", "generalizations", []string{"Add"})

	v, err := eng.VM.Execute(`"Restricted-Add" "Add" applics-redundant?`)
	if err != nil {
		t.Fatalf("applics-redundant? error: %v", err)
	}
	if v.AsBool() {
		t.Fatalf("applics-redundant? should return false when domains differ (multiset), got true")
	}
}

// TestApplicsRedundantDomainPermutationStillRedundant verifies that a reversed
// domain (the Transpose case) compares equal as a multiset, so behaviorally
// identical Transpose outputs are still flagged redundant — this is the
// regression guardrail against the 2026-04-19 commutative-Transpose killing.
func TestApplicsRedundantDomainPermutation(t *testing.T) {
	eng := newTestEngine(t)

	// child.domain = [Number, Number] (same as Add after reversal).
	child := unit.New("Pseudo-Transpose-Add")
	child.Set("isA", []string{"BinaryOp", "Op", "MathOp", "Anything"})
	child.SetWorth(500)
	child.Set("domain", []string{"Number", "Number"})
	child.Set("range", []string{"Number"})
	child.Set("defn", `+`) // commutative-identical to Add.defn.
	child.Set("creditors", []string{"H-Transpose"})
	child.Set("applics", []map[string]any{
		{"args": []string{"N-2", "N-3"}, "output": "N-5"},
		{"args": []string{"N-5", "N-7"}, "output": "N-12"},
		{"args": []string{"N-3", "N-11"}, "output": "N-14"},
	})
	eng.Store.Put(child)
	eng.Store.SetSlot("Pseudo-Transpose-Add", "generalizations", []string{"Add"})

	v, err := eng.VM.Execute(`"Pseudo-Transpose-Add" "Add" applics-redundant?`)
	if err != nil {
		t.Fatalf("applics-redundant? error: %v", err)
	}
	if !v.AsBool() {
		t.Fatalf("applics-redundant? should return true for domain-equal + output-equal child, got false")
	}
}
```

- [ ] **Step 2: Run the new tests to verify TestApplicsRedundantDomainMismatch fails**

Run: `go test ./internal/engine/ -run TestApplicsRedundantDomain -v`
Expected: `TestApplicsRedundantDomainMismatch` FAILS (returns true today because outputs match and there's no domain precheck); `TestApplicsRedundantDomainPermutation` passes today (already-correct behavior).

- [ ] **Step 3: Add the multiset domain precheck**

In `internal/dsl/builtins_math.go`, locate `bApplicsRedundant` (starts near line 832). Immediately after the `parentDefn == ""` check and before the `applicsRaw` read, insert:

```go
	// Domain-multiset precheck: if the unit and parent operate on different
	// multisets of domain types, behavioral match on sampled applics does
	// not imply redundancy (Restrict's narrowed domain is the discovery).
	// Multiset (not ordered) compare so reversed-domain Transpose on a
	// commutative op still reaches the output-matching loop below.
	uDomain := u.GetStrings("domain")
	pDomain := parent.GetStrings("domain")
	if !multisetEqualStrings(uDomain, pDomain) {
		vm.push(BoolVal(false))
		return nil
	}
```

Then add a package-level helper in the same file (near the other small helpers, e.g. after `semanticValuesEqual`):

```go
// multisetEqualStrings returns true iff a and b contain the same strings
// with the same multiplicities. Ignores order. nil and empty compare equal.
func multisetEqualStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run both new tests to verify PASS**

Run: `go test ./internal/engine/ -run TestApplicsRedundantDomain -v`
Expected: both PASS.

- [ ] **Step 5: Run existing semantic-dup tests to confirm no regression**

Run: `go test ./internal/engine/ -run TestH19CriterialSparesSpecializations -v`
Also re-run the original `applics-redundant?` tests referenced in `engine_test.go:3009-3059`:
Run: `go test ./internal/engine/ -run applics-redundant -v` (or the containing test name — grep `grep -n "TestSemantic\|TestApplicsRedundant" internal/engine/engine_test.go`)
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/dsl/builtins_math.go internal/engine/engine_test.go
git commit -m "feat: Phase 5.6 D — multiset domain precheck in applics-redundant?"
```

---

## Task 4: Refactor `H-SemanticDup.ifPotentiallyRelevant` gate

**Files:**
- Modify: `domains/common/heuristics.cue:1136-1152` (H-SemanticDup `ifPotentiallyRelevant`)

- [ ] **Step 1: Read the current gate**

Current body (at `domains/common/heuristics.cue:1136-1152`):

```
ifPotentiallyRelevant: #"""
	"ArgU" @ "Op" isa?
	"ArgU" @ "creditors" get-slot nil !=
	and
	"ArgU" @ "creditors" get-slot "H-Transpose" list-contains
	"ArgU" @ "creditors" get-slot "H-Compose" list-contains
	or
	and
	"ArgU" @ "generalizations" get-slot nil !=
	and
	"ArgU" @ "applics" get-slot nil !=
	and
	"ArgU" @ "applics" get-slot 3 >=
	and
	"ArgU" @ "semDupChecked" get-slot nil =
	and
	"""#
```

- [ ] **Step 2: Replace the hardcoded allowlist with an isA-walk over creditors**

Replace with:

```
ifPotentiallyRelevant: #"""
	"ArgU" @ "Op" isa?
	"ArgU" @ "creditors" get-slot nil !=
	and
	false "metaOp" !
	"ArgU" @ "creditors" get-slot
	each
		it "MetaOpHeuristic" isa?
		if
			true "metaOp" !
		then
	end
	"metaOp" @
	and
	"ArgU" @ "generalizations" get-slot nil !=
	and
	"ArgU" @ "applics" get-slot nil !=
	and
	"ArgU" @ "applics" get-slot 3 >=
	and
	"ArgU" @ "semDupChecked" get-slot nil =
	and
	"""#
```

Rationale: walks every creditor and checks `isa? MetaOpHeuristic`; any hit sets `metaOp` flag to true. Avoids needing a new `any` helper. Idiomatic to existing CUE heuristics in this file (see H-Compose `each` loop at `domains/common/heuristics.cue:1112-1126`).

- [ ] **Step 3: Build + run all existing H-SemanticDup tests**

Run: `go build ./... && go test ./internal/engine/ -run TestH19CriterialSparesSpecializations -v`
Expected: PASS. (The H19 test is the tripwire for "gate too loose kills legitimate H-Specialize output" — since H-Specialize is not `isA MetaOpHeuristic`, the gate correctly excludes its outputs.)

Also run: `go test ./internal/engine/ -v 2>&1 | grep -E 'FAIL|PASS: .*SemanticDup|PASS: .*Transpose|PASS: .*Compose'`
Expected: no FAIL.

- [ ] **Step 4: Commit**

```bash
git add domains/common/heuristics.cue
git commit -m "feat: Phase 5.6 D — H-SemanticDup gate walks creditors isA MetaOpHeuristic"
```

---

## Task 5: `restrict-op` DSL builtin + unit test

**Files:**
- Modify: `internal/dsl/builtins_math.go` (append `bRestrictOp` after `bComposeOps`; register in builtins map)
- Modify: `internal/engine/engine_test.go` (append `TestRestrictOpCreatesNarrowedUnit`)

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/engine_test.go`:

```go
// TestRestrictOpCreatesNarrowedUnit verifies restrict-op builds a new Op
// unit with one domain position narrowed to a specialization of the
// parent's type at that position, delegating defn, and correct creditor.
func TestRestrictOpCreatesNarrowedUnit(t *testing.T) {
	eng := newTestEngine(t)

	// Add is seeded with domain=[Number,Number]; Number has specializations
	// including PrimeNum. So restrict-op must pick some specialization for
	// position 0 or 1 and build the narrowed unit.
	v, err := eng.VM.Execute(`"Add" restrict-op`)
	if err != nil {
		t.Fatalf("restrict-op: %v", err)
	}
	newName := v.AsString()
	if newName == "" {
		t.Fatalf("restrict-op returned empty/nil — expected a Restrict-Add-* name")
	}
	u := eng.Store.Get(newName)
	if u == nil {
		t.Fatalf("new unit %q not found in store", newName)
	}

	// Structural checks.
	if !strings.HasPrefix(newName, "Restrict-Add-") {
		t.Errorf("expected name prefix Restrict-Add-, got %q", newName)
	}
	gens := u.GetStrings("generalizations")
	if len(gens) != 1 || gens[0] != "Add" {
		t.Errorf("generalizations = %v, want [Add]", gens)
	}
	creds := u.GetStrings("creditors")
	if len(creds) != 1 || creds[0] != "H-Restrict" {
		t.Errorf("creditors = %v, want [H-Restrict]", creds)
	}
	domain := u.GetStrings("domain")
	if len(domain) != 2 {
		t.Fatalf("domain len %d, want 2: %v", len(domain), domain)
	}
	// Exactly one of the two domain positions must be a specialization of Number
	// (i.e. not "Number" itself), and the other must remain "Number".
	numberCount := 0
	specCount := 0
	for _, d := range domain {
		if d == "Number" {
			numberCount++
		} else if eng.Store.IsA(d, "Number") {
			specCount++
		}
	}
	if numberCount != 1 || specCount != 1 {
		t.Errorf("domain %v: want exactly one Number and one Number-specialization", domain)
	}
	// Range unchanged.
	rng := u.GetStrings("range")
	if len(rng) != 1 || rng[0] != "Number" {
		t.Errorf("range = %v, want [Number]", rng)
	}
	// Defn delegates to parent.
	defn := u.GetString("defn")
	if !strings.Contains(defn, "Add") || !strings.Contains(defn, "apply-op-args") {
		t.Errorf("defn = %q, want containing Add and apply-op-args", defn)
	}
	// One-shot flag set on parent.
	parent := eng.Store.Get("Add")
	flag, _ := parent.Get("restrictRan").(bool)
	if !flag {
		t.Errorf("Add.restrictRan should be true after restrict-op")
	}
}
```

(The test file already imports `strings` and `unit`; if not, add them to the existing import block.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/engine/ -run TestRestrictOpCreatesNarrowedUnit -v`
Expected: FAIL — `restrict-op` is not a registered builtin.

- [ ] **Step 3: Implement `bRestrictOp`**

Append to `internal/dsl/builtins_math.go` (after `bComposeOps`, before `bApplicsRedundant`):

```go
// restrict-op: ( opName -- newOpName | "" )
// Picks a random domain position whose type has ≥1 specialization, chooses
// a random specialization of that type, and creates Restrict-<op>-<N> with
// the narrowed domain, same range/arity, and a defn delegating to parent
// via apply-op-args. Sets <op>.restrictRan = true (one-shot guard).
// Returns "" on precondition failure (not an Op, missing defn, no
// specializable domain position).
func bRestrictOp(vm *VM) error {
	opName := vm.pop().AsString()
	u := vm.Store.Get(opName)
	if u == nil {
		vm.push(StringVal(""))
		return nil
	}
	if !vm.Store.IsA(opName, "Op") {
		vm.push(StringVal(""))
		return nil
	}
	defn := u.GetString("defn")
	if defn == "" {
		vm.push(StringVal(""))
		return nil
	}
	domain := u.GetStrings("domain")
	if len(domain) == 0 {
		vm.push(StringVal(""))
		return nil
	}

	// Collect positions where domain[i] has ≥1 immediate specialization.
	type cand struct {
		pos    int
		specs  []string
	}
	var candidates []cand
	for i, t := range domain {
		specs := vm.Store.Specializations(t)
		if len(specs) > 0 {
			candidates = append(candidates, cand{pos: i, specs: specs})
		}
	}
	if len(candidates) == 0 {
		vm.push(StringVal(""))
		return nil
	}

	c := candidates[vm.Rng.Intn(len(candidates))]
	chosen := c.specs[vm.Rng.Intn(len(c.specs))]

	newDomain := make([]string, len(domain))
	copy(newDomain, domain)
	newDomain[c.pos] = chosen

	// Gensym. Start at 1; bump until unused.
	var newName string
	for n := 1; ; n++ {
		newName = fmt.Sprintf("Restrict-%s-%d", opName, n)
		if !vm.Store.Has(newName) {
			break
		}
	}

	// Worth: average of parent and static H-Restrict worth (600) —
	// matches EURISKO's AverageWorths convention.
	parentWorth := u.Worth()
	newWorth := (parentWorth + 600) / 2

	// Delegating defn: push each arg back onto the stack, then call parent.
	// arity N: "<opName>" apply-op-args consumes the N args we re-supply.
	newDefnStr := fmt.Sprintf(`"%s" apply-op-args`, opName)

	newU := unit.New(newName)
	newU.Set("isA", append([]string{}, u.GetStrings("isA")...))
	newU.SetWorth(newWorth)
	newU.Set("domain", newDomain)
	newU.Set("range", append([]string{}, u.GetStrings("range")...))
	newU.Set("arity", u.Get("arity"))
	newU.Set("defn", newDefnStr)
	newU.Set("creditors", []string{"H-Restrict"})
	vm.Store.Put(newU)
	vm.Store.SetSlot(newName, "generalizations", []string{opName})
	vm.Store.SetSlot(newName, "extensions", []string{opName})
	vm.Store.SetSlot(opName, "restrictRan", true)
	vm.push(StringVal(newName))
	return nil
}
```

Then register it. Locate the builtin map in `internal/dsl/builtins_math.go` (near `builtins["applics-redundant?"] = bApplicsRedundant` at line 96):

Add line:
```go
	builtins["restrict-op"] = bRestrictOp
```

(If `u.Worth()` isn't the right accessor, substitute with whatever `transpose-op` uses to read worth — confirmed by reading `bTransposeOp`. If no `Worth()` method, fall back to `u.Get("worth").(int)` with a default of 500.)

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: success. If `apply-op-args` is not a registered builtin, reality-check: grep `grep -n "apply-op-args\|apply-op" internal/dsl/`. `apply-op-args` should exist (per spec referenced in existing phases); if missing, confirm the correct delegating verb before proceeding.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestRestrictOpCreatesNarrowedUnit -v`
Expected: PASS.

- [ ] **Step 6: Run full test suite**

Run: `go test ./... 2>&1 | tail -30`
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/dsl/builtins_math.go internal/engine/engine_test.go
git commit -m "feat: Phase 5.6 D — restrict-op DSL builtin"
```

---

## Task 6: `H-Restrict` heuristic

**Files:**
- Modify: `domains/common/heuristics.cue` (append new heuristic after H-Compose, before H-SemanticDup)

- [ ] **Step 1: Write the failing engine-loop smoke test**

Append to `internal/engine/engine_test.go`:

```go
// TestHRestrictFiresOnEligibleOp runs the engine long enough for H-Restrict
// to detect Add (has applics + domain with specializations) and create a
// Restrict-Add-* unit. Asserts one-shot behavior: second tick creates no
// additional Restrict-Add unit.
func TestHRestrictFiresOnEligibleOp(t *testing.T) {
	eng := newTestEngine(t)

	// Seed a recorded applic on Add so the ≥1-applics gate passes.
	// (Seed Number instances N-2/N-3/N-5 already exist.)
	add := eng.Store.Get("Add")
	add.Set("applics", []map[string]any{
		{"args": []string{"N-2", "N-3"}, "output": "N-5"},
	})
	eng.Store.Put(add)

	// Queue an agenda task on Add so H-Restrict has a firing opportunity.
	eng.Agenda.Add(engine.Task{Priority: 500, UnitName: "Add", SlotName: "examples", Reason: "test: exercise H-Restrict"})

	// Tick enough cycles for the agenda to work through Add's task.
	for i := 0; i < 50; i++ {
		if !eng.TickOne() {
			break
		}
	}

	// Find any Restrict-Add-* unit.
	found := 0
	for _, name := range eng.Store.All() {
		if strings.HasPrefix(name, "Restrict-Add-") {
			found++
		}
	}
	if found == 0 {
		t.Fatalf("expected ≥1 Restrict-Add-* unit, got 0")
	}

	// One-shot check: tick another 50 cycles, re-count, should still be ≤ first count.
	firstCount := found
	eng.Agenda.Add(engine.Task{Priority: 500, UnitName: "Add", SlotName: "examples", Reason: "test: re-exercise H-Restrict"})
	for i := 0; i < 50; i++ {
		if !eng.TickOne() {
			break
		}
	}
	found2 := 0
	for _, name := range eng.Store.All() {
		if strings.HasPrefix(name, "Restrict-Add-") {
			found2++
		}
	}
	if found2 > firstCount {
		t.Errorf("one-shot guard failed: %d Restrict-Add units after first pass, %d after second", firstCount, found2)
	}
}

// TestHRestrictSkipsOpWithoutApplics — zero applics on Add disables H-Restrict.
func TestHRestrictSkipsOpWithoutApplics(t *testing.T) {
	eng := newTestEngine(t)

	// Clear applics on Add (seed leaves some; we want zero).
	add := eng.Store.Get("Add")
	add.Set("applics", []map[string]any{})
	eng.Store.Put(add)

	eng.Agenda.Add(engine.Task{Priority: 500, UnitName: "Add", SlotName: "examples", Reason: "test: no-applics Add"})
	for i := 0; i < 50; i++ {
		if !eng.TickOne() {
			break
		}
	}
	for _, name := range eng.Store.All() {
		if strings.HasPrefix(name, "Restrict-Add-") {
			t.Fatalf("H-Restrict fired despite zero applics on Add: created %s", name)
		}
	}
}

// TestHRestrictSkipsOpWithoutSpecializableDomain — op whose domain types
// have no specializations is ineligible.
func TestHRestrictSkipsOpWithoutSpecializableDomain(t *testing.T) {
	eng := newTestEngine(t)

	// Create a synthetic op whose domain type has no specializations.
	// "TruthValue" exists in seed data without specializations.
	opU := unit.New("TestNoSpec")
	opU.Set("isA", []string{"UnaryOp", "Op", "MathOp", "Anything"})
	opU.SetWorth(500)
	opU.Set("domain", []string{"TruthValue"})
	opU.Set("range", []string{"TruthValue"})
	opU.Set("defn", `not`)
	opU.Set("applics", []map[string]any{
		{"args": []string{"True"}, "output": "False"},
	})
	eng.Store.Put(opU)

	eng.Agenda.Add(engine.Task{Priority: 500, UnitName: "TestNoSpec", SlotName: "examples", Reason: "test: no-spec domain"})
	for i := 0; i < 50; i++ {
		if !eng.TickOne() {
			break
		}
	}
	for _, name := range eng.Store.All() {
		if strings.HasPrefix(name, "Restrict-TestNoSpec-") {
			t.Fatalf("H-Restrict fired despite no specializable domain: %s", name)
		}
	}
}
```

(Adjust `eng.Agenda.Add` / `eng.TickOne` / `engine.Task` to match actual engine API — check the existing H-Transpose or H29 smoke tests for the exact idiom. If the harness uses `eng.RunForN(50)` or similar, use that instead.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/engine/ -run TestHRestrict -v`
Expected: all three FAIL — H-Restrict is not yet defined.

- [ ] **Step 3: Add the H-Restrict heuristic**

Insert into `domains/common/heuristics.cue` between H-Compose and H-SemanticDup:

```cue
{
	name:    "H-Restrict"
	worth:   600
	isA: ["Heuristic", "MetaOpHeuristic", "Anything"]
	english: "Create a narrowed version of an Op by specializing one domain position"
	overallRecord: {successes: 0, failures: 0}
	ifPotentiallyRelevant: #"""
		"ArgU" @ "Op" isa?
		"ArgU" @ "defn" get-slot nil !=
		and
		"ArgU" @ "applics" get-slot nil !=
		and
		"ArgU" @ "applics" get-slot list-length 0 >
		and
		"ArgU" @ "restrictRan" get-slot nil =
		and
		false "hasSpec" !
		"ArgU" @ "domain" get-slot
		each
			it "specializations" get-slot
			dup nil !=
			swap
			if
				list-length 0 >
				if
					true "hasSpec" !
				then
			else
				drop
			then
		end
		"hasSpec" @
		and
		"""#
	thenCompute: #"""
		"ArgU" @ restrict-op
		"newOp" !
		"newOp" @ "" !=
		if
			400 "newOp" @ "examples" "Examples for restricted op" add-task
			"Restricted " "ArgU" @ concat " → " concat "newOp" @ concat print
		then
		"""#
},
```

Notes:
- The `hasSpec` walk mirrors the idiom in the refactored H-SemanticDup gate (Task 4) and the H-Compose loop (`domains/common/heuristics.cue:1112-1126`). If the DSL's `each` + `get-slot nil` + branch idiom looks clumsy vs. existing heuristics, match the exact local convention.
- `restrict-op` sets `ArgU.restrictRan = true` as a side effect, so the one-shot guard on subsequent ticks is enforced by the builtin (and doubly by the gate).
- `list-length` must exist in the DSL (check `grep -n "list-length" internal/dsl/builtins.go`). If it's spelled differently (e.g. `length`), use the actual name.

- [ ] **Step 4: Run the three H-Restrict tests**

Run: `go test ./internal/engine/ -run TestHRestrict -v`
Expected: all three PASS.

- [ ] **Step 5: Run full engine test suite**

Run: `go test ./... 2>&1 | tail -30`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add domains/common/heuristics.cue internal/engine/engine_test.go
git commit -m "feat: Phase 5.6 D — H-Restrict heuristic"
```

---

## Task 7: H-SemanticDup end-to-end exemption + regression tests

**Files:**
- Modify: `internal/engine/engine_test.go` (append two tests)

- [ ] **Step 1: Write the failing/regression tests**

Append to `internal/engine/engine_test.go`:

```go
// TestSemanticDupExemptsRestrict verifies the full path: H-Restrict creates
// Restrict-Add; H-SemanticDup observes it has ≥3 behaviorally-matching applics
// vs Add; but multiset domain precheck returns false; so Restrict-Add survives.
func TestSemanticDupExemptsRestrict(t *testing.T) {
	eng := newTestEngine(t)

	// Hand-construct Restrict-Add directly (same shape restrict-op would produce).
	r := unit.New("Restrict-Add-test")
	r.Set("isA", []string{"BinaryOp", "Op", "MathOp", "Anything"})
	r.SetWorth(550)
	r.Set("domain", []string{"PrimeNum", "Number"})
	r.Set("range", []string{"Number"})
	r.Set("defn", `"Add" apply-op-args`)
	r.Set("creditors", []string{"H-Restrict"})
	r.Set("applics", []map[string]any{
		{"args": []string{"N-2", "N-3"}, "output": "N-5"},
		{"args": []string{"N-5", "N-7"}, "output": "N-12"},
		{"args": []string{"N-3", "N-11"}, "output": "N-14"},
	})
	eng.Store.Put(r)
	eng.Store.SetSlot("Restrict-Add-test", "generalizations", []string{"Add"})

	// Give H-SemanticDup a chance to fire.
	eng.Agenda.Add(engine.Task{Priority: 500, UnitName: "Restrict-Add-test", SlotName: "examples", Reason: "test: sem-dup exemption"})
	for i := 0; i < 50; i++ {
		if !eng.TickOne() {
			break
		}
	}

	// Assert the unit survives.
	if eng.Store.Get("Restrict-Add-test") == nil {
		t.Fatalf("Restrict-Add-test was killed — multiset domain precheck should have spared it")
	}
}

// TestSemanticDupStillKillsCommutativeTranspose — regression. Seeds a fake
// commutative Transpose whose domain multiset equals parent's and outputs
// match. H-SemanticDup must kill it (2026-04-19 behavior preserved).
func TestSemanticDupStillKillsCommutativeTranspose(t *testing.T) {
	eng := newTestEngine(t)

	tr := unit.New("Transpose-Add-test")
	tr.Set("isA", []string{"BinaryOp", "Op", "MathOp", "Anything"})
	tr.SetWorth(500)
	tr.Set("domain", []string{"Number", "Number"})
	tr.Set("range", []string{"Number"})
	tr.Set("defn", `swap +`)
	tr.Set("creditors", []string{"H-Transpose"})
	tr.Set("applics", []map[string]any{
		{"args": []string{"N-2", "N-3"}, "output": "N-5"},
		{"args": []string{"N-5", "N-7"}, "output": "N-12"},
		{"args": []string{"N-3", "N-11"}, "output": "N-14"},
	})
	eng.Store.Put(tr)
	eng.Store.SetSlot("Transpose-Add-test", "generalizations", []string{"Add"})

	eng.Agenda.Add(engine.Task{Priority: 500, UnitName: "Transpose-Add-test", SlotName: "examples", Reason: "test: sem-dup kill"})
	for i := 0; i < 50; i++ {
		if !eng.TickOne() {
			break
		}
	}

	if eng.Store.Get("Transpose-Add-test") != nil {
		t.Fatalf("Transpose-Add-test should have been killed by H-SemanticDup (behavioral + multiset-domain-equal match)")
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/engine/ -run 'TestSemanticDupExemptsRestrict|TestSemanticDupStillKillsCommutativeTranspose' -v`
Expected: both PASS (Task 3's precheck and Task 4's refactor already make this behavior correct; these tests are end-to-end verification).

If either fails, diagnose:
- Exempt fails → precheck in Task 3 isn't firing on `Restrict-Add-test`. Confirm `domain` multiset `[PrimeNum, Number] != [Number, Number]`.
- Regression fails → domain compare is ordered not multiset. Recheck `multisetEqualStrings`.

- [ ] **Step 3: Run full test suite**

Run: `go test ./... 2>&1 | tail -30`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/engine/engine_test.go
git commit -m "test: Phase 5.6 D — H-SemanticDup exempt-Restrict + Transpose-kill regression"
```

---

## Task 8: Update phase doc, close out phase 5.6 D

**Files:**
- Modify: `docs/eurisko-parity-phases.md`

- [ ] **Step 1: Edit the phase doc**

In `docs/eurisko-parity-phases.md`, find the line:

```
- **D Restrict + InvertOp** -- deferred. Restrict partially exists via `restrictedTo` (H6-Specialize); InvertOp is genuinely complex and low priority.
```

Replace with:

```
- **D Restrict** -- COMPLETE (2026-04-21). `restrict-op` DSL builtin + `H-Restrict` heuristic creates `Restrict-<op>-<N>` with one domain position narrowed to a random specialization; delegating defn via `apply-op-args`; one-shot via `restrictRan` flag. New `MetaOpHeuristic` isA category; H-Transpose/H-Compose/H-Restrict tagged; H-SemanticDup gate refactored to walk `creditors isA MetaOpHeuristic` instead of hardcoded allowlist. `applics-redundant?` gained multiset domain-equality precheck — auto-exempts Restrict (narrower domain) while preserving commutative-Transpose killing (reversed domain = same multiset). Spec: `docs/superpowers/specs/2026-04-21-restrict-and-metaop-gate-design.md`. Plan: `docs/superpowers/plans/2026-04-21-restrict-and-metaop-gate.md`.
- **D' InvertOp** -- deferred. EURISKO worth 100; requires inverse-search infra; no concrete demand yet.
```

Also update the Phase 5 summary row (search for `5.6 A/B/C.1/C.2` and extend to `5.6 A/B/C.1/C.2/D`).

- [ ] **Step 2: Commit the close-out**

```bash
git add docs/eurisko-parity-phases.md
git commit -m "docs: mark Phase 5.6 D (Restrict) COMPLETE"
```

- [ ] **Step 3: Update auto-memory**

Update `/Users/chazu/.claude/projects/-Users-chazu-dev-go-nous/memory/project_nous_phases.md` frontmatter and body to reflect Phase 5.6 D complete. Remove the H-SemanticDup creditor-gate followup from "Standing followups" (it's now resolved).

---

## Self-Review

**Spec coverage:** every spec section has a task:
- Restrict firing gate (spec §Design decisions) → Task 6 H-Restrict ifPotentiallyRelevant.
- `restrict-op` builtin behavior (spec §Design decisions) → Task 5.
- MetaOpHeuristic category (spec §Design decisions) → Task 1, tagging in Tasks 2 + 6.
- H-SemanticDup gate refactor (spec §Design decisions) → Task 4.
- `applics-redundant?` domain precheck (spec §Design decisions) → Task 3.
- All seven tests in spec §Testing → Tasks 2, 3, 5, 6, 7.

**Placeholder scan:** no TBD/TODO/"handle edge cases"; all code-changing steps include actual code. Task 5 Step 3 has one "substitute if method name differs" note on `u.Worth()`, which is a reality-check instruction, not a placeholder — the engineer verifies against `bTransposeOp` (which is read at the top of this plan's research phase).

**Type consistency:** `restrict-op` returns empty string `""` on failure (matches `bRestrictOp` body in Task 5 and `v.AsString() == ""` check in Task 5 test). Gate uses `"" !=` in H-Restrict thenCompute (Task 6). Consistent throughout.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-21-restrict-and-metaop-gate.md`. Two execution options:

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
