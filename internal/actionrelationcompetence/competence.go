// Package actionrelationcompetence owns the safe, pre-review semantic
// competence universe. It has no fixture seed, policy, learned artifact, or
// protected-panel authority.
package actionrelationcompetence

import (
	"bytes"
	"fmt"

	"github.com/chazu/nous/internal/actionrelationoracle"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

const MaximumSequences = 40320

type Report struct {
	Sequences int  `json:"sequences"`
	Steps     int  `json:"steps"`
	Passed    bool `json:"passed"`
}

// Run checks every permutation of one frozen eight-occurrence competence
// history against the independent oracle. The caller, not production
// semantics, owns enumeration.
func Run() (Report, error) {
	initial := actionrelations.State{Cells: []actionrelations.Cell{{Name: "c0", Value: 0}, {Name: "c1", Value: 1}, {Name: "c2", Value: 2}}}
	actions := []actionrelations.SemanticAction{
		{Kind: "set", XRole: "c0", N: 0},
		{Kind: "set", XRole: "c0", N: 3},
		{Kind: "set", XRole: "c1", N: 1},
		{Kind: "set", XRole: "c1", N: 2},
		{Kind: "set", XRole: "c2", N: 0},
		{Kind: "set", XRole: "c2", N: 3},
		{Kind: "emit", Symbol: "a"},
		{Kind: "emit", Symbol: "b"},
	}
	indices := []int{0, 1, 2, 3, 4, 5, 6, 7}
	report := Report{}
	var visit func(int) error
	visit = func(index int) error {
		if index < len(indices) {
			for candidate := index; candidate < len(indices); candidate++ {
				indices[index], indices[candidate] = indices[candidate], indices[index]
				if err := visit(index + 1); err != nil {
					return err
				}
				indices[index], indices[candidate] = indices[candidate], indices[index]
			}
			return nil
		}
		production := initial
		oracleJSON, _ := initial.CanonicalJSON()
		for _, actionIndex := range indices {
			action := actions[actionIndex]
			next, outcome, err := actionrelations.Apply(production, action)
			if err != nil || outcome != "applied" {
				return fmt.Errorf("production step %d: %s %w", report.Sequences, outcome, err)
			}
			actionJSON, _ := action.CanonicalJSON()
			oracle, err := actionrelationoracle.Apply(oracleJSON, actionJSON)
			if err != nil || !oracle.Applicable {
				return fmt.Errorf("oracle step %d: %w", report.Sequences, err)
			}
			productionJSON, _ := next.CanonicalJSON()
			if !bytes.Equal(productionJSON, oracle.State) {
				return fmt.Errorf("history disagreement sequence=%d action=%d", report.Sequences, actionIndex)
			}
			production, oracleJSON = next, oracle.State
			report.Steps++
		}
		report.Sequences++
		return nil
	}
	if err := visit(0); err != nil {
		return report, err
	}
	report.Passed = report.Sequences == MaximumSequences && report.Steps == MaximumSequences*len(actions)
	if !report.Passed {
		return report, fmt.Errorf("competence cardinality mismatch: %+v", report)
	}
	return report, nil
}
