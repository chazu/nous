package engine

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
)

func testEngine(t *testing.T) (*Engine, *bytes.Buffer) {
	t.Helper()
	store := unit.NewStore()
	ag := agenda.New()
	seed.LoadMath(store)
	seed.LoadHeuristics(store)

	eng := New(store, ag)
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf
	eng.Verbosity = 2
	return eng, buf
}

func TestEngineRuns(t *testing.T) {
	eng, _ := testEngine(t)
	eng.MaxCycles = 10

	err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if eng.Cycle() < 10 {
		// Should reach max cycles
		t.Errorf("expected at least 10 cycles, got %d", eng.Cycle())
	}
}

func TestEngineCreatesUnits(t *testing.T) {
	eng, _ := testEngine(t)
	eng.MaxCycles = 30
	eng.Verbosity = 0

	initialCount := eng.Store.Count()

	err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	finalCount := eng.Store.Count()
	if finalCount <= initialCount {
		t.Errorf("expected new units to be created: initial=%d final=%d", initialCount, finalCount)
	}
}

func TestEngineContextCancel(t *testing.T) {
	eng, _ := testEngine(t)
	eng.MaxCycles = 10000
	eng.Verbosity = 0

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := eng.Run(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestCreditPunishment(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// Create a heuristic that will be punished
	h := unit.New("H-Bad")
	h.SetWorth(600)
	h.Set("isA", []string{"Heuristic"})
	store.Put(h)

	// Create a unit with that heuristic as creditor
	u := unit.New("FailedUnit")
	u.SetWorth(50)
	u.Set("creditors", []string{"H-Bad"})
	u.Set("isA", []string{"Set"})
	store.Put(u)

	// Simulate kill-unit: snapshot the unit then delete it
	eng.VM.DeletedSnapshots = map[string]map[string]any{
		"FailedUnit": {
			"worth":     50,
			"creditors": []string{"H-Bad"},
			"isA":       []string{"Set"},
		},
	}
	store.Delete("FailedUnit")

	eng.HandleDeletedUnit("FailedUnit")

	if h.Worth() != 300 {
		t.Errorf("expected H-Bad worth halved to 300, got %d", h.Worth())
	}

	// HindSight should have created an avoidance rule
	if !store.Has("HAvoid-FailedUnit") {
		t.Error("expected HAvoid-FailedUnit to be created")
	}
	avoid := store.Get("HAvoid-FailedUnit")
	if avoid.GetString("avoidance_creditor") != "H-Bad" {
		t.Errorf("avoidance rule should track creditor H-Bad, got %s",
			avoid.GetString("avoidance_creditor"))
	}
}

func TestTrackApplics(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	h := unit.New("H-Test")
	h.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	store.Put(h)

	eng.trackApplics("H-Test", "SomeUnit", true)
	eng.trackApplics("H-Test", "OtherUnit", false)
	eng.trackApplics("H-Test", "ThirdUnit", true)

	record := h.GetMap("overallRecord")
	if record == nil {
		t.Fatal("overallRecord is nil")
	}
	if toInt(record["successes"]) != 2 {
		t.Errorf("expected 2 successes, got %d", toInt(record["successes"]))
	}
	if toInt(record["failures"]) != 1 {
		t.Errorf("expected 1 failure, got %d", toInt(record["failures"]))
	}
}

func TestHFindExamples(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()

	// Minimal setup
	s := unit.New("Shape")
	s.SetWorth(500)
	s.Set("isA", []string{"Anything"})
	store.Put(s)

	c := unit.New("Circle")
	c.SetWorth(400)
	c.Set("isA", []string{"Shape", "Anything"})
	store.Put(c)

	h := unit.New("H-FindExamples")
	h.SetWorth(700)
	h.Set("isA", []string{"Heuristic", "Anything"})
	h.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	h.Set("ifWorkingOnTask", `"CurSlot" @ "examples" =`)
	h.Set("thenCompute", `
		"CurUnit" @ examples
		"found" !
		"found" @ list-length 0 >
		if
			"found" @ "CurUnit" @ "examples" set-slot
		then
	`)
	store.Put(h)

	put := func(name string) {
		u := unit.New(name)
		u.Set("isA", []string{"Anything"})
		store.Put(u)
	}
	put("Anything")
	put("Heuristic")

	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	// Create a task for examples of Shape
	ag.Push(&agenda.Task{Priority: 500, UnitName: "Shape", SlotName: "examples", Reasons: []string{"test"}})

	eng.MaxCycles = 1
	eng.Run(context.Background())

	// Shape should now have examples
	examples := store.Get("Shape").Get("examples")
	if examples == nil {
		t.Fatal("Shape should have examples after H-FindExamples fires")
	}
}

func TestEngineOutput(t *testing.T) {
	eng, buf := testEngine(t)
	eng.MaxCycles = 5

	eng.Run(context.Background())

	output := buf.String()
	if !strings.Contains(output, "Cycle") {
		t.Error("expected cycle output")
	}
	if !strings.Contains(output, "fired") {
		t.Error("expected heuristic firing output")
	}
}

func TestDumpWorths(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	buf := &bytes.Buffer{}
	eng.Out = buf

	a := unit.New("Alpha")
	a.SetWorth(700)
	store.Put(a)

	b := unit.New("Beta")
	b.SetWorth(300)
	store.Put(b)

	eng.DumpWorths()

	output := buf.String()
	if !strings.Contains(output, "Alpha") || !strings.Contains(output, "Beta") {
		t.Error("DumpWorths should list all units")
	}
	// Alpha should appear before Beta (higher worth)
	alphaIdx := strings.Index(output, "Alpha")
	betaIdx := strings.Index(output, "Beta")
	if alphaIdx > betaIdx {
		t.Error("Alpha (700) should appear before Beta (300)")
	}
}

func TestTrackApplicsDeferredFailure(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// Create heuristic H-Creator with existing successes
	h := unit.New("H-Creator")
	h.SetWorth(600)
	h.Set("isA", []string{"Heuristic"})
	h.Set("overallRecord", map[string]any{"successes": 5, "failures": 0})
	store.Put(h)

	// Create unit BadUnit with H-Creator as creditor
	bad := unit.New("BadUnit")
	bad.SetWorth(50)
	bad.Set("creditors", []string{"H-Creator"})
	bad.Set("isA", []string{"Set"})
	store.Put(bad)

	// Simulate unit death: snapshot then delete
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
	if record == nil {
		t.Fatal("overallRecord is nil")
	}
	if toInt(record["successes"]) != 5 {
		t.Errorf("expected 5 successes preserved, got %d", toInt(record["successes"]))
	}
	if toInt(record["failures"]) != 1 {
		t.Errorf("expected 1 failure from deferred death, got %d", toInt(record["failures"]))
	}
}

func TestPerformanceBasedMutation(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	// Set mutation config
	eng.MutConfig.Enabled = true
	eng.MutConfig.MinApplics = 10
	eng.MutConfig.MutationThreshold = 0.3
	eng.MutConfig.MaxMutants = 20

	// Create Heuristic type unit
	hType := unit.New("Heuristic")
	hType.Set("isA", []string{"Anything"})
	store.Put(hType)

	anything := unit.New("Anything")
	anything.Set("isA", []string{"Anything"})
	store.Put(anything)

	// H-Bad: ratio 2/12 = 0.17, below threshold
	hBad := unit.New("H-Bad")
	hBad.SetWorth(500)
	hBad.Set("isA", []string{"Heuristic", "Anything"})
	hBad.Set("overallRecord", map[string]any{"successes": 2, "failures": 10})
	hBad.Set("thenCompute", "1 drop")
	store.Put(hBad)

	// H-Good: ratio 8/10 = 0.80, above threshold
	hGood := unit.New("H-Good")
	hGood.SetWorth(500)
	hGood.Set("isA", []string{"Heuristic", "Anything"})
	hGood.Set("overallRecord", map[string]any{"successes": 8, "failures": 2})
	hGood.Set("thenCompute", "1 drop")
	store.Put(hGood)

	eng.tryMutateHeuristic()

	// Check that a mutant of H-Bad was created
	foundBadMutant := false
	foundGoodMutant := false
	for _, name := range store.All() {
		u := store.Get(name)
		if u == nil {
			continue
		}
		mutOf := u.GetString("mutant_of")
		if mutOf == "H-Bad" {
			foundBadMutant = true
		}
		if mutOf == "H-Good" {
			foundGoodMutant = true
		}
	}

	if !foundBadMutant {
		t.Error("expected a mutant of H-Bad to be created")
	}
	if foundGoodMutant {
		t.Error("did not expect a mutant of H-Good")
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

	eng.MutConfig.Enabled = true
	eng.MutConfig.MinApplics = 10
	eng.MutConfig.MutationThreshold = 0.3
	eng.MutConfig.MaxMutants = 20

	// Create Heuristic type unit
	hType := unit.New("Heuristic")
	hType.Set("isA", []string{"Anything"})
	store.Put(hType)

	anything := unit.New("Anything")
	anything.Set("isA", []string{"Anything"})
	store.Put(anything)

	// H-Good: ratio 0.80, above threshold
	hGood := unit.New("H-Good")
	hGood.SetWorth(500)
	hGood.Set("isA", []string{"Heuristic", "Anything"})
	hGood.Set("overallRecord", map[string]any{"successes": 8, "failures": 2})
	hGood.Set("thenCompute", "1 drop")
	store.Put(hGood)

	countBefore := store.Count()

	eng.tryMutateHeuristic()

	if store.Count() != countBefore {
		t.Errorf("expected no new units, store count changed from %d to %d", countBefore, store.Count())
	}
}

func TestWorthGrowthReward(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// Create H-Creator at worth 500
	hCreator := unit.New("H-Creator")
	hCreator.SetWorth(500)
	hCreator.Set("isA", []string{"Heuristic"})
	store.Put(hCreator)

	// Create ChildUnit at worth 400, creditors: ["H-Creator"], creationWorth: 400, lastRewardedWorth: 400
	child := unit.New("ChildUnit")
	child.SetWorth(400)
	child.Set("isA", []string{"Set"})
	child.Set("creditors", []string{"H-Creator"})
	child.Set("creationWorth", 400)
	child.Set("lastRewardedWorth", 400)
	store.Put(child)

	// Bump ChildUnit to worth 600
	child.SetWorth(600)

	// Call rewardForWorthGrowth
	eng.rewardForWorthGrowth()

	// Assert: H-Creator worth is 600 (500 + delta/2 = 500 + 100)
	if hCreator.Worth() != 600 {
		t.Errorf("expected H-Creator worth 600, got %d", hCreator.Worth())
	}

	// Assert: lastRewardedWorth updated to 600
	if child.GetInt("lastRewardedWorth") != 600 {
		t.Errorf("expected lastRewardedWorth 600, got %d", child.GetInt("lastRewardedWorth"))
	}

	// Call again, assert no double-dipping
	eng.rewardForWorthGrowth()
	if hCreator.Worth() != 600 {
		t.Errorf("expected H-Creator worth still 600 after second call, got %d", hCreator.Worth())
	}
}

func TestWorthGrowthRewardSkipsHeuristics(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// Create H-Meta
	hMeta := unit.New("H-Meta")
	hMeta.SetWorth(500)
	hMeta.Set("isA", []string{"Heuristic"})
	store.Put(hMeta)

	// Create H-Child (a heuristic with creditors and creationWorth)
	hChild := unit.New("H-Child")
	hChild.SetWorth(800)
	hChild.Set("isA", []string{"Heuristic"})
	hChild.Set("creditors", []string{"H-Meta"})
	hChild.Set("creationWorth", 500)
	hChild.Set("lastRewardedWorth", 500)
	store.Put(hChild)

	// Call rewardForWorthGrowth
	eng.rewardForWorthGrowth()

	// Assert: H-Meta worth is NOT changed (heuristics are skipped)
	if hMeta.Worth() != 500 {
		t.Errorf("expected H-Meta worth unchanged at 500, got %d", hMeta.Worth())
	}
}

func TestTrackApplicsNoOpFailure(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()

	// Create a minimal "Anything" so isA lookups work
	anything := unit.New("Anything")
	anything.Set("isA", []string{"Anything"})
	store.Put(anything)

	heuristic := unit.New("Heuristic")
	heuristic.Set("isA", []string{"Anything"})
	store.Put(heuristic)

	// Create a heuristic whose thenCompute fires but produces nothing.
	// "1 drop" pushes 1 then drops it — net effect is zero.
	h := unit.New("H-NoOp")
	h.SetWorth(500)
	h.Set("isA", []string{"Heuristic", "Anything"})
	h.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	h.Set("thenCompute", "1 drop")
	store.Put(h)

	// Create a target unit
	target := unit.New("Target")
	target.SetWorth(500)
	target.Set("isA", []string{"Anything"})
	store.Put(target)

	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	// Fire the heuristic via a task
	ag.Push(&agenda.Task{
		Priority: 500,
		UnitName: "Target",
		SlotName: "examples",
		Reasons:  []string{"test"},
	})
	eng.MaxCycles = 1
	eng.Run(context.Background())

	// The heuristic fired (no if-guards) but produced no output.
	// overallRecord should have failures=1, successes=0.
	record := h.GetMap("overallRecord")
	if record == nil {
		t.Fatal("overallRecord is nil")
	}
	if toInt(record["successes"]) != 0 {
		t.Errorf("expected 0 successes, got %d", toInt(record["successes"]))
	}
	if toInt(record["failures"]) != 1 {
		t.Errorf("expected 1 failure, got %d", toInt(record["failures"]))
	}
}

func TestHAvoidValidation(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// Create a heuristic that will be the creditor
	h := unit.New("H-Creator")
	h.SetWorth(600)
	h.Set("isA", []string{"Heuristic"})
	store.Put(h)

	// Create a GraveRecord
	grave := GraveRecord{
		Name:      "DeadUnit",
		IsA:       []string{"Set"},
		Creditors: []string{"H-Creator"},
		Worth:     50,
		Cycle:     5,
	}

	eng.createAvoidanceRule(grave)

	if !store.Has("HAvoid-DeadUnit") {
		t.Fatal("expected HAvoid-DeadUnit to be created")
	}
	avoid := store.Get("HAvoid-DeadUnit")

	// Assert: worth is 300 (unproven), not 600
	if avoid.Worth() != 300 {
		t.Errorf("expected HAvoid worth 300, got %d", avoid.Worth())
	}

	// Assert: ifPotentiallyRelevant is non-empty and tokenizes
	ifProg := avoid.GetString("ifPotentiallyRelevant")
	if ifProg == "" {
		t.Fatal("expected non-empty ifPotentiallyRelevant")
	}
	tokens := dsl.Tokenize(ifProg)
	if len(tokens) == 0 {
		t.Error("expected ifPotentiallyRelevant to tokenize to non-empty tokens")
	}
}

func TestHAvoidPromotion(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// Create HAvoidRule type so Examples works
	hAvoidType := unit.New("HAvoidRule")
	hAvoidType.Set("isA", []string{"Anything"})
	store.Put(hAvoidType)

	anything := unit.New("Anything")
	anything.Set("isA", []string{"Anything"})
	store.Put(anything)

	// Create HAvoid unit at worth 300 with 3 successes
	avoid := unit.New("HAvoid-Test")
	avoid.SetWorth(300)
	avoid.Set("isA", []string{"HAvoidRule", "Heuristic", "Anything"})
	avoid.Set("overallRecord", map[string]any{"successes": 3, "failures": 0})
	avoid.Set("creationCycle", 10)
	store.Put(avoid)

	eng.cycle = 100
	eng.promoteOrDemoteHAvoidRules()

	if avoid.Worth() != 600 {
		t.Errorf("expected HAvoid worth promoted to 600, got %d", avoid.Worth())
	}
}

func TestHAvoidDemotion(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// Create HAvoidRule type so Examples works
	hAvoidType := unit.New("HAvoidRule")
	hAvoidType.Set("isA", []string{"Anything"})
	store.Put(hAvoidType)

	anything := unit.New("Anything")
	anything.Set("isA", []string{"Anything"})
	store.Put(anything)

	// Create HAvoid unit at worth 300 with zero firings, creationCycle 10
	avoid := unit.New("HAvoid-Idle")
	avoid.SetWorth(300)
	avoid.Set("isA", []string{"HAvoidRule", "Heuristic", "Anything"})
	avoid.Set("overallRecord", map[string]any{"successes": 0, "failures": 0})
	avoid.Set("creationCycle", 10)
	store.Put(avoid)

	eng.cycle = 250 // age = 240 > 200
	eng.promoteOrDemoteHAvoidRules()

	if avoid.Worth() != 100 {
		t.Errorf("expected HAvoid worth demoted to 100, got %d", avoid.Worth())
	}
}

func TestHAnalyzeApplics(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()

	// Type hierarchy
	anything := unit.New("Anything")
	anything.Set("isA", []string{"Anything"})
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

	// Create H-Skewed with overall ratio ~0.57 (in 0.3-0.7 range)
	hSkewed := unit.New("H-Skewed")
	hSkewed.SetWorth(500)
	hSkewed.Set("isA", []string{"Heuristic", "Anything"})
	hSkewed.Set("overallRecord", map[string]any{"successes": 8, "failures": 6})
	hSkewed.Set("thenCompute", "1 drop")
	hSkewed.Set("ifPotentiallyRelevant", "true")

	// Build applics: 8 successes on Set-type units, 6 failures on Number-type units
	applics := make([]map[string]any, 0, 14)
	for i := 0; i < 8; i++ {
		name := fmt.Sprintf("SetUnit-%d", i)
		u := unit.New(name)
		u.Set("isA", []string{"Set", "Anything"})
		u.SetWorth(400)
		store.Put(u)
		applics = append(applics, map[string]any{"target": name, "result": true})
	}
	for i := 0; i < 6; i++ {
		name := fmt.Sprintf("NumUnit-%d", i)
		u := unit.New(name)
		u.Set("isA", []string{"Number", "Anything"})
		u.SetWorth(400)
		store.Put(u)
		applics = append(applics, map[string]any{"target": name, "result": false})
	}
	hSkewed.Set("applics", applics)
	store.Put(hSkewed)

	// Load seed heuristics (includes H-AnalyzeApplics)
	seed.LoadHeuristics(store)

	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	// Fire heuristics on H-Skewed (Level 2 focus)
	eng.WorkOnUnit("H-Skewed")

	// Assert: a unit with creditors including "H-AnalyzeApplics" was created
	found := false
	for _, name := range store.All() {
		u := store.Get(name)
		if u == nil {
			continue
		}
		creds := u.GetStrings("creditors")
		for _, c := range creds {
			if c == "H-AnalyzeApplics" {
				found = true
				// Verify it's specialized from H-Skewed
				if u.GetString("specialized_from") != "H-Skewed" {
					t.Errorf("expected specialized_from=H-Skewed, got %s", u.GetString("specialized_from"))
				}
				if u.GetString("specialized_type") != "Set" {
					t.Errorf("expected specialized_type=Set, got %s", u.GetString("specialized_type"))
				}
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		t.Error("expected a unit with creditors including H-AnalyzeApplics to be created")
	}
}
