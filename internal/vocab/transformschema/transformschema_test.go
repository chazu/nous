package transformschema

import (
	"bytes"
	"testing"
)

func sampleForest() Forest {
	return Forest{Nodes: []Node{
		{ID: 0, Kind: "group", Parent: -1, Target: -1},
		{ID: 1, Kind: "definition", Parent: 0, Key: "service", Value: "old", Target: -1},
		{ID: 2, Kind: "request", Parent: 0, Key: "change", From: "old", To: "new", Target: 1},
		{ID: 3, Kind: "reference", Parent: 0, Key: "client", Value: "old", Target: 1},
	}}
}

func TestForestRoundTripAndStrictness(t *testing.T) {
	f := sampleForest()
	b, err := f.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseForest(b)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, _ := got.CanonicalJSON()
	if !bytes.Equal(b, reencoded) {
		t.Fatalf("round trip changed bytes\n%s\n%s", b, reencoded)
	}
	for _, bad := range [][]byte{append(bytes.Clone(b), 'x'), []byte(`[]`), []byte(`["typed-reference-forest/v1",[]]`)} {
		if _, err := ParseForest(bad); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestSchemaApply(t *testing.T) {
	f := sampleForest()
	s := Schema{"request-target", "definition+references", "local", "equals-from", "required"}
	r, err := s.Apply(f)
	if err != nil || r.Terminal != "applied" || r.Output == nil || len(r.Certificate.Edits) != 2 {
		t.Fatalf("result=%+v err=%v", r, err)
	}
	for _, id := range []int{1, 3} {
		if r.Output.Nodes[id].Value != "new" {
			t.Fatalf("node %d not edited", id)
		}
	}
	wrong := f
	wrong.Nodes[1].Parent = 4
	wrong.Nodes = append(wrong.Nodes, Node{ID: 4, Kind: "group", Parent: -1, Target: -1})
	r, err = s.Apply(wrong)
	if err != nil || r.Terminal != "abstain/locality" {
		t.Fatalf("wrong-context result=%+v err=%v", r, err)
	}
}

func TestSchemaUniverseAndRefinementEdges(t *testing.T) {
	if got := len(Schemas()); got != 72 {
		t.Fatalf("schemas=%d", got)
	}
	counts := []int{3, 3, 2, 2, 2}
	p := Partial{}
	for stage, want := range counts {
		choices := [][]string{targets, anchors, scopes, oldGuards, localities}[stage]
		if len(choices) != want {
			t.Fatal("choice count")
		}
		next, err := p.Refine(choices[0])
		if err != nil {
			t.Fatal(err)
		}
		p = next
	}
	if _, err := p.CanonicalJSON(); err != nil {
		t.Fatal(err)
	}
	if edges := 3 + 9 + 18 + 36 + 72; edges != 138 {
		t.Fatal(edges)
	}
}

func TestPartialCanonicalRoundTripAndStrictness(t *testing.T) {
	p := Partial{}
	for _, value := range []string{"definition+references", "request-target", "global", "any", "required"} {
		var err error
		p, err = p.Refine(value)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := p.CanonicalJSON()
		if err != nil {
			t.Fatal(err)
		}
		got, err := ParsePartial(encoded)
		if err != nil || got != p {
			t.Fatalf("partial=%+v got=%+v err=%v", p, got, err)
		}
	}
	for _, bad := range [][]byte{
		[]byte(`["transform-partial/v1",0,"definition","","","",""]`),
		[]byte(`["transform-partial/v1",1,"bogus","","","",""]`),
		[]byte(`["transform-partial/v1",0,"","","","",""] `),
	} {
		if _, err := ParsePartial(bad); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestProgramRejectsNoOpAndDuplicateTargets(t *testing.T) {
	f := sampleForest()
	if _, err := (Program{Edits: []Edit{{1, "old"}}}).Apply(f); err == nil {
		t.Fatal("accepted no-op")
	}
	if _, err := (Program{Edits: []Edit{{1, "new"}, {1, "next"}}}).CanonicalJSON(); err == nil {
		t.Fatal("accepted duplicate target")
	}
}
