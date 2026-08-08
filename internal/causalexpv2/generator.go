package causalexpv2

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/chazu/nous/internal/causaloracle"
	"github.com/chazu/nous/internal/causalv2"
)

type Cohort string

const (
	CohortCostSkewed  Cohort = "cost-skewed"
	CohortBalanced    Cohort = "balanced"
	CohortEquivalence Cohort = "equivalence"
	CohortIrrelevant  Cohort = "irrelevant"
)

type PublicFixture = causalv2.PublicFixture
type PrivateFixture = causalv2.PrivateFixture

// protectedGeneratorCalls is test-observable only inside this package. It
// proves authorization failures stop before any training/validation/locked
// generator access rather than merely failing after hidden fixtures open.
var protectedGeneratorCalls atomic.Int64

func cohortFor(index int) Cohort {
	switch index % 8 {
	case 0, 1, 2, 3:
		return CohortCostSkewed
	case 4, 5:
		return CohortBalanced
	case 6:
		return CohortEquivalence
	default:
		return CohortIrrelevant
	}
}

func stream(seed int64, panel Panel, label string, attempt int) *rand.Rand {
	sum := sha256.Sum256([]byte(fmt.Sprintf("active-causal-diagnosis/v2|%s|%d|%s|%d", panel, seed, label, attempt)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(sum[:8]), binary.BigEndian.Uint64(sum[8:16])))
}

func shuffle[T any](values []T, random *rand.Rand) {
	for i := len(values) - 1; i > 0; i-- {
		j := random.IntN(i + 1)
		values[i], values[j] = values[j], values[i]
	}
}

func modelUniverse() []string {
	models := causaloracle.Enumerate()
	codes := make([]string, len(models))
	for i, model := range models {
		codes[i], _ = causaloracle.Code(model)
	}
	return codes
}

func drawCosts(seed int64, panel Panel, attempt int, cohort Cohort) [3]int {
	random := stream(seed, panel, "cost-values", attempt)
	var values []int
	if cohort == CohortCostSkewed {
		// Normative v2 behavior: exactly these three draws; no unused balanced
		// values may be consumed first.
		values = []int{1 + random.IntN(10), 30 + random.IntN(21), 80 + random.IntN(21)}
	} else {
		values = []int{20 + random.IntN(21), 20 + random.IntN(21), 20 + random.IntN(21)}
	}
	positions := []int{0, 1, 2}
	shuffle(positions, stream(seed, panel, "cost-assignment", attempt))
	var costs [3]int
	for i, position := range positions {
		costs[position] = values[i]
	}
	return costs
}

func hasIrrelevant(posterior []string) bool {
	for variable := 0; variable < 3; variable++ {
		irrelevant := true
		for value := 0; value < 2; value++ {
			cells, err := causaloracle.Partition(posterior, fmt.Sprintf("do:%d=%d", variable, value))
			if err != nil || len(cells) != 1 {
				irrelevant = false
			}
		}
		if irrelevant {
			return true
		}
	}
	return false
}

func generate(panel Panel, seed int64, index int) (PrivateFixture, error) {
	if panel == PanelTraining || panel == PanelValidation || panel == PanelLocked {
		protectedGeneratorCalls.Add(1)
	}
	cohort := cohortFor(index)
	universe := modelUniverse()
	for attempt := 0; attempt <= 4095; attempt++ {
		pool := append([]string(nil), universe...)
		shuffle(pool, stream(seed, panel, "pool", attempt))
		pool = append([]string(nil), pool[:32]...)
		sort.Strings(pool)
		hidden := pool[stream(seed, panel, "hidden-member", attempt).IntN(len(pool))]
		passive, err := causaloracle.Observe(hidden)
		if err != nil {
			return PrivateFixture{}, err
		}
		posterior := make([]string, 0, len(pool))
		for _, code := range pool {
			outcome, observeErr := causaloracle.Observe(code)
			if observeErr != nil {
				return PrivateFixture{}, observeErr
			}
			if outcome == passive {
				posterior = append(posterior, code)
			}
		}
		if len(posterior) < 8 || len(posterior) > 32 {
			continue
		}
		if cohort == CohortEquivalence {
			signature, _ := causaloracle.Signature(hidden)
			count := 0
			for _, code := range posterior {
				candidate, _ := causaloracle.Signature(code)
				if candidate == signature {
					count++
				}
			}
			if count != 2 {
				continue
			}
		}
		if cohort == CohortIrrelevant && !hasIrrelevant(posterior) {
			continue
		}

		fixture := PublicFixture{Seed: seed, GeneratorAttempt: attempt, Cohort: string(cohort), PassiveOutcome: passive, Pool: pool, InitialPosterior: posterior}
		aliases := []string{"node-u", "node-v", "node-w"}
		shuffle(aliases, stream(seed, panel, "alias", attempt))
		fixture.Aliases = aliases
		fixture.Presentation = make([]int, 32)
		for i := range fixture.Presentation {
			fixture.Presentation[i] = i
		}
		shuffle(fixture.Presentation, stream(seed, panel, "list-order", attempt))
		costs := drawCosts(seed, panel, attempt, cohort)
		fixture.Costs = costs[:]
		randomActions := stream(seed, panel, "uniform-random", attempt)
		actions := causaloracle.Actions()
		for i := 0; i < 10; i++ {
			fixture.UniformRandomActions = append(fixture.UniformRandomActions, actions[randomActions.IntN(len(actions))].Code())
		}
		fixture.OpaqueToken, err = causalv2.PublicToken(string(panel), seed, attempt)
		if err != nil {
			return PrivateFixture{}, err
		}
		err = causalv2.SignPublicFixture(&fixture)
		if err != nil {
			return PrivateFixture{}, err
		}
		private := PrivateFixture{PublicFixture: fixture, HiddenHypothesis: hidden}
		err = causalv2.SignPrivateFixture(&private)
		return private, err
	}
	return PrivateFixture{}, fmt.Errorf("no %s fixture for seed %d", cohort, seed)
}

// GenerateDevelopment is the only repeatable seed generator. It rejects all
// seeds and indexes outside the preregistered development panel.
func (DiagnosticDevelopmentCapability) GenerateDevelopment(seed int64, index int) (PrivateFixture, error) {
	range_ := causalv2.PreregisteredManifest().DevelopmentSeeds
	if index < 0 || index >= range_.Count || seed != range_.Start+int64(index)*range_.Step {
		return PrivateFixture{}, errors.New("seed/index is outside the development panel")
	}
	return generate(PanelDevelopment, seed, index)
}

// TeacherRegistry is private experiment state. It deliberately exposes no
// method which returns, predicts with, or enumerates a hidden hypothesis.
type TeacherRegistry struct {
	mu      sync.RWMutex
	hidden  map[string]string
	claimed map[string]bool
}

func NewTeacherRegistry() *TeacherRegistry {
	return &TeacherRegistry{hidden: make(map[string]string), claimed: make(map[string]bool)}
}

func (registry *TeacherRegistry) Bind(fixture PrivateFixture) error {
	if registry == nil {
		return errors.New("nil teacher registry")
	}
	encoded, err := causalv2.CanonicalJSON(fixture)
	if err != nil {
		return err
	}
	if _, err := causalv2.VerifyPrivateFixture(encoded); err != nil {
		return err
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	token := fixture.PublicFixture.OpaqueToken
	if prior, exists := registry.hidden[token]; exists && prior != fixture.HiddenHypothesis {
		return errors.New("opaque token already has a different hidden binding")
	}
	registry.hidden[token] = fixture.HiddenHypothesis
	return nil
}

// Teacher returns a single-use private teacher through the narrow oracle
// interface. A token can be claimed only once from one registry.
func (registry *TeacherRegistry) Teacher(token string) (causaloracle.Teacher, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.claimed[token] {
		return nil, errors.New("opaque token already claimed")
	}
	hidden, ok := registry.hidden[token]
	if !ok {
		return nil, errors.New("unknown opaque token")
	}
	teacher, err := causaloracle.NewTeacher(token, hidden)
	if err != nil {
		return nil, err
	}
	registry.claimed[token] = true
	return teacher, nil
}

func VerifyPrivateFixture(fixture PrivateFixture) error {
	encoded, err := causalv2.CanonicalJSON(fixture)
	if err != nil {
		return err
	}
	_, err = causalv2.VerifyPrivateFixture(encoded)
	return err
}
