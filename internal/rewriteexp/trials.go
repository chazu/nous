// Package rewriteexp runs deterministic experiments over the rewrite
// vocabulary. Benchmark generation and ablations remain outside the kernel.
package rewriteexp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/credit"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	rewritevocab "github.com/chazu/nous/internal/vocab/rewrite"
)

const engineCycles = 220

type ruleSpec struct {
	Name  string
	Left  string
	Right string
}

type pair struct {
	First  int
	Second int
}

type example struct {
	Name     string
	Input    string
	Expected string
}

type problem struct {
	Rules    []ruleSpec
	Target   pair
	Examples []example
	HeldOut  []example
}

type RobustnessReport struct {
	Problems         int
	Recovered        int
	UniquePromotions int
	FalsePromotions  int
	HeldOutCases     int
	HeldOutFailures  int
}

type StrategyReport struct {
	Tasks           int
	Solved          int
	Evaluations     int
	MaxEvaluations  int
	MeanEvaluations float64
}

type CohortReport struct {
	Contextual     StrategyReport
	Scalar         StrategyReport
	ScalarReserved StrategyReport
	Reset          StrategyReport
	Randomized     StrategyReport
	Exhaustive     StrategyReport
}

type PairwiseReport struct {
	ContextualWins   int
	ContextualLosses int
	Ties             int
}

type IsolationReport struct {
	Checks               int
	WrongContextMatches  int
	AbsentContextMatches int
}

type CurriculumReport struct {
	Curricula int
	Budget    int
	Overall   CohortReport
	Cohorts   map[string]CohortReport
	Paired    map[string]PairwiseReport
	Isolation IsolationReport
}

type Report struct {
	Seed       int64
	Robustness RobustnessReport
	Curriculum CurriculumReport
}

func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

func Run(domainsDir string, seedValue int64, problems, curricula, budget int) (Report, error) {
	robustness, err := RunRobustness(domainsDir, seedValue, problems)
	if err != nil {
		return Report{}, err
	}
	curriculum, err := RunCurriculum(domainsDir, seedValue+1, curricula, budget)
	if err != nil {
		return Report{}, err
	}
	return Report{Seed: seedValue, Robustness: robustness, Curriculum: curriculum}, nil
}

func RunRobustness(domainsDir string, seedValue int64, count int) (RobustnessReport, error) {
	if count <= 0 {
		return RobustnessReport{}, fmt.Errorf("problem count must be positive")
	}
	rng := rand.New(rand.NewSource(seedValue))
	report := RobustnessReport{Problems: count}
	for index := 0; index < count; index++ {
		problem, err := generateProblem(rng, fmt.Sprintf("R%03d", index))
		if err != nil {
			return report, fmt.Errorf("generate robustness problem %d: %w", index, err)
		}
		store, eng, err := executeProblem(domainsDir, problem)
		if err != nil {
			return report, fmt.Errorf("execute robustness problem %d: %w", index, err)
		}
		promoted := promotedPrograms(store)
		if len(promoted) == 1 {
			report.UniquePromotions++
		}
		targetNames := []string{problem.Rules[problem.Target.First].Name, problem.Rules[problem.Target.Second].Name}
		matched := false
		matchedName := ""
		for _, name := range promoted {
			components := store.Get(name).GetStrings("components")
			if equalStrings(components, targetNames) {
				matched = true
				matchedName = name
			} else {
				report.FalsePromotions++
			}
		}
		if matched {
			report.Recovered++
		}
		for _, heldOut := range problem.HeldOut {
			report.HeldOutCases++
			if !matched {
				report.HeldOutFailures++
				continue
			}
			value, execErr := eng.VM.Execute(fmt.Sprintf("%q %q apply-op", heldOut.Input, matchedName))
			if execErr != nil || value.IsNil() || value.AsString() != heldOut.Expected {
				report.HeldOutFailures++
			}
		}
	}
	return report, nil
}

func RunCurriculum(domainsDir string, seedValue int64, count, budget int) (CurriculumReport, error) {
	if count <= 0 || budget <= 0 || budget > 12 {
		return CurriculumReport{}, fmt.Errorf("curricula must be positive and budget must be in [1,12]")
	}
	generationRNG := rand.New(rand.NewSource(seedValue))
	report := CurriculumReport{
		Curricula: count,
		Budget:    budget,
		Cohorts: map[string]CohortReport{
			"reuse-both": {},
			"reuse-one":  {},
			"unrelated":  {},
		},
		Paired: map[string]PairwiseReport{
			"scalar":          {},
			"scalar-reserved": {},
			"randomized":      {},
		},
	}
	cohortNames := []string{"reuse-both", "reuse-one", "unrelated"}
	for index := 0; index < count; index++ {
		cohort := cohortNames[index%len(cohortNames)]
		phaseOne, phaseTwo, err := generateCurriculum(generationRNG, fmt.Sprintf("C%03d", index), cohort)
		if err != nil {
			return report, fmt.Errorf("generate curriculum %d: %w", index, err)
		}
		store, _, err := executeProblem(domainsDir, phaseOne)
		if err != nil {
			return report, fmt.Errorf("execute curriculum %d phase one: %w", index, err)
		}
		learnedWorth := make(map[string]int, len(phaseOne.Rules))
		resetWorth := make(map[string]int, len(phaseOne.Rules))
		for _, rule := range phaseOne.Rules {
			learnedWorth[rule.Name] = store.Get(rule.Name).Worth()
			resetWorth[rule.Name] = 600
		}
		firstName := phaseOne.Rules[phaseOne.Target.First].Name
		secondName := phaseOne.Rules[phaseOne.Target.Second].Name
		if learnedWorth[firstName] != 750 || learnedWorth[secondName] != 750 {
			return report, fmt.Errorf("curriculum %d did not earn expected component credit", index)
		}
		decision := rewritevocab.DecisionKey(firstName, secondName)
		if got := credit.RewardTotal(store, credit.DecisionTuple(rewritevocab.CreditContext, decision)); got != 300 {
			return report, fmt.Errorf("curriculum %d decision credit = %d, want 300", index, got)
		}

		policyRNG := rand.New(rand.NewSource(curriculumPolicySeed(seedValue, index)))
		sharedOrder := append([]pair(nil), allPairs()...)
		policyRNG.Shuffle(len(sharedOrder), func(i, j int) { sharedOrder[i], sharedOrder[j] = sharedOrder[j], sharedOrder[i] })
		contextualOrder := contextualPairs(phaseTwo, store, rewritevocab.CreditContext, sharedOrder)
		scalarOrder := rankedPairs(phaseTwo, learnedWorth)
		scalarReservedOrder := scalarReservedPairs(phaseTwo, learnedWorth, sharedOrder)
		resetOrder := rankedPairs(phaseTwo, resetWorth)
		randomOrder := append([]pair(nil), sharedOrder...)
		exhaustiveOrder := allPairs()
		if !equalPairs(contextualPairs(phaseTwo, store, "wrong/context/v1", sharedOrder), sharedOrder) ||
			!equalPairs(contextualPairs(phaseTwo, store, "", sharedOrder), sharedOrder) {
			return report, fmt.Errorf("curriculum %d contextual isolation changed exploration order", index)
		}
		report.Isolation.Checks++
		report.Isolation.WrongContextMatches++
		report.Isolation.AbsentContextMatches++

		contextualSolved, contextualEvaluations := solveWithinBudget(phaseTwo, contextualOrder, budget)
		scalarSolved, scalarEvaluations := solveWithinBudget(phaseTwo, scalarOrder, budget)
		scalarReservedSolved, scalarReservedEvaluations := solveWithinBudget(phaseTwo, scalarReservedOrder, budget)
		resetSolved, resetEvaluations := solveWithinBudget(phaseTwo, resetOrder, budget)
		randomSolved, randomEvaluations := solveWithinBudget(phaseTwo, randomOrder, budget)
		exhaustiveSolved, exhaustiveEvaluations := solveWithinBudget(phaseTwo, exhaustiveOrder, len(exhaustiveOrder))

		cohortReport := report.Cohorts[cohort]
		updateStrategy(&cohortReport.Contextual, contextualSolved, contextualEvaluations)
		updateStrategy(&cohortReport.Scalar, scalarSolved, scalarEvaluations)
		updateStrategy(&cohortReport.ScalarReserved, scalarReservedSolved, scalarReservedEvaluations)
		updateStrategy(&cohortReport.Reset, resetSolved, resetEvaluations)
		updateStrategy(&cohortReport.Randomized, randomSolved, randomEvaluations)
		updateStrategy(&cohortReport.Exhaustive, exhaustiveSolved, exhaustiveEvaluations)
		report.Cohorts[cohort] = cohortReport

		updateStrategy(&report.Overall.Contextual, contextualSolved, contextualEvaluations)
		updateStrategy(&report.Overall.Scalar, scalarSolved, scalarEvaluations)
		updateStrategy(&report.Overall.ScalarReserved, scalarReservedSolved, scalarReservedEvaluations)
		updateStrategy(&report.Overall.Reset, resetSolved, resetEvaluations)
		updateStrategy(&report.Overall.Randomized, randomSolved, randomEvaluations)
		updateStrategy(&report.Overall.Exhaustive, exhaustiveSolved, exhaustiveEvaluations)
		updatePairwise(report.Paired, "scalar", contextualSolved, scalarSolved)
		updatePairwise(report.Paired, "scalar-reserved", contextualSolved, scalarReservedSolved)
		updatePairwise(report.Paired, "randomized", contextualSolved, randomSolved)
	}
	finalizeCohort(&report.Overall)
	for name, cohort := range report.Cohorts {
		finalizeCohort(&cohort)
		report.Cohorts[name] = cohort
	}
	return report, nil
}

func executeProblem(domainsDir string, problem problem) (*unit.Store, *engine.Engine, error) {
	previousDomainsDir := seed.DomainsDir
	seed.DomainsDir = domainsDir
	defer func() { seed.DomainsDir = previousDomainsDir }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "rewrite"); err != nil {
		return nil, nil, err
	}
	for _, name := range store.Examples("PrimitiveRewriteOp") {
		if name != "PrimitiveRewriteOp" && store.Get(name).Has("rewriteLeft") {
			store.Delete(name)
		}
	}
	for _, name := range store.Examples("RewriteTrainingExample") {
		if name != "RewriteTrainingExample" && store.Get(name).Has("input") {
			store.Delete(name)
		}
	}
	for _, spec := range problem.Rules {
		op := unit.New(spec.Name)
		op.SetWorth(600)
		op.Set("isA", []string{"PrimitiveRewriteOp", "UnaryOp", "Op", "Anything"})
		op.Set("domain", []string{"RewriteString"})
		op.Set("range", []string{"RewriteString"})
		op.Set("arity", 1)
		op.Set("rewriteLeft", spec.Left)
		op.Set("rewriteRight", spec.Right)
		op.Set("defn", fmt.Sprintf("%q %q rewrite-replace-all", spec.Left, spec.Right))
		store.Put(op)
	}
	for _, spec := range problem.Examples {
		example := unit.New(spec.Name)
		example.SetWorth(600)
		example.Set("isA", []string{"RewriteTrainingExample", "Anything"})
		example.Set("input", spec.Input)
		example.Set("expected", spec.Expected)
		store.Put(example)
	}
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = engineCycles
	eng.MutConfig.Enabled = false
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		return nil, nil, err
	}
	return store, eng, nil
}

func generateProblem(rng *rand.Rand, prefix string) (problem, error) {
	for attempt := 0; attempt < 500; attempt++ {
		rules := randomRules(rng, prefix)
		pairs := allPairs()
		target := pairs[rng.Intn(len(pairs))]
		examples, heldOut, ok := distinguishTarget(rules, target, prefix, rng)
		if ok {
			return problem{Rules: rules, Target: target, Examples: examples, HeldOut: heldOut}, nil
		}
	}
	return problem{}, fmt.Errorf("could not generate uniquely identifiable problem")
}

func generateCurriculum(rng *rand.Rand, prefix, cohort string) (problem, problem, error) {
	for attempt := 0; attempt < 500; attempt++ {
		phaseOne, err := generateProblem(rng, prefix+"A")
		if err != nil {
			return problem{}, problem{}, err
		}
		candidates := allPairs()
		rng.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
		for _, target := range candidates {
			shared := sharedComponents(phaseOne.Target, target)
			if (cohort == "reuse-both" && target != phaseOne.Target) ||
				(cohort == "reuse-one" && shared != 1) ||
				(cohort == "unrelated" && shared != 0) {
				continue
			}
			examples, heldOut, ok := distinguishTarget(phaseOne.Rules, target, prefix+"B", rng)
			if ok {
				phaseTwo := problem{Rules: phaseOne.Rules, Target: target, Examples: examples, HeldOut: heldOut}
				return phaseOne, phaseTwo, nil
			}
		}
	}
	return problem{}, problem{}, fmt.Errorf("could not generate %s curriculum", cohort)
}

func randomRules(rng *rand.Rand, prefix string) []ruleSpec {
	const alphabet = "abcdef"
	rules := make([]ruleSpec, 0, 4)
	seen := map[string]bool{}
	for len(rules) < 4 {
		left := randomText(rng, alphabet, 1+rng.Intn(2))
		right := randomText(rng, alphabet, rng.Intn(3))
		key := left + "|" + right
		if left == right || seen[key] {
			continue
		}
		seen[key] = true
		rules = append(rules, ruleSpec{Name: fmt.Sprintf("%s-P%d", prefix, len(rules)), Left: left, Right: right})
	}
	return rules
}

func distinguishTarget(rules []ruleSpec, target pair, prefix string, rng *rand.Rand) ([]example, []example, bool) {
	probes := generatedProbes("abcdef", 4)
	rng.Shuffle(len(probes), func(i, j int) { probes[i], probes[j] = probes[j], probes[i] })
	remaining := map[pair]bool{}
	for _, candidate := range allPairs() {
		if candidate != target {
			remaining[candidate] = true
		}
	}
	var selected []string
	used := map[string]bool{}
	for len(remaining) > 0 && len(selected) < 8 {
		best := ""
		bestEliminated := 0
		for _, probe := range probes {
			if used[probe] {
				continue
			}
			want, ok := applyPair(probe, rules, target)
			if !ok {
				continue
			}
			eliminated := 0
			for candidate := range remaining {
				got, candidateOK := applyPair(probe, rules, candidate)
				if !candidateOK || got != want {
					eliminated++
				}
			}
			if eliminated > bestEliminated {
				best, bestEliminated = probe, eliminated
			}
		}
		if bestEliminated == 0 {
			return nil, nil, false
		}
		selected = append(selected, best)
		used[best] = true
		want, _ := applyPair(best, rules, target)
		for candidate := range remaining {
			got, ok := applyPair(best, rules, candidate)
			if !ok || got != want {
				delete(remaining, candidate)
			}
		}
	}
	if len(remaining) != 0 {
		return nil, nil, false
	}
	for _, probe := range probes {
		if len(selected) >= 4 {
			break
		}
		if !used[probe] {
			selected = append(selected, probe)
			used[probe] = true
		}
	}
	examples := make([]example, len(selected))
	for index, input := range selected {
		expected, _ := applyPair(input, rules, target)
		examples[index] = example{Name: fmt.Sprintf("%s-E%d", prefix, index), Input: input, Expected: expected}
	}
	var heldOut []example
	for _, probe := range probes {
		if used[probe] {
			continue
		}
		expected, ok := applyPair(probe, rules, target)
		if !ok {
			continue
		}
		heldOut = append(heldOut, example{Name: fmt.Sprintf("%s-H%d", prefix, len(heldOut)), Input: probe, Expected: expected})
		if len(heldOut) == 32 {
			break
		}
	}
	return examples, heldOut, len(heldOut) == 32
}

func rankedPairs(problem problem, worth map[string]int) []pair {
	pairs := allPairs()
	sort.Slice(pairs, func(i, j int) bool {
		a := worth[problem.Rules[pairs[i].First].Name] + worth[problem.Rules[pairs[i].Second].Name]
		b := worth[problem.Rules[pairs[j].First].Name] + worth[problem.Rules[pairs[j].Second].Name]
		if a != b {
			return a > b
		}
		return pairKey(problem, pairs[i]) < pairKey(problem, pairs[j])
	})
	return pairs
}

func contextualPairs(problem problem, store *unit.Store, contextName string, exploration []pair) []pair {
	bestIndex := -1
	bestScore := 0
	for index, candidate := range exploration {
		decision := rewritevocab.DecisionKey(
			problem.Rules[candidate.First].Name,
			problem.Rules[candidate.Second].Name,
		)
		score := credit.RewardTotal(store, credit.DecisionTuple(contextName, decision))
		if score > bestScore {
			bestIndex, bestScore = index, score
		}
	}
	if bestIndex < 0 {
		return append([]pair(nil), exploration...)
	}
	return movePairFirst(exploration, bestIndex)
}

func scalarReservedPairs(problem problem, worth map[string]int, exploration []pair) []pair {
	bestIndex := -1
	bestScore := -1
	for index, candidate := range exploration {
		score := worth[problem.Rules[candidate.First].Name] + worth[problem.Rules[candidate.Second].Name]
		if score > bestScore {
			bestIndex, bestScore = index, score
		}
	}
	if bestIndex < 0 {
		return append([]pair(nil), exploration...)
	}
	return movePairFirst(exploration, bestIndex)
}

func movePairFirst(order []pair, index int) []pair {
	out := make([]pair, 0, len(order))
	out = append(out, order[index])
	out = append(out, order[:index]...)
	out = append(out, order[index+1:]...)
	return out
}

func solveWithinBudget(problem problem, order []pair, budget int) (bool, int) {
	limit := budget
	if limit > len(order) {
		limit = len(order)
	}
	for index := 0; index < limit; index++ {
		candidate := order[index]
		matched := true
		for _, example := range problem.Examples {
			actual, ok := applyPair(example.Input, problem.Rules, candidate)
			if !ok || actual != example.Expected {
				matched = false
				break
			}
		}
		if matched {
			return true, index + 1
		}
	}
	return false, limit
}

func updateStrategy(report *StrategyReport, solved bool, evaluations int) {
	report.Tasks++
	if solved {
		report.Solved++
	}
	report.Evaluations += evaluations
	if evaluations > report.MaxEvaluations {
		report.MaxEvaluations = evaluations
	}
	report.MeanEvaluations += float64(evaluations)
}

func updatePairwise(reports map[string]PairwiseReport, name string, contextual, other bool) {
	report := reports[name]
	switch {
	case contextual && !other:
		report.ContextualWins++
	case !contextual && other:
		report.ContextualLosses++
	default:
		report.Ties++
	}
	reports[name] = report
}

func finalizeStrategy(report *StrategyReport) {
	if report.Tasks != 0 {
		report.MeanEvaluations /= float64(report.Tasks)
	}
}

func finalizeCohort(report *CohortReport) {
	finalizeStrategy(&report.Contextual)
	finalizeStrategy(&report.Scalar)
	finalizeStrategy(&report.ScalarReserved)
	finalizeStrategy(&report.Reset)
	finalizeStrategy(&report.Randomized)
	finalizeStrategy(&report.Exhaustive)
}

func equalPairs(a, b []pair) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func curriculumPolicySeed(seedValue int64, index int) int64 {
	value := uint64(seedValue) + uint64(index+1)*0x9e3779b97f4a7c15
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	return int64(value ^ (value >> 31))
}

func promotedPrograms(store *unit.Store) []string {
	var programs []string
	for _, name := range store.Examples("RewriteProgramSchema") {
		if name == "RewriteProgramSchema" {
			continue
		}
		program := store.Get(name).GetString("program")
		if program != "" {
			programs = append(programs, program)
		}
	}
	sort.Strings(programs)
	return programs
}

func allPairs() []pair {
	var pairs []pair
	for first := 0; first < 4; first++ {
		for second := 0; second < 4; second++ {
			if first != second {
				pairs = append(pairs, pair{First: first, Second: second})
			}
		}
	}
	return pairs
}

func pairKey(problem problem, candidate pair) string {
	return problem.Rules[candidate.First].Name + "|" + problem.Rules[candidate.Second].Name
}

func sharedComponents(a, b pair) int {
	seen := map[int]bool{a.First: true, a.Second: true}
	shared := 0
	if seen[b.First] {
		shared++
	}
	if seen[b.Second] {
		shared++
	}
	return shared
}

func applyPair(input string, rules []ruleSpec, candidate pair) (string, bool) {
	first, ok := referenceReplace(input, rules[candidate.First].Left, rules[candidate.First].Right)
	if !ok {
		return "", false
	}
	return referenceReplace(first, rules[candidate.Second].Left, rules[candidate.Second].Right)
}

func referenceReplace(input, left, right string) (string, bool) {
	valid := func(text string) bool {
		if len(text) > 256 {
			return false
		}
		for index := 0; index < len(text); index++ {
			if text[index] < 'a' || text[index] > 'z' {
				return false
			}
		}
		return true
	}
	if !valid(input) || left == "" || len(left) > 8 || len(right) > 8 || !valid(left) || !valid(right) {
		return "", false
	}
	var out strings.Builder
	for position := 0; position < len(input); {
		if strings.HasPrefix(input[position:], left) {
			out.WriteString(right)
			position += len(left)
		} else {
			out.WriteByte(input[position])
			position++
		}
		if out.Len() > 256 {
			return "", false
		}
	}
	return out.String(), true
}

func generatedProbes(alphabet string, maxLength int) []string {
	probes := []string{""}
	frontier := []string{""}
	for depth := 0; depth < maxLength; depth++ {
		var next []string
		for _, prefix := range frontier {
			for index := 0; index < len(alphabet); index++ {
				value := prefix + string(alphabet[index])
				probes = append(probes, value)
				next = append(next, value)
			}
		}
		frontier = next
	}
	return probes
}

func randomText(rng *rand.Rand, alphabet string, length int) string {
	var out strings.Builder
	for index := 0; index < length; index++ {
		out.WriteByte(alphabet[rng.Intn(len(alphabet))])
	}
	return out.String()
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}
