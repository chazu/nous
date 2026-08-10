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
	"github.com/chazu/nous/internal/actionrelationledger"
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
	Scope      string
	RunID      string
	engine     *engine.Engine
	meterToken string
	closed     bool
}

func Begin(domainsDir, token string) (*Session, error) {
	return BeginFamilyScope(domainsDir, token, 0, "nous")
}

func BeginFamily(domainsDir, token string, family int) (*Session, error) {
	return BeginFamilyScope(domainsDir, token, family, "nous")
}

func BeginFor(domainsDir, token string, family, curriculum int, panel, authority string) (*Session, error) {
	return BeginFamilyScopeFor(domainsDir, token, family, "nous", panel, authority, curriculum)
}

func BeginNoGuard(domainsDir, token string, family int) (*Session, error) {
	return BeginFamilyScope(domainsDir, token, family, "no-guard")
}

func BeginNoGuardFor(domainsDir, token string, family, curriculum int, panel, authority string) (*Session, error) {
	return BeginFamilyScopeFor(domainsDir, token, family, "no-guard", panel, authority, curriculum)
}

func BeginFamilyScope(domainsDir, token string, family int, scope string) (*Session, error) {
	return BeginFamilyScopeFor(domainsDir, token, family, scope, "development", acceptedPlanCommit, 0)
}

func BeginFamilyScopeFor(domainsDir, token string, family int, scope, panel, authority string, curriculum int) (*Session, error) {
	if scope != "nous" && scope != "no-guard" {
		return nil, fmt.Errorf("invalid acquisition scope")
	}
	training, err := actionrelationfixturecore.TrainingFamily(family)
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
	runID, err := actionrelationledger.AcquisitionRunID(panel, authority, curriculum, scope)
	if err != nil {
		return nil, err
	}
	experiment := unit.New("AR.Experiment." + token)
	experiment.Set("isA", []string{"ActionRelationExperiment", "Anything"})
	experiment.Set("expectedObservationCount", len(training))
	meterToken := "arm:" + token
	operationCodes := acquisitionOperationSchedule(training, scope)
	reservationNames := make([]string, len(operationCodes))
	meterPlan := make([]dsl.ActionRelationMeterPlanEntry, len(operationCodes))
	for sequence, code := range operationCodes {
		taskDigest := actionrelationledger.TaskDigest(runID, sequence, code)
		reservation, err := actionrelationledger.BuildReservation(runID, taskDigest, []uint8{code}, sequence, acquisitionLifecycleCap)
		if err != nil || reservation.Status != "reserved" {
			return nil, fmt.Errorf("reserve acquisition operation %d: %w", sequence, err)
		}
		name := fmt.Sprintf("AR.Reservation.%s.%05d", runID, sequence)
		u := unit.New(name)
		u.Set("isA", []string{"CompoundWorkReservation", "Anything"})
		u.Set("canonicalObject", string(reservation.Canonical))
		u.Set("objectDigest", reservation.Digest)
		store.Put(u)
		reservationNames[sequence] = name
		meterPlan[sequence] = dsl.ActionRelationMeterPlanEntry{Code: uint16(code), SourceTaskDigest: reservation.Digest}
	}
	if err := dsl.RegisterActionRelationMeterPlan(meterToken, meterPlan); err != nil {
		return nil, err
	}
	fail := func(err error) (*Session, error) {
		dsl.UnregisterActionRelationMeter(meterToken)
		return nil, err
	}
	experiment.Set("meterToken", meterToken)
	experiment.Set("runID", runID)
	experiment.Set("reservationUnits", reservationNames)
	firstA, err := actionrelations.ParseOccurrence(training[0].AOccurrence)
	if err != nil {
		return fail(err)
	}
	firstB, err := actionrelations.ParseOccurrence(training[0].BOccurrence)
	if err != nil {
		return fail(err)
	}
	pattern, err := actionrelations.PatternFor(firstA, firstB)
	if err != nil {
		return fail(err)
	}
	patternJSON, _ := pattern.CanonicalJSON()
	experiment.Set("pattern", string(patternJSON))
	experiment.Set("patternUnit", putCanonical(store, "ActionRelationPattern", patternJSON))
	experiment.Set("family", family)
	experiment.Set("familyName", actionrelationfixturecore.FamilyNames[family])
	experiment.Set("scope", scope)
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
	session := &Session{Store: store, Experiment: experiment.Name, Scope: scope, RunID: runID, engine: eng, meterToken: meterToken}
	run, err := session.Snapshot()
	if err != nil {
		return fail(err)
	}
	wantCandidates, wantEdges, wantLiterals, wantGuardResults := 451, 450, 13920, 7216
	if scope == "no-guard" {
		wantCandidates, wantEdges, wantLiterals, wantGuardResults = 1, 0, 0, 16
	}
	if run.Observations != 16 || run.Candidates != wantCandidates || run.Edges != wantEdges || run.LiteralRows != wantLiterals || run.GuardResults != wantGuardResults || run.CandidateResults != wantCandidates || run.Winners < 1 || experiment.GetBool("candidatesFinalized") != true {
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
	wantCandidates, wantEvaluations := 451, 2
	if s != nil && s.Scope == "no-guard" {
		wantCandidates, wantEvaluations = 1, 1
	}
	if s == nil || s.closed || len(roots.CandidateLeaves) != wantCandidates || len(roots.EvaluationTableRoots) != wantEvaluations || s.Scope == "nous" && !actionrelationsDigest(roots.EdgeTableRoot) || s.Scope == "no-guard" && roots.EdgeTableRoot != zeroDigest {
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
	if err := dsl.ActionRelationMeterPlanComplete(s.meterToken); err != nil {
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
	return ExecuteFamily(domainsDir, token, 0)
}

func ExecuteFamily(domainsDir, token string, family int) (Run, error) {
	return ExecuteFamilyScope(domainsDir, token, family, "nous")
}

func ExecuteNoGuard(domainsDir, token string, family int) (Run, error) {
	return ExecuteFamilyScope(domainsDir, token, family, "no-guard")
}

func ExecuteFamilyScope(domainsDir, token string, family int, scope string) (Run, error) {
	session, err := BeginFamilyScope(domainsDir, token, family, scope)
	if err != nil {
		return Run{}, err
	}
	run, err := session.Snapshot()
	if err != nil {
		session.Abort()
		return Run{}, err
	}
	experiment := run.Store.Get(run.Experiment)
	candidateLeaves := developmentDigests("candidate-"+scope, len(experiment.GetStrings("candidateUnits")))
	winnerLeaves := developmentDigests("winner", len(experiment.GetStrings("winnerResultUnits")))
	for index, name := range experiment.GetStrings("candidateUnits") {
		run.Store.Get(name).Set("tableLeafDigest", candidateLeaves[index])
	}
	for index, name := range experiment.GetStrings("winnerResultUnits") {
		run.Store.Get(name).Set("tableLeafDigest", winnerLeaves[index])
	}
	edgeRoot := zeroDigest
	evaluationRoots := developmentDigests("evaluation-table-"+scope, 1)
	if scope == "nous" {
		edgeRoot = developmentDigests("edge-table", 1)[0]
		evaluationRoots = developmentDigests("evaluation-table", 2)
	}
	return session.BindEvidence(EvidenceRoots{CandidateLeaves: candidateLeaves, EdgeTableRoot: edgeRoot, EvaluationTableRoots: evaluationRoots, WinnerLeaves: winnerLeaves})
}

const zeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

const (
	acceptedPlanCommit      = "a3e18b10a01cf83315bff398586e91cd33544861"
	acquisitionLifecycleCap = 2_000_000
)

func acquisitionOperationSchedule(training []actionrelationfixturecore.Case, scope string) []uint8 {
	var result []uint8
	for _, testCase := range training {
		result = append(result, 5, 5, 4, 4)
		if testCase.AInitiallyApplicable {
			result = append(result, 5, 4)
		}
		if testCase.BInitiallyApplicable {
			result = append(result, 5, 4)
		}
		if testCase.Label == "commutes" || testCase.Label == "conflicts" {
			result = append(result, 6)
		}
	}
	guards := actionrelations.EnumerateGuards()
	if scope == "no-guard" {
		guards = guards[:1]
	}
	result = append(result, 1)
	if scope == "nous" {
		for range guards[1:] {
			result = append(result, 3, 2)
		}
	}
	for _, guard := range guards {
		for range training {
			for range guard.Literals {
				result = append(result, 7)
			}
			result = append(result, 22)
		}
	}
	for range guards {
		result = append(result, 20)
	}
	return append(result, 8)
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
