// Package causalrun implements the hidden-free online execution boundary for
// active causal diagnosis v2.
package causalrun

import (
	"errors"
	"fmt"
)

const (
	EpisodeEvaluationCap = 4096
	EpisodeWorkCap       = 8192
	EpisodeCycleCap      = 5000
	EpisodeUnitCap       = 1000
	DynamicStateCap      = 531441
	DynamicWorkCap       = 4000000
)

// Counts is the exact causal-work/v2 counter order. EngineCycles and
// AttributedUnits are independent resource counters and are deliberately not
// included in TotalWork.
type Counts struct {
	SCMEvaluations           int `json:"scm_evaluations"`
	PartitionAssignments     int `json:"partition_assignments"`
	CellAccumulations        int `json:"cell_accumulations"`
	RuleComparisons          int `json:"rule_comparisons"`
	PosteriorChecks          int `json:"posterior_checks"`
	ArtifactMaterializations int `json:"artifact_materializations"`
	TranscriptFields         int `json:"transcript_fields"`
	ProfileFields            int `json:"profile_fields"`
	MemoStates               int `json:"memo_states"`
	MemoLookups              int `json:"memo_lookups"`
	QEvaluations             int `json:"q_evaluations"`
	TableLookups             int `json:"table_lookups"`
	EngineCycles             int `json:"engine_cycles"`
	AttributedUnits          int `json:"attributed_units"`
	TotalWork                int `json:"total_work"`
}

// Array returns the normative compact positional representation.
func (c Counts) Array() [15]int {
	return [15]int{
		c.SCMEvaluations, c.PartitionAssignments, c.CellAccumulations,
		c.RuleComparisons, c.PosteriorChecks, c.ArtifactMaterializations,
		c.TranscriptFields, c.ProfileFields, c.MemoStates, c.MemoLookups,
		c.QEvaluations, c.TableLookups, c.EngineCycles, c.AttributedUnits,
		c.TotalWork,
	}
}

func (c *Counts) recompute() {
	c.TotalWork = c.SCMEvaluations + c.PartitionAssignments +
		c.CellAccumulations + c.RuleComparisons + c.PosteriorChecks +
		c.ArtifactMaterializations + c.TranscriptFields + c.ProfileFields +
		c.MemoStates + c.MemoLookups + c.QEvaluations + c.TableLookups
}

func (c Counts) ValidateEquation() error {
	want := c
	want.recompute()
	if c.TotalWork != want.TotalWork {
		return fmt.Errorf("causal work total=%d, want %d", c.TotalWork, want.TotalWork)
	}
	for i, n := range c.Array() {
		if n < 0 {
			return fmt.Errorf("causal work counter %d is negative", i)
		}
	}
	return nil
}

// WorkMeter is runner-owned. It is never stored in a Unit or exposed to a CUE
// program as a mutable slot.
type WorkMeter struct {
	counts  Counts
	dynamic bool
}

func (m *WorkMeter) Counts() Counts { return m.counts }

func (m *WorkMeter) add(field *int, n int) error {
	if n < 0 {
		return errors.New("negative causal work charge")
	}
	*field += n
	m.counts.recompute()
	return m.checkProductionCaps()
}

func (m *WorkMeter) checkProductionCaps() error {
	if m.dynamic {
		if m.counts.MemoStates > DynamicStateCap {
			return fmt.Errorf("dynamic state cap exceeded: %d", m.counts.MemoStates)
		}
		if m.counts.TotalWork > DynamicWorkCap {
			return fmt.Errorf("dynamic work cap exceeded: %d", m.counts.TotalWork)
		}
		return nil
	}
	if m.counts.SCMEvaluations > EpisodeEvaluationCap {
		return fmt.Errorf("SCM evaluation cap exceeded: %d", m.counts.SCMEvaluations)
	}
	if m.counts.TotalWork > EpisodeWorkCap {
		return fmt.Errorf("semantic work cap exceeded: %d", m.counts.TotalWork)
	}
	if m.counts.EngineCycles > EpisodeCycleCap {
		return fmt.Errorf("engine cycle cap exceeded: %d", m.counts.EngineCycles)
	}
	if m.counts.AttributedUnits > EpisodeUnitCap {
		return fmt.Errorf("attributed unit cap exceeded: %d", m.counts.AttributedUnits)
	}
	return nil
}

func (m *WorkMeter) chargeSCM(n int) error        { return m.add(&m.counts.SCMEvaluations, n) }
func (m *WorkMeter) chargeAssignment(n int) error { return m.add(&m.counts.PartitionAssignments, n) }
func (m *WorkMeter) chargeCell(n int) error       { return m.add(&m.counts.CellAccumulations, n) }
func (m *WorkMeter) chargeComparison(n int) error { return m.add(&m.counts.RuleComparisons, n) }
func (m *WorkMeter) chargePosterior(n int) error  { return m.add(&m.counts.PosteriorChecks, n) }
func (m *WorkMeter) chargeArtifact(n int) error   { return m.add(&m.counts.ArtifactMaterializations, n) }
func (m *WorkMeter) chargeTranscript(n int) error { return m.add(&m.counts.TranscriptFields, n) }
func (m *WorkMeter) chargeProfile(n int) error    { return m.add(&m.counts.ProfileFields, n) }
func (m *WorkMeter) chargeMemoState(n int) error  { return m.add(&m.counts.MemoStates, n) }
func (m *WorkMeter) chargeMemoLookup(n int) error { return m.add(&m.counts.MemoLookups, n) }
func (m *WorkMeter) chargeQ(n int) error          { return m.add(&m.counts.QEvaluations, n) }
func (m *WorkMeter) chargeTable(n int) error      { return m.add(&m.counts.TableLookups, n) }
func (m *WorkMeter) chargeCycle(n int) error      { return m.add(&m.counts.EngineCycles, n) }
func (m *WorkMeter) chargeUnit(n int) error       { return m.add(&m.counts.AttributedUnits, n) }

func (m *WorkMeter) remaining() (evaluations, work, cycles, units int) {
	return EpisodeEvaluationCap - m.counts.SCMEvaluations,
		EpisodeWorkCap - m.counts.TotalWork,
		EpisodeCycleCap - m.counts.EngineCycles,
		EpisodeUnitCap - m.counts.AttributedUnits
}
