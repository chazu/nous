package transformexp

import (
	"fmt"
	"io"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/transformfixturecore"
	"github.com/chazu/nous/internal/unit"
)

type acquisitionRun struct {
	Store        *unit.Store
	Terminal     string
	Programs     []string
	Candidates   []string
	Root         string
	Edges        []string
	Artifact     string
	TasksPopped  int
	MeterRecords []dsl.TransformMeterRecord
}

func runAcquisition(domainsDir string, trainingBytes []byte, token string) (acquisitionRun, error) {
	return runAcquisitionConfigured(domainsDir, trainingBytes, token, nil)
}

func runAcquisitionConfigured(domainsDir string, trainingBytes []byte, token string, configure func(*unit.Store)) (acquisitionRun, error) {
	training, err := transformfixturecore.ParseTraining(trainingBytes)
	if err != nil {
		return acquisitionRun{}, err
	}
	store := unit.NewStore()
	previous := seed.DomainsDir
	seed.DomainsDir = domainsDir
	err = seed.LoadDomain(store, "transformschema")
	seed.DomainsDir = previous
	if err != nil {
		return acquisitionRun{}, err
	}
	if configure != nil {
		configure(store)
	}
	experiment := unit.New("TS.Experiment." + token)
	experiment.Set("isA", []string{"TransformLearningExperiment", "Anything"})
	meterToken := "tsm:" + token
	if err := dsl.RegisterTransformMeter(meterToken); err != nil {
		return acquisitionRun{}, err
	}
	defer dsl.UnregisterTransformMeter(meterToken)
	experiment.Set("meterToken", meterToken)
	store.Put(experiment)
	for _, c := range training.Cases {
		name := fmt.Sprintf("TS.Example.%s.%s", token, c.Token)
		u := unit.New(name)
		u.Set("isA", []string{"TransformTrainingCase", "Anything"})
		u.Set("experiment", experiment.Name)
		u.Set("token", c.Token)
		u.Set("kind", c.Kind)
		u.Set("before", string(c.Before))
		u.Set("after", string(c.After))
		store.Put(u)
	}
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MutConfig.Enabled = false
	eng.MaxCycles = 2000
	if err := eng.VM.InitError(); err != nil {
		return acquisitionRun{}, err
	}
	ag.Push(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: "tsAcquire", Reasons: []string{"Begin transformation acquisition"}})
	popped := 0
	for ag.Len() > 0 {
		if popped >= 2000 {
			return acquisitionRun{}, fmt.Errorf("task cap")
		}
		task := ag.Pop()
		eng.WorkOnTask(task)
		if eng.LastError != nil {
			return acquisitionRun{}, fmt.Errorf("%s.%s: %w", task.UnitName, task.SlotName, eng.LastError)
		}
		if len(eng.VM.DeletedUnits) != 0 {
			return acquisitionRun{}, fmt.Errorf("deleted units")
		}
		popped++
	}
	records, err := dsl.TransformMeterSnapshot(meterToken)
	if err != nil {
		return acquisitionRun{}, err
	}
	return acquisitionRun{
		Store:        store,
		Terminal:     experiment.GetString("terminal"),
		Programs:     experiment.GetStrings("programUnits"),
		Candidates:   experiment.GetStrings("candidateUnits"),
		Root:         experiment.GetString("rootCandidate"),
		Edges:        experiment.GetStrings("edgeUnits"),
		Artifact:     experiment.GetString("artifactUnit"),
		TasksPopped:  popped,
		MeterRecords: records,
	}, nil
}
