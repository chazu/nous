# Phase 5.8 — Logical Operations as Units Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Seed six logical operations (And/Or/Not/Implies/TheFirstOf/TheSecondOf) + LogicOp category + True/False TruthValue instances as CUE units in `domains/math/logic.cue`.

**Architecture:** Pure CUE addition — one new file, no Go code changes, no new heuristics. Existing meta-op heuristics pick them up organically.

**Tech Stack:** CUE, Go test harness (`testEngine`), existing DSL builtins (`and`, `or`, `not`, `swap`, `drop`).

**Spec:** `docs/superpowers/specs/2026-04-21-logical-ops-as-units-design.md`.

---

## Task 1: Create `domains/math/logic.cue` with all units

**Files:**
- Create: `/Users/chazu/dev/go/nous/domains/math/logic.cue`

- [ ] **Step 1: Create the file with all nine units**

Use existing `domains/math/operations.cue` as formatting template (tabs, `units: [...]` structure, trailing comma on each field/unit).

Content of `/Users/chazu/dev/go/nous/domains/math/logic.cue`:

```cue
// Logical operations as first-class units (Phase 5.8).
// Follows EURISKO's LogicOp family: And, Or, Not, Implies, TheFirstOf, TheSecondOf.
// Domains restricted to TruthValue (tighter than EURISKO's Anything) except the
// polymorphic projections TheFirstOf/TheSecondOf. Generalizations chain mirrors
// EURISKO wiring for discovery heuristics.
units: [
	{
		name:    "LogicOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: logical operations over truth values"
	},
	{
		name:  "True"
		worth: 400
		isA: ["TruthValue", "Anything"]
		data: true
	},
	{
		name:  "False"
		worth: 400
		isA: ["TruthValue", "Anything"]
		data: false
	},
	{
		name:    "Not"
		worth:   500
		isA: ["UnaryOp", "Op", "LogicOp", "UnaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["TruthValue"]
		range:   ["TruthValue"]
		arity:   1
		english: "Logical negation of a truth value"
		defn: #"""
			not
			"""#
		examples: [
			{args: true, result: false},
			{args: false, result: true},
		]
	},
	{
		name:    "And"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["TruthValue", "TruthValue"]
		range:   ["TruthValue"]
		arity:   2
		english: "Logical conjunction of two truth values"
		generalizations: ["TheFirstOf", "TheSecondOf", "Or"]
		defn: #"""
			and
			"""#
		examples: [
			{args: true, args2: true, result: true},
			{args: true, args2: false, result: false},
			{args: false, args2: false, result: false},
		]
	},
	{
		name:    "Or"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["TruthValue", "TruthValue"]
		range:   ["TruthValue"]
		arity:   2
		english: "Logical disjunction of two truth values"
		defn: #"""
			or
			"""#
		examples: [
			{args: true, args2: false, result: true},
			{args: false, args2: false, result: false},
			{args: true, args2: true, result: true},
		]
	},
	{
		name:    "Implies"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["TruthValue", "TruthValue"]
		range:   ["TruthValue"]
		arity:   2
		english: "Material implication: (not x) or y"
		defn: #"""
			swap not swap or
			"""#
		examples: [
			{args: true, args2: true, result: true},
			{args: true, args2: false, result: false},
			{args: false, args2: false, result: true},
		]
	},
	{
		name:    "TheFirstOf"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["Anything", "Anything"]
		range:   ["Anything"]
		arity:   2
		english: "Polymorphic projection returning the first argument"
		generalizations: ["Or"]
		defn: #"""
			swap drop
			"""#
	},
	{
		name:    "TheSecondOf"
		worth:   500
		isA: ["BinaryOp", "Op", "LogicOp", "BinaryPred", "Pred", "MathOp", "MathConcept", "Anything"]
		domain:  ["Anything", "Anything"]
		range:   ["Anything"]
		arity:   2
		english: "Polymorphic projection returning the second argument"
		generalizations: ["Or"]
		defn: #"""
			drop
			"""#
	},
]
```

- [ ] **Step 2: Build to confirm CUE loads**

```
cd /Users/chazu/dev/go/nous && go build ./...
```
Expected: exit 0. If there are CUE validation errors (missing commas, wrong types, undefined categories like `UnaryPred` / `BinaryPred` not existing elsewhere), fix them — or remove the offending isA entry. Verify via: `grep -rn 'name: *"UnaryPred"\|name: *"BinaryPred"\|name: *"Pred"' domains/`. If `UnaryPred`/`BinaryPred` are undefined in this codebase, drop them from isA lists; `Pred` should already exist (used extensively in predicates.cue).

- [ ] **Step 3: Run engine-load smoke test**

```
cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestSeedInitialAgendaCoversAllOps -v 2>&1 | tail -10
```
Expected: PASS. This confirms the new units load and get seeded onto the agenda.

- [ ] **Step 4: Commit**

```
git add domains/math/logic.cue
git commit -m "$(cat <<'EOF'
feat: Phase 5.8 — logical operations as units (LogicOp + And/Or/Not/Implies/TheFirstOf/TheSecondOf)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Verification tests

**Files:**
- Modify: `/Users/chazu/dev/go/nous/internal/engine/engine_test.go`

- [ ] **Step 1: Write the tests**

Append to `internal/engine/engine_test.go`:

```go
// TestLogicalOpUnitsPresent — Phase 5.8: all logical-op units load from CUE
// with expected isA + category membership.
func TestLogicalOpUnitsPresent(t *testing.T) {
	eng, _ := testEngine(t)

	wantUnits := []string{"LogicOp", "True", "False", "And", "Or", "Not", "Implies", "TheFirstOf", "TheSecondOf"}
	for _, name := range wantUnits {
		if !eng.Store.Has(name) {
			t.Errorf("unit %q not loaded", name)
		}
	}

	// Category membership: every logical op isA LogicOp.
	for _, op := range []string{"And", "Or", "Not", "Implies", "TheFirstOf", "TheSecondOf"} {
		if !eng.Store.IsA(op, "LogicOp") {
			t.Errorf("%s should isA LogicOp", op)
		}
	}
	// And/Or/Not/Implies are Ops.
	for _, op := range []string{"And", "Or", "Not", "Implies", "TheFirstOf", "TheSecondOf"} {
		if !eng.Store.IsA(op, "Op") {
			t.Errorf("%s should isA Op", op)
		}
	}
	// True/False are TruthValue instances.
	for _, tv := range []string{"True", "False"} {
		if !eng.Store.IsA(tv, "TruthValue") {
			t.Errorf("%s should isA TruthValue", tv)
		}
	}
}

// TestLogicOpDefnsExecute — smoke-test each logical op's defn runs and
// produces the correct boolean result on the stack.
func TestLogicOpDefnsExecute(t *testing.T) {
	eng, _ := testEngine(t)

	cases := []struct {
		name   string
		script string
		want   bool
	}{
		{"Not true", `true "Not" @ "defn" get-slot run`, false},
		{"Not false", `false "Not" @ "defn" get-slot run`, true},
		{"And tt", `true true "And" @ "defn" get-slot run`, true},
		{"And tf", `true false "And" @ "defn" get-slot run`, false},
		{"Or ff", `false false "Or" @ "defn" get-slot run`, false},
		{"Or tf", `true false "Or" @ "defn" get-slot run`, true},
		{"Implies tt", `true true "Implies" @ "defn" get-slot run`, true},
		{"Implies tf", `true false "Implies" @ "defn" get-slot run`, false},
		{"Implies ff", `false false "Implies" @ "defn" get-slot run`, true},
		{"TheFirstOf", `true false "TheFirstOf" @ "defn" get-slot run`, true},
		{"TheSecondOf", `true false "TheSecondOf" @ "defn" get-slot run`, false},
	}

	for _, c := range cases {
		v, err := eng.VM.Execute(c.script)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if v.AsBool() != c.want {
			t.Errorf("%s: got %v, want %v", c.name, v.AsBool(), c.want)
		}
	}
}
```

**NOTE on the `run` verb**: the test scripts use `run` to execute a defn string fetched via `get-slot`. Check that `run` exists as a DSL builtin:
```
grep -n '"run"' /Users/chazu/dev/go/nous/internal/dsl/*.go | head -3
```
If `run` is not registered, inline the defn directly in each test case (use the defn body string literally rather than fetching via get-slot). Example fallback for Not:
```go
{"Not true", `true not`, false},
```

- [ ] **Step 2: Run tests**

```
cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run "TestLogicalOpUnitsPresent|TestLogicOpDefnsExecute" -v
```
Expected: both PASS. If `run` is missing and tests fail, switch to inlining the defn body (e.g. `true not`, `true false and`, `true not true or` for Implies, etc.).

- [ ] **Step 3: Full engine suite**

```
go test ./internal/engine/ 2>&1 | tail -5
```
Expected: all PASS.

- [ ] **Step 4: Commit**

```
git commit -m "$(cat <<'EOF'
test: Phase 5.8 — verify logical-op units load + defns execute correctly

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Phase-doc closeout

**Files:**
- Modify: `/Users/chazu/dev/go/nous/docs/eurisko-parity-phases.md`

- [ ] **Step 1: Mark Phase 5.8 COMPLETE**

Find the line:
```
**5.8: Logical operations as units**
AND, OR, NOT, Implies, TheFirstOf, TheSecondOf as operation units with defn and domain/range.
```

Replace with:
```
**5.8: Logical operations as units** -- COMPLETE (2026-04-21)
Six logical ops (And, Or, Not, Implies, TheFirstOf, TheSecondOf) + LogicOp category + True/False TruthValue instances landed in `domains/math/logic.cue`. And/Or/Not/Implies use TruthValue domains (tighter than EURISKO's Anything); TheFirstOf/TheSecondOf are polymorphic (Anything). Generalizations chain mirrors EURISKO: And → [TheFirstOf, TheSecondOf, Or]. No new heuristics — existing H-Transpose/H-Compose/H-Restrict pick them up organically. Spec: `docs/superpowers/specs/2026-04-21-logical-ops-as-units-design.md`. Plan: `docs/superpowers/plans/2026-04-21-logical-ops-as-units.md`.
```

- [ ] **Step 2: Update the Phase 5 summary row**

Find the phase-summary table row for Phase 5 (search `5.6 A/B/C.1/C.2/D`). Extend completed-slices to include `5.8`.

- [ ] **Step 3: Commit**

```
git add docs/eurisko-parity-phases.md
git commit -m "$(cat <<'EOF'
docs: mark Phase 5.8 (logical ops as units) COMPLETE

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```
