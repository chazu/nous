package causalexp

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/chazu/nous/internal/causaloracle"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

type Cohort string

const (
	CostSkewed  Cohort = "cost-skewed"
	Balanced    Cohort = "balanced"
	Equivalence Cohort = "equivalence"
	Irrelevant  Cohort = "irrelevant"
)

type Fixture struct {
	Seed             int64     `json:"seed"`
	GeneratorAttempt int       `json:"generator_attempt"`
	Cohort           Cohort    `json:"cohort"`
	Aliases          [3]string `json:"aliases"`
	Costs            [3]int    `json:"costs"`
	PassiveOutcome   string    `json:"passive_outcome"`
	Pool             []string  `json:"pool"`
	Presentation     []string  `json:"-"`
	InitialPosterior []string  `json:"initial_posterior"`
	HiddenHypothesis string    `json:"hidden_hypothesis"`
	FixtureDigest    string    `json:"fixture_digest"`
	Token            string    `json:"-"`
	RandomActions    []string  `json:"-"`
}

func cohortFor(index int) Cohort {
	switch index % 8 {
	case 0, 1, 2, 3:
		return CostSkewed
	case 4, 5:
		return Balanced
	case 6:
		return Equivalence
	default:
		return Irrelevant
	}
}
func stream(seed int64, panel, label string, attempt int) *rand.Rand {
	sum := sha256.Sum256([]byte(fmt.Sprintf("active-causal-diagnosis/v1|%s|%d|%s|%d", panel, seed, label, attempt)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(sum[:8]), binary.BigEndian.Uint64(sum[8:16])))
}
func shuffle[T any](values []T, r *rand.Rand) {
	for i := len(values) - 1; i > 0; i-- {
		j := r.IntN(i + 1)
		values[i], values[j] = values[j], values[i]
	}
}
func codesUniverse() []string {
	models := causaloracle.Enumerate()
	out := make([]string, len(models))
	for i, m := range models {
		out[i], _ = causaloracle.Code(m)
	}
	return out
}
func publicFixture(f Fixture) any {
	return struct {
		Seed             int64     `json:"seed"`
		GeneratorAttempt int       `json:"generator_attempt"`
		Cohort           Cohort    `json:"cohort"`
		Aliases          [3]string `json:"aliases"`
		Costs            [3]int    `json:"costs"`
		PassiveOutcome   string    `json:"passive_outcome"`
		Pool             []string  `json:"pool"`
		InitialPosterior []string  `json:"initial_posterior"`
		FixtureDigest    string    `json:"fixture_digest"`
	}{f.Seed, f.GeneratorAttempt, f.Cohort, f.Aliases, f.Costs, f.PassiveOutcome, f.Pool, f.InitialPosterior, ""}
}
func Generate(panel string, seed int64, index int) (Fixture, error) {
	cohort := cohortFor(index)
	universe := codesUniverse()
	for attempt := 0; attempt <= 4095; attempt++ {
		pool := append([]string(nil), universe...)
		shuffle(pool, stream(seed, panel, "pool", attempt))
		pool = append([]string(nil), pool[:32]...)
		sort.Strings(pool)
		hidden := pool[stream(seed, panel, "hidden-member", attempt).IntN(len(pool))]
		passive, _ := causaloracle.Observe(hidden)
		var posterior []string
		for _, code := range pool {
			o, _ := causaloracle.Observe(code)
			if o == passive {
				posterior = append(posterior, code)
			}
		}
		if len(posterior) < 8 || len(posterior) > 32 {
			continue
		}
		if cohort == Equivalence {
			sig, _ := causaloracle.Signature(hidden)
			count := 0
			for _, code := range posterior {
				s, _ := causaloracle.Signature(code)
				if s == sig {
					count++
				}
			}
			if count != 2 {
				continue
			}
		}
		if cohort == Irrelevant && !hasIrrelevant(posterior) {
			continue
		}
		f := Fixture{Seed: seed, GeneratorAttempt: attempt, Cohort: cohort, Pool: pool, InitialPosterior: posterior, HiddenHypothesis: hidden, PassiveOutcome: passive}
		aliases := []string{"node-u", "node-v", "node-w"}
		shuffle(aliases, stream(seed, panel, "alias", attempt))
		copy(f.Aliases[:], aliases)
		presentation := append([]string(nil), pool...)
		shuffle(presentation, stream(seed, panel, "list-order", attempt))
		f.Presentation = presentation
		f.Costs = drawCosts(seed, panel, attempt, cohort)
		random := stream(seed, panel, "uniform-random", attempt)
		for i := 0; i < 10; i++ {
			f.RandomActions = append(f.RandomActions, causal.Actions()[random.IntN(6)].Code())
		}
		digest, _ := causal.Digest("causal-public-fixture/v1", publicFixture(f))
		f.FixtureDigest = digest
		token, _ := causal.Digest("causal-private-token/v1", struct {
			Seed    int64
			Attempt int
			Hidden  string
		}{seed, attempt, hidden})
		f.Token = token
		return f, nil
	}
	return Fixture{}, fmt.Errorf("no %s fixture for seed %d", cohort, seed)
}
func hasIrrelevant(p []string) bool {
	for variable := 0; variable < 3; variable++ {
		irrelevant := true
		for value := 0; value < 2; value++ {
			cells, _ := causaloracle.Partition(p, fmt.Sprintf("do:%d=%d", variable, value))
			if len(cells) != 1 {
				irrelevant = false
			}
		}
		if irrelevant {
			return true
		}
	}
	return false
}
func drawCosts(seed int64, panel string, attempt int, cohort Cohort) [3]int {
	r := stream(seed, panel, "cost-values", attempt)
	values := []int{20 + r.IntN(21), 20 + r.IntN(21), 20 + r.IntN(21)}
	if cohort == CostSkewed {
		values = []int{1 + r.IntN(10), 30 + r.IntN(21), 80 + r.IntN(21)}
	}
	positions := []int{0, 1, 2}
	shuffle(positions, stream(seed, panel, "cost-assignment", attempt))
	var costs [3]int
	for i, p := range positions {
		costs[p] = values[i]
	}
	return costs
}
