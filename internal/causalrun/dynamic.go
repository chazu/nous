package causalrun

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

type dynamicValue struct {
	finite bool
	value  *big.Rat
	action string
}

// DynamicProofEvaluation is the closed, read-only observation surface used by
// the development DP proof. It exposes the exact value and tariff counters for
// one public state without executing a realized or hidden-member trajectory.
type DynamicProofEvaluation struct {
	Finite      bool   `json:"finite"`
	Numerator   string `json:"numerator"`
	Denominator string `json:"denominator"`
	Action      string `json:"action"`
	Counts      Counts `json:"counts"`
}

const (
	dynamicReachableStateBound = 1873
	dynamicCompleteWorkBound   = 193117
)

// DynamicBenchmark is the complete, same-source benchmark for one dynamic
// episode. Counts includes table construction, the realized trajectory, and
// one separately metered trajectory for every member of InitialPosterior.
type DynamicBenchmark struct {
	RealizedCost               int    `json:"realized_cost"`
	UniformExpectedNumerator   string `json:"uniform_expected_numerator"`
	UniformExpectedDenominator string `json:"uniform_expected_denominator"`
	MemberSimulations          int    `json:"member_simulations"`
	Counts                     Counts `json:"counts"`
}

type dynamicChoice struct {
	posterior string
	mask      uint8
	action    string
}

// DynamicPolicy is the production, hidden-free dynamic policy. It depends
// only on the public initial posterior, public intervention costs, current
// posterior, and the already-consumed actions.
type DynamicPolicy struct {
	initial   []string
	costs     [3]int
	memo      map[string]dynamicValue
	meter     WorkMeter
	built     bool
	realized  []dynamicChoice
	benchmark *DynamicBenchmark
}

func (d *DynamicPolicy) ensureTable() error {
	if d.built {
		return nil
	}
	if _, err := d.value(d.initial, 0); err != nil {
		return err
	}
	d.built = true
	return nil
}

func (d *DynamicPolicy) tableAction(posterior []string, mask uint8) (string, error) {
	if err := d.ensureTable(); err != nil {
		return "", err
	}
	if err := d.meter.chargeTable(1); err != nil {
		return "", err
	}
	value, ok := d.memo[dynamicStateKey(posterior, mask)]
	if !ok {
		return "", errors.New("dynamic table lacks reachable state")
	}
	if !value.finite || value.action == "" {
		return "", errors.New("no finite dynamic action")
	}
	return value.action, nil
}

func NewDynamicPolicy(initial []string, costs [3]int) (*DynamicPolicy, error) {
	if err := validatePosterior(initial, 8, causal.MaximumPool); err != nil {
		return nil, err
	}
	for _, cost := range costs {
		if cost < 1 || cost > 100 {
			return nil, fmt.Errorf("invalid intervention cost %d", cost)
		}
	}
	return &DynamicPolicy{
		initial: append([]string(nil), initial...),
		costs:   costs,
		memo:    make(map[string]dynamicValue),
		meter:   WorkMeter{dynamic: true},
	}, nil
}

func dynamicTerminal(initial, posterior []string) bool {
	return len(posterior) == 1 || causal.CompleteClass(initial, posterior)
}

func dynamicStateKey(posterior []string, mask uint8) string {
	canonical := append([]string(nil), posterior...)
	sort.Strings(canonical)
	return fmt.Sprintf("%02x|%s", mask, strings.Join(canonical, "\x00"))
}

func actionMask(consumed []string) (uint8, error) {
	var mask uint8
	actions := causal.Actions()
	for _, code := range consumed {
		found := false
		for index, action := range actions {
			if action.Code() == code {
				mask |= 1 << index
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("invalid consumed action %q", code)
		}
	}
	return mask, nil
}

func (d *DynamicPolicy) value(posterior []string, mask uint8) (dynamicValue, error) {
	if err := d.meter.chargeMemoLookup(1); err != nil {
		return dynamicValue{}, err
	}
	key := dynamicStateKey(posterior, mask)
	if cached, ok := d.memo[key]; ok {
		return cached, nil
	}
	if err := d.meter.chargeMemoState(1); err != nil {
		return dynamicValue{}, err
	}
	if d.meter.counts.MemoStates > DynamicStateCap {
		return dynamicValue{}, errors.New("dynamic state cap exceeded")
	}
	if dynamicTerminal(d.initial, posterior) {
		terminal := dynamicValue{finite: true, value: new(big.Rat)}
		d.memo[key] = terminal
		return terminal, nil
	}

	var best dynamicValue
	for index, action := range causal.Actions() {
		if mask&(1<<index) != 0 {
			continue
		}
		if err := d.meter.chargeQ(1); err != nil {
			return dynamicValue{}, err
		}
		cells, err := causal.Partition(posterior, action.Code())
		if err != nil {
			return dynamicValue{}, err
		}
		if len(cells) <= 1 {
			continue
		}
		q := new(big.Rat).SetInt64(int64(d.costs[action.Variable]))
		finite := true
		for _, cell := range cells {
			if err := d.meter.chargeTable(1); err != nil {
				return dynamicValue{}, err
			}
			child, err := d.value(cell.Hypotheses, mask|(1<<index))
			if err != nil {
				return dynamicValue{}, err
			}
			if !child.finite {
				finite = false
				break
			}
			term := new(big.Rat).Mul(
				new(big.Rat).SetFrac64(int64(len(cell.Hypotheses)), int64(len(posterior))),
				child.value,
			)
			q.Add(q, term)
		}
		code := action.Code()
		if finite && (!best.finite || q.Cmp(best.value) < 0 || (q.Cmp(best.value) == 0 && code < best.action)) {
			best = dynamicValue{finite: true, value: new(big.Rat).Set(q), action: code}
		}
		if d.meter.counts.TotalWork > DynamicWorkCap {
			return dynamicValue{}, errors.New("dynamic work cap exceeded")
		}
	}
	d.memo[key] = best
	return best, nil
}

func (d *DynamicPolicy) Choose(posterior, consumed []string) (string, error) {
	if err := validatePosterior(posterior, 1, causal.MaximumPool); err != nil {
		return "", err
	}
	mask, err := actionMask(consumed)
	if err != nil {
		return "", err
	}
	action, err := d.tableAction(posterior, mask)
	if err != nil {
		return "", err
	}
	d.realized = append(d.realized, dynamicChoice{posterior: dynamicStateKey(posterior, mask), mask: mask, action: action})
	return action, nil
}

func (d *DynamicPolicy) Counts() Counts { return d.meter.Counts() }

// CompleteBenchmark completes the production policy's meter from public
// episode data. It uses the already-built table and same meter for every
// possible member trajectory; no second policy or unmetered lookup is used.
func (d *DynamicPolicy) CompleteBenchmark(actions, outcomes []string) (DynamicBenchmark, error) {
	if d.benchmark != nil {
		return *d.benchmark, nil
	}
	if len(actions) != len(outcomes) || len(actions) != len(d.realized) {
		return DynamicBenchmark{}, errors.New("dynamic realized trace cardinality mismatch")
	}
	posterior := append([]string(nil), d.initial...)
	var mask uint8
	realizedCost := 0
	for index, action := range actions {
		if err := validateThreeBitOutcome(outcomes[index]); err != nil {
			return DynamicBenchmark{}, err
		}
		key := dynamicStateKey(posterior, mask)
		choice := d.realized[index]
		if choice.posterior != key || choice.mask != mask || choice.action != action {
			return DynamicBenchmark{}, fmt.Errorf("dynamic realized step %d differs from production lookup", index)
		}
		value, ok := d.memo[key]
		if !ok || !value.finite || value.action != action {
			return DynamicBenchmark{}, fmt.Errorf("dynamic realized step %d differs from table", index)
		}
		actionValue, err := causal.ParseAction(action)
		if err != nil {
			return DynamicBenchmark{}, err
		}
		realizedCost += d.costs[actionValue.Variable]
		posterior, err = posteriorForOutcome(posterior, action, outcomes[index])
		if err != nil {
			return DynamicBenchmark{}, err
		}
		mask |= 1 << actionIndex(action)
	}
	if !dynamicTerminal(d.initial, posterior) {
		return DynamicBenchmark{}, errors.New("dynamic realized trace is not terminal")
	}

	totalCost := new(big.Rat)
	for _, hidden := range d.initial {
		cost, err := d.simulateMember(hidden)
		if err != nil {
			return DynamicBenchmark{}, err
		}
		totalCost.Add(totalCost, new(big.Rat).SetInt64(int64(cost)))
	}
	expected := totalCost.Quo(totalCost, new(big.Rat).SetInt64(int64(len(d.initial))))
	counts := d.Counts()
	if counts.MemoStates > dynamicReachableStateBound {
		return DynamicBenchmark{}, fmt.Errorf("dynamic reachable states=%d exceeds %d", counts.MemoStates, dynamicReachableStateBound)
	}
	if counts.TotalWork > dynamicCompleteWorkBound {
		return DynamicBenchmark{}, fmt.Errorf("complete dynamic work=%d exceeds %d", counts.TotalWork, dynamicCompleteWorkBound)
	}
	result := DynamicBenchmark{
		RealizedCost: realizedCost, UniformExpectedNumerator: expected.Num().String(),
		UniformExpectedDenominator: expected.Denom().String(), MemberSimulations: len(d.initial), Counts: counts,
	}
	d.benchmark = &result
	return result, nil
}

func (d *DynamicPolicy) simulateMember(hidden string) (int, error) {
	posterior := append([]string(nil), d.initial...)
	var mask uint8
	cost := 0
	for steps := 0; !dynamicTerminal(d.initial, posterior); steps++ {
		if steps >= len(causal.Actions()) {
			return 0, errors.New("dynamic member simulation exceeded action set")
		}
		action, err := d.tableAction(posterior, mask)
		if err != nil {
			return 0, err
		}
		actionValue, err := causal.ParseAction(action)
		if err != nil {
			return 0, err
		}
		cost += d.costs[actionValue.Variable]
		outcome, err := causal.PredictCode(hidden, action)
		if err != nil {
			return 0, err
		}
		posterior, err = posteriorForOutcome(posterior, action, outcome)
		if err != nil {
			return 0, err
		}
		mask |= 1 << actionIndex(action)
	}
	return cost, nil
}

func actionIndex(code string) uint8 {
	for index, action := range causal.Actions() {
		if action.Code() == code {
			return uint8(index)
		}
	}
	return uint8(len(causal.Actions()))
}

func posteriorForOutcome(posterior []string, action, outcome string) ([]string, error) {
	cells, err := causal.Partition(posterior, action)
	if err != nil {
		return nil, err
	}
	for _, cell := range cells {
		if cell.Outcome == outcome {
			return append([]string(nil), cell.Hypotheses...), nil
		}
	}
	return nil, fmt.Errorf("outcome %q absent from dynamic partition", outcome)
}

// ReconstructDynamicBenchmark is the independent public-data verifier for a
// retained benchmark. It rebuilds the policy, replays the realized decisions,
// and then meters all hidden-member simulations.
func ReconstructDynamicBenchmark(initial []string, costs [3]int, actions, outcomes []string) (DynamicBenchmark, error) {
	policy, err := NewDynamicPolicy(initial, costs)
	if err != nil {
		return DynamicBenchmark{}, err
	}
	posterior := append([]string(nil), initial...)
	consumed := make([]string, 0, len(actions))
	for index, want := range actions {
		got, chooseErr := policy.Choose(posterior, consumed)
		if chooseErr != nil {
			return DynamicBenchmark{}, chooseErr
		}
		if got != want {
			return DynamicBenchmark{}, fmt.Errorf("dynamic action %d=%q, want %q", index, got, want)
		}
		if index >= len(outcomes) {
			return DynamicBenchmark{}, errors.New("dynamic outcome trace is short")
		}
		posterior, err = posteriorForOutcome(posterior, want, outcomes[index])
		if err != nil {
			return DynamicBenchmark{}, err
		}
		consumed = append(consumed, want)
	}
	return policy.CompleteBenchmark(actions, outcomes)
}

// EvaluateDynamicProof evaluates one public DP state using the production
// implementation. It exists solely so an independent development proof can
// compare exact values, choices, and causal-work/v2 charges. Unlike Choose, it
// does not add a realized-trajectory table lookup.
func EvaluateDynamicProof(initial []string, costs [3]int, posterior, consumed []string) (DynamicProofEvaluation, error) {
	policy, err := NewDynamicPolicy(initial, costs)
	if err != nil {
		return DynamicProofEvaluation{}, err
	}
	if err := validatePosterior(posterior, 1, causal.MaximumPool); err != nil {
		return DynamicProofEvaluation{}, err
	}
	mask, err := actionMask(consumed)
	if err != nil {
		return DynamicProofEvaluation{}, err
	}
	value, err := policy.value(posterior, mask)
	if err != nil {
		return DynamicProofEvaluation{}, err
	}
	result := DynamicProofEvaluation{
		Finite: value.finite,
		Action: value.action,
		Counts: policy.Counts(),
	}
	if value.finite {
		result.Numerator = value.value.Num().String()
		result.Denominator = value.value.Denom().String()
	}
	return result, nil
}

func validatePosterior(codes []string, minimum, maximum int) error {
	if len(codes) < minimum || len(codes) > maximum {
		return fmt.Errorf("posterior size=%d outside [%d,%d]", len(codes), minimum, maximum)
	}
	previous := ""
	for index, code := range codes {
		if _, err := causal.Parse(code); err != nil {
			return fmt.Errorf("posterior[%d]: %w", index, err)
		}
		if index > 0 && code <= previous {
			return errors.New("posterior is not strictly sorted")
		}
		previous = code
	}
	return nil
}
