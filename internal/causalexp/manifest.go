package causalexp

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
type Manifest struct {
	ExperimentVersion               string    `json:"experiment_version"`
	GeneratorVersion                string    `json:"generator_version"`
	HypothesisVersion               string    `json:"hypothesis_version"`
	AcquisitionVersion              string    `json:"acquisition_version"`
	OracleVersion                   string    `json:"oracle_version"`
	TeacherVersion                  string    `json:"teacher_version"`
	TrainingVersion                 string    `json:"training_version"`
	BaselineVersion                 string    `json:"baseline_version"`
	CostVersion                     string    `json:"cost_version"`
	ReportVersion                   string    `json:"report_version"`
	StatisticsVersion               string    `json:"statistics_version"`
	ProfileVersion                  string    `json:"profile_version"`
	DevelopmentSeeds                SeedRange `json:"development_seeds"`
	TrainingSeeds                   SeedRange `json:"training_seeds"`
	ValidationSeeds                 SeedRange `json:"validation_seeds"`
	LockedSeeds                     SeedRange `json:"locked_seeds"`
	CohortAssignment                string    `json:"cohort_assignment"`
	LockedCohorts                   Cohorts   `json:"locked_cohorts"`
	Variables                       int       `json:"variables"`
	MaximumIndegree                 int       `json:"maximum_indegree"`
	MaximumPool                     int       `json:"maximum_pool"`
	MinimumInitialPosterior         int       `json:"minimum_initial_posterior"`
	LegalInterventions              int       `json:"legal_interventions"`
	MaximumConsumedInterventions    int       `json:"maximum_consumed_interventions"`
	CandidateAcquisitionRules       int       `json:"candidate_acquisition_rules"`
	InterventionCostMinimum         int       `json:"intervention_cost_minimum"`
	InterventionCostMaximum         int       `json:"intervention_cost_maximum"`
	EpisodeCostCeiling              int       `json:"episode_cost_ceiling"`
	InvalidOrExhaustedScore         int       `json:"invalid_or_exhausted_score"`
	EpisodeEngineCycleCap           int       `json:"episode_engine_cycle_cap"`
	EpisodeAttributedUnitCap        int       `json:"episode_attributed_unit_cap"`
	DescriptorByteCap               int       `json:"descriptor_byte_cap"`
	CurriculumAttributedUnitCap     int       `json:"curriculum_attributed_unit_cap"`
	CurriculumSemanticWorkCap       int       `json:"curriculum_semantic_work_cap"`
	CurriculumEngineCycleCap        int       `json:"curriculum_engine_cycle_cap"`
	EpisodeHypothesisEvaluationCap  int       `json:"episode_hypothesis_evaluation_cap"`
	TrainingHypothesisEvaluationCap int       `json:"training_hypothesis_evaluation_cap"`
	DynamicStateCap                 int       `json:"dynamic_state_cap"`
	DynamicWorkCap                  int       `json:"dynamic_work_cap"`
	EpisodeSemanticWorkCap          int       `json:"episode_semantic_work_cap"`
	ReportByteCap                   int       `json:"report_byte_cap"`
	FixtureRecordByteCap            int       `json:"fixture_record_byte_cap"`
	ApplicationCertificateByteCap   int       `json:"application_certificate_byte_cap"`
	TrainingEpisodeReportByteCap    int       `json:"training_episode_report_byte_cap"`
	TrainingEpisodeBundleByteCap    int       `json:"training_episode_bundle_byte_cap"`
	NonrecordReportByteCap          int       `json:"nonrecord_report_byte_cap"`
	MaximumLimitations              int       `json:"maximum_limitations"`
	LimitationByteCap               int       `json:"limitation_byte_cap"`
	LockedAccuracyGate              float64   `json:"locked_accuracy_gate"`
	MinimumPrimaryReduction         float64   `json:"minimum_primary_reduction"`
	IntegrityContract               string    `json:"integrity_contract"`
	DuplicatePolicy                 string    `json:"duplicate_policy"`
	CachePolicy                     string    `json:"cache_policy"`
	Alpha                           float64   `json:"alpha"`
	BootstrapReplicates             int       `json:"bootstrap_replicates"`
	RandomizationReplicates         int       `json:"randomization_replicates"`
	BootstrapIndicesZeroBased       []int     `json:"bootstrap_indices_zero_based"`
	ContrastSeedRule                string    `json:"contrast_seed_rule"`
	TiePolicy                       string    `json:"tie_policy"`
	MutationEnabled                 bool      `json:"mutation_enabled"`
}

func PreregisteredManifest() Manifest {
	return Manifest{
		ExperimentVersion: "active-causal-diagnosis/v1", GeneratorVersion: "three-binary-scm/v1", HypothesisVersion: "ordered-dag-mechanisms/v1", AcquisitionVersion: "lexicographic-pairs/v1", OracleVersion: "independent-scm-enumerator/v1", TeacherVersion: "opaque-single-response/v1", TrainingVersion: "credit-curriculum/v1", BaselineVersion: "exact-partition-policies/v1", CostVersion: "intervention-cost/v1", ReportVersion: "causal-diagnosis-report/v1", StatisticsVersion: "paired-resampling/v1", ProfileVersion: "causal-profile/v1",
		DevelopmentSeeds: SeedRange{12001, 16, 1}, TrainingSeeds: SeedRange{22001, 12, 1}, ValidationSeeds: SeedRange{32001, 32, 1}, LockedSeeds: SeedRange{42001, 64, 1}, CohortAssignment: "index-mod-8:0-3-cost-skewed,4-5-balanced,6-equivalence,7-irrelevant", LockedCohorts: Cohorts{32, 16, 8, 8},
		Variables: 3, MaximumIndegree: 2, MaximumPool: 32, MinimumInitialPosterior: 8, LegalInterventions: 6, MaximumConsumedInterventions: 10, CandidateAcquisitionRules: 40, InterventionCostMinimum: 1, InterventionCostMaximum: 100, EpisodeCostCeiling: 1000, InvalidOrExhaustedScore: 1001, EpisodeEngineCycleCap: 5000, EpisodeAttributedUnitCap: 1000, DescriptorByteCap: 8192, CurriculumAttributedUnitCap: 4096, CurriculumSemanticWorkCap: 32768, CurriculumEngineCycleCap: 2048, EpisodeHypothesisEvaluationCap: 4096, TrainingHypothesisEvaluationCap: 2000000, DynamicStateCap: 531441, DynamicWorkCap: 4000000, EpisodeSemanticWorkCap: 8192, ReportByteCap: 16777216, FixtureRecordByteCap: 8192, ApplicationCertificateByteCap: 1024, TrainingEpisodeReportByteCap: 8192, TrainingEpisodeBundleByteCap: 8388608, NonrecordReportByteCap: 1048576, MaximumLimitations: 32, LimitationByteCap: 512,
		LockedAccuracyGate: 1, MinimumPrimaryReduction: .10, IntegrityContract: "budgeted-transcript", DuplicatePolicy: "canonical-code-deduplicate-before-profile", CachePolicy: "episode-policy-local-semantic-partition-cache", Alpha: .05, BootstrapReplicates: 10000, RandomizationReplicates: 10000, BootstrapIndicesZeroBased: []int{249, 9749}, ContrastSeedRule: "active-causal-diagnosis/v1|locked|<information-gain|cost-skewed>|<randomization|bootstrap>", TiePolicy: "all-ties-reported-first-semantic-code-executed", MutationEnabled: false,
	}
}
