package transformexp

import (
	"reflect"
	"testing"
)

func TestPolicyViewExcludesSealedTruthAndHeldoutPayload(t *testing.T) {
	viewType := reflect.TypeOf(policyCurriculum{})
	for _, forbidden := range []string{"Family", "Seed", "SeedCommitment", "AcceptedAttempt", "Latent", "Expected", "Heldout"} {
		if _, ok := viewType.FieldByName(forbidden); ok {
			t.Fatalf("policy view exposes sealed field %s", forbidden)
		}
	}

	c, err := makeCurriculum(0, 0, 841300)
	if err != nil {
		t.Fatal(err)
	}
	c.Heldout = []byte("not heldout JSON")
	view, err := decodePolicyView(c)
	if err != nil {
		t.Fatalf("training-only policy decode inspected heldout payload: %v", err)
	}
	if view.HeldoutDigest != digestBytes(c.Heldout) {
		t.Fatal("policy view did not retain the committed heldout digest")
	}
	if _, err := decodeHeldoutInputs(c); err == nil {
		t.Fatal("heldout release accepted malformed payload")
	}
}
