// Package kuberepairexp runs the preregistered v2 Kubernetes repair experiment.
package kuberepairexp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"sort"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/credit"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/kuberepairfixture"
	"github.com/chazu/nous/internal/kuberepairoracle"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	kuberepair "github.com/chazu/nous/internal/vocab/kuberepair"
	programvocab "github.com/chazu/nous/internal/vocab/programsynth"
)

const (
	reportVersion    = "kubernetes-selector-reference-trials/v2"
	synthesisMethod  = "ordered-bound-atomic-edits-up-to-3/v1"
	failureLoss      = 402
	bootstrapSamples = 10000
)

type DiagnosticEvents struct {
	PlanApplications int `json:"plan_applications"`
	PlansEmitted     int `json:"plans_emitted"`
	ScoreKeys        int `json:"score_keys"`
	Comparisons      int `json:"comparisons"`
	OracleAudits     int `json:"oracle_audits"`
}

type PolicyResult struct {
	Solved             bool             `json:"solved"`
	TerminalCalls      int              `json:"terminal_calls"`
	SelectedLength     int              `json:"selected_length"`
	SelectedPlan       string           `json:"selected_plan,omitempty"`
	TraceSHA256        string           `json:"trace_sha256"`
	OrderSHA256        string           `json:"order_sha256"`
	UnsafeAccepted     bool             `json:"unsafe_accepted"`
	OracleDisagreement bool             `json:"oracle_disagreement"`
	Diagnostics        DiagnosticEvents `json:"diagnostics"`
}

type TaskResult struct {
	ID                    string       `json:"id"`
	Cohort                string       `json:"cohort"`
	Stratum               string       `json:"stratum,omitempty"`
	Edits                 int          `json:"edits"`
	EligiblePlans         int          `json:"eligible_plans"`
	SyntacticCandidates   int          `json:"syntactic_candidates"`
	MinimumLength         int          `json:"minimum_length"`
	MinimumPlans          int          `json:"minimum_plans"`
	MinimumSemanticStates int          `json:"minimum_semantic_states"`
	Contextual            PolicyResult `json:"contextual"`
	Constraint            PolicyResult `json:"constraint"`
	NoCredit              PolicyResult `json:"no_credit"`
	WrongContext          PolicyResult `json:"wrong_context"`
	Reset                 PolicyResult `json:"reset"`
	Scalar                PolicyResult `json:"scalar"`
}

type Aggregate struct {
	Tasks                   int     `json:"tasks"`
	MeanContextualLoss      float64 `json:"mean_contextual_loss"`
	MeanConstraintLoss      float64 `json:"mean_constraint_loss"`
	MeanNoCreditLoss        float64 `json:"mean_no_credit_loss"`
	ContextualVsConstraint  float64 `json:"contextual_vs_constraint"`
	ContextualVsNoCredit    float64 `json:"contextual_vs_no_credit"`
	BootstrapConstraintLow  float64 `json:"bootstrap_constraint_low"`
	BootstrapConstraintHigh float64 `json:"bootstrap_constraint_high"`
	BootstrapNoCreditLow    float64 `json:"bootstrap_no_credit_low"`
	BootstrapNoCreditHigh   float64 `json:"bootstrap_no_credit_high"`
}

type TrainingReport struct {
	Seeds               []int64  `json:"seeds"`
	Tasks               int      `json:"tasks"`
	CandidatesCreated   int      `json:"candidates_created"`
	TerminalCalls       int      `json:"terminal_calls"`
	CreditedFeatures    []string `json:"credited_features"`
	CanonicalProfileSHA string   `json:"canonical_profile_sha256"`
	CallLogSHA256       string   `json:"call_log_sha256"`
}

type PhaseATaskResult struct {
	ID                    string `json:"id"`
	Terminal              string `json:"terminal"`
	Edits                 int    `json:"edits"`
	Candidates            int    `json:"candidates"`
	MinimumLength         int    `json:"minimum_length"`
	MinimumPlans          int    `json:"minimum_plans"`
	MinimumSemanticStates int    `json:"minimum_semantic_states"`
	EditUniverseEqual     bool   `json:"edit_universe_equal"`
	MinimumSetEqual       bool   `json:"minimum_set_equal"`
	SemanticStatesEqual   bool   `json:"semantic_states_equal"`
	EvidenceComplete      bool   `json:"evidence_complete"`
}

type PhaseAReport struct {
	Positive       bool               `json:"positive"`
	IntegrityValid bool               `json:"integrity_valid"`
	Tasks          []PhaseATaskResult `json:"tasks"`
}

type PowerReport struct {
	OuterReplicates int     `json:"outer_replicates"`
	InnerReplicates int     `json:"inner_replicates"`
	Passing         int     `json:"passing"`
	Power           float64 `json:"power"`
	Accepted        bool    `json:"accepted"`
}

type Report struct {
	Version         string               `json:"version"`
	Panel           string               `json:"panel"`
	RootHex         string               `json:"root_hex"`
	Outcome         string               `json:"outcome"`
	IntegrityValid  bool                 `json:"integrity_valid"`
	PhaseA          PhaseAReport         `json:"phase_a"`
	Training        TrainingReport       `json:"training"`
	Component       Aggregate            `json:"component_primary"`
	TwoFeature      Aggregate            `json:"two_feature"`
	ThreeFeature    Aggregate            `json:"three_feature_recombined"`
	Cohorts         map[string]Aggregate `json:"cohorts"`
	Power           *PowerReport         `json:"power,omitempty"`
	Tasks           []TaskResult         `json:"tasks"`
	Limitations     []string             `json:"limitations"`
	IntegrityErrors []string             `json:"integrity_errors,omitempty"`
}

func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

type creditProfile struct {
	decision  map[string]int
	component map[string]int
	position  map[string]int
	relation  map[string]int
	scalar    map[string]int
}

func emptyProfile() creditProfile {
	return creditProfile{decision: map[string]int{}, component: map[string]int{}, position: map[string]int{}, relation: map[string]int{}, scalar: map[string]int{}}
}

type plan struct {
	edits     []int
	result    string
	features  []string
	relations []string
	key       string
}

type trialCase struct {
	c       kuberepairfixture.Case
	cohort  string
	stratum string
}

func Run(domainsDir, panel string) (Report, error) {
	if panel == "locked" {
		return Report{}, fmt.Errorf("v2 locked panel requires guarded RunLockedV2")
	}
	root, phaseACount, count, err := panelSpec(panel)
	if err != nil {
		return Report{}, err
	}
	if panel == "validation" {
		development, devErr := runPanel(domainsDir, "development", rootHex(751001), 6, 24, true)
		if devErr != nil {
			return Report{}, fmt.Errorf("development prerequisite: %w", devErr)
		}
		if development.Power == nil || !development.Power.Accepted {
			return Report{}, fmt.Errorf("development power %.3f is below 0.80", development.Power.Power)
		}
	}
	return runPanel(domainsDir, panel, rootHex(root), phaseACount, count, panel == "development")
}

func panelSpec(panel string) (int64, int, int, error) {
	switch panel {
	case "development":
		return 751001, 6, 24, nil
	case "validation":
		return 752001, 12, 48, nil
	default:
		return 0, 0, 0, fmt.Errorf("unknown v2 panel %q", panel)
	}
}

func runPanel(domainsDir, panel, root string, phaseACount, count int, computePower bool) (Report, error) {
	profile, training, err := train(domainsDir)
	if err != nil {
		return Report{}, fmt.Errorf("training: %w", err)
	}
	phaseA, err := runPhaseA(domainsDir, root, phaseACount)
	if err != nil {
		return Report{}, fmt.Errorf("phase A: %w", err)
	}
	report := Report{
		Version: reportVersion, Panel: panel, RootHex: root, PhaseA: phaseA, Training: training,
		IntegrityValid: phaseA.IntegrityValid, Cohorts: map[string]Aggregate{},
		Limitations: []string{
			"Synthetic bounded objects and human-supplied edit features do not establish production Kubernetes repair.",
			"The primary endpoint counts terminal evaluations, not wall time or scalable agenda pruning.",
			"Component tasks deliberately match the three-feature training curriculum.",
		},
	}
	tasks, err := panelCases(root, count)
	if err != nil {
		return report, err
	}
	for index, task := range tasks {
		result, runErr := runTask(task, profile, root, index)
		if runErr != nil {
			report.IntegrityValid = false
			report.IntegrityErrors = append(report.IntegrityErrors, fmt.Sprintf("task %s: %v", task.c.ID, runErr))
			report.Outcome = "invalid"
			return report, nil
		}
		report.Tasks = append(report.Tasks, result)
	}
	for _, cohort := range []string{"exact", "cross-role", "unrelated", "co-minimal", "already-correct", "no-solution"} {
		report.Cohorts[cohort] = aggregate(report.Tasks, cohort, "", 0, root, cohort)
	}
	report.TwoFeature = aggregate(report.Tasks, "component", "two-feature", training.TerminalCalls, root, "two-no-credit")
	report.ThreeFeature = aggregate(report.Tasks, "component", "three-feature-recombined", training.TerminalCalls, root, "three-no-credit")
	report.Component = aggregate(report.Tasks, "component", "", training.TerminalCalls, root, "component")
	report.Cohorts["component"] = report.Component
	report.IntegrityValid = report.IntegrityValid && integrity(report, profile)
	if computePower {
		power := estimatePower(report.Tasks, training.TerminalCalls, root)
		report.Power = &power
	}
	report.Outcome = classify(report)
	return report, nil
}

func train(domainsDir string) (creditProfile, TrainingReport, error) {
	profile := emptyProfile()
	cases, err := kuberepairfixture.Training(750001)
	if err != nil {
		return profile, TrainingReport{}, err
	}
	report := TrainingReport{Seeds: []int64{750001, 750002, 750003}, Tasks: len(cases)}
	var callLog []string
	for caseIndex, caseData := range cases {
		caseData = withOpaqueHandle(caseData, rootHex(750001), caseIndex)
		cleanup, registerErr := kuberepair.RegisterIntent(caseData.Handle, caseData.Intent)
		if registerErr != nil {
			return profile, report, registerErr
		}
		handle, _ := kuberepair.EncodeHandle(caseData.Handle)
		_ = kuberepair.EqualOrSatisfies(caseData.Public, handle)
		emptyLog := kuberepair.EvaluationLog(caseData.Handle)
		if len(emptyLog) != 1 {
			cleanup()
			return profile, report, fmt.Errorf("training %s empty-call log = %d", caseData.ID, len(emptyLog))
		}
		callLog = append(callLog, emptyLog...)
		report.TerminalCalls += len(emptyLog)
		cleanup()

		store, selected, candidates, discoverCleanup, runErr := discover(domainsDir, caseData)
		if runErr != nil {
			if discoverCleanup != nil {
				discoverCleanup()
			}
			return profile, report, runErr
		}
		if len(selected) != 1 || store.Get(selected[0]).GetInt("programLength") != 1 {
			discoverCleanup()
			return profile, report, fmt.Errorf("training %s selected %d non-unique programs", caseData.ID, len(selected))
		}
		report.CandidatesCreated += candidates
		candidateLog := kuberepair.EvaluationLog(caseData.Handle)
		if len(candidateLog) == 0 || len(candidateLog) > candidates {
			discoverCleanup()
			return profile, report, fmt.Errorf("training %s candidate-call log = %d for %d candidates", caseData.ID, len(candidateLog), candidates)
		}
		callLog = append(callLog, candidateLog...)
		report.TerminalCalls += len(candidateLog)
		creditEngine := engine.New(store, agenda.New())
		creditEngine.Out = io.Discard
		creditEngine.VM.Out = io.Discard
		creditEngine.MutConfig.Enabled = false
		creditEngine.MaxCycles = 16
		if err := creditEngine.Run(context.Background()); err != nil {
			discoverCleanup()
			return profile, report, err
		}
		program := store.Get(selected[0])
		component := store.Get(program.GetStrings("components")[0])
		feature := component.GetString("creditFeatureKey")
		relation := component.GetString("creditRelationKey")
		subject := component.GetString("creditFeatureSubject")
		relationSubject := component.GetString("creditRelationSubject")
		decision, _ := credit.StructuralDecisionKey(synthesisMethod, []string{feature})
		if err := validateTrainingCredit(store, program, component, feature, relation, subject, relationSubject); err != nil {
			discoverCleanup()
			return profile, report, err
		}
		profile.decision[decision] += credit.RewardTotal(store, credit.DecisionTuple(kuberepair.CreditContext, decision))
		profile.component[feature] += credit.RewardTotal(store, credit.Tuple{Context: kuberepair.CreditContext, Subject: subject, Role: "component"})
		profile.position[feature+"|step-1"] += credit.RewardTotal(store, credit.Tuple{Context: kuberepair.CreditContext, Subject: subject, Role: "step-1"})
		profile.relation[relation] += credit.RewardTotal(store, credit.Tuple{Context: kuberepair.CreditContext, Subject: relationSubject, Role: "relation"})
		profile.scalar[feature] = store.Get(subject).Worth()
		discoverCleanup()
	}
	for feature := range profile.component {
		report.CreditedFeatures = append(report.CreditedFeatures, feature)
	}
	sort.Strings(report.CreditedFeatures)
	encoded, _ := json.Marshal(profileSnapshot(profile))
	digest := sha256.Sum256(encoded)
	report.CanonicalProfileSHA = hex.EncodeToString(digest[:])
	encodedLog, _ := json.Marshal(callLog)
	logDigest := sha256.Sum256(encodedLog)
	report.CallLogSHA256 = hex.EncodeToString(logDigest[:])
	return profile, report, nil
}

func validateTrainingCredit(store *unit.Store, program, component *unit.Unit, feature, relation, subject, relationSubject string) error {
	creditors := program.GetStrings("creditors")
	roles := program.GetStrings("creditRoles")
	semantics := []string{component.GetString("semanticOpcode")}
	concreteDecision, concreteErr := programvocab.DecisionKey(synthesisMethod, semantics)
	structuralDecision, structuralErr := credit.StructuralDecisionKey(synthesisMethod, []string{feature})
	if concreteErr != nil || structuralErr != nil ||
		program.GetString("synthesisMethod") != synthesisMethod ||
		program.GetString("creditContext") != kuberepair.CreditContext ||
		program.GetString("creditDecision") != concreteDecision ||
		!equalStringSlices(program.GetStrings("components"), []string{component.Name}) ||
		!equalStringSlices(program.GetStrings("semanticSequence"), semantics) ||
		!equalStringSlices(creditors, []string{"H-EnumerateBoundedPrograms", component.Name}) ||
		!equalStringSlices(roles, []string{"synthesis", "step-1"}) ||
		component.GetString("creditFeatureKey") != feature ||
		component.GetString("creditRelationKey") != relation ||
		component.GetString("creditFeatureSubject") != subject ||
		component.GetString("creditRelationSubject") != relationSubject ||
		store.Get(subject) == nil || store.Get(subject).GetString("creditFeatureKey") != feature ||
		store.Get(relationSubject) == nil || store.Get(relationSubject).GetString("creditRelationKey") != relation {
		return fmt.Errorf("training %s malformed concrete provenance", program.Name)
	}
	expected := map[credit.Tuple]int{
		credit.DecisionTuple(kuberepair.CreditContext, concreteDecision):                              300,
		{Context: kuberepair.CreditContext, Subject: "H-EnumerateBoundedPrograms", Role: "synthesis"}: 150,
		{Context: kuberepair.CreditContext, Subject: component.Name, Role: "step-1"}:                  150,
		credit.DecisionTuple(kuberepair.CreditContext, structuralDecision):                            300,
		{Context: kuberepair.CreditContext, Subject: subject, Role: "component"}:                      150,
		{Context: kuberepair.CreditContext, Subject: subject, Role: "step-1"}:                         150,
		{Context: kuberepair.CreditContext, Subject: relationSubject, Role: "relation"}:               150,
	}
	if feature == "" || relation == "" || len(store.Examples(credit.Category))-1 != len(expected) {
		return fmt.Errorf("training %s incomplete structural declaration or record set", program.Name)
	}
	for tuple, want := range expected {
		record := credit.Lookup(store, tuple)
		if record == nil || record.GetInt("rewardTotal") != want || record.GetInt("evidenceCount") != 1 || record.GetString("lastSourceUnit") != program.Name {
			return fmt.Errorf("training %s invalid credit tuple %#v", program.Name, tuple)
		}
	}
	if store.Get(subject).Worth() != 650 {
		return fmt.Errorf("training %s feature worth = %d", program.Name, store.Get(subject).Worth())
	}
	return nil
}

func profileSnapshot(profile creditProfile) map[string]map[string]int {
	return map[string]map[string]int{"decision": profile.decision, "component": profile.component, "position": profile.position, "relation": profile.relation, "scalar": profile.scalar}
}

func discover(domainsDir string, c kuberepairfixture.Case) (*unit.Store, []string, int, func(), error) {
	cleanup, err := kuberepair.RegisterIntent(c.Handle, c.Intent)
	if err != nil {
		return nil, nil, 0, nil, err
	}
	previous := seed.DomainsDir
	seed.DomainsDir = domainsDir
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "kuberepair"); err != nil {
		cleanup()
		return nil, nil, 0, nil, err
	}
	for _, name := range store.Examples("KubeAtomicEdit") {
		if name != "KubeAtomicEdit" {
			store.Delete(name)
		}
	}
	for index, encoded := range c.Edits {
		feature, relation, featureErr := kuberepair.FeatureKey(encoded)
		if featureErr != nil {
			cleanup()
			return nil, nil, 0, nil, featureErr
		}
		featureName := stableUnitName("KubeDynamicFeature", feature)
		relationName := stableUnitName("KubeDynamicRelation", relation)
		if !store.Has(featureName) {
			u := unit.New(featureName)
			u.SetWorth(500)
			u.Set("isA", []string{"KubeRepairFeature", "Anything"})
			u.Set("creditFeatureKey", feature)
			store.Put(u)
		}
		if !store.Has(relationName) {
			u := unit.New(relationName)
			u.SetWorth(500)
			u.Set("isA", []string{"KubeRepairRelation", "Anything"})
			u.Set("creditRelationKey", relation)
			store.Put(u)
		}
		u := unit.New(fmt.Sprintf("KubeDynamicEdit%02d", index))
		u.SetWorth(500)
		u.Set("isA", []string{"KubeAtomicEdit", "UnaryOp", "Op", "Anything"})
		u.Set("domain", []string{"KubeRepairValue"})
		u.Set("range", []string{"KubeRepairValue"})
		u.Set("arity", 1)
		u.Set("semanticOpcode", semanticOpcode(encoded))
		u.Set("creditFeatureSubject", featureName)
		u.Set("creditFeatureKey", feature)
		u.Set("creditRelationSubject", relationName)
		u.Set("creditRelationKey", relation)
		u.Set("defn", fmt.Sprintf("%q kube-apply-edit-b64", base64.RawURLEncoding.EncodeToString([]byte(encoded))))
		store.Put(u)
	}
	handle, _ := kuberepair.EncodeHandle(c.Handle)
	store.Get("KubeRepairSeedExample").Set("input", c.Public)
	store.Get("KubeRepairSeedExample").Set("expected", handle)
	store.Get("KubeRepairSeedProbe").Set("data", c.Public)
	experiment := store.Get("KubeRepairExperiment")
	experiment.Set("experimentKey", "dynamic-v2/task")
	experiment.Set("expectedCandidateCount", 0)
	experiment.Set("evaluatedCandidateCount", 0)
	eng := engine.New(store, agenda.New())
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MutConfig.Enabled = false
	eng.WorkOnTask(&agenda.Task{Priority: 800, UnitName: experiment.Name, SlotName: experiment.GetString("synthesisTaskSlot")})
	for _, candidate := range experiment.GetStrings("candidateUnits") {
		eng.WorkOnTask(&agenda.Task{Priority: 700, UnitName: candidate, SlotName: experiment.GetString("evaluationTaskSlot")})
	}
	eng.WorkOnTask(&agenda.Task{Priority: 600, UnitName: experiment.Name, SlotName: experiment.GetString("finalizationTaskSlot")})
	return store, experiment.GetStrings("selectedPrograms"), len(experiment.GetStrings("candidateUnits")), cleanup, nil
}

func stableUnitName(prefix, key string) string {
	sum := sha256.Sum256([]byte(key))
	return prefix + "-" + hex.EncodeToString(sum[:8])
}
func semanticOpcode(encoded string) string {
	sum := sha256.Sum256([]byte(encoded))
	return "bound-" + hex.EncodeToString(sum[:8])
}

func runPhaseA(domainsDir, root string, count int) (PhaseAReport, error) {
	cases, err := phaseACases(root, count)
	if err != nil {
		return PhaseAReport{}, err
	}
	report := PhaseAReport{Positive: true, IntegrityValid: true}
	for _, c := range cases {
		intent := oracleIntent(c)
		analysis, analyzeErr := kuberepairoracle.Analyze(c.Public, intent, 3)
		if analyzeErr != nil {
			return report, analyzeErr
		}
		result := PhaseATaskResult{ID: c.ID, Terminal: analysis.Result.Terminal, Edits: len(c.Edits), MinimumLength: analysis.Result.MinimumLength, MinimumPlans: len(analysis.Result.Plans), MinimumSemanticStates: len(analysis.Result.States), EditUniverseEqual: equalStringSlices(c.Edits, analysis.Edits)}
		if len(c.Edits) < 1 || len(c.Edits) > 8 || !result.EditUniverseEqual {
			result.EvidenceComplete = false
			report.IntegrityValid = false
			report.Positive = false
			report.Tasks = append(report.Tasks, result)
			continue
		}
		cleanup, registerErr := kuberepair.RegisterIntent(c.Handle, c.Intent)
		if registerErr != nil {
			return report, registerErr
		}
		handle, _ := kuberepair.EncodeHandle(c.Handle)
		productionEmpty := kuberepair.EqualOrSatisfies(c.Public, handle)
		oracleEmpty, _ := kuberepairoracle.Satisfies(c.Public, intent)
		cleanup()
		if productionEmpty != oracleEmpty {
			report.IntegrityValid = false
			report.Positive = false
			report.Tasks = append(report.Tasks, result)
			continue
		}
		if productionEmpty {
			result.MinimumSetEqual = analysis.Result.Terminal == "already-correct"
			result.SemanticStatesEqual = result.MinimumSetEqual
			result.EvidenceComplete = true
			report.Tasks = append(report.Tasks, result)
			continue
		}
		store, selected, candidates, discoverCleanup, discoverErr := discover(domainsDir, c)
		if discoverErr != nil {
			if discoverCleanup != nil {
				discoverCleanup()
			}
			return report, discoverErr
		}
		result.Candidates = candidates
		expectedCandidates := len(c.Edits) + len(c.Edits)*len(c.Edits) + len(c.Edits)*len(c.Edits)*len(c.Edits)
		experiment := store.Get("KubeRepairExperiment")
		result.EvidenceComplete = candidates == expectedCandidates && experiment.GetInt("evaluatedCandidateCount") == candidates && experiment.GetBool("generationComplete") && experiment.GetBool("finalizationComplete")
		var actualSequences, actualStates []string
		for _, name := range selected {
			sequence := store.Get(name).GetStrings("semanticSequence")
			actualSequences = append(actualSequences, strings.Join(sequence, "/"))
		}
		// Finalized candidates contain materialized definitions. Delete the
		// dynamic primitives before replay so parity cannot be satisfied by
		// reconstructing fixture edits outside the synthesized artifact.
		for _, primitive := range store.Examples("KubeAtomicEdit") {
			if primitive != "KubeAtomicEdit" {
				store.Delete(primitive)
			}
		}
		replayEngine := engine.New(store, agenda.New())
		replayEngine.Out = io.Discard
		replayEngine.VM.Out = io.Discard
		for _, name := range selected {
			value, replayErr := replayEngine.VM.Execute(fmt.Sprintf("%q %q get-slot %q apply-op", "KubeRepairSeedExample", "input", name))
			if replayErr != nil || value.IsNil() {
				result.EvidenceComplete = false
				continue
			}
			actualStates = append(actualStates, semanticState(value.AsString()))
		}
		var expectedSequences []string
		for _, sequence := range analysis.Result.Plans {
			parts := make([]string, len(sequence))
			for i, editIndex := range sequence {
				parts[i] = semanticOpcode(c.Edits[editIndex])
			}
			expectedSequences = append(expectedSequences, strings.Join(parts, "/"))
		}
		sort.Strings(actualSequences)
		sort.Strings(expectedSequences)
		sort.Strings(actualStates)
		actualStates = uniqueStrings(actualStates)
		result.MinimumSetEqual = equalStringSlices(actualSequences, expectedSequences)
		result.SemanticStatesEqual = equalStringSlices(actualStates, analysis.Result.States)
		if !result.MinimumSetEqual || !result.SemanticStatesEqual || !result.EvidenceComplete {
			report.Positive = false
		}
		discoverCleanup()
		report.Tasks = append(report.Tasks, result)
	}
	return report, nil
}

func phaseACases(root string, count int) ([]kuberepairfixture.Case, error) {
	var out []kuberepairfixture.Case
	for i := 0; i < count; i++ {
		var value int64
		switch root {
		case rootHex(751001):
			value = 761001 + int64(i)
		case rootHex(752001):
			value = 762001 + int64(i)
		default:
			value = streamInt64("phase-a-case", root, i)
		}
		var c kuberepairfixture.Case
		var err error
		switch i % 6 {
		case 0:
			var training []kuberepairfixture.Case
			training, err = kuberepairfixture.Training(value)
			if err == nil {
				c = training[0]
			}
		case 1:
			c, err = kuberepairfixture.Recomposition(value, kuberepairfixture.FaultTemplate|kuberepairfixture.FaultService)
		case 2:
			c, err = kuberepairfixture.Recomposition(value, kuberepairfixture.FaultTemplate|kuberepairfixture.FaultService|kuberepairfixture.FaultExtraSelector)
		case 3:
			c, err = kuberepairfixture.Recomposition(value, 0)
		case 4:
			c, err = kuberepairfixture.Unrelated(value, true)
		case 5:
			c, err = kuberepairfixture.Unrelated(value, false)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, withOpaqueHandle(c, root, i))
	}
	return out, nil
}

func panelCases(root string, count int) ([]trialCase, error) {
	type cohortCount struct {
		cohort, stratum string
		count           int
	}
	counts := []cohortCount{{"exact", "", 6}, {"component", "two-feature", 6}, {"component", "three-feature-recombined", 6}, {"cross-role", "", 2}, {"unrelated", "", 1}, {"co-minimal", "", 1}, {"already-correct", "", 1}, {"no-solution", "", 1}}
	if count == 48 {
		for i := range counts {
			counts[i].count *= 2
		}
	}
	var out []trialCase
	ordinal := 0
	for _, spec := range counts {
		for j := 0; j < spec.count; j++ {
			value := streamInt64("phase-b-case", root, ordinal)
			var c kuberepairfixture.Case
			var err error
			switch spec.cohort {
			case "exact":
				var values []kuberepairfixture.Case
				values, err = kuberepairfixture.Training(value)
				if err == nil {
					c = values[j%3]
				}
			case "component":
				mask := kuberepairfixture.FaultTemplate | kuberepairfixture.FaultService
				if spec.stratum == "two-feature" && j%2 == 1 {
					mask = kuberepairfixture.FaultTemplate | kuberepairfixture.FaultExtraSelector
				}
				if spec.stratum == "three-feature-recombined" {
					mask = kuberepairfixture.FaultTemplate | kuberepairfixture.FaultService | kuberepairfixture.FaultExtraSelector
				}
				c, err = kuberepairfixture.Recomposition(value, mask)
			case "cross-role", "co-minimal":
				c, err = kuberepairfixture.CrossRole(value)
			case "unrelated":
				c, err = kuberepairfixture.Unrelated(value, false)
			case "already-correct":
				c, err = kuberepairfixture.Recomposition(value, 0)
			case "no-solution":
				c, err = kuberepairfixture.Unrelated(value, true)
			}
			if err != nil {
				return nil, err
			}
			c = withOpaqueHandle(c, root, 1000000+ordinal)
			out = append(out, trialCase{c: c, cohort: spec.cohort, stratum: spec.stratum})
			ordinal++
		}
	}
	if len(out) != count {
		return nil, fmt.Errorf("panel mapping produced %d tasks, want %d", len(out), count)
	}
	return out, nil
}

func enumeratePlans(c kuberepairfixture.Case) ([]plan, DiagnosticEvents, error) {
	var out []plan
	events := DiagnosticEvents{}
	var walk func(string, []int, int)
	walk = func(state string, sequence []int, remaining int) {
		if remaining == 0 {
			return
		}
		for index, edit := range c.Edits {
			events.PlanApplications++
			next, err := kuberepair.Apply(state, edit)
			if err != nil {
				continue
			}
			indices := append(append([]int(nil), sequence...), index)
			features := make([]string, len(indices))
			relations := make([]string, len(indices))
			parts := make([]string, len(indices))
			for i, editIndex := range indices {
				features[i], relations[i], _ = kuberepair.FeatureKey(c.Edits[editIndex])
				parts[i] = fmt.Sprintf("%03d", editIndex)
			}
			out = append(out, plan{edits: indices, result: next, features: features, relations: relations, key: strings.Join(parts, "/")})
			events.PlansEmitted++
			walk(next, indices, remaining-1)
		}
	}
	walk(c.Public, nil, 3)
	return out, events, nil
}

type ranking struct {
	order     []int
	orderHash string
	events    DiagnosticEvents
}

func runTask(task trialCase, profile creditProfile, root string, ordinal int) (TaskResult, error) {
	plans, commonEvents, err := enumeratePlans(task.c)
	if err != nil {
		return TaskResult{}, err
	}
	oracleEdits, err := kuberepairoracle.EnumerateEdits(task.c.Public)
	if err != nil {
		return TaskResult{}, err
	}
	if !equalStringSlices(task.c.Edits, oracleEdits) {
		return TaskResult{}, fmt.Errorf("production/oracle edit universe disagreement")
	}
	base := make([]int, len(plans))
	for i := range base {
		base[i] = i
	}
	rng := streamRand("permutation", root, ordinal)
	for i := len(base) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		base[i], base[j] = base[j], base[i]
	}
	base = groupByLength(base, plans)

	contextual := rankPlans(base, plans, func(p plan) []int { return contextScore(p, profile) }, true)
	empty := emptyProfile()
	wrong := rankPlans(base, plans, func(p plan) []int { return contextScore(p, empty) }, true)
	resetProfile := resetCreditProfile(plans)
	reset := rankPlans(base, plans, func(p plan) []int { return contextScore(p, resetProfile) }, true)
	scalar := rankPlans(base, plans, func(p plan) []int { return []int{scalarScore(p, profile)} }, true)
	constraint := rankPlans(base, plans, func(p plan) []int {
		vector, _ := kuberepair.PublicViolationVector(p.result)
		return negateForAscending(vector)
	}, true)
	noCredit := ranking{order: append([]int(nil), base...), orderHash: orderDigest(base, plans)}

	cleanup, err := kuberepair.RegisterIntent(task.c.Handle, task.c.Intent)
	if err != nil {
		return TaskResult{}, err
	}
	defer cleanup()
	handle, _ := kuberepair.EncodeHandle(task.c.Handle)
	runs := []*policyRun{
		searchProduction(task.c, plans, contextual, handle, commonEvents),
		searchProduction(task.c, plans, constraint, handle, commonEvents),
		searchProduction(task.c, plans, noCredit, handle, commonEvents),
		searchProduction(task.c, plans, wrong, handle, commonEvents),
		searchProduction(task.c, plans, reset, handle, commonEvents),
		searchProduction(task.c, plans, scalar, handle, commonEvents),
	}
	analysis, err := kuberepairoracle.Analyze(task.c.Public, oracleIntent(task.c), 3)
	if err != nil {
		return TaskResult{}, err
	}
	if !equalStringSlices(task.c.Edits, analysis.Edits) {
		return TaskResult{}, fmt.Errorf("post-search oracle edit universe disagreement")
	}
	if err := validateCohort(task, plans, analysis, profile); err != nil {
		return TaskResult{}, err
	}
	for _, run := range runs {
		auditPolicy(run, task.c, plans, analysis)
	}
	result := TaskResult{ID: task.c.ID, Cohort: task.cohort, Stratum: task.stratum, Edits: len(task.c.Edits), EligiblePlans: len(plans), SyntacticCandidates: len(task.c.Edits) + len(task.c.Edits)*len(task.c.Edits) + len(task.c.Edits)*len(task.c.Edits)*len(task.c.Edits), MinimumLength: analysis.Result.MinimumLength, MinimumPlans: len(analysis.Result.Plans), MinimumSemanticStates: len(analysis.Result.States)}
	result.Contextual = runs[0].result
	result.Constraint = runs[1].result
	result.NoCredit = runs[2].result
	result.WrongContext = runs[3].result
	result.Reset = runs[4].result
	result.Scalar = runs[5].result
	return result, nil
}

func rankPlans(base []int, plans []plan, key func(plan) []int, descending bool) ranking {
	keys := make(map[int][]int, len(plans))
	events := DiagnosticEvents{}
	for index := range plans {
		keys[index] = key(plans[index])
		events.ScoreKeys++
	}
	var order []int
	comparisons := 0
	for length := 1; length <= 3; length++ {
		var stratum []int
		for _, index := range base {
			if len(plans[index].edits) == length {
				stratum = append(stratum, index)
			}
		}
		sorted, count := stableMerge(stratum, func(left, right int) bool {
			cmp := compareInts(keys[left], keys[right])
			if descending {
				return cmp > 0
			}
			return cmp < 0
		})
		order = append(order, sorted...)
		comparisons += count
	}
	events.Comparisons = comparisons
	return ranking{order: order, orderHash: orderDigest(order, plans), events: events}
}

func resetCreditProfile(plans []plan) creditProfile {
	profile := emptyProfile()
	for _, value := range plans {
		decision, _ := credit.StructuralDecisionKey(synthesisMethod, value.features)
		profile.decision[decision] = 0
		for index, feature := range value.features {
			profile.component[feature] = 0
			profile.position[feature+fmt.Sprintf("|step-%d", index+1)] = 0
			profile.scalar[feature] = 500
			profile.relation[value.relations[index]] = 0
		}
	}
	return profile
}

func orderDigest(order []int, plans []plan) string {
	hash := sha256.New()
	for _, index := range order {
		hash.Write([]byte(plans[index].key + "\n"))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func stableMerge(values []int, less func(int, int) bool) ([]int, int) {
	if len(values) < 2 {
		return append([]int(nil), values...), 0
	}
	mid := len(values) / 2
	left, lc := stableMerge(values[:mid], less)
	right, rc := stableMerge(values[mid:], less)
	out := make([]int, 0, len(values))
	comparisons := lc + rc
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		comparisons++
		if less(right[j], left[i]) {
			out = append(out, right[j])
			j++
		} else {
			out = append(out, left[i])
			i++
		}
	}
	out = append(out, left[i:]...)
	out = append(out, right[j:]...)
	return out, comparisons
}

func groupByLength(base []int, plans []plan) []int {
	out := make([]int, 0, len(base))
	for length := 1; length <= 3; length++ {
		for _, index := range base {
			if len(plans[index].edits) == length {
				out = append(out, index)
			}
		}
	}
	return out
}

type policyRun struct {
	result    PolicyResult
	attempted []int
}

func searchProduction(c kuberepairfixture.Case, plans []plan, ranked ranking, handle string, common DiagnosticEvents) *policyRun {
	run := &policyRun{result: PolicyResult{SelectedLength: -1, OrderSHA256: ranked.orderHash, Diagnostics: common}}
	result := &run.result
	result.Diagnostics.ScoreKeys += ranked.events.ScoreKeys
	result.Diagnostics.Comparisons += ranked.events.Comparisons
	trace := sha256.New()
	trace.Write([]byte("empty\n"))
	if kuberepair.EqualOrSatisfies(c.Public, handle) {
		result.Solved = true
		result.SelectedLength = 0
		result.SelectedPlan = "empty"
		result.TerminalCalls = 1
		result.TraceSHA256 = hex.EncodeToString(trace.Sum(nil))
		return run
	}
	result.TerminalCalls = 1
	for _, index := range ranked.order {
		run.attempted = append(run.attempted, index)
		trace.Write([]byte(plans[index].key + "\n"))
		result.TerminalCalls++
		if kuberepair.EqualOrSatisfies(plans[index].result, handle) {
			result.Solved = true
			result.SelectedLength = len(plans[index].edits)
			result.SelectedPlan = plans[index].key
			break
		}
	}
	if !result.Solved && len(run.attempted) == len(ranked.order) {
		result.Solved = true
		result.SelectedPlan = "no-solution"
	}
	result.TraceSHA256 = hex.EncodeToString(trace.Sum(nil))
	return run
}

func auditPolicy(run *policyRun, c kuberepairfixture.Case, plans []plan, analysis kuberepairoracle.Analysis) {
	result := &run.result
	independent, err := kuberepairoracle.Satisfies(c.Public, oracleIntent(c))
	result.Diagnostics.OracleAudits++
	if err != nil || independent != (result.SelectedPlan == "empty") {
		result.OracleDisagreement = true
	}
	for _, index := range run.attempted {
		independent, err = kuberepairoracle.Satisfies(plans[index].result, oracleIntent(c))
		result.Diagnostics.OracleAudits++
		selected := plans[index].key == result.SelectedPlan
		if err != nil || independent != selected {
			result.OracleDisagreement = true
		}
		if selected && !independent {
			result.UnsafeAccepted = true
		}
	}
	minimum := map[string]bool{}
	for _, sequence := range analysis.Result.Plans {
		minimum[sequenceKey(sequence)] = true
	}
	switch result.SelectedPlan {
	case "empty":
		if analysis.Result.Terminal != "already-correct" {
			result.OracleDisagreement = true
		}
	case "no-solution":
		if analysis.Result.Terminal != "no-solution" {
			result.OracleDisagreement = true
		}
	default:
		if result.SelectedLength != analysis.Result.MinimumLength || !minimum[result.SelectedPlan] {
			result.OracleDisagreement = true
		}
	}
}

func contextScore(value plan, p creditProfile) []int {
	decision, _ := credit.StructuralDecisionKey(synthesisMethod, value.features)
	score := []int{p.decision[decision], 0, 0, 0}
	for i, feature := range value.features {
		score[1] += p.component[feature]
		score[2] += p.position[feature+fmt.Sprintf("|step-%d", i+1)]
		score[3] += p.relation[value.relations[i]]
	}
	return score
}
func scalarScore(value plan, p creditProfile) int {
	total := 0
	for _, feature := range value.features {
		worth := p.scalar[feature]
		if worth == 0 {
			worth = 500
		}
		total += worth
	}
	return total
}
func negateForAscending(values []int) []int {
	out := make([]int, len(values))
	for i, v := range values {
		out[i] = -v
	}
	return out
}
func compareInts(left, right []int) int {
	for i := range left {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return 0
}

func validateCohort(task trialCase, plans []plan, analysis kuberepairoracle.Analysis, profile creditProfile) error {
	if len(task.c.Edits) < 1 || len(task.c.Edits) > 8 {
		return fmt.Errorf("edit count %d outside 1..8", len(task.c.Edits))
	}
	result := analysis.Result
	switch task.cohort {
	case "exact":
		if result.Terminal != "solution" || result.MinimumLength != 1 {
			return fmt.Errorf("exact cohort drift")
		}
		for _, sequence := range result.Plans {
			feature, _, _ := kuberepair.FeatureKey(task.c.Edits[sequence[0]])
			decision, _ := credit.StructuralDecisionKey(synthesisMethod, []string{feature})
			if profile.component[feature] == 0 || profile.decision[decision] == 0 {
				return fmt.Errorf("exact minimum lacks trained credit")
			}
		}
	case "component":
		want := 2
		if task.stratum == "three-feature-recombined" {
			want = 3
		}
		if result.Terminal != "solution" || result.MinimumLength != want {
			return fmt.Errorf("component cohort drift")
		}
		for _, sequence := range result.Plans {
			seen := map[string]bool{}
			features := make([]string, len(sequence))
			for i, index := range sequence {
				features[i], _, _ = kuberepair.FeatureKey(task.c.Edits[index])
				if profile.component[features[i]] == 0 {
					return fmt.Errorf("uncredited minimum component")
				}
				seen[features[i]] = true
			}
			if len(seen) != want {
				return fmt.Errorf("component features are not distinct")
			}
			decision, _ := credit.StructuralDecisionKey(synthesisMethod, features)
			if profile.decision[decision] != 0 {
				return fmt.Errorf("minimum sequence has exact decision credit")
			}
		}
		minimum := map[string]bool{}
		for _, sequence := range result.Plans {
			minimum[sequenceKey(sequence)] = true
		}
		decoy := false
		for _, candidate := range plans {
			if len(candidate.edits) != result.MinimumLength || minimum[candidate.key] {
				continue
			}
			allCredited := true
			for _, feature := range candidate.features {
				allCredited = allCredited && profile.component[feature] != 0
			}
			if allCredited {
				decoy = true
				break
			}
		}
		if !decoy {
			return fmt.Errorf("component cohort lacks misleading credited decoy")
		}
	case "cross-role":
		if result.Terminal != "solution" || result.MinimumLength != 1 || len(result.Plans) < 2 {
			return fmt.Errorf("cross-role drift")
		}
		transferred := false
		for _, sequence := range result.Plans {
			f, r, _ := kuberepair.FeatureKey(task.c.Edits[sequence[0]])
			transferred = transferred || profile.component[f] == 0 && profile.relation[r] != 0
		}
		if !transferred {
			return fmt.Errorf("cross-role cohort lacks an uncredited component with credited relation")
		}
	case "co-minimal":
		if result.Terminal != "solution" || result.MinimumLength != 1 || len(result.Plans) < 2 {
			return fmt.Errorf("co-minimal drift")
		}
	case "unrelated":
		if result.Terminal != "solution" {
			return fmt.Errorf("unrelated drift")
		}
		for _, sequence := range result.Plans {
			for _, index := range sequence {
				f, r, _ := kuberepair.FeatureKey(task.c.Edits[index])
				if profile.component[f] != 0 || profile.relation[r] != 0 {
					return fmt.Errorf("unrelated minimum is credited")
				}
			}
		}
	case "already-correct":
		if result.Terminal != "already-correct" {
			return fmt.Errorf("already-correct drift")
		}
	case "no-solution":
		if result.Terminal != "no-solution" {
			return fmt.Errorf("no-solution drift")
		}
	default:
		return fmt.Errorf("unknown cohort %q", task.cohort)
	}
	return nil
}

func integrity(report Report, profile creditProfile) bool {
	if !report.PhaseA.IntegrityValid || report.Training.Tasks != 3 || len(report.Training.CreditedFeatures) != 3 {
		return false
	}
	for _, task := range report.Tasks {
		for _, result := range []PolicyResult{task.Contextual, task.Constraint, task.NoCredit, task.WrongContext, task.Reset, task.Scalar} {
			if result.UnsafeAccepted || result.OracleDisagreement || !result.Solved || result.TerminalCalls > 401 {
				return false
			}
		}
		if task.WrongContext.OrderSHA256 != task.NoCredit.OrderSHA256 || task.Reset.OrderSHA256 != task.NoCredit.OrderSHA256 || task.WrongContext.TraceSHA256 != task.NoCredit.TraceSHA256 || task.WrongContext.TerminalCalls != task.NoCredit.TerminalCalls || task.Reset.TraceSHA256 != task.NoCredit.TraceSHA256 || task.Reset.TerminalCalls != task.NoCredit.TerminalCalls {
			return false
		}
	}
	_ = profile
	return true
}

func aggregate(tasks []TaskResult, cohort, stratum string, trainingCalls int, root, comparator string) Aggregate {
	var selected []TaskResult
	for _, task := range tasks {
		if task.Cohort == cohort && (stratum == "" || task.Stratum == stratum) {
			selected = append(selected, task)
		}
	}
	result := Aggregate{Tasks: len(selected)}
	if len(selected) == 0 {
		return result
	}
	ctx := make([]float64, len(selected))
	constraint := make([]float64, len(selected))
	noCredit := make([]float64, len(selected))
	share := float64(trainingCalls) / 32
	for i, task := range selected {
		ctx[i] = float64(loss(task.Contextual)) + share
		constraint[i] = float64(loss(task.Constraint))
		noCredit[i] = float64(loss(task.NoCredit))
	}
	result.MeanContextualLoss = mean(ctx)
	result.MeanConstraintLoss = mean(constraint)
	result.MeanNoCreditLoss = mean(noCredit)
	result.ContextualVsConstraint = relative(result.MeanContextualLoss, result.MeanConstraintLoss)
	result.ContextualVsNoCredit = relative(result.MeanContextualLoss, result.MeanNoCreditLoss)
	result.BootstrapConstraintLow, result.BootstrapConstraintHigh = bootstrapAt(ctx, constraint, root, "inference/"+comparator+"-constraint", 0, bootstrapSamples)
	result.BootstrapNoCreditLow, result.BootstrapNoCreditHigh = bootstrapAt(ctx, noCredit, root, "inference/"+comparator+"-no-credit", 0, bootstrapSamples)
	return result
}
func loss(result PolicyResult) int {
	if result.Solved {
		return result.TerminalCalls
	}
	return failureLoss
}
func mean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
func relative(left, right float64) float64 {
	if right == 0 {
		return 0
	}
	return (left - right) / right
}

func bootstrapAt(left, right []float64, root, domain string, ordinal, replicates int) (float64, float64) {
	if len(left) == 0 {
		return 0, 0
	}
	rng := streamRand(domain, root, ordinal)
	values := make([]float64, replicates)
	for r := range values {
		l, rh := 0.0, 0.0
		for range left {
			i := rng.IntN(len(left))
			l += left[i]
			rh += right[i]
		}
		values[r] = relative(l/float64(len(left)), rh/float64(len(left)))
	}
	sort.Float64s(values)
	low := replicates/40 - 1
	high := replicates - replicates/40 - 1
	if low < 0 {
		low = 0
	}
	return values[low], values[high]
}

func classify(report Report) string {
	if !report.IntegrityValid {
		return "invalid"
	}
	if !report.PhaseA.Positive {
		return "valid-null"
	}
	if report.TwoFeature.Tasks == 0 || report.ThreeFeature.Tasks == 0 {
		return "valid-null"
	}
	unrelated := report.Cohorts["unrelated"]
	negativeTransfer := unrelated.Tasks == 0 || relativeWithoutTraining(report.Tasks, "unrelated") <= .10
	if report.Component.ContextualVsConstraint <= -.15 && report.Component.BootstrapConstraintHigh < 0 && report.TwoFeature.ContextualVsNoCredit <= -.10 && report.TwoFeature.BootstrapNoCreditHigh < 0 && report.ThreeFeature.ContextualVsNoCredit <= -.10 && report.ThreeFeature.BootstrapNoCreditHigh < 0 && negativeTransfer {
		return "valid-positive"
	}
	return "valid-null"
}

func relativeWithoutTraining(tasks []TaskResult, cohort string) float64 {
	var left, right []float64
	for _, task := range tasks {
		if task.Cohort == cohort {
			left = append(left, float64(loss(task.Contextual)))
			right = append(right, float64(loss(task.NoCredit)))
		}
	}
	if len(left) == 0 {
		return 0
	}
	return relative(mean(left), mean(right))
}

func estimatePower(tasks []TaskResult, trainingCalls int, root string) PowerReport {
	report := PowerReport{OuterReplicates: 2000, InnerReplicates: 2000}
	var two, three []TaskResult
	for _, task := range tasks {
		if task.Stratum == "two-feature" {
			two = append(two, task)
		}
		if task.Stratum == "three-feature-recombined" {
			three = append(three, task)
		}
	}
	if len(two) == 0 || len(three) == 0 {
		return report
	}
	outer := streamRand("power-outer", root, 0)
	for simulation := 0; simulation < report.OuterReplicates; simulation++ {
		sampleTwo := sampleTasks(two, 16, outer)
		sampleThree := sampleTasks(three, 16, outer)
		pooled := append(append([]TaskResult(nil), sampleTwo...), sampleThree...)
		if powerGate(pooled, sampleTwo, sampleThree, trainingCalls, root, simulation, report.InnerReplicates) {
			report.Passing++
		}
	}
	report.Power = float64(report.Passing) / float64(report.OuterReplicates)
	report.Accepted = report.Power >= .80
	return report
}
func sampleTasks(values []TaskResult, count int, rng *rand.Rand) []TaskResult {
	out := make([]TaskResult, count)
	for i := range out {
		out[i] = values[rng.IntN(len(values))]
	}
	return out
}
func powerGate(pooled, two, three []TaskResult, trainingCalls int, root string, simulation, replicates int) bool {
	check := func(tasks []TaskResult, which string, constraint bool, threshold float64) bool {
		share := float64(trainingCalls) / 32
		left := make([]float64, len(tasks))
		right := make([]float64, len(tasks))
		for i, t := range tasks {
			left[i] = float64(loss(t.Contextual)) + share
			if constraint {
				right[i] = float64(loss(t.Constraint))
			} else {
				right[i] = float64(loss(t.NoCredit))
			}
		}
		point := relative(mean(left), mean(right))
		_, high := bootstrapAt(left, right, root, "power-inner/"+which, simulation, replicates)
		return point <= threshold && high < 0
	}
	return check(pooled, "constraint", true, -.15) && check(two, "two", false, -.10) && check(three, "three", false, -.10)
}

func oracleIntent(c kuberepairfixture.Case) kuberepairoracle.Intent {
	return kuberepairoracle.Intent{DesiredPods: c.Intent.DesiredPods, BackendPort: c.Intent.BackendPort, ReadinessPorts: c.Intent.ReadinessPorts, ProtectedDigest: c.Intent.ProtectedDigest}
}
func semanticState(encoded string) string {
	bundle, err := kuberepair.DecodeBundle(encoded)
	if err != nil {
		return ""
	}
	bundle.Writes = nil
	value, _ := kuberepair.EncodeBundle(bundle)
	return value
}
func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:1]
	for _, v := range values[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	return out
}
func sequenceKey(values []int) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%03d", v)
	}
	return strings.Join(parts, "/")
}
func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
func rootHex(value int64) string { return fmt.Sprintf("%064x", value) }
func withOpaqueHandle(c kuberepairfixture.Case, root string, ordinal int) kuberepairfixture.Case {
	encoded, _ := json.Marshal([]any{"kuberepair-v2-stream/v1", "intent-handle", root, ordinal})
	digest := sha256.Sum256(encoded)
	c.Handle = hex.EncodeToString(digest[:])
	return c
}
func streamRand(domain, root string, ordinal int) *rand.Rand {
	encoded, _ := json.Marshal([]any{"kuberepair-v2-stream/v1", domain, root, ordinal})
	digest := sha256.Sum256(encoded)
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[0:8]), binary.BigEndian.Uint64(digest[8:16])))
}
func streamInt64(domain, root string, ordinal int) int64 {
	return int64(streamRand(domain, root, ordinal).Uint64() & 0x3fffffffffffffff)
}
