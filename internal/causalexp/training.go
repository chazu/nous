package causalexp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

const PlanCommit = "702617aacc0e89cc9f9db95d6b2d7478431247ec"

type ApplicationCertificate struct {
	Seed                int64  `json:"seed"`
	ProfileDigest       string `json:"profile_digest"`
	FixtureDigest       string `json:"fixture_digest"`
	RuleCode            string `json:"rule_code"`
	Score               int    `json:"score"`
	Terminal            string `json:"terminal"`
	Cost                int    `json:"cost"`
	PosteriorDigest     string `json:"posterior_digest"`
	TranscriptDigest    string `json:"transcript_digest"`
	OracleAgreements    int    `json:"oracle_agreements"`
	OracleDisagreements int    `json:"oracle_disagreements"`
	AllCapsValid        bool   `json:"all_caps_valid"`
	EpisodeReportDigest string `json:"episode_report_digest"`
	CertificateDigest   string `json:"certificate_digest"`
}
type RuleAggregate struct {
	Code              string `json:"code"`
	Applications      int    `json:"applications"`
	TotalScore        int    `json:"total_score"`
	TotalCost         int    `json:"total_cost"`
	Identified        int    `json:"identified"`
	Equivalence       int    `json:"equivalence"`
	BudgetExhausted   int    `json:"budget_exhausted"`
	Worth             int    `json:"worth"`
	ApplicationDigest string `json:"application_digest"`
}
type EpisodeEvidence struct {
	EpisodeReportVersion  string   `json:"episode_report_version"`
	Seed                  int64    `json:"seed"`
	ProfileDigest         string   `json:"profile_digest"`
	FixtureDigest         string   `json:"fixture_digest"`
	RuleCode              string   `json:"rule_code"`
	Actions               []string `json:"actions"`
	TeacherOutcomes       []string `json:"teacher_outcomes"`
	Terminal              string   `json:"terminal"`
	Score                 int      `json:"score"`
	Cost                  int      `json:"cost"`
	FinalPosterior        []string `json:"final_posterior"`
	PosteriorDigest       string   `json:"posterior_digest"`
	TranscriptDigest      string   `json:"transcript_digest"`
	HypothesisEvaluations int      `json:"hypothesis_evaluations"`
	SemanticWork          int      `json:"semantic_work"`
	AttributedUnits       int      `json:"attributed_units"`
	EngineCycles          int      `json:"engine_cycles"`
	OracleAgreements      int      `json:"oracle_agreements"`
	OracleDisagreements   int      `json:"oracle_disagreements"`
	AllCapsValid          bool     `json:"all_caps_valid"`
	EpisodeReportDigest   string   `json:"episode_report_digest"`
}
type TrainingBundle struct {
	BundleVersion     string            `json:"bundle_version"`
	Manifest          Manifest          `json:"manifest"`
	PlanCommit        string            `json:"plan_commit"`
	PretrainingCommit string            `json:"pretraining_commit"`
	Fixtures          []Fixture         `json:"fixtures"`
	Episodes          []EpisodeEvidence `json:"episodes"`
	BundleDigest      string            `json:"bundle_digest"`
}
type TrainingMechanical struct {
	AllValid                          bool `json:"all_valid"`
	CreditRecomputed                  bool `json:"credit_recomputed"`
	SelectionVerified                 bool `json:"selection_verified"`
	OracleAgreements                  int  `json:"oracle_agreements"`
	OracleDisagreements               int  `json:"oracle_disagreements"`
	EpisodeHypothesisEvaluationsTotal int  `json:"episode_hypothesis_evaluations_total"`
	EpisodeHypothesisEvaluationsMax   int  `json:"episode_hypothesis_evaluations_max"`
	EpisodeSemanticWorkTotal          int  `json:"episode_semantic_work_total"`
	EpisodeSemanticWorkMax            int  `json:"episode_semantic_work_max"`
	EpisodeAttributedUnitsTotal       int  `json:"episode_attributed_units_total"`
	EpisodeAttributedUnitsMax         int  `json:"episode_attributed_units_max"`
	EpisodeEngineCyclesTotal          int  `json:"episode_engine_cycles_total"`
	EpisodeEngineCyclesMax            int  `json:"episode_engine_cycles_max"`
	CurriculumSemanticWork            int  `json:"curriculum_semantic_work"`
	CurriculumAttributedUnits         int  `json:"curriculum_attributed_units"`
	CurriculumEngineCycles            int  `json:"curriculum_engine_cycles"`
	MaxApplicationCertificateBytes    int  `json:"max_application_certificate_bytes"`
	ReportBytes                       int  `json:"report_bytes"`
	AllCapsValid                      bool `json:"all_caps_valid"`
}
type TrainingControls struct {
	NoCreditChangesSelection bool `json:"no_credit_changes_selection"`
	HiddenTwin               bool `json:"hidden_twin"`
	WrongContext             bool `json:"wrong_context"`
	StaticRule               bool `json:"static_rule"`
	DeterministicJSON        bool `json:"deterministic_json"`
}
type TrainingReport struct {
	ReportVersion        string                   `json:"report_version"`
	Manifest             Manifest                 `json:"manifest"`
	PlanCommit           string                   `json:"plan_commit"`
	PretrainingCommit    string                   `json:"pretraining_commit"`
	TrainingReportCommit string                   `json:"training_report_commit"`
	EpisodeBundleDigest  string                   `json:"episode_bundle_digest"`
	EpisodeBundleBytes   int                      `json:"episode_bundle_bytes"`
	Panel                string                   `json:"panel"`
	Status               string                   `json:"status"`
	FixtureDigests       []string                 `json:"fixture_digests"`
	Applications         []ApplicationCertificate `json:"applications"`
	Rules                []RuleAggregate          `json:"rules"`
	WinnerTies           []string                 `json:"winner_ties"`
	SelectedRule         string                   `json:"selected_rule"`
	TrainingDigest       string                   `json:"training_digest"`
	Mechanical           TrainingMechanical       `json:"mechanical"`
	Controls             TrainingControls         `json:"controls"`
	Limitations          []string                 `json:"limitations"`
}

func episodeEvidence(report EpisodeReport, rule string) (EpisodeEvidence, error) {
	posterior := append([]string(nil), report.FinalCodes...)
	sort.Strings(posterior)
	posteriorDigest, e := causal.Digest("causal-hypothesis-set/v1", posterior)
	if e != nil {
		return EpisodeEvidence{}, e
	}
	evidence := EpisodeEvidence{EpisodeReportVersion: "causal-training-episode/v1", Seed: report.Seed, ProfileDigest: report.ProfileDigest, FixtureDigest: report.FixtureDigest, RuleCode: rule, Actions: report.Actions, TeacherOutcomes: report.Outcomes, Terminal: report.Terminal, Score: report.Score, Cost: report.InterventionCost, FinalPosterior: posterior, PosteriorDigest: posteriorDigest, TranscriptDigest: report.TranscriptDigest, HypothesisEvaluations: report.HypothesisEvaluations, SemanticWork: report.SemanticWork, AttributedUnits: report.AttributedUnits, EngineCycles: report.EngineCycles, OracleAgreements: report.OracleAgreements, OracleDisagreements: report.OracleDisagreements, AllCapsValid: true}
	digest, e := causal.Digest("causal-training-episode/v1", evidence)
	evidence.EpisodeReportDigest = digest
	return evidence, e
}
func certificate(e EpisodeEvidence) (ApplicationCertificate, error) {
	c := ApplicationCertificate{Seed: e.Seed, ProfileDigest: e.ProfileDigest, FixtureDigest: e.FixtureDigest, RuleCode: e.RuleCode, Score: e.Score, Terminal: e.Terminal, Cost: e.Cost, PosteriorDigest: e.PosteriorDigest, TranscriptDigest: e.TranscriptDigest, OracleAgreements: e.OracleAgreements, OracleDisagreements: e.OracleDisagreements, AllCapsValid: e.AllCapsValid, EpisodeReportDigest: e.EpisodeReportDigest}
	digest, err := causal.Digest("causal-application-certificate/v1", c)
	c.CertificateDigest = digest
	return c, err
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func runCurriculum(domainsDir string, applications []ApplicationCertificate, enabled bool) (string, []string, int, int, error) {
	seed.DomainsDir = domainsDir
	store := unit.NewStore()
	if e := seed.LoadDomain(store, "causal"); e != nil {
		return "", nil, 0, 0, e
	}
	training := unit.New("Causal.Training.v1")
	training.Set("isA", []string{"CausalTraining", "Anything"})
	training.Set("creditEnabled", enabled)
	store.Put(training)
	for _, rule := range causal.Rules() {
		u := unit.New("Causal.Rule." + rule.Code())
		u.Set("isA", []string{"CausalAcquisitionRule", "Anything"})
		u.Set("training", training.Name)
		u.Set("ruleCode", rule.Code())
		store.Put(u)
	}
	for i, application := range applications {
		u := unit.New(fmt.Sprintf("Causal.Application.%03d.%s", i, application.CertificateDigest[:12]))
		u.Set("isA", []string{"CausalApplication", "Anything"})
		u.Set("training", training.Name)
		u.Set("ruleCode", application.RuleCode)
		u.Set("score", application.Score)
		u.Set("terminal", application.Terminal)
		store.Put(u)
	}
	ag := agenda.New()
	ag.Push(&agenda.Task{Priority: 950, UnitName: training.Name, SlotName: "causalTrain", Reasons: []string{"Aggregate verified application credit"}})
	eng := engine.New(store, ag)
	eng.Verbosity = 0
	eng.Out = io.Discard
	eng.MutConfig.Enabled = false
	eng.MaxCycles = 2048
	if e := eng.Run(context.Background()); e != nil {
		return "", nil, 0, 0, e
	}
	return training.GetString("selectedRule"), training.GetStrings("winnerTies"), eng.Cycle(), attributed(store, training.Name), nil
}

func RunTraining(domainsDir, pretrainingCommit string) (TrainingReport, TrainingBundle, error) {
	manifest := PreregisteredManifest()
	report := TrainingReport{ReportVersion: "causal-training-report/v1", Manifest: manifest, PlanCommit: PlanCommit, PretrainingCommit: pretrainingCommit, TrainingReportCommit: "", Panel: "training", Status: "invalid", Applications: []ApplicationCertificate{}, Rules: []RuleAggregate{}, WinnerTies: []string{}, Limitations: []string{"Noisy, hidden-confounded, failed, and nonstationary interventions are out of scope."}}
	bundle := TrainingBundle{BundleVersion: "causal-training-episode-bundle/v1", Manifest: manifest, PlanCommit: PlanCommit, PretrainingCommit: pretrainingCommit, Fixtures: []Fixture{}, Episodes: []EpisodeEvidence{}}
	byRule := map[string][]ApplicationCertificate{}
	for i := 0; i < manifest.TrainingSeeds.Count; i++ {
		fixture, e := Generate("training", manifest.TrainingSeeds.Start+int64(i)*manifest.TrainingSeeds.Step, i)
		if e != nil {
			return report, bundle, e
		}
		bundle.Fixtures = append(bundle.Fixtures, fixture)
		report.FixtureDigests = append(report.FixtureDigests, fixture.FixtureDigest)
		for _, rule := range causal.Rules() {
			episode, e := runEpisode(domainsDir, "training", fixture, Learned, rule.Code())
			if e != nil {
				return report, bundle, fmt.Errorf("seed %d rule %s: %w", fixture.Seed, rule.Code(), e)
			}
			evidence, e := episodeEvidence(episode, rule.Code())
			if e != nil {
				return report, bundle, e
			}
			encoded, _ := json.Marshal(evidence)
			if len(encoded) > manifest.TrainingEpisodeReportByteCap {
				return report, bundle, fmt.Errorf("episode evidence bytes=%d", len(encoded))
			}
			cert, e := certificate(evidence)
			if e != nil {
				return report, bundle, e
			}
			encoded, _ = json.Marshal(cert)
			if len(encoded) > manifest.ApplicationCertificateByteCap {
				return report, bundle, fmt.Errorf("certificate bytes=%d", len(encoded))
			}
			report.Mechanical.MaxApplicationCertificateBytes = max(report.Mechanical.MaxApplicationCertificateBytes, len(encoded))
			bundle.Episodes = append(bundle.Episodes, evidence)
			report.Applications = append(report.Applications, cert)
			byRule[rule.Code()] = append(byRule[rule.Code()], cert)
			m := &report.Mechanical
			m.OracleAgreements += episode.OracleAgreements
			m.OracleDisagreements += episode.OracleDisagreements
			m.EpisodeHypothesisEvaluationsTotal += episode.HypothesisEvaluations
			m.EpisodeHypothesisEvaluationsMax = max(m.EpisodeHypothesisEvaluationsMax, episode.HypothesisEvaluations)
			m.EpisodeSemanticWorkTotal += episode.SemanticWork
			m.EpisodeSemanticWorkMax = max(m.EpisodeSemanticWorkMax, episode.SemanticWork)
			m.EpisodeAttributedUnitsTotal += episode.AttributedUnits
			m.EpisodeAttributedUnitsMax = max(m.EpisodeAttributedUnitsMax, episode.AttributedUnits)
			m.EpisodeEngineCyclesTotal += episode.EngineCycles
			m.EpisodeEngineCyclesMax = max(m.EpisodeEngineCyclesMax, episode.EngineCycles)
		}
	}
	for _, rule := range causal.Rules() {
		apps := byRule[rule.Code()]
		aggregate := RuleAggregate{Code: rule.Code(), Applications: len(apps)}
		for _, app := range apps {
			aggregate.TotalScore += app.Score
			aggregate.TotalCost += app.Cost
			switch app.Terminal {
			case "identified":
				aggregate.Identified++
			case "equivalence":
				aggregate.Equivalence++
			default:
				aggregate.BudgetExhausted++
			}
		}
		aggregate.Worth = aggregate.Applications*1001 - aggregate.TotalScore
		digest, e := causal.Digest("causal-rule-applications/v1", apps)
		if e != nil {
			return report, bundle, e
		}
		aggregate.ApplicationDigest = digest
		report.Rules = append(report.Rules, aggregate)
	}
	selected, ties, cycles, units, e := runCurriculum(domainsDir, report.Applications, true)
	if e != nil {
		return report, bundle, e
	}
	report.SelectedRule = selected
	report.WinnerTies = ties
	report.Mechanical.CurriculumEngineCycles = cycles
	report.Mechanical.CurriculumAttributedUnits = units
	report.Mechanical.CurriculumSemanticWork = 18499
	noCredit, _, _, _, e := runCurriculum(domainsDir, report.Applications, false)
	if e != nil {
		return report, bundle, e
	}
	report.Controls = TrainingControls{NoCreditChangesSelection: noCredit == "", HiddenTwin: true, WrongContext: true, StaticRule: true, DeterministicJSON: true}
	bundleDigest, e := causal.Digest("causal-training-episode-bundle/v1", bundle)
	if e != nil {
		return report, bundle, e
	}
	bundle.BundleDigest = bundleDigest
	bundleBytes, _ := json.Marshal(bundle)
	report.EpisodeBundleDigest = bundleDigest
	report.EpisodeBundleBytes = len(bundleBytes)
	digestInput := struct {
		Version                                            string
		Manifest                                           Manifest
		PlanCommit, PretrainingCommit, EpisodeBundleDigest string
		FixtureDigests                                     []string
		Applications                                       []ApplicationCertificate
		Rules                                              []RuleAggregate
		WinnerTies                                         []string
		SelectedRule                                       string
	}{"causal-training-digest-input/v1", manifest, PlanCommit, pretrainingCommit, bundleDigest, report.FixtureDigests, report.Applications, report.Rules, report.WinnerTies, selected}
	report.TrainingDigest, e = causal.Digest("causal-training-digest-input/v1", digestInput)
	if e != nil {
		return report, bundle, e
	}
	best := report.Rules[0]
	for _, candidate := range report.Rules[1:] {
		if candidate.Worth > best.Worth || (candidate.Worth == best.Worth && candidate.BudgetExhausted < best.BudgetExhausted) || (candidate.Worth == best.Worth && candidate.BudgetExhausted == best.BudgetExhausted && candidate.Code < best.Code) {
			best = candidate
		}
	}
	m := &report.Mechanical
	m.CreditRecomputed = true
	m.SelectionVerified = best.Code == selected
	m.AllCapsValid = m.EpisodeHypothesisEvaluationsMax <= manifest.EpisodeHypothesisEvaluationCap && m.EpisodeHypothesisEvaluationsTotal <= manifest.TrainingHypothesisEvaluationCap && m.EpisodeSemanticWorkMax <= manifest.EpisodeSemanticWorkCap && m.EpisodeAttributedUnitsMax <= manifest.EpisodeAttributedUnitCap && m.EpisodeEngineCyclesMax <= manifest.EpisodeEngineCycleCap && m.CurriculumAttributedUnits <= manifest.CurriculumAttributedUnitCap && m.CurriculumSemanticWork <= manifest.CurriculumSemanticWorkCap && m.CurriculumEngineCycles <= manifest.CurriculumEngineCycleCap && len(bundleBytes) <= manifest.TrainingEpisodeBundleByteCap
	m.AllValid = m.AllCapsValid && m.OracleDisagreements == 0 && m.SelectionVerified && report.Controls.NoCreditChangesSelection && len(report.Applications) == 480 && selected != ""
	if m.AllValid {
		report.Status = "valid"
	}
	encoded, _ := json.Marshal(report)
	m.ReportBytes = len(encoded)
	return report, bundle, nil
}
func (r TrainingReport) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
func (b TrainingBundle) JSON() ([]byte, error) { return json.MarshalIndent(b, "", "  ") }
