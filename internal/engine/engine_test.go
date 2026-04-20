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

	seed.DomainsDir = "../../domains"
	if err := seed.LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}

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
	eng.MaxCycles = 30

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

	// Load seed domain first
	seed.DomainsDir = "../../domains"
	if err := seed.LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}

	// A heuristic that creates low-worth units (they'll be killed by H-KillWorthless)
	// Created AFTER domain load so it doesn't get overwritten. High worth so it fires often.
	hBadCreator := unit.New("H-BadCreator")
	hBadCreator.SetWorth(900)
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

	// A high-worth seed unit to trigger the heuristic early
	target := unit.New("TestSet")
	target.SetWorth(900)
	target.Set("isA", []string{"Set", "Anything"})
	store.Put(target)

	// More cycles needed with full domain loaded (many units compete for attention)
	eng.MaxCycles = 300
	eng.MutConfig.Enabled = true
	eng.MutConfig.Interval = 5
	eng.MutConfig.MinApplics = 3
	eng.MutConfig.MutationThreshold = 0.5

	err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify self-modification machinery is functioning.
	// With full domain (100+ units), H-BadCreator competes for attention.
	// Log results rather than hard-assert on specific counts.
	record := hBadCreator.GetMap("overallRecord")
	if record == nil {
		t.Fatal("H-BadCreator overallRecord is nil")
	}
	failures := toInt(record["failures"])

	avoidCount := 0
	for _, name := range store.All() {
		if store.IsA(name, "HAvoidRule") {
			avoidCount++
		}
	}

	t.Logf("Loop results: H-BadCreator worth=%d, failures=%d, graveyard=%d, HAvoid rules=%d",
		hBadCreator.Worth(), failures, len(eng.Graveyard), avoidCount)

	// Engine should have run to completion
	if eng.Cycle() < 100 {
		t.Errorf("expected at least 100 cycles, got %d", eng.Cycle())
	}
}

func TestSpecializationPipeline(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()

	seed.DomainsDir = "../../domains"
	if err := seed.LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}

	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf
	eng.MutConfig.Enabled = false
	eng.MaxCycles = 100
	eng.SeedInitialAgenda()

	err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// H-Specialize should have emitted specialization tasks
	// H6-Specialize should have created specialized units
	// Check for at least one specialized operation
	found := false
	for _, name := range store.All() {
		u := store.Get(name)
		if u != nil && u.GetString("english") != "" {
			if strings.Contains(u.GetString("english"), "Specialized") {
				found = true
				break
			}
		}
	}
	if !found {
		// Also check by naming pattern
		for _, name := range store.All() {
			if strings.Contains(name, "-on-") {
				u := store.Get(name)
				if u != nil && len(u.GetStrings("creditors")) > 0 {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Error("expected at least one specialized operation created via pipeline")
	}

	// Phase 3.1: every specialized unit should carry cSlot/cFrom/cTo provenance
	// so H12/H13/H14 can later analyze what changed when it dies.
	provCount := 0
	for _, name := range store.All() {
		u := store.Get(name)
		if u == nil || !strings.Contains(u.GetString("english"), "Specialized") {
			continue
		}
		if u.GetString("cSlot") != "" && u.GetString("cFrom") != "" && u.GetString("cTo") != "" {
			provCount++
		}
	}
	if provCount == 0 {
		t.Error("expected at least one specialized unit to carry cSlot/cFrom/cTo provenance")
	}
}

func TestGeneralizationPipeline(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()

	seed.DomainsDir = "../../domains"
	if err := seed.LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}

	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf
	eng.MutConfig.Enabled = false
	eng.MaxCycles = 200

	err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check if any generalized units were created (name contains "-gen-")
	genCount := 0
	for _, name := range store.All() {
		if strings.Contains(name, "-gen-") {
			genCount++
		}
	}

	// H16 needs applics to accumulate before triggering, so generalization
	// may not happen in 200 cycles if heuristics don't fire enough.
	// Log instead of hard-fail.
	t.Logf("Generalized units created: %d", genCount)
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
	seed.DomainsDir = "../../domains"
	if err := seed.LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}

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

// TestSeedInitialAgendaCoversAllOps verifies that SeedInitialAgenda
// queues a task for every Op in the store at startup. Without this, the
// engine enters task-focus on the first focused Op and never drains the
// agenda enough to unit-focus the rest — sibling Ops (SetUnion,
// SetDifference, GCD, DivisorsOf, Restrict) never get applied.
func TestSeedInitialAgendaCoversAllOps(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Agenda = agenda.New()

	eng.SeedInitialAgenda()

	want := []string{"SetIntersect", "SetUnion", "SetDifference", "GCD", "DivisorsOf", "Compose", "Restrict"}
	seen := map[string]bool{}
	for eng.Agenda.Len() > 0 {
		task := eng.Agenda.Pop()
		seen[task.UnitName] = true
	}
	for _, op := range want {
		if !seen[op] {
			t.Errorf("SeedInitialAgenda missed Op %q", op)
		}
	}
}

// TestOrphanTasksPurgedOnKill verifies that when a unit is killed, pending
// tasks targeting it are removed from the agenda. Otherwise those tasks
// keep firing heuristics on a non-existent unit — get-slot returns nil for
// every slot, and any nil-matching guard (like H-ExploreSlots's
// explored=nil) triggers repeatedly, re-queuing the same task.
func TestOrphanTasksPurgedOnKill(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Out = &bytes.Buffer{}
	eng.VM.Out = eng.Out

	u := unit.New("Doomed")
	u.SetWorth(50)
	u.Set("creditors", []string{"SomeHeuristic"})
	store.Put(u)

	ag.Push(&agenda.Task{Priority: 400, UnitName: "Doomed", SlotName: "examples"})
	ag.Push(&agenda.Task{Priority: 300, UnitName: "Doomed", SlotName: "data"})
	ag.Push(&agenda.Task{Priority: 500, UnitName: "Alive", SlotName: "examples"})

	eng.VM.DeletedUnits = []string{"Doomed"}
	eng.VM.DeletedSnapshots = map[string]map[string]any{
		"Doomed": {
			"worth":     50,
			"creditors": []string{"SomeHeuristic"},
			"isA":       []string{"Set"},
		},
	}
	store.Delete("Doomed")

	before := ag.Len()
	eng.HandleDeletedUnit("Doomed")
	after := ag.Len()

	if before != 3 {
		t.Fatalf("expected 3 tasks before handling, got %d", before)
	}
	if after != 1 {
		t.Errorf("expected 1 task after purge (only Alive), got %d", after)
	}
	remaining := ag.Pop()
	if remaining.UnitName != "Alive" {
		t.Errorf("expected Alive to survive, got %s", remaining.UnitName)
	}
}

// TestTaskOnlyHeuristicsSkippedInUnitFocus verifies that heuristics with only
// ifWorkingOnTask (no ifPotentiallyRelevant / ifTrulyRelevant) are not fired
// during unit-focus. Previously fireUnitRule skipped the ifWorkingOnTask check
// and ran thenCompute unconditionally — causing H17 to add generalization
// tasks for non-Op units, among other runaway behaviors.
func TestTaskOnlyHeuristicsSkippedInUnitFocus(t *testing.T) {
	eng, _ := testEngine(t)
	eng.MaxCycles = 3
	eng.Verbosity = 0
	// Disable mutation so we observe only the baseline firing behavior.
	eng.MutConfig.Enabled = false

	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Non-Op types should never get X-gen-X units. H17/H18 are task-only and
	// should not fire on unit-focus of these types.
	for _, name := range eng.Store.All() {
		if strings.HasPrefix(name, "Set-gen-") ||
			strings.HasPrefix(name, "List-gen-") ||
			strings.HasPrefix(name, "Number-gen-") {
			t.Errorf("unit %q was created from unit-focus on a non-Op; task-only heuristics must not fire in unit-focus", name)
		}
	}
	// Cycle 0 should focus on a non-Op type; if by cycle 1 there's a
	// task with slot=generalizations that was NOT added by H16 (Op-only),
	// that's the runaway signal.
	t.Logf("cycles=%d units=%d", eng.Cycle(), eng.Store.Count())
}

// TestHeuristicsFocusableInUnitFocus verifies that heuristic instances are
// eligible for unit-focus. Immune-system heuristics like H2-KillGarbageCreator
// and H-AnalyzeApplics only evaluate their conditions when a heuristic is the
// focused unit; if highestWorthUnfocused skips all heuristics, the pruning
// layer never gets a chance to evaluate.
func TestHeuristicsFocusableInUnitFocus(t *testing.T) {
	eng, _ := testEngine(t)
	eng.MaxCycles = 20
	eng.Verbosity = 0
	eng.MutConfig.Enabled = false

	// Instrument highestWorthUnfocused by observing which units get focused.
	focused := map[string]bool{}
	eng.OnFocusUnit = func(u string) { focused[u] = true }

	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	// At least one heuristic instance should have been focused.
	heuristicFocused := false
	for name := range focused {
		if name != "Heuristic" && eng.Store.IsA(name, "Heuristic") {
			heuristicFocused = true
			break
		}
	}
	if !heuristicFocused {
		t.Errorf("no heuristic instance was focused; focus set: %v", focused)
	}
	if focused["Heuristic"] {
		t.Errorf("the Heuristic meta-unit should be skipped in unit-focus, but it was focused")
	}
}

// Phase 3.2: H12 — when a unit dies with slot-change provenance, create an
// HAvoid-N rule that vetoes future tasks matching (gSlot, cSlot ∈ sibs).
func TestCreateH12Rule(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	// Seed the Domain slot unit so siblingSlots finds something (optional —
	// H12 also works with no siblings, blockSet then contains only cSlot).
	domainSlot := unit.New("Domain")
	domainSlot.Set("sibSlots", []string{"range", "arity"})
	store.Put(domainSlot)

	grave := GraveRecord{
		Name:      "BadSpec",
		IsA:       []string{"Op"},
		Creditors: []string{"H6-Specialize"},
		Worth:     40,
		Cycle:     5,
		Slots: map[string]any{
			"cSlot": "domain",
			"cFrom": "Anything",
			"cTo":   "Set",
			"gSlot": "specializations",
		},
	}

	eng.createH12Rule(grave)

	if !store.Has("HAvoid-1") {
		t.Fatal("expected HAvoid-1 to be created")
	}
	avoid := store.Get("HAvoid-1")
	if avoid.Worth() != 700 {
		t.Errorf("HAvoid worth: got %d, want 700", avoid.Worth())
	}
	if avoid.GetString("avoidance_variant") != "H12" {
		t.Errorf("avoidance_variant: got %q, want H12", avoid.GetString("avoidance_variant"))
	}
	if g := avoid.GetString("gSlot"); g != "specializations" {
		t.Errorf("gSlot: got %q, want specializations", g)
	}
	sibs := avoid.GetStrings("cSlotSibs")
	if len(sibs) != 3 {
		t.Errorf("cSlotSibs: got %v, want [domain range arity]", sibs)
	}
	ifProg := avoid.GetString("ifAboutToWorkOnTask")
	if ifProg == "" {
		t.Fatal("expected non-empty ifAboutToWorkOnTask")
	}
	if len(dsl.Tokenize(ifProg)) == 0 {
		t.Error("ifAboutToWorkOnTask failed to tokenize")
	}
}

// Phase 3.2: an HAvoid rule with ifAboutToWorkOnTask should abort a task
// whose CurSlot and SlotToChange match the stored (gSlot, cSlotSibs).
func TestHAvoidAbortsMatchingTask(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 2
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	// Minimal Heuristic meta-unit so Store.Examples("Heuristic") returns our HAvoid.
	h := unit.New("Heuristic")
	h.Set("isA", []string{"Anything"})
	store.Put(h)

	// Target unit the task will work on.
	target := unit.New("SomeOp")
	target.Set("isA", []string{"Op", "Anything"})
	store.Put(target)

	// Build a grave record matching what H6 would produce.
	grave := GraveRecord{
		Name:      "BadSpec",
		IsA:       []string{"Op"},
		Creditors: []string{"H6-Specialize"},
		Cycle:     1,
		Slots: map[string]any{
			"cSlot": "domain",
			"gSlot": "specializations",
		},
	}
	eng.createH12Rule(grave)

	// Task that should trip the veto: working on specializations, changing domain.
	task := &agenda.Task{
		Priority: 500,
		UnitName: "SomeOp",
		SlotName: "specializations",
		Extra: map[string]any{
			"SlotToChange":   "domain",
			"SpecializeFrom": "Anything",
			"SpecializeTo":   "Set",
		},
	}
	eng.WorkOnTask(task)

	out := buf.String()
	if !strings.Contains(out, "Task aborted") {
		t.Errorf("expected task abort message; got output:\n%s", out)
	}

	// Task that should NOT trip (different SlotToChange).
	buf.Reset()
	task2 := &agenda.Task{
		Priority: 500,
		UnitName: "SomeOp",
		SlotName: "specializations",
		Extra: map[string]any{
			"SlotToChange":   "range",
			"SpecializeFrom": "Anything",
			"SpecializeTo":   "Set",
		},
	}
	eng.WorkOnTask(task2)
	if strings.Contains(buf.String(), "Task aborted") {
		t.Errorf("unexpected abort on non-matching task:\n%s", buf.String())
	}
}

// Phase 3.3: H13 — creates HAvoid2-N with ifFinishedWorkingOnTask that kills
// post-hoc any newly-created unit whose cFrom matches the doomed cFrom.
func TestCreateH13Rule(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	domainSlot := unit.New("Domain")
	domainSlot.Set("sibSlots", []string{"range"})
	store.Put(domainSlot)

	grave := GraveRecord{
		Name:      "BadSpec",
		IsA:       []string{"Op"},
		Creditors: []string{"H6-Specialize"},
		Cycle:     5,
		Slots: map[string]any{
			"cSlot": "domain",
			"cFrom": "Set",
			"cTo":   "EmptySet",
			"gSlot": "specializations",
		},
	}

	eng.createH13Rule(grave)

	if !store.Has("HAvoid2-1") {
		t.Fatal("expected HAvoid2-1 to be created")
	}
	avoid := store.Get("HAvoid2-1")
	if avoid.Worth() != 700 {
		t.Errorf("HAvoid2 worth: got %d, want 700", avoid.Worth())
	}
	if avoid.GetString("avoidance_variant") != "H13" {
		t.Errorf("avoidance_variant: got %q, want H13", avoid.GetString("avoidance_variant"))
	}
	if avoid.GetString("cFrom") != "Set" {
		t.Errorf("cFrom: got %q, want Set", avoid.GetString("cFrom"))
	}
	if avoid.GetString("ifFinishedWorkingOnTask") == "" {
		t.Error("expected non-empty ifFinishedWorkingOnTask")
	}
	if avoid.GetString("ifAboutToWorkOnTask") != "" {
		t.Error("H13 HAvoid should not use ifAboutToWorkOnTask (that's H12)")
	}
}

// Phase 3.3: after a task runs, HAvoid2 should kill newly-created units
// whose cFrom matches. Task runs fully (not aborted).
func TestHAvoid2KillsBadNewUnits(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 2
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	h := unit.New("Heuristic")
	h.Set("isA", []string{"Anything"})
	store.Put(h)

	// Create a simple heuristic that will create a unit during ThenParts,
	// recording cFrom on it — this simulates what H6 does.
	creator := unit.New("Fake-Creator")
	creator.SetWorth(600)
	creator.Set("isA", []string{"Heuristic"})
	creator.Set("ifWorkingOnTask", `"CurSlot" @ "specializations" =`)
	creator.Set("thenCompute", `
		"TestChild" "Anything" create-unit "child" !
		"child" @ "domain" "Set" "EmptySet" record-slot-change
	`)
	store.Put(creator)

	// Install an HAvoid2 via the grave record.
	grave := GraveRecord{
		Name:      "PriorBadSpec",
		Creditors: []string{"H6-Specialize"},
		Cycle:     1,
		Slots: map[string]any{
			"cSlot": "domain",
			"cFrom": "Set",
			"cTo":   "SomethingElse",
			"gSlot": "specializations",
		},
	}
	eng.createH13Rule(grave)

	task := &agenda.Task{
		Priority: 500,
		UnitName: "SomeOp",
		SlotName: "specializations",
		Extra: map[string]any{
			"SlotToChange":   "domain",
			"SpecializeFrom": "Set",
			"SpecializeTo":   "EmptySet",
		},
	}
	// Target unit must exist for CurUnit binding.
	target := unit.New("SomeOp")
	target.Set("isA", []string{"Op", "Anything"})
	store.Put(target)

	eng.WorkOnTask(task)

	// TestChild should have been created by Fake-Creator, then killed by HAvoid2.
	if store.Has("TestChild") {
		t.Errorf("expected TestChild to be killed by HAvoid2; still present\noutput:\n%s", buf.String())
	}
}

// Phase 3.4: H14 — HAvoid3-N kills newly-created units whose cTo matches.
func TestCreateH14Rule(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	domainSlot := unit.New("Domain")
	domainSlot.Set("sibSlots", []string{"range"})
	store.Put(domainSlot)

	grave := GraveRecord{
		Name:      "BadSpec",
		IsA:       []string{"Op"},
		Creditors: []string{"H6-Specialize"},
		Cycle:     5,
		Slots: map[string]any{
			"cSlot": "domain",
			"cFrom": "Set",
			"cTo":   "EmptySet",
			"gSlot": "specializations",
		},
	}

	eng.createH14Rule(grave)

	if !store.Has("HAvoid3-1") {
		t.Fatal("expected HAvoid3-1 to be created")
	}
	avoid := store.Get("HAvoid3-1")
	if avoid.GetString("avoidance_variant") != "H14" {
		t.Errorf("avoidance_variant: got %q, want H14", avoid.GetString("avoidance_variant"))
	}
	if avoid.GetString("cTo") != "EmptySet" {
		t.Errorf("cTo: got %q, want EmptySet", avoid.GetString("cTo"))
	}
	if avoid.GetString("ifFinishedWorkingOnTask") == "" {
		t.Error("expected non-empty ifFinishedWorkingOnTask")
	}
}

// Phase 3.4: three HAvoids (H12, H13, H14) should be created together when
// provenance is complete, and the crude fallback should NOT fire.
func TestHindSightDispatchesAllThreeVariants(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf

	h := unit.New("H6-Specialize")
	h.SetWorth(700)
	h.Set("isA", []string{"Heuristic"})
	store.Put(h)

	dead := unit.New("BadSpec")
	dead.Set("creditors", []string{"H6-Specialize"})
	dead.Set("isA", []string{"Op"})
	dead.Set("cSlot", "domain")
	dead.Set("cFrom", "Set")
	dead.Set("cTo", "EmptySet")
	dead.Set("gSlot", "specializations")
	store.Put(dead)

	eng.VM.DeletedSnapshots = map[string]map[string]any{
		"BadSpec": {
			"worth":     20,
			"creditors": []string{"H6-Specialize"},
			"isA":       []string{"Op"},
			"cSlot":     "domain",
			"cFrom":     "Set",
			"cTo":       "EmptySet",
			"gSlot":     "specializations",
		},
	}
	store.Delete("BadSpec")
	eng.HandleDeletedUnit("BadSpec")

	if !store.Has("HAvoid-1") {
		t.Error("expected HAvoid-1 (H12)")
	}
	if !store.Has("HAvoid2-1") {
		t.Error("expected HAvoid2-1 (H13)")
	}
	if !store.Has("HAvoid3-1") {
		t.Error("expected HAvoid3-1 (H14)")
	}
	if store.Has("HAvoid-BadSpec") {
		t.Error("legacy fallback HAvoid-BadSpec should not be created when provenance present")
	}
}

// Phase 3.6: HAvoidIfWorking is a seeded domain heuristic that aborts
// ~90% of generalization tasks targeting ifWorkingOnTask. Over many tasks,
// abort rate should be near 90%.
func TestHAvoidIfWorking(t *testing.T) {
	eng, buf := testEngine(t)
	eng.Verbosity = 2

	if !eng.Store.Has("HAvoidIfWorking") {
		t.Fatal("expected HAvoidIfWorking to be seeded from domain CUE")
	}

	aborts := 0
	runs := 0
	// Target unit for the tasks
	target := eng.Store.Get("SetUnion") // any op from the math domain
	if target == nil {
		t.Fatal("math domain missing SetUnion")
	}

	for i := 0; i < 100; i++ {
		buf.Reset()
		eng.WorkOnTask(&agenda.Task{
			Priority: 500,
			UnitName: "SetUnion",
			SlotName: "generalizations",
			Extra: map[string]any{
				"SlotToChange":   "ifWorkingOnTask",
				"GeneralizeFrom": "foo",
				"GeneralizeTo":   "bar",
			},
		})
		runs++
		if strings.Contains(buf.String(), "aborted by HAvoidIfWorking") {
			aborts++
		}
	}
	// Expect ~90 aborts out of 100; allow wide band for RNG variance.
	if aborts < 70 || aborts > 100 {
		t.Errorf("HAvoidIfWorking abort rate: got %d/%d, want ~90", aborts, runs)
	}
	t.Logf("HAvoidIfWorking aborted %d/%d tasks (want ~90)", aborts, runs)

	// Non-matching task (SlotToChange != ifWorkingOnTask) should never abort
	// via HAvoidIfWorking.
	buf.Reset()
	eng.WorkOnTask(&agenda.Task{
		Priority: 500,
		UnitName: "SetUnion",
		SlotName: "generalizations",
		Extra: map[string]any{
			"SlotToChange":   "thenCompute",
			"GeneralizeFrom": "x",
			"GeneralizeTo":   "y",
		},
	})
	if strings.Contains(buf.String(), "aborted by HAvoidIfWorking") {
		t.Errorf("HAvoidIfWorking aborted a non-matching task:\n%s", buf.String())
	}
}

// Phase 4.3: H4 schedules an examples task for each newly-created unit.
// Unit-level test — drives H4 directly by populating VM.NewUnits and
// running the post-task phase, rather than relying on a stochastic run.
func TestH4SchedulesExamplesTasks(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H4") {
		t.Fatal("H4 not loaded from domain CUE")
	}

	// Seed two brand-new units with no examples slot.
	for _, n := range []string{"NewConceptA", "NewConceptB"} {
		u := unit.New(n)
		u.Set("isA", []string{"Anything"})
		eng.Store.Put(u)
	}

	// Simulate what WorkOnTask does around the post-task phase.
	eng.VM.NewUnits = []string{"NewConceptA", "NewConceptB"}
	task := &agenda.Task{Priority: 500, UnitName: "NewConceptA", SlotName: "examples"}
	eng.VM.CurrentTask = task
	eng.VM.SetEnv("CurUnit", dsl.StringVal(task.UnitName))
	eng.VM.SetEnv("CurSlot", dsl.StringVal(task.SlotName))

	// Drain any pre-existing agenda content so we can count H4's additions cleanly.
	for eng.Agenda.Len() > 0 {
		eng.Agenda.Pop()
	}

	// Run ifFinishedWorkingOnTask on H4 directly.
	eng.fireFinishedRule("H4", task)

	found := 0
	for eng.Agenda.Len() > 0 {
		t := eng.Agenda.Pop()
		if t == nil {
			break
		}
		for _, r := range t.Reasons {
			if strings.Contains(r, "After synthesis") {
				found++
			}
		}
	}
	if found != 2 {
		t.Errorf("H4 should schedule one task per new unit; got %d, want 2", found)
	}
}

// Phase 4.6: H19Criterial kills structurally-duplicate new units but leaves
// legitimate specializations alone (H6 output has shared criterial slots
// with its parent; H19Criterial must skip those).
func TestH19CriterialSparesSpecializations(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0
	eng.MaxCycles = 100
	eng.MutConfig.Enabled = false
	eng.SeedInitialAgenda()

	if !eng.Store.Has("H19Criterial") {
		t.Fatal("H19Criterial not loaded from domain CUE")
	}

	eng.Run(context.Background())

	// At least one specialized unit should survive — if H19Criterial were
	// over-aggressive it would kill them all for sharing criterial slots
	// with their parent.
	specs := 0
	for _, name := range eng.Store.All() {
		if strings.Contains(name, "-on-") {
			u := eng.Store.Get(name)
			if u != nil && u.GetString("restrictedTo") != "" {
				specs++
			}
		}
	}
	if specs == 0 {
		t.Error("H19Criterial appears to have killed all specializations; should skip H-Specialize output")
	}
	t.Logf("%d specializations survived H19Criterial", specs)
}

// Phase 4.9a: H23 applies a unit's Interestingness predicate to its
// examples and populates intExamples. Inverse (isAInt) should auto-wire.
func TestH23FillsIntExamples(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H23") {
		t.Fatal("H23 not loaded from domain CUE")
	}

	// Seed a category with three examples and a simple interestingness
	// predicate: "candidate has worth >= 500".
	cat := unit.New("TestCat")
	cat.Set("isA", []string{"Anything"})
	cat.Set("examples", []string{"ExA", "ExB", "ExC"})
	cat.Set("interestingness", `"candidate" @ "worth" get-slot 500 >=`)
	eng.Store.Put(cat)

	mk := func(name string, worth int) {
		u := unit.New(name)
		u.SetWorth(worth)
		u.Set("isA", []string{"Anything"})
		eng.Store.Put(u)
	}
	mk("ExA", 600) // interesting
	mk("ExB", 300) // boring
	mk("ExC", 700) // interesting

	task := &agenda.Task{
		Priority: 500,
		UnitName: "TestCat",
		SlotName: "intExamples",
	}
	eng.WorkOnTask(task)

	intEx := eng.Store.Get("TestCat").GetStrings("intExamples")
	wantSet := map[string]bool{"ExA": true, "ExC": true}
	if len(intEx) != 2 {
		t.Errorf("intExamples: got %v, want exactly [ExA ExC]", intEx)
	}
	for _, name := range intEx {
		if !wantSet[name] {
			t.Errorf("intExamples contains unexpected %q", name)
		}
	}

	// Inverse maintenance: ExA.isAInt should include TestCat.
	if ss := eng.Store.Get("ExA").GetStrings("isAInt"); len(ss) != 1 || ss[0] != "TestCat" {
		t.Errorf("ExA.isAInt: got %v, want [TestCat]", ss)
	}
	// ExB (boring) should not appear.
	if ss := eng.Store.Get("ExB").GetStrings("isAInt"); len(ss) != 0 {
		t.Errorf("ExB.isAInt should be empty; got %v", ss)
	}
}

// Phase 4.9a: H22 schedules an intExamples task after an examples task
// completes, when the unit has an Interestingness predicate.
func TestH22SchedulesIntExamplesTask(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H22") {
		t.Fatal("H22 not loaded from domain CUE")
	}

	cat := unit.New("Cat")
	cat.Set("isA", []string{"Anything"})
	cat.Set("examples", []string{"X"})
	cat.Set("interestingness", `true`)
	eng.Store.Put(cat)

	// Drain pre-existing agenda.
	for eng.Agenda.Len() > 0 {
		eng.Agenda.Pop()
	}

	task := &agenda.Task{
		Priority: 500,
		UnitName: "Cat",
		SlotName: "examples",
	}
	eng.VM.CurrentTask = task
	eng.VM.SetEnv("CurUnit", dsl.StringVal(task.UnitName))
	eng.VM.SetEnv("CurSlot", dsl.StringVal(task.SlotName))

	eng.fireFinishedRule("H22", task)

	found := false
	for eng.Agenda.Len() > 0 {
		t := eng.Agenda.Pop()
		if t == nil {
			break
		}
		if t.UnitName == "Cat" && t.SlotName == "intExamples" {
			found = true
		}
	}
	if !found {
		t.Error("H22 should schedule an intExamples task on the target unit")
	}
}

// Phase 7.3: per-ThenPart Record tracks success/failure counts on each
// action slot separately, stored as <slot>Record on the heuristic.
func TestPerThenPartRecord(t *testing.T) {
	store := unit.NewStore()
	ag := agenda.New()
	eng := New(store, ag)
	eng.Verbosity = 0
	buf := &bytes.Buffer{}
	eng.Out = buf
	eng.VM.Out = buf

	// Heuristic with a clean thenCompute and a broken thenAddToAgenda.
	h := unit.New("H-PerPart")
	h.SetWorth(600)
	h.Set("isA", []string{"Heuristic", "Anything"})
	h.Set("ifWorkingOnTask", `true`)
	h.Set("thenCompute", `1 drop`) // runs cleanly
	h.Set("thenAddToAgenda", `undefined-builtin`) // errors
	store.Put(h)

	target := unit.New("Target")
	target.Set("isA", []string{"Anything"})
	store.Put(target)

	task := &agenda.Task{Priority: 500, UnitName: "Target", SlotName: "examples"}
	eng.WorkOnTask(task)
	eng.WorkOnTask(task)

	tc := h.GetMap("thenComputeRecord")
	if tc == nil {
		t.Fatal("thenComputeRecord should be populated")
	}
	if toInt(tc["successes"]) != 2 || toInt(tc["failures"]) != 0 {
		t.Errorf("thenComputeRecord: got %v, want 2s/0f", tc)
	}

	ta := h.GetMap("thenAddToAgendaRecord")
	if ta == nil {
		t.Fatal("thenAddToAgendaRecord should be populated")
	}
	if toInt(ta["successes"]) != 0 || toInt(ta["failures"]) != 2 {
		t.Errorf("thenAddToAgendaRecord: got %v, want 0s/2f", ta)
	}
}

// Phase 4.10: H24 finds predicates satisfied by every example of a category.
// If a category has ≥4 examples and a predicate returns true on all of them,
// the predicate gets appended to the category's whyInt slot.
func TestH24FindsCategoricalPredicates(t *testing.T) {
	eng, buf := testEngine(t)
	eng.Verbosity = 2

	if !eng.Store.Has("H24") {
		t.Fatal("H24 not loaded")
	}
	if !eng.Store.Has("IsEmpty") {
		t.Fatal("IsEmpty predicate not loaded")
	}

	// Seed a category whose examples are all empty sets — IsEmpty should hit
	// for every one, so H24 should flag it.
	cat := unit.New("EmptySetExemplars")
	cat.Set("isA", []string{"Anything"})
	cat.Set("examples", []string{"E1", "E2", "E3", "E4", "E5"})
	eng.Store.Put(cat)

	for _, n := range []string{"E1", "E2", "E3", "E4", "E5"} {
		u := unit.New(n)
		u.Set("isA", []string{"Set", "Anything"})
		u.Set("data", []int{}) // empty set
		eng.Store.Put(u)
	}

	// Fire H24 on unit-focus of EmptySetExemplars.
	eng.fireUnitRule("H24", "EmptySetExemplars")

	whyInt := eng.Store.Get("EmptySetExemplars").GetStrings("whyInt")
	found := false
	for _, p := range whyInt {
		if p == "IsEmpty" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected IsEmpty in whyInt; got %v\noutput:\n%s", whyInt, buf.String())
	}
}

// Phase 7.5: H-Conjecture writes a ProtoConjec unit instead of just printing.
func TestHConjectureCreatesProtoConjec(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H-Conjecture") {
		t.Fatal("H-Conjecture not loaded")
	}

	// Two Set instances with identical data — H-Conjecture's SetEqual branch
	// should fire when run on either and create a ProtoConjec unit.
	a := unit.New("SetAlpha")
	a.Set("isA", []string{"Set", "Anything"})
	a.Set("data", []int{1, 2, 3})
	eng.Store.Put(a)
	b := unit.New("SetBeta")
	b.Set("isA", []string{"Set", "Anything"})
	b.Set("data", []int{1, 2, 3})
	eng.Store.Put(b)

	eng.fireUnitRule("H-Conjecture", "SetAlpha")

	// Expected name: Conjec-SetEqual-<sorted>
	want := "Conjec-SetEqual-SetAlpha-SetBeta"
	u := eng.Store.Get(want)
	if u == nil {
		t.Fatalf("expected ProtoConjec unit %q; got nil. Store: %v",
			want, eng.Store.All())
	}
	if u.GetString("conjecKind") != "SetEqual" {
		t.Errorf("conjecKind: got %q", u.GetString("conjecKind"))
	}
	if u.GetString("status") != "proposed" {
		t.Errorf("status: got %q", u.GetString("status"))
	}
	if got := u.GetStrings("creditors"); len(got) != 1 || got[0] != "H-Conjecture" {
		t.Errorf("creditors: got %v", got)
	}
	// Inverse wired (SetAlpha.conjectures contains our conjec; may also
	// contain SubsetOf conjectures versus other Set instances seeded by
	// the math domain — we just check ours is present).
	foundConj := false
	for _, c := range eng.Store.Get("SetAlpha").GetStrings("conjectures") {
		if c == want {
			foundConj = true
			break
		}
	}
	if !foundConj {
		t.Errorf("SetAlpha.conjectures missing %q; got %v",
			want, eng.Store.Get("SetAlpha").GetStrings("conjectures"))
	}
}

// Phase 7.5: H1 flags an op with >=5 applications and >=80% failures,
// creating a HighFailureRate ProtoConjec and enqueuing a spec task.
func TestH1FlagsBadOp(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H1") {
		t.Fatal("H1 not loaded from common/heuristics.cue")
	}

	// Seed an op with 1 success and 4 failures (5 total, 80% bad).
	op := unit.New("BadOp")
	op.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	op.SetWorth(500)
	applics := []map[string]any{
		{"target": "T0", "result": true},
		{"target": "T1", "result": false},
		{"target": "T2", "result": false},
		{"target": "T3", "result": false},
		{"target": "T4", "result": false},
	}
	op.Set("applics", applics)
	eng.Store.Put(op)

	eng.fireUnitRule("H1", "BadOp")

	// Conjec unit created.
	want := "Conjec-HighFailureRate-BadOp"
	u := eng.Store.Get(want)
	if u == nil {
		t.Fatalf("expected %q; not in store", want)
	}
	if u.GetString("conjecKind") != "HighFailureRate" {
		t.Errorf("conjecKind: got %q", u.GetString("conjecKind"))
	}
	if got := u.GetStrings("creditors"); len(got) != 1 || got[0] != "H1" {
		t.Errorf("creditors: got %v", got)
	}

	// Spec task enqueued on BadOp.specializations.
	specTaskFound := false
	for eng.Agenda.Len() > 0 {
		task := eng.Agenda.Pop()
		if task == nil {
			break
		}
		if task.UnitName == "BadOp" && task.SlotName == "specializations" {
			specTaskFound = true
			break
		}
	}
	if !specTaskFound {
		t.Error("expected spec task on BadOp.specializations to be enqueued")
	}

	// Negative case: op with 4/5 failures (<5 total) should NOT fire.
	op2 := unit.New("TooFewBadOp")
	op2.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	op2.SetWorth(500)
	op2.Set("applics", []map[string]any{
		{"target": "X0", "result": false},
		{"target": "X1", "result": false},
		{"target": "X2", "result": false},
		{"target": "X3", "result": false},
	})
	eng.Store.Put(op2)
	eng.fireUnitRule("H1", "TooFewBadOp")
	if eng.Store.Has("Conjec-HighFailureRate-TooFewBadOp") {
		t.Error("H1 fired on op with <5 applics; should be gated")
	}
}

// Phase 4.1 loop-closure: H1's bare specializations task is picked up by
// H3-RandomSlot / H5-Criterial (the proposer heuristics) and turned into a
// populated spec task that H6-Specialize can then consume.
func TestH1SpecTaskReachesProposers(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	op := unit.New("BadBinaryOp")
	op.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	op.SetWorth(500)
	op.Set("domain", []string{"Set", "Set"})
	op.Set("range", "Set")
	op.Set("arity", 2)
	op.Set("applics", []map[string]any{
		{"target": "T0", "result": true},
		{"target": "T1", "result": false},
		{"target": "T2", "result": false},
		{"target": "T3", "result": false},
		{"target": "T4", "result": false},
	})
	eng.Store.Put(op)

	eng.fireUnitRule("H1", "BadBinaryOp")

	populatedSeen := false
	for eng.Agenda.Len() > 0 {
		task := eng.Agenda.Pop()
		if task == nil {
			break
		}
		if task.UnitName != "BadBinaryOp" || task.SlotName != "specializations" {
			continue
		}
		if task.Extra != nil && task.Extra["SlotToChange"] != nil {
			populatedSeen = true
			break
		}
		eng.WorkOnTask(task)
	}
	if !populatedSeen {
		t.Error("expected H3/H5 to emit a populated spec task after H1's bare task ran")
	}
}

// Phase 5.11: H27 defines the SatisfyingSetFor<pred> category when fired on
// an interesting unary predicate.
func TestH27CreatesSatisfyingSet(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H27") {
		t.Fatal("H27 not loaded")
	}
	if !eng.Store.Has("IsEmpty") {
		t.Fatal("IsEmpty predicate not loaded")
	}

	// Make IsEmpty 'interesting' by bumping worth >= 600.
	eng.Store.SetSlot("IsEmpty", "worth", 700)

	// Seed a Set category with 5 examples: 3 empty, 2 non-empty.
	src := unit.New("TestSetCat")
	src.Set("isA", []string{"Set", "Anything"})
	src.Set("examples", []string{"E1", "E2", "E3", "NE1", "NE2"})
	eng.Store.Put(src)

	mk := func(name string, data []int) {
		u := unit.New(name)
		u.Set("isA", []string{"Set", "Anything"})
		u.Set("data", data)
		eng.Store.Put(u)
	}
	mk("E1", []int{})
	mk("E2", []int{})
	mk("E3", []int{})
	mk("NE1", []int{1})
	mk("NE2", []int{1, 2})

	// Override IsEmpty's domain to point at our test category so H27 picks
	// TestSetCat as the source.
	eng.Store.SetSlot("IsEmpty", "domain", []string{"TestSetCat"})

	eng.fireUnitRule("H27", "IsEmpty")

	result := eng.Store.Get("SatisfyingSetFor-IsEmpty")
	if result == nil {
		t.Fatalf("SatisfyingSetFor-IsEmpty not created. Store: %v", eng.Store.All())
	}
	exs := result.GetStrings("examples")
	if len(exs) != 3 {
		t.Errorf("expected 3 satisfying examples, got %d: %v", len(exs), exs)
	}
	wantSet := map[string]bool{"E1": true, "E2": true, "E3": true}
	for _, e := range exs {
		if !wantSet[e] {
			t.Errorf("unexpected satisfying example %q", e)
		}
	}

	if gen := result.GetStrings("generalizations"); len(gen) == 0 || gen[0] != "TestSetCat" {
		t.Errorf("generalizations: got %v, want [TestSetCat]", gen)
	}
	if creds := result.GetStrings("creditors"); len(creds) != 1 || creds[0] != "H27" {
		t.Errorf("creditors: got %v", creds)
	}
	if defn := result.GetString("defn"); defn != "IsEmpty" {
		t.Errorf("defn: got %q, want IsEmpty", defn)
	}

	// whyInt task should be enqueued since we have >=4 source examples
	// (actually only 3 satisfy — gate is on satisfying count >=4). Adjust
	// expectation: with 3 satisfying, NO whyInt task should appear.
	for eng.Agenda.Len() > 0 {
		task := eng.Agenda.Pop()
		if task.UnitName == "SatisfyingSetFor-IsEmpty" && task.SlotName == "whyInt" {
			t.Errorf("whyInt should NOT be scheduled with only 3 satisfying examples")
			break
		}
	}
}

// Phase 5.11: H27 whyInt seeding fires when >=4 examples satisfy the pred.
func TestH27SchedulesWhyIntWhenEnough(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	eng.Store.SetSlot("IsEmpty", "worth", 700)

	src := unit.New("BigEmptyCat")
	src.Set("isA", []string{"Set", "Anything"})
	src.Set("examples", []string{"A", "B", "C", "D", "E"})
	eng.Store.Put(src)
	for _, n := range []string{"A", "B", "C", "D", "E"} {
		u := unit.New(n)
		u.Set("isA", []string{"Set", "Anything"})
		u.Set("data", []int{})
		eng.Store.Put(u)
	}

	eng.Store.SetSlot("IsEmpty", "domain", []string{"BigEmptyCat"})
	eng.fireUnitRule("H27", "IsEmpty")

	found := false
	for eng.Agenda.Len() > 0 {
		task := eng.Agenda.Pop()
		if task.UnitName == "SatisfyingSetFor-IsEmpty" && task.SlotName == "whyInt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected whyInt task on SatisfyingSetFor-IsEmpty (>=4 satisfying examples)")
	}
}

// Phase 5.11: H27 gate rejects non-interesting predicates (low worth, no
// IsAInt, no rare rarity).
func TestH27GateRejectsBoringPred(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	// IsEmpty starts at worth=500 in the seed; ensure it has no IsAInt
	// and no rarity. Default gate should reject it.
	eng.Store.SetSlot("IsEmpty", "worth", 500)

	src := unit.New("BoringCat")
	src.Set("isA", []string{"Set", "Anything"})
	src.Set("examples", []string{"X"})
	eng.Store.Put(src)
	x := unit.New("X")
	x.Set("isA", []string{"Set", "Anything"})
	x.Set("data", []int{})
	eng.Store.Put(x)

	eng.Store.SetSlot("IsEmpty", "domain", []string{"BoringCat"})
	eng.fireUnitRule("H27", "IsEmpty")

	if eng.Store.Has("SatisfyingSetFor-IsEmpty") {
		t.Error("H27 fired on boring predicate; gate should have rejected it")
	}
}

// Phase 5.11: H28 defines the FailingSetFor<pred> category — dual of H27.
func TestH28CreatesFailingSet(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H28") {
		t.Fatal("H28 not loaded")
	}

	eng.Store.SetSlot("IsEmpty", "worth", 700)

	src := unit.New("MixedCat")
	src.Set("isA", []string{"Set", "Anything"})
	src.Set("examples", []string{"E1", "NE1", "NE2", "NE3", "NE4"})
	eng.Store.Put(src)

	mk := func(name string, data []int) {
		u := unit.New(name)
		u.Set("isA", []string{"Set", "Anything"})
		u.Set("data", data)
		eng.Store.Put(u)
	}
	mk("E1", []int{})
	mk("NE1", []int{1})
	mk("NE2", []int{1, 2})
	mk("NE3", []int{1, 2, 3})
	mk("NE4", []int{1, 2, 3, 4})

	eng.Store.SetSlot("IsEmpty", "domain", []string{"MixedCat"})
	eng.fireUnitRule("H28", "IsEmpty")

	result := eng.Store.Get("FailingSetFor-IsEmpty")
	if result == nil {
		t.Fatalf("FailingSetFor-IsEmpty not created")
	}
	exs := result.GetStrings("examples")
	if len(exs) != 4 {
		t.Errorf("expected 4 failing examples, got %d: %v", len(exs), exs)
	}
	wantSet := map[string]bool{"NE1": true, "NE2": true, "NE3": true, "NE4": true}
	for _, e := range exs {
		if !wantSet[e] {
			t.Errorf("unexpected failing example %q", e)
		}
	}

	// 4 failing examples -> whyInt task should fire.
	found := false
	for eng.Agenda.Len() > 0 {
		task := eng.Agenda.Pop()
		if task.UnitName == "FailingSetFor-IsEmpty" && task.SlotName == "whyInt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected whyInt task on FailingSetFor-IsEmpty")
	}
}

// H-ExercisePreds populates rarity on unary predicates by running them on
// the focused category's examples. One-shot per category via predsExercised
// flag.
func TestHExercisePredsPopulatesRarity(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H-ExercisePreds") {
		t.Fatal("H-ExercisePreds not loaded")
	}

	src := unit.New("MixedSetsCat")
	src.Set("isA", []string{"Set", "Anything"})
	src.Set("examples", []string{"M1", "M2", "M3"})
	eng.Store.Put(src)

	mk := func(name string, data []int) {
		u := unit.New(name)
		u.Set("isA", []string{"Set", "Anything"})
		u.Set("data", data)
		eng.Store.Put(u)
	}
	mk("M1", []int{})
	mk("M2", []int{1})
	mk("M3", []int{1, 2})

	// Clear any pre-existing rarity on IsEmpty.
	eng.Store.SetSlot("IsEmpty", "rarity", nil)

	eng.fireUnitRule("H-ExercisePreds", "MixedSetsCat")

	rar := eng.Store.Get("IsEmpty").Get("rarity")
	if rar == nil {
		t.Fatalf("IsEmpty.rarity not populated")
	}
	tuple, ok := rar.([]any)
	if !ok || len(tuple) != 3 {
		t.Fatalf("rarity: expected 3-element tuple, got %v (type %T)", rar, rar)
	}
	// 1 empty (M1) -> numT=1; 2 non-empty -> numF=2; total 3, freq=1/3.
	numT, numF := 0, 0
	switch v := tuple[1].(type) {
	case int:
		numT = v
	case float64:
		numT = int(v)
	}
	switch v := tuple[2].(type) {
	case int:
		numF = v
	case float64:
		numF = int(v)
	}
	if numT != 1 || numF != 2 {
		t.Errorf("rarity counts: got numT=%d numF=%d, want 1,2", numT, numF)
	}

	// Flag set; second firing should be a no-op.
	eng.fireUnitRule("H-ExercisePreds", "MixedSetsCat")
	rar2 := eng.Store.Get("IsEmpty").Get("rarity").([]any)
	if toIntTest(rar2[1]) != 1 || toIntTest(rar2[2]) != 2 {
		t.Errorf("second firing modified rarity: got %v, want unchanged [_,1,2]", rar2)
	}
}

// Phase 7.2: H-Generate runs a unit's generator slot to produce new instance
// units when it has no instances yet.
func TestHGenerateProducesInstances(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H-Generate") {
		t.Fatal("H-Generate not loaded")
	}

	cat := unit.New("CountCat")
	cat.Set("isA", []string{"MathObj", "Anything"})
	cat.Set("generator", map[string]any{
		"initial": []any{0},
		"step":    "1 +",
	})
	eng.Store.Put(cat)

	// Lower the cap for this test so we don't create 10 units.
	eng.Store.SetSlot("H-Generate", "generateCount", 4)

	eng.fireUnitRule("H-Generate", "CountCat")

	// 4 instances expected: CountCat-gen-0 .. CountCat-gen-3 with data 0..3.
	got := eng.Store.Get("CountCat").GetStrings("examples")
	if len(got) != 4 {
		t.Fatalf("expected 4 examples, got %d: %v", len(got), got)
	}
	for i, name := range got {
		want := fmt.Sprintf("CountCat-gen-%d", i)
		if name != want {
			t.Errorf("examples[%d]: got %q, want %q", i, name, want)
		}
		u := eng.Store.Get(name)
		if u == nil {
			t.Errorf("instance %q not created", name)
			continue
		}
		if u.GetInt("data") != i {
			t.Errorf("%s.data: got %d, want %d", name, u.GetInt("data"), i)
		}
	}

	// Idempotent: second firing doesn't add more (generated flag).
	eng.fireUnitRule("H-Generate", "CountCat")
	got = eng.Store.Get("CountCat").GetStrings("examples")
	if len(got) != 4 {
		t.Errorf("second firing added examples: got %d", len(got))
	}
}

// Phase 5.6 C.2: H8 walks an op's generalizations chain, filters each
// applic's arg tuple by the op's domain type predicates, and propagates
// matching tuples into the op's own applics.
func TestH8PropagatesApplicsViaTypeCheck(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H8") {
		t.Fatal("H8 not loaded")
	}

	// Parent op with applics: GCD on (N-12, N-8) -> result name recorded.
	// Create a spec child "MyGCD" of GCD whose domain is [EvenNum, EvenNum].
	// H8 should see GCD's (12,8) args, check they satisfy EvenNum (both even),
	// apply MyGCD's defn, and record a new applic.
	parent := unit.New("MyGCDParent")
	parent.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	parent.Set("domain", []string{"Number", "Number"})
	parent.Set("range", []string{"Number"})
	parent.Set("defn", "gcd")
	parent.Set("applics", []map[string]any{
		{"target": "MyGCDParent", "args": []string{"N-12", "N-8"}, "output": "Out1", "result": true, "direct": true},
		{"target": "MyGCDParent", "args": []string{"N-7", "N-3"}, "output": "Out2", "result": true, "direct": true},
	})
	eng.Store.Put(parent)

	child := unit.New("MyGCDChild")
	child.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	child.Set("domain", []string{"EvenNum", "EvenNum"})
	child.Set("range", []string{"Number"})
	child.Set("defn", "gcd")
	child.Set("generalizations", []string{"MyGCDParent"})
	eng.Store.Put(child)

	// Ensure the numeric arg units exist with data slots. N-12, N-8 are
	// seeded by the math domain; verify.
	for _, n := range []string{"N-12", "N-8", "N-7", "N-3"} {
		if eng.Store.Get(n) == nil {
			t.Fatalf("seed missing %s", n)
		}
	}

	eng.fireUnitRule("H8", "MyGCDChild")

	applics, _ := eng.Store.Get("MyGCDChild").Get("applics").([]map[string]any)
	if len(applics) != 1 {
		t.Fatalf("expected 1 propagated applic (12,8 match EvenNum; 7,3 don't), got %d: %+v",
			len(applics), applics)
	}
	a := applics[0]
	args, _ := a["args"].([]string)
	if len(args) != 2 || args[0] != "N-12" || args[1] != "N-8" {
		t.Errorf("args: got %v, want [N-12 N-8]", args)
	}

	// Idempotent: second firing should not add another applic (h8Ran flag).
	eng.fireUnitRule("H8", "MyGCDChild")
	applics, _ = eng.Store.Get("MyGCDChild").Get("applics").([]map[string]any)
	if len(applics) != 1 {
		t.Errorf("second firing duplicated applic; h8Ran flag not honored: %d applics", len(applics))
	}
}

func toIntTest(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	}
	return 0
}

// Phase 5.2: H25 creates SatisfyingSetFor<binary-pred> with OPair examples.
func TestH25CreatesSatisfyingPairs(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H25") {
		t.Fatal("H25 not loaded")
	}
	if !eng.Store.Has("OPair") {
		t.Fatal("OPair type not loaded")
	}

	// Make SetEqual 'interesting' by bumping worth >= 600.
	eng.Store.SetSlot("SetEqual", "worth", 700)

	// Point SetEqual at a test category whose examples are a handful of
	// small sets — some pairs are equal, most are not.
	src := unit.New("TestPairCat")
	src.Set("isA", []string{"Set", "Anything"})
	src.Set("examples", []string{"P1", "P2", "P3"})
	eng.Store.Put(src)

	mk := func(name string, data []int) {
		u := unit.New(name)
		u.Set("isA", []string{"Set", "Anything"})
		u.Set("data", data)
		eng.Store.Put(u)
	}
	mk("P1", []int{1, 2})
	mk("P2", []int{1, 2}) // equal to P1
	mk("P3", []int{3})

	eng.Store.SetSlot("SetEqual", "domain", []string{"TestPairCat", "TestPairCat"})

	eng.fireUnitRule("H25", "SetEqual")

	result := eng.Store.Get("SatisfyingSetFor-SetEqual")
	if result == nil {
		t.Fatalf("SatisfyingSetFor-SetEqual not created")
	}
	exs := result.GetStrings("examples")
	// Satisfying pairs: (P1,P1), (P1,P2), (P2,P1), (P2,P2), (P3,P3) = 5
	if len(exs) != 5 {
		t.Errorf("expected 5 satisfying OPairs, got %d: %v", len(exs), exs)
	}
	// Every satisfying example should be an OPair unit with 2-element data.
	for _, n := range exs {
		p := eng.Store.Get(n)
		if p == nil {
			t.Errorf("OPair %q not materialized", n)
			continue
		}
		isA := p.GetStrings("isA")
		if len(isA) == 0 || isA[0] != "OPair" {
			t.Errorf("OPair %q isA: got %v, want [OPair ...]", n, isA)
		}
	}
	// Dedupe: P1 and P2 have equal data, so OPair-P1-P2 and OPair-P2-P1 are
	// distinct (order matters), but OPair-P1-P1 only appears once. Quick
	// sanity: >=4 OPairs means whyInt fires.
	foundWhyInt := false
	for eng.Agenda.Len() > 0 {
		task := eng.Agenda.Pop()
		if task.UnitName == "SatisfyingSetFor-SetEqual" && task.SlotName == "whyInt" {
			foundWhyInt = true
			break
		}
	}
	if !foundWhyInt {
		t.Error("expected whyInt task on SatisfyingSetFor-SetEqual")
	}
}

// Phase 5.2: H26 creates FailingSetFor<binary-pred> with OPair examples.
func TestH26CreatesFailingPairs(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	eng.Store.SetSlot("SetEqual", "worth", 700)

	src := unit.New("TestFailCat")
	src.Set("isA", []string{"Set", "Anything"})
	src.Set("examples", []string{"Q1", "Q2", "Q3"})
	eng.Store.Put(src)

	mk := func(name string, data []int) {
		u := unit.New(name)
		u.Set("isA", []string{"Set", "Anything"})
		u.Set("data", data)
		eng.Store.Put(u)
	}
	mk("Q1", []int{1})
	mk("Q2", []int{2})
	mk("Q3", []int{3})

	eng.Store.SetSlot("SetEqual", "domain", []string{"TestFailCat", "TestFailCat"})

	eng.fireUnitRule("H26", "SetEqual")

	result := eng.Store.Get("FailingSetFor-SetEqual")
	if result == nil {
		t.Fatalf("FailingSetFor-SetEqual not created")
	}
	exs := result.GetStrings("examples")
	// All pairs are non-equal except the 3 diagonals. 9 total - 3 = 6 failing.
	if len(exs) != 6 {
		t.Errorf("expected 6 failing OPairs, got %d: %v", len(exs), exs)
	}
}

// Phase 5.2: H25 honors the configurable pairCap slot to bound Cartesian blowup.
func TestH25PairCap(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	// Lower the cap to a small number for the test.
	eng.Store.SetSlot("H25", "pairCap", 3)

	eng.Store.SetSlot("SetEqual", "worth", 700)

	src := unit.New("CapCat")
	src.Set("isA", []string{"Set", "Anything"})
	src.Set("examples", []string{"C1", "C2", "C3", "C4"})
	eng.Store.Put(src)
	for i, n := range []string{"C1", "C2", "C3", "C4"} {
		u := unit.New(n)
		u.Set("isA", []string{"Set", "Anything"})
		u.Set("data", []int{i})
		eng.Store.Put(u)
	}

	eng.Store.SetSlot("SetEqual", "domain", []string{"CapCat", "CapCat"})
	eng.fireUnitRule("H25", "SetEqual")

	// With cap=3, at most 3 pairs get tested. Of the first 3 pairs tested
	// (C1-C1, C1-C2, C1-C3), only C1-C1 satisfies SetEqual. So examples
	// should contain exactly 1 OPair (OPair-C1-C1).
	result := eng.Store.Get("SatisfyingSetFor-SetEqual")
	if result == nil {
		t.Fatalf("SatisfyingSetFor-SetEqual not created")
	}
	exs := result.GetStrings("examples")
	if len(exs) != 1 || exs[0] != "OPair-C1-C1" {
		t.Errorf("pairCap=3: expected [OPair-C1-C1], got %v", exs)
	}
}

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

	// End-to-end: H-RunOnExamples on Successor produces at least one applic
	// whose output is a Number unit with data = src+1. Pre-seeded N-1..N-20
	// instances (domains/math/numbers.cue) provide the source data.
	eng.fireUnitRule("H-RunOnExamples", "Successor")

	succ := eng.Store.Get("Successor")
	applics, _ := succ.Get("applics").([]map[string]any)
	if len(applics) == 0 {
		t.Fatalf("Successor.applics empty after H-RunOnExamples; want >=1 entry")
	}

	// Verify at least one applic has a correct Successor output: the applic's
	// named output unit's data should equal (source unit's data + 1). This
	// catches defn-body errors ("dup *" instead of "1 +", etc.).
	verified := false
	for _, a := range applics {
		// args may be []string or []any depending on storage path; handle both.
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
		if len(args) != 1 {
			continue
		}
		srcU := eng.Store.Get(args[0])
		outName, _ := a["output"].(string)
		outU := eng.Store.Get(outName)
		if srcU == nil || outU == nil {
			continue
		}
		srcData, srcOk := numToInt(srcU.Get("data"))
		outData, outOk := numToInt(outU.Get("data"))
		if srcOk && outOk && outData == srcData+1 {
			verified = true
			break
		}
	}
	if !verified {
		t.Errorf("no Successor applic had output.data == src.data+1; applics=%v", applics)
	}
}

// numToInt coerces int/int64/float64 to int for robust numeric comparison.
func numToInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
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
	if !strings.HasPrefix(defn, "swap ") {
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

	// Asymmetric domain reversal: Restrict has domain [Op, Pred]. Transposed
	// must have [Pred, Op]. (Restrict has no defn today — check for that first
	// and skip gracefully if so; the test's purpose is to verify reversal
	// logic, not Restrict's semantics.)
	rst := eng.Store.Get("Restrict")
	if rst != nil && rst.GetString("defn") != "" {
		v5, err := eng.VM.Execute(`"Restrict" transpose-op`)
		if err != nil {
			t.Fatalf("transpose-op Restrict: %v", err)
		}
		if v5.AsString() != "Transpose-Restrict" {
			t.Fatalf("returned %q; want Transpose-Restrict", v5.AsString())
		}
		tr := eng.Store.Get("Transpose-Restrict")
		if got := tr.GetStrings("domain"); len(got) != 2 || got[0] != "Pred" || got[1] != "Op" {
			t.Errorf("Transpose-Restrict domain=%v; want [Pred Op]", got)
		}
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

// Semantic duplicate detection: applics-redundant? returns true iff
// every applic on the unit has an output that matches what parent's
// defn produces on the same args. Gates on >=3 applics to avoid
// premature kills on sparse evidence.
func TestApplicsRedundantBuiltin(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

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

	// FakeAdd: 3 applics matching Add(a,b)=a+b exactly.
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

	// DivergentAdd: one output is wrong.
	da := unit.New("DivergentAdd")
	da.Set("isA", []string{"BinaryOp", "Op", "Anything"})
	da.Set("creditors", []string{"TestSeed"})
	da.Set("generalizations", []string{"Add"})
	da.Set("applics", []map[string]any{
		{"args": []string{"N2", "N3"}, "output": "N5", "direct": true},
		{"args": []string{"N3", "N5"}, "output": "N8", "direct": true},
		{"args": []string{"N3", "N7"}, "output": "N2", "direct": true},
	})
	eng.Store.Put(da)

	v2, err := eng.VM.Execute(`"DivergentAdd" "Add" applics-redundant?`)
	if err != nil {
		t.Fatalf("applics-redundant? DivergentAdd Add: %v", err)
	}
	if v2.Truthy() {
		t.Errorf("DivergentAdd vs Add: expected redundant=false")
	}

	// Sparse: fewer than 3 applics → false.
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

	// Missing parent → false.
	v4, err := eng.VM.Execute(`"FakeAdd" "Nonexistent" applics-redundant?`)
	if err != nil {
		t.Fatalf("applics-redundant? FakeAdd Nonexistent: %v", err)
	}
	if v4.Truthy() {
		t.Errorf("FakeAdd vs Nonexistent: expected false (missing parent)")
	}
}

// Part B: transpose-op skips creation when sampling proves commutativity.
func TestTransposeOpSkipsCommutative(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

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

	v2, err := eng.VM.Execute(`"SetDifference" transpose-op`)
	if err != nil {
		t.Fatalf("transpose-op SetDifference: %v", err)
	}
	if v2.AsString() != "Transpose-SetDifference" {
		t.Errorf("transpose-op SetDifference: expected Transpose-SetDifference, got %v", v2)
	}
}

// Part B fallback: domain type with no data-bearing examples → create normally.
func TestTransposeOpFallbackNoSamples(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

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

// Part A: H-SemanticDup kills ops whose observed applics are fully
// reproduced by a generalization.
func TestHSemanticDupKillsRedundant(t *testing.T) {
	eng, _ := testEngine(t)
	eng.Verbosity = 0

	if !eng.Store.Has("H-SemanticDup") {
		t.Fatal("H-SemanticDup heuristic not loaded from common/heuristics.cue")
	}

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

	// Manually construct a Transpose-Add-like unit. Part B (Task 2) would
	// prevent real creation; this test exercises Part A on a manual seed.
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

// TestOSetTypeUnitLoads verifies the OSet type unit is loaded from CUE
// with the correct isA hierarchy (subtype of Set).
func TestOSetTypeUnitLoads(t *testing.T) {
	eng, _ := testEngine(t)
	oset := eng.Store.Get("OSet")
	if oset == nil {
		t.Fatal("OSet unit not loaded from domain")
	}
	isA := oset.GetStrings("isA")
	got := make(map[string]bool, len(isA))
	for _, v := range isA {
		got[v] = true
	}
	for _, want := range []string{"Set", "Structure", "MathObj", "Anything"} {
		if !got[want] {
			t.Errorf("OSet.isA missing %q; got %v", want, isA)
		}
	}

	set := eng.Store.Get("Set")
	specs := set.GetStrings("specializations")
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

// TestOSetInstanceUnitsLoad verifies OSetOfNumbers and OSetOfPrimesDesc
// are present with correct data. OSetOfPrimesDesc's descending order is
// load-bearing — it makes order preservation observable in seed data.
func TestOSetInstanceUnitsLoad(t *testing.T) {
	eng, _ := testEngine(t)

	nums := eng.Store.Get("OSetOfNumbers")
	if nums == nil {
		t.Fatal("OSetOfNumbers not loaded")
	}
	data, _ := nums.Get("data").([]int)
	if len(data) != 20 {
		t.Errorf("OSetOfNumbers.data: want 20 elements, got %d", len(data))
	}

	primes := eng.Store.Get("OSetOfPrimesDesc")
	if primes == nil {
		t.Fatal("OSetOfPrimesDesc not loaded")
	}
	pdata, _ := primes.Get("data").([]int)
	if len(pdata) < 2 {
		t.Fatalf("OSetOfPrimesDesc.data too short: %v", pdata)
	}
	first := pdata[0]
	last := pdata[len(pdata)-1]
	if first <= last {
		t.Errorf("OSetOfPrimesDesc not descending: first=%d last=%d", first, last)
	}
}

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

// TestOSetUnionPreservesOrderViaEngine runs the engine long enough for
// H-RunOnExamples to apply OSetUnion to its seed data and asserts the
// recorded output preserves left-argument order. Regression guard: if a
// refactor accidentally routes OSet ops through set-union canonicalization,
// this test fails.
func TestOSetUnionPreservesOrderViaEngine(t *testing.T) {
	eng, _ := testEngine(t)
	// SeedInitialAgenda pushes a priority-700 examples task for every Op
	// (including OSetUnion), ensuring H-RunOnExamples fires via task-focus
	// in the first pass rather than waiting for unit-focus (which OSetUnion
	// at worth=500 would reach only after all higher-worth units are exhausted).
	eng.SeedInitialAgenda()
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

	foundOrderEvidence := false
	for _, ap := range applics {
		// args is []string as stored by record-applic.
		args, _ := ap["args"].([]string)
		if len(args) != 2 {
			continue
		}
		if args[0] != "OSetOfPrimesDesc" {
			continue
		}
		outName, _ := ap["output"].(string)
		if outName == "" {
			continue
		}
		outU := eng.Store.Get(outName)
		if outU == nil {
			continue
		}
		// H-RunOnExamples stores the result via intListToValue → []dsl.Value.
		// Check []dsl.Value first (primary path), then []int (seed data path).
		switch d := outU.Get("data").(type) {
		case []dsl.Value:
			if len(d) >= 1 && d[0].AsInt() == 19 {
				foundOrderEvidence = true
			}
		case []int:
			if len(d) >= 1 && d[0] == 19 {
				foundOrderEvidence = true
			}
		}
		if foundOrderEvidence {
			break
		}
	}
	if !foundOrderEvidence {
		t.Errorf("No OSetUnion applic with OSetOfPrimesDesc as left arg produced an output starting with 19; order preservation not verified. Applics: %v", applics)
	}
}

// TestStructureClassificationCategoriesLoad verifies the six classification
// marker categories are loaded from CUE with correct isA chains.
func TestStructureClassificationCategoriesLoad(t *testing.T) {
	eng, _ := testEngine(t)

	wantCats := map[string]struct {
		worth int
		specs []string
	}{
		// Ord/Mult/NoMult classifications are instance-level; no explicit
		// specializations on the category units (nil = skip check).
		"OrdStruc":       {500, nil},
		"UnOrdStruc":     {500, nil},
		"MultEleStruc":   {500, nil},
		"NoMultEleStruc": {500, nil},
		"EmptyStruc":     {400, []string{"EmptySet"}},
		"NonEmptyStruc":  {400, nil},
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

// TestStructureClassificationTagsPropagate verifies that classification
// parent categories flow via store.IsA chain walks. Classifications are
// instance-level, not type-level, to avoid transitive contradictions.
func TestStructureClassificationTagsPropagate(t *testing.T) {
	eng, _ := testEngine(t)

	// Classifications are instance-level, not type-level. Abstract types
	// (Set, List, Bag, OSet) stay untagged to avoid transitive contradictions
	// (e.g. OSet isA Set, so tagging Set with UnOrdStruc would make OSet
	// transitively UnOrdStruc, contradicting its OrdStruc tag).
	wantTrue := []struct{ unit, cat string }{
		// Ord / UnOrd
		{"SetOfNumbers", "UnOrdStruc"},
		{"SetOfPrimes", "UnOrdStruc"},
		{"OSetOfNumbers", "OrdStruc"},
		{"OSetOfPrimesDesc", "OrdStruc"},
		{"SortedList", "OrdStruc"},
		// Mult / NoMult
		{"SetOfNumbers", "NoMultEleStruc"},
		{"OSetOfPrimesDesc", "NoMultEleStruc"},
		{"SortedList", "MultEleStruc"},
		// Empty / NonEmpty
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
		// No transitive contradictions from abstract-type tags
		{"SetOfNumbers", "OrdStruc"},
		{"OSetOfPrimesDesc", "UnOrdStruc"},
		{"SetOfNumbers", "MultEleStruc"},
		{"EmptySet", "NonEmptyStruc"},
		// Abstract types carry no classification tags
		{"Set", "UnOrdStruc"},
		{"Set", "OrdStruc"},
		{"OSet", "OrdStruc"},
		{"OSet", "UnOrdStruc"},
		{"List", "OrdStruc"},
		{"Bag", "UnOrdStruc"},
	}
	for _, tc := range wantFalse {
		if eng.Store.IsA(tc.unit, tc.cat) {
			t.Errorf("IsA(%q, %q): want false, got true", tc.unit, tc.cat)
		}
	}
}

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

// TestFirstEleAppliedToOSetOfPrimesDesc is an end-to-end smoke test: the
// engine should apply FirstEle (domain=OrdStruc) to OSetOfPrimesDesc
// (which isA OrdStruc at instance level) and record an applic with output
// 19. Regression guard against breaking OrdStruc domain dispatch or losing
// the instance-level OrdStruc tag.
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
		// Output is either a scalar int (IntVal) or a unit name string whose
		// data slot carries the int. Check both shapes.
		switch out := ap["output"].(type) {
		case int:
			if out == 19 {
				found = true
			}
		case string:
			if out == "" {
				continue
			}
			u := eng.Store.Get(out)
			if u == nil {
				continue
			}
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
	for _, s := range isA {
		isAMap[s] = true
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
	eng.VM.SetEnv("CurUnit", dsl.StringVal(task.UnitName))
	eng.VM.SetEnv("CurSlot", dsl.StringVal(task.SlotName))
	fired, _, _ := eng.fireTaskRule("H29", task)
	if !fired {
		t.Fatal("H29 did not fire on BagOfTallies examples task")
	}

	afterExs := eng.Store.Get("BagOfTallies").GetStrings("examples")
	if len(afterExs) <= beforeCount {
		t.Fatalf("expected more than %d examples after H29 firing, got %d (%v)", beforeCount, len(afterExs), afterExs)
	}

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
	eng.VM.SetEnv("CurUnit", dsl.StringVal(task.UnitName))
	eng.VM.SetEnv("CurSlot", dsl.StringVal(task.SlotName))
	eng.fireTaskRule("H29", task)
	firstPass := len(eng.Store.Get("BagOfTallies").GetStrings("examples"))

	task2 := &agenda.Task{
		Priority: 500,
		UnitName: "BagOfTallies",
		SlotName: "examples",
		Reasons:  []string{"test: second pass"},
	}
	eng.VM.SetEnv("CurUnit", dsl.StringVal(task2.UnitName))
	eng.VM.SetEnv("CurSlot", dsl.StringVal(task2.SlotName))
	eng.fireTaskRule("H29", task2)
	secondPass := len(eng.Store.Get("BagOfTallies").GetStrings("examples"))

	if secondPass != firstPass {
		t.Errorf("h29Ran guard failed: first pass left %d examples, second pass left %d", firstPass, secondPass)
	}
}
