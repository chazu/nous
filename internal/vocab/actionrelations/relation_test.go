package actionrelations

import "testing"

func TestRelationAndArtifactRoundTrip(t *testing.T) {
	digestA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	relations := []Relation{
		{Pattern: Pattern{Kinds: []string{"set", "set"}, Roles: []int{0, -1, 1, -1}}, Guard: Guard{}, PositiveObservations: []string{digestA}, NegativeObservations: []string{digestB}},
		{Pattern: Pattern{Kinds: []string{"add", "add"}, Roles: []int{0, -1, 0, -1}}, Guard: Guard{Literals: []Literal{{Atom: "combined-adds-in-bounds", Polarity: true}}}, PositiveObservations: []string{digestB}, NegativeObservations: []string{}},
	}
	resolved := map[string]Relation{}
	for _, relation := range relations {
		encoded, err := relation.CanonicalJSON()
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseRelation(encoded)
		if err != nil {
			t.Fatal(err)
		}
		digest, _ := parsed.Digest()
		resolved[digest] = parsed
	}
	artifact, err := NewArtifact(relations, digestA)
	if err != nil {
		t.Fatal(err)
	}
	if err := artifact.ValidateResolved(resolved); err != nil {
		t.Fatal(err)
	}
	encoded, _ := artifact.CanonicalJSON()
	parsed, err := ParseArtifact(encoded)
	if err != nil || parsed.SemanticTrainingRoot != digestA {
		t.Fatalf("artifact=%#v err=%v", parsed, err)
	}
}

func TestArtifactRejectsDigestOrderThatIsNotRelationByteOrder(t *testing.T) {
	root := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	first := Relation{Pattern: Pattern{Kinds: []string{"add", "add"}, Roles: []int{0, -1, 1, -1}}, Guard: Guard{}}
	second := Relation{Pattern: Pattern{Kinds: []string{"set", "set"}, Roles: []int{0, -1, 1, -1}}, Guard: Guard{}}
	artifact, err := NewArtifact([]Relation{second, first}, root)
	if err != nil {
		t.Fatal(err)
	}
	resolved := map[string]Relation{}
	for _, relation := range []Relation{first, second} {
		digest, _ := relation.Digest()
		resolved[digest] = relation
	}
	artifact.RelationDigests[0], artifact.RelationDigests[1] = artifact.RelationDigests[1], artifact.RelationDigests[0]
	if err := artifact.ValidateResolved(resolved); err == nil {
		t.Fatal("accepted artifact in digest rather than relation-byte order")
	}
}
