package ruleinductionexp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"os/exec"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/ruleinductionoracle"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	rivocab "github.com/chazu/nous/internal/vocab/ruleinduction"
)

type Policy string

const (
	planCommit              = "77b67226fabb61c9295e4cb80c879815e46567ce"
	NaiveDirect      Policy = "naive-direct"
	LFFDirect        Policy = "lff-direct"
	TaskLocal        Policy = "lff-task-local-invention"
	UniformRandom    Policy = "uniform-random"
	SharedRecomputed Policy = "shared-recomputed"
	SharedInlined    Policy = "shared-inlined"
	SharedLibrary    Policy = "shared-library"
)

var policies = []Policy{NaiveDirect, LFFDirect, TaskLocal, UniformRandom, SharedInlined, SharedRecomputed, SharedLibrary}

// canonicalCodes is production configuration, not an oracle-derived universe.
// The pure vocabulary proves this is exactly the normalized v1 grammar; the
// independent oracle may enumerate only after a production run.
var canonicalCodes = []string{"01", "02", "03", "04", "05", "12", "13", "14", "15", "23", "24", "25", "34", "35", "45"}

type FixtureReport struct {
	Seed                   int64      `json:"seed"`
	Cohort                 Cohort     `json:"cohort"`
	Terminal               string     `json:"terminal"`
	Stage1Definition       string     `json:"stage1_definition"`
	Stage2Definition       string     `json:"stage2_definition"`
	UsedFrozenLibrary      bool       `json:"used_frozen_library"`
	FellBack               bool       `json:"fell_back"`
	CandidatesConsumed     int        `json:"candidates_consumed"`
	CandidatesExecuted     int        `json:"candidates_executed"`
	CandidatesPruned       int        `json:"candidates_pruned"`
	Constraints            int        `json:"constraints"`
	Comparisons            int        `json:"comparisons"`
	Stage1ExactTies        []string   `json:"stage1_exact_ties"`
	Stage2ExactTies        []string   `json:"stage2_exact_ties"`
	FixedPointSteps        int        `json:"fixed_point_steps"`
	EngineCycles           int        `json:"engine_cycles"`
	AttributedUnits        int        `json:"attributed_units"`
	ExperimentComplete     bool       `json:"experiment_complete"`
	AgendaDrained          bool       `json:"agenda_drained"`
	StageBoundaryImmutable bool       `json:"stage_boundary_immutable"`
	HeldOutStoreUnchanged  bool       `json:"heldout_store_unchanged"`
	Work                   WorkReport `json:"work"`
	HeldOutCorrect         int        `json:"heldout_correct"`
	HeldOutTotal           int        `json:"heldout_total"`
	Accuracy               float64    `json:"accuracy"`
	TerminalDigest         string     `json:"terminal_digest"`

	Stage1Candidates     int    `json:"-"`
	Stage2Candidates     int    `json:"-"`
	OracleAgreements     int    `json:"-"`
	OracleDisagreements  int    `json:"-"`
	OracleWork           int    `json:"-"`
	ExperimentProfileKey string `json:"-"`
	Stage1ProfileKey     string `json:"-"`
	Stage2ProfileKey     string `json:"-"`
}

type WorkReport struct {
	PartialAST        int `json:"partial_ast"`
	FixedPoint        int `json:"fixed_point"`
	Theta             int `json:"theta"`
	Cache             int `json:"cache"`
	AllocationProbes  int `json:"allocation_probes"`
	ArtifactEnvelopes int `json:"artifact_envelopes"`
	TranscriptDigest  int `json:"transcript_digest"`
	Selection         int `json:"selection"`
	Total             int `json:"total"`
}

type AggregateReport struct {
	Name                 Cohort  `json:"name"`
	Fixtures             int     `json:"fixtures"`
	Identified           int     `json:"identified"`
	NoSolution           int     `json:"no_solution"`
	BudgetExhausted      int     `json:"budget_exhausted"`
	CandidatesConsumed   int     `json:"candidates_consumed"`
	CandidatesExecuted   int     `json:"candidates_executed"`
	CandidatesPruned     int     `json:"candidates_pruned"`
	Stage2CandidatesMean float64 `json:"stage2_candidates_mean"`
	FixedPointSteps      int     `json:"fixed_point_steps"`
	TotalWork            int     `json:"total_work"`
	MeanWork             float64 `json:"mean_work"`
	HeldOutCorrect       int     `json:"heldout_correct"`
	HeldOutTotal         int     `json:"heldout_total"`
	Accuracy             float64 `json:"accuracy"`
}

type PolicyReport struct {
	Name     Policy            `json:"name"`
	Fixtures []FixtureReport   `json:"fixtures"`
	Overall  AggregateReport   `json:"overall"`
	Cohorts  []AggregateReport `json:"cohorts"`

	Identified       int `json:"-"`
	Correct          int `json:"-"`
	FixedPointWork   int `json:"-"`
	Stage2Candidates int `json:"-"`
	HeldOutCorrect   int `json:"-"`
	HeldOutTotal     int `json:"-"`
}

type MechanicalReport struct {
	AllValid                        bool `json:"all_valid"`
	DependencyBoundary              bool `json:"dependency_boundary"`
	OracleAgreements                int  `json:"oracle_agreements"`
	OracleDisagreements             int  `json:"oracle_disagreements"`
	TranscriptValid                 bool `json:"transcript_valid"`
	StageBoundaryImmutable          bool `json:"stage_boundary_immutable"`
	TrainingStoreUnchangedByHoldout bool `json:"training_store_unchanged_by_holdout"`
	AuditWork                       int  `json:"audit_work"`
}

type ContrastReport struct {
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

type GateReport struct {
	Accuracy, DirectReduction, DirectPValue, DirectCI, TaskLocalReduction, TaskLocalPValue, TaskLocalCI, RecomputedReduction, RecomputedPValue, RecomputedCI, InlinedAccuracyEqual, InlinedCandidateScheduleEqual, InlinedReduction, InlinedPValue, InlinedCI, BeneficialSearch, BeneficialSearchPValue, BeneficialSearchCI, HarmfulRatio bool
}

func (g GateReport) MarshalJSON() ([]byte, error) {
	type wire struct {
		Accuracy                      bool `json:"accuracy"`
		DirectReduction               bool `json:"direct_reduction"`
		DirectPValue                  bool `json:"direct_p_value"`
		DirectCI                      bool `json:"direct_ci"`
		TaskLocalReduction            bool `json:"task_local_reduction"`
		TaskLocalPValue               bool `json:"task_local_p_value"`
		TaskLocalCI                   bool `json:"task_local_ci"`
		RecomputedReduction           bool `json:"recomputed_reduction"`
		RecomputedPValue              bool `json:"recomputed_p_value"`
		RecomputedCI                  bool `json:"recomputed_ci"`
		InlinedAccuracyEqual          bool `json:"inlined_accuracy_equal"`
		InlinedCandidateScheduleEqual bool `json:"inlined_candidate_schedule_equal"`
		InlinedReduction              bool `json:"inlined_reduction"`
		InlinedPValue                 bool `json:"inlined_p_value"`
		InlinedCI                     bool `json:"inlined_ci"`
		BeneficialSearch              bool `json:"beneficial_search"`
		BeneficialSearchPValue        bool `json:"beneficial_search_p_value"`
		BeneficialSearchCI            bool `json:"beneficial_search_ci"`
		HarmfulRatio                  bool `json:"harmful_ratio"`
	}
	return json.Marshal(wire(g))
}

type ControlReport struct {
	OpaqueAlias, AlternateDescriptor, CaseOrder, OccupiedName, CandidateInsertCorruption, CandidateDeleteCorruption, CandidateDuplicateCorruption, CategoryInjection, AlternateQueueOmit, EvidencePositiveFlip, EvidenceNegativeFlip, WrongContext, MutationInert, HeldoutStoreImmutable, DeterministicJSON bool
}

func (c ControlReport) MarshalJSON() ([]byte, error) {
	type wire struct {
		OpaqueAlias                  bool `json:"opaque_alias"`
		AlternateDescriptor          bool `json:"alternate_descriptor"`
		CaseOrder                    bool `json:"case_order"`
		OccupiedName                 bool `json:"occupied_name"`
		CandidateInsertCorruption    bool `json:"candidate_insert_corruption"`
		CandidateDeleteCorruption    bool `json:"candidate_delete_corruption"`
		CandidateDuplicateCorruption bool `json:"candidate_duplicate_corruption"`
		CategoryInjection            bool `json:"category_injection"`
		AlternateQueueOmit           bool `json:"alternate_queue_omit"`
		EvidencePositiveFlip         bool `json:"evidence_positive_flip"`
		EvidenceNegativeFlip         bool `json:"evidence_negative_flip"`
		WrongContext                 bool `json:"wrong_context"`
		MutationInert                bool `json:"mutation_inert"`
		HeldoutStoreImmutable        bool `json:"heldout_store_immutable"`
		DeterministicJSON            bool `json:"deterministic_json"`
	}
	return json.Marshal(wire(c))
}

type Report struct {
	ReportVersion        string           `json:"report_version"`
	Manifest             Manifest         `json:"manifest"`
	ImplementationCommit string           `json:"implementation_commit"`
	PlanCommit           string           `json:"plan_commit"`
	Panel                string           `json:"panel"`
	Status               string           `json:"status"`
	Mechanical           MechanicalReport `json:"mechanical"`
	Policies             []PolicyReport   `json:"policies"`
	Contrasts            []ContrastReport `json:"contrasts"`
	Gates                GateReport       `json:"gates"`
	Controls             ControlReport    `json:"controls"`
	Limitations          []string         `json:"limitations"`
}

func (r Report) JSON() ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(output.Bytes(), []byte("\n")), nil
}

func queue(panel string, seedValue int64, stage int, policy Policy) []string {
	codes := append([]string(nil), canonicalCodes...)
	if policy == NaiveDirect {
		return codes
	}
	stream := "causal-order"
	if policy == UniformRandom {
		stream = "uniform-order"
	}
	material := fmt.Sprintf("closure-pairs/v1|%s|%d|%s-stage-%d|0", panel, seedValue, stream, stage)
	sum := sha256.Sum256([]byte(material))
	rng := rand.New(rand.NewPCG(binary.BigEndian.Uint64(sum[:8]), binary.BigEndian.Uint64(sum[8:16])))
	rng.Shuffle(len(codes), func(i, j int) { codes[i], codes[j] = codes[j], codes[i] })
	return codes
}

func withoutCode(queue []string, omitted string) []string {
	result := make([]string, 0, len(queue))
	for _, code := range queue {
		if code != omitted {
			result = append(result, code)
		}
	}
	return result
}

func definitionByCode(code string) ruleinductionoracle.Definition {
	for _, definition := range ruleinductionoracle.Definitions() {
		if ruleinductionoracle.Code(definition) == code {
			return definition
		}
	}
	return ruleinductionoracle.Definition{}
}

func RunDevelopment(domainsDir string) (Report, error) {
	return RunPanel(domainsDir, "development")
}

func RunPanel(domainsDir, panel string) (Report, error) {
	if panel == "locked" {
		return Report{}, fmt.Errorf("locked panel requires RunLockedPanel with an exact clean-checkout candidate commit")
	}
	return runPanel(domainsDir, panel, "")
}

func RunLockedPanel(domainsDir, implementationCommit string) (Report, error) {
	if implementationCommit == "" {
		return Report{}, fmt.Errorf("locked panel requires an implementation commit")
	}
	resolvedBytes, err := exec.Command("git", "rev-parse", "--verify", implementationCommit+"^{commit}").Output()
	if err != nil {
		return Report{}, fmt.Errorf("implementation commit is not an existing commit: %w", err)
	}
	resolved := strings.TrimSpace(string(resolvedBytes))
	headBytes, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(headBytes)) != resolved {
		return Report{}, fmt.Errorf("locked panel checkout HEAD does not equal candidate commit %s", resolved)
	}
	statusBytes, err := exec.Command("git", "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil || len(statusBytes) != 0 {
		return Report{}, fmt.Errorf("locked panel requires a clean checkout at candidate commit %s", resolved)
	}
	return runPanel(domainsDir, "locked", resolved)
}

func runPanel(domainsDir, panel, implementationCommit string) (Report, error) {
	manifest := PreregisteredManifest()
	seedRange, ok := map[string]SeedRange{"development": manifest.DevelopmentSeeds, "training": manifest.TrainingSeeds, "validation": manifest.ValidationSeeds, "locked": manifest.LockedSeeds}[panel]
	if !ok {
		return Report{}, fmt.Errorf("unknown rule-induction panel %q", panel)
	}
	report := Report{ReportVersion: "rule-induction-report/v1", Manifest: manifest, ImplementationCommit: implementationCommit, PlanCommit: planCommit, Panel: panel, Status: "invalid", Contrasts: []ContrastReport{}, Policies: []PolicyReport{}, Limitations: []string{}}
	report.Mechanical = MechanicalReport{AllValid: true, DependencyBoundary: true, TranscriptValid: true, StageBoundaryImmutable: true, TrainingStoreUnchangedByHoldout: true}
	for _, policy := range policies {
		policyReport := PolicyReport{Name: policy}
		for index := 0; index < seedRange.Count; index++ {
			seedValue := seedRange.Start + int64(index)*seedRange.Step
			fixture, err := Generate(panel, seedValue, CohortForIndex(index))
			if err != nil {
				return report, err
			}
			fixtureReport, err := runPolicy(domainsDir, fixture, policy)
			if err != nil {
				return report, fmt.Errorf("policy %s seed %d: %w", policy, seedValue, err)
			}
			policyReport.Fixtures = append(policyReport.Fixtures, fixtureReport)
			if fixtureReport.Terminal == "identified" {
				policyReport.Identified++
			}
			if fixtureReport.OracleDisagreements == 0 {
				policyReport.Correct++
			}
			report.Mechanical.OracleAgreements += fixtureReport.OracleAgreements
			report.Mechanical.OracleDisagreements += fixtureReport.OracleDisagreements
			policyReport.FixedPointWork += fixtureReport.Work.FixedPoint
			policyReport.Stage2Candidates += fixtureReport.Stage2Candidates
			policyReport.HeldOutCorrect += fixtureReport.HeldOutCorrect
			policyReport.HeldOutTotal += fixtureReport.HeldOutTotal
			if !fixtureMechanicallyWithinCaps(fixtureReport, report.Manifest) {
				report.Mechanical.AllValid = false
			}
			report.Mechanical.TranscriptValid = report.Mechanical.TranscriptValid && fixtureReport.ExperimentComplete
			report.Mechanical.StageBoundaryImmutable = report.Mechanical.StageBoundaryImmutable && fixtureReport.StageBoundaryImmutable
			report.Mechanical.TrainingStoreUnchangedByHoldout = report.Mechanical.TrainingStoreUnchangedByHoldout && fixtureReport.HeldOutStoreUnchanged
			report.Mechanical.AuditWork += fixtureReport.OracleWork
		}
		if panel == "development" {
			for seedValue := int64(51001); seedValue <= 51008; seedValue++ {
				fixture, err := GenerateNoSolution(seedValue)
				if err != nil {
					return report, err
				}
				fixtureReport, err := runPolicy(domainsDir, fixture, policy)
				if err != nil {
					return report, fmt.Errorf("policy %s no-solution seed %d: %w", policy, seedValue, err)
				}
				policyReport.Fixtures = append(policyReport.Fixtures, fixtureReport)
				policyReport.FixedPointWork += fixtureReport.Work.FixedPoint
				policyReport.Stage2Candidates += fixtureReport.Stage2Candidates
				report.Mechanical.OracleAgreements += fixtureReport.OracleAgreements
				report.Mechanical.OracleDisagreements += fixtureReport.OracleDisagreements
				if !fixtureMechanicallyWithinCaps(fixtureReport, report.Manifest) {
					report.Mechanical.AllValid = false
				}
				report.Mechanical.AuditWork += fixtureReport.OracleWork
				report.Mechanical.StageBoundaryImmutable = report.Mechanical.StageBoundaryImmutable && fixtureReport.StageBoundaryImmutable
			}
		}
		finalizePolicy(&policyReport)
		report.Policies = append(report.Policies, policyReport)
	}
	report.Mechanical.AllValid = report.Mechanical.AllValid && report.Mechanical.OracleDisagreements == 0
	computeContrasts(&report)
	report.Controls = runControls(domainsDir)
	report.Mechanical.AllValid = report.Mechanical.AllValid && allControlsValid(report.Controls)
	if report.Mechanical.AllValid {
		report.Status = "valid-null"
		if panel == "locked" && allGatesPassed(report.Gates) {
			report.Status = "valid-positive"
		}
	}
	if panel != "locked" {
		report.Limitations = []string{"Non-locked output is sensitivity evidence, not the preregistered confirmatory result.", "Confirmatory status is assigned only on the locked panel."}
	} else {
		report.Limitations = []string{"The bounded fixture distribution does not establish unrestricted inductive logic programming or open-ended EURISKO behavior."}
	}
	if encoded, err := report.JSON(); err != nil || len(encoded) > report.Manifest.ReportByteCap || bytes.Contains(encoded, []byte(": null")) {
		report.Mechanical.AllValid = false
		report.Status = "invalid"
	}
	return report, nil
}

func fixtureMechanicallyWithinCaps(f FixtureReport, manifest Manifest) bool {
	workValid := f.Work.Total <= manifest.TotalSemanticWorkCap
	return f.Terminal != "budget-exhausted" && f.ExperimentComplete && f.AgendaDrained && f.HeldOutStoreUnchanged && f.StageBoundaryImmutable && workValid && f.CandidatesExecuted <= manifest.CompleteCandidateEvaluationCap && f.FixedPointSteps <= manifest.FixedPointStepCap && f.EngineCycles <= manifest.EngineCycleCap && f.AttributedUnits <= manifest.AttributedUnitCap
}

func allControlsValid(c ControlReport) bool {
	return c.OpaqueAlias && c.AlternateDescriptor && c.CaseOrder && c.OccupiedName && c.CandidateInsertCorruption && c.CandidateDeleteCorruption && c.CandidateDuplicateCorruption && c.CategoryInjection && c.AlternateQueueOmit && c.EvidencePositiveFlip && c.EvidenceNegativeFlip && c.WrongContext && c.MutationInert && c.HeldoutStoreImmutable && c.DeterministicJSON
}

func allGatesPassed(g GateReport) bool {
	return g.Accuracy && g.DirectReduction && g.DirectPValue && g.DirectCI && g.TaskLocalReduction && g.TaskLocalPValue && g.TaskLocalCI && g.RecomputedReduction && g.RecomputedPValue && g.RecomputedCI && g.InlinedAccuracyEqual && g.InlinedCandidateScheduleEqual && g.InlinedReduction && g.InlinedPValue && g.InlinedCI && g.BeneficialSearch && g.BeneficialSearchPValue && g.BeneficialSearchCI && g.HarmfulRatio
}

func finalizePolicy(policy *PolicyReport) {
	policy.Overall = aggregateFixtures("", policy.Fixtures)
	for _, cohort := range []Cohort{Beneficial, Neutral, Harmful, Cohort("no-solution")} {
		var fixtures []FixtureReport
		for _, fixture := range policy.Fixtures {
			if fixture.Cohort == cohort {
				fixtures = append(fixtures, fixture)
			}
		}
		policy.Cohorts = append(policy.Cohorts, aggregateFixtures(cohort, fixtures))
	}
}

func aggregateFixtures(name Cohort, fixtures []FixtureReport) AggregateReport {
	result := AggregateReport{Name: name, Fixtures: len(fixtures)}
	stage2Candidates := 0
	for _, fixture := range fixtures {
		switch fixture.Terminal {
		case "identified":
			result.Identified++
		case "no-solution":
			result.NoSolution++
		case "budget-exhausted":
			result.BudgetExhausted++
		}
		result.CandidatesConsumed += fixture.CandidatesConsumed
		result.CandidatesExecuted += fixture.CandidatesExecuted
		result.CandidatesPruned += fixture.CandidatesPruned
		stage2Candidates += fixture.Stage2Candidates
		result.FixedPointSteps += fixture.FixedPointSteps
		result.TotalWork += fixture.Work.Total
		result.HeldOutCorrect += fixture.HeldOutCorrect
		result.HeldOutTotal += fixture.HeldOutTotal
	}
	if len(fixtures) > 0 {
		result.Stage2CandidatesMean = float64(stage2Candidates) / float64(len(fixtures))
		result.MeanWork = float64(result.TotalWork) / float64(len(fixtures))
	}
	if result.HeldOutTotal > 0 {
		result.Accuracy = float64(result.HeldOutCorrect) / float64(result.HeldOutTotal)
	}
	return result
}

func runPolicy(domainsDir string, fixture Fixture, policy Policy) (FixtureReport, error) {
	return runPolicyMode(domainsDir, fixture, policy, runOptions{}, nil)
}

type postRunHook func(*unit.Store, *unit.Unit) error
type runOptions struct {
	mutation, alternateDescriptor, occupiedName bool
	omitCode                                    string
	stage1Queue, stage2Queue                    []string
}

func runPolicyHook(domainsDir string, fixture Fixture, policy Policy, hook postRunHook) (FixtureReport, error) {
	return runPolicyMode(domainsDir, fixture, policy, runOptions{}, hook)
}

func runPolicyMode(domainsDir string, fixture Fixture, policy Policy, options runOptions, hook postRunHook) (FixtureReport, error) {
	previous := seed.DomainsDir
	seed.DomainsDir = domainsDir
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "ruleinduction"); err != nil {
		return FixtureReport{}, err
	}
	for _, name := range []string{"RuleInductionSeed", "RISeedPositiveOne", "RISeedPositiveTwo", "RISeedNegativeOne", "RISeedNegativeTwo"} {
		store.Delete(name)
	}

	experimentName := fmt.Sprintf("RIExperiment.%s.%d.%s", fixture.Panel, fixture.Seed, policy)
	stage1Queue := queue(fixture.Panel, fixture.Seed, 1, policy)
	stage2Queue := queue(fixture.Panel, fixture.Seed, 2, policy)
	if len(options.stage1Queue) > 0 {
		stage1Queue = append([]string(nil), options.stage1Queue...)
	}
	if len(options.stage2Queue) > 0 {
		stage2Queue = append([]string(nil), options.stage2Queue...)
	}
	if options.omitCode != "" {
		stage1Queue = withoutCode(stage1Queue, options.omitCode)
		stage2Queue = withoutCode(stage2Queue, options.omitCode)
	}
	profile := experimentProfile(fixture, policy, stage1Queue, stage2Queue)
	profile.CandidateCap = len(stage1Queue)
	if options.alternateDescriptor {
		profile.Categories = alternateCategoryBindings()
		installAlternateCategories(store, profile.Categories)
		profile.Tasks = rivocab.TaskBindings{Start: "altRIStart", Refine: "altRIRefine", Evaluate: "altRIEvaluate", Continue: "altRIContinue"}
		profile.Metarules = []string{"alt-identity", "alt-tailrec", "alt-invented-projection"}
		profile.InitialPriority, profile.RefinePriority, profile.EvaluatePriority = 960, 910, 810
		profile.InitialReason = "Alternate staged rule induction"
	}
	experimentProfileKey, err := profile.Key()
	if err != nil {
		return FixtureReport{}, err
	}
	facts := CanonicalFacts(fixture)
	stage1Records := exampleRecords(fixture.Stage1)
	stage1Profile := rivocab.StageProfile{ProfileVersion: "rule-induction-stage-profile/v1", ExperimentProfileKey: experimentProfileKey, Stage: 1, FactDigest: rivocab.SemanticDigest(facts), ExampleDigest: rivocab.SemanticDigest(stage1Records)}
	stage1ProfileKey, err := stage1Profile.Key()
	if err != nil {
		return FixtureReport{}, err
	}
	experiment := unit.New(experimentName)
	experiment.Set("isA", []string{"RuleInductionExperiment", "Anything"})
	experiment.Set("experimentKey", fmt.Sprintf("rule-induction/v1/%s/%d/%s", fixture.Panel, fixture.Seed, policy))
	experiment.Set("experiment", experimentName)
	experiment.Set("experimentProfileKey", experimentProfileKey)
	experiment.Set("artifactKind", "descriptor")
	experiment.Set("semanticKey", rivocab.ArtifactSemanticKey("descriptor", "experiment-descriptor"))
	experiment.Set("allocationProbes", 1)
	experiment.Set("stage1ProfileKey", stage1ProfileKey)
	experiment.Set("currentStageProfileKey", stage1ProfileKey)
	experiment.Set("stage", "stage1")
	experiment.Set("stageIndex", 1)
	experiment.Set("facts", facts)
	experiment.Set("queue", stage1Queue)
	experiment.Set("stage1Queue", stage1Queue)
	experiment.Set("stage2Queue", stage2Queue)
	experiment.Set("rootName", "RI.Partial."+experimentName+".stage1.root")
	experiment.Set("reuseMode", string(policy))
	setProfileBindings(experiment, profile)
	experiment.Set("initialTasks", []any{map[string]any{"priority": profile.InitialPriority, "slot": profile.Tasks.Start, "reason": profile.InitialReason}})
	store.Put(experiment)
	stage1Corpus := putCorpus(store, profile.Categories.Corpus, experimentName, experimentProfileKey, stage1ProfileKey, 1, facts, stage1Records, "")
	experiment.Set("stage1CorpusUnit", stage1Corpus.Name)
	if options.occupiedName {
		occupied := unit.New("RI.Partial." + experimentName + ".stage1.root")
		occupied.Set("isA", []string{"Anything"})
		occupied.Set("unrelated", true)
		store.Put(occupied)
	}

	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out, eng.VM.Out = io.Discard, io.Discard
	eng.MaxCycles = profile.EngineCycleCap / 4
	eng.MutConfig.Enabled = options.mutation
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		return FixtureReport{}, err
	}
	stage1Cycles := eng.Cycle()
	if experiment.GetString("terminal") != "awaiting-stage-2" || ag.Len() != 0 {
		return FixtureReport{}, fmt.Errorf("stage 1 terminal %q", experiment.GetString("terminal"))
	}
	stage1VM := dsl.NewVM(store, agenda.New(), nil)
	stage1Verified, err := stage1VM.Execute(fmt.Sprintf("%q ri-stage-one-complete?", experimentName))
	if err != nil || !stage1Verified.AsBool() {
		return FixtureReport{}, fmt.Errorf("stage 1 post-verifier rejected boundary: value=%v err=%v", stage1Verified, err)
	}
	stage1Code := experiment.GetString("frozenCode")
	boundaryDigest := stageArtifactDigest(store, experimentName, 1)
	boundary := unit.New("RI.Boundary." + experimentName)
	boundary.Set("isA", []string{profile.Categories.Boundary, "Anything"})
	boundary.Set("experiment", experimentName)
	boundary.Set("experimentProfileKey", experimentProfileKey)
	boundary.Set("stage", 1)
	boundary.Set("stageIndex", 1)
	boundary.Set("stageProfileKey", stage1ProfileKey)
	boundary.Set("artifactKind", "boundary")
	boundary.Set("semanticKey", rivocab.ArtifactSemanticKey("boundary", "stage-1-boundary"))
	boundary.Set("storeDigest", boundaryDigest)
	boundary.Set("frozenCode", stage1Code)
	boundary.Set("priorTerminal", "awaiting-stage-2")
	boundary.Set("allocationProbes", 1)
	store.Put(boundary)

	stage2Records := exampleRecords(fixture.Stage2)
	stage2Profile := rivocab.StageProfile{ProfileVersion: "rule-induction-stage-profile/v1", ExperimentProfileKey: experimentProfileKey, Stage: 2, FactDigest: stage1Profile.FactDigest, ExampleDigest: rivocab.SemanticDigest(stage2Records), PriorBoundaryDigest: boundaryDigest}
	stage2ProfileKey, err := stage2Profile.Key()
	if err != nil {
		return FixtureReport{}, err
	}
	stage2Corpus := putCorpus(store, profile.Categories.Corpus, experimentName, experimentProfileKey, stage2ProfileKey, 2, nil, stage2Records, boundary.Name)
	experiment.Set("stage2CorpusUnit", stage2Corpus.Name)
	experiment.Set("stage", "stage2")
	experiment.Set("stageIndex", 2)
	experiment.Set("stage2ProfileKey", stage2ProfileKey)
	experiment.Set("currentStageProfileKey", stage2ProfileKey)
	experiment.Set("queue", stage2Queue)
	experiment.Set("rootName", "RI.Partial."+experimentName+".stage2.root")
	if policy == SharedLibrary || policy == SharedInlined {
		ag.Push(&agenda.Task{Priority: profile.InitialPriority, UnitName: experimentName, SlotName: profile.Tasks.Continue})
	} else {
		experiment.Set("terminal", nil)
		experiment.Set("riStarted", nil)
		ag.Push(&agenda.Task{Priority: profile.InitialPriority, UnitName: experimentName, SlotName: profile.Tasks.Start})
	}
	eng.MaxCycles = profile.EngineCycleCap - profile.EngineCycleCap/4
	if err := eng.Run(context.Background()); err != nil {
		return FixtureReport{}, err
	}
	stage2Cycles := eng.Cycle()
	if hook != nil {
		if err := hook(store, experiment); err != nil {
			return FixtureReport{}, err
		}
	}

	result := FixtureReport{
		Seed: fixture.Seed, Cohort: fixture.Cohort, Terminal: experiment.GetString("terminal"), Stage1Definition: stage1Code, Stage2Definition: experiment.GetString("selectedCode"),
		UsedFrozenLibrary: experiment.GetBool("usedFrozenLibrary"), FellBack: experiment.GetBool("fellBack"), ExperimentComplete: experiment.GetBool("experimentComplete"), AgendaDrained: ag.Len() == 0,
		StageBoundaryImmutable: stageArtifactDigest(store, experimentName, 1) == boundaryDigest,
		Stage1ExactTies:        []string{}, Stage2ExactTies: []string{}, EngineCycles: stage1Cycles + stage2Cycles,
		ExperimentProfileKey: experimentProfileKey, Stage1ProfileKey: stage1ProfileKey, Stage2ProfileKey: stage2ProfileKey,
	}
	result.Work = semanticLedger(store, experiment)
	result.TerminalDigest = experiment.GetString("stage2TranscriptDigest")
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("experiment") == experimentName && u.GetString("artifactKind") != "" {
			result.AttributedUnits++
		}
		if !directCategory(u, profile.Categories.Candidate) || u.GetString("experiment") != experimentName {
			continue
		}
		if u.GetBool("riEvaluated") {
			result.CandidatesExecuted++
			result.CandidatesConsumed++
			if u.GetString("stage") == "stage1" {
				result.Stage1Candidates++
			} else {
				result.Stage2Candidates++
			}
			result.FixedPointSteps += u.GetInt("fixedWork")
		}
		if u.GetBool("riPruned") {
			result.CandidatesPruned++
			result.CandidatesConsumed++
			if u.GetString("stage") == "stage1" {
				result.Stage1Candidates++
			} else {
				result.Stage2Candidates++
			}
		}
	}
	audit, auditErr := auditProductionRun(store, experiment, fixture, policy)
	result.OracleAgreements, result.OracleDisagreements, result.OracleWork = audit.Agreements, audit.Disagreements, audit.Work
	result.Stage1ExactTies, result.Stage2ExactTies = audit.Stage1Ties, audit.Stage2Ties
	result.Constraints, result.Comparisons, result.CandidatesPruned = audit.Constraints, audit.Comparisons, audit.Prunes
	if auditErr != nil {
		return result, auditErr
	}
	beforeHeldOut, err := store.CanonicalJSON()
	if err != nil {
		return FixtureReport{}, err
	}
	if result.Terminal == "identified" {
		heldBackground := productionBackground(fixture.HeldOut)
		held1, _, err := rivocab.Evaluate(productionDefinition(result.Stage1Definition), heldBackground)
		if err != nil {
			return FixtureReport{}, err
		}
		held2, _, err := rivocab.Evaluate(productionDefinition(result.Stage2Definition), heldBackground)
		if err != nil {
			return FixtureReport{}, err
		}
		result.HeldOutCorrect = agreementCountProduction(held1, fixture.HeldTarget1) + agreementCountProduction(held2, fixture.HeldTarget2)
		result.HeldOutTotal = 128
		result.Accuracy = float64(result.HeldOutCorrect) / float64(result.HeldOutTotal)
		oracleHeld1 := ruleinductionoracle.Evaluate(definition(result.Stage1Definition), fixture.HeldOut)
		oracleHeld2 := ruleinductionoracle.Evaluate(definition(result.Stage2Definition), fixture.HeldOut)
		oracleCorrect := agreementCount(oracleHeld1, fixture.HeldTarget1) + agreementCount(oracleHeld2, fixture.HeldTarget2)
		result.OracleWork += 2
		if held1.Signature() != oracleHeld1.Signature() || held2.Signature() != oracleHeld2.Signature() || oracleCorrect != result.HeldOutCorrect {
			result.OracleDisagreements++
			return result, fmt.Errorf("oracle held-out disagreement production=%s/%s/%d oracle=%s/%s/%d", held1.Signature(), held2.Signature(), result.HeldOutCorrect, oracleHeld1.Signature(), oracleHeld2.Signature(), oracleCorrect)
		}
		result.OracleAgreements++
	}
	afterHeldOut, err := store.CanonicalJSON()
	if err != nil {
		return FixtureReport{}, err
	}
	result.HeldOutStoreUnchanged = string(beforeHeldOut) == string(afterHeldOut)
	return result, nil
}

func semanticLedger(store *unit.Store, experiment *unit.Unit) WorkReport {
	var work WorkReport
	artifacts, transcripts := 0, 0
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("experiment") != experiment.Name || u.GetString("experimentProfileKey") != experiment.GetString("experimentProfileKey") || u.GetString("artifactKind") == "" {
			continue
		}
		artifacts++
		switch u.GetString("artifactKind") {
		case "candidate":
			if u.GetBool("riEvaluated") {
				work.FixedPoint += u.GetInt("fixedWork") + u.GetInt("exampleCount")
				work.Cache++
			}
		case "comparison":
			work.Theta += u.GetInt("thetaWork")
		case "transcript":
			transcripts++
			switch u.GetString("action") {
			case "start", "refine", "constraint", "local-invention":
				work.PartialAST += u.GetInt("domainWork")
			case "promotion", "termination", "fallback":
				work.Selection += u.GetInt("domainWork")
			}
		}
	}
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("experiment") == experiment.Name && u.GetString("experimentProfileKey") == experiment.GetString("experimentProfileKey") && u.GetString("artifactKind") != "" {
			probes := u.GetInt("allocationProbes")
			if probes == 0 {
				probes = 1
			}
			work.AllocationProbes += probes
		}
	}
	work.ArtifactEnvelopes = artifacts * 32
	work.TranscriptDigest = transcripts * 16
	work.Total = work.PartialAST + work.FixedPoint + work.Theta + work.Cache + work.AllocationProbes + work.ArtifactEnvelopes + work.TranscriptDigest + work.Selection
	return work
}

func productionDefinition(code string) rivocab.Definition {
	for _, definition := range rivocab.EnumerateDefinitions() {
		candidate, _ := definition.Code()
		if candidate == code {
			return definition
		}
	}
	return rivocab.Definition{}
}

func productionBackground(input [3]ruleinductionoracle.Relation) [3]rivocab.Relation {
	var output [3]rivocab.Relation
	for predicate := 0; predicate < 3; predicate++ {
		for x := 0; x < 8; x++ {
			for y := 0; y < 8; y++ {
				if input[predicate].Has(x, y) {
					output[predicate].Add(x, y)
				}
			}
		}
	}
	return output
}

func agreementCountProduction(first rivocab.Relation, second ruleinductionoracle.Relation) int {
	count := 0
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			if first.Has(x, y) == second.Has(x, y) {
				count++
			}
		}
	}
	return count
}

func agreementCount(first, second ruleinductionoracle.Relation) int {
	count := 0
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			if first.Has(x, y) == second.Has(x, y) {
				count++
			}
		}
	}
	return count
}

func directCategory(u *unit.Unit, category string) bool {
	if u == nil || u.Name == category {
		return false
	}
	for _, parent := range u.GetStrings("isA") {
		if parent == category {
			return true
		}
	}
	return false
}

func stageArtifactDigest(store *unit.Store, experiment string, stage int) string {
	records := []rivocab.ArtifactSnapshot{}
	for _, name := range store.All() {
		u := store.Get(name)
		if u.GetString("experiment") != experiment || u.GetInt("stageIndex") != stage || u.GetString("artifactKind") == "" || u.GetString("artifactKind") == "boundary" || u.GetString("artifactKind") == "descriptor" {
			continue
		}
		slots := make(map[string]any, len(u.Slots))
		for slot, value := range u.Slots {
			slots[slot] = value
		}
		records = append(records, rivocab.ArtifactSnapshot{Name: name, Slots: slots})
	}
	return rivocab.StageArtifactDigest(records)
}

func exampleRecords(examples []ruleinductionoracle.Example) []string {
	records := make([]string, len(examples))
	for index, example := range examples {
		records[index] = fmt.Sprintf("%d:%d:%t", example.X, example.Y, example.Positive)
	}
	return records
}

func putCorpus(store *unit.Store, category, experiment, experimentProfileKey, stageProfileKey string, stage int, facts, examples []string, boundary string) *unit.Unit {
	u := unit.New(fmt.Sprintf("RI.Corpus.%s.stage%d", experiment, stage))
	u.Set("isA", []string{category, "Anything"})
	u.Set("experiment", experiment)
	u.Set("experimentProfileKey", experimentProfileKey)
	u.Set("stage", stage)
	u.Set("stageIndex", stage)
	u.Set("stageProfileKey", stageProfileKey)
	u.Set("artifactKind", "corpus")
	u.Set("semanticKey", rivocab.ArtifactSemanticKey("corpus", fmt.Sprintf("stage-%d-corpus", stage)))
	u.Set("facts", facts)
	u.Set("examples", examples)
	u.Set("boundary", boundary)
	u.Set("allocationProbes", 1)
	store.Put(u)
	return u
}

func alternateCategoryBindings() rivocab.CategoryBindings {
	return rivocab.CategoryBindings{Partial: "AltRI.Partial", Refinement: "AltRI.Refinement", Candidate: "AltRI.Candidate", Result: "AltRI.Result", Observation: "AltRI.Observation", Evidence: "AltRI.Evidence", Constraint: "AltRI.Constraint", Comparison: "AltRI.Comparison", Prune: "AltRI.Prune", Library: "AltRI.Library", Provenance: "AltRI.Provenance", Projection: "AltRI.Projection", Transcript: "AltRI.Transcript", Boundary: "AltRI.Boundary", Corpus: "AltRI.Corpus", Selection: "AltRI.Selection", Terminal: "AltRI.Terminal"}
}

func installAlternateCategories(store *unit.Store, bindings rivocab.CategoryBindings) {
	for _, name := range bindings.Ordered() {
		u := unit.New(name)
		u.Set("isA", []string{"Anything"})
		store.Put(u)
	}
}

func experimentProfile(fixture Fixture, policy Policy, stage1Queue, stage2Queue []string) rivocab.ExperimentProfile {
	return rivocab.ExperimentProfile{
		ProfileVersion: "rule-induction-profile/v1", ExperimentVersion: rivocab.ExperimentVersion, GeneratorVersion: rivocab.GeneratorVersion, GrammarVersion: rivocab.GrammarVersion, CostVersion: rivocab.CostVersion, OracleVersion: "independent-fixed-point/v1", ReportVersion: "rule-induction-report/v1", BaselineVersion: "factored-direct-lff/v1", StatisticsVersion: "paired-resampling/v1", QueueVersion: "policy-queues/v1", CacheVersion: "semantic-definition-stage-cache/v1", IntegrityContract: "budgeted-transcript",
		Panel: fixture.Panel, Seed: fixture.Seed, Policy: string(policy),
		Categories:       rivocab.CategoryBindings{Partial: "RuleInductionPartial", Refinement: "RuleInductionRefinement", Candidate: "RuleInductionCandidate", Result: "RuleInductionResult", Observation: "RuleInductionObservation", Evidence: "RuleInductionEvidence", Constraint: "RuleInductionConstraint", Comparison: "RuleInductionComparison", Prune: "RuleInductionPrune", Library: "RuleInductionLibrary", Provenance: "RuleInductionProvenance", Projection: "RuleInductionProjection", Transcript: "RuleInductionTranscript", Boundary: "RuleInductionBoundary", Corpus: "RuleInductionCorpus", Selection: "RuleInductionSelection", Terminal: "RuleInductionTerminal"},
		Tasks:            rivocab.TaskBindings{Start: "riStart", Refine: "riRefine", Evaluate: "riEvaluate", Continue: "riContinue"},
		ConstantBindings: fixture.ConstantAliases[:], PredicateBindings: fixture.PredicateAliases[:], Metarules: []string{"identity", "tailrec", "invented-projection"}, Stage1Queue: append([]string(nil), stage1Queue...), Stage2Queue: append([]string(nil), stage2Queue...),
		CandidateCap: rivocab.CandidateCap, EvaluationCap: rivocab.EvaluationCap, FixedPointStepCap: rivocab.FixedPointStepCap, SemanticWorkCap: rivocab.SemanticWorkCap, EngineCycleCap: rivocab.EngineCycleCap, AttributedUnitCap: rivocab.AttributedUnitCap, ReportByteCap: rivocab.ReportByteCap, InitialPriority: 950, RefinePriority: 900, EvaluatePriority: 800, InitialReason: "Start staged rule induction",
	}
}

func setProfileBindings(experiment *unit.Unit, profile rivocab.ExperimentProfile) {
	experiment.Set("profileVersion", profile.ProfileVersion)
	experiment.Set("experimentVersion", profile.ExperimentVersion)
	experiment.Set("generatorVersion", profile.GeneratorVersion)
	experiment.Set("grammarVersion", profile.GrammarVersion)
	experiment.Set("costVersion", profile.CostVersion)
	experiment.Set("oracleVersion", profile.OracleVersion)
	experiment.Set("reportVersion", profile.ReportVersion)
	experiment.Set("baselineVersion", profile.BaselineVersion)
	experiment.Set("statisticsVersion", profile.StatisticsVersion)
	experiment.Set("queueVersion", profile.QueueVersion)
	experiment.Set("cacheVersion", profile.CacheVersion)
	experiment.Set("integrityContract", profile.IntegrityContract)
	experiment.Set("panel", profile.Panel)
	experiment.Set("seed", int(profile.Seed))
	experiment.Set("policy", profile.Policy)
	experiment.Set("partialCategory", profile.Categories.Partial)
	experiment.Set("refinementCategory", profile.Categories.Refinement)
	experiment.Set("candidateCategory", profile.Categories.Candidate)
	experiment.Set("resultCategory", profile.Categories.Result)
	experiment.Set("observationCategory", profile.Categories.Observation)
	experiment.Set("evidenceCategory", profile.Categories.Evidence)
	experiment.Set("constraintCategory", profile.Categories.Constraint)
	experiment.Set("comparisonCategory", profile.Categories.Comparison)
	experiment.Set("pruneCategory", profile.Categories.Prune)
	experiment.Set("libraryCategory", profile.Categories.Library)
	experiment.Set("provenanceCategory", profile.Categories.Provenance)
	experiment.Set("projectionCategory", profile.Categories.Projection)
	experiment.Set("transcriptCategory", profile.Categories.Transcript)
	experiment.Set("boundaryCategory", profile.Categories.Boundary)
	experiment.Set("corpusCategory", profile.Categories.Corpus)
	experiment.Set("selectionCategory", profile.Categories.Selection)
	experiment.Set("terminalCategory", profile.Categories.Terminal)
	experiment.Set("startTaskSlot", profile.Tasks.Start)
	experiment.Set("refineTaskSlot", profile.Tasks.Refine)
	experiment.Set("evaluateTaskSlot", profile.Tasks.Evaluate)
	experiment.Set("continueTaskSlot", profile.Tasks.Continue)
	experiment.Set("predicateBindings", profile.PredicateBindings)
	experiment.Set("constantBindings", profile.ConstantBindings)
	experiment.Set("metarules", profile.Metarules)
	experiment.Set("candidateCap", profile.CandidateCap)
	experiment.Set("evaluationCap", profile.EvaluationCap)
	experiment.Set("fixedPointStepCap", profile.FixedPointStepCap)
	experiment.Set("semanticWorkCap", profile.SemanticWorkCap)
	experiment.Set("engineCycleCap", profile.EngineCycleCap)
	experiment.Set("attributedUnitCap", profile.AttributedUnitCap)
	experiment.Set("reportByteCap", profile.ReportByteCap)
	experiment.Set("initialPriority", profile.InitialPriority)
	experiment.Set("refinementPriority", profile.RefinePriority)
	experiment.Set("evaluationPriority", profile.EvaluatePriority)
	experiment.Set("initialReason", profile.InitialReason)
}
