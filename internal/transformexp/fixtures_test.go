package transformexp

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/chazu/nous/internal/transformfixturecore"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

func TestFrozenLockedDerivationGoldenVectors(t *testing.T) {
	root, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	vectors := []struct {
		Preimage []any
		Want     string
	}{
		{[]any{"part3/transform-schema/v1", "family-permutation", "locked"}, "d6de2f17cd2c339d64319b33f4ab6d114ffc27af6fe5de639d8644b02ae82555"},
		{[]any{"part3/transform-schema/v1", "locked-curriculum", 0}, "bde0c0fd318cfb28a628dc98525e641a222b8de1f923f4f127b4475f7003e8d3"},
		{[]any{"statistics", "locked", 0, "bootstrap/nous-vs-pbe"}, "80c6071cbe6dc9fcbba5f6a8ad0133c43ae581f76633f7d1539ea92c8cfc3d95"},
		{[]any{"transform-panel/v1", "locked"}, "a89bcc5abedf45759afdb0dbf6bddf7e8c60b588cdbf40d12780065e2970da77"},
	}
	for _, vector := range vectors {
		if got := hex.EncodeToString(lockedHMAC(root, vector.Preimage)); got != vector.Want {
			t.Fatalf("HMAC(%v)=%s want %s", vector.Preimage, got, vector.Want)
		}
	}
}

func TestCurriculumHeldoutInputsAreDistinctFromAllTrainingInputs(t *testing.T) {
	for family := range familySchemas {
		value, err := makeCurriculum(family, family, 910000+uint64(family))
		if err != nil {
			t.Fatal(err)
		}
		training, _ := transformfixturecore.ParseTraining(value.Training)
		heldout, _ := transformfixturecore.ParseHeldout(value.Heldout)
		if !fixtureInputsDisjoint(training, heldout) {
			t.Fatalf("family %d contains a repeated forest", family)
		}
		trainingSymbols := fixtureSymbols(t, training.Cases)
		for _, item := range heldout.Cases {
			forest, err := transformschema.ParseForest(item.Before)
			if err != nil {
				t.Fatal(err)
			}
			for _, node := range forest.Nodes {
				for _, symbol := range []string{node.Key, node.Value, node.From, node.To} {
					if symbol != "" && trainingSymbols[symbol] {
						t.Fatalf("family %d heldout symbol %q appears in training", family, symbol)
					}
				}
			}
		}
	}
}

func fixtureSymbols(t *testing.T, cases []transformfixturecore.TrainingCase) map[string]bool {
	t.Helper()
	result := map[string]bool{}
	for _, item := range cases {
		forest, err := transformschema.ParseForest(item.Before)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range forest.Nodes {
			for _, symbol := range []string{node.Key, node.Value, node.From, node.To} {
				if symbol != "" {
					result[symbol] = true
				}
			}
		}
	}
	return result
}

func TestGeneratedPositiveStructuralVariantsAreValidAndHeldoutNovel(t *testing.T) {
	training := map[string]bool{}
	for variant := 0; variant < 4; variant++ {
		value := positiveForest(841001, variant, false)
		if err := value.Validate(); err != nil {
			t.Fatalf("training variant %d: %v: %+v", variant, err, value)
		}
		encoded, _ := value.CanonicalJSON()
		training[digestBytes(encoded)] = true
	}
	for variant := 0; variant < 4; variant++ {
		value := positiveForest(841001, variant, true)
		if err := value.Validate(); err != nil {
			t.Fatalf("heldout variant %d: %v: %+v", variant, err, value)
		}
		encoded, _ := value.CanonicalJSON()
		if training[digestBytes(encoded)] {
			t.Fatalf("heldout variant %d duplicates training", variant)
		}
	}
}

func TestDevelopmentPanelCountsAndTruth(t *testing.T) {
	panel, err := developmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	if len(panel) != 48 {
		t.Fatalf("count=%d", len(panel))
	}
	counts := make([]int, 9)
	for _, c := range panel {
		counts[c.Family]++
		training, err := transformfixturecore.ParseTraining(c.Training)
		if err != nil {
			t.Fatal(err)
		}
		latent, err := transformschema.ParseSchema(c.Latent)
		if err != nil {
			t.Fatal(err)
		}
		for _, example := range training.Cases {
			f, _ := transformschema.ParseForest(example.Before)
			r, e := latent.Apply(f)
			if e != nil {
				t.Fatal(e)
			}
			if example.Kind == "positive" {
				out, _ := r.Output.CanonicalJSON()
				if r.Terminal != "applied" || !bytes.Equal(out, example.After) {
					t.Fatalf("family %d positive mismatch", c.Family)
				}
			} else if len(r.Terminal) < 8 || r.Terminal[:8] != "abstain/" {
				t.Fatalf("family %d negative=%s", c.Family, r.Terminal)
			}
		}
	}
	want := []int{6, 6, 6, 5, 5, 5, 5, 5, 5}
	for i := range want {
		if counts[i] != want[i] {
			t.Fatalf("family %d=%d", i, counts[i])
		}
	}
}

func TestManifestAndGeneratorDeterminism(t *testing.T) {
	if err := validateManifest(); err != nil {
		t.Fatal(err)
	}
	a, err := developmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	b, err := developmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	for i := range a {
		if a[i].Family != b[i].Family || !bytes.Equal(a[i].Training, b[i].Training) || !bytes.Equal(a[i].Heldout, b[i].Heldout) {
			t.Fatalf("curriculum %d drifted", i)
		}
	}
}

func TestEveryFamilyHasUniqueMinimumTrainingSchema(t *testing.T) {
	for family := range familySchemas {
		c, err := makeCurriculum(family, family, 900000+uint64(family))
		if err != nil {
			t.Fatal(err)
		}
		training, _ := transformfixturecore.ParseTraining(c.Training)
		best := 1 << 30
		var winners [][]byte
		for _, candidate := range transformschema.Schemas() {
			exact := true
			for _, example := range training.Cases {
				f, _ := transformschema.ParseForest(example.Before)
				r, _ := candidate.Apply(f)
				if example.Kind == "positive" {
					if r.Output == nil {
						exact = false
						break
					}
					out, _ := r.Output.CanonicalJSON()
					if r.Terminal != "applied" || !bytes.Equal(out, example.After) {
						exact = false
						break
					}
				} else if len(r.Terminal) < 8 || r.Terminal[:8] != "abstain/" {
					exact = false
					break
				}
			}
			if !exact {
				continue
			}
			cost := schemaCost(candidate)
			encoded, _ := candidate.CanonicalJSON()
			if cost < best {
				best = cost
				winners = [][]byte{encoded}
			} else if cost == best {
				winners = append(winners, encoded)
			}
		}
		if len(winners) != 1 || !bytes.Equal(winners[0], c.Latent) {
			t.Fatalf("family %d winners=%q latent=%s", family, winners, c.Latent)
		}
	}
}

func schemaCost(s transformschema.Schema) int {
	cost := map[string]int{"request-target": 1, "from-value": 2, "first-local": 3, "definition": 1, "references": 1, "definition+references": 2, "local": 1, "global": 2, "equals-from": 2, "any": 1, "required": 2, "none": 1}
	return cost[s.Anchor] + cost[s.Targets] + cost[s.ReferenceScope] + cost[s.OldGuard] + cost[s.Locality]
}
