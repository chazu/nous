package causaldpproof

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

type exactValue struct {
	finite bool
	value  *big.Rat
	action string
}

type tinyDP struct {
	initial []string
	costs   [3]int
	corpus  frozenCorpus
	memo    map[string]exactValue
	counts  causalrun.Counts
}

type comparisonRecord struct {
	ComparisonPool int              `json:"comparison_pool"`
	Mask           uint8            `json:"mask"`
	Posterior      []string         `json:"posterior"`
	Finite         bool             `json:"finite"`
	Numerator      string           `json:"numerator"`
	Denominator    string           `json:"denominator"`
	Action         string           `json:"action"`
	Counts         causalrun.Counts `json:"counts"`
}

func runExactDP(corpus frozenCorpus) (ExactDPReport, error) {
	report := ExactDPReport{
		Models:         len(corpus.models),
		PassiveClasses: len(corpus.passiveClasses),
	}
	signatures := make(map[string]bool)
	for _, model := range corpus.models {
		signatures[strings.Join(model.Outcomes[:], "/")] = true
	}
	report.ObservationalClasses = len(signatures)
	if report.Models != wantModels || report.ObservationalClasses != wantObservationalClasses {
		return report, fmt.Errorf("frozen comparator corpus has models=%d classes=%d", report.Models, report.ObservationalClasses)
	}
	pools, err := comparisonPools(corpus)
	if err != nil {
		return report, err
	}
	report.ComparisonPools = len(pools)
	if report.ComparisonPools != wantObservationalClasses {
		return report, fmt.Errorf("comparison pools=%d, want %d", report.ComparisonPools, wantObservationalClasses)
	}

	var records []comparisonRecord
	coveredModels := make(map[string]bool)
	coveredClasses := make(map[string]bool)
	for classIndex, pool := range pools {
		initial := pool.initial
		if len(initial) < 8 || len(initial) > causal.MaximumPool {
			return report, fmt.Errorf("comparison pool %d size=%d is outside production DP bounds", classIndex, len(initial))
		}
		fullMask := uint8((1 << len(causal.Actions())) - 1)
		if !containsPosterior(inducedGroups(corpus, initial, fullMask), pool.target) {
			return report, fmt.Errorf("comparison pool %d does not induce its target observational class", classIndex)
		}
		for _, code := range pool.target {
			coveredModels[code] = true
		}
		coveredClasses[pool.signature] = true
		for mask := uint8(0); mask < 1<<len(causal.Actions()); mask++ {
			report.ActionSubsetGroups++
			consumed := consumedActions(mask)
			for _, posterior := range inducedGroups(corpus, initial, mask) {
				production, err := causalrun.EvaluateDynamicProof(initial, proofCosts, posterior, consumed)
				if err != nil {
					return report, fmt.Errorf("production class=%d mask=%02x posterior=%d: %w", classIndex, mask, len(posterior), err)
				}
				independent := newTinyDP(initial, proofCosts, corpus)
				oracleValue := independent.value(posterior, mask)
				oracle := proofEvaluation(oracleValue, independent.counts)
				if err := compareEvaluations(production, oracle); err != nil {
					return report, fmt.Errorf("DP mismatch class=%d mask=%02x posterior=%v: %w", classIndex, mask, posterior, err)
				}
				report.Comparisons++
				if independent.terminal(posterior) {
					report.TerminalComparisons++
				} else if oracle.Finite {
					report.FiniteActionComparisons++
				} else {
					report.NonfiniteComparisons++
				}
				addCounts(&report.ProductionCounts, production.Counts)
				addCounts(&report.IndependentCounts, oracle.Counts)
				records = append(records, comparisonRecord{
					ComparisonPool: classIndex,
					Mask:           mask,
					Posterior:      append([]string(nil), posterior...),
					Finite:         oracle.Finite,
					Numerator:      oracle.Numerator,
					Denominator:    oracle.Denominator,
					Action:         oracle.Action,
					Counts:         oracle.Counts,
				})
			}
		}
	}
	if len(coveredModels) != wantModels {
		return report, fmt.Errorf("tiny DP covered %d models, want %d", len(coveredModels), wantModels)
	}
	if len(coveredClasses) != wantObservationalClasses {
		return report, fmt.Errorf("tiny DP covered %d observational classes, want %d", len(coveredClasses), wantObservationalClasses)
	}
	if report.ProductionCounts != report.IndependentCounts {
		return report, fmt.Errorf("aggregate DP counts differ: production=%+v independent=%+v", report.ProductionCounts, report.IndependentCounts)
	}
	report.ComparisonDigest, err = causalv2.Digest(comparisonDigestDomain, records)
	if err != nil {
		return report, err
	}
	return report, nil
}

func containsPosterior(groups [][]string, target []string) bool {
	for _, group := range groups {
		if len(group) != len(target) {
			continue
		}
		match := true
		for index := range group {
			if group[index] != target[index] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

type comparisonPool struct {
	signature string
	target    []string
	initial   []string
}

// comparisonPools creates one minimum-size legal production pool for each
// complete observational-equivalence class. Lexical padding is public and
// deterministic; the target classes jointly cover the full 72-model corpus.
func comparisonPools(corpus frozenCorpus) ([]comparisonPool, error) {
	bySignature := make(map[string][]string)
	var allCodes []string
	for _, model := range corpus.models {
		signature := strings.Join(model.Outcomes[:], "/")
		bySignature[signature] = append(bySignature[signature], model.Code)
		allCodes = append(allCodes, model.Code)
	}
	sort.Strings(allCodes)
	var pools []comparisonPool
	for _, signature := range sortedKeys(bySignature) {
		target := append([]string(nil), bySignature[signature]...)
		sort.Strings(target)
		if len(target) > causal.MaximumPool {
			return nil, fmt.Errorf("observational class size=%d exceeds maximum pool", len(target))
		}
		initial := append([]string(nil), target...)
		present := make(map[string]bool, len(initial))
		for _, code := range initial {
			present[code] = true
		}
		for _, code := range allCodes {
			if len(initial) >= 8 {
				break
			}
			if !present[code] {
				initial = append(initial, code)
				present[code] = true
			}
		}
		sort.Strings(initial)
		pools = append(pools, comparisonPool{signature: signature, target: target, initial: initial})
	}
	return pools, nil
}

func newTinyDP(initial []string, costs [3]int, corpus frozenCorpus) *tinyDP {
	return &tinyDP{
		initial: append([]string(nil), initial...),
		costs:   costs,
		corpus:  corpus,
		memo:    make(map[string]exactValue),
	}
}

func (d *tinyDP) value(posterior []string, mask uint8) exactValue {
	d.chargeMemoLookup()
	key := tinyStateKey(posterior, mask)
	if cached, ok := d.memo[key]; ok {
		return cached
	}
	d.chargeMemoState()
	if d.terminal(posterior) {
		terminal := exactValue{finite: true, value: new(big.Rat)}
		d.memo[key] = terminal
		return terminal
	}

	var best exactValue
	for actionIndex, action := range causal.Actions() {
		if mask&(1<<actionIndex) != 0 {
			continue
		}
		d.chargeQ()
		cells := d.partition(posterior, actionIndex)
		if len(cells) <= 1 {
			continue
		}
		q := new(big.Rat).SetInt64(int64(d.costs[action.Variable]))
		finite := true
		for _, cell := range cells {
			d.chargeTable()
			child := d.value(cell, mask|(1<<actionIndex))
			if !child.finite {
				finite = false
				break
			}
			weight := new(big.Rat).SetFrac64(int64(len(cell)), int64(len(posterior)))
			q.Add(q, new(big.Rat).Mul(weight, child.value))
		}
		code := action.Code()
		if finite && (!best.finite || q.Cmp(best.value) < 0 || (q.Cmp(best.value) == 0 && code < best.action)) {
			best = exactValue{finite: true, value: new(big.Rat).Set(q), action: code}
		}
	}
	d.memo[key] = best
	return best
}

func (d *tinyDP) terminal(posterior []string) bool {
	if len(posterior) == 1 {
		return true
	}
	want := d.signature(posterior[0])
	var complete []string
	for _, code := range d.initial {
		if d.signature(code) == want {
			complete = append(complete, code)
		}
	}
	if len(complete) != len(posterior) {
		return false
	}
	for index := range complete {
		if complete[index] != posterior[index] {
			return false
		}
	}
	return true
}

func (d *tinyDP) signature(code string) string {
	model := d.corpus.byCode[code]
	return strings.Join(model.Outcomes[:], "/")
}

func (d *tinyDP) partition(posterior []string, actionIndex int) [][]string {
	byOutcome := make(map[string][]string)
	for _, code := range posterior {
		outcome := d.corpus.byCode[code].Outcomes[actionIndex]
		byOutcome[outcome] = append(byOutcome[outcome], code)
	}
	var cells [][]string
	for value := 0; value < 8; value++ {
		outcome := fmt.Sprintf("%03b", value)
		if cell := byOutcome[outcome]; len(cell) != 0 {
			cells = append(cells, append([]string(nil), cell...))
		}
	}
	return cells
}

func (d *tinyDP) chargeMemoState() {
	d.counts.MemoStates++
	d.counts.TotalWork++
}

func (d *tinyDP) chargeMemoLookup() {
	d.counts.MemoLookups++
	d.counts.TotalWork++
}

func (d *tinyDP) chargeQ() {
	d.counts.QEvaluations++
	d.counts.TotalWork++
}

func (d *tinyDP) chargeTable() {
	d.counts.TableLookups++
	d.counts.TotalWork++
}

func tinyStateKey(posterior []string, mask uint8) string {
	canonical := append([]string(nil), posterior...)
	sort.Strings(canonical)
	return fmt.Sprintf("%02x|%s", mask, strings.Join(canonical, "\x00"))
}

func consumedActions(mask uint8) []string {
	var consumed []string
	for index, action := range causal.Actions() {
		if mask&(1<<index) != 0 {
			consumed = append(consumed, action.Code())
		}
	}
	return consumed
}

func proofEvaluation(value exactValue, counts causalrun.Counts) causalrun.DynamicProofEvaluation {
	result := causalrun.DynamicProofEvaluation{Finite: value.finite, Action: value.action, Counts: counts}
	if value.finite {
		result.Numerator = value.value.Num().String()
		result.Denominator = value.value.Denom().String()
	}
	return result
}

func compareEvaluations(production, independent causalrun.DynamicProofEvaluation) error {
	if production.Finite != independent.Finite ||
		production.Numerator != independent.Numerator ||
		production.Denominator != independent.Denominator ||
		production.Action != independent.Action {
		return fmt.Errorf("value production=%+v independent=%+v", production, independent)
	}
	if production.Counts != independent.Counts {
		return fmt.Errorf("counts production=%+v independent=%+v", production.Counts, independent.Counts)
	}
	if err := production.Counts.ValidateEquation(); err != nil {
		return err
	}
	return nil
}

func addCounts(total *causalrun.Counts, value causalrun.Counts) {
	total.SCMEvaluations += value.SCMEvaluations
	total.PartitionAssignments += value.PartitionAssignments
	total.CellAccumulations += value.CellAccumulations
	total.RuleComparisons += value.RuleComparisons
	total.PosteriorChecks += value.PosteriorChecks
	total.ArtifactMaterializations += value.ArtifactMaterializations
	total.TranscriptFields += value.TranscriptFields
	total.ProfileFields += value.ProfileFields
	total.MemoStates += value.MemoStates
	total.MemoLookups += value.MemoLookups
	total.QEvaluations += value.QEvaluations
	total.TableLookups += value.TableLookups
	total.EngineCycles += value.EngineCycles
	total.AttributedUnits += value.AttributedUnits
	total.TotalWork += value.TotalWork
}
