package transformexp

import (
	"bytes"
	"encoding/hex"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/transformfixturecore"
	"github.com/chazu/nous/internal/transformoracle"
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

func TestPublicPermutationAndPurposeStreamGoldenVectors(t *testing.T) {
	panelCommitment := digestBytes(mustJSON([]any{"transform-panel/v1", "development", 841001}))
	seedCommitment := digestBytes(mustJSON([]any{"transform-seed/v1", "development", uint64(841001)}))
	wantStreams := map[string][2]uint64{
		"structure":        {0x2051e0743fb114f7, 0xb9cc40b83d51fc1b},
		"aliases":          {0xee2f635ebed39fa3, 0xb8ec9eb16f841789},
		"scalars":          {0xeba47e7f1fad87b7, 0x253373ccf8d86669},
		"child-order":      {0xb7c2bf7d1b655a44, 0xb623a457136f0096},
		"case-tokens":      {0xc3ee98e89448c9a4, 0x553f364d9a07414c},
		"case-order":       {0x826576001885aa80, 0xf49312eba699baa5},
		"production-queue": {0x544baaf39bfd7c40, 0xc1615a8541006f07},
		"random-policy":    {0x975fb9b5ed9eca09, 0x63abf3ab9c1949b4},
		"baseline-ties":    {0x8f0f0c40154aaa53, 0x7fa3de7348c0a341},
	}
	for purpose, want := range wantStreams {
		rng := fixtureStream(panelCommitment, seedCommitment, 0, purpose)
		if got := [2]uint64{rng.Uint64(), rng.Uint64()}; got != want {
			t.Fatalf("%s stream=%016x want=%016x", purpose, got, want)
		}
	}
	panel, err := developmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	gotDevelopment := make([]int, len(panel))
	for index := range panel {
		gotDevelopment[index] = panel[index].Family
	}
	wantDevelopment := []int{1, 8, 7, 2, 2, 5, 4, 8, 0, 5, 5, 3, 0, 1, 7, 2, 0, 8, 2, 4, 1, 0, 6, 0, 0, 6, 3, 5, 4, 2, 6, 8, 3, 7, 2, 7, 6, 4, 1, 1, 6, 3, 1, 4, 7, 3, 8, 5}
	if !slices.Equal(gotDevelopment, wantDevelopment) {
		t.Fatalf("development family permutation=%v", gotDevelopment)
	}
	validation, err := validationPanel()
	if err != nil {
		t.Fatal(err)
	}
	gotValidation := make([]int, len(validation))
	for index := range validation {
		gotValidation[index] = validation[index].Family
	}
	wantValidation := []int{4, 0, 0, 6, 3, 8, 2, 2, 5, 8, 3, 4, 7, 3, 1, 0, 2, 1, 1, 2, 8, 5, 3, 3, 0, 8, 8, 0, 5, 0, 5, 7, 1, 4, 4, 4, 2, 0, 4, 8, 1, 5, 4, 5, 3, 7, 4, 5, 6, 1, 3, 7, 1, 0, 0, 8, 3, 6, 6, 1, 6, 3, 7, 1, 4, 2, 7, 7, 2, 6, 7, 6, 5, 2, 7, 3, 6, 8, 8, 7, 4, 2, 2, 5, 5, 2, 5, 4, 0, 1, 0, 3, 6, 6, 1, 8}
	if !slices.Equal(gotValidation, wantValidation) {
		t.Fatalf("validation family permutation=%v", gotValidation)
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

func TestGeneratorAcceptanceMatrixIsExactAndIndependentlyAudited(t *testing.T) {
	value, err := makeCurriculum(0, 8, 910008)
	if err != nil {
		t.Fatal(err)
	}
	ledger := value.GeneratorLedger
	if ledger.Applications != 72*16 || ledger.Work != 109161 || !digestString(ledger.MatrixSHA256) || !ledger.Accepted {
		t.Fatalf("generator ledger=%+v", ledger)
	}
	scorer, err := scorerFixtureBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := transformoracle.AuditAcceptance(value.Training, value.Heldout, scorer)
	if err != nil {
		t.Fatal(err)
	}
	if audit.Applications != ledger.Applications || audit.Work != ledger.Work || audit.MatrixSHA256 != ledger.MatrixSHA256 || audit.Accepted != ledger.Accepted {
		t.Fatalf("generator=%+v oracle=%+v", ledger, audit)
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
