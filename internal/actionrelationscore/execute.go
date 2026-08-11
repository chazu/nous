package actionrelationscore

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationacquire"
	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationfixturecore"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/actionrelationutility"
	"github.com/chazu/nous/internal/dsl"
	actionrelations "github.com/chazu/nous/internal/vocab/actionrelations"
)

type Acquisition struct {
	Evidence actionrelationexp.AcquisitionEvidence
	Boundary actionrelationexp.AcquisitionBoundary
}

type CurriculumResult struct {
	Panel          string
	Authority      string
	Curriculum     int
	Family         int
	Nous           Acquisition
	NoGuard        Acquisition
	Runs           map[actionrelationsearch.Policy][]actionrelationutility.SearchRun
	WorldRows      []WorldPolicyRow
	CurriculumRows []CurriculumPolicyRow
	OperationRoots map[string]actionrelationexp.OperationRoot
}

func ExecuteCurriculum(domainsDir string, generated actionrelationfixture.GeneratedAttempt) (CurriculumResult, error) {
	context := generated.Context
	// This exported helper is deliberately development-only. Validation and
	// locked policy execution will be reachable solely from the staged guarded
	// panel caller after implementation acceptance.
	if context.Panel != "development" || context.Authority != "development-public-v1" || actionrelationfixture.VerifyCurriculumFixture(generated.Fixture) != nil || generated.Fixture.Panel != context.Panel || generated.Fixture.Curriculum != context.Curriculum || generated.Curriculum.Family != context.Curriculum%8 || len(generated.Curriculum.Worlds) != 6 || len(generated.Truth.Worlds) != 6 {
		return CurriculumResult{}, fmt.Errorf("invalid scorer curriculum authority")
	}
	return executeCurriculum(domainsDir, generated)
}

func executeCurriculum(domainsDir string, generated actionrelationfixture.GeneratedAttempt) (CurriculumResult, error) {
	training, err := actionrelationfixturecore.PublicCases(generated.Training)
	if err != nil {
		return CurriculumResult{}, err
	}
	worlds := make([]PublicWorld, len(generated.Curriculum.Worlds))
	for index, view := range generated.Curriculum.Worlds {
		worlds[index] = PublicWorld{State: view.State, Actions: slices.Clone(view.Actions)}
	}
	public := PublicCurriculum{Curriculum: generated.Context.Curriculum, Training: training, Worlds: worlds}
	result, err := executePolicyCurriculum(domainsDir, generated.Context.Panel, generated.Context.Authority, public)
	if err != nil {
		return result, err
	}
	return scorePolicyCurriculum(generated, result)
}

func executePolicyCurriculum(domainsDir, panel, authority string, public PublicCurriculum) (CurriculumResult, error) {
	result := CurriculumResult{
		Panel: panel, Authority: authority, Curriculum: public.Curriculum,
		Runs:           map[actionrelationsearch.Policy][]actionrelationutility.SearchRun{},
		OperationRoots: map[string]actionrelationexp.OperationRoot{},
	}
	if len(public.Training) != actionrelationfixturecore.TrainingCount || len(public.Worlds) != 6 {
		return result, fmt.Errorf("invalid public policy curriculum")
	}
	nous, err := executeAcquisition(domainsDir, panel, authority, public.Curriculum, public.Training, "nous")
	if err != nil {
		return result, err
	}
	result.Nous = nous
	noGuard, err := executeAcquisition(domainsDir, panel, authority, public.Curriculum, public.Training, "no-guard")
	if err != nil {
		return result, err
	}
	result.NoGuard = noGuard
	result.OperationRoots[nous.Evidence.Transcript.RunRoot.Digest] = nous.Evidence.Transcript.RunRoot
	result.OperationRoots[noGuard.Evidence.Transcript.RunRoot.Digest] = noGuard.Evidence.Transcript.RunRoot

	for policyOrdinal, policy := range Policies {
		acquisition := acquisitionForPolicy(result, policy)
		acquisitionVector, _, _, _, _, err := acquisitionFields(acquisition, policy)
		if err != nil {
			return result, fmt.Errorf("policy %s acquisition: %w", policy, err)
		}
		lifecycle := acquisitionVector
		priorPhysical, histories := 0, 0
		worldRuns := make([]actionrelationutility.SearchRun, 6)
		for worldOrdinal, view := range public.Worlds {
			world := actionrelations.World{State: view.State, Actions: view.Actions}
			budget := actionrelationutility.WorkBudget{LifecycleCap: LifecycleCap, PhysicalCap: policyPhysicalCap(policy), PriorPhysical: priorPhysical, ReservedTerminals: 5 - worldOrdinal}
			token := fmt.Sprintf("policy-c%04d-p%02d-w%d", public.Curriculum, policyOrdinal, worldOrdinal)
			var run actionrelationutility.SearchRun
			switch policy {
			case actionrelationsearch.NousSleep, actionrelationsearch.NoGuardSleep, actionrelationsearch.LearnedNoUse:
				run, err = actionrelationutility.ExecuteLearnedPolicyWithBudget(acquisition.Evidence.Run.Store, acquisition.Evidence.Run.Artifact, acquisition.Boundary.BoundaryUnit, world, policy, panel, authority, public.Curriculum, worldOrdinal, lifecycle, budget, token)
			default:
				run, err = actionrelationutility.ExecutePolicyContinuing(domainsDir, world, policy, panel, authority, public.Curriculum, worldOrdinal, lifecycle, budget, token)
			}
			if err != nil {
				return result, fmt.Errorf("policy %s world %d: %w", policy, worldOrdinal, err)
			}
			lifecycle = run.WorkVector
			priorPhysical += run.PhysicalWork
			histories += run.Search.HistoryCount
			if histories > HistoryCap {
				return result, fmt.Errorf("policy %s crossed curriculum history cap", policy)
			}
			worldRuns[worldOrdinal] = run
			result.OperationRoots[run.RunRoot.Digest] = run.RunRoot
		}
		result.Runs[policy] = worldRuns
	}
	return result, nil
}

func scorePolicyCurriculum(generated actionrelationfixture.GeneratedAttempt, result CurriculumResult) (CurriculumResult, error) {
	context := generated.Context
	if result.Panel != context.Panel || result.Authority != context.Authority || result.Curriculum != context.Curriculum || len(generated.Truth.Worlds) != 6 || len(generated.Curriculum.Worlds) != 6 {
		return result, fmt.Errorf("private scorer and public policy authority differ")
	}
	result.Family = generated.Curriculum.Family
	for _, policy := range Policies {
		acquisition := acquisitionForPolicy(result, policy)
		acquisitionVector, terminal, artifactDigest, acquisitionTerminalDigest, acquisitionRoot, err := acquisitionFields(acquisition, policy)
		if err != nil {
			return result, fmt.Errorf("policy %s acquisition: %w", policy, err)
		}
		training := trainingMatchCounts(acquisition)
		lifecycle, searchVector := acquisitionVector, [12]int{}
		behaviorEqual, aggregateTerminal := true, terminal
		children := make([]string, 0, 7)
		if acquisitionRoot != "" {
			children = append(children, acquisitionRoot)
		}
		worldRows := make([]WorldPolicyRow, 6)
		for worldOrdinal, view := range generated.Curriculum.Worlds {
			run := result.Runs[policy][worldOrdinal]
			currentVector, err := actionrelationutility.MeterWorkVector(run.Records)
			if err != nil {
				return result, err
			}
			addVector(&searchVector, currentVector)
			lifecycle = run.WorkVector
			truth := generated.Truth.Worlds[worldOrdinal]
			equal := run.Terminal == "completed" && slices.Equal(run.Search.TerminalDigests, truth.Terminals)
			behaviorEqual = behaviorEqual && equal
			matchCounts, err := utilityMatchCounts(run.Records, truth, training)
			if err != nil {
				return result, fmt.Errorf("policy %s world %d match counts: %w", policy, worldOrdinal, err)
			}
			workTerminal, terminalSet, remaining := zeroDigest, zeroDigest, LifecycleCap-run.WorkTotal
			if run.Terminal == "budget-exhausted" {
				workTerminal, remaining = run.WorkTerminal.Digest, 0
				if aggregateTerminal == "completed" || aggregateTerminal == "not-applicable" {
					aggregateTerminal = "budget-exhausted"
				}
			} else {
				terminalSet = run.Search.TerminalSet.Digest
			}
			row, err := BuildWorldPolicyRow(WorldPolicyRow{
				Panel: context.Panel, Curriculum: context.Curriculum, Family: result.Family, WorldOrdinal: worldOrdinal,
				Stratum: view.Stratum, WorldDigest: truth.WorldDigest, Policy: policy, SearchTerminal: run.Terminal,
				UtilityWorkVector: currentVector, UtilityTotal: sum(currentVector), MatchCounts: matchCounts,
				CertificateCounts: certificateCounts(run.Records), SleepCount: run.Search.SleepPropagations,
				HistoryCount: run.Search.HistoryCount, TerminalSetDigest: terminalSet,
				WorkTerminalDigestOrZero: workTerminal, BehaviorEqual: equal, BudgetRemaining: remaining,
				OperationRoot: run.RunRoot,
			})
			if err != nil {
				return result, err
			}
			worldRows[worldOrdinal] = row
			children = append(children, run.RunRoot.Digest)
		}
		if aggregateTerminal == "not-applicable" || aggregateTerminal == "completed" || aggregateTerminal == "no-discovery" {
			aggregateTerminal = "completed"
		}
		worldDigests := make([]string, 6)
		for index, row := range worldRows {
			worldDigests[index] = row.Digest
		}
		contextWire, _ := json.Marshal([]any{
			"actionrelation-curriculum-policy-operation-context/v1", context.Panel, context.Curriculum, result.Family, string(policy),
			terminal, artifactDigest, acquisitionVector, acquisitionTerminalDigest, worldDigests, aggregateTerminal,
			searchVector, sum(searchVector), lifecycle, sum(lifecycle), behaviorEqual,
		})
		operationRoot, err := actionrelationexp.BuildOperationConcat(digest(contextWire), children)
		if err != nil {
			return result, err
		}
		remaining := LifecycleCap - sum(lifecycle)
		if aggregateTerminal == "budget-exhausted" {
			remaining = 0
		}
		curriculumRow, err := BuildCurriculumPolicyRow(CurriculumPolicyRow{
			Panel: context.Panel, Curriculum: context.Curriculum, Family: result.Family, Policy: policy,
			AcquisitionTerminal: terminal, ArtifactDigest: artifactDigest, AcquisitionWorkVector: acquisitionVector,
			AcquisitionWorkTerminalDigestOrZero: acquisitionTerminalDigest, WorldRowDigests: worldDigests,
			AggregateTerminal: aggregateTerminal, SearchWorkVector: searchVector, SearchTotal: sum(searchVector),
			LifecycleWorkVector: lifecycle, LifecycleTotal: sum(lifecycle), BehaviorEqual: behaviorEqual,
			BudgetRemaining: remaining, OperationRoot: operationRoot,
		})
		if err != nil {
			return result, err
		}
		result.WorldRows = append(result.WorldRows, worldRows...)
		result.CurriculumRows = append(result.CurriculumRows, curriculumRow)
		result.OperationRoots[operationRoot.Digest] = operationRoot
	}
	// Canonical panel order is world ordinal then policy, not execution order.
	slices.SortFunc(result.WorldRows, func(a, b WorldPolicyRow) int {
		if a.WorldOrdinal != b.WorldOrdinal {
			return a.WorldOrdinal - b.WorldOrdinal
		}
		return policyIndex(a.Policy) - policyIndex(b.Policy)
	})
	return result, nil
}

func executeAcquisition(domainsDir, panel, authority string, curriculum int, training []actionrelationfixturecore.PublicCase, scope string) (Acquisition, error) {
	token := fmt.Sprintf("score-acquire-%s-c%04d", scope, curriculum)
	var session *actionrelationacquire.Session
	var err error
	if scope == "nous" {
		session, err = actionrelationacquire.BeginPublicFor(domainsDir, token, training, scope, panel, authority, curriculum)
	} else {
		session, err = actionrelationacquire.BeginPublicFor(domainsDir, token, training, scope, panel, authority, curriculum)
	}
	if err != nil {
		return Acquisition{}, err
	}
	var evidence actionrelationexp.AcquisitionEvidence
	if scope == "nous" {
		evidence, err = actionrelationexp.CompleteAcquisitionFor(session, curriculum, panel, authority)
	} else {
		evidence, err = actionrelationexp.CompleteNoGuardAcquisitionFor(session, curriculum, panel, authority)
	}
	if err != nil {
		return Acquisition{}, err
	}
	evidenceRoot, err := actionrelationexp.EvidenceRoot(panel)
	if err != nil {
		return Acquisition{}, err
	}
	boundary, err := actionrelationexp.BuildAcquisitionBoundaryAt(evidenceRoot, evidence, curriculum, scope)
	if err != nil {
		return Acquisition{}, err
	}
	return Acquisition{Evidence: evidence, Boundary: boundary}, nil
}

func acquisitionForPolicy(result CurriculumResult, policy actionrelationsearch.Policy) Acquisition {
	switch policy {
	case actionrelationsearch.NousSleep, actionrelationsearch.LearnedNoUse:
		return result.Nous
	case actionrelationsearch.NoGuardSleep:
		return result.NoGuard
	default:
		return Acquisition{}
	}
}

func acquisitionFields(acquisition Acquisition, policy actionrelationsearch.Policy) ([12]int, string, string, string, string, error) {
	if !slices.Contains([]actionrelationsearch.Policy{actionrelationsearch.NousSleep, actionrelationsearch.NoGuardSleep, actionrelationsearch.LearnedNoUse}, policy) {
		return [12]int{}, "not-applicable", zeroDigest, zeroDigest, "", nil
	}
	if acquisition.Evidence.Run.Store == nil || acquisition.Evidence.Run.Experiment == "" {
		return [12]int{}, "", "", "", "", fmt.Errorf("missing acquisition")
	}
	vector, err := actionrelationutility.MeterWorkVector(acquisition.Evidence.Run.MeterRecords)
	if err != nil {
		return [12]int{}, "", "", "", "", err
	}
	experiment := acquisition.Evidence.Run.Store.Get(acquisition.Evidence.Run.Experiment)
	artifact := acquisition.Evidence.Run.Store.Get(acquisition.Evidence.Run.Artifact)
	if experiment == nil || artifact == nil || experiment.GetString("terminal") != "completed" || !digestText(artifact.GetString("objectDigest")) {
		return [12]int{}, "", "", "", "", fmt.Errorf("acquisition did not complete")
	}
	return vector, "completed", artifact.GetString("objectDigest"), zeroDigest, acquisition.Evidence.Transcript.RunRoot.Digest, nil
}

func trainingMatchCounts(acquisition Acquisition) MatchCounts {
	if acquisition.Evidence.Run.Store == nil {
		return MatchCounts{}
	}
	experiment := acquisition.Evidence.Run.Store.Get(acquisition.Evidence.Run.Experiment)
	if experiment == nil || len(experiment.GetStrings("winnerResultUnits")) == 0 {
		return MatchCounts{}
	}
	winner := acquisition.Evidence.Run.Store.Get(experiment.GetStrings("winnerResultUnits")[0])
	if winner == nil {
		return MatchCounts{}
	}
	tp, fp := winner.GetInt("positiveCoverage"), winner.GetInt("falseMatches")
	return MatchCounts{TrainingPositive: 8, TrainingNegative: 8, TrainingTruePositive: tp, TrainingFalsePositive: fp, TrainingFalseNegative: 8 - tp}
}

func utilityMatchCounts(records []dsl.ActionRelationMeterRecord, truth actionrelationfixture.WorldTruth, base MatchCounts) (MatchCounts, error) {
	trace, err := collectPolicyPairTrace(records)
	if err != nil {
		return MatchCounts{}, err
	}
	return scorePolicyPairTrace(trace, truth, base)
}

func certificateCounts(records []dsl.ActionRelationMeterRecord) CertificateCounts {
	result := CertificateCounts{}
	for _, record := range records {
		if record.Code == 18 && record.Status == 1 {
			result.Attempted++
		}
		if (record.Code != 25 && !(record.Code == 18 && record.Status == 3)) || len(record.Outputs) != 1 {
			continue
		}
		var row []json.RawMessage
		var terminal string
		if json.Unmarshal(record.Outputs[0], &row) != nil || len(row) != 12 || json.Unmarshal(row[9], &terminal) != nil {
			continue
		}
		if record.Code == 25 && terminal == "certified" {
			result.Successful++
		} else if record.Code == 18 && terminal == "certified" {
			result.CachedSuccess++
		} else if record.Code == 18 {
			result.CachedFailure++
		}
	}
	return result
}

func addVector(target *[12]int, value [12]int) {
	for index := range target {
		target[index] += value[index]
	}
}

func policyPhysicalCap(policy actionrelationsearch.Policy) int {
	if policy == actionrelationsearch.DynamicSleep || policy == actionrelationsearch.NousSleep {
		return 8192
	}
	return 4096
}

func policyIndex(policy actionrelationsearch.Policy) int {
	for index, candidate := range Policies {
		if candidate == policy {
			return index
		}
	}
	return -1
}
