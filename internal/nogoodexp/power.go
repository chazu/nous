package nogoodexp

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
)

const (
	PowerOuterReplicates = 2000
	PowerInnerReplicates = 2000
)

type PowerEstimate struct {
	Passing    int      `json:"passing"`
	Replicates int      `json:"replicates"`
	Fraction   Fraction `json:"fraction"`
	Authorized bool     `json:"authorized"`
}

func EstimateDevelopmentPower(execution PanelExecution) (PowerEstimate, error) {
	return estimateDevelopmentPower(execution, PowerOuterReplicates, PowerInnerReplicates)
}

func estimateDevelopmentPower(execution PanelExecution, outerReplicates, innerReplicates int) (PowerEstimate, error) {
	if outerReplicates <= 0 || innerReplicates < 40 {
		return PowerEstimate{}, fmt.Errorf("invalid power replicate counts")
	}
	paired, err := pairedDevelopmentTasks(execution)
	if err != nil {
		return PowerEstimate{}, err
	}
	strata := stratify(paired)
	workers := min(runtime.GOMAXPROCS(0), outerReplicates)
	jobs := make(chan int)
	results := make(chan bool, outerReplicates)
	errors := make(chan error, 1)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for outer := range jobs {
				passed, replicateErr := developmentPowerReplicate(strata, execution.AcquisitionWork, outer, innerReplicates)
				if replicateErr != nil {
					select {
					case errors <- replicateErr:
					default:
					}
					return
				}
				results <- passed
			}
		}()
	}
	go func() {
		defer close(jobs)
		for outer := 0; outer < outerReplicates; outer++ {
			jobs <- outer
		}
	}()
	go func() {
		wait.Wait()
		close(results)
	}()
	passing, completed := 0, 0
	for passed := range results {
		completed++
		if passed {
			passing++
		}
	}
	select {
	case err := <-errors:
		return PowerEstimate{}, err
	default:
	}
	if completed != outerReplicates {
		return PowerEstimate{}, fmt.Errorf("power simulation completed %d of %d replicates", completed, outerReplicates)
	}
	estimate := PowerEstimate{Passing: passing, Replicates: outerReplicates, Fraction: Fraction{int64(passing), int64(outerReplicates)}}
	estimate.Authorized = passing*100 >= 80*outerReplicates
	return estimate, nil
}

func pairedDevelopmentTasks(execution PanelExecution) ([]pairedTask, error) {
	if execution.Panel != "development" {
		return nil, fmt.Errorf("power requires development execution")
	}
	selected, err := policyByName(execution, "nous-generalized", "mac-cbj")
	if err != nil {
		return nil, err
	}
	learned, mac := selected[0], selected[1]
	if len(learned.Tasks) != DevelopmentTaskCount || len(mac.Tasks) != DevelopmentTaskCount {
		return nil, fmt.Errorf("power requires %d paired development tasks", DevelopmentTaskCount)
	}
	paired := make([]pairedTask, DevelopmentTaskCount)
	for index := range paired {
		if learned.Tasks[index].Ordinal != mac.Tasks[index].Ordinal || learned.Tasks[index].Cohort != mac.Tasks[index].Cohort {
			return nil, fmt.Errorf("power paired task order mismatch")
		}
		difference, ok := checkedSub(learned.Tasks[index].Work, mac.Tasks[index].Work)
		if !ok || mac.Tasks[index].Work <= 0 {
			return nil, fmt.Errorf("power paired work invalid")
		}
		cell := semanticCell(learned.Tasks[index])
		if cell == "" {
			return nil, fmt.Errorf("power task has unknown semantic cell")
		}
		paired[index] = pairedTask{ordinal: index, cell: cell, cohort: learned.Tasks[index].Cohort, d: difference, m: mac.Tasks[index].Work}
	}
	return paired, nil
}

func developmentPowerReplicate(development map[string][]pairedTask, acquisition int64, outer, innerReplicates int) (bool, error) {
	panelRNG := statisticsStream("power", 832001, outer, "panel")
	synthetic := make([]pairedTask, 0, LockedTaskCount)
	for _, cell := range orderedCells() {
		members := development[cell]
		count := lockedCellCount(cell)
		if len(members) == 0 || count == 0 {
			return false, fmt.Errorf("power cell %s has %d development members and target count %d", cell, len(members), count)
		}
		for range count {
			selected := members[panelRNG.Uint64N(uint64(len(members)))]
			synthetic = append(synthetic, selected)
		}
	}
	if len(synthetic) != LockedTaskCount {
		return false, fmt.Errorf("synthetic locked panel has %d tasks", len(synthetic))
	}
	var numerator, denominator, harmNumerator, harmDenominator int64
	var ok bool
	for _, task := range synthetic {
		if numerator, ok = checkedAdd(numerator, task.d); !ok {
			return false, fmt.Errorf("power point overflow")
		}
		if denominator, ok = checkedAdd(denominator, task.m); !ok {
			return false, fmt.Errorf("power denominator overflow")
		}
		if task.cohort == "near-miss" || task.cohort == "irrelevant" {
			harmNumerator, ok = checkedAdd(harmNumerator, task.d)
			if !ok {
				return false, fmt.Errorf("power harm overflow")
			}
			harmDenominator, ok = checkedAdd(harmDenominator, task.m)
			if !ok {
				return false, fmt.Errorf("power harm denominator overflow")
			}
		}
	}
	numerator, ok = checkedAdd(numerator, acquisition)
	if !ok || denominator <= 0 || harmDenominator <= 0 {
		return false, fmt.Errorf("power point invalid")
	}
	bootstrap := make([]struct {
		fraction  Fraction
		replicate int
	}, innerReplicates)
	bootstrapRNG := statisticsStream("power", 832001, outer, "bootstrap")
	syntheticStrata := stratify(synthetic)
	for replicate := 0; replicate < innerReplicates; replicate++ {
		bootNumerator, bootDenominator := acquisition, int64(0)
		for _, cell := range orderedCells() {
			members := syntheticStrata[cell]
			for range members {
				sampled := members[bootstrapRNG.Uint64N(uint64(len(members)))]
				bootNumerator, ok = checkedAdd(bootNumerator, sampled.d)
				if !ok {
					return false, fmt.Errorf("power bootstrap numerator overflow")
				}
				bootDenominator, ok = checkedAdd(bootDenominator, sampled.m)
				if !ok {
					return false, fmt.Errorf("power bootstrap denominator overflow")
				}
			}
		}
		bootstrap[replicate] = struct {
			fraction  Fraction
			replicate int
		}{Fraction{bootNumerator, bootDenominator}, replicate}
	}
	sort.Slice(bootstrap, func(i, j int) bool {
		if comparison := compareFraction(bootstrap[i].fraction, bootstrap[j].fraction); comparison != 0 {
			return comparison < 0
		}
		return bootstrap[i].replicate < bootstrap[j].replicate
	})
	upperIndex := innerReplicates - innerReplicates/40 - 1
	observed, ok := checkedMul(LockedTaskCount, numerator)
	if !ok || observed == math.MinInt64 {
		return false, fmt.Errorf("power observed statistic overflow")
	}
	randomRNG := statisticsStream("power", 832001, outer, "randomization")
	extreme := 0
	for range innerReplicates {
		var randomized int64
		for _, task := range synthetic {
			value, multiplyOK := checkedMul(LockedTaskCount, task.d)
			if !multiplyOK {
				return false, fmt.Errorf("power randomization overflow")
			}
			value, ok = checkedAdd(value, acquisition)
			if !ok || value == math.MinInt64 {
				return false, fmt.Errorf("power randomization overflow")
			}
			if randomRNG.Uint64N(2) == 0 {
				randomized, ok = checkedAdd(randomized, value)
			} else {
				randomized, ok = checkedSub(randomized, value)
			}
			if !ok {
				return false, fmt.Errorf("power randomization sum overflow")
			}
		}
		if randomized == math.MinInt64 {
			return false, fmt.Errorf("power randomization absolute overflow")
		}
		if abs64(randomized) >= abs64(observed) {
			extreme++
		}
	}
	return numerator < 0 && bootstrap[upperIndex].fraction.Numerator < 0 && float64(1+extreme)/float64(1+innerReplicates) < .05 && float64(harmNumerator)/float64(harmDenominator) <= .10, nil
}

func lockedCellCount(cell string) int {
	switch cell[0] {
	case 'r':
		return 78
	case 'n':
		return 4
	case 'i', 'u':
		return 3
	default:
		return 0
	}
}
