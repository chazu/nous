package nogoodexp

import (
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
)

func TestLockedFixtureGenerationIsGuardLocalAndDeterministic(t *testing.T) {
	if _, err := lockedPanel("bad"); err == nil {
		t.Fatal("accepted bad locked root")
	}
	root := "0000000000000000000000000000000000000000000000000000000000000000"
	first, err := lockedPanel(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := lockedPanel(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != LockedTaskCount || len(second) != LockedTaskCount {
		t.Fatalf("locked task counts = %d/%d", len(first), len(second))
	}
	counts := map[nogoodfixture.Cohort]int{}
	for index := range first {
		counts[first[index].Cohort]++
		if string(first[index].ProblemJSON) != string(second[index].ProblemJSON) || first[index].Decision != second[index].Decision {
			t.Fatalf("locked task %d is nondeterministic", index)
		}
	}
	if counts[nogoodfixture.Reusable] != 312 || counts[nogoodfixture.NearMiss] != 48 || counts[nogoodfixture.Irrelevant] != 12 || counts[nogoodfixture.IndependentUnsat] != 12 {
		t.Fatalf("locked counts = %#v", counts)
	}
}
