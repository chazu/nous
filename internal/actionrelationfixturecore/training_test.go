package actionrelationfixturecore

import (
	"testing"

	"github.com/chazu/nous/internal/actionrelationoracle"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

func TestTrainingHasFrozenBalanceAndDiagnostics(t *testing.T) {
	cases, err := Training()
	if err != nil {
		t.Fatal(err)
	}
	labels := map[string]int{}
	seen := map[string]bool{}
	for _, testCase := range cases {
		a, err := actionrelations.ParseOccurrence(testCase.AOccurrence)
		if err != nil {
			t.Fatal(err)
		}
		b, err := actionrelations.ParseOccurrence(testCase.BOccurrence)
		if err != nil {
			t.Fatal(err)
		}
		observation, err := actionrelationoracle.Observe(testCase.State, mustActionJSON(a.Action), mustActionJSON(b.Action))
		if err != nil || observation.Label != testCase.Label {
			t.Fatalf("case %d label=%q oracle=%q err=%v", testCase.Ordinal, testCase.Label, observation.Label, err)
		}
		key := string(testCase.State) + string(testCase.AOccurrence) + string(testCase.BOccurrence)
		if seen[key] {
			t.Fatalf("duplicate case %d", testCase.Ordinal)
		}
		seen[key] = true
		labels[testCase.Label]++
	}
	if labels["commutes"] != 8 || labels["conflicts"] != 2 || len(cases) != 16 {
		t.Fatalf("labels=%v", labels)
	}
	for _, label := range []string{"a-enables-b", "b-enables-a", "a-disables-b", "b-disables-a", "mutual-disables", "inapplicable"} {
		if labels[label] != 1 {
			t.Fatalf("missing diagnostic %s: %v", label, labels)
		}
	}
}
