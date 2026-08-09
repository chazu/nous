package transformexp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/chazu/nous/internal/transformfixturecore"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

var errRejectedCurriculumAttempt = errors.New("rejected curriculum attempt")

type expectedCase struct {
	Token, Terminal string
	Output          []byte
}

type curriculum struct {
	Ordinal          int
	Family           int
	Seed             uint64
	SeedCommitment   string
	PanelCommitment  string
	AcceptedAttempt  int
	Panel            string
	Training         []byte
	Heldout          []byte
	Expected         []expectedCase
	Latent           []byte
	PolicyTokens     map[Policy]string
	PolicyRandomness map[Policy][2]uint64
}

var familySchemas = []transformschema.Schema{
	{Anchor: "request-target", Targets: "definition", ReferenceScope: "local", OldGuard: "any", Locality: "required"},
	{Anchor: "request-target", Targets: "references", ReferenceScope: "local", OldGuard: "equals-from", Locality: "required"},
	{Anchor: "request-target", Targets: "references", ReferenceScope: "local", OldGuard: "any", Locality: "required"},
	{Anchor: "request-target", Targets: "references", ReferenceScope: "global", OldGuard: "equals-from", Locality: "required"},
	{Anchor: "request-target", Targets: "references", ReferenceScope: "global", OldGuard: "any", Locality: "required"},
	{Anchor: "request-target", Targets: "definition+references", ReferenceScope: "local", OldGuard: "equals-from", Locality: "required"},
	{Anchor: "request-target", Targets: "definition+references", ReferenceScope: "local", OldGuard: "any", Locality: "required"},
	{Anchor: "request-target", Targets: "definition+references", ReferenceScope: "global", OldGuard: "equals-from", Locality: "required"},
	{Anchor: "request-target", Targets: "definition+references", ReferenceScope: "global", OldGuard: "any", Locality: "required"},
}

func developmentPanel() ([]curriculum, error) {
	return publicPanel("development", 841001, []int{6, 6, 6, 5, 5, 5, 5, 5, 5})
}

func validationPanel() ([]curriculum, error) {
	return publicPanel("validation", 842001, []int{11, 11, 11, 11, 11, 11, 10, 10, 10})
}

// lockedPanel is deliberately unexported. Its sole production caller is the
// repository guard, after that guard has durably claimed the locked attempt.
func lockedPanel(root []byte) ([]curriculum, [][2]uint64, error) {
	if len(root) != sha256.Size {
		return nil, nil, fmt.Errorf("locked root must contain 32 bytes")
	}
	counts := []int{15, 15, 14, 14, 14, 14, 14, 14, 14}
	materials := make([][]byte, LockedCount)
	seeds := make([]uint64, LockedCount)
	for ordinal := range seeds {
		materials[ordinal] = lockedHMAC(root, []any{"part3/transform-schema/v1", "locked-curriculum", ordinal})
		seeds[ordinal] = binary.BigEndian.Uint64(materials[ordinal][:8])
	}
	families := make([]int, 0, LockedCount)
	for family, count := range counts {
		for range count {
			families = append(families, family)
		}
	}
	permutationDigest := lockedHMAC(root, []any{"part3/transform-schema/v1", "family-permutation", "locked"})
	rng := rand.New(rand.NewPCG(binary.BigEndian.Uint64(permutationDigest[:8]), binary.BigEndian.Uint64(permutationDigest[8:16])))
	rng.Shuffle(len(families), func(i, j int) { families[i], families[j] = families[j], families[i] })
	panelCommitment := hex.EncodeToString(lockedHMAC(root, []any{"transform-panel/v1", "locked"}))
	panel := make([]curriculum, LockedCount)
	for ordinal := range panel {
		value, err := makeCurriculumWithAuthority(ordinal, families[ordinal], seeds[ordinal], panelCommitment, digestBytes(materials[ordinal]))
		if err != nil {
			return nil, nil, fmt.Errorf("locked curriculum %d: %w", ordinal, err)
		}
		value.Panel = "locked"
		panel[ordinal] = value
	}
	pairs := make([][2]uint64, 20000)
	for replicate := 0; replicate < 10000; replicate++ {
		for purposeIndex, purpose := range []string{"bootstrap/nous-vs-pbe", "randomization/nous-vs-pbe"} {
			material := lockedHMAC(root, []any{"statistics", "locked", replicate, purpose})
			index := replicate
			if purposeIndex == 1 {
				index += 10000
			}
			pairs[index] = [2]uint64{binary.BigEndian.Uint64(material[:8]), binary.BigEndian.Uint64(material[8:16])}
		}
	}
	for index := range seeds {
		seeds[index] = 0
		for byteIndex := range materials[index] {
			materials[index][byteIndex] = 0
		}
	}
	return panel, pairs, nil
}

func lockedHMAC(root []byte, preimage []any) []byte {
	mac := hmac.New(sha256.New, root)
	mac.Write(mustJSON(preimage))
	return mac.Sum(nil)
}

func publicPanel(panel string, start uint64, counts []int) ([]curriculum, error) {
	var families []int
	for family, count := range counts {
		for range count {
			families = append(families, family)
		}
	}
	seed := sha256.Sum256(mustJSON([]any{"part3/transform-schema/v1", "family-permutation", panel, start}))
	rng := rand.New(rand.NewPCG(binary.BigEndian.Uint64(seed[:8]), binary.BigEndian.Uint64(seed[8:16])))
	rng.Shuffle(len(families), func(i, j int) { families[i], families[j] = families[j], families[i] })
	panelCommitment := digestBytes(mustJSON([]any{"transform-panel/v1", panel, start}))
	out := make([]curriculum, len(families))
	for i, family := range families {
		curriculumSeed := start + uint64(i)
		seedCommitment := digestBytes(mustJSON([]any{"transform-seed/v1", panel, curriculumSeed}))
		c, err := makeCurriculumWithAuthority(i, family, curriculumSeed, panelCommitment, seedCommitment)
		if err != nil {
			return nil, fmt.Errorf("curriculum %d: %w", i, err)
		}
		c.Panel = panel
		out[i] = c
	}
	return out, nil
}

func makeCurriculum(ordinal, family int, seed uint64) (curriculum, error) {
	panelCommitment := digestBytes(mustJSON([]any{"transform-panel/v1", "safe", seed}))
	seedCommitment := digestBytes(mustJSON([]any{"transform-seed/v1", "safe", seed}))
	return makeCurriculumWithAuthority(ordinal, family, seed, panelCommitment, seedCommitment)
}

func makeCurriculumWithAuthority(ordinal, family int, seed uint64, panelCommitment, seedCommitment string) (curriculum, error) {
	for attempt := 0; attempt < 100; attempt++ {
		value, err := generateCurriculumAttempt(ordinal, family, seed, panelCommitment, seedCommitment, attempt)
		if errors.Is(err, errRejectedCurriculumAttempt) {
			continue
		}
		return value, err
	}
	return curriculum{}, fmt.Errorf("curriculum generator exhausted 100 attempts")
}

func generateCurriculumAttempt(ordinal, family int, seed uint64, panelCommitment, seedCommitment string, attempt int) (curriculum, error) {
	if family < 0 || family >= len(familySchemas) {
		return curriculum{}, fmt.Errorf("family %d", family)
	}
	latent := familySchemas[family]
	latentBytes, _ := latent.CanonicalJSON()
	training := transformfixturecore.Training{ProfileDigest: transformfixturecore.ProfileDigest()}
	heldout := transformfixturecore.Heldout{ProfileDigest: transformfixturecore.ProfileDigest()}
	var expected []expectedCase
	aliasRNG := fixtureStream(panelCommitment, seedCommitment, attempt, "aliases")
	aliasSeed := aliasRNG.Uint64()
	for i := 0; i < 4; i++ {
		before := positiveForest(aliasSeed, i, false)
		result, err := latent.Apply(before)
		if err != nil || result.Terminal != "applied" {
			return curriculum{}, errRejectedCurriculumAttempt
		}
		beforeBytes, _ := before.CanonicalJSON()
		afterBytes, _ := result.Output.CanonicalJSON()
		token := fixtureToken(panelCommitment, seedCommitment, attempt, i)
		training.Cases = append(training.Cases, transformfixturecore.TrainingCase{Token: token, Kind: "positive", Before: beforeBytes, After: afterBytes})
	}
	for i, kind := range []string{"zero", "two", "noop", "wrong-context"} {
		before := negativeForest(aliasSeed, i, false, latent, kind)
		beforeBytes, _ := before.CanonicalJSON()
		training.Cases = append(training.Cases, transformfixturecore.TrainingCase{Token: fixtureToken(panelCommitment, seedCommitment, attempt, 4+i), Kind: "abstain", Before: beforeBytes})
	}
	for i := 0; i < 4; i++ {
		before := positiveForest(aliasSeed, i, true)
		result, _ := latent.Apply(before)
		beforeBytes, _ := before.CanonicalJSON()
		output, _ := result.Output.CanonicalJSON()
		token := fixtureToken(panelCommitment, seedCommitment, attempt, 8+i)
		heldout.Cases = append(heldout.Cases, transformfixturecore.HeldoutCase{Token: token, Before: beforeBytes})
		expected = append(expected, expectedCase{token, "applied", output})
	}
	for i, kind := range []string{"zero", "two", "noop", "wrong-context"} {
		before := negativeForest(aliasSeed, i, true, latent, kind)
		beforeBytes, _ := before.CanonicalJSON()
		token := fixtureToken(panelCommitment, seedCommitment, attempt, 12+i)
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
	if !fixtureInputsDisjoint(training, heldout) {
		return curriculum{}, errRejectedCurriculumAttempt
	}
	unique, err := uniqueMinimum(training, latentBytes)
	if err != nil || !unique {
		if err != nil {
			return curriculum{}, err
		}
		return curriculum{}, errRejectedCurriculumAttempt
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
	policyTokens := map[Policy]string{}
	policyRandomness := map[Policy][2]uint64{}
	queueRNG := fixtureStream(panelCommitment, seedCommitment, attempt, "production-queue")
	randomRNG := fixtureStream(panelCommitment, seedCommitment, attempt, "random-policy")
	for _, policy := range empiricalPolicies {
		policyTokens[policy] = fmt.Sprintf("%016x", queueRNG.Uint64())
		policyRandomness[policy] = [2]uint64{randomRNG.Uint64(), randomRNG.Uint64()}
	}
	return curriculum{Ordinal: ordinal, Family: family, Seed: seed, SeedCommitment: seedCommitment, PanelCommitment: panelCommitment, AcceptedAttempt: attempt, Training: trainingBytes, Heldout: heldoutBytes, Expected: expected, Latent: latentBytes, PolicyTokens: policyTokens, PolicyRandomness: policyRandomness}, nil
}

func fixtureInputsDisjoint(training transformfixturecore.Training, heldout transformfixturecore.Heldout) bool {
	seen := map[string]bool{}
	for _, item := range training.Cases {
		digest := digestBytes(item.Before)
		if seen[digest] {
			return false
		}
		seen[digest] = true
	}
	for _, item := range heldout.Cases {
		digest := digestBytes(item.Before)
		if seen[digest] {
			return false
		}
		seen[digest] = true
	}
	return true
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

func positiveForest(seed uint64, variant int, heldout bool) transformschema.Forest {
	trainingWords := [][5]string{{"base", "old", "new", "other", "spare"}, {"start", "prior", "next", "aside", "extra"}, {"plain", "early", "later", "alien", "decoy"}, {"root", "before", "after", "remote", "unused"}}
	heldoutWords := [][5]string{{"alpha", "amber", "azure", "ashen", "apple"}, {"brisk", "brown", "blue", "beige", "birch"}, {"clear", "coral", "cyan", "cream", "cedar"}, {"dense", "dark", "dawn", "dull", "drift"}}
	words := trainingWords
	if heldout {
		words = heldoutWords
	}
	w := words[(int(seed%uint64(len(words)))+variant)%len(words)]
	f := transformschema.Forest{Nodes: []transformschema.Node{
		{ID: 0, Kind: "group", Parent: -1, Target: -1},
		{ID: 1, Kind: "definition", Parent: 0, Key: "decoy", Value: w[4], Target: -1},
		{ID: 2, Kind: "definition", Parent: 0, Key: "service", Value: w[0], Target: -1},
		{ID: 3, Kind: "request", Parent: 0, Key: "change", From: w[1], To: w[2], Target: 2},
		{ID: 4, Kind: "reference", Parent: 0, Key: "local", Value: w[1], Target: 2},
		{ID: 5, Kind: "reference", Parent: 0, Key: "aside", Value: w[3], Target: 2},
		{ID: 6, Kind: "group", Parent: -1, Target: -1},
		{ID: 7, Kind: "reference", Parent: 6, Key: "remote", Value: w[1], Target: 2},
		{ID: 8, Kind: "reference", Parent: 6, Key: "far", Value: w[3], Target: 2},
	}}
	if heldout {
		keys := []string{"mirror", "object", "update", "near", "side", "distant", "farther", "vacant"}
		for index := range f.Nodes {
			if f.Nodes[index].Kind != "group" {
				f.Nodes[index].Key = keys[index-1]
			}
		}
	}
	if variant == 0 {
		f.Nodes = f.Nodes[:6]
	}
	if variant == 3 {
		f.Nodes[1].Value = w[1]
	}
	// A combined/global/any schema must stay within the four-edit grammar while
	// the batch collectively distinguishes local/global and equals/any.
	if variant%2 == 0 {
		if len(f.Nodes) > 8 {
			f.Nodes[8].Target = 1
		}
	} else {
		f.Nodes[5].Target = 1
	}
	if variant == 2 {
		mapping := []int{6, 3, 7, 1, 8, 2, 0, 5, 4}
		for index := range f.Nodes {
			node := &f.Nodes[index]
			node.ID = mapping[node.ID]
			if node.Parent >= 0 {
				node.Parent = mapping[node.Parent]
			}
			if node.Target >= 0 {
				node.Target = mapping[node.Target]
			}
		}
	}
	return f
}

func negativeForest(seed uint64, variant int, heldout bool, latent transformschema.Schema, kind string) transformschema.Forest {
	f := positiveForest(seed, variant, heldout)
	find := func(kind string) *transformschema.Node {
		for index := range f.Nodes {
			if f.Nodes[index].Kind == kind {
				return &f.Nodes[index]
			}
		}
		return nil
	}
	request := find("request")
	var definition *transformschema.Node
	for index := range f.Nodes {
		if f.Nodes[index].ID == request.Target {
			definition = &f.Nodes[index]
			break
		}
	}
	switch kind {
	case "zero":
		value := "quiet"
		if heldout {
			value = "silent"
		}
		*request = transformschema.Node{ID: request.ID, Kind: "decoy", Parent: request.Parent, Key: request.Key, Value: value, Target: -1}
	case "two":
		key := "second"
		if heldout {
			key = "double"
		}
		f.Nodes = append(f.Nodes, transformschema.Node{ID: len(f.Nodes), Kind: "request", Parent: request.Parent, Key: key, From: request.From, To: request.To, Target: request.Target})
	case "noop":
		if latent.Targets == "definition" || latent.Targets == "definition+references" {
			request.To = definition.Value
		} else {
			for index := range f.Nodes {
				if f.Nodes[index].Kind == "reference" && f.Nodes[index].Parent == request.Parent && f.Nodes[index].Target == request.Target {
					request.To = f.Nodes[index].Value
					break
				}
			}
		}
	case "wrong-context":
		remoteGroup := -1
		for _, node := range f.Nodes {
			if node.Kind == "group" && node.ID != request.Parent {
				remoteGroup = node.ID
				break
			}
		}
		if remoteGroup < 0 {
			remoteGroup = len(f.Nodes)
			f.Nodes = append(f.Nodes, transformschema.Node{ID: remoteGroup, Kind: "group", Parent: -1, Target: -1})
		}
		definition.Parent = remoteGroup
		definition.Key = "external"
		if heldout {
			definition.Key = "outside"
		}
	}
	return f
}

func fixtureStream(panelCommitment, seedCommitment string, attempt int, purpose string) *rand.Rand {
	d := sha256.Sum256(mustJSON([]any{"part3/transform-schema/v1", panelCommitment, seedCommitment, attempt, purpose}))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(d[:8]), binary.BigEndian.Uint64(d[8:16])))
}

func fixtureToken(panelCommitment, seedCommitment string, attempt, index int) string {
	rng := fixtureStream(panelCommitment, seedCommitment, attempt, "case-tokens")
	for range index {
		_ = rng.Uint64()
	}
	return fmt.Sprintf("%016x", rng.Uint64())
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
