package actionrelationfixture

import (
	"errors"
	"fmt"
)

var (
	ErrGeneratorWorkCap   = errors.New("generator attempt work cap exhausted")
	ErrGeneratorPredicate = errors.New("generator acceptance predicate failed")
)

type AttemptMeter struct {
	context DrawContext
	draws   DrawBlock
	phases  []GeneratorPhase
	total   int
	cap     int
	failed  bool
	closed  bool
}

func BeginAttemptMeter(context DrawContext) (*AttemptMeter, error) {
	draws, err := PrecommitDraws(context)
	if err != nil {
		return nil, err
	}
	return &AttemptMeter{
		context: context,
		draws:   draws,
		phases: []GeneratorPhase{{
			Name: "draw-precommit", StartWork: 0, EndWork: 66,
			Predicate: "exact-66-draws", Status: "passed",
		}},
		total: 66,
		cap:   GeneratorAttemptWorkCap,
	}, nil
}

// RunPhase supplies a reservation function that must be called immediately
// before each charged semantic event. The meter adds the named phase predicate
// only after the work callback returns, then closes the attempt on any failure.
func (m *AttemptMeter) RunPhase(work func(reserve func() error) (bool, error)) error {
	if m == nil || m.closed || m.failed || work == nil || len(m.phases) >= len(generatorPhaseVocabulary) {
		return fmt.Errorf("generator phase is not available")
	}
	vocabulary := generatorPhaseVocabulary[len(m.phases)]
	start := m.total
	active := true
	reserve := func() error {
		if !active || m.closed || m.failed {
			return fmt.Errorf("generator reservation escaped its phase")
		}
		if m.total >= m.cap {
			return ErrGeneratorWorkCap
		}
		m.total++
		return nil
	}
	passed, workErr := work(reserve)
	active = false
	status := "failed"
	if !errors.Is(workErr, ErrGeneratorWorkCap) {
		if m.total >= m.cap {
			workErr = ErrGeneratorWorkCap
		} else {
			// Every phase ends with its named acceptance/bound predicate.
			m.total++
			if workErr == nil && passed {
				status = "passed"
			}
		}
	}
	m.phases = append(m.phases, GeneratorPhase{
		Name: vocabulary.name, StartWork: start, EndWork: m.total,
		Predicate: vocabulary.predicate, Status: status,
	})
	if status == "failed" {
		m.failed = true
		if workErr != nil {
			return workErr
		}
		return ErrGeneratorPredicate
	}
	return nil
}

func (m *AttemptMeter) Draws() DrawBlock {
	if m == nil {
		return DrawBlock{}
	}
	return m.draws
}

func (m *AttemptMeter) TotalWork() int {
	if m == nil {
		return 0
	}
	return m.total
}

func (m *AttemptMeter) Close() (AttemptLedger, error) {
	if m == nil || m.closed || !m.failed && len(m.phases) != len(generatorPhaseVocabulary) {
		return AttemptLedger{}, fmt.Errorf("generator attempt has not reached a terminal")
	}
	m.closed = true
	terminal := "rejected"
	if !m.failed {
		terminal = "accepted"
	}
	return SealAttemptLedger(m.context, m.draws, m.phases, terminal)
}
