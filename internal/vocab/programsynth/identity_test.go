package programsynth

import "testing"

func TestDecisionIdentityPreservesOrderAndRepetition(t *testing.T) {
	first, err := DecisionKey("ordered/v1", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	for _, sequence := range [][]string{{"b", "a"}, {"a", "a"}, {"a"}} {
		other, err := DecisionKey("ordered/v1", sequence)
		if err != nil {
			t.Fatal(err)
		}
		if other == first {
			t.Fatalf("sequence %v reused ordered identity", sequence)
		}
	}
	alias, _ := DecisionKey("ordered/v1", []string{"a", "b"})
	if alias != first {
		t.Fatal("same semantic sequence changed identity")
	}
}

func TestIdentityBounds(t *testing.T) {
	if ValidSequence(nil) || ValidSequence([]string{"bad key"}) || ValidSequence([]string{"a", "b", "c", "d"}) {
		t.Fatal("invalid sequence accepted")
	}
}
