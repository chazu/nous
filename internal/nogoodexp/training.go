// Package nogoodexp owns the constraint/nogood experiments. It may populate
// public fixture units and drive ordinary agenda tasks, but it never inserts a
// candidate, selected mask, proof, or learned artifact.
package nogoodexp

import (
	"fmt"
	"io"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
)

const TrainingTaskCap = 2000

type TrainingRun struct {
	Store       *unit.Store
	TasksPopped int
	Terminal    string
	Artifact    string
}

func RunTraining(domainsDir string) (TrainingRun, error) {
	training, err := nogoodfixture.Training()
	if err != nil {
		return TrainingRun{}, err
	}
	promotion, err := nogoodfixture.PromotionCases()
	if err != nil {
		return TrainingRun{}, err
	}
	store := unit.NewStore()
	previousDomainsDir := seed.DomainsDir
	seed.DomainsDir = domainsDir
	err = seed.LoadDomain(store, "nogoods")
	seed.DomainsDir = previousDomainsDir
	if err != nil {
		return TrainingRun{}, err
	}

	experiment := unit.New("NG.Training.Experiment")
	experiment.Set("isA", []string{"NogoodLearningExperiment", "Anything"})
	var exampleNames []string
	for _, task := range training {
		name := fmt.Sprintf("NG.Training.Example.%d", task.Ordinal)
		example := unit.New(name)
		example.Set("isA", []string{"NogoodTrainingExample", "Anything"})
		example.Set("problem", string(task.ProblemJSON))
		example.Set("decisionVariable", task.Decision.Variable)
		example.Set("decisionColor", task.Decision.Color)
		store.Put(example)
		exampleNames = append(exampleNames, name)
	}
	var promotionNames []string
	for _, testCase := range promotion {
		name := fmt.Sprintf("NG.Promotion.Case.%02d", testCase.Ordinal)
		promotionUnit := unit.New(name)
		promotionUnit.Set("isA", []string{"NogoodPromotionCase", "Anything"})
		promotionUnit.Set("problem", string(testCase.ProblemJSON))
		promotionUnit.Set("decisionVariable", testCase.Decision.Variable)
		promotionUnit.Set("decisionColor", testCase.Decision.Color)
		promotionUnit.Set("anchor", testCase.Binding.Anchor)
		promotionUnit.Set("x", testCase.Binding.X)
		promotionUnit.Set("y", testCase.Binding.Y)
		promotionUnit.Set("blocked", testCase.Binding.Blocked)
		promotionUnit.Set("escape", testCase.Binding.Escape)
		promotionUnit.Set("only", testCase.Binding.Only)
		promotionUnit.Set("xColor", testCase.Completion.XColor)
		promotionUnit.Set("yColor", testCase.Completion.YColor)
		store.Put(promotionUnit)
		promotionNames = append(promotionNames, name)
	}
	experiment.Set("trainingExamples", exampleNames)
	experiment.Set("promotionCases", promotionNames)
	store.Put(experiment)

	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return TrainingRun{}, fmt.Errorf("initialize nogood VM: %w", err)
	}
	ag.Push(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: "ngStart", Reasons: []string{"Begin bounded nogood acquisition"}})
	popped := 0
	for ag.Len() > 0 {
		if popped >= TrainingTaskCap {
			experiment.Set("terminal", "budget-exhausted")
			return TrainingRun{Store: store, TasksPopped: popped, Terminal: "budget-exhausted"}, nil
		}
		task := ag.Pop()
		if task == nil {
			return TrainingRun{}, fmt.Errorf("agenda length was nonzero but Pop returned nil")
		}
		eng.WorkOnTask(task)
		if len(eng.VM.DeletedUnits) != 0 {
			return TrainingRun{}, fmt.Errorf("nogood heuristic deleted units")
		}
		popped++
	}
	terminal := experiment.GetString("terminal")
	artifact := experiment.GetString("artifactUnit")
	if terminal == "" {
		return TrainingRun{}, fmt.Errorf("training ended without a terminal")
	}
	return TrainingRun{Store: store, TasksPopped: popped, Terminal: terminal, Artifact: artifact}, nil
}
