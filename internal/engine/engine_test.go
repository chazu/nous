package engine

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
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
