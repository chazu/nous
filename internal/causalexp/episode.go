package causalexp

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/causaloracle"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

type Policy string

const (
	Learned         Policy = "learned"
	InformationGain Policy = "information-gain-per-cost"
	WorstSplit      Policy = "worst-split-per-cost"
	Lexical         Policy = "lexical-fixed"
	UniformRandom   Policy = "uniform-random"
	PassiveOnly     Policy = "passive-only"
	DynamicOptimal  Policy = "dynamic-optimal"
)

var Policies = []Policy{Learned, InformationGain, WorstSplit, Lexical, UniformRandom, PassiveOnly, DynamicOptimal}

type EpisodeReport struct {
	Seed                  int64    `json:"seed"`
	Cohort                Cohort   `json:"cohort"`
	Terminal              string   `json:"terminal"`
	Score                 int      `json:"score"`
	InterventionCost      int      `json:"intervention_cost"`
	Actions               []string `json:"actions"`
	ActionCount           int      `json:"action_count"`
	InitialPosterior      int      `json:"initial_posterior"`
	FinalPosterior        int      `json:"final_posterior"`
	Correct               bool     `json:"correct"`
	TeacherRetained       bool     `json:"teacher_retained"`
	EquivalenceComplete   bool     `json:"equivalence_complete"`
	HypothesisEvaluations int      `json:"hypothesis_evaluations"`
	SemanticWork          int      `json:"semantic_work"`
	EngineCycles          int      `json:"engine_cycles"`
	AttributedUnits       int      `json:"attributed_units"`
	CacheHits             int      `json:"cache_hits"`
	CacheMisses           int      `json:"cache_misses"`
	TranscriptDigest      string   `json:"transcript_digest"`
	OracleAgreements      int      `json:"oracle_agreements"`
	OracleDisagreements   int      `json:"oracle_disagreements"`
	FixtureDigest         string   `json:"-"`
	ProfileDigest         string   `json:"-"`
	FinalCodes            []string `json:"-"`
	Outcomes              []string `json:"-"`
	DynamicStates         int      `json:"-"`
	DynamicWork           int      `json:"-"`
	DynamicExpectedCost   float64  `json:"-"`
}

func acquisition(policy Policy, frozen string) string {
	switch policy {
	case Learned:
		return frozen
	case InformationGain:
		return "P=H;M=gain;S=C"
	case WorstSplit:
		return "P=W;M=gain;S=C"
	case Lexical:
		return "lexical-fixed"
	case UniformRandom:
		return "uniform-random"
	case DynamicOptimal:
		return "dynamic-optimal"
	}
	return ""
}
func profileDigest(panel string, fixture Fixture, policy Policy, code string) (string, error) {
	return causal.Digest("causal-profile/v1", struct {
		Version, Experiment, Panel   string
		Seed                         int64
		Acquisition, Fixture, Digest string
	}{causal.ProfileVersion, causal.ExperimentV1, panel, fixture.Seed, code, fixture.FixtureDigest, ""})
}
func terminalFor(initial, p []string) string {
	if len(p) == 1 {
		return "identified"
	}
	if causal.CompleteClass(initial, p) {
		return "equivalence"
	}
	return ""
}
func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
func partitionsAgree(production []causal.Cell, oracle []causaloracle.Cell) bool {
	if len(production) != len(oracle) {
		return false
	}
	for i := range production {
		if production[i].Outcome != oracle[i].Outcome || strings.Join(production[i].Hypotheses, "\x00") != strings.Join(oracle[i].Models, "\x00") {
			return false
		}
	}
	return true
}
func attributed(store *unit.Store, experiment string) int {
	count := 1
	for _, name := range store.All() {
		if u := store.Get(name); u != nil && u.GetString("experiment") == experiment {
			count++
		}
	}
	return count
}

func runEpisode(domainsDir, panel string, fixture Fixture, policy Policy, frozen string) (EpisodeReport, error) {
	report := EpisodeReport{Seed: fixture.Seed, Cohort: fixture.Cohort, Actions: []string{}, InitialPosterior: len(fixture.InitialPosterior), FixtureDigest: fixture.FixtureDigest}
	code := acquisition(policy, frozen)
	digest, e := profileDigest(panel, fixture, policy, code)
	if e != nil {
		return report, e
	}
	report.ProfileDigest = digest
	initial := append([]string(nil), fixture.InitialPosterior...)
	posterior := append([]string(nil), initial...)
	if term := terminalFor(initial, posterior); term != "" {
		report.Terminal = term
		report.FinalPosterior = len(posterior)
		report.FinalCodes = posterior
		report.Correct = contains(posterior, fixture.HiddenHypothesis)
		report.TeacherRetained = report.Correct
		report.EquivalenceComplete = term == "equivalence"
		return report, nil
	}
	if policy == PassiveOnly {
		report.Terminal = "budget-exhausted"
		report.Score = 1001
		report.FinalPosterior = len(posterior)
		report.FinalCodes = posterior
		report.TeacherRetained = true
		return report, nil
	}
	seed.DomainsDir = domainsDir
	store := unit.NewStore()
	if e := seed.LoadDomain(store, "causal"); e != nil {
		return report, e
	}
	ag := agenda.New()
	name := fmt.Sprintf("Causal.Episode.%s.%d.%s", panel, fixture.Seed, policy)
	descriptor := unit.New(name)
	descriptor.Set("isA", []string{"CausalExperiment", "Anything"})
	descriptor.Set("profileVersion", causal.ProfileVersion)
	descriptor.Set("experimentVersion", causal.ExperimentV1)
	descriptor.Set("profileDigest", digest)
	descriptor.Set("fixtureDigest", fixture.FixtureDigest)
	descriptor.Set("posterior", posterior)
	descriptor.Set("initialPosterior", initial)
	descriptor.Set("acquisitionCode", code)
	descriptor.Set("cost0", fixture.Costs[0])
	descriptor.Set("cost1", fixture.Costs[1])
	descriptor.Set("cost2", fixture.Costs[2])
	descriptor.Set("state", "ready")
	descriptor.Set("proposeTaskSlot", "causalPropose")
	descriptor.Set("updateTaskSlot", "causalUpdate")
	descriptor.Set("actionCount", 0)
	descriptor.Set("step", 0)
	descriptor.Set("totalCost", 0)
	descriptor.Set("consumedActions", []string{})
	descriptor.Set("transcriptDigest", "")
	store.Put(descriptor)
	eng := engine.New(store, ag)
	eng.Verbosity = 0
	eng.Out = io.Discard
	eng.MutConfig.Enabled = false
	teacher, e := causaloracle.NewTeacher(fixture.Token, fixture.HiddenHypothesis)
	if e != nil {
		return report, e
	}
	dynamic := causaloracle.NewDynamicPolicy(initial, fixture.Costs)
	for step := 0; step < 10; step++ {
		if policy == UniformRandom {
			descriptor.Set("forcedAction", fixture.RandomActions[step])
		} else if policy == DynamicOptimal {
			forced, e := dynamic.Choose(posterior, report.Actions)
			if e != nil {
				return report, e
			}
			descriptor.Set("forcedAction", forced)
		}
		ag.Push(&agenda.Task{Priority: 900, UnitName: name, SlotName: "causalPropose", Reasons: []string{"Propose one bounded intervention"}})
		eng.MaxCycles = 5000 - report.EngineCycles
		if eng.MaxCycles <= 0 {
			return report, fmt.Errorf("engine cycle cap")
		}
		if e := eng.Run(context.Background()); e != nil {
			return report, e
		}
		report.EngineCycles += eng.Cycle()
		if descriptor.GetString("state") != "awaiting-teacher" {
			return report, fmt.Errorf("proposal boundary state=%s", descriptor.GetString("state"))
		}
		action := descriptor.GetString("selectedAction")
		if _, e := causal.ParseAction(action); e != nil {
			return report, e
		}
		productionCells, e := causal.Partition(posterior, action)
		if e != nil {
			return report, e
		}
		oracleCells, e := causaloracle.Partition(posterior, action)
		if e != nil {
			return report, e
		}
		if !partitionsAgree(productionCells, oracleCells) {
			report.OracleDisagreements++
			return report, fmt.Errorf("partition disagreement")
		}
		report.OracleAgreements += len(posterior)
		report.HypothesisEvaluations += len(posterior) * 6
		outcome, e := teacher.Respond(fixture.Token, action)
		if e != nil {
			return report, e
		}
		expected, e := causaloracle.Filter(posterior, action, outcome)
		if e != nil {
			return report, e
		}
		responseName := fmt.Sprintf("Causal.Response.%d.%d.%s", fixture.Seed, step, action)
		response := unit.New(responseName)
		response.Set("isA", []string{"CausalTeacherResult", "Anything"})
		response.Set("experiment", name)
		response.Set("action", action)
		response.Set("outcome", outcome)
		store.Put(response)
		descriptor.Set("responseUnit", responseName)
		descriptor.Set("state", "response-present")
		ag.Push(&agenda.Task{Priority: 900, UnitName: name, SlotName: "causalUpdate", Reasons: []string{"Consume one authorized teacher result"}})
		eng.MaxCycles = 5000 - report.EngineCycles
		if eng.MaxCycles <= 0 {
			return report, fmt.Errorf("engine cycle cap")
		}
		if e := eng.Run(context.Background()); e != nil {
			return report, e
		}
		report.EngineCycles += eng.Cycle()
		posterior = descriptor.GetStrings("posterior")
		if strings.Join(posterior, "\x00") != strings.Join(expected, "\x00") {
			report.OracleDisagreements++
			return report, fmt.Errorf("posterior disagreement")
		}
		report.OracleAgreements += len(expected)
		report.Actions = append(report.Actions, action)
		report.Outcomes = append(report.Outcomes, outcome)
		if descriptor.GetString("state") != "ready" {
			report.Terminal = descriptor.GetString("terminal")
			break
		}
	}
	if report.Terminal == "" {
		report.Terminal = "budget-exhausted"
	}
	report.ActionCount = len(report.Actions)
	report.InterventionCost = descriptor.GetInt("totalCost")
	report.FinalPosterior = len(posterior)
	report.FinalCodes = append([]string(nil), posterior...)
	report.TranscriptDigest = descriptor.GetString("transcriptDigest")
	report.TeacherRetained = contains(posterior, fixture.HiddenHypothesis)
	report.EquivalenceComplete = causal.CompleteClass(initial, posterior)
	report.Correct = report.TeacherRetained && ((report.Terminal == "identified" && len(posterior) == 1) || (report.Terminal == "equivalence" && report.EquivalenceComplete))
	if report.Correct {
		report.Score = report.InterventionCost
	} else {
		report.Score = 1001
	}
	report.AttributedUnits = attributed(store, name)
	report.SemanticWork = report.HypothesisEvaluations + report.AttributedUnits + report.ActionCount*64
	if policy == DynamicOptimal {
		expected, err := dynamic.ExpectedCost()
		if err != nil {
			return report, err
		}
		report.DynamicExpectedCost, _ = expected.Float64()
		report.DynamicStates, report.DynamicWork = dynamic.States, dynamic.Work
	}
	if report.HypothesisEvaluations > 4096 || report.SemanticWork > 8192 || report.AttributedUnits > 1000 || report.EngineCycles > 5000 {
		return report, fmt.Errorf("episode cap exceeded eval=%d work=%d units=%d cycles=%d", report.HypothesisEvaluations, report.SemanticWork, report.AttributedUnits, report.EngineCycles)
	}
	return report, nil
}
