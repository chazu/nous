package causal

import "testing"

func TestUniverseAndEquivalenceClasses(t *testing.T) {
	universe := Enumerate()
	if len(universe) != 72 {
		t.Fatalf("universe=%d", len(universe))
	}
	classes := map[string]int{}
	for _, h := range universe {
		code, err := Code(h)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := Parse(code)
		if err != nil {
			t.Fatal(err)
		}
		sig, _ := Signature(code)
		classes[sig]++
		if len(parsed.Nodes) != 3 {
			t.Fatal("round trip")
		}
	}
	singles, pairs := 0, 0
	for _, size := range classes {
		if size == 1 {
			singles++
		} else if size == 2 {
			pairs++
		} else {
			t.Fatalf("class size=%d", size)
		}
	}
	if len(classes) != 58 || singles != 44 || pairs != 14 {
		t.Fatalf("classes=%d singles=%d pairs=%d", len(classes), singles, pairs)
	}
}

func TestActionOrderAndRuleCardinality(t *testing.T) {
	want := []string{"do:0=0", "do:0=1", "do:1=0", "do:1=1", "do:2=0", "do:2=1"}
	for i, action := range Actions() {
		if action.Code() != want[i] {
			t.Fatalf("action %d=%s", i, action.Code())
		}
	}
	if len(Rules()) != 40 {
		t.Fatalf("rules=%d", len(Rules()))
	}
}

func TestHandWitness(t *testing.T) {
	hs := []Hypothesis{
		{Nodes: []Node{{[]int{}, "constant-0"}, {[]int{0}, "copy"}, {[]int{1}, "copy"}}},
		{Nodes: []Node{{[]int{}, "constant-0"}, {[]int{}, "constant-0"}, {[]int{0}, "copy"}}},
		{Nodes: []Node{{[]int{}, "constant-0"}, {[]int{0}, "copy"}, {[]int{0, 1}, "or"}}},
	}
	var codes []string
	for _, h := range hs {
		code, err := Code(h)
		if err != nil {
			t.Fatal(err)
		}
		codes = append(codes, code)
	}
	a, _ := ParseAction("do:0=1")
	want := []string{"111", "101", "111"}
	for i, h := range hs {
		got, _ := Evaluate(h, &a)
		if OutcomeCode(got) != want[i] {
			t.Fatalf("h%d=%s", i+1, OutcomeCode(got))
		}
	}
	if !CompleteClass(codes, []string{codes[0], codes[2]}) {
		t.Fatal("expected complete pair")
	}
}
