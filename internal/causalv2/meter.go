package causalv2

import (
	"errors"
	"fmt"
	"slices"
)

const (
	MeterArrayDomain     = "causal-meter-array/v2"
	TaskMeterItemsDomain = "causal-task-meter-items/v2"
)

var MeterNames = []string{"production", "teacher", "certificate-replay", "post-selection-replay", "oracle-audit", "dp", "controls", "curriculum"}

type Counter struct {
	SCMEvaluations           int64 `json:"scm_evaluations"`
	PartitionAssignments     int64 `json:"partition_assignments"`
	CellAccumulations        int64 `json:"cell_accumulations"`
	RuleComparisons          int64 `json:"rule_comparisons"`
	PosteriorChecks          int64 `json:"posterior_checks"`
	ArtifactMaterializations int64 `json:"artifact_materializations"`
	TranscriptFields         int64 `json:"transcript_fields"`
	ProfileFields            int64 `json:"profile_fields"`
	MemoStates               int64 `json:"memo_states"`
	MemoLookups              int64 `json:"memo_lookups"`
	QEvaluations             int64 `json:"q_evaluations"`
	TableLookups             int64 `json:"table_lookups"`
	EngineCycles             int64 `json:"engine_cycles"`
	AttributedUnits          int64 `json:"attributed_units"`
	TotalWork                int64 `json:"total_work"`
}

func CounterFromCounts(counts [15]int64) Counter {
	return Counter{counts[0], counts[1], counts[2], counts[3], counts[4], counts[5], counts[6], counts[7], counts[8], counts[9], counts[10], counts[11], counts[12], counts[13], counts[14]}
}

func (counter Counter) Counts() [15]int64 {
	return [15]int64{counter.SCMEvaluations, counter.PartitionAssignments, counter.CellAccumulations, counter.RuleComparisons, counter.PosteriorChecks, counter.ArtifactMaterializations, counter.TranscriptFields, counter.ProfileFields, counter.MemoStates, counter.MemoLookups, counter.QEvaluations, counter.TableLookups, counter.EngineCycles, counter.AttributedUnits, counter.TotalWork}
}

func (counter Counter) ComputedTotalWork() int64 {
	return counter.SCMEvaluations + counter.PartitionAssignments + counter.CellAccumulations + counter.RuleComparisons + counter.PosteriorChecks + counter.ArtifactMaterializations + counter.TranscriptFields + counter.ProfileFields + counter.MemoStates + counter.MemoLookups + counter.QEvaluations + counter.TableLookups
}

func (counter Counter) Validate() error {
	for position, count := range counter.Counts() {
		if count < 0 {
			return fmt.Errorf("counter position %d is negative", position)
		}
	}
	if counter.TotalWork != counter.ComputedTotalWork() {
		return fmt.Errorf("total_work=%d, computed %d", counter.TotalWork, counter.ComputedTotalWork())
	}
	return nil
}

func addCounter(a, b Counter) (Counter, error) {
	ac, bc := a.Counts(), b.Counts()
	var result [15]int64
	for i := range result {
		if bc[i] > 0 && ac[i] > int64(^uint64(0)>>1)-bc[i] {
			return Counter{}, errors.New("counter sum overflow")
		}
		result[i] = ac[i] + bc[i]
	}
	return CounterFromCounts(result), nil
}

func maxCounter(a, b Counter) Counter {
	ac, bc := a.Counts(), b.Counts()
	for i := range ac {
		if bc[i] > ac[i] {
			ac[i] = bc[i]
		}
	}
	return CounterFromCounts(ac)
}

type MeterItem struct {
	Name   string    `json:"name"`
	Active bool      `json:"active"`
	Counts [15]int64 `json:"counts"`
}

func (item MeterItem) Counter() Counter { return CounterFromCounts(item.Counts) }

func (item MeterItem) Validate() error {
	if !slices.Contains(MeterNames, item.Name) {
		return fmt.Errorf("invalid meter name %q", item.Name)
	}
	counter := item.Counter()
	if err := counter.Validate(); err != nil {
		return err
	}
	if !item.Active && counter != (Counter{}) {
		return errors.New("inactive meter item has nonzero counts")
	}
	return nil
}

func ValidateMeterItems(items []MeterItem) error {
	if len(items) != len(MeterNames) {
		return fmt.Errorf("meter item count=%d, want %d", len(items), len(MeterNames))
	}
	for i, item := range items {
		if item.Name != MeterNames[i] {
			return fmt.Errorf("meter item %d name=%q, want %q", i, item.Name, MeterNames[i])
		}
		if err := item.Validate(); err != nil {
			return fmt.Errorf("meter item %q: %w", item.Name, err)
		}
	}
	encoded, err := CanonicalJSON(items)
	if err != nil {
		return err
	}
	if len(encoded)-2 > PreregisteredManifest().EpisodeMeterItemsByteCap {
		return errors.New("meter item contents exceed byte cap")
	}
	return nil
}

type TaskMeterItem struct {
	Name    string    `json:"name"`
	Subject string    `json:"subject"`
	Counts  [15]int64 `json:"counts"`
}

func (item TaskMeterItem) Counter() Counter { return CounterFromCounts(item.Counts) }

func (item TaskMeterItem) Validate() error {
	if !slices.Contains([]string{"certificate-replay", "post-selection-replay", "curriculum"}, item.Name) {
		return fmt.Errorf("invalid task meter name %q", item.Name)
	}
	if item.Subject == "" {
		return errors.New("empty task meter subject")
	}
	if err := item.Counter().Validate(); err != nil {
		return err
	}
	encoded, err := CanonicalJSON(item)
	if err != nil {
		return err
	}
	if len(encoded) > PreregisteredManifest().TaskMeterItemByteCap {
		return errors.New("task meter item exceeds byte cap")
	}
	return nil
}

func TaskMeterItemsDigest(items []TaskMeterItem) (string, error) {
	seen := make(map[string]bool, len(items))
	previousRank := -1
	for _, item := range items {
		if err := item.Validate(); err != nil {
			return "", err
		}
		rank := slices.Index([]string{"certificate-replay", "post-selection-replay", "curriculum"}, item.Name)
		if rank < previousRank {
			return "", errors.New("task meter categories are not in canonical order")
		}
		key := item.Name + "\x00" + item.Subject
		if seen[key] {
			return "", errors.New("duplicate task meter subject")
		}
		seen[key] = true
		previousRank = rank
	}
	return Digest(TaskMeterItemsDomain, items)
}

type MeterCaps struct {
	PerEpisodeEvaluationCap int64 `json:"per_episode_evaluation_cap"`
	AggregateEvaluationCap  int64 `json:"aggregate_evaluation_cap"`
	PerEpisodeStateCap      int64 `json:"per_episode_state_cap"`
	AggregateStateCap       int64 `json:"aggregate_state_cap"`
	PerEpisodeWorkCap       int64 `json:"per_episode_work_cap"`
	AggregateWorkCap        int64 `json:"aggregate_work_cap"`
	PerEpisodeUnitCap       int64 `json:"per_episode_unit_cap"`
	AggregateUnitCap        int64 `json:"aggregate_unit_cap"`
	PerEpisodeCycleCap      int64 `json:"per_episode_cycle_cap"`
	AggregateCycleCap       int64 `json:"aggregate_cycle_cap"`
}

type AggregateMeter struct {
	Name     string    `json:"name"`
	Episodes int64     `json:"episodes"`
	Totals   Counter   `json:"totals"`
	Maxima   Counter   `json:"maxima"`
	Caps     MeterCaps `json:"caps"`
	Valid    bool      `json:"valid"`
}

func CapsFor(name, context string, n int64) (MeterCaps, error) {
	if n < 0 {
		return MeterCaps{}, errors.New("negative meter cardinality")
	}
	m := PreregisteredManifest()
	mul := func(a int64) int64 { return n * a }
	switch name {
	case "production":
		aggregateEvaluation := mul(int64(m.EpisodeHypothesisEvaluationCap))
		if context == "training" {
			aggregateEvaluation = int64(m.TrainingHypothesisEvaluationCap)
		} else if context != "evaluation" {
			return MeterCaps{}, errors.New("production context must be training or evaluation")
		}
		return MeterCaps{int64(m.EpisodeHypothesisEvaluationCap), aggregateEvaluation, 0, 0, int64(m.EpisodeSemanticWorkCap), mul(int64(m.EpisodeSemanticWorkCap)), int64(m.EpisodeAttributedUnitCap), mul(int64(m.EpisodeAttributedUnitCap)), int64(m.EpisodeEngineCycleCap), mul(int64(m.EpisodeEngineCycleCap))}, nil
	case "teacher":
		return MeterCaps{PerEpisodeEvaluationCap: int64(m.TeacherEvaluationCap), AggregateEvaluationCap: mul(int64(m.TeacherEvaluationCap))}, nil
	case "certificate-replay", "post-selection-replay":
		aggregateWork := m.CertificateReplaySemanticWorkCap
		if name == "post-selection-replay" {
			aggregateWork = m.PostSelectionReplaySemanticWorkCap
		}
		return MeterCaps{int64(m.EpisodeHypothesisEvaluationCap), mul(int64(m.EpisodeHypothesisEvaluationCap)), 0, 0, int64(m.EpisodeSemanticWorkCap), int64(aggregateWork), int64(m.EpisodeAttributedUnitCap), mul(int64(m.EpisodeAttributedUnitCap)), int64(m.EpisodeEngineCycleCap), mul(int64(m.EpisodeEngineCycleCap))}, nil
	case "oracle-audit":
		return MeterCaps{AggregateWorkCap: int64(m.OracleAuditWorkCap)}, nil
	case "dp":
		return MeterCaps{PerEpisodeStateCap: int64(m.DynamicStateCap), AggregateStateCap: mul(int64(m.DynamicStateCap)), PerEpisodeWorkCap: int64(m.DynamicWorkCap), AggregateWorkCap: mul(int64(m.DynamicWorkCap))}, nil
	case "controls":
		return MeterCaps{AggregateWorkCap: int64(m.ControlWorkCap), AggregateUnitCap: int64(m.ControlAttributedUnitCap)}, nil
	case "curriculum":
		return MeterCaps{AggregateWorkCap: int64(m.CurriculumSemanticWorkCap), AggregateUnitCap: int64(m.CurriculumAttributedUnitCap), AggregateCycleCap: int64(m.CurriculumEngineCycleCap)}, nil
	default:
		return MeterCaps{}, fmt.Errorf("invalid meter name %q", name)
	}
}

func capsPass(maxima, totals Counter, caps MeterCaps) bool {
	checks := [][2]int64{
		{maxima.SCMEvaluations, caps.PerEpisodeEvaluationCap}, {totals.SCMEvaluations, caps.AggregateEvaluationCap},
		{maxima.MemoStates, caps.PerEpisodeStateCap}, {totals.MemoStates, caps.AggregateStateCap},
		{maxima.TotalWork, caps.PerEpisodeWorkCap}, {totals.TotalWork, caps.AggregateWorkCap},
		{maxima.AttributedUnits, caps.PerEpisodeUnitCap}, {totals.AttributedUnits, caps.AggregateUnitCap},
		{maxima.EngineCycles, caps.PerEpisodeCycleCap}, {totals.EngineCycles, caps.AggregateCycleCap},
	}
	for _, check := range checks {
		if check[1] != 0 && check[0] > check[1] {
			return false
		}
	}
	return true
}

func NewAggregateMeter(name, context string, counters []Counter) (AggregateMeter, error) {
	if !slices.Contains(MeterNames, name) {
		return AggregateMeter{}, fmt.Errorf("invalid meter name %q", name)
	}
	var totals, maxima Counter
	for _, counter := range counters {
		if err := counter.Validate(); err != nil {
			return AggregateMeter{}, err
		}
		var err error
		totals, err = addCounter(totals, counter)
		if err != nil {
			return AggregateMeter{}, err
		}
		maxima = maxCounter(maxima, counter)
	}
	caps, err := CapsFor(name, context, int64(len(counters)))
	if err != nil {
		return AggregateMeter{}, err
	}
	return AggregateMeter{Name: name, Episodes: int64(len(counters)), Totals: totals, Maxima: maxima, Caps: caps, Valid: capsPass(maxima, totals, caps)}, nil
}

func VerifyAggregateMeter(meter AggregateMeter, context string, counters []Counter) error {
	want, err := NewAggregateMeter(meter.Name, context, counters)
	if err != nil {
		return err
	}
	if meter != want {
		return errors.New("aggregate meter does not equal source-item reconstruction")
	}
	return nil
}

// ExpectedCardinality encodes the accepted source-item ownership rules. A
// return value of -1 means the cardinality is variable but independently
// bounded (curriculum tasks).
func ExpectedCardinality(name, context string, panelSeedCount int) (int, error) {
	if panelSeedCount < 0 {
		return 0, errors.New("negative panel seed count")
	}
	switch context {
	case "training":
		switch name {
		case "production", "teacher", "oracle-audit", "certificate-replay", "post-selection-replay":
			return 480, nil
		case "dp":
			return 12, nil
		case "controls":
			return 18, nil
		case "curriculum":
			return -1, nil
		}
	case "evaluation":
		switch name {
		case "production", "teacher", "oracle-audit":
			return 7 * panelSeedCount, nil
		case "dp":
			return panelSeedCount, nil
		case "certificate-replay", "post-selection-replay":
			return 480, nil
		case "controls":
			return 18, nil
		case "curriculum":
			return 0, nil
		}
	}
	return 0, fmt.Errorf("no cardinality rule for meter=%q context=%q", name, context)
}

func VerifyAggregateCardinality(meter AggregateMeter, context string, panelSeedCount int) error {
	want, err := ExpectedCardinality(meter.Name, context, panelSeedCount)
	if err != nil {
		return err
	}
	if want == -1 {
		if meter.Episodes < 0 || meter.Episodes > 525 {
			return errors.New("curriculum task cardinality outside 0..525")
		}
		return nil
	}
	if meter.Episodes != int64(want) {
		return fmt.Errorf("meter %q episodes=%d, want %d", meter.Name, meter.Episodes, want)
	}
	return nil
}

func MeterArrayDigest(meters []AggregateMeter) (string, error) {
	if len(meters) != len(MeterNames) {
		return "", errors.New("aggregate meter array must contain eight meters")
	}
	for i, meter := range meters {
		if meter.Name != MeterNames[i] {
			return "", errors.New("aggregate meters are out of order")
		}
	}
	return Digest(MeterArrayDomain, meters)
}
