package transformbaseline

import (
	"testing"

	"github.com/chazu/nous/internal/transformfixturecore"
)

func TestSchemaUniverseAndOrdering(t *testing.T) {
	all := schemas()
	if len(all) != 72 {
		t.Fatalf("schemas=%d", len(all))
	}
	slicesCopy := append([]schema(nil), all...)
	// Sorting must put the least-description canonical tuple first.
	for i := 0; i < len(slicesCopy); i++ {
		for j := i + 1; j < len(slicesCopy); j++ {
			if compareSchema(slicesCopy[j], slicesCopy[i]) < 0 {
				slicesCopy[i], slicesCopy[j] = slicesCopy[j], slicesCopy[i]
			}
		}
	}
	first := string(encodeSchema(slicesCopy[0]))
	if first != `["transform-schema/v1","request-target","definition","local","any","none"]` {
		t.Fatal(first)
	}
}

func TestProfileBoundaryDoesNotImportSemantics(t *testing.T) {
	if len(transformfixturecore.ProfileDigest()) != 64 {
		t.Fatal("profile")
	}
}
