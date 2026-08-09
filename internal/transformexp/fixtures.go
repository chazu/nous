package transformexp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

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
	GeneratorLedger  acceptanceLedger
	Queue            []byte
	Scorer           []byte
}

type acceptanceLedger struct {
	Applications int
	Work         int64
	MatrixSHA256 string
	Accepted     bool
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
	shuffleFixtureValues(rng, families)
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
	shuffleFixtureValues(rng, families)
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
	structureRNG := fixtureStream(panelCommitment, seedCommitment, attempt, "structure")
	aliasRNG := fixtureStream(panelCommitment, seedCommitment, attempt, "aliases")
	scalarRNG := fixtureStream(panelCommitment, seedCommitment, attempt, "scalars")
	childRNG := fixtureStream(panelCommitment, seedCommitment, attempt, "child-order")
	caseOrderRNG := fixtureStream(panelCommitment, seedCommitment, attempt, "case-order")
	structureSeed, aliasSeed, scalarSeed, childSeed := structureRNG.Uint64(), aliasRNG.Uint64(), scalarRNG.Uint64(), childRNG.Uint64()
	trainingOrder := []int{0, 1, 2, 3, 4, 5, 6, 7}
	heldoutOrder := []int{0, 1, 2, 3, 4, 5, 6, 7}
	shuffleFixtureValues(caseOrderRNG, trainingOrder)
	shuffleFixtureValues(caseOrderRNG, heldoutOrder)
	for position, spec := range trainingOrder {
		if spec >= 4 {
			continue
		}
		before := streamedPositiveForest(structureSeed, aliasSeed, scalarSeed, childSeed, spec, false)
		result, err := latent.Apply(before)
		if err != nil || result.Terminal != "applied" {
			return curriculum{}, errRejectedCurriculumAttempt
		}
		beforeBytes, _ := before.CanonicalJSON()
		afterBytes, _ := result.Output.CanonicalJSON()
		token := fixtureToken(panelCommitment, seedCommitment, attempt, position)
		training.Cases = append(training.Cases, transformfixturecore.TrainingCase{Token: token, Kind: "positive", Before: beforeBytes, After: afterBytes})
	}
	negativeKinds := []string{"zero", "two", "noop", "wrong-context"}
	for position, spec := range trainingOrder {
		if spec < 4 {
			continue
		}
		negative := spec - 4
		before := streamedNegativeForest(structureSeed, aliasSeed, scalarSeed, childSeed, negative, false, latent, negativeKinds[negative])
		beforeBytes, _ := before.CanonicalJSON()
		training.Cases = append(training.Cases, transformfixturecore.TrainingCase{Token: fixtureToken(panelCommitment, seedCommitment, attempt, position), Kind: "abstain", Before: beforeBytes})
	}
	for position, spec := range heldoutOrder {
		if spec >= 4 {
			continue
		}
		before := streamedPositiveForest(structureSeed^0x9e3779b97f4a7c15, aliasSeed, scalarSeed, childSeed, spec, true)
		result, _ := latent.Apply(before)
		beforeBytes, _ := before.CanonicalJSON()
		output, _ := result.Output.CanonicalJSON()
		token := fixtureToken(panelCommitment, seedCommitment, attempt, 8+position)
		heldout.Cases = append(heldout.Cases, transformfixturecore.HeldoutCase{Token: token, Before: beforeBytes})
		expected = append(expected, expectedCase{token, "applied", output})
	}
	for position, spec := range heldoutOrder {
		if spec < 4 {
			continue
		}
		negative := spec - 4
		before := streamedNegativeForest(structureSeed^0x9e3779b97f4a7c15, aliasSeed, scalarSeed, childSeed, negative, true, latent, negativeKinds[negative])
		beforeBytes, _ := before.CanonicalJSON()
		token := fixtureToken(panelCommitment, seedCommitment, attempt, 8+position)
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
	ledger, err := generatorAcceptanceMatrix(training, heldout, expected, latentBytes)
	if err != nil || !ledger.Accepted {
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
	tieRNG := fixtureStream(panelCommitment, seedCommitment, attempt, "baseline-ties")
	for _, policy := range empiricalPolicies {
		policyTokens[policy] = fmt.Sprintf("%016x", queueRNG.Uint64())
		stream := tieRNG
		if policy == RandomPBE {
			stream = randomRNG
		}
		policyRandomness[policy] = [2]uint64{stream.Uint64(), stream.Uint64()}
	}
	return curriculum{Ordinal: ordinal, Family: family, Seed: seed, SeedCommitment: seedCommitment, PanelCommitment: panelCommitment, AcceptedAttempt: attempt, Training: trainingBytes, Heldout: heldoutBytes, Expected: expected, Latent: latentBytes, PolicyTokens: policyTokens, PolicyRandomness: policyRandomness, GeneratorLedger: ledger}, nil
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

func generatorAcceptanceMatrix(training transformfixturecore.Training, heldout transformfixturecore.Heldout, expected []expectedCase, latent []byte) (acceptanceLedger, error) {
	const generatorAcceptanceWork int64 = 109161
	truth := make(map[string]expectedCase, len(expected))
	for _, item := range expected {
		truth[item.Token] = item
	}
	best := int(^uint(0) >> 1)
	var winners [][]byte
	var matrixRows []any
	applications := 0
	latentHeldoutExact := false
	schemas := transformschema.Schemas()
	if len(schemas) != 72 || len(training.Cases) != 8 || len(heldout.Cases) != 8 || len(expected) != 8 {
		return acceptanceLedger{}, errors.New("generator acceptance matrix dimensions")
	}
	trainingCases := slices.Clone(training.Cases)
	heldoutCases := slices.Clone(heldout.Cases)
	slices.SortFunc(trainingCases, func(a, b transformfixturecore.TrainingCase) int { return strings.Compare(a.Token, b.Token) })
	slices.SortFunc(heldoutCases, func(a, b transformfixturecore.HeldoutCase) int { return strings.Compare(a.Token, b.Token) })
	for _, candidate := range schemas {
		trainingExact := true
		heldoutExact := true
		var outcomes []any
		for _, example := range trainingCases {
			f, err := transformschema.ParseForest(example.Before)
			if err != nil {
				return acceptanceLedger{}, err
			}
			r, err := candidate.Apply(f)
			if err != nil {
				return acceptanceLedger{}, err
			}
			applications++
			outputDigest := ""
			var output []byte
			if r.Output != nil {
				output, _ = r.Output.CanonicalJSON()
				outputDigest = digestBytes(output)
			}
			matches := false
			if example.Kind == "positive" {
				matches = r.Terminal == "applied" && bytes.Equal(output, example.After)
			} else {
				matches = len(r.Terminal) >= 8 && r.Terminal[:8] == "abstain/"
			}
			trainingExact = trainingExact && matches
			outcomes = append(outcomes, []any{example.Token, r.Terminal, outputDigest, matches})
		}
		for _, example := range heldoutCases {
			f, err := transformschema.ParseForest(example.Before)
			if err != nil {
				return acceptanceLedger{}, err
			}
			r, err := candidate.Apply(f)
			if err != nil {
				return acceptanceLedger{}, err
			}
			applications++
			outputDigest := ""
			var output []byte
			if r.Output != nil {
				output, _ = r.Output.CanonicalJSON()
				outputDigest = digestBytes(output)
			}
			want, ok := truth[example.Token]
			matches := ok && (want.Terminal == "applied" && r.Terminal == "applied" && bytes.Equal(output, want.Output) || want.Terminal == "abstain" && len(r.Terminal) >= 8 && r.Terminal[:8] == "abstain/")
			heldoutExact = heldoutExact && matches
			outcomes = append(outcomes, []any{example.Token, r.Terminal, outputDigest, matches})
		}
		encoded, _ := candidate.CanonicalJSON()
		matrixRows = append(matrixRows, []any{digestBytes(encoded), trainingExact, heldoutExact, outcomes})
		if bytes.Equal(encoded, latent) {
			latentHeldoutExact = heldoutExact
		}
		if trainingExact {
			cost := schemaDescription(candidate)
			if cost < best {
				best = cost
				winners = [][]byte{encoded}
			} else if cost == best {
				winners = append(winners, encoded)
			}
		}
	}
	if applications != 72*16 {
		return acceptanceLedger{}, errors.New("generator acceptance matrix application count")
	}
	matrix := mustJSON([]any{"transform-generator-acceptance-matrix/v1", matrixRows})
	accepted := len(winners) == 1 && slices.Equal(winners[0], latent) && latentHeldoutExact
	return acceptanceLedger{applications, generatorAcceptanceWork, digestBytes(matrix), accepted}, nil
}

func schemaDescription(s transformschema.Schema) int {
	cost := map[string]int{"request-target": 1, "from-value": 2, "first-local": 3, "definition": 1, "references": 1, "definition+references": 2, "local": 1, "global": 2, "equals-from": 2, "any": 1, "required": 2, "none": 1}
	return cost[s.Anchor] + cost[s.Targets] + cost[s.ReferenceScope] + cost[s.OldGuard] + cost[s.Locality]
}

func positiveForest(seed uint64, variant int, heldout bool) transformschema.Forest {
	return streamedPositiveForest(seed, seed, seed, seed, variant, heldout)
}

func streamedPositiveForest(structureSeed, aliasSeed, scalarSeed, childSeed uint64, variant int, heldout bool) transformschema.Forest {
	trainingWords := [][5]string{{"base", "old", "new", "other", "spare"}, {"start", "prior", "next", "aside", "extra"}, {"plain", "early", "later", "alien", "decoy"}, {"root", "before", "after", "remote", "unused"}}
	heldoutWords := [][5]string{{"alpha", "amber", "azure", "ashen", "apple"}, {"brisk", "brown", "blue", "beige", "birch"}, {"clear", "coral", "cyan", "cream", "cedar"}, {"dense", "dark", "dawn", "dull", "drift"}}
	words := trainingWords
	if heldout {
		words = heldoutWords
	}
	structuralVariant := (int(structureSeed%4) + variant) % 4
	w := words[(int(scalarSeed%uint64(len(words)))+variant)%len(words)]
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
		rotateStrings(keys, int(aliasSeed%uint64(len(keys))))
		for index := range f.Nodes {
			if f.Nodes[index].Kind != "group" {
				f.Nodes[index].Key = keys[index-1]
			}
		}
	} else {
		keys := []string{"decoy", "service", "change", "local", "aside", "remote", "far"}
		rotateStrings(keys, int(aliasSeed%uint64(len(keys))))
		key := 0
		for index := range f.Nodes {
			if f.Nodes[index].Kind != "group" {
				f.Nodes[index].Key = keys[key]
				key++
			}
		}
	}
	if structuralVariant == 0 {
		f.Nodes = f.Nodes[:6]
	}
	if structuralVariant == 3 {
		f.Nodes[1].Value = w[1]
	}
	// A combined/global/any schema must stay within the four-edit grammar while
	// the batch collectively distinguishes local/global and equals/any.
	if structuralVariant%2 == 0 {
		if len(f.Nodes) > 8 {
			f.Nodes[8].Target = 1
		}
	} else {
		f.Nodes[5].Target = 1
	}
	if structuralVariant == 2 {
		mapping := make([]int, len(f.Nodes))
		for index := range mapping {
			mapping[index] = index
		}
		rng := rand.New(rand.NewPCG(childSeed, childSeed^0xd1b54a32d192ed03))
		shuffleFixtureValues(rng, mapping)
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
	return streamedNegativeForest(seed, seed, seed, seed, variant, heldout, latent, kind)
}

func streamedNegativeForest(structureSeed, aliasSeed, scalarSeed, childSeed uint64, variant int, heldout bool, latent transformschema.Schema, kind string) transformschema.Forest {
	f := streamedPositiveForest(structureSeed, aliasSeed, scalarSeed, childSeed, variant, heldout)
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

func shuffleFixtureValues[T any](rng *rand.Rand, values []T) {
	for index := len(values) - 1; index > 0; index-- {
		other := int(rng.Uint64N(uint64(index + 1)))
		values[index], values[other] = values[other], values[index]
	}
}

func rotateStrings(values []string, offset int) {
	if len(values) == 0 || offset%len(values) == 0 {
		return
	}
	offset %= len(values)
	copy(values, append(slices.Clone(values[offset:]), values[:offset]...))
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
