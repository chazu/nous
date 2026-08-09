package transformexp

import (
	"slices"
	"testing"

	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

func TestTransformationStatisticsAreDeterministicAndStratified(t *testing.T) {
	rows := make([]pairedTransformRow, 18)
	for i := range rows {
		rows[i] = pairedTransformRow{Ordinal: i, Family: i % 9, NousSuccess: true, PBESuccess: i%3 != 0, NonmatchingNousWork: 4, NonmatchingPBEWork: 5}
	}
	first, err := computeTransformInference(rows, "development", 841001, 1000, 1000)
	if err != nil {
		t.Fatal(err)
	}
	second, err := computeTransformInference(rows, "development", 841001, 1000, 1000)
	if err != nil || first != second {
		t.Fatalf("nondeterministic inference first=%+v second=%+v err=%v", first, second, err)
	}
	if first.Point != (rationalPoint{6, 18}) || first.NousSuccesses != 18 || first.PBESuccesses != 12 || first.NonmatchingNous != 72 || first.NonmatchingPBE != 90 {
		t.Fatalf("inference=%+v", first)
	}
	broken := slices.Clone(rows)
	for i := range broken {
		broken[i].Family = 0
	}
	if _, err := computeTransformInference(broken, "development", 841001, 100, 100); err == nil {
		t.Fatal("accepted empty family strata")
	}
}

func TestFrozenPBEAnswerRanks(t *testing.T) {
	candidates := transformschema.Schemas()
	slices.SortFunc(candidates, compareProductionSchemas)
	ranks := make([]int, len(familySchemas))
	for family, schema := range familySchemas {
		ranks[family] = slices.Index(candidates, schema) + 1
	}
	want := []int{4, 15, 7, 31, 17, 32, 19, 51, 34}
	if !slices.Equal(ranks, want) {
		t.Fatalf("answer ranks=%v want=%v", ranks, want)
	}
}

func compareProductionSchemas(a, b transformschema.Schema) int {
	if difference := schemaDescription(a) - schemaDescription(b); difference != 0 {
		return difference
	}
	orders := [][]string{
		{"request-target", "from-value", "first-local"},
		{"definition", "references", "definition+references"},
		{"local", "global"},
		{"equals-from", "any"},
		{"required", "none"},
	}
	left := []string{a.Anchor, a.Targets, a.ReferenceScope, a.OldGuard, a.Locality}
	right := []string{b.Anchor, b.Targets, b.ReferenceScope, b.OldGuard, b.Locality}
	for i := range left {
		if difference := slices.Index(orders[i], left[i]) - slices.Index(orders[i], right[i]); difference != 0 {
			return difference
		}
	}
	return 0
}
