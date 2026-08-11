package actionrelationscore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/actionrelationexp"
	"github.com/chazu/nous/internal/actionrelationfixture"
	"github.com/chazu/nous/internal/actionrelationledger"
	"github.com/chazu/nous/internal/actionrelationsearch"
	"github.com/chazu/nous/internal/actionrelationutility"
	"github.com/chazu/nous/internal/dsl"
)

type PolicyPairTrace struct {
	State       string
	Occurrences []string
	Results     []bool
}

type PolicyAcquisitionSummary struct {
	Scope             string
	Terminal          string
	ArtifactDigest    string
	WorkVector        [12]int
	WorkTerminal      string
	OperationRoot     actionrelationexp.OperationRoot
	BoundaryCanonical []byte
	Training          MatchCounts
}

type PolicyWorldSummary struct {
	Policy              actionrelationsearch.Policy
	WorldOrdinal        int
	RunID               string
	WorldDigest         string
	Terminal            string
	TerminalDigests     []string
	TerminalSetDigest   string
	WorkTerminal        string
	UtilityWorkVector   [12]int
	LifecycleVector     [12]int
	WorkTotal           int
	WorkCap             int
	BudgetRemaining     int
	CertificateCounts   CertificateCounts
	SleepCount          int
	HistoryCount        int
	OperationRoot       actionrelationexp.OperationRoot
	PairTrace           []PolicyPairTrace
	RejectedReservation []byte
}

type PolicyCurriculumSummary struct {
	Curriculum   int
	Acquisitions []PolicyAcquisitionSummary
	Worlds       []PolicyWorldSummary
}

func summarizePolicyCurriculum(value CurriculumResult) (PolicyCurriculumSummary, error) {
	result := PolicyCurriculumSummary{Curriculum: value.Curriculum}
	for _, item := range []struct {
		scope string
		value Acquisition
	}{{"nous", value.Nous}, {"no-guard", value.NoGuard}} {
		vector, terminal, artifact, workTerminal, _, err := acquisitionFields(item.value, map[string]actionrelationsearch.Policy{"nous": actionrelationsearch.NousSleep, "no-guard": actionrelationsearch.NoGuardSleep}[item.scope])
		if err != nil {
			return result, err
		}
		result.Acquisitions = append(result.Acquisitions, PolicyAcquisitionSummary{Scope: item.scope, Terminal: terminal, ArtifactDigest: artifact, WorkVector: vector, WorkTerminal: workTerminal, OperationRoot: item.value.Evidence.Transcript.RunRoot, BoundaryCanonical: slices.Clone(item.value.Boundary.Canonical), Training: trainingMatchCounts(item.value)})
	}
	for _, policy := range Policies {
		for worldOrdinal, run := range value.Runs[policy] {
			current, err := actionrelationutility.MeterWorkVector(run.Records)
			if err != nil {
				return result, err
			}
			trace, err := collectPolicyPairTrace(run.Records)
			if err != nil {
				return result, err
			}
			terminalSet, workTerminal, remaining := zeroDigest, zeroDigest, LifecycleCap-run.WorkTotal
			if run.Terminal == "budget-exhausted" {
				workTerminal, remaining = run.WorkTerminal.Digest, 0
			} else {
				terminalSet = run.Search.TerminalSet.Digest
			}
			result.Worlds = append(result.Worlds, PolicyWorldSummary{
				Policy: policy, WorldOrdinal: worldOrdinal, RunID: run.RunID, WorldDigest: run.WorldDigest, Terminal: run.Terminal,
				TerminalDigests: slices.Clone(run.Search.TerminalDigests), TerminalSetDigest: terminalSet, WorkTerminal: workTerminal,
				UtilityWorkVector: current, LifecycleVector: run.WorkVector, WorkTotal: run.WorkTotal, WorkCap: run.WorkCap, BudgetRemaining: remaining,
				CertificateCounts: certificateCounts(run.Records), SleepCount: run.Search.SleepPropagations, HistoryCount: run.Search.HistoryCount,
				OperationRoot: run.RunRoot, PairTrace: trace, RejectedReservation: slices.Clone(run.WorkTerminal.RejectedReservation.Canonical),
			})
		}
	}
	if len(result.Acquisitions) != 2 || len(result.Worlds) != 42 {
		return PolicyCurriculumSummary{}, fmt.Errorf("policy curriculum summary cardinality mismatch")
	}
	return result, nil
}

func collectPolicyPairTrace(records []dsl.ActionRelationMeterRecord) ([]PolicyPairTrace, error) {
	var result []PolicyPairTrace
	var current *PolicyPairTrace
	finish := func() error {
		if current == nil {
			return nil
		}
		if current.State == "" || len(current.Occurrences) != 2 {
			return fmt.Errorf("incomplete learned-pair trace")
		}
		result = append(result, *current)
		current = nil
		return nil
	}
	for _, record := range records {
		if record.Code == 21 {
			if current != nil && len(current.Occurrences) == 2 {
				if err := finish(); err != nil {
					return nil, err
				}
			}
			if current == nil {
				current = &PolicyPairTrace{}
			}
			if len(record.Outputs) != 1 {
				return nil, fmt.Errorf("applicability call lacks row")
			}
			var row []json.RawMessage
			var tag, state, occurrence, status string
			var applicable bool
			if json.Unmarshal(record.Outputs[0], &row) != nil || len(row) != 5 || json.Unmarshal(row[0], &tag) != nil || json.Unmarshal(row[1], &state) != nil || json.Unmarshal(row[2], &occurrence) != nil || json.Unmarshal(row[3], &applicable) != nil || json.Unmarshal(row[4], &status) != nil || tag != "action-applicability-row/v1" || status != "valid" || !applicable {
				return nil, fmt.Errorf("invalid learned-pair applicability row")
			}
			if current.State != "" && current.State != state {
				return nil, fmt.Errorf("learned pair changed state")
			}
			current.State = state
			current.Occurrences = append(current.Occurrences, occurrence)
		}
		if record.Code == 9 {
			if current == nil || len(record.Outputs) != 1 {
				return nil, fmt.Errorf("relation match lacks pair authority")
			}
			var row []json.RawMessage
			var matched bool
			if json.Unmarshal(record.Outputs[0], &row) != nil || len(row) != 12 || json.Unmarshal(row[10], &matched) != nil {
				return nil, fmt.Errorf("invalid relation match row")
			}
			current.Results = append(current.Results, matched)
		}
	}
	if err := finish(); err != nil {
		return nil, err
	}
	return result, nil
}

func scorePolicyPairTrace(trace []PolicyPairTrace, truth actionrelationfixture.WorldTruth, base MatchCounts) (MatchCounts, error) {
	lookup := map[string]string{}
	for _, row := range truth.PairRows {
		lookup[row.StateDigest+row.ADigest+row.BDigest] = row.Label
	}
	for _, current := range trace {
		base.UtilityAttempts++
		matched := len(current.Results) > 0
		for _, value := range current.Results {
			matched = matched && value
		}
		if !matched {
			continue
		}
		base.UtilityMatches++
		a, b := current.Occurrences[0], current.Occurrences[1]
		if a > b {
			a, b = b, a
		}
		label, ok := lookup[current.State+a+b]
		if !ok {
			return MatchCounts{}, fmt.Errorf("matched pair absent from scorer truth")
		}
		if label != "commutes" {
			base.UtilityFalseMatches++
		}
	}
	return base, nil
}

func scorePolicyCurriculumSummary(generated actionrelationfixture.GeneratedAttempt, raw PolicyCurriculumSummary) (CurriculumResult, error) {
	context := generated.Context
	result := CurriculumResult{Panel: context.Panel, Authority: context.Authority, Curriculum: context.Curriculum, Family: generated.Curriculum.Family, OperationRoots: map[string]actionrelationexp.OperationRoot{}}
	if raw.Curriculum != context.Curriculum || len(raw.Acquisitions) != 2 || raw.Acquisitions[0].Scope != "nous" || raw.Acquisitions[1].Scope != "no-guard" || len(raw.Worlds) != 42 {
		return result, fmt.Errorf("private scorer and raw policy summary differ")
	}
	acquisitionFor := func(policy actionrelationsearch.Policy) PolicyAcquisitionSummary {
		switch policy {
		case actionrelationsearch.NousSleep, actionrelationsearch.LearnedNoUse:
			return raw.Acquisitions[0]
		case actionrelationsearch.NoGuardSleep:
			return raw.Acquisitions[1]
		default:
			return PolicyAcquisitionSummary{Terminal: "not-applicable", ArtifactDigest: zeroDigest, WorkTerminal: zeroDigest}
		}
	}
	for policyOrdinal, policy := range Policies {
		acquisition := acquisitionFor(policy)
		lifecycle, searchVector := acquisition.WorkVector, [12]int{}
		priorPhysical := 0
		behaviorEqual, aggregateTerminal := true, acquisition.Terminal
		children := []string{}
		if acquisition.OperationRoot.Digest != "" {
			children = append(children, acquisition.OperationRoot.Digest)
			result.OperationRoots[acquisition.OperationRoot.Digest] = acquisition.OperationRoot
		}
		worldRows := make([]WorldPolicyRow, 6)
		for worldOrdinal, view := range generated.Curriculum.Worlds {
			rowIndex := policyOrdinal*6 + worldOrdinal
			world := raw.Worlds[rowIndex]
			truth := generated.Truth.Worlds[worldOrdinal]
			reservedTerminals := 5 - worldOrdinal
			wantCap := sum(lifecycle) + policyPhysicalCap(policy) - reservedTerminals - priorPhysical
			if lifecycleCap := LifecycleCap - reservedTerminals; lifecycleCap < wantCap {
				wantCap = lifecycleCap
			}
			if world.Policy != policy || world.WorldOrdinal != worldOrdinal || world.WorldDigest != truth.WorldDigest || world.WorkTotal != sum(world.LifecycleVector) || world.WorkCap != wantCap || world.BudgetRemaining < 0 {
				return result, fmt.Errorf("raw policy world identity changed")
			}
			if world.Terminal == "budget-exhausted" {
				rejected, err := actionrelationledger.ParseReservation(world.RejectedReservation)
				if err != nil || actionrelationledger.VerifyReservation(rejected, world.WorkCap) != nil || rejected.Status != "rejected-cap" || rejected.RunID != world.RunID || rejected.Digest != world.WorkTerminal {
					return result, fmt.Errorf("raw rejected reservation does not prove budget exhaustion")
				}
			} else if len(world.RejectedReservation) != 0 || world.WorkTerminal != zeroDigest {
				return result, fmt.Errorf("completed world carries rejected reservation")
			}
			addVector(&searchVector, world.UtilityWorkVector)
			priorPhysical += sum(world.UtilityWorkVector)
			lifecycle = world.LifecycleVector
			equal := world.Terminal == "completed" && slices.Equal(world.TerminalDigests, truth.Terminals)
			behaviorEqual = behaviorEqual && equal
			matches, err := scorePolicyPairTrace(world.PairTrace, truth, acquisition.Training)
			if err != nil {
				return result, err
			}
			built, err := BuildWorldPolicyRow(WorldPolicyRow{
				Panel: context.Panel, Curriculum: context.Curriculum, Family: result.Family, WorldOrdinal: worldOrdinal,
				Stratum: view.Stratum, WorldDigest: truth.WorldDigest, Policy: policy, SearchTerminal: world.Terminal,
				UtilityWorkVector: world.UtilityWorkVector, UtilityTotal: sum(world.UtilityWorkVector), MatchCounts: matches,
				CertificateCounts: world.CertificateCounts, SleepCount: world.SleepCount, HistoryCount: world.HistoryCount,
				TerminalSetDigest: world.TerminalSetDigest, WorkTerminalDigestOrZero: world.WorkTerminal,
				BehaviorEqual: equal, BudgetRemaining: world.BudgetRemaining, OperationRoot: world.OperationRoot,
			})
			if err != nil {
				return result, err
			}
			worldRows[worldOrdinal] = built
			children = append(children, world.OperationRoot.Digest)
			result.OperationRoots[world.OperationRoot.Digest] = world.OperationRoot
			if world.Terminal == "budget-exhausted" && (aggregateTerminal == "completed" || aggregateTerminal == "not-applicable") {
				aggregateTerminal = "budget-exhausted"
			}
		}
		if aggregateTerminal == "not-applicable" || aggregateTerminal == "completed" || aggregateTerminal == "no-discovery" {
			aggregateTerminal = "completed"
		}
		worldDigests := make([]string, 6)
		for index, row := range worldRows {
			worldDigests[index] = row.Digest
		}
		contextWire, _ := json.Marshal([]any{"actionrelation-curriculum-policy-operation-context/v1", context.Panel, context.Curriculum, result.Family, string(policy), acquisition.Terminal, acquisition.ArtifactDigest, acquisition.WorkVector, acquisition.WorkTerminal, worldDigests, aggregateTerminal, searchVector, sum(searchVector), lifecycle, sum(lifecycle), behaviorEqual})
		operationRoot, err := actionrelationexp.BuildOperationConcat(digest(contextWire), children)
		if err != nil {
			return result, err
		}
		remaining := LifecycleCap - sum(lifecycle)
		if aggregateTerminal == "budget-exhausted" {
			remaining = 0
		}
		curriculumRow, err := BuildCurriculumPolicyRow(CurriculumPolicyRow{Panel: context.Panel, Curriculum: context.Curriculum, Family: result.Family, Policy: policy, AcquisitionTerminal: acquisition.Terminal, ArtifactDigest: acquisition.ArtifactDigest, AcquisitionWorkVector: acquisition.WorkVector, AcquisitionWorkTerminalDigestOrZero: acquisition.WorkTerminal, WorldRowDigests: worldDigests, AggregateTerminal: aggregateTerminal, SearchWorkVector: searchVector, SearchTotal: sum(searchVector), LifecycleWorkVector: lifecycle, LifecycleTotal: sum(lifecycle), BehaviorEqual: behaviorEqual, BudgetRemaining: remaining, OperationRoot: operationRoot})
		if err != nil {
			return result, err
		}
		result.WorldRows = append(result.WorldRows, worldRows...)
		result.CurriculumRows = append(result.CurriculumRows, curriculumRow)
		result.OperationRoots[operationRoot.Digest] = operationRoot
	}
	slices.SortFunc(result.WorldRows, func(a, b WorldPolicyRow) int {
		if a.WorldOrdinal != b.WorldOrdinal {
			return a.WorldOrdinal - b.WorldOrdinal
		}
		return policyIndex(a.Policy) - policyIndex(b.Policy)
	})
	return result, nil
}

func equalPolicySummary(left, right PolicyCurriculumSummary) bool {
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return bytes.Equal(a, b)
}
