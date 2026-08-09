package nogoodfixture

import (
	"slices"
	"testing"

	"github.com/chazu/nous/internal/nogoodoracle"
)

func TestVariablePositionGoldenVectors(t *testing.T) {
	tests := []struct {
		panel   string
		root    any
		ordinal int
		n       int
		want    []int
	}{
		{"training", 831001, 0, 3, []int{0, 1, 2}},
		{"development", 832001, 0, 8, []int{5, 2, 1, 4, 3, 6, 0, 7}},
		{"validation", 833001, 0, 8, []int{6, 0, 1, 4, 7, 3, 5, 2}},
		{"locked", "0000000000000000000000000000000000000000000000000000000000000000", 0, 8, []int{5, 1, 6, 7, 0, 2, 4, 3}},
	}
	for _, test := range tests {
		got := permutation(test.n, stream(test.panel, test.root, test.ordinal, "variable-positions"))
		if !slices.Equal(got, test.want) {
			t.Fatalf("%s permutation = %v, want %v", test.panel, got, test.want)
		}
	}
}

func TestPanelCountsCellsAndOracleTruth(t *testing.T) {
	for _, panel := range []string{"development", "validation"} {
		tasks, err := Panel(panel)
		if err != nil {
			t.Fatal(err)
		}
		wantTotal := 96
		if panel == "validation" {
			wantTotal = 192
		}
		if len(tasks) != wantTotal {
			t.Fatalf("%s tasks = %d", panel, len(tasks))
		}
		cells := map[[3]int]int{}
		for _, task := range tasks {
			result, err := nogoodoracle.Enumerate(task.ProblemJSON, nogoodoracle.Literal{Variable: task.Decision.Variable, Color: task.Decision.Color})
			if err != nil {
				t.Fatalf("%s task %d: %v", panel, task.Ordinal, err)
			}
			wantSAT := task.Cohort == NearMiss || task.Cohort == Irrelevant
			if result.Satisfiable != wantSAT {
				t.Fatalf("%s task %d cohort %s satisfiable=%v", panel, task.Ordinal, task.Cohort, result.Satisfiable)
			}
			cohortIndex := 0
			switch task.Cohort {
			case NearMiss:
				cohortIndex = 1
			case Irrelevant:
				cohortIndex = 2
			case IndependentUnsat:
				cohortIndex = 3
			}
			bit := task.MissingBit
			cells[[3]int{cohortIndex, task.Template, bit}]++
		}
		multiplier := 1
		if panel == "validation" {
			multiplier = 2
		}
		for template := 0; template < 4; template++ {
			if cells[[3]int{0, template, -1}] != 14*multiplier {
				t.Fatalf("%s reusable template %d count=%d", panel, template, cells[[3]int{0, template, -1}])
			}
			for bit := 0; bit < 3; bit++ {
				if cells[[3]int{1, template, bit}] != 2*multiplier {
					t.Fatalf("%s near template %d bit %d count=%d", panel, template, bit, cells[[3]int{1, template, bit}])
				}
			}
			if cells[[3]int{2, template, -1}] != 2*multiplier || cells[[3]int{3, template, -1}] != 2*multiplier {
				t.Fatalf("%s nonreusable template %d counts=%d/%d", panel, template, cells[[3]int{2, template, -1}], cells[[3]int{3, template, -1}])
			}
		}
	}
}

func TestLockedRootValidationAndCounts(t *testing.T) {
	if _, err := LockedPanel("bad"); err == nil {
		t.Fatal("accepted bad locked root")
	}
	tasks, err := LockedPanel("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 384 {
		t.Fatalf("locked tasks = %d", len(tasks))
	}
	counts := map[Cohort]int{}
	for _, task := range tasks {
		counts[task.Cohort]++
	}
	if counts[Reusable] != 312 || counts[NearMiss] != 48 || counts[Irrelevant] != 12 || counts[IndependentUnsat] != 12 {
		t.Fatalf("locked counts = %#v", counts)
	}
}
