package transformbaseline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/chazu/nous/internal/transformfixturecore"
)

func TestConcreteReplayUsesOnlyCanonicalInputDigest(t *testing.T) {
	batch := transformfixturecore.ProgramBatch{}
	var target []byte
	for index, value := range []string{"old", "cold", "warm", "mild"} {
		forest, _ := json.Marshal([]any{"typed-reference-forest/v1", []any{
			[]any{0, "group", -1, "", "", "", "", -1},
			[]any{1, "definition", 0, "d", value, "", "", -1},
		}})
		program, _ := json.Marshal([]any{"concrete-program/v1", []any{[]any{"set-value/v1", 1, "new"}}})
		digest := sha256.Sum256(forest)
		batch.Rows = append(batch.Rows, transformfixturecore.ProgramRow{Token: string([]byte("000000000000000")) + string(byte('0'+index)), BeforeDigest: hex.EncodeToString(digest[:]), Program: program})
		if index == 0 {
			target = forest
		}
	}
	encoded, err := batch.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	application, err := Replay(encoded, "ffffffffffffffff", target)
	if err != nil || application.Terminal != "applied" {
		t.Fatalf("digest hit with unrelated token = %+v, %v", application, err)
	}
}

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

func TestIndependentApplicationEventsMatchGoldenVectors(t *testing.T) {
	rows := []any{
		[]any{0, "group", -1, "", "", "", "", -1},
		[]any{1, "definition", 0, "d", "old", "", "", -1},
		[]any{2, "request", 0, "q", "", "old", "new", 1},
	}
	for id := 3; id <= 8; id++ {
		target := 1
		if id >= 6 {
			target = 9
		}
		rows = append(rows, []any{id, "reference", 0, []string{"a", "b", "c", "e", "f", "g"}[id-3], "old", "", "", target})
	}
	rows = append(rows,
		[]any{9, "definition", 0, "h", "other", "", "", -1},
		[]any{10, "decoy", 0, "i", "spare", "", "", -1},
		[]any{11, "decoy", 0, "j", "unused", "", "", -1},
	)
	forest, _ := json.Marshal([]any{"typed-reference-forest/v1", rows})
	for anchor, want := range map[string][12]int64{
		"request-target": {12, 8, 7, 0, 0, 0, 4, 4, 26, 12, 1, 1},
		"from-value":     {12, 8, 6, 0, 0, 0, 4, 4, 27, 12, 1, 1},
		"first-local":    {12, 9, 6, 0, 0, 0, 4, 4, 27, 12, 1, 1},
	} {
		schemaBytes := encodeSchema(schema{anchor, "definition+references", "global", "any", "required"})
		application, events, err := ApplySchemaMetered(forest, schemaBytes, "heldout")
		if err != nil || application.Terminal != "applied" {
			t.Fatalf("%s application=%+v err=%v", anchor, application, err)
		}
		_, comparisons, err := CompareOutputsMetered(application.Output, application.Output, "heldout")
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, comparisons...)
		var got [12]int64
		for _, event := range events {
			got[event.Category]++
		}
		if got != want {
			t.Fatalf("%s vector=%v want=%v", anchor, got, want)
		}
	}
}

func TestProfileBoundaryDoesNotImportSemantics(t *testing.T) {
	if len(transformfixturecore.ProfileDigest()) != 64 {
		t.Fatal("profile")
	}
}
