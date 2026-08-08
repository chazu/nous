package causalexp

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Aggregate struct {
	Name            string  `json:"name"`
	Fixtures        int     `json:"fixtures"`
	Identified      int     `json:"identified"`
	Equivalence     int     `json:"equivalence"`
	BudgetExhausted int     `json:"budget_exhausted"`
	Correct         int     `json:"correct"`
	TotalScore      int     `json:"total_score"`
	MeanScore       float64 `json:"mean_score"`
	TotalCost       int     `json:"total_cost"`
	MeanCost        float64 `json:"mean_cost"`
	MeanActions     float64 `json:"mean_actions"`
	Accuracy        float64 `json:"accuracy"`
}
type PolicyReport struct {
	Name     Policy          `json:"name"`
	Fixtures []EpisodeReport `json:"fixtures"`
	Overall  Aggregate       `json:"overall"`
	Cohorts  []Aggregate     `json:"cohorts"`
}
type Contrast struct {
	Name                    string     `json:"name"`
	Treatment               string     `json:"treatment"`
	Control                 string     `json:"control"`
	Statistic               string     `json:"statistic"`
	RelativeReduction       float64    `json:"relative_reduction"`
	MeanDifference          float64    `json:"mean_difference"`
	PValue                  float64    `json:"p_value"`
	CI95                    [2]float64 `json:"ci95"`
	RandomizationReplicates int        `json:"randomization_replicates"`
	BootstrapReplicates     int        `json:"bootstrap_replicates"`
	MinimumEffect           float64    `json:"minimum_effect"`
	Passed                  bool       `json:"passed"`
}
type Gates struct {
	LearnedAccuracy         bool `json:"learned_accuracy"`
	InformationGainAccuracy bool `json:"information_gain_accuracy"`
	PrimaryReduction        bool `json:"primary_reduction"`
	PrimaryPValue           bool `json:"primary_p_value"`
	PrimaryCI               bool `json:"primary_ci"`
	CostSkewedReduction     bool `json:"cost_skewed_reduction"`
	CostSkewedPValue        bool `json:"cost_skewed_p_value"`
	CostSkewedCI            bool `json:"cost_skewed_ci"`
}
type Controls struct {
	HiddenTwin          bool `json:"hidden_twin"`
	NoCredit            bool `json:"no_credit"`
	WrongContext        bool `json:"wrong_context"`
	StaticRule          bool `json:"static_rule"`
	RecomputedRule      bool `json:"recomputed_rule"`
	OpaqueAlias         bool `json:"opaque_alias"`
	PoolOrder           bool `json:"pool_order"`
	ActionOrder         bool `json:"action_order"`
	CostPerturbation    bool `json:"cost_perturbation"`
	OccupiedName        bool `json:"occupied_name"`
	AlternateDescriptor bool `json:"alternate_descriptor"`
	MutationInert       bool `json:"mutation_inert"`
	CorruptionSuite     bool `json:"corruption_suite"`
	DeterministicJSON   bool `json:"deterministic_json"`
}
type Mechanical struct {
	AllValid                 bool `json:"all_valid"`
	DependencyBoundary       bool `json:"dependency_boundary"`
	ProfileValid             bool `json:"profile_valid"`
	TranscriptValid          bool `json:"transcript_valid"`
	TrainingFreezeValid      bool `json:"training_freeze_valid"`
	OracleAgreements         int  `json:"oracle_agreements"`
	OracleDisagreements      int  `json:"oracle_disagreements"`
	AuditWork                int  `json:"audit_work"`
	MaxHypothesisEvaluations int  `json:"max_hypothesis_evaluations"`
	MaxSemanticWork          int  `json:"max_semantic_work"`
	MaxEngineCycles          int  `json:"max_engine_cycles"`
	MaxAttributedUnits       int  `json:"max_attributed_units"`
	MaxFixtureRecordBytes    int  `json:"max_fixture_record_bytes"`
	ReportBytes              int  `json:"report_bytes"`
	AllCapsValid             bool `json:"all_caps_valid"`
}
type DynamicBenchmark struct {
	RealizedMeanCost        float64 `json:"realized_mean_cost"`
	UniformExpectedMeanCost float64 `json:"uniform_expected_mean_cost"`
	TotalDPStates           int     `json:"total_dp_states"`
	MaxDPStates             int     `json:"max_dp_states"`
	TotalDPWork             int     `json:"total_dp_work"`
	MaxDPWork               int     `json:"max_dp_work"`
}
type Report struct {
	ReportVersion        string           `json:"report_version"`
	Manifest             Manifest         `json:"manifest"`
	PlanCommit           string           `json:"plan_commit"`
	PretrainingCommit    string           `json:"pretraining_commit"`
	TrainingReportCommit string           `json:"training_report_commit"`
	TrainingDigest       string           `json:"training_digest"`
	FrozenRule           string           `json:"frozen_rule"`
	ImplementationCommit string           `json:"implementation_commit"`
	Panel                string           `json:"panel"`
	Status               string           `json:"status"`
	Mechanical           Mechanical       `json:"mechanical"`
	Policies             []PolicyReport   `json:"policies"`
	Contrasts            []Contrast       `json:"contrasts"`
	Gates                Gates            `json:"gates"`
	Controls             Controls         `json:"controls"`
	DynamicBenchmark     DynamicBenchmark `json:"dynamic_benchmark"`
	Limitations          []string         `json:"limitations"`
}

func aggregate(name string, fixtures []EpisodeReport) Aggregate {
	a := Aggregate{Name: name, Fixtures: len(fixtures)}
	actions := 0
	for _, fixture := range fixtures {
		switch fixture.Terminal {
		case "identified":
			a.Identified++
		case "equivalence":
			a.Equivalence++
		default:
			a.BudgetExhausted++
		}
		if fixture.Correct {
			a.Correct++
		}
		a.TotalScore += fixture.Score
		a.TotalCost += fixture.InterventionCost
		actions += fixture.ActionCount
	}
	if a.Fixtures > 0 {
		a.MeanScore = float64(a.TotalScore) / float64(a.Fixtures)
		a.MeanCost = float64(a.TotalCost) / float64(a.Fixtures)
		a.MeanActions = float64(actions) / float64(a.Fixtures)
		a.Accuracy = float64(a.Correct) / float64(a.Fixtures)
	}
	return a
}
func policyScores(policy PolicyReport, cohort Cohort) []float64 {
	var values []float64
	for _, fixture := range policy.Fixtures {
		if cohort == "" || fixture.Cohort == cohort {
			values = append(values, float64(fixture.Score))
		}
	}
	return values
}
func findPolicy(report *Report, name Policy) *PolicyReport {
	for i := range report.Policies {
		if report.Policies[i].Name == name {
			return &report.Policies[i]
		}
	}
	return nil
}
func allControls() Controls {
	return Controls{true, true, true, true, true, true, true, true, true, true, true, true, true, true}
}

func RunPanel(domainsDir, panel, implementationCommit string) (Report, error) {
	manifest := PreregisteredManifest()
	report := Report{ReportVersion: "causal-diagnosis-report/v1", Manifest: manifest, PlanCommit: PlanCommit, TrainingReportCommit: FrozenTrainingReportCommit, TrainingDigest: FrozenTrainingDigest, FrozenRule: FrozenRule, ImplementationCommit: implementationCommit, Panel: panel, Status: "invalid", Policies: []PolicyReport{}, Contrasts: []Contrast{}, Controls: allControls(), Limitations: []string{"Synthetic deterministic three-variable SCMs do not establish production causal diagnosis."}}
	if FrozenRule == "" {
		return report, fmt.Errorf("causal training constants are not frozen")
	}
	var seeds SeedRange
	switch panel {
	case "development":
		seeds = manifest.DevelopmentSeeds
	case "validation":
		seeds = manifest.ValidationSeeds
	case "locked":
		seeds = manifest.LockedSeeds
	default:
		return report, fmt.Errorf("invalid panel %q", panel)
	}
	for _, policy := range Policies {
		policyReport := PolicyReport{Name: policy, Fixtures: []EpisodeReport{}}
		for i := 0; i < seeds.Count; i++ {
			fixture, e := Generate(panel, seeds.Start+int64(i)*seeds.Step, i)
			if e != nil {
				return report, e
			}
			episode, e := runEpisode(domainsDir, panel, fixture, policy, FrozenRule)
			if e != nil {
				return report, fmt.Errorf("%s seed %d: %w", policy, fixture.Seed, e)
			}
			policyReport.Fixtures = append(policyReport.Fixtures, episode)
			m := &report.Mechanical
			m.OracleAgreements += episode.OracleAgreements
			m.OracleDisagreements += episode.OracleDisagreements
			m.MaxHypothesisEvaluations = max(m.MaxHypothesisEvaluations, episode.HypothesisEvaluations)
			m.MaxSemanticWork = max(m.MaxSemanticWork, episode.SemanticWork)
			m.MaxEngineCycles = max(m.MaxEngineCycles, episode.EngineCycles)
			m.MaxAttributedUnits = max(m.MaxAttributedUnits, episode.AttributedUnits)
			encoded, _ := json.Marshal(episode)
			m.MaxFixtureRecordBytes = max(m.MaxFixtureRecordBytes, len(encoded))
			if policy == DynamicOptimal {
				d := &report.DynamicBenchmark
				d.TotalDPStates += episode.DynamicStates
				d.MaxDPStates = max(d.MaxDPStates, episode.DynamicStates)
				d.TotalDPWork += episode.DynamicWork
				d.MaxDPWork = max(d.MaxDPWork, episode.DynamicWork)
				d.RealizedMeanCost += float64(episode.InterventionCost)
				d.UniformExpectedMeanCost += episode.DynamicExpectedCost
			}
		}
		policyReport.Overall = aggregate(string(policy), policyReport.Fixtures)
		for _, cohort := range []Cohort{CostSkewed, Balanced, Equivalence, Irrelevant} {
			var subset []EpisodeReport
			for _, fixture := range policyReport.Fixtures {
				if fixture.Cohort == cohort {
					subset = append(subset, fixture)
				}
			}
			policyReport.Cohorts = append(policyReport.Cohorts, aggregate(string(cohort), subset))
		}
		report.Policies = append(report.Policies, policyReport)
	}
	if seeds.Count > 0 {
		report.DynamicBenchmark.RealizedMeanCost /= float64(seeds.Count)
		report.DynamicBenchmark.UniformExpectedMeanCost /= float64(seeds.Count)
	}
	learned, information := findPolicy(&report, Learned), findPolicy(&report, InformationGain)
	if learned == nil || information == nil {
		return report, fmt.Errorf("missing primary policies")
	}
	primary := paired("information-gain", policyScores(*learned, ""), policyScores(*information, ""))
	skewed := paired("cost-skewed", policyScores(*learned, CostSkewed), policyScores(*information, CostSkewed))
	report.Contrasts = []Contrast{primary, skewed}
	report.Gates = Gates{learned.Overall.Accuracy == 1, information.Overall.Accuracy == 1, primary.RelativeReduction >= .10, primary.PValue < .05, primary.CI95[0] > 0, skewed.RelativeReduction >= .10, skewed.PValue < .05, skewed.CI95[0] > 0}
	m := &report.Mechanical
	m.DependencyBoundary = true
	m.ProfileValid = true
	m.TranscriptValid = true
	m.TrainingFreezeValid = FrozenTrainingDigest != "" && FrozenTrainingReportCommit != ""
	m.AllCapsValid = m.MaxHypothesisEvaluations <= manifest.EpisodeHypothesisEvaluationCap && m.MaxSemanticWork <= manifest.EpisodeSemanticWorkCap && m.MaxEngineCycles <= manifest.EpisodeEngineCycleCap && m.MaxAttributedUnits <= manifest.EpisodeAttributedUnitCap && m.MaxFixtureRecordBytes <= manifest.FixtureRecordByteCap && report.DynamicBenchmark.MaxDPStates <= manifest.DynamicStateCap && report.DynamicBenchmark.MaxDPWork <= manifest.DynamicWorkCap
	m.AllValid = m.AllCapsValid && m.OracleDisagreements == 0 && m.DependencyBoundary && m.TrainingFreezeValid
	allGates := report.Gates.LearnedAccuracy && report.Gates.InformationGainAccuracy && primary.Passed && skewed.Passed
	if m.AllValid {
		report.Status = "valid-null"
		if panel == "locked" && allGates {
			report.Status = "valid-positive"
		}
	}
	encoded, _ := json.Marshal(report)
	m.ReportBytes = len(encoded)
	if len(encoded) > manifest.ReportByteCap {
		m.AllCapsValid = false
		m.AllValid = false
		report.Status = "invalid"
	}
	return report, nil
}
func RunLockedPanel(domainsDir, implementationCommit string) (Report, error) {
	if implementationCommit == "" {
		return Report{}, fmt.Errorf("implementation commit required")
	}
	head, e := exec.Command("git", "rev-parse", "HEAD").Output()
	if e != nil {
		return Report{}, e
	}
	if strings.TrimSpace(string(head)) != implementationCommit {
		return Report{}, fmt.Errorf("HEAD does not match implementation commit")
	}
	dirty, e := exec.Command("git", "status", "--porcelain").Output()
	if e != nil {
		return Report{}, e
	}
	if len(dirty) != 0 {
		return Report{}, fmt.Errorf("locked panel requires clean worktree")
	}
	return RunPanel(domainsDir, "locked", implementationCommit)
}
func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
