package ruleinductionoracle

import "testing"

func TestIndependentWitness(t *testing.T) {
	var background [3]Relation
	for _, edge := range [][2]int{{0, 1}, {1, 2}, {1, 3}, {3, 4}} {
		background[0].Add(edge[0], edge[1])
	}
	got := Evaluate(Definition{{Background: 0}, {Recursive: true, Background: 0}}, background)
	if got.Signature() != "0111100000111000000000000000100000000000000000000000000000000000" {
		t.Fatalf("unexpected closure %s", got.Signature())
	}
}

func TestIndependentCardinality(t *testing.T) {
	if got := len(Definitions()); got != 15 {
		t.Fatalf("definitions = %d", got)
	}
	theories := JointTheories()
	shared := 0
	for _, theory := range theories {
		if theory.Shared {
			shared++
		}
	}
	if len(theories) != 240 || shared != 15 || len(theories)-shared != 225 {
		t.Fatalf("joint theories total/shared/direct = %d/%d/%d", len(theories), shared, len(theories)-shared)
	}
}
