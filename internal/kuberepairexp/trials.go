// Package kuberepairexp runs the preregistered bounded ordering experiment.
package kuberepairexp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
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
)

const (
	workCap             = 393216
	exhaustedLoss       = workCap + 1
	terminalAttemptWork = 304
	synthesisMethod     = "ordered-bound-atomic-edits-up-to-3/v1"
)

type PolicyResult struct {
	Solved         bool `json:"solved"`
	Work           int  `json:"work"`
	Attempts       int  `json:"attempts"`
	UnsafeAccepted bool `json:"unsafe_accepted"`
}

type TaskResult struct {
	ID            string       `json:"id"`
	Cohort        string       `json:"cohort"`
	Edits         int          `json:"edits"`
	Plans         int          `json:"plans"`
	MinimumLength int          `json:"minimum_length"`
	MinimumPlans  int          `json:"minimum_plans"`
	Contextual    PolicyResult `json:"contextual"`
	Constraint    PolicyResult `json:"constraint"`
	NoCredit      PolicyResult `json:"no_credit"`
	WrongContext  PolicyResult `json:"wrong_context"`
	Reset         PolicyResult `json:"reset"`
	Scalar        PolicyResult `json:"scalar"`
}

type Aggregate struct {
	Tasks                   int     `json:"tasks"`
	ContextualSolved        int     `json:"contextual_solved"`
	ConstraintSolved        int     `json:"constraint_solved"`
	NoCreditSolved          int     `json:"no_credit_solved"`
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
	Tasks            int      `json:"tasks"`
	CandidatePlans   int      `json:"candidate_plans"`
	Work             int      `json:"work"`
	CreditedFeatures []string `json:"credited_features"`
}

type Report struct {
	Version        string               `json:"version"`
	Panel          string               `json:"panel"`
	Seed           int64                `json:"seed"`
	Outcome        string               `json:"outcome"`
	IntegrityValid bool                 `json:"integrity_valid"`
	Training       TrainingReport       `json:"training"`
	Component      Aggregate            `json:"component_primary"`
	Cohorts        map[string]Aggregate `json:"cohorts"`
	Tasks          []TaskResult         `json:"tasks"`
	Limitations    []string             `json:"limitations"`
}

func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

type creditProfile struct {
	decision  map[string]int
	component map[string]int
	position  map[string]int
	relation  map[string]int
	scalar    map[string]int
}

type plan struct {
	edits     []int
	result    string
	features  []string
	relations []string
	key       string
}

func Run(domainsDir, panel string) (Report, error) {
	seedValue, count, err := panelSpec(panel)
	if err != nil {
		return Report{}, err
	}
	profile, training, err := train(domainsDir, seedValue-2000)
	if err != nil {
		return Report{}, fmt.Errorf("training: %w", err)
	}
	tasks, err := panelCases(seedValue, count)
	if err != nil {
		return Report{}, err
	}
	report := Report{Version: "kubernetes-selector-reference-trials/v1", Panel: panel, Seed: seedValue, IntegrityValid: true, Training: training, Cohorts: map[string]Aggregate{}, Limitations: []string{"Synthetic bounded objects and atomic edits do not establish production Kubernetes repair.", "Phase B measures finite candidate ordering, not agenda-level pruning or scalable search.", "The human-supplied edit grammar is not repair-rule invention."}}
	for index, task := range tasks {
		result, runErr := runTask(task, profile, seedValue, int64(index))
		if runErr != nil {
			return report, fmt.Errorf("task %s: %w", task.c.ID, runErr)
		}
		report.Tasks = append(report.Tasks, result)
	}
	for _, cohort := range []string{"component", "exact", "cross-role", "unrelated", "co-minimal", "already-correct", "no-solution"} {
		report.Cohorts[cohort] = aggregate(report.Tasks, cohort, 0, seedValue)
	}
	report.Component = aggregate(report.Tasks, "component", training.Work/32, seedValue^0x9e3779b9)
	report.IntegrityValid = integrity(report)
	report.Outcome = classify(report)
	return report, nil
}

func panelSpec(panel string) (int64, int, error) {
	switch panel {
	case "development":
		return 741001, 24, nil
	case "validation":
		return 742001, 48, nil
	case "locked":
		return 743001, 96, nil
	default:
		return 0, 0, fmt.Errorf("unknown panel %q", panel)
	}
}

type trialCase struct {
	c      kuberepairfixture.Case
	cohort string
}

func panelCases(seedValue int64, count int) ([]trialCase, error) {
	var out []trialCase
	add := func(c kuberepairfixture.Case, cohort string) { out = append(out, trialCase{c: c, cohort: cohort}) }
	if count == 96 {
		for i := 0; i < 32; i++ {
			training, err := kuberepairfixture.Training(seedValue + int64(i*3))
			if err != nil {
				return nil, err
			}
			add(training[i%3], "exact")
		}
		masks := []int{3, 5, 6, 7}
		for i := 0; i < 32; i++ {
			c, err := kuberepairfixture.Recomposition(seedValue+100+int64(i), masks[i%4])
			if err != nil {
				return nil, err
			}
			add(c, "component")
		}
		for i := 0; i < 8; i++ {
			c, err := kuberepairfixture.CrossRole(seedValue + 200 + int64(i))
			if err != nil {
				return nil, err
			}
			add(c, "cross-role")
		}
		for i := 0; i < 8; i++ {
			c, err := kuberepairfixture.Unrelated(seedValue+300+int64(i), false)
			if err != nil {
				return nil, err
			}
			add(c, "unrelated")
		}
		for i := 0; i < 4; i++ {
			c, err := kuberepairfixture.CrossRole(seedValue + 400 + int64(i))
			if err != nil {
				return nil, err
			}
			add(c, "co-minimal")
		}
		for i := 0; i < 4; i++ {
			c, err := kuberepairfixture.Recomposition(seedValue+500+int64(i), 0)
			if err != nil {
				return nil, err
			}
			add(c, "already-correct")
		}
		for i := 0; i < 8; i++ {
			c, err := kuberepairfixture.Unrelated(seedValue+600+int64(i), true)
			if err != nil {
				return nil, err
			}
			add(c, "no-solution")
		}
		return out, nil
	}
	// Development and validation preserve the primary/control proportions.
	for i := 0; i < count; i++ {
		var c kuberepairfixture.Case
		var err error
		cohort := "component"
		switch i % 6 {
		case 0:
			c, err = kuberepairfixture.Recomposition(seedValue+int64(i), 3)
		case 1:
			c, err = kuberepairfixture.Recomposition(seedValue+int64(i), 5)
		case 2:
			c, err = kuberepairfixture.Recomposition(seedValue+int64(i), 6)
		case 3:
			c, err = kuberepairfixture.Recomposition(seedValue+int64(i), 7)
		case 4:
			c, err = kuberepairfixture.CrossRole(seedValue + int64(i))
			cohort = "cross-role"
		case 5:
			c, err = kuberepairfixture.Unrelated(seedValue+int64(i), false)
			cohort = "unrelated"
		}
		if err != nil {
			return nil, err
		}
		add(c, cohort)
	}
	return out, nil
}

func train(domainsDir string, seedValue int64) (creditProfile, TrainingReport, error) {
	profile := creditProfile{decision: map[string]int{}, component: map[string]int{}, position: map[string]int{}, relation: map[string]int{}, scalar: map[string]int{}}
	cases, err := kuberepairfixture.Training(seedValue)
	if err != nil {
		return profile, TrainingReport{}, err
	}
	report := TrainingReport{Tasks: len(cases)}
	for _, caseData := range cases {
		store, selected, candidates, cleanup, runErr := discover(domainsDir, caseData)
		if cleanup != nil {
			defer cleanup()
		}
		if runErr != nil {
			return profile, report, runErr
		}
		if len(selected) != 1 || store.Get(selected[0]).GetInt("programLength") != 1 {
			return profile, report, fmt.Errorf("training %s selected %d non-unique programs", caseData.ID, len(selected))
		}
		creditEngine := engine.New(store, agenda.New())
		creditEngine.Out = io.Discard
		creditEngine.VM.Out = io.Discard
		creditEngine.MutConfig.Enabled = false
		creditEngine.MaxCycles = 16
		if err := creditEngine.Run(context.Background()); err != nil {
			return profile, report, err
		}
		program := store.Get(selected[0])
		component := store.Get(program.GetStrings("components")[0])
		feature := component.GetString("creditFeatureKey")
		relation := component.GetString("creditRelationKey")
		subject := component.GetString("creditFeatureSubject")
		relationSubject := component.GetString("creditRelationSubject")
		decision, _ := credit.StructuralDecisionKey(synthesisMethod, []string{feature})
		profile.decision[decision] += credit.RewardTotal(store, credit.DecisionTuple(kuberepair.CreditContext, decision))
		profile.component[feature] += credit.RewardTotal(store, credit.Tuple{Context: kuberepair.CreditContext, Subject: subject, Role: "component"})
		profile.position[feature+"|step-1"] += credit.RewardTotal(store, credit.Tuple{Context: kuberepair.CreditContext, Subject: subject, Role: "step-1"})
		profile.relation[relation] += credit.RewardTotal(store, credit.Tuple{Context: kuberepair.CreditContext, Subject: relationSubject, Role: "relation"})
		profile.scalar[feature] = store.Get(subject).Worth()
		report.CandidatePlans += candidates
		report.Work += commonWork(len(caseData.Edits), candidates) + candidates*terminalAttemptWork
	}
	for feature := range profile.component {
		report.CreditedFeatures = append(report.CreditedFeatures, feature)
	}
	sort.Strings(report.CreditedFeatures)
	return profile, report, nil
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
	featureNames := map[string]string{}
	relationNames := map[string]string{}
	for index, encoded := range c.Edits {
		feature, relation, _ := kuberepair.FeatureKey(encoded)
		featureName := stableUnitName("KubeDynamicFeature", feature)
		relationName := stableUnitName("KubeDynamicRelation", relation)
		featureNames[feature] = featureName
		relationNames[relation] = relationName
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
		sum := sha256.Sum256([]byte(encoded))
		u.Set("semanticOpcode", "bound-"+hex.EncodeToString(sum[:8]))
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
	experiment.Set("experimentKey", "dynamic/"+c.ID)
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
func enumeratePlans(c kuberepairfixture.Case) ([]plan, error) {
	var out []plan
	var walk func(string, []int, int)
	walk = func(state string, sequence []int, remaining int) {
		if remaining == 0 {
			return
		}
		for index, edit := range c.Edits {
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
			walk(next, indices, remaining-1)
		}
	}
	walk(c.Public, nil, 3)
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out, nil
}

func runTask(task trialCase, profile creditProfile, panelSeed, index int64) (TaskResult, error) {
	plans, err := enumeratePlans(task.c)
	if err != nil {
		return TaskResult{}, err
	}
	oracle, err := kuberepairoracle.Solve(task.c.Public, task.c.Edits, kuberepairoracle.Intent{DesiredPods: task.c.Intent.DesiredPods, BackendPort: task.c.Intent.BackendPort, ReadinessPorts: task.c.Intent.ReadinessPorts, ProtectedDigest: task.c.Intent.ProtectedDigest}, 3)
	if err != nil {
		return TaskResult{}, err
	}
	base := make([]int, len(plans))
	for i := range base {
		base[i] = i
	}
	rng := rand.New(rand.NewSource((panelSeed + index) ^ 0x51f15e))
	rng.Shuffle(len(base), func(i, j int) { base[i], base[j] = base[j], base[i] })
	contextual := append([]int(nil), base...)
	sort.SliceStable(contextual, func(i, j int) bool { return contextualLess(plans[contextual[i]], plans[contextual[j]], profile) })
	scalar := append([]int(nil), base...)
	sort.SliceStable(scalar, func(i, j int) bool {
		return scalarScore(plans[scalar[i]], profile) > scalarScore(plans[scalar[j]], profile)
	})
	constraint := append([]int(nil), base...)
	vectors := make(map[int][]int, len(plans))
	for i, p := range plans {
		v, _ := kuberepair.PublicViolationVector(p.result)
		v = append(v, len(p.edits))
		vectors[i] = v
	}
	sort.SliceStable(constraint, func(i, j int) bool { return lexLess(vectors[constraint[i]], vectors[constraint[j]]) })
	common := commonWork(len(task.c.Edits), len(plans))
	contextRank := contextualRankWork(len(plans))
	constraintRank := constraintRankWork(len(plans))
	cleanup, err := kuberepair.RegisterIntent(task.c.Handle, task.c.Intent)
	if err != nil {
		return TaskResult{}, err
	}
	defer cleanup()
	handle, _ := kuberepair.EncodeHandle(task.c.Handle)
	result := TaskResult{ID: task.c.ID, Cohort: task.cohort, Edits: len(task.c.Edits), Plans: len(plans), MinimumLength: oracle.MinimumLength, MinimumPlans: len(oracle.Plans)}
	result.Contextual = search(plans, contextual, oracle, handle, common+contextRank)
	result.Constraint = search(plans, constraint, oracle, handle, common+constraintRank)
	result.NoCredit = search(plans, base, oracle, handle, common)
	result.WrongContext = search(plans, base, oracle, handle, common)
	result.Reset = search(plans, base, oracle, handle, common)
	result.Scalar = search(plans, scalar, oracle, handle, common+sortWork(len(plans), 2))
	return result, nil
}

func search(plans []plan, order []int, oracle kuberepairoracle.Result, handle string, initial int) PolicyResult {
	result := PolicyResult{Work: initial}
	if oracle.Terminal == "already-correct" {
		result.Solved = true
		return result
	}
	for _, index := range order {
		if result.Work+terminalAttemptWork > workCap {
			result.Work = exhaustedLoss
			return result
		}
		result.Work += terminalAttemptWork
		result.Attempts++
		matches := kuberepair.EqualOrSatisfies(plans[index].result, handle)
		if matches && len(plans[index].edits) == oracle.MinimumLength {
			result.Solved = true
			return result
		}
	}
	if oracle.Terminal == "no-solution" {
		result.Solved = true
	}
	return result
}
func contextualLess(a, b plan, p creditProfile) bool {
	as := contextScore(a, p)
	bs := contextScore(b, p)
	for i := range as {
		if as[i] != bs[i] {
			return as[i] > bs[i]
		}
	}
	return false
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
	score := 0
	for _, feature := range value.features {
		worth := p.scalar[feature]
		if worth == 0 {
			worth = 500
		}
		score += worth
	}
	return score
}
func lexLess(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func commonWork(edits, plans int) int { return 512 + edits*5 + plans + plans*3 + plans*3 + 583 }
func mergeComparisons(plans int) int {
	if plans < 2 {
		return 0
	}
	levels := 0
	for n := 1; n < plans; n *= 2 {
		levels++
	}
	return plans * levels
}
func sortWork(plans, tuple int) int    { return mergeComparisons(plans) * (1 + tuple) }
func contextualRankWork(plans int) int { return plans*20 + sortWork(plans, 5) }
func constraintRankWork(plans int) int { return plans*(96+64+7) + sortWork(plans, 7) }

func aggregate(tasks []TaskResult, cohort string, trainingShare int, seedValue int64) Aggregate {
	var selected []TaskResult
	for _, task := range tasks {
		if task.Cohort == cohort {
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
	for i, task := range selected {
		if task.Contextual.Solved {
			result.ContextualSolved++
		}
		if task.Constraint.Solved {
			result.ConstraintSolved++
		}
		if task.NoCredit.Solved {
			result.NoCreditSolved++
		}
		ctx[i] = float64(loss(task.Contextual) + trainingShare)
		constraint[i] = float64(loss(task.Constraint))
		noCredit[i] = float64(loss(task.NoCredit))
	}
	result.MeanContextualLoss = mean(ctx)
	result.MeanConstraintLoss = mean(constraint)
	result.MeanNoCreditLoss = mean(noCredit)
	result.ContextualVsConstraint = (result.MeanContextualLoss - result.MeanConstraintLoss) / result.MeanConstraintLoss
	result.ContextualVsNoCredit = (result.MeanContextualLoss - result.MeanNoCreditLoss) / result.MeanNoCreditLoss
	result.BootstrapConstraintLow, result.BootstrapConstraintHigh = bootstrap(ctx, constraint, seedValue)
	result.BootstrapNoCreditLow, result.BootstrapNoCreditHigh = bootstrap(ctx, noCredit, seedValue^0x13579)
	return result
}
func loss(result PolicyResult) int {
	if result.Solved {
		return result.Work
	}
	return exhaustedLoss
}
func mean(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}
func bootstrap(left, right []float64, seedValue int64) (float64, float64) {
	if len(left) == 0 {
		return 0, 0
	}
	rng := rand.New(rand.NewSource(seedValue))
	values := make([]float64, 10000)
	for r := range values {
		l, rh := 0.0, 0.0
		for range left {
			index := rng.Intn(len(left))
			l += left[index]
			rh += right[index]
		}
		l /= float64(len(left))
		rh /= float64(len(left))
		values[r] = (l - rh) / rh
	}
	sort.Float64s(values)
	return values[249], values[9749]
}
func integrity(report Report) bool {
	for _, task := range report.Tasks {
		if task.WrongContext != task.NoCredit || task.Reset != task.NoCredit || task.Contextual.UnsafeAccepted || task.Constraint.UnsafeAccepted || task.NoCredit.UnsafeAccepted {
			return false
		}
	}
	return report.Training.Tasks == 3 && len(report.Training.CreditedFeatures) == 3
}
func classify(report Report) string {
	if !report.IntegrityValid {
		return "invalid"
	}
	c := report.Component
	if c.Tasks == 0 {
		return "valid-null"
	}
	safety := float64(c.ContextualSolved) / float64(c.Tasks)
	if safety >= .9 && c.ContextualVsConstraint <= -.15 && c.BootstrapConstraintHigh < 0 && c.ContextualVsNoCredit <= -.10 && c.BootstrapNoCreditHigh < 0 {
		return "valid-positive"
	}
	return "valid-null"
}
