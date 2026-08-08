package causalexpv2

import (
	"errors"
	"fmt"

	"github.com/chazu/nous/internal/causalv2"
)

type Counter = causalv2.Counter
type MeterItem = causalv2.MeterItem
type TaskMeterItem = causalv2.TaskMeterItem
type MeterCaps = causalv2.MeterCaps
type AggregateMeter = causalv2.AggregateMeter

type MeterScope string

const (
	MeterTraining   MeterScope = "training"
	MeterEvaluation MeterScope = "evaluation"
)

func VerifyEpisodeMeterItems(items []MeterItem) error {
	return causalv2.ValidateMeterItems(items)
}

func VerifyTaskMeterItems(items []TaskMeterItem) (string, error) {
	return causalv2.TaskMeterItemsDigest(items)
}

// ReconstructMeters computes the only valid aggregate array from source items.
// Inactive episode items are checked but excluded from N, totals, and maxima.
func ReconstructMeters(scope MeterScope, episodeItems [][]MeterItem, tasks []TaskMeterItem, controls [][15]int64) ([]AggregateMeter, string, error) {
	grouped := make(map[string][]Counter, len(causalv2.MeterNames))
	for _, array := range episodeItems {
		if err := causalv2.ValidateMeterItems(array); err != nil {
			return nil, "", err
		}
		for _, item := range array {
			if item.Active {
				grouped[item.Name] = append(grouped[item.Name], item.Counter())
			}
		}
	}
	if _, err := causalv2.TaskMeterItemsDigest(tasks); err != nil {
		return nil, "", err
	}
	for _, item := range tasks {
		grouped[item.Name] = append(grouped[item.Name], item.Counter())
	}
	for _, counts := range controls {
		counter := causalv2.CounterFromCounts(counts)
		if err := counter.Validate(); err != nil {
			return nil, "", err
		}
		grouped["controls"] = append(grouped["controls"], counter)
	}
	meters := make([]AggregateMeter, 0, len(causalv2.MeterNames))
	for _, name := range causalv2.MeterNames {
		meter, err := causalv2.NewAggregateMeter(name, string(scope), grouped[name])
		if err != nil {
			return nil, "", err
		}
		meters = append(meters, meter)
	}
	digest, err := causalv2.MeterArrayDigest(meters)
	return meters, digest, err
}

func VerifyAggregateMeters(got []AggregateMeter, scope MeterScope, episodeItems [][]MeterItem, tasks []TaskMeterItem, controls [][15]int64) error {
	want, _, err := ReconstructMeters(scope, episodeItems, tasks, controls)
	if err != nil {
		return err
	}
	gotBytes, _ := causalv2.CanonicalJSON(got)
	wantBytes, _ := causalv2.CanonicalJSON(want)
	if string(gotBytes) != string(wantBytes) {
		return errors.New("aggregate meter array does not reconstruct from source items")
	}
	return nil
}

func verifyMeterCardinality(scope MeterScope, meters []AggregateMeter, panelSeedCount int) error {
	if len(meters) != len(causalv2.MeterNames) {
		return errors.New("aggregate meter array has wrong cardinality")
	}
	byName := make(map[string]int64, len(meters))
	for _, meter := range meters {
		byName[meter.Name] = meter.Episodes
	}
	wantEpisodes := int64(480)
	wantDP := int64(12)
	if scope == MeterEvaluation {
		wantEpisodes = int64(7 * panelSeedCount)
		wantDP = int64(panelSeedCount)
	}
	wantCurriculum := int64(525)
	if scope == MeterEvaluation {
		wantCurriculum = 0
	}
	wants := map[string]int64{"production": wantEpisodes, "teacher": wantEpisodes, "oracle-audit": wantEpisodes, "dp": wantDP, "certificate-replay": 480, "post-selection-replay": 480, "controls": 18, "curriculum": wantCurriculum}
	for name, want := range wants {
		if byName[name] != want {
			return fmt.Errorf("meter %s N=%d, want %d", name, byName[name], want)
		}
	}
	return nil
}
