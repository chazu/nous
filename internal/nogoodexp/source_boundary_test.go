package nogoodexp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLockedFixtureGenerationHasOneProductionCallSurface(t *testing.T) {
	fixtureSource, err := os.ReadFile(filepath.Join("..", "nogoodfixture", "fixture.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(fixtureSource), "LockedPanel") || strings.Contains(string(fixtureSource), `panel == "locked"`) || strings.Contains(string(fixtureSource), `"locked":`) {
		t.Fatal("public fixture package exposes a locked-panel surface")
	}
	coreSource, err := os.ReadFile(filepath.Join("..", "nogoodfixturecore", "utility.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{`"locked"`, "Locked", "private root", "SeedAuthority", "part3/nogoods"} {
		if strings.Contains(string(coreSource), forbidden) {
			t.Fatalf("generic fixture constructor contains protected authority token %q", forbidden)
		}
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	productionCalls := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		productionCalls += strings.Count(string(data), "lockedPanel(")
	}
	// One definition in locked_fixture.go and one call in the guarded executor.
	if productionCalls != 2 {
		t.Fatalf("locked fixture production surfaces = %d, want definition plus one guarded call", productionCalls)
	}
}

func TestValidationFixtureGenerationHasOneProductionCallSurface(t *testing.T) {
	for _, relative := range []string{filepath.Join("..", "nogoodfixture", "fixture.go"), filepath.Join("..", "nogoodfixture", "competence.go")} {
		data, err := os.ReadFile(relative)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), `"validation"`) || strings.Contains(string(data), "ValidationPanel") || strings.Contains(string(data), "ValidationCompetence") {
			t.Fatalf("public fixture source %s exposes validation construction", relative)
		}
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	panelSurfaces, competenceSurfaces := 0, 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		panelSurfaces += strings.Count(string(data), "validationPanel(")
		competenceSurfaces += strings.Count(string(data), "validationCompetence(")
	}
	if panelSurfaces != 2 || competenceSurfaces != 2 {
		t.Fatalf("validation production surfaces panel=%d competence=%d, want one definition and guarded caller each", panelSurfaces, competenceSurfaces)
	}
}

func TestProtectedPanelObservationHasOnlyGuardedExportedExecutors(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"func BuildDevelopmentEvidence(", "func RunDevelopmentExecution(", "func PersistDevelopmentEvidence(", "func InferPanel("}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(entry.Name())
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, surface := range forbidden {
			if strings.Contains(string(data), surface) {
				t.Fatalf("unguarded protected observation surface %q in %s", surface, entry.Name())
			}
		}
	}
}
