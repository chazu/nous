package ruleinduction

import "testing"

func TestDefinitionCardinality(t *testing.T) {
	definitions := EnumerateDefinitions()
	if len(definitions) != 15 {
		t.Fatalf("definitions = %d, want 15", len(definitions))
	}
	codes, err := CanonicalCodes(definitions)
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index < len(codes); index++ {
		if codes[index-1] >= codes[index] {
			t.Fatalf("codes not unique and sorted: %v", codes)
		}
	}
	if got := len(definitions) + len(definitions)*len(definitions); got != 240 {
		t.Fatalf("joint grammar = %d, want 240", got)
	}
}

func TestPartialTreeAndOneFieldRefinement(t *testing.T) {
	levels := []int{1}
	current := []Partial{RootPartial()}
	for depth := 0; depth < 4; depth++ {
		var next []Partial
		for _, partial := range current {
			children, err := RefineOne(partial)
			if err != nil {
				t.Fatal(err)
			}
			for _, child := range children {
				if child.Bound() != partial.Bound()+1 {
					t.Fatalf("refinement bound %d fields", child.Bound()-partial.Bound())
				}
			}
			next = append(next, children...)
		}
		levels = append(levels, len(next))
		current = next
	}
	want := []int{1, 2, 6, 12, 36}
	for index := range want {
		if levels[index] != want[index] {
			t.Fatalf("levels = %v, want %v", levels, want)
		}
	}
	legal := map[string]bool{}
	for _, partial := range current {
		definition, err := partial.Definition()
		if err != nil {
			continue
		}
		code, _ := definition.Code()
		legal[code] = true
	}
	if len(legal) != 15 {
		t.Fatalf("legal complete definitions = %d, want 15", len(legal))
	}
}

func TestBeneficialWitnessClosure(t *testing.T) {
	var background [PredicateCount]Relation
	for _, edge := range [][2]int{{0, 1}, {1, 2}, {1, 3}, {3, 4}} {
		background[0].Add(edge[0], edge[1])
	}
	definition := Definition{Clauses: [2]Clause{
		{Kind: Identity, Background: 0},
		{Kind: TailRec, Background: 0},
	}}
	got, work, err := Evaluate(definition, background)
	if err != nil {
		t.Fatal(err)
	}
	wantPairs := [][2]int{{0, 1}, {0, 2}, {0, 3}, {0, 4}, {1, 2}, {1, 3}, {1, 4}, {3, 4}}
	var want Relation
	for _, item := range wantPairs {
		want.Add(item[0], item[1])
	}
	if got != want {
		t.Fatalf("closure\n got %s\nwant %s", got.Signature(), want.Signature())
	}
	if work.FixedPointTotal() <= 0 || work.FixedPointTotal() > 3560 {
		t.Fatalf("fixed-point work = %d", work.FixedPointTotal())
	}
}

func TestMalformedDefinitionsFail(t *testing.T) {
	tests := []Definition{
		{Clauses: [2]Clause{{Kind: Identity, Background: 0}, {Kind: Identity, Background: 0}}},
		{Clauses: [2]Clause{{Kind: Identity, Background: -1}, {Kind: TailRec, Background: 0}}},
		{Clauses: [2]Clause{{Kind: ClauseKind(9), Background: 0}, {Kind: TailRec, Background: 0}}},
	}
	for _, test := range tests {
		if _, _, err := Evaluate(test, [PredicateCount]Relation{}); err == nil {
			t.Fatalf("Evaluate(%+v) succeeded", test)
		}
	}
}

func TestStructuralSubsumptionIsReflexive(t *testing.T) {
	for _, definition := range EnumerateDefinitions() {
		ok, work, err := StructurallySubsumes(definition, definition)
		if err != nil || !ok || work <= 0 || work > 96 {
			t.Fatalf("subsumes = (%t,%d,%v)", ok, work, err)
		}
	}
}
