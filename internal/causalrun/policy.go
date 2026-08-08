package causalrun

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

type Policy string

const (
	PolicyInformationGain Policy = "information-gain-per-cost"
	PolicyWorstSplit      Policy = "worst-split-per-cost"
	PolicyLexical         Policy = "lexical-fixed"
	PolicyUniformRandom   Policy = "uniform-random"
	PolicyPassiveOnly     Policy = "passive-only"
	PolicyDynamicOptimal  Policy = "dynamic-optimal"
)

type featureWire struct {
	E int    `json:"E"`
	W int    `json:"W"`
	H string `json:"H"`
	C int    `json:"C"`
	R int    `json:"R"`
}

type candidate struct {
	action   string
	cells    []causal.Cell
	features causal.Features
}

func canonicalCells(cells []causal.Cell) []causal.Cell {
	out := make([]causal.Cell, len(cells))
	for index, cell := range cells {
		out[index] = causal.Cell{
			Outcome:    cell.Outcome,
			Hypotheses: append([]string(nil), cell.Hypotheses...),
		}
	}
	return out
}

func cloneFeatures(features causal.Features) causal.Features {
	cloned := features
	if features.EntropyProduct != nil {
		cloned.EntropyProduct = new(big.Int).Set(features.EntropyProduct)
	}
	return cloned
}

func partitionWithMeter(posterior []string, action string, meter *WorkMeter) ([]causal.Cell, error) {
	if err := validatePosterior(posterior, 1, causal.MaximumPool); err != nil {
		return nil, err
	}
	if _, err := causal.ParseAction(action); err != nil {
		return nil, err
	}
	byOutcome := make(map[string][]string)
	for _, code := range posterior {
		if err := meter.chargeSCM(1); err != nil {
			return nil, err
		}
		outcome, err := causal.PredictCode(code, action)
		if err != nil {
			return nil, err
		}
		if err := meter.chargeAssignment(1); err != nil {
			return nil, err
		}
		byOutcome[outcome] = append(byOutcome[outcome], code)
	}
	var cells []causal.Cell
	for value := 0; value < 8; value++ {
		outcome := fmt.Sprintf("%03b", value)
		if hypotheses := byOutcome[outcome]; len(hypotheses) != 0 {
			sort.Strings(hypotheses)
			cells = append(cells, causal.Cell{Outcome: outcome, Hypotheses: hypotheses})
		}
	}
	return cells, nil
}

func featuresFromCells(cells []causal.Cell, cost int, repeated bool, meter *WorkMeter) (causal.Features, error) {
	features := causal.Features{Cost: cost, EntropyProduct: big.NewInt(1)}
	if repeated {
		features.Repeat = 1
	}
	for _, cell := range cells {
		if err := meter.chargeCell(3); err != nil {
			return causal.Features{}, err
		}
		n := len(cell.Hypotheses)
		features.ExpectedNumerator += n * n
		if n > features.Worst {
			features.Worst = n
		}
		features.EntropyProduct.Mul(
			features.EntropyProduct,
			new(big.Int).Exp(big.NewInt(int64(n)), big.NewInt(int64(n)), nil),
		)
	}
	return features, nil
}

func wireFeatures(features causal.Features) featureWire {
	return featureWire{
		E: features.ExpectedNumerator,
		W: features.Worst,
		H: features.EntropyProduct.String(),
		C: features.Cost,
		R: features.Repeat,
	}
}

func acquisitionRule(code string) (causal.Rule, bool, error) {
	switch code {
	case string(PolicyInformationGain):
		rule, err := causal.ParseRule("P=H;M=gain;S=C")
		return rule, true, err
	case string(PolicyWorstSplit):
		rule, err := causal.ParseRule("P=W;M=gain;S=C")
		return rule, true, err
	case string(PolicyLexical), string(PolicyUniformRandom), string(PolicyPassiveOnly), string(PolicyDynamicOptimal):
		return causal.Rule{}, false, nil
	default:
		rule, err := causal.ParseRule(code)
		if err != nil {
			return causal.Rule{}, false, err
		}
		return rule, true, nil
	}
}

func containsAction(actions []string, want string) bool {
	for _, action := range actions {
		if action == want {
			return true
		}
	}
	return false
}

func compareCandidates(code string, posteriorSize int, left, right candidate, forced string, meter *WorkMeter) (int, error) {
	if err := meter.chargeComparison(1); err != nil {
		return 0, err
	}
	switch code {
	case string(PolicyUniformRandom), string(PolicyDynamicOptimal):
		if left.action == forced && right.action != forced {
			return -1, nil
		}
		if right.action == forced && left.action != forced {
			return 1, nil
		}
		return strings.Compare(left.action, right.action), nil
	case string(PolicyLexical):
		if left.features.Repeat < right.features.Repeat {
			return -1, nil
		}
		if left.features.Repeat > right.features.Repeat {
			return 1, nil
		}
		return strings.Compare(left.action, right.action), nil
	case string(PolicyPassiveOnly):
		return 0, errors.New("passive-only has no action comparison")
	default:
		rule, ok, err := acquisitionRule(code)
		if err != nil || !ok {
			return 0, fmt.Errorf("invalid acquisition code %q", code)
		}
		comparison := causal.Compare(rule, posteriorSize, left.features, right.features)
		if comparison == 0 {
			return strings.Compare(left.action, right.action), nil
		}
		return comparison, nil
	}
}

func candidatesTie(code string, posteriorSize int, left, right candidate, forced string, meter *WorkMeter) (bool, error) {
	if err := meter.chargeComparison(1); err != nil {
		return false, err
	}
	switch code {
	case string(PolicyUniformRandom), string(PolicyDynamicOptimal):
		return left.action == forced && right.action == forced, nil
	case string(PolicyLexical):
		return left.features.Repeat == right.features.Repeat, nil
	case string(PolicyPassiveOnly):
		return false, errors.New("passive-only has no tie set")
	default:
		rule, ok, err := acquisitionRule(code)
		if err != nil || !ok {
			return false, fmt.Errorf("invalid acquisition code %q", code)
		}
		return causal.Compare(rule, posteriorSize, left.features, right.features) == 0, nil
	}
}
