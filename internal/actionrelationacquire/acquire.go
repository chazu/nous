// Package actionrelationacquire drives ordinary CUE acquisition tasks. It may
// populate public fixture units, but it never inserts a candidate, observation,
// guard result, relation, or learned artifact itself.
package actionrelationacquire

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/chazu/nous/internal/actionrelationfixturecore"
	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type Run struct {
	Store            *unit.Store
	Experiment       string
	Observations     int
	Candidates       int
	Edges            int
	LiteralRows      int
	GuardResults     int
	CandidateResults int
	Winners          int
	Artifact         string
}

func Execute(domainsDir, token string) (Run, error) {
	training, err := actionrelationfixturecore.Training()
	if err != nil {
		return Run{}, err
	}
	store := unit.NewStore()
	previous := seed.DomainsDir
	seed.DomainsDir = domainsDir
	err = seed.LoadDomain(store, "actionrelations")
	seed.DomainsDir = previous
	if err != nil {
		return Run{}, err
	}
	experiment := unit.New("AR.Experiment." + token)
	experiment.Set("isA", []string{"ActionRelationExperiment", "Anything"})
	experiment.Set("expectedObservationCount", len(training))
	pattern := actionrelations.Pattern{Kinds: []string{"add", "add"}, Roles: []int{0, -1, 1, -1}}
	patternJSON, _ := pattern.CanonicalJSON()
	experiment.Set("pattern", string(patternJSON))
	trainingBytes, _ := json.Marshal(training)
	trainingDigest := sha256.Sum256(trainingBytes)
	experiment.Set("semanticTrainingRoot", hex.EncodeToString(trainingDigest[:]))
	viewDigest := sha256.Sum256(append([]byte("actionrelation-view-evidence/v1\x00"), trainingBytes...))
	experiment.Set("viewEvidenceRoot", hex.EncodeToString(viewDigest[:]))
	store.Put(experiment)
	for _, testCase := range training {
		name := fmt.Sprintf("AR.Training.%s.%02d", token, testCase.Ordinal)
		u := unit.New(name)
		u.Set("isA", []string{"ActionRelationTrainingCase", "Anything"})
		u.Set("experiment", experiment.Name)
		u.Set("ordinal", testCase.Ordinal)
		u.Set("state", string(testCase.State))
		u.Set("aOccurrence", string(testCase.AOccurrence))
		u.Set("bOccurrence", string(testCase.BOccurrence))
		u.Set("label", testCase.Label)
		store.Put(u)
	}
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.VM.InitError(); err != nil {
		return Run{}, err
	}
	for _, slot := range []string{"arObserve", "arAllocate", "arEvaluate", "arFinalize"} {
		eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: slot})
		if eng.LastError != nil {
			return Run{}, fmt.Errorf("%s: %w", slot, eng.LastError)
		}
	}
	run := Run{
		Store: store, Experiment: experiment.Name,
		Observations: len(experiment.GetStrings("observationUnits")), Candidates: len(experiment.GetStrings("candidateUnits")),
		Edges: len(experiment.GetStrings("edgeUnits")), LiteralRows: len(experiment.GetStrings("literalRowUnits")), GuardResults: len(experiment.GetStrings("guardResultUnits")),
		CandidateResults: len(experiment.GetStrings("candidateResultUnits")), Winners: len(experiment.GetStrings("winnerResultUnits")), Artifact: experiment.GetString("artifactUnit"),
	}
	if run.Observations != 16 || run.Candidates != 451 || run.Edges != 450 || run.LiteralRows != 13920 || run.GuardResults != 7216 || run.CandidateResults != 451 || run.Winners < 1 || run.Artifact == "" || experiment.GetString("terminal") != "completed" {
		return run, fmt.Errorf("acquisition cardinality mismatch: %+v", run)
	}
	return run, nil
}
