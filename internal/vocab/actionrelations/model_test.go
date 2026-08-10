package actionrelations

import (
	"bytes"
	"testing"
)

func TestParsersRequireCanonicalBytes(t *testing.T) {
	canonical := []byte(`["finite-action-state/v1",[["c0",0]],[]]`)
	if _, err := ParseState(canonical); err != nil {
		t.Fatalf("canonical state: %v", err)
	}
	for _, invalid := range [][]byte{
		[]byte(`[ "finite-action-state/v1",[["c0",0]],[]]`),
		[]byte(`["finite-action-state/v1",[["c0",0.0]],[]]`),
		[]byte(`["finite-action-state/v1",[["c1",0],["c0",1]],[]]`),
	} {
		if _, err := ParseState(invalid); err == nil {
			t.Fatalf("accepted noncanonical state %s", invalid)
		}
	}
}

func TestWholeWorldNormalizationErasesPresentation(t *testing.T) {
	left := World{
		State: State{Cells: []Cell{{Name: "alice", Value: 3}, {Name: "bob", Value: 0}}},
		Actions: []Action{
			{Name: "give", Kind: "transfer", X: "alice", Y: "bob", N: 1},
			{Name: "take", Kind: "claim", X: "bob"},
		},
	}
	right := World{
		State: State{Cells: []Cell{{Name: "moon", Value: 0}, {Name: "sun", Value: 3}}},
		Actions: []Action{
			{Name: "renamed", Kind: "claim", X: "moon"},
			{Name: "other", Kind: "transfer", X: "sun", Y: "moon", N: 1},
		},
	}
	a, err := left.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	b, err := right.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	ab, _ := a.CanonicalJSON()
	bb, _ := b.CanonicalJSON()
	if !bytes.Equal(ab, bb) {
		t.Fatalf("alpha-equivalent worlds differ:\n%s\n%s", ab, bb)
	}
}

func TestOccurrenceOrdinalsAreSemanticAndConsecutive(t *testing.T) {
	a := SemanticAction{Kind: "claim", XRole: "c0"}
	b := SemanticAction{Kind: "emit", Symbol: "x"}
	occurrences, err := AssignOccurrences([]SemanticAction{a, b, a})
	if err != nil {
		t.Fatal(err)
	}
	if occurrences[0].Action != a || occurrences[0].Ordinal != 0 || occurrences[1].Action != a || occurrences[1].Ordinal != 1 || occurrences[2].Action != b || occurrences[2].Ordinal != 0 {
		t.Fatalf("unexpected occurrences: %#v", occurrences)
	}
}
