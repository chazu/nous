package transformexp

import (
	"bytes"
	"testing"

	"github.com/chazu/nous/internal/transformfixturecore"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

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
