package causalrun

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/causalv2"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

// ControlName is intentionally closed: online controls cannot be replaced by
// caller-provided hooks or predicates.
type ControlName string

const (
	ControlHiddenTwin          ControlName = "hidden-twin"
	ControlWrongContext        ControlName = "wrong-context"
	ControlStaticRule          ControlName = "static-rule"
	ControlRecomputedRule      ControlName = "recomputed-rule"
	ControlOpaqueAlias         ControlName = "opaque-alias"
	ControlPresentationOrder   ControlName = "presentation-order"
	ControlProposalOrder       ControlName = "proposal-order"
	ControlCostPerturbation    ControlName = "cost-perturbation"
	ControlOccupiedName        ControlName = "occupied-name"
	ControlAlternateDescriptor ControlName = "alternate-descriptor"
	ControlMutationInert       ControlName = "mutation-inert"
	ControlChildVM             ControlName = "child-vm"
	ControlStaleResponse       ControlName = "stale-response"
	ControlDuplicateResponse   ControlName = "duplicate-response"
	ControlCorruptionSuite     ControlName = "corruption-suite"
	ControlDeterministicJSON   ControlName = "deterministic-json"
)

// ControlInput contains canonical public contexts and narrow teacher
// capabilities. Paired inputs are mandatory for controls which compare two
// independently signed contexts or two hidden bindings.
type ControlInput struct {
	FixtureBytes       []byte
	ProfileBytes       []byte
	Teacher            Teacher
	PairedFixtureBytes []byte
	PairedProfileBytes []byte
	PairedTeacher      Teacher
	BaselineArtifacts  [][]byte
	StaticRuleCode     string
	SelectedRuleCode   string
}

// ControlObservation is semantic evidence emitted by an executed trial. The
// before/after teacher counts make the fail-before-new-evidence predicate
// explicit and locally checkable.
type ControlObservation struct {
	Name               ControlName            `json:"name"`
	FixtureDigest      string                 `json:"fixture_digest"`
	Treatment          causalv2.ControlResult `json:"treatment"`
	Control            causalv2.ControlResult `json:"control"`
	Observed           string                 `json:"observed"`
	Counts             Counts                 `json:"counts"`
	Passed             bool                   `json:"passed"`
	TeacherCallsBefore int                    `json:"teacher_calls_before"`
	TeacherCallsAfter  int                    `json:"teacher_calls_after"`
	TreatmentMutation  MutationEvidence       `json:"treatment_mutation"`
	ControlMutation    MutationEvidence       `json:"control_mutation"`
	TreatmentCounts    Counts                 `json:"treatment_counts"`
	ControlCounts      Counts                 `json:"control_counts"`
	AllCacheMisses     bool                   `json:"all_cache_misses"`
	TreatmentCache     CacheTrace             `json:"treatment_cache"`
	ControlCache       CacheTrace             `json:"control_cache"`
	ChildVM            ChildVMEvidence        `json:"child_vm"`
}

type ChildVMEvidence struct {
	FixtureDigest      string `json:"fixture_digest"`
	ProfileDigest      string `json:"profile_digest"`
	Operation          string `json:"operation"`
	ArtifactsBefore    int    `json:"artifacts_before"`
	ArtifactsAfter     int    `json:"artifacts_after"`
	MeterCountsBefore  Counts `json:"meter_counts_before"`
	MeterCountsAfter   Counts `json:"meter_counts_after"`
	TeacherCallsBefore int    `json:"teacher_calls_before"`
	TeacherCallsAfter  int    `json:"teacher_calls_after"`
	FailureCode        string `json:"failure_code"`
}

type countingTeacher struct {
	inner Teacher
	calls int
}

func (t *countingTeacher) Respond(token, action string) (string, error) {
	t.calls++
	return t.inner.Respond(token, action)
}

type controlRunOptions struct {
	reverseProposals bool
	mutation         bool
	mutationProbe    bool
	recompute        bool
}

func executeControlEpisode(ctx context.Context, fixtureBytes, profileBytes []byte, teacher Teacher, options controlRunOptions) (EpisodeResult, error) {
	runner, err := NewEpisode(fixtureBytes, profileBytes, teacher)
	if err != nil {
		return EpisodeResult{}, err
	}
	defer runner.Close()
	if options.reverseProposals {
		slices.Reverse(runner.proposalActions)
	}
	runner.engine.MutConfig.Enabled = false
	runner.disableCacheReuse = options.recompute
	result, err := runner.Run(ctx)
	if err != nil {
		return EpisodeResult{}, err
	}
	if options.mutationProbe {
		result.Mutation, err = executeMutationProbe(ctx, runner, options.mutation)
		if err != nil {
			return EpisodeResult{}, err
		}
	}
	return result, nil
}

func executeMutationProbe(ctx context.Context, runner *Runner, enabled bool) (MutationEvidence, error) {
	target := runner.store.Get("H-Causal-V2-Propose")
	if target == nil {
		return MutationEvidence{}, errors.New("mutation probe target is absent")
	}
	target.Set("overallRecord", map[string]any{"successes": 0, "failures": 1})
	probe := engine.New(runner.store, agenda.New())
	probe.Verbosity = 0
	probe.MutConfig.Enabled = enabled
	probe.MutConfig.Interval = 1
	probe.MutConfig.MaxMutants = 1
	probe.MutConfig.MutantWorth = 400
	probe.MutConfig.ValidateOnly = false
	probe.MutConfig.MinApplics = 1
	probe.MutConfig.MutationThreshold = 2
	probe.MaxCycles = 2
	if err := probe.Run(ctx); err != nil {
		return MutationEvidence{}, fmt.Errorf("post-terminal mutation control cycles: %w", err)
	}
	mutants := mutationRecords(runner.store)
	evidence := MutationEvidence{
		Config:      MutationConfigEvidence{Enabled: enabled, Interval: 1, MaxMutants: 1, MutantWorth: 400, ValidateOnly: false, MinApplics: 1, MutationThreshold: 2},
		Mutants:     mutants,
		MeterCounts: Counts{EngineCycles: probe.Cycle(), AttributedUnits: len(mutants)},
	}
	if enabled && len(mutants) == 0 {
		return MutationEvidence{}, errors.New("enabled mutation probe created no mutant")
	}
	if !enabled && len(mutants) != 0 {
		return MutationEvidence{}, errors.New("disabled mutation probe created a mutant")
	}
	return evidence, nil
}

func mutationRecords(store *unit.Store) []MutantRecord {
	var records []MutantRecord
	for _, name := range store.All() {
		u := store.Get(name)
		if u == nil || u.GetString("mutant_of") == "" {
			continue
		}
		slot := u.GetString("mutation_slot")
		records = append(records, MutantRecord{
			Name: name, MutantOf: u.GetString("mutant_of"), SourceSlot: slot,
			Operation: u.GetString("mutation_op"), ProgramDigest: mutationProgramDigest(u.GetString(slot)), Worth: u.Worth(),
		})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })
	return records
}

func controlResult(result EpisodeResult, fixtureBytes []byte) (causalv2.ControlResult, error) {
	fixture, err := causalv2.VerifyPublicFixture(fixtureBytes)
	if err != nil {
		return causalv2.ControlResult{}, err
	}
	posteriorDigests := make([]string, 0, len(result.Actions)+1)
	for _, encoded := range result.Artifacts {
		artifact, err := causalv2.VerifyArtifact(encoded)
		if err != nil {
			return causalv2.ControlResult{}, err
		}
		if artifact.Kind != "posterior" {
			continue
		}
		payload, err := causalv2.StrictDecode[causalv2.PosteriorPayload](artifact.Payload)
		if err != nil {
			return causalv2.ControlResult{}, err
		}
		posteriorDigests = append(posteriorDigests, payload.SemanticSetDigest)
	}
	costs := make([]int, len(result.Actions))
	for index, code := range result.Actions {
		action, err := causal.ParseAction(code)
		if err != nil {
			return causalv2.ControlResult{}, err
		}
		costs[index] = fixture.Costs[action.Variable]
	}
	return causalv2.ControlResult{
		ProfileDigest: result.ProfileDigest, Actions: append([]string(nil), result.Actions...),
		Outcomes: append([]string(nil), result.TeacherOutcomes...), PosteriorDigests: posteriorDigests,
		Costs: costs, Terminal: result.Terminal, Score: result.Score,
		TranscriptDigest: result.TranscriptDigest,
	}, nil
}

func emptyControlResult(profileDigest, failure string) causalv2.ControlResult {
	return causalv2.ControlResult{ProfileDigest: profileDigest, Actions: []string{}, Outcomes: []string{}, PosteriorDigests: []string{}, Costs: []int{}, FailureCode: failure}
}

func semanticControlEqual(left, right causalv2.ControlResult) bool {
	return slices.Equal(left.Actions, right.Actions) && slices.Equal(left.Outcomes, right.Outcomes) &&
		slices.Equal(left.PosteriorDigests, right.PosteriorDigests) && slices.Equal(left.Costs, right.Costs) &&
		left.Terminal == right.Terminal && left.Score == right.Score
}

func addCounts(left, right Counts) Counts {
	return Counts{
		SCMEvaluations: left.SCMEvaluations + right.SCMEvaluations, PartitionAssignments: left.PartitionAssignments + right.PartitionAssignments,
		CellAccumulations: left.CellAccumulations + right.CellAccumulations, RuleComparisons: left.RuleComparisons + right.RuleComparisons,
		PosteriorChecks: left.PosteriorChecks + right.PosteriorChecks, ArtifactMaterializations: left.ArtifactMaterializations + right.ArtifactMaterializations,
		TranscriptFields: left.TranscriptFields + right.TranscriptFields, ProfileFields: left.ProfileFields + right.ProfileFields,
		MemoStates: left.MemoStates + right.MemoStates, MemoLookups: left.MemoLookups + right.MemoLookups,
		QEvaluations: left.QEvaluations + right.QEvaluations, TableLookups: left.TableLookups + right.TableLookups,
		EngineCycles: left.EngineCycles + right.EngineCycles, AttributedUnits: left.AttributedUnits + right.AttributedUnits,
		TotalWork: left.TotalWork + right.TotalWork,
	}
}

func verifiedContext(input ControlInput) (causalv2.PublicFixture, causalv2.Profile, error) {
	profile, err := causalv2.VerifyProfile(input.ProfileBytes)
	if err != nil {
		return causalv2.PublicFixture{}, causalv2.Profile{}, err
	}
	fixture, err := causalv2.VerifyPublicFixtureForPanel(input.FixtureBytes, profile.Panel)
	if err != nil {
		return causalv2.PublicFixture{}, causalv2.Profile{}, err
	}
	if profile.FixtureDigest != fixture.FixtureDigest {
		return causalv2.PublicFixture{}, causalv2.Profile{}, errors.New("profile/fixture mismatch")
	}
	return fixture, profile, nil
}

func pairedRequired(input ControlInput) error {
	if input.PairedTeacher == nil {
		return errors.New("control requires paired teacher")
	}
	return nil
}

func pairedContext(input ControlInput) ([]byte, []byte) {
	fixtureBytes, profileBytes := input.PairedFixtureBytes, input.PairedProfileBytes
	if len(fixtureBytes) == 0 {
		fixtureBytes = input.FixtureBytes
	}
	if len(profileBytes) == 0 {
		profileBytes = input.ProfileBytes
	}
	return fixtureBytes, profileBytes
}

// ExecuteControl executes one named online adversarial trial. It never exposes
// a Runner, Store, allocator, engine, or caller-supplied predicate.
func ExecuteControl(ctx context.Context, name ControlName, input ControlInput) (ControlObservation, error) {
	fixture, profile, err := verifiedContext(input)
	if err != nil {
		return ControlObservation{}, err
	}
	if input.Teacher == nil {
		return ControlObservation{}, errors.New("control requires primary teacher")
	}
	observation := ControlObservation{Name: name, FixtureDigest: fixture.FixtureDigest}
	primary := &countingTeacher{inner: input.Teacher}

	runPair := func(primaryOptions, pairedOptions controlRunOptions) (EpisodeResult, EpisodeResult, error) {
		if err := pairedRequired(input); err != nil {
			return EpisodeResult{}, EpisodeResult{}, err
		}
		pairedFixture, pairedProfile := pairedContext(input)
		paired := &countingTeacher{inner: input.PairedTeacher}
		left, err := executeControlEpisode(ctx, input.FixtureBytes, input.ProfileBytes, primary, primaryOptions)
		if err != nil {
			return EpisodeResult{}, EpisodeResult{}, err
		}
		right, err := executeControlEpisode(ctx, pairedFixture, pairedProfile, paired, pairedOptions)
		observation.TeacherCallsAfter = primary.calls + paired.calls
		return left, right, err
	}
	setPair := func(left EpisodeResult, leftFixture []byte, right EpisodeResult, rightFixture []byte) error {
		observation.Treatment, err = controlResult(left, leftFixture)
		if err != nil {
			return err
		}
		observation.Control, err = controlResult(right, rightFixture)
		observation.Counts = addCounts(left.ProductionCounts, right.ProductionCounts)
		observation.TreatmentCounts = left.ProductionCounts
		observation.ControlCounts = right.ProductionCounts
		observation.TreatmentCache = left.CacheTrace
		observation.ControlCache = right.CacheTrace
		return err
	}

	switch name {
	case ControlHiddenTwin:
		if err := pairedRequired(input); err != nil {
			return ControlObservation{}, err
		}
		paired := &countingTeacher{inner: input.PairedTeacher}
		pairedFixture, pairedProfileBytes := pairedContext(input)
		left, err := NewEpisode(input.FixtureBytes, input.ProfileBytes, primary)
		if err != nil {
			return ControlObservation{}, err
		}
		defer left.Close()
		right, err := NewEpisode(pairedFixture, pairedProfileBytes, paired)
		if err != nil {
			return ControlObservation{}, err
		}
		defer right.Close()
		leftBoundary, leftErr := left.AdvanceToTeacher(ctx)
		rightBoundary, rightErr := right.AdvanceToTeacher(ctx)
		observation.Treatment = emptyControlResult(profile.ProfileDigest, "")
		pairedProfile, _ := causalv2.VerifyProfile(pairedProfileBytes)
		observation.Control = emptyControlResult(pairedProfile.ProfileDigest, "")
		observation.Counts = addCounts(left.meter.Counts(), right.meter.Counts())
		observation.TeacherCallsAfter = primary.calls + paired.calls
		observation.Passed = leftErr == nil && rightErr == nil && leftBoundary == rightBoundary && artifactBytesEqual(left.ArtifactBytes(), right.ArtifactBytes()) && observation.TeacherCallsAfter == 0
		observation.Observed = "semantic-projection-equal"
	case ControlWrongContext:
		if len(input.PairedProfileBytes) == 0 {
			return ControlObservation{}, errors.New("wrong-context requires an alternate signed profile")
		}
		observation.TeacherCallsBefore = primary.calls
		runner, trialErr := NewEpisode(input.FixtureBytes, input.PairedProfileBytes, primary)
		if runner != nil {
			runner.Close()
		}
		observation.TeacherCallsAfter = primary.calls
		observation.Treatment = emptyControlResult(profile.ProfileDigest, "wrong-context-rejected")
		observation.Control = emptyControlResult("", "")
		observation.Passed = trialErr != nil && observation.TeacherCallsAfter == observation.TeacherCallsBefore
		observation.Observed = "fail-closed:wrong-context-rejected"
	case ControlStaticRule, ControlRecomputedRule, ControlOpaqueAlias, ControlPresentationOrder:
		pairedFixture, pairedProfileBytes := pairedContext(input)
		pairedProfile, profileErr := causalv2.VerifyProfile(pairedProfileBytes)
		if profileErr != nil {
			return ControlObservation{}, profileErr
		}
		pairedPublic, fixtureErr := causalv2.VerifyPublicFixture(pairedFixture)
		if fixtureErr != nil {
			return ControlObservation{}, fixtureErr
		}
		pairedOptions := controlRunOptions{}
		switch name {
		case ControlStaticRule:
			if input.SelectedRuleCode == "" || input.StaticRuleCode == "" {
				return ControlObservation{}, errors.New("static-rule requires selected and static rule codes")
			}
			if profile.AcquisitionCode != input.SelectedRuleCode || pairedProfile.AcquisitionCode != input.StaticRuleCode {
				return ControlObservation{}, errors.New("static-rule profiles do not encode the declared learned/static rules")
			}
		case ControlRecomputedRule:
			if input.SelectedRuleCode == "" {
				return ControlObservation{}, errors.New("recomputed-rule requires selected rule code")
			}
			if profile.AcquisitionCode != input.SelectedRuleCode || pairedProfile.AcquisitionCode != input.SelectedRuleCode {
				return ControlObservation{}, errors.New("recomputed-rule profiles do not freeze the selected rule")
			}
			pairedOptions.recompute = true
		case ControlOpaqueAlias:
			if slices.Equal(fixture.Aliases, pairedPublic.Aliases) {
				return ControlObservation{}, errors.New("opaque-alias control requires a changed alias assignment")
			}
		case ControlPresentationOrder:
			if slices.Equal(fixture.Presentation, pairedPublic.Presentation) {
				return ControlObservation{}, errors.New("presentation-order control requires a changed presentation")
			}
		}
		left, right, err := runPair(controlRunOptions{}, pairedOptions)
		if err != nil {
			return ControlObservation{}, err
		}
		if err := setPair(left, input.FixtureBytes, right, pairedFixture); err != nil {
			return ControlObservation{}, err
		}
		observation.Passed = semanticControlEqual(observation.Treatment, observation.Control)
		observation.Observed = "semantic-projection-equal"
		if name == ControlStaticRule {
			// This is a semantic-first baseline, not an invariance control. Its
			// result is retained for comparison and is not expected to equal the
			// learned policy.
			observation.Passed = observation.Treatment.FailureCode == "" && observation.Control.FailureCode == ""
			observation.Observed = "static-baseline-executed"
		}
		if name == ControlRecomputedRule {
			observation.AllCacheMisses = right.CacheTrace.Hits == 0 && right.CacheTrace.Misses == len(right.CacheTrace.Statuses) && len(right.CacheTrace.Statuses) == 6*len(right.Actions)
			observation.Passed = observation.Passed && right.ProductionCounts.TotalWork >= left.ProductionCounts.TotalWork && observation.AllCacheMisses
		}
	case ControlProposalOrder, ControlMutationInert:
		if input.PairedTeacher == nil {
			return ControlObservation{}, errors.New("control requires an independent paired teacher")
		}
		leftOptions := controlRunOptions{}
		if name == ControlMutationInert {
			leftOptions.mutationProbe = true
		}
		left, err := executeControlEpisode(ctx, input.FixtureBytes, input.ProfileBytes, primary, leftOptions)
		if err != nil {
			return ControlObservation{}, err
		}
		paired := &countingTeacher{inner: input.PairedTeacher}
		options := controlRunOptions{}
		if name == ControlProposalOrder {
			options.reverseProposals = true
		}
		if name == ControlMutationInert {
			options.mutation = true
			options.mutationProbe = true
		}
		right, err := executeControlEpisode(ctx, input.FixtureBytes, input.ProfileBytes, paired, options)
		if err != nil {
			return ControlObservation{}, err
		}
		if err := setPair(left, input.FixtureBytes, right, input.FixtureBytes); err != nil {
			return ControlObservation{}, err
		}
		observation.TeacherCallsAfter = primary.calls + paired.calls
		observation.Passed = semanticControlEqual(observation.Treatment, observation.Control)
		if name == ControlMutationInert {
			observation.TreatmentMutation = left.Mutation
			observation.ControlMutation = right.Mutation
			observation.Passed = observation.Passed &&
				!left.Mutation.Config.Enabled && len(left.Mutation.Mutants) == 0 &&
				right.Mutation.Config.Enabled && len(right.Mutation.Mutants) >= 1
			observation.Observed = fmt.Sprintf(
				"semantic-projection-equal;off-mutants=%d;on-mutants=%d",
				len(left.Mutation.Mutants), len(right.Mutation.Mutants),
			)
		} else {
			observation.Observed = "semantic-projection-equal"
		}
	case ControlChildVM:
		runner, err := NewEpisode(input.FixtureBytes, input.ProfileBytes, primary)
		if err != nil {
			return ControlObservation{}, err
		}
		defer runner.Close()
		left, err := runner.Run(ctx)
		if err != nil {
			return ControlObservation{}, err
		}
		observation.Treatment, err = controlResult(left, input.FixtureBytes)
		if err != nil {
			return ControlObservation{}, err
		}
		beforeArtifacts := len(runner.artifacts)
		beforeCounts := runner.meter.Counts()
		beforeTeacher := primary.calls
		probeErr := dsl.ProbeCausalChildTaskDenial(runner.engine.VM, runner.runtimeName, proposalTaskSlot)
		afterCounts := runner.meter.Counts()
		observation.ChildVM = ChildVMEvidence{
			FixtureDigest: fixture.FixtureDigest, ProfileDigest: profile.ProfileDigest,
			Operation: "causal-v2-task-valid?", ArtifactsBefore: beforeArtifacts, ArtifactsAfter: len(runner.artifacts),
			MeterCountsBefore: beforeCounts, MeterCountsAfter: afterCounts,
			TeacherCallsBefore: beforeTeacher, TeacherCallsAfter: primary.calls, FailureCode: "child-vm-unauthorized",
		}
		observation.Control = emptyControlResult(profile.ProfileDigest, "child-vm-unauthorized")
		observation.Counts = left.ProductionCounts
		observation.TeacherCallsAfter = primary.calls
		observation.Passed = probeErr != nil && strings.Contains(probeErr.Error(), "child-vm-unauthorized") &&
			beforeArtifacts == len(runner.artifacts) && beforeCounts == afterCounts && beforeTeacher == primary.calls
		observation.Observed = "fail-closed:child-vm-unauthorized"
	case ControlCostPerturbation:
		if err := pairedRequired(input); err != nil {
			return ControlObservation{}, err
		}
		if len(input.PairedFixtureBytes) == 0 || len(input.PairedProfileBytes) == 0 {
			return ControlObservation{}, errors.New("cost perturbation requires an independently signed paired context")
		}
		if stale, staleErr := NewEpisode(input.PairedFixtureBytes, input.ProfileBytes, primary); staleErr == nil {
			stale.Close()
			return ControlObservation{}, errors.New("cost perturbation accepted stale profile")
		}
		left, right, err := runPair(controlRunOptions{}, controlRunOptions{})
		if err != nil {
			return ControlObservation{}, err
		}
		if err := setPair(left, input.FixtureBytes, right, input.PairedFixtureBytes); err != nil {
			return ControlObservation{}, err
		}
		observation.Passed = slices.Equal(observation.Treatment.Actions, observation.Control.Actions) && slices.Equal(observation.Treatment.Outcomes, observation.Control.Outcomes) && !slices.Equal(observation.Treatment.Costs, observation.Control.Costs)
		observation.Observed = "stale-rejected-fresh-recomputed"
	case ControlOccupiedName:
		if input.PairedTeacher == nil {
			return ControlObservation{}, errors.New("occupied-name requires a baseline teacher")
		}
		baseline, err := NewEpisode(input.FixtureBytes, input.ProfileBytes, input.PairedTeacher)
		if err != nil {
			return ControlObservation{}, err
		}
		if _, err := baseline.AdvanceToTeacher(ctx); err != nil {
			baseline.Close()
			return ControlObservation{}, err
		}
		action := baseline.proposalActions[0]
		collision := baseline.current[action].proposal
		baseline.Close()
		target, err := NewEpisode(input.FixtureBytes, input.ProfileBytes, primary)
		if err != nil {
			return ControlObservation{}, err
		}
		occupied := unit.New(artifactName(collision.Kind, collision.Digest))
		occupied.Set("sealed", false)
		target.store.Put(occupied)
		observation.TeacherCallsBefore = primary.calls
		_, trialErr := target.AdvanceToTeacher(ctx)
		observation.TeacherCallsAfter = primary.calls
		observation.Counts = target.meter.Counts()
		target.Close()
		observation.Treatment = emptyControlResult(profile.ProfileDigest, "occupied-name-rejected")
		observation.Control = emptyControlResult("", "")
		observation.Passed = trialErr != nil && observation.TeacherCallsAfter == observation.TeacherCallsBefore
		observation.Observed = "fail-closed:occupied-name-rejected"
	case ControlAlternateDescriptor:
		target, err := NewEpisode(input.FixtureBytes, input.ProfileBytes, primary)
		if err != nil {
			return ControlObservation{}, err
		}
		target.cursor.latestSnapshotDigest = causalv2.ZeroDigest
		target.syncCursor()
		observation.TeacherCallsBefore = primary.calls
		_, trialErr := target.AdvanceToTeacher(ctx)
		observation.TeacherCallsAfter = primary.calls
		observation.Counts = target.meter.Counts()
		target.Close()
		observation.Treatment = emptyControlResult(profile.ProfileDigest, "alternate-descriptor-rejected")
		observation.Control = emptyControlResult("", "")
		observation.Passed = trialErr != nil && observation.TeacherCallsAfter == observation.TeacherCallsBefore
		observation.Observed = "fail-closed:alternate-descriptor-rejected"
	case ControlStaleResponse, ControlDuplicateResponse:
		target, err := NewEpisode(input.FixtureBytes, input.ProfileBytes, primary)
		if err != nil {
			return ControlObservation{}, err
		}
		if _, err := target.AdvanceToTeacher(ctx); err != nil {
			target.Close()
			return ControlObservation{}, err
		}
		if err := target.Respond(ctx); err != nil {
			target.Close()
			return ControlObservation{}, err
		}
		observation.TeacherCallsBefore = primary.calls
		trialErr := target.Respond(ctx)
		observation.TeacherCallsAfter = primary.calls
		observation.Counts = target.meter.Counts()
		target.Close()
		failure := string(name) + "-rejected"
		observation.Treatment = emptyControlResult(profile.ProfileDigest, failure)
		observation.Control = emptyControlResult("", "")
		observation.Passed = trialErr != nil && observation.TeacherCallsAfter == observation.TeacherCallsBefore
		observation.Observed = "fail-closed:" + failure
	case ControlCorruptionSuite:
		var artifacts [][]byte
		var result EpisodeResult
		if len(input.BaselineArtifacts) != 0 {
			artifacts = cloneCanonicalBytes(input.BaselineArtifacts)
		} else {
			result, err = executeControlEpisode(ctx, input.FixtureBytes, input.ProfileBytes, primary, controlRunOptions{})
			if err != nil {
				return ControlObservation{}, err
			}
			artifacts = cloneCanonicalBytes(result.Artifacts)
			observation.Counts = result.ProductionCounts
		}
		if len(artifacts) == 0 {
			return ControlObservation{}, errors.New("corruption control has no artifacts")
		}
		if _, err := VerifyEpisode(input.FixtureBytes, input.ProfileBytes, artifacts); err != nil {
			return ControlObservation{}, fmt.Errorf("corruption baseline does not verify: %w", err)
		}
		cases, err := enumerateCorruptionCases(artifacts)
		if err != nil {
			return ControlObservation{}, err
		}
		observation.TeacherCallsBefore = primary.calls
		verificationMeter := WorkMeter{dynamic: true}
		accepted := ""
		for _, trial := range cases {
			if err := verificationMeter.chargeProfile(1); err != nil {
				return ControlObservation{}, err
			}
			if err := verificationMeter.chargeArtifact(len(trial.artifacts)); err != nil {
				return ControlObservation{}, err
			}
			if _, trialErr := VerifyEpisode(input.FixtureBytes, input.ProfileBytes, trial.artifacts); trialErr == nil {
				accepted = trial.name
				break
			}
		}
		observation.TeacherCallsAfter = primary.calls
		observation.Treatment = emptyControlResult(profile.ProfileDigest, "corruption-suite-rejected")
		observation.Control = emptyControlResult("", "")
		observation.Counts = addCounts(observation.Counts, verificationMeter.Counts())
		observation.Passed = accepted == "" && observation.TeacherCallsAfter == observation.TeacherCallsBefore
		observation.Observed = fmt.Sprintf("fail-closed:corruption-suite-rejected;cases=%d", len(cases))
		if accepted != "" {
			observation.Observed += ";accepted=" + accepted
		}
	case ControlDeterministicJSON:
		if input.PairedTeacher == nil {
			return ControlObservation{}, errors.New("deterministic-json requires an independent paired teacher")
		}
		left, err := executeControlEpisode(ctx, input.FixtureBytes, input.ProfileBytes, primary, controlRunOptions{})
		if err != nil {
			return ControlObservation{}, err
		}
		paired := &countingTeacher{inner: input.PairedTeacher}
		right, err := executeControlEpisode(ctx, input.FixtureBytes, input.ProfileBytes, paired, controlRunOptions{})
		if err != nil {
			return ControlObservation{}, err
		}
		if err := setPair(left, input.FixtureBytes, right, input.FixtureBytes); err != nil {
			return ControlObservation{}, err
		}
		observation.TeacherCallsAfter = primary.calls + paired.calls
		observation.Passed = artifactBytesEqual(left.Artifacts, right.Artifacts) && observation.Treatment.ProfileDigest == observation.Control.ProfileDigest && observation.Treatment.TranscriptDigest == observation.Control.TranscriptDigest
		observation.Observed = "canonical-bytes-equal"
	default:
		return ControlObservation{}, fmt.Errorf("unknown causal online control %q", name)
	}
	if err := observation.Counts.ValidateEquation(); err != nil {
		return ControlObservation{}, err
	}
	return observation, nil
}

func allTranscriptCacheMisses(artifacts [][]byte) bool {
	found := false
	for _, encoded := range artifacts {
		artifact, err := causalv2.VerifyArtifact(encoded)
		if err != nil || artifact.Kind != "transcript" {
			continue
		}
		entry, err := causalv2.StrictDecode[causalv2.TranscriptEntry](artifact.Payload)
		if err != nil || entry.CacheStatus != "miss" {
			return false
		}
		found = true
	}
	return found
}

func artifactBytesEqual(left, right [][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !bytes.Equal(left[index], right[index]) {
			return false
		}
	}
	return true
}

type corruptionTrial struct {
	name      string
	artifacts [][]byte
}

func enumerateCorruptionCases(baseline [][]byte) ([]corruptionTrial, error) {
	const episodeKindCount = 15
	firstByKind := make(map[string]int, episodeKindCount)
	kinds := make([]string, len(baseline))
	for index, encoded := range baseline {
		artifact, err := causalv2.VerifyArtifact(encoded)
		if err != nil {
			return nil, fmt.Errorf("baseline artifact %d: %w", index, err)
		}
		kinds[index] = artifact.Kind
		if _, exists := firstByKind[artifact.Kind]; !exists {
			firstByKind[artifact.Kind] = index
		}
	}
	var trials []corruptionTrial
	for _, kind := range causalv2.ArtifactKinds[:episodeKindCount] {
		index, ok := firstByKind[kind]
		if !ok {
			return nil, fmt.Errorf("corruption baseline lacks artifact kind %q", kind)
		}
		encoded := baseline[index]
		var envelope map[string]any
		if err := json.Unmarshal(encoded, &envelope); err != nil {
			return nil, err
		}
		outerFields := make([]string, 0, len(envelope))
		for field := range envelope {
			outerFields = append(outerFields, field)
		}
		sort.Strings(outerFields)
		for _, field := range outerFields {
			changed := cloneJSONMap(envelope)
			changed[field] = corruptJSONValue(changed[field])
			candidate := cloneCanonicalBytes(baseline)
			candidate[index], _ = json.Marshal(changed)
			trials = append(trials, corruptionTrial{fmt.Sprintf("kind-%s-field-%s", kind, field), candidate})
		}
		payload, ok := envelope["payload"].(map[string]any)
		if ok {
			for _, payloadMutation := range jsonFieldCorruptions(payload, "payload") {
				changed := cloneJSONMap(envelope)
				changed["payload"] = payloadMutation.value
				candidate := cloneCanonicalBytes(baseline)
				candidate[index], _ = json.Marshal(changed)
				trials = append(trials, corruptionTrial{fmt.Sprintf("kind-%s-%s", kind, payloadMutation.path), candidate})
			}
		}

		deleted := cloneCanonicalBytes(baseline)
		deleted = append(deleted[:index], deleted[index+1:]...)
		trials = append(trials, corruptionTrial{fmt.Sprintf("delete-kind-%s", kind), deleted})
		duplicated := cloneCanonicalBytes(baseline)
		duplicated = append(duplicated[:index], append([][]byte{append([]byte(nil), baseline[index]...)}, duplicated[index:]...)...)
		trials = append(trials, corruptionTrial{fmt.Sprintf("duplicate-kind-%s", kind), duplicated})

		artifact, err := causalv2.VerifyArtifact(encoded)
		if err != nil {
			return nil, err
		}
		forged, err := causalv2.NewArtifact(artifact.ProfileDigest, artifact.Scope, artifact.Step, artifact.Kind, artifact.Payload, artifact.ChargeIndex+1)
		if err != nil {
			return nil, err
		}
		forgedBytes, err := causalv2.CanonicalJSON(forged)
		if err != nil {
			return nil, err
		}
		candidate := cloneCanonicalBytes(baseline)
		candidate[index] = forgedBytes
		trials = append(trials, corruptionTrial{fmt.Sprintf("forge-kind-%s", kind), candidate})
	}
	seenPairs := make(map[string]bool)
	for index := 0; index+1 < len(baseline); index++ {
		left, right := kinds[index], kinds[index+1]
		pair := left + "\x00" + right
		if seenPairs[pair] {
			continue
		}
		seenPairs[pair] = true
		candidate := cloneCanonicalBytes(baseline)
		candidate[index], candidate[index+1] = candidate[index+1], candidate[index]
		trials = append(trials, corruptionTrial{fmt.Sprintf("reorder-kind-%s-%s", left, right), candidate})
	}
	if len(trials) > 486 {
		return nil, fmt.Errorf("closed corruption enumeration has %d cases, cap is 486", len(trials))
	}
	return trials, nil
}

// CorruptionCaseEvidence is the compact, independently reconstructible record
// for one closed corruption case. The mutation descriptor is deliberately the
// canonical case name; mutated ledgers are reconstructed rather than retained.
type CorruptionCaseEvidence struct {
	Name               string `json:"name"`
	MutationDescriptor string `json:"mutation_descriptor"`
	MutatedBytesDigest string `json:"mutated_bytes_digest"`
	RejectionCode      string `json:"rejection_code"`
	MeterCounts        Counts `json:"meter_counts"`
}

func corruptionLedgerDigest(artifacts [][]byte) (string, error) {
	canonical, err := causalv2.CanonicalJSON(artifacts)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

// CorruptionCases runs the frozen representative enumerator and returns exact
// evidence for every rejected mutation in canonical order.
func CorruptionCases(fixtureBytes, profileBytes []byte, baseline [][]byte) ([]CorruptionCaseEvidence, error) {
	if _, err := VerifyEpisode(fixtureBytes, profileBytes, baseline); err != nil {
		return nil, fmt.Errorf("corruption baseline: %w", err)
	}
	trials, err := enumerateCorruptionCases(baseline)
	if err != nil {
		return nil, err
	}
	evidence := make([]CorruptionCaseEvidence, len(trials))
	for index, trial := range trials {
		meter := WorkMeter{dynamic: true}
		if err := meter.chargeProfile(1); err != nil {
			return nil, err
		}
		if err := meter.chargeArtifact(len(trial.artifacts)); err != nil {
			return nil, err
		}
		if _, verifyErr := VerifyEpisode(fixtureBytes, profileBytes, trial.artifacts); verifyErr == nil {
			return nil, fmt.Errorf("corruption case %q was accepted", trial.name)
		}
		digest, err := corruptionLedgerDigest(trial.artifacts)
		if err != nil {
			return nil, err
		}
		evidence[index] = CorruptionCaseEvidence{
			Name: trial.name, MutationDescriptor: trial.name, MutatedBytesDigest: digest,
			RejectionCode: "corruption-rejected", MeterCounts: meter.Counts(),
		}
	}
	return evidence, nil
}

// VerifyCorruptionCases reconstructs the entire closed case set and rejects a
// subset, extension, reorder, altered digest, failure code, or meter count.
func VerifyCorruptionCases(fixtureBytes, profileBytes []byte, baseline [][]byte, evidence []CorruptionCaseEvidence) (Counts, error) {
	want, err := CorruptionCases(fixtureBytes, profileBytes, baseline)
	if err != nil {
		return Counts{}, err
	}
	if !slices.Equal(evidence, want) {
		return Counts{}, errors.New("corruption evidence differs from exact closed enumeration")
	}
	var aggregate Counts
	for _, item := range want {
		aggregate = addCounts(aggregate, item.MeterCounts)
	}
	return aggregate, nil
}

// CorruptionCaseNames exposes the closed enumerator's exact ordered case set
// without exposing mutation callbacks or allowing callers to redefine it.
func CorruptionCaseNames(baseline [][]byte) ([]string, error) {
	trials, err := enumerateCorruptionCases(baseline)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(trials))
	for index, trial := range trials {
		names[index] = trial.name
	}
	return names, nil
}

// VerifyCorruptionSuite reconstructs every closed corruption case from the
// retained baseline and requires each one to fail verification. CaseNames is
// an exact ordered witness, not a caller-selected subset.
func VerifyCorruptionSuite(fixtureBytes, profileBytes []byte, baseline [][]byte, caseNames []string) (Counts, error) {
	evidence, err := CorruptionCases(fixtureBytes, profileBytes, baseline)
	if err != nil {
		return Counts{}, err
	}
	wantNames := make([]string, len(evidence))
	for index, item := range evidence {
		wantNames[index] = item.Name
	}
	if !slices.Equal(caseNames, wantNames) {
		return Counts{}, errors.New("corruption case names differ from exact closed enumeration")
	}
	return VerifyCorruptionCases(fixtureBytes, profileBytes, baseline, evidence)
}

func cloneJSONMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

type jsonFieldCorruption struct {
	path  string
	value map[string]any
}

// jsonFieldCorruptions changes every named object field and descends through
// the first representative object in arrays, covering nested partition-cell
// fields without multiplying equivalent attacks by every array index.
func jsonFieldCorruptions(source map[string]any, prefix string) []jsonFieldCorruption {
	fields := make([]string, 0, len(source))
	for field := range source {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	var result []jsonFieldCorruption
	for _, field := range fields {
		path := prefix + "." + field
		direct := cloneJSONMap(source)
		direct[field] = corruptJSONValue(direct[field])
		result = append(result, jsonFieldCorruption{path: path, value: direct})
		switch child := source[field].(type) {
		case map[string]any:
			for _, nested := range jsonFieldCorruptions(child, path) {
				changed := cloneJSONMap(source)
				changed[field] = nested.value
				result = append(result, jsonFieldCorruption{path: nested.path, value: changed})
			}
		case []any:
			if len(child) == 0 {
				continue
			}
			object, ok := child[0].(map[string]any)
			if !ok {
				continue
			}
			for _, nested := range jsonFieldCorruptions(object, path+"[0]") {
				changed := cloneJSONMap(source)
				array := append([]any(nil), child...)
				array[0] = nested.value
				changed[field] = array
				result = append(result, jsonFieldCorruption{path: nested.path, value: changed})
			}
		}
	}
	return result
}

func corruptJSONValue(value any) any {
	switch typed := value.(type) {
	case string:
		return typed + "-corrupt"
	case float64:
		return typed + 1
	case bool:
		return !typed
	case []any:
		if len(typed) == 0 {
			return []any{"corrupt"}
		}
		changed := append([]any(nil), typed...)
		changed[0] = corruptJSONValue(changed[0])
		return changed
	case map[string]any:
		changed := cloneJSONMap(typed)
		changed["unexpected_corruption_field"] = true
		return changed
	default:
		return "corrupt"
	}
}

func cloneCanonicalBytes(source [][]byte) [][]byte {
	result := make([][]byte, len(source))
	for index := range source {
		result[index] = append([]byte(nil), source[index]...)
	}
	return result
}
