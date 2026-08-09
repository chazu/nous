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
		{[]any{"part3/transform-schema/v2", "family-permutation", "locked"}, "23fac563f45ebb1ac1242762dc0714f0d1d1aab3f5a35898b192b213fdd16619"},
		{[]any{"part3/transform-schema/v2", "locked-curriculum", 0}, "4c222114d212ca13dc7f27d2eabb4f6b64047ed41966d17b0e47994db9f38db8"},
		{[]any{"transform-panel/v1", "locked"}, "a89bcc5abedf45759afdb0dbf6bddf7e8c60b588cdbf40d12780065e2970da77"},
	}
	pairs, err := lockedStatisticsPairs(digestBytes(root))
	if err != nil || pairs[0] != [2]uint64{0xcc2105fc441a86b0, 0xd0b8dd6bffe90cae} {
		t.Fatalf("locked public statistics pair=%x err=%v", pairs[0], err)
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
		"structure":        {0x694b1b8f9a450694, 0x2f6b4f5c76cfc00d},
		"aliases":          {0x38f419a5ab92a7fd, 0x885ad2b60af482e8},
		"scalars":          {0xa61253e93eb85866, 0xb26676a2c46f9111},
		"child-order":      {0xa0e849281ceaa48f, 0x530ab82beac4a823},
		"case-tokens":      {0x0559ebafea716e76, 0x6c92340b81e5f568},
		"case-order":       {0xcad1b1adf1ead774, 0x15cd9a92fafaed4e},
		"production-queue": {0xf82ba2d2230a82e1, 0xd987b43b08271f2f},
		"random-policy":    {0x6029a6a81c832f08, 0x8c1467f89a093789},
		"baseline-ties":    {0xaa366cbe9a3fe639, 0xe278d57d06abc64b},
	}
	for purpose, want := range wantStreams {
		rng := fixtureStream(panelCommitment, seedCommitment, 0, purpose)
		if got := [2]uint64{rng.Uint64(), rng.Uint64()}; got != want {
			t.Fatalf("%s stream=%016x want=%016x", purpose, got, want)
		}
	}
	gotDevelopment := publicFamilyPermutation("development", 841001, []int{6, 6, 6, 5, 5, 5, 5, 5, 5})
	wantDevelopment := []int{2, 5, 8, 3, 4, 2, 3, 6, 6, 5, 4, 8, 1, 0, 2, 1, 2, 7, 8, 0, 3, 8, 0, 7, 6, 5, 2, 1, 5, 1, 0, 7, 7, 4, 0, 6, 1, 0, 4, 3, 6, 7, 8, 4, 1, 2, 3, 5}
	if !slices.Equal(gotDevelopment, wantDevelopment) {
		t.Fatalf("development family permutation=%v", gotDevelopment)
	}
	gotValidation := publicFamilyPermutation("validation", 842001, []int{11, 11, 11, 11, 11, 11, 10, 10, 10})
	wantValidation := []int{2, 5, 0, 1, 8, 7, 4, 3, 4, 0, 4, 3, 6, 4, 8, 7, 3, 0, 6, 3, 7, 8, 3, 4, 7, 8, 5, 1, 3, 2, 4, 5, 5, 5, 1, 6, 5, 7, 5, 0, 3, 8, 2, 4, 5, 8, 6, 6, 5, 1, 0, 2, 1, 0, 0, 0, 3, 1, 2, 6, 6, 1, 5, 8, 1, 4, 8, 8, 4, 1, 7, 7, 2, 5, 2, 4, 1, 2, 2, 7, 0, 0, 7, 1, 3, 2, 3, 6, 4, 3, 2, 0, 7, 6, 6, 8}
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

func TestPublicFamilyCountsAndSafeCurriculumTruth(t *testing.T) {
	panel := publicFamilyPermutation("development", 841001, []int{6, 6, 6, 5, 5, 5, 5, 5, 5})
	counts := make([]int, 9)
	for _, family := range panel {
		counts[family]++
	}
	for family := range familySchemas {
		c, err := makeCurriculum(family, family, 990000+uint64(family))
		if err != nil {
			t.Fatal(err)
		}
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
					t.Fatalf("family %d positive mismatch", family)
				}
			} else if len(r.Terminal) < 8 || r.Terminal[:8] != "abstain/" {
				t.Fatalf("family %d negative=%s", family, r.Terminal)
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
	for family := range familySchemas {
		a, err := makeCurriculum(family, family, 991000+uint64(family))
		if err != nil {
			t.Fatal(err)
		}
		b, err := makeCurriculum(family, family, 991000+uint64(family))
		if err != nil {
			t.Fatal(err)
		}
		if a.Family != b.Family || !bytes.Equal(a.Training, b.Training) || !bytes.Equal(a.Heldout, b.Heldout) {
			t.Fatalf("curriculum %d drifted", family)
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
