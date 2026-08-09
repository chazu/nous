// Package nogoodexp owns the constraint/nogood experiments. It may populate
// public fixture units and drive ordinary agenda tasks, but it never inserts a
// candidate, selected mask, proof, or learned artifact.
package nogoodexp

import (
	"fmt"
	"io"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
)

const TrainingTaskCap = 2000

type TrainingRun struct {
	Store        *unit.Store
	TasksPopped  int
	Terminal     string
	Artifact     string
	MeterRecords []dsl.NogoodMeterRecord
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
	meterToken := "ngm:training:part3-nogoods-v1"
	if err := dsl.RegisterNogoodMeter(meterToken); err != nil {
		return TrainingRun{}, err
	}
	defer dsl.UnregisterNogoodMeter(meterToken)
	experiment.Set("meterToken", meterToken)
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
	if err := dsl.ChargeNogoodMeter(meterToken, "agenda-enqueue", experiment.Name, "ngStart", "ok", 12); err != nil {
		return TrainingRun{}, err
	}
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
		if err := dsl.ChargeNogoodMeter(meterToken, "agenda-dequeue", task.UnitName, task.SlotName, "ok", 12); err != nil {
			return TrainingRun{}, err
		}
		eng.WorkOnTask(task)
		if eng.LastError != nil {
			return TrainingRun{}, fmt.Errorf("nogood heuristic execution on %s.%s: %w", task.UnitName, task.SlotName, eng.LastError)
		}
		if len(eng.VM.DeletedUnits) != 0 {
			return TrainingRun{}, fmt.Errorf("nogood heuristic deleted units")
		}
		if err := chargeMeterOperations(meterToken, 12, task.UnitName+"."+task.SlotName, engineDispatchOperations); err != nil {
			return TrainingRun{}, err
		}
		popped++
	}
	terminal := experiment.GetString("terminal")
	artifact := experiment.GetString("artifactUnit")
	if terminal == "" {
		return TrainingRun{}, fmt.Errorf("training ended without a terminal")
	}
	records, err := dsl.NogoodMeterSnapshot(meterToken)
	if err != nil {
		return TrainingRun{}, err
	}
	if store.Count() > 200000 {
		return TrainingRun{}, fmt.Errorf("training store has %d attributed units", store.Count())
	}
	return TrainingRun{Store: store, TasksPopped: popped, Terminal: terminal, Artifact: artifact, MeterRecords: records}, nil
}
