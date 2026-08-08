package ruleinductionexp

type SeedRange struct {
	Start int64 `json:"start"`
	Count int   `json:"count"`
	Step  int64 `json:"step"`
}

type CohortCounts struct {
	Beneficial int `json:"beneficial"`
	Neutral    int `json:"neutral"`
	Harmful    int `json:"harmful"`
}

type PolicyQueues struct {
	NaiveDirect       string `json:"naive-direct"`
	UniformRandom     string `json:"uniform-random"`
	AllCausalPolicies string `json:"all-causal-policies"`
}

type Manifest struct {
	ExperimentVersion                   string       `json:"experiment_version"`
	GeneratorVersion                    string       `json:"generator_version"`
	GrammarVersion                      string       `json:"grammar_version"`
	OracleVersion                       string       `json:"oracle_version"`
	CostVersion                         string       `json:"cost_version"`
	ReportVersion                       string       `json:"report_version"`
	BaselineVersion                     string       `json:"baseline_version"`
	StatisticsVersion                   string       `json:"statistics_version"`
	QueueVersion                        string       `json:"queue_version"`
	CacheVersion                        string       `json:"cache_version"`
	IntegrityContract                   string       `json:"integrity_contract"`
	DevelopmentSeeds                    SeedRange    `json:"development_seeds"`
	TrainingSeeds                       SeedRange    `json:"training_seeds"`
	ValidationSeeds                     SeedRange    `json:"validation_seeds"`
	LockedSeeds                         SeedRange    `json:"locked_seeds"`
	CohortAssignment                    string       `json:"cohort_assignment"`
	LockedCohorts                       CohortCounts `json:"locked_cohorts"`
	Constants                           int          `json:"constants"`
	BackgroundPredicates                int          `json:"background_predicates"`
	TargetPredicates                    int          `json:"target_predicates"`
	InventedPredicates                  int          `json:"invented_predicates"`
	Variables                           []string     `json:"variables"`
	Metarules                           []string     `json:"metarules"`
	ClausesPerTheory                    int          `json:"clauses_per_theory"`
	BodyLiteralsPerClause               int          `json:"body_literals_per_clause"`
	NormalizedDefinitionsPerTask        int          `json:"normalized_definitions_per_task"`
	NormalizedJointTheories             int          `json:"normalized_joint_theories"`
	CompleteCandidateEvaluationCap      int          `json:"complete_candidate_evaluation_cap"`
	FixedPointStepCap                   int          `json:"fixed_point_step_cap"`
	TotalSemanticWorkCap                int          `json:"total_semantic_work_cap"`
	EngineCycleCap                      int          `json:"engine_cycle_cap"`
	AttributedUnitCap                   int          `json:"attributed_unit_cap"`
	ReportByteCap                       int          `json:"report_byte_cap"`
	LockedAccuracyGate                  float64      `json:"locked_accuracy_gate"`
	MinimumPrimaryReduction             float64      `json:"minimum_primary_reduction"`
	MinimumRecomputedReduction          float64      `json:"minimum_recomputed_reduction"`
	MinimumInlinedReduction             float64      `json:"minimum_inlined_reduction"`
	MaximumHarmfulRatio                 float64      `json:"maximum_harmful_ratio"`
	MinimumBeneficialCandidateReduction float64      `json:"minimum_beneficial_stage2_candidate_reduction"`
	PolicyQueues                        PolicyQueues `json:"policy_queues"`
	CachePolicy                         string       `json:"cache_policy"`
	Alpha                               float64      `json:"alpha"`
	ConfidenceInterval                  string       `json:"confidence_interval"`
	PairedTest                          string       `json:"paired_test"`
	BootstrapReplicates                 int          `json:"bootstrap_replicates"`
	RandomizationReplicates             int          `json:"randomization_replicates"`
	BootstrapIndicesZeroBased           []int        `json:"bootstrap_indices_zero_based"`
	ContrastSeedRule                    string       `json:"contrast_seed_rule"`
	TiePolicy                           string       `json:"tie_policy"`
	AuditWorkInPrimary                  bool         `json:"audit_work_in_primary"`
	MutationEnabled                     bool         `json:"mutation_enabled"`
}

func PreregisteredManifest() Manifest {
	return Manifest{
		ExperimentVersion: "rule-induction/v1", GeneratorVersion: "closure-pairs/v1", GrammarVersion: "binary-horn-three-relations/v1", OracleVersion: "independent-fixed-point/v1", CostVersion: "semantic-work/v1", ReportVersion: "rule-induction-report/v1", BaselineVersion: "factored-direct-lff/v1", StatisticsVersion: "paired-resampling/v1", QueueVersion: "policy-queues/v1", CacheVersion: "semantic-definition-stage-cache/v1", IntegrityContract: "budgeted-transcript",
		DevelopmentSeeds: SeedRange{11001, 16, 1}, TrainingSeeds: SeedRange{21001, 64, 1}, ValidationSeeds: SeedRange{31001, 32, 1}, LockedSeeds: SeedRange{41001, 64, 1}, CohortAssignment: "index-mod-16:0-9-beneficial,10-12-neutral,13-15-harmful", LockedCohorts: CohortCounts{40, 12, 12},
		Constants: 8, BackgroundPredicates: 3, TargetPredicates: 2, InventedPredicates: 1, Variables: []string{"X", "Y", "Z"}, Metarules: []string{"identity", "tailrec", "invented-projection"}, ClausesPerTheory: 4, BodyLiteralsPerClause: 2, NormalizedDefinitionsPerTask: 15, NormalizedJointTheories: 240,
		CompleteCandidateEvaluationCap: 31, FixedPointStepCap: 1000000, TotalSemanticWorkCap: 500000, EngineCycleCap: 1000, AttributedUnitCap: 100000, ReportByteCap: 8388608,
		LockedAccuracyGate: 1.0, MinimumPrimaryReduction: 0.15, MinimumRecomputedReduction: 0.15, MinimumInlinedReduction: 0.05, MaximumHarmfulRatio: 2.0, MinimumBeneficialCandidateReduction: 0.25,
		PolicyQueues: PolicyQueues{"canonical-code", "uniform-order-stage-<1|2>", "causal-order-stage-<1|2>"}, CachePolicy: "stage-local-semantic-key;only-shared-library-carries-frozen-I", Alpha: 0.05, ConfidenceInterval: "paired-bootstrap-two-sided-95", PairedTest: "paired-randomization-two-sided", BootstrapReplicates: 10000, RandomizationReplicates: 10000, BootstrapIndicesZeroBased: []int{249, 9749}, ContrastSeedRule: "rule-induction/v1|locked|<direct|task-local|recomputed|inlined|beneficial-candidates>|<randomization|bootstrap>", TiePolicy: "first-exact-frozen-queue-report-evaluated-exact-ties", AuditWorkInPrimary: false, MutationEnabled: false,
	}
}
