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
	"github.com/chazu/nous/internal/actionrelationwire"
	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
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
	MeterRecords     []dsl.ActionRelationMeterRecord
}

type EvidenceRoots struct {
	CandidateLeaves      []string
	EdgeTableRoot        string
	EvaluationTableRoots []string
	WinnerLeaves         []string
}

type Session struct {
	Store      *unit.Store
	Experiment string
	engine     *engine.Engine
	meterToken string
	closed     bool
}

func Begin(domainsDir, token string) (*Session, error) {
	training, err := actionrelationfixturecore.Training()
	if err != nil {
		return nil, err
	}
	store := unit.NewStore()
	previous := seed.DomainsDir
	seed.DomainsDir = domainsDir
	err = seed.LoadDomain(store, "actionrelations")
	seed.DomainsDir = previous
	if err != nil {
		return nil, err
	}
	if err := installSemanticInputs(store, training); err != nil {
		return nil, err
	}
	experiment := unit.New("AR.Experiment." + token)
	experiment.Set("isA", []string{"ActionRelationExperiment", "Anything"})
	experiment.Set("expectedObservationCount", len(training))
	meterToken := "arm:" + token
	if err := dsl.RegisterActionRelationMeter(meterToken); err != nil {
		return nil, err
	}
	fail := func(err error) (*Session, error) {
		dsl.UnregisterActionRelationMeter(meterToken)
		return nil, err
	}
	experiment.Set("meterToken", meterToken)
	pattern := actionrelations.Pattern{Kinds: []string{"add", "add"}, Roles: []int{0, -1, 1, -1}}
	patternJSON, _ := pattern.CanonicalJSON()
	experiment.Set("pattern", string(patternJSON))
	experiment.Set("patternUnit", putCanonical(store, "ActionRelationPattern", patternJSON))
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
		return fail(err)
	}
	for _, slot := range []string{"arObserve", "arAllocate", "arEvaluate"} {
		eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: slot})
		if eng.LastError != nil {
			return fail(fmt.Errorf("%s: %w", slot, eng.LastError))
		}
	}
	if err := installPresentationViews(store, experiment, training); err != nil {
		return fail(err)
	}
	eng.WorkOnTask(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: "arFinalize"})
	if eng.LastError != nil {
		return fail(fmt.Errorf("arFinalize: %w", eng.LastError))
	}
	session := &Session{Store: store, Experiment: experiment.Name, engine: eng, meterToken: meterToken}
	run, err := session.Snapshot()
	if err != nil {
		return fail(err)
	}
	if run.Observations != 16 || run.Candidates != 451 || run.Edges != 450 || run.LiteralRows != 13920 || run.GuardResults != 7216 || run.CandidateResults != 451 || run.Winners < 1 || experiment.GetBool("candidatesFinalized") != true {
		session.Abort()
		return nil, fmt.Errorf("partial acquisition cardinality mismatch: %+v", run)
	}
	return session, nil
}

func (s *Session) Snapshot() (Run, error) {
	if s == nil || s.closed || s.Store == nil {
		return Run{}, fmt.Errorf("closed acquisition session")
	}
	experiment := s.Store.Get(s.Experiment)
	if experiment == nil {
		return Run{}, fmt.Errorf("missing acquisition experiment")
	}
	meterRecords, err := dsl.ActionRelationMeterSnapshot(s.meterToken)
	if err != nil {
		return Run{}, err
	}
	return Run{
		Store: s.Store, Experiment: s.Experiment,
		Observations: len(experiment.GetStrings("observationUnits")), Candidates: len(experiment.GetStrings("candidateUnits")),
		Edges: len(experiment.GetStrings("edgeUnits")), LiteralRows: len(experiment.GetStrings("literalRowUnits")), GuardResults: len(experiment.GetStrings("guardResultUnits")),
		CandidateResults: len(experiment.GetStrings("candidateResultUnits")), Winners: len(experiment.GetStrings("winnerResultUnits")), Artifact: experiment.GetString("artifactUnit"),
		MeterRecords: meterRecords,
	}, nil
}

func (s *Session) BindEvidence(roots EvidenceRoots) (Run, error) {
	if s == nil || s.closed || len(roots.CandidateLeaves) != 451 || len(roots.EvaluationTableRoots) != 2 {
		return Run{}, fmt.Errorf("invalid evidence-bound acquisition session")
	}
	experiment := s.Store.Get(s.Experiment)
	if experiment == nil || len(roots.WinnerLeaves) != len(experiment.GetStrings("winnerResultUnits")) {
		return Run{}, fmt.Errorf("winner evidence mismatch")
	}
	for _, digest := range append(append(append([]string{}, roots.CandidateLeaves...), roots.EvaluationTableRoots...), append([]string{roots.EdgeTableRoot}, roots.WinnerLeaves...)...) {
		if !actionrelationsDigest(digest) {
			return Run{}, fmt.Errorf("invalid evidence root digest")
		}
	}
	experiment.Set("candidateLeafDigests", roots.CandidateLeaves)
	experiment.Set("edgeTableRoot", roots.EdgeTableRoot)
	experiment.Set("evaluationTableRoots", roots.EvaluationTableRoots)
	experiment.Set("winnerLeafDigests", roots.WinnerLeaves)
	experiment.Set("evidenceRootsReady", true)
	s.engine.WorkOnTask(&agenda.Task{Priority: 900, UnitName: experiment.Name, SlotName: "arClose"})
	if s.engine.LastError != nil {
		s.Abort()
		return Run{}, fmt.Errorf("arClose: %w", s.engine.LastError)
	}
	run, err := s.Snapshot()
	if err != nil {
		s.Abort()
		return Run{}, err
	}
	s.closed = true
	dsl.UnregisterActionRelationMeter(s.meterToken)
	if run.Artifact == "" || experiment.GetString("terminal") != "completed" || !experiment.GetBool("guardSearchClosed") {
		return run, fmt.Errorf("evidence-bound acquisition did not close")
	}
	return run, nil
}

func (s *Session) Abort() {
	if s != nil && !s.closed {
		s.closed = true
		dsl.UnregisterActionRelationMeter(s.meterToken)
	}
}

// Execute is a development helper for semantic/search tests. Protected
// experiment execution uses Begin, builds the physical ARTB manifests, and
// supplies their exact roots through BindEvidence.
func Execute(domainsDir, token string) (Run, error) {
	session, err := Begin(domainsDir, token)
	if err != nil {
		return Run{}, err
	}
	run, err := session.Snapshot()
	if err != nil {
		session.Abort()
		return Run{}, err
	}
	experiment := run.Store.Get(run.Experiment)
	candidateLeaves := developmentDigests("candidate", len(experiment.GetStrings("candidateUnits")))
	winnerLeaves := developmentDigests("winner", len(experiment.GetStrings("winnerResultUnits")))
	for index, name := range experiment.GetStrings("candidateUnits") {
		run.Store.Get(name).Set("tableLeafDigest", candidateLeaves[index])
	}
	for index, name := range experiment.GetStrings("winnerResultUnits") {
		run.Store.Get(name).Set("tableLeafDigest", winnerLeaves[index])
	}
	return session.BindEvidence(EvidenceRoots{
		CandidateLeaves: candidateLeaves, EdgeTableRoot: developmentDigests("edge-table", 1)[0],
		EvaluationTableRoots: developmentDigests("evaluation-table", 2), WinnerLeaves: winnerLeaves,
	})
}

func developmentDigests(label string, count int) []string {
	result := make([]string, count)
	for index := range result {
		wire, _ := json.Marshal([]any{"actionrelation-development-only-root/v1", label, index})
		digest := sha256.Sum256(wire)
		result[index] = hex.EncodeToString(digest[:])
	}
	return result
}

func actionrelationsDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func installPresentationViews(store *unit.Store, experiment *unit.Unit, training []actionrelationfixturecore.Case) error {
	observations := experiment.GetStrings("observationUnits")
	if len(observations) != len(training) {
		return fmt.Errorf("view observation count mismatch")
	}
	observationDigests := make([]string, len(observations))
	for index, name := range observations {
		observation := store.Get(name)
		if observation == nil {
			return fmt.Errorf("missing observation %d", index)
		}
		observationDigests[index] = observation.GetString("objectDigest")
	}
	semanticTrainingRoot, err := actionrelationwire.RootDigest("semantic-training", observationDigests)
	if err != nil {
		return err
	}
	experiment.Set("semanticTrainingRoot", semanticTrainingRoot)
	var names, presentationNames, proofNames []string
	var rootRows []any
	for index, testCase := range training {
		views, err := actionrelationfixturecore.Views(testCase)
		if err != nil {
			return err
		}
		observation := store.Get(observations[index])
		if observation == nil {
			return fmt.Errorf("missing observation %d", index)
		}
		for _, view := range views {
			viewName := fmt.Sprintf("AR.Presentation.%s.%02d.%d", experiment.Name, index, view.Bank)
			viewUnit := unit.New(viewName)
			viewUnit.Set("isA", []string{"ActionPresentationView", "Anything"})
			viewUnit.Set("canonicalObject", string(view.Canonical))
			viewUnit.Set("objectDigest", view.Digest)
			store.Put(viewUnit)
			presentationNames = append(presentationNames, viewName)
			proofName := fmt.Sprintf("AR.NormalizationProof.%s.%02d.%d", experiment.Name, index, view.Bank)
			proofUnit := unit.New(proofName)
			proofUnit.Set("isA", []string{"ActionNormalizationProof", "Anything"})
			proofUnit.Set("canonicalObject", string(view.Proof))
			proofUnit.Set("objectDigest", view.ProofDigest)
			store.Put(proofUnit)
			proofNames = append(proofNames, proofName)
			wire, _ := json.Marshal([]any{"action-view-evidence/v1", observation.GetString("objectDigest"), view.Digest, view.ProofDigest})
			digestBytes := sha256.Sum256(wire)
			digest := hex.EncodeToString(digestBytes[:])
			name := fmt.Sprintf("AR.View.%s.%02d.%d", experiment.Name, index, view.Bank)
			u := unit.New(name)
			u.Set("isA", []string{"ActionPresentationViewEvidence", "Anything"})
			u.Set("canonicalObject", string(wire))
			u.Set("objectDigest", digest)
			u.Set("observationDigest", observation.GetString("objectDigest"))
			u.Set("viewDigest", view.Digest)
			u.Set("normalizationProofDigest", view.ProofDigest)
			u.Set("semanticWorldDigest", view.SemanticWorldDigest)
			u.Set("originalStateDigest", view.OriginalStateDigest)
			u.Set("originalActionsRoot", view.OriginalActionsRoot)
			u.Set("occurrenceMapRoot", view.OccurrenceMapRoot)
			u.Set("bank", view.Bank)
			u.Set("cellCount", view.CellCount)
			u.Set("actionCount", view.ActionCount)
			store.Put(u)
			names = append(names, name)
			rootRows = append(rootRows, []any{observation.GetString("objectDigest"), view.Bank, digest})
		}
	}
	rootDigest, err := actionrelationwire.RootDigest("view-evidence", rootRows)
	if err != nil {
		return err
	}
	experiment.Set("viewEvidenceUnits", names)
	experiment.Set("presentationViewUnits", presentationNames)
	experiment.Set("normalizationProofUnits", proofNames)
	experiment.Set("viewEvidenceRoot", rootDigest)
	trainingWire, _ := json.Marshal([]any{"action-training-evidence/v1", semanticTrainingRoot, rootDigest})
	experiment.Set("trainingEvidenceUnit", putCanonical(store, "ActionTrainingEvidence", trainingWire))
	return nil
}

func installSemanticInputs(store *unit.Store, training []actionrelationfixturecore.Case) error {
	for _, testCase := range training {
		state, err := actionrelations.ParseState(testCase.State)
		if err != nil {
			return err
		}
		a, err := actionrelations.ParseOccurrence(testCase.AOccurrence)
		if err != nil {
			return err
		}
		b, err := actionrelations.ParseOccurrence(testCase.BOccurrence)
		if err != nil {
			return err
		}
		putCanonical(store, "FiniteActionState", testCase.State)
		for _, occurrence := range []actionrelations.Occurrence{a, b} {
			actionJSON, _ := occurrence.Action.CanonicalJSON()
			occurrenceJSON, _ := occurrence.CanonicalJSON()
			putCanonical(store, "FiniteSemanticAction", actionJSON)
			putCanonical(store, "ActionOccurrence", occurrenceJSON)
		}
		world := actionrelations.NormalizedWorld{State: state, Actions: []actionrelations.SemanticAction{a.Action, b.Action}}
		worldJSON, err := world.CanonicalJSON()
		if err != nil {
			return err
		}
		putCanonical(store, "FiniteActionWorldCore", worldJSON)
	}
	return nil
}

func putCanonical(store *unit.Store, category string, canonical []byte) string {
	digestBytes := sha256.Sum256(canonical)
	digest := hex.EncodeToString(digestBytes[:])
	name := "AR.Object." + category + "." + digest
	if store.Get(name) == nil {
		u := unit.New(name)
		u.Set("isA", []string{category, "Anything"})
		u.Set("canonicalObject", string(canonical))
		u.Set("objectDigest", digest)
		store.Put(u)
	}
	return name
}
