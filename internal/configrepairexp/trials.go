// Package configrepairexp runs deterministic reality-gate trials over the
// configuration-repair vocabulary. The fixtures are bounded fact-model
// translations of Kubernetes and Terraform incidents, not raw YAML or HCL.
package configrepairexp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/dsl"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	configvocab "github.com/chazu/nous/internal/vocab/configrepair"
)

const engineCycles = 700

type assignment struct {
	Key       string
	Bad       string
	Good      string
	Kind      string
	Minimum   int
	Required  bool
	Protected bool
}

type scenario struct {
	ID              string
	Technology      string
	Name            string
	Symptom         string
	ProtectedIntent string
	ExpectedRepair  string
	UnsafeShortcut  string
	Oracle          string
	Fidelity        string
	Assignments     []assignment
}

type repairSpec struct {
	Name   string
	Repair configvocab.Repair
	Unsafe bool
}

type exampleSpec struct {
	Name          string
	Configuration []string
}

// ScenarioReport records what the actual Nous heuristics and an independent
// direct enumeration did with one translated incident.
type ScenarioReport struct {
	ID                       string   `json:"id"`
	Technology               string   `json:"technology"`
	Name                     string   `json:"name"`
	Symptom                  string   `json:"symptom"`
	ProtectedIntent          string   `json:"protected_intent"`
	ExpectedRepair           string   `json:"expected_repair"`
	UnsafeShortcut           string   `json:"unsafe_shortcut"`
	SemanticOracleGoal       string   `json:"semantic_oracle_goal"`
	Fidelity                 string   `json:"fidelity"`
	CandidatePlans           int      `json:"candidate_plans"`
	PromotedPlans            int      `json:"promoted_plans"`
	ExpectedPlanRecovered    bool     `json:"expected_plan_recovered"`
	UniquePromotion          bool     `json:"unique_promotion"`
	UnsafeCandidates         int      `json:"unsafe_candidates"`
	UnsafeCandidatesRejected int      `json:"unsafe_candidates_rejected"`
	HeldOutCases             int      `json:"held_out_cases"`
	HeldOutFailures          int      `json:"held_out_failures"`
	BaselineSolutions        int      `json:"baseline_solutions"`
	NousBaselineAgreement    bool     `json:"nous_baseline_agreement"`
	ExpectedRepairs          []string `json:"expected_repairs"`
}

// Report summarizes the bounded trial without claiming raw manifest or HCL
// support. Limitations are part of the machine-readable result.
type Report struct {
	Scenarios                   int              `json:"scenarios"`
	Kubernetes                  int              `json:"kubernetes"`
	Terraform                   int              `json:"terraform"`
	NousExpectedPlansRecovered  int              `json:"nous_expected_plans_recovered"`
	NousUniquePromotions        int              `json:"nous_unique_promotions"`
	HeldOutCases                int              `json:"held_out_cases"`
	HeldOutFailures             int              `json:"held_out_failures"`
	UnsafeCandidates            int              `json:"unsafe_candidates"`
	UnsafeCandidatesRejected    int              `json:"unsafe_candidates_rejected"`
	BaselineExpectedPlans       int              `json:"baseline_expected_plans_recovered"`
	ExactNousBaselineAgreements int              `json:"exact_nous_baseline_agreements"`
	Results                     []ScenarioReport `json:"results"`
	Limitations                 []string         `json:"limitations"`
}

func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

// Run submits each scenario to a fresh Nous configuration-repair store, then
// tests the promoted program on held-out variants and compares it with direct
// bounded enumeration over the same repair catalog.
func Run(domainsDir string) (Report, error) {
	scenarios := scenarios()
	report := Report{Scenarios: len(scenarios), Limitations: limitations()}
	for _, problem := range scenarios {
		result, baselineRecovered, err := runScenario(domainsDir, problem)
		if err != nil {
			return report, fmt.Errorf("scenario %s: %w", problem.ID, err)
		}
		report.Results = append(report.Results, result)
		switch problem.Technology {
		case "kubernetes":
			report.Kubernetes++
		case "terraform":
			report.Terraform++
		}
		if result.ExpectedPlanRecovered {
			report.NousExpectedPlansRecovered++
		}
		if result.UniquePromotion {
			report.NousUniquePromotions++
		}
		if baselineRecovered {
			report.BaselineExpectedPlans++
		}
		if result.NousBaselineAgreement {
			report.ExactNousBaselineAgreements++
		}
		report.HeldOutCases += result.HeldOutCases
		report.HeldOutFailures += result.HeldOutFailures
		report.UnsafeCandidates += result.UnsafeCandidates
		report.UnsafeCandidatesRejected += result.UnsafeCandidatesRejected
	}
	return report, nil
}

func runScenario(domainsDir string, problem scenario) (ScenarioReport, bool, error) {
	store, repairs, training, heldOut, err := buildStore(domainsDir, problem)
	if err != nil {
		return ScenarioReport{}, false, err
	}
	ag := agenda.New()
	eng := engine.New(store, ag)
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MaxCycles = engineCycles
	eng.MutConfig.Enabled = false
	eng.SeedInitialAgenda()
	if err := eng.Run(context.Background()); err != nil {
		return ScenarioReport{}, false, fmt.Errorf("run Nous: %w", err)
	}
	if ag.Len() != 0 {
		return ScenarioReport{}, false, fmt.Errorf("agenda did not drain: %d tasks remain", ag.Len())
	}

	programs := categoryUnits(store, "CompositeConfigurationRepair", "components")
	promoted := categoryUnits(store, "ConfigurationRepairSchema", "program")
	promotedSets := make([]string, 0, len(promoted))
	promotedPrograms := make([]string, 0, len(promoted))
	for _, schemaName := range promoted {
		programName := store.Get(schemaName).GetString("program")
		promotedPrograms = append(promotedPrograms, programName)
		promotedSets = append(promotedSets, semanticPlan(store, programName))
	}
	sort.Strings(promotedSets)
	expected := expectedPlan(repairs)
	expectedKey := semanticRepairs(expected)
	expectedRecovered := contains(promotedSets, expectedKey)

	unsafeCandidates := 0
	unsafeRejected := 0
	unsafeNames := map[string]bool{}
	for _, repair := range repairs {
		if repair.Unsafe {
			unsafeNames[repair.Name] = true
		}
	}
	for _, programName := range programs {
		unsafe := false
		for _, component := range store.Get(programName).GetStrings("components") {
			unsafe = unsafe || unsafeNames[component]
		}
		if !unsafe {
			continue
		}
		unsafeCandidates++
		if !contains(promotedPrograms, programName) {
			unsafeRejected++
		}
	}

	heldOutFailures := 0
	matchedProgram := ""
	for _, schemaName := range promoted {
		programName := store.Get(schemaName).GetString("program")
		if semanticPlan(store, programName) == expectedKey {
			matchedProgram = programName
			break
		}
	}
	for _, example := range heldOut {
		if matchedProgram == "" {
			heldOutFailures++
			continue
		}
		value, execErr := eng.VM.Execute(dslList(example.Configuration) + " " + strconv.Quote(matchedProgram) + " apply-op")
		if execErr != nil || value.IsNil() {
			heldOutFailures++
			continue
		}
		actual, ok := dslStrings(value)
		if !ok || !successful(actual, example.Configuration, schemaData(problem)) {
			heldOutFailures++
		}
	}

	baselineSets := baselineSolutions(training, schemaData(problem), repairs)
	baselineRecovered := contains(baselineSets, expectedKey)
	result := ScenarioReport{
		ID:                       problem.ID,
		Technology:               problem.Technology,
		Name:                     problem.Name,
		Symptom:                  problem.Symptom,
		ProtectedIntent:          problem.ProtectedIntent,
		ExpectedRepair:           problem.ExpectedRepair,
		UnsafeShortcut:           problem.UnsafeShortcut,
		SemanticOracleGoal:       problem.Oracle,
		Fidelity:                 problem.Fidelity,
		CandidatePlans:           len(programs),
		PromotedPlans:            len(promoted),
		ExpectedPlanRecovered:    expectedRecovered,
		UniquePromotion:          len(promoted) == 1 && expectedRecovered,
		UnsafeCandidates:         unsafeCandidates,
		UnsafeCandidatesRejected: unsafeRejected,
		HeldOutCases:             len(heldOut),
		HeldOutFailures:          heldOutFailures,
		BaselineSolutions:        len(baselineSets),
		NousBaselineAgreement:    equalStrings(promotedSets, baselineSets),
		ExpectedRepairs:          strings.Split(expectedKey, ","),
	}
	return result, baselineRecovered, nil
}

func buildStore(domainsDir string, problem scenario) (*unit.Store, []repairSpec, []exampleSpec, []exampleSpec, error) {
	previous := seed.DomainsDir
	seed.DomainsDir = domainsDir
	defer func() { seed.DomainsDir = previous }()
	store := unit.NewStore()
	if err := seed.LoadDomain(store, "configrepair"); err != nil {
		return nil, nil, nil, nil, err
	}
	for _, item := range []struct{ category, slot string }{
		{"ConfigurationSchema", "data"},
		{"ConfigurationRepairExample", "configuration"},
		{"PrimitiveConfigurationRepair", "repairKey"},
	} {
		for _, name := range categoryUnits(store, item.category, item.slot) {
			store.Delete(name)
		}
	}

	schemaName := problem.ID + "-Schema"
	schema := unit.New(schemaName)
	schema.SetWorth(700)
	schema.Set("isA", []string{"ConfigurationSchema", "Anything"})
	schema.Set("data", schemaData(problem))
	store.Put(schema)

	training, heldOut := examples(problem)
	for _, example := range training {
		u := unit.New(example.Name)
		u.SetWorth(650)
		u.Set("isA", []string{"ConfigurationRepairExample", "Anything"})
		u.Set("schema", schemaName)
		u.Set("configuration", example.Configuration)
		store.Put(u)
	}

	var repairs []repairSpec
	repairIndex := 0
	for _, spec := range problem.Assignments {
		if !spec.Required {
			continue
		}
		repairIndex++
		repairs = append(repairs, repairSpec{
			Name:   problem.ID + "-Repair-" + strconv.Itoa(repairIndex),
			Repair: configvocab.Repair{Key: spec.Key, Value: spec.Good},
		})
	}
	for index, shortcut := range []configvocab.Repair{
		{Key: "environment", Value: "development"},
		{Key: "owner", Value: "unowned"},
	} {
		repairs = append(repairs, repairSpec{Name: problem.ID + "-Unsafe-" + strconv.Itoa(index+1), Repair: shortcut, Unsafe: true})
	}
	for _, spec := range repairs {
		u := unit.New(spec.Name)
		u.SetWorth(600)
		u.Set("isA", []string{"PrimitiveConfigurationRepair", "UnaryOp", "Op", "Anything"})
		u.Set("domain", []string{"Configuration"})
		u.Set("range", []string{"Configuration"})
		u.Set("arity", 1)
		u.Set("repairKey", spec.Repair.Key)
		u.Set("repairValue", spec.Repair.Value)
		u.Set("defn", strconv.Quote(spec.Repair.Key)+" "+strconv.Quote(spec.Repair.Value)+" config-set")
		store.Put(u)
	}
	return store, repairs, training, heldOut, nil
}

func schemaData(problem scenario) []string {
	data := []string{
		"field:environment:string", "field:owner:string", "field:region:string",
		"required:environment", "required:owner", "required:region",
		"protected:environment", "protected:owner", "protected:region",
	}
	for _, spec := range problem.Assignments {
		switch spec.Kind {
		case "bool":
			data = append(data, "field:"+spec.Key+":bool")
		case "int":
			data = append(data, "field:"+spec.Key+":int:0:65535")
		default:
			data = append(data, "field:"+spec.Key+":string")
		}
		data = append(data, "required:"+spec.Key)
		if spec.Protected {
			data = append(data, "protected:"+spec.Key)
		}
		if spec.Required {
			if spec.Minimum > 0 {
				data = append(data, fmt.Sprintf("min-if:environment=production,%s=%d", spec.Key, spec.Minimum))
			} else {
				data = append(data, "eq-if:environment=production,"+spec.Key+"="+spec.Good)
			}
		}
	}
	return data
}

func examples(problem scenario) ([]exampleSpec, []exampleSpec) {
	owners := []string{"payments", "platform", "orders", "identity", "billing", "search"}
	regions := []string{"us_east_1", "us_west_2", "eu_west_1", "ap_southeast_2", "ca_central_1", "eu_central_1"}
	makeExample := func(index int, heldOut bool) exampleSpec {
		configuration := []string{"environment=production", "owner=" + owners[index], "region=" + regions[index]}
		for _, spec := range problem.Assignments {
			configuration = append(configuration, spec.Key+"="+spec.Bad)
		}
		kind := "Train"
		if heldOut {
			kind = "HeldOut"
		}
		return exampleSpec{Name: fmt.Sprintf("%s-%s-%d", problem.ID, kind, index+1), Configuration: configuration}
	}
	training := make([]exampleSpec, 0, 4)
	for index := 0; index < 4; index++ {
		training = append(training, makeExample(index, false))
	}
	return training, []exampleSpec{makeExample(4, true), makeExample(5, true)}
}

func baselineSolutions(examples []exampleSpec, schema []string, repairs []repairSpec) []string {
	var solutions []string
	for _, indexes := range subsets(len(repairs), configvocab.MaxPlanSize) {
		plan := make([]configvocab.Repair, 0, len(indexes))
		for _, index := range indexes {
			plan = append(plan, repairs[index].Repair)
		}
		valid := true
		for _, example := range examples {
			actual, err := configvocab.ApplyPlan(example.Configuration, plan)
			if err != nil || !successful(actual, example.Configuration, schema) {
				valid = false
				break
			}
		}
		if valid {
			solutions = append(solutions, semanticRepairs(plan))
		}
	}
	sort.Strings(solutions)
	return solutions
}

func successful(actual, before, schema []string) bool {
	satisfied, err := configvocab.Satisfies(actual, schema)
	if err != nil || !satisfied {
		return false
	}
	preserved, err := configvocab.PreservesProtected(before, actual, schema)
	if err != nil || !preserved {
		return false
	}
	return true
}

func expectedPlan(repairs []repairSpec) []configvocab.Repair {
	var expected []configvocab.Repair
	for _, spec := range repairs {
		if !spec.Unsafe {
			expected = append(expected, spec.Repair)
		}
	}
	return expected
}

func semanticPlan(store *unit.Store, programName string) string {
	var repairs []configvocab.Repair
	for _, component := range store.Get(programName).GetStrings("components") {
		u := store.Get(component)
		repairs = append(repairs, configvocab.Repair{Key: u.GetString("repairKey"), Value: u.GetString("repairValue")})
	}
	return semanticRepairs(repairs)
}

func semanticRepairs(repairs []configvocab.Repair) string {
	items := make([]string, 0, len(repairs))
	for _, repair := range repairs {
		items = append(items, repair.Key+"="+repair.Value)
	}
	sort.Strings(items)
	return strings.Join(items, ",")
}

func categoryUnits(store *unit.Store, category, slot string) []string {
	var names []string
	for _, name := range store.Examples(category) {
		if name != category && store.Get(name).Has(slot) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func subsets(count, maximum int) [][]int {
	var result [][]int
	var visit func(int, []int)
	visit = func(next int, selected []int) {
		if len(selected) > 0 {
			result = append(result, append([]int(nil), selected...))
		}
		if len(selected) == maximum {
			return
		}
		for index := next; index < count; index++ {
			visit(index+1, append(selected, index))
		}
	}
	visit(0, nil)
	return result
}

func dslList(values []string) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.Quote(value)
	}
	return strings.Join(parts, " ") + " " + strconv.Itoa(len(values)) + " list-of"
}

func dslStrings(value dsl.Value) ([]string, bool) {
	if value.Kind() != dsl.VList {
		return nil, false
	}
	items := value.AsList()
	result := make([]string, len(items))
	for index, item := range items {
		if item.Kind() != dsl.VString {
			return nil, false
		}
		result[index] = item.AsString()
	}
	return result, true
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func equalStrings(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func limitations() []string {
	return []string{
		"Inputs are hand-translated flat facts, not raw Kubernetes YAML or Terraform HCL.",
		"Candidate catalogs already contain the exact repair values; Nous selects subsets but does not invent patches.",
		"Nested maps, lists, sets, quantities, CIDRs, expressions, absence, graph relations, and ordered edits are collapsed to scalar facts.",
		"Protected intent is scalar equality, not reachability, availability, privilege, state-address, or downtime predicates.",
		"The exhaustive baseline searches the identical bounded catalog, so agreement demonstrates correctness rather than an advantage over enumeration.",
	}
}
