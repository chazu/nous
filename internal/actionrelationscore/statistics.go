package actionrelationscore

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"math/bits"
	"slices"
	"sort"

	"github.com/chazu/nous/internal/actionrelationsearch"
)

const (
	InferenceReplicates  = 10_000
	PowerOuterReplicates = 2_000
	PowerInnerReplicates = 2_000
)

type Fraction struct {
	Numerator   int
	Denominator int
}

func (f Fraction) Wire() []int { return []int{f.Numerator, f.Denominator} }

type AmortizationRow struct {
	Panel         string
	Curriculum    int
	Family        int
	Acquisition   int
	DynamicSearch int
	NousSearch    int
	Batches       int
	Infinite      bool
	Status        string
}

func (r AmortizationRow) wire() []any {
	batches := any(r.Batches)
	if r.Infinite {
		batches = "infinite"
	}
	return []any{"actionrelation-amortization/v1", r.Panel, r.Curriculum, r.Family, r.Acquisition, r.NousSearch, r.DynamicSearch, r.DynamicSearch - r.NousSearch, batches, r.Status}
}

type Inference struct {
	PrimarySearchRatio   Fraction
	LifecycleRatio       Fraction
	AmortizationRows     []AmortizationRow
	ConfidenceInterval   [2]Fraction
	RandomizationP       Fraction
	RandomizationExtreme int
	SavingCoverage       Fraction
	Power                Fraction
	PowerSuccesses       int
}

type pairedCurriculum struct {
	curriculum int
	family     int
	nous       CurriculumPolicyRow
	dynamic    CurriculumPolicyRow
	mechanical bool
}

func Infer(panel, authority string, worldRows []WorldPolicyRow, rows []CurriculumPolicyRow) (Inference, error) {
	pairs, err := pairedRows(panel, worldRows, rows)
	if err != nil {
		return Inference{}, err
	}
	result, err := inferPairs(panel, authority, pairs, InferenceReplicates, 249, 9749, "bootstrap-family-row", "randomization-swap", nil)
	if err != nil {
		return Inference{}, err
	}
	result.Power = Fraction{0, 1}
	if panel == "development" {
		power, successes, err := estimatePower(authority, pairs, PowerOuterReplicates, PowerInnerReplicates)
		if err != nil {
			return Inference{}, err
		}
		result.Power, result.PowerSuccesses = power, successes
	}
	return result, nil
}

func pairedRows(panel string, worldRows []WorldPolicyRow, rows []CurriculumPolicyRow) ([]pairedCurriculum, error) {
	want := map[string]int{"development": 16, "validation": 24, "locked": 32}[panel]
	if want == 0 || len(rows) != want*len(Policies) || len(worldRows) != want*6*len(Policies) {
		return nil, fmt.Errorf("%s inference requires %d curriculum-policy rows", panel, want*len(Policies))
	}
	mechanical := make([]bool, want)
	for index := range mechanical {
		mechanical[index] = true
	}
	seenWorld := map[[3]int]bool{}
	for _, row := range worldRows {
		key := [3]int{row.Curriculum, row.WorldOrdinal, policyIndex(row.Policy)}
		if VerifyWorldPolicyRow(row) != nil || row.Panel != panel || row.Curriculum < 0 || row.Curriculum >= want || row.Family != row.Curriculum%8 || key[2] < 0 || seenWorld[key] {
			return nil, fmt.Errorf("invalid or duplicate inference world row")
		}
		seenWorld[key] = true
		mechanical[row.Curriculum] = mechanical[row.Curriculum] && row.SearchTerminal == "completed" && row.BehaviorEqual
		if row.Policy == actionrelationsearch.NousSleep && row.MatchCounts.UtilityFalseMatches != 0 {
			mechanical[row.Curriculum] = false
		}
	}
	byCurriculum := make([]map[actionrelationsearch.Policy]CurriculumPolicyRow, want)
	for index := range byCurriculum {
		byCurriculum[index] = map[actionrelationsearch.Policy]CurriculumPolicyRow{}
	}
	for _, row := range rows {
		if VerifyCurriculumPolicyRow(row) != nil || row.Panel != panel || row.Curriculum < 0 || row.Curriculum >= want || row.Family != row.Curriculum%8 || byCurriculum[row.Curriculum][row.Policy].Digest != "" {
			return nil, fmt.Errorf("invalid or duplicate inference row")
		}
		byCurriculum[row.Curriculum][row.Policy] = row
	}
	pairs := make([]pairedCurriculum, want)
	for curriculum, policies := range byCurriculum {
		if len(policies) != len(Policies) {
			return nil, fmt.Errorf("curriculum %d lacks a policy", curriculum)
		}
		nous, dynamic := policies[actionrelationsearch.NousSleep], policies[actionrelationsearch.DynamicSleep]
		if nous.AggregateTerminal != "completed" || dynamic.AggregateTerminal != "completed" || !nous.BehaviorEqual || !dynamic.BehaviorEqual || dynamic.SearchTotal <= 0 {
			return nil, fmt.Errorf("curriculum %d primary policies did not mechanically complete", curriculum)
		}
		pairs[curriculum] = pairedCurriculum{curriculum: curriculum, family: curriculum % 8, nous: nous, dynamic: dynamic, mechanical: mechanical[curriculum]}
	}
	return pairs, nil
}

func inferPairs(panel, authority string, pairs []pairedCurriculum, replicates, lowerIndex, upperIndex int, bootstrapNamespace, randomizationNamespace string, prefix []int) (Inference, error) {
	if replicates < 1 || lowerIndex < 0 || upperIndex < lowerIndex || upperIndex >= replicates || len(pairs) == 0 {
		return Inference{}, fmt.Errorf("invalid inference schedule")
	}
	result := Inference{AmortizationRows: make([]AmortizationRow, len(pairs))}
	nousSearch, dynamicSearch, nousLifecycle, dynamicLifecycle, savings := 0, 0, 0, 0, 0
	byFamily := make([][]pairedCurriculum, 8)
	for index, pair := range pairs {
		if pair.family < 0 || pair.family > 7 {
			return Inference{}, fmt.Errorf("invalid pair family")
		}
		byFamily[pair.family] = append(byFamily[pair.family], pair)
		nousSearch += pair.nous.SearchTotal
		dynamicSearch += pair.dynamic.SearchTotal
		nousLifecycle += pair.nous.LifecycleTotal
		dynamicLifecycle += pair.dynamic.LifecycleTotal
		difference := pair.dynamic.SearchTotal - pair.nous.SearchTotal
		row := AmortizationRow{Panel: panel, Curriculum: pair.curriculum, Family: pair.family, Acquisition: sum(pair.nous.AcquisitionWorkVector), DynamicSearch: pair.dynamic.SearchTotal, NousSearch: pair.nous.SearchTotal, Status: "complete"}
		if difference <= 0 {
			row.Infinite = true
		} else {
			row.Batches = (row.Acquisition + difference - 1) / difference
			savings++
		}
		result.AmortizationRows[index] = row
	}
	if dynamicSearch <= 0 || dynamicLifecycle <= 0 {
		return Inference{}, fmt.Errorf("zero inference denominator")
	}
	result.PrimarySearchRatio = Fraction{nousSearch, dynamicSearch}
	result.LifecycleRatio = Fraction{nousLifecycle, dynamicLifecycle}
	result.SavingCoverage = Fraction{savings, len(pairs)}

	type bootstrapValue struct {
		fraction  Fraction
		replicate int
	}
	bootstrap := make([]bootstrapValue, replicates)
	for replicate := 0; replicate < replicates; replicate++ {
		numerator, denominator := 0, 0
		for family, members := range byFamily {
			if len(members) == 0 {
				return Inference{}, fmt.Errorf("empty bootstrap family %d", family)
			}
			for slot := range members {
				fields := append(slices.Clone(prefix), replicate, family, slot)
				selected := members[statPick(statDraw(panel, authority, bootstrapNamespace, fields), len(members))]
				numerator += selected.nous.SearchTotal
				denominator += selected.dynamic.SearchTotal
			}
		}
		bootstrap[replicate] = bootstrapValue{Fraction{numerator, denominator}, replicate}
	}
	sort.Slice(bootstrap, func(i, j int) bool {
		if comparison := compareFraction(bootstrap[i].fraction, bootstrap[j].fraction); comparison != 0 {
			return comparison < 0
		}
		return bootstrap[i].replicate < bootstrap[j].replicate
	})
	result.ConfidenceInterval = [2]Fraction{bootstrap[lowerIndex].fraction, bootstrap[upperIndex].fraction}

	observed := absBig(big.NewInt(int64(nousSearch - dynamicSearch)))
	extreme := 0
	for replicate := 0; replicate < replicates; replicate++ {
		randomized := new(big.Int)
		for curriculum, pair := range pairs {
			fields := append(slices.Clone(prefix), replicate, curriculum)
			difference := big.NewInt(int64(pair.nous.SearchTotal - pair.dynamic.SearchTotal))
			if statDraw(panel, authority, randomizationNamespace, fields)&1 != 0 {
				difference.Neg(difference)
			}
			randomized.Add(randomized, difference)
		}
		if absBig(randomized).Cmp(observed) >= 0 {
			extreme++
		}
	}
	result.RandomizationExtreme = extreme
	result.RandomizationP = Fraction{1 + extreme, 1 + replicates}
	return result, nil
}

func estimatePower(authority string, development []pairedCurriculum, outerReplicates, innerReplicates int) (Fraction, int, error) {
	if len(development) != 16 || outerReplicates < 1 || innerReplicates < 1 {
		return Fraction{}, 0, fmt.Errorf("power requires sixteen development curricula")
	}
	byFamily := make([][]pairedCurriculum, 8)
	for _, pair := range development {
		byFamily[pair.family] = append(byFamily[pair.family], pair)
	}
	successes := 0
	for outer := 0; outer < outerReplicates; outer++ {
		synthetic := make([]pairedCurriculum, 0, 32)
		for slot := 0; slot < 4; slot++ {
			for family := 0; family < 8; family++ {
				members := byFamily[family]
				if len(members) != 2 {
					return Fraction{}, 0, fmt.Errorf("development family %d does not have two rows", family)
				}
				selected := members[statPick(statDraw("development", authority, "power-outer-family-row", []int{outer, family, slot}), 2)]
				selected.curriculum = slot*8 + family
				synthetic = append(synthetic, selected)
			}
		}
		inference, err := inferPairs("development", authority, synthetic, innerReplicates, innerReplicates*25/1000-1, innerReplicates*975/1000-1, "power-inner-bootstrap-row", "power-inner-randomization-swap", []int{outer})
		if err != nil {
			return Fraction{}, 0, err
		}
		if lockedPositive(inference, synthetic) {
			successes++
		}
	}
	return Fraction{successes, outerReplicates}, successes, nil
}

func lockedPositive(inference Inference, pairs []pairedCurriculum) bool {
	for _, pair := range pairs {
		if !pair.mechanical {
			return false
		}
	}
	return compareFraction(inference.PrimarySearchRatio, Fraction{85, 100}) <= 0 && compareFraction(inference.ConfidenceInterval[1], Fraction{1, 1}) < 0 && compareFraction(inference.SavingCoverage, Fraction{80, 100}) >= 0 && compareFraction(inference.RandomizationP, Fraction{5, 100}) < 0
}

func statDraw(panel, authority, namespace string, fields []int) uint64 {
	wire, _ := json.Marshal([]any{"actionrelation-stat-draw/v1", panel, authority, namespace, fields})
	value := sha256.Sum256(wire)
	return binary.BigEndian.Uint64(value[:8])
}

func statPick(draw uint64, n int) int {
	high, _ := bits.Mul64(draw, uint64(n))
	return int(high)
}

func compareFraction(left, right Fraction) int {
	a := new(big.Int).Mul(big.NewInt(int64(left.Numerator)), big.NewInt(int64(right.Denominator)))
	b := new(big.Int).Mul(big.NewInt(int64(right.Numerator)), big.NewInt(int64(left.Denominator)))
	return a.Cmp(b)
}

func absBig(value *big.Int) *big.Int {
	if value.Sign() < 0 {
		return new(big.Int).Neg(value)
	}
	return new(big.Int).Set(value)
}
