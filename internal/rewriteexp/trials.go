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
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
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
	MeanEvaluations float64
}

type CohortReport struct {
	Learned    StrategyReport
	Reset      StrategyReport
	Randomized StrategyReport
	Exhaustive StrategyReport
}

type CurriculumReport struct {
	Curricula int
	Budget    int
	Overall   CohortReport
	Cohorts   map[string]CohortReport
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
	rng := rand.New(rand.NewSource(seedValue))
	report := CurriculumReport{
		Curricula: count,
		Budget:    budget,
		Cohorts: map[string]CohortReport{
			"reuse-both": {},
			"reuse-one":  {},
			"unrelated":  {},
		},
	}
	cohortNames := []string{"reuse-both", "reuse-one", "unrelated"}
	for index := 0; index < count; index++ {
		cohort := cohortNames[index%len(cohortNames)]
		phaseOne, phaseTwo, err := generateCurriculum(rng, fmt.Sprintf("C%03d", index), cohort)
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

		learnedOrder := rankedPairs(phaseTwo, learnedWorth)
		resetOrder := rankedPairs(phaseTwo, resetWorth)
		randomOrder := append([]pair(nil), allPairs()...)
		rng.Shuffle(len(randomOrder), func(i, j int) { randomOrder[i], randomOrder[j] = randomOrder[j], randomOrder[i] })
		exhaustiveOrder := allPairs()

		cohortReport := report.Cohorts[cohort]
		updateStrategy(&cohortReport.Learned, solveRank(phaseTwo, learnedOrder), budget)
		updateStrategy(&cohortReport.Reset, solveRank(phaseTwo, resetOrder), budget)
		updateStrategy(&cohortReport.Randomized, solveRank(phaseTwo, randomOrder), budget)
		updateStrategy(&cohortReport.Exhaustive, solveRank(phaseTwo, exhaustiveOrder), 12)
		report.Cohorts[cohort] = cohortReport

		updateStrategy(&report.Overall.Learned, solveRank(phaseTwo, learnedOrder), budget)
		updateStrategy(&report.Overall.Reset, solveRank(phaseTwo, resetOrder), budget)
		updateStrategy(&report.Overall.Randomized, solveRank(phaseTwo, randomOrder), budget)
		updateStrategy(&report.Overall.Exhaustive, solveRank(phaseTwo, exhaustiveOrder), 12)
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

func solveRank(problem problem, order []pair) int {
	for index, candidate := range order {
		matched := true
		for _, example := range problem.Examples {
			actual, ok := applyPair(example.Input, problem.Rules, candidate)
			if !ok || actual != example.Expected {
				matched = false
				break
			}
		}
		if matched {
			return index + 1
		}
	}
	return len(order) + 1
}

func updateStrategy(report *StrategyReport, rank, budget int) {
	report.Tasks++
	if rank <= budget {
		report.Solved++
		report.MeanEvaluations += float64(rank)
	} else {
		report.MeanEvaluations += float64(budget)
	}
}

func finalizeStrategy(report *StrategyReport) {
	if report.Tasks != 0 {
		report.MeanEvaluations /= float64(report.Tasks)
	}
}

func finalizeCohort(report *CohortReport) {
	finalizeStrategy(&report.Learned)
	finalizeStrategy(&report.Reset)
	finalizeStrategy(&report.Randomized)
	finalizeStrategy(&report.Exhaustive)
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
