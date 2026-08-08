package causalrun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/causalrun/taskbridge"
	"github.com/chazu/nous/internal/causalv2"
	"github.com/chazu/nous/internal/cueload"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

type State string

const (
	StateReady           State = "ready"
	StateProposing       State = "proposing"
	StateAwaitingTeacher State = "awaiting-teacher"
	StateResponsePresent State = "response-present"
	StateUpdating        State = "updating"
	StateTerminal        State = "terminal"
)

const (
	proposalTaskSlot      = "causalV2Propose"
	authorizationTaskSlot = "causalV2Authorize"
	updateTaskSlot        = "causalV2Update"
	finalizeTaskSlot      = "causalV2Finalize"
)

// Teacher is the sole online authority crossing the hidden-data boundary.
// No store, profile, posterior, or hidden-model accessor is provided.
type Teacher interface {
	Respond(opaqueToken, canonicalActionCode string) (threeBitOutcome string, err error)
}

type runtimeCursor struct {
	state                State
	latestSnapshotDigest string
}

// Boundary is a read-only view of a verified runtime boundary.
type Boundary struct {
	State                   State  `json:"state"`
	Step                    int    `json:"step"`
	SelectedAction          string `json:"selected_action"`
	SelectionArtifactDigest string `json:"selection_artifact_digest"`
	AuthorizationDigest     string `json:"authorization_digest"`
}

// EpisodeResult is the production-visible result. It intentionally contains
// no hidden hypothesis or correctness judgment; those belong to the
// post-terminal experimental audit package.
type EpisodeResult struct {
	Seed             int64            `json:"seed"`
	ProfileDigest    string           `json:"profile_digest"`
	FixtureDigest    string           `json:"fixture_digest"`
	AcquisitionCode  string           `json:"acquisition_code"`
	Actions          []string         `json:"actions"`
	TeacherOutcomes  []string         `json:"teacher_outcomes"`
	Terminal         string           `json:"terminal"`
	Score            int              `json:"score"`
	Cost             int              `json:"cost"`
	FinalPosterior   []string         `json:"final_posterior"`
	PosteriorDigest  string           `json:"posterior_digest"`
	TranscriptDigest string           `json:"transcript_digest"`
	ProductionCounts Counts           `json:"production_counts"`
	TeacherCounts    Counts           `json:"teacher_counts"`
	DynamicCounts    Counts           `json:"dynamic_counts"`
	DynamicBenchmark DynamicBenchmark `json:"dynamic_benchmark"`
	CacheTrace       CacheTrace       `json:"cache_trace"`
	Artifacts        [][]byte         `json:"-"`
	Mutation         MutationEvidence `json:"-"`
}

// MutationEvidence is control-only evidence copied from the engine's actual
// mutation hook. It is excluded from episode JSON and scoring.
type MutationEvidence struct {
	Config      MutationConfigEvidence `json:"config"`
	Mutants     []MutantRecord         `json:"mutants"`
	MeterCounts Counts                 `json:"meter_counts"`
}

type MutationConfigEvidence struct {
	Enabled           bool    `json:"enabled"`
	Interval          int     `json:"interval"`
	MaxMutants        int     `json:"max_mutants"`
	MutantWorth       int     `json:"mutant_worth"`
	ValidateOnly      bool    `json:"validate_only"`
	MinApplics        int     `json:"min_applics"`
	MutationThreshold float64 `json:"mutation_threshold"`
}

type MutantRecord struct {
	Name          string `json:"name"`
	MutantOf      string `json:"mutant_of"`
	SourceSlot    string `json:"source_slot"`
	Operation     string `json:"operation"`
	ProgramDigest string `json:"program_digest"`
	Worth         int    `json:"worth"`
}

func mutationProgramDigest(program string) string {
	sum := sha256.Sum256([]byte(program))
	return hex.EncodeToString(sum[:])
}

// CacheTrace records every six-action semantic-cache lookup in step-major,
// canonical-action order. It is production provenance, not work tariff.
type CacheTrace struct {
	Statuses []string `json:"statuses"`
	Hits     int      `json:"hits"`
	Misses   int      `json:"misses"`
}

type Runner struct {
	fixtureBytes []byte
	profileBytes []byte
	fixture      causalv2.PublicFixture
	profile      causalv2.Profile
	teacher      Teacher

	store        *unit.Store
	agenda       *agenda.Agenda
	engine       *engine.Engine
	cursor       runtimeCursor
	meter        WorkMeter
	teacherMeter WorkMeter
	dynamic      *DynamicPolicy

	episodeKey               string
	step                     int
	posterior                []string
	initial                  []string
	posteriorArtifact        artifactRef
	initialPosteriorArtifact artifactRef
	totalCost                int
	consumed                 []string
	actions                  []string
	outcomes                 []string
	transcriptDigest         string
	terminal                 string
	pendingTerminal          string

	artifacts              []artifactRef
	byDigest               map[string]artifactRef
	byRequest              map[string]artifactRef
	cache                  map[string]cachedCandidate
	cacheTrace             CacheTrace
	current                map[string]candidateArtifacts
	forcedActionCode       string
	proposalActions        []string
	disableCacheReuse      bool
	ties                   []artifactRef
	selection              artifactRef
	pendingAuthorization   causalv2.Authorization
	authorization          artifactRef
	result                 artifactRef
	lastTranscript         artifactRef
	pendingUpdate          *updateDecision
	pendingPosterior       artifactRef
	pendingConsumption     artifactRef
	pendingSnapshotDigest  string
	updateEliminationIndex int
	teacherCalled          bool
	runtimeName            string
	cueMaterializing       bool
	driverMaterializing    bool
	taskErr                error
	cueExecutions          map[string]int
	closed                 bool
	activeCUETask          string
	taskScope              *taskbridge.Scope
	revokeTaskScope        func()
}

var runtimeSequence atomic.Uint64

// NewEpisode strictly verifies canonical public bytes before allocating any
// store state. The returned runner owns an isolated store and agenda.
func NewEpisode(publicFixtureBytes, profileBytes []byte, teacher Teacher) (*Runner, error) {
	if teacher == nil {
		return nil, errors.New("nil causal teacher")
	}
	if len(profileBytes) > causalv2.PreregisteredManifest().DescriptorByteCap {
		return nil, errors.New("profile exceeds descriptor byte cap")
	}
	profile, err := causalv2.VerifyProfile(profileBytes)
	if err != nil {
		return nil, fmt.Errorf("profile: %w", err)
	}
	fixture, err := causalv2.VerifyPublicFixtureForPanel(publicFixtureBytes, profile.Panel)
	if err != nil {
		return nil, fmt.Errorf("public fixture: %w", err)
	}
	if profile.FixtureDigest != fixture.FixtureDigest {
		return nil, errors.New("profile is bound to a different fixture")
	}
	if profile.Seed != fixture.Seed {
		return nil, errors.New("profile seed does not match fixture seed")
	}
	if _, _, err := acquisitionRule(profile.AcquisitionCode); err != nil {
		return nil, err
	}

	costs := [3]int{fixture.Costs[0], fixture.Costs[1], fixture.Costs[2]}
	dynamic, err := NewDynamicPolicy(fixture.InitialPosterior, costs)
	if err != nil {
		return nil, err
	}
	store := unit.NewStore()
	if err := loadCausalDomain(store); err != nil {
		return nil, err
	}
	runner := &Runner{
		fixtureBytes: append([]byte(nil), publicFixtureBytes...),
		profileBytes: append([]byte(nil), profileBytes...),
		fixture:      fixture,
		profile:      profile,
		teacher:      teacher,
		store:        store,
		agenda:       agenda.New(),
		cursor:       runtimeCursor{state: StateReady},
		dynamic:      dynamic,
		episodeKey:   profile.ProfileDigest,
		posterior:    append([]string(nil), fixture.InitialPosterior...),
		initial:      append([]string(nil), fixture.InitialPosterior...),
		consumed:     []string{}, actions: []string{}, outcomes: []string{},
		byDigest:      make(map[string]artifactRef),
		byRequest:     make(map[string]artifactRef),
		cache:         make(map[string]cachedCandidate),
		current:       make(map[string]candidateArtifacts),
		cueExecutions: make(map[string]int),
	}
	for _, action := range causal.Actions() {
		runner.proposalActions = append(runner.proposalActions, action.Code())
	}
	runner.runtimeName = fmt.Sprintf("Causal.Runtime.%s.%d", profile.ProfileDigest, runtimeSequence.Add(1))
	runner.engine = engine.New(runner.store, runner.agenda)
	runner.engine.Verbosity = 0
	runner.engine.Out = io.Discard
	runner.engine.MutConfig.Enabled = false
	runner.taskScope = taskbridge.NewScope()
	runner.revokeTaskScope, err = dsl.RegisterCausalTaskScope(runner.engine.VM, runner.taskScope)
	if err != nil {
		return nil, err
	}
	if err := runner.taskScope.Register(runner.runtimeName, runner.validTask, runner.beginTask, runner.operation, runner.endTask); err != nil {
		runner.revokeTaskScope()
		return nil, err
	}
	runtimeUnit := unit.New(runner.runtimeName)
	runtimeUnit.Set("isA", []string{"CausalRuntimeCursor", "Anything"})
	runtimeUnit.Set("state", string(StateReady))
	runtimeUnit.Set("latestSnapshotDigest", "")
	runner.store.Put(runtimeUnit)
	if err := runner.meter.chargeProfile(64); err != nil {
		runner.Close()
		return nil, err
	}
	runner.driverMaterializing = true
	if err := runner.initialize(); err != nil {
		runner.driverMaterializing = false
		runner.Close()
		return nil, err
	}
	runner.driverMaterializing = false
	return runner, nil
}

func loadCausalDomain(store *unit.Store) error {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("cannot locate causal CUE domain")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "domains"))
	for _, name := range []string{"common", "causal"} {
		defs, err := cueload.LoadDir(filepath.Join(root, name))
		if err != nil {
			return fmt.Errorf("load %s CUE domain: %w", name, err)
		}
		for _, def := range defs {
			u := unit.New(def.Name)
			u.SetWorth(def.Worth)
			if len(def.IsA) != 0 {
				u.Set("isA", def.IsA)
			}
			for slot, value := range def.Slots {
				u.Set(slot, value)
			}
			store.Put(u)
		}
	}
	return nil
}

// Close revokes the CUE task capability. It is idempotent.
func (r *Runner) Close() {
	if r == nil || r.closed {
		return
	}
	r.closed = true
	r.taskScope.Unregister(r.runtimeName)
	if r.revokeTaskScope != nil {
		r.revokeTaskScope()
	}
}

func (r *Runner) syncCursor() {
	if cursor := r.store.Get(r.runtimeName); cursor != nil {
		cursor.Set("state", string(r.cursor.state))
		cursor.Set("latestSnapshotDigest", r.cursor.latestSnapshotDigest)
	}
}

func (r *Runner) validTask(slot string) bool {
	if r.closed || r.cueMaterializing {
		return false
	}
	switch slot {
	case proposalTaskSlot:
		return r.cursor.state == StateProposing
	case authorizationTaskSlot:
		return r.cursor.state == StateProposing && r.pendingAuthorization.AuthorizationDigest != ""
	case updateTaskSlot:
		return r.cursor.state == StateResponsePresent
	case finalizeTaskSlot:
		return r.cursor.state == StateReady && r.pendingTerminal != ""
	default:
		return false
	}
}

func (r *Runner) beginTask(slot string) error {
	if !r.validTask(slot) {
		return fmt.Errorf("invalid CUE causal task %q", slot)
	}
	r.cueMaterializing = true
	r.activeCUETask = slot
	if err := r.meter.chargeCycle(1); err != nil {
		r.taskErr = err
		return err
	}
	return nil
}

func (r *Runner) operation(name string, arguments ...string) (value any, err error) {
	if !r.cueMaterializing || r.activeCUETask == "" {
		return nil, errors.New("causal operation is outside an active CUE task")
	}
	defer func() {
		if err != nil {
			r.taskErr = err
		}
	}()
	if err = r.requireOperationTask(name); err != nil {
		return nil, err
	}
	switch name {
	case "actions":
		value = append([]string(nil), r.proposalActions...)
	case "prepare-proposal":
		err = r.prepareProposal()
	case "materialize-cache":
		err = r.materializeCandidateCache(arguments)
	case "materialize-proposal":
		err = r.materializeCandidateProposal(arguments)
	case "materialize-partition":
		err = r.materializeCandidatePartition(arguments)
	case "materialize-score":
		err = r.materializeCandidateScore(arguments)
	case "better":
		value, err = r.candidateBetter(arguments)
	case "equal-score":
		value, err = r.candidateEqual(arguments)
	case "materialize-tie":
		err = r.materializeCandidateTie(arguments)
	case "materialize-selection":
		err = r.materializeCandidateSelection(arguments)
	case "materialize-authorization":
		err = r.materializeAuthorization()
	case "materialize-awaiting-snapshot":
		err = r.materializeAwaitingSnapshot()
	case "prepare-update":
		err = r.prepareUpdate()
	case "eliminated":
		value, err = r.updateEliminated()
	case "materialize-elimination":
		err = r.materializeUpdateElimination(arguments)
	case "materialize-posterior":
		err = r.materializeUpdatePosterior()
	case "materialize-consumption":
		err = r.materializeUpdateConsumption()
	case "materialize-transcript":
		err = r.materializeUpdateTranscript()
	case "materialize-next-snapshot":
		err = r.materializeUpdateSnapshot()
	case "terminal":
		value = r.terminal != ""
	case "materialize-terminal":
		err = r.materializeUpdateTerminal()
	case "finish-update":
		err = r.finishUpdate()
	case "finalize-zero":
		err = r.finalizeZeroAction(r.pendingTerminal)
	default:
		err = fmt.Errorf("unknown causal CUE operation %q", name)
	}
	return value, err
}

func (r *Runner) requireOperationTask(name string) error {
	var want string
	switch name {
	case "actions", "prepare-proposal", "materialize-cache", "materialize-proposal", "materialize-partition", "materialize-score", "better", "equal-score", "materialize-tie", "materialize-selection":
		want = proposalTaskSlot
	case "materialize-authorization", "materialize-awaiting-snapshot":
		want = authorizationTaskSlot
	case "prepare-update", "eliminated", "materialize-elimination", "materialize-posterior", "materialize-consumption", "materialize-transcript", "materialize-next-snapshot", "terminal", "materialize-terminal", "finish-update":
		want = updateTaskSlot
	case "finalize-zero":
		want = finalizeTaskSlot
	default:
		return fmt.Errorf("unknown causal CUE operation %q", name)
	}
	if r.activeCUETask != want {
		return fmt.Errorf("causal CUE operation %q belongs to %q, active %q", name, want, r.activeCUETask)
	}
	return nil
}

func (r *Runner) endTask(slot string) error {
	if !r.cueMaterializing || r.activeCUETask != slot {
		return fmt.Errorf("CUE task end %q does not match active %q", slot, r.activeCUETask)
	}
	r.cueMaterializing = false
	r.activeCUETask = ""
	r.syncCursor()
	if r.taskErr != nil {
		return r.taskErr
	}
	r.cueExecutions[slot]++
	return nil
}

func (r *Runner) initialize() error {
	observation, err := r.allocate(0, "observation", observationPayload{Outcome: r.fixture.PassiveOutcome})
	if err != nil {
		return err
	}
	_ = observation
	posterior, err := r.materializePosterior(0, r.posterior)
	if err != nil {
		return err
	}
	r.posteriorArtifact = posterior
	r.initialPosteriorArtifact = posterior
	return r.materializeSnapshot(StateReady)
}

func (r *Runner) State() State { return r.cursor.state }

func (r *Runner) Boundary() Boundary {
	return Boundary{
		State:                   r.cursor.state,
		Step:                    r.step,
		SelectedAction:          r.selectedAction(),
		SelectionArtifactDigest: r.selection.Digest,
		AuthorizationDigest:     r.authorization.Digest,
	}
}

func (r *Runner) selectedAction() string {
	if selected, ok := r.currentAction(); ok {
		return selected
	}
	return ""
}

func (r *Runner) currentAction() (string, bool) {
	for action, artifacts := range r.current {
		if artifacts.candidate.action == action && r.selection.Digest != "" {
			var payload selectionPayload
			if err := r.decodePayload(r.selection, &payload); err == nil && payload.Action == action {
				return action, true
			}
		}
	}
	return "", false
}

// AdvanceToTeacher runs exactly one proposal and authorization cycle and
// stops at the verified external-teacher boundary.
func (r *Runner) AdvanceToTeacher(ctx context.Context) (Boundary, error) {
	if r.cursor.state != StateReady {
		return r.Boundary(), fmt.Errorf("advance from state %q", r.cursor.state)
	}
	if terminal := terminalFor(r.initial, r.posterior, len(r.consumed)); terminal != "" || r.profile.AcquisitionCode == string(PolicyPassiveOnly) {
		if terminal == "" {
			terminal = "budget-exhausted"
		}
		r.pendingTerminal = terminal
		r.agenda.Push(&agenda.Task{Priority: 900, UnitName: r.runtimeName, SlotName: finalizeTaskSlot})
		if err := r.drainOne(ctx, finalizeTaskSlot); err != nil {
			return r.Boundary(), err
		}
		return r.Boundary(), nil
	}
	if r.agenda.Len() != 0 {
		return r.Boundary(), errors.New("nonempty agenda at ready boundary")
	}
	r.cursor.state = StateProposing
	r.syncCursor()
	r.agenda.Push(&agenda.Task{Priority: 900, UnitName: r.runtimeName, SlotName: proposalTaskSlot})
	if err := r.drainOne(ctx, proposalTaskSlot); err != nil {
		return r.Boundary(), err
	}
	authorization, err := verifyProposalBoundary(r)
	if err != nil {
		return r.Boundary(), err
	}
	r.pendingAuthorization = authorization
	r.agenda.Push(&agenda.Task{Priority: 900, UnitName: r.runtimeName, SlotName: authorizationTaskSlot})
	if err := r.drainOne(ctx, authorizationTaskSlot); err != nil {
		return r.Boundary(), err
	}
	if err := verifyAuthorizationBoundary(r); err != nil {
		return r.Boundary(), err
	}
	return r.Boundary(), nil
}

// Respond invokes the narrow teacher exactly once for the current verified
// authorization and then lets the sole update task consume that result.
func (r *Runner) Respond(ctx context.Context) error {
	if err := verifyAuthorizationBoundary(r); err != nil {
		return err
	}
	if r.teacherCalled {
		return errors.New("teacher already called for current authorization")
	}
	action := r.selectedAction()
	outcome, err := r.teacher.Respond(r.fixture.OpaqueToken, action)
	if err != nil {
		return err
	}
	r.teacherCalled = true
	if err := validateThreeBitOutcome(outcome); err != nil {
		return err
	}
	if err := r.teacherMeter.chargeSCM(1); err != nil {
		return err
	}
	r.driverMaterializing = true
	r.result, err = r.allocate(r.step, "result", resultPayload{
		AuthorizationArtifactDigest: r.authorization.Digest,
		Action:                      action,
		Outcome:                     outcome,
	})
	r.driverMaterializing = false
	if err != nil {
		return err
	}
	r.cursor.state = StateResponsePresent
	r.syncCursor()
	r.agenda.Push(&agenda.Task{Priority: 900, UnitName: r.runtimeName, SlotName: updateTaskSlot})
	if err := r.drainOne(ctx, updateTaskSlot); err != nil {
		return err
	}
	return verifyReadyOrTerminalBoundary(r)
}

// Run executes an episode to a verified terminal, never exposing hidden data.
func (r *Runner) Run(ctx context.Context) (EpisodeResult, error) {
	for r.cursor.state != StateTerminal {
		if _, err := r.AdvanceToTeacher(ctx); err != nil {
			return EpisodeResult{}, err
		}
		if r.cursor.state == StateTerminal {
			break
		}
		if err := r.Respond(ctx); err != nil {
			return EpisodeResult{}, err
		}
	}
	return r.Result()
}

func (r *Runner) Result() (EpisodeResult, error) {
	if err := verifyReadyOrTerminalBoundary(r); err != nil {
		return EpisodeResult{}, err
	}
	if r.cursor.state != StateTerminal {
		return EpisodeResult{}, errors.New("episode is not terminal")
	}
	wantCacheStatuses := 6 * len(r.actions)
	if len(r.cacheTrace.Statuses) != wantCacheStatuses || r.cacheTrace.Hits+r.cacheTrace.Misses != wantCacheStatuses {
		return EpisodeResult{}, fmt.Errorf("cache trace cardinality=%d/%d, want %d", len(r.cacheTrace.Statuses), r.cacheTrace.Hits+r.cacheTrace.Misses, wantCacheStatuses)
	}
	posteriorDigest, err := hypothesisSetDigest(r.posterior)
	if err != nil {
		return EpisodeResult{}, err
	}
	score := causal.InvalidScore
	if r.terminal == "identified" || r.terminal == "equivalence" {
		score = r.totalCost
	}
	artifacts := make([][]byte, len(r.artifacts))
	for index, artifact := range r.artifacts {
		artifacts[index] = append([]byte(nil), artifact.Canonical...)
	}
	dynamicBenchmark := DynamicBenchmark{}
	if r.profile.AcquisitionCode == string(PolicyDynamicOptimal) {
		dynamicBenchmark, err = r.dynamic.CompleteBenchmark(r.actions, r.outcomes)
		if err != nil {
			return EpisodeResult{}, err
		}
	}
	return EpisodeResult{
		Seed: r.fixture.Seed, ProfileDigest: r.profile.ProfileDigest,
		FixtureDigest: r.fixture.FixtureDigest, AcquisitionCode: r.profile.AcquisitionCode,
		Actions: append([]string(nil), r.actions...), TeacherOutcomes: append([]string(nil), r.outcomes...),
		Terminal: r.terminal, Score: score, Cost: r.totalCost,
		FinalPosterior: append([]string(nil), r.posterior...), PosteriorDigest: posteriorDigest,
		TranscriptDigest: r.transcriptDigest, ProductionCounts: r.meter.Counts(),
		TeacherCounts: r.teacherMeter.Counts(), DynamicCounts: dynamicBenchmark.Counts,
		DynamicBenchmark: dynamicBenchmark,
		CacheTrace:       CacheTrace{Statuses: append([]string(nil), r.cacheTrace.Statuses...), Hits: r.cacheTrace.Hits, Misses: r.cacheTrace.Misses},
		Artifacts:        artifacts,
	}, nil
}

func terminalFor(initial, posterior []string, actionCount int) string {
	if len(posterior) == 1 {
		return "identified"
	}
	if causal.CompleteClass(initial, posterior) {
		return "equivalence"
	}
	if actionCount >= causal.MaximumActions {
		return "budget-exhausted"
	}
	return ""
}

func validateThreeBitOutcome(outcome string) error {
	if len(outcome) != 3 || strings.Trim(outcome, "01") != "" {
		return fmt.Errorf("invalid teacher outcome %q", outcome)
	}
	return nil
}

func (r *Runner) cacheKey(posteriorDigest, action string) string {
	return fmt.Sprintf("%s\x00%s\x00%s", r.profile.ProfileDigest, posteriorDigest, action)
}

func (r *Runner) forcedAction() (string, error) {
	switch r.profile.AcquisitionCode {
	case string(PolicyUniformRandom):
		if r.step < 0 || r.step >= len(r.fixture.UniformRandomActions) {
			return "", errors.New("uniform-random prefix exhausted")
		}
		return r.fixture.UniformRandomActions[r.step], nil
	case string(PolicyDynamicOptimal):
		return r.dynamic.Choose(r.posterior, r.consumed)
	default:
		return "", nil
	}
}

func (r *Runner) actionCost(code string) (int, error) {
	action, err := causal.ParseAction(code)
	if err != nil {
		return 0, err
	}
	return r.fixture.Costs[action.Variable], nil
}

func (r *Runner) drainOne(ctx context.Context, wantSlot string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	task := r.agenda.Peek()
	if task == nil || task.UnitName != r.runtimeName || task.SlotName != wantSlot || r.agenda.Len() != 1 {
		return fmt.Errorf("unexpected causal task at %s", wantSlot)
	}
	r.taskErr = nil
	beforeTasks := r.engine.TaskNum
	beforeExecutions := r.cueExecutions[wantSlot]
	r.engine.MaxCycles = 1
	if err := r.engine.Run(ctx); err != nil {
		return err
	}
	if r.taskErr != nil {
		r.cueMaterializing = false
		r.activeCUETask = ""
		r.syncCursor()
		return r.taskErr
	}
	if r.cueMaterializing || r.activeCUETask != "" {
		r.cueMaterializing = false
		r.activeCUETask = ""
		r.syncCursor()
		return fmt.Errorf("CUE task %s did not reach its guarded end", wantSlot)
	}
	if r.engine.TaskNum != beforeTasks+1 || r.engine.Cycle() != 1 || r.agenda.Len() != 0 || r.cueExecutions[wantSlot] != beforeExecutions+1 {
		return fmt.Errorf("CUE task %s did not execute exactly once", wantSlot)
	}
	return nil
}

func equalStrings(left, right []string) bool { return slices.Equal(left, right) }
