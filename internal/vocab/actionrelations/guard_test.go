package actionrelations

import "testing"

func TestGuardSpaceAndRefinementTreeAreFrozen(t *testing.T) {
	guards := EnumerateGuards()
	if len(guards) != 451 {
		t.Fatalf("guard count=%d want 451", len(guards))
	}
	digests := map[string]bool{}
	edges := 0
	for _, guard := range guards {
		if err := guard.Validate(); err != nil {
			t.Fatal(err)
		}
		digest, _ := guard.Digest()
		if digests[digest] {
			t.Fatalf("duplicate guard %s", digest)
		}
		digests[digest] = true
		parent, ok, err := guard.Parent()
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			edges++
			if err := parent.Validate(); err != nil || len(parent.Literals) != len(guard.Literals)-1 {
				t.Fatalf("bad parent %#v: %v", parent, err)
			}
		}
	}
	if edges != 450 {
		t.Fatalf("edge count=%d want 450", edges)
	}
}

func TestFactsAndAtoms(t *testing.T) {
	state := twoCellState(0, 3)
	state.Events = []string{"seen"}
	a := Occurrence{Action: SemanticAction{Kind: "add", XRole: "c0", N: 1}}
	b := Occurrence{Action: SemanticAction{Kind: "add", XRole: "c0", N: -1}}
	af, err := Facts(state, a)
	if err != nil {
		t.Fatal(err)
	}
	bf, err := Facts(state, b)
	if err != nil {
		t.Fatal(err)
	}
	wants := map[string]bool{
		"read-write-disjoint":     false,
		"primary-same":            true,
		"argument-equal":          false,
		"argument-opposite":       true,
		"shared-value-zero":       true,
		"shared-value-max":        false,
		"a-primary-zero":          true,
		"combined-adds-in-bounds": false,
	}
	for atom, want := range wants {
		got, err := EvaluateAtom(atom, af, bf)
		if err != nil || got != want {
			t.Errorf("%s=%v want %v err=%v", atom, got, want, err)
		}
	}
	guard := Guard{Literals: []Literal{{Atom: "primary-same", Polarity: true}, {Atom: "argument-opposite", Polarity: true}}}
	// Guard order is the frozen atom order, not lexical string order.
	guard = Guard{Literals: []Literal{{Atom: "primary-same", Polarity: true}, {Atom: "argument-opposite", Polarity: true}}}
	matched, err := guard.Evaluate(af, bf)
	if err != nil || !matched {
		t.Fatalf("guard match=%v err=%v", matched, err)
	}
}

func TestPatternErasesRolesButPreservesAliasTopology(t *testing.T) {
	a := Occurrence{Action: SemanticAction{Kind: "transfer", XRole: "c2", YRole: "c0", N: 1}}
	b := Occurrence{Action: SemanticAction{Kind: "swap", XRole: "c0", YRole: "c1"}}
	pattern, err := PatternFor(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(pattern.Roles) != 4 || pattern.Roles[0] != 0 || pattern.Roles[1] != 1 || pattern.Roles[2] != 2 || pattern.Roles[3] != 0 {
		t.Fatalf("unexpected alias topology %#v", pattern)
	}
}
