// Package causaldpproof implements deterministic, development-only proofs for
// the active-causal-diagnosis dynamic-programming oracle. It has no panel,
// authorization, teacher, or evidence-publication capability.
package causaldpproof

import (
	"errors"
	"fmt"
	"math/bits"
	"sort"
	"strings"

	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

const (
	ReportVersion             = "causal-dp-proof/v1"
	comparisonDigestDomain    = "causal-dp-proof-comparisons/v1"
	reportDigestDomain        = "causal-dp-proof-report/v1"
	corpusDigestDomain        = "causal-dp-proof-corpus/v1"
	wantModels                = 72
	wantObservationalClasses  = 58
	wantReachableStateBound   = 1873
	wantTableConstructionWork = 192919
	wantTrajectoryLookupWork  = 198
	wantTotalWorkBound        = 193117
)

var proofCosts = [3]int{1, 2, 3}

// CombinatorialReport records the cheap exhaustive proof over legal passive
// subsets. It deliberately does not invoke either DP implementation.
type CombinatorialReport struct {
	PassiveClasses             int    `json:"passive_classes"`
	PassiveClassSizes          []int  `json:"passive_class_sizes"`
	LegalSubsets               int    `json:"legal_subsets"`
	ActionPartitions           int    `json:"action_partitions"`
	ActionSubsetChecks         int    `json:"action_subset_checks"`
	InducedPosteriorCells      int    `json:"induced_posterior_cells"`
	MaximumPoolSize            int    `json:"maximum_pool_size"`
	MaximumCellsPerAction      int    `json:"maximum_cells_per_action"`
	DepthStateBounds           [7]int `json:"depth_state_bounds"`
	ReachableStateBound        int    `json:"reachable_state_bound"`
	TableConstructionWorkBound int    `json:"table_construction_work_bound"`
	TrajectoryLookupWorkBound  int    `json:"trajectory_lookup_work_bound"`
	TotalWorkBound             int    `json:"total_work_bound"`
}

// ExactDPReport records exhaustive differential checks over every public state
// induced by every action subset in the frozen comparison pools.
type ExactDPReport struct {
	Models                  int              `json:"models"`
	PassiveClasses          int              `json:"passive_classes"`
	ObservationalClasses    int              `json:"observational_classes"`
	ComparisonPools         int              `json:"comparison_pools"`
	ActionSubsetGroups      int              `json:"action_subset_groups"`
	Comparisons             int              `json:"comparisons"`
	TerminalComparisons     int              `json:"terminal_comparisons"`
	FiniteActionComparisons int              `json:"finite_action_comparisons"`
	NonfiniteComparisons    int              `json:"nonfinite_comparisons"`
	ProductionCounts        causalrun.Counts `json:"production_counts"`
	IndependentCounts       causalrun.Counts `json:"independent_counts"`
	ComparisonDigest        string           `json:"comparison_digest"`
}

// Report is the deterministic result of both mandated DP proofs.
type Report struct {
	Version       string              `json:"version"`
	CorpusDigest  string              `json:"corpus_digest"`
	Costs         [3]int              `json:"costs"`
	Combinatorial CombinatorialReport `json:"combinatorial"`
	ExactDP       ExactDPReport       `json:"exact_dp"`
	ReportDigest  string              `json:"report_digest"`
}

type corpusModel struct {
	Code     string    `json:"code"`
	Passive  string    `json:"passive"`
	Outcomes [6]string `json:"outcomes"`
}

type frozenCorpus struct {
	models         []corpusModel
	byCode         map[string]corpusModel
	passiveClasses [][]string
}

// Run executes the cheap combinatorial proof and the independent tiny-DP
// differential proof. It never reads protected fixtures or panel artifacts.
func Run() (Report, error) {
	corpus, corpusDigest, err := buildFrozenCorpus()
	if err != nil {
		return Report{}, err
	}
	combinatorial, err := runCombinatorial(corpus)
	if err != nil {
		return Report{}, err
	}
	exact, err := runExactDP(corpus)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		Version:       ReportVersion,
		CorpusDigest:  corpusDigest,
		Costs:         proofCosts,
		Combinatorial: combinatorial,
		ExactDP:       exact,
	}
	report.ReportDigest, err = causalv2.Digest(reportDigestDomain, report)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func buildFrozenCorpus() (frozenCorpus, string, error) {
	actions := causal.Actions()
	hypotheses := causal.Enumerate()
	corpus := frozenCorpus{byCode: make(map[string]corpusModel, len(hypotheses))}
	byPassive := make(map[string][]string)
	observational := make(map[string]bool)
	for _, hypothesis := range hypotheses {
		code, err := causal.Code(hypothesis)
		if err != nil {
			return frozenCorpus{}, "", err
		}
		passive, err := causal.Evaluate(hypothesis, nil)
		if err != nil {
			return frozenCorpus{}, "", err
		}
		model := corpusModel{Code: code, Passive: causal.OutcomeCode(passive)}
		for index, action := range actions {
			outcome, evaluateErr := causal.Evaluate(hypothesis, &action)
			if evaluateErr != nil {
				return frozenCorpus{}, "", evaluateErr
			}
			model.Outcomes[index] = causal.OutcomeCode(outcome)
		}
		corpus.models = append(corpus.models, model)
		corpus.byCode[code] = model
		byPassive[model.Passive] = append(byPassive[model.Passive], code)
		observational[strings.Join(model.Outcomes[:], "/")] = true
	}
	if len(corpus.models) != wantModels {
		return frozenCorpus{}, "", fmt.Errorf("frozen corpus models=%d, want %d", len(corpus.models), wantModels)
	}
	if len(observational) != wantObservationalClasses {
		return frozenCorpus{}, "", fmt.Errorf("observational classes=%d, want %d", len(observational), wantObservationalClasses)
	}
	passiveKeys := sortedKeys(byPassive)
	for _, key := range passiveKeys {
		class := append([]string(nil), byPassive[key]...)
		sort.Strings(class)
		corpus.passiveClasses = append(corpus.passiveClasses, class)
	}
	digest, err := causalv2.Digest(corpusDigestDomain, corpus.models)
	if err != nil {
		return frozenCorpus{}, "", err
	}
	return corpus, digest, nil
}

func runCombinatorial(corpus frozenCorpus) (CombinatorialReport, error) {
	report := CombinatorialReport{PassiveClasses: len(corpus.passiveClasses)}
	for _, class := range corpus.passiveClasses {
		report.PassiveClassSizes = append(report.PassiveClassSizes, len(class))
		maximum := len(class)
		if maximum > causal.MaximumPool {
			maximum = causal.MaximumPool
		}
		for size := 8; size <= maximum; size++ {
			err := enumerateSubsets(class, size, func(subset []string) error {
				report.LegalSubsets++
				if len(subset) > report.MaximumPoolSize {
					report.MaximumPoolSize = len(subset)
				}
				if len(subset) > causal.MaximumPool {
					return fmt.Errorf("enumerated pool size=%d exceeds %d", len(subset), causal.MaximumPool)
				}
				for _, action := range causal.Actions() {
					cells, partitionErr := causal.Partition(subset, action.Code())
					if partitionErr != nil {
						return partitionErr
					}
					report.ActionPartitions++
					if len(cells) > report.MaximumCellsPerAction {
						report.MaximumCellsPerAction = len(cells)
					}
					if len(cells) > 8 {
						return fmt.Errorf("action %s produced %d cells", action.Code(), len(cells))
					}
				}
				for mask := uint8(0); mask < 1<<len(causal.Actions()); mask++ {
					groups := inducedGroups(corpus, subset, mask)
					depth := bits.OnesCount8(mask)
					bound := minInt(powInt(8, depth), causal.MaximumPool)
					report.ActionSubsetChecks++
					report.InducedPosteriorCells += len(groups)
					if len(groups) > bound {
						return fmt.Errorf("mask=%02x induced %d states, bound %d", mask, len(groups), bound)
					}
				}
				return nil
			})
			if err != nil {
				return report, err
			}
		}
	}
	for depth := 0; depth <= len(causal.Actions()); depth++ {
		report.DepthStateBounds[depth] = minInt(powInt(8, depth), causal.MaximumPool)
		report.ReachableStateBound += choose(len(causal.Actions()), depth) * report.DepthStateBounds[depth]
	}
	perState := 1 + len(causal.Actions()) + 2*8*len(causal.Actions())
	report.TableConstructionWorkBound = report.ReachableStateBound * perState
	report.TrajectoryLookupWorkBound = (1 + causal.MaximumPool) * len(causal.Actions())
	report.TotalWorkBound = report.TableConstructionWorkBound + report.TrajectoryLookupWorkBound
	if report.ReachableStateBound != wantReachableStateBound ||
		report.TableConstructionWorkBound != wantTableConstructionWork ||
		report.TrajectoryLookupWorkBound != wantTrajectoryLookupWork ||
		report.TotalWorkBound != wantTotalWorkBound {
		return report, fmt.Errorf("analytical bounds changed: states=%d table=%d trajectory=%d total=%d",
			report.ReachableStateBound, report.TableConstructionWorkBound,
			report.TrajectoryLookupWorkBound, report.TotalWorkBound)
	}
	return report, nil
}

func enumerateSubsets(source []string, size int, visit func([]string) error) error {
	if size < 0 || size > len(source) {
		return errors.New("invalid subset size")
	}
	selected := make([]string, size)
	var walk func(int, int) error
	walk = func(sourceIndex, selectedIndex int) error {
		if selectedIndex == size {
			return visit(append([]string(nil), selected...))
		}
		remaining := size - selectedIndex
		for index := sourceIndex; index <= len(source)-remaining; index++ {
			selected[selectedIndex] = source[index]
			if err := walk(index+1, selectedIndex+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(0, 0)
}

func inducedGroups(corpus frozenCorpus, posterior []string, mask uint8) [][]string {
	byOutcomes := make(map[string][]string)
	for _, code := range posterior {
		model := corpus.byCode[code]
		var key strings.Builder
		for action := range model.Outcomes {
			if mask&(1<<action) != 0 {
				key.WriteString(model.Outcomes[action])
				key.WriteByte('/')
			}
		}
		byOutcomes[key.String()] = append(byOutcomes[key.String()], code)
	}
	keys := sortedKeys(byOutcomes)
	groups := make([][]string, 0, len(keys))
	for _, key := range keys {
		group := append([]string(nil), byOutcomes[key]...)
		sort.Strings(group)
		groups = append(groups, group)
	}
	return groups
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func choose(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	result := 1
	for index := 1; index <= k; index++ {
		result = result * (n - index + 1) / index
	}
	return result
}

func powInt(base, exponent int) int {
	result := 1
	for range exponent {
		result *= base
	}
	return result
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
