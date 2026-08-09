package nogoodexp

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"sort"
)

const InferenceReplicates = 10000

type Fraction struct {
	Numerator   int64 `json:"numerator"`
	Denominator int64 `json:"denominator"`
}

func (fraction Fraction) Float64() float64 {
	if fraction.Denominator == 0 {
		return 0
	}
	return float64(fraction.Numerator) / float64(fraction.Denominator)
}

type Inference struct {
	Point                Fraction `json:"point"`
	IntervalLower        Fraction `json:"interval_lower"`
	IntervalUpper        Fraction `json:"interval_upper"`
	RandomizationExtreme int      `json:"randomization_extreme"`
	RandomizationP       Fraction `json:"randomization_p"`
	NonreusableHarm      Fraction `json:"nonreusable_harm"`
	Classification       string   `json:"classification"`
}

type pairedTask struct {
	ordinal int
	cell    string
	cohort  string
	d       int64
	m       int64
}

func InferDevelopment(execution PanelExecution) (Inference, error) {
	if execution.Panel != "development" || len(execution.Policies) != len(RequiredPolicies) {
		return Inference{}, fmt.Errorf("inference requires a complete development execution")
	}
	selected, err := policyByName(execution, "nous-generalized", "mac-cbj")
	if err != nil {
		return Inference{}, err
	}
	learned, mac := selected[0], selected[1]
	if len(learned.Tasks) != 96 || len(mac.Tasks) != 96 {
		return Inference{}, fmt.Errorf("development inference requires 96 paired tasks")
	}
	paired := make([]pairedTask, len(mac.Tasks))
	var numerator, denominator, harmNumerator, harmDenominator int64
	var ok bool
	for index := range paired {
		if learned.Tasks[index].Ordinal != mac.Tasks[index].Ordinal || learned.Tasks[index].Cohort != mac.Tasks[index].Cohort {
			return Inference{}, fmt.Errorf("paired task order mismatch")
		}
		cell := semanticCell(learned.Tasks[index])
		if cell == "" || learned.Tasks[index].Work < 0 || mac.Tasks[index].Work <= 0 {
			return Inference{}, fmt.Errorf("invalid paired work at task %d", index)
		}
		difference, ok := checkedSub(learned.Tasks[index].Work, mac.Tasks[index].Work)
		if !ok {
			return Inference{}, fmt.Errorf("paired difference overflow")
		}
		paired[index] = pairedTask{ordinal: learned.Tasks[index].Ordinal, cell: cell, cohort: learned.Tasks[index].Cohort, d: difference, m: mac.Tasks[index].Work}
		if numerator, ok = checkedAdd(numerator, paired[index].d); !ok {
			return Inference{}, fmt.Errorf("point numerator overflow")
		}
		if denominator, ok = checkedAdd(denominator, paired[index].m); !ok {
			return Inference{}, fmt.Errorf("point denominator overflow")
		}
		if paired[index].cohort == "near-miss" || paired[index].cohort == "irrelevant" {
			if harmNumerator, ok = checkedAdd(harmNumerator, paired[index].d); !ok {
				return Inference{}, fmt.Errorf("harm numerator overflow")
			}
			if harmDenominator, ok = checkedAdd(harmDenominator, paired[index].m); !ok {
				return Inference{}, fmt.Errorf("harm denominator overflow")
			}
		}
	}
	if execution.AcquisitionWork < 0 {
		return Inference{}, fmt.Errorf("negative acquisition work")
	}
	if numerator, ok = checkedAdd(numerator, execution.AcquisitionWork); !ok {
		return Inference{}, fmt.Errorf("lifecycle numerator overflow")
	}
	if denominator <= 0 || harmDenominator <= 0 {
		return Inference{}, fmt.Errorf("invalid inference denominator")
	}
	point := Fraction{numerator, denominator}
	strata := stratify(paired)
	bootstrap := make([]struct {
		fraction  Fraction
		replicate int
	}, InferenceReplicates)
	for replicate := 0; replicate < InferenceReplicates; replicate++ {
		rng := statisticsStream("development", 832001, replicate, "bootstrap/nous-vs-mac")
		bootNumerator, bootDenominator := execution.AcquisitionWork, int64(0)
		for _, cell := range orderedCells() {
			members := strata[cell]
			if len(members) == 0 {
				return Inference{}, fmt.Errorf("empty development stratum %s", cell)
			}
			for range members {
				sampled := members[rng.Uint64N(uint64(len(members)))]
				if bootNumerator, ok = checkedAdd(bootNumerator, sampled.d); !ok {
					return Inference{}, fmt.Errorf("bootstrap numerator overflow")
				}
				if bootDenominator, ok = checkedAdd(bootDenominator, sampled.m); !ok {
					return Inference{}, fmt.Errorf("bootstrap denominator overflow")
				}
			}
		}
		bootstrap[replicate].fraction = Fraction{bootNumerator, bootDenominator}
		bootstrap[replicate].replicate = replicate
	}
	sort.Slice(bootstrap, func(i, j int) bool {
		comparison := compareFraction(bootstrap[i].fraction, bootstrap[j].fraction)
		if comparison != 0 {
			return comparison < 0
		}
		return bootstrap[i].replicate < bootstrap[j].replicate
	})
	observed, ok := checkedMul(int64(len(paired)), numerator)
	if !ok || observed == math.MinInt64 {
		return Inference{}, fmt.Errorf("observed statistic overflow")
	}
	extreme := 0
	for replicate := 0; replicate < InferenceReplicates; replicate++ {
		rng := statisticsStream("development", 832001, replicate, "randomization/nous-vs-mac")
		var randomized int64
		for _, cell := range orderedCells() {
			for _, item := range strata[cell] {
				value, multiplyOK := checkedMul(int64(len(paired)), item.d)
				if !multiplyOK {
					return Inference{}, fmt.Errorf("randomization task overflow")
				}
				value, ok = checkedAdd(value, execution.AcquisitionWork)
				if !ok || value == math.MinInt64 {
					return Inference{}, fmt.Errorf("randomization task overflow")
				}
				if rng.Uint64N(2) == 0 {
					randomized, ok = checkedAdd(randomized, value)
				} else {
					randomized, ok = checkedSub(randomized, value)
				}
				if !ok {
					return Inference{}, fmt.Errorf("randomization sum overflow")
				}
			}
		}
		if randomized == math.MinInt64 {
			return Inference{}, fmt.Errorf("randomization absolute value overflow")
		}
		if abs64(randomized) >= abs64(observed) {
			extreme++
		}
	}
	inference := Inference{
		Point: point, IntervalLower: bootstrap[249].fraction, IntervalUpper: bootstrap[9749].fraction,
		RandomizationExtreme: extreme, RandomizationP: Fraction{int64(1 + extreme), 10001},
		NonreusableHarm: Fraction{harmNumerator, harmDenominator}, Classification: "valid-null",
	}
	if point.Numerator < 0 && inference.IntervalUpper.Numerator < 0 && inference.RandomizationP.Float64() < 0.05 && inference.NonreusableHarm.Float64() <= 0.10 {
		inference.Classification = "valid-positive"
	}
	return inference, nil
}

func policyByName(execution PanelExecution, names ...string) ([]PolicyExecution, error) {
	result := make([]PolicyExecution, len(names))
	for index, name := range names {
		found := false
		for _, policy := range execution.Policies {
			if policy.Policy == name {
				result[index], found = policy, true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("missing policy %s", name)
		}
	}
	return result, nil
}

func semanticCell(outcome TaskOutcome) string {
	// Development fixture ordinals are frozen: reusable 0..55, near 56..79,
	// irrelevant 80..87, independent-unsat 88..95.
	switch outcome.Cohort {
	case "reusable":
		return fmt.Sprintf("r:%d", outcome.Ordinal%4)
	case "near-miss":
		local := outcome.Ordinal - 56
		return fmt.Sprintf("n:%d:%d", outcome.Ordinal%4, local%3)
	case "irrelevant":
		return fmt.Sprintf("i:%d", outcome.Ordinal%4)
	case "independent-unsat":
		return fmt.Sprintf("u:%d", outcome.Ordinal%4)
	default:
		return ""
	}
}

func orderedCells() []string {
	var cells []string
	for template := 0; template < 4; template++ {
		cells = append(cells, fmt.Sprintf("r:%d", template))
	}
	for template := 0; template < 4; template++ {
		for bit := 0; bit < 3; bit++ {
			cells = append(cells, fmt.Sprintf("n:%d:%d", template, bit))
		}
	}
	for template := 0; template < 4; template++ {
		cells = append(cells, fmt.Sprintf("i:%d", template))
	}
	for template := 0; template < 4; template++ {
		cells = append(cells, fmt.Sprintf("u:%d", template))
	}
	return cells
}

func stratify(tasks []pairedTask) map[string][]pairedTask {
	result := map[string][]pairedTask{}
	for _, task := range tasks {
		result[task.cell] = append(result[task.cell], task)
	}
	return result
}

func statisticsStream(panel string, root any, replicate int, purpose string) *rand.Rand {
	encoded, _ := json.Marshal([]any{"part3/nogoods/v1", panel, root, replicate, purpose})
	digest := sha256.Sum256(encoded)
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func compareFraction(left, right Fraction) int {
	a := new(big.Int).Mul(big.NewInt(left.Numerator), big.NewInt(right.Denominator))
	b := new(big.Int).Mul(big.NewInt(right.Numerator), big.NewInt(left.Denominator))
	return a.Cmp(b)
}

func abs64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func checkedAdd(left, right int64) (int64, bool) {
	if right > 0 && left > math.MaxInt64-right || right < 0 && left < math.MinInt64-right {
		return 0, false
	}
	return left + right, true
}

func checkedSub(left, right int64) (int64, bool) {
	if right == math.MinInt64 {
		if left >= 0 {
			return 0, false
		}
		return left - right, true
	}
	return checkedAdd(left, -right)
}

func checkedMul(left, right int64) (int64, bool) {
	if left == 0 || right == 0 {
		return 0, true
	}
	if left == -1 && right == math.MinInt64 || right == -1 && left == math.MinInt64 {
		return 0, false
	}
	product := left * right
	return product, product/right == left
}
