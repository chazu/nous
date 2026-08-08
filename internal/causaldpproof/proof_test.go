package causaldpproof

import (
	"bytes"
	"os"
	"testing"

	"github.com/chazu/nous/internal/causalv2"
)

func TestRunMatchesGolden(t *testing.T) {
	report, err := Run()
	if err != nil {
		t.Fatal(err)
	}
	if report.Combinatorial.ReachableStateBound != wantReachableStateBound ||
		report.Combinatorial.TotalWorkBound != wantTotalWorkBound {
		t.Fatalf("analytical bounds=%+v", report.Combinatorial)
	}
	if report.ExactDP.Models != wantModels ||
		report.ExactDP.ObservationalClasses != wantObservationalClasses ||
		report.ExactDP.ProductionCounts != report.ExactDP.IndependentCounts {
		t.Fatalf("exact DP report=%+v", report.ExactDP)
	}
	encoded, err := causalv2.CanonicalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile("testdata/report.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	golden = bytes.TrimSuffix(golden, []byte("\n"))
	if !bytes.Equal(encoded, golden) {
		t.Fatalf("proof report changed\n got: %s\nwant: %s", encoded, golden)
	}
}
