package transformexp

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/chazu/nous/internal/transformfixturecore"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

type expectedCase struct {
	Token, Terminal string
	Output          []byte
}

type curriculum struct {
	Ordinal  int
	Family   int
	Seed     uint64
	Panel    string
	Training []byte
	Heldout  []byte
	Expected []expectedCase
	Latent   []byte
}

var familySchemas = []transformschema.Schema{
	{"request-target", "definition", "local", "any", "required"},
	{"request-target", "references", "local", "equals-from", "required"},
	{"request-target", "references", "local", "any", "required"},
	{"request-target", "references", "global", "equals-from", "required"},
	{"request-target", "references", "global", "any", "required"},
	{"request-target", "definition+references", "local", "equals-from", "required"},
	{"request-target", "definition+references", "local", "any", "required"},
	{"request-target", "definition+references", "global", "equals-from", "required"},
	{"request-target", "definition+references", "global", "any", "required"},
}

func developmentPanel() ([]curriculum, error) {
	return publicPanel("development", 841001, []int{6, 6, 6, 5, 5, 5, 5, 5, 5})
}

func validationPanel() ([]curriculum, error) {
	return publicPanel("validation", 842001, []int{11, 11, 11, 11, 11, 11, 10, 10, 10})
}

func publicPanel(panel string, start uint64, counts []int) ([]curriculum, error) {
	var families []int
	for family, count := range counts {
		for range count {
			families = append(families, family)
		}
	}
	seed := sha256.Sum256(mustJSON([]any{"part3/transform-schema/v1", panel, "family-permutation", start}))
	rng := rand.New(rand.NewPCG(binary.BigEndian.Uint64(seed[:8]), binary.BigEndian.Uint64(seed[8:16])))
	rng.Shuffle(len(families), func(i, j int) { families[i], families[j] = families[j], families[i] })
	out := make([]curriculum, len(families))
	for i, family := range families {
		c, err := makeCurriculum(i, family, start+uint64(i))
		if err != nil {
			return nil, fmt.Errorf("curriculum %d: %w", i, err)
		}
		c.Panel = panel
		out[i] = c
	}
	return out, nil
}

func makeCurriculum(ordinal, family int, seed uint64) (curriculum, error) {
	if family < 0 || family >= len(familySchemas) {
		return curriculum{}, fmt.Errorf("family %d", family)
	}
	latent := familySchemas[family]
	latentBytes, _ := latent.CanonicalJSON()
	training := transformfixturecore.Training{ProfileDigest: transformfixturecore.ProfileDigest()}
	heldout := transformfixturecore.Heldout{ProfileDigest: transformfixturecore.ProfileDigest()}
	var expected []expectedCase
	for i := 0; i < 4; i++ {
		before := positiveForest(seed, i)
		result, err := latent.Apply(before)
		if err != nil || result.Terminal != "applied" {
			return curriculum{}, fmt.Errorf("latent positive: %s %v", result.Terminal, err)
		}
		beforeBytes, _ := before.CanonicalJSON()
		afterBytes, _ := result.Output.CanonicalJSON()
		token := caseToken(seed, "train-positive", i)
		training.Cases = append(training.Cases, transformfixturecore.TrainingCase{Token: token, Kind: "positive", Before: beforeBytes, After: afterBytes})
	}
	for i, kind := range []string{"zero", "two", "noop", "wrong-context"} {
		before := negativeForest(seed, i, latent, kind)
		beforeBytes, _ := before.CanonicalJSON()
		training.Cases = append(training.Cases, transformfixturecore.TrainingCase{Token: caseToken(seed, "train-abstain", i), Kind: "abstain", Before: beforeBytes})
	}
	for i := 0; i < 4; i++ {
		before := positiveForest(seed+10000, i)
		result, _ := latent.Apply(before)
		beforeBytes, _ := before.CanonicalJSON()
		output, _ := result.Output.CanonicalJSON()
		token := caseToken(seed, "held-positive", i)
		heldout.Cases = append(heldout.Cases, transformfixturecore.HeldoutCase{Token: token, Before: beforeBytes})
		expected = append(expected, expectedCase{token, "applied", output})
	}
	for i, kind := range []string{"zero", "two", "noop", "wrong-context"} {
		before := negativeForest(seed+10000, i, latent, kind)
		beforeBytes, _ := before.CanonicalJSON()
		token := caseToken(seed, "held-abstain", i)
		heldout.Cases = append(heldout.Cases, transformfixturecore.HeldoutCase{Token: token, Before: beforeBytes})
		expected = append(expected, expectedCase{token, "abstain", nil})
	}
	trainingBytes, err := training.CanonicalJSON()
	if err != nil {
		return curriculum{}, err
	}
	heldoutBytes, err := heldout.CanonicalJSON()
	if err != nil {
		return curriculum{}, err
	}
	unique, err := uniqueMinimum(training, latentBytes)
	if err != nil || !unique {
		return curriculum{}, fmt.Errorf("latent is not unique minimum: %v", err)
	}
	slices.SortFunc(expected, func(a, b expectedCase) int {
		if a.Token < b.Token {
			return -1
		}
		if a.Token > b.Token {
			return 1
		}
		return 0
	})
	return curriculum{Ordinal: ordinal, Family: family, Seed: seed, Training: trainingBytes, Heldout: heldoutBytes, Expected: expected, Latent: latentBytes}, nil
}

func uniqueMinimum(training transformfixturecore.Training, latent []byte) (bool, error) {
	best := int(^uint(0) >> 1)
	var winners [][]byte
	for _, candidate := range transformschema.Schemas() {
		exact := true
		for _, example := range training.Cases {
			f, err := transformschema.ParseForest(example.Before)
			if err != nil {
				return false, err
			}
			r, err := candidate.Apply(f)
			if err != nil {
				return false, err
			}
			if example.Kind == "positive" {
				if r.Output == nil {
					exact = false
					break
				}
				out, _ := r.Output.CanonicalJSON()
				if r.Terminal != "applied" || !slices.Equal(out, example.After) {
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
		cost := schemaDescription(candidate)
		encoded, _ := candidate.CanonicalJSON()
		if cost < best {
			best = cost
			winners = [][]byte{encoded}
		} else if cost == best {
			winners = append(winners, encoded)
		}
	}
	return len(winners) == 1 && slices.Equal(winners[0], latent), nil
}

func schemaDescription(s transformschema.Schema) int {
	cost := map[string]int{"request-target": 1, "from-value": 2, "first-local": 3, "definition": 1, "references": 1, "definition+references": 2, "local": 1, "global": 2, "equals-from": 2, "any": 1, "required": 2, "none": 1}
	return cost[s.Anchor] + cost[s.Targets] + cost[s.ReferenceScope] + cost[s.OldGuard] + cost[s.Locality]
}

func positiveForest(seed uint64, variant int) transformschema.Forest {
	words := [][5]string{{"base", "old", "new", "other", "spare"}, {"start", "prior", "next", "aside", "extra"}, {"plain", "early", "later", "alien", "decoy"}, {"root", "before", "after", "remote", "unused"}}
	w := words[(int(seed)+variant)%len(words)]
	f := transformschema.Forest{Nodes: []transformschema.Node{
		{ID: 0, Kind: "group", Parent: -1, Target: -1},
		{ID: 1, Kind: "definition", Parent: 0, Key: "decoy", Value: w[1], Target: -1},
		{ID: 2, Kind: "definition", Parent: 0, Key: "service", Value: w[0], Target: -1},
		{ID: 3, Kind: "request", Parent: 0, Key: "change", From: w[1], To: w[2], Target: 2},
		{ID: 4, Kind: "reference", Parent: 0, Key: "local", Value: w[1], Target: 2},
		{ID: 5, Kind: "reference", Parent: 0, Key: "aside", Value: w[3], Target: 2},
		{ID: 6, Kind: "group", Parent: -1, Target: -1},
		{ID: 7, Kind: "reference", Parent: 6, Key: "remote", Value: w[1], Target: 2},
		{ID: 8, Kind: "reference", Parent: 6, Key: "far", Value: w[3], Target: 2},
	}}
	// A combined/global/any schema must stay within the four-edit grammar while
	// the batch collectively distinguishes local/global and equals/any.
	if variant%2 == 0 {
		f.Nodes[8].Target = 1
	} else {
		f.Nodes[5].Target = 1
	}
	return f
}

func negativeForest(seed uint64, variant int, latent transformschema.Schema, kind string) transformschema.Forest {
	f := positiveForest(seed, variant)
	switch kind {
	case "zero":
		f.Nodes[3] = transformschema.Node{ID: 3, Kind: "decoy", Parent: 0, Key: "change", Value: "quiet", Target: -1}
	case "two":
		f.Nodes = append(f.Nodes, transformschema.Node{ID: 9, Kind: "request", Parent: 0, Key: "second", From: f.Nodes[3].From, To: f.Nodes[3].To, Target: 2})
	case "noop":
		if latent.Targets == "definition" || latent.Targets == "definition+references" {
			f.Nodes[3].To = f.Nodes[2].Value
		} else {
			f.Nodes[3].To = f.Nodes[4].Value
		}
	case "wrong-context":
		f.Nodes[2].Parent = 6
		f.Nodes[2].Key = "external"
	}
	return f
}

func caseToken(seed uint64, purpose string, i int) string {
	d := sha256.Sum256(mustJSON([]any{"part3/transform-schema/v1", seed, purpose, i}))
	return hex.EncodeToString(d[:8])
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
