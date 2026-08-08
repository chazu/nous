package ruleinduction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

type ArtifactSnapshot struct {
	Name  string         `json:"name"`
	Slots map[string]any `json:"slots"`
}

// StageArtifactDigest commits to the complete authoritative stage artifact
// set. Callers construct snapshots independently from their own stores.
func StageArtifactDigest(records []ArtifactSnapshot) string {
	sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })
	return SemanticDigest(records)
}

const (
	ExperimentVersion = "rule-induction/v1"
	GeneratorVersion  = "closure-pairs/v1"
	GrammarVersion    = "binary-horn-three-relations/v1"
	CostVersion       = "semantic-work/v1"
	CandidateCap      = 15
	EvaluationCap     = 31
	FixedPointStepCap = 1000000
	SemanticWorkCap   = 500000
	EngineCycleCap    = 1000
	AttributedUnitCap = 100000
	ReportByteCap     = 8388608
)

var validPolicies = map[string]bool{
	"naive-direct": true, "lff-direct": true,
	"lff-task-local-invention": true, "uniform-random": true,
	"shared-recomputed": true, "shared-inlined": true,
	"shared-library": true,
}

type CategoryBindings struct {
	Partial     string `json:"partial"`
	Refinement  string `json:"refinement"`
	Candidate   string `json:"candidate"`
	Result      string `json:"result"`
	Observation string `json:"observation"`
	Evidence    string `json:"evidence"`
	Constraint  string `json:"constraint"`
	Comparison  string `json:"comparison"`
	Prune       string `json:"prune"`
	Library     string `json:"library"`
	Provenance  string `json:"provenance"`
	Projection  string `json:"projection"`
	Transcript  string `json:"transcript"`
	Boundary    string `json:"boundary"`
	Corpus      string `json:"corpus"`
	Selection   string `json:"selection"`
	Terminal    string `json:"terminal"`
}

func (b CategoryBindings) Ordered() []string {
	return []string{b.Partial, b.Refinement, b.Candidate, b.Result, b.Observation, b.Evidence, b.Constraint, b.Comparison, b.Prune, b.Library, b.Provenance, b.Projection, b.Transcript, b.Boundary, b.Corpus, b.Selection, b.Terminal}
}

type TaskBindings struct {
	Start    string `json:"start"`
	Refine   string `json:"refine"`
	Evaluate string `json:"evaluate"`
	Continue string `json:"continue"`
}

type ExperimentProfile struct {
	ProfileVersion    string           `json:"profile_version"`
	ExperimentVersion string           `json:"experiment_version"`
	GeneratorVersion  string           `json:"generator_version"`
	GrammarVersion    string           `json:"grammar_version"`
	CostVersion       string           `json:"cost_version"`
	OracleVersion     string           `json:"oracle_version"`
	ReportVersion     string           `json:"report_version"`
	BaselineVersion   string           `json:"baseline_version"`
	StatisticsVersion string           `json:"statistics_version"`
	QueueVersion      string           `json:"queue_version"`
	CacheVersion      string           `json:"cache_version"`
	IntegrityContract string           `json:"integrity_contract"`
	Panel             string           `json:"panel"`
	Seed              int64            `json:"seed"`
	Policy            string           `json:"policy"`
	Categories        CategoryBindings `json:"categories"`
	Tasks             TaskBindings     `json:"tasks"`
	ConstantBindings  []string         `json:"constant_bindings"`
	PredicateBindings []string         `json:"predicate_bindings"`
	Metarules         []string         `json:"metarules"`
	Stage1Queue       []string         `json:"stage1_queue"`
	Stage2Queue       []string         `json:"stage2_queue"`
	CandidateCap      int              `json:"candidate_cap"`
	EvaluationCap     int              `json:"evaluation_cap"`
	FixedPointStepCap int              `json:"fixed_point_step_cap"`
	SemanticWorkCap   int              `json:"semantic_work_cap"`
	EngineCycleCap    int              `json:"engine_cycle_cap"`
	AttributedUnitCap int              `json:"attributed_unit_cap"`
	ReportByteCap     int              `json:"report_byte_cap"`
	InitialPriority   int              `json:"initial_priority"`
	RefinePriority    int              `json:"refine_priority"`
	EvaluatePriority  int              `json:"evaluate_priority"`
	InitialReason     string           `json:"initial_reason"`
}

func (p ExperimentProfile) Valid() bool {
	if p.ProfileVersion != "rule-induction-profile/v1" || p.ExperimentVersion != ExperimentVersion || p.GeneratorVersion != GeneratorVersion || p.GrammarVersion != GrammarVersion || p.CostVersion != CostVersion ||
		p.OracleVersion != "independent-fixed-point/v1" || p.ReportVersion != "rule-induction-report/v1" || p.BaselineVersion != "factored-direct-lff/v1" || p.StatisticsVersion != "paired-resampling/v1" || p.QueueVersion != "policy-queues/v1" || p.CacheVersion != "semantic-definition-stage-cache/v1" || p.IntegrityContract != "budgeted-transcript" ||
		p.Panel == "" || !validPolicies[p.Policy] || p.CandidateCap < 1 || p.CandidateCap > CandidateCap || p.EvaluationCap != EvaluationCap || p.FixedPointStepCap != FixedPointStepCap || p.SemanticWorkCap != SemanticWorkCap || p.EngineCycleCap != EngineCycleCap || p.AttributedUnitCap != AttributedUnitCap || p.ReportByteCap != ReportByteCap ||
		p.InitialPriority < 1 || p.RefinePriority < 1 || p.EvaluatePriority < 1 || p.InitialReason == "" || len(p.ConstantBindings) != ConstantCount || len(p.PredicateBindings) != PredicateCount || len(p.Metarules) != 3 || !validCodeQueue(p.Stage1Queue, p.CandidateCap) || !validCodeQueue(p.Stage2Queue, p.CandidateCap) {
		return false
	}
	seen := map[string]bool{}
	bindings := append([]string{}, p.Categories.Ordered()...)
	bindings = append(bindings, p.ConstantBindings...)
	bindings = append(bindings, p.PredicateBindings...)
	bindings = append(bindings, p.Metarules...)
	bindings = append(bindings, p.Tasks.Start, p.Tasks.Refine, p.Tasks.Evaluate, p.Tasks.Continue)
	for _, value := range bindings {
		if value == "" || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func validCodeQueue(queue []string, cap int) bool {
	want, _ := CanonicalCodes(EnumerateDefinitions())
	if len(queue) != cap {
		return false
	}
	legal, seen := map[string]bool{}, map[string]bool{}
	for _, code := range want {
		legal[code] = true
	}
	for _, code := range queue {
		if !legal[code] || seen[code] {
			return false
		}
		seen[code] = true
	}
	return true
}

func profileKey(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (p ExperimentProfile) Key() (string, error) {
	if !p.Valid() {
		return "", errors.New("invalid rule-induction experiment profile")
	}
	return profileKey(p)
}

type StageProfile struct {
	ProfileVersion       string `json:"profile_version"`
	ExperimentProfileKey string `json:"experiment_profile_key"`
	Stage                int    `json:"stage"`
	FactDigest           string `json:"fact_digest"`
	ExampleDigest        string `json:"example_digest"`
	PriorBoundaryDigest  string `json:"prior_boundary_digest"`
}

func (p StageProfile) Key() (string, error) {
	if p.ProfileVersion != "rule-induction-stage-profile/v1" || len(p.ExperimentProfileKey) != 71 || (p.Stage != 1 && p.Stage != 2) || len(p.FactDigest) != 71 || len(p.ExampleDigest) != 71 || (p.Stage == 1 && p.PriorBoundaryDigest != "") || (p.Stage == 2 && len(p.PriorBoundaryDigest) != 71) {
		return "", errors.New("invalid rule-induction stage profile")
	}
	return profileKey(p)
}

func SemanticDigest(value any) string {
	key, _ := profileKey(value)
	return key
}

func ArtifactSemanticKey(kind, semantic string) string {
	sum := sha256.Sum256([]byte(ExperimentVersion + "\x00" + kind + "\x00" + semantic))
	return "sha256:" + hex.EncodeToString(sum[:])
}
