# Self-Modification Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the four disconnected feedback loops in nous's credit assignment and mutation system so heuristics improve over time.

**Architecture:** Bottom-up wiring. Each loop builds on the previous: accurate failure data (Loop 1) enables performance-based mutation, which enables pattern analysis (Loop 3) and HindSight validation (Loop 4). Loop 2 (reward) is independent but needed for the full credit cycle.

**Tech Stack:** Go, stdlib testing, no new dependencies.

---

### Task 1: No-Op Firing Detection

Record a failure when a heuristic fires but produces no new units and no new agenda items.

**Files:**
- Modify: `internal/engine/fire.go:114-129` (executeThenParts)
- Modify: `internal/engine/engine.go:119-155` (WorkOnTask, WorkOnUnit)
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test for no-op detection**

```go
func TestTrackApplicsNoOpFailure(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	// Type hierarchy
	anything := unit.New("Anything")
	anything.Set("isA", []string{})
	store.Put(anything)

	heuristic := unit.New("Heuristic")
	heuristic.Set("isA", []string{"Anything"})
	store.Put(heuristic)

	// A heuristic that fires but does nothing
	h := unit.New("H-NoOp")
	h.SetWorth(500)
	h.Set("isA", []string{"Heuristic", "Anything"})
	h.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	h.Set("ifPotentiallyRelevant", `true`)
	h.Set("thenCompute", `1 drop`) // fires but produces nothing
	store.Put(h)

	// A target unit
	target := unit.New("Target")
	target.SetWorth(500)
	target.Set("isA", []string{"Anything"})
	store.Put(target)

	// Push a task and run one cycle
	ag.Push(&agenda.Task{Priority: 500, UnitName: "Target", SlotName: "examples", Reasons: []string{"test"}})
	eng.MaxCycles = 1
	eng.MutConfig.Enabled = false
	eng.Run(context.Background())

	record := h.GetMap("overallRecord")
	if record == nil {
		t.Fatal("overallRecord is nil")
	}
	if toInt(record["failures"]) != 1 {
		t.Errorf("expected 1 failure for no-op firing, got %d", toInt(record["failures"]))
	}
	if toInt(record["successes"]) != 0 {
		t.Errorf("expected 0 successes, got %d", toInt(record["successes"]))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestTrackApplicsNoOpFailure -v`
Expected: FAIL (successes=1 because current code always records true)

- [ ] **Step 3: Add snapshot-based no-op detection to executeThenParts**

In `internal/engine/fire.go`, change `executeThenParts` to return whether output was produced:

```go
// executeThenParts runs all ThenPart slots of a heuristic.
// Returns (abort, producedOutput).
func (e *Engine) executeThenParts(h *unit.Unit, heuristicName string) (bool, bool) {
	// Snapshot before
	storeBefore := e.Store.Count()
	agendaBefore := e.Agenda.Len()

	for _, slot := range unit.ThenPartSlots() {
		prog := h.GetString(slot)
		if prog == "" {
			continue
		}
		_, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true // abort counts as output (intentional action)
			}
			e.log(3, "    %s.%s error: %v", heuristicName, slot, err)
		}
	}

	storeAfter := e.Store.Count()
	agendaAfter := e.Agenda.Len()
	produced := storeAfter > storeBefore || agendaAfter > agendaBefore

	return false, produced
}
```

- [ ] **Step 4: Update fireTaskRule and fireUnitRule to use the new return values**

In `internal/engine/fire.go`, update the callers:

```go
func (e *Engine) fireTaskRule(heuristic string, task *agenda.Task) (bool, bool) {
	// ... (existing condition checks unchanged through line 62) ...

	// All conditions passed -- execute ThenParts
	abort, produced := e.executeThenParts(h, heuristic)
	return true, abort
	// Note: produced is not used here -- it's read in WorkOnTask
}
```

Wait -- `fireTaskRule` returns (fired, abort). We need to also return `produced`. Change the return signature:

```go
// fireTaskRule fires a heuristic's IfTaskParts against a task, then executes ThenParts if all pass.
// Returns (fired, abort, producedOutput).
func (e *Engine) fireTaskRule(heuristic string, task *agenda.Task) (bool, bool, bool) {
	h := e.Store.Get(heuristic)
	if h == nil {
		return false, false, false
	}

	e.VM.SetEnv("ArgU", dsl.StringVal(task.UnitName))

	if prog := h.GetString("ifPotentiallyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, true
			}
			e.log(3, "    %s.ifPotentiallyRelevant error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	if prog := h.GetString("ifTrulyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, true
			}
			e.log(3, "    %s.ifTrulyRelevant error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	if prog := h.GetString("ifWorkingOnTask"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, true
			}
			e.log(3, "    %s.ifWorkingOnTask error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	abort, produced := e.executeThenParts(h, heuristic)
	return true, abort, produced
}

// fireUnitRule fires a heuristic against a unit (Level 2: when agenda is empty).
// Returns (fired, abort, producedOutput).
func (e *Engine) fireUnitRule(heuristic string, targetUnit string) (bool, bool, bool) {
	h := e.Store.Get(heuristic)
	if h == nil {
		return false, false, false
	}

	e.VM.SetEnv("ArgU", dsl.StringVal(targetUnit))
	e.VM.SetEnv("CurUnit", dsl.StringVal(targetUnit))

	if prog := h.GetString("ifPotentiallyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, true
			}
			e.log(3, "    %s.ifPotentiallyRelevant error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	if prog := h.GetString("ifTrulyRelevant"); prog != "" {
		v, err := e.VM.Execute(prog)
		if err != nil {
			if dsl.IsAbort(err) {
				return true, true, true
			}
			e.log(3, "    %s.ifTrulyRelevant error: %v", heuristic, err)
			return false, false, false
		}
		if !v.Truthy() {
			return false, false, false
		}
	}

	abort, produced := e.executeThenParts(h, heuristic)
	return true, abort, produced
}
```

- [ ] **Step 5: Update WorkOnTask and WorkOnUnit to record success/failure based on output**

In `internal/engine/engine.go`:

```go
func (e *Engine) WorkOnTask(task *agenda.Task) {
	e.TaskNum++

	e.VM.SetEnv("CurUnit", dsl.StringVal(task.UnitName))
	e.VM.SetEnv("CurSlot", dsl.StringVal(task.SlotName))
	e.VM.SetEnv("CurPri", dsl.IntVal(task.Priority))
	e.VM.SetEnv("TaskNum", dsl.IntVal(e.TaskNum))

	heuristics := e.Store.Examples("Heuristic")
	for _, h := range heuristics {
		fired, abort, produced := e.fireTaskRule(h, task)
		if fired {
			e.trackApplics(h, task.UnitName, produced)
			e.log(2, "  Heuristic %s fired on task %s.%s (output=%v)", h, task.UnitName, task.SlotName, produced)
		}
		if abort {
			e.log(2, "  Task aborted by %s", h)
			return
		}
	}
}

func (e *Engine) WorkOnUnit(u string) {
	e.VM.SetEnv("CurUnit", dsl.StringVal(u))
	e.VM.SetEnv("ArgU", dsl.StringVal(u))

	heuristics := e.Store.Examples("Heuristic")
	for _, h := range heuristics {
		fired, _, produced := e.fireUnitRule(h, u)
		if fired {
			e.trackApplics(h, u, produced)
			e.log(2, "  Heuristic %s fired on unit %s (output=%v)", h, u, produced)
		}
	}
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestTrackApplicsNoOpFailure -v`
Expected: PASS

- [ ] **Step 7: Run all engine tests to check for regressions**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -v`
Expected: All pass

- [ ] **Step 8: Commit**

```bash
cd /Users/chazu/dev/go/nous
git add internal/engine/fire.go internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: detect no-op heuristic firings and record as failures

executeThenParts now snapshots store size and agenda size before/after
ThenParts execution. If neither grew, the firing is recorded as a
failure in trackApplics. This gives the self-modification loop accurate
data on which heuristics produce output vs fire uselessly."
```

---

### Task 2: Deferred Failure on Unit Death

When a unit dies, record a failure in each creditor heuristic's applics.

**Files:**
- Modify: `internal/engine/credit.go:98-127` (HandleDeletedUnit)
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestTrackApplicsDeferredFailure(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// A heuristic that will get deferred failure
	h := unit.New("H-Creator")
	h.SetWorth(600)
	h.Set("isA", []string{"Heuristic"})
	h.Set("overallRecord", map[string]any{"successes": 5, "failures": 0})
	store.Put(h)

	// A unit created by that heuristic
	u := unit.New("BadUnit")
	u.SetWorth(50)
	u.Set("creditors", []string{"H-Creator"})
	u.Set("isA", []string{"Set"})
	store.Put(u)

	// Simulate unit death
	eng.VM.DeletedSnapshots = map[string]map[string]any{
		"BadUnit": {
			"worth":     50,
			"creditors": []string{"H-Creator"},
			"isA":       []string{"Set"},
		},
	}
	store.Delete("BadUnit")

	eng.HandleDeletedUnit("BadUnit")

	record := h.GetMap("overallRecord")
	if toInt(record["failures"]) != 1 {
		t.Errorf("expected 1 deferred failure, got %d", toInt(record["failures"]))
	}
	// Existing successes should be preserved
	if toInt(record["successes"]) != 5 {
		t.Errorf("expected 5 successes preserved, got %d", toInt(record["successes"]))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestTrackApplicsDeferredFailure -v`
Expected: FAIL (failures=0 because HandleDeletedUnit doesn't call trackApplics)

- [ ] **Step 3: Add trackApplics call to HandleDeletedUnit**

In `internal/engine/credit.go`, add after line 123 (`e.punishCreators`):

```go
func (e *Engine) HandleDeletedUnit(unitName string) {
	snapshot := e.VM.DeletedSnapshots[unitName]
	if snapshot == nil {
		return
	}

	creditors, _ := snapshot["creditors"].([]string)
	isA, _ := snapshot["isA"].([]string)
	worth := toInt(snapshot["worth"])

	grave := GraveRecord{
		Name:      unitName,
		IsA:       isA,
		Creditors: creditors,
		Worth:     worth,
		Slots:     snapshot,
		Cycle:     e.cycle,
	}
	e.Graveyard = append(e.Graveyard, grave)

	e.log(1, "  Unit %s killed (was worth %d, creditors: %v)", unitName, worth, creditors)

	// Punish creditors
	e.punishCreators(unitName, snapshot)

	// Record deferred failure in each creditor's applics
	for _, creditor := range creditors {
		if e.Store.Get(creditor) != nil {
			e.trackApplics(creditor, unitName, false)
		}
	}

	// HindSight: create an avoidance heuristic
	e.createAvoidanceRule(grave)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestTrackApplicsDeferredFailure -v`
Expected: PASS

- [ ] **Step 5: Run all engine tests**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -v`
Expected: All pass

- [ ] **Step 6: Commit**

```bash
cd /Users/chazu/dev/go/nous
git add internal/engine/credit.go internal/engine/engine_test.go
git commit -m "feat: record deferred failure in creditor applics when units die

When HandleDeletedUnit processes a killed unit, each creditor heuristic
now gets a failure recorded in its applics via trackApplics. This closes
the deferred feedback path: heuristics that create units that later die
accumulate failure records proportional to their kill rate."
```

---

### Task 3: Performance-Based Mutation Trigger

Replace time-based mutation with performance-based: scan heuristics for low success ratios and mutate the worst performer.

**Files:**
- Modify: `internal/engine/mutation.go:13-19` (MutationConfig), `34-114` (tryMutateHeuristic)
- Modify: `internal/engine/engine.go:109-112` (mutation trigger in Run)
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestPerformanceBasedMutation(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	anything := unit.New("Anything")
	anything.Set("isA", []string{})
	store.Put(anything)

	heuristic := unit.New("Heuristic")
	heuristic.Set("isA", []string{"Anything"})
	store.Put(heuristic)

	// A heuristic with terrible success ratio (2/12 = 0.17)
	bad := unit.New("H-Bad")
	bad.SetWorth(500)
	bad.Set("isA", []string{"Heuristic", "Anything"})
	bad.Set("overallRecord", map[string]any{"successes": 2, "failures": 10})
	bad.Set("thenCompute", `1 drop`)
	store.Put(bad)

	// A heuristic with good success ratio (8/10 = 0.80)
	good := unit.New("H-Good")
	good.SetWorth(500)
	good.Set("isA", []string{"Heuristic", "Anything"})
	good.Set("overallRecord", map[string]any{"successes": 8, "failures": 2})
	good.Set("thenCompute", `1 drop`)
	store.Put(good)

	eng.MutConfig.Enabled = true
	eng.MutConfig.MinApplics = 10
	eng.MutConfig.MutationThreshold = 0.3

	eng.tryMutateHeuristic()

	// Should have created a mutant of H-Bad, not H-Good
	found := false
	for _, name := range store.All() {
		u := store.Get(name)
		if u != nil && u.GetString("mutant_of") == "H-Bad" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a mutant of H-Bad (worst performer) to be created")
	}

	// Should NOT have mutated H-Good
	for _, name := range store.All() {
		u := store.Get(name)
		if u != nil && u.GetString("mutant_of") == "H-Good" {
			t.Error("should not have mutated H-Good (success ratio above threshold)")
		}
	}
}

func TestNoMutationWhenAllAdequate(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	anything := unit.New("Anything")
	anything.Set("isA", []string{})
	store.Put(anything)

	heuristic := unit.New("Heuristic")
	heuristic.Set("isA", []string{"Anything"})
	store.Put(heuristic)

	// All heuristics performing well
	good := unit.New("H-Good")
	good.SetWorth(500)
	good.Set("isA", []string{"Heuristic", "Anything"})
	good.Set("overallRecord", map[string]any{"successes": 8, "failures": 2})
	good.Set("thenCompute", `1 drop`)
	store.Put(good)

	eng.MutConfig.Enabled = true
	eng.MutConfig.MinApplics = 10
	eng.MutConfig.MutationThreshold = 0.3

	countBefore := store.Count()
	eng.tryMutateHeuristic()
	countAfter := store.Count()

	if countAfter != countBefore {
		t.Error("should not have created any mutants when all heuristics are adequate")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run "TestPerformanceBasedMutation|TestNoMutationWhenAllAdequate" -v`
Expected: FAIL (MinApplics and MutationThreshold fields don't exist, current logic picks random by worth)

- [ ] **Step 3: Add new fields to MutationConfig**

In `internal/engine/mutation.go`:

```go
type MutationConfig struct {
	Enabled           bool
	Interval          int     // check for mutation candidates every N cycles
	MaxMutants        int     // max live mutant heuristics at once
	MutantWorth       int     // starting worth for mutant heuristics
	ValidateOnly      bool    // if true, only keep mutations that pass validation
	MinApplics        int     // minimum total firings before eligible for performance eval
	MutationThreshold float64 // success ratio below which mutation triggers
}

func DefaultMutationConfig() MutationConfig {
	return MutationConfig{
		Enabled:           true,
		Interval:          10,
		MaxMutants:        20,
		MutantWorth:       400,
		ValidateOnly:      true,
		MinApplics:        10,
		MutationThreshold: 0.3,
	}
}
```

- [ ] **Step 4: Replace pickHeuristicByWorth with pickWorstPerformer**

In `internal/engine/mutation.go`, replace `pickHeuristicByWorth` with:

```go
// pickWorstPerformer finds the heuristic with the lowest success ratio
// among those with enough applics data. Returns nil if no candidates.
func (e *Engine) pickWorstPerformer() *unit.Unit {
	heuristics := e.Store.Examples("Heuristic")

	type candidate struct {
		name  string
		ratio float64
		worth int
	}
	var candidates []candidate

	for _, name := range heuristics {
		u := e.Store.Get(name)
		if u == nil || name == "Heuristic" {
			continue
		}
		record := u.GetMap("overallRecord")
		if record == nil {
			continue
		}
		s := toInt(record["successes"])
		f := toInt(record["failures"])
		total := s + f
		if total < e.MutConfig.MinApplics {
			continue
		}
		ratio := float64(s) / float64(total)
		if ratio >= e.MutConfig.MutationThreshold {
			continue
		}
		candidates = append(candidates, candidate{name, ratio, u.Worth()})
	}

	if len(candidates) == 0 {
		return nil
	}

	// Pick worst: lowest ratio, then lowest worth on tie
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ratio != candidates[j].ratio {
			return candidates[i].ratio < candidates[j].ratio
		}
		return candidates[i].worth < candidates[j].worth
	})

	return e.Store.Get(candidates[0].name)
}
```

- [ ] **Step 5: Update tryMutateHeuristic to use pickWorstPerformer**

In `internal/engine/mutation.go`, replace the `parent := e.pickHeuristicByWorth()` call:

```go
func (e *Engine) tryMutateHeuristic() {
	if e.mutator == nil || e.MutConfig.MaxMutants <= 0 {
		return
	}

	// Count existing mutants
	mutantCount := 0
	for _, name := range e.Store.All() {
		u := e.Store.Get(name)
		if u != nil && u.GetString("mutant_of") != "" {
			mutantCount++
		}
	}
	if mutantCount >= e.MutConfig.MaxMutants {
		e.log(2, "  Mutation: at mutant cap (%d/%d)", mutantCount, e.MutConfig.MaxMutants)
		return
	}

	// Pick worst performer (performance-based, not random)
	parent := e.pickWorstPerformer()
	if parent == nil {
		e.log(2, "  Mutation: no candidates below threshold %.2f", e.MutConfig.MutationThreshold)
		return
	}

	// Pick a program slot to mutate
	slot, prog := e.pickProgramSlot(parent)
	if prog == "" {
		return
	}

	// Apply mutation
	mutated, op := e.mutator.Mutate(prog)
	if op == nil {
		return
	}

	// Validate
	if e.MutConfig.ValidateOnly && !mutate.Validate(mutated, e.Store) {
		e.log(3, "  Mutation: invalid mutant of %s.%s (%s), discarded", parent.Name, slot, op.Kind)
		return
	}

	// Create the mutant heuristic
	mutantName := fmt.Sprintf("M-%s-%d", parent.Name, e.cycle)
	if e.Store.Has(mutantName) {
		return
	}

	m := unit.New(mutantName)
	m.SetWorth(e.MutConfig.MutantWorth)
	m.Set("isA", []string{"Heuristic", "MutantHeuristic", "Anything"})
	// Inherit parent's creditors plus the parent itself
	parentCreditors := parent.GetStrings("creditors")
	mutantCreditors := append([]string{parent.Name}, parentCreditors...)
	m.Set("creditors", mutantCreditors)
	m.Set("mutant_of", parent.Name)
	m.Set("mutation_op", op.Kind)
	m.Set("mutation_slot", slot)
	m.Set("mutation_from", op.From)
	m.Set("mutation_to", op.To)
	m.Set("mutation_cycle", e.cycle)
	m.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})

	// Copy all program slots from parent, replacing the mutated one
	for _, s := range programSlots() {
		p := parent.GetString(s)
		if p != "" {
			if s == slot {
				m.Set(s, mutated)
			} else {
				m.Set(s, p)
			}
		}
	}

	if parent.GetString("english") != "" {
		m.Set("english", fmt.Sprintf("Mutant of %s (%s %s.%s: %s->%s)",
			parent.Name, op.Kind, parent.Name, slot, op.From, op.To))
	}

	e.Store.Put(m)
	e.log(1, "  Mutation: created %s (%s on %s.%s: %s -> %s)",
		mutantName, op.Kind, parent.Name, slot, op.From, op.To)
}
```

Note: keep `pickHeuristicByWorth` and `pickProgramSlot` -- `pickProgramSlot` is still used.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run "TestPerformanceBasedMutation|TestNoMutationWhenAllAdequate" -v`
Expected: PASS

- [ ] **Step 7: Run all engine tests**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -v`
Expected: All pass

- [ ] **Step 8: Commit**

```bash
cd /Users/chazu/dev/go/nous
git add internal/engine/mutation.go internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: replace time-based mutation with performance-based trigger

Mutation now scans heuristic applics for low success ratios instead of
picking random heuristics by worth. pickWorstPerformer replaces
pickHeuristicByWorth. Heuristics need MinApplics (default 10) firings
before being evaluated, and are only mutated if success ratio falls
below MutationThreshold (default 0.3)."
```

---

### Task 4: Worth-Growth Reward

Wire rewardCreators to fire when units' worth grows above their creation baseline.

**Files:**
- Modify: `internal/engine/credit.go` (add rewardForWorthGrowth method)
- Modify: `internal/engine/engine.go:74-117` (call reward in Run loop)
- Modify: `internal/dsl/builtins.go:277-301` (set creationWorth in create-unit)
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestWorthGrowthReward(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// A creator heuristic
	creator := unit.New("H-Creator")
	creator.SetWorth(500)
	creator.Set("isA", []string{"Heuristic"})
	store.Put(creator)

	// A unit created by that heuristic, starting at worth 400
	child := unit.New("ChildUnit")
	child.SetWorth(400)
	child.Set("isA", []string{"Set"})
	child.Set("creditors", []string{"H-Creator"})
	child.Set("creationWorth", 400)
	child.Set("lastRewardedWorth", 400)
	store.Put(child)

	// Simulate worth growth (domain heuristics boosted it to 600)
	child.SetWorth(600)

	eng.rewardForWorthGrowth()

	// Creator should have been rewarded: delta = 600 - 400 = 200, reward = 200/2 = 100
	if creator.Worth() != 600 {
		t.Errorf("expected creator worth boosted to 600, got %d", creator.Worth())
	}

	// lastRewardedWorth should be updated
	if child.GetInt("lastRewardedWorth") != 600 {
		t.Errorf("expected lastRewardedWorth updated to 600, got %d", child.GetInt("lastRewardedWorth"))
	}

	// Running again with no further growth should not reward again
	eng.rewardForWorthGrowth()
	if creator.Worth() != 600 {
		t.Errorf("expected no double reward, creator worth should stay 600, got %d", creator.Worth())
	}
}

func TestWorthGrowthRewardSkipsHeuristics(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// A heuristic that was created by another heuristic
	meta := unit.New("H-Meta")
	meta.SetWorth(500)
	meta.Set("isA", []string{"Heuristic"})
	store.Put(meta)

	child := unit.New("H-Child")
	child.SetWorth(700)
	child.Set("isA", []string{"Heuristic"})
	child.Set("creditors", []string{"H-Meta"})
	child.Set("creationWorth", 400)
	child.Set("lastRewardedWorth", 400)
	store.Put(child)

	eng.rewardForWorthGrowth()

	// H-Meta should NOT be rewarded (child is a Heuristic, skip)
	if meta.Worth() != 500 {
		t.Errorf("expected no reward for heuristic-created-heuristic, meta worth should stay 500, got %d", meta.Worth())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run "TestWorthGrowthReward" -v`
Expected: FAIL (rewardForWorthGrowth does not exist)

- [ ] **Step 3: Implement rewardForWorthGrowth**

In `internal/engine/credit.go`, add:

```go
// rewardForWorthGrowth scans non-heuristic units for worth growth above
// their creation baseline and rewards their creditors proportionally.
func (e *Engine) rewardForWorthGrowth() {
	for _, name := range e.Store.All() {
		u := e.Store.Get(name)
		if u == nil {
			continue
		}
		// Skip heuristics (they're evaluated by applics, not worth growth)
		if e.Store.IsA(name, "Heuristic") {
			continue
		}
		// Skip units without creditors (seed units)
		creditors := u.GetStrings("creditors")
		if len(creditors) == 0 {
			continue
		}
		// Skip units without creation tracking
		if !u.Has("creationWorth") {
			continue
		}

		lastRewarded := u.GetInt("lastRewardedWorth")
		currentWorth := u.Worth()

		if currentWorth > lastRewarded {
			delta := currentWorth - lastRewarded
			reward := delta / 2
			if reward > 0 {
				e.rewardCreators(name, reward)
				u.Set("lastRewardedWorth", currentWorth)
				e.log(2, "  Reward: %s grew %d->%d, rewarding creditors +%d",
					name, lastRewarded, currentWorth, reward)
			}
		}
	}
}
```

- [ ] **Step 4: Set creationWorth and lastRewardedWorth in create-unit builtin**

In `internal/dsl/builtins.go`, update `bCreateUnit`:

```go
func bCreateUnit(vm *VM) error {
	parentCategory := vm.pop()
	name := vm.pop()
	nameStr := name.AsString()
	u := vm.Store.Get(nameStr)
	if u != nil {
		// Already exists
		vm.push(StringVal(nameStr))
		return nil
	}
	u = &unit.Unit{
		Name:  nameStr,
		Slots: map[string]any{},
	}
	parent := parentCategory.AsString()
	if parent != "" {
		u.Set("isA", []string{parent})
	}
	u.Set("worth", 500) // default worth for new units
	u.Set("creationWorth", 500)
	u.Set("lastRewardedWorth", 500)
	u.Set("isNew", true)
	vm.Store.Put(u)
	vm.NewUnits = append(vm.NewUnits, nameStr)
	vm.push(StringVal(nameStr))
	return nil
}
```

- [ ] **Step 5: Wire rewardForWorthGrowth into the Run loop**

In `internal/engine/engine.go`, add after the mutation block in `Run`:

```go
		// Periodic mutation
		if e.MutConfig.Enabled && e.cycle > 0 && e.cycle%e.MutConfig.Interval == 0 {
			e.tryMutateHeuristic()
		}

		// Periodic worth-growth reward (every 10 cycles)
		if e.cycle > 0 && e.cycle%10 == 0 {
			e.rewardForWorthGrowth()
		}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run "TestWorthGrowthReward" -v`
Expected: PASS

- [ ] **Step 7: Run all tests**

Run: `cd /Users/chazu/dev/go/nous && go test ./... -v`
Expected: All pass

- [ ] **Step 8: Commit**

```bash
cd /Users/chazu/dev/go/nous
git add internal/engine/credit.go internal/engine/engine.go internal/dsl/builtins.go internal/engine/engine_test.go
git commit -m "feat: wire worth-growth reward to close positive credit loop

rewardForWorthGrowth scans non-heuristic units every 10 cycles for
worth growth above their creation baseline. Creditors receive half
the delta as a worth boost. High-water mark prevents double-dipping.
create-unit now sets creationWorth and lastRewardedWorth slots."
```

---

### Task 5: DSL Builtins for Applics Inspection

Add builtins so meta-heuristics can inspect applics data from within DSL programs.

**Files:**
- Modify: `internal/dsl/builtins.go` (add 3 builtins)
- Test: `internal/dsl/vm_test.go` (or wherever DSL tests live)

- [ ] **Step 1: Check where DSL tests live**

Run: `cd /Users/chazu/dev/go/nous && ls internal/dsl/*_test.go`

- [ ] **Step 2: Write failing tests for new builtins**

```go
func TestApplicsSuccessRatio(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag)
	buf := &bytes.Buffer{}
	vm.Out = buf

	h := unit.New("H-Test")
	h.Set("overallRecord", map[string]any{"successes": 7, "failures": 3})
	store.Put(h)

	v, err := vm.Execute(`"H-Test" applics-success-ratio`)
	if err != nil {
		t.Fatal(err)
	}
	ratio := v.AsFloat()
	if ratio < 0.69 || ratio > 0.71 {
		t.Errorf("expected ratio ~0.7, got %f", ratio)
	}
}

func TestApplicsSuccessRatioNoRecord(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag)
	buf := &bytes.Buffer{}
	vm.Out = buf

	h := unit.New("H-Empty")
	store.Put(h)

	v, err := vm.Execute(`"H-Empty" applics-success-ratio`)
	if err != nil {
		t.Fatal(err)
	}
	// No record = 0.0
	if v.AsFloat() != 0 {
		t.Errorf("expected 0.0 for no record, got %f", v.AsFloat())
	}
}

func TestGetApplics(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag)
	buf := &bytes.Buffer{}
	vm.Out = buf

	h := unit.New("H-Test")
	h.Set("applics", []map[string]any{
		{"target": "UnitA", "result": true},
		{"target": "UnitB", "result": false},
	})
	store.Put(h)

	v, err := vm.Execute(`"H-Test" get-applics`)
	if err != nil {
		t.Fatal(err)
	}
	// Should return the count (applics is []map[string]any -> IntVal via anyToValue)
	if v.AsInt() != 2 {
		t.Errorf("expected applics count 2, got %d", v.AsInt())
	}
}

func TestApplicsByType(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	vm := NewVM(store, ag)
	buf := &bytes.Buffer{}
	vm.Out = buf

	// Set up units with types
	setUnit := unit.New("UnitA")
	setUnit.Set("isA", []string{"Set"})
	store.Put(setUnit)

	numUnit := unit.New("UnitB")
	numUnit.Set("isA", []string{"Number"})
	store.Put(numUnit)

	h := unit.New("H-Test")
	h.Set("applics", []map[string]any{
		{"target": "UnitA", "result": true},
		{"target": "UnitA", "result": true},
		{"target": "UnitB", "result": false},
	})
	store.Put(h)

	v, err := vm.Execute(`"H-Test" applics-by-type`)
	if err != nil {
		t.Fatal(err)
	}
	// Returns a string representation of the map
	s := v.AsString()
	if s == "" || s == "nil" {
		t.Error("expected non-empty applics-by-type result")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -run "TestApplics" -v`
Expected: FAIL (builtins don't exist)

- [ ] **Step 4: Implement the builtins**

In `internal/dsl/builtins.go`, add to the `builtins` map:

```go
	// Applics inspection (for meta-heuristics)
	"get-applics":          bGetApplics,
	"applics-success-ratio": bApplicsSuccessRatio,
	"applics-by-type":      bApplicsByType,
```

And the implementations:

```go
// Applics inspection builtins

func bGetApplics(vm *VM) error {
	name := vm.pop()
	u := vm.Store.Get(name.AsString())
	if u == nil {
		vm.push(Nil())
		return nil
	}
	raw := u.Get("applics")
	vm.push(anyToValue(raw))
	return nil
}

func bApplicsSuccessRatio(vm *VM) error {
	name := vm.pop()
	u := vm.Store.Get(name.AsString())
	if u == nil {
		vm.push(FloatVal(0))
		return nil
	}
	record := u.GetMap("overallRecord")
	if record == nil {
		vm.push(FloatVal(0))
		return nil
	}
	s := toIntDSL(record["successes"])
	f := toIntDSL(record["failures"])
	total := s + f
	if total == 0 {
		vm.push(FloatVal(0))
		return nil
	}
	vm.push(FloatVal(float64(s) / float64(total)))
	return nil
}

func bApplicsByType(vm *VM) error {
	name := vm.pop()
	u := vm.Store.Get(name.AsString())
	if u == nil {
		vm.push(Nil())
		return nil
	}
	applics, _ := u.Get("applics").([]map[string]any)
	if len(applics) == 0 {
		vm.push(Nil())
		return nil
	}

	// Group by target unit's first isA type
	type counts struct {
		successes int
		failures  int
	}
	byType := make(map[string]*counts)

	for _, a := range applics {
		target, _ := a["target"].(string)
		result, _ := a["result"].(bool)
		targetUnit := vm.Store.Get(target)
		typeName := "unknown"
		if targetUnit != nil {
			isA := targetUnit.GetStrings("isA")
			if len(isA) > 0 {
				typeName = isA[0]
			}
		}
		c, ok := byType[typeName]
		if !ok {
			c = &counts{}
			byType[typeName] = c
		}
		if result {
			c.successes++
		} else {
			c.failures++
		}
	}

	// Convert to map[string]any for anyToValue
	result := make(map[string]any)
	for typ, c := range byType {
		result[typ] = map[string]any{"s": c.successes, "f": c.failures}
	}
	vm.push(anyToValue(result))
	return nil
}

func toIntDSL(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	default:
		return 0
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -run "TestApplics" -v`
Expected: PASS

- [ ] **Step 6: Run all DSL tests**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/dsl/ -v`
Expected: All pass

- [ ] **Step 7: Commit**

```bash
cd /Users/chazu/dev/go/nous
git add internal/dsl/builtins.go internal/dsl/*_test.go
git commit -m "feat: add applics inspection builtins for meta-heuristics

Three new DSL words: applics-success-ratio (returns float),
get-applics (returns applics list), applics-by-type (groups
success/failure by target unit's isA type). These enable
meta-heuristics to reason about heuristic performance from
within the DSL."
```

---

### Task 6: H-AnalyzeApplics Meta-Heuristic

Add a seed meta-heuristic that detects type-skewed success patterns and creates specialized copies.

**Files:**
- Modify: `internal/seed/heuristics.go` (add hAnalyzeApplics)
- Modify: `internal/seed/observations.go:17-23` (register in observation domain)
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestHAnalyzeApplics(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf
	eng.MutConfig.Enabled = false

	// Type hierarchy
	anything := unit.New("Anything")
	anything.Set("isA", []string{})
	store.Put(anything)

	heuristic := unit.New("Heuristic")
	heuristic.Set("isA", []string{"Anything"})
	store.Put(heuristic)

	setType := unit.New("Set")
	setType.Set("isA", []string{"Anything"})
	store.Put(setType)

	numType := unit.New("Number")
	numType.Set("isA", []string{"Anything"})
	store.Put(numType)

	// A heuristic with type-skewed applics: succeeds on Sets, fails on Numbers
	target := unit.New("H-Skewed")
	target.SetWorth(500)
	target.Set("isA", []string{"Heuristic", "Anything"})
	target.Set("overallRecord", map[string]any{"successes": 8, "failures": 6})
	target.Set("thenCompute", `1 drop`)
	target.Set("ifPotentiallyRelevant", `true`)
	// Applics: 8 successes on Set targets, 6 failures on Number targets
	applics := make([]map[string]any, 0)
	for i := 0; i < 8; i++ {
		name := fmt.Sprintf("SetUnit%d", i)
		u := unit.New(name)
		u.Set("isA", []string{"Set", "Anything"})
		store.Put(u)
		applics = append(applics, map[string]any{"target": name, "result": true})
	}
	for i := 0; i < 6; i++ {
		name := fmt.Sprintf("NumUnit%d", i)
		u := unit.New(name)
		u.Set("isA", []string{"Number", "Anything"})
		store.Put(u)
		applics = append(applics, map[string]any{"target": name, "result": false})
	}
	target.Set("applics", applics)
	store.Put(target)

	// Load the meta-heuristic
	seed.LoadHeuristics(store)

	// Focus on H-Skewed (Level 2 unit focus)
	eng.WorkOnUnit("H-Skewed")

	// Should have created a specialized version
	found := false
	for _, name := range store.All() {
		u := store.Get(name)
		if u != nil {
			creditors := u.GetStrings("creditors")
			for _, c := range creditors {
				if c == "H-AnalyzeApplics" {
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Error("expected H-AnalyzeApplics to create a specialized heuristic")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestHAnalyzeApplics -v`
Expected: FAIL (H-AnalyzeApplics doesn't exist)

- [ ] **Step 3: Implement H-AnalyzeApplics**

In `internal/seed/heuristics.go`, add `hAnalyzeApplics(s)` call to `LoadHeuristics`:

```go
func LoadHeuristics(s *unit.Store) {
	hFindExamples(s)
	hRunOnExamples(s)
	hCheckExtremes(s)
	hSpecialize(s)
	hCheckDomain(s)
	hConjecture(s)
	hExploreSlots(s)
	hKillWorthless(s)
	hPenalizeTrivial(s)
	hBoostInteresting(s)
	hAnalyzeApplics(s)
}
```

And the implementation:

```go
// H-AnalyzeApplics: Meta-heuristic that inspects other heuristics' applics
// for type-skewed success patterns and creates specialized copies.
func hAnalyzeApplics(s *unit.Store) {
	h := putHeuristic(s, "H-AnalyzeApplics", 600)
	h.Set("english", "Inspect applics for type-skewed success patterns and propose specializations")

	// Only fires on heuristics with enough data and middling success ratio
	h.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Heuristic" isa?
		"ArgU" @ "H-AnalyzeApplics" !=
		and
	`)

	h.Set("ifTrulyRelevant", `
		"ArgU" @ applics-success-ratio "ratio" !
		"ratio" @ 0.3 >=
		"ratio" @ 0.7 <=
		and
		"ArgU" @ get-applics nil !=
		and
	`)

	// Analyze applics by type and create a specialized copy if skew detected.
	// This is implemented as a Go-side helper because the DSL lacks map
	// iteration. The thenCompute calls a builtin that does the heavy lifting.
	h.Set("thenCompute", `
		"ArgU" @ analyze-and-specialize
	`)
}
```

- [ ] **Step 4: Implement the analyze-and-specialize builtin**

In `internal/dsl/builtins.go`, add to the builtins map:

```go
	"analyze-and-specialize": bAnalyzeAndSpecialize,
```

And the implementation:

```go
func bAnalyzeAndSpecialize(vm *VM) error {
	name := vm.pop()
	nameStr := name.AsString()
	u := vm.Store.Get(nameStr)
	if u == nil {
		vm.push(BoolVal(false))
		return nil
	}

	applics, _ := u.Get("applics").([]map[string]any)
	if len(applics) < 10 {
		vm.push(BoolVal(false))
		return nil
	}

	// Group by target type
	type counts struct {
		successes int
		failures  int
	}
	byType := make(map[string]*counts)

	for _, a := range applics {
		target, _ := a["target"].(string)
		result, _ := a["result"].(bool)
		targetUnit := vm.Store.Get(target)
		typeName := "unknown"
		if targetUnit != nil {
			isA := targetUnit.GetStrings("isA")
			if len(isA) > 0 {
				typeName = isA[0]
			}
		}
		c, ok := byType[typeName]
		if !ok {
			c = &counts{}
			byType[typeName] = c
		}
		if result {
			c.successes++
		} else {
			c.failures++
		}
	}

	// Find the best type (highest success rate with at least 3 data points)
	bestType := ""
	bestRatio := 0.0
	for typ, c := range byType {
		total := c.successes + c.failures
		if total < 3 || typ == "unknown" {
			continue
		}
		ratio := float64(c.successes) / float64(total)
		if ratio > bestRatio {
			bestRatio = ratio
			bestType = typ
		}
	}

	// Need clear skew: best type ratio > 0.7 and overall ratio < 0.7
	if bestType == "" || bestRatio <= 0.7 {
		vm.push(BoolVal(false))
		return nil
	}

	// Create specialized copy
	specName := nameStr + "-on-" + bestType
	if vm.Store.Has(specName) {
		vm.push(BoolVal(false))
		return nil
	}

	spec := &unit.Unit{
		Name:  specName,
		Slots: map[string]any{},
	}
	spec.Set("isA", []string{"Heuristic", "Anything"})
	spec.SetWorth(u.Worth())
	spec.Set("creditors", []string{"H-AnalyzeApplics"})
	spec.Set("specialized_from", nameStr)
	spec.Set("specialized_type", bestType)
	spec.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})

	// Copy program slots, prepending type check to ifPotentiallyRelevant
	for _, slot := range []string{
		"ifPotentiallyRelevant", "ifTrulyRelevant", "ifWorkingOnTask",
		"ifFinishedWorkingOnTask", "thenCompute", "thenAddToAgenda",
		"thenDefineNewConcepts", "thenDeleteOldConcepts", "thenPrintToUser",
		"thenConjecture",
	} {
		prog := u.GetString(slot)
		if prog == "" {
			continue
		}
		if slot == "ifPotentiallyRelevant" {
			// Prepend type check
			prog = fmt.Sprintf(`"ArgU" @ "%s" isa? `, bestType) + prog + " and"
		}
		spec.Set(slot, prog)
	}

	if u.GetString("english") != "" {
		spec.Set("english", fmt.Sprintf("Specialized %s for %s targets", nameStr, bestType))
	}

	vm.Store.Put(spec)
	vm.NewUnits = append(vm.NewUnits, specName)
	vm.push(BoolVal(true))
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestHAnalyzeApplics -v`
Expected: PASS

- [ ] **Step 6: Register H-AnalyzeApplics in observation domain too**

In `internal/seed/observations.go`, add to `LoadObservationHeuristics`:

```go
func LoadObservationHeuristics(s *unit.Store) {
	hFindScopeHotspots(s)
	hCorroborateObstacles(s)
	hConjectureFromPatterns(s)
	hBoostCorroborated(s)
	hPenalizeStaleObservations(s)
	hAnalyzeApplics(s)
}
```

- [ ] **Step 7: Run all tests**

Run: `cd /Users/chazu/dev/go/nous && go test ./... -v`
Expected: All pass

- [ ] **Step 8: Commit**

```bash
cd /Users/chazu/dev/go/nous
git add internal/seed/heuristics.go internal/seed/observations.go internal/dsl/builtins.go internal/engine/engine_test.go
git commit -m "feat: add H-AnalyzeApplics meta-heuristic

Inspects other heuristics' applics for type-skewed success patterns.
When a heuristic succeeds mostly on one type and fails on others,
creates a specialized copy with a type guard in ifPotentiallyRelevant.
The specialization credits H-AnalyzeApplics, closing the meta-credit
loop. Registered in both math and observation domains."
```

---

### Task 7: HindSight Validation

Validate generated HAvoid programs, track effectiveness, promote/demote based on firing history.

**Files:**
- Modify: `internal/engine/credit.go:134-197` (createAvoidanceRule)
- Modify: `internal/engine/engine.go` (add demotion check)
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the failing test for DSL validation on creation**

```go
func TestHAvoidValidation(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// Create a grave record
	grave := GraveRecord{
		Name:      "BadUnit",
		IsA:       []string{"Set"},
		Creditors: []string{"H-Creator"},
		Worth:     50,
		Cycle:     5,
	}

	// Set type
	setType := unit.New("Set")
	setType.Set("isA", []string{"Anything"})
	store.Put(setType)

	eng.createAvoidanceRule(grave)

	avoid := store.Get("HAvoid-BadUnit")
	if avoid == nil {
		t.Fatal("expected HAvoid-BadUnit to be created")
	}

	// Should start at worth 300
	if avoid.Worth() != 300 {
		t.Errorf("expected HAvoid worth 300, got %d", avoid.Worth())
	}

	// The ifPotentiallyRelevant program should tokenize cleanly
	prog := avoid.GetString("ifPotentiallyRelevant")
	if prog == "" {
		t.Fatal("expected ifPotentiallyRelevant to be set")
	}
	tokens := dsl.Tokenize(prog)
	if len(tokens) == 0 {
		t.Error("expected non-empty token list from HAvoid program")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestHAvoidValidation -v`
Expected: FAIL (HAvoid starts at worth 600, not 300)

- [ ] **Step 3: Update createAvoidanceRule with validation and correct starting worth**

In `internal/engine/credit.go`, update `createAvoidanceRule`:

```go
func (e *Engine) createAvoidanceRule(grave GraveRecord) {
	if len(grave.Creditors) == 0 {
		return
	}

	avoidName := "HAvoid-" + sanitizeName(grave.Name)
	if e.Store.Has(avoidName) {
		return
	}

	failedType := "Anything"
	if len(grave.IsA) > 0 {
		failedType = grave.IsA[0]
	}

	creditor := grave.Creditors[0]

	ifProg := fmt.Sprintf(`
		"ArgU" @ "%s" isa?
		"ArgU" @ "creditors" get-slot nil !=
		and
		if
			"ArgU" @ "creditors" get-slot
			each
				it "cred" !
				"cred" @ "%s" =
				if
					abort
				then
			end
		then
		false
	`, failedType, creditor)

	// Validate: dry-run parse through tokenizer
	tokens := dsl.Tokenize(ifProg)
	if len(tokens) == 0 {
		e.log(1, "  HindSight: discarded invalid HAvoid program for %s", grave.Name)
		return
	}

	avoid := unit.New(avoidName)
	avoid.SetWorth(300) // unproven -- start low
	avoid.Set("isA", []string{"Heuristic", "HAvoidRule", "Anything"})
	avoid.Set("english", fmt.Sprintf("Avoid: %s creating %s-type units (learned from %s dying)",
		creditor, failedType, grave.Name))
	avoid.Set("creditors", []string{"H-HindSight"})
	avoid.Set("ifPotentiallyRelevant", ifProg)
	avoid.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	avoid.Set("avoidance_of", grave.Name)
	avoid.Set("avoidance_creditor", creditor)
	avoid.Set("avoidance_type", failedType)
	avoid.Set("creationCycle", e.cycle)

	e.Store.Put(avoid)
	e.log(1, "  HindSight: created %s (blocks %s from making %s-type units)",
		avoidName, creditor, failedType)
}
```

- [ ] **Step 4: Write test for HAvoid promotion after 3 firings**

```go
func TestHAvoidPromotion(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	avoid := unit.New("HAvoid-Test")
	avoid.SetWorth(300)
	avoid.Set("isA", []string{"Heuristic", "HAvoidRule", "Anything"})
	avoid.Set("overallRecord", map[string]any{"successes": 3, "failures": 0})
	avoid.Set("creationCycle", 0)
	store.Put(avoid)

	eng.promoteOrDemoteHAvoidRules()

	if avoid.Worth() != 600 {
		t.Errorf("expected HAvoid promoted to 600 after 3 successes, got %d", avoid.Worth())
	}
}

func TestHAvoidDemotion(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	eng.cycle = 250
	buf := &bytes.Buffer{}
	eng.Out = buf

	avoid := unit.New("HAvoid-Stale")
	avoid.SetWorth(300)
	avoid.Set("isA", []string{"Heuristic", "HAvoidRule", "Anything"})
	avoid.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	avoid.Set("creationCycle", 10)
	store.Put(avoid)

	eng.promoteOrDemoteHAvoidRules()

	if avoid.Worth() != 100 {
		t.Errorf("expected HAvoid demoted to 100 after 200+ idle cycles, got %d", avoid.Worth())
	}
}
```

- [ ] **Step 5: Run tests to verify they fail**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run "TestHAvoid" -v`
Expected: FAIL (promoteOrDemoteHAvoidRules doesn't exist)

- [ ] **Step 6: Implement promoteOrDemoteHAvoidRules**

In `internal/engine/credit.go`, add:

```go
// promoteOrDemoteHAvoidRules evaluates HAvoid rule effectiveness.
// Rules with 3+ successful firings get promoted to worth 600.
// Rules idle for 200+ cycles get demoted to worth 100.
func (e *Engine) promoteOrDemoteHAvoidRules() {
	for _, name := range e.Store.Examples("HAvoidRule") {
		u := e.Store.Get(name)
		if u == nil {
			continue
		}

		record := u.GetMap("overallRecord")
		if record == nil {
			continue
		}

		successes := toInt(record["successes"])
		creationCycle := u.GetInt("creationCycle")
		age := e.cycle - creationCycle

		// Promote: 3+ successful firings
		if successes >= 3 && u.Worth() < 600 {
			u.SetWorth(600)
			e.log(1, "  HAvoid: promoted %s to 600 (proven useful, %d firings)", name, successes)
			continue
		}

		// Demote: 200+ cycles with zero firings
		total := successes + toInt(record["failures"])
		if age >= 200 && total == 0 && u.Worth() > 100 {
			u.SetWorth(100)
			e.log(1, "  HAvoid: demoted %s to 100 (idle for %d cycles)", name, age)
		}
	}
}
```

- [ ] **Step 7: Wire promoteOrDemoteHAvoidRules into the Run loop**

In `internal/engine/engine.go`, add after the reward block:

```go
		// Periodic HAvoid evaluation (every 50 cycles)
		if e.cycle > 0 && e.cycle%50 == 0 {
			e.promoteOrDemoteHAvoidRules()
		}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run "TestHAvoid" -v`
Expected: PASS

- [ ] **Step 9: Run all tests**

Run: `cd /Users/chazu/dev/go/nous && go test ./... -v`
Expected: All pass

- [ ] **Step 10: Commit**

```bash
cd /Users/chazu/dev/go/nous
git add internal/engine/credit.go internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: validate HAvoid rules and track effectiveness

HAvoid rules now start at worth 300 (unproven) and get promoted to
600 after 3+ successful firings, or demoted to 100 after 200 idle
cycles. Generated DSL programs are validated through the tokenizer
before being stored. The demotion path means HAvoid rules that
reach worth 0 get killed by H-KillWorthless, triggering recursive
HindSight (HAvoid-HAvoid)."
```

---

### Task 8: End-to-End Self-Modification Test

Verify the full loop: heuristic creates bad unit, unit dies, creditor punished, failure recorded, mutation triggered.

**Files:**
- Test: `internal/engine/engine_test.go`

- [ ] **Step 1: Write the end-to-end test**

```go
func TestSelfModificationLoop(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	// Type hierarchy
	anything := unit.New("Anything")
	anything.Set("isA", []string{})
	store.Put(anything)

	heuristic := unit.New("Heuristic")
	heuristic.Set("isA", []string{"Anything"})
	store.Put(heuristic)

	setType := unit.New("Set")
	setType.Set("isA", []string{"Anything"})
	store.Put(setType)

	// A heuristic that creates low-worth units (they'll be killed by H-KillWorthless)
	hBadCreator := unit.New("H-BadCreator")
	hBadCreator.SetWorth(500)
	hBadCreator.Set("isA", []string{"Heuristic", "Anything"})
	hBadCreator.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	hBadCreator.Set("ifPotentiallyRelevant", `
		"ArgU" @ "Set" isa?
		"ArgU" @ "Heuristic" isa? not
		and
	`)
	hBadCreator.Set("thenCompute", `
		"BadChild-" "ArgU" @ concat "childName" !
		"childName" @ unit-exists? not
		if
			"childName" @ "Set" create-unit drop
			50 "childName" @ "worth" set-slot
			"H-BadCreator" "childName" @ "creditors" set-slot
		then
	`)
	store.Put(hBadCreator)

	// Load H-KillWorthless (kills units with worth < 100)
	seed.LoadHeuristics(store)

	// Add a seed unit to trigger the heuristic
	target := unit.New("TestSet")
	target.SetWorth(500)
	target.Set("isA", []string{"Set", "Anything"})
	store.Put(target)

	// Run for enough cycles to see the loop
	eng.MaxCycles = 50
	eng.MutConfig.Enabled = true
	eng.MutConfig.Interval = 5
	eng.MutConfig.MinApplics = 3
	eng.MutConfig.MutationThreshold = 0.5

	err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify the loop happened:
	// 1. H-BadCreator should have applics failures (from unit deaths)
	record := hBadCreator.GetMap("overallRecord")
	if record == nil {
		t.Fatal("H-BadCreator overallRecord is nil")
	}
	failures := toInt(record["failures"])
	if failures == 0 {
		t.Error("expected H-BadCreator to have accumulated failures from unit deaths")
	}

	// 2. H-BadCreator's worth should have decreased (from punishCreators)
	if hBadCreator.Worth() >= 500 {
		t.Errorf("expected H-BadCreator worth to decrease below 500, got %d", hBadCreator.Worth())
	}

	// 3. Graveyard should have entries
	if len(eng.Graveyard) == 0 {
		t.Error("expected units in the graveyard")
	}

	// 4. HAvoid rules should exist
	avoidCount := 0
	for _, name := range store.All() {
		if store.IsA(name, "HAvoidRule") {
			avoidCount++
		}
	}
	if avoidCount == 0 {
		t.Error("expected HAvoid rules to be created via HindSight")
	}

	t.Logf("Loop results: H-BadCreator worth=%d, failures=%d, graveyard=%d, HAvoid rules=%d",
		hBadCreator.Worth(), failures, len(eng.Graveyard), avoidCount)
}
```

- [ ] **Step 2: Run the test**

Run: `cd /Users/chazu/dev/go/nous && go test ./internal/engine/ -run TestSelfModificationLoop -v`
Expected: PASS (all four loops working together)

- [ ] **Step 3: Run the full test suite**

Run: `cd /Users/chazu/dev/go/nous && go test ./... -v`
Expected: All pass

- [ ] **Step 4: Commit**

```bash
cd /Users/chazu/dev/go/nous
git add internal/engine/engine_test.go
git commit -m "test: end-to-end self-modification loop verification

Verifies the full cycle: H-BadCreator creates low-worth units,
H-KillWorthless kills them, HandleDeletedUnit records deferred
failures and creates HAvoid rules, creditor worth decreases.
Confirms all four feedback loops work together."
```

---

### Task 9: Clean Up Dead Code

Remove `pickHeuristicByWorth` (replaced by `pickWorstPerformer`) and `trackRarity` (orphaned, never called).

**Files:**
- Modify: `internal/engine/mutation.go:116-165` (remove pickHeuristicByWorth)
- Modify: `internal/engine/credit.go:76-96` (remove trackRarity)

- [ ] **Step 1: Verify pickHeuristicByWorth is unused**

Run: `cd /Users/chazu/dev/go/nous && grep -r "pickHeuristicByWorth" internal/`
Expected: Only the definition in mutation.go (no callers)

- [ ] **Step 2: Verify trackRarity is unused**

Run: `cd /Users/chazu/dev/go/nous && grep -r "trackRarity" internal/`
Expected: Only the definition in credit.go (no callers)

- [ ] **Step 3: Remove both functions**

Remove `pickHeuristicByWorth` from `internal/engine/mutation.go` (lines 116-165).
Remove `trackRarity` from `internal/engine/credit.go` (lines 76-96).

- [ ] **Step 4: Run all tests**

Run: `cd /Users/chazu/dev/go/nous && go test ./... -v`
Expected: All pass (no callers means no breakage)

- [ ] **Step 5: Commit**

```bash
cd /Users/chazu/dev/go/nous
git add internal/engine/mutation.go internal/engine/credit.go
git commit -m "chore: remove dead code (pickHeuristicByWorth, trackRarity)

pickHeuristicByWorth replaced by pickWorstPerformer in Task 3.
trackRarity was never called anywhere."
```
