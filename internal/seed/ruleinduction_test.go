package seed

import (
	"context"
	"io"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
)

func loadRuleInduction(t *testing.T) *unit.Store {
	t.Helper()
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "ruleinduction"); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestRuleInductionSeedFreezesStageOneArtifact(t *testing.T) {
	store := loadRuleInduction(t)
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = 500
	eng.MutConfig.Enabled = false
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	experiment := store.Get("RuleInductionSeed")
	if got := experiment.GetString("terminal"); got != "awaiting-stage-2" {
		t.Fatalf("terminal = %q", got)
	}
	if got := experiment.GetString("frozenCode"); got != "03" {
		t.Fatalf("frozen code = %q", got)
	}
	if len(experiment.GetString("frozenSignature")) != 64 {
		t.Fatalf("signature length = %d", len(experiment.GetString("frozenSignature")))
	}
	if ag.Len() != 0 {
		t.Fatalf("agenda length = %d", ag.Len())
	}
	candidates := 0
	for _, name := range store.All() {
		if store.Get(name).GetString("experiment") == experiment.Name && store.IsA(name, "RuleInductionCandidate") {
			candidates++
		}
	}
	if candidates != 15 {
		t.Fatalf("candidates = %d, want 15", candidates)
	}
}

func TestRuleInductionStageTwoReusesFrozenPredicate(t *testing.T) {
	store := loadRuleInduction(t)
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = 500
	eng.MutConfig.Enabled = false
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	experiment := store.Get("RuleInductionSeed")
	for index, example := range []struct {
		x, y     int
		positive bool
	}{{0, 2, true}, {0, 4, true}, {2, 0, false}, {4, 0, false}} {
		u := unit.New("RIStageTwoExample" + string(rune('A'+index)))
		u.Set("isA", []string{"RuleInductionExample", "Anything"})
		u.Set("experiment", experiment.Name)
		u.Set("stage", "stage2")
		u.Set("x", example.x)
		u.Set("y", example.y)
		u.Set("positive", example.positive)
		store.Put(u)
	}
	experiment.Set("stage", "stage2")
	experiment.Set("rootName", "RI.Partial.RuleInductionSeed.stage2.root")
	ag.Push(&agenda.Task{Priority: 950, UnitName: experiment.Name, SlotName: "riContinue"})
	eng.MaxCycles = 500
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := experiment.GetString("terminal"); got != "identified" {
		t.Fatalf("terminal = %q", got)
	}
	if !experiment.GetBool("usedFrozenLibrary") || experiment.GetBool("fellBack") {
		t.Fatalf("reuse/fallback = %v/%v", experiment.GetBool("usedFrozenLibrary"), experiment.GetBool("fellBack"))
	}
	if experiment.GetString("selectedCode") != "03" {
		t.Fatalf("selected = %q", experiment.GetString("selectedCode"))
	}
}

func TestRuleInductionStageTwoFallsBackToLocalSearch(t *testing.T) {
	store := loadRuleInduction(t)
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = 500
	eng.MutConfig.Enabled = false
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	experiment := store.Get("RuleInductionSeed")
	for index, example := range []struct {
		x, y     int
		positive bool
	}{{5, 6, true}, {0, 4, false}, {0, 1, false}, {6, 7, false}} {
		u := unit.New("RIFallbackExample" + string(rune('A'+index)))
		u.Set("isA", []string{"RuleInductionExample", "Anything"})
		u.Set("experiment", experiment.Name)
		u.Set("stage", "stage2")
		u.Set("x", example.x)
		u.Set("y", example.y)
		u.Set("positive", example.positive)
		store.Put(u)
	}
	experiment.Set("stage", "stage2")
	experiment.Set("rootName", "RI.Partial.RuleInductionSeed.stage2.root")
	ag.Push(&agenda.Task{Priority: 950, UnitName: experiment.Name, SlotName: "riContinue"})
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := experiment.GetString("terminal"); got != "identified" {
		t.Fatalf("terminal = %q", got)
	}
	if !experiment.GetBool("fellBack") || experiment.GetBool("usedFrozenLibrary") {
		t.Fatalf("fallback/reuse = %v/%v", experiment.GetBool("fellBack"), experiment.GetBool("usedFrozenLibrary"))
	}
	if experiment.GetString("selectedCode") == experiment.GetString("frozenCode") {
		t.Fatalf("fallback retained frozen code %q", experiment.GetString("frozenCode"))
	}
}
