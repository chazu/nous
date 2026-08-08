package causalv2

import "reflect"

type SeedRange struct {
	Start int64 `json:"start"`
	Count int   `json:"count"`
	Step  int64 `json:"step"`
}

type Cohorts struct {
	CostSkewed  int `json:"cost-skewed"`
	Balanced    int `json:"balanced"`
	Equivalence int `json:"equivalence"`
	Irrelevant  int `json:"irrelevant"`
}

// Manifest field order is the normative revision-4 order.
type Manifest struct {
	ExperimentVersion                        string    `json:"experiment_version"`
	GeneratorVersion                         string    `json:"generator_version"`
	HypothesisVersion                        string    `json:"hypothesis_version"`
	AcquisitionVersion                       string    `json:"acquisition_version"`
	OracleVersion                            string    `json:"oracle_version"`
	TeacherVersion                           string    `json:"teacher_version"`
	TrainingVersion                          string    `json:"training_version"`
	BaselineVersion                          string    `json:"baseline_version"`
	CostVersion                              string    `json:"cost_version"`
	ReportVersion                            string    `json:"report_version"`
	StatisticsVersion                        string    `json:"statistics_version"`
	ProfileVersion                           string    `json:"profile_version"`
	DevelopmentSeeds                         SeedRange `json:"development_seeds"`
	TrainingSeeds                            SeedRange `json:"training_seeds"`
	ValidationSeeds                          SeedRange `json:"validation_seeds"`
	LockedSeeds                              SeedRange `json:"locked_seeds"`
	CohortAssignment                         string    `json:"cohort_assignment"`
	LockedCohorts                            Cohorts   `json:"locked_cohorts"`
	Variables                                int       `json:"variables"`
	MaximumIndegree                          int       `json:"maximum_indegree"`
	MaximumPool                              int       `json:"maximum_pool"`
	MinimumInitialPosterior                  int       `json:"minimum_initial_posterior"`
	LegalInterventions                       int       `json:"legal_interventions"`
	MaximumConsumedInterventions             int       `json:"maximum_consumed_interventions"`
	CandidateAcquisitionRules                int       `json:"candidate_acquisition_rules"`
	InterventionCostMinimum                  int       `json:"intervention_cost_minimum"`
	InterventionCostMaximum                  int       `json:"intervention_cost_maximum"`
	EpisodeCostCeiling                       int       `json:"episode_cost_ceiling"`
	InvalidOrExhaustedScore                  int       `json:"invalid_or_exhausted_score"`
	EpisodeEngineCycleCap                    int       `json:"episode_engine_cycle_cap"`
	EpisodeAttributedUnitCap                 int       `json:"episode_attributed_unit_cap"`
	DescriptorByteCap                        int       `json:"descriptor_byte_cap"`
	CurriculumAttributedUnitCap              int       `json:"curriculum_attributed_unit_cap"`
	CurriculumSemanticWorkCap                int       `json:"curriculum_semantic_work_cap"`
	CurriculumEngineCycleCap                 int       `json:"curriculum_engine_cycle_cap"`
	EpisodeHypothesisEvaluationCap           int       `json:"episode_hypothesis_evaluation_cap"`
	TrainingHypothesisEvaluationCap          int       `json:"training_hypothesis_evaluation_cap"`
	CertificateReplaySemanticWorkCap         int       `json:"certificate_replay_semantic_work_cap"`
	PostSelectionReplaySemanticWorkCap       int       `json:"post_selection_replay_semantic_work_cap"`
	OracleAuditWorkCap                       int       `json:"oracle_audit_work_cap"`
	ControlWorkCap                           int       `json:"control_work_cap"`
	ControlAttributedUnitCap                 int       `json:"control_attributed_unit_cap"`
	DynamicStateCap                          int       `json:"dynamic_state_cap"`
	DynamicWorkCap                           int       `json:"dynamic_work_cap"`
	EpisodeSemanticWorkCap                   int       `json:"episode_semantic_work_cap"`
	TeacherEvaluationCap                     int       `json:"teacher_evaluation_cap"`
	ReportByteCap                            int       `json:"report_byte_cap"`
	FixtureRecordByteCap                     int       `json:"fixture_record_byte_cap"`
	EvaluationFixtureBaseByteCap             int       `json:"evaluation_fixture_base_byte_cap"`
	EvaluationFixtureMeterItemsByteCap       int       `json:"evaluation_fixture_meter_items_byte_cap"`
	TrainingFixtureByteCap                   int       `json:"training_fixture_byte_cap"`
	ApplicationCertificateByteCap            int       `json:"application_certificate_byte_cap"`
	TaskMeterItemByteCap                     int       `json:"task_meter_item_byte_cap"`
	ControlCertificateByteCap                int       `json:"control_certificate_byte_cap"`
	ControlBundleByteCap                     int       `json:"control_bundle_byte_cap"`
	ControlEvidenceByteCap                   int       `json:"control_evidence_byte_cap"`
	ControlEvidenceNoCreditArtifactsByteCap  int       `json:"control_evidence_no_credit_artifacts_byte_cap"`
	ControlEvidenceCorruptionBaselineByteCap int       `json:"control_evidence_corruption_baseline_byte_cap"`
	ControlEvidenceCorruptionCasesByteCap    int       `json:"control_evidence_corruption_cases_byte_cap"`
	ControlEvidenceDependencyFilesByteCap    int       `json:"control_evidence_dependency_files_byte_cap"`
	ControlEvidenceOtherRecordsByteCap       int       `json:"control_evidence_other_records_byte_cap"`
	TrainingEpisodeBaseByteCap               int       `json:"training_episode_base_byte_cap"`
	EpisodeMeterItemsByteCap                 int       `json:"episode_meter_items_byte_cap"`
	TrainingEpisodeReportByteCap             int       `json:"training_episode_report_byte_cap"`
	TrainingEpisodeBundleByteCap             int       `json:"training_episode_bundle_byte_cap"`
	NonrecordReportByteCap                   int       `json:"nonrecord_report_byte_cap"`
	MaximumLimitations                       int       `json:"maximum_limitations"`
	LimitationByteCap                        int       `json:"limitation_byte_cap"`
	LockedAccuracyGate                       float64   `json:"locked_accuracy_gate"`
	MinimumPrimaryReduction                  float64   `json:"minimum_primary_reduction"`
	IntegrityContract                        string    `json:"integrity_contract"`
	DuplicatePolicy                          string    `json:"duplicate_policy"`
	CachePolicy                              string    `json:"cache_policy"`
	Alpha                                    float64   `json:"alpha"`
	BootstrapReplicates                      int       `json:"bootstrap_replicates"`
	RandomizationReplicates                  int       `json:"randomization_replicates"`
	BootstrapIndicesZeroBased                []int     `json:"bootstrap_indices_zero_based"`
	ContrastSeedRule                         string    `json:"contrast_seed_rule"`
	TiePolicy                                string    `json:"tie_policy"`
	MutationEnabled                          bool      `json:"mutation_enabled"`
}

func PreregisteredManifest() Manifest {
	return Manifest{
		ExperimentVersion: "active-causal-diagnosis/v2", GeneratorVersion: "three-binary-scm/v2", HypothesisVersion: "ordered-dag-mechanisms/v1", AcquisitionVersion: "lexicographic-pairs/v1", OracleVersion: "independent-scm-enumerator/v2", TeacherVersion: "opaque-single-response/v2", TrainingVersion: "credit-curriculum/v2", BaselineVersion: "exact-partition-policies/v2", CostVersion: "intervention-cost/v1", ReportVersion: "causal-diagnosis-report/v2", StatisticsVersion: "paired-resampling/v1", ProfileVersion: "causal-profile/v2",
		DevelopmentSeeds: SeedRange{112001, 16, 1}, TrainingSeeds: SeedRange{122001, 12, 1}, ValidationSeeds: SeedRange{132001, 32, 1}, LockedSeeds: SeedRange{142001, 64, 1}, CohortAssignment: "index-mod-8:0-3-cost-skewed,4-5-balanced,6-equivalence,7-irrelevant", LockedCohorts: Cohorts{32, 16, 8, 8},
		Variables: 3, MaximumIndegree: 2, MaximumPool: 32, MinimumInitialPosterior: 8, LegalInterventions: 6, MaximumConsumedInterventions: 10, CandidateAcquisitionRules: 40, InterventionCostMinimum: 1, InterventionCostMaximum: 100, EpisodeCostCeiling: 1000, InvalidOrExhaustedScore: 1001,
		EpisodeEngineCycleCap: 5000, EpisodeAttributedUnitCap: 1000, DescriptorByteCap: 8192, CurriculumAttributedUnitCap: 4096, CurriculumSemanticWorkCap: 32768, CurriculumEngineCycleCap: 2048, EpisodeHypothesisEvaluationCap: 4096, TrainingHypothesisEvaluationCap: 2000000, CertificateReplaySemanticWorkCap: 3932160, PostSelectionReplaySemanticWorkCap: 3932160, OracleAuditWorkCap: 4000000, ControlWorkCap: 262144, ControlAttributedUnitCap: 18000, DynamicStateCap: 531441, DynamicWorkCap: 4000000, EpisodeSemanticWorkCap: 8192, TeacherEvaluationCap: 10,
		ReportByteCap: 16777216, FixtureRecordByteCap: 8192, EvaluationFixtureBaseByteCap: 6144, EvaluationFixtureMeterItemsByteCap: 2048, TrainingFixtureByteCap: 16384, ApplicationCertificateByteCap: 1024, TaskMeterItemByteCap: 1024, ControlCertificateByteCap: 4096, ControlBundleByteCap: 2097152,
		ControlEvidenceByteCap: 4194304, ControlEvidenceNoCreditArtifactsByteCap: 1572864, ControlEvidenceCorruptionBaselineByteCap: 524288, ControlEvidenceCorruptionCasesByteCap: 1048576, ControlEvidenceDependencyFilesByteCap: 262144, ControlEvidenceOtherRecordsByteCap: 524288,
		TrainingEpisodeBaseByteCap: 6144, EpisodeMeterItemsByteCap: 2048, TrainingEpisodeReportByteCap: 8192, TrainingEpisodeBundleByteCap: 8388608, NonrecordReportByteCap: 1048576, MaximumLimitations: 32, LimitationByteCap: 512,
		LockedAccuracyGate: 1, MinimumPrimaryReduction: .10, IntegrityContract: "budgeted-transcript/v2", DuplicatePolicy: "canonical-code-deduplicate-before-profile", CachePolicy: "episode-policy-local-semantic-partition-cache/v2", Alpha: .05, BootstrapReplicates: 10000, RandomizationReplicates: 10000, BootstrapIndicesZeroBased: []int{249, 9749}, ContrastSeedRule: "active-causal-diagnosis/v2|locked|<information-gain|cost-skewed>|<randomization|bootstrap>", TiePolicy: "all-ties-reported-first-semantic-code-executed", MutationEnabled: false,
	}
}

func ValidateManifest(manifest Manifest) error {
	if !reflect.DeepEqual(manifest, PreregisteredManifest()) {
		return ErrManifestMismatch
	}
	return nil
}

var ErrManifestMismatch = manifestError("manifest does not equal the accepted active-causal-diagnosis/v2 manifest")

type manifestError string

func (e manifestError) Error() string { return string(e) }
