package causalexpv2

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/chazu/nous/internal/causalv2"
)

type ApplicationCertificate = causalv2.ApplicationCertificate
type RuleAggregate = causalv2.RuleAggregatePayload

type EpisodeEvidence struct {
	EpisodeReportVersion  string      `json:"episode_report_version"`
	Seed                  int64       `json:"seed"`
	ProfileDigest         string      `json:"profile_digest"`
	FixtureDigest         string      `json:"fixture_digest"`
	RuleCode              string      `json:"rule_code"`
	Actions               []string    `json:"actions"`
	TeacherOutcomes       []string    `json:"teacher_outcomes"`
	Terminal              string      `json:"terminal"`
	Score                 int         `json:"score"`
	Cost                  int         `json:"cost"`
	FinalPosterior        []string    `json:"final_posterior"`
	PosteriorDigest       string      `json:"posterior_digest"`
	TranscriptDigest      string      `json:"transcript_digest"`
	HypothesisEvaluations int         `json:"hypothesis_evaluations"`
	SemanticWork          int         `json:"semantic_work"`
	AttributedUnits       int         `json:"attributed_units"`
	EngineCycles          int         `json:"engine_cycles"`
	OracleAgreements      int         `json:"oracle_agreements"`
	OracleDisagreements   int         `json:"oracle_disagreements"`
	MeterItems            []MeterItem `json:"meter_items"`
	AllCapsValid          bool        `json:"all_caps_valid"`
	EpisodeReportDigest   string      `json:"episode_report_digest"`
}

type TrainingBundle struct {
	BundleVersion     string            `json:"bundle_version"`
	Manifest          causalv2.Manifest `json:"manifest"`
	PlanCommit        string            `json:"plan_commit"`
	PretrainingCommit string            `json:"pretraining_commit"`
	Fixtures          []PrivateFixture  `json:"fixtures"`
	Episodes          []EpisodeEvidence `json:"episodes"`
	BundleDigest      string            `json:"bundle_digest"`
}

type ControlResult = causalv2.ControlResult
type ControlCertificate = causalv2.ControlCertificate
type ControlBundle = causalv2.ControlBundle
type ControlEvidence = causalv2.ControlEvidence

type TrainingMechanical struct {
	AllValid                       bool             `json:"all_valid"`
	CreditRecomputed               bool             `json:"credit_recomputed"`
	SelectionVerified              bool             `json:"selection_verified"`
	OracleAgreements               int              `json:"oracle_agreements"`
	OracleDisagreements            int              `json:"oracle_disagreements"`
	Meters                         []AggregateMeter `json:"meters"`
	MaxDescriptorBytes             int              `json:"max_descriptor_bytes"`
	MaxTrainingEpisodeReportBytes  int              `json:"max_training_episode_report_bytes"`
	MaxApplicationCertificateBytes int              `json:"max_application_certificate_bytes"`
	NonrecordBytes                 string           `json:"nonrecord_bytes"`
	ReportBytes                    string           `json:"report_bytes"`
	AllCapsValid                   bool             `json:"all_caps_valid"`
}

type TrainingControls struct {
	NoCreditChangesSelection bool `json:"no_credit_changes_selection"`
	HiddenTwin               bool `json:"hidden_twin"`
	WrongContext             bool `json:"wrong_context"`
	StaticRule               bool `json:"static_rule"`
	DeterministicJSON        bool `json:"deterministic_json"`
}

type TrainingReport struct {
	ReportVersion         string                   `json:"report_version"`
	Manifest              causalv2.Manifest        `json:"manifest"`
	PlanCommit            string                   `json:"plan_commit"`
	PretrainingCommit     string                   `json:"pretraining_commit"`
	TrainingReportCommit  string                   `json:"training_report_commit"`
	EpisodeBundleDigest   string                   `json:"episode_bundle_digest"`
	EpisodeBundleBytes    int                      `json:"episode_bundle_bytes"`
	ControlBundle         ControlBundle            `json:"control_bundle"`
	ControlBundleDigest   string                   `json:"control_bundle_digest"`
	ControlEvidence       ControlEvidence          `json:"control_evidence"`
	ControlEvidenceDigest string                   `json:"control_evidence_digest"`
	TaskMeterItems        []TaskMeterItem          `json:"task_meter_items"`
	TaskMeterItemsDigest  string                   `json:"task_meter_items_digest"`
	Panel                 string                   `json:"panel"`
	Status                string                   `json:"status"`
	FixtureDigests        []string                 `json:"fixture_digests"`
	Applications          []ApplicationCertificate `json:"applications"`
	Rules                 []RuleAggregate          `json:"rules"`
	WinnerTies            []string                 `json:"winner_ties"`
	SelectedRule          string                   `json:"selected_rule"`
	TrainingDigest        string                   `json:"training_digest"`
	Mechanical            TrainingMechanical       `json:"mechanical"`
	Controls              TrainingControls         `json:"controls"`
	Limitations           []string                 `json:"limitations"`
}

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

type Policy string

type EvaluationFixture struct {
	Seed                  int64       `json:"seed"`
	Cohort                Cohort      `json:"cohort"`
	Terminal              string      `json:"terminal"`
	Score                 int         `json:"score"`
	InterventionCost      int         `json:"intervention_cost"`
	Actions               []string    `json:"actions"`
	ActionCount           int         `json:"action_count"`
	InitialPosterior      int         `json:"initial_posterior"`
	FinalPosterior        int         `json:"final_posterior"`
	Correct               bool        `json:"correct"`
	TeacherRetained       bool        `json:"teacher_retained"`
	EquivalenceComplete   bool        `json:"equivalence_complete"`
	HypothesisEvaluations int         `json:"hypothesis_evaluations"`
	SemanticWork          int         `json:"semantic_work"`
	EngineCycles          int         `json:"engine_cycles"`
	AttributedUnits       int         `json:"attributed_units"`
	CacheHits             int         `json:"cache_hits"`
	CacheMisses           int         `json:"cache_misses"`
	TranscriptDigest      string      `json:"transcript_digest"`
	OracleAgreements      int         `json:"oracle_agreements"`
	OracleDisagreements   int         `json:"oracle_disagreements"`
	MeterItems            []MeterItem `json:"meter_items"`
	AllCapsValid          bool        `json:"all_caps_valid"`
}

type PolicyReport struct {
	Name     Policy              `json:"name"`
	Fixtures []EvaluationFixture `json:"fixtures"`
	Overall  Aggregate           `json:"overall"`
	Cohorts  []Aggregate         `json:"cohorts"`
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

type EvaluationMechanical struct {
	AllValid              bool             `json:"all_valid"`
	DependencyBoundary    bool             `json:"dependency_boundary"`
	ProfileValid          bool             `json:"profile_valid"`
	TranscriptValid       bool             `json:"transcript_valid"`
	TrainingFreezeValid   bool             `json:"training_freeze_valid"`
	OracleAgreements      int              `json:"oracle_agreements"`
	OracleDisagreements   int              `json:"oracle_disagreements"`
	Meters                []AggregateMeter `json:"meters"`
	MaxDescriptorBytes    int              `json:"max_descriptor_bytes"`
	MaxFixtureRecordBytes int              `json:"max_fixture_record_bytes"`
	NonrecordBytes        string           `json:"nonrecord_bytes"`
	ReportBytes           string           `json:"report_bytes"`
	AllCapsValid          bool             `json:"all_caps_valid"`
}

type DynamicBenchmark struct {
	RealizedMeanCost        float64 `json:"realized_mean_cost"`
	UniformExpectedMeanCost float64 `json:"uniform_expected_mean_cost"`
	TotalDPStates           int     `json:"total_dp_states"`
	MaxDPStates             int     `json:"max_dp_states"`
	TotalDPWork             int     `json:"total_dp_work"`
	MaxDPWork               int     `json:"max_dp_work"`
}

type EvaluationReport struct {
	ReportVersion         string               `json:"report_version"`
	Manifest              causalv2.Manifest    `json:"manifest"`
	PlanCommit            string               `json:"plan_commit"`
	PretrainingCommit     string               `json:"pretraining_commit"`
	TrainingReportCommit  string               `json:"training_report_commit"`
	TrainingDigest        string               `json:"training_digest"`
	FrozenRule            string               `json:"frozen_rule"`
	ImplementationCommit  string               `json:"implementation_commit"`
	Panel                 string               `json:"panel"`
	Status                string               `json:"status"`
	ControlBundle         ControlBundle        `json:"control_bundle"`
	ControlBundleDigest   string               `json:"control_bundle_digest"`
	ControlEvidence       ControlEvidence      `json:"control_evidence"`
	ControlEvidenceDigest string               `json:"control_evidence_digest"`
	TaskMeterItems        []TaskMeterItem      `json:"task_meter_items"`
	TaskMeterItemsDigest  string               `json:"task_meter_items_digest"`
	Mechanical            EvaluationMechanical `json:"mechanical"`
	Policies              []PolicyReport       `json:"policies"`
	Contrasts             []Contrast           `json:"contrasts"`
	Gates                 Gates                `json:"gates"`
	Controls              Controls             `json:"controls"`
	DynamicBenchmark      DynamicBenchmark     `json:"dynamic_benchmark"`
	ReportDigest          string               `json:"report_digest"`
	Limitations           []string             `json:"limitations"`
}

func fixedBytes(length int) (string, error) {
	if length < 0 || length > 99999999 {
		return "", errors.New("byte count exceeds fixed-width field")
	}
	return fmt.Sprintf("%08d", length), nil
}

func parseFixedBytes(value string) (int, error) {
	if len(value) != 8 {
		return 0, errors.New("byte count is not eight decimal digits")
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, errors.New("byte count is not eight decimal digits")
		}
	}
	return strconv.Atoi(value)
}

func signSelfDigest(domain string, value any) (string, error) {
	return causalv2.Digest(domain, value)
}

func SignEpisode(evidence *EpisodeEvidence) error {
	if evidence == nil || evidence.EpisodeReportVersion != "causal-training-episode/v2" {
		return errors.New("invalid episode evidence")
	}
	if err := VerifyEpisodeMeterItems(evidence.MeterItems); err != nil {
		return err
	}
	evidence.EpisodeReportDigest = ""
	digest, err := signSelfDigest("causal-training-episode/v2", *evidence)
	evidence.EpisodeReportDigest = digest
	return err
}

func VerifyEpisode(evidence EpisodeEvidence) error {
	got := evidence.EpisodeReportDigest
	if got == "" {
		return errors.New("empty episode digest")
	}
	if err := SignEpisode(&evidence); err != nil {
		return err
	}
	if got != evidence.EpisodeReportDigest {
		return errors.New("episode digest mismatch")
	}
	base := evidence
	base.MeterItems = []MeterItem{}
	baseBytes, _ := causalv2.CanonicalJSON(base)
	meterBytes, _ := causalv2.CanonicalJSON(evidence.MeterItems)
	manifest := causalv2.PreregisteredManifest()
	if len(baseBytes) > manifest.TrainingEpisodeBaseByteCap || len(meterBytes)-2 > manifest.EpisodeMeterItemsByteCap {
		return errors.New("episode encoding subcap exceeded")
	}
	return nil
}
